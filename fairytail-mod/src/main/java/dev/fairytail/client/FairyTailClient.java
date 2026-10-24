package dev.fairytail.client;

import dev.fairytail.FairyTailMod;
import dev.fairytail.client.fx.ParticleProviders;
import dev.fairytail.key.ZodiacKey;
import net.minecraft.client.Minecraft;
import net.minecraft.client.model.PlayerModel;
import net.minecraft.client.player.AbstractClientPlayer;
import net.minecraft.client.renderer.entity.player.PlayerRenderer;
import net.minecraft.client.resources.PlayerSkin;
import net.minecraft.client.resources.model.BakedModel;
import net.minecraft.client.resources.model.ModelResourceLocation;
import net.neoforged.api.distmarker.Dist;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.fml.common.Mod;
import net.neoforged.neoforge.client.event.CalculatePlayerTurnEvent;
import net.neoforged.neoforge.client.event.ClientPlayerNetworkEvent;
import net.neoforged.neoforge.client.event.ClientTickEvent;
import net.neoforged.neoforge.client.event.EntityRenderersEvent;
import net.neoforged.neoforge.client.event.InputEvent;
import net.neoforged.neoforge.client.event.ModelEvent;
import net.neoforged.neoforge.client.event.MovementInputUpdateEvent;
import net.neoforged.neoforge.client.event.RegisterGuiLayersEvent;
import net.neoforged.neoforge.client.event.RegisterParticleProvidersEvent;
import net.neoforged.neoforge.client.event.RenderGuiLayerEvent;
import net.neoforged.neoforge.client.event.RenderHandEvent;
import net.neoforged.neoforge.client.event.RenderLevelStageEvent;
import net.neoforged.neoforge.client.event.ViewportEvent;
import net.neoforged.neoforge.client.gui.VanillaGuiLayers;
import net.neoforged.neoforge.common.NeoForge;

import java.util.Map;

/** Client-only wiring. Loaded only on the physical client (dist = CLIENT). */
@Mod(value = FairyTailMod.MOD_ID, dist = Dist.CLIENT)
public class FairyTailClient {
    public FairyTailClient(IEventBus modBus) {
        modBus.addListener(RegisterParticleProvidersEvent.class, ParticleProviders::register);
        modBus.addListener(ModelEvent.RegisterAdditional.class, this::registerHandModels);
        modBus.addListener(ModelEvent.ModifyBakingResult.class, this::swapKeyModels);
        modBus.addListener(EntityRenderersEvent.AddLayers.class, this::addLayers);
        modBus.addListener(RegisterGuiLayersEvent.class, e ->
                e.registerAboveAll(CinematicOverlay.LAYER_ID, CinematicOverlay::render));

        IEventBus bus = NeoForge.EVENT_BUS;
        bus.addListener(ClientTickEvent.Post.class, e -> {
            ClientCastManager.tick();
            AutoTest.tick();
        });
        bus.addListener(RenderLevelStageEvent.class, GateRenderer::render);
        bus.addListener(ViewportEvent.ComputeFov.class, e -> e.setFOV(CinematicCamera.fov(e.getFOV(), (float) e.getPartialTick())));
        bus.addListener(RenderHandEvent.class, e -> {
            if (CinematicCamera.active()) e.setCanceled(true);
        });
        bus.addListener(RenderGuiLayerEvent.Pre.class, e -> {
            if (CinematicCamera.active() && !CinematicOverlay.LAYER_ID.equals(e.getName())
                    && !VanillaGuiLayers.CHAT.equals(e.getName())) {
                e.setCanceled(true);
            }
        });
        // lock the player's controls for the duration of the cinematic
        bus.addListener(MovementInputUpdateEvent.class, e -> {
            if (ClientCastManager.local() != null) {
                var in = e.getInput();
                in.leftImpulse = 0;
                in.forwardImpulse = 0;
                in.up = in.down = in.left = in.right = false;
                in.jumping = false;
                in.shiftKeyDown = false;
            }
        });
        bus.addListener(InputEvent.InteractionKeyMappingTriggered.class, e -> {
            if (ClientCastManager.local() != null) {
                e.setCanceled(true);
                e.setSwingHand(false);
            }
        });
        bus.addListener(CalculatePlayerTurnEvent.class, e -> {
            if (ClientCastManager.local() != null) {
                e.setMouseSensitivity(-1.0 / 3.0); // (s * 0.6 + 0.2) == 0: no look input during the cinematic
                e.setCinematicCameraEnabled(false);
            }
        });
        // every way out of a world must drop any cast state
        bus.addListener(ClientPlayerNetworkEvent.LoggingOut.class, e -> ClientCastManager.clear());
        bus.addListener(ClientPlayerNetworkEvent.LoggingIn.class, e -> ClientCastManager.clear());
        bus.addListener(ClientPlayerNetworkEvent.Clone.class, e -> ClientCastManager.clear());
    }

    private void registerHandModels(ModelEvent.RegisterAdditional e) {
        for (ZodiacKey k : ZodiacKey.values()) e.register(CastKeyLayer.handModel(k));
    }

    /** Wrap each key's inventory model so non-GUI contexts render the 3D hand model. */
    private void swapKeyModels(ModelEvent.ModifyBakingResult e) {
        Map<ModelResourceLocation, BakedModel> models = e.getModels();
        for (ZodiacKey k : ZodiacKey.values()) {
            ModelResourceLocation inv = new ModelResourceLocation(FairyTailMod.id(k.itemName()), "inventory");
            BakedModel icon = models.get(inv);
            BakedModel hand = models.get(CastKeyLayer.handModel(k));
            if (icon != null && hand != null) {
                models.put(inv, new KeyBakedModel(icon, hand));
            } else {
                FairyTailMod.LOGGER.warn("Missing model for {} (icon={}, hand={}); using whatever exists", k.itemName(), icon != null, hand != null);
            }
        }
    }

    private void addLayers(EntityRenderersEvent.AddLayers e) {
        for (PlayerSkin.Model skin : e.getSkins()) {
            if (e.getSkin(skin) instanceof PlayerRenderer renderer) {
                renderer.addLayer(new CastKeyLayer(renderer));
            }
        }
    }
}
