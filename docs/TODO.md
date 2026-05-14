# TODO

* mcp
  * [X] engram
  * [X] gitnexus
  * [X] context7
  * [X] deepwiki
  * [X] exa
  * [ ] playwright
  * [ ] ai-mcp-sequentialthinking
  * [ ] agent-browser
* tools
  * [X] engram
  * [X] gitnexus
  * [X] ccc
  * [ ] codemaps
* memory
  * [-] engram
    * [X] memory versioning (`version.id`, `version.previous_id`, `timestamps.*`)
    * [X] deterministic conflict resolution rules
    * [X] cross-source ranking strategy with weighted score + freshness decay
    * [X] vector index mapping schema (versioned fields/dimensions/metric)
    * [X] HNSW tuning guidance (`M`, `efConstruction`, memory-optimized variants)
    * [X] pre-allocation + `sync.Pool` policy for hot paths
  * [-] gitnexus
  * [-] ccc
  * [ ] codemaps
* skillgrid-cli
  * installer
  * tui
    * engram integration ?
    * engram tui
  * web ui
    * engram integration ?
* search
  * researcher persona
  * it search
    * use mcp
  * code search
    * use ccc gitnexus
  * [X] agent personas
  * paralel search subagents
  * Missing skills: deep-research, tavily, perplexity-sonar, and the whole brave-* set (brave-web-search, brave-news-search, etc.)
* documentation operations
  * Missing skills: documentation-lookup, documentation-and-adrs, documentation-templates
* [-] ui design
  * [X] sdd-design-ui
  * [ ] Missing skills: design-taste-frontend, frontend-ui-engineering, high-end-visual-design, impeccable, superdesign, huashu-design, image-to-code, etc.
* questions
  * Assign different AI models to different SDD phases
  * Build Loop
  * Specialist persona board
* integrte project level ide plens
  * force kilo, kiro, antigravity to use project level plens
  * use thos ofr PRD, spec creation cicle.
    * `.kilo/plans`

## Components

* [ ] AGENTS.md
  * [ ] CONTEXT.md
  * [X] Karpathy rules (`.agents/rules/skillgrid-karpathy-coding.mdc`)
* rules
* command
  * sdd-init
    * [X] sdd-init skill
    * [X] store to engram
    * [X] `_shared/skillgrid-handoff.md`
    * [ ] status, executive_summary, artifacts, and next_recommended ???
  * sdd-explore
    * [X] sdd-explore skill
    * [X] store to engram
    * [X] `_shared/skillgrid-handoff.md`
    * [ ] status, executive_summary, artifacts, and next_recommended ???
  * sdd-brainstorm
    * [X] Advanced questioning -> sdd-clarify skill
    * [X] sdd-explore skill
    * [X] sdd-propose skill
    * [X] sdd-spec skill
    * [X] sdd-design skill
    * [X] sdd-prd skill
    * [X] sdd-task skill
    * [X] `_shared/skillgrid-handoff.md`
  * sdd-apply
    * [X] smart/dum side
    * [X] process one slice at a time
    * [X] respect `[HITL]` and `[AFK]` labels
    * [X] `_shared/skillgrid-handoff.md`
  * sdd-archive
    * [X] `_shared/skillgrid-handoff.md`
  * sdd-verify
    * [X] `_shared/skillgrid-handoff.md`
  * sdd-ui-design
* skill
  * [X] `_shared/engram-convention.md`
  * [-] `_shared/openspec-convention.md`
    * [ ] Artifact File Paths
  * [X] `_shared/sdd-phase-common.md`
  * [-] `_shared/skillgrid-convention.md`
  * [-] `_shared/skillgrid-handoff.md`
  * [-] sdd-init
    * [X] config backend mode
    * [X] persist context
      * [X] `_shared/engram-convention.md`
      * [X] `_shared/openspec-convention.md`
      * [ ] `_shared/skillgrid-handoff.md`
    * [X] detect files
    * [X] skillgrid folder structure
    * [X] penspec folder structure
    * [-] openspec/config.yaml
      * [ ] ticketing integration
    * [X] create skill-registry
    * [X] Persist Project Context
      * [X] `_shared/skillgrid-convention.md`
    * [X] initialize ccc and gitnexus
    * [X] Return Summary for hybrid mode
      * [X] `_shared/skillgrid-convention.md`
  * [-] sdd-explore
    * [ ] exploration.md 
    * [X] persist context
      * [X] `_shared/engram-convention.md`
      * [X] `_shared/openspec-convention.md`
      * [ ] `_shared/skillgrid-handoff.md`
    * [X] `_shared/sdd-phase-common.md`
    * [X] Persist Artifact
      * [X] `_shared/skillgrid-convention.md`
    * [X] Import PRD Artifacts
  * [-] sdd-propose
    * [!] PRDs vs proposal.md !!! 
  * [-] sdd-design
  * [-] sdd-spec
  * [-] sdd-task
    * [X] agents to group tasks into vertical slice units
    * [X] add `[HITL]` or `[AFK]` labels.
  * [-] sdd-apply
  * [-] sdd-verify
  * [-] sdd-archive
  * [-] sdd-ui-design
  * [ ] sdd-test - runs tests and captures evidence tied to success criteria.
    * Missing skills: playwright, browser-testing-with-devtools, e2e-testing, e2e-runner
  * [ ] sdd-security - performs a deeper security pass when needed.
    * Missing skills: security-review, security-scan, semgrep-security, trivy-security, vulnerability-scanner


  * [ ] orchestrator ???
  * [ ] skill-registry
  * [ ] skillgrid-import-artifacts ???
  * [X] `parallel-delegate` — multi-lane sub-agent handoffs and merge (supersedes planned `skillgrid-parallel-research`).
  * [X] gitnexus-*
  * [X] ccc
  * [X] context7
  * [ ] deepwiki
  * [X] exa-search
  * [ ] playwright

