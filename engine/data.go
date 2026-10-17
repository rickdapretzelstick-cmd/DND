package engine

import "sort"

// Abilities in display order.
var Abilities = []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}

// Skills maps each skill to the ability it uses.
var Skills = map[string]string{
	"athletics": "STR", "acrobatics": "DEX", "sleight_of_hand": "DEX", "stealth": "DEX",
	"arcana": "INT", "history": "INT", "investigation": "INT", "nature": "INT", "religion": "INT",
	"animal_handling": "WIS", "insight": "WIS", "medicine": "WIS", "perception": "WIS", "survival": "WIS",
	"deception": "CHA", "intimidation": "CHA", "performance": "CHA", "persuasion": "CHA",
}

type Race struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Bonus  map[string]int `json:"bonus"`
	Speed  int            `json:"speed"`
	Traits string         `json:"traits"`
}

var Races = map[string]Race{
	"human":      {"human", "Human", map[string]int{"STR": 1, "DEX": 1, "CON": 1, "INT": 1, "WIS": 1, "CHA": 1}, 30, "Versatile and ambitious."},
	"elf":        {"elf", "Elf", map[string]int{"DEX": 2, "WIS": 1}, 30, "Keen senses (proficient in Perception), darkvision."},
	"dwarf":      {"dwarf", "Dwarf", map[string]int{"CON": 2, "WIS": 1}, 25, "Tough (+1 HP per level), darkvision."},
	"halfling":   {"halfling", "Halfling", map[string]int{"DEX": 2, "CHA": 1}, 25, "Lucky: reroll natural 1s on attacks and checks."},
	"half-orc":   {"half-orc", "Half-Orc", map[string]int{"STR": 2, "CON": 1}, 30, "Relentless: drop to 1 HP instead of 0 once per long rest."},
	"tiefling":   {"tiefling", "Tiefling", map[string]int{"CHA": 2, "INT": 1}, 30, "Resistant to fire, darkvision."},
	"dragonborn": {"dragonborn", "Dragonborn", map[string]int{"STR": 2, "CHA": 1}, 30, "Draconic heritage, imposing presence."},
}

// Feature IDs a class can use in play. The engine implements each one.
const (
	FSecondWind   = "second_wind"
	FActionSurge  = "action_surge"
	FRage         = "rage"
	FSneakAttack  = "sneak_attack"
	FFireBolt     = "fire_bolt"
	FMagicMissile = "magic_missile"
	FSacredFlame  = "sacred_flame"
	FCureWounds   = "cure_wounds"
	FHuntersMark  = "hunters_mark"
	FHealingWord  = "healing_word"
	FViciousMock  = "vicious_mockery"
	FLayOnHands   = "lay_on_hands"
	FDivineSmite  = "divine_smite"
)

type Ability struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Desc    string `json:"desc"`
	Target  string `json:"target"` // "enemy", "ally", "self"
	Cost    string `json:"cost"`   // "slot", "use", "free" (cantrip/passive)
	Bonus   bool   `json:"bonus"`  // bonus action
	Passive bool   `json:"passive"`
}

var AbilityList = map[string]Ability{
	FSecondWind:   {FSecondWind, "Second Wind", "Bonus action: heal 1d10 + level. Once per short rest.", "self", "use", true, false},
	FActionSurge:  {FActionSurge, "Action Surge", "Take one extra action this turn. Once per short rest.", "self", "use", true, false},
	FRage:         {FRage, "Rage", "Bonus action: +2 melee damage and take half damage until combat ends.", "self", "use", true, false},
	FSneakAttack:  {FSneakAttack, "Sneak Attack", "Once per turn, a hit deals extra (level+1)/2 d6 damage.", "self", "free", false, true},
	FFireBolt:     {FFireBolt, "Fire Bolt", "Cantrip: ranged spell attack, 1d10 fire (2d10 at level 5).", "enemy", "free", false, false},
	FMagicMissile: {FMagicMissile, "Magic Missile", "1st-level spell: three darts that never miss, 1d4+1 each.", "enemy", "slot", false, false},
	FSacredFlame:  {FSacredFlame, "Sacred Flame", "Cantrip: target makes a DEX save or takes 1d8 radiant (2d8 at level 5).", "enemy", "free", false, false},
	FCureWounds:   {FCureWounds, "Cure Wounds", "1st-level spell: touch an ally to heal 1d8 + WIS.", "ally", "slot", false, false},
	FHealingWord:  {FHealingWord, "Healing Word", "1st-level spell, bonus action: heal an ally 1d4 + CHA at range.", "ally", "slot", true, false},
	FViciousMock:  {FViciousMock, "Vicious Mockery", "Cantrip: WIS save or 1d4 psychic and disadvantage on its next attack.", "enemy", "free", false, false},
	FHuntersMark:  {FHuntersMark, "Hunter's Mark", "Spell, bonus action: mark an enemy; your hits deal +1d6 to it.", "enemy", "slot", true, false},
	FLayOnHands:   {FLayOnHands, "Lay on Hands", "Heal an ally from a pool of 5 x level HP (uses up to 10 per touch).", "ally", "use", false, false},
	FDivineSmite:  {FDivineSmite, "Divine Smite", "Spend a slot on your next melee hit for +2d8 radiant.", "self", "slot", true, false},
}

