package engine

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"time"
)

const (
	tickRate      = 200 * time.Millisecond
	roamWander    = 1.4 // tiles per second
	roamChase     = 2.7
	aggroRange    = 6.0
	touchRange    = 0.9
	joinRange     = 11.0
	maxStep       = 2.5 // max tiles a client may move between updates
	gameHourSecs  = 50.0
	hubSafeRadius = 6
)

// Move applies a client's position update. It returns the authoritative position.
func (e *Engine) Move(token string, x, y float64, dir int, moving bool) (float64, float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ch, err := e.charOf(token)
	if err != nil {
		return 0, 0, err
	}
	if ch.InBattle || ch.Dead || ch.HP <= 0 {
		ch.Moving = false
		return ch.X, ch.Y, nil
	}
	if math.Hypot(x-ch.X, y-ch.Y) > maxStep || !e.W.Walkable(x, y) {
		return ch.X, ch.Y, fmt.Errorf("blocked")
	}
	ch.X, ch.Y, ch.Dir, ch.Moving = x, y, dir&3, moving
	e.posDirty = true
	e.onArrive(ch)
	return ch.X, ch.Y, nil
}

// onArrive handles discovery and quest triggers when a character moves. Caller holds the lock.
func (e *Engine) onArrive(ch *Character) {
	if t := e.W.InTown(ch.X, ch.Y); t != nil && !e.C.Discovered[t.ID] {
		e.C.Discovered[t.ID] = true
		e.Log("system", ch.Name, fmt.Sprintf("🗺️ %s arrives in %s. %s", ch.Name, t.Name, t.Desc))
		e.C.Location = t.Name
		if e.AIActive() && e.C.Started {
			e.enqueue(DMEvent{Kind: "arrive", Summary: t.ID, Actions: []PendingAction{{CharID: ch.ID, CharName: ch.Name}}})
		}
	}
	if e.C.Combat != nil {
		return
	}
	for _, q := range e.C.Quests {
		if q.Stage != "active" || len(q.Encounter) == 0 {
			continue
		}
		if math.Hypot(ch.X-float64(q.X), ch.Y-float64(q.Y)) < 3.5 {
			if ch.Level < q.MinLevel-1 && q.Kind == "main" {
				if e.warned[q.ID+ch.ID] {
					continue
				}
				e.warned[q.ID+ch.ID] = true
				e.Log("dm", "DM", fmt.Sprintf("A crushing cursed presence pushes %s back. This is beyond a %s sorcerer — come back at level %d.", ch.Name, ch.Grade(), q.MinLevel))
				continue
			}
			e.startQuestBattle(q, ch)
			return
		}
	}
}

func (e *Engine) startQuestBattle(q *Quest, trigger *Character) {
	party := e.nearbyFighters(trigger.X, trigger.Y)
	site := e.W.Site(q.Site)
	biome := e.biomeAt(float64(q.X), float64(q.Y))
	if site != nil {
		biome = site.Kind
	}
	enc := q.Encounter
	if _, err := e.StartBattle(BattleOpts{Spawns: enc, Party: party, X: float64(q.X), Y: float64(q.Y), Biome: biome, Title: q.Title, QuestID: q.ID}); err != nil {
		e.Log("system", "", "Battle failed to start: "+err.Error())
	}
}

func (e *Engine) nearbyFighters(x, y float64) []*Character {
	var out []*Character
	for _, ch := range e.sortedChars() {
		p := e.C.Players[ch.Owner]
		if ch.Dead || ch.HP <= 0 || ch.InBattle || p == nil || !p.Online {
			continue
		}
		if math.Hypot(ch.X-x, ch.Y-y) <= joinRange {
			out = append(out, ch)
		}
	}
	return out
}

func (e *Engine) biomeAt(x, y float64) string {
	for _, s := range e.W.Sites {
		if math.Hypot(float64(s.X)-x, float64(s.Y)-y) < 7 {
			return s.Kind
		}
	}
	switch e.W.At(int(x), int(y)) {
	case TForest, TFlowers:
		return "forest"
	case TPaved, TSidewalk, TPlaza:
		return "city"
	case TSand, TBridge:
		return "shore"
	case TGravel:
		return "school"
	case TCursed:
		return "cursed"
	case THill, TRock:
		return "mountain"
	}
	return "field"
}

// ---------- the world tick ----------

func (e *Engine) worldTicker() {
	t := time.NewTicker(tickRate)
	defer t.Stop()
	for range t.C {
		e.mu.Lock()
		if e.W != nil && e.C.Started {
			e.tick(tickRate.Seconds())
		}
		dirty := e.posDirty
		e.posDirty = false
		e.mu.Unlock()
		if dirty && e.OnPos != nil {
			e.OnPos()
		}
	}
}

func (e *Engine) anyoneOnline() bool {
	for _, p := range e.C.Players {
		if p.Online {
			return true
		}
	}
	return false
}

