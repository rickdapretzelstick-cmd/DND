package engine

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
)

// The improv quest builder: turns "can I be Shoko's assistant?" into a real chain of steps built
// from the actual NPCs, shops and sites around. The AI DM does this with create_quest; this is the
// same idea for the offline DM and as the "ask about odd jobs" button.

type jobScript struct {
	title    string
	intro    string
	patterns []string
	rtitle   string
	item     string
	itemDesc string
	wait     string
	flavor   string
	curse    string
	skills   []SkillSpec
}

var jobScripts = map[string]jobScript{
	"shoko": {"Infirmary Assistant", "\"You want to help? Fine — nurses are cheaper than curses. Start with the supply order, then I'll show you which end of a bandage is which.\"",
		[]string{"fetch", "check", "wait", "hunt"}, "Shoko's Assistant", "Box of Medical Supplies", "Sterile gauze, antiseptic and a suspicious amount of cough syrup.",
		"Sterilize the instruments (wait about an hour)", "sick", "Ward-Sickness Curse", []SkillSpec{
			{Name: "Bedside Manner", Desc: "Calm hands, calm voice. Your healing hits a little harder.", Kind: "passive", Mod: "heal", Amount: 2},
			{Name: "Field Dressing", Desc: "Slap a bandage on an ally mid-fight.", Kind: "active", Effect: "heal", Power: 1, VFX: "heal", Color: "#95d5b2"},
			{Name: "Triage Eye", Desc: "You can tell who's really hurt.", Kind: "passive", Mod: "skill", Skill: "medicine", Amount: 2}}},
	"guard": {"Neighborhood Watch", "\"Night's coming and my shift partner called in sick. Walk the beat with me — eyes open, mouth shut.\"",
		[]string{"visit", "check", "hunt"}, "Honorary Watchman", "", "", "", "shadow", "Alley Prowler", []SkillSpec{
			{Name: "Night Eyes", Desc: "Hours on the beat taught you to spot what doesn't belong.", Kind: "passive", Mod: "skill", Skill: "perception", Amount: 2},
			{Name: "Whistle Blast", Desc: "A shrill, official whistle. The enemy flinches (WIS save or stunned).", Kind: "active", Effect: "stun", Power: 1, VFX: "voice", Color: "#ffd166"}}},
	"shop": {"Shop Hand", "\"Another pair of hands? Sure. Stock the shelves, take a delivery, don't eat the merchandise.\"",
		[]string{"fetch", "check", "wait"}, "Shop Hand", "Crate of Goods", "Heavy, rattling, labeled FRAGILE in three languages.", "Count the till and tidy the shelves (wait about an hour)", "hunger", "Shoplifting Spirit", []SkillSpec{
			{Name: "Haggler", Desc: "You've watched a thousand bargains. Shops charge the whole party a little less.", Kind: "passive", Mod: "discount", Amount: 5},
			{Name: "Stock-Room Lifting", Desc: "Carrying crates builds character. And arms.", Kind: "passive", Mod: "skill", Skill: "athletics", Amount: 2}}},
	"inn": {"Innkeeper's Helper", "\"Dishes, firewood, and something's been scratching in the cellar. Help me out and your next night is on the house.\"",
		[]string{"fetch", "wait", "hunt"}, "Inn Helper", "Sack of Rice and Greens", "Tonight's dinner, still dirt-dusted.", "Keep the broth simmering (wait about an hour)", "hunger", "Cellar Gnawer", []SkillSpec{
			{Name: "Comfort Food", Desc: "A hot bowl for a friend in the middle of a fight.", Kind: "active", Effect: "heal", Power: 2, VFX: "heal", Color: "#f4a261"},
			{Name: "Well Rested", Desc: "Honest work and honest meals. A little tougher.", Kind: "passive", Mod: "hp", Amount: 3}}},
	"elder": {"Town Fixer", "\"Everyone brings me their problems. I'd love to bring some of them to you.\"",
		[]string{"check", "check", "visit", "hunt"}, "Town Fixer", "", "", "", "fear", "Neighborhood Dread", []SkillSpec{
			{Name: "Local Fame", Desc: "People know your face; shops charge the whole party a little less.", Kind: "passive", Mod: "discount", Amount: 5},
			{Name: "Peacemaker", Desc: "Brace an ally with a few calming words.", Kind: "active", Effect: "buff", Status: "guard", Power: 1, VFX: "shield", Color: "#9fc3ff"}}},
	"gojo": {"Gojo's Sweets Run", "\"Oh, you want to *help* me? Perfect. I'm out of dorayaki and Ijichi 'lost' my card.\"",
		[]string{"fetch", "wait"}, "Gojo's Favorite Errand Runner", "Limited-Edition Dorayaki", "Warm. Gojo will absolutely check whether you ate one.", "Wait out the line at the counter (about an hour)", "hunger", "Sugar Craving", []SkillSpec{
			{Name: "Sugar Rush", Desc: "Pure glucose. You act sooner in a fight.", Kind: "passive", Mod: "init", Amount: 2},
			{Name: "Infinite Sweet Tooth", Desc: "Hype yourself up like it's the last crepe on Earth.", Kind: "active", Effect: "buff", Status: "empowered", Power: 2, VFX: "aura_gold", Color: "#ffd166"}}},
	"monk": {"Keeper of the Shrine", "\"The shrine needs tending before dusk. Sweep, light incense, and listen carefully while you do.\"",
		[]string{"visit", "fetch", "wait"}, "Shrine Keeper", "Bundle of Incense", "Smells like sandalwood and old promises.", "Meditate by the shrine (wait about an hour)", "fear", "Neglected Offering", []SkillSpec{
			{Name: "Calm Mind", Desc: "Quiet breathing steadies your cursed energy.", Kind: "passive", Mod: "ce", Amount: 2},
			{Name: "Purifying Clap", Desc: "A sharp clap that leaves a curse exposed.", Kind: "active", Effect: "debuff", Status: "exposed", Power: 1, VFX: "clap", Color: "#e0aaff"}}},
	"yaga": {"The Principal's Errand", "\"Paperwork, my dear student, is also a kind of battlefield. Take this report where it needs to go.\"",
		[]string{"fetch", "check"}, "Yaga's Courier", "Sealed Report", "Stamped, signed, and absolutely not to be opened.", "", "noise", "Gossip Static", []SkillSpec{
			{Name: "Bureaucratic Patience", Desc: "You can wait out anyone. Persuasion checks improve.", Kind: "passive", Mod: "skill", Skill: "persuasion", Amount: 2}}},
	"maki": {"Tool Tester", "\"Test these on something that deserves it. Don't break them. Try not to break yourself.\"",
		[]string{"fetch", "hunt"}, "Maki's Field Tester", "Crate of Cursed Tools", "Smells like oil and bad ideas.", "", "rage", "Rusted Grudge", []SkillSpec{
			{Name: "Dull Blade Wisdom", Desc: "You learned to hit where it counts. Small, steady damage bonus.", Kind: "passive", Mod: "damage", Amount: 1}}},
	"ijichi": {"Assistant to the Assistant", "\"I-I'm not supposed to delegate this... but please. Forms, signatures, a stop at the shop. You'd be saving my life.\"",
		[]string{"fetch", "check"}, "Ijichi's Intern", "Stack of Forms", "Triplicate. All of it.", "", "noise", "Paperwork Phantom", []SkillSpec{
			{Name: "Overworked Reflexes", Desc: "Living on caffeine and panic. You act sooner.", Kind: "passive", Mod: "init", Amount: 2},
			{Name: "Paperwork Shield", Desc: "A binder over the ribs, somehow effective.", Kind: "passive", Mod: "ac", Amount: 1}}},
	"mercenary": {"Fixer's Courier", "\"No questions, no names. Pick something up, drop something off, and keep your ears open.\"",
		[]string{"fetch", "hunt", "check"}, "Fixer's Courier", "Unmarked Package", "It ticks very quietly. Probably fine.", "", "shadow", "Backroom Shade", []SkillSpec{
			{Name: "Shady Contacts", Desc: "You know a guy. Shops charge the whole party less.", Kind: "passive", Mod: "discount", Amount: 5}}},
}

