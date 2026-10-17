// Overworld rendering.
import { buildingSprite, drawBuildingShadow } from "./buildings.js";
import { chunkProps, propSprite } from "./props.js";
import { renderChunk } from "./terrain.js";
import { curseSprite } from "./curses.js";
import { TILE, T, WALKABLE, MINIMAP_COLORS, makeCanvas, charSprite, hash } from "./pixel.js";

const CHUNK = 32;
// collision footprints (half-width, half-depth, lift) in tiles for props you can't walk through
const SOLID = { oak: [0.55, 0.35, 0.1], sakura: [0.55, 0.35, 0.1], autumn: [0.55, 0.35, 0.1], pine: [0.45, 0.3, 0.1], boulder: [1.1, 0.6, 0.5], crystal: [0.6, 0.35, 0.3], dead: [0.35, 0.25, 0.1], stall: [1.5, 0.7, 0.4], fountain: [2.3, 1.1, 0.8], planter: [0.8, 0.45, 0.35], lamp: [0.25, 0.2, 0.1], vending: [0.5, 0.3, 0.2], bench: [0.8, 0.3, 0.2], lantern: [0.3, 0.25, 0.1] };

export class WorldView {
  constructor(world) {
    this.w = world;
    this.tiles = Uint8Array.from(atob(world.tiles), (c) => c.charCodeAt(0));
    this.chunks = new Map();
    this.buildingsByChunk = new Map();
    for (const b of world.buildings) {
      const cx0 = Math.floor((b.x - 1) / CHUNK), cx1 = Math.floor((b.x + b.w + 1) / CHUNK);
      const cy0 = Math.floor((b.y - 1) / CHUNK), cy1 = Math.floor((b.y + b.h + 1) / CHUNK);
      for (let cy = cy0; cy <= cy1; cy++) for (let cx = cx0; cx <= cx1; cx++) {
        const k = cx + "," + cy;
        if (!this.buildingsByChunk.has(k)) this.buildingsByChunk.set(k, []);
        this.buildingsByChunk.get(k).push(b);
      }
    }
    this.minimap = this.buildMinimap();
    this.walkers = this.makeWalkers();
    this.lastT = 0;
  }

  // Townsfolk who just live there: they wander, pause and chat so towns never feel empty.
  makeWalkers() {
    const out = [];
    const palette = ["#14141c", "#22223b", "#264653", "#2a9d8f", "#e9c46a", "#e76f51", "#9d0208", "#f8f9fa", "#6c584c", "#457b9d", "#7b2cbf", "#ff8fab", "#3d405b"];
    const hairs = ["#1b1b1b", "#3b2a1a", "#6b4423", "#a0522d", "#d4a373", "#f1e3c4", "#2d6a4f", "#457b9d"];
    for (const t of this.w.towns) {
      const n = Math.max(3, Math.min(18, Math.round(t.r / 3.2)));
      for (let i = 0; i < n; i++) {
        const h = (k) => hash(t.x + i * 17, t.y + k, 4242);
        const look = { skin: Math.floor(h(1) * 4), hair: Math.floor(h(2) * 12), hairColor: hairs[Math.floor(h(3) * hairs.length)], outfit: Math.floor(h(4) * 8), color1: palette[Math.floor(h(5) * palette.length)], color2: palette[Math.floor(h(6) * palette.length)], eyes: "#3b2a1a", build: Math.floor(h(7) * 3), accessory: h(8) > 0.85 ? Math.floor(1 + h(9) * 7) : 0 };
        let x = t.x, y = t.y;
        for (let tries = 0; tries < 40; tries++) {
          const a = hash(i, tries, t.x) * 6.283, r = Math.sqrt(hash(tries, i, t.y)) * t.r * 0.85;
          const px = t.x + Math.cos(a) * r, py = t.y + Math.sin(a) * r;
          if (this.walkable(px, py)) { x = px; y = py; break; }
        }
        out.push({ town: t, look, x, y, tx: x, ty: y, wait: h(10) * 3, speed: 1.1 + h(11) * 1.1, dir: 0, moving: false, id: t.id + i, awake: h(12) > 0.3 });
      }
    }
    return out;
  }

