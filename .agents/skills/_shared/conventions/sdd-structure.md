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
| `interviewing` | Grilling the user to a shared understanding (drives ADRs) |
| `architectural-decision-records` | Domain model (glossary) + ADR authoring / supersession |
| `acceptance-test-authoring` | BDD `acceptance.feature` scenarios from intent |
| `research` / `deep-research` | External fact-finding → `findings.md` (Research section) |
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
| `work-unit-commits` | Commit protocol + checkpoint resume handle |
| `isolated-workspace` | Worktree isolation before execution |
| `ponytail` | Lazy-minimal solution discipline (auto, on coding tasks) |
| `qa` | Quality gate → `qa-report.md` (test plan + verdict + evidence) |
| `requesting-code-review` / `parallel-code-review` | Review |
| `receiving-code-review` | Process findings |
| `ship` | Integrate to base + move change folder to `archive/` |
| `reflect` | **Terminal** — retrospective + final report + session close → `report.md` |
| `resume` | Re-orient from durable state at session start / context rot |

## Naming

- **Change**: `YYYY-MM-DD-<topic>` (e.g. `2025-03-01-dark-mode`). Never reused.
- **Mnemonic slot**: `skillgrid/{YYYY-MM-DD-<topic>}/<artifact>` (`briefing|blueprint|findings|tasks|execution-progress|qa-report|report`).

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
│       ├── findings.md         # research/spike/sketch consolidated evidence (optional)
│       └── qa-report.md        # qa (test plan + gate verdict + evidence)
├── archive/                    # CLOSED changes (committed, immutable); created lazily by ship
│   └── YYYY-MM-DD-<topic>/     #   moved here by ship; reflect appends report.md
├── sdd/                        # scratch (gitignored)
│   ├── checkpoint.json         # resume handle, derived from git log + [skillgrid-context]
│   ├── <plan-basename>/
│   │   ├── progress.md         # execution ledger
│   │   ├── briefs/             # implementer briefs
│   │   └── reviews/            # review packages
│   └── debug/<date>-<slug>/    # structured-debugging trail (state.md here is a debug artifact)
├── glossary/                   # domain vocabulary (business.md + technical.md)
├── adr/                        # architectural decision records
└── brainstorm/                 # visual companion state (gitignored)
```

## Artifact File Paths

| Skill | Creates / updates | Path |
|---|---|---|
| onboarding | skeleton | `config.yaml`, AGENTS block, `.gitignore` |
| brainstorming | briefing + scenarios | `specs/<topic>/briefing.md`, `specs/<topic>/acceptance.feature`, `specs/<topic>/adr.md` |
| research | findings | `specs/<topic>/findings.md` (`## Research:` section) |
| spike | findings | `specs/<topic>/findings.md` (`## Spike:` section) |
| sketch | findings | `specs/<topic>/findings.md` (`## Sketch:` section) |
| writing-blueprints | plan | `specs/<topic>/blueprint.md` |
| slicing | tickets | `specs/<topic>/tasks.md` |
| ticketing | tracker IDs | `specs/<topic>/tasks.md` (Tracker ID fields) |
| execution | progress | `sdd/<plan>/progress.md` + `tasks.md` `[x]` marks + `sdd/checkpoint.json` |
| qa | report | `specs/<topic>/qa-report.md` (test plan + verdict + evidence) |
| review | findings | `specs/<topic>/` (review artifacts) |
| ship | move | **moves** `specs/<topic>/` → `archive/YYYY-MM-DD-<topic>/` |
| reflect | final report | `archive/YYYY-MM-DD-<topic>/report.md` + Mnemonic session close |

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
- **`reflect`** then runs read-only over the archived folder: it writes `report.md` (integration evidence + retrospective + archive summary) into it and owns the Mnemonic session close.
- `archive/` is created lazily by `ship` if absent. Archived changes are **never deleted or modified** after the move (except the single additive `report.md` that `reflect` writes).
