// Turn-based battle scene: backgrounds, actors, and technique VFX.
import { charSprite, curseSprite, hash, shade } from "./pixel.js";

const VFX = {
  beam: ["blood_beam", "love_beam", "beam", "amber", "sonic", "spear"],
  orb: ["red", "purple", "bullet", "nails", "embers", "balls", "spit", "fire_arrow", "talisman", "copy", "explode"],
  pull: ["blue"],
  rain: ["meteor", "blades", "collapse"],
  ground: ["eruption", "web", "roots", "flood", "isomer", "fish", "doors", "puppets"],
  slash: ["slash", "cleave", "scissors", "claw", "bite", "tool", "dogs", "ratio"],
  ring: ["voice", "voice_blast", "voice_crush", "wail", "stare", "clap"],
  aura: ["aura_red", "aura_gold", "aura_blue", "aura_white", "vow", "frames", "infinity", "simple"],
  heal: ["rct", "heal"],
  shield: ["shield", "wall"],
  melee: ["punch", "fire_punch", "rough", "fever", "soul", "freeze_frame", "hands", "slam", "rush", "tongue", "lash", "resonance", "blackflash"],
  bolt: ["lightning", "chain"],
  summon: ["rika", "summon"],
};
const VFX_KIND = {};
for (const [k, list] of Object.entries(VFX)) for (const v of list) VFX_KIND[v] = k;

const DOMAINS = {
  domain_void: { sky: ["#000008", "#0b1a3a"], floor: "#dbeafe", motif: "stars" },
  domain_shrine: { sky: ["#1a0000", "#7f0000"], floor: "#4a0000", motif: "shrine" },
  domain_shadow: { sky: ["#000000", "#1b1030"], floor: "#05040a", motif: "ripples" },
  domain_volcano: { sky: ["#2b0000", "#9d0208"], floor: "#3a0a00", motif: "volcano" },
  domain_pachinko: { sky: ["#012a1f", "#06d6a0"], floor: "#f8f9fa", motif: "lights" },
  domain_love: { sky: ["#000000", "#2b0a16"], floor: "#d6d0c4", motif: "crosses" },
  domain_hands: { sky: ["#001233", "#023e8a"], floor: "#03045e", motif: "hands" },
  domain_moon: { sky: ["#03071e", "#1e6091"], floor: "#0d1b2a", motif: "moon" },
  domain_dolls: { sky: ["#1a0505", "#9b2226"], floor: "#2b0a0a", motif: "dolls" },
  domain_blood: { sky: ["#1a0000", "#6a040f"], floor: "#9d0208", motif: "pond" },
  domain_office: { sky: ["#2b2b2b", "#6c584c"], floor: "#3a3a3a", motif: "clocks" },
  domain_stadium: { sky: ["#0b132b", "#22577a"], floor: "#2d6a4f", motif: "crowd" },
  domain_foundry: { sky: ["#0d1b2a", "#415a77"], floor: "#1b263b", motif: "gears" },
  domain_storm: { sky: ["#10002b", "#3a0ca3"], floor: "#240046", motif: "storm" },
  domain_voice: { sky: ["#10002b", "#3c096c"], floor: "#240046", motif: "rings" },
  domain_grave: { sky: ["#111111", "#495057"], floor: "#2b2b2b", motif: "graves" },
  domain_flowers: { sky: ["#081c15", "#2d6a4f"], floor: "#1b4332", motif: "flowers" },
  domain_sea: { sky: ["#03045e", "#90e0ef"], floor: "#e9d8a6", motif: "horizon" },
  domain_womb: { sky: ["#0b0012", "#3c096c"], floor: "#10002b", motif: "swirl" },
};

const BIOMES = {
  field: { sky: ["#7ec8ff", "#d8f3ff"], floor: "#5aa046", motif: "hills" },
  forest: { sky: ["#5b8c5a", "#b7e4c7"], floor: "#2f6b2f", motif: "trees" },
  grove: { sky: ["#1b4332", "#52796f"], floor: "#1e3d2a", motif: "trees" },
  city: { sky: ["#4a6fa5", "#c0d6f2"], floor: "#46464b", motif: "city" },
  apartment: { sky: ["#4a6fa5", "#c0d6f2"], floor: "#46464b", motif: "city" },
  school: { sky: ["#8ecae6", "#e0fbfc"], floor: "#c8c3b4", motif: "school" },
  detention: { sky: ["#6c757d", "#ced4da"], floor: "#9a9a8a", motif: "school" },
  hospital: { sky: ["#adb5bd", "#e9ecef"], floor: "#dfe7e7", motif: "hospital" },
  tunnel: { sky: ["#0b0b0f", "#2b2b30"], floor: "#3d342a", motif: "tunnel" },
  shrine: { sky: ["#3d0c0c", "#c08497"], floor: "#beb4aa", motif: "torii" },
  cemetery: { sky: ["#2b2d42", "#8d99ae"], floor: "#4a5a3a", motif: "graves" },
  grave: { sky: ["#2b2d42", "#8d99ae"], floor: "#4a5a3a", motif: "graves" },
  factory: { sky: ["#6c584c", "#a5a58d"], floor: "#5e5953", motif: "factory" },
  lake: { sky: ["#48cae4", "#caf0f8"], floor: "#dccd96", motif: "water" },
  shore: { sky: ["#48cae4", "#caf0f8"], floor: "#dccd96", motif: "water" },
  mountain: { sky: ["#8ecae6", "#edf6f9"], floor: "#7a756e", motif: "mountains" },
  cursed: { sky: ["#240046", "#7b2cbf"], floor: "#3c2a4d", motif: "fog" },
};

export class BattleView {
  constructor() {
    this.fxQueue = [];
    this.lastFx = 0;
    this.playing = null;
    this.playUntil = 0;
    this.anim = {}; // per actor: {lunge, hit, flash, shake}
    this.particles = [];
    this.beams = [];
    this.texts = [];
    this.flash = null;
    this.banner = null;
    this.shake = 0;
    this.layout = {};
    this.combatKey = "";
    this.fade = 0;
  }

  // Called with each new state snapshot.
  update(combat, chars) {
    this.combat = combat;
    this.chars = chars;
    if (!combat) return;
    const key = combat.title + combat.order.map((o) => o.id).join();
    if (key !== this.combatKey) {
      this.combatKey = key;
      this.lastFx = Math.max(0, ...combat.fx.map((f) => f.id)) - 0;
      this.fxQueue = [];
      this.fade = performance.now();
      this.banner = { text: combat.title || "Battle", sub: "Roll for initiative", color: "#fff", t0: performance.now(), dur: 1800 };
      // A fresh battle replays its opening (curses that act first); joining late skips history.
      if (combat.round <= 1) this.lastFx = 0;
    }
    for (const f of combat.fx) {
      if (f.id > this.lastFx) {
        this.fxQueue.push(f);
        this.lastFx = f.id;
      }
    }
  }

  busy() { return this.fxQueue.length > 0 || performance.now() < this.playUntil; }

  actorPos(id, W, H) {
    return this.layout[id];
  }

  computeLayout(W, H) {
    const cb = this.combat;
    const party = cb.order.filter((o) => !o.isMonster);
    const mons = cb.monsters;
    const L = {};
    const pScale = Math.max(4, Math.floor(H / 135));
    party.forEach((o, i) => {
      const n = party.length;
      const y = H * (n > 1 ? 0.47 + (i / (n - 1)) * 0.12 : 0.53);
      const x = W * 0.25 + (i % 2) * W * 0.05;
      L[o.id] = { x, y, scale: pScale, party: true };
    });
    mons.forEach((m, i) => {
      const n = mons.length;
      const y = H * (n > 1 ? 0.46 + (i / (n - 1)) * 0.13 : 0.53);
      const x = n > 4 ? W * (0.60 + (i % 2) * 0.16 + Math.floor(i / 2) * 0.04) : W * (n > 1 ? 0.58 + (i / (n - 1)) * 0.2 : 0.7);
      const size = m.sprite.size || 1;
      const scale = Math.max(3, Math.floor((H * (0.14 + size * 0.06)) / (16 + size * 12)));
      L[m.id] = { x, y, scale, party: false };
    });
    this.layout = L;
    return L;
  }

