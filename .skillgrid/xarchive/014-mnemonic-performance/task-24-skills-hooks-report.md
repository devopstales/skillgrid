# Step 24 — Skills Framework + Lifecycle Hooks (014-mnemonic-performance)

## Summary

Implemented the skills framework (skills as observations with `memory_type=skill`),
intent-based skill matching, the four lifecycle hooks (session-start / pre-edit /
prompt-submit / session-stop), the opt-in-with-timeout behavior, and the
`mem skills` / `mem hook` CLI. All changes are additive: a new `skills.go` +
`skills_test.go` (and `mem_skills_test.go`), the `skill` memory_type added to
step 18's constants/validation, the `mnemonic.hooks` config section, and the two
CLI subcommands. No tool contract changed shape; `modernc.org/sqlite` (no CGO)
stays; no new migration table was needed (skills are plain observations).

## 24.1 — Skills stored as observations, matched by intent [RED]

**Storage:** A skill is an observation with `memory_type = "skill"` (the new 10th
valid category, `memory.MemoryTypeSkill`) and a `topic_key` shaped
`skill/<intent>/<lang>`:
- `intent` — one of the 4-intent contract (exploration / debugging / review /
  refactor, reused from steps 19/22). This is the single source of the skill's
  applicable intent.
- `lang` — an empty language tag (any-language skill) or a specific project
  language (e.g. `go`).

**Added `skill` to step 18's valid types:** Yes. `MemoryTypeSkill = "skill"` was
added to the `MemoryType*` constant block, the `memoryTypes` lookup set, the
two error messages (Save + RecentWithType), and the surrounding comments in
`internal/mnemonic/memory/service.go`. `IsValidMemoryType("skill")` now returns
true; saving with `memory_type: skill` round-trips.

**MatchSkills logic** (`Skill` struct + `MatchSkills(ctx, intent, mentionedFiles,
projectLangs)` in `skills.go`):
1. Build the "wanted languages" set from `projectLangs` plus the extensions of
   `mentionedFiles` (a file like `auth.go` signals `go` via `extLanguage`).
2. SQL scan: `memory_type = 'skill' AND topic_key LIKE 'skill/<intent>/%'`
   (intent-scoped from the start), capped at `skillMatchLimit` (100).
3. Relevance: a skill whose (non-empty) language is in the wanted set scores
   1.0; everything else (any-language, or a language not wanted) scores 0.5.
   With no wanted languages at all, all skills are equally relevant.
4. Sort by relevance descending, then `topic_key` for determinism. Return
   `[]Skill` (id, title, content, intent, language, topic_key, relevance).

## 24.2 — Lifecycle hooks inject context at the right moments [RED]

`RunHook(ctx, hookType string, payload HookPayload) (HookResult, error)` in
`skills.go`. Four hook types (each a pure, deterministic pass over the store):

- **session-start** — classify the query's intent (`ClassifyWorkIntent`), return
  the project's recent memories (`Recent`) + the skills matched by that intent
  (`MatchSkills`). With no query, intent = exploration and only memories.
- **pre-edit** — run `AnalyzeImpact` (step 23) for the single target file and
  return the risk classification (`*ImpactResult`: is_hub, dependents,
  impact level). Requires `payload.File`.
- **prompt-submit** — classify the prompt's work intent (`ClassifyWorkIntent`)
  and return it in `HookResult.Intent`.
- **session-stop** — run the session-close distillation for the session. To avoid
  an import cycle (the `layer` package imports `memory`), distillation is
  delegated through an injectable seam `SetDistillRunner(func(ctx, svc,
  sessionID) (bool, error))`; the service layer wires the real `layer.Distill` at
  open time. A nil runner reports `Distilled: false` (a no-op, not an error).

**Opt-in + timeout (24.3):** `SetHooks(HooksConfig{Enabled, Timeout})` — Enabled
defaults to false (no hook runs → `IsHooksDisabled` sentinel); Timeout defaults
to `DefaultHookTimeout` (30s). `RunHook` wraps the hook work in
`context.WithTimeout(ctx, timeout)` and maps a `context.DeadlineExceeded` to a
descriptive `hookTimeoutError` (`IsHookTimeout`). Unknown hook types are rejected
before any work runs. A test seam `SetHookFunc(hookType, fn)` replaces one hook's
work so the timeout is exercised deterministically (the seam observes
`ctx.Done()`).

**Structs** (in `skills.go`):
```go
type HookPayload struct {
    File      string `json:"file,omitempty"`
    Query     string `json:"query,omitempty"`
    SessionID string `json:"session_id,omitempty"`
}
type HookResult struct {
    Memories  []Observation `json:"memories,omitempty"`
    Skills    []Skill       `json:"skills,omitempty"`
    Intent    Intent        `json:"intent,omitempty"`
    Risk      *ImpactResult `json:"risk,omitempty"`
    Distilled bool          `json:"distilled,omitempty"`
}
type HooksConfig struct {
    Enabled bool
    Timeout time.Duration
}
```

