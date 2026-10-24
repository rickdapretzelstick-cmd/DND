package dev.fairytail.registry;

import dev.fairytail.FairyTailMod;
import net.minecraft.core.particles.ParticleType;
import net.minecraft.core.particles.SimpleParticleType;
import net.minecraft.core.registries.Registries;
import net.neoforged.neoforge.registries.DeferredHolder;
import net.neoforged.neoforge.registries.DeferredRegister;

public final class ModParticles {
    public static final DeferredRegister<ParticleType<?>> PARTICLES = DeferredRegister.create(Registries.PARTICLE_TYPE, FairyTailMod.MOD_ID);

    public static final DeferredHolder<ParticleType<?>, SimpleParticleType> SPARKLE = PARTICLES.register("sparkle", () -> new SimpleParticleType(false));
    public static final DeferredHolder<ParticleType<?>, SimpleParticleType> FLARE = PARTICLES.register("flare", () -> new SimpleParticleType(false));
    public static final DeferredHolder<ParticleType<?>, SimpleParticleType> MOTE = PARTICLES.register("mote", () -> new SimpleParticleType(false));

    private ModParticles() {}
}
