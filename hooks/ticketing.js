#!/usr/bin/env node
// ticketing.js — shared library for the tracker-aware git hooks.
//
// Required with `require('./ticketing')` by ticketing-guard.js,
// ticketing-pre-push.js, ticketing-post-merge.js and checkpoint-state.js.
// No side effects on require; every helper fail-opens (returns a neutral
// value) on infra errors so a hook never blocks on a broken environment.
//
// Tracker selection comes from `.skillgrid/config.yaml` (`ticketing.enabled`,
// `ticketing.type`, `ticketing.backlogmd.dir`), mirroring the `ticketing`
// skill. Env overrides (checked first):
//   SKILLGRID_TICKETING_ENABLED=0|false  disable all tracker guards
//   SKILLGRID_TICKETING_TYPE=<type>      override ticketing.type
//   SKILLGRID_TRACKER_REF=<id>           active Tracker ID, consumed by the
//                                        prepare-commit-msg injector seam
// CLI-ready: pure functions over (root, config); a future binary rebind keeps
// the same contract.
'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

// Tracker ID shapes, one per supported `ticketing.type`. A footer value is
// valid when it contains at least one of these (a Backlog TASK-069 also
// matches the Jira shape — that is fine, any match counts as an ID).
const ID_PATTERNS = [
  /\bTASK-\d+(?:\.\d+)?\b/i, // backlogmd: TASK-069, TASK-019.03
  /#[1-9]\d*/, // gh / glab: #42
  /\b[A-Z][A-Z0-9]*-\d+\b/, // jira: PROJ-123 (uppercase only, to avoid matching prose)
];

// Footer keywords from the ticketing lifecycle (`Refs:` per commit, `Closes`
// on done). Singular GitHub variants accepted.
const FOOTER_RE = /^(Refs?|Closes?|Fixes?|Resolves?)\s*:\s*(.+?)\s*$/i;

// Closing keywords carry explicit close intent; a malformed one is a real
// defect (pre-push blocks it) while a malformed Refs is only a warning.
const CLOSING_KEYWORDS = new Set(['close', 'closes', 'fix', 'fixes', 'resolve', 'resolves']);

