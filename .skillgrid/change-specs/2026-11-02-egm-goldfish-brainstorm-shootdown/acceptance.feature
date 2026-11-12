Feature: EGM Goldfish - shootdown, plan review, drafting disciplines, drift report, verification consolidation

# Track A — Brainstorm Shootdown

Scenario: happy path six-lens-shootdown
  Given the six-lens shootdown runs for a New Project or New Function brainstorm
  When the six lens subagents are dispatched
  Then the reviewer prompt names the anchored hypothesis and the reading list
  And the prompt contains no summary of the brainstorming conversation
  And the subagent is a freshly spawned one (not the session's own context)
  And all six are dispatched in parallel (not sequentially)

Scenario: happy path shootdown-path-scope
  Given a repo state
  When the check runs
  Then `SKILL.md`'s New Project and New Function path descriptions name the shootdown
  And the Bounded and Prototype descriptions do not
  And `references/paths.md`'s checklists reflect the insertion point

Scenario: happy path blocker-disposition-gate
  Given a brainstorm whose shootdown findings await disposition
  When the spec is ready to be written
  Then a blueprint with an undispositioned 🔴 Blocker finding does not have a committed `briefing.md`
  And a blueprint with all 🔴 Blockers dispositioned has a `## Shootdown Findings` section in `briefing.md` with the severity table and dispositions

Scenario: happy path shootdown-findings-in-briefing
  Given a repo state
  When the check runs
  Then `templates/briefing.md` contains the `## Shootdown Findings` section with the three sub-parts
  And a filled-in `briefing.md` for a New Project / New Function change has the section populated (or an explicit "Skipped: bounded/prototype path" note for those paths)

Scenario: happy path anti-sycophancy-rules
  Given a repo state
  When the check runs
  Then `SKILL.md` contains the Anti-Sycophancy & Stress-Testing section
  And every rule is cited from `review-discipline.md` (not restated)
  And the banned-adjective list is in exactly one place (`review-discipline.md`)

# Track B — Goldfish Review Skill

Scenario: happy path goldfish-review-reference-set
  Given a repo state
  When the check runs
  Then `writing-blueprints/references/goldfish-review.md` exists with the 5-stage pipeline + 4 presets
  And `references/goldfish-review-loop.md` exists with convergence rules
  And `references/goldfish-resolve.md` exists with categorization rules + interactive gate
  And `references/harnesses.md` exists
  And the reference set is runnable standalone on a named design doc
  And no new top-level skill is created

Scenario: happy path five-stage-pipeline
  Given a repo state
  When the check runs
  Then `writing-blueprints/references/goldfish-review.md` names all five stages and their targets
  And the stage list is not a single combined prompt
  And Stage 8 re-runs zoom-out on the updated doc after Stage 7 remediation

Scenario: happy path four-presets
  Given a repo state
  When the check runs
  Then `writing-blueprints/references/goldfish-review.md` documents all 4 presets
  And the preset gate is single-select
  And the topological order is preserved

Scenario: happy path sharpening-loop-convergence
  Given a repo state
  When the check runs
  Then `references/goldfish-review-loop.md` documents all 6 rules
  And the loop does not auto-advance (turn-taking)
  And convergence requires 2 clean rounds
  And settled findings are not re-opened

Scenario: happy path elephant-resolution
  Given a repo state
  When the check runs
  Then `references/goldfish-resolve.md` documents the 4 categories + 4 rules + 5 gate options
  And the resolution runs in the main agent context (not a subagent)
  And the ledger round entry carries the per-finding report (A1)

Scenario: happy path persona-briefs-present
  Given a repo state
  When the check runs
  Then all 13 persona brief files exist
  And each is ≤150 lines
  And each names its audit vectors (or attack question, for lens briefs)
  And its output format matches the finding schema
  And no `synthesizer.md` exists
  And no `brainstorming/agents/` directory exists

Scenario: happy path review-discipline-cited
  Given a repo state
  When the check runs
  Then `review-discipline.md` exists and is cited by every file in `_shared/elephant-goldfish/` AND by `writing-blueprints/references/goldfish-review.md`
  And `prose-style.md` contains a "No flattery" section
  And the banned-adjective list is in exactly one place (`review-discipline.md`)

Scenario: happy path review-artifacts-committed
  Given a repo state after a Goldfish review of a ≥3-task blueprint and a Track A shootdown
  When the check runs
  Then the stage report files exist (or a `not run` marker for a failed stage) and a `review-ledger.md` round entry exists, both committed before `acceptance.feature` is written
  And `briefing.md`'s `## Shootdown Findings` section is populated and committed
  And after any review run, the chat turn ends with the consolidated response (status + severity + Blocker/Major findings + report path)
  And a turn that ends with only a file path does not satisfy this requirement

Scenario: happy path writing-blueprints-invokes-goldfish-review
  Given a repo state
  When the check runs
  Then `writing-blueprints/SKILL.md`'s validation block names the Goldfish review reference set (`references/goldfish-review.md`)
  And `references/plan-review.md` is deleted or marked superseded
  And the invocation passes the correct TargetFile and GoldfishContext

Scenario: happy path skill-size-within-tier
  Given a repo state
  When the check runs
  Then `node .agents/skills/qa/scripts/skill-size-budget.mjs check` exits 0 for `brainstorming`, `writing-blueprints`, `qa`, and `code-review` (Track E)
  And the SKILL.md diffs are net-neutral or small

Scenario: happy path after-code-untouched
  Given the change's committed file set
  When the check runs
  Then `git diff --name-only` for the change lists only the files in R16's scope (including Track E's R39–R43 file set)
  And `sdd-structure.md` receives only R43 citation updates (no phase or resume marker change)
  And the `qa`/`receiving-code-review` diffs are additive one-liners or short sections (no behavior change to existing paths)
  And `code-review`'s output contract matches the old skills' (`review.md`, `REVIEW-PASS`/`BACK-TO-APPLY` counts, ledger round entries)

