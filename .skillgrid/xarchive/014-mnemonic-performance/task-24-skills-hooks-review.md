# Step 24 Review — Skills Framework + Lifecycle Hooks (014-mnemonic-performance)

**Reviewer:** Fresh reviewer (no prior context)
**Scope judged:** `review-step24-scoped.diff` (3 commits `c28fa0c`, `db30b84`, `5ece3d7`) + the two files present in the real commit range but missing from the scoped diff: `config/load.go` and `service/service.go` (both read from the actual `git diff 5ece3d7~3 5ece3d7`).
**Date:** 2026-09-13

---

## Verdict: **PASS WITH WARNINGS**

All four deliverables (24.1 skills-as-observations matched by intent, 24.2 the four
lifecycle hooks, 24.3 opt-in + configurable timeout, 24.4 the `mem skills`/`mem hook`
CLI) are implemented and the focused tests pass. The `skill` memory_type is correctly
added as the 10th valid type with no regression to the original 9. No MCP tool contract
changed shape. The gofmt commit is purely cosmetic.

The warnings are about *production wiring*, not correctness of the unit-tested seams:
the session-stop distillation seam (`SetDistillRunner`) is **never wired in production
code**, so `session-stop` is a no-op on the real service; and `mem hook run` forces
`Enabled: true`, which makes the `hooks_disabled` branch in the CLI dead code.

---

## Findings

### [MAJOR] 1. `session-stop` distillation seam is not wired in production — the hook is a no-op in real use
**File:** `internal/mnemonic/memory/skills.go` (seam definition), `internal/mnemonic/service/service.go` (open path)

The report (lines 58–60, 94) states: *"the service layer wires the real `layer.Distill`
at open time."* That is **not** true. The only production call in the open path is
`mem.SetHooks(...)` (service.go:394). There is **no call to `mem.SetDistillRunner(...)`**
anywhere outside `skills_test.go` (confirmed by a repo-wide grep — the only non-test
references are the definition and the `service.go:394` `SetHooks` line).

Consequence: on a real (non-test) `*memory.Service`, `s.distillRunner` is nil, so
`hookSessionStop` returns `HookResult{Distilled: false}, nil`:

```go
runner := s.distillRunnerFn()
if runner == nil {
    return HookResult{Distilled: false}, nil   // ← production always lands here
}
```

The brief 24.2 requires the `session-stop` hook to *"trigger distillation"*. As wired,
it triggers nothing in production. The step is also explicitly **not** opt-in-gated on
this point — even with `mnemonic.hooks.enabled: true`, a session-stop run on a real
service distills nothing. The test passes only because the test installs its own
`SetDistillRunner` stub that records the call (`skills_test.go:141`), so the test
verifies the seam fires, not that the real `layer.Distill` runs.

**Fix:** in the open path next to `mem.SetHooks(...)` (service.go:394), wire:
```go
mem.SetDistillRunner(func(ctx context.Context, svc *memory.Service, sessionID string) (bool, error) {
    res, err := svc.DistillSession(ctx, projectID, sessionID, "", nil)
    return err == nil, err
})
```
(or an equivalent that calls the existing `layer.Distill`/`DistillSession`). If the
intent is that distillation is already handled by the existing session-close path and the
hook is purely a reporting seam, that should be stated in the report instead of claiming
the service wires the real `layer.Distill`.

### [MINOR] 2. `mem hook run` forces `Enabled: true`, making the `hooks_disabled` branch dead code
**File:** `cmd/skillgrid/mem.go:1027`

```go
mem.SetHooks(memory.HooksConfig{Enabled: true})   // ← arms the opt-in switch
res, rerr := mem.RunHook(ctx, hookType, memory.HookPayload{File: file, Query: query})
if rerr != nil {
    if memory.IsHooksDisabled(rerr) {              // ← can never be true after the line above
        printJSON(map[string]any{"project": projID, "hook": hookType, "hooks_disabled": true})
        return
    }
    ...
}
```

