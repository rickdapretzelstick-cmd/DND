package dev.fairytail.registry;

import dev.fairytail.FairyTailMod;
import net.minecraft.core.registries.Registries;
import net.minecraft.sounds.SoundEvent;
import net.neoforged.neoforge.registries.DeferredHolder;
import net.neoforged.neoforge.registries.DeferredRegister;

/** Five layered, original (procedurally synthesised) sounds; see tools/gen_assets.py. */
public final class ModSounds {
    public static final DeferredRegister<SoundEvent> SOUNDS = DeferredRegister.create(Registries.SOUND_EVENT, FairyTailMod.MOD_ID);

    public static final DeferredHolder<SoundEvent, SoundEvent> KEY_JINGLE = reg("key_jingle");
    public static final DeferredHolder<SoundEvent, SoundEvent> RESONANCE_RISE = reg("resonance_rise");
    public static final DeferredHolder<SoundEvent, SoundEvent> ENERGY_BUILDUP = reg("energy_buildup");
    public static final DeferredHolder<SoundEvent, SoundEvent> GATE_OPEN = reg("gate_open");
    public static final DeferredHolder<SoundEvent, SoundEvent> SHIMMER_DECAY = reg("shimmer_decay");

    private static DeferredHolder<SoundEvent, SoundEvent> reg(String name) {
        return SOUNDS.register(name, () -> SoundEvent.createVariableRangeEvent(FairyTailMod.id(name)));
    }

    private ModSounds() {}
}
