# Skills

The 28 skills under `.agents/skills/`. Do not duplicate a general capability as a stage skill — stages load general skills.

## Quick path

| Group | Rule | Examples |
|-------|------|----------|
| **Entry / router** | Invoked before any response; routes the phase | `using-skillgrid` |
| **Workflow stages** | Own a pipeline stage; write `.skillgrid/` artifacts | `onboarding`, `writing-blueprints`, `slicing`, `qa`, `ticketing` |
| **Discovery** | Reduce ambiguity before locking a blueprint | `interviewing`, `brainstorming`, `spike`, `sketch`, `research`, `deep-research` |
| **Execution** | Run the plan; commit; resume | `simple-execution`, `subagent-execution`, `parallel-execution`, `work-unit-commits`, `resume` |
| **Quality** | Evidence-based verification and review | `test-driven-development`, `test-driven-verification`, `acceptance-test-authoring`, `requesting-code-review`, `receiving-code-review`, `parallel-code-review` |
| **Cross-cutting** | Reusable across stages and outside SDD | `mnemonic`, `isolated-workspace`, `structured-debugging`, `architectural-decision-records`, `ponytail` |

Priority: `using-skillgrid` → workflow stage → the general skills that stage loads.

## Entry / router

| Skill | Role |
|-------|------|
| `using-skillgrid` | Invoke the relevant skill before ANY response. Config/resume/domain-model checks, context discipline, skill priority, platform adaptation. Carries per-platform tool references (`codex`, `pi`, `antigravity`, `hermes`, `gemini`). |

## Workflow stages

| Skill | Role |
|-------|------|
| `onboarding` | One-time project setup: detect stack/testing/tracker/security, confirm facts (blocking), write `.skillgrid/config.yaml` + `AGENTS.md` block. |
| `writing-blueprints` | Write a bite-sized TDD plan (`blueprint.md`): Must-Haves, one-way-door tags, no placeholders, self-review + fresh-eyes plan review, then hand off. |
| `slicing` | Break the blueprint into vertical tracer-bullet tickets with dependency edges and execution waves → `tasks.md`. |
| `qa` | The quality gate: test plan, goal-backward verification, verification-gap / traceability / TDD-evidence / test-quality / security / code-quality audits → 4-state gate (PASS / CONCERNS / FAIL / WAIVED). |
| `ticketing` | Publish sliced tickets to the configured tracker (Backlog.md / gh / glab / jira); run the status machine (backlog → ready → in-progress → review → done); privacy review. |

## Discovery

| Skill | Role |
|-------|------|
| `interviewing` | Interview the user via a design-tree "frontier" in rounds until a weighted clarity gate passes; maintain glossary/ADRs as terms resolve. |
| `brainstorming` | Turn ideas into designs via 4 classified paths (spike / bounded / new-project / new-function) with a hard approval gate before implementation. Has a visual companion (local server + templates). |
| `spike` | Throwaway feasibility experiment producing an evidence-gated verdict (VALIDATED / INVALIDATED / PARTIAL), an investigation trail, and one liftable pure module. |
| `sketch` | Build throwaway interactive UI mockups (2–3 structurally different variants, tab-switchable) to get a felt verdict + constraints for the real build. |
| `research` | Lightweight single-pass research against primary sources with epistemics (research firewall) and type packs; writes a cited `research.md`. |
| `deep-research` | Heavy research: fan out parallel researcher subagents (researcher / verifier / red-team), verify load-bearing claims, red-team conclusions, synthesize a cited findings file. |

## Execution

| Skill | Role |
|-------|------|
| `simple-execution` | Inline plan execution: load/review plan, TDD per task, QA gate, request review, finish branch; stops and asks on blockers. |
| `subagent-execution` | Execute a plan with a fresh implementer subagent per task, per-task spec+quality review, 5-round fix loop with breaker/adjudication, final QA + whole-branch two-axis review; ledger-based. |
| `parallel-execution` | Dispatch one subagent per independent problem domain in the same message (parallel); integrate from a fan-out ledger. |
| `work-unit-commits` | Canonical commit protocol: conventional commits, atomic sizing, `[skillgrid-context]` block, git-hook guards, `checkpoint.json` resume handle. |
| `resume` | Re-orient from durable state (`state.md`, execution ledger, `checkpoint.json`, Mnemonic fallback) — files win over conversation memory; includes a context save/restore protocol. |

## Quality

| Skill | Role |
|-------|------|
| `test-driven-development` | Iron-law TDD: no production code without a failing test first; red → green → refactor. |
| `test-driven-verification` | The `#### Gates` contract: every requirement carries `G<n>` oracles (`CHECK:` + `EXPECT:`, `manual`, or `ABANDON` with reason); a gate is met only when freshly run. |
| `acceptance-test-authoring` | Author BDD acceptance suites: Gherkin-in-Markdown `acceptance.feature`, cucumber-js runner, extraction contract, page-object model, zone rule. BDD is always on. |
| `requesting-code-review` | Dispatch two parallel reviewer subagents on two separate axes — Standards (glossary/ADRs) and Spec (BDD scenarios) — then triage findings. |
| `receiving-code-review` | Handle review feedback with technical rigor (verify, push back, no performative agreement); own the triage → fix → validate loop (3-round cap). |
| `parallel-code-review` | Heavy review: fan out 6 specialist reviewers (standards, spec, edge cases, verification gaps, security, red team) in parallel, then triage / dedup / verdict. |

## Close-out (ship → reflect)

The tail after review: `qa → review → ship → reflect`.

| Skill | Role |
|-------|------|
| `ship` | Integrate the change to its base branch (merge / PR / keep — tests green on the integrated tree), then mechanically move the change folder `specs/` → `archive/` with a `diff -r` readback. Writes `ship-report.md`. No release mechanics, no docs check. |
| `reflect` | Terminal phase: sourced retrospective (Decisions / Lessons / Patterns / Surprises) + acceptance verdict (advisory) + final-state `archive-report.md` with observation-ID lineage; owns the Mnemonic session close. |

## Cross-cutting

| Skill | Role |
|-------|------|
| `mnemonic` | Operate Skillgrid's persistent memory (SQLite + FTS5): mem_save/recall protocol, code-index ladder, web cache, session summary rules. |
| `isolated-workspace` | Ensure an isolated workspace (native worktree tools first, git worktree fallback); detect existing isolation; run setup + baseline tests. |
| `structured-debugging` | Root-cause-first debugging: 4 phases (investigate → pattern → hypothesis → implement); 3 fixes fail → question architecture; debug state file survives compaction. |
| `architectural-decision-records` | Build/sharpen the domain model (glossary `business.md` + `technical.md`) and record ADRs (5 styles, immutable once accepted, per-change ADR Review Manifest). |
| `ponytail` | Enforce the laziest working solution via a 7-rung ladder (YAGNI → codebase → stdlib → native → installed dep → one-liner → minimal); always-on by default. |

## Frontmatter note

28 of 30 skills carry YAML frontmatter with `name` + `description`. Two are exceptions — `acceptance-test-authoring` and `mnemonic` start with a prose `#` header and no frontmatter. If a skill loader keys off frontmatter, add a `name`/`description` block to these two.

## Where skills come from

AISkillGrid unifies stage and general skills inspired by superpowers, intent-driven-template, mattpocock, gentleman-ai, BMAD, and gsd-core. See [Start here](00-start-here.md#sources-of-logic).

## Next step

[Workflow usage](03-workflow-usage.md) — the day-to-day pipeline.