type Class struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	HitDie     int      `json:"hitDie"`
	Priority   []string `json:"priority"` // ability score priority for auto-assignment
	Skills     []string `json:"skills"`
	Weapons    []string `json:"weapons"`
	Armor      string   `json:"armor"`
	Shield     bool     `json:"shield"`
	Features   []string `json:"features"`
	CastStat   string   `json:"castStat"`
	SlotsAt1   int      `json:"slotsAt1"`
	ExtraItems []string `json:"extraItems"`
	Blurb      string   `json:"blurb"`
}

var Classes = map[string]Class{
	"fighter": {"fighter", "Fighter", 10, []string{"STR", "CON", "DEX", "WIS", "CHA", "INT"},
		[]string{"athletics", "intimidation", "perception"}, []string{"longsword", "handaxe"}, "chain_mail", true,
		[]string{FSecondWind, FActionSurge}, "", 0, []string{"potion_healing"}, "A master of weapons and armor. Tough and reliable."},
	"barbarian": {"barbarian", "Barbarian", 12, []string{"STR", "CON", "DEX", "WIS", "CHA", "INT"},
		[]string{"athletics", "survival", "intimidation"}, []string{"greataxe", "handaxe"}, "unarmored_barb", false,
		[]string{FRage}, "", 0, []string{"potion_healing"}, "A furious warrior who shrugs off blows while raging."},
	"rogue": {"rogue", "Rogue", 8, []string{"DEX", "CON", "INT", "WIS", "CHA", "STR"},
		[]string{"stealth", "sleight_of_hand", "acrobatics", "deception", "investigation", "perception"}, []string{"shortsword", "shortbow"}, "leather", false,
		[]string{FSneakAttack}, "", 0, []string{"thieves_tools", "potion_healing"}, "Sneaky and precise. Huge damage once per turn."},
	"ranger": {"ranger", "Ranger", 10, []string{"DEX", "WIS", "CON", "STR", "INT", "CHA"},
		[]string{"survival", "nature", "stealth", "perception"}, []string{"longbow", "shortsword"}, "studded_leather", false,
		[]string{FHuntersMark}, "WIS", 1, []string{"potion_healing"}, "A hunter and tracker deadly with a bow."},
	"wizard": {"wizard", "Wizard", 6, []string{"INT", "CON", "DEX", "WIS", "CHA", "STR"},
		[]string{"arcana", "history", "investigation"}, []string{"quarterstaff"}, "none", false,
		[]string{FFireBolt, FMagicMissile}, "INT", 2, []string{"spellbook", "potion_healing"}, "A scholar of magic. Fragile but powerful at range."},
	"cleric": {"cleric", "Cleric", 8, []string{"WIS", "CON", "STR", "DEX", "CHA", "INT"},
		[]string{"religion", "medicine", "insight"}, []string{"mace"}, "chain_shirt", true,
		[]string{FSacredFlame, FCureWounds}, "WIS", 2, []string{"holy_symbol"}, "A divine healer who can also hold the front line."},
	"bard": {"bard", "Bard", 8, []string{"CHA", "DEX", "CON", "WIS", "INT", "STR"},
		[]string{"performance", "persuasion", "deception", "insight"}, []string{"rapier", "dagger"}, "leather", false,
		[]string{FViciousMock, FHealingWord}, "CHA", 2, []string{"lute", "potion_healing"}, "A silver-tongued performer whose words wound and heal."},
	"paladin": {"paladin", "Paladin", 10, []string{"STR", "CHA", "CON", "WIS", "DEX", "INT"},
		[]string{"athletics", "persuasion", "religion"}, []string{"longsword", "javelin"}, "chain_mail", true,
		[]string{FLayOnHands, FDivineSmite}, "CHA", 1, []string{"holy_symbol"}, "A holy knight with healing hands and smiting blows."},
}