  hitTest(mx, my) {
    for (const [id, p] of Object.entries(this.layout)) {
      const r = p.party ? 12 * p.scale : 14 * p.scale;
      if (Math.abs(mx - p.x) < r && my < p.y + 10 && my > p.y - (p.party ? 38 : 30) * p.scale) return id;
    }
    return null;
  }

  // ---------- FX playback ----------

  step(now, W, H) {
    if (now < this.playUntil || this.fxQueue.length === 0) return;
    const f = this.fxQueue.shift();
    this.play(f, now, W, H);
  }

  play(f, now, W, H) {
    const L = this.layout;
    const src = L[f.src];
    const dur = f.kind === "domain" ? 1900 : f.kind === "blackflash" ? 1300 : f.kind === "status" ? 450 : f.kind === "text" ? 700 : 850;
    this.playUntil = now + dur;
    if (f.kind === "text") {
      this.banner = { text: f.text, sub: "", color: f.color || "#fff", t0: now, dur: 1100, small: true };
      return;
    }
    if (f.kind === "domain") {
      this.banner = { text: "DOMAIN EXPANSION", sub: (f.name || "").replace(/^Domain Expansion:\s*/, ""), color: f.color || "#fff", t0: now, dur: 2200 };
      this.flash = { color: "#ffffff", t0: now, dur: 600 };
      this.shake = 10;
      for (let i = 0; i < 80; i++) this.particles.push(part(W / 2, H / 2, (Math.random() - 0.5) * 900, (Math.random() - 0.5) * 900, f.color || "#fff", 900, 4));
      return;
    }
    if (f.kind === "domain_end") {
      this.flash = { color: "#ffffff", t0: now, dur: 300 };
      for (let i = 0; i < 60; i++) this.particles.push(part(Math.random() * W, Math.random() * H * 0.6, (Math.random() - 0.5) * 300, 300 + Math.random() * 400, "#e9ecef", 900, 5, "shard"));
      this.banner = { text: "Domain collapses", sub: "", color: "#adb5bd", t0: now, dur: 900, small: true };
      return;
    }
    const kind = VFX_KIND[f.vfx] || "orb";
    const color = f.color || "#fff";
    if (src && (kind === "melee" || kind === "slash") && f.hits?.some((h) => L[h.id] && !L[h.id].party !== !src.party)) {
      this.anim[f.src] = { ...(this.anim[f.src] || {}), lunge: now };
    }
    if (src) {
      if (kind === "aura") this.aura(src, color, now);
      if (kind === "ring") this.rings(src, color, now);
    }
    const impactDelay = kind === "orb" || kind === "pull" ? 320 : kind === "rain" ? 380 : 180;
    for (const h of f.hits || []) {
      const t = L[h.id];
      if (!t) continue;
      this.vfxAt(kind, f.vfx, src, t, color, now, h);
      setTimeout(() => this.impact(h, t, color, f.kind === "blackflash" && h.bf), impactDelay);
    }
    if (f.kind === "blackflash") {
      setTimeout(() => {
        this.flash = { color: "#000000", t0: performance.now(), dur: 380, invert: true };
        this.banner = { text: "BLACK FLASH", sub: "", color: "#ff1f3d", t0: performance.now(), dur: 1300, bf: true };
        this.shake = 22;
      }, impactDelay - 40);
    }
    if (f.vfx === "blackflash" && f.kind === "move" && /overdrive/i.test(f.name || "")) {
      this.banner = { text: "CURSED OVERDRIVE", sub: "", color: "#ffd166", t0: now, dur: 1400, bf: true };
      this.flash = { color: "#ff2e63", t0: now, dur: 500 };
      this.shake = 20;
    }
    if (f.name && f.kind !== "status") this.caption = { text: f.name, t0: now, color, who: f.src };
  }

  aura(p, color, now) {
    for (let i = 0; i < 30; i++) {
      const a = Math.random() * Math.PI * 2;
      this.particles.push(part(p.x + Math.cos(a) * 14 * p.scale * 0.5, p.y - Math.random() * 24 * p.scale, 0, -60 - Math.random() * 90, color, 700, 3));
    }
  }

  rings(p, color, now) {
    for (let i = 0; i < 3; i++) this.beams.push({ type: "ring", x: p.x, y: p.y - 12 * p.scale, color, t0: now + i * 120, dur: 600, r: 30 * p.scale });
  }

  vfxAt(kind, vfx, s, t, color, now, h) {
    const tx = t.x, ty = t.y - (t.party ? 18 : 14) * t.scale;
    const sx = s ? s.x : tx, sy = s ? s.y - 12 * s.scale : ty - 200;
    switch (kind) {
      case "beam":
        this.beams.push({ type: "beam", x1: sx, y1: sy, x2: tx, y2: ty, color, t0: now, dur: 420, w: vfx === "love_beam" || vfx === "amber" ? 26 : 12 });
        break;
      case "orb":
        this.beams.push({ type: "orb", x1: sx, y1: sy, x2: tx, y2: ty, color, t0: now, dur: 320, r: vfx === "purple" ? 34 : vfx === "red" ? 16 : 7, n: vfx === "nails" || vfx === "embers" || vfx === "balls" ? 3 : 1 });
        break;
      case "pull":
        this.beams.push({ type: "orb", x1: tx, y1: ty - 30, x2: tx, y2: ty - 30, color, t0: now, dur: 500, r: 18, n: 1 });
        for (let i = 0; i < 26; i++) {
          const a = Math.random() * 6.28, d = 120 + Math.random() * 80;
          this.particles.push(part(tx + Math.cos(a) * d, ty + Math.sin(a) * d, -Math.cos(a) * d * 2, -Math.sin(a) * d * 2, color, 450, 3));
        }
        break;
      case "rain":
        for (let i = 0; i < 6; i++) this.beams.push({ type: "orb", x1: tx + (Math.random() - 0.5) * 120, y1: -40, x2: tx + (Math.random() - 0.5) * 40, y2: ty, color, t0: now + i * 40, dur: 340, r: vfx === "meteor" ? 26 : 6, n: 1 });
        break;
      case "ground":
        for (let i = 0; i < 4; i++) this.beams.push({ type: "pillar", x: tx + (i - 1.5) * 16, y: t.y, color, t0: now + i * 50, dur: 520, h: 40 + Math.random() * 60 });
        break;
      case "slash":
        for (let i = 0; i < (vfx === "cleave" ? 2 : 3); i++) this.beams.push({ type: "slash", x: tx, y: ty, color: vfx === "cleave" ? "#ff4d6d" : "#ffffff", t0: now + 90 + i * 70, dur: 260, a: (vfx === "cleave" ? (i ? -0.8 : 0.8) : -0.6 + i * 0.5), len: 60 + t.scale * 6 });
        break;
      case "bolt":
        this.beams.push({ type: "bolt", x1: tx + (Math.random() - 0.5) * 40, y1: -20, x2: tx, y2: ty, color, t0: now + 100, dur: 300 });
        if (s) this.beams.push({ type: "bolt", x1: sx, y1: sy, x2: tx, y2: ty, color, t0: now, dur: 300 });
        break;
      case "heal":
        for (let i = 0; i < 18; i++) this.particles.push(part(tx + (Math.random() - 0.5) * 50, ty + 30, 0, -80 - Math.random() * 60, "#95d5b2", 800, 4, "plus"));
        break;
      case "shield":
        this.beams.push({ type: "bubble", x: tx, y: ty, color, t0: now, dur: 700, r: 22 * t.scale * 0.6 });
        break;
      case "summon":
        this.beams.push({ type: "hand", x: tx, y: ty, color, t0: now, dur: 520, r: 30 * t.scale * 0.5 });
        break;
      case "ring":
        this.beams.push({ type: "ring", x: tx, y: ty, color, t0: now + 120, dur: 500, r: 24 * t.scale * 0.5 });
        break;
      case "aura":
        if (t !== s) this.aura(t, color, now);
        break;
      default:
        break;
    }
  }

