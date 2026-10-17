package engine

// Content tables for the offline Dungeon Master.

type roomDef struct{ Name, Desc string }

type themeDef struct {
	Titles []string
	Places []string
	Hooks  []string
	Lore   []string
	Entry  []roomDef
	Rooms  []roomDef
	Lairs  []roomDef
	Sounds []string
	Intro  string
}

var Themes = map[string]themeDef{
	"crypt": {
		Titles: []string{"The Sunken Crypt of %s", "The Tomb of %s", "Barrow of %s the Forsaken", "The Ossuary Beneath %s"},
		Places: []string{"Varn Holloway", "Saint Ilmara", "Old Grimsby", "Kettlemoor", "the Ashen Abbey"},
		Hooks: []string{
			"Graves in the village churchyard have been found empty, their earth clawed open from below. The priest begs you to find where the dead are walking to.",
			"A lantern-bearer saw robed figures carrying a coffin into the old crypt at midnight. Since then, the dogs won't stop howling.",
			"The baron's daughter vanished near the barrow. Her silver locket was found on the crypt's threshold, cold as ice.",
		},
		Lore: []string{
			"Old texts speak of a necromancer who bound his soul to the crypt's foundation stones. Radiant light and holy words weaken the things he raised.",
			"The crypt was sealed by paladins a century ago. Whatever they locked inside has had a long time to grow hungry.",
		},
		Entry: []roomDef{
			{"Crypt Gate", "Iron gates hang crooked on rusted hinges. Stone steps slick with moss descend into a darkness that smells of wet earth and old bone."},
			{"Mourners' Stair", "A spiral stair winds down past niches stuffed with yellowed skulls. Each step echoes a little too long."},
		},
		Rooms: []roomDef{
			{"Hall of Effigies", "Stone knights lie atop their tombs in two silent rows. One effigy's sword is missing — and the dust around it is disturbed."},
			{"Embalming Chamber", "Rusted hooks and cracked jars line the walls. A sweet, rotten smell hangs in the air, and something drips steadily in the dark."},
			{"Flooded Catacomb", "Black water rises to your knees. Bones drift past your legs, and the ceiling arches low over carved names worn smooth by time."},
			{"Chapel of Ash", "A ruined altar stands beneath a shattered rose window of stained glass. Candles — freshly lit — burn on the altar."},
			{"Bone Gallery", "Walls built entirely of stacked skulls and femurs curve into the gloom. Here and there, an empty socket seems to follow you."},
			{"Collapsed Ossuary", "Half the ceiling has fallen in. Roots push through the rubble, and coffins lie split open as if burst from inside."},
		},
		Lairs: []roomDef{
			{"The Black Sepulchre", "A vast vaulted tomb lit by sickly green flame. In the center, a great sarcophagus stands open — and the thing that rose from it turns to face you."},
		},
		Sounds: []string{"the scrape of bone on stone", "a low, wordless chanting", "slow, dragging footsteps", "the clatter of something falling apart and pulling itself back together"},
		Intro:  "Beneath the old graveyard, the dead have stopped resting.",
	},
	"cave": {
		Titles: []string{"The Caverns of %s", "The %s Delve", "The Deep Warrens of %s", "The Hollow Under %s"},
		Places: []string{"Redfang Hill", "Gnasher's Gorge", "Mount Kael", "Blackroot", "the Weeping Cliffs"},
		Hooks: []string{
			"Raiders from the hills have been stealing livestock and kidnapping miners. Tracks lead to a cave mouth in the cliffs above town.",
			"The silver mine went quiet three days ago. The foreman's last message read only: 'Something is digging up toward us.'",
			"A merchant caravan was dragged into the gorge. The survivors speak of a chieftain who wears a crown of teeth.",
		},
		Lore: []string{
			"The tribes of the deep caves worship whatever is strongest. Kill their leader and the rest may scatter.",
			"These tunnels were dug by dwarves long ago. Dwarven stonework marks the safe paths — and the trapped ones.",
		},
		Entry: []roomDef{
			{"Cave Mouth", "A jagged opening in the cliff exhales cold air. Bones of sheep and cattle litter the entrance, gnawed clean."},
			{"Mine Entrance", "Broken mine carts and snapped support beams frame a tunnel leading into the mountain. A lantern lies smashed on the tracks."},
		},
		Rooms: []roomDef{
			{"Dripping Gallery", "Stalactites hang like teeth from a ceiling lost in shadow. Water drips into pools that glow faintly blue."},
			{"Fungus Grotto", "Giant mushrooms tower overhead, releasing drifting spores that glitter in your torchlight."},
			{"Chasm Bridge", "A narrow rope bridge sways over a chasm whose bottom you cannot see. The ropes look recently repaired — badly."},
			{"Raider Camp", "Smoldering cookfires, filthy bedrolls, and piles of stolen goods fill this wide chamber."},
			{"Echoing Crossroads", "Five tunnels meet in a round cave. Crude arrows scratched into the walls point in different directions."},
			{"Underground River", "A fast black river cuts through the cavern. A few slick stepping stones cross it."},
		},
		Lairs: []roomDef{
			{"The Throne of Bones", "A huge cavern lit by a roaring bonfire. On a throne built of skulls and stolen treasure sits the master of these caves, who rises with a roar."},
		},
		Sounds: []string{"guttural voices arguing", "the drip of water and a low growl", "claws skittering on stone", "drums, slow and heavy"},
		Intro:  "In the hills above town, something has made a home in the dark.",
	},
	"forest": {
		Titles: []string{"The Shadows of %s", "The %s Wildwood", "Hunt in %s", "The Thorns of %s"},
		Places: []string{"Wyrmwood", "the Greyfell Forest", "Mossmere", "the Elderglade", "Thornwick Vale"},
		Hooks: []string{
			"Woodcutters have stopped returning from the deep forest. Their axes were found hanging from the branches of a dead oak.",
			"Bandits have closed the King's Road, and the town is running short on food. Their hideout lies somewhere in the wood.",
			"A beast the size of a cart has been seen stalking the forest edge. Hunters who went after it came back in pieces — or not at all.",
		},
		Lore: []string{
			"The druids say the forest's heart was poisoned when the old shrine was defiled. What lairs there now is something that feeds on that corruption.",
			"The ranger lodge records describe the beast's lair: a ruined shrine in a clearing where no birds sing.",
		},
		Entry: []roomDef{
			{"Forest Edge", "The road narrows to a trail as ancient trees close in overhead. Birdsong fades the deeper you go."},
			{"Hunter's Path", "A deer trail winds through ferns taller than a man. Fresh claw marks score the bark of a nearby tree."},
		},
		Rooms: []roomDef{
			{"Misty Hollow", "Fog pools in this sunken glade. Shapes move between the trees — maybe deer, maybe not."},
			{"Abandoned Woodcamp", "Tents slashed open, an overturned cookpot, and a trail of something dragged into the brush."},
			{"Spider Thicket", "Thick webs stretch between the trees like sails, heavy with wrapped bundles the size of people."},
			{"Old Stone Circle", "Moss-covered standing stones ring a clearing. The air hums faintly, and the grass inside the circle is dead."},
			{"Rushing Ford", "A cold river cuts across the path. The remains of a wooden bridge jut from the water."},
			{"Hanging Oak", "A colossal dead oak with branches strung with bones and wind chimes made of teeth."},
		},
		Lairs: []roomDef{
			{"The Ruined Shrine", "A collapsed temple overgrown with black thorns. Something huge stirs in the shadows of the broken altar — it has been waiting for you."},
		},
		Sounds: []string{"branches snapping under heavy feet", "a howl answered by another", "rough laughter and the clink of coins", "a deep, rumbling breath"},
		Intro:  "The forest has always been dangerous. Lately, it has become hungry.",
	},
	"keep": {
		Titles: []string{"The Ruins of %s Keep", "The Fall of %s", "Siege of %s", "The Burning Tower of %s"},
		Places: []string{"Ravenmoor", "Caer Dunholt", "Highwatch", "Morrow's End", "Ironspire"},
		Hooks: []string{
			"The abandoned keep on the hill has lights in its windows again. A cult has taken it, and villagers have started disappearing.",
			"A warband has seized the old border fortress and is gathering strength. The lord offers a fat reward to anyone who can break them.",
			"Someone in the ruined keep is summoning something. Each night the sky above the tower glows a little redder.",
		},
		Lore: []string{
			"The keep's last lord made a pact with a dark power. His heir means to finish the ritual he started.",
			"The fortress has a forgotten postern gate on its north side — the old guard captain's journals mention it.",
		},
		Entry: []roomDef{
			{"Broken Gatehouse", "The portcullis is jammed half-open. Arrow slits stare down at you from the walls above, dark and silent."},
			{"Overgrown Courtyard", "Weeds choke the flagstones of the courtyard. A dry fountain sits in the middle, its statue beheaded."},
		},
		Rooms: []roomDef{
			{"Great Hall", "Long tables lie overturned beneath tattered banners. A feast was abandoned here mid-meal, the food long rotten."},
			{"Barracks", "Rows of rusted bunks and weapon racks. Some of the weapons are newly oiled."},
			{"Armory", "Empty weapon stands and smashed crates — someone has been arming an army."},
			{"Library", "Shelves of rotting books. A reading desk holds a journal written in a shaky hand, its last page torn out."},
			{"Dungeon Cells", "Iron cells line a damp corridor. Scratches on the walls count days — many days."},
			{"Collapsed Tower Stair", "A spiral stair climbs through a tower whose upper half has fallen. Wind howls through the gaps."},
		},
		Lairs: []roomDef{
			{"The Ritual Chamber", "At the top of the keep, a summoning circle burns with crimson fire. The master of this place turns from the ritual, furious at the interruption."},
		},
		Sounds: []string{"marching boots and barked orders", "chanting in an ugly tongue", "the clang of a hammer on an anvil", "a scream, cut short"},
		Intro:  "The old keep was supposed to be empty. It isn't.",
	},
}

