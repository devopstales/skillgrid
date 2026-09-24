#!/usr/bin/env node
// gate-lint.js — advisory static linter for acceptance.feature gate blocks.
//
// Mirrors the useful subset of unlazy's gate-lint.mjs: flags structural problems
// an agent can fix in the spec before committing, WITHOUT executing anything.
// This is a linter, not a gate: it emits warnings and exits 0. It rides the
// pre-commit dispatcher (checkpoint-state.js guard) so a malformed gate is
// surfaced at commit time, not at Stop time.
//
// Flags (per gate / requirement):
//   - a runnable gate missing CHECK or EXPECT (one present, one absent)
//   - a CHECK present but blank, or an EXPECT present but blank
//   - an EXPECT that looks like a tautology (contains only a digit/word that
//     also appears verbatim in the CHECK echo — heuristic, low-confidence)
//   - an ABANDON with a blank reason
//   - a requirement that declares a happy-path scenario but carries no gate
//     (a traceability gap — the same check gate-state reports as `missing`)
//   - a gate id that does not match G<n> (or G<n><letter>)
//
// Exit code: always 0 (warnings only). No spec / no gates -> silent.
'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

function git(cwd, ...args) {
  return spawnSync('git', ['-C', cwd, ...args], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
}

const ROOT = git(process.cwd(), 'rev-parse', '--show-toplevel').stdout.trim();
if (!ROOT) process.exit(0);

// Target: an explicit --spec, else any acceptance.feature in the staged diff,
// else the most-recently-modified one under the specs root.
let SPEC = '';
const argv = process.argv.slice(2);
for (let i = 0; i < argv.length; i++) {
  if (argv[i] === '--spec') SPEC = argv[++i] || '';
}
let SPECS_ROOT = '.skillgrid/specs';
const cfg = `${ROOT}/.skillgrid/config.yaml`;
if (fs.existsSync(cfg)) {
  for (const line of fs.readFileSync(cfg, 'utf8').split('\n')) {
    if (/^[ \t]*specs_dir:/.test(line)) {
      const v = line.replace(/^[^:]*:[ \t]*/, '').replace(/[ \t]*#.*$/, '').replace(/["']+/g, '');
      if (v) SPECS_ROOT = v;
    }
  }
}
if (!SPEC) {
  SPEC =
    git(ROOT, 'diff', '--cached', '--name-only').stdout
      .split('\n')
      .filter(Boolean)
      .find((f) => /\/acceptance\.feature$/.test(f)) || '';
}
if (!SPEC || !fs.existsSync(`${ROOT}/${SPEC}`)) {
  SPEC =
    git(ROOT, 'diff', '--name-only', 'HEAD').stdout
      .split('\n')
      .filter(Boolean)
      .find((f) => /\/acceptance\.feature$/.test(f)) || '';
}
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
if (!SPEC || !fs.existsSync(`${ROOT}/${SPEC}`)) process.exit(0);
const SPEC_ABS = `${ROOT}/${SPEC}`;

// Single pass: parse gates (id/kind/check/expect/abandon-reason) and
// per-requirement happy+gatecount, emitting warning lines.
const FENCE_RE = /^[ \t]*```(gherkin)?[ \t]*$/;
const REQ_RE = /^###[ \t]+Requirement:[ \t]*(.*)$/;
const SCEN_RE = /^####[ \t]+Scenario:/;
const GATE_RE = /^G[0-9]+[A-Za-z]*:/;

const warnings = [];
let inFence = false;
let cur = null; // open gate
let req = null; // { name, happy, ngates }
let reqName = '';

function flushGate() {
  if (!cur) return;
  const warn = (m) => warnings.push(m);
  if (!/^G[0-9]+[A-Za-z]*$/.test(cur.id)) warn(`gate id must match G<n> (line: ${cur.id})`);
  if (cur.abandoned) {
    // reason is the text after "ABANDON" on the G<n>: line
    if (/^[ \t]*$/.test(cur.reason)) warn(`ABANDON ${cur.id} needs a non-blank reason`);
  } else if (cur.hasCheck !== cur.hasExpect) {
    warn(`gate ${cur.id} is runnable but missing ${cur.hasCheck ? 'EXPECT' : 'CHECK'}`);
  } else if (cur.hasCheck && (/^[ \t]*$/.test(cur.check) || /^[ \t]*$/.test(cur.expect))) {
    warn(`gate ${cur.id} has a blank CHECK or EXPECT`);
  }
  if (cur.hasExpect && /^[ \t]*[0-9]+[ \t]*$/.test(cur.expect) && cur.check.includes(cur.expect.trim())) {
    warn(`gate ${cur.id} EXPECT is a bare number that also appears in CHECK (tautology?)`);
  }
  cur = null;
}

function flushReq() {
  if (reqName !== '' && req && req.happy && req.ngates === 0) {
    warnings.push(`requirement "${reqName}" declares a happy-path scenario but has no G<n> gate (traceability gap)`);
  }
}

for (const raw of fs.readFileSync(SPEC_ABS, 'utf8').split('\n')) {
  const $0 = raw.replace(/\n$/, '');
  const reqm = $0.match(REQ_RE);
  if (reqm) {
    flushGate();
    flushReq();
    reqName = reqm[1].trim();
    req = { name: reqName, happy: false, ngates: 0 };
    continue;
  }
  if (SCEN_RE.test($0)) {
    if (/happy/.test($0) && req) req.happy = true;
    continue;
  }
  if (FENCE_RE.test($0)) {
    inFence = !inFence;
    continue;
  }
  if (inFence) continue;
  if (/^[ \t]*#/.test($0)) continue;
  if (GATE_RE.test($0)) {
    flushGate();
    const id = $0.match(/^G[0-9]+[A-Za-z]*/)[0];
    if (req) req.ngates++;
    const rest = $0.replace(/^G[0-9]+[A-Za-z]*:[ \t]*/, '');
    const abandoned = /^ABANDON([ \t]|$)/.test(rest);
    cur = {
      id,
      check: '',
      expect: '',
      abandoned,
      reason: abandoned ? rest.replace(/^ABANDON[ \t]*/, '') : '',
      hasCheck: false,
      hasExpect: false,
    };
    continue;
  }
  const cm = $0.match(/^[ \t]*CHECK:[ \t]*(.*)$/);
  if (cm && cur) {
    cur.check = cm[1].trim();
    cur.hasCheck = true;
    continue;
  }
  const em = $0.match(/^[ \t]*EXPECT:[ \t]*(.*)$/);
  if (em && cur) {
    cur.expect = em[1].trim();
    cur.hasExpect = true;
  }
}
flushGate();
flushReq();

for (const w of warnings) {
  process.stdout.write(`${SPEC}: ${w}\n`);
}
process.exit(0);
