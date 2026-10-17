package engine

import "time"

type Player struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Token    string    `json:"token"`
	CharID   string    `json:"charId"`
	Online   bool      `json:"online"`
	LastSeen time.Time `json:"lastSeen"`
	// RolledStats holds a 4d6-drop-lowest array the player rolled during character creation.
	RolledStats []int `json:"rolledStats,omitempty"`
}

type Character struct {
	ID        string         `json:"id"`
	Owner     string         `json:"owner"`
	Name      string         `json:"name"`
	Race      string         `json:"race"`
	Class     string         `json:"class"`
	Level     int            `json:"level"`
	XP        int            `json:"xp"`
	Stats     map[string]int `json:"stats"`
	MaxHP     int            `json:"maxHp"`
	HP        int            `json:"hp"`
	TempHP    int            `json:"tempHp"`
	AC        int            `json:"ac"`
	Gold      int            `json:"gold"`
	Inventory []Item         `json:"inventory"`
	Slots     int            `json:"slots"`
	SlotsMax  int            `json:"slotsMax"`
	Uses      map[string]int `json:"uses"`
	Features  []string       `json:"features"`
	Skills    []string       `json:"skills"`
	Backstory string         `json:"backstory"`

	DeathSaves int    `json:"deathSaves"`
	DeathFails int    `json:"deathFails"`
	Stable     bool   `json:"stable"`
	Dead       bool   `json:"dead"`
	Raging     bool   `json:"raging"`
	Dodging    bool   `json:"dodging"`
	Smiting    bool   `json:"smiting"`
	MarkTarget string `json:"markTarget,omitempty"`
	Relentless bool   `json:"relentlessUsed"`
	Fled       bool   `json:"fled"`
}

type Monster struct {
	ID       string `json:"id"`
	Template string `json:"template"`
	Name     string `json:"name"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"maxHp"`
	AC       int    `json:"ac"`
	ToHit    int    `json:"toHit"`
	DexMod   int    `json:"dexMod"`
	Damage   string `json:"damage"`
	DmgType  string `json:"dmgType"`
	Attacks  int    `json:"attacks"`
	Attack   string `json:"attack"`
	Saves    int    `json:"saves"`
	XP       int    `json:"xp"`
	Gold     string `json:"gold"`
	Boss     bool   `json:"boss"`
	Mocked   bool   `json:"mocked"`
}

type Combatant struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsMonster bool   `json:"isMonster"`
	Init      int    `json:"init"`
}

type Combat struct {
	Round       int         `json:"round"`
	Turn        int         `json:"turn"`
	Order       []Combatant `json:"order"`
	Monsters    []*Monster  `json:"monsters"`
	ActionUsed  bool        `json:"actionUsed"`
	BonusUsed   bool        `json:"bonusUsed"`
	ExtraAction bool        `json:"extraAction"`
	SneakUsed   bool        `json:"sneakUsed"`
	Defeated    []*Monster  `json:"defeated"`
}

type LogEntry struct {
	ID   int       `json:"id"`
	Kind string    `json:"kind"` // dm, action, ooc, roll, combat, system, level
	Who  string    `json:"who,omitempty"`
	Text string    `json:"text"`
	Time time.Time `json:"time"`
}

type Room struct {
	Name      string   `json:"name"`
	Desc      string   `json:"desc"`
	Encounter []string `json:"encounter,omitempty"`
	Elite     bool     `json:"elite,omitempty"`
	Cleared   bool     `json:"cleared"`
	Searched  bool     `json:"searched"`
	Trap      string   `json:"trap,omitempty"`
	TrapFound bool     `json:"trapFound"`
	TrapDone  bool     `json:"trapDone"`
	Loot      []string `json:"loot,omitempty"`
	Gold      int      `json:"gold,omitempty"`
	Chest     bool     `json:"chest,omitempty"`
	ChestOpen bool     `json:"chestOpen"`
	Locked    bool     `json:"locked,omitempty"`
	NPC       string   `json:"npc,omitempty"`
	NPCTalked bool     `json:"npcTalked"`
	Hint      string   `json:"hint,omitempty"`
	Town      bool     `json:"town,omitempty"`
	Visited   bool     `json:"visited"`
}

// Adventure is the procedurally generated dungeon the offline DM runs.
type Adventure struct {
	Theme    string `json:"theme"`
	Title    string `json:"title"`
	Hook     string `json:"hook"`
	Boss     string `json:"boss"`
	Lore     string `json:"lore"`
	Rooms    []Room `json:"rooms"`
	Index    int    `json:"index"`
	Chapter  int    `json:"chapter"`
	Complete bool   `json:"complete"`
	Sneaking bool   `json:"sneaking"`
}

type Campaign struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Setting    string                `json:"setting"`
	Mode       string                `json:"mode"` // "ai" or "offline"
	Created    time.Time             `json:"created"`
	Started    bool                  `json:"started"`
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
	Adventure  *Adventure            `json:"adventure"`
}