  stepWalkers(dt, cam, night) {
    for (const wk of this.walkers) {
      if (Math.hypot(wk.x - cam.x, wk.y - cam.y) > 90) { wk.moving = false; continue; }
      if (wk.wait > 0) { wk.wait -= dt; wk.moving = false; continue; }
      let dx = wk.tx - wk.x, dy = wk.ty - wk.y;
      const d = Math.hypot(dx, dy);
      if (d < 0.6) {
        for (let tries = 0; tries < 12; tries++) {
          const a = Math.random() * 6.283, r = Math.sqrt(Math.random()) * wk.town.r * 0.85;
          const px = wk.town.x + Math.cos(a) * r, py = wk.town.y + Math.sin(a) * r;
          if (this.walkable(px, py) && Math.hypot(px - wk.x, py - wk.y) < 40) { wk.tx = px; wk.ty = py; break; }
        }
        wk.wait = 0.5 + Math.random() * 4; wk.moving = false;
        continue;
      }
      const sp = wk.speed * dt, nx = wk.x + (dx / d) * sp, ny = wk.y + (dy / d) * sp;
      if (this.walkable(nx, ny) && this.walkable(nx, ny - 0.4)) {
        wk.x = nx; wk.y = ny; wk.moving = true;
        const sx = Math.abs(dx) > d * 0.38 ? Math.sign(dx) : 0, sy = Math.abs(dy) > d * 0.38 ? Math.sign(dy) : 0;
        wk.dir = sx && sy ? (sy > 0 ? (sx < 0 ? 4 : 5) : (sx < 0 ? 6 : 7)) : sx ? (sx < 0 ? 1 : 2) : sy < 0 ? 3 : 0;
      } else { wk.tx = wk.x; wk.ty = wk.y; wk.wait = 0.3; wk.moving = false; }
    }
  }

  at(x, y) {
    if (x < 0 || y < 0 || x >= this.w.w || y >= this.w.h) return T.DEEP;
    return this.tiles[y * this.w.w + x];
  }

  walkableTile(x, y) { return WALKABLE.has(this.at(Math.floor(x), Math.floor(y))); }

  walkable(x, y) {
    if (!WALKABLE.has(this.at(Math.floor(x), Math.floor(y)))) return false;
    // solid props (trunks, boulders, stalls...) block you too; only chunks already built are consulted
    const c = this.chunks.get(Math.floor(x / CHUNK) + "," + Math.floor(y / CHUNK));
    if (c && c.solids) for (const b of c.solids) if (Math.abs(x - b.x) < b.rx && Math.abs(y - b.y) < b.ry) return false;
    return true;
  }

  chunk(cx, cy) {
    const k = cx + "," + cy;
    let c = this.chunks.get(k);
    if (c) {
      this.chunks.delete(k);
      this.chunks.set(k, c);
      return c;
    }
    const t0 = performance.now();
    const r = renderChunk((x, y) => this.at(x, y), cx, cy);
    const cv = r.ground;
    cv.foam = r.foam;
    const bctx = cv.getContext("2d");
    for (const b of this.buildingsByChunk.get(k) || []) {
      drawBuildingShadow(bctx, b, (b.x - cx * CHUNK) * TILE, (b.y - cy * CHUNK) * TILE);
    }
    cv.props = chunkProps((x, y) => this.at(x, y), cx, cy);
    cv.solids = [];
    for (const p of cv.props) {
      const sd = SOLID[p.kind];
      if (sd) cv.solids.push({ x: p.x, y: p.y - 0.1, rx: sd[0], ry: sd[1] });
    }
    cv.ms = performance.now() - t0;
    this.chunks.set(k, cv);
    if (this.chunks.size > 30) this.chunks.delete(this.chunks.keys().next().value);
    return cv;
  }

