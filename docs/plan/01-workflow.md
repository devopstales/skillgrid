# ICEbreaker SDD Plan

Spec Driven Development Workflow using `Markdown` structure as spec source and selectable (GitHub, GitLab, Jira, Backlog.md) ticketing system. `Backlog.md` is the default tracker. It uses `Mnemonic` for long-term memory and code indexing.

## ICEbreaker SDD Workflow

```
init -> explore -> propose -> design -> tasks -> spec -> apply -> verify -> archive
```

* init:
    * `sdd-init`
        * Check `AGENTS.md` or `CLAUDE.md` exists. Search `project_name`, `tech_stack`, `testing_capabilities`, `issue_tracker`
        * Check `docs/skillgrid/config.yaml` exists. Search `project_name`, `tech_stack`, `testing_capabilities`, `issue_tracker`
        * Check Mnemonic. Read `project_name`, `tech_stack`, `testing_capabilities`, `issue_tracker`
        * Check git repo - determine `project_name`, `issue_tracker`
        * Inspect project files (`package.json`, `go.mod`, `pyproject.toml`, CI, lint/test config) - determine `tech_stack`
        * Detect test runner, test layers, coverage, linter, type checker, and formatter - determine `testing_capabilities`
        * Validate findings with user
        * Initialize persistence:
            * Build skill registry at `docs/agents/skill-registry.md`
            * Agent config `AGENTS.md` or `CLAUDE.md`
            * `docs/skillgrid/` skeleton: `config.yaml`, `changes/`, `archive/`
            * Write Mnemonic observations
            * If `backlogmd` selected, initialize it too
* explore:
    * `sdd-explore` investiage
        * Read `docs/skillgrid/config.yaml` for project context (rules, context)
        * Recover Mnemonic context: `mem_search("sdd-init/{project}")` + `mem_get_observation`, `mem_search("sdd/<NNN-slug>/")`
        * Use code-index ladder: `code_status -> code_index -> code_search -> code_read`
        * Investigate entry points, affected modules, existing tests, patterns, dependencies/coupling
        * Analyze & compare approaches (pros/cons/complexity/effort matrix)
        * Persist artifact: `docs/skillgrid/changes/<NNN-slug>/research.md` + Mnemonic `sdd/<NNN-slug>/research` (`topic_key` upsert)
        * Return structured envelope (status, summary, artifacts, next, risks)
* propose:
    * `sdd-propose` shape
        * **Reserve the NNN change number** (scan `docs/skillgrid/changes/` + `archive/` + Mnemonic `sdd/{project}/changelog`, take `max+1`, zero-pad to 3 digits)
        * Recover context: `mem_search("sdd/<NNN-slug>/research")` + `mem_get_observation`, `sdd-init/{project}`, `docs/skillgrid/config.yaml`
        * Interactive only: intent question round (business/product, not delivery mechanics)
        * Write `intent.md`: Business Problem, Target Users, Business Rules, Success Criteria (UAT-level), Scope, **Step Blueprint** (contract with sdd-tasks), Affected Areas, Risks, Rollback, Dependencies
        * Persist: `docs/skillgrid/changes/<NNN-slug>/intent.md` + Mnemonic `sdd/<NNN-slug>/intent` (`topic_key` upsert)
        * Append NNN reservation to `sdd/{project}/changelog` (Mnemonic)
        * Enforce <500 words; every intent has rollback plan + success criteria
