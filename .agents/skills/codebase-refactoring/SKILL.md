---
name: codebase-refactoring
description: "Three-part workflow to improve overall codebase health with jscpd — find and fix duplicated code, find and remove or refactor dead code, then find and simplify the largest/most complex files. Findings are treated as claims verified in fresh context before any code is touched, the major conclusions get a red-team pass, and a high-complexity file can fan out design-it-twice candidates. Starts from --health/--dashboard to prioritize, re-measures the score to close, and reports what it couldn't verify rather than claiming a clean pass."
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  part-of: skillgrid
  based_on: codebase-refactoring 1.0 + code-research (verification + red-team) + code-review (blindness + verdicts) + codebase-design (design-it-twice) + codebase-inspection (measurement discipline)
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

**Announce at start:** "I'm using the skillgrid:codebase-refactoring skill to
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

```bash
npx jscpd --complexity --reporters ai --summary-top 10 <path>
# or, ranked alongside duplication and size:
npx jscpd --reporters ai --summary --summary-by complexity <path>
```

```
Complexity by complexity (6 files, 6 folders):
files (tokens/lines/size/cx):
c/checkout.c 176/32/710/13
rust/status.rs 112/24/481/9
swift/Profile.swift 88/21/446/8
```

`cx` is a language-aware cyclomatic-complexity estimate — roughly one path
per function plus one per branch (`if`, loops, `&&`/`||`, `match`/`switch`
arms, the ternary and Elvis operators, each counted the way the language
actually spells it) — read as a ranking signal, not an exact metric. Prose
and data files (Markdown, JSON, YAML, lock files, …) always score `0`; a
large README is not a refactoring target. jscpd's own health score treats a
file with complexity **50 or higher** as "complex" (the threshold its
`dead code`-style half-life curve is calibrated against) — a reasonable
default cutoff for "worth reading" if the summary doesn't make an obvious
one clear.

### Workflow

1. Take the top N files by complexity (and, from the `--summary` form, by
   size — a huge low-complexity file, e.g. a big data table, is a different
   problem than a small high-complexity one)
2. Read each file and name the actual driver: deep nesting, a long function
   doing several unrelated things, a large `switch`/`if`-chain over a type
   or state, or a file that is really several modules glued together
3. Pick the matching strategy:
   - **Deep nesting** → guard clauses / early return, flatten the happy path
   - **Long function, several responsibilities** → extract function per
     responsibility, named for what it does
   - **Branching over a fixed set of cases** (type, enum, status) →
     replace with a lookup table/map, or polymorphism if each case already
     has its own type
   - **File doing too much** → split along its actual seams (one export
     group per file), not an arbitrary line-count cut
4. **For a high-complexity file (cx ≥ 50) where the split isn't obvious, run
   design-it-twice.** Per `skillgrid:codebase-design`'s
   [DESIGN-IT-TWICE.md](../codebase-design/DESIGN-IT-TWICE.md): spawn 2–3
   subagents in parallel, each producing a **radically different**
   decomposition of the file (one minimizing the public interface, one
   maximizing testability through a seam, one optimizing for the most common
   caller). Compare them on **depth** (leverage at the interface), **locality**
   (where change concentrates), and **seam placement**, then pick — or hybrid.
   This is the one place in the pass where the first idea is unlikely to be
   the best, so you pay the parallel cost deliberately rather than on every
   file.
5. Refactor one file at a time; keep behavior identical — this is a
   complexity pass, not a rewrite. If a duplication clone (step 1) or a dead
   branch (step 2) turns up while reading, still handle it in its own step
   so the reason for each change stays traceable
6. **Verify the split actually reduced complexity.** Re-run `--complexity` on
   the touched files to confirm `cx` actually dropped per function — a split
   that only moves code around without reducing branching per function didn't
   fix anything, it just renamed the problem. If `cx` didn't drop, the split
   is `unverified` as a complexity fix and gets reported as such, not as a
   win.

## Step 4 — Red-team the major conclusions

