package engine

import (
	"fmt"
	"sort"
	"strings"
)

type Spawn struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
	Elite bool   `json:"elite"`
	Name  string `json:"name"`
}

func (e *Engine) spawnMonster(sp Spawn, n, of int) (*Monster, error) {
	t, ok := Bestiary[sp.Type]
	if !ok {
		return nil, fmt.Errorf("unknown monster type %q", sp.Type)
	}
	m := &Monster{ID: e.nextID("m"), Template: t.ID, Name: t.Name, AC: t.AC, ToHit: t.ToHit, DexMod: t.DexMod,
		Damage: t.Damage, DmgType: t.DmgType, Attacks: t.Attacks, Attack: t.Attack, Saves: t.Saves, XP: t.XP, Gold: t.Gold, Boss: t.Boss}
	m.MaxHP = max(1, MustRoll(t.HP).Total)
	if sp.Elite {
		m.Name = pick([]string{"Chieftain", "Elder", "Dread", "Alpha", "Warlord"}) + " " + t.Name
		m.MaxHP = m.MaxHP*2 + 5
		m.AC++
		m.ToHit += 2
		m.Damage += "+2"
		m.Attacks = max(2, m.Attacks)
		m.XP = m.XP*3 + 50
		m.Boss = true
		if m.Gold == "" {
			m.Gold = "4d10"
		}
	}
	if sp.Name != "" {
		m.Name = strings.TrimSpace(sp.Name)
		if of > 1 {
			m.Name = fmt.Sprintf("%s %d", m.Name, n)
		}
	} else if of > 1 {
		m.Name = fmt.Sprintf("%s %d", m.Name, n)
	}
	m.HP = m.MaxHP
	return m, nil
}

// StartCombat rolls initiative and begins a fight. surprise: "party" (party ambushes), "enemies", or "".
func (e *Engine) StartCombat(spawns []Spawn, surprise string) (string, error) {
	if e.C.Combat != nil {
		return "", fmt.Errorf("combat is already running")
	}
	cb := &Combat{Round: 1}
	total := 0
	for _, sp := range spawns {
		cnt := min(max(sp.Count, 1), 12)
		for i := 1; i <= cnt; i++ {
			if total >= 16 {
				break
			}
			m, err := e.spawnMonster(sp, i, cnt)
			if err != nil {
				return "", err
			}
			cb.Monsters = append(cb.Monsters, m)
			total++
		}
	}
	if len(cb.Monsters) == 0 {
		return "", fmt.Errorf("no monsters")
	}
	var lines []string
	for _, ch := range e.sortedChars() {
		ch.Dodging, ch.Raging, ch.Smiting, ch.MarkTarget, ch.Fled = false, false, false, "", false
		if ch.Dead {
			continue
		}
		r, _ := ch.d20(0)
		init := r + ch.Mod("DEX")
		if surprise == "party" {
			init += 10
		}
		cb.Order = append(cb.Order, Combatant{ID: ch.ID, Name: ch.Name, Init: init})
		lines = append(lines, fmt.Sprintf("%s %d", ch.Name, init))
	}
	for _, m := range cb.Monsters {
		init := Die(20) + m.DexMod
		if surprise == "enemies" {
			init += 10
		}
		cb.Order = append(cb.Order, Combatant{ID: m.ID, Name: m.Name, IsMonster: true, Init: init})
		lines = append(lines, fmt.Sprintf("%s %d", m.Name, init))
	}
	sort.SliceStable(cb.Order, func(i, j int) bool {
		if cb.Order[i].Init != cb.Order[j].Init {
			return cb.Order[i].Init > cb.Order[j].Init
		}
		return !cb.Order[i].IsMonster && cb.Order[j].IsMonster
	})
	e.C.Combat = cb
	names := make([]string, 0, len(cb.Monsters))
	for _, m := range cb.Monsters {
		names = append(names, m.Name)
	}
	e.Log("combat", "", "⚔️ Combat! Foes: "+strings.Join(names, ", ")+". Initiative — "+strings.Join(lines, ", ")+".")
	e.runAutoTurns()
	return "Combat started. Initiative order: " + strings.Join(lines, ", "), nil
}

