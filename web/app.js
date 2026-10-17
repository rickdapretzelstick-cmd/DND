"use strict";

const $ = (s, el = document) => el.querySelector(s);
const $$ = (s, el = document) => [...el.querySelectorAll(s)];
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const sign = (n) => (n >= 0 ? "+" + n : String(n));
const mod = (score) => Math.floor((score - 10) / 2);

let token = localStorage.getItem("dnd_token") || "";
let me = { playerId: "", isHost: false, name: "" };
let DATA = null;
let S = null; // latest snapshot
let mode = "act";
let target = "";
let allyTarget = "";
let lastLogId = 0;
let lastCampaign = "";
let es = null;

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
  try {
    await api(path, body ?? {});
    if (okMsg) toast(okMsg, true);
    return true;
  } catch (e) {
    toast(e.message);
    return false;
  }
}

let toastTimer;
function toast(msg, ok) {
  const t = $("#toast");
  t.textContent = msg;
  t.className = ok ? "ok" : "";
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => t.classList.add("hidden"), 3500);
}

function show(id) {
  for (const s of ["join", "create", "game"]) $("#" + s).classList.toggle("hidden", s !== id);
}

// ---------- boot ----------
async function boot() {
  DATA = await api("/api/data");
  const who = await fetch("/api/whoami?token=" + encodeURIComponent(token)).then((r) => r.json());
  me.isHost = who.isHost;
  $("#btn-host").classList.toggle("hidden", !me.isHost);
  if (who.playerId) {
    me.playerId = who.playerId;
    me.name = who.name;
    connect();
  } else {
    show("join");
    $("#join-name").focus();
  }
  buildDice();
}

$("#btn-join").onclick = joinTable;
$("#join-name").onkeydown = (e) => e.key === "Enter" && joinTable();
async function joinTable() {
  const name = $("#join-name").value.trim();
  if (!name) return toast("Enter your name");
  try {
    const r = await api("/api/join", { name, token });
    token = r.token;
    localStorage.setItem("dnd_token", token);
    me.playerId = r.playerId;
    me.name = r.name;
    connect();
  } catch (e) { toast(e.message); }
}

function connect() {
  if (es) es.close();
  es = new EventSource("/api/events?token=" + encodeURIComponent(token));
  es.onmessage = (ev) => { S = JSON.parse(ev.data); render(); };
  es.onopen = () => setConn(true);
  es.onerror = () => setConn(false);
}

function setConn(on) {
  const c = $("#hdr-conn");
  c.querySelector(".dot").classList.toggle("on", on);
  c.querySelector("span:last-child").textContent = on ? "connected" : "reconnecting…";
}

// ---------- helpers ----------
const myPlayer = () => S?.players?.find((p) => p.id === me.playerId);
const myChar = () => { const p = myPlayer(); return p && S.characters.find((c) => c.id === p.charId); };
const charById = (id) => S.characters.find((c) => c.id === id);
const currentTurn = () => (S.combat ? S.combat.order[S.combat.turn % S.combat.order.length] : null);
const raceName = (id) => DATA.races.find((r) => r.id === id)?.name || id;
const classDef = (id) => DATA.classes.find((c) => c.id === id);

function hpClass(hp, max) {
  const r = max ? hp / max : 0;
  return r > 0.5 ? "" : r > 0.25 ? "mid" : "low";
}

function md(text) {
  // Tiny, safe markdown: escape first, then **bold**, *italic*, paragraphs.
  let h = esc(text).replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>").replace(/(^|[^*])\*([^*\n]+)\*/g, "$1<em>$2</em>");
  return h.split(/\n{2,}/).map((p) => "<p>" + p.replace(/\n/g, "<br>") + "</p>").join("");
}

// ---------- render ----------
function render() {
  if (!S) return;
  const p = myPlayer();
  if (!p) { // campaign switched or we were removed: rejoin with same token
    api("/api/join", { name: me.name || "Adventurer", token }).catch(() => {});
    return;
  }
  renderHeader();
  const ch = myChar();
  if (!ch) {
    if ($("#create").classList.contains("hidden")) openCreate();
    show("create");
    return;
  }
  show("game");
  renderLog();
  renderScene();
  renderParty();
  renderPlayers();
  renderCombat();
  renderSheet(ch);
  renderComposer(ch);
}

