package engine

import "math"

// WorldScale is how many tiles one tile of the hand-designed layout becomes. One tile is about a
// metre, so a house is ~12 m wide, a town ~70 m across and the whole map ~800 m: a walk that
// takes minutes, not seconds. The layout is designed small and then enlarged: natural terrain gets
// organic, noise-warped borders; roads, buildings and plazas are scaled exactly.
const WorldScale = 4

func isStructure(t byte) bool {
	switch t {
	case TPaved, TPlaza, TBuilding, TBridge, TWall, TSidewalk, TPath, TGravel, TDirt:
		return true
	}
	return false
}

func (w *World) Upscale(f int) *World {
	if f <= 1 {
		return w
	}
	ow, oh := w.W, w.H
	nw, nh := ow*f, oh*f
	// Natural terrain only: structure tiles take the commonest natural neighbour.
	nat := make([]byte, ow*oh)
	for y := 0; y < oh; y++ {
		for x := 0; x < ow; x++ {
			t := w.At(x, y)
			if !isStructure(t) {
				nat[y*ow+x] = t
				continue
			}
			var cnt [TTileCount]int
			best, bn := byte(TGrass), 0
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					q := w.At(x+dx, y+dy)
					if isStructure(q) {
						continue
					}
					cnt[q]++
					if cnt[q] > bn {
						best, bn = q, cnt[q]
					}
				}
			}
			if t == TBridge {
				best = TWater
			}
			nat[y*ow+x] = best
		}
	}
	noise := valueNoise{seed: uint64(w.Seed) + 77}
	tiles := make([]byte, nw*nh)
	natAt := func(x, y int) byte {
		if x < 0 || y < 0 || x >= ow || y >= oh {
			return TDeep
		}
		return nat[y*ow+x]
	}
	for Y := 0; Y < nh; Y++ {
		for X := 0; X < nw; X++ {
			fx, fy := float64(X)/float64(f), float64(Y)/float64(f)
			// domain warp: wander the border by up to ~0.9 macro tiles
			wx := (noise.fbm(float64(X)/9.0, float64(Y)/9.0, 3) - 0.5) * 1.9
			wy := (noise.fbm(float64(X)/9.0+31.7, float64(Y)/9.0+11.3, 3) - 0.5) * 1.9
			mx, my := int(math.Floor(fx+wx)), int(math.Floor(fy+wy))
			t := natAt(mx, my)
			// keep deep water deep and shallows shallow: don't warp across a shore by more than the data says
			if s := w.At(int(fx), int(fy)); isStructure(s) {
				t = natAt(int(fx), int(fy))
			}
			tiles[Y*nw+X] = t
		}
	}
	for y := 0; y < oh; y++ {
		for x := 0; x < ow; x++ {
			t := w.At(x, y)
			if !isStructure(t) {
				continue
			}
			for dy := 0; dy < f; dy++ {
				for dx := 0; dx < f; dx++ {
					tiles[(y*f+dy)*nw+x*f+dx] = t
				}
			}
		}
	}
	out := &World{Seed: w.Seed, W: nw, H: nh, Tiles: tiles, SpawnX: w.SpawnX*f + f/2, SpawnY: w.SpawnY*f + f/2}
	for _, t := range w.Towns {
		t.X, t.Y, t.R = t.X*f+f/2, t.Y*f+f/2, t.R*f
		out.Towns = append(out.Towns, t)
	}
	for _, b := range w.Buildings {
		b.X, b.Y, b.W, b.H = b.X*f, b.Y*f, b.W*f, b.H*f
		out.Buildings = append(out.Buildings, b)
	}
	for _, s := range w.Sites {
		s.X, s.Y = s.X*f+f/2, s.Y*f+f/2
		out.Sites = append(out.Sites, s)
	}
	for _, n := range w.NPCs {
		n.X, n.Y = n.X*float64(f), n.Y*float64(f)
		out.NPCs = append(out.NPCs, n)
	}
	// make sure nobody starts inside something solid
	for i := range out.NPCs {
		n := &out.NPCs[i]
		if !out.Walkable(n.X, n.Y) {
			if x, y, ok := out.nearestTile(int(n.X), int(n.Y), 12, func(t byte) bool { return walkable[t] }); ok {
				n.X, n.Y = float64(x)+0.5, float64(y)+0.5
			}
		}
	}
	return out
}