func (e *Engine) currentCombatant() *Combatant {
	cb := e.C.Combat
	if cb == nil || len(cb.Order) == 0 {
		return nil
	}
	return &cb.Order[cb.Turn%len(cb.Order)]
}

func (e *Engine) monster(id string) *Monster {
	if e.C.Combat == nil {
		return nil
	}
	for _, m := range e.C.Combat.Monsters {
		if m.ID == id {
			return m
		}
	}
	return nil
}

func (e *Engine) liveMonsters() []*Monster {
	var out []*Monster
	if e.C.Combat == nil {
		return nil
	}
	for _, m := range e.C.Combat.Monsters {
		if m.HP > 0 {
			out = append(out, m)
		}
	}
	return out
}

func (e *Engine) advanceTurn() {
	cb := e.C.Combat
	cb.ActionUsed, cb.BonusUsed, cb.ExtraAction, cb.SneakUsed = false, false, false, false
	cb.Turn++
	if cb.Turn >= len(cb.Order) {
		cb.Turn = 0
		cb.Round++
	}
	if cur := e.currentCombatant(); cur != nil && !cur.IsMonster {
		if ch := e.C.Characters[cur.ID]; ch != nil {
			ch.Dodging = false
		}
	}
}

// runAutoTurns plays monster turns and skips downed characters until a hero can act.
func (e *Engine) runAutoTurns() {
	for guard := 0; guard < 200 && e.C.Combat != nil; guard++ {
		if e.checkCombatEnd() {
			return
		}
		cur := e.currentCombatant()
		if cur.IsMonster {
			m := e.monster(cur.ID)
			if m != nil && m.HP > 0 {
				e.monsterTurn(m)
				if e.checkCombatEnd() {
					return
				}
			}
			e.advanceTurn()
			continue
		}
		ch := e.C.Characters[cur.ID]
		if ch == nil || ch.Dead || ch.Fled {
			e.advanceTurn()
			continue
		}
		if ch.HP <= 0 {
			if !ch.Stable {
				e.Log("combat", ch.Name, "💀 "+ch.DeathSave())
				if ch.HP > 0 {
					return // a natural 20: they act this turn
				}
			}
			e.advanceTurn()
			continue
		}
		return
	}
}

func (e *Engine) monsterTurn(m *Monster) {
	var targets []*Character
	for _, ch := range e.sortedChars() {
		if ch.Conscious() {
			targets = append(targets, ch)
		}
	}
	if len(targets) == 0 {
		return
	}
	// Prefer whoever hurt it most recently isn't tracked; pick the softest target sometimes.
	t := pick(targets)
	if chance(0.35) {
		sort.Slice(targets, func(i, j int) bool { return targets[i].AC < targets[j].AC })
		t = targets[0]
	}
	var parts []string
	for i := 0; i < max(1, m.Attacks); i++ {
		if !t.Conscious() {
			break
		}
		adv := 0
		if t.Dodging || m.Mocked {
			adv = -1
		}
		m.Mocked = false
		roll, dice := D20(adv)
		total := roll + m.ToHit
		advTxt := ""
		if len(dice) > 1 {
			advTxt = " (disadvantage)"
		}
		if roll == 1 || (roll != 20 && total < t.AC) {
			parts = append(parts, fmt.Sprintf("%s %s at %s but misses (%d vs AC %d)%s.", m.Name, m.Attack, t.Name, total, t.AC, advTxt))
			continue
		}
		dmgExpr := m.Damage
		if roll == 20 {
			dmgExpr = DoubleDice(dmgExpr)
		}
		d := MustRoll(dmgExpr)
		dmg := max(1, d.Total)
		crit := ""
		if roll == 20 {
			crit = " CRITICAL HIT!"
		}
		shown := dmg
		if t.Raging {
			shown = dmg / 2
		}
		line := fmt.Sprintf("%s %s at %s —%s %d %s damage (%d vs AC %d).", m.Name, m.Attack, t.Name, crit, shown, m.DmgType, total, t.AC)
		if after := t.TakeDamage(dmg, roll == 20); after != "" {
			line += " " + after
		}
		parts = append(parts, line)
	}
	e.Log("combat", m.Name, "👹 "+strings.Join(parts, " "))
}

