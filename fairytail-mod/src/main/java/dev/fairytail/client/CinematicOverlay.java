package dev.fairytail.client;

import com.mojang.blaze3d.systems.RenderSystem;
import dev.fairytail.FairyTailMod;
import dev.fairytail.cast.CastTimeline;
import dev.fairytail.cast.Ease;
import dev.fairytail.client.fx.CastEffects;
import net.minecraft.client.DeltaTracker;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.Font;
import net.minecraft.client.gui.GuiGraphics;
import net.minecraft.resources.ResourceLocation;
import net.minecraft.util.Mth;

import static dev.fairytail.cast.CastTimeline.*;

/**
 * Screen-space part of the cinematic: widescreen bars (the picture is the middle 90 % of the screen),
 * vignette, the golden flash, and the invocation text. Fully derived from the cast timeline; the only
 * state is the cast itself, so a cancelled cast leaves nothing behind.
 */
public final class CinematicOverlay {
    public static final ResourceLocation LAYER_ID = FairyTailMod.id("cinematic_overlay");
    private static final float BAR_FRACTION = 0.05f; // 5 % top + 5 % bottom -> 90 % picture

    public static void reset() {
        // intentionally stateless; kept so every exit path has one obvious call
    }

    public static void render(GuiGraphics g, DeltaTracker delta) {
        Minecraft mc = Minecraft.getInstance();
        ClientCast c = ClientCastManager.local();
        if (c == null || mc.player == null) return;
        float partial = delta.getGameTimeDeltaPartialTick(false);
        float t = c.time(mc.player, partial);
        float w = CastTimeline.cameraWeight(t);
        if (w <= 0.001f) return;
        int W = g.guiWidth(), H = g.guiHeight();

        // vignette
        int va = (int) (96 * w);
        int edge = Math.max(24, H / 6);
        g.fillGradient(0, 0, W, edge, va << 24, 0);
        g.fillGradient(0, H - edge, W, H, 0, va << 24);
        g.fillGradient(0, 0, edge, H, va << 24, 0);
        g.fillGradient(W - edge, 0, W, H, 0, va << 24);
        // 3 s of golden bloom on the edges while energy builds
        float build = smooth(THRUST - 6, IMPACT, t) * (1 - smooth(IMPACT, IMPACT + 18, t));
        if (build > 0) {
            int ga = (int) (70 * build);
            int gold = (ga << 24) | 0xFFC040;
            g.fillGradient(0, H - edge * 2, W, H, 0x00FFC040, gold);
        }

        // widescreen bars
        int bar = Math.round(H * BAR_FRACTION * Ease.IN_OUT_CUBIC.apply(w));
        g.fill(0, 0, W, bar, 0xFF000000);
        g.fill(0, H - bar, W, H, 0xFF000000);
        int ruleA = (int) (170 * w);
        g.fill(0, bar, W, bar + 1, (ruleA << 24) | 0xD9A93A);
        g.fill(0, H - bar - 1, W, H - bar, (ruleA << 24) | 0xD9A93A);

        invocation(g, mc.font, c, t, W, H);
        shout(g, mc.font, c, t, W, H);

        float flash = CastTimeline.flash(t);
        if (flash > 0.003f) {
            int a = (int) (Mth.clamp(flash, 0, 1) * 255);
            g.fill(0, 0, W, H, (a << 24) | 0xFFF6D8);
        }
    }

