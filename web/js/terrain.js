// Painterly pixel terrain. Each 32x32-tile chunk is rendered per pixel: natural terrain
// borders are domain-warped so they wander organically instead of following the tile grid,
// coastlines come from a smoothed signed-distance field (so water gets real depth and
// shallows), and every material has its own colour ramp, texture and decals.
import { T, hash } from "./pixel.js";

export const TILE = 16;
export const CHUNK = 32;
const M = 6; // tile margin around a chunk for the distance field

// ---------- noise ----------

export function vnoise(x, y, s) {
  const xi = Math.floor(x), yi = Math.floor(y);
  const xf = x - xi, yf = y - yi;
  const u = xf * xf * (3 - 2 * xf), v = yf * yf * (3 - 2 * yf);
  const a = hash(xi, yi, s), b = hash(xi + 1, yi, s), c = hash(xi, yi + 1, s), d = hash(xi + 1, yi + 1, s);
  return a + (b - a) * u + (c - a) * v + (a - b - c + d) * u * v;
}

export function fbm(x, y, s) {
  return (vnoise(x, y, s) * 0.55 + vnoise(x * 2.03, y * 2.03, s + 1) * 0.3 + vnoise(x * 4.1, y * 4.1, s + 2) * 0.15);
}

// ---------- colour ----------