function git(cwd, ...args) {
  try {
    return spawnSync('git', args, { cwd: cwd || undefined, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
  } catch (_e) {
    return { status: 1, stdout: '', stderr: '' };
  }
}

function repoRoot(cwd) {
  const r = git(cwd, 'rev-parse', '--show-toplevel');
  if (r.status !== 0) return '';
  return (r.stdout || '').trim();
}

// Minimal YAML scan: reads only the keys the hooks need. Nested
// `backlogmd.dir` is matched at any indent; first match wins.
function readTicketingConfig(root) {
  const cfg = { enabled: true, type: 'backlogmd', backlogDir: '.backlog' };
  if (!root) return { enabled: false, type: '', backlogDir: '.backlog' };
  const file = path.join(root, '.skillgrid', 'config.yaml');
  if (fs.existsSync(file)) {
    const lines = fs.readFileSync(file, 'utf8').split('\n');
    // Two-pass: first find the ticketing: block, then read keys inside it.
    let inTicketing = false;
    let inBacklogmd = false;
    for (const line of lines) {
      if (/^\s*ticketing\s*:\s*(?:#.*)?$/.test(line)) { inTicketing = true; inBacklogmd = false; continue; }
      if (/^[A-Za-z_][A-Za-z0-9_-]*\s*:/.test(line)) {
        // A new top-level key ends the ticketing block (top-level keys start
        // at column 0; indented keys belong to the current block).
        if (/^[^ \t]/.test(line)) { inTicketing = false; inBacklogmd = false; continue; }
      }
      if (!inTicketing) continue;
      let kv = line.match(/^\s*enabled\s*:\s*(\S+)\s*(?:#.*)?$/);
      if (kv && !inBacklogmd) {
        cfg.enabled = !/^(false|no|off|0)$/i.test(kv[1].replace(/["']/g, ''));
        continue;
      }
      kv = line.match(/^\s*type\s*:\s*(\S+)\s*(?:#.*)?$/);
      if (kv && !inBacklogmd) {
        cfg.type = kv[1].replace(/["']/g, '');
        continue;
      }
      if (/^\s*backlogmd\s*:\s*(?:#.*)?$/.test(line)) { inBacklogmd = true; continue; }
      kv = line.match(/^\s*dir\s*:\s*(\S+)\s*(?:#.*)?$/);
      if (kv && inBacklogmd) {
        cfg.backlogDir = kv[1].replace(/["']/g, '').replace(/\/+$/, '') || '.backlog';
        inBacklogmd = false;
      }
    }
  }
  const envEnabled = process.env.SKILLGRID_TICKETING_ENABLED;
  if (envEnabled !== undefined) cfg.enabled = !/^(false|no|off|0)$/i.test(envEnabled);
  if (process.env.SKILLGRID_TICKETING_TYPE) cfg.type = process.env.SKILLGRID_TICKETING_TYPE;
  return cfg;
}

// Split a commit message into footer entries. Comment lines (#...) are
// ignored. Returns [{ keyword, value }] in source order.
function parseFooters(text) {
  const out = [];
  for (const raw of (text || '').split('\n')) {
    const line = raw.replace(/\s+$/, '');
    if (/^\s*#/.test(line) || line.trim() === '') continue;
    const m = line.match(FOOTER_RE);
    if (m) out.push({ keyword: m[1], value: m[2] });
  }
  return out;
}

function valueHasId(value) {
  return ID_PATTERNS.some((re) => re.test(value));
}

// Every footer present must reference a parseable Tracker ID. Returns the
// invalid entries (empty array = valid or no footers). Footers are never
// required — absence always passes.
function invalidFooters(text) {
  return parseFooters(text).filter((f) => !valueHasId(f.value));
}

function isClosingKeyword(keyword) {
  return CLOSING_KEYWORDS.has((keyword || '').toLowerCase());
}

// Staged files under a prefix (relative to the repo root), fail-open [].
function stagedFiles(root, prefix, diffFilter) {
  const r = git(root, 'diff', '--cached', '--name-only', `--diff-filter=${diffFilter || 'ACMR'}`);
  if (r.status !== 0) return [];
  const p = `${prefix.replace(/\/+$/, '')}/`;
  return (r.stdout || '').split('\n').map((s) => s.trim()).filter((s) => s && s.startsWith(p));
}

// Staged (index) content of a path, or null when unreadable (deleted,
// renamed-away, or git error) — callers skip nulls (fail-open).
function stagedContent(root, rel) {
  const r = git(root, 'show', `:${rel}`);
  if (r.status !== 0) return null;
  return r.stdout || '';
}

// Backlog completeness gate (see ticketing skill): type + references + DoD
// are FATAL when missing; the Implementation Plan is advisory (WARNING) —
// the `backlog` CLI itself creates plan-less triage tasks, so blocking on
// the plan would fight the CLI. Returns { fatals: [], warnings: [] }.
function checkBacklogTask(content) {
  const fatals = [];
  const warnings = [];
  const text = content || '';

  const fm = text.match(/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/);
  const front = fm ? fm[1] : '';
  if (!/^type:\s*\S+/m.test(front)) {
    fatals.push('missing frontmatter `type:` (e.g. `type: feature`)');
  }

  const refLine = front.match(/^references:\s*(.*)$/m);
  let refCount = 0;
  if (refLine) {
    if (/\[\s*\]/.test(refLine[1])) {
      refCount = 0;
    } else if (refLine[1].trim() !== '') {
      refCount = 1; // inline value
    } else {
      // Count list items until the next frontmatter key or blank+key.
      const after = front.slice(front.indexOf(refLine[0]) + refLine[0].length);
      for (const l of after.split('\n')) {
        if (/^\s*-\s*\S/.test(l)) { refCount += 1; continue; }
        if (/^\s*$/.test(l)) continue;
        break;
      }
    }
  }
  if (refCount === 0) {
    fatals.push('empty `references:` — list at least one path/URL (e.g. `- hooks/ticketing.js`)');
  }

  const dod = text.match(/<!--\s*DOD:BEGIN\s*-->([\s\S]*?)(?:<!--\s*DOD:END\s*-->|$)/);
  if (!dod) {
    fatals.push('missing `## Definition of Done` with `<!-- DOD:BEGIN -->` markers');
  } else if (!/^- \[[ xX]\]\s+\S/m.test(dod[1])) {
    fatals.push('empty Definition of Done — add checklist items (`- [ ] ...`)');
  }

  const plan = text.match(/<!--\s*SECTION:PLAN:BEGIN\s*-->([\s\S]*?)(?:<!--\s*SECTION:PLAN:END\s*-->|$)/);
  const planBody = plan ? plan[1].split('\n').map((l) => l.trim()).filter((l) => l && !/^<!--/.test(l)) : [];
  if (planBody.length === 0) {
    warnings.push('missing Implementation Plan (`<!-- SECTION:PLAN:BEGIN -->`) — seed it via `backlog task edit <ID> --plan ...`');
  }

  return { fatals, warnings };
}

module.exports = {
  ID_PATTERNS,
  CLOSING_KEYWORDS,
  git,
  repoRoot,
  readTicketingConfig,
  parseFooters,
  valueHasId,
  invalidFooters,
  isClosingKeyword,
  stagedFiles,
  stagedContent,
  checkBacklogTask,
};