var ThemeOrder = []string{"cave", "forest", "crypt", "keep"}

var Towns = []string{"Millbrook", "Oakhaven", "Briarwood", "Stonebridge", "Duskhollow", "Larkspur"}

type npcDef struct{ Name, Desc, Gift, Hint string }

var NPCs = []npcDef{
	{"a wounded adventurer", "Slumped against the wall sits a wounded adventurer clutching a bloodied side.", "potion_healing", "They gasp a warning: \"The leader... strikes hardest at whoever stands closest. Don't bunch up.\""},
	{"a trapped merchant", "A portly merchant is tied to a post, gagged and wide-eyed with hope at the sight of you.", "gem", "Untied, he babbles: \"They keep their treasure in the deepest chamber, in a chest by the leader's seat!\""},
	{"a ghostly knight", "A translucent knight in old armor stands at attention, watching you with sorrowful eyes.", "potion_healing", "The ghost's voice is like wind: \"I fell to the master of this place. Avenge me, and I may rest.\""},
	{"a frightened kobold", "A small kobold cowers behind a barrel, holding a stick like a spear.", "potion_healing", "The kobold squeaks: \"Boss is mean! Kobold show you safe way, yes? Just no stabbing!\""},
	{"a lost child", "A filthy, terrified child hides in a corner, eyes enormous in the torchlight.", "silver_idol", "The child whispers: \"The big one sleeps after it eats. I saw it.\" They press a small silver idol into your hand."},
	{"a mad hermit", "A wild-haired hermit sits cross-legged, humming and arranging pebbles in patterns.", "potion_greater", "The hermit cackles: \"Fire and light! Fire and light! The dark hates fire and light!\""},
}

