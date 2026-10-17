// Procedural pixel art: characters, curses, tiles and buildings. No image files.

export const TILE = 16;

export const SKIN = ["#f6d7c3", "#eec1a0", "#d9a07c", "#b97a56", "#8d5a3b", "#5e3a24"];
export const HAIR_STYLES = ["Short", "Spiky", "Long", "Bob", "Slicked", "Fringe", "Wild White", "Ponytail", "Buzz", "Bun", "Twin Tails", "Undercut"];
export const OUTFITS = ["Jujutsu High Uniform", "Hoodie", "Long Coat", "T-shirt", "School Blazer", "Suit", "Lab Coat", "Kimono"];
export const ACCESSORIES = ["None", "Blindfold", "Glasses", "Sunglasses", "Goggles", "Collar Mask", "Stitches", "Headband"];
export const BUILDS = ["Slim", "Average", "Broad"];

// ---------- helpers ----------

export function makeCanvas(w, h) {
  const c = document.createElement("canvas");
  c.width = w;
  c.height = h;
  const ctx = c.getContext("2d");
  ctx.imageSmoothingEnabled = false;
  return [c, ctx];
}

export function hash(x, y, s = 0) {
  let h = (x * 374761393 + y * 668265263 + s * 2147483647) | 0;
  h = Math.imul(h ^ (h >>> 13), 1274126177);
  h ^= h >>> 16;
  return (h >>> 0) / 4294967296;
}

export function rng(seed) {
  let s = seed >>> 0 || 1;
  return () => {
    s ^= s << 13; s >>>= 0;
    s ^= s >>> 17;
    s ^= s << 5; s >>>= 0;
    return s / 4294967296;
  };
}

function strSeed(str) {
  let h = 2166136261;
  for (const ch of String(str)) h = Math.imul(h ^ ch.charCodeAt(0), 16777619);
  return h >>> 0;
}

export function shade(hex, amt) {
  const n = parseInt(hex.slice(1), 16);
  let r = (n >> 16) & 255, g = (n >> 8) & 255, b = n & 255;
  if (amt < 0) { r *= 1 + amt; g *= 1 + amt; b *= 1 + amt; }
  else { r += (255 - r) * amt; g += (255 - g) * amt; b += (255 - b) * amt; }
  return "#" + [r, g, b].map((v) => Math.max(0, Math.min(255, Math.round(v))).toString(16).padStart(2, "0")).join("");
}

const px = (ctx, x, y, w, h, c) => { ctx.fillStyle = c; ctx.fillRect(x, y, w, h); };

// ---------- characters (24 x 36, chibi anime proportions) ----------

export const CHAR_W = 24, CHAR_H = 36;
const charCache = new Map();

const HSTYLES = [
  { bang: "part", top: "none", side: 4 },
  { bang: "spiky", top: "spikes", side: 4 },
  { bang: "part", top: "none", side: 15, long: 1 },
  { bang: "straight", top: "none", side: 9 },
  { bang: "slick", top: "none", side: 3 },
  { bang: "swoop", top: "none", side: 7 },
  { bang: "spiky", top: "wild", side: 5 },
  { bang: "part", top: "none", side: 4, pony: 1 },
  { bang: "buzz", top: "none", side: 1 },
  { bang: "part", top: "bun", side: 4 },
  { bang: "straight", top: "none", side: 5, tails: 1 },
  { bang: "swoop", top: "tall", side: 2 },
];

// Outline pass: dark, hue-aware outline around the silhouette.
function outline(cv) {
  const w = cv.width, h = cv.height, ctx = cv.getContext("2d");
  const img = ctx.getImageData(0, 0, w, h), d = img.data, out = new Uint8ClampedArray(d);
  for (let y = 0; y < h; y++) for (let x = 0; x < w; x++) {
    const i = (y * w + x) * 4;
    if (d[i + 3] > 40) continue;
    let r = 0, g = 0, b = 0, n = 0;
    for (const [dx, dy] of [[-1, 0], [1, 0], [0, -1], [0, 1]]) {
      const xx = x + dx, yy = y + dy;
      if (xx < 0 || yy < 0 || xx >= w || yy >= h) continue;
      const j = (yy * w + xx) * 4;
      if (d[j + 3] > 200) { r += d[j]; g += d[j + 1]; b += d[j + 2]; n++; }
    }
    if (n) { out[i] = r / n * 0.28; out[i + 1] = g / n * 0.24; out[i + 2] = b / n * 0.32 + 6; out[i + 3] = 255; }
  }
  img.data.set(out);
  ctx.putImageData(img, 0, 0);
}

// dir: 0 down, 1 left, 2 right, 3 up. frame 0..3 (walk).
export function charSprite(look, dir = 0, frame = 0) {
  const key = JSON.stringify(look) + "|" + dir + "|" + frame;
  let c = charCache.get(key);
  if (c) return c;
  if (dir === 2) {
    const src = charSprite(look, 1, frame);
    const [m, mx] = makeCanvas(CHAR_W, CHAR_H);
    mx.translate(CHAR_W, 0);
    mx.scale(-1, 1);
    mx.drawImage(src, 0, 0);
    charCache.set(key, m);
    return m;
  }
  const [cv, ctx] = makeCanvas(CHAR_W, CHAR_H);
  // dirs 4..7 are diagonal 3/4 views: 4 SW, 5 SE (front-ish), 6 NW, 7 NE (back-ish)
  if (dir >= 4) drawChar(ctx, look || {}, dir < 6 ? 0 : 3, frame, dir & 1 ? 1 : -1);
  else drawChar(ctx, look || {}, dir, frame);
  outline(cv);
  ctx.fillStyle = "rgba(8,6,20,0.3)";
  ctx.beginPath(); ctx.ellipse(12, 34, 7.5, 1.7, 0, 0, Math.PI * 2); ctx.fill();
  charCache.set(key, cv);
  if (charCache.size > 1500) charCache.delete(charCache.keys().next().value);
  return cv;
}

