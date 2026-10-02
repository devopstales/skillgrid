# SDD Structure Convention (shared across all Skillgrid skills)

Single source of truth for filesystem layout, artifact names, and phase order. If a skill disagrees with this file, **this file wins**.

## Phase Order

This is the only full chain. Other skills cite this section in one line and restate only their own previous and next step.

```
brainstorming → [research | spike | sketch] → writing-blueprints → slicing → (user gate) → ticketing → execution → qa → requesting-code-review → receiving-code-review → ship → reflect
```

`research`, `spike`, and `sketch` are optional pre-blueprint gates. The user gate after `slicing` is mandatory: the user confirms the slice before execution. Fast-track waivers skip earlier planning skills only as `fast-track.md` allows; they do not skip the tail (`qa` through `reflect`).

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
| `requesting-code-review` / `parallel-code-review` | Review → `review.md` |
| `receiving-code-review` | Process findings |
| `ship` | Integrate to base + move change folder to `archive/` |
| `reflect` | **Terminal** — completes `report.md`'s retro half in place in the archive + session close |
| `resume` | Re-orient from durable state at session start / context rot |

## Naming

- **Change**: `YYYY-MM-DD-<topic>` (e.g. `2025-03-01-dark-mode`). Never reused.
- **Mnemonic slot**: `skillgrid/{YYYY-MM-DD-<topic>}/<artifact>` (`briefing|blueprint|findings|tasks|execution-progress|report|review`).

## Directory Structure

```
.skillgrid/
├── config.yaml                 # REQUIRED static SoT — stack, testing, tracker, rules.*
├── state.yaml                  # DYNAMIC project state — phase, current change, progress (updated by pipeline skills)
├── ASSUMPTIONS.md              # DURABLE root record (committed) — project understanding + ALL architectural decisions (### ADR-NNNN entries) + locked constraints (the "bible")
├── ARCHITECTURE.md             # DURABLE root record (committed, created lazily) — live repo/program structure
├── spikes/                     # DURABLE feasibility probes (committed, permanently retained) — NNN-name/ dirs (spike.md + throwaway code)
│   └── NNN-name/
├── artifacts/                  # DURABLE project knowledge (committed, cross-change, topic-indexed)
│   ├── README.md               #   topic index (what lives here, what's where)
│   ├── 01-business-terms.md    #   domain/product/workflow vocabulary (was glossary/business.md)
│   ├── 02-technical-terms.md   #   architecture/platform/protocol vocabulary (was glossary/technical.md)
│   └── 06-research-findings.md #   cross-change distilled research (carried forward from specs/)
├── specs/                      # ACTIVE change artifacts (committed); closed changes move to archive/
│   └── YYYY-MM-DD-<topic>/
│       ├── briefing.md         # brainstorming (falsifiable requirements, goal)
│       ├── acceptance.feature  # BDD scenarios (the traceability oracle)
│       ├── blueprint.md        # writing-blueprints (technical plan)
│       ├── tasks.md            # slicing (tickets, waves, dependencies)
│       ├── adr.md              # ADR Review Manifest (per-change; pointers to ASSUMPTIONS.md ### ADR-NNNN entries)
│       ├── findings.md         # research/spike/sketch consolidated evidence (optional)
│       ├── report.md           # two-phase: qa writes the QA half (test plan + gate verdict + evidence)
│       └── review.md           # requesting-code-review audit record; read by ship and reflect
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

## Resume markers

`resume` and `using-skillgrid` read these markers to pick the next skill. If another skill's table disagrees with this section, this section wins.

| Marker | Next |
|---|---|
| `report.md` absent, or `## Gate Decision` verdict empty / `PENDING` / still the template placeholder | `qa` |
| QA verdict filled (`PASS` / `CONCERNS` / `FAIL` / `WAIVED`) and `review.md` missing or its `## Verdict` floor not met | `requesting-code-review` |
| `review.md` floor met (`met` or `met-with-fixes`) and the folder is still under `specs/` | `receiving-code-review`, then `ship` |
| Folder in `archive/`, QA verdict filled, retro sections empty | `reflect` |
| Folder in `archive/` and retro sections filled (`## Acceptance Verdict` already chosen, or `## Lessons` / `## Decisions` non-empty) | none — cycle complete |

