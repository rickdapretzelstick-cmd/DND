#!/usr/bin/env python3
"""
Procedural asset generator for the Fairy Tail Celestial Spirit Keys mod.

Everything here is original work (no ripped art or audio):
  * 12 Blockbench-compatible "Java Block/Item" key models (cuboid elements)
  * per-key 32x32 material texture, and 64x64 inventory icons rendered from the
    very same cuboid data (so icon and held model always agree)
  * zodiac glyph / gate-ring / ray / glow / particle textures
  * layered synthesized sound effects (OGG Vorbis through ffmpeg)

Usage:  python3 tools/gen_assets.py [models|textures|sounds|all]
"""
import json, math, os, random, subprocess, sys, tempfile
import numpy as np
from PIL import Image, ImageDraw, ImageFilter, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
ASSETS = os.path.join(HERE, "..", "src", "main", "resources", "assets", "fairytail")
FONT = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"


def out(*p):
    path = os.path.join(ASSETS, *p)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    return path


# --------------------------------------------------------------------------------------
# Key definitions
# --------------------------------------------------------------------------------------
# id, glyph codepoint, gem colour (accent), bit teeth (y1, y2, length)
KEYS = {
    "aquarius":    (0x2652, (70, 175, 255),  [(1.0, 2.6, 1.8), (3.4, 4.8, 1.2), (5.6, 7.0, 1.8)]),
    "taurus":      (0x2649, (235, 75, 50),   [(1.0, 2.4, 1.6), (3.0, 4.2, 2.0), (5.0, 6.4, 1.2)]),
    "cancer":      (0x264B, (255, 125, 70),  [(1.0, 2.2, 1.2), (2.8, 4.4, 1.8), (5.2, 6.6, 1.4)]),
    "virgo":       (0x264D, (255, 150, 205), [(1.0, 2.8, 1.4), (3.6, 4.8, 1.8), (5.6, 6.6, 1.0)]),
    "sagittarius": (0x2650, (120, 205, 90),  [(1.0, 2.0, 2.0), (2.8, 4.0, 1.2), (4.8, 6.4, 1.6)]),
    "leo":         (0x264C, (255, 170, 45),  [(1.0, 2.6, 1.6), (3.4, 4.4, 2.0), (5.2, 6.8, 1.4)]),
    "aries":       (0x2648, (255, 120, 165), [(1.0, 2.2, 1.8), (3.0, 5.0, 1.0), (5.8, 7.0, 1.6)]),
    "gemini":      (0x264A, (175, 125, 255), [(1.0, 2.4, 1.4), (3.2, 4.2, 1.8), (5.0, 6.0, 1.4), (6.6, 7.4, 1.0)]),
    "scorpio":     (0x264F, (205, 45, 70),   [(1.0, 2.0, 1.8), (2.8, 3.8, 1.0), (4.6, 6.6, 1.6)]),
    "capricorn":   (0x2651, (95, 120, 210),  [(1.0, 2.8, 1.2), (3.6, 5.2, 1.8), (6.0, 7.0, 1.2)]),
    "pisces":      (0x2653, (90, 225, 215),  [(1.0, 2.4, 1.6), (3.2, 4.0, 1.0), (4.8, 6.4, 1.8)]),
    "libra":       (0x264E, (230, 230, 255), [(1.0, 2.6, 1.0), (3.4, 5.0, 1.8), (5.8, 7.0, 1.4)]),
}

SWATCH = {"gold": (0, 0), "dark": (16, 0), "gem": (0, 16), "light": (16, 16)}


class Box:
    def __init__(self, a, b, mat="gold", rot=None):
        self.a = [min(a[i], b[i]) for i in range(3)]
        self.b = [max(a[i], b[i]) for i in range(3)]
        self.mat = mat
        self.rot = rot  # (axis, angle, origin) or None

    def mirrored(self):
        a = [16 - self.b[0], self.a[1], self.a[2]]
        b = [16 - self.a[0], self.b[1], self.b[2]]
        rot = None
        if self.rot:
            ax, ang, o = self.rot
            rot = (ax, -ang if ax == "z" else ang, [16 - o[0], o[1], o[2]])
        return Box(a, b, self.mat, rot)


def B(x1, y1, z1, x2, y2, z2, mat="gold", rot=None):
    return Box((x1, y1, z1), (x2, y2, z2), mat, rot)


def sym(boxes):
    """boxes + their mirror image across x=8"""
    return boxes + [b.mirrored() for b in boxes]


def tilted(x1, y1, z1, x2, y2, z2, angle, mat="gold"):
    """box rotated about its own centre on the z axis (angle in 22.5 degree steps)"""
    o = [(x1 + x2) / 2, (y1 + y2) / 2, (z1 + z2) / 2]
    return Box((x1, y1, z1), (x2, y2, z2), mat, ("z", angle, o))


def ring(cx, cy, r, t, z0, z1, mat="gold", skip=()):
    """octagonal ring in the XY plane out of 8 (possibly tilted) segments"""
    boxes = []
    seg = 2 * r * math.tan(math.radians(22.5)) + t * 0.9
    for k in range(8):
        if k in skip:
            continue
        a = math.radians(45 * k)
        px, py = cx + r * math.cos(a), cy + r * math.sin(a)
        tang = (45 * k + 90) % 180
        if tang == 0:
            boxes.append(B(px - seg / 2, py - t / 2, z0, px + seg / 2, py + t / 2, z1, mat))
        elif tang == 90:
            boxes.append(B(px - t / 2, py - seg / 2, z0, px + t / 2, py + seg / 2, z1, mat))
        else:
            ang = 45 if tang == 45 else -45
            boxes.append(tilted(px - seg / 2, py - t / 2, z0, px + seg / 2, py + t / 2, z1, ang, mat))
    return boxes


def base_key(teeth):
    boxes = [
        B(7.4, 1.0, 7.4, 8.6, 10.0, 8.6),
        B(7.6, 0.4, 7.6, 8.4, 1.0, 8.4, "light"),
        B(6.9, 9.2, 6.9, 9.1, 10.2, 9.1, "dark"),
        B(7.0, 8.2, 7.0, 9.0, 8.6, 9.0, "light"),
        B(7.2, 4.0, 7.2, 8.8, 4.4, 8.8, "dark"),
    ]
    for (y1, y2, ln) in teeth:
        boxes.append(B(8.6, y1, 7.5, 8.6 + ln, y2, 8.5, "gold"))
    return boxes


