package engine

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestWorldGen(t *testing.T) {
	for seed := int64(1); seed <= 30; seed++ {
		w := GenerateWorld(seed)
		if len(w.Towns) < 6 {
			t.Errorf("seed %d: only %d towns", seed, len(w.Towns))
		}
		if len(w.Sites) < 8 {
			t.Errorf("seed %d: only %d sites", seed, len(w.Sites))
		}
		if !w.Walkable(float64(w.SpawnX)+0.5, float64(w.SpawnY)+0.5) {
			t.Errorf("seed %d: spawn not walkable", seed)
		}
		// Every town and site must be reachable on foot from the spawn.
		reach := w.reachable(w.SpawnX, w.SpawnY)
		for _, tw := range w.Towns {
			if !nearReach(w, reach, tw.X, tw.Y, 3) {
				t.Errorf("seed %d: town %s unreachable", seed, tw.Name)
			}
		}
		for _, s := range w.Sites {
			if !nearReach(w, reach, s.X, s.Y+1, 3) {
				t.Errorf("seed %d: site %s unreachable", seed, s.Name)
			}
		}
		for _, n := range w.NPCs {
			if !w.Walkable(n.X, n.Y) && n.Role != "gojo" {
				t.Errorf("seed %d: npc %s on blocked tile", seed, n.Name)
			}
		}
	}
	if p := os.Getenv("WORLD_DUMP"); p != "" {
		b, _ := json.Marshal(GenerateWorld(7))
		os.WriteFile(p, b, 0o644)
	}
}

func nearReach(w *World, reach []bool, x, y, r int) bool {
	for yy := y - r; yy <= y+r; yy++ {
		for xx := x - r; xx <= x+r; xx++ {
			if xx >= 0 && yy >= 0 && xx < w.W && yy < w.H && reach[yy*w.W+xx] && math.Hypot(float64(xx-x), float64(yy-y)) <= float64(r) {
				return true
			}
		}
	}
	return false
}
