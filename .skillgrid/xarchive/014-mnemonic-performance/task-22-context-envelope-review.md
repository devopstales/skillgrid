# Step 22 Review — Working set + intent classification + universal context envelope

Change: `014-mnemonic-performance` / `22-context-envelope`
Commits: `4eedadd` (envelope.go + envelope_test.go), `897dd6e` (mem.go + mem_context_test.go)
Fresh reviewer, no prior context.

## Verdict

**PASS WITH WARNINGS**

The working set tracking and intent classification — the two areas flagged as
CRITICAL — are both **correct** and genuinely tested. All 7 required checks pass.
The warnings are architectural (the working set is inert in the only shipped
caller) and a keyword-table fidelity gap vs. the brief, none of which block merge.

---

## Findings

### MAJOR — the working set is never populated by any real caller

**File:** `skillgrid-cli/cmd/skillgrid/mem.go:395`

The only shipped call site of the working-set API hardcodes an **empty** root and
never calls `RecordEdit`:

```go
ws := memory.NewWorkingSet("")
intent := memory.ClassifyWorkIntent("")
env, err := h.Memory().GenerateContextEnvelope(hCtx(), ws, intent, dataDir)
```

Consequences:
- Because `projectRoot == ""`, `inProject()` returns `true` for *every* path —
  the "files outside the project are excluded" guarantee that 22.1 exists to
  provide is **never actually enforced** in the CLI (the root is empty, so there
  is no project scope to exclude against).
- `working_set.files` in a real `mem context --envelope` output is **always
  `[]`** — the envelope's working-set section is inert in the one shipped
  consumer.

