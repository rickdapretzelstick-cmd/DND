// Procedural prop sprites (trees, bushes, boulders...) placed deterministically per tile.
import { T, hash } from "./pixel.js";

const cache = new Map();

function rgb(h) { const n = parseInt(h.slice(1), 16); return [(n >> 16) & 255, (n >> 8) & 255, n & 255]; }
function shade(c, k) { return c.map((v) => Math.max(0, Math.min(255, Math.round(v * k)))); }

const PALETTES = {
  oak: ["#2f6b2f", "#3f8a3a", "#58a948", "#86cc5e"],
  oak2: ["#33692f", "#468c3a", "#6aaa4a", "#a0d060"],
  pine: ["#1d4a3a", "#27604a", "#348060", "#4fa078"],
  sakura: ["#c0688a", "#e08aa8", "#f4b0c6", "#ffd9e4"],
  autumn: ["#9a3a1a", "#c4561e", "#e88a2a", "#f6c04a"],
  bush: ["#2c6030", "#3e8238", "#5aa244", "#86c85a"],
  berry: ["#2c6030", "#3e8238", "#5aa244", "#d8304a"],
  rock: ["#4a4d52", "#6a6e74", "#8c9096", "#b4b8bc"],
  crystal: ["#4a1f7a", "#7b2cbf", "#a85ce8", "#e0b0ff"],
};

function sprite(w, h, paint) {
  const cv = document.createElement("canvas");
  cv.width = w; cv.height = h;
  const ctx = cv.getContext("2d");
  const img = ctx.createImageData(w, h);
  paint(img.data, w, h);
  // 1px dark outline around opaque pixels
  const d = img.data, out = new Uint8ClampedArray(d);
  for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) {
    const i = (y * w + x) * 4;
    if (d[i + 3]) continue;
    const nb = (xx, yy) => xx >= 0 && yy >= 0 && xx < w && yy < h && d[(yy * w + xx) * 4 + 3];
    if (nb(x - 1, y) || nb(x + 1, y) || nb(x, y - 1) || nb(x, y + 1)) {
      out[i] = 18; out[i + 1] = 24; out[i + 2] = 28; out[i + 3] = 235;
    }
  }
  img.data.set(out);
  ctx.putImageData(img, 0, 0);
  return cv;
}

function put(d, w, x, y, c) {
  if (x < 0 || y < 0 || x >= w) return;
  const i = (y * w + x) * 4;
  d[i] = c[0]; d[i + 1] = c[1]; d[i + 2] = c[2]; d[i + 3] = 255;
}

// Lit blob: ramp index from light direction (top-left) with dithered leaf noise.
function blob(d, w, cx, cy, rx, ry, ramp, seed) {
  for (let y = Math.floor(cy - ry); y <= cy + ry; y++) for (let x = Math.floor(cx - rx); x <= cx + rx; x++) {
    const nx = (x + 0.5 - cx) / rx, ny = (y + 0.5 - cy) / ry;
    const r2 = nx * nx + ny * ny;
    if (r2 > 1) continue;
    const l = 0.5 - (nx * 0.55 + ny * 0.75) * 0.5 + (hash(x, y, seed) - 0.5) * 0.45 - r2 * 0.2;
    const k = l < 0.22 ? 0 : l < 0.5 ? 1 : l < 0.78 ? 2 : 3;
    put(d, w, x, y, rgb(ramp[k]));
  }
}

function trunk(d, w, x, y0, y1, tw, col) {
  for (let y = y0; y < y1; y++) for (let i = 0; i < tw; i++) {
    put(d, w, x + i, y, shade(col, i === 0 ? 1.15 : i === tw - 1 ? 0.7 : 0.95));
  }
}

