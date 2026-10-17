package engine

import (
	"strings"
	"testing"
)

func TestImprovNurseAssistantQuest(t *testing.T) {
	e := newTestEngine(t)
	_, ch := addHero(t, e, "Sam", "non_clan", "limitless")
	e.mu.Lock()
	defer e.mu.Unlock()
	shoko := e.npcByID("shoko")
	if shoko == nil {
		t.Fatal("no shoko")
	}
	q, line := e.ImprovQuest(ch, shoko, "can I become your assistant? I want a job")
	if q == nil || len(q.Steps) < 3 {
		t.Fatalf("expected a multi-step quest, got %+v (%s)", q, line)
	}
	if !strings.Contains(q.RewardTitle, "Assistant") || q.RewardSkill == nil {
		t.Fatalf("job should reward a title and a skill: %+v", q)
	}
	q.Stage = "active"
	e.syncQuest(q)
	guard := 0
	for q.Stage == "active" && guard < 20 {
		guard++
		i, st := q.cur()
		if st == nil {
			break
		}
		switch st.Type {
		case "talk", "deliver":
			e.questEvent("talk", ch, st.Target)
		case "wait":
			st.Until = 0
			e.questEvent("hour", ch, "")
		case "defeat":
			if len(q.Encounter) == 0 {
				t.Fatal("defeat step must arm a battle")
			}
			e.questEvent("victory", ch, q.ID)
		case "goto":
			ch.X, ch.Y = float64(st.X), float64(st.Y)
			e.questEvent("arrive", ch, "")
		}
		if j, s2 := q.cur(); s2 != nil && j == i && st.Type != "wait" {
			t.Fatalf("step %d (%s) didn't advance: %+v", i, st.Type, st)
		}
	}
	if q.Stage != "return" {
		t.Fatalf("quest should wait for the report-back, stage=%s", q.Stage)
	}
	if msg := e.turnIn(ch, shoko, q.ID); msg == "" {
		t.Fatal("turn in failed")
	}
	if q.Stage != "done" || ch.Title != q.RewardTitle || len(ch.Custom) != 1 {
		t.Fatalf("rewards missing: stage=%s title=%q skills=%d", q.Stage, ch.Title, len(ch.Custom))
	}
}

func TestDMToolsBuildQuestSkillAndCurse(t *testing.T) {
	e := newTestEngine(t)
	_, ch := addHero(t, e, "Mika", "non_clan", "ten_shadows")
	d := &ClaudeDM{}
	out, isErr := d.runTool(e, "create_curse", `{"name":"Cafeteria Mold","description":"Grew in the staff fridge","theme":"sick","grade":1}`, ch.ID)
	if isErr || !strings.Contains(out, "x_cafeteria_mold") {
		t.Fatalf("create_curse: %s", out)
	}
	out, isErr = d.runTool(e, "create_quest", `{"title":"Fridge Duty","description":"Something's growing.","giver_npc_id":"shoko","report_to_giver":true,"for_character_id":"`+ch.ID+`",
	  "steps":[{"text":"Look around the school","type":"goto","target":"high"},{"text":"Exorcise the mold","type":"defeat","target":"high","curses":[{"type":"x_cafeteria_mold","count":1}]},{"text":"Think it over","type":"custom"}],
	  "reward_skill":{"name":"Iron Stomach","kind":"passive","mod":"hp","amount":99},"reward_xp":120}`, ch.ID)
	if isErr {
		t.Fatalf("create_quest: %s", out)
	}
	e.mu.Lock()
	q := e.C.Quests[len(e.C.Quests)-1]
	e.mu.Unlock()
	if len(q.Steps) != 3 || q.RewardSkill == nil {
		t.Fatalf("quest not built: %+v", q)
	}
	for i := 0; i < 3; i++ {
		if out, isErr := d.runTool(e, "complete_quest_step", `{"quest_id":"`+q.ID+`"}`, ch.ID); isErr {
			t.Fatalf("complete step: %s", out)
		}
	}
	if q.Stage != "return" {
		t.Fatalf("stage = %s", q.Stage)
	}
	out, isErr = d.runTool(e, "create_skill", `{"character_id":"`+ch.ID+`","skill":{"name":"Whistle Blast","kind":"active","effect":"stun","power":2}}`, ch.ID)
	if isErr {
		t.Fatalf("create_skill: %s", out)
	}
	found := false
	for _, a := range ch.KnownAbilities() {
		if a.Name == "Whistle Blast" {
			found = true
		}
	}
	if !found {
		t.Fatal("custom active should appear among known abilities")
	}
	// A fabricated curse can fight in a real battle with the fabricated skill.
	e.mu.Lock()
	_, err := e.StartBattle(BattleOpts{Spawns: []Spawn{{Type: "x_cafeteria_mold", Count: 1}}, Party: []*Character{ch}, Title: "Mold"})
	e.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSkillCapsAreBounded(t *testing.T) {
	cs, err := BuildSkill(3, "cs_x", SkillSpec{Name: "Huge", Kind: "passive", Mod: "ac", Amount: 50})
	if err != nil || cs.Amount != 1 {
		t.Fatalf("ac must be capped at +1: %+v %v", cs, err)
	}
	cs, _ = BuildSkill(3, "cs_y", SkillSpec{Name: "Heals", Kind: "passive", Mod: "heal", Amount: 50})
	if cs.Amount != 2 {
		t.Fatalf("heal cap: %d", cs.Amount)
	}
	if _, err := BuildSkill(3, "cs_z", SkillSpec{Name: "Bad", Kind: "active", Effect: "nuke"}); err == nil {
		t.Fatal("unknown active effect must be rejected")
	}
}
