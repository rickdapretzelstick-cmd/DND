// Procedural prop sprites (trees, bushes, boulders...) placed deterministically per tile.
import { T, hash } from "./pixel.js";

const cache = new Map();
function shade2(h) { return "#" + rgb(h).map((v) => Math.round(v * 0.7).toString(16).padStart(2, "0")).join(""); }
export const LIT_KINDS = { lamp: { dy: 24, r: 70 }, lantern: { dy: 12, r: 46 } };

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
      const ramp = PALETTES[kind === "oak" && v % 2 ? "oak2" : kind], T = 1.9, q = (n) => Math.round(n * T);
      const sp = sprite(q(34), q(40), (d, w) => {
        trunk(d, w, q(15), q(26), q(39), q(4), kind === "sakura" ? [96, 62, 56] : [104, 72, 44]);
        blob(d, w, q(17), q(12), q(13), q(11), ramp, seed);
        blob(d, w, q(10), q(18), q(8), q(7), ramp, seed + 1);
        blob(d, w, q(24), q(18), q(8), q(7), ramp, seed + 2);
        blob(d, w, q(17), q(20), q(10), q(6), ramp, seed + 3);
        for (let i = 0; i < 6; i++) blob(d, w, q(8 + (i * 5) % 20), q(10 + ((i * 7) % 10)), q(4), q(3.4), ramp, seed + 10 + i);
      });
      return { cv: sp, ax: q(17), ay: q(38) };
    }
    case "pine": {
      const T = 1.9, q = (n) => Math.round(n * T);
      const sp = sprite(q(26), q(46), (d, w) => {
        trunk(d, w, q(11), q(36), q(45), q(3), [90, 62, 40]);
        for (let t = 0; t < 4; t++) {
          const y = q(4 + t * 9), half = q(5 + t * 2), hgt = q(13);
          for (let yy = 0; yy < hgt; yy++) {
            const hw = Math.floor((yy / (hgt - 1)) * half) + 1;
            for (let x = q(13) - hw; x <= q(13) + hw; x++) {
              const l = 0.55 - ((x - q(13)) / half) * 0.5 - (yy / (hgt + 2)) * 0.2 + (hash(x, y + yy, seed) - 0.5) * 0.4;
              put(d, w, x, y + yy, rgb(PALETTES.pine[l < 0.25 ? 0 : l < 0.5 ? 1 : l < 0.75 ? 2 : 3]));
            }
          }
        }
      });
      return { cv: sp, ax: q(13), ay: q(45) };
    }
    case "bush": case "berry": {
      const sp = sprite(30, 21, (d, w) => {
        blob(d, w, 15, 11, 13, 9, PALETTES[kind], seed);
        blob(d, w, 9, 12, 8, 6, PALETTES[kind], seed + 1);
        blob(d, w, 21, 12, 8, 6, PALETTES[kind], seed + 2);
      });
      return { cv: sp, ax: 15, ay: 18 };
    }
    case "boulder": {
      const sp = sprite(40, 30, (d, w) => {
        blob(d, w, 20, 17, 18, 13, PALETTES.rock, seed);
        blob(d, w, 11, 21, 10, 8, PALETTES.rock, seed + 1);
        blob(d, w, 29, 22, 9, 7, PALETTES.rock, seed + 2);
      });
      return { cv: sp, ax: 20, ay: 27 };
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
    case "lamp": {
      const sp = sprite(10, 30, (d, w) => {
        trunk(d, w, 4, 8, 29, 2, [46, 50, 58]);
        trunk(d, w, 2, 28, 30, 6, [46, 50, 58]);
        for (let y = 1; y < 8; y++) for (let x = 2; x < 8; x++) put(d, w, x, y, rgb(y < 3 ? "#fff3c0" : "#ffd36a"));
        for (let x = 1; x < 9; x++) put(d, w, x, 0, rgb("#2e323a"));
      });
      return { cv: sp, ax: 5, ay: 29 };
    }
    case "lantern": {
      const sp = sprite(12, 20, (d, w) => {
        for (let y = 0; y < 3; y++) for (let x = 1 + y * 0; x < 11; x++) put(d, w, x, y, rgb(y ? "#8c9096" : "#b4b8bc"));
        for (let y = 3; y < 8; y++) for (let x = 3; x < 9; x++) put(d, w, x, y, rgb(x === 3 || x === 8 ? "#6a6e74" : "#ffcf6a"));
        for (let y = 8; y < 18; y++) for (let x = 4; x < 8; x++) put(d, w, x, y, rgb(x < 5 ? "#b4b8bc" : "#6a6e74"));
        for (let x = 2; x < 10; x++) { put(d, w, x, 18, rgb("#8c9096")); put(d, w, x, 19, rgb("#6a6e74")); }
      });
      return { cv: sp, ax: 6, ay: 19 };
    }
    case "vending": {
      const col = [["#d43a3a", "#ff7a7a"], ["#2a74d4", "#7ab0ff"], ["#2aa464", "#7affb0"]][v % 3];
      const sp = sprite(12, 20, (d, w) => {
        for (let y = 0; y < 19; y++) for (let x = 0; x < 11; x++) put(d, w, x, y, rgb(x < 1 ? col[1] : x > 9 ? shade2(col[0]) : col[0]));
        for (let y = 2; y < 11; y++) for (let x = 2; x < 9; x++) put(d, w, x, y, rgb((x + y) % 3 ? "#dff4ff" : "#9ad0ee"));
        for (let x = 2; x < 9; x++) put(d, w, x, 13, rgb("#222"));
        for (let x = 3; x < 7; x++) put(d, w, x, 16, rgb("#111"));
      });
      return { cv: sp, ax: 6, ay: 19 };
    }
    case "bench": {
      const sp = sprite(18, 11, (d, w) => {
        for (let x = 0; x < 18; x++) { put(d, w, x, 1, rgb("#9a6a3a")); put(d, w, x, 2, rgb("#9a6a3a")); put(d, w, x, 5, rgb("#b07a44")); put(d, w, x, 6, rgb("#7a4e28")); }
        for (const x of [1, 15]) for (let y = 2; y < 9; y++) { put(d, w, x, y, rgb("#3a3a42")); put(d, w, x + 1, y, rgb("#2a2a32")); }
      });
      return { cv: sp, ax: 9, ay: 9 };
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
      if (h < 0.2 && (h < 0.15 || !nearHill)) out.push({ x: px, y: py, kind, v });
      if (h > 0.93) out.push({ x: tx + (1 - j1) * 0.8, y: ty + 0.9, kind: "bush", v: v & 1 });
    } else if (t === T.GRASS || t === T.FLOWERS) {
      if (h < 0.004) out.push({ x: px, y: py, kind: "oak", v });
      else if (h < 0.014) out.push({ x: px, y: py, kind: h < 0.02 && t === T.FLOWERS ? "berry" : "bush", v: v & 1 });
    } else if (t === T.HILL) {
      if (h < 0.02) out.push({ x: px, y: py, kind: "pine", v });
      else if (h < 0.035) out.push({ x: px, y: py, kind: "boulder", v: v & 1 });
    } else if (t === T.ROCK) {
      if (h < 0.06) out.push({ x: px, y: py, kind: "boulder", v: v & 1 });
    } else if (t === T.CURSED) {
      if (h < 0.015) out.push({ x: px, y: py, kind: "dead", v: v & 1 });
      else if (h < 0.03) out.push({ x: px, y: py, kind: "crystal", v: v & 1 });
    } else if (t === T.SIDEWALK) {
      const nearRoad = [at(tx - 1, ty), at(tx + 1, ty), at(tx, ty - 1), at(tx, ty + 1)].includes(T.PAVED);
      if (nearRoad && (tx * 3 + ty * 5) % 19 === 0) out.push({ x: tx + 0.5, y: ty + 0.9, kind: "lamp", v: 0 });
      else if (h < 0.006 && at(tx, ty - 1) === T.BUILDING) out.push({ x: tx + 0.5, y: ty + 0.95, kind: "vending", v });
    } else if (t === T.PLAZA) {
      if (tx % 14 === 0 && ty % 14 === 0 && h < 0.8) out.push({ x: tx + 0.5, y: ty + 0.9, kind: "lantern", v: 0 });
      else if (h < 0.004) out.push({ x: tx + 0.5, y: ty + 0.8, kind: "bench", v: 0 });
    } else if (t === T.WATER && h < 0.015) {
      const shore = [at(tx - 1, ty), at(tx + 1, ty), at(tx, ty - 1), at(tx, ty + 1)].some((n) => n === T.SAND || n === T.GRASS);
      if (shore) out.push({ x: px, y: py, kind: "reed", v: 0 });
    }
  }
  return out;
}
