#!/usr/bin/env node
// trail — the debug hypothesis-trail state machine (debug SKILL.md, "The
// Hypothesis Trail"). The judgment (which hypothesis to try next) stays in the
// agent; the bookkeeping (append, count, open list, 3-fix gate) is here so it
// cannot drift across compactions.
//
// The trail is a markdown table in state.md:
//   | hypothesis | source | evidence | verdict |
//   |------------|--------|----------|---------|
//   | X is the cause | goldfish | repro shows Y | open |
// verdicts: open | confirmed | ruled out
// fix attempts: lines of the form `fix-attempt: <n> — <one line>`
//
// Usage:
//   node trail.mjs init <dir>                  create the trail file if absent
//   node trail.mjs append <dir> <hyp> <source> <evidence> [verdict]
//                                             add a row (verdict defaults to "open")
//   node trail.mjs list <dir>                   print the whole table
//   node trail.mjs open <dir>                   print only open rows
//   node trail.mjs count <dir>                  print fix-attempt count
//   node trail.mjs status <dir>                 print JSON: attempts, open count,
//                                               gate (go | architecture-question),
//                                               confirmed rows
//
// Exit codes: 0 = ok, 2 = usage error or missing trail.
import { existsSync, mkdirSync, readFileSync, writeFileSync, appendFileSync } from "node:fs";
import { join } from "node:path";

function usage(msg) {
  if (msg) console.error(msg);
  console.error("usage: node trail.mjs <init|append|list|open|count|status> <dir> [args...]");
  process.exit(2);
}

const [cmd, dir, ...rest] = process.argv.slice(2);
if (!cmd || !dir) usage();
const state = join(dir, "state.md");

// --- init ----------------------------------------------------------------------

if (cmd === "init") {
  mkdirSync(dir, { recursive: true });
  if (!existsSync(state)) {
    writeFileSync(state, "# Debug trail\n\n| hypothesis | source | evidence | verdict |\n|------------|--------|----------|---------|\n");
    console.log(`created ${state}`);
  } else {
    console.log(`exists ${state}`);
  }
  process.exit(0);
}

if (!existsSync(state)) usage(`no trail: ${state} (run: node trail.mjs init ${dir})`);

// --- append --------------------------------------------------------------------

if (cmd === "append") {
  if (rest.length < 3) usage();
  // one line per cell: a newline in a cell breaks the table
  const [hyp, src, ev] = rest.slice(0, 3).map((s) => s.replace(/\n/g, " "));
  const verdict = rest[3] || "open";
  if (!["open", "confirmed", "ruled out"].includes(verdict)) {
    usage(`bad verdict: ${verdict} (open | confirmed | ruled out)`);
  }
  appendFileSync(state, `| ${hyp} | ${src} | ${ev} | ${verdict} |\n`);
  console.log(`appended: ${hyp} [${verdict}]`);
  process.exit(0);
}

// --- table rows ------------------------------------------------------------------

function rows(want) {
  const lines = readFileSync(state, "utf8").split("\n");
  const out = [];
  for (const line of lines) {
    if (/^\| hypothesis \|/.test(line)) continue;   // header
    if (/^\|[-| ]+\|$/.test(line)) continue;        // separator
    if (!line.startsWith("|")) continue;
    const cells = line.split("|").map((c) => c.trim());
    // cells: ["", hypothesis, source, evidence, verdict, ""]
    const verdict = cells[4] || "";
    if (!want || verdict === want) out.push(line);
  }
  return out;
}

switch (cmd) {
  case "list":
    for (const line of rows("")) console.log(line);
    break;
  case "open":
    for (const line of rows("open")) console.log(line);
    break;
  case "count": {
    const n = readFileSync(state, "utf8").split("\n").filter((l) => l.startsWith("fix-attempt:")).length;
    console.log(n);
    break;
  }
  case "status": {
    const attempts = readFileSync(state, "utf8").split("\n").filter((l) => l.startsWith("fix-attempt:")).length;
    const open = rows("open").length;
    const confirmed = rows("confirmed").length;
    const gate = attempts >= 3 ? "architecture-question" : "go";
    console.log(JSON.stringify({ attempts, open, confirmed, gate }));
    break;
  }
  default:
    usage();
}