def head_aquarius():
    b = [
        B(6.4, 15.2, 7.0, 9.6, 16.0, 9.0), B(7.0, 14.0, 7.2, 9.0, 15.2, 8.8),
        B(5.6, 12.8, 6.6, 10.4, 14.0, 9.4), B(4.8, 10.6, 6.4, 11.2, 12.8, 9.6),
        B(6.0, 10.0, 7.0, 10.0, 10.6, 9.0, "dark"),
        B(7.2, 11.0, 9.6, 8.8, 12.6, 9.9, "gem"), B(7.2, 11.0, 6.1, 8.8, 12.6, 6.4, "gem"),
    ]
    for i in range(5):  # wave band, front and back
        y = 11.7 + (0.5 if i % 2 else 0)
        for z0, z1 in ((9.5, 9.75), (6.25, 6.5)):
            b.append(B(4.9 + i * 1.3, y, z0, 6.0 + i * 1.3, y + 0.45, z1, "dark"))
    b += sym(ring(3.9, 12.6, 1.4, 0.7, 7.4, 8.6))
    b += sym([B(3.2, 8.8, 7.6, 4.0, 9.8, 8.4, "gem")])  # falling water drops
    b += [B(7.5, 16.0, 7.5, 8.5, 16.6, 8.5, "gem")]
    return b


def head_taurus():
    b = [
        B(6.2, 10.0, 6.8, 9.8, 13.6, 9.2), B(6.6, 9.4, 6.4, 9.4, 10.8, 9.6, "light"),
        B(6.8, 12.2, 9.2, 7.6, 12.9, 9.5, "gem"),
        B(7.6, 13.2, 9.0, 8.4, 13.9, 9.4, "light"),
    ]
    b += ring(8, 9.1, 0.9, 0.35, 8.6, 9.8, "light")
    b += sym([
        B(4.8, 13.0, 7.2, 6.4, 14.0, 8.8), B(3.6, 13.8, 7.2, 5.0, 14.8, 8.8),
        B(3.0, 14.6, 7.2, 4.0, 16.2, 8.8, "light"),
        B(5.0, 11.6, 7.4, 6.2, 12.6, 8.6, "dark"),
    ])
    b += [B(6.8, 12.2, 6.4, 7.6, 12.9, 6.7, "gem"), B(8.4, 12.2, 6.4, 9.2, 12.9, 6.7, "gem")]
    return b


def head_cancer():
    b = [
        B(5.6, 10.4, 6.6, 10.4, 13.0, 9.4), B(6.4, 13.0, 7.0, 9.6, 13.7, 9.0, "dark"),
        B(7.0, 11.0, 9.4, 9.0, 12.4, 9.7, "gem"),
    ]
    b += sym([
        B(6.4, 13.7, 7.5, 6.9, 14.8, 8.0), B(6.1, 14.8, 7.3, 7.1, 15.6, 8.3, "gem"),
        B(3.2, 11.4, 7.2, 5.6, 12.2, 8.8), B(2.2, 11.6, 6.8, 3.8, 13.6, 9.2),
        B(2.2, 13.6, 7.0, 3.0, 15.8, 9.0, "light"), B(3.4, 13.6, 7.0, 4.2, 15.0, 9.0),
        B(5.0, 9.8, 7.0, 5.8, 10.6, 8.0, "dark"),
    ])
    return b


def head_virgo():
    b = ring(8, 14.7, 1.7, 0.5, 7.6, 8.4, "light")
    b += [
        B(7.0, 13.4, 7.2, 9.0, 15.2, 8.8, "light"), B(6.6, 10.2, 7.0, 9.4, 13.4, 9.0, "dark"),
        B(5.4, 10.0, 6.6, 10.6, 11.0, 9.4), B(7.3, 11.6, 9.0, 8.7, 12.8, 9.3, "gem"),
        B(7.3, 11.6, 6.7, 8.7, 12.8, 7.0, "gem"),
    ]
    b += sym([B(6.0, 10.8, 6.8, 6.9, 14.6, 9.2)])
    # wheat ear on the right
    b += [B(11.3, 10.4, 7.7, 11.8, 14.2, 8.3, "dark")]
    for i in range(3):
        y = 12.2 + i * 1.1
        b += [B(10.7, y, 7.6, 11.3, y + 0.8, 8.4, "gem"), B(11.8, y, 7.6, 12.4, y + 0.8, 8.4, "gem")]
    b += [B(11.2, 14.4, 7.6, 11.9, 15.4, 8.4, "gem")]
    return b


def head_sagittarius():
    b = ring(9.6, 13.0, 3.4, 0.8, 7.4, 8.6, skip=(7, 0, 1))
    b += [
        B(9.4, 9.6, 7.8, 9.8, 16.4, 8.2, "dark"),
        B(4.6, 12.8, 7.8, 14.2, 13.2, 8.2),
        B(14.2, 12.3, 7.6, 15.0, 13.7, 8.4, "light"), B(15.0, 12.8, 7.7, 15.8, 13.2, 8.3, "light"),
        tilted(4.0, 11.9, 7.8, 5.4, 12.3, 8.2, 45, "gem"), tilted(4.0, 13.7, 7.8, 5.4, 14.1, 8.2, -45, "gem"),
        B(9.0, 12.5, 7.3, 10.2, 13.5, 8.7, "gem"),
    ]
    return b


def head_leo():
    cx, cy = 8, 12.7
    b = ring(cx, cy, 2.9, 1.3, 6.8, 9.2)
    for k in range(8):
        a = math.radians(45 * k)
        px, py = cx + 3.8 * math.cos(a), cy + 3.8 * math.sin(a)
        if k % 2:
            b.append(tilted(px - 0.7, py - 0.7, 6.9, px + 0.7, py + 0.7, 9.1, 45, "light"))
        else:
            b.append(B(px - 0.6, py - 0.6, 7.0, px + 0.6, py + 0.6, 9.0, "light"))
    b += [
        B(6.4, 11.4, 6.6, 9.6, 14.2, 9.6, "dark"), B(7.0, 11.4, 9.4, 9.0, 12.8, 10.2),
        B(7.6, 12.4, 10.0, 8.4, 12.9, 10.4, "light"),
        B(6.8, 13.1, 9.5, 7.5, 13.7, 9.8, "gem"), B(8.5, 13.1, 9.5, 9.2, 13.7, 9.8, "gem"),
        B(6.8, 13.1, 6.3, 7.5, 13.7, 6.6, "gem"), B(8.5, 13.1, 6.3, 9.2, 13.7, 6.6, "gem"),
    ]
    return b


def head_aries():
    b = [
        B(7.0, 9.8, 7.0, 9.0, 13.0, 9.0), B(7.2, 9.4, 7.4, 8.8, 10.4, 8.6, "dark"),
        B(6.8, 13.0, 6.8, 9.2, 14.4, 9.2, "light"), B(7.5, 12.1, 9.0, 8.5, 12.9, 9.3, "gem"),
    ]
    b += sym(ring(5.6, 12.6, 2.1, 1.0, 7.0, 9.0, skip=(0,)) + ring(5.6, 12.6, 0.7, 0.5, 7.4, 8.6, "dark"))
    b += sym([B(4.6, 10.4, 7.2, 6.4, 11.0, 8.8, "dark")])
    return b


