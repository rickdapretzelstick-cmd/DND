# Fairy Tail: Celestial Spirit Keys — Phase 1

NeoForge 1.21.1 mod: Lucy's twelve Golden Zodiac Keys and a cinematic **Gate Opening**. No spirits are
summoned (Phase 2 is not started).

## Build
```
./gradlew build            # -> build/libs/fairytail-0.1.0.jar  (needs JDK 21)
```

## Use
Creative tab **Fairy Tail: Golden Zodiac Keys** (or `/give @s fairytail:key_aquarius`, `key_taurus`, ...).
Hold a key and **right-click**. The server validates, then every nearby client plays the cast; only the
caster gets the cinematic camera/HUD. Cast length 96 ticks (4.8 s), then a 5.4 s cooldown.

## What happens (one shared 96-tick timeline; see docs/REFERENCE_BREAKDOWN.md)
1. Camera swings round to the front; the key is raised and glints.
2. Medium close-up; invocation typed: `Open, Gate of the Water Bearer!`; energy gathers.
3. Draw-back, the arm extends with the key's end pointing straight out of the fist, the key **turns**.
4. Only after the turn completes: flash, the gate opens (cyan wedge/rune circle, ring of all 12 signs,
   spikes), name slams in (`AQUARIUS!`). Nothing is summoned.
5. **Hold**: the arm stays out, key inside the circle, until tick 80; then everything eases back and the
   camera/pose/HUD return. Death, dimension change, logout or a lost packet also end it cleanly.

## Layout
| Area | Files |
|---|---|
| Registration / metadata | `registry/*`, `key/ZodiacKey.java` (titles, signs, accents, spins) |
| Timeline + keyframes | `cast/CastTimeline`, `PoseTracks`, `ShotRig`, `Track`, `Ease` |
| Server + network | `server/ServerCastManager`, `net/*` |
| Player animation | `mixin/PlayerModelMixin`, `client/model/*` (arm split with a real elbow, skin preserved) |
| Key in hand | `client/CastKeyLayer`, `client/KeyBakedModel` |
| Camera / HUD / text | `mixin/CameraMixin`, `client/CinematicCamera`, `client/CinematicOverlay` |
| Particles / gate / sound | `client/fx/*`, `client/GateRenderer`, `registry/ModSounds` |
| Assets | `tools/gen_assets.py` regenerates all models, icons, textures, sounds |

## Reproducible tests
Server: `./gradlew runServer` (accept `run/eula.txt`, `online-mode=false`, op `Dev`).
Headless client + screenshots (software GL, real item -> server -> packet -> client path):
```
tools/autotest.sh aquarius 8,36,56,62,72     # key, screenshot ticks
tools/autotest.sh all 44,72                   # every key
tools/autotest.sh aquarius 40,62 back 2       # 3rd-person start, 2 repeats
tools/autotest.sh aquarius 36,56 first 1 true # no particles: inspect pose/key only
```
Output: `run/screenshots/`, log lines `[autotest] ... restored=true`.

## Verified vs not (honest status)
Verified in this sandbox: builds a JAR; dedicated server boots with the mod; headless client (Mesa software
GL) joins a dedicated server, casts through the real path, camera/pose/HUD restore (`restored=true`) in
first-person, third-person-back and left-handed casts; screenshots inspected for all stages on Aquarius.
**Not verified:** audio (no sound device here, sounds are synthesized but never heard), high/low FPS
behaviour beyond the ~5 fps software renderer, real multiplayer with a second viewer, left-handed casts,
slim-vs-classic skin comparison, armour/elytra interaction, other mods.
**Not canon-verified:** exact key head designs (cuboid approximations from general knowledge), the circle
geometry/spin (one user-supplied frame), choreography timings. Refine key models in Blockbench
(`assets/fairytail/models/item/hand/*.json` are plain Java Block/Item models).
