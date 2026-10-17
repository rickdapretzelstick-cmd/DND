package engine

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

var StandardArray = []int{15, 14, 13, 12, 10, 8}

// Roll4d6 rolls six ability scores, each 4d6 drop the lowest, sorted high to low.
func Roll4d6() []int {
	out := make([]int, 6)
	for i := range out {
		d := []int{Die(6), Die(6), Die(6), Die(6)}
		sort.Ints(d)
		out[i] = d[1] + d[2] + d[3]
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

// AssignByPriority maps sorted-descending scores onto a class's ability priority.
func AssignByPriority(scores []int, class Class) map[string]int {
	s := slices.Clone(scores)
	sort.Sort(sort.Reverse(sort.IntSlice(s)))
	m := map[string]int{}
	for i, ab := range class.Priority {
		m[ab] = s[i]
	}
	return m
}

func sameMultiset(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := slices.Clone(a), slices.Clone(b)
	sort.Ints(x)
	sort.Ints(y)
	return slices.Equal(x, y)
}

func (c *Character) Mod(ab string) int { return Mod(c.Stats[ab]) }

func (c *Character) ClassDef() Class { return Classes[c.Class] }

func (c *Character) Prof() int { return ProfBonus(c.Level) }

func (c *Character) Conscious() bool { return !c.Dead && c.HP > 0 && !c.Fled }

func (c *Character) HasFeature(f string) bool { return slices.Contains(c.Features, f) }

func (c *Character) Equipped(kind string) *Item {
	for i := range c.Inventory {
		if c.Inventory[i].Equipped && c.Inventory[i].Kind == kind {
			return &c.Inventory[i]
		}
	}
	return nil
}

// Weapon returns the equipped weapon, or an unarmed strike.
func (c *Character) Weapon() Item {
	if w := c.Equipped("weapon"); w != nil {
		return *w
	}
	return Item{ID: "unarmed", Name: "Fists", Kind: "weapon", Damage: "1", DmgType: "bludgeoning"}
}

func (c *Character) RecalcAC() {
	dex := c.Mod("DEX")
	ac := 10 + dex
	if a := c.Equipped("armor"); a != nil {
		switch {
		case a.ID == "unarmored_barb":
			ac = 10 + dex + c.Mod("CON")
		case a.MaxDex < 0:
			ac = a.BaseAC + dex
		default:
			ac = a.BaseAC + min(dex, a.MaxDex)
		}
		ac += a.Bonus
	} else if c.Class == "barbarian" {
		ac = 10 + dex + c.Mod("CON")
	}
	if s := c.Equipped("shield"); s != nil {
		ac += s.BaseAC + s.Bonus
	}
	c.AC = ac
}

// AttackStat picks STR or DEX for a weapon.
func (c *Character) AttackStat(w Item) string {
	if w.Ranged {
		return "DEX"
	}
	if w.Finesse && c.Mod("DEX") > c.Mod("STR") {
		return "DEX"
	}
	return "STR"
}

func (c *Character) AttackBonus(w Item) int { return c.Mod(c.AttackStat(w)) + c.Prof() + w.Bonus }

func (c *Character) SpellAttack() int { return c.Mod(c.ClassDef().CastStat) + c.Prof() }

func (c *Character) SpellDC() int { return 8 + c.Mod(c.ClassDef().CastStat) + c.Prof() }

func (c *Character) NumAttacks() int {
	switch c.Class {
	case "fighter":
		if c.Level >= 11 {
			return 3
		}
		if c.Level >= 5 {
			return 2
		}
	case "barbarian", "ranger", "paladin":
		if c.Level >= 5 {
			return 2
		}
	}
	return 1
}

func (c *Character) SkillProficient(skill string) bool {
	if slices.Contains(c.Skills, skill) {
		return true
	}
	return c.Race == "elf" && skill == "perception"
}

// NormalizeSkill turns user/AI text like "Sleight of Hand" or "dex" into a skill id or ability code.
func NormalizeSkill(s string) string {
	t := strings.ToLower(strings.TrimSpace(s))
	t = strings.ReplaceAll(t, " ", "_")
	t = strings.ReplaceAll(t, "-", "_")
	if _, ok := Skills[t]; ok {
		return t
	}
	ab := map[string]string{"str": "STR", "strength": "STR", "dex": "DEX", "dexterity": "DEX", "con": "CON",
		"constitution": "CON", "int": "INT", "intelligence": "INT", "wis": "WIS", "wisdom": "WIS", "cha": "CHA", "charisma": "CHA"}
	if a, ok := ab[t]; ok {
		return a
	}
	return ""
}

// CheckResult describes a resolved d20 test.
type CheckResult struct {
	Name    string
	Label   string
	Roll    int
	Dice    []int
	Bonus   int
	Total   int
	DC      int
	Success bool
	Nat20   bool
	Nat1    bool
}

func (r CheckResult) String() string {
	dice := ""
	if len(r.Dice) > 1 {
		dice = fmt.Sprintf(" (rolled %d and %d)", r.Dice[0], r.Dice[1])
	}
	out := fmt.Sprintf("%s — %s: d20 %d%s %+d = %d", r.Name, r.Label, r.Roll, dice, r.Bonus, r.Total)
	if r.DC > 0 {
		if r.Success {
			out += fmt.Sprintf(" vs DC %d ✔ success", r.DC)
		} else {
			out += fmt.Sprintf(" vs DC %d ✘ failure", r.DC)
		}
	}
	return out
}

func (c *Character) d20(adv int) (int, []int) {
	r, d := D20(adv)
	if c.Race == "halfling" && r == 1 {
		r = Die(20)
		d = append(d, r)
	}
	return r, d
}

// Check rolls an ability check for a skill id or ability code.
func (c *Character) Check(skillOrAbility string, dc, adv int) CheckResult {
	key := NormalizeSkill(skillOrAbility)
	if key == "" {
		key = "WIS"
	}
	ab, label := key, key
	bonus := 0
	if a, ok := Skills[key]; ok {
		ab = a
		label = SkillLabel(key)
		if c.SkillProficient(key) {
			bonus += c.Prof()
		}
	}
	bonus += c.Mod(ab)
	roll, dice := c.d20(adv)
	total := roll + bonus
	return CheckResult{Name: c.Name, Label: label + " check", Roll: roll, Dice: dice, Bonus: bonus, Total: total,
		DC: dc, Success: dc == 0 || total >= dc, Nat20: roll == 20, Nat1: roll == 1}
}

func (c *Character) Save(ab string, dc, adv int) CheckResult {
	if n := NormalizeSkill(ab); len(n) == 3 && n == strings.ToUpper(n) {
		ab = n
	} else {
		ab = "CON"
	}
	bonus := c.Mod(ab)
	cl := c.ClassDef()
	if cl.Priority[0] == ab || cl.Priority[1] == ab {
		bonus += c.Prof()
	}
	roll, dice := c.d20(adv)
	total := roll + bonus
	return CheckResult{Name: c.Name, Label: ab + " save", Roll: roll, Dice: dice, Bonus: bonus, Total: total,
		DC: dc, Success: total >= dc, Nat20: roll == 20, Nat1: roll == 1}
}

func (c *Character) AddItem(it Item) {
	if it.Qty < 1 {
		it.Qty = 1
	}
	for i := range c.Inventory {
		if c.Inventory[i].ID == it.ID && c.Inventory[i].Name == it.Name && (it.Kind == "potion" || it.Kind == "gear" || it.Kind == "treasure") {
			c.Inventory[i].Qty += it.Qty
			return
		}
	}
	c.Inventory = append(c.Inventory, it)
}

// RemoveItem removes qty of the first item whose id or name matches.
func (c *Character) RemoveItem(idOrName string, qty int) (Item, bool) {
	q := strings.ToLower(idOrName)
	for i := range c.Inventory {
		it := c.Inventory[i]
		if strings.ToLower(it.ID) == q || strings.ToLower(it.Name) == q || (q != "" && strings.Contains(strings.ToLower(it.Name), q)) {
			if qty <= 0 || qty >= it.Qty {
				c.Inventory = slices.Delete(c.Inventory, i, i+1)
				return it, true
			}
			c.Inventory[i].Qty -= qty
			it.Qty = qty
			return it, true
		}
	}
	return Item{}, false
}

func (c *Character) Equip(index int) error {
	if index < 0 || index >= len(c.Inventory) {
		return fmt.Errorf("no such item")
	}
	it := &c.Inventory[index]
	switch it.Kind {
	case "weapon", "armor", "shield":
	default:
		return fmt.Errorf("%s can't be equipped", it.Name)
	}
	if it.Equipped {
		it.Equipped = false
	} else {
		for i := range c.Inventory {
			if c.Inventory[i].Kind == it.Kind {
				c.Inventory[i].Equipped = false
			}
		}
		it.Equipped = true
	}
	c.RecalcAC()
	return nil
}

// featureUses returns how many uses a limited feature has at the character's level.
func (c *Character) featureUses(f string) int {
	switch f {
	case FSecondWind:
		return 1
	case FActionSurge:
		if c.Level >= 2 {
			return 1
		}
		return 0
	case FRage:
		switch {
		case c.Level >= 12:
			return 5
		case c.Level >= 6:
			return 4
		case c.Level >= 3:
			return 3
		}
		return 2
	case FLayOnHands:
		return 5 * c.Level
	}
	return 0
}

func (c *Character) restoreUses(shortOnly bool) {
	if c.Uses == nil {
		c.Uses = map[string]int{}
	}
	for _, f := range c.Features {
		if shortOnly && f != FSecondWind && f != FActionSurge {
			continue
		}
		if n := c.featureUses(f); n > 0 {
			c.Uses[f] = n
		}
	}
}

func (c *Character) ShortRest() int {
	if c.Dead {
		return 0
	}
	cl := c.ClassDef()
	dice := max(1, (c.Level+1)/2)
	healed := 0
	for i := 0; i < dice; i++ {
		healed += max(1, Die(cl.HitDie)+c.Mod("CON"))
	}
	c.heal(healed)
	c.restoreUses(true)
	if c.Class == "wizard" { // arcane recovery, simplified
		c.Slots = min(c.SlotsMax, c.Slots+max(1, c.Level/2))
	}
	return healed
}

func (c *Character) LongRest() {
	if c.Dead {
		return
	}
	c.HP = c.MaxHP
	c.Slots = c.SlotsMax
	c.restoreUses(false)
	c.DeathSaves, c.DeathFails, c.Stable = 0, 0, false
	c.Relentless = false
}

func (c *Character) heal(n int) int {
	if c.Dead || n <= 0 {
		return 0
	}
	before := c.HP
	c.HP = min(c.MaxHP, c.HP+n)
	if c.HP > 0 {
		c.DeathSaves, c.DeathFails, c.Stable = 0, 0, false
	}
	return c.HP - before
}

// TakeDamage applies damage and reports what happened.
func (c *Character) TakeDamage(n int, crit bool) string {
	if c.Dead {
		return ""
	}
	if c.Raging {
		n = n / 2
	}
	if c.HP <= 0 {
		c.DeathFails += 1
		if crit {
			c.DeathFails++
		}
		if c.DeathFails >= 3 {
			c.Dead = true
			return fmt.Sprintf("%s takes a blow while down and dies.", c.Name)
		}
		return fmt.Sprintf("%s takes a blow while down (death failures: %d/3).", c.Name, c.DeathFails)
	}
	if c.TempHP > 0 {
		absorb := min(c.TempHP, n)
		c.TempHP -= absorb
		n -= absorb
	}
	c.HP -= n
	if c.HP <= 0 {
		if -c.HP >= c.MaxHP { // massive damage
			c.HP = 0
			c.Dead = true
			return fmt.Sprintf("%s is slain outright!", c.Name)
		}
		if c.Race == "half-orc" && !c.Relentless {
			c.Relentless = true
			c.HP = 1
			return fmt.Sprintf("%s refuses to fall — Relentless Endurance leaves them at 1 HP!", c.Name)
		}
		c.HP = 0
		c.Raging = false
		return fmt.Sprintf("%s falls unconscious!", c.Name)
	}
	return ""
}

// DeathSave rolls a death saving throw for a downed character.
func (c *Character) DeathSave() string {
	r, _ := D20(0)
	switch {
	case r == 20:
		c.HP = 1
		c.DeathSaves, c.DeathFails = 0, 0
		return fmt.Sprintf("%s rolls a natural 20 on a death save and springs back up with 1 HP!", c.Name)
	case r == 1:
		c.DeathFails += 2
	case r >= 10:
		c.DeathSaves++
	default:
		c.DeathFails++
	}
	if c.DeathFails >= 3 {
		c.Dead = true
		return fmt.Sprintf("%s rolls %d on a death save... and dies.", c.Name, r)
	}
	if c.DeathSaves >= 3 {
		c.Stable = true
		return fmt.Sprintf("%s rolls %d and stabilizes.", c.Name, r)
	}
	return fmt.Sprintf("%s rolls %d on a death save (successes %d/3, failures %d/3).", c.Name, r, c.DeathSaves, c.DeathFails)
}

// GainXP adds experience and returns level-up messages.
func (c *Character) GainXP(n int) []string {
	if c.Dead {
		return nil
	}
	c.XP += n
	var msgs []string
	for c.Level < MaxLevel && c.XP >= XPThresholds[c.Level] {
		c.Level++
		cl := c.ClassDef()
		gain := cl.HitDie/2 + 1 + c.Mod("CON")
		if c.Race == "dwarf" {
			gain++
		}
		gain = max(1, gain)
		c.MaxHP += gain
		c.HP += gain
		oldSlots := c.SlotsMax
		c.SlotsMax = SlotsFor(cl, c.Level)
		c.Slots += c.SlotsMax - oldSlots
		// Ability score improvement at 4, 8, 12: +2 to the primary ability.
		if c.Level%4 == 0 {
			p := cl.Priority[0]
			if c.Stats[p] < 20 {
				c.Stats[p] = min(20, c.Stats[p]+2)
			} else {
				c.Stats[cl.Priority[1]] = min(20, c.Stats[cl.Priority[1]]+2)
			}
			c.RecalcAC()
		}
		c.restoreUses(false)
		msgs = append(msgs, fmt.Sprintf("⭐ %s reaches level %d! (+%d max HP)", c.Name, c.Level, gain))
	}
	return msgs
}

// NewCharacter builds a level 1 character.
func NewCharacter(id, owner, name, raceID, classID string, base map[string]int, backstory string) (*Character, error) {
	race, ok := Races[raceID]
	if !ok {
		return nil, fmt.Errorf("unknown race %q", raceID)
	}
	cl, ok := Classes[classID]
	if !ok {
		return nil, fmt.Errorf("unknown class %q", classID)
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 40 {
		return nil, fmt.Errorf("name must be 1-40 characters")
	}
	stats := map[string]int{}
	for _, ab := range Abilities {
		stats[ab] = base[ab] + race.Bonus[ab]
	}
	c := &Character{ID: id, Owner: owner, Name: name, Race: raceID, Class: classID, Level: 1, Stats: stats,
		Gold: 10 + Die(10)*2, Features: slices.Clone(cl.Features), Skills: slices.Clone(cl.Skills),
		Uses: map[string]int{}, Backstory: strings.TrimSpace(backstory)}
	c.MaxHP = max(1, cl.HitDie+c.Mod("CON"))
	if raceID == "dwarf" {
		c.MaxHP++
	}
	c.HP = c.MaxHP
	c.SlotsMax = SlotsFor(cl, 1)
	c.Slots = c.SlotsMax
	for i, w := range cl.Weapons {
		it, _ := NewItem(w)
		it.Equipped = i == 0
		c.AddItem(it)
	}
	if cl.Armor != "" && cl.Armor != "none" && cl.Armor != "unarmored_barb" {
		it, _ := NewItem(cl.Armor)
		it.Equipped = true
		c.AddItem(it)
	}
	if cl.Shield {
		it, _ := NewItem("shield")
		it.Equipped = true
		c.AddItem(it)
	}
	for _, x := range append(slices.Clone(cl.ExtraItems), "torch", "rations", "rope") {
		it, _ := NewItem(x)
		c.AddItem(it)
	}
	c.RecalcAC()
	c.restoreUses(false)
	return c, nil
}

// SkillLabel turns "sleight_of_hand" into "Sleight of Hand".
func SkillLabel(key string) string {
	words := strings.Split(key, "_")
	for i, w := range words {
		if w != "of" && w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
