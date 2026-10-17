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
			  {"type":"tool_use","id":"tu_1","name":"roll_check","input":{"character_id":"c2","skill":"perception","dc":12,"reason":"spotting the ambush"}},
			  {"type":"tool_use","id":"tu_2","name":"start_combat","input":{"monsters":[{"type":"goblin","count":2}],"surprise":"none"}}],
			"usage":{"input_tokens":10,"output_tokens":10}}`)
			return
		}
		io.WriteString(w, `{"id":"msg_2","type":"message","role":"assistant","model":"claude-opus-5-5","stop_reason":"end_turn",
		"content":[{"type":"text","text":"Two goblins leap from the brush, blades flashing!"}],"usage":{"input_tokens":10,"output_tokens":10}}`)
	}))
	defer srv.Close()

	e := newTestEngine(t)
	e.Cfg.APIKey = "sk-test"
	e.Cfg.Effort = "low"
	e.C.Mode = "ai"
	tok, ch := addHero(t, e, "Vex", "elf", "ranger")
	_ = tok
	e.ai = &ClaudeDM{baseURL: srv.URL}
	err := e.ai.Handle(context.Background(), e, DMEvent{Kind: "actions", Actions: []PendingAction{{CharID: ch.ID, CharName: ch.Name, Text: "I scan the treeline"}}})
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
		t.Errorf("thinking should be omitted (adaptive by default)")
	}
	if !strings.Contains(betas[0], "server-side-fallback-2026-07-01") {
		t.Errorf("beta header missing: %q", betas[0])
	}
	tools, _ := first["tools"].([]any)
	if len(tools) != 13 {
		t.Errorf("expected 13 tools, got %d", len(tools))
	}
	// Second request: original user turn, assistant turn replayed unchanged (incl. thinking), tool results together.
	msgs, _ := bodies[1]["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages on second call, got %d", len(msgs))
	}
	asst := msgs[1].(map[string]any)["content"].([]any)
	if asst[0].(map[string]any)["type"] != "thinking" {
		t.Errorf("thinking block not replayed: %v", asst[0])
	}
	results := msgs[2].(map[string]any)["content"].([]any)
	if len(results) != 2 {
		t.Errorf("tool results should be in one message, got %d", len(results))
	}
	if e.C.Combat == nil || len(e.C.Combat.Monsters) != 2 {
		t.Errorf("combat not started by tool")
	}
	last := e.C.Log[len(e.C.Log)-1]
	found := false
	for _, l := range e.C.Log {
		if l.Kind == "dm" && strings.Contains(l.Text, "Two goblins") {
			found = true
		}
	}
	if !found {
		t.Errorf("narration not logged; last log: %+v", last)
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
	addHero(t, e, "Vex", "elf", "ranger")
	e.ai = &ClaudeDM{baseURL: srv.URL}
	err := e.ai.Handle(context.Background(), e, DMEvent{Kind: "start"})
	if err == nil || !strings.Contains(err.Error(), "API key was rejected") {
		t.Fatalf("expected friendly key error, got %v", err)
	}
}