// checkCombatEnd ends the fight if one side is out. Returns true if combat ended.
func (e *Engine) checkCombatEnd() bool {
	cb := e.C.Combat
	if cb == nil {
		return true
	}
	if len(e.liveMonsters()) == 0 {
		e.endCombat("victory", 0)
		return true
	}
	anyUp, anyIn := false, false
	for _, c := range cb.Order {
		if c.IsMonster {
			continue
		}
		ch := e.C.Characters[c.ID]
		if ch == nil || ch.Dead {
			continue
		}
		if !ch.Fled {
			anyIn = true
			if ch.HP > 0 {
				anyUp = true
			}
		}
	}
	if !anyIn {
		e.endCombat("escaped", 0)
		return true
	}
	if !anyUp {
		e.endCombat("defeat", 0)
		return true
	}
	return false
}

// endCombat closes a fight. outcome: victory, escaped, defeat, surrender, truce.
func (e *Engine) endCombat(outcome string, bonusXP int) {
	cb := e.C.Combat
	if cb == nil {
		return
	}
	e.C.Combat = nil
	var slain []string
	xp, gold := 0, 0
	for _, m := range cb.Monsters {
		if m.HP <= 0 {
			slain = append(slain, m.Name)
			xp += m.XP
			if m.Gold != "" {
				gold += max(0, MustRoll(m.Gold).Total)
			}
		} else if outcome == "surrender" {
			xp += m.XP / 2
		}
	}
	for _, ch := range e.C.Characters {
		ch.Raging, ch.Dodging, ch.Smiting, ch.MarkTarget, ch.Fled = false, false, false, "", false
	}
	summary := ""
	switch outcome {
	case "victory":
		e.Log("combat", "", "🏆 Victory! All foes are defeated.")
		summary = "The party won the fight. Defeated: " + strings.Join(slain, ", ") + "."
	case "escaped":
		e.Log("combat", "", "🏃 The party escapes the fight.")
		summary = "The party fled from the fight. Enemies remain."
	case "defeat":
		e.Log("combat", "", "☠️ The party has fallen...")
		summary = "Every hero was knocked out or killed. The enemies won."
		for _, ch := range e.C.Characters {
			if !ch.Dead && ch.HP <= 0 {
				ch.HP, ch.Stable, ch.DeathSaves, ch.DeathFails = 1, false, 0, 0
			}
		}
		xp, gold = 0, 0
	default:
		e.Log("combat", "", "🕊️ The fight ends ("+outcome+").")
		summary = "The fight ended by " + outcome + "."
	}
	xp += bonusXP
	if xp > 0 {
		e.AwardXP(xp, "for the battle")
	}
	if gold > 0 {
		e.splitGold(gold, "looted from the fallen")
	}
	summary += fmt.Sprintf(" XP awarded: %d. Gold looted: %d.", xp, gold)
	e.enqueue(DMEvent{Kind: map[bool]string{true: "defeat", false: "combat_end"}[outcome == "defeat"], Summary: outcome + ": " + summary})
}

func (e *Engine) splitGold(total int, reason string) {
	living := e.livingChars()
	if len(living) == 0 || total == 0 {
		return
	}
	each := total / len(living)
	rem := total - each*len(living)
	for i, ch := range living {
		ch.Gold += each
		if i < rem {
			ch.Gold++
		}
		if ch.Gold < 0 {
			ch.Gold = 0
		}
	}
	e.logf("system", "", "💰 The party gains %d gold (%s).", total, reason)
}

