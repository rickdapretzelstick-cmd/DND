package engine

import (
	"fmt"
	"strings"
)

// CustomSkill is a perk invented in play (by the AI DM or the quest generator) for an odd job
// or a funny feat. Every number is chosen by the engine from a bounded menu, so a DM can't
// break the game: passives are small, actives scale with the hero's level.
type CustomSkill struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Desc    string   `json:"desc"`
	Kind    string   `json:"kind"`              // passive, active
	Mod     string   `json:"mod,omitempty"`     // passive: skill, ac, hp, ce, init, heal, damage, discount
	Skill   string   `json:"skill,omitempty"`   // for mod=skill
	Amount  int      `json:"amount,omitempty"`  // passive strength
	Source  string   `json:"source,omitempty"`  // where it came from, e.g. the quest title
	Ability *Ability `json:"ability,omitempty"` // active: the battle move
}

const maxCustomSkills = 8

var passiveMods = map[string]string{
	"skill": "to a skill's checks", "ac": "armor class", "hp": "max HP", "ce": "max cursed energy",
	"init": "initiative", "heal": "to healing you give", "damage": "damage on your attacks", "discount": "% off at shops (whole party)",
}

// Passive sums a character's custom passive bonuses for a key like "ac" or "skill:medicine".
func (c *Character) Passive(key string) int {
	sum := 0
	for _, s := range c.Custom {
		if s.Kind != "passive" {
			continue
		}
		k := s.Mod
		if s.Mod == "skill" {
			k = "skill:" + s.Skill
		}
		if k == key {
			sum += s.Amount
		}
	}
	return sum
}

func passiveCap(mod string, level int) int {
	switch mod {
	case "hp":
		return 2 + level
	case "ce":
		return 2
	case "heal", "damage":
		return 2
	case "discount":
		return 10
	case "init":
		return 3
	}
	return 3 // skill, ac handled below
}

var activeVFX = map[string]string{
	"damage": "punch", "heal": "heal", "buff": "aura_gold", "debuff": "stare", "stun": "voice",
}

var allowedVFX = map[string]bool{"punch": true, "slash": true, "beam": true, "red": true, "purple": true, "aura_gold": true, "aura_blue": true, "aura_red": true,
	"heal": true, "shield": true, "stare": true, "wail": true, "claw": true, "bite": true, "lightning": true, "voice": true, "tool": true, "clap": true, "embers": true}

var buffStatuses = map[string]bool{"guard": true, "empowered": true, "regen": true, "haste": true}
var debuffStatuses = map[string]bool{"exposed": true, "weakened": true, "marked": true}

// SkillSpec is what a DM (or generator) asks for.
type SkillSpec struct {
	Name, Desc, Kind   string
	Mod, Skill         string
	Amount             int
	Effect, Status     string
	Power              int
	VFX, Color, Source string
}