function makeSprite(kind, v) {
  const seed = v * 31 + 7;
  switch (kind) {
    case "oak": case "sakura": case "autumn": {
      const ramp = PALETTES[kind === "oak" && v % 2 ? "oak2" : kind];
      const sp = sprite(34, 40, (d, w) => {
        trunk(d, w, 15, 26, 38, 4, kind === "sakura" ? [96, 62, 56] : [104, 72, 44]);
        blob(d, w, 17, 12, 13, 11, ramp, seed);
        blob(d, w, 10, 18, 8, 7, ramp, seed + 1);
        blob(d, w, 24, 18, 8, 7, ramp, seed + 2);
        blob(d, w, 17, 20, 10, 6, ramp, seed + 3);
      });
      return { cv: sp, ax: 17, ay: 37 };
    }
    case "pine": {
      const sp = sprite(26, 46, (d, w) => {
        trunk(d, w, 11, 36, 44, 3, [90, 62, 40]);
        for (let t = 0; t < 4; t++) {
          const y = 4 + t * 9, half = 5 + t * 2;
          for (let yy = 0; yy < 13; yy++) {
            const hw = Math.floor((yy / 12) * half) + 1;
            for (let x = 13 - hw; x <= 13 + hw; x++) {
              const l = 0.55 - ((x - 13) / half) * 0.5 - (yy / 14) * 0.2 + (hash(x, y + yy, seed) - 0.5) * 0.4;
              put(d, w, x, y + yy, rgb(PALETTES.pine[l < 0.25 ? 0 : l < 0.5 ? 1 : l < 0.75 ? 2 : 3]));
            }
          }
        }
      });
      return { cv: sp, ax: 13, ay: 44 };
    }
    case "bush": case "berry": {
      const sp = sprite(20, 14, (d, w) => {
        blob(d, w, 10, 7, 9, 6, PALETTES[kind], seed);
        blob(d, w, 6, 8, 5, 4, PALETTES[kind], seed + 1);
        blob(d, w, 14, 8, 5, 4, PALETTES[kind], seed + 2);
      });
      return { cv: sp, ax: 10, ay: 12 };
    }
    case "boulder": {
      const sp = sprite(24, 18, (d, w) => {
        blob(d, w, 12, 10, 11, 8, PALETTES.rock, seed);
        blob(d, w, 7, 12, 6, 5, PALETTES.rock, seed + 1);
      });
      return { cv: sp, ax: 12, ay: 16 };
    }
    case "crystal": {
      const sp = sprite(22, 30, (d, w) => {
        for (const [bx, top, hw] of [[11, 2, 4], [5, 12, 3], [17, 10, 3]]) {
          for (let y = top; y < 27; y++) {
            const k = Math.min(hw, Math.floor((y - top) * 0.7) + 1);
            for (let x = bx - k; x <= bx + k; x++) {
              const l = (x - bx) / k;
              put(d, w, x, y, rgb(PALETTES.crystal[l < -0.4 ? 3 : l < 0.3 ? 2 : l < 0.7 ? 1 : 0]));
            }
          }
        }
      });
      return { cv: sp, ax: 11, ay: 26 };
    }
    case "dead": {
      const sp = sprite(26, 36, (d, w) => {
        const c = [58, 44, 66];
        trunk(d, w, 12, 12, 34, 3, c);
        for (let i = 0; i < 8; i++) { put(d, w, 9 - i / 2 | 0, 14 - i, shade(c, 1.1)); put(d, w, 15 + i / 2 | 0, 18 - i, shade(c, 1.1)); }
        for (let i = 0; i < 6; i++) put(d, w, 8 + i, 22 - i, shade(c, 0.9));
      });
      return { cv: sp, ax: 13, ay: 33 };
    }
    case "reed": {
      const sp = sprite(10, 16, (d, w) => {
        for (const [x, top] of [[2, 4], [5, 1], [7, 6]]) for (let y = top; y < 14; y++) put(d, w, x, y, rgb(y < top + 3 ? "#8a6a3a" : "#5c9a4a"));
      });
      return { cv: sp, ax: 5, ay: 14 };
    }
  }
}

export function propSprite(kind, v) {
  const k = kind + v;
  let s = cache.get(k);
  if (!s) { s = makeSprite(kind, v); cache.set(k, s); }
  return s;
}

const SOLID = new Set([T.GRASS, T.FLOWERS, T.FOREST, T.HILL]);

// Props for one chunk: [{x, y, kind, v}] with tile-space foot position. Not tied to a world size.
export function chunkProps(at, cx, cy, size = 32) {
  const out = [];
  for (let y = 0; y < size; y++) for (let x = 0; x < size; x++) {
    const tx = cx * size + x, ty = cy * size + y;
    const t = at(tx, ty);
    const h = hash(tx, ty, 91), j1 = hash(tx, ty, 92), j2 = hash(tx, ty, 93);
    const px = tx + 0.15 + j1 * 0.7, py = ty + 0.55 + j2 * 0.4;
    const v = Math.floor(hash(tx, ty, 94) * 4);
    if (t === T.FOREST) {
      const nearHill = [at(tx - 2, ty), at(tx + 2, ty), at(tx, ty - 2), at(tx, ty + 2)].some((n) => n === T.HILL || n === T.PEAK || n === T.ROCK);
      const grove = hash(Math.floor(tx / 9), Math.floor(ty / 9), 95);
      const kind = nearHill ? "pine" : grove > 0.86 ? "sakura" : grove < 0.1 ? "autumn" : "oak";
      if (h < 0.78 || !nearHill) out.push({ x: px, y: py, kind, v });
      if (h > 0.8) out.push({ x: tx + (1 - j1) * 0.8, y: ty + 0.9, kind: "bush", v: v & 1 });
    } else if (t === T.GRASS || t === T.FLOWERS) {
      if (h < 0.012) out.push({ x: px, y: py, kind: "oak", v });
      else if (h < 0.035) out.push({ x: px, y: py, kind: h < 0.02 && t === T.FLOWERS ? "berry" : "bush", v: v & 1 });
    } else if (t === T.HILL) {
      if (h < 0.05) out.push({ x: px, y: py, kind: "pine", v });
      else if (h < 0.09) out.push({ x: px, y: py, kind: "boulder", v: v & 1 });
    } else if (t === T.ROCK) {
      if (h < 0.16) out.push({ x: px, y: py, kind: "boulder", v: v & 1 });
    } else if (t === T.CURSED) {
      if (h < 0.06) out.push({ x: px, y: py, kind: "dead", v: v & 1 });
      else if (h < 0.11) out.push({ x: px, y: py, kind: "crystal", v: v & 1 });
    } else if (t === T.WATER && h < 0.04) {
      const shore = [at(tx - 1, ty), at(tx + 1, ty), at(tx, ty - 1), at(tx, ty + 1)].some((n) => n === T.SAND || n === T.GRASS);
      if (shore) out.push({ x: px, y: py, kind: "reed", v: 0 });
    }
  }
  return out;
}
