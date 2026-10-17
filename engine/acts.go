package engine

import (
	"fmt"
	"math"
	"strings"
)

// The main story: five acts with branching consequences tracked in flags.
//
// Act 1  Sukuna's Finger    (Gojo)    — who ends up with the finger?
// Act 2  The Patchwork Curse (Nanami) — can you save Junpei from Mahito?
// Act 3  Disaster Curses    (Gojo)    — defend a town from Jogo, or hunt Hanami?
// Act 4  Goodwill Event     (Yaga)    — Kyoto's students, and an invasion
// Act 5  Shibuya Incident   (Gojo)    — is Gojo sealed? Who stands with you?

var actLevels = map[int]int{1: 1, 2: 3, 3: 5, 4: 7, 5: 9}

func (e *Engine) flag(k string) string { return e.C.Flags[k] }

func (e *Engine) gate(act int) (bool, string) {
	need := actLevels[act]
	if e.PartyLevel() >= need {
		return true, ""
	}
	return false, fmt.Sprintf("(This mission needs a party of about level %d. Take side jobs from town mayors and Nanami, and exorcise roaming curses to grow stronger.)", need)
}

func (e *Engine) storyOptions(ch *Character, n *NPC, d *Dialog, ctx *dialogCtx) {
	switch n.Role {
	case "gojo":
		switch {
		case e.C.Act == 0:
			d.opt("⭐ \"What's our first mission?\"", "story:act1_offer", ctx)
		case e.flag("act2") == "done" && e.flag("act3") == "":
			d.opt("⭐ \"Anything big going on?\"", "story:act3_offer", ctx)
		case e.flag("act4") == "done" && e.flag("act5") == "":
			d.opt("⭐ \"Sensei... is something wrong?\"", "story:act5_offer", ctx)
		case e.flag("act1") == "done" && e.flag("act2") == "":
			d.Text += " Oh, Nanami was looking for you. Something about overtime. He hates overtime."
		}
		d.opt("📖 Teach me about jujutsu.", "story:lore_gojo", ctx)
	case "nanami":
		if e.flag("act1") == "done" && e.flag("act2") == "" {
			d.opt("⭐ \"You were looking for me?\"", "story:act2_offer", ctx)
		}
	case "yaga":
		if e.flag("act3") == "done" && e.flag("act4") == "" {
			d.opt("⭐ \"Principal, you called for us?\"", "story:act4_offer", ctx)
		}
		d.opt("📖 Why do sorcerers fight?", "story:lore_yaga", ctx)
	case "junpei":
		if e.flag("act2") == "find" {
			d.opt("⭐ \"Hey. Can we talk?\"", "story:junpei", ctx)
		}
	}
}

var gojoLore = []string{
	"Cursed energy comes from negative emotions. Everyone leaks it. Sorcerers just know how to use it. Simple, right?",
	"A Domain Expansion builds your innate domain with cursed energy. Inside, your technique is guaranteed to hit. Mine? Infinite information. Don't think about it too hard~",
	"Black Flash happens when you hit within 0.000001 seconds of your cursed energy. Nobody can do it on purpose. Hit hard enough, often enough, and you'll find out what it feels like.",
	"Binding Vows: make a promise, gain power. Explain your technique to your enemy and it gets stronger. Weird? That's jujutsu.",
	"Reverse Cursed Technique: multiply negative energy by negative energy and you get positive energy. Then you can heal. Shoko does it by 'feel'. Nobody understands her explanations.",
	"Simple Domain was made to counter Domain Expansions. If someone expands on you and you can't expand back... use it. Or run.",
	"Curses are graded 4 to Special. A Grade 1 needs a Grade 1 sorcerer. A Special Grade needs... well, me.",
}