function drawChar(ctx, L, dir, frame, tq = 0) {
  const R = (x, y, w, h, c) => { ctx.fillStyle = c; ctx.fillRect(Math.round(x), Math.round(y), w, h); };
  const skin = SKIN[L.skin ?? 1] || SKIN[1], skinD = shade(skin, -0.16), skinL = shade(skin, 0.12);
  const hair = L.hairColor || "#1b1b1b", hairD = shade(hair, -0.35), hairL = shade(hair, 0.3);
  const c1 = L.color1 || "#14141c", c2 = L.color2 || "#2a2a3a";
  const c1D = shade(c1, -0.3), c1L = shade(c1, 0.14), c2D = shade(c2, -0.3);
  const eyeC = L.eyes || "#3b2a1a", eyeD = shade(eyeC, -0.4), eyeL = shade(eyeC, 0.45);
  const build = L.build ?? 1, outfit = L.outfit ?? 0, acc = L.accessory ?? 0;
  const st = HSTYLES[L.hair ?? 0] || HSTYLES[0];
  const side = dir === 1, up = dir === 3, front = dir === 0;
  const step = frame % 4;
  const bob = step === 1 || step === 3 ? 1 : 0;
  const bw = (build === 0 ? 10 : build === 2 ? 14 : 12) - (side ? 3 : 0);
  const bx = 12 - bw / 2;
  const hy = 4 + bob;                                // head top
  const ty = 16 + bob;                               // torso top
  const longOutfit = outfit === 2 || outfit === 6 || outfit === 7;
  const torsoH = longOutfit ? 14 : 11;
  const shirt = outfit === 3 ? c1 : outfit === 6 ? "#eceff2" : c1;

  // ---- back hair (behind everything) ----
  const backHair = () => {
    if (st.long) { R(3, hy + 4, 18, st.side + 1 - 3, hair); R(3, hy + 4 + st.side - 5, 18, 2, hairD); }
    if (st.tails && !up) { R(0, hy + 3, 4, 14, hair); R(20, hy + 3, 4, 14, hair); R(0, hy + 14, 4, 3, hairD); R(20, hy + 14, 4, 3, hairD); R(1, hy + 3, 1, 10, hairL); R(21, hy + 3, 1, 10, hairL); }
    if (st.pony && !side && !up) R(19, hy + 4, 4, 11, hairD);
  };
  if (!side) backHair();

  // ---- legs ----
  const pants = outfit === 3 ? "#3a5a8c" : outfit === 6 ? "#2b2d42" : outfit === 5 ? shade(c1, -0.1) : shade(c1, -0.12);
  const shoe = "#16141c", shoeL = "#33303c";
  const lo = step === 1 ? -1 : step === 3 ? 1 : 0;   // leg lift offset
  const lift = 2;
  if (side) {
    const s1 = step === 1 ? -2 : step === 3 ? 2 : 0;
    R(9 + s1, 27 + bob, 5, 6, shade(pants, -0.18)); R(9 + s1, 32 + bob, 6, 2, shoe);
    R(11 - s1, 27 + bob, 5, 6, pants); R(11 - s1, 27 + bob, 1, 6, shade(pants, 0.12)); R(10 - s1, 32 + bob, 6, 2, shoe); R(10 - s1, 32 + bob, 6, 1, shoeL);
  } else if (outfit !== 7) {
    const lx = 12 - Math.floor(bw / 2) + 1;
    R(lx, 27 + bob, 4, 6 - (lo === -1 ? lift : 0), pants); R(lx, 27 + bob, 1, 6, shade(pants, 0.1));
    R(lx - 1, 32 + bob - (lo === -1 ? lift : 0), 5, 2, shoe); R(lx - 1, 32 + bob - (lo === -1 ? lift : 0), 5, 1, shoeL);
    const rx = 12 + Math.floor(bw / 2) - 5;
    R(rx, 27 + bob, 4, 6 - (lo === 1 ? lift : 0), shade(pants, -0.14)); R(rx + 3, 27 + bob, 1, 6, shade(pants, -0.3));
    R(rx, 32 + bob - (lo === 1 ? lift : 0), 5, 2, shoe); R(rx, 32 + bob - (lo === 1 ? lift : 0), 5, 1, shoeL);
  } else {
    R(9 - lo, 32 + bob, 3, 2, "#6b4a2b"); R(14 + lo, 32 + bob, 3, 2, "#6b4a2b");
  }

  // ---- torso ----
  if (tq) ctx.translate(tq, 0);
  const tc = outfit === 6 ? "#eceff2" : c1;
  R(bx + 2, ty, bw - 4, 1, tc); R(bx + 1, ty + 1, bw - 2, 1, tc); R(bx, ty + 2, bw, torsoH - 2, tc);   // rounded shoulders
  R(bx, ty + 1, 2, torsoH - 1, shade(tc, 0.12)); R(bx + bw - 2, ty + 1, 2, torsoH - 1, shade(tc, -0.2));
  R(bx, ty + torsoH - 1, bw, 1, shade(tc, -0.3));
  if (front) {
    const cx = 12;
    switch (outfit) {
      case 0: // high-collar uniform
        R(cx - 3, ty, 6, 3, c1D); R(cx - 1, ty + 1, 2, 9, c1D);
        for (let i = 0; i < 3; i++) { R(cx - 3, ty + 3 + i * 3, 1, 1, "#f2c94c"); R(cx + 2, ty + 3 + i * 3, 1, 1, "#f2c94c"); }
        R(bx, ty + 9, bw, 1, c2); break;
      case 1: // hoodie
        R(cx - 4, ty, 8, 3, c2); R(cx - 3, ty + 3, 1, 4, "#f4f4f4"); R(cx + 2, ty + 3, 1, 4, "#f4f4f4");
        R(bx + 2, ty + 7, bw - 4, 3, shade(c1, -0.18)); R(bx + 2, ty + 7, bw - 4, 1, shade(c1, -0.35)); break;
      case 2: // long coat
        R(cx - 1, ty, 2, torsoH, c2); R(cx - 4, ty, 3, 4, shade(c1, 0.2)); R(cx + 1, ty, 3, 4, shade(c1, -0.1));
        R(bx, ty + torsoH - 3, bw, 2, c2D); for (let i = 0; i < 3; i++) R(cx + 1, ty + 4 + i * 3, 1, 1, "#d8c27a"); break;
      case 3: // t-shirt
        R(cx - 3, ty, 6, 2, shade(skin, -0.05)); R(bx + 3, ty + 3, bw - 6, 4, c2); R(bx + 4, ty + 4, bw - 8, 1, shade(c2, 0.3)); break;
      case 4: // blazer
        R(cx - 3, ty, 6, 8, "#f4f4f4"); R(cx - 4, ty, 2, 9, shade(c1, -0.25)); R(cx + 2, ty, 2, 9, shade(c1, -0.25));
        R(cx - 1, ty + 1, 2, 6, c2); R(cx - 1, ty + 1, 2, 1, shade(c2, 0.3)); break;
      case 5: // suit
        R(cx - 3, ty, 6, 7, "#f4f4f4"); R(cx - 4, ty, 2, 10, shade(c1, -0.25)); R(cx + 2, ty, 2, 10, shade(c1, -0.25));
        R(cx - 1, ty + 1, 2, 8, c2); R(cx - 1, ty + 8, 2, 1, c2D); break;
      case 6: // lab coat
        R(cx - 3, ty, 6, 6, c1); R(cx - 1, ty + 5, 2, torsoH - 5, "#c8ccd2"); R(bx + 2, ty + 6, 3, 3, "#d4d8de"); R(bx + 2, ty + 6, 3, 1, c2); break;
      case 7: // kimono with obi
        R(cx - 3, ty, 2, 6, shade(c1, -0.25)); R(cx + 1, ty, 2, 6, c2D); R(cx - 3, ty, 1, 7, c2);
        R(bx - 1, ty + 6, bw + 2, 4, c2); R(bx - 1, ty + 6, bw + 2, 1, shade(c2, 0.3)); R(cx - 1, ty + 7, 2, 4, c2D); break;
    }
  } else if (up) {
    if (outfit === 1) R(bx + 1, ty, bw - 2, 5, c2);
    if (outfit === 7) { R(bx - 1, ty + 6, bw + 2, 4, c2); R(10, ty + 8, 4, 5, c2D); }
    if (outfit === 0) R(bx, ty + 9, bw, 1, c2);
  } else if (side) {
    if (outfit === 7) R(bx - 1, ty + 6, bw + 2, 4, c2);
    if (outfit === 2) R(bx, ty + torsoH - 3, bw, 2, c2D);
    if (outfit === 0) R(bx, ty + 9, bw, 1, c2);
  }
  // ---- arms ----
  const swing = step === 1 ? 1 : step === 3 ? -1 : 0;
  const sleeve = outfit === 3 ? skin : outfit === 6 ? "#e6e9ec" : c1;
  const cuff = outfit === 0 ? c2 : outfit === 7 ? c2D : null;
  if (side) {
    const ax = 10 + (swing ? swing * 2 : 0);
    R(ax, ty + 1, 4, 9, shade(sleeve, -0.12)); R(ax, ty + 1, 1, 9, shade(sleeve, 0.1));
    if (cuff) R(ax, ty + 8, 4, 1, cuff);
    R(ax, ty + 10, 4, 2, skin); R(ax, ty + 11, 4, 1, skinD);
  } else {
    const a1 = swing, a2 = -swing;
    R(bx - 3, ty + 1 + a1, 3, 9, sleeve); R(bx - 3, ty + 1 + a1, 1, 9, shade(sleeve, 0.12)); if (cuff) R(bx - 3, ty + 8 + a1, 3, 1, cuff);
    R(bx - 3, ty + 10 + a1, 3, 2, skin); R(bx - 3, ty + 11 + a1, 3, 1, skinD);
    R(bx + bw, ty + 1 + a2, 3, 9, shade(sleeve, -0.16)); R(bx + bw + 2, ty + 1 + a2, 1, 9, shade(sleeve, -0.32)); if (cuff) R(bx + bw, ty + 8 + a2, 3, 1, cuff);
    R(bx + bw, ty + 10 + a2, 3, 2, skin); R(bx + bw, ty + 11 + a2, 3, 1, skinD);
  }
  if (side) { /* hair mass behind head for side view */ backHair(); }

  // ---- head ----
  if (tq) ctx.translate(tq, 0);
  const rows = [8, 12, 14, 16, 16, 16, 16, 15, 14, 11, 8, 4];
  const hw = side ? 14 : 16;
  const hx0 = side ? 5 : 4;
  R(10, hy + 11, 4, 3, skinD);                       // neck
  rows.forEach((w0, i) => {
    const w = Math.min(w0, hw);
    const x = side ? hx0 + (hw - w) / 2 + (i > 4 ? -0 : 0) : 12 - w / 2;
    R(x, hy + i, w, 1, skin);
    if (side) { R(x + w - 2, hy + i, 2, 1, skinD); } else { R(x + w - 2, hy + i, 2, 1, i > 3 ? skinD : skin); R(x, hy + i, 1, 1, skinL); }
  });
  if (!up) {
    R(side ? 6 : 6, hy + 10, 12, 1, skinD);
    R(side ? 9 : 5, hy + 11, 2, 1, skinD); // placeholder chin tone
  }
  // ---- face ----
  const ey = hy + 6;
  const eye = (x) => {                 // big anime eye, 4x5
    R(x, ey, 4, 5, "#fdfdfd");
    R(x, ey, 4, 1, "#17121f");         // upper lash
    R(x + 1, ey + 1, 3, 4, eyeC); R(x + 1, ey + 1, 3, 1, eyeD); R(x + 1, ey + 4, 3, 1, eyeL);
    R(x + 2, ey + 2, 1, 2, "#0d0a14");
    R(x + 1, ey + 1, 1, 1, "#ffffff");
    R(x - (x < 12 ? 1 : 0), ey - 1, 1, 1, "#17121f");
  };
  if (front) {
    eye(6); eye(14);
    R(5, ey - 2, 4, 1, hairD); R(15, ey - 2, 4, 1, hairD);              // brows
    R(11, hy + 10, 2, 1, shade(skin, -0.38));                            // mouth
    R(5, hy + 9, 2, 1, "rgba(255,130,130,0.35)"); R(17, hy + 9, 2, 1, "rgba(255,130,130,0.35)");
    if (tq) {                                                            // 3/4 turn: narrow the far eye, add a nose
      const fx = tq > 0 ? 6 : 16;
      R(fx, ey - 1, 2, 6, skin); R(tq > 0 ? 5 : 17, ey - 2, 3, 1, skin);
      R(tq > 0 ? 17 : 6, hy + 8, 1, 2, skinD); R(tq > 0 ? 18 : 5, hy + 8, 1, 3, skinD);
    }
  } else if (side) {
    eye(7);
    R(6, ey - 2, 4, 1, hairD);
    R(4, ey + 3, 1, 2, skin);                                            // nose
    R(7, hy + 10, 2, 1, shade(skin, -0.38));
    R(9, hy + 9, 2, 1, "rgba(255,130,130,0.35)");
  }

  // ---- accessories (on face) ----
  if (!up) {
    const ex = side ? [7] : [6, 14];
    switch (acc) {
      case 1: R(side ? 4 : 3, ey - 1, side ? 12 : 18, 4, "#0c0b12"); R(side ? 4 : 3, ey - 1, side ? 12 : 18, 1, "#2a2836"); if (!side) { R(19, ey + 2, 4, 1, "#0c0b12"); } break; // blindfold
      case 2: for (const x of ex) { R(x - 1, ey - 1, 6, 1, "#2c2c34"); R(x - 1, ey + 5, 6, 1, "#2c2c34"); R(x - 1, ey - 1, 1, 7, "#2c2c34"); R(x + 4, ey - 1, 1, 7, "#2c2c34"); R(x, ey + 1, 1, 1, "rgba(200,230,255,0.6)"); } if (!side) R(10, ey + 1, 4, 1, "#2c2c34"); break; // glasses
      case 3: for (const x of ex) { R(x - 1, ey - 1, 6, 5, "#0e0e14"); R(x, ey, 2, 1, "#4a4a5a"); } if (!side) R(10, ey, 4, 1, "#0e0e14"); break; // sunglasses
      case 4: for (const x of ex) { R(x - 1, ey - 1, 6, 6, "#d9a93a"); R(x, ey, 4, 4, "#24577a"); R(x, ey, 2, 1, "#9ad8f2"); } if (!side) R(10, ey + 1, 4, 2, "#8a6a2a"); break; // goggles
      case 5: R(side ? 4 : 4, hy + 8, side ? 11 : 16, 6, c1); R(side ? 4 : 4, hy + 8, side ? 11 : 16, 1, c1L); break; // collar mask
      case 6: R(side ? 6 : 5, hy + 1, side ? 9 : 14, 2, "#7a1d1d"); for (let i = 0; i < 5; i++) R((side ? 6 : 5) + i * (side ? 2 : 3), hy, 1, 4, "#7a1d1d"); break; // stitches
      case 7: R(3, hy + 3, 18, 3, "#cc1f2a"); R(3, hy + 3, 18, 1, "#ee4a52"); R(1, hy + 4, 3, 5, "#cc1f2a"); break; // headband
    }
  } else if (acc === 1) R(3, ey - 1, 18, 3, "#0c0b12");
  else if (acc === 7) { R(3, hy + 3, 18, 3, "#cc1f2a"); R(19, hy + 4, 4, 6, "#cc1f2a"); }

  // ---- hair front ----
  const cap = (x0, w, y) => R(x0, y, w, 1, hair);
  const top = hy - 2;
  const spikes = (n, hmax, jag) => {
    for (let i = 0; i < n; i++) {
      const sx = 4 + Math.round((i / (n - 1)) * 13), h0 = hmax - (i % 2) * jag - (i === 0 || i === n - 1 ? 1 : 0);
      for (let k = 0; k < h0; k++) R(sx + (k > h0 - 2 ? 1 : 0), top - k, k > h0 - 2 ? 1 : 3, 1, k > h0 - 3 ? hairL : hair);
    }
  };
  if (up) {
    cap(8, 8, top); cap(6, 12, top + 1); cap(5, 14, top + 2);
    R(4, top + 3, 16, 12, hair); R(4, top + 3, 16, 1, hairL); R(4, top + 13, 16, 2, hairD);
    for (let i = 0; i < 6; i++) R(6 + i * 3, top + 5 + (i % 2) * 3, 1, 4, hairD);
    if (st.long) { R(4, top + 14, 16, st.side - 6, hair); R(4, top + st.side + 6, 16, 2, hairD); }
    if (st.pony) { R(10, top + 10, 4, 12, hair); R(10, top + 20, 4, 2, hairD); R(10, top + 9, 4, 1, "#cc1f2a"); }
    if (st.tails) { R(1, top + 6, 4, 14, hair); R(19, top + 6, 4, 14, hair); }
  } else {
    const sideLen = st.side;
    const fx0 = side ? 5 : 4, fw = side ? 14 : 16;
    cap(side ? 8 : 8, 8, top); cap(side ? 6 : 6, side ? 11 : 12, top + 1);
    R(fx0, top + 2, fw, 3, hair);
    R(fx0 + 1, top + 1, side ? 9 : 10, 1, hairL);
    R(fx0 + 2, top + 2, 4, 1, hairL);
    if (side) {                         // hair covering the back of the head
      R(12, top + 3, 7, 9 + Math.max(0, sideLen - 9 > 0 ? 3 : sideLen - 4), hair);
      R(17, top + 3, 2, 9, hairD);
      if (sideLen > 9) R(12, top + 3, 7, sideLen, hair);
      if (st.pony) R(17, top + 5, 5, 10, hairD);
      if (st.tails) R(15, top + 5, 5, 12, hair);
    } else {
      R(4, top + 5, 2, sideLen, hair); R(18, top + 5, 2, sideLen, hairD);
      R(4, top + 5, 1, sideLen, hairL);
    }
    // bangs
    const bx0 = fx0, by = top + 5;
    switch (st.bang) {
      case "part": R(bx0, by, 7, 2, hair); R(bx0 + 1, by + 2, 4, 1, hair);
        if (!side) { R(13, by, 7, 2, hair); R(15, by + 2, 4, 1, hair); } break;
      case "straight": R(bx0, by, fw, 2, hair); for (let i = 0; i < fw; i += 4) R(bx0 + i, by + 2, 2, 1, hair); break;
      case "spiky": for (let i = 0; i < (side ? 3 : 5); i++) { const x = bx0 + 1 + i * 3, l = 3 + (i % 2); R(x, by, 3, 1, hair); R(x, by + 1, 2, l - 1, hair); R(x, by + l - 1, 1, 1, hairD); } break;
      case "swoop": R(bx0, by, fw, 2, hair); R(bx0, by + 2, side ? 8 : 9, 2, hair); R(bx0, by + 4, 4, 1, hairD); break;
      case "slick": R(bx0 + 1, by - 1, fw - 2, 1, hairL); break;
      case "buzz": break;
    }
    if (st.bang === "buzz") { R(fx0, top + 2, fw, 2, hairD); }
    if (st.top === "spikes") spikes(5, 4, 2);
    if (st.top === "wild") spikes(7, 7, 3);
    if (st.top === "tall") { R(8, top - 3, 9, 4, hair); R(8, top - 3, 9, 1, hairL); R(7, top - 1, 3, 2, hairD); }
    if (st.top === "bun") { R(9, top - 5, 7, 5, hair); R(9, top - 5, 7, 1, hairL); R(10, top - 4, 2, 1, hairL); R(11, top - 1, 3, 1, "#cc1f2a"); }
    if (st.pony && !side) R(18, top + 6, 3, 7, hairD);
    // hair shine band
    if (!side && st.bang !== "buzz") { R(7, top + 2, 6, 1, hairL); }
  }
}

