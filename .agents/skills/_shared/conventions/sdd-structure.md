# SDD Structure Convention (shared across all Skillgrid skills)

Single source of truth for filesystem layout, artifact names, and phase order. If a skill disagrees with this file, **this file wins**.

## Phase Order

```
brainstorming → [research] [spike] [sketch] → writing-blueprints → slicing → ticketing → execution → qa → review → ship → reflect
```

| Skill | Role |
|---|---|
| `using-skillgrid` | Orchestrator — detect, classify, route, resume, user gate |
| `onboarding` | Bootstrap — detect facts; write `config.yaml` + `state.yaml` + `artifacts/` zone + AGENTS block |
| `brainstorming` | Requirements gathering, ADR manifest |
| `interviewing` | Grilling the user to a shared understanding (drives ADRs) |
| `architectural-decision-records` | Domain model (terms in `artifacts/`) + ADR authoring / supersession |
| `acceptance-test-authoring` | BDD `acceptance.feature` scenarios from intent |
| `research` / `code-research` | External fact-finding → `findings.md` (Research section) |
| `spike` | Feasibility probe → `findings.md` (Spike section) |
| `sketch` | UI/interaction variants → `findings.md` (Sketch section) |
| `writing-blueprints` | Technical plan → `blueprint.md` |
| `slicing` | Vertical tickets → `tasks.md` |
| `ticketing` | Publish to tracker, track status |
| `subagent-execution` / `simple-execution` | Implement tickets |
| `parallel-execution` | Fan out independent tasks across subagents |
| `structured-debugging` | Root-cause-first debugging discipline |
| `test-driven-development` | RED → GREEN → TRIANGULATE → REFACTOR |
| `test-driven-verification` | Evidence before any "done" claim |
| `work-unit-commits` | Commit protocol + commit events for resume |
| `isolated-workspace` | Worktree isolation before execution |
| `ponytail` | Lazy-minimal solution discipline (auto, on coding tasks) |
| `qa` | Quality gate → `report.md` (QA half: test plan + verdict + evidence) |
| `requesting-code-review` / `parallel-code-review` | Review |
| `receiving-code-review` | Process findings |
| `ship` | Integrate to base + move change folder to `archive/` |
| `reflect` | **Terminal** — completes `report.md`'s retro half in place in the archive + session close |
| `resume` | Re-orient from durable state at session start / context rot |

## Naming

- **Change**: `YYYY-MM-DD-<topic>` (e.g. `2025-03-01-dark-mode`). Never reused.
- **Mnemonic slot**: `skillgrid/{YYYY-MM-DD-<topic>}/<artifact>` (`briefing|blueprint|findings|tasks|execution-progress|report`).

## Directory Structure

```
.skillgrid/
├── config.yaml                 # REQUIRED static SoT — stack, testing, tracker, rules.*
├── state.yaml                  # DYNAMIC project state — phase, current change, progress (updated by pipeline skills)
├── artifacts/                  # DURABLE project knowledge (committed, cross-change, topic-indexed)
│   ├── README.md               #   topic index (what lives here, what's where)
│   ├── 00-prd.md               #   product requirements (was docs/PRD.md)
│   ├── 00-architecture.md      #   system architecture (was docs/ARCHITECTURE.md; created lazily)
│   ├── 01-business-terms.md    #   domain/product/workflow vocabulary (was glossary/business.md)
│   ├── 02-technical-terms.md   #   architecture/platform/protocol vocabulary (was glossary/technical.md)
│   ├── 03-adr-index.md         #   ADR index — number, title, status, supersedes, one-line gist
│   ├── 04-adr-NNNN-slug.md     #   ADR records (global, immutable, supersedes trail) (was adr/NNNN-slug.md)
│   ├── 05-locked-constraints.md#   user-locked project-wide boundaries (the "bible")
│   └── 06-research-findings.md #   cross-change distilled research (carried forward from specs/)
├── specs/                      # ACTIVE change artifacts (committed); closed changes move to archive/
│   └── YYYY-MM-DD-<topic>/
│       ├── briefing.md         # brainstorming (falsifiable requirements, goal)
│       ├── acceptance.feature  # BDD scenarios (the traceability oracle)
│       ├── blueprint.md        # writing-blueprints (technical plan)
│       ├── tasks.md            # slicing (tickets, waves, dependencies)
│       ├── adr.md              # ADR Review Manifest (per-change; pointers to artifacts/04-adr-*)
│       ├── findings.md         # research/spike/sketch consolidated evidence (optional)
│       └── report.md           # two-phase: qa writes the QA half (test plan + gate verdict + evidence)
├── archive/                    # CLOSED changes (committed, immutable); created lazily by ship
│   └── YYYY-MM-DD-<topic>/     #   moved here by ship; reflect completes report.md's retro half in place
├── sdd/                        # scratch (gitignored)
│   ├── <plan-basename>/
│   │   ├── progress.md         # execution ledger
│   │   ├── briefs/             # implementer briefs
│   │   └── reviews/            # review packages
│   └── debug/<date>-<slug>/    # structured-debugging trail (state.md here is a debug artifact)
└── brainstorm/                 # visual companion state (gitignored)
```

