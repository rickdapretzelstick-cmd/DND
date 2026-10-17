// Building sprites: 3/4 projection boxes drawn once per building and y-sorted with entities.
import { drawBuilding as drawFlatBuilding, hash, shade } from "./pixel.js";

const ROOFS = ["#7a2c2c", "#33496b", "#2b6a62", "#5a4a40", "#6e1f26", "#3e444b"];
const WALLS = ["#efe6d2", "#dccfbd", "#f2e8cf", "#d3dcb4", "#e8e4dc", "#d9b48a"];
const BODIES = ["#8d99ae", "#a9b0b8", "#6c757d", "#a5a58d", "#b7b7a4", "#7f8c8d"];
const NEON = ["#ff3d7f", "#35e0ff", "#ffd23d", "#8cff5a", "#c46bff"];
const FONT = '"Noto Sans CJK JP","Noto Sans JP","WenQuanYi Zen Hei","Unifont JP",sans-serif';

const MX = 8; // horizontal margin of the sprite (eaves)

function R(c, x, y, w, h, col) { c.fillStyle = col; c.fillRect(Math.round(x), Math.round(y), Math.round(w), Math.round(h)); }

function kawara(c, x, y, w, h, base, seed) {
  R(c, x, y, w, h, base);
  for (let r = 0; r * 4 < h; r++) {
    const yy = y + r * 4, t = r / Math.max(1, h / 4);
    R(c, x, yy, w, 1, shade(base, 0.22 - t * 0.1));        // lit top edge of each tile row
    R(c, x, yy + 3, w, 1, shade(base, -0.35));             // shadow under row
    for (let xx = (r % 2) * 3; xx < w; xx += 6) R(c, x + xx, yy + 1, 1, 2, shade(base, -0.22));
  }
  for (let i = 0; i < w * h / 40; i++) {
    const nx = Math.floor(hash(i, seed, 5) * w), ny = Math.floor(hash(seed, i, 6) * h);
    R(c, x + nx, y + ny, 1, 1, shade(base, hash(i, seed, 7) > 0.5 ? 0.12 : -0.12));
  }
}

function jp(c, b, W, H, ext) {
  const kind = b.kind, big = kind === "temple";
  const temple = big || kind === "shrine_small" || kind === "dorm" || kind === "infirmary";
  const roof = temple ? (big ? "#2f3350" : "#3d405b") : ROOFS[b.color % ROOFS.length];
  const wall = WALLS[(b.color + 2) % WALLS.length];
  const ox = MX, oy = ext;
  const wallH = Math.min(big ? 30 : 26, H - 8);
  const wy = oy + H - wallH;
  // wall + foundation
  R(c, ox + 1, wy, W - 2, wallH, wall);
  R(c, ox + 1, wy, 3, wallH, shade(wall, 0.1));
  R(c, ox + W - 4, wy, 3, wallH, shade(wall, -0.12));
  R(c, ox, oy + H - 3, W, 3, "#7d7a76"); R(c, ox, oy + H - 3, W, 1, "#a6a39c");
  // posts and shoji
  const step = 16;
  for (let x = 0; x < W; x += step) {
    const sx = ox + x + 2;
    R(c, sx, wy, 3, wallH - 3, "#5b3a22"); R(c, sx, wy, 1, wallH - 3, "#7a5233");
    const pw = Math.min(step - 3, W - x - 6);
    if (pw > 5 && x + step < W) {
      R(c, sx + 3, wy + 4, pw, wallH - 12, "#fff0c4");
      R(c, sx + 3, wy + 4, pw, 1, "#a98350"); R(c, sx + 3, wy + wallH - 9, pw, 1, "#a98350");
      for (let g = 4; g < pw; g += 4) R(c, sx + 3 + g, wy + 4, 1, wallH - 12, "#c9a56a");
      R(c, sx + 3, wy + 4 + (wallH - 12) / 2, pw, 1, "#c9a56a");
      R(c, sx + 3, wy + 4, pw, 2, "rgba(255,170,60,0.35)");
    }
  }
  R(c, ox + W - 5, wy, 3, wallH - 3, "#5b3a22");
  // door / noren
  const sign = { shop: ["#2a8f82", "店"], inn: ["#d9553a", "宿"], elder: ["#6c584c", "町"], shrine_small: ["#a30a14", "神"], infirmary: ["#f4f4f0", "+"], temple: ["#a30a14", "呪"], dorm: ["#3d5a80", "寮"] }[kind];
  const dx = ox + Math.round(W / 2) - 8;
  if (sign && !big && kind !== "shrine_small" && kind !== "dorm") {
    R(c, dx, wy + 3, 16, 3, "#4a2e1a");
    R(c, dx + 1, wy + 6, 14, 10, sign[0]);
    R(c, dx + 7, wy + 6, 1, 7, shade(sign[0], -0.3));
    c.fillStyle = sign[0] === "#f4f4f0" ? "#d00000" : "#fff";
    c.font = `bold 9px ${FONT}`; c.textAlign = "center"; c.fillText(sign[1], dx + 8, wy + 14);
    c.textAlign = "left";
  } else {
    R(c, dx + 1, wy + 3, 14, wallH - 3, big || kind === "shrine_small" ? "#8e0c16" : "#5c3d1e");
    R(c, dx + 7, wy + 3, 1, wallH - 3, "#2a1608"); R(c, dx + 12, wy + 11, 2, 2, "#e9c46a");
  }
  // roof
  const top = oy - 6, rh = (wy - top) + 4;
  const ex = temple ? 7 : 5;
  const rx = ox - ex, rw = W + ex * 2;
  kawara(c, rx, top, rw, rh, roof, b.id);
  R(c, rx, top, rw, 3, shade(roof, 0.35));                         // ridge cap
  R(c, rx, top + 3, rw, 1, shade(roof, -0.45));
  R(c, rx, top + rh - 2, rw, 2, shade(roof, -0.5));                // eave edge
  R(c, rx, top + rh, rw, 3, "rgba(20,12,8,0.45)");                 // eave shadow on wall
  if (temple) {                                                    // upturned corners
    R(c, rx - 2, top + rh - 5, 3, 3, roof); R(c, rx + rw - 1, top + rh - 5, 3, 3, roof);
    R(c, rx - 2, top + rh - 5, 3, 1, shade(roof, 0.3));
  }
  if (big) {                                                       // second tier + pillars
    const w2 = Math.round(W * 0.55), x2 = ox + Math.round((W - w2) / 2);
    R(c, x2, top - 14, w2, 12, "#8e0c16");
    R(c, x2 + 2, top - 12, 3, 12, "#b01a22"); R(c, x2 + w2 - 5, top - 12, 3, 12, "#b01a22");
    kawara(c, x2 - 5, top - 22, w2 + 10, 10, roof, b.id + 3);
    R(c, x2 - 5, top - 22, w2 + 10, 2, shade(roof, 0.35));
    for (let x = 2; x < W - 4; x += 22) { R(c, ox + x, wy + 2, 4, wallH - 3, "#a30a14"); R(c, ox + x, wy + 2, 1, wallH - 3, "#d03a3a"); }
  }
  // chimney/lantern accent
  if (kind === "inn") { R(c, ox + 4, wy + wallH - 8, 4, 6, "#ffcf5a"); R(c, ox + 4, wy + wallH - 9, 4, 1, "#5b3a22"); }
}