func (e *Engine) storyAction(ch *Character, n *NPC, arg string, ctx *dialogCtx) *Dialog {
	d := e.newDialog(n, "")
	closeOpt := func() { d.opt("👋 Got it.", "close", ctx) }
	switch arg {
	case "lore_gojo":
		d.Text = pickS(rngTL(), gojoLore)
		d.opt("📖 Tell me more.", "story:lore_gojo", ctx)
		closeOpt()
	case "lore_yaga":
		d.Text = "\"Jujutsu sorcerers do not die without regrets.\" That's what we say. You'll be called to fight things that come from people's worst feelings, and to die young doing it. If you can't find a reason you can live with — leave now. That is not cowardice."
		closeOpt()

	case "act1_offer":
		d.Text = "Your first mission! Tiny one. Super easy. There's a Special Grade cursed object — one of Ryomen Sukuna's twenty fingers — sitting in an old school, and the curses around it have been eating it like candy. Go get it back~ Oh, and whatever ate it might be a little... big."
		d.opt("✅ \"Leave it to us.\"", "story:act1_accept", ctx)
		d.opt("❓ \"Who's Sukuna?\"", "story:act1_sukuna", ctx)
		d.opt("Later.", "close", ctx)
	case "act1_sukuna":
		d.Text = "The King of Curses. A sorcerer from a thousand years ago so strong that even dead, his twenty fingers can't be destroyed — only sealed. If someone ate all twenty... yeah. Let's not."
		d.opt("✅ \"Okay. We'll get the finger.\"", "story:act1_accept", ctx)
		closeOpt()
	case "act1_accept":
		if e.flag("act1") != "" {
			d.Text = "You already have your orders~"
			closeOpt()
			break
		}
		hx, hy := float64(e.W.Towns[0].X), float64(e.W.Towns[0].Y)
		s := e.pickSite(hx, hy, []string{"school", "detention", "hospital"})
		if s == nil {
			d.Text = "Huh, no cursed sites around? Lucky."
			closeOpt()
			break
		}
		size := max(1, len(e.livingChars()))
		q := e.addQuest(&Quest{Kind: "main", Act: 1, Giver: "gojo", Site: s.ID, X: s.X, Y: s.Y, Stage: "active", MinLevel: 1,
			Title: "Act I — Sukuna's Finger", Objective: "Retrieve Sukuna's finger from " + s.Name + ".",
			Desc:      "A finger of the King of Curses is hidden at " + s.Name + ", drawing curses like moths to a flame.",
			Encounter: act1Encounter(size),
			RewardXP:  300 * size, RewardYen: 50000, RewardItem: "ce_tonic"})
		e.C.Act, e.C.Flags["act1"] = 1, "active"
		e.C.Quest = q.Title
		e.Log("system", ch.Name, "📜 Main quest: "+q.Title+" — "+q.Objective)
		d.Text = "Great! Ijichi can drive you if you get tired of walking. Don't die~ It'd be such a pain to explain."
		closeOpt()

	case "act2_offer":
		if ok, why := e.gate(2); !ok {
			d.Text = "Not yet. " + why
			closeOpt()
			break
		}
		t := e.storyTown()
		d.Text = fmt.Sprintf("Three bodies were found in %s. Not killed — reshaped. Twisted into things that shouldn't be able to live. Witnesses saw a high-school boy named Junpei Yoshino talking to a man with patchwork skin. Find the boy first. Carefully. He may not know what he's dealing with.", t.Name)
		d.opt("✅ \"We'll find Junpei.\"", "story:act2_accept", ctx)
		d.opt("Later.", "close", ctx)
	case "act2_accept":
		if e.flag("act2") != "" {
			d.Text = "You have your assignment."
			closeOpt()
			break
		}
		t := e.storyTown()
		e.C.Flags["act2_town"] = t.ID
		jx, jy := float64(t.X)-3.5, float64(t.Y)+3.5
		for i := 0; i < 20 && !e.W.Walkable(jx, jy); i++ {
			jx += 1
		}
		e.C.ExtraNPCs = append(e.C.ExtraNPCs, &NPC{ID: "junpei", Name: "Junpei Yoshino", Role: "junpei", X: jx, Y: jy, Town: t.ID,
			Look: Appearance{Skin: 0, Hair: 5, HairColor: "#2b2d42", Eyes: "#3b2a1a", Outfit: 4, Color1: "#264653", Color2: "#e9c46a", Build: 0},
			Desc: "A quiet high-schooler with a long fringe over his eyes. He's been skipping school. He looks like he hasn't slept."})
		q := e.addQuest(&Quest{Kind: "main", Act: 2, Giver: "nanami", X: int(jx), Y: int(jy), Stage: "active", MinLevel: 3,
			Title: "Act II — The Patchwork Curse", Objective: "Find Junpei Yoshino in " + t.Name + " and talk to him.",
			Desc: "Bodies reshaped into horrors. A lonely boy. A smiling man with stitches for skin.", RewardXP: 900 * max(1, len(e.livingChars())) / 2, RewardYen: 120000})
		e.C.Act, e.C.Flags["act2"] = 2, "find"
		e.C.Quest = q.Title
		e.Log("system", ch.Name, "📜 Main quest: "+q.Title+" — "+q.Objective)
		d.Text = "Good. And — it's not overtime yet. Let's keep it that way."
		closeOpt()
	case "junpei":
		d.Text = "Junpei won't meet your eyes. \"...Are you with them? The people who 'protect' everyone? Nobody protected me. Mahito-san... he listens. He says people's souls are just shapes. He says that means nothing anyone does to me really matters.\""
		d.opt("💬 \"That sounds lonely. Tell me what happened to you.\"", "story:junpei_listen", ctx)
		d.opt("⚠️ \"That man is a curse. He's using you.\"", "story:junpei_warn", ctx)
		d.opt("😠 \"Where is he? Tell me now.\"", "story:junpei_threat", ctx)
	case "junpei_listen":
		e.C.Flags["junpei_trust"] = "1"
		d.Text = "He talks for a long time — about the bullies, the burns on his forehead, his mom working late. When he finishes, he looks surprised anyone listened. \"...He's at the old building near town. He said something big happens tonight. Please... don't let him hurt my mom.\""
		e.act2Reveal(ch)
		closeOpt()
	case "junpei_warn":
		res := ch.Check("insight", 12, 0)
		e.Log("roll", ch.Name, "🎲 "+res.String())
		if res.Success {
			e.C.Flags["junpei_trust"] = "1"
			d.Text = "You choose your words carefully. Something in your voice gets through. \"...He did say he wanted to see what my mom's soul looks like. I thought it was a joke.\" He tells you where Mahito hides."
		} else {
			e.C.Flags["junpei_trust"] = "0"
			d.Text = "\"You're just like everyone else.\" He pulls away — but lets slip where he's going tonight."
		}
		e.act2Reveal(ch)
		closeOpt()
	case "junpei_threat":
		e.C.Flags["junpei_trust"] = "0"
		d.Text = "He flinches, then bolts. A crumpled flyer falls from his bag with an address scrawled on it — and a drawing of a man with stitches."
		e.npcState("junpei").Gone = true
		e.act2Reveal(ch)
		closeOpt()

	case "act3_offer":
		if ok, why := e.gate(3); !ok {
			d.Text = "Hmm, you're not quite ready for this one~ " + why
			closeOpt()
			break
		}
		t := e.storyTown()
		s := e.pickSite(float64(t.X), float64(t.Y), []string{"grove", "shrine", "lake"})
		if s == nil {
			s = e.pickSite(float64(t.X), float64(t.Y), nil)
		}
		e.C.Flags["act3_town"], e.C.Flags["act3_site"] = t.ID, s.ID
		e.C.Flags["act3"] = "offered"
		e.offerChoice(&Choice{ID: "act3", Source: "act3", Speaker: "Satoru Gojo", Title: "Two Disasters at Once",
			Text: fmt.Sprintf("Bad news~ Two Disaster Curses are moving. Jogo — a walking volcano — is heading for %s. Hanami, a forest curse, is nesting at %s, growing stronger by the day. I've got a meeting with the Higher-ups I can't skip. You can only stop one in time. Choose.", t.Name, s.Name),
			Options: []ChoiceOption{
				{ID: "town", Text: fmt.Sprintf("Defend %s from Jogo", t.Name), Result: "The people come first. You head for the town — knowing Hanami will grow stronger."},
				{ID: "hanami", Text: "Hunt Hanami before she grows", Result: fmt.Sprintf("You head for the forest. Behind you, smoke is already rising over %s.", t.Name)},
			}})
		d.Text = "(Gojo puts the decision to the party.)"
		closeOpt()

	case "act4_offer":
		if ok, why := e.gate(4); !ok {
			d.Text = "Not yet. " + why
			closeOpt()
			break
		}
		vessel := e.vessel()
		text := "The Kyoto Goodwill Event is today — a competition between our school and Kyoto's. "
		if vessel != nil {
			text += fmt.Sprintf("But I have it on good authority that Kyoto's principal ordered his students to kill %s during the event. The Higher-ups want the vessel dead. ", vessel.Name)
		} else {
			text += "Kyoto's students are proud and they've been told our school is reckless. Expect them to come at you hard. "
		}
		text += "How you handle them is up to you."
		e.C.Flags["act4"] = "offered"
		e.offerChoice(&Choice{ID: "act4", Source: "act4", Speaker: "Principal Yaga", Title: "The Goodwill Event", Text: text,
			Options: []ChoiceOption{
				{ID: "fight", Text: "Beat them fair and square", Result: "\"Then show them what Tokyo is made of.\""},
				{ID: "talk", Text: "Convince them to stand down", Need: "Will check (DC 15)", Result: "check:persuasion:15|The Kyoto students lower their weapons. \"...Fine. We'll do this properly.\" They stand with you.|\"Nice speech. Now fight.\"||flag:ally_kyoto=1"},
			}})
		d.Text = "(Yaga puts the decision to the party.)"
		closeOpt()

	case "act5_offer":
		if ok, why := e.gate(5); !ok {
			d.Text = "Hmm, not yet. " + why
			closeOpt()
			break
		}
		e.C.Flags["act5"] = "offered"
		e.offerChoice(&Choice{ID: "act5", Source: "act5", Speaker: "Satoru Gojo", Title: "Halloween in Shibuya",
			Text: "October 31st. A veil just came down over Shibuya Station, trapping thousands of people inside. The curses are asking for me. Only me. It's obviously a trap~ I'm going anyway. You can come with me to the station, or get the civilians out first.",
			Options: []ChoiceOption{
				{ID: "with_gojo", Text: "Go with Gojo to the station", Result: "\"Heh. Don't slow me down.\" He smiles — the real one, for once."},
				{ID: "evacuate", Text: "Evacuate the civilians first", Effects: []string{"rep:civ+3"}, Result: "Thousands of people stream out through the gaps you open in the veil. By the time you reach the station, it's quiet. Too quiet."},
			}})
		d.Text = "(Gojo puts the decision to the party.)"
		closeOpt()
	default:
		d.Text = "..."
		closeOpt()
	}
	return d
}