function renderHeader() {
  $("#hdr-campaign").textContent = S.name;
  $("#hdr-quest").textContent = S.quest ? "— " + S.quest : "";
  const dm = $("#hdr-dm");
  dm.className = "badge " + (S.ai ? "ai" : "offline");
  dm.textContent = S.ai ? "🧠 AI DM" : (S.mode === "ai" ? "🎲 Offline DM (no API key)" : "🎲 Offline DM");
  document.title = S.name + " — D&D Campaign Runner";
}

function renderScene() {
  $("#scene-loc").textContent = "📍 " + (S.location || "Somewhere");
  let d = S.sceneDesc || "";
  if (S.adventure) d = `Chapter ${S.adventure.chapter}: ${S.adventure.title} — area ${S.adventure.index + 1} of ${S.adventure.rooms}`;
  $("#scene-desc").textContent = d;
  $("#thinking").classList.toggle("hidden", !S.dmBusy);
}

function renderLog() {
  const log = $("#log");
  const nearBottom = log.scrollHeight - log.scrollTop - log.clientHeight < 120;
  if (S.id !== lastCampaign) {
    log.innerHTML = "";
    lastLogId = 0;
    lastCampaign = S.id;
  }
  const frag = document.createDocumentFragment();
  for (const e of S.log) {
    if (e.id <= lastLogId) continue;
    lastLogId = e.id;
    const div = document.createElement("div");
    div.className = "entry " + e.kind;
    const t = new Date(e.time).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    if (e.kind === "dm") div.innerHTML = md(e.text);
    else if (e.kind === "action") div.innerHTML = `<span class="who">${esc(e.who)}</span> <span class="txt">${esc(e.text)}</span><span class="time">${t}</span>`;
    else if (e.kind === "ooc") div.innerHTML = `💬 <span class="who">${esc(e.who)}:</span> ${esc(e.text)}<span class="time">${t}</span>`;
    else div.innerHTML = md(e.text).replace(/^<p>|<\/p>$/g, "");
    frag.appendChild(div);
  }
  if (frag.childNodes.length) {
    log.appendChild(frag);
    if (nearBottom || log.dataset.init !== "1") log.scrollTop = log.scrollHeight;
    log.dataset.init = "1";
  }
}

function pcCard(c, turnId) {
  const owner = S.players.find((p) => p.id === c.owner);
  const tags = [];
  if (c.dead) tags.push('<span class="tag r">dead</span>');
  else if (c.hp <= 0) tags.push(`<span class="tag r">${c.stable ? "stable" : "dying " + c.deathSaves + "✔ " + c.deathFails + "✘"}</span>`);
  if (c.raging) tags.push('<span class="tag r">raging</span>');
  if (c.dodging) tags.push('<span class="tag g">dodging</span>');
  if (c.smiting) tags.push('<span class="tag p">smite ready</span>');
  if (c.fled) tags.push('<span class="tag">fled</span>');
  if (c.slotsMax) tags.push(`<span class="tag p">slots ${c.slots}/${c.slotsMax}</span>`);
  const cls = ["pc", c.owner === me.playerId ? "me" : "", turnId === c.id ? "turn" : "", c.dead ? "dead" : c.hp <= 0 ? "down" : ""].join(" ");
  return `<div class="${cls}">
    <div class="row"><span class="nm grow">${esc(c.name)}</span><span class="small">AC ${c.ac}</span></div>
    <div class="sub">L${c.level} ${esc(raceName(c.race))} ${esc(classDef(c.class)?.name)} · ${esc(owner?.name || "?")}</div>
    <div class="bar ${hpClass(c.hp, c.maxHp)}"><i style="width:${Math.max(0, (100 * c.hp) / c.maxHp)}%"></i></div>
    <div class="row small"><span class="grow">HP ${c.hp}/${c.maxHp}</span><span class="muted">${c.gold} gp</span></div>
    <div class="tags">${tags.join("")}</div>
  </div>`;
}

function renderParty() {
  const cur = currentTurn();
  $("#party").innerHTML = S.characters.map((c) => pcCard(c, cur?.id)).join("") || '<p class="muted">No heroes yet.</p>';
}

