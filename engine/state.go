package engine

import "time"

type Player struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Token       string    `json:"token"`
	CharID      string    `json:"charId"`
	Online      bool      `json:"online"`
	LastSeen    time.Time `json:"lastSeen"`
	RolledStats []int     `json:"rolledStats,omitempty"`
}

// Appearance drives the procedural pixel sprite.
type Appearance struct {
	Skin      int    `json:"skin"`
	Hair      int    `json:"hair"`
	HairColor string `json:"hairColor"`
	Eyes      string `json:"eyes"`
	Outfit    int    `json:"outfit"`
	Color1    string `json:"color1"`
	Color2    string `json:"color2"`
	Accessory int    `json:"accessory"`
	Build     int    `json:"build"`
}

type Character struct {
	ID        string         `json:"id"`
	Owner     string         `json:"owner"`
	Name      string         `json:"name"`
	Race      string         `json:"race"`  // origin id
	Class     string         `json:"class"` // technique id
	Level     int            `json:"level"`
	XP        int            `json:"xp"`
	Stats     map[string]int `json:"stats"`
	MaxHP     int            `json:"maxHp"`
	HP        int            `json:"hp"`
	AC        int            `json:"ac"`
	Gold      int            `json:"gold"` // yen
	Inventory []Item         `json:"inventory"`
	CE        int            `json:"ce"`
	CEMax     int            `json:"ceMax"`
	Skills    []string       `json:"skills"`
	Backstory string         `json:"backstory"`
	Look      Appearance     `json:"look"`
	Title     string         `json:"title,omitempty"`  // e.g. "Town Guard of Akegawa" (set by the DM)
	Notes     []string       `json:"notes,omitempty"`  // DM-tracked facts about this hero
	Vessel    bool           `json:"vessel,omitempty"` // swallowed Sukuna's finger

	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Dir    int     `json:"dir"` // 0 down, 1 left, 2 right, 3 up, 4 SW, 5 SE, 6 NW, 7 NE
	Moving bool    `json:"moving"`

	Status     map[string]int `json:"status"`
	Used       map[string]int `json:"used"` // per-battle ability uses
	DeathSaves int            `json:"deathSaves"`
	DeathFails int            `json:"deathFails"`
	Stable     bool           `json:"stable"`
	Dead       bool           `json:"dead"`
	Fled       bool           `json:"fled"`
	InBattle   bool           `json:"inBattle"`
}

type Monster struct {
	ID         string         `json:"id"`
	Template   string         `json:"template"`
	Name       string         `json:"name"`
	HP         int            `json:"hp"`
	MaxHP      int            `json:"maxHp"`
	AC         int            `json:"ac"`
	ToHit      int            `json:"toHit"`
	Init       int            `json:"init"`
	Save       int            `json:"save"`
	Rank       int            `json:"rank"`
	XP         int            `json:"xp"`
	Yen        string         `json:"yen"`
	Boss       bool           `json:"boss"`
	Domain     string         `json:"domain,omitempty"`
	Escapes    bool           `json:"escapes,omitempty"`
	Status     map[string]int `json:"status"`
	Sprite     SpriteSpec     `json:"sprite"`
	DomainUsed bool           `json:"domainUsed"`
	Power      int            `json:"power,omitempty"`
}

type Combatant struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsMonster bool   `json:"isMonster"`
	Init      int    `json:"init"`
}

// FX is one visual beat of a battle for the client to animate.
type FX struct {
	ID    int     `json:"id"`
	Kind  string  `json:"kind"` // move, domain, domain_end, blackflash, status, death, turn, text
	Src   string  `json:"src"`
	Name  string  `json:"name"`
	VFX   string  `json:"vfx"`
	Color string  `json:"color"`
	Hits  []FXHit `json:"hits"`
	Text  string  `json:"text,omitempty"`
}

type FXHit struct {
	ID     string `json:"id"`
	Dmg    int    `json:"dmg,omitempty"`
	Heal   int    `json:"heal,omitempty"`
	Miss   bool   `json:"miss,omitempty"`
	Crit   bool   `json:"crit,omitempty"`
	BF     bool   `json:"bf,omitempty"`
	Status string `json:"status,omitempty"`
	Down   bool   `json:"down,omitempty"`
	Saved  bool   `json:"saved,omitempty"`
}

type DomainState struct {
	Owner   string `json:"owner"`
	Ability string `json:"ability"`
	Rounds  int    `json:"rounds"`
	Party   bool   `json:"party"` // cast by the party side
	Name    string `json:"name"`
	VFX     string `json:"vfx"`
	Color   string `json:"color"`
}