// CombatReq is a structured combat command from a player.
type CombatReq struct {
	Type    string `json:"type"` // attack, ability, item, dodge, flee, end
	Target  string `json:"target"`
	Ability string `json:"ability"`
	Item    int    `json:"item"`
	Weapon  int    `json:"weapon"`
}

func (e *Engine) CombatAction(token string, r CombatReq) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ch, err := e.charOf(token)
	if err != nil {
		return err
	}
	return e.combatAction(ch, r)
}

// SkipTurn lets the host move past an absent player.
func (e *Engine) SkipTurn() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.C.Combat == nil {
		return
	}
	if cur := e.currentCombatant(); cur != nil && !cur.IsMonster {
		e.logf("combat", "", "⏭️ The host skips %s's turn.", cur.Name)
		e.advanceTurn()
		e.runAutoTurns()
	}
}

func (e *Engine) combatAction(ch *Character, r CombatReq) error {
	cb := e.C.Combat
	if cb == nil {
		return fmt.Errorf("there's no fight right now")
	}
	cur := e.currentCombatant()
	if cur == nil || cur.ID != ch.ID {
		return fmt.Errorf("it's not your turn")
	}
	if !ch.Conscious() {
		return fmt.Errorf("you can't act right now")
	}
	usesAction := true
	switch r.Type {
	case "end":
		e.endTurn()
		return nil
	case "attack":
		if err := e.needAction(); err != nil {
			return err
		}
		m := e.monster(r.Target)
		if m == nil || m.HP <= 0 {
			return fmt.Errorf("pick a living target")
		}
		e.pcAttack(ch, m)
	case "dodge":
		if err := e.needAction(); err != nil {
			return err
		}
		ch.Dodging = true
		e.logf("combat", ch.Name, "🛡️ %s takes the Dodge action — attacks against them have disadvantage.", ch.Name)
	case "flee":
		if err := e.needAction(); err != nil {
			return err
		}
		best := 0
		for _, m := range e.liveMonsters() {
			best = max(best, m.DexMod)
		}
		res := ch.Check("acrobatics", 10+best+len(e.liveMonsters()), 0)
		if res.Success {
			ch.Fled = true
			e.logf("combat", ch.Name, "🏃 %s breaks away and escapes! (%s)", ch.Name, res)
		} else {
			e.logf("combat", ch.Name, "🏃 %s tries to flee but is cut off. (%s)", ch.Name, res)
		}
	case "item":
		if err := e.needAction(); err != nil {
			return err
		}
		if r.Item < 0 || r.Item >= len(ch.Inventory) {
			return fmt.Errorf("no such item")
		}
		target := r.Target
		if target == "" {
			target = ch.ID
		}
		if err := e.drink(ch, r.Item, target); err != nil {
			return err
		}
	case "ability":
		ab, ok := AbilityList[r.Ability]
		if !ok || !ch.HasFeature(r.Ability) || ab.Passive {
			return fmt.Errorf("you can't use that")
		}
		if ab.Bonus {
			if cb.BonusUsed {
				return fmt.Errorf("you've already used your bonus action")
			}
		} else if err := e.needAction(); err != nil {
			return err
		}
		if err := e.useAbility(ch, ab, r.Target); err != nil {
			return err
		}
		if ab.Bonus {
			cb.BonusUsed = true
			usesAction = false
		}
	default:
		return fmt.Errorf("unknown action")
	}
	if e.C.Combat == nil {
		return nil
	}
	if usesAction {
		if cb.ExtraAction {
			cb.ExtraAction = false
		} else {
			cb.ActionUsed = true
		}
	}
	if e.checkCombatEnd() {
		return nil
	}
	if ch.Fled || (cb.ActionUsed && (cb.BonusUsed || !e.hasBonusOption(ch))) {
		e.endTurn()
	}
	e.changed()
	return nil
}

func (e *Engine) needAction() error {
	if e.C.Combat.ActionUsed && !e.C.Combat.ExtraAction {
		return fmt.Errorf("you've used your action — use a bonus action or end your turn")
	}
	return nil
}

