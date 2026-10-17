package engine

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"sort"
	"strings"
)

// OfflineDM runs a procedurally generated adventure with no network access.
// All methods are called with the engine lock held.
type OfflineDM struct{}

func (d *OfflineDM) Handle(e *Engine, ev DMEvent) {
	c := e.C
	switch ev.Kind {
	case "start":
		if c.Mode == "ai" && c.Adventure == nil {
			e.Log("dm", "DM", "The tale begins at "+c.Location+". Tell the Dungeon Master what you'd like to do.")
			return
		}
		d.ensureAdventure(e)
		d.intro(e)
	case "join":
		if ch, ok := c.Characters[ev.Summary]; ok {
			e.Log("dm", "DM", fmt.Sprintf("%s, %s, catches up with the party. %s", ch.Name, article(Races[ch.Race].Name+" "+Classes[ch.Class].Name), pick([]string{
				"They look ready for trouble.", "Introductions can wait — there's work to do.", "The party gains a new ally."})))
		}
	case "combat_end":
		d.afterCombat(e, ev.Summary)
	case "defeat":
		d.afterDefeat(e)
	case "actions":
		for _, a := range ev.Actions {
			ch, ok := c.Characters[a.CharID]
			if !ok || ch.Dead {
				continue
			}
			d.resolve(e, ch, a.Text)
		}
	}
}

func article(s string) string {
	if s != "" && strings.ContainsRune("AEIOUaeiou", rune(s[0])) {
		return "an " + s
	}
	return "a " + s
}

func (d *OfflineDM) ensureAdventure(e *Engine) {
	if e.C.Adventure == nil {
		e.C.Adventure = d.generate(e, 1)
	}
}

func (d *OfflineDM) generate(e *Engine, chapter int) *Adventure {
	themeID := ThemeOrder[rand.IntN(len(ThemeOrder))]
	if e.C.Adventure != nil {
		for themeID == e.C.Adventure.Theme {
			themeID = ThemeOrder[rand.IntN(len(ThemeOrder))]
		}
	}
	th := Themes[themeID]
	place := pick(th.Places)
	a := &Adventure{Theme: themeID, Title: fmt.Sprintf(pick(th.Titles), place), Hook: pick(th.Hooks), Lore: pick(th.Lore), Chapter: chapter}
	lvl := e.PartyLevel()
	size := max(1, len(e.livingChars()))
	budget := EncounterBudget[min(lvl, len(EncounterBudget))-1] * size

	town := pick(Towns)
	a.Rooms = append(a.Rooms, Room{Name: "The town of " + town, Town: true, Visited: true, NPC: "merchant",
		Desc: fmt.Sprintf("The town of %s bustles around the Crossroads Inn. A merchant hawks potions from a cart (\"buy potion\", 50 gp; \"buy greater potion\", 150 gp), and the inn offers a safe long rest. When you're ready, set out for %s.", town, a.Title)})

	entry := pick(th.Entry)
	a.Rooms = append(a.Rooms, Room{Name: entry.Name, Desc: entry.Desc})
	if chance(0.5) {
		a.Rooms[1].Encounter = d.encounter(themeID, budget/2, size)
	}
	pool := append([]roomDef(nil), th.Rooms...)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	n := 3 + rand.IntN(2)
	tier := min(2, (lvl-1)/4)
	usedNPC := false
	for i := 0; i < n && i < len(pool); i++ {
		r := Room{Name: pool[i].Name, Desc: pool[i].Desc}
		roll := rand.Float64()
		switch {
		case roll < 0.45:
			r.Encounter = d.encounter(themeID, budget, size)
			if chance(0.3) {
				r.Gold = MustRoll(fmt.Sprintf("%dd10", 2+lvl)).Total
			}
		case roll < 0.65:
			r.Trap = fmt.Sprint(rand.IntN(len(Traps)))
			if chance(0.5) {
				r.Loot = []string{pick(LootTiers[tier])}
			}
		case roll < 0.82:
			r.Chest = true
			r.Locked = chance(0.6)
			r.Loot = []string{pick(LootTiers[tier]), "potion_healing"}
			r.Gold = MustRoll(fmt.Sprintf("%dd10+10", 3+lvl)).Total
			if chance(0.3) {
				r.Trap = fmt.Sprint(rand.IntN(len(Traps)))
			}
		case roll < 0.92 && !usedNPC:
			usedNPC = true
			r.NPC = fmt.Sprint(rand.IntN(len(NPCs)))
		default:
			r.Gold = MustRoll("2d10").Total
		}
		a.Rooms = append(a.Rooms, r)
	}
	lair := pick(th.Lairs)
	boss := Room{Name: lair.Name, Desc: lair.Desc, Chest: true, Loot: []string{pick(LootTiers[tier]), pick(LootTiers[min(2, tier+1)])},
		Gold: MustRoll(fmt.Sprintf("%dd10+20", 4+lvl*2)).Total}
	boss.Encounter, a.Boss = d.bossEncounter(themeID, budget, size)
	a.Rooms = append(a.Rooms, boss)
	for i := 1; i < len(a.Rooms)-1; i++ {
		next := a.Rooms[i+1]
		if len(next.Encounter) > 0 {
			a.Rooms[i].Hint = "From up ahead you hear " + pick(th.Sounds) + "."
		} else if next.Trap != "" {
			a.Rooms[i].Hint = "The way ahead is eerily quiet. Too quiet — watch your step."
		} else {
			a.Rooms[i].Hint = "You hear nothing ahead but your own breathing."
		}
	}
	return a
}

