// Curse renderer. Bodies are assembled from lit parts (ellipses, tapered capsules, bevelled boxes),
// shaded per pixel from a height field (banded + dithered for a pixel-art look), then given glowing eyes,
// teeth and details. Sprites face left. `hi` renders at twice the resolution for battle scenes.
import { makeCanvas, shade, hash, outline, crisp } from "./pixel.js";

const cache = new Map();

export function curseSize(spec, hi = false) {
  const S = 16 + (spec.size || 1) * 12;
  return hi ? S * 2 : S;
}

function strSeed(str) {
  let h = 2166136261;
  for (const ch of String(str)) h = Math.imul(h ^ ch.charCodeAt(0), 16777619);
  return h >>> 0;
}

export function curseSprite(spec, id = "", frame = 0, hi = false) {
  const key = id + JSON.stringify(spec) + "|" + frame + (hi ? "h" : "");
  let c = cache.get(key);
  if (c) return c;
  const S = curseSize(spec, hi);
  const [cv, ctx] = makeCanvas(S, S);
  const seed = strSeed(id || spec.color);
  paint(ctx, S, spec, seed, frame ? 1 : 0);
  crisp(cv);
  outline(cv);
  cache.set(key, cv);
  if (cache.size > 400) cache.delete(cache.keys().next().value);
  return cv;
}

// ---------- colour helpers ----------
function rgb(hex) { const n = parseInt(hex.slice(1), 16); return [(n >> 16) & 255, (n >> 8) & 255, n & 255]; }
function lum(hex) { const [r, g, b] = rgb(hex); return (0.299 * r + 0.587 * g + 0.114 * b) / 255; }
function glowOf(spec) {
  const c = spec.color2 || "#ffd166";
  return lum(c) < 0.35 ? shade(c, 0.55) : c;
}
const BAYER = [0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5];

const MATS = {
  skin: { amp: 0.06, spec: 0.0 }, chitin: { amp: 0.07, spec: 0.5 }, slime: { amp: 0.05, spec: 0.9 }, cloth: { amp: 0.06, spec: 0 },
  metal: { amp: 0.03, spec: 1 }, bone: { amp: 0.08, spec: 0.2 }, ghost: { amp: 0.05, spec: 0 }, void: { amp: 0.04, spec: 0 }, fire: { amp: 0, spec: 0 },
};

// ---------- parts ----------
// e: ellipse, c: tapered capsule, r: bevelled rounded box. Each has height(x, y) > 0 inside (0 outside).
function mkPart(k, o) {
  const p = { ...o, k, z: o.z ?? 0, mat: o.mat || "skin", a: o.a ?? 1 };
  if (k === "e") {
    p.bb = [o.x - o.rx, o.y - o.ry, o.x + o.rx, o.y + o.ry];
    p.rz = Math.min(o.rx, o.ry) * 0.95;
    p.h = (x, y) => { const q = ((x - o.x) / o.rx) ** 2 + ((y - o.y) / o.ry) ** 2; return q >= 1 ? 0 : Math.sqrt(1 - q) * p.rz; };
  } else if (k === "c") {
    const rm = Math.max(o.r1, o.r2);
    p.bb = [Math.min(o.x1, o.x2) - rm, Math.min(o.y1, o.y2) - rm, Math.max(o.x1, o.x2) + rm, Math.max(o.y1, o.y2) + rm];
    p.rz = rm;
    const dx = o.x2 - o.x1, dy = o.y2 - o.y1, len2 = dx * dx + dy * dy || 1;
    p.h = (x, y) => {
      let t = ((x - o.x1) * dx + (y - o.y1) * dy) / len2; t = Math.max(0, Math.min(1, t));
      const r = o.r1 + (o.r2 - o.r1) * t, d = Math.hypot(x - (o.x1 + dx * t), y - (o.y1 + dy * t));
      return d >= r ? 0 : Math.sqrt(1 - (d / r) ** 2) * r;
    };
  } else {
    const bev = o.bev ?? Math.min(o.w, o.h) * 0.22;
    p.bb = [o.x - o.w / 2, o.y - o.h / 2, o.x + o.w / 2, o.y + o.h / 2];
    p.rz = bev;
    p.h = (x, y) => {
      const ax = o.w / 2 - Math.abs(x - o.x), ay = o.h / 2 - Math.abs(y - o.y);
      if (ax <= 0 || ay <= 0) return 0;
      const rad = o.rad ?? 0;
      if (rad > 0 && ax < rad && ay < rad && Math.hypot(rad - ax, rad - ay) > rad) return 0;
      return Math.min(ax, ay, bev);
    };
  }
  return p;
}

const L = (() => { const v = [-0.55, -0.65, 0.55], m = Math.hypot(...v); return v.map((x) => x / m); })();

function paint(ctx, S, spec, seed, f) {
  const k = S / 64;
  const parts = [], feats = [];
  const C = {
    spec, S, k, f, r: (i, j = 0) => hash(i, j, seed), glow: glowOf(spec),
    E: (x, y, rx, ry, col, mat, z, o = {}) => parts.push(mkPart("e", { x: x * k, y: y * k, rx: rx * k, ry: ry * k, col, mat, z, ...o })),
    K: (x1, y1, x2, y2, r1, r2, col, mat, z, o = {}) => parts.push(mkPart("c", { x1: x1 * k, y1: y1 * k, x2: x2 * k, y2: y2 * k, r1: r1 * k, r2: r2 * k, col, mat, z, ...o })),
    B: (x, y, w, h, col, mat, z, o = {}) => parts.push(mkPart("r", { x: x * k, y: y * k, w: w * k, h: h * k, col, mat, z, rad: (o.rad || 0) * k, bev: o.bev ? o.bev * k : undefined, ...o })),
    F: (fn) => feats.push(fn),
  };
  const shape = BUILDERS[spec.shape] || BUILDERS.blob;
  shape(C);
  decorate(C);
  render(ctx, S, parts, seed, spec);
  for (const fn of feats) fn(ctx, k);
}