## `state.yaml` and `artifacts/`

Two complementary files, both committed. They are the project-level state and knowledge — distinct from the per-change `specs/` artifacts.

**`state.yaml`** is the *dynamic* project state — where the project is right now. It is updated on every phase transition by the pipeline skill that owns that phase. Schema:

```yaml
schema: skillgrid/state/v1
pipeline:
  current_phase: "execution"          # brainstorming | blueprint | slicing | execution | qa | review | ship | reflect
  current_change: "2026-09-21-sessions-activity-unification"
  status: in_progress
progress:
  completed_changes: 18
  blocked_changes: 0
constraints_ref: .skillgrid/artifacts/05-locked-constraints.md   # pointer, not a copy
notes: "Serial development: one change at a time."
```

- `pipeline.current_change` + `current_phase` are the **resume pointer** — `resume` and `using-skillgrid` read them first, replacing the old "find newest `specs/` dir" heuristic.
- `constraints_ref` points at `05-locked-constraints.md`; it does not duplicate the constraints.
- Onboarding creates it. `ship` increments `completed_changes` and clears `current_change`. `reflect` writes the terminal phase.

**`artifacts/`** is the *durable* project knowledge — what you need to know to work on this project, organized by topic and surviving across changes. It replaces the former `docs/PRD.md`, `docs/ARCHITECTURE.md`, `.skillgrid/glossary/`, and `.skillgrid/adr/`. It is the single SoT for PRD, architecture, vocabulary, and ADRs. The per-change `findings.md` stays in `specs/` for traceability; `reflect` lifts the durable, still-relevant findings into `artifacts/06-research-findings.md` so they survive the archive.

## Artifact File Paths

| Skill | Creates / updates | Path |
|---|---|---|
| onboarding | skeleton | `config.yaml`, `state.yaml`, `artifacts/` zone (README + 05-locked-constraints), AGENTS block, `.gitignore` |
| brainstorming | briefing + scenarios + state | `specs/<topic>/briefing.md`, `specs/<topic>/acceptance.feature`, `specs/<topic>/adr.md`, `state.yaml` (phase + current_change) |
| architectural-decision-records | terms + ADRs | `artifacts/01-business-terms.md`, `artifacts/02-technical-terms.md`, `artifacts/04-adr-NNNN-slug.md`, `artifacts/03-adr-index.md` |
| research | findings | `specs/<topic>/findings.md` (`## Research:` section) + durable findings to `artifacts/06-research-findings.md` |
| spike | findings | `specs/<topic>/findings.md` (`## Spike:` section) |
| sketch | findings | `specs/<topic>/findings.md` (`## Sketch:` section) |
| writing-blueprints | plan + state | `specs/<topic>/blueprint.md`, `state.yaml` (phase) |
| slicing | tickets + state | `specs/<topic>/tasks.md`, `state.yaml` (phase) |
| ticketing | tracker IDs | `specs/<topic>/tasks.md` (Tracker ID fields) |
| execution | progress + state | `sdd/<plan>/progress.md` + `tasks.md` `[x]` marks + session `commit` events, `state.yaml` (phase) |
| qa | report (QA half) + state | `specs/<topic>/report.md` (test plan + verdict + evidence; retro half left empty), `state.yaml` (phase) |
| review | findings + state | `specs/<topic>/` (review artifacts), `state.yaml` (phase) |
| ship | move + state | **moves** `specs/<topic>/` → `archive/YYYY-MM-DD-<topic>/`; `state.yaml` (completed_changes +1, current_change cleared) |
| reflect | report (retro half) + state + artifacts | completes `archive/YYYY-MM-DD-<topic>/report.md` in place (from `## Final-State Facts` onward); lifts durable findings to `artifacts/06-research-findings.md`; `state.yaml` (terminal phase) + Mnemonic session close |