// storyTown picks the most fitting town for the next story beat (not the High or the city).
func (e *Engine) storyTown() *Town {
	var best *Town
	for i := range e.W.Towns {
		t := &e.W.Towns[i]
		if t.Kind == "school" || t.Kind == "city" || e.C.Flags["burned_"+t.ID] == "1" || t.ID == e.C.Flags["act2_town"] {
			continue
		}
		if best == nil || math.Hypot(float64(t.X-e.W.Towns[0].X), float64(t.Y-e.W.Towns[0].Y)) < math.Hypot(float64(best.X-e.W.Towns[0].X), float64(best.Y-e.W.Towns[0].Y)) {
			best = t
		}
	}
	if best == nil {
		best = &e.W.Towns[len(e.W.Towns)-1]
	}
	return best
}

func (e *Engine) vessel() *Character {
	for _, ch := range e.sortedChars() {
		if ch.Vessel && !ch.Dead {
			return ch
		}
	}
	return nil
}

func (e *Engine) mainQuest(act int) *Quest {
	for _, q := range e.C.Quests {
		if q.Kind == "main" && q.Act == act && q.Stage != "done" {
			return q
		}
	}
	return nil
}

func (e *Engine) act2Reveal(ch *Character) {
	q := e.mainQuest(2)
	t := e.W.Town(e.C.Flags["act2_town"])
	if q == nil || t == nil {
		return
	}
	s := e.pickSite(float64(t.X), float64(t.Y), []string{"factory", "apartment", "school", "detention"})
	if s == nil {
		return
	}
	size := max(1, len(e.livingChars()))
	q.Site, q.X, q.Y = s.ID, s.X, s.Y
	q.Objective = "Stop the patchwork curse at " + s.Name + "."
	q.Encounter = act2Encounter(size)
	e.C.Flags["act2"] = "hunt"
	e.Log("system", ch.Name, "📜 "+q.Title+": "+q.Objective)
}