// ---------- curses ----------

const curseCache = new Map();

// Returns a canvas for a curse sprite: spec from the bestiary, frame 0/1 for idle animation.
export function curseSprite(spec, id = "", frame = 0) {
  const key = id + JSON.stringify(spec) + frame;
  let c = curseCache.get(key);
  if (c) return c;
  const size = spec.size || 1;
  const S = 16 + size * 12;
  const [cv, ctx] = makeCanvas(S, S);
  drawCurse(ctx, spec, S, rng(strSeed(id || spec.color)), frame);
  outline(cv);
  curseCache.set(key, cv);
  return cv;
}

function eyesAt(ctx, list, color = "#ffe066", pupil = "#111") {
  for (const [x, y, s] of list) {
    px(ctx, x, y, s, s, "#fff");
    px(ctx, x + Math.floor(s / 2), y + Math.floor(s / 2), Math.max(1, Math.floor(s / 2)), Math.max(1, Math.floor(s / 2)), pupil);
    if (s > 2) px(ctx, x, y, s, 1, color);
  }
}

function grin(ctx, x, y, w, color = "#1a0000") {
  px(ctx, x, y, w, 2, color);
  for (let i = 0; i < w; i += 2) px(ctx, x + i, y, 1, 1, "#fff");
  px(ctx, x - 1, y - 1, 1, 1, color); px(ctx, x + w, y - 1, 1, 1, color);
}

