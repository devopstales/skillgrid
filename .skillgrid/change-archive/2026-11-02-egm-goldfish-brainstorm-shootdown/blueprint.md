# EGM Goldfish Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-execution (recommended) or simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2 (config `rules.tiers.default`; the spec carries no `Tier:` line — noted as an assumption in Global Constraints)

**Build shape:** Tracer thread — the tracer is the shared EGM infrastructure + the merged `code-review` skill (Tasks 3–16): the load-bearing vertical slice every other track cites. A full track (A) lands end-to-end next; B, C, D expand.

**Goal:** Port the Elephant-Goldfish Model onto skillgrid as four review/drafting tracks (brainstorm shootdown, plan review, drafting disciplines, drift report) plus one verification-skill consolidation, all as Markdown/JSON skill artifacts.

**Architecture:** Five ordered track sections in one blueprint: shared EGM persona-brief + discipline infrastructure → Track E (merge `requesting-code-review` + `parallel-code-review` into `code-review`, demote `review-ledger` to a shared reference, hoist the verification-gap canon, qa carries debt surfacing) → Track A (brainstorming gains the 6-lens fresh-context shootdown) → Track B (writing-blueprints gains the `goldfish-review/` reference set, replacing `plan-review.md`) → Track C (section-by-section drafting + 8 disciplines) → Track D (qa gains advisory drift mode) → global citation/ADR reconciliation. No code; verification is structural (grep, `wc -l`, the size-budget script; gherkin lint once the BDD runner is installed — it is not in this environment).

**Tech Stack:** Markdown skill files, `_shared/skill-size-budget.json`, git, `node` (size-budget script). BDD: `acceptance.feature` per config `bdd:` — verified by structural checks in this environment (the `acceptance-tests/` cucumber runner is not installed here — see Task 1 Step 3 note).

**Spec:** `.skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/briefing.md` — Amendments A1 + A2 applied; requirements R1–R43. Every task cites the R-IDs it satisfies; the spec is the content source (the engineer reads spec + blueprint; the blueprint pins files, order, and verification).

**Findings:** None — this change ran no research/prototype/sketch.

## Hypothesis

- **Right conditions:** after Task 32, the repo has exactly one review skill (`code-review`) reproducing both old selection behaviors; 13 EGM persona briefs each carrying the isolation + audit-vector + output-format contract; brainstorming gates spec-writing on a dispositioned 6-lens shootdown; writing-blueprints validates via the 5-stage goldfish pipeline with a 2-clean-rounds convergence cap and drafts section-by-section with per-section gates; qa reports drift advisory-only; ADRs 0062–0065 are LOCKED (0063 superseding 0054); zero live citations of the three deleted skill names; the size-budget script exits 0; `go build ./...` / `go test ./...` unchanged (docs-only).
- **Wrong conditions:** any live (non-historical) citation of `requesting-code-review`, `parallel-code-review`, or the `review-ledger` skill; drift in the `review.md` / ledger-round / `REVIEW-PASS`-`BACK-TO-APPLY` contract that `ship` consumes; a persona brief over 150 lines or missing a mandated section; the qa Drift-mode section ≥ 40 lines; the banned-adjective list present in more than one file.
- **Thinnest check:** Task 33's verification run.
- **Door check:** D1–D3 below.

## Terms

None beyond the spec. Change-local terms are defined in the briefing's `## Terms` (Goldfish shootdown, Goldfish review, drafting disciplines, drift report, Elephant, Goldfish isolation, Blindness Contract) and are used here with the same meaning.

## Must-Haves (goal-backward verification)

**Truths** (observable outcomes the verifier checks):
1. `.agents/skills/code-review/SKILL.md` exists and its topology step reproduces both old selection behaviors: <50 changed lines/low-risk → 3-axis (Standards + Spec + Security); ≥50 lines or high-risk → core five (Standards, Spec, Edge cases, Verification gaps, Security) + conditional Accessibility/Performance; red team always last (R39).
2. The Blindness Contract (no-leaks + grades A/B/C, grade-C-diagnostic-only), the six-value verdict table, and the resilience rule each appear exactly once in the repo's review surface (R39).
3. `requesting-code-review/`, `parallel-code-review/`, `review-ledger/` skill directories are deleted; zero live (non-historical) citations anywhere in the repo (R41, R43).
4. The output contract is preserved byte-structurally: `code-review/templates/review.md` keeps the section shape `ship` parses; `REVIEW-PASS`/`BACK-TO-APPLY` + Critical/Warn counts remain; the per-change `review-ledger.md` artifact format is unchanged (R39, R41, R42).
5. `_shared/craft/review-ledger.md` carries the full ledger contract (round format, frozen candidate + invalidation, dispositions fixed/parked/waived, lineage, debt lifecycle, 3-changes-stale flag); `code-review`, `goldfish-resolve.md`, and `qa` cite it (R41, R42).
6. 13 persona briefs exist in `_shared/elephant-goldfish/` (7 review + 6 lens), each ≤150 lines with `## Isolation mandate`, `## Audit vectors`, `## Output format` (R11, R1).
7. The 4-part Goldfish isolation contract (fresh context, read-only, no project instructions, no network/MCP) is written once in shared references; part 1 cites the code-review Blindness Contract; the no-subagent-tool rule is STOP, not degrade (R36, R37).
8. brainstorming's New Project / New Function paths dispatch 6 fresh lens subagents after the feasibility check and before `briefing.md` is written; `templates/briefing.md` has `## Shootdown Findings`; every blocker is dispositioned (resolved / accepted-with-note / downgraded-to-non-goal) before the spec is written (R1–R4).
9. writing-blueprints' validation seam dispatches the 5-stage goldfish pipeline (4 single-select presets, topological order preserved); convergence = 2 consecutive clean rounds, cap 5; turn-taking, no auto-advance; `plan-review.md` is gone (R6–R10).
10. Goldfish findings use reviewer vocabulary in transit and are normalized to CRITICAL/WARNING/SUGGESTION exactly once, at the record boundary (ledger entry); a parked CRITICAL surfaces as a WARNING at qa time via the debt-surfacing step (R7, R17).
11. writing-blueprints drafts section-by-section with per-section approval gates, a per-task gate, and the 8 drafting disciplines; an explicit 5-step ingestion confirmation (read briefing/findings/adr, summarize, confirm) precedes any drafting (R18–R25).
12. qa's Drift mode is <40 lines in SKILL.md, advisory (never the 4-state gate), citing 3 new references; `skill-size-budget.mjs` exits 0 for the drift surface; reflect's checklist points at it (R26–R35).
13. ADRs 0062 (shootdown), 0063 (plan review, supersedes 0054), 0064 (drafting disciplines), 0065 (verification-skill consolidation) exist and are LOCKED in ASSUMPTIONS.md; the 0054 row is marked superseded.
14. The banned-adjective list lives in exactly one file: `_shared/craft/review-discipline.md` (R12).

**Artifacts** (files that must exist with real content, not stubs):
- `.agents/skills/code-review/SKILL.md` — topology + shared protocol + dispatch + output contract
- `.agents/skills/code-review/reviewers/{code-reviewer,spec-reviewer,edge-case-hunter,verification-gap,security,accessibility,performance,red-team}.md` — 8 reviewer templates (security merged per R40)
- `.agents/skills/code-review/templates/review.md` — carried-over audit record
- `_shared/craft/review-ledger.md`, `_shared/craft/review-discipline.md`, `_shared/verification/verification-gap.md`
- `_shared/elephant-goldfish/` — 13 persona briefs
- `.agents/skills/brainstorming/references/shootdown.md`; edits to `brainstorming/SKILL.md`, `paths.md`, `templates/briefing.md`
- `.agents/skills/writing-blueprints/references/goldfish-{review,review-loop,resolve,harnesses}.md`, `references/section-by-section.md`; edits to `writing-blueprints/SKILL.md`, `templates/blueprint.md`; deletion of `references/plan-review.md`
- `.agents/skills/qa/references/{drift-report,divergence-categories,reconciliation}.md`; edits to `qa/SKILL.md`, `reflect/SKILL.md`
- `.skillgrid/projectwiki/decisions/04-adr-00{62,63,64,65}-*.md`; ASSUMPTIONS.md LOCKED rows
- `.skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/acceptance.feature`

**Key links** (connections that must work together):
- code-review's Blindness Contract ← cited by every persona brief's `## Isolation mandate` (part 1), by `goldfish-review.md` dispatch, and by `harnesses.md`.
- `_shared/craft/review-ledger.md` ← cited by `code-review/SKILL.md` (round entries), `goldfish-resolve.md` (Goldfish rounds), `qa/SKILL.md` (debt-surfacing step), `drift-report.md` (finding rounds).
- Persona briefs ← dispatched by `shootdown.md` (6 lenses) and `goldfish-review.md` (7 review briefs).
- `review-discipline.md` ← cited by brainstorming's Anti-Sycophancy section, the goldfish references, and prose-style's No-flattery rule — never restated.
- `verification-gap.md` ← cited by qa's audit and the code-review verification-gap reviewer (single home).
- qa's debt-surfacing step is where R17 parked Goldfish findings surface at qa time.

**One-way-door decisions** (flag for explicit user approval before implementing):
- **D1 (Task 16):** delete the three skill directories (`requesting-code-review`, `parallel-code-review`, `review-ledger`) after ~15 citation sites migrate. Reversible by git revert, but every referring skill breaks until the citation migration is complete — the deletion and the migration must land in the same task sequence.
- **D2 (Task 24):** `plan-review.md` is replaced by the `goldfish-review/` reference set — writing-blueprints' validation seam changes from a 6-check inline review to a 5-stage fresh-context pipeline.
- **D3 (Task 11):** `review-ledger` stops being an invocable skill; its contract moves to `_shared/craft/review-ledger.md` and its Step 5 (debt surfacing) moves into `qa`. The `skillgrid:review-ledger` invocation form disappears.

## Global Constraints

- Flat skill layout: skills live at `.agents/skills/<name>/`; the only skill-local subdirectories are `references/` and `templates/`. No new top-level skill except `code-review` (A1 consolidated the goldfish tracks into existing skills; A2 created `code-review`).
- Persona briefs ≤150 lines each (R11).
- qa SKILL.md Drift-mode section <40 lines (R35).
- Banned-adjective list in exactly one place: `_shared/craft/review-discipline.md` (R12).
- Severity normalization happens exactly once, at the record boundary: raw findings carry reviewer vocabulary; ledger entries carry CRITICAL/WARNING/SUGGESTION (R7).
- After-code-half semantics are untouched (R16): qa's 4-state gate, `ship`'s REVIEW-PASS/BACK-TO-APPLY, the `review.md` shape, ledger round entries — preserved.
- No alias skills (R43): no redirect/stub left behind the deleted names.
- Conventional commits only; spec-zone artifacts (acceptance.feature, ADRs, ASSUMPTIONS) commit before code-zone skill edits (pre-commit zone guard).
- No new third-party dependencies; prose + one JSON file.
- Tier T2 assumed from config default (spec has no `Tier:` line).

## File Structure

**New — shared:**
- `_shared/elephant-goldfish/` — 13 persona briefs (7 review: `zoom-out`, `comprehension`, `design-critic`, `implementation-readiness`, `codebase-details`, `structural`, `style`; 6 lens: `lens-contrarian-yagni`, `lens-cost-economics`, `lens-ux-ergonomics`, `lens-migration-compat`, `lens-observability-sre`, `lens-tech-debt`)
- `_shared/craft/review-discipline.md` — anti-sycophancy + stress-testing rules; single home of the banned-adjective list
- `_shared/craft/review-ledger.md` — ledger contract (demoted from skill)
- `_shared/verification/verification-gap.md` — three-gap-shape canon (hoisted from `qa/references/`)

**New — code-review skill:**
- `.agents/skills/code-review/SKILL.md` — topology step, shared protocol (once), dispatch, output contract
- `.agents/skills/code-review/reviewers/code-reviewer.md` — core: standards axis (carried from requesting-code-review)
- `.agents/skills/code-review/reviewers/spec-reviewer.md` — core: spec-compliance axis (carried)
- `.agents/skills/code-review/reviewers/edge-case-hunter.md` — core: edge-case hunter (carried from parallel-code-review)
- `.agents/skills/code-review/reviewers/verification-gap.md` — core: verification gaps (carried; cites the hoisted canon)
- `.agents/skills/code-review/reviewers/security.md` — core: merged OWASP 5-step + four-part triage rubric + report format (R40)
- `.agents/skills/code-review/reviewers/accessibility.md` — conditional: diff touches UI
- `.agents/skills/code-review/reviewers/performance.md` — conditional: diff touches data access/rendering
- `.agents/skills/code-review/reviewers/red-team.md` — always last; reads merged findings
- `.agents/skills/code-review/templates/review.md` — carried-over audit record

**New — brainstorming (Track A):**
- `.agents/skills/brainstorming/references/shootdown.md` — 6-lens shootdown workflow

**New — writing-blueprints (Tracks B + C):**
- `references/goldfish-review.md` — 5-stage pipeline + 4 presets + dispatch + report path
- `references/goldfish-review-loop.md` — sharpening loop + convergence + churn + turn-taking
- `references/goldfish-resolve.md` — Elephant-side resolution (CONTEST/ESCALATE/PROPOSE-FIX/DECLINE)
- `references/harnesses.md` — R36 STOP rule + R38 harness portability table
- `references/section-by-section.md` — ingestion, section order, gates, 8 disciplines

