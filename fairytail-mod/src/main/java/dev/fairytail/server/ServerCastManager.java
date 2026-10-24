package dev.fairytail.server;

import dev.fairytail.key.ZodiacKey;
import dev.fairytail.net.CastEndPayload;
import dev.fairytail.net.CastStartPayload;
import net.minecraft.server.level.ServerPlayer;
import net.minecraft.world.InteractionHand;
import net.minecraft.world.entity.HumanoidArm;
import net.neoforged.bus.api.SubscribeEvent;
import net.neoforged.neoforge.event.entity.living.LivingDeathEvent;
import net.neoforged.neoforge.event.entity.player.PlayerEvent;
import net.neoforged.neoforge.event.tick.ServerTickEvent;
import net.neoforged.neoforge.network.PacketDistributor;

import java.util.Iterator;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Authoritative side of a Gate Opening cast. The server decides whether a cast may start, tells every
 * client that can see the player to play the animation, and ends it (normally or on interruption).
 * The camera/HUD/particles/sound are purely client-side presentation driven by these two packets.
 */
public final class ServerCastManager {
    /** Cast length in ticks. Must match {@code CastTimeline.TOTAL} on the client (96 ticks = 4.8 s). */
    public static final int CAST_TICKS = 96;

    private record Cast(ZodiacKey key, boolean leftArm, long startTick, int entityId) {}

    private static final Map<UUID, Cast> ACTIVE = new ConcurrentHashMap<>();

    /** @return true if the cast started */
    public static boolean tryStart(ServerPlayer player, ZodiacKey key, InteractionHand hand) {
        if (ACTIVE.containsKey(player.getUUID())) return false;
        if (player.isSpectator() || player.isDeadOrDying() || player.isSleeping() || player.isPassenger()) return false;
        HumanoidArm arm = hand == InteractionHand.MAIN_HAND ? player.getMainArm() : player.getMainArm().getOpposite();
        boolean left = arm == HumanoidArm.LEFT;
        long now = player.serverLevel().getGameTime();
        ACTIVE.put(player.getUUID(), new Cast(key, left, now, player.getId()));
        PacketDistributor.sendToPlayersTrackingEntityAndSelf(player, new CastStartPayload(player.getId(), key.ordinal(), left, 0));
        return true;
    }

    public static boolean isCasting(ServerPlayer player) {
        return ACTIVE.containsKey(player.getUUID());
    }

    private static void finish(ServerPlayer player, boolean interrupted) {
        Cast cast = ACTIVE.remove(player.getUUID());
        if (cast != null) {
            PacketDistributor.sendToPlayersTrackingEntityAndSelf(player, new CastEndPayload(cast.entityId(), interrupted));
        }
    }

    @SubscribeEvent
    public static void onServerTick(ServerTickEvent.Post event) {
        if (ACTIVE.isEmpty()) return;
        Iterator<Map.Entry<UUID, Cast>> it = ACTIVE.entrySet().iterator();
        while (it.hasNext()) {
            Map.Entry<UUID, Cast> e = it.next();
            ServerPlayer p = event.getServer().getPlayerList().getPlayer(e.getKey());
            if (p == null || p.isRemoved()) {
                it.remove();
            } else if (p.isDeadOrDying() || p.isSpectator()) {
                finish(p, true);
            } else if (p.serverLevel().getGameTime() - e.getValue().startTick() >= CAST_TICKS + 2) {
                finish(p, false);
            }
        }
    }

    @SubscribeEvent
    public static void onDeath(LivingDeathEvent event) {
        if (event.getEntity() instanceof ServerPlayer p) finish(p, true);
    }

    @SubscribeEvent
    public static void onDimensionChange(PlayerEvent.PlayerChangedDimensionEvent event) {
        if (event.getEntity() instanceof ServerPlayer p) finish(p, true);
    }

    @SubscribeEvent
    public static void onLogout(PlayerEvent.PlayerLoggedOutEvent event) {
        if (event.getEntity() instanceof ServerPlayer p) finish(p, true);
    }

    /** A player that walks into view mid-cast joins the animation at the right moment. */
    @SubscribeEvent
    public static void onStartTracking(PlayerEvent.StartTracking event) {
        if (event.getTarget() instanceof ServerPlayer target && event.getEntity() instanceof ServerPlayer viewer) {
            Cast cast = ACTIVE.get(target.getUUID());
            if (cast != null) {
                int elapsed = (int) (target.serverLevel().getGameTime() - cast.startTick());
                PacketDistributor.sendToPlayer(viewer, new CastStartPayload(target.getId(), cast.key().ordinal(), cast.leftArm(), elapsed));
            }
        }
    }

    private ServerCastManager() {}
}
