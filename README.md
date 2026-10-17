# 👁️ Jujutsu Chronicles (D&D Campaign Runner)

A self-running, multiplayer **Jujutsu Kaisen RPG** for Windows. One person runs the `.exe`; everyone joins from a
browser. You design a pixel-art sorcerer, walk a procedurally generated world with **WASD**, exorcise curses in
animated turn-based battles, and play a branching story — run by an AI Dungeon Master (or an offline one).

## What's in it
- **Visual world:** a generated island with towns, the city of Shibuya, Tokyo Jujutsu High, roads, bridges, forests
  and 14 cursed sites. Day/night cycle, minimap, speech bubbles, other players moving in real time.
- **Pixel character creator:** skin, 12 hair styles, colors, 8 outfits, accessories (blindfold, glasses, collar mask...).
- **15 cursed techniques** (homebrew, anime-inspired): Limitless, Ten Shadows, Cursed Speech, Straw Doll, Blood
  Manipulation, Shrine, Disaster Flames, Idle Transfiguration, Ratio, Boogie Woogie, Projection Sorcery, Private Pure
  Love Train, Cursed Partner, Heavenly Restriction, Construction, Mythical Beast Amber — 4 moves each plus a
  **Domain Expansion**. Everyone gets Black Flash, Binding Vows, Simple Domain and Reverse Cursed Technique.
- **6 clans** (Gojo, Zenin, Kamo, Inumaki, Half-Curse, Non-Clan) and **26 curses** from Grade 4 to Special Grade
  (Jogo, Hanami, Dagon, Mahito, Kenjaku, Sukuna...). Curses roam the map; touch one and a battle starts.
- **Battles:** turn order, cursed energy, statuses, domain clashes, 19 animated domain arenas, Black Flash.
- **Story:** a five-act main story that branches on your choices (Sukuna's finger, Junpei, Jogo vs Hanami, the Kyoto
  event, Shibuya) with different endings, plus endless side jobs from town mayors, shops, inns, Shoko's healing and
  Ijichi's rides between towns.
- **Anything goes (AI DM):** with an Anthropic API key, type anything — the AI DM can create NPCs, quests, titles,
  items and party choices in response. The offline DM handles common actions without internet.

Controls: **WASD** move (Shift to run) · **E** talk · **Enter** type an action · **C** character · **Q** quests ·
**M** map · **1-9** pick options / techniques.

## Quick start

1. Download **`release/DnD-Campaign-Runner.exe`** from this repo. On an ARM Windows laptop, download the
   `-arm64.exe` build from the Actions artifacts instead.
2. Put it in its own folder (e.g. `Documents\DnD`) and double-click it.
   - Windows SmartScreen may say "Windows protected your PC" because the app isn't code-signed. Click
     **More info → Run anyway**.
   - When Windows Firewall asks, tick **Private networks** and click **Allow**. Otherwise friends can't connect.
3. Your browser opens the game. The black console window shows the address friends should open, for example
   `http://192.168.1.23:7777`. **Keep that window open while you play**. Closing it stops the game (it saves
   automatically).
4. Everyone enters their name and creates a character.
5. You (the host) click **⚙ Host → Begin adventure**.

