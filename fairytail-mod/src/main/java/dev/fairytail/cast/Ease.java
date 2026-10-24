package dev.fairytail.cast;

import java.util.function.DoubleUnaryOperator;

/** Easing curves over [0,1]. Segment easing is applied on the segment that *ends* at a keyframe. */
public enum Ease {
    LINEAR(x -> x),
    IN_QUAD(x -> x * x),
    OUT_QUAD(x -> 1 - (1 - x) * (1 - x)),
    IN_OUT_SINE(x -> -(Math.cos(Math.PI * x) - 1) / 2),
    IN_OUT_CUBIC(x -> x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2),
    OUT_CUBIC(x -> 1 - Math.pow(1 - x, 3)),
    IN_CUBIC(x -> x * x * x),
    OUT_EXPO(x -> x >= 1 ? 1 : 1 - Math.pow(2, -10 * x)),
    /** Overshoots slightly, then settles: good for a snapping thrust. */
    OUT_BACK(x -> {
        double c1 = 1.35, c3 = c1 + 1;
        return 1 + c3 * Math.pow(x - 1, 3) + c1 * Math.pow(x - 1, 2);
    }),
    IN_OUT_BACK(x -> {
        double c1 = 1.2, c2 = c1 * 1.525;
        return x < 0.5
                ? (Math.pow(2 * x, 2) * ((c2 + 1) * 2 * x - c2)) / 2
                : (Math.pow(2 * x - 2, 2) * ((c2 + 1) * (x * 2 - 2) + c2) + 2) / 2;
    });

    private final DoubleUnaryOperator fn;

    Ease(DoubleUnaryOperator fn) {
        this.fn = fn;
    }

    public float apply(float x) {
        return (float) fn.applyAsDouble(Math.max(0, Math.min(1, x)));
    }
}
