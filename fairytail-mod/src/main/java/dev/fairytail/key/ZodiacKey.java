package dev.fairytail.key;

import net.minecraft.core.particles.ParticleOptions;
import net.minecraft.core.particles.ParticleTypes;

import java.util.Locale;

/**
 * Metadata for the twelve Golden Zodiac Keys. All per-key presentation data lives here so the
 * cinematic framework itself stays key-agnostic.
 *
 * <p>{@code gateTitle} is the phrase used in the invocation "Open, Gate of the {gateTitle}!";
 * {@code epithet} is the descriptive name shown in tooltips.
 */
public enum ZodiacKey {
    AQUARIUS("Aquarius", "Water Bearer", "The Water Bearer", 0x2652, 0x46AFFF, ParticleTypes.FALLING_WATER, 1, 0.9),
    TAURUS("Taurus", "Golden Bull", "The Golden Bull", 0x2649, 0xEB4B32, ParticleTypes.FLAME, 1, 1.0),
    CANCER("Cancer", "Crab", "The Giant Crab", 0x264B, 0xFF7D46, ParticleTypes.BUBBLE_POP, 2, 1.0),
    VIRGO("Virgo", "Maiden", "The Maiden", 0x264D, 0xFF96CD, ParticleTypes.CHERRY_LEAVES, 1, 0.8),
    SAGITTARIUS("Sagittarius", "Archer", "The Archer", 0x2650, 0x78CD5A, ParticleTypes.END_ROD, 1, 1.5),
    LEO("Leo", "Lion", "The Lion", 0x264C, 0xFFAA2D, ParticleTypes.SMALL_FLAME, 1, 1.0),
    ARIES("Aries", "Ram", "The Ram", 0x2648, 0xFF78A5, ParticleTypes.CLOUD, 1, 0.9),
    GEMINI("Gemini", "Twins", "The Twins", 0x264A, 0xAF7DFF, ParticleTypes.WITCH, 2, 1.0),
    SCORPIO("Scorpio", "Scorpion", "The Scorpion", 0x264F, 0xCD2D46, ParticleTypes.CRIMSON_SPORE, 1, 1.0),
    CAPRICORN("Capricorn", "Goat", "The Goat", 0x2651, 0x5F78D2, ParticleTypes.SNOWFLAKE, 1, 1.1),
    PISCES("Pisces", "Fish", "The Paired Fish", 0x2653, 0x5AE1D7, ParticleTypes.BUBBLE, 2, 1.0),
    LIBRA("Libra", "Scales", "The Scales", 0x264E, 0xE6E6FF, ParticleTypes.GLOW, 2, 1.0);

    public final String id = name().toLowerCase(Locale.ROOT);
    public final String spiritName;
    public final String gateTitle;
    public final String epithet;
    public final String glyph;
    /** Accent colour (0xRRGGBB) used to tint the gem, inner ring and secondary particles. */
    public final int accent;
    /** Vanilla particle mixed into this key's effects as a zodiac-specific flourish. */
    public final ParticleOptions flourish;
    /** How many counter-rotating orbiters circle the key head (twins/fish/scales use two). */
    public final int orbiters;
    /** Relative speed of the gathering swirl. */
    public final double swirlSpeed;

    ZodiacKey(String spiritName, String gateTitle, String epithet, int glyphCodePoint, int accent,
              ParticleOptions flourish, int orbiters, double swirlSpeed) {
        this.spiritName = spiritName;
        this.gateTitle = gateTitle;
        this.epithet = epithet;
        this.glyph = new String(Character.toChars(glyphCodePoint));
        this.accent = accent;
        this.flourish = flourish;
        this.orbiters = orbiters;
        this.swirlSpeed = swirlSpeed;
    }

    /** "Open, Gate of the Water Bearer!" */
    public String invocation() {
        return "Open, Gate of the " + gateTitle + "!";
    }

    /** "AQUARIUS!" */
    public String shout() {
        return spiritName.toUpperCase(Locale.ROOT) + "!";
    }

    public String itemName() {
        return "key_" + id;
    }

    public static ZodiacKey byIndex(int i) {
        ZodiacKey[] v = values();
        return v[Math.floorMod(i, v.length)];
    }
}
