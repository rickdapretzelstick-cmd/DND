package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/rickdapretzelstick-cmd/DND/engine"
)

type Server struct {
	eng     *engine.Engine
	mux     *http.ServeMux
	mu      sync.Mutex
	clients map[chan []byte]struct{}
	tokens  map[string]int
	notify  chan struct{}
}

func NewServer(eng *engine.Engine, static fs.FS) *Server {
	s := &Server{eng: eng, mux: http.NewServeMux(), clients: map[chan []byte]struct{}{}, tokens: map[string]int{}, notify: make(chan struct{}, 1)}
	eng.OnChange = func() {
		select {
		case s.notify <- struct{}{}:
		default:
		}
	}
	go s.broadcaster()

	s.mux.Handle("/", http.FileServer(http.FS(static)))
	s.mux.HandleFunc("/api/events", s.events)
	s.mux.HandleFunc("/api/data", s.data)
	s.mux.HandleFunc("/api/whoami", s.whoami)
	s.mux.HandleFunc("/api/join", s.join)

	player := func(path string, fn func(token string, body json.RawMessage) error) {
		s.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "POST only", http.StatusMethodNotAllowed)
				return
			}
			var body json.RawMessage
			_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body)
			reply(w, fn(r.Header.Get("X-Token"), body))
		})
	}
	s.mux.HandleFunc("/api/rollstats", func(w http.ResponseWriter, r *http.Request) {
		stats, err := eng.RollStats(r.Header.Get("X-Token"))
		if err != nil {
			reply(w, err)
			return
		}
		writeJSON(w, map[string]any{"stats": stats})
	})
	player("/api/character", func(t string, b json.RawMessage) error {
		var req engine.CreateReq
		if err := json.Unmarshal(b, &req); err != nil {
			return err
		}
		return eng.CreateCharacter(t, req)
	})
	player("/api/retire", func(t string, b json.RawMessage) error {
		var req struct{ CharID string }
		_ = json.Unmarshal(b, &req)
		return eng.RetireCharacter(t, req.CharID, false)
	})
	player("/api/act", func(t string, b json.RawMessage) error {
		var req struct{ Text string }
		_ = json.Unmarshal(b, &req)
		return eng.Act(t, req.Text)
	})
	player("/api/ooc", func(t string, b json.RawMessage) error {
		var req struct{ Text string }
		_ = json.Unmarshal(b, &req)
		return eng.OOC(t, req.Text)
	})
	player("/api/roll", func(t string, b json.RawMessage) error {
		var req struct{ Dice, Reason string }
		_ = json.Unmarshal(b, &req)
		return eng.RollDice(t, req.Dice, req.Reason)
	})
	player("/api/equip", func(t string, b json.RawMessage) error {
		var req struct{ Index int }
		_ = json.Unmarshal(b, &req)
		return eng.Equip(t, req.Index)
	})
	player("/api/use", func(t string, b json.RawMessage) error {
		var req struct {
			Index  int
			Target string
		}
		_ = json.Unmarshal(b, &req)
		return eng.UseItem(t, req.Index, req.Target)
	})
	player("/api/give", func(t string, b json.RawMessage) error {
		var req struct {
			Index int
			To    string
		}
		_ = json.Unmarshal(b, &req)
		return eng.GiveItem(t, req.Index, req.To)
	})
	player("/api/ability", func(t string, b json.RawMessage) error {
		var req struct{ Ability, Target string }
		_ = json.Unmarshal(b, &req)
		return eng.UseAbilityOutOfCombat(t, req.Ability, req.Target)
	})
	player("/api/combat", func(t string, b json.RawMessage) error {
		var req engine.CombatReq
		if err := json.Unmarshal(b, &req); err != nil {
			return err
		}
		return eng.CombatAction(t, req)
	})

	// Host-only controls: only the computer running the program may use them.
	hostOnly := func(path string, fn func(body json.RawMessage) error) {
		s.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if !isLocal(r) {
				http.Error(w, "host only", http.StatusForbidden)
				return
			}
			if r.Method != http.MethodPost {
				http.Error(w, "POST only", http.StatusMethodNotAllowed)
				return
			}
			var body json.RawMessage
			_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body)
			reply(w, fn(body))
		})
	}
	hostOnly("/api/host/new", func(b json.RawMessage) error {
		var req struct{ Name, Setting, Mode string }
		_ = json.Unmarshal(b, &req)
		eng.NewCampaign(req.Name, req.Setting, req.Mode)
		return nil
	})
	hostOnly("/api/host/load", func(b json.RawMessage) error {
		var req struct{ ID string }
		_ = json.Unmarshal(b, &req)
		return eng.LoadCampaign(req.ID)
	})
	hostOnly("/api/host/delete", func(b json.RawMessage) error {
		var req struct{ ID string }
		_ = json.Unmarshal(b, &req)
		return eng.DeleteCampaign(req.ID)
	})
	hostOnly("/api/host/settings", func(b json.RawMessage) error {
		var req engine.SettingsUpdate
		if err := json.Unmarshal(b, &req); err != nil {
			return err
		}
		eng.UpdateSettings(req)
		return nil
	})
	hostOnly("/api/host/start", func(json.RawMessage) error { return eng.StartAdventure() })
	hostOnly("/api/host/skip", func(json.RawMessage) error { eng.SkipTurn(); return nil })
	hostOnly("/api/host/rest", func(b json.RawMessage) error {
		var req struct{ Kind string }
		_ = json.Unmarshal(b, &req)
		eng.HostRest(req.Kind)
		return nil
	})
	hostOnly("/api/host/retire", func(b json.RawMessage) error {
		var req struct{ CharID string }
		_ = json.Unmarshal(b, &req)
		return eng.RetireCharacter("", req.CharID, true)
	})
	s.mux.HandleFunc("/api/host/campaigns", func(w http.ResponseWriter, r *http.Request) {
		if !isLocal(r) {
			http.Error(w, "host only", http.StatusForbidden)
			return
		}
		writeJSON(w, eng.ListCampaigns())
	})
	s.mux.HandleFunc("/api/host/info", func(w http.ResponseWriter, r *http.Request) {
		if !isLocal(r) {
			http.Error(w, "host only", http.StatusForbidden)
			return
		}
		port := 7777
		if a, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr); ok {
			if ta, ok := a.(*net.TCPAddr); ok {
				port = ta.Port
			}
		}
		var urls []string
		for _, ip := range lanIPs() {
			urls = append(urls, fmt.Sprintf("http://%s:%d", ip, port))
		}
		var info struct {
			URLs    []string `json:"urls"`
			HasKey  bool     `json:"hasKey"`
			KeyHint string   `json:"keyHint"`
			Model   string   `json:"model"`
			Effort  string   `json:"effort"`
			Version string   `json:"version"`
		}
		info.URLs, info.Version = urls, version
		eng.Do(func() {
			k := eng.APIKey()
			info.HasKey = k != ""
			if len(k) > 8 {
				info.KeyHint = "…" + k[len(k)-4:]
			}
			info.Model, info.Effort = eng.Cfg.Model, eng.Cfg.Effort
		})
		writeJSON(w, info)
	})
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	s.mux.ServeHTTP(w, r)
}

