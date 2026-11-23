# SDD Structure Convention (shared across all Skillgrid skills)

Single source of truth for filesystem layout, artifact names, and phase order. If a skill disagrees with this file, **this file wins**.

## Phase Order

This is the only full chain. Other skills cite this section in one line and restate only their own previous and next step.

**The strict sequence is the only order.** Skills follow one another in this
order. A skill MUST NOT run unless every preceding skill has completed.
There is no divergence: no skill runs in parallel with its predecessor or
successor, no skill is skipped (except the single `drift` exception below),
and no skill is revisited out of order. The three delegated skills
(`questioning`, `research`, `prototype`) run **inline** inside
`brainstorm`/`blueprint` at their canonical position — they are not
independent stops in the chain.

```
skillgrid → brainstorm → questioning → research → prototype → blueprint → slicing → execute → test → adversarial-authoring → drift → ship
```

| # | Skill | Position |
|---|---|---|
| 0 | `skillgrid` | router — classify, route, resume |
| 1 | `brainstorm` | classify path → design → spec (`briefing.md`) |
| 2 | `questioning` | interview loop (inline in brainstorm) — design tree, frontier rounds, clarity gate |
| 3 | `research` | fact-finding (inline in brainstorm/blueprint) → `findings.md` |
| 4 | `prototype` | feasibility probe (inline in blueprint, user-opt-in) → `findings.md` |
| 5 | `blueprint` | plan + acceptance spec → `blueprint.md` + `acceptance.feature` |
| 6 | `slicing` | vertical tickets → `tasks.md` + user gate |
| 7 | `execute` | implement tickets |
| 8 | `test` | verdict → `report.md` (QA half) |
| 9 | `adversarial-authoring` | council → `review.md` |
| 10 | `drift` | detect divergence (the single exception — may run on demand at any point after `execute`, not only in sequence) |
| 11 | `ship` | integrate + archive |

**Enforcement:** a skill that starts before its predecessor has completed is
a phase-order violation. The phase-derivation table below (deepest artifact
present) is the machine-readable check: if the deepest artifact does not match
the skill about to run, the skill MUST halt and report the divergence. The
`state.yaml` drift guard (`qa/scripts/state-drift-check.mjs`) is the read-only
verifier.

`skillgrid:blueprint` owns a five-phase flow: Phase 1 research (delegated to `skillgrid:research`), Phase 2 section-by-section draft with the Section Approval Gate, Phase 3 project-wiki persistence, Phase 4 prototype (delegated to `skillgrid:prototype`, user-selectable), Phase 5 write file. **Artifact creation order is permanent and sequential:** briefing → blueprint → blueprint validation → cucumber test file → tasks → slices. Never create the next artifact without the previous one existing. The user gate after `slicing` is mandatory: the user confirms the slice before execution. Fast-track waivers skip earlier planning skills only as `fast-track.md` allows; they do not skip the tail (`test` through `ship`).