function blobShape(ctx, cx, cy, rx, ry, color, r, jag = 0.18) {
  for (let y = -ry; y <= ry; y++) {
    for (let x = -rx; x <= rx; x++) {
      const d = (x * x) / (rx * rx) + (y * y) / (ry * ry);
      if (d <= 1 - jag * r()) {
        px(ctx, cx + x, cy + y, 1, 1, d > 0.7 ? shade(color, -0.25) : y < -ry * 0.4 && x < 0 ? shade(color, 0.15) : color);
      }
    }
  }
}

function drawCurse(ctx, sp, S, r, frame) {
  const c = sp.color, c2 = sp.color2 || shade(c, -0.4);
  const cx = S / 2, base = S - 3;
  const b = frame ? 1 : 0;
  ctx.fillStyle = "rgba(0,0,0,0.3)";
  ctx.beginPath(); ctx.ellipse(cx, base + 1, S * 0.32, 2, 0, 0, Math.PI * 2); ctx.fill();
  const nEyes = sp.eyes ?? 2;
  const eyeList = (x0, y0, w, h, sz = 2) => {
    const out = [];
    for (let i = 0; i < nEyes; i++) out.push([Math.round(x0 + r() * (w - sz)), Math.round(y0 + r() * (h - sz)), sz]);
    return out;
  };
  switch (sp.shape) {
    case "head": {
      const R = S * 0.32;
      if (sp.extra === "wings") {
        const wy = frame ? -3 : 1;
        px(ctx, cx - R - 6, cx - 6 + wy, 7, 4, "rgba(220,240,255,0.7)"); px(ctx, cx + R - 1, cx - 6 + wy, 7, 4, "rgba(220,240,255,0.7)");
      }
      blobShape(ctx, cx, cx - b, R, R, c, r, 0.05);
      eyesAt(ctx, [[cx - R + 2, cx - R / 2 - b, 4], [cx + R - 6, cx - R / 2 - b, 4]], c2, "#7f0000");
      if (nEyes > 2) eyesAt(ctx, [[cx - 1, cx - R + 3 - b, 3]]);
      grin(ctx, cx - R / 2, cx + R / 3 - b, R);
      for (let i = 0; i < 3; i++) px(ctx, cx - 3 + i * 3, cx + R - b, 1, 4, shade(c, -0.4));
      break;
    }
    case "blob": {
      const rx = S * 0.36, ry = S * 0.3;
      blobShape(ctx, cx, base - ry - b, rx, ry, c, r);
      eyesAt(ctx, eyeList(cx - rx + 3, base - ry * 2 + 3 - b, rx * 2 - 6, ry, 3), c2);
      if (sp.mouth === "maw") { px(ctx, cx - rx / 2, base - ry / 2 - b, rx, 3, "#1a0000"); for (let i = 0; i < rx; i += 2) px(ctx, cx - rx / 2 + i, base - ry / 2 - b, 1, 2, "#fff"); }
      if (sp.mouth === "stitched") { px(ctx, cx - 4, base - ry / 2, 8, 1, "#3b0000"); for (let i = 0; i < 4; i++) px(ctx, cx - 4 + i * 2, base - ry / 2 - 1, 1, 3, "#3b0000"); }
      if (sp.extra === "tentacles" || sp.extra === "arms") {
        for (let i = 0; i < 4; i++) {
          const x = cx - rx + 3 + i * (rx / 2);
          for (let j = 0; j < 5; j++) px(ctx, x + Math.round(Math.sin((j + i + frame) * 0.9) * 1.5), base - 4 + j - (j > 3 ? 1 : 0), 2, 1, c2);
        }
      }
      break;
    }
    case "humanoid": {
      const h = S * 0.78, top = base - h;
      const w = Math.max(6, Math.round(S * 0.22));
      const body = c2;
      const half = sp.extra === "halfbody";
      const legTop = top + h * 0.62;
      if (!half) {
        px(ctx, cx - w / 2, legTop, w / 2 - 1, base - legTop, shade(body, -0.25));
        px(ctx, cx + 1, legTop, w / 2 - 1, base - legTop, shade(body, -0.3));
      }
      const torsoY = top + h * 0.24 - b, torsoH = half ? h * 0.5 : h * 0.4;
      px(ctx, cx - w / 2 - 1, torsoY, w + 2, torsoH, body);
      if (sp.extra === "robe" || sp.extra === "veil") px(ctx, cx - w / 2 - 2, torsoY + torsoH - 2, w + 4, base - torsoY - torsoH + 2 - (half ? 6 : 0), sp.extra === "veil" ? "#f8f9fa" : body);
      // Long arms
      const armLen = half ? h * 0.3 : h * 0.45;
      px(ctx, cx - w / 2 - 3, torsoY + 1, 2, armLen, shade(c, -0.1)); px(ctx, cx + w / 2 + 1, torsoY + 1, 2, armLen, shade(c, -0.1));
      px(ctx, cx - w / 2 - 4, torsoY + armLen, 3, 2, c); px(ctx, cx + w / 2 + 1, torsoY + armLen, 3, 2, c);
      // Head
      const hs = Math.max(6, Math.round(S * 0.22));
      const hy = top - b;
      px(ctx, cx - hs / 2, hy, hs, hs, c);
      px(ctx, cx - hs / 2, hy + hs - 1, hs, 1, shade(c, -0.25));
      if (nEyes > 0) {
        const ex = cx - hs / 2 + 1;
        if (nEyes >= 4) eyesAt(ctx, [[ex, hy + 2, 2], [ex + hs - 4, hy + 2, 2], [ex, hy + 4, 2], [ex + hs - 4, hy + 4, 2]], c2, "#d00000");
        else if (nEyes === 1) eyesAt(ctx, [[cx - 1, hy + 2, 3]], c2, "#d00000");
        else eyesAt(ctx, [[ex, hy + 3, 2], [ex + hs - 4, hy + 3, 2]], c2, "#111");
      }
      if (sp.mouth === "grin") grin(ctx, cx - hs / 2 + 1, hy + hs - 3, hs - 2);
      if (sp.mouth === "stitched") { px(ctx, cx - hs / 2, hy + 1, hs, 1, "#4a0000"); for (let i = 0; i < hs; i += 2) px(ctx, cx - hs / 2 + i, hy, 1, 3, "#4a0000"); }
      switch (sp.extra) {
        case "veil": px(ctx, cx - hs / 2 - 1, hy - 1, hs + 2, hs + 3, "rgba(248,249,250,0.55)"); break;
        case "scissors": px(ctx, cx + w / 2 + 3, torsoY + armLen - 8, 2, 10, "#adb5bd"); px(ctx, cx + w / 2 + 5, torsoY + armLen - 8, 2, 10, "#ced4da"); break;
        case "hood": px(ctx, cx - hs / 2 - 1, hy - 1, hs + 2, 3, body); px(ctx, cx - hs / 2 - 1, hy, 1, hs, body); px(ctx, cx + hs / 2, hy, 1, hs, body); break;
        case "mask": px(ctx, cx - hs / 2, hy + 2, hs, hs - 3, "#f8f9fa"); px(ctx, cx - 2, hy + 3, 1, 1, "#000"); px(ctx, cx + 1, hy + 3, 1, 1, "#000"); break;
        case "volcano": px(ctx, cx - hs / 2 + 1, hy - 4, hs - 2, 4, c); px(ctx, cx - 1, hy - 5, 2, 2, "#d00000"); px(ctx, cx - 2, hy - 8 - frame, 4, 3, "rgba(80,80,80,0.6)"); break;
        case "flowers": px(ctx, cx - hs / 2, hy - 2, hs, 2, "#3a5a40"); px(ctx, cx - 3, hy + 2, 2, 2, "#80b918"); px(ctx, cx + 1, hy + 2, 2, 2, "#80b918"); px(ctx, cx - hs / 2 - 2, hy - 4, 3, 3, "#ffafcc"); px(ctx, cx + hs / 2, hy - 4, 3, 3, "#ffafcc"); break;
        case "stitches": for (let i = 0; i < 4; i++) px(ctx, cx - w / 2 + i * 2, torsoY + 3 + i, 3, 1, "#4a0000"); break;
        case "marks": px(ctx, cx - hs / 2 + 1, hy + hs - 3, 2, 1, "#111"); px(ctx, cx + hs / 2 - 3, hy + hs - 3, 2, 1, "#111"); px(ctx, cx - 1, hy, 2, 2, "#111"); break;
        case "uniform": px(ctx, cx - 1, torsoY, 2, torsoH, "#fca311"); break;
        case "robe": px(ctx, cx - 1, torsoY, 2, torsoH, "#f8f9fa"); px(ctx, cx - hs / 2, hy - 1, hs, 2, "#111"); px(ctx, cx - hs / 2, hy + 1, 2, 4, "#111"); px(ctx, cx - hs / 2, hy + 1, hs, 1, "#7f1d1d"); break;
      }
      break;
    }
    case "insect": {
      const ry = S * 0.18, rx = S * 0.34;
      const y = base - ry - 3 - b;
      const legs = sp.extra === "legs" ? 8 : 6;
      for (let i = 0; i < legs; i++) {
        const lx = cx - rx + 2 + (i % (legs / 2)) * ((rx * 2 - 4) / (legs / 2 - 1 || 1));
        const dir = i < legs / 2 ? -1 : 1;
        px(ctx, lx, y, 1, ry + 3, shade(c, -0.4));
        px(ctx, lx + dir * (frame ? 1 : 2), base - 2, 2, 1, shade(c, -0.4));
      }
      blobShape(ctx, cx + rx * 0.3, y, rx * 0.7, ry, c, r, 0.08);
      blobShape(ctx, cx - rx * 0.5, y - 2, rx * 0.45, ry * 0.9, shade(c, 0.1), r, 0.05);
      eyesAt(ctx, eyeList(cx - rx * 0.8, y - ry, rx * 0.6, ry, 2), c2, "#a4161a");
      if (sp.mouth === "grin") grin(ctx, cx - rx * 0.8, y + 1, Math.round(rx * 0.6));
      if (sp.extra === "arms") { px(ctx, cx - rx - 3, y - ry - 2, 2, ry + 2, c2); px(ctx, cx - rx - 5, y - ry - 3, 4, 2, c2); }
      break;
    }
    case "beast": {
      const rx = S * 0.36, ry = S * 0.22;
      const y = base - ry - 3;
      for (let i = 0; i < 4; i++) px(ctx, cx - rx + 3 + i * (rx * 0.6), y + ry - 2, 3, 5 - (i % 2 === frame ? 1 : 0), shade(c, -0.3));
      blobShape(ctx, cx, y - b, rx, ry, c, r, 0.1);
      if (sp.extra === "ribs") for (let i = 0; i < 4; i++) px(ctx, cx - 4 + i * 3, y - ry / 2 - b, 1, ry, "#e9ecef");
      if (sp.extra === "hands") { px(ctx, cx - rx, y + ry - 2, 4, 3, "#f1c27d"); px(ctx, cx + rx - 4, y + ry - 2, 4, 3, "#f1c27d"); }
      const hx = cx - rx - 2;
      blobShape(ctx, hx + 4, y - ry / 2 - b, ry * 0.9, ry * 0.8, shade(c, 0.05), r, 0.05);
      if (sp.mouth === "maw") { px(ctx, hx, y - b, ry * 1.6, 2, "#1a0000"); for (let i = 0; i < ry * 1.6; i += 2) px(ctx, hx + i, y - b, 1, 1, "#fff"); }
      if (nEyes > 0) eyesAt(ctx, [[hx + 1, y - ry - b, 2], [hx + 5, y - ry - b, 2]], c2, "#7f0000");
      else { px(ctx, hx + 1, y - ry - b, 2, 2, "#ff6d00"); px(ctx, hx + 5, y - ry - b, 2, 2, "#ff6d00"); }
      break;
    }
    case "serpent": {
      const seg = 10;
      for (let i = seg; i >= 0; i--) {
        const t = i / seg;
        const x = cx - S * 0.4 + t * S * 0.8;
        const y = base - 6 - Math.sin(t * Math.PI * 2 + frame * 0.6) * S * 0.12 - (1 - t) * S * 0.3;
        const rr = 2 + (1 - t) * S * 0.08;
        const col = sp.extra === "rainbow" ? ["#ff006e", "#fb5607", "#ffbe0b", "#3a86ff", "#8338ec"][i % 5] : i % 2 ? c : shade(c, -0.12);
        blobShape(ctx, x, y, rr, rr, col, r, 0);
        if (sp.extra === "windows" && i % 2 === 0) px(ctx, x - 1, y - 1, 2, 2, "#ffd60a");
        if (i === 0) {
          eyesAt(ctx, [[x - rr + 1, y - rr + 1, 2], [x + rr - 3, y - rr + 1, 2]], c2, "#111");
          if (nEyes > 2) eyesAt(ctx, [[x - 1, y - 1, 2]], c2);
          if (sp.mouth === "maw") px(ctx, x - rr, y + rr / 2, rr * 2, 2, "#1a0000");
        }
      }
      break;
    }
    case "giant": {
      const rx = S * 0.42, ry = S * 0.36;
      const y = base - ry - b;
      blobShape(ctx, cx, y, rx, ry, c, r, 0.12);
      if (sp.extra === "arms") {
        px(ctx, cx - rx - 3, y - 4, 5, ry + 4, shade(c, -0.15)); px(ctx, cx + rx - 2, y - 4, 5, ry + 4, shade(c, -0.15));
        px(ctx, cx - rx - 5, y + ry - 2, 8, 4, shade(c, -0.25)); px(ctx, cx + rx - 3, y + ry - 2, 8, 4, shade(c, -0.25));
      }
      if (sp.extra === "coffin") { px(ctx, cx + rx - 6, y - ry - 6, 6, 14, "#495057"); px(ctx, cx + rx - 5, y - ry - 4, 4, 1, "#adb5bd"); }
      eyesAt(ctx, eyeList(cx - rx + 4, y - ry + 3, rx * 2 - 8, ry * 0.8, 3), c2, "#7f0000");
      px(ctx, cx - rx / 2, y + ry / 4, rx, 4, "#1a0000");
      for (let i = 0; i < rx; i += 2) { px(ctx, cx - rx / 2 + i, y + ry / 4, 1, 2, "#fff"); px(ctx, cx - rx / 2 + i + 1, y + ry / 4 + 3, 1, 1, "#fff"); }
      break;
    }
    default:
      blobShape(ctx, cx, base - S * 0.3, S * 0.3, S * 0.3, c, r);
  }
}