This is **documented** in the report ("Working set is in-memory/session-scoped…
a fresh CLI process shows an empty working set… by design") and the brief's
22.1.c only asks to "track edits during the session" (in-memory), so it is a
defensible design and not a contract break. But the reviewer should be explicit:
the exclusion logic and per-file accumulation are only *exercised* by the unit
test, not by any production path. If the intended design is a genuinely useful
cross-invocation working set, this needs either (a) a `RecordEdit` hook wired
into the agent's edit calls (MCP tool layer — currently untouched), or (b) the
project root actually resolved from the open project and passed in. As shipped,
the feature works but only proves itself in the test.

**Fix (if the feature is meant to be live):** resolve the project root in
`runMemContext` (e.g. from `svc` / the open project) and thread edit events into
`RecordEdit`; otherwise document the working set as a test/programmatic surface
until a caller exists.

---

### MINOR — `ClassifyWorkIntent` drops the brief's explicit `error` keyword

**File:** `skillgrid-cli/internal/mnemonic/memory/envelope.go:274-277`

The brief 22.2.c specifies the debugging keyword set as
`debugging (fix, bug, error, crash, fail)`. The implementation's table is a
**superset** of every brief keyword set:

```go
{IntentDebugging, []string{
    "fix", "bug", "error", "crash", "fail", "broken", "stack trace",
    "exception", "panic", "regress", "why does", "why is",
}},
{IntentReview,   []string{"check", "review", "pr", "pull request", "diff", "merge"}},
{IntentRefactor, []string{"extract", "rename", "move", "split", "refactor", "restructure", "cleanup", "clean up"}},
```

`fix/bug/error/crash/fail` (debugging), `check/review/PR/diff` (review), and
`extract/rename/move/split` (refactor) are all present, so the 4 required cases
classify correctly. The minor point is purely traceability: the table is a
*broader* transcription of 22.2.c (it adds `broken, regress, why does, why is,
merge, pull request, restructure, cleanup, clean up`). Harmless superset — no
test case regresses; noted so the divergence from the brief's literal list is
on the record.

(Severity downgraded: MINOR / informational — no test case regresses.)

---

### MINOR — handoff is a path *reference*, not the generated handoff

**File:** `skillgrid-cli/internal/mnemonic/memory/envelope.go:199-201`

```go
if ref := HandoffPath(dataDir); ref != "" {
    env.HandoffRefs = []string{ref}
}
```

The brief 22.3.a lists `handoff_refs (path to handoff.latest.json)` as the
expected shape, so a *path reference* is **exactly what the brief specifies** —
this is **correct, not a stub**. Flagging only to confirm it was checked: the
envelope does not embed the handoff *contents* (it points at the file), which
matches the brief and the step-21 `HandoffPath` contract. No fix required.

---

### NIT — `NewWorkingSet` doc comment contradicts the empty-root semantics

**File:** `skillgrid-cli/internal/mnemonic/memory/envelope.go:50-52`

```go
// NewWorkingSet creates an empty working set scoped to projectRoot. An empty
// root means "no scope filter" — every recorded path is accepted.
```

This is internally consistent, but the CLI caller (`mem.go:395`) passes `""`,
which means the *only* real working set in the codebase has **no scope filter**
(see MAJOR finding). The comment is accurate; the usage is what's surprising.
No change needed beyond resolving the MAJOR item.

---

## Evidence checked (7/7)

1. **Working set tracking (22.1)** — ✅ CORRECT.
   - `RecordEdit` (`envelope.go:63-74`) accumulates `EditCount++` and
     `NetLines += linesAdded - linesRemoved` per file (keyed by slash-normalized
     path). Verified by `TestWorkingSetTracking`: AGENTS.md edited twice →
     `EditCount==2`, `NetLines==12` = `(10-2)+(5-1)`; auth.go → `EditCount==1`,
     `NetLines==26` = `30-4`. **Real test, correct math.**
   - Project scoping (`inProject`, `envelope.go:79-86`) uses
     `p == root || HasPrefix(p, root+"/")` with `filepath.ToSlash` on both —
     correct, avoids the `/work/proj` vs `/work/proj2` sibling-prefix bug.
     Outside file `/etc/hosts` excluded (asserted absent). **Real.**
   - Hub flag reuses step-21 `isHubFile` / `hubFilePatterns` (same list —
     AGENTS.md is flagged, auth.go is not). **Consistent with step 21.**
   - In-memory, session-scoped (map in struct, no DB). Confirmed.
   - ⚠️ The MAJOR caveat: this logic is unit-test-exercised only; the shipped
     CLI caller never populates it (empty root).

2. **Intent classification (22.2)** — ✅ CORRECT.
   - `ClassifyWorkIntent` (`envelope.go:137-147`) checks a fixed-order table
     (debugging → review → refactor → default exploration) and defaults to
     `IntentExploration`.
   - `TestIntentClassificationInEnvelope` covers **8 cases** (the 4 required plus
     crash/list/rename/no-markers). All pass: fix→debugging, what→exploration,
     check/PR→review, extract→refactor. **Real, all 4 required cases present.**
   - **Reuses step-19 `Intent` type** — confirmed: `Intent` is defined once in
     `retrieval.go:37`; `envelope.go` uses `Intent`, `IntentDebugging`,
     `IntentReview`, `IntentRefactor`, `IntentExploration` and defines **no
     duplicate type**. ✅
   - **Step-19 `ClassifyIntent` is UNTOUCHED** — `retrieval.go` is not in the
     diff; the new function is a distinct `ClassifyWorkIntent` with its own
     `workIntentKeywords` table. No name collision, no modification. ✅

3. **ContextEnvelope (22.3)** — ✅ CORRECT.
   - `ContextEnvelope` combines `project` + `working_set` + `intent` +
     `matched_skills` + `handoff_refs`. ✅
   - Valid JSON confirmed by `TestContextEnvelopeStructure` (unmarshals, asserts
     all 5 keys, round-trips intent to `debugging`, file_count 42, 2 working
     files, matched_skills `[debugging tdd]`, handoff ref). ✅
   - Size limit: `MarshalJSON`/`marshalJSON` enforce a 64KB cap, dropping
     lowest-net-delta working-set files first; uses a separate `envelopeWire`
     type to avoid `MarshalJSON` re-entrancy (the classic infinite-recursion
     trap). `TestContextEnvelopeSizeLimit` (2000 files → under cap, still valid
     JSON) and my added edge case (single 70KB-path file → still valid JSON,
     empty working set) both pass. **Rollback boundary holds.** ✅
   - `GenerateContextEnvelope` reads file count (`codeindexFiles`) + languages
     (new `distinctLanguages`) from the store; `MatchedSkillsForIntent` is a
     deterministic intent→skills map. Handoff is a path ref (see MINOR #3 —
     correct per brief).

4. **CLI (22.4)** — ✅ CORRECT.
   - `mem context` prints the sessions summary by default; `--envelope` prints
     full JSON. `TestMemContextCLI` (in `cmd/skillgrid`) asserts both, valid
     JSON, all 5 keys, project name, default `exploration` intent, non-empty
     matched_skills, handoff ref. **Passes.**
   - Error handling: `svc.Open` failure → stderr + `exit 1`; envelope gen
     error → stderr + `exit 1`; marshal error → stderr + `exit 1`. The
     no-codeindex case (empty project) is proven graceful by the passing test
     (file_count=0, no error). ✅

5. **Additive-ness / tool contract** — ✅ HELD.
   - Diff touches **only 4 files** (envelope.go, envelope_test.go, mem.go,
     mem_context_test.go). The `mcp/` package is untouched — `mem_*`/`code_*`/
     `web_*` (ln) tool signatures are unchanged. (The `exploreEnvelope` type in
     `tools_code_explore.go` is a pre-existing, unrelated name.) ✅
   - Envelope is purely additive: new file + new `--envelope` flag + new
     `runMemContext` params (the existing default path is preserved unchanged —
     the `limit<=0 / memBudgetOpts` block just moved below the new branch).

6. **No migration** — ✅ CONFIRMED.
   - Working set is in-memory (map), no DB table. Latest migration is
     `031_handoff.sql`; **no `032_working_set.sql`** (or any working-set/envelope
     SQL) was added. As the report states, no migration was needed.

7. **TDD discipline** — ✅ MET.
   - (a) Working set: edit counts, net deltas, project scoping, hub flag — all
     asserted in `TestWorkingSetTracking`. ✅
   - (b) Intent: 8 cases incl. the 4 required — `TestIntentClassificationInEnvelope`. ✅
   - (c) CLI: `TestMemContextCLI` covers default + `--envelope` + JSON validity. ✅
   - RED evidence in the report (build-fail → pass; the 3 misclassifications
     against step-19 patterns; the MarshalJSON stack-overflow → envelopeWire
     fix) is consistent with the final code. I re-ran the focused suite and the
     CLI test — all pass.

---

## Approved to merge

**Yes.** The two CRITICAL areas (working-set tracking correctness, intent
classification) are correct and genuinely tested; all 7 checks pass; the change
is additive, breaks no MCP tool contract, adds no migration, and the
size-limit rollback boundary holds. The MAJOR finding (working set inert in the
shipped CLI caller) is documented, brief-consistent, and non-blocking — track it
as a follow-up if a live working set is the end goal.
