package dev.fairytail.net;

import dev.fairytail.FairyTailMod;
import io.netty.buffer.ByteBuf;
import net.minecraft.network.codec.ByteBufCodecs;
import net.minecraft.network.codec.StreamCodec;
import net.minecraft.network.protocol.common.custom.CustomPacketPayload;

/**
 * Server -> client: an entity began a Gate Opening cast.
 *
 * @param elapsed ticks already played (non-zero when a client starts tracking a player mid-cast)
 * @param leftArm which arm holds the key
 */
public record CastStartPayload(int entityId, int keyIndex, boolean leftArm, int elapsed) implements CustomPacketPayload {
    public static final Type<CastStartPayload> TYPE = new Type<>(FairyTailMod.id("cast_start"));
    public static final StreamCodec<ByteBuf, CastStartPayload> CODEC = StreamCodec.composite(
            ByteBufCodecs.VAR_INT, CastStartPayload::entityId,
            ByteBufCodecs.VAR_INT, CastStartPayload::keyIndex,
            ByteBufCodecs.BOOL, CastStartPayload::leftArm,
            ByteBufCodecs.VAR_INT, CastStartPayload::elapsed,
            CastStartPayload::new);

    @Override
    public Type<? extends CustomPacketPayload> type() {
        return TYPE;
    }
}
