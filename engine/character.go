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

// AssignByPriority maps scores onto a technique's ability priority.
func AssignByPriority(scores []int, tech Technique) map[string]int {
	s := slices.Clone(scores)
	sort.Sort(sort.Reverse(sort.IntSlice(s)))
	m := map[string]int{}
	for i, ab := range tech.Priority {
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
func (c *Character) Tech() Technique   { return Techniques[c.Class] }
func (c *Character) Prof() int         { return ProfBonus(c.Level) }
func (c *Character) Conscious() bool   { return !c.Dead && c.HP > 0 && !c.Fled }
func (c *Character) Grade() string     { return SorcererGrade(c.Level) }

// CastMod is the ability modifier that powers techniques.
func (c *Character) CastMod() int { return c.Mod(c.Tech().Priority[0]) }

func (c *Character) SpellAttack() int { return c.CastMod() + c.Prof() }

func (c *Character) SpellDC() int {
	dc := 8 + c.CastMod() + c.Prof()
	if c.Race == "inumaki" {
		dc++
	}
	return dc
}

func (c *Character) Equipped(kind string) *Item {
	for i := range c.Inventory {
		if c.Inventory[i].Equipped && c.Inventory[i].Kind == kind {
			return &c.Inventory[i]
		}
	}
	return nil
}

func (c *Character) Weapon() Item {
	if w := c.Equipped("weapon"); w != nil {
		return *w
	}
	it := ItemList["fists"]
	return it
}

func (c *Character) RecalcAC() {
	dex := c.Mod("DEX")
	ac := 10 + dex
	if a := c.Equipped("armor"); a != nil {
		if a.MaxDex < 0 {
			ac = a.BaseAC + dex
		} else {
			ac = a.BaseAC + min(dex, a.MaxDex)
		}
		ac += a.Bonus
	}
	if c.Class == "heavenly_restriction" {
		ac += 2
	}
	c.AC = ac + c.Passive("ac")
}

func (c *Character) AttackStat(w Item) string {
	if w.Ranged {
		return "DEX"
	}
	if w.Finesse && c.Mod("DEX") > c.Mod("STR") {
		return "DEX"
	}
	return "STR"
}

func (c *Character) AttackBonus(w Item) int {
	b := c.Mod(c.AttackStat(w)) + c.Prof() + w.Bonus
	if c.Race == "zenin" {
		b++
	}
	return b
}

func (c *Character) NumAttacks() int {
	switch {
	case c.Class == "heavenly_restriction" && c.Level >= 3:
		if c.Level >= 9 {
			return 3
		}
		return 2
	case c.Level >= 5 && (c.Tech().HitDie >= 10):
		return 2
	}
	return 1
}

// CEMaxFor computes the cursed-energy pool.
func (c *Character) CEMaxFor() int {
	if c.Tech().NoCE {
		return 0
	}
	n := 6 + 3*c.Level + 2*max(0, c.CastMod())
	if c.Race == "kamo" {
		n += 5
	}
	return n
}

// AbilityCost returns the CE cost after origin discounts.
func (c *Character) AbilityCost(a Ability) int {
	cost := a.Cost
	if c.Race == "gojo" && cost > 1 {
		cost--
	}
	return cost
}

// KnownAbilities lists every ability the character can use, in display order.
func (c *Character) KnownAbilities() []Ability {
	var out []Ability
	for _, id := range []string{"strike", "guard", "binding_vow", "simple_domain", "rct", "overdrive"} {
		a := AbilityList[id]
		if c.Level >= a.MinLevel {
			if id == "simple_domain" && c.Tech().NoCE {
				continue
			}
			out = append(out, a)
		}
	}
	t := c.Tech()
	for _, id := range t.Moves {
		if a := AbilityList[id]; c.Level >= a.MinLevel {
			out = append(out, a)
		}
	}
	if d, ok := AbilityList[t.Domain]; ok && c.Level >= d.MinLevel {
		out = append(out, d)
	}
	for _, cs := range c.Custom {
		if cs.Kind == "active" && cs.Ability != nil {
			out = append(out, *cs.Ability)
		}
	}
	return out
}

func (c *Character) Knows(id string) bool {
	for _, a := range c.KnownAbilities() {
		if a.ID == id {
			return true
		}
	}
	return false
}

func (c *Character) SkillProficient(skill string) bool { return slices.Contains(c.Skills, skill) }

// NormalizeSkill turns text like "Sleight of Hand" or "dex" into a skill id or ability code.
func NormalizeSkill(s string) string {
	t := strings.ToLower(strings.TrimSpace(s))
	t = strings.ReplaceAll(t, " ", "_")
	t = strings.ReplaceAll(t, "-", "_")
	alias := map[string]string{"arcana": "jujutsu_lore", "religion": "barriers", "nature": "survival", "animal_handling": "insight", "lore": "jujutsu_lore", "sense": "curse_sense"}
	if a, ok := alias[t]; ok {
		t = a
	}
	if _, ok := Skills[t]; ok {
		return t
	}
	ab := map[string]string{"str": "STR", "strength": "STR", "dex": "DEX", "dexterity": "DEX", "agility": "DEX", "con": "CON",
		"constitution": "CON", "toughness": "CON", "int": "INT", "intelligence": "INT", "technique": "INT", "wis": "WIS", "wisdom": "WIS",
		"perception_ability": "WIS", "cha": "CHA", "charisma": "CHA", "will": "CHA"}
	if a, ok := ab[t]; ok {
		return a
	}
	return ""
}

func SkillLabel(key string) string {
	words := strings.Split(key, "_")
	for i, w := range words {
		if w != "of" && w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

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

func (c *Character) Check(skillOrAbility string, dc, adv int) CheckResult {
	key := NormalizeSkill(skillOrAbility)
	if key == "" {
		key = "WIS"
	}
	ab, label := key, AbilityNames[key]
	bonus := 0
	if a, ok := Skills[key]; ok {
		ab = a
		label = SkillLabel(key)
		if c.SkillProficient(key) {
			bonus += c.Prof()
		}
	}
	bonus += c.Mod(ab) + c.Passive("skill:"+key)
	roll, dice := D20(adv)
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
	t := c.Tech()
	if t.Priority[0] == ab || t.Priority[1] == ab {
		bonus += c.Prof()
	}
	roll, dice := D20(adv)
	total := roll + bonus
	return CheckResult{Name: c.Name, Label: AbilityNames[ab] + " save", Roll: roll, Dice: dice, Bonus: bonus, Total: total,
		DC: dc, Success: total >= dc, Nat20: roll == 20, Nat1: roll == 1}
}

func (c *Character) AddItem(it Item) {
	if it.Qty < 1 {
		it.Qty = 1
	}
	for i := range c.Inventory {
		if c.Inventory[i].ID == it.ID && c.Inventory[i].Name == it.Name && it.Kind != "weapon" && it.Kind != "armor" {
			c.Inventory[i].Qty += it.Qty
			return
		}
	}
	c.Inventory = append(c.Inventory, it)
}

func (c *Character) HasItem(id string) bool {
	for _, it := range c.Inventory {
		if it.ID == id {
			return true
		}
	}
	return false
}

func (c *Character) RemoveItem(idOrName string, qty int) (Item, bool) {
	q := strings.ToLower(idOrName)
	for i := range c.Inventory {
		it := c.Inventory[i]
		if strings.ToLower(it.ID) == q || strings.ToLower(it.Name) == q || (q != "" && strings.Contains(strings.ToLower(it.Name), q)) {
			if qty <= 0 || qty >= it.Qty {
				c.Inventory = slices.Delete(c.Inventory, i, i+1)
				c.RecalcAC()
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
	if it.Kind != "weapon" && it.Kind != "armor" {
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

func (c *Character) ShortRest() int {
	if c.Dead {
		return 0
	}
	dice := max(1, (c.Level+1)/2)
	healed := 0
	for i := 0; i < dice; i++ {
		healed += max(1, Die(c.Tech().HitDie)+c.Mod("CON"))
	}
	got := c.heal(healed)
	c.CE = min(c.CEMax, c.CE+c.CEMax/2)
	return got
}

func (c *Character) LongRest() {
	if c.Dead {
		return
	}
	c.HP = c.MaxHP
	c.CE = c.CEMax
	c.DeathSaves, c.DeathFails, c.Stable = 0, 0, false
	c.Status = map[string]int{}
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

// TakeDamage applies final damage and reports what happened.
func (c *Character) TakeDamage(n int, crit bool) string {
	if c.Dead || n <= 0 {
		return ""
	}
	if c.HP <= 0 {
		c.DeathFails++
		if crit {
			c.DeathFails++
		}
		if c.DeathFails >= 3 {
			c.Dead = true
			return fmt.Sprintf("%s dies.", c.Name)
		}
		return fmt.Sprintf("%s takes a blow while down (failures %d/3).", c.Name, c.DeathFails)
	}
	c.HP -= n
	if c.HP <= 0 {
		c.HP = 0
		return fmt.Sprintf("%s goes down!", c.Name)
	}
	return ""
}

func (c *Character) DeathSave() string {
	r, _ := D20(0)
	switch {
	case r == 20:
		c.HP = 1
		c.DeathSaves, c.DeathFails = 0, 0
		return fmt.Sprintf("%s rolls a natural 20 and forces themself back up with 1 HP!", c.Name)
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
	return fmt.Sprintf("%s rolls %d on a death save (%d✔ %d✘).", c.Name, r, c.DeathSaves, c.DeathFails)
}

func (c *Character) GainXP(n int) []string {
	if c.Dead {
		return nil
	}
	c.XP += n
	var msgs []string
	for c.Level < MaxLevel && c.XP >= XPThresholds[c.Level] {
		oldGrade := c.Grade()
		c.Level++
		t := c.Tech()
		gain := max(1, t.HitDie/2+1+c.Mod("CON"))
		if c.Vessel {
			gain += 2
		}
		c.MaxHP += gain
		c.HP += gain
		if c.Level%4 == 0 {
			p := t.Priority[0]
			if c.Stats[p] < 20 {
				c.Stats[p] = min(20, c.Stats[p]+2)
			} else {
				c.Stats[t.Priority[1]] = min(20, c.Stats[t.Priority[1]]+2)
			}
			c.RecalcAC()
		}
		old := c.CEMax
		c.CEMax = c.CEMaxFor()
		c.CE += c.CEMax - old
		msg := fmt.Sprintf("⭐ %s reaches level %d! (+%d max HP)", c.Name, c.Level, gain)
		if g := c.Grade(); g != oldGrade {
			msg += fmt.Sprintf(" Jujutsu High promotes them to %s sorcerer.", g)
		}
		for _, a := range c.KnownAbilities() {
			if a.MinLevel == c.Level {
				msg += fmt.Sprintf(" New: %s.", a.Name)
			}
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

func NewCharacter(id, owner, name, originID, techID string, base map[string]int, backstory string, look Appearance) (*Character, error) {
	origin, ok := Origins[originID]
	if !ok {
		return nil, fmt.Errorf("unknown origin %q", originID)
	}
	tech, ok := Techniques[techID]
	if !ok {
		return nil, fmt.Errorf("unknown technique %q", techID)
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 40 {
		return nil, fmt.Errorf("name must be 1-40 characters")
	}
	stats := map[string]int{}
	for _, ab := range Abilities {
		stats[ab] = base[ab] + origin.Bonus[ab]
	}
	skills := slices.Clone(tech.Skills)
	if originID == "non_clan" {
		for _, s := range []string{"perception", "persuasion", "athletics", "investigation"} {
			if !slices.Contains(skills, s) {
				skills = append(skills, s)
				break
			}
		}
	}
	c := &Character{ID: id, Owner: owner, Name: name, Race: originID, Class: techID, Level: 1, Stats: stats,
		Gold: 5000 + Die(10)*1000, Skills: skills, Backstory: strings.TrimSpace(backstory), Look: look,
		Status: map[string]int{}, Used: map[string]int{}}
	c.MaxHP = max(1, 2*(tech.HitDie+c.Mod("CON"))) // first fights should be survivable
	if tech.NoCE {
		c.MaxHP += 4
	}
	c.HP = c.MaxHP
	c.CEMax = c.CEMaxFor()
	c.CE = c.CEMax
	if w, ok := NewItem(tech.Weapon); ok {
		w.Equipped = true
		c.AddItem(w)
	}
	if a, ok := NewItem(tech.Armor); ok {
		a.Equipped = true
		c.AddItem(a)
	}
	for _, x := range []string{"medicine", "medicine", "ce_tonic", "phone", "student_id"} {
		it, _ := NewItem(x)
		c.AddItem(it)
	}
	if techID == "cursed_speech" {
		it, _ := NewItem("cough_syrup")
		it.Qty = 2
		c.AddItem(it)
	}
	c.RecalcAC()
	return c, nil
}