function renderPlayers() {
  $("#players").innerHTML = S.players.map((p) => {
    const c = charById(p.charId);
    return `<div class="row"><span class="dot ${p.online ? "on" : ""}"></span><span class="grow">${esc(p.name)}</span><span class="muted">${c ? esc(c.name) : "making a hero…"}</span></div>`;
  }).join("");
}

function renderCombat() {
  const box = $("#combat");
  const cb = S.combat;
  box.classList.toggle("hidden", !cb);
  if (!cb) { target = ""; return; }
  const cur = currentTurn();
  const ch = myChar();
  const mine = cur && ch && cur.id === ch.id;
  const alive = cb.monsters.filter((m) => m.hp > 0);
  if (!alive.find((m) => m.id === target)) target = alive[0]?.id || "";

  const order = cb.order.map((o, i) => {
    let gone = false;
    if (o.isMonster) gone = (cb.monsters.find((m) => m.id === o.id)?.hp ?? 0) <= 0;
    else { const c = charById(o.id); gone = !c || c.dead || c.fled; }
    return `<li class="${i === cb.turn ? "cur" : ""} ${gone ? "gone" : ""}"><span class="n">${o.init}</span>${o.isMonster ? "👹" : "🛡️"} ${esc(o.name)}</li>`;
  }).join("");

  const foes = cb.monsters.map((m) => `<div class="foe ${m.hp <= 0 ? "dead" : ""} ${m.id === target ? "sel" : ""}" data-id="${m.id}">
      <div class="row"><b class="grow">${esc(m.name)}</b><span class="small muted">AC ${m.ac}</span></div>
      <div class="bar"><i style="width:${(100 * m.hp) / m.maxHp}%"></i></div>
      <div class="small muted">${m.hp <= 0 ? "defeated" : foeState(m)}</div>
    </div>`).join("");

  let actions = "";
  if (mine) {
    const w = (ch.inventory.find((i) => i.equipped && i.kind === "weapon") || { name: "Fists" }).name;
    const abil = ch.features.map((f) => DATA.abilities[f]).filter((a) => a && !a.passive && available(ch, a));
    const potions = ch.inventory.map((it, i) => ({ it, i })).filter((x) => x.it.kind === "potion");
    const allies = S.characters.filter((c) => !c.dead);
    if (!allies.find((c) => c.id === allyTarget)) allyTarget = ch.id;
    actions = `<div class="yourturn">⚔ Your turn, ${esc(ch.name)}!</div>
      <div class="small muted">Action ${cb.actionUsed && !cb.extraAction ? "used" : "ready"} · Bonus action ${cb.bonusUsed ? "used" : "ready"}. Click an enemy to target it.</div>
      <div class="actions">
        <button class="btn primary wide" data-c="attack">🗡️ Attack with ${esc(w)}</button>
        ${abil.map((a) => `<button class="btn" data-ab="${a.id}" title="${esc(a.desc)}">${abilIcon(a)} ${esc(a.name)}${abilCost(ch, a)}</button>`).join("")}
        ${potions.map((x) => `<button class="btn" data-potion="${x.i}">🧪 ${esc(x.it.name)}${x.it.qty > 1 ? " ×" + x.it.qty : ""}</button>`).join("")}
        <button class="btn" data-c="dodge">🛡️ Dodge</button>
        <button class="btn" data-c="flee">🏃 Flee</button>
        <button class="btn wide" data-c="end">⏭ End turn</button>
      </div>
      <label>Heal / help target</label>
      <select id="ally">${allies.map((c) => `<option value="${c.id}" ${c.id === allyTarget ? "selected" : ""}>${esc(c.name)} (${c.hp}/${c.maxHp})</option>`).join("")}</select>`;
  } else if (cur) {
    actions = `<p class="muted small">Waiting for <b>${esc(cur.name)}</b>…</p>`;
    if (me.isHost && !cur.isMonster) actions += `<button class="btn small" id="c-skip">Skip their turn</button>`;
  }
  box.innerHTML = `<h2>⚔ Combat — Round ${cb.round}</h2><ul class="init">${order}</ul>${foes}${actions}`;

  $$(".foe", box).forEach((el) => el.onclick = () => { if (!el.classList.contains("dead")) { target = el.dataset.id; renderCombat(); } });
  $$("[data-c]", box).forEach((b) => b.onclick = () => act("/api/combat", { type: b.dataset.c, target }));
  $$("[data-ab]", box).forEach((b) => b.onclick = () => {
    const a = DATA.abilities[b.dataset.ab];
    const t = a.target === "ally" ? allyTarget : a.target === "enemy" ? target : ch.id;
    act("/api/combat", { type: "ability", ability: a.id, target: t });
  });
  $$("[data-potion]", box).forEach((b) => b.onclick = () => act("/api/combat", { type: "item", item: +b.dataset.potion, target: allyTarget || ch.id }));
  const ally = $("#ally", box);
  if (ally) ally.onchange = () => (allyTarget = ally.value);
  const skip = $("#c-skip", box);
  if (skip) skip.onclick = () => act("/api/host/skip");
}