Planning-phase detection (which spec files exist, whether `tasks.md` is checked) stays in `skillgrid:resume`. The tail above is the part that used to collapse review and ship into one step.

## STATUS Banner

`briefing.md` and `tasks.md` carry a machine-readable status on line 3 (a blockquote, after the `#` title). The `/plans` HTTP endpoint and the mockup's Changes panel parse it with the regex `^>\s*\*\*STATUS:\*\*\s*`?([a-z][a-z0-9-]+)`?` — the value must be lowercase, alphanumeric + hyphens, optionally backtick-wrapped.

**Format:** `> **STATUS:** \`<value>\` (YYYY-MM-DD)`

| File | Initial value | Updated by |
|---|---|---|
| `briefing.md` | `draft` | brainstorming (write), ship (→ `shipped`) |
| `tasks.md` | `sliced` | slicing (write), execution (→ `in-progress` / `complete`), ship (→ `shipped`) |

**Values:** `draft` → `sliced` → `in-progress` → `complete` → `shipped`. Side states: `stalled`, `blocked`, `revised`, `superseded`.

**Rule:** every spec dir's `briefing.md` MUST carry the banner. `tasks.md` MUST carry it once slicing produces it. The `pre-commit` zone guard does not check it; the `/plans` endpoint returns `status: ""` when absent (the UI renders "unknown").

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
constraints_ref: .skillgrid/ASSUMPTIONS.md                       # pointer, not a copy
notes: "Serial development: one change at a time."
```

- `pipeline.current_change` + `current_phase` are the **resume pointer** — `resume` and `using-skillgrid` read them first, replacing the old "find newest `specs/` dir" heuristic.
- `constraints_ref` points at `.skillgrid/ASSUMPTIONS.md`; it does not duplicate the constraints.
- Onboarding creates it. `ship` increments `completed_changes` and clears `current_change`. `reflect` writes the terminal phase.

**`ASSUMPTIONS.md`** (root) is the *durable* project understanding — the four tiers `## VERIFIED` / `## INFERRED (HYPOTHESIS)` / `## LOCKED` / `## Open Questions`, where `## LOCKED` holds the in-force ADR set (`### In-force set` table + `### ADR-NNNN` entries), the user-locked project-wide boundaries (`### Locked constraints`, which render into the AGENTS.md `### Rules` section), and the locked assumptions. It replaces the former `docs/PRD.md`, `artifacts/04-adr-*.md`, `artifacts/03-adr-index.md`, and `artifacts/05-locked-constraints.md` — decisions now live as `### ADR-NNNN` *entries*, never as separate files.

**`ARCHITECTURE.md`** (root) is the *live* repo/program structure record — created lazily as the repo takes shape.

**`artifacts/`** is the *durable* topic-indexed knowledge that is not decisions: the terms glossary (`01/02-*-terms.md`) and cross-change distilled research (`06-research-findings.md`). It is the single SoT for vocabulary and research; decisions and structure live at the root. The per-change `findings.md` stays in `specs/` for traceability; `reflect` lifts the durable, still-relevant findings into `artifacts/06-research-findings.md` so they survive the archive.

## Artifact File Paths