func (e *Engine) tick(dt float64) {
	if !e.anyoneOnline() {
		return
	}
	prevHour := int(e.C.Clock)
	e.C.Clock += dt / gameHourSecs
	if e.C.Clock >= 24 {
		e.C.Clock -= 24
		e.C.Day++
	}
	if int(e.C.Clock) != prevHour {
		e.posDirty = true
		if int(e.C.Clock) == 20 {
			e.Log("system", "", "🌙 Night falls. Curses grow bolder in the dark.")
		} else if int(e.C.Clock) == 6 {
			e.Log("system", "", "🌅 Dawn breaks.")
		}
	}
	e.populate()
	if e.C.Combat != nil {
		return
	}
	var walkers []*Character
	for _, ch := range e.C.Characters {
		p := e.C.Players[ch.Owner]
		if p != nil && p.Online && ch.Conscious() && !ch.InBattle {
			walkers = append(walkers, ch)
		}
	}
	for _, r := range e.Roamers {
		if r.cool > 0 {
			r.cool--
		}
		// Chase the nearest sorcerer in range, else wander near home.
		var target *Character
		best := aggroRange
		for _, ch := range walkers {
			if d := math.Hypot(ch.X-r.X, ch.Y-r.Y); d < best && e.W.InTown(ch.X, ch.Y) == nil {
				best, target = d, ch
			}
		}
		speed := roamWander
		if target != nil && r.cool == 0 {
			r.tx, r.ty = target.X, target.Y
			speed = roamChase
		} else if math.Hypot(r.tx-r.X, r.ty-r.Y) < 0.3 || (r.tx == 0 && r.ty == 0) || chance(0.01) {
			r.tx = r.HomeX + rand.Float64()*12 - 6
			r.ty = r.HomeY + rand.Float64()*12 - 6
		}
		dx, dy := r.tx-r.X, r.ty-r.Y
		d := math.Hypot(dx, dy)
		if d > 0.05 {
			step := math.Min(d, speed*dt)
			nx, ny := r.X+dx/d*step, r.Y+dy/d*step
			if e.W.Walkable(nx, ny) && e.W.InTown(nx, ny) == nil {
				r.X, r.Y = nx, ny
				e.posDirty = true
			} else {
				r.tx, r.ty = r.HomeX, r.HomeY
			}
		}
		if target != nil && r.cool == 0 && math.Hypot(target.X-r.X, target.Y-r.Y) < touchRange {
			e.roamerBattle(r, target)
			return
		}
	}
}

func (e *Engine) roamerBattle(r *Roamer, trigger *Character) {
	party := e.nearbyFighters(trigger.X, trigger.Y)
	var spawns []Spawn
	counts := map[string]int{}
	var order []string
	for _, k := range r.Group {
		if counts[k] == 0 {
			order = append(order, k)
		}
		counts[k]++
	}
	for _, k := range order {
		spawns = append(spawns, Spawn{Type: k, Count: counts[k], Elite: r.Elite && k == r.Kind})
	}
	title := "Curse encounter"
	if night := e.C.Clock >= 20 || e.C.Clock < 5; night {
		title = "Night ambush"
	}
	if _, err := e.StartBattle(BattleOpts{Spawns: spawns, Party: party, X: r.X, Y: r.Y, Biome: e.biomeAt(r.X, r.Y), Title: title, RoamerID: r.ID}); err != nil {
		r.cool = 30
	}
}

// populate keeps a healthy number of curses roaming the wilds.
func (e *Engine) populate() {
	want := 26 + len(e.W.Sites)
	if e.C.Clock >= 20 || e.C.Clock < 5 {
		want += 10
	}
	tries := 0
	for len(e.Roamers) < want && tries < 6 {
		tries++
		e.spawnRoamer()
	}
}

