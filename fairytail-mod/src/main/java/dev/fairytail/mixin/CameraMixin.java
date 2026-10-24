package dev.fairytail.mixin;

import dev.fairytail.client.CinematicCamera;
import net.minecraft.client.Camera;
import net.minecraft.world.entity.Entity;
import net.minecraft.world.level.BlockGetter;
import net.minecraft.world.phys.Vec3;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.Shadow;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;

@Mixin(Camera.class)
public abstract class CameraMixin {
    @Shadow protected abstract void setRotation(float yaw, float pitch, float roll);

    @Shadow protected abstract void setPosition(Vec3 pos);

    @Shadow private boolean detached;

    @Shadow public abstract Vec3 getPosition();

    @Shadow public abstract float getYRot();

    @Shadow public abstract float getXRot();

    /** Runs after vanilla (and any other mod) has placed the camera; the cinematic blends over that result. */
    @Inject(method = "setup", at = @At("TAIL"))
    private void fairytail$cinematic(BlockGetter level, Entity entity, boolean detachedIn, boolean mirrored, float partial, CallbackInfo ci) {
        CinematicCamera.Result r = CinematicCamera.compute(level, entity, partial, getPosition(), getYRot(), getXRot());
        if (r != null) {
            setPosition(r.pos());
            setRotation(r.yaw(), r.pitch(), r.roll());
            if (r.weight() > 0.12f) this.detached = true; // render the player's own model
        }
    }
}
