package dev.fairytail.client;

import dev.fairytail.cast.CastTimeline;
import dev.fairytail.client.fx.CastEffects;
import dev.fairytail.key.ZodiacKey;
import net.minecraft.client.Minecraft;
import net.minecraft.client.player.AbstractClientPlayer;
import net.minecraft.client.player.LocalPlayer;
import net.minecraft.world.entity.Entity;

import java.util.HashMap;
import java.util.Iterator;
import java.util.Map;

/**
 * Owns every active client-side cast and is the single place that ends one. Anything that can go wrong
 * (death, dimension change, disconnect, entity unloaded, lost packet) funnels into {@link #end} /
 * {@link #clear}, and because the camera, pose and overlay are all *derived* from the cast's presence
 * (nothing is stored on the player or the options), removing the cast fully restores gameplay.
 */
public final class ClientCastManager {
    private static final Map<Integer, ClientCast> ACTIVE = new HashMap<>();
    /** Casts announced for an entity this client has not finished spawning yet: id -> {key, left, elapsed, ttl}. */
    private static final Map<Integer, int[]> PENDING = new HashMap<>();

    public static void start(int entityId, int keyIndex, boolean left, int elapsed) {
        Minecraft mc = Minecraft.getInstance();
        if (mc.level == null) return;
        Entity e = mc.level.getEntity(entityId);
        if (!(e instanceof AbstractClientPlayer p)) {
            PENDING.put(entityId, new int[]{keyIndex, left ? 1 : 0, elapsed, 60});
            return;
        }
        boolean local = p == mc.player;
        float facing = local ? p.getYRot() : p.yBodyRot;
        ACTIVE.put(entityId, new ClientCast(entityId, ZodiacKey.byIndex(keyIndex), left, p.tickCount - elapsed, local, facing));
        if (local) {
            p.yBodyRot = p.yBodyRotO = facing;
        }
    }

    public static void end(int entityId, boolean interrupted) {
        PENDING.remove(entityId);
        ClientCast c = ACTIVE.remove(entityId);
        if (c != null && c.local) CinematicOverlay.reset();
    }

    public static void clear() {
        ACTIVE.clear();
        PENDING.clear();
        CinematicOverlay.reset();
    }

    public static ClientCast get(int entityId) {
        return ACTIVE.isEmpty() ? null : ACTIVE.get(entityId);
    }

    /** The local player's cast, or null. Cheap: called every frame by camera/overlay/input hooks. */
    public static ClientCast local() {
        Minecraft mc = Minecraft.getInstance();
        if (ACTIVE.isEmpty() || mc.player == null) return null;
        ClientCast c = ACTIVE.get(mc.player.getId());
        return c != null && c.local ? c : null;
    }

    public static Iterable<ClientCast> all() {
        return ACTIVE.values();
    }

    public static void tick() {
        Minecraft mc = Minecraft.getInstance();
        if (mc.level == null || mc.player == null) {
            if (!ACTIVE.isEmpty() || !PENDING.isEmpty()) clear();
            return;
        }
        if (!PENDING.isEmpty()) {
            Iterator<Map.Entry<Integer, int[]>> pit = PENDING.entrySet().iterator();
            Map<Integer, int[]> ready = new HashMap<>();
            while (pit.hasNext()) {
                Map.Entry<Integer, int[]> en = pit.next();
                int[] v = en.getValue();
                if (mc.level.getEntity(en.getKey()) instanceof AbstractClientPlayer) {
                    ready.put(en.getKey(), v);
                    pit.remove();
                } else if (--v[3] <= 0) {
                    pit.remove();
                }
            }
            ready.forEach((id, v) -> start(id, v[0], v[1] == 1, v[2]));
        }
        Iterator<ClientCast> it = ACTIVE.values().iterator();
        while (it.hasNext()) {
            ClientCast c = it.next();
            Entity e = mc.level.getEntity(c.entityId);
            if (!(e instanceof AbstractClientPlayer p) || !p.isAlive() || p.isSpectator()) {
                it.remove();
                if (c.local) CinematicOverlay.reset();
                continue;
            }
            int t = p.tickCount - c.startTick;
            if (t > CastTimeline.TOTAL + 3) { // safety net if the end packet was lost
                it.remove();
                if (c.local) CinematicOverlay.reset();
                continue;
            }
            if (c.local && p instanceof LocalPlayer) {
                p.yBodyRot = p.yBodyRotO = c.facingYaw;
            }
            for (int k = Math.max(0, c.lastProcessedTick + 1); k <= t; k++) {
                CastEffects.tick(c, p, k);
            }
            c.lastProcessedTick = t;
        }
    }

    private ClientCastManager() {}
}
