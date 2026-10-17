package engine

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Dialog is a conversation window shown to one player.
type Dialog struct {
	NPC     string         `json:"npc"`
	Name    string         `json:"name"`
	Role    string         `json:"role"`
	Look    Appearance     `json:"look"`
	Text    string         `json:"text"`
	Options []DialogOption `json:"options"`
	Free    bool           `json:"free"`
}

type DialogOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type dialogCtx struct {
	npc     string
	actions map[string]string
}

func (d *Dialog) opt(text, action string, ctx *dialogCtx) {
	id := strconv.Itoa(len(d.Options) + 1)
	d.Options = append(d.Options, DialogOption{ID: id, Text: text})
	ctx.actions[id] = action
}

// ---------- NPC access ----------

func (e *Engine) allNPCs() []*NPC {
	var out []*NPC
	for i := range e.W.NPCs {
		n := &e.W.NPCs[i]
		if st := e.C.NPCs[n.ID]; st != nil && st.Gone {
			continue
		}
		out = append(out, n)
	}
	for _, n := range e.C.ExtraNPCs {
		if st := e.C.NPCs[n.ID]; st != nil && st.Gone {
			continue
		}
		out = append(out, n)
	}
	return out
}

func (e *Engine) npcByID(id string) *NPC {
	for _, n := range e.allNPCs() {
		if n.ID == id {
			return n
		}
	}
	return nil
}

func (e *Engine) npcState(id string) *NPCState {
	st := e.C.NPCs[id]
	if st == nil {
		st = &NPCState{}
		e.C.NPCs[id] = st
	}
	return st
}

// nearbyNPCs returns NPCs within r tiles, nearest first.
func (e *Engine) nearbyNPCs(x, y, r float64) []*NPC {
	var out []*NPC
	for _, n := range e.allNPCs() {
		if math.Hypot(n.X-x, n.Y-y) <= r {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return math.Hypot(out[i].X-x, out[i].Y-y) < math.Hypot(out[j].X-x, out[j].Y-y)
	})
	return out
}

func (e *Engine) partyNear(ch *Character, r float64) []*Character {
	var out []*Character
	for _, o := range e.sortedChars() {
		if !o.Dead && math.Hypot(o.X-ch.X, o.Y-ch.Y) <= r {
			out = append(out, o)
		}
	}
	return out
}

// ---------- interaction entry points ----------

func (e *Engine) Interact(token, npcID string) (*Dialog, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, ch, err := e.charOf(token)
	if err != nil {
		return nil, err
	}
	n := e.npcByID(npcID)
	if n == nil {
		return nil, fmt.Errorf("nobody's there")
	}
	if math.Hypot(n.X-ch.X, n.Y-ch.Y) > 3.2 {
		return nil, fmt.Errorf("get closer to talk")
	}
	if ch.InBattle {
		return nil, fmt.Errorf("you're in a fight")
	}
	ctx := &dialogCtx{npc: n.ID, actions: map[string]string{}}
	d := e.npcDialog(ch, n, ctx)
	e.dialogs[p.ID] = ctx
	e.npcState(n.ID).Talked++
	return d, nil
}

func (e *Engine) ChooseDialog(token, optID string) (*Dialog, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, ch, err := e.charOf(token)
	if err != nil {
		return nil, err
	}
	ctx := e.dialogs[p.ID]
	if ctx == nil {
		return nil, fmt.Errorf("that conversation is over")
	}
	act, ok := ctx.actions[optID]
	if !ok {
		return nil, fmt.Errorf("pick one of the options")
	}
	n := e.npcByID(ctx.npc)
	if n == nil {
		delete(e.dialogs, p.ID)
		return nil, nil
	}
	next := &dialogCtx{npc: n.ID, actions: map[string]string{}}
	d := e.dialogAction(ch, n, act, next)
	if d == nil {
		delete(e.dialogs, p.ID)
	} else {
		e.dialogs[p.ID] = next
	}
	e.changed()
	return d, nil
}

func (e *Engine) newDialog(n *NPC, text string) *Dialog {
	return &Dialog{NPC: n.ID, Name: n.Name, Role: n.Role, Look: n.Look, Text: text}
}