function foeState(m) {
  const r = m.hp / m.maxHp;
  return r >= 1 ? "unhurt" : r > 0.6 ? "lightly wounded" : r > 0.3 ? "bloodied" : "near death";
}

function abilIcon(a) {
  return { fire_bolt: "🔥", magic_missile: "✨", sacred_flame: "☀️", cure_wounds: "💚", healing_word: "💚", rage: "😡", second_wind: "💪", action_surge: "⚡", hunters_mark: "🎯", vicious_mockery: "🎭", lay_on_hands: "🙏", divine_smite: "⚜️" }[a.id] || "✦";
}

function available(ch, a) {
  return !(a.id === "action_surge" && ch.level < 2);
}

function abilCost(ch, a) {
  if (a.cost === "slot") return ` (${ch.slots})`;
  if (a.cost === "use") return ` (${ch.uses?.[a.id] ?? 0})`;
  return "";
}

function renderSheet(ch) {
  const cl = classDef(ch.class);
  const prof = 2 + Math.floor((ch.level - 1) / 4);
  const next = DATA.xp[ch.level] ?? ch.xp;
  const prev = DATA.xp[ch.level - 1] ?? 0;
  const xpPct = next > prev ? Math.min(100, (100 * (ch.xp - prev)) / (next - prev)) : 100;
  const skills = Object.keys(DATA.skills).sort().map((s) => {
    const p = ch.skills.includes(s) || (ch.race === "elf" && s === "perception");
    const v = mod(ch.stats[DATA.skills[s]]) + (p ? prof : 0);
    return `<div class="${p ? "prof" : ""}">${p ? "●" : "○"} ${s.replace(/_/g, " ")} ${sign(v)}</div>`;
  }).join("");
  const others = S.characters.filter((c) => c.id !== ch.id && !c.dead);
  const inv = ch.inventory.map((it, i) => {
    const btns = [];
    if (["weapon", "armor", "shield"].includes(it.kind)) btns.push(`<button class="btn" data-eq="${i}">${it.equipped ? "Unequip" : "Equip"}</button>`);
    if (it.kind === "potion") btns.push(`<button class="btn" data-use="${i}">Drink</button>`);
    if (others.length && !it.equipped) btns.push(`<button class="btn" data-give="${i}" title="Give to a party member">Give</button>`);
    const info = it.damage ? `${it.damage} ${it.dmgType}` : it.heal ? `heals ${it.heal}` : it.kind === "armor" || it.kind === "shield" ? `AC ${it.baseAC}${it.kind === "shield" ? "+" : ""}` : it.desc || "";
    return `<li title="${esc(it.desc || "")}"><span>${esc(it.name)}${it.qty > 1 ? " ×" + it.qty : ""} ${it.equipped ? '<span class="eq">EQUIPPED</span>' : ""}<br><span class="muted small">${esc(info)}</span></span><span class="acts">${btns.join("")}</span></li>`;
  }).join("");
  const abil = ch.features.map((f) => DATA.abilities[f]).filter((a) => a && available(ch, a));
  const outOfCombat = !S.combat && !ch.dead && ch.hp > 0;
  $("#sheet").innerHTML = `
    <div class="sheet-head"><div><div class="nm">${esc(ch.name)}</div>
      <div class="muted small">Level ${ch.level} ${esc(raceName(ch.race))} ${esc(cl?.name)}</div></div>
      <button class="btn small ghost" id="btn-retire" title="Retire this hero and make a new one">⟲</button></div>
    <div class="big-stats">
      <div><b>${ch.hp}/${ch.maxHp}</b><span>Hit points</span></div>
      <div><b>${ch.ac}</b><span>Armor class</span></div>
      <div><b>${sign(prof)}</b><span>Proficiency</span></div>
    </div>
    <div class="row small"><span class="grow">XP ${ch.xp}${ch.level < 12 ? " / " + next : ""}</span><span>💰 ${ch.gold} gp</span>${ch.slotsMax ? `<span>✨ ${ch.slots}/${ch.slotsMax} slots</span>` : ""}</div>
    <div class="xpbar"><i style="width:${xpPct}%"></i></div>
    <div class="mini-stats" style="margin-top:8px">${DATA.abilityOrder.map((a) => `<div><div class="ab">${a}</div><div class="v">${ch.stats[a]}</div><div class="small muted">${sign(mod(ch.stats[a]))}</div></div>`).join("")}</div>
    <details open><summary>Abilities</summary>
      ${abil.map((a) => `<div class="small" style="margin-bottom:6px"><b>${abilIcon(a)} ${esc(a.name)}</b>${abilCost(ch, a)} — <span class="muted">${esc(a.desc)}</span>
        ${outOfCombat && (a.target === "ally" || a.id === "second_wind") ? `<button class="btn small" data-oab="${a.id}">Use</button>` : ""}</div>`).join("")}
    </details>
    <details open><summary>Inventory</summary><ul class="inv">${inv}</ul></details>
    <details><summary>Skills</summary><div class="skills">${skills}</div></details>
    ${ch.backstory ? `<details><summary>Backstory</summary><p class="small muted">${esc(ch.backstory)}</p></details>` : ""}
    ${ch.dead ? `<p class="small" style="color:var(--red)">${esc(ch.name)} has died. Retire them (⟲) to roll a new hero.</p>` : ""}`;

  $$("[data-eq]").forEach((b) => b.onclick = () => act("/api/equip", { index: +b.dataset.eq }));
  $$("[data-use]").forEach((b) => b.onclick = () => act("/api/use", { index: +b.dataset.use, target: ch.id }));
  $$("[data-give]").forEach((b) => b.onclick = () => {
    const names = others.map((c, i) => `${i + 1}. ${c.name}`).join("\n");
    const n = prompt("Give to which party member?\n" + names, "1");
    const c = others[(+n || 0) - 1];
    if (c) act("/api/give", { index: +b.dataset.give, to: c.id });
  });
  $$("[data-oab]").forEach((b) => b.onclick = () => {
    const a = DATA.abilities[b.dataset.oab];
    let t = ch.id;
    if (a.target === "ally" && others.length) {
      const all = [ch, ...others];
      const n = prompt("Who?\n" + all.map((c, i) => `${i + 1}. ${c.name} (${c.hp}/${c.maxHp})`).join("\n"), "1");
      const c = all[(+n || 0) - 1];
      if (!c) return;
      t = c.id;
    }
    act("/api/ability", { ability: a.id, target: t });
  });
  $("#btn-retire").onclick = () => {
    if (confirm(`Retire ${ch.name}? They leave the party for good and you'll create a new character.`)) act("/api/retire", { charId: ch.id });
  };
}