Scenario: happy path parked-finding-surfaced-at-qa
  Given a blueprint with one parked CRITICAL finding
  When `qa` runs
  Then it shows a WARNING naming the parked finding and its path
  And the gate verdict is otherwise unaffected (WARNING, never CRITICAL)

# Track C — Drafting Disciplines

Scenario: happy path ingestion-confirmation
  Given a writing-blueprints run on a change with a briefing
  When the ingestion step runs before drafting
  Then `SKILL.md` contains an "Ingest Predecessor Context" section
  And the agent presents a summary of the briefing's key decisions before drafting the first section
  And the user confirms before drafting proceeds

Scenario: happy path section-by-section-drafting
  Given a repo state
  When the check runs
  Then `SKILL.md` contains a "Section-by-Section Drafting" section that names `references/section-by-section.md`
  And the agent writes each section to `blueprint.md` and presents the approval gate before proceeding to the next
  And the chat summary precedes the gate (no empty chat bubbles)

Scenario: happy path per-task-gate
  Given a repo state
  When the check runs
  Then the agent does not write Task N+1 until Task N is approved
  And the per-task gate is present in `references/section-by-section.md`
  And the task template is structurally unchanged

Scenario: happy path spec-reconciliation
  Given a repo state
  When the check runs
  Then `SKILL.md`'s Self-Review section contains the Spec Reconciliation check
  And a blueprint with a contradiction vs the briefing reports it (quoting both)
  And the agent does not silently change the blueprint to match the briefing

Scenario: happy path owner-decides-architecture
  Given a repo state
  When the check runs
  Then the Owed-Decision Gate's option presentation does not contain "recommended" or a default marker for architectural choices
  And each option has mechanism + trade-offs + failure conditions
  And mechanical choices are decided by the agent without a gate

Scenario: happy path alternatives-from-briefing
  Given a repo state
  When the check runs
  Then `SKILL.md` contains the alternatives-from-briefing instruction
  And a blueprint with rejected alternatives in the briefing notes them
  And a blueprint with no alternatives in the briefing notes "No alternatives recorded in briefing."

Scenario: happy path drafting-anti-sycophancy
  Given a repo state
  When the check runs
  Then `SKILL.md` contains the Tone & Discipline section
  And every rule is cited from `review-discipline.md` (not restated)
  And the banned-adjective list is in exactly one place

Scenario: happy path artifact-first-delivery
  Given a repo state
  When the check runs
  Then `SKILL.md` contains the artifact-first delivery rule
  And the agent writes to `blueprint.md` and gives the path rather than printing the content in chat

# Track D — Drift Report

Scenario: happy path drift-report-skill
  Given a repo state
  When the check runs
  Then `qa/references/drift-report.md` exists with the drift detection protocol
  And `qa/references/divergence-categories.md` exists with the 4 categories
  And `qa/references/reconciliation.md` exists with the two-way protocol
  And `qa/SKILL.md` contains the Drift mode section
  And the mode is runnable on a spec dir or project-wide
  And no new top-level skill is created

Scenario: happy path four-divergence-categories
  Given a repo state
  When the check runs
  Then `references/divergence-categories.md` defines all 4 categories with at least one example each
  And the drift report classifies every finding into exactly one category

Scenario: happy path target-artifacts
  Given a repo state
  When the check runs
  Then drift mode on a spec dir reads the 4 spec artifacts
  And project-wide mode scans all spec dirs + AGENTS.md + in-force ADRs + projectwiki pages
  And the report is scoped to the target set

Scenario: happy path sequential-source-grounding
  Given a repo state
  When the check runs
  Then spec dirs are processed sequentially (not in parallel)
  And code search is used to resolve references
  And unresolvable references are reported as findings + a prompt to add links