// npcDialog builds the opening of a conversation.
func (e *Engine) npcDialog(ch *Character, n *NPC, ctx *dialogCtx) *Dialog {
	d := e.newDialog(n, e.greeting(ch, n))
	// Story beats first.
	e.storyOptions(ch, n, d, ctx)
	// Quest turn-ins and offers.
	for _, q := range e.C.Quests {
		if q.Giver == n.ID && q.Stage == "return" {
			d.opt("📜 Report: "+q.Title, "turnin:"+q.ID, ctx)
		}
	}
	switch n.Role {
	case "elder", "nanami":
		if q := e.questFrom(n.ID, "side"); q == nil {
			d.opt("❓ Any curse trouble I can help with?", "sidequest", ctx)
		} else if q.Stage == "active" {
			d.opt("❓ About "+q.Title+"...", "questinfo:"+q.ID, ctx)
		}
	case "shop":
		d.opt("🛒 Let me see your wares.", "shop:basic", ctx)
		d.opt("💴 I have things to sell.", "sell", ctx)
	case "maki":
		d.opt("🗡️ Can I buy some cursed tools?", "shop:tools", ctx)
	case "mercenary":
		d.opt("💎 Show me the rare stuff.", "shop:rare", ctx)
		d.opt("💴 I have things to sell.", "sell", ctx)
	case "inn":
		d.opt(fmt.Sprintf("🛏️ A room for the night (¥%s)", commas(innPrice)), "inn", ctx)
	case "shoko":
		d.opt("🩹 Patch us up, please.", "heal", ctx)
	case "ijichi":
		d.opt("🚗 Could you drive us somewhere?", "travel", ctx)
	case "monk":
		d.opt("🙏 Ask for a blessing", "blessing", ctx)
	}
	d.opt("🗨️ Rumors?", "rumor", ctx)
	d.opt("💬 Say something...", "free", ctx)
	d.opt("👋 Goodbye", "close", ctx)
	return d
}

const innPrice = 3000

func (e *Engine) greeting(ch *Character, n *NPC) string {
	st := e.npcState(n.ID)
	switch n.Role {
	case "gojo":
		if e.C.Act == 0 {
			return fmt.Sprintf("Yo! You must be %s. Welcome to Tokyo Jujutsu High~ I'm your teacher, Satoru Gojo. The strongest, by the way.", ch.Name)
		}
		return pickS(rngTL(), []string{"Yo~ Still alive? Good, good.", "Don't worry. I'm the strongest.", "Sweets are brain fuel. Want some kikufuku?", "The Higher-ups are being annoying again. So, what's up?"})
	case "yaga":
		return pickS(rngTL(), []string{"Hmph. Why are you here?", "A sorcerer must know why they fight. Do you?", "Don't touch the dolls."})
	case "shoko":
		return pickS(rngTL(), []string{"Got hurt? Sit down.", "Try not to die, it's a hassle to fix.", "You look terrible. I mean that medically."})
	case "ijichi":
		return pickS(rngTL(), []string{"Ah! Y-yes? Do you need a ride?", "I've drawn the veil... er, how can I help?", "Please don't tell Gojo-san I was napping."})
	case "nanami":
		return pickS(rngTL(), []string{"I'd rather not do overtime. Make it quick.", "Labor is crap. But someone has to do it.", "Ah. A student. What do you need?"})
	case "maki":
		return pickS(rngTL(), []string{"What? If you're gonna stare, buy something.", "Curses don't care about your clan. Neither do I.", "Need a weapon that actually hits? I got spares."})
	case "mercenary":
		return "Money is the one thing that never betrays you. What can I sell you?"
	case "junpei":
		if e.C.Flags["junpei_trust"] == "1" {
			return "Oh... it's you again. Hey."
		}
		return "...What do you want? Leave me alone."
	}
	if len(n.Lines) > 0 {
		return n.Lines[st.Talked%len(n.Lines)]
	}
	town := ""
	if t := e.W.Town(n.Town); t != nil {
		town = t.Name
	}
	if e.C.Flags["burned_"+n.Town] == "1" {
		return pickS(rngTL(), []string{"Half the town burned while you were away. Where were the sorcerers?", "We're rebuilding... somehow."})
	}
	switch n.Role {
	case "elder":
		return fmt.Sprintf("Welcome to %s, sorcerer. Please — people here have been seeing things.", town)
	case "shop":
		return "Irasshaimase! Medicine, tonics, talismans — curse-hunting essentials."
	case "inn":
		return "Welcome! A room, a bath and a hot meal. Curses can't follow you in here."
	case "guard":
		return "Move along... wait, you're with that school, aren't you? Something's wrong around here."
	case "monk":
		return "The air is heavy with cursed energy lately. Be careful, young one."
	}
	return pickS(rngTL(), []string{"Hm? Can I help you?", "Nice weather, at least.", "You're one of those 'special school' kids, right?", fmt.Sprintf("Ever been to %s before?", town)})
}

func rngTL() *rand.Rand { return rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())) }

