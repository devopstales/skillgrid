/**
 * skillgrid-parse.mjs — shared, deterministic parsers for the skillgrid SDD
 * spec zone. Single source of truth for phase derivation, marker parsing,
 * ledger parsing, and verification-scope atoms, so the drift guard, the
 * phase router, and the resume skill all agree on the same facts.
 *
 * Per `_shared/craft/deterministic-boundary.md`: the derivation that is a pure
 * function of the files on disk lives here, not in skill prose. Each consumer
 * (state-drift-check.mjs, phase-router.mjs) owns only its own verdict/IO.
 */

import { readFileSync, existsSync, readdirSync } from "node:fs";
import { join } from "node:path";

// --- verification-scope.md atoms (worst-scope-wins) ---
// Severity: UNREADABLE > UNSCOPED > TRUNCATED > COMPLETE.
export const SCOPE_RANK = { COMPLETE: 0, TRUNCATED: 1, UNSCOPED: 2, UNREADABLE: 3 };

export function worstScope(a, b) {
  if (!a) return b || "COMPLETE";
  if (!b) return a;
  return SCOPE_RANK[b] > SCOPE_RANK[a] ? b : a;
}

// --- I/O defaults (pure logic above; these are the thin fs helpers) ---

export function readDirFile(dir, file) {
  try {
    return readFileSync(join(dir, file), "utf8");
  } catch {
    return null;
  }
}

export function pathExists(p) {
  try {
    return existsSync(p);
  } catch {
    return false;
  }
}

/** List immediate subdirectory names of a dir (empty on error). */
export function listSubdirs(dir) {
  try {
    return readdirSync(dir, { withFileTypes: true })
      .filter((e) => e.isDirectory())
      .map((e) => e.name);
  } catch {
    return [];
  }
}

// --- Phase derivation (the canonical sdd-structure.md table) ---
// The deepest artifact present in the change dir determines the phase. This is
// the single source of truth for routing; state.yaml's phase is a cache, not
// the source.
//
// NOTE: extends the old 5-state derivePhase (spec|blueprint|slicing|execution|qa)
// with the tail states the sdd-structure.md table defines (reflect, complete).
// `reflect` / `complete` require the change dir's location (specs/ vs xarchive/),
// which is why they take an `archived` flag.

const REVIEW_FLOOR_MET = /^(met|met-with-fixes)\b/i;

/**
 * Derive the phase for one change dir.
 * @param {object} opts
 * @param {string} opts.changeDir   abs path to the change dir (specs/<slug> or xarchive/<slug>)
 * @param {string} [opts.sddDir]    abs path to .skillgrid/sdd (for the execution signal)
 * @param {boolean} [opts.archived] true when the dir is under xarchive/
 * @returns {{phase: string, marker: string}} phase ∈
 *   spec|blueprint|slicing|execution|qa|review|reflect|complete|unknown
 */
export function derivePhase(opts) {
  const { changeDir, sddDir = null, archived = false } = opts;
  const has = (f) => pathExists(join(changeDir, f));
  const read = (f) => readDirFile(changeDir, f);

  const briefing = has("briefing.md");
  const report = has("report.md");
  const reportText = report ? read("report.md") : null;
  const verdict = verdictFromReport(reportText);

  // Archived (xarchive/): the cycle tail lives here.
  if (archived) {
    if (retroFilled(reportText)) return { phase: "complete", marker: "xarchive + retro filled" };
    return { phase: "reflect", marker: report && verdict ? "xarchive + verdict, retro empty" : "xarchive (retro empty)" };
  }

  if (!briefing) return { phase: "unknown", marker: "no briefing.md" };

  // Verdict filled + review.md floor met, still under specs/ → review.
  if (report && verdict) {
    if (has("review.md")) {
      const floor = reviewFloor(read("review.md"));
      if (floor && REVIEW_FLOOR_MET.test(floor)) {
        return { phase: "review", marker: "report verdict + review floor met" };
      }
    }
    // Verdict filled but review floor not met → qa/review handoff.
    return { phase: "qa", marker: "report verdict present, review floor not met" };
  }

  // report.md exists (QA half, Test Plan written, verdict not yet) → qa.
  if (report) return { phase: "qa", marker: "report.md present (QA half)" };

  if (has("tasks.md")) {
    if (executionSignal(changeDir, sddDir)) {
      return { phase: "execution", marker: "tasks.md + execution signal" };
    }
    return { phase: "slicing", marker: "tasks.md (no execution signal)" };
  }

  if (has("blueprint.md")) return { phase: "blueprint", marker: "blueprint.md" };
  return { phase: "spec", marker: "briefing.md" };
}

/** True when tasks.md or a sdd/<plan>/progress.md carries an [x] mark. */
export function executionSignal(changeDir, sddDir) {
  if (sddDir && pathExists(sddDir)) {
    for (const entry of listSubdirs(sddDir)) {
      const progress = join(sddDir, entry, "progress.md");
      if (pathExists(progress)) {
        const content = readDirFile(sddDir, join(entry, "progress.md"));
        if (/\[x\]/i.test(content || "")) return true;
      }
    }
  }
  const tasks = readDirFile(changeDir, "tasks.md");
  return /\[x\]/i.test(tasks || "");
}