  impact(h, t, color, bf) {
    const now = performance.now();
    const ty = t.y - (t.party ? 18 : 14) * t.scale;
    const a = (this.anim[h.id] = { ...(this.anim[h.id] || {}) });
    if (h.miss) {
      this.texts.push(txt(t.x, ty - 20, h.status === "nullified" ? "NULLIFIED" : h.saved ? "RESISTED" : "MISS", "#ced4da", 18, now));
      a.dodge = now;
      return;
    }
    if (h.dmg) {
      a.hit = now;
      this.punch = { t0: now, x: t.x, y: ty, k: bf ? 0.16 : h.crit ? 0.1 : 0.05 };
      this.beams.push({ type: "burst", x: t.x, y: ty, color: bf ? "#ff1f3d" : "#ffffff", t0: now, dur: 260, r: bf ? 220 : 90 + h.dmg });
      this.shake = Math.max(this.shake, Math.min(14, 3 + h.dmg / 6));
      const size = bf ? 46 : h.crit ? 34 : 24 + Math.min(14, h.dmg / 5);
      this.texts.push(txt(t.x + (Math.random() - 0.5) * 30, ty - 30, String(h.dmg), bf ? "#ff1f3d" : h.crit ? "#ffd166" : "#ffffff", size, now, bf));
      for (let i = 0; i < (bf ? 50 : 14); i++) this.particles.push(part(t.x, ty, (Math.random() - 0.5) * 500, (Math.random() - 0.7) * 500, bf ? (i % 2 ? "#000" : "#ff1f3d") : color, 500, bf ? 5 : 3));
      if (bf) for (let i = 0; i < 5; i++) this.beams.push({ type: "bolt", x1: t.x, y1: ty, x2: t.x + (Math.random() - 0.5) * 300, y2: ty + (Math.random() - 0.5) * 240, color: i % 2 ? "#000000" : "#ff1f3d", t0: now, dur: 500 });
    }
    if (h.heal) this.texts.push(txt(t.x, ty - 34, "+" + h.heal, "#80ed99", 26, now));
    const words = { stun: "STUNNED", stunned: "STUNNED", exposed: "EXPOSED", weakened: "WEAKENED", marked: "MARKED", burn: "BURNING", bleed: "BLEEDING", poison: "POISONED", infinity: "INFINITY", guard: "GUARD", empowered: "EMPOWERED", haste: "HASTE", zone: "IN THE ZONE", regen: "REGEN", simple_domain: "SIMPLE DOMAIN", jackpot: "JACKPOT!", riichi: "RIICHI...", dispelled: "NULLIFIED" };
    if (h.status && words[h.status]) this.texts.push(txt(t.x, ty - 62, words[h.status], h.status === "jackpot" ? "#06d6a0" : "#c77dff", 16, now + 120));
    if (h.down) a.down = now;
  }

  // ---------- drawing ----------

  draw(ctx, W, H, now, opts) {
    const cb = this.combat;
    if (!cb) return;
    this.computeLayout(W, H);
    this.step(now, W, H);
    ctx.save();
    if (this.punch) {
      const pt = (now - this.punch.t0) / 420;
      if (pt >= 1) this.punch = null;
      else {
        const z = 1 + this.punch.k * Math.sin(pt * Math.PI);
        ctx.translate(this.punch.x, this.punch.y); ctx.scale(z, z); ctx.translate(-this.punch.x, -this.punch.y);
      }
    }
    if (this.shake > 0.2) {
      ctx.translate((Math.random() - 0.5) * this.shake, (Math.random() - 0.5) * this.shake);
      this.shake *= 0.88;
    }
    if (cb.domain) drawBackground(ctx, W, H, DOMAINS[cb.domain.vfx] || DOMAINS.domain_void, now, true);
    else drawBackground(ctx, W, H, BIOMES[cb.biome] || BIOMES.field, now, false, opts.clock);
    finishBackground(ctx, W, H, cb.domain ? DOMAINS[cb.domain.vfx] || DOMAINS.domain_void : BIOMES[cb.biome] || BIOMES.field, now, !!cb.domain, opts.clock);

    // Actors
    const L = this.layout;
    const cur = cb.order[cb.turn % cb.order.length];
    const entries = Object.entries(L).sort((a, b) => a[1].y - b[1].y);
    for (const [id, p] of entries) {
      const a = this.anim[id] || {};
      let dx = 0, dy = 0;
      if (a.lunge && now - a.lunge < 380) dx = Math.sin(((now - a.lunge) / 380) * Math.PI) * (p.party ? 70 : -70);
      if (a.hit && now - a.hit < 300) dx += (Math.random() - 0.5) * 10;
      if (a.dodge && now - a.dodge < 300) dy -= Math.sin(((now - a.dodge) / 300) * Math.PI) * 18;
      const bob = Math.sin(now / 400 + p.x) * 2;
      const isCur = cur && cur.id === id;
      // ground shadow and turn ring
      ctx.fillStyle = "rgba(0,0,0,0.35)";
      ctx.beginPath(); ctx.ellipse(p.x + dx, p.y, 12 * p.scale * (p.party ? 0.6 : 0.8), 3 * p.scale, 0, 0, Math.PI * 2); ctx.fill();
      if (isCur) {
        ctx.strokeStyle = "#ffd166"; ctx.lineWidth = 3;
        ctx.beginPath(); ctx.ellipse(p.x, p.y, 14 * p.scale * (p.party ? 0.6 : 0.8), 4 * p.scale, 0, 0, Math.PI * 2); ctx.stroke();
      }
      if (opts.target === id) {
        ctx.strokeStyle = "#ff4d6d"; ctx.lineWidth = 3;
        ctx.setLineDash([6, 4]);
        ctx.beginPath(); ctx.ellipse(p.x, p.y, 16 * p.scale * (p.party ? 0.6 : 0.8), 5 * p.scale, 0, 0, Math.PI * 2); ctx.stroke();
        ctx.setLineDash([]);
      }
      let spr, w, h, alpha = 1, hp = 0, max = 1, name = "";
      if (p.party) {
        const ch = this.chars.find((c) => c.id === id);
        if (!ch) continue;
        hp = ch.hp; max = ch.maxHp; name = ch.name;
        const down = ch.hp <= 0 || ch.dead;
        spr = charSprite(ch.look, 2, 0);
        w = 24 * p.scale; h = 36 * p.scale;
        if (ch.fled) alpha = 0.25;
        ctx.save();
        ctx.globalAlpha = alpha;
        if (down) {
          ctx.translate(p.x + dx, p.y - 4);
          ctx.rotate(Math.PI / 2);
          ctx.drawImage(spr, -h / 2, -w / 2, w, h);
        } else {
          ctx.drawImage(spr, Math.round(p.x - w / 2 + dx), Math.round(p.y - h + bob + dy), w, h);
        }
        ctx.restore();
        this.statusIcons(ctx, ch.status, p.x, p.y + 8);
      } else {
        const m = cb.monsters.find((x) => x.id === id);
        if (!m) continue;
        hp = m.hp; max = m.maxHp; name = m.name;
        spr = curseSprite(m.sprite, m.template, Math.floor(now / 350) % 2);
        w = spr.width * p.scale; h = spr.height * p.scale;
        if (m.hp <= 0) {
          const t = a.down ? Math.min(1, (now - a.down) / 900) : 1;
          alpha = 1 - t;
          if (alpha > 0 && Math.random() < 0.5) this.particles.push(part(p.x + (Math.random() - 0.5) * w * 0.6, p.y - Math.random() * h, 0, -60, "#2b2b2b", 600, 5));
        }
        if (alpha > 0) {
          ctx.save();
          ctx.globalAlpha = alpha;
          ctx.translate(Math.round(p.x + dx), 0);
          ctx.scale(-1, 1);
          ctx.drawImage(spr, Math.round(-w / 2), Math.round(p.y - h + bob + dy), w, h);
          if (a.hit && now - a.hit < 140) {
            ctx.globalCompositeOperation = "lighter";
            ctx.globalAlpha = 0.7;
            ctx.drawImage(spr, Math.round(-w / 2), Math.round(p.y - h + bob + dy), w, h);
          }
          ctx.restore();
        }
        this.statusIcons(ctx, m.status, p.x, p.y + 8);
        if (m.hp <= 0) continue;
      }
      // HP bar + name (+ telegraphed move)
      const bw = Math.max(96, w * 0.9);
      const by = p.y - h - 22;
      const nm = name.length > 22 ? name.slice(0, 21) + "…" : name;
      ctx.font = "800 12px system-ui, sans-serif";
      const nw = ctx.measureText(nm).width;
      const bx = p.x - bw / 2;
      ctx.fillStyle = "rgba(8,6,16,0.82)";
      ctx.beginPath(); ctx.moveTo(bx - 4, by - 17); ctx.lineTo(bx + Math.max(bw, nw + 18) + 4, by - 17); ctx.lineTo(bx + Math.max(bw, nw + 18) - 4, by + 11); ctx.lineTo(bx - 4, by + 11); ctx.closePath(); ctx.fill();
      ctx.fillStyle = isCur ? "#ffd166" : p.party ? "#7dd3fc" : "#fb7185";
      ctx.fillRect(bx - 4, by - 17, 3, 28);
      ctx.fillStyle = "#2a0a10";
      ctx.beginPath(); ctx.moveTo(bx + 3, by + 1); ctx.lineTo(bx + bw, by + 1); ctx.lineTo(bx + bw - 4, by + 8); ctx.lineTo(bx, by + 8); ctx.closePath(); ctx.fill();
      const frac = Math.max(0, hp / max);
      ctx.fillStyle = frac > 0.5 ? "#52b788" : frac > 0.25 ? "#e9c46a" : "#e63946";
      ctx.beginPath(); ctx.moveTo(bx + 3, by + 1); ctx.lineTo(bx + 3 + (bw - 3) * frac, by + 1); ctx.lineTo(bx + (bw - 4) * frac, by + 8); ctx.lineTo(bx, by + 8); ctx.closePath(); ctx.fill();
      ctx.fillStyle = isCur ? "#ffd166" : "#fff";
      ctx.fillText(nm, bx + 3, by - 5);
      ctx.font = "700 9px system-ui, sans-serif";
      ctx.fillStyle = "rgba(255,255,255,0.8)";
      ctx.fillText(`${Math.max(0, hp)}/${max}`, bx + bw - 3 - ctx.measureText(`${Math.max(0, hp)}/${max}`).width, by - 6);
      if (!p.party) {
        const m = cb.monsters.find((x) => x.id === id);
        if (m && m.next && m.hp > 0) {
          const pulse = 0.65 + 0.35 * Math.sin(now / 180);
          const label = "⚠ " + m.next + (m.nextAoe ? " (ALL)" : "");
          ctx.font = "800 11px system-ui, sans-serif";
          const lw = ctx.measureText(label).width + 14;
          ctx.fillStyle = `rgba(120,10,30,${0.75 * pulse + 0.2})`;
          ctx.fillRect(p.x - lw / 2, by - 40, lw, 18);
          ctx.strokeStyle = "#ff4d6d"; ctx.lineWidth = 1; ctx.strokeRect(p.x - lw / 2 + 0.5, by - 39.5, lw - 1, 17);
          ctx.fillStyle = "#fff"; ctx.textAlign = "center"; ctx.fillText(label, p.x, by - 27); ctx.textAlign = "left";
        }
      }
    }

    // Effects
    this.drawEffects(ctx, now);
    ctx.restore();

    // Screen flash
    if (this.flash) {
      const t = (now - this.flash.t0) / this.flash.dur;
      if (t >= 1) this.flash = null;
      else {
        ctx.fillStyle = this.flash.invert ? `rgba(0,0,0,${0.85 * (1 - t)})` : hexA(this.flash.color, 0.8 * (1 - t));
        ctx.fillRect(0, 0, W, H);
        if (this.flash.invert && t < 0.5) { ctx.fillStyle = `rgba(255,20,40,${0.25 * (1 - t * 2)})`; ctx.fillRect(0, 0, W, H); }
      }
    }
    // Opening fade
    const ft = (now - this.fade) / 600;
    if (ft < 1) { ctx.fillStyle = `rgba(0,0,0,${1 - ft})`; ctx.fillRect(0, 0, W, H); }
    this.drawBanner(ctx, W, H, now);
    if (this.caption && now - this.caption.t0 < 1300) {
      const p = this.layout[this.caption.who];
      ctx.font = "bold 18px system-ui, sans-serif";
      const tw = ctx.measureText(this.caption.text).width;
      const x = W / 2 - tw / 2, y = H * 0.14;
      ctx.fillStyle = "rgba(0,0,0,0.6)";
      ctx.fillRect(x - 14, y - 22, tw + 28, 32);
      ctx.fillStyle = this.caption.color;
      ctx.fillRect(x - 14, y - 22, 4, 32);
      ctx.fillStyle = "#fff";
      ctx.fillText(this.caption.text, x, y);
      void p;
    }
  }

