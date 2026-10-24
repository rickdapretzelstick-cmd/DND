package dev.fairytail.net;

import dev.fairytail.client.ClientPayloadHandler;
import net.neoforged.neoforge.network.event.RegisterPayloadHandlersEvent;
import net.neoforged.neoforge.network.registration.PayloadRegistrar;

public final class FairyTailNetwork {
    private static final String PROTOCOL = "1";

    public static void register(RegisterPayloadHandlersEvent event) {
        PayloadRegistrar registrar = event.registrar(PROTOCOL);
        registrar.playToClient(CastStartPayload.TYPE, CastStartPayload.CODEC, ClientPayloadHandler::onStart);
        registrar.playToClient(CastEndPayload.TYPE, CastEndPayload.CODEC, ClientPayloadHandler::onEnd);
    }

    private FairyTailNetwork() {}
}
