# SDD Structure Convention (shared across all SDD skills)

Single source of truth for filesystem layout, artifact names, and phase order. If a skill disagrees with this file, **this file wins**.

Canonical workflow (implemented skills are authoritative):

```
onboard → [explore] → propose → design → spec → tasks → apply ⇄ verify → [review] → archive
```

Fast-track branch (see `fast-track.md`): `trivial`/`small` changes may skip design/spec with a recorded waiver.

## Phase Order

Optional before locking the proposal:

```
[sdd-explore / exploration.md] → [design-spike] → propose
```

| Skill | Role |
|---|---|
| `use-skillgrid` | Orchestrator — detect, classify, route, resume, user gate |
| `sdd-onboard` | Bootstrap orchestrator (greenfield/brownfield); detect facts; write `config.yaml` + skeleton + AGENTS block |
| `sdd-explore` | **Helper** — optional change-scoped `exploration.md` (rots) |
| `sdd-propose` | Write `proposal.md` (intent, scope, Capabilities contract, hypothesis, rollback) |
| `sdd-design` | Write `design.md` (HOW: architecture, file changes, threat matrix) |
| `sdd-spec` | Write delta/full specs under `specs/{domain}/spec.md` (WHAT: RFC 2119 + Given/When/Then) |
| `sdd-tasks` | Write `tasks.md` (dependency-ordered phases + Review Workload Forecast) |
| `sdd-apply` | Execute assigned tasks; mark `[x]` in `tasks.md`; persist `apply-progress` |
| `sdd-verify` | Write `verify-report.md`; verdict PASS / PASS WITH WARNINGS / FAIL; findings → apply |
| `sdd-review` | Optional post-verify gate (runs on human approval) — verdict REVIEW-PASS / BACK-TO-APPLY / waived |
| `sdd-archive` | Sync delta specs to main specs; move `changes/` → date-prefixed `archive/` |

Onboard helpers (references under `sdd-onboard/references/`, not top-level stages): `map-codebase`, `agent-context`, `constraints`, `domain`.

Registry file is **optional** — not an init gate.

## Naming

- **Change**: kebab-case `{change-name}` (e.g. `add-dark-mode`). A `NNN-` numeric prefix (e.g. `014-…`) is allowed but not required. Never reused once archived.
- **Mnemonic slot**: `sdd/{change-name}/<artifact>` (`proposal|design|spec|tasks|apply-progress|verify-report|archive-report`).

## Directory Structure

```
docs/skillgrid/
├── config.yaml                 # REQUIRED SoT — stack, context, tracker, rules.*
├── agents/
│   ├── issue-tracker.md
│   ├── triage-labels.md
│   └── skill-registry.md       # OPTIONAL generated index
├── glossary/                   # sibling of agents/ (not nested)
│   ├── business.md
│   └── technical.md
├── specs/                      # main specs (source of truth; updated by sdd-archive)
│   └── {domain}/spec.md
├── changes/
│   └── {change-name}/
│       ├── proposal.md         # sdd-propose (REQUIRED)
│       ├── design.md           # sdd-design (waivable on fast-track)
│       ├── specs/              # sdd-spec delta/full specs (waivable on fast-track)
│       │   └── {domain}/spec.md
│       ├── tasks.md            # sdd-tasks; marked [x] by sdd-apply (REQUIRED)
│       ├── exploration.md      # sdd-explore, optional; lifetime = this change
│       └── verify-report.md    # sdd-verify (REQUIRED before archive)
└── archive/
    └── YYYY-MM-DD-{change-name}/
```

No `steps/` tree. No companion `*-glossary-reference.md`. No required root `CONSTRAINTS.md` / `CONTEXT.md`. ADRs under `docs/adr/` only on **promote**.

Scratch (git-ignored, never committed): `.skillgrid/sdd/{change-name}/` — `subagent-execution` briefs, implementer reports, review diffs, `progress.md` ledger. Resolved by `scripts/sdd-workspace <tasks.md>`; deleted at finish. A `handoff` brief is different: OS temp (`/tmp`), never under `.skillgrid/`.

## Artifact File Paths

