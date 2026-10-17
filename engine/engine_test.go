package engine

import (
	"strings"
	"testing"
	"time"
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
	if r, err := Roll("d20-1+1d4"); err != nil || r.Total < 1 || r.Total > 23 {
		t.Errorf("multi-term roll failed: %+v %v", r, err)
	}
	if got := DoubleDice("1d8+2d6+3"); got != "2d8+4d6+3" {
		t.Errorf("DoubleDice = %q", got)
	}
}

func TestMod(t *testing.T) {
	cases := map[int]int{1: -5, 8: -1, 9: -1, 10: 0, 11: 0, 12: 1, 15: 2, 20: 5}
	for s, want := range cases {
		if got := Mod(s); got != want {
			t.Errorf("Mod(%d)=%d want %d", s, got, want)
		}
	}
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	e := &Engine{Dir: t.TempDir(), cfgPath: t.TempDir() + "/cfg.json", wake: make(chan struct{}, 1), offline: &OfflineDM{}, ai: &ClaudeDM{}}
	e.Cfg.Model = DefaultModel
	e.newCampaign("Test", "", "offline")
	return e
}

func addHero(t *testing.T, e *Engine, name, race, class string) (string, *Character) {
	t.Helper()
	p, err := e.Join(name+"'s player", "")
	if err != nil {
		t.Fatal(err)
	}
	stats := AssignByPriority(StandardArray, Classes[class])
	if err := e.CreateCharacter(p.Token, CreateReq{Name: name, Race: race, Class: class, Stats: stats}); err != nil {
		t.Fatal(err)
	}
	return p.Token, e.C.Characters[p.CharID]
}

func TestCharacterCreation(t *testing.T) {
	e := newTestEngine(t)
	for id := range Classes {
		for rid := range Races {
			_, ch := addHero(t, e, id+rid, rid, id)
			if ch.HP <= 0 || ch.AC < 10 || ch.Weapon().Damage == "" {
				t.Errorf("%s %s: bad character %+v", rid, id, ch)
			}
		}
	}
	p, _ := e.Join("cheater", "")
	err := e.CreateCharacter(p.Token, CreateReq{Name: "X", Race: "human", Class: "fighter", Stats: map[string]int{"STR": 18, "DEX": 18, "CON": 18, "INT": 18, "WIS": 18, "CHA": 18}})
	if err == nil {
		t.Error("expected invalid stats to be rejected")
	}
}

func TestLevelUp(t *testing.T) {
	e := newTestEngine(t)
	_, ch := addHero(t, e, "Wiz", "elf", "wizard")
	hp := ch.MaxHP
	msgs := ch.GainXP(950)
	if ch.Level != 3 || len(msgs) != 2 || ch.MaxHP <= hp || ch.SlotsMax != SlotsFor(Classes["wizard"], 3) {
		t.Errorf("level up wrong: level %d msgs %v hp %d slots %d", ch.Level, msgs, ch.MaxHP, ch.SlotsMax)
	}
}

// TestOfflinePlaythrough lets a bot party play many adventures with the offline DM.
func TestOfflinePlaythrough(t *testing.T) {
	for run := 0; run < 25; run++ {
		e := newTestEngine(t)
		var tokens []string
		classes := []string{"fighter", "wizard", "cleric", "rogue", "barbarian", "ranger", "bard", "paladin"}
		for i := 0; i < 1+run%4; i++ {
			tok, _ := addHero(t, e, "Hero"+string(rune('A'+i)), "human", classes[(run+i)%len(classes)])
			tokens = append(tokens, tok)
		}
		if err := e.StartAdventure(); err != nil {
			t.Fatal(err)
		}
		e.drain()
		cmds := []string{"set out", "look around", "search the room", "listen at the door", "open the chest", "pick the lock",
			"disarm the trap", "talk to them", "short rest", "press on", "sneak ahead", "recall lore", "climb the wall", "buy potion", "move on"}
		chapters := 1
		for step := 0; step < 400; step++ {
			e.mu.Lock()
			if e.C.Combat != nil {
				e.botCombat()
				e.mu.Unlock()
				e.drain()
				continue
			}
			e.mu.Unlock()
			tok := tokens[step%len(tokens)]
			e.mu.Lock()
			_, ch, _ := e.charOf(tok)
			e.mu.Unlock()
			if ch == nil || ch.Dead {
				continue
			}
			cmd := cmds[step%len(cmds)]
			if step%3 == 0 {
				cmd = "move on"
			}
			if err := e.Act(tok, cmd); err != nil {
				t.Fatalf("act %q: %v", cmd, err)
			}
			e.drain()
			e.mu.Lock()
			if e.C.Adventure != nil && e.C.Adventure.Chapter > chapters {
				chapters = e.C.Adventure.Chapter
			}
			alive := len(e.livingChars())
			e.mu.Unlock()
			if alive == 0 {
				break
			}
		}
		if err := e.Save(); err != nil {
			t.Fatal(err)
		}
		if _, err := e.readCampaign(e.C.ID); err != nil {
			t.Fatal(err)
		}
		if len(e.SnapshotJSON()) < 100 {
			t.Fatal("empty snapshot")
		}
		t.Logf("run %d: party of %d reached chapter %d, party level %d, %d log lines", run, len(tokens), chapters, e.PartyLevel(), len(e.C.Log))
	}
}

// drain runs queued DM events synchronously (no API, no debounce).
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

