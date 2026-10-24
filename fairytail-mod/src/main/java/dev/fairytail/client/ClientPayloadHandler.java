package dev.fairytail.client;

import dev.fairytail.net.CastEndPayload;
import dev.fairytail.net.CastStartPayload;
import net.neoforged.neoforge.network.handling.IPayloadContext;

/** Kept in its own class so the dedicated server never loads client classes through the registrar. */
public final class ClientPayloadHandler {
    public static void onStart(CastStartPayload p, IPayloadContext ctx) {
        ClientCastManager.start(p.entityId(), p.keyIndex(), p.leftArm(), p.elapsed());
    }

    public static void onEnd(CastEndPayload p, IPayloadContext ctx) {
        ClientCastManager.end(p.entityId(), p.interrupted());
    }

    private ClientPayloadHandler() {}
}