export function hex(h) {
  const n = parseInt(h.slice(1), 16);
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

function ramp(list) {
  const stops = list.map(hex);
  return (t) => {
    t = t < 0 ? 0 : t > 0.999 ? 0.999 : t;
    const f = t * (stops.length - 1);
    const i = Math.floor(f), k = f - i;
    const a = stops[i], b = stops[i + 1];
    return [a[0] + (b[0] - a[0]) * k, a[1] + (b[1] - a[1]) * k, a[2] + (b[2] - a[2]) * k];
  };
}

const pack = (r, g, b) => (255 << 24) | (Math.max(0, Math.min(255, b | 0)) << 16) | (Math.max(0, Math.min(255, g | 0)) << 8) | Math.max(0, Math.min(255, r | 0));
const mix = (a, b, k) => [a[0] + (b[0] - a[0]) * k, a[1] + (b[1] - a[1]) * k, a[2] + (b[2] - a[2]) * k];

const R = {
  grass: ramp(["#2f6a2c", "#3e7f33", "#4e943b", "#5fa645", "#74b752", "#8cc562"]),
  dry: ramp(["#6f8233", "#859640", "#9caa4f", "#b2bb62"]),
  forest: ramp(["#1d4524", "#25542a", "#2f6431", "#3b7438"]),
  hill: ramp(["#4f6e36", "#5f7f40", "#71904b", "#84a157", "#97b066"]),
  sand: ramp(["#c8ad74", "#d8c088", "#e6d19e", "#f0e0b6"]),
  wetsand: ramp(["#9c8a5e", "#ad9a6c", "#bba87a"]),
  dirt: ramp(["#6e4f31", "#83603b", "#977246", "#a98352"]),
  path: ramp(["#937454", "#a78763", "#b89874", "#c8aa86"]),
  gravel: ramp(["#8f897d", "#a39d90", "#b8b2a5", "#cdc8bb"]),
  rock: ramp(["#5d5c58", "#6f6e68", "#83817a", "#98968e", "#adaba2"]),
  snow: ramp(["#a9b6c9", "#c4cfde", "#dde5ef", "#eef3f9", "#ffffff"]),
  cursed: ramp(["#1d1226", "#291832", "#35203f", "#43284d"]),
  soil: ramp(["#5b3d22", "#6b4a2b", "#7a5634"]),
  crop: ramp(["#5f8a2a", "#7a9d32", "#9ab040", "#c2b44e", "#d8c160"]),
  water: ramp(["#7fd8cf", "#52b9c6", "#3792bf", "#2a71b0", "#225a9c", "#1c4787", "#173a73"]),
  asphalt: ramp(["#34363b", "#3c3e44", "#45474d"]),
  paver: ramp(["#9d9b95", "#aaa8a1", "#b7b5ae", "#c3c1ba"]),
  plaza: ramp(["#a49b8c", "#b3aa9b", "#c2b9aa", "#cfc7b9"]),
  plank: ramp(["#6e4a2c", "#7f5833", "#8f653b", "#a07345"]),
};

const NATURAL = new Set([T.DEEP, T.WATER, T.SAND, T.GRASS, T.FLOWERS, T.FOREST, T.ROCK, T.HILL, T.DIRT, T.CURSED, T.FIELD, T.PEAK, T.PATH, T.GRAVEL]);
const isWater = (t) => t === T.DEEP || t === T.WATER || t === T.BRIDGE;
const GREEN = new Set([T.GRASS, T.FLOWERS, T.FOREST, T.HILL]);

// ---------- distance field ----------

// Signed distance (in tiles) from each tile centre to the coast: >0 water, <0 land.
function coastField(at, tx0, ty0, N) {
  const INF = 99;
  const dW = new Float32Array(N * N), dL = new Float32Array(N * N);
  for (let y = 0; y < N; y++) for (let x = 0; x < N; x++) {
    const w = isWater(at(tx0 + x, ty0 + y));
    dW[y * N + x] = w ? INF : 0;
    dL[y * N + x] = w ? 0 : INF;
  }
  const D = 1.4142;
  for (const d of [dW, dL]) {
    for (let y = 0; y < N; y++) for (let x = 0; x < N; x++) {
      const i = y * N + x;
      let v = d[i];
      if (x > 0) v = Math.min(v, d[i - 1] + 1);
      if (y > 0) {
        v = Math.min(v, d[i - N] + 1);
        if (x > 0) v = Math.min(v, d[i - N - 1] + D);
        if (x < N - 1) v = Math.min(v, d[i - N + 1] + D);
      }
      d[i] = v;
    }
    for (let y = N - 1; y >= 0; y--) for (let x = N - 1; x >= 0; x--) {
      const i = y * N + x;
      let v = d[i];
      if (x < N - 1) v = Math.min(v, d[i + 1] + 1);
      if (y < N - 1) {
        v = Math.min(v, d[i + N] + 1);
        if (x < N - 1) v = Math.min(v, d[i + N + 1] + D);
        if (x > 0) v = Math.min(v, d[i + N - 1] + D);
      }
      d[i] = v;
    }
  }
  const F = new Float32Array(N * N);
  for (let i = 0; i < N * N; i++) F[i] = dW[i] > 0 ? Math.min(dW[i], 8) - 0.5 : -(Math.min(dL[i], 8) - 0.5);
  return F;
}

function sampleF(F, N, fx, fy) {
  const x0 = Math.max(0, Math.min(N - 2, Math.floor(fx))), y0 = Math.max(0, Math.min(N - 2, Math.floor(fy)));
  const kx = Math.max(0, Math.min(1, fx - x0)), ky = Math.max(0, Math.min(1, fy - y0));
  const a = F[y0 * N + x0], b = F[y0 * N + x0 + 1], c = F[(y0 + 1) * N + x0], d = F[(y0 + 1) * N + x0 + 1];
  return a + (b - a) * kx + (c - a) * ky + (a - b - c + d) * kx * ky;
}

// ---------- material shaders (return [r,g,b]) ----------

function landColor(t, X, Y, f, nb) {
  const n1 = vnoise(X / 70, Y / 70, 5);
  const n2 = vnoise(X / 17, Y / 17, 6);
  const j = hash(X >> 1, Y >> 1, 7) - 0.5;
  switch (t) {
    case T.GRASS: case T.FLOWERS: {
      let c = R.grass(0.15 + 0.5 * n1 + 0.3 * n2 + j * 0.16);
      const n3 = vnoise(X / 150, Y / 150, 8);
      if (n3 > 0.6) c = mix(c, R.dry(n2 + j * 0.2), Math.min(1, (n3 - 0.6) * 4) * 0.7);
      if (f > -0.22) c = mix(c, [90, 96, 52], 0.35); // muddy bank
      return c;
    }
    case T.FOREST: return R.forest(0.2 + 0.45 * n1 + 0.3 * n2 + j * 0.2);
    case T.HILL: {
      // fake relief: light from the top-left
      const g = vnoise((X - 3) / 22, (Y - 3) / 22, 11) - vnoise(X / 22, Y / 22, 11);
      return R.hill(0.45 + g * 3 + 0.25 * n1 + j * 0.15);
    }
    case T.SAND: {
      if (f > -0.35) return R.wetsand(0.5 + j * 0.4 + (f + 0.35) * 0.5);
      const dune = Math.sin((X + vnoise(X / 30, Y / 30, 12) * 40) / 7) * 0.12;
      return R.sand(0.35 + 0.35 * n2 + dune + j * 0.15);
    }
    case T.DIRT: return R.dirt(0.35 + 0.4 * n2 + j * 0.3);
    case T.PATH: return R.path(0.35 + 0.35 * n2 + j * 0.3);
    case T.GRAVEL: return R.gravel(0.45 + 0.2 * n2 + (hash(X, Y, 9) - 0.5) * 0.7);
    case T.FIELD: {
      const row = ((Y >> 2) & 1) === 0;
      const ripe = vnoise(Math.floor(X / 16) / 3, Math.floor(Y / 16) / 3, 13);
      if (!row) return R.soil(0.4 + j);
      const stalk = hash(X, Y >> 2, 14) < 0.75;
      return stalk ? R.crop(ripe * 0.8 + (Y & 3) * 0.06 + j * 0.15) : R.soil(0.6 + j);
    }
    case T.ROCK: {
      const g = vnoise((X - 3) / 14, (Y - 3) / 14, 15) - vnoise(X / 14, Y / 14, 15);
      const c = R.rock(0.5 + g * 3 + j * 0.2);
      return n2 > 0.62 ? mix(c, R.hill(0.4), 0.55) : c;
    }
    case T.PEAK: {
      const g = vnoise((X - 3) / 18, (Y - 3) / 18, 16) - vnoise(X / 18, Y / 18, 16);
      return R.snow(0.6 + g * 3.5 + j * 0.1);
    }
    case T.CURSED: {
      const v = Math.abs(vnoise(X / 13, Y / 13, 17) - 0.5);
      if (v < 0.025) return [190, 80, 255];
      if (v < 0.05) return [110, 40, 150];
      return R.cursed(0.3 + 0.4 * n2 + j * 0.3);
    }
  }
  return R.grass(0.5);
}

function waterColor(d, X, Y, deep) {
  if (deep) d = Math.max(d, 2.2 + d * 0.5);
  const n = vnoise(X / 40, Y / 40, 18);
  let c = R.water(Math.min(1, d / 6.5) + (n - 0.5) * 0.08);
  if (d < 1.4) {
    // caustics in the shallows
    const cv = Math.abs(vnoise(X / 9, Y / 9, 19) - 0.5) + Math.abs(vnoise(X / 5, Y / 5, 20) - 0.5) * 0.5;
    if (cv < 0.06) c = mix(c, [210, 245, 240], 0.35 * (1 - d / 1.4));
  }
  return c;
}

function hardColor(t, lx, ly, tx, ty, at, X, Y) {
  const j = hash(X, Y, 23) - 0.5;
  switch (t) {
    case T.PAVED: {
      let c = R.asphalt(0.5 + j * 0.5 + (vnoise(X / 20, Y / 20, 24) - 0.5) * 0.6);
      const up = at(tx, ty - 1) === T.PAVED, dn = at(tx, ty + 1) === T.PAVED, lf = at(tx - 1, ty) === T.PAVED, rt = at(tx + 1, ty) === T.PAVED;
      if (up && dn && !(lf && rt) && tx % 2 === 0 && lx >= 7 && lx <= 8 && ly >= 3 && ly <= 12) c = [222, 214, 170];
      else if (lf && rt && !(up && dn) && ty % 2 === 0 && ly >= 7 && ly <= 8 && lx >= 3 && lx <= 12) c = [222, 214, 170];
      if ((!up && ly === 0) || (!lf && lx === 0)) c = [40, 41, 45];
      if ((!dn && ly === 15) || (!rt && lx === 15)) c = mix(c, [130, 130, 128], 0.5);
      if (hash(tx, ty, 25) < 0.08 && Math.abs(lx - 4 - ly * 0.6) < 0.6 && ly > 4 && ly < 13) c = [30, 31, 34];
      return c;
    }
    case T.SIDEWALK: {
      const px = lx >> 3, py = ly >> 3;
      if (lx % 8 === 0 || ly % 8 === 0) return [126, 124, 118];
      let c = R.paver(hash(tx * 2 + px, ty * 2 + py, 26) * 0.8 + j * 0.15);
      if (lx % 8 === 1 || ly % 8 === 1) c = mix(c, [255, 255, 255], 0.15);
      if (at(tx, ty - 1) === T.PAVED && ly < 2) c = [92, 92, 90];
      return c;
    }
    case T.PLAZA: {
      const row = ly >> 2, off = (row + ty * 4) & 1 ? 4 : 0;
      const bx = (lx + off) >> 3;
      if (ly % 4 === 0 || (lx + off) % 8 === 0) return [128, 120, 108];
      let c = R.plaza(hash(tx * 3 + bx, ty * 4 + row, 27) + j * 0.2);
      if (ly % 4 === 1) c = mix(c, [255, 250, 240], 0.18);
      if (ly % 4 === 3) c = mix(c, [60, 50, 40], 0.12);
      return c;
    }
    case T.BRIDGE: {
      const vert = isWater(at(tx - 1, ty)) && isWater(at(tx + 1, ty)) ? true : !(isWater(at(tx, ty - 1)) && isWater(at(tx, ty + 1)));
      const a = vert ? lx : ly, b = vert ? ly : lx; // a across the bridge, b along it
      if (a <= 1 || a >= 14) return a === 0 || a === 15 ? [52, 34, 20] : [95, 64, 38];
      const plank = b >> 2;
      if (b % 4 === 3) return [58, 38, 22];
      let c = R.plank(hash(tx * 4 + plank, ty * 4 + plank, 28) + j * 0.3);
      if ((a === 4 || a === 11) && b % 4 === 1) c = [50, 40, 34]; // nails
      return c;
    }
    case T.WALL: {
      if (ly < 2) return [70, 76, 92];
      if (ly < 6) return ly === 2 ? [128, 136, 152] : (lx + ly) % 4 === 0 ? [58, 63, 78] : [86, 93, 110];
      if (ly === 6) return [40, 40, 46];
      if (ly < 13) return mix([232, 225, 210], [200, 192, 176], (ly - 7) / 6 + j * 0.1);
      return R.rock(0.4 + hash(tx * 4 + (lx >> 2), ly, 29) * 0.4);
    }
    case T.BUILDING: return [44, 40, 38];
  }
  return [255, 0, 255];
}

// ---------- decals painted into the pixel buffer ----------

const FLOWER = ["#ff8fab", "#ffffff", "#ffd166", "#cdb4db", "#ff595e", "#a0c4ff"].map(hex);

function decals(buf, t, tx, ty, lx0, ly0, put, f) {
  const h = (i) => hash(tx, ty, 40 + i);
  if (t === T.GRASS || t === T.FOREST || t === T.HILL || t === T.FLOWERS) {
    const n = Math.floor(h(0) * (t === T.FOREST ? 2 : 4));
    for (let i = 0; i < n; i++) {
      const x = lx0 + 2 + Math.floor(h(i * 2 + 1) * 12), y = ly0 + 3 + Math.floor(h(i * 2 + 2) * 11);
      const dark = t === T.FOREST ? [22, 52, 26] : [44, 96, 40], lite = t === T.FOREST ? [60, 110, 56] : [130, 196, 96];
      put(x, y, dark); put(x - 1, y - 1, dark); put(x + 1, y - 1, dark); put(x, y - 2, lite); put(x - 1, y - 2, lite); put(x + 2, y - 2, lite);
    }
  }
  if (t === T.FLOWERS || (t === T.GRASS && h(10) < 0.05)) {
    const n = t === T.FLOWERS ? 3 + Math.floor(h(11) * 5) : 2;
    const col = FLOWER[Math.floor(h(12) * FLOWER.length)];
    for (let i = 0; i < n; i++) {
      const x = lx0 + 2 + Math.floor(h(i + 13) * 12), y = ly0 + 2 + Math.floor(h(i + 23) * 12);
      const c = h(i + 33) < 0.7 ? col : FLOWER[Math.floor(h(i + 43) * FLOWER.length)];
      put(x, y + 1, [40, 90, 36]);
      put(x - 1, y, c); put(x + 1, y, c); put(x, y - 1, c); put(x, y + 0, [255, 220, 90]);
    }
  }
  if ((t === T.DIRT || t === T.PATH || t === T.SAND) && h(50) < 0.6) {
    for (let i = 0; i < 3; i++) {
      const x = lx0 + 1 + Math.floor(h(51 + i) * 14), y = ly0 + 1 + Math.floor(h(54 + i) * 14);
      const c = t === T.SAND ? (h(57 + i) < 0.3 ? [250, 240, 230] : [176, 150, 108]) : [120, 104, 88];
      put(x, y, c); put(x + 1, y, mix(c, [255, 255, 255], 0.3)); put(x, y + 1, mix(c, [0, 0, 0], 0.3));
    }
  }
  if (t === T.CURSED && h(60) < 0.3) {
    const x = lx0 + 3 + Math.floor(h(61) * 9), y = ly0 + 5 + Math.floor(h(62) * 8);
    for (let i = 0; i < 4; i++) put(x + i, y, [220, 214, 200]);
    put(x - 1, y - 1, [220, 214, 200]); put(x - 1, y + 1, [220, 214, 200]); put(x + 4, y - 1, [220, 214, 200]); put(x + 4, y + 1, [220, 214, 200]);
  }
}

// ---------- chunk renderer ----------

// Returns { ground: canvas, field: Float32Array(N*N), N, foam: canvas }.
export function renderChunk(at, cx, cy) {
  const S = CHUNK * TILE;
  const tx0 = cx * CHUNK - M, ty0 = cy * CHUNK - M, N = CHUNK + 2 * M;
  const F = coastField(at, tx0, ty0, N);
  const cv = document.createElement("canvas");
  cv.width = cv.height = S;
  const ctx = cv.getContext("2d");
  const img = ctx.createImageData(S, S);
  const buf = new Uint32Array(img.data.buffer);
  const foamCv = document.createElement("canvas");
  foamCv.width = foamCv.height = S;
  const fctx = foamCv.getContext("2d");
  const fimg = fctx.createImageData(S, S);
  const fbuf = new Uint32Array(fimg.data.buffer);
  const X0 = cx * CHUNK * TILE, Y0 = cy * CHUNK * TILE;
  const fieldAt = new Float32Array(S * S);

  for (let py = 0; py < S; py++) {
    const Y = Y0 + py;
    const ty = Math.floor(Y / TILE);
    for (let px = 0; px < S; px++) {
      const X = X0 + px;
      const tx = Math.floor(X / TILE);
      const t0 = at(tx, ty);
      let c;
      const i = py * S + px;
      if (!NATURAL.has(t0) && t0 !== T.BRIDGE) {
        c = hardColor(t0, X - tx * TILE, Y - ty * TILE, tx, ty, at, X, Y);
        fieldAt[i] = -9;
      } else {
        let f = sampleF(F, N, X / TILE - 0.5 - tx0, Y / TILE - 0.5 - ty0);
        f += (vnoise(X / 10, Y / 10, 21) - 0.5) * 0.55 + (vnoise(X / 3.5, Y / 3.5, 22) - 0.5) * 0.12;
        fieldAt[i] = f;
        if (t0 === T.BRIDGE) {
          c = hardColor(t0, X - tx * TILE, Y - ty * TILE, tx, ty, at, X, Y);
        } else if (f > 0) {
          c = waterColor(f, X, Y, t0 === T.DEEP);
          if (f < 0.09) c = mix(c, [235, 250, 248], 0.75);
        } else {
          const wx = X + (vnoise(X / 11, Y / 11, 31) - 0.5) * 12, wy = Y + (vnoise(X / 11, Y / 11, 32) - 0.5) * 12;
          let t = at(Math.floor(wx / TILE), Math.floor(wy / TILE));
          if (!NATURAL.has(t) || isWater(t)) t = t0;
          if (isWater(t)) {
            // a land pixel inside a water tile: borrow the nearest land neighbour
            t = T.SAND;
            for (const [dx, dy] of [[0, 1], [0, -1], [1, 0], [-1, 0], [1, 1], [-1, -1], [1, -1], [-1, 1]]) {
              const n = at(tx + dx, ty + dy);
              if (NATURAL.has(n) && !isWater(n)) { t = n; break; }
            }
          }
          c = landColor(t, X, Y, f);
        }
        // Foam bands for the animated shoreline (stored separately)
        if (f > 0.1 && f < 0.5) {
          const band = Math.abs(((f - 0.1) * 7 + vnoise(X / 6, Y / 6, 33) * 1.4) % 2 - 1);
          if (band < 0.22) fbuf[i] = pack(240, 252, 255);
        }
      }
      buf[i] = pack(c[0], c[1], c[2]);
    }
  }

  // Decals
  const put = (x, y, c) => {
    if (x < 0 || y < 0 || x >= S || y >= S) return;
    const i = y * S + x;
    if (fieldAt[i] > -0.15 && fieldAt[i] !== -9) return; // keep water/coast clean
    buf[i] = pack(c[0], c[1], c[2]);
  };
  for (let y = 0; y < CHUNK; y++) for (let x = 0; x < CHUNK; x++) {
    const tx = cx * CHUNK + x, ty = cy * CHUNK + y;
    const t = at(tx, ty);
    if (NATURAL.has(t) && !isWater(t)) decals(buf, t, tx, ty, x * TILE, y * TILE, put);
  }

  // Soft contact shadow along the base of walls and building footprints
  for (let py = 1; py < S; py++) for (let px = 0; px < S; px++) {
    const X = X0 + px, Y = Y0 + py;
    const above = at(Math.floor(X / TILE), Math.floor((Y - 1) / TILE));
    const here = at(Math.floor(X / TILE), Math.floor(Y / TILE));
    if ((above === T.WALL || above === T.BUILDING) && here !== above && Y % TILE < 3) {
      const i = py * S + px, v = buf[i];
      const k = 0.62 + (Y % TILE) * 0.1;
      buf[i] = pack((v & 255) * k, ((v >> 8) & 255) * k, ((v >> 16) & 255) * k);
    }
  }
  ctx.putImageData(img, 0, 0);
  fctx.putImageData(fimg, 0, 0);
  return { ground: cv, foam: foamCv, F, N, tx0, ty0 };
}

// Coast distance at a tile position using a rendered chunk's field (for props).
export function fieldAtTile(ch, tx, ty) {
  const x = tx - ch.tx0, y = ty - ch.ty0;
  if (x < 0 || y < 0 || x >= ch.N || y >= ch.N) return -9;
  return ch.F[y * ch.N + x];
}

export { GREEN, NATURAL, isWater, R as RAMPS, mix, pack };
