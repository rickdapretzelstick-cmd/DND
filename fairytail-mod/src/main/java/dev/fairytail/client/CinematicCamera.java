package dev.fairytail.client;

import dev.fairytail.cast.CastTimeline;
import dev.fairytail.cast.ShotRig;
import net.minecraft.client.Minecraft;
import net.minecraft.util.Mth;
import net.minecraft.world.entity.Entity;
import net.minecraft.world.level.BlockGetter;
import net.minecraft.world.level.ClipContext;
import net.minecraft.world.phys.HitResult;
import net.minecraft.world.phys.Vec3;

/**
 * Computes the cinematic camera for the local player's cast from {@link ShotRig}. It never touches
 * {@code Options} or the player's rotation: it is evaluated every frame from the cast state and
 * blended over the vanilla camera by {@link CastTimeline#cameraWeight}, so when the cast ends the
 * vanilla camera simply is the camera again.
 */
public final class CinematicCamera {
    public record Result(Vec3 pos, float yaw, float pitch, float roll, float weight) {}

    /** @return null when no cinematic should be applied to this camera setup */
    public static Result compute(BlockGetter level, Entity entity, float partial, Vec3 origPos, float origYaw, float origPitch) {
        ClientCast c = ClientCastManager.local();
        if (c == null || entity != Minecraft.getInstance().player) return null;
        float t = c.time(entity, partial);
        float w = CastTimeline.cameraWeight(t);
        if (w <= 0.001f) return null;

        float s = c.leftArm ? -1f : 1f;
        Vec3 pivot = entity.getPosition(partial).add(0, entity.getBbHeight() * 0.72, 0);
        double yaw = Math.toRadians(c.facingYaw);
        Vec3 fwd = new Vec3(-Math.sin(yaw), 0, Math.cos(yaw));
        Vec3 right = new Vec3(-Math.cos(yaw), 0, -Math.sin(yaw)); // the caster's own right-hand side
        Vec3 up = new Vec3(0, 1, 0);

        Vec3 desired = pivot
                .add(fwd.scale(ShotRig.CAM_F.eval(t)))
                .add(right.scale(ShotRig.CAM_R.eval(t) * s))
                .add(up.scale(ShotRig.CAM_U.eval(t)));

        float shake = ShotRig.SHAKE.eval(t);
        if (shake > 0) {
            desired = desired.add(Math.sin(t * 37.1) * shake, Math.sin(t * 29.7 + 2) * shake, Math.sin(t * 41.3 + 4) * shake);
        }

        // keep the lens out of walls: pull in along pivot -> camera if something solid is in the way
        HitResult hit = level.clip(new ClipContext(pivot, desired, ClipContext.Block.COLLIDER, ClipContext.Fluid.NONE, entity));
        if (hit.getType() == HitResult.Type.BLOCK) {
            Vec3 dir = desired.subtract(pivot);
            double len = dir.length();
            double safe = Math.max(0.55, pivot.distanceTo(hit.getLocation()) - 0.3);
            if (len > 1e-4) desired = pivot.add(dir.scale(Math.min(1.0, safe / len)));
        }

        Vec3 faceTarget = pivot.add(up.scale(ShotRig.TGT_U.eval(t)));
        Vec3 keyTarget = c.hasAnchor ? c.anchor : pivot.add(fwd.scale(0.7)).add(up.scale(0.3));
        Vec3 look = faceTarget.lerp(keyTarget, ShotRig.KEY_FOCUS.eval(t));

        Vec3 d = look.subtract(desired);
        double horiz = Math.sqrt(d.x * d.x + d.z * d.z);
        float tYaw = (float) (Mth.atan2(-d.x, d.z) * Mth.RAD_TO_DEG);
        float tPitch = (float) (-Mth.atan2(d.y, horiz) * Mth.RAD_TO_DEG);
        if (shake > 0) {
            tYaw += (float) Math.sin(t * 33.3) * shake * 14f;
            tPitch += (float) Math.sin(t * 27.9 + 1) * shake * 14f;
        }

        // sweep wide around the character during the swing-round (and back out again)
        Vec3 arc = right.scale(s * 1.4 * Math.sin(Math.PI * w));
        Vec3 pos = origPos.lerp(desired, w).add(arc);
        float yawOut = Mth.rotLerp(w, origYaw, tYaw);
        float pitchOut = Mth.lerp(w, origPitch, tPitch);
        float roll = ShotRig.ROLL.eval(t) * w;
        return new Result(pos, yawOut, pitchOut, roll, w);
    }

    /** Field of view for the current frame (called from ViewportEvent.ComputeFov). */
    public static double fov(double baseFov, float partial) {
        ClientCast c = ClientCastManager.local();
        Minecraft mc = Minecraft.getInstance();
        if (c == null || mc.player == null) return baseFov;
        float t = c.time(mc.player, partial);
        float w = CastTimeline.cameraWeight(t);
        return w <= 0 ? baseFov : Mth.lerp(w, baseFov, ShotRig.FOV.eval(t));
    }

    /** True while the cinematic owns the screen (HUD hidden, input locked). */
    public static boolean active() {
        ClientCast c = ClientCastManager.local();
        Minecraft mc = Minecraft.getInstance();
        return c != null && mc.player != null && CastTimeline.cameraWeight(c.time(mc.player, 0)) > 0.001f;
    }

    private CinematicCamera() {}
}
