package dev.fairytail.registry;

import dev.fairytail.FairyTailMod;
import dev.fairytail.key.CelestialKeyItem;
import dev.fairytail.key.ZodiacKey;
import net.minecraft.world.item.Item;
import net.neoforged.neoforge.registries.DeferredHolder;
import net.neoforged.neoforge.registries.DeferredRegister;

import java.util.EnumMap;
import java.util.Map;

public final class ModItems {
    public static final DeferredRegister.Items ITEMS = DeferredRegister.createItems(FairyTailMod.MOD_ID);
    public static final Map<ZodiacKey, DeferredHolder<Item, CelestialKeyItem>> KEYS = new EnumMap<>(ZodiacKey.class);

    static {
        for (ZodiacKey key : ZodiacKey.values()) {
            KEYS.put(key, ITEMS.register(key.itemName(), () -> new CelestialKeyItem(key)));
        }
    }

    private ModItems() {}
}