// afterStoryChoice handles party decisions that start story fights.
func (e *Engine) afterStoryChoice(source, opt string, ch *Character) {
	size := max(1, len(e.livingChars()))
	switch source {
	case "act1":
		e.C.Flags["finger"] = opt
		if q := e.mainQuest(1); q != nil {
			q.Objective = "Report back to Gojo."
		}
	case "act2":
		if opt == "school" {
			e.C.Flags["ally_junpei"] = "1"
		}
	case "act3":
		t := e.W.Town(e.C.Flags["act3_town"])
		s := e.W.Site(e.C.Flags["act3_site"])
		e.C.Flags["act3"] = opt
		var q *Quest
		if opt == "town" {
			e.C.Flags["hanami_strong"] = "1"
			q = &Quest{Kind: "main", Act: 3, Giver: "gojo", X: t.X, Y: t.Y + 2, Title: "Act III — Disaster Flames", MinLevel: 5,
				Objective: "Stop Jogo before " + t.Name + " burns.", Desc: "A curse born of fear of the earth itself is marching on " + t.Name + ".",
				Encounter: act3Jogo(size)}
		} else {
			q = &Quest{Kind: "main", Act: 3, Giver: "gojo", Site: s.ID, X: s.X, Y: s.Y, Title: "Act III — The Forest's Wrath", MinLevel: 5,
				Objective: "Hunt Hanami at " + s.Name + ".", Desc: "A Disaster Curse of the forest, nesting where the trees whisper.",
				Encounter: act3Hanami(size)}
		}
		q.Stage, q.RewardXP, q.RewardYen = "active", 2700*size/2, 300000
		e.addQuest(q)
		e.C.Act = 3
		e.C.Quest = q.Title
		e.Log("system", ch.Name, "📜 Main quest: "+q.Title+" — "+q.Objective)
	case "act4":
		hx, hy := e.W.Towns[0].X, e.W.Towns[0].Y
		var enc []Spawn
		if e.C.Flags["ally_kyoto"] != "1" {
			enc = act4Kyoto(size)
		}
		q := &Quest{Kind: "main", Act: 4, Giver: "yaga", X: hx, Y: hy + 3, Stage: "active", MinLevel: 7, Title: "Act IV — The Goodwill Event",
			Objective: "Face the Kyoto students on the school grounds.", Desc: "Rivalry, pride — and something watching from outside the veil.",
			Encounter: enc, RewardXP: 6500 * size / 3, RewardYen: 500000}
		if len(enc) == 0 {
			e.act4Invasion(q)
		}
		e.addQuest(q)
		e.C.Act = 4
		e.C.Quest = q.Title
		e.Log("system", ch.Name, "📜 Main quest: "+q.Title+" — "+q.Objective)
	case "act5":
		e.C.Flags["act5"] = opt
		if opt == "with_gojo" && (e.C.Rep["gojo"] >= 2 || e.C.Flags["finger"] == "gojo" || e.C.Flags["ally_junpei"] == "1") {
			e.C.Flags["gojo_free"] = "1"
		} else {
			e.C.Flags["gojo_sealed"] = "1"
		}
		city := e.W.Town("city")
		if city == nil {
			city = &e.W.Towns[0]
		}
		enc := act5Encounter(size, e.C.Flags["gojo_free"] == "1", e.C.Flags["mahito_escaped"] == "1", e.C.Flags["act3"] == "hanami" && e.C.Flags["jogo_dead"] != "1")
		q := &Quest{Kind: "main", Act: 5, Giver: "gojo", X: city.X, Y: city.Y, Stage: "active", MinLevel: 9, Title: "Act V — The Shibuya Incident",
			Objective: "Reach Shibuya Station and stop the man with the stitches.", Desc: "The veil over Shibuya. A thousand-year-old plan. A prison made of a cube.",
			Encounter: enc, RewardXP: 30000, RewardYen: 2000000}
		if e.C.Flags["gojo_sealed"] == "1" {
			q.Giver = "yaga"
		}
		e.addQuest(q)
		e.C.Act = 5
		e.C.Quest = q.Title
		e.Log("system", ch.Name, "📜 Main quest: "+q.Title+" — "+q.Objective)
	case "act5_vessel":
		if opt != "sukuna" && e.C.Flags["sukuna_out"] != "1" {
			e.C.Flags["act5"] = "won"
			e.epilogue()
			return
		}
		{
			q := e.mainQuest(5)
			if q != nil {
				q.Stage = "active"
				q.Encounter = []Spawn{{Type: "sukuna", Count: 1, Power: sukunaPower(size), Name: "Ryomen Sukuna"}}
				q.Objective = "Survive the King of Curses."
				e.C.Flags["sukuna_free"] = "1"
			}
		}
	}
}

