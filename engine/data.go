package engine

import "sort"

// Abilities in display order. Internally D&D codes, shown with jujutsu names in the UI.
var Abilities = []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}

var AbilityNames = map[string]string{
	"STR": "Strength", "DEX": "Agility", "CON": "Toughness",
	"INT": "Technique", "WIS": "Perception", "CHA": "Will",
}

// Skills maps each skill to the ability it uses.
var Skills = map[string]string{
	"athletics": "STR", "acrobatics": "DEX", "sleight_of_hand": "DEX", "stealth": "DEX",
	"jujutsu_lore": "INT", "history": "INT", "investigation": "INT", "barriers": "INT", "medicine": "INT",
	"curse_sense": "WIS", "insight": "WIS", "perception": "WIS", "survival": "WIS",
	"deception": "CHA", "intimidation": "CHA", "performance": "CHA", "persuasion": "CHA",
}

// Origin is a sorcerer's family background (the "race" slot).
type Origin struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Bonus  map[string]int `json:"bonus"`
	Traits string         `json:"traits"`
}

var Origins = map[string]Origin{
	"non_clan":   {"non_clan", "Non-Clan Sorcerer", map[string]int{"STR": 1, "DEX": 1, "CON": 1, "INT": 1, "WIS": 1, "CHA": 1}, "Self-made. +1 to every ability and one extra skill."},
	"gojo":       {"gojo", "Gojo Clan", map[string]int{"INT": 2, "WIS": 1}, "Refined cursed energy: every technique costs 1 less CE (minimum 1)."},
	"zenin":      {"zenin", "Zenin Clan", map[string]int{"STR": 2, "DEX": 1}, "Raised on cursed tools: +1 to hit and +2 damage with weapon strikes."},
	"kamo":       {"kamo", "Kamo Clan", map[string]int{"INT": 2, "CON": 1}, "Deep reserves: +5 maximum cursed energy."},
	"inumaki":    {"inumaki", "Inumaki Clan", map[string]int{"CHA": 2, "WIS": 1}, "Words with weight: your technique save DCs are 1 higher."},
	"half_curse": {"half_curse", "Half-Curse", map[string]int{"CON": 2, "CHA": 1}, "Born of a curse: regenerate 2 HP each turn in battle, take 25% less damage from techniques."},
}

// Effect is one piece of what an ability does.
type Effect struct {
	Kind    string `json:"kind"`            // attack, save, auto, heal, status, cleanse, copy, gamble
	Target  string `json:"target"`          // enemy, all_enemies, ally, self, all_allies
	Dice    string `json:"dice,omitempty"`  // base dice
	Scale   string `json:"scale,omitempty"` // extra dice added at levels 5, 9 and 12
	DmgType string `json:"dmgType,omitempty"`
	Save    string `json:"save,omitempty"`
	Half    bool   `json:"half,omitempty"`
	Hits    int    `json:"hits,omitempty"`
	Status  string `json:"status,omitempty"`
	Rounds  int    `json:"rounds,omitempty"`
	PctMax  int    `json:"pctMax,omitempty"` // extra damage as % of the target's max HP
	AddMod  bool   `json:"addMod,omitempty"`
	Melee   bool   `json:"melee,omitempty"` // can Black Flash
}

// Ability is a cursed technique move, a universal jujutsu skill, or a domain.
type Ability struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Desc      string   `json:"desc"`
	Tech      string   `json:"tech"`
	Cost      int      `json:"cost"`
	HPCost    int      `json:"hpCost,omitempty"`
	Bonus     bool     `json:"bonus,omitempty"`
	MinLevel  int      `json:"minLevel"`
	Domain    bool     `json:"domain,omitempty"`
	Rounds    int      `json:"rounds,omitempty"`
	PerBattle int      `json:"perBattle,omitempty"`
	Effects   []Effect `json:"effects"`
	VFX       string   `json:"vfx"`
	Color     string   `json:"color"`
	Universal bool     `json:"universal,omitempty"`
	Passive   bool     `json:"passive,omitempty"`
}

// Technique is the innate cursed technique (the "class" slot).
type Technique struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Blurb    string   `json:"blurb"`
	HitDie   int      `json:"hitDie"`
	Priority []string `json:"priority"`
	Skills   []string `json:"skills"`
	Weapon   string   `json:"weapon"`
	Armor    string   `json:"armor"`
	Moves    []string `json:"moves"`
	Domain   string   `json:"domain"`
	Color    string   `json:"color"`
	NoCE     bool     `json:"noCE,omitempty"`
	Inspired string   `json:"inspired"`
}

func e(kind, target, dice string) Effect { return Effect{Kind: kind, Target: target, Dice: dice} }

var AbilityList = map[string]Ability{}
var Techniques = map[string]Technique{}

func addAbility(a Ability) { AbilityList[a.ID] = a }