function render(ctx, S, parts, seed, spec) {
  parts.sort((a, b) => a.z - b.z);
  const idx = new Int16Array(S * S).fill(-1);
  parts.forEach((p, i) => {
    const x0 = Math.max(0, Math.floor(p.bb[0])), y0 = Math.max(0, Math.floor(p.bb[1])), x1 = Math.min(S - 1, Math.ceil(p.bb[2])), y1 = Math.min(S - 1, Math.ceil(p.bb[3]));
    for (let y = y0; y <= y1; y++) for (let x = x0; x <= x1; x++) if (p.h(x + 0.5, y + 0.5) > 0) idx[y * S + x] = i;
  });
  const img = ctx.createImageData(S, S), d = img.data;
  const rim2 = rgb(spec.color2 || "#ffffff");
  for (let y = 0; y < S; y++) for (let x = 0; x < S; x++) {
    const pi = idx[y * S + x];
    if (pi < 0) continue;
    const p = parts[pi], m = MATS[p.mat] || MATS.skin, px = x + 0.5, py = y + 0.5;
    const h = p.h(px, py);
    const hx = p.h(px + 1, py) - p.h(px - 1, py), hy = p.h(px, py + 1) - p.h(px, py - 1);
    let nx = -hx * 0.9, ny = -hy * 0.9, nz = 1;
    const nl = Math.hypot(nx, ny, nz); nx /= nl; ny /= nl; nz /= nl;
    const dif = Math.max(0, nx * L[0] + ny * L[1] + nz * L[2]);
    const edge = Math.min(1, h / (p.rz * 0.5 + 0.001));
    let t = (0.26 + 0.88 * dif) * (0.7 + 0.3 * edge) + (hash(x, y, seed) - 0.5) * m.amp * 2;
    let col = rgb(p.col);
    if (p.mat === "fire") {
      const g = Math.max(0, Math.min(1, 1 - (py / S) * 0.9 + (hash(x, y, seed + 3) - 0.5) * 0.5));
      col = g > 0.66 ? [255, 235, 140] : g > 0.4 ? [255, 160, 40] : [210, 60, 20]; t = 0.95;
    }
    const bands = 6, v = Math.max(0, Math.min(0.999, t)) * bands, fl = Math.floor(v);
    const band = fl + (v - fl > 0.25 + BAYER[(y & 3) * 4 + (x & 3)] / 32 ? 1 : 0);
    const bt = Math.min(1, band / (bands - 1));
    const ramp = p.mat === "fire" ? 0 : (bt - 0.45) * 1.05;
    let r = col[0], g = col[1], b = col[2];
    if (ramp < 0) { const f = 1 + ramp * 1.1; r *= f * 1.02; g *= f; b *= f * 1.03; }
    else { r += (255 - r) * ramp * 0.55; g += (255 - g) * ramp * 0.55; b += (255 - b) * ramp * 0.55; }
    const rim = Math.pow(1 - nz, 2.2) * Math.max(0, nx * 0.6 + ny * 0.8);
    if (rim > 0.08 && p.mat !== "fire") { const q = Math.min(0.32, rim * 0.5); r += (rim2[0] - r) * q; g += (rim2[1] - g) * q; b += (rim2[2] - b) * q; r += 14 * rim; g += 10 * rim; }
    if (m.spec > 0) {
      const hv = [L[0], L[1], L[2] + 1], hl = Math.hypot(...hv), s = Math.pow(Math.max(0, (nx * hv[0] + ny * hv[1] + nz * hv[2]) / hl), 26) * m.spec;
      if (s > 0.55) { r = 255; g = 255; b = 255; } else if (s > 0.3) { r += (255 - r) * 0.5; g += (255 - g) * 0.5; b += (255 - b) * 0.5; }
    }
    // contact shadow where a higher part sits right next to this one
    for (const [ox, oy] of [[-1, 0], [0, -1], [1, 0], [0, 1]]) {
      const qx = x + ox, qy = y + oy;
      if (qx < 0 || qy < 0 || qx >= S || qy >= S) continue;
      const qi = idx[qy * S + qx];
      if (qi >= 0 && qi !== pi && parts[qi].z > p.z && parts[qi].a >= 1) { r *= 0.74; g *= 0.74; b *= 0.8; break; }
    }
    const o = (y * S + x) * 4;
    d[o] = Math.max(0, Math.min(255, r)); d[o + 1] = Math.max(0, Math.min(255, g)); d[o + 2] = Math.max(0, Math.min(255, b));
    d[o + 3] = p.a < 1 ? Math.round(255 * p.a * (0.55 + 0.45 * Math.min(1, h / (p.rz * 0.3 + 0.001)))) : 255;
  }
  ctx.putImageData(img, 0, 0);
}

// ---------- feature helpers (drawn on top, in sprite pixels) ----------
function R(ctx, x, y, w, h, col) { ctx.fillStyle = col; ctx.fillRect(Math.round(x), Math.round(y), Math.max(1, Math.round(w)), Math.max(1, Math.round(h))); }
function disc(ctx, cx, cy, r, col) {
  ctx.fillStyle = col;
  for (let y = -Math.ceil(r); y <= Math.ceil(r); y++) {
    const w = Math.floor(Math.sqrt(Math.max(0, r * r - y * y)) + 0.35);
    if (w >= 0) ctx.fillRect(Math.round(cx - w), Math.round(cy + y), w * 2 + 1, 1);
  }
}
function eye(ctx, k, x, y, r, glow, slit = true) {
  const R0 = Math.max(2, Math.round(r * k));
  disc(ctx, x, y, R0 + 1, "#100812");
  disc(ctx, x, y, R0, glow);
  disc(ctx, x - R0 * 0.2, y - R0 * 0.25, Math.max(1, R0 * 0.6), shade(glow, 0.45));
  if (slit) R(ctx, x - Math.max(0, R0 * 0.12), y - R0 + 1, Math.max(1, Math.round(R0 * 0.34)), R0 * 2 - 1, "#100812"); else disc(ctx, x, y, Math.max(1, R0 * 0.42), "#100812");
  R(ctx, x - R0 * 0.55, y - R0 * 0.7, Math.max(1, Math.round(R0 * 0.38)), Math.max(1, Math.round(R0 * 0.38)), "#ffffff");
}
function fang(ctx, x, y, w, h, up, col = "#f4efe2") {
  for (let i = 0; i < h; i++) { const ww = Math.max(1, Math.round(w * (1 - i / h))); R(ctx, x + (w - ww) / 2, up ? y + i : y - i, ww, 1, i === h - 1 ? shade(col, -0.25) : col); }
}
function mouth(ctx, k, x, y, w, h, kind) {
  if (kind === "none") return;
  if (kind === "stitched") {
    R(ctx, x, y, w, Math.max(1, k * 0.8), "#2b0a10");
    for (let i = 0; i < w; i += Math.max(2, Math.round(k * 2))) R(ctx, x + i, y - k, Math.max(1, k * 0.7), k * 3, "#2b0a10");
    return;
  }
  if (kind === "beak") { R(ctx, x, y, w, h * 0.5, "#e9b44c"); R(ctx, x + 1, y + h * 0.5, w - 2, h * 0.4, "#b5832f"); return; }
  const maw = kind === "maw";
  const hh = Math.max(2, Math.round(h * (maw ? 1 : 0.55)));
  R(ctx, x, y, w, hh, "#1a0508"); R(ctx, x + 1, y + hh * 0.55, Math.max(1, w - 2), Math.max(1, hh * 0.45), "#6b0f1a");
  const n = Math.max(3, Math.round(w / (k * 3)));
  for (let i = 0; i < n; i++) {
    const fx = x + 1 + i * ((w - 3) / n);
    fang(ctx, fx, y, Math.max(2, (w / n) * 0.8), Math.max(2, Math.round(hh * 0.45)), true);
    if (maw) fang(ctx, fx + w / n / 2, y + hh - 1, Math.max(2, (w / n) * 0.8), Math.max(2, Math.round(hh * 0.4)), false);
  }
}