type Item struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"` // weapon, armor, shield, potion, gear, treasure
	Damage   string `json:"damage,omitempty"`
	DmgType  string `json:"dmgType,omitempty"`
	Finesse  bool   `json:"finesse,omitempty"`
	Ranged   bool   `json:"ranged,omitempty"`
	BaseAC   int    `json:"baseAC,omitempty"`
	MaxDex   int    `json:"maxDex,omitempty"` // -1 = unlimited
	Heal     string `json:"heal,omitempty"`
	Bonus    int    `json:"bonus,omitempty"` // magic +X
	Value    int    `json:"value"`
	Desc     string `json:"desc,omitempty"`
	Qty      int    `json:"qty"`
	Custom   bool   `json:"custom,omitempty"`
	Equipped bool   `json:"equipped,omitempty"`
}

var ItemList = map[string]Item{
	"dagger":          {ID: "dagger", Name: "Dagger", Kind: "weapon", Damage: "1d4", DmgType: "piercing", Finesse: true, Value: 2},
	"shortsword":      {ID: "shortsword", Name: "Shortsword", Kind: "weapon", Damage: "1d6", DmgType: "piercing", Finesse: true, Value: 10},
	"rapier":          {ID: "rapier", Name: "Rapier", Kind: "weapon", Damage: "1d8", DmgType: "piercing", Finesse: true, Value: 25},
	"longsword":       {ID: "longsword", Name: "Longsword", Kind: "weapon", Damage: "1d8", DmgType: "slashing", Value: 15},
	"greataxe":        {ID: "greataxe", Name: "Greataxe", Kind: "weapon", Damage: "1d12", DmgType: "slashing", Value: 30},
	"greatsword":      {ID: "greatsword", Name: "Greatsword", Kind: "weapon", Damage: "2d6", DmgType: "slashing", Value: 50},
	"handaxe":         {ID: "handaxe", Name: "Handaxe", Kind: "weapon", Damage: "1d6", DmgType: "slashing", Value: 5},
	"mace":            {ID: "mace", Name: "Mace", Kind: "weapon", Damage: "1d6", DmgType: "bludgeoning", Value: 5},
	"warhammer":       {ID: "warhammer", Name: "Warhammer", Kind: "weapon", Damage: "1d8", DmgType: "bludgeoning", Value: 15},
	"quarterstaff":    {ID: "quarterstaff", Name: "Quarterstaff", Kind: "weapon", Damage: "1d6", DmgType: "bludgeoning", Value: 1},
	"javelin":         {ID: "javelin", Name: "Javelin", Kind: "weapon", Damage: "1d6", DmgType: "piercing", Value: 1},
	"shortbow":        {ID: "shortbow", Name: "Shortbow", Kind: "weapon", Damage: "1d6", DmgType: "piercing", Ranged: true, Value: 25},
	"longbow":         {ID: "longbow", Name: "Longbow", Kind: "weapon", Damage: "1d8", DmgType: "piercing", Ranged: true, Value: 50},
	"none":            {ID: "none", Name: "Clothes", Kind: "armor", BaseAC: 10, MaxDex: -1},
	"unarmored_barb":  {ID: "unarmored_barb", Name: "Unarmored Defense", Kind: "armor", BaseAC: 10, MaxDex: -1, Desc: "AC 10 + DEX + CON"},
	"leather":         {ID: "leather", Name: "Leather Armor", Kind: "armor", BaseAC: 11, MaxDex: -1, Value: 10},
	"studded_leather": {ID: "studded_leather", Name: "Studded Leather", Kind: "armor", BaseAC: 12, MaxDex: -1, Value: 45},
	"chain_shirt":     {ID: "chain_shirt", Name: "Chain Shirt", Kind: "armor", BaseAC: 13, MaxDex: 2, Value: 50},
	"half_plate":      {ID: "half_plate", Name: "Half Plate", Kind: "armor", BaseAC: 15, MaxDex: 2, Value: 750},
	"chain_mail":      {ID: "chain_mail", Name: "Chain Mail", Kind: "armor", BaseAC: 16, MaxDex: 0, Value: 75},
	"plate":           {ID: "plate", Name: "Plate Armor", Kind: "armor", BaseAC: 18, MaxDex: 0, Value: 1500},
	"shield":          {ID: "shield", Name: "Shield", Kind: "shield", BaseAC: 2, Value: 10},
	"potion_healing":  {ID: "potion_healing", Name: "Potion of Healing", Kind: "potion", Heal: "2d4+2", Value: 50},
	"potion_greater":  {ID: "potion_greater", Name: "Potion of Greater Healing", Kind: "potion", Heal: "4d4+4", Value: 150},
	"thieves_tools":   {ID: "thieves_tools", Name: "Thieves' Tools", Kind: "gear", Value: 25},
	"spellbook":       {ID: "spellbook", Name: "Spellbook", Kind: "gear", Value: 50},
	"holy_symbol":     {ID: "holy_symbol", Name: "Holy Symbol", Kind: "gear", Value: 5},
	"lute":            {ID: "lute", Name: "Lute", Kind: "gear", Value: 35},
	"rope":            {ID: "rope", Name: "Hempen Rope (50 ft)", Kind: "gear", Value: 1},
	"torch":           {ID: "torch", Name: "Torch", Kind: "gear", Value: 1},
	"rations":         {ID: "rations", Name: "Rations (1 day)", Kind: "gear", Value: 1},
	"longsword_p1":    {ID: "longsword_p1", Name: "+1 Longsword", Kind: "weapon", Damage: "1d8", DmgType: "slashing", Bonus: 1, Value: 500},
	"dagger_p1":       {ID: "dagger_p1", Name: "+1 Dagger", Kind: "weapon", Damage: "1d4", DmgType: "piercing", Finesse: true, Bonus: 1, Value: 400},
	"longbow_p1":      {ID: "longbow_p1", Name: "+1 Longbow", Kind: "weapon", Damage: "1d8", DmgType: "piercing", Ranged: true, Bonus: 1, Value: 500},
	"gem":             {ID: "gem", Name: "Polished Gemstone", Kind: "treasure", Value: 50},
	"silver_idol":     {ID: "silver_idol", Name: "Silver Idol", Kind: "treasure", Value: 100},
}