| Skill | Role |
|---|---|
| `using-skillgrid` | Orchestrator — detect, classify, route, resume, user gate |
| `onboarding` | Bootstrap — detect facts; write `config.yaml` + `state.yaml` + the `.skillgrid/` knowledge bundle zone + AGENTS block |
| `brainstorming` | Requirements gathering via `skillgrid:questioning` (delegated), ADR manifest |
| `interviewing` | Grilling the user to a shared understanding (drives ADRs) |
| `blueprint` Phase 5 (acceptance spec) | BDD `acceptance.feature` — valid Cucumber syntax, traceability oracle (folded in from the deleted `acceptance-test-authoring` skill) |
| `blueprint` | Five-phase plan → `blueprint.md`: Phase 1 research (delegated to `skillgrid:research`) → Phase 2 section-by-section draft (Section Approval Gate) → Phase 3 project-wiki persistence → [Phase 4 prototype (delegated to `skillgrid:prototype`), user-selectable] → Phase 5 write file; **must** run goldfish review + user validation before completion |
| ADR records | Domain model (terms in `glossary/`) + ADR authoring / supersession — contract + templates in `_shared/adr-records/` (cited, not restated); driven by `brainstorm` during the interview; adopting an external method/pattern follows `_shared/rules/adoption-decisions.md` (method-not-plugin test, Not-adopted list, smoke test, rename sweep, park-don't-defer) |
| `projectwiki` | Read/maintain the `.skillgrid/projectwiki/` vault via the `projectwiki` engine (`mnemonic projectwiki ...`) — recall before re-deriving; concepts live in `decisions/`/`invariants/`/`workflows/`/`glossary/`/`gotchas/`/`architecture/` with `index/` projections rebuilt by `projectwiki index` in the same edit; ingests search/research + reflect results as typed, cross-linked pages (`DATA-LOADING.md`: `sources/` + `syntheses/` + `concepts/`). Top-level skill (cited, not restated) |
| `research` | External fact-finding → `findings.md` (Research section); standalone skill, also run inline as blueprint Phase 1 |
| `questioning` | Interview loop: design tree, frontier rounds, clarity gate; standalone skill, also run inline by brainstorm |
| `prototype` | Feasibility probe → `findings.md` (Prototype section); standalone skill, also run inline as blueprint Phase 4 |
| `sketch` | UI/interaction variants → `findings.md` (Sketch section) |
| `slicing` | Vertical tickets → `tasks.md`; **always** uses task creation skills |
| `ticketing` | Publish to tracker, track status |
| `subagent-execution` / `simple-execution` | *(absorbed into `execute` modes)* — implement tickets (subagent / inline mode) |
| `parallel-execution` | *(absorbed into `execute` fan-out mode)* — fan out independent tasks across subagents |
| `structured-debugging` | Root-cause-first debugging discipline |
| `test-driven-development` | RED → GREEN → TRIANGULATE → REFACTOR |
| `test-driven-verification` | Evidence before any "done" claim |
| `_shared/work-unit-commits` | Commit protocol + commit events for resume (contract + hooks, cited by execution skills) |
| `isolated-workspace` | *(absorbed into `execute` Step 0)* — worktree isolation before execution |
| `ponytail` | *(folded into `execute` as the lazy discipline)* — lazy-minimal solution ladder (fully active on every coding task during execution) |
| `qa` | Quality gate → `report.md` (QA half: test plan + verdict + evidence) |
| `code-review` | Review → `review.md` |
| `receiving-code-review` | Process findings |
| `ship` | Integrate to base + move change folder to `change-archive/` |
| `reflect` | **Terminal** — completes `report.md`'s retro half in place in the archive + session close |
| `environment-retro` | Sub-phase of `reflect` — sweeps the agent *environment* (checks, navigation, steering no-ops, tool economy) and appends `## Environment Retro` to `report.md`; proposes fixes, never edits code |
| `resume` | Re-orient from durable state at session start / context rot |

## Naming

- **Change**: `YYYY-MM-DD-<topic>` (e.g. `2025-03-01-dark-mode`). Never reused.
- **Mnemonic slot**: `skillgrid/{YYYY-MM-DD-<topic>}/<artifact>` (`briefing|blueprint|findings|tasks|execution-progress|report|review`).

## Directory Structure

```
.skillgrid/
├── config.yaml                 # REQUIRED static SoT — stack, testing, tracker, rules.*
├── state.yaml                  # DYNAMIC project state — phase, current change, genesis block (updated by pipeline skills)
├── deferred-work.md            # Cross-change deferred-work ledger (append-only DW-<n> entries; the review triage "defer" sink; per _shared/craft/deferred-work.md)
├── work/                       # LIVE execution scratch (gitignored) — one dir per in-flight change
│   └── YYYY-MM-DD-<topic>/
│       ├── progress.md         # execution ledger
│       ├── briefs/             # implementer briefs
│       ├── attempts/           # saved attempts + revisions (attempt-1.md, revision-rK.md)
│       └── packets/            # critic/auditor packets
├── prototype/                  # DURABLE feasibility probes (committed, permanently retained) — one dir per change
│   └── YYYY-MM-DD-<topic>/
├── change-specs/               # ACTIVE change artifacts (committed); closed changes move to change-archive/
│   └── YYYY-MM-DD-<topic>/
│       ├── briefing.md         # brainstorming (falsifiable requirements, goal)
│       ├── acceptance.feature  # BDD scenarios (the traceability oracle)
│       ├── blueprint.md        # blueprint (technical plan)
│       ├── tasks.md            # slicing (tickets, waves, dependencies)
│       ├── adr.md              # ADR Review Manifest (per-change; pointers to projectwiki/decisions/NNNN-slug.md)
│       ├── findings.md         # research/prototype/sketch consolidated evidence (optional)
│       ├── ledger.md           # consistency ledger (terms, invariants, interfaces, errors, config)
│       ├── evaluations/        # panel verdicts
│       ├── report.md           # two-phase: qa writes the QA half (test plan + gate verdict + evidence)
│       └── review.md           # code-review audit record; read by ship and reflect
├── change-archive/             # CLOSED changes (committed, immutable); created lazily by ship
│   └── YYYY-MM-DD-<topic>/     #   moved here by ship; reflect completes report.md's retro half in place
├── projectwiki/                # LLM-wiki (committed, cross-change) — the agent writes and maintains it
│   ├── ASSUMPTIONS.md          # DURABLE root record — VERIFIED/INFERRED/LOCKED/Open Questions; first file a reader opens
│   ├── ARCHITECTURE.md         # DURABLE root record (created lazily) — live repo/program structure
│   ├── index.md                # wiki catalog — every page, one-line summary, by category; first file the agent opens
│   ├── log.md                  # append-only timeline — ## [YYYY-MM-DD] <op> | <title>
│   ├── backlog/                # backlogmd tracker folder
│   ├── glossary/               #   01-business-terms.md + 02-technical-terms.md
│   ├── invariants/             #   locked constraints (05-locked-constraints.md)
│   ├── decisions/              #   one ADR per file (NNNN-slug.md); ASSUMPTIONS.md § LOCKED stores the path
│   ├── raw/                    #   immutable source docs (research, ADR context, prior-art) — never edited
│   │   └── assets/             #   images and attachments
│   ├── sources/                #   one summary page per raw doc, with citations
│   ├── entities/               #   things: services, APIs, users, data stores, external systems
│   ├── concepts/               #   ideas: invariants, patterns, error model, config precedence
│   └── syntheses/              #   cross-entity analyses, comparisons, overviews
├── cache/                      # machine-local cache (gitignored)
└── config.d/                   # machine-local config overrides (created lazily by `mnemonic init`; indexing.yaml)
```

## Resume markers

`resume` and `using-skillgrid` read these markers to pick the next skill. If another skill's table disagrees with this section, this section wins.

| Marker | Next |
|---|---|
| `report.md` absent, or `## Gate Decision` verdict empty / `PENDING` / still the template placeholder | `qa` |
| QA verdict filled (`PASS` / `CONCERNS` / `FAIL` / `WAIVED`) and `review.md` missing or its `## Verdict` floor not met | `code-review` |
| `review.md` floor met (`met` or `met-with-fixes`) and the folder is still under `change-specs/` | `receiving-code-review`, then `ship` |
| Folder in `change-archive/`, QA verdict filled, retro sections empty | `reflect` |
| Folder in `change-archive/` and retro sections filled (`## Acceptance Verdict` already chosen, or `## Lessons` / `## Decisions` non-empty) | none — cycle complete |

Planning-phase detection (which spec files exist, whether `tasks.md` is checked) stays in `skillgrid:resume`. The tail above is the part that used to collapse review and ship into one step.

## Status Contract

> The typed handoff between pipeline skills. Every skill that acts on a change MUST derive its state from this contract before acting. The contract reuses existing state (`state.yaml`, STATUS banners, artifact presence) — it adds no new files, no new binary, no new schema.

### Phase derivation (canonical)

The deepest artifact present in the change dir determines the phase. This is the single source of truth for routing — `state.yaml`'s `changes.<slug>.phase` is a cache, not the source.

| Deepest artifact present | Derived phase | Next skill |
|---|---|---|
| `briefing.md` (STATUS `draft` / `revised`) | `spec` | user review — the user reads and flips the STATUS banner to `approved` (record `blocked(spec)` meanwhile) |
| `briefing.md` (STATUS `approved`) | `spec` | `blueprint` |
| `blueprint.md` (no validation) | `blueprint` | `blueprint-validation` (critic subagent, then the user flips the STATUS banner to `approved`) |
| `blueprint.md` (validated, STATUS `proposed`) | `blueprint-validated` | user review — the user reads and flips the STATUS banner to `approved` |
| `blueprint.md` (validated, STATUS `approved` / `assumed` / `accepted`) | `blueprint-validated` | `blueprint` Phase 5 (acceptance spec) |
| `acceptance.feature` (STATUS `draft` / `revised`) | `spec-tests` | user review — the user reads and flips the STATUS banner to `approved` (record `blocked(spec-tests)` meanwhile) |
| `acceptance.feature` (STATUS `approved`) | `spec-tests` | `slicing` |
| `tasks.md` (no execution signal) | `slicing` | `execute` |
| `tasks.md` + execution signal (`[x]` marks or `work/<YYYY-MM-DD-topic>/progress.md`) | `execution` | `test` |
| `report.md` (QA half written) | `test` | `adversarial-authoring` |
| `review.md` (floor met) | `adversarial-authoring` | `drift` → `ship` |
| Folder in `change-archive/` | `complete` | none |

**Strict-sequence check:** a skill about to run MUST verify the deepest
artifact present matches its predecessor's output. If the deepest artifact
does not match, the skill halts and reports the divergence. E.g. `test` MUST
not start if `tasks.md` has no execution signal (i.e. `execute` has not
completed). `adversarial-authoring` MUST not start if `report.md` is absent
(i.e. `test` has not completed). `ship` MUST not start if `review.md` is
absent (i.e. `adversarial-authoring` has not completed).

### User approval gate

`briefing.md` and `blueprint.md` are read by the user, not only by agents. The
next phase that consumes one of them MUST find its STATUS banner at an
approved value:

- `blueprint` (any phase) requires `briefing.md` STATUS `approved`.
- `blueprint` Phase 6 (commit) requires `acceptance.feature` STATUS `approved`.
- `blueprint` Phase 5 (acceptance spec) and `slicing` require `blueprint.md`
  STATUS `approved`, `assumed`, or `accepted` — the last two are created only
  at a user decision point, so they count as read.

If the banner is not there, the skill HALTs, tells the user exactly which
line to flip, and records `blocked(<phase>): <file> awaiting user approval`
in `state.yaml` — `resume` reports it. The approval is the user's own
hand-edit of the banner (see § STATUS Banner), not a chat reply; the agent
never writes `approved` itself.

### Directional implication (validation semantics)

A **present artifact requires only the artifacts that canonically precede it** —
never a full-inventory check. The artifact order above is a sequence, not a
checklist:

- `briefing.md` + `blueprint.md` (no `acceptance.feature`, no `tasks.md`) is a
  normal, valid mid-phase state (phase `blueprint`) — it **passes** any
  phase-order check.
- `tasks.md` present without `acceptance.feature` is a **violation**: the
  cucumber file canonically precedes tasks.
- `tasks.md` (the deepest artifact, index 3) implies `blueprint.md` and
  `acceptance.feature` — never `briefing.md`, which is already present.

Every validator of spec-folder state (the `state.yaml` drift guard, a
`skillgrid validate`-style command, resume's phase derivation) MUST use this
semantics: the deepest present artifact is checked against the set of
artifacts before it, and only missing predecessors are findings. A
full-inventory rule flags every in-flight change and trains the agent to
ignore it.

### Resume rule

`resume` and `using-skillgrid` MUST derive phase from artifacts first, then cross-check `state.yaml`. If they disagree, **artifacts win** (the spec zone is the deeper truth — same principle as the state-drift-guard). The skill records the disagreement as a dated entry in `.skillgrid/projectwiki/index/log.md` and patches the phase to match. `state.yaml` holds no freeform journal — prose goes to the KB write log, not `state.yaml`.

### Edit scope guard

The `tasks.md` `Files` column defines the edit scope for the change. A subagent that needs to modify a file outside the named set MUST halt and checkpoint — it does not silently expand scope. The checkpoint message names the file and the ticket that would own it. This is the skillgrid equivalent of codex's `actionContext.allowedEditRoots`: the scope is declared upfront, not inferred at edit time.

For fast-track changes (no `tasks.md`), the edit scope is the set of files named in the briefing's `## Scope` section.

### Blocked reasons

If a phase cannot proceed (missing prerequisite, blocked dependency, unmet precondition), the skill MUST record the reason in `state.yaml` under the affected change entry using the format:

```
blocked(<phase>): <one-line reason>
```

Example: `blocked(execution): ticket T003 blocked by T001 (incomplete)`.

A `blocked(...)` entry is the machine-readable signal that `resume` reads to report the blocker to the human. An unexplained halt (no `blocked(...)` entry) is a bug in the skill, not a state.

### nextRecommended

The derived phase + blocked reasons together determine the next recommended action:

- No `blocked(...)` entry → the next skill in the phase derivation table above.
- `blocked(<phase>): <reason>` → `resolve-blockers` (the human or the orchestrator must address the named reason before the phase can proceed).
- Phase is `complete` → `none` (cycle complete).

This is a bounded routing token, not human prose. The human-readable explanation lives in `blockedReasons`; the routing decision lives in `nextRecommended`.

## STATUS Banner

`briefing.md`, `blueprint.md`, `acceptance.feature`, and `tasks.md` carry a machine-readable status on line 3 (a blockquote, after the `#` title). The `/plans` HTTP endpoint and the mockup's Changes panel parse it with the regex `^>\s*\*\*STATUS:\*\*\s*`?([a-z][a-z0-9-]+)`?` — the value must be lowercase, alphanumeric + hyphens, optionally backtick-wrapped.

**Format:** `> **STATUS:** \`<value>\` (YYYY-MM-DD)`

| File | Initial value | Updated by |
|---|---|---|
| `briefing.md` | `draft` | brainstorming (write), user (→ `approved`, hand-edit), ship (→ `shipped`) |
| `blueprint.md` | `proposed` | blueprint (write; `assumed` only via the Owed-Decision Gate), user (→ `approved`, hand-edit), Ratify (→ `accepted`), ship (→ `shipped`) |
| `acceptance.feature` | `draft` | blueprint Phase 5 (write), user (→ `approved`, hand-edit), ship (→ `shipped`) |
| `tasks.md` | `sliced` | slicing (write), execution (→ `in-progress` / `complete`), ship (→ `shipped`) |

**Values:** `proposed` / `draft` → `approved` → `sliced` → `in-progress` → `complete` → `shipped`. Side states: `assumed`, `accepted`, `stalled`, `blocked`, `revised`, `superseded`.

**Ownership:** the agent may write every value except `approved`. `approved` is a user-only value: after reading the document, the user hand-edits the banner to `approved (YYYY-MM-DD)` — that edit IS the approval record (the User approval gate above). An agent edit after approval reverts the banner to `revised`, and the user must re-read and re-flip. A consuming skill confirms approval by re-reading line 3 from disk and quoting the banner in its reply — an agent that quotes an `approved` line the user never typed has self-approved.

**Rule:** every spec dir's `briefing.md` MUST carry the banner. `blueprint.md` MUST carry it once blueprint writes it. `acceptance.feature` MUST carry it once blueprint Phase 5 writes it. `tasks.md` MUST carry it once slicing produces it. The `pre-commit` zone guard does not check it; the `/plans` endpoint returns `status: ""` when absent (the UI renders "unknown").

## `state.yaml` and the knowledge bundle

Two complementary files, both committed. They are the project-level state and knowledge — distinct from the per-change `change-specs/` artifacts.

**`state.yaml`** is the *dynamic* project state — where the project is right now. It is updated on every phase transition by the pipeline skill that owns that phase. Schema (v2, `changes:` map):

```yaml
schema: skillgrid/state/v2
changes:
  2026-09-21-sessions-activity-unification:
    phase: "execution"                # spec | blueprint | slicing | execution | qa | review | ship | reflect | complete
    status: in_progress               # in_progress | blocked | complete
    gate: ""                          # pending user gate (User gate contract); "" = none
    autonomy: check-in                # check-in | autonomous (the user's "go to the end" choice)
  2026-10-08-context-orchestrator:
    phase: "execution"
    status: in_progress
progress:
  completed_changes: 18
  blocked_changes: 0
constraints_ref: .skillgrid/projectwiki/ASSUMPTIONS.md                       # pointer, not a copy
```

- The `changes:` map is the **resume pointer** — `resume` and `using-skillgrid` read it first, replacing the old "find newest `change-specs/` dir" heuristic. Each entry's `slug` is the `change-specs/<slug>/` directory name; `phase` + `status` describe where that change is right now.
- **Parallel development:** multiple changes may coexist in the map simultaneously. There is no single "active" change — every entry is its own independent pipeline position.
- `constraints_ref` points at `.skillgrid/projectwiki/ASSUMPTIONS.md` (the durable project understanding); operational rules live in AGENTS.md's `### Rules` section, not in `state.yaml`.
- Onboarding creates it with an empty `changes:` map. `ship` removes the change's entry from the map and increments `completed_changes`. `reflect` sets the entry's `phase: complete`.
- **Legacy v1** (single `pipeline.current_change` + `current_phase`) is still parsed by the drift guard for backward compatibility, but new writes use v2.

### User gate contract

A user gate is a persisted decision point, not a conversational pause. It applies to spec-dir changes (entries in `changes:`); Bounded and fast-track work without a spec dir stays conversational.

- **Write on reach:** the skill that pauses for a user decision writes `changes.<slug>.gate: <gate>` in the same edit that updates `phase`. Named gates: `blueprint-validation` (`blueprint`, at the User Review Gate), `owed-decision` (`blueprint`, when the in-flight doubt gate finds an unsourced load-bearing claim — the panel: resolve first / not a real decision / build on the assumption; banner `assumed` when the user picks the third), `user-gate-slice` (`slicing`, when it writes `tasks.md`), `qa-verdict` (`qa`, when the verdict is `CONCERNS` or `FAIL` — a human decision is required), `ship-decision` (`ship`, when the override path needs a human).
- **Screen:** one screen, readable in thirty seconds — **Done** (what was saved, with file names), **Assumed** (new assumptions, one line each, at most five), **Next** (the next step and roughly how long it takes), plus the reply menu: "ok to continue · tell me what to change · go to the end."
- **Clear on respond:** the user's choice (ok, the named option, or a recorded override) clears the field in the same edit that advances the phase. A change request is written to the artifact under gate, the affected output is redone (at most twice; after that, record the disagreement and ask which version to keep), and the screen re-shows.
- **Re-show on resume:** `resume` finds a non-empty `gate` and re-shows the same screen before any other action. It never passes the gate itself. A status report at a gated change names the pending gate without advancing.
- **`autonomy`:** default `check-in`. "Go to the end" (or equivalent) sets `autonomy: autonomous` — from then on, gates become one-line records (the plan's ledger during execution, a dated note in the artifact under gate during planning) instead of stops, except hard stops: a `blocked(...)` entry, missing tools or permissions, or a failed gate still persist and wait. "Check in again" reverts to `check-in`.

**`projectwiki/ASSUMPTIONS.md`** is the *durable* project understanding — the four tiers `## VERIFIED` / `## INFERRED (HYPOTHESIS)` / `## LOCKED` / `## Open Questions`. `## LOCKED` is **the ADR index only**: one pointer row per in-force decision (a short description + the path to the record). `status`, `supersedes`, and `date` live in the file's frontmatter, not the row. The decision body is the file. Operational rules that are *not* decisions (build toolchain floor, commit format, dependency policy, spec/code zone order) live in AGENTS.md's `### Rules` section, not here.

**`projectwiki/ARCHITECTURE.md`** is the *live* repo/program structure record — created lazily as the repo takes shape.

The **project wiki vault** (`.skillgrid/projectwiki/`, OKF v0.2) holds the product + stack understanding (`ASSUMPTIONS.md` § VERIFIED), the terms glossary (`projectwiki/glossary/01/02-*-terms.md`), locked constraints (`projectwiki/invariants/`), and one ADR file per locked decision (`projectwiki/decisions/NNNN-slug.md`). Durable cross-change findings land as `projectwiki/decisions/` ADRs or `projectwiki/gotchas/` pages. The per-change `findings.md` stays in `change-specs/` for traceability; `reflect` lifts the durable, still-relevant findings into the vault so they survive the archive.

## Artifact File Paths

| Skill | Creates / updates | Path |
|---|---|---|
| onboarding | skeleton | `config.yaml`, `state.yaml`, `projectwiki/ASSUMPTIONS.md`, `projectwiki/ARCHITECTURE.md` (lazy), project wiki vault (via `projectwiki bootstrap`: `projectwiki/config.json` sentinel + `PROJECTWIKI_SCHEMA.md` + `.ignore` + page-type folders), AGENTS block, `.gitignore` |
| brainstorming | briefing + state (questioning via `skillgrid:questioning`) | `change-specs/<topic>/briefing.md`, `change-specs/<topic>/adr.md`, `projectwiki/ASSUMPTIONS.md` (new project or scope change → product + stack VERIFIED tier; VERIFIED/INFERRED tiers auto-written; LOCKED on user OK), `state.yaml` (`changes.<topic>` entry) |
| `_shared/adr-records` (via brainstorm) | terms + ADRs | `projectwiki/glossary/01-business-terms.md`, `projectwiki/glossary/02-technical-terms.md` (from the shared business/technical templates), `projectwiki/decisions/NNNN-slug.md` (body per config `adr_style` template in `_shared/adr-records/README.md`), `projectwiki/ASSUMPTIONS.md` (§ `LOCKED` ADR index pointer row only), `<specs_root>/<topic>/adr.md` (manifest from `brainstorm/references/adr-manifest.md`) |
| `projectwiki` | concepts + catalogs | `.skillgrid/projectwiki/<decisions\|invariants\|workflows\|glossary\|gotchas\|architecture>/*.md` + `projectwiki/index/` projections (same-edit rule: every page write is followed by `projectwiki index`); ADR numbering stays with `_shared/adr-records/README.md`; derived index is engine-owned (`projectwiki index` rebuilds) |
| `projectwiki` (`DATA-LOADING.md`) | ingested research + reflect results | `.skillgrid/projectwiki/sources/*.md` (one `SourceSummary` per external source, full `external:` citation), `syntheses/*.md` (one hub per research question), `concepts/*.md` (cut-out reusable ideas) + `preferences/<area>.md` from reflect; same-edit `projectwiki index` |
| research | findings | `change-specs/<topic>/findings.md` (`## Research:` section) + durable findings to the project wiki vault (`projectwiki/decisions/` ADR or `projectwiki/gotchas/` page) |
| prototype | findings + probe | `prototype/<YYYY-MM-DD-topic>/` (`prototype.md` + throwaway code, permanently retained) + `change-specs/<topic>/findings.md` (`## Prototype:` section) |
| sketch | findings | `change-specs/<topic>/findings.md` (`## Sketch:` section) |
| blueprint | plan + critic validation + state | `change-specs/<topic>/blueprint.md`, `state.yaml` (phase → `blueprint` then `blueprint-validated`) |
| blueprint Phase 5 (acceptance spec) | BDD scenarios (valid Cucumber syntax) + state | `change-specs/<topic>/acceptance.feature`, `state.yaml` (phase) |
| slicing | tickets + state (always via task creation skills) | `change-specs/<topic>/tasks.md`, `state.yaml` (phase) |
| ticketing | tracker IDs | `change-specs/<topic>/tasks.md` (Tracker ID fields) |
| execution | progress + state | `work/<YYYY-MM-DD-topic>/progress.md` + `tasks.md` `[x]` marks + session `commit` events, `state.yaml` (phase) |
| qa | report (QA half) + state | `change-specs/<topic>/report.md` (test plan + verdict + evidence; retro half left empty), `state.yaml` (phase) |
| review | findings + state | `change-specs/<topic>/` (review artifacts), `state.yaml` (phase) |
| ship | move + state | **moves** `change-specs/<topic>/` → `change-archive/YYYY-MM-DD-<topic>/`; `state.yaml` (remove `changes.<topic>` entry, completed_changes +1) |
| reflect | report (retro half) + state + artifacts | completes `change-archive/YYYY-MM-DD-<topic>/report.md` in place (from `## Final-State Facts` onward); lifts durable findings to the project wiki vault (`projectwiki/decisions/` ADR or `projectwiki/gotchas/` page); `state.yaml` (terminal phase) + Mnemonic session close |

## Writing Rules

- Create the change folder **before** writing the briefing.
- READ before UPDATE; never blind overwrite.
- Commit spec-zone changes (`.skillgrid/change-specs/`) before code changes — the `pre-commit` zone guard enforces it.
- ADRs live as `projectwiki/decisions/NNNN-slug.md` — one file per decision, created lazily, global, not per-change. `projectwiki/ASSUMPTIONS.md` § `LOCKED` (the ADR index) stores a short description + the path only; `status`/`supersedes`/`date` live in the file's frontmatter. The per-change `change-specs/<topic>/adr.md` is a manifest of pointers to those files, never a copy.
- Terms live in `projectwiki/glossary/01-business-terms.md` / `02-technical-terms.md` — a glossary and nothing else.
- `state.yaml` is updated on every phase transition (see the Artifact File Paths table); it is the dynamic counterparty to the knowledge bundle (the durable knowledge).
- Knowledge bundle files are committed (durable knowledge that survives across machines); `state.yaml` is committed (the change log of phase transitions lives in its git history).

## `state.yaml` drift guard

`.agents/skills/qa/scripts/state-drift-check.mjs` is a **read-only** verifier that compares
`.skillgrid/state.yaml` against the spec zone (`.skillgrid/change-specs/`) and
reports a named drift verdict. It does **not** own the write — pipeline
skills keep writing `state.yaml` directly at every phase transition; the
guard owns the verification.

**Run:** `node .agents/skills/qa/scripts/state-drift-check.mjs [project-root]` (defaults to
cwd). ADR: `.skillgrid/projectwiki/decisions/0008-state-drift-guard-js-yaml.md` (Node.js + `yaml` package).

**Exit codes:** 0 clean, 1 drift (prints field/stale/derived table), 2
usage/parse error.

### The three checks

| Check | Spec-zone source of truth | `state.yaml` field | Drift when |
|-------|--------------------------|-------------------|------------|
| 1 phase | deepest present artifact in `change-specs/<change>/` | `changes.<slug>.phase` | map entry's phase ≠ derived phase |
| 2 presence | every `change-specs/<slug>/` dir vs `changes:` map keys | `changes:` map | dir exists without a map entry, or map entry names a dir that doesn't exist (bidirectional) |
| 3 completion | `report.md` shippable verdict (PASS, WAIVED, or CONCERNS+override) on any change dir | `progress.completed_changes` | shippable change exists but counter is 0 |

### Phase derivation (Check 1)

The deepest present artifact in the change dir determines the phase,
matching the Artifact File Paths table above:

| Deepest artifact present | Derived phase |
|--------------------------|---------------|
| `briefing.md` (+ `acceptance.feature`) | `spec` |
| `blueprint.md` | `blueprint` |
| `tasks.md` (no execution signal) | `slicing` |
| `tasks.md` + `work/<YYYY-MM-DD-topic>/progress.md` with `[x]` or `tasks.md` with `[x]` | `execution` |
| `report.md` (any content — qa writes the Test Plan first, then the verdict) | `qa` |

For v2: every `change-specs/<slug>/` dir is compared against its `changes.<slug>` entry. A dir with no map entry is presence drift (Check 2); a map entry with no dir is also presence drift. Phase drift (Check 1) is per-entry.

### Wiring

- **`skillgrid:qa`** — Step 9.5 runs the guard. A drift hit renders as a
  **WARNING** in `report.md`'s `## State Drift` section (never CRITICAL,
  never affects the four-state verdict — same treatment as
  decision-debt). If drifted, qa patches `state.yaml` to match the derived
  values and commits the fix with the report.
- **`skillgrid:resume`** — unchanged (the guard is qa-scoped; resume keeps
  its existing "trust the spec zone" prose).

### Rule

Pipeline skills own the write; the guard owns the verification. A drift hit
means a skill forgot to update `state.yaml` at a phase transition — the spec
zone is the deeper truth, and the guard names the fix.

## Broken Windows ledger

`.skillgrid/WINDOWS.md` is a **project-level** cross-change defect register.
It tracks advisory findings (WARNING/SUGGESTION) that were not fixed before
archive. It is committed and survives the archive move.

**Format:** a markdown table with columns `Window | Change | Severity |
Finding | Waived By | Date | Status`. Window IDs are sequential (`W001`,
`W002`, …). Status is `open` until explicitly closed.

**Lifecycle:**
- **`qa`** (Step 10, after rendering the gate): auto-appends each open
  WARNING/SUGGESTION finding to WINDOWS.md. If 0 open findings, no change.
- **`ship`**: no change — windows are already recorded; the archive move
  does not touch WINDOWS.md (it is project-level, not per-change).
- **`reflect`**: no change.
- **Future:** a `qa` pre-check or `review-backlog` skill reads WINDOWS.md
  and surfaces open windows as context ("N open windows from prior changes").

**Rule:** qa auto-appends (mechanical copy of the report's open findings).
The human can edit WINDOWS.md to waive, fix, or close windows. The ledger is
advisory — it never blocks a gate.

## Session lock (`.skillgrid/.lock`)

An advisory lockfile that surfaces concurrent-session conflicts. It does not
block — it warns.

**File:** `.skillgrid/.lock` (JSON): `{ "change": "...", "session_id": "...",
"acquired_at": "..." }`.

**CLI:** `node scripts/state-lock.mjs <acquire|release|check> [args]`.

**Lifecycle:**
- **`brainstorming`** (acquire): after adding the `changes.<topic>` entry, runs
  `acquire <topic> [session-id]`. Exit 1 (active lock for a different
  change) → surface the conflict warning (advisory). Exit 0 → continue.
- **`ship`** (release): after removing the `changes.<topic>` entry, runs
  `release <topic>`. Always exit 0.
- **Stale threshold:** 2 hours. A stale lock is overwritten with a warning.

**Rule:** advisory only. With parallel development, the lock warns about
concurrent sessions touching the *same* change; different changes can
proceed independently.

## Byte budget (SKILL.md size ratchet)

Prevents unbounded growth of skill files. Each skill has a per-tier ceiling;
the budget JSON records the last-known size.

**Budget file:** `.agents/skills/_shared/skill-size-budget.json` — `{ ceiling, tiers, skills }`.

**CLI:** `node scripts/skill-size-budget.mjs check [--write] [project-root]` —
shipped in the skill source repo at
`.agents/skills/skill-creator/scripts/skill-size-budget.mjs`; consumer projects
run it from their installed skill tree (the same path shape this rule's other
scripts use — the `qa` skill's QA path runs the check before rendering the gate).

**Tiers:** `standard` 22KB (default), `large` 38KB, `xl` 40KB. Skills
explicitly assigned a tier in the `tiers` map; unlisted skills use `standard`.

**Measured scope:** a skill's size is `SKILL.md` + `references/` + `scripts/`
+ `evals/`. Bulky non-prose material that would push a skill over its ceiling
lives in a **peer folder** (sibling of `references/`, e.g. `style-guides/`) so
it stays out of the measured scope — the `SKILL.md` spine is the routing table,
the peer folder is the load-on-demand reference.

**Evals are part of the skill.** `evals/evals.json` sits in the measured scope
and is a first-class artifact: per the `skill-creator` TDD gate, a pipeline
skill without evals is incomplete (the RED step has no failing test to watch).
A skill that grows past its ceiling to carry new behavior should push that
behavior into `references/` — or drop the eval — rather than raise the ceiling.

**Lifecycle:**
- **`qa`** (check): runs `check` before rendering the gate. Overages become
  WARNING findings in the report (advisory, never blocks).
- **`--write`**: regenerates the `skills` size map (keeps `ceiling` and
  `tiers`). Run after intentionally growing a skill.

**Rule:** advisory. The ratchet warns; a human raises the ceiling or trims
content. It never blocks a gate.

## Ship drift (structure drift check)

Surfaces files changed outside the anticipated set at QA time — before
integration, while the spec is still the contract. The anticipated set is
the union of `tasks.md` Files column and `blueprint.md` Global Constraints
paths.

**CLI:** `node .agents/skills/qa/scripts/ship-drift-check.mjs check <base-ref>
--anticipated <dir>... [--exclude <glob>...] [project-root]`.

**Default excludes:** `*.lock`, `*.sum`, `package-lock.json`, `yarn.lock`,
`pnpm-lock.yaml`.

**Lifecycle:**
- **`qa`** (Step 9.6): resolves `<base-ref>` from `tasks.md` → `Chain
  strategy:` and runs the check. Exit 1 (drift) → WARNING finding in the
  report's `## Structure Drift` section. Advisory — never blocks the gate.

**Rule:** advisory. Surfaces scope creep at the point where it's still
cheap to fix (before integration).

## Archive

On completion, **`ship`** moves the change folder out of the active `change-specs/` zone into the top-level, immutable `change-archive/` zone:

```
.skillgrid/change-specs/YYYY-MM-DD-<topic>/  ──►  .skillgrid/change-archive/YYYY-MM-DD-<topic>/
```

- **`change-specs/` is active-only.** A folder present in `change-specs/` is in-flight; a folder in `change-archive/` is closed. The date prefix in the folder name is preserved as the archive key.
- **`change-archive/` is the single closed sink for the spec zone** — the only place a closed change's committed artifacts (briefing, blueprint, tasks, report, review) live. It is distinct from `work/`, which is **live execution scratch** (the Execution Ledger, ADR-0020): `work/<YYYY-MM-DD-topic>/` holds the in-flight `progress.md` / briefs / attempts / packets, is gitignored, and is ephemeral — `git clean -fdx` is the assumed disposal. A closed change's ledger is **not moved into `change-archive/`**; it simply stops being live. The two are separate sinks for separate lifetimes — never mix them, and never leave a closed change's ledger in `work/`.
- **The move is mechanical** (shell `git mv` / `mv` only, never model Read→Write) with a **`diff -r` readback** against a pre-move snapshot — an empty diff is the only passing evidence. See `ship` (Step 7).
- **`reflect`** then runs read-only over the archived folder: it completes `report.md`'s retro half in place (from `## Final-State Facts` onward — integration evidence + retrospective + archive summary) and owns the Mnemonic session close.
- `change-archive/` is created lazily by `ship` if absent. Archived changes are **never deleted or modified** after the move (except the retro sections of `report.md`, completed in place by `reflect` after the move — the QA half is pre-move and included in the `diff -r` readback; the retro half is appended after).