func isLocal(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func reply(w http.ResponseWriter, err error) {
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) broadcaster() {
	for range s.notify {
		time.Sleep(40 * time.Millisecond) // coalesce bursts of changes
		snap := s.eng.SnapshotJSON()
		s.mu.Lock()
		for ch := range s.clients {
			select {
			case ch <- snap:
			default:
				// Client is behind: replace its pending snapshot with the newest.
				select {
				case <-ch:
				default:
				}
				select {
				case ch <- snap:
				default:
				}
			}
		}
		s.mu.Unlock()
	}
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	token := r.URL.Query().Get("token")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	ch := make(chan []byte, 1)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()
	s.eng.SetOnline(token, true)
	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
		// Only mark offline if no other tab for this token is still connected.
		go func() {
			time.Sleep(3 * time.Second)
			if !s.tokenConnected(token) {
				s.eng.SetOnline(token, false)
			}
		}()
	}()
	s.mu.Lock()
	s.tokens[token]++
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.tokens[token]--
		s.mu.Unlock()
	}()

	send := func(b []byte) bool {
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return false
		}
		fl.Flush()
		return true
	}
	if !send(s.eng.SnapshotJSON()) {
		return
	}
	tick := time.NewTicker(20 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case b := <-ch:
			if !send(b) {
				return
			}
		case <-tick.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			fl.Flush()
		}
	}
}

func (s *Server) tokenConnected(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokens[token] > 0
}

func (s *Server) data(w http.ResponseWriter, r *http.Request) {
	type classView struct {
		engine.Class
		Abilities []engine.Ability `json:"abilities"`
	}
	classes := []classView{}
	for _, c := range engine.Classes {
		cv := classView{Class: c}
		for _, f := range c.Features {
			cv.Abilities = append(cv.Abilities, engine.AbilityList[f])
		}
		classes = append(classes, cv)
	}
	sort.Slice(classes, func(i, j int) bool { return classes[i].Name < classes[j].Name })
	races := []engine.Race{}
	for _, rc := range engine.Races {
		races = append(races, rc)
	}
	sort.Slice(races, func(i, j int) bool { return races[i].Name < races[j].Name })
	writeJSON(w, map[string]any{
		"races": races, "classes": classes, "abilities": engine.AbilityList, "skills": engine.Skills,
		"standardArray": engine.StandardArray, "abilityOrder": engine.Abilities, "xp": engine.XPThresholds,
	})
}

func (s *Server) whoami(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{"isHost": isLocal(r)}
	if p := s.eng.PlayerByToken(r.URL.Query().Get("token")); p != nil {
		out["playerId"] = p.ID
		out["name"] = p.Name
	}
	writeJSON(w, out)
}

func (s *Server) join(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Token string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		reply(w, err)
		return
	}
	p, err := s.eng.Join(req.Name, req.Token)
	if err != nil {
		reply(w, err)
		return
	}
	writeJSON(w, map[string]any{"token": p.Token, "playerId": p.ID, "name": p.Name, "isHost": isLocal(r)})
}