// NewItem returns a fresh copy of a catalog item with quantity 1.
func NewItem(id string) (Item, bool) {
	it, ok := ItemList[id]
	if !ok {
		return Item{}, false
	}
	it.Qty = 1
	return it, true
}

type MonsterTemplate struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	CR      float64  `json:"cr"`
	XP      int      `json:"xp"`
	HP      string   `json:"hp"`
	AC      int      `json:"ac"`
	DexMod  int      `json:"dexMod"`
	ToHit   int      `json:"toHit"`
	Damage  string   `json:"damage"`
	DmgType string   `json:"dmgType"`
	Attacks int      `json:"attacks"`
	Attack  string   `json:"attack"`
	Saves   int      `json:"saves"` // flat save bonus
	Tags    []string `json:"tags"`
	Gold    string   `json:"gold"`
	Boss    bool     `json:"boss"`
}

var Bestiary = map[string]MonsterTemplate{
	"rat":            {ID: "rat", Name: "Giant Rat", CR: 0.125, XP: 25, HP: "2d6", AC: 12, DexMod: 2, ToHit: 4, Damage: "1d4+2", DmgType: "piercing", Attacks: 1, Attack: "bites", Tags: []string{"beast", "cave", "sewer"}},
	"kobold":         {ID: "kobold", Name: "Kobold", CR: 0.125, XP: 25, HP: "2d6-1", AC: 12, DexMod: 2, ToHit: 4, Damage: "1d4+2", DmgType: "piercing", Attacks: 1, Attack: "stabs with a dagger", Tags: []string{"cave", "humanoid"}, Gold: "1d6"},
	"goblin":         {ID: "goblin", Name: "Goblin", CR: 0.25, XP: 50, HP: "2d6", AC: 15, DexMod: 2, ToHit: 4, Damage: "1d6+2", DmgType: "slashing", Attacks: 1, Attack: "slashes with a scimitar", Tags: []string{"cave", "forest", "humanoid"}, Gold: "2d6"},
	"skeleton":       {ID: "skeleton", Name: "Skeleton", CR: 0.25, XP: 50, HP: "2d8+4", AC: 13, DexMod: 2, ToHit: 4, Damage: "1d6+2", DmgType: "piercing", Attacks: 1, Attack: "hacks with a rusted blade", Tags: []string{"crypt", "undead"}},
	"zombie":         {ID: "zombie", Name: "Zombie", CR: 0.25, XP: 50, HP: "3d8+9", AC: 8, DexMod: -2, ToHit: 3, Damage: "1d6+1", DmgType: "bludgeoning", Attacks: 1, Attack: "slams", Tags: []string{"crypt", "undead"}},
	"wolf":           {ID: "wolf", Name: "Wolf", CR: 0.25, XP: 50, HP: "2d8+2", AC: 13, DexMod: 2, ToHit: 4, Damage: "2d4+2", DmgType: "piercing", Attacks: 1, Attack: "bites", Tags: []string{"forest", "beast"}},
	"bandit":         {ID: "bandit", Name: "Bandit", CR: 0.125, XP: 25, HP: "2d8+2", AC: 12, DexMod: 1, ToHit: 3, Damage: "1d6+1", DmgType: "slashing", Attacks: 1, Attack: "swings a scimitar", Tags: []string{"forest", "road", "humanoid"}, Gold: "3d6"},
	"cultist":        {ID: "cultist", Name: "Cultist", CR: 0.125, XP: 25, HP: "2d8", AC: 12, DexMod: 1, ToHit: 3, Damage: "1d6+1", DmgType: "slashing", Attacks: 1, Attack: "slashes with a ritual blade", Tags: []string{"crypt", "keep", "humanoid"}, Gold: "2d6"},
	"giant_spider":   {ID: "giant_spider", Name: "Giant Spider", CR: 1, XP: 200, HP: "4d10+4", AC: 14, DexMod: 3, ToHit: 5, Damage: "1d8+3", DmgType: "piercing", Attacks: 1, Attack: "bites with venomous fangs", Tags: []string{"cave", "forest", "beast"}},
	"orc":            {ID: "orc", Name: "Orc", CR: 0.5, XP: 100, HP: "2d8+6", AC: 13, DexMod: 1, ToHit: 5, Damage: "1d12+3", DmgType: "slashing", Attacks: 1, Attack: "cleaves with a greataxe", Tags: []string{"cave", "keep", "humanoid"}, Gold: "2d10"},
	"hobgoblin":      {ID: "hobgoblin", Name: "Hobgoblin", CR: 0.5, XP: 100, HP: "2d8+2", AC: 18, DexMod: 1, ToHit: 3, Damage: "1d8+1", DmgType: "slashing", Attacks: 1, Attack: "strikes with a longsword", Tags: []string{"keep", "humanoid"}, Gold: "2d10"},
	"ghoul":          {ID: "ghoul", Name: "Ghoul", CR: 1, XP: 200, HP: "5d8", AC: 12, DexMod: 2, ToHit: 4, Damage: "2d6+2", DmgType: "slashing", Attacks: 1, Attack: "rakes with paralyzing claws", Tags: []string{"crypt", "undead"}},
	"bugbear":        {ID: "bugbear", Name: "Bugbear", CR: 1, XP: 200, HP: "5d8+5", AC: 16, DexMod: 2, ToHit: 4, Damage: "2d8+2", DmgType: "piercing", Attacks: 1, Attack: "smashes with a morningstar", Tags: []string{"cave", "forest", "humanoid"}, Gold: "3d10"},
	"dire_wolf":      {ID: "dire_wolf", Name: "Dire Wolf", CR: 1, XP: 200, HP: "5d10+10", AC: 14, DexMod: 2, ToHit: 5, Damage: "2d6+3", DmgType: "piercing", Attacks: 1, Attack: "savages with its jaws", Tags: []string{"forest", "beast"}},
	"cult_fanatic":   {ID: "cult_fanatic", Name: "Cult Fanatic", CR: 2, XP: 450, HP: "6d8", AC: 13, DexMod: 2, ToHit: 4, Damage: "2d4+2", DmgType: "necrotic", Attacks: 2, Attack: "lashes out with dark magic", Tags: []string{"crypt", "keep", "humanoid"}, Gold: "4d10"},
	"bandit_captain": {ID: "bandit_captain", Name: "Bandit Captain", CR: 2, XP: 450, HP: "10d8+20", AC: 15, DexMod: 3, ToHit: 5, Damage: "1d6+3", DmgType: "slashing", Attacks: 3, Attack: "strikes with scimitar and dagger", Tags: []string{"forest", "road", "humanoid"}, Gold: "10d10", Boss: true},
	"ogre":           {ID: "ogre", Name: "Ogre", CR: 2, XP: 450, HP: "7d10+21", AC: 11, DexMod: -1, ToHit: 6, Damage: "2d8+4", DmgType: "bludgeoning", Attacks: 1, Attack: "swings a massive club", Tags: []string{"cave", "forest", "giant"}, Gold: "4d10", Boss: true},
	"wight":          {ID: "wight", Name: "Wight", CR: 3, XP: 700, HP: "6d8+18", AC: 14, DexMod: 2, ToHit: 4, Damage: "1d8+2", DmgType: "slashing", Attacks: 2, Attack: "strikes with a life-draining blade", Tags: []string{"crypt", "undead"}, Gold: "6d10", Boss: true},
	"owlbear":        {ID: "owlbear", Name: "Owlbear", CR: 3, XP: 700, HP: "7d10+21", AC: 13, DexMod: 1, ToHit: 7, Damage: "1d10+5", DmgType: "slashing", Attacks: 2, Attack: "rends with beak and claws", Tags: []string{"forest", "beast"}, Boss: true},
	"minotaur":       {ID: "minotaur", Name: "Minotaur", CR: 3, XP: 700, HP: "9d10+27", AC: 14, DexMod: 0, ToHit: 6, Damage: "2d12+4", DmgType: "slashing", Attacks: 1, Attack: "gores and swings its greataxe", Tags: []string{"cave", "keep", "monstrosity"}, Gold: "6d10", Boss: true},
	"troll":          {ID: "troll", Name: "Troll", CR: 5, XP: 1800, HP: "8d10+40", AC: 15, DexMod: 1, ToHit: 7, Damage: "2d6+4", DmgType: "slashing", Attacks: 3, Attack: "tears with claws and bite", Tags: []string{"cave", "forest", "giant"}, Gold: "8d10", Boss: true},
	"young_dragon":   {ID: "young_dragon", Name: "Young Green Dragon", CR: 8, XP: 3900, HP: "16d10+48", AC: 18, DexMod: 1, ToHit: 7, Damage: "2d10+4", DmgType: "piercing", Attacks: 3, Attack: "bites and claws", Saves: 4, Tags: []string{"forest", "keep", "dragon"}, Gold: "20d10", Boss: true},
	"lich_acolyte":   {ID: "lich_acolyte", Name: "Necromancer", CR: 4, XP: 1100, HP: "12d8", AC: 12, DexMod: 2, ToHit: 6, Damage: "3d8", DmgType: "necrotic", Attacks: 1, Attack: "hurls a bolt of necrotic energy", Saves: 3, Tags: []string{"crypt", "keep", "humanoid"}, Gold: "8d10", Boss: true},
}

// SortedMonsterIDs lists bestiary ids from weakest to strongest.
func SortedMonsterIDs() []string {
	ids := make([]string, 0, len(Bestiary))
	for id := range Bestiary {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		a, b := Bestiary[ids[i]], Bestiary[ids[j]]
		if a.CR != b.CR {
			return a.CR < b.CR
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

// SlotsFor returns the simplified spell slot pool for a caster at a level.
func SlotsFor(c Class, level int) int {
	if c.SlotsAt1 == 0 {
		return 0
	}
	if c.SlotsAt1 == 1 { // half casters
		return 1 + level/2
	}
	return 2 + level
}