func (e *Engine) act4Invasion(q *Quest) {
	size := max(1, len(e.livingChars()))
	q.Objective = "A Special Grade curse breaks through the veil! Drive it off."
	if e.C.Flags["hanami_strong"] == "1" {
		q.Encounter = act4Invader("hanami", size)
	} else {
		q.Encounter = act4Invader("dagon", size)
	}
	e.C.Flags["act4"] = "invasion"
}

// mainVictory runs after the party wins a main-story fight.
func (e *Engine) mainVictory(q *Quest, party []*Character) {
	switch q.Act {
	case 1:
		holder := party[0]
		it, _ := NewItem("sukuna_finger")
		holder.AddItem(it)
		e.Log("system", holder.Name, "🩸 "+holder.Name+" recovers Sukuna's Finger. It's warm. It twitches.")
		e.offerChoice(&Choice{ID: "act1", Source: "act1", Speaker: "DM", Title: "Sukuna's Finger",
			Text: "The finger lies in your hand, wrapped in old talisman paper. You can feel it whispering to whoever holds it. What do you do with it?",
			Options: []ChoiceOption{
				{ID: "higher", Text: "Seal it and hand it to the Higher-ups, as ordered", Effects: []string{"remove:sukuna_finger", "rep:higher+2"}, Result: "The Higher-ups' envoy takes it with gloved hands and no thanks. Somewhere, a vault door closes."},
				{ID: "gojo", Text: "Give it to Gojo personally", Effects: []string{"remove:sukuna_finger", "rep:gojo+2", "rep:higher-1"}, Result: "\"Oh? You don't trust the old men either? Smart kid.\" Gojo pockets the finger like it's a candy wrapper."},
				{ID: "swallow", Text: "Swallow it (the one who chooses)", Effects: []string{"remove:sukuna_finger", "vessel", "rep:higher-3", "flag:vessel=1"}, Result: "It goes down like a stone. For a heartbeat, a second mouth opens on your cheek and laughs. Then — silence. You are still you. For now."},
			}})
	case 2:
		if e.C.Flags["junpei_trust"] == "1" {
			e.offerChoice(&Choice{ID: "act2", Source: "act2", Speaker: "DM", Title: "Junpei",
				Text: "Mahito is gone, laughing as he went. In the corner, Junpei is shaking — Mahito's hand was on his face when you arrived. He's alive. He's looking at you.",
				Options: []ChoiceOption{
					{ID: "school", Text: "\"Come to Jujutsu High. You have cursed energy — learn to use it.\"", Effects: []string{"rep:gojo+1", "journal:Junpei Yoshino joined Jujutsu High after Mahito nearly reshaped him."}, Result: "\"...Okay.\" For the first time, he smiles."},
					{ID: "home", Text: "\"Go home to your mom. Forget all of this.\"", Effects: []string{"rep:civ+1", "journal:Junpei Yoshino went home to his mother."}, Result: "He nods and runs. You hope Mahito forgets him too."},
				}})
		} else {
			e.C.Flags["junpei_dead"] = "1"
			e.npcState("junpei").Gone = true
			e.Log("dm", "DM", "You find Junpei too late. Mahito's hand was already on his face. What's left of him doesn't look human anymore — and somewhere, Mahito is laughing. (Junpei's story ends here.)")
			e.C.Journal = append(e.C.Journal, "Junpei Yoshino was transfigured by Mahito. The party couldn't save him.")
		}
		if e.C.Flags["mahito_escaped"] == "1" {
			e.C.Journal = append(e.C.Journal, "Mahito escaped and will return.")
		}
	case 3:
		t := e.W.Town(e.C.Flags["act3_town"])
		if e.C.Flags["act3"] == "hanami" && t != nil {
			e.C.Flags["burned_"+t.ID] = "1"
			e.C.Rep["civ"] -= 2
			e.Log("dm", "DM", fmt.Sprintf("Hanami falls back into the earth, wounded deep. But when you return to the road, %s is ash. Jogo was there and gone. The survivors don't look at you.", t.Name))
			e.C.Journal = append(e.C.Journal, t.Name+" was burned by Jogo while the party hunted Hanami.")
		} else if t != nil {
			e.C.Flags["jogo_dead"] = "1"
			e.Log("dm", "DM", fmt.Sprintf("Jogo crumbles into smoke and cinders, cursing your name. %s still stands. Somewhere in the forest, Hanami grows.", t.Name))
			e.C.Journal = append(e.C.Journal, "The party destroyed Jogo and saved "+t.Name+". Hanami was left to grow.")
		}
	case 4:
		if e.C.Flags["act4"] != "invasion" {
			q.Stage = "active"
			e.act4Invasion(q)
			e.Log("dm", "DM", "The Kyoto students are still picking themselves up when the sky cracks. A veil you didn't draw slams down over the school — and something enormous steps through. (Move to face it!)")
			e.C.Flags["ally_kyoto"] = "1"
			return
		}
		e.Log("dm", "DM", "The Special Grade curse retreats through the broken veil. Tokyo and Kyoto students stand side by side, bloodied. Principal Yaga says nothing — but he looks proud.")
		e.C.Flags["act4"] = "won"
	case 5:
		if v := e.vessel(); v != nil && e.C.Flags["sukuna_free"] == "" {
			e.offerChoice(&Choice{ID: "act5_vessel", Source: "act5_vessel", Speaker: "Ryomen Sukuna", Title: "The King Stirs", For: v.ID,
				Text: fmt.Sprintf("As Kenjaku falls, %s's vision turns red. A voice rises from inside: \"You've weakened yourself, brat. Let me out — just for a minute.\"", v.Name),
				Options: []ChoiceOption{
					{ID: "resist", Text: "Fight it down (WIS check)", Need: "WIS DC 15", Result: "check:perception:15|You clamp down on the voice. It laughs, and sinks. \"Another time.\"|Your body stops being yours.|flag:sukuna_out=1"},
					{ID: "sukuna", Text: "Let him out", Result: "Black markings crawl across your skin. Sukuna stretches, grinning with your mouth. \"Now. Who wants to die first?\""},
				}})
			return
		}
		e.C.Flags["act5"] = "won"
		e.epilogue()
	}
}