// ---------- tiles ----------

export const T = {
  DEEP: 0, WATER: 1, SAND: 2, GRASS: 3, FLOWERS: 4, FOREST: 5, ROCK: 6, HILL: 7, DIRT: 8, PAVED: 9, PLAZA: 10,
  BUILDING: 11, BRIDGE: 12, CURSED: 13, FIELD: 14, SIDEWALK: 15, PEAK: 16, PATH: 17, GRAVEL: 18, WALL: 19,
};

export const WALKABLE = new Set([T.SAND, T.GRASS, T.FLOWERS, T.HILL, T.DIRT, T.PAVED, T.PLAZA, T.BRIDGE, T.CURSED, T.FIELD, T.SIDEWALK, T.PATH, T.GRAVEL]);

export const MINIMAP_COLORS = {
  0: "#163a73", 1: "#2a63ae", 2: "#dccd96", 3: "#5aa046", 4: "#78b450", 5: "#1e6428", 6: "#7a756e", 7: "#6e8c50", 8: "#aa8250",
  9: "#46464b", 10: "#beb4aa", 11: "#a24a3a", 12: "#8c643c", 13: "#6e3c78", 14: "#bea850", 15: "#969696", 16: "#f0f0f5", 17: "#b99664", 18: "#c8c3b4", 19: "#504638",
};