func themeMonsters(theme string, boss bool) []MonsterTemplate {
	var out []MonsterTemplate
	for _, id := range SortedMonsterIDs() {
		t := Bestiary[id]
		if t.Boss != boss {
			continue
		}
		for _, tag := range t.Tags {
			if tag == theme {
				out = append(out, t)
				break
			}
		}
	}
	return out
}

func (d *OfflineDM) encounter(theme string, budget, size int) []string {
	cands := themeMonsters(theme, false)
	var fit []MonsterTemplate
	for _, t := range cands {
		if t.XP <= budget {
			fit = append(fit, t)
		}
	}
	if len(fit) == 0 {
		fit = cands[:1]
	}
	primary := pick(fit)
	count := 1
	for count < min(size+1, 6) && groupXP(primary.XP*(count+1), count+1) <= budget {
		count++
	}
	if count > 1 && chance(0.3) {
		count--
	}
	var out []string
	for i := 0; i < count; i++ {
		out = append(out, primary.ID)
	}
	for _, t := range fit {
		if t.ID != primary.ID && groupXP(primary.XP*count+t.XP, count+1) <= budget && count+1 <= size+1 && chance(0.5) {
			out = append(out, t.ID)
			break
		}
	}
	return out
}

// groupXP applies the 5e multiplier for fighting several monsters at once.
func groupXP(xp, n int) int {
	switch {
	case n <= 1:
		return xp
	case n == 2:
		return xp * 3 / 2
	case n <= 6:
		return xp * 2
	}
	return xp * 5 / 2
}

func (d *OfflineDM) bossEncounter(theme string, budget, size int) ([]string, string) {
	var best *MonsterTemplate
	for _, t := range themeMonsters(theme, true) {
		if t.XP <= budget*2 {
			tt := t
			if best == nil || tt.XP > best.XP {
				best = &tt
			}
		}
	}
	var out []string
	name := ""
	if best != nil {
		out = append(out, best.ID)
		name = best.Name
	} else {
		// Too weak for a true boss: crown the toughest local as a chieftain.
		minions := themeMonsters(theme, false)
		pickT := minions[0]
		for _, t := range minions {
			if t.XP*3+50 <= budget*2 {
				pickT = t
			}
		}
		out = append(out, pickT.ID+"*")
		name = "a " + pickT.Name + " chieftain"
	}
	spent := 0
	if best != nil {
		spent = best.XP
	} else {
		spent = Bestiary[strings.TrimSuffix(out[0], "*")].XP*3 + 50
	}
	if left := budget*2 - spent; left > 0 {
		out = append(out, d.encounter(theme, left/2, size)...)
	}
	return out, name
}

func spawnsFrom(ids []string) []Spawn {
	counts := map[string]int{}
	var order []string
	for _, id := range ids {
		if counts[id] == 0 {
			order = append(order, id)
		}
		counts[id]++
	}
	var out []Spawn
	for _, id := range order {
		out = append(out, Spawn{Type: strings.TrimSuffix(id, "*"), Count: counts[id], Elite: strings.HasSuffix(id, "*")})
	}
	return out
}

func (d *OfflineDM) room(e *Engine) *Room {
	a := e.C.Adventure
	if a == nil || a.Index >= len(a.Rooms) {
		return nil
	}
	return &a.Rooms[a.Index]
}

func (d *OfflineDM) intro(e *Engine) {
	a := e.C.Adventure
	th := Themes[a.Theme]
	e.C.Quest = a.Title
	r := d.room(e)
	e.C.Location = r.Name
	e.Log("dm", "DM", fmt.Sprintf("**Chapter %d: %s**\n\n%s %s\n\n%s", a.Chapter, a.Title, th.Intro, a.Hook, r.Desc))
	if a.Chapter == 1 {
		e.Log("system", "", "🎲 Offline DM tips: type what your character does — e.g. \"set out\", \"look around\", \"search the room\", \"listen at the door\", \"sneak ahead\", \"talk to the stranger\", \"open the chest\", \"pick the lock\", \"disarm the trap\", \"buy potion\", \"short rest\", \"long rest\". In combat, use the action buttons on your turn.")
	}
}

func (d *OfflineDM) enter(e *Engine, idx int) {
	a := e.C.Adventure
	a.Index = idx
	r := d.room(e)
	e.C.Location = r.Name
	first := !r.Visited
	r.Visited = true
	text := fmt.Sprintf("**%s**\n\n%s", r.Name, r.Desc)
	if !first {
		text = fmt.Sprintf("**%s** (again)\n\n%s", r.Name, r.Desc)
	}
	if r.NPC != "" && !r.Town {
		n := NPCs[atoi(r.NPC)]
		text += " " + n.Desc
	}
	if r.Chest && !r.ChestOpen {
		text += " A sturdy chest sits against the far wall."
	}
	e.Log("dm", "DM", text)

	if r.Town {
		return
	}
	if len(r.Encounter) > 0 && !r.Cleared {
		surprise := ""
		if a.Sneaking {
			surprise = "party"
			e.Log("dm", "DM", "Thanks to your stealth, the enemies haven't noticed you yet — you strike first!")
		} else if chance(0.2) {
			surprise = "enemies"
			e.Log("dm", "DM", "It's an ambush! The enemies were waiting for you.")
		}
		a.Sneaking = false
		e.Log("dm", "DM", d.foesLine(r.Encounter, idx == len(a.Rooms)-1))
		if _, err := e.StartCombat(spawnsFrom(r.Encounter), surprise); err != nil {
			e.Log("system", "", "Combat failed to start: "+err.Error())
		}
		return
	}
	a.Sneaking = false
	if r.Trap != "" && !r.TrapFound && !r.TrapDone {
		t := Traps[atoi(r.Trap)]
		best := d.bestAt(e, "perception")
		res := best.Check("perception", t.DC, 0)
		if res.Success {
			r.TrapFound = true
			e.Log("roll", best.Name, "🎲 "+res.String())
			e.Log("dm", "DM", best.Name+" stops the party just in time. "+t.Found+" You'll need to disarm it or carefully get past.")
		} else {
			d.springTrap(e, r, t, []*Character{pick(e.consciousChars())})
		}
	}
	if !r.Cleared && len(r.Encounter) == 0 && r.Trap == "" && !r.Chest && r.NPC == "" {
		r.Cleared = true
	}
}