type trapDef struct {
	Name, Save, Trigger, Found string
	DC                         int
	Damage                     string
}

var Traps = []trapDef{
	{"pressure plate", "DEX", "A flagstone clicks underfoot — darts hiss from the walls!", "You notice a slightly raised flagstone with tiny holes in the walls on either side: a dart trap.", 12, "2d6"},
	{"pit trap", "DEX", "The floor gives way into a spiked pit!", "The floor ahead sounds hollow. A concealed pit trap!", 12, "2d6"},
	{"swinging blade", "DEX", "A rusted blade swings out of the ceiling!", "A thin slot runs across the ceiling — a swinging blade trap waits there.", 13, "2d8"},
	{"poison gas vent", "CON", "Green gas billows from cracks in the floor!", "You smell something acrid and spot discolored vents in the floor: a gas trap.", 13, "3d6"},
	{"tripwire and rockfall", "DEX", "A tripwire snaps and rocks rain from above!", "A nearly invisible tripwire stretches across the path, tied to a pile of loose rocks above.", 12, "2d10"},
}

var LootTiers = [][]string{
	{"potion_healing", "gem", "dagger_p1", "potion_healing", "silver_idol"},
	{"potion_greater", "longsword_p1", "longbow_p1", "half_plate", "gem", "potion_healing"},
	{"potion_greater", "plate", "longsword_p1", "longbow_p1", "potion_greater"},
}

// EncounterBudget is the XP budget for a medium fight per character, by level.
var EncounterBudget = []int{50, 100, 150, 250, 500, 600, 750, 900, 1100, 1200, 1600, 2000}

var successLines = map[string][]string{
	"athletics":       {"With a grunt of effort, you manage it.", "Muscles straining, you succeed."},
	"acrobatics":      {"You move with nimble grace and pull it off.", "Light on your feet, you make it look easy."},
	"stealth":         {"You melt into the shadows without a sound.", "Not a whisper betrays you."},
	"sleight_of_hand": {"Your fingers work quickly and precisely. Done.", "A deft twist, a soft click — success."},
	"perception":      {"Your senses sharpen and you pick out a detail others would miss.", "You notice something important."},
	"investigation":   {"Careful examination pays off.", "Your methodical search turns something up."},
	"survival":        {"You read the signs of the wild like a book.", "The trail is clear to your trained eye."},
	"medicine":        {"Your careful hands do good work.", "You know exactly what's needed."},
	"religion":        {"Sacred knowledge comes back to you.", "The rites you learned serve you well."},
	"arcana":          {"Arcane theory clicks into place in your mind.", "You recognize the pattern of the magic."},
	"history":         {"You recall an old story that fits.", "History offers its lessons."},
	"persuasion":      {"Your words land well.", "They seem convinced."},
	"deception":       {"They buy it completely.", "Your lie is flawless."},
	"intimidation":    {"They flinch back from you.", "Fear flickers in their eyes."},
	"performance":     {"Your performance is captivating.", "You hold everyone spellbound."},
	"insight":         {"You read the truth behind the mask.", "You sense what's really going on."},
	"nature":          {"You know this land and its creatures.", "Nature's ways are clear to you."},
	"animal_handling": {"The beast calms under your touch.", "You earn the animal's trust."},
}

var failLines = []string{
	"It doesn't go as planned.", "Not this time.", "You fall short.", "Something goes wrong.", "Despite your efforts, nothing comes of it.",
}
