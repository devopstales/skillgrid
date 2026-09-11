"use strict";
/* Dashboard shell router (P1). Path-routed menu entries (/welcome, /tracker,
 * …); the server serves the same shell for each path. The Welcome entry is
 * fully static (zero fetches). Data entries land in P2–P6; until then they
 * render as labeled disabled stubs. Project list loads lazily — never on the
 * Welcome entry. */
const $ = (s) => document.querySelector(s);

const STUBS = {
  memory: { phase: "P4", title: "Memory", body: "Search, observation detail, pin/unpin/delete, plus governance views." },
  code: { phase: "P5", title: "Code", body: "Index status, freshness banner, BM25 search with source view." },
  sessions: { phase: "P6", title: "Sessions", body: "Session list with titles, recent context, and summaries." },
};

const LIVE = ["welcome", "tracker", "docs"];

const ROUTES = ["welcome", "tracker", "docs", "memory", "code", "sessions"];

let project = localStorage.getItem("sgmn-project") || "";
let projectsLoaded = false;

function currentRoute() {
  const seg = location.pathname.replace(/\/+$/, "").split("/").pop() || "welcome";
  return ROUTES.includes(seg) ? seg : "welcome";
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
  if (route === "welcome") {
    welcome.hidden = false;
    stub.hidden = true;
    return;
  }
  // Future entries progressively go live in P3–P6; until then, stubs.
  if (!LIVE.includes(route)) {
    const info = STUBS[route];
    welcome.hidden = true;
    stub.hidden = false;
    stub.innerHTML =
      `<div class="stub-card"><span class="phase-id next">${esc(info.phase)}</span>` +
      `<h2>${esc(info.title)}</h2><p>${esc(info.body)}</p>` +
      `<p>Coming in ${esc(info.phase)} — not built yet.</p></div>`;
    return;
  }
  if (route === "tracker") {
    welcome.hidden = true;
    stub.hidden = false;
    renderTracker(stub);
    // Project context matters only on data entries; load it lazily here.
    ensureProjects();
    return;
  }
  if (route === "docs") {
    welcome.hidden = true;
    stub.hidden = false;
    renderDocs(stub);
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

let trk = { provider: "backlogmd", query: "", priority: "all", config: null, tasks: [], connected: {} };

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
  return { provider, task: q.get("task") || "" };
}

async function renderTracker(box) {
  const init = trkQuery();
  if (init.provider) {
    trk.provider = init.provider;
    trk.config = null;
    trk.tasks = [];
  }
  box.innerHTML =
    `<div class="trk-page"><header class="trk-head"><h1>Tracker</h1>` +
    `<p class="muted">Normalized tasks across your ticketing systems. Switch providers to view each source on one board.</p></header>` +
    `<div id="trk-body"><div class="trk-loading" role="status"><span class="spinner" aria-hidden="true"></span>Loading tracker…</div></div></div>`;
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
  await trkLoad(box, init.task);
}

async function trkLoad(box, openTaskId) {
  const body = box.querySelector("#trk-body");
  const meta = TRK_PROVIDERS.find((p) => p.id === trk.provider);
  const switcher = TRK_PROVIDERS.map((p) => {
    const active = p.id === trk.provider;
    const dot = trk.connected[p.id] === false ? `<span class="trk-conn-dot" title="Not connected" aria-label="Not connected"></span>` : "";
    return `<button type="button" role="tab" aria-selected="${active}" data-provider="${p.id}"` +
      ` class="trk-sw${active ? " active" : ""}">${esc(p.label)}${dot}</button>`;
  }).join("");
  let main;
  if (trk.connected[trk.provider] === false && !trk.config) {
    main = `<div class="trk-empty-state"><p class="trk-empty-title">${esc(meta.label)} is not connected</p>` +
      `<p class="muted">Configure the ${esc(meta.label)} adapter: ${esc(meta.source)}.</p></div>`;
  } else {
    try {
      const [cfg, data] = await Promise.all([
        trk.config || trkFetch(`/tracker/config?provider=${encodeURIComponent(trk.provider)}`),
        trkFetch(`/tracker/tasks?provider=${encodeURIComponent(trk.provider)}`),
      ]);
      trk.config = cfg;
      trk.tasks = data.tasks || [];
      main = trkBoard();
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
    `<input type="search" id="trk-q" value="${esc(trk.query)}" placeholder="Filter by title, id or label..." aria-label="Filter tasks"></div>` +
    `<div class="trk-prio" role="group" aria-label="Priority filter">` +
    TRK_PRIORITIES.map((p) => `<button type="button" data-prio="${p}" aria-pressed="${trk.priority === p}"` +
      ` class="trk-prio-btn${trk.priority === p ? " active" : ""}">${p}</button>`).join("") +
    `</div></div><div id="trk-board-wrap">${main}</div><div id="trk-overlay"></div>`;
  body.querySelectorAll(".trk-sw").forEach((b) => b.addEventListener("click", () => {
    trk.provider = b.dataset.provider;
    trk.config = null;
    trk.tasks = [];
    trkSyncUrl(null);
    renderTracker(box);
  }));
  const q = body.querySelector("#trk-q");
  q.addEventListener("input", () => {
    trk.query = q.value;
    body.querySelector("#trk-board-wrap").innerHTML = trkBoard();
    trkBindCards(body, box);
  });
  body.querySelectorAll(".trk-prio-btn").forEach((b) => b.addEventListener("click", () => {
    trk.priority = b.dataset.prio;
    body.querySelectorAll(".trk-prio-btn").forEach((x) => {
      const on = x === b;
      x.classList.toggle("active", on);
      x.setAttribute("aria-pressed", String(on));
    });
    body.querySelector("#trk-board-wrap").innerHTML = trkBoard();
    trkBindCards(body, box);
  }));
  trkBindCards(body, box);
  if (openTaskId) {
    const match = trk.tasks.find((t) => t.id === openTaskId);
    if (match) trkOpenDetail(box, match);
  }
}

function trkFiltered() {
  const q = trk.query.trim().toLowerCase();
  return trk.tasks.filter((t) => {
    if (trk.priority !== "all" && (t.priority || "").toLowerCase() !== trk.priority) return false;
    if (!q) return true;
    return (t.title || "").toLowerCase().includes(q) ||
      (t.id || "").toLowerCase().includes(q) ||
      (t.labels || []).some((l) => l.toLowerCase().includes(q));
  });
}

function trkBoard() {
  const tasks = trkFiltered();
  return `<div class="trk-board">` + TRK_COLUMNS.map((col) => {
    const cards = tasks.filter((t) => (t.board || "todo") === col.id).map(trkCard).join("");
    return `<section class="trk-col" aria-label="${esc(col.label)}">` +
      `<header><h2>${esc(col.label)}</h2><span class="trk-col-count">${tasks.filter((t) => (t.board || "todo") === col.id).length}</span></header>` +
      `<div class="trk-col-body">` + (cards || `<p class="trk-col-empty">empty</p>`) + `</div></section>`;
  }).join("") + `</div>`;
}

function trkCard(t) {
  const labels = (t.labels || []).map((l) => `<span class="trk-chip">${esc(l)}</span>`).join("");
  const pri = (t.priority || "").toLowerCase();
  const badge = pri ? `<span class="trk-prio-badge pri-${esc(pri)}">${esc(t.priority)}</span>` : "";
  const who = (t.assignees && t.assignees.length)
    ? `<span class="trk-avatar" title="${esc(t.assignees.join(", "))}">${esc(trkInitials(t.assignees[0]))}</span>`
    : `<span class="trk-unassigned">unassigned</span>`;
  const docs = (t.doc_refs && t.doc_refs.length)
    ? `<span class="trk-docrefs" title="${t.doc_refs.length} linked document(s)">▤ ${t.doc_refs.length}</span>` : "";
  const rel = trkRelative(t.updated_at);
  return `<button type="button" class="trk-card" data-id="${esc(t.id)}">` +
    `<span class="trk-card-top"><span class="trk-card-id">${esc(t.id)}</span>${badge}</span>` +
    `<span class="trk-card-title">${esc(t.title)}</span>` +
    (labels ? `<span class="trk-labels">${labels}</span>` : "") +
    `<span class="trk-card-foot"><span class="trk-foot-left">${who}${docs}</span>` +
    (rel ? `<span class="trk-rel">${esc(rel)}</span>` : "") + `</span></button>`;
}

function trkBindCards(body, box) {
  body.querySelectorAll(".trk-card").forEach((el) => {
    el.addEventListener("click", () => {
      const match = trk.tasks.find((t) => t.id === el.dataset.id);
      if (match) trkOpenDetail(box, match);
    });
  });
}

function trkSyncUrl(taskId) {
  const q = new URLSearchParams(location.search);
  q.set("provider", trk.provider);
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
  overlay.innerHTML =
    `<div class="trk-dialog" role="dialog" aria-modal="true" aria-label="Task detail">` +
    `<button type="button" class="trk-backdrop" aria-label="Close details" data-close></button>` +
    `<aside class="trk-panel"><header>` +
    `<span class="trk-panel-id"><span class="trk-dot ${dot}" aria-hidden="true"></span><span class="trk-mono">${esc(it.id)}</span></span>` +
    `<span><button type="button" class="trk-iconbtn" data-close aria-label="Close">✕</button></span></header>` +
    `<div class="trk-panel-body"><h2>${esc(it.title)}</h2>` +
    `<dl class="trk-dl">` +
    `<dt>Status</dt><dd><span class="trk-dot ${dot}" aria-hidden="true"></span>${esc(it.status)}</dd>` +
    (it.priority ? `<dt>Priority</dt><dd><span class="trk-prio-badge pri-${esc(pri)}">${esc(it.priority)}</span></dd>` : "") +
    `<dt>Assignee</dt><dd>${who}</dd>` +
    (labels ? `<dt>Labels</dt><dd><span class="trk-labels">${labels}</span></dd>` : "") +
    `<dt>Move to…</dt><dd><span class="trk-move"><select id="trk-status" class="project-select">` +
    statuses.map((s) => `<option value="${esc(s)}"${s === it.status ? " selected" : ""}>${esc(s)}</option>`).join("") +
    `</select> <button class="btn" id="trk-apply" type="button">Apply</button> <span class="muted" id="trk-msg"></span></span></dd>` +
    `</dl>` +
    (it.description ? `<h3>Description</h3><pre class="trk-desc">${esc(it.description)}</pre>` : "") +
    (docs ? `<h3>Linked documents</h3><ul class="trk-docs">${docs}</ul>` : "") +
    `</div><footer><span class="trk-mono">` +
    (it.created_at ? `created ${esc(trkRelative(it.created_at))} · ` : "") +
    (it.updated_at ? `updated ${esc(trkRelative(it.updated_at))}` : "") +
    `</span></footer></aside></div>`;
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
  const onKey = (e) => { if (e.key === "Escape") close(); };
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
    `<section class="doc-file"><h3>change.md</h3><pre class="doc-md">${esc(d.change_md)}</pre></section>` +
    (d.tasks_md ? `<section class="doc-file"><h3>tasks.md</h3><pre class="doc-md">${esc(d.tasks_md)}</pre></section>` : "") +
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
  return `<a href="/docs?change=${encodeURIComponent(m[1])}" data-docchange="${esc(m[1])}">${esc(ref)}</a>`;
}
