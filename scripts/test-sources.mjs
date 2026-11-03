#!/usr/bin/env node
/**
 * test-sources.mjs — fixture-driven tests for sources.mjs (the Node port of
 * sources.py, plan item J). The citation ledger owns the url → [n] mapping:
 * add/list/ingest/quote/render/verify. Tests cover id assignment, URL
 * normalization (idempotent add), ingest from JSON, verbatim quote checking,
 * render styles, and verify's hallucination/coverage/evidence checks.
 *
 * Usage: node scripts/test-sources.mjs
 * Exit: 0 all pass, 1 any fail.
 */

import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const HERE = dirname(fileURLToPath(import.meta.url));
// sources.mjs lives in the grounded-citations skill (the single source of truth
// for the citation ledger), not the repo scripts/ dir.
// sources.mjs lives in the grounded-citations skill. HERE is scripts/, so the
// skill is at <repo>/.agents/skills/knowledge/grounded-citations/scripts/.
const SRC = join(HERE, "..", ".agents", "skills", "knowledge", "grounded-citations", "scripts", "sources.mjs");

let passed = 0, failed = 0;
const failures = [];
function assert(name, cond, detail = "") {
  if (cond) { passed++; console.log(`  ok   ${name}`); }
  else { failed++; failures.push(name); console.log(`  FAIL ${name}${detail ? ` — ${detail}` : ""}`); }
}

function tmpLedger() {
  const root = mkdtempSync(join(tmpdir(), "sgsrc-"));
  const ledger = join(root, "ledger.json");
  return { root, ledger, cleanup: () => rmSync(root, { recursive: true, force: true }) };
}