**New — qa (Track D):**
- `references/drift-report.md` — drift workflow + artifact + size rule
- `references/divergence-categories.md` — 4 categories + heuristics
- `references/reconciliation.md` — two-way reconciliation + dispositions (report format is R31's, owned by drift-report.md)

**Modified:**
- `.agents/skills/brainstorming/SKILL.md` (shootdown step + Anti-Sycophancy section), `paths.md`, `templates/briefing.md`
- `.agents/skills/writing-blueprints/SKILL.md` (Ingest Predecessor Context, Section-by-Section Drafting, validation rewrite, Spec Reconciliation check, Owed-Decision Gate no-rank presentation, Tone & Discipline, artifact-first rule), `templates/blueprint.md` (Review Summary section + Self-Review reminder comment)
- `.agents/skills/qa/SKILL.md` (debt-surfacing step, routing, Drift-mode section)
- `.agents/skills/reflect/SKILL.md` (checklist item)
- `.agents/skills/_shared/rules/prose-style.md` (No-flattery rule)
- `_shared/planning/rigor-tiers.md`, `_shared/verification/floor.md`, `_shared/rules/code-standards.md`, `_shared/rules/sdd-structure.md`, `_shared/craft/deterministic-boundary.md`, `_shared/craft/measurement.md`, `_shared/agent-config/block.md` (citation migration)
- `.agents/skills/{ship,subagent-execution,simple-execution,simplify-code,style-review,dogfood,deep-research}/SKILL.md` (+ `subagent-execution/references/final-review.md`) (citation migration)
- `.agents/skills/receiving-code-review/SKILL.md` (A1 light edit: cites `review-discipline.md`)
- `_shared/skill-size-budget.json` (remove 3 names, add `code-review`)
- `AGENTS.md` (workflow line)
- `README.md`, `CONTRIBUTING.md`, `docs/tutorials/your-first-change.md`, `docs/user-guide/{01-layout,02-skills,03-workflow-usage,06-multi-agent-work,08-guiding-principles}.md` (citation migration, Task 13)
- `.skillgrid/projectwiki/ARCHITECTURE.md`, `.skillgrid/projectwiki/workflows/development-workflow.md`, `.skillgrid/projectwiki/workflows/sdd-pipeline.md` (citation migration, Task 13)
- `.skillgrid/projectwiki/ASSUMPTIONS.md` (LOCKED rows; 0054 superseded)

**Deleted:**
- `.agents/skills/requesting-code-review/`, `.agents/skills/parallel-code-review/`, `.agents/skills/review-ledger/`
- `.agents/skills/writing-blueprints/references/plan-review.md`

---

### Task 1: Materialize `acceptance.feature` (spec zone)

**Files:**
- Create: `.skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/acceptance.feature`

**Interfaces:**
- Consumes: the spec's `Acceptance scenario:` lines (the spec names one scenario per requirement)
- Produces: the scenario names every later task's `SATISFIES` line references
- Deletion test: n/a (spec-zone input for Tasks 3–32)

- [ ] **Step 1: Extract the scenario list**

Run: `grep -n "Acceptance scenario:" .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/briefing.md`
Expected: one line per named scenario (`happy path code-review-topology`, `happy path security-template-merge`, `happy path ledger-as-reference`, `happy path qa-debt-surfacing`, `happy path harness-portability`, etc.) — each carries the requirement number and the scenario name.

- [ ] **Step 2: Write the feature file**

One `Feature:` (the change name) with one `Scenario:` per extracted line. Each scenario's Given/When/Then is the corresponding requirement's `Acceptance:` text from the spec, split into steps (no invented steps — the spec's acceptance text is the scenario body). Scenarios for structural checks (file exists, line budgets, grep-zero) are written as Given a repo state / When the check runs / Then the expected output holds.

- [ ] **Step 3: Verify structure (the repo's cucumber runner is not installed — note below)**

Run:
```bash
cd .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown
n=$(grep -c "Acceptance scenario:" briefing.md)
s=$(grep -cE "^Scenario:" acceptance.feature)
steps=$(grep -cE "^  (Given|When|Then|And) " acceptance.feature)
echo "named=$n scenarios=$s steps=$steps"
```
Expected: `scenarios` equals `named` (one `Scenario:` per spec-named scenario); `steps` > 0.

**Environment note (verified 2026-11-11):** the BDD runner is not installed here — `acceptance-tests/` is empty (no `node_modules`), a global `cucumber` 6.0.7 package shadows the `cucumber-js` binary, and `gherkin-lint` exits 1 (no `.gherkin-lintrc` anywhere in the repo). Once the runner is installed, run the feature under `cucumber-js` for a full parse + step check. For this docs-only change the scenarios are structural assertions executed by Task 33, so the structural check above is the gate for this change.

- [ ] **Step 4: Commit (spec zone — before any skill-file edit)**

```bash
git add .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/acceptance.feature
git commit -m "test: materialize acceptance scenarios for egm-goldfish change"
```

**SATISFIES:** every scenario materialized here (they are the BDD surface for Tasks 3–32).

### Task 2: ADRs 0062–0065 + ASSUMPTIONS LOCKED rows (spec zone)

**Files:**
- Create: `.skillgrid/projectwiki/decisions/04-adr-0062-goldfish-brainstorm-shootdown.md`
- Create: `.skillgrid/projectwiki/decisions/04-adr-0063-goldfish-plan-review.md`
- Create: `.skillgrid/projectwiki/decisions/04-adr-0064-goldfish-drafting-disciplines.md`
- Create: `.skillgrid/projectwiki/decisions/04-adr-0065-verification-skill-consolidation.md`
- Modify: `.skillgrid/projectwiki/ASSUMPTIONS.md` (§ LOCKED)

**Interfaces:**
- Consumes: the spec's `## Decisions (ADR)` section (one decision per track: shootdown, plan review, drafting disciplines, consolidation) + the existing ADR format (follow `04-adr-0060-wiki-autolearning-structure.md`: Status / Context / Decision / Consequences)
- Produces: the in-force ADR manifest the later tasks cite; 0063 marks 0054 superseded
- Deletion test: n/a

- [ ] **Step 1: Write the four ADR bodies**

Each ADR: the Decision line is the spec's `## Decisions (ADR)` one-liner verbatim (0062: before-design idea validation is a fresh-context 6-lens Goldfish shootdown subflow of brainstorming; 0063: before-code plan review is a fresh-context 5-stage multi-persona Goldfish reference set inside writing-blueprints, sharpening loop + Elephant-side resolution in `review-ledger`, after-code half unchanged — **supersedes ADR-0054**, carried as the ADR frontmatter `supersedes:` field + a `Supersedes:` line in the body; 0064: section-by-section drafting with per-section gates, owner-decides-architecture, blueprint is the single source of truth, drift is a mode of qa; 0065: single `code-review` skill with topology step, ledger is a shared reference, qa/receiving-code-review keep their roles, observable behavior unchanged). Context = the spec's track problem statement (condensed, citing R-IDs); Consequences = the spec's acceptance criteria + the one-way doors it carries.

- [ ] **Step 2: Update ASSUMPTIONS.md § LOCKED**

Add 4 rows (0062–0065, each: ADR id, one-line decision, link). Edit the existing 0054 row to mark it superseded by 0063 (keep the row — historical record).

- [ ] **Step 3: Verify**

Run: `ls .skillgrid/projectwiki/decisions/ | grep -E "00(62|63|64|65)"`
Expected: 4 files listed.
Run: `for n in 0062 0063 0064 0065; do grep -c "04-adr-$n" .skillgrid/projectwiki/ASSUMPTIONS.md; done`
Expected: ≥1 for each.
Run: `grep -A1 "04-adr-0054" .skillgrid/projectwiki/ASSUMPTIONS.md | grep -i supersed`
Expected: the 0054 row shows the supersession.

- [ ] **Step 4: Commit (spec zone)**

```bash
git add .skillgrid/projectwiki/decisions/ .skillgrid/projectwiki/ASSUMPTIONS.md
git commit -m "docs(adr): 0062-0065 egm tracks + verification consolidation (0063 supersedes 0054)"
```

**SATISFIES:** structural check — Must-Have truth 13 (scenarios materialized for it in Task 1).

### Task 3: `_shared/craft/review-discipline.md` (shared, R5/R12)

**Files:**
- Create: `.agents/skills/_shared/craft/review-discipline.md`

**Interfaces:**
- Consumes: spec §Track A R5 (anti-sycophancy & stress-testing rules apply to the shootdown) + the EGM source's review-discipline content
- Produces: the single home of the banned-adjective list + the anti-sycophancy/stress-testing rules, cited by brainstorming (Task 19), the goldfish references (Tasks 21–23), and prose-style (Task 6)
- Deletion test: every citing skill would restate the rules privately again — the duplication R17 forbids

- [ ] **Step 1: Write the file**

Sections: `## Anti-sycophancy rules` (the full rule set from spec R5/R12: default to stress-testing, critique first, no glazing, no apologies, generate counter-arguments, reject echo-chamber framing), `## Plain-English standard` (no ungrounded promotional adjectives, no decorative adverbs — the normative home of the lists), `## Banned vocabulary` (banned validating adjectives + banned decorative adverbs — this file is the only place in the repo where the lists appear), `## Cited by` (the citation contract: every file in `_shared/elephant-goldfish/`, `writing-blueprints/references/goldfish-review.md`, and brainstorming's Anti-Sycophancy section cite this file — R12's acceptance check, re-verified in Task 33). ≤150 lines.

- [ ] **Step 2: Verify single-home + budget**

Run: `wc -l .agents/skills/_shared/craft/review-discipline.md`
Expected: ≤150.
(Repo-wide single-home check runs in Task 33 — the other homes don't exist yet.)

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/_shared/craft/review-discipline.md
git commit -m "feat(skills): shared review-discipline (anti-sycophancy, banned adjectives)"
```

**SATISFIES:** `happy path anti-sycophancy-rules` (structural half — the rules' home exists).

### Task 4: 7 review persona briefs (shared, R11/R37)

**Files:**
- Create: `.agents/skills/_shared/elephant-goldfish/zoom-out.md`
- Create: `.agents/skills/_shared/elephant-goldfish/comprehension.md`
- Create: `.agents/skills/_shared/elephant-goldfish/design-critic.md`
- Create: `.agents/skills/_shared/elephant-goldfish/implementation-readiness.md`
- Create: `.agents/skills/_shared/elephant-goldfish/codebase-details.md`
- Create: `.agents/skills/_shared/elephant-goldfish/structural.md`
- Create: `.agents/skills/_shared/elephant-goldfish/style.md`

**Interfaces:**
- Consumes: the EGM source personas (`github.com/google/elephant-goldfish-model` — `zoom_out`, `comprehension`, `design_critic`, `implementation_readiness`, `codebase_details`, `structural`; `style` folded from EGM's review-discipline) + spec R6–R8 (stage roles, isolation, output format)
- Produces: the dispatchable briefs for Track B's 5-stage pipeline (Task 20) — one brief per stage/persona
- Deletion test: goldfish-review.md would need to inline all 7 persona prompts — the fan-out the shared directory exists to prevent

- [ ] **Step 1: Write the 7 briefs**

Each brief (R11 contract): `# <Role>` (the persona's one-paragraph job, ported from EGM, scoped to plan-review use), `## Isolation mandate` (R37's four parts: 1. fresh context — no conversation history, no author narrative; 2. read-only — no file edits, no shell, no writes; a reviewer that edits the target has not reviewed, its findings are discarded; 3. no project instructions — the reviewer sees ONLY the persona brief, the target file path(s), the explicit reading list, and the `Mode: BRAINSTORM|REVIEW` flag; 4. no network, no MCP — docs needed are saved as files and passed by path; part 1 cites `code-review`'s Blindness Contract + independence grades (R39) and replaces the grade-C-degrades fallback with R36's STOP rule), `## Audit vectors` (the spec's per-persona vector list for its stage — copied from spec §Track B), `## Output format` (the spec's finding schema verbatim: `Severity: Blocker|Major|Nit` / `Affected scope: <heading or line range>` / `Description` / `Rationale`; severities normalize to CRITICAL|WARNING|SUGGESTION only at the record boundary). Every brief carries one closing line: `Discipline: per `_shared/craft/review-discipline.md`.` (R12 — all 13 briefs, checked in Task 33). Stage mapping (R11): `style.md` is a standalone brief used by Stages 7 and 9 — not folded away. ≤150 lines each. No `synthesizer.md`.

- [ ] **Step 2: Verify shape + budget**

Run: `for f in .agents/skills/_shared/elephant-goldfish/{zoom-out,comprehension,design-critic,implementation-readiness,codebase-details,structural,style}.md; do wc -l "$f"; grep -c "^## " "$f"; done`
Expected: each file ≤150 lines; each contains all of `## Isolation mandate`, `## Audit vectors`, `## Output format` (verify: `grep -L "## Audit vectors" .agents/skills/_shared/elephant-goldfish/*.md` → empty).

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/_shared/elephant-goldfish/
git commit -m "feat(skills): 7 goldfish review persona briefs (EGM port)"
```

**SATISFIES:** the persona-shape scenario for R8 (Task 1).

### Task 5: 6 BRAINSTORM lens briefs (shared, R1/R11)

**Files:**
- Create: `.agents/skills/_shared/elephant-goldfish/lens-contrarian-yagni.md`
- Create: `.agents/skills/_shared/elephant-goldfish/lens-cost-economics.md`
- Create: `.agents/skills/_shared/elephant-goldfish/lens-ux-ergonomics.md`
- Create: `.agents/skills/_shared/elephant-goldfish/lens-migration-compat.md`
- Create: `.agents/skills/_shared/elephant-goldfish/lens-observability-sre.md`
- Create: `.agents/skills/_shared/elephant-goldfish/lens-tech-debt.md`

**Interfaces:**
- Consumes: EGM's dual-mode personas trimmed to BRAINSTORM mode (idea-validation, not plan-review) + spec §Track A R1–R4 (lens list, core questions, blocker-must-resolve gate)
- Produces: the dispatchable briefs for the brainstorming shootdown (Task 17)
- Deletion test: shootdown.md would inline 6 persona prompts — same duplication argument as Task 4

- [ ] **Step 1: Write the 6 briefs**

Each brief (R11: the Track A attack questions as their own rubrics — NOT the review personas): `# <Lens Role>` (the lens's idea-validation angle per spec R1), `## Attack question` (the spec's R1 core question for this lens, verbatim: (a) *Can existing code/libs solve 90% of this without new code?* (b) *Peak QPS, memory, storage, token budget at scale?* (c) *Is the CLI/API intuitive? Are errors actionable?* (d) *How do existing clients/data migrate? Schema breaks?* (e) *Can oncall triage this at 3am? Are metrics traceable?* (f) *Tightly-coupled state machines that hurt in 2 years?* — one per lens), `## Isolation mandate` (R37's four parts as in Task 4, scoped to the anchored hypothesis + the relevant existing code/files as the reading list — R1: the subagent gets ONLY the anchored hypothesis + relevant code, no conversation history, no author narrative; dispatched with `Mode: BRAINSTORM`), `## Blocker rule` (this lens may raise Blocker findings; the shootdown gate requires disposition before spec-writing — R1/R3), `## Output format` (same finding schema as Task 4), the same closing discipline-citation line as Task 4 (R12). ≤150 lines each.

- [ ] **Step 2: Verify shape + budget**

Run: `for f in .agents/skills/_shared/elephant-goldfish/lens-*.md; do wc -l "$f"; done; grep -L "## Blocker rule" .agents/skills/_shared/elephant-goldfish/lens-*.md`
Expected: all ≤150 lines; `grep -L` prints nothing.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/_shared/elephant-goldfish/lens-*.md
git commit -m "feat(skills): 6 brainstorm lens briefs (EGM BRAINSTORM-mode port)"
```

**SATISFIES:** the persona-shape scenario for R1 (Task 1).

### Task 6: prose-style — No-flattery rule (shared, R12)

**Files:**
- Modify: `.agents/skills/_shared/rules/prose-style.md` (append one rule to the rule list; the file is the 21-rule agent-style canon)

**Interfaces:**
- Consumes: Task 3's `review-discipline.md` (the rule cites it, never restates the banned list)
- Produces: the generation-time flattery ban that applies to all agent-written Markdown, including shootdown and review artifacts
- Deletion test: flattery reappears in generated prose with no generation-time rule to catch it

- [ ] **Step 1: Append the rule**

One rule entry in the existing list format: **No flattery.** Generated prose MUST NOT use complimentary or self-congratulatory language about the work, the author, or the reader (banned-adjective list per `_shared/craft/review-discipline.md` — cited, not restated).

- [ ] **Step 2: Verify**

Run: `grep -n "No flattery" .agents/skills/_shared/rules/prose-style.md`
Expected: 1 hit.
Run: `grep -c "review-discipline.md" .agents/skills/_shared/rules/prose-style.md`
Expected: ≥1.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/_shared/rules/prose-style.md
git commit -m "feat(skills): prose-style No-flattery rule (cites review-discipline)"
```

**SATISFIES:** `happy path review-discipline-cited` (Task 1) — the prose-style half.

### Task 7: `code-review/SKILL.md` — merged review skill (Track E, R39)

> ⚠ one-way: D1 — this task starts the consolidation that ends in the three-directory deletion (Task 16). The new skill must be complete before the old ones are removed.

**Files:**
- Create: `.agents/skills/code-review/SKILL.md`
- Read (sources, unmodified until Task 16): `.agents/skills/requesting-code-review/SKILL.md`, `.agents/skills/parallel-code-review/SKILL.md`

**Interfaces:**
- Consumes: both old skills' SKILL.md — selection rules carried verbatim (R39: "nothing is lost")
- Produces: the single review skill that Tasks 13–14 migrate citations to; the Blindness Contract home that Tasks 4/5 briefs and Task 21 cite
- Deletion test: both old skills would exist again, plus a chooser rule — the two-skill fork R39 eliminates

- [ ] **Step 1: Write the topology step**

The frontmatter (`name: code-review`; description: topology-driven code review — 3-axis for small/low-risk diffs, core-five fan-out + conditional specialists for large/high-risk, red team last). Then the topology section replacing the two-skill choice, with the spec's R39 selection rules verbatim:
- <50 changed lines, low risk → 3-axis pass (Standards + Spec + Security)
- ≥50 changed lines or high-risk (auth, data migration, money, concurrency, public API) → core five (Standards, Spec, Edge cases, Verification gaps, Security) + conditional Accessibility (diff touches UI) / Performance (diff touches data access/rendering)
- Red team always last (reads merged findings); tier floor per `_shared/planning/rigor-tiers.md` still applies (T3: ≥1 fresh-model reviewer)

- [ ] **Step 2: Write the shared protocol (exactly once)**

Three blocks, each appearing exactly once in the repo's review surface:
1. **Blindness Contract** — no-leaks rule + independence grades A/B/C + grade-C-diagnostic-only (carried from requesting-code-review).
2. **Six-value verdict table** — `high` / `medium` / `low` / `false` / `maybe-false` / `could-not-verify`, mapped to ledger severities (carried from parallel-code-review).
3. **Resilience rule** — a failed lens is logged by name; all-lenses-failed = "possibly incomplete", never "clean".

- [ ] **Step 3: Write dispatch + output contract**

Dispatch: fresh subagent per lens, prompt = reviewer template path + diff scope + Blindness Contract; templates in `reviewers/` (Task 9); red team last with the merged findings. Output contract preserved: `templates/review.md` audit record (Task 10) + `REVIEW-PASS` / `BACK-TO-APPLY` + Critical/Warn counts (the contract `ship` consumes) + a ledger round entry per `_shared/craft/review-ledger.md` (Task 11) + fix-loop routing: accepted fixes route to `receiving-code-review` (the fix loop keeps its role — R16/ADR-0065, no behavior change).

- [ ] **Step 4: Verify uniqueness + completeness**

Run: `grep -c "Blindness Contract" .agents/skills/code-review/SKILL.md`
Expected: ≥1 (defined here; other files cite, they do not restate — final repo-wide uniqueness check in Task 33).
Run: `grep -cE "maybe-false|could-not-verify" .agents/skills/code-review/SKILL.md`
Expected: ≥1 (the full six-value table is present).
Run: `grep -nE "50 changed lines|red team" .agents/skills/code-review/SKILL.md`
Expected: topology branches + red-team-last present.

- [ ] **Step 5: Commit**

```bash
git add .agents/skills/code-review/SKILL.md
git commit -m "feat(skills): code-review skill (merged requesting+parallel, topology step)"
```

**SATISFIES:** `happy path code-review-topology` (Task 1).

### Task 8: Hoist the verification-gap canon (Track E, R42)

**Files:**
- Create: `.agents/skills/_shared/verification/verification-gap.md`
- Modify: `.agents/skills/qa/references/verification-gap.md` → replaced by a one-line pointer (the file moves to `_shared/`; keep the old path as a pointer so in-flight citations resolve during migration, deleted in Task 13 once qa's citations point at the new path)

**Interfaces:**
- Consumes: `qa/references/verification-gap.md` (three-gap-shape canon + evidence rule) — the content source
- Produces: the single canon cited by qa's audit AND the code-review verification-gap reviewer (Task 9) — the double-home R42 removes
- Deletion test: the canon exists in two places that can drift — the R42 bug

- [ ] **Step 1: Move the content**

`git mv .agents/skills/qa/references/verification-gap.md .agents/skills/_shared/verification/verification-gap.md`, then edit the moved file: replace skill-local framing with shared-reference framing (citable from any skill; keep the three gap shapes + evidence rule intact).

- [ ] **Step 2: Leave a pointer at the old path**

Write `.agents/skills/qa/references/verification-gap.md` as: `Moved to `_shared/verification/verification-gap.md` (R42). Citation target: the new path.` — Task 13 deletes it once qa's citations point at the new path (the pointer exists only for the interim between Tasks 8 and 13).

- [ ] **Step 3: Verify**

Run: `test -f .agents/skills/_shared/verification/verification-gap.md && echo ok`
Expected: `ok`.
Run: `grep -c "gap" .agents/skills/_shared/verification/verification-gap.md`
Expected: ≥3 (canon intact).

- [ ] **Step 4: Commit**

```bash
git add .agents/skills/_shared/verification/verification-gap.md .agents/skills/qa/references/verification-gap.md
git commit -m "refactor(skills): hoist verification-gap canon to _shared (R42)"
```

**SATISFIES:** `happy path qa-debt-surfacing` (structural half — the canon home).

### Task 9: 8 reviewer templates (Track E, R39 + R40)

**Files:**
- Create: `.agents/skills/code-review/reviewers/code-reviewer.md`
- Create: `.agents/skills/code-review/reviewers/spec-reviewer.md`
- Create: `.agents/skills/code-review/reviewers/edge-case-hunter.md`
- Create: `.agents/skills/code-review/reviewers/verification-gap.md`
- Create: `.agents/skills/code-review/reviewers/security.md`
- Create: `.agents/skills/code-review/reviewers/accessibility.md`
- Create: `.agents/skills/code-review/reviewers/performance.md`
- Create: `.agents/skills/code-review/reviewers/red-team.md`
- Read (sources, unmodified until Task 16): `.agents/skills/requesting-code-review/references/*.md` (the 2-axis templates), `.agents/skills/parallel-code-review/reviewers/*.md`

**Interfaces:**
- Consumes: requesting-code-review's Standards + Spec templates (verbatim carry); parallel-code-review's reviewer prompts (edge-cases, accessibility, performance, red-team verbatim carry); requesting's `references/security-reviewer.md` (230-line: OWASP 5-step + four-part triage rubric + report format)
- Produces: the dispatch payloads Task 7's topology references — the 3-axis pass and the full fan-out must dispatch the *same* security specialist (R40)
- Deletion test: each axis would need its prompt inlined in SKILL.md — the template-per-lens structure R39 keeps

- [ ] **Step 1: Carry the core five**

`code-reviewer.md` + `spec-reviewer.md`: moved from `requesting-code-review/references/` unchanged except the skill-name header. `edge-case-hunter.md`: moved from `parallel-code-review/reviewers/`. `verification-gap.md`: copy parallel's verification-gap reviewer prompt, then replace its inline gap-shape text with a citation: `gap shapes + evidence rule per `_shared/verification/verification-gap.md`` (Task 8's home). `security.md` (R40): the requesting-code-review 230-line template wins — OWASP 5-step workflow + four-part triage rubric + report format, citing the `owasp-security` skill; parallel's 70-line variant is discarded (nothing unique — its deltas are subsumed by the full workflow; note this in a one-line comment at the top of the merged file).

- [ ] **Step 2: Carry the conditional + red team**

`accessibility.md` (UI diffs), `performance.md` (data access / rendering diffs), `red-team.md` (always last; prompt explicitly receives the other specialists' merged findings and hunts for what the consensus missed).

- [ ] **Step 3: Verify one security template, all eight present**

Run: `ls .agents/skills/code-review/reviewers/ | wc -l`
Expected: 8.
Run: `grep -c "OWASP" .agents/skills/code-review/reviewers/security.md`
Expected: ≥1.
Run: `grep -c "verification-gap.md" .agents/skills/code-review/reviewers/verification-gap.md`
Expected: ≥1.

- [ ] **Step 4: Commit**

```bash
git add .agents/skills/code-review/reviewers/
git commit -m "feat(skills): code-review reviewer templates (security merged, gap canon cited)"
```

**SATISFIES:** `happy path security-template-merge` (Task 1).

### Task 10: `code-review/templates/review.md` — carried-over output contract (Track E, R39)

**Files:**
- Create: `.agents/skills/code-review/templates/review.md`
- Read (source): `.agents/skills/requesting-code-review/templates/review.md`

**Interfaces:**
- Consumes: requesting-code-review's review.md template — the audit record `ship` parses
- Produces: the output contract (REVIEW-PASS/BACK-TO-APPLY + Critical/Warn counts + section shape) preserved byte-structurally (Must-Have truth 4)
- Deletion test: ship's review-signal path has no template to consume — the after-code contract breaks

- [ ] **Step 1: Copy the template**

Copy requesting-code-review's `templates/review.md` to `.agents/skills/code-review/templates/review.md` unchanged. If it embeds the skill name in headers, update the header to `code-review` only — every section heading and field `ship` or the ledger reads stays identical.

- [ ] **Step 2: Verify structural equality against the old template (before Task 16 deletes it)**

Run: `diff <(grep -E "^##|Critical|Warning|REVIEW-PASS|BACK-TO-APPLY" .agents/skills/requesting-code-review/templates/review.md) <(grep -E "^##|Critical|Warning|REVIEW-PASS|BACK-TO-APPLY" .agents/skills/code-review/templates/review.md)`
Expected: empty (or only the skill-name header line differs).

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/code-review/templates/review.md
git commit -m "feat(skills): code-review review.md template (output contract carried)"
```

**SATISFIES:** `happy path code-review-topology` (output-contract half).

### Task 11: Demote `review-ledger` to a shared reference (Track E, R41)

> ⚠ one-way: D3 — after this task the `skillgrid:review-ledger` invocation form no longer exists; the contract lives in a cited reference.

**Files:**
- Create: `.agents/skills/_shared/craft/review-ledger.md`
- Read (source): `.agents/skills/review-ledger/SKILL.md` (v1.0: round-entry format, frozen-candidate + invalidation, dispositions, lineage, debt lifecycle)

**Interfaces:**
- Consumes: the ledger skill's full contract
- Produces: the cited home for ledger rounds — consumed by `code-review` (Task 7 already cites it), `goldfish-resolve.md` (Task 22), `qa`'s debt-surfacing step (Task 12), and drift-mode finding rounds (Task 28)
- Deletion test: every consumer restates the round format privately — the drift R41 fixes

- [ ] **Step 1: Write the shared reference**

Sections: `## Round entry format` (the per-change `review-ledger.md` artifact shape — unchanged), `## Frozen candidate + invalidation`, `## Dispositions` (fixed / parked / waived), `## Lineage`, `## Debt lifecycle` (open→addressed→closed / expired), `## Stale flag` (3-changes-stale rule). Strip skill-invocation framing ("invoked from exactly three fixed points" becomes: the three fixed points *cite* this reference). Keep the contract byte-faithful — this is a demotion, not a rewrite. Add the A1 one-liner: goldfish-review rounds are ledger rounds (the Reviewer column accepts the Goldfish review; the debt-surfacing step covers parked goldfish findings).

- [ ] **Step 2: Verify contract completeness**

Run: `grep -cE "frozen|parked|waived|lineage|expired|stale" .agents/skills/_shared/craft/review-ledger.md`
Expected: ≥6 (all contract elements present).

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/_shared/craft/review-ledger.md
git commit -m "refactor(skills): review-ledger demoted to _shared/craft reference (R41)"
```

**SATISFIES:** `happy path ledger-as-reference` (Task 1).

### Task 12: `qa` carries debt surfacing + routing (Track E, R42)

**Files:**
- Modify: `.agents/skills/qa/SKILL.md`

**Interfaces:**
- Consumes: `review-ledger` Step 5 (debt surfacing — carried verbatim per R42) + Task 11's shared reference
- Produces: the at-qa-time surface where parked findings (code-review, Goldfish per R17, drift per R32) appear as WARNING context, never a gate
- Deletion test: parked debt is never surfaced again — the review-ledger Step 5 orphan R42 adopts

- [ ] **Step 1: Add the named step**

Insert a step named **Surface review debt** into qa's process (position: after the review-signal read, before the gate verdict): read prior archived changes' `review-ledger.md` files; surface open `parked` findings (context, not a gate); WARNING on file overlap with the current change. Content carried over verbatim from the ledger skill's Step 5, with the citation target now `_shared/craft/review-ledger.md`. Also (R17, A1 light edit): in qa's decision-debt paragraph, add one sentence naming goldfish-parked CRITICAL findings (surfaced via this debt-surfacing step) as a second source of the existing decision-debt WARNING — additive only; the four-state gate is unchanged (R16).

- [ ] **Step 2: Update routing + When-NOT-to-use**

qa's routing table and "When NOT to use" sections: replace `requesting-code-review` / `parallel-code-review` citations with `code-review` (topology per tier / diff size per its SKILL.md).

- [ ] **Step 3: Verify**

Run: `grep -n "Surface review debt" .agents/skills/qa/SKILL.md`
Expected: 1 hit.
Run: `grep -nE "requesting-code-review|parallel-code-review" .agents/skills/qa/SKILL.md`
Expected: 0 hits.

- [ ] **Step 4: Commit**

```bash
git add .agents/skills/qa/SKILL.md
git commit -m "feat(skills): qa surfaces review debt; routing to code-review (R42)"
```

**SATISFIES:** `happy path qa-debt-surfacing` (Task 1).

### Task 13: Citation migration — ~15 sites (Track E, R43)

**Files:**
- Modify (skill citation sites — the spec's R43 list): `.agents/skills/_shared/planning/rigor-tiers.md`, `.agents/skills/_shared/verification/floor.md`, `.agents/skills/_shared/rules/code-standards.md`, `.agents/skills/_shared/rules/sdd-structure.md` (5 sites), `.agents/skills/_shared/craft/deterministic-boundary.md`, `.agents/skills/_shared/craft/measurement.md`, `.agents/skills/_shared/agent-config/block.md`, `.agents/skills/ship/SKILL.md`, `.agents/skills/subagent-execution/SKILL.md` + `.agents/skills/subagent-execution/references/final-review.md`, `.agents/skills/simple-execution/SKILL.md`, `.agents/skills/simplify-code/SKILL.md`, `.agents/skills/style-review/SKILL.md`, `.agents/skills/dogfood/SKILL.md`, `.agents/skills/deep-research/skill-routing.md`
- Modify (live sites outside `.agents/skills/` — verified by the Step 1 grep, 2026-11-11): `README.md`, `CONTRIBUTING.md`, `docs/tutorials/your-first-change.md`, `docs/user-guide/01-layout.md`, `docs/user-guide/02-skills.md`, `docs/user-guide/03-workflow-usage.md`, `docs/user-guide/06-multi-agent-work.md`, `docs/user-guide/08-guiding-principles.md` (5 sites), `.skillgrid/projectwiki/ARCHITECTURE.md` (3 sites), `.skillgrid/projectwiki/workflows/development-workflow.md` (8 sites, incl. the `**Skill:** review-ledger` pipeline step), `.skillgrid/projectwiki/workflows/sdd-pipeline.md` (2 sites)
- Modify (Track E's own remaining consumers): `.agents/skills/qa/references/audit-procedure.md` (cites the hoisted verification-gap canon + `code-review`), `.agents/skills/receiving-code-review/SKILL.md` (A1 light edit: its no-gratitude examples become one citation of `_shared/craft/review-discipline.md`)
- Delete: `.agents/skills/qa/references/verification-gap.md` (Task 8's interim pointer — nothing references the old path after this task)
- (AGENTS.md is Task 15; `_shared/skill-size-budget.json` is Task 14)

**Scope rule (C1):** files *inside* the three deletion-target directories are NOT migration sites — they die in Task 16; do not edit them (Tasks 9/10's "sources, unmodified until Task 16" invariant).

**Interfaces:**
- Consumes: the deleted skills' former citation semantics (which skill, for which review, at which point)
- Produces: a repo where every review dispatch names `code-review` (topology picks the pass) — the precondition for Task 16's deletion
- Deletion test: Task 16 would leave dangling citations — this task is the migration that makes the deletion safe

- [ ] **Step 1: Enumerate the sites (repo-wide, excluding the three dirs that Task 16 deletes)**

Run:
```bash
grep -rn "requesting-code-review\|parallel-code-review\|skillgrid:review-ledger" . \
  --include="*.md" --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=.worktrees \
  --exclude-dir=requesting-code-review --exclude-dir=parallel-code-review --exclude-dir=review-ledger \
  | grep -v "change-archive" | grep -v "change-specs/" | grep -v "meta/.sdd/" | grep -v "decisions/04-adr-0054"
```
Expected: the site list above (skills + docs + projectwiki + README/CONTRIBUTING; the spec's own `briefing.md`, other change-specs, the superseded ADR-0054 body, scratch, and worktrees are historical/other-checkout — exempt, listed in Task 33 Step 1). Any file the list misses gets handled too; report the count.

- [ ] **Step 2: Migrate each site**

Rule per site: `requesting-code-review` or `parallel-code-review` → `code-review` (keep the surrounding semantics: "T2 two-axis" becomes "code-review per its topology"; "T3 parallel with fresh model" becomes "code-review fan-out; T3 fresh-model floor per `_shared/planning/rigor-tiers.md`"). `review-ledger` as a *skill invocation* → citation of `_shared/craft/review-ledger.md` (e.g. development-workflow.md's `**Skill:** review-ledger` step becomes the shared-reference citation); `review-ledger.md` as an *artifact name* (the per-change file) stays untouched. `qa/references/audit-procedure.md` and `receiving-code-review/SKILL.md` per the Files list. Then delete Task 8's interim pointer at `qa/references/verification-gap.md`. Projectwiki pages are vault pages (PW-INV-3: `index/` is derived) — after editing the `.skillgrid/projectwiki/` pages, run the projectwiki incremental index (`mnemonic projectwiki index`) so the FTS index is not left stale. No site may keep an alias or a "formerly known as" note (R43: no aliases).

- [ ] **Step 3: Verify zero live citations (repo-wide, same exclusions as Step 1 + the three dirs still present)**

Run:
```bash
grep -rn "requesting-code-review\|parallel-code-review\|skillgrid:review-ledger" . \
  --include="*.md" --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=.worktrees \
  --exclude-dir=requesting-code-review --exclude-dir=parallel-code-review --exclude-dir=review-ledger \
  | grep -v "change-archive" | grep -v "change-specs/" | grep -v "meta/.sdd/" | grep -v "decisions/04-adr-0054" | wc -l
```
Expected: 0 (the artifact-name `review-ledger.md` may still appear — it is the per-change file, not a skill; the three dirs' self-citations are exempt until Task 16).

- [ ] **Step 4: Commit**

```bash
git add -u .agents/skills/
git commit -m "refactor(skills): migrate ~15 citations to code-review + shared ledger reference (R43)"
```

**SATISFIES:** `happy path code-review-topology` (cutover half) + the clean-cutover scenario (Task 1).

### Task 14: `skill-size-budget.json` update (Track E, R43)

**Files:**
- Modify: `.agents/skills/_shared/skill-size-budget.json` (remove the 3 deleted skill names; add `code-review`)

**Interfaces:**
- Consumes: the two old entries' byte ceilings (the `code-review` entry = the larger of the two — the merged skill is the union)
- Produces: a budget map the size-check script validates against (qa runs it)
- Deletion test: the script validates 3 dead entries + misses the new skill

- [ ] **Step 1: Edit the JSON**

The file's structure: `ceiling` (standard 22000 / large 38000 / xl 40000 / external 90000), `tiers` (skill → tier), `skills` (skill → last-known baseline size in bytes). Remove the two keys `requesting-code-review` (14090) and `parallel-code-review` (13172) — `review-ledger` has no key (never was in the budget). Add `code-review: 14090` to `skills` (the larger of the two old baselines, per R43's "one `code-review` entry at the post-merge ceiling"); `code-review` is in no `tiers` entry → the standard 22000 ceiling applies. Keep JSON valid (no comments, trailing-comma-free).

- [ ] **Step 2: Run the budget script**

Run: `node .agents/skills/qa/scripts/skill-size-budget.mjs check`
Expected: exit 0 (the new `code-review` SKILL.md is within its ceiling; nothing over budget).

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/_shared/skill-size-budget.json
git commit -m "chore(skills): size budget — drop deleted skills, add code-review"
```

**SATISFIES:** the clean-cutover scenario (budget half, Task 1).

### Task 15: AGENTS.md workflow line (Track E, R43)

**Files:**
- Modify: `AGENTS.md` (the `## Workflow` line that names `requesting-code-review`)

**Interfaces:**
- Consumes: the current workflow line (`… → qa → requesting-code-review → receiving-code-review → ship → …`)
- Produces: the canonical workflow naming `code-review`
- Deletion test: the repo's own agent instructions point at a deleted skill

- [ ] **Step 1: Edit the line**

`requesting-code-review` → `code-review` in the workflow line. Leave the rest of the line intact.

- [ ] **Step 2: Verify**

Run: `grep -n "code-review" AGENTS.md`
Expected: the workflow line, no other occurrences.
Run: `grep -cn "requesting-code-review" AGENTS.md`
Expected: 0.

- [ ] **Step 3: Commit**

```bash
git add AGENTS.md
git commit -m "docs: workflow line — requesting-code-review → code-review"
```

**SATISFIES:** the clean-cutover scenario (Task 1).

### Task 16: Delete the three skill directories (Track E, R43)

> ⚠ one-way: D1 — explicit user approval required before this task runs (see one-way-door list). Precondition: Tasks 7–15 all committed.

**Files:**
- Delete: `.agents/skills/requesting-code-review/` (entire directory)
- Delete: `.agents/skills/parallel-code-review/` (entire directory)
- Delete: `.agents/skills/review-ledger/` (entire directory)

**Interfaces:**
- Consumes: nothing (the content is fully carried into `code-review/` + `_shared/craft/review-ledger.md`)
- Produces: the post-consolidation skill surface — no aliases, no stubs
- Deletion test: n/a (this is the deletion)

- [ ] **Step 1: Confirm the precondition (zero citations OUTSIDE the three dirs)**

Run:
```bash
grep -rn "requesting-code-review\|parallel-code-review\|skillgrid:review-ledger" . \
  --include="*.md" --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=.worktrees \
  --exclude-dir=requesting-code-review --exclude-dir=parallel-code-review --exclude-dir=review-ledger \
  | grep -v "change-archive" | grep -v "change-specs/" | grep -v "meta/.sdd/" | grep -v "decisions/04-adr-0054" | wc -l
```
Expected: 0. (Self-citations *inside* the three dirs are expected — they die with the dirs. Anything outside is a straggler: fix it first — the deletion must not orphan a citation.)

- [ ] **Step 2: Delete**

```bash
git rm -r .agents/skills/requesting-code-review .agents/skills/parallel-code-review .agents/skills/review-ledger
```
If `git rm` reports staged/local changes inside a directory, the Task 13 scope rule was violated — `git restore --staged --worktree` those specific files first, then re-run (the sources must be unmodified per Tasks 9/10).

- [ ] **Step 3: Verify the deletion state**

Run: `git status --porcelain | grep -c "^D "`
Expected: equals the total file count of the three directories (spot-check: `git status --porcelain | grep "^D " | head`).
Run: `ls .agents/skills/ | grep -E "requesting|parallel-code|review-ledger"`
Expected: nothing.

- [ ] **Step 4: Commit**

```bash
git commit -m "refactor(skills): delete requesting-code-review, parallel-code-review, review-ledger (R43)"
```

**SATISFIES:** `happy path ledger-as-reference` + the clean-cutover scenario (Task 1).

### Task 17: `brainstorming/references/shootdown.md` (Track A, R1–R3/R36/R37)

**Files:**
- Create: `.agents/skills/brainstorming/references/shootdown.md`

**Interfaces:**
- Consumes: Tasks 4/5's persona briefs (the 6 lens files), spec §Track A (R1 load-bearing fresh-context rule, R2 dispatch point, R3 findings → briefing, R4 gate), the 4-part isolation contract (spec R37)
- Produces: the shootdown workflow Task 18 wires into brainstorming's paths
- Deletion test: the shootdown logic inlines into SKILL.md — the references/ split brainstorming already uses

- [ ] **Step 1: Write the workflow**

Sections:
1. `## Dispatch point (R2)` — New Project / New Function paths only; runs **after the codebase feasibility check passes and before the ADR manifest is written** (the brainstorming flow is interview → approaches → design → feasibility → ADR manifest → briefing — the shootdown sits between feasibility and the ADR manifest, so the manifest is written against a shootdowned design). Bounded (one-file small) and prototype (already a probe) paths skip it; their path descriptions do not name the shootdown. Path ratchet unchanged: if a bounded task upgrades to New Function mid-task, the shootdown becomes mandatory at the new path.
2. `## Freshness check (R36)` — before the six lenses spawn: *"If your harness has no subagent tool, stop and tell the user: a review in the same context is not a Goldfish review."* The skill halts and names the EGM rule it is enforcing — no partial run, no silent degradation, no "we'll count this one as a review anyway."
3. `## The six lenses (R1)` — all six dispatched **in parallel** (not sequentially), one freshly spawned `reviewer` subagent (fallback `general`) per lens brief: `lens-contrarian-yagni.md`, `lens-cost-economics.md`, `lens-ux-ergonomics.md`, `lens-migration-compat.md`, `lens-observability-sre.md`, `lens-tech-debt.md` (all under `_shared/elephant-goldfish/`). Dispatch prompt = lens brief path + the anchored hypothesis + the explicit reading list (the relevant existing code/files) + the R37 four-part isolation contract re-asserted verbatim + `Mode: BRAINSTORM`. Each subagent is given ONLY the anchored hypothesis + relevant code — no conversation history, no author narrative. A review run in the author's own context MUST NOT count toward R1.
4. `## Error handling` — a reviewer that times out / returns no report: retry once, then record `not run` (never zero findings); a lens that is `not run` caps its contribution at "not verified," not "clean."
5. `## Synthesis + finding gate (R3)` — after the six lenses report, the main agent (the Elephant) synthesizes inline: dedupes overlapping findings, groups by severity (🔴 Blocker / 🟡 Major / 🟢 Nit). The user is shown the 🔴 Blockers and disposes of each: **resolved** (design revised, loop back to the feasibility check) / **accepted-with-note** (recorded in the briefing's `## Shootdown Findings` as a known risk) / **downgraded-to-non-goal** (moved to the spec's Out of Scope). 🟡/🟢 are recorded but do not block. The spec is NOT written until all 🔴 Blockers are dispositioned (the dispositions land in Task 19's section).
6. `## Consolidated chat response` — the synthesis turn ends with: the severity table (normalized vocabulary at the record boundary), every 🔴 finding with its disposition options, and the path where the findings will land. A turn that ends with only a file path does not satisfy this.

- [ ] **Step 2: Verify**

Run: `grep -oE "lens-[a-z-]+\.md" .agents/skills/brainstorming/references/shootdown.md | sort -u | wc -l`
Expected: 6 (each lens brief named explicitly).
Run: `grep -n "STOP" .agents/skills/brainstorming/references/shootdown.md`
Expected: the freshness check present.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/brainstorming/references/shootdown.md
git commit -m "feat(skills): brainstorming shootdown workflow (6 fresh lenses, blocker gate)"
```

**SATISFIES:** the shootdown scenarios for R1–R4 (Task 1).

### Task 18: brainstorming SKILL + paths — wire the shootdown (Track A, R2/R5)

**Files:**
- Modify: `.agents/skills/brainstorming/SKILL.md` (paths: insert shootdown step; add `## Anti-Sycophancy & Stress-Testing` section)
- Modify: `.agents/skills/brainstorming/paths.md` (checklist step)

**Interfaces:**
- Consumes: Task 17's workflow + Task 3's `review-discipline.md` (cited, not restated)
- Produces: the gated brainstorming flow — feasibility → shootdown → dispositioned findings → spec
- Deletion test: the gate exists only in a reference nobody's path calls — the wiring is the behavior

- [ ] **Step 1: Edit the paths**

In SKILL.md's New Project / New Function path descriptions and in `paths.md`'s checklists (update the process-flow diagram too, per the module list), insert the shootdown step **between the codebase feasibility check and the ADR manifest** as a one-line pointer: `Run the Goldfish shootdown per `references/shootdown.md` (6 parallel fresh lenses; all 🔴 Blockers dispositioned before the ADR manifest is written).` The Bounded and Prototype path descriptions do NOT name the shootdown (R2); the ratchet (bounded → New Function mid-task makes it mandatory) is stated where the path descriptions live.

- [ ] **Step 2: Add the Anti-Sycophancy section**

`## Anti-Sycophancy & Stress-Testing` in SKILL.md: the rules are cited from `_shared/craft/review-discipline.md` (not restated) and stated to apply to the shootdown subagents and the main agent's synthesis. The banned-adjective list appears here nowhere — one grep-verifiable home (Task 3).

- [ ] **Step 3: Verify**

Run: `grep -n "shootdown" .agents/skills/brainstorming/SKILL.md .agents/skills/brainstorming/paths.md`
Expected: ≥2 hits total, one per file minimum.
Run: `grep -n "Anti-Sycophancy" .agents/skills/brainstorming/SKILL.md`
Expected: 1 heading.
Run: `grep -c "review-discipline.md" .agents/skills/brainstorming/SKILL.md`
Expected: ≥1.

- [ ] **Step 4: Commit**

```bash
git add .agents/skills/brainstorming/SKILL.md .agents/skills/brainstorming/paths.md
git commit -m "feat(skills): brainstorming paths gate on the shootdown; anti-sycophancy section"
```

**SATISFIES:** `happy path anti-sycophancy-rules` (Task 1).

### Task 19: brainstorming `templates/briefing.md` — Shootdown Findings (Track A, R3/R4)

**Files:**
- Modify: `.agents/skills/brainstorming/templates/briefing.md` (add `## Shootdown Findings` section to the briefing scaffold)

**Interfaces:**
- Consumes: Task 17's finding gate (the section is where dispositions are recorded)
- Produces: the durable record of the shootdown in every future briefing — recoverable without session memory (repo-is-truth)
- Deletion test: shootdown findings live in chat and evaporate — the ADR-0013 violation

- [ ] **Step 1: Add the section**

Insert `## Shootdown Findings` into the briefing template **between the `## Implementation Decisions` and `## Testing Decisions` sections** (the template's order is … → Approaches Considered → Requirements → Implementation Decisions → Testing Decisions → …). The section carries the three R4 sub-parts: (1) the anchored hypothesis (2–3 sentences, as confirmed by the user at the readiness gate), (2) the severity table (🔴/🟡/🟢 rows: lens, description, disposition — resolved / accepted-with-note / downgraded-to-non-goal — plus note), (3) a one-line overall verdict (survived / survived-with-notes / needs-revision). Plus the gate line: *all 🔴 Blocker findings dispositioned before the spec is written — an undispositioned blocker blocks the write.* No separate `concepts_brief.md` — `briefing.md` is the single source of truth (ADR-0013). Bounded/prototype changes carry an explicit `Skipped: bounded/prototype path` note in the section.

- [ ] **Step 2: Verify**

Run: `grep -n "Shootdown Findings" .agents/skills/brainstorming/templates/briefing.md`
Expected: 1 heading.
Run: `grep -c "dispositioned" .agents/skills/brainstorming/templates/briefing.md`
Expected: ≥1.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/brainstorming/templates/briefing.md
git commit -m "feat(skills): briefing template gains Shootdown Findings + disposition gate"
```

**SATISFIES:** the shootdown-finding scenarios for R3 (Task 1).

### Task 20: `writing-blueprints/references/goldfish-review.md` (Track B, R6–R8/R13/R36/R37)

**Files:**
- Create: `.agents/skills/writing-blueprints/references/goldfish-review.md`

**Interfaces:**
- Consumes: Tasks 4/5's persona briefs (the 7 review files), spec §Track B R6 (5-stage pipeline + 4 presets), R7 (severity normalization), R8 (dispatch shape), R37 (isolation), the code-review Blindness Contract (Task 7)
- Produces: the dispatch protocol Task 24 wires into writing-blueprints' validation seam; the report path (`## Review Summary` in the blueprint, Task 25)
- Deletion test: the pipeline definition inlines into SKILL.md — the references/ split writing-blueprints already uses

- [ ] **Step 1: Write the pipeline**

Sections:
1. `## The five stages (R7)` — one line per stage (the stage list is NOT a single combined prompt): **Stage 5 — Initial Zoom Out** (target: full `blueprint.md`; YAGNI, reuse, anti-weeds, altitude → dispatch `zoom-out.md` → `zoom_out_report.md`); **Stage 6 — Comprehension Test** (target: `blueprint.md` + referenced files; paraphrase first, self-containment → dispatch `comprehension.md` → `comprehension_report.md`); **Stage 7 — Critic Review** (target: `blueprint.md`, the *how*; dispatch `design-critic.md` + `structural.md` + `style.md` — three fresh subagents, one per brief, findings merged → `critic_report.md`); **Stage 8 — Intermediate Zoom Out** (target: full `blueprint.md`, re-read after Stages 6+7 fixes; catches altitude drift + scope creep introduced by fixes — re-dispatch `zoom-out.md` as a *fresh* subagent, not the Stage 5 one → `intermediate_zoom_out_report.md`); **Stage 9 — Implementation Readiness** (target: `blueprint.md` task steps + codebase; actionability, codebase accuracy → dispatch `implementation-readiness.md` + `codebase-details.md` + `style.md` → `implementation_readiness_report.md`). After each stage's resolution the main agent folds findings inline — no synthesizer persona; the Consolidated preset's cross-stage dedup + traceability matrix is also main-agent work (R11 stage mapping: `style.md` is used by Stages 7 and 9).
2. `## Presets (single-select, R8)` — the four presets, selected by the user or inferred from the invocation context: **Interactive** (checkpoint after each stage — full pipeline with user checkpoint gates between stages; default for New Project / New Function blueprints), **Consolidated** (no pauses — full pipeline, findings consolidated into `consolidated_review_report.md` with cross-stage dedup + traceability matrix), **Feature-spec-only** (Stages 5-8, skip 9), **Implementation-only** (Stage 9 only). Single-select gate if not provided at invocation; multi-select prohibited; the topological dependency 5→6→7→8→9 is preserved under every preset.
3. `## Dispatch protocol (R7/R36/R37)` — freshness check first: R36's STOP rule applies before Stage 5 is dispatched (verbatim rule in `harnesses.md`, Task 23 — a same-context run MUST NOT count toward R7). Each stage dispatches fresh `reviewer` subagents (fallback `general`), one per brief, prompt = persona brief path + target path + explicit reading list (the files the target references) + the R37 four-part isolation contract re-asserted verbatim + `Mode: REVIEW`; subagents never see the author conversation, project instructions, or the network. Error handling: a reviewer that times out / returns no report → retry once → record `not run` (never zero findings); a stage whose reviewer is `not run` caps that stage's verdict at "not verified," not "clean." Every stage report's header carries the one-line harness-isolation attestation (which of the four parts the harness enforced vs. could not; a harness that cannot enforce part 3 says so — no silent pass).
4. `## Record boundary (R7)` — findings carry reviewer vocabulary in transit; at the ledger round entry (per `_shared/craft/review-ledger.md`, Task 11) they are normalized to CRITICAL/WARNING/SUGGESTION. This is the single normalization point.
5. `## Committed report artifacts (R13)` — each stage writes its findings to `.skillgrid/change-specs/<topic>/review/`: `zoom_out_report.md`, `comprehension_report.md`, `critic_report.md`, `intermediate_zoom_out_report.md`, `implementation_readiness_report.md` (the five stage files, committed, not chat-only); the Consolidated preset additionally writes `consolidated_review_report.md`. The resolution record is the `review-ledger.md` round entry (per-finding AI suggestion + justification + proposed edit in the ledger's Dispositions, with finding IDs, dispositions, fix-commit lineage — no separate goldfish report file for resolutions). The blueprint's `## Review Summary` section (Task 25) carries the condensed verdict; parked findings flow to qa via the debt-surfacing step (R17 — implemented in Task 12, cited here). **Consolidated chat response (R13):** after any run — all presets — the chat turn ends with the stage/reviewer status table, the severity breakdown in normalized vocabulary, every CRITICAL/WARNING finding with its proposed disposition, and the report path + ledger entry. A turn that ends with only a file path does not satisfy this requirement.

- [ ] **Step 2: Verify**

Run: `grep -oE "Stage [5-9] — [A-Za-z ]+" .agents/skills/writing-blueprints/references/goldfish-review.md | sort -u | wc -l`
Expected: 5 (each stage is a distinct entry, not one combined prompt — R7's acceptance).
Run: `grep -cE "structural\.md|style\.md" .agents/skills/writing-blueprints/references/goldfish-review.md`
Expected: ≥2 (Stage 7's composition includes structural + style per R11).
Run: `grep -cE "Interactive|Consolidated|Feature-spec-only|Implementation-only" .agents/skills/writing-blueprints/references/goldfish-review.md`
Expected: ≥4 (all four preset names — R8).

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/writing-blueprints/references/goldfish-review.md
git commit -m "feat(skills): goldfish-review 5-stage pipeline + presets (Track B)"
```

**SATISFIES:** the goldfish-pipeline scenarios for R6–R8 (Task 1).

### Task 21: `references/goldfish-review-loop.md` (Track B, R9)

**Files:**
- Create: `.agents/skills/writing-blueprints/references/goldfish-review-loop.md`

**Interfaces:**
- Consumes: Task 20's pipeline (the loop iterates it), the ledger's frozen-candidate + invalidation rules (Task 11)
- Produces: the sharpening-loop mechanics Task 24's rewritten validation section invokes
- Deletion test: convergence/churn/turn-taking rules scatter across SKILL.md and the loop file — the two homes drift

- [ ] **Step 1: Write the loop**

Sections (R9's six rules, verbatim semantics): `## Multi-round` (up to 5 rounds per stage; each round: spawn reviewers → collect findings → resolve (categorize + fix) → frozen-candidate re-check); `## Convergence` (2 consecutive **clean rounds** — a round is clean = no new Blocker or Major finding that is not settled; one clean round is not enough — a fresh reviewer can find a different gap in the same text); `## Settled findings` (contested or accepted-without-fix; kept across rounds; if a new reviewer raises a settled finding again — same scope, same issue — do NOT re-open it); `## Fixed findings` (an applied fix does NOT settle a finding; if a later reviewer raises the same issue at the same scope, the fix failed — re-open); `## Churn` (a new finding is at the scope of an applied fix, and reverting that fix would resolve it; on churn, ask the user: **accept and proceed / continue past the cap / return to drafting**); `## Turn-taking` (after each round's resolution the loop goes idle and waits for the user to confirm "run another round" — the agent does not auto-loop); `## Frozen-candidate re-check` (A1, replaces the original line-range re-scan: before each round, re-read `blueprint.md` — line numbers shift after edits — and re-hash it; if the hash changed since the round started, the round is invalidated per the `review-ledger` frozen-candidate rule (Task 11) and re-dispatched).

- [ ] **Step 2: Verify**

Run: `grep -n "2 consecutive clean" .agents/skills/writing-blueprints/references/goldfish-review-loop.md`
Expected: 1 hit (the convergence rule, stated once).
Run: `grep -cE "cap|5 rounds|churn|turn" .agents/skills/writing-blueprints/references/goldfish-review-loop.md`
Expected: ≥4.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/writing-blueprints/references/goldfish-review-loop.md
git commit -m "feat(skills): goldfish sharpening loop (convergence, churn, turn-taking)"
```

**SATISFIES:** `happy path sharpening-loop-convergence` (Task 1).

### Task 22: `references/goldfish-resolve.md` (Track B, R10/R17)

**Files:**
- Create: `.agents/skills/writing-blueprints/references/goldfish-resolve.md`

**Interfaces:**
- Consumes: the code-review triage protocol (Task 7's skill, via receiving-code-review's buckets: fix now / defer / human look / noise) + the ledger reference (Task 11) + qa's debt-surfacing step (Task 12)
- Produces: the Elephant-side resolution gate — findings are resolved by the main agent (Elephant), never silently, before the blueprint advances
- Deletion test: findings get triaged by the Goldfish reviewers themselves — the same-context rubber-stamp R10 exists to kill

- [ ] **Step 1: Write the resolution workflow**

Sections:
1. `## The four actions` — each finding is categorized into exactly one: **CONTEST** (inaccurate — cite evidence the reviewer could read), **ESCALATE** (inaccurate but evidence is out-of-doc — flag for the user), **PROPOSE-FIX** (valid — name the exact edit), **DECLINE** (nit — record, don't fix). **Triage adapter (A1 — an adapter, not a new system):** CONTEST = the code-review triage verdict `false` (state what disproves it — R39); ESCALATE = `could-not-verify` → `receiving-code-review`'s human-look bucket; PROPOSE-FIX = the fix-now bucket; DECLINE = the low/noise bucket. The buckets are cited from `receiving-code-review`, not restated — this file adds only the doc-review deltas below.
2. `## The four rules` — a contest must be checkable ("this was intentional" is not evidence); do not add context just to satisfy a reviewer (if only a zero-context subagent would need it → ESCALATE); do not contest comprehension gaps (a reviewer that read the target and still missed it describes a self-containment defect — PROPOSE-FIX); do not change severity to win. The author wrote the document — the desire to contest is not evidence the reviewer is wrong.
3. `## Resolution context` — resolution runs in the Elephant's context (the main agent, never a subagent): it has the institutional knowledge to write good fixes and defend design choices; reviewers are never asked to self-critique.
4. `## Interactive gate` — after each round's resolution the user picks: **Auto accept** (batch-apply all PROPOSE-FIX) / **Continue** (no changes) / **Interactive break down** (per-item: Accept / Skip / Discuss) / **Comment on file** (user edits the report directly) / **Custom instruction** (free text). The gate never auto-advances — the loop stays idle (turn-taking per `goldfish-review-loop.md`).
5. `## Parked findings (R17)` — a finding may be parked with its path recorded; the gate verdict is otherwise unaffected (WARNING, never CRITICAL); at qa time the debt-surfacing step (Task 12) surfaces it as a WARNING naming the parked finding and its path.
6. `## Ledger round` — every resolution round writes a ledger round entry per `_shared/craft/review-ledger.md` (dispositions: fixed / parked / waived; severities normalized at this boundary per Task 20's record-boundary rule).

- [ ] **Step 2: Verify**

Run: `grep -cE "CONTEST|ESCALATE|PROPOSE-FIX|DECLINE" .agents/skills/writing-blueprints/references/goldfish-resolve.md`
Expected: ≥4.
Run: `grep -n "parked" .agents/skills/writing-blueprints/references/goldfish-resolve.md | head -3`
Expected: the R17 parked-finding flow present.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/writing-blueprints/references/goldfish-resolve.md
git commit -m "feat(skills): goldfish resolution workflow (Elephant-side, 4 actions, parking)"
```

**SATISFIES:** `happy path parked-finding-surfaced-at-qa` (Task 1).

### Task 23: `references/harnesses.md` (cross-track, R36/R38)

**Files:**
- Create: `.agents/skills/writing-blueprints/references/harnesses.md`

**Interfaces:**
- Consumes: spec R36 (freshness non-negotiable) + R38 (portability across Claude Code, OpenCode, Cursor, Pi)
- Produces: the shared STOP rule + the harness capability table — cited by shootdown.md (Task 17 already carries the rule inline) and goldfish-review.md (Task 20)
- Deletion test: every dispatching skill re-derives subagent availability per harness — the table is the single source

- [ ] **Step 1: Write the file**

Sections:
1. `## Freshness rule (R36)` — a harness without fresh-context subagents does NOT degrade — it STOPS. State verbatim: *"If your harness has no subagent tool, stop and tell the user: a review in the same context is not a Goldfish review."* The check applies at each track's first dispatch point (shootdown: before the six lenses spawn; plan review: before Stage 5 is dispatched; drift: before the first `codebase-details` reviewer dispatches). The stop is not a suggestion: the skill halts, tells the user a same-context review is not a Goldfish review, and names the EGM rule it is enforcing.
2. `## Harness table (R38)` — a port of EGM's `docs/harnesses.md`, scoped to the four harnesses; rows: **Claude Code** (subagent tool: `Agent` (Task); agents at `.claude-plugin/agents/<persona>.md`; isolation: `omitClaudeMd: true`, `tools: [Read, Grep, Glob]`, `model: inherit`; nesting depth 3 — the main context may dispatch reviewers directly), **OpenCode** (subagent tool: `subagent`; agents at `.opencode/agents/<persona>.md`; isolation: `mode: subagent`, `permissions: [edit: deny, shell: deny]`), **Cursor** (subagent tool: Cursor's Task subagent; agents at `.cursor/agents/<persona>.md`; isolation: read-only via agent permission settings; `rules/` files are not loaded for subagents by default — verify at install), **Pi** (subagent capability detected at runtime via native delegation — MCP + agent delegation since v0.99; if the current session cannot spawn a fresh-context subagent, R36's stop rule applies — do not degrade). Columns: subagent tool, read-only mechanism, project-instruction exclusion (R37 part 3), partial-enforcement note (a harness that cannot enforce part 3 says so in the report header — no silent pass). Plus a `## Contributing a new harness` section (per the module list). **Deferral (not built in this change):** the OpenCode Goldfish plugin — `.opencode/plugins/goldfish/index.ts` (`context` hook stripping instruction sources, `tool.transform` denying edit/write/shell, `session.hook("prompt")` denying web tools) is the spec's Open Questions deferral candidate; document it here as the native-enforcement path for OpenCode.

- [ ] **Step 2: Verify**

Run: `grep -cE "Claude Code|OpenCode|Cursor|Pi" .agents/skills/writing-blueprints/references/harnesses.md`
Expected: ≥4.
Run: `grep -n "STOP" .agents/skills/writing-blueprints/references/harnesses.md`
Expected: the freshness rule present.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/writing-blueprints/references/harnesses.md
git commit -m "feat(skills): harness portability table + freshness STOP rule (R36/R38)"
```

**SATISFIES:** `happy path harness-portability` (Task 1).

### Task 24: Replace `plan-review.md` with the goldfish validation seam (Track B, R6/R14/R15)

> ⚠ one-way: D2 — the validation seam changes from a 6-check inline review to the 5-stage fresh-context pipeline.

**Files:**
- Delete: `.agents/skills/writing-blueprints/references/plan-review.md`
- Modify: `.agents/skills/writing-blueprints/SKILL.md` (the Plan Review / validation section)

**Interfaces:**
- Consumes: Tasks 20–23's four references (the pipeline, loop, resolution, harnesses)
- Produces: writing-blueprints' validation = self-review checklist → goldfish-review dispatch (per `references/goldfish-review.md`) → user-decide-then-fix loop (mandatory, while the blueprint is still PROPOSED) → commit → state.yaml phase update → execution handoff
- Deletion test: the old 6-check review survives as a second validation path — the two gates diverge

- [ ] **Step 1: Delete the old reference**

```bash
git rm .agents/skills/writing-blueprints/references/plan-review.md
```

- [ ] **Step 2: Rewrite the validation section in SKILL.md**

Replace the validation block (R14, verbatim order): **Self-Review** → **Owed-Decision Gate** → **Goldfish review** (`references/goldfish-review.md`, run on `blueprint.md`) → **User Review Gate** → **Execution Handoff**. The Goldfish step runs the 5-stage pipeline (preset single-select) with the sharpening loop per `references/goldfish-review-loop.md` (convergence: 2 consecutive clean rounds, cap 5, turn-taking) and the resolution gate per `references/goldfish-resolve.md`. The invocation passes (R14): `TargetFile` = `blueprint.md`, `GoldfishContext` = `briefing.md` + referenced files, `Step` = inferred from the path classification (Interactive for New Project, Consolidated for New Function with ≥3 tasks). Keep: the self-review checklist (unchanged — it runs first), the user-decide-then-fix loop requirement, the commit + `state.yaml` phase update, the execution handoff (slicing decision for ≥3 tasks; subagent-driven vs inline). Delete: the single generic Critic subagent, the old 6-check Plan Review prompt and its one-pass cap — both **replaced** by the Goldfish reference set per R14 (the checks map onto the stages, nothing is lost: failure modes + edge cases → Stage 9 (implementation-readiness + codebase-details), scope coherence → Stages 5/8 (zoom-out), feasibility → Stage 9 (codebase-details), ordering → Stage 7 (structural), test strategy → Stage 9 (implementation-readiness), ADR compliance → Stage 7 (design-critic)). SKILL.md stays thin (R15): the rewrite is a pointer to the reference set + short rewrites — no protocol detail inlined; the size-budget check in Task 33 re-confirms the ceiling holds.

- [ ] **Step 3: Verify**

Run: `test ! -f .agents/skills/writing-blueprints/references/plan-review.md && echo gone`
Expected: `gone`.
Run: `grep -c "goldfish" .agents/skills/writing-blueprints/SKILL.md`
Expected: ≥3 (dispatch + loop + resolve citations).
Run: `grep -n "plan-review" .agents/skills/writing-blueprints/SKILL.md`
Expected: 0.

- [ ] **Step 4: Commit**

```bash
git add -u .agents/skills/writing-blueprints/
git commit -m "refactor(skills): writing-blueprints validation seam → goldfish pipeline (plan-review deleted)"
```

**SATISFIES:** the goldfish-pipeline scenarios for R6 (Task 1).

### Task 25: `templates/blueprint.md` — Review Summary section (Track B, R13/R14)

**Files:**
- Modify: `.agents/skills/writing-blueprints/templates/blueprint.md` (the `## Plan Review` output block → `## Review Summary`)

**Interfaces:**
- Consumes: Task 20's report path (the pipeline writes its summary here)
- Produces: the blueprint scaffold whose final section matches the goldfish pipeline's output
- Deletion test: the pipeline's report has no scaffold section — every blueprint hand-writes a differently-shaped summary

- [ ] **Step 1: Rename + reshape the block**

The template has no `## Plan Review` block (headings: Title → Terms → Must-Haves → Global Constraints → File Structure → Task skeletons) — so this is an **addition**: append a `## Review Summary` sample block after the Task skeletons: fields = verdict (READY FOR EXECUTION / NEEDS REVISION), rounds run + convergence state, findings by severity (normalized vocabulary per the record boundary), parked findings (path + severity — the R17 surface), reviewed date. Also add a `<!-- Self-Review reminder: ... -->` comment in the template header block (Track C's decision): the self-review checklist + the Spec Reconciliation check run before the User Review Gate.

- [ ] **Step 2: Verify**

Run: `grep -n "Review Summary" .agents/skills/writing-blueprints/templates/blueprint.md`
Expected: ≥1.
Run: `grep -n "Self-Review reminder" .agents/skills/writing-blueprints/templates/blueprint.md`
Expected: 1.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/writing-blueprints/templates/blueprint.md
git commit -m "feat(skills): blueprint template — Plan Review → Review Summary"
```

**SATISFIES:** the goldfish-pipeline scenarios for R6 (report-path half).

### Task 26: `references/section-by-section.md` (Track C, R18–R25)

**Files:**
- Create: `.agents/skills/writing-blueprints/references/section-by-section.md`

**Interfaces:**
- Consumes: spec §Track C (R18 ingestion, R19 section order + gate, R20 per-task gate, R21 Spec Reconciliation protocol, R25 artifact-first — R22/R23/R24 are SKILL.md-level, Task 27) + EGM's `egm-draft` section-by-section model
- Produces: the drafting process Task 27 wires into writing-blueprints — how every blueprint (including the next change's) gets built
- Deletion test: drafting stays one-shot with a single end-of-file review — the failure mode Track C exists to kill

- [ ] **Step 1: Write the drafting workflow**

Sections:
1. `## Ingestion (R18)` — before writing any section: (1) read `briefing.md` in full, (2) read `findings.md` if present, (3) read `adr.md` if present, (4) present a 2–3 sentence summary of the briefing's key decisions to the user, (5) confirm the draft starts from the right context. Light-weight — the briefing already exists; the step makes ingestion explicit and adds the confirmation gate.
2. `## Section order (R19)` — the drafting order, verbatim from the spec: Header → Hypothesis → Must-Haves → Threat Matrix (if applicable) → File Structure → Global Constraints → Tasks 1..N → Self-Review → Commit. One line per section with its content contract (point at the SKILL.md section — cite, don't restate). This is the order sections are drafted and gated; the final document's layout follows `templates/blueprint.md`.
3. `## Per-section gate (R19)` — artifact-first (R25): write the section to `blueprint.md` (the file is the deliverable — never paste 200 lines into chat); then present in the same chat turn: which section was drafted, the 2–3 key decisions in it (one line each), and the path to `blueprint.md` — the chat summary precedes the gate (no empty chat bubbles); then the single-select approval gate: **"Approve [Section] and proceed"** / **"Request revisions to [Section]"**. The user can revisit any earlier section at any point.
4. `## Per-task gate (R20)` — each task in the Tasks section gets its own approval gate (the section gate at task granularity): the agent writes Task 1, presents the gate, the user approves, then the agent writes Task 2. The task template (Files / Interfaces / SATISFIES / Steps with code) is structurally unchanged — only the approval cadence changes. This is the highest-value addition of Track C: it catches a wrong interface, a missing file, or a bad test strategy *before* the agent writes 15 more tasks.
5. `## Spec Reconciliation (R21)` — the full protocol for the Self-Review check Task 27 adds to SKILL.md: re-read `briefing.md` and compare it against the blueprint. **Contradiction:** the briefing decided a question and the blueprint decides it differently — report every one, quoting the briefing and the blueprint. **Addition:** the blueprint decides a question the briefing did not raise — report it only if it changes the architecture, an external interface, or a dependency. List contradictions first. Do NOT change the blueprint to match the briefing on your own: the blueprint can be right, because drafting reads code that the brainstorm did not. If nothing diverges, say so in one line. This is the check "no Goldfish can do, because no Goldfish reads the brief."
6. `## The 8 drafting disciplines` — Track C's eight disciplines ARE the eight Track C requirements, ported from EGM's `egm-draft`: explicit ingestion (R18), section-by-section drafting (R19), per-task gate (R20), spec reconciliation (R21), owner-decides-architecture (R22), alternatives-from-briefing (R23), anti-sycophancy + plain-English (R24), artifact-first delivery (R25). One line each: what it is + where it lives (this file: R18–R21 protocol; SKILL.md sections: R22–R25 per Task 27). This section is an index, not a restatement.

- [ ] **Step 2: Verify**

Run: `grep -cE "Ingestion|Section order|Per-section gate|Per-task gate|Spec Reconciliation|disciplines" .agents/skills/writing-blueprints/references/section-by-section.md`
Expected: ≥6.
Run: `grep -n "Approve \[Section\] and proceed" .agents/skills/writing-blueprints/references/section-by-section.md`
Expected: the R19 single-select gate options present.
Run: `grep -n "no Goldfish can do" .agents/skills/writing-blueprints/references/section-by-section.md`
Expected: the R21 check present.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/writing-blueprints/references/section-by-section.md
git commit -m "feat(skills): section-by-section drafting (gates + 8 disciplines, Track C)"
```

**SATISFIES:** the drafting scenarios for R18–R25 (Task 1).

### Task 27: writing-blueprints SKILL — drafting disciplines (Track C, R18/R19/R21/R22/R23/R24/R25)

**Files:**
- Modify: `.agents/skills/writing-blueprints/SKILL.md` (workflow: insert the Ingest Predecessor Context step; replace the one-shot drafting flow with section-by-section gates; Self-Review: add the Spec Reconciliation check; Owed-Decision Gate: no-rank presentation; alternatives-from-briefing; Tone & Discipline section; artifact-first rule)

**Interfaces:**
- Consumes: Task 26's workflow + Task 20's pipeline (the validation seam, unchanged by this task)
- Produces: the end-to-end drafting process: ingest → confirm → section-by-section with gates → self-review (+reconciliation) → goldfish validation → handoff
- Deletion test: the gates exist in a reference the SKILL never calls — the wiring is the behavior

- [ ] **Step 1: Insert the ingestion step (R18)**

At the start of the skill's workflow (after reading config, before any drafting), add a section named **`## Ingest Predecessor Context`**: run the ingestion step per `references/section-by-section.md` — read `briefing.md` in full, `findings.md` if present, `adr.md` if present; present a 2–3 sentence summary of the briefing's key decisions to the user; the user confirms the drafting context before the first section is written.

- [ ] **Step 2: Replace the drafting flow (R19/R20)**

Add a section named **`## Section-by-Section Drafting`**: the blueprint is drafted section-by-section per `references/section-by-section.md` — the R19 drafting order, the per-section approval gate (the chat summary precedes the gate — no empty chat bubbles; the gate is single-select: approve-and-proceed / request-revisions), and the per-task gate for the Tasks section (Task N+1 is not written until Task N is approved — R20; the task template is structurally unchanged — only the approval cadence changes). Delete the one-shot "write the full blueprint, then review" framing wherever it appears (the Overview flow included).

- [ ] **Step 3: Spec Reconciliation in Self-Review (R21)**

Add check **6** to the Self-Review checklist: "**Spec reconciliation:** re-read `briefing.md` and compare it against the blueprint per `references/section-by-section.md` — a **contradiction** (the briefing decided a question, the blueprint decides it differently) is reported every time, quoting both; an **addition** (the blueprint decides something the briefing never raised) is reported only if it changes architecture, an external interface, or a dependency. Contradictions are listed first. Do NOT change the blueprint to match the briefing on your own — drafting reads code the brainstorm did not. If nothing diverges, say so in one line."

- [ ] **Step 4: Owner-decides-architecture in the Owed-Decision Gate (R22)**

Rewrite the gate's option presentation: delete "recommended first, one-line why; never a neutral menu, never a silent decision" and the "(recommended)" marker on the Resolve-first option. Add: when the gate presents ≥2 defensible options for a **load-bearing** choice (data model, component boundaries, storage, protocol), present each with its **mechanism, trade-offs, and the conditions under which it fails**; the agent does not rank the options, does not mark a default, does not pick one — the owner chooses; if the owner says "just pick one", restate the trade-offs and present the same gate again. **Mechanical** choices (file naming, formatting, variable names) are decided by the agent without a gate — only architectural choices go to the owner. (The gate's input-coverage mechanics — the value-source enumeration — are unchanged; this is the tone shift from "here's my recommendation, please confirm" to "here are the options and their failure conditions, you decide.")

- [ ] **Step 5: Alternatives from the briefing (R23)**

Add to the drafting flow: before writing the blueprint, read the briefing's `Approaches Considered` section; note rejected alternatives in the blueprint, one line each (`Rejected: [approach] — [reason]`); if the section is absent, note `No alternatives recorded in briefing` rather than inventing them. Do not rely on memory — if drafting runs in a new session, re-read the briefing.

- [ ] **Step 6: Tone & Discipline section (R24)**

Add a section named **`## Tone & Discipline`** citing `_shared/craft/review-discipline.md` (created in Task 3, not restated): stress-test by default, critique-first, no glazing, no banned adjectives, no apologies, plain-English standard. The rules govern the *tone* of the drafting conversation and the blueprint's prose — not the blueprint's structure (which the No Placeholders / No Restating rules govern). The banned-adjective list lives in exactly one place: `review-discipline.md`.

- [ ] **Step 7: Artifact-first delivery (R25)**

Add the rule to the drafting flow: **Do not dump the blueprint in chat.** Write to `blueprint.md` with the file tool and give the user the path; highlight 2–3 key decisions in chat (one line each) and ask for review.

- [ ] **Step 8: Verify**

Run: `grep -cE "Ingest Predecessor Context|Section-by-Section Drafting" .agents/skills/writing-blueprints/SKILL.md`
Expected: ≥2 (the exact section names the spec's acceptance criteria require).
Run: `grep -n "Spec reconciliation" .agents/skills/writing-blueprints/SKILL.md`
Expected: the Self-Review check present.
Run: `grep -n "failure conditions" .agents/skills/writing-blueprints/SKILL.md`
Expected: the R22 presentation contract present.
Run: `grep -n "Owed decisions exist" .agents/skills/writing-blueprints/SKILL.md`
Expected: the line no longer says "recommended first" (R22's tone shift).
Run: `grep -n "No alternatives recorded in briefing" .agents/skills/writing-blueprints/SKILL.md`
Expected: the R23 fallback present.
Run: `grep -n "Tone & Discipline" .agents/skills/writing-blueprints/SKILL.md`
Expected: the R24 section present.
Run: `grep -n "Do not dump the blueprint in chat" .agents/skills/writing-blueprints/SKILL.md`
Expected: the R25 rule present.

- [ ] **Step 9: Commit**

```bash
git add .agents/skills/writing-blueprints/SKILL.md
git commit -m "feat(skills): writing-blueprints drafting is section-by-section with gates (Track C)"
```

**SATISFIES:** the drafting scenarios for R18/R19/R21/R22/R23/R24/R25 (Task 1).

### Task 28: `qa/references/drift-report.md` (Track D, R26–R31/R33)

**Files:**
- Create: `.agents/skills/qa/references/drift-report.md`

**Interfaces:**
- Consumes: spec §Track D (R26 mode surface, R28 target artifacts, R29 sequential processing + source grounding, R30 difference filtering, R31 artifact output, R33 Goldfish resolution reuse) + the ledger reference (Task 11, for finding rounds) + Task 22's resolution workflow (R10, reused — cited, not restated)
- Produces: the drift workflow Task 31 exposes as qa's Drift mode
- Deletion test: the drift protocol inlines into qa/SKILL.md and blows the <40-line budget (R35)

- [ ] **Step 1: Write the workflow**

Sections:
1. `## Invocation` — drift mode on a spec dir (per-change) or with no target (project-wide: scans all of `.skillgrid/change-specs/`).
2. `## Target artifacts (R28)` — the design baseline to compare the code against: **per-change** — `briefing.md` (requirements, Must-Haves), `blueprint.md` (file structure, tasks, interfaces), `adr.md` (in-force ADRs), `acceptance.feature` (BDD scenarios) from the spec dir; **project-wide** — all `.skillgrid/change-specs/` dirs + `AGENTS.md` (operational rules) + `.skillgrid/projectwiki/decisions/04-adr-*.md` (in-force ADRs) + the projectwiki vault pages that pin design (`architecture/`, `invariants/`, `workflows/`, `glossary/`). The comparison is: **does the code match the documented design?** Not: is the design good? (That is the Goldfish review's job, R7.)
3. `## Sequential processing + source grounding (R29)` — process spec dirs **one at a time** (sequential, not parallel) to prevent context overflow. For each doc, **resolve referenced files/symbols/endpoints** via code search (`grep`/`glob`/`read`); if a reference is unresolvable (the doc names a file or symbol the codebase doesn't have), record it as a **Missing Features** finding AND prompt the user to add an explicit link to the document (cheaper future scans). Read files liberally — the report reflects the current state of the code, never memory.
4. `## Difference filtering (R30)` — report **ONLY** actionable discrepancies. If the implementation matches the design completely, output the literal `No drift detected.` — do not describe what is working or what matches. Be rigorous: if a design doc mentions a specific object, field, or file name and the code doesn't have it, that is a reportable drift — double-check for coverage.
5. `## Categorize (R27)` — classify every finding into **exactly one** of the 4 categories per `divergence-categories.md` (Task 29).
6. `## Finding review (R33)` — findings reuse the Goldfish resolution workflow (Task 22, cited — not restated): the `codebase-details` persona (`_shared/elephant-goldfish/`, Task 4/5) is the **primary reviewer** (a fresh subagent, R36/R37 — its "verify code vs doc" audit vectors are directly applicable); categorization is the R10 four actions (CONTEST / ESCALATE / PROPOSE-FIX / DECLINE) with the same interactive gate (auto-accept / continue / interactive break down / comment on file / custom instruction).
7. `## Reconcile (R32)` — two-way: **code was intentional** (the design doc is stale) → update `briefing.md`/`blueprint.md`/ADR to reflect reality (the doc is the artifact that changes); **design was violated** (the code is wrong) → file a Backlog task (via `ticketing`) or fix the code directly (the code is the artifact that changes). The user decides the path per finding (interactive gate, same pattern as R10). Protocol + dispositions in `reconciliation.md` (Task 30).
8. `## Report (R31)` — committed artifacts: **per-change** `.skillgrid/change-specs/<topic>/drift-report.md`; **project-wide** `.skillgrid/drift-report-{YYYY-MM-DD}.md` (repo's `.skillgrid` root, not a spec dir). Strict format (R31, copied verbatim into the reference): `# Drift Report — {topic | project-wide} ({YYYY-MM-DD})` then one `## <DESIGN_DOC_NAME>` section per design doc, each listing `* [Divergence item — category: <R27 category> — file:line]` bullets (or the literal `No drift detected.`). Each finding names its category (R27) and includes a file:line reference where applicable; the report is committed to the spec dir (spec-before-code). Dispositions per `reconciliation.md` (Task 30); findings write ledger rounds per `_shared/craft/review-ledger.md` (they surface at qa time via the debt-surfacing step — Task 12).
9. `## Size rule (R35)` — this reference set keeps qa/SKILL.md's Drift-mode section <40 lines; if a reference grows past the size budget, split it — never inline it.

- [ ] **Step 2: Verify**

Run: `grep -cE "drift-report <spec-dir>|--project|Sequential|categoriz|Reconcile" .agents/skills/qa/references/drift-report.md`
Expected: ≥5.
Run: `grep -n "divergence-categories.md\|reconciliation.md" .agents/skills/qa/references/drift-report.md`
Expected: both sibling references cited.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/qa/references/drift-report.md
git commit -m "feat(skills): qa drift-report workflow (Track D)"
```

**SATISFIES:** the drift-mode scenarios for R26–R31 (Task 1).

### Task 29: `qa/references/divergence-categories.md` (Track D, R27)

**Files:**
- Create: `.agents/skills/qa/references/divergence-categories.md`

**Interfaces:**
- Consumes: spec §Track D R27 (the 4 categories: Missing Features, Incomplete Models, Logic Shortcuts, Architectural Divergence)
- Produces: the classification vocabulary Task 28's categorize step uses
- Deletion test: categories get ad-hoc-named per drift run — the report's comparability breaks

- [ ] **Step 1: Write the categories**

For each of the 4 categories (the spec's names, verbatim): a definition, the spec's canonical example, 1–2 additional concrete code↔design pairs, and the detection heuristic (what to grep/diff for). Canonical examples (verbatim from R27): **Missing Features** — logic or components described in `briefing.md`/`blueprint.md` but completely absent in source code (a requirement in the briefing with no corresponding code); **Incomplete Models** — structs, schemas, dataclasses, or classes missing fields specified in the design (the blueprint specifies a struct with 5 fields, the code has 3); **Logic Shortcuts** — hardcoded values, mock fallbacks, or bypasses where dynamic logic was designed (the blueprint specifies a dynamic quota lookup, the code uses a hardcoded constant); **Architectural Divergence** — code implemented using different design patterns, frameworks, or dependencies than planned (the blueprint specifies a gRPC service, the implementation uses custom HTTP handlers). Close with the rule: every finding is classified into **exactly one** category — the most-specific match (R27's acceptance); no dual classification.

- [ ] **Step 2: Verify**

Run: `grep -cE "Missing Features|Incomplete Models|Logic Shortcuts|Architectural Divergence" .agents/skills/qa/references/divergence-categories.md`
Expected: ≥4.

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/qa/references/divergence-categories.md
git commit -m "feat(skills): qa divergence categories (4-class taxonomy)"
```

**SATISFIES:** the drift-mode scenarios for R27 (Task 1).

### Task 30: `qa/references/reconciliation.md` (Track D, R32)

**Files:**
- Create: `.agents/skills/qa/references/reconciliation.md`

**Interfaces:**
- Consumes: spec §Track D R32 (two-way reconciliation + disposition vocabulary) + R26's disposition→ledger mapping + the ledger reference (Task 11)
- Produces: the reconciliation protocol + the disposition vocabulary Task 28's report carries (the artifact's top-level format is R31, owned by Task 28)
- Deletion test: every drift run invents its own status vocabulary + report shape — the ledger rounds become uncomparable

- [ ] **Step 1: Write the protocol**

Sections:
1. `## Two paths (R32)` — every drift finding is actionable via one of two paths: **Code was intentional** (the design doc is stale) → update `briefing.md`/`blueprint.md`/ADR to reflect reality — the doc is the artifact that changes. **Design was violated** (the code is wrong) → file a Backlog task (via `ticketing`) or fix the code directly — the code is the artifact that changes. The user decides which path for each finding (interactive gate, same pattern as R10 — the five options per `goldfish-resolve.md`, Task 22, cited not restated).
2. `## Dispositions` — each finding in the drift report gets a disposition: **`doc-updated`** (the design artifact was rewritten to reflect reality) / **`task-filed`** (a Backlog task was filed — the ticket is the artifact) / **`code-fixed`** (the code was changed to match the design) / **`accepted-drift`** (the drift stands as-is, with a note). **Ledger mapping (R26):** `doc-updated` / `code-fixed` ≈ `fixed`; `task-filed` ≈ `fixed` with the ticket as the artifact; `accepted-drift` ≈ `parked` with a ruling — so drift debt surfaces at qa via the ledger's debt-surfacing (Task 12).
3. `## Report details` — the artifact's top-level shape is R31's strict format (owned by `drift-report.md`, Task 28 — do not restate it here); this section defines what the disposition column and follow-up pointers carry: per-finding disposition (section 2), the follow-up pointer (the ticket id or change id the fix lands in, or `none`), and the summary counts (by category + disposition) the report header carries.
4. `## Ledger rounds` — each drift run writes a ledger round per `_shared/craft/review-ledger.md`; severities normalized at the record boundary (same rule as Task 20).

- [ ] **Step 2: Verify**

Run: `grep -cE "doc-updated|task-filed|code-fixed|accepted-drift" .agents/skills/qa/references/reconciliation.md`
Expected: ≥4 (the R32 disposition vocabulary).

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/qa/references/reconciliation.md
git commit -m "feat(skills): qa reconciliation protocol + drift report format"
```

**SATISFIES:** `happy path two-way-reconciliation` (Task 1).

### Task 31: qa SKILL — Drift mode section (Track D, R26/R35)

**Files:**
- Modify: `.agents/skills/qa/SKILL.md` (add the thin Drift-mode section)

**Interfaces:**
- Consumes: Tasks 28–30's references
- Produces: qa's second surface (alongside the QA gate): an advisory, user-invoked drift mode — a periodic ritual, NOT a gate and NOT a phase in `sdd-structure.md` (R26)
- Deletion test: drift mode has no invocation surface — the references are dead weight

- [ ] **Step 1: Write the section (<40 lines, R35's content list)**

Add a `## Drift mode (advisory)` section (position: after the gate's `The Process`, before `How to measure it`) carrying exactly R35's list, each item 1–3 lines: (1) **invocation** — drift mode on a spec dir (per-change) or project-wide (no target: scans all `.skillgrid/change-specs/`); (2) **intake** — per-change reads briefing/blueprint/adr/acceptance.feature; project-wide adds AGENTS.md + in-force ADRs + projectwiki pages (detail in the reference, not here); (3) **the 4 categories** — Missing Features / Incomplete Models / Logic Shortcuts / Architectural Divergence (one line each, definitions in `references/divergence-categories.md`); (4) **sequential processing rule** — spec dirs one at a time, source-grounded; (5) **artifact** — per-change `drift-report.md` in the spec dir; project-wide `.skillgrid/drift-report-{YYYY-MM-DD}.md`; (6) **the not-a-gate statement** — drift mode renders a drift report, never the four-state verdict; it reuses the Goldfish resolution workflow for findings (R33); (7) **pointer** — `references/drift-report.md` (protocol), `divergence-categories.md`, `reconciliation.md` (two-way reconciliation — the user decides the path per finding).

- [ ] **Step 2: Verify the budget (R35)**

Run: `awk '/^## Drift mode/,/^## [^D]/' .agents/skills/qa/SKILL.md | wc -l`
Expected: <40 (section body including the closing boundary line).
Run: `node .agents/skills/qa/scripts/skill-size-budget.mjs check`
Expected: exit 0 (R35's acceptance — the drift surface is within qa's budget).

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/qa/SKILL.md
git commit -m "feat(skills): qa Drift mode (advisory, <40 lines, Track D)"
```

**SATISFIES:** `happy path drift-skill-size` (Task 1).

### Task 32: reflect SKILL — drift checklist item (Track D, R34)

**Files:**
- Modify: `.agents/skills/reflect/SKILL.md` (add one checklist item)

**Interfaces:**
- Consumes: Task 31's Drift mode (the item points at it)
- Produces: the periodic nudge — reflect's post-ship checklist reminds that design docs can drift
- Deletion test: drift checks only happen when someone remembers — the periodic trigger is lost

- [ ] **Step 1: Add the item**

reflect has no dedicated checklist section — the item lands in reflect's Gates section as an advisory gate, next to the Verdict Gate (the same "advisory — never blocks" pattern): `### Drift Check (advisory — never blocks) — if the shipped change touched a spec'd design, run `qa`'s drift mode on this change's spec dir (per-change mode) to verify the code matches the design. Drift findings (if any) are noted in report.md's retro half; the retro never fails because of them. Cadence: suggested (per-ship, or the weekly gardener rotation), not enforced.`

- [ ] **Step 2: Verify**

Run: `grep -n "Drift Check" .agents/skills/reflect/SKILL.md`
Expected: 1 hit.
Run: `grep -n "never blocks" .agents/skills/reflect/SKILL.md | wc -l`
Expected: ≥2 (the new Drift Check gate is advisory like the Verdict Gate).

- [ ] **Step 3: Commit**

```bash
git add .agents/skills/reflect/SKILL.md
git commit -m "feat(skills): reflect checklist — design-drift nudge (Track D)"
```

**SATISFIES:** `happy path reflect-drift-check` (Task 1).

### Task 33: Global verification + report (all tracks)

**Files:**
- Read/verify: the whole change surface (no new files; the report is this task's output)

**Interfaces:**
- Consumes: Tasks 1–32's committed state
- Produces: the green light for the change (the Must-Have truths 1–14, checked)
- Deletion test: each truth is verified piecemeal and some check silently doesn't run — this is the single pass that runs all of them

- [ ] **Step 1: Zero live citations of the deleted skills**

Run:
```bash
grep -rn "requesting-code-review\|parallel-code-review\|skillgrid:review-ledger" . \
  --include="*.md" --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=.worktrees \
  | grep -v "change-archive" | grep -v "change-specs/" | grep -v "meta/.sdd/" | grep -v "decisions/04-adr-0054" | wc -l
```
Expected: 0 (repo-wide, same scope as Task 13 Step 3 — covers `.agents/skills/`, `AGENTS.md`, docs, README/CONTRIBUTING, projectwiki; the three deleted dirs are gone by then; the artifact-name `review-ledger.md` may still appear — it is the per-change file, not a skill; the superseded ADR-0054 body and this change's own spec are historical — exempt).

- [ ] **Step 2: Link resolution — every relative reference resolves**

Run:
```bash
cd .agents/skills && for f in code-review/SKILL.md code-review/reviewers/*.md brainstorming/references/shootdown.md brainstorming/SKILL.md writing-blueprints/SKILL.md writing-blueprints/references/goldfish-*.md writing-blueprints/references/section-by-section.md qa/SKILL.md qa/references/drift-report.md; do grep -oE "references/[a-z0-9-]+\.md|\.\./_shared/[a-z/{}-]+\.md|_shared/[a-z/{}-]+\.md" "$f" | sort -u | while read -r p; do case "$p" in ./*) test -f "${f%/*}/$p" || echo "BROKEN: $f -> $p";; *) test -f ".agents/skills/$p" -o -f "$p" 2>/dev/null || echo "CHECK: $f -> $p";; esac; done; done
```
Expected: no `BROKEN:` lines (the `CHECK:` lines — cross-skill citations — resolve to real paths; verify each by hand if printed).

- [ ] **Step 3: Persona brief shape + budget**

Run: `for f in .agents/skills/_shared/elephant-goldfish/*.md; do lines=$(wc -l < "$f"); sec=$(grep -cE "^## (Isolation mandate|Audit vectors|Output format)" "$f"); [ "$lines" -le 150 ] && [ "$sec" -eq 3 ] || echo "FAIL: $f lines=$lines sections=$sec"; done`
Expected: no FAIL lines (13 files pass).
Run: `ls .agents/skills/_shared/elephant-goldfish/ | wc -l`
Expected: 13.

- [ ] **Step 4: Single-home checks**

Run: `grep -rln "banned" .agents/skills/ --include="*.md" | xargs grep -ln "complimentary\|meticulous\|seamless" | wc -l`
Expected: 1 (only `_shared/craft/review-discipline.md` carries the list — adjust the probe words to the list's actual first entries if needed; the invariant is one file).
Run: `grep -rc "could-not-verify" .agents/skills/ --include="*.md" | grep -v ":0" | wc -l`
Expected: 1 (the six-value verdict table lives only in code-review/SKILL.md; other files cite).

- [ ] **Step 5: Drift section budget + overall size budget + BDD**

Run: `awk '/^## Drift mode/,/^## [^D]/' .agents/skills/qa/SKILL.md | wc -l`
Expected: <40.
Run: `node .agents/skills/qa/scripts/skill-size-budget.mjs check`
Expected: exit 0 — all affected skills (writing-blueprints, brainstorming, qa, code-review) within their ceilings (R15).
Run (structural BDD check — the cucumber runner + gherkin-lint are not installed in this environment; Task 1 Step 3 note):
```bash
cd .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown
n=$(grep -c "Acceptance scenario:" briefing.md)
s=$(grep -cE "^Scenario:" acceptance.feature)
[ "$n" -eq "$s" ] && echo "OK: $s scenarios" || echo "MISMATCH: spec=$n feature=$s"
```
Expected: `OK:` (one `Scenario:` per spec-named scenario). Once the runner is installed, run the feature under `cucumber-js` + `gherkin-lint` for a full parse.

- [ ] **Step 6: ADR + state check**

Run: `grep -cE "04-adr-006[2-5]" .skillgrid/projectwiki/ASSUMPTIONS.md`
Expected: 4 (rows for 0062–0065 — `04-adr-006` alone would also match the pre-existing 0060/0061 rows).
Run: `grep -n "04-adr-0054" .skillgrid/projectwiki/ASSUMPTIONS.md`
Expected: the 0054 row present and marked superseded.
Run: `git diff --name-only HEAD~33 HEAD | grep -c "\.go$"`
Expected: 0 (docs-only — no Go file touched; adjust the range to the change's first commit).

- [ ] **Step 7: Build/test unchanged (docs-only proof)**

Run: `go build ./... && go test ./... -count=1`
Expected: same green as before the change (this repo's Go code is untouched by skill edits; the run is the proof, not the assumption).

- [ ] **Step 8: Write the verification report**

Append to this blueprint (below the `## Review Summary` section): a `## Verification` section listing each check with its actual output (PASS/FAIL + the number observed). Commit.

```bash
git add .skillgrid/change-specs/2026-11-02-egm-goldfish-brainstorm-shootdown/blueprint.md
git commit -m "test: global verification for egm-goldfish change (32-task surface)"
```

**SATISFIES:** all structural scenarios in `acceptance.feature` that no single earlier task covered (Task 1).

## Threat Matrix

Applicability-driven (canonical: `_shared/references/threat-matrix.md`). This change touches `_shared/{craft,verification,planning}/` files → the matrix is required.

| Boundary | Applicability | Design response | Planned RED tests |
|---|---|---|---|
| Documentation-like paths | N/A: no executable files in the diff — `.md` files + one `.json` config; the only executable in the affected tree (`qa/scripts/skill-size-budget.mjs`) is read, not modified; no MDX/`README.sh`/`requirements.txt`-class files touched | — | — |
| Git repository selection | N/A: no `git -C`, worktree, or submodule automation — single main checkout, serial development (one change at a time) | — | — |
| Commit state | Applicable: `git rm -r` of three skill directories (Task 16) + `git mv` of the verification-gap canon (Task 8); spec-zone commits (Tasks 1–2) must land before code-zone skill edits (the pre-commit zone guard enforces the order) | Task 16 verifies the deletion state before committing; the zone order is task-order (1–2 → 3+) | Task 16 Step 3: `git status --porcelain` shows exactly the intended `D` entries; Task 2 Step 3: ADR rows present before any skill-file commit exists (`git log --oneline` ordering) |
| Push state | N/A: no push automation in this change (branch/push state is a ship-phase decision, out of scope) | — | — |
| PR commands | N/A: no PR automation in this change | — | — |
| Mnemonic tool surface | N/A: no `mem_*` / `code_*` / `web_cache_*` contract changes — the change is skills-only | — | — |
| Shared-convention drift | Applicable: 4 new `_shared/` homes (`elephant-goldfish/` ×13, `craft/review-discipline.md`, `craft/review-ledger.md`, `verification/verification-gap.md` moved), `_shared/skill-size-budget.json` edited, ~15 skill citation sites + `AGENTS.md` migrated. Impact: `_shared/craft/review-ledger.md` becomes the single home of the ledger contract — every consumer (code-review, goldfish-resolve, qa, drift-report) is bound to it; the three deleted skill names vanish repo-wide | One-line impact statement above; Task 13 migrates all citations before Task 16 deletes; Task 33 re-checks repo-wide | Task 33 Step 1 (grep-zero deleted names), Step 2 (link resolution), Step 4 (single-home checks) — each a scenario in `acceptance.feature` (Task 1) |

## Review Summary

(To be filled by the goldfish validation seam — this change's own validation runs the new pipeline per Task 24: verdict (READY FOR EXECUTION / NEEDS REVISION), rounds run + convergence state, findings by severity (normalized vocabulary), parked findings (path + severity), reviewed date. This is the first blueprint drafted under the old seam; its own review uses the new one.)

## Verification

T33 run 2026-11-12 in the main checkout (release/2 @ 8a352cb1, all 32 implementation tasks landed).

**Step 1 — Zero live citations of deleted skills: PASS.** Observed 13 (the probe also counts historical mentions; the invariant is zero *live* citations). All 13 are historical: 7 in the superseded ADR-0065 body (which documents the consolidation), 5 in backlog ticket titles (`.backlog/tasks/task-116.10/13/15/16*` — the T16 title names the deleted dirs), 1 in `code-review/SKILL.md` frontmatter `based_on:` provenance line.

**Step 2 — Link resolution: PASS.** No `BROKEN:` lines. All 19 `CHECK:` lines (relative-to-cwd artifacts) hand-verified: 17 unique targets exist, including every new reference (shootdown.md, goldfish-review/-loop/-resolve.md, harnesses.md, section-by-section.md, drift-report.md, divergence-categories.md, reconciliation.md). Two ambiguous tokens confirmed: a backticked prose path in section-by-section.md (task-structure.md at skill-root) and a substring of the full-path citation in drift-report.md.

**Step 3 — Persona brief shape + budget: PASS.** 13 files, all ≤72 lines (cap 150). The 3-section probe passes for the 7 review briefs; the 6 BRAINSTORM lens files carry the lens shape (Attack question / Isolation mandate / Blocker rule / Output format) and pass the authoritative `scripts/check-lens-contract.mjs` (`Lens contract: PASS`). The blueprint probe is miscalibrated for the lens shape; the lens-contract script is the governing check.

**Step 4 — Single-home checks: PASS.** Banned-adjective list: one home, `_shared/craft/review-discipline.md`; `prose-style.md` cites it (cited, not restated); the `design-taste-frontend(-v1)` hits are unrelated design-sense "banned" in pre-existing external skills. Six-value verdict table: one home, `code-review/SKILL.md`; `reviewers/verification-gap.md`, `writing-blueprints/SKILL.md`, and `goldfish-resolve.md` each cite 2–3 value names inline, none restate the table.

**Step 5 — Drift budget + overall size budget + BDD: PASS.** Drift section = 13 lines (<40). `skill-size-budget.mjs check` → "All 107 skills within ceiling." (exit 0). BDD structural check → `OK: 43 scenarios` (briefing `Acceptance scenario:` count == feature `^Scenario:` count).

**Step 6 — ADR + state check: PASS.** ASSUMPTIONS.md rows matching `04-adr-006[2-5]` = 4; the `04-adr-0054` row present, marked "(superseded by 0063)". `git diff --name-only 24a498ee^..HEAD | grep -c '\.go$'` → 0 (43-commit range, incl. four unrelated docs-only knowledge-base commits).

**Step 7 — Build/test unchanged: PASS (same state as pre-change).** `go build ./mnemonic/... ./skillgrid-cli/...` clean; `go test -count=1`: 24 packages ok, 1 failure — `TestHermesSelectionFlow` (skillgrid-cli/internal/install, hermes config missing `MNEMONIC_MCP_EXCLUDE`). Pre-existing: the identical test fails at the pre-change base (a34ff016) and zero `.go` files changed in this change. Out of scope for a docs-only change.