// dialogAction executes an option. Returns the next dialog, or nil to close.
func (e *Engine) dialogAction(ch *Character, n *NPC, act string, ctx *dialogCtx) *Dialog {
	verb, arg, _ := strings.Cut(act, ":")
	back := func(text string) *Dialog {
		d := e.npcDialog(ch, n, ctx)
		if text != "" {
			d.Text = text
		}
		return d
	}
	switch verb {
	case "close":
		return nil
	case "free":
		d := e.newDialog(n, "(Type what you say or do — anything at all.)")
		d.Free = true
		d.opt("👋 Goodbye", "close", ctx)
		return d
	case "rumor":
		return back(e.rumor(n))
	case "story":
		return e.storyAction(ch, n, arg, ctx)
	case "sidequest":
		q := e.makeSideQuest(n)
		if q == nil {
			return back("Things are quiet right now, thank goodness. Check back later.")
		}
		d := e.newDialog(n, q.Desc+fmt.Sprintf("\n\nReward: %d XP, ¥%s. Recommended: level %d+.", q.RewardXP, commas(q.RewardYen), q.MinLevel))
		d.opt("✅ I'll handle it.", "accept:"+q.ID, ctx)
		d.opt("Not now.", "decline:"+q.ID, ctx)
		return d
	case "accept":
		if q := e.quest(arg); q != nil && q.Stage == "offered" {
			q.Stage = "active"
			e.Log("system", ch.Name, fmt.Sprintf("📜 New quest: %s — %s", q.Title, q.Objective))
			return back(pickS(rngTL(), []string{"Thank you. Please be careful.", "Bless you. The map shows the way.", "We're counting on you."}))
		}
		return back("")
	case "decline":
		e.C.Quests = slices.DeleteFunc(e.C.Quests, func(q *Quest) bool { return q.ID == arg && q.Stage == "offered" })
		return back("I understand.")
	case "questinfo":
		if q := e.quest(arg); q != nil {
			return back(q.Desc + "\n\nObjective: " + q.Objective)
		}
		return back("")
	case "turnin":
		return back(e.turnIn(ch, n, arg))
	case "shop":
		return e.shopDialog(ch, n, arg, ctx, "")
	case "buy":
		item, kind, _ := strings.Cut(arg, "@")
		return e.shopDialog(ch, n, kind, ctx, e.buy(ch, n, item))
	case "sell":
		return e.sellDialog(ch, n, ctx, "")
	case "sellidx":
		i, _ := strconv.Atoi(arg)
		return e.sellDialog(ch, n, ctx, e.sell(ch, i))
	case "inn":
		if ch.Gold < innPrice {
			return back("Sorry — that's ¥3,000 for the night.")
		}
		ch.Gold -= innPrice
		for _, o := range e.partyNear(ch, 8) {
			o.LongRest()
		}
		e.C.Clock = 7
		e.Log("system", ch.Name, "🛏️ The party rests at the inn. HP and cursed energy fully restored. Morning comes.")
		return back("Good morning! Breakfast is grilled fish. Safe travels!")
	case "heal":
		var healed []string
		for _, o := range e.partyNear(ch, 10) {
			o.LongRest()
			healed = append(healed, o.Name)
		}
		e.Log("system", "Shoko", "🩹 Shoko patches up "+strings.Join(healed, ", ")+" with Reverse Cursed Technique. Fully restored.")
		return back("Done. Try to come back in fewer pieces next time.")
	case "travel":
		d := e.newDialog(n, "Wh-where to? I can take you anywhere you've been before.")
		for _, t := range e.W.Towns {
			if e.C.Discovered[t.ID] && math.Hypot(float64(t.X)-ch.X, float64(t.Y)-ch.Y) > float64(t.R)+3 {
				d.opt("🚗 "+t.Name, "go:"+t.ID, ctx)
			}
		}
		d.opt("Never mind.", "close", ctx)
		return d
	case "go":
		t := e.W.Town(arg)
		if t == nil {
			return nil
		}
		riders := e.partyNear(ch, 8)
		e.teleport(riders, float64(t.X)+0.5, float64(t.Y)+2.5)
		names := make([]string, len(riders))
		for i, r := range riders {
			names[i] = r.Name
		}
		e.Log("system", "Ijichi", fmt.Sprintf("🚗 Ijichi drives %s to %s.", strings.Join(names, ", "), t.Name))
		e.C.Location = t.Name
		return nil
	case "blessing":
		st := e.npcState(n.ID)
		if st.Mood > 0 {
			return back("I have already given you what protection I can.")
		}
		st.Mood = 1
		it, _ := NewItem("talisman")
		it.Qty = 2
		ch.AddItem(it)
		e.Log("system", ch.Name, "🙏 "+n.Name+" gives "+ch.Name+" two exorcism talismans.")
		return back("May these keep the darkness at bay. Two talismans — use them well.")
	}
	return back("")
}