  statusIcons(ctx, st, x, y) {
    if (!st) return;
    const icons = { stun: "💫", exposed: "🎯", weakened: "🥀", marked: "📍", burn: "🔥", bleed: "🩸", poison: "☠️", infinity: "♾️", guard: "🛡️", empowered: "💪", haste: "⏩", zone: "⚡", regen: "💚", simple_domain: "🔵", jackpot: "🎰", burnout: "💤", skip: "💫" };
    const list = Object.keys(st).filter((k) => icons[k] && st[k] > 0);
    ctx.font = "14px sans-serif";
    ctx.textAlign = "center";
    list.forEach((k, i) => ctx.fillText(icons[k], x + (i - (list.length - 1) / 2) * 18, y + 14));
    ctx.textAlign = "left";
  }

  drawEffects(ctx, now) {
    const dt = 1 / 60;
    this.beams = this.beams.filter((b) => now - b.t0 < b.dur);
    for (const b of this.beams) {
      const t = (now - b.t0) / b.dur;
      if (t < 0) continue;
      ctx.save();
      switch (b.type) {
        case "beam": {
          const w = b.w * (t < 0.2 ? t / 0.2 : 1 - (t - 0.2) / 0.8 * 0.6);
          ctx.globalCompositeOperation = "lighter";
          ctx.strokeStyle = hexA(b.color, 0.5); ctx.lineWidth = w * 2.2; ctx.lineCap = "round";
          ctx.beginPath(); ctx.moveTo(b.x1, b.y1); ctx.lineTo(b.x2, b.y2); ctx.stroke();
          ctx.strokeStyle = b.color; ctx.lineWidth = w;
          ctx.beginPath(); ctx.moveTo(b.x1, b.y1); ctx.lineTo(b.x2, b.y2); ctx.stroke();
          ctx.strokeStyle = "#fff"; ctx.lineWidth = w * 0.35;
          ctx.beginPath(); ctx.moveTo(b.x1, b.y1); ctx.lineTo(b.x2, b.y2); ctx.stroke();
          break;
        }
        case "orb": {
          for (let i = 0; i < b.n; i++) {
            const tt = Math.min(1, t * 1.1 - i * 0.08);
            if (tt < 0) continue;
            const x = b.x1 + (b.x2 - b.x1) * tt, y = b.y1 + (b.y2 - b.y1) * tt - Math.sin(tt * Math.PI) * 30;
            ctx.globalCompositeOperation = "lighter";
            const g = ctx.createRadialGradient(x, y, 0, x, y, b.r * 2);
            g.addColorStop(0, "#ffffff"); g.addColorStop(0.35, b.color); g.addColorStop(1, "rgba(0,0,0,0)");
            ctx.fillStyle = g;
            ctx.beginPath(); ctx.arc(x, y, b.r * 2, 0, Math.PI * 2); ctx.fill();
          }
          break;
        }
        case "burst": {
          ctx.globalCompositeOperation = "lighter";
          ctx.strokeStyle = hexA(b.color, 0.9 * (1 - t));
          ctx.lineWidth = 3;
          for (let i = 0; i < 14; i++) {
            const a = (i / 14) * Math.PI * 2 + (i % 2) * 0.12, r0 = b.r * (0.15 + t * 0.55), r1 = b.r * (0.3 + t * 0.9);
            ctx.beginPath(); ctx.moveTo(b.x + Math.cos(a) * r0, b.y + Math.sin(a) * r0); ctx.lineTo(b.x + Math.cos(a) * r1, b.y + Math.sin(a) * r1); ctx.stroke();
          }
          ctx.fillStyle = hexA("#ffffff", 0.7 * (1 - t) * (1 - t));
          ctx.beginPath(); ctx.arc(b.x, b.y, b.r * 0.3 * (1 - t) + 4, 0, Math.PI * 2); ctx.fill();
          break;
        }
        case "pillar": {
          const h = b.h * Math.sin(Math.min(1, t * 1.5) * Math.PI);
          ctx.globalCompositeOperation = "lighter";
          ctx.fillStyle = hexA(b.color, 0.75);
          ctx.fillRect(b.x - 7, b.y - h, 14, h);
          ctx.fillStyle = "rgba(255,255,255,0.6)";
          ctx.fillRect(b.x - 2, b.y - h, 4, h);
          break;
        }
        case "slash": {
          const l = b.len * Math.min(1, t * 3);
          ctx.strokeStyle = b.color; ctx.lineWidth = 4 * (1 - t) + 1;
          ctx.shadowColor = b.color; ctx.shadowBlur = 10;
          ctx.beginPath();
          ctx.moveTo(b.x - Math.cos(b.a) * l / 2, b.y - Math.sin(b.a) * l / 2);
          ctx.lineTo(b.x + Math.cos(b.a) * l / 2, b.y + Math.sin(b.a) * l / 2);
          ctx.stroke();
          break;
        }
        case "bolt": {
          ctx.strokeStyle = b.color; ctx.lineWidth = 3; ctx.shadowColor = b.color; ctx.shadowBlur = 14;
          ctx.globalAlpha = 1 - t;
          ctx.beginPath(); ctx.moveTo(b.x1, b.y1);
          const n = 7;
          for (let i = 1; i <= n; i++) {
            const k = i / n;
            ctx.lineTo(b.x1 + (b.x2 - b.x1) * k + (i < n ? (Math.random() - 0.5) * 30 : 0), b.y1 + (b.y2 - b.y1) * k + (i < n ? (Math.random() - 0.5) * 20 : 0));
          }
          ctx.stroke();
          break;
        }
        case "ring": {
          ctx.strokeStyle = hexA(b.color, 1 - t); ctx.lineWidth = 4;
          ctx.beginPath(); ctx.arc(b.x, b.y, b.r * (0.3 + t * 1.5), 0, Math.PI * 2); ctx.stroke();
          break;
        }
        case "bubble": {
          ctx.fillStyle = hexA(b.color, 0.25 * (1 - t)); ctx.strokeStyle = hexA(b.color, 0.9 * (1 - t)); ctx.lineWidth = 3;
          ctx.beginPath(); ctx.arc(b.x, b.y, b.r, 0, Math.PI * 2); ctx.fill(); ctx.stroke();
          break;
        }
        case "hand": {
          const y = b.y - 120 + Math.min(1, t * 2.5) * 110;
          ctx.globalAlpha = 1 - Math.max(0, t - 0.6) * 2.5;
          ctx.fillStyle = shade(b.color, -0.2);
          ctx.beginPath(); ctx.ellipse(b.x, y, b.r * 1.2, b.r, 0, 0, Math.PI * 2); ctx.fill();
          for (let i = 0; i < 4; i++) ctx.fillRect(b.x - b.r + i * b.r * 0.6, y - b.r * 1.8, b.r * 0.4, b.r);
          break;
        }
      }
      ctx.restore();
    }
    // particles
    this.particles = this.particles.filter((p) => now - p.t0 < p.life);
    for (const p of this.particles) {
      const t = (now - p.t0) / p.life;
      p.x += p.vx * dt; p.y += p.vy * dt; p.vy += (p.kind === "shard" ? 600 : 80) * dt;
      ctx.globalAlpha = 1 - t;
      ctx.fillStyle = p.color;
      if (p.kind === "plus") { ctx.fillRect(p.x - 1, p.y - 4, 3, 9); ctx.fillRect(p.x - 4, p.y - 1, 9, 3); }
      else ctx.fillRect(p.x, p.y, p.size, p.size);
    }
    ctx.globalAlpha = 1;
    // texts
    this.texts = this.texts.filter((tx) => now - tx.t0 < 1300);
    for (const tx of this.texts) {
      const t = (now - tx.t0) / 1300;
      if (t < 0) continue;
      const y = tx.y - t * 50;
      ctx.font = `900 ${tx.size}px system-ui, sans-serif`;
      ctx.textAlign = "center";
      ctx.globalAlpha = t > 0.7 ? 1 - (t - 0.7) / 0.3 : 1;
      ctx.lineWidth = tx.bf ? 7 : 5;
      ctx.strokeStyle = tx.bf ? "#000" : "rgba(0,0,0,0.85)";
      ctx.strokeText(tx.text, tx.x, y);
      ctx.fillStyle = tx.color;
      ctx.fillText(tx.text, tx.x, y);
    }
    ctx.globalAlpha = 1;
    ctx.textAlign = "left";
  }

