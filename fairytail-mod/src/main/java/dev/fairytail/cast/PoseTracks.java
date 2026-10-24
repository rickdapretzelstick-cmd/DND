package dev.fairytail.cast;

import static dev.fairytail.cast.CastTimeline.*;

/**
 * Keyframes for the custom casting animation. Values are written for a RIGHT-handed cast in Minecraft
 * model space (radians); {@code ClientCastPose} mirrors them for a left-hand cast.
 *
 * <p>The choreography follows the way Lucy opens a gate in the series (recalled from the anime, see
 * docs/REFERENCE_BREAKDOWN.md): the key is brought up in front of her, presented, drawn back for a beat,
 * then thrust forward at arm's length and turned like a key in a lock while she calls the gate.
 *
 * <p>Model-space reminders: rightArm.xRot = -PI/2 lifts the arm to horizontal-forward; negative arm
 * yRot sweeps the right arm across the chest; negative forearm xRot bends the elbow up.
 */
public final class PoseTracks {
    // shoulder (upper arm)
    public static final Track SHOULDER_X = Track.at(0)
            .to(PREP_END, -0.85, Ease.OUT_CUBIC)
            .to(RAISE, -1.32, Ease.IN_OUT_CUBIC)
            .to(DRAW_BACK, -1.08, Ease.IN_OUT_SINE)
            .to(THRUST, -1.60, Ease.OUT_BACK)
            .to(HOLD_END, -1.56, Ease.IN_OUT_SINE)
            .to(RELEASE, -0.95, Ease.IN_OUT_CUBIC)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();
    public static final Track SHOULDER_YAW = Track.at(0)
            .to(PREP_END, -0.18, Ease.OUT_CUBIC)
            .to(RAISE, -0.36, Ease.IN_OUT_SINE)
            .to(DRAW_BACK, -0.30, Ease.IN_OUT_SINE)
            .to(THRUST, -0.05, Ease.OUT_CUBIC)
            .to(HOLD_END, -0.08, Ease.IN_OUT_SINE)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();
    public static final Track SHOULDER_ROLL = Track.at(0)
            .to(RAISE, 0.12, Ease.IN_OUT_SINE)
            .to(THRUST, 0.04, Ease.OUT_CUBIC)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();
    /** how far the shoulder is shrugged up (model units) */
    public static final Track SHRUG = Track.at(0)
            .to(DRAW_BACK, 0.2, Ease.IN_OUT_SINE)
            .to(THRUST, 0.9, Ease.OUT_CUBIC)
            .to(HOLD_END, 0.7, Ease.IN_OUT_SINE)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();

    // elbow (forearm relative to upper arm); negative bends the forearm upward/forward
    public static final Track ELBOW = Track.at(0)
            .to(PREP_END, -1.40, Ease.OUT_CUBIC)
            .to(RAISE, -1.95, Ease.IN_OUT_CUBIC)
            .to(DRAW_BACK, -2.15, Ease.IN_OUT_SINE)
            .to(THRUST, -0.04, Ease.OUT_EXPO)
            .to(HOLD_END, -0.14, Ease.IN_OUT_SINE)
            .to(RELEASE, -0.9, Ease.IN_OUT_CUBIC)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();

    // key in hand: roll about its own axis (the "turn of the key") and tilt relative to the forearm
    public static final Track KEY_TWIST = Track.at(0)
            .to(DRAW_BACK, 0, Ease.LINEAR)
            .to(THRUST + 1, 0.12, Ease.LINEAR)
            .to(TURN_END, 1.571f, Ease.IN_OUT_BACK)
            .to(HOLD_END, 1.571f, Ease.LINEAR)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();
    public static final Track KEY_FLEX = Track.at(0.1f)
            .to(PREP_END, -0.15f, Ease.IN_OUT_SINE)
            .to(RAISE, -0.30f, Ease.IN_OUT_SINE)
            .to(THRUST, -0.62f, Ease.OUT_CUBIC)
            .to(HOLD_END, -0.5f, Ease.IN_OUT_SINE)
            .to(TOTAL, 0.1f, Ease.IN_OUT_CUBIC).build();
    /** key size multiplier: grows slightly as it is presented, pulses on the turn */
    public static final Track KEY_SCALE = Track.at(0.85f)
            .to(RAISE, 1.0f, Ease.OUT_CUBIC)
            .to(THRUST, 1.08f, Ease.OUT_CUBIC)
            .to(IMPACT, 1.22f, Ease.IN_OUT_SINE)
            .to(IMPACT + 6, 1.0f, Ease.OUT_CUBIC)
            .to(TOTAL, 0.85f, Ease.IN_OUT_SINE).build();