func (e *Engine) rumor(n *NPC) string {
	var opts []string
	for _, s := range e.W.Sites {
		if s.Town == n.Town || n.Town == "" {
			opts = append(opts, fmt.Sprintf("They say people who go near the %s don't come back the same. %s-level stuff, if you ask me.", s.Name, GradeNames[s.Rank]))
		}
	}
	for _, q := range e.C.Quests {
		if q.Stage == "active" {
			opts = append(opts, "Heard about \""+q.Title+"\"? Everyone's whispering about it.")
		}
	}
	opts = append(opts,
		"Curses come from people's negative emotions. Fear, hatred, regret... the bigger the crowd, the bigger the curse.",
		"Night's when the nasty ones come out. Stick to the roads.",
		"A man with stitches across his forehead was asking about 'the vessel'. Gave me the creeps.",
		"Somebody saw a white-haired guy with a blindfold eating crepes at three in the morning. Weird city.")
	return pickS(rngTL(), opts)
}

// ---------- shops ----------

var shopStock = map[string][]string{
	"basic": {"medicine", "strong_medicine", "ce_tonic", "cough_syrup", "talisman", "veil_tag", "cursed_glasses"},
	"tools": {"slaughter_demon", "katana", "staff", "revolver", "reinforced", "tactical", "playful_cloud"},
	"rare":  {"dragon_bone", "cursed_garb", "split_soul", "inverted_spear", "strong_medicine"},
}

func (e *Engine) price(base int) int {
	p := base
	switch civ := e.C.Rep["civ"]; {
	case civ >= 3:
		p = p * 80 / 100
	case civ <= -3:
		p = p * 130 / 100
	}
	return max(100, p)
}

func (e *Engine) shopDialog(ch *Character, n *NPC, kind string, ctx *dialogCtx, msg string) *Dialog {
	if e.C.Flags["burned_"+n.Town] == "1" && kind == "basic" {
		return e.newDialog(n, "The shop burned down. I've got nothing left to sell.")
	}
	text := msg
	if text == "" {
		text = fmt.Sprintf("Take a look. You have ¥%s.", commas(ch.Gold))
	} else {
		text += fmt.Sprintf(" (You have ¥%s.)", commas(ch.Gold))
	}
	d := e.newDialog(n, text)
	for _, id := range shopStock[kind] {
		it := ItemList[id]
		label := it.Name
		if it.Damage != "" && it.Kind == "weapon" {
			label += " (" + it.Damage + ")"
		} else if it.BaseAC > 0 {
			label += fmt.Sprintf(" (AC %d)", it.BaseAC)
		}
		d.opt(fmt.Sprintf("%s — ¥%s", label, commas(e.price(it.Value))), "buy:"+id+"@"+kind, ctx)
	}
	d.opt("Done shopping.", "close", ctx)
	return d
}

func (e *Engine) buy(ch *Character, n *NPC, id string) string {
	it, ok := NewItem(id)
	if !ok {
		return "We don't have that."
	}
	p := e.price(it.Value)
	if ch.Gold < p {
		return fmt.Sprintf("That's ¥%s. You're short.", commas(p))
	}
	ch.Gold -= p
	ch.AddItem(it)
	e.Log("system", ch.Name, fmt.Sprintf("🛒 %s buys %s for ¥%s.", ch.Name, it.Name, commas(p)))
	return "Thank you! " + it.Name + " is yours."
}

func (e *Engine) sellDialog(ch *Character, n *NPC, ctx *dialogCtx, msg string) *Dialog {
	if msg == "" {
		msg = "What are you selling?"
	}
	d := e.newDialog(n, msg)
	for i, it := range ch.Inventory {
		if it.Equipped || it.Value <= 0 || it.Kind == "quest" {
			continue
		}
		d.opt(fmt.Sprintf("%s — ¥%s", it.Name, commas(sellValue(it))), fmt.Sprintf("sellidx:%d", i), ctx)
	}
	d.opt("Done.", "close", ctx)
	return d
}

func sellValue(it Item) int {
	if it.Kind == "treasure" {
		return it.Value
	}
	return it.Value / 2
}

func (e *Engine) sell(ch *Character, i int) string {
	if i < 0 || i >= len(ch.Inventory) {
		return "Hm?"
	}
	it := ch.Inventory[i]
	if it.Equipped || it.Value <= 0 || it.Kind == "quest" {
		return "I can't buy that."
	}
	v := sellValue(it)
	ch.Gold += v
	ch.RemoveItem(it.ID, 1)
	e.Log("system", ch.Name, fmt.Sprintf("💴 %s sells %s for ¥%s.", ch.Name, it.Name, commas(v)))
	return "Pleasure doing business."
}