  drawBanner(ctx, W, H, now) {
    const b = this.banner;
    if (!b) return;
    const t = (now - b.t0) / b.dur;
    if (t >= 1) { this.banner = null; return; }
    const ease = t < 0.15 ? t / 0.15 : t > 0.8 ? 1 - (t - 0.8) / 0.2 : 1;
    ctx.save();
    ctx.globalAlpha = ease;
    if (b.small) {
      ctx.font = "900 26px system-ui, sans-serif";
      ctx.textAlign = "center";
      ctx.lineWidth = 6; ctx.strokeStyle = "#000";
      ctx.strokeText(b.text, W / 2, H * 0.24);
      ctx.fillStyle = b.color;
      ctx.fillText(b.text, W / 2, H * 0.24);
    } else {
      const bh = b.bf ? 120 : 110;
      const y = H * 0.42;
      ctx.fillStyle = b.bf ? "rgba(0,0,0,0.9)" : "rgba(0,0,0,0.75)";
      ctx.fillRect(0, y - bh / 2, W * Math.min(1, t * 6), bh);
      ctx.fillStyle = b.color;
      ctx.fillRect(0, y - bh / 2, W * Math.min(1, t * 6), 4);
      ctx.fillRect(0, y + bh / 2 - 4, W * Math.min(1, t * 6), 4);
      ctx.textAlign = "center";
      ctx.font = `900 ${b.bf ? 64 : 46}px system-ui, sans-serif`;
      ctx.lineWidth = 8; ctx.strokeStyle = "#000";
      const shakeX = b.bf ? (Math.random() - 0.5) * 6 : 0;
      ctx.strokeText(b.text, W / 2 + shakeX, y + (b.sub ? -2 : 18));
      ctx.fillStyle = b.bf ? "#ff1f3d" : "#fff";
      ctx.fillText(b.text, W / 2 + shakeX, y + (b.sub ? -2 : 18));
      if (b.sub) {
        ctx.font = "700 24px system-ui, sans-serif";
        ctx.fillStyle = b.color;
        ctx.fillText(b.sub, W / 2, y + 34);
      }
    }
    ctx.restore();
  }
}

function part(x, y, vx, vy, color, life, size, kind) {
  return { x, y, vx, vy, color, life, size, kind, t0: performance.now() };
}

function txt(x, y, text, color, size, t0, bf) {
  return { x, y, text, color, size, t0, bf };
}

function hexA(hex, a) {
  if (!hex || hex[0] !== "#") return `rgba(255,255,255,${a})`;
  const n = parseInt(hex.slice(1), 16);
  return `rgba(${(n >> 16) & 255},${(n >> 8) & 255},${n & 255},${Math.max(0, a)})`;
}

// ---------- backgrounds ----------