function renderComposer(ch) {
  const hint = $("#start-hint");
  if (!S.started) {
    hint.classList.remove("hidden");
    hint.textContent = me.isHost ? "When everyone has a character, open ⚙ Host and press Begin adventure." : "Waiting for the host to begin the adventure… (Table talk works now.)";
  } else hint.classList.add("hidden");
  $("#msg").placeholder = mode === "act" ? (S.combat ? "Try something other than attacking (talk, trick, look around)…" : "What does " + ch.name + " do? (e.g. \"I search the altar for hidden compartments\")") : "Talk to the table out of character…";
}

$$(".tab").forEach((t) => t.onclick = () => {
  $$(".tab").forEach((x) => x.classList.remove("sel"));
  t.classList.add("sel");
  mode = t.dataset.mode;
  if (S) renderComposer(myChar() || { name: "your character" });
  $("#msg").focus();
});

$("#btn-send").onclick = send;
$("#msg").onkeydown = (e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); send(); } };
async function send() {
  const box = $("#msg");
  const text = box.value.trim();
  if (!text) return;
  const ok = await act(mode === "act" ? "/api/act" : "/api/ooc", { text });
  if (ok) box.value = "";
  box.focus();
}

function buildDice() {
  $("#dice").innerHTML = ["d4", "d6", "d8", "d10", "d12", "d20", "d100"].map((d) => `<button class="btn" data-d="1${d}">${d}</button>`).join("");
  $$("#dice [data-d]").forEach((b) => b.onclick = () => act("/api/roll", { dice: b.dataset.d }));
}

