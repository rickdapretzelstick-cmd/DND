// Tiny synthesized soundscape: no audio files. Everything is generated with Web Audio.
let ctx = null, master = null, musicGain = null, sfxGain = null;
let muted = localStorage.getItem("jjk_mute") === "1";
let musicMode = "", musicTimer = 0, step = 0, lastStep = 0;

function init() {
  if (ctx) return;
  try {
    ctx = new (window.AudioContext || window.webkitAudioContext)();
    master = ctx.createGain(); master.gain.value = muted ? 0 : 0.7; master.connect(ctx.destination);
    sfxGain = ctx.createGain(); sfxGain.gain.value = 0.55; sfxGain.connect(master);
    musicGain = ctx.createGain(); musicGain.gain.value = 0.16; musicGain.connect(master);
  } catch { ctx = null; }
}
export function unlock() { init(); if (ctx && ctx.state === "suspended") ctx.resume(); }
addEventListener("pointerdown", unlock); addEventListener("keydown", unlock);

export function isMuted() { return muted; }
export function toggleMute() {
  muted = !muted; localStorage.setItem("jjk_mute", muted ? "1" : "0");
  init(); if (master) master.gain.value = muted ? 0 : 0.7;
  return muted;
}

function tone(freq, t0, dur, type = "sine", vol = 0.3, slideTo = 0, dest = sfxGain) {
  if (!ctx) return;
  const o = ctx.createOscillator(), g = ctx.createGain();
  o.type = type; o.frequency.setValueAtTime(freq, t0);
  if (slideTo) o.frequency.exponentialRampToValueAtTime(Math.max(20, slideTo), t0 + dur);
  g.gain.setValueAtTime(0.0001, t0); g.gain.exponentialRampToValueAtTime(vol, t0 + Math.min(0.02, dur / 3)); g.gain.exponentialRampToValueAtTime(0.0001, t0 + dur);
  o.connect(g); g.connect(dest); o.start(t0); o.stop(t0 + dur + 0.05);
}
function noise(t0, dur, vol = 0.3, lo = 200, hi = 6000, dest = sfxGain) {
  if (!ctx) return;
  const n = Math.floor(ctx.sampleRate * dur), buf = ctx.createBuffer(1, n, ctx.sampleRate), d = buf.getChannelData(0);
  for (let i = 0; i < n; i++) d[i] = (Math.random() * 2 - 1) * (1 - i / n);
  const s = ctx.createBufferSource(); s.buffer = buf;
  const f = ctx.createBiquadFilter(); f.type = "bandpass"; f.frequency.value = Math.sqrt(lo * hi); f.Q.value = 0.7;
  const g = ctx.createGain(); g.gain.value = vol;
  s.connect(f); f.connect(g); g.connect(dest); s.start(t0);
}

