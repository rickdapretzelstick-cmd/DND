# Handoff: visual overhaul of Jujutsu Chronicles

Repo `rickdapretzelstick-cmd/DND`, branch `claude/busy-ride-lcumgy`. Commit and push there. Never force-push.

## What the app is
- A self-running multiplayer JJK (anime) RPG, shipped as a Windows exe.
- The Go server (`main.go`, `server.go`, `engine/`) embeds a canvas web client (`web/`) via `//go:embed web`.
- The web client is only served from the embed, so rebuild the binary after every web change.
- Build and run locally:
  `go build -o /tmp/r/dnd-runner . && /tmp/r/dnd-runner -port 7790 -no-browser -data /tmp/r/data`
- Tests: `go test ./engine/`. There is a rare (about 1 in 15) flake in the story playthrough test.
- Windows release: `tools/build.sh <version>` produces `dist/`. Copy the amd64 exe to `release/DnD-Campaign-Runner.exe`. Current version is v0.3.0.
- Do not confuse this repo with `/mod` (a Minecraft mod). It is unrelated.

## The user's request
> "we need an insane Visual overhaul use references from similar concepts and games to not have a chud output anything UI buildings, floors water people, we need a big upgrade"

- References: WorldBox (soft colour-varied terrain, individual tree sprites), Stardew Valley / Eastward / Sea of Stars (detail, shading, lighting), Persona 5 / P3R menus, and the JJK anime look.
- Before this work, the world looked like flat 16px "old Nintendo" squares at 4x zoom.
- The user prefers progress updates and screenshots over long verification loops.

## Status

### Done (committed)
`web/js/terrain.js` is a per-pixel chunk renderer. It is NOT wired in yet.
- `renderChunk(at, cx, cy)` returns `{ground, foam, F, N, tx0, ty0}` for a 32×32-tile chunk at 16px per tile.
- Natural terrain borders are domain-warped, so they wander instead of following the grid.
- Coastlines come from a signed distance field, giving water depth, shallows caustics and a separate foam canvas for animation.
- Each material has its own colour ramp: grass with dry patches, hills and snow with shading, crop rows, cursed ground with glowing veins.
- Roads, sidewalks, plaza, bridge and wall tiles are detailed.
- Decals cover grass tufts, flowers, pebbles, shells and bones.
- `fieldAtTile(chunk, tx, ty)` returns coast distance, intended for prop placement.
- It has not been run yet. Expect bugs, and measure chunk render time; it is roughly 4M hash calls per chunk. If a frame stalls, build at most 1–2 chunks per frame.

### Uncommitted at handoff
`web/index.html` and `web/style.css` may hold UI changes from a helper agent in the previous session. Those agents do not survive a session change. Check `git status` and `git diff`: keep the changes if they look good and work, otherwise `git checkout` them.

### Not done
1. **Wire `terrain.js` into `web/js/world.js`.**
   - `WorldView.chunk()` should call `renderChunk`, using `this.at` as the tile accessor.
   - Draw the `foam` canvas with alpha that pulses over time.
   - Keep the LRU chunk cache.
   - Lower the zoom in `main.js` (around line 736) to about `round(H/(16*18))`, clamped to 2..4. The current value is too zoomed in.
2. **Props (new `web/js/props.js`).**
   - Deterministic per tile from `hash(tx,ty,…)`: one jittered tree per FOREST tile, varied by noise. Tree types are round deciduous, pine near hills/peaks, sakura groves and occasional autumn trees.
   - Bushes and berry bushes on grass; boulders on ROCK; dead trees and purple crystals on CURSED; reeds and lily pads in the shallows.
   - In towns and the city: street lamps along PAVED edges, vending machines on sidewalks, stone lanterns and benches on PLAZA.
   - Generate sprites procedurally into cached canvases: shaded canopy blobs lit from the top-left, plus a dark outline.
   - Bake shadows into the ground chunk. Draw the props themselves y-sorted together with entities in `WorldView.draw`. Collect props per chunk once.
   - Lamps add night lights.