func init() {
	// ---------- universal jujutsu ----------
	addAbility(Ability{ID: "strike", Name: "Cursed Strike", Universal: true, Desc: "A blow reinforced with cursed energy. Natural 20 (19+ when In the Zone) is a BLACK FLASH: 2.5x damage, +3 CE.",
		Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "weapon", AddMod: true, Melee: true}}, VFX: "punch", Color: "#ffffff"})
	addAbility(Ability{ID: "guard", Name: "Guard", Universal: true, Desc: "Brace with cursed energy: take half damage and attacks against you have disadvantage until your next turn.",
		Effects: []Effect{{Kind: "status", Target: "self", Status: "guard", Rounds: 1}}, VFX: "shield", Color: "#9fc3ff"})
	addAbility(Ability{ID: "binding_vow", Name: "Binding Vow: Reveal", Universal: true, Bonus: true, PerBattle: 1, Desc: "Explain your technique to your enemy. The vow empowers you: +2 to hit and +25% damage for 3 rounds.",
		Effects: []Effect{{Kind: "status", Target: "self", Status: "empowered", Rounds: 3}}, VFX: "vow", Color: "#ffd166"})
	addAbility(Ability{ID: "simple_domain", Name: "Simple Domain", Universal: true, Bonus: true, MinLevel: 3, Cost: 2, Desc: "A tiny barrier that neutralizes a domain's sure-hit for 2 rounds.",
		Effects: []Effect{{Kind: "status", Target: "self", Status: "simple_domain", Rounds: 2}}, VFX: "simple", Color: "#bde0fe"})
	addAbility(Ability{ID: "rct", Name: "Reverse Cursed Technique", Universal: true, MinLevel: 6, Cost: 5, Desc: "Multiply negative energy by itself to heal: 3d8 + level HP.",
		Effects: []Effect{{Kind: "heal", Target: "self", Dice: "3d8", Scale: "1d8", AddMod: true}}, VFX: "rct", Color: "#c7f9cc"})

	t := func(tc Technique, moves []Ability, domain Ability) {
		for _, m := range moves {
			m.Tech = tc.ID
			addAbility(m)
			tc.Moves = append(tc.Moves, m.ID)
		}
		domain.Tech, domain.Domain = tc.ID, true
		if domain.MinLevel == 0 {
			domain.MinLevel = 6
		}
		if domain.Rounds == 0 {
			domain.Rounds = 3
		}
		if domain.Cost == 0 && !tc.NoCE {
			domain.Cost = 10
		}
		domain.PerBattle = 1
		addAbility(domain)
		tc.Domain = domain.ID
		Techniques[tc.ID] = tc
	}

	t(Technique{ID: "limitless", Name: "Limitless", Inspired: "Satoru Gojo", HitDie: 8, Priority: []string{"INT", "WIS", "CON", "DEX", "CHA", "STR"},
		Skills: []string{"jujutsu_lore", "curse_sense", "insight"}, Weapon: "fists", Armor: "uniform", Color: "#5aa9ff",
		Blurb: "Infinity made real. Bend space to pull, repel and erase. Fragile CE costs, overwhelming payoff."},
		[]Ability{
			{ID: "infinity", Name: "Infinity", Bonus: true, Cost: 2, Desc: "Slow everything approaching you to a crawl: attacks against you have disadvantage and deal half damage for 2 rounds.", Effects: []Effect{{Kind: "status", Target: "self", Status: "infinity", Rounds: 2}}, VFX: "infinity", Color: "#a8d8ff"},
			{ID: "lapse_blue", Name: "Lapse: Blue", Cost: 3, Desc: "A point of attraction crushes every enemy together: 2d8 force each (STR save half) and they are Exposed.", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "2d8", Scale: "1d8", DmgType: "force", Save: "STR", Half: true, Status: "exposed", Rounds: 1}}, VFX: "blue", Color: "#3a86ff"},
			{ID: "reversal_red", Name: "Reversal: Red", Cost: 4, Desc: "Repulsion fired as an orb: 4d10 force on a hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "4d10", Scale: "1d10", DmgType: "force", AddMod: true}}, VFX: "red", Color: "#ff3b3b"},
			{ID: "hollow_purple", Name: "Hollow Technique: Purple", MinLevel: 5, Cost: 9, Desc: "Collide Blue and Red into imaginary mass that erases what it touches: 8d10 to every enemy (DEX save half).", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "8d10", Scale: "2d10", DmgType: "erasure", Save: "DEX", Half: true}}, VFX: "purple", Color: "#b14aed"},
		},
		Ability{ID: "unlimited_void", Name: "Domain Expansion: Unlimited Void", Desc: "Flood every enemy with infinite information. They are Stunned (no save) at the start of each of your turns for 3 rounds; your attacks always hit.",
			Effects: []Effect{{Kind: "status", Target: "all_enemies", Status: "stun", Rounds: 1}, {Kind: "auto", Target: "all_enemies", Dice: "1d10", DmgType: "psychic"}}, VFX: "domain_void", Color: "#e0f2ff"})

	t(Technique{ID: "ten_shadows", Name: "Ten Shadows", Inspired: "Megumi Fushiguro", HitDie: 8, Priority: []string{"WIS", "INT", "CON", "DEX", "STR", "CHA"},
		Skills: []string{"survival", "curse_sense", "stealth"}, Weapon: "fists", Armor: "uniform", Color: "#4a4e69",
		Blurb: "Shadows become shikigami. Hounds, birds, toads and a flood-bringing elephant answer your hand signs."},
		[]Ability{
			{ID: "divine_dogs", Name: "Divine Dogs", Cost: 2, Desc: "Two shadow hounds maul a target: 2 attacks of 1d8 + mod.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "1d8", Scale: "1d8", DmgType: "slashing", Hits: 2, AddMod: true}}, VFX: "dogs", Color: "#cdd5e0"},
			{ID: "nue", Name: "Nue", Cost: 3, Desc: "The owl-shikigami dives with lightning: 3d8 (DEX save half); failure Stuns for 1 round.", Effects: []Effect{{Kind: "save", Target: "enemy", Dice: "3d8", Scale: "1d8", DmgType: "lightning", Save: "DEX", Half: true, Status: "stun", Rounds: 1}}, VFX: "lightning", Color: "#ffe66d"},
			{ID: "toad", Name: "Toad", Bonus: true, Cost: 2, Desc: "A tongue lashes out and binds: STR save or Bound (Exposed and can't act) for 1 round.", Effects: []Effect{{Kind: "save", Target: "enemy", Save: "STR", Status: "stun", Rounds: 1}}, VFX: "tongue", Color: "#7bd389"},
			{ID: "max_elephant", Name: "Max Elephant", MinLevel: 5, Cost: 7, Desc: "A colossal elephant floods the field: 6d8 bludgeoning to all enemies (STR save half), Exposed.", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "6d8", Scale: "2d8", DmgType: "bludgeoning", Save: "STR", Half: true, Status: "exposed", Rounds: 1}}, VFX: "flood", Color: "#4cc9f0"},
		},
		Ability{ID: "chimera_shadow_garden", Name: "Domain Expansion: Chimera Shadow Garden", Desc: "The ground becomes a sea of shadow. Each of your turns: shikigami strike every enemy for 3d8 and they are Exposed; allies are Empowered.",
			Effects: []Effect{{Kind: "auto", Target: "all_enemies", Dice: "3d8", DmgType: "slashing", Status: "exposed", Rounds: 1}, {Kind: "status", Target: "all_allies", Status: "empowered", Rounds: 1}}, VFX: "domain_shadow", Color: "#22223b"})

	t(Technique{ID: "cursed_speech", Name: "Cursed Speech", Inspired: "Toge Inumaki", HitDie: 8, Priority: []string{"CHA", "CON", "DEX", "WIS", "INT", "STR"},
		Skills: []string{"intimidation", "persuasion", "perception"}, Weapon: "fists", Armor: "uniform", Color: "#c77dff",
		Blurb: "Every word is a command. Enemies obey — but your throat pays in HP for every order."},
		[]Ability{
			{ID: "dont_move", Name: "\"Don't Move.\"", Cost: 2, HPCost: 2, Desc: "WIS save or Stunned for 1 round. Costs 2 HP (throat strain).", Effects: []Effect{{Kind: "save", Target: "enemy", Save: "WIS", Status: "stun", Rounds: 1}}, VFX: "voice", Color: "#c77dff"},
			{ID: "blast_away", Name: "\"Blast Away.\"", Cost: 3, HPCost: 2, Desc: "4d8 force (CON save half) and the target is Exposed.", Effects: []Effect{{Kind: "save", Target: "enemy", Dice: "4d8", Scale: "1d8", DmgType: "force", Save: "CON", Half: true, Status: "exposed", Rounds: 1}}, VFX: "voice_blast", Color: "#e0aaff"},
			{ID: "crush", Name: "\"Crush.\"", MinLevel: 3, Cost: 4, HPCost: 3, Desc: "5d8 bludgeoning (WIS save half).", Effects: []Effect{{Kind: "save", Target: "enemy", Dice: "5d8", Scale: "2d8", DmgType: "bludgeoning", Save: "WIS", Half: true}}, VFX: "voice_crush", Color: "#9d4edd"},
			{ID: "sleep", Name: "\"Sleep.\"", MinLevel: 5, Cost: 6, HPCost: 5, Desc: "Every enemy makes a WIS save or is Stunned for 2 rounds.", Effects: []Effect{{Kind: "save", Target: "all_enemies", Save: "WIS", Status: "stun", Rounds: 2}}, VFX: "voice", Color: "#7b2cbf"},
		},
		Ability{ID: "serpent_throat", Name: "Domain Expansion: Hall of a Silent Throat", Desc: "(Homebrew) Your voice echoes from every wall. Each of your turns: every enemy takes 2d10 psychic and makes a WIS save or is Stunned.",
			Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "2d10", DmgType: "psychic", Save: "WIS", Status: "stun", Rounds: 1}}, VFX: "domain_voice", Color: "#3c096c"})

	t(Technique{ID: "straw_doll", Name: "Straw Doll", Inspired: "Nobara Kugisaki", HitDie: 8, Priority: []string{"DEX", "CHA", "CON", "WIS", "INT", "STR"},
		Skills: []string{"intimidation", "perception", "sleight_of_hand"}, Weapon: "hammer", Armor: "uniform", Color: "#e76f51",
		Blurb: "Hammer, nails and a straw doll. Mark enemies with nails, then make them resonate."},
		[]Ability{
			{ID: "nails", Name: "Hammer & Nails", Cost: 1, Desc: "Fire three cursed nails: 3 attacks of 1d6 + mod. Each hit Marks the target (+1d8 from all party hits).", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "1d6", Scale: "1d6", DmgType: "piercing", Hits: 3, AddMod: true, Status: "marked", Rounds: 3}}, VFX: "nails", Color: "#f4a261"},
			{ID: "hairpin", Name: "Hairpin", Cost: 3, Desc: "Nails embedded in enemies explode: 3d8 to every enemy (DEX save half).", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "3d8", Scale: "1d8", DmgType: "force", Save: "DEX", Half: true}}, VFX: "explode", Color: "#e9c46a"},
			{ID: "resonance", Name: "Resonance", Cost: 4, HPCost: 1, Desc: "Hammer a nail into the straw doll: 4d8 to the target that can't be dodged, +3d8 if Marked.", Effects: []Effect{{Kind: "auto", Target: "enemy", Dice: "4d8", Scale: "1d8", DmgType: "psychic"}}, VFX: "resonance", Color: "#e63946"},
			{ID: "black_flash_doll", Name: "Resonance: Black Flash", MinLevel: 5, Cost: 7, Desc: "Resonance on a nail driven by a Black Flash: 9d8 to one target.", Effects: []Effect{{Kind: "auto", Target: "enemy", Dice: "9d8", Scale: "2d8", DmgType: "psychic"}}, VFX: "blackflash", Color: "#d00000"},
		},
		Ability{ID: "doll_hall", Name: "Domain Expansion: Garden of Thousand Dolls", Desc: "(Homebrew) Straw dolls hang from every branch. Each of your turns: every enemy is Marked and resonates for 3d8.",
			Effects: []Effect{{Kind: "auto", Target: "all_enemies", Dice: "3d8", DmgType: "psychic", Status: "marked", Rounds: 2}}, VFX: "domain_dolls", Color: "#9b2226"})

	t(Technique{ID: "blood_manipulation", Name: "Blood Manipulation", Inspired: "Noritoshi Kamo / Choso", HitDie: 10, Priority: []string{"CON", "INT", "DEX", "STR", "WIS", "CHA"},
		Skills: []string{"medicine", "jujutsu_lore", "athletics"}, Weapon: "fists", Armor: "uniform", Color: "#c1121f",
		Blurb: "Your blood is your weapon. Spend HP alongside CE for brutal, precise damage."},
		[]Ability{
			{ID: "piercing_blood", Name: "Piercing Blood", Cost: 2, HPCost: 2, Desc: "Compressed blood fired at supersonic speed: 4d8 piercing on a hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "4d8", Scale: "1d8", DmgType: "piercing", AddMod: true}}, VFX: "blood_beam", Color: "#d90429"},
			{ID: "flowing_red_scale", Name: "Flowing Red Scale", Bonus: true, Cost: 2, HPCost: 2, Desc: "Boost your body's output: Empowered and Hasted for 2 rounds.", Effects: []Effect{{Kind: "status", Target: "self", Status: "empowered", Rounds: 2}, {Kind: "status", Target: "self", Status: "haste", Rounds: 1}}, VFX: "aura_red", Color: "#ef233c"},
			{ID: "slicing_exorcism", Name: "Slicing Exorcism", Cost: 3, HPCost: 1, Desc: "Blood buzzsaws carve through all enemies: 2d8 each (DEX save half) and Bleeding.", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "2d8", Scale: "1d8", DmgType: "slashing", Save: "DEX", Half: true, Status: "bleed", Rounds: 2}}, VFX: "blood_saw", Color: "#9d0208"},
			{ID: "supernova", Name: "Supernova", MinLevel: 5, Cost: 6, HPCost: 4, Desc: "Orbs of compressed blood burst at once: 6d8 to every enemy (DEX save half).", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "6d8", Scale: "2d8", DmgType: "piercing", Save: "DEX", Half: true}}, VFX: "blood_nova", Color: "#ff006e"},
		},
		Ability{ID: "crimson_garden", Name: "Domain Expansion: Crimson Lotus Garden", Desc: "(Homebrew) A pond of your blood. Each of your turns: every enemy takes 3d8 and Bleeds, and you regenerate.",
			Effects: []Effect{{Kind: "auto", Target: "all_enemies", Dice: "3d8", DmgType: "piercing", Status: "bleed", Rounds: 2}, {Kind: "status", Target: "self", Status: "regen", Rounds: 1}}, VFX: "domain_blood", Color: "#6a040f"})

	t(Technique{ID: "shrine", Name: "Shrine", Inspired: "Ryomen Sukuna", HitDie: 8, Priority: []string{"INT", "STR", "CON", "DEX", "CHA", "WIS"},
		Skills: []string{"intimidation", "jujutsu_lore", "athletics"}, Weapon: "fists", Armor: "uniform", Color: "#d62828",
		Blurb: "The King of Curses' technique. Invisible slashes that adjust to their target. Cruel and efficient."},
		[]Ability{
			{ID: "dismantle", Name: "Dismantle", Cost: 2, Desc: "Invisible slashes: 2 attacks of 1d10 + mod.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "1d10", Scale: "1d10", DmgType: "slashing", Hits: 2, AddMod: true}}, VFX: "slash", Color: "#ffffff"},
			{ID: "cleave", Name: "Cleave", Cost: 4, Desc: "A slash calibrated to the target's toughness: 2d8 + 15% of its max HP on a hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "2d8", Scale: "1d8", DmgType: "slashing", PctMax: 15, AddMod: true}}, VFX: "cleave", Color: "#ff4d6d"},
			{ID: "spiderweb", Name: "Dismantle: Spiderweb", MinLevel: 3, Cost: 4, Desc: "Slashes radiate through the ground under every enemy: 3d8 (DEX save half), Exposed.", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "3d8", Scale: "1d8", DmgType: "slashing", Save: "DEX", Half: true, Status: "exposed", Rounds: 1}}, VFX: "web", Color: "#ffccd5"},
			{ID: "fuga", Name: "Open: Fuga", MinLevel: 5, Cost: 8, Desc: "Draw a bow of flame: 7d8 fire to every enemy (DEX save half) and Burning.", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "7d8", Scale: "2d8", DmgType: "fire", Save: "DEX", Half: true, Status: "burn", Rounds: 2}}, VFX: "fire_arrow", Color: "#ff7b00"},
		},
		Ability{ID: "malevolent_shrine", Name: "Domain Expansion: Malevolent Shrine", Rounds: 2, Desc: "A domain without a barrier. Each of your turns: endless Cleave and Dismantle hit every enemy for 5d10, no save.",
			Effects: []Effect{{Kind: "auto", Target: "all_enemies", Dice: "5d10", DmgType: "slashing"}}, VFX: "domain_shrine", Color: "#7f0000"})

	t(Technique{ID: "disaster_flames", Name: "Disaster Flames", Inspired: "Jogo", HitDie: 8, Priority: []string{"INT", "CON", "DEX", "WIS", "CHA", "STR"},
		Skills: []string{"jujutsu_lore", "intimidation", "survival"}, Weapon: "fists", Armor: "uniform", Color: "#f77f00",
		Blurb: "Volcanic heat bent to your will. Burn everything and keep it burning."},
		[]Ability{
			{ID: "ember_insects", Name: "Ember Insects", Cost: 2, Desc: "Burning insects swarm a target: 3 attacks of 1d6 fire. Hits set it Burning.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "1d6", Scale: "1d6", DmgType: "fire", Hits: 3, Status: "burn", Rounds: 2}}, VFX: "embers", Color: "#fcbf49"},
			{ID: "eruption", Name: "Eruption", Cost: 4, Desc: "The ground erupts under all enemies: 3d8 fire (DEX save half), Burning.", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "3d8", Scale: "1d8", DmgType: "fire", Save: "DEX", Half: true, Status: "burn", Rounds: 2}}, VFX: "eruption", Color: "#f77f00"},
			{ID: "flame_grip", Name: "Volcanic Grip", Cost: 3, Desc: "Grab and incinerate: 4d10 fire on a melee hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "4d10", Scale: "1d10", DmgType: "fire", Melee: true}}, VFX: "fire_punch", Color: "#d62828"},
			{ID: "maximum_meteor", Name: "Maximum: Meteor", MinLevel: 5, Cost: 9, Desc: "Drop a miniature sun: 8d8 fire to every enemy (DEX save half), Burning.", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "8d8", Scale: "2d8", DmgType: "fire", Save: "DEX", Half: true, Status: "burn", Rounds: 3}}, VFX: "meteor", Color: "#ff5400"},
		},
		Ability{ID: "iron_mountain", Name: "Domain Expansion: Coffin of the Iron Mountain", Desc: "The inside of a volcano. Each of your turns: every enemy takes 4d8 fire and Burns.",
			Effects: []Effect{{Kind: "auto", Target: "all_enemies", Dice: "4d8", DmgType: "fire", Status: "burn", Rounds: 2}}, VFX: "domain_volcano", Color: "#9d0208"})

	t(Technique{ID: "idle_transfiguration", Name: "Idle Transfiguration", Inspired: "Mahito", HitDie: 10, Priority: []string{"WIS", "CON", "DEX", "STR", "INT", "CHA"},
		Skills: []string{"insight", "deception", "medicine"}, Weapon: "fists", Armor: "uniform", Color: "#8ecae6",
		Blurb: "Touch the soul and reshape it — theirs into ruin, yours into whatever you need."},
		[]Ability{
			{ID: "soul_touch", Name: "Soul Touch", Cost: 3, Desc: "A palm on the soul: 3d10 necrotic on a melee hit, ignores Infinity and guards, Weakened.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "3d10", Scale: "1d10", DmgType: "necrotic", Melee: true, Status: "weakened", Rounds: 2}}, VFX: "soul", Color: "#90e0ef"},
			{ID: "body_reshape", Name: "Body Repair", Bonus: true, Cost: 2, Desc: "Reshape your own soul: heal 2d8 + mod and Guard.", Effects: []Effect{{Kind: "heal", Target: "self", Dice: "2d8", Scale: "1d8", AddMod: true}, {Kind: "status", Target: "self", Status: "guard", Rounds: 1}}, VFX: "rct", Color: "#caf0f8"},
			{ID: "transfigured", Name: "Transfigured Humans", Cost: 3, Desc: "Release reshaped puppets: 3 attacks of 1d8 + mod.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "1d8", Scale: "1d8", DmgType: "bludgeoning", Hits: 3, AddMod: true}}, VFX: "puppets", Color: "#adb5bd"},
			{ID: "isomer", Name: "Polymorphic Soul Isomer", MinLevel: 5, Cost: 7, Desc: "Detonate fused souls among the enemy: 6d8 necrotic to all (CON save half).", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "6d8", Scale: "2d8", DmgType: "necrotic", Save: "CON", Half: true}}, VFX: "isomer", Color: "#48cae4"},
		},
		Ability{ID: "self_embodiment", Name: "Domain Expansion: Self-Embodiment of Perfection", Desc: "Hands close around everything. Each of your turns: every enemy makes a CON save or takes 5d10 necrotic and is Weakened.",
			Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "5d10", DmgType: "necrotic", Save: "CON", Status: "weakened", Rounds: 1}}, VFX: "domain_hands", Color: "#023e8a"})

	t(Technique{ID: "ratio", Name: "Ratio Technique", Inspired: "Kento Nanami", HitDie: 10, Priority: []string{"STR", "INT", "CON", "DEX", "WIS", "CHA"},
		Skills: []string{"investigation", "insight", "athletics"}, Weapon: "blunt_blade", Armor: "suit", Color: "#e9c46a",
		Blurb: "Divide any target 7:3 and strike the weak point. When the clock runs out, Overtime begins."},
		[]Ability{
			{ID: "ratio_strike", Name: "7:3 Strike", Cost: 1, Desc: "A weapon strike at the weak point: +2d8 damage, critical on 18-20.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "weapon+2d8", Scale: "1d8", AddMod: true, Melee: true}}, VFX: "ratio", Color: "#ffd60a"},
			{ID: "collapse", Name: "Collapse", Cost: 4, Desc: "Strike a structure's weak point and bury every enemy: 4d8 bludgeoning (DEX save half).", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "4d8", Scale: "1d8", DmgType: "bludgeoning", Save: "DEX", Half: true}}, VFX: "collapse", Color: "#adb5bd"},
			{ID: "overtime", Name: "Overtime", Bonus: true, PerBattle: 1, Desc: "Loosen your tie. Empowered and In the Zone for 4 rounds.", Effects: []Effect{{Kind: "status", Target: "self", Status: "empowered", Rounds: 4}, {Kind: "status", Target: "self", Status: "zone", Rounds: 4}}, VFX: "aura_gold", Color: "#ffc300"},
			{ID: "ratio_finale", Name: "Overtime: Final Ratio", MinLevel: 5, Cost: 6, Desc: "Everything in one 7:3 blow: weapon + 6d10 on a hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "weapon+6d10", Scale: "2d10", AddMod: true, Melee: true}}, VFX: "ratio", Color: "#ffb703"},
		},
		Ability{ID: "overtime_office", Name: "Domain Expansion: Endless Overtime", Desc: "(Homebrew) An office that never closes. Each of your turns: every enemy is Exposed and Weakened and takes 3d10.",
			Effects: []Effect{{Kind: "auto", Target: "all_enemies", Dice: "3d10", DmgType: "bludgeoning", Status: "exposed", Rounds: 1}, {Kind: "status", Target: "all_enemies", Status: "weakened", Rounds: 1}}, VFX: "domain_office", Color: "#6c584c"})

	t(Technique{ID: "boogie_woogie", Name: "Boogie Woogie", Inspired: "Aoi Todo", HitDie: 12, Priority: []string{"STR", "CON", "DEX", "CHA", "WIS", "INT"},
		Skills: []string{"athletics", "performance", "intimidation"}, Weapon: "fists", Armor: "uniform", Color: "#2a9d8f",
		Blurb: "Clap to swap places. Break your enemy's rhythm and set up your best friends."},
		[]Ability{
			{ID: "clap_swap", Name: "Clap: Swap", Bonus: true, Cost: 1, Desc: "Swap positions mid-fight: one ally is Empowered and the target is Exposed for 1 round.", Effects: []Effect{{Kind: "status", Target: "ally", Status: "empowered", Rounds: 1}, {Kind: "status", Target: "enemy", Status: "exposed", Rounds: 1}}, VFX: "clap", Color: "#80ed99"},
			{ID: "feint_clap", Name: "Feint Clap", Cost: 1, Desc: "Did you swap or not? The target is Weakened and Exposed (WIS save negates).", Effects: []Effect{{Kind: "save", Target: "enemy", Save: "WIS", Status: "weakened", Rounds: 2}, {Kind: "status", Target: "enemy", Status: "exposed", Rounds: 1}}, VFX: "clap", Color: "#57cc99"},
			{ID: "haymaker", Name: "Brotherly Haymaker", Cost: 2, Desc: "A colossal punch: weapon + 3d8 on a hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "weapon+3d8", Scale: "1d8", AddMod: true, Melee: true}}, VFX: "punch", Color: "#ffffff"},
			{ID: "boogie_barrage", Name: "Boogie Woogie Barrage", MinLevel: 5, Cost: 6, Desc: "Swap-swap-swap: 4 attacks of 1d10 + mod from every direction.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "1d10", Scale: "1d10", Hits: 4, AddMod: true, Melee: true}}, VFX: "clap", Color: "#38a3a5"},
		},
		Ability{ID: "best_friend_stadium", Name: "Domain Expansion: Best Friend Stadium", Desc: "(Homebrew) A roaring crowd of memories. Each of your turns: allies are Hasted and Empowered, enemies Exposed.",
			Effects: []Effect{{Kind: "status", Target: "all_allies", Status: "haste", Rounds: 1}, {Kind: "status", Target: "all_allies", Status: "empowered", Rounds: 1}, {Kind: "status", Target: "all_enemies", Status: "exposed", Rounds: 1}}, VFX: "domain_stadium", Color: "#22577a"})

	t(Technique{ID: "projection", Name: "Projection Sorcery", Inspired: "Naobito / Naoya Zenin", HitDie: 8, Priority: []string{"DEX", "INT", "CON", "WIS", "STR", "CHA"},
		Skills: []string{"acrobatics", "perception", "stealth"}, Weapon: "fists", Armor: "uniform", Color: "#90be6d",
		Blurb: "Divide one second into 24 frames. Move at impossible speed — and freeze anyone who can't keep up."},
		[]Ability{
			{ID: "frames_24", Name: "24 FPS Rule", Bonus: true, Cost: 2, Desc: "Program your movement: Hasted (an extra action) for 1 round and In the Zone.", Effects: []Effect{{Kind: "status", Target: "self", Status: "haste", Rounds: 1}, {Kind: "status", Target: "self", Status: "zone", Rounds: 2}}, VFX: "frames", Color: "#b5e48c"},
			{ID: "frame_freeze", Name: "Frame Freeze", Cost: 3, Desc: "Touch them and force the rule: 2d8 + mod on a hit, and they're Stunned (frozen in a frame) for 1 round.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "2d8", Scale: "1d8", AddMod: true, Melee: true, Status: "stun", Rounds: 1}}, VFX: "freeze_frame", Color: "#d9ed92"},
			{ID: "rapid_frames", Name: "Rapid Projection", Cost: 2, Desc: "Three blows inside one second: 3 attacks of 1d8 + mod.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "1d8", Scale: "1d8", Hits: 3, AddMod: true, Melee: true}}, VFX: "frames", Color: "#99d98c"},
			{ID: "sonic_strike", Name: "Mach Projection", MinLevel: 5, Cost: 6, Desc: "Accelerate frame by frame past the speed of sound: 7d8 on a hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "7d8", Scale: "2d8", AddMod: true, Melee: true}}, VFX: "sonic", Color: "#52b69a"},
		},
		Ability{ID: "moon_palace", Name: "Domain Expansion: Time Cell Moon Palace", Desc: "Every cell of their bodies must obey your frames. Each of your turns: every enemy takes 4d8 and makes a DEX save or is Stunned.",
			Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "4d8", DmgType: "force", Save: "DEX", Status: "stun", Rounds: 1}}, VFX: "domain_moon", Color: "#1e6091"})

	t(Technique{ID: "private_pure_love_train", Name: "Private Pure Love Train", Inspired: "Kinji Hakari", HitDie: 10, Priority: []string{"CON", "STR", "CHA", "DEX", "WIS", "INT"},
		Skills: []string{"athletics", "performance", "deception"}, Weapon: "fists", Armor: "uniform", Color: "#06d6a0",
		Blurb: "Pachinko and fists. Gamble for the jackpot: infinite cursed energy and a body that heals itself."},
		[]Ability{
			{ID: "rough_energy", Name: "Rough Energy", Cost: 2, Desc: "Unrefined cursed energy in a punch: weapon + 2d8 on a hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "weapon+2d8", Scale: "1d8", AddMod: true, Melee: true}}, VFX: "rough", Color: "#06d6a0"},
			{ID: "shutter_doors", Name: "Shutter Doors", Cost: 3, Desc: "Train doors slam on every enemy: DEX save or Stunned for 1 round.", Effects: []Effect{{Kind: "save", Target: "all_enemies", Save: "DEX", Status: "stun", Rounds: 1}}, VFX: "doors", Color: "#bde0fe"},
			{ID: "reserve_balls", Name: "Reserve Balls", Cost: 1, Desc: "Steel pachinko balls: 2 attacks of 1d10.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "1d10", Scale: "1d10", Hits: 2, DmgType: "bludgeoning"}}, VFX: "balls", Color: "#ced4da"},
			{ID: "fever_breaker", Name: "Fever Breaker", MinLevel: 5, Cost: 6, Desc: "Doors, a dropkick and a train: weapon + 6d8 on a hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "weapon+6d8", Scale: "2d8", AddMod: true, Melee: true}}, VFX: "fever", Color: "#ef476f"},
		},
		Ability{ID: "idle_death_gamble", Name: "Domain Expansion: Idle Death Gamble", Desc: "A pachinko machine with no sure-hit damage. Each of your turns, spin: 1 in 3 is JACKPOT — full HP and CE, Empowered, for the rest of the fight.",
			Effects: []Effect{{Kind: "gamble", Target: "self"}}, VFX: "domain_pachinko", Color: "#06d6a0"})

	t(Technique{ID: "cursed_partner", Name: "Cursed Partner", Inspired: "Yuta Okkotsu & Rika", HitDie: 10, Priority: []string{"CHA", "CON", "STR", "DEX", "WIS", "INT"},
		Skills: []string{"persuasion", "athletics", "curse_sense"}, Weapon: "katana", Armor: "uniform", Color: "#f72585",
		Blurb: "A Special Grade curse loves you. Command her, borrow her bottomless energy, and copy the techniques of others."},
		[]Ability{
			{ID: "rika_smash", Name: "Rika: Smash", Cost: 3, Desc: "Rika's fist comes down: 4d10 on a hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "4d10", Scale: "1d10", DmgType: "bludgeoning"}}, VFX: "rika", Color: "#ff70a6"},
			{ID: "rika_shield", Name: "Rika: Embrace", Bonus: true, Cost: 2, Desc: "Rika wraps around an ally: Guard for 1 round.", Effects: []Effect{{Kind: "status", Target: "ally", Status: "guard", Rounds: 1}}, VFX: "shield", Color: "#ff99c8"},
			{ID: "copy", Name: "Copy", Cost: 4, Desc: "Rika eats a technique she has seen: use a random move from another party member's technique (or the enemy's).", Effects: []Effect{{Kind: "copy", Target: "enemy"}}, VFX: "copy", Color: "#f15bb5"},
			{ID: "true_love_beam", Name: "Pure Love Beam", MinLevel: 5, Cost: 9, Desc: "Everything poured into one beam: 8d8 to every enemy (DEX save half).", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "8d8", Scale: "2d8", DmgType: "force", Save: "DEX", Half: true}}, VFX: "love_beam", Color: "#ff5d8f"},
		},
		Ability{ID: "authentic_mutual_love", Name: "Domain Expansion: Authentic Mutual Love", Desc: "A field of crosses and swords, each holding a copied technique. Each of your turns: a random copied technique fires for free, plus 2d10 to all enemies.",
			Effects: []Effect{{Kind: "auto", Target: "all_enemies", Dice: "2d10", DmgType: "slashing"}, {Kind: "copy", Target: "enemy"}}, VFX: "domain_love", Color: "#590d22"})

	t(Technique{ID: "heavenly_restriction", Name: "Heavenly Restriction", Inspired: "Toji / Maki Zenin", HitDie: 12, Priority: []string{"STR", "DEX", "CON", "WIS", "INT", "CHA"},
		Skills: []string{"athletics", "acrobatics", "perception", "stealth"}, Weapon: "playful_cloud", Armor: "uniform", Color: "#6c757d", NoCE: true,
		Blurb: "Zero cursed energy, a monstrous body. Immune to sure-hit, invisible to barriers, deadly with cursed tools. Moves are limited per battle instead of costing CE."},
		[]Ability{
			{ID: "tool_flurry", Name: "Cursed Tool Flurry", PerBattle: 3, Desc: "Three blinding strikes with your cursed tool: 3 attacks of weapon + mod.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "weapon", Hits: 3, AddMod: true, Melee: true}}, VFX: "tool", Color: "#adb5bd"},
			{ID: "spear_of_heaven", Name: "Inverted Spear Thrust", PerBattle: 2, Desc: "A tool that nullifies techniques: 3d8 + mod on a hit and strips all of the target's buffs (and breaks a domain).", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "3d8", Scale: "1d8", AddMod: true, Melee: true}, {Kind: "cleanse", Target: "enemy"}}, VFX: "spear", Color: "#e9ecef"},
			{ID: "sixth_sense", Name: "Sixth Sense", Bonus: true, PerBattle: 2, Desc: "Read the attack before it comes: Infinity-like evasion for 1 round.", Effects: []Effect{{Kind: "status", Target: "self", Status: "infinity", Rounds: 1}}, VFX: "dash", Color: "#dee2e6"},
			{ID: "soul_split", Name: "Split Soul Katana", MinLevel: 5, PerBattle: 2, Desc: "Cut the soul, not the body: 6d10 that can't be dodged.", Effects: []Effect{{Kind: "auto", Target: "enemy", Dice: "6d10", Scale: "2d10", DmgType: "slashing"}}, VFX: "slash", Color: "#f8f9fa"},
		},
		Ability{ID: "unbound_body", Name: "Unbound Body", Desc: "You have no domain — you don't need one. Hasted, Empowered and In the Zone for the rest of the fight. (Passive: domains' sure-hit never affects you.)",
			Effects: []Effect{{Kind: "status", Target: "self", Status: "haste", Rounds: 9}, {Kind: "status", Target: "self", Status: "empowered", Rounds: 9}, {Kind: "status", Target: "self", Status: "zone", Rounds: 9}}, VFX: "aura_white", Color: "#ffffff", Rounds: 1})

	t(Technique{ID: "construction", Name: "Construction", Inspired: "Mai Zenin", HitDie: 8, Priority: []string{"INT", "DEX", "WIS", "CON", "CHA", "STR"},
		Skills: []string{"investigation", "sleight_of_hand", "barriers"}, Weapon: "revolver", Armor: "uniform", Color: "#ffafcc",
		Blurb: "Create matter from nothing — bullets, barricades, blades. Expensive, but you're never out of ammo."},
		[]Ability{
			{ID: "conjured_bullet", Name: "Conjured Bullet", Cost: 1, Desc: "A cursed round made from nothing: 2d10 + mod piercing on a hit.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "2d10", Scale: "1d10", DmgType: "piercing", AddMod: true}}, VFX: "bullet", Color: "#ffc8dd"},
			{ID: "barricade", Name: "Barricade", Bonus: true, Cost: 2, Desc: "Construct a wall before an ally: Guard for 1 round.", Effects: []Effect{{Kind: "status", Target: "ally", Status: "guard", Rounds: 1}}, VFX: "wall", Color: "#cdb4db"},
			{ID: "construct_blade", Name: "Construct: Blades", Cost: 3, Desc: "Rain conjured blades on all enemies: 3d8 (DEX save half).", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "3d8", Scale: "1d8", DmgType: "slashing", Save: "DEX", Half: true}}, VFX: "blades", Color: "#bde0fe"},
			{ID: "final_construct", Name: "Last Creation", MinLevel: 5, Cost: 8, Desc: "Construct something perfect: heal an ally 4d8 + mod and give them Empowered.", Effects: []Effect{{Kind: "heal", Target: "ally", Dice: "4d8", Scale: "1d8", AddMod: true}, {Kind: "status", Target: "ally", Status: "empowered", Rounds: 3}}, VFX: "rct", Color: "#ffafcc"},
		},
		Ability{ID: "foundry", Name: "Domain Expansion: Infinite Foundry", Desc: "(Homebrew) A workshop where anything can be made. Each of your turns: every enemy takes 3d8 from constructed spikes, and every ally is Guarded.",
			Effects: []Effect{{Kind: "auto", Target: "all_enemies", Dice: "3d8", DmgType: "piercing"}, {Kind: "status", Target: "all_allies", Status: "guard", Rounds: 1}}, VFX: "domain_foundry", Color: "#a2d2ff"})

	t(Technique{ID: "mythical_beast_amber", Name: "Mythical Beast Amber", Inspired: "Hajime Kashimo", HitDie: 8, Priority: []string{"DEX", "CON", "STR", "INT", "WIS", "CHA"},
		Skills: []string{"acrobatics", "athletics", "intimidation"}, Weapon: "staff", Armor: "uniform", Color: "#4cc9f0",
		Blurb: "Your cursed energy is electricity. Strike at the speed of lightning — and one day, become it."},
		[]Ability{
			{ID: "lightning_strike", Name: "Lightning Strike", Cost: 2, Desc: "A staff blow carrying current: weapon + 2d6 lightning, and the target is Exposed.", Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: "weapon+2d6", Scale: "1d6", DmgType: "lightning", AddMod: true, Melee: true, Status: "exposed", Rounds: 1}}, VFX: "lightning", Color: "#4cc9f0"},
			{ID: "charge", Name: "Electric Charge", Bonus: true, Cost: 1, Desc: "Store current in your body: Empowered for 2 rounds.", Effects: []Effect{{Kind: "status", Target: "self", Status: "empowered", Rounds: 2}}, VFX: "aura_blue", Color: "#72efdd"},
			{ID: "chain_lightning", Name: "Chain Discharge", Cost: 4, Desc: "Lightning arcs between all enemies: 3d10 (DEX save half).", Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "3d10", Scale: "1d10", DmgType: "lightning", Save: "DEX", Half: true}}, VFX: "chain", Color: "#4895ef"},
			{ID: "amber_beast", Name: "Mythical Beast Amber", MinLevel: 5, Cost: 10, PerBattle: 1, Desc: "Your body becomes lightning, once in your life... or once per battle here: 10d10 to one enemy, unavoidable. You take 2d10.", Effects: []Effect{{Kind: "auto", Target: "enemy", Dice: "10d10", Scale: "2d10", DmgType: "lightning"}}, VFX: "amber", Color: "#ffffff", HPCost: 10},
		},
		Ability{ID: "storm_domain", Name: "Domain Expansion: Thunderhead", Desc: "(Homebrew) A storm in a jar. Each of your turns: lightning strikes every enemy for 4d8 (DEX save half), Exposed.",
			Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "4d8", DmgType: "lightning", Save: "DEX", Half: true, Status: "exposed", Rounds: 1}}, VFX: "domain_storm", Color: "#3a0ca3"})
}