| Skill | Creates / updates | Path |
|---|---|---|
| onboarding | skeleton | `config.yaml`, `state.yaml`, `ASSUMPTIONS.md`, `ARCHITECTURE.md` (lazy), `artifacts/` zone (README + 06-research-findings), AGENTS block, `.gitignore` |
| brainstorming | briefing + scenarios + state | `specs/<topic>/briefing.md`, `specs/<topic>/acceptance.feature`, `specs/<topic>/adr.md`, `ASSUMPTIONS.md` (VERIFIED/INFERRED tiers auto-written; LOCKED on user OK), `state.yaml` (phase + current_change) |
| architectural-decision-records | terms + ADRs | `artifacts/01-business-terms.md`, `artifacts/02-technical-terms.md`, `ASSUMPTIONS.md` (`### ADR-NNNN` entries under § LOCKED + `### In-force set` table) |
| research | findings | `specs/<topic>/findings.md` (`## Research:` section) + durable findings to `artifacts/06-research-findings.md` |
| spike | findings + probe | `spikes/NNN-name/` (`spike.md` + throwaway code, permanently retained) + `specs/<topic>/findings.md` (`## Spike:` section) |
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
- ADRs live as `### ADR-NNNN` entries under `## LOCKED` in `ASSUMPTIONS.md` — created lazily on first use; global, not per-change. The per-change `specs/<topic>/adr.md` is a manifest of *pointers* to these entries, never a copy.
- Terms live in `artifacts/01-business-terms.md` / `02-technical-terms.md` — a glossary and nothing else.
- `state.yaml` is updated on every phase transition (see the Artifact File Paths table); it is the dynamic counterparty to `artifacts/` (the durable knowledge).
- `artifacts/` files are committed (durable knowledge that survives across machines); `state.yaml` is committed (the change log of phase transitions lives in its git history).

## `state.yaml` drift guard

`.agents/skills/verification/qa/scripts/state-drift-check.mjs` is a **read-only** verifier that compares
`.skillgrid/state.yaml` against the spec zone (`.skillgrid/specs/`) and
reports a named drift verdict. It does **not** own the write — pipeline
skills keep writing `state.yaml` directly at every phase transition; the
guard owns the verification.

**Run:** `node .agents/skills/verification/qa/scripts/state-drift-check.mjs [project-root]` (defaults to
cwd). ADR: `ASSUMPTIONS.md` § `### ADR-0008` (Node.js + `yaml` package).

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
- **`brainstorming`** (acquire): after setting `current_change`, runs
  `acquire <topic> [session-id]`. Exit 1 (active lock for a different
  change) → surface the conflict warning (advisory). Exit 0 → continue.
- **`ship`** (release): after clearing `current_change`, runs
  `release <topic>`. Always exit 0.
- **Stale threshold:** 2 hours. A stale lock is overwritten with a warning.

**Rule:** advisory only. The serial rule ("one change at a time") is a human
decision; the lock surfaces conflicts without enforcing them.

## Byte budget (SKILL.md size ratchet)

Prevents unbounded growth of skill files. Each skill has a per-tier ceiling;
the budget JSON records the last-known size.

**Budget file:** `.agents/skills/_shared/skill-size-budget.json` — `{ ceiling, tiers, skills }`.

**CLI:** `node .agents/skills/verification/qa/scripts/skill-size-budget.mjs check [--write] [project-root]`.

**Tiers:** `standard` 22KB (default), `large` 38KB, `xl` 40KB. Skills
explicitly assigned a tier in the `tiers` map; unlisted skills use `standard`.

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

**CLI:** `node .agents/skills/verification/qa/scripts/ship-drift-check.mjs check <base-ref>
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

On completion, **`ship`** moves the change folder out of the active `specs/` zone into a top-level, immutable `archive/` zone:

```
.skillgrid/specs/YYYY-MM-DD-<topic>/  ──►  .skillgrid/archive/YYYY-MM-DD-<topic>/
```

- **`specs/` is active-only.** A folder present in `specs/` is in-flight; a folder in `archive/` is closed. The date prefix in the folder name is preserved as the archive key.
- **The move is mechanical** (shell `git mv` / `mv` only, never model Read→Write) with a **`diff -r` readback** against a pre-move snapshot — an empty diff is the only passing evidence. See `ship` (Step 7).
- **`reflect`** then runs read-only over the archived folder: it completes `report.md`'s retro half in place (from `## Final-State Facts` onward — integration evidence + retrospective + archive summary) and owns the Mnemonic session close.
- `archive/` is created lazily by `ship` if absent. Archived changes are **never deleted or modified** after the move (except the retro sections of `report.md`, completed in place by `reflect` after the move — the QA half is pre-move and included in the `diff -r` readback; the retro half is appended after).