3. **Buildings (new `web/js/buildings.js`), drawn as y-sorted sprites.**
   - Use a 3/4 box projection: the top face (roof) is the footprint rect shifted up by height Z; the front wall is a w×Z rect at the footprint bottom.
   - Footprint tiles are `T.BUILDING`. The y-sort key is `b.y + b.h`.
   - Building kinds come from the server: `house`, `shop`, `inn`, `elder`, `shrine_small`, `temple`, `dorm`, `infirmary`, `gate` (torii), city `office`/`apartment`/`store`/`tower`/`station`, and `site_*` (abandoned school/hospital/factory/apartment/detention, shrine, grove, cemetery, tunnel, lake).
   - Japanese houses: kawara tile roofs with eaves, wood posts, shoji panels, and noren curtains with 店/宿/町.
   - City buildings: tall towers with window grids, neon vertical kanji signs and rooftop units. The station gets a big 渋谷 sign.
   - The temple gets a two-tier roof and red pillars. Sites get ruins, ivy and miasma.
   - Bake shadows. Lit windows act as night lights.
   - Remove the old `drawTile`/`drawBuilding`/`drawSite` code from `pixel.js` once nothing imports it.
4. **Atmosphere in `world.js`.**
   - Drifting cloud shadows.
   - Sakura petals and leaves by day; fireflies at night.
   - Purple wisps rising from CURSED tiles and sites.
   - Water sparkle, a vignette and time-of-day colour grading. `drawLighting` already exists.
   - Make the minimap prettier.
5. **Characters and curses (`pixel.js` `charSprite`/`curseSprite`).** A helper agent had started this; check whether anything landed.
   - Target sprites are about 24×36 with chibi-anime proportions, big eyes, 3-tone shading, automatic 1px outline and a 4-frame walk.
   - Keep the look fields and the 12 hair / 8 outfit / 8 accessory indices.
   - Update the sprite-size call sites:
     - `main.js`: the drawImage calls of `charSprite` (around lines 222, 382, 460, 605).
     - `battle.js`: around lines 329–330.
     - `world.js`: `drawCharacter`.
6. **UI (`style.css`, `index.html`, HUD code in `main.js`) and battle scenes (`battle.js`).** This was the other helper's track; check what landed.
   - UI target: Persona/JJK styling with angular ink panels, cursed-energy glows, segmented HP/CE bars, a visual-novel dialog box and ability cards.
   - Battle target: painted backdrops, screen shake, hit flashes, and Black Flash and domain-expansion spectacle.
   - Fonts must work offline (no CDN).
   - Keep every element id that the e2e scripts use: `#join-name`, `#btn-join`, `#create`, `[data-tech]`, `#cr-name`, `#btn-create`, `#hud`, `#dialog`, `#dialog-options`, `#battle-ui`, `#battle-menu .ab[data-ab]`, `#prompt`.
7. **Verify with Playwright screenshots**, then commit, rebuild the exe and push. Look at every screenshot before calling it done.
   - Load Playwright from `/opt/node-tools/node_modules/playwright/index.mjs`; Chromium is preinstalled.
   - Flow to drive: join, create a character (`[data-tech="limitless"]`), POST `/api/host/start`, walk with WASD, screenshot.
   - Use `/api/world` for town coordinates.

## Later request (after the overhaul)
The user wants an **infinite**, chunk-generated WorldBox-style world. It needs more biomes (desert, snow, jungle, swamp, savanna) and endless towns, sites and quests that scale with distance and level, while keeping towns playable (mayor, shop, inn). The current world is fixed at 200×200 and generated server-side in `engine/world.go`. Design the client chunk and prop systems so they don't assume a fixed size.

## Commit rules
End every commit message with:
```
Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01QCSmfA6bT7LK76BDTfP4GP
```
Never put model identifiers in code or files. Never commit `dist/` build junk beyond `release/`, or anything under `build/`.