// SortedTechniqueIDs lists techniques alphabetically by name.
func SortedTechniqueIDs() []string {
	ids := make([]string, 0, len(Techniques))
	for id := range Techniques {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return Techniques[ids[i]].Name < Techniques[ids[j]].Name })
	return ids
}

// ---------- items ----------

type Item struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"` // weapon, armor, consumable, gear, treasure, quest
	Damage   string `json:"damage,omitempty"`
	DmgType  string `json:"dmgType,omitempty"`
	Finesse  bool   `json:"finesse,omitempty"`
	Ranged   bool   `json:"ranged,omitempty"`
	BaseAC   int    `json:"baseAC,omitempty"`
	MaxDex   int    `json:"maxDex,omitempty"` // -1 = unlimited
	Heal     string `json:"heal,omitempty"`
	CE       int    `json:"ce,omitempty"`
	Bonus    int    `json:"bonus,omitempty"`
	Special  string `json:"special,omitempty"` // ignore_ac, dispel, ce_boost
	Value    int    `json:"value"`
	Desc     string `json:"desc,omitempty"`
	Qty      int    `json:"qty"`
	Custom   bool   `json:"custom,omitempty"`
	Equipped bool   `json:"equipped,omitempty"`
	Grade    int    `json:"grade,omitempty"`
}

