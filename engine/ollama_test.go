package engine

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaBackend(t *testing.T) {
	var bodies []map[string]any
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			io.WriteString(w, `{"models":[{"name":"qwen3:14b"},{"name":"llama3.1:8b"}]}`)
			return
		}
		b, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		bodies = append(bodies, m)
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch calls {
		case 1:
			io.WriteString(w, `{"message":{"role":"assistant","content":"","tool_calls":[
				{"function":{"name":"roll_check","arguments":{"character_id":"c2","skill":"persuasion","dc":13,"reason":"asking to join"}}},
				{"function":{"name":"set_title","arguments":"{\"character_id\":\"c2\",\"title\":\"Watch Recruit\"}"}},
				{"function":{"name":"rest","arguments":{"kind":"long"}}}]}}`)
		default:
			io.WriteString(w, `{"message":{"role":"assistant","content":"<think>hmm</think>Fine. You start tonight, recruit."}}`)
		}
	}))
	defer srv.Close()

	e := newTestEngine(t)
	e.C.Mode = "ai"
	e.Cfg.Provider = "ollama"
	e.Cfg.OllamaURL = srv.URL
	e.Cfg.fillOllamaDefaults()
	if !e.AIActive() {
		t.Fatal("ollama provider should count as AI active without an API key")
	}
	_, ch := addHero(t, e, "Vex", "non_clan", "projection")
	var guard *NPC
	for _, n := range e.allNPCs() {
		if n.Role == "guard" {
			guard = n
			break
		}
	}
	err := e.backend().Handle(context.Background(), e, DMEvent{Kind: "talk", Summary: guard.ID, Actions: []PendingAction{{CharID: ch.ID, CharName: ch.Name, Text: "Can I join?"}}})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
	first := bodies[0]
	if first["model"] != "qwen3:14b" || first["stream"] != false || first["think"] != false {
		t.Errorf("bad request: model %v stream %v think %v", first["model"], first["stream"], first["think"])
	}
	if opts, _ := first["options"].(map[string]any); opts["num_ctx"] != float64(DefaultOllamaCtx) {
		t.Errorf("num_ctx not set: %v", first["options"])
	}
	if tools, _ := first["tools"].([]any); len(tools) != len(localTools) {
		t.Errorf("expected %d tools, got %d", len(localTools), len(tools))
	}
	msgs, _ := bodies[1]["messages"].([]any)
	if len(msgs) != 2+1+3 {
		t.Fatalf("second request should carry system, user, assistant and 3 tool results; got %d", len(msgs))
	}
	if last := msgs[len(msgs)-1].(map[string]any); !strings.HasPrefix(last["content"].(string), "Error: unknown tool") {
		t.Errorf("tool outside the local set should be refused: %v", last)
	}
	if ch.Title != "Watch Recruit" {
		t.Errorf("string-encoded arguments not handled: title %q", ch.Title)
	}
	l := e.C.Log[len(e.C.Log)-1]
	if l.Kind != "say" || l.Who != guard.Name || strings.Contains(l.Text, "think") {
		t.Errorf("reply not logged cleanly: %+v", l)
	}
	models, has, err := e.OllamaStatus(context.Background())
	if err != nil || !has || len(models) != 2 {
		t.Errorf("status: %v %v %v", models, has, err)
	}
	if p := localSystemPrompt(); !strings.Contains(p, "fly_head") || len(p) > 6000 {
		t.Errorf("local prompt wrong size or missing curses (%d chars)", len(p))
	}
}

func TestOllamaMissingModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		io.WriteString(w, `{"error":"model \"qwen3:14b\" not found, try pulling it first"}`)
	}))
	defer srv.Close()
	e := newTestEngine(t)
	e.C.Mode = "ai"
	e.Cfg.Provider, e.Cfg.OllamaURL = "ollama", srv.URL
	e.Cfg.fillOllamaDefaults()
	addHero(t, e, "Vex", "non_clan", "projection")
	err := e.backend().Handle(context.Background(), e, DMEvent{Kind: "start"})
	if err == nil || !strings.Contains(err.Error(), "ollama pull qwen3:14b") {
		t.Fatalf("expected a pull hint, got %v", err)
	}
	e.Cfg.OllamaURL = "http://127.0.0.1:1"
	if err := e.backend().Handle(context.Background(), e, DMEvent{Kind: "start"}); err == nil || !strings.Contains(err.Error(), "is it running") {
		t.Fatalf("expected a not-running hint, got %v", err)
	}
}