function speckle(ctx, x0, y0, colors, density, tx, ty, s = 1) {
  for (let y = 0; y < 16; y++) for (let x = 0; x < 16; x++) {
    const h = hash(tx * 16 + x, ty * 16 + y, s);
    if (h < density) px(ctx, x0 + x, y0 + y, 1, 1, colors[Math.floor(hash(tx * 16 + x, ty * 16 + y, s + 7) * colors.length)]);
  }
}

// Draws one tile at (x0,y0) on ctx. at(tx,ty) gives neighbouring tile types for edges.
export function drawTile(ctx, t, x0, y0, tx, ty, at) {
  const fill = (c) => px(ctx, x0, y0, 16, 16, c);
  const grassBase = () => { fill("#5aa046"); speckle(ctx, x0, y0, ["#4e9440", "#66ad50", "#4a8a3a"], 0.35, tx, ty, 1); };
  switch (t) {
    case T.DEEP: fill("#163a73"); speckle(ctx, x0, y0, ["#1a4282", "#13336a"], 0.2, tx, ty, 2); break;
    case T.WATER: fill("#2a63ae"); speckle(ctx, x0, y0, ["#3170bd", "#2558a0"], 0.2, tx, ty, 3); break;
    case T.SAND: fill("#dccd96"); speckle(ctx, x0, y0, ["#cfbf86", "#e6d8a6"], 0.3, tx, ty, 4); break;
    case T.GRASS: grassBase();
      if (hash(tx, ty, 9) < 0.18) { px(ctx, x0 + 4, y0 + 8, 1, 3, "#3e7a32"); px(ctx, x0 + 5, y0 + 7, 1, 3, "#3e7a32"); px(ctx, x0 + 11, y0 + 3, 1, 3, "#3e7a32"); }
      break;
    case T.FLOWERS: grassBase(); speckle(ctx, x0, y0, ["#ffafcc", "#ffd166", "#f8f9fa", "#cdb4db"], 0.06, tx, ty, 5); break;
    case T.FIELD: fill("#b9a24c"); for (let i = 0; i < 16; i += 4) px(ctx, x0, y0 + i, 16, 1, "#9c8838"); speckle(ctx, x0, y0, ["#d4bd60"], 0.15, tx, ty, 6); break;
    case T.HILL: fill("#6e8c50"); speckle(ctx, x0, y0, ["#5f7c44", "#7c9a5c", "#8a8a6e"], 0.35, tx, ty, 7); break;
    case T.FOREST: {
      grassBase();
      fill("#2f6b2f");
      const ox = Math.floor(hash(tx, ty, 11) * 3) - 1;
      px(ctx, x0 + 7 + ox, y0 + 11, 2, 5, "#5c3d1e");
      ctx.fillStyle = "#1e5426"; ctx.beginPath(); ctx.arc(x0 + 8 + ox, y0 + 7, 7, 0, Math.PI * 2); ctx.fill();
      ctx.fillStyle = "#2a7a34"; ctx.beginPath(); ctx.arc(x0 + 7 + ox, y0 + 6, 5, 0, Math.PI * 2); ctx.fill();
      px(ctx, x0 + 5 + ox, y0 + 3, 3, 2, "#45a34f");
      break;
    }
    case T.ROCK: fill("#6e8c50"); {
      ctx.fillStyle = "#7a756e"; ctx.beginPath(); ctx.moveTo(x0, y0 + 16); ctx.lineTo(x0 + 6, y0 + 2); ctx.lineTo(x0 + 10, y0 + 6); ctx.lineTo(x0 + 16, y0 + 16); ctx.fill();
      px(ctx, x0 + 5, y0 + 3, 2, 6, "#9a958d"); px(ctx, x0 + 10, y0 + 9, 4, 2, "#5e5953");
    } break;
    case T.PEAK: fill("#a9a6a0"); {
      ctx.fillStyle = "#f0f0f5"; ctx.beginPath(); ctx.moveTo(x0, y0 + 16); ctx.lineTo(x0 + 8, y0); ctx.lineTo(x0 + 16, y0 + 16); ctx.fill();
      px(ctx, x0 + 9, y0 + 4, 2, 8, "#d0d0dc");
    } break;
    case T.DIRT: case T.PATH: {
      const base = t === T.DIRT ? "#aa8250" : "#b99664";
      fill(base); speckle(ctx, x0, y0, [shade(base, -0.12), shade(base, 0.12)], 0.25, tx, ty, 8);
      // grass fringe where it meets grass
      const g = "#5aa046";
      const isG = (v) => v === T.GRASS || v === T.FLOWERS || v === T.FOREST || v === T.HILL;
      if (isG(at(tx, ty - 1))) for (let i = 0; i < 16; i++) if (hash(tx * 16 + i, ty, 12) < 0.6) px(ctx, x0 + i, y0, 1, 1, g);
      if (isG(at(tx, ty + 1))) for (let i = 0; i < 16; i++) if (hash(tx * 16 + i, ty, 13) < 0.6) px(ctx, x0 + i, y0 + 15, 1, 1, g);
      if (isG(at(tx - 1, ty))) for (let i = 0; i < 16; i++) if (hash(tx, ty * 16 + i, 14) < 0.6) px(ctx, x0, y0 + i, 1, 1, g);
      if (isG(at(tx + 1, ty))) for (let i = 0; i < 16; i++) if (hash(tx, ty * 16 + i, 15) < 0.6) px(ctx, x0 + 15, y0 + i, 1, 1, g);
      break;
    }
    case T.PAVED: fill("#46464b"); speckle(ctx, x0, y0, ["#3e3e43", "#505055"], 0.2, tx, ty, 16);
      if (at(tx, ty - 1) === T.PAVED && at(tx, ty + 1) === T.PAVED && tx % 2 === 0) px(ctx, x0 + 7, y0 + 4, 2, 8, "#d8d8b0");
      else if (at(tx - 1, ty) === T.PAVED && at(tx + 1, ty) === T.PAVED && ty % 2 === 0) px(ctx, x0 + 4, y0 + 7, 8, 2, "#d8d8b0");
      break;
    case T.SIDEWALK: fill("#a5a5a5"); px(ctx, x0, y0, 16, 1, "#8d8d8d"); px(ctx, x0, y0, 1, 16, "#8d8d8d"); px(ctx, x0 + 8, y0, 1, 16, "#959595"); break;
    case T.PLAZA: fill("#beb4aa"); for (let i = 0; i < 16; i += 8) { px(ctx, x0, y0 + i, 16, 1, "#a89e94"); px(ctx, x0 + ((ty % 2) * 4) + i, y0, 1, 16, "#a89e94"); } break;
    case T.BRIDGE: fill("#8c643c"); for (let i = 0; i < 16; i += 3) px(ctx, x0, y0 + i, 16, 1, "#6e4c2c"); px(ctx, x0, y0, 1, 16, "#5c3d1e"); px(ctx, x0 + 15, y0, 1, 16, "#5c3d1e"); break;
    case T.CURSED: fill("#4a3b55"); speckle(ctx, x0, y0, ["#5e3c78", "#3a2a48", "#7b2cbf", "#2b1d36"], 0.4, tx, ty, 17);
      if (hash(tx, ty, 18) < 0.25) { px(ctx, x0 + 3, y0 + 5, 6, 1, "#1a0d24"); px(ctx, x0 + 8, y0 + 6, 5, 1, "#1a0d24"); }
      break;
    case T.GRAVEL: fill("#c8c3b4"); speckle(ctx, x0, y0, ["#b4ae9d", "#d8d3c5", "#a8a290"], 0.4, tx, ty, 19); break;
    case T.WALL: fill("#6b5d4d"); px(ctx, x0, y0, 16, 4, "#3d342a"); for (let i = 0; i < 16; i += 6) px(ctx, x0 + i, y0 + 4, 1, 12, "#574b3e"); px(ctx, x0, y0 + 10, 16, 1, "#574b3e"); px(ctx, x0, y0 + 3, 16, 1, "#8a7a68"); break;
    case T.BUILDING: fill("#4a4a4a"); break;
    default: fill("#ff00ff");
  }
}