* design:
    * `sdd-design` architect
        * Recover context: `mem_search("sdd/<NNN-slug>/intent")` + `mem_get_observation` (required — Step Blueprint is the contract), `sdd/<NNN-slug>/research` (required), `sdd-init/{project}`, `docs/skillgrid/config.yaml`, `docs/skillgrid/archive/NNN-slug/plan.md` (prior art)
        * Read actual code via code-index ladder: `code_status -> code_index -> code_search -> code_read` (no external binaries)
        * **Per-step WHAT** block for every intent Step Blueprint entry (user-observable behavior; becomes the spec's Gherkin scenarios)
        * Applicability-driven threat matrix: mark each row `Applicable` or `N/A: reason`; applicable rows carry the owning step — plan → tasks → spec
        * Write `plan.md`: Technical Approach, Architecture Decisions (Choice/Alternatives/Rationale), Data Flow, **Impacted Files Map (with a Step column)**, Step WHAT, Mnemonic Integration, Threat Matrix, Migration/Rollout, Open Questions
        * Persist: `docs/skillgrid/changes/<NNN-slug>/plan.md` + Mnemonic `sdd/<NNN-slug>/plan` (`topic_key` upsert)
        * Enforce <850 words; each applicable threat row names its owning step and propagates to tasks then spec
* tasks:
    * `sdd-tasks` decompose — runs BEFORE spec (the step tree must exist before acceptance is written)
        * **Own the step tree and NN numbering**: create `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/` for every intent Step Blueprint entry
        * Inherit the plan's per-step threat rows into per-step RED tasks (before their production tasks)
        * Assign every file in the plan's Impacted Files map to exactly one step; encode inter-step deps as `Depends on:` notes
        * Write each step's `tasks.md` (punch-list + four change-level guard lines in step 01)
        * Persist: `steps/<NN-name>/tasks.md` per step + Mnemonic `sdd/<NNN-slug>/tasks` (concatenated by step)
* spec:
    * `sdd-spec` — per-step Gherkin acceptance
        * For **every** step folder (created by sdd-tasks), write `steps/<NN-name>/acceptance.feature`
        * One `Feature` + ≥ 3 `Scenarios` (`@happy` `@edge` `@failure`); scenario names unique and referenceable
        * Every plan per-step WHAT bullet → a scenario; every applicable threat-row → a scenario in its owning step
        * Persist: each `steps/<NN-name>/acceptance.feature` + Mnemonic `sdd/<NNN-slug>/spec` (concatenated by step)
* apply:
    * git-worktree
    * execute-task (per assigned step / task)
    * subagent-driven-development
    * TDD (Strict TDD or Standard per `docs/skillgrid/config.yaml` → `rules.apply.tdd`)
    * BDD (per step's `acceptance.feature` scenarios)
    * unlazy
    * Mark each `steps/<NN-name>/tasks.md` task `[x]` as it completes
    * Produce a per-step evidence table (focused test, acceptance coverage, runtime harness, rollback boundary)
    * Persist cumulative apply-progress to Mnemonic `sdd/<NNN-slug>/apply-progress`
* verify:
    * **Per step**: write `steps/<NN-name>/verification.md` with verdict `PASS | PASS WITH WARNINGS | FAIL`
    * Gate: every scenario in that step's `acceptance.feature` maps to a passing test run at runtime
    * Strict TDD mode adds TDD Compliance, Assertion Quality, Test Layer, Changed-File Coverage, Quality Metrics sections
    * Persist concatenated per-step reports to Mnemonic `sdd/<NNN-slug>/verification`
* archive:
    * **No cross-tree merge** (v2 model has no main-specs root; the acceptance contract lives in each step)
    * Gate on every step's `verification.md` verdict being `PASS` or `PASS WITH WARNINGS`; gate on every step's `tasks.md` having no unchecked tasks
    * Mechanical shell move `docs/skillgrid/changes/<NNN-slug>/` → `docs/skillgrid/archive/<NNN-slug>/` (`git mv` or `mv`), mandatory `diff -r` readback (empty diff = pass)
    * Persist `archive-report` to Mnemonic `sdd/<NNN-slug>/archive-report` with observation IDs of every artifact read (lineage)

## File Structure

### Artifact File Structure

```bash
.backlog/
├── archive/
├── assets/
│   └── images/
├── completed/
├── decisions/
├── docs/
├── drafts/
├── milestones/
├── tasks/
├── config.yml
├── readme.md
docs/
├── agents/
│   ├── skill-registry.md
│   ├── issue-tracker.md
│   └── domain.md
docs/
└── skillgrid/
    ├── config.yaml                 # stack, context, rules per phase
    ├── changes/                            # Active development branch/context
    │   └── <NNN-slug>/                   # Specific change context (e.g., 001-oauth-login)
    │       ├── intent.md                   # WHY & WHAT: Business goals & High-level UAT criteria
    │       ├── research.md                 # SPIKE & FINDINGS: API investigation & discovery notes
    │       ├── plan.md                     # HOW: Architecture decisions & impacted files map + per-step WHAT
    │       ├── state.yaml                  # DAG state (survives compaction)
    │       └── steps/                      # SUBTASKS: Broken down sequential execution phases
    │           └── <NN>-<step-name>/       # Isolated step (e.g., 01-db-migration)
    │               ├── tasks.md            # Execution punch-list for the AI agent
    │               ├── acceptance.feature  # End-to-End User Acceptance Test (Gherkin/BDD style)
    │               └── verification.md     # Per-step PASS/FAIL gate + execution evidence
    └── archive/                            # HISTORICAL RECORD
        └── <NNN-slug>/                     # Completed changes moved here after successful gates
```

### Notes

* NNN = 3-digit zero-padded change number, reserved by `sdd-propose`. Never reused.
* NN = 2-digit zero-padded step number within a change, allocated by `sdd-tasks`. Never renumbered after creation.
* Per-step files: `tasks.md` (sdd-tasks → sdd-apply marks `[x]`), `acceptance.feature` (sdd-spec), `verification.md` (sdd-verify). All three live in the same `steps/<NN-name>/` folder.
* Archive in v2 is a **pure move** of the entire change folder — there is no "merge into main specs" step because there is no main-specs tree in the new layout.

### Sub-Agent Scratch Structure

```bash
.ICEbreaker/sdd/
└── <change-id>/
    ├── context/
    │   └── phase-input.md
    ├── phase-status/
    │   ├── init-status.md
    │   ├── explore-status.md
    │   ├── propose-status.md
    │   ├── design-status.md
    │   ├── spec-status.md
    │   ├── tasks-status.md
    │   ├── apply-status.md
    │   ├── verify-status.md
    │   └── archive-status.md
    ├── work-unit/
    │   ├── assignments.md
    │   └── results/
    └── progress/
        ├── task-progress.md
        └── evidence/
```

## Memory Topics (Mnemonic)

Always `hybrid` — persist to both Mnemonic and filesystem.

Cross-cutting primitives:
* `questioning` — questioning/clarification skill (classify + design tree, frontier, rounds, recommendations, approval gate) used by sdd-explore, sdd-propose, sdd-init. Persists to `sdd/<NNN-slug>/grill` (Mnemonic) and `docs/skillgrid/changes/<NNN-slug>/interview.md`.

```
sdd-init/{project}
sdd/{project}/issue_tracker
sdd/{project}/testing-capabilities
sdd/{project}/changelog
sdd/<NNN-slug>/research
sdd/<NNN-slug>/intent
sdd/<NNN-slug>/plan
sdd/<NNN-slug>/tasks        (per step, concatenated)
sdd/<NNN-slug>/spec         (per step acceptance.feature, concatenated)
sdd/<NNN-slug>/issue-creation
sdd/<NNN-slug>/apply-progress
sdd/<NNN-slug>/verification (per step, concatenated)
sdd/<NNN-slug>/archive-report
sdd/<NNN-slug>/state
```

## Ticketing

* Default tracker: **Backlog.md** (`backlogmd`)
* Selectable: GitHub (`gh` CLI), GitLab (`glab` CLI), Jira (`jira` CLI), Backlog.md (`backlog` CLI)
* Tracker choice persisted to Mnemonic `sdd/{project}/issue_tracker` during `sdd-init`
* Conventions in `_shared/issue-tracker/`; triage labels in `_shared/triage-labels.md`
* Backlog.md storage: `.backlog/tasks/<ID>.md` (set `backlog_directory: .backlog` in `backlog.config.yml`)
* `issue-creation` skill maps tasks → tracker issues; duplicate-search first


commit-convention
conventions into AGENTS.md as rules.


tree-sitter-cli
engram-compat
openspec
gitnexus


```
               /\
              /  \
          .--/\--/\--.
         /  /  \/  \  \
        /  /   /\   \  \
   .---/  /   /  \   \  \---.
  /   /  /   / /\ \   \  \   \
 /   /  /___/ /  \ \___\  \   \
/   /  /____\/____\/____\  \   \
\   \  \    /\    /\    /  /   /
 \   \  \  /  \  /  \  /  /   /
  \   \  \/____\/____\/  /   /
   '---\  \   \  /   /  /---'
        \  \   \/   /  /
         \  \  /\  /  /
          '--\/--\/--'
              \  /
               \/
```

```
               /\
              /  \
             / /\ \
         _  / /  \ \  _
        / \/ /    \ \/ \
       /  / /      \ \  \
      /  / /   /\   \ \  \
     /  / /   /  \   \ \  \
    /  / /___/ /\ \___\ \  \
    \  \ \___\ \/ /___/ /  /
     \  \ \   \  /   / /  /
      \  \ \   \/   / /  /
       \  \/        \/  /
        \_/ \      / \_/
            \ \  / /
             \ \/ /
              \  /
               \/
```