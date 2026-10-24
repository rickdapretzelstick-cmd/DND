package dev.fairytail.client;

import dev.fairytail.key.ZodiacKey;
import net.minecraft.world.entity.Entity;
import net.minecraft.world.phys.Vec3;

/** Client-side state of one in-progress cast (one per casting player, local or remote). */
public final class ClientCast {
    public final int entityId;
    public final ZodiacKey key;
    public final boolean leftArm;
    /** The caster's {@code tickCount} at which timeline t == 0. */
    public final int startTick;
    public final boolean local;
    /** Yaw (degrees) the cinematic treats as "front"; locked at cast start for the local player. */
    public float facingYaw;

    // key head position / axis in world space, captured from the real hand transform each frame
    public Vec3 anchor = Vec3.ZERO;
    public Vec3 prevAnchor = Vec3.ZERO;
    public Vec3 axis = new Vec3(0, 1, 0);
    public boolean hasAnchor;

    int lastProcessedTick = -1;

    ClientCast(int entityId, ZodiacKey key, boolean leftArm, int startTick, boolean local, float facingYaw) {
        this.entityId = entityId;
        this.key = key;
        this.leftArm = leftArm;
        this.startTick = startTick;
        this.local = local;
        this.facingYaw = facingYaw;
    }

    /** Timeline position in ticks including the sub-tick fraction. */
    public float time(Entity e, float partialTick) {
        return e.tickCount - startTick + partialTick;
    }

    public void setAnchor(Vec3 head, Vec3 axisDir) {
        this.prevAnchor = hasAnchor ? this.anchor : head;
        this.anchor = head;
        this.axis = axisDir;
        this.hasAnchor = true;
    }
}
