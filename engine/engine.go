package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	mrand "math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Config holds host settings saved next to the executable.
type Config struct {
	Provider    string `json:"provider"` // "anthropic" or "ollama"
	OllamaURL   string `json:"ollamaUrl"`
	OllamaModel string `json:"ollamaModel"`
	OllamaCtx   int    `json:"ollamaCtx"`
	APIKey      string `json:"apiKey"`
	Model       string `json:"model"`
	Effort      string `json:"effort"`
	Port        int    `json:"port"`
	LastID      string `json:"lastCampaign"`
}

// PendingAction is something a player asked the DM to resolve.
type PendingAction struct {
	PlayerID string
	CharID   string
	CharName string
	Text     string
}

// DMEvent is a unit of work for the Dungeon Master.
type DMEvent struct {
	Kind    string // start, actions, talk, combat_end, defeat, join, arrive, choice
	Actions []PendingAction
	Summary string
	Quest   string
}

type Engine struct {
	mu       sync.Mutex
	C        *Campaign
	W        *World
	Roamers  []*Roamer
	Dir      string
	cfgPath  string
	Cfg      Config
	OnChange func()
	OnPos    func()
	DMBusy   bool

	queue       []DMEvent
	wake        chan struct{}
	offline     *OfflineDM
	ollama      *OllamaDM
	testBackend DMBackend
	ai          DMBackend
	dirty       bool
	posDirty    bool
	warned      map[string]bool
	dialogs     map[string]*dialogCtx
}

const DefaultModel = "claude-opus-5-5"

func New(dir string) *Engine {
	e := newBare(dir)
	if b, err := os.ReadFile(e.cfgPath); err == nil {
		_ = json.Unmarshal(b, &e.Cfg)
	}
	if e.Cfg.Model == "" {
		e.Cfg.Model = DefaultModel
	}
	if e.Cfg.Effort == "" {
		e.Cfg.Effort = "low"
	}
	e.Cfg.fillOllamaDefaults()
	if e.Cfg.LastID != "" {
		if c, err := e.readCampaign(e.Cfg.LastID); err == nil {
			e.setCampaign(c)
		}
	}
	if e.C == nil {
		mode := "offline"
		if e.APIKey() != "" {
			mode = "ai"
		}
		e.newCampaign("Jujutsu Chronicles", "", mode)
	}
	for _, p := range e.C.Players {
		p.Online = false
	}
	go e.dmWorker()
	go e.autosaver()
	go e.worldTicker()
	return e
}

func newBare(dir string) *Engine {
	e := &Engine{Dir: filepath.Join(dir, "saves"), cfgPath: filepath.Join(dir, "dnd-runner-config.json"),
		wake: make(chan struct{}, 1), offline: &OfflineDM{}, ai: &ClaudeDM{}, ollama: &OllamaDM{}, warned: map[string]bool{}, dialogs: map[string]*dialogCtx{}}
	_ = os.MkdirAll(e.Dir, 0o755)
	return e
}

func (e *Engine) APIKey() string {
	if e.Cfg.APIKey != "" {
		return e.Cfg.APIKey
	}
	return os.Getenv("ANTHROPIC_API_KEY")
}

// DMBackend is an AI Dungeon Master implementation.
type DMBackend interface {
	Handle(ctx context.Context, e *Engine, ev DMEvent) error
}

func (e *Engine) AIActive() bool {
	if e.C.Mode != "ai" {
		return false
	}
	if e.Cfg.Provider == "ollama" {
		return true
	}
	return e.APIKey() != ""
}

// backend picks the AI DM for the configured provider.
func (e *Engine) backend() DMBackend {
	if e.testBackend != nil {
		return e.testBackend
	}
	if e.Cfg.Provider == "ollama" {
		return e.ollama
	}
	return e.ai
}

func (e *Engine) saveConfig() {
	b, _ := json.MarshalIndent(e.Cfg, "", "  ")
	_ = os.WriteFile(e.cfgPath, b, 0o600)
}

func randToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// setCampaign installs a campaign and (re)generates its world.
func (e *Engine) setCampaign(c *Campaign) {
	if c.Players == nil {
		c.Players = map[string]*Player{}
	}
	if c.Characters == nil {
		c.Characters = map[string]*Character{}
	}
	if c.Flags == nil {
		c.Flags = map[string]string{}
	}
	if c.Rep == nil {
		c.Rep = map[string]int{}
	}
	if c.Cleared == nil {
		c.Cleared = map[string]bool{}
	}
	if c.Discovered == nil {
		c.Discovered = map[string]bool{"high": true}
	}
	if c.NPCs == nil {
		c.NPCs = map[string]*NPCState{}
	}
	if c.Seed == 0 {
		c.Seed = mrand.Int64N(1 << 40)
	}
	if c.Clock == 0 && c.Day == 0 {
		c.Clock, c.Day = 9, 1
	}
	e.C = c
	e.registerCustomCurses()
	e.W = GenerateWorld(c.Seed)
	e.Roamers = nil
	e.queue = nil
	e.dialogs = map[string]*dialogCtx{}
	for _, ch := range c.Characters {
		if ch.Status == nil {
			ch.Status = map[string]int{}
		}
		if ch.Used == nil {
			ch.Used = map[string]int{}
		}
		if c.Combat == nil {
			ch.InBattle = false
		}
		if !e.W.Walkable(ch.X, ch.Y) {
			ch.X, ch.Y = e.hubSpawn()
		}
	}
}

func (e *Engine) newCampaign(name, setting, mode string) *Campaign {
	slug := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(slug) > 30 {
		slug = slug[:30]
	}
	if slug == "" {
		slug = "campaign"
	}
	c := &Campaign{ID: fmt.Sprintf("%s-%s", slug, time.Now().Format("20060102-150405")), Name: name,
		Setting: setting, Mode: mode, Created: time.Now(), Location: "Tokyo Jujutsu High"}
	e.setCampaign(c)
	e.Cfg.LastID = c.ID
	e.saveConfig()
	e.logf("system", "", "📜 \"%s\" begins. Make a sorcerer, then the host presses Begin.", name)
	e.dirty = true
	return c
}

func (e *Engine) changed() {
	e.dirty = true
	if e.OnChange != nil {
		e.OnChange()
	}
}

func (e *Engine) logf(kind, who, format string, args ...any) {
	e.Log(kind, who, fmt.Sprintf(format, args...))
}

// Log appends a line to the story log. Caller holds the lock.
func (e *Engine) Log(kind, who, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	c := e.C
	c.NextLogID++
	c.Log = append(c.Log, LogEntry{ID: c.NextLogID, Kind: kind, Who: who, Text: text, Time: time.Now()})
	if len(c.Log) > 3000 {
		c.Log = c.Log[len(c.Log)-2500:]
	}
	e.changed()
}

func (e *Engine) nextID(prefix string) string {
	e.C.NextID++
	return fmt.Sprintf("%s%d", prefix, e.C.NextID)
}

// Do runs f with the engine lock held.
func (e *Engine) Do(f func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	f()
}

// ---------- persistence ----------

func (e *Engine) readCampaign(id string) (*Campaign, error) {
	if strings.ContainsAny(id, `/\.`) {
		return nil, fmt.Errorf("bad campaign id")
	}
	b, err := os.ReadFile(filepath.Join(e.Dir, id+".json"))
	if err != nil {
		return nil, err
	}
	var c Campaign
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (e *Engine) saveLocked() error {
	b, err := json.MarshalIndent(e.C, "", " ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(e.Dir, e.C.ID+".json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	e.dirty = false
	return os.Rename(tmp, filepath.Join(e.Dir, e.C.ID+".json"))
}

func (e *Engine) Save() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.saveLocked()
}

func (e *Engine) autosaver() {
	for range time.Tick(3 * time.Second) {
		e.mu.Lock()
		if e.dirty {
			if err := e.saveLocked(); err != nil {
				log.Println("autosave failed:", err)
			}
		}
		e.mu.Unlock()
	}
}

type CampaignInfo struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Mode     string    `json:"mode"`
	Modified time.Time `json:"modified"`
	Heroes   []string  `json:"heroes"`
	Current  bool      `json:"current"`
}