Before declaring the pass done, the **major conclusions** — the handful of
structural claims the pass actually rests on ("this module is safe to
delete", "these two files are safe to merge", "this split reduces complexity")
— get a **fresh-context red-team pass** per `skillgrid:code-research`'s
red-team discipline. A skeptic subagent is given the conclusion and a search
budget, but **no supporting evidence and no run context** — its job is to find
the evidence the conclusion is wrong:

- the deleted "dead" module that a string-based router, a test, or a plugin
  still reaches
- the "merged" clone that actually had divergent behavior
- the "simplified" file whose split moved complexity into a caller instead of
  removing it

Weigh what comes back, don't just append it: a conclusion that survives gets
its strongest counter-argument acknowledged in the report; one that doesn't is
revised before the report states it. Zero findings after a real search is
itself reportable — say what was searched for and not found. A low-stakes,
single-file pass can skip this; a pass that deletes a public module or splits
a load-bearing file cannot.

## Step 5 — Confirm the improvement

```bash
npx jscpd --health --reporters ai <path>
```

Compare the score and grade to the Step 0 measurement, **on the same tool and
the same scope** (measurement discipline). A dimension that got worse
(duplication crept back in while extracting a shared complexity fix, say) is a
signal to revisit that step, not just note it — re-run the relevant pass
(`--summary`, `--dead-code`, `--complexity`) rather than only trusting the one
aggregate number. Report the delta as **improvement / regression / not
comparable** — the last when the dimensions or scope changed between runs, per
the measurement discipline above.

## Tips

- Don't run all three passes on a codebase you've never seen before without
  reading `--dashboard` first — on a project where only one dimension is
  actually bad, spending equal effort on all three wastes time on the two
  that are already fine
- A `--health-input` file (coverage, security scan results) can be layered
  onto the same score if the project already tracks those; it doesn't change
  anything about this workflow, it just adds more dimensions to Step 0
- Commit each step separately (duplication fixes, dead-code removal,
  complexity simplification) — a reviewer verifying "did this actually reduce
  X" wants a diff scoped to X
- **Verification is the cost of the pass, not overhead.** A pass that deleted
  18 "unused" exports without a single independent grep is 18 unverified
  claims, not 18 deletions. Budget for the verifier.

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "jscpd gave it 95 confidence, I'll just delete it" | Confidence is a floor for what's reported, not a proof. Dynamic access, a string router, and an unparsed file all hide real uses. Verify the load-bearing ones in fresh context. |
| "One grep confirms it's dead, so it's verified" | Only if the grep is independent of the tool's own traversal. A grep that reproduces jscpd's import analysis is the same source. Look the way a maintainer would — tests, plugins, dynamic imports the tool doesn't parse. |
| "I'll verify at the end, one pass over the deletions" | End-of-pass rewrites degrade — an hour of accumulated context makes you trust what you deleted. Verify at landing, per finding, in fresh context. |
| "This is just a refactor, the red team is overkill" | The red team reads the conclusion with no supporting evidence and hunts the bear case. It catches the confident wrong deletion the pass's shared context made comfortable. Off for a single-file pass; on when a public module or load-bearing file is touched. |
| "The first split I thought of is fine, why design it twice?" | For a high-complexity file the first decomposition is unlikely to be the best. Design-it-twice is the deliberate parallel cost that catches the better seam — paid on the hard files, not every file. |
| "The score went up, so the pass worked" | Only if the same dimensions and the same scope were measured both times. A dimension that went from `n/a` to measured isn't improvement — it's a different measurement. Say which. |
| "A lens failed mid-run, but the rest looked clean, so it's a clean pass" | A pass with a step that didn't finish is "possibly incomplete", never "clean". Log which paths didn't run and report it. |

## Red Flags

**Never:**
- Delete an `unused-export` or `unused-file` on confidence alone without an
  independent check
- Let a verifier anchor on the tool's confidence instead of looking the way a
  maintainer would
- Present a Grade C (in-context, report already in hand) verification as
  independent evidence
- Confidently pass a check that isn't inferable from static analysis — route
  it to `could-not-verify` and name the held-out test that would settle it
- Merge two "identical" clones without saying which behaviors are actually
  identical
- Claim a clean pass when a step failed, timed out, or returned empty
- Read a raw health-score delta as improvement when the dimensions or scope
  changed between runs
- Skip the red team on a pass that deletes a public module or splits a
  load-bearing file because "it feels right"
- Re-run jscpd and call a non-dropped `cx` a complexity win

## Verification

- [ ] Step 0 measurement states the tool and the `<path>` scope, and the
      Step 5 re-measurement uses the same tool and scope (or the report says
      the delta is not comparable and why)
- [ ] Every load-bearing dead-code finding (unused-file, unused-export)
      carries a verdict from the table and an independence grade; Grade C
      verifications are reported as diagnostic only, not as "verified before
      deleting"
- [ ] No `could-not-verify` or `unverified` finding was silently deleted —
      each was routed to a human look or a held-out test, named
- [ ] The project's own test suite and build were run after each deletion
      batch, not only jscpd's re-run
- [ ] Each touched high-complexity file was re-measured with `--complexity`
      and the `cx` drop is confirmed (or the split is reported as
      `unverified` as a complexity fix)
- [ ] The major conclusions (safe-to-delete, safe-to-merge, complexity-reduced)
      went through a red-team pass with no supporting evidence given, and the
      outcome (survived / revised) is reflected in the report
- [ ] The resilience log is present: any step that failed, timed out, or
      returned empty is named, and no "clean pass" was claimed while one is
      pending
- [ ] The report separates removed (verified) / skipped (with reason) / not
      reached, rather than listing only what was done
- [ ] Each step was committed separately, so a reviewer can scope a diff to
      one dimension

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
