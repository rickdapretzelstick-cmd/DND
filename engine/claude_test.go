package engine

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestClaudeToolsLocal(t *testing.T) {
	e := newTestEngine(t)
	_, ch := addHero(t, e, "Rook", "non_clan", "straw_doll")
	d := &ClaudeDM{}
	town := e.W.Towns[2].ID
	site := e.W.Sites[0].ID
	cases := []struct {
		name, input string
		wantErr     bool
	}{
		{"roll_check", `{"character_id":"` + ch.ID + `","skill":"stealth","dc":12,"reason":"sneaking"}`, false},
		{"roll_check", `{"character_id":"nobody","skill":"stealth","dc":12}`, true},
		{"roll_save", `{"character_id":"` + ch.ID + `","ability":"DEX","dc":13,"reason":"rockfall"}`, false},
		{"give_item", `{"character_id":"` + ch.ID + `","item_id":"medicine","quantity":2}`, false},
		{"give_item", `{"character_id":"` + ch.ID + `","name":"Grandma's Lucky Charm","description":"Smells of incense","kind":"gear"}`, false},
		{"give_item", `{"character_id":"` + ch.ID + `","name":"Rusty Naginata","kind":"weapon","damage":"1d10"}`, false},
		{"take_item", `{"character_id":"` + ch.ID + `","item":"grandma's lucky charm"}`, false},
		{"give_yen", `{"character_id":"party","amount":40000,"reason":"reward"}`, false},
		{"give_yen", `{"character_id":"` + ch.ID + `","amount":-500}`, false},
		{"award_xp", `{"amount":300,"reason":"clever plan"}`, false},
		{"damage_character", `{"character_id":"` + ch.ID + `","dice":"1d4","reason":"trap"}`, false},
		{"heal_character", `{"character_id":"` + ch.ID + `","dice":"2d4+2"}`, false},
		{"set_title", `{"character_id":"` + ch.ID + `","title":"Ramen Cook of Akegawa"}`, false},
		{"character_note", `{"character_id":"` + ch.ID + `","note":"Owes Mei Mei a favor"}`, false},
		{"create_npc", `{"name":"Nowhere Man","description":"x","near":"atlantis"}`, false},
		{"npc_note", `{"npc_id":"gojo","note":"Thinks Rook is funny"}`, false},
		{"create_quest", `{"title":"Ramen Thief","description":"Someone steals broth at night","objective":"Catch the thief","location":"` + site + `","curses":[{"type":"bloat_toad","count":1}],"reward_xp":200,"reward_yen":30000}`, false},
		{"create_quest", `{"title":"Bad","description":"x","objective":"x","location":"nowhere"}`, true},
		{"offer_choice", `{"title":"The Broth","speaker":"Tanaka","text":"Share the recipe?","options":["Yes","No"]}`, false},
		{"offer_choice", `{"title":"Again","text":"x","options":["a","b"]}`, true},
		{"set_flag", `{"key":"tanaka_friend","value":"yes"}`, false},
		{"adjust_reputation", `{"faction":"civ","delta":2,"reason":"helped"}`, false},
		{"spawn_curse", `{"curses":["grave_hound","grave_hound"],"near":"` + town + `"}`, false},
		{"spawn_curse", `{"curses":["unicorn"],"near":"` + town + `"}`, true},
		{"move_party", `{"destination":"` + town + `"}`, false},
		{"create_npc", `{"name":"Old Man Tanaka","description":"A retired sorcerer who runs a ramen stand","near":"` + ch.ID + `","role":"shop","hair_color":"#cccccc"}`, false},
		{"advance_time", `{"hours":5}`, false},
		{"update_campaign", `{"quest":"Find the ramen thief","note":"Tanaka knows Gojo."}`, false},
		{"start_battle", `{"curses":[{"type":"dragon_lord","count":1}]}`, true},
		{"start_battle", `{"curses":[{"type":"fly_head","count":1,"name":"Buzzy"}],"surprise":"party","title":"Ambush"}`, false},
		{"end_battle", `{"outcome":"enemies_fled"}`, false},
		{"rest", `{"kind":"long"}`, false},
		{"roll_dice", `{"dice":"1d100","reason":"weather"}`, false},
		{"nope", `{}`, true},
		{"roll_check", `not json`, true},
	}
	for _, c := range cases {
		out, isErr := d.runTool(e, c.name, c.input, ch.ID)
		if isErr != c.wantErr {
			t.Errorf("%s %s: isErr=%v out=%q", c.name, c.input, isErr, out)
		}
	}
	if ch.Title != "Ramen Cook of Akegawa" || len(ch.Notes) != 1 || e.C.Flags["tanaka_friend"] != "yes" || e.C.Rep["civ"] != 2 {
		t.Errorf("tool effects missing: %q %v %v %v", ch.Title, ch.Notes, e.C.Flags, e.C.Rep)
	}
	if len(e.C.ExtraNPCs) < 2 || e.C.Choice == nil {
		t.Errorf("npc/choice not created")
	}
	if !ch.HasItem("custom_rusty_naginata") {
		t.Error("custom weapon missing")
	}
	turn := d.buildTurn(e, DMEvent{Kind: "actions", Actions: []PendingAction{{CharID: ch.ID, CharName: ch.Name, Text: "Can I become a guard?"}}})
	for _, want := range []string{"Can I become a guard?", ch.ID, "<surroundings>", "Reputation", "Old Man Tanaka"} {
		if !strings.Contains(turn, want) {
			t.Errorf("turn prompt missing %q", want)
		}
	}
	if !strings.Contains(systemPromptText(), "Fly Head") || len(dmTools) != 26 {
		t.Errorf("system prompt or tools malformed (%d tools)", len(dmTools))
	}
}