func atoi(s string) int {
	n := 0
	fmt.Sscan(s, &n)
	return n
}

func (d *OfflineDM) foesLine(ids []string, boss bool) string {
	sp := spawnsFrom(ids)
	var parts []string
	for _, s := range sp {
		t := Bestiary[s.Type]
		name := t.Name
		if s.Elite {
			name = "a hulking " + t.Name + " chieftain"
		} else if s.Count > 1 {
			name = fmt.Sprintf("%d %ss", s.Count, t.Name)
		} else {
			name = article(t.Name)
		}
		parts = append(parts, name)
	}
	if boss {
		return "The master of this place rises before you: " + strings.Join(parts, ", ") + ". Roll for initiative!"
	}
	return pick([]string{"Enemies! ", "Out of the darkness come ", "You're not alone — "}) + strings.Join(parts, ", ") + ". Roll for initiative!"
}

func (e *Engine) consciousChars() []*Character {
	var out []*Character
	for _, ch := range e.sortedChars() {
		if ch.Conscious() {
			out = append(out, ch)
		}
	}
	if len(out) == 0 {
		out = e.livingChars()
	}
	return out
}

func (d *OfflineDM) bestAt(e *Engine, skill string) *Character {
	cs := e.consciousChars()
	sort.SliceStable(cs, func(i, j int) bool {
		return skillBonus(cs[i], skill) > skillBonus(cs[j], skill)
	})
	return cs[0]
}

func skillBonus(c *Character, skill string) int {
	b := c.Mod(Skills[skill])
	if c.SkillProficient(skill) {
		b += c.Prof()
	}
	return b
}

func (d *OfflineDM) springTrap(e *Engine, r *Room, t trapDef, victims []*Character) {
	r.TrapDone = true
	e.Log("dm", "DM", t.Trigger)
	for _, v := range victims {
		if v == nil || !v.Conscious() {
			continue
		}
		res := v.Save(t.Save, t.DC, 0)
		dmg := MustRoll(t.Damage).Total
		if res.Success {
			dmg /= 2
		}
		note := e.hurtOutsideCombat(v, dmg)
		e.Log("roll", v.Name, fmt.Sprintf("🎲 %s — takes %d damage. %s", res, dmg, note))
	}
}

var (
	reMove     = regexp.MustCompile(`\b(go|move|continue|proceed|advance|onward|forward|deeper|next|set out|head (out|on|in|deeper|forward)|enter|descend|explore|press on|keep going|venture|leave town|travel)\b`)
	reBack     = regexp.MustCompile(`\b(go back|retreat|return|turn back|head back|back to town)\b`)
	reLong     = regexp.MustCompile(`\b(long rest|sleep|camp|make camp|rest for the night|stay at the inn)\b`)
	reShort    = regexp.MustCompile(`\b(short rest|rest|catch (my|our) breath|take a break|bandage up)\b`)
	reSearch   = regexp.MustCompile(`\b(search|investigate|loot|rummage|examine the|inspect the|check the|look for|scour)\b`)
	reLook     = regexp.MustCompile(`\b(look|observe|survey|where are we|describe|what do (i|we) see|look around)\b`)
	reListen   = regexp.MustCompile(`\b(listen|scout|peek|sniff|eavesdrop|spy)\b`)
	reSneak    = regexp.MustCompile(`\b(sneak|hide|stealth|stealthily|quietly|creep|tiptoe)\b`)
	reDisarm   = regexp.MustCompile(`\b(disarm|disable|defuse|jam the|deactivate)\b`)
	reAvoid    = regexp.MustCompile(`\b(jump over|avoid|step around|carefully (cross|pass|step)|climb around|edge around)\b`)
	reLock     = regexp.MustCompile(`\b(pick the lock|pick lock|lockpick|unlock)\b`)
	reForce    = regexp.MustCompile(`\b(force|smash|break|bash|kick|pry)\b`)
	reOpen     = regexp.MustCompile(`\b(open|take|grab)\b.*\b(chest|box|coffer|trunk|treasure)\b|\bchest\b`)
	reTalk     = regexp.MustCompile(`\b(talk|speak|ask|greet|chat|question|help (him|her|them)|untie|free (him|her|them)|approach|hello|hi)\b`)
	rePersuade = regexp.MustCompile(`\b(persuade|convince|negotiate|surrender|parley|reason with|yield|bargain)\b`)
	reIntim    = regexp.MustCompile(`\b(intimidate|threaten|scare|menace|demand)\b`)
	reLore     = regexp.MustCompile(`\b(recall|remember|lore|legend|history|arcana|study|research|what do (i|we) know|pray|religion)\b`)
	reBuy      = regexp.MustCompile(`\b(buy|purchase)\b`)
	reSell     = regexp.MustCompile(`\bsell\b`)
	reMedic    = regexp.MustCompile(`\b(stabilize|medicine|treat|tend)\b`)
)