/** Parse the Gate Decision verdict out of report.md. Null when absent/placeholder. */
export function verdictFromReport(text) {
  if (!text) return null;
  // `**Verdict:** <word>` — take only the first word so a template placeholder
  // (`PASS / CONCERNS / FAIL / WAIVED`) is not mistaken for a filled verdict.
  let m = text.match(/\*\*Verdict:\*\*\s*([A-Za-z]+)/i);
  if (m) {
    const v = normalizeVerdict(m[1]);
    if (v) return v;
  }
  // Real-world variant: a bare `**CONCERNS**` line under ## Gate Decision.
  const section = gateDecisionSection(text);
  if (section) {
    m = section.match(/\*\*(PASS|CONCERNS|FAIL|WAIVED)\*\*/i);
    if (m) return normalizeVerdict(m[1]);
  }
  return null;
}

export function gateDecisionSection(text) {
  const idx = text.search(/^##\s+Gate Decision/m);
  if (idx < 0) return null;
  const rest = text.slice(idx);
  const next = rest.slice(1).search(/^##\s/m);
  return next < 0 ? rest : rest.slice(0, next + 1);
}

export function normalizeVerdict(v) {
  const u = String(v).toUpperCase();
  return u === "PASS" || u === "CONCERNS" || u === "FAIL" || u === "WAIVED" ? u : null;
}

/** The review.md `## Verdict` → `**Floor (decides):** <value>` token. */
export function reviewFloor(text) {
  if (!text) return null;
  const m = text.match(/\*\*Floor \(decides\):\*\*\s*([A-Za-z-]+(?:\s[A-Za-z-]+)?)/);
  return m ? m[1].trim() : null;
}

/**
 * The reflect-completed signal: a chosen `## Acceptance Verdict` verdict line,
 * or non-empty `## Lessons` / `## Decisions` tables.
 */
export function retroFilled(text) {
  if (!text) return false;
  const av = sectionBetween(text, "Acceptance Verdict");
  if (av && /\*\*Verdict:\*\*\s*(accepted|rejected)/i.test(av)) return true;
  const lessons = sectionBetween(text, "Lessons");
  if (lessons && tableHasDataRows(lessons)) return true;
  const decisions = sectionBetween(text, "Decisions");
  if (decisions && tableHasDataRows(decisions)) return true;
  return false;
}

function sectionBetween(text, heading) {
  // From `## <heading>` to the next `## ` (level 2) or end of string.
  const lines = text.split("\n");
  const start = lines.findIndex((l) => new RegExp(`^##\\s+${heading}\\b`).test(l));
  if (start < 0) return null;
  const out = [];
  for (let i = start + 1; i < lines.length; i++) {
    if (/^##\s/.test(lines[i])) break;
    out.push(lines[i]);
  }
  return out.join("\n");
}

function tableHasDataRows(section) {
  // A data row is a `|` line after the header row, that is not a separator and
  // has at least one non-empty cell. The first `|` line is the header (ignored);
  // the second is the separator (ignored). Everything after is data.
  const pipeLines = section.split("\n").map((l) => l.trim()).filter((l) => l.startsWith("|"));
  if (pipeLines.length < 3) return false; // header + separator only (or fewer)
  const data = pipeLines.slice(2);
  for (const t of data) {
    if (/^\|[\s:|-]+\|$/.test(t)) continue; // separator row
    const cells = t.split("|").slice(1, -1).map((c) => c.trim());
    if (cells.length && cells.some((c) => c && !/^[-: ]+$/.test(c))) return true;
  }
  return false;
}

// --- STATUS banner (sdd-structure.md § STATUS Banner) ---
// `> **STATUS:** \`<value>\` (YYYY-MM-DD)` — value lowercase alnum+hyphen.
export const STATUS_RE = /^>\s*\*\*STATUS:\*\*\s*`?([a-z][a-z0-9-]+)`?\s*(?:\((\d{4}-\d{2}-\d{2})\))?/;
export const STATUS_VALUES = new Set([
  "draft", "sliced", "in-progress", "complete", "shipped",
  "stalled", "blocked", "revised", "superseded",
]);

/** Parse the STATUS banner from a briefing.md / tasks.md string. */
export function parseStatusBanner(text) {
  if (!text) return null;
  const lines = text.split("\n");
  for (let i = 0; i < Math.min(5, lines.length); i++) {
    const m = lines[i].match(STATUS_RE);
    if (m) return { value: m[1], date: m[2] || null };
  }
  return null;
}

// --- Execution ledger (progress.md) ---
// A done task: `Task <N>: complete` (state word case-insensitive).
export function parseLedger(text) {
  const done = new Set();
  const inFlight = [];
  if (!text) return { done: [], inFlight, next: null };
  for (const line of text.split("\n")) {
    const m = line.match(/Task\s+(\d+)\s*:\s*([a-z-]+)/i);
    if (!m) continue;
    const n = parseInt(m[1], 10);
    const state = m[2].toLowerCase();
    if (state === "complete" || state === "done") {
      done.add(n);
    } else if (!inFlight.includes(n)) {
      inFlight.push(n);
    }
  }
  const all = [...new Set([...done, ...inFlight])].sort((a, b) => a - b);
  const next = all.find((n) => !done.has(n)) ?? null;
  return { done: [...done].sort((a, b) => a - b), inFlight, next };
}

// --- blocked(...) entries (sdd-structure.md § Blocked reasons) ---
export function parseBlocked(text) {
  if (!text) return [];
  const out = [];
  const re = /blocked\(([a-z0-9-]+)\):\s*(.+)/g;
  let m;
  while ((m = re.exec(text))) out.push({ phase: m[1], reason: m[2].trim() });
  return out;
}

// --- acceptance.feature scenario enumeration ---
// Scenarios come from `#### Scenario:` / `#### Scenario Outline:` headings;
// requirements from `### Requirement:` headings. Fences (```gherkin) hold steps
// only and are not scenarios. This is the same structure extract-gherkin.cjs
// consumes — one source of truth for "how many scenarios, named what."
export function parseScenarios(text) {
  if (!text) return [];
  const out = [];
  let requirement = null;
  for (const line of text.split("\n")) {
    const req = line.match(/^###\s+Requirement:\s*(.+?)\s*$/i);
    if (req) {
      requirement = req[1].trim();
      continue;
    }
    const sc = line.match(/^####\s+Scenario(?:\s+Outline)?:\s*(.+?)\s*$/i);
    if (sc) {
      out.push({ name: sc[1].trim(), requirement });
      continue;
    }
    // A `## ` (level 2) heading ends the current requirement scope; a `# `
    // (level 1) heading is the title.
    if (/^#{1,2}\s/.test(line)) requirement = null;
  }
  return out;
}

/** Normalize a scenario name for matching (casefold, collapse dashes/whitespace). */
export function normalizeName(s) {
  return String(s)
    .toLowerCase()
    .replace(/[—–-]/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

// --- report.md TDD Evidence table ---
/**
 * The `## TDD Evidence` table in a report.md → one row object per data row.
 * Column layouts vary between changes:
 *   | Ticket | RED commit | GREEN commit | Test file |   (faster-install)
 *   | Ticket | RED Test   | GREEN Test   | Commit    |   (hermes)
 * Either way the RED cell, GREEN cell, and an optional test-file / commit cell
 * are extracted by header name. A cell that does not exist is null.
 * Returns [] when there is no TDD Evidence section.
 */
export function parseTddEvidenceTable(reportText) {
  if (!reportText) return [];
  const section = sectionBetween(reportText, "TDD Evidence");
  if (!section) return [];
  const pipeLines = section
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l.startsWith("|"));
  if (pipeLines.length < 3) return [];
  const cells = (line) => line.split("|").slice(1, -1).map((c) => c.trim());
  const header = cells(pipeLines[0]).map((h) => h.toLowerCase());
  const redIdx = header.findIndex((h) => h.startsWith("red"));
  const greenIdx = header.findIndex((h) => h.startsWith("green"));
  const fileIdx = header.findIndex((h) => h.includes("file"));
  const commitIdx = header.findIndex((h) => h === "commit");
  const rows = [];
  for (let i = 2; i < pipeLines.length; i++) {
    const line = pipeLines[i];
    if (/^\|[\s:|-]+\|$/.test(line)) continue;
    const c = cells(line);
    if (!c.length || c.every((x) => !x)) continue;
    rows.push({
      ticket: c[0] || "",
      red: redIdx >= 0 ? c[redIdx] || null : null,
      green: greenIdx >= 0 ? c[greenIdx] || null : null,
      testFile: fileIdx >= 0 ? c[fileIdx] || null : null,
      commit: commitIdx >= 0 ? c[commitIdx] || null : null,
    });
  }
  return rows;
}

/**
 * Extract git commit hashes (7–40 hex chars) from a TDD table cell.
 * A cell may hold one hash, several comma/space-separated, `N/A (…)`, or
 * test names instead of hashes. Returns the hashes found.
 */
export function hashesFromCell(cell) {
  if (!cell) return [];
  return String(cell).match(/\b[0-9a-f]{7,40}\b/gi) || [];
}

/** A TDD cell that is `N/A` (any casing, optionally with a parenthetical). */
export function isNaCell(cell) {
  if (!cell) return false;
  return /^n\/?a\b/i.test(String(cell).trim());
}

// --- Next-skill routing (sdd-structure.md § Phase derivation / Resume markers) ---
// Derived phase → the skill that owns the next step. A `blocked(...)` entry
// overrides this with `resolve-blockers` (handled by the consumer).
export const NEXT_SKILL = {
  spec: "writing-blueprints",
  blueprint: "blueprint-validation",
  slicing: "slicing",
  execution: "qa",
  qa: "requesting-code-review",
  review: "receiving-code-review",
  reflect: "reflect",
  complete: null,
  unknown: "resume",
};

export function nextSkillFor(phase) {
  return Object.prototype.hasOwnProperty.call(NEXT_SKILL, phase) ? NEXT_SKILL[phase] : null;
}
