import { TILE, charSprite, curseSprite, makeCanvas, SKIN, HAIR_STYLES, OUTFITS, ACCESSORIES, BUILDS } from "./pixel.js";
import { WorldView, drawCharacter, drawCurseEntity, nameplate, marker, bubble, drawLighting } from "./world.js";
import { BattleView } from "./battle.js";

const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const sign = (n) => (n >= 0 ? "+" + n : String(n));
const mod = (s) => Math.floor((s - 10) / 2);
const yen = (n) => "¥" + Number(n || 0).toLocaleString();

// ---------- state ----------
let token = localStorage.getItem("jjk_token") || "";
const me = { playerId: "", isHost: false, name: "" };
let DATA = null;
let S = null; // snapshot
let world = null;
let worldSeed = null;
const battle = new BattleView();
const ents = new Map(); // id -> {x,y,tx,ty,dir,moving}
let roamers = new Map();
let clock = 9;
const keys = new Set();
let lastLogId = 0;
const bubbles = new Map(); // name -> {text, t0, dur, kind}
let dialog = null;
let watchBattle = false;
let battleTarget = "";
let storyQueue = [];
let storyTimer = 0;
let panel = "";
let lastFrame = performance.now();
let lastSent = { x: 0, y: 0, t: 0, moving: false };
let sendTimer = 0;
let zoom = 3;