type Combat struct {
	Round       int          `json:"round"`
	Turn        int          `json:"turn"`
	Order       []Combatant  `json:"order"`
	Monsters    []*Monster   `json:"monsters"`
	ActionUsed  bool         `json:"actionUsed"`
	BonusUsed   bool         `json:"bonusUsed"`
	ExtraAction bool         `json:"extraAction"`
	Domain      *DomainState `json:"domain"`
	FX          []FX         `json:"fx"`
	FXSeq       int          `json:"fxSeq"`
	Biome       string       `json:"biome"`
	X           float64      `json:"x"`
	Y           float64      `json:"y"`
	RoamerID    string       `json:"roamerId,omitempty"`
	QuestID     string       `json:"questId,omitempty"`
	Title       string       `json:"title"`
}

type LogEntry struct {
	ID   int       `json:"id"`
	Kind string    `json:"kind"` // dm, action, ooc, roll, combat, system, level, say
	Who  string    `json:"who,omitempty"`
	Text string    `json:"text"`
	Time time.Time `json:"time"`
}

// Roamer is a curse wandering the overworld.
type Roamer struct {
	ID     string   `json:"id"`
	Kind   string   `json:"kind"`
	Group  []string `json:"group"`
	X      float64  `json:"x"`
	Y      float64  `json:"y"`
	HomeX  float64  `json:"homeX"`
	HomeY  float64  `json:"homeY"`
	Site   string   `json:"site,omitempty"`
	Quest  string   `json:"quest,omitempty"`
	Elite  bool     `json:"elite,omitempty"`
	Name   string   `json:"name,omitempty"`
	tx, ty float64
	cool   int
}

type Quest struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Desc       string  `json:"desc"`
	Kind       string  `json:"kind"` // main, side, ai
	Giver      string  `json:"giver"`
	GiverName  string  `json:"giverName"`
	Site       string  `json:"site,omitempty"`
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Stage      string  `json:"stage"` // active, return, done, failed
	Encounter  []Spawn `json:"encounter,omitempty"`
	RewardXP   int     `json:"rewardXp"`
	RewardYen  int     `json:"rewardYen"`
	RewardItem string  `json:"rewardItem,omitempty"`
	Choice     string  `json:"choice,omitempty"` // choice event fired after victory
	Act        int     `json:"act,omitempty"`
	MinLevel   int     `json:"minLevel"`
	Objective  string  `json:"objective"`
}

// Choice is a decision put to the whole party.
type Choice struct {
	ID      string         `json:"id"`
	Title   string         `json:"title"`
	Speaker string         `json:"speaker"`
	Text    string         `json:"text"`
	Options []ChoiceOption `json:"options"`
	Source  string         `json:"source"`        // story event id, or "ai"
	For     string         `json:"for,omitempty"` // character id the choice is about
}

type ChoiceOption struct {
	ID      string   `json:"id"`
	Text    string   `json:"text"`
	Effects []string `json:"effects,omitempty"`
	Result  string   `json:"result,omitempty"`
	Need    string   `json:"need,omitempty"` // shown requirement, e.g. "CHA check"
}

// NPCState is mutable per-campaign NPC data on top of the generated world.
type NPCState struct {
	Gone   bool     `json:"gone,omitempty"`
	Mood   int      `json:"mood,omitempty"`
	Notes  []string `json:"notes,omitempty"`
	Talked int      `json:"talked,omitempty"`
}

type Campaign struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Setting    string                `json:"setting"`
	Mode       string                `json:"mode"` // "ai" or "offline"
	Created    time.Time             `json:"created"`
	Started    bool                  `json:"started"`
	Seed       int64                 `json:"seed"`
	Players    map[string]*Player    `json:"players"`
	Characters map[string]*Character `json:"characters"`
	Log        []LogEntry            `json:"log"`
	NextLogID  int                   `json:"nextLogId"`
	NextID     int                   `json:"nextId"`
	Combat     *Combat               `json:"combat"`
	Location   string                `json:"location"`
	SceneDesc  string                `json:"sceneDesc"`
	Quest      string                `json:"quest"`
	Journal    []string              `json:"journal"`
	Quests     []*Quest              `json:"quests"`
	Flags      map[string]string     `json:"flags"`
	Rep        map[string]int        `json:"rep"`
	Act        int                   `json:"act"`
	Choice     *Choice               `json:"choice"`
	Cleared    map[string]bool       `json:"cleared"`
	Discovered map[string]bool       `json:"discovered"`
	NPCs       map[string]*NPCState  `json:"npcs"`
	ExtraNPCs  []*NPC                `json:"extraNpcs"`
	Day        int                   `json:"day"`
	Clock      float64               `json:"clock"` // 0..24 hours
}