// ---------- quests ----------

func (e *Engine) quest(id string) *Quest {
	for _, q := range e.C.Quests {
		if q.ID == id {
			return q
		}
	}
	return nil
}

func (e *Engine) questFrom(npc, kind string) *Quest {
	for _, q := range e.C.Quests {
		if q.Giver == npc && q.Kind == kind && (q.Stage == "active" || q.Stage == "return" || q.Stage == "offered") {
			return q
		}
	}
	return nil
}

func (e *Engine) siteBusy(id string) bool {
	for _, q := range e.C.Quests {
		if q.Site == id && (q.Stage == "active" || q.Stage == "offered") {
			return true
		}
	}
	return false
}

// pickSite chooses a free site near a point, preferring the given kinds.
func (e *Engine) pickSite(x, y float64, kinds []string) *Site {
	var best *Site
	bd := math.MaxFloat64
	for pass := 0; pass < 2 && best == nil; pass++ {
		for i := range e.W.Sites {
			s := &e.W.Sites[i]
			if e.siteBusy(s.ID) {
				continue
			}
			if pass == 0 && len(kinds) > 0 && !slices.Contains(kinds, s.Kind) {
				continue
			}
			d := math.Hypot(float64(s.X)-x, float64(s.Y)-y)
			if d < bd {
				best, bd = s, d
			}
		}
	}
	return best
}

func (e *Engine) addQuest(q *Quest) *Quest {
	q.ID = e.nextID("q")
	if q.Stage == "" {
		q.Stage = "offered"
	}
	if n := e.npcByID(q.Giver); n != nil {
		q.GiverName = n.Name
	}
	e.C.Quests = append(e.C.Quests, q)
	return q
}

func (e *Engine) partyBudget() (int, int) {
	lvl := e.PartyLevel()
	size := max(1, len(e.livingChars()))
	return EncounterBudget[min(lvl, len(EncounterBudget))-1] * size, size
}

// encounterFor builds a fight of roughly the given XP budget from curses matching a tag.
func encounterFor(tag string, maxRank, budget, size int, boss bool) []Spawn {
	var cands []CurseTemplate
	for _, id := range SortedCurseIDs() {
		t := Bestiary[id]
		if t.Rank > maxRank || slices.Contains(t.Tags, "special") || slices.Contains(t.Tags, "mahito") || slices.Contains(t.Tags, "kyoto") {
			continue
		}
		if slices.Contains(t.Tags, tag) || slices.Contains(t.Tags, "any") {
			cands = append(cands, t)
		}
	}
	if len(cands) == 0 {
		cands = []CurseTemplate{Bestiary["fly_head"]}
	}
	var out []Spawn
	if boss {
		// The strongest candidate that fits 60% of the budget leads, enraged if it's weak.
		lead := cands[0]
		for _, c := range cands {
			if c.XP <= budget*6/10 && c.XP >= lead.XP {
				lead = c
			}
		}
		elite := lead.XP*3+50 <= budget
		out = append(out, Spawn{Type: lead.ID, Count: 1, Elite: elite})
		spent := lead.XP
		if elite {
			spent = lead.XP*3 + 50
		}
		budget -= spent
	}
	var minions []CurseTemplate
	for _, c := range cands {
		if !c.Boss && c.XP <= max(25, budget) {
			minions = append(minions, c)
		}
	}
	if len(minions) > 0 && budget > 0 {
		m := minions[rand.IntN(len(minions))]
		n := min(size+1, max(1, budget/max(1, m.XP*2)))
		if !boss {
			n = max(n, 2)
		}
		out = append(out, Spawn{Type: m.ID, Count: min(n, 5)})
	}
	return out
}

var sideTitles = map[string][]string{
	"school":    {"Seven Wonders of %s", "After-School Haunting at %s", "Missing Students at %s"},
	"hospital":  {"Night Shift at %s", "The Patients of %s", "Whispers in %s"},
	"tunnel":    {"The Last Train Through %s", "Voices in %s"},
	"shrine":    {"Desecration at %s", "The Angry God of %s"},
	"cemetery":  {"Restless Graves of %s", "Mourners at %s"},
	"factory":   {"Shift Change at %s", "Something in %s"},
	"apartment": {"Room 404 of %s", "The Neighbors at %s"},
	"grove":     {"Lost Hikers in %s", "The Hanging Trees of %s"},
	"lake":      {"Drowned Voices at %s", "The Thing Under %s"},
	"detention": {"Lights Out at %s", "Riot at %s"},
}