func (e *Engine) mainTurnIn(ch *Character, q *Quest) string {
	e.C.Flags[fmt.Sprintf("act%d", q.Act)] = "done"
	switch q.Act {
	case 1:
		switch e.C.Flags["finger"] {
		case "gojo":
			return "Good work~ And thanks for trusting me with the finger. The old men would have used it for something stupid. Next up — Nanami has a job. Try not to annoy him."
		case "swallow":
			return "...You ate it. Of course you did. The Higher-ups want you executed, by the way! Don't worry, I talked them into a suspended sentence. You'll die once you've eaten all twenty. Fun, right? Go see Nanami."
		}
		return "Mission complete! The Higher-ups have their finger. Hope they choke on it~ Go see Nanami next."
	case 2:
		if e.C.Flags["junpei_dead"] == "1" {
			return "...I see. It's not your fault. Remember this feeling. Curses like Mahito feed on it. Take the evening off. That's an order."
		}
		return "You saved the boy. That's worth more than any exorcism. Mahito will be back, though. Curses like him always are."
	case 3:
		if e.C.Flags["act3"] == "hanami" {
			return "Hanami's wounded, but... a whole town. You made a choice no one should have to. That's jujutsu. Now get some rest."
		}
		return "You protected the town! See? Teaching is my real talent~ Hanami's still out there, though. Stay sharp."
	case 4:
		return "Well fought. Both schools will remember today. ...Something is coming. Something that planned all of this. Be ready."
	case 5:
		return "It's over. For now."
	}
	return "Well done."
}