  atmosphere(ctx, W, H, v, zoom, now) {
    const { left, top, tpx, ox, oy, tx0, ty0, tx1, ty1 } = v;
    const clock = this.clock ?? 12;
    const night = clock >= 19 || clock < 5;
    // drifting cloud shadows
    if (!WorldView.cloudCv) {
      const [cv, c] = makeCanvas(64, 64);
      const im = c.createImageData(64, 64);
      for (let y = 0; y < 64; y++) for (let x = 0; x < 64; x++) {
        // tileable value noise
        let a = 0, amp = 1, tot = 0;
        for (let o = 0; o < 3; o++) {
          const f = 4 << o, gx = (x / 64) * f, gy = (y / 64) * f;
          const x0 = Math.floor(gx), y0 = Math.floor(gy), fx = gx - x0, fy = gy - y0;
          const hh = (ix, iy) => hash(((ix % f) + f) % f, ((iy % f) + f) % f, 40 + o);
          const sx = fx * fx * (3 - 2 * fx), sy = fy * fy * (3 - 2 * fy);
          const top2 = hh(x0, y0) * (1 - sx) + hh(x0 + 1, y0) * sx, bot = hh(x0, y0 + 1) * (1 - sx) + hh(x0 + 1, y0 + 1) * sx;
          a += (top2 * (1 - sy) + bot * sy) * amp; tot += amp; amp *= 0.5;
        }
        const t = Math.max(0, (a / tot - 0.5) * 3.2);
        const i = (y * 64 + x) * 4;
        im.data[i] = 10; im.data[i + 1] = 20; im.data[i + 2] = 40; im.data[i + 3] = Math.min(255, t * 120);
      }
      c.putImageData(im, 0, 0);
      WorldView.cloudCv = cv;
    }
    if (!night) {
      const size = 56 * tpx, dx = (now / 1000) * 0.9 * tpx, dy = (now / 1000) * 0.35 * tpx;
      const sx0 = ((ox + dx) % size + size) % size - size, sy0 = ((oy + dy) % size + size) % size - size;
      ctx.imageSmoothingEnabled = true;
      ctx.globalAlpha = 0.5;
      for (let x = sx0; x < W; x += size) for (let y = sy0; y < H; y += size) ctx.drawImage(WorldView.cloudCv, x, y, size, size);
      ctx.globalAlpha = 1;
      ctx.imageSmoothingEnabled = false;
    }
    // cursed wisps rising from corrupted ground
    for (let y = ty0; y <= ty1; y++) for (let x = tx0; x <= tx1; x++) {
      if (this.at(x, y) !== T.CURSED) continue;
      const h = hash(x, y, 77);
      if (h > 0.07) continue;
      const ph = ((now / 2600) + h * 14) % 1;
      const sx = ox + (x + 0.5 + Math.sin(ph * 6 + h * 20) * 0.25) * tpx, sy = oy + (y + 0.8) * tpx - ph * 30 * zoom;
      ctx.fillStyle = `rgba(${150 + h * 600 | 0},70,230,${(1 - ph) * 0.5})`;
      ctx.fillRect(Math.round(sx), Math.round(sy), zoom * 2, zoom * (3 + ph * 3));
      ctx.fillStyle = `rgba(230,190,255,${(1 - ph) * 0.4})`;
      ctx.fillRect(Math.round(sx), Math.round(sy), zoom, zoom * 2);
    }
    // sakura petals / leaves by day, fireflies by night (screen-space, wrap around)
    const N = night ? 36 : 44;
    for (let i = 0; i < N; i++) {
      const h1 = hash(i, 1, 5), h2 = hash(i, 2, 5), h3 = hash(i, 3, 5);
      if (night) {
        const px = (h1 * W + Math.sin(now / 1800 + i) * 40 + (now / 40) * (h3 - 0.5)) % W;
        const py = (h2 * H + Math.cos(now / 2300 + i * 2) * 30) % H;
        const a = 0.35 + 0.65 * Math.max(0, Math.sin(now / 500 + i * 1.7));
        ctx.globalCompositeOperation = "lighter";
        const g = ctx.createRadialGradient(px, py, 0, px, py, 7 * zoom);
        g.addColorStop(0, `rgba(210,255,120,${a * 0.9})`); g.addColorStop(1, "rgba(210,255,120,0)");
        ctx.fillStyle = g; ctx.fillRect(px - 8 * zoom, py - 8 * zoom, 16 * zoom, 16 * zoom);
        ctx.globalCompositeOperation = "source-over";
      } else {
        const t = now / 1000;
        const px = (((h1 * W + t * (28 + h3 * 30) + Math.sin(t * 1.6 + i) * 18) % (W + 40)) + W + 40) % (W + 40) - 20;
        const py = (((h2 * H + t * (22 + h3 * 20)) % (H + 40)) + H + 40) % (H + 40) - 20;
        ctx.fillStyle = i % 5 === 0 ? "#d9a13b" : i % 2 ? "#ffc4d6" : "#ffe0ea";
        const flip = Math.sin(t * 5 + i) > 0;
        ctx.fillRect(Math.round(px), Math.round(py), zoom * (flip ? 2 : 1), zoom * (flip ? 1 : 2));
      }
    }
    // vignette
    if (!WorldView.vig || WorldView.vig.width !== W || WorldView.vig.height !== H) {
      const [cv, c] = makeCanvas(W, H);
      const g = c.createRadialGradient(W / 2, H / 2, Math.min(W, H) * 0.45, W / 2, H / 2, Math.hypot(W, H) * 0.56);
      g.addColorStop(0, "rgba(6,8,24,0)"); g.addColorStop(1, "rgba(6,8,24,0.55)");
      c.fillStyle = g; c.fillRect(0, 0, W, H);
      WorldView.vig = cv;
    }
    ctx.drawImage(WorldView.vig, 0, 0);
  }