def head_gemini():
    b = []
    b += [  # left twin (gold) and right twin (light)
        B(3.2, 13.4, 7.2, 5.2, 15.4, 8.8, "light"), B(3.0, 10.2, 7.0, 5.4, 13.4, 9.0, "dark"),
        B(10.8, 13.4, 7.2, 12.8, 15.4, 8.8, "gold"), B(10.6, 10.2, 7.0, 13.0, 13.4, 9.0, "light"),
        B(5.4, 11.8, 7.4, 10.6, 12.6, 8.6),
        tilted(7.1, 11.5, 7.0, 8.9, 13.3, 9.0, 45, "gem"),
        B(3.4, 15.4, 7.6, 5.0, 16.0, 8.4, "gem"), B(11.0, 15.4, 7.6, 12.6, 16.0, 8.4, "gem"),
    ]
    return b


def head_scorpio():
    b = [
        B(6.4, 10.2, 6.8, 9.6, 12.2, 9.2), B(7.2, 10.8, 9.0, 8.8, 11.8, 9.4, "gem"),
        B(4.6, 11.2, 7.2, 6.4, 12.0, 8.8), B(3.4, 11.6, 6.8, 4.8, 13.6, 9.2),
        B(3.4, 13.6, 7.0, 4.0, 15.2, 9.0, "light"), B(4.4, 13.6, 7.0, 5.0, 14.6, 9.0),
        # tail curling over
        B(9.6, 10.6, 7.2, 11.0, 11.6, 8.8), B(10.4, 11.6, 7.2, 11.8, 13.0, 8.8),
        B(10.6, 13.0, 7.2, 12.0, 14.6, 8.8), B(9.6, 14.6, 7.2, 11.4, 15.8, 8.8),
        B(8.6, 14.8, 7.3, 9.8, 16.0, 8.7), B(8.2, 13.4, 7.4, 9.0, 14.9, 8.6, "gem"),
        B(5.2, 9.6, 7.0, 6.0, 10.4, 8.0, "dark"), B(10.0, 9.6, 7.0, 10.8, 10.4, 8.0, "dark"),
    ]
    return b


def head_capricorn():
    b = [
        B(6.8, 10.2, 7.0, 9.2, 13.2, 9.0), B(7.4, 9.2, 7.6, 8.6, 10.2, 8.4, "dark"),
        B(7.2, 12.0, 9.0, 8.8, 12.7, 9.3, "gem"),
    ]
    # long swept-back goat horns on the left...
    b += [
        B(6.0, 12.6, 7.4, 7.0, 14.0, 8.6), B(5.2, 13.8, 7.4, 6.4, 14.8, 8.6),
        B(4.2, 14.6, 7.4, 5.4, 15.6, 8.6), B(3.4, 15.2, 7.4, 4.4, 16.2, 8.6, "light"),
        B(2.4, 14.4, 7.4, 3.4, 15.6, 8.6, "light"),
    ]
    # ...and the sea-goat's curling fish tail on the right
    b += [
        B(9.2, 10.4, 7.4, 10.6, 11.4, 8.6), B(10.4, 11.4, 7.4, 11.6, 12.4, 8.6),
        B(11.2, 12.4, 7.4, 12.4, 13.4, 8.6), B(12.0, 13.4, 7.6, 13.0, 15.0, 8.4, "light"),
        B(13.0, 13.4, 7.6, 14.0, 14.4, 8.4, "light"), B(10.6, 12.6, 6.9, 11.2, 13.2, 7.4, "gem"),
        B(9.2, 12.6, 7.4, 10.0, 13.6, 8.6),
    ]
    return b


def head_pisces():
    b = [
        # fish 1 (head up, left)
        B(4.2, 11.0, 7.2, 6.6, 14.4, 8.8), B(4.6, 14.4, 7.2, 6.2, 15.4, 8.8), B(4.6, 9.8, 7.4, 6.2, 11.0, 8.6),
        B(3.8, 9.2, 7.6, 7.0, 10.0, 8.4, "light"), B(5.0, 14.6, 8.8, 5.6, 15.0, 9.0, "gem"),
        B(4.1, 12.6, 8.8, 6.7, 12.9, 9.0, "dark"), B(4.1, 11.8, 8.8, 6.7, 12.1, 9.0, "dark"),
        # fish 2 (head down, right)
        B(9.4, 9.8, 7.2, 11.8, 13.2, 8.8), B(9.8, 9.0, 7.2, 11.4, 9.8, 8.8), B(9.8, 13.2, 7.4, 11.4, 14.4, 8.6),
        B(9.0, 14.4, 7.6, 12.2, 15.2, 8.4, "light"), B(10.4, 9.2, 8.8, 11.0, 9.6, 9.0, "gem"),
        B(9.3, 11.5, 8.8, 11.9, 11.8, 9.0, "dark"), B(9.3, 10.7, 8.8, 11.9, 11.0, 9.0, "dark"),
        # cord and knot
        B(6.6, 12.4, 7.8, 9.4, 12.8, 8.2, "dark"), B(7.3, 11.9, 7.3, 8.7, 13.3, 8.7, "gem"),
    ]
    return b


def head_libra():
    b = [
        B(7.6, 10.0, 7.6, 8.4, 15.0, 8.4), B(3.0, 14.6, 7.6, 13.0, 15.2, 8.4),
        B(7.2, 15.2, 7.2, 8.8, 16.4, 8.8, "gem"),
        B(7.2, 10.0, 7.2, 8.8, 10.8, 8.8, "light"),
    ]
    b += sym([
        B(3.0, 12.4, 7.9, 3.2, 14.6, 8.1, "dark"), B(4.8, 12.4, 7.9, 5.0, 14.6, 8.1, "dark"),
        B(2.4, 12.0, 7.0, 5.6, 12.4, 9.0), B(2.2, 12.4, 6.8, 2.6, 12.9, 9.2, "light"),
        B(5.4, 12.4, 6.8, 5.8, 12.9, 9.2, "light"),
    ])
    b += [B(3.4, 12.4, 7.5, 4.6, 13.0, 8.5, "gem")]
    return b


HEADS = {
    "aquarius": head_aquarius, "taurus": head_taurus, "cancer": head_cancer, "virgo": head_virgo,
    "sagittarius": head_sagittarius, "leo": head_leo, "aries": head_aries, "gemini": head_gemini,
    "scorpio": head_scorpio, "capricorn": head_capricorn, "pisces": head_pisces, "libra": head_libra,
}


