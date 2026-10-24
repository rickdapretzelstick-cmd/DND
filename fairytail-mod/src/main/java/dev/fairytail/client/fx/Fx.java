package dev.fairytail.client.fx;

import net.minecraft.client.Minecraft;
import net.minecraft.client.multiplayer.ClientLevel;
import net.minecraft.client.particle.SpriteSet;

/** Small factory around {@link GlowParticle}; sprite sets are captured when the providers register. */
public final class Fx {
    static SpriteSet sparkle, flare, mote;

    public static void sparkle(ClientLevel l, double x, double y, double z, double vx, double vy, double vz, int rgb, float size, int life) {
        add(new GlowParticle(l, x, y, z, vx, vy, vz, sparkle, rgb, size, life, 0f, 0.92f, true));
    }

    /** Sparkle that sinks gently, for lingering energy. */
    public static void drift(ClientLevel l, double x, double y, double z, double vx, double vy, double vz, int rgb, float size, int life, float gravity) {
        add(new GlowParticle(l, x, y, z, vx, vy, vz, sparkle, rgb, size, life, gravity, 0.96f, true));
    }

    public static void mote(ClientLevel l, double x, double y, double z, double vx, double vy, double vz, int rgb, float size, int life) {
        add(new GlowParticle(l, x, y, z, vx, vy, vz, mote, rgb, size, life, 0f, 0.97f, false));
    }

    /** Frictionless mote: keeps its velocity, used for energy that converges onto the key. */
    public static void streak(ClientLevel l, double x, double y, double z, double vx, double vy, double vz, int rgb, float size, int life) {
        add(new GlowParticle(l, x, y, z, vx, vy, vz, mote, rgb, size, life, 0f, 1.0f, false));
    }

    public static void flare(ClientLevel l, double x, double y, double z, int rgb, float size, int life) {
        add(new GlowParticle(l, x, y, z, 0, 0, 0, flare, rgb, size, life, 0f, 1f, false));
    }

    private static void add(GlowParticle p) {
        Minecraft.getInstance().particleEngine.add(p);
    }

    private Fx() {}
}
