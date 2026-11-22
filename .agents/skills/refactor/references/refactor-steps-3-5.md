# Refactor Steps 3-5: Complexity, Red-Team, Confirm

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

## Decision Gates

| Condition | Action |
|---|---|
| Lowest sub-score names a different dimension than the default order | Fix that dimension first — the score names where the codebase hurts most. |
| A dimension is `n/a` (tool could not measure it) | Skip that step — do not treat it as clean. |
| Health-score delta with changed dimensions or scope between runs | Report **not comparable** — never read the raw delta as improvement. |
| A dimension got worse after a step | Re-run that step's pass (`--summary` / `--dead-code` / `--complexity`) — revisit, don't just note it. |
| A deletion rests on tool confidence alone | Verify independently in fresh context, the way a maintainer would (tests, plugins, dynamic imports the tool doesn't parse). |
| A check is not inferable from static analysis | Route it to `could-not-verify` and name the held-out test that would settle it. |
| A lens failed mid-run | The pass is "possibly incomplete", never "clean" — log the paths that didn't run. |
| A single-file pass | Red team may be off. |
| A public module or load-bearing file is touched/split | Red team on; design-it-twice for the high-complexity file. |

## Output Contract

The deliverable is the refactoring report. It MUST contain:

- **Before score** — `npx jscpd --health --reporters ai <path>` output with
  the tool and scope named (measurement discipline).
- **Per-finding verdicts** — the six-value table (`verified` / `disputed` /
  `unverified` / `overturned` / `could-not-verify` / `false`) with the
  independence grade (A / B / C) per verification; `could-not-verify` and
  `unverified` entries carry their held-out test or the paths that didn't run.
- **After score** — same tool, same scope; the delta reported as
  improvement / regression / not comparable.
- **Per-step commit SHAs** — each pass (duplication / dead code / complexity)
  committed separately, so a reviewer can verify one dimension at a time.

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

