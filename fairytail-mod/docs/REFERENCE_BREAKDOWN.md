# Reference → Implementation breakdown

## Honest note on sources

This phase was developed in a sandbox that **cannot play video or fetch fandom-wiki pages** (the wiki
returned HTTP 402; search returned only low-quality mirrors). So this document is **not** a frame-by-frame
transcription of the anime. It is built from what is well established about the series, and each row
says how confident I am. **Please review rows marked ◐/○ against the footage you care about** and tell me
what to adjust: every number below is a keyframe in `cast/PoseTracks.java` / `cast/ShotRig.java`, so
corrections are cheap.

Confidence: ● well established · ◐ recalled, plausible, worth checking · ○ my own dramatisation

## Canonical gate titles

The invocation is `Open, Gate of the <title>! <Spirit>!` (`ZodiacKey.invocation()` / `shout()`).

| Key | Gate title used | Epithet in tooltip | Confidence |
|---|---|---|---|
| Aquarius | Water Bearer | The Water Bearer | ● |
| Taurus | Golden Bull | The Golden Bull | ● |
| Cancer | Crab | The Giant Crab | ◐ (anime chant says "Crab"; your list says "Giant Crab" → kept as epithet) |
| Virgo | Maiden | The Maiden | ● |
| Sagittarius | Archer | The Archer | ● |
| Leo | Lion | The Lion | ● |
| Aries | Ram | The Ram | ● |
| Gemini | Twins | The Twins | ● |
| Scorpio | Scorpion | The Scorpion | ● |
| Capricorn | Goat | The Goat | ● |
| Pisces | Fish | The Paired Fish | ◐ ("Paired Fish" kept as epithet) |
| Libra | Scales | The Scales | ● |

Titles live in one enum (`key/ZodiacKey.java`); changing one string changes the invocation, the shout and
the tooltip.

## What Lucy does → what the mod does

| Observation | Conf. | Implementation |
|---|---|---|
| Lucy holds a golden key in her **right hand** and presents it in front of her; her stance is a small lunge with the free arm thrown back for balance | ● | Casting arm = the hand that used the item (right by default, mirrored for left). `LEG_FRONT/LEG_BACK`, `OFF_ARM_X/ROLL` |
| She **extends the key forward at arm's length** toward the point where the gate will appear, then **turns it like a key in a lock** | ● | `SHOULDER_X` reaches −1.60 rad (horizontal) at the thrust; `ELBOW` snaps from −2.15 to ≈0 (`OUT_EXPO`); `KEY_TWIST` turns the key 90° about its own axis between ticks 51–58 |
| Brief **wind-up** before the thrust | ◐ | draw-back at tick 44 (`TORSO_YAW` +0.34, elbow −2.15), 6-tick thrust, damped overshoot (`PoseTracks.settle`) |
| She **calls the gate aloud**: "Open, Gate of the …! Name!" | ● | `CinematicOverlay.invocation()` types the line in shot 2 (ticks 18–36); `shout()` slams the spirit name at tick 53 |
| Gate is a **cyan/white glowing circle**: thick ring, band of rune text, a disc split into dark wedges around a bright centre, bold curved light spikes bursting outward (**user-supplied anime frame, user confirms it applies to every key**) | ● (one frame) | `GateRenderer` + `effect/gate_design.png` (wedge disc + rune band), `effect/gate_outer.png` (ring of all 12 signs), `effect/spikes.png`; circle stands perpendicular to the key, centred just ahead of its tip so the key points into it |
| Colours: cyan-white gate light, soft glow, sparkles; not purple. The *key itself* has a soft golden glow | ◐ | `ZodiacKey.gateColor()` = `0x6FE6FF` for all keys; 12 % per-key accent tint |
| Spirits arrive on a **flash of light with a bell-like chime** | ◐ | impact at tick 62: white-gold flash, `gate_open` = low boom + two-note bell ("ting-tong") + airy whoosh (original synthesis) |
| Camera: close-ups of the face, the hand and the key, a push-in on the key at the moment of calling | ◐ | 5 shots, see below |
| Variation between episodes (static circle vs. flourish, with/without a pose hold) | ◐ | not reproduced as separate variants; one canonical, polished take. The framework (`CastTimeline` + tracks) takes new tracks without engine changes |

Per-key variation: the centre sign, a faint accent tint, one vanilla "flourish" particle and the swirl
speed (all in `ZodiacKey`). **Circle geometry and spin are identical for every key and spin directions
are unverified** — I could not find per-key circle references. If individual keys differ in the anime,
send frames and I will add per-key designs (the renderer already takes a texture + spin per key).

Hold behaviour (user request): the arm and key stay extended inside the open circle from the activation
(tick 62) to tick 80. The key's bit always points straight out of the fist, rigid with the forearm; it is
gripped by its bow just past the hand so it does not sink into the arm.

## Shot list (the single shared timeline, 20 ticks/s, 96 ticks = 4.8 s)

| Ticks | Shot | Camera (`ShotRig`) | Pose (`PoseTracks`) | FX / UI / Sound |
|---|---|---|---|---|
| 0–16 | 1 Preparation | swings wide around the character from the player's own view to the front (blend weight 0→1, plus a side arc) | arm rises, elbow folds, torso eases in, eyes on the key | key jingle (t3), glint flares (t9, t13), trails begin |
| 16–40 | 2 Declaration | slow dolly-in to a medium close-up of face + key | key raised beside the face, head lifts | invocation typed, resonance rise (t18), energy motes converge (t16→), helical swirl + orbiters, buildup (t38) |
| 40–62 | 3 Activation | tiny pull-back (anticipation) then push to ~1.5 m, low angle, dutch tilt up to −7°, look-at slides from face to key | draw-back → thrust (t50) → key turn (t51–58); shoulder shrug, torso unwinds | key-turn flare + ring (t54), name slam (t53), edge-glow, rising pitch |
| 62–82 | 4 Gate Opening | **punch out**: dolly back + FOV 33→82 on a hit, shake | held extended pose, spring settle | white-gold flash, gate ring bursts open (OUT_EXPO), rays, shock ring, sign bloom, 70 sparkles + ring bursts, `gate_open` (t62), shimmer decay (t66) |
| 82–96 | 5 Return | camera weight 1→0 (blends back to the player's true camera), FOV eases back | pose weight 1→0 | effects dissolve, HUD returns |

Camera ranges are in character space (forward / toward the casting arm / up from a chest-height pivot), so the
cinematic works for any position, either hand, and first- or third-person start. Rays are cast from
pivot to camera and the lens is pulled in if a block is in the way.
