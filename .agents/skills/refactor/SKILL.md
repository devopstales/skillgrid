---
name: refactor
description: "Improve overall codebase health with jscpd: find and fix duplicated code, dead code, and the largest/most-complex files. Use when asked to clean up, improve, refactor, or pay down tech debt in a codebase — not to fix one specific bug. Findings are claims verified in fresh context before any code is touched."
license: MIT
metadata:
  author: devopstales
 version: "2.2"
  part-of: skillgrid
  based_on: codebase-refactoring 1.0 + code-research (verification + red-team) + code-review (blindness + verdicts) + codebase-design (design-it-twice) + codebase-inspection (measurement discipline)
  created_by: human
  related_skills: [debug]
  version: "2.4"
---

# Codebase Refactoring

## Contents
- Start here: measure, then prioritize / The Finding Contract (blindness, verdict table, resilience rule)
- Step 1 Duplicated code / Step 2 Dead code / Step 3 Big & complex files / Step 4 Red-team the major conclusions / Step 5 Confirm the improvement
- Tips / Common Rationalizations / Red Flags / Verification / References

A guided pass over a codebase's three biggest maintenance costs — duplicated
code, dead code, and complexity — using jscpd to find each and a strategy to
fix it. Use this skill when asked to "clean up", "improve", "refactor", or
"pay down tech debt in" a codebase, rather than to fix one specific bug.

**When NOT to use:** a single known bug or one named function to change —
that's `skillgrid:debug` or a direct edit, not a codebase-wide health pass.
Also not for a one-file readability tweak; the three-pass measure→prioritize
→verify loop only pays off at codebase scale.

**Announce at start:** "I'm using the skillgrid:refactor skill to
measure, then prioritize, this codebase."

**Iron law:**

```
A FINDING IS A CLAIM, NOT A FIX — VERIFY BEFORE YOU TOUCH CODE
```

