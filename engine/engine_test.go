package engine

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestRoll(t *testing.T) {
	for i := 0; i < 500; i++ {
		r, err := Roll("2d6+3")
		if err != nil || r.Total < 5 || r.Total > 15 || len(r.Rolls) != 2 {
			t.Fatalf("bad roll %+v %v", r, err)
		}
	}
	for _, bad := range []string{"", "abc", "2d", "1000d6", "d0"} {
		if _, err := Roll(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
	if got := DoubleDice("1d8+2d6+3"); got != "2d8+4d6+3" {
		t.Errorf("DoubleDice = %q", got)
	}
	if got := DiceOnly("2d6+3+1d8"); got != "2d6+1d8" {
		t.Errorf("DiceOnly = %q", got)
	}
	if rollYen("4d6*1000") < 4000 {
		t.Error("rollYen too small")
	}
}

func TestDataIntegrity(t *testing.T) {
	for id, tc := range Techniques {
		if len(tc.Moves) != 4 {
			t.Errorf("%s: %d moves", id, len(tc.Moves))
		}
		if _, ok := AbilityList[tc.Domain]; !ok {
			t.Errorf("%s: missing domain", id)
		}
		if _, ok := ItemList[tc.Weapon]; !ok {
			t.Errorf("%s: unknown weapon %s", id, tc.Weapon)
		}
		if _, ok := ItemList[tc.Armor]; !ok {
			t.Errorf("%s: unknown armor %s", id, tc.Armor)
		}
		if len(tc.Priority) != 6 {
			t.Errorf("%s: priority", id)
		}
	}
	for id, a := range AbilityList {
		for _, ef := range a.Effects {
			if ef.Dice != "" && !strings.Contains(ef.Dice, "weapon") {
				if _, err := Roll(ef.Dice); err != nil {
					t.Errorf("%s: bad dice %q", id, ef.Dice)
				}
			}
			if ef.Save != "" && AbilityNames[ef.Save] == "" {
				t.Errorf("%s: bad save %q", id, ef.Save)
			}
		}
	}
	for id, c := range Bestiary {
		if _, err := Roll(c.HP); err != nil {
			t.Errorf("%s: bad HP %q", id, c.HP)
		}
		if c.Domain != "" {
			if _, ok := AbilityList[c.Domain]; !ok {
				t.Errorf("%s: unknown domain %s", id, c.Domain)
			}
		}
		for _, m := range c.Moves {
			for _, ef := range m.Effects {
				if _, err := Roll(ef.Dice); ef.Dice != "" && err != nil {
					t.Errorf("%s: bad move dice %q", id, ef.Dice)
				}
			}
		}
	}
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	e := newBare(t.TempDir())
	e.cfgPath = t.TempDir() + "/cfg.json"
	e.Cfg.Model = DefaultModel
	e.newCampaign("Test", "", "offline")
	e.C.Seed = 7
	e.W = GenerateWorld(7)
	return e
}

func addHero(t *testing.T, e *Engine, name, origin, tech string) (string, *Character) {
	t.Helper()
	p, err := e.Join(name+"'s player", "")
	if err != nil {
		t.Fatal(err)
	}
	p.Online = true
	stats := AssignByPriority(StandardArray, Techniques[tech])
	if err := e.CreateCharacter(p.Token, CreateReq{Name: name, Race: origin, Class: tech, Stats: stats, Look: Appearance{Hair: 3, Outfit: 1}}); err != nil {
		t.Fatal(err)
	}
	return p.Token, e.C.Characters[p.CharID]
}

func TestCharacterCreation(t *testing.T) {
	e := newTestEngine(t)
	for tid := range Techniques {
		for oid := range Origins {
			_, ch := addHero(t, e, "H"+oid[:2]+tid[:3], oid, tid)
			if ch.HP <= 0 || ch.AC < 10 || ch.Weapon().Damage == "" || (ch.CEMax == 0) != Techniques[tid].NoCE {
				t.Errorf("%s %s: bad character %+v", oid, tid, ch)
			}
			if !e.W.Walkable(ch.X, ch.Y) {
				t.Errorf("spawned on blocked tile")
			}
		}
	}
	p, _ := e.Join("cheater", "")
	err := e.CreateCharacter(p.Token, CreateReq{Name: "X", Race: "gojo", Class: "limitless", Stats: map[string]int{"STR": 18, "DEX": 18, "CON": 18, "INT": 18, "WIS": 18, "CHA": 18}})
	if err == nil {
		t.Error("expected invalid stats to be rejected")
	}
}

// fight runs the current battle with simple bots. Caller must not hold the lock.
func fight(t *testing.T, e *Engine, rotate bool) string {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	turn := 0
	for guard := 0; guard < 500 && e.C.Combat != nil; guard++ {
		cur := e.currentCombatant()
		ch := e.C.Characters[cur.ID]
		if ch == nil {
			t.Fatal("bot asked to act for a monster")
		}
		lm := e.liveMonsters()
		target := ""
		if len(lm) > 0 {
			target = lm[0].ID
		}
		abil := ch.KnownAbilities()
		tried := 0
		for _, i := range rotation(len(abil), turn, rotate) {
			a := abil[i]
			if a.ID == "guard" {
				continue
			}
			tgt := target
			if len(a.Effects) > 0 && (a.Effects[0].Target == "ally" || a.Effects[0].Target == "self") {
				tgt = ch.ID
			}
			if err := e.combatAction(ch, CombatReq{Type: "ability", Ability: a.ID, Target: tgt}); err == nil {
				tried++
				break
			}
		}
		turn++
		if e.C.Combat != nil {
			if cur2 := e.currentCombatant(); cur2 != nil && cur2.ID == ch.ID {
				if err := e.combatAction(ch, CombatReq{Type: "end"}); err != nil {
					t.Fatalf("end turn: %v", err)
				}
			}
		}
	}
	if e.C.Combat != nil {
		t.Fatal("battle never ended")
	}
	for _, l := range e.C.Log[max(0, len(e.C.Log)-6):] {
		if strings.Contains(l.Text, "Exorcism complete") {
			return "victory"
		}
		if strings.Contains(l.Text, "have fallen") {
			return "defeat"
		}
	}
	return "other"
}

func rotation(n, turn int, rotate bool) []int {
	out := make([]int, 0, n)
	if rotate {
		for i := 0; i < n; i++ {
			out = append(out, (turn+i)%n)
		}
		return out
	}
	for i := n - 1; i >= 0; i-- { // strongest (latest) moves first
		out = append(out, i)
	}
	return out
}

func TestEveryTechniqueInBattle(t *testing.T) {
	results := map[string]int{}
	for tid := range Techniques {
		for _, level := range []int{1, 9} {
			e := newTestEngine(t)
			_, ch := addHero(t, e, "Bot", "non_clan", tid)
			if level > 1 {
				ch.GainXP(XPThresholds[level-1])
				ch.HP, ch.CE = ch.MaxHP, ch.CEMax
			}
			ally := "ten_shadows"
			if tid == ally {
				ally = "shrine"
			}
			_, a2 := addHero(t, e, "Ally", "kamo", ally)
			a2.GainXP(ch.XP)
			e.mu.Lock()
			foes := []Spawn{{Type: "crawler", Count: 2}, {Type: "fly_head", Count: 1}}
			if level > 1 {
				foes = []Spawn{{Type: "kuchisake", Count: 1}, {Type: "smallpox", Count: 1, Power: 60}}
			}
			_, err := e.StartBattle(BattleOpts{Spawns: foes, Party: []*Character{ch, a2}, Biome: "school", Title: "Test"})
			e.mu.Unlock()
			if err != nil {
				t.Fatal(err)
			}
			res := fight(t, e, true)
			results[res]++
			// Every move should have produced battle FX/log without panicking.
			if ch.Status == nil || ch.InBattle {
				t.Errorf("%s: state not cleaned up after battle", tid)
			}
		}
	}
	t.Logf("technique battles: %v", results)
}

func TestDomainClashAndBlackFlash(t *testing.T) {
	e := newTestEngine(t)
	_, ch := addHero(t, e, "Gojo2", "gojo", "limitless")
	ch.GainXP(XPThresholds[7])
	ch.HP, ch.CE = ch.MaxHP, ch.CEMax
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, err := e.StartBattle(BattleOpts{Spawns: []Spawn{{Type: "jogo", Count: 1, Power: 60}}, Party: []*Character{ch}, Title: "Clash"}); err != nil {
		t.Fatal(err)
	}
	// Force Jogo's domain, then expand ours on top.
	jogo := e.liveMonsters()[0]
	e.applyAbility(fighter{m: jogo}, AbilityList["iron_mountain"], "", false)
	if e.C.Combat.Domain == nil || e.C.Combat.Domain.Party {
		t.Fatal("Jogo's domain didn't start")
	}
	e.applyAbility(fighter{c: ch}, AbilityList["unlimited_void"], "", false)
	if e.C.Combat == nil {
		return
	}
	found := false
	for _, f := range e.C.Combat.FX {
		if f.Name == "DOMAIN CLASH" {
			found = true
		}
	}
	if !found {
		t.Error("no domain clash FX")
	}
	// Black Flash: force a crit through the zone + many swings.
	bf := false
	for i := 0; i < 200 && e.C.Combat != nil && !bf; i++ {
		jogo.HP = jogo.MaxHP
		ch.Status["zone"] = 99
		hits, _, b := e.resolveEffect(fighter{c: ch}, fighter{m: jogo}, AbilityList["strike"].Effects[0], AbilityList["strike"], false)
		bf = b && len(hits) > 0 && hits[0].BF
	}
	if !bf {
		t.Error("never saw a Black Flash in 200 swings in the zone")
	}
}

// TestStoryPlaythrough walks a party through all five acts via the real dialog/choice APIs.
func TestStoryPlaythrough(t *testing.T) {
	for variant := 0; variant < 3; variant++ {
		e := newTestEngine(t)
		toks := []string{}
		chars := []*Character{}
		for i, tech := range []string{"limitless", "shrine", "ten_shadows"} {
			tok, ch := addHero(t, e, fmt.Sprintf("Hero%d", i), "gojo", tech)
			toks = append(toks, tok)
			chars = append(chars, ch)
		}
		if err := e.StartAdventure(); err != nil {
			t.Fatal(err)
		}
		e.drain()
		talk := func(npcID string, picks ...string) {
			t.Helper()
			e.mu.Lock()
			n := e.npcByID(npcID)
			if n == nil {
				e.mu.Unlock()
				t.Fatalf("no npc %s", npcID)
			}
			for _, ch := range chars {
				ch.X, ch.Y = e.safeSpot(n.X, n.Y+1)
			}
			e.mu.Unlock()
			d, err := e.Interact(toks[0], npcID)
			if err != nil {
				t.Fatalf("interact %s: %v", npcID, err)
			}
			for _, want := range picks {
				id := ""
				for _, o := range d.Options {
					if strings.Contains(o.Text, want) {
						id = o.ID
					}
				}
				if id == "" {
					var opts []string
					for _, o := range d.Options {
						opts = append(opts, o.Text)
					}
					t.Fatalf("%s: no option %q in %v (text %q)", npcID, want, opts, d.Text)
				}
				d, err = e.ChooseDialog(toks[0], id)
				if err != nil {
					t.Fatal(err)
				}
				if d == nil {
					break
				}
			}
			e.drain()
		}
		choose := func(opt string) {
			t.Helper()
			if e.C.Choice == nil {
				t.Fatalf("expected a party choice (act %d)", e.C.Act)
			}
			id := ""
			for _, o := range e.C.Choice.Options {
				if o.ID == opt || strings.Contains(o.Text, opt) {
					id = o.ID
				}
			}
			if id == "" {
				id = e.C.Choice.Options[0].ID
			}
			if err := e.Choose(toks[0], id); err != nil {
				t.Fatal(err)
			}
			e.drain()
		}
		levelUp := func(lvl int) {
			for _, ch := range chars {
				if ch.Level < lvl {
					ch.GainXP(XPThresholds[lvl-1] - ch.XP)
				}
				ch.LongRest()
			}
		}
		goFight := func(act int) {
			t.Helper()
			for attempt := 0; attempt < 8; attempt++ {
				q := e.mainQuest(act)
				if q == nil || q.Stage != "active" {
					return
				}
				e.mu.Lock()
				for _, ch := range chars {
					ch.Dead = false // a fallen bot "rolls a new sorcerer"
					ch.LongRest()
					ch.X, ch.Y = e.safeSpot(float64(q.X), float64(q.Y)+1)
				}
				trigger := chars[0]
				for _, c := range chars {
					if !c.Dead {
						trigger = c
						break
					}
				}
				e.onArrive(trigger)
				started := e.C.Combat != nil
				wiped := len(e.C.Log) > 0 && strings.Contains(e.C.Log[len(e.C.Log)-1].Text, "infirmary")
				e.mu.Unlock()
				if !started && wiped {
					e.drain()
					continue
				}
				if !started {
					for _, c := range chars {
						t.Logf("%s hp %d dead %v inBattle %v at %.1f,%.1f q %d,%d", c.Name, c.HP, c.Dead, c.InBattle, c.X, c.Y, q.X, q.Y)
					}
					for _, l := range e.C.Log[max(0, len(e.C.Log)-4):] {
						t.Log(l.Text)
					}
					t.Fatalf("act %d battle didn't start at %d,%d", act, q.X, q.Y)
				}
				res := fight(t, e, false)
				t.Logf("act %d attempt %d: %s (levels %d)", act, attempt, res, chars[0].Level)
				e.drain()
			}
		}
		// Act 1
		talk("gojo", "first mission", "Leave it to us")
		goFight(1)
		choose([]string{"higher", "gojo", "swallow"}[variant])
		talk("gojo", "Report")
		if e.C.Flags["act1"] != "done" {
			t.Fatalf("act1 not done: %v", e.C.Flags)
		}
		// Act 2
		levelUp(3)
		talk("nanami", "looking for me", "find Junpei")
		talk("junpei", "Can we talk", []string{"lonely", "curse", "Where is he"}[variant])
		goFight(2)
		if e.C.Choice != nil {
			choose("school")
		}
		talk("nanami", "Report")
		// Act 3
		levelUp(5)
		talk("gojo", "Anything big")
		choose([]string{"town", "hanami", "town"}[variant])
		goFight(3)
		talk("gojo", "Report")
		// Act 4
		levelUp(7)
		talk("yaga", "called for us")
		choose([]string{"fight", "talk", "fight"}[variant])
		goFight(4)
		goFight(4)
		talk("yaga", "Report")
		// Act 5
		levelUp(9)
		talk("gojo", "Sensei")
		choose([]string{"with_gojo", "evacuate", "with_gojo"}[variant])
		goFight(5)
		if e.C.Choice != nil { // the vessel's struggle
			choose([]string{"resist", "sukuna", "resist"}[variant])
			goFight(5)
		}
		if e.C.Act != 6 {
			t.Fatalf("variant %d: story didn't reach the epilogue (act %d, flags %v)", variant, e.C.Act, e.C.Flags)
		}
		t.Logf("variant %d ending flags: %v rep %v", variant, e.C.Flags, e.C.Rep)
	}
}

// drain runs queued DM events synchronously with the offline DM.
func (e *Engine) drain() {
	for i := 0; i < 50; i++ {
		e.mu.Lock()
		if len(e.queue) == 0 {
			e.mu.Unlock()
			return
		}
		ev := e.queue[0]
		e.queue = e.queue[1:]
		e.offline.Handle(e, ev)
		e.mu.Unlock()
	}
}

func TestSideQuestAndShops(t *testing.T) {
	e := newTestEngine(t)
	tok, ch := addHero(t, e, "Mai", "zenin", "construction")
	_ = e.StartAdventure()
	e.drain()
	var elder *NPC
	for _, n := range e.allNPCs() {
		if n.Role == "elder" {
			elder = n
			break
		}
	}
	ch.X, ch.Y = e.safeSpot(elder.X, elder.Y+1)
	d, err := e.Interact(tok, elder.ID)
	if err != nil {
		t.Fatal(err)
	}
	pick := func(sub string) {
		t.Helper()
		for _, o := range d.Options {
			if strings.Contains(o.Text, sub) {
				d, err = e.ChooseDialog(tok, o.ID)
				if err != nil {
					t.Fatal(err)
				}
				return
			}
		}
		t.Fatalf("no option %q", sub)
	}
	pick("curse trouble")
	pick("handle it")
	q := e.questFrom(elder.ID, "side")
	if q == nil || q.Stage != "active" || len(q.Encounter) == 0 {
		t.Fatalf("side quest not active: %+v", q)
	}
	ch.GainXP(2000)
	ch.LongRest()
	e.mu.Lock()
	ch.X, ch.Y = e.safeSpot(float64(q.X), float64(q.Y)+1)
	e.onArrive(ch)
	e.mu.Unlock()
	if res := fight(t, e, false); res != "victory" {
		t.Logf("side quest battle: %s", res)
	}
	if e.C.Choice != nil {
		if err := e.Choose(tok, e.C.Choice.Options[0].ID); err != nil {
			t.Fatal(err)
		}
	}
	if q.Stage == "return" {
		ch.X, ch.Y = e.safeSpot(elder.X, elder.Y+1)
		d, _ = e.Interact(tok, elder.ID)
		pick("Report")
		if q.Stage != "done" {
			t.Error("quest not turned in")
		}
	}
	// Shop + inn in the same town.
	for _, n := range e.allNPCs() {
		if n.Town == elder.Town && n.Role == "shop" {
			ch.X, ch.Y = e.safeSpot(n.X, n.Y+1)
			ch.Gold = 50000
			d, _ = e.Interact(tok, n.ID)
			pick("wares")
			pick("Cursed Energy Tonic")
			if !strings.Contains(d.Text, "yours") {
				t.Errorf("buy failed: %q", d.Text)
			}
		}
	}
}

func TestOfflineFreeText(t *testing.T) {
	e := newTestEngine(t)
	tok, ch := addHero(t, e, "Jeff", "non_clan", "boogie_woogie")
	_ = e.StartAdventure()
	e.drain()
	var guard *NPC
	for _, n := range e.allNPCs() {
		if n.Role == "guard" {
			guard = n
			break
		}
	}
	ch.X, ch.Y = e.safeSpot(guard.X, guard.Y+1)
	for _, text := range []string{"I walk up to the guards", "hello guards", "Can I become a guard myself?", "I ask them how to become a guard",
		"I ask if he has anything for sale", "who are you?", "any rumors about curses?", "I meditate and train", "I sense for cursed energy",
		"I eat a snack", "I search the area", "I put up a veil", "I do a backflip", "xyzzy"} {
		before := len(e.C.Log)
		if err := e.Act(tok, text); err != nil {
			t.Fatal(err)
		}
		e.drain()
		if len(e.C.Log) <= before+1 {
			t.Errorf("no response to %q", text)
		}
	}
	if !strings.Contains(ch.Title, "Officer") {
		t.Logf("guard job attempt didn't succeed this time (title %q) — dice", ch.Title)
	}
	if err := e.Talk(tok, guard.ID, "What's the strangest thing you've seen here?"); err != nil {
		t.Fatal(err)
	}
	e.drain()
	last := e.C.Log[len(e.C.Log)-1]
	if last.Kind != "say" || last.Who != guard.Name {
		t.Errorf("NPC didn't answer in a speech bubble: %+v", last)
	}
}

func TestRoamersAndMovement(t *testing.T) {
	e := newTestEngine(t)
	tok, ch := addHero(t, e, "Walker", "non_clan", "ratio")
	e.C.Started = true
	e.mu.Lock()
	for i := 0; i < 50; i++ {
		e.tick(0.2)
	}
	n := len(e.Roamers)
	e.mu.Unlock()
	if n < 20 {
		t.Errorf("only %d roamers", n)
	}
	// Legal step, then an illegal teleport.
	x, y, err := e.Move(tok, ch.X, ch.Y-0.4, 3, true)
	if err != nil || math.Abs(y-(ch.Y)) > 0.01 {
		t.Errorf("legal move rejected: %v %v,%v", err, x, y)
	}
	if _, _, err := e.Move(tok, ch.X+30, ch.Y, 0, true); err == nil {
		t.Error("teleport move accepted")
	}
	// Walk a roamer into the hero to trigger a battle.
	e.mu.Lock()
	r := e.Roamers[0]
	ch.X, ch.Y = e.safeSpot(r.X, r.Y)
	if e.W.InTown(ch.X, ch.Y) == nil {
		r.X, r.Y, r.cool = ch.X, ch.Y, 0
		e.tick(0.2)
		if e.C.Combat == nil && !strings.Contains(e.C.Log[len(e.C.Log)-1].Text, "infirmary") {
			t.Errorf("touching a roamer didn't start a battle: roamer %+v ch %.1f,%.1f town %v log %q", *r, ch.X, ch.Y, e.W.InTown(ch.X, ch.Y), e.C.Log[len(e.C.Log)-1].Text)
		}
	}
	e.mu.Unlock()
	if len(e.PosJSON()) < 50 || len(e.SnapshotJSON()) < 200 {
		t.Error("empty payloads")
	}
	if err := e.Save(); err != nil {
		t.Fatal(err)
	}
}
