# Step 22 — Working set + intent classification + universal context envelope

Change: `014-mnemonic-performance` / `22-context-envelope`
Branch: `feat/v2-structure`
Verdict: **PASS**

## What was implemented per sub-task

### 22.1 — Working set tracks files edited, edit counts, net line deltas

Created `skillgrid-cli/internal/mnemonic/memory/envelope.go` with an in-memory,
session-scoped `WorkingSet`. A fresh CLI process cannot rely on prior in-process
state, so the working set is built per invocation from explicit `RecordEdit`
calls (no DB table — the design note "likely in-memory" was correct; no
`032_working_set.sql` migration was needed).

- `NewWorkingSet(projectRoot string) *WorkingSet` — scopes the set to a project
  root.
- `RecordEdit(path string, linesAdded, linesRemoved int)` — accumulates per-file
  edit counts and net line deltas (`added - removed`); files **outside** the
  project root are excluded.
- `Files() []WorkingFile` — a deterministically (path-sorted) snapshot.
- Hub-file status reuses step 21's `isHubFile` / `hubFilePatterns` (the same
  fixed hub-file list) so the flag is consistent with the handoff.

`WorkingFile` struct (quoted):

```go
type WorkingFile struct {
	Path      string `json:"path"`
	EditCount int    `json:"edit_count"`
	NetLines  int    `json:"net_lines"`
	IsHub     bool   `json:"is_hub"`
}
```

### 22.2 — Intent classification returns exploration/debugging/review/refactor

**Decision: created a NEW function `ClassifyWorkIntent` rather than simply
aliasing step 19's `ClassifyIntent`, but REUSED step 19's `Intent` type** (the
same 4-intent set). Why: step 19's `ClassifyIntent` returns the same four
intents, but its keyword table (tuned for *search* queries: `fail`, `broken`,
`check the changes`, `pr feedback`, `split`, `rename`, …) does **not** match
the step-22 *work* vocabulary. The three step-22 RED cases that fail against
step 19's patterns:
- `"fix the null pointer in auth.go"` — step 19 has no `fix` keyword →
  misclassified as exploration.
- `"check the PR for payment.go"` — step 19's review keyword is `pr feedback`
  (a multi-word phrase); bare `pr`/`check` don't match → misclassified.
- `"extract the validation into a separate function"` — step 19 has no `extract`
  keyword → misclassified.

So `ClassifyWorkIntent` carries its own `workIntentKeywords` table (debugging:
`fix, bug, error, crash, fail, …`; review: `check, review, pr, pull request,
diff, …`; refactor: `extract, rename, move, split, …`; exploration: default
catch-all), checked in a fixed deterministic order. This is additive (step 19's
`ClassifyIntent` and its `Intent` type are untouched) and avoids both a name
collision and keyword duplication.

### 22.3 — ContextEnvelope JSON

Implemented in `envelope.go`. The envelope combines project metadata, working
set, intent, matched skills, and handoff refs. `ContextEnvelope` struct
(quoted):

```go
type ProjectMeta struct {
	Name      string   `json:"name"`
	FileCount int      `json:"file_count"`
	Languages []string `json:"languages"`
}

type ContextEnvelope struct {
	Project       ProjectMeta   `json:"project"`
	WorkingSet    EnvelopeWS    `json:"working_set"`
	Intent        Intent        `json:"intent"`
	MatchedSkills []string      `json:"matched_skills"`
	HandoffRefs   []string      `json:"handoff_refs"`
}

type EnvelopeWS struct {
	Files []WorkingFile `json:"files"`
}
```