## Writing Rules

- Create the change folder **before** writing the briefing.
- READ before UPDATE; never blind overwrite.
- Commit spec-zone changes (`.skillgrid/specs/`) before code changes — the `pre-commit` zone guard enforces it.
- ADRs under `artifacts/04-adr-NNNN-slug.md` — created lazily on first use; global, not per-change. The per-change `specs/<topic>/adr.md` is a manifest of *pointers* to these, never a copy.
- Terms live in `artifacts/01-business-terms.md` / `02-technical-terms.md` — a glossary and nothing else.
- `state.yaml` is updated on every phase transition (see the Artifact File Paths table); it is the dynamic counterparty to `artifacts/` (the durable knowledge).
- `artifacts/` files are committed (durable knowledge that survives across machines); `state.yaml` is committed (the change log of phase transitions lives in its git history).

## `state.yaml` drift guard

`scripts/state-drift-check.mjs` is a **read-only** verifier that compares
`.skillgrid/state.yaml` against the spec zone (`.skillgrid/specs/`) and
reports a named drift verdict. It does **not** own the write — pipeline
skills keep writing `state.yaml` directly at every phase transition; the
guard owns the verification.

**Run:** `node scripts/state-drift-check.mjs [project-root]` (defaults to
cwd). ADR: `04-adr-0008` (Node.js + `yaml` package).

**Exit codes:** 0 clean, 1 drift (prints field/stale/derived table), 2
usage/parse error.

### The three checks

| Check | Spec-zone source of truth | `state.yaml` field | Drift when |
|-------|--------------------------|-------------------|------------|
| 1 phase | deepest present artifact in `specs/<change>/` | `pipeline.current_phase` | field ≠ derived phase |
| 2 change | active change dir under `specs/` | `pipeline.current_change` | field names a dir that isn't the active one |
| 3 completion | `report.md` shippable verdict (PASS, WAIVED, or CONCERNS+override) on any change dir | `progress.completed_changes` | shippable change exists but counter is 0 |

### Phase derivation (Check 1)

The deepest present artifact in the change dir determines the phase,
matching the Artifact File Paths table above:

| Deepest artifact present | Derived phase |
|--------------------------|---------------|
| `briefing.md` (+ `acceptance.feature`) | `spec` |
| `blueprint.md` | `blueprint` |
| `tasks.md` (no execution signal) | `slicing` |
| `tasks.md` + `sdd/<plan>/progress.md` with `[x]` or `tasks.md` with `[x]` | `execution` |
| `report.md` (any content — qa writes the Test Plan first, then the verdict) | `qa` |

When `current_change` is empty (shipped/idle), no phase is derived — the
dirs under `specs/` are closed changes, not the active one.

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

## Archive

On completion, **`ship`** moves the change folder out of the active `specs/` zone into a top-level, immutable `archive/` zone:

```
.skillgrid/specs/YYYY-MM-DD-<topic>/  ──►  .skillgrid/archive/YYYY-MM-DD-<topic>/
```

- **`specs/` is active-only.** A folder present in `specs/` is in-flight; a folder in `archive/` is closed. The date prefix in the folder name is preserved as the archive key.
- **The move is mechanical** (shell `git mv` / `mv` only, never model Read→Write) with a **`diff -r` readback** against a pre-move snapshot — an empty diff is the only passing evidence. See `ship` (Step 7).
- **`reflect`** then runs read-only over the archived folder: it completes `report.md`'s retro half in place (from `## Final-State Facts` onward — integration evidence + retrospective + archive summary) and owns the Mnemonic session close.
- `archive/` is created lazily by `ship` if absent. Archived changes are **never deleted or modified** after the move (except the retro sections of `report.md`, completed in place by `reflect` after the move — the QA half is pre-move and included in the `diff -r` readback; the retro half is appended after).