func (e *Engine) ListCampaigns() []CampaignInfo {
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.saveLocked()
	files, _ := filepath.Glob(filepath.Join(e.Dir, "*.json"))
	var out []CampaignInfo
	for _, f := range files {
		id := strings.TrimSuffix(filepath.Base(f), ".json")
		c, err := e.readCampaign(id)
		if err != nil {
			continue
		}
		st, _ := os.Stat(f)
		info := CampaignInfo{ID: c.ID, Name: c.Name, Mode: c.Mode, Modified: st.ModTime(), Current: c.ID == e.C.ID}
		for _, ch := range c.Characters {
			info.Heroes = append(info.Heroes, fmt.Sprintf("%s (L%d %s)", ch.Name, ch.Level, Techniques[ch.Class].Name))
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out
}

func (e *Engine) carryPlayers(old *Campaign) {
	for _, p := range old.Players {
		if existing, ok := e.C.Players[p.ID]; ok {
			existing.Online = p.Online
			continue
		}
		np := *p
		np.CharID = ""
		e.C.Players[p.ID] = &np
	}
}

func (e *Engine) NewCampaign(name, setting, mode string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.saveLocked()
	old := e.C
	if strings.TrimSpace(name) == "" {
		name = "Jujutsu Chronicles"
	}
	if mode != "ai" {
		mode = "offline"
	}
	e.newCampaign(strings.TrimSpace(name), strings.TrimSpace(setting), mode)
	e.carryPlayers(old)
	e.changed()
}

func (e *Engine) LoadCampaign(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	c, err := e.readCampaign(id)
	if err != nil {
		return err
	}
	_ = e.saveLocked()
	old := e.C
	e.setCampaign(c)
	for _, p := range e.C.Players {
		p.Online = false
	}
	e.carryPlayers(old)
	e.Cfg.LastID = c.ID
	e.saveConfig()
	e.logf("system", "", "📂 Campaign \"%s\" loaded.", c.Name)
	return nil
}

func (e *Engine) DeleteCampaign(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if id == e.C.ID {
		return fmt.Errorf("can't delete the campaign that's currently loaded")
	}
	if strings.ContainsAny(id, `/\.`) {
		return fmt.Errorf("bad id")
	}
	return os.Remove(filepath.Join(e.Dir, id+".json"))
}

type SettingsUpdate struct {
	Provider    string  `json:"provider"`
	OllamaURL   string  `json:"ollamaUrl"`
	OllamaModel string  `json:"ollamaModel"`
	OllamaCtx   int     `json:"ollamaCtx"`
	APIKey      *string `json:"apiKey"`
	Model       string  `json:"model"`
	Effort      string  `json:"effort"`
	Mode        string  `json:"mode"`
	Setting     *string `json:"setting"`
	Name        string  `json:"name"`
}

func (e *Engine) UpdateSettings(u SettingsUpdate) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if u.APIKey != nil {
		e.Cfg.APIKey = strings.TrimSpace(*u.APIKey)
	}
	if u.Provider == "anthropic" || u.Provider == "ollama" {
		e.Cfg.Provider = u.Provider
	}
	if v := strings.TrimSpace(u.OllamaURL); v != "" {
		e.Cfg.OllamaURL = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(u.OllamaModel); v != "" {
		e.Cfg.OllamaModel = v
	}
	if u.OllamaCtx >= 2048 && u.OllamaCtx <= 131072 {
		e.Cfg.OllamaCtx = u.OllamaCtx
	}
	if u.Model != "" {
		e.Cfg.Model = strings.TrimSpace(u.Model)
	}
	switch u.Effort {
	case "low", "medium", "high", "xhigh", "max":
		e.Cfg.Effort = u.Effort
	}
	if (u.Mode == "ai" || u.Mode == "offline") && u.Mode != e.C.Mode {
		e.C.Mode = u.Mode
		if u.Mode == "ai" {
			e.Log("system", "", "🧠 The AI Dungeon Master takes over the narration.")
		} else {
			e.Log("system", "", "🎲 The offline Dungeon Master takes over the narration.")
		}
	}
	if u.Setting != nil {
		e.C.Setting = strings.TrimSpace(*u.Setting)
	}
	if u.Name != "" {
		e.C.Name = strings.TrimSpace(u.Name)
	}
	e.saveConfig()
	e.changed()
}

// ---------- snapshot ----------

type publicPlayer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	CharID string `json:"charId"`
	Online bool   `json:"online"`
}

type Snapshot struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Setting      string                   `json:"setting"`
	Mode         string                   `json:"mode"`
	Started      bool                     `json:"started"`
	Seed         int64                    `json:"seed"`
	Location     string                   `json:"location"`
	Quest        string                   `json:"quest"`
	Act          int                      `json:"act"`
	Players      []publicPlayer           `json:"players"`
	Characters   []*Character             `json:"characters"`
	Log          []LogEntry               `json:"log"`
	Combat       *Combat                  `json:"combat"`
	LastBattle   *BattleResult            `json:"lastBattle"`
	CustomCurses map[string]CurseTemplate `json:"customCurses,omitempty"`
	Quests       []*Quest                 `json:"quests"`
	Choice       *Choice                  `json:"choice"`
	Rep          map[string]int           `json:"rep"`
	Discovered   map[string]bool          `json:"discovered"`
	Cleared      map[string]bool          `json:"cleared"`
	Extra        []*NPC                   `json:"extraNpcs"`
	Gone         []string                 `json:"goneNpcs"`
	Burned       []string                 `json:"burned"`
	Journal      []string                 `json:"journal"`
	Markers      map[string]string        `json:"markers"`
	Flags        map[string]string        `json:"flags"`
	Clock        float64                  `json:"clock"`
	Day          int                      `json:"day"`
	DMBusy       bool                     `json:"dmBusy"`
	AI           bool                     `json:"ai"`
	HasKey       bool                     `json:"hasKey"`
	Model        string                   `json:"model"`
	Effort       string                   `json:"effort"`
}

