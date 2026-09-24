#!/usr/bin/env node
// gate-state.js — the gate state engine (JS port of unlazy's lib/gates.mjs).
//
// Parses a change's acceptance.feature `#### Gates` blocks and reports each
// G<n>'s state. skillgrid's format has NO checkbox (unlazy used `- [ ]`); a
// gate is met only when its CHECK command freshly exits 0 AND its output
// matches EXPECT — see test-driven-verification/SKILL.md ("the declared oracle,
// run fresh, is the oracle"). So the two modes:
//
//   --status    NON-executing. Report the ledger: which gates are runnable
//               (reported as `unmet (not run)`), which are manual, which
//               ABANDON, and whether any happy-path requirement is missing a
//               gate. Never runs a CHECK, writes nothing. Exit 0 always.
//   --reverify  Executing. Run every runnable CHECK, compare output to EXPECT,
//               report met / unmet / abandoned / manual / missing. This is the
//               fresh-evidence path the Stop hook uses.
//
// Scope: pass --spec <acceptance.feature>, or auto-discover. Discovery order:
//   1. an acceptance.feature touched in the current diff (git diff HEAD)
//   2. the most-recently-modified acceptance.feature under <specs_root>
// (checkpoint.json was a superseded session-events artifact; it is no longer read.)
// specs_root defaults to .skillgrid/specs (overridable via config bdd.specs_dir).
//
// Output (stdout), one line per gate / requirement, plus a summary:
//   G1=met   G2=unmet   G3=abandoned   G4=manual   REQ=<req>=missing
//   SUMMARY met=.. unmet=.. abandoned=.. manual=.. missing=..
// Exit code is 0 in both modes; gate-stop.js interprets the states.
'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