function city(c, b, W, H, ext) {
  const kind = b.kind;
  const Z = { tower: 96, office: 58, apartment: 44, store: 12, station: 26 }[kind] || 30;
  const body = BODIES[b.color % BODIES.length];
  const ox = MX, oy = ext - Z;
  const fh = H + Z, fy = oy + 6;           // front facade (below roof cap)
  R(c, ox, oy + 6, W, fh - 6, body);
  R(c, ox, oy + 6, 2, fh - 6, shade(body, 0.18));
  R(c, ox + W - 3, oy + 6, 3, fh - 6, shade(body, -0.25));
  // roof cap (top face) with units
  R(c, ox - 1, oy, W + 2, 7, shade(body, 0.3));
  R(c, ox - 1, oy + 6, W + 2, 2, shade(body, -0.4));
  R(c, ox + 3, oy + 2, 8, 4, "#4a5058"); R(c, ox + 4, oy + 3, 6, 1, "#7a8088");
  if (Z > 20) { R(c, ox + W - 8, oy - 12, 1, 12, "#444"); R(c, ox + W - 10, oy - 9, 5, 1, "#444"); R(c, ox + W - 8, oy - 13, 1, 1, "#ff3030"); }
  // window grid
  for (let y = fy + 6; y < oy + fh - 20; y += 9) for (let x = 4; x < W - 6; x += 8) {
    const lit = hash(b.id, x * 7 + y) > 0.55;
    R(c, ox + x, y, 5, 6, "#28324a");
    R(c, ox + x + 1, y + 1, 3, 4, lit ? "#ffe39a" : hash(b.id, y, x) > 0.5 ? "#86b8e0" : "#5a86b0");
    if (!lit) R(c, ox + x + 1, y + 1, 1, 2, "#c8e4ff");
    R(c, ox + x - 1, y + 6, 7, 1, shade(body, -0.2));
  }
  // ground floor
  const gy = oy + fh - 17;
  R(c, ox, gy, W, 17, shade(body, -0.28));
  R(c, ox, gy, W, 1, shade(body, 0.2));
  if (kind === "station") {
    R(c, ox + 6, oy + 12, W - 12, 14, "#16182a"); R(c, ox + 6, oy + 12, W - 12, 1, "#ff3d7f");
    c.fillStyle = "#fff"; c.font = `bold 11px ${FONT}`; c.textAlign = "center"; c.fillText("渋谷", ox + W / 2, oy + 24); c.textAlign = "left";
    R(c, ox + W / 2 - 14, gy + 3, 28, 14, "#101216"); R(c, ox + W / 2 - 14, gy + 2, 28, 2, "#35e0ff");
  } else {
    R(c, ox + W / 2 - 6, gy + 4, 12, 13, "#12161c"); R(c, ox + W / 2 - 6, gy + 4, 12, 1, "#7a8088");
    if (kind === "store") {
      const a = [["#e63946", "#f8f9fa"], ["#2a9d8f", "#f8f9fa"], ["#f4a261", "#f8f9fa"]][b.color % 3];
      for (let x = 0; x < W; x += 6) R(c, ox + x, gy - 3, 6, 6, (x / 6) % 2 ? a[1] : a[0]);
    }
  }
  // neon vertical sign
  if (kind !== "station" && kind !== "store" && W >= 28) {
    const nc = NEON[(b.id + b.color) % NEON.length], chars = ["酒", "夜", "麺", "薬", "館", "星"];
    const sx = ox + W - 1, sh = Math.min(34, fh - 22);
    R(c, sx, fy + 8, 8, sh, "#0c0c14"); R(c, sx, fy + 8, 8, 1, nc); R(c, sx + 7, fy + 8, 1, sh, nc);
    c.fillStyle = nc; c.font = `bold 7px ${FONT}`; c.textAlign = "center";
    for (let i = 0; i < Math.floor(sh / 8); i++) c.fillText(chars[(b.id * 3 + i) % chars.length], sx + 4, fy + 15 + i * 8);
    c.textAlign = "left";
  }
}