## Plan

### Objective

Ship high-impact workflow upgrades: enforceable pipelines, strict phase gates, model routing, persona-board decisions, and Norse-themed operator clarity where it helps.

### Rethink personas

### Workflow backlog

- [-] Enforcement & review (partial — see Milestone 1):
  - [-] branch-finish protocol (`verify -> merge/PR/keep/discard -> cleanup`)
    - [X] explicit post-merge index refresh (`ccc index`, `npx gitnexus analyze`)
  - [ ] optional git-worktree execution mode for risky/parallel slices — **detailed checklist: [Git worktrees](#git-worktrees-plan)**

### Git worktrees (plan)

**Context (from product notes):** Cursor already supports Task / parallel agents; `parallel-delegate` covers *what* to hand off and merge. **Git worktrees** add an optional *where*: isolated working directories per branch, aligned with `openspec-git-discipline` (proposal on `main` before apply from branch/worktree) and with branch/PR hygiene (`engram-branch-pr`, `engram-commit-hygiene`). Do **not** duplicate the full SDD pipeline in a worktree skill—keep it mechanics + gates + pointers.

**Goal:** A short, project-neutral hub artifact so agents and humans can spin parallel feature lanes safely (especially next to OpenSpec apply-from-worktree flows).

**Deliverables**

- [ ] **Skill** `.agents/skills/git-worktrees/` (or `skillgrid-git-worktrees/`) — single `SKILL.md`:
  - [ ] When to use: parallel implementation lane, risky experiment, long-running review branch, paired with OpenSpec apply instructions that allow worktree *if* proposal state is on `main` (cite `openspec-git-discipline`).
  - [ ] Commands / checklist: `git worktree list`, `git worktree add <path> <branch>`, linked clone vs same-repo path conventions, naming convention for worktree dirs (e.g. `../repo.wt/<branch-slug>`).
  - [ ] Cleanup: `git worktree remove`, prune, confirm no uncommitted loss; branch delete policy after merge.
  - [ ] **Do not** auto-commit or merge; explicit user consent (same bar as `openspec-git-discipline`).
- [ ] **Docs** — `docs/04-commands.md` (optional `/sdd-worktree` or document as skill-only), `docs/02-workflow-usage.md` (one subsection: when worktree vs single clone), cross-link `openspec-git-discipline` and branch-finish ritual.
- [ ] **Registry** — add skill row + compact rules block in `.skillgrid/project/SKILL_REGISTRY.md` when the skill lands.
- [ ] **Optional workflow** `.agents/workflows/sdd-worktree.md` — thin wrapper that loads the skill + returns standard envelope (`status`, `next_recommended`), only if we want a slash command mirror.

**Branch-finish ritual (overlap, do not fork):** One-page “finishing a development branch” behavior stays aligned with existing `engram-branch-pr` + `openspec-git-discipline` + archive/sync; the worktree plan should **link** to that ritual (squash/rebase policy, final verify, post-merge cleanup) rather than re-specify it.

**Parallel agents:** Use existing `parallel-delegate` for dispatch/merge; worktree skill only supplies filesystem/git layout and safety checks. Execute wericle slices: https://lobehub.com/skills/akornmeier-claude-config-openspec-dev?activeTab=skill

### Milestone 3 — Model Routing + Session Efficiency

- [ ] Add per-phase model routing in config:
  - [ ] `explore` fast/cheap
  - [ ] `apply` balanced
  - [ ] `verify/design` stronger reasoning
- [ ] Add runtime preset switching (without reinstall/restart)
- [ ] Add subagent session reuse for repeated slices/boards
- [ ] Add safe auto-continuation:
  - [ ] cooldown
  - [ ] AFK-only advancement
  - [ ] fresh verification precondition

### Workflow Hardening Backlog

- [ ] Optional worktree mode for risky or parallel implementation lanes — **see [Git worktrees](#git-worktrees-plan)**
- [ ] Agent health-check command (ping all required personas + MCP readiness)
- [X] Enforce split usage: `sdd-*` executors + Nordic gate personas (`tyr`/`heimdall` hard gate)
- [ ] Persona report contract template:
  - [ ] severity
  - [ ] evidence path
  - [ ] impacted artifact(s)
  - [ ] disposition (`must-fix | accept-risk | follow-up`)
- [ ] CI guards:
  - [ ] tasks label validator
  - [ ] spec matrix presence
  - [ ] gate result must exist before archive