def key_boxes(name):
    return base_key(KEYS[name][2]) + HEADS[name]()


# --------------------------------------------------------------------------------------
# Material swatch texture (32x32)
# --------------------------------------------------------------------------------------
def lerp(a, b, t):
    return tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(len(a)))


def swatch_colors(gem):
    return {
        "gold": ((255, 238, 150), (196, 136, 28)),
        "dark": ((176, 120, 30), (104, 66, 12)),
        "light": ((255, 252, 214), (255, 215, 96)),
        "gem": (lerp(gem, (255, 255, 255), 0.55), lerp(gem, (0, 0, 0), 0.35)),
    }


def make_swatch_texture(name):
    gem = KEYS[name][1]
    rng = random.Random(name)
    img = Image.new("RGBA", (32, 32), (0, 0, 0, 255))
    px = img.load()
    cols = swatch_colors(gem)
    for mat, (ox, oy) in SWATCH.items():
        top, bot = cols[mat]
        for y in range(16):
            for x in range(16):
                t = y / 15.0
                c = lerp(top, bot, t)
                if mat == "gem":  # faceted: diagonal lighter band + radial falloff
                    d = math.hypot(x - 7.5, y - 7.5) / 10.0
                    c = lerp(cols["gem"][0], cols["gem"][1], min(1.0, d + 0.1))
                    if (x + y) % 8 < 2:
                        c = lerp(c, (255, 255, 255), 0.35)
                else:  # brushed metal streaks + sparkle
                    s = (math.sin(x * 1.7 + y * 0.35) * 0.5 + 0.5) * 0.08
                    c = lerp(c, (255, 255, 255), s)
                    if rng.random() < 0.04:
                        c = lerp(c, (255, 255, 235), 0.6)
                px[ox + x, oy + y] = c + (255,)
    return img


def swatch_avg(name, mat):
    top, bot = swatch_colors(KEYS[name][1])[mat]
    return lerp(top, bot, 0.5)


# --------------------------------------------------------------------------------------
# Model JSON
# --------------------------------------------------------------------------------------
FACES = ("north", "east", "south", "west", "up", "down")


def element_json(box):
    sx, sy = SWATCH[box.mat]
    uv = [sx + 1, sy + 1, sx + 15, sy + 15]
    e = {"from": [round(v, 3) for v in box.a], "to": [round(v, 3) for v in box.b]}
    if box.rot:
        ax, ang, o = box.rot
        e["rotation"] = {"angle": ang, "axis": ax, "origin": [round(v, 3) for v in o]}
    e["faces"] = {f: {"uv": uv, "texture": "#0"} for f in FACES}
    return e


def write_model(name):
    boxes = key_boxes(name)
    model = {
        "credit": "Procedurally generated for the Fairy Tail mod (tools/gen_assets.py). Open in Blockbench as Java Block/Item.",
        "texture_size": [32, 32],
        "textures": {"0": f"fairytail:item/key_{name}", "particle": f"fairytail:item/key_{name}"},
        "elements": [element_json(b) for b in boxes],
        "display": {
            "thirdperson_righthand": {"rotation": [0, 0, 0], "translation": [0, 3, 1], "scale": [0.9, 0.9, 0.9]},
            "thirdperson_lefthand": {"rotation": [0, 0, 0], "translation": [0, 3, 1], "scale": [0.9, 0.9, 0.9]},
            "firstperson_righthand": {"rotation": [0, 60, 15], "translation": [0, 4, 2], "scale": [0.85, 0.85, 0.85]},
            "firstperson_lefthand": {"rotation": [0, 60, 15], "translation": [0, 4, 2], "scale": [0.85, 0.85, 0.85]},
            "ground": {"rotation": [0, 0, 20], "translation": [0, 3, 0], "scale": [0.7, 0.7, 0.7]},
            "fixed": {"rotation": [0, 90, 20], "translation": [0, 0, 0], "scale": [0.9, 0.9, 0.9]},
            "head": {"rotation": [0, 0, 0], "translation": [0, 0, 0], "scale": [1, 1, 1]},
            "gui": {"rotation": [20, 40, 0], "translation": [0, 0, 0], "scale": [0.8, 0.8, 0.8]},
        },
    }
    with open(out("models", "item", "hand", f"key_{name}.json"), "w") as f:
        json.dump(model, f, indent=1)
    # inventory model (flat, high-res generated icon)
    inv = {"parent": "minecraft:item/generated", "textures": {"layer0": f"fairytail:item/icon_{name}"}}
    with open(out("models", "item", f"key_{name}.json"), "w") as f:
        json.dump(inv, f, indent=1)
    make_swatch_texture(name).save(out("textures", "item", f"key_{name}.png"))
    return boxes


# --------------------------------------------------------------------------------------
# Icon renderer: tiny orthographic software rasteriser over the same cuboids
# --------------------------------------------------------------------------------------
def rot_matrix(axis, deg):
    a = math.radians(deg)
    c, s = math.cos(a), math.sin(a)
    if axis == "x":
        return np.array([[1, 0, 0], [0, c, -s], [0, s, c]])
    if axis == "y":
        return np.array([[c, 0, s], [0, 1, 0], [-s, 0, c]])
    return np.array([[c, -s, 0], [s, c, 0], [0, 0, 1]])


CUBE_FACES = {  # face -> (corner indices, outward normal)
    "south": ((4, 5, 7, 6), (0, 0, 1)), "north": ((1, 0, 2, 3), (0, 0, -1)),
    "east": ((5, 1, 3, 7), (1, 0, 0)), "west": ((0, 4, 6, 2), (-1, 0, 0)),
    "up": ((2, 6, 7, 3), (0, 1, 0)), "down": ((0, 1, 5, 4), (0, -1, 0)),
}


def corners(box):
    (x1, y1, z1), (x2, y2, z2) = box.a, box.b
    pts = np.array([[x, y, z] for x in (x1, x2) for y in (y1, y2) for z in (z1, z2)], dtype=float)
    # index = x*4 + y*2 + z  -> 0:(-,-,-) 1:(-,-,+) ... map to my CUBE_FACES numbering below
    return pts


