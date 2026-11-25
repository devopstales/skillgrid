# Proposal: Skill Registry + Workflow Prompt Projection

> **STATUS:** `proposed` (2026-11-25, three work items ported from the 2026-11-25
> gentle-ai study)
> Source of study: `../gentle-ai/` — `docs/skill-registry.md` (index, not summary;
> "delegators pass paths, not digested rules"), `internal/components/agentguidance/
> routing.go` + `inject.go` (the canonical routing block projected into every agent's
> prompt), `docs/trigger-rules.md` § Installation and refresh ("the managed
> projection is replaced without duplication")
> In-force ADR: ADR-0016 (registry index path — `.skillgrid/projectwiki/
> skill-registry.md`, accepted 2026-11-25)
> Scope: **WI-A** the skill-registry index + refresh generator + the exact-path
> handoff contract; **WI-B** the canonical workflow block projected into the host
> agent file with a managed marker and an idempotent re-sync; **WI-C** hook-based
> enforcement in which the hooks run **`skillgrid next`** and act on its verdict —
> the first emitter of the `skillgrid.phase-status/v1` envelope frozen by
> `2026-11-23-pipeline-envelope` (its shape was reserved exactly for this emitter).
> No `advance`/CAS/journal, no new schema zone in this pass.
> Status values per `_shared/rules/sdd-structure.md` § STATUS Banner.

## 1. What gentle-ai does (findings)

### 1.1 Skill registry

A project-local **disk index** of skill name, full description (incl. YAML folded
multiline), scope (`project`/`user`), and the **exact `SKILL.md` path** —
`.atl/skill-registry.md` there, `skill-registry.md` in this repo's pinned location
(ADR-0016). Contract points worth porting:

- **"An index, not a generated summary."** The registry never digests a skill's
  body; the full `SKILL.md` stays the source of truth. gentle-ai explicitly
  rejected compact per-skill rules ("could lose nuance and break custom skills");
  tokens are spent only when a subagent actually needs a skill.
- **Refresh is mechanical**: `gentle-ai skill-registry refresh` scans project
  skill roots first, global agent skill roots second; parses frontmatter;
  dedupes (project wins; first candidate wins within a scope); **byte-compares
  the rendered index and rewrites only on change**; `list` / `--json` inspect
  without writing. A schema-stamped SHA-256 cache is a *non-authoritative hint*,
  repaired on drift — the scan is always authoritative.
- **Exclusions are deliberate and silent**: internal plumbing (`_shared`) and
  reserved prefixes are dropped without warning.
- **Refresh is wired into session-start hooks** (Codex, Claude Code, OpenCode,
  Pi) — the bookkeeping does not depend on the model remembering it.
- **The loading contract is the point**: the orchestrator matches the task
  against full descriptions and passes subagents the exact paths under a
  "Read these exact files before reading, writing, reviewing, testing, or
  creating artifacts" header. The subagent reads the full files.

### 1.2 The workflow is written into the agent's own prompt

`gentle-ai install` / `gentle-ai sync` render the **canonical routing block**
(ODD protocol steps in mandatory order, task-size criteria with measured numeric
budgets, delegation triggers declared "mandatory, not advisory") from one source
(`routing.go`) into every supported agent's owned prompt file/config, delimited
by a managed marker. Three properties matter:

- **Idempotent replacement**: found marker → replace the managed region, never
  append a second copy; the refresh is byte-compared so an unchanged block is
  never rewritten. User bytes outside the markers are untouched.
- **Routing semantics only**: "The rendered block deliberately carries routing
  semantics only. It states no runtime observation, issues no lifecycle
  authority, and activates no remote mechanism, so an offline agent reading it
  can still route work correctly." Repo-specific facts live in a separate
  managed region (`# Standards`), not in the workflow block.
- **Single render source**: one Go function projects to all adapters; adapters
  deliver, they never restate a variant. "Each consumer restates a variant" is
  the named failure mode.

## 2. The gap here

| # | Gap | Evidence |
|---|---|---|
| G1 | No index of the skill family. 22 `SKILL.md` files, but discovery is harness-native description matching; there is no machine-readable list, no `--json` view, and no guaranteed path for a skill to hand to a subagent | `ls .agents/skills/*/SKILL.md` (22); no `skill-registry*` file anywhere in the repo |
| G2 | Subagent handoffs in `execute` reference skills by **name** (`skillgrid:execute`); resolution is left to the harness, and a fresh subagent with a bounded context packet may never load the full skill + `references/` the parent meant it to follow | `docs/02-pipeline.md` § Multi-Agent Execution ("bounded context packet"); no path contract in `execute/SKILL.md` |
| G3 | Naive scanning picks up non-invokable skills: `_shared/SKILL.md` carries `disable-model-invocation: true` / `user-invocable: false`, and `references/` + `templates/` dirs nest SKILL-shaped docs — exclusion semantics are undefined | `.agents/skills/_shared/SKILL.md` frontmatter |
| G4 | The installed `<!-- skillgrid:start -->` block (onboarding step 4) points at rules and lists phase **names**, but the canonical workflow *semantics* — strict phase order, phase-derivation (deepest artifact wins), the user-approval gate ("the agent never writes `approved`"), the evidence rule, handoff-by-path — live only in skill markdown that an agent loads *after* it has already picked a skill | onboarding-write.md §4; `_shared/rules/sdd-structure.md` § Phase Order / Status Contract |
| G5 | No re-sync path: when the library updates (host repos carry a copied `.agents/`), the managed block is only rewritten by a full re-onboarding; `skillgrid init` re-renders the `# Standards` preamble from `config.yaml` but not the workflow block, and neither is byte-compared as a standalone operation | onboarding-write.md §4, §10; INSTALL_TEMPLATE.md ("copy the .agents/ folder") |

## 3. WI-A — Skill registry (ADR-0016)

### 3.1 Refresh generator

**New:** `scripts/skill-registry.mjs` (+ `scripts/test-skill-registry.mjs`),
same discipline as the existing scripts (ESM, no deps, exit-code contract
0/1/2, project root as first arg, default `.`; `bun test` layer). A `skillgrid
registry` CLI subcommand may wrap it later — the script is the contract, so the
library works without the CLI on PATH (same fallback rule as `skillgrid init`).

Behavior:

- **Scan order**: `.agents/skills/` (project scope — host repos carry the copied
  folder) first, then `~/.agents/skills/` (user scope) only with
  `--include-user`. Direct skills, then one category level per root — same
  depth rule as gentle-ai.
- **Parse** frontmatter: `name` (directory-name fallback), full `description`
  (folded multiline included), `version`, and the invocation flags.
- **Exclude (deliberate, silent)**: `_shared/` (internal plumbing,
  `disable-model-invocation`), `skill-creator`'s eval/fixture skills, anything
  under `references/`, `templates/`, `scripts/`, `evals/`.
- **Render** `.skillgrid/projectwiki/skill-registry.md` (path pinned by
  ADR-0016): a markdown table `Skill | Trigger / description | Scope | Path`,
  path = repo-relative exact `SKILL.md`, deterministic order (scope, then name),
  header stating the loading contract ("Read the exact file before work —
  delegators pass paths, not digested rules").
- **Idempotent**: byte-compare rendered bytes; unchanged index is never
  rewritten (timestamp stability for git + tooling). `--force` rewrites
  anyway; `--list` prints without writing; `--json` for the machine view
  (includes descriptions).
- **Refusals (exit 2)**: skill dir with unparseable frontmatter *and* no
  directory-name fallback context (ambiguous name); a `name:` colliding across
  scopes without the project-wins rule resolving it (it always will — listed so
  the rule is testable).

**Self-hosting proof:** the first commit of this change is the registry
generated on this repo's own 22 skills (minus exclusions) — the file exists, is
correct by inspection, and a second run leaves it byte-identical.

### 3.2 Handoff contract (skill prose)

**Edit:** `execute/SKILL.md` (subagent-driven + parallel fan-out modes) and
`slicing/SKILL.md` (task briefs): every delegated handoff carries a
`## Skills to load before work` section listing the **exact** `SKILL.md` +
`references/...` paths the worker must read before reading, writing, reviewing,
testing, or creating artifacts — paths, not digested rules. Name-only
references (`skillgrid:execute`) remain valid for *model-invoked* routing but
are not a valid subagent handoff.

**Edit:** `skillgrid/SKILL.md` Smart Entry: the registry is the name→path
resolution aid (the disk index is a cache; a missing/stale index is
regenerated by the script, never guessed from memory).

### 3.3 Refresh at the boundary (via WI-C)

No separate hook, no session-start poll: the WI-C **boundary trigger** runs
`skillgrid next` when a phase boundary is reached, and its envelope `context`
carries a `registry: fresh | stale | missing` advisory with the runnable
refresh step — one command, one trigger. The script stays the manual fallback.

## 4. WI-B — The workflow is written into the agent's own prompt

### 4.1 Canonical workflow section in the managed block

**New:** `.agents/skills/skillgrid/references/patterns/agent-workflow.md` — the
single render source for the workflow semantics, consumed by the existing
`agent-config.md` block render (onboarding step 4) and by `skillgrid sync` (one
code path; nothing restates a variant). Content — **routing semantics only**
(no runtime observation, no repo facts, no lifecycle authority; an offline
agent reading it can still route work):

- The strict phase chain (one line, from `sdd-structure.md` § Phase Order) and
  the no-skip/no-revisit rule.
- **Phase derivation**: deepest artifact present determines the phase;
  artifacts win over `state.yaml`; the strict-sequence check (a skill halts
  when the deepest artifact does not match its predecessor's output).
- **The user-approval gate** in its operative form: banner hand-edit *is* the
  approval; the agent never writes `approved`; a consuming skill re-reads line 3
  from disk and quotes the banner.
- **The evidence rule**: a phase is not complete without its evidence
  ("Seems right" is not a checkbox); `blocked(<phase>): <reason>` is the
  machine-readable halt signal.
- **The delegation rule**: delegate work, not responsibility; handoffs carry
  exact skill paths (cross-references WI-A 3.2).
- **The resume rule**: re-orient from durable state (Smart Entry) at session
  start / context rot; `state.yaml` holds no freeform journal.
- Fast-track pointer (waivers skip planning phases only, never the tail).

Explicitly **out of the block**: repo-specific facts (the `# Standards`
preamble's job), the tracker line (config-driven, already rendered), the
`### Rules` section (owned by AGENTS.md itself — the block points at it).

### 4.2 Projection + re-sync

- The onboarding step-4 render of the `<!-- skillgrid:start -->…<!--
  skillgrid:end -->` region gains the workflow section from 4.1 (replacing the
  current phase-name list). Marker contract unchanged: found → replace
  inclusive, never a second copy; not found → append; multi-platform rule
  unchanged (full block in `AGENTS.md`, one-line pointer in `CLAUDE.md`).
- **New: `skillgrid sync`** (CLI subcommand; script fallback
  `scripts/workflow-projection.mjs`) — re-renders **only** the managed
  projection (preamble + workflow block) from `config.yaml` + the library's
  canonical sources. No detection, no config rewrite, no tracker bootstrap:
  that stays onboarding. Byte-compared: an unchanged render touches nothing.
- **The update loop becomes**: copy new `.agents/` into the host repo → run
  `skillgrid sync` → the managed block reflects the new library, user bytes
  outside the markers untouched. That is gentle-ai's
  "the managed projection is replaced without duplication", in skillgrid's
  two-region shape.
- **Render provenance:** `skillgrid sync` records the sha256 of its render
  sources (`agents:` config digest + `agent-config.md` + `agent-workflow.md`)
  in a marker-adjacent HTML comment. `skillgrid next` (WI-C) compares it and
  reports `workflow_block: current | drifted | missing` as an advisory in its
  `context` — the drift check has no hook of its own.

## 5. WI-C — Hook-based enforcement: the hooks run `skillgrid next`

The hooks get **no CLI surface of their own** — they run `skillgrid next` and
act on the verdict it returns. The trigger is a **phase boundary** ("the next
stage is needed"), not session start: gentle-ai's model is "enter the
lifecycle once per candidate, when the work becomes reviewable" — here, when a
phase's evidence is done and the next stage's artifact or decision is due.

### 5.1 `skillgrid next` (read-only; first `skillgrid.phase-status/v1` emitter)

**New:** `_shared/scripts/phase-status.mjs` (the name the envelope proposal
reserved: "the authority script (later change) emits it, so the shape is
already settled when it lands") + a `skillgrid next` CLI subcommand wrapping
it (script fallback when the CLI is not on PATH — house rule).
**Dependency:** the envelope change's WU1/WU2 (frozen
`phase-status.schema.json` + `validate-envelope.mjs`); until they land, tests
assert against the two frozen fixture shapes (`phase-status-execute`,
`phase-status-gate-pending`).

Read-only: it never mutates artifacts or gates. Its single write is the
resume-rule patch — when `state.yaml`'s phase cache disagrees with the deepest
artifact, artifacts win, the CLI patches the cache and appends the dated
`projectwiki/log.md` entry (the router's documented behavior, moved into code).

- **Derives** per `sdd-structure.md` § Phase derivation: deepest artifact
  present → phase; STATUS banners read from line 3 of disk (the
  agent-never-writes-`approved` rule is a read, not a trust); `gate` and
  `blocked(...)` entries from `state.yaml`.
- **Emits** the frozen `skillgrid.phase-status/v1` envelope,
  `next_transition.kind` ∈ `execute | collect | stop`:
  - `execute` — the one valid next step, runnable and in-band ("run
    `skillgrid:execute` — wave 2, tickets T005–T006");
  - `collect` — a user gate is pending: the exact banner line to flip (the
    `phase-status-gate-pending` fixture case);
  - `stop` — blocked or complete: `continuation.step` runnable per the
    in-band rule (a `stop` carrying a `<placeholder>` template is a
    validation failure).
  - `context` (free-form per the envelope contract) carries the advisory
    bookkeeping, each with its runnable fix step: `registry:
    fresh|stale|missing`, `workflow_block: current|drifted|missing` — one
    command covers enforcement *and* freshness; there is no second hook
    command.
- **Exit codes** (house contract): 0 `execute`, 1 `collect`/`stop` (details on
  stdout), 2 usage/parse. Hooks read the envelope, not the exit code.
- **Test:** every emitted envelope validated through `validate-envelope.mjs`
  against the frozen schema (negative halves first: a `stop` with a template
  placeholder is rejected); one test per row of the phase-derivation table,
  including the state-vs-artifact drift row.

### 5.2 Hook wiring (`plugins/omp/skillgrid-guards`)

The plugin gains a `skillgrid next` client — spawn, **cached per turn** (one
call per boundary, not per tool call; CLI failure → undefined, the existing
fail-open convention) — and two boundary wirings:

1. **Boundary trigger — `turn_end`, conditional.** Fires only when the next
   stage is actually needed: the turn contained ≥1 `write`/`edit` against
   `.skillgrid/change-specs/` (the plugin already sees every `tool_call`, so
   the condition is tracked, not guessed). It runs `skillgrid next` and
   injects the verdict — the exact next step, or the pending gate with its
   banner line, plus the `context` advisories (stale registry / drifted block
   with their fix steps). **Once per boundary:** the verdict is deduped by
   envelope hash — a turn that advanced nothing is silent (gentle-ai: "remind
   only about candidates that session itself produced").
2. **Pre-tool phase enforcement — `tool_call` on `write`/`edit`** (spec-zone
   targets): the cached verdict rules before the static P0 trio — a write the
   CLI marks an artifact-order violation (e.g. `tasks.md` created while
   `acceptance.feature` is not `approved`) is blocked with the CLI's in-band
   continuation. The trio (file set, banner ownership, artifact order) is the
   **fail-open floor**: it still decides when the CLI is unavailable, but the
   transition table lives in one place — the script; the TS artifact-order
   logic is marked the fallback copy mirroring the derivation table.

**Enforcement semantics:** block only on a verdict that carries a runnable
continuation (the in-band rule — a denial without one is a CLI bug → fail
open); never block on `execute` (the verdict says what to do next, it does not
forbid current work); the hook never mutates — state transitions stay with the
phase-owning skills, and so does `advance` (§6).

**Router rewiring:** `skillgrid/SKILL.md` Smart Entry becomes: "route only
from the envelope `skillgrid next` returns; artifact derivation is the
fallback when the CLI is unavailable." The router relays a verdict instead of
re-deriving from memory — the prose table in `sdd-structure.md` stays the
human-readable spec the script implements.

## 6. Out of scope

- `skillgrid next` stays **read-only** in this pass: no `advance` verb, no
  CAS/journal, no gate mutation — the phase-owning skills keep writing
  `state.yaml` transitions (the envelope change's full proposal lists
  `status/advance` + CAS + journal; this pass lands the status half).
- Per-agent adapter rendering (gentle-ai renders 18 variants; skillgrid targets
  AGENTS.md/CLAUDE.md — one template, pointers not copies).
- Schemas/envelopes for the registry (it is a derived index, not a contract
  envelope); the `--include-user` global-scope scan beyond the single
  `~/.agents/skills/` root.

## 7. Phasing + acceptance sketch

| Phase | Content | Acceptance (→ `acceptance.feature` later) |
|---|---|---|
| A1 | `skill-registry.mjs` + test + first generated index on this repo | `happy path registry-refresh-idempotent` (second run byte-identical); `happy path registry-excludes-shared` (`_shared` absent from output, present in tree); `happy path registry-list-no-write` |
| A2 | Handoff-contract prose edits (execute, slicing, skillgrid router) | `happy path handoff-names-exact-paths` (a fan-out brief in `work/` carries the `## Skills to load before work` section with resolvable paths) |
| B1 | `agent-workflow.md` + render integration + `skillgrid sync` | `happy path sync-replaces-managed-region` (region replaced, no duplicate markers); `happy path sync-byte-identical-noop` (unchanged render → mtime/bytes untouched) |
| B2 | Re-sync proof on a fixture: a kubedash-shaped AGENTS.md with user edits outside the markers | `happy path sync-preserves-user-bytes` (diff outside the region is empty after sync) |
| C1 | `_shared/scripts/phase-status.mjs` (`skillgrid next`) + tests | `happy path next-routes-from-deepest-artifact` (derivation-table rows); `happy path next-stop-carries-runnable-continuation` (in-band rule); `happy path next-drift-patches-cache` (artifacts win, log entry written) |
| C2 | Guards plugin: `skillgrid next` client + boundary trigger + pre-tool enforcement | `happy path hook-quiet-when-no-advance` (turn without spec-zone writes → no call, no context); `happy path hook-injects-verdict` (boundary turn → context carries the exact next step / banner line, deduped on repeat); `happy path hook-blocks-order-violation` (blocked with the CLI's in-band continuation); `happy path hook-fail-open` (CLI missing → static trio decides, session unbricked) |

**Sequencing:** A1 → A2 → B1 → B2 → C1 → C2; each phase is a work-unit commit,
spec-zone artifacts land first (this proposal → user approval → `briefing.md`
at the brainstorm phase). C1 may run parallel to B against the frozen envelope
shapes; C2 needs A1 + B1 (its advisories reference both).

**Repo state this builds on:** ADR-0016 accepted; the `pipeline-envelope`
proposal (intake) owns the machine shapes — C1 consumes its WU1/WU2
(`phase-status.schema.json`, `validate-envelope.mjs`); the `selfimprove-p0…p4`
stack owns the improvement loop; nothing in this pass changes `state.yaml`
semantics (WI-C's resume-rule cache patch is its only write) or the tracker;
the guards plugin gains the WI-C wirings only.
