package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// More everyday-horror curses: the bestiary grows beyond the classics.
func init() {
	curse(CurseTemplate{ID: "vending_wraith", Name: "Vending Machine Spirit", Rank: 0, XP: 30, HP: "3d8", AC: 13, Init: 0, ToHit: 4, Tags: []string{"city", "school", "any"},
		Moves:  []CurseMove{hit("Can Barrage", "1d6+2", "bludgeoning", 3, "orb", "#ef476f"), hitS("Scalding Coffee", "1d6", "fire", 1, "spit", "#fb8500", "burn", 1)},
		Sprite: SpriteSpec{Shape: "giant", Color: "#ef476f", Color2: "#ffd166", Eyes: 2, Mouth: "grin", Size: 1, Extra: "windows"}, Yen: "1d6*150", Desc: "Fear of the lonely 3 a.m. light. It only takes exact change."})
	curse(CurseTemplate{ID: "karakasa", Name: "Umbrella Yokai", Rank: 0, XP: 25, HP: "2d8", AC: 12, Init: 4, ToHit: 3, Tags: []string{"forest", "field", "any", "shrine"},
		Moves:  []CurseMove{hit("Spin Slash", "1d6+2", "slashing", 3, "slash", "#8ecae6"), aoe("Rain Gust", "1d6", "cold", "DEX", 1, "wail", "#8ecae6", "", 0)},
		Sprite: SpriteSpec{Shape: "head", Color: "#5390d9", Color2: "#ffd166", Eyes: 1, Mouth: "grin", Size: 1, Extra: "wings"}, Yen: "1d6*100", Desc: "A forgotten umbrella with one eye and a hopping, bad attitude."})
	curse(CurseTemplate{ID: "lantern_child", Name: "Lantern Child", Rank: 0, XP: 30, HP: "2d8", AC: 11, Init: 2, ToHit: 3, Tags: []string{"grove", "shrine", "grave", "any"},
		Moves:  []CurseMove{hitS("Will-o-Wisp", "1d6+1", "fire", 3, "embers", "#ffb703", "burn", 1), hitS("Lead Astray", "1d4", "psychic", 1, "stare", "#caf0f8", "weakened", 1)},
		Sprite: SpriteSpec{Shape: "blob", Color: "#ffb703", Color2: "#023047", Eyes: 2, Mouth: "none", Size: 1, Extra: "flames"}, Yen: "1d6*100", Desc: "A paper lantern that leads travelers off the path."})
	curse(CurseTemplate{ID: "cafeteria_glutton", Name: "Cafeteria Glutton", Rank: 1, XP: 110, HP: "6d10", AC: 12, Init: -1, ToHit: 5, Save: 2, Tags: []string{"school", "hospital", "any"},
		Moves:  []CurseMove{hit("Devour", "2d8+3", "piercing", 3, "bite", "#e07a5f"), aoe("Tray Storm", "2d6", "bludgeoning", "DEX", 2, "rain", "#f2cc8f", "", 0)},
		Sprite: SpriteSpec{Shape: "blob", Color: "#e07a5f", Color2: "#f2cc8f", Eyes: 3, Mouth: "maw", Size: 2, Extra: "arms"}, Yen: "2d6*500", Desc: "Born from a thousand lunch-line resentments. It is always still hungry."})
	curse(CurseTemplate{ID: "origami_swarm", Name: "Paper Crane Swarm", Rank: 1, XP: 100, HP: "4d8", AC: 14, Init: 5, ToHit: 5, Tags: []string{"school", "shrine", "any"},
		Moves:  []CurseMove{aoe("Razor Flock", "2d6+1", "slashing", "DEX", 3, "blades", "#f8f9fa", "bleed", 1), hit("Beak Dive", "2d6+2", "piercing", 2, "bite", "#e5e5e5")},
		Sprite: SpriteSpec{Shape: "swarm", Color: "#f8f9fa", Color2: "#e63946", Eyes: 0, Mouth: "none", Size: 2, Extra: "wings"}, Yen: "2d6*400", Desc: "A thousand prayers folded into knives."})
	curse(CurseTemplate{ID: "bath_scalder", Name: "Bathhouse Scalder", Rank: 1, XP: 120, HP: "6d8", AC: 13, Init: 1, ToHit: 5, Tags: []string{"water", "city", "lake"},
		Moves:  []CurseMove{aoe("Boiling Steam", "3d6", "fire", "CON", 3, "meteor", "#ffb4a2", "burn", 1), hit("Scrub Slam", "2d6+3", "bludgeoning", 2, "slam", "#ffcdb2")},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#ffb4a2", Color2: "#6d6875", Eyes: 2, Mouth: "grin", Size: 2, Extra: "hood"}, Yen: "2d6*500", Desc: "The thing that makes the water just slightly too hot."})
	curse(CurseTemplate{ID: "alley_cat_tyrant", Name: "Alley Cat Tyrant", Rank: 1, XP: 100, HP: "4d8", AC: 15, Init: 8, ToHit: 6, Tags: []string{"city", "any", "forest"},
		Moves:  []CurseMove{hit("Claw Flurry", "2d6+3", "slashing", 3, "claw", "#adb5bd"), hitS("Hiss", "1d6", "psychic", 1, "stare", "#ffd166", "weakened", 1)},
		Sprite: SpriteSpec{Shape: "beast", Color: "#6c757d", Color2: "#ffd166", Eyes: 2, Mouth: "maw", Size: 1, Extra: "crown"}, Yen: "2d6*400", Desc: "The self-appointed king of every back alley. Smug. Fast."})
	curse(CurseTemplate{ID: "static_choir", Name: "Static Choir", Rank: 1, XP: 120, HP: "5d8", AC: 12, Init: 2, ToHit: 5, Tags: []string{"factory", "city", "tunnel", "hospital"},
		Moves:  []CurseMove{aoe("White Noise", "2d8", "psychic", "WIS", 3, "wail", "#ced4da", "weakened", 1), hit("Feedback Screech", "2d6+3", "psychic", 2, "stare", "#ced4da")},
		Sprite: SpriteSpec{Shape: "head", Color: "#adb5bd", Color2: "#212529", Eyes: 4, Mouth: "maw", Size: 2, Extra: "windows"}, Yen: "2d6*500", Desc: "A radio left on in an empty room, singing along to nothing."})
	curse(CurseTemplate{ID: "mirror_double", Name: "Mirror Double", Rank: 2, XP: 320, HP: "8d8", AC: 15, Init: 3, ToHit: 6, Save: 3, Tags: []string{"any", "school", "hospital", "city"},
		Moves:  []CurseMove{hit("Reflected Strike", "2d10+4", "psychic", 3, "stare", "#caf0f8"), aoe("Shatter", "3d6", "slashing", "DEX", 2, "blades", "#caf0f8", "bleed", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#caf0f8", Color2: "#023e8a", Eyes: 2, Mouth: "none", Size: 2, Extra: "halo"}, Yen: "4d6*1000", Desc: "It wears your posture and your worst expression."})
	curse(CurseTemplate{ID: "shadow_commuter", Name: "Shadow Commuter", Rank: 2, XP: 300, HP: "7d8", AC: 14, Init: 4, ToHit: 6, Save: 3, Tags: []string{"tunnel", "city", "any"},
		Moves:  []CurseMove{hitS("Crowd Push", "2d8+3", "bludgeoning", 3, "rush", "#343a40", "exposed", 1), hitS("Dead Stare", "2d6", "necrotic", 1, "stare", "#6c757d", "weakened", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#343a40", Color2: "#adb5bd", Eyes: 2, Mouth: "none", Size: 2, Extra: "hood"}, Yen: "3d6*1000", Desc: "The nobody in the packed train car who never gets off."})
	curse(CurseTemplate{ID: "festival_mask", Name: "Festival Mask Dancer", Rank: 2, XP: 330, HP: "8d8", AC: 14, Init: 5, ToHit: 6, Save: 3, Tags: []string{"shrine", "grove", "any", "city"},
		Moves:  []CurseMove{hitS("Drum Strike", "2d8+3", "bludgeoning", 3, "punch", "#e63946", "exposed", 1), aoe("Mad Dance", "3d6", "psychic", "WIS", 2, "ring", "#ffd166", "stun", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#e63946", Color2: "#ffd166", Eyes: 2, Mouth: "grin", Size: 2, Extra: "mask"}, Yen: "4d6*1000", Desc: "A festival that ended a century ago and never told the dancers."})
	curse(CurseTemplate{ID: "well_woman", Name: "Well Dweller", Rank: 3, XP: 950, HP: "12d10", AC: 15, Init: 2, ToHit: 8, Save: 4, Tags: []string{"grave", "shrine", "lake", "grove"},
		Moves:  []CurseMove{hitS("Hair Drag", "3d8+5", "necrotic", 3, "hands", "#adb5bd", "stun", 1), aoe("Rising Flood", "4d8", "cold", "STR", 1, "flood", "#90e0ef", "weakened", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#e9ecef", Color2: "#212529", Eyes: 2, Mouth: "grin", Size: 3, Extra: "veil"}, Yen: "6d6*3000", Desc: "She was lowered into the well. She remembers who did it."})
}

// ---------- curses invented in play ----------

type curseFlavor struct {
	words  *regexp.Regexp
	sprite SpriteSpec
	moves  []func(r int) CurseMove
}

func rd(r int, table []string) string { return table[min(r, len(table)-1)] }

var (
	meleeDice = []string{"1d6+1", "2d6+3", "2d10+4", "3d8+5", "3d10+5", "4d10+5"}
	lightDice = []string{"1d4", "1d8", "2d8", "3d8", "3d10", "4d8"}
	aoeDice   = []string{"1d6", "2d6", "3d6", "4d6", "4d8", "5d8"}
)

var curseFlavors = map[string]curseFlavor{
	"sick": {regexp.MustCompile(`sick|disease|mold|plague|fever|infect|germ|ward|virus|cough|rot`), SpriteSpec{Shape: "blob", Color: "#84a98c", Color2: "#d4a373", Eyes: 3, Mouth: "maw", Extra: "tentacles"}, []func(int) CurseMove{
		func(r int) CurseMove {
			return hitS("Fever Touch", rd(r, lightDice), "poison", 3, "hands", "#84a98c", "poison", 2)
		},
		func(r int) CurseMove {
			return aoe("Spore Cloud", rd(r, aoeDice), "poison", "CON", 1, "spit", "#a7c957", "weakened", 1)
		}}},
	"hunger": {regexp.MustCompile(`hunger|glutton|food|cafeteria|eat|ramen|starv|feast|kitchen|bake`), SpriteSpec{Shape: "beast", Color: "#e07a5f", Color2: "#f2cc8f", Eyes: 2, Mouth: "maw", Extra: "ribs"}, []func(int) CurseMove{
		func(r int) CurseMove { return hit("Gnash", rd(r, meleeDice), "piercing", 3, "bite", "#e07a5f") },
		func(r int) CurseMove {
			return hitS("Swallow Whole", rd(r, lightDice), "bludgeoning", 1, "bite", "#9d0208", "exposed", 1)
		}}},
	"fear": {regexp.MustCompile(`fear|ghost|haunt|scream|nightmare|dread|terror|whisper|creep|spirit`), SpriteSpec{Shape: "humanoid", Color: "#f8f9fa", Color2: "#7209b7", Eyes: 2, Mouth: "grin", Extra: "veil"}, []func(int) CurseMove{
		func(r int) CurseMove { return hit("Cold Grasp", rd(r, meleeDice), "cold", 3, "hands", "#caf0f8") },
		func(r int) CurseMove {
			return hitS("Dread Stare", rd(r, lightDice), "psychic", 2, "stare", "#7209b7", "weakened", 1)
		}}},
	"rage": {regexp.MustCompile(`rage|anger|fight|riot|spite|bully|brawl|grudge|rival|hate`), SpriteSpec{Shape: "giant", Color: "#d00000", Color2: "#ffba08", Eyes: 2, Mouth: "maw", Extra: "horns"}, []func(int) CurseMove{
		func(r int) CurseMove { return hit("Rampage", rd(r, meleeDice), "bludgeoning", 3, "slam", "#d00000") },
		func(r int) CurseMove {
			return hitS("Bellow", rd(r, lightDice), "psychic", 1, "wail", "#ffba08", "exposed", 1)
		}}},
	"decay": {regexp.MustCompile(`decay|grave|dead|corpse|bone|mildew|bury|tomb|funeral|skelet`), SpriteSpec{Shape: "humanoid", Color: "#adb5bd", Color2: "#495057", Eyes: 2, Mouth: "stitched", Extra: "coffin"}, []func(int) CurseMove{
		func(r int) CurseMove { return hit("Grave Hand", rd(r, meleeDice), "necrotic", 3, "hands", "#adb5bd") },
		func(r int) CurseMove {
			return aoe("Tolling Bell", rd(r, aoeDice), "necrotic", "CON", 1, "wail", "#6c757d", "weakened", 1)
		}}},
	"noise": {regexp.MustCompile(`noise|sound|radio|loud|bell|song|static|echo|speaker|music|siren`), SpriteSpec{Shape: "head", Color: "#ced4da", Color2: "#212529", Eyes: 4, Mouth: "maw", Extra: "windows"}, []func(int) CurseMove{
		func(r int) CurseMove {
			return aoe("Screech", rd(r, aoeDice), "psychic", "WIS", 2, "wail", "#ced4da", "weakened", 1)
		},
		func(r int) CurseMove { return hit("Feedback", rd(r, meleeDice), "psychic", 2, "stare", "#ced4da") }}},
	"shadow": {regexp.MustCompile(`shadow|alley|silhouette|dusk|night|dark|commut|crowd`), SpriteSpec{Shape: "humanoid", Color: "#343a40", Color2: "#adb5bd", Eyes: 2, Mouth: "none", Extra: "hood"}, []func(int) CurseMove{
		func(r int) CurseMove {
			return hitS("Shadow Shove", rd(r, meleeDice), "bludgeoning", 3, "rush", "#343a40", "exposed", 1)
		},
		func(r int) CurseMove {
			return hitS("Smother", rd(r, lightDice), "necrotic", 1, "stare", "#6c757d", "weakened", 1)
		}}},
	"fire": {regexp.MustCompile(`fire|burn|smoke|ember|kiln|stove|forge|furnace|cinder|boil|steam`), SpriteSpec{Shape: "blob", Color: "#ff6d00", Color2: "#ffd166", Eyes: 2, Mouth: "grin", Extra: "flames"}, []func(int) CurseMove{
		func(r int) CurseMove {
			return hitS("Scorch", rd(r, lightDice), "fire", 3, "fire_punch", "#ff6d00", "burn", 1)
		},
		func(r int) CurseMove {
			return aoe("Flare", rd(r, aoeDice), "fire", "DEX", 1, "embers", "#ffd166", "burn", 1)
		}}},
	"water": {regexp.MustCompile(`water|drown|flood|rain|lake|river|pool|sea|bath|well|tide|wet|fish`), SpriteSpec{Shape: "serpent", Color: "#48cae4", Color2: "#023e8a", Eyes: 2, Mouth: "maw", Extra: "tentacles"}, []func(int) CurseMove{
		func(r int) CurseMove {
			return hit("Drag Under", rd(r, meleeDice), "bludgeoning", 3, "hands", "#48cae4")
		},
		func(r int) CurseMove {
			return aoe("Surge", rd(r, aoeDice), "cold", "STR", 1, "flood", "#90e0ef", "weakened", 1)
		}}},
}

func pickFlavor(text string) string {
	t := strings.ToLower(text)
	for _, k := range []string{"sick", "hunger", "noise", "water", "fire", "decay", "rage", "shadow", "fear"} {
		if curseFlavors[k].words.MatchString(t) {
			return k
		}
	}
	return "fear"
}

var rankStats = []struct {
	hp      string
	ac, hit int
	xp      int
	yen     string
}{
	{"3d8", 12, 4, 30, "1d6*150"}, {"5d8", 13, 5, 100, "2d6*500"}, {"8d8", 14, 6, 300, "4d6*1000"},
	{"13d10", 15, 8, 950, "6d6*3000"}, {"16d10", 16, 8, 1800, "10d6*3000"},
}

// NewCustomCurse builds a bounded-stat curse from a name, a theme and a grade (0 = Grade 4 .. 4 = Semi-Special).
func (e *Engine) NewCustomCurse(name, desc, flavor string, rank int, boss bool) (CurseTemplate, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 50 {
		return CurseTemplate{}, fmt.Errorf("give the curse a short name")
	}
	rank = min(4, max(0, rank))
	if _, ok := curseFlavors[flavor]; !ok {
		flavor = pickFlavor(name + " " + desc)
	}
	fl, rs := curseFlavors[flavor], rankStats[rank]
	id := "x_" + strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(name), "_"), "_")
	if t, ok := Bestiary[id]; ok && t.Name != name {
		id += "_" + e.nextID("c")
	}
	sp := fl.sprite
	sp.Size = 1 + rank/2
	if boss {
		sp.Size = min(4, sp.Size+1)
	}
	t := CurseTemplate{ID: id, Name: name, Rank: rank, XP: rs.xp, HP: rs.hp, AC: rs.ac, ToHit: rs.hit, Save: rank, Init: 2,
		Tags: []string{"any", flavor}, Sprite: sp, Yen: rs.yen, Boss: boss && rank >= 2, Desc: strings.TrimSpace(desc)}
	if t.Desc == "" {
		t.Desc = "A curse born from " + flavor + ", given a name by the people who suffer it."
	}
	for _, mk := range fl.moves {
		t.Moves = append(t.Moves, mk(rank))
	}
	t.Grade = GradeNames[rank]
	if e.C.CustomCurses == nil {
		e.C.CustomCurses = map[string]CurseTemplate{}
	}
	e.C.CustomCurses[id] = t
	Bestiary[id] = t
	return t, nil
}

func (e *Engine) registerCustomCurses() {
	for id, t := range e.C.CustomCurses {
		Bestiary[id] = t
	}
}