    // torso & head
    /** body yRot: positive winds the (right) shoulder back, negative brings it forward */
    public static final Track TORSO_YAW = Track.at(0)
            .to(PREP_END, 0.10, Ease.IN_OUT_SINE)
            .to(DRAW_BACK, 0.34, Ease.IN_OUT_CUBIC)
            .to(THRUST + 1, -0.46, Ease.OUT_EXPO)
            .to(HOLD_END, -0.38, Ease.IN_OUT_SINE)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();
    public static final Track HEAD_PITCH = Track.at(0)
            .to(PREP_END, 0.20, Ease.IN_OUT_SINE)      // eyes on the key
            .to(RAISE, -0.05, Ease.IN_OUT_SINE)         // lifts, resolute
            .to(DRAW_BACK, 0.12, Ease.IN_OUT_SINE)
            .to(THRUST + 2, -0.14, Ease.OUT_CUBIC)      // chin up as she calls the gate
            .to(HOLD_END, -0.08, Ease.IN_OUT_SINE)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();
    /** extra head yaw on top of the counter-rotation that keeps the face toward the viewer */
    public static final Track HEAD_YAW_BIAS = Track.at(0)
            .to(PREP_END, 0.10, Ease.IN_OUT_SINE)
            .to(RAISE, -0.06, Ease.IN_OUT_SINE)
            .to(THRUST, -0.12, Ease.OUT_CUBIC)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();

    // balancing arm and stance
    public static final Track OFF_ARM_X = Track.at(0)
            .to(PREP_END, 0.18, Ease.IN_OUT_SINE)
            .to(DRAW_BACK, -0.45, Ease.IN_OUT_SINE)
            .to(THRUST + 2, 0.40, Ease.OUT_BACK)
            .to(HOLD_END, 0.30, Ease.IN_OUT_SINE)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();
    public static final Track OFF_ARM_ROLL = Track.at(0)
            .to(DRAW_BACK, 0.10, Ease.IN_OUT_SINE)
            .to(THRUST + 2, 0.55, Ease.OUT_BACK)
            .to(HOLD_END, 0.42, Ease.IN_OUT_SINE)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();
    /** leg of the casting side steps forward (negative) while the other pushes back */
    public static final Track LEG_FRONT = Track.at(0)
            .to(DRAW_BACK, 0.12, Ease.IN_OUT_SINE)
            .to(THRUST + 1, -0.38, Ease.OUT_CUBIC)
            .to(HOLD_END, -0.32, Ease.IN_OUT_SINE)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();
    public static final Track LEG_BACK = Track.at(0)
            .to(DRAW_BACK, -0.10, Ease.IN_OUT_SINE)
            .to(THRUST + 1, 0.36, Ease.OUT_CUBIC)
            .to(HOLD_END, 0.30, Ease.IN_OUT_SINE)
            .to(TOTAL, 0, Ease.IN_OUT_CUBIC).build();

    /** Overlapping action: a damped spring that rings through the arm after the thrust lands. */
    public static float settle(float t) {
        float d = t - THRUST;
        if (d < 0) return 0;
        return (float) (Math.exp(-d * 0.22) * Math.sin(d * 0.95) * 0.10);
    }

    private PoseTracks() {}
}
