#!/usr/bin/env node
// session-start.js — inject the using-skillgrid router into every new session.
//
// The router (using-skillgrid) currently relies on its frontmatter description
// matching to be loaded. That is a soft guarantee: a harness that does not
// auto-load skills, or one whose router description changes, can start a
// session with the router out of context. This SessionStart hook closes that
// gap — it emits the router's routing rules as a priority-IMPORTANT message so
// the agent has them before its first response, in any session, in any repo.
//
// Harness: Claude Code (and any harness that reads a SessionStart hook's stdout
// as a JSON hook payload). Other harnesses can reuse the script; each wires it
// in its own settings file (see .claude/settings.json for the Claude Code form).
//
// Output contract: a single JSON object on stdout. On any error the hook degrades
// to an INFO note and exits 0 — it never blocks session start.
'use strict';

const fs = require('node:fs');
const path = require('node:path');

const ROOT = path.join(__dirname, '..');
const ROUTER = path.join(ROOT, '.agents', 'skills', 'using-skillgrid', 'SKILL.md');

const INTRO = 'skillgrid loaded. Route through the using-skillgrid flow before ANY response (including clarifying questions).';

try {
  if (fs.existsSync(ROUTER)) {
    const msg = fs.readFileSync(ROUTER, 'utf8');
    // Build the payload via JSON.stringify so escaping is exactly right.
    const payload = { hookSpecificName: 'SessionStart', priority: 'IMPORTANT', message: `${INTRO}\n\n${msg}` };
    process.stdout.write(`${JSON.stringify(payload)}\n`);
  } else {
    const payload = {
      hookSpecificName: 'SessionStart',
      priority: 'INFO',
      message: `skillgrid: using-skillgrid router not found at ${ROUTER} - skills may still be available individually.`,
    };
    process.stdout.write(`${JSON.stringify(payload)}\n`);
  }
} catch {
  const payload = {
    hookSpecificName: 'SessionStart',
    priority: 'INFO',
    message: `skillgrid: using-skillgrid router not found at ${ROUTER} - skills may still be available individually.`,
  };
  process.stdout.write(`${JSON.stringify(payload)}\n`);
}
process.exit(0);
