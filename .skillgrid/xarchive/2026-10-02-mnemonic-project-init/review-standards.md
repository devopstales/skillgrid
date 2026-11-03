# Standards Review — mnemonic-project-init

**Range:** `f739d371`..`17652346`  
**Axis:** Standards (glossary + named ADRs + code-standards smell baseline)  
**Independence:** Grade A — reviewed from `review-range.diff` + standards sources only; no other axis reports consulted.  
**Spec completeness:** not judged.

### Strengths

- **ADR-0012 honored.** Ingest goes through `svc.OpenForDirectory` → `memory.Save` with stable `TopicKey` values (`init/docs/<relpath>`). No Claude OS “four KBs”, no second markdown knowledge store, no parallel SQLite. Matches glossary *Project Init* / *Topic Key* and ADR-0012’s one-store rule.
- **ADR-0016 honored.** Change is a DX CLI sequencer (`projectInit` → boot → existing `RunCodeIndex` → ingest). It does not add a new brain engine, MCP tool, or retrieval path.
- **ADR-0019 / ADR-0020 honored.** No new decision body inlined into `ASSUMPTIONS.md`; no teams/ledger ownership move. QA report lives under the SDD ledger path; state only flips phase.
- **Ubiquitous language.** `projectInit` / `skillgrid init`, onboarding step titled “Project init”, `*service.ProjectHandle`, and `TopicKey` stay inside `.skillgrid/artifacts/02-technical-terms.md`. Rejected “Knowledge Base” / four-KB shape does not appear in code or skill prose.
- **Module split.** `init_boot.go` / `init_cmd.go` / `init_ingest.go` each have one nameable reason to change (boot file, orchestration, ingest). Entry `runProjectInit` stays thin.
- **Error taxonomy at the seam.** Boot failure is fatal; index/open/ingest failures accumulate into `initResult.Errors` and still exit success — recoverable vs fatal is visible to the caller.
- **Behavior tests.** Upsert/force/sentinel-once, path jail, missing defaults skipped, non-fatal index/extra-docs, and onboarding handoff assertions exercise real side effects rather than pure mocks.

### Issues

#### Critical (Must Fix)

_(none)_

#### Important (Should Fix)

1. **Hard standard — agent-config placeholder contract clobbered on every init**
   - File: `skillgrid-cli/cmd/skillgrid/init_boot.go` (`renderSentinel`, ~lines 148–154 in the new file)
   - Standard: `.agents/skills/_shared/agent-config/block.md` — `{tracker_line}` and `{memory_line}` must be filled from the active tracker and mnemonic status; the block is the SSOT for sentinel content.
   - Issue: `renderSentinel` always substitutes `defaultTracker` (`None — work local-only from tasks.md.`) and `enabledMemoryLine`. `upsertSentinel` then replaces the entire sentinel region. Re-running `skillgrid init` on a Backlog/GitHub/GitLab project (or mnemonic-disabled project) silently rewrites correct lines.
   - Why it matters: the boot upsert is destructive relative to the documented fill contract; agents then see a wrong tracker/memory spine on every prompt.
   - Fix: Read tracker + mnemonic from `.skillgrid/config.yaml` (same mapping table as `block.md`); preserve existing placeholder lines when already present if detection is deferred.

2. **Hard standard — boot-path selection contradicts agent-config SSOT**
   - File: `skillgrid-cli/cmd/skillgrid/init_boot.go` (`resolveBootPath`, ~lines 102–123)
   - Standard: `block.md` § Idempotent upsert / Gotchas — if `AGENTS.md` is absent, **create `AGENTS.md`**; only add a CLAUDE pointer when both exist. Never treat sole `CLAUDE.md` as the full-block target.
   - Issue: When only `CLAUDE.md` exists, code returns `CLAUDE.md` as the boot path and writes the full sentinel there.
   - Why it matters: two-source drift risk the shared agent-config rules explicitly forbid; comment claims the template is “from block.md” while the path rule diverges.
   - Fix: Always prefer/create `AGENTS.md`; if `CLAUDE.md` exists alongside, append the one-line pointer only (`ensureClaudePointer` already covers the dual-file case).

