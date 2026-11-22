---
name: drift
effort: standard
description: Use after execute or ship, or on demand, to detect where the code has diverged from a design doc, plan, or project-wiki page — missing features, incomplete models, logic shortcuts, architectural divergence. Reports only the differences; if the code matches, it says "No drift detected."
license: MIT
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
  based_on: elephant-goldfish-model:egm-drift-report (doc-vs-code drift detection, report-only-differences filter)
  created_by: human
  version: "1.3"
---

# Drift

Detect where the **code** has diverged from the **documentation** that claims to
describe it. Read the doc, locate the cited source files and symbols, read the
**current** code, and report **only** the divergences. The doc is the
specification; the code is the implementation; drift is the gap.

**Announce at start:** "I'm using the skillgrid:drift skill to check
code-vs-doc drift."

## Contents
- When to Use / The Process (Input, Locate, Verify, Detect, Filter, Report) /
  The Gate / Red Flags / Verification

## When to Use

- **After `execute` / before `ship` archive** — the change's plan, `blueprint.md`,
  and any project-wiki pages it touched should still match the code that landed.
  A BLOCKER-level divergence (a feature the spec claims that the code lacks) is
  a **waive-or-fix** gate, not a silent pass.
- **On demand** — "check for drift on `docs/design.md`" or "is the wiki page
  `[[type/slug]]` still accurate?"
- **After a refactor or dependency upgrade** that may have changed behavior
  without updating the doc.

**When NOT to use:** the doc is a pure process note with no code claims
("how we run staging"), or the code has no corresponding doc to drift against
(nothing to compare).

## The Process

### Step 0: Input

1. Take the **target doc(s)** — a design doc, `blueprint.md`, or project-wiki
   page path(s). If none are given, ask the user which doc(s) to check.
2. If the doc names specific files, symbols, or line ranges, use those as the
   primary locate targets. If it names only prose (a feature, a behavior),
   locate the code by search.
3. Read the doc **in full** first. Identify:
   - **Intended behavior** — what the doc says the system does.
   - **Named artifacts** — file paths, class/function names, config keys,
     API endpoints, data-model fields, CLI flags.
   - **Constraining claims** — "the retry limit is 3", "the cache TTL is 60s",
     "the queue is at-least-once". These are the most drift-prone: a number in
     the doc that the code has since changed.

### Step 1: Locate source

For each named artifact in the doc:
- If the doc gives a **path** — read that file.
- If the doc gives a **symbol** (function, class, endpoint) — search for the
  definition. If the symbol is absent, that is itself a drift finding (missing
  feature).
- If the doc gives **prose only** — use code search to find the implementation
  the doc describes. If you cannot locate it after a reasonable search, record
  it as **UNLOCATED** (not drift — you could not find the code to compare).

**Read the live files.** Never drift from memory of what the code "should" be.
The current state of the file is the only source of truth. A doc that was
accurate a commit ago is not the comparison target — the code is.

### Step 2: Verify

For each located artifact, read the **current** source. Extract:
- The actual public API surface (signatures, exported names).
- The actual behavior (the logic the code performs, not what the comment says).
- The actual constants (retry counts, TTLs, thresholds, queue semantics).
- The actual data model (fields present, fields absent).

### Step 3: Detect drift

Compare the doc's claims to the code's reality. Report **only** these four
categories:

| Category | Meaning | Example |
|---|---|---|
| **Missing feature** | Logic the doc describes that is absent in the code. | Doc says "the exporter batches rows"; no batch logic exists. |
| **Incomplete model** | A data model (dataclass, struct, proto, schema) missing fields the doc specifies. | Doc says the `User` model has `mfa_enabled`; the code's `User` does not. |
| **Logic shortcut** | A hardcoded value where the doc specifies dynamic logic (or vice-versa). | Doc says "the timeout is computed from the SLA"; the code has `timeout = 30`. |
| **Architectural divergence** | The implementation uses a different pattern, library, or boundary than the doc proposed. | Doc says "the cache is in-process LRU"; the code uses Redis. |

**Not drift:**
- The code does **more** than the doc says (extra features, extra fields) —
  the doc is incomplete, but the code is not wrong. Note it as a doc gap if it
  is material, but it is not a code drift.
