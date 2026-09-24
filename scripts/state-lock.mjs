#!/usr/bin/env node
/**
 * state-lock.mjs — advisory lockfile for concurrent skillgrid sessions.
 *
 * Creates/removes/checks .skillgrid/.lock to surface concurrent-session
 * conflicts. Advisory only: it warns, never blocks.
 *
 * Usage:
 *   node scripts/state-lock.mjs acquire <change> [session-id] [project-root]
 *   node scripts/state-lock.mjs release <change> [project-root]
 *   node scripts/state-lock.mjs check [project-root]
 *
 * Exit codes:
 *   acquire: 0 (acquired or idempotent), 1 (conflict with active lock)
 *   release: 0 (released or no-op)
 *   check:   0 (no lock), 1 (active lock), 2 (stale lock)
 */

import { readFileSync, writeFileSync, rmSync, existsSync, statSync, openSync, closeSync, utimesSync } from "node:fs";
import { join, resolve } from "node:path";

const STALE_HOURS = 2;

function lockPath(root) {
  return join(root, ".skillgrid", ".lock");
}

function readLock(path) {
  if (!existsSync(path)) return null;
  try {
    return JSON.parse(readFileSync(path, "utf8"));
  } catch {
    return null;
  }
}

function isStale(lock) {
  if (!lock || !lock.acquired_at) return false;
  const age = Date.now() - new Date(lock.acquired_at).getTime();
  return age > STALE_HOURS * 3600 * 1000;
}

function ageStr(lock) {
  const age = Date.now() - new Date(lock.acquired_at).getTime();
  const hours = Math.floor(age / 3600000);
  const mins = Math.floor((age % 3600000) / 60000);
  return `${hours}h ${mins}m`;
}

function cmdAcquire(change, sessionId, root) {
  const lp = lockPath(root);
  const existing = readLock(lp);

  if (existing && existing.change === change) {
    // Idempotent: same change already holds the lock
    return { code: 0, stdout: "", stderr: "" };
  }

  if (existing && !isStale(existing)) {
    // Active lock for a different change → conflict (advisory)
    return {
      code: 1,
      stdout: "",
      stderr: `CONFLICT: lock held by ${existing.change} since ${existing.acquired_at} (age: ${ageStr(existing)})`,
    };
  }

  if (existing && isStale(existing)) {
    // Stale lock → overwrite with warning
    const payload = { change, session_id: sessionId, acquired_at: new Date().toISOString() };
    writeFileSync(lp, JSON.stringify(payload, null, 2));
    return {
      code: 0,
      stdout: "",
      stderr: `STALE: overwrote lock from ${existing.change} (age: ${ageStr(existing)})`,
    };
  }

  // No lock → create
  const payload = { change, session_id: sessionId, acquired_at: new Date().toISOString() };
  writeFileSync(lp, JSON.stringify(payload, null, 2));
  return { code: 0, stdout: "", stderr: "" };
}

function cmdRelease(change, root) {
  const lp = lockPath(root);
  const existing = readLock(lp);
  if (existing && existing.change === change) {
    rmSync(lp);
  }
  return { code: 0, stdout: "", stderr: "" };
}

function cmdCheck(root) {
  const lp = lockPath(root);
  const existing = readLock(lp);

  if (!existing) {
    return { code: 0, stdout: "NO LOCK", stderr: "" };
  }

  if (isStale(existing)) {
    return {
      code: 2,
      stdout: `STALE: ${existing.change} since ${existing.acquired_at} (age: ${ageStr(existing)})`,
      stderr: "",
    };
  }

  return {
    code: 1,
    stdout: `LOCKED: ${existing.change} since ${existing.acquired_at} (age: ${ageStr(existing)})`,
    stderr: "",
  };
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1) {
    console.error("Usage: state-lock.mjs <acquire|release|check> [args...]");
    process.exit(1);
  }

  const cmd = args[0];

  // project-root is the last arg if it's a path (not a known keyword)
  let root = ".";
  const rest = args.slice(1);
  // If the last arg looks like a path (contains / or is . or ..), it's the root
  if (rest.length > 0 && (rest[rest.length - 1].includes("/") || rest[rest.length - 1] === "." || rest[rest.length - 1] === "..")) {
    root = resolve(rest.pop());
  } else {
    root = resolve(root);
  }

  let result;
  switch (cmd) {
    case "acquire": {
      if (rest.length < 1) {
        console.error("Usage: state-lock.mjs acquire <change> [session-id] [project-root]");
        process.exit(1);
      }
      const change = rest[0];
      const sessionId = rest[1] || "";
      result = cmdAcquire(change, sessionId, root);
      break;
    }
    case "release": {
      if (rest.length < 1) {
        console.error("Usage: state-lock.mjs release <change> [project-root]");
        process.exit(1);
      }
      const change = rest[0];
      result = cmdRelease(change, root);
      break;
    }
    case "check": {
      result = cmdCheck(root);
      break;
    }
    default:
      console.error(`Unknown command: ${cmd}`);
      process.exit(1);
  }

  if (result.stdout) process.stdout.write(result.stdout + "\n");
  if (result.stderr) process.stderr.write(result.stderr + "\n");
  process.exit(result.code);
}

main();
