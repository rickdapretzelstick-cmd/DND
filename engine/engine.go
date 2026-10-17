package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
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
	APIKey string `json:"apiKey"`
	Model  string `json:"model"`
	Effort string `json:"effort"`
	Port   int    `json:"port"`
	LastID string `json:"lastCampaign"`
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
	Kind    string // "start", "actions", "combat_end", "defeat", "join"
	Actions []PendingAction
	Summary string
}

type Engine struct {
	mu       sync.Mutex
	C        *Campaign
	Dir      string
	cfgPath  string
	Cfg      Config
	OnChange func()
	DMBusy   bool

	queue   []DMEvent
	wake    chan struct{}
	offline *OfflineDM
	ai      *ClaudeDM
	dirty   bool
}

const DefaultModel = "claude-opus-5-5"

func New(dir string) *Engine {
	e := &Engine{Dir: filepath.Join(dir, "saves"), cfgPath: filepath.Join(dir, "dnd-runner-config.json"),
		wake: make(chan struct{}, 1), offline: &OfflineDM{}, ai: &ClaudeDM{}}
	_ = os.MkdirAll(e.Dir, 0o755)
	if b, err := os.ReadFile(e.cfgPath); err == nil {
		_ = json.Unmarshal(b, &e.Cfg)
	}
	if e.Cfg.Model == "" {
		e.Cfg.Model = DefaultModel
	}
	if e.Cfg.Effort == "" {
		e.Cfg.Effort = "medium"
	}
	if e.Cfg.LastID != "" {
		if c, err := e.readCampaign(e.Cfg.LastID); err == nil {
			e.C = c
		}
	}
	if e.C == nil {
		mode := "offline"
		if e.APIKey() != "" {
			mode = "ai"
		}
		e.C = e.newCampaign("The First Adventure", "", mode)
	}
	for _, p := range e.C.Players {
		p.Online = false
	}
	go e.dmWorker()
	go e.autosaver()
	return e
}

// APIKey returns the configured key, falling back to the environment.
func (e *Engine) APIKey() string {
	if e.Cfg.APIKey != "" {
		return e.Cfg.APIKey
	}
	return os.Getenv("ANTHROPIC_API_KEY")
}

func (e *Engine) AIActive() bool { return e.C.Mode == "ai" && e.APIKey() != "" }

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