    /** "Open, Gate of the Water Bearer!" typed out letter by letter in shot 2. */
    private static void invocation(GuiGraphics g, Font font, ClientCast c, float t, int W, int H) {
        float in = smooth(18, 36, t);
        float out = 1 - smooth(54, 62, t);
        if (in <= 0 || out <= 0) return;
        String text = c.key.invocation();
        int shown = Math.min(text.length(), (int) Math.ceil(text.length() * in));
        float scale = Math.min(2.4f, (W * 0.8f) / Math.max(1, font.width(text)));
        float alpha = out;
        float y = H * 0.84f;
        float x0 = W / 2f - font.width(text) * scale / 2f;
        float x = x0;
        for (int i = 0; i < shown; i++) {
            String ch = String.valueOf(text.charAt(i));
            float age = in * text.length() - i; // chars flash white as they appear
            float flashK = Mth.clamp(1f - age / 2.5f, 0, 1);
            float shimmer = 0.5f + 0.5f * Mth.sin(t * 0.35f - i * 0.45f);
            int rgb = CastEffects.mix(CastEffects.mix(0xFFCF5A, 0xFFF1B0, shimmer * 0.7f), 0xFFFFFF, Math.max(flashK, 0f));
            float rise = (1f - Mth.clamp(age / 3f, 0, 1)) * 5f;
            drawGlyph(g, font, ch, x, y + rise, scale, rgb, alpha);
            x += font.width(ch) * scale;
        }
    }

    /** "AQUARIUS!" slams in as the key turns. */
    private static void shout(GuiGraphics g, Font font, ClientCast c, float t, int W, int H) {
        float start = IMPACT - 9;
        if (t < start) return;
        float k = Ease.OUT_EXPO.apply((t - start) / 4f);
        float out = 1 - smooth(78, 90, t);
        if (out <= 0) return;
        String text = c.key.shout();
        float scale = Mth.lerp(k, 5.0f, 3.0f);
        float spacing = 3f * scale / 3f;
        float total = 0;
        for (int i = 0; i < text.length(); i++) total += font.width(String.valueOf(text.charAt(i))) * scale + spacing;
        total -= spacing;
        float x = W / 2f - total / 2f;
        float y = H * 0.17f;
        float alpha = Math.min(1f, k * 1.4f) * out;

        // the key's own zodiac sign above the name
        ResourceLocation glyph = FairyTailMod.id("textures/gui/glyph_" + c.key.id + ".png");
        int size = (int) (Mth.lerp(k, 120, 84));
        RenderSystem.enableBlend();
        RenderSystem.blendFunc(com.mojang.blaze3d.platform.GlStateManager.SourceFactor.SRC_ALPHA, com.mojang.blaze3d.platform.GlStateManager.DestFactor.ONE);
        g.setColor(1f, 0.86f, 0.45f, alpha * 0.9f);
        g.blit(glyph, W / 2 - size / 2, (int) y - size - 6, 0, 0, size, size, size, size);
        g.setColor(1f, 1f, 1f, 1f);
        RenderSystem.defaultBlendFunc();

        for (int i = 0; i < text.length(); i++) {
            String ch = String.valueOf(text.charAt(i));
            float shimmer = 0.5f + 0.5f * Mth.sin(t * 0.4f - i * 0.6f);
            int rgb = CastEffects.mix(0xFFC845, 0xFFFFFF, shimmer * 0.75f);
            float jitter = t < IMPACT + 2 ? (float) Math.sin(t * 60 + i * 9) * 1.5f : 0;
            drawGlyph(g, font, ch, x + jitter, y, scale, rgb, alpha);
            x += font.width(ch) * scale + spacing;
        }
    }

    private static void drawGlyph(GuiGraphics g, Font font, String ch, float x, float y, float scale, int rgb, float alpha) {
        int a = (int) (Mth.clamp(alpha, 0, 1) * 255);
        if (a < 6) return;
        g.pose().pushPose();
        g.pose().translate(x, y, 0);
        g.pose().scale(scale, scale, 1f);
        int outline = (a << 24) | 0x7A4A08;
        int glow = ((a / 3) << 24) | 0xFFB020;
        for (int[] o : new int[][]{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}, {0, -1}, {0, 1}, {-1, 0}, {1, 0}}) {
            g.drawString(font, ch, o[0], o[1], outline, false);
        }
        for (int[] o : new int[][]{{-2, 0}, {2, 0}, {0, -2}, {0, 2}}) {
            g.drawString(font, ch, o[0], o[1], glow, false);
        }
        g.drawString(font, ch, 0, 0, (a << 24) | (rgb & 0xFFFFFF), false);
        g.pose().popPose();
    }

    private static float smooth(float a, float b, float t) {
        return CastTimeline.smooth(a, b, t);
    }

    private CinematicOverlay() {}
}
