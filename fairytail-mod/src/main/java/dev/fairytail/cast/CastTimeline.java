package dev.fairytail.cast;

/**
 * The single clock every presentation system reads: pose, camera, overlay text, particles and sound all
 * evaluate against the same tick value {@code t} (0 .. TOTAL), so they cannot drift apart.
 *
 * <pre>
 *  0 ─ 16   Shot 1  Preparation   camera swings round to the front, key raised, first glint
 * 16 ─ 40   Shot 2  Declaration   medium close-up, key presented, invocation typed, energy gathers
 * 40 ─ 62   Shot 3  Activation    anticipation draw-back, push-in on the arm, thrust + key turn
 * 62 ─ 82   Shot 4  Gate Opening  golden flash, gate ring + zodiac glyphs burst open, camera punches out
 * 82 ─ 96   Shot 5  Return        effects dissolve, camera + pose blend back to gameplay
 * </pre>
 */
public final class CastTimeline {
    /** 96 ticks = 4.8 s. Keep equal to ServerCastManager.CAST_TICKS. */
    public static final int TOTAL = 96;

    // marker ticks
    public static final float PREP_END = 16;
    public static final float RAISE = 30;
    public static final float DRAW_BACK = 44;
    /** arm is fully extended and the key is "in the lock" */
    public static final float THRUST = 50;
    /** the key starts turning, then the turn completes, and only THEN the gate activates */
    public static final float TURN_START = 52;
    public static final float TURN_END = 61;
    public static final float IMPACT = 62;
    /** arm stays held out, key inside the circle, until here */
    public static final float HOLD_END = 80;
    public static final float RELEASE = 86;

    // sound cues
    public static final int CUE_KEY_JINGLE = 3;
    public static final int CUE_RESONANCE = 18;
    public static final int CUE_BUILDUP = 38;
    public static final int CUE_GATE = (int) IMPACT;
    public static final int CUE_SHIMMER = 66;
    public static final int CUE_CLICK_A = 55;
    public static final int CUE_CLICK_B = 60;

    public static float smooth(float a, float b, float t) {
        float k = Math.max(0, Math.min(1, (t - a) / (b - a)));
        return k * k * (3 - 2 * k);
    }

    /** 0..1 how strongly the cinematic camera is blended in (ramps in over shot 1, out over shot 5). */
    public static float cameraWeight(float t) {
        return smooth(0, 14, t) * (1 - smooth(RELEASE - 2, TOTAL - 1, t));
    }

    /** 0..1 how strongly the cast pose overrides the normal player pose. */
    public static float poseWeight(float t) {
        return smooth(0, 7, t) * (1 - smooth(RELEASE + 2, TOTAL, t));
    }

    /** Full-screen golden flash intensity: a small pulse on the key turn, a full white-out at the impact. */
    public static float flash(float t) {
        float f = 0;
        if (t >= TURN_END - 1 && t < IMPACT - 1) f = 0.2f * smooth(TURN_END - 1, IMPACT - 1, t);
        if (t >= IMPACT - 1) f = Math.max(f, 1 - smooth(IMPACT, IMPACT + 9, t));
        return f;
    }

    private CastTimeline() {}
}