// botCombat plays the current hero's turn. Caller holds the lock.
func (e *Engine) botCombat() {
	cur := e.currentCombatant()
	ch := e.C.Characters[cur.ID]
	lm := e.liveMonsters()
	if ch == nil || len(lm) == 0 {
		return
	}
	// Heal a downed ally if possible.
	for _, o := range e.sortedChars() {
		if !o.Dead && o.HP <= 0 {
			for _, f := range []string{FCureWounds, FHealingWord, FLayOnHands} {
				if ch.HasFeature(f) && e.combatAction(ch, CombatReq{Type: "ability", Ability: f, Target: o.ID}) == nil {
					return
				}
			}
		}
	}
	for _, f := range []string{FRage, FHuntersMark, FDivineSmite} {
		if ch.HasFeature(f) && e.C.Combat != nil && !e.C.Combat.BonusUsed {
			_ = e.combatAction(ch, CombatReq{Type: "ability", Ability: f, Target: lm[0].ID})
		}
	}
	if e.C.Combat == nil || e.currentCombatant().ID != ch.ID {
		return
	}
	var err error
	switch {
	case ch.HasFeature(FMagicMissile) && ch.Slots > 0:
		err = e.combatAction(ch, CombatReq{Type: "ability", Ability: FMagicMissile, Target: lm[0].ID})
	case ch.HasFeature(FFireBolt):
		err = e.combatAction(ch, CombatReq{Type: "ability", Ability: FFireBolt, Target: lm[0].ID})
	case ch.HasFeature(FSacredFlame) && Die(2) == 1:
		err = e.combatAction(ch, CombatReq{Type: "ability", Ability: FSacredFlame, Target: lm[0].ID})
	case ch.HasFeature(FViciousMock) && Die(2) == 1:
		err = e.combatAction(ch, CombatReq{Type: "ability", Ability: FViciousMock, Target: lm[0].ID})
	default:
		err = e.combatAction(ch, CombatReq{Type: "attack", Target: lm[0].ID})
	}
	if err != nil && !strings.Contains(err.Error(), "turn") {
		err = e.combatAction(ch, CombatReq{Type: "attack", Target: lm[0].ID})
	}
	if e.C.Combat != nil && e.currentCombatant().ID == ch.ID {
		_ = e.combatAction(ch, CombatReq{Type: "end"})
	}
}

func TestClaudeToolsLocal(t *testing.T) {
	e := newTestEngine(t)
	_, ch := addHero(t, e, "Rook", "halfling", "rogue")
	d := &ClaudeDM{}
	cases := []struct {
		name, input string
		wantErr     bool
	}{
		{"roll_check", `{"character_id":"` + ch.ID + `","skill":"stealth","dc":12,"reason":"sneaking"}`, false},
		{"roll_check", `{"character_id":"nobody","skill":"stealth","dc":12}`, true},
		{"roll_save", `{"character_id":"` + ch.ID + `","ability":"DEX","dc":13,"reason":"rockfall"}`, false},
		{"give_item", `{"character_id":"` + ch.ID + `","item_id":"potion_healing","quantity":2}`, false},
		{"give_item", `{"character_id":"` + ch.ID + `","name":"Map of the Undercity","description":"Shows a hidden passage"}`, false},
		{"take_item", `{"character_id":"` + ch.ID + `","item":"map of the undercity"}`, false},
		{"give_gold", `{"character_id":"party","amount":40,"reason":"reward"}`, false},
		{"give_gold", `{"character_id":"` + ch.ID + `","amount":-5}`, false},
		{"award_xp", `{"amount":300,"reason":"clever plan"}`, false},
		{"damage_character", `{"character_id":"` + ch.ID + `","dice":"1d4","reason":"trap"}`, false},
		{"heal_character", `{"character_id":"` + ch.ID + `","dice":"2d4+2"}`, false},
		{"update_campaign", `{"quest":"Find the lost heir","location":"Old Mill","note":"The miller lies."}`, false},
		{"start_combat", `{"monsters":[{"type":"dragon_lord","count":1}]}`, true},
		{"start_combat", `{"monsters":[{"type":"goblin","count":3},{"type":"goblin","count":1,"elite":true,"name":"Grisk"}],"surprise":"party"}`, false},
		{"end_combat", `{"outcome":"surrender"}`, false},
		{"rest", `{"kind":"long"}`, false},
		{"roll_dice", `{"dice":"1d100","reason":"weather"}`, false},
		{"nope", `{}`, true},
		{"roll_check", `not json`, true},
	}
	for _, c := range cases {
		out, isErr := d.runTool(e, c.name, c.input)
		if isErr != c.wantErr {
			t.Errorf("%s %s: isErr=%v out=%q", c.name, c.input, isErr, out)
		}
	}
	if ch.Level < 2 || e.C.Quest != "Find the lost heir" || len(e.C.Journal) != 1 {
		t.Errorf("tool effects missing: level %d quest %q journal %v", ch.Level, e.C.Quest, e.C.Journal)
	}
	turn := d.buildTurn(e, DMEvent{Kind: "actions", Actions: []PendingAction{{CharID: ch.ID, CharName: ch.Name, Text: "I pick the lock"}}})
	if !strings.Contains(turn, "I pick the lock") || !strings.Contains(turn, ch.ID) {
		t.Errorf("turn prompt missing content:\n%s", turn)
	}
	if !strings.Contains(systemPrompt, "goblin") || len(dmTools) != 13 {
		t.Errorf("system prompt or tools malformed")
	}
	_ = time.Now
}
