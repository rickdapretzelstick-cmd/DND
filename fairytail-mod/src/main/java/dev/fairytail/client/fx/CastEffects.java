package dev.fairytail.client.fx;

import dev.fairytail.cast.CastTimeline;
import dev.fairytail.client.ClientCast;
import dev.fairytail.key.ZodiacKey;
import dev.fairytail.registry.ModSounds;
import net.minecraft.client.Minecraft;
import net.minecraft.client.multiplayer.ClientLevel;
import net.minecraft.client.player.AbstractClientPlayer;
import net.minecraft.sounds.SoundEvent;
import net.minecraft.sounds.SoundSource;
import net.minecraft.util.Mth;
import net.minecraft.util.RandomSource;
import net.minecraft.world.phys.Vec3;

import static dev.fairytail.cast.CastTimeline.*;

/**
 * The effect director. Called once per integer timeline tick for every active cast (local or remote);
 * everything it does is a pure function of the tick number, so a client that starts tracking a cast
 * mid-way catches up consistently and replays are identical.
 *
 * <p>Layers: (1) key glint, (2) gathering motes converging on the key head, (3) a helical swirl around the
 * key axis plus orbiters, (4) thin trails along the key's path, (5) the impact burst, (6) lingering
 * shimmer, and the zodiac-specific flourish particle on top.
 */
public final class CastEffects {
    private static final int GOLD = 0xFFD75E;
    private static final int PALE = 0xFFF3B8;
    private static final int WHITE = 0xFFFFFF;

