package engine

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// OfflineDM narrates without a network. Everything structural (quests, story,
// battles, shops) works the same in both modes; this handles free-form text.
// All methods are called with the engine lock held.
type OfflineDM struct {
	hinted bool
}

func (d *OfflineDM) Handle(e *Engine, ev DMEvent) {
	switch ev.Kind {
	case "start":
		e.C.Location = "Tokyo Jujutsu High"
		e.Log("dm", "DM", "**Spring. Tokyo Jujutsu High.**\n\nThe school hides in the wooded hills on the edge of the city, disguised as a religious college. You've each been scouted for the same reason: you can see curses — the monsters born from human fear, hatred and regret — and you have cursed energy of your own.\n\nA tall man with white hair and a black blindfold is waiting in the courtyard, waving far too enthusiastically.")
		e.Log("system", "", "🎮 WASD / arrow keys to move · E to talk to people · Enter to type anything your character does · Click the HUD for your sheet, quests and map. Go talk to Gojo!")
	case "join":
		if ch, ok := e.C.Characters[ev.Summary]; ok {
			e.Log("dm", "DM", fmt.Sprintf("A new student arrives at the gate: %s, a %s from the %s. Gojo waves them over.", ch.Name, Techniques[ch.Class].Name+" user", Origins[ch.Race].Name))
		}
	case "combat_end":
		if strings.HasPrefix(ev.Summary, "victory") {
			e.Log("dm", "DM", pickS(rngTL(), []string{
				"The curse dissolves into black smoke. The air feels lighter.",
				"Silence. Just your breathing and the hum of leftover cursed energy.",
				"Exorcised. Somewhere, someone sleeps easier tonight without knowing why.",
			}))
		}
	case "talk":
		n := e.npcByID(ev.Summary)
		for _, a := range ev.Actions {
			ch := e.C.Characters[a.CharID]
			if n != nil && ch != nil {
				d.npcReply(e, ch, n, a.Text)
			}
		}
	case "actions":
		for _, a := range ev.Actions {
			ch, ok := e.C.Characters[a.CharID]
			if !ok || ch.Dead {
				continue
			}
			d.resolve(e, ch, a.Text)
		}
	}
}

var (
	reSpeech   = regexp.MustCompile(`(^|\b)(talk|ask|say|tell|greet|hello|hi|hey|chat|speak|introduce|thank|apologi[sz]e|shout|yell|whisper)\b|["“”']|\?$`)
	reJob      = regexp.MustCompile(`\b(job|work|hire|hiring|become|join|apprentice|guard|employ|enlist|recruit|volunteer)\b`)
	reShop     = regexp.MustCompile(`\b(buy|sell|wares|for sale|price|cost|shop|store|purchase|trade)\b`)
	reWho      = regexp.MustCompile(`\b(who are you|your name|what do you do|about yourself)\b`)
	reRumor    = regexp.MustCompile(`\b(rumou?r|curse|monster|strange|weird|happen|news|disappear|missing|danger)\b`)
	reLoreTalk = regexp.MustCompile(`\b(gojo|sukuna|finger|jujutsu|domain|sorcerer|black flash|technique|higher.ups|kenjaku|mahito)\b`)
	reRest     = regexp.MustCompile(`\b(rest|sleep|nap|camp|lie down|catch (my|our) breath)\b`)
	reSearch   = regexp.MustCompile(`\b(search|investigate|look|examine|inspect|explore|scan|check)\b`)
	reSense    = regexp.MustCompile(`\b(sense|feel for|detect|track) .*(curse|energy|presence)|\bcurse sense\b|\bsense\b`)
	reTrain    = regexp.MustCompile(`\b(train|practice|meditate|spar|exercise|workout|focus my|study)\b`)
	reEat      = regexp.MustCompile(`\b(eat|drink|food|snack|ramen|meal|hungry|thirsty|coffee|tea)\b`)
	reFlee     = regexp.MustCompile(`\b(surrender|parley|negotiate|reason|persuade|convince|intimidate|threaten)\b`)
)