jscpd's dead-code, duplication, and complexity reports are **claims filed by a
tool**, each with a confidence and a known failure mode (dynamic access, a
string-based router, a template file the tool doesn't parse). The tool proves
nothing *reaches* or *doesn't reach* code from an entry point with certainty —
it proves what its traversal saw. So every load-bearing finding is verified in
**fresh context** against an independent source (a grep the tool doesn't do,
the project's own test suite, the build) before any code is deleted, merged,
or split. A refactoring pass that deletes on the tool's say-so alone is a
hypothesis in costume — it gets verified as one.

The three passes are otherwise independent and safe to run in any order;
**duplication → dead code → complexity** is the default because fixing
duplication first means step 2 isn't wasting effort tracing call sites that a
later extract-function pass will move anyway, and clearing dead code before
step 3 means the files ranked for simplification are ones actually worth
simplifying, not ones about to be deleted.

## Start here: measure, then prioritize

```bash
npx jscpd --health --reporters ai <path>
```

```
health 74 B (duplication 75, dead-code 72, complexity 76; 93 code lines)
```

The health score is a weighted mix of three sub-scores, each a share of the
code lines — high is good, 100 is clean. The lowest of the three names where
the codebase actually hurts most; if it isn't the order below, do that
dimension first instead. `n/a` on a dimension (e.g. `dead code n/a`) means
jscpd could not measure it — dead-code detection needs JavaScript,
TypeScript or Python to be a real share of the code — skip that step rather
than treating it as clean.

For the full picture behind that score — largest duplicated formats, most
complex files, largest dead-code findings, all in one screen — run `npx
jscpd --dashboard` instead. Its `console` output (the default) is the only
reporter with that per-file detail; `--reporters ai` on `--dashboard` prints
the same one-line summary shown above, not the breakdown, so don't reach for
it expecting more than the score. Use `--reporters json` on `--dashboard` if
you need the detail in a parseable form.

**Complementary health signal (codescene MCP, when wired):** the `codescene`
MCP server (`@codescene/codehealth-mcp`) measures a different thing — a Code
Health score, not duplication / dead-code / complexity. Use it for the
before/after delta, not for the refactoring decisions: `code_health_score`
for a quick number on the file about to change, `code_health_review` for the
file-level detail, and `analyze_change_set` before the PR for the
branch-level view. Quote the score delta (before → after) in the report as
the health outcome; let jscpd's sub-scores drive which dimension to fix.
If the MCP is not wired, skip this — jscpd's health score is sufficient on
its own.

**Measurement discipline** (from `skillgrid:codebase-inspection`): the number
must come from a stated tool, and the scope must exclude dependency/build
dirs. Record the tool that produced the score and the `<path>` it scanned, so
the Step 4 re-measurement is comparable. Two health scores are only comparable
when built from the same dimensions and the same scope — if a dimension went
from `n/a` to measured (or the reverse) because the file mix changed between
runs, say so rather than reading the raw score delta as improvement or
regression.

## The Finding Contract (applies to every step)

This is the shared discipline; each step instantiates it. It borrows
`skillgrid:code-research`'s verification epistemics and `skillgrid:code-review`'s
blindness + verdict discipline, scaled to a tool-reported finding.

### Blindness contract

The value of the verification is its independence from the tool's own
assumptions. A finding is verified by **looking at the code the way a
maintainer would, not the way the tool did** — a fresh pass that does not
anchor on the tool's confidence. Two rules keep it real:

- **No leaks.** When you dispatch a verifier subagent (Step 2 dead code, or the
  Step 3 red team), hand it the finding and a search budget — never your
  narrative, your confidence in the pass, or a prior verdict on the same
  file. That is what makes the verdict independent evidence rather than you
  talking to yourself.
- **Independence grade.** Label each verification with an independence grade:
  - **A** — fresh context, no shared history with the pass (a dispatched
    subagent that only sees the finding).
  - **B** — fresh context but you grepped and read in the same session as you
    decided to run the pass.
  - **C** — you verified in your own context while already holding the tool's
    report and your intent to delete.

A **Grade C** verification is **diagnostic only**: it can guide a fix, but it
cannot stand as the "verified before deleting" evidence this skill requires.
If the runtime cannot isolate a fresh context (no subagent dispatch), record
the verification as Grade C and say so — never present it as independent.

### Verdict table (six values)

The pass renders exactly one verdict per finding, after verification:

| Verdict | Meaning | Action |
|---------|---------|--------|
| `verified` | Independent check agrees the code is dead/duplicated/complex as reported | Apply the fix |
| `disputed` | Two independent checks disagree (e.g. grep says used, tool says unused) | Report both, both cited; do NOT auto-delete — route to a human look |
| `unverified` | No independent check was possible within budget | The finding stays, **flagged** — not deleted, not passed silently |
| `overturned` | The independent check contradicts the finding | Corrected; the tool's confidence was wrong, the original is noted |
| `could-not-verify` | The check is not inferable from static analysis alone — it needs a runtime, a held-out test, or state the diff doesn't show. **Never a confident pass.** | Emit the concrete held-out test that would settle it; route to a human look |
| `false` | You checked; the bad outcome does not apply here | Say what disproves it; drop the finding |

`could-not-verify` and `unverified` entries are **never dropped** — they route
to a human look or a held-out test, because a silent pass on a check you
couldn't infer is the failure mode this skill exists to catch. This is the
same refusal to "confidently pass a check you can't infer" that
`skillgrid:code-review` enforces on its verdicts.

### Resilience rule

A step that fails, times out, or returns empty is **logged by name**: if the
dead-code traversal fails on part of the tree, record which paths didn't run
and continue with the survivors. If **all** of a step's findings are
`unverified` or the step itself didn't finish, the pass is **"possibly
incomplete", never "clean"** — report that it may be incomplete and list which
paths or categories didn't run. This mirrors `skillgrid:code-review`'s
resilience rule verbatim.

## Step 1 — Duplicated code

```bash
npx jscpd --reporters ai --summary <path>
```

Find and read each clone, decide whether it is a real copy worth merging, and
apply the matching refactoring strategy (extract function, parameterize,
extract a shared module or constant, or a base class/template for structural
duplication). Full guided workflow, triage rules for renamed/near-miss clones,
and worked examples per clone kind:

→ **`skillgrid:dry-refactoring`** — run this now, then come back here for
steps 2 and 3. Its own `--summary` hotspot ranking is the same one the health
score's duplication dimension reads from, so a file it flags as a duplication
hotspot is the same file that dragged the score down.

**Verification for duplicates:** a duplicate finding is lower-risk than a
dead-code finding (merging two copies is reversible and the build catches a
wrong merge), so Grade B verification (a careful read of both clones in
session) is acceptable. But a clone that *looks* identical and isn't (one has
an extra guard, one handles an edge case) is a `disputed` finding — do not
merge the two into one until you can say which behaviors are identical.

## Step 2 — Dead code

```bash
npx jscpd --dead-code --reporters ai <path>
```

```
basta dead code report — 18 findings across 30 files (22.1% of 272 lines)
Confidence is 0-100; anything below 90 has a listed reason it may be wrong. Verify a finding before deleting the code.
unused-file typescript/src/legacy-export.ts:1:1 confidence=95 typescript/src/legacy-export.ts is never imported and is not an entry point
unused-export typescript/src/invoice.ts:21:17 confidence=85 exported function `renderReceipt` is never imported
unused-symbol typescript/src/invoice.ts:29:10 confidence=90 function `describeTotal` is never used in typescript/src/legacy-export.ts
unused-import typescript/src/invoice.ts:2:10 confidence=100 `roundToCents` is imported but never used
```

This is a graph traversal from the project's entry points (`package.json`
`main`/`bin`/`exports`/`scripts`, `pyproject.toml` scripts, framework
conventions, or `--entry <glob>`), not a reference count — a helper whose
only caller is itself dead is reported too, so deleting one finding can make
another true that wasn't reported yet. Covers JavaScript, TypeScript, JSX,
TSX, Vue, Svelte, Astro and Python.