func (e *Engine) makeSideQuest(n *NPC) *Quest {
	if q := e.questFrom(n.ID, "side"); q != nil {
		return q
	}
	x, y := n.X, n.Y
	s := e.pickSite(x, y, nil)
	if s == nil {
		return nil
	}
	budget, size := e.partyBudget()
	maxRank := min(4, (e.PartyLevel()+1)/2)
	tag := s.Tag
	titles := sideTitles[s.Kind]
	if len(titles) == 0 {
		titles = []string{"Trouble at %s"}
	}
	title := fmt.Sprintf(pickS(rngTL(), titles), s.Name)
	q := &Quest{Kind: "side", Giver: n.ID, Site: s.ID, X: s.X, Y: s.Y, Title: title, MinLevel: max(1, e.PartyLevel()-1),
		Objective: "Exorcise the curse at " + s.Name + ", then report back.",
		Encounter: encounterFor(tag, maxRank, budget*12/10, size, true),
		RewardXP:  budget / 2, RewardYen: 10000 + budget*40,
		Choice: sideChoices[rand.IntN(len(sideChoices))].ID}
	lead := Bestiary[q.Encounter[0].Type]
	q.Desc = fmt.Sprintf("%s A %s has been sensed at the %s — a %s, if the Window's report is right. %s",
		pickS(rngTL(), []string{"Please, you have to help us.", "I'm at my wits' end.", "The police can't explain it."}),
		strings.ToLower(lead.Grade)+" curse", s.Name, lead.Name, lead.Desc)
	return e.addQuest(q)
}

func (e *Engine) turnIn(ch *Character, n *NPC, id string) string {
	q := e.quest(id)
	if q == nil || q.Stage != "return" {
		return ""
	}
	q.Stage = "done"
	party := e.partyNear(ch, 12)
	e.awardXPTo(party, q.RewardXP, "quest: "+q.Title)
	e.splitYen(party, q.RewardYen, "reward for "+q.Title)
	if q.RewardItem != "" {
		if it, ok := NewItem(q.RewardItem); ok {
			ch.AddItem(it)
			e.Log("system", ch.Name, "🎁 "+ch.Name+" receives "+it.Name+".")
		}
	}
	e.C.Rep["civ"]++
	e.Log("system", "", "✅ Quest complete: "+q.Title)
	if q.Kind == "main" {
		return e.mainTurnIn(ch, q)
	}
	return pickS(rngTL(), []string{"Thank you! The town can sleep again.", "You've saved us. Please, take this.", "I'll tell everyone what you did."})
}

// afterBattle wires a finished fight back into the world. Caller holds the lock.
func (e *Engine) afterBattle(cb *Combat, outcome string, party []*Character) {
	if cb.RoamerID != "" {
		if r := e.roamer(cb.RoamerID); r != nil {
			switch outcome {
			case "victory", "surrender", "truce":
				e.removeRoamer(r.ID)
			default:
				r.cool = 60
				r.X, r.Y = r.HomeX, r.HomeY
			}
		}
	}
	if outcome == "defeat" {
		if shoko := e.npcByID("shoko"); shoko != nil {
			e.teleport(party, shoko.X-1, shoko.Y+1)
		} else {
			hx, hy := e.hubSpawn()
			e.teleport(party, hx, hy)
		}
		e.Log("dm", "DM", "Everything goes black... You wake up in Jujutsu High's infirmary. Shoko is lighting a cigarette. \"Ijichi dragged you back. You're welcome.\"")
		return
	}
	q := e.quest(cb.QuestID)
	if q == nil || outcome == "escaped" {
		return
	}
	if outcome != "victory" && outcome != "surrender" && outcome != "truce" {
		return
	}
	q.Stage = "return"
	if q.Site != "" {
		e.C.Cleared[q.Site] = true
	}
	e.Log("system", "", fmt.Sprintf("📜 %s — objective complete. Report to %s.", q.Title, q.GiverName))
	if q.Kind == "main" {
		e.mainVictory(q, party)
		return
	}
	if q.Choice != "" {
		for _, c := range sideChoices {
			if c.ID == q.Choice {
				cc := c
				cc.Text = strings.ReplaceAll(cc.Text, "%site%", q.Title)
				e.offerChoice(&cc)
			}
		}
	}
}

// ---------- party choices ----------

func (e *Engine) offerChoice(c *Choice) {
	cp := *c
	cp.Options = slices.Clone(c.Options)
	e.C.Choice = &cp
	e.Log("dm", cp.Speaker, "❗ "+cp.Title+" — "+cp.Text)
	e.changed()
}