**Config:** New `mnemonic.hooks` section in `config/load.go`
(`Hooks{Enabled, Timeout}` + `hooksSection` + `mergeHooks`), wired in
`service/service.go` (`mem.SetHooks(memory.HooksConfig{...})`). `Enabled` applies
as-is (absent → false); a blank/malformed timeout keeps the 30s default.

## 24.4 — `mem skills` and `mem hook` CLI [AFK]

In `cmd/skillgrid/mem.go` (manual flag dispatcher, matching the existing pattern):
- `mem skills list` — lists the project's skills (`RecentWithType(skill, 100)`),
  each with id/title/intent/language/topic_key/content.
- `mem skills add <intent> <name> <content> [lang]` — creates a skill
  observation (`memory_type=skill`, `topic_key=skill/<intent>/<lang>`).
- `mem hook list` — prints the 4 configured hook types + the project's
  `enabled` state (from config) + the effective timeout.
- `mem hook run <type> [--query Q] [--hook-file F]` — runs one hook and prints
  its result JSON. (A `--query` flag was added; `--file` was already used by
  export, so the hook's file flag is `--hook-file`.) The CLI is the explicit
  "run this hook" surface, so `runMemHook` arms `Enabled: true` before dispatch;
  a still-disabled run reports `hooks_disabled`.

## RED → GREEN evidence

- 24.1/24.2/24.3: wrote `skills_test.go` first → `go test ... -run
  'TestSkillsMatchingByIntent|TestLifecycleHookSessionStart|TestHooksOptInWithTimeout'`
  failed to build (undefined `MatchSkills`, `RunHook`, `HooksConfig`, `HookPayload`,
  `SetHooks`) → implemented `skills.go` + the `skill` memory_type + config → the
  three tests PASS.
- 24.4: wrote `mem_skills_test.go` first → `go test ./cmd/skillgrid/ -run
  TestMemSkillsAndHookCLI` failed (`error: unknown mem command "skills"`) →
  implemented the subcommands → PASS.

## Test evidence (final, HEAD = 5ece3d7)

Focused (all PASS):
```
=== RUN   TestSkillsMatchingByIntent
--- PASS: TestSkillsMatchingByIntent (0.61s)
=== RUN   TestLifecycleHookSessionStart
--- PASS: TestLifecycleHookSessionStart (0.75s)
=== RUN   TestHooksOptInWithTimeout
--- PASS: TestHooksOptInWithTimeout (1.50s)
```

Full relevant suites:
```
ok  skillgrid-cli/internal/mnemonic/memory   30.019s
ok  skillgrid-cli/cmd/skillgrid             100.865s
```

005/008/010/011/013 baselines:
```
ok  skillgrid-cli/internal/mnemonic/store       5.863s
ok  skillgrid-cli/internal/mnemonic/service    50.104s  (see note below)
ok  skillgrid-cli/internal/mnemonic/codeindex 18.359s
ok  skillgrid-cli/internal/mnemonic/route      1.478s
ok  skillgrid-cli/internal/mnemonic/affected   8.862s
ok  skillgrid-cli/internal/mnemonic/community  3.692s
ok  skillgrid-cli/internal/mnemonic/pdg        6.497s
```

**Note:** `service.TestReindexStructuralIsEmbedderFree` fails with
`count gamma: sql: database is closed`. This is PRE-EXISTING and unrelated to
step 24: it fails identically at the step-23 baseline (`ea37f6b`, verified via a
throwaway worktree) — a flaky "database is closed" in the reindex-structural test
(gamma store handle closed during a count). Excluding that one test, the service
suite is `ok`.

## Commits

- `c28fa0c` feat(mnemonic): skills framework, lifecycle hooks, opt-in with timeout
- `db30b84` feat(mnemonic): add mem skills and mem hook CLI subcommands
- `5ece3d7` style(mnemonic): gofmt alignment for skills/hooks files

## Concerns

1. **Pre-existing flaky baseline** — `service.TestReindexStructuralIsEmbedderFree`
   fails at the step-23 baseline and is unrelated to this step (database-closed
   race). Not introduced here; left as-is.
2. **`mem hook run` arms `Enabled: true`** — the CLI is the explicit hook-execution
   surface, so `runMemHook` forces the opt-in switch on before dispatch. This is a
   deliberate UX choice (the command means "run this hook now"); the config
   `mnemonic.hooks.enabled` still governs the non-CLI (MCP/HTTP) paths. If the
   intent is that the CLI should honor the config gate too, the `SetHooks` call in
   `runMemHook` should be removed and the `--query`/`--hook-file` run would then
   report `hooks_disabled` when config is off.
3. **No 034 migration** — skills are stored as plain observations with
   `memory_type=skill`; no new table/column was required, so migration `034` was
   not needed (the next free number remains 034).