// ---------- api ----------
async function api(path, body) {
  const res = await fetch(path, {
    method: body === undefined ? "GET" : "POST",
    headers: { "Content-Type": "application/json", "X-Token": token },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  let data = {};
  try { data = await res.json(); } catch { /* empty */ }
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}
async function act(path, body, okMsg) {
  try { const r = await api(path, body ?? {}); if (okMsg) toast(okMsg, true); return r || true; }
  catch (e) { toast(e.message); return false; }
}
let toastT;
function toast(msg, ok) {
  const t = $("#toast");
  t.textContent = msg;
  t.className = ok ? "ok" : "";
  clearTimeout(toastT);
  toastT = setTimeout(() => t.classList.add("hidden"), 3500);
}
function show(id) {
  for (const s of ["join", "create"]) $("#" + s).classList.toggle("hidden", s !== id);
  $("#hud").classList.toggle("hidden", id !== "game");
}

// ---------- boot ----------
async function boot() {
  DATA = await api("/api/data");
  const who = await fetch("/api/whoami?token=" + encodeURIComponent(token)).then((r) => r.json());
  me.isHost = who.isHost;
  $("#btn-host").classList.toggle("hidden", !me.isHost);
  if ("ontouchstart" in window) $("#dpad").classList.remove("hidden");
  if (who.playerId) { me.playerId = who.playerId; me.name = who.name; connect(); }
  else { show("join"); $("#join-name").focus(); }
  requestAnimationFrame(frame);
}

$("#btn-join").onclick = joinTable;
$("#join-name").onkeydown = (e) => e.key === "Enter" && joinTable();
async function joinTable() {
  const name = $("#join-name").value.trim();
  if (!name) return toast("Enter your name");
  try {
    const r = await api("/api/join", { name, token });
    token = r.token;
    localStorage.setItem("jjk_token", token);
    me.playerId = r.playerId; me.name = r.name;
    connect();
  } catch (e) { toast(e.message); }
}

let es;
function connect() {
  if (es) es.close();
  es = new EventSource("/api/events?token=" + encodeURIComponent(token));
  es.addEventListener("state", (ev) => onState(JSON.parse(ev.data)));
  es.addEventListener("pos", (ev) => onPos(JSON.parse(ev.data)));
  es.onopen = () => setConn(true);
  es.onerror = () => setConn(false);
}
function setConn(on) { $("#conn").innerHTML = `<i class="dot ${on ? "on" : ""}"></i>${on ? "live" : "offline"}`; }

const myPlayer = () => S?.players?.find((p) => p.id === me.playerId);
const myChar = () => { const p = myPlayer(); return p && S.characters.find((c) => c.id === p.charId); };
const charById = (id) => S?.characters.find((c) => c.id === id);

async function onState(snap) {
  const prevCombat = S?.combat;
  S = snap;
  if (!myPlayer()) { api("/api/join", { name: me.name || "Sorcerer", token }).catch(() => {}); return; }
  if (worldSeed !== S.seed) {
    worldSeed = S.seed;
    world = null;
    const w = await api("/api/world");
    world = new WorldView(w);
    ents.clear();
    lastLogId = 0;
  }
  const ch = myChar();
  if (!ch) {
    if ($("#create").classList.contains("hidden")) openCreator();
    show("create");
    return;
  }
  show("game");
  for (const c of S.characters) {
    const e = ents.get(c.id);
    if (!e) ents.set(c.id, { x: c.x, y: c.y, tx: c.x, ty: c.y, dir: c.dir, moving: false });
    else if (Math.hypot(e.x - c.x, e.y - c.y) > 4) { e.x = e.tx = c.x; e.y = e.ty = c.y; }
  }
  battle.update(S.combat, S.characters);
  if (!S.combat) watchBattle = false;
  if (prevCombat && !S.combat) battleTarget = "";
  ingestLog();
  renderHUD();
}

function onPos(p) {
  clock = p.clock;
  const mine = myChar();
  for (const c of p.chars) {
    let e = ents.get(c.id);
    if (!e) { e = { x: c.x, y: c.y, tx: c.x, ty: c.y, dir: c.d, moving: c.m }; ents.set(c.id, e); }
    if (mine && c.id === mine.id) {
      // Trust the server when it moved us (Ijichi, the DM, a correction) and we aren't steering.
      const d = Math.hypot(e.x - c.x, e.y - c.y);
      if (d > 3 || (d > 0.75 && !e.moving && performance.now() - lastSent.t > 400)) { e.x = e.tx = c.x; e.y = e.ty = c.y; }
      continue;
    }
    e.tx = c.x; e.ty = c.y; e.dir = c.d; e.moving = c.m;
  }
  const next = new Map();
  for (const r of p.roamers) {
    const old = roamers.get(r.id);
    next.set(r.id, old ? Object.assign(old, { tx: r.x, ty: r.y, n: r.n }) : { ...r, tx: r.x, ty: r.y });
  }
  roamers = next;
}

// ---------- log, bubbles & story cards ----------
function ingestLog() {
  const fresh = S.log.filter((l) => l.id > lastLogId);
  const firstLoad = lastLogId === 0;
  if (fresh.length) lastLogId = fresh[fresh.length - 1].id;
  if (firstLoad) { renderLog(); return; }
  for (const l of fresh) {
    if (l.kind === "say") {
      bubbles.set(l.who, { text: l.text, t0: performance.now(), dur: 5000 + l.text.length * 45, kind: "say" });
      if (dialog && dialog.name === l.who) { dialog.text = l.text; dialog.waiting = false; renderDialog(); }
    } else if (l.kind === "action") {
      const t = l.text.length > 140 ? l.text.slice(0, 140) + "…" : l.text;
      bubbles.set(l.who, { text: t, t0: performance.now(), dur: 4000 + t.length * 30, kind: "action" });
    } else if (l.kind === "dm") {
      storyQueue.push(l);
    } else if (l.kind === "level") {
      toast(l.text, true);
    }
  }
  renderLog();
  pumpStory();
}

function md(text) {
  let h = esc(text).replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>").replace(/(^|[^*])\*([^*\n]+)\*/g, "$1<em>$2</em>");
  return h.split(/\n{2,}/).map((p) => "<p>" + p.replace(/\n/g, "<br>") + "</p>").join("");
}

function renderLog() {
  const box = $("#log-lines");
  const near = box.scrollHeight - box.scrollTop - box.clientHeight < 60;
  box.innerHTML = S.log.slice(-120).map((l) => {
    let body = esc(l.text);
    if (l.kind === "action") body = `<b class="who">${esc(l.who)}</b> ${body}`;
    else if (l.kind === "ooc") body = `💬 <b class="who">${esc(l.who)}:</b> ${body}`;
    else if (l.kind === "say") body = `<b>${esc(l.who)}:</b> “${body}”`;
    else if (l.kind === "dm") body = esc(l.text.replace(/\*\*/g, ""));
    return `<div class="le ${l.kind}">${body}</div>`;
  }).join("");
  if (near || $("#log").classList.contains("collapsed")) box.scrollTop = box.scrollHeight;
}

function pumpStory() {
  const card = $("#story-card");
  const mine = S && myChar();
  if (dialog || S?.choice || (S?.combat && (mine?.inBattle || watchBattle))) { card.classList.add("hidden"); return; }
  if (!card.classList.contains("hidden") && performance.now() < storyTimer) return;
  const next = storyQueue.shift();
  if (!next) { card.classList.add("hidden"); return; }
  card.innerHTML = md(next.text) + `<span class="hint">click to dismiss</span>`;
  card.classList.remove("hidden");
  storyTimer = performance.now() + Math.min(16000, 5000 + next.text.length * 35);
}
$("#story-card").onclick = () => { storyTimer = 0; $("#story-card").classList.add("hidden"); pumpStory(); };
setInterval(() => { if (performance.now() > storyTimer) pumpStory(); }, 500);
$("#log-head").onclick = () => { $("#log").classList.toggle("collapsed"); $("#log-toggle").textContent = $("#log").classList.contains("collapsed") ? "▲" : "▼"; renderLog(); };

// ---------- HUD ----------
function hpClass(f) { return f > 0.5 ? "" : f > 0.25 ? "mid" : "low"; }

function portrait(look, size = 40) {
  const [c, ctx] = makeCanvas(16, 16);
  ctx.drawImage(charSprite(look, 0, 0), 0, -1);
  return c.toDataURL();
}

function renderHUD() {
  const mine = myChar();
  // Party frames
  $("#party-frames").innerHTML = S.characters.map((c) => {
    const owner = S.players.find((p) => p.id === c.owner);
    return `<div class="frame ${c.id === mine.id ? "me" : ""} ${c.hp <= 0 ? "down" : ""}" data-char="${c.id}">
      <img src="${portrait(c.look)}" width="40" height="40" style="image-rendering:pixelated;background:#1d1830;border-radius:6px">
      <div class="info"><div class="nm">${esc(c.name)} ${owner && !owner.online ? '<span class="muted small">(away)</span>' : ""}</div>
      <div class="sub">Lv ${c.level} ${esc(techName(c.class))}${c.title ? " · " + esc(c.title) : ""}</div>
      <div class="bars"><div class="bar ${hpClass(c.hp / c.maxHp)}"><i style="width:${(100 * Math.max(0, c.hp)) / c.maxHp}%"></i></div>
      ${c.ceMax ? `<div class="bar ce"><i style="width:${(100 * c.ce) / c.ceMax}%"></i></div>` : ""}</div></div></div>`;
  }).join("");
  // Clock & DM badge
  $("#dm-badge").textContent = S.ai ? (S.model?.startsWith("ollama:") ? "🖥️ Local AI DM" : "🧠 AI DM") : "🎲 Offline DM";
  $("#dm-badge").title = S.ai ? "The AI Dungeon Master is narrating — do anything." : "Offline DM. The host can enable the AI DM for free-form play.";
  // Tracker
  const active = S.quests.filter((q) => q.stage === "active" || q.stage === "return");
  $("#tracker").innerHTML = active.length ? active.map((q) => `<div class="q ${q.kind}"><b>${esc(q.title)}</b><span class="st">${q.stage === "return" ? "Report to " + esc(q.giverName || "giver") : "Active"}</span><div>${esc(q.objective)}</div></div>`).join("")
    : `<div class="muted">${S.started ? "No active quests. Look for ! and + markers over people's heads." : "Waiting for the host to begin the story…"}</div>`;
  // Battle UI
  renderBattleUI();
  // Choice
  renderChoice();
  // Spectate banner
  const inB = mine.inBattle;
  const sp = $("#spectate");
  if (S.combat && !inB) {
    sp.classList.remove("hidden");
    sp.innerHTML = `⚔ ${esc(S.combat.title)} in progress <button class="mini" id="watch">${watchBattle ? "Back to map" : "Watch"}</button>`;
    $("#watch").onclick = () => { watchBattle = !watchBattle; renderHUD(); };
  } else sp.classList.add("hidden");
  if (panel) renderPanel();
}

const techName = (id) => DATA.techniques.find((t) => t.id === id)?.name || id;
const originName = (id) => DATA.origins.find((o) => o.id === id)?.name || id;

// ---------- known abilities (mirrors the server) ----------
function knownAbilities(ch) {
  const out = [];
  const tech = DATA.techniques.find((t) => t.id === ch.class);
  for (const id of ["strike", "guard", "binding_vow", "simple_domain", "rct"]) {
    const a = DATA.abilities[id];
    if (ch.level >= (a.minLevel || 0) && !(id === "simple_domain" && tech?.noCE)) out.push(a);
  }
  for (const id of tech?.moves || []) { const a = DATA.abilities[id]; if (a && ch.level >= (a.minLevel || 0)) out.push(a); }
  const d = DATA.abilities[tech?.domain];
  if (d && ch.level >= d.minLevel) out.push(d);
  return out;
}
const abilityCost = (ch, a) => (ch.race === "gojo" && a.cost > 1 ? a.cost - 1 : a.cost) || 0;

// ---------- battle UI ----------
function renderBattleUI() {
  const ui = $("#battle-ui");
  const mine = myChar();
  const cb = S.combat;
  const visible = cb && (mine.inBattle || watchBattle);
  ui.classList.toggle("hidden", !visible);
  $("#bar").classList.toggle("hidden", !!visible);
  $("#log").classList.toggle("hidden", !!visible);
  $("#tracker").classList.toggle("hidden", !!visible);
  $("#party-frames").classList.toggle("hidden", !!visible);
  if (!visible) return;
  const cur = cb.order[cb.turn % cb.order.length];
  $("#turn-order").innerHTML = (cb.domain ? `<span class="chip dom">🌀 ${esc(cb.domain.name.replace("Domain Expansion: ", ""))} (${cb.domain.rounds})</span>` : "") + `<span class="chip">R${cb.round}</span>` +
    cb.order.map((o, i) => {
      let out = false;
      if (o.isMonster) out = (cb.monsters.find((m) => m.id === o.id)?.hp ?? 0) <= 0;
      else { const c = charById(o.id); out = !c || c.dead || c.fled; }
      return `<span class="chip ${o.isMonster ? "foe" : ""} ${i === cb.turn % cb.order.length ? "cur" : ""} ${out ? "out" : ""}">${o.isMonster ? "👹" : "🛡️"} ${esc(o.name)}</span>`;
    }).join("");
  const combatLines = S.log.filter((l) => l.kind === "combat" || l.kind === "roll").slice(-4);
  $("#battle-ticker").innerHTML = combatLines.map((l) => `<div>${esc(l.text)}</div>`).join("");
  const menu = $("#battle-menu");
  const mineTurn = cur && cur.id === mine.id && mine.inBattle;
  if (!mineTurn) {
    menu.innerHTML = `<div class="head"><span class="muted">${mine.inBattle ? "Waiting for" : "Watching —"} <b>${esc(cur?.name || "")}</b>…</span>${me.isHost && cur && !cur.isMonster ? '<button class="mini" id="b-skip">Skip their turn</button>' : ""}</div>`;
    const sk = $("#b-skip");
    if (sk) sk.onclick = () => act("/api/host/skip", {});
    return;
  }
  const abil = knownAbilities(mine);
  const alive = cb.monsters.filter((m) => m.hp > 0);
  if (!alive.find((m) => m.id === battleTarget) && !S.characters.find((c) => c.id === battleTarget)) battleTarget = alive[0]?.id || "";
  const tgtName = cb.monsters.find((m) => m.id === battleTarget)?.name || charById(battleTarget)?.name || "—";
  const used = mine.used || {};
  const btns = abil.map((a, i) => {
    const cost = abilityCost(mine, a);
    const burnt = mine.status?.burnout && a.tech && !a.universal;
    const bad = (cost > mine.ce && !(mine.status?.jackpot)) || (a.perBattle && (used[a.id] || 0) >= a.perBattle) || (a.hpCost && mine.hp <= a.hpCost) || burnt ||
      (a.bonus && cb.bonusUsed) || (!a.bonus && cb.actionUsed && !cb.extraAction);
    const costTxt = [cost ? cost + " CE" : "", a.hpCost ? a.hpCost + " HP" : "", a.perBattle ? `${a.perBattle - (used[a.id] || 0)}/${a.perBattle} left` : "", a.bonus ? "bonus" : ""].filter(Boolean).join(" · ") || "free";
    return `<button class="ab ${a.domain ? "dom" : ""}" style="--c:${a.color}" data-ab="${a.id}" ${bad ? "disabled" : ""} data-tip="${esc(a.desc)}"><span class="key">${i < 9 ? i + 1 : ""}</span><b>${esc(a.name)}</b><span class="cost">${esc(costTxt)}</span></button>`;
  }).join("");
  const items = mine.inventory.map((it, i) => ({ it, i })).filter((x) => x.it.kind === "consumable");
  menu.innerHTML = `<div class="head"><span class="yourturn">⚔ Your turn, ${esc(mine.name)}!</span>
    <span class="small muted">Target: <b style="color:#ff8fab">${esc(tgtName)}</b> (click a sprite) · Action ${cb.actionUsed && !cb.extraAction ? "used" : "ready"} · Bonus ${cb.bonusUsed ? "used" : "ready"} · CE ${mine.ce}/${mine.ceMax}</span></div>
    <div class="abilities">${btns}
      ${items.map((x) => `<button class="ab" style="--c:#95d5b2" data-item="${x.i}" data-tip="${esc(x.it.desc || x.it.name)}"><b>🧪 ${esc(x.it.name)}${x.it.qty > 1 ? " ×" + x.it.qty : ""}</b><span class="cost">item · action</span></button>`).join("")}
      <button class="ab" style="--c:#adb5bd" data-c="flee"><b>🏃 Flee</b><span class="cost">Agility check</span></button>
      <button class="ab" style="--c:#ffd166" data-c="end"><b>⏭ End turn</b><span class="cost">pass</span></button>
    </div>`;
  $$("[data-ab]", menu).forEach((b) => (b.onclick = () => useAbility(b.dataset.ab)));
  $$("[data-item]", menu).forEach((b) => (b.onclick = () => act("/api/combat", { type: "item", item: +b.dataset.item, target: battleTarget || mine.id })));
  $$("[data-c]", menu).forEach((b) => (b.onclick = () => act("/api/combat", { type: b.dataset.c })));
  bindTips(menu);
}

function useAbility(id) {
  const a = DATA.abilities[id];
  const mine = myChar();
  const tgtKind = a.effects?.[0]?.target || "enemy";
  let target = battleTarget;
  if ((tgtKind === "ally" || tgtKind === "self") && !charById(battleTarget)) target = mine.id;
  act("/api/combat", { type: "ability", ability: id, target });
}

function bindTips(root) {
  let tip = $("#tooltip");
  if (!tip) { tip = document.createElement("div"); tip.id = "tooltip"; tip.className = "hidden"; document.body.appendChild(tip); }
  $$("[data-tip]", root).forEach((el) => {
    el.onmouseenter = () => { tip.textContent = el.dataset.tip; tip.classList.remove("hidden"); };
    el.onmousemove = (e) => { tip.style.left = Math.min(innerWidth - 310, e.clientX + 14) + "px"; tip.style.top = e.clientY - 70 + "px"; };
    el.onmouseleave = () => tip.classList.add("hidden");
  });
}

// ---------- party choice ----------
function renderChoice() {
  const c = S.choice;
  const box = $("#choice");
  if (!c) { box.classList.add("hidden"); return; }
  box.classList.remove("hidden");
  $("#choice-speaker").textContent = c.speaker || "";
  $("#choice-title").textContent = c.title;
  $("#choice-text").textContent = c.text;
  $("#choice-options").innerHTML = c.options.map((o) => `<button data-opt="${o.id}">${o.need ? `<span class="need">${esc(o.need)}</span>` : ""}${esc(o.text)}</button>`).join("");
  $$("#choice-options button").forEach((b) => (b.onclick = () => act("/api/choose", { option: b.dataset.opt })));
}

// ---------- dialog ----------
async function interact(npcId) {
  $("#story-card").classList.add("hidden");
  const r = await act("/api/interact", { npc: npcId });
  if (r && r.dialog) { dialog = r.dialog; renderDialog(); }
}
function renderDialog() {
  const box = $("#dialog");
  if (!dialog) { box.classList.add("hidden"); return; }
  box.classList.remove("hidden");
  $("#dialog-name").textContent = dialog.name;
  $("#dialog-text").textContent = dialog.waiting ? "…" : dialog.text;
  const pc = $("#dialog-portrait").getContext("2d");
  pc.imageSmoothingEnabled = false;
  pc.clearRect(0, 0, 96, 144);
  pc.drawImage(charSprite(dialog.look, 0, 0), 0, 0, 96, 144);
  $("#dialog-options").innerHTML = (dialog.options || []).map((o, i) => `<button data-o="${o.id}"><span class="k">${i + 1}</span>${esc(o.text)}</button>`).join("");
  $$("#dialog-options button").forEach((b) => (b.onclick = () => chooseDialog(b.dataset.o)));
  $("#dialog-free").classList.toggle("hidden", !dialog.free);
  if (dialog.free) setTimeout(() => $("#dialog-input").focus(), 30);
}
async function chooseDialog(id) {
  try {
    const r = await api("/api/dialog", { option: id });
    const keepFree = r.dialog?.free;
    dialog = r.dialog || null;
    if (dialog && keepFree) dialog.free = true;
    renderDialog();
  } catch (e) { toast(e.message); }
}
async function sayInDialog() {
  const t = $("#dialog-input").value.trim();
  if (!t || !dialog) return;
  $("#dialog-input").value = "";
  dialog.waiting = true;
  renderDialog();
  if (!(await act("/api/talk", { npc: dialog.npc, text: t }))) { dialog.waiting = false; renderDialog(); }
}
$("#dialog-send").onclick = sayInDialog;
$("#dialog-input").onkeydown = (e) => { if (e.key === "Enter") sayInDialog(); if (e.key === "Escape") closeDialog(); e.stopPropagation(); };
$("#dialog-close").onclick = () => closeDialog();
function closeDialog() { dialog = null; renderDialog(); }

// ---------- action bar ----------
async function sendAct() {
  const box = $("#act");
  const text = box.value.trim();
  if (!text) { box.blur(); return; }
  const mode = $("#act-mode").value;
  if (await act(mode === "act" ? "/api/act" : "/api/ooc", { text })) box.value = "";
  box.blur();
}
$("#act-send").onclick = sendAct;
$("#act").onkeydown = (e) => { e.stopPropagation(); if (e.key === "Enter") sendAct(); if (e.key === "Escape") $("#act").blur(); };

// ---------- panels ----------
$$("[data-panel]").forEach((b) => (b.onclick = () => togglePanel(b.dataset.panel)));
$("#panel-close").onclick = () => togglePanel("");
$("#minimap").onclick = () => togglePanel("map");
function togglePanel(p) {
  panel = panel === p ? "" : p;
  $("#panel").classList.toggle("hidden", !panel);
  if (panel) renderPanel();
}

function renderPanel() {
  const body = $("#panel-body");
  const ch = myChar();
  if (panel === "sheet") {
    const tech = DATA.techniques.find((t) => t.id === ch.class);
    const prof = 2 + Math.floor((ch.level - 1) / 4);
    const next = DATA.xp[ch.level] ?? ch.xp;
    const others = S.characters.filter((c) => c.id !== ch.id && !c.dead);
    body.innerHTML = `<div class="sheet"><div><canvas id="sheet-sprite" width="192" height="288"></canvas>
      <div class="muted small" style="margin-top:6px">${esc(originName(ch.race))}</div>
      ${ch.title ? `<div style="color:var(--gold)">🏅 ${esc(ch.title)}</div>` : ""}${ch.vessel ? '<div style="color:#ff4d6d">🩸 Vessel of Sukuna</div>' : ""}
      <button class="mini" id="retire" style="margin-top:8px">Retire this sorcerer</button></div>
      <div><h2>${esc(ch.name)}</h2><div class="muted">Level ${ch.level} · ${esc(tech?.name)} · ${gradeOf(ch.level)} sorcerer · XP ${ch.xp}${ch.level < 12 ? " / " + next : ""}</div>
      <div class="kv"><div><b>${ch.hp}/${ch.maxHp}</b><span>HP</span></div><div><b>${ch.ce}/${ch.ceMax}</b><span>Cursed energy</span></div><div><b>${ch.ac}</b><span>Defense</span></div>
      <div><b>${sign(prof)}</b><span>Proficiency</span></div><div><b>${yen(ch.gold)}</b><span>Yen</span></div><div><b>${esc(gradeOf(ch.level))}</b><span>Grade</span></div></div>
      <div class="kv" style="grid-template-columns:repeat(6,1fr)">${DATA.abilityOrder.map((a) => `<div><b>${ch.stats[a]}</b><span>${DATA.abilityNames[a]} ${sign(mod(ch.stats[a]))}</span></div>`).join("")}</div>
      <h3>Techniques</h3>${knownAbilities(ch).map((a) => `<div class="abl"><b>${esc(a.name)}</b> <span class="muted">${a.cost ? abilityCost(ch, a) + " CE" : ""}${a.hpCost ? " · " + a.hpCost + " HP" : ""}${a.bonus ? " · bonus" : ""}</span> — ${esc(a.desc)}
        ${!ch.inBattle && a.effects?.some((e) => e.kind === "heal") ? `<button class="mini" data-heal="${a.id}">Use</button>` : ""}</div>`).join("")}
      <div class="muted small">Coming: ${(tech?.abilities || []).filter((a) => a.minLevel > ch.level).map((a) => `${esc(a.name)} (Lv ${a.minLevel})`).join(", ") || "—"}</div>
      <h3>Inventory</h3><ul class="inv">${ch.inventory.map((it, i) => `<li><span>${esc(it.name)}${it.qty > 1 ? " ×" + it.qty : ""}${it.equipped ? '<span class="eq">EQUIPPED</span>' : ""}<br><span class="muted small">${esc(itemInfo(it))}</span></span>
        <span class="acts">${["weapon", "armor"].includes(it.kind) ? `<button class="mini" data-eq="${i}">${it.equipped ? "Unequip" : "Equip"}</button>` : ""}
        ${it.kind === "consumable" && (it.heal || it.ce) ? `<button class="mini" data-use="${i}">Use</button>` : ""}
        ${others.length && !it.equipped ? `<button class="mini" data-give="${i}">Give</button>` : ""}</span></li>`).join("")}</ul>
      <h3>Standing</h3><div class="rep">${[["higher", "Higher-ups"], ["civ", "Civilians"], ["curse", "Underworld"], ["gojo", "Gojo's trust"]].map(([k, n]) => `<div><b>${sign(S.rep?.[k] || 0)}</b>${n}</div>`).join("")}</div>
      ${ch.notes?.length ? `<h3>The DM remembers</h3><ul>${ch.notes.map((n) => `<li class="small">${esc(n)}</li>`).join("")}</ul>` : ""}
      ${ch.backstory ? `<h3>Backstory</h3><p class="small muted">${esc(ch.backstory)}</p>` : ""}</div></div>`;
    const sc = $("#sheet-sprite").getContext("2d");
    sc.imageSmoothingEnabled = false;
    sc.drawImage(charSprite(ch.look, 0, 0), 0, 0, 192, 288);
    $$("[data-eq]", body).forEach((b) => (b.onclick = () => act("/api/equip", { index: +b.dataset.eq })));
    $$("[data-use]", body).forEach((b) => (b.onclick = () => act("/api/use", { index: +b.dataset.use, target: ch.id })));
    $$("[data-heal]", body).forEach((b) => (b.onclick = () => act("/api/ability", { ability: b.dataset.heal, target: ch.id })));
    $$("[data-give]", body).forEach((b) => (b.onclick = () => {
      const n = prompt("Give to:\n" + others.map((c, i) => `${i + 1}. ${c.name}`).join("\n"), "1");
      const c = others[(+n || 0) - 1];
      if (c) act("/api/give", { index: +b.dataset.give, to: c.id });
    }));
    $("#retire").onclick = () => { if (confirm(`Retire ${ch.name} forever and create a new sorcerer?`)) act("/api/retire", { charId: ch.id }); };
  } else if (panel === "quests") {
    const qs = S.quests.slice().reverse();
    body.innerHTML = `<h2>📜 Quests</h2>${qs.length ? qs.map((q) => `<div class="qcard ${q.kind}"><h4>${esc(q.title)} <span class="muted small">— ${q.kind === "main" ? "Main story" : q.kind === "ai" ? "Story" : "Job"} · ${esc(q.stage)}</span></h4>
      <div>${esc(q.desc)}</div><div class="small" style="margin-top:4px">🎯 ${esc(q.objective)}</div>
      <div class="small muted">${q.giverName ? "From " + esc(q.giverName) + " · " : ""}Reward: ${q.rewardXp} XP, ${yen(q.rewardYen)}${q.minLevel > 1 ? " · Recommended level " + q.minLevel : ""}</div></div>`).join("") : '<p class="muted">No quests yet.</p>'}
      <h3>Journal</h3>${S.journal?.length ? `<ul>${S.journal.map((j) => `<li class="small">${esc(j)}</li>`).join("")}</ul>` : '<p class="muted small">Your story is just beginning.</p>'}`;
  } else if (panel === "map") {
    body.innerHTML = `<h2>🗺️ Map</h2><canvas id="bigmap"></canvas><p class="muted small">◆ you · ● party · ! quest · ☠ cursed site · Towns you've visited are named. Ijichi at Jujutsu High can drive you to any visited town.</p>`;
    drawBigMap($("#bigmap"));
  } else {
    body.innerHTML = "";
  }
}
const gradeOf = (l) => (l >= 11 ? "Special Grade" : l >= 9 ? "Grade 1" : l >= 7 ? "Semi-Grade 1" : l >= 5 ? "Grade 2" : l >= 3 ? "Grade 3" : "Grade 4");
function itemInfo(it) {
  if (it.kind === "weapon") return `${it.damage} ${it.dmgType || ""}${it.bonus ? " · +" + it.bonus : ""}${it.special === "ignore_ac" ? " · ignores armor" : it.special === "dispel" ? " · nullifies techniques" : ""}`;
  if (it.kind === "armor") return `Defense ${it.baseAC}${it.maxDex >= 0 ? " (max Agility +" + it.maxDex + ")" : " + Agility"}`;
  if (it.heal || it.ce) return [it.heal ? "heals " + it.heal : "", it.ce ? "+" + it.ce + " CE" : ""].filter(Boolean).join(", ");
  if (it.damage) return "deals " + it.damage;
  return it.desc || it.kind;
}

function drawBigMap(cv) {
  if (!world) return;
  const W = world.w.w, H = world.w.h, s = 4;
  cv.width = W * s; cv.height = H * s;
  const ctx = cv.getContext("2d");
  ctx.imageSmoothingEnabled = false;
  ctx.drawImage(world.minimap, 0, 0, W * s, H * s);
  ctx.font = "bold 13px system-ui";
  ctx.textAlign = "center";
  for (const s2 of world.w.sites) {
    ctx.fillStyle = S.cleared?.[s2.id] ? "#adb5bd" : "#c77dff";
    ctx.fillText("☠", s2.x * s, s2.y * s);
  }
  for (const t of world.w.towns) {
    const known = S.discovered?.[t.id];
    ctx.fillStyle = "rgba(0,0,0,0.6)";
    const label = known ? t.name : "???";
    const tw = ctx.measureText(label).width;
    ctx.fillRect(t.x * s - tw / 2 - 4, t.y * s - 30, tw + 8, 17);
    ctx.fillStyle = S.burned?.includes(t.id) ? "#ff7b00" : "#fff";
    ctx.fillText(label, t.x * s, t.y * s - 17);
  }
  for (const q of S.quests) {
    if (q.stage !== "active") continue;
    ctx.fillStyle = q.kind === "main" ? "#ff4d6d" : "#ffd60a";
    ctx.font = "bold 22px system-ui";
    ctx.fillText("!", q.x * s, q.y * s);
  }
  for (const c of S.characters) {
    const e = ents.get(c.id) || c;
    ctx.fillStyle = c.id === myChar().id ? "#ffd166" : "#4cc9f0";
    ctx.beginPath(); ctx.arc(e.x * s, e.y * s, c.id === myChar().id ? 7 : 5, 0, 7); ctx.fill();
    ctx.strokeStyle = "#000"; ctx.lineWidth = 2; ctx.stroke();
  }
  ctx.textAlign = "left";
}

// ---------- character creator ----------
const HAIR_COLORS = ["#1b1b1b", "#3b2a1a", "#6b4423", "#a0522d", "#d4a373", "#f1e3c4", "#f8f9fa", "#e63946", "#ff8fab", "#2d6a4f", "#457b9d", "#7b2cbf"];
const CLOTH_COLORS = ["#14141c", "#22223b", "#2b2d42", "#3d405b", "#264653", "#2a9d8f", "#e9c46a", "#e76f51", "#9d0208", "#f8f9fa", "#6c584c", "#457b9d", "#7b2cbf", "#ff8fab"];
const EYE_COLORS = ["#3b2a1a", "#1b1b1b", "#4a6fa5", "#7ec8ff", "#6a994e", "#d00000", "#ffd166", "#b14aed"];
let cr = null;
let previewDir = 0;

function openCreator() {
  cr = cr || {
    name: "", race: "non_clan", tech: "limitless", method: "standard", values: [...DATA.standardArray], base: {}, swap: null,
    look: { skin: 1, hair: 1, hairColor: "#1b1b1b", eyes: "#3b2a1a", outfit: 0, color1: "#14141c", color2: "#2a2a3a", accessory: 0, build: 1 },
  };
  autoAssign();
  drawCreator();
}

function autoAssign() {
  const t = DATA.techniques.find((x) => x.id === cr.tech);
  const vals = [...cr.values].sort((a, b) => b - a);
  cr.base = {};
  t.priority.forEach((ab, i) => (cr.base[ab] = vals[i]));
  cr.swap = null;
}

function stepper(label, list, key) {
  return `<div class="look-row"><span class="lbl">${label}</span><div class="stepper"><button class="mini" data-step="${key}" data-d="-1">◀</button><span>${esc(list[cr.look[key]])}</span><button class="mini" data-step="${key}" data-d="1">▶</button></div></div>`;
}
function swatches(label, colors, key, custom = true) {
  return `<div class="look-row"><span class="lbl">${label}</span><div class="swatches">${colors.map((c) => `<span class="sw ${cr.look[key] === c ? "sel" : ""}" style="background:${c}" data-sw="${key}" data-c="${c}"></span>`).join("")}${custom ? `<input type="color" value="${cr.look[key]}" data-pick="${key}">` : ""}</div></div>`;
}

function drawCreator() {
  const L = cr.look;
  $("#look-controls").innerHTML =
    `<div class="look-row"><span class="lbl">Skin</span><div class="swatches">${SKIN.map((c, i) => `<span class="sw ${L.skin === i ? "sel" : ""}" style="background:${c}" data-skin="${i}"></span>`).join("")}</div></div>` +
    stepper("Hair", HAIR_STYLES, "hair") + swatches("Hair color", HAIR_COLORS, "hairColor") + swatches("Eyes", EYE_COLORS, "eyes") +
    stepper("Outfit", OUTFITS, "outfit") + swatches("Main color", CLOTH_COLORS, "color1") + swatches("Accent", CLOTH_COLORS, "color2") +
    stepper("Accessory", ACCESSORIES, "accessory") + stepper("Build", BUILDS, "build");
  $$("[data-skin]").forEach((el) => (el.onclick = () => { L.skin = +el.dataset.skin; drawCreator(); }));
  $$("[data-step]").forEach((el) => (el.onclick = () => {
    const k = el.dataset.step;
    const n = { hair: HAIR_STYLES, outfit: OUTFITS, accessory: ACCESSORIES, build: BUILDS }[k].length;
    L[k] = (L[k] + +el.dataset.d + n) % n;
    drawCreator();
  }));
  $$("[data-sw]").forEach((el) => (el.onclick = () => { L[el.dataset.sw] = el.dataset.c; drawCreator(); }));
  $$("[data-pick]").forEach((el) => (el.oninput = () => { L[el.dataset.pick] = el.value; drawPreview(); }));

  $("#cr-origins").innerHTML = DATA.origins.map((o) => `<div class="card ${cr.race === o.id ? "sel" : ""}" data-origin="${o.id}"><b>${esc(o.name)}</b>
    <span class="muted">${Object.entries(o.bonus).map(([k, v]) => DATA.abilityNames[k] + " " + sign(v)).join(", ")}</span><div class="muted">${esc(o.traits)}</div></div>`).join("");
  $$("[data-origin]").forEach((el) => (el.onclick = () => { cr.race = el.dataset.origin; drawCreator(); }));
  $("#cr-techs").innerHTML = DATA.techniques.map((t) => `<div class="card ${cr.tech === t.id ? "sel" : ""}" data-tech="${t.id}"><b>${esc(t.name)}</b><span class="muted">like ${esc(t.inspired)} · d${t.hitDie} HP</span><div class="bar" style="background:${t.color}"></div></div>`).join("");
  $$("[data-tech]").forEach((el) => (el.onclick = () => { cr.tech = el.dataset.tech; autoAssign(); drawCreator(); }));
  const t = DATA.techniques.find((x) => x.id === cr.tech);
  $("#cr-tech-info").innerHTML = `<div style="margin-bottom:6px">${esc(t.blurb)}</div>` + t.abilities.map((a) => `<div class="mv"><b>${esc(a.name)}</b> <span class="muted">${a.minLevel > 1 ? "(Lv " + a.minLevel + ")" : ""} ${a.cost ? a.cost + " CE" : a.perBattle ? a.perBattle + "/battle" : ""}${a.hpCost ? " · " + a.hpCost + " HP" : ""}</span> — ${esc(a.desc)}</div>`).join("") +
    `<div class="muted small" style="margin-top:6px">Every sorcerer also has Cursed Strike (can Black Flash), Guard, Binding Vow, Simple Domain (Lv 3) and Reverse Cursed Technique (Lv 6).</div>`;
  const o = DATA.origins.find((x) => x.id === cr.race);
  $("#cr-stats").innerHTML = DATA.abilityOrder.map((ab) => {
    const b = o.bonus[ab] || 0, tot = cr.base[ab] + b;
    return `<div class="stat ${cr.swap === ab ? "sel" : ""}" data-ab="${ab}"><div class="ab">${DATA.abilityNames[ab]}</div><div class="v">${tot}</div><div class="m">${sign(mod(tot))}${b ? ` <span class="muted">(+${b})</span>` : ""}</div></div>`;
  }).join("");
  $$("#cr-stats .stat").forEach((el) => (el.onclick = () => {
    const ab = el.dataset.ab;
    if (!cr.swap) cr.swap = ab;
    else if (cr.swap === ab) cr.swap = null;
    else { [cr.base[ab], cr.base[cr.swap]] = [cr.base[cr.swap], cr.base[ab]]; cr.swap = null; }
    drawCreator();
  }));
  const con = cr.base.CON + (o.bonus.CON || 0);
  $("#cr-preview").textContent = `Starting HP ${Math.max(1, t.hitDie + mod(con) + (t.noCE ? 4 : 0))} · ${t.noCE ? "no cursed energy" : "CE " + (9 + 2 * Math.max(0, mod(cr.base[t.priority[0]] + (o.bonus[t.priority[0]] || 0))) + (cr.race === "kamo" ? 5 : 0))}`;
  drawPreview();
}
function drawPreview() {
  const c = $("#preview").getContext("2d");
  c.imageSmoothingEnabled = false;
  c.clearRect(0, 0, 192, 288);
  c.drawImage(charSprite(cr.look, previewDir, Math.floor(performance.now() / 160) % 4), 0, 0, 192, 288);
}
setInterval(() => { if (!$("#create").classList.contains("hidden") && cr) drawPreview(); }, 160);
$("#rot-l").onclick = () => { previewDir = [1, 3, 0, 2][previewDir]; drawPreview(); };
$("#rot-r").onclick = () => { previewDir = [2, 0, 3, 1][previewDir]; drawPreview(); };
$("#randomize").onclick = () => {
  const r = (n) => Math.floor(Math.random() * n);
  Object.assign(cr.look, { skin: r(SKIN.length), hair: r(HAIR_STYLES.length), hairColor: HAIR_COLORS[r(HAIR_COLORS.length)], eyes: EYE_COLORS[r(EYE_COLORS.length)], outfit: r(OUTFITS.length), color1: CLOTH_COLORS[r(CLOTH_COLORS.length)], color2: CLOTH_COLORS[r(CLOTH_COLORS.length)], accessory: r(3) ? 0 : r(ACCESSORIES.length), build: r(3) });
  drawCreator();
};
$("#cr-std").onclick = () => { cr.method = "standard"; cr.values = [...DATA.standardArray]; autoAssign(); drawCreator(); };
$("#cr-roll").onclick = async () => { const r = await act("/api/rollstats", {}); if (r?.stats) { cr.method = "roll"; cr.values = r.stats; autoAssign(); drawCreator(); } };
$("#btn-create").onclick = async () => {
  const name = $("#cr-name").value.trim();
  if (!name) return toast("Name your sorcerer");
  const ok = await act("/api/character", { name, race: cr.race, class: cr.tech, method: cr.method, stats: cr.base, backstory: $("#cr-back").value, look: cr.look });
  if (ok) { cr = null; $("#cr-name").value = ""; $("#cr-back").value = ""; }
};

// ---------- input ----------
const typing = () => ["INPUT", "TEXTAREA", "SELECT"].includes(document.activeElement?.tagName);
addEventListener("keydown", (e) => {
  if ($("#hud").classList.contains("hidden") || typing()) return;
  const k = e.key.toLowerCase();
  if (["w", "a", "s", "d", "arrowup", "arrowdown", "arrowleft", "arrowright", "shift"].includes(k)) { keys.add(k); e.preventDefault(); return; }
  if (S?.choice && /^[1-5]$/.test(k)) { const o = S.choice.options[+k - 1]; if (o) act("/api/choose", { option: o.id }); return; }
  if (dialog && /^[1-9]$/.test(k)) { const o = dialog.options[+k - 1]; if (o) chooseDialog(o.id); return; }
  const mine = myChar();
  if (mine?.inBattle && S.combat && /^[1-9]$/.test(k)) { const b = $$("#battle-menu [data-ab]")[+k - 1]; if (b && !b.disabled) b.click(); return; }
  if (k === "escape") { if (dialog) closeDialog(); else if (panel) togglePanel(""); return; }
  if (k === "enter") { e.preventDefault(); if (!mine?.inBattle) $("#act").focus(); return; }
  if (k === "e") { if (nearNPC) interact(nearNPC.id); return; }
  if (k === "c") togglePanel("sheet");
  if (k === "q") togglePanel("quests");
  if (k === "m") togglePanel("map");
  if (k === "l") $("#log-head").click();
});
addEventListener("keyup", (e) => keys.delete(e.key.toLowerCase()));
addEventListener("blur", () => keys.clear());
$$("#dpad button").forEach((b) => {
  const map = { up: "w", down: "s", left: "a", right: "d" };
  const k = b.dataset.k;
  const on = (ev) => { ev.preventDefault(); if (k === "e") { if (nearNPC) interact(nearNPC.id); } else keys.add(map[k]); };
  const off = (ev) => { ev.preventDefault(); keys.delete(map[k]); };
  b.addEventListener("touchstart", on); b.addEventListener("touchend", off); b.addEventListener("mousedown", on); b.addEventListener("mouseup", off);
});

const cv = $("#cv");
const ctx = cv.getContext("2d");
cv.addEventListener("click", (e) => {
  if (!S?.combat) return;
  const mine = myChar();
  if (!(mine?.inBattle || watchBattle)) return;
  const id = battle.hitTest(e.clientX, e.clientY);
  if (id) { battleTarget = id; renderBattleUI(); }
});

// ---------- game loop ----------
let nearNPC = null;

function npcList() {
  if (!world) return [];
  const gone = new Set(S.goneNpcs || []);
  return [...world.w.npcs, ...(S.extraNpcs || [])].filter((n) => !gone.has(n.id));
}

function moveMe(dt) {
  const mine = myChar();
  if (!mine || !world) return;
  const e = ents.get(mine.id);
  if (!e) return;
  const busy = mine.inBattle || mine.hp <= 0 || mine.dead || typing() || dialog || S.choice;
  let dx = 0, dy = 0;
  if (!busy) {
    if (keys.has("w") || keys.has("arrowup")) dy -= 1;
    if (keys.has("s") || keys.has("arrowdown")) dy += 1;
    if (keys.has("a") || keys.has("arrowleft")) dx -= 1;
    if (keys.has("d") || keys.has("arrowright")) dx += 1;
  }
  const moving = dx !== 0 || dy !== 0;
  if (moving) {
    const len = Math.hypot(dx, dy);
    const speed = keys.has("shift") ? 6.5 : 4.2;
    const step = (speed * dt) / len;
    const ok = (x, y) => world.walkable(x - 0.28, y) && world.walkable(x + 0.28, y) && world.walkable(x - 0.28, y - 0.25) && world.walkable(x + 0.28, y - 0.25);
    if (ok(e.x + dx * step, e.y)) e.x += dx * step;
    if (ok(e.x, e.y + dy * step)) e.y += dy * step;
    e.dir = Math.abs(dx) > Math.abs(dy) ? (dx < 0 ? 1 : 2) : dy < 0 ? 3 : 0;
  }
  e.moving = moving;
  e.tx = e.x; e.ty = e.y;
  const now = performance.now();
  if (now - lastSent.t > 100 && (Math.hypot(e.x - lastSent.x, e.y - lastSent.y) > 0.02 || moving !== lastSent.moving)) {
    lastSent = { x: e.x, y: e.y, t: now, moving };
    fetch("/api/move", { method: "POST", headers: { "Content-Type": "application/json", "X-Token": token }, body: JSON.stringify({ x: e.x, y: e.y, dir: e.dir, moving }) })
      .then((r) => r.json()).then((r) => { if (r.error === "blocked") { e.x = r.x; e.y = r.y; } }).catch(() => {});
  }
}

function frame(now) {
  requestAnimationFrame(frame);
  const dt = Math.min(0.05, (now - lastFrame) / 1000);
  lastFrame = now;
  const W = innerWidth, H = innerHeight;
  if (cv.width !== W * devicePixelRatio || cv.height !== H * devicePixelRatio) {
    cv.width = W * devicePixelRatio; cv.height = H * devicePixelRatio;
  }
  ctx.setTransform(devicePixelRatio, 0, 0, devicePixelRatio, 0, 0);
  ctx.imageSmoothingEnabled = false;
  if (!S || !world || $("#hud").classList.contains("hidden")) { ctx.fillStyle = "#0b0b12"; ctx.fillRect(0, 0, W, H); return; }
  const mine = myChar();
  if (!mine) return;
  if (S.combat && (mine.inBattle || watchBattle)) {
    $("#story-card").classList.add("hidden");
    battle.draw(ctx, W, H, now, { target: battleTarget, clock });
    $("#prompt").classList.add("hidden");
    return;
  }
  moveMe(dt);
  // interpolate others
  for (const [id, e] of ents) {
    if (id === mine.id) continue;
    const k = Math.min(1, dt * 8);
    e.x += (e.tx - e.x) * k; e.y += (e.ty - e.y) * k;
  }
  for (const r of roamers.values()) {
    const k = Math.min(1, dt * 6);
    const dx = r.tx - r.x;
    r.x += dx * k; r.y += (r.ty - r.y) * k;
    if (Math.abs(dx) > 0.01) r.face = dx < 0 ? -1 : 1;
  }
  zoom = Math.max(2, Math.min(4, Math.round(H / (16 * 15))));
  const me2 = ents.get(mine.id);
  const cam = { x: me2.x, y: me2.y - 0.5 };
  // Build entity list
  const list = [];
  const markers = S.markers || {};
  nearNPC = null;
  let bestD = 2.3;
  for (const n of npcList()) {
    const d = Math.hypot(n.x - me2.x, n.y - me2.y);
    if (d < bestD) { bestD = d; nearNPC = n; }
    const face = d < 4 ? (Math.abs(me2.x - n.x) > Math.abs(me2.y - n.y) ? (me2.x < n.x ? 1 : 2) : me2.y < n.y ? 3 : 0) : 0;
    list.push({ x: n.x, y: n.y, draw: (c, sx, sy, z, t) => {
      const top = drawCharacter(c, sx, sy, z, n.look, face, false, t);
      if (d < 7 || markers[n.id]) nameplate(c, n.name, sx, top - 2, n.role === "gojo" ? "#7ec8ff" : "#ffe8a3");
      if (markers[n.id]) marker(c, markers[n.id], sx, top - 18, t);
      drawBubble(c, n.name, sx, top - 18, t);
    } });
  }
  for (const c of S.characters) {
    const e = ents.get(c.id);
    if (!e || c.dead) continue;
    const owner = S.players.find((p) => p.id === c.owner);
    list.push({ x: e.x, y: e.y, draw: (cx, sx, sy, z, t) => {
      const top = drawCharacter(cx, sx, sy, z, c.look, e.dir, e.moving, t, { aura: c.id === mine.id ? "rgba(255,209,102,0.25)" : "rgba(76,201,240,0.2)", ghost: owner && !owner.online });
      nameplate(cx, c.name, sx, top - 2, c.id === mine.id ? "#ffd166" : "#bde0fe", c.title || "");
      drawBubble(cx, c.name, sx, top - (c.title ? 32 : 18), t);
    } });
  }
  for (const r of roamers.values()) {
    const tpl = DATA.curses[r.k];
    if (!tpl) continue;
    list.push({ x: r.x, y: r.y, draw: (c, sx, sy, z, t) => {
      const top = drawCurseEntity(c, sx, sy, z, tpl.sprite, tpl.id, t, { elite: r.e });
      if (Math.hypot(r.x - me2.x, r.y - me2.y) < 7) nameplate(c, `${r.n || tpl.name}${r.c > 1 ? " ×" + r.c : ""}`, sx, top - 2, r.e ? "#ff4d6d" : "#e0aaff", tpl.grade);
    } });
  }
  // Quest markers in the world
  for (const q of S.quests) {
    if (q.stage !== "active") continue;
    list.push({ x: q.x + 0.5, y: q.y + 0.5, draw: (c, sx, sy, z, t) => {
      const pulse = 0.5 + 0.5 * Math.sin(t / 250);
      c.strokeStyle = q.kind === "main" ? `rgba(255,77,109,${pulse})` : `rgba(255,214,10,${pulse})`;
      c.lineWidth = 3;
      c.beginPath(); c.ellipse(sx, sy, 3.5 * 16 * z * 0.5, 1.6 * 16 * z * 0.5, 0, 0, 7); c.stroke();
      marker(c, "!", sx, sy - 20 * z, t);
    } });
  }
  const view = world.draw(ctx, W, H, cam, zoom, now, list);
  // Lighting
  const lights = [];
  for (const c of S.characters) {
    const e = ents.get(c.id);
    if (!e) continue;
    const [sx, sy] = view.toScreen(e.x, e.y);
    lights.push({ x: sx, y: sy - 8 * zoom, r: 90 * zoom * 0.5 + 40 });
  }
  for (const n of npcList()) {
    if (["inn", "shop", "elder", "monk"].includes(n.role)) {
      const [sx, sy] = view.toScreen(n.x, n.y - 1);
      if (sx > -200 && sy > -200 && sx < W + 200 && sy < H + 200) lights.push({ x: sx, y: sy, r: 70 * zoom * 0.5 + 30, warm: true });
    }
  }
  drawLighting(ctx, W, H, clock, lights);
  // Interaction prompt
  const pr = $("#prompt");
  if (nearNPC && !dialog && !S.choice) {
    pr.classList.remove("hidden");
    pr.innerHTML = `<kbd>E</kbd> Talk to ${esc(nearNPC.name)}`;
  } else pr.classList.add("hidden");
  drawMinimap(me2);
  const h = Math.floor(clock), m = Math.floor((clock - h) * 60);
  $("#clock").textContent = `Day ${S.day} · ${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")} ${clock >= 19 || clock < 5 ? "🌙" : "☀️"}`;
}

function drawBubble(c, name, sx, sy, t) {
  const b = bubbles.get(name);
  if (!b) return;
  if (performance.now() - b.t0 > b.dur) { bubbles.delete(name); return; }
  bubble(c, b.text, sx, sy, 260, b.kind);
}

let mmT = 0;
function drawMinimap(me2) {
  const now = performance.now();
  if (now - mmT < 150) return;
  mmT = now;
  const c = $("#minimap");
  const x = c.getContext("2d");
  x.imageSmoothingEnabled = false;
  const R = 42; // tiles shown each way
  x.fillStyle = "#0b1a3a";
  x.fillRect(0, 0, c.width, c.height);
  const s = c.width / (R * 2);
  x.drawImage(world.minimap, me2.x - R, me2.y - R, R * 2, R * 2, 0, 0, c.width, c.height);
  const to = (wx, wy) => [(wx - me2.x + R) * s, (wy - me2.y + R) * s];
  for (const r of roamers.values()) { const [px, py] = to(r.x, r.y); x.fillStyle = r.e ? "#ff4d6d" : "#c77dff"; x.fillRect(px - 2, py - 2, 4, 4); }
  for (const n of npcList()) { if (!S.markers?.[n.id]) continue; const [px, py] = to(n.x, n.y); x.fillStyle = S.markers[n.id] === "?" ? "#4cc9f0" : "#ffd60a"; x.fillRect(px - 2, py - 2, 5, 5); }
  for (const q of S.quests) {
    if (q.stage !== "active") continue;
    let [px, py] = to(q.x, q.y);
    px = Math.max(6, Math.min(c.width - 6, px)); py = Math.max(6, Math.min(c.height - 6, py));
    x.fillStyle = q.kind === "main" ? "#ff4d6d" : "#ffd60a";
    x.font = "bold 14px system-ui"; x.fillText("!", px - 3, py + 5);
  }
  for (const ch of S.characters) {
    const e = ents.get(ch.id);
    if (!e) continue;
    const [px, py] = to(e.x, e.y);
    x.fillStyle = ch.id === myChar().id ? "#ffd166" : "#4cc9f0";
    x.beginPath(); x.arc(px, py, 3.5, 0, 7); x.fill();
  }
}

// ---------- host ----------
$("#btn-host").onclick = openHost;
$("#host-close").onclick = () => $("#host-modal").classList.add("hidden");
async function openHost() {
  $("#host-modal").classList.remove("hidden");
  try {
    const info = await api("/api/host/info");
    $("#host-links").innerHTML = info.urls.length ? "Friends on your Wi-Fi open: " + info.urls.map((u) => `<code>${esc(u)}</code>`).join(" ") + `<br><span class="muted">Not on the same network? See the README (Tailscale).</span>` : "No network address found.";
    $("#h-keyhint").textContent = info.hasKey ? `(saved ${info.keyHint})` : "(none saved)";
    $("#h-model").value = info.model;
    $("#h-effort").value = info.effort;
    $("#h-provider").value = info.provider || "anthropic";
    $("#h-omodel").value = info.ollamaModel || "qwen3:14b";
    $("#h-ourl").value = info.ollamaUrl || "http://localhost:11434";
    $("#h-octx").value = String(info.ollamaCtx || 16384);
    showProvider();
  } catch (e) { toast(e.message); }
  if (S) { $("#h-mode").value = S.mode; $("#h-setting").value = S.setting || ""; $("#h-start").disabled = S.started; }
  loadCamps();
}
async function loadCamps() {
  try {
    const list = await api("/api/host/campaigns");
    $("#h-camps").innerHTML = list.map((c) => `<div class="camp ${c.current ? "cur" : ""}"><div class="grow"><b>${esc(c.name)}</b> <span class="muted small">${c.mode === "ai" ? "🧠" : "🎲"} ${new Date(c.modified).toLocaleString()}</span><div class="small muted">${esc((c.heroes || []).join(", ") || "no sorcerers yet")}</div></div>
      ${c.current ? '<span class="small">current</span>' : `<button class="mini" data-load="${c.id}">Load</button><button class="mini" data-del="${c.id}">Delete</button>`}</div>`).join("");
    $$("[data-load]").forEach((b) => (b.onclick = async () => { if (await act("/api/host/load", { id: b.dataset.load }, "Campaign loaded")) loadCamps(); }));
    $$("[data-del]").forEach((b) => (b.onclick = async () => { if (confirm("Delete this campaign forever?") && (await act("/api/host/delete", { id: b.dataset.del }))) loadCamps(); }));
  } catch (e) { toast(e.message); }
}
$("#h-start").onclick = async () => { if (await act("/api/host/start", {}, "The story begins!")) $("#host-modal").classList.add("hidden"); };
$("#h-skip").onclick = () => act("/api/host/skip", {});
$("#h-long").onclick = () => act("/api/host/rest", { kind: "long" });
$("#h-save").onclick = async () => {
  const body = { mode: $("#h-mode").value, model: $("#h-model").value, effort: $("#h-effort").value, setting: $("#h-setting").value,
    provider: $("#h-provider").value, ollamaModel: $("#h-omodel").value.trim(), ollamaUrl: $("#h-ourl").value.trim(), ollamaCtx: +$("#h-octx").value };
  const k = $("#h-key").value.trim();
  if (k) body.apiKey = k;
  if (await act("/api/host/settings", body, "Saved")) { $("#h-key").value = ""; openHost(); }
};
function showProvider() {
  const o = $("#h-provider").value === "ollama";
  $("#h-ollama").classList.toggle("hidden", !o);
  $("#h-claude").classList.toggle("hidden", o);
}
$("#h-provider").onchange = showProvider;
$("#h-otest").onclick = async () => {
  const st = $("#h-ostatus");
  st.textContent = "Checking…";
  // Save the address/model first so the test checks what's typed.
  await act("/api/host/settings", { ollamaModel: $("#h-omodel").value.trim(), ollamaUrl: $("#h-ourl").value.trim() });
  try {
    const r = await api("/api/host/ollama-test");
    if (r.error) st.innerHTML = `<span style="color:#ff8fab">❌ ${esc(r.error)}</span>`;
    else if (!r.hasModel) st.innerHTML = `<span style="color:#ffd166">⚠️ Ollama is running, but "${esc($("#h-omodel").value)}" isn't installed. Run <code>ollama pull ${esc($("#h-omodel").value)}</code>. Installed: ${esc((r.models || []).join(", ") || "none")}</span>`;
    else st.innerHTML = `<span style="color:#80ed99">✅ Ollama is running and ${esc($("#h-omodel").value)} is ready. Choose "Local model", AI mode, and Save.</span>`;
  } catch (e) { st.textContent = e.message; }
};
$("#h-new").onclick = async () => {
  if (await act("/api/host/new", { name: $("#h-new-name").value, setting: $("#h-new-setting").value, mode: $("#h-new-mode").value }, "New world created")) $("#host-modal").classList.add("hidden");
};

boot().catch((e) => toast("Couldn't reach the game server: " + e.message));