// Choose resolves the party choice. Any player may decide.
func (e *Engine) Choose(token, optID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ch, err := e.charOf(token)
	if err != nil {
		return err
	}
	c := e.C.Choice
	if c == nil {
		return fmt.Errorf("there's no decision to make")
	}
	var opt *ChoiceOption
	for i := range c.Options {
		if c.Options[i].ID == optID {
			opt = &c.Options[i]
		}
	}
	if opt == nil {
		return fmt.Errorf("pick one of the options")
	}
	e.C.Choice = nil
	e.Log("action", ch.Name, "chooses: "+opt.Text)
	chooser := ch
	if c.For != "" {
		if f, ok := e.C.Characters[c.For]; ok {
			chooser = f
		}
	}
	result := opt.Result
	if strings.HasPrefix(result, "check:") {
		// "check:persuasion:14|success text|failure text|failure effects"
		parts := strings.Split(result, "|")
		spec := strings.Split(parts[0], ":")
		dc, _ := strconv.Atoi(spec[2])
		res := ch.Check(spec[1], dc, 0)
		e.Log("roll", ch.Name, "🎲 "+res.String())
		if res.Success {
			result = parts[1]
			if len(parts) > 4 && parts[4] != "" {
				opt.Effects = append(opt.Effects, strings.Split(parts[4], ",")...)
			}
		} else {
			result = parts[2]
			if len(parts) > 3 && parts[3] != "" {
				opt.Effects = append(opt.Effects, strings.Split(parts[3], ",")...)
			}
		}
	}
	for _, m := range e.applyEffects(opt.Effects, chooser) {
		e.Log("system", "", m)
	}
	if result != "" {
		e.Log("dm", c.Speaker, result)
	}
	if c.Source == "ai" || e.AIActive() {
		e.enqueue(DMEvent{Kind: "choice", Summary: fmt.Sprintf("%s — the party chose: %q (decided by %s).", c.Title, opt.Text, ch.Name)})
	}
	if strings.HasPrefix(c.Source, "act") {
		e.afterStoryChoice(c.Source, opt.ID, ch)
	}
	e.changed()
	return nil
}

// applyEffects runs effect strings from story data.
func (e *Engine) applyEffects(effects []string, ch *Character) []string {
	var msgs []string
	for _, ef := range effects {
		ef = strings.TrimSpace(ef)
		k, v, _ := strings.Cut(ef, ":")
		switch k {
		case "flag":
			fk, fv, _ := strings.Cut(v, "=")
			e.C.Flags[fk] = fv
		case "rep":
			for _, fac := range []string{"higher", "civ", "curse", "gojo"} {
				if strings.HasPrefix(v, fac) {
					n, _ := strconv.Atoi(strings.TrimPrefix(v, fac))
					e.C.Rep[fac] += n
					names := map[string]string{"higher": "the Higher-ups", "civ": "civilians", "curse": "the curse-user underworld", "gojo": "Gojo"}
					if n > 0 {
						msgs = append(msgs, fmt.Sprintf("📈 Standing with %s improved.", names[fac]))
					} else if n < 0 {
						msgs = append(msgs, fmt.Sprintf("📉 Standing with %s dropped.", names[fac]))
					}
				}
			}
		case "xp":
			n, _ := strconv.Atoi(v)
			e.awardXPTo(e.livingChars(), n, "")
		case "yen":
			n, _ := strconv.Atoi(v)
			ch.Gold = max(0, ch.Gold+n)
			if n > 0 {
				msgs = append(msgs, fmt.Sprintf("💴 %s receives ¥%s.", ch.Name, commas(n)))
			}
		case "item":
			if it, ok := NewItem(v); ok {
				ch.AddItem(it)
				msgs = append(msgs, fmt.Sprintf("🎁 %s receives %s.", ch.Name, it.Name))
			}
		case "remove":
			for _, o := range e.C.Characters {
				o.RemoveItem(v, 0)
			}
		case "vessel":
			ch.Vessel = true
			ch.Stats["STR"] = min(20, ch.Stats["STR"]+2)
			ch.Stats["CON"] = min(20, ch.Stats["CON"]+2)
			ch.MaxHP += 10
			ch.HP = ch.MaxHP
			ch.Title = "Vessel of Sukuna"
			ch.RecalcAC()
			msgs = append(msgs, "🩸 Black markings spread under "+ch.Name+"'s eyes. Something ancient laughs inside them. (+2 STR, +2 CON, +10 max HP)")
		case "journal":
			e.C.Journal = append(e.C.Journal, v)
		case "burn":
			e.C.Flags["burned_"+v] = "1"
			if t := e.W.Town(v); t != nil {
				msgs = append(msgs, "🔥 "+t.Name+" burns.")
			}
		case "act":
			n, _ := strconv.Atoi(v)
			e.C.Act = n
		case "title":
			ch.Title = v
		}
	}
	return msgs
}

