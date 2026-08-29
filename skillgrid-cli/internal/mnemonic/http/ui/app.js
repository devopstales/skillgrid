"use strict";
const $ = (s) => document.querySelector(s);
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
let project = localStorage.getItem("sgmn-project") || "";

function pill(kind, text) {
  return `<span class="pill ${kind}">${esc(text)}</span>`;
}
function projectQS() {
  return project ? "?project=" + encodeURIComponent(project) : "";
}
async function api(url, opts) {
  const r = await fetch(url, { headers: { "Content-Type": "application/json" }, ...opts });
  const text = await r.text();
  let body = null;
  try { body = JSON.parse(text); } catch { body = text; }
  if (!r.ok) throw new Error(body && body.error ? body.error : r.status + " " + r.statusText);
  return body;
}

function setTab(name) {
  document.querySelectorAll(".tabs button").forEach((b) => b.classList.toggle("active", b.dataset.tab === name));
  ["mem", "code", "web"].forEach((t) => $("#tab-" + t).classList.toggle("hidden", t !== name));
}
document.querySelectorAll(".tabs button").forEach((b) => b.addEventListener("click", () => setTab(b.dataset.tab)));
$("#reload").addEventListener("click", loadAll);

async function loadProjects() {
  const box = $("#project");
  let ids = [];
  try {
    const r = await api("/projects");
    ids = r.projects || [];
  } catch (e) {
    $("#proj-error").textContent = "Failed to list projects: " + e.message;
    $("#proj-error").classList.remove("hidden");
    return;
  }
  $("#proj-error").classList.add("hidden");
  box.innerHTML = "";
  if (!ids.length) {
    const o = document.createElement("option");
    o.textContent = "(no project stores found — save an observation first)";
    box.appendChild(o);
    return;
  }
  for (const id of ids) {
    const o = document.createElement("option");
    o.value = id; o.textContent = id;
    if (id === project) o.selected = true;
    box.appendChild(o);
  }
  if (!project) project = ids[0];
  box.value = project;
  localStorage.setItem("sgmn-project", project);
}
$("#project").addEventListener("change", (e) => {
  project = e.target.value;
  localStorage.setItem("sgmn-project", project);
  loadAll();
});

async function loadStatus() {
  const box = $("#status-cards");
  if (!project) { box.innerHTML = '<div class="card"><div class="muted">Pick a project to load status.</div></div>'; return; }
  const [health, mem, code, web] = await Promise.all([
    api("/health"),
    api("/memory/status" + projectQS()),
    api("/code/status" + projectQS()),
    api("/web/status" + projectQS()),
  ]);
  const memPill = mem.observation_count > 0 ? pill("ok", mem.observation_count + " obs") : pill("warn", "empty");
  const codePill = code.stale ? pill("warn", "stale") : pill("ok", "fresh");
  const webPill = web.expired_entries > 0 ? pill("warn", web.expired_entries + " expired") : pill("ok", "no expired");
  const byType = Object.entries(mem.by_type || {}).map(([k, v]) => `<span class="badge">${esc(k)} ${v}</span>`).join("");
  const bySource = Object.entries(web.by_source || {}).map(([k, v]) => `<span class="badge">${esc(k)} ${v}</span>`).join("");
  box.innerHTML = `
    <div class="card"><div class="k">Memory ${memPill}</div><div class="v">${mem.observation_count}<sub> observations</sub></div><div>${byType || '<span class="muted">none</span>'}</div><div class="muted">${mem.active_sessions}<sub> active</sub> / ${mem.total_sessions}<sub> sessions</sub> · ${esc(mem.newest_created || "no saves")}</div></div>
    <div class="card"><div class="k">Code index ${codePill}</div><div class="v">${code.file_count}<sub> files</sub> / ${code.chunk_count}<sub> chunks</sub></div><div class="muted">${esc(code.last_indexed || "never indexed")}</div></div>
    <div class="card"><div class="k">Web cache ${webPill}</div><div class="v">${web.total_entries}<sub> entries</sub></div><div>${bySource || '<span class="muted">empty</span>'}</div><div class="muted">${esc(web.oldest_fetch || "—")} … ${esc(web.newest_fetch || "—")}</div></div>`;
  $("#health-info").textContent = health.service + " v" + health.version;
}

