package dev.fairytail.key;

import dev.fairytail.registry.ModItems;
import dev.fairytail.server.ServerCastManager;
import net.minecraft.ChatFormatting;
import net.minecraft.network.chat.Component;
import net.minecraft.server.level.ServerPlayer;
import net.minecraft.world.InteractionHand;
import net.minecraft.world.InteractionResultHolder;
import net.minecraft.world.entity.player.Player;
import net.minecraft.world.item.Item;
import net.minecraft.world.item.ItemStack;
import net.minecraft.world.item.Rarity;
import net.minecraft.world.item.TooltipFlag;
import net.minecraft.world.item.UseAnim;
import net.minecraft.world.level.Level;

import java.util.List;

/**
 * One of Lucy's twelve Golden Zodiac Keys. Right-clicking asks the server to start the Gate Opening
 * cast; the server validates it and tells clients to play the animation / cinematic.
 * Phase 1 deliberately summons nothing.
 */
public class CelestialKeyItem extends Item {
    /** Total lock-out after a cast, in ticks (cast length + a short breather). */
    public static final int COOLDOWN_TICKS = ServerCastManager.CAST_TICKS + 12;

    private final ZodiacKey key;

    public CelestialKeyItem(ZodiacKey key) {
        super(new Item.Properties().stacksTo(1).rarity(Rarity.EPIC).fireResistant());
        this.key = key;
    }

    public ZodiacKey zodiac() {
        return key;
    }

    @Override
    public InteractionResultHolder<ItemStack> use(Level level, Player player, InteractionHand hand) {
        ItemStack stack = player.getItemInHand(hand);
        if (!level.isClientSide && player instanceof ServerPlayer sp && ServerCastManager.tryStart(sp, key, hand)) {
            ModItems.KEYS.values().forEach(h -> sp.getCooldowns().addCooldown(h.get(), COOLDOWN_TICKS));
        }
        return InteractionResultHolder.consume(stack);
    }

    @Override
    public UseAnim getUseAnimation(ItemStack stack) {
        return UseAnim.NONE;
    }

    @Override
    public void appendHoverText(ItemStack stack, TooltipContext context, List<Component> tooltip, TooltipFlag flag) {
        tooltip.add(Component.literal(key.glyph + " Golden Zodiac Key").withStyle(ChatFormatting.GOLD));
        tooltip.add(Component.literal("Gate of the " + key.gateTitle).withStyle(ChatFormatting.YELLOW));
        tooltip.add(Component.literal(key.epithet).withStyle(ChatFormatting.GRAY, ChatFormatting.ITALIC));
        tooltip.add(Component.translatable("tooltip.fairytail.key_use").withStyle(ChatFormatting.DARK_GRAY));
    }
}