var sideChoices = []Choice{
	{ID: "side_curse_user", Title: "A Curse User Surrenders", Speaker: "DM",
		Text: "In the wreckage of %site%, a teenage curse user who was feeding the curse throws up their hands. \"Please — they said they'd kill my family if I didn't!\"",
		Options: []ChoiceOption{
			{ID: "report", Text: "Turn them over to the Higher-ups", Effects: []string{"rep:higher+1", "rep:curse-1"}, Result: "The Higher-ups collect the teen. You never hear what happened to them."},
			{ID: "release", Text: "Let them go with a warning", Effects: []string{"rep:curse+1", "rep:higher-1"}, Result: "The teen flees into the night. Somewhere, someone owes you one."},
			{ID: "recruit", Text: "Make them your informant", Effects: []string{"rep:curse+1", "flag:informant=1", "yen:5000"}, Result: "\"I'll tell you everything I hear. I swear.\" They hand over their savings as a show of faith."},
		}},
	{ID: "side_witnesses", Title: "Witnesses", Speaker: "DM",
		Text: "Civilians saw everything at %site% — the curse, the techniques, the blood. They're staring at you.",
		Options: []ChoiceOption{
			{ID: "erase", Text: "Call Ijichi to handle it (the official way)", Effects: []string{"rep:higher+1"}, Result: "Ijichi arrives with a cover story about a gas leak. Nobody remembers quite what they saw."},
			{ID: "truth", Text: "Tell them the truth about curses", Effects: []string{"rep:civ+2", "rep:higher-1"}, Result: "They listen, terrified but grateful. Word spreads: there are people protecting them."},
			{ID: "talk", Text: "Calm them down yourself", Need: "Will check", Result: "check:persuasion:13|You talk them down gently. They go home believing a storm caused the damage.|Your explanation makes it worse. Someone posts a video online.|rep:higher-1"},
		}},
	{ID: "side_object", Title: "A Cursed Object", Speaker: "DM",
		Text: "At the heart of %site% you find what fed the curse: an old cursed object, humming with energy.",
		Options: []ChoiceOption{
			{ID: "seal", Text: "Seal it and deliver it to the Higher-ups", Effects: []string{"rep:higher+1", "yen:20000"}, Result: "Sealed and logged. The Higher-ups pay a 'disposal fee'."},
			{ID: "keep", Text: "Keep it", Effects: []string{"item:cursed_trinket", "rep:curse+1"}, Result: "You pocket it. It feels warm against your leg."},
			{ID: "destroy", Text: "Destroy it on the spot", Effects: []string{"rep:civ+1", "xp:100"}, Result: "It shrieks as it breaks. The air feels lighter."},
		}},
	{ID: "side_dying_curse", Title: "Last Words", Speaker: "DM",
		Text: "As the curse at %site% dissolves, it speaks with a human voice: \"He... made me... the man with the stitches...\"",
		Options: []ChoiceOption{
			{ID: "listen", Text: "Listen to everything it says", Effects: []string{"flag:stitch_clue=1", "journal:A dying curse spoke of 'the man with stitches' who made it.", "xp:150"}, Result: "It describes a man with stitches across his forehead, gathering curses for 'the day the world is reborn.' Then it's gone."},
			{ID: "finish", Text: "Finish it. Curses lie.", Effects: []string{"rep:higher+1"}, Result: "One more exorcism. One less voice."},
		}},
	{ID: "side_rival", Title: "A Rival Claim", Speaker: "Zenin Retainer",
		Text: "A sorcerer from the Zenin clan arrives as %site% falls silent. \"This exorcism was ours. Hand over the report — and the fee.\"",
		Options: []ChoiceOption{
			{ID: "concede", Text: "Let them have it", Effects: []string{"rep:higher+1", "yen:-5000"}, Result: "They take the credit and leave without a word of thanks."},
			{ID: "argue", Text: "Stand your ground", Need: "Will check", Result: "check:intimidation:14|They back down, muttering about manners. The fee is yours.|They laugh in your face and file the report first.|rep:higher-1"},
			{ID: "fightword", Text: "Remind them who did the work", Effects: []string{"rep:curse+1", "xp:100"}, Result: "Words are exchanged. Nobody draws, but the Zenin clan will remember your name."},
		}},
}
