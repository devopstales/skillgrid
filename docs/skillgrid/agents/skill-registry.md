# Skill Registry (generated index — OPTIONAL, never a workflow gate)

Source: `.agents/skills/*/SKILL.md` frontmatter (`name` + `description`).
Regenerate when skills are added/renamed. Canonical workflow:
`onboard → [explore] → propose → design → spec → tasks → apply ⇄ verify → [review] → archive`
(see `.agents/skills/_shared/conventions/sdd-structure.md`).

## SDD core (`family: sdd` — orchestrator delegates; `delegate_only: true` = never run inline)

| Skill | Entry / role |
|---|---|
| `use-skillgrid` | Entry router — classify, detect init, route, resume, user gate |
| `sdd-onboard` | Bootstrap (greenfield/brownfield): map → init → agent-context → constraints → domain |
| `sdd-init` | Legacy init variant — detect facts, write skeleton + registry + agent block (see Gotchas) |
| `sdd-explore` | Optional read-only exploration → `exploration.md` |
| `sdd-propose` | `proposal.md` (intent, Capabilities contract, hypothesis, rollback) |
| `sdd-design` | `design.md` (HOW + threat matrix) |
| `sdd-spec` | `specs/{domain}/spec.md` (WHAT: RFC 2119 + Given/When/Then, delta semantics) |
| `sdd-tasks` | `tasks.md` (phases + Review Workload Forecast + RED-test tasks) |
| `sdd-apply` | Writes code, marks `[x]`, persists `apply-progress` |
| `sdd-verify` | `verify-report.md` — PASS / PASS WITH WARNINGS / FAIL (judge, never fix) |
| `sdd-review` | Optional post-verify gate (human approval) — REVIEW-PASS / BACK-TO-APPLY / waived |
| `sdd-archive` | Spec sync + mechanical move to `archive/YYYY-MM-DD-{change-name}/` |

Gotchas: `sdd-init` overlaps `sdd-onboard` (Step 3 = init) — `use-skillgrid` routes to `sdd-onboard`;
`design-spike` is a pre-propose helper, not a stage.

## Execution / quality helpers

| Skill | When |
|---|---|
| `subagent-execution` | Dispatch a fresh subagent per task + review after each (parallel-investigation protocol included) |
| `simple-execution` | Small/tightly-coupled plans inline, one task at a time (RED/GREEN/TRIANGULATE/REFACTOR when TDD) |
| `verification` | Before claiming done — run checks, evidence before assertions |
| `tdd` | Feature/bugfix — failing test first, then minimal code, then refactor |
| `questioning` | Ambiguous scope — stress-test intent branch by branch before design |
| `requesting-code-review` | High-risk step / pre-archive — dispatch a fresh reviewer with crafted context |
| `review-reception` | Receiving review — verify each finding against code, one item at a time |
| `quality-security-review` | Advisory lens after verify passes — vet/gofmt/trivy + checklist (never blocking) |
| `debugging` | Any bug/failure — root-cause first, before fixes |
| `work-unit-commits` | Reviewable commits — one deliverable behavior per commit |
| `unlazy` | Long/multi-part work — acceptance gates + Depth Tree + re-verification |
| `finishing-a-development-branch` | Change complete — decide how to integrate |
| `isolated-workspace` | Feature work needing isolation (worktree fallback) |

## Knowledge / memory / research

| Skill | When |
|---|---|
| `mnemonic` | ALWAYS ACTIVE — memory, code index, web cache protocol |
| `mnemonic-code-index` | Retired — redirect to `mnemonic` (Step 5) |
| `investigate` | Research against primary sources → one cited Markdown file |
| `dispatching-parallel-agents` | Retired — redirect to `subagent-execution` |
| `issue-creation` | File tracker issues (Backlog.md / GitHub / GitLab / Jira) |
| `glossary` | Authoring specs/docs — term reuse before inventing terms |
| `codebase-design` | Deep-module vocabulary for designing/improving module interfaces |
| `creating-skills` | Create/refine a skill from a repeating workflow |

## Design / frontend / misc

| Skill | When |
|---|---|
| `design-spike` | Throwaway prototype (taste/UI/architecture/smoke) before locking `proposal.md` — marked PROTOTYPE |
| `design-taste-frontend` | Landing pages/portfolios/redesigns without templated look |
| `redesign-existing-projects` | Upgrade existing sites to premium quality without breaking function |
| `archify` | Architecture/workflow/sequence diagrams as explorable HTML |
| `graphify` | Knowledge graphs (code/docs/papers) — HTML + JSON + audit |
| `adhd` | Open-ended ideation — parallel divergent branches, score/prune/deepen |
| `playwright` | Playwright E2E — Page Objects, selectors, MCP workflow |
| `improve-codebase-architecture` | Scan for deepening opportunities → HTML report → grill |
| `_shared` | Not invokable — shared conventions, references, tracker templates, agent-config |

## Shared references (canonical, single-source)

- `_shared/references/threat-matrix.md` — applicability-driven threat matrix
- `_shared/references/strict-tdd.md` — RED → GREEN → TRIANGULATE → REFACTOR module
- `_shared/references/delta-spec-format.md` — ADDED/MODIFIED/REMOVED/RENAMED + RFC 2119
- `_shared/conventions/sdd-structure.md` — layout, order, artifact paths (wins on conflict)
- `_shared/conventions/fast-track.md` — trivial/small waivers
- `_shared/conventions/hybrid-degradation.md` — degraded store modes + `Degraded:` envelope line
