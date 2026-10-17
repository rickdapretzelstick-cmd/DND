package engine

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"strings"
)

type Spawn struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
	Elite bool   `json:"elite"`
	Name  string `json:"name"`
	Power int    `json:"power,omitempty"` // percent strength for story bosses met early (0 = full)
}

// fighter wraps either side of a battle.
type fighter struct {
	c *Character
	m *Monster
}

func (f fighter) ok() bool { return f.c != nil || f.m != nil }
func (f fighter) id() string {
	if f.c != nil {
		return f.c.ID
	}
	return f.m.ID
}
func (f fighter) name() string {
	if f.c != nil {
		return f.c.Name
	}
	return f.m.Name
}
func (f fighter) party() bool { return f.c != nil }
func (f fighter) alive() bool {
	if f.c != nil {
		return f.c.Conscious()
	}
	return f.m.HP > 0
}
func (f fighter) status() map[string]int {
	if f.c != nil {
		if f.c.Status == nil {
			f.c.Status = map[string]int{}
		}
		return f.c.Status
	}
	if f.m.Status == nil {
		f.m.Status = map[string]int{}
	}
	return f.m.Status
}
func (f fighter) has(s string) bool { return f.status()[s] > 0 }
func (f fighter) ac() int {
	if f.c != nil {
		return f.c.AC
	}
	return f.m.AC
}
func (f fighter) maxHP() int {
	if f.c != nil {
		return f.c.MaxHP
	}
	return f.m.MaxHP
}
func (f fighter) hp() int {
	if f.c != nil {
		return f.c.HP
	}
	return f.m.HP
}
func (f fighter) level() int {
	if f.c != nil {
		return f.c.Level
	}
	return 1 + f.m.Rank*2
}
func (f fighter) dc() int {
	if f.c != nil {
		return f.c.SpellDC()
	}
	return 8 + f.m.ToHit
}
func (f fighter) save(ab string, dc int) (bool, string) {
	if f.c != nil {
		r := f.c.Save(ab, dc, 0)
		return r.Success, r.String()
	}
	roll := Die(20)
	total := roll + f.m.Save
	return total >= dc, fmt.Sprintf("%s — %s save: d20 %d %+d = %d vs DC %d", f.m.Name, AbilityNames[ab], roll, f.m.Save, total, dc)
}

// damage applies final damage. Returns what was dealt, a note, and whether it dropped.
func (f fighter) damage(n int, crit bool) (int, string, bool) {
	if n < 0 {
		n = 0
	}
	if f.c != nil {
		wasUp := f.c.HP > 0
		// Once per battle, a blow that would drop a healthy sorcerer from over half HP leaves them clinging on at 1.
		pre := ""
		if f.c.InBattle && n >= f.c.HP && f.c.HP*2 > f.c.MaxHP && f.c.Used["resolve"] == 0 {
			f.c.Used["resolve"] = 1
			n = f.c.HP - 1
			pre = "💢 Refusing to fall! "
		}
		note := pre + f.c.TakeDamage(n, crit)
		return n, note, wasUp && f.c.HP <= 0
	}
	f.m.HP -= n
	if f.m.HP <= 0 {
		f.m.HP = 0
		return n, "", true
	}
	return n, "", false
}

func (f fighter) heal(n int) int {
	if f.c != nil {
		return f.c.heal(n)
	}
	before := f.m.HP
	f.m.HP = min(f.m.MaxHP, f.m.HP+n)
	return f.m.HP - before
}