func (e *Engine) SnapshotJSON() []byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	c := e.C
	s := Snapshot{ID: c.ID, Name: c.Name, Setting: c.Setting, Mode: c.Mode, Started: c.Started, Seed: c.Seed, Location: c.Location,
		Quest: c.Quest, Act: c.Act, Combat: c.Combat, LastBattle: c.LastBattle, CustomCurses: c.CustomCurses, Rep: c.Rep, Discovered: c.Discovered, Cleared: c.Cleared, Extra: c.ExtraNPCs,
		Journal: c.Journal, DMBusy: e.DMBusy, AI: e.AIActive(), HasKey: e.APIKey() != "" || e.Cfg.Provider == "ollama", Model: e.modelLabel(), Effort: e.Cfg.Effort,
		Players: []publicPlayer{}, Characters: []*Character{}, Log: []LogEntry{}, Quests: []*Quest{}, Gone: []string{}, Burned: []string{}}
	for _, p := range c.Players {
		s.Players = append(s.Players, publicPlayer{p.ID, p.Name, p.CharID, p.Online})
	}
	sort.Slice(s.Players, func(i, j int) bool { return s.Players[i].ID < s.Players[j].ID })
	s.Characters = append(s.Characters, e.sortedChars()...)
	logs := c.Log
	if len(logs) > 200 {
		logs = logs[len(logs)-200:]
	}
	s.Log = append(s.Log, logs...)
	for _, q := range c.Quests {
		if q.Stage != "offered" {
			s.Quests = append(s.Quests, q)
		}
	}
	if c.Choice != nil {
		pc := *c.Choice
		pc.Options = nil
		for _, o := range c.Choice.Options {
			pc.Options = append(pc.Options, ChoiceOption{ID: o.ID, Text: o.Text, Need: o.Need})
		}
		s.Choice = &pc
	}
	for id, st := range c.NPCs {
		if st.Gone {
			s.Gone = append(s.Gone, id)
		}
	}
	for k, v := range c.Flags {
		if strings.HasPrefix(k, "burned_") && v == "1" {
			s.Burned = append(s.Burned, strings.TrimPrefix(k, "burned_"))
		}
	}
	s.Markers = e.npcMarkers()
	s.Flags = map[string]string{}
	for k, v := range c.Flags {
		if !strings.HasPrefix(k, "train_") {
			s.Flags[k] = v
		}
	}
	s.Clock, s.Day = c.Clock, c.Day
	b, _ := json.Marshal(s)
	return b
}