- The doc is ambiguous and the code is a reasonable interpretation.
- Test files, fixtures, and build manifests (they are not the doc's subject).

### Step 4: Filter

**Report ONLY the differences.** Do not describe what matches. If the code
matches the doc exactly for a given artifact, do not write about it. If the
code matches the doc for **every** artifact, the entire report is:

```
No drift detected.
```

This filter keeps the report cheap and the signal high. A drift report that
lists matching items is a status report, not a drift report.

### Step 5: Report

Append to `{YYYY-MM-DD}_DRIFT_REPORT.md` in the working directory (or the
change-specs dir for an in-flight change). Strict format:

```markdown
## <DOC_NAME>

- [MISSING FEATURE] <description> — doc says <X>, code has <nothing>.
- [INCOMPLETE MODEL] <field> absent from <type> — doc specifies it, code omits it.
- [LOGIC SHORTCUT] <file:line> — <hardcoded/dynamic mismatch>.
- [ARCH DIVERGENCE] <subsystem> — doc: <proposed>, code: <actual>.

<or "No drift detected.">
```

One line per divergence, prefixed with the category tag. No prose paragraphs.
No "the code looks correct overall."

### Step 6: Guidance (rigor rules)

- **Be rigorous.** If the doc mentions a specific object, field, or file name
  and the code does not have it, that is a reportable drift. Double-check for
  coverage before calling it missing — search for the symbol in case it moved.
- **Read files liberally.** A drift claim is only as good as the file read
  behind it. If you are unsure whether a field exists, read the file; do not
  guess.
- **Process sequentially.** Process one doc at a time, one artifact at a time,
  to keep the tool output buffer from overflowing. Do not fan out the drift
  check across docs in parallel.
- **Prompt for missing links.** If the doc's symbol/path links are missing or
  stale, prompt the user: "The doc does not name the source for <claim>.
  Adding the path/symbol makes the next drift check cheaper." Record the gap
  in the report as a **DOC LINK** note, not a drift finding.

## Decision Gates

| Condition | Action |
|---|---|
| Doc has no code claims (pure process note) | Stop — nothing to drift against; report so. |
| Artifact found but the code does *more* than the doc says | Not drift — note as a doc gap if material. |
| Search cannot locate the code the doc names | Record **UNLOCATED** with the search trace — not a drift finding. |
| A claimed number/constant differs from the code's value | Reportable drift (logic shortcut) — cite both values with file:line. |
| BLOCKER divergence at ship gate | **WAIVE or FIX** — named human waiver with recorded risk, or fix before archive. |
| MAJOR divergence at ship gate | Surface in the ship report; user decides. Not a hard block. |
| Code matches the doc for every artifact | Report exactly: "No drift detected." — nothing else. |
| Drift claim based on memory, not a file read | Re-read the live file before reporting it. |

## Output Contract

The drift report is the deliverable — appended to
`{YYYY-MM-DD}_DRIFT_REPORT.md` (working directory, or the change-specs dir for
an in-flight change). Exact shape:

- One line per divergence, prefixed with its category tag
  (`MISSING FEATURE` / `INCOMPLETE MODEL` / `LOGIC SHORTCUT` /
  `ARCH DIVERGENCE`), each naming the doc claim and the code state with
  file:line.
- `DOC LINK` notes for missing/stale source links — separate from findings.
- `UNLOCATED` entries with their search trace — separate from findings.
- If nothing diverged: exactly `No drift detected.` — no other content.
- No matching items, no prose paragraphs, no "looks correct overall."

## The Gate (at ship)

When run as a pre-archive gate (after `execute`, before `ship` archives the
change):

- **BLOCKER divergence** (a missing feature the spec's Must-Haves claim, or an
  architectural divergence that breaks a stated invariant) → the gate is
  **WAIVE or FIX**. The user must either waive it (named human, recorded risk)
  or the change is not archived. A BLOCKER drift that is silently passed is a
  shipped inconsistency.
- **MAJOR divergence** (incomplete model, logic shortcut) → surface in the
  ship report; the user decides. Not a hard block, but visible.
- **No drift** → the gate passes silently. The report says "No drift
  detected." and the change archives.

The drift report is **context, not a replacement** for the QA gate. QA verifies
the change's own acceptance scenarios; drift verifies the change's claims
against the broader spec. Both are needed.

## Red Flags

- A drift report that lists matching items (it is a status report, not a drift
  report)
- A "No drift detected" with no file reads behind it (the check was not run)
- A BLOCKER divergence archived without a named waiver
- A drift finding based on memory of the code, not a file read
- A doc link that was never updated after the code moved (the next drift
  check will UNLOCATE every artifact)
- The drift check skipped because "the tests passed" — tests verify behavior,
  not doc-accuracy

## Verification

- [ ] Every named artifact in the doc was located (or marked UNLOCATED with a
      search trace)
- [ ] Every located artifact was read from the live file (not from memory)
- [ ] The report contains only divergences (no matching items)
- [ ] "No drift detected" is present only when every artifact matched
- [ ] BLOCKER divergences have a named waiver or a fix commit before archive
- [ ] Doc-link gaps are noted (DOC LINK), not silently dropped
- [ ] Every drift claim cites a file read or search trace (a claim is not evidence — per `../_shared/rules/evidence-claims.md`)

## How to measure it

Per `_shared/craft/measurement.md`.

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | % of drift checks that locate 100% of named artifacts (no UNLOCATED) | drift report | should rise |
| Lagging | BLOCKER drifts discovered at ship (after they should have been caught at execute) | drift report | should fall |

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "The tests pass, so the code matches the spec" | Tests verify the spec's acceptance scenarios. They do not verify that every claim in the design doc is implemented. A missing feature with no test is not caught by a green suite. |
| "The doc is old, the code is right" | Then the doc is drift. The report says so. An old doc that contradicts the code is a trap for the next reader — fix the doc or fix the code. |
| "It's a minor field, not worth reporting" | A missing field on a data model is an INCOMPLETE MODEL. The next integration that expects that field breaks. Report it. |
| "I can't find the code, so it probably doesn't exist" | UNLOCATED is not the same as MISSING. If you searched and could not find it, say UNLOCATED. If the symbol is genuinely absent, that is a MISSING FEATURE — but the search trace is the evidence. |
| "Drift is the doc's problem, not mine" | If you changed the code and the doc now disagrees, the drift is yours. The doc is the specification; the code that drifted from it is the change you made. |

## References

- [../skill-creator/references/rationalizations.md](../skill-creator/references/rationalizations.md) — the authoring rationalizations (script extraction, size, spine sections); load when you or the user push back on this skill's shape
