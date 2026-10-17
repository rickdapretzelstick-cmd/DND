// Route finding for the on-screen guide: A* over coarse cells, then line-of-sight smoothing.
import { WALKABLE, T } from "./pixel.js";

const CELL = 4;
const ROAD = new Set([T.PAVED, T.PATH, T.GRAVEL, T.SIDEWALK, T.PLAZA, T.DIRT, T.BRIDGE]);

export function findPath(at, W, H, sx, sy, tx, ty, maxNodes = 60000) {
  const cw = Math.ceil(W / CELL), ch = Math.ceil(H / CELL);
  const cellOk = (cx, cy) => {
    if (cx < 0 || cy < 0 || cx >= cw || cy >= ch) return false;
    const x = cx * CELL + CELL / 2, y = cy * CELL + CELL / 2;
    return WALKABLE.has(at(Math.floor(x), Math.floor(y))) && WALKABLE.has(at(Math.floor(x - 1), Math.floor(y - 1))) && WALKABLE.has(at(Math.floor(x + 1), Math.floor(y + 1)));
  };
  const cost = (cx, cy) => (ROAD.has(at(cx * CELL + 2, cy * CELL + 2)) ? 0.65 : 1);
  const nearest = (x, y) => {
    const cx = Math.floor(x / CELL), cy = Math.floor(y / CELL);
    if (cellOk(cx, cy)) return [cx, cy];
    for (let r = 1; r < 14; r++) for (let dy = -r; dy <= r; dy++) for (let dx = -r; dx <= r; dx++) {
      if (Math.max(Math.abs(dx), Math.abs(dy)) === r && cellOk(cx + dx, cy + dy)) return [cx + dx, cy + dy];
    }
    return null;
  };
  const s = nearest(sx, sy), g = nearest(tx, ty);
  if (!s || !g) return null;
  const key = (cx, cy) => cy * cw + cx;
  const gs = new Map([[key(...s), 0]]), from = new Map(), heap = [[0, s[0], s[1]]];
  const push = (n) => { heap.push(n); let i = heap.length - 1; while (i > 0) { const p = (i - 1) >> 1; if (heap[p][0] <= heap[i][0]) break; [heap[p], heap[i]] = [heap[i], heap[p]]; i = p; } };
  const pop = () => { const top = heap[0], last = heap.pop(); if (heap.length) { heap[0] = last; let i = 0; for (;;) { let l = 2 * i + 1, r = l + 1, m = i; if (l < heap.length && heap[l][0] < heap[m][0]) m = l; if (r < heap.length && heap[r][0] < heap[m][0]) m = r; if (m === i) break; [heap[m], heap[i]] = [heap[i], heap[m]]; i = m; } } return top; };
  let nodes = 0, found = false;
  const h = (cx, cy) => Math.hypot(cx - g[0], cy - g[1]) * 0.7;
  while (heap.length && nodes++ < maxNodes) {
    const [, cx, cy] = pop();
    if (cx === g[0] && cy === g[1]) { found = true; break; }
    const base = gs.get(key(cx, cy));
    for (let dy = -1; dy <= 1; dy++) for (let dx = -1; dx <= 1; dx++) {
      if (!dx && !dy) continue;
      const nx = cx + dx, ny = cy + dy;
      if (!cellOk(nx, ny)) continue;
      if (dx && dy && (!cellOk(cx + dx, cy) || !cellOk(cx, cy + dy))) continue;
      const ng = base + (dx && dy ? 1.414 : 1) * cost(nx, ny);
      const k = key(nx, ny);
      if (ng < (gs.get(k) ?? 1e18)) { gs.set(k, ng); from.set(k, [cx, cy]); push([ng + h(nx, ny), nx, ny]); }
    }
  }
  if (!found) return null;
  const cells = [];
  for (let c = g; c; c = from.get(key(...c))) { cells.push(c); if (c[0] === s[0] && c[1] === s[1]) break; }
  cells.reverse();
  const pts = [[sx, sy], ...cells.map(([cx, cy]) => [cx * CELL + CELL / 2, cy * CELL + CELL / 2]), [tx, ty]];
  // line-of-sight smoothing
  const clear = (a, b) => {
    const d = Math.hypot(b[0] - a[0], b[1] - a[1]), n = Math.ceil(d / 1.5);
    for (let i = 1; i < n; i++) { const t = i / n; if (!WALKABLE.has(at(Math.floor(a[0] + (b[0] - a[0]) * t), Math.floor(a[1] + (b[1] - a[1]) * t)))) return false; }
    return true;
  };
  const out = [pts[0]];
  let i = 0;
  while (i < pts.length - 1) {
    let j = pts.length - 1;
    while (j > i + 1 && !clear(pts[i], pts[j])) j--;
    out.push(pts[j]); i = j;
  }
  return out;
}