func (d *OfflineDM) resolve(e *Engine, ch *Character, text string) {
	t := strings.ToLower(strings.TrimSpace(text))
	if e.C.Combat != nil && ch.InBattle {
		d.battleTalk(e, ch, t)
		return
	}
	near := e.nearbyNPCs(ch.X, ch.Y, 4)
	var target *NPC
	for _, n := range near {
		first := strings.ToLower(strings.Fields(n.Name)[0])
		last := strings.ToLower(n.Name)
		if strings.Contains(t, first) || strings.Contains(t, last) || strings.Contains(t, n.Role) {
			target = n
			break
		}
	}
	if target == nil && len(near) > 0 && (reSpeech.MatchString(t) || reJob.MatchString(t) || reShop.MatchString(t) || reWho.MatchString(t)) {
		target = near[0]
	}
	switch {
	case target != nil:
		d.npcReply(e, ch, target, text)
	case reSense.MatchString(t):
		d.sense(e, ch)
	case reRest.MatchString(t):
		d.rest(e, ch)
	case reTrain.MatchString(t):
		d.train(e, ch)
	case reEat.MatchString(t):
		d.eat(e, ch)
	case reSearch.MatchString(t):
		d.search(e, ch)
	case reSpeech.MatchString(t):
		e.Log("dm", "DM", fmt.Sprintf("%s speaks, but there's nobody close enough to hear. (Walk up to someone first.)", ch.Name))
	default:
		d.generic(e, ch, t)
	}
	if !d.hinted && !e.AIActive() {
		d.hinted = true
		e.Log("system", "", "💡 The offline DM understands common actions. For truly anything-goes roleplay — any question, any plan, any NPC — the host can switch on the AI Dungeon Master in ⚙ Host (free with Ollama on a decent graphics card).")
	}
}

func (d *OfflineDM) battleTalk(e *Engine, ch *Character, t string) {
	human := false
	for _, m := range e.liveMonsters() {
		if Bestiary[m.Template].Human && !m.Boss {
			human = true
		}
	}
	if reFlee.MatchString(t) && human {
		res := ch.Check("persuasion", 16, 0)
		if strings.Contains(t, "threat") || strings.Contains(t, "intimidat") {
			res = ch.Check("intimidation", 16, 0)
		}
		e.Log("roll", ch.Name, "🎲 "+res.String())
		if res.Success {
			e.Log("dm", "DM", "Your opponents exchange a look, lower their weapons and back off.")
			e.endCombat("surrender")
			return
		}
		e.Log("dm", "DM", "\"Not a chance.\" They press the attack.")
		return
	}
	e.Log("dm", "DM", fmt.Sprintf("%s: curses don't negotiate — they only understand cursed energy. Pick a technique on your turn!", ch.Name))
}

// npcReply is the offline NPC conversation engine.
func (d *OfflineDM) npcReply(e *Engine, ch *Character, n *NPC, text string) {
	t := strings.ToLower(text)
	say := func(line string) { e.Log("say", n.Name, line) }
	switch {
	case reJob.MatchString(t) || reErrand.MatchString(t) || reHunt.MatchString(t) || reMystery.MatchString(t) || reMentor.MatchString(t):
		if q := e.questFrom(n.ID, "ai"); q != nil && (q.Stage == "active" || q.Stage == "return") {
			say("We already have something going: \"" + q.Title + "\". Finish that first.")
			return
		}
		q, line := e.ImprovQuest(ch, n, text)
		say(line)
		if q != nil {
			q.Stage = "active"
			e.Log("system", ch.Name, fmt.Sprintf("📜 New quest: %s — %s", q.Title, q.Objective))
		}
	case reShop.MatchString(t):
		if n.Role == "shop" || n.Role == "maki" || n.Role == "mercenary" {
			kind := map[string]string{"shop": "basic", "maki": "tools", "mercenary": "rare"}[n.Role]
			var names []string
			for _, id := range shopStock[kind] {
				names = append(names, fmt.Sprintf("%s (¥%s)", ItemList[id].Name, commas(e.price(ItemList[id].Value))))
			}
			say("Sure! I've got " + strings.Join(names, ", ") + ". (Press E to browse.)")
		} else {
			say("I'm not a shop. The general store is near the plaza.")
		}
	case reWho.MatchString(t):
		say(n.Desc)
	case reLoreTalk.MatchString(t) && (n.Role == "gojo" || n.Role == "yaga" || n.Role == "nanami" || n.Role == "maki"):
		say(pickS(rngTL(), gojoLore))
	case reRumor.MatchString(t):
		say(e.rumor(n))
	case strings.Contains(t, "thank"):
		say(pickS(rngTL(), []string{"No problem.", "Anytime.", "Just doing my part."}))
	default:
		say(smallTalk(e, ch, n, t))
	}
}

