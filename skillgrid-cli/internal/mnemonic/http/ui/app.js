"use strict";
/* Dashboard shell router (P1). Path-routed menu entries (/welcome, /tracker,
 * …); the server serves the same shell for each path. The Welcome entry is
 * fully static (zero fetches). Data entries land in P2–P6; until then they
 * render as labeled disabled stubs. Project list loads lazily — never on the
 * Welcome entry. */
const $ = (s) => document.querySelector(s);

const STUBS = {
  code: { phase: "P5", title: "Code", body: "Index status, freshness banner, BM25 search with source view." },
  sessions: { phase: "P6", title: "Sessions", body: "Session list with titles, recent context, and summaries." },
};

const LIVE = ["welcome", "tracker", "docs", "memory"];

const ROUTES = ["welcome", "tracker", "docs", "memory", "code", "sessions", "swagger"];

let project = localStorage.getItem("sgmn-project") || "";
let projectsLoaded = false;

function currentRoute() {
  const seg = location.pathname.replace(/\/+$/, "").split("/").pop() || "welcome";
  if (seg === "swagger-ui") return "swagger";
  return ROUTES.includes(seg) ? seg : "welcome";
}

/* Swagger UI is embedded in the main-col content area (sidebar stays
 * visible). The bundle scripts are injected once on the swagger route;
 * SwaggerUIBundle mounts into #swagger-ui inside the injected node. */
let swaggerLoaded = false;
function loadSwagger(box) {
  if (!box.querySelector("#swagger-ui")) {
    box.innerHTML = `<div id="swagger-ui"></div>`;
  }
  if (swaggerLoaded) return;
  swaggerLoaded = true;
  ["swagger-ui-bundle.js", "swagger-ui-standalone-preset.js", "swagger-initializer.js"].forEach((f) => {
    const s = document.createElement("script");
    s.src = `/swagger-ui/${f}`;
    document.head.appendChild(s);
  });
}

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[c]));
}

async function api(url) {
  const r = await fetch(url, { headers: { Accept: "application/json" } });
  const text = await r.text();
  let body = null;
  try { body = JSON.parse(text); } catch { body = text; }
  if (!r.ok) throw new Error(body && body.error ? body.error : r.status + " " + r.statusText);
  return body;
}

function render() {
  const route = currentRoute();
  document.querySelectorAll("#menu .nav-link[data-route]").forEach((a) => {
    if (a.dataset.route === route) a.setAttribute("aria-current", "page");
    else a.removeAttribute("aria-current");
  });
  $("#crumb-current").textContent = route;
  const welcome = $("#view-welcome");
  const stub = $("#view-stub");
  const swagger = $("#view-swagger");
  // Hide every view first; each branch shows only its own. Without this, the
  // Swagger mount lingers at the end of other entries after a visit.
  welcome.hidden = true;
  stub.hidden = true;
  swagger.hidden = true;
  if (route === "swagger") {
    swagger.hidden = false;
    loadSwagger(swagger);
    return;
  }
  if (route === "welcome") {
    welcome.hidden = false;
    return;
  }
  // Future entries progressively go live in P3–P6; until then, stubs.
  if (!LIVE.includes(route)) {
    const info = STUBS[route];
    stub.hidden = false;
    stub.innerHTML =
      `<div class="stub-card"><span class="phase-id next">${esc(info.phase)}</span>` +
      `<h2>${esc(info.title)}</h2><p>${esc(info.body)}</p>` +
      `<p>Coming in ${esc(info.phase)} — not built yet.</p></div>`;
    return;
  }
  if (route === "tracker") {
    stub.hidden = false;
    // Defer to a microtask so the initial load runs AFTER the top-level
    // tracker consts (TRK_PROVIDERS/…) are initialized — renderTracker's
    // synchronous prefix would otherwise throw a TDZ ReferenceError.
    Promise.resolve().then(() => renderTracker(stub));
    // Project context matters only on data entries; load it lazily here.
    ensureProjects();
    return;
  }
  if (route === "docs") {
    stub.hidden = false;
    stub.hidden = false;
    renderDocs(stub);
    return;
  }
  if (route === "memory") {
    stub.hidden = false;
    // Defer to a microtask so the initial load runs AFTER the top-level
    // memory consts (MEM_VISIBILITIES/…) are initialized — renderMemory's
    // synchronous prefix would otherwise throw a TDZ ReferenceError.
    Promise.resolve().then(() => renderMemory(stub));
    ensureProjects();
    return;
  }
}

async function ensureProjects() {
  if (projectsLoaded) return;
  try {
    const r = await api("/projects");
    const ids = r.projects || [];
    const box = $("#project");
    box.innerHTML = "";
    if (!ids.length) {
      const o = document.createElement("option");
      o.textContent = "(no project stores found)";
      box.appendChild(o);
    } else {
      for (const id of ids) {
        const o = document.createElement("option");
        o.value = id;
        o.textContent = id;
        if (id === project) o.selected = true;
        box.appendChild(o);
      }
      if (!project || !ids.includes(project)) {
        project = ids[0];
        localStorage.setItem("sgmn-project", project);
      }
      box.value = project;
    }
    box.hidden = false;
    projectsLoaded = true;
  } catch {
    /* shell stays usable without the project list; retry on next entry */
  }
}

$("#project").addEventListener("change", (e) => {
  project = e.target.value;
  localStorage.setItem("sgmn-project", project);
});

document.querySelectorAll("#menu .nav-link[data-route]").forEach((a) => {
  a.addEventListener("click", (e) => {
    e.preventDefault();
    history.pushState(null, "", a.getAttribute("href"));
    render();
  });
});
window.addEventListener("popstate", render);
render();


/* Tracker entry (P2) — vanilla port of the skillgrid-ui BacklogView:
 * provider switcher, search + priority filter, fixed 4-column board,
 * rich cards, slide-over detail. Backend normalizes every provider into one
 * DTO with a canonical `board` column (demo: STATUS_COLUMNS). */
const TRK_PROVIDERS = [
  { id: "backlogmd", label: "Backlog.md", source: "Local backlog CLI (--json list/view, text config)" },
  { id: "jira", label: "Jira", source: "jira CLI (JQL list, view, move)" },
  { id: "gitlab", label: "GitLab", source: "glab CLI (JSON list/view, close/reopen)" },
  { id: "github", label: "GitHub", source: "gh CLI (JSON list/view, close/reopen)" },
];
const TRK_COLUMNS = [
  { id: "todo", label: "To Do" },
  { id: "in_progress", label: "In Progress" },
  { id: "blocked", label: "Blocked" },
  { id: "done", label: "Done" },
];
const TRK_PRIORITIES = ["all", "critical", "high", "medium", "low"];
const TRK_VIEWS = ["board", "list"];

let trk = {
  provider: "backlogmd", query: "", priority: "all",
  config: null, tasks: [], connected: {},
  view: "board", drag: null,
};

// boardStatus maps a provider-native status to its canonical board column so a
// drop can be reverse-mapped back to a provider status (the POST body).
function boardStatus(colId, statuses) {
  if (!statuses || !statuses.length) return colId;
  for (const s of statuses) {
    if (boardFor(s) === colId) return s;
  }
  return { in_progress: "In Progress", done: "Done", blocked: "Blocked", todo: "To Do" }[colId] || colId;
}

function boardFor(status) {
  const s = String(status || "").toLowerCase();
  if (s.includes("block")) return "blocked";
  if (s.includes("progress") || s.includes("doing") || s.includes("review") || s.includes("active")) return "in_progress";
  if (s.includes("done") || s.includes("close") || s.includes("complete") || s.includes("fix")) return "done";
  return "todo";
}

