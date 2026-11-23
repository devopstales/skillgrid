import { test, expect } from "bun:test";
import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import {
  REF_BASE,
  AGENT_BRANCH_RE,
  IGNORED_DIRS,
  MAX_UNTRACKED_FILE_SIZE,
  MAX_UNTRACKED_DIR_FILES,
  MAX_PER_SESSION,
} from "../src/core/constants.ts";

/**
 * Contract-drift guard (R13 / blueprint Task 9): the plugin's constants must
 * equal the values stated in the skillgrid-skills contract. The contract file
 * carries a machine-readable fence (added by Task 11, in
 * `.agents/skills/_shared/work-unit-commits/README.md` § Scratch
 * checkpoints): a line-anchored `~~~skillgrid-constants` block at column 0,
 * one `KEY: value` line per constant, closed by `~~~` at column 0. One parse
 * of that block — no prose regexes.
 *
 * The plugin repo has NO in-repo contract, so the env var is REQUIRED:
 * unset → fail naming the env. File missing or fence missing → fail.
 * Mismatch → fail naming the field(s) and both values.
 */

interface Fence {
  [key: string]: string;
}

function extractFence(src: string): Fence | null {
  const lines = src.split("\n");
  const start = lines.findIndex((l) => l === "~~~skillgrid-constants");
  if (start < 0) return null;
  const end = lines.findIndex((l, i) => i > start && l === "~~~");
  if (end < 0) return null;
  const out: Fence = {};
  for (const line of lines.slice(start + 1, end)) {
    const m = line.match(/^([A-Z_]+):\s?(.*)$/);
    if (m) out[m[1]] = m[2];
  }
  return Object.keys(out).length > 0 ? out : null;
}

/** Parse a JSON array literal (the fence's IGNORED_DIRS form) or null. */
function parseArrayLiteral(raw: string): string[] | null {
  try {
    const v = JSON.parse(raw);
    return Array.isArray(v) ? v : null;
  } catch {
    return null;
  }
}

test("plugin constants equal the contract file's values", async () => {
  const path = process.env.SKILLGRID_CONTRACT_PATH
    ?? resolve(import.meta.dir, "../../../.agents/skills/_shared/work-unit-commits/README.md");
  expect(path, "set SKILLGRID_CONTRACT_PATH to the skillgrid-skills work-unit-commits README").toBeTruthy();

  let src: string;
  try {
    src = await readFile(path!, "utf8");
  } catch (e) {
    throw new Error(`contract file unreadable at ${path}: ${String(e)}`);
  }

  const fence = extractFence(src);
  expect(fence, `contract fence not found at ${path} — Task 11 not applied?`).not.toBeNull();

  const diffs: string[] = [];
  /** The fence's value for `key`: unquoted JSON string literals, numbers verbatim. */
  const fenceValue = (key: string): string | null => {
    const raw = fence![key];
    if (raw === undefined) return null;
    if (raw.startsWith('"')) {
      try {
        return JSON.parse(raw) as string;
      } catch {
        return raw;
      }
    }
    return raw;
  };
  const check = (key: string, actual: string) => {
    const got = fenceValue(key);
    if (got === null) diffs.push(`${key}: fence omits ${key}`);
    else if (got !== actual) diffs.push(`${key}: fence '${got}' ≠ plugin '${actual}'`);
  };

  check("REF_BASE", REF_BASE);
  check("AGENT_BRANCH_RE", AGENT_BRANCH_RE.source);

  const fenceDirs = parseArrayLiteral(fence!.IGNORED_DIRS ?? "");
  // IGNORED_DIRS is a Set in the plugin, an ordered array in the fence —
  // compare as sorted arrays (set-vs-order).
  if (fenceDirs === null)
    diffs.push(`IGNORED_DIRS: fence value '${fence!.IGNORED_DIRS}' is not a JSON array`);
  else if (fenceDirs.slice().sort().join("\n") !== [...IGNORED_DIRS].sort().join("\n"))
    diffs.push(
      `IGNORED_DIRS: fence [${fenceDirs.join(", ")}] ≠ plugin [${[...IGNORED_DIRS].sort().join(", ")}]`,
    );

  check("MAX_UNTRACKED_FILE_SIZE", String(MAX_UNTRACKED_FILE_SIZE));
  check("MAX_UNTRACKED_DIR_FILES", String(MAX_UNTRACKED_DIR_FILES));
  check("MAX_PER_SESSION", String(MAX_PER_SESSION));

  expect(diffs, `constants drift from the contract fence:\n${diffs.join("\n")}`).toEqual([]);
});