async function loadMem() {
  if (!project) return;
  const box = $("#mem-sessions");
  try {
    const r = await api("/context?limit=10&project=" + encodeURIComponent(project));
    const rows = (r.sessions || []).map((s) => {
      const label = s.title || (s.summary ? s.summary.split("\n")[0] : "(unnamed)");
      return `
      <div class="row" title="${esc(s.title || label)}"><span class="t">${esc(label)}</span>
      <span class="m">${esc(s.started_at)} · ${esc(s.status)}</span></div>`;
    }).join("");
    box.innerHTML = rows ? '<div class="list">' + rows + "</div>" : '<div class="list"><div class="empty">No sessions yet.</div></div>';
    await renderOBSList([]);
  } catch (e) { box.innerHTML = `<div class="error">${esc(e.message)}</div>`; }
}

async function renderOBSList(items) {
  const box = $("#mem-list");
  if (!items.length) { box.innerHTML = '<div class="list"><div class="empty">No observations. Use mem_save / POST /observations to add one.</div></div>'; return; }
  box.innerHTML = '<div class="list">' + items.map((o) => `
    <div class="row" data-id="${o.id}">
      <span class="t">${pill("dim", o.type)} ${esc(o.title)}</span>
      <span class="m">${o.topic_key ? esc(o.topic_key) + " · " : ""}${esc(o.created_at)}</span>
    </div>`).join("") + "</div>";
  box.querySelectorAll(".row").forEach((el) => el.addEventListener("click", () => showObs(el.dataset.id)));
}

async function showObs(id) {
  try {
    const o = await api("/observations?limit=500" + projectQS());
    const obs = (o.observations || []).find((x) => x.id == id);
    if (!obs) throw new Error("observation not in recent list");
    const d = $("#mem-detail");
    d.innerHTML = `
      <div class="muted">${pill("dim", obs.type)} ${obs.scope ? pill("dim", obs.scope) : ""} ${obs.topic_key ? pill("dim", obs.topic_key) : ""} r${obs.revision_count}</div>
      <div class="muted" style="margin:6px 0">${esc(obs.title)} · ${esc(obs.created_at)} → ${esc(obs.updated_at)}</div>
      <pre>${esc(obs.content)}</pre>`;
    d.classList.remove("hidden");
  } catch (e) { $("#mem-detail").innerHTML = `<div class="error">${esc(e.message)}</div>`; $("#mem-detail").classList.remove("hidden"); }
}

$("#mem-search-btn").addEventListener("click", async () => {
  const q = $("#mem-search").value.trim();
  if (!q) return;
  try {
    const r = await api("/search?project=" + encodeURIComponent(project) + "&query=" + encodeURIComponent(q));
    $("#mem-detail").classList.add("hidden");
    await renderOBSList(r.observations || []);
  } catch (e) { $("#mem-list").innerHTML = `<div class="error">${esc(e.message)}</div>`; }
});
$("#mem-reset").addEventListener("click", () => { $("#mem-search").value = ""; $("#mem-detail").classList.add("hidden"); loadMem(); });
["#mem-search", "#code-q", "#web-q"].forEach((s) => $(s).addEventListener("keydown", (e) => {
  if (e.key === "Enter") {
    if (s === "#mem-search") $("#mem-search-btn").click();
    if (s === "#code-q") $("#code-search").click();
    if (s === "#web-q") $("#web-search").click();
  }
}));

async function loadCodeFiles() {
  if (!project) return;
  try {
    const r = await api("/code/files?project=" + encodeURIComponent(project));
    const inp = $("#code-paths");
    const list = r.files || [];
    if (!list.length) { inp.setAttribute("placeholder", "no indexed files — run skillgrid index"); return; }
    inp.addEventListener("input", function autocomplete() {
      const v = inp.value.toLowerCase();
      const d = document.createElement("datalist");
      inp.removeAttribute("list");
      const filtered = v ? list.filter((p) => p.toLowerCase().includes(v)).slice(0, 50) : list.slice(0, 50);
      d.id = "paths-list";
      filtered.forEach((p) => { const o = document.createElement("option"); o.value = p; d.appendChild(o); });
      if (!document.getElementById("paths-list")) document.body.appendChild(d);
      else document.getElementById("paths-list").replaceWith(d);
      inp.setAttribute("list", "paths-list");
    });
  } catch (e) { /* ignore */ }
}

