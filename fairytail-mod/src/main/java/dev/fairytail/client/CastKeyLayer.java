package dev.fairytail.client;

import com.mojang.blaze3d.vertex.PoseStack;
import com.mojang.math.Axis;
import dev.fairytail.FairyTailMod;
import dev.fairytail.cast.CastTimeline;
import dev.fairytail.cast.PoseTracks;
import dev.fairytail.client.model.ElbowModel;
import dev.fairytail.registry.ModItems;
import net.minecraft.client.Minecraft;
import net.minecraft.util.Mth;
import net.minecraft.client.model.PlayerModel;
import net.minecraft.client.model.geom.ModelPart;
import net.minecraft.client.player.AbstractClientPlayer;
import net.minecraft.client.renderer.LightTexture;
import net.minecraft.client.renderer.MultiBufferSource;
import net.minecraft.client.renderer.entity.RenderLayerParent;
import net.minecraft.client.renderer.entity.layers.RenderLayer;
import net.minecraft.client.renderer.texture.OverlayTexture;
import net.minecraft.client.resources.model.BakedModel;
import net.minecraft.client.resources.model.ModelResourceLocation;
import net.minecraft.world.item.ItemDisplayContext;
import net.minecraft.world.item.ItemStack;
import org.joml.Matrix4f;
import org.joml.Vector3f;
import net.minecraft.world.phys.Vec3;

/**
 * Draws the key during a cast, parented to the real forearm transform (so it follows the elbow, wrist
 * flex and twist exactly), and records the key tip's world position for particles, camera and gate.
 */
public class CastKeyLayer extends RenderLayer<AbstractClientPlayer, PlayerModel<AbstractClientPlayer>> {
    /** Key size in blocks at scale 1.0 (the model itself is 1 block tall). */
    private static final float KEY_SCALE = 0.5f;
    /** Model-space Y (in 1/16 block) of the grip, and of the bit's tip (the "keyhole" point). */
    private static final float GRIP_Y = 14.2f;
    private static final float TIP_Y = 0.4f;
    private static final float TIP_OFFSET = (TIP_Y - GRIP_Y) / 16f;

    public CastKeyLayer(RenderLayerParent<AbstractClientPlayer, PlayerModel<AbstractClientPlayer>> parent) {
        super(parent);
    }

    public static ModelResourceLocation handModel(dev.fairytail.key.ZodiacKey key) {
        return ModelResourceLocation.standalone(FairyTailMod.id("item/hand/" + key.itemName()));
    }

    @Override
    public void render(PoseStack ps, MultiBufferSource buffer, int light, AbstractClientPlayer p, float limbSwing,
                       float limbSwingAmount, float partial, float ageInTicks, float netHeadYaw, float headPitch) {
        ClientCast c = ClientCastManager.get(p.getId());
        if (c == null) return;
        float t = ageInTicks - c.startTick;
        PlayerModel<AbstractClientPlayer> m = getParentModel();
        if (!(m instanceof ElbowModel elbow)) return;
        ModelPart arm = c.leftArm ? m.leftArm : m.rightArm;
        ModelPart fore = elbow.fairytail$forearm(c.leftArm);
        if (fore == null) return;

        float w = CastTimeline.poseWeight(t);

        ps.pushPose();
        arm.translateAndRotate(ps);
        fore.translateAndRotate(ps);
        ps.translate(0f, 0.40f, 0f); // just past the end of the hand (6 px = 0.375), so the bow sits in the fingers, not inside the arm

        // The key is rigid with the forearm: its bit (model -Y) points straight out of the fist along the
        // forearm, whatever the arm is doing. Measured in-game: -90 deg about X turns the model's +Y (head)
        // back toward the wrist, so -Y (the bit) leads out of the hand.
        ps.mulPose(Axis.XP.rotation(-Mth.HALF_PI + PoseTracks.KEY_FLEX.eval(t)));
        ps.mulPose(Axis.XP.rotationDegrees(-90f));
        ps.mulPose(Axis.YP.rotationDegrees(180f));
        float side = c.leftArm ? -1f : 1f;
        ps.mulPose(Axis.YP.rotation(PoseTracks.KEY_TWIST.eval(t) * side)); // turn of the key about its own axis
        float k = KEY_SCALE * PoseTracks.KEY_SCALE.eval(t);
        ps.scale(k, k, k);

        recordAnchor(c, ps);

        ps.translate(0f, (8f - GRIP_Y) / 16f, 0f); // the key is pinched by its bow: only the last ~2 px of the head is behind the fingers
        Minecraft mc = Minecraft.getInstance();
        BakedModel model = mc.getModelManager().getModel(handModel(c.key));
        if (model != mc.getModelManager().getMissingModel()) {
            ItemStack stack = new ItemStack(ModItems.KEYS.get(c.key).get());
            int lit = t >= CastTimeline.RAISE ? LightTexture.pack(15, 15) : light;
            mc.getItemRenderer().render(stack, ItemDisplayContext.NONE, false, ps, buffer, lit, OverlayTexture.NO_OVERLAY, model);
        }
        ps.popPose();
    }

    private static void recordAnchor(ClientCast c, PoseStack ps) {
        Matrix4f pose = ps.last().pose();
        Vector3f head = pose.transformPosition(0f, TIP_OFFSET, 0f, new Vector3f());
        Vector3f axis = pose.transformDirection(0f, 1f, 0f, new Vector3f()).normalize();
        Vec3 cam = Minecraft.getInstance().gameRenderer.getMainCamera().getPosition();
        c.setAnchor(new Vec3(cam.x + head.x, cam.y + head.y, cam.z + head.z), new Vec3(axis.x, axis.y, axis.z));
    }
}
