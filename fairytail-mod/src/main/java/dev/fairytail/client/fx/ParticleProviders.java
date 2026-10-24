package dev.fairytail.client.fx;

import dev.fairytail.registry.ModParticles;
import net.minecraft.client.particle.ParticleProvider;
import net.minecraft.client.particle.SpriteSet;
import net.minecraft.core.particles.SimpleParticleType;
import net.neoforged.neoforge.client.event.RegisterParticleProvidersEvent;

public final class ParticleProviders {
    public static void register(RegisterParticleProvidersEvent e) {
        e.registerSpriteSet(ModParticles.SPARKLE.get(), s -> { Fx.sparkle = s; return provider(s, 0.2f); });
        e.registerSpriteSet(ModParticles.FLARE.get(), s -> { Fx.flare = s; return provider(s, 0.6f); });
        e.registerSpriteSet(ModParticles.MOTE.get(), s -> { Fx.mote = s; return provider(s, 0.12f); });
    }

    /** Provider used by /particle: gold, short-lived. */
    private static ParticleProvider<SimpleParticleType> provider(SpriteSet s, float size) {
        return (type, level, x, y, z, vx, vy, vz) -> new GlowParticle(level, x, y, z, vx, vy, vz, s, 0xFFD75E, size, 20, 0f, 0.95f, true);
    }

    private ParticleProviders() {}
}
