package dev.fairytail.cast;

import java.util.ArrayList;
import java.util.List;

/** A keyframed scalar curve over the cast timeline (time in ticks). Immutable once built. */
public final class Track {
    private final float[] times;
    private final float[] values;
    private final Ease[] eases;

    private Track(float[] times, float[] values, Ease[] eases) {
        this.times = times;
        this.values = values;
        this.eases = eases;
    }

    public static Builder at(double v0) {
        return new Builder((float) v0);
    }

    public float eval(float t) {
        int n = times.length;
        if (t <= times[0]) return values[0];
        if (t >= times[n - 1]) return values[n - 1];
        int i = 1;
        while (t > times[i]) i++;
        float span = times[i] - times[i - 1];
        float k = eases[i].apply((t - times[i - 1]) / span);
        return values[i - 1] + (values[i] - values[i - 1]) * k;
    }

    public static final class Builder {
        private final List<float[]> keys = new ArrayList<>();
        private final List<Ease> eases = new ArrayList<>();

        private Builder(float v0) {
            keys.add(new float[]{0, v0});
            eases.add(Ease.LINEAR);
        }

        public Builder to(double time, double value, Ease ease) {
            keys.add(new float[]{(float) time, (float) value});
            eases.add(ease);
            return this;
        }

        public Track build() {
            int n = keys.size();
            float[] t = new float[n], v = new float[n];
            for (int i = 0; i < n; i++) {
                t[i] = keys.get(i)[0];
                v[i] = keys.get(i)[1];
                if (i > 0 && t[i] <= t[i - 1]) throw new IllegalStateException("Track times must strictly increase");
            }
            return new Track(t, v, eases.toArray(new Ease[0]));
        }
    }
}
