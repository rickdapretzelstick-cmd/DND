package dev.fairytail;

import dev.fairytail.net.FairyTailNetwork;
import dev.fairytail.registry.ModCreativeTabs;
import dev.fairytail.registry.ModItems;
import dev.fairytail.registry.ModParticles;
import dev.fairytail.registry.ModSounds;
import dev.fairytail.server.ServerCastManager;
import net.minecraft.resources.ResourceLocation;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.fml.common.Mod;
import net.neoforged.neoforge.common.NeoForge;

/** Common (both sides) entry point: registries, networking and the server-side cast validator. */
@Mod(FairyTailMod.MOD_ID)
public class FairyTailMod {
    public static final String MOD_ID = "fairytail";
    public static final org.slf4j.Logger LOGGER = com.mojang.logging.LogUtils.getLogger();

    public FairyTailMod(IEventBus modBus) {
        ModItems.ITEMS.register(modBus);
        ModSounds.SOUNDS.register(modBus);
        ModParticles.PARTICLES.register(modBus);
        ModCreativeTabs.TABS.register(modBus);
        modBus.addListener(FairyTailNetwork::register);
        NeoForge.EVENT_BUS.register(ServerCastManager.class);
    }

    public static ResourceLocation id(String path) {
        return ResourceLocation.fromNamespaceAndPath(MOD_ID, path);
    }
}