$("#code-search").addEventListener("click", async () => {
  const q = $("#code-q").value.trim();
  if (!q) return;
  const box = $("#code-hits");
  box.innerHTML = '<div class="list"><div class="empty">Searching…</div></div>';
  try {
    const r = await api("/code/search?project=" + encodeURIComponent(project) + "&query=" + encodeURIComponent(q));
    const hits = r.hits || [];
    if (!hits.length) { box.innerHTML = '<div class="list"><div class="empty">No matches.</div></div>'; return; }
    box.innerHTML = '<div class="list">' + hits.map((h) => `
      <div class="row" data-path="${esc(h.path)}" data-start="${h.start_line}" data-end="${h.end_line}">
        <span class="t"><span style="font-family:var(--mono);font-size:13px">${esc(h.path)}:${h.start_line}–${h.end_line}</span></span>
        <span class="m">${esc(h.snippet).slice(0, 80)}</span></div>`).join("") + "</div>";
    box.querySelectorAll(".row").forEach((el) => el.addEventListener("click", () => readCode(el.dataset.path, el.dataset.start, el.dataset.end)));
  } catch (e) { box.innerHTML = `<div class="error">${esc(e.message)}</div>`; }
});

async function readCode(path, start, end) {
  const box = $("#code-detail");
  box.innerHTML = '<div class="muted" style="margin:8px 0">Loading…</div>';
  try {
    let url = "/code/read?project=" + encodeURIComponent(project) + "&path=" + encodeURIComponent(path);
    if (start) url += "&start_line=" + start;
    if (end) url += "&end_line=" + end;
    const r = await api(url);
    box.innerHTML = `
      <div class="muted" style="margin:8px 0">${esc(r.path)}:${r.start_line}–${r.end_line}</div>
      <pre>${esc(r.text)}</pre>`;
  } catch (e) { box.innerHTML = `<div class="error">${esc(e.message)}</div>`; }
}

$("#code-read").addEventListener("click", async () => {
  const path = $("#code-paths").value.trim();
  if (!path) return;
  const lines = $("#code-lines").value.trim();
  let start = "", end = "";
  const m = lines.match(/^(\d+)?\s*[:\-]\s*(\d+)?$/);
  if (m) { start = m[1] || ""; end = m[2] || ""; }
  if (start && !end) end = start;
  $("#code-paths").value = path;
  await readCode(path, start, end);
});

$("#web-search").addEventListener("click", async () => {
  const q = $("#web-q").value.trim();
  if (!q) return;
  const box = $("#web-hits");
  let url = "/web/search?project=" + encodeURIComponent(project) + "&query=" + encodeURIComponent(q);
  const src = $("#web-source").value;
  if (src) url += "&source=" + encodeURIComponent(src);
  url += "&fresh_only=" + (!$("#web-stale").checked);
  box.innerHTML = '<div class="list"><div class="empty">Searching…</div></div>';
  try {
    const r = await api(url);
    const hits = r.entries || [];
    if (!hits.length) { box.innerHTML = '<div class="list"><div class="empty">No cached entries match.</div></div>'; return; }
    box.innerHTML = '<div class="list">' + hits.map((h) => `
      <div class="row" data-id="${h.id}">
        <span class="t">${pill("dim", h.source)} ${esc(h.title || h.url || h.library_id || ("entry " + h.id))}</span>
        <span class="m">${esc(h.fetched_at)}${h.expires_at ? " → " + esc(h.expires_at) : ""}</span></div>`).join("") + "</div>";
    box.querySelectorAll(".row").forEach((el) => el.addEventListener("click", () => showWeb(el.dataset.id)));
  } catch (e) { box.innerHTML = `<div class="error">${esc(e.message)}</div>`; }
});

async function showWeb(id) {
  const box = $("#web-detail");
  box.innerHTML = '<div class="muted" style="margin:8px 0">Loading snapshot…</div>';
  try {
    const e = await api("/web/entry/" + encodeURIComponent(id) + "?project=" + encodeURIComponent(project));
    const fresh = e.expires_at && new Date(e.expires_at).getTime() > Date.now();
    box.innerHTML = `
      <div class="muted" style="margin:8px 0">${pill("dim", e.source)} ${fresh ? pill("ok", "fresh") : pill("warn", "stale")} ${e.url ? '· <a href="' + esc(e.url) + '" target="_blank" rel="noopener">' + esc(e.url) + "</a>" : ""}</div>
      ${e.title ? '<div class="muted" style="margin:6px 0">' + esc(e.title) + " · fetched " + esc(e.fetched_at) + "</div>" : ""}
      <pre>${esc(e.content || "(no content stored)")}</pre>`;
  } catch (err) { box.innerHTML = `<div class="error">${esc(err.message)}</div>`; }
}

async function loadAll() {
  if (!project) return;
  await Promise.allSettled([loadStatus(), loadMem(), loadCodeFiles()]);
}

(async () => {
  await loadProjects();
  if (project) loadAll();
})();