func (d *OfflineDM) resolve(e *Engine, ch *Character, text string) {
	t := strings.ToLower(text)
	c := e.C
	if c.Adventure == nil {
		if c.Mode == "ai" { // AI fallback with no generated dungeon: generic resolution only
			d.generic(e, ch, t)
			return
		}
		d.ensureAdventure(e)
	}
	a := c.Adventure
	r := d.room(e)

	if c.Combat != nil {
		switch {
		case rePersuade.MatchString(t) || reIntim.MatchString(t):
			d.parley(e, ch, t)
		default:
			e.Log("dm", "DM", fmt.Sprintf("%s, the battle rages around you! Use the combat buttons on your turn — attack, cast, use an item, dodge, or flee. (You can still try to talk your foes into surrendering.)", ch.Name))
		}
		return
	}

	switch {
	case reBuy.MatchString(t):
		d.buy(e, ch, t, r)
	case reSell.MatchString(t):
		d.sell(e, ch, t, r)
	case reLong.MatchString(t):
		d.rest(e, "long")
	case reShort.MatchString(t) && !strings.Contains(t, "rest of"):
		d.rest(e, "short")
	case reMedic.MatchString(t):
		d.stabilize(e, ch)
	case reDisarm.MatchString(t):
		d.disarm(e, ch, r)
	case reAvoid.MatchString(t) && r.TrapFound && !r.TrapDone:
		d.avoidTrap(e, ch, r)
	case reLock.MatchString(t):
		d.pickLock(e, ch, r)
	case reOpen.MatchString(t):
		d.openChest(e, ch, r, false)
	case reForce.MatchString(t) && r.Chest && !r.ChestOpen:
		d.openChest(e, ch, r, true)
	case reBack.MatchString(t):
		d.goBack(e, ch, t)
	case reSneak.MatchString(t):
		d.sneak(e, ch, t)
	case reSearch.MatchString(t):
		d.search(e, ch, r)
	case reListen.MatchString(t):
		d.listen(e, ch, r)
	case reTalk.MatchString(t) || rePersuade.MatchString(t) || reIntim.MatchString(t):
		d.talk(e, ch, r, t)
	case reLore.MatchString(t):
		d.lore(e, ch, t)
	case reMove.MatchString(t):
		d.move(e, ch)
	case reLook.MatchString(t):
		d.look(e, r)
	default:
		_ = a
		d.generic(e, ch, t)
	}
}

func (d *OfflineDM) rollLog(e *Engine, res CheckResult) {
	e.Log("roll", res.Name, "🎲 "+res.String())
}

func (d *OfflineDM) move(e *Engine, ch *Character) {
	a := e.C.Adventure
	r := d.room(e)
	if r.Trap != "" && r.TrapFound && !r.TrapDone {
		t := Traps[atoi(r.Trap)]
		e.Log("dm", "DM", "The party tries to edge past the "+t.Name+"...")
		var victims []*Character
		for _, c := range e.consciousChars() {
			if res := c.Save(t.Save, t.DC-2, 1); !res.Success {
				victims = append(victims, c)
			}
		}
		if len(victims) == 0 {
			r.TrapDone = true
			e.Log("dm", "DM", "Everyone makes it past safely.")
		} else {
			d.springTrap(e, r, t, victims)
		}
	}
	if a.Index >= len(a.Rooms)-1 {
		if a.Complete {
			d.newChapter(e)
		} else {
			e.Log("dm", "DM", "This is the deepest part of "+a.Title+". There's nowhere further to go.")
		}
		return
	}
	if a.Index == 0 {
		e.Log("dm", "DM", fmt.Sprintf("The party leaves town and travels to %s.", a.Title))
	}
	d.enter(e, a.Index+1)
}

func (d *OfflineDM) goBack(e *Engine, ch *Character, t string) {
	a := e.C.Adventure
	if a.Index == 0 {
		e.Log("dm", "DM", "You're already in town.")
		return
	}
	if strings.Contains(t, "town") || a.Index == 1 {
		e.Log("dm", "DM", "The party makes the long trek back to town.")
		d.enter(e, 0)
		return
	}
	d.enter(e, a.Index-1)
}

func (d *OfflineDM) look(e *Engine, r *Room) {
	text := fmt.Sprintf("**%s** — %s", r.Name, r.Desc)
	if r.TrapFound && !r.TrapDone {
		text += " The trap you found is still armed."
	}
	if r.Chest && !r.ChestOpen {
		if r.Locked {
			text += " A locked chest waits here."
		} else {
			text += " A chest waits here."
		}
	}
	if r.Hint != "" && !r.Town {
		text += " " + r.Hint
	}
	e.Log("dm", "DM", text)
}

func (d *OfflineDM) listen(e *Engine, ch *Character, r *Room) {
	res := ch.Check("perception", 11, 0)
	d.rollLog(e, res)
	if res.Success && r.Hint != "" {
		e.Log("dm", "DM", ch.Name+" listens carefully. "+r.Hint)
	} else if res.Success {
		e.Log("dm", "DM", ch.Name+" listens but hears nothing of note.")
	} else {
		e.Log("dm", "DM", ch.Name+" strains to listen, but can't make anything out over the noise of the party.")
	}
}