// BuildSkill validates a spec against the level-scaled limits and returns the finished skill.
func BuildSkill(level int, id string, sp SkillSpec) (CustomSkill, error) {
	sp.Name, sp.Desc = strings.TrimSpace(sp.Name), strings.TrimSpace(sp.Desc)
	if sp.Name == "" || len(sp.Name) > 40 {
		return CustomSkill{}, fmt.Errorf("give the skill a short name")
	}
	if len(sp.Desc) > 300 {
		sp.Desc = sp.Desc[:300]
	}
	cs := CustomSkill{ID: id, Name: sp.Name, Desc: sp.Desc, Source: sp.Source}
	switch sp.Kind {
	case "passive", "":
		cs.Kind = "passive"
		if _, ok := passiveMods[sp.Mod]; !ok {
			return cs, fmt.Errorf("passive mod must be one of: skill, ac, hp, ce, init, heal, damage, discount")
		}
		cs.Mod = sp.Mod
		if sp.Mod == "skill" {
			k := NormalizeSkill(sp.Skill)
			if _, ok := Skills[k]; !ok {
				return cs, fmt.Errorf("unknown skill %q", sp.Skill)
			}
			cs.Skill = k
		}
		amt := sp.Amount
		if amt < 1 {
			amt = 1
		}
		limit := passiveCap(sp.Mod, level)
		if sp.Mod == "ac" {
			limit = 1 // armor class is precious
		}
		cs.Amount = min(amt, limit)
		if cs.Desc == "" {
			cs.Desc = fmt.Sprintf("+%d %s.", cs.Amount, passiveMods[sp.Mod])
		}
	case "active":
		cs.Kind = "active"
		p := min(3, max(1, sp.Power))
		n := p + level/4
		color := sp.Color
		if !strings.HasPrefix(color, "#") || len(color) != 7 {
			color = "#ffd166"
		}
		vfx := sp.VFX
		if !allowedVFX[vfx] {
			vfx = activeVFX[sp.Effect]
		}
		ab := Ability{ID: id, Name: cs.Name, Desc: cs.Desc, Tech: "custom", Cost: p, PerBattle: 4 - p, VFX: vfx, Color: color}
		switch sp.Effect {
		case "damage":
			ab.Effects = []Effect{{Kind: "attack", Target: "enemy", Dice: fmt.Sprintf("%dd6", n), DmgType: "force", AddMod: true}}
		case "heal":
			ab.Cost = p + 1
			ab.Effects = []Effect{{Kind: "heal", Target: "ally", Dice: fmt.Sprintf("%dd6", n), AddMod: true}}
		case "buff":
			st := sp.Status
			if !buffStatuses[st] {
				st = "empowered"
			}
			if st == "haste" && p < 3 {
				st = "empowered"
			}
			ab.Bonus = true
			ab.Effects = []Effect{{Kind: "status", Target: "ally", Status: st, Rounds: 1 + p/2}}
		case "debuff":
			st := sp.Status
			if !debuffStatuses[st] {
				st = "exposed"
			}
			ab.Effects = []Effect{{Kind: "status", Target: "enemy", Status: st, Rounds: 1 + p/2}}
		case "stun":
			ab.Cost = p + 1
			ab.Effects = []Effect{{Kind: "save", Target: "enemy", Save: "WIS", Status: "stun", Rounds: 1}}
		default:
			return cs, fmt.Errorf("active effect must be damage, heal, buff, debuff or stun")
		}
		if ab.Desc == "" {
			ab.Desc = map[string]string{"damage": "A fabricated trick that hurts.", "heal": "Patch an ally up.", "buff": "Rally yourself or an ally.",
				"debuff": "Leave an enemy open.", "stun": "Freeze an enemy in place (WIS save)."}[sp.Effect]
			cs.Desc = ab.Desc
		}
		cs.Ability = &ab
	default:
		return cs, fmt.Errorf("kind must be passive or active")
	}
	return cs, nil
}

// GrantSkill adds a custom skill to a character, applying one-time effects (max HP, CE).
func (e *Engine) GrantSkill(ch *Character, sp SkillSpec) (CustomSkill, error) {
	if len(ch.Custom) >= maxCustomSkills {
		return CustomSkill{}, fmt.Errorf("%s already has %d fabricated skills (the maximum)", ch.Name, maxCustomSkills)
	}
	for _, s := range ch.Custom {
		if strings.EqualFold(s.Name, strings.TrimSpace(sp.Name)) {
			return s, fmt.Errorf("%s already has %q", ch.Name, s.Name)
		}
	}
	cs, err := BuildSkill(ch.Level, "cs_"+e.nextID("sk"), sp)
	if err != nil {
		return cs, err
	}
	if cs.Kind == "passive" {
		switch cs.Mod {
		case "hp":
			ch.MaxHP += cs.Amount
			ch.HP += cs.Amount
		case "ce":
			ch.CEMax += cs.Amount
			ch.CE += cs.Amount
		case "ac":
			ch.Custom = append(ch.Custom, cs)
			ch.RecalcAC()
			ch.Custom = ch.Custom[:len(ch.Custom)-1]
		}
	}
	ch.Custom = append(ch.Custom, cs)
	label := "passive"
	if cs.Kind == "active" {
		label = "technique"
	}
	e.Log("system", ch.Name, fmt.Sprintf("🌟 %s learned a new %s: %s — %s", ch.Name, label, cs.Name, cs.Desc))
	return cs, nil
}

// bestDiscount is the largest shop discount (percent) any living sorcerer has fabricated.
func (e *Engine) bestDiscount() int {
	best := 0
	for _, ch := range e.C.Characters {
		if !ch.Dead {
			best = max(best, ch.Passive("discount"))
		}
	}
	return best
}