var (
	reGreet   = regexp.MustCompile(`\b(hi|hello|hey|yo|howdy|greetings|good (morning|evening|afternoon)|sup)\b`)
	reHowAre  = regexp.MustCompile(`how are you|how'?s it going|how have you been|you ok|you okay|what'?s up`)
	reFood    = regexp.MustCompile(`\b(food|eat|hungry|ramen|lunch|dinner|breakfast|snack|cook|delicious|favou?rite food)\b`)
	reWeather = regexp.MustCompile(`\b(weather|rain|sunny|cold|hot|night|dark|morning|day|sky)\b`)
	rePraise  = regexp.MustCompile(`\b(nice|cool|awesome|great|amazing|beautiful|pretty|handsome|love|thank|thanks|impressive|strong)\b`)
	reInsult  = regexp.MustCompile(`\b(stupid|idiot|dumb|ugly|hate|shut up|loser|useless|boring)\b`)
	reJoke    = regexp.MustCompile(`\b(joke|funny|lol|haha|pun|riddle)\b`)
	reMoney   = regexp.MustCompile(`\b(money|yen|rich|poor|pay|paid|salary|broke)\b`)
	reFamily  = regexp.MustCompile(`\b(family|mother|father|mom|dad|sister|brother|kids?|home|parents?)\b`)
	reFight   = regexp.MustCompile(`\b(fight|battle|duel|spar|kill|attack)\b`)
	reWhere   = regexp.MustCompile(`\b(where|which way|directions?|how do i get|lost)\b`)
)

// smallTalk gives the offline DM a believable answer to everyday chatter, flavoured by who is speaking.
func smallTalk(e *Engine, ch *Character, n *NPC, t string) string {
	pick := func(xs ...string) string { return pickS(rngTL(), xs) }
	first := strings.Fields(ch.Name)[0]
	night := e.C.Clock >= 20 || e.C.Clock < 5
	switch {
	case reInsult.MatchString(t):
		return pick("Wow. Okay. Rude.", "...I'm going to pretend I didn't hear that.", "Say that again and see what happens, "+first+".")
	case reGreet.MatchString(t):
		if night {
			return pick("Out this late, "+first+"? Careful. It's not a good hour.", "Evening. Keep your voice down — curses like the dark.")
		}
		return pick("Hey, "+first+"!", "Oh, hello. Nice to see a friendly face.", "Morning's better when someone says hi.")
	case reHowAre.MatchString(t):
		return pick("Tired, honestly. Everyone's on edge lately.", "Can't complain. Well — I could, but nobody asks.", "Better now that someone's talking to me.")
	case reFood.MatchString(t):
		switch n.Role {
		case "inn":
			return "Grilled fish with rice and miso — the whole town swears by it. Rent a room and you get a bowl."
		case "shop":
			return "I live on convenience-store onigiri. Don't tell my mother."
		default:
			return pick("Ramen. Obviously. Any time, any weather.", "Dorayaki, if I'm honest. Gojo ruined everything else for me.", "Hot soba on a cold day. That's the good stuff.")
		}
	case reWeather.MatchString(t):
		if night {
			return pick("It's night, so don't wander far from the lanterns.", "Clear sky, but the curses are out. Funny how that works.")
		}
		return pick("Nice enough. Good day for walking.", "The air's thick lately. Feels like before a storm — or a curse.")
	case rePraise.MatchString(t):
		return pick("Ha! You're too kind.", "Well, now I'm blushing.", "Thanks. That actually means a lot.")
	case reJoke.MatchString(t):
		return pick("What do you call a curse that pays rent? A landlord. ...I'll show myself out.", "Why did the sorcerer cross the road? Because the curse was on the other side.", "Ha. Okay, that's one for the road.")
	case reMoney.MatchString(t):
		if n.Role == "mercenary" {
			return "Everything has a price, kid. Even me."
		}
		return pick("Money's tight. Sorcerer work pays, I hear — that, or you could ask me for odd jobs.", "Don't ask me about yen. Ask the shopkeeper. Or do some work.")
	case reFamily.MatchString(t):
		return pick("Family's... complicated. Ask me again when we both have time.", "My family's fine. Worried, but fine.")
	case reFight.MatchString(t):
		return pick("Save it for the curses.", "I'd rather not. Ask someone who enjoys that sort of thing.")
	case reWhere.MatchString(t):
		if s := e.pickSite(n.X, n.Y, nil); s != nil {
			return fmt.Sprintf("If it's trouble you're after, try %s — about %d m %s of here. I'd bring friends.", s.Name, int(math.Hypot(float64(s.X)-n.X, float64(s.Y)-n.Y)), compass(float64(s.X)-n.X, float64(s.Y)-n.Y))
		}
		return "Follow the roads. Keep the school behind you — or in front of you, I always forget."
	}
	// Nothing matched: speak in character and nudge toward things the offline DM can actually do.
	lines := []string{
		n.Desc,
		"Hm. Not sure how to answer that. Ask me about work or rumors, maybe?",
		"I'll think about it. In the meantime — anything I can help with? Or the other way around?",
		"Interesting. Go on... or ask me something I can actually answer.",
	}
	if n.Role == "gojo" {
		lines = []string{"Hmm~ Interesting! Not sure I follow, but interesting!", "Ask me again after I've had some sweets.", "Heh. You're fun."}
	}
	return pick(lines...)
}

