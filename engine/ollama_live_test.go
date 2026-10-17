package engine

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestOllamaLive talks to a real Ollama server (OLLAMA_LIVE=model, e.g. qwen3:0.6b).
func TestOllamaLive(t *testing.T) {
	model := os.Getenv("OLLAMA_LIVE")
	if model == "" {
		t.Skip("set OLLAMA_LIVE=<model>")
	}
	e := newTestEngine(t)
	e.C.Mode = "ai"
	e.Cfg.Provider, e.Cfg.OllamaModel, e.Cfg.OllamaCtx = "ollama", model, 8192
	e.Cfg.fillOllamaDefaults()
	_, ch := addHero(t, e, "Jeff", "non_clan", "boogie_woogie")
	var guard *NPC
	for _, n := range e.allNPCs() {
		if n.Role == "guard" {
			guard = n
			break
		}
	}
	ch.X, ch.Y = e.safeSpot(guard.X, guard.Y+1)
	for _, ev := range []DMEvent{
		{Kind: "talk", Summary: guard.ID, Actions: []PendingAction{{CharID: ch.ID, CharName: ch.Name, Text: "Can I become a guard myself? I'll do anything."}}},
		{Kind: "actions", Actions: []PendingAction{{CharID: ch.ID, CharName: ch.Name, Text: "I try to sneak into the mayor's house to find out why people keep disappearing"}}},
	} {
		start := time.Now()
		before := len(e.C.Log)
		ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
		err := e.backend().Handle(ctx, e, ev)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range e.C.Log[before:] {
			t.Logf("[%s] %s: %s", l.Kind, l.Who, l.Text)
		}
		t.Logf("took %s; title=%q", time.Since(start).Round(time.Millisecond), ch.Title)
	}
}