// ---------- character creation ----------
let cr = { race: "human", class: "fighter", method: "standard", base: [] , swap: null };

function openCreate() {
  $("#create-who").textContent = "Player: " + (me.name || "");
  $("#cr-races").innerHTML = DATA.races.map((r) => `<div class="choice" data-race="${r.id}"><b>${esc(r.name)}</b>
    <span class="small">${Object.entries(r.bonus).map(([k, v]) => k + " " + sign(v)).join(", ")}</span><div class="small">${esc(r.traits)}</div></div>`).join("");
  $("#cr-classes").innerHTML = DATA.classes.map((c) => `<div class="choice" data-class="${c.id}"><b>${esc(c.name)}</b>
    <span class="small">d${c.hitDie} HP · ${c.priority[0]}</span><div class="small">${esc(c.blurb)}</div></div>`).join("");
  $$("[data-race]").forEach((el) => el.onclick = () => { cr.race = el.dataset.race; drawCreate(); });
  $$("[data-class]").forEach((el) => el.onclick = () => { cr.class = el.dataset.class; autoAssign(); drawCreate(); });
  setMethod("standard", DATA.standardArray);
}

function setMethod(m, vals) {
  cr.method = m;
  cr.values = [...vals].sort((a, b) => b - a);
  autoAssign();
  drawCreate();
}

function autoAssign() {
  const pr = classDef(cr.class).priority;
  cr.base = {};
  pr.forEach((ab, i) => (cr.base[ab] = cr.values[i]));
  cr.swap = null;
}

function drawCreate() {
  $$("[data-race]").forEach((el) => el.classList.toggle("sel", el.dataset.race === cr.race));
  $$("[data-class]").forEach((el) => el.classList.toggle("sel", el.dataset.class === cr.class));
  const race = DATA.races.find((r) => r.id === cr.race);
  const cl = classDef(cr.class);
  $("#cr-class-info").innerHTML = `<b>${esc(cl.name)} abilities:</b> ` + cl.abilities.map((a) => `${esc(a.name)} — ${esc(a.desc)}`).join(" · ") + `<br><b>Skills:</b> ${cl.skills.map((s) => s.replace(/_/g, " ")).join(", ")}`;
  $("#cr-stats").innerHTML = DATA.abilityOrder.map((ab) => {
    const b = race.bonus[ab] || 0;
    const tot = cr.base[ab] + b;
    return `<div class="stat ${cr.swap === ab ? "sel" : ""}" data-ab="${ab}"><div class="ab">${ab}</div><div class="val">${tot}</div>
      <div class="mod">${sign(mod(tot))}</div><div class="bon">${b ? "base " + cr.base[ab] + " " + sign(b) : "base " + cr.base[ab]}</div></div>`;
  }).join("");
  $$("#cr-stats .stat").forEach((el) => el.onclick = () => {
    const ab = el.dataset.ab;
    if (!cr.swap) cr.swap = ab;
    else if (cr.swap === ab) cr.swap = null;
    else { [cr.base[ab], cr.base[cr.swap]] = [cr.base[cr.swap], cr.base[ab]]; cr.swap = null; }
    drawCreate();
  });
  const con = cr.base.CON + (race.bonus.CON || 0);
  const hp = Math.max(1, cl.hitDie + mod(con) + (cr.race === "dwarf" ? 1 : 0));
  $("#cr-preview").textContent = `Starting HP ${hp} · ${cr.method === "roll" ? "rolled scores" : "standard array"}`;
}

