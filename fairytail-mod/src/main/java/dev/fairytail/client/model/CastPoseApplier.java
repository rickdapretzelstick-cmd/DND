package dev.fairytail.client.model;

import dev.fairytail.cast.CastTimeline;
import dev.fairytail.cast.PoseTracks;
import dev.fairytail.client.ClientCast;
import net.minecraft.client.model.PlayerModel;
import net.minecraft.client.model.geom.ModelPart;
import net.minecraft.util.Mth;
import net.minecraft.world.entity.LivingEntity;

/**
 * Drives the player model from {@link PoseTracks}. Runs at the tail of PlayerModel.setupAnim, i.e. after
 * vanilla has posed the model, and blends toward the cast pose by {@link CastTimeline#poseWeight} so
 * entry and exit are seamless whatever the player was doing (walking, swimming...).
 */
public final class CastPoseApplier {

    public static void apply(PlayerModel<? extends LivingEntity> m, ElbowModel elbow, ClientCast cast, float t) {
        float w = CastTimeline.poseWeight(t);
        if (w <= 0f) return;
        boolean left = cast.leftArm;
        float s = left ? -1f : 1f; // mirror factor: tracks are authored for the right arm
        ModelPart arm = left ? m.leftArm : m.rightArm;
        ModelPart off = left ? m.rightArm : m.leftArm;
        ModelPart fore = elbow.fairytail$forearm(left);
        ModelPart foreSleeve = elbow.fairytail$forearmSleeve(left);
        float settle = PoseTracks.settle(t);

        // torso twist; shoulders are not children of the body in the vanilla rig, so move them around it
        float torso = PoseTracks.TORSO_YAW.eval(t) * s;
        m.body.yRot = Mth.lerp(w, m.body.yRot, torso);
        float sin = Mth.sin(m.body.yRot), cos = Mth.cos(m.body.yRot);
        m.rightArm.z = Mth.lerp(w, m.rightArm.z, sin * 5f);
        m.rightArm.x = Mth.lerp(w, m.rightArm.x, -cos * 5f);
        m.leftArm.z = Mth.lerp(w, m.leftArm.z, -sin * 5f);
        m.leftArm.x = Mth.lerp(w, m.leftArm.x, cos * 5f);

        // casting arm: shoulder, upper arm, elbow
        arm.xRot = Mth.lerp(w, arm.xRot, PoseTracks.SHOULDER_X.eval(t) + settle);
        arm.yRot = Mth.lerp(w, arm.yRot, PoseTracks.SHOULDER_YAW.eval(t) * s + m.body.yRot);
        arm.zRot = Mth.lerp(w, arm.zRot, PoseTracks.SHOULDER_ROLL.eval(t) * s);
        arm.y = Mth.lerp(w, arm.y, 2f - PoseTracks.SHRUG.eval(t));
        float elbowAngle = (PoseTracks.ELBOW.eval(t) - settle * 0.8f) * w;
        fore.xRot = elbowAngle;
        foreSleeve.xRot = elbowAngle;

        // balancing arm
        off.xRot = Mth.lerp(w, off.xRot, PoseTracks.OFF_ARM_X.eval(t));
        off.yRot = Mth.lerp(w, off.yRot, m.body.yRot);
        off.zRot = Mth.lerp(w, off.zRot, -PoseTracks.OFF_ARM_ROLL.eval(t) * s);

        // head keeps facing the viewer while the torso twists
        m.head.xRot = Mth.lerp(w, m.head.xRot, PoseTracks.HEAD_PITCH.eval(t));
        m.head.yRot = Mth.lerp(w, m.head.yRot, -torso * 0.9f + PoseTracks.HEAD_YAW_BIAS.eval(t) * s);

        // stance
        ModelPart front = left ? m.leftLeg : m.rightLeg;
        ModelPart back = left ? m.rightLeg : m.leftLeg;
        front.xRot = Mth.lerp(w, front.xRot, PoseTracks.LEG_FRONT.eval(t));
        back.xRot = Mth.lerp(w, back.xRot, PoseTracks.LEG_BACK.eval(t));
        front.zRot = Mth.lerp(w, front.zRot, 0.04f * s);
        back.zRot = Mth.lerp(w, back.zRot, -0.05f * s);

        syncOverlays(m, elbow);
    }

    /** Re-copy the 2nd-skin-layer parts, which vanilla copied *before* we changed the base parts. */
    public static void syncOverlays(PlayerModel<? extends LivingEntity> m, ElbowModel elbow) {
        m.hat.copyFrom(m.head);
        m.jacket.copyFrom(m.body);
        m.leftPants.copyFrom(m.leftLeg);
        m.rightPants.copyFrom(m.rightLeg);
        m.leftSleeve.copyFrom(m.leftArm);
        m.rightSleeve.copyFrom(m.rightArm);
    }

    /** The player model instance is shared between players, so forearms must be reset every frame. */
    public static void resetElbows(ElbowModel elbow) {
        for (boolean left : new boolean[]{false, true}) {
            ModelPart f = elbow.fairytail$forearm(left);
            ModelPart fs = elbow.fairytail$forearmSleeve(left);
            if (f != null) f.xRot = 0f;
            if (fs != null) fs.xRot = 0f;
        }
    }

    private CastPoseApplier() {}
}