// ---------- builders (64-unit design space, facing left) ----------
const BUILDERS = {
  humanoid(C) {
    const { spec, f, E, K, B, F, glow } = C, c = spec.color, c2 = spec.color2 || shade(c, -0.4), ex = spec.extra, b = f ? 1 : 0;
    const body = ex === "uniform" ? "#1d2230" : c2, cloth = shade(body, 0);
    const half = ex === "halfbody", veil = ex === "veil", robe = ex === "robe", kid = veil && (spec.size || 1) === 1;
    const hairC = "#17111c";
    const sw = f ? 1.2 : -0.8;
    E(32, 61.5, 15, 2.2, "#000", "void", -2, { a: 0.28 });
    if (half) {
      K(32, 50, 46, 59, 5, 2.4, "#7a1d1d", "skin", 0);
      K(25, 32, 15 + sw, 52, 2.7, 2.2, c, "skin", 1); K(39, 32, 49 - sw, 52, 2.7, 2.2, c, "skin", 1);
      E(14 + sw, 53.5, 3.2, 2.6, c, "skin", 1); E(50 - sw, 53.5, 3.2, 2.6, c, "skin", 1);
      K(32, 27 + b * 0.4, 32, 50, 8, 4.5, body, "cloth", 3);
    } else if (kid) {
      K(29.5, 48, 29, 58, 2.6, 2.3, c, "skin", 1); K(34.5, 48, 35, 58, 2.6, 2.3, c, "skin", 1);
      E(28, 59.4, 3.6, 1.8, "#e8e4da", "skin", 2); E(36, 59.4, 3.6, 1.8, "#e8e4da", "skin", 2);
      K(32, 31, 32, 49, 7.4, 10.6, "#b5262c", "cloth", 2);
      K(32, 26, 32, 40, 6.4, 6.8, "#f4f1ea", "cloth", 3);
      K(25.5, 29, 20 + sw, 45, 2.3, 1.9, "#f4f1ea", "cloth", 2); K(38.5, 29, 44 - sw, 45, 2.3, 1.9, "#f4f1ea", "cloth", 2);
      E(19.8 + sw, 46.6, 2.4, 2.8, c, "skin", 2); E(44.2 - sw, 46.6, 2.4, 2.8, c, "skin", 2);
    } else if (veil || robe) {
      K(32, 24, 32, 59.5, 8, robe ? 11 : 13.5, veil ? "#efece4" : body, "cloth", 3);
      E(32, 59.5, robe ? 11 : 13.5, 2.2, shade(veil ? "#efece4" : body, -0.25), "cloth", 3);
      K(24.5, 26, 20 + sw, 46, 2.7, 2.2, veil ? "#efece4" : body, "cloth", 2); K(39.5, 26, 44 - sw, 46, 2.7, 2.2, veil ? "#efece4" : body, "cloth", 2);
      E(19.8 + sw, 48, 2.6, 3, c, "skin", 2); E(44.2 - sw, 48, 2.6, 3, c, "skin", 2);
    } else {
      K(29, 40, 28.2, 58, 3.5, 3.1, shade(body, -0.32), "cloth", 1); K(35, 40, 35.8, 58, 3.5, 3.1, shade(body, -0.38), "cloth", 1);
      E(27.4, 59.4, 4.4, 2, "#17151c", "skin", 2); E(36.6, 59.4, 4.4, 2, "#17151c", "skin", 2);
      K(32, 24 + b * 0.4, 32, 43, 7.6, 6.3, cloth, "cloth", 3);
      E(32, 25.4, 9.6, 3.6, cloth, "cloth", 3);
      K(24, 27, 20.5 + sw, 46, 2.5, 2, c, "skin", 2); K(40, 27, 43.5 - sw, 46, 2.5, 2, c, "skin", 2);
      E(20.2 + sw, 48, 2.6, 3.1, c, "skin", 2); E(43.8 - sw, 48, 2.6, 3.1, c, "skin", 2);
    }
    const hy = kid ? 17.5 + b * 0.3 : 14.5 + b * 0.3;
    if (veil && !kid) { E(32, 22, 11.5, 17, hairC, "cloth", 1); K(24.5, 14, 24, 40, 3.2, 1.6, hairC, "cloth", 9); K(39.5, 14, 40, 40, 3.2, 1.6, hairC, "cloth", 9); }
    if (kid) { E(32, hy - 0.6, 10.6, 10, hairC, "cloth", 4); E(32, hy - 5.4, 9.8, 4.4, hairC, "cloth", 7); K(23, hy, 22.6, hy + 7, 3, 2.2, hairC, "cloth", 7); K(41, hy, 41.4, hy + 7, 3, 2.2, hairC, "cloth", 7); }
    const hairy = !veil && !kid && !["hood", "mask", "volcano", "stitches", "marks"].includes(ex || "") && lum(c) > 0.3;
    if (hairy) { E(32, hy + 6, 10.4, 15, hairC, "cloth", 1); E(32, hy - 5, 8.6, 4.2, hairC, "cloth", 7); K(24.4, hy - 2, 24, hy + 18, 3.2, 1.6, hairC, "cloth", 7); K(39.6, hy - 2, 40, hy + 18, 3.2, 1.6, hairC, "cloth", 7); }
    if (half && !hairy) K(32, 14, 31, 50, 9, 2.4, hairC, "cloth", 0);
    E(32, 22.4, 2.8, 2.4, shade(c, -0.15), "skin", 2);
    let headCol = c;
    if (ex === "volcano") headCol = "#3d2a26";
    E(32, hy, kid ? 8.4 : 7.8, kid ? 8.4 : 8.8, headCol, ex === "volcano" ? "chitin" : "skin", 5);
    if (ex === "hood") { E(32, hy - 0.8, 10.5, 11.2, shade(body, -0.15), "cloth", 6); E(32, hy + 1.4, 6.4, 6.8, "#0a0910", "void", 7); }
    if (ex === "mask") E(32, hy + 0.8, 6.9, 7.6, "#f4f1ea", "bone", 7);
    if (veil && !kid) E(32, hy - 0.6, 11.2, 12.8, "#ffffff", "ghost", 8, { a: 0.5 });
    if (veil && !kid) F((g, k) => { g.fillStyle = "rgba(255,255,255,0.55)"; for (let i = 0; i < 6; i++) R(g, (22.5 + i * 3.5) * k, (hy + 10.5 + (i % 2)) * k, 2 * k, 1.2 * k, "rgba(255,255,255,0.6)"); for (let i = 0; i < 5; i++) R(g, (24 + i * 4) * k, 32 * k, Math.max(1, k * 0.8), 26 * k, "rgba(120,110,130,0.45)"); R(g, 28 * k, 44 * k, 3 * k, 8 * k, "rgba(150,20,30,0.7)"); R(g, 35 * k, 50 * k, 2 * k, 6 * k, "rgba(150,20,30,0.6)"); });
    if (kid) F((g, k) => { R(g, 26 * k, 40 * k, 12 * k, Math.max(1, k * 0.9), "#7d1b20"); });
    if (ex === "halo") F((g, k) => { g.strokeStyle = "#ffe08a"; g.lineWidth = Math.max(1, k * 1.2); g.beginPath(); g.ellipse(32 * k, (hy - 11 + b * 0.3) * k, 8 * k, 2.4 * k, 0, 0, 7); g.stroke(); });
    // face
    const fy = (hy + 0.8) * C.k, n = spec.eyes ?? 2;
    const dark = ex === "hood" || ex === "mask";
    F((g, k) => {
      if (ex === "mask") { disc(g, 29 * k, fy - 0.2 * k, 1.5 * k, "#0b0b10"); disc(g, 35 * k, fy - 0.2 * k, 1.5 * k, "#0b0b10"); R(g, 32 * k - 1, fy + 3.2 * k, 2, 2 * k, "#c1121f"); }
      const er = kid ? 1.8 : n >= 4 ? 1.1 : 1.5;
      if (n === 1) eye(g, k, 32 * k, fy, 2.2, glow);
      else if (n >= 2) { eye(g, k, (kid ? 28 : 28.4) * k, fy, er, glow); eye(g, k, (kid ? 36 : 35.6) * k, fy, er, glow); }
      if (n >= 4) { eye(g, k, 29.2 * k, fy + 3.6 * k, 1, glow); eye(g, k, 34.8 * k, fy + 3.6 * k, 1, glow); }
      if (n >= 3 && n < 4) eye(g, k, 32 * k, fy - 4.2 * k, 1.2, glow);
      if (!dark && ex === "scissors") { R(g, 25 * k, fy + 4.6 * k, 14 * k, Math.max(1, k * 1.1), "#7a0a16"); for (let i = 0; i < 9; i++) R(g, (26 + i * 1.4) * k, fy + 3.8 * k, Math.max(1, k * 0.7), 1.8 * k, "#f4efe2"); R(g, 24 * k, fy + 3.4 * k, Math.max(1, k * 0.9), 3 * k, "#7a0a16"); R(g, 39 * k, fy + 3.4 * k, Math.max(1, k * 0.9), 3 * k, "#7a0a16"); }
      else if (!dark && spec.mouth && spec.mouth !== "none") mouth(g, k, 28.4 * k, fy + 4.6 * k, 7.2 * k, 2 * k, spec.mouth);
      if (!dark && ex === "stitches") { R(g, 26 * k, (hy - 3) * k, 12 * k, Math.max(1, k * 0.8), "#3a0a10"); for (let i = 0; i < 6; i++) R(g, (26.5 + i * 2.2) * k, (hy - 4) * k, Math.max(1, k * 0.7), 3 * k, "#3a0a10"); }
    });
    if (ex === "stitches") F((g, k) => { for (let i = 0; i < 5; i++) { R(g, (28 + i * 1.3) * k, (28 + i * 2.2) * k, 3 * k, Math.max(1, k * 0.7), "#2b0a10"); R(g, (29 + i * 1.3) * k, (27 + i * 2.2) * k, Math.max(1, k * 0.7), 3 * k, "#2b0a10"); } });
    if (ex === "uniform") F((g, k) => { for (let i = 0; i < 4; i++) R(g, 31.2 * k, (28 + i * 3.4) * k, 1.6 * k, 1.6 * k, "#f2c94c"); });
    if (ex === "scissors") F((g, k) => {
      const hx = (43.8 - sw) * k, hyy = 49 * k;
      for (const sgn of [-1, 1]) { g.fillStyle = sgn < 0 ? "#dfe3e8" : "#a9b0b8"; for (let i = 0; i < 22 * k; i++) { const t = i / (22 * k); g.fillRect(Math.round(hx + sgn * t * 3.6 * k - 1), Math.round(hyy - i - 1), Math.max(1, Math.round((1 - t) * 3 * k + 1)), 1); } }
      disc(g, hx - 4 * k, hyy + 3 * k, 2.4 * k, "#8d99ae"); disc(g, hx + 4 * k, hyy + 3 * k, 2.4 * k, "#8d99ae"); disc(g, hx - 4 * k, hyy + 3 * k, 1.2 * k, "#14101c"); disc(g, hx + 4 * k, hyy + 3 * k, 1.2 * k, "#14101c");
    });
    if (ex === "flowers") {
      const petals = ["#ffafcc", "#ffc8dd", "#cdb4db", "#ffb4a2"];
      for (let i = 0; i < 8; i++) { const a = (i / 8) * 6.283; C.E(32 + Math.cos(a) * 9, hy - 3 + Math.sin(a) * 6.8, 2.7, 2.7, petals[i % 4], "skin", 8); }
      K(30, 30, 24, 40, 1.2, 1, "#3a5a40", "skin", 4); K(34, 34, 41, 44, 1.2, 1, "#3a5a40", "skin", 4);
      E(24, 40, 2, 2, "#ffafcc", "skin", 5); E(41, 44, 2, 2, "#ffafcc", "skin", 5);
    }
    if (ex === "marks") {   // King of Curses: a second pair of arms, face stripes
      K(26, 31, 18 + sw, 50, 2.3, 1.9, c, "skin", 2); K(38, 31, 46 - sw, 50, 2.3, 1.9, c, "skin", 2);
      E(17.8 + sw, 51.6, 2.4, 2.9, c, "skin", 2); E(46.2 - sw, 51.6, 2.4, 2.9, c, "skin", 2);
      F((g, k) => { for (const x of [26.6, 36.4]) R(g, x * k, (hy - 1.5) * k, 3 * k, Math.max(1, k * 0.9), "#3a0a10"); R(g, 31.5 * k, (hy - 7.6) * k, Math.max(1, k * 1.1), 3 * k, "#3a0a10"); for (let i = 0; i < 4; i++) R(g, (25.4 + i * 0.7) * k, (hy + 3 + i * 1.2) * k, 3 * k, Math.max(1, k * 0.8), "#3a0a10"); });
    }
    if (ex === "volcano") {
      F((g, k) => { for (const [x, y, w, h] of [[28, 30, 1.2, 9], [35, 33, 1.2, 8], [24, 42, 8, 1.1], [33, 47, 6, 1.1], [41, 30, 1.1, 7], [22, 34, 1.1, 8]]) R(g, x * k, y * k, w * k, h * k, "#ff7a00"); });
      E(32, hy - 8, 4.2, 3.6, "#2c1f1c", "chitin", 6);
      for (let i = 0; i < 5; i++) C.E(26 + i * 3, hy - 8 - (i % 2) * 3, 2, 4, "#ff9100", "fire", 7);
      F((g, k) => {
        R(g, 30 * k, (hy - 4) * k, 1.4 * k, 6 * k, "#ff6d00"); R(g, 34 * k, (hy - 3) * k, 1.4 * k, 5 * k, "#ffb703"); R(g, 27 * k, (hy + 3) * k, 6 * k, 1.2 * k, "#ff6d00");
        for (let i = 0; i < 3; i++) disc(g, (31 + i * 2 - f) * k, (hy - 12 - i * 3 - f * 2) * k, (1.8 + i * 0.9) * k, `rgba(90,80,80,${0.55 - i * 0.14})`);
      });
    }
  },

  head(C) {
    const { spec, f, E, K, F, glow } = C, c = spec.color, c2 = spec.color2 || shade(c, -0.4), b = f ? 1.4 : 0;
    E(32, 59, 12, 2, "#000", "void", -2, { a: 0.25 });
    if (spec.extra === "wings") {
      const wy = f ? -6 : 0;
      for (const s of [-1, 1]) { E(32 + s * 21, 22 + wy, 14, 6, "#dff3ff", "ghost", 0, { a: 0.55 }); E(32 + s * 18, 30 + wy * 0.6, 11, 4.5, "#c8e6fa", "ghost", 0, { a: 0.5 }); }
    }
    K(32, 44, 31, 58, 4, 2, shade(c, -0.25), "skin", 1);
    E(32, 31 - b, 19, 17.5, c, "skin", 3);
    E(32, 38 - b, 14, 9, shade(c, 0.1), "skin", 4);
    E(18, 40 - b, 5, 4, shade(c, -0.1), "skin", 4); E(46, 40 - b, 5, 4, shade(c, -0.1), "skin", 4);
    if (spec.extra === "windows") for (let i = 0; i < 6; i++) C.B(18 + (i % 3) * 14, 20 + Math.floor(i / 3) * 8, 7, 5, i % 2 ? "#ffd166" : "#fff3b0", "metal", 6, { bev: 1 });
    const n = spec.eyes ?? 2, fy = (24 - b) * C.k;
    F((g, k) => {
      if (n <= 2) { eye(g, k, 24 * k, fy, 4.2, glow); eye(g, k, 40 * k, fy, 4.2, glow); }
      else { eye(g, k, 22 * k, fy + 1 * k, 3.2, glow); eye(g, k, 42 * k, fy + 1 * k, 3.2, glow); eye(g, k, 32 * k, fy - 4 * k, 3.6, glow); if (n >= 4) { eye(g, k, 28 * k, fy + 4 * k, 2, glow); eye(g, k, 36 * k, fy + 4 * k, 2, glow); } }
      mouth(g, k, 21 * k, (36 - b) * k, 22 * k, 9 * k, spec.mouth === "none" ? "grin" : spec.mouth || "grin");
      R(g, 26 * k, (30 - b) * k, 3 * k, k, shade(c, -0.4)); R(g, 36 * k, (30 - b) * k, 3 * k, k, shade(c, -0.4));
    });
  },

  blob(C) {
    const { spec, f, E, K, F, glow, r } = C, c = spec.color, c2 = spec.color2 || shade(c, -0.4), b = f ? 1.2 : 0, ex = spec.extra;
    E(32, 61, 22, 2.4, "#000", "void", -2, { a: 0.25 });
    if (ex === "tentacles" || ex === "arms") {
      for (let i = 0; i < 5; i++) {
        const bx = 14 + i * 9.2, dir = i % 2 ? 1 : -1;
        const ph = (f ? 1 : -1) * dir;
        K(bx, 50, bx + 3 * ph, 58, 3.4, 2.4, shade(c, -0.1), "slime", 1); K(bx + 3 * ph, 58, bx + 8 * ph + dir * 2, 61, 2.4, 1.1, shade(c, -0.18), "slime", 1);
      }
    }
    E(32, 44 - b, 24, 17 + b * 0.6, c, "slime", 3);
    E(18, 50, 12, 9, shade(c, -0.08), "slime", 3); E(47, 51, 11, 8.4, shade(c, -0.1), "slime", 3);
    E(26, 34 - b, 11, 8, shade(c, 0.06), "slime", 4); E(41, 36 - b, 9, 7, shade(c, 0.03), "slime", 4);
    if (ex === "arms") { K(11, 40, 5, 54, 3, 2.2, shade(c, -0.15), "slime", 4); K(53, 40, 59, 54, 3, 2.2, shade(c, -0.15), "slime", 4); }
    if (ex === "flames") for (let i = 0; i < 5; i++) C.E(16 + i * 8, 26 - (i % 2) * 5 - b * 2, 4.4, 8 + (i % 3) * 2, "#ff9100", "fire", 6);
    for (let i = 0; i < 3; i++) C.E(14 + r(i, 7) * 36, 58 - r(i, 8) * 3, 1.5, 3.2, shade(c, -0.1), "slime", 5);
    const n = Math.max(0, spec.eyes ?? 2), eyes = [];
    for (let i = 0; i < n; i++) { const cols = Math.ceil(Math.sqrt(n * 1.6)), cx = i % cols, cy = Math.floor(i / cols); eyes.push([18 + (cx + 0.5 + (r(i, 1) - 0.5) * 0.5) * (28 / cols), 30 + (cy + 0.5 + (r(i, 2) - 0.5) * 0.4) * (14 / Math.ceil(n / cols)) - b, 1.5 + r(i, 3) * 1.6]); }
    F((g, k) => {
      for (const [x, y, rr] of eyes) eye(g, k, x * k, y * k, rr, glow, false);
      if (spec.mouth === "maw") mouth(g, k, 22 * k, (48 - b) * k, 20 * k, 8 * k, "maw");
      else if (spec.mouth === "stitched") mouth(g, k, 22 * k, (50 - b) * k, 20 * k, 2, "stitched");
      else if (spec.mouth && spec.mouth !== "none") mouth(g, k, 24 * k, (49 - b) * k, 16 * k, 6 * k, spec.mouth);
    });
  },

  insect(C) {
    if (C.spec.extra === "legs") return BUILDERS.spider(C);
    const { spec, f, E, K, F, glow } = C, c = spec.color, c2 = spec.color2 || shade(c, -0.4), ex = spec.extra, b = f ? 1 : 0;
    E(34, 61, 26, 2.2, "#000", "void", -2, { a: 0.25 });
    const dark = shade(c, -0.42);
    // thin mid/front legs, big bent hind leg
    for (let i = 0; i < 2; i++) {
      const bx = 25 + i * 6, ft = bx + (f ? (i ? 3 : -3) : (i ? -3 : 3)) - 6;
      K(bx, 40, bx - 5, 33, 1.5, 1.2, dark, "chitin", 0); K(bx - 5, 33, ft, 60, 1.3, 0.8, shade(c, -0.3), "chitin", 0);
    }
    K(40, 40, 52, 22 - b * 2, 3.8, 2.8, shade(c, -0.12), "chitin", 1); K(52, 22 - b * 2, 58, 59, 2.4, 0.9, shade(c, -0.28), "chitin", 1);
    E(58, 60, 3.4, 1.4, dark, "chitin", 1);
    if (ex === "wings") for (const dy of [0, 5]) E(44, 27 - dy - (f ? 2 : 0), 17, 4.8, "#e8f6ff", "ghost", 6, { a: 0.5 });
    E(45, 38, 17.5, 8.8, c, "chitin", 2);
    for (let i = 0; i < 5; i++) K(33 + i * 5.6, 31, 34 + i * 5.6, 45, 0.9, 0.7, shade(c, -0.38), "chitin", 3, { a: 0.6 });
    E(45, 34, 14, 3.2, shade(c, 0.16), "chitin", 3);
    E(28, 36, 9, 8, shade(c, 0.07), "chitin", 3);
    E(17.5, 31.5, 8, 7, c, "chitin", 4);
    E(12.5, 29, 3.8, 4.6, c2, "chitin", 5); E(20, 28.6, 3.6, 4.2, c2, "chitin", 5);
    K(14, 25, 7, 14 - b * 2, 0.9, 0.5, dark, "chitin", 5); K(20, 25, 25, 12 - b * 2, 0.9, 0.5, dark, "chitin", 5);
    K(14, 37, 10.6, 42, 1.4, 0.8, shade(c, -0.5), "chitin", 5); K(20, 37, 21.6, 42, 1.4, 0.8, shade(c, -0.5), "chitin", 5);
    if (ex === "arms" || spec.mouth === "beak") {
      K(23, 34, 14, 25, 2.6, 1.9, shade(c, -0.08), "chitin", 5); K(14, 25, 4, 41 - b * 2, 3, 0.5, "#eaf1d6", "metal", 6);
      K(25, 39, 18, 31, 2.4, 1.8, shade(c, -0.16), "chitin", 5); K(18, 31, 8, 47 + b, 2.7, 0.5, "#d0dab6", "metal", 6);
    }
    F((g, k) => { eye(g, k, 12.8 * k, 29 * k, 2.4, glow, false); eye(g, k, 20 * k, 28.6 * k, 2, glow, false); });
  },

  beast(C) {
    if (C.spec.extra === "hands") return BUILDERS.toad(C);
    const { spec, f, E, K, F, glow } = C, c = spec.color, c2 = spec.color2 || shade(c, -0.4), ex = spec.extra, b = f ? 1 : 0;
    E(32, 61, 25, 2, "#000", "void", -2, { a: 0.25 });
    const g1 = f ? 3 : -3;
    K(48, 44, 52 - g1, 59, 4, 2.4, shade(c, -0.3), "skin", 0); K(40, 46, 41 + g1, 59, 4.4, 2.6, shade(c, -0.35), "skin", 0);
    K(60, 36, 55, 28 - b * 2, 1.6, 0.8, shade(c, -0.1), "skin", 0, { z: 0 });
    E(38, 38, 18, 10, c, "skin", 2);
    E(26, 37, 10, 10.4, shade(c, 0.06), "skin", 3);
    K(26, 46, 24 + g1, 59, 4.2, 2.6, shade(c, -0.1), "skin", 4); K(34, 46, 33 - g1, 59, 4.2, 2.6, shade(c, -0.15), "skin", 4);
    E(23 + g1, 60, 4, 2, shade(c, -0.25), "skin", 5); E(32 - g1, 60, 4, 2, shade(c, -0.3), "skin", 5);
    E(15, 32 - b, 8.2, 7.4, shade(c, 0.1), "skin", 5);
    E(8.4, 35 - b, 5.4, 4, shade(c, 0.05), "skin", 6);
    E(17, 24 - b, 2.6, 4, c2, "skin", 4); E(22, 25 - b, 2.4, 3.6, c2, "skin", 4);
    if (ex === "hands") { E(23 + g1, 59.4, 3.2, 3.2, "#f1c27d", "skin", 6); E(32 - g1, 59.4, 3.2, 3.2, "#f1c27d", "skin", 6); }
    if (ex === "crown") F((g, k) => { for (let i = 0; i < 4; i++) { R(g, (12 + i * 2.6) * k, (24 - b - (i % 2) * 2) * k, 1.8 * k, (4 + (i % 2) * 2) * k, "#ffd166"); } R(g, 12 * k, 27.4 * k - b * k, 9 * k, 1.4 * k, "#e0a800"); });
    if (ex === "ribs") F((g, k) => { for (let i = 0; i < 5; i++) { R(g, (28 + i * 3.4) * k, (32 - (i % 2) * 0.6) * k, 1.3 * k, 10 * k, "#efe9dc"); R(g, (28 + i * 3.4) * k, 41 * k, 3 * k, 1.1 * k, "#efe9dc"); } });
    F((g, k) => {
      if ((spec.eyes ?? 2) > 0) { eye(g, k, 13 * k, (31 - b) * k, 1.9, glow); eye(g, k, 18.4 * k, (30.4 - b) * k, 1.6, glow); }
      else { disc(g, 13 * k, (31 - b) * k, 2 * k, "#0b0508"); disc(g, 18 * k, (30.4 - b) * k, 1.7 * k, "#0b0508"); R(g, 13 * k, (31 - b) * k, 1, 1, "#ff6d00"); }
      R(g, 4.2 * k, (33.6 - b) * k, 1.8 * k, 1.6 * k, "#16101a");
      if (spec.mouth === "maw" || spec.mouth === "grin") mouth(g, k, 5 * k, (37 - b) * k, 9 * k, 5 * k, "maw");
    });
  },

  spider(C) {
    const { spec, f, E, K, F, glow, r } = C, c = spec.color, c2 = spec.color2 || shade(c, -0.4), b = f ? 1 : 0;
    E(32, 61, 29, 2.2, "#000", "void", -2, { a: 0.28 });
    const dark = shade(c, -0.4);
    for (let side = 0; side < 2; side++) for (let i = 0; i < 4; i++) {
      const sx = side ? 1 : -1, bx = 32 + sx * 5, by = 40 - i * 1.4, spread = 12 + i * 3 - (i === 3 ? 3 : 0), wob = (f ? 1 : -1) * (i % 2 ? 1.2 : -1.2);
      const kx = bx + sx * spread * 0.6, ky = 20 + i * 3 - b, fx = bx + sx * (spread + 10) + wob, fy = 60;
      K(bx, by, kx, ky, 2, 1.5, dark, "chitin", side ? 1 : 0); K(kx, ky, fx, fy, 1.5, 0.7, shade(c, -0.25), "chitin", side ? 1 : 0);
    }
    E(40, 40, 15.5, 12.5, c, "chitin", 2);
    E(40, 36, 9, 5, shade(c, 0.14), "chitin", 3);
    for (let i = 0; i < 5; i++) E(36 + (i % 3) * 5, 32 + Math.floor(i / 3) * 6, 1.2, 1.2, c2, "chitin", 4);
    E(24, 38, 9, 8, shade(c, 0.05), "chitin", 3);
    E(17, 37, 6.4, 5.6, c, "chitin", 4);
    K(14, 41, 12, 47, 1.3, 0.7, shade(c, -0.5), "chitin", 5); K(20, 41, 22, 47, 1.3, 0.7, shade(c, -0.5), "chitin", 5);
    F((g, k) => { const n = Math.min(8, spec.eyes ?? 8); for (let i = 0; i < n; i++) eye(g, k, (12 + (i % 4) * 3.4) * k, (33.4 + Math.floor(i / 4) * 3.2 - (i % 2)) * k, i < 2 ? 1.5 : 1, glow, false); });
  },

  toad(C) {
    const { spec, f, E, K, F, glow, r } = C, c = spec.color, c2 = spec.color2 || shade(c, -0.4), b = f ? 1.2 : 0;
    E(32, 61, 27, 2.2, "#000", "void", -2, { a: 0.28 });
    E(14, 58, 6, 2.8, "#f1c27d", "skin", 0); E(50, 58, 6, 2.8, "#f1c27d", "skin", 0);
    K(18, 50, 12, 58, 6, 4, shade(c, -0.2), "slime", 1); K(46, 50, 53, 58, 6, 4, shade(c, -0.25), "slime", 1);
    E(32, 44 - b, 26, 14.5 + b * 0.6, c, "slime", 2);
    E(32, 50, 17, 8, shade(c, 0.22), "slime", 3);
    for (let i = 0; i < 9; i++) E(14 + r(i, 1) * 36, 33 + r(i, 2) * 8 - b, 1.7 + r(i, 3) * 1.5, 1.4 + r(i, 3), shade(c, r(i, 4) > 0.5 ? -0.22 : 0.12), "slime", 4);
    E(22, 29 - b, 9, 8.4, shade(c, 0.05), "slime", 5); E(42, 29 - b, 9, 8.4, shade(c, 0.02), "slime", 5);
    E(16, 52, 4.2, 4.4, "#f1c27d", "skin", 4); E(48, 52, 4.2, 4.4, "#f1c27d", "skin", 4);
    for (const x of [10, 22, 42, 54]) E(x, 33, 1.2, 3, "#f1c27d", "skin", 6, { a: 0 });
    F((g, k) => {
      eye(g, k, 22 * k, (29 - b) * k, 3.6, glow, true); eye(g, k, 42 * k, (29 - b) * k, 3.6, glow, true);
      R(g, 10 * k, (42 - b) * k, 44 * k, Math.max(1, 1.6 * k), "#241018");
      for (let i = 0; i < 7; i++) fang(g, (14 + i * 5.6) * k, (42 - b) * k, 3 * k, 3 * k, true);
      for (const x of [10, 20, 44, 54]) for (let j = 0; j < 4; j++) R(g, (x + j * 1.7) * k, 56 * k, Math.max(1, k * 0.9), 4 * k, "#b98a52");
    });
  },

  serpent(C) {
    const { spec, f, E, K, B, F, glow } = C, c = spec.color, c2 = spec.color2 || shade(c, -0.4), ex = spec.extra;
    E(32, 61, 28, 2, "#000", "void", -2, { a: 0.25 });
    if (ex === "windows") return BUILDERS.train(C);
    const seg = ex === "windows" ? 6 : 13, pts = [];
    for (let i = 0; i <= seg; i++) {
      const t = i / seg;
      pts.push([52 - t * 38, 52 - Math.sin(t * Math.PI * 1.6 + (f ? 0.7 : 0)) * 6 * (ex === "windows" ? 0.25 : 1) - t * (ex === "windows" ? 4 : 18), 3.4 + t * (ex === "windows" ? 4.5 : 3.8)]);
    }
    const rainbow = ["#ff006e", "#fb5607", "#ffbe0b", "#06d6a0", "#3a86ff", "#8338ec"];
    pts.forEach(([x, y, r], i) => {
      const col = ex === "rainbow" ? rainbow[i % 6] : ex === "windows" ? (i % 2 ? c : shade(c, -0.12)) : i % 2 ? c : shade(c, -0.14);
      if (ex === "windows") { B(x, y, 12, 11, col, "metal", i, { rad: 2.2, bev: 2 }); }
      else E(x, y, r * 1.18, r, col, "skin", i);
    });
    if (ex === "windows") {
      pts.forEach(([x, y], i) => F((g, k) => { R(g, (x - 3.5) * k, (y - 3) * k, 7 * k, 3.4 * k, "#ffd166"); R(g, (x - 3.5) * k, (y - 3) * k, 7 * k, 1, "#fff3b0"); if (i < pts.length - 1) { disc(g, (x - 3) * k, (y + 6) * k, 1.8 * k, "#14101c"); disc(g, (x + 3) * k, (y + 6) * k, 1.8 * k, "#14101c"); } }));
    }
    const [hx, hy, hr] = pts[pts.length - 1];
    if (ex !== "windows") { E(hx - 3, hy, hr * 1.35, hr * 1.05, shade(c, 0.08), "skin", seg + 1); }
    F((g, k) => {
      if (ex === "windows") { disc(g, (hx - 6) * k, (hy - 1) * k, 3.2 * k, "#fff3b0"); disc(g, (hx - 6) * k, (hy - 1) * k, 1.8 * k, "#fff"); return; }
      eye(g, k, (hx - 4) * k, (hy - 2) * k, 2.2, glow); eye(g, k, (hx + 2) * k, (hy - 2.6) * k, 1.9, glow);
      if (spec.mouth !== "none") { mouth(g, k, (hx - 11) * k, (hy + 1.5) * k, 10 * k, 4.6 * k, "maw"); R(g, (hx - 14) * k, (hy + 3.4) * k, 4 * k, k, "#c1121f"); }
    });
  },

  train(C) {
    const { spec, f, E, K, B, F, glow } = C, c = spec.color, c2 = spec.color2 || "#ffd166", b = f ? 1 : 0;
    E(32, 61.5, 31, 2, "#000", "void", -2, { a: 0.28 });
    // three cars trailing off to the right, a grinning locomotive on the left
    B(52, 38, 17, 22, shade(c, -0.12), "metal", 1, { rad: 2, bev: 2.4 });
    B(36, 37, 17, 23, c, "metal", 2, { rad: 2, bev: 2.4 });
    B(22, 35, 19, 27, shade(c, 0.05), "metal", 3, { rad: 2.4, bev: 2.8 });
    B(14, 31, 17, 22, shade(c, -0.02), "metal", 4, { rad: 2.4, bev: 2.8 });
    K(44.2, 44, 44.6, 44, 1, 1, "#222", "metal", 0); K(28, 46, 28.4, 46, 1, 1, "#222", "metal", 0);
    B(9, 53, 10, 4.5, "#4a1d1d", "metal", 5, { bev: 1 });
    E(14, 20, 6.6, 3.4, shade(c, -0.25), "metal", 5);
    F((g, k) => {
      for (const [x, y] of [[52, 33], [36, 32], [22, 29]]) for (let i = -1; i <= 1; i += 2) { R(g, (x + i * 4.8 - 2.2) * k, y * k, 4.4 * k, 6.4 * k, "#ffd166"); R(g, (x + i * 4.8 - 2.2) * k, y * k, 4.4 * k, Math.max(1, k * 0.8), "#fff3b0"); if ((x + i + f) % 3 === 0) R(g, (x + i * 4.8 - 1) * k, (y + 2) * k, 2 * k, 3 * k, "#5a1a1a"); }
      for (const x of [50, 42, 34, 26, 18, 10]) { disc(g, x * k, 58.5 * k, 2.5 * k, "#17151c"); disc(g, x * k, 58.5 * k, 1.1 * k, "#6c757d"); }
      eye(g, k, 11 * k, 27 * k, 3.2, glow); eye(g, k, 19 * k, 26 * k, 2.8, glow);
      R(g, 7 * k, 38 * k, 14 * k, 6 * k, "#150a0e"); for (let i = 0; i < 6; i++) { fang(g, (8 + i * 2.2) * k, 38 * k, 2 * k, 3 * k, true); fang(g, (9 + i * 2.2) * k, 44 * k, 2 * k, 3 * k, false); }
      for (let i = 0; i < 3; i++) disc(g, (14 - i * 2 - f) * k, (12 - i * 3.4 - f * 2) * k, (2 + i) * k, `rgba(120,115,125,${0.55 - i * 0.14})`);
    });
  },

  giant(C) {
    const { spec, f, E, K, B, F, glow, r } = C, c = spec.color, c2 = spec.color2 || shade(c, -0.4), ex = spec.extra, b = f ? 1.2 : 0;
    E(32, 61, 26, 2.4, "#000", "void", -2, { a: 0.28 });
    if (ex === "windows") {   // box-shaped: vending machine
      B(32, 34, 28, 52, c, "metal", 2, { rad: 2.5, bev: 3.2 });
      B(31, 30, 20, 26, "#16202e", "metal", 3, { rad: 1.2, bev: 1.4 });
      for (let i = 0; i < 3; i++) for (let j = 0; j < 3; j++) C.B(24.6 + i * 6.8, 21.5 + j * 8, 4.6, 6, [c2, "#9ad0ec", "#e9c46a"][(i + j) % 3], "skin", 4, { bev: 1 });
      B(32, 52, 18, 6.4, "#07070d", "void", 3, { bev: 1 }); B(46, 28, 4, 22, shade(c, -0.2), "metal", 3, { bev: 1 });
      E(46, 22, 1.4, 1.4, "#ffd166", "metal", 5); E(46, 28, 1.4, 1.4, "#06d6a0", "metal", 5);
      F((g, k) => { eye(g, k, 26 * k, (30 - b) * k, 3.2, glow); eye(g, k, 37 * k, (30 - b) * k, 3.2, glow); mouth(g, k, 24 * k, 49 * k, 16 * k, 5 * k, "maw"); });
      return;
    }
    K(24, 46, 22, 59, 6, 4.4, shade(c, -0.3), "skin", 0); K(40, 46, 42, 59, 6, 4.4, shade(c, -0.35), "skin", 0);
    E(32, 36 - b, 21, 20, c, "skin", 3);
    E(32, 42 - b, 15, 10, shade(c, 0.07), "skin", 4);
    E(11, 29, 8.4, 8, shade(c, -0.05), "skin", 4); E(53, 29, 8.4, 8, shade(c, -0.1), "skin", 4);
    const arm = f ? 2 : 0;
    K(9, 32, 5 - arm * 0.3, 52 + arm, 6.6, 5.4, shade(c, -0.12), "skin", 5); K(55, 32, 59 + arm * 0.3, 52 + arm, 6.6, 5.4, shade(c, -0.18), "skin", 5);
    E(5 - arm * 0.3, 55 + arm, 7.2, 6, shade(c, -0.2), "skin", 6); E(59 + arm * 0.3, 55 + arm, 7.2, 6, shade(c, -0.25), "skin", 6);
    if (ex === "arms") { K(14, 38, 3, 44, 3.6, 2.4, shade(c, -0.2), "skin", 6); K(50, 38, 61, 44, 3.6, 2.4, shade(c, -0.2), "skin", 6); }
    E(32, 14 - b, 8.4, 7.8, shade(c, 0.05), "skin", 6);
    E(32, 10 - b, 9, 3, shade(c, -0.3), "skin", 7);
    if (ex === "coffin") { B(50, 22, 11, 34, "#4a4f57", "metal", 8, { rad: 2, bev: 2 }); B(50, 22, 5, 26, "#6c757d", "metal", 9, { bev: 1 }); F((g, k) => R(g, 49 * k, 16 * k, 2 * k, 10 * k, "#adb5bd")); }
    const n = spec.eyes ?? 2;
    F((g, k) => {
      eye(g, k, 28.4 * k, (14.4 - b) * k, 2, glow); eye(g, k, 36 * k, (14.4 - b) * k, 2, glow);
      for (let i = 0; i < Math.max(0, n - 2); i++) eye(g, k, (22 + r(i, 4) * 20) * k, (28 + r(i, 5) * 8 - b) * k, 1.6 + r(i, 6), glow, false);
      mouth(g, k, 22 * k, (42 - b) * k, 20 * k, 9 * k, "maw");
    });
  },

  swarm(C) {
    const { spec, f, E, F, glow, r } = C, c = spec.color, c2 = spec.color2 || shade(c, -0.4);
    E(32, 56, 22, 3, "#000", "void", -2, { a: 0.18 });
    E(32, 34, 13, 11, c2, "ghost", 0, { a: 0.35 });
    const n = 26;
    const birds = [];
    for (let i = 0; i < n; i++) { const a = (i / n) * 6.283 * 2.1 + r(i, 1), rad = 5 + r(i, 2) * 22; birds.push([32 + Math.cos(a) * rad * 1.05, 34 + Math.sin(a) * rad * 0.62, r(i, 3)]); }
    birds.sort((p, q) => p[1] - q[1]);
    F((g, k) => {
      birds.forEach(([x, y, v], i) => {
        const col = i % 5 === 0 ? c2 : v > 0.5 ? c : shade(c, -0.14), dk = shade(col, -0.4), fl = (f + i) & 1;
        const px = Math.round(x * k), py = Math.round(y * k), s = Math.max(2, Math.round(k * 1.7));
        R(g, px - s * 2, py + (fl ? -s : s * 0.5), s * 2, s, col); R(g, px, py + (fl ? -s : s * 0.5), s * 2, s, shade(col, -0.12));
        R(g, px - s, py, s * 2, s * 1.3, col); R(g, px - s, py + s, s * 2, Math.max(1, s * 0.5), dk);
        R(g, px - s * 2.2, py - Math.max(1, s * 0.4), Math.max(1, s * 0.8), Math.max(1, s * 0.8), c2);
      });
      if ((spec.eyes ?? 0) > 0) eye(g, k, 32 * k, 33 * k, 2.4, glow);
    });
  },
};

// Extras that apply to any body (drawn after the builder). Currently horns.
function decorate(C) {
  const { spec, E, K, k } = C;
  if (spec.extra === "horns") {
    const top = { humanoid: [28.5, 8], head: [24, 16], blob: [24, 28], giant: [28, 8], beast: [14, 23] }[spec.shape] || [28, 12];
    const h = spec.shape === "giant" || spec.shape === "blob" ? 9 : 6;
    K(top[0], top[1] + 2, top[0] - 3, top[1] - h, 2.4, 0.5, "#efe3c8", "bone", 9); K(64 - top[0], top[1] + 2, 67 - top[0], top[1] - h, 2.4, 0.5, "#efe3c8", "bone", 9);
  }
}