var ItemList = map[string]Item{
	"fists":            {ID: "fists", Name: "Cursed-Energy Fists", Kind: "weapon", Damage: "1d6", DmgType: "bludgeoning", Finesse: true, Desc: "Your own body, reinforced."},
	"hammer":           {ID: "hammer", Name: "Straw Doll Hammer", Kind: "weapon", Damage: "1d8", DmgType: "bludgeoning", Value: 3000},
	"blunt_blade":      {ID: "blunt_blade", Name: "Wrapped Blunt Blade", Kind: "weapon", Damage: "1d10", DmgType: "bludgeoning", Value: 8000, Desc: "A cleaver wrapped in a patterned cloth."},
	"katana":           {ID: "katana", Name: "Cursed Katana", Kind: "weapon", Damage: "1d8", DmgType: "slashing", Finesse: true, Value: 12000},
	"staff":            {ID: "staff", Name: "Jo Staff", Kind: "weapon", Damage: "1d8", DmgType: "bludgeoning", Finesse: true, Value: 5000},
	"revolver":         {ID: "revolver", Name: "Revolver", Kind: "weapon", Damage: "1d10", DmgType: "piercing", Ranged: true, Value: 15000},
	"playful_cloud":    {ID: "playful_cloud", Name: "Playful Cloud", Kind: "weapon", Damage: "2d6", DmgType: "bludgeoning", Bonus: 1, Value: 500000, Grade: 4, Desc: "A three-section staff of pure physical force — a Special Grade tool."},
	"slaughter_demon":  {ID: "slaughter_demon", Name: "Slaughter Demon", Kind: "weapon", Damage: "1d6", DmgType: "slashing", Finesse: true, Bonus: 1, Value: 60000, Grade: 2, Desc: "A short blade that drinks cursed energy."},
	"dragon_bone":      {ID: "dragon_bone", Name: "Dragon-Bone", Kind: "weapon", Damage: "2d8", DmgType: "slashing", Bonus: 1, Value: 200000, Grade: 1, Desc: "Stores the impact of blows and releases it as a jet."},
	"split_soul":       {ID: "split_soul", Name: "Split Soul Katana", Kind: "weapon", Damage: "1d10", DmgType: "slashing", Finesse: true, Bonus: 2, Special: "ignore_ac", Value: 900000, Grade: 4, Desc: "Cuts the soul directly: attacks with it ignore armor."},
	"inverted_spear":   {ID: "inverted_spear", Name: "Inverted Spear of Heaven", Kind: "weapon", Damage: "1d8", DmgType: "piercing", Finesse: true, Bonus: 2, Special: "dispel", Value: 1500000, Grade: 4, Desc: "Nullifies techniques: hits strip the target's buffs and shatter domains."},
	"uniform":          {ID: "uniform", Name: "Jujutsu High Uniform", Kind: "armor", BaseAC: 11, MaxDex: -1, Desc: "Custom-tailored, warded."},
	"suit":             {ID: "suit", Name: "Office Suit & Tie", Kind: "armor", BaseAC: 12, MaxDex: 2, Desc: "Overtime-ready."},
	"reinforced":       {ID: "reinforced", Name: "Reinforced Uniform", Kind: "armor", BaseAC: 13, MaxDex: 3, Value: 30000},
	"tactical":         {ID: "tactical", Name: "Window Tactical Vest", Kind: "armor", BaseAC: 14, MaxDex: 2, Value: 80000},
	"cursed_garb":      {ID: "cursed_garb", Name: "Cursed Garment", Kind: "armor", BaseAC: 16, MaxDex: 1, Value: 250000, Grade: 2, Desc: "Woven from a curse's own hide."},
	"medicine":         {ID: "medicine", Name: "Shoko's Medicine", Kind: "consumable", Heal: "2d4+4", Value: 3000, Desc: "Bitter, effective."},
	"strong_medicine":  {ID: "strong_medicine", Name: "Emergency Stitch Kit", Kind: "consumable", Heal: "4d4+8", Value: 9000},
	"ce_tonic":         {ID: "ce_tonic", Name: "Cursed Energy Tonic", Kind: "consumable", CE: 6, Value: 5000, Desc: "Restores 6 cursed energy."},
	"cough_syrup":      {ID: "cough_syrup", Name: "Cough Syrup", Kind: "consumable", Heal: "2d4+2", CE: 2, Value: 1500, Desc: "Every Cursed Speech user's best friend."},
	"talisman":         {ID: "talisman", Name: "Exorcism Talisman", Kind: "consumable", Damage: "3d6", DmgType: "radiant", Value: 4000, Desc: "Throw at a curse: 3d6, never misses."},
	"cursed_glasses":   {ID: "cursed_glasses", Name: "Cursed Glasses", Kind: "gear", Value: 10000, Desc: "Let ordinary people see curses."},
	"veil_tag":         {ID: "veil_tag", Name: "Veil Charm", Kind: "gear", Value: 2000, Desc: "\"Emerging from darkness, darker than darkness...\""},
	"phone":            {ID: "phone", Name: "Smartphone", Kind: "gear", Desc: "Ijichi's number is saved."},
	"student_id":       {ID: "student_id", Name: "Jujutsu High Student ID", Kind: "gear"},
	"sukuna_finger":    {ID: "sukuna_finger", Name: "Sukuna's Finger", Kind: "quest", Value: 0, Desc: "A Special Grade cursed object. It is warm."},
	"cursed_womb_part": {ID: "cursed_womb_part", Name: "Cursed Womb: Death Painting", Kind: "quest", Desc: "A preserved cursed womb. Unsettling."},
	"lucky_coin":       {ID: "lucky_coin", Name: "Pachinko Token", Kind: "treasure", Value: 500},
	"cursed_trinket":   {ID: "cursed_trinket", Name: "Low-Grade Cursed Object", Kind: "treasure", Value: 8000, Desc: "Sells well to the right buyer."},
}