func (d *OfflineDM) sense(e *Engine, ch *Character) {
	res := ch.Check("curse_sense", 10, 0)
	e.Log("roll", ch.Name, "🎲 "+res.String())
	if !res.Success {
		e.Log("dm", "DM", ch.Name+" reaches out with their senses but feels only static.")
		return
	}
	var best *Roamer
	bd := math.MaxFloat64
	for _, r := range e.Roamers {
		if dd := math.Hypot(r.X-ch.X, r.Y-ch.Y); dd < bd {
			best, bd = r, dd
		}
	}
	if best == nil {
		e.Log("dm", "DM", "Nothing nearby. Just the faint hum of ordinary people's anxieties.")
		return
	}
	t := Bestiary[best.Kind]
	e.Log("dm", "DM", fmt.Sprintf("%s feels it: a %s presence about %d meters %s. Something like a %s.", ch.Name, strings.ToLower(t.Grade), int(bd*4), compass(best.X-ch.X, best.Y-ch.Y), t.Name))
}

func compass(dx, dy float64) string {
	a := math.Atan2(-dy, dx) * 180 / math.Pi
	dirs := []string{"east", "northeast", "north", "northwest", "west", "southwest", "south", "southeast"}
	i := int(math.Round(a/45)+8) % 8
	return "to the " + dirs[i]
}

func (d *OfflineDM) rest(e *Engine, ch *Character) {
	if t := e.W.InTown(ch.X, ch.Y); t != nil {
		if t.Kind == "school" {
			for _, o := range e.partyNear(ch, 10) {
				o.LongRest()
			}
			e.Log("dm", "DM", "You crash in the dorms. Morning comes with birdsong and Gojo banging on the door. (Fully rested.)")
			e.C.Clock = 7
			return
		}
		e.Log("dm", "DM", "You could nap on a bench — or get a proper room at the inn (talk to the innkeeper).")
		ch.ShortRest()
		return
	}
	if (e.C.Clock >= 20 || e.C.Clock < 5) && chance(0.35) {
		e.Log("dm", "DM", "You close your eyes... and something closes in.")
		if r, err := e.SpawnRoamerAt(ch.X+2, ch.Y, []string{"grave_hound", "fly_head"}, "", false); err == nil {
			r.cool = 0
		}
		return
	}
	var parts []string
	for _, o := range e.partyNear(ch, 6) {
		parts = append(parts, fmt.Sprintf("%s +%d HP", o.Name, o.ShortRest()))
	}
	e.Log("system", "", "☕ Short rest: "+strings.Join(parts, ", ")+" (and half their cursed energy back).")
}

func (d *OfflineDM) train(e *Engine, ch *Character) {
	key := "train_" + ch.ID
	if last, ok := e.C.Flags[key]; ok && last == fmt.Sprint(e.C.Day) {
		e.Log("dm", "DM", ch.Name+" has trained enough today. Muscles — and cursed energy — need rest to grow.")
		return
	}
	e.C.Flags[key] = fmt.Sprint(e.C.Day)
	res := ch.Check(ch.Tech().Priority[0], 12, 0)
	e.Log("roll", ch.Name, "🎲 "+res.String())
	if res.Success {
		xp := 30 * ch.Level
		e.Log("dm", "DM", fmt.Sprintf("%s drills their %s until it feels like breathing. (+%d XP)", ch.Name, ch.Tech().Name, xp))
		for _, m := range ch.GainXP(xp) {
			e.Log("level", ch.Name, m)
		}
	} else {
		e.Log("dm", "DM", ch.Name+" trains hard but can't find the rhythm today. Tomorrow.")
	}
}

func (d *OfflineDM) eat(e *Engine, ch *Character) {
	if e.W.InTown(ch.X, ch.Y) != nil {
		if ch.Gold >= 800 {
			ch.Gold -= 800
			got := ch.heal(4)
			e.Log("dm", "DM", fmt.Sprintf("%s grabs something warm from a konbini. ¥800. (+%d HP)", ch.Name, got))
			return
		}
		e.Log("dm", "DM", ch.Name+" checks their wallet. Not even ¥800 for an onigiri.")
		return
	}
	e.Log("dm", "DM", ch.Name+" eats a squashed rice ball from their pocket. It's fine. It's fine.")
}

