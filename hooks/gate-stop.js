#!/usr/bin/env node
// gate-stop.js — the gate-level Stop agent hook (port of unlazy's stop-hook.mjs).
//
// The suite-level stop-tests.js only proves the whole test suite is green.
// This proves the stronger thing: every G<n> in the change's acceptance.feature
// is met, abandoned, or manual. "A rule asks; a hook guarantees" — the
// test-driven-verification completion rule ("no done-claim while any happy-path
// gate is unmet or abandoned") becomes a guarantee here: the agent cannot end
// its turn while a non-abandoned gate is unmet or a happy-path requirement is
// missing its gate.
//
// Decision (on the Stop event), after a fresh --reverify of the change's spec:
//   - any gate unmet OR any missing happy-path gate  -> BLOCK (exit 2), list them
//   - only abandoned gates remain                     -> ALLOW with HANDOFF note
//   - all met / manual / abandoned                    -> ALLOW
//   - no spec discoverable                            -> ALLOW (degrade, like stop-tests)
//
// Loop guard (unlazy's MAX_BLOCKS): if the agent is stopped repeatedly with the
// SAME resolved gate state and makes no progress, release after MAX_BLOCKS to
// avoid trapping a session on a genuinely-impossible gate. The guard keys on a
// hash of the resolved state (met/unmet/abandoned/missing), NOT raw spec bytes —
// a comment edit must not re-arm it. State lives in .skillgrid/sdd/gate-stop-state.json.
//
// This is an AGENT harness hook (Stop event), not a git hook. The harness feeds
// session JSON on stdin; we read session_id for the guard key. Staged to
// ~/.skillgrid/git-hooks/ by `skillgrid install`. CLI-ready: skillgrid-cli
// rebinds it later.
'use strict';

const { spawnSync } = require('node:child_process');
const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');

function git(...args) {
  return spawnSync('git', args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

function readStdin() {
  try {
    return fs.readFileSync(0, 'utf8');
  } catch {
    return '';
  }
}

const MAX_BLOCKS = Number(process.env.SKILLGRID_GATE_MAX_BLOCKS || 6);
const SCRIPT_DIR = __dirname;
const GATE_STATE = path.join(SCRIPT_DIR, 'gate-state.js');

// Drain stdin (harness JSON) but capture session_id if present.
let sessionId = (readStdin().match(/"session_id"\s*:\s*"([^"]+)"/) || [])[1] || 'anonymous';

const ROOT = git('rev-parse', '--show-toplevel').stdout.trim();
if (!ROOT) process.exit(0);
const STATE_FILE = `${ROOT}/.skillgrid/sdd/gate-stop-state.json`;

// 24-char sha256, no external dependency.
function hash24(s) {
  return crypto.createHash('sha256').update(s).digest('hex').slice(0, 24);
}

function allow(msg) {
  if (msg) process.stderr.write(`${msg}\n`);
  process.exit(0);
}
function block(msg) {
  process.stderr.write(`${msg}\n`);
  process.exit(2);
}

// ---- fresh evidence ----------------------------------------------------------
const report = spawnSync(process.execPath, [GATE_STATE, '--reverify', '--root', ROOT], {
  encoding: 'utf8',
  stdio: ['ignore', 'pipe', 'pipe'],
}).stdout || '';
const summaryLines = report.split('\n').filter((l) => /^SUMMARY /.test(l));
const summaryLine = summaryLines[summaryLines.length - 1] || '';

if (!summaryLine || /^NO_SPEC/.test(report)) {
  allow(''); // no spec to verify against — degrade, don't trap
}

const sm = {};
for (const pair of summaryLine.replace(/^SUMMARY /, '').split(/\s+/).filter(Boolean)) {
  const [k, v] = pair.split('=');
  sm[k] = Number(v);
}
const met = sm.met || 0;
const unmet = sm.unmet || 0;
const abandoned = sm.abandoned || 0;
const manual = sm.manual || 0;
const missing = sm.missing || 0;

// Outstanding = unmet gates + missing happy-path requirements. Parse the
// per-gate lines, not the summary, so the block message names them.
const unmetList = report
  .split('\n')
  .filter((l) => /^[^=]+=unmet/.test(l))
  .map((l) => l.split('=unmet')[0]);
const missingList = report
  .split('\n')
  .filter((l) => /^REQ=.*=missing/.test(l))
  .map((l) => l.replace(/^REQ=/, '').split('=missing')[0]);

// ---- clean state: reset the guard, allow (with handoff if abandoned) --------
if (unmet === 0 && missing === 0) {
  try {
    fs.rmSync(STATE_FILE, { force: true });
  } catch {}
  if (abandoned > 0) {
    allow(`gate-stop: HANDOFF REQUIRED — ${abandoned} abandoned gate(s). Surface them in the completion report (never a pass).`);
  }
  allow('');
}

const outstanding = [...unmetList, ...missingList].filter(Boolean);
let outCount = unmet + missing;
if (outstanding.length === 0) outCount = 0;

// ---- loop guard: key on resolved state, not raw bytes ------------------------
// The guard accumulates per (session, repo) while the resolved gate state is
// UNCHANGED, and resets to 1 the moment the state changes (i.e. the agent made
// progress — a gate flipped to met, a new gate appeared, etc.). We therefore
// store BOTH the session-scoped block count and the progress hash it was
// measured against.
const sessionKey = hash24(`${sessionId}${ROOT}`);
const progressKey = hash24(
  report
    .split('\n')
    .filter((l) => /^(G[0-9]+[A-Za-z]*=|REQ=)/.test(l))
    .sort()
    .join('\n')
);

let blocks = 0;
let prevProgress = '';
try {
  const state = JSON.parse(fs.readFileSync(STATE_FILE, 'utf8'));
  const entry = state && state.sessions && state.sessions[sessionKey];
  if (entry) {
    prevProgress = entry.hash || '';
    blocks = entry.blocks || 0;
  }
} catch {}
// reset if the resolved state changed (progress), else accumulate
if (prevProgress && prevProgress !== progressKey) blocks = 0;
blocks += 1;

fs.mkdirSync(path.dirname(STATE_FILE), { recursive: true });
fs.writeFileSync(
  STATE_FILE,
  `${JSON.stringify(
    {
      schema: 'skillgrid/gate-stop/v1',
      sessions: { [sessionKey]: { hash: progressKey, blocks, updated: new Date().toISOString().replace(/\.\d+Z$/, 'Z') } },
    },
    null,
    2
  )}\n`
);

let list = outstanding.join('\n');
if (outstanding.length > 5) list = `${outstanding.slice(0, 5).join('\n')} (+${outstanding.length - 5} more)`;

if (blocks > MAX_BLOCKS) {
  allow(
    `gate-stop: releasing after ${MAX_BLOCKS} blocks without gate progress; ${outCount} item(s) remain (${list}). Use ABANDON <id> <reason> only when a gate is genuinely impossible — it surfaces as a handoff, never a pass.`
  );
}

block(
  `gate-stop: ${outCount} gate/requirement item(s) need work before you may stop: ${list}. Run 'node hooks/gate-state.js --reverify' to see fresh evidence. ABANDON <id> <reason> is terminal and non-successful — surface it as a handoff, never a pass.`
);