  buildMinimap() {
    const W = this.w.w, H = this.w.h;
    const [cv, ctx] = makeCanvas(W, H);
    const img = ctx.createImageData(W, H);
    const height = (t) => (t === T.PEAK ? 3 : t === T.ROCK ? 2.2 : t === T.HILL ? 1.4 : t === T.FOREST ? 0.6 : t === T.BUILDING ? 1.2 : 0);
    const land = (t) => t !== T.DEEP && t !== T.WATER;
    for (let y = 0; y < H; y++) for (let x = 0; x < W; x++) {
      const t = this.tiles[y * W + x];
      const n = parseInt((MINIMAP_COLORS[t] || "#000").slice(1), 16);
      let r = (n >> 16) & 255, g = (n >> 8) & 255, bl = n & 255;
      let k = 0.92 + hash(x, y, 3) * 0.16;
      k += (height(this.at(x - 1, y - 1)) - height(this.at(x + 1, y + 1))) * 0.07;
      if (t === T.FOREST && hash(x, y, 4) > 0.55) k *= 0.8;
      if (t === T.WATER || t === T.DEEP) {
        const shore = [this.at(x - 1, y), this.at(x + 1, y), this.at(x, y - 1), this.at(x, y + 1)].some(land);
        if (shore) { r = 120; g = 190; bl = 215; k = 1; }
      } else if (t === T.BUILDING) { r = 200; g = 80; bl = 60; }
      else if (t === T.SAND || land(t)) {
        const wet = [this.at(x - 1, y), this.at(x + 1, y), this.at(x, y - 1), this.at(x, y + 1)].some((q) => q === T.WATER || q === T.DEEP);
        if (wet) k *= 0.88;
      }
      const i = (y * W + x) * 4;
      img.data[i] = Math.min(255, r * k); img.data[i + 1] = Math.min(255, g * k); img.data[i + 2] = Math.min(255, bl * k); img.data[i + 3] = 255;
    }
    ctx.putImageData(img, 0, 0);
    return cv;
  }