| Skill | Creates / updates | Path |
|---|---|---|
| sdd-onboard | skeleton | `config.yaml`, `agents/` stubs, `glossary/` stubs, `changes/`, `archive/`, AGENTS block |
| sdd-onboard → references/map-codebase.md | optional map | `docs/skillgrid/codebase/` |
| sdd-onboard → references/agent-context.md | harness pointer | `AGENTS.md` (+ one-line pointers elsewhere) |
| sdd-onboard → references/constraints.md | quality bar | `config.yaml` `rules.*` |
| sdd-onboard → references/domain.md | vocabulary | `docs/skillgrid/glossary/{business,technical}.md` |
| sdd-explore | exploration | `changes/{change-name}/exploration.md` |
| sdd-propose | proposal | `changes/{change-name}/proposal.md` |
| sdd-design | design | `changes/{change-name}/design.md` |
| sdd-spec | specs | `changes/{change-name}/specs/{domain}/spec.md` |
| sdd-tasks | tasks | `changes/{change-name}/tasks.md` |
| sdd-apply | progress | `tasks.md` `[x]` marks + Mnemonic `apply-progress` |
| sdd-verify | report | `changes/{change-name}/verify-report.md` |
| sdd-archive | sync + move | delta specs → `specs/{domain}/spec.md`; `changes/{change-name}/` → `archive/YYYY-MM-DD-{change-name}/` |

## Reading Artifacts

```
Proposal:   docs/skillgrid/changes/{change-name}/proposal.md
Design:     docs/skillgrid/changes/{change-name}/design.md
Specs:      docs/skillgrid/changes/{change-name}/specs/{domain}/spec.md
Tasks:      docs/skillgrid/changes/{change-name}/tasks.md
Report:     docs/skillgrid/changes/{change-name}/verify-report.md
Exploration: docs/skillgrid/changes/{change-name}/exploration.md (optional)
Config:     docs/skillgrid/config.yaml
Glossary:   docs/skillgrid/glossary/{business,technical}.md
```

## Writing Rules

- Create the change folder **before** writing the proposal; never reuse an archived name.
- READ before UPDATE; never blind overwrite.
- Glossary: `docs/skillgrid/glossary/` — fold first-use into main artifacts; no companion reference files.
- Architecture decisions default in `proposal.md`/`design.md`; promote to `docs/adr/` only when the decision outlives the change (two-way links).
- `exploration.md` lifetime = this change; may rot; do not promote to `codebase/` or ADRs by default.
- Detail formats (shapes, envelopes, merge semantics) are canonical in each phase skill (`sdd-propose`, `sdd-design`, `sdd-spec`, `sdd-tasks`, `sdd-apply`, `sdd-verify`, `sdd-archive`) and in `_shared/references/` — this file defines paths and order only.
- Legacy templates under `_shared/templates/` (`template-change.md`, `template-tasks.md`, `template-acceptance.feature`) are **deprecated** — kept for history, not used by current phases.

## Artifact Shapes (summary)

- `proposal.md`: Intent, Scope (in/out), Capabilities contract (New/Modified), Approach (intent only), Hypothesis (right + wrong), MVP, Door check, Classification, Verification floor, Trust Boundaries, Risks, Rollback, Success Criteria. Budget: under 450 words.
- `design.md`: HOW — architecture decisions with rationale, data flow, file changes, interfaces, testing strategy, threat matrix (every row Applicable or N/A:reason).
- `specs/{domain}/spec.md`: WHAT — full spec for New capabilities, delta spec (ADDED/MODIFIED/REMOVED/RENAMED) for Modified ones; RFC 2119 + Given/When/Then; MODIFIED is replace-semantics (copy the entire requirement block, then edit).
- `tasks.md`: Review Workload Forecast with the four plain-text guard lines (`Decision needed before apply:`, `Chained PRs recommended:`, `Chain strategy:`, `400-line budget risk:` — matched literally downstream), then dependency-ordered phases with hierarchical numbering; RED-test tasks before their production tasks.
- `verify-report.md`: YAML envelope + compliance matrix + execution evidence; verdict derived from blockers (any CRITICAL, unchecked task, or non-zero test/build exit → FAIL).

## Acceptance Format

Scenarios live inside the spec files (Given/When/Then per requirement: happy path + at least one edge/failure). No separate change-level `acceptance.feature` in the current pipeline.

## Archive

Sync delta specs to main specs first, then mechanical move (shell only, never Read→Write) with `diff -r` readback:

```
docs/skillgrid/changes/{change-name}/  ──►  docs/skillgrid/archive/YYYY-MM-DD-{change-name}/
```

Gate: verify-report exists with PASS or PASS WITH WARNINGS (never FAIL/CRITICAL); no unchecked tasks (or approved, proved reconciliation recorded); review advisory recorded (waived/PASS/BACK-TO-APPLY notes).

## Legacy

- Pre-v3 (`intent.md` + `plan.md` + `steps/`) and v3 trees remain valid history.
- v4 `change.md` + `NNN-slug` + `acceptance.feature` model (with `sdd-design`/`sdd-tasks` absorbed) is **superseded** — the implemented `proposal/design/spec/tasks` pipeline above is canonical. Do not start new work in the v4 shape.