func (e *Engine) spawnRoamer() {
	hub := e.W.Towns[0]
	var site *Site
	if len(e.W.Sites) > 0 && chance(0.4) {
		s := e.W.Sites[rand.IntN(len(e.W.Sites))]
		near := 0
		for _, r := range e.Roamers {
			if r.Site == s.ID {
				near++
			}
		}
		if near < 2 {
			site = &s
		}
	}
	var x, y float64
	ok := false
	for i := 0; i < 30 && !ok; i++ {
		if site != nil {
			x = float64(site.X) + rand.Float64()*10 - 5
			y = float64(site.Y) + rand.Float64()*10 - 5
		} else {
			x = rand.Float64() * float64(e.W.W)
			y = rand.Float64() * float64(e.W.H)
		}
		if !e.W.Walkable(x, y) {
			continue
		}
		if t, d := e.W.NearestTown(x, y); t != nil && d < float64(t.R)+6 {
			continue
		}
		ok = true
		for _, ch := range e.C.Characters {
			if math.Hypot(ch.X-x, ch.Y-y) < 9 {
				ok = false
			}
		}
	}
	if !ok {
		return
	}
	dist := math.Hypot(x-float64(hub.X), y-float64(hub.Y))
	rank := min(3, int(dist/30))
	tag := e.biomeAt(x, y)
	if site != nil {
		tag = site.Tag
		rank = max(rank, site.Rank)
	}
	rank = min(rank, e.PartyLevel()/2)
	var cands []string
	for _, id := range SortedCurseIDs() {
		t := Bestiary[id]
		if t.Boss || t.Rank != rank || slices.Contains(t.Tags, "special") || slices.Contains(t.Tags, "mahito") || slices.Contains(t.Tags, "kyoto") {
			continue
		}
		if slices.Contains(t.Tags, tag) || slices.Contains(t.Tags, "any") {
			cands = append(cands, id)
		}
	}
	if len(cands) == 0 {
		for _, id := range SortedCurseIDs() {
			t := Bestiary[id]
			if !t.Boss && t.Rank <= rank && !slices.Contains(t.Tags, "special") && !slices.Contains(t.Tags, "mahito") && !slices.Contains(t.Tags, "kyoto") {
				cands = append(cands, id)
			}
		}
	}
	if len(cands) == 0 {
		return
	}
	kind := cands[rand.IntN(len(cands))]
	group := []string{kind}
	party := max(1, len(e.livingChars()))
	maxGroup := party
	if Bestiary[kind].Rank == 0 {
		maxGroup++
	}
	extra := rand.IntN(min(4, maxGroup))
	for i := 0; i < extra; i++ {
		group = append(group, cands[rand.IntN(len(cands))])
	}
	r := &Roamer{ID: e.nextID("r"), Kind: kind, Group: group, X: x, Y: y, HomeX: x, HomeY: y}
	if site != nil {
		r.Site = site.ID
	}
	if chance(0.06) {
		r.Elite = true
	}
	e.Roamers = append(e.Roamers, r)
	e.posDirty = true
}

// SpawnRoamerAt puts a specific curse group on the map (used by the DM).
func (e *Engine) SpawnRoamerAt(x, y float64, group []string, name string, elite bool) (*Roamer, error) {
	if len(group) == 0 {
		return nil, fmt.Errorf("no curses")
	}
	for _, k := range group {
		if _, ok := Bestiary[k]; !ok {
			return nil, fmt.Errorf("unknown curse %q", k)
		}
	}
	for i := 0; i < 40 && !e.W.Walkable(x, y); i++ {
		x += rand.Float64()*4 - 2
		y += rand.Float64()*4 - 2
	}
	r := &Roamer{ID: e.nextID("r"), Kind: group[0], Group: group, X: x, Y: y, HomeX: x, HomeY: y, Name: name, Elite: elite, cool: 15}
	e.Roamers = append(e.Roamers, r)
	e.posDirty = true
	return r, nil
}

func (e *Engine) removeRoamer(id string) {
	e.Roamers = slices.DeleteFunc(e.Roamers, func(r *Roamer) bool { return r.ID == id })
	e.posDirty = true
}

func (e *Engine) roamer(id string) *Roamer {
	for _, r := range e.Roamers {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// ---------- position snapshots ----------

type posChar struct {
	ID     string  `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Dir    int     `json:"d"`
	Moving bool    `json:"m"`
}

type posRoamer struct {
	ID    string  `json:"id"`
	Kind  string  `json:"k"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Elite bool    `json:"e,omitempty"`
	Name  string  `json:"n,omitempty"`
	N     int     `json:"c"`
}

type posNPC struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

// PosJSON is the frequent, lightweight movement update.
func (e *Engine) PosJSON() []byte {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := struct {
		Chars   []posChar   `json:"chars"`
		Roamers []posRoamer `json:"roamers"`
		Clock   float64     `json:"clock"`
		Day     int         `json:"day"`
	}{Chars: []posChar{}, Roamers: []posRoamer{}, Clock: e.C.Clock, Day: e.C.Day}
	for _, ch := range e.C.Characters {
		out.Chars = append(out.Chars, posChar{ch.ID, round2(ch.X), round2(ch.Y), ch.Dir, ch.Moving})
	}
	for _, r := range e.Roamers {
		out.Roamers = append(out.Roamers, posRoamer{r.ID, r.Kind, round2(r.X), round2(r.Y), r.Elite, r.Name, len(r.Group)})
	}
	b, _ := json.Marshal(out)
	return b
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

// Teleport moves characters to a town (Ijichi's car, the DM).
func (e *Engine) teleport(chars []*Character, x, y float64) {
	x, y = e.safeSpot(x, y)
	for i, ch := range chars {
		ch.X, ch.Y = e.safeSpot(x+float64(i%3)-1, y+float64(i/3))
		ch.Moving = false
	}
	e.posDirty = true
}

// safeSpot returns the nearest walkable tile center to (x, y).
func (e *Engine) safeSpot(x, y float64) (float64, float64) {
	if e.W.Walkable(x, y) {
		return x, y
	}
	cx, cy := int(x), int(y)
	for r := 1; r < 20; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue
				}
				if px, py := float64(cx+dx)+0.5, float64(cy+dy)+0.5; e.W.Walkable(px, py) {
					return px, py
				}
			}
		}
	}
	return e.hubSpawn()
}

func (e *Engine) hubSpawn() (float64, float64) {
	return float64(e.W.SpawnX) + 0.5, float64(e.W.SpawnY) + 0.5
}