func (e *Engine) hasBonusOption(ch *Character) bool {
	for _, f := range ch.Features {
		ab := AbilityList[f]
		if !ab.Bonus {
			continue
		}
		switch ab.Cost {
		case "slot":
			if ch.Slots > 0 {
				return true
			}
		case "use":
			if ch.Uses[f] > 0 && !(f == FRage && ch.Raging) {
				return true
			}
		default:
			return true
		}
	}
	return false
}

func (e *Engine) endTurn() {
	if e.C.Combat == nil {
		return
	}
	e.advanceTurn()
	e.runAutoTurns()
	e.changed()
}

func (e *Engine) damageMonster(m *Monster, dmg int) string {
	m.HP -= dmg
	if m.HP <= 0 {
		m.HP = 0
		return " " + pick([]string{"%s collapses!", "%s is slain!", "%s falls and does not rise.", "%s crumples to the ground."})
	}
	if m.HP*4 <= m.MaxHP {
		return " %s is badly wounded."
	}
	return ""
}

func (e *Engine) pcAttack(ch *Character, m *Monster) {
	w := ch.Weapon()
	cb := e.C.Combat
	var parts []string
	for i := 0; i < ch.NumAttacks() && m.HP > 0; i++ {
		bonus := ch.AttackBonus(w)
		roll, _ := ch.d20(0)
		total := roll + bonus
		if roll == 1 || (roll != 20 && total < m.AC) {
			parts = append(parts, fmt.Sprintf("%s attacks %s with %s — miss (%d vs AC %d).", ch.Name, m.Name, w.Name, total, m.AC))
			continue
		}
		crit := roll == 20
		dmgMod := ch.Mod(ch.AttackStat(w)) + w.Bonus
		expr := w.Damage
		extra := []string{}
		if ch.Raging && !w.Ranged {
			dmgMod += 2
			extra = append(extra, "rage")
		}
		if ch.HasFeature(FSneakAttack) && !cb.SneakUsed {
			cb.SneakUsed = true
			expr += fmt.Sprintf("+%dd6", (ch.Level+1)/2)
			extra = append(extra, "sneak attack")
		}
		if ch.MarkTarget == m.ID {
			expr += "+1d6"
			extra = append(extra, "hunter's mark")
		}
		if ch.Smiting && !w.Ranged {
			ch.Smiting = false
			expr += "+2d8"
			extra = append(extra, "divine smite")
		}
		if crit {
			expr = DoubleDice(expr)
		}
		d := MustRoll(expr)
		dmg := max(1, d.Total+dmgMod)
		label := "hit"
		if crit {
			label = "CRITICAL HIT"
		}
		ex := ""
		if len(extra) > 0 {
			ex = " (" + strings.Join(extra, ", ") + ")"
		}
		line := fmt.Sprintf("%s attacks %s with %s — %s! %d %s damage%s (%d vs AC %d).", ch.Name, m.Name, w.Name, label, dmg, w.DmgType, ex, total, m.AC)
		if s := e.damageMonster(m, dmg); s != "" {
			line += strings.ReplaceAll(s, "%s", m.Name)
		}
		parts = append(parts, line)
		if m.HP <= 0 {
			if next := e.liveMonsters(); len(next) > 0 && i+1 < ch.NumAttacks() {
				m = next[0]
			}
		}
	}
	e.Log("combat", ch.Name, "🗡️ "+strings.Join(parts, " "))
}

