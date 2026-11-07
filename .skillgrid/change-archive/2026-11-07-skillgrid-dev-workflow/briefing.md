# Development Workflow — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-07)

## Problem / Intent

Developers (human or agent) need one ordered, step-by-step development workflow
for Delivery changes in this repo: for each step — which skill to invoke, which
subagents to dispatch, which MCP tools to use, which files are written, and
where the user gates are. Today the pipeline knowledge is split across
`.agents/skills/` (one skill per phase), `_shared/rules/sdd-structure.md`
(layout + phase order), and `.skillgrid/config.yaml` (gates/commands); the
durable concept page (`workflows/sdd-pipeline.md`) captures the *what and why*
but not the executable *how*. The approved plan
`skillgrid-dev-workflow-plan` (2026-11-07, user-approved 2026-11-07) is the
content of this change; it lands as a repo artifact so it survives sessions
(ADR-0013: the repo is the source of truth — a session-local `local://` plan
dies with the session).

## Purpose & Success Criteria

- **Purpose:** a new session (human or agent) can execute any Delivery change
  top-to-bottom from one document: step → skill → subagents → MCP → CLI →
  files → gates, plus the cross-cutting tool contracts.
- **Success criteria:**
  - `.skillgrid/projectwiki/workflows/development-workflow.md` exists with OKF
    frontmatter and covers Steps 0–10 (each: skill, subagents, MCP, CLI,
    files, gates) + cross-cutting tool usage + subagent summary table +
    assumptions & contingencies, faithful to the approved plan.
  - `workflows/sdd-pipeline.md` cross-references the new page as the
    step-by-step execution guide.
  - The `Development Workflow` term is in the business glossary.
  - The KB index is rebuilt in the same edit (same-edit rule);
    `mnemonic kb validate --strict` exits 0 and `mnemonic kb lint` is clean.
- **Out of scope:**
  - No changes to pipeline skills (`.agents/skills/`), `sdd-structure.md`, or
    `config.yaml`.
  - No new ADR (doc placement clears no ADR bar).
  - No v1→v2 `state.yaml` migration (separate change; this change updates the
    v1 pointer only — drift-free, see Open Questions).
  - No changes to the 16 in-flight changes.

## Context

- ADR-0048 (hybrid SoT): the `.skillgrid/` Markdown bundle is the SoT for the
  project wiki (decisions/invariants/glossary/reference/workflows).
  `workflows/` already holds `sdd-pipeline.md` (the durable concept page) —
  this page is its step-by-step complement, same zone, same frontmatter shape.
- ADR-0013: the repo is the source of truth; the workflow must live in the
  repo, not a session artifact.
- The approved plan inlines the config facts it relies on (from
  `.skillgrid/config.yaml`): `mnemonic.enabled: true`; `ticketing.type:
  hermes-kanban` (board `skillgrid`, assignee `default`); `bdd.enabled: true`
  (stack javascript); `rules.tiers.default: T2`; `rules.fast_track.enabled:
  true`; test `go test ./...`; build/setup `go build ./...`; format
  `gofmt -w ./...`; lint none (go vet inline); trivy + jscpd advisory.
- Vault conventions: OKF v0.2 frontmatter (`type: BusinessWorkflow`,
  `title:`), agent-style v0.9.0 prose (ADR-0051); `index/` projections are
  engine-owned (never hand-edited); same-edit rule — every vault page write is
  followed by `mnemonic kb index`.
- Pre-ship gate sequence (plan, Cross-cutting): `kb index` → `kb validate
  --strict` (drift → exit 1) → `kb lint` (target: clean).
- Fits existing patterns: **yes** — mirrors `workflows/sdd-pipeline.md`
  frontmatter + section style; the cross-link follows that page's
  "Participating sites" pointer pattern; no new zone, no new dependency.

## Approaches Considered

- **Chosen:** Adapt the approved plan into a vault page under `workflows/` —
  faithful to the plan's step structure and content, vault frontmatter + prose
  conventions, cross-linked from `sdd-pipeline.md`, indexed in the same edit.