  // Draw the world. cam: {x, y} in tiles (center), zoom: screen px per art px.
  draw(ctx, W, H, cam, zoom, now, ents) {
    const tpx = TILE * zoom;
    const left = cam.x - W / tpx / 2, top = cam.y - H / tpx / 2;
    const ox = -left * tpx, oy = -top * tpx;
    ctx.imageSmoothingEnabled = false;
    ctx.fillStyle = "#163a73";
    ctx.fillRect(0, 0, W, H);
    const cx0 = Math.floor(left / CHUNK), cy0 = Math.floor(top / CHUNK);
    const cx1 = Math.floor((left + W / tpx) / CHUNK), cy1 = Math.floor((top + H / tpx) / CHUNK);
    for (let cy = cy0; cy <= cy1; cy++) for (let cx = cx0; cx <= cx1; cx++) {
      if (cx < 0 || cy < 0 || cx * CHUNK >= this.w.w || cy * CHUNK >= this.w.h) continue;
      const c = this.chunk(cx, cy);
      const dx = Math.floor(ox + cx * CHUNK * tpx), dy = Math.floor(oy + cy * CHUNK * tpx);
      ctx.drawImage(c, dx, dy, CHUNK * tpx, CHUNK * tpx);
      if (c.foam) {
        ctx.globalAlpha = 0.45 + 0.35 * Math.sin(now / 700 + cx * 1.3 + cy);
        ctx.drawImage(c.foam, dx, dy, CHUNK * tpx, CHUNK * tpx);
        ctx.globalAlpha = 1;
      }
    }
    // Water shimmer
    const tx0 = Math.max(0, Math.floor(left)), ty0 = Math.max(0, Math.floor(top));
    const tx1 = Math.min(this.w.w - 1, Math.ceil(left + W / tpx)), ty1 = Math.min(this.w.h - 1, Math.ceil(top + H / tpx));
    const phase = Math.floor(now / 400);
    ctx.fillStyle = "rgba(200,230,255,0.55)";
    for (let y = ty0; y <= ty1; y++) for (let x = tx0; x <= tx1; x++) {
      const t = this.at(x, y);
      if (t !== T.WATER && t !== T.DEEP) continue;
      const h = hash(x, y, phase % 7);
      if (h < 0.35) {
        const sx = ox + (x + h * 2.2) * tpx, sy = oy + (y + hash(y, x, phase % 5)) * tpx;
        ctx.fillRect(Math.floor(sx), Math.floor(sy), 3 * zoom, zoom);
      }
    }
    const toScreen = (x, y) => [ox + x * tpx, oy + y * tpx];
    ents = ents.slice();
    const dtw = this.lastT ? Math.min(0.1, (now - this.lastT) / 1000) : 0;
    this.lastT = now;
    const camc = { x: left + W / tpx / 2, y: top + H / tpx / 2 };
    const night = (this.clock ?? 12) >= 21 || (this.clock ?? 12) < 5;
    this.stepWalkers(dtw, camc, night);
    for (const wk of this.walkers) {
      if (night && !wk.awake) continue;
      if (wk.x < left - 2 || wk.x > left + W / tpx + 2 || wk.y < top - 2 || wk.y > top + H / tpx + 4) continue;
      ents.push({ x: wk.x, y: wk.y, draw: (c2, sx, sy, z, t) => drawCharacter(c2, sx, sy, z, wk.look, wk.dir, wk.moving, t + wk.x * 97) });
    }
    this.lights = [];
    const LIT = { lamp: [24, 70], lantern: [12, 46] };
    for (const b of this.w.buildings) {
      if (b.x + b.w + 3 < left || b.x - 3 > left + W / tpx || b.y + b.h + 1 < top || b.y - 6 > top + H / tpx + 4) continue;
      if (["inn", "shop", "temple", "elder", "station", "office", "apartment", "tower", "store", "infirmary", "dorm"].includes(b.kind)) {
        this.lights.push({ x: ox + (b.x + b.w / 2) * tpx, y: oy + (b.y + b.h) * tpx - 12 * zoom, r: Math.max(b.w, 3) * 7 * zoom + 24, warm: true });
      }
      ents.push({ x: b.x + b.w / 2, y: b.y + b.h, draw: (c2, sx, sy, z) => {
        const sp = buildingSprite(b);
        c2.drawImage(sp.cv, Math.round(sx - (b.w * 8 + sp.ox) * z), Math.round(sy - (sp.ext + sp.H) * z), sp.cv.width * z, sp.cv.height * z);
      } });
    }
    for (let cy = cy0; cy <= cy1; cy++) for (let cx = cx0; cx <= cx1; cx++) {
      const c = this.chunks.get(cx + "," + cy);
      if (!c || !c.props) continue;
      for (const p of c.props) {
        if (p.x < left - 3 || p.x > left + W / tpx + 3 || p.y < top - 1 || p.y > top + H / tpx + 4) continue;
        const sp = propSprite(p.kind, p.v);
        if (LIT[p.kind]) this.lights.push({ x: ox + p.x * tpx, y: oy + p.y * tpx - LIT[p.kind][0] * zoom, r: LIT[p.kind][1] * zoom * 0.5 + 24, warm: true });
        ents.push({ x: p.x, y: p.y, draw: (c2, sx, sy, z, t) => {
          c2.fillStyle = "rgba(10,20,30,0.28)";
          c2.beginPath(); c2.ellipse(sx + z * 2, sy, sp.ax * z * 0.55, Math.max(2, sp.cv.height * 0.07) * z, 0, 0, 7); c2.fill();
          const sway = p.kind === "oak" || p.kind === "sakura" || p.kind === "autumn" ? Math.sin(t / 900 + p.x * 3) * 0.4 * z : 0;
          c2.drawImage(sp.cv, Math.round(sx - sp.ax * z + sway), Math.round(sy - sp.ay * z), sp.cv.width * z, sp.cv.height * z);
        } });
      }
    }
    // Entities, y-sorted
    ents.sort((a, b) => a.y - b.y);
    for (const e of ents) {
      const [sx, sy] = toScreen(e.x, e.y);
      if (sx < -80 || sy < -80 || sx > W + 80 || sy > H + 120) continue;
      e.draw(ctx, sx, sy, zoom, now);
    }
    this.atmosphere(ctx, W, H, { left, top, tpx, ox, oy, tx0, ty0, tx1, ty1 }, zoom, now);
    return { left, top, tpx, toScreen, lights: this.lights };
  }
}