func NewItem(id string) (Item, bool) {
	it, ok := ItemList[id]
	if !ok {
		return Item{}, false
	}
	it.Qty = 1
	return it, true
}

// ---------- curses ----------

// SpriteSpec tells the client how to draw a curse procedurally.
type SpriteSpec struct {
	Shape  string `json:"shape"` // blob, humanoid, insect, beast, head, serpent, giant, swarm
	Color  string `json:"color"`
	Color2 string `json:"color2"`
	Eyes   int    `json:"eyes"`
	Mouth  string `json:"mouth"` // grin, maw, none, stitched, beak
	Size   int    `json:"size"`  // 1 small .. 4 colossal
	Extra  string `json:"extra"` // wings, horns, tentacles, arms, flames, flowers, crown, scissors, coffin, veil, halo
}

type CurseMove struct {
	Name    string   `json:"name"`
	Weight  int      `json:"weight"`
	Effects []Effect `json:"effects"`
	VFX     string   `json:"vfx"`
	Color   string   `json:"color"`
}

type CurseTemplate struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Grade   string      `json:"grade"`
	Rank    int         `json:"rank"` // 0 = Grade 4 ... 5 = Special Grade
	XP      int         `json:"xp"`
	HP      string      `json:"hp"`
	AC      int         `json:"ac"`
	Init    int         `json:"init"`
	ToHit   int         `json:"toHit"`
	Save    int         `json:"save"`
	Moves   []CurseMove `json:"moves"`
	Domain  string      `json:"domain,omitempty"`
	Tags    []string    `json:"tags"`
	Sprite  SpriteSpec  `json:"sprite"`
	Yen     string      `json:"yen"`
	Boss    bool        `json:"boss"`
	Human   bool        `json:"human"`
	Desc    string      `json:"desc"`
	Escapes bool        `json:"escapes,omitempty"`
}

