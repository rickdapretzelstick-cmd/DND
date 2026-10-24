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
    private static final ResourceLocation RING = tex("effect/gate_ring");
    private static final ResourceLocation RAYS = tex("effect/rays");
    private static final ResourceLocation GLOW = tex("effect/glow");
    private static final ResourceLocation FLARE = tex("particle/flare");
    private static final Map<ZodiacKey, ResourceLocation> GLYPHS = new EnumMap<>(ZodiacKey.class);

    static {
        for (ZodiacKey k : ZodiacKey.values()) GLYPHS.put(k, tex("gui/glyph_" + k.id));
    }

    private static ResourceLocation tex(String p) {
        return FairyTailMod.id("textures/" + p + ".png");
    }

    public static void render(RenderLevelStageEvent event) {
        if (event.getStage() != RenderLevelStageEvent.Stage.AFTER_PARTICLES) return;
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
            if (t < 4 || t > TOTAL - 6) continue;
            draw(ps, buf, cam, c, p, t);
        }
        buf.endBatch();
    }

    private static void draw(PoseStack ps, MultiBufferSource.BufferSource buf, Camera cam, ClientCast c, AbstractClientPlayer p, float t) {
        Vec3 camPos = cam.getPosition();
        Vec3 a = c.anchor.subtract(camPos);
        Vec3 fwd = CastEffects.forward(c, p);
        // plane basis for the gate: it faces the way the caster faces
        Vector3f n = new Vector3f((float) fwd.x, (float) fwd.y, (float) fwd.z);
        Vector3f gx = new Vector3f(n.z, 0, -n.x).normalize();
        Vector3f gy = new Vector3f(0, 1, 0);
        Vector3f left = new Vector3f(cam.getLeftVector());
        Vector3f up = new Vector3f(cam.getUpVector());

        int accent = c.key.accent;
        Matrix4f m = ps.last().pose();

        // key halo + 4-point flare, swelling as the invocation builds
        float build = smooth(6, IMPACT, t);
        float pulse = 0.88f + 0.12f * Mth.sin(t * 1.3f);
        float halo = (0.16f + 0.5f * build) * pulse * (1 - smooth(IMPACT + 6, IMPACT + 22, t) * 0.6f);
        float haloA = Math.min(1f, 0.25f + 0.75f * build);
        quad(buf, GLOW, m, a, left, up, halo, 0, argb(haloA, 0xFFD060));
        quad(buf, FLARE, m, a, left, up, halo * 1.5f, t * 0.04f, argb(haloA * 0.9f, 0xFFF4C0));

        // key turn pulse
        if (t >= THRUST + 2 && t < IMPACT) {
            float k = (t - THRUST - 2) / (IMPACT - THRUST - 2);
            quad(buf, FLARE, m, a, left, up, 0.4f + 0.9f * k, t * 0.2f, argb(0.5f + 0.5f * k, 0xFFFFFF));
        }

        // gate ring
        if (t >= THRUST - 2 && t < TOTAL - 8) {
            Vec3 center = a.add(fwd.scale(0.1));
            float radius, alpha;
            if (t < IMPACT) {
                float k = smooth(THRUST - 2, IMPACT, t);
                radius = 0.18f + 0.32f * k;
                alpha = 0.15f + 0.65f * k;
            } else {
                float k = Ease.OUT_EXPO.apply((t - IMPACT) / 11f);
                radius = 0.5f + 2.25f * k + 0.35f * smooth(IMPACT + 11, HOLD_END + 8, t);
                alpha = 1f - smooth(IMPACT + 8, TOTAL - 10, t);
            }
            float spin = t * 0.05f;
            planeQuad(buf, RING, m, center, gx, gy, radius, spin, argb(alpha, 0xFFD770));
            planeQuad(buf, RING, m, center.add(fwd.scale(0.03)), gx, gy, radius * 0.62f, -spin * 1.7f, argb(alpha * 0.8f, mix(0xFFEFB0, accent, 0.45f)));
            planeQuad(buf, GLYPHS.get(c.key), m, center.add(fwd.scale(0.05)), gx, gy, radius * 0.3f, 0, argb(alpha * (0.7f + 0.3f * Mth.sin(t * 0.5f)), 0xFFFFFF));

            if (t >= IMPACT) { // shock ring + counter ring
                float sk = Ease.OUT_EXPO.apply((t - IMPACT) / 9f);
                float sa = 1f - smooth(IMPACT, IMPACT + 12, t);
                planeQuad(buf, RING, m, center.add(fwd.scale(-0.04)), gx, gy, 0.5f + 4.4f * sk, -spin * 0.6f, argb(sa * 0.85f, 0xFFFFFF));
            }
        }

        // rays at the moment of opening
        if (t >= IMPACT - 3 && t < IMPACT + 16) {
            float k = Ease.OUT_EXPO.apply(Mth.clamp((t - IMPACT + 3) / 12f, 0, 1));
            float ra = (1f - smooth(IMPACT + 2, IMPACT + 16, t)) * Math.min(1f, (t - IMPACT + 3) / 2f);
            quad(buf, RAYS, m, a, left, up, 0.4f + 3.6f * k, t * 0.03f, argb(ra, 0xFFE9A0));
            quad(buf, RAYS, m, a, left, up, 0.3f + 2.3f * k, -t * 0.05f, argb(ra * 0.8f, 0xFFFFFF));
        }

        // the key's sign blooms outward after the opening
        if (t >= IMPACT && t < IMPACT + 22) {
            float k = (t - IMPACT) / 22f;
            float s = 0.5f + 1.0f * Ease.OUT_CUBIC.apply(k);
            quad(buf, GLYPHS.get(c.key), m, a, left, up, s, 0, argb((1f - k) * 0.8f, mix(0xFFFFFF, accent, 0.3f)));
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
