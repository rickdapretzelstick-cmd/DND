package engine

import (
	"fmt"
	"os"
	"testing"
)

func TestDumpLog(t *testing.T) {
	if os.Getenv("DUMP") == "" {
		t.Skip()
	}
	e := newTestEngine(t)
	var tokens []string
	for i, c := range []string{"fighter", "wizard", "cleric"} {
		tok, _ := addHero(t, e, []string{"Brann", "Elara", "Tobin"}[i], "human", c)
		tokens = append(tokens, tok)
	}
	e.StartAdventure()
	e.drain()
	cmds := []string{"set out", "look around", "search the room", "listen", "move on", "open the chest", "talk to them", "move on"}
	for step := 0; step < 60; step++ {
		e.mu.Lock()
		if e.C.Combat != nil {
			e.botCombat()
			e.mu.Unlock()
			e.drain()
			continue
		}
		e.mu.Unlock()
		e.Act(tokens[step%3], cmds[step%len(cmds)])
		e.drain()
	}
	for _, l := range e.C.Log {
		fmt.Printf("[%s] %s\n", l.Kind, l.Text)
	}
}