func (d *OfflineDM) sneak(e *Engine, ch *Character, t string) {
	a := e.C.Adventure
	if a.Index == 0 {
		e.Log("dm", "DM", "There's no need to sneak around town... though the guards do give you a funny look.")
		return
	}
	var lines []string
	ok := 0
	party := e.consciousChars()
	for _, c := range party {
		res := c.Check("stealth", 12, 0)
		lines = append(lines, res.String())
		if res.Success {
			ok++
		}
	}
	e.Log("roll", ch.Name, "🎲 Group stealth: "+strings.Join(lines, "; "))
	if ok*2 >= len(party) {
		a.Sneaking = true
		e.Log("dm", "DM", "Moving as one, the party slips silently forward. If enemies lie ahead, you'll catch them unaware. (Keep moving to press the advantage.)")
		if reMove.MatchString(t) {
			d.move(e, ch)
		}
	} else {
		a.Sneaking = false
		e.Log("dm", "DM", "A clatter of armor echoes down the passage. So much for stealth.")
	}
}

func (d *OfflineDM) search(e *Engine, ch *Character, r *Room) {
	if r.Town {
		e.Log("dm", "DM", "You poke around the market stalls. The merchant raises an eyebrow: \"Buying or just looking?\"")
		return
	}
	res := ch.Check("investigation", 12, 0)
	d.rollLog(e, res)
	if !res.Success {
		e.Log("dm", "DM", ch.Name+" searches but turns up nothing useful. Maybe someone else could take a look?")
		return
	}
	found := false
	var msgs []string
	if r.Trap != "" && !r.TrapFound && !r.TrapDone {
		r.TrapFound = true
		msgs = append(msgs, Traps[atoi(r.Trap)].Found)
		found = true
	}
	if !r.Searched {
		r.Searched = true
		if r.Gold > 0 && !r.Chest {
			ch.Gold += r.Gold
			msgs = append(msgs, fmt.Sprintf("%s finds a pouch with %d gold.", ch.Name, r.Gold))
			r.Gold = 0
			found = true
		}
		if len(r.Loot) > 0 && !r.Chest {
			for _, id := range r.Loot {
				if it, ok := NewItem(id); ok {
					ch.AddItem(it)
					msgs = append(msgs, fmt.Sprintf("%s discovers %s.", ch.Name, it.Name))
				}
			}
			r.Loot = nil
			found = true
		}
		if !found && chance(0.4) {
			g := Die(10) + e.PartyLevel()*2
			ch.Gold += g
			msgs = append(msgs, fmt.Sprintf("Tucked in a crack, %s finds %d gold coins.", ch.Name, g))
			found = true
		}
	}
	if r.Chest && !r.ChestOpen {
		msgs = append(msgs, "The chest here looks "+map[bool]string{true: "locked", false: "unlocked"}[r.Locked]+".")
		found = true
	}
	if !found {
		msgs = append(msgs, ch.Name+" searches thoroughly. There's nothing more to find here.")
	}
	e.Log("dm", "DM", strings.Join(msgs, " "))
	d.maybeClear(r)
}

func (d *OfflineDM) maybeClear(r *Room) {
	if len(r.Encounter) == 0 && (r.Trap == "" || r.TrapDone) && (!r.Chest || r.ChestOpen) {
		r.Cleared = true
	}
}

func (d *OfflineDM) disarm(e *Engine, ch *Character, r *Room) {
	if r.Trap == "" || r.TrapDone {
		e.Log("dm", "DM", "There's no armed trap here that you know of.")
		return
	}
	if !r.TrapFound {
		e.Log("dm", "DM", ch.Name+" doesn't know where a trap would be. Search the room first.")
		return
	}
	t := Traps[atoi(r.Trap)]
	adv := 0
	if ch.Class == "rogue" {
		adv = 1
	}
	res := ch.Check("sleight_of_hand", t.DC+1, adv)
	d.rollLog(e, res)
	if res.Success {
		r.TrapDone = true
		e.Log("dm", "DM", fmt.Sprintf("%s carefully disables the %s. Click. Safe.", ch.Name, t.Name))
		e.AwardXP(25*e.PartyLevel(), "disarming a trap")
	} else if res.Total < t.DC-4 {
		d.springTrap(e, r, t, []*Character{ch})
	} else {
		e.Log("dm", "DM", ch.Name+" fumbles with the mechanism but stops before setting it off. It's still armed.")
	}
	d.maybeClear(r)
}

func (d *OfflineDM) avoidTrap(e *Engine, ch *Character, r *Room) {
	t := Traps[atoi(r.Trap)]
	res := ch.Check("acrobatics", t.DC, 0)
	d.rollLog(e, res)
	if res.Success {
		e.Log("dm", "DM", ch.Name+" finds a safe way around the "+t.Name+" and guides the others through. (The party can now move on safely.)")
		r.TrapDone = true
	} else {
		d.springTrap(e, r, t, []*Character{ch})
	}
	d.maybeClear(r)
}