// ---------- buildings ----------

const ROOF = ["#a23b2c", "#3d5a80", "#2a9d8f", "#6c584c", "#9b2226", "#495057"];
const WALL = ["#e9e3d5", "#d6ccc2", "#f2e8cf", "#ccd5ae", "#e5e5e5", "#d4a373"];

export function drawBuilding(ctx, b, x0, y0) {
  const W = b.w * 16, H = b.h * 16;
  const roof = ROOF[b.color % ROOF.length];
  const wall = WALL[(b.color + 2) % WALL.length];
  const kind = b.kind;
  const door = (dx, color = "#5c3d1e") => { px(ctx, x0 + dx - 4, y0 + H - 12, 8, 12, color); px(ctx, x0 + dx + 2, y0 + H - 6, 1, 1, "#e9c46a"); };
  const windows = (y, color = "#9bd1ff", step = 12) => {
    for (let x = 6; x < W - 8; x += step) { px(ctx, x0 + x, y0 + y, 6, 6, "#33415c"); px(ctx, x0 + x + 1, y0 + y + 1, 4, 4, color); }
  };
  if (kind === "gate") {
    // Torii
    px(ctx, x0 + 2, y0 - 6, W - 4, 3, "#c1121f"); px(ctx, x0, y0 - 9, W, 3, "#9d0208");
    px(ctx, x0 + 5, y0 - 3, 3, H + 3, "#c1121f"); px(ctx, x0 + W - 8, y0 - 3, 3, H + 3, "#c1121f");
    return;
  }
  if (kind.startsWith("site_")) {
    drawSite(ctx, b, x0, y0, W, H);
    return;
  }
  const city = ["office", "apartment", "store", "tower", "station"].includes(kind);
  if (city) {
    const body = ["#8d99ae", "#adb5bd", "#6c757d", "#a5a58d", "#b7b7a4", "#7f8c8d"][b.color % 6];
    px(ctx, x0, y0, W, H, body);
    px(ctx, x0, y0, W, 6, shade(body, -0.25));
    px(ctx, x0, y0, 2, H, shade(body, 0.12));
    for (let y = 9; y < H - 14; y += 10) windows(y, hash(b.id, y) > 0.5 ? "#ffe8a3" : "#9bd1ff", 10);
    if (kind === "station") {
      px(ctx, x0 + 6, y0 + 8, W - 12, 10, "#2b2d42");
      ctx.fillStyle = "#f8f9fa"; ctx.font = "bold 8px monospace"; ctx.fillText("SHIBUYA", x0 + W / 2 - 16, y0 + 16);
      px(ctx, x0 + W / 2 - 12, y0 + H - 14, 24, 14, "#212529");
    } else {
      door(W / 2, "#343a40");
      if (kind === "store") px(ctx, x0 + 4, y0 + H - 20, W - 8, 4, ["#e63946", "#2a9d8f", "#f4a261"][b.color % 3]);
    }
    return;
  }
  const roofH = Math.min(H - 14, Math.max(14, Math.floor(H * 0.55)));
  // Walls
  px(ctx, x0 + 1, y0 + roofH - 2, W - 2, H - roofH + 2, wall);
  px(ctx, x0 + 1, y0 + H - 2, W - 2, 2, shade(wall, -0.2));
  // Roof with eaves (Japanese style)
  const temple = kind === "temple" || kind === "shrine_small" || kind === "dorm" || kind === "infirmary";
  const rc = temple ? "#3d405b" : roof;
  px(ctx, x0 - 2, y0 + 2, W + 4, roofH, rc);
  for (let y = 4; y < roofH; y += 3) px(ctx, x0 - 2, y0 + y, W + 4, 1, shade(rc, -0.2));
  px(ctx, x0 - 3, y0 + roofH, W + 6, 2, shade(rc, -0.35));
  px(ctx, x0 + 2, y0 + 2, W - 4, 2, shade(rc, 0.2));
  if (temple) { px(ctx, x0 - 4, y0 + 1, 3, 3, rc); px(ctx, x0 + W + 1, y0 + 1, 3, 3, rc); }
  const wy = roofH + 4;
  if (H - roofH > 14) windows(wy, "#ffe8a3", 14);
  door(W / 2, kind === "shrine_small" || kind === "temple" ? "#9d0208" : "#5c3d1e");
  // Signs
  const sign = { shop: ["#2a9d8f", "店"], inn: ["#e76f51", "宿"], elder: ["#6c584c", "町"], shrine_small: ["#9d0208", "神"], infirmary: ["#f8f9fa", "+"], temple: ["#9d0208", "呪"], dorm: ["#3d5a80", "寮"] }[kind];
  if (sign) {
    px(ctx, x0 + W / 2 - 6, y0 + roofH + 1, 12, 9, sign[0]);
    ctx.fillStyle = sign[0] === "#f8f9fa" ? "#d00000" : "#fff";
    ctx.font = "8px sans-serif";
    ctx.fillText(sign[1], x0 + W / 2 - 4, y0 + roofH + 9);
  }
}