func (e *Engine) epilogue() {
	var parts []string
	if e.C.Flags["gojo_free"] == "1" {
		parts = append(parts, "Gojo walks out of Shibuya Station unsealed, stretching. \"That cube thing was rude.\" The strongest is still free.")
	} else {
		parts = append(parts, "The Prison Realm sits on the station floor, a small cube holding the strongest sorcerer. The world will have to learn to live without him — until you find a way to open it.")
	}
	if e.C.Flags["sukuna_free"] == "1" {
		parts = append(parts, "The King of Curses walked the streets of Shibuya for one night. The vessel came back. Not everyone did.")
	} else if e.vessel() != nil {
		parts = append(parts, "The vessel kept control. For now, Sukuna sleeps.")
	}
	if e.C.Flags["ally_junpei"] == "1" {
		parts = append(parts, "Junpei Yoshino stood at the barrier, guiding civilians out — a sorcerer now.")
	}
	if e.C.Flags["ally_kyoto"] == "1" {
		parts = append(parts, "Kyoto's students fought beside you to the end.")
	}
	if e.C.Rep["civ"] >= 4 {
		parts = append(parts, "The people you saved remember your faces. Some of them will become Windows — the eyes that help sorcerers find curses.")
	} else if e.C.Rep["civ"] <= -2 {
		parts = append(parts, "The people of this land have lost faith in sorcerers. The curses born from that will be terrible.")
	}
	if e.C.Rep["higher"] <= -3 {
		parts = append(parts, "The Higher-ups have marked your names. They will not forget.")
	}
	parts = append(parts, "**The story continues.** Curses never stop being born. Take jobs, explore, grow — the world is yours.")
	e.Log("dm", "DM", "**Epilogue**\n\n"+strings.Join(parts, "\n\n"))
	e.C.Act = 6
	e.C.Journal = append(e.C.Journal, "The Shibuya Incident is over.")
}