const SFX = {
  click: (t) => { tone(880, t, 0.06, "square", 0.08); },
  select: (t) => { tone(660, t, 0.05, "triangle", 0.1); tone(990, t + 0.05, 0.08, "triangle", 0.1); },
  step: (t) => { noise(t, 0.05, 0.05, 120, 900); },
  hit: (t) => { noise(t, 0.14, 0.45, 150, 3500); tone(140, t, 0.16, "sine", 0.4, 60); },
  crit: (t) => { noise(t, 0.2, 0.55, 200, 5000); tone(220, t, 0.25, "sawtooth", 0.3, 70); tone(880, t + 0.02, 0.18, "square", 0.12, 1400); },
  blackflash: (t) => { noise(t, 0.5, 0.7, 100, 6000); tone(70, t, 0.6, "sawtooth", 0.5, 30); tone(1200, t, 0.35, "square", 0.15, 200); },
  miss: (t) => { noise(t, 0.12, 0.15, 800, 5000); },
  heal: (t) => { [523, 659, 784, 1046].forEach((f, i) => tone(f, t + i * 0.07, 0.25, "sine", 0.16)); },
  buff: (t) => { tone(440, t, 0.3, "triangle", 0.14, 880); },
  cast: (t) => { tone(300, t, 0.3, "sawtooth", 0.12, 900); },
  beam: (t) => { noise(t, 0.35, 0.3, 300, 8000); tone(500, t, 0.35, "sawtooth", 0.18, 120); },
  domain: (t) => { tone(55, t, 1.6, "sawtooth", 0.4, 40); tone(110, t, 1.6, "square", 0.15, 80); noise(t, 1.2, 0.35, 80, 2000); },
  overdrive: (t) => { [262, 330, 392, 523, 659].forEach((f, i) => tone(f, t + i * 0.06, 0.35, "square", 0.14)); noise(t + 0.3, 0.4, 0.5, 200, 7000); tone(60, t + 0.3, 0.5, "sawtooth", 0.45, 30); },
  turn: (t) => { tone(587, t, 0.08, "triangle", 0.12); tone(784, t + 0.08, 0.12, "triangle", 0.12); },
  warn: (t) => { tone(220, t, 0.12, "square", 0.1); tone(196, t + 0.13, 0.16, "square", 0.1); },
  victory: (t) => { [523, 659, 784, 1046, 784, 1046, 1318].forEach((f, i) => tone(f, t + i * 0.11, i > 5 ? 0.6 : 0.16, "square", 0.13)); },
  defeat: (t) => { [330, 311, 294, 262].forEach((f, i) => tone(f, t + i * 0.25, 0.5, "sawtooth", 0.14)); },
  levelup: (t) => { [392, 523, 659, 784, 1046].forEach((f, i) => tone(f, t + i * 0.09, 0.3, "triangle", 0.16)); },
  quest: (t) => { tone(659, t, 0.15, "triangle", 0.15); tone(988, t + 0.12, 0.3, "triangle", 0.15); },
  questdone: (t) => { [523, 659, 784, 1046].forEach((f, i) => tone(f, t + i * 0.1, 0.35, "triangle", 0.15)); },
  item: (t) => { tone(1200, t, 0.08, "sine", 0.12); tone(1600, t + 0.07, 0.12, "sine", 0.12); },
  coin: (t) => { tone(1318, t, 0.07, "square", 0.08); tone(1760, t + 0.07, 0.2, "square", 0.08); },
  encounter: (t) => { tone(98, t, 0.5, "sawtooth", 0.3, 55); noise(t, 0.3, 0.3, 200, 3000); tone(740, t + 0.1, 0.2, "square", 0.12); tone(554, t + 0.28, 0.3, "square", 0.12); },
};

let lastPlayed = {};
export function sfx(name, delay = 0) {
  if (muted) return;
  init(); if (!ctx || ctx.state !== "running") return;
  const nowT = performance.now();
  if (lastPlayed[name] && nowT - lastPlayed[name] < 45) return;
  lastPlayed[name] = nowT;
  const f = SFX[name]; if (f) f(ctx.currentTime + delay + 0.01);
}

// ----- generative music: a slow pad by day, a driving pulse in battle -----
const SCALES = { calm: [0, 3, 5, 7, 10], night: [0, 2, 3, 7, 8], battle: [0, 1, 5, 7, 8] };
export function setMusic(mode) {
  if (mode === musicMode) return;
  musicMode = mode;
}
export function musicTick(now) {
  if (!ctx || muted || ctx.state !== "running" || !musicMode) return;
  const beat = musicMode === "battle" ? 0.19 : 0.9;
  const t = ctx.currentTime;
  if (t < lastStep + beat) return;
  lastStep = t + 0.001;
  step++;
  const root = musicMode === "battle" ? 55 : musicMode === "night" ? 49 : 65.4;
  const sc = SCALES[musicMode === "battle" ? "battle" : musicMode === "night" ? "night" : "calm"];
  if (musicMode === "battle") {
    const deg = sc[(step * 3 + (step >> 3)) % sc.length];
    tone(root * Math.pow(2, deg / 12), t, 0.16, "sawtooth", 0.5, 0, musicGain);
    if (step % 4 === 0) { tone(root * 2, t, 0.3, "square", 0.25, root, musicGain); noise(t, 0.05, 0.4, 2000, 9000, musicGain); }
    if (step % 8 === 4) tone(root * 4 * Math.pow(2, sc[(step >> 2) % sc.length] / 12), t, 0.4, "triangle", 0.3, 0, musicGain);
  } else {
    if (step % 4 === 1) { const deg = sc[(step * 7) % sc.length]; tone(root * 4 * Math.pow(2, deg / 12), t, 2.4, "sine", 0.35, 0, musicGain); }
    if (step % 8 === 0) { tone(root, t, 4.2, "triangle", 0.5, 0, musicGain); tone(root * 1.5, t, 4.2, "sine", 0.3, 0, musicGain); }
  }
}
