package engine

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
)

// Tile types. Keep in sync with web/js/tiles.js.
const (
	TDeep = iota
	TWater
	TSand
	TGrass
	TFlowers
	TForest
	TRock
	THill
	TDirt
	TPaved
	TPlaza
	TBuilding
	TBridge
	TCursed
	TField
	TSidewalk
	TPeak
	TPath
	TGravel
	TWall
	TTileCount
)

var walkable = [TTileCount]bool{
	TSand: true, TGrass: true, TFlowers: true, THill: true, TDirt: true, TPaved: true, TPlaza: true,
	TBridge: true, TCursed: true, TField: true, TSidewalk: true, TPath: true, TGravel: true,
}

const WorldW, WorldH = 200, 200

type Town struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"` // school, city, town, village
	X    int    `json:"x"`
	Y    int    `json:"y"`
	R    int    `json:"r"`
	Desc string `json:"desc"`
}

type Building struct {
	ID    int    `json:"id"`
	Kind  string `json:"kind"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	Town  string `json:"town"`
	Name  string `json:"name"`
	Color int    `json:"color"`
}

type Site struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Rank int    `json:"rank"`
	Town string `json:"town"`
	Tag  string `json:"tag"`
}

type NPC struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Role  string     `json:"role"` // gojo, yaga, shoko, ijichi, nanami, elder, shop, inn, villager, guard, monk, custom...
	X     float64    `json:"x"`
	Y     float64    `json:"y"`
	Town  string     `json:"town"`
	Look  Appearance `json:"look"`
	Lines []string   `json:"lines,omitempty"`
	Desc  string     `json:"desc,omitempty"`
}

type World struct {
	Seed      int64      `json:"seed"`
	W         int        `json:"w"`
	H         int        `json:"h"`
	Tiles     []byte     `json:"tiles"`
	Towns     []Town     `json:"towns"`
	Buildings []Building `json:"buildings"`
	Sites     []Site     `json:"sites"`
	NPCs      []NPC      `json:"npcs"`
	SpawnX    int        `json:"spawnX"`
	SpawnY    int        `json:"spawnY"`
}

func (w *World) At(x, y int) byte {
	if x < 0 || y < 0 || x >= w.W || y >= w.H {
		return TDeep
	}
	return w.Tiles[y*w.W+x]
}

func (w *World) Set(x, y int, t byte) {
	if x < 0 || y < 0 || x >= w.W || y >= w.H {
		return
	}
	w.Tiles[y*w.W+x] = t
}

func (w *World) Walkable(x, y float64) bool {
	t := w.At(int(math.Floor(x)), int(math.Floor(y)))
	return walkable[t]
}

func (w *World) Town(id string) *Town {
	for i := range w.Towns {
		if w.Towns[i].ID == id {
			return &w.Towns[i]
		}
	}
	return nil
}

func (w *World) Site(id string) *Site {
	for i := range w.Sites {
		if w.Sites[i].ID == id {
			return &w.Sites[i]
		}
	}
	return nil
}

// NearestTown returns the closest town and its distance.
func (w *World) NearestTown(x, y float64) (*Town, float64) {
	var best *Town
	bd := math.MaxFloat64
	for i := range w.Towns {
		t := &w.Towns[i]
		d := math.Hypot(float64(t.X)-x, float64(t.Y)-y)
		if d < bd {
			best, bd = t, d
		}
	}
	return best, bd
}

// InTown reports whether a point is inside a town's safe radius.
func (w *World) InTown(x, y float64) *Town {
	t, d := w.NearestTown(x, y)
	if t != nil && d <= float64(t.R)+2 {
		return t
	}
	return nil
}

// ---------- noise ----------

type valueNoise struct{ seed uint64 }

func (n valueNoise) hash(x, y int) float64 {
	h := uint64(x)*0x9E3779B97F4A7C15 ^ uint64(y)*0xC2B2AE3D27D4EB4F ^ n.seed
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return float64(h%1_000_000) / 1_000_000
}

func smooth(t float64) float64 { return t * t * (3 - 2*t) }

func (n valueNoise) at(x, y float64) float64 {
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := smooth(x-float64(x0)), smooth(y-float64(y0))
	a := n.hash(x0, y0)
	b := n.hash(x0+1, y0)
	c := n.hash(x0, y0+1)
	d := n.hash(x0+1, y0+1)
	return (a*(1-fx)+b*fx)*(1-fy) + (c*(1-fx)+d*fx)*fy
}

func (n valueNoise) fbm(x, y float64, oct int) float64 {
	sum, amp, freq, norm := 0.0, 1.0, 1.0, 0.0
	for i := 0; i < oct; i++ {
		sum += amp * n.at(x*freq, y*freq)
		norm += amp
		amp *= 0.5
		freq *= 2
	}
	return sum / norm
}

// ---------- generation ----------

var townNames = []string{"Akegawa", "Hoshimura", "Kumodani", "Shiranui", "Tsukinomori", "Yamabuki", "Sugisawa", "Satozakura", "Mizuhara", "Kagerou", "Hinokawa", "Yasohachi"}

var siteKinds = []struct {
	Kind, Name, Tag string
}{
	{"school", "Abandoned %s Middle School", "school"},
	{"hospital", "Closed %s General Hospital", "hospital"},
	{"tunnel", "%s Old Railway Tunnel", "tunnel"},
	{"shrine", "Ruined %s Shrine", "shrine"},
	{"cemetery", "%s Hillside Cemetery", "grave"},
	{"factory", "Derelict %s Factory", "factory"},
	{"apartment", "Vacant %s Apartments", "city"},
	{"grove", "%s Suicide Grove", "forest"},
	{"lake", "%s Lakeshore Pier", "water"},
	{"detention", "%s Juvenile Detention Center", "school"},
}

// GenerateWorld builds the overworld deterministically from a seed.
func GenerateWorld(seed int64) *World {
	r := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0xA5A5))
	w := &World{Seed: seed, W: WorldW, H: WorldH, Tiles: make([]byte, WorldW*WorldH)}
	elev := valueNoise{uint64(seed) * 31}
	moist := valueNoise{uint64(seed)*131 + 7}
	detail := valueNoise{uint64(seed)*977 + 3}
	cx, cy := float64(w.W)/2, float64(w.H)/2
	for y := 0; y < w.H; y++ {
		for x := 0; x < w.W; x++ {
			dx, dy := (float64(x)-cx)/cx, (float64(y)-cy)/cy
			d := math.Sqrt(dx*dx + dy*dy)
			e := elev.fbm(float64(x)/42, float64(y)/42, 5) - 0.55*d*d + 0.18
			m := moist.fbm(float64(x)/30, float64(y)/30, 4)
			var t byte
			switch {
			case e < 0.30:
				t = TDeep
			case e < 0.35:
				t = TWater
			case e < 0.385:
				t = TSand
			case e > 0.80:
				t = TPeak
			case e > 0.72:
				t = TRock
			case e > 0.64:
				t = THill
			case m > 0.60:
				t = TForest
			case m > 0.54:
				if detail.at(float64(x)/3, float64(y)/3) > 0.45 {
					t = TForest
				} else {
					t = TFlowers
				}
			case m < 0.36 && e < 0.5:
				t = TField
			default:
				t = TGrass
				if detail.at(float64(x)/2.5, float64(y)/2.5) > 0.8 {
					t = TFlowers
				}
			}
			w.Tiles[y*w.W+x] = t
		}
	}

	landOK := func(x, y, rad int) bool {
		n, land := 0, 0
		for yy := y - rad; yy <= y+rad; yy++ {
			for xx := x - rad; xx <= x+rad; xx++ {
				n++
				t := w.At(xx, yy)
				if t != TDeep && t != TWater && t != TPeak {
					land++
				}
			}
		}
		return float64(land)/float64(n) > 0.92
	}
	findSpot := func(minD, maxD float64, ox, oy, rad int, others []Town, spacing float64) (int, int, bool) {
		for tries := 0; tries < 4000; tries++ {
			a := r.Float64() * 2 * math.Pi
			dist := minD + r.Float64()*(maxD-minD)
			x, y := ox+int(math.Cos(a)*dist), oy+int(math.Sin(a)*dist)
			if x < rad+4 || y < rad+4 || x >= w.W-rad-4 || y >= w.H-rad-4 || !landOK(x, y, rad) {
				continue
			}
			ok := true
			for _, o := range others {
				if math.Hypot(float64(o.X-x), float64(o.Y-y)) < spacing+float64(o.R) {
					ok = false
					break
				}
			}
			if ok {
				return x, y, true
			}
		}
		return 0, 0, false
	}

	// Jujutsu High near the center.
	hx, hy, ok := findSpot(0, 30, int(cx), int(cy), 10, nil, 0)
	if !ok {
		hx, hy = int(cx), int(cy)
		w.clear(hx, hy, 12, TGrass)
	}
	towns := []Town{{ID: "high", Name: "Tokyo Jujutsu High", Kind: "school", X: hx, Y: hy, R: 9,
		Desc: "A secluded school in the hills, disguised as a religious college. Home base."}}
	if x, y, ok := findSpot(45, 85, hx, hy, 16, towns, 30); ok {
		towns = append(towns, Town{ID: "city", Name: "Shibuya", Kind: "city", X: x, Y: y, R: 16, Desc: "A dense ward of towers, crossings and train lines. Where curses gather in crowds."})
	}
	names := append([]string(nil), townNames...)
	r.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })
	for i := 0; len(towns) < 8 && i < 40; i++ {
		rad := 7 + r.IntN(3)
		x, y, ok := findSpot(28, 92, hx, hy, rad, towns, 24)
		if !ok {
			continue
		}
		kind := "town"
		if rad <= 7 {
			kind = "village"
		}
		n := names[len(towns)%len(names)]
		towns = append(towns, Town{ID: fmt.Sprintf("t%d", len(towns)), Name: n, Kind: kind, X: x, Y: y, R: rad,
			Desc: pickR(r, []string{"A quiet farming %s with an old shrine.", "A riverside %s famous for its festival.", "A sleepy %s where people disappear more often than they should.", "A shopping-street %s with a busy station.", "A hot-spring %s nestled in the hills."}, kind)})
	}
	w.Towns = towns

	bid := 0
	for _, t := range towns {
		switch t.Kind {
		case "school":
			bid = w.layoutHigh(t, bid)
		case "city":
			bid = w.layoutCity(r, t, bid)
		default:
			bid = w.layoutTown(r, t, bid)
		}
	}
	entrance := func(t Town) (int, int) {
		if t.Kind == "school" {
			return t.X, t.Y + t.R + 4
		}
		return t.X, t.Y
	}

	// Roads between towns (MST + a couple of extra links).
	type edge struct {
		a, b int
		d    float64
	}
	var edges []edge
	for i := range towns {
		for j := i + 1; j < len(towns); j++ {
			edges = append(edges, edge{i, j, math.Hypot(float64(towns[i].X-towns[j].X), float64(towns[i].Y-towns[j].Y))})
		}
	}
	sort.Slice(edges, func(i, j int) bool { return edges[i].d < edges[j].d })
	parent := make([]int, len(towns))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	var roads []edge
	extra := 0
	for _, e := range edges {
		if find(e.a) != find(e.b) {
			parent[find(e.a)] = find(e.b)
			roads = append(roads, e)
		} else if extra < 2 && e.d < 75 {
			roads = append(roads, e)
			extra++
		}
	}
	for _, e := range roads {
		tile := byte(TDirt)
		if towns[e.a].Kind == "city" || towns[e.b].Kind == "city" {
			tile = TPaved
		}
		ax, ay := entrance(towns[e.a])
		bx, by := entrance(towns[e.b])
		w.road(ax, ay, bx, by, tile, 2)
	}

	// Cursed sites in the wilds.
	kinds := append(siteKinds[:0:0], siteKinds...)
	r.Shuffle(len(kinds), func(i, j int) { kinds[i], kinds[j] = kinds[j], kinds[i] })
	var placed []Town
	placed = append(placed, towns...)
	for i := 0; len(w.Sites) < 14 && i < 60; i++ {
		k := kinds[len(w.Sites)%len(kinds)]
		x, y, ok := findSpot(20, 95, hx, hy, 4, placed, 12)
		if !ok {
			continue
		}
		near := w.nearestNamedTown(float64(x), float64(y))
		d := math.Hypot(float64(x-hx), float64(y-hy))
		rank := min(3, int(d/28))
		s := Site{ID: fmt.Sprintf("s%d", len(w.Sites)), Kind: k.Kind, Tag: k.Tag, X: x, Y: y, Rank: rank, Town: near.ID,
			Name: fmt.Sprintf(k.Name, near.Name)}
		w.Sites = append(w.Sites, s)
		placed = append(placed, Town{X: x, Y: y, R: 4})
		// Cursed ground, a landmark and a path to the nearest road.
		for yy := y - 5; yy <= y+5; yy++ {
			for xx := x - 5; xx <= x+5; xx++ {
				dd := math.Hypot(float64(xx-x), float64(yy-y))
				t := w.At(xx, yy)
				if dd < 5.2 && walkable[t] || (dd < 4 && (t == TForest || t == TFlowers || t == TRock)) {
					w.Set(xx, yy, TCursed)
				}
			}
		}
		bw, bh := 5, 4
		if k.Kind == "shrine" || k.Kind == "tunnel" || k.Kind == "grove" || k.Kind == "lake" || k.Kind == "cemetery" {
			bw, bh = 3, 3
		}
		w.Buildings = append(w.Buildings, Building{ID: bid, Kind: "site_" + k.Kind, X: x - bw/2, Y: y - bh - 1, W: bw, H: bh, Name: s.Name, Town: s.ID})
		bid++
		w.fill(x-bw/2, y-bh-1, bw, bh, TBuilding)
		if rx, ry, ok := w.nearestTile(x, y+2, 25, func(t byte) bool { return t == TDirt || t == TPaved }); ok {
			w.road(x, y+1, rx, ry, TPath, 1)
		}
	}

	// Spawn at the High's gate; make sure it's walkable.
	w.SpawnX, w.SpawnY = hx, hy+7
	w.Set(w.SpawnX, w.SpawnY, TGravel)
	w.connectAll()
	w.placeNPCs(r)
	return w
}

func (w *World) nearestNamedTown(x, y float64) *Town {
	var best *Town
	bd := math.MaxFloat64
	for i := range w.Towns {
		t := &w.Towns[i]
		if t.Kind == "school" && len(w.Towns) > 1 {
			continue
		}
		if d := math.Hypot(float64(t.X)-x, float64(t.Y)-y); d < bd {
			best, bd = t, d
		}
	}
	return best
}

// connectAll guarantees every town and site can be reached on foot from the spawn.
func (w *World) connectAll() {
	type target struct{ x, y int }
	var targets []target
	for _, t := range w.Towns {
		targets = append(targets, target{t.X, t.Y + 1})
		if t.Kind == "school" {
			targets[len(targets)-1] = target{t.X, t.Y + t.R + 3}
		}
	}
	for _, s := range w.Sites {
		targets = append(targets, target{s.X, s.Y + 1})
	}
	for pass := 0; pass < len(targets); pass++ {
		reach := w.reachable(w.SpawnX, w.SpawnY)
		fixed := true
		for _, t := range targets {
			if reach[t.y*w.W+t.x] || !walkable[w.At(t.x, t.y)] && w.nearReachable(reach, t.x, t.y, 2) {
				continue
			}
			// Path from the target to the nearest reachable tile.
			bx, by, bd := -1, -1, math.MaxFloat64
			for i, ok := range reach {
				if ok {
					x, y := i%w.W, i/w.W
					if d := math.Hypot(float64(x-t.x), float64(y-t.y)); d < bd {
						bx, by, bd = x, y, d
					}
				}
			}
			if bx >= 0 {
				if !walkable[w.At(t.x, t.y)] {
					w.Set(t.x, t.y, TPath)
				}
				w.road(t.x, t.y, bx, by, TPath, 1)
				fixed = false
				break
			}
		}
		if fixed {
			return
		}
	}
}

func (w *World) nearReachable(reach []bool, x, y, r int) bool {
	for yy := y - r; yy <= y+r; yy++ {
		for xx := x - r; xx <= x+r; xx++ {
			if xx >= 0 && yy >= 0 && xx < w.W && yy < w.H && reach[yy*w.W+xx] {
				return true
			}
		}
	}
	return false
}

func (w *World) reachable(sx, sy int) []bool {
	seen := make([]bool, w.W*w.H)
	q := []int{sy*w.W + sx}
	seen[q[0]] = true
	for len(q) > 0 {
		i := q[0]
		q = q[1:]
		x, y := i%w.W, i/w.W
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx < 0 || ny < 0 || nx >= w.W || ny >= w.H {
				continue
			}
			j := ny*w.W + nx
			if !seen[j] && walkable[w.Tiles[j]] {
				seen[j] = true
				q = append(q, j)
			}
		}
	}
	return seen
}

func pickR(r *rand.Rand, xs []string, arg string) string {
	return fmt.Sprintf(xs[r.IntN(len(xs))], arg)
}

func pickS(r *rand.Rand, xs []string) string { return xs[r.IntN(len(xs))] }

func (w *World) clear(cx, cy, rad int, t byte) {
	for y := cy - rad; y <= cy+rad; y++ {
		for x := cx - rad; x <= cx+rad; x++ {
			if math.Hypot(float64(x-cx), float64(y-cy)) <= float64(rad)+0.5 {
				w.Set(x, y, t)
			}
		}
	}
}

func (w *World) fill(x, y, ww, hh int, t byte) {
	for yy := y; yy < y+hh; yy++ {
		for xx := x; xx < x+ww; xx++ {
			w.Set(xx, yy, t)
		}
	}
}

func (w *World) rectFree(x, y, ww, hh int, ok func(byte) bool) bool {
	for yy := y; yy < y+hh; yy++ {
		for xx := x; xx < x+ww; xx++ {
			if !ok(w.At(xx, yy)) {
				return false
			}
		}
	}
	return true
}

func (w *World) nearestTile(cx, cy, maxR int, ok func(byte) bool) (int, int, bool) {
	for rad := 1; rad <= maxR; rad++ {
		for y := cy - rad; y <= cy+rad; y++ {
			for x := cx - rad; x <= cx+rad; x++ {
				if (abs(x-cx) == rad || abs(y-cy) == rad) && ok(w.At(x, y)) {
					return x, y, true
				}
			}
		}
	}
	return 0, 0, false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ---------- roads (A*) ----------

type pqItem struct {
	idx  int
	cost float64
}
type pq []pqItem

func (p pq) Len() int           { return len(p) }
func (p pq) Less(i, j int) bool { return p[i].cost < p[j].cost }
func (p pq) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p *pq) Push(x any)        { *p = append(*p, x.(pqItem)) }
func (p *pq) Pop() any          { o := *p; it := o[len(o)-1]; *p = o[:len(o)-1]; return it }

func tileCost(t byte) float64 {
	switch t {
	case TDirt, TPaved, TBridge, TPath:
		return 0.4
	case TGrass, TField, TPlaza, TGravel, TSidewalk, TCursed:
		return 1
	case TFlowers, TSand:
		return 1.4
	case THill:
		return 3
	case TForest:
		return 3.5
	case TRock:
		return 9
	case TWater:
		return 14
	case TDeep:
		return 45
	case TPeak:
		return 60
	}
	return 1e9 // buildings, walls
}

func (w *World) road(x0, y0, x1, y1 int, tile byte, width int) {
	n := w.W * w.H
	dist := make([]float64, n)
	prev := make([]int32, n)
	for i := range dist {
		dist[i] = math.Inf(1)
		prev[i] = -1
	}
	start, goal := y0*w.W+x0, y1*w.W+x1
	dist[start] = 0
	h := &pq{{start, 0}}
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for h.Len() > 0 {
		it := heap.Pop(h).(pqItem)
		if it.idx == goal {
			break
		}
		cx, cy := it.idx%w.W, it.idx/w.W
		base := dist[it.idx]
		if it.cost-heuristic(cx, cy, x1, y1) > base+1e-9 {
			continue
		}
		for _, d := range dirs {
			nx, ny := cx+d[0], cy+d[1]
			if nx < 1 || ny < 1 || nx >= w.W-1 || ny >= w.H-1 {
				continue
			}
			ni := ny*w.W + nx
			c := tileCost(w.Tiles[ni])
			if ni == goal {
				c = 1
			}
			if c >= 1e9 {
				continue
			}
			nd := base + c
			if nd < dist[ni] {
				dist[ni] = nd
				prev[ni] = int32(it.idx)
				heap.Push(h, pqItem{ni, nd + heuristic(nx, ny, x1, y1)})
			}
		}
	}
	if prev[goal] < 0 && goal != start {
		return
	}
	for i := goal; i >= 0; i = int(prev[i]) {
		x, y := i%w.W, i/w.W
		for oy := 0; oy < width; oy++ {
			for ox := 0; ox < width; ox++ {
				t := w.At(x+ox, y+oy)
				switch t {
				case TBuilding, TWall, TPlaza:
				case TWater, TDeep:
					w.Set(x+ox, y+oy, TBridge)
				default:
					if t != TPaved || tile == TPaved {
						w.Set(x+ox, y+oy, tile)
					}
				}
			}
		}
		if i == start {
			break
		}
	}
}

func heuristic(x, y, gx, gy int) float64 { return 0.4 * float64(abs(x-gx)+abs(y-gy)) }

// ---------- settlements ----------

func (w *World) addBuilding(b Building) int {
	w.Buildings = append(w.Buildings, b)
	w.fill(b.X, b.Y, b.W, b.H, TBuilding)
	return b.ID + 1
}

func (w *World) layoutHigh(t Town, bid int) int {
	// Gravel grounds ringed by a wall, a torii gate on the south side.
	w.clear(t.X, t.Y, t.R+2, TForest)
	w.clear(t.X, t.Y, t.R, TGravel)
	for y := t.Y - t.R; y <= t.Y+t.R; y++ {
		for x := t.X - t.R; x <= t.X+t.R; x++ {
			d := math.Hypot(float64(x-t.X), float64(y-t.Y))
			if d > float64(t.R)-0.6 && d <= float64(t.R)+0.5 {
				w.Set(x, y, TWall)
			}
		}
	}
	for x := t.X - 1; x <= t.X+1; x++ {
		for y := t.Y + t.R - 1; y <= t.Y+t.R+4; y++ {
			w.Set(x, y, TPath)
		}
	}
	for x := t.X - 4; x <= t.X+4; x++ {
		for y := t.Y + t.R + 2; y <= t.Y+t.R+5; y++ {
			if w.At(x, y) != TPath {
				w.Set(x, y, TGravel)
			}
		}
	}
	w.fill(t.X-1, t.Y-2, 3, 10, TPlaza)
	bid = w.addBuilding(Building{ID: bid, Kind: "temple", X: t.X - 4, Y: t.Y - 7, W: 9, H: 4, Town: t.ID, Name: "Main Hall"})
	bid = w.addBuilding(Building{ID: bid, Kind: "dorm", X: t.X - 8, Y: t.Y - 1, W: 5, H: 3, Town: t.ID, Name: "Dormitory"})
	bid = w.addBuilding(Building{ID: bid, Kind: "infirmary", X: t.X + 4, Y: t.Y - 1, W: 5, H: 3, Town: t.ID, Name: "Infirmary"})
	bid = w.addBuilding(Building{ID: bid, Kind: "gate", X: t.X - 1, Y: t.Y + t.R + 1, W: 3, H: 1, Town: t.ID, Name: "Torii"})
	// The torii is decorative and walk-through.
	w.fill(t.X-1, t.Y+t.R+1, 3, 1, TPath)
	return bid
}

func (w *World) layoutTown(r *rand.Rand, t Town, bid int) int {
	w.clear(t.X, t.Y, t.R+1, TGrass)
	w.fill(t.X-2, t.Y-2, 5, 5, TPlaza)
	for i := -t.R; i <= t.R; i++ {
		w.Set(t.X+i, t.Y, TDirt)
		w.Set(t.X+i, t.Y+1, TDirt)
		w.Set(t.X, t.Y+i, TDirt)
		w.Set(t.X+1, t.Y+i, TDirt)
	}
	w.fill(t.X-2, t.Y-2, 5, 5, TPlaza)
	kinds := []string{"shop", "inn", "elder", "shrine_small"}
	n := 0
	free := func(b byte) bool { return b == TGrass || b == TFlowers || b == TField }
	for tries := 0; tries < 300 && n < 9; tries++ {
		bw, bh := 3+r.IntN(3), 3+r.IntN(2)
		qx := []int{-1, 1}[r.IntN(2)]
		qy := []int{-1, 1}[r.IntN(2)]
		x := t.X + qx*(2+r.IntN(t.R-3))
		y := t.Y + qy*(2+r.IntN(t.R-3))
		if qx < 0 {
			x -= bw
		} else {
			x += 2
		}
		if qy < 0 {
			y -= bh + 1
		} else {
			y += 2
		}
		if !w.rectFree(x-1, y-1, bw+2, bh+2, free) {
			continue
		}
		if math.Hypot(float64(x+bw/2-t.X), float64(y+bh/2-t.Y)) > float64(t.R) {
			continue
		}
		kind := "house"
		if n < len(kinds) {
			kind = kinds[n]
		}
		name := map[string]string{"shop": "General Store", "inn": "Inn", "elder": "Mayor's House", "shrine_small": "Shrine", "house": "House"}[kind]
		bid = w.addBuilding(Building{ID: bid, Kind: kind, X: x, Y: y, W: bw, H: bh, Town: t.ID, Name: name, Color: r.IntN(6)})
		// Front path from the door down to the street.
		dx := x + bw/2
		for yy := y + bh; yy <= y+bh+1; yy++ {
			if free(w.At(dx, yy)) {
				w.Set(dx, yy, TPath)
			}
		}
		n++
	}
	// Some fields around villages.
	for i := 0; i < 3; i++ {
		fx := t.X + r.IntN(2*t.R+6) - t.R - 3
		fy := t.Y + r.IntN(2*t.R+6) - t.R - 3
		if math.Hypot(float64(fx-t.X), float64(fy-t.Y)) > float64(t.R)-1 {
			for yy := fy; yy < fy+4; yy++ {
				for xx := fx; xx < fx+5; xx++ {
					if w.At(xx, yy) == TGrass {
						w.Set(xx, yy, TField)
					}
				}
			}
		}
	}
	return bid
}

func (w *World) layoutCity(r *rand.Rand, t Town, bid int) int {
	x0, y0 := t.X-t.R, t.Y-t.R
	size := 2 * t.R
	for y := y0 - 1; y <= y0+size+1; y++ {
		for x := x0 - 1; x <= x0+size+1; x++ {
			w.Set(x, y, TSidewalk)
		}
	}
	const block = 9
	for y := y0; y <= y0+size; y++ {
		for x := x0; x <= x0+size; x++ {
			if (x-x0)%block < 2 || (y-y0)%block < 2 {
				w.Set(x, y, TPaved)
			}
		}
	}
	kinds := []string{"office", "apartment", "store", "office", "tower", "apartment"}
	stationDone := false
	for by := y0 + 2; by+block-2 <= y0+size; by += block {
		for bx := x0 + 2; bx+block-2 <= x0+size; bx += block {
			inner := block - 2
			if !stationDone && math.Abs(float64(bx+inner/2-t.X)) < block && math.Abs(float64(by+inner/2-t.Y)) < block {
				stationDone = true
				bid = w.addBuilding(Building{ID: bid, Kind: "station", X: bx + 1, Y: by + 1, W: inner - 2, H: inner - 3, Town: t.ID, Name: "Shibuya Station"})
				continue
			}
			// Two buildings per block, with a sidewalk gap.
			if r.IntN(5) == 0 {
				w.fill(bx+1, by+1, inner-2, inner-2, TPlaza)
				continue
			}
			bw := inner/2 - 1
			bid = w.addBuilding(Building{ID: bid, Kind: kinds[r.IntN(len(kinds))], X: bx + 1, Y: by + 1, W: bw, H: inner - 3, Town: t.ID, Color: r.IntN(6)})
			bid = w.addBuilding(Building{ID: bid, Kind: kinds[r.IntN(len(kinds))], X: bx + 2 + bw, Y: by + 1, W: inner - 3 - bw, H: inner - 3, Town: t.ID, Color: r.IntN(6)})
		}
	}
	return bid
}

// ---------- NPCs ----------

var skinTones = 6

func randomLook(r *rand.Rand) Appearance {
	hair := []string{"#1b1b1b", "#3b2a1a", "#6b4423", "#a0522d", "#d4a373", "#f1e3c4", "#8d99ae", "#b5838d"}
	cols := []string{"#22223b", "#4a4e69", "#9a8c98", "#c9ada7", "#6b705c", "#a5a58d", "#3d405b", "#e07a5f", "#81b29a", "#f2cc8f", "#457b9d", "#e63946"}
	eyes := []string{"#3b2a1a", "#1b1b1b", "#4a6fa5", "#6a994e"}
	return Appearance{Skin: r.IntN(skinTones), Hair: r.IntN(10), HairColor: hair[r.IntN(len(hair))], Eyes: eyes[r.IntN(len(eyes))],
		Outfit: 3 + r.IntN(5), Color1: cols[r.IntN(len(cols))], Color2: cols[r.IntN(len(cols))], Accessory: 0, Build: r.IntN(3)}
}

var firstNames = []string{"Haruto", "Yui", "Sota", "Aoi", "Ren", "Hina", "Daiki", "Mio", "Kaito", "Saki", "Takumi", "Emi", "Riku", "Nanami", "Kenta", "Yuna", "Shun", "Akari", "Itsuki", "Mei", "Goro", "Fumiko", "Tetsu", "Chiyo"}

func (w *World) placeNPCs(r *rand.Rand) {
	hi := w.Towns[0]
	canon := []NPC{
		{ID: "gojo", Name: "Satoru Gojo", Role: "gojo", X: float64(hi.X) + 0.5, Y: float64(hi.Y) - 2, Look: Appearance{Skin: 0, Hair: 6, HairColor: "#f8f9fa", Eyes: "#7ec8ff", Outfit: 0, Color1: "#14141c", Color2: "#2a2a3a", Accessory: 1, Build: 2},
			Desc: "The strongest sorcerer. A blindfold over eyes that see everything, and an attitude to match."},
		{ID: "yaga", Name: "Principal Yaga", Role: "yaga", X: float64(hi.X) - 3, Y: float64(hi.Y) - 2, Look: Appearance{Skin: 3, Hair: 9, HairColor: "#1b1b1b", Eyes: "#1b1b1b", Outfit: 5, Color1: "#2b2d42", Color2: "#8d99ae", Accessory: 3, Build: 2},
			Desc: "Principal of Tokyo Jujutsu High. Gruff, sews cursed dolls, cares more than he lets on."},
		{ID: "shoko", Name: "Shoko Ieiri", Role: "shoko", X: float64(hi.X) + 6.5, Y: float64(hi.Y) + 2.5, Look: Appearance{Skin: 1, Hair: 3, HairColor: "#5c4033", Eyes: "#3b2a1a", Outfit: 6, Color1: "#f8f9fa", Color2: "#212529", Accessory: 0, Build: 0},
			Desc: "The school doctor. One of the only people who can heal others with Reverse Cursed Technique."},
		{ID: "ijichi", Name: "Kiyotaka Ijichi", Role: "ijichi", X: float64(hi.X) + 2, Y: float64(hi.Y) + float64(hi.R) + 3, Look: Appearance{Skin: 1, Hair: 2, HairColor: "#1b1b1b", Eyes: "#1b1b1b", Outfit: 5, Color1: "#212529", Color2: "#adb5bd", Accessory: 2, Build: 0},
			Desc: "Assistant supervisor and driver. Nervous, overworked, utterly reliable. Can drive you to any town you've visited."},
		{ID: "nanami", Name: "Kento Nanami", Role: "nanami", X: float64(hi.X) - 6, Y: float64(hi.Y) + 3, Look: Appearance{Skin: 1, Hair: 4, HairColor: "#f1e3c4", Eyes: "#6a994e", Outfit: 5, Color1: "#d4c39b", Color2: "#e63946", Accessory: 4, Build: 2},
			Desc: "A Grade 1 sorcerer and former salaryman. Hates overtime. Gives side work to students."},
		{ID: "maki", Name: "Maki Zenin", Role: "maki", X: float64(hi.X) - 2, Y: float64(hi.Y) + 5, Look: Appearance{Skin: 1, Hair: 7, HairColor: "#2d6a4f", Eyes: "#ffd166", Outfit: 0, Color1: "#14141c", Color2: "#2a2a3a", Accessory: 2, Build: 1},
			Desc: "A second-year who sees curses only through her glasses — and beats them anyway. Sells spare cursed tools."},
	}
	w.NPCs = append(w.NPCs, canon...)
	for _, t := range w.Towns[1:] {
		place := func(id, role, name string, x, y float64, desc string) {
			n := NPC{ID: id, Name: name, Role: role, X: x, Y: y, Town: t.ID, Look: randomLook(r), Desc: desc}
			w.NPCs = append(w.NPCs, n)
		}
		for _, b := range w.Buildings {
			if b.Town != t.ID {
				continue
			}
			dx, dy := float64(b.X+b.W/2)+0.5, float64(b.Y+b.H)+0.6
			fn := firstNames[r.IntN(len(firstNames))]
			switch b.Kind {
			case "shop":
				place(fmt.Sprintf("shop_%s", t.ID), "shop", fn+" the Shopkeeper", dx, dy, "Runs the "+t.Name+" general store. Knows every rumor.")
			case "inn":
				place(fmt.Sprintf("inn_%s", t.ID), "inn", fn+" the Innkeeper", dx, dy, "Keeps the inn of "+t.Name+". Rooms are cheap, the bath is hot.")
			case "elder":
				place(fmt.Sprintf("elder_%s", t.ID), "elder", "Mayor "+fn, dx, dy, "The worried mayor of "+t.Name+". Requests help with curse incidents.")
			case "shrine_small":
				place(fmt.Sprintf("monk_%s", t.ID), "monk", "Priest "+fn, dx, dy, "Tends the "+t.Name+" shrine. Senses curses, can't fight them.")
			case "station":
				place("station_master", "shop", "Station Vendor "+fn, dx, dy, "Sells snacks, tonics and gossip by the ticket gates.")
			}
		}
		count := 3
		if t.Kind == "city" {
			count = 9
			place("meimei", "mercenary", "Mei Mei", float64(t.X)+3.5, float64(t.Y)+float64(t.R)/2, "A Grade 1 sorcerer who does anything for money. Sells rare cursed tools — at a price.")
		}
		for i := 0; i < count; i++ {
			for tries := 0; tries < 40; tries++ {
				x := float64(t.X+r.IntN(2*t.R)-t.R) + 0.5
				y := float64(t.Y+r.IntN(2*t.R)-t.R) + 0.5
				if w.Walkable(x, y) {
					role := "villager"
					if i == 0 && t.Kind != "city" {
						role = "guard"
					}
					name := firstNames[r.IntN(len(firstNames))]
					desc := pickR(r, []string{"A local who %s.", "A tired resident who %s.", "A curious passer-by who %s."},
						pickR(r, []string{"works at the %s", "is worried about the %s", "keeps hearing noises from the %s"}, pickS(r, []string{"mill", "station", "school", "river", "old shrine", "konbini"})))
					if role == "guard" {
						name = "Officer " + name
						desc = "A local police officer. Can't see curses, but knows something is wrong."
					}
					place(fmt.Sprintf("v_%s_%d", t.ID, i), role, name, x, y, desc)
					break
				}
			}
		}
	}
}