// npcMarkers shows "!" for story/quest offers and "?" for quests to turn in.
func (e *Engine) npcMarkers() map[string]string {
	m := map[string]string{}
	f := e.C.Flags
	if e.C.Started {
		switch {
		case e.C.Act == 0:
			m["gojo"] = "!"
		case f["act1"] == "done" && f["act2"] == "":
			m["nanami"] = "!"
		case f["act2"] == "find":
			m["junpei"] = "!"
		case f["act2"] == "done" && f["act3"] == "":
			m["gojo"] = "!"
		case f["act3"] == "done" && f["act4"] == "":
			m["yaga"] = "!"
		case f["act4"] == "done" && f["act5"] == "":
			m["gojo"] = "!"
		}
	}
	for _, n := range e.allNPCs() {
		if n.Role == "elder" && e.questFrom(n.ID, "side") == nil && m[n.ID] == "" {
			m[n.ID] = "+"
		}
		if e.hasWork(n) && m[n.ID] == "" {
			m[n.ID] = "+"
		}
	}
	for _, q := range e.C.Quests {
		if q.Stage == "return" && q.Giver != "" {
			m[q.Giver] = "?"
		}
		if q.Stage == "active" {
			if _, st := q.cur(); st != nil && (st.Type == "talk" || st.Type == "deliver") && m[st.Target] == "" {
				m[st.Target] = "!"
			}
		}
	}
	return m
}

// WorldJSON is the static map, fetched once per campaign.
func (e *Engine) WorldJSON() []byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	b, _ := json.Marshal(e.W)
	return b
}

// ---------- players & characters ----------