function drawBackground(ctx, W, H, bg, now, domain, clock = 12) {
  const horizon = H * 0.43;
  const g = ctx.createLinearGradient(0, 0, 0, horizon);
  let s0 = bg.sky[0], s1 = bg.sky[1];
  if (!domain && (clock >= 19 || clock < 5)) { s0 = "#03071e"; s1 = "#1d3557"; }
  g.addColorStop(0, s0); g.addColorStop(1, s1);
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, W, horizon);
  const fg = ctx.createLinearGradient(0, horizon, 0, H);
  fg.addColorStop(0, shade(bg.floor, 0.08)); fg.addColorStop(1, shade(bg.floor, -0.35));
  ctx.fillStyle = fg;
  ctx.fillRect(0, horizon, W, H - horizon);
  // dithered floor texture
  ctx.fillStyle = "rgba(0,0,0,0.08)";
  for (let y = horizon; y < H; y += 6) for (let x = (y / 6) % 2 ? 0 : 6; x < W; x += 12) ctx.fillRect(x, y, 6, 3);
  const m = bg.motif;
  const sil = (c) => { ctx.fillStyle = c; };
  const r = (i, s = 1) => hash(i, s, 99);
  switch (m) {
    case "hills":
      sil("#4f9a3f"); for (let i = 0; i < 5; i++) { ctx.beginPath(); ctx.ellipse(W * (i / 4), horizon, W * 0.2, H * 0.12, 0, Math.PI, 0); ctx.fill(); }
      clouds(ctx, W, H, now); break;
    case "trees":
      for (let i = 0; i < 14; i++) { const x = r(i) * W, h = H * (0.15 + r(i, 2) * 0.15); sil(i % 2 ? "#1b4332" : "#2d6a4f"); ctx.beginPath(); ctx.moveTo(x - 30, horizon); ctx.lineTo(x, horizon - h); ctx.lineTo(x + 30, horizon); ctx.fill(); }
      break;
    case "city":
      for (let i = 0; i < 12; i++) {
        const x = (i / 12) * W, w = W / 12 - 6, h = H * (0.15 + r(i) * 0.3);
        sil(i % 2 ? "#3d405b" : "#4a4e69"); ctx.fillRect(x, horizon - h, w, h);
        ctx.fillStyle = clock >= 18 || clock < 6 ? "#ffd166" : "rgba(200,230,255,0.6)";
        for (let y = horizon - h + 8; y < horizon - 8; y += 14) for (let xx = x + 5; xx < x + w - 6; xx += 12) if (r(xx, y) > 0.4) ctx.fillRect(xx, y, 5, 6);
      }
      break;
    case "school": case "hospital": {
      const c = m === "hospital" ? "#e9ecef" : "#e5d9c6";
      sil(c); ctx.fillRect(W * 0.15, horizon - H * 0.28, W * 0.7, H * 0.28);
      sil(shade(c, -0.3)); ctx.fillRect(W * 0.15, horizon - H * 0.3, W * 0.7, H * 0.03);
      ctx.fillStyle = "#33415c";
      for (let y = horizon - H * 0.25; y < horizon - 16; y += H * 0.08) for (let x = W * 0.18; x < W * 0.82; x += W * 0.06) ctx.fillRect(x, y, W * 0.035, H * 0.045);
      if (m === "school") { sil(c); ctx.fillRect(W * 0.46, horizon - H * 0.38, W * 0.08, H * 0.1); ctx.fillStyle = "#fff"; ctx.beginPath(); ctx.arc(W * 0.5, horizon - H * 0.33, H * 0.03, 0, 7); ctx.fill(); }
      else { ctx.fillStyle = "#d00000"; ctx.fillRect(W * 0.48, horizon - H * 0.36, W * 0.04, H * 0.012); ctx.fillRect(W * 0.494, horizon - H * 0.38, W * 0.012, H * 0.05); }
      break;
    }
    case "tunnel":
      sil("#2b2b30"); ctx.fillRect(0, 0, W, horizon);
      ctx.fillStyle = "#000"; ctx.beginPath(); ctx.arc(W / 2, horizon, H * 0.35, Math.PI, 0); ctx.fill();
      ctx.fillStyle = "#ffd166"; for (let i = 0; i < 6; i++) ctx.fillRect(W * (0.1 + i * 0.16), H * 0.1, 8, 4);
      break;
    case "torii":
      sil("#9d0208"); ctx.fillRect(W * 0.35, horizon - H * 0.32, W * 0.3, H * 0.035); ctx.fillRect(W * 0.33, horizon - H * 0.36, W * 0.34, H * 0.03);
      ctx.fillRect(W * 0.39, horizon - H * 0.32, W * 0.025, H * 0.32); ctx.fillRect(W * 0.585, horizon - H * 0.32, W * 0.025, H * 0.32);
      break;
    case "graves":
      for (let i = 0; i < 16; i++) { const x = r(i) * W, h = 30 + r(i, 3) * 30; sil(i % 3 ? "#6c757d" : "#495057"); ctx.fillRect(x, horizon - h, 22, h); ctx.fillRect(x + 4, horizon - h - 8, 14, 8); }
      fog(ctx, W, horizon, now, "rgba(200,200,220,0.12)");
      break;
    case "factory":
      sil("#4a4e69"); ctx.fillRect(W * 0.1, horizon - H * 0.2, W * 0.8, H * 0.2);
      for (let i = 0; i < 4; i++) { ctx.fillRect(W * (0.2 + i * 0.18), horizon - H * 0.38, 20, H * 0.2); }
      fog(ctx, W, horizon * 0.5, now, "rgba(120,120,120,0.2)");
      break;
    case "water":
      ctx.fillStyle = "#2a63ae"; ctx.fillRect(0, horizon - H * 0.12, W, H * 0.12);
      ctx.fillStyle = "rgba(255,255,255,0.4)"; for (let i = 0; i < 30; i++) ctx.fillRect((r(i) * W + now / 30) % W, horizon - H * 0.1 + r(i, 2) * H * 0.09, 18, 2);
      break;
    case "mountains":
      for (let i = 0; i < 6; i++) { sil(i % 2 ? "#6c757d" : "#8d99ae"); ctx.beginPath(); ctx.moveTo(W * (i / 5) - 140, horizon); ctx.lineTo(W * (i / 5), horizon - H * (0.25 + r(i) * 0.2)); ctx.lineTo(W * (i / 5) + 140, horizon); ctx.fill(); }
      break;
    case "fog":
      fog(ctx, W, horizon, now, "rgba(180,100,255,0.15)");
      break;
    // ----- domains -----
    case "stars":
      for (let i = 0; i < 260; i++) {
        const x = r(i) * W, y = r(i, 2) * H, tw = 0.5 + 0.5 * Math.sin(now / 200 + i);
        ctx.fillStyle = `rgba(255,255,255,${tw})`; ctx.fillRect(x, y, 2, 2);
      }
      ctx.strokeStyle = "rgba(180,220,255,0.25)";
      for (let i = 0; i < 12; i++) { ctx.beginPath(); ctx.arc(W / 2, H / 2, ((now / 8 + i * 70) % 900), 0, 7); ctx.stroke(); }
      break;
    case "shrine":
      sil("#1a0000");
      ctx.fillRect(W * 0.38, horizon - H * 0.34, W * 0.24, H * 0.34);
      ctx.beginPath(); ctx.moveTo(W * 0.3, horizon - H * 0.32); ctx.lineTo(W * 0.5, horizon - H * 0.48); ctx.lineTo(W * 0.7, horizon - H * 0.32); ctx.fill();
      ctx.fillStyle = "#9d0208"; ctx.fillRect(W * 0.4, horizon - H * 0.3, W * 0.015, H * 0.3); ctx.fillRect(W * 0.585, horizon - H * 0.3, W * 0.015, H * 0.3);
      ctx.fillStyle = "#fff"; for (let i = 0; i < 10; i++) ctx.fillRect(W * 0.44 + i * W * 0.012, horizon - H * 0.12, W * 0.008, H * 0.025);
      ctx.fillStyle = "rgba(160,0,0,0.5)"; ctx.beginPath(); ctx.ellipse(W / 2, H * 0.8, W * 0.45, H * 0.12, 0, 0, 7); ctx.fill();
      for (let i = 0; i < 10; i++) { ctx.strokeStyle = "rgba(255,255,255,0.7)"; ctx.lineWidth = 2; const x = (r(i) * W + now / 2 * (i % 3 + 1)) % W; ctx.beginPath(); ctx.moveTo(x, r(i, 4) * H); ctx.lineTo(x + 60, r(i, 4) * H - 40); ctx.stroke(); }
      break;
    case "ripples":
      ctx.strokeStyle = "rgba(120,80,200,0.3)"; for (let i = 0; i < 8; i++) { ctx.beginPath(); ctx.ellipse(W / 2, H * 0.75, ((now / 10 + i * 60) % 480) * 1.6, ((now / 10 + i * 60) % 480) * 0.3, 0, 0, 7); ctx.stroke(); }
      for (let i = 0; i < 10; i++) { sil("#0b0b12"); ctx.beginPath(); ctx.ellipse(r(i) * W, horizon - 20, 18, 60 + r(i) * 40, 0, 0, 7); ctx.fill(); }
      break;
    case "volcano":
      sil("#1a0000"); ctx.beginPath(); ctx.moveTo(W * 0.2, horizon); ctx.lineTo(W * 0.45, horizon - H * 0.4); ctx.lineTo(W * 0.55, horizon - H * 0.4); ctx.lineTo(W * 0.8, horizon); ctx.fill();
      ctx.fillStyle = "#ff5400"; ctx.fillRect(W * 0.46, horizon - H * 0.41, W * 0.08, 6);
      for (let i = 0; i < 40; i++) { ctx.fillStyle = "#ff9e00"; ctx.fillRect(r(i) * W, (H - (now / 4 + r(i, 2) * H) % H), 3, 3); }
      ctx.strokeStyle = "#ff5400"; ctx.lineWidth = 2; for (let i = 0; i < 8; i++) { ctx.beginPath(); ctx.moveTo(r(i) * W, H); ctx.lineTo(r(i) * W + 40, horizon + 30); ctx.stroke(); }
      break;
    case "lights":
      for (let i = 0; i < 60; i++) { const on = Math.floor(now / 120 + i) % 3 === 0; ctx.fillStyle = on ? ["#ffd60a", "#ef476f", "#06d6a0", "#118ab2"][i % 4] : "rgba(255,255,255,0.15)"; ctx.beginPath(); ctx.arc(r(i) * W, r(i, 2) * horizon, 6, 0, 7); ctx.fill(); }
      ctx.font = "900 60px monospace"; ctx.fillStyle = "rgba(239,71,111,0.5)"; ctx.textAlign = "center"; ctx.fillText(String(Math.floor(now / 90) % 1000).padStart(3, "7"), W / 2, H * 0.22); ctx.textAlign = "left";
      break;
    case "crosses":
      for (let i = 0; i < 18; i++) { const x = r(i) * W, h = 40 + r(i, 2) * 50; sil("#f8f9fa"); ctx.fillRect(x, horizon - h, 6, h); ctx.fillRect(x - 12, horizon - h + 12, 30, 6); }
      for (let i = 0; i < 12; i++) { sil("#adb5bd"); ctx.save(); ctx.translate(r(i, 5) * W, H * 0.75 + r(i, 6) * H * 0.15); ctx.rotate(0.2 - r(i) * 0.4); ctx.fillRect(-2, -50, 4, 50); ctx.fillRect(-8, -50, 16, 3); ctx.restore(); }
      ctx.strokeStyle = "rgba(255,112,166,0.3)"; ctx.lineWidth = 3; for (let i = 0; i < 5; i++) { ctx.beginPath(); ctx.arc(W / 2, H * 0.2, 80 + i * 40 + Math.sin(now / 500) * 10, 0, 7); ctx.stroke(); }
      break;
    case "hands":
      for (let i = 0; i < 40; i++) {
        const x = r(i) * W, y = r(i, 2) * H;
        ctx.fillStyle = "rgba(200,230,255,0.35)"; ctx.beginPath(); ctx.ellipse(x, y, 18, 22, 0, 0, 7); ctx.fill();
        for (let f = 0; f < 4; f++) ctx.fillRect(x - 14 + f * 8, y - 40, 6, 22);
      }
      break;
    case "moon":
      ctx.fillStyle = "#fefae0"; ctx.beginPath(); ctx.arc(W * 0.7, H * 0.2, H * 0.12, 0, 7); ctx.fill();
      ctx.strokeStyle = "rgba(255,255,255,0.5)"; ctx.lineWidth = 3;
      for (let i = 0; i < 6; i++) { const x = ((now / 3 + i * 220) % (W + 200)) - 200; ctx.strokeRect(x, H * 0.38, 150, 90); ctx.strokeRect(x + 10, H * 0.38 + 10, 130, 70); }
      break;
    case "dolls":
      for (let i = 0; i < 20; i++) { const x = r(i) * W, len = 30 + r(i, 2) * 120, sw = Math.sin(now / 600 + i) * 10; ctx.strokeStyle = "#ced4da"; ctx.lineWidth = 1; ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x + sw, len); ctx.stroke(); ctx.fillStyle = "#e9c46a"; ctx.fillRect(x + sw - 6, len, 12, 18); ctx.fillRect(x + sw - 10, len + 5, 20, 4); }
      break;
    case "pond":
      ctx.fillStyle = "rgba(208,0,0,0.6)"; ctx.beginPath(); ctx.ellipse(W / 2, H * 0.78, W * 0.48, H * 0.16, 0, 0, 7); ctx.fill();
      for (let i = 0; i < 12; i++) { ctx.fillStyle = "#ff8fab"; const x = r(i) * W, y = horizon + r(i, 2) * (H - horizon); ctx.beginPath(); ctx.ellipse(x, y, 16, 6, 0, 0, 7); ctx.fill(); }
      break;
    case "clocks":
      for (let i = 0; i < 9; i++) { const x = W * (0.1 + (i % 5) * 0.2), y = H * (0.12 + Math.floor(i / 5) * 0.2); ctx.strokeStyle = "#f8f9fa"; ctx.lineWidth = 3; ctx.beginPath(); ctx.arc(x, y, 30, 0, 7); ctx.stroke(); const a = now / (200 + i * 50); ctx.beginPath(); ctx.moveTo(x, y); ctx.lineTo(x + Math.cos(a) * 24, y + Math.sin(a) * 24); ctx.stroke(); }
      break;
    case "crowd":
      for (let i = 0; i < 80; i++) { ctx.fillStyle = ["#e63946", "#f1faee", "#a8dadc", "#457b9d", "#ffd166"][i % 5]; ctx.beginPath(); ctx.arc((i / 80) * W, horizon - 30 - (i % 3) * 18 + Math.sin(now / 150 + i) * 4, 8, 0, 7); ctx.fill(); }
      ctx.fillStyle = "rgba(255,255,255,0.8)"; for (let i = 0; i < 4; i++) { ctx.beginPath(); ctx.arc(W * (0.1 + i * 0.27), 20, 14, 0, 7); ctx.fill(); }
      break;
    case "gears":
      for (let i = 0; i < 8; i++) { const x = r(i) * W, y = r(i, 2) * horizon, rad = 20 + r(i, 3) * 40; ctx.save(); ctx.translate(x, y); ctx.rotate(now / 1000 * (i % 2 ? 1 : -1)); ctx.strokeStyle = "rgba(162,210,255,0.5)"; ctx.lineWidth = 6; ctx.beginPath(); ctx.arc(0, 0, rad, 0, 7); ctx.stroke(); for (let k = 0; k < 8; k++) { ctx.rotate(Math.PI / 4); ctx.fillStyle = "rgba(162,210,255,0.5)"; ctx.fillRect(rad - 2, -5, 12, 10); } ctx.restore(); }
      break;
    case "storm":
      clouds(ctx, W, H, now, "rgba(60,9,108,0.8)");
      if (Math.floor(now / 300) % 5 === 0) { ctx.strokeStyle = "#fff"; ctx.lineWidth = 3; ctx.beginPath(); let x = r(Math.floor(now / 300)) * W, y = 0; ctx.moveTo(x, y); while (y < horizon) { x += (Math.random() - 0.5) * 50; y += 30; ctx.lineTo(x, y); } ctx.stroke(); }
      break;
    case "rings":
      ctx.strokeStyle = "rgba(199,125,255,0.35)"; ctx.lineWidth = 4; for (let i = 0; i < 10; i++) { ctx.beginPath(); ctx.arc(W / 2, H * 0.45, (now / 6 + i * 80) % 800, 0, 7); ctx.stroke(); }
      break;
    case "flowers":
      for (let i = 0; i < 120; i++) { ctx.fillStyle = ["#ffafcc", "#ffd166", "#caffbf", "#f8f9fa"][i % 4]; ctx.fillRect(r(i) * W, horizon + r(i, 2) * (H - horizon), 5, 5); }
      for (let i = 0; i < 30; i++) { ctx.fillStyle = "rgba(255,255,200,0.5)"; ctx.fillRect(r(i, 5) * W, (r(i, 6) * horizon + now / 20) % horizon, 3, 3); }
      break;
    case "horizon":
      ctx.fillStyle = "#0077b6"; ctx.fillRect(0, horizon - H * 0.15, W, H * 0.15);
      for (let i = 0; i < 40; i++) { ctx.fillStyle = "#03045e"; const x = (r(i) * W + now / (5 + i % 5)) % W; ctx.beginPath(); ctx.ellipse(x, r(i, 2) * horizon * 0.8, 12, 5, 0, 0, 7); ctx.fill(); }
      break;
    case "swirl":
      for (let i = 0; i < 6; i++) { ctx.strokeStyle = `rgba(157,78,221,${0.15 + i * 0.05})`; ctx.lineWidth = 8; ctx.beginPath(); ctx.arc(W / 2, H * 0.4, 60 + i * 60, now / 800 + i, now / 800 + i + 4); ctx.stroke(); }
      break;
  }
  if (domain) {
    const v = ctx.createRadialGradient(W / 2, H / 2, H * 0.3, W / 2, H / 2, H);
    v.addColorStop(0, "rgba(0,0,0,0)"); v.addColorStop(1, "rgba(0,0,0,0.6)");
    ctx.fillStyle = v; ctx.fillRect(0, 0, W, H);
  }
}

