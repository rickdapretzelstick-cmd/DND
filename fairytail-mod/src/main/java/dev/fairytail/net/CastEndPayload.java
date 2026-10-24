package dev.fairytail.net;

import dev.fairytail.FairyTailMod;
import io.netty.buffer.ByteBuf;
import net.minecraft.network.codec.ByteBufCodecs;
import net.minecraft.network.codec.StreamCodec;
import net.minecraft.network.protocol.common.custom.CustomPacketPayload;

/** Server -> client: a cast finished or was interrupted (death, dimension change, logout...). */
public record CastEndPayload(int entityId, boolean interrupted) implements CustomPacketPayload {
    public static final Type<CastEndPayload> TYPE = new Type<>(FairyTailMod.id("cast_end"));
    public static final StreamCodec<ByteBuf, CastEndPayload> CODEC = StreamCodec.composite(
            ByteBufCodecs.VAR_INT, CastEndPayload::entityId,
            ByteBufCodecs.BOOL, CastEndPayload::interrupted,
            CastEndPayload::new);

    @Override
    public Type<? extends CustomPacketPayload> type() {
        return TYPE;
    }
}