func (d *OfflineDM) search(e *Engine, ch *Character) {
	var site *Site
	for i := range e.W.Sites {
		s := &e.W.Sites[i]
		if math.Hypot(float64(s.X)-ch.X, float64(s.Y)-ch.Y) < 8 {
			site = s
		}
	}
	res := ch.Check("investigation", 12, 0)
	e.Log("roll", ch.Name, "🎲 "+res.String())
	if site != nil {
		if res.Success {
			e.Log("dm", "DM", fmt.Sprintf("%s studies the %s: residue of cursed energy everywhere, claw marks, a %s's lingering stench. %s", ch.Name, site.Name, strings.ToLower(GradeNames[site.Rank]), map[bool]string{true: "It's been cleansed — mostly.", false: "Whatever lives here is still close."}[e.C.Cleared[site.ID]]))
			if chance(0.3) {
				it, _ := NewItem("cursed_trinket")
				ch.AddItem(it)
				e.Log("system", ch.Name, "🎁 "+ch.Name+" finds a low-grade cursed object.")
			}
			return
		}
		e.Log("dm", "DM", "The cursed energy here is so thick it's hard to think straight.")
		return
	}
	if res.Success && chance(0.35) {
		y := 500 + Die(10)*100
		ch.Gold += y
		e.Log("dm", "DM", fmt.Sprintf("%s finds ¥%s someone dropped. Lucky.", ch.Name, commas(y)))
		return
	}
	if t := e.W.InTown(ch.X, ch.Y); t != nil {
		e.Log("dm", "DM", fmt.Sprintf("%s looks around %s. %s", ch.Name, t.Name, t.Desc))
		return
	}
	e.Log("dm", "DM", ch.Name+" looks around. "+pickS(rngTL(), []string{"Wind in the grass. Crows. Nothing human.", "The road stretches on. A shrine bell rings somewhere far away.", "Footprints — too many toes. Old, at least."}))
}

func (d *OfflineDM) generic(e *Engine, ch *Character, t string) {
	verbs := []struct {
		re    *regexp.Regexp
		skill string
	}{
		{regexp.MustCompile(`\b(climb|jump|swim|lift|push|pull|break|smash|kick|punch|throw|carry|run|sprint)\b`), "athletics"},
		{regexp.MustCompile(`\b(balance|flip|vault|dodge|tumble|parkour)\b`), "acrobatics"},
		{regexp.MustCompile(`\b(sneak|hide|stealth|quietly|tiptoe)\b`), "stealth"},
		{regexp.MustCompile(`\b(steal|pickpocket|lockpick|pick the lock|unlock)\b`), "sleight_of_hand"},
		{regexp.MustCompile(`\b(lie|bluff|deceive|trick|disguise|pretend)\b`), "deception"},
		{regexp.MustCompile(`\b(sing|dance|perform|play|show off)\b`), "performance"},
		{regexp.MustCompile(`\b(barrier|veil|seal|ward|talisman)\b`), "barriers"},
		{regexp.MustCompile(`\b(heal|bandage|treat|first aid)\b`), "medicine"},
		{regexp.MustCompile(`\b(remember|recall|lore|history|know about)\b`), "jujutsu_lore"},
		{regexp.MustCompile(`\b(listen|watch|spot|notice|peek)\b`), "perception"},
	}
	for _, v := range verbs {
		if v.re.MatchString(t) {
			res := ch.Check(v.skill, 12, 0)
			e.Log("roll", ch.Name, "🎲 "+res.String())
			if v.skill == "barriers" && res.Success {
				e.Log("dm", "DM", fmt.Sprintf("%s chants: \"Emerging from darkness, darker than darkness, purify the impure.\" A veil drops over the area — no ordinary person will see what happens here.", ch.Name))
				return
			}
			if res.Success {
				e.Log("dm", "DM", fmt.Sprintf("%s pulls it off. %s", ch.Name, pickS(rngTL(), []string{"Smooth.", "Nobody saw a thing.", "Even Gojo would've been impressed. Probably.", "That went better than expected."})))
			} else {
				e.Log("dm", "DM", fmt.Sprintf("%s tries — and it doesn't work out. %s", ch.Name, pickS(rngTL(), []string{"Embarrassing.", "Maybe next time.", "A crow caws, mockingly."})))
			}
			return
		}
	}
	e.Log("dm", "DM", fmt.Sprintf("%s %s", ch.Name, pickS(rngTL(), []string{"does just that.", "goes for it. The world doesn't seem to mind.", "does so. Life goes on around them."})))
}
