package dev.fairytail.mixin;

import dev.fairytail.client.ClientCast;
import dev.fairytail.client.ClientCastManager;
import net.minecraft.client.renderer.entity.layers.ItemInHandLayer;
import net.minecraft.world.entity.HumanoidArm;
import net.minecraft.world.entity.LivingEntity;
import net.minecraft.world.item.ItemStack;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;
import com.mojang.blaze3d.vertex.PoseStack;
import net.minecraft.client.renderer.MultiBufferSource;
import net.minecraft.world.item.ItemDisplayContext;

/** While casting, the key is drawn by CastKeyLayer from the real forearm transform instead. */
@Mixin(ItemInHandLayer.class)
public abstract class ItemInHandLayerMixin {
    @Inject(method = "renderArmWithItem", at = @At("HEAD"), cancellable = true)
    private void fairytail$skipWhileCasting(LivingEntity entity, ItemStack stack, ItemDisplayContext ctx, HumanoidArm arm,
                                            PoseStack poseStack, MultiBufferSource buffer, int light, CallbackInfo ci) {
        ClientCast cast = ClientCastManager.get(entity.getId());
        if (cast != null && (arm == HumanoidArm.LEFT) == cast.leftArm && stack.getItem() instanceof dev.fairytail.key.CelestialKeyItem) {
            ci.cancel();
        }
    }
}
