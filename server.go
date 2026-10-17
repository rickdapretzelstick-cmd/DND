package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rickdapretzelstick-cmd/DND/engine"
)

type Server struct {
	eng       *engine.Engine
	mux       *http.ServeMux
	mu        sync.Mutex
	clients   map[chan []byte]struct{}
	tokens    map[string]int
	notify    chan struct{}
	posNotify chan struct{}
}

func NewServer(eng *engine.Engine, static fs.FS) *Server {
	s := &Server{eng: eng, mux: http.NewServeMux(), clients: map[chan []byte]struct{}{}, tokens: map[string]int{}, notify: make(chan struct{}, 1), posNotify: make(chan struct{}, 1)}
	eng.OnChange = func() {
		select {
		case s.notify <- struct{}{}:
		default:
		}
	}
	eng.OnPos = func() {
		select {
		case s.posNotify <- struct{}{}:
		default:
		}
	}
	go s.broadcaster()
	go s.posBroadcaster()

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
	s.mux.HandleFunc("/api/world", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(eng.WorldJSON())
	})
	s.mux.HandleFunc("/api/move", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			X, Y   float64
			Dir    int
			Moving bool
		}
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&req)
		x, y, err := eng.Move(r.Header.Get("X-Token"), req.X, req.Y, req.Dir, req.Moving)
		out := map[string]any{"x": x, "y": y}
		if err != nil {
			out["error"] = err.Error()
		}
		writeJSON(w, out)
	})
	dialogReply := func(w http.ResponseWriter, d *engine.Dialog, err error) {
		if err != nil {
			reply(w, err)
			return
		}
		writeJSON(w, map[string]any{"dialog": d})
	}
	s.mux.HandleFunc("/api/interact", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ NPC string }
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req)
		d, err := eng.Interact(r.Header.Get("X-Token"), req.NPC)
		dialogReply(w, d, err)
	})
	s.mux.HandleFunc("/api/dialog", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Option string }
		_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req)
		d, err := eng.ChooseDialog(r.Header.Get("X-Token"), req.Option)
		dialogReply(w, d, err)
	})
	player("/api/talk", func(t string, b json.RawMessage) error {
		var req struct{ NPC, Text string }
		_ = json.Unmarshal(b, &req)
		return eng.Talk(t, req.NPC, req.Text)
	})
	player("/api/choose", func(t string, b json.RawMessage) error {
		var req struct{ Option string }
		_ = json.Unmarshal(b, &req)
		return eng.Choose(t, req.Option)
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

func sseFrame(event string, data []byte) []byte {
	return []byte("event: " + event + "\ndata: " + string(data) + "\n\n")
}

func (s *Server) posBroadcaster() {
	for range s.posNotify {
		s.fanout(sseFrame("pos", s.eng.PosJSON()))
		time.Sleep(90 * time.Millisecond)
	}
}

func (s *Server) broadcaster() {
	for range s.notify {
		time.Sleep(40 * time.Millisecond) // coalesce bursts of changes
		s.fanout(sseFrame("state", s.eng.SnapshotJSON()))
	}
}

func (s *Server) fanout(frame []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- frame:
		default:
			// A stuck client: drop its oldest frame to make room.
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- frame:
			default:
			}
		}
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
	ch := make(chan []byte, 32)
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
		if _, err := w.Write(b); err != nil {
			return false
		}
		fl.Flush()
		return true
	}
	if !send(sseFrame("state", s.eng.SnapshotJSON())) || !send(sseFrame("pos", s.eng.PosJSON())) {
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
	type techView struct {
		engine.Technique
		Abilities []engine.Ability `json:"abilities"`
	}
	var techs []techView
	for _, id := range engine.SortedTechniqueIDs() {
		t := engine.Techniques[id]
		tv := techView{Technique: t}
		for _, m := range t.Moves {
			tv.Abilities = append(tv.Abilities, engine.AbilityList[m])
		}
		tv.Abilities = append(tv.Abilities, engine.AbilityList[t.Domain])
		techs = append(techs, tv)
	}
	origins := []engine.Origin{}
	for _, id := range []string{"non_clan", "gojo", "zenin", "kamo", "inumaki", "half_curse"} {
		origins = append(origins, engine.Origins[id])
	}
	writeJSON(w, map[string]any{
		"origins": origins, "techniques": techs, "abilities": engine.AbilityList, "skills": engine.Skills,
		"standardArray": engine.StandardArray, "abilityOrder": engine.Abilities, "abilityNames": engine.AbilityNames,
		"xp": engine.XPThresholds, "curses": engine.Bestiary, "items": engine.ItemList, "grades": engine.GradeNames,
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
