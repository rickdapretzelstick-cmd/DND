package engine

import (
	"fmt"
	"math"
	"strings"
)

// ---------- dynamic quests: a chain of steps the engine tracks by itself ----------

func (q *Quest) cur() (int, *QuestStep) {
	for i := range q.Steps {
		if !q.Steps[i].Done {
			return i, &q.Steps[i]
		}
	}
	return -1, nil
}

func (e *Engine) absHours() float64 { return float64(e.C.Day)*24 + e.C.Clock }

func matchItem(it Item, want string) bool {
	return strings.EqualFold(it.ID, want) || strings.EqualFold(it.Name, want)
}

func (e *Engine) partyHas(ch *Character, want string, count int) bool {
	have := 0
	for _, c := range e.livingChars() {
		for _, it := range c.Inventory {
			if matchItem(it, want) {
				have += max(1, it.Qty)
			}
		}
	}
	_ = ch
	return have >= max(1, count)
}

func (e *Engine) partyTake(want string, count int) {
	left := max(1, count)
	for _, c := range e.livingChars() {
		for left > 0 {
			var id string
			for _, it := range c.Inventory {
				if matchItem(it, want) {
					id = it.ID + "\x00" + it.Name
					break
				}
			}
			if id == "" {
				break
			}
			parts := strings.SplitN(id, "\x00", 2)
			target := parts[0]
			if target == "" {
				target = parts[1]
			}
			if _, ok := c.RemoveItem(target, 1); !ok {
				break
			}
			left--
		}
	}
}

// questItem builds a quest-kind item a step can hand over.
func questItem(name, desc string) Item {
	return Item{ID: "qi_" + strings.ToLower(strings.Join(strings.Fields(name), "_")), Name: name, Kind: "quest", Desc: desc, Qty: 1, Custom: true}
}

// syncQuest points the quest's marker, objective and battle at its current step.
func (e *Engine) syncQuest(q *Quest) {
	if len(q.Steps) == 0 {
		return
	}
	i, st := q.cur()
	if st == nil {
		return
	}
	q.Objective = st.Text
	if st.X != 0 || st.Y != 0 {
		q.X, q.Y = st.X, st.Y
	}
	q.Encounter = nil
	if st.Type == "defeat" {
		q.Encounter = st.Spawns
	}
	if st.Type == "wait" && st.Until == 0 {
		st.Until = e.absHours() + float64(max(1, st.Hours))
	}
	_ = i
}

// addSteps resolves places/NPCs on new steps and attaches them to a quest.
func (e *Engine) prepareStep(st *QuestStep) error {
	switch st.Type {
	case "goto", "defeat":
		if st.Target != "" {
			x, y, _, ok := e.resolvePlace(st.Target)
			if !ok {
				return fmt.Errorf("unknown place %q for step %q", st.Target, st.Text)
			}
			st.X, st.Y = int(x), int(y)
		}
		if st.X == 0 && st.Y == 0 {
			return fmt.Errorf("step %q needs a target place", st.Text)
		}
	case "talk", "deliver":
		n := e.npcByID(st.Target)
		if n == nil {
			return fmt.Errorf("step %q needs target = an NPC id (create_npc first if needed)", st.Text)
		}
		st.X, st.Y = int(n.X), int(n.Y)
		if st.Type == "deliver" && st.Item == "" {
			return fmt.Errorf("deliver step %q needs an item", st.Text)
		}
	case "collect":
		if st.Item == "" {
			return fmt.Errorf("collect step %q needs an item", st.Text)
		}
		if st.Target != "" {
			if x, y, _, ok := e.resolvePlace(st.Target); ok {
				st.X, st.Y = int(x), int(y)
			}
		}
	case "wait":
		st.Hours = min(24, max(1, st.Hours))
	case "custom":
		if st.Target != "" {
			if x, y, _, ok := e.resolvePlace(st.Target); ok {
				st.X, st.Y = int(x), int(y)
			}
		}
	default:
		return fmt.Errorf("step type must be goto, talk, deliver, collect, defeat, wait or custom")
	}
	if st.Text == "" {
		return fmt.Errorf("every step needs text")
	}
	if len(st.Spawns) > 0 {
		for _, sp := range st.Spawns {
			if _, ok := Bestiary[sp.Type]; !ok {
				return fmt.Errorf("unknown curse %q", sp.Type)
			}
		}
	}
	return nil
}

