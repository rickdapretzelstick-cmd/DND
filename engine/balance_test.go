package engine

import (
	"math/rand/v2"
	"os"
	"testing"
)

// TestBalance reports win rates of each story fight at its recommended level (BALANCE=1).
func TestBalance(t *testing.T) {
	if os.Getenv("BALANCE") == "" {
		t.Skip("set BALANCE=1")
	}
	techs := SortedTechniqueIDs()
	fights := []struct {
		name  string
		level int
		enc   func(size int) []Spawn
	}{
		{"act1 finger bearer", 1, func(s int) []Spawn {
			return act1Encounter(s)
		}},
		{"act2 mahito", 3, act2Encounter},
		{"act3 jogo", 5, act3Jogo},
		{"act3 hanami", 5, act3Hanami},
		{"act4 kyoto", 7, act4Kyoto},
		{"act4 dagon", 7, func(s int) []Spawn { return act4Invader("dagon", s) }},
		{"act4 hanami", 7, func(s int) []Spawn { return act4Invader("hanami", s) }},
		{"act5 kenjaku", 9, func(s int) []Spawn { return act5Encounter(s, false, true, true) }},
		{"act5 kenjaku+gojo", 9, func(s int) []Spawn { return act5Encounter(s, true, false, false) }},
		{"act5 sukuna", 9, func(s int) []Spawn { return []Spawn{{Type: "sukuna", Count: 1, Power: sukunaPower(s)}} }},
	}
	for _, f := range fights {
		for _, size := range []int{1, 3} {
			wins := 0
			const runs = 40
			for i := 0; i < runs; i++ {
				e := newTestEngine(t)
				var party []*Character
				for j := 0; j < size; j++ {
					_, ch := addHero(t, e, "B", "non_clan", techs[rand.IntN(len(techs))])
					if f.level > 1 {
						ch.GainXP(XPThresholds[f.level-1])
					}
					ch.LongRest()
					party = append(party, ch)
				}
				e.mu.Lock()
				_, err := e.StartBattle(BattleOpts{Spawns: f.enc(size), Party: party, Title: f.name})
				e.mu.Unlock()
				if err != nil {
					t.Fatal(err)
				}
				if fight(t, e, false) == "victory" {
					wins++
				}
			}
			t.Logf("%-20s party %d @ L%d: %3d%% wins", f.name, size, f.level, wins*100/runs)
		}
	}
}