// ---------- entity draw helpers ----------

export function drawCharacter(ctx, sx, sy, zoom, look, dir, moving, now, opts = {}) {
  const frame = moving ? Math.floor(now / 140) % 4 : 0;
  const spr = charSprite(look, dir, frame);
  const w = 24 * zoom, h = 36 * zoom;
  const x = Math.round(sx - w / 2), y = Math.round(sy - h + 2 * zoom);
  if (opts.aura) {
    ctx.fillStyle = opts.aura;
    ctx.beginPath(); ctx.ellipse(sx, sy, 9 * zoom, 4 * zoom, 0, 0, Math.PI * 2); ctx.fill();
  }
  if (opts.ghost) ctx.globalAlpha = 0.45;
  ctx.drawImage(spr, x, y, w, h);
  ctx.globalAlpha = 1;
  return y;
}

export function drawCurseEntity(ctx, sx, sy, zoom, spec, id, now, opts = {}) {
  const frame = Math.floor(now / 350) % 2;
  const spr = curseSprite(spec, id, frame);
  const scale = Math.min(zoom, Math.max(1, Math.round((zoom * 28) / spr.height)));
  const w = spr.width * scale, h = spr.height * scale;
  const pulse = 0.35 + 0.15 * Math.sin(now / 300);
  ctx.fillStyle = opts.elite ? `rgba(220,30,30,${pulse})` : `rgba(123,44,191,${pulse})`;
  ctx.beginPath(); ctx.ellipse(sx, sy, w * 0.45, 4 * zoom, 0, 0, Math.PI * 2); ctx.fill();
  ctx.drawImage(spr, Math.round(sx - w / 2), Math.round(sy - h + 2 * zoom), w, h);
  return sy - h;
}

export function nameplate(ctx, text, sx, sy, color = "#fff", sub = "") {
  ctx.font = "bold 12px system-ui, sans-serif";
  ctx.textAlign = "center";
  const w = ctx.measureText(text).width;
  ctx.fillStyle = "rgba(0,0,0,0.55)";
  ctx.fillRect(sx - w / 2 - 4, sy - 14, w + 8, 16);
  ctx.fillStyle = color;
  ctx.fillText(text, sx, sy - 2);
  if (sub) {
    ctx.font = "11px system-ui, sans-serif";
    const w2 = ctx.measureText(sub).width;
    ctx.fillStyle = "rgba(0,0,0,0.45)";
    ctx.fillRect(sx - w2 / 2 - 3, sy - 28, w2 + 6, 13);
    ctx.fillStyle = "#ffd166";
    ctx.fillText(sub, sx, sy - 18);
  }
  ctx.textAlign = "left";
}

export function marker(ctx, sym, sx, sy, now) {
  const bounce = Math.sin(now / 200) * 3;
  const colors = { "!": "#ffd60a", "?": "#4cc9f0", "+": "#adb5bd" };
  ctx.font = "bold 22px system-ui, sans-serif";
  ctx.textAlign = "center";
  ctx.lineWidth = 4;
  ctx.strokeStyle = "#000";
  ctx.strokeText(sym, sx, sy - 6 + bounce);
  ctx.fillStyle = colors[sym] || "#fff";
  ctx.fillText(sym, sx, sy - 6 + bounce);
  ctx.textAlign = "left";
}