export function drawSite(ctx, b, x0, y0, W, H) {
  const k = b.kind.slice(5);
  switch (k) {
    case "shrine": case "grove": {
      px(ctx, x0 + 2, y0 + 4, W - 4, 3, "#5c1a1b"); px(ctx, x0, y0 + 1, W, 3, "#3d0c0c");
      px(ctx, x0 + 5, y0 + 6, 3, H - 6, "#5c1a1b"); px(ctx, x0 + W - 8, y0 + 6, 3, H - 6, "#5c1a1b");
      for (let i = 0; i < 4; i++) px(ctx, x0 + 10 + i * 6, y0 + 8 + (i % 2) * 3, 1, 8, "#e9ecef");
      break;
    }
    case "cemetery": {
      for (let i = 0; i < 6; i++) {
        const gx = x0 + 3 + (i % 3) * 15, gy = y0 + 6 + Math.floor(i / 3) * 18;
        px(ctx, gx, gy, 8, 12, "#6c757d"); px(ctx, gx + 1, gy - 2, 6, 2, "#6c757d"); px(ctx, gx + 2, gy + 3, 4, 1, "#343a40");
      }
      break;
    }
    case "tunnel": {
      px(ctx, x0, y0, W, H, "#5e5953");
      ctx.fillStyle = "#0b0b0f"; ctx.beginPath(); ctx.arc(x0 + W / 2, y0 + H, W / 2 - 6, Math.PI, 0); ctx.fill();
      px(ctx, x0 + 4, y0 + H - 3, W - 8, 3, "#3d342a");
      break;
    }
    case "lake": {
      px(ctx, x0, y0 + H - 10, W, 10, "#6e4c2c");
      for (let i = 0; i < W; i += 8) px(ctx, x0 + i, y0 + H - 10, 2, 12, "#4a2f1a");
      px(ctx, x0 + W - 12, y0 + 4, 4, H - 14, "#4a2f1a"); px(ctx, x0 + W - 16, y0 + 2, 12, 4, "#ffd166");
      break;
    }
    default: {
      // Abandoned building: cracked, boarded windows, moss.
      const body = { school: "#c9c0b0", hospital: "#dfe7e7", factory: "#7d7d7d", apartment: "#a69f94", detention: "#9a9a8a" }[k] || "#9a9a9a";
      px(ctx, x0, y0 + 4, W, H - 4, body);
      px(ctx, x0, y0, W, 6, shade(body, -0.35));
      for (let y = 10; y < H - 16; y += 10) for (let x = 5; x < W - 8; x += 11) {
        px(ctx, x0 + x, y0 + y, 7, 6, "#1b1b1b");
        if (hash(b.id, x * y) > 0.5) { px(ctx, x0 + x - 1, y0 + y + 2, 9, 2, "#6e4c2c"); }
      }
      px(ctx, x0 + W / 2 - 5, y0 + H - 14, 10, 14, "#0b0b0f");
      for (let i = 0; i < 5; i++) px(ctx, x0 + Math.floor(hash(b.id, i) * W), y0 + 6 + Math.floor(hash(i, b.id) * (H - 10)), 1, 6, "#2b2b2b");
      px(ctx, x0, y0 + H - 4, W, 4, "#4c6b3c");
      if (k === "hospital") { px(ctx, x0 + W - 12, y0 + 8, 8, 2, "#d00000"); px(ctx, x0 + W - 9, y0 + 5, 2, 8, "#d00000"); }
      if (k === "factory") { px(ctx, x0 + W - 10, y0 - 14, 6, 18, "#5c5c5c"); }
      if (k === "school") { px(ctx, x0 + W / 2 - 4, y0 - 6, 8, 8, "#c9c0b0"); px(ctx, x0 + W / 2 - 2, y0 - 4, 4, 4, "#f8f9fa"); }
    }
  }
  // Curse miasma
  for (let i = 0; i < 10; i++) px(ctx, x0 + Math.floor(hash(b.id, i, 3) * W), y0 + Math.floor(hash(i, b.id, 4) * H), 2, 2, "rgba(123,44,191,0.55)");
}