def render_icon(name, size=64, ss=8):
    boxes = key_boxes(name)
    R = rot_matrix("x", -10) @ rot_matrix("y", 16) @ rot_matrix("z", -40)
    light = np.array([-0.35, 0.8, 0.55]); light /= np.linalg.norm(light)
    polys = []
    for box in boxes:
        pts = corners(box)
        if box.rot:
            ax, ang, o = box.rot
            M = rot_matrix(ax, ang)
            pts = (pts - o) @ M.T + o
        pts = (pts - np.array([8, 8, 8])) @ R.T
        # corners are ordered x-major: idx = xi*4 + yi*2 + zi
        faces = {
            "south": ((1, 5, 7, 3), (0, 0, 1)), "north": ((4, 0, 2, 6), (0, 0, -1)),
            "east": ((5, 4, 6, 7), (1, 0, 0)), "west": ((0, 1, 3, 2), (-1, 0, 0)),
            "up": ((2, 3, 7, 6), (0, 1, 0)), "down": ((0, 4, 5, 1), (0, -1, 0)),
        }
        base = np.array(swatch_avg(name, box.mat), dtype=float)
        for fname, (idx, n) in faces.items():
            n = np.array(n, dtype=float)
            if box.rot:
                n = rot_matrix(*box.rot[:2]) @ n
            n = R @ n
            if n[2] <= 0:
                continue
            shade = 0.45 + 0.75 * max(0.0, float(n @ light))
            col = np.clip(base * shade, 0, 255)
            # bright rim on upward facing edges to mimic metal highlights
            depth = float(pts[list(idx)][:, 2].mean())
            polys.append((depth, [tuple(pts[i][:2]) for i in idx], tuple(int(c) for c in col), n[1] > 0.5))
    polys.sort(key=lambda p: p[0])
    # fit into frame
    allp = np.array([p for poly in polys for p in poly[1]])
    mn, mx = allp.min(axis=0), allp.max(axis=0)
    span = max(mx - mn)
    scale = (size * ss * 0.88) / span
    off = np.array([size * ss / 2, size * ss / 2]) - (mn + mx) / 2 * scale
    img = Image.new("RGBA", (size * ss, size * ss), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    for _, pts, col, top in polys:
        p = [(x * scale + off[0], size * ss - (y * scale + off[1])) for x, y in pts]
        d.polygon(p, fill=col + (255,))
        edge = tuple(min(255, int(c * 1.18 + 14)) for c in col)
        d.line(p + [p[0]], fill=edge + (255,), width=max(1, ss // 4))
    img = img.resize((size, size), Image.LANCZOS)
    # dark outline for readability
    alpha = img.split()[3]
    grown = alpha.filter(ImageFilter.MaxFilter(3))
    outline = Image.new("RGBA", img.size, (58, 34, 6, 255))
    outline.putalpha(grown)
    final = Image.alpha_composite(outline, img)
    # soft golden sparkle
    px = final.load()
    rng = random.Random(name + "spark")
    for _ in range(3):
        sx, sy = rng.randint(6, size - 7), rng.randint(6, size - 7)
        if px[sx, sy][3] > 200:
            for dx, dy, a in ((0, 0, 255), (1, 0, 150), (-1, 0, 150), (0, 1, 150), (0, -1, 150)):
                px[sx + dx, sy + dy] = (255, 255, 235, a)
    return final


# --------------------------------------------------------------------------------------
# Shared textures: glyphs, gate ring, rays, glow, particles
# --------------------------------------------------------------------------------------
def glow_tex(n, power=2.2):
    y, x = np.mgrid[0:n, 0:n]
    r = np.hypot(x - (n - 1) / 2, y - (n - 1) / 2) / (n / 2)
    a = np.clip(1 - r, 0, 1) ** power
    arr = np.zeros((n, n, 4), np.uint8)
    arr[..., :3] = 255
    arr[..., 3] = (a * 255).astype(np.uint8)
    return Image.fromarray(arr)


def star_tex(n, arm=0.5, sharp=6.0, glow=0.25):
    y, x = np.mgrid[0:n, 0:n]
    dx, dy = (x - (n - 1) / 2) / (n / 2), (y - (n - 1) / 2) / (n / 2)
    r = np.hypot(dx, dy)
    cross = np.maximum(np.exp(-np.abs(dx) * n * 0.35) * np.clip(1 - np.abs(dy) / arm * 1.0, 0, 1) ** 0 * (np.abs(dy) < 1),
                       np.exp(-np.abs(dy) * n * 0.35) * (np.abs(dx) < 1))
    spoke = np.exp(-np.abs(dx) * sharp * 3) * np.clip(1 - np.abs(dy), 0, 1) ** 1.5
    spoke2 = np.exp(-np.abs(dy) * sharp * 3) * np.clip(1 - np.abs(dx), 0, 1) ** 1.5
    core = np.clip(1 - r, 0, 1) ** 2.5
    a = np.clip(np.maximum(spoke, spoke2) + core * (0.6 + glow), 0, 1)
    arr = np.zeros((n, n, 4), np.uint8)
    arr[..., :3] = 255
    arr[..., 3] = (a * 255).astype(np.uint8)
    return Image.fromarray(arr)


def glyph_tex(cp, n=256):
    font = ImageFont.truetype(FONT, int(n * 0.78))
    img = Image.new("L", (n, n), 0)
    d = ImageDraw.Draw(img)
    ch = chr(cp)
    bb = d.textbbox((0, 0), ch, font=font)
    d.text(((n - (bb[2] - bb[0])) / 2 - bb[0], (n - (bb[3] - bb[1])) / 2 - bb[1]), ch, font=font, fill=255)
    core = img.filter(ImageFilter.MaxFilter(3))
    halo = img.filter(ImageFilter.GaussianBlur(n / 28))
    a = np.clip(np.array(core, float) / 255 + np.array(halo, float) / 255 * 0.22, 0, 1)
    arr = np.zeros((n, n, 4), np.uint8)
    arr[..., :3] = 255
    arr[..., 3] = (a * 255).astype(np.uint8)
    return Image.fromarray(arr)


def gate_ring_tex(n=512):
    S = 2
    big = n * S
    img = Image.new("L", (big, big), 0)
    d = ImageDraw.Draw(img)
    c = big / 2

    def circ(r, w, fill=255):
        d.ellipse([c - r, c - r, c + r, c + r], outline=fill, width=int(w))

    circ(c * 0.97, 6 * S); circ(c * 0.93, 2 * S); circ(c * 0.70, 3 * S); circ(c * 0.66, 1.5 * S)
    circ(c * 0.46, 3 * S); circ(c * 0.30, 2 * S)
    # tick marks between r=0.70 and 0.93 are the glyph track: draw 72 ticks on the inner ring
    for i in range(72):
        a = math.radians(i * 5)
        r0, r1 = c * (0.70 if i % 6 else 0.66), c * (0.745 if i % 6 else 0.78)
        d.line([c + r0 * math.cos(a), c + r0 * math.sin(a), c + r1 * math.cos(a), c + r1 * math.sin(a)], fill=255, width=2 * S)
    # all 12 zodiac glyphs around the gate
    font = ImageFont.truetype(FONT, int(n * 0.105) * S)
    for i in range(12):
        a = math.radians(i * 30 - 90)
        gx, gy = c + c * 0.815 * math.cos(a), c + c * 0.815 * math.sin(a)
        tile = Image.new("L", (160 * S, 160 * S), 0)
        td = ImageDraw.Draw(tile)
        ch = chr(0x2648 + i)
        bb = td.textbbox((0, 0), ch, font=font)
        td.text(((160 * S - (bb[2] - bb[0])) / 2 - bb[0], (160 * S - (bb[3] - bb[1])) / 2 - bb[1]), ch, font=font, fill=255)
        tile = tile.rotate(-(i * 30), resample=Image.BICUBIC)
        img.paste(255, (int(gx - 80 * S), int(gy - 80 * S)), tile)
    # {12/5} star polygon + hexagram inside
    pts = [(c + c * 0.46 * math.cos(math.radians(i * 30 - 90)), c + c * 0.46 * math.sin(math.radians(i * 30 - 90))) for i in range(12)]
    for i in range(12):
        d.line([pts[i], pts[(i + 5) % 12]], fill=255, width=2 * S)
    hp = [(c + c * 0.30 * math.cos(math.radians(i * 60 - 90)), c + c * 0.30 * math.sin(math.radians(i * 60 - 90))) for i in range(6)]
    for i in range(6):
        d.line([hp[i], hp[(i + 2) % 6]], fill=255, width=2 * S)
    # tiny stars on the outer band
    for i in range(24):
        a = math.radians(i * 15 + 7.5)
        sx, sy = c + c * 0.895 * math.cos(a), c + c * 0.895 * math.sin(a)
        d.ellipse([sx - 3 * S, sy - 3 * S, sx + 3 * S, sy + 3 * S], fill=255)
    # central 8-point starburst
    for i in range(8):
        a = math.radians(i * 45)
        L = c * (0.2 if i % 2 else 0.26)
        d.polygon([(c, c), (c + L * math.cos(a - 0.18), c + L * math.sin(a - 0.18)),
                   (c + L * 1.0 * math.cos(a), c + L * math.sin(a)), (c + L * math.cos(a + 0.18), c + L * math.sin(a + 0.18))], fill=255)
    img = img.resize((n, n), Image.LANCZOS)
    halo = img.filter(ImageFilter.GaussianBlur(n / 120))
    a = np.clip(np.array(img, float) / 255 + np.array(halo, float) / 255 * 0.6, 0, 1)
    arr = np.zeros((n, n, 4), np.uint8)
    arr[..., :3] = 255
    arr[..., 3] = (a * 255).astype(np.uint8)
    return Image.fromarray(arr)


def _finish(img, n):
    img = img.resize((n, n), Image.LANCZOS)
    halo = img.filter(ImageFilter.GaussianBlur(n / 120))
    a = np.clip(np.array(img, float) / 255 + np.array(halo, float) / 255 * 0.6, 0, 1)
    arr = np.zeros((n, n, 4), np.uint8)
    arr[..., :3] = 255
    arr[..., 3] = (a * 255).astype(np.uint8)
    return Image.fromarray(arr)


def gate_outer_tex(n=512):
    """Outer ring shared by every key: all twelve zodiac signs on a tick-marked band."""
    S = 2
    big = n * S
    img = Image.new("L", (big, big), 0)
    d = ImageDraw.Draw(img)
    c = big / 2

    def circ(r, w):
        d.ellipse([c - r, c - r, c + r, c + r], outline=255, width=int(w))

    circ(c * 0.97, 6 * S); circ(c * 0.93, 2 * S); circ(c * 0.70, 3 * S); circ(c * 0.66, 1.5 * S)
    for i in range(72):
        a = math.radians(i * 5)
        r0, r1 = c * (0.70 if i % 6 else 0.66), c * (0.745 if i % 6 else 0.78)
        d.line([c + r0 * math.cos(a), c + r0 * math.sin(a), c + r1 * math.cos(a), c + r1 * math.sin(a)], fill=255, width=2 * S)
    font = ImageFont.truetype(FONT, int(n * 0.105) * S)
    for i in range(12):
        a = math.radians(i * 30 - 90)
        gx, gy = c + c * 0.815 * math.cos(a), c + c * 0.815 * math.sin(a)
        tile = Image.new("L", (160 * S, 160 * S), 0)
        td = ImageDraw.Draw(tile)
        ch = chr(0x2648 + i)
        bb = td.textbbox((0, 0), ch, font=font)
        td.text(((160 * S - (bb[2] - bb[0])) / 2 - bb[0], (160 * S - (bb[3] - bb[1])) / 2 - bb[1]), ch, font=font, fill=255)
        tile = tile.rotate(-(i * 30), resample=Image.BICUBIC)
        img.paste(255, (int(gx - 80 * S), int(gy - 80 * S)), tile)
    for i in range(24):
        a = math.radians(i * 15 + 7.5)
        sx, sy = c + c * 0.895 * math.cos(a), c + c * 0.895 * math.sin(a)
        d.ellipse([sx - 3 * S, sy - 3 * S, sx + 3 * S, sy + 3 * S], fill=255)
    return _finish(img, n)


def gate_design_tex(n=512):
    """The gate's inner circle, after the user-supplied anime frame: thick ring, band of rune marks, and a
    disc split into dark wedges (aperture blades) around a bright centre. Used for ALL keys."""
    S = 2
    big = n * S
    img = Image.new("L", (big, big), 0)
    d = ImageDraw.Draw(img)
    c = big / 2
    rng = random.Random(77)

    def pt(r, a):
        return (c + r * math.cos(a), c + r * math.sin(a))

    def circ(r, w, v=255):
        d.ellipse([c - r, c - r, c + r, c + r], outline=v, width=int(w))

    circ(c * 0.97, 12 * S); circ(c * 0.86, 3 * S); circ(c * 0.64, 4 * S)
    for i in range(96):  # rune band: short bent strokes
        a = math.radians(i * 3.75)
        r0, r1 = c * rng.uniform(0.68, 0.72), c * rng.uniform(0.78, 0.84)
        mid = pt((r0 + r1) / 2, a + rng.uniform(-0.03, 0.03))
        d.line([pt(r0, a), mid, pt(r1, a + rng.uniform(-0.04, 0.04))], fill=255, width=3 * S)
        if i % 3 == 0:
            q = pt((r0 + r1) / 2, a + 0.02)
            d.ellipse([q[0] - 3 * S, q[1] - 3 * S, q[0] + 3 * S, q[1] + 3 * S], fill=255)
    d.ellipse([c - c * 0.58, c - c * 0.58, c + c * 0.58, c + c * 0.58], fill=80)  # translucent disc
    for i in range(6):  # dark aperture blades cut out of it
        a = math.radians(i * 60 + 10)
        d.polygon([pt(c * 0.58, a - 0.34), pt(c * 0.58, a + 0.34), pt(c * 0.17, a + 0.12), pt(c * 0.17, a - 0.12)], fill=0)
        d.line([pt(c * 0.58, a - 0.34), pt(c * 0.17, a - 0.12), pt(c * 0.17, a + 0.12), pt(c * 0.58, a + 0.34)], fill=255, width=3 * S)
    circ(c * 0.17, 5 * S)
    return _finish(img, n)


def spikes_tex(n=512):
    """Bold, curved, tapering light spikes radiating from the centre (burst layer)."""
    S = 2
    big = n * S
    img = Image.new("L", (big, big), 0)
    d = ImageDraw.Draw(img)
    c = big / 2
    for k in range(16):
        base = math.radians(k * 22.5)
        L = c * (0.98 if k % 2 == 0 else 0.7)
        W = 0.16 if k % 2 == 0 else 0.11
        left, right = [], []
        for i in range(41):
            u = i / 40
            r = c * 0.12 + (L - c * 0.12) * u
            bend = 0.35 * u * u * (1 if k % 4 < 2 else -1)
            w = W * (1 - u) ** 0.8
            left.append((c + r * math.cos(base + bend + w), c + r * math.sin(base + bend + w)))
            right.append((c + r * math.cos(base + bend - w), c + r * math.sin(base + bend - w)))
        d.polygon(left + right[::-1], fill=255)
    img = img.filter(ImageFilter.GaussianBlur(big / 150)).resize((n, n), Image.LANCZOS)
    out = np.zeros((n, n, 4), np.uint8)
    out[..., :3] = 255
    out[..., 3] = np.array(img)
    return Image.fromarray(out)


def rays_tex(n=512, count=36, seed=7):
    rng = random.Random(seed)
    y, x = np.mgrid[0:n, 0:n]
    dx, dy = (x - (n - 1) / 2) / (n / 2), (y - (n - 1) / 2) / (n / 2)
    r = np.hypot(dx, dy)
    th = np.arctan2(dy, dx)
    a = np.zeros_like(r)
    for _ in range(count):
        ang = rng.uniform(-math.pi, math.pi)
        width = rng.uniform(0.012, 0.05)
        length = rng.uniform(0.55, 1.0)
        dth = np.abs((th - ang + math.pi) % (2 * math.pi) - math.pi)
        a = np.maximum(a, np.exp(-(dth / width) ** 2) * np.clip(1 - r / length, 0, 1) ** 1.2 * rng.uniform(0.5, 1.0))
    a *= np.clip(r * 6, 0, 1)
    a = np.maximum(a, np.clip(1 - r * 1.6, 0, 1) ** 2)
    arr = np.zeros((n, n, 4), np.uint8)
    arr[..., :3] = 255
    arr[..., 3] = (np.clip(a, 0, 1) * 255).astype(np.uint8)
    return Image.fromarray(arr)


def premult(img):
    """RenderType.eyes blends with (ONE, ONE): alpha is ignored, so effect textures store RGB * alpha."""
    a = np.array(img).astype(np.float32)
    al = a[..., 3:4] / 255.0
    a[..., :3] *= al
    a[..., 3] = 255
    return Image.fromarray(a.astype(np.uint8))


def make_textures():
    for name in KEYS:
        write_model(name)
        render_icon(name).save(out("textures", "item", f"icon_{name}.png"))
        premult(glyph_tex(KEYS[name][0])).save(out("textures", "gui", f"glyph_{name}.png"))
    premult(gate_outer_tex()).save(out("textures", "effect", "gate_outer.png"))
    premult(rays_tex()).save(out("textures", "effect", "rays.png"))
    premult(glow_tex(128)).save(out("textures", "effect", "glow.png"))
    premult(star_tex(128, sharp=3.0, glow=0.5)).save(out("textures", "effect", "flare.png"))
    premult(gate_design_tex()).save(out("textures", "effect", "gate_design.png"))
    premult(spikes_tex()).save(out("textures", "effect", "spikes.png"))
    premult(spikes_tex()).save(out("textures", "effect", "spikes.png"))
    # particle sprites
    star_tex(32).save(out("textures", "particle", "sparkle.png"))
    star_tex(64, sharp=3.0, glow=0.5).save(out("textures", "particle", "flare.png"))
    glow_tex(16, 1.6).save(out("textures", "particle", "mote.png"))
    for n_ in ("sparkle", "flare", "mote"):
        with open(out("particles", f"{n_}.json"), "w") as f:
            json.dump({"textures": [f"fairytail:{n_}"]}, f, indent=1)


# --------------------------------------------------------------------------------------
# Sound synthesis (original, procedural)
# --------------------------------------------------------------------------------------
SR = 44100


def tarr(d):
    return np.arange(int(SR * d)) / SR


def bell(f, d, amp=1.0, decay=3.0):
    t = tarr(d)
    ratios = [(0.5, 0.35, 1.0), (1.0, 1.0, 1.0), (1.19, 0.5, 1.4), (1.56, 0.3, 1.6), (2.0, 0.55, 1.8), (2.74, 0.35, 2.4), (3.76, 0.2, 3.2), (5.4, 0.12, 4.5)]
    s = np.zeros_like(t)
    for r, a, dk in ratios:
        s += a * np.sin(2 * np.pi * f * r * t) * np.exp(-t * decay * dk)
    s *= np.minimum(1, t / 0.002)
    return s * amp


def place(buf, snd, at):
    i = int(at * SR)
    n = min(len(snd), len(buf) - i)
    if n > 0:
        buf[i:i + n] += snd[:n]


def reverb(x, tail=1.2, mix=0.28, seed=3):
    rng = np.random.default_rng(seed)
    n = int(SR * tail)
    ir = rng.standard_normal(n) * np.exp(-np.arange(n) / SR * 4.2)
    ir[0] = 0
    L = len(x) + n
    wet = np.fft.irfft(np.fft.rfft(x, L) * np.fft.rfft(ir, L), L)
    wet /= (np.abs(wet).max() + 1e-9) / (np.abs(x).max() + 1e-9)
    y = np.zeros(L)
    y[:len(x)] += x * (1 - mix)
    y += wet * mix
    return y


def norm(x, peak=0.85, fade=0.02):
    x = x / (np.abs(x).max() + 1e-9) * peak
    n = int(SR * fade)
    x[:n] *= np.linspace(0, 1, n)
    x[-n:] *= np.linspace(1, 0, n)
    return x


def snd_key_jingle():
    buf = np.zeros(int(SR * 1.1))
    rng = np.random.default_rng(1)
    for at, f, a in ((0.0, 2450, 1.0), (0.11, 3150, 0.8), (0.23, 2750, 0.9), (0.34, 3500, 0.55), (0.5, 2900, 0.35)):
        t = tarr(0.5)
        s = np.zeros_like(t)
        for r, am, dk in ((1, 1, 9), (2.76, 0.6, 14), (5.4, 0.4, 20), (8.93, 0.25, 28)):
            s += am * np.sin(2 * np.pi * f * r * t * (1 + 0.002 * rng.standard_normal())) * np.exp(-t * dk)
        click = rng.standard_normal(len(t)) * np.exp(-t * 700) * 0.6
        place(buf, (s + click) * a, at)
    return norm(reverb(buf, 0.5, 0.18))


def snd_resonance_rise():
    d = 2.4
    t = tarr(d)
    s = np.zeros_like(t)
    for f0, a in ((220, 1.0), (330, 0.7), (440, 0.8), (554.4, 0.5), (660, 0.5)):
        ph = 2 * np.pi * np.cumsum(f0 * (1 + 0.5 * (t / d) ** 1.6) / SR)
        s += a * (np.sin(ph) + 0.35 * np.sin(2 * ph) + 0.15 * np.sin(3 * ph) + 0.2 * np.sin(ph * 1.004))
    s *= (0.5 + 0.5 * np.sin(2 * np.pi * (6 + 6 * t / d) * t)) * 0.4 + 0.6
    s *= (t / d) ** 1.5
    return norm(reverb(s, 1.0, 0.3), 0.7, 0.05)


def snd_energy_buildup():
    d = 2.2
    t = tarr(d)
    rng = np.random.default_rng(5)
    noise = rng.standard_normal(len(t))
    spec = np.fft.rfft(noise)
    freqs = np.fft.rfftfreq(len(t), 1 / SR)
    bright = np.exp(-((freqs - 5200) / 2600) ** 2) + 0.4 * np.exp(-((freqs - 1400) / 900) ** 2)
    bn = np.fft.irfft(spec * bright, len(t))
    sweep = np.sin(2 * np.pi * np.cumsum(70 * 2 ** (t / d * 2.2) / SR))
    swell = (t / d) ** 2.2
    s = (bn / np.abs(bn).max() * 0.6 + sweep * 0.9) * swell
    sp = np.zeros_like(t)
    for _ in range(60):  # accelerating sparkles
        at = d * rng.random() ** 0.5
        f = rng.uniform(3000, 8000)
        tt = tarr(0.15)
        place(sp, np.sin(2 * np.pi * f * tt) * np.exp(-tt * 40) * 0.25 * (at / d + 0.2), at)
    return norm(reverb(s + sp, 0.9, 0.22), 0.8, 0.05)


def snd_gate_open():
    d = 3.6
    buf = np.zeros(int(SR * d))
    rng = np.random.default_rng(9)
    t = tarr(1.4)
    boom = np.sin(2 * np.pi * (48 + 60 * np.exp(-t * 9)) * t) * np.exp(-t * 2.8)
    place(buf, boom * 1.0, 0.0)
    place(buf, rng.standard_normal(len(tarr(0.25))) * np.exp(-tarr(0.25) * 22) * 0.5, 0.0)
    place(buf, bell(1318.5, 3.0, 0.9, 1.1), 0.0)     # "ting"
    place(buf, bell(987.8, 3.0, 0.8, 1.0), 0.19)     # "tong"
    place(buf, bell(2637.0, 2.4, 0.35, 1.6), 0.0)
    place(buf, bell(1975.5, 2.4, 0.3, 1.5), 0.19)
    place(buf, bell(659.3, 3.2, 0.4, 0.8), 0.19)
    # rising air whoosh into the strike tail
    wn = rng.standard_normal(len(buf))
    freqs = np.fft.rfftfreq(len(buf), 1 / SR)
    wn = np.fft.irfft(np.fft.rfft(wn) * np.exp(-((freqs - 3500) / 2200) ** 2), len(buf))
    env = np.exp(-tarr(d) * 2.2) * np.minimum(1, tarr(d) / 0.02)
    buf += wn / np.abs(wn).max() * env * 0.25
    return norm(reverb(buf, 1.6, 0.35), 0.9, 0.01)


def snd_shimmer_decay():
    d = 2.8
    buf = np.zeros(int(SR * d))
    rng = np.random.default_rng(11)
    for _ in range(46):
        at = (d * 0.7) * rng.random() ** 1.6
        f = rng.choice([1568, 1760, 2093, 2349, 2637, 3136, 3520, 4186, 5274])
        dur = rng.uniform(0.3, 0.9)
        tt = tarr(dur)
        place(buf, (np.sin(2 * np.pi * f * tt) + 0.3 * np.sin(2 * np.pi * f * 2.01 * tt)) * np.exp(-tt * rng.uniform(4, 9)) * rng.uniform(0.2, 0.6) * (1 - at / d), at)
    t = tarr(d)
    pad = sum(np.sin(2 * np.pi * f * t) for f in (523.3, 659.3, 784.0, 1046.5)) * np.exp(-t * 1.8) * 0.18
    return norm(reverb(buf + pad, 1.4, 0.4), 0.75, 0.05)


def write_ogg(name, x):
    pcm = (np.clip(x, -1, 1) * 32767).astype("<i2")
    with tempfile.TemporaryDirectory() as td:
        wav = os.path.join(td, "a.wav")
        import wave
        with wave.open(wav, "wb") as w:
            w.setnchannels(1); w.setsampwidth(2); w.setframerate(SR); w.writeframes(pcm.tobytes())
        subprocess.run(["ffmpeg", "-y", "-loglevel", "error", "-i", wav, "-c:a", "libvorbis", "-q:a", "5", out("sounds", name + ".ogg")], check=True)


def make_sounds():
    sounds = {
        "key_jingle": snd_key_jingle, "resonance_rise": snd_resonance_rise,
        "energy_buildup": snd_energy_buildup, "gate_open": snd_gate_open, "shimmer_decay": snd_shimmer_decay,
    }
    manifest = {}
    for name, fn in sounds.items():
        write_ogg(name, fn())
        manifest[name] = {"sounds": [f"fairytail:{name}"], "subtitle": f"subtitles.fairytail.{name}"}
    with open(out("sounds.json"), "w") as f:
        json.dump(manifest, f, indent=1)


if __name__ == "__main__":
    what = sys.argv[1] if len(sys.argv) > 1 else "all"
    if what in ("textures", "models", "all"):
        make_textures()
    if what in ("sounds", "all"):
        make_sounds()
    print("done")