function trkInitials(name) {
  const parts = String(name || "").replace(/^@/, "").split(/[.\-_ ]+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return String(name || "").replace(/^@/, "").slice(0, 2).toUpperCase();
}

function trkRelative(iso) {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return "";
  const diff = Math.max(0, Date.now() - then);
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.floor(months / 12)}y ago`;
}

async function trkFetch(path) {
  const r = await fetch(path, { headers: { Accept: "application/json" } });
  const text = await r.text();
  let body = null;
  try { body = JSON.parse(text); } catch { body = { error: text }; }
  if (!r.ok) {
    const err = new Error((body && body.error) || `${r.status} ${r.statusText}`);
    err.status = r.status;
    err.provider = body && body.provider;
    throw err;
  }
  return body;
}

function trkQuery() {
  const q = new URLSearchParams(location.search);
  const p = (q.get("provider") || "").toLowerCase();
  const alias = { "backlog.md": "backlogmd", backlog: "backlogmd", gh: "github", glab: "gitlab" };
  const provider = TRK_PROVIDERS.some((x) => x.id === p) ? p : (alias[p] || "");
  const view = TRK_VIEWS.includes(q.get("view")) ? q.get("view") : "";
  return { provider, task: q.get("task") || "", view };
}

async function renderTracker(box) {
  // Probe every provider's config once so the switcher shows real
  // connected states (cheap reads; failures render as not-connected).
  const probes = await Promise.all(TRK_PROVIDERS.map(async (p) => {
    if (trk.connected[p.id] !== undefined) return;
    try {
      await trkFetch(`/tracker/config?provider=${encodeURIComponent(p.id)}`);
      trk.connected[p.id] = true;
    } catch {
      trk.connected[p.id] = false;
    }
  }));
  void probes;
  // trkLoad self-detects a fresh render (empty #trk-body) and reads the
  // initial provider/view/task state from the URL, so no state is threaded here.
  await trkLoad(box, null);
}

async function trkLoad(box, openTaskId, opts) {
  const keep = !!(opts && opts.keep);
  let body = box.querySelector("#trk-body");
  // Self-detect a fresh render: on the very first trkLoad the #trk-body is
  // still empty (renderTracker hasn't filled it), so (re)read the initial
  // state from the URL and lay down the skeleton. Subsequent loads (provider /
  // view / refresh / post-move) operate on the already-mounted board.
  if (!body) {
    const init = trkQuery();
    if (init.provider) {
      trk.provider = init.provider;
      trk.config = null;
      trk.tasks = [];
    }
    if (init.view) trk.view = init.view;
    openTaskId = openTaskId || init.task || null;
    box.innerHTML =
      `<div class="trk-page"><header class="trk-head"><h1>Tracker</h1>` +
      `<p class="muted">A visual board for your ticketing system. Drag tasks to change status, or click one for details.</p></header>` +
      `<div id="trk-body"><div class="trk-loading" role="status"><span class="spinner" aria-hidden="true"></span>Loading tracker…</div></div></div>`;
    body = box.querySelector("#trk-body");
  }
  const meta = TRK_PROVIDERS.find((p) => p.id === trk.provider);
  const switcher = TRK_PROVIDERS.map((p) => {
    const active = p.id === trk.provider;
    const dot = trk.connected[p.id] === false ? `<span class="trk-conn-dot" title="Not connected" aria-label="Not connected"></span>` : "";
    return `<button type="button" role="tab" aria-selected="${active}" data-provider="${p.id}"` +
      ` class="trk-sw${active ? " active" : ""}">${esc(p.label)}${dot}</button>`;
  }).join("");
  const viewTabs = TRK_VIEWS.map((v) =>
    `<button type="button" data-view="${v}" aria-pressed="${trk.view === v}" class="trk-view-btn${trk.view === v ? " active" : ""}">${v === "board" ? "Board" : "All"}</button>`).join("");
  let main;
  if (trk.connected[trk.provider] === false && !trk.config) {
    main = `<div class="trk-empty-state"><p class="trk-empty-title">${esc(meta.label)} is not connected</p>` +
      `<p class="muted">Configure the ${esc(meta.label)} adapter: ${esc(meta.source)}.</p></div>`;
  } else if (keep) {
    main = box.querySelector("#trk-main") ? box.querySelector("#trk-main").innerHTML : "";
  } else {
    try {
      const [cfg, data] = await Promise.all([
        trk.config || trkFetch(`/tracker/config?provider=${encodeURIComponent(trk.provider)}`),
        trkFetch(`/tracker/tasks?provider=${encodeURIComponent(trk.provider)}`),
      ]);
      trk.config = cfg;
      trk.tasks = data.tasks || [];
      main = trkMain();
    } catch (e) {
      main = `<div class="trk-empty-state"><p class="trk-empty-title">Failed to load tasks</p>` +
        `<p class="muted">${esc(e.message)}</p></div>`;
    }
  }
  body.innerHTML =
    `<div class="trk-top"><div class="trk-switcher" role="tablist" aria-label="Ticketing provider">${switcher}</div>` +
    `<p class="trk-source">${esc(meta.source)}</p></div>` +
    `<div class="trk-filters"><div class="trk-search-wrap">` +
    `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>` +
    `<input type="search" id="trk-q" value="${esc(trk.query)}" placeholder="Filter by title, id, label or assignee..." aria-label="Filter tasks"></div>` +
    `<div class="trk-prio" role="group" aria-label="Priority filter">` +
    TRK_PRIORITIES.map((p) => `<button type="button" data-prio="${p}" aria-pressed="${trk.priority === p}"` +
      ` class="trk-prio-btn${trk.priority === p ? " active" : ""}">${p}</button>`).join("") +
    `</div>` +
    `<div class="trk-viewswitch" role="group" aria-label="View">` +
    `<button type="button" id="trk-refresh" class="trk-refresh" title="Refresh" aria-label="Refresh">⟳</button>` +
    `<span class="trk-viewtabs">${viewTabs}</span></div>` +
    `</div>` +
    `<div class="trk-stats" id="trk-stats"></div>` +
    `<div id="trk-main">${main}</div><div id="trk-overlay"></div>`;
  body.querySelectorAll(".trk-sw").forEach((b) => b.addEventListener("click", () => {
    trk.provider = b.dataset.provider;
    trk.config = null;
    trk.tasks = [];
    trkSyncUrl(null);
    renderTracker(box);
  }));
  body.querySelectorAll(".trk-view-btn").forEach((b) => b.addEventListener("click", () => {
    trk.view = b.dataset.view;
    trkSyncUrl(null);
    trkRenderMain(box);
  }));
  const q = body.querySelector("#trk-q");
  q.addEventListener("input", () => {
    trk.query = q.value;
    trkRenderMain(box);
  });
  body.querySelectorAll(".trk-prio-btn").forEach((b) => b.addEventListener("click", () => {
    trk.priority = b.dataset.prio;
    body.querySelectorAll(".trk-prio-btn").forEach((x) => {
      const on = x === b;
      x.classList.toggle("active", on);
      x.setAttribute("aria-pressed", String(on));
    });
    trkRenderMain(box);
  }));
  body.querySelector("#trk-refresh").addEventListener("click", () => {
    trk.config = null;
    trkLoad(box, null);
  });
  trkRenderMain(box);
  if (openTaskId) {
    const match = trk.tasks.find((t) => t.id === openTaskId);
    if (match) trkOpenDetail(box, match);
  }
}

// trkRenderMain (re)draws the board or list from cached tasks — used by the
// search/priority/view filters without a network round trip.
function trkRenderMain(box) {
  const main = box.querySelector("#trk-main");
  const stats = box.querySelector("#trk-stats");
  if (!main) return;
  if (trk.connected[trk.provider] === false && !trk.config) {
    // not-connected state already rendered in trkLoad; nothing to redraw.
    return;
  }
  main.innerHTML = trkMain();
  if (stats) stats.innerHTML = trkStats();
  trkBindMain(box);
}

function trkMain() {
  return trk.view === "list" ? trkList() : trkBoard();
}

function trkStats() {
  const tasks = trkFiltered();
  const by = (b) => tasks.filter((t) => (t.board || "todo") === b).length;
  const total = trk.tasks.length;
  return TRK_COLUMNS.map((c) =>
    `<span class="trk-stat"><span class="trk-stat-dot st-${c.id}" aria-hidden="true"></span>${by(c.id)} <span class="trk-stat-label">${esc(c.label)}</span></span>`
  ).join("") + `<span class="trk-stat trk-stat-total"><span class="trk-stat-dot st-total"></span>${total} <span class="trk-stat-label">total</span></span>`;
}

function trkFiltered() {
  const q = trk.query.trim().toLowerCase();
  return trk.tasks.filter((t) => {
    if (trk.priority !== "all" && (t.priority || "").toLowerCase() !== trk.priority) return false;
    if (!q) return true;
    return (t.title || "").toLowerCase().includes(q) ||
      (t.id || "").toLowerCase().includes(q) ||
      (t.labels || []).some((l) => l.toLowerCase().includes(q)) ||
      (t.assignees || []).some((a) => String(a).toLowerCase().includes(q));
  });
}

function trkBoard() {
  const tasks = trkFiltered();
  return `<div class="trk-board" id="trk-board">` + TRK_COLUMNS.map((col) => {
    const cards = tasks.filter((t) => (t.board || "todo") === col.id).map(trkCard).join("");
    return `<section class="trk-col" data-col="${col.id}" aria-label="${esc(col.label)}">` +
      `<header><h2>${esc(col.label)}</h2><span class="trk-col-count">${tasks.filter((t) => (t.board || "todo") === col.id).length}</span></header>` +
      `<div class="trk-col-body" data-drop="${col.id}">` + (cards || `<p class="trk-col-empty">drop tasks here</p>`) + `</div></section>`;
  }).join("") + `</div>`;
}

function trkList() {
  const tasks = trkFiltered();
  if (!tasks.length) return `<div class="trk-empty-state"><p class="trk-empty-title">No tasks match the current filters</p></div>`;
  return `<div class="trk-list" id="trk-list">` + tasks.map(trkRow).join("") + `</div>`;
}

function trkRow(t) {
  const dots = { todo: "trk-dot-todo", in_progress: "trk-dot-prog", blocked: "trk-dot-block", done: "trk-dot-done" };
  const dot = dots[t.board] || dots.todo;
  const pri = (t.priority || "").toLowerCase();
  const badge = pri ? `<span class="trk-prio-badge pri-${esc(pri)}">${esc(t.priority)}</span>` : "";
  const labels = (t.labels || []).slice(0, 3).map((l) => `<span class="trk-chip">${esc(l)}</span>`).join("");
  const who = (t.assignees && t.assignees.length) ? `<span class="trk-avatar" title="${esc(t.assignees.join(", "))}">${esc(trkInitials(t.assignees[0]))}</span>` : "";
  const rel = trkRelative(t.updated_at);
  return `<div class="trk-row" data-id="${esc(t.id)}" role="button" tabindex="0">` +
    `<span class="trk-row-id"><span class="trk-dot ${dot}" aria-hidden="true"></span>${esc(t.id)}</span>` +
    `<span class="trk-row-title">${esc(t.title)}</span>` +
    (badge ? `<span class="trk-row-pri">${badge}</span>` : "") +
    (labels ? `<span class="trk-row-labels">${labels}</span>` : "") +
    `<span class="trk-row-foot">${who}${rel ? `<span class="trk-rel">${esc(rel)}</span>` : ""}</span>` +
    `</div>`;
}

function trkCard(t) {
  const labels = (t.labels || []).map((l) => `<span class="trk-chip">${esc(l)}</span>`).join("");
  const pri = (t.priority || "").toLowerCase();
  const badge = pri ? `<span class="trk-prio-badge pri-${esc(pri)}">${esc(t.priority)}</span>` : "";
  const ac = (t.ac_total && (t.ac_completed !== undefined && t.ac_completed !== null))
    ? `<span class="trk-ac" title="${t.ac_completed}/${t.ac_total} acceptance criteria">✓ ${t.ac_completed}/${t.ac_total}</span>` : "";
  const who = (t.assignees && t.assignees.length)
    ? `<span class="trk-avatar" title="${esc(t.assignees.join(", "))}">${esc(trkInitials(t.assignees[0]))}</span>`
    : `<span class="trk-unassigned">unassigned</span>`;
  const docs = (t.doc_refs && t.doc_refs.length)
    ? `<span class="trk-docrefs" title="${t.doc_refs.length} linked document(s)">▤ ${t.doc_refs.length}</span>` : "";
  const rel = trkRelative(t.updated_at);
  return `<button type="button" class="trk-card" draggable="true" data-id="${esc(t.id)}" data-board="${esc(t.board || "todo")}">` +
    `<span class="trk-card-top"><span class="trk-card-id">${esc(t.id)}</span>${badge}</span>` +
    `<span class="trk-card-title">${esc(t.title)}</span>` +
    (labels ? `<span class="trk-labels">${labels}</span>` : "") +
    `<span class="trk-card-foot"><span class="trk-foot-left">${who}${ac}${docs}</span>` +
    (rel ? `<span class="trk-rel">${esc(rel)}</span>` : "") + `</span></button>`;
}

function trkBindMain(box) {
  const body = box.querySelector("#trk-body");
  // Board cards: open on click, move on drag-and-drop.
  body.querySelectorAll(".trk-card").forEach((el) => {
    el.addEventListener("click", () => {
      const match = trk.tasks.find((t) => t.id === el.dataset.id);
      if (match) trkOpenDetail(box, match);
    });
    el.addEventListener("dragstart", (e) => {
      trk.drag = { id: el.dataset.id, from: el.dataset.board };
      el.classList.add("trk-dragging");
      e.dataTransfer.effectAllowed = "move";
      try { e.dataTransfer.setData("text/plain", el.dataset.id); } catch { /* IE-less */ }
    });
    el.addEventListener("dragend", (e) => {
      e.target.classList.remove("trk-dragging");
      trk.drag = null;
      trkClearDrop(box);
    });
  });
  // Columns: drop targets.
  body.querySelectorAll("[data-drop]").forEach((zone) => {
    zone.addEventListener("dragover", (e) => {
      if (!trk.drag) return;
      e.preventDefault();
      e.dataTransfer.dropEffect = "move";
      zone.classList.add("trk-dropover");
    });
    zone.addEventListener("dragleave", () => zone.classList.remove("trk-dropover"));
    zone.addEventListener("drop", async (e) => {
      e.preventDefault();
      zone.classList.remove("trk-dropover");
      const id = trk.drag ? trk.drag.id : (e.dataTransfer.getData("text/plain") || "");
      const target = zone.dataset.drop;
      const from = trk.drag ? trk.drag.from : null;
      trk.drag = null;
      if (!id || !target || target === from) return;
      await trkMove(box, id, target);
    });
  });
  // List rows: open on click / Enter.
  body.querySelectorAll(".trk-row").forEach((el) => {
    el.addEventListener("click", () => {
      const match = trk.tasks.find((t) => t.id === el.dataset.id);
      if (match) trkOpenDetail(box, match);
    });
    el.addEventListener("keydown", (e) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        const match = trk.tasks.find((t) => t.id === el.dataset.id);
        if (match) trkOpenDetail(box, match);
      }
    });
  });
}

function trkClearDrop(box) {
  box.querySelectorAll("[data-drop]").forEach((z) => z.classList.remove("trk-dropover"));
}

// trkMove changes a task's status via the provider-compatible status endpoint,
// then refetches the board (the provider is the source of truth for the new
// status/label, which may differ from a naive column map).
async function trkMove(box, id, targetCol) {
  const statuses = (trk.config && trk.config.statuses) || [];
  const status = boardStatus(targetCol, statuses);
  try {
    await trackerFetch(`/tracker/tasks/${encodeURIComponent(id)}/status?provider=${encodeURIComponent(trk.provider)}`, {
      method: "POST",
      body: JSON.stringify({ status }),
    });
    trk.config = null;
    await trkLoad(box, null, { keep: true });
  } catch (e) {
    const ov = box.querySelector("#trk-overlay");
    if (ov) {
      ov.innerHTML = `<div class="trk-toast" role="alert">Failed to move: ${esc(e.message)}</div>`;
      setTimeout(() => { if (ov.firstChild) ov.firstChild.remove(); }, 3500);
    }
  }
}

function trkSyncUrl(taskId) {
  const q = new URLSearchParams(location.search);
  q.set("provider", trk.provider);
  q.set("view", trk.view);
  if (taskId) q.set("task", taskId);
  else q.delete("task");
  history.replaceState(null, "", `${location.pathname}?${q.toString()}`);
}

async function trkOpenDetail(box, seed) {
  const overlay = box.querySelector("#trk-overlay");
  trkSyncUrl(seed.id);
  // Fetch the full item (description, doc refs); fall back to the seed row.
  let it = seed;
  try {
    it = await trkFetch(`/tracker/tasks/${encodeURIComponent(seed.id)}?provider=${encodeURIComponent(trk.provider)}`);
  } catch { /* seed row stays */ }
  const cfg = trk.config || {};
  const statuses = cfg.statuses && cfg.statuses.length ? cfg.statuses : [it.status];
  const dots = { todo: "trk-dot-todo", in_progress: "trk-dot-prog", blocked: "trk-dot-block", done: "trk-dot-done" };
  const dot = dots[it.board] || dots.todo;
  const pri = (it.priority || "").toLowerCase();
  const who = (it.assignees && it.assignees.length)
    ? `<span class="trk-avatar">${esc(trkInitials(it.assignees[0]))}</span><span class="trk-mono">${esc(it.assignees.join(", "))}</span>`
    : `<span class="trk-unassigned">unassigned</span>`;
  const labels = (it.labels || []).map((l) => `<span class="trk-chip">${esc(l)}</span>`).join("");
  const docs = (it.doc_refs || []).map((d) =>
    `<li><span class="trk-docref">▤ ${trkDocLink(d)}</span></li>`).join("");
  const ac = (it.ac_total && (it.ac_completed !== undefined && it.ac_completed !== null))
    ? `<div class="trk-ac-card"><span class="trk-ac-label">Acceptance</span><span class="trk-ac-val">✓ ${it.ac_completed}/${it.ac_total}</span></div>` : "";
  const dates = `<div class="trk-dates"><span><strong>Created:</strong> ${esc(it.created_at || "—")}</span>` +
    `<span><strong>Updated:</strong> ${esc(it.updated_at || "—")}</span></div>`;
  overlay.innerHTML =
    `<div class="trk-modal" role="dialog" aria-modal="true" aria-label="Task detail">` +
    `<button type="button" class="trk-backdrop" aria-label="Close details" data-close></button>` +
    `<div class="trk-modal-card">` +
    `<header class="trk-modal-head">` +
    `<span class="trk-panel-id"><span class="trk-dot ${dot}" aria-hidden="true"></span><span class="trk-mono">${esc(it.id)}</span>` +
    (pri ? `<span class="trk-prio-badge pri-${esc(pri)}">${esc(it.priority)}</span>` : "") + `</span>` +
    `<span class="trk-modal-actions"><span class="muted" id="trk-msg"></span>` +
    `<button type="button" class="btn" id="trk-apply" type="button">Apply</button>` +
    `<select id="trk-status" class="project-select" aria-label="Move to status">` +
    statuses.map((s) => `<option value="${esc(s)}"${s === it.status ? " selected" : ""}>${esc(s)}</option>`).join("") +
    `</select>` +
    `<button type="button" class="trk-iconbtn" data-close aria-label="Close">✕</button></span>` +
    `</header>` +
    `<div class="trk-modal-grid">` +
    `<div class="trk-modal-main"><h2>${esc(it.title)}</h2>` +
    (it.description ? `<div class="trk-desc">${mdToHtml(it.description)}</div>` : `<p class="muted trk-no-desc">No description.</p>`) +
    (docs ? `<h3>Linked documents</h3><ul class="trk-docs">${docs}</ul>` : "") +
    `</div>` +
    `<aside class="trk-modal-side">` +
    `<dl class="trk-dl">` +
    `<dt>Status</dt><dd><span class="trk-dot ${dot}" aria-hidden="true"></span>${esc(it.status)}</dd>` +
    (it.type ? `<dt>Type</dt><dd>${esc(it.type)}</dd>` : "") +
    `<dt>Assignee</dt><dd>${who}</dd>` +
    (labels ? `<dt>Labels</dt><dd><span class="trk-labels">${labels}</span></dd>` : "") +
    `</dl>` +
    ac + dates +
    `</aside>` +
    `</div>` +
    `</div></div>`;
  const close = () => {
    overlay.innerHTML = "";
    trkSyncUrl(null);
    document.removeEventListener("keydown", onKey);
  };
  overlay.querySelectorAll("a[data-docchange]").forEach((a) =>
    a.addEventListener("click", (ev) => {
      ev.preventDefault();
      close();
      history.pushState(null, "", `/docs?change=${encodeURIComponent(a.dataset.docchange)}`);
      render();
    }));
  const onKey = (e) => {
    if (e.key === "Escape") close();
    // Cmd/Ctrl+S saves the selected status (matches the Backlog.md modal).
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "s") {
      e.preventDefault();
      overlay.querySelector("#trk-apply").click();
    }
  };
  document.addEventListener("keydown", onKey);
  overlay.querySelectorAll("[data-close]").forEach((b) => b.addEventListener("click", close));
  overlay.querySelector("#trk-apply").addEventListener("click", async () => {
    const msg = overlay.querySelector("#trk-msg");
    msg.textContent = "Saving…";
    try {
      const updated = await trackerFetch(`/tracker/tasks/${encodeURIComponent(it.id)}/status?provider=${encodeURIComponent(trk.provider)}`, {
        method: "POST",
        body: JSON.stringify({ status: overlay.querySelector("#trk-status").value }),
      });
      msg.textContent = `Now: ${updated.status}`;
      trk.config = null;
      await trkLoad(box, null);
    } catch (e) {
      msg.textContent = `Failed: ${e.message}`;
    }
  });
}

async function trackerFetch(url, opts) {
  const r = await fetch(url, {
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    ...opts,
  });
  const text = await r.text();
  let body = null;
  try { body = JSON.parse(text); } catch { body = { error: text }; }
  if (!r.ok) {
    const err = new Error((body && body.error) || `${r.status} ${r.statusText}`);
    err.status = r.status;
    err.provider = body && body.provider;
    throw err;
  }
  return body;
}

/* Docs entry (P3) — SDD change list + change.md/tasks.md viewer with
 * two-way tracker links: docs → tracker via the change's Ticket: id,
 * tracker detail → docs via the item's change-path doc refs. */
let docList = [];

function docQuery() {
  return new URLSearchParams(location.search).get("change") || "";
}

function docStripTicket(raw) {
  return String(raw || "")
    .replace(/^[\s`*(]+/, "").replace(/[\s`)!]+$/, "")
    .replace(/\s*\(.+$/, "").trim();
}

function docStatusClass(status) {
  const s = (status || "").toLowerCase();
  if (s === "done" || s === "archived" || s === "complete") return "done";
  if (s.includes("progress") || s === "in-progress") return "prog";
  return "other";
}

function docSyncUrl(name) {
  const q = new URLSearchParams(location.search);
  if (name) q.set("change", name);
  else q.delete("change");
  const qs = q.toString();
  history.replaceState(null, "", `${location.pathname}${qs ? "?" + qs : ""}`);
}

async function renderDocs(box) {
  box.innerHTML =
    `<div class="doc-page"><header class="doc-head"><h1>Docs</h1>` +
    `<p class="muted">SDD change documents — change.md and tasks.md — sandboxed to docs/skillgrid, read-only.</p></header>` +
    `<div id="doc-body"><div class="doc-loading" role="status"><span class="spinner" aria-hidden="true"></span>Loading changes…</div></div></div>`;
  const body = box.querySelector("#doc-body");
  let changes;
  try {
    const data = await api("/docs/changes");
    changes = data.changes || [];
  } catch (e) {
    body.innerHTML =
      `<div class="trk-empty-state"><p class="trk-empty-title">Failed to load changes</p>` +
      `<p class="muted">${esc(e.message)}</p></div>`;
    return;
  }
  docList = changes;
  const rows = changes.map((c) =>
    `<li><button type="button" class="doc-row" data-name="${esc(c.name)}">` +
    `<span class="doc-row-name">${esc(c.name)}</span>` +
    `<span class="doc-status st-${docStatusClass(c.status)}">${esc(c.status || "unknown")}</span>` +
    (c.ticket ? `<span class="doc-ticket" title="Open in Tracker">${esc(docStripTicket(c.ticket))}</span>` : "") +
    `</button></li>`).join("");
  body.innerHTML =
    `<ul class="doc-list">${rows || `<li class="doc-empty">No changes found.</li>`}</ul>`;
  body.querySelectorAll(".doc-row").forEach((el) =>
    el.addEventListener("click", () => docOpen(box, el.dataset.name)));
  const init = docQuery();
  if (init) docOpen(box, init);
}

async function docOpen(box, name) {
  const body = box.querySelector("#doc-body");
  docSyncUrl(name);
  body.innerHTML = `<div class="doc-loading" role="status"><span class="spinner" aria-hidden="true"></span>Loading ${esc(name)}…</div>`;
  let d;
  try {
    d = await api(`/docs/changes/${encodeURIComponent(name)}`);
  } catch (e) {
    body.innerHTML =
      `<div class="trk-empty-state"><p class="trk-empty-title">Failed to load ${esc(name)}</p>` +
      `<p class="muted">${esc(e.message)}</p><p><a class="doc-back" href="/docs" data-back>← Back to changes</a></p></div>`;
    body.querySelector("[data-back]")?.addEventListener("click", (ev) => {
      ev.preventDefault();
      history.pushState(null, "", "/docs");
      renderDocs(box);
    });
    return;
  }
  const ticket = d.ticket ? docStripTicket(d.ticket) : "";
  const provider = trk.provider;
  const links = ticket
    ? `<a class="doc-open-ticket btn" data-ticket="${esc(ticket)}">Open tracker item</a>`
    : `<span class="muted doc-no-ticket">No tracker ticket</span>`;
  body.innerHTML =
    `<div class="doc-view">` +
    `<header class="doc-view-head">` +
    `<button type="button" class="doc-back" data-back>← Back to changes</button>` +
    `<h2>${esc(name)}</h2>` +
    `<span class="doc-status st-${docStatusClass(d.status)}">${esc(d.status || "unknown")}</span>` +
    `<span class="doc-view-links">${links}</span>` +
    `</header>` +
    `<section class="doc-file"><h3>change.md</h3><div class="doc-md">${mdToHtml(d.change_md)}</div></section>` +
    (d.tasks_md ? `<section class="doc-file"><h3>tasks.md</h3><div class="doc-md">${mdToHtml(d.tasks_md)}</div></section>` : "") +
    `</div>`;
  body.querySelector("[data-back]").addEventListener("click", (ev) => {
    ev.preventDefault();
    history.pushState(null, "", "/docs");
    docSyncUrl("");
    renderDocs(box);
  });
  body.querySelector("[data-ticket]")?.addEventListener("click", (ev) => {
    ev.preventDefault();
    const q = new URLSearchParams({ provider, task: ev.currentTarget.dataset.ticket });
    history.pushState(null, "", `/tracker?${q.toString()}`);
    render();
  });
}

/* Tracker detail → SDD docs (03.4): doc refs that point at a change path
 * link back into the Docs entry. */
function trkDocLink(ref) {
  const m = String(ref).match(/changes\/([a-z0-9][a-z0-9-]*)/);
  if (!m) return esc(ref);
  return `<a href="/docs?change=${encodeURIComponent(m[1])}">${esc(ref)}</a>`;
}

/* Minimal, dependency-free CommonMark-ish renderer for the Docs entry.
 * No CDN / no external lib (the binary must work offline). Supports:
 * fenced code blocks, headings, GFM tables, blockquote (incl. the
 * `> **STATUS:**` header line), hr, task lists (- [ ] / - [x]), ul/ol,
 * paragraphs, and inline bold/italic/strikethrough/code/link. Raw HTML is
 * escaped first (docs are trusted repo files, never executed). */
function mdInline(t) {
  t = esc(t);
  const code = [];
  t = t.replace(/`([^`]+)`/g, (_, c) => { code.push(c); return `\u0000${code.length - 1}\u0000`; });
  t = t.replace(/\*\*\*([^*]+)\*\*\*/g, "<strong><em>$1</em></strong>");
  t = t.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  t = t.replace(/\*([^*\n]+)\*/g, "<em>$1</em>");
  t = t.replace(/~~([^~]+)~~/g, "<del>$1</del>");
  t = t.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g,
    (_, label, href) => `<a href="${href}">${label}</a>`);
  t = t.replace(/\u0000(\d+)\u0000/g, (_, i) => `<code>${code[+i]}</code>`);
  return t;
}

function mdTable(rows) {
  const cell = (row) => row.replace(/^\s*\|/, "").replace(/\|\s*$/, "").split("|").map((c) => c.trim());
  const header = cell(rows[0]);
  const aligns = rows.length > 1 ? cell(rows[1]).map((s) => {
    const l = s.startsWith(":"), r = s.endsWith(":");
    return l && r ? "center" : r ? "right" : l ? "left" : "";
  }) : [];
  const body = rows.slice(2);
  const attr = (a) => a ? ` style="text-align:${a}"` : "";
  const thead = "<tr>" + header.map((h, i) =>
    `<th${attr(aligns[i])}>${mdInline(h)}</th>`).join("") + "</tr>";
  const tbody = body.map((r) => "<tr>" +
    cell(r).map((c, i) => `<td${attr(aligns[i])}>${mdInline(c)}</td>`).join("") +
    "</tr>").join("");
  return `<table><thead>${thead}</thead>${tbody ? `<tbody>${tbody}</tbody>` : ""}</table>`;
}

function mdToHtml(md) {
  const lines = String(md || "").replace(/\r\n?/g, "\n").split("\n");
  const out = [];
  let i = 0;
  const isBlank = (s) => /^\s*$/.test(s);
  const isTableSep = (s) => /^\s*\|?[\s:|-]+\|[\s:|-]*$/.test(s) && s.includes("-");
  const isHead = (s) => /^\s{0,3}#{1,6}\s+/.test(s);
  const isQuote = (s) => /^\s{0,3}>/.test(s);
  const isFence = (s) => /^\s{0,3}```/.test(s) || /^\s{0,3}~~~/.test(s);
  const isUl = (s) => /^\s{0,3}[-*+]\s+/.test(s);
  const isOl = (s) => /^\s{0,3}\d+[.)]\s+/.test(s);
  const isHr = (s) => /^\s{0,3}([-*_])(\s*\1){2,}\s*$/.test(s);

  while (i < lines.length) {
    const line = lines[i];
    if (isBlank(line)) { i++; continue; }
    if (isFence(line)) {
      const fence = line.trim().slice(0, 3);
      const lang = line.trim().slice(3).trim();
      const buf = [];
      i++;
      while (i < lines.length && !lines[i].trimStart().startsWith(fence)) {
        buf.push(lines[i]);
        i++;
      }
      i++; // skip closing fence
      const langCls = lang ? ` class="language-${esc(lang)}"` : "";
      out.push(`<pre><code${langCls}>${esc(buf.join("\n"))}</code></pre>`);
      continue;
    }
    if (isHead(line)) {
      const m = line.match(/^\s{0,3}(#{1,6})\s+(.*)$/);
      const level = m[1].length;
      const text = m[2].replace(/\s+#+\s*$/, "");
      out.push(`<h${level}>${mdInline(text)}</h${level}>`);
      i++;
      continue;
    }
    if (isHr(line)) { out.push("<hr>"); i++; continue; }
    if (isQuote(line)) {
      const buf = [];
      while (i < lines.length && (isQuote(lines[i]) || (!isBlank(lines[i]) && buf.length))) {
        if (isBlank(lines[i])) break;
        buf.push(lines[i].replace(/^\s{0,3}>\s?/, ""));
        i++;
      }
      out.push(`<blockquote>${mdToHtml(buf.join("\n"))}</blockquote>`);
      continue;
    }
    if (line.includes("|") && i + 1 < lines.length && isTableSep(lines[i + 1])) {
      const rows = [line, lines[i + 1]];
      i += 2;
      while (i < lines.length && lines[i].includes("|") && !isBlank(lines[i])) {
        rows.push(lines[i]);
        i++;
      }
      out.push(mdTable(rows));
      continue;
    }
    if (isUl(line) || isOl(line)) {
      const ordered = isOl(line);
      const items = [];
      while (i < lines.length && (ordered ? isOl(lines[i]) : isUl(lines[i]))) {
        let text = lines[i].replace(ordered ? /^\s{0,3}\d+[.)]\s+/ : /^\s{0,3}[-*+]\s+/, "");
        const task = text.match(/^\[([ xX])\]\s+(.*)$/);
        if (task) {
          const done = task[1].toLowerCase() === "x";
          items.push(`<li class="task${done ? " done" : ""}"><span class="task-box">${done ? "✓" : ""}</span> ${mdInline(task[2])}</li>`);
        } else {
          items.push(`<li>${mdInline(text)}</li>`);
        }
        i++;
      }
      const tag = ordered ? "ol" : "ul";
      out.push(`<${tag}>${items.join("")}</${tag}>`);
      continue;
    }
    const buf = [line];
    i++;
    while (i < lines.length) {
      const l = lines[i];
      if (isBlank(l) || isHead(l) || isFence(l) || isQuote(l) || isUl(l) || isOl(l) || isHr(l)) break;
      buf.push(l);
      i++;
    }
    out.push(`<p>${buf.map(mdInline).join("<br>")}</p>`);
  }
  return out.join("\n");
}

/* Memory entry (P4) — a control panel over Mnemonic observations: a
 * master-detail split (search/results on the left, a governance detail
 * pane on the right). Mirrors the tracker entry's structure (async entry,
 * state object, per-widget render fns, bind fns) and reuses the .trk-*
 * visual language. Every governance widget is isolated: a failed/absent
 * 013 field kills that widget only, never the search or the pane. */
const MEM_VISIBILITIES = ["private", "team", "restricted", "agent"];
const MEM_STATUSES = ["active", "superseded", "archived"];
const MEM_WEB_SOURCES = ["context7", "exa", "deepwiki", "fetch", "manual"];

let mem = {
  query: "", results: [], recent: null, current: null, governance: null,
  webQuery: "", web: null, webStatus: null,
  view: "search", // "search" | "web"
  showNumbers: {},
  _timer: null,
};

function memToken() {
  // Write routes are gated by SKILLGRID_HTTP_TOKEN; the dashboard carries the
  // bearer token in localStorage (set in Settings, P5+). Absent → no header,
  // which the server accepts when no token is configured.
  return localStorage.getItem("sgmn-token") || "";
}

function memWriteHeaders() {
  const h = { Accept: "application/json", "Content-Type": "application/json" };
  if (memToken()) h["Authorization"] = "Bearer " + memToken();
  return h;
}

// memWrite is the token-attaching fetch for write-gated memory routes
// (pin/unpin/share/status/PATCH/DELETE). Mirrors trackerFetch: parses the
// body, throws an Error with .status on !ok so callers can surface 400/401.
async function memWrite(url, opts) {
  const o = Object.assign({ method: "POST" }, opts || {});
  o.headers = Object.assign(memWriteHeaders(), o.headers || {});
  const r = await fetch(url, o);
  const text = await r.text();
  let body = null;
  try { body = JSON.parse(text); } catch { body = { error: text }; }
  if (!r.ok) {
    const err = new Error((body && body.error) || `${r.status} ${r.statusText}`);
    err.status = r.status;
    throw err;
  }
  return body;
}

// memUrl appends the project param to a memory route (the project var is the
// selected store, same source the tracker reads via ensureProjects). The path
// may already carry its own query string (e.g. /search?query=…&limit=…), so
// use & when one is present — a second ? would 400 ("project is required").
function memUrl(path) {
  const sep = path.includes("?") ? "&" : "?";
  return `${path}${sep}project=${encodeURIComponent(project || "")}`;
}

async function renderMemory(box) {
  // Ensure a project is selected before any project-scoped fetch. The memory
  // routes require ?project=, and this entry is deferred to a microtask (TDZ),
  // so without this the first visit fires ?project= empty → 400 → blank board.
  // Idempotent: returns immediately once the project list has loaded.
  await ensureProjects();
  box.innerHTML =
    `<div class="mem-page"><header class="mem-head"><h1>Memory</h1>` +
    `<p class="muted">Search observations, open the detail pane, and run the 013 governance controls (pin, share, status, in-place edit).</p></header>` +
    `<div class="mem-top">` +
    `<div class="mem-viewtabs" role="group" aria-label="Memory view">` +
    `<button type="button" data-memview="search" class="mem-view-tab${mem.view === "search" ? " active" : ""}">Observations</button>` +
    `<button type="button" data-memview="web" class="mem-view-tab${mem.view === "web" ? " active" : ""}">Web cache</button>` +
    `</div>` +
    `<p class="mem-source" id="mem-source"></p>` +
    `</div>` +
    `<div class="mem-body" id="mem-body"><div class="mem-loading" role="status"><span class="spinner" aria-hidden="true"></span>Loading memory…</div></div></div>`;
  const body = box.querySelector("#mem-body");
  body.querySelectorAll(".mem-view-tab").forEach((b) =>
    b.addEventListener("click", () => { mem.view = b.dataset.memview; renderMemory(box); }));
  if (mem.view === "web") { memRenderWeb(box); return; }
  memRenderMain(box);
}

async function memRenderMain(box) {
  const body = box.querySelector("#mem-body");
  const currentId = mem.current ? mem.current.id : null;
  body.innerHTML =
    `<div class="mem-grid">` +
    `<section class="mem-search-col" aria-label="Search">` +
    `<div class="mem-search-wrap">` +
    `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>` +
    `<input type="search" id="mem-q" value="${esc(mem.query)}" placeholder="Search observations by title, type, content..." aria-label="Search observations"></div>` +
    `<div class="mem-colhead"><span id="mem-count">0</span>` +
    `<button type="button" id="mem-raw-toggle" class="mem-numbtn" aria-pressed="false">show numbers</button></div>` +
    `<div id="mem-results" class="mem-results"></div>` +
    `<pre class="mem-raw" id="mem-raw" hidden></pre>` +
    `</section>` +
    `<section class="mem-detail" id="mem-detail" aria-label="Detail">` +
    (currentId ? `<div class="mem-loading" role="status"><span class="spinner" aria-hidden="true"></span>Loading detail…</div>` : memEmpty()) +
    `</section>` +
    `</div>`;
  const q = body.querySelector("#mem-q");
  q.addEventListener("input", () => {
    mem.query = q.value;
    if (mem._timer) clearTimeout(mem._timer);
    mem._timer = setTimeout(() => memLoad(box), 300);
  });
  body.querySelector("#mem-raw-toggle").addEventListener("click", () =>
    showNumbers(body.querySelector("#mem-results"),
      () => ({ observations: mem.results, recent: mem.recent }),
      body.querySelector("#mem-raw"), body.querySelector("#mem-raw-toggle")));
  // Seed the results: cached recent list when the query is empty, else search.
  await memLoad(box);
  if (currentId) memOpenDetail(box, currentId);
}

// memLoad refetches the left column (search if a query is set, otherwise the
// recent list) and re-renders the rows. A search failure is an isolated
// widget error in the results box — the detail pane stays interactive.
async function memLoad(box) {
  const results = (box && box.querySelector("#mem-results")) || document.querySelector("#mem-results");
  const count = (box && box.querySelector("#mem-count")) || document.querySelector("#mem-count");
  if (!results) return;
  try {
    if (mem.query.trim()) {
      const data = await api(memUrl(`/search?query=${encodeURIComponent(mem.query.trim())}&match_mode=any&limit=20`));
      mem.results = data.observations || [];
    } else {
      let recent = mem.recent;
      if (!recent) {
        const data = await api(memUrl("/observations?limit=50"));
        recent = data.observations || [];
        mem.recent = recent;
      }
      mem.results = recent;
    }
    results.innerHTML = memResultsHtml();
    if (count) count.textContent = String(mem.results.length);
    memBindResults(box);
  } catch (e) {
    results.innerHTML =
      `<div class="trk-empty-state"><p class="trk-empty-title">Search failed</p>` +
      `<p class="muted">${esc(e.message)}</p></div>`;
    if (count) count.textContent = "–";
  }
}

function memResultsHtml() {
  if (!mem.results.length) {
    if (mem.query.trim()) {
      return `<div class="trk-empty-state"><p class="trk-empty-title">No observations match ${esc(mem.query.trim())}</p>` +
        `<p class="muted">Try a different term, or clear the search to see recent memory.</p></div>`;
    }
    return memEmpty();
  }
  return `<div class="mem-list" id="mem-list">${mem.results.map(memRow).join("")}</div>`;
}

// memRow is one results row (the asset-library twin): id, title, type badge,
// visibility badge, status, owner, retrieval-usage, and a pinned marker.
function memRow(o) {
  const type = o.type || "—";
  const vis = o.visibility || "private";
  const status = o.status || "active";
  const pinned = o.pinned ? `<span class="mem-pin" title="Pinned" aria-label="Pinned">📌</span>` : "";
  const usage = o.retrieval_usage ? `<span class="mem-usage" title="Retrieval usage">↻ ${esc(o.retrieval_usage)}</span>` : "";
  const owner = o.owner ? `<span class="mem-owner" title="Owner">${esc(o.owner)}</span>` : "";
  const revs = o.revision_count ? `<span class="mem-revs" title="Revisions">v${esc(o.revision_count)}</span>` : "";
  return `<div class="mem-item" data-id="${esc(o.id)}" role="button" tabindex="0">` +
    `<span class="mem-item-id">${pinned}<span class="mem-mono">${esc(o.id)}</span></span>` +
    `<span class="mem-item-title">${esc(o.title || "(untitled)")}</span>` +
    `<span class="mem-item-badges">` +
    `<span class="mem-badge mem-type-${esc(type)}">${esc(type)}</span>` +
    `<span class="mem-badge mem-vis-${esc(vis)}">${esc(vis)}</span>` +
    `<span class="mem-badge mem-status-${esc(status)}">${esc(status)}</span>` +
    `</span>` +
    `<span class="mem-item-foot">${owner}${usage}${revs}</span>` +
    `</div>`;
}

function memEmpty() {
  const prompts = memSuggestedPromptsHtml();
  return `<div class="mem-empty">` +
    `<p class="trk-empty-title">Nothing selected</p>` +
    `<p class="muted">Search above, or start from a recent session prompt.</p>` +
    prompts +
    `</div>`;
}

// memSuggestedPromptsHtml turns recent session prompts (GET /context) into
// clickable chips that fill the search box — so the empty state is never blank.
function memSuggestedPromptsHtml() {
  const sessions = (typeof memContextCache !== "undefined" && memContextCache) || [];
  const chips = sessions
    .filter((s) => (s.title || s.summary || "").trim())
    .slice(0, 8)
    .map((s) => {
      const label = (s.title || s.summary || "").trim().split("\n")[0].slice(0, 60);
      return `<button type="button" class="mem-prompt-chip" data-prompt="${esc(label)}">${esc(label)}</button>`;
    }).join("");
  if (!chips) return `<p class="muted mem-no-prompts">No recent session prompts yet.</p>`;
  return `<div class="mem-prompts"><span class="mem-prompts-label">Suggested prompts</span><div class="mem-prompt-list">${chips}</div></div>`;
}

let memContextCache = [];
async function memLoadContext(box) {
  try {
    const data = await api(memUrl("/context?limit=10"));
    memContextCache = data.sessions || [];
  } catch {
    memContextCache = [];
  }
  // Re-render only if the empty-state prompt list is on screen.
  const empty = (box && box.querySelector("#mem-detail")) || document.querySelector("#mem-detail");
  if (empty && !mem.current) empty.innerHTML = memEmpty();
  memBindPrompts(box);
}

function memBindResults(box) {
  const scope = box || document;
  scope.querySelectorAll("#mem-results .mem-item").forEach((el) => {
    el.addEventListener("click", () => memOpenDetail(scope, el.dataset.id));
    el.addEventListener("keydown", (e) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        memOpenDetail(scope, el.dataset.id);
      }
    });
  });
  memBindPrompts(scope);
}

function memBindPrompts(box) {
  const scope = box || document;
  scope.querySelectorAll(".mem-prompt-chip").forEach((el) =>
    el.addEventListener("click", () => {
      const q = scope.querySelector("#mem-q");
      if (q) q.value = el.dataset.prompt || "";
      mem.query = q ? q.value : el.dataset.prompt;
      memLoad(scope);
      if (q) q.focus();
    }));
}

async function memOpenDetail(box, id) {
  const detail = (box && box.querySelector("#mem-detail")) || document.querySelector("#mem-detail");
  if (!detail) return;
  detail.innerHTML = `<div class="mem-loading" role="status"><span class="spinner" aria-hidden="true"></span>Loading ${esc(id)}…</div>`;
  let o;
  try {
    o = await api(memUrl(`/observations/${encodeURIComponent(id)}`));
  } catch (e) {
    // 404/500: an error message in the pane, dashboard stays interactive.
    detail.innerHTML =
      `<div class="mem-detail-error"><p class="trk-empty-title">Could not load observation ${esc(id)}</p>` +
      `<p class="muted">${esc(e.message)}</p>` +
      `<button type="button" class="mem-back" data-back>← Back to results</button></div>`;
    detail.querySelector("[data-back]").addEventListener("click", () => { mem.current = null; memRenderMain(box || document); });
    return;
  }
  mem.current = o;
  detail.innerHTML = memDetailHtml(o);
  memBindDetail(box, o);
  memLoadGovernance(box, o.id);
  memLoadRelations(box, o.id);
}

function memDetailHtml(o) {
  const vis = o.visibility || "private";
  const status = o.status || "active";
  const type = o.type || "—";
  return `<div class="mem-detail-card">` +
    `<header class="mem-detail-head">` +
    `<span class="mem-panel-id"><span class="mem-mono">${esc(o.id)}</span>` +
    `<span class="mem-badge mem-type-${esc(type)}">${esc(type)}</span>` +
    `<span class="mem-badge mem-vis-${esc(vis)}">${esc(vis)}</span>` +
    `<span class="mem-badge mem-status-${esc(status)}">${esc(status)}</span>` +
    (o.pinned ? `<span class="mem-pin" title="Pinned">📌</span>` : "") +
    `</span>` +
    `<span class="mem-detail-actions">` +
    `<span class="muted" id="mem-msg"></span>` +
    `<button type="button" class="btn" id="mem-pin">${o.pinned ? "Unpin" : "Pin"}</button>` +
    `<button type="button" class="btn" id="mem-edit">Edit</button>` +
    `<button type="button" class="btn mem-danger" id="mem-delete">Delete</button>` +
    `</span></header>` +
    `<div class="mem-detail-main">` +
    `<h2>${esc(o.title || "(untitled)")}</h2>` +
    `<div class="mem-content">${o.content ? mdToHtml(o.content) : `<p class="muted">No content.</p>`}</div>` +
    `<div id="mem-edit-area" hidden>` +
    `<textarea id="mem-edit-text" class="mem-edit-text" aria-label="Edit content" rows="8"></textarea>` +
    `<div class="mem-edit-actions"><button type="button" class="btn" id="mem-save">Save</button>` +
    `<button type="button" class="mem-back" id="mem-cancel-edit">Cancel</button></div>` +
    `</div>` +
    `<dl class="mem-dl">` +
    (o.session_id ? `<dt>Session</dt><dd class="mem-mono">${esc(o.session_id)}</dd>` : "") +
    (o.scope ? `<dt>Scope</dt><dd>${esc(o.scope)}</dd>` : "") +
    (o.topic_key ? `<dt>Topic key</dt><dd class="mem-mono">${esc(o.topic_key)}</dd>` : "") +
    `<dt>Created</dt><dd>${esc(o.created_at || "—")}</dd>` +
    `<dt>Updated</dt><dd>${esc(o.updated_at || "—")}</dd>` +
    `</dl>` +
    `</div>` +
    `<aside class="mem-side">` +
    `<div id="mem-gov"><div class="mem-loading" role="status"><span class="spinner" aria-hidden="true"></span>governance…</div></div>` +
    `<div id="mem-rels"><div class="mem-loading" role="status"><span class="spinner" aria-hidden="true"></span>relations…</div></div>` +
    `</aside>` +
    `</div>`;
}

function memBindDetail(box, o) {
  const scope = box || document;
  const detail = scope.querySelector("#mem-detail");
  const msg = () => detail.querySelector("#mem-msg");
  const setMsg = (t) => { const m = msg(); if (m) m.textContent = t; };
  const pin = detail.querySelector("#mem-pin");
  if (pin) pin.addEventListener("click", async () => {
    const on = !o.pinned;
    try {
      const verb = on ? "/pin" : "/unpin";
      const url = memUrl(`/memory/observations/${encodeURIComponent(o.id)}${verb}`);
      await memWrite(url, { method: "POST" });
      o.pinned = on;
      setMsg(on ? "Pinned" : "Unpinned");
      memRenderMain(scope);      // refresh results (pinned flag)
      memOpenDetail(scope, o.id); // re-fetch the detail
    } catch (e) {
      setMsg(`Failed: ${e.message}`);
    }
  });
  const del = detail.querySelector("#mem-delete");
  if (del) del.addEventListener("click", async () => {
    if (!confirm(`Delete observation ${o.id}? This is a soft delete.`)) return;
    try {
      await memWrite(memUrl(`/memory/observations/${encodeURIComponent(o.id)}`), { method: "DELETE" });
      mem.current = null;
      mem.recent = null;
      memRenderMain(scope);
    } catch (e) {
      setMsg(`Failed: ${e.message}`);
    }
  });
  // In-place edit (04.11): a textarea pre-filled with the content; Save calls
  // PATCH which appends a 013 version server-side (prior content → governance).
  const edit = detail.querySelector("#mem-edit");
  const area = detail.querySelector("#mem-edit-area");
  const ta = detail.querySelector("#mem-edit-text");
  if (edit && ta) edit.addEventListener("click", () => {
    ta.value = o.content || "";
    area.hidden = false;
    detail.querySelector(".mem-content").style.display = "none";
    ta.focus();
  });
  const cancel = detail.querySelector("#mem-cancel-edit");
  if (cancel) cancel.addEventListener("click", () => {
    area.hidden = true;
    detail.querySelector(".mem-content").style.display = "";
  });
  const save = detail.querySelector("#mem-save");
  if (save) save.addEventListener("click", async () => {
    try {
      await memWrite(memUrl(`/memory/observations/${encodeURIComponent(o.id)}`), {
        method: "PATCH",
        body: JSON.stringify({ content: ta.value }),
      });
      mem.recent = null;
      memRenderMain(scope);      // new content is current + version history refreshes
      memOpenDetail(scope, o.id);
    } catch (e) {
      setMsg(`Failed: ${e.message}`);
    }
  });
  memBindGovernance(scope, o);
}

// ── Governance (013) widgets — each isolated; a failure shows a placeholder
//    in that widget only. ────────────────────────────────────────────────
async function memLoadGovernance(box, id) {
  const gov = (box && box.querySelector("#mem-gov")) || document.querySelector("#mem-gov");
  if (!gov) return;
  let g;
  try {
    g = await api(memUrl(`/memory/observations/${encodeURIComponent(id)}/governance`));
  } catch (e) {
    gov.innerHTML = memPlaceholder("Governance", `unavailable — ${e.message}`);
    return;
  }
  mem.governance = g;
  gov.innerHTML = memGovernanceHtml(g, id) +
    `<button type="button" class="mem-numbtn" id="mem-gov-raw">show numbers</button><pre class="mem-raw" id="mem-gov-raw" hidden></pre>`;
  gov.querySelector("#mem-gov-raw").textContent = JSON.stringify(g, null, 2);
  showNumbers(gov, () => g, gov.querySelector("#mem-gov-raw"), gov.querySelector("#mem-gov-raw"));
  memBindGovernance(box, mem.current || { id, visibility: g.visibility, status: g.status });
}

function memGovernanceHtml(g, id) {
  const versions = (g.versions || []).slice().reverse();
  const grants = g.grants || [];
  return `<div class="mem-gov-card">` +
    `<h3>Governance</h3>` +
    `<dl class="mem-dl">` +
    `<dt>Owner</dt><dd>${esc(g.owner || "—")}</dd>` +
    `<dt>Status</dt><dd><span class="mem-badge mem-status-${esc(g.status || "active")}">${esc(g.status || "active")}</span></dd>` +
    `<dt>Revisions</dt><dd>${esc(g.revision_count ?? 0)}</dd>` +
    `<dt>Retrieval usage</dt><dd>${esc(g.retrieval_usage ?? 0)}</dd>` +
    (g.restricted_no_grants ? `<dt>ACL</dt><dd class="muted">restricted, owner-only</dd>` : "") +
    `</dl>` +
    `<div class="mem-share">` +
    `<label for="mem-visibility">Visibility</label>` +
    `<div class="mem-share-row">` +
    `<select id="mem-visibility" class="project-select" aria-label="Visibility">` +
    MEM_VISIBILITIES.map((v) => `<option value="${v}"${v === (g.visibility || "private") ? " selected" : ""}>${v}</option>`).join("") +
    `</select>` +
    `<button type="button" class="btn" id="mem-share-btn">Share</button>` +
    `</div>` +
    (g.visibility === "restricted" ? "" : "") +
    `<div id="mem-grants" class="mem-grants">${grants.length ? `<span class="muted">Grants: ${grants.map((x) => `<span class="trk-chip">${esc(x.grantee)}${x.grant_type ? ` (${esc(x.grant_type)})` : ""}</span>`).join(" ")}</span>` : `<span class="muted">No ACL grants.</span>`}` +
    `<input type="text" id="mem-grantee" class="mem-grantee" placeholder="grantee (for restricted)" aria-label="Grantee" hidden>` +
    `</div>` +
    `<div class="mem-status-ctrl">` +
    `<label for="mem-status">Lifecycle status</label>` +
    `<div class="mem-share-row">` +
    `<select id="mem-status" class="project-select" aria-label="Status">` +
    MEM_STATUSES.map((s) => `<option value="${s}"${s === (g.status || "active") ? " selected" : ""}>${s}</option>`).join("") +
    `</select>` +
    `<button type="button" class="btn" id="mem-status-btn">Set</button>` +
    `</div></div>` +
    `<div class="mem-versions"><h4>Version history</h4>` +
    (versions.length
      ? `<ol class="mem-version-list">${versions.map((v) =>
          `<li class="mem-version"><span class="mem-mono">v${esc(v.revision)}</span>` +
          `<span class="muted">${esc((v.created_at || "").slice(0, 10))}</span>` +
          `<span class="mem-version-preview">${esc((v.content || "").replace(/\s+/g, " ").slice(0, 80))}</span></li>`).join("")}</ol>`
      : `<p class="muted">No prior versions.</p>`) +
    `</div>` +
    memPlaceholder("Layer drill-down", "013 L0 → L1 → L2 → L3 layered memory — available when 013 lands; flat pre-013 view shown in the detail pane.") +
    memPlaceholder("Agent loadout", "Read-only view of visibility=agent bindings — available when 013 lands.") +
    `</div>`;
}

function memBindGovernance(box, o) {
  const scope = box || document;
  const gov = scope.querySelector("#mem-gov");
  if (!gov) return;
  const share = gov.querySelector("#mem-share-btn");
  const visSel = gov.querySelector("#mem-visibility");
  const grantee = gov.querySelector("#mem-grantee");
  // The grantee (ACL) editor is shown for a restricted observation.
  if (grantee && visSel) grantee.hidden = visSel.value !== "restricted";
  if (share) share.addEventListener("click", async () => {
    const visibility = visSel.value;
    // Explicit click required — never a default. Confirm before changing.
    if (!confirm(`Change visibility to "${visibility}"?`)) return;
    const grants = [];
    if (visibility === "restricted" && grantee && grantee.value.trim()) {
      grants.push(grantee.value.trim());
    }
    try {
      const res = await memWrite(memUrl(`/memory/observations/${encodeURIComponent(o.id)}/share`), {
        method: "POST",
        body: JSON.stringify({ visibility, grants }),
      });
      o.visibility = res.visibility || visibility;
      memLoadGovernance(scope, o.id);
      memRenderMain(scope); // refresh the results visibility badge
    } catch (e) {
      // 400 for an unknown visibility target — render the reason inline.
      const card = gov.querySelector(".mem-share");
      if (card) card.insertAdjacentHTML("beforeend", `<p class="mem-share-error">${esc(e.message)}</p>`);
    }
  });
  if (visSel) visSel.addEventListener("change", () => {
    if (grantee) grantee.hidden = visSel.value !== "restricted";
  });
  const setBtn = gov.querySelector("#mem-status-btn");
  const statusSel = gov.querySelector("#mem-status");
  if (setBtn) setBtn.addEventListener("click", async () => {
    try {
      const res = await memWrite(memUrl(`/memory/observations/${encodeURIComponent(o.id)}/status`), {
        method: "POST",
        body: JSON.stringify({ status: statusSel.value }),
      });
      o.status = res.status || statusSel.value;
      memLoadGovernance(scope, o.id);
      memRenderMain(scope);
    } catch (e) {
      const card = gov.querySelector(".mem-status-ctrl");
      if (card) card.insertAdjacentHTML("beforeend", `<p class="mem-share-error">${esc(e.message)}</p>`);
    }
  });
}

// ── Relations drill-down (04.x) ─────────────────────────────────────────
async function memLoadRelations(box, id) {
  const rels = (box && box.querySelector("#mem-rels")) || document.querySelector("#mem-rels");
  if (!rels) return;
  let data;
  try {
    data = await api(memUrl(`/relations/${encodeURIComponent(id)}`));
  } catch (e) {
    rels.innerHTML = memPlaceholder("Relations", `unavailable — ${e.message}`);
    return;
  }
  const list = data.relations || [];
  rels.innerHTML = `<div class="mem-rels-card"><h3>Relations</h3>` +
    (list.length
      ? `<ul class="mem-rel-list">${list.map(memRelRow).join("")}</ul>`
      : `<p class="muted">No relations.</p>`) +
    `<button type="button" class="mem-numbtn" id="mem-rel-raw">show numbers</button><pre class="mem-raw" id="mem-rel-raw" hidden></pre>` +
    `</div>`;
  rels.querySelector("#mem-rel-raw").textContent = JSON.stringify(data, null, 2);
  showNumbers(rels, () => data, rels.querySelector("#mem-rel-raw"), rels.querySelector("#mem-rel-raw"));
  rels.querySelectorAll(".mem-rel").forEach((el) =>
    el.addEventListener("click", () => {
      const target = el.dataset.other;
      if (target) memOpenDetail(box, target);
    }));
}

function memRelRow(r) {
  // Confidence badge: e.g. EXTRACTED 0.9. The relation string is the type.
  const conf = (r.confidence !== undefined && r.confidence !== null) ? Number(r.confidence) : null;
  const confBadge = conf !== null
    ? `<span class="mem-conf" title="Confidence">${conf.toFixed(2)}</span>` : "";
  const selfId = mem.current ? String(mem.current.id) : "";
  const other = String(r.src_obs_id) === selfId ? r.dst_obs_id : r.src_obs_id;
  const dir = String(r.src_obs_id) === selfId ? "→" : "←";
  return `<li><button type="button" class="mem-rel" data-other="${esc(other)}">` +
    `<span class="mem-badge mem-rel-type">${esc(r.relation || "related")}</span>` +
    confBadge +
    `<span class="mem-mono mem-rel-other">${dir} ${esc(other)}</span>` +
    (r.reason ? `<span class="muted mem-rel-reason">${esc(r.reason)}</span>` : "") +
    `</button></li>`;
}

// ── Web-cache sub-section (04.9) — the old viewer, moved into Memory. ──────
async function memRenderWeb(box) {
  const body = box.querySelector("#mem-body");
  body.innerHTML =
    `<div class="mem-web">` +
    `<div class="mem-search-wrap">` +
    `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>` +
    `<input type="search" id="mem-web-q" value="${esc(mem.webQuery)}" placeholder="Search cached web research..." aria-label="Search web cache"></div>` +
    `<div class="mem-colhead"><span id="mem-web-status"></span>` +
    `<select id="mem-web-src" class="project-select" aria-label="Web source">` +
    `<option value="">all sources</option>` +
    MEM_WEB_SOURCES.map((s) => `<option value="${s}">${s}</option>`).join("") +
    `</select>` +
    `<button type="button" id="mem-web-raw-toggle" class="mem-numbtn" aria-pressed="false">show numbers</button></div>` +
    `<div id="mem-web-results" class="mem-results"></div>` +
    `<pre class="mem-raw" id="mem-web-raw" hidden></pre>` +
    `</div>`;
  const q = body.querySelector("#mem-web-q");
  let timer = null;
  q.addEventListener("input", () => {
    mem.webQuery = q.value;
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => memLoadWeb(box), 300);
  });
  body.querySelector("#mem-web-src").addEventListener("change", () => memLoadWeb(box));
  body.querySelector("#mem-web-raw-toggle").addEventListener("click", () =>
    showNumbers(body.querySelector("#mem-web-results"),
      () => ({ entries: mem.web && mem.web.entries, status: mem.webStatus }),
      body.querySelector("#mem-web-raw"), body.querySelector("#mem-web-raw-toggle")));
  memLoadWeb(box);
}

async function memLoadWeb(box) {
  const results = (box && box.querySelector("#mem-web-results")) || document.querySelector("#mem-web-results");
  const status = (box && box.querySelector("#mem-web-status")) || document.querySelector("#mem-web-status");
  const src = (box && box.querySelector("#mem-web-src")) || document.querySelector("#mem-web-src");
  // Status is a side channel; its failure must not kill the search widget.
  (async () => {
    try {
      const st = await api(memUrl("/web/status"));
      mem.webStatus = st;
      if (status) status.textContent = `${st.total_entries ?? 0} entries cached`;
    } catch {
      if (status) status.textContent = "";
    }
  })();
  const q = (mem.webQuery || "").trim();
  const source = src ? (src.value || "") : "";
  const qs = `query=${encodeURIComponent(q)}&limit=20${source ? `&source=${encodeURIComponent(source)}` : ""}`;
  try {
    const data = await api(memUrl(`/web/search?${qs}`));
    mem.web = data;
    const entries = data.entries || [];
    results.innerHTML = entries.length
      ? `<div class="mem-web-list">${entries.map(memWebRow).join("")}</div>`
      : `<div class="trk-empty-state"><p class="trk-empty-title">No cached web entries${q ? ` for ${esc(q)}` : ""}</p></div>`;
  } catch (e) {
    results.innerHTML =
      `<div class="trk-empty-state"><p class="trk-empty-title">Web cache search failed</p>` +
      `<p class="muted">${esc(e.message)}</p></div>`;
  }
}

function memWebRow(e) {
  return `<div class="mem-web-item">` +
    `<span class="mem-web-id"><span class="mem-mono">${esc(e.id)}</span>` +
    (e.source ? `<span class="mem-badge mem-type-${esc(e.source)}">${esc(e.source)}</span>` : "") +
    `</span>` +
    `<span class="mem-web-title">${esc(e.title || e.url || "(untitled)")}</span>` +
    (e.url ? `<span class="mem-web-url mem-mono">${esc(e.url)}</span>` : "") +
    (e.fetched_at ? `<span class="muted mem-mono">${esc((e.fetched_at || "").slice(0, 10))}</span>` : "") +
    `</div>`;
}

// ── Show-numbers / raw-JSON toggle (04.x) ────────────────────────────────
// showNumbers appends a "show numbers" toggle that reveals the widget's raw
// JSON in a <pre class="mem-raw">. getData() is called lazily on toggle so the
// <pre> always reflects current widget state.
function showNumbers(container, getData, pre, btn) {
  if (!pre) return;
  pre.textContent = JSON.stringify(getData(), null, 2);
  if (btn) {
    btn.setAttribute("aria-pressed", String(mem.showNumbers[btn.id] || false));
    btn.onclick = () => {
      pre.hidden = !pre.hidden;
      mem.showNumbers[btn.id] = !pre.hidden;
      btn.setAttribute("aria-pressed", String(!pre.hidden));
      btn.textContent = pre.hidden ? "show numbers" : "hide numbers";
      if (!pre.hidden) pre.textContent = JSON.stringify(getData(), null, 2);
    };
  }
}

// memPlaceholder renders a labeled, collapsed box for a 013 feature that has
// not landed yet (forward-compat: layer drill-down, agent loadout).
function memPlaceholder(label, detail) {
  return `<details class="mem-placeholder">` +
    `<summary>${esc(label)} <span class="muted mem-placeholder-avail">not available yet</span></summary>` +
    `<p class="muted">${esc(detail || "Available in a later 013 step.")}</p>` +
    `</details>`;
}