    public static void tick(ClientCast c, AbstractClientPlayer p, int tick) {
        if (!(p.level() instanceof ClientLevel level)) return;
        RandomSource rng = level.random;
        ZodiacKey key = c.key;
        Vec3 a = anchor(c, p);
        Vec3 axis = c.axis.lengthSqr() < 1e-4 ? new Vec3(0, 1, 0) : c.axis.normalize();
        int accent = mix(GOLD, key.accent, 0.55f);

        sounds(level, a, tick, c);

        // (4) trails: a short fading streak laid along the path the key head travelled since last tick
        if (tick >= 4 && tick <= 84) {
            Vec3 prev = c.hasAnchor ? c.prevAnchor : a;
            double speed = a.distanceTo(prev);
            int n = 2 + (int) Math.min(6, speed * 40);
            for (int i = 0; i < n; i++) {
                Vec3 q = prev.lerp(a, (i + rng.nextFloat()) / n);
                Fx.mote(level, q.x, q.y, q.z, rnd(rng, 0.004), rnd(rng, 0.004) + 0.003, rnd(rng, 0.004), i % 3 == 0 ? WHITE : GOLD, 0.11f + (float) speed, 7 + rng.nextInt(5));
            }
        }

        // (1) shimmer: the key catching the light
        if (tick == 9 || tick == 13) {
            Fx.flare(level, a.x, a.y, a.z, PALE, 0.55f, 6);
            for (int i = 0; i < 6; i++) Fx.sparkle(level, a.x + rnd(rng, 0.12), a.y + rnd(rng, 0.12), a.z + rnd(rng, 0.12), rnd(rng, 0.02), 0.01 + rng.nextFloat() * 0.02, rnd(rng, 0.02), PALE, 0.13f, 14);
        }

        // (2) gathering energy: motes spawn on a shrinking shell and fly inward to the key head
        if (tick >= 16 && tick < THRUST + 6) {
            float prog = Mth.clamp((tick - 16) / 34f, 0, 1);
            int n = 2 + (int) (prog * 7);
            for (int i = 0; i < n; i++) {
                Vec3 dir = randomDir(rng);
                double r = Mth.lerp(prog, 1.7, 0.9) * (0.6 + rng.nextFloat() * 0.6);
                Vec3 o = a.add(dir.scale(r));
                int life = 10 + rng.nextInt(5);
                Vec3 v = a.subtract(o).scale(1.0 / life);
                Fx.streak(level, o.x, o.y, o.z, v.x, v.y, v.z, rng.nextInt(4) == 0 ? accent : GOLD, 0.12f + prog * 0.08f, life);
            }
        }

        // (3) helical swirl up the key axis + counter-rotating orbiters
        if (tick >= 20 && tick < IMPACT) {
            float prog = Mth.clamp((tick - 20) / 42f, 0, 1);
            Vec3 u = axis.cross(Math.abs(axis.y) > 0.9 ? new Vec3(1, 0, 0) : new Vec3(0, 1, 0)).normalize();
            Vec3 v = axis.cross(u).normalize();
            double base = tick * 0.55 * key.swirlSpeed;
            double radius = 0.22 + 0.16 * (1 - prog);
            for (int k = 0; k < key.orbiters; k++) {
                double ang = base * (k == 0 ? 1 : -1.3) + k * Math.PI;
                Vec3 off = u.scale(Math.cos(ang) * radius).add(v.scale(Math.sin(ang) * radius)).add(axis.scale(Math.sin(ang * 0.5) * 0.18));
                Vec3 q = a.add(off);
                Fx.sparkle(level, q.x, q.y, q.z, 0, 0.004, 0, k == 0 ? GOLD : accent, 0.17f, 9);
            }
        }

        // flourish: the zodiac's own element
        if (tick >= 24 && tick < IMPACT && tick % 2 == 0) {
            level.addParticle(key.flourish, a.x + rnd(rng, 0.35), a.y + rnd(rng, 0.25), a.z + rnd(rng, 0.35), rnd(rng, 0.01), 0.01, rnd(rng, 0.01));
        }

        // key turn: a small pulse
        if (tick == (int) THRUST + 4) {
            Fx.flare(level, a.x, a.y, a.z, GOLD, 1.1f, 8);
            ring(level, rng, a, axis, 14, 0.12, 0.05, accent, 0.14f);
        }

        // (5) impact: the gate opens
        if (tick == (int) IMPACT) {
            Fx.flare(level, a.x, a.y, a.z, WHITE, 4.2f, 10);
            Fx.flare(level, a.x, a.y, a.z, GOLD, 6.5f, 14);
            Vec3 fwd = forward(c, p);
            ring(level, rng, a, fwd, 72, 0.22, 0.015, GOLD, 0.2f);
            ring(level, rng, a, fwd, 36, 0.14, 0.01, WHITE, 0.16f);
            ring(level, rng, a, fwd, 30, 0.30, 0.02, accent, 0.18f);
            for (int i = 0; i < 70; i++) {
                Vec3 d = randomDir(rng);
                double sp = 0.05 + rng.nextFloat() * 0.22;
                Fx.sparkle(level, a.x, a.y, a.z, d.x * sp, d.y * sp + 0.02, d.z * sp, rng.nextInt(3) == 0 ? WHITE : GOLD, 0.16f + rng.nextFloat() * 0.18f, 20 + rng.nextInt(14));
            }
            for (int i = 0; i < 24; i++) {
                Vec3 d = randomDir(rng);
                level.addParticle(key.flourish, a.x, a.y, a.z, d.x * 0.18, d.y * 0.18 + 0.03, d.z * 0.18);
            }
        }

        // (6) lingering shimmer after the gate has opened
        if (tick > IMPACT && tick <= 84) {
            float fade = 1f - (tick - IMPACT) / 24f;
            int n = (int) Math.max(1, 7 * fade);
            Vec3 fwd = forward(c, p);
            for (int i = 0; i < n; i++) {
                Vec3 d = randomDir(rng);
                double r = 0.4 + rng.nextFloat() * 1.8 * (1 - 0.5 * (1 - fade));
                Vec3 o = a.add(d.scale(r)).add(fwd.scale(0.15));
                Fx.drift(level, o.x, o.y, o.z, rnd(rng, 0.01), 0.012 + rng.nextFloat() * 0.015, rnd(rng, 0.01), rng.nextInt(3) == 0 ? WHITE : GOLD, 0.12f + rng.nextFloat() * 0.12f, 22 + rng.nextInt(16), -0.1f);
            }
        }
    }