// ---------- story encounters (tuned with TestBalance) ----------

func scaleP(base, per, size int) int { return base + per*min(size-1, 3) }

func act1Encounter(size int) []Spawn {
	return []Spawn{{Type: "finger_bearer", Count: 1, Power: scaleP(42, 8, size)}, {Type: "fly_head", Count: max(1, min(size-1, 2))}}
}

func act2Encounter(size int) []Spawn {
	return []Spawn{{Type: "mahito", Count: 1, Power: scaleP(34, 7, size), Name: "Mahito"}, {Type: "transfigured", Count: max(1, min(size-1, 2))}}
}

func act3Jogo(size int) []Spawn {
	return []Spawn{{Type: "jogo", Count: 1, Power: scaleP(36, 6, size)}, {Type: "fly_head", Count: min(size, 3), Name: "Ember Fly"}}
}

func act3Hanami(size int) []Spawn {
	return []Spawn{{Type: "hanami", Count: 1, Power: scaleP(36, 6, size)}, {Type: "grasshopper", Count: max(1, min(size-1, 2))}}
}

func act4Kyoto(size int) []Spawn {
	return []Spawn{{Type: "kyoto_student", Count: min(size+1, 4), Name: "Kyoto Student"}}
}

func act4Invader(kind string, size int) []Spawn {
	if kind == "hanami" {
		return []Spawn{{Type: "hanami", Count: 1, Power: scaleP(66, 6, size)}}
	}
	return []Spawn{{Type: "dagon", Count: 1, Power: scaleP(78, 6, size)}}
}

func act5Encounter(size int, gojoFree, mahito, jogo bool) []Spawn {
	power := scaleP(55, 6, size)
	if gojoFree {
		power = scaleP(45, 5, size)
	}
	enc := []Spawn{{Type: "kenjaku", Count: 1, Power: power}}
	if mahito {
		enc = append(enc, Spawn{Type: "mahito", Count: 1, Power: 30})
	}
	if jogo {
		enc = append(enc, Spawn{Type: "jogo", Count: 1, Power: 28})
	}
	return enc
}

func sukunaPower(size int) int { return scaleP(72, 7, size) }