- `Service.GenerateContextEnvelope(ctx, ws, intent, dataDir)` — the store-driven
  entry point the brief specifies. It reads the project's indexed **file count**
  (via the step-21 `codeindexFiles` helper) and **languages** (the distinct
  `symbols.language` values from the codeindex), fills matched skills via
  `MatchedSkillsForIntent`, and sets the handoff ref to
  `handoff.latest.json` (step 21's `HandoffPath`).
- `NewContextEnvelope(ws, intent)` — a store-free assembler (used by tests and
  as the building block).
- **Size limit (rollback boundary):** `MarshalJSON` enforces the configurable
  `mnemonic.envelope.max_size` cap (default 64KB, `defaultEnvelopeMaxSize`). If
  the full envelope exceeds the limit it progressively drops the smallest
  net-delta working-set files first (most-changed files survive longest) until
  it fits. A dedicated `envelopeWire` struct (no `MarshalJSON` method) is used
  for the raw encoding — this avoids the infinite-recursion pitfall where
  `json.Marshal` on the envelope type would re-enter `MarshalJSON`.

### 22.4 — `mem context` CLI with `--envelope` flag

Modified `skillgrid-cli/cmd/skillgrid/mem.go` (the manual flag/dispatcher
pattern, not cobra):
- Added the `--envelope` bool flag.
- `runMemContext` now takes `dataDir` and `envelope`. Default behavior is
  unchanged (recent session summaries, budgeted read path). With `--envelope`
  it calls `h.Memory().GenerateContextEnvelope(...)` and writes the full JSON to
  stdout. The working set starts empty for a fresh CLI process (no prior
  session edits in-process); the intent defaults to exploration (no query
  passed).
- `TestMemContextCLI` (new `mem_context_test.go`) asserts: default `mem context`
  prints a `sessions` summary; `--envelope` prints valid JSON with the
  `project`, `working_set`, `intent`, `matched_skills`, `handoff_refs` keys, the
  project name, a non-empty `matched_skills` for the default intent, and a
  `handoff_refs` pointing at `handoff.latest.json`.

## Consumed (not rewritten)
- Step 19 `ClassifyIntent` / `Intent` type (`retrieval.go:70`) — type reused.
- Step 21 `GenerateHandoff` / `isHubFile` / `hubFilePatterns` / `codeindexFiles`
  / `HandoffPath` (`handoff.go`) — hub flag + file count + handoff ref.
- Step 08 `improve()` / step 13 importance — not directly wired (the envelope
  assembles working set + intent + skills + handoff; re-ranking is a read-path
  concern, not envelope assembly).

## RED → GREEN evidence

- **22.1 RED**: `go test ... -run 'TestWorkingSetTracking'` →
  `build failed` (`undefined: NewWorkingSet`, `undefined: WorkingFile`).
- **22.2 RED**: `TestIntentClassificationInEnvelope` → after a first impl that
  aliased step 19's `ClassifyIntent`, 3 cases failed
  (`fix…=exploration want debugging`, `check the PR…=exploration want review`,
  `extract…=exploration want refactor`) → replaced with `ClassifyWorkIntent`.
- **22.3 RED**: `TestContextEnvelopeStructure` → `build failed`
  (`undefined: NewContextEnvelope`); first impl hit a **stack overflow** from
  `MarshalJSON` recursion → fixed with `envelopeWire`.
- **22.4 GREEN**: `TestMemContextCLI` passed.

## Full test output

Focused suite:

```
$ go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestWorkingSetTracking|TestIntentClassificationInEnvelope|TestContextEnvelopeStructure|TestContextEnvelopeSizeLimit' -count=1 -v
=== RUN   TestWorkingSetTracking
--- PASS: TestWorkingSetTracking (0.00s)
=== RUN   TestIntentClassificationInEnvelope
--- PASS: TestIntentClassificationInEnvelope (0.00s)
=== RUN   TestContextEnvelopeSizeLimit
--- PASS: TestContextEnvelopeSizeLimit (0.00s)
=== RUN   TestContextEnvelopeStructure
--- PASS: TestContextEnvelopeStructure (0.00s)
PASS
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  0.924s
```

Full memory package:

```
$ go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  42.006s
```

Full CLI package:

```
$ go test ./skillgrid-cli/cmd/skillgrid/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  83.767s
```

Baselines (005/008/010/011/013):

```
$ go test ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/service/... \
    ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/route/... \
    ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/internal/mnemonic/community/... \
    ./skillgrid-cli/internal/mnemonic/pdg/... -count=1
ok  .../store        6.829s
FAIL .../service     TestReindexStructuralIsEmbedderFree — "sql: database is closed"
ok  .../codeindex   18.671s
ok  .../route        4.771s
ok  .../affected    16.079s
ok  .../community    2.632s
ok  .../pdg          6.360s
```

### Pre-existing failure (not caused by step 22)

`TestReindexStructuralIsEmbedderFree` (service pkg) fails with
`sql: database is closed`. Verified pre-existing: `git stash` + re-run on HEAD
(`6d50101`) reproduces the identical failure **without** my changes. My changes
are additive and touch only the `memory` and `cmd/skillgrid` packages; the
service package's reindex test is untouched. Treated as a pre-existing flake
(counting-spy reindex race on a concurrently-closed DB).

## Commits

| SHA | Subject |
|-----|---------|
| `4eedadd` | feat(mnemonic): working set, intent classification, and universal context envelope |
| `897dd6e` | feat(mnemonic): add mem context CLI with envelope flag |

(22.1/22.2/22.3 are coupled in a single `envelope.go` + `envelope_test.go` pair,
so they ship as one commit; 22.4 — `mem.go` + `mem_context_test.go` — is a
second commit. No `git add -A`; only the 4 changed files were staged.)

## Concerns

- **Pre-existing service-pkg failure** (`TestReindexStructuralIsEmbedderFree`)
  is a "database is closed" race in a reindex counting-spy test; it predates this
  step and is out of scope for step 22 (different package, untouched files).
- **Working set is in-memory/session-scoped**, so a fresh `mem context
  --envelope` CLI process shows an empty working set (by design — a CLI process
  cannot see a prior session's in-memory edits). The envelope still carries the
  project metadata, intent, matched skills, and handoff refs. If cross-process
  working-set persistence is wanted later, it would need a `032_working_set.sql`
  migration (not required by the brief).
- **Commit granularity**: the brief lists four per-sub-task commits; because
  22.1/22.2/22.3 share one source file they are committed together, with the
  message calling out each sub-task. 22.4 is its own commit.