    private static void sounds(ClientLevel level, Vec3 a, int tick, ClientCast c) {
        float vol = c.local ? 1.0f : 0.8f;
        if (tick == CUE_KEY_JINGLE) play(level, a, ModSounds.KEY_JINGLE.get(), 0.9f * vol, 1.0f);
        if (tick == CUE_RESONANCE) play(level, a, ModSounds.RESONANCE_RISE.get(), 0.75f * vol, 1.0f);
        if (tick == CUE_BUILDUP) play(level, a, ModSounds.ENERGY_BUILDUP.get(), 0.8f * vol, 1.0f);
        if (tick == (int) THRUST + 1) play(level, a, ModSounds.KEY_JINGLE.get(), 0.7f * vol, 1.35f);
        if (tick == CUE_GATE) play(level, a, ModSounds.GATE_OPEN.get(), 1.25f * vol, 1.0f);
        if (tick == CUE_SHIMMER) play(level, a, ModSounds.SHIMMER_DECAY.get(), 0.9f * vol, 1.0f);
    }

    private static void play(ClientLevel level, Vec3 at, SoundEvent ev, float vol, float pitch) {
        level.playLocalSound(at.x, at.y, at.z, ev, SoundSource.PLAYERS, vol, pitch, false);
    }

    /** A ring of sparkles in the plane perpendicular to {@code normal}, expanding outward. */
    private static void ring(ClientLevel level, RandomSource rng, Vec3 c, Vec3 normal, int count, double speed, double jitter, int rgb, float size) {
        Vec3 n = normal.normalize();
        Vec3 u = n.cross(Math.abs(n.y) > 0.9 ? new Vec3(1, 0, 0) : new Vec3(0, 1, 0)).normalize();
        Vec3 v = n.cross(u).normalize();
        for (int i = 0; i < count; i++) {
            double ang = i * Math.PI * 2 / count;
            Vec3 d = u.scale(Math.cos(ang)).add(v.scale(Math.sin(ang)));
            Fx.sparkle(level, c.x, c.y, c.z, d.x * speed + rnd(rng, jitter), d.y * speed + rnd(rng, jitter), d.z * speed + rnd(rng, jitter), rgb, size, 18 + rng.nextInt(6));
        }
    }

    /** World position of the key head; falls back to a plausible spot until the first real transform arrives. */
    public static Vec3 anchor(ClientCast c, AbstractClientPlayer p) {
        if (c.hasAnchor) return c.anchor;
        Vec3 f = forward(c, p);
        return p.position().add(0, p.getBbHeight() * 0.8, 0).add(f.scale(0.6));
    }

    public static Vec3 forward(ClientCast c, AbstractClientPlayer p) {
        double y = Math.toRadians(c.local ? c.facingYaw : p.yBodyRot);
        return new Vec3(-Math.sin(y), 0, Math.cos(y));
    }

    private static Vec3 randomDir(RandomSource r) {
        double z = r.nextDouble() * 2 - 1, th = r.nextDouble() * Math.PI * 2, s = Math.sqrt(1 - z * z);
        return new Vec3(s * Math.cos(th), z, s * Math.sin(th));
    }

    private static double rnd(RandomSource r, double amp) {
        return (r.nextDouble() - 0.5) * 2 * amp;
    }

    public static int mix(int a, int b, float t) {
        int r = (int) Mth.lerp(t, (a >> 16) & 255, (b >> 16) & 255);
        int g = (int) Mth.lerp(t, (a >> 8) & 255, (b >> 8) & 255);
        int bl = (int) Mth.lerp(t, a & 255, b & 255);
        return (r << 16) | (g << 8) | bl;
    }

    private CastEffects() {}
}
