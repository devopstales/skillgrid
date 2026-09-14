# SDD Structure Convention (shared across all Skillgrid skills)

Single source of truth for filesystem layout, artifact names, and phase order. If a skill disagrees with this file, **this file wins**.

## Phase Order

```
brainstorming → [research] [spike] [sketch] → writing-blueprints → slicing → ticketing → execution → qa → review → ship → reflect
```

| Skill | Role |
|---|---|
| `using-skillgrid` | Orchestrator — detect, classify, route, resume, user gate |
| `onboarding` | Bootstrap — detect facts; write `config.yaml` + AGENTS block |
| `brainstorming` | Requirements gathering, ADR manifest |
| `research` / `deep-research` | External fact-finding → `research.md` |
| `spike` | Feasibility probe → `findings.md` |
| `sketch` | UI/interaction variants → `findings.md` |
| `writing-blueprints` | Technical plan → `blueprint.md` |
| `slicing` | Vertical tickets → `tasks.md` |
| `ticketing` | Publish to tracker, track status |
| `subagent-execution` / `simple-execution` | Implement tickets |
| `qa` | Quality gate → `qa-report.md` |
| `requesting-code-review` / `parallel-code-review` | Review |
| `receiving-code-review` | Process findings |
| `ship` | Integrate to base + move change folder to `archive/` → `ship-report.md` |
| `reflect` | **Terminal** — retrospective + archive-report + session close → `retrospective.md` + `archive-report.md` |

## Naming

- **Change**: `YYYY-MM-DD-<topic>` (e.g. `2025-03-01-dark-mode`). Never reused.
- **Mnemonic slot**: `skillgrid/{YYYY-MM-DD-<topic>}/<artifact>` (`briefing|blueprint|tasks|execution-progress|review-report|ship-report|retrospective|archive-report`).

## Directory Structure

```
.skillgrid/
├── config.yaml                 # REQUIRED SoT — stack, testing, tracker, rules.*
├── specs/                      # ACTIVE change artifacts (committed); closed changes move to archive/
│   └── YYYY-MM-DD-<topic>/
│       ├── briefing.md         # brainstorming (falsifiable requirements, goal)
│       ├── acceptance.feature  # BDD scenarios (the traceability oracle)
│       ├── blueprint.md        # writing-blueprints (technical plan)
│       ├── tasks.md            # slicing (tickets, waves, dependencies)
│       ├── adr.md              # ADR Review Manifest (from brainstorming)
│       ├── research.md         # research / deep-research (optional)
│       ├── findings.md         # spike / sketch consolidated results (optional)
│       ├── test-plan.md        # qa (test plan with layer selection)
│       ├── qa-report.md        # qa (gate verdict + evidence)
│       └── ship-report.md      # ship (integration + move evidence)
├── archive/                    # CLOSED changes (committed, immutable); created lazily by ship
│   └── YYYY-MM-DD-<topic>/     #   moved here by ship; reflect appends retrospective.md + archive-report.md
├── sdd/                        # scratch (gitignored)
│   └── <plan-basename>/
│       ├── progress.md         # execution ledger
│       ├── briefs/             # implementer briefs
│       └── reviews/            # review packages
├── glossary/                   # domain vocabulary (business.md + technical.md)
├── adr/                        # architectural decision records
└── brainstorm/                 # visual companion state (gitignored)
```

## Artifact File Paths

| Skill | Creates / updates | Path |
|---|---|---|
| onboarding | skeleton | `config.yaml`, AGENTS block, `.gitignore` |
| brainstorming | briefing + scenarios | `specs/<topic>/briefing.md`, `specs/<topic>/acceptance.feature`, `specs/<topic>/adr.md` |
| research | findings | `specs/<topic>/research.md` |
| spike | findings | `specs/<topic>/findings.md` |
| sketch | findings | `specs/<topic>/findings.md` |
| writing-blueprints | plan | `specs/<topic>/blueprint.md` |
| slicing | tickets | `specs/<topic>/tasks.md` |
| ticketing | tracker IDs | `specs/<topic>/tasks.md` (Tracker ID fields) |
| execution | progress | `sdd/<plan>/progress.md` + `tasks.md` `[x]` marks |
| qa | report | `specs/<topic>/test-plan.md`, `specs/<topic>/qa-report.md` |
| review | findings | `specs/<topic>/` (review artifacts) |
| ship | report + move | `specs/<topic>/ship-report.md`, then **moves** `specs/<topic>/` → `archive/YYYY-MM-DD-<topic>/` |
| reflect | retrospective + archive-report | `archive/YYYY-MM-DD-<topic>/{retrospective,archive-report}.md` + Mnemonic session close |

## Writing Rules

- Create the change folder **before** writing the briefing.
- READ before UPDATE; never blind overwrite.
- Commit spec-zone changes (`.skillgrid/specs/`) before code changes — the `pre-commit` zone guard enforces it.
- ADRs under `.skillgrid/adr/` — created lazily on first use.

## Archive

On completion, **`ship`** moves the change folder out of the active `specs/` zone into a top-level, immutable `archive/` zone:

```
.skillgrid/specs/YYYY-MM-DD-<topic>/  ──►  .skillgrid/archive/YYYY-MM-DD-<topic>/
```

- **`specs/` is active-only.** A folder present in `specs/` is in-flight; a folder in `archive/` is closed. The date prefix in the folder name is preserved as the archive key.
- **The move is mechanical** (shell `git mv` / `mv` only, never model Read→Write) with a **`diff -r` readback** against a pre-move snapshot — an empty diff is the only passing evidence. See `ship` (Step 7).
- **`ship` writes `ship-report.md` before the move** (so it lands in the archive) and runs the move as its last step.
- **`reflect`** then runs read-only over the archived folder: it writes `retrospective.md` + `archive-report.md` into it and owns the Mnemonic session close.
- `archive/` is created lazily by `ship` if absent. Archived changes are **never deleted or modified** after the move (except the two additive reports `reflect` appends).
