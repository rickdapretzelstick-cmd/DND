package dev.fairytail.client;

import com.mojang.blaze3d.vertex.PoseStack;
import net.minecraft.client.resources.model.BakedModel;
import net.minecraft.world.item.ItemDisplayContext;
import net.neoforged.neoforge.client.model.BakedModelWrapper;

/**
 * One item id, two looks: the flat hand-drawn-from-the-model icon in inventories (GUI), and the full 3D
 * key model everywhere else (held, dropped, item frames...).
 */
public class KeyBakedModel extends BakedModelWrapper<BakedModel> {
    private final BakedModel hand;

    public KeyBakedModel(BakedModel icon, BakedModel hand) {
        super(icon);
        this.hand = hand;
    }

    @Override
    public BakedModel applyTransform(ItemDisplayContext ctx, PoseStack poseStack, boolean leftHand) {
        if (ctx == ItemDisplayContext.GUI) {
            return super.applyTransform(ctx, poseStack, leftHand);
        }
        return hand.applyTransform(ctx, poseStack, leftHand);
    }
}