func (e *Engine) Join(name, token string) (*Player, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if token != "" {
		for _, p := range e.C.Players {
			if p.Token == token {
				if n := strings.TrimSpace(name); n != "" && len(n) <= 30 {
					p.Name = n
				}
				return p, nil
			}
		}
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 30 {
		return nil, fmt.Errorf("enter a name (1-30 characters)")
	}
	if token == "" {
		token = randToken()
	}
	p := &Player{ID: e.nextID("p"), Name: name, Token: token}
	e.C.Players[p.ID] = p
	e.logf("system", "", "👋 %s joined.", name)
	return p, nil
}

func (e *Engine) PlayerByToken(token string) *Player {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.playerByToken(token)
}

func (e *Engine) playerByToken(token string) *Player {
	if token == "" {
		return nil
	}
	for _, p := range e.C.Players {
		if p.Token == token {
			return p
		}
	}
	return nil
}

func (e *Engine) SetOnline(token string, online bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if p := e.playerByToken(token); p != nil && p.Online != online {
		p.Online = online
		p.LastSeen = time.Now()
		if ch := e.C.Characters[p.CharID]; ch != nil {
			ch.Moving = false
		}
		e.posDirty = true
		e.changed()
	}
}

func (e *Engine) RollStats(token string) ([]int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.playerByToken(token)
	if p == nil {
		return nil, fmt.Errorf("join first")
	}
	p.RolledStats = Roll4d6()
	return p.RolledStats, nil
}

type CreateReq struct {
	Name      string         `json:"name"`
	Race      string         `json:"race"`
	Class     string         `json:"class"`
	Method    string         `json:"method"`
	Stats     map[string]int `json:"stats"`
	Backstory string         `json:"backstory"`
	Look      Appearance     `json:"look"`
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func cleanLook(l Appearance) Appearance {
	fix := func(c, def string) string {
		if hexColor.MatchString(c) {
			return c
		}
		return def
	}
	l.HairColor = fix(l.HairColor, "#1b1b1b")
	l.Eyes = fix(l.Eyes, "#3b2a1a")
	l.Color1 = fix(l.Color1, "#14141c")
	l.Color2 = fix(l.Color2, "#2a2a3a")
	l.Skin = max(0, min(l.Skin, 5))
	l.Hair = max(0, min(l.Hair, 11))
	l.Outfit = max(0, min(l.Outfit, 7))
	l.Accessory = max(0, min(l.Accessory, 7))
	l.Build = max(0, min(l.Build, 2))
	return l
}

func (e *Engine) CreateCharacter(token string, r CreateReq) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.playerByToken(token)
	if p == nil {
		return fmt.Errorf("join first")
	}
	if ch, ok := e.C.Characters[p.CharID]; ok && !ch.Dead {
		return fmt.Errorf("you already have a living sorcerer (%s)", ch.Name)
	}
	vals := make([]int, 0, 6)
	for _, ab := range Abilities {
		vals = append(vals, r.Stats[ab])
	}
	switch r.Method {
	case "roll":
		if p.RolledStats == nil || !sameMultiset(vals, p.RolledStats) {
			return fmt.Errorf("stats must use the scores you rolled")
		}
	default:
		if !sameMultiset(vals, StandardArray) {
			return fmt.Errorf("stats must use the standard array 15, 14, 13, 12, 10, 8")
		}
	}
	if len(r.Backstory) > 1500 {
		r.Backstory = r.Backstory[:1500]
	}
	ch, err := NewCharacter(e.nextID("c"), p.ID, r.Name, r.Race, r.Class, r.Stats, r.Backstory, cleanLook(r.Look))
	if err != nil {
		return err
	}
	best := 0
	for _, o := range e.C.Characters {
		if !o.Dead && o.XP > best {
			best = o.XP
		}
	}
	if best > 0 {
		ch.GainXP(best * 3 / 4)
		ch.HP, ch.CE = ch.MaxHP, ch.CEMax
	}
	ch.X, ch.Y = e.hubSpawn()
	ch.X += float64(len(e.C.Characters)%3) - 1
	e.C.Characters[ch.ID] = ch
	p.CharID = ch.ID
	p.RolledStats = nil
	e.logf("system", "", "🛡️ %s created %s — %s, %s user.", p.Name, ch.Name, Origins[ch.Race].Name, Techniques[ch.Class].Name)
	e.posDirty = true
	if e.C.Started {
		e.enqueue(DMEvent{Kind: "join", Summary: ch.ID})
	}
	return nil
}

func (e *Engine) RetireCharacter(token, charID string, host bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	ch, ok := e.C.Characters[charID]
	if !ok {
		return fmt.Errorf("no such character")
	}
	p := e.playerByToken(token)
	if !host && (p == nil || ch.Owner != p.ID) {
		return fmt.Errorf("that's not your character")
	}
	if ch.InBattle {
		return fmt.Errorf("can't retire a sorcerer mid-battle")
	}
	delete(e.C.Characters, charID)
	for _, pl := range e.C.Players {
		if pl.CharID == charID {
			pl.CharID = ""
		}
	}
	e.logf("system", "", "🕯️ %s leaves the story.", ch.Name)
	e.posDirty = true
	return nil
}

func (e *Engine) charOf(token string) (*Player, *Character, error) {
	p := e.playerByToken(token)
	if p == nil {
		return nil, nil, fmt.Errorf("join first")
	}
	ch, ok := e.C.Characters[p.CharID]
	if !ok {
		return p, nil, fmt.Errorf("create a sorcerer first")
	}
	return p, ch, nil
}

// ---------- table actions ----------

func (e *Engine) OOC(token, text string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.playerByToken(token)
	if p == nil {
		return fmt.Errorf("join first")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len(text) > 1000 {
		text = text[:1000]
	}
	e.Log("ooc", p.Name, text)
	return nil
}

// Act sends an in-character action to the DM: anything at all.
func (e *Engine) Act(token, text string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, ch, err := e.charOf(token)
	if err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len(text) > 1000 {
		text = text[:1000]
	}
	if ch.Dead {
		return fmt.Errorf("%s is dead — retire them and make a new sorcerer", ch.Name)
	}
	e.Log("action", ch.Name, text)
	e.enqueue(DMEvent{Kind: "actions", Actions: []PendingAction{{PlayerID: p.ID, CharID: ch.ID, CharName: ch.Name, Text: text}}})
	return nil
}

// Talk is free-form speech to a specific NPC (from the dialog box).
func (e *Engine) Talk(token, npcID, text string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, ch, err := e.charOf(token)
	if err != nil {
		return err
	}
	n := e.npcByID(npcID)
	if n == nil {
		return fmt.Errorf("nobody's there")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len(text) > 800 {
		text = text[:800]
	}
	e.Log("action", ch.Name, "to "+n.Name+": "+text)
	e.questEvent("talk", ch, n.ID)
	e.enqueue(DMEvent{Kind: "talk", Summary: n.ID, Actions: []PendingAction{{PlayerID: p.ID, CharID: ch.ID, CharName: ch.Name, Text: text}}})
	return nil
}

func (e *Engine) RollDice(token, expr, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.playerByToken(token)
	if p == nil {
		return fmt.Errorf("join first")
	}
	r, err := Roll(expr)
	if err != nil {
		return err
	}
	who := p.Name
	if ch, ok := e.C.Characters[p.CharID]; ok {
		who = ch.Name
	}
	msg := fmt.Sprintf("🎲 %s rolls %s: %s", who, r.Expr, r)
	if reason = strings.TrimSpace(reason); reason != "" {
		msg += " — " + reason
	}
	e.Log("roll", who, msg)
	return nil
}

func (e *Engine) Equip(token string, idx int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ch, err := e.charOf(token)
	if err != nil {
		return err
	}
	if ch.InBattle {
		return fmt.Errorf("no time to change gear mid-battle")
	}
	if err := ch.Equip(idx); err != nil {
		return err
	}
	e.changed()
	return nil
}

func (e *Engine) GiveItem(token string, idx int, toChar string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ch, err := e.charOf(token)
	if err != nil {
		return err
	}
	to, ok := e.C.Characters[toChar]
	if !ok || to.ID == ch.ID {
		return fmt.Errorf("pick another sorcerer")
	}
	if idx < 0 || idx >= len(ch.Inventory) {
		return fmt.Errorf("no such item")
	}
	it := ch.Inventory[idx]
	ch.Inventory = append(ch.Inventory[:idx], ch.Inventory[idx+1:]...)
	it.Equipped = false
	to.AddItem(it)
	ch.RecalcAC()
	e.logf("system", "", "🤝 %s gives %s to %s.", ch.Name, it.Name, to.Name)
	return nil
}

func (e *Engine) UseItem(token string, idx int, targetID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ch, err := e.charOf(token)
	if err != nil {
		return err
	}
	if ch.InBattle {
		return e.combatAction(ch, CombatReq{Type: "item", Item: idx, Target: targetID})
	}
	if idx < 0 || idx >= len(ch.Inventory) {
		return fmt.Errorf("no such item")
	}
	it := ch.Inventory[idx]
	if it.Kind != "consumable" || (it.Heal == "" && it.CE == 0) {
		return fmt.Errorf("%s isn't something you use like that — type what you do with it instead", it.Name)
	}
	target := ch
	if t, ok := e.C.Characters[targetID]; ok && !t.Dead {
		target = t
	}
	ch.RemoveItem(it.ID, 1)
	msg := fmt.Sprintf("🧪 %s uses %s on %s:", ch.Name, it.Name, target.Name)
	if it.Heal != "" {
		msg += fmt.Sprintf(" +%d HP", target.heal(MustRoll(it.Heal).Total))
	}
	if it.CE > 0 {
		target.CE = min(target.CEMax, target.CE+it.CE)
		msg += fmt.Sprintf(" +%d CE", it.CE)
	}
	e.Log("system", ch.Name, msg+".")
	return nil
}

// ---------- host controls ----------

func (e *Engine) StartAdventure() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.C.Characters) == 0 {
		return fmt.Errorf("at least one player needs a sorcerer first")
	}
	if e.C.Started {
		return fmt.Errorf("the story has already begun")
	}
	e.C.Started = true
	e.Log("system", "", "🌸 The story begins!")
	e.enqueue(DMEvent{Kind: "start"})
	return nil
}

func (e *Engine) HostRest(kind string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.C.Combat != nil {
		return
	}
	e.restParty(kind)
}

func (e *Engine) restParty(kind string) {
	if kind == "long" {
		for _, ch := range e.C.Characters {
			ch.LongRest()
		}
		e.Log("system", "", "🌙 Everyone takes a long rest. HP and cursed energy fully restored.")
		return
	}
	var parts []string
	for _, ch := range e.livingChars() {
		parts = append(parts, fmt.Sprintf("%s +%d HP", ch.Name, ch.ShortRest()))
	}
	e.Log("system", "", "☕ Short rest: "+strings.Join(parts, ", ")+".")
}

func (e *Engine) sortedChars() []*Character {
	out := make([]*Character, 0, len(e.C.Characters))
	for _, ch := range e.C.Characters {
		out = append(out, ch)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (e *Engine) livingChars() []*Character {
	var out []*Character
	for _, ch := range e.sortedChars() {
		if !ch.Dead {
			out = append(out, ch)
		}
	}
	return out
}

func (e *Engine) PartyLevel() int {
	ch := e.livingChars()
	if len(ch) == 0 {
		return 1
	}
	t := 0
	for _, c := range ch {
		t += c.Level
	}
	return max(1, (t+len(ch)/2)/len(ch))
}

// AwardXP splits XP across living characters.
func (e *Engine) AwardXP(total int, reason string) { e.awardXPTo(e.livingChars(), total, reason) }

// ---------- DM queue ----------

func (e *Engine) enqueue(ev DMEvent) {
	e.queue = append(e.queue, ev)
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

func (e *Engine) dmWorker() {
	for range e.wake {
		for {
			time.Sleep(700 * time.Millisecond)
			e.mu.Lock()
			if len(e.queue) == 0 {
				e.mu.Unlock()
				break
			}
			ev := e.queue[0]
			rest := e.queue[1:]
			if ev.Kind == "actions" {
				var keep []DMEvent
				for _, o := range rest {
					if o.Kind == "actions" {
						ev.Actions = append(ev.Actions, o.Actions...)
					} else {
						keep = append(keep, o)
					}
				}
				rest = keep
			}
			e.queue = rest
			ai := e.AIActive()
			if !ai {
				e.offline.Handle(e, ev)
				e.mu.Unlock()
				continue
			}
			e.DMBusy = true
			e.changed()
			e.mu.Unlock()

			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			err := e.backend().Handle(ctx, e, ev)
			cancel()

			e.mu.Lock()
			if err != nil {
				log.Println("AI DM error:", err)
				e.Log("system", "", "⚠️ The AI Dungeon Master couldn't answer ("+shortErr(err)+"). The offline DM covers this moment.")
				e.offline.Handle(e, ev)
			}
			e.DMBusy = false
			e.changed()
			e.mu.Unlock()
		}
	}
}

func shortErr(err error) string {
	s := err.Error()
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
