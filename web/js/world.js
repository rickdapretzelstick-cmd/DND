// Overworld rendering.
import { renderChunk } from "./terrain.js";
import { TILE, T, WALKABLE, MINIMAP_COLORS, makeCanvas, drawTile, drawBuilding, charSprite, curseSprite, hash } from "./pixel.js";

const CHUNK = 32;

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
  }

  at(x, y) {
    if (x < 0 || y < 0 || x >= this.w.w || y >= this.w.h) return T.DEEP;
    return this.tiles[y * this.w.w + x];
  }

  walkable(x, y) { return WALKABLE.has(this.at(Math.floor(x), Math.floor(y))); }

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
      drawBuilding(bctx, b, (b.x - cx * CHUNK) * TILE, (b.y - cy * CHUNK) * TILE);
    }
    cv.ms = performance.now() - t0;
    this.chunks.set(k, cv);
    if (this.chunks.size > 30) this.chunks.delete(this.chunks.keys().next().value);
    return cv;
  }

  buildMinimap() {
    const [cv, ctx] = makeCanvas(this.w.w, this.w.h);
    const img = ctx.createImageData(this.w.w, this.w.h);
    for (let i = 0; i < this.tiles.length; i++) {
      const c = MINIMAP_COLORS[this.tiles[i]] || "#000";
      const n = parseInt(c.slice(1), 16);
      img.data[i * 4] = (n >> 16) & 255;
      img.data[i * 4 + 1] = (n >> 8) & 255;
      img.data[i * 4 + 2] = n & 255;
      img.data[i * 4 + 3] = 255;
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
    // Entities, y-sorted
    ents.sort((a, b) => a.y - b.y);
    for (const e of ents) {
      const [sx, sy] = toScreen(e.x, e.y);
      if (sx < -80 || sy < -80 || sx > W + 80 || sy > H + 120) continue;
      e.draw(ctx, sx, sy, zoom, now);
    }
    return { left, top, tpx, toScreen };
  }
}

// ---------- entity draw helpers ----------

export function drawCharacter(ctx, sx, sy, zoom, look, dir, moving, now, opts = {}) {
  const frame = moving ? Math.floor(now / 140) % 4 : 0;
  const spr = charSprite(look, dir, frame);
  const w = 16 * zoom, h = 24 * zoom;
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