**The load-bearing claims are the `unused-export` and `unused-file` findings**
— the handful whose deletion actually changes the public surface. Those get a
**fresh-context verifier** (Grade A when a subagent is available) per
`skillgrid:code-research`'s verifier discipline: the verifier gets the finding
and a search budget, and hunts for the **independent source** that confirms or
contradicts it — a grep the tool's traversal didn't do, a string-based router
reference, a dynamic import, a test that references the export, a plugin file
the tool doesn't parse. The verifier rules on status (`verified` / `disputed` /
`unverified` / `overturned`) and does not rewrite the finding. Verification
happens **as each finding lands**, not as an end-of-pass rewrite — the same
"verify at landing, never at the end" rule `skillgrid:code-research` uses,
because an hour of accumulated context makes you trust what the tool reported.

`unused-import` and `unused-symbol` are lower-stakes (the build or a re-run
catches a wrong deletion) and may be verified Grade B, but the same rule
holds: a finding below 90 confidence carries a stated reason it might be
wrong, and you read that reason before deleting.

### The four categories, and what to do with each

| Category | Meaning | Usual fix | Verification weight |
|----------|---------|-----------|---------------------|
| `unused-file` | Never imported anywhere, not an entry point | Delete the file | **Load-bearing** — fresh-context verifier (Grade A) |
| `unused-export` | Exported, but no other file imports it | Delete it, or drop the `export` keyword if something in the same file still uses it | **Load-bearing** — fresh-context verifier (Grade A) |
| `unused-symbol` | A module-private declaration nothing in its own file references | Delete it | Grade B acceptable |
| `unused-import` | Imported but never referenced in the file | Delete the import line | Grade B acceptable |

### Workflow

1. Run with `--reporters ai` (add `--dead-code-categories` to focus on one
   category, `--include-tests` if test-only usage shouldn't count as "used",
   `--include-entry-exports` to also flag an entry file's own unused
   exports, normally excluded since its whole surface is the public API)
2. Sort by confidence, high to low. Everything is 0-100; below 90 the finding
   carries a stated reason it might be wrong — `eval`/`getattr` calls,
   decorators the analyzer doesn't recognize, a wildcard re-export, the name
   showing up only inside a string literal, or a file elsewhere that failed
   to parse (see `statistics.unparsedFiles` in the JSON report). Read the
   reason before deleting; `--min-confidence` (default 60) only sets the
   floor for what's reported, it doesn't make a low-confidence finding safe
3. For each **load-bearing** finding (unused-file, unused-export), dispatch a
   fresh-context verifier or verify Grade A by hand: grep the symbol name
   across the repo *the way the tool didn't* (dynamic access, a string-based
   router, a template file jscpd doesn't parse can all hide a real use), check
   the test suite, and check the build. Render one verdict from the table.
   For `unused-import`/`unused-symbol`, verify Grade B in session
4. Apply the fix from the table above **only to `verified` findings**. A
   `disputed` finding goes to a human look with both checks cited; an
   `overturned` one is dropped with the tool's wrong confidence noted
5. Run the project's own test suite and build after each batch of deletions,
   not only jscpd's re-run: dead-code analysis proves nothing *reaches* the
   code from an entry point, not that removing it can't break a build step
6. Re-run `--dead-code` after a batch: a file whose only import was the one
   you just deleted may now be dead itself — that cascade is expected, keep
   going until a clean pass, not just once
7. **Report what you couldn't verify.** Separate the report into: what you
   removed (verified), what you skipped and why (low confidence, a reason you
   couldn't rule out, `could-not-verify` routed to a held-out test, or
   intentionally-kept dead code such as a public library export), and what the
   run didn't reach (the resilience log). A pass that removed on confidence
   alone is a `Grade C` pass reported as a `Grade A` one.

## Step 3 — Big and complex files

See `references/refactor-steps-3-5.md` for the full procedure.

## References

- **`skillgrid:dry-refactoring`** — the duplication pass this skill delegates
  Step 1 to (full guided workflow, triage for renamed/near-miss clones).
- **`skillgrid:code-research`** — the verification epistemics and red-team
  discipline this skill scales to a tool-reported finding:
  [researchers/verifier.md](../code-research/researchers/verifier.md)
  (the 4-value claim status + independence rule) and
  [researchers/red-team.md](../code-research/researchers/red-team.md)
  (the fresh-context skeptic on a major conclusion).
- **`skillgrid:code-review`** — the blindness contract, the six-value verdict
  table, and the resilience rule this skill's Finding Contract instantiates.
- **`skillgrid:codebase-design`** — the deep-module vocabulary (depth,
  locality, seam) and the
  [design-it-twice pattern](../codebase-design/DESIGN-IT-TWICE.md) this
  skill uses for high-complexity splits.
- **`skillgrid:codebase-inspection`** — the measurement discipline (state the
  tool, exclude dependency dirs, same-dimension comparability) this skill
  applies to the health score.
- [../skill-creator/references/rationalizations.md](../skill-creator/references/rationalizations.md) — the authoring rationalizations (script extraction, size, spine sections); load when you or the user push back on this skill's shape
