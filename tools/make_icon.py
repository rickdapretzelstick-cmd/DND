"""Draws the app icon (a gold d20 on a dark red tile) into winres/icon.png.

Run from the repo root:  python3 tools/make_icon.py
Then rebuild the Windows resources with tools/build.sh.
"""
import math
from PIL import Image, ImageDraw, ImageFont

S = 1024
img = Image.new("RGBA", (S, S), (0, 0, 0, 0))
d = ImageDraw.Draw(img)

# Background tile with a subtle vertical gradient.
tile = Image.new("RGBA", (S, S))
td = ImageDraw.Draw(tile)
for y in range(S):
    t = y / S
    td.line([(0, y), (S, y)], fill=(int(120 - 50 * t), int(28 - 10 * t), int(24 - 8 * t), 255))
mask = Image.new("L", (S, S), 0)
ImageDraw.Draw(mask).rounded_rectangle([24, 24, S - 24, S - 24], radius=200, fill=255)
img.paste(tile, (0, 0), mask)
d.rounded_rectangle([24, 24, S - 24, S - 24], radius=200, outline=(217, 167, 74, 255), width=18)

# d20 silhouette: hexagon with the classic inner triangle facets.
cx, cy, r = S / 2, S / 2 + 10, 360
hexagon = [(cx + r * math.cos(math.radians(a)), cy + r * math.sin(math.radians(a))) for a in range(-90, 270, 60)]
gold, light, dark = (240, 197, 108, 255), (250, 222, 160, 255), (176, 124, 40, 255)
d.polygon(hexagon, fill=dark)
top, ur, lr, bot, ll, ul = hexagon
inner = [(cx, cy - r * 0.55), (cx + r * 0.52, cy + r * 0.32), (cx - r * 0.52, cy + r * 0.32)]
d.polygon(inner, fill=light)
for a, b in [(ul, top), (top, ur)]:
    d.polygon([a, b, inner[0]], fill=gold)
d.polygon([ur, lr, inner[1]], fill=gold)
d.polygon([ul, ll, inner[2]], fill=gold)
d.polygon([ll, bot, inner[2]], fill=dark)
d.polygon([bot, lr, inner[1]], fill=dark)
d.polygon([bot, inner[1], inner[2]], fill=gold)
edge = (90, 50, 14, 255)
d.line(hexagon + [hexagon[0]], fill=edge, width=14, joint="curve")
for p in hexagon:
    for q in inner:
        if math.dist(p, q) < r * 1.05:
            d.line([p, q], fill=edge, width=8)
d.line(inner + [inner[0]], fill=edge, width=10, joint="curve")

font = None
for name in ["DejaVuSerif-Bold.ttf", "/usr/share/fonts/truetype/dejavu/DejaVuSerif-Bold.ttf", "georgiab.ttf"]:
    try:
        font = ImageFont.truetype(name, 190)
        break
    except OSError:
        pass
font = font or ImageFont.load_default()
d.text((cx, cy + r * 0.05), "20", font=font, fill=(110, 30, 20, 255), anchor="mm")

img.resize((256, 256), Image.LANCZOS).save("winres/icon.png")
print("wrote winres/icon.png")