// Depth pass over any backdrop: perspective floor, light, drifting motes, foreground framing, vignette.
function finishBackground(ctx, W, H, bg, now, domain, clock = 12) {
  const horizon = H * 0.43;
  const night = !domain && (clock >= 19 || clock < 5);
  // perspective floor lines
  ctx.save();
  ctx.beginPath(); ctx.rect(0, horizon, W, H - horizon); ctx.clip();
  ctx.strokeStyle = "rgba(0,0,0,0.13)"; ctx.lineWidth = 2;
  for (let i = -12; i <= 12; i++) {
    ctx.beginPath(); ctx.moveTo(W / 2 + i * 26, horizon); ctx.lineTo(W / 2 + i * 150, H); ctx.stroke();
  }
  for (let k = 1; k < 7; k++) {
    const y = horizon + (H - horizon) * Math.pow(k / 7, 1.8);
    ctx.strokeStyle = `rgba(0,0,0,${0.05 + k * 0.012})`; ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke();
  }
  // soft light pool where the fighters stand
  const pool = ctx.createRadialGradient(W * 0.5, H * 0.66, 10, W * 0.5, H * 0.66, W * 0.5);
  pool.addColorStop(0, night ? "rgba(120,140,255,0.16)" : "rgba(255,244,200,0.22)"); pool.addColorStop(1, "rgba(0,0,0,0)");
  ctx.fillStyle = pool; ctx.fillRect(0, horizon, W, H - horizon);
  ctx.restore();
  // horizon haze
  const hz = ctx.createLinearGradient(0, horizon - H * 0.1, 0, horizon + H * 0.04);
  hz.addColorStop(0, "rgba(255,255,255,0)"); hz.addColorStop(0.7, night ? "rgba(90,110,200,0.16)" : "rgba(255,255,255,0.22)"); hz.addColorStop(1, "rgba(255,255,255,0)");
  ctx.fillStyle = hz; ctx.fillRect(0, horizon - H * 0.1, W, H * 0.14);
  // god rays
  if (!domain) {
    ctx.save(); ctx.globalCompositeOperation = "lighter";
    for (let i = 0; i < 5; i++) {
      const x0 = W * (0.1 + i * 0.2) + Math.sin(now / 3000 + i) * 20;
      const g = ctx.createLinearGradient(x0, 0, x0 + 120, horizon);
      g.addColorStop(0, night ? "rgba(140,160,255,0.08)" : "rgba(255,240,190,0.13)"); g.addColorStop(1, "rgba(255,240,190,0)");
      ctx.fillStyle = g;
      ctx.beginPath(); ctx.moveTo(x0, 0); ctx.lineTo(x0 + 46, 0); ctx.lineTo(x0 + 220, horizon); ctx.lineTo(x0 + 120, horizon); ctx.closePath(); ctx.fill();
    }
    ctx.restore();
  }
  // drifting motes
  const motifs = { fog: "#c77dff", factory: "#ffb347", tunnel: "#9aa0a6", graves: "#cfd8dc", torii: "#ffc4d6", trees: "#d8f3a0", hills: "#fff6c8" };
  const mc = domain ? "#ffffff" : motifs[bg.motif] || "#ffffff";
  ctx.fillStyle = mc;
  for (let i = 0; i < 34; i++) {
    const x = (hash(i, 1, 21) * W + now * (0.012 + hash(i, 2, 21) * 0.02)) % W, y = (hash(i, 3, 21) * H + Math.sin(now / 900 + i) * 8 + now * -0.01 * hash(i, 4, 21)) % H;
    ctx.globalAlpha = 0.25 + 0.4 * Math.abs(Math.sin(now / 700 + i));
    ctx.fillRect(Math.round(x), Math.round((y + H) % H), 2, 2);
  }
  ctx.globalAlpha = 1;
  // foreground framing blades / silhouettes
  if (!domain) {
    ctx.fillStyle = shade(bg.floor, -0.55);
    for (let i = 0; i < 46; i++) {
      const x = (i / 46) * W + hash(i, 5, 22) * 14, h0 = 14 + hash(i, 6, 22) * 30 + (i % 7 === 0 ? 24 : 0), sway = Math.sin(now / 600 + i) * 3;
      ctx.beginPath(); ctx.moveTo(x - 5, H); ctx.lineTo(x + sway, H - h0); ctx.lineTo(x + 5, H); ctx.fill();
    }
  }
  // vignette
  const v = ctx.createRadialGradient(W / 2, H / 2, Math.min(W, H) * 0.4, W / 2, H / 2, Math.hypot(W, H) * 0.62);
  v.addColorStop(0, "rgba(0,0,0,0)"); v.addColorStop(1, domain ? "rgba(0,0,0,0.55)" : "rgba(5,4,16,0.5)");
  ctx.fillStyle = v; ctx.fillRect(0, 0, W, H);
}

function clouds(ctx, W, H, now, color = "rgba(255,255,255,0.8)") {
  ctx.fillStyle = color;
  for (let i = 0; i < 5; i++) {
    const x = ((hash(i, 1) * W + now / (40 + i * 10)) % (W + 300)) - 150, y = hash(i, 2) * H * 0.25;
    ctx.beginPath(); ctx.ellipse(x, y, 70, 18, 0, 0, 7); ctx.ellipse(x + 40, y - 10, 40, 16, 0, 0, 7); ctx.fill();
  }
}

function fog(ctx, W, y, now, color) {
  ctx.fillStyle = color;
  for (let i = 0; i < 6; i++) {
    const x = ((hash(i, 7) * W + now / 50) % (W + 400)) - 200;
    ctx.beginPath(); ctx.ellipse(x, y - 20 + hash(i, 8) * 40, 200, 30, 0, 0, 7); ctx.fill();
  }
}