var defaultScript = jobScript{"Odd Jobs", "\"Huh. Honestly? Yes. There's always something that needs doing around here.\"",
	[]string{"check", "fetch", "visit"}, "Local Helper", "Wrapped Parcel", "String-tied, smells faintly of the sea.", "Wait for the dust to settle (about an hour)", "fear", "Neighborhood Dread", []SkillSpec{
		{Name: "Odd Jobber", Desc: "You've done a bit of everything. Checks across the board get easier.", Kind: "passive", Mod: "skill", Skill: "investigation", Amount: 1},
		{Name: "Quick Fix", Desc: "Duct tape and optimism on an ally.", Kind: "active", Effect: "heal", Power: 1, VFX: "heal", Color: "#95d5b2"}}}

var (
	reTitleWord = regexp.MustCompile(`\b(assistant|apprentice|intern|deputy|sidekick|helper|partner|student|disciple)\b`)
	reErrand    = regexp.MustCompile(`\b(errand|favou?r|deliver|fetch|bring|run to|pick up)\b`)
	reHunt      = regexp.MustCompile(`\b(pest|infest|rats?|hunt|exorcis|clear out|monster)\b`)
	reMystery   = regexp.MustCompile(`\b(investigate|mystery|missing|find out|look into|rumou?r)\b`)
	reMentor    = regexp.MustCompile(`\b(teach|train me|lesson|mentor|learn from|master)\b`)
)