func (e *Engine) useAbility(ch *Character, ab Ability, target string) error {
	cb := e.C.Combat
	spend := func() error {
		switch ab.Cost {
		case "slot":
			if ch.Slots <= 0 {
				return fmt.Errorf("no spell slots left — rest to recover them")
			}
			ch.Slots--
		case "use":
			if ch.Uses[ab.ID] <= 0 {
				return fmt.Errorf("no uses of %s left — rest to recover", ab.Name)
			}
			if ab.ID != FLayOnHands {
				ch.Uses[ab.ID]--
			}
		}
		return nil
	}
	enemy := func() (*Monster, error) {
		m := e.monster(target)
		if m == nil || m.HP <= 0 {
			return nil, fmt.Errorf("pick a living enemy")
		}
		return m, nil
	}
	ally := func() (*Character, error) {
		t, ok := e.C.Characters[target]
		if !ok {
			t = ch
		}
		if t.Dead || t.Fled {
			return nil, fmt.Errorf("%s can't be helped", t.Name)
		}
		return t, nil
	}
	cantrip := func(base string) string {
		if ch.Level >= 11 {
			return "3" + base[1:]
		}
		if ch.Level >= 5 {
			return "2" + base[1:]
		}
		return base
	}
	switch ab.ID {
	case FSecondWind:
		if err := spend(); err != nil {
			return err
		}
		r := MustRoll(fmt.Sprintf("1d10+%d", ch.Level))
		e.logf("combat", ch.Name, "💪 %s catches a Second Wind and heals %d HP.", ch.Name, ch.heal(r.Total))
	case FActionSurge:
		if err := spend(); err != nil {
			return err
		}
		cb.ExtraAction = true
		e.logf("combat", ch.Name, "⚡ %s surges with energy — an extra action this turn!", ch.Name)
	case FRage:
		if ch.Raging {
			return fmt.Errorf("already raging")
		}
		if err := spend(); err != nil {
			return err
		}
		ch.Raging = true
		e.logf("combat", ch.Name, "😡 %s flies into a RAGE! (+2 melee damage, half damage taken)", ch.Name)
	case FFireBolt, FMagicMissile, FSacredFlame, FViciousMock:
		m, err := enemy()
		if err != nil {
			return err
		}
		if err := spend(); err != nil {
			return err
		}
		switch ab.ID {
		case FFireBolt:
			roll, _ := ch.d20(0)
			total := roll + ch.SpellAttack()
			if roll == 1 || (roll != 20 && total < m.AC) {
				e.logf("combat", ch.Name, "🔥 %s hurls a Fire Bolt at %s — miss (%d vs AC %d).", ch.Name, m.Name, total, m.AC)
				break
			}
			expr := cantrip("1d10")
			if roll == 20 {
				expr = DoubleDice(expr)
			}
			d := MustRoll(expr)
			s := e.damageMonster(m, d.Total)
			e.logf("combat", ch.Name, "🔥 %s's Fire Bolt scorches %s for %d fire damage (%d vs AC %d).%s", ch.Name, m.Name, d.Total, total, m.AC, strings.ReplaceAll(s, "%s", m.Name))
		case FMagicMissile:
			darts := 3 + ch.Level/4
			d := 0
			for i := 0; i < darts; i++ {
				d += Die(4) + 1
			}
			s := e.damageMonster(m, d)
			e.logf("combat", ch.Name, "✨ %s looses %d glowing darts of Magic Missile at %s — %d force damage, no roll needed.%s", ch.Name, darts, m.Name, d, strings.ReplaceAll(s, "%s", m.Name))
		case FSacredFlame, FViciousMock:
			save := Die(20) + m.DexMod + m.Saves
			if ab.ID == FViciousMock {
				save = Die(20) + m.Saves
			}
			dc := ch.SpellDC()
			if save >= dc {
				e.logf("combat", ch.Name, "%s casts %s on %s, but it resists (save %d vs DC %d).", ch.Name, ab.Name, m.Name, save, dc)
				break
			}
			expr, verb := cantrip("1d8"), "is engulfed in radiant flame"
			if ab.ID == FViciousMock {
				expr, verb = cantrip("1d4"), "reels from a cutting insult"
				m.Mocked = true
			}
			d := MustRoll(expr)
			s := e.damageMonster(m, d.Total)
			e.logf("combat", ch.Name, "%s casts %s — %s %s and takes %d damage (save %d vs DC %d).%s", ch.Name, ab.Name, m.Name, verb, d.Total, save, dc, strings.ReplaceAll(s, "%s", m.Name))
		}
	case FCureWounds, FHealingWord:
		t, err := ally()
		if err != nil {
			return err
		}
		if err := spend(); err != nil {
			return err
		}
		expr := fmt.Sprintf("1d8+%d", max(0, ch.Mod("WIS")))
		if ab.ID == FHealingWord {
			expr = fmt.Sprintf("1d4+%d", max(0, ch.Mod("CHA")))
		}
		r := MustRoll(expr)
		got := t.heal(r.Total + ch.Level/3)
		e.logf("combat", ch.Name, "💚 %s casts %s on %s, restoring %d HP.", ch.Name, ab.Name, t.Name, got)
	case FHuntersMark:
		m, err := enemy()
		if err != nil {
			return err
		}
		if err := spend(); err != nil {
			return err
		}
		ch.MarkTarget = m.ID
		e.logf("combat", ch.Name, "🎯 %s marks %s as their quarry (+1d6 damage on hits).", ch.Name, m.Name)
	case FLayOnHands:
		t, err := ally()
		if err != nil {
			return err
		}
		pool := ch.Uses[FLayOnHands]
		need := min(10, t.MaxHP-t.HP, pool)
		if need <= 0 {
			if pool <= 0 {
				return fmt.Errorf("your healing pool is empty — take a long rest")
			}
			return fmt.Errorf("%s is already at full health", t.Name)
		}
		ch.Uses[FLayOnHands] -= need
		e.logf("combat", ch.Name, "🙏 %s lays hands on %s, healing %d HP (pool left: %d).", ch.Name, t.Name, t.heal(need), ch.Uses[FLayOnHands])
	case FDivineSmite:
		if ch.Smiting {
			return fmt.Errorf("your next hit is already charged")
		}
		if err := spend(); err != nil {
			return err
		}
		ch.Smiting = true
		e.logf("combat", ch.Name, "⚜️ %s's weapon glows with holy light — the next melee hit smites for +2d8.", ch.Name)
	default:
		return fmt.Errorf("that ability can't be used now")
	}
	return nil
}