$("#cr-std").onclick = () => setMethod("standard", DATA.standardArray);
$("#cr-roll").onclick = async () => {
  try {
    const r = await api("/api/rollstats", {});
    setMethod("roll", r.stats);
  } catch (e) { toast(e.message); }
};

$("#btn-create").onclick = async () => {
  const name = $("#cr-name").value.trim();
  if (!name) return toast("Give your character a name");
  const ok = await act("/api/character", { name, race: cr.race, class: cr.class, method: cr.method, stats: cr.base, backstory: $("#cr-back").value });
  if (ok) { $("#cr-name").value = ""; $("#cr-back").value = ""; }
};

// ---------- host ----------
$("#btn-host").onclick = openHost;
$("#host-close").onclick = () => $("#host-modal").classList.add("hidden");
$("#host-modal").onclick = (e) => { if (e.target.id === "host-modal") $("#host-modal").classList.add("hidden"); };

async function openHost() {
  $("#host-modal").classList.remove("hidden");
  try {
    const info = await api("/api/host/info");
    $("#host-links").innerHTML = info.urls.length
      ? "Friends on the same Wi-Fi/network open: " + info.urls.map((u) => `<code>${esc(u)}</code>`).join(" ") + `<br><span class="muted">Playing over the internet? See the README (Tailscale / port forwarding).</span>`
      : "No network address found — only this computer can join.";
    $("#h-keyhint").textContent = info.hasKey ? `(saved ${info.keyHint})` : "(none saved)";
    $("#h-model").value = info.model;
    if (!$("#h-model").value) { const o = new Option(info.model, info.model); $("#h-model").add(o); $("#h-model").value = info.model; }
    $("#h-effort").value = info.effort;
  } catch (e) { toast(e.message); }
  if (S) {
    $("#h-mode").value = S.mode;
    $("#h-setting").value = S.setting || "";
    $("#h-start").disabled = S.started;
  }
  loadCamps();
}

async function loadCamps() {
  try {
    const list = await api("/api/host/campaigns");
    $("#h-camps").innerHTML = list.map((c) => `<div class="camp ${c.current ? "cur" : ""}"><div class="grow"><b>${esc(c.name)}</b> <span class="muted small">${c.mode === "ai" ? "🧠" : "🎲"} ${new Date(c.modified).toLocaleString()}</span>
      <div class="small muted">${esc((c.heroes || []).join(", ") || "no heroes yet")}</div></div>
      ${c.current ? '<span class="small">current</span>' : `<button class="btn small" data-load="${c.id}">Load</button><button class="btn small danger" data-del="${c.id}">Delete</button>`}</div>`).join("");
    $$("[data-load]").forEach((b) => b.onclick = async () => { if (await act("/api/host/load", { id: b.dataset.load }, "Campaign loaded")) { resetView(); loadCamps(); } });
    $$("[data-del]").forEach((b) => b.onclick = async () => { if (confirm("Delete this campaign forever?") && await act("/api/host/delete", { id: b.dataset.del })) loadCamps(); });
  } catch (e) { toast(e.message); }
}

function resetView() { lastLogId = 0; lastCampaign = ""; $("#log").innerHTML = ""; }

$("#h-start").onclick = async () => { if (await act("/api/host/start", {}, "The adventure begins!")) $("#host-modal").classList.add("hidden"); };
$("#h-skip").onclick = () => act("/api/host/skip", {});
$("#h-short").onclick = () => act("/api/host/rest", { kind: "short" });
$("#h-long").onclick = () => act("/api/host/rest", { kind: "long" });
$("#h-save").onclick = async () => {
  const body = { mode: $("#h-mode").value, model: $("#h-model").value, effort: $("#h-effort").value, setting: $("#h-setting").value };
  const k = $("#h-key").value.trim();
  if (k) body.apiKey = k;
  if (await act("/api/host/settings", body, "Settings saved")) { $("#h-key").value = ""; openHost(); }
};
$("#h-new").onclick = async () => {
  const body = { name: $("#h-new-name").value, setting: $("#h-new-setting").value, mode: $("#h-new-mode").value };
  if (await act("/api/host/new", body, "New campaign created")) { resetView(); $("#host-modal").classList.add("hidden"); }
};

boot().catch((e) => toast("Couldn't reach the game server: " + e.message));