var GradeNames = []string{"Grade 4", "Grade 3", "Grade 2", "Grade 1", "Semi-Special", "Special Grade"}

func hit(name, dice, dmg string, w int, vfx, color string) CurseMove {
	return CurseMove{Name: name, Weight: w, Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: dice, DmgType: dmg}}, VFX: vfx, Color: color}
}

func hitS(name, dice, dmg string, w int, vfx, color, status string, rounds int) CurseMove {
	return CurseMove{Name: name, Weight: w, Effects: []Effect{{Kind: "attack", Target: "enemy", Dice: dice, DmgType: dmg, Status: status, Rounds: rounds}}, VFX: vfx, Color: color}
}

func aoe(name, dice, dmg, save string, w int, vfx, color, status string, rounds int) CurseMove {
	return CurseMove{Name: name, Weight: w, Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: dice, DmgType: dmg, Save: save, Half: true, Status: status, Rounds: rounds}}, VFX: vfx, Color: color}
}

var Bestiary = map[string]CurseTemplate{}

func curse(c CurseTemplate) {
	c.Grade = GradeNames[c.Rank]
	Bestiary[c.ID] = c
}

func init() {
	// Grade 4
	curse(CurseTemplate{ID: "fly_head", Name: "Fly Head", Rank: 0, XP: 25, HP: "2d6", AC: 11, Init: 3, ToHit: 3, Save: 0, Tags: []string{"any", "city", "school"},
		Moves:  []CurseMove{hit("Bite", "1d6+1", "piercing", 3, "bite", "#a7c957"), hitS("Bile Spit", "1d4", "acid", 1, "spit", "#a7c957", "weakened", 1)},
		Sprite: SpriteSpec{Shape: "head", Color: "#6a994e", Color2: "#bc4749", Eyes: 3, Mouth: "grin", Size: 1, Extra: "wings"}, Yen: "1d6*100", Desc: "The lowest of curses, born from everyday disgust."})
	curse(CurseTemplate{ID: "crawler", Name: "School Crawler", Rank: 0, XP: 35, HP: "3d6", AC: 12, Init: 3, ToHit: 4, Tags: []string{"school", "hospital", "any"},
		Moves:  []CurseMove{hit("Pounce", "1d8+2", "slashing", 3, "claw", "#e9edc9"), hit("Raking Swipe", "1d6+2", "slashing", 2, "claw", "#e9edc9")},
		Sprite: SpriteSpec{Shape: "insect", Color: "#e9edc9", Color2: "#9a031e", Eyes: 5, Mouth: "grin", Size: 1, Extra: "arms"}, Yen: "1d6*200", Desc: "Fear of exams and hallways at night, given legs."})
	curse(CurseTemplate{ID: "hanako", Name: "Toilet Hanako", Rank: 0, XP: 40, HP: "3d8", AC: 12, Init: 2, ToHit: 4, Tags: []string{"school"},
		Moves:  []CurseMove{hitS("Drag Under", "1d8+2", "bludgeoning", 2, "hands", "#90e0ef", "weakened", 1), hit("Wet Grasp", "1d6+2", "cold", 2, "hands", "#90e0ef")},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#f8f9fa", Color2: "#c1121f", Eyes: 2, Mouth: "none", Size: 1, Extra: "veil"}, Yen: "2d6*100", Desc: "A rumor every student has heard. \"Hanako-san, are you there?\""})
	curse(CurseTemplate{ID: "eye_cluster", Name: "Eye Cluster", Rank: 0, XP: 30, HP: "2d8", AC: 10, Init: 0, ToHit: 3, Tags: []string{"any", "city"},
		Moves:  []CurseMove{hitS("Stare", "1d6", "psychic", 2, "stare", "#ffbe0b", "weakened", 1), hit("Lash", "1d6+1", "bludgeoning", 2, "lash", "#8338ec")},
		Sprite: SpriteSpec{Shape: "blob", Color: "#8338ec", Color2: "#ffbe0b", Eyes: 7, Mouth: "none", Size: 1, Extra: "tentacles"}, Yen: "1d6*100", Desc: "A lump of watching. It never blinks."})
	// Grade 3
	curse(CurseTemplate{ID: "grasshopper", Name: "Grasshopper Curse", Rank: 1, XP: 100, HP: "5d8", AC: 13, Init: 3, ToHit: 5, Save: 1, Tags: []string{"forest", "field", "any"},
		Moves:  []CurseMove{hit("Scythe Arms", "2d6+3", "slashing", 3, "slash", "#55a630"), hitS("Leap Kick", "1d10+3", "bludgeoning", 1, "punch", "#55a630", "exposed", 1)},
		Sprite: SpriteSpec{Shape: "insect", Color: "#55a630", Color2: "#d4d700", Eyes: 2, Mouth: "beak", Size: 2, Extra: "arms"}, Yen: "2d6*500", Desc: "Fear of plagues of insects. Fast and vicious."})
	curse(CurseTemplate{ID: "teke_teke", Name: "Teke-Teke", Rank: 1, XP: 100, HP: "4d8", AC: 14, Init: 5, ToHit: 5, Save: 1, Tags: []string{"city", "tunnel", "school"},
		Moves:  []CurseMove{hit("Scuttling Slash", "2d6+3", "slashing", 3, "claw", "#d00000"), hitS("Clatter", "1d6", "psychic", 1, "stare", "#d00000", "weakened", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#ced4da", Color2: "#9d0208", Eyes: 2, Mouth: "grin", Size: 1, Extra: "halfbody"}, Yen: "2d6*500", Desc: "Half a girl, dragging herself on her elbows faster than you can run."})
	curse(CurseTemplate{ID: "bloat_toad", Name: "Bloat Toad", Rank: 1, XP: 110, HP: "6d10", AC: 11, Init: -1, ToHit: 5, Save: 2, Tags: []string{"water", "field", "forest"},
		Moves:  []CurseMove{aoe("Belly Flop", "2d8", "bludgeoning", "DEX", 2, "slam", "#6c584c", "", 0), hit("Tongue", "1d10+3", "bludgeoning", 2, "tongue", "#e07a5f")},
		Sprite: SpriteSpec{Shape: "beast", Color: "#606c38", Color2: "#e07a5f", Eyes: 2, Mouth: "maw", Size: 2, Extra: "hands"}, Yen: "2d6*500", Desc: "A cart-sized toad with human hands for feet."})
	curse(CurseTemplate{ID: "veiled_bride", Name: "Veiled Bride", Rank: 1, XP: 120, HP: "5d8", AC: 13, Init: 2, ToHit: 5, Save: 2, Tags: []string{"grave", "shrine", "school"},
		Moves:  []CurseMove{hitS("Grasp", "2d6+2", "cold", 2, "hands", "#f8f9fa", "weakened", 1), aoe("Wail", "2d6", "psychic", "WIS", 1, "wail", "#e5e5e5", "", 0)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#f8f9fa", Color2: "#212529", Eyes: 2, Mouth: "grin", Size: 2, Extra: "veil"}, Yen: "3d6*500", Desc: "A jilted bride that never left the chapel."})
	curse(CurseTemplate{ID: "grave_hound", Name: "Grave Hound", Rank: 1, XP: 90, HP: "4d8", AC: 13, Init: 4, ToHit: 5, Save: 1, Tags: []string{"grave", "forest", "field"},
		Moves:  []CurseMove{hit("Savage Bite", "2d6+2", "piercing", 3, "bite", "#9d0208"), hitS("Howl", "1d4", "psychic", 1, "wail", "#ff8800", "weakened", 1)},
		Sprite: SpriteSpec{Shape: "beast", Color: "#9d0208", Color2: "#e9ecef", Eyes: 0, Mouth: "maw", Size: 1, Extra: "ribs"}, Yen: "2d6*400", Desc: "Flayed, eyeless, hunting in packs."})
	// Grade 2
	curse(CurseTemplate{ID: "kuchisake", Name: "Slit-Mouthed Woman", Rank: 2, XP: 300, HP: "8d8", AC: 14, Init: 4, ToHit: 6, Save: 3, Tags: []string{"city", "tunnel", "any"},
		Moves:  []CurseMove{hit("Giant Scissors", "2d10+4", "slashing", 3, "scissors", "#adb5bd"), hitS("\"Am I pretty?\"", "1d8", "psychic", 1, "stare", "#ffccd5", "stun", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#ffe5ec", Color2: "#9d0208", Eyes: 2, Mouth: "grin", Size: 2, Extra: "scissors"}, Yen: "4d6*1000", Desc: "\"Am I pretty?\" There is no right answer."})
	curse(CurseTemplate{ID: "train_ghost", Name: "Last Train Phantom", Rank: 2, XP: 320, HP: "9d8", AC: 13, Init: 3, ToHit: 6, Save: 3, Tags: []string{"tunnel", "city"},
		Moves:  []CurseMove{aoe("Rush Through", "3d8", "bludgeoning", "DEX", 2, "rush", "#ffd60a", "", 0), hit("Door Crush", "2d8+4", "bludgeoning", 2, "doors", "#adb5bd")},
		Sprite: SpriteSpec{Shape: "serpent", Color: "#495057", Color2: "#ffd60a", Eyes: 6, Mouth: "maw", Size: 3, Extra: "windows"}, Yen: "4d6*1000", Desc: "A train that departs at 0:00 and never arrives."})
	curse(CurseTemplate{ID: "cursed_corpse", Name: "Cursed Corpse", Rank: 2, XP: 280, HP: "8d8", AC: 15, Init: 1, ToHit: 6, Save: 2, Tags: []string{"any", "school", "factory"},
		Moves:  []CurseMove{hit("Stitched Fist", "2d8+4", "bludgeoning", 3, "punch", "#dda15e"), hitS("Core Burst", "2d6", "force", 1, "explode", "#dda15e", "exposed", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#dda15e", Color2: "#283618", Eyes: 2, Mouth: "stitched", Size: 2, Extra: "stitches"}, Yen: "3d6*1000", Desc: "A puppet stuffed with a curse. Someone made this."})
	curse(CurseTemplate{ID: "curse_user", Name: "Curse User", Rank: 2, XP: 300, HP: "8d8", AC: 14, Init: 2, ToHit: 6, Save: 3, Human: true, Tags: []string{"city", "any", "factory"},
		Moves:  []CurseMove{hit("Cursed Knife", "2d6+4", "slashing", 3, "slash", "#ffffff"), hitS("Hex", "2d8", "necrotic", 2, "soul", "#7209b7", "weakened", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#343a40", Color2: "#7209b7", Eyes: 2, Mouth: "none", Size: 1, Extra: "hood"}, Yen: "6d6*1000", Desc: "A sorcerer who uses jujutsu for money, revenge or worse."})
	curse(CurseTemplate{ID: "rainbow_serpent", Name: "Rainbow Serpent", Rank: 2, XP: 350, HP: "10d8", AC: 15, Init: 2, ToHit: 6, Save: 3, Tags: []string{"water", "shrine", "forest"},
		Moves:  []CurseMove{hit("Constrict", "2d10+4", "bludgeoning", 3, "bite", "#ff006e"), aoe("Prismatic Breath", "3d8", "radiant", "DEX", 1, "beam", "#3a86ff", "", 0)},
		Sprite: SpriteSpec{Shape: "serpent", Color: "#ff006e", Color2: "#3a86ff", Eyes: 2, Mouth: "maw", Size: 2, Extra: "rainbow"}, Yen: "5d6*1000", Desc: "A shrine's forgotten guardian, gone bad."})
	// Grade 1
	curse(CurseTemplate{ID: "earth_spider", Name: "Tsuchigumo", Rank: 3, XP: 900, HP: "13d10", AC: 16, Init: 2, ToHit: 8, Save: 4, Tags: []string{"forest", "tunnel", "shrine"},
		Moves:  []CurseMove{hit("Venom Fang", "3d8+5", "poison", 3, "bite", "#80b918"), aoe("Web Storm", "3d8", "bludgeoning", "DEX", 1, "web", "#f8f9fa", "stun", 1)},
		Sprite: SpriteSpec{Shape: "insect", Color: "#3c3c3c", Color2: "#80b918", Eyes: 8, Mouth: "maw", Size: 3, Extra: "legs"}, Yen: "6d6*3000", Desc: "An earth spider the size of a house."})
	curse(CurseTemplate{ID: "finger_bearer", Name: "Finger Bearer", Rank: 3, XP: 1100, HP: "14d10", AC: 15, Init: 1, ToHit: 8, Save: 4, Boss: true, Tags: []string{"school", "hospital"},
		Moves:  []CurseMove{hit("Crushing Palm", "3d10+5", "bludgeoning", 3, "slam", "#7b2cbf"), aoe("Cursed Roar", "4d8", "psychic", "WIS", 1, "wail", "#9d4edd", "weakened", 1)},
		Sprite: SpriteSpec{Shape: "giant", Color: "#5a189a", Color2: "#ff9e00", Eyes: 5, Mouth: "maw", Size: 3, Extra: "arms"}, Yen: "8d6*3000", Desc: "A curse that swallowed one of Sukuna's fingers and grew fat on it."})
	curse(CurseTemplate{ID: "kenjaku_puppet", Name: "Q Cultist Enforcer", Rank: 3, XP: 900, HP: "12d8", AC: 16, Init: 3, ToHit: 8, Save: 4, Human: true, Tags: []string{"city", "factory", "any"},
		Moves:  []CurseMove{hit("Cursed Spear", "3d8+5", "piercing", 3, "spear", "#ffffff"), aoe("Grenade Tag", "4d6", "fire", "DEX", 1, "explode", "#fb8500", "burn", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#212529", Color2: "#ffb703", Eyes: 2, Mouth: "none", Size: 2, Extra: "mask"}, Yen: "10d6*1000", Desc: "A curse-user cult enforcer."})
	curse(CurseTemplate{ID: "smallpox", Name: "Smallpox Deity", Rank: 4, XP: 1800, HP: "16d10", AC: 16, Init: 0, ToHit: 8, Save: 5, Boss: true, Domain: "graveyard_domain", Tags: []string{"grave", "hospital", "shrine"},
		Moves:  []CurseMove{hitS("Pustule Burst", "3d10+5", "poison", 3, "spit", "#d4a373", "poison", 2), aoe("Graveyard Bell", "4d8", "necrotic", "CON", 1, "wail", "#6c757d", "weakened", 1)},
		Sprite: SpriteSpec{Shape: "giant", Color: "#d4a373", Color2: "#495057", Eyes: 3, Mouth: "maw", Size: 3, Extra: "coffin"}, Yen: "10d6*3000", Desc: "A vengeful god of disease. Its domain buries you alive."})
	// Special grades (Disaster Curses etc.)
	curse(CurseTemplate{ID: "jogo", Name: "Jogo", Rank: 5, XP: 3500, HP: "20d10", AC: 17, Init: 3, ToHit: 9, Save: 7, Boss: true, Domain: "iron_mountain", Tags: []string{"special"},
		Moves:  []CurseMove{aoe("Ember Insects", "4d8", "fire", "DEX", 2, "embers", "#fcbf49", "burn", 2), hitS("Volcanic Fist", "4d10+5", "fire", 2, "fire_punch", "#d62828", "burn", 2), aoe("Maximum: Meteor", "8d8", "fire", "DEX", 1, "meteor", "#ff5400", "burn", 2)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#e9e5d0", Color2: "#d00000", Eyes: 1, Mouth: "grin", Size: 2, Extra: "volcano"}, Yen: "20d6*5000", Desc: "A Disaster Curse born from the fear of the earth. Short-tempered, terribly strong."})
	curse(CurseTemplate{ID: "hanami", Name: "Hanami", Rank: 5, XP: 3200, HP: "22d10", AC: 18, Init: 1, ToHit: 8, Save: 7, Boss: true, Domain: "ceremonial_sea", Tags: []string{"special"},
		Moves:  []CurseMove{hitS("Cursed Buds", "3d10", "necrotic", 2, "flowers", "#80b918", "weakened", 2), aoe("Root Wave", "5d8", "bludgeoning", "STR", 2, "roots", "#606c38", "stun", 1), hit("Wooden Ball", "4d10+5", "bludgeoning", 1, "slam", "#7f5539")},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#e9e5d0", Color2: "#3a5a40", Eyes: 2, Mouth: "none", Size: 3, Extra: "flowers"}, Yen: "20d6*5000", Desc: "A Disaster Curse of the forest. Calm, polite, and willing to wipe out humanity for the planet."})
	curse(CurseTemplate{ID: "dagon", Name: "Dagon", Rank: 5, XP: 3200, HP: "22d10", AC: 16, Init: 1, ToHit: 8, Save: 6, Boss: true, Domain: "horizon_skandha", Tags: []string{"special"},
		Moves:  []CurseMove{aoe("Death Swarm", "5d8", "piercing", "DEX", 3, "fish", "#0077b6", "bleed", 2), hit("Tidal Slam", "4d10+5", "bludgeoning", 1, "flood", "#48cae4")},
		Sprite: SpriteSpec{Shape: "blob", Color: "#c9184a", Color2: "#0077b6", Eyes: 2, Mouth: "maw", Size: 3, Extra: "tentacles"}, Yen: "20d6*5000", Desc: "A Disaster Curse of the sea, ripening into something terrible."})
	curse(CurseTemplate{ID: "mahito", Name: "Mahito", Rank: 5, XP: 4000, HP: "20d10", AC: 17, Init: 4, ToHit: 9, Save: 7, Boss: true, Domain: "self_embodiment", Escapes: true, Tags: []string{"special"},
		Moves:  []CurseMove{hitS("Idle Transfiguration", "4d10+5", "necrotic", 3, "soul", "#90e0ef", "weakened", 2), aoe("Transfigured Horde", "4d8", "bludgeoning", "DEX", 1, "puppets", "#adb5bd", "", 0), hit("Reshaped Blade Arm", "3d10+5", "slashing", 2, "slash", "#caf0f8")},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#bde0fe", Color2: "#457b9d", Eyes: 2, Mouth: "stitched", Size: 2, Extra: "stitches"}, Yen: "25d6*5000", Desc: "A curse born from humans' hatred of humans. He plays with souls like clay."})
	curse(CurseTemplate{ID: "kenjaku", Name: "Kenjaku (in Geto's body)", Rank: 5, XP: 6000, HP: "26d10", AC: 18, Init: 3, ToHit: 10, Save: 8, Boss: true, Human: true, Domain: "womb_profusion", Tags: []string{"special"},
		Moves:  []CurseMove{aoe("Uzumaki", "6d10", "force", "DEX", 2, "purple", "#7209b7", "", 0), hit("Curse Manipulation: Summon", "4d10+6", "slashing", 2, "summon", "#3c096c"), aoe("Gravity", "5d8", "bludgeoning", "STR", 1, "blue", "#560bad", "stun", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#212529", Color2: "#f8f9fa", Eyes: 2, Mouth: "stitched", Size: 2, Extra: "robe"}, Yen: "40d6*5000", Desc: "A thousand-year-old sorcerer wearing a dead friend's body. The stitches across his forehead never heal."})
	curse(CurseTemplate{ID: "sukuna", Name: "Ryomen Sukuna", Rank: 5, XP: 9000, HP: "30d10", AC: 19, Init: 5, ToHit: 11, Save: 9, Boss: true, Human: true, Domain: "malevolent_shrine", Tags: []string{"special"},
		Moves:  []CurseMove{hit("Dismantle", "5d10+6", "slashing", 3, "slash", "#ffffff"), aoe("Cleave Storm", "6d10", "slashing", "DEX", 2, "cleave", "#ff4d6d", "", 0), aoe("Fuga", "8d10", "fire", "DEX", 1, "fire_arrow", "#ff7b00", "burn", 2)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#f2e9e4", Color2: "#9d0208", Eyes: 4, Mouth: "grin", Size: 2, Extra: "marks"}, Yen: "50d6*5000", Desc: "The King of Curses."})
	curse(CurseTemplate{ID: "transfigured", Name: "Transfigured Human", Rank: 1, XP: 80, HP: "4d8", AC: 12, Init: 1, ToHit: 5, Save: 1, Tags: []string{"mahito"},
		Moves:  []CurseMove{hit("Twisted Grab", "2d6+2", "bludgeoning", 3, "hands", "#adb5bd")},
		Sprite: SpriteSpec{Shape: "blob", Color: "#adb5bd", Color2: "#6c757d", Eyes: 3, Mouth: "stitched", Size: 1, Extra: "arms"}, Yen: "", Desc: "Once a person. Mahito made it this way."})
	curse(CurseTemplate{ID: "kyoto_student", Name: "Kyoto Student", Rank: 2, XP: 250, HP: "8d8", AC: 15, Init: 3, ToHit: 6, Save: 3, Human: true, Tags: []string{"kyoto"},
		Moves:  []CurseMove{hit("Technique Strike", "2d8+4", "force", 3, "punch", "#ffffff"), hitS("Puppet Shot", "2d6+3", "piercing", 1, "bullet", "#ffd166", "exposed", 1)},
		Sprite: SpriteSpec{Shape: "humanoid", Color: "#14213d", Color2: "#fca311", Eyes: 2, Mouth: "none", Size: 1, Extra: "uniform"}, Yen: "", Desc: "A rival from Kyoto Jujutsu High. Not evil — just very competitive."})

	// Curse domains (cast once when bloodied).
	addAbility(Ability{ID: "graveyard_domain", Name: "Domain Expansion: Graveyard", Domain: true, Rounds: 2, Desc: "Coffins close over every enemy: 4d8 and Stunned unless they save (CON).",
		Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "4d8", DmgType: "necrotic", Save: "CON", Status: "stun", Rounds: 1}}, VFX: "domain_grave", Color: "#495057"})
	addAbility(Ability{ID: "ceremonial_sea", Name: "Domain Expansion: Ceremonial Sea of Light", Domain: true, Rounds: 2, Desc: "Flowers drain the will to fight: 3d8 and Weakened.",
		Effects: []Effect{{Kind: "auto", Target: "all_enemies", Dice: "3d8", DmgType: "necrotic", Status: "weakened", Rounds: 1}}, VFX: "domain_flowers", Color: "#80b918"})
	addAbility(Ability{ID: "horizon_skandha", Name: "Domain Expansion: Horizon of the Captivating Skandha", Domain: true, Rounds: 2, Desc: "An endless beach and infinite shikigami fish: 5d8, no save.",
		Effects: []Effect{{Kind: "auto", Target: "all_enemies", Dice: "5d8", DmgType: "piercing", Status: "bleed", Rounds: 1}}, VFX: "domain_sea", Color: "#0096c7"})
	addAbility(Ability{ID: "womb_profusion", Name: "Domain Expansion: Womb Profusion", Domain: true, Rounds: 2, Desc: "Kenjaku's domain: 5d10 and Stunned (WIS save).",
		Effects: []Effect{{Kind: "save", Target: "all_enemies", Dice: "5d10", DmgType: "psychic", Save: "WIS", Status: "stun", Rounds: 1}}, VFX: "domain_womb", Color: "#3c096c"})
}

// SortedCurseIDs lists bestiary ids from weakest to strongest.
func SortedCurseIDs() []string {
	ids := make([]string, 0, len(Bestiary))
	for id := range Bestiary {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := Bestiary[ids[i]], Bestiary[ids[j]]
		if a.Rank != b.Rank {
			return a.Rank < b.Rank
		}
		if a.XP != b.XP {
			return a.XP < b.XP
		}
		return a.ID < b.ID
	})
	return ids
}

// XPThresholds[level-1] is the XP needed to reach that level.
var XPThresholds = []int{0, 300, 900, 2700, 6500, 14000, 23000, 34000, 48000, 64000, 85000, 100000}

const MaxLevel = 12

func ProfBonus(level int) int { return 2 + (level-1)/4 }

func Mod(score int) int {
	if score >= 10 {
		return (score - 10) / 2
	}
	return -((11 - score) / 2)
}

// SorcererGrade maps character level to an official sorcerer grade.
func SorcererGrade(level int) string {
	switch {
	case level >= 11:
		return "Special Grade"
	case level >= 9:
		return "Grade 1"
	case level >= 7:
		return "Semi-Grade 1"
	case level >= 5:
		return "Grade 2"
	case level >= 3:
		return "Grade 3"
	}
	return "Grade 4"
}

// EncounterBudget is the XP budget for a medium fight per character, by level.
var EncounterBudget = []int{50, 100, 150, 250, 500, 600, 750, 900, 1100, 1200, 1600, 2000}