func (d *OfflineDM) pickLock(e *Engine, ch *Character, r *Room) {
	if !r.Chest || r.ChestOpen {
		e.Log("dm", "DM", "There's nothing locked here.")
		return
	}
	if !r.Locked {
		e.Log("dm", "DM", "The chest isn't even locked.")
		d.openChest(e, ch, r, false)
		return
	}
	hasTools := false
	for _, it := range ch.Inventory {
		if it.ID == "thieves_tools" {
			hasTools = true
		}
	}
	dc := 15
	if hasTools {
		dc = 12
	}
	res := ch.Check("sleight_of_hand", dc, 0)
	d.rollLog(e, res)
	if res.Success {
		r.Locked = false
		e.Log("dm", "DM", ch.Name+" works the lock until it gives with a satisfying click.")
		d.openChest(e, ch, r, false)
	} else {
		e.Log("dm", "DM", ch.Name+" can't get the lock open. Maybe try again — or just smash it.")
	}
}

func (d *OfflineDM) openChest(e *Engine, ch *Character, r *Room, force bool) {
	if !r.Chest || r.ChestOpen {
		e.Log("dm", "DM", "There's no unopened chest here.")
		return
	}
	if r.Locked {
		if !force {
			e.Log("dm", "DM", "The chest is locked. Pick the lock, or try to force it open.")
			return
		}
		res := ch.Check("athletics", 15, 0)
		d.rollLog(e, res)
		if !res.Success {
			e.Log("dm", "DM", ch.Name+" heaves at the lid, but the chest holds firm.")
			return
		}
		e.Log("dm", "DM", "With a crack of splintering wood, "+ch.Name+" wrenches the chest open!")
		r.Locked = false
	}
	if r.Trap != "" && !r.TrapDone {
		d.springTrap(e, r, Traps[atoi(r.Trap)], []*Character{ch})
	}
	r.ChestOpen = true
	var got []string
	for _, id := range r.Loot {
		if it, ok := NewItem(id); ok {
			ch.AddItem(it)
			got = append(got, it.Name)
		}
	}
	r.Loot = nil
	msg := ch.Name + " opens the chest"
	if len(got) > 0 {
		msg += " and finds: " + strings.Join(got, ", ")
	}
	e.Log("dm", "DM", msg+".")
	if r.Gold > 0 {
		e.splitGold(r.Gold, "from the chest")
		r.Gold = 0
	}
	d.maybeClear(r)
}

func (d *OfflineDM) talk(e *Engine, ch *Character, r *Room, t string) {
	if r.Town {
		d.townTalk(e, ch)
		return
	}
	if r.NPC == "" {
		e.Log("dm", "DM", fmt.Sprintf("%s speaks, but only echoes answer.", ch.Name))
		return
	}
	n := NPCs[atoi(r.NPC)]
	if r.NPCTalked {
		e.Log("dm", "DM", fmt.Sprintf("%s has told you all they know. %s", strings.ToUpper(n.Name[:1])+n.Name[1:], n.Hint))
		return
	}
	skill := "persuasion"
	if reIntim.MatchString(t) {
		skill = "intimidation"
	}
	res := ch.Check(skill, 11, 0)
	d.rollLog(e, res)
	r.NPCTalked = true
	r.Cleared = true
	if res.Success {
		it, _ := NewItem(n.Gift)
		ch.AddItem(it)
		e.Log("dm", "DM", fmt.Sprintf("%s wins the trust of %s. %s In gratitude, they give %s %s.", ch.Name, n.Name, n.Hint, ch.Name, article(it.Name)))
		e.AwardXP(50*e.PartyLevel(), "helping "+n.Name)
	} else {
		e.Log("dm", "DM", fmt.Sprintf("%s is wary of %s, but eventually mutters: %s", strings.ToUpper(n.Name[:1])+n.Name[1:], ch.Name, n.Hint))
	}
}

func (d *OfflineDM) townTalk(e *Engine, ch *Character) {
	a := e.C.Adventure
	res := ch.Check("persuasion", 12, 0)
	d.rollLog(e, res)
	if res.Success {
		e.Log("dm", "DM", "The innkeeper leans in close: \""+a.Lore+"\"")
	} else {
		e.Log("dm", "DM", "The locals mostly grumble about the weather — but all of them agree: "+strings.ToLower(a.Hook[:1])+a.Hook[1:])
	}
}

func (d *OfflineDM) lore(e *Engine, ch *Character, t string) {
	skill := "history"
	switch {
	case strings.Contains(t, "pray") || strings.Contains(t, "religion"):
		skill = "religion"
	case strings.Contains(t, "arcana") || strings.Contains(t, "magic"):
		skill = "arcana"
	case e.C.Adventure.Theme == "forest":
		skill = "nature"
	}
	res := ch.Check(skill, 12, 0)
	d.rollLog(e, res)
	if res.Success {
		e.Log("dm", "DM", ch.Name+" recalls: "+e.C.Adventure.Lore)
	} else {
		e.Log("dm", "DM", ch.Name+" racks their memory but comes up blank.")
	}
}

func (d *OfflineDM) parley(e *Engine, ch *Character, t string) {
	cb := e.C.Combat
	humanoid := true
	for _, m := range e.liveMonsters() {
		tt := Bestiary[m.Template]
		h := false
		for _, tag := range tt.Tags {
			if tag == "humanoid" {
				h = true
			}
		}
		if !h || m.Boss {
			humanoid = false
		}
	}
	if !humanoid {
		e.Log("dm", "DM", "Your foes can't be reasoned with — they only want blood!")
		return
	}
	skill := "persuasion"
	if reIntim.MatchString(t) {
		skill = "intimidation"
	}
	alive, total := len(e.liveMonsters()), len(cb.Monsters)
	dc := 17 - 6*(total-alive)/max(1, total)
	res := ch.Check(skill, dc, 0)
	d.rollLog(e, res)
	if res.Success {
		e.Log("dm", "DM", "The remaining enemies exchange glances, then throw down their weapons and flee into the dark!")
		e.endCombat("surrender", 0)
	} else {
		e.Log("dm", "DM", "Your foes snarl and fight on.")
	}
}