- **Rejected:** Verbatim copy of the plan file — the plan's markdown is
  session-oriented (inline execution directives, `local://` recovery notes,
  "inline plan intact" instructions) and would read as a paste; the plan's
  section structure survives as the page's structure, re-stated as durable
  repo content.
- **Rejected:** Thin pointer page + external plan — the external plan is
  session-local and dies with the session (ADR-0013); the repo must carry the
  workflow.

## Requirements

1. **Workflow page exists with all steps:** `.skillgrid/projectwiki/workflows/development-workflow.md` is a vault page covering the full pipeline.
   - **Current:** `workflows/` holds only `sdd-pipeline.md` (the durable concept page: phases, zone rule, terminal close) — no step-by-step execution guide.
   - **Target:** the page exists with OKF frontmatter (`type: BusinessWorkflow`, `title: "Development Workflow"`), a Context section (what the workflow is, which change classes it applies to, the config facts it relies on), `## Step 0` … `## Step 10` sections each naming the skill, subagents, MCP usage, CLI, files, and gates for that step, a cross-cutting tool-usage section (memory, code index ladder, knowledge base, commit protocol, resume), a subagent summary table, and an assumptions & contingencies section.
   - **Acceptance:** `test -f` passes; frontmatter carries `type: BusinessWorkflow`; headings `## Step 0` through `## Step 10` all present (11 step sections including Step 0).
   - **Acceptance scenario:** `happy path workflow-page-exists-with-all-steps` → `acceptance.feature`
2. **Step fidelity:** every step preserves the approved plan's substance.
   - **Current:** the content lives only in the session-local approved plan.
   - **Target:** for each of Steps 0–10 the page names the same skill, the same subagent pattern (counts + roles), the same MCP tools, the same files, and the same user gates as the approved plan; the cross-cutting section carries the Code Index Ladder (binding order), the KB read/write contracts (same-edit rule, pre-ship gate), the commit protocol (work-unit-commits), and the resume procedure; the subagent summary table matches the plan's table; the assumptions & contingencies section carries the hermes-kanban discovery STOP, the MCP build-drift fallbacks, the qa-gate path drift + direct-run contingency, the state.yaml v1/v2 note, and the parallel-worktrees rule.
   - **Acceptance:** per-step spot checks: Step 1 names `brainstorming` + the HARD GATE; Step 2 names `writing-blueprints` + critic subagent (mandatory T2/T3) + plan-review subagent; Step 3 names `acceptance-test-authoring` + cucumber dry-run + gherkin lint; Step 4 names `slicing` + `ticketing` + the 4 literal Delivery Strategy guard lines + the 4.3 user gate (Implement/Revise); Step 5 names `subagent-execution` + per-ticket loop (claim → brief → implementer → TDD → review → fix loop ≤5 → complete) + the four stops; Step 6 names `qa` + 0 subagents + the 9.5 deterministic gate chain + four-state verdict; Step 7 names `requesting-code-review` + exactly 3 parallel reviewers + the parallel-code-review escalation (5–7 specialists + red team LAST); Step 8 names `receiving-code-review`; Step 9 names `ship` + the pre-mutation QA hard gate + the mechanical `git mv` + `diff -r` readback; Step 10 names `reflect` + the terminal session close + web-cache human gate.
   - **Acceptance scenario:** `happy path step-fidelity-spot-checks` → `acceptance.feature`
3. **Cross-linking:** the concept page points to the execution guide.
   - **Current:** `sdd-pipeline.md` has no pointer to a step-by-step workflow.
   - **Target:** `sdd-pipeline.md` references `development-workflow.md` (one line, in the intro or Participating sites) as the step-by-step execution guide for Delivery changes.
   - **Acceptance:** `grep -q 'development-workflow' .skillgrid/projectwiki/workflows/sdd-pipeline.md` exits 0.
   - **Acceptance scenario:** `happy path pipeline-page-cross-links` → `acceptance.feature`
4. **Glossary term:** the new term is resolvable.
   - **Current:** no `Development Workflow` term in `glossary/01-business-terms.md`.
   - **Target:** one glossary row defining `Development Workflow` (the ordered step-by-step execution guide for Delivery changes; the executable complement to the SDD pipeline concept page), with Use-When and Avoid columns per the table format.
   - **Acceptance:** `grep -q 'Development Workflow' .skillgrid/projectwiki/glossary/01-business-terms.md` exits 0.
   - **Acceptance scenario:** `happy path glossary-term-present` → `acceptance.feature`
5. **KB indexed and valid (no new drift):** the vault projections are rebuilt and the new page adds no drift.
   - **Current:** the projectwiki index predates the new page; `projectwiki validate --strict` (non-JSON) already exits 1 on 2 pre-existing unresolved (unattested) grounded pages — `workflows/sdd-pipeline.md` and `decisions/04-adr-0050-harness2-host-integration.md` (verified 2026-11-07; not from this change).
   - **Target:** after the page + cross-link + glossary edits, a freshly-built `mnemonic` binary (from `mnemonic-v2/` source — the installed 0.2.0 predates the kb→projectwiki rename, commit 5686caba) runs `projectwiki index` in the same edit (same-edit rule); the new page carries resolvable frontmatter `anchors` so it is **attested → fresh**; `projectwiki validate --strict` reports the new page **fresh** and the 2 pre-existing drifts only (no new drift from this change); `projectwiki lint` reports clean (no new findings).
   - **Acceptance:** `projectwiki index` exits 0 (new page indexed) + `projectwiki lint` exits 0 clean + `projectwiki validate --strict` reports the new page fresh with the 2 pre-existing drifts only (no new drift). Note: `validate --strict` (non-JSON) exits 1 on the 2 pre-existing drifts regardless — the gate's pass criterion is **"no new drift from this change"**, not "exit 0" (user decision 2026-11-07, no-new-drift gate).
   - **Acceptance scenario:** `happy path kb-indexed-and-valid` → `acceptance.feature`
6. **Edge coverage — drift is detectable:** the strict validator still catches a broken anchor in the new page.
   - **Current:** `projectwiki validate --strict` exits 1 on the 2 pre-existing unresolved pages (baseline drift = 2).
   - **Target:** a temporarily broken anchor in `development-workflow.md` makes **that page** drift in `projectwiki validate --strict` (drift count rises from the 2-drift baseline to 3), and restoring the anchor returns the page to fresh (back to the 2-drift baseline) — the validator is not weakened by the new page.
   - **Acceptance:** throwaway mutation test (freshly-built binary): baseline (new page fresh, 2 pre-existing drifts) → break one anchor in `development-workflow.md` → re-index → new page drifts (3 drifts) → restore the anchor → re-index → new page fresh (2 drifts). (Executed as a QA throwaway, not a permanent test.)
   - **Acceptance scenario:** `failure kb-validate-detects-anchor-drift` → `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:** new vault page `workflows/development-workflow.md`; a one-line cross-reference edit to `workflows/sdd-pipeline.md`; a one-row edit to `glossary/01-business-terms.md`; KB index rebuild. No Go/JS code changes.
- **Interfaces:** page section contract — `# Development Workflow` title; `## Context`; `## Step 0` … `## Step 10` (each with Skill / Subagents / MCP / CLI / Files / Gates sub-structure per the plan); `## Cross-cutting tool usage`; `## Subagent summary`; `## Assumptions & contingencies`.
- **Data flow:** approved plan (in-session content) → vault page; `projectwiki index` (freshly-built binary) re-derives `index/` projections (engine-owned); `validate --strict` + `lint` gate the move.
- **Error handling:** `projectwiki validate --strict` reports the new page drifted (broken anchor) → fix the page anchor/pin and re-index in the same edit; `projectwiki lint` findings → fix or record in `.skillgrid/WINDOWS.md`; `hermes kanban` discovery failure at ticketing → hard STOP (never publish to a fallback board silently).
- **Dependencies:** none new (Go: zero; Node: zero) — the gate uses a freshly-built `mnemonic` binary from `mnemonic-v2/` source (the installed 0.2.0 is stale, see Assumptions) and the existing projectwiki vault engine.