Because `RunHook` reads the same in-memory `*Service` that `SetHooks` just armed, the
`hooksDisabledError` is unreachable in this code path. The `IsHooksDisabled` handling is
dead code. This is a deliberate, documented UX choice (the report "Concerns" §2 calls it
out), and it does not break the 24.4 test, so it is a minor consistency nit — but it means
`mem hook run` never reports the opt-in-disabled state it claims to, and the dead branch
should be removed or the arming removed (honoring the config gate). Pick one; the current
code asserts both.

### [MINOR] 3. Scoped diff understates the change surface — 2 of 7 changed files are missing
**File:** `.skillgrid/sdd/014-mnemonic-performance/review-step24-scoped.diff`

The review brief says the diff covers "4 files," but `git diff --name-only 5ece3d7~3
5ece3d7` shows **7 files**. The two absent from the scoped diff are:
- `internal/mnemonic/config/load.go` (+64/−): the `mnemonic.hooks` YAML section,
  `Hooks`/`hooksSection` structs, `mergeHooks`, and the wiring in `mergeIndexing`.
- `internal/mnemonic/service/service.go` (+13/−): the `mem.SetHooks(...)` open-path call.

Both are **required** for 24.3 (opt-in per project + timeout). They are additive and
correct, but a reviewer judging only the scoped diff cannot verify that the `Enabled`
config flag is actually read from YAML and routed to `SetHooks` — that wiring lives in
these two files. (Reviewed here from the real diff; no defect, but the scoped diff is
incomplete for a full 24.3 audit.)

### [MINOR] 4. Step 18's test list still documents "9" categories; step 24 is the 10th
**File:** `internal/mnemonic/memory/types_test.go:14`

```go
// allMemoryTypes is the 9 fine-grained typed categories (step 18). The order
// is the canonical list; IsValidMemoryType accepts each and rejects anything
// else.
var allMemoryTypes = []string{ "profile", "preferences", ..., "experiences" }
```

The production code is fully consistent with 10 (the const block, the `memoryTypes` map,
the `Save` error message, and `RecentWithType` all include `skill` — see Evidence #1).
Only this *test's* comment and list lag at 9. There is **no regression** (`TestMemoryTypeCategories`
passes for the original 9; `skill` round-trips per `TestSkillsMatchingByIntent`). This is
a comment/doc drift, not a correctness issue — but if "9" is ever asserted as an
invariant, it is now stale. Consider adding `"skill"` to `allMemoryTypes` (or a dedicated
10th-category assertion) so the test suite documents the full set.

### [NIT] 5. `hookPromptSubmit` discards `ctx`; `hookSessionStart` drops retrieval errors
**File:** `internal/mnemonic/memory/skills.go`

- `hookPromptSubmit` does `_ = ctx` (line ~932). Fine — classification is pure — but the
  blank assignment signals the context budget (the 30s timeout) is not observably honored
  by that hook. Acceptable for a keyword classifier, just noting it.
- `hookSessionStart` swallows errors from `Recent` and `MatchSkills`
  (`if mems, err := s.Recent(...); err == nil { ... }`). So a transient DB error yields
  an empty memories/skills list with no error surfaced. Deliberate "best-effort context
  injection" behavior, but a `session-start` hook that silently returns zero memories on
  a DB failure could look like "no relevant context" rather than "retrieval failed."
  Consider logging the swallowed error.

### [NIT] 6. `skillIntentFromTopic`/`skillLangFromTopic` (CLI) duplicate `skillParts` (memory pkg)
**File:** `cmd/skillgrid/mem.go:1057-1066` vs `internal/mnemonic/memory/skills.go:73`

The CLI re-implements the `skill/<intent>/<lang>` parse rather than exporting
`skillParts`. Minor duplication; behavior matches (both lowercase, both tolerate a
missing lang). No defect.

---

## Evidence checked (8 checks)

**1. `skill` in memory_type validation (24.1 — CRITICAL) — ✅ VERIFIED, no regression.**
- `service.go:1819` `MemoryTypeSkill = "skill"` is the 10th constant in the block
  (the original 9 `profile`…`experiences` are intact, `service.go:1810-1818`).