func (d *OfflineDM) rest(e *Engine, kind string) {
	a := e.C.Adventure
	r := d.room(e)
	if a != nil && !r.Town {
		risk := 0.1
		if kind == "long" {
			risk = 0.3
		}
		if r.Cleared && chance(0.5) {
			risk /= 2
		}
		if chance(risk) {
			th := a.Theme
			lvl := e.PartyLevel()
			ids := d.encounter(th, EncounterBudget[min(lvl, len(EncounterBudget))-1]*max(1, len(e.livingChars()))/2, len(e.livingChars()))
			e.Log("dm", "DM", "Your rest is interrupted! "+d.foesLine(ids, false))
			if _, err := e.StartCombat(spawnsFrom(ids), "enemies"); err != nil {
				e.Log("system", "", err.Error())
			}
			return
		}
	}
	e.restParty(kind)
	if kind == "long" && r != nil && r.Town {
		e.Log("dm", "DM", "You sleep soundly at the Crossroads Inn and wake to the smell of fresh bread.")
	}
}

func (d *OfflineDM) stabilize(e *Engine, ch *Character) {
	for _, o := range e.sortedChars() {
		if !o.Dead && o.HP <= 0 && o.ID != ch.ID {
			res := ch.Check("medicine", 10, 0)
			d.rollLog(e, res)
			if res.Success {
				o.Stable = true
				o.HP = 1
				e.Log("dm", "DM", fmt.Sprintf("%s binds %s's wounds. They groan back to consciousness with 1 HP.", ch.Name, o.Name))
			} else {
				e.Log("dm", "DM", ch.Name+"'s hands shake — the bandage slips. Try again!")
			}
			return
		}
	}
	e.Log("dm", "DM", "Nobody here needs emergency treatment. (A short rest will patch up minor wounds.)")
}

func (d *OfflineDM) buy(e *Engine, ch *Character, t string, r *Room) {
	if r == nil || !r.Town {
		e.Log("dm", "DM", "There's no one to buy from here. Head back to town.")
		return
	}
	id, price := "potion_healing", 50
	if strings.Contains(t, "greater") {
		id, price = "potion_greater", 150
	}
	for _, w := range []string{"longsword", "shortsword", "dagger", "rapier", "greataxe", "greatsword", "warhammer", "mace", "shortbow", "longbow", "javelin", "handaxe", "shield", "chain_mail", "chain shirt", "leather", "studded", "half plate", "plate", "rope", "torch", "rations"} {
		if strings.Contains(t, w) {
			key := strings.ReplaceAll(w, " ", "_")
			switch key {
			case "studded":
				key = "studded_leather"
			}
			if it, ok := ItemList[key]; ok {
				id, price = key, it.Value
			}
			break
		}
	}
	it, _ := NewItem(id)
	if ch.Gold < price {
		e.Log("dm", "DM", fmt.Sprintf("The merchant shakes his head. \"%s costs %d gold, friend. You've got %d.\"", it.Name, price, ch.Gold))
		return
	}
	ch.Gold -= price
	ch.AddItem(it)
	e.Log("dm", "DM", fmt.Sprintf("%s buys %s for %d gold. \"Pleasure doing business!\"", ch.Name, article(it.Name), price))
}

func (d *OfflineDM) sell(e *Engine, ch *Character, t string, r *Room) {
	if r == nil || !r.Town {
		e.Log("dm", "DM", "There's no one to sell to here. Head back to town.")
		return
	}
	for i, it := range ch.Inventory {
		if it.Equipped || it.Value == 0 {
			continue
		}
		name := strings.ToLower(it.Name)
		if strings.Contains(t, name) || strings.Contains(t, it.ID) || (it.Kind == "treasure" && (strings.Contains(t, "treasure") || strings.Contains(t, "gem") || strings.Contains(t, "idol"))) {
			price := it.Value / 2
			if it.Kind == "treasure" {
				price = it.Value
			}
			ch.Inventory[i].Qty--
			if ch.Inventory[i].Qty <= 0 {
				ch.Inventory = append(ch.Inventory[:i], ch.Inventory[i+1:]...)
			}
			ch.Gold += price
			e.Log("dm", "DM", fmt.Sprintf("%s sells %s for %d gold.", ch.Name, it.Name, price))
			return
		}
	}
	e.Log("dm", "DM", "The merchant looks puzzled. \"Sell what, exactly?\" (Name an unequipped item you carry.)")
}

