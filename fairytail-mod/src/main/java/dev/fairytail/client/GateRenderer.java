package dev.fairytail.client;

import com.mojang.blaze3d.vertex.PoseStack;
import com.mojang.blaze3d.vertex.VertexConsumer;
import dev.fairytail.FairyTailMod;
import dev.fairytail.cast.CastTimeline;
import dev.fairytail.cast.Ease;
import dev.fairytail.client.fx.CastEffects;
import dev.fairytail.key.ZodiacKey;
import net.minecraft.client.Camera;
import net.minecraft.client.Minecraft;
import net.minecraft.client.player.AbstractClientPlayer;
import net.minecraft.client.renderer.MultiBufferSource;
import net.minecraft.client.renderer.RenderType;
import net.minecraft.client.renderer.texture.OverlayTexture;
import net.minecraft.resources.ResourceLocation;
import net.minecraft.util.Mth;
import net.minecraft.world.entity.Entity;
import net.minecraft.world.phys.Vec3;
import net.neoforged.neoforge.client.event.RenderLevelStageEvent;
import org.joml.Matrix4f;
import org.joml.Vector3f;

import java.util.EnumMap;
import java.util.Map;

import static dev.fairytail.cast.CastTimeline.*;

/**
 * World-space magic: the key's halo, the golden gate ring with all twelve zodiac signs, the shock ring,
 * the light rays, and the key's own sign. Everything is additive (RenderType.eyes) and a pure function of
 * the cast timeline, so other players see the same gate when they are near a caster.
 */
public final class GateRenderer {
    private static final ResourceLocation RING_OUTER = tex("effect/gate_outer");
    private static final ResourceLocation RAYS = tex("effect/rays");
    private static final ResourceLocation DESIGN = tex("effect/gate_design");
    private static final ResourceLocation SPIKES = tex("effect/spikes");
    private static final ResourceLocation GLOW = tex("effect/glow");
    private static final ResourceLocation FLARE = tex("effect/flare");
    private static final Map<ZodiacKey, ResourceLocation> GLYPHS = new EnumMap<>(ZodiacKey.class);

    static {
        for (ZodiacKey k : ZodiacKey.values()) {
            GLYPHS.put(k, tex("gui/glyph_" + k.id));
        }
    }

    private static ResourceLocation tex(String p) {
        return FairyTailMod.id("textures/" + p + ".png");
    }

    public static void render(RenderLevelStageEvent event) {
        if (event.getStage() != RenderLevelStageEvent.Stage.AFTER_PARTICLES || AutoTest.NO_FX) return;
        if (ClientCastManager.local() == null && !ClientCastManager.all().iterator().hasNext()) return;
        Minecraft mc = Minecraft.getInstance();
        if (mc.level == null) return;
        float partial = event.getPartialTick().getGameTimeDeltaPartialTick(false);
        Camera cam = event.getCamera();
        PoseStack ps = event.getPoseStack();
        MultiBufferSource.BufferSource buf = mc.renderBuffers().bufferSource();

        for (ClientCast c : ClientCastManager.all()) {
            Entity e = mc.level.getEntity(c.entityId);
            if (!(e instanceof AbstractClientPlayer p) || !c.hasAnchor) continue;
            float t = c.time(p, partial);
            if (t < 4 || t > TOTAL - 2) continue;
            draw(ps, buf, cam, c, p, t);
        }
        buf.endBatch();
    }