Scenario: happy path difference-filtering
  Given a repo state
  When the check runs
  Then a clean spec dir produces "No drift detected"
  And a drifted spec dir produces a bulleted list of specific divergences with file:line references
  And no "what matches" descriptions

Scenario: happy path drift-report-artifact
  Given a repo state
  When the check runs
  Then the report file exists at the correct path
  And each finding has a category + file:line reference
  And a clean report says "No drift detected."

Scenario: happy path two-way-reconciliation
  Given a repo state
  When the check runs
  Then `references/reconciliation.md` documents both paths
  And each finding in the report has a disposition
  And the user decides the path per finding

Scenario: happy path drift-reuses-resolution
  Given a repo state
  When the check runs
  Then the drift mode's findings use the R10 categorization (CONTEST/ESCALATE/PROPOSE-FIX/DECLINE)
  And the `codebase-details` persona is the primary reviewer
  And the interactive gate matches R10

Scenario: happy path reflect-drift-check
  Given a repo state
  When the check runs
  Then `reflect/SKILL.md`'s checklist contains the drift check item (pointing at `qa`'s drift mode)
  And it is marked advisory (not blocking)
  And the reflect report notes drift findings if present

Scenario: happy path drift-skill-size
  Given a repo state
  When the check runs
  Then `node .agents/skills/qa/scripts/skill-size-budget.mjs check` exits 0 for `qa`
  And the Drift mode section is < 40 lines

# Cross-track — Goldfish Isolation & Harness Portability

Scenario: happy path no-subagent-stop
  Given a repo state
  When the check runs
  Then `brainstorming/references/shootdown.md`, `writing-blueprints/references/goldfish-review.md`, and `qa/references/drift-report.md` each contain the stop rule verbatim (or its normative equivalent)
  And a same-context run recorded as "review complete" in any of the three tracks is a defect

Scenario: happy path goldfish-isolation-contract
  Given a repo state
  When the check runs
  Then each persona brief in `_shared/elephant-goldfish/` has a `## Isolation mandate` naming all four parts
  And the dispatch prompt in `brainstorming/references/shootdown.md`, `writing-blueprints/references/goldfish-review.md`, and `qa/references/drift-report.md` re-asserts the four parts verbatim
  And the review report format (R10, R13) includes a one-line harness-isolation attestation (which of the four parts the harness enforced vs. could not)

Scenario: happy path harness-portability
  Given a repo state
  When the check runs
  Then `writing-blueprints/references/harnesses.md` exists with a table covering Claude Code, OpenCode, Cursor, Pi (subagent tool, read-only mechanism, project-instruction exclusion mechanism, partial-enforcement note where applicable)
  And each skill's dispatch section names the protocol, not a specific harness
  And no harness-specific string (`Agent`, `invoke_subagent`, `omitClaudeMd`) appears in `SKILL.md` or persona briefs — only in `references/harnesses.md` and the adapter files
  And if the OpenCode adapter is in scope, it loads and strips instruction sources from a Goldfish child session in a smoke test, and the attestation reads "stripped by goldfish plugin, provider base prompt remains"
  And if deferred, the OpenCode attestation reads "harness could not exclude project instructions — persona mandate only" until the plugin lands

# Track E — Verification Skill Consolidation

Scenario: happy path code-review-topology
  Given a repo state
  When the check runs
  Then `.agents/skills/code-review/SKILL.md` exists
  And the two old skill directories are deleted
  And the topology step reproduces both old selection behaviors (small diff → 3-axis; large/high-risk → fan-out + conditional specialists)
  And the Blindness Contract + grades + verdict table each appear exactly once
  And `ship`'s review-signal path works against the carried-over output

Scenario: happy path security-template-merge
  Given a repo state
  When the check runs
  Then one security template exists under `code-review/reviewers/`
  And it contains the 5-step workflow + rubric + report format
  And both topologies reference it

Scenario: happy path ledger-as-reference
  Given a repo state
  When the check runs
  Then `_shared/craft/review-ledger.md` exists with the full contract
  And the `review-ledger/` skill directory is deleted
  And `code-review` and `writing-blueprints/references/goldfish-resolve.md` cite it

Scenario: happy path qa-debt-surfacing
  Given a repo state
  When the check runs
  Then `qa/SKILL.md` contains the debt-surfacing step
  And no `qa` file cites a deleted skill name
  And the canon exists in one `_shared/verification/` file

Scenario: happy path review-consolidation-cutover
  Given a repo state
  When the check runs
  Then zero surviving `skillgrid:` citations to `requesting-code-review`, `parallel-code-review`, or `review-ledger` exist anywhere in the repo (outside `.skillgrid/change-archive/`)
  And the three old skill directories do not exist
  And `node .agents/skills/qa/scripts/skill-size-budget.mjs check` exits 0