// UseAbilityOutOfCombat allows healing spells and abilities between fights.
func (e *Engine) UseAbilityOutOfCombat(token, ability, target string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ch, err := e.charOf(token)
	if err != nil {
		return err
	}
	if e.C.Combat != nil {
		return e.combatAction(ch, CombatReq{Type: "ability", Ability: ability, Target: target})
	}
	ab, ok := AbilityList[ability]
	if !ok || !ch.HasFeature(ability) {
		return fmt.Errorf("you can't use that")
	}
	if ab.Target != "ally" && ability != FSecondWind {
		return fmt.Errorf("%s is only useful in a fight", ab.Name)
	}
	if !ch.Conscious() {
		return fmt.Errorf("you're unconscious")
	}
	return e.useAbility(ch, ab, target)
}

// CombatSummary describes the fight for the DM prompt.
func (e *Engine) CombatSummary() string {
	cb := e.C.Combat
	if cb == nil {
		return "No combat."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Round %d. Turn: %s.\n", cb.Round, e.currentCombatant().Name)
	for _, m := range cb.Monsters {
		fmt.Fprintf(&b, "- %s [%s] HP %d/%d AC %d\n", m.Name, m.ID, m.HP, m.MaxHP, m.AC)
	}
	return b.String()
}

// hurtOutsideCombat applies hazard damage. A hero dropped outside a fight is
// stabilized by the party instead of rolling death saves alone.
func (e *Engine) hurtOutsideCombat(ch *Character, dmg int) string {
	note := ch.TakeDamage(dmg, false)
	if e.C.Combat == nil && !ch.Dead && ch.HP <= 0 {
		ch.Stable = true
		note = strings.TrimSpace(note + " The party drags them clear — unconscious but stable. A rest or healing will bring them round.")
	}
	return note
}