// questEvent advances any quest whose current step the event satisfies.
// kind: arrive, talk (arg = npc id), items, hour, victory (arg = quest id).
func (e *Engine) questEvent(kind string, ch *Character, arg string) {
	for _, q := range e.C.Quests {
		if q.Stage != "active" || len(q.Steps) == 0 {
			continue
		}
		for guard := 0; guard < 12; guard++ {
			i, st := q.cur()
			if st == nil || !e.stepMet(q, st, kind, ch, arg) {
				break
			}
			e.completeStep(q, i, ch)
			if q.Stage != "active" {
				break
			}
			kind = "items" // a freshly handed-over item may satisfy the next step at once
		}
	}
}

func (e *Engine) stepMet(q *Quest, st *QuestStep, kind string, ch *Character, arg string) bool {
	switch st.Type {
	case "goto":
		return kind == "arrive" && ch != nil && math.Hypot(ch.X-float64(st.X), ch.Y-float64(st.Y)) < 3.5
	case "defeat":
		return kind == "victory" && arg == q.ID
	case "talk":
		return kind == "talk" && arg == st.Target
	case "deliver":
		return kind == "talk" && arg == st.Target && e.partyHas(ch, st.Item, st.Count)
	case "collect":
		return e.partyHas(ch, st.Item, st.Count)
	case "wait":
		return kind == "hour" && e.absHours() >= st.Until
	}
	return false
}

func (e *Engine) completeStep(q *Quest, i int, ch *Character) {
	st := &q.Steps[i]
	if st.Done {
		return
	}
	st.Done = true
	if st.Type == "deliver" {
		e.partyTake(st.Item, st.Count)
	}
	who := ch
	if who == nil {
		if l := e.livingChars(); len(l) > 0 {
			who = l[0]
		}
	}
	if st.GiveOnDone != "" && who != nil {
		who.AddItem(questItem(st.GiveOnDone, "For \""+q.Title+"\"."))
		e.Log("system", who.Name, "🎒 "+who.Name+" receives "+st.GiveOnDone+".")
	}
	if st.Say != "" {
		if n := e.npcByID(st.Target); n != nil {
			e.Log("say", n.Name, st.Say)
		}
	}
	left := 0
	for _, s := range q.Steps {
		if !s.Done {
			left++
		}
	}
	e.Log("system", "", fmt.Sprintf("✔ %s — %s (%d/%d)", q.Title, st.Text, len(q.Steps)-left, len(q.Steps)))
	if left > 0 {
		e.syncQuest(q)
		_, nx := q.cur()
		e.Log("system", "", "➡ Next: "+nx.Text)
		return
	}
	q.Objective = "All done."
	if q.Report && q.Giver != "" && e.npcByID(q.Giver) != nil {
		q.Stage = "return"
		e.Log("system", "", "📜 "+q.Title+": report back to "+q.GiverName+".")
		return
	}
	e.payQuest(q, who)
}

// payQuest completes a quest and hands out every reward, including a fabricated skill and title.
func (e *Engine) payQuest(q *Quest, ch *Character) {
	q.Stage = "done"
	var party []*Character
	if q.For != "" {
		if c := e.C.Characters[q.For]; c != nil && !c.Dead {
			party = []*Character{c}
			ch = c
		}
	}
	if party == nil {
		party = e.livingChars()
		if ch != nil {
			party = e.partyNear(ch, 14)
			if len(party) == 0 {
				party = []*Character{ch}
			}
		}
	}
	e.awardXPTo(party, q.RewardXP, "quest: "+q.Title)
	e.splitYen(party, q.RewardYen, "reward for "+q.Title)
	if len(party) > 0 {
		if it, ok := NewItem(q.RewardItem); ok {
			party[0].AddItem(it)
			e.Log("system", party[0].Name, "🎁 "+party[0].Name+" receives "+it.Name+".")
		}
		recv := party[0]
		if ch != nil {
			recv = ch
		}
		if q.RewardTitle != "" {
			recv.Title = q.RewardTitle
			e.Log("system", recv.Name, "🏅 "+recv.Name+" is now "+recv.Title+".")
		}
		if q.RewardSkill != nil {
			sp := *q.RewardSkill
			sp.Source = q.Title
			if _, err := e.GrantSkill(recv, sp); err != nil {
				e.Log("system", recv.Name, "(skill reward skipped: "+err.Error()+")")
			}
		}
	}
	e.C.Rep["civ"]++
	e.Log("system", "", "✅ Quest complete: "+q.Title)
}

// stepsHint is shown to the DM so it can see where each quest stands.
func (q *Quest) stepsHint() string {
	var b strings.Builder
	for i, s := range q.Steps {
		mark := " "
		if s.Done {
			mark = "x"
		}
		fmt.Fprintf(&b, "\n    [%s] %d. (%s) %s", mark, i+1, s.Type, s.Text)
	}
	return b.String()
}