### Turning on the AI Dungeon Master
**⚙ Host → Dungeon Master**: paste an API key from [console.anthropic.com](https://console.anthropic.com)
(API Keys), choose **AI Dungeon Master**, and click **Save**. You can also add setting and tone notes ("gothic
horror in a cursed swamp, keep it PG-13"). The key is stored only in `dnd-runner-config.json` next to the exe.

Model choices: **Opus 5.5** (default, best storytelling), **Sonnet 5.5** (faster and cheaper), **Haiku 5.5**
(fastest and cheapest). A lower thinking effort gives faster replies.

### Playing with friends who aren't on your Wi-Fi
The game is a small web server on your PC. For friends elsewhere, either:
- **Tailscale (easiest and safest):** everyone installs [Tailscale](https://tailscale.com) and joins your
  tailnet. Friends then open `http://<your-tailscale-ip>:7777`.
- **Port forwarding:** forward TCP port 7777 on your router to your PC and share your public IP. This exposes
  the game to the internet, so only do it while playing.

## How to play

- **🎭 Action:** describe what your character does ("I check the altar for hidden compartments", "I try to
  convince the guard we're merchants"). The DM resolves it, calling for dice rolls when needed. If several
  players act at the same moment, the DM answers all of them together.
- **💬 Table talk:** out-of-character chat. The DM ignores it.
- **Dice buttons:** public rolls anyone can see.
- **Combat:** when a fight starts, an initiative tracker appears. On your turn, click an enemy to target it, then
  **Attack**, cast a spell or use an ability, drink a potion, **Dodge**, **Flee**, or **End turn**. Monsters act
  automatically. If a player is away, the host can **Skip** their turn.
- **Your sheet** (right side): HP, AC, stats, skills, abilities, inventory (equip, drink, or give items to
  party members), and XP toward the next level.
- **Death:** at 0 HP you're down and roll death saves on your turns. Healing brings you back. Three failures
  kill a hero; click ⟲ on your sheet to retire them and roll a new one (they join near the party's level).
  If the whole party falls, the story continues: you wake up somewhere, with consequences.

### Offline DM phrases
It understands plain English, including: *set out / move on / go deeper*, *go back / return to town*,
*look around*, *search the room*, *listen*, *sneak ahead*, *open the chest*, *pick the lock*, *force it open*,
*disarm the trap*, *jump over the trap*, *talk to the stranger*, *recall lore / pray*, *stabilize*,
*buy potion*, *buy greater potion*, *buy longsword*, *sell gem*, *short rest*, *long rest*. Other actions
("climb the wall", "track the beast", "lie to the guard") become ability checks.

## Rules (5e-lite)
- 7 races (Human, Elf, Dwarf, Halfling, Half-Orc, Tiefling, Dragonborn) and 8 classes (Fighter, Barbarian,
  Rogue, Ranger, Wizard, Cleric, Bard, Paladin), each with signature abilities: Second Wind, Action Surge, Rage,
  Sneak Attack, Fire Bolt, Magic Missile, Sacred Flame, Cure Wounds, Healing Word, Vicious Mockery, Hunter's
  Mark, Lay on Hands, Divine Smite.
- Ability scores: standard array or 4d6-drop-lowest, rolled server-side so nobody can cheat.
- d20 checks with skill proficiency, advantage and disadvantage, saving throws, criticals, death saves, and
  short and long rests. Leveling runs to level 12, with Extra Attack at 5, cantrip scaling, and ability score
  increases.
- 23 monsters from Giant Rats to a Young Green Dragon, with elite "boss" variants.

## Saves
Campaigns autosave every few seconds to `saves\` next to the exe. If that folder isn't writable, they go to
`%AppData%\DnD-Campaign-Runner`. **⚙ Host → Campaigns** lets you start new campaigns and load or delete old
ones. Players keep their seat when you switch.

Command-line options: `-port 7778` (use a different port), `-data D:\DnD` (choose the saves folder),
`-no-browser`.

## Building from source
Requires Go 1.24+.

```bash
go test ./...                 # unit tests and simulated bot campaigns
go run .                      # run locally on http://localhost:7777
tools/build.sh 0.1.0          # Windows .exe (x64 + ARM) with icon -> dist/
python3 tools/make_icon.py    # regenerate the icon (needs Pillow)
```

Layout:
- `main.go`, `server.go`: the exe. It runs an HTTP server with live updates (Server-Sent Events) and serves the
  embedded web UI.
- `engine/`: rules, characters, combat, campaigns, persistence (`engine.go`, `character.go`, `combat.go`), the
  offline DM (`offline.go`, `content.go`), and the Claude DM (`claude.go`, which uses tools for rolls, combat,
  loot, XP and campaign memory).
- `web/`: the browser UI (plain HTML/CSS/JS, no external downloads, so it works fully offline).


## Play online for free

Open the host panel (⚙ Host) and click **🌍 Go online**. The app opens a free tunnel (Cloudflare quick tunnel, or Windows' built-in `ssh` to localhost.run, no account needed; it may download a small helper the first time) and shows a link plus a room code. Send the link to friends; they need the code to join. Only you (the person at the host computer) get host controls. Click **Stop sharing** to close it. Set `DND_DEBUG=1` only for development test hooks.


## Dynamic quests, fabricated skills and invented curses

Quests aren't limited to a list. Ask anyone "any odd jobs?" (or type "can I be your assistant?" to the offline DM) and the game builds a quest from the real NPCs, shops and sites nearby: errands, deliveries, waiting out time, checking on people, exorcising a curse made up for the job. The tracker shows every step, how far away the next one is, and an arrow at the screen edge points to it. Finishing a job can earn a title and a **fabricated skill** (for example the nurse's assistant's *Bedside Manner*, +2 healing).

With an AI DM (Claude or a local Ollama model) the same system is open-ended: the DM can call `create_quest` with its own steps for any scheme a player proposes, `add_quest_step` / `update_quest` as the story changes, `create_skill` for odd perks (passives are small and capped, active moves scale with level), and `create_curse` to invent a new curse with balanced stats, moves and a sprite. The engine owns every number, so the AI can be as creative as it likes with names and fiction without breaking the game.