    private static void draw(PoseStack ps, MultiBufferSource.BufferSource buf, Camera cam, ClientCast c, AbstractClientPlayer p, float t) {
        Vec3 camPos = cam.getPosition();
        Vec3 a = c.anchor.subtract(camPos); // key tip = the keyhole
        // The circle stands perpendicular to the key, ahead of its tip, so the key points into its centre
        Vec3 out = c.axis.lengthSqr() < 1e-4 ? CastEffects.forward(c, p) : c.axis.scale(-1).normalize();
        Vector3f n = new Vector3f((float) out.x, (float) out.y, (float) out.z);
        Vector3f ref = Math.abs(n.y) > 0.9f ? new Vector3f(1, 0, 0) : new Vector3f(0, 1, 0);
        Vector3f gx = new Vector3f(n).cross(ref).normalize();
        Vector3f gy = new Vector3f(n).cross(gx).normalize();
        Vector3f left = new Vector3f(cam.getLeftVector());
        Vector3f up = new Vector3f(cam.getUpVector());

        ZodiacKey key = c.key;
        int gc = key.gateColor();
        int accent = mix(gc, key.accent, 0.12f);
        Matrix4f m = ps.last().pose();

        // halo + flare at the keyhole, swelling as the invocation builds
        float build = smooth(6, IMPACT, t);
        float pulse = 0.88f + 0.12f * Mth.sin(t * 1.3f);
        float halo = (0.09f + 0.26f * build) * pulse * (1 - smooth(IMPACT, IMPACT + 8, t) * 0.85f);
        float near = Mth.clamp((float) a.length() / 3.5f, 0.12f, 1f); // the keyhole can be close to the lens
        halo *= near;
        float haloA = Math.min(0.8f, 0.2f + 0.6f * build) * (0.4f + 0.6f * near);
        quad(buf, GLOW, m, a, left, up, halo, 0, argb(haloA, gc));
        quad(buf, FLARE, m, a, left, up, halo * 1.5f, t * 0.04f, argb(haloA * 0.9f, mix(gc, 0xFFFFFF, 0.6f)));

        // the turn of the key: a spark ratchets round the tip while it turns
        if (t >= TURN_START && t < IMPACT) {
            float k = (t - TURN_START) / (IMPACT - TURN_START);
            quad(buf, FLARE, m, a, left, up, (0.15f + 0.4f * k) * near, t * 0.35f, argb(0.3f + 0.4f * k, 0xFFFFFF));
        }

        // the gate: outer ring of 12 signs, the key's own circle, optional counter-rotating twin layer
        if (t >= THRUST - 4 && t < TOTAL - 3) {
            Vec3 center = a.add(out.scale(0.12));
            float radius, alpha;
            if (t < IMPACT) {
                float k = smooth(THRUST - 4, IMPACT, t);   // keyhole circle grows while the key turns
                radius = 0.16f + 0.3f * k;
                alpha = 0.2f + 0.6f * k;
            } else {
                float k = Ease.OUT_EXPO.apply((t - IMPACT) / 11f);
                radius = 0.46f + 1.0f * k + 0.1f * smooth(IMPACT + 11, HOLD_END, t);
                alpha = 1f - smooth(HOLD_END, HOLD_END + 7, t);   // stays fully open for the whole hold
            }
            float mid = (float) (key.midSpin() * t), outer = (float) (key.outerSpin() * t);
            Vec3 c0 = center;
            planeQuad(buf, RING_OUTER, m, c0, gx, gy, radius, outer, argb(alpha, gc));
            planeQuad(buf, DESIGN, m, c0.add(out.scale(0.02)), gx, gy, radius * 0.74f, mid, argb(alpha, mix(gc, 0xFFFFFF, 0.35f)));
            planeQuad(buf, GLYPHS.get(key), m, c0.add(out.scale(0.05)), gx, gy, radius * 0.22f, 0, argb(alpha * (0.35f + 0.15f * Mth.sin(t * 0.5f)), mix(0xFFFFFF, accent, 0.35f)));

            if (t >= IMPACT) { // shock ring
                float sk = Ease.OUT_EXPO.apply((t - IMPACT) / 9f);
                float sa = 1f - smooth(IMPACT, IMPACT + 12, t);
                planeQuad(buf, RING_OUTER, m, c0.add(out.scale(-0.04)), gx, gy, 0.5f + 2.6f * sk, -outer * 0.6f, argb(sa * 0.7f, 0xFFFFFF));
            }
        }

        // rays at the moment of opening
        if (t >= IMPACT - 3 && t < IMPACT + 9) {
            float k = Ease.OUT_EXPO.apply(Mth.clamp((t - IMPACT + 3) / 12f, 0, 1));
            float ra = (1f - smooth(IMPACT + 1, IMPACT + 9, t)) * Math.min(1f, (t - IMPACT + 3) / 2f);
            // bold curved light spikes bursting from the gate, as in the reference frame
            planeQuad(buf, SPIKES, m, a.add(out.scale(0.1)), gx, gy, 0.4f + 2.0f * k, t * 0.025f, argb(ra * 0.5f, gc));
            planeQuad(buf, SPIKES, m, a.add(out.scale(0.12)), gx, gy, 0.3f + 1.6f * k, -t * 0.04f + 0.4f, argb(ra * 0.35f, 0xFFFFFF));
        }
    }

    private static void quad(MultiBufferSource buf, ResourceLocation tex, Matrix4f m, Vec3 c, Vector3f ax, Vector3f ay, float half, float angle, int argb) {
        emit(buf.getBuffer(RenderType.eyes(tex)), m, c, ax, ay, half, angle, argb);
    }

    private static void planeQuad(MultiBufferSource buf, ResourceLocation tex, Matrix4f m, Vec3 c, Vector3f ax, Vector3f ay, float half, float angle, int argb) {
        emit(buf.getBuffer(RenderType.eyes(tex)), m, c, ax, ay, half, angle, argb);
    }

    /** One quad drawn double-sided (the gate must read from either side). */
    private static void emit(VertexConsumer vc, Matrix4f m, Vec3 c, Vector3f ax, Vector3f ay, float half, float angle, int argb) {
        float cs = Mth.cos(angle), sn = Mth.sin(angle);
        Vector3f x = new Vector3f(ax).mul(cs).add(new Vector3f(ay).mul(sn)).mul(half);
        Vector3f y = new Vector3f(ay).mul(cs).sub(new Vector3f(ax).mul(sn)).mul(half);
        float[][] corners = {{-1, -1, 0, 1}, {1, -1, 1, 1}, {1, 1, 1, 0}, {-1, 1, 0, 0}};
        int a = (argb >> 24) & 255, r = (argb >> 16) & 255, g = (argb >> 8) & 255, b = argb & 255;
        for (int pass = 0; pass < 2; pass++) {
            for (int i = 0; i < 4; i++) {
                float[] k = corners[pass == 0 ? i : 3 - i];
                float px = (float) c.x + x.x * k[0] + y.x * k[1];
                float py = (float) c.y + x.y * k[0] + y.y * k[1];
                float pz = (float) c.z + x.z * k[0] + y.z * k[1];
                vc.addVertex(m, px, py, pz).setColor(r, g, b, a).setUv(k[2], k[3])
                        .setOverlay(OverlayTexture.NO_OVERLAY).setLight(0xF000F0).setNormal(0, 0, 1);
            }
        }
    }

    private static float smooth(float a, float b, float t) {
        return CastTimeline.smooth(a, b, t);
    }

    private static int argb(float a, int rgb) {
        return ((int) (Mth.clamp(a, 0, 1) * 255) << 24) | (rgb & 0xFFFFFF);
    }

    private static int mix(int a, int b, float t) {
        return CastEffects.mix(a, b, t);
    }

    private GateRenderer() {}
}
