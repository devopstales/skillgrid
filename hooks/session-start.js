#!/usr/bin/env node
// session-start.js — inject the using-skillgrid router into new sessions of
// skillgrid projects (omp SessionStart hook via @hsingjui/pi-hooks).
//
// Full-content variant of the router bootstrap: emits the whole using-skillgrid
// SKILL.md as a priority-IMPORTANT SessionStart message so the agent has the
// routing rules before its first response.
//
// Project-conditional (R3): injection fires only when a skillgrid project is
// detected — a `.skillgrid/config.yaml` exists at or above the session cwd.
// Fail-open (R4): any failure path emits an INFO note and exits 0; an unreadable
// config is treated as "no project"; session start never breaks and never shows a
// user-visible error.
//
// Main-agent only (R3): child/subagent sessions skip injection.
//
// Harness: Claude Code (and any harness that reads a SessionStart hook's stdout
// as a JSON hook payload). Other harnesses can reuse the script; each wires it in
// its own settings file (see .claude/settings.json for the Claude Code form).
//
// Output contract (pi-hooks v0.0.2 / Claude Code compatible): the injected
// context goes to stdout as `hookSpecificOutput.additionalContext`. INFO
// paths (not a project, child session, missing router) write a note to
// stderr and emit no stdout, so nothing is injected. On any error the hook
// degrades to an INFO note and exits 0 — it never blocks session start.
'use strict';

const fs = require('node:fs');
const path = require('node:path');

const ROOT = path.join(__dirname, '..');
const ROUTER = path.join(ROOT, '.agents', 'skills', 'using-skillgrid', 'SKILL.md');
const PROJECT_CONFIG = path.join('.skillgrid', 'config.yaml');

const INTRO = 'skillgrid loaded. Route through the using-skillgrid flow before ANY response (including clarifying questions).';
const INFO_NOT_FOUND = `skillgrid: using-skillgrid router not found at ${ROUTER} - skills may still be available individually.`;
const INFO_NOT_PROJECT = 'skillgrid: no skillgrid project detected (.skillgrid/config.yaml not found) - router injection skipped.';
const INFO_NOT_MAIN = 'skillgrid: session is not a main-agent session - router injection skipped.';

function emit(priority, message) {
  if (priority === 'INFO') {
    // No context to inject: keep the note on stderr for debugging, stdout
    // stays empty so pi-hooks appends nothing to the session context.
    process.stderr.write(`${message}\n`);
    return;
  }
  const payload = {
    hookSpecificOutput: {
      hookEventName: 'SessionStart',
      additionalContext: message,
    },
  };
  process.stdout.write(`${JSON.stringify(payload)}\n`);
}

// Project detection: walk up from the session cwd for `.skillgrid/config.yaml`,
// bounded at the filesystem root (path.resolve makes relative cwd absolute, so the
// walk always terminates). Mirrors the cursor hook's walk-up (TICKET-01 bound).
function isSkillgridProject(cwd) {
  let dir = path.resolve(cwd);
  for (;;) {
    const candidate = path.join(dir, PROJECT_CONFIG);
    if (fs.existsSync(candidate)) {
      // Fail-open: an unreadable config is treated as "no project".
      try {
        fs.accessSync(candidate, fs.constants.R_OK);
        return true;
      } catch {
        return false;
      }
    }
    const parent = path.dirname(dir);
    if (parent === dir) return false; // bounded at filesystem root
    dir = parent;
  }
}

function readStdinJson() {
  return new Promise((resolve) => {
    let raw = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', (c) => { raw += c; });
    process.stdin.on('end', () => {
      try { resolve(JSON.parse(raw || '{}')); } catch { resolve(null); }
    });
    process.stdin.on('error', () => resolve(null));
  });
}

function isChildSession(input) {
  if (!input || typeof input !== 'object') return false;
  const kinds = [input.agent_kind, input.agentType, input.agent_type]
    .filter((v) => typeof v === 'string')
    .map((v) => v.trim().toLowerCase())
    .filter(Boolean);
  if (kinds.length > 0) return !kinds.includes('main') && !kinds.includes('agent');
  const parent = input.parent_id ?? input.parentId ?? input.parent_session_id ?? input.parentSessionId ?? input.parentSession;
  if (typeof parent === 'string' && parent) return true;
  const sub = input.subagent ?? input.isSubagent ?? input.subAgent;
  return sub === true || sub === 'true';
}

async function main() {
  const input = await readStdinJson();
  const child = isChildSession(input);
  if (child) {
    emit('INFO', INFO_NOT_MAIN);
    return;
  }
  if (isSkillgridProject(process.cwd())) {
    if (fs.existsSync(ROUTER)) {
      const msg = fs.readFileSync(ROUTER, 'utf8');
      emit('IMPORTANT', `${INTRO}\n\n${msg}`);
    } else {
      emit('INFO', INFO_NOT_FOUND);
    }
  } else {
    emit('INFO', INFO_NOT_PROJECT);
  }
}

main()
  .then(() => process.exit(0))
  .catch(() => {
    emit('INFO', INFO_NOT_FOUND);
    process.exit(0);
  });
