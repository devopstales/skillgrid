#!/usr/bin/env node
/**
 * ship-drift-check.mjs — structure drift guard for ship.
 *
 * Compares the actual git diff (base..HEAD) against the anticipated file
 * set (from tasks.md Files + blueprint.md Global Constraints). Files changed
 * outside the anticipated set are "structure drift" — the change touched
 * files it didn't declare.
 *
 * Usage:
 *   node scripts/ship-drift-check.mjs check <base-ref> [options] [project-root]
 *     --anticipated <path>  Add a path prefix to the anticipated set (repeatable)
 *     --exclude <glob>      Add an exclude pattern (repeatable)
 *
 * Exit codes:
 *   0 — no drift (all changes within anticipated set or excluded)
 *   1 — drift detected (prints table of unanticipated files)
 *   2 — usage/parse error
 *
 * Default excludes: *.lock, *.sum, package-lock.json, yarn.lock, pnpm-lock.yaml
 */

import { execSync } from "node:child_process";
import { resolve } from "node:path";

const DEFAULT_EXCLUDES = [
  "*.lock",
  "*.sum",
  "package-lock.json",
  "yarn.lock",
  "pnpm-lock.yaml",
];

function git(root, ...args) {
  return execSync(`git ${args.join(" ")}`, { cwd: root, encoding: "utf8", stdio: "pipe" }).trim();
}

// Returns { files, scope } per verification-scope.md: an empty changed-file
// set is only a real answer (COMPLETE) when the diff actually ran. A git error
// (bad base ref, not a repo) yields an empty set with scope UNSCOPED — a
// non-answer, never a silent clean bill.
function getChangedFiles(root, baseRef) {
  try {
    const out = git(root, "diff", "--name-only", baseRef, "HEAD");
    return { files: out.split("\n").filter(Boolean), scope: "COMPLETE" };
  } catch {
    return { files: [], scope: "UNSCOPED" };
  }
}

function matchPrefix(filePath, prefixes) {
  return prefixes.some((prefix) => {
    const normalized = prefix.endsWith("/") ? prefix : prefix + "/";
    if (prefix.endsWith("/")) return filePath.startsWith(prefix);
    return filePath.startsWith(normalized) || filePath === prefix;
  });
}

function matchExclude(filePath, patterns) {
  return patterns.some((pattern) => {
    if (pattern.startsWith("*.")) {
      const ext = pattern.slice(1);
      return filePath.endsWith(ext);
    }
    if (pattern.includes("*")) {
      const regex = new RegExp("^" + pattern.replace(/\./g, "\\.").replace(/\*/g, ".*") + "$");
      return regex.test(filePath);
    }
    return filePath === pattern || filePath.endsWith("/" + pattern);
  });
}

function main() {
  const args = process.argv.slice(2);
  if (args.length < 1 || args[0] !== "check") {
    console.error("Usage: ship-drift-check.mjs check <base-ref> [--anticipated <path>...] [--exclude <glob>...] [project-root]");
    process.exit(2);
  }

  const rest = args.slice(1);
  let baseRef = null;
  let root = ".";
  const anticipated = [];
  const excludes = [...DEFAULT_EXCLUDES];

  for (let i = 0; i < rest.length; i++) {
    if (rest[i] === "--anticipated") {
      i++;
      while (i < rest.length && !rest[i].startsWith("--") && rest[i] !== baseRef) {
        anticipated.push(rest[i]);
        i++;
      }
      i--;
    } else if (rest[i] === "--exclude") {
      i++;
      while (i < rest.length && !rest[i].startsWith("--")) {
        excludes.push(rest[i]);
        i++;
      }
      i--;
    } else if (!baseRef) {
      baseRef = rest[i];
    }
  }

  // Last arg that looks like a path is the project root
  const lastArg = rest[rest.length - 1];
  if (lastArg && (lastArg.includes("/") || lastArg === "." || lastArg === "..")) {
    root = resolve(lastArg);
  }

  if (!baseRef) {
    console.error("ERROR: base ref required");
    process.exit(2);
  }

  const { files: changed, scope } = getChangedFiles(root, baseRef);

  if (changed.length === 0) {
    console.log("DRIFT: none");
    console.log(`SCOPE: ${scope}`);
    process.exit(0);
  }

  const drift = [];
  for (const file of changed) {
    if (matchExclude(file, excludes)) continue;
    if (matchPrefix(file, anticipated)) continue;
    drift.push(file);
  }

  if (drift.length === 0) {
    console.log("DRIFT: none");
    console.log(`SCOPE: ${scope}`);
    process.exit(0);
  }

  console.log("STRUCTURE DRIFT DETECTED:");
  console.log("");
  console.log("| File | Status |");
  console.log("|------|--------|");
  for (const f of drift) {
    console.log(`| ${f} | unanticipated |`);
  }
  console.log("");
  console.log(`${drift.length} file(s) changed outside the anticipated set.`);
  console.log(`SCOPE: ${scope}`);
  console.log("Add them to the anticipated set or investigate why they changed.");
  process.exit(1);
}

main();