func (e *Engine) newCampaign(name, setting, mode string) *Campaign {
	slug := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(slug) > 30 {
		slug = slug[:30]
	}
	if slug == "" {
		slug = "campaign"
	}
	c := &Campaign{ID: fmt.Sprintf("%s-%s", slug, time.Now().Format("20060102-150405")), Name: name,
		Setting: setting, Mode: mode, Created: time.Now(), Players: map[string]*Player{},
		Characters: map[string]*Character{}, Location: "The Crossroads Inn"}
	e.Cfg.LastID = c.ID
	e.saveConfig()
	e.C = c
	e.logf("system", "", "📜 Campaign \"%s\" created. Players: join, make a character, then the host presses Begin Adventure.", name)
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
	if c.Players == nil {
		c.Players = map[string]*Player{}
	}
	if c.Characters == nil {
		c.Characters = map[string]*Character{}
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
			info.Heroes = append(info.Heroes, fmt.Sprintf("%s (L%d %s)", ch.Name, ch.Level, Classes[ch.Class].Name))
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out
}

// carryPlayers keeps everyone connected when the host switches campaigns.
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
	e.queue = nil
	if strings.TrimSpace(name) == "" {
		name = "A New Adventure"
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
	e.queue = nil
	e.C = c
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
	APIKey  *string `json:"apiKey"`
	Model   string  `json:"model"`
	Effort  string  `json:"effort"`
	Mode    string  `json:"mode"`
	Setting *string `json:"setting"`
	Name    string  `json:"name"`
}

func (e *Engine) UpdateSettings(u SettingsUpdate) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if u.APIKey != nil {
		e.Cfg.APIKey = strings.TrimSpace(*u.APIKey)
	}
	if u.Model != "" {
		e.Cfg.Model = strings.TrimSpace(u.Model)
	}
	switch u.Effort {
	case "low", "medium", "high", "xhigh", "max":
		e.Cfg.Effort = u.Effort
	}
	if u.Mode == "ai" || u.Mode == "offline" {
		if u.Mode != e.C.Mode {
			e.C.Mode = u.Mode
			if u.Mode == "ai" {
				e.Log("system", "", "🧠 The AI Dungeon Master takes the screen.")
			} else {
				e.Log("system", "", "🎲 The offline Dungeon Master takes the screen.")
			}
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

type advView struct {
	Title   string `json:"title"`
	Hook    string `json:"hook"`
	Room    string `json:"room"`
	Index   int    `json:"index"`
	Rooms   int    `json:"rooms"`
	Chapter int    `json:"chapter"`
}

type Snapshot struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Setting    string         `json:"setting"`
	Mode       string         `json:"mode"`
	Started    bool           `json:"started"`
	Location   string         `json:"location"`
	SceneDesc  string         `json:"sceneDesc"`
	Quest      string         `json:"quest"`
	Players    []publicPlayer `json:"players"`
	Characters []*Character   `json:"characters"`
	Log        []LogEntry     `json:"log"`
	Combat     *Combat        `json:"combat"`
	Adventure  *advView       `json:"adventure"`
	DMBusy     bool           `json:"dmBusy"`
	AI         bool           `json:"ai"`
	HasKey     bool           `json:"hasKey"`
	Model      string         `json:"model"`
	Effort     string         `json:"effort"`
}

func (e *Engine) SnapshotJSON() []byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	c := e.C
	s := Snapshot{ID: c.ID, Name: c.Name, Setting: c.Setting, Mode: c.Mode, Started: c.Started, Location: c.Location,
		SceneDesc: c.SceneDesc, Quest: c.Quest, Combat: c.Combat, DMBusy: e.DMBusy, AI: e.AIActive(),
		HasKey: e.APIKey() != "", Model: e.Cfg.Model, Effort: e.Cfg.Effort,
		Players: []publicPlayer{}, Characters: []*Character{}, Log: []LogEntry{}}
	for _, p := range c.Players {
		s.Players = append(s.Players, publicPlayer{p.ID, p.Name, p.CharID, p.Online})
	}
	sort.Slice(s.Players, func(i, j int) bool { return s.Players[i].ID < s.Players[j].ID })
	for _, ch := range c.Characters {
		s.Characters = append(s.Characters, ch)
	}
	sort.Slice(s.Characters, func(i, j int) bool { return s.Characters[i].ID < s.Characters[j].ID })
	logs := c.Log
	if len(logs) > 300 {
		logs = logs[len(logs)-300:]
	}
	s.Log = append(s.Log, logs...)
	if a := c.Adventure; a != nil && c.Mode == "offline" {
		v := &advView{Title: a.Title, Hook: a.Hook, Index: a.Index, Rooms: len(a.Rooms), Chapter: a.Chapter}
		if a.Index < len(a.Rooms) {
			v.Room = a.Rooms[a.Index].Name
		}
		s.Adventure = v
	}
	b, _ := json.Marshal(s)
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
	e.logf("system", "", "👋 %s joined the table.", name)
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
	e.logf("roll", p.Name, "🎲 %s rolls ability scores (4d6 drop lowest): %v", p.Name, p.RolledStats)
	return p.RolledStats, nil
}

type CreateReq struct {
	Name      string         `json:"name"`
	Race      string         `json:"race"`
	Class     string         `json:"class"`
	Method    string         `json:"method"`
	Stats     map[string]int `json:"stats"`
	Backstory string         `json:"backstory"`
}

func (e *Engine) CreateCharacter(token string, r CreateReq) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	p := e.playerByToken(token)
	if p == nil {
		return fmt.Errorf("join first")
	}
	if ch, ok := e.C.Characters[p.CharID]; ok && !ch.Dead {
		return fmt.Errorf("you already have a living character (%s)", ch.Name)
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
	ch, err := NewCharacter(e.nextID("c"), p.ID, r.Name, r.Race, r.Class, r.Stats, r.Backstory)
	if err != nil {
		return err
	}
	// Late joiners start near the party's level.
	best := 0
	for _, o := range e.C.Characters {
		if !o.Dead && o.XP > best {
			best = o.XP
		}
	}
	if best > 0 {
		ch.GainXP(best * 3 / 4)
		ch.HP = ch.MaxHP
	}
	e.C.Characters[ch.ID] = ch
	p.CharID = ch.ID
	p.RolledStats = nil
	e.logf("system", "", "🛡️ %s created %s, a level %d %s %s.", p.Name, ch.Name, ch.Level, Races[ch.Race].Name, Classes[ch.Class].Name)
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
	if e.C.Combat != nil {
		return fmt.Errorf("can't retire a character mid-combat")
	}
	delete(e.C.Characters, charID)
	for _, pl := range e.C.Players {
		if pl.CharID == charID {
			pl.CharID = ""
		}
	}
	e.logf("system", "", "🕯️ %s leaves the party.", ch.Name)
	return nil
}

func (e *Engine) charOf(token string) (*Player, *Character, error) {
	p := e.playerByToken(token)
	if p == nil {
		return nil, nil, fmt.Errorf("join first")
	}
	ch, ok := e.C.Characters[p.CharID]
	if !ok {
		return p, nil, fmt.Errorf("create a character first")
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

// Act sends an in-character action to the DM.
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
	if !e.C.Started {
		return fmt.Errorf("the host hasn't begun the adventure yet")
	}
	if ch.Dead {
		return fmt.Errorf("%s is dead — retire them and roll a new hero", ch.Name)
	}
	e.Log("action", ch.Name, text)
	e.enqueue(DMEvent{Kind: "actions", Actions: []PendingAction{{PlayerID: p.ID, CharID: ch.ID, CharName: ch.Name, Text: text}}})
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
	if e.C.Combat != nil {
		if cur := e.currentCombatant(); cur == nil || cur.ID != ch.ID {
			return fmt.Errorf("wait for your turn to swap gear")
		}
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
		return fmt.Errorf("pick another party member")
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

// UseItem drinks a potion (outside combat, or as your action in combat).
func (e *Engine) UseItem(token string, idx int, targetID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ch, err := e.charOf(token)
	if err != nil {
		return err
	}
	if e.C.Combat != nil {
		return e.combatAction(ch, CombatReq{Type: "item", Item: idx, Target: targetID})
	}
	if idx < 0 || idx >= len(ch.Inventory) {
		return fmt.Errorf("no such item")
	}
	return e.drink(ch, idx, targetID)
}

func (e *Engine) drink(ch *Character, idx int, targetID string) error {
	it := ch.Inventory[idx]
	if it.Kind != "potion" || it.Heal == "" {
		return fmt.Errorf("%s can't be used that way — describe what you do with it instead", it.Name)
	}
	if !ch.Conscious() && targetID == ch.ID {
		return fmt.Errorf("you're unconscious")
	}
	target := ch
	if t, ok := e.C.Characters[targetID]; ok {
		target = t
	}
	if target.Dead {
		return fmt.Errorf("%s is beyond a potion's help", target.Name)
	}
	ch.RemoveItem(it.ID, 1)
	r := MustRoll(it.Heal)
	got := target.heal(r.Total)
	if target == ch {
		e.logf("combat", ch.Name, "🧪 %s drinks a %s and heals %d HP (%s).", ch.Name, it.Name, got, r)
	} else {
		e.logf("combat", ch.Name, "🧪 %s pours a %s down %s's throat — %d HP restored (%s).", ch.Name, it.Name, target.Name, got, r)
	}
	return nil
}

// ---------- host controls ----------

func (e *Engine) StartAdventure() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.C.Characters) == 0 {
		return fmt.Errorf("at least one player needs a character first")
	}
	if e.C.Started {
		return fmt.Errorf("the adventure has already begun")
	}
	e.C.Started = true
	e.Log("system", "", "🐉 The adventure begins!")
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
		e.Log("system", "", "🌙 The party takes a long rest. HP, spell slots and abilities are fully restored.")
		return
	}
	var parts []string
	for _, ch := range e.sortedChars() {
		if ch.Dead {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s +%d HP", ch.Name, ch.ShortRest()))
	}
	e.Log("system", "", "☕ The party takes a short rest: "+strings.Join(parts, ", ")+".")
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

// PartyLevel is the average level of the living party.
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

// AwardXP splits XP across living characters and logs level-ups.
func (e *Engine) AwardXP(total int, reason string) {
	living := e.livingChars()
	if total <= 0 || len(living) == 0 {
		return
	}
	each := max(1, total/len(living))
	msg := fmt.Sprintf("✨ The party earns %d XP (%d each)", total, each)
	if reason != "" {
		msg += " — " + reason
	}
	e.Log("system", "", msg+".")
	for _, ch := range living {
		for _, m := range ch.GainXP(each) {
			e.Log("level", ch.Name, m)
		}
	}
}

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
			// Give other players a moment so simultaneous actions resolve together.
			time.Sleep(1200 * time.Millisecond)
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
			e.DMBusy = true
			e.changed()
			if !ai {
				e.offline.Handle(e, ev)
				e.DMBusy = false
				e.changed()
				e.mu.Unlock()
				continue
			}
			e.mu.Unlock()

			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			err := e.ai.Handle(ctx, e, ev)
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