function run(ledger, ...args) {
  const full = ["--ledger", ledger, ...args];
  try {
    const stdout = execFileSync("node", [SRC, ...full], { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"] });
    return { code: 0, stdout, stderr: "" };
  } catch (e) {
    return { code: e.status ?? -1, stdout: e.stdout ?? "", stderr: e.stderr ?? "" };
  }
}

// 1. add assigns sequential ids, idempotent on re-add (same url → same id)
{
  const p = tmpLedger();
  let r = run(p.ledger, "add", "https://example.com/a", "--title", "A");
  assert("add: exit 0", r.code === 0, r.stderr);
  assert("add: [1] a", /\[1\]\s+https:\/\/example\.com\/a/.test(r.stdout), r.stdout);
  r = run(p.ledger, "add", "https://example.com/b", "--title", "B");
  assert("add: [2] b", /\[2\]\s+https:\/\/example\.com\/b/.test(r.stdout), r.stdout);
  // Re-add the same URL (with a fragment) → same id, no new entry.
  r = run(p.ledger, "add", "https://example.com/a#section");
  assert("add-idempotent: [1] again", /\[1\]\s+https:\/\/example\.com\/a/.test(r.stdout), r.stdout);
  r = run(p.ledger, "list");
  assert("add-idempotent: still 2 sources", (r.stdout.match(/^\[\d+\]/gm) || []).length === 2, r.stdout);
  p.cleanup();
}

// 2. reset clears the ledger
{
  const p = tmpLedger();
  run(p.ledger, "add", "https://example.com/a");
  let r = run(p.ledger, "reset");
  assert("reset: exit 0", r.code === 0, r.stderr);
  r = run(p.ledger, "list");
  assert("reset: empty", /ledger is empty/i.test(r.stdout), r.stdout);
  p.cleanup();
}

// 3. ingest walks JSON tool output and registers urls
{
  const p = tmpLedger();
  const json = join(p.root, "out.json");
  writeFileSync(json, JSON.stringify({
    data: { web: [
      { url: "https://a.com/1", title: "One" },
      { url: "https://b.com/2", title: "Two" },
      { url: "https://a.com/1", title: "One dup" },
    ] },
  }));
  const r = run(p.ledger, "ingest", json);
  assert("ingest: exit 0", r.code === 0, r.stderr);
  assert("ingest: [1] a", /\[1\]\s+https:\/\/a\.com\/1/.test(r.stdout), r.stdout);
  assert("ingest: [2] b", /\[2\]\s+https:\/\/b\.com\/2/.test(r.stdout), r.stdout);
  assert("ingest: deduped (no [3])", !/\[3\]/.test(r.stdout), r.stdout);
  p.cleanup();
}

// 4. quote: verbatim (modulo whitespace/markup) accepted; paraphrase rejected
{
  const p = tmpLedger();
  run(p.ledger, "add", "https://example.com/a");
  const page = join(p.root, "page.txt");
  writeFileSync(page, "The quick brown fox jumps over the lazy dog today.\n");
  let r = run(p.ledger, "quote", "1", "--text", "the quick brown fox jumps over the lazy dog today", "--from", page);
  assert("quote: verbatim accepted (case-insensitive)", r.code === 0, r.stderr);
  r = run(p.ledger, "quote", "1", "--text", "the fox ran very fast across the field", "--from", page);
  assert("quote: paraphrase rejected", r.code !== 0, r.stdout + r.stderr);
  p.cleanup();
}

// 5. render markdown / evidence styles
{
  const p = tmpLedger();
  run(p.ledger, "add", "https://example.com/a", "--title", "A");
  let r = run(p.ledger, "render");
  assert("render: markdown header", /## Sources/.test(r.stdout), r.stdout);
  assert("render: [1] line", /\[1\] https:\/\/example\.com\/a — A/.test(r.stdout), r.stdout);
  r = run(p.ledger, "render", "--style", "footnotes");
  assert("render: footnotes", /\[\^1\]: https:\/\/example\.com\/a — A/.test(r.stdout), r.stdout);
  r = run(p.ledger, "render", "--style", "bibtex");
  assert("render: bibtex", /@misc\{source1,/.test(r.stdout), r.stdout);
  p.cleanup();
}

// 6. render --cited-in only includes cited ids
{
  const p = tmpLedger();
  run(p.ledger, "add", "https://example.com/a");
  run(p.ledger, "add", "https://example.com/b");
  const draft = join(p.root, "draft.md");
  writeFileSync(draft, "A claim here.[1]\n\nAnother.[1]\n");
  const r = run(p.ledger, "render", "--cited-in", draft);
  assert("cited-in: only [1]", /\[1\] https:\/\/example\.com\/a/.test(r.stdout), r.stdout);
  assert("cited-in: no [2]", !/\[2\]/.test(r.stdout), r.stdout);
  p.cleanup();
}

// 7. verify: hallucinated id fails
{
  const p = tmpLedger();
  run(p.ledger, "add", "https://example.com/a");
  const draft = join(p.root, "draft.md");
  writeFileSync(draft, "A claim.[1] And another.[9]\n\n## Sources\n\n[1] https://example.com/a\n");
  const r = run(p.ledger, "verify", draft);
  assert("verify: unknown id fails", r.code === 1, r.stdout + r.stderr);
  assert("verify: names [9]", /\[9\]/.test(r.stderr), r.stderr);
  p.cleanup();
}

// 8. verify: clean draft passes
{
  const p = tmpLedger();
  run(p.ledger, "add", "https://example.com/a");
  const draft = join(p.root, "draft.md");
  writeFileSync(draft, "A claim.[1]\n\n## Sources\n\n[1] https://example.com/a\n");
  const r = run(p.ledger, "verify", draft);
  assert("verify: clean passes", r.code === 0, r.stdout + r.stderr);
  assert("verify: says OK", /citations OK/.test(r.stdout), r.stdout);
  p.cleanup();
}

// 9. verify --min-coverage fails thinly-cited prose
{
  const p = tmpLedger();
  run(p.ledger, "add", "https://example.com/a");
  const draft = join(p.root, "draft.md");
  // 4 sentences, only 1 cited → 25% coverage, below 0.5.
  writeFileSync(draft,
    "First sentence with a citation here.[1]\n" +
    "Second sentence has no citation at all.\n" +
    "Third sentence is also uncited really.\n" +
    "Fourth sentence likewise lacks one.\n\n" +
    "## Sources\n\n[1] https://example.com/a\n");
  const r = run(p.ledger, "verify", draft, "--min-coverage", "0.5");
  assert("verify: low coverage fails", r.code === 1, r.stdout + r.stderr);
  assert("verify: names coverage", /coverage/.test(r.stderr), r.stderr);
  p.cleanup();
}

// 10. render --replace-in rewrites the Sources block idempotently
{
  const p = tmpLedger();
  run(p.ledger, "add", "https://example.com/a", "--title", "A");
  const draft = join(p.root, "draft.md");
  writeFileSync(draft, "A claim.[1]\n\n## Sources\n\n[1] https://example.com/a\n");
  let r = run(p.ledger, "render", "--replace-in", draft);
  assert("replace-in: exit 0", r.code === 0, r.stderr);
  const after1 = readFileSync(draft, "utf8");
  assert("replace-in: block present", /## Sources/.test(after1), after1);
  // Run again → still exactly one Sources block (idempotent).
  r = run(p.ledger, "render", "--replace-in", draft);
  const after2 = readFileSync(draft, "utf8");
  assert("replace-in: idempotent (one block)", (after2.match(/## Sources/g) || []).length === 1, after2);
  p.cleanup();
}

console.log(`\n${passed + failed} tests: ${passed} passed, ${failed} failed`);
if (failures.length) { console.log("Failures:"); for (const f of failures) console.log(`  - ${f}`); }
process.exit(failed > 0 ? 1 : 0);
