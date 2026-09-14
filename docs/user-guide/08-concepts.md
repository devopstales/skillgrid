# Concepts

The recurring ideas across skills and hooks. These are the named seams that make the pipeline work — the same term appears in multiple skills and means the same thing.

## The practices (why the pipeline exists)

The whole hub is an expression of four engineering practices. Everything below is the *mechanism*; this section is the *intent*.

### Spec-Driven Development (SDD)

The pipeline is SDD: `onboarding → interviewing → writing-blueprints → slicing → [approval gate] → apply ⇄ qa → ticketing`.

- **Specs are the source of truth**, not the chat. The change's contract lives under `.skillgrid/specs/…` (`briefing.md`, `acceptance.feature`, `blueprint.md`, `tasks.md`) and is what verify/QA trace back to.
- **A written change contract before code.** Blueprint + sliced tasks + Gherkin acceptance exist and are approved *before* implementation starts.
- **A human approval gate** after spec, never auto-skipped.
- **Evidence-based verify.** The QA gate runs oracles and traces to scenarios; findings re-enter apply.
- **Durable, resumable state** (`state.md`, `checkpoint.json`) so a change survives compaction.

SDD is what makes "done" a claim with evidence instead of an assertion.

### Intent-Driven Development (IDD)

IDD is the *front half* of the pipeline: lock the **why** and the user-visible end state *before* any design or code, by grilling ambiguity out of the idea. `interviewing` (based on mattpocock's *grill*) is the engine, and the **brainstorming hard approval gate** is where intent is made explicit and approved.

- **Intent is locked first.** "Interview the user relentlessly about a plan, decision, or idea until you reach a shared understanding." The outcome, the boundary, the constraints, and the acceptance criteria are settled *before* a blueprint is written.
- **A design tree, not a flat Q&A.** "Map the problem as a **design tree**: every decision branches into the decisions that hang off it." The **frontier** is every decision whose prerequisites are already settled — "the questions you can ask *now* without guessing at answers you haven't heard yet." Ask the whole frontier in one round, wait, recompute, next round. A question whose answer depends on another still-open question belongs to a *later* round.
- **Rules:** "Finding facts is your job, never the user's." "The decisions are the user's. … Never decide for them." "One round at a time."
- **The domain model is written as you go.** Terms are written to the glossary the moment they resolve; load-bearing decisions clear the ADR bar and get recorded there. "A glossary that only changes after the interview is a summary, not the session's output." The interview ends with the glossary and ADRs *already* reflecting what was decided, not a to-do to backfill.
- **Exit is the clarity gate** (not just "no more questions"). The interview is "not done when the frontier is empty. It's done when the frontier is empty **AND** the clarity gate passes" — see [The Clarity Gate](#the-clarity-gate-interviewing).

IDD is what makes the rest of the pipeline *worth doing*: a blueprint, slice, and gate only mean something if the intent behind them was actually agreed first.

### Test-Driven Development (TDD)

`test-driven-development` — "Write the test first. Watch it fail. Write minimal code to pass." Core principle: "If you didn't watch the test fail, you don't know if it tests the right thing."

**The Iron Law:** "NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST. Write code before the test? Delete it. Start over. … Delete means delete. Implement fresh from tests. Period." "Violating the letter of the rules is violating the spirit of the rules."

**Red → Green → Refactor:** RED (write a failing test, verify it fails *correctly*) → GREEN (minimal code, verify all green) → REFACTOR (clean up, tests still green) → next.

Used **always** for new features, bug fixes, refactoring, and behavior changes. Exceptions only with your human's sign-off: throwaway prototypes, generated code, configuration. "Thinking 'skip TDD just this once'? Stop. That's rationalization."

TDD is the *per-task* engine; the `#### Gates` oracles and the goal-backward QA check are the *change-level* evidence on top of it.

### Vertical slices

A slice is a **narrow but COMPLETE** path through every layer — "UI → API → DB → config" — not a horizontal layer dump. "Vertical, not horizontal." "All DB, then all API, then all UI" is the anti-pattern.

This is the sizing unit for `slicing` (tracer-bullet tickets, ~500–1500 lines, each fitting one fresh agent context) and for the work-unit commit ("~100 lines as the soft ceiling … a vertical slice that must move together is fine"). Slices keep context small, keep commits atomic, and keep each unit independently testable and revertable.

### Behavior-Driven Development (BDD)

`acceptance-test-authoring` — Gherkin-in-Markdown `acceptance.feature` describes observable behaviour; it is **always on** ("non-negotiable, like TDD — `bdd.enabled` is always `true`").

BDD is the *language of the contract*: the scenarios are what the QA traceability matrix and the `#### Gates` oracles verify against. "Every requirement has ≥1 happy + edge + failure scenario." Specs use domain (glossary) language, never implementation jargon, so the behaviour stays readable to a non-engineer.

**How the five fit together:** IDD locks the *intent* first (grill + clarity gate + brainstorming approval). SDD provides the *pipeline* and the *written contract*. BDD provides the *behavioural language* of that contract. TDD is the *per-task* proof engine. Vertical slices are the *size unit* that keeps every one of them inside a context window. A slice is a thin SDD unit, written as BDD scenarios, proved by TDD, verified by the gates — all in service of the intent IDD locked at the front.

## The PIV loop

The high-level loop the whole pipeline follows: **prime → plan → implement → validate → review → commit → PR.** "prime" = onboard/interview (lock intent); "plan" = blueprint + slice; "implement" = apply with TDD; "validate" = the QA gate; "review" = code review; "commit" = work-unit commit; "PR" = ticketing/branch finishing. Every skillgrid phase is a step of this loop — the loop is the spine, the skills are the muscles.

## Verification-before-completion

"Evidence before claims, always." The principle that the entire review discipline operationalizes: **"Agent completed → VCS diff shows changes, NOT agent reports 'success'."** A task is not done because the agent said done — it is done because the diff, the freshly-run gate, and the traceable test prove it. "Do Not Trust the Report" (in the review discipline) is this rule applied to subagent output. No completion claim without fresh verification evidence, in the same message.

## Domain model and the shared `.skillgrid/` home

The pipeline's shared "domain model" home. `architectural-decision-records`: "Actively build and sharpen the project's domain model as you design."

```text
.skillgrid/
├── config.yaml
├── glossary/            # business.md + technical.md (domain model)
├── adr/                 # 0001-slug.md … sequential, monotonic, never reused
├── sdd/                 # checkpoint.json, debug/, gate-stop-state.json
└── specs/YYYY-MM-DD-<topic>/
    ├── state.md
    ├── briefing.md
    ├── acceptance.feature
    ├── blueprint.md
    ├── tasks.md
    ├── adr.md           # ADR Review Manifest (pointers only)
    ├── findings.md      # consolidated spike/sketch contract
    ├── research.md
    ├── test-plan.md
    └── qa-report.md
```

### Glossary

A glossary and nothing else — "no implementation details, no specs, no scratch. This is the rule that breaks in the field: left unchecked, models treat 'write to the glossary' as permission to persist every answer."

- `.skillgrid/glossary/business.md` — domain, product, workflow terms.
- `.skillgrid/glossary/technical.md` — architecture, platform, protocol terms (and the codebase-design vocabulary, seeded here).
- Multi-context repos add a `CONTEXT-MAP.md`.
- "A glossary that only changes after the interview is a summary, not the session's output." `interviewing` writes terms inline as they resolve.
- BDD specs use glossary terms, never implementation jargon.

### Codebase-design vocabulary (deep modules)

`writing-blueprints/references/codebase-design.md` seeds a fixed vocabulary into `.skillgrid/glossary/technical.md` that must be used exactly — "don't substitute 'component,' 'service,' 'API,' or 'boundary.'"

| Use | Avoid |
|-----|-------|
| **Module** | unit / component / service |
| **Interface** | API / signature |
| **Seam** (Michael Feathers: "a place where you can alter behaviour without editing in that place") | boundary |
| **Adapter**, **Depth**, **Leverage**, **Locality** | — |

Two falsifiable design gates:

- **The deletion test:** "Imagine deleting the module. If complexity vanishes, it was a pass-through. If complexity reappears across N callers, it was earning its keep." A blueprint task creating a module must name, in its `Interfaces` field, what complexity reappears if removed.
- **The one-adapter rule:** "One adapter means a hypothetical seam; two adapters mean a real one. Don't introduce a seam (port) unless at least two adapters are justified (typically production + test)."
- **Design It Twice:** for a hard interface, spawn 3+ subagents producing radically different interfaces, then compare.

## The Clarity Gate (interviewing)

The interview is "**not** done when the frontier is empty. It's done when the frontier is empty **AND** the clarity gate passes."

Score each dimension 0.0 (completely unclear) to 1.0 (crystal clear):

| Dimension | Weight | Minimum | What it measures |
|-----------|--------|---------|------------------|
| Goal Clarity | 35% | 0.75 | Is the outcome specific and measurable? |
| Boundary Clarity | 25% | 0.70 | What's in scope vs out of scope? |
| Constraint Clarity | 20% | 0.65 | Performance, compatibility, data requirements? |
| Acceptance Criteria | 20% | 0.70 | How do we know it's done? |

**Clarity score** = `1.0 − (0.35×goal + 0.25×boundary + 0.20×constraint + 0.20×acceptance)`. The formula is inverted: a *low* score is the pass condition. **Gate:** clarity ≤ 0.20 AND all dimensions ≥ their minimums → shared understanding reached. On fail, ask targeted questions at the lowest dimension, re-score, repeat. A dimension below its minimum is marked **⚠** and treated as an assumption. Final scores land in the briefing's **Clarity Report**.

## Brainstorming: four classified paths + hard approval gate

`brainstorming` classifies every idea into one path and applies a **HARD-GATE**: "Do NOT invoke any implementation skill, write any code, scaffold any project, or take any implementation action until you have told your human partner what you intend and they have approved it. … the ceremony scales with the task; the approval gate never does."

| Path | Definition | Terminal state |
|------|------------|----------------|
| **Spike** | A feasibility question ("can we…") whose output is an answer, not code you keep. **Tiny** (answer inline, no file) or **Real probe** (delegate to `spike`; anything built stays throwaway). | A reported recommendation. |
| **Bounded** | A well-scoped change to code that already exists (a flag, small endpoint, one-file fix). "If there is no existing flow to change, the task is not bounded." | Implement directly; no spec/plan doc. |
| **New Project** | Greenfield, a new subsystem, or a restructure of how components fit. | `docs/PRD.md` + `docs/ARCHITECTURE.md` + spec `briefing.md`. |
| **New Function** | A new feature/endpoint/capability in a project whose architecture is in place. | Spec `briefing.md` (+ update global PRD/ARCHITECTURE only if they exist AND change). |

**The ratchet:** "When in doubt between two paths, take the heavier one. The ratchet is one-way: hidden complexity discovered mid-task upgrades the path — stop, say so, and step up. Nothing downgrades mid-task."

The two full paths (New Project / New Function) create `state.md` at the first phase transition and append at every transition; the only skill invoked after is `writing-blueprints`. Spike/bounded skip `state.md` — they're short by design. Just-in-time offers (not upfront): visual companion, `sketch` (only when 2+ meaningfully different options whose choice depends on *feeling* it), `research` (first external fact not in the codebase).

## Must-Haves + one-way-door (writing-blueprints)

**Must-Haves** are "the observable outcomes that MUST be true when the plan is complete. The verifier checks the result against this list, not just that tasks exist." Three categories, every item verifiable:

- **Truths** — observable behaviors that must hold.
- **Artifacts** — files that must exist with real implementation, not stubs.
- **Key links** — critical connections between artifacts that must work together.

"If you can't state a truth, artifact, or link for a spec requirement, the requirement is ambiguous — go back to the spec."

**One-way-door** = "hard to reverse: it needs a migration, breaks a published contract, changes a public API shape, or is otherwise expensive to undo." Tag at the top of the task: `> ⚠ one-way: <decision>` and **STOP to get explicit user approval before proceeding**. All one-way-door decisions are listed in the blueprint's Must-Haves under "One-way-door decisions" (or "None"). `work-unit-commits`: "One-way-door decisions still get an ADR … the `Decisions:` line is the per-task record, not a replacement for the ADR."

## Tracer bullets + execution waves (slicing)

"Break a blueprint into vertical tracer-bullet tickets that each fit a single fresh agent context window. Vertical, not horizontal. Each ticket is a narrow but COMPLETE path through every layer (UI → API → DB → config)."

**Sizing:** "A ticket is the right size when a single agent session can implement it, test it, and commit it without needing to re-read the blueprint. Rough guide: **~500–1500 lines of change (20–50% tests)**." Template size label: `~<lines> (S <500 / M 500-1500 / L >1500)`.

**Dependency edges:** "Dependency = implemented, not sliced. Ticket B is blocked by ticket A until A is done, not until A is written down." Each ticket carries `- **Blocks:**` / `- **Blocked by:**` lines. A later-wave ticket waits for its dependency to be *complete* (reviewed, committed), not just started.

**Waves (algorithm):**

1. **Wave 1:** all tickets with no blockers (run in parallel)
2. **Wave 2:** tickets blocked only by Wave 1 tickets
3. **Wave N:** tickets blocked by Wave N−1 tickets

Encoded in `tasks.md` as a mermaid `Dependency Graph` block plus an **Execution Order** section (`Wave 1 (parallel): TICKET-01, …`). Self-review check: "no ticket appears in a wave before its blockers are in earlier waves."

## The 4-state QA gate (qa)

"The gate is the decision, not the test suite." "Task completion ≠ goal achievement. Passing tests ≠ verified behavior."

| Verdict | Criteria |
|---------|----------|
| **PASS** | ALL of: all truths VERIFIED; all scenarios covered by a test that ran and passed; no CRITICAL findings; no MISSING_RED; all code-quality gates PASS or N/A; P0 pass rate ≥ `quality.p0_pass_rate`; P1 ≥ `quality.p1_pass_rate`; coverage ≥ `quality.coverage_min` (if > 0); mutation ≥ `quality.mutation_min` (if > 0); no Trivy finding ≥ `security.trivy.fail_on` (if set). |
| **CONCERNS** | No CRITICAL findings and all hard gates PASS, BUT ≥ 1 of: a PRESENT_BEHAVIOR_UNVERIFIED level, a WARNING finding, a P0 without triangulation, a low-risk missing adoption, or a lint warning. Open items named with a path to resolution. |
| **FAIL** | ANY of: a truth UNVERIFIED; a scenario with no covering test; a test that ran and failed; a MISSING_RED; a regression gap with no covering test; a missing-oracle gap (happy-path scenario with no `G<n>`); a CRITICAL security finding; a code-quality gate FAIL; a Trivy finding ≥ `security.trivy.fail_on`. |
| **WAIVED** | The human explicitly waived a specific gate criterion. Waiver + criterion + accepted risk recorded in the report. **WAIVED is never a machine decision.** |

**Default config thresholds** (`quality:`, when unset): `coverage_min` 80, `mutation_min` 80, `p0_pass_rate` 100, `p1_pass_rate` 95. Trivy: `severities` "CRITICAL", `scan_types` "vuln,secret,misconfig", `fail_on` "" (= report only). **A threshold of 0 means disabled — a gate with threshold 0 is N/A, it cannot FAIL.**

**The Trivy security gate** (PASS item 10 / the FAIL row): `fail_on` empty means Trivy findings are reported but can never fail the gate; set `fail_on` to a severity (e.g. "CRITICAL") and any finding at or above it FAILs the gate. This is the machine-enforced security decision; the 6-specialist parallel review (Security specialist) is the human-facing lens over the same findings.

**Six hard rules:** (1) A human decision always overrides the machine verdict. (2) A change that fails with no human decision is recorded as **not accepted**, never silently accepted. (3) A non-empty `pending` list in the traceability matrix makes the gate **FAIL**, including headless. (4) Passing tests do not substitute for running the system — the goal-backward check is the evidence, not the suite count. (5) A code-quality gate at threshold 0 is N/A. (6) An `ABANDON`-ed gate is a handoff, never a pass — it keeps the gate unmet, routing to FAIL or WAIVED on the human's explicit decision.

**Fix-loop cap:** "3 rounds. If the gate is still FAIL after 3 rounds of fixes, stop and escalate. The finding is likely not a test gap — it's an architectural or scope problem that needs a human decision." Routing: PASS → code review (or `parallel-code-review` for 50+ lines / high-risk); CONCERNS → fix/defer/human-look, re-run; FAIL → fix each with a failing test first; WAIVED → record, proceed.

## Goal-backward verification (qa)

"Force stance: Assume the goal was NOT achieved until codebase evidence proves it. Your starting hypothesis: tasks completed, goal missed. Falsify the implementation narrative."

Four levels, one named test per level:

| Level | Question | Evidence that passes |
|-------|----------|----------------------|
| **Truth** | Is the falsifiable claim actually true? | A test/scenario that exercises the behavior end-to-end and passed. |
| **Artifact** | Does the named file exist, is it substantive, is it wired in? | File with real content (not a stub) + an import/registration connecting it. |
| **Key Link** | Is the wiring between components actually exercised? | A test that calls through the link and asserts the output at the far end. |
| **Data Flow** | Does real data move through the system as claimed? | A test with a concrete input producing an observable output at the boundary. |

Statuses: **VERIFIED** (a test exercised it and passed); **PRESENT_BEHAVIOR_UNVERIFIED** (code exists and is wired, but no test exercises the state transition / cleanup / ordering invariant — "Never counts as VERIFIED. Routes to human."); **UNVERIFIED** (no evidence found). "Presence is not behavior. A grep that confirms a function exists is not verification. A test that calls it and asserts the output is."

## Verification gap + traceability (qa)

"Find changed behavior that could break without reliable verification catching it." One question: *"If this behavior broke where it's actually used, would verification fail?"*

Gap shapes: **Regression gap** (regresses where used, no covering test fails); **Missing-adoption gap** (a place that should now use the new behavior doesn't — caller not updated, flag not flipped, migration not run); **Broken-verification gap** (a test appears to cover it but is skipped/flaky/not-run/too weak — mock-only, snapshot-only, asserts the mock not the code); **Missing-oracle gap** (a happy-path scenario with no `G<n>` — "a scenario with no oracle is a traceability gap, not a completed behavior").

**Evidence rule:** "Read the test before claiming what it covers. A test name is not evidence. Search the whole repo by symbol + import references before claiming no test exists. Run the test … to confirm it actually executes. **Triage trusts a gap finding as filed.**" Each finding names a **Smallest Regression** — the specific input/state/call sequence that would break.

**Traceability:** matrix = every `acceptance.feature` scenario → covering test → did it run? → pass/fail? "A scenario with no test = traceability gap. A test that ran and failed = CRITICAL finding. A test that exists but was skipped or filtered = treat as `pending`. **Never edit the expectation to match the code.** If a test disagrees with the matrix, fix the code."

## Gates (`#### Gates`) + the Iron Law

Every requirement's happy-path scenario carries a `G<n>` oracle — the verifiable definition of done. "NO COMPLETION CLAIMS WITHOUT FRESH VERIFICATION EVIDENCE. If you haven't run the verification command in this message, you cannot claim it passes."

| Form | Meaning |
|------|---------|
| **Runnable** | `G<n>` with `CHECK:` (a repo-owned command) + `EXPECT:` (a success-only marker) + `EVIDENCE: pending`. Met when CHECK exits 0 AND output matches EXPECT (substring, or `/.../` POSIX ERE, optional `i`), **freshly run**. |
| **Manual** | No command can decide; `G<n>` + `EVIDENCE: pending`, reviewed proportionally to risk. |
| **ABANDON** | `G<n>: ABANDON <non-empty reason and handoff>`. "Terminal and non-successful; it is a handoff, never a pass." |

"The declared oracle is the oracle. A scenario is met **only** when its `G<n>` CHECK exits 0 and its output matches EXPECT, freshly run. 'I ran *some* test' is not verification. You may not claim 'done' while any happy-path scenario's gate is unmet or abandoned. Report met / unmet / abandoned counts."

`test-driven-verification` authors them; `gate-state.sh` evaluates them (`--status` non-executing / `--reverify` executing); `gate-stop.sh` blocks the agent's Stop while any is unmet; `gate-lint.sh` lints structure at commit time. `ponytail` re-binds the "check" rule to these gates: "Lazy code without its check is unfinished … a non-trivial task MUST leave its happy-path scenario's gate green." See [Hooks](04-hooks.md).

## BDD extraction contract (acceptance-test-authoring)

Gherkin-in-Markdown: "Markdown headings carry the capability, requirement, and scenario structure, while ` ```gherkin ` fences contain only Given/When/Then steps. The runner extracts them into real `.feature` files on every run, synthesizing `Feature:`/`Rule:`/`Scenario:` from the headings." "This file is the definition. `javascript/extract-gherkin.cjs` is a binding of it. When behavior changes, this file changes first."

| Markdown line (outside any fence) | Emitted Gherkin line |
|-----------------------------------|----------------------|
| `# <title>` (the single H1) | `Feature: <title>` |
| `### Requirement: <name>` | `  Rule: <name>` |
| `#### Scenario: <name>` | `    Scenario: <name>` |
| `#### Scenario Outline: <name>` | `    Scenario Outline: <name>` |
| any line inside a ` ```gherkin ` fence | copied **verbatim**, column unchanged |
| everything else (prose, other headings, fence markers) | blank line |

**Line fidelity invariant:** "Every input line maps to exactly one output line … the extracted file has the IDENTICAL line count and **line N of the `.feature` is line N of the `.md`**." "Never 'improve' the extractor to collapse blank lines." Fence opener = 3+ backticks at column 0, info string **exactly** `gherkin`; steps indented 6 spaces verbatim. **Hard errors:** no H1; more than one H1; a `#### Scenario:` with no fence before the next heading/EOF; `Feature:`/`Rule:`/`Scenario:`/`Scenario Outline:`/`Example:` inside a fence; indented fence opener. Zero gherkin fences is fine (prose-only spec).

**Scenario classification** is by naming prefix, not Gherkin tags: `happy path` / `edge` / `failure`. "Every requirement has ≥1 happy + edge + failure scenario." Priority `@p0`/`@p1` and trace markers `@step-NN` live in the blueprint/tasks `SATISFIES` field, not in the Gherkin.

**Zone rule (recurring):** "BDD is always on (non-negotiable, like TDD) — `bdd.enabled` is always `true`." `.skillgrid/specs/` is always the **spec zone**; the rest is the **code zone**. "Edit `.skillgrid/specs/` or code in a single commit — never both uncommitted. Commit specs before code." Enforced by the `pre-commit` zone guard.

## ADRs: styles, immutability, manifest (architectural-decision-records)

**IRON RULE:** "ADRs are immutable once accepted. You MUST NOT edit a prior accepted ADR under any circumstance — not its status, not its body, not its date. The accepted ADR is a frozen historical record."

| `adr_style` | Template | Shape |
|-------------|----------|-------|
| `madr-full` | madr-full.md | Detailed tradeoff record |
| `madr-minimal` (default) | madr-minimal.md | Context / Considered Options / Decision Outcome / Consequences |
| `nygard` | nygard.md | Classic: Status / Context / Decision / Consequences |
| `y-statement` | y-statement.md | One-sentence decision |
| `custom` | custom.md | Project-specific |

**Fixed machine-readable header** (all styles): `status`: `proposed | accepted | deprecated | superseded`; `date`: `YYYY-MM-DD`; `supersedes`: `ADR-NNNN` (only when replacing a prior in-force ADR). "The header is fixed; the body is not." This is what makes the in-force set computable.

**In-force derivation:** "An ADR is in force when its status is `accepted` **and** no later ADR's `supersedes` points at it." Consumers walk `supersedes` links. Numbering: `0001-slug.md`, sequential, monotonic, never reused. All ADRs live in the single `.skillgrid/adr/` folder.

**ADR bar (offer only when all three true):** (1) Hard to reverse, (2) Surprising without context, (3) Result of a real trade-off. "If any of the three is missing, skip the ADR and say which test failed."

**ADR Review Manifest:** one per change at `…/specs/…/adr.md`. "The change's ADR completion marker and the read source for downstream skills. It holds *pointers* only — never duplicate a repo ADR's Context / Decision / Consequences." Sections: **In-Force ADRs Reviewed** / **New Durable ADRs Created** / **Supersessions**. "If nothing meets the bar, say so explicitly … Don't invent ADRs to fill the manifest."

## The research firewall + epistemics + type packs (research / deep-research)

**The research firewall:** "Project context — briefs, specs, code, memory, the glossary — shapes *what to ask*, never *what is true*. It is inadmissible as evidence: every claim in the findings file traces to a source you fetched. The firewall is why a fact your codebase *assumes* still gets re-verified against the source that owns it." In `deep-research` it's stronger: "Every researcher subagent runs behind it: it gets its brief and nothing else — no project files, no ambient context."

**Epistemics (two standing rules):** (1) "Never conclude from training data alone. What you already know proposes hypotheses, queries, and structure — it is not evidence. Conclusions require evidence retrieved *this run*. A claim you cannot evidence is stated as `unverified` or dropped, never asserted." (2) The research firewall.

**Sourcing:** "Primary sources only … Follow every claim back to the source that owns it." "A claim is a sentence with a source — publisher, publication date, access date. No naked numbers." "Answer engines (Perplexity, Grok, and kin) are aggregators too — chase their citations and cite *those*, never the engine." "Thin public data is reported as thin; absence of evidence is a finding; freshness is part of truth."

**Three type packs** (`research/types/`), each with Dimensions / Craft / Freshness bars / Two-source classes:

| Type | Use when | Freshness bars |
|------|----------|----------------|
| `technical` | adopt a tech, design an integration, ground an architecture, assess feasibility | versions/compat ≤ 1 mo · ecosystem signals ≤ 6 mo · landscape ≤ 12 mo (AI-adjacent ≤ 3 mo) · patterns ≤ 2 yr |
| `competitive` | position against named competitors, battlecard | pricing/features ≤ 3 mo · trajectory ≤ 6 mo · sentiment ≤ 12 mo |
| `domain` | learn a domain's rules, constraints, vocabulary | regulations/protocol ≤ 6 mo · operational patterns ≤ 12 mo · conventions ≤ 2 yr |

Confidence per claim: **high** (verified, fresh, credible) / **medium** (single credible source, fresh) / **low** (stale, weak publisher, or disputed) / `unverified`. Findings land at `{specs_root}/YYYY-MM-DD-<topic>/research.md`; "every load-bearing claim is cited inline `[n]` and resolves in the source appendix."

**The boundary between research / spike / sketch / deep-research:** research = a fact not in the codebase, one inline pass; spike = needs code executed; sketch = needs *feeling* a layout; deep-research = wide or high-stakes.

## Deep-research roles: researcher / verifier / red-team

`deep-research` fans out parallel researcher subagents, then verifies load-bearing claims and red-teams the conclusions before synthesizing a cited findings file.

- **Researcher** — runs behind the firewall; returns a **digest, not raw results** — findings as claims `{claim, source, publisher, pub_date, accessed, confidence, class}`, plus leads and what it could not find. "Write each digest to `<run-folder>/digests/` the moment it lands — the conversation is a control channel, never the store." A two-source-class claim carries `needs_second_source: true` (the verifier closes it, not the researcher).
- **Verifier** (fresh context, at landing, per dimension) — checks the load-bearing claims against an **independent source** (different publisher, different underlying data). Outcomes: **verified** / **disputed** ("independent sources materially disagree; report both, both cited, never averaged") / **unverified** / **overturned** ("corrected, original noted"). "Verification happens as material lands, per dimension — never as an end-of-run rewrite pass."
- **Red-team** (fresh-context skeptic) — given the conclusion and a budget but **no supporting evidence and no run context**; hunts disconfirming evidence. Verdict: **holds | weakened | overturned**. "Zero findings after a real search is itself reportable." Runs off for low-stakes single-dimension runs; on by default for high-stakes conclusions.

**Effort presets:** `quick` (2 subagents, 5 sources/dim, 1 round) / `standard` default (3, 8, 2) / `deep` (6, 12, 3). "The user's explicit request beats the preset. `depth` is a cap, not a quota." Topologies: breadth-first (split dimensions) / depth-first (split angles) / straightforward (one subagent, no fan-out).

## Spike: verdict + liftable pure module

Verdict (3 states, evidence-gated): `VALIDATED ✓` / `INVALIDATED ✗` / `PARTIAL ⚠`. Evidence rule: "a verdict is a claim with a demonstration. If you cannot point to specific output, a log line, a measured number, or a screenshot … the verdict is `PARTIAL` at best with an `unverified` flag — never `VALIDATED` by vibes." VALIDATED requires evidence beyond a single happy-path test (at least one edge case exercised, or a comparison spike).

**The liftable pure module:** "The one thing you lift into the real codebase is a **pure module**." Rules: pure (no DOM/fetch/side effects); labeled (`## Liftable Module` with path, I/O signature, dependencies); named for the real codebase (glossary vocabulary); no spike-specific constants (inject as params). "If the question is 'can we build the whole feature this way end-to-end,' you may want a tracer bullet instead; a spike stays small and single-purpose."

## Sketch: 2–3 variants, winner + constraints

"Build throwaway interactive UI mockups to answer a 'does this layout/interaction feel right?' question. Produces 2–3 dramatically different variants the user can switch between, a marked winner, and the constraints for the real build." Build **2–3 variants** in one HTML file (tab-switchable), "meaningfully different" in the first round (not the same layout with different colors); "Never more than 4." Viewport previews: **375px / 768px / 1280px**. Prefer adjusting an existing page (`?variant=` param or dev toggle) over a throwaway standalone file. Synthesis is first-class ("A's layout, C's palette" → labeled synthesis variant).

## Work-unit commits: `[skillgrid-context]` + atomic sizing

"Turn 'commit after every task' into an *enforced* checkpoint: a conventional, atomic, independently-revertable commit that carries a machine-readable `[skillgrid-context]` block, guarded by git hooks, with a derived `checkpoint.json` resume handle. The commit is the durable record; the JSON is the resume pointer. They cannot drift because the JSON is derived from history."

```
type(scope): concise description

- key change 1
- key change 2

[skillgrid-context]
Task: <task or ticket id>
Decisions: <key choices this unit made, and why>
Remaining: <what is left in this logical unit — empty if done>
Tried: <failed approaches worth recording — omit line if none>
[/skillgrid-context]
```

**Conventional-commit subject regex:** `^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([a-z0-9._-]+\))?!?: .+` — `type` required, `(scope)` optional lowercase, `!` marks breaking change. "No `Co-Authored-By`, `Generated-By`, or any AI attribution trailer."

**Atomic sizing:** "One logical change per commit." "**~100 lines as the soft ceiling**" (a vertical slice that must move together is fine). "**Independently revertable:** prefer additive changes; do not delete and replace in the same commit — split into a delete commit and an add commit." "**Spec before code (zone rule).**" "Commit **after** a verified gate (a green `G<n>` — not 'I think it's done'), and **before** a long-running install/build/test command."

`checkpoint.json` (`skillgrid/checkpoint/v1`) is derived from `git log -1`; never hand-edit. Snapshot via `checkpoint-state.sh snapshot` (idempotent); restore via `restore`. **Commit first, then review** — "Review diffs `merge-base...HEAD`; an uncommitted change is invisible to it."

## The three state layers + resume

"Conversation memory rots (compaction, session death, context overflow). Files do not. When state files and your recollection disagree, **the files win, every time.**" "The conversation is a control channel, the files are the store." Mnemonic is "a fallback index, not a layer."

| Layer | File | Answers | Lifetime |
|-------|------|---------|----------|
| **Phase** | `.skillgrid/specs/YYYY-MM-DD-<topic>/state.md` | What phase, what's done/open/decided? | Committed; survives `git clean -fdx` |
| **Task** | `.skillgrid/sdd/<plan>/progress.md` | Which task done? What was ruled? | Git-ignored; deleted when review clean |
| **Code** | `.skillgrid/sdd/checkpoint.json` (derived from git log) | Where in code did last unit leave off? | Derived; always reconstructable |

`state.md` format: `Phase: <spike | interview | design | blueprint | slicing | execution | review | done>` + `Done:` / `Open:` / `Decisions:` / `Execution:` + a `## Handoff` narrative paragraph. Resume: "On resume: read the state file first (skillgrid:resume); resume at the first `open` item."

## Structured debugging: four phases + the 3-fix rule

**Iron Law:** "NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST. If you haven't completed Phase 1, you cannot propose fixes." "ALWAYS find root cause before attempting fixes. Symptom fixes are failure."

1. **Root Cause Investigation** — read errors carefully, reproduce consistently, check recent changes, instrument each boundary, trace data flow (fix at source, not symptom).
2. **Pattern Analysis** — find working examples, compare against references, identify differences.
3. **Hypothesis and Testing** — "Form a single hypothesis … 'I think X is the root cause because Y.' Test minimally, one variable at a time. Didn't work? Form a NEW hypothesis. DON'T add more fixes on top."
4. **Implementation** — "Create a failing test case (MUST have before fixing) … Implement a single fix. No 'while I'm here' improvements. Verify the fix with `test-driven-verification`."

**The 3-fix rule:** "If < 3: Return to Phase 1, re-analyze. **If ≥ 3: STOP and question the architecture** — DON'T attempt Fix #4 without architectural discussion." Red flag: **"'One more fix attempt' (when already tried 2+)."** "This is NOT a failed hypothesis — this is a wrong architecture." The debug state trail is written to `.skillgrid/sdd/debug/<date>-<slug>/state.md` (`hypothesis | evidence | verdict`), git-ignored, survives compaction. "On root cause: if the cause is a design decision, record an ADR; if it is a bug, the TDD evidence in the fix commit is the durable record."

This "3 rounds then escalate" mirrors the QA fix-loop cap — both cap iteration at 3 and route to a human/architecture question.

**Scope-lock during debugging:** "recurring bugs in the same files are an architectural smell" — if fixes keep landing in the same place, stop patching and question the design. The debug discipline pairs with a **freeze/scope-lock**: block edits outside the area under investigation (fail-closed — a tool call that can't be parsed is denied, not allowed) so the debugger can't accidentally "fix" unrelated code and mask the real cause.

## Ponytail ladder + check rule

"Channels a senior dev who has seen everything, and questions whether the task needs to exist at all (YAGNI)." The ladder (7 rungs, stop at the first that holds): (1) Does this need to exist at all? (YAGNI) → (2) Already in this codebase? → (3) Stdlib does it? → (4) Native platform feature covers it? → (5) Already-installed dependency solves it? → (6) Can it be one line? → (7) Only then: the minimum code that works. Intensities: `lite` / `full` (default) / `ultra`.

**The check rule:** "Lazy code without its check is unfinished. In the skillgrid flow the 'check' is … the `#### Gates` block the task's requirements carry in `acceptance.feature`." **Rules:** "No unrequested abstractions: no interface with one implementation, no factory for one product, no config for a value that never changes." "Deletion over addition. Boring over clever." Mark deliberate simplifications with a `ponytail:` comment naming the ceiling and upgrade path. (The design-side twin of the one-adapter rule.)

## Skill authoring as TDD

"Writing skills IS Test-Driven Development applied to process documentation. If you didn't watch an agent fail without the skill, you don't know if the skill teaches the right thing. NO SKILL WITHOUT A FAILING TEST FIRST … Write skill before testing? Delete it." The description field is "When to Use, NOT What the Skill Does" — descriptions that summarize the workflow create a shortcut agents will take. Two conventions that recur in skillgrid's skill format come from here: **close every loophole explicitly** ("Violating the letter of the rules is violating the spirit of the rules") and the **Red Flags / Rationalizations table** — the "These thoughts mean STOP — you're rationalizing" lists at the bottom of load-bearing skills. This is why skillgrid skills read the way they do: they were written TDD-style, against observed agent failures.

## `findings.md` — consolidated spike/sketch contract

"The consolidated file is the single downstream contract that `writing-blueprints` reads — it must contain every spike's and sketch's verdict, what's liftable, and the constraints for the build." At `.skillgrid/specs/YYYY-MM-DD-<topic>/findings.md`: spike appends `## Spike: NNN-name` (Verdict / What we learned / What's liftable / Constraints for the build); sketch appends `## Sketch: NNN-name` (Winner / Rationale / What's liftable / Constraints / Mode). "Every design decision in the blueprint that rests on a feasibility result or a chosen layout must cite it."

## Branch finishing

When the QA gate passes and the branch is clean, the agent presents exactly three options and lets the human choose — "the integration decision is theirs": **1. Merge back to the base branch locally**, **2. Push and create a Pull Request**, **3. Keep the branch as-is.** Discarding the branch requires the user to type the word `discard` to confirm ("Only the typed word `discard` authorizes deletion"). Worktree cleanup is provenance-based — "Never `--force` on your own initiative." This is the tail of the PIV loop (`commit → PR`): the agent never auto-merges, auto-pushes, or auto-discards.

## The `SATISFIES` traceability link

The explicit wire between BDD scenarios, tasks, and the QA matrix. Every blueprint task and every slice ticket carries a `**SATISFIES:** <scenario-name>` pointing at the `acceptance.feature` scenario it makes green. "The `SATISFIES` scenario names in `tasks.md` are the traceability oracle for `skillgrid:qa` — every scenario must be referenceable from a test. Name them precisely."

**Acceptance-first (BDD is always on):** "every ticket carries a `SATISFIES: <scenario-name>` pointing at its acceptance scenario … The scenario's RED state is confirmed before the implementation that turns it green. Order scenario/RED tickets ahead of their implementation tickets in the graph." A blueprint task's field: "the acceptance scenario in `acceptance.feature` this task makes green — BDD is always on."

The `@p0`/`@p1` priority and `@step-NN` trace markers live in the blueprint/tasks `SATISFIES` field, **not** in the Gherkin (the extracted Gherkin carries no tags — see [The BDD extraction contract](#bdd-extraction-contract-acceptance-test-authoring)). Scenario names must be unique and referenceable.

This is how a change is traced end-to-end: scenario (BDD) → task `SATISFIES` → slice ticket → QA traceability matrix → the `G<n>` gate that proves it.

## The review discipline

### Two-axis review

`requesting-code-review` runs **two** `general-purpose` subagents **concurrently** (do not pollute each other's context): **Standards** — "Does the code follow this repo's documented standards?" — and **Spec** — "Does the code implement what was asked?"

"Present the two reports under `## Standards` and `## Spec` headings, side by side, verbatim or lightly cleaned. End with one line per axis: finding count + worst issue *within* that axis. **Never pick a single cross-axis winner** — that's the reranking the separation exists to prevent." "A change can be spec-perfect but violate every convention, or beautifully written but implement the wrong thing; the two axes are deliberately separate."

**Escalation threshold:** "if the whole-branch diff is large (50+ changed lines) or high-risk (auth, data migration, money, concurrency, public API), escalate the final review from the two-axis pass to `skillgrid:parallel-code-review`."

### The smell baseline

A fixed set of 12 Fowler code smells (*Refactoring*, ch.3) that applies even when the repo documents nothing: Mysterious Name, Duplicated Code, Feature Envy, Data Clumps, Primitive Obsession, Repeated Switches, Shotgun Surgery, Divergent Change, Speculative Generality, Message Chains, Middle Man, Refused Bequest. Two binding rules: "**The repo overrides.** A documented standard (glossary/ADR) always wins; where it endorses something the baseline would flag, suppress the smell." "**Always a judgement call.** Each is a labelled heuristic, never a hard violation. Skip anything tooling (lint) already enforces."

### The no-subagents contract

"the implementer never dispatches subagents — not helpers, and never a reviewer. Review arrives from you, after the report. In real sessions, every reviewer a worker spawned duplicated the task review the controller dispatched anyway — a full extra review seat per task." "A reviewer you spawn duplicates that review at full cost, and its approval counts for nothing in the process."

### "Do Not Trust the Report"

"Treat the implementer's report as unverified claims about the code. It may be incomplete, inaccurate, or optimistic. Verify the claims against the diff. Design rationales in the report are claims too … is the implementer grading their own work." "Judge the code on its merits — a stated rationale never downgrades a finding's severity."

**"plan-mandated":** "If the plan or brief explicitly mandates something this rubric calls a defect … that IS a finding — report it as Important, labeled plan-mandated. The plan's authorship does not grade its own work; the human decides."

**"⚠️ Cannot verify from diff":** requirements that live in unchanged code or span tasks. "These do not block the rest of the review, but you must resolve each one yourself before marking the task complete … If you confirm an item is a real gap, treat it as a failed spec review — it enters the fix loop with the other findings."

### Triage (receiving-code-review)

"Sort findings before touching code. If direction is unclear, surface them grouped and ASK rather than fixing everything by default." Four buckets: **Fix now** (real, in-scope, belongs with this change), **Defer** (real but later; log as a tracker issue), **Human look** (needs manual inspection before trusting), **Noise** (won't-fix or misread; say why, drop it). "Don't let the reviewer dictate scope — 'real, but later' is a valid and common call." Fix order: "Blocking issues → Simple fixes → Complex fixes. Test each fix individually. Verify no regressions. **Cap at 3 rounds** — beyond that, surface the remaining items to the human with the rulings you made and why." Forbidden responses: any gratitude expression ("You're absolutely right!", "Great point!", "Let me implement that now") before verification.

**YAGNI Check:** "IF reviewer suggests 'implementing properly': grep codebase for actual usage. IF unused: 'This endpoint isn't called. Remove it (YAGNI)?' IF used: Then implement properly."

### The 5-round fix loop + breaker + adjudication

subagent-execution: "A fix round is one fix dispatch plus one scoped re-review. **Five rounds maximum per task.**" Rounds 1–3 resume the original implementer (context intact), sending the open findings verbatim. Rounds 4–5 dispatch a fresh implementer on a more capable model. "The breaker. When round 5's re-review still leaves findings open, stop dispatching. **Adjudicate each open finding yourself.**" Three adjudication outcomes: **park** (reviewer wrong), **park** (real but not load-bearing), **rule** (real and load-bearing — a later task builds on it). "Adjudicate only at the cap. Adjudicating earlier to end a loop is pre-judging with a different name. Every adjudication is a ledger entry — a silent discard is forbidden." "Never fix findings yourself in the controller session — your context stays clean for coordination, and controller fixes skip review."

### The 6-specialist parallel review

`parallel-code-review` fans out six specialist reviewers in parallel: **Standards, Spec, Edge cases, Verification gaps, Security, Red team**. Sizing: "< 50 changed lines → run only Standards + Spec (the two-axis set). ≥ 50 changed lines, or high-risk → run all six." "Red team runs last: dispatch it after the other five return, and hand it their merged findings so it hunts for what they missed, not what they found." "The diff is passed as a **path**, not inlined text." Resilience: "if a specialist fails … log its name and continue with the survivors. If **all** specialists fail or return empty, do **not** claim a clean review."

**Deduplicate by fingerprint** (`path:line:category`); findings from 2+ specialists pointing at the same defect collapse into one entry, tagged `MULTI-REVIEWER CONFIRMED (a + b)`. "Two findings belong in one entry only when the same defect produced both." The entry's severity is the highest of its members. **The 5-verdict triage** (`high` / `medium` / `low` / `false` / `maybe-false`): "The coordinator verifies; specialists don't grade — they lack the full context to grade severity." Reject any `false`; any `low` whose fix adds more complexity than the harm; or any whose only fix is to edit the spec under review.

The specialist lenses: **Edge Case Hunter** (mechanical path/boundary enumeration — control flow, domain boundaries, *implicit branches*, handle lifetime, call-site mismatches, and a `deletion` check; "Do NOT assign severity"); **Verification Gap Hunter** (the 3 gap shapes + "the smallest concrete regression the consumer would observe"); **Red Team** (reads the merged findings, hunts the surface the others didn't cover — integration boundaries, failure modes, ordering, assumptions, "the boring stuff"); **Security** (injection, authn/authz/IDOR, data exposure, crypto, deserialization, supply-chain, concurrency/TOCTOU — "Name the concrete attack or exposure — 'improves security' is not a finding").

## The Depth Tree (Decompose & Gate)

subagent-execution decomposes a plan into **leaves (tasks) under branches (shared interfaces)** and gates dispatch on verified dependencies, not file order.

- **Inventory the outcomes:** "Reread the plan/spec and list every independently-omittable outcome. A plan whose tasks omit an outcome nobody owns is a half-done plan."
- **File ownership:** `Task <N>: Owns: <repo-relative globs>` and `Task <N>: Needs: <task ids or interfaces it builds on>`. "These are coordination metadata, not a filesystem sandbox — a task may still read outside its `Owns`, but it may not write to another in-flight task's `Owns`."
- **Dependency-gated dispatch:** "Dispatch a task only when every task named in its `Needs:` has a `Task <M>: complete` ledger line."
- **Branch integration gate:** "If two or more tasks share an interface or file, name the task that *integrates* them and give it an extra `G<n>` gate: the child tasks' outputs compose and the shared interface holds end-to-end."
- **Parent re-verify:** "A leaf's self-pass does not count as verification. Before marking a task complete, re-run its `#### Gates` oracles yourself."

**The Concurrent Coupled Build** (parallel-execution) is the sibling for 2+ tasks that must run at once AND touch overlapping files/interfaces. Four conditions (all must be true): 2+ implementation tasks, touch same file(s) or shared interface, write sets can be made **disjoint**, want them in flight at the same time. "If any of these fail — especially 'write sets can be made disjoint' — do not parallelize. Fall back to `skillgrid:subagent-execution`." "Never dispatch a leaf whose `Owns:` intersects another in-flight leaf's `Owns:`. A wave is a set of leaves whose `Needs:` are all satisfied." The parent re-verifies: re-run Gates, re-verify the integration gate, confirm no leaf wrote outside its `Owns:`. Keep it light — "it does not track lease state, wave state, or a `dispatch.json`. That machinery only pays off at 4+ leaves in flight." The **fan-out ledger** (`parallel-ledger.md`, one row per agent `agent | task | status | changed files | test result`) is the store: "summaries returned to your context rot with it … Integrate from the ledger + git, not from in-context summaries."

**The implementer report contract:** an implementer reports one of four statuses — `DONE | DONE_WITH_CONCERNS | BLOCKED | NEEDS_CONTEXT` — each with a distinct controller response (DONE → build the review package; DONE_WITH_CONCERNS → read the concerns first; NEEDS_CONTEXT → provide context and re-dispatch; BLOCKED → assess and re-dispatch with a more capable model or break the task). "Never ignore an escalation or force the same model to retry without changes. If the implementer said it's stuck, something needs to change."

**Controller rules of the loop:** **Rulings, not stalls** — "A running plan does not wait on a human … the spec is the binding authority, the plan is its argument, and your judgment settles what neither answers. Record every decision in the ledger as `Ruling: <what> — <why> — <what it costs if wrong>`." Four stop conditions (only these): an irreversible/destructive operation, a security-sensitive action, a side effect outside this worktree, or a plan so broken that every path is a guess. **Continuous execution** — "Do not pause to check in between tasks … 'Should I continue?' prompts and progress summaries waste their time." **Batch small same-shape work** — compose ONE dispatch for several small, same-kind edits; reserve one-dispatch-per-task for work needing its own judgment/tests/review surface. **Bounded waits** — "never poll … with short timeouts, and never sit in one silent, open-ended wait either … wait in bounded stretches (five to ten minutes)." **Pre-flight review** — before Task 1, scan the plan once for conflicts, "writing down what you checked as you check it"; "the scan's output is a table, not a verdict." **The ledger** (`progress.md`) is the recovery map: "the commits it names exist in git even when your context no longer remembers creating them."

## The ticketing rules

`ticketing` maps SDD work to the tracker and runs a status machine. The **status machine**: `backlog → ready → in-progress → review → done` — `backlog` (set by `ticketing` at publish), `ready` ("all blockers done", set at wave start), `in-progress` (set by `subagent-execution` / `simple-execution`), `review` (implementation done), `done` ("review clean, merged", set by `requesting-code-review` on a clean review). Epic: `backlog → in-progress (first ticket starts) → done (all tickets done)`.

**The 5 canonical triage roles** (each maps to a label/status string in the active tracker): `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. "If the user's tracker already uses different label names, record the override per-role in `.skillgrid/config.yaml` — do not rename the roles."

**Backlog completeness gate** (when the tracker is Backlog.md, a ticket is not published until all four are present): **Type** (frontmatter `type:`), **References** (frontmatter `references:`), **Definition of Done** (body `## Definition of Done`), **Implementation Plan** (body `## Implementation Plan`) — plus Description, Acceptance Criteria, and `priority:`. "SDD tickets: seed References to `briefing.md` / `tasks.md` / `acceptance.feature`; seed Plan from the blueprint steps; seed DoD from project defaults. Thin one-line description stubs are **forbidden**."

**Pre-submission privacy review** (mandatory, every tracker): scan the body immediately before publishing and replace environment-specific data with placeholders — project names → `<project-name>`, usernames → `<user>`, hostnames → `<hostname>`, keys/tokens/passwords → `<token>`/`<password>`, internal ports/IPs → `<host>:<port>`. "Do NOT redact intentionally public identifiers … Rule of thumb: if the reader can run the reproduction after the replacement, sanitization is correct."

**Per-phase lifecycle** (creation alone is not enough — every later execution phase owns a tracker update): start → `in-progress` + "apply began" comment; per commit → footer `Refs: <ID>`; review → `review` + verdict comment; done → `done`, closing commit uses `Closes <ID>`. "If no tracker ID exists … skip all tracker mutations. Do not invent an id."

**Title convention:** `[TYPE] Brief description (COMPONENT)`, where `[TYPE]` ∈ `{BUG, FEATURE, ENHANCEMENT, REFACTOR, DOCS, CHORE}`. "Title `[TYPE]` is not a substitute for frontmatter `type:` — both are required." **Priority** (4-tier): Critical (production down, data loss, security), High (blocks users, no workaround), Medium (workaround, subset of users), Low (cosmetic, internal). **Slicing strategy per tracker:** Backlog.md one task per ticket parented to an epic; GitHub one issue per ticket as a sub-issue; GitLab `/blocked_by` links; Jira domain = epic, ticket = story, acceptance = sub-tasks.

## The Mnemonic protocol

`mnemonic` (SQLite + FTS5, single `skillgrid` binary) — persistent memory, code index, web cache.

**The orientation ladder** (primary repo navigation; "Do **not** dump the whole tree as primary orientation"): 1. `code_status` (health + Index Freshness) → 2. `code_map` (structural overview) → 3. `symbols` / `outline` (narrow the unit) → 4. `code_related` (neighbors via Edges) → 5. `code_read` (exact slice). "`code_read` is **only** for a path + line range already narrowed by map/symbols/related/search. Never read a whole file speculatively."

**Index freshness:** `code_status` reports `freshness` (`fresh` | `lag` | `empty` | `unknown`) — "Unreadable mtime/hash → `unknown` (never false-clean)." "After clone, branch switch, large edits: check status; reindex when not `fresh`." `stale=true` is **empty-index compat only**; lag is `freshness=lag`, not silent "fine."

**The search intent router:** retrieval "is not 'six equal corpora' as the only model. The **Search Intent Router** picks a daily path; advanced corpora are an escape hatch." Identifier-shaped query → `symbols`; structural / `func $NAME` / matcher → `grep`; decision / remember / past work → `mem`; unclassified → `code`. "Multi-corpus results must keep **provenance** (mem vs code never silently fused unlabeled). The `semantic` corpus must never pretend full quality when the Local Code Embedder is unavailable — never silent degrade to FTS-as-semantic."

**The deterministic artifact naming convention:** "ALL Skillgrid artifacts persisted to Mnemonic MUST follow this deterministic naming: `title: skillgrid/{YYYY-MM-DD-<topic>}/{artifact-type}`, `topic_key: skillgrid/{YYYY-MM-DD-<topic>}/{artifact-type}`, `type: architecture`, `scope: project`." "`title` and `topic_key` must be identical — exact-match recovery depends on it." Artifact types include `briefing`, `blueprint`, `tasks`, `spec`, `ticketing`, `execution-progress`, `review-report`, `state`, `tech_stack`, etc. "Upserts: same `topic_key` + `scope` → UPDATE (overwrite), not INSERT … Mnemonic is working memory, not an audit trail."

**The 2-step recovery:** "Step 1: `mem_search(query: …)` → truncated preview + ID. Step 2: `mem_get_observation(id: …)` → complete content. Search previews are always truncated; `mem_get_observation` is the only way to get full content." "When retrieving multiple artifacts, group all searches first, then all retrievals."

**Web cache:** Context7/Exa/DeepWiki/WebFetch snapshots with TTLs — "context7 720h, exa 168h, deepwiki 336h, fetch 168h", cap 256 KB. `web_cache_lookup` before the remote call; `web_cache_save` within the same turn on miss.

## The resume protocols

`resume` re-orient from durable state. "Conversation memory rots … **the files win, every time.**"

**Context-save** (the explicit pause-and-persist, when stopping or under context pressure): 1. refresh `state.md` (phase layer), 2. append current position to `progress.md` (task layer), 3. commit the work unit (code layer), 4. **narrative handoff** — "one short paragraph in `state.md` — what you were thinking, what you would do next, what to watch for. This is the one thing the structured layers do not capture", 5. Mnemonic mirror (`mem_save` for `state` + `execution-progress`), 6. commit `state.md` (spec zone).

**Context-restore:** "locate → read the layers → announce → continue." **The recovered-position announcement** (one line): "Resuming `<topic>`: phase=execution, wave 2, task 3/5 done, last commit `<sha>`. Next: task 4." "Say which source you recovered from when it was not the normal path."

**"Persist before pressure, not after loss"** (using-skillgrid): "When a session is clearly long — many subagents dispatched, large tool outputs accumulating, or the harness showing a compaction indicator — STOP and: (1) append the current position to the active `state.md` and, during execution, the plan's ledger, (2) commit the work unit, (3) if the remaining work is large, run the save path in skillgrid:resume." "State writes are dual: the in-repo file is the source of truth; when `mnemonic.enabled: true`, `mem_save` mirrors it as an index and backup. If they disagree, the in-repo file wins." "After compaction ('FIRST ACTION REQUIRED'), re-orient via skillgrid:resume before continuing — files first, mnemonic only as fallback."

## Gherkin authoring discipline

`acceptance-test-authoring` — the 6 authoring rules ("folded from the gherkin-authoring discipline"):

1. **Domain language** — "Use the project's glossary terms. Never use implementation jargon (class names, method names, DB column names) in steps."
2. **Observable outcomes** — "Then steps state what the user or system observes — a response, a state change, a message. Never assert internal state ('the cache is populated')."
3. **One behavior per scenario** — "If a scenario needs a second `And` clause that introduces a new precondition, split it."
4. **Falsifiable** — "Every scenario must be able to fail. 'The system works correctly' is not a step."
5. **Given/When/Then balance** — "A scenario with no `When` is a precondition list, not a scenario. A scenario with no `Then` proves nothing."
6. **No implementation details in scenario names** — "'Login succeeds with valid credentials' not 'POST /auth returns 200'."

**The Page Object Model:** "Step definitions must read as intent; all UI knowledge lives in page objects." Page objects live under `acceptance-tests/support/pages/`, one per screen/flow, encapsulating routes, form field names, selectors, and ids; they expose intent-level methods (`open()`, `submit_signup(...)`, `error_message()`). "Parse responses with the stack's HTML parser, never with regexes over raw HTML" (cucumber-js: `cheerio`). "Step definitions contain no selectors, regexes, or URLs; only page-object calls and assertions."

**The 5 runner invariants:** (1) `acceptance-tests/` is an independent test project; its hooks boot the app before the suite and shut it down after, so the suite runs with a single command. (2) The default run executes every `acceptance.feature` under `.skillgrid/specs/`. (3) "A green suite is the gate for archive, and archive must never change suite results." (4) Every run generates an HTML report under `acceptance-tests/reports/`. (5) Verify extraction whenever the runner config, extractor, or `.skillgrid/specs/` tree changes.

**Workflow cadence:** "Implement one pending step definition at a time: run the suite so the step fails for the right reason, implement until it passes, then commit. The suite's red scenarios at propose time are the change's work list." "Finish only when every scenario passes with zero pending or undefined steps and the HTML report is generated."

## Isolated workspace

"Detect existing isolation first. Then use native tools. Then fall back to git. **Never fight the harness.**" "A native tool (e.g. `EnterWorktree`) owns placement, branching, and cleanup. Bypassing it is the #1 mistake — it creates phantom state your harness can't see or manage." Step 0 detects before creating: `GIT_DIR != GIT_COMMON` → already in a linked worktree (skip); the **submodule guard** (`git rev-parse --show-superproject-working-tree` returns a path → you're in a submodule, treat as a normal repo); empty branch → detached HEAD ("cannot branch/push/PR from sandbox"). "Honor any existing declared preference without asking. If the user declines consent, work in place." "MUST verify the directory is ignored before creating a worktree: `git check-ignore -q .worktrees`." **Verify clean baseline:** "Run tests to ensure the workspace starts clean. A dirty baseline makes every later failure ambiguous. Proceeding past failures is your human partner's call."

## Cross-cutting invariants (short reference)

- **A rule asks; a hook guarantees.** When an invariant is non-negotiable, wire a hook — see [Hooks](04-hooks.md).
- **The smart zone (≤ ~40% context).** Keep the orchestrator under ~40% of the window; offload dumps/reads/research/implementation to subagents, files, Mnemonic. See [Multi-agent work](06-multi-agent-work.md).
- **Vertical slices over horizontal layers.** Thin end-to-end paths, not "all DB then all API then all UI."
- **Spec zone XOR code zone.** Commit `.skillgrid/specs/` before the code that satisfies it — never both uncommitted.
- **Durable state over chat memory.** Files win over conversation memory, every time.
- **Never edit the expectation to match the code.** Fix the code.

## Next step

Back to [Start here](00-start-here.md).