function git(cwd, ...args) {
  return spawnSync('git', ['-C', cwd, ...args], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

// ---- arg parsing -------------------------------------------------------------
let MODE = 'status';
let SPEC = '';
let SPECS_ROOT = '';
const TIMEOUT = Number(process.env.SKILLGRID_GATE_TIMEOUT || 120) * 1000;
let ROOT = '';

const argv = process.argv.slice(2);
for (let i = 0; i < argv.length; i++) {
  switch (argv[i]) {
    case '--status':
      MODE = 'status';
      break;
    case '--reverify':
      MODE = 'reverify';
      break;
    case '--spec':
      SPEC = argv[++i] || '';
      break;
    case '--specs-root':
      SPECS_ROOT = argv[++i] || '';
      break;
    case '--root':
      ROOT = argv[++i] || '';
      break;
    case '-h':
    case '--help':
      process.stdout.write('usage: gate-state.js [--status|--reverify] [--spec FILE] [--specs-root DIR] [--root DIR]\n');
      process.exit(0);
      break;
    default:
      process.stderr.write(`gate-state: unknown option ${argv[i]}\n`);
      process.exit(2);
  }
}

if (!ROOT) {
  ROOT = git(process.cwd(), 'rev-parse', '--show-toplevel').stdout.trim() || process.cwd();
}

// ---- config: bdd.specs_dir ---------------------------------------------------
function configSpecsDir() {
  const cfg = `${ROOT}/.skillgrid/config.yaml`;
  if (!fs.existsSync(cfg)) return '';
  for (const line of fs.readFileSync(cfg, 'utf8').split('\n')) {
    if (/^[ \t]*specs_dir:/.test(line)) {
      const v = line.replace(/^[^:]*:[ \t]*/, '').replace(/[ \t]*#.*$/, '').replace(/["']+/g, '');
      if (v) return v;
    }
  }
  return '';
}

if (!SPECS_ROOT) {
  SPECS_ROOT = configSpecsDir() || '.skillgrid/specs';
}

// ---- spec discovery ----------------------------------------------------------
// checkpoint.json was a superseded session-events artifact; it is no longer read.
if (!SPEC) {
  const diff = git(ROOT, 'diff', '--name-only', 'HEAD', '--').stdout.split('\n').filter(Boolean);
  SPEC = diff.find((f) => /\/acceptance\.feature$/.test(f)) || '';
  if (!SPEC || !fs.existsSync(`${ROOT}/${SPEC}`)) {
    const dir = path.join(ROOT, SPECS_ROOT);
    if (fs.existsSync(dir)) {
      let newest = '';
      let newestMtime = -1;
      (function walk(d) {
        for (const e of fs.readdirSync(d, { withFileTypes: true })) {
          const p = path.join(d, e.name);
          if (e.isDirectory()) walk(p);
          else if (e.name === 'acceptance.feature') {
            const m = fs.statSync(p).mtimeMs;
            if (m > newestMtime) {
              newestMtime = m;
              newest = p;
            }
          }
        }
      })(dir);
      if (newest) SPEC = path.relative(ROOT, newest);
    }
  }
}

if (!SPEC || !fs.existsSync(`${ROOT}/${SPEC}`)) {
  process.stdout.write('NO_SPEC\nSUMMARY met=0 unmet=0 abandoned=0 manual=0 missing=0\n');
  process.exit(0);
}
const SPEC_ABS = `${ROOT}/${SPEC}`;

const FENCE_RE = /^[ \t]*```(gherkin)?[ \t]*$/;
const REQ_RE = /^###[ \t]+Requirement:[ \t]*(.*)$/;
const SCEN_RE = /^####[ \t]+Scenario:/;
const GATE_RE = /^G[0-9]+[A-Za-z]*:/;
const GID_RE = /^G[0-9]+[A-Za-z]*/;
const CHECK_RE = /^[ \t]*CHECK:[ \t]*(.*)$/;
const EXPECT_RE = /^[ \t]*EXPECT:[ \t]*(.*)$/;

// Parse a feature file into records:
//   { kind: 'gate', id, line, check, expect, abandoned, reqIdx }
//   { kind: 'missing', req, reqIdx }  — requirement declared a happy-path
//                                       scenario but carries no gate.
function parseGates(text) {
  const lines = text.split('\n');
  const records = [];
  const reqs = []; // { happy, ngates }
  let inFence = false;
  let cur = null; // open gate record

  const flushGate = () => {
    if (cur) {
      const line = cur.abandoned ? 'ABANDON' : cur.check !== '' ? 'RUN' : 'MAN';
      records.push({ kind: 'gate', id: cur.id, line, check: cur.check, expect: cur.expect, reqIdx: cur.reqIdx });
      cur = null;
    }
  };

  let reqIdx = -1;
  lines.forEach((raw) => {
    const $0 = raw.replace(/\n$/, '');
    if (FENCE_RE.test($0)) {
      inFence = !inFence;
      flushGate();
      return;
    }
    if (inFence) return;
    if (/^[ \t]*#/.test($0)) return;

    const reqm = $0.match(REQ_RE);
    if (reqm) {
      flushGate();
      reqIdx = reqs.length;
      reqs.push({ happy: false, ngates: 0, name: reqm[1].trim() });
      return;
    }
    if (SCEN_RE.test($0)) {
      if (/happy/.test($0) && reqIdx >= 0) reqs[reqIdx].happy = true;
      return;
    }
    if (GATE_RE.test($0)) {
      flushGate();
      const id = $0.match(GID_RE)[0];
      const rest = $0.replace(/^G[0-9]+[A-Za-z]*:[ \t]*/, '');
      if (reqIdx >= 0) reqs[reqIdx].ngates++;
      const abandoned = /^ABANDON([ \t]|$)/.test(rest);
      cur = { id, check: '', expect: '', abandoned, reqIdx };
      return;
    }
    const cm = $0.match(CHECK_RE);
    if (cm && cur) {
      cur.check = cm[1].trim();
      return;
    }
    const em = $0.match(EXPECT_RE);
    if (em && cur) cur.expect = em[1].trim();
  });
  flushGate();

  // missing: happy-path requirement with no gate. Gate records were pushed in
  // source order; a requirement with no gates has no anchor, so its missing
  // entry lands in requirement order after all gates of prior requirements.
  reqs.forEach((r, i) => {
    if (r.happy && r.ngates === 0) records.push({ kind: 'missing', req: r.name, reqIdx: i });
  });
  return records;
}

// ---- CHECK execution ---------------------------------------------------------
// Run a CHECK command, bounded by a timer. Output (stdout+stderr) combined,
// exit code propagated.
function runCheck(cmd) {
  const r = spawnSync('bash', ['-c', cmd], { encoding: 'utf8', timeout: TIMEOUT, stdio: ['ignore', 'pipe', 'pipe'] });
  return { code: r.status === null ? 1 : r.status, output: `${r.stdout || ''}${r.stderr || ''}` };
}

// ---- EXPECT matching ---------------------------------------------------------
// Default: literal substring. A value wrapped in /.../ is a POSIX ERE; a
// trailing 'i' flag is honored best-effort.
function expectMatches(output, expect) {
  if (expect.length > 2 && expect.startsWith('/')) {
    const last = expect.lastIndexOf('/');
    if (last > 1) {
      const body = expect.slice(1, last);
      const flags = expect.slice(last + 1);
      if (/i/.test(flags)) return new RegExp(body, 'i').test(output);
      return new RegExp(body).test(output);
    }
  }
  return output.includes(expect);
}

// ---- classify ----------------------------------------------------------------
let met = 0;
let unmet = 0;
let abandoned = 0;
let manual = 0;
let missing = 0;
const OUT = [];

for (const r of parseGates(fs.readFileSync(SPEC_ABS, 'utf8'))) {
  if (r.kind === 'missing') {
    missing++;
    OUT.push(`REQ=${r.req}=missing`);
    continue;
  }
  switch (r.line) {
    case 'ABANDON':
      abandoned++;
      OUT.push(`${r.id}=abandoned`);
      break;
    case 'MAN':
      manual++;
      OUT.push(`${r.id}=manual`);
      break;
    case 'RUN':
      if (MODE === 'reverify' && r.check !== '') {
        const { code, output } = runCheck(r.check);
        if (code === 0 && (r.expect === '' || expectMatches(output, r.expect))) {
          met++;
          OUT.push(`${r.id}=met`);
        } else {
          unmet++;
          OUT.push(`${r.id}=unmet`);
        }
      } else {
        unmet++;
        OUT.push(`${r.id}=unmet (not run; --reverify to execute)`);
      }
      break;
  }
}

process.stdout.write(`${OUT.join('\n')}\n`);
process.stdout.write(`SUMMARY met=${met} unmet=${unmet} abandoned=${abandoned} manual=${manual} missing=${missing}\n`);
process.exit(0);