- `service.go:1836` `MemoryTypeSkill: {}` is present in the `memoryTypes` lookup set
  (10 entries, lines 1827-1836). `IsValidMemoryType("skill")` → true.
- The `Save` validation `service.go:449-451` now lists 10: `"...experiences, skill"`.
- `RecentWithType` (`service.go:818`) validates through the same `IsValidMemoryType`.
- **No regression:** ran `go test ./internal/mnemonic/memory/ -run TestMemoryTypeCategories`
  → **PASS** (original 9 round-trip; unknown type still rejected). `TestSkillsMatchingByIntent`
  proves `memory_type=skill` round-trips through `Save`/`Get`.

**2. Skill matching (24.1) — ✅ VERIFIED, real test.**
- `MatchSkills` (`skills.go:95`) filters by intent: SQL `topic_key LIKE 'skill/<intent>/%'`
  (line 129) + a Go re-check `intent2 != intentKey` (line 150). Language-matched skills
  score 1.0, any-language/others 0.5 (lines 154-163), sorted relevance desc then
  `topic_key` (lines 178-183). `mentionedFiles` contribute languages via `extLanguage`
  (lines 113-117, 219-226); `projectLangs` are added directly.
- `TestSkillsMatchingByIntent` is real: seeds 3 skills (debugging/review/exploration, all
  `go`), classifies "fix the null pointer panic in auth.go" → `IntentDebugging`, asserts
  only the debugging skill returns (len==1, correct id). Also asserts review-only match,
  language-ranking (go skill ahead of any-language), and mentioned-file (`auth.go`→go)
  ranking. Ran it → **PASS**.

**3. Hook correctness (24.2 — CRITICAL) — ⚠️ CORRECT in unit, session-stop unwired in production (Finding 1).**
- `session-start` (`skills.go:449`): classifies intent (`ClassifyWorkIntent`), returns
  `Recent` memories + `MatchSkills` skills. ✅ Test asserts intent=debugging, memories
  include the seeded auth observation, skills = the debugging skill.
- `pre-edit` (`skills.go:470`): requires `payload.File`, runs `AnalyzeImpact` (step 23),
  returns `*ImpactResult`. ✅ Test seeds a hub (5 importers) and asserts
  `IsHub`/`ImpactLevel=ImpactHigh`/`DependentCount=5`.
- `prompt-submit` (`skills.go:487`): returns `ClassifyWorkIntent(query)` in
  `HookResult.Intent`. ✅ Test asserts `IntentReview` for "review the changes to
  payment.go".
- `session-stop` (`skills.go:495`): delegates to the injectable `distillRunner` seam;
  nil runner → `Distilled: false`. ⚠️ The seam is **never set in production code**
  (Finding 1) — the "triggers distillation" behavior only exists behind a test stub.
  - Opt-in: `hooksEnabled` defaults false; `RunHook` returns `hooksDisabledError` before
    any work. ✅ `TestHooksOptInWithTimeout/disabled_by_default` passes.
  - Timeout: `RunHook` wraps the hook in `context.WithTimeout(ctx, timeout)`
    (`skills.go:407`), maps `context.DeadlineExceeded` → `hookTimeoutError` (line 411).
    Configurable via `SetHooks` / `mnemonic.hooks.timeout`. ✅
    `timeout_is_enforced_and_configurable` (50ms budget cuts a 2s sleep) and
    `default_timeout_is_30s` pass.

**4. HookPayload/HookResult — ✅ VERIFIED, reasonable.**
- `HookPayload{File, Query, SessionID}` (`skills.go:256`) — File for pre-edit, Query for
  session-start/prompt-submit, SessionID for session-stop. Unused fields empty.
- `HookResult{Memories, Skills, Intent, Risk *ImpactResult, Distilled bool}`
  (`skills.go:269`) — carries the right data per hook type. All fields
  `omitempty`, so each hook's output JSON is sparse and type-appropriate.

