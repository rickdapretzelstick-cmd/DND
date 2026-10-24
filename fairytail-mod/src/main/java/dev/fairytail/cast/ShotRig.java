package dev.fairytail.cast;

import static dev.fairytail.cast.CastTimeline.*;

/**
 * Camera keyframes in "character space": distances (blocks) from a pivot at the caster's chest,
 * along the caster's facing (F), toward the casting arm's side (R) and up (U). Writing the shots this
 * way keeps them valid for any player position and either hand.
 */
public final class ShotRig {
    // camera position
    public static final Track CAM_F = Track.at(2.9f)
            .to(PREP_END, 2.9, Ease.IN_OUT_CUBIC)
            .to(RAISE, 2.45, Ease.IN_OUT_SINE)            // shot 2: slow dolly-in, medium close-up
            .to(DRAW_BACK, 2.25, Ease.IN_OUT_SINE)
            .to(DRAW_BACK + 3, 2.38, Ease.OUT_QUAD)       // tiny pull-back: anticipation
            .to(THRUST + 6, 1.72, Ease.IN_OUT_CUBIC)      // shot 3: push toward the outstretched arm
            .to(IMPACT, 1.52, Ease.LINEAR)
            .to(IMPACT + 3, 2.9, Ease.OUT_EXPO)           // shot 4: punch out as the gate opens
            .to(RELEASE, 3.5, Ease.OUT_CUBIC).build();
    public static final Track CAM_R = Track.at(0.75f)
            .to(PREP_END, 0.75, Ease.IN_OUT_CUBIC)
            .to(DRAW_BACK, 0.55, Ease.IN_OUT_SINE)
            .to(THRUST + 6, 0.95, Ease.IN_OUT_CUBIC)
            .to(IMPACT, 0.88, Ease.LINEAR)
            .to(IMPACT + 3, 0.30, Ease.OUT_EXPO)
            .to(RELEASE, 0.55, Ease.OUT_CUBIC).build();
    public static final Track CAM_U = Track.at(0.15f)
            .to(PREP_END, 0.18, Ease.IN_OUT_CUBIC)
            .to(RAISE, 0.26, Ease.IN_OUT_SINE)
            .to(DRAW_BACK, 0.30, Ease.IN_OUT_SINE)
            .to(THRUST + 6, -0.10, Ease.IN_OUT_CUBIC)     // low angle on the thrust
            .to(IMPACT, -0.04, Ease.LINEAR)
            .to(IMPACT + 3, 0.22, Ease.OUT_EXPO)
            .to(RELEASE, 0.42, Ease.OUT_CUBIC).build();

    // look-at target (face-level) before blending toward the key
    public static final Track TGT_U = Track.at(0.22f)
            .to(DRAW_BACK, 0.34, Ease.IN_OUT_SINE)
            .to(THRUST + 6, 0.10, Ease.IN_OUT_CUBIC)
            .to(IMPACT + 3, 0.30, Ease.OUT_CUBIC).build();
    /** 0 = look at the face, 1 = look at the key */
    public static final Track KEY_FOCUS = Track.at(0)
            .to(PREP_END, 0.25, Ease.IN_OUT_SINE)
            .to(RAISE, 0.12, Ease.IN_OUT_SINE)
            .to(DRAW_BACK + 2, 0.30, Ease.IN_OUT_SINE)
            .to(THRUST + 6, 0.92, Ease.IN_OUT_CUBIC)
            .to(IMPACT, 0.95, Ease.LINEAR)
            .to(IMPACT + 4, 0.55, Ease.OUT_CUBIC)
            .to(HOLD_END + 8, 0.0, Ease.IN_OUT_SINE).build();

    public static final Track FOV = Track.at(64)
            .to(PREP_END, 60, Ease.IN_OUT_CUBIC)
            .to(DRAW_BACK, 48, Ease.IN_OUT_SINE)
            .to(THRUST + 6, 36, Ease.IN_OUT_CUBIC)
            .to(IMPACT, 33, Ease.LINEAR)
            .to(IMPACT + 3, 82, Ease.OUT_EXPO)
            .to(RELEASE, 60, Ease.OUT_CUBIC).build();
    /** Dutch tilt in degrees */
    public static final Track ROLL = Track.at(0)
            .to(PREP_END, -1.5, Ease.IN_OUT_SINE)
            .to(DRAW_BACK, 1.5, Ease.IN_OUT_SINE)
            .to(THRUST + 6, -6.5, Ease.IN_OUT_CUBIC)
            .to(IMPACT, -7.0, Ease.LINEAR)
            .to(IMPACT + 4, 3.5, Ease.OUT_EXPO)
            .to(RELEASE, 0, Ease.IN_OUT_SINE).build();
    /** Camera shake amplitude (blocks). */
    public static final Track SHAKE = Track.at(0)
            .to(THRUST, 0, Ease.LINEAR)
            .to(THRUST + 1, 0.012, Ease.OUT_QUAD)
            .to(THRUST + 7, 0, Ease.IN_OUT_SINE)
            .to(IMPACT - 0.01f, 0, Ease.LINEAR)
            .to(IMPACT + 0.4f, 0.11, Ease.OUT_QUAD)
            .to(IMPACT + 14, 0, Ease.OUT_CUBIC).build();

    private ShotRig() {}
}
