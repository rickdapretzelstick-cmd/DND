package dev.fairytail.mixin;

import dev.fairytail.client.ClientCast;
import dev.fairytail.client.ClientCastManager;
import dev.fairytail.client.model.ArmSplitter;
import dev.fairytail.client.model.CastPoseApplier;
import dev.fairytail.client.model.ElbowModel;
import net.minecraft.client.model.PlayerModel;
import net.minecraft.client.model.geom.ModelPart;
import net.minecraft.client.player.AbstractClientPlayer;
import net.minecraft.world.entity.LivingEntity;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.Unique;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;

@Mixin(PlayerModel.class)
public abstract class PlayerModelMixin implements ElbowModel {
    @Unique private ModelPart fairytail$forearmR, fairytail$forearmL, fairytail$sleeveR, fairytail$sleeveL;

    @Inject(method = "<init>", at = @At("TAIL"))
    private void fairytail$splitArms(ModelPart root, boolean slim, CallbackInfo ci) {
        PlayerModel<?> m = (PlayerModel<?>) (Object) this;
        float w = slim ? 3f : 4f;
        fairytail$forearmR = ArmSplitter.split(m.rightArm, 40, 16, slim ? -2f : -3f, w, 0f);
        fairytail$sleeveR = ArmSplitter.split(m.rightSleeve, 40, 32, slim ? -2f : -3f, w, 0.25f);
        fairytail$forearmL = ArmSplitter.split(m.leftArm, 32, 48, -1f, w, 0f);
        fairytail$sleeveL = ArmSplitter.split(m.leftSleeve, 48, 48, -1f, w, 0.25f);
    }

    @Inject(method = "setupAnim(Lnet/minecraft/world/entity/LivingEntity;FFFFF)V", at = @At("TAIL"))
    private void fairytail$castPose(LivingEntity entity, float limbSwing, float limbSwingAmount, float ageInTicks,
                                    float netHeadYaw, float headPitch, CallbackInfo ci) {
        CastPoseApplier.resetElbows(this);
        if (entity instanceof AbstractClientPlayer p) {
            ClientCast cast = ClientCastManager.get(p.getId());
            if (cast != null) {
                @SuppressWarnings("unchecked")
                PlayerModel<LivingEntity> self = (PlayerModel<LivingEntity>) (Object) this;
                CastPoseApplier.apply(self, this, cast, ageInTicks - cast.startTick);
            }
        }
    }

    @Override
    public ModelPart fairytail$forearm(boolean left) {
        return left ? fairytail$forearmL : fairytail$forearmR;
    }

    @Override
    public ModelPart fairytail$forearmSleeve(boolean left) {
        return left ? fairytail$sleeveL : fairytail$sleeveR;
    }
}