func (d *OfflineDM) generic(e *Engine, ch *Character, t string) {
	verbs := []struct {
		re    *regexp.Regexp
		skill string
	}{
		{regexp.MustCompile(`\b(climb|jump|swim|lift|push|pull|break|force|bash|smash|kick|grapple|wrestle|carry)\b`), "athletics"},
		{regexp.MustCompile(`\b(balance|tumble|flip|vault|swing)\b`), "acrobatics"},
		{regexp.MustCompile(`\b(steal|pickpocket|pick|palm)\b`), "sleight_of_hand"},
		{regexp.MustCompile(`\b(track|forage|hunt|navigate|follow the)\b`), "survival"},
		{regexp.MustCompile(`\b(lie|bluff|deceive|trick|disguise)\b`), "deception"},
		{regexp.MustCompile(`\b(sing|play|dance|perform|juggle|tell a story)\b`), "performance"},
		{regexp.MustCompile(`\b(sense|motive|trust|read (his|her|their))\b`), "insight"},
		{regexp.MustCompile(`\b(calm|tame|ride|pet)\b`), "animal_handling"},
		{regexp.MustCompile(`\b(identify|read|decipher|magic|spell|rune)\b`), "arcana"},
		{regexp.MustCompile(`\b(look|watch|spot|search|perceive|notice)\b`), "perception"},
	}
	for _, v := range verbs {
		if v.re.MatchString(t) {
			res := ch.Check(v.skill, 12, 0)
			d.rollLog(e, res)
			if res.Success {
				e.Log("dm", "DM", fmt.Sprintf("%s %s", ch.Name+":", pick(successLines[v.skill])))
				if chance(0.25) && e.C.Adventure != nil {
					g := Die(6) + e.PartyLevel()
					ch.Gold += g
					e.Log("dm", "DM", fmt.Sprintf("Your effort turns up %d gold coins.", g))
				}
			} else {
				e.Log("dm", "DM", fmt.Sprintf("%s: %s", ch.Name, pick(failLines)))
			}
			return
		}
	}
	e.Log("dm", "DM", fmt.Sprintf("%s %s %s", ch.Name, pick([]string{"does so.", "gives it a try. Nothing much changes.", "takes a moment.", "does that — the world shrugs."}),
		pick([]string{"(Try: move on, search, listen, sneak, talk, open the chest, rest.)", "", ""})))
}

func (d *OfflineDM) afterCombat(e *Engine, summary string) {
	a := e.C.Adventure
	if a == nil {
		e.Log("dm", "DM", "The dust settles. The party catches its breath.")
		return
	}
	r := d.room(e)
	switch {
	case strings.HasPrefix(summary, "escaped"):
		e.Log("dm", "DM", "You scramble back the way you came, the sounds of pursuit fading behind you.")
		if a.Index > 1 {
			a.Index--
		}
		e.C.Location = d.room(e).Name
		return
	case strings.HasPrefix(summary, "victory"), strings.HasPrefix(summary, "surrender"):
	default:
		return
	}
	r.Encounter = nil
	d.maybeClear(r)
	if a.Index == len(a.Rooms)-1 && !a.Complete {
		a.Complete = true
		bonus := EncounterBudget[min(e.PartyLevel(), len(EncounterBudget))-1] * max(1, len(e.livingChars()))
		e.Log("dm", "DM", fmt.Sprintf("The master of %s falls! A hush settles over the place as its evil fades. You have triumphed.", a.Title))
		e.AwardXP(bonus, "completing the quest")
		e.Log("dm", "DM", "The lair still holds its treasure — open the chest! When you're ready, \"move on\" to return to town for the next chapter.")
		return
	}
	e.Log("dm", "DM", pick([]string{
		"The last foe falls. Silence returns, broken only by your heavy breathing.",
		"Victory. You wipe your blades and take stock of your wounds.",
		"The fight is over. Somewhere ahead, something may have heard the noise.",
	})+" "+r.Hint)
}

func (d *OfflineDM) newChapter(e *Engine) {
	old := e.C.Adventure
	if r := &old.Rooms[len(old.Rooms)-1]; r.Chest && !r.ChestOpen {
		r.ChestOpen = true
		var got []string
		living := e.livingChars()
		for i, id := range r.Loot {
			if it, ok := NewItem(id); ok && len(living) > 0 {
				living[i%len(living)].AddItem(it)
				got = append(got, it.Name+" ("+living[i%len(living)].Name+")")
			}
		}
		if len(got) > 0 {
			e.Log("dm", "DM", "Before leaving, you claim the lair's treasure: "+strings.Join(got, ", ")+".")
		}
		e.splitGold(r.Gold, "the lair's hoard")
	}
	e.C.Journal = append(e.C.Journal, fmt.Sprintf("Chapter %d: completed %s, defeating %s.", old.Chapter, old.Title, old.Boss))
	e.C.Adventure = d.generate(e, old.Chapter+1)
	e.Log("dm", "DM", "You return to town as heroes. Ale flows freely, and the inn's beds are soft. But rumors of new trouble are already spreading...")
	e.restParty("long")
	d.intro(e)
}

func (d *OfflineDM) afterDefeat(e *Engine) {
	a := e.C.Adventure
	if a == nil {
		e.Log("dm", "DM", "Darkness takes you... You awaken later, battered, robbed, but alive.")
		return
	}
	for _, ch := range e.C.Characters {
		ch.Gold /= 2
	}
	r := d.room(e)
	if r != nil && !r.Cleared && len(r.Encounter) == 0 {
		r.Encounter = d.encounter(a.Theme, EncounterBudget[min(e.PartyLevel(), len(EncounterBudget))-1]*max(1, len(e.livingChars())), len(e.livingChars()))
	}
	e.Log("dm", "DM", "Darkness takes you... You awaken in the town's temple, bandaged and aching. A passing patrol dragged you out — and your purses are lighter for it. (Half your gold is gone.) The danger still waits.")
	for _, ch := range e.C.Characters {
		ch.LongRest()
	}
	a.Index = 0
	e.C.Location = a.Rooms[0].Name
}