func (e *Engine) fighterByID(id string) fighter {
	if ch, ok := e.C.Characters[id]; ok {
		return fighter{c: ch}
	}
	if m := e.monster(id); m != nil {
		return fighter{m: m}
	}
	return fighter{}
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

// battleChars lists the characters taking part in the current fight.
func (e *Engine) battleChars() []*Character {
	var out []*Character
	if e.C.Combat == nil {
		return nil
	}
	for _, o := range e.C.Combat.Order {
		if !o.IsMonster {
			if ch, ok := e.C.Characters[o.ID]; ok {
				out = append(out, ch)
			}
		}
	}
	return out
}

func (e *Engine) side(party bool) []fighter {
	var out []fighter
	if party {
		for _, ch := range e.battleChars() {
			if ch.Conscious() {
				out = append(out, fighter{c: ch})
			}
		}
		return out
	}
	for _, m := range e.liveMonsters() {
		out = append(out, fighter{m: m})
	}
	return out
}

func (e *Engine) fx(f FX) {
	cb := e.C.Combat
	if cb == nil {
		return
	}
	if f.Kind == "move" || f.Kind == "blackflash" {
		g := 0
		for _, h := range f.Hits {
			switch {
			case h.Miss && e.fighterByID(h.ID).party(): // a dodge feeds the meter too
				g = max(g, 1)
			case !h.Miss && h.Dmg > 0 && (h.Crit || h.BF):
				g = 2
			case !h.Miss && h.Dmg > 0:
				g = max(g, 1)
			}
		}
		cb.Flow = min(cb.FlowMax, cb.Flow+g)
	}
	cb.FXSeq++
	f.ID = cb.FXSeq
	cb.FX = append(cb.FX, f)
	if len(cb.FX) > 40 {
		cb.FX = cb.FX[len(cb.FX)-40:]
	}
}

func (e *Engine) spawnMonster(sp Spawn, n, of int) (*Monster, error) {
	t, ok := Bestiary[sp.Type]
	if !ok {
		return nil, fmt.Errorf("unknown curse type %q", sp.Type)
	}
	m := &Monster{ID: e.nextID("m"), Template: t.ID, Name: t.Name, AC: t.AC, ToHit: t.ToHit, Init: t.Init, Save: t.Save, Rank: t.Rank,
		XP: t.XP, Yen: t.Yen, Boss: t.Boss, Domain: t.Domain, Escapes: t.Escapes, Status: map[string]int{}, Sprite: t.Sprite}
	m.MaxHP = max(1, MustRoll(t.HP).Total)
	// Curses scale with the number of sorcerers fighting them.
	if parts := len(e.C.Combat.Order); parts > 1 && t.Boss {
		m.MaxHP = m.MaxHP * (100 + 30*min(parts-1, 3)) / 100
	}
	if sp.Elite {
		m.Name = "Enraged " + t.Name
		m.MaxHP = m.MaxHP*2 + 5
		m.AC++
		m.ToHit += 2
		m.XP = m.XP*3 + 50
		m.Boss = true
		m.Sprite.Size = min(4, m.Sprite.Size+1)
	}
	if sp.Power > 0 && sp.Power < 100 {
		m.Power = sp.Power
		m.MaxHP = max(10, m.MaxHP*sp.Power/100)
		m.ToHit -= (100 - sp.Power) / 18
		m.AC -= (100 - sp.Power) / 30
		m.Save -= (100 - sp.Power) / 25
		m.XP = m.XP * (sp.Power + 40) / 140
	}
	if sp.Name != "" {
		m.Name = strings.TrimSpace(sp.Name)
	}
	if of > 1 && sp.Name == "" {
		m.Name = fmt.Sprintf("%s %c", m.Name, 'A'+rune(n-1))
	}
	m.HP = m.MaxHP
	e.planMove(m)
	return m, nil
}

// planMove picks the monster's next move ahead of time so the party can see it coming.
func (e *Engine) planMove(m *Monster) {
	t := Bestiary[m.Template]
	total := 0
	for _, mv := range t.Moves {
		total += mv.Weight
	}
	if total == 0 {
		return
	}
	roll := rand.IntN(total)
	m.nextIdx = 0
	for i, x := range t.Moves {
		if roll < x.Weight {
			m.nextIdx = i
			break
		}
		roll -= x.Weight
	}
	mv := t.Moves[m.nextIdx]
	m.Next, m.NextKind, m.NextAoE = mv.Name, "attack", false
	if len(mv.Effects) > 0 {
		m.NextKind = mv.Effects[0].Kind
		m.NextAoE = mv.Effects[0].Target == "all_enemies"
	}
}

type BattleOpts struct {
	Spawns   []Spawn
	Party    []*Character
	X, Y     float64
	Biome    string
	Title    string
	RoamerID string
	QuestID  string
	Surprise string // "party", "enemies" or ""
}

// StartBattle begins a fight. Caller holds the lock.
func (e *Engine) StartBattle(o BattleOpts) (string, error) {
	if e.C.Combat != nil {
		return "", fmt.Errorf("a battle is already underway")
	}
	var party []*Character
	for _, ch := range o.Party {
		if ch != nil && !ch.Dead && ch.HP > 0 {
			party = append(party, ch)
		}
	}
	if len(party) == 0 {
		return "", fmt.Errorf("no sorcerer can fight")
	}
	cb := &Combat{FlowMax: 10, Round: 1, Biome: o.Biome, X: o.X, Y: o.Y, RoamerID: o.RoamerID, QuestID: o.QuestID, Title: o.Title}
	e.C.Combat = cb
	var lines []string
	for _, ch := range party {
		ch.Status, ch.Used, ch.Fled, ch.InBattle = map[string]int{}, map[string]int{}, false, true
		r, _ := D20(0)
		init := r + ch.Mod("DEX")
		if o.Surprise != "enemies" {
			init += 8 // the party gets the first move unless ambushed
		}
		if o.Surprise == "party" {
			init += 10
		}
		cb.Order = append(cb.Order, Combatant{ID: ch.ID, Name: ch.Name, Init: init})
		lines = append(lines, fmt.Sprintf("%s %d", ch.Name, init))
	}
	total := 0
	for _, sp := range o.Spawns {
		cnt := min(max(sp.Count, 1), 8)
		for i := 1; i <= cnt && total < 10; i++ {
			m, err := e.spawnMonster(sp, i, cnt)
			if err != nil {
				e.C.Combat = nil
				for _, ch := range party {
					ch.InBattle = false
				}
				return "", err
			}
			cb.Monsters = append(cb.Monsters, m)
			total++
		}
	}
	if total == 0 {
		e.C.Combat = nil
		return "", fmt.Errorf("no curses")
	}
	for _, m := range cb.Monsters {
		init := Die(20) + m.Init
		if o.Surprise == "enemies" {
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
	names := make([]string, 0, len(cb.Monsters))
	for _, m := range cb.Monsters {
		names = append(names, fmt.Sprintf("%s (%s)", m.Name, GradeNames[m.Rank]))
	}
	if cb.Title == "" {
		cb.Title = "Exorcism"
	}
	e.Log("combat", "", "⚔️ "+cb.Title+"! "+strings.Join(names, ", ")+". Initiative — "+strings.Join(lines, ", ")+".")
	e.fx(FX{Kind: "text", Text: "Battle start!"})
	e.beginTurn()
	e.runAutoTurns()
	return "Battle started. Initiative: " + strings.Join(lines, ", "), nil
}

func (e *Engine) currentCombatant() *Combatant {
	cb := e.C.Combat
	if cb == nil || len(cb.Order) == 0 {
		return nil
	}
	return &cb.Order[cb.Turn%len(cb.Order)]
}

func (e *Engine) current() fighter {
	if c := e.currentCombatant(); c != nil {
		return e.fighterByID(c.ID)
	}
	return fighter{}
}

func (e *Engine) advanceTurn() {
	cb := e.C.Combat
	cb.ActionUsed, cb.BonusUsed, cb.ExtraAction = false, false, false
	cb.Turn++
	if cb.Turn >= len(cb.Order) {
		cb.Turn = 0
		cb.Round++
	}
	e.beginTurn()
}

var buffs = []string{"infinity", "guard", "empowered", "haste", "regen", "zone", "jackpot", "simple_domain"}

// beginTurn runs start-of-turn effects for whoever is up. Caller holds the lock.
func (e *Engine) beginTurn() {
	cb := e.C.Combat
	if cb == nil {
		return
	}
	f := e.current()
	if !f.ok() || (f.m != nil && f.m.HP <= 0) || (f.c != nil && (f.c.Dead || f.c.Fled || f.c.HP <= 0)) {
		return
	}
	st := f.status()
	// Domain pulse for its owner.
	if d := cb.Domain; d != nil && d.Owner == f.id() {
		ab := AbilityList[d.Ability]
		e.applyAbility(f, ab, "", true)
		d.Rounds--
		if d.Rounds <= 0 {
			e.collapseDomain(true)
		}
		if e.C.Combat == nil {
			return
		}
	}
	// Damage over time and regeneration.
	var hits []FXHit
	dot := func(name, dice, label string) {
		if st[name] > 0 {
			d := MustRoll(dice).Total
			dealt, note, down := f.damage(d, false)
			hits = append(hits, FXHit{ID: f.id(), Dmg: dealt, Status: name, Down: down})
			e.Log("combat", f.name(), fmt.Sprintf("%s %s takes %d %s damage. %s", map[string]string{"burn": "🔥", "bleed": "🩸", "poison": "☠️"}[name], f.name(), dealt, label, note))
		}
	}
	dot("burn", "2d6", "burn")
	dot("bleed", "1d8+2", "bleed")
	dot("poison", "2d4+2", "poison")
	if st["regen"] > 0 {
		h := f.heal(MustRoll(fmt.Sprintf("1d8+%d", f.level())).Total)
		hits = append(hits, FXHit{ID: f.id(), Heal: h})
	}
	if st["jackpot"] > 0 && f.c != nil {
		h := f.heal(f.c.MaxHP)
		f.c.CE = f.c.CEMax
		if h > 0 {
			hits = append(hits, FXHit{ID: f.id(), Heal: h, Status: "jackpot"})
		}
	}
	if f.c != nil {
		if f.c.Race == "half_curse" {
			if h := f.heal(2); h > 0 {
				hits = append(hits, FXHit{ID: f.id(), Heal: h})
			}
		}
		if !f.c.Tech().NoCE {
			f.c.CE = min(f.c.CEMax, f.c.CE+1)
		}
	}
	if len(hits) > 0 {
		e.fx(FX{Kind: "status", Src: f.id(), Hits: hits})
	}
	if st["haste"] > 0 {
		cb.ExtraAction = true
	}
	// Tick down statuses this fighter carries.
	stunned := st["stun"] > 0
	for k, v := range st {
		if v > 0 {
			st[k] = v - 1
		}
		if st[k] <= 0 {
			delete(st, k)
		}
	}
	if stunned {
		st["skip"] = 1
		if f.m != nil && f.m.Boss {
			st["resolve"] = 2 // bosses can't be stun-locked
		}
	}
	if !f.alive() {
		e.checkCombatEnd()
	}
}

func (e *Engine) collapseDomain(natural bool) {
	cb := e.C.Combat
	if cb == nil || cb.Domain == nil {
		return
	}
	d := cb.Domain
	cb.Domain = nil
	owner := e.fighterByID(d.Owner)
	if owner.ok() && natural {
		owner.status()["burnout"] = 2
		e.Log("combat", owner.name(), "🌀 "+d.Name+" collapses. "+owner.name()+"'s technique burns out for a moment.")
	} else {
		e.Log("combat", "", "💥 "+d.Name+" shatters!")
	}
	e.fx(FX{Kind: "domain_end", Src: d.Owner, Name: d.Name, Color: d.Color})
}

// runAutoTurns plays curse turns and skips those who can't act until a sorcerer is up.
func (e *Engine) runAutoTurns() {
	for guard := 0; guard < 300 && e.C.Combat != nil; guard++ {
		if e.checkCombatEnd() {
			return
		}
		cur := e.currentCombatant()
		f := e.fighterByID(cur.ID)
		if !f.ok() {
			e.advanceTurn()
			continue
		}
		if f.m != nil {
			if f.m.HP > 0 {
				if f.status()["skip"] > 0 {
					delete(f.status(), "skip")
					e.Log("combat", f.name(), "💫 "+f.name()+" is stunned and loses its turn.")
					e.fx(FX{Kind: "status", Src: f.id(), Hits: []FXHit{{ID: f.id(), Status: "stunned"}}})
				} else {
					e.monsterTurn(f.m)
				}
				if e.checkCombatEnd() {
					return
				}
			}
			e.advanceTurn()
			continue
		}
		ch := f.c
		if ch.Dead || ch.Fled {
			e.advanceTurn()
			continue
		}
		if ch.HP <= 0 {
			if !ch.Stable {
				e.Log("combat", ch.Name, "💀 "+ch.DeathSave())
				if ch.HP > 0 {
					return
				}
			}
			e.advanceTurn()
			continue
		}
		if ch.Status["skip"] > 0 {
			delete(ch.Status, "skip")
			e.Log("combat", ch.Name, "💫 "+ch.Name+" is stunned and loses the turn.")
			e.fx(FX{Kind: "status", Src: ch.ID, Hits: []FXHit{{ID: ch.ID, Status: "stunned"}}})
			e.advanceTurn()
			continue
		}
		return
	}
}

func (e *Engine) monsterTurn(m *Monster) {
	t := Bestiary[m.Template]
	self := fighter{m: m}
	if m.Escapes && m.HP*10 < m.MaxHP*3 {
		m.HP = 0
		m.XP /= 2
		e.Log("combat", m.Name, "🌫️ "+m.Name+" laughs, reshapes into something small and slips away!")
		e.fx(FX{Kind: "text", Src: m.ID, Text: m.Name + " escapes!"})
		e.C.Flags[m.Template+"_escaped"] = "1"
		return
	}
	if m.Domain != "" && !m.DomainUsed && m.HP*2 < m.MaxHP {
		m.DomainUsed = true
		e.applyAbility(self, AbilityList[m.Domain], "", false)
		return
	}
	if len(t.Moves) == 0 {
		return
	}
	mv := t.Moves[min(m.nextIdx, len(t.Moves)-1)]
	targets := e.side(true)
	if len(targets) == 0 {
		return
	}
	target := targets[rand.IntN(len(targets))]
	if chance(0.35) {
		sort.Slice(targets, func(i, j int) bool { return targets[i].hp() < targets[j].hp() })
		target = targets[0]
	}
	ab := Ability{ID: "curse_" + m.Template, Name: mv.Name, Effects: mv.Effects, VFX: mv.VFX, Color: mv.Color}
	e.applyAbility(self, ab, target.id(), false)
	e.planMove(m)
}

// checkCombatEnd ends the fight if one side is out. Returns true if it ended.
func (e *Engine) checkCombatEnd() bool {
	cb := e.C.Combat
	if cb == nil {
		return true
	}
	if len(e.liveMonsters()) == 0 {
		e.endCombat("victory")
		return true
	}
	anyUp, anyIn := false, false
	for _, ch := range e.battleChars() {
		if ch.Dead || ch.Fled {
			continue
		}
		anyIn = true
		if ch.HP > 0 {
			anyUp = true
		}
	}
	if !anyIn {
		e.endCombat("escaped")
		return true
	}
	if !anyUp {
		e.endCombat("defeat")
		return true
	}
	return false
}

// endCombat closes a fight: victory, escaped, defeat, surrender, truce.
func (e *Engine) endCombat(outcome string) {
	cb := e.C.Combat
	if cb == nil {
		return
	}
	party := e.battleChars()
	e.C.Combat = nil
	var lastXP, lastYen int
	defer func() {
		seq := 1
		if e.C.LastBattle != nil {
			seq = e.C.LastBattle.Seq + 1
		}
		e.C.LastBattle = &BattleResult{Seq: seq, Title: cb.Title, Outcome: outcome, Rounds: cb.Round}
		for _, m := range cb.Monsters {
			if m.HP <= 0 {
				e.C.LastBattle.Slain = append(e.C.LastBattle.Slain, m.Name)
			}
		}
		e.C.LastBattle.XP, e.C.LastBattle.Yen = lastXP, lastYen
	}()
	xp, yen := 0, 0
	var slain []string
	for _, m := range cb.Monsters {
		if m.HP <= 0 {
			slain = append(slain, m.Name)
			xp += m.XP
			if m.Yen != "" {
				yen += max(0, rollYen(m.Yen))
			}
		} else if outcome == "surrender" || outcome == "truce" {
			xp += m.XP / 2
		}
	}
	for _, ch := range party {
		ch.Status, ch.Used, ch.InBattle, ch.Fled = map[string]int{}, map[string]int{}, false, false
	}
	summary := ""
	switch outcome {
	case "victory":
		e.Log("combat", "", "🏆 Exorcism complete!")
		summary = "The sorcerers won. Exorcised: " + strings.Join(slain, ", ") + "."
	case "escaped":
		e.Log("combat", "", "🏃 The sorcerers retreat.")
		summary = "The party retreated. The curses remain."
		xp, yen = 0, 0
	case "defeat":
		e.Log("combat", "", "☠️ The sorcerers have fallen...")
		summary = "Every sorcerer in the fight was knocked out. The curses won."
		xp, yen = 0, 0
		for _, ch := range party {
			if !ch.Dead {
				ch.HP, ch.Stable, ch.DeathSaves, ch.DeathFails = 1, false, 0, 0
			}
		}
	default:
		e.Log("combat", "", "🕊️ The fight ends ("+outcome+").")
		summary = "The fight ended: " + outcome + "."
	}
	if xp > 0 {
		e.awardXPTo(party, xp, "for the exorcism")
	}
	if yen > 0 {
		e.splitYen(party, yen, "exorcism fee from Jujutsu HQ")
	}
	e.afterBattle(cb, outcome, party)
	lastXP, lastYen = xp, yen
	summary += fmt.Sprintf(" XP: %d. Yen: %d.", xp, yen)
	kind := "combat_end"
	if outcome == "defeat" {
		kind = "defeat"
	}
	e.enqueue(DMEvent{Kind: kind, Summary: outcome + ": " + summary, Quest: cb.QuestID})
}

func rollYen(expr string) int {
	// "4d6*1000"
	mult := 1
	if i := strings.Index(expr, "*"); i > 0 {
		fmt.Sscan(expr[i+1:], &mult)
		expr = expr[:i]
	}
	return MustRoll(expr).Total * mult
}

func (e *Engine) splitYen(chars []*Character, total int, reason string) {
	var living []*Character
	for _, ch := range chars {
		if !ch.Dead {
			living = append(living, ch)
		}
	}
	if len(living) == 0 || total == 0 {
		return
	}
	each := total / len(living)
	for _, ch := range living {
		ch.Gold = max(0, ch.Gold+each)
	}
	e.logf("system", "", "💴 ¥%s each (%s).", commas(each), reason)
}

func commas(n int) string {
	s := fmt.Sprint(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var out []string
	for len(s) > 3 {
		out = append([]string{s[len(s)-3:]}, out...)
		s = s[:len(s)-3]
	}
	out = append([]string{s}, out...)
	if neg {
		return "-" + strings.Join(out, ",")
	}
	return strings.Join(out, ",")
}

// CombatReq is a structured battle command from a player.
type CombatReq struct {
	Type    string `json:"type"` // ability, item, flee, end
	Target  string `json:"target"`
	Ability string `json:"ability"`
	Item    int    `json:"item"`
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

func (e *Engine) SkipTurn() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.C.Combat == nil {
		return
	}
	if cur := e.currentCombatant(); cur != nil && !cur.IsMonster {
		e.logf("combat", "", "⏭️ %s's turn is skipped.", cur.Name)
		e.advanceTurn()
		e.runAutoTurns()
		e.changed()
	}
}

func (e *Engine) combatAction(ch *Character, r CombatReq) error {
	cb := e.C.Combat
	if cb == nil {
		return fmt.Errorf("there's no battle right now")
	}
	cur := e.currentCombatant()
	if cur == nil || cur.ID != ch.ID {
		return fmt.Errorf("it's not your turn")
	}
	if !ch.Conscious() {
		return fmt.Errorf("you can't act right now")
	}
	usesAction, usesBonus := true, false
	switch r.Type {
	case "end":
		e.endTurn()
		return nil
	case "flee":
		if err := e.needAction(); err != nil {
			return err
		}
		rank := 0
		for _, m := range e.liveMonsters() {
			rank = max(rank, m.Rank)
		}
		res := ch.Check("acrobatics", 10+rank*2, 0)
		if res.Success {
			ch.Fled = true
			e.logf("combat", ch.Name, "🏃 %s breaks away! (%s)", ch.Name, res)
			e.fx(FX{Kind: "text", Src: ch.ID, Text: ch.Name + " escapes"})
		} else {
			e.logf("combat", ch.Name, "🏃 %s tries to run but is cut off. (%s)", ch.Name, res)
			e.fx(FX{Kind: "text", Src: ch.ID, Text: "Can't escape!"})
		}
	case "item":
		if err := e.needAction(); err != nil {
			return err
		}
		if err := e.useItem(ch, r.Item, r.Target); err != nil {
			return err
		}
	case "ability":
		ab, ok := AbilityList[r.Ability]
		if !ok || !ch.Knows(r.Ability) || ab.Passive {
			return fmt.Errorf("you can't use that")
		}
		if ab.Bonus {
			if cb.BonusUsed {
				return fmt.Errorf("you've already used your bonus action")
			}
		} else if err := e.needAction(); err != nil {
			return err
		}
		if err := e.payFor(ch, ab); err != nil {
			return err
		}
		e.applyAbility(fighter{c: ch}, ab, r.Target, false)
		if ab.Bonus {
			usesAction, usesBonus = false, true
		}
	default:
		return fmt.Errorf("unknown action")
	}
	if e.C.Combat == nil {
		e.changed()
		return nil
	}
	if usesBonus {
		cb.BonusUsed = true
	}
	if usesAction {
		if cb.ExtraAction {
			cb.ExtraAction = false
		} else {
			cb.ActionUsed = true
		}
	}
	if e.checkCombatEnd() {
		e.changed()
		return nil
	}
	if ch.Fled || !ch.Conscious() || (cb.ActionUsed && (cb.BonusUsed || !e.hasBonusOption(ch))) {
		e.endTurn()
	}
	e.changed()
	return nil
}

func (e *Engine) payFor(ch *Character, ab Ability) error {
	if ab.ID == "overdrive" {
		if cb := e.C.Combat; cb.Flow < cb.FlowMax {
			return fmt.Errorf("the Flow meter isn't full (%d/%d)", cb.Flow, cb.FlowMax)
		}
		e.C.Combat.Flow = 0
		return nil
	}
	if ab.Tech != "" && ch.Status["burnout"] > 0 && !ab.Universal {
		return fmt.Errorf("your technique is burnt out — use universal moves this turn")
	}
	cost := ch.AbilityCost(ab)
	if ch.Status["jackpot"] > 0 {
		cost = 0
	}
	if cost > ch.CE {
		return fmt.Errorf("not enough cursed energy (%d needed, %d left)", cost, ch.CE)
	}
	if ab.PerBattle > 0 && ch.Used[ab.ID] >= ab.PerBattle {
		return fmt.Errorf("%s can only be used %d time(s) per battle", ab.Name, ab.PerBattle)
	}
	if ab.HPCost > 0 && ch.HP <= ab.HPCost {
		return fmt.Errorf("you don't have the HP to pay for %s", ab.Name)
	}
	ch.CE -= cost
	if ab.PerBattle > 0 {
		ch.Used[ab.ID]++
	}
	if ab.HPCost > 0 {
		ch.HP -= ab.HPCost
	}
	return nil
}

func (e *Engine) needAction() error {
	if e.C.Combat.ActionUsed && !e.C.Combat.ExtraAction {
		return fmt.Errorf("you've used your action — use a bonus action or end your turn")
	}
	return nil
}

func (e *Engine) hasBonusOption(ch *Character) bool {
	for _, a := range ch.KnownAbilities() {
		if !a.Bonus {
			continue
		}
		if ch.AbilityCost(a) <= ch.CE && (a.PerBattle == 0 || ch.Used[a.ID] < a.PerBattle) && ch.HP > a.HPCost {
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

func (e *Engine) useItem(ch *Character, idx int, targetID string) error {
	if idx < 0 || idx >= len(ch.Inventory) {
		return fmt.Errorf("no such item")
	}
	it := ch.Inventory[idx]
	if it.Kind != "consumable" {
		return fmt.Errorf("%s can't be used in battle", it.Name)
	}
	if it.Damage != "" {
		t := e.monster(targetID)
		if t == nil || t.HP <= 0 {
			if lm := e.liveMonsters(); len(lm) > 0 {
				t = lm[0]
			} else {
				return fmt.Errorf("no target")
			}
		}
		ch.RemoveItem(it.ID, 1)
		d := MustRoll(it.Damage).Total
		_, _, down := fighter{m: t}.damage(d, false)
		e.logf("combat", ch.Name, "📜 %s slaps an %s on %s: %d damage.", ch.Name, it.Name, t.Name, d)
		e.fx(FX{Kind: "move", Src: ch.ID, Name: it.Name, VFX: "talisman", Color: "#fff3b0", Hits: []FXHit{{ID: t.ID, Dmg: d, Down: down}}})
		return nil
	}
	target := ch
	if t, ok := e.C.Characters[targetID]; ok && !t.Dead {
		target = t
	}
	ch.RemoveItem(it.ID, 1)
	hit := FXHit{ID: target.ID}
	msg := fmt.Sprintf("🧪 %s uses %s on %s:", ch.Name, it.Name, target.Name)
	if it.Heal != "" {
		hit.Heal = target.heal(MustRoll(it.Heal).Total)
		msg += fmt.Sprintf(" +%d HP", hit.Heal)
	}
	if it.CE > 0 {
		target.CE = min(target.CEMax, target.CE+it.CE)
		msg += fmt.Sprintf(" +%d CE", it.CE)
	}
	e.Log("combat", ch.Name, msg+".")
	e.fx(FX{Kind: "move", Src: ch.ID, Name: it.Name, VFX: "heal", Color: "#95d5b2", Hits: []FXHit{hit}})
	return nil
}

func tiers(level int) int {
	n := 0
	for _, l := range []int{5, 9, 12} {
		if level >= l {
			n++
		}
	}
	return n
}

// diceFor builds the damage expression for an effect used by src.
func (e *Engine) diceFor(src fighter, eff Effect) (expr string, mod int, weapon *Item) {
	expr = eff.Dice
	if strings.Contains(expr, "weapon") {
		if src.c != nil {
			w := src.c.Weapon()
			weapon = &w
			expr = strings.ReplaceAll(expr, "weapon", w.Damage)
			if eff.AddMod {
				mod = src.c.Mod(src.c.AttackStat(w)) + w.Bonus
				if src.c.Race == "zenin" {
					mod += 2
				}
			}
		} else {
			expr = strings.ReplaceAll(expr, "weapon", "1d8")
		}
	} else if eff.AddMod && src.c != nil {
		mod = src.c.CastMod()
	}
	if eff.Scale != "" && src.c != nil {
		for i := 0; i < tiers(src.c.Level); i++ {
			expr += "+" + eff.Scale
		}
	}
	return
}

func (e *Engine) damageMult(src, tgt fighter, technique bool) float64 {
	m := 1.0
	if src.has("empowered") {
		m *= 1.25
	}
	if src.has("weakened") {
		m *= 0.75
	}
	if tgt.has("exposed") {
		m *= 1.25
	}
	if tgt.has("infinity") {
		m *= 0.5
	}
	if tgt.has("guard") {
		m *= 0.5
	}
	if src.m != nil && src.m.Power > 0 {
		m *= math.Max(0.35, float64(src.m.Power)/100)
	}
	if technique && tgt.c != nil && tgt.c.Race == "half_curse" {
		m *= 0.75
	}
	return m
}

func (e *Engine) applyStatus(src, tgt fighter, status string, rounds int, fromDomain bool) string {
	if status == "" || rounds <= 0 || !tgt.ok() {
		return ""
	}
	st := tgt.status()
	if status == "stun" && !fromDomain && st["resolve"] > 0 {
		return ""
	}
	if status == "stun" && tgt.m != nil && tgt.m.Boss && !fromDomain {
		rounds = 1
	}
	if rounds > st[status] {
		st[status] = rounds
	}
	return status
}

// applyAbility resolves an ability for any fighter. Caller holds the lock.
func (e *Engine) applyAbility(src fighter, ab Ability, targetID string, domainPulse bool) {
	cb := e.C.Combat
	if cb == nil || !src.ok() {
		return
	}
	enemies := e.side(!src.party())
	allies := e.side(src.party())
	pick := func(list []fighter, id string, fallback fighter) fighter {
		for _, f := range list {
			if f.id() == id {
				return f
			}
		}
		if fallback.ok() {
			return fallback
		}
		if len(list) > 0 {
			return list[0]
		}
		return fighter{}
	}
	firstEnemy := fighter{}
	if len(enemies) > 0 {
		firstEnemy = enemies[0]
	}

	// Domain cast: set up the domain state (handles clashes), then pulse once.
	if ab.Domain && !domainPulse {
		if ab.Rounds > 1 {
			if d := cb.Domain; d != nil && d.Party != src.party() {
				owner := e.fighterByID(d.Owner)
				a := Die(20) + src.level()
				b := Die(20) + owner.level()
				e.fx(FX{Kind: "domain", Src: src.id(), Name: "DOMAIN CLASH", VFX: "clash", Color: ab.Color, Text: fmt.Sprintf("%d vs %d", a, b)})
				if a < b {
					e.logf("combat", src.name(), "🌀 %s expands %s — but %s's domain overwhelms it (%d vs %d)!", src.name(), ab.Name, owner.name(), a, b)
					return
				}
				e.logf("combat", src.name(), "🌀 DOMAIN CLASH! %s's %s engulfs %s's (%d vs %d)!", src.name(), ab.Name, d.Name, a, b)
				e.collapseDomain(false)
			}
			cb.Domain = &DomainState{Owner: src.id(), Ability: ab.ID, Rounds: ab.Rounds - 1, Party: src.party(), Name: ab.Name, VFX: ab.VFX, Color: ab.Color}
		}
		e.logf("combat", src.name(), "🌀 %s: \"%s\"", src.name(), ab.Name)
		e.fx(FX{Kind: "domain", Src: src.id(), Name: ab.Name, VFX: ab.VFX, Color: ab.Color})
	}

	var hits []FXHit
	var lines []string
	bf := false
	for _, eff := range ab.Effects {
		var targets []fighter
		switch eff.Target {
		case "self":
			targets = []fighter{src}
		case "ally":
			targets = []fighter{pick(allies, targetID, src)}
		case "all_allies":
			targets = allies
		case "all_enemies":
			targets = enemies
		default:
			if t := pick(enemies, targetID, firstEnemy); t.ok() {
				targets = []fighter{t}
			}
		}
		for _, tgt := range targets {
			if !tgt.ok() {
				continue
			}
			// Sure-hit can't touch those inside a Simple Domain or with no cursed energy.
			if domainPulse && tgt.party() != src.party() && (tgt.has("simple_domain") || (tgt.c != nil && tgt.c.Tech().NoCE)) {
				hits = append(hits, FXHit{ID: tgt.id(), Miss: true, Status: "nullified"})
				lines = append(lines, tgt.name()+" nullifies the sure-hit")
				continue
			}
			h, line, blackFlash := e.resolveEffect(src, tgt, eff, ab, domainPulse)
			hits = append(hits, h...)
			if line != "" {
				lines = append(lines, line)
			}
			bf = bf || blackFlash
		}
	}
	kind := "move"
	if bf {
		kind = "blackflash"
	}
	name := ab.Name
	if domainPulse {
		name = ab.Name
	}
	e.fx(FX{Kind: kind, Src: src.id(), Name: name, VFX: ab.VFX, Color: ab.Color, Hits: hits})
	icon := "✨"
	if src.m != nil {
		icon = "👹"
	}
	if domainPulse {
		icon = "🌀"
	}
	text := fmt.Sprintf("%s %s uses %s", icon, src.name(), ab.Name)
	if len(lines) > 0 {
		text += ": " + strings.Join(lines, "; ")
	}
	e.Log("combat", src.name(), text+".")
	e.checkCombatEnd()
}

// resolveEffect applies one effect to one target.
func (e *Engine) resolveEffect(src, tgt fighter, eff Effect, ab Ability, domainPulse bool) ([]FXHit, string, bool) {
	technique := ab.Tech != "" || src.m != nil
	switch eff.Kind {
	case "heal":
		expr, mod, _ := e.diceFor(src, eff)
		amt := MustRoll(expr).Total + mod
		if ab.ID == "rct" && src.c != nil {
			amt += src.c.Level
		}
		got := tgt.heal(amt)
		return []FXHit{{ID: tgt.id(), Heal: got}}, fmt.Sprintf("%s heals %d", tgt.name(), got), false
	case "status":
		s := e.applyStatus(src, tgt, eff.Status, eff.Rounds, domainPulse)
		if s == "" {
			return []FXHit{{ID: tgt.id(), Miss: true}}, tgt.name() + " resists", false
		}
		return []FXHit{{ID: tgt.id(), Status: s}}, fmt.Sprintf("%s is %s", tgt.name(), statusWord(s)), false
	case "cleanse":
		for _, b := range buffs {
			delete(tgt.status(), b)
		}
		if d := e.C.Combat.Domain; d != nil && d.Owner == tgt.id() {
			e.collapseDomain(false)
		}
		return []FXHit{{ID: tgt.id(), Status: "dispelled"}}, tgt.name() + "'s techniques are nullified", false
	case "gamble":
		spin := Die(6)
		if spin <= 2 {
			tgt.status()["jackpot"] = 99
			tgt.status()["empowered"] = 99
			if tgt.c != nil {
				tgt.c.CE = tgt.c.CEMax
			}
			got := tgt.heal(tgt.maxHP())
			e.fx(FX{Kind: "text", Src: tgt.id(), Text: "JACKPOT!", Color: "#06d6a0"})
			return []FXHit{{ID: tgt.id(), Heal: got, Status: "jackpot"}}, "🎰 JACKPOT! Infinite cursed energy, a body that heals itself", false
		}
		tgt.status()["empowered"] = max(tgt.status()["empowered"], 1)
		return []FXHit{{ID: tgt.id(), Status: "riichi"}}, fmt.Sprintf("🎰 the reels stop on %d... no jackpot this spin", Die(9)*111), false
	case "copy":
		var pool []Ability
		for _, ch := range e.battleChars() {
			if src.c != nil && ch.ID == src.c.ID {
				continue
			}
			for _, a := range ch.KnownAbilities() {
				if a.Tech != "" && !a.Domain && a.Tech != "cursed_partner" {
					pool = append(pool, a)
				}
			}
		}
		if len(pool) == 0 {
			for _, id := range SortedTechniqueIDs() {
				if id == "cursed_partner" {
					continue
				}
				for _, mid := range Techniques[id].Moves {
					if a := AbilityList[mid]; a.MinLevel <= src.level() {
						pool = append(pool, a)
					}
				}
			}
		}
		if len(pool) == 0 {
			return nil, "", false
		}
		c := pool[rand.IntN(len(pool))]
		e.fx(FX{Kind: "text", Src: src.id(), Text: "Copy: " + c.Name, Color: c.Color})
		var hits []FXHit
		var lines []string
		bf := false
		for _, ce := range c.Effects {
			t := tgt
			switch ce.Target {
			case "self", "ally", "all_allies":
				t = src
			}
			if !t.alive() && ce.Target == "enemy" {
				if en := e.side(!src.party()); len(en) > 0 {
					t = en[0]
				}
			}
			targets := []fighter{t}
			if ce.Target == "all_enemies" {
				targets = e.side(!src.party())
			}
			for _, tt := range targets {
				h, l, b := e.resolveEffect(src, tt, ce, c, false)
				hits = append(hits, h...)
				if l != "" {
					lines = append(lines, l)
				}
				bf = bf || b
			}
		}
		return hits, "copied " + c.Name + " — " + strings.Join(lines, ", "), bf
	}

	// Damage-dealing kinds: attack, save, auto.
	hitsN := max(1, eff.Hits)
	var out []FXHit
	var parts []string
	blackFlash := false
	for i := 0; i < hitsN; i++ {
		if !tgt.alive() && tgt.m != nil {
			break
		}
		expr, mod, weapon := e.diceFor(src, eff)
		crit, landed, saved := false, true, false
		detail := ""
		switch eff.Kind {
		case "attack":
			adv := 0
			if tgt.has("exposed") || tgt.has("stun") || tgt.has("skip") || src.has("empowered") {
				adv++
			}
			if tgt.has("infinity") || tgt.has("guard") || src.has("weakened") {
				adv--
			}
			adv = max(-1, min(1, adv))
			roll, _ := D20(adv)
			bonus := 0
			if src.c != nil {
				if weapon != nil {
					bonus = src.c.AttackBonus(*weapon)
				} else {
					bonus = src.c.SpellAttack()
				}
			} else {
				bonus = src.m.ToHit
			}
			if src.has("empowered") {
				bonus += 2
			}
			critOn := 20
			if src.c != nil {
				if src.has("zone") {
					critOn = 19
				}
				if ab.ID == "ratio_strike" || ab.ID == "ratio_finale" {
					critOn = min(critOn, 18)
				}
			}
			total := roll + bonus
			sure := e.C.Combat.Domain != nil && e.C.Combat.Domain.Owner == src.id()
			ignore := weapon != nil && weapon.Special == "ignore_ac"
			crit = roll >= critOn
			landed = crit || sure || ignore || (roll != 1 && total >= tgt.ac())
			detail = fmt.Sprintf("%d vs AC %d", total, tgt.ac())
			if sure {
				detail = "sure-hit"
			}
		case "save":
			ok, _ := tgt.save(eff.Save, src.dc())
			saved = ok
			detail = fmt.Sprintf("%s save %s vs DC %d", AbilityNames[eff.Save], map[bool]string{true: "made", false: "failed"}[ok], src.dc())
			if ok && !eff.Half {
				landed = false
			}
		}
		if !landed {
			out = append(out, FXHit{ID: tgt.id(), Miss: true, Saved: saved})
			if eff.Kind == "attack" {
				parts = append(parts, fmt.Sprintf("misses %s (%s)", tgt.name(), detail))
			} else {
				parts = append(parts, fmt.Sprintf("%s resists (%s)", tgt.name(), detail))
			}
			continue
		}
		dmg := 0
		bfHit := false
		if expr != "" {
			r := MustRoll(expr)
			base := r.Total
			if crit {
				base += MustRoll(DiceOnly(expr)).Total
			}
			if crit && eff.Melee && src.c != nil {
				// BLACK FLASH: cursed energy within 0.000001 seconds of the blow.
				bfHit = true
				blackFlash = true
				base = int(float64(base) * 2.5)
				src.c.CE = min(src.c.CEMax, src.c.CE+3)
				src.status()["zone"] = 99
			}
			dmg = base + mod
			if eff.PctMax > 0 {
				dmg += tgt.maxHP() * eff.PctMax / 100
			}
			if tgt.has("marked") && src.party() {
				dmg += Die(8)
			}
			if ab.ID == "resonance" && tgt.has("marked") {
				dmg += MustRoll("3d8").Total
			}
			if saved && eff.Half {
				dmg /= 2
			}
			dmg = int(math.Round(float64(dmg) * e.damageMult(src, tgt, technique)))
			if dmg < 1 {
				dmg = 1
			}
		}
		dealt, note, down := 0, "", false
		if dmg > 0 {
			dealt, note, down = tgt.damage(dmg, crit)
		}
		status := ""
		if eff.Status != "" && !(saved && eff.Kind == "save") {
			status = e.applyStatus(src, tgt, eff.Status, eff.Rounds, domainPulse)
		}
		if weapon != nil && weapon.Special == "dispel" {
			for _, b := range buffs {
				delete(tgt.status(), b)
			}
		}
		out = append(out, FXHit{ID: tgt.id(), Dmg: dealt, Crit: crit, BF: bfHit, Status: status, Down: down, Saved: saved})
		p := ""
		switch {
		case bfHit:
			p = fmt.Sprintf("⚫⚡ BLACK FLASH on %s for %d", tgt.name(), dealt)
		case crit:
			p = fmt.Sprintf("CRITICAL on %s for %d", tgt.name(), dealt)
		case dmg > 0:
			p = fmt.Sprintf("%s takes %d", tgt.name(), dealt)
		default:
			p = tgt.name()
		}
		if eff.DmgType != "" && dmg > 0 {
			p += " " + eff.DmgType
		}
		if detail != "" {
			p += " (" + detail + ")"
		}
		if status != "" {
			p += ", " + statusWord(status)
		}
		if down {
			if tgt.m != nil {
				p += " — exorcised!"
			} else if note != "" {
				p += " — " + note
			}
		}
		parts = append(parts, p)
	}
	return out, strings.Join(parts, ", "), blackFlash
}

// DiceOnly strips flat modifiers: "2d6+3" -> "2d6".
func DiceOnly(expr string) string {
	clean := strings.ReplaceAll(expr, " ", "")
	var out []string
	for _, m := range diceTerm.FindAllStringSubmatch(clean, -1) {
		if m[3] != "" {
			n := m[2]
			if n == "" {
				n = "1"
			}
			out = append(out, n+"d"+m[3])
		}
	}
	if len(out) == 0 {
		return "0"
	}
	return strings.Join(out, "+")
}

func statusWord(s string) string {
	switch s {
	case "stun":
		return "stunned"
	case "exposed":
		return "exposed"
	case "weakened":
		return "weakened"
	case "marked":
		return "marked by a nail"
	case "burn":
		return "burning"
	case "bleed":
		return "bleeding"
	case "poison":
		return "poisoned"
	case "infinity":
		return "protected by Infinity"
	case "guard":
		return "guarded"
	case "empowered":
		return "empowered"
	case "haste":
		return "hasted"
	case "zone":
		return "in the zone"
	case "regen":
		return "regenerating"
	case "simple_domain":
		return "inside a Simple Domain"
	case "jackpot":
		return "on a JACKPOT"
	}
	return s
}

// UseAbilityOutOfCombat allows healing techniques between fights.
func (e *Engine) UseAbilityOutOfCombat(token, ability, target string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ch, err := e.charOf(token)
	if err != nil {
		return err
	}
	if e.C.Combat != nil && ch.InBattle {
		return e.combatAction(ch, CombatReq{Type: "ability", Ability: ability, Target: target})
	}
	ab, ok := AbilityList[ability]
	if !ok || !ch.Knows(ability) {
		return fmt.Errorf("you can't use that")
	}
	healing := false
	for _, ef := range ab.Effects {
		if ef.Kind == "heal" {
			healing = true
		}
	}
	if !healing {
		return fmt.Errorf("%s is for battle", ab.Name)
	}
	if !ch.Conscious() {
		return fmt.Errorf("you're unconscious")
	}
	if err := e.payFor(ch, ab); err != nil {
		return err
	}
	tgt := ch
	if t, ok := e.C.Characters[target]; ok && !t.Dead {
		tgt = t
	}
	for _, ef := range ab.Effects {
		if ef.Kind == "heal" {
			expr, mod, _ := e.diceFor(fighter{c: ch}, ef)
			amt := MustRoll(expr).Total + mod
			if ab.ID == "rct" {
				amt += ch.Level
				tgt = ch
			}
			e.logf("system", ch.Name, "💚 %s uses %s: %s recovers %d HP.", ch.Name, ab.Name, tgt.Name, tgt.heal(amt))
		}
	}
	e.changed()
	return nil
}

// CombatSummary describes the fight for the DM prompt.
func (e *Engine) CombatSummary() string {
	cb := e.C.Combat
	if cb == nil {
		return "No battle."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s — round %d. Turn: %s.\n", cb.Title, cb.Round, e.currentCombatant().Name)
	for _, m := range cb.Monsters {
		fmt.Fprintf(&b, "- %s [%s] (%s) HP %d/%d\n", m.Name, m.ID, GradeNames[m.Rank], m.HP, m.MaxHP)
	}
	if cb.Domain != nil {
		fmt.Fprintf(&b, "Active domain: %s (%d rounds left)\n", cb.Domain.Name, cb.Domain.Rounds)
	}
	return b.String()
}

func (e *Engine) awardXPTo(chars []*Character, total int, reason string) {
	var living []*Character
	for _, ch := range chars {
		if !ch.Dead {
			living = append(living, ch)
		}
	}
	if total <= 0 || len(living) == 0 {
		return
	}
	each := max(1, total/len(living))
	msg := fmt.Sprintf("✨ +%d XP each", each)
	if reason != "" {
		msg += " — " + reason
	}
	e.Log("system", "", msg+".")
	for _, ch := range living {
		for _, m := range ch.GainXP(each) {
			e.Log("level", ch.Name, m)
		}
	}
}

// hurtOutsideCombat applies hazard damage; a sorcerer dropped outside a fight is stabilized by the party.
func (e *Engine) hurtOutsideCombat(ch *Character, dmg int) string {
	note := ch.TakeDamage(dmg, false)
	if !ch.Dead && ch.HP <= 0 {
		ch.Stable = true
		note = strings.TrimSpace(note + " They're unconscious but stable — Shoko or a rest will fix it.")
	}
	return note
}
