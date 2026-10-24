package dev.fairytail.registry;

import dev.fairytail.FairyTailMod;
import dev.fairytail.key.ZodiacKey;
import net.minecraft.core.registries.Registries;
import net.minecraft.network.chat.Component;
import net.minecraft.world.item.CreativeModeTab;
import net.minecraft.world.item.ItemStack;
import net.neoforged.neoforge.registries.DeferredHolder;
import net.neoforged.neoforge.registries.DeferredRegister;

public final class ModCreativeTabs {
    public static final DeferredRegister<CreativeModeTab> TABS = DeferredRegister.create(Registries.CREATIVE_MODE_TAB, FairyTailMod.MOD_ID);

    public static final DeferredHolder<CreativeModeTab, CreativeModeTab> KEYS = TABS.register("zodiac_keys", () ->
            CreativeModeTab.builder()
                    .title(Component.translatable("itemGroup.fairytail.zodiac_keys"))
                    .icon(() -> new ItemStack(ModItems.KEYS.get(ZodiacKey.AQUARIUS).get()))
                    .displayItems((params, out) -> ModItems.KEYS.values().forEach(h -> out.accept(h.get())))
                    .build());

    private ModCreativeTabs() {}
}
