# SDD Structure Convention (shared across all SDD skills)

Single source of truth for filesystem layout, artifact names, and phase order. If a skill disagrees with this file, **this file wins**.

Plan: `docs/plan/01-workflow-new.md` (v4).

## Phase Order (v4)

```
onboard → propose → spec → apply ⇄ verify → archive
```

Optional before locking `change.md`:

```
[sdd-explore / research.md] → [design-spike] → propose
```

| Skill | Role |
|---|---|
| `use-skillgrid` | Orchestrator — detect, classify, route, resume, user gate |
| `sdd-onboard` | Bootstrap orchestrator (greenfield/brownfield) |
| `sdd-init` | Detect facts; write `config.yaml` + skeleton + AGENTS block |
| `sdd-explore` | **Helper** — optional change-scoped `research.md` (rots) |
| `sdd-propose` | Reserve NNN; write `change.md` (WHY+HOW) |
| `sdd-spec` | Own NN; write `tasks.md` (blocking DAG) + `acceptance.feature` |
| `sdd-apply` | Execute unblocked tasks; mark `[x]` + State |
| `sdd-verify` | Verdicts + trace + human QA plan; findings → apply |
| `sdd-archive` | Pure move `changes/` → `archive/` |

Onboard helpers (not top-level stages): `sdd-map-codebase`, `sdd-agent-context`, `sdd-constraints`, `sdd-domain`.

Retired: `sdd-design`, `sdd-tasks` (absorbed). Registry file is **optional** — not an init gate.

## Numbering

- **Change**: 3-digit `NNN` (e.g. `001-oauth-login`). Reserved by `sdd-propose`. Never reused.
- **Step**: 2-digit `NN` (e.g. `01-db-migration`). Allocated by `sdd-spec`. Never renumbered after creation.

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
├── codebase/                   # OPTIONAL brownfield map
├── changes/
│   └── <NNN-slug>/
│       ├── research.md         # optional; lifetime = this change
│       ├── change.md
│       ├── tasks.md            # State + steps + Depends + verify + QA plan section
│       ├── acceptance.feature
│       ├── qa-plan.md          # optional; or ## QA plan in tasks.md
│       └── interview.md        # optional questioning log
└── archive/
    └── <NNN-slug>/
```

No `steps/` tree. No companion `*-glossary-reference.md`. No required root `CONSTRAINTS.md` / `CONTEXT.md`. ADRs under `docs/adr/` only on **promote** (see plan).

## Artifact File Paths

| Skill | Creates / updates | Path |
|---|---|---|
| sdd-init / sdd-onboard | skeleton | `config.yaml`, `agents/` stubs, `glossary/` stubs, `changes/`, `archive/`, AGENTS block |
| sdd-map-codebase | optional map | `docs/skillgrid/codebase/` |
| sdd-agent-context | harness pointer | `AGENTS.md` (+ one-line pointers elsewhere) |
| sdd-constraints | quality bar | `config.yaml` `rules.*` |
| sdd-domain | vocabulary | `docs/skillgrid/glossary/{business,technical}.md` |
| sdd-explore | research | `changes/<NNN-slug>/research.md` |
| sdd-propose | change | `changes/<NNN-slug>/change.md` |
| sdd-spec | tasks + acceptance | `tasks.md`, `acceptance.feature` |
| sdd-apply | progress | `tasks.md` checkboxes + `## State` |
| sdd-verify | verdicts + QA | `tasks.md` `### Verification`; `qa-plan.md` or `## QA plan` |
| sdd-archive | move | `changes/<NNN-slug>/` → `archive/<NNN-slug>/` |

## Reading Artifacts

```
Research:   docs/skillgrid/changes/<NNN-slug>/research.md
Change:     docs/skillgrid/changes/<NNN-slug>/change.md
Tasks:      docs/skillgrid/changes/<NNN-slug>/tasks.md
Acceptance: docs/skillgrid/changes/<NNN-slug>/acceptance.feature
QA plan:    docs/skillgrid/changes/<NNN-slug>/qa-plan.md  (or ## QA plan in tasks.md)
Config:     docs/skillgrid/config.yaml
Glossary:   docs/skillgrid/glossary/{business,technical}.md
```

## Writing Rules

- Reserve NNN **before** creating the change folder.
- READ before UPDATE; never blind overwrite.
- Glossary: `docs/skillgrid/glossary/` — fold first-use into main artifacts; no companion reference files.
- Architecture decisions default in `change.md`; promote to `docs/adr/` only when the decision outlives the change (two-way links).
- `research.md` lifetime = this change; may rot; do not promote to `codebase/` or ADRs by default.
- Templates (mandatory):
  - `sdd-propose` → `templates/template-change.md`
  - `sdd-spec` → `templates/template-tasks.md` + `templates/template-acceptance.feature`

## `change.md` Shape

Canonical: `templates/template-change.md`. Header should list `Research:` and `Prototype:` when present.

## `tasks.md` Shape

Canonical: `templates/template-tasks.md`.

Must include **blocking/depends** edges (`Depends on:`) so apply can parallelize unblocked work.

```markdown
## State
## Step map
## NN-<name>
  Depends on: <NN or none>
  … Tasks … Verification … Commit
## QA plan          # or separate qa-plan.md from sdd-verify
## Archive gate checklist
```

## Acceptance Format

One change-level `acceptance.feature`; Features tagged `@step-NN`; ≥1 `@happy` + `@edge` + `@failure` per step. Canonical: `templates/template-acceptance.feature`.

## Archive

Pure move (not copy):

```
docs/skillgrid/changes/<NNN-slug>/  ──►  docs/skillgrid/archive/<NNN-slug>/
```

Gate: no unchecked tasks; every step `PASS` or `PASS WITH WARNINGS`; human QA accepted or waived.

## Legacy

Pre-v3 (`intent.md` + `plan.md` + `steps/`) and v3 trees remain valid history. New work uses **v4** only.