func (e *Engine) townOf(n *NPC) string { return n.Town }

// pickNPC finds another NPC (optionally of a role), preferring the same town.
func (e *Engine) pickNPC(from *NPC, roles ...string) *NPC {
	var same, any []*NPC
	for _, n := range e.allNPCs() {
		if n.ID == from.ID {
			continue
		}
		ok := len(roles) == 0
		for _, r := range roles {
			if n.Role == r {
				ok = true
			}
		}
		if !ok {
			continue
		}
		if n.Town == from.Town {
			same = append(same, n)
		}
		any = append(any, n)
	}
	if len(same) > 0 {
		return same[rand.IntN(len(same))]
	}
	if len(any) > 0 {
		return any[rand.IntN(len(any))]
	}
	return nil
}

// ImprovQuest builds a quest tailored to the player's request and the NPC they're asking.
// It returns the NPC's spoken line. Caller holds the lock; the quest is returned in the "offered" stage.
func (e *Engine) ImprovQuest(ch *Character, n *NPC, text string) (*Quest, string) {
	t := strings.ToLower(text)
	sc, ok := jobScripts[n.Role]
	if !ok {
		sc = defaultScript
	}
	switch {
	case reMentor.MatchString(t) && n.Role != "shoko":
		sc.title, sc.patterns = "Lessons from "+n.Name, []string{"wait", "hunt", "check"}
		sc.intro = "\"You want to learn? Then work first. Do exactly as I say and complain later.\""
		sc.wait = "Run drills until your arms shake (wait about an hour)"
		sc.rtitle = n.Name + "'s Student"
	case reMystery.MatchString(t):
		sc.title, sc.patterns = "Looking Into It", []string{"check", "visit", "hunt"}
		sc.intro = "\"If you really want to dig into this, start by talking to people. Then we'll see how deep it goes.\""
	case reHunt.MatchString(t):
		sc.title, sc.patterns = "Pest Control", []string{"visit", "hunt"}
		sc.intro = "\"Something's moved in and it won't leave. If you can handle it, I'd be grateful.\""
	case reErrand.MatchString(t) && !reJob.MatchString(t):
		sc.title, sc.patterns = "Errand for "+n.Name, []string{"fetch"}
		sc.intro = "\"An errand? Yes, actually — that would help me more than you know.\""
	}
	if m := reTitleWord.FindString(t); m != "" {
		first := strings.Fields(n.Name)[0]
		sc.rtitle = first + "'s " + strings.ToUpper(m[:1]) + m[1:]
		sc.title = strings.ToUpper(m[:1]) + m[1:] + " to " + n.Name
	}

	shop := e.pickNPC(n, "shop", "inn")
	person := e.pickNPC(n, "villager", "guard", "elder", "monk", "sorcerer", "inn", "shop")
	if person == nil {
		person = shop
	}
	if shop == nil {
		shop = person
	}
	q := &Quest{Kind: "ai", Giver: n.ID, Stage: "offered", Title: sc.title, For: ch.ID, Report: true, RewardTitle: sc.rtitle, MinLevel: 1}
	nm := n.Name
	for _, p := range sc.patterns {
		switch p {
		case "fetch":
			if shop == nil {
				continue
			}
			it := sc.item
			if it == "" {
				it = defaultScript.item
			}
			q.Steps = append(q.Steps,
				QuestStep{Type: "talk", Target: shop.ID, Text: fmt.Sprintf("Pick up %s's order from %s", nm, shop.Name), GiveOnDone: it,
					Say: pickS(rngTL(), []string{"Oh, it's for " + nm + "? Right, it's all packed. Careful with it.", "Ah — " + nm + "'s order. Here you go, don't drop it.", "Finally someone shows up for this. It's all yours."})},
				QuestStep{Type: "deliver", Target: n.ID, Item: it, Count: 1, Text: fmt.Sprintf("Bring the %s back to %s", it, nm),
					Say: pickS(rngTL(), []string{"Perfect timing. Thank you.", "Good. Right on schedule.", "You actually brought it intact. Impressive."})})
		case "check":
			if person == nil {
				continue
			}
			q.Steps = append(q.Steps, QuestStep{Type: "talk", Target: person.ID, Text: fmt.Sprintf("Check in on %s — %s sent you", person.Name, nm),
				Say: pickS(rngTL(), []string{"Oh! " + nm + " sent you? Tell them I'm doing fine, better than fine.", "Huh, " + nm + " is checking up on me? ...Tell them thanks.", "Tell " + nm + " I haven't forgotten about it."})})
		case "wait":
			w := sc.wait
			if w == "" {
				w = defaultScript.wait
			}
			q.Steps = append(q.Steps, QuestStep{Type: "wait", Hours: 1, Text: w})
		case "visit":
			if s := e.pickSite(n.X, n.Y, nil); s != nil {
				q.Steps = append(q.Steps, QuestStep{Type: "goto", Target: s.ID, Text: fmt.Sprintf("Have a look around %s for %s", s.Name, nm)})
			}
		case "hunt":
			s := e.pickSite(n.X, n.Y, nil)
			if s == nil {
				continue
			}
			rank := min(2, (e.PartyLevel()-1)/2)
			cn := sc.curse
			if cn == "" {
				cn = defaultScript.curse
			}
			ct, err := e.NewCustomCurse(cn, fmt.Sprintf("A curse that grew around %s, feeding on everyone's quiet worry.", s.Name), sc.flavor, rank, false)
			if err != nil {
				continue
			}
			_, size := e.partyBudget()
			q.Steps = append(q.Steps, QuestStep{Type: "defeat", Target: s.ID, Text: fmt.Sprintf("Exorcise the %s lurking near %s", ct.Name, s.Name), Spawns: []Spawn{{Type: ct.ID, Count: min(2, max(1, size))}}})
		}
	}
	if len(q.Steps) == 0 {
		return nil, "\"Hm. I'd say yes, but I can't think of anything. Check back later.\""
	}
	budget, _ := e.partyBudget()
	q.RewardXP = max(60, budget/2)
	q.RewardYen = 5000 + budget*30
	if len(sc.skills) > 0 {
		s := sc.skills[rand.IntN(len(sc.skills))]
		q.RewardSkill = &s
	}
	q.Desc = sc.intro + "\n\nSteps:" + stepsList(q.Steps)
	e.prepareAll(q)
	e.addQuest(q)
	e.syncQuest(q)
	return q, sc.intro
}

func stepsList(steps []QuestStep) string {
	var b strings.Builder
	for i, s := range steps {
		fmt.Fprintf(&b, "\n %d. %s", i+1, s.Text)
	}
	return b.String()
}

func (e *Engine) prepareAll(q *Quest) {
	kept := q.Steps[:0]
	for _, s := range q.Steps {
		if err := e.prepareStep(&s); err == nil {
			kept = append(kept, s)
		}
	}
	q.Steps = kept
}