## Testing Decisions

- **What makes a good test:** verify external behavior — file existence, structural headings, link/term presence, command exit codes — not content paraphrase.
- **Modules to test:** the vault page (structure + per-step fidelity spot checks), `sdd-pipeline.md` (cross-link), the glossary (term row), the `mnemonic kb` commands (exit codes).
- **Prior art:** the qa-gate chain's structural checks (`check-adr-invariants.mjs`, `okf-lint.mjs`) — grep/exit-code style verification over vault files.
- **Edge cases:** missing frontmatter (okf-lint), stale index after edit (validate --strict exit 1), anchor drift in the new page (Requirement 6 throwaway mutation).

## Impact on Global Docs

- `.skillgrid/projectwiki/reference/00-prd.md`: None (the `reference/` zone is absent in this checkout; a workflow doc is not PRD scope).
- `.skillgrid/projectwiki/ASSUMPTIONS.md`: None (no new ADR; no new product-understanding fact — the page itself is the artifact).
- `.skillgrid/projectwiki/ARCHITECTURE.md`: None.

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.95  | 0.75 | ✓      | approved plan fully specifies the goal |
| Boundary Clarity    | 0.90  | 0.70 | ✓      | explicit out-of-scope list         |
| Constraint Clarity  | 0.90  | 0.65 | ✓      | vault conventions, same-edit rule, zone rule, ADR bar, pre-ship gate |
| Acceptance Criteria | 0.95  | 0.70 | ✓      | 6 falsifiable requirements with pass/fail checks |
| **Clarity**         | **0.05** | ≤0.20 | ✓ | 1 − min(dimensions) |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | Path classification | New Function — a new capability (the workflow doc) in an existing project; the design content is the user-approved plan (2026-11-07) |
| 2     | Where does the doc live, in what form? | `workflows/development-workflow.md` (ADR-0048 SoT zone, beside `sdd-pipeline.md`); adapted vault page (OKF frontmatter + agent-style prose), not a verbatim plan paste; cross-linked from the concept page; no ADR (below bar) |

## Open Questions & Assumptions

- **Assumption:** the 16 in-flight v1-era changes stay out of scope; a full
  v1→v2 `state.yaml` migration is a separate change. This change updates the
  v1 pointer only (`pipeline.current_change` → this change), which is
  drift-free per `state-drift-check.mjs` v1 branch; creating a v2 `changes:`
  map with one of 17 changes would register 16 presence drifts.
- **Assumption:** the hermes kanban board `skillgrid` is reachable at
  Step 4.2; discovery failure is a hard STOP (per the plan).
- **Assumption (verified 2026-11-07):** the installed `mnemonic` 0.2.0 binary is
  **stale** — it predates the kb→projectwiki rename (commit 5686caba), exposes
  `kb` (expects `.skillgrid/knowledge/`, resolves to old project `aiskillgrid`),
  and has no `projectwiki` command. The pre-ship gate therefore runs on a
  freshly-built binary from `mnemonic-v2/` source
  (`go build -o <tmp> ./cmd/mnemonic` → `projectwiki index/validate/lint`,
  resolves to project `skillgrid-v2`, store `skillgrid-v2.sqlite`). The 2
  pre-existing unresolved drifts (`workflows/sdd-pipeline.md`,
  `decisions/04-adr-0050-harness2-host-integration.md`) are a baseline, not from
  this change (see R5/R6).
- **Question:** none blocking.

## Decisions (ADR)

None — no decision for this change cleared the ADR bar (hard to reverse +
surprising without context + real trade-off). Doc placement in an existing
zone is trivially reversible and follows ADR-0048 directly.

## Terms

- `Development Workflow` → `.skillgrid/projectwiki/glossary/01-business-terms.md` — new term introduced by this change (the ordered step-by-step execution guide for Delivery changes; the executable complement to the SDD pipeline concept page).