3. **Hard standard — duplicated sentinel body vs documented SSOT**
   - File: `skillgrid-cli/cmd/skillgrid/init_boot.go` (`sentinelTemplate`, ~lines 23–69)
   - Standard: `block.md` opening line — “Single source of truth for the `## Skillgrid` block… must not drift.” Code-standards smell: **Duplicated Code**.
   - Issue: Full sentinel markdown is copy-pasted into a Go string constant. No parity test against `block.md`.
   - Why it matters: next edit to the lean AGENTS spine will update one copy and leave the other; every-prompt context then disagrees with the skill docs.
   - Fix: Load/parse `block.md` at build or runtime, or add a golden test that diffs the rendered template against the markdown fenced block.

4. **Hard standard — package-level mutable seam**
   - File: `skillgrid-cli/cmd/skillgrid/init_cmd.go` (`var initRunIndex = func…`, ~lines 90–98)
   - Standard: `.agents/skills/_shared/rules/code-standards.md` — “No global mutable state”; checklist “No shared mutable state without a single owner.”
   - Issue: Tests reassign `initRunIndex`. Production and test share a mutable package var.
   - Why it matters: parallel tests or future callers can race; the seam is not in the function signature (Explicit over implicit).
   - Fix: Pass an `indexFn` (or small interface) into `projectInit`; keep the default adapter local to `runProjectInit`.

5. **Hard standard — swallowed walk errors on `--docs` directories**
   - File: `skillgrid-cli/cmd/skillgrid/init_ingest.go` (`ingestExtra` WalkDir callback, ~lines 133–145)
   - Standard: code-standards Errors — “Never swallow.”
   - Issue: Default-root walk appends `walkErr` into `errs`. Extra-docs walk does `if walkErr != nil || d.IsDir() || !… { return nil }` and drops the error.
   - Why it matters: partial `--docs` failures look like success (`Errors` empty, `Ingested` under-count); inconsistent with the default-path path.
   - Fix: Mirror the default walk’s `errs = append(errs, …)` handling (and don’t overwrite a single `errMsg` for multi-file failures).

#### Minor (Nice to Have)

1. **Optional: Duplicated Code — directory walk shape**
   - File: `init_ingest.go` default-root walk vs `ingestExtra` directory branch
   - Smell: **Duplicated Code** (same WalkDir + `saveIngestFile` loop).
   - Fix: One `ingestTree(absDir, root)` helper used by both paths.

2. **Optional: Help test does not exercise the real flag seam**
   - File: `init_cmd_test.go` `TestInitHelpListsFlags`
   - Issue: Rebuilds Usage with `fs.String("docs",…)` instead of calling `newInitFlagSet` (which registers `stringSliceFlag` via `Var`).
   - Why it matters: help text can drift from the production flag set; repeatable `--docs` registration is untested at the flag layer.
   - Fix: Call `newInitFlagSet(fs)` then `fs.Usage()`.

3. **Consider: Session ownership**
   - File: `init_ingest.go` (`SessionStart` without `SessionEnd`)
   - Standard: “Ownership is explicit… who closes/frees it.”
   - Issue: Init opens a workspace session for ingest and never ends it. Same pattern exists in `mem skills add`, so not unique — but still leaves open sessions in the store.
   - Fix: `defer` `SessionEnd` (best-effort) after ingest completes.

4. **FYI: `countTopic` uses `Recent(50)`**
   - File: `init_cmd_test.go`
   - Fragile if a fixture ever exceeds 50 recent rows; not a production defect. Prefer a topic-keyed query helper if one exists later.

### Assessment

**Standards met?** With fixes

**Reasoning:** ADR-0012/0016/0019/0020 and glossary vocabulary for Project Init / Topic Key / no Knowledge Bases are solid. The load-bearing gaps are against the documented agent-config SSOT (`block.md` path + placeholder fill + duplicated template) plus one global mutable test seam and swallowed `--docs` walk errors. Those are local, required fixes; the walk/help/session items are optional polish.

### Axis summary

Standards: With fixes — ADRs/glossary clean; agent-config SSOT + mutable index seam + swallowed docs-walk errors need fixes.
