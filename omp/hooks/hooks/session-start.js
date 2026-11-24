#!/usr/bin/env node
// session-start.js — inject the skillgrid router into every new session.
//
// The router (the `skillgrid` skill) relies on its frontmatter description
// matching to be loaded. That is a soft guarantee: a harness that does not
// auto-load skills, or a router description that changes, can start a session
// with the router out of context. This SessionStart hook closes that gap — it
// emits the router's routing rules as a priority-IMPORTANT message so the
// agent has them before its first response, in any session, in any repo.
//
// Harness: Claude Code (and any harness that reads a SessionStart hook's
// stdout as a JSON hook payload). Each host wires it in its own settings file.
//
// Output contract: a single JSON object on stdout. On any error the hook
// degrades to an INFO note and exits 0 — it never blocks session start.

'use strict';

const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');

const ROOT = path.join(os.homedir(), '.skillgrid');

// Candidate router paths, in priority order:
//  1. explicit override (tests, custom installs)
//  2. flat install:  ~/.skillgrid/.agents/skills/skillgrid/SKILL.md
//  3. categorized install: ~/.skillgrid/.agents/skills/<category>/skillgrid/SKILL.md
function findRouter() {
  if (process.env.SKILLGRID_ROUTER && fs.existsSync(process.env.SKILLGRID_ROUTER)) {
    return process.env.SKILLGRID_ROUTER;
  }
  const base = path.join(ROOT, '.agents', 'skills');
  const flat = path.join(base, 'skillgrid', 'SKILL.md');
  if (fs.existsSync(flat)) return flat;
  let categories = [];
  try {
    categories = fs.readdirSync(base, { withFileTypes: true });
  } catch {
    return null;
  }
  for (const entry of categories) {
    if (!entry.isDirectory() || entry.name.startsWith('.')) continue;
    const candidate = path.join(base, entry.name, 'skillgrid', 'SKILL.md');
    if (fs.existsSync(candidate)) return candidate;
  }
  return null;
}

const ROUTER = findRouter();

const INTRO =
  'skillgrid loaded. Route through the skillgrid router flow before ANY response (including clarifying questions).';

try {
  if (ROUTER) {
    const msg = fs.readFileSync(ROUTER, 'utf8');
    const payload = { hookSpecificName: 'SessionStart', priority: 'IMPORTANT', message: `${INTRO}\n\n${msg}` };
    process.stdout.write(`${JSON.stringify(payload)}\n`);
  } else {
    const payload = {
      hookSpecificName: 'SessionStart',
      priority: 'INFO',
      message: `skillgrid: router not found under ${path.join(ROOT, '.agents', 'skills')} (set SKILLGRID_ROUTER to override) - skills may still be available individually.`,
    };
    process.stdout.write(`${JSON.stringify(payload)}\n`);
  }
} catch {
  const payload = {
    hookSpecificName: 'SessionStart',
    priority: 'INFO',
    message: 'skillgrid: session-start hook degraded (router unreadable) - skills may still be available individually.',
  };
  process.stdout.write(`${JSON.stringify(payload)}\n`);
}
process.exit(0);