export function bubble(ctx, text, sx, sy, maxW = 260, kind = "say") {
  ctx.font = "13px system-ui, sans-serif";
  const words = text.split(/\s+/);
  const lines = [];
  let line = "";
  for (const w of words) {
    const t = line ? line + " " + w : w;
    if (ctx.measureText(t).width > maxW && line) { lines.push(line); line = w; } else line = t;
    if (lines.length >= 5) break;
  }
  if (line && lines.length < 6) lines.push(line);
  if (lines.length >= 6) lines[5] = lines[5].slice(0, 30) + "…";
  const w = Math.min(maxW, Math.max(...lines.map((l) => ctx.measureText(l).width))) + 16;
  const h = lines.length * 16 + 10;
  const x = sx - w / 2, y = sy - h - 12;
  ctx.fillStyle = kind === "action" ? "rgba(30,40,70,0.92)" : "rgba(255,255,255,0.96)";
  ctx.strokeStyle = "rgba(0,0,0,0.6)";
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.roundRect(x, y, w, h, 8);
  ctx.fill();
  ctx.stroke();
  ctx.beginPath();
  ctx.moveTo(sx - 6, y + h); ctx.lineTo(sx, y + h + 9); ctx.lineTo(sx + 6, y + h);
  ctx.fill();
  ctx.fillStyle = kind === "action" ? "#e0e7ff" : "#111";
  if (kind === "action") ctx.font = "italic 13px system-ui, sans-serif";
  lines.forEach((l, i) => ctx.fillText(l, x + 8, y + 19 + i * 16));
}

// Night overlay with lights cut out around characters and lanterns.
export function drawLighting(ctx, W, H, clock, lights) {
  let dark = 0;
  if (clock >= 19 || clock < 5) dark = 0.62;
  if (clock >= 17 && clock < 19) dark = ((clock - 17) / 2) * 0.62;
  if (clock >= 5 && clock < 7) dark = (1 - (clock - 5) / 2) * 0.62;
  const dusk = clock >= 16.5 && clock < 19 ? Math.sin(((clock - 16.5) / 2.5) * Math.PI) * 0.18 : 0;
  if (dusk > 0) {
    ctx.fillStyle = `rgba(255,120,40,${dusk})`;
    ctx.fillRect(0, 0, W, H);
  }
  if (dark <= 0.01) return;
  if (!drawLighting.cv || drawLighting.cv.width !== W || drawLighting.cv.height !== H) {
    [drawLighting.cv, drawLighting.ctx] = makeCanvas(W, H);
  }
  const lc = drawLighting.ctx;
  lc.globalCompositeOperation = "source-over";
  lc.clearRect(0, 0, W, H);
  lc.fillStyle = `rgba(8,10,40,${dark})`;
  lc.fillRect(0, 0, W, H);
  lc.globalCompositeOperation = "destination-out";
  for (const l of lights) {
    const g = lc.createRadialGradient(l.x, l.y, 0, l.x, l.y, l.r);
    g.addColorStop(0, "rgba(0,0,0,0.95)");
    g.addColorStop(1, "rgba(0,0,0,0)");
    lc.fillStyle = g;
    lc.beginPath(); lc.arc(l.x, l.y, l.r, 0, Math.PI * 2); lc.fill();
  }
  ctx.drawImage(drawLighting.cv, 0, 0);
  // warm glow
  ctx.globalCompositeOperation = "lighter";
  for (const l of lights) {
    if (!l.warm) continue;
    const g = ctx.createRadialGradient(l.x, l.y, 0, l.x, l.y, l.r * 0.6);
    g.addColorStop(0, "rgba(255,180,90,0.18)");
    g.addColorStop(1, "rgba(255,180,90,0)");
    ctx.fillStyle = g;
    ctx.beginPath(); ctx.arc(l.x, l.y, l.r * 0.6, 0, Math.PI * 2); ctx.fill();
  }
  ctx.globalCompositeOperation = "source-over";
}