func TestClaudeLoopAgainstFakeAPI(t *testing.T) {
	var mu sync.Mutex
	var bodies []map[string]any
	var betas []string
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		mu.Lock()
		bodies = append(bodies, m)
		betas = append(betas, r.Header.Get("anthropic-beta"))
		calls++
		n := calls
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			io.WriteString(w, `{"id":"msg_1","type":"message","role":"assistant","model":"claude-opus-5-5","stop_reason":"tool_use",
			"content":[{"type":"thinking","thinking":"","signature":"sig"},
			  {"type":"tool_use","id":"tu_1","name":"roll_check","input":{"character_id":"c2","skill":"persuasion","dc":13,"reason":"asking to join the watch"}},
			  {"type":"tool_use","id":"tu_2","name":"set_title","input":{"character_id":"c2","title":"Night Watch Recruit"}}],
			"usage":{"input_tokens":10,"output_tokens":10}}`)
			return
		}
		io.WriteString(w, `{"id":"msg_2","type":"message","role":"assistant","model":"claude-opus-5-5","stop_reason":"end_turn",
		"content":[{"type":"text","text":"Fine. You start tonight, recruit. Bring a flashlight."}],"usage":{"input_tokens":10,"output_tokens":10}}`)
	}))
	defer srv.Close()

	e := newTestEngine(t)
	e.Cfg.APIKey = "sk-test"
	e.Cfg.Effort = "low"
	e.C.Mode = "ai"
	_, ch := addHero(t, e, "Vex", "non_clan", "projection")
	var guard *NPC
	for _, n := range e.allNPCs() {
		if n.Role == "guard" {
			guard = n
			break
		}
	}
	e.ai = &ClaudeDM{baseURL: srv.URL}
	err := e.ai.Handle(context.Background(), e, DMEvent{Kind: "talk", Summary: guard.ID, Actions: []PendingAction{{CharID: ch.ID, CharName: ch.Name, Text: "Can I join the night watch?"}}})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 API calls, got %d", calls)
	}
	first := bodies[0]
	if first["model"] != "claude-opus-5-5" || first["fallbacks"] != "default" {
		t.Errorf("model/fallbacks wrong: %v %v", first["model"], first["fallbacks"])
	}
	if oc, _ := first["output_config"].(map[string]any); oc["effort"] != "low" {
		t.Errorf("effort missing: %v", first["output_config"])
	}
	if _, ok := first["thinking"]; ok {
		t.Error("thinking should be omitted (adaptive by default)")
	}
	if !strings.Contains(betas[0], "server-side-fallback-2026-07-01") {
		t.Errorf("beta header missing: %q", betas[0])
	}
	msgs, _ := bodies[1]["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages on second call, got %d", len(msgs))
	}
	asst := msgs[1].(map[string]any)["content"].([]any)
	if asst[0].(map[string]any)["type"] != "thinking" {
		t.Errorf("thinking block not replayed: %v", asst[0])
	}
	if res := msgs[2].(map[string]any)["content"].([]any); len(res) != 2 {
		t.Errorf("tool results should be in one message, got %d", len(res))
	}
	if ch.Title != "Night Watch Recruit" {
		t.Errorf("title not set: %q", ch.Title)
	}
	last := e.C.Log[len(e.C.Log)-1]
	if last.Kind != "say" || last.Who != guard.Name || !strings.Contains(last.Text, "recruit") {
		t.Errorf("NPC reply not logged as speech: %+v", last)
	}
}

func TestClaudeBadKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	}))
	defer srv.Close()
	e := newTestEngine(t)
	e.Cfg.APIKey = "bad"
	addHero(t, e, "Vex", "non_clan", "projection")
	e.ai = &ClaudeDM{baseURL: srv.URL}
	err := e.ai.Handle(context.Background(), e, DMEvent{Kind: "start"})
	if err == nil || !strings.Contains(err.Error(), "API key was rejected") {
		t.Fatalf("expected friendly key error, got %v", err)
	}
}