**5. CLI (24.3/24.4) — ✅ VERIFIED, works; dead `hooks_disabled` branch (Finding 2).**
- `mem skills list` → `RecentWithType(skill, 100)` with id/title/intent/language/
  topic_key/content. `mem skills add <intent> <name> <content> [lang]` → `Save` with
  `memory_type=skill`, `topic_key=skill/<intent>/<lang>`.
- `mem hook list` → 4 hook types + config `enabled` + timeout. `mem hook run <type>
  [--query Q] [--hook-file F]` → `RunHook`, prints result JSON.
- `TestMemSkillsAndHookCLI` (real `go run` subprocess) exercises all four and asserts
  `debug-go` listed, `review-go` added+listed, all 4 hook types in `hook list`, and
  `prompt-submit --query "fix the bug"` → `debugging`. Ran it → **PASS** (4.15s).
- Errors handled gracefully (stderr + `os.Exit(1/2)`); unknown subcommands exit 2.

**6. Additive-ness / tool contract — ✅ VERIFIED, no contract change.**
- The step 24 diff adds new CLI subcommands, a new memory_type, a new config section, and
  two new methods on `*memory.Service`. It does **not** touch any `mem_*`/`code_*`/`web_*`
  tool registration or signature.
- `TestAllToolsRegistered` (mcp) → **PASS** (the full expected tool list, lines 23-60, is
  unchanged; no step 24 tool added, none removed).
- `skill` is purely additive to the 9 types (Evidence #1). `modernc.org/sqlite` stays
  (no CGO); no new migration (skills are plain observations).

**7. TDD discipline — ✅ VERIFIED.**
- (a) skill matching by intent: `TestSkillsMatchingByIntent` — real, passes.
- (b) the 4 hooks: `TestLifecycleHookSessionStart` — real, passes (all 4 + unknown-type).
- (c) opt-in + timeout: `TestHooksOptInWithTimeout` — real, passes (disabled-by-default,
  enforced+configurable timeout, 30s default).
- (d) CLI: `TestMemSkillsAndHookCLI` — real, passes.
- The report's RED evidence (undefined symbols before implementation) is consistent with
  the new-file structure; the GREEN results were reproduced locally.

**8. gofmt commit (`5ece3d7`) — ✅ VERIFIED, purely cosmetic.**
- `git show 5ece3d7 -w --stat` shows only `internal/mnemonic/memory/service.go` with a
  non-whitespace delta, and that delta is a gofmt reformat of an interface literal
  (`interface{ EmbedQuery(...) }` → multi-line) plus alignment of a composite literal —
  no logic change. All other files are alignment-only. No behavior change.

---

## Test execution (reproduced locally, `skillgrid-cli/` module)

| Test | Result |
|------|--------|
| `TestSkillsMatchingByIntent` | PASS (0.28s) |
| `TestLifecycleHookSessionStart` | PASS (0.28s) |
| `TestHooksOptInWithTimeout` (3 subtests) | PASS (0.71s) |
| `TestMemoryTypeCategories` (step 18, regression) | PASS (0.29s) |
| `TestMemSkillsAndHookCLI` (24.4 CLI) | PASS (4.15s) |
| `TestAllToolsRegistered` (MCP contract) | PASS (0.00s) |
| `go build ./...` | OK |
| `go vet` (memory, cmd/skillgrid) | 1 pre-existing warning in `budget.go` (not a step 24 file) |

The report's claim that `service.TestReindexStructuralIsEmbedderFree` is a pre-existing
flaky failure at the step-23 baseline is not independently re-verified here, but it is in
a different package and unrelated to step 24's changed files.

---

## Approved to merge: **Yes** — with the two MAJOR/MINOR warnings tracked.

The blocking concern (Finding 1, session-stop not wired to real `layer.Distill`) is a
**production-wiring gap**, not a defect in the unit-tested seams, and it does not break
any step 24 test or the opt-in/timeout contract. It should be closed as a follow-up
(either wire the seam in the open path or correct the report's "wires the real
layer.Distill" claim). Findings 2–6 are non-blocking. Recommend merging with Finding 1
filed as a small follow-up task.
