package dev.fairytail.client.fx;

import com.mojang.blaze3d.systems.RenderSystem;
import com.mojang.blaze3d.vertex.BufferBuilder;
import com.mojang.blaze3d.vertex.DefaultVertexFormat;
import com.mojang.blaze3d.vertex.Tesselator;
import com.mojang.blaze3d.vertex.VertexFormat;
import net.minecraft.client.multiplayer.ClientLevel;
import net.minecraft.client.particle.ParticleRenderType;
import net.minecraft.client.particle.SpriteSet;
import net.minecraft.client.particle.TextureSheetParticle;
import net.minecraft.client.renderer.GameRenderer;
import net.minecraft.client.renderer.texture.TextureAtlas;
import net.minecraft.client.renderer.texture.TextureManager;
import com.mojang.blaze3d.platform.GlStateManager;

/** Self-lit, additively blended particle used for every celestial sparkle, mote and flare. */
public class GlowParticle extends TextureSheetParticle {
    public static final ParticleRenderType ADDITIVE = new ParticleRenderType() {
        @Override
        public BufferBuilder begin(Tesselator tesselator, TextureManager textures) {
            RenderSystem.depthMask(false);
            RenderSystem.enableBlend();
            RenderSystem.blendFunc(GlStateManager.SourceFactor.SRC_ALPHA, GlStateManager.DestFactor.ONE);
            RenderSystem.setShader(GameRenderer::getParticleShader);
            RenderSystem.setShaderTexture(0, TextureAtlas.LOCATION_PARTICLES);
            return tesselator.begin(VertexFormat.Mode.QUADS, DefaultVertexFormat.PARTICLE);
        }

        @Override
        public String toString() {
            return "fairytail:additive";
        }
    };

    private final float baseSize;
    private final float gravity;
    private final float friction;
    private final boolean twinkle;
    private final float spin;

    public GlowParticle(ClientLevel level, double x, double y, double z, double vx, double vy, double vz,
                        SpriteSet sprites, int rgb, float size, int life, float gravity, float friction, boolean twinkle) {
        super(level, x, y, z);
        this.xd = vx;
        this.yd = vy;
        this.zd = vz;
        this.baseSize = size;
        this.quadSize = size;
        this.lifetime = Math.max(2, life);
        this.gravity = gravity;
        this.friction = friction;
        this.twinkle = twinkle;
        this.hasPhysics = false;
        this.spin = (this.random.nextFloat() - 0.5f) * 0.3f;
        this.roll = this.oRoll = this.random.nextFloat() * 6.2831f;
        this.rCol = ((rgb >> 16) & 0xFF) / 255f;
        this.gCol = ((rgb >> 8) & 0xFF) / 255f;
        this.bCol = (rgb & 0xFF) / 255f;
        this.pickSprite(sprites);
    }

    @Override
    public void tick() {
        this.xo = this.x;
        this.yo = this.y;
        this.zo = this.z;
        if (this.age++ >= this.lifetime) {
            this.remove();
            return;
        }
        this.yd -= gravity * 0.04;
        this.move(this.xd, this.yd, this.zd);
        this.xd *= friction;
        this.yd *= friction;
        this.zd *= friction;
        this.oRoll = this.roll;
        this.roll += spin;
        float k = (float) this.age / this.lifetime;
        this.alpha = Math.max(0f, 1f - k * k);
    }

    @Override
    public float getQuadSize(float partialTick) {
        float k = ((float) this.age + partialTick) / this.lifetime;
        float s = baseSize * (1f - 0.55f * k);
        if (twinkle) s *= 0.75f + 0.25f * (float) Math.sin((this.age + partialTick) * 1.9 + this.roll * 5);
        return s;
    }

    @Override
    protected int getLightColor(float partialTick) {
        return 0xF000F0;
    }

    @Override
    public ParticleRenderType getRenderType() {
        return ADDITIVE;
    }
}