function ruin(c, b, W, H, ext) {
  c.save(); c.translate(0, ext - 12);
  drawFlatBuilding(c, b, MX, 12 + 0);
  c.restore();
  // ivy streaks
  for (let i = 0; i < 8; i++) {
    const x = MX + Math.floor(hash(b.id, i, 11) * W), y = ext - 12 + 12 + 4 + Math.floor(hash(i, b.id, 12) * 6);
    const l = 4 + Math.floor(hash(i, b.id, 13) * 12);
    R(c, x, y, 2, l, "#3d6b34"); R(c, x, y, 1, l, "#5a9a46");
  }
}

const U = 2; // buildings are drawn at half size, then doubled, so doors and windows suit a 1:1 world

function make(b) {
  const bs = { ...b, w: b.w / U, h: b.h / U };
  const W = bs.w * 16, H = bs.h * 16;
  const kind = b.kind;
  const isCity = ["office", "apartment", "store", "tower", "station"].includes(kind);
  const ext = isCity ? ({ tower: 110, office: 72, apartment: 58, store: 26, station: 40 }[kind] || 44) : kind === "temple" ? 56 : kind.startsWith("site_") || kind === "gate" ? 30 : 34;
  const small = document.createElement("canvas");
  small.width = W + MX * 2 + 6; small.height = H + ext + 4;
  const c = small.getContext("2d");
  c.imageSmoothingEnabled = false;
  if (isCity) city(c, bs, W, H, ext);
  else if (kind === "gate" || kind.startsWith("site_")) ruin(c, bs, W, H, ext);
  else jp(c, bs, W, H, ext);
  const cv = document.createElement("canvas");
  cv.width = small.width * U; cv.height = small.height * U;
  const g = cv.getContext("2d");
  g.imageSmoothingEnabled = false;
  g.drawImage(small, 0, 0, cv.width, cv.height);
  return { cv, ox: MX * U, oy: ext * U, W: b.w * 16, H: b.h * 16, ext: ext * U };
}

const cache = new Map();
export function buildingSprite(b) {
  let s = cache.get(b.id);
  if (!s) { s = make(b); cache.set(b.id, s); }
  return s;
}

// Baked onto the ground chunk: soft cast shadow to the lower-right.
export function drawBuildingShadow(ctx, b, x0, y0) {
  const W = b.w * 16, H = b.h * 16;
  const tall = ["tower", "office", "apartment"].includes(b.kind) ? 70 : b.kind === "temple" ? 28 : 16;
  ctx.fillStyle = "rgba(12,16,34,0.30)";
  ctx.beginPath();
  ctx.moveTo(x0 + 2, y0 + H);
  ctx.lineTo(x0 + W, y0 + H);
  ctx.lineTo(x0 + W + tall, y0 + H - tall * 0.2);
  ctx.lineTo(x0 + W + tall, y0 + 6);
  ctx.lineTo(x0 + W, y0 + 2);
  ctx.closePath(); ctx.fill();
  ctx.fillStyle = "rgba(12,16,34,0.22)";
  ctx.fillRect(x0 - 2, y0 + H, W + 6, 3);
}
