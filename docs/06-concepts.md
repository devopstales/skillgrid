# Concepts

The recurring ideas across skills and hooks. These are the named seams that make the pipeline work — the same term appears in multiple skills and means the same thing. Every concept below names its v2 home: a skill, a `_shared/` file, or a hook.

## The practices (why the pipeline exists)

The whole hub is an expression of five engineering practices. Everything below is the *mechanism*; this section is the *intent*.

### Spec-Driven Development (SDD)

The pipeline is SDD: `skillgrid → brainstorm → questioning → research → prototype → blueprint → slicing → execute → test → adversarial-authoring → drift → ship` (per `_shared/rules/sdd-structure.md` § Phase Order).

- **Specs are the source of truth**, not the chat. The change's contract lives under `.skillgrid/change-specs/YYYY-MM-DD-<topic>/` (`briefing.md`, `acceptance.feature`, `blueprint.md`, `tasks.md`) and is what `test` traces back to.
- **A written change contract before code.** Blueprint + sliced tasks + Gherkin acceptance exist and are approved *before* implementation starts. Artifact creation order is permanent and sequential: briefing → blueprint → blueprint validation → cucumber test file → tasks → slices.
- **A human approval gate** after spec, never auto-skipped — the user's own hand-edit of the line-3 STATUS banner (`draft`/`proposed` → `approved`); the agent never writes `approved`.
- **Evidence-based verify.** The `test` gate runs oracles and traces to scenarios; findings re-enter execution.
- **Durable, resumable state** — `state.yaml` plus the spec-zone artifacts that name the phase — so a change survives compaction.

SDD is what makes "done" a claim with evidence instead of an assertion.

### Intent-Driven Development (IDD)

IDD is the *front half* of the pipeline: lock the **why** and the user-visible end state *before* any design or code, by grilling ambiguity out of the idea. `questioning` (the interview loop, run inline by `brainstorm`) is the engine, and the **brainstorm hard approval gate** is where intent is made explicit and approved.

- **Intent is locked first.** The outcome, the boundary, the constraints, and the acceptance criteria are settled *before* a blueprint is written.
- **A design tree, not a flat Q&A.** Map the problem as a **design tree**: every decision branches into the decisions that hang off it. The **frontier** is every decision whose prerequisites are already settled — "the questions you can ask *now* without guessing at answers you haven't heard yet." Ask the whole frontier in one round, wait, recompute, next round.
- **Rules:** "Finding facts is your job, never the user's." "The decisions are the user's. Never decide for them." "One round at a time."
- **The domain model is written as you go.** Terms are written to the glossary the moment they resolve; load-bearing decisions clear the ADR bar and get recorded. "A glossary that only changes after the interview is a summary, not the session's output."
- **Exit is the clarity gate** (not just "no more questions").

IDD is what makes the rest of the pipeline *worth doing*: a blueprint, slice, and gate only mean something if the intent behind them was actually agreed first.

### Test-Driven Development (TDD)

The canon lives in `_shared/references/strict-tdd.md`: "Write the test first. Watch it fail. Write minimal code to pass." Core principle: "If you didn't watch the test fail, you don't know if it tests the right thing."

**The Iron Law:** "NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST. Write code before the test? Delete it. Start over. Delete means delete. Implement fresh from tests. Period." "Violating the letter of the rules is violating the spirit of the rules."

**RED → GREEN → TRIANGULATE → REFACTOR:** RED (write a failing test, verify it fails *correctly*) → GREEN (minimal code, verify all green) → TRIANGULATE (a second test with different inputs against the same behavior — kills `return 42` implementations) → REFACTOR (clean up, tests still green) → next.

Used **always** for new features, bug fixes, refactoring, and behavior changes. Exceptions only with the human's sign-off: throwaway prototypes, generated code, configuration. "Thinking 'skip TDD just this once'? Stop. That's rationalization."

**The test pyramid + duplicate coverage guard** (per `_shared/references/strict-tdd.md`): select the *highest layer that fits* each behavior; if a lower layer already covers the behavior, use the lower layer (Duplicate Coverage Guard). Don't add a test at a higher layer when a cheaper layer covers the same behavior.

TDD is the *per-task* engine; the `G<n>` oracles and the goal-backward check are the *change-level* evidence on top of it.

### Vertical slices

A slice is a **narrow but COMPLETE** path through every layer — "UI → API → DB → config" — not a horizontal layer dump. "Vertical, not horizontal." "All DB, then all API, then all UI" is the anti-pattern.

This is the sizing unit for `slicing` (tracer-bullet tickets, ~500–1500 lines, each fitting one fresh agent context) and for the work-unit commit ("~100 lines as the soft ceiling … a vertical slice that must move together is fine"). Slices keep context small, keep commits atomic, and keep each unit independently testable and revertable.

### Behavior-Driven Development (BDD)

`acceptance.feature` (Gherkin-in-Markdown, authored in `blueprint` Phase 5) describes observable behaviour; it is **always on** ("non-negotiable, like TDD — `bdd.enabled` is always `true`").

BDD is the *language of the contract*: the scenarios are what the traceability matrix and the `G<n>` oracles verify against. "Every requirement has ≥1 happy + edge + failure scenario." Specs use domain (glossary) language, never implementation jargon, so the behaviour stays readable to a non-engineer.

**How the five fit together:** IDD locks the *intent* first (interview + clarity gate + brainstorm approval). SDD provides the *pipeline* and the *written contract*. BDD provides the *behavioural language* of that contract. TDD is the *per-task* proof engine. Vertical slices are the *size unit* that keeps every one of them inside a context window. A slice is a thin SDD unit, written as BDD scenarios, proved by TDD, verified by the gates — all in service of the intent IDD locked at the front.

## The phase chain (the spine)

The high-level loop the whole pipeline follows is the v2 phase chain: `skillgrid → brainstorm → questioning → research → prototype → blueprint → slicing → execute → test → adversarial-authoring → drift → ship`. `skillgrid` is the router (classify, route, resume); `questioning`/`research`/`prototype` run *inline* inside `brainstorm`/`blueprint`, not as independent stops. Every pipeline skill is a step of this chain — the chain is the spine, the skills are the muscles.

| Legacy loop step (v1 PIV) | v2 home |
|---|---|
| prime (lock intent) | `brainstorm` + `questioning` |
| plan | `blueprint` + `slicing` |
| implement | `execute` (TDD, lazy discipline, waves) |
| validate | `test` (four-state verdict → `report.md`) |
| review | `test` review area (`review.md`) + `adversarial-authoring` (model council on high-stakes artifacts) |
| commit / PR | `_shared/work-unit-commits` + `ship` |

## Verification-before-completion

"Evidence before claims, always." The principle the entire review discipline operationalizes: **"Agent completed → VCS diff shows changes, NOT agent reports 'success'."** A task is not done because the agent said done — it is done because the diff, the freshly-run gate, and the traceable test prove it. "Do Not Trust the Report" (in the review discipline) is this rule applied to subagent output. No completion claim without fresh verification evidence, in the same message.

## Domain model and the shared `.skillgrid/` home

The pipeline's shared "domain model" home (full layout: `_shared/rules/sdd-structure.md` § Directory Structure):

```text
.skillgrid/
├── config.yaml                 # REQUIRED static SoT — stack, testing, tracker, rules.*
├── state.yaml                  # DYNAMIC project state — per-change phase map
├── deferred-work.md            # cross-change deferred-work ledger (DW-<n>)
├── work/                       # LIVE execution scratch (gitignored) — ledger, briefs, attempts
├── change-specs/               # ACTIVE change artifacts (committed)
│   └── YYYY-MM-DD-<topic>/
│       ├── briefing.md         # brainstorm (falsifiable requirements, goal)
│       ├── acceptance.feature  # BDD scenarios (the traceability oracle)
│       ├── blueprint.md        # blueprint (technical plan)
│       ├── tasks.md            # slicing (tickets, waves, dependencies)
│       ├── adr.md              # ADR Review Manifest (pointers only)
│       ├── findings.md         # research/prototype/sketch consolidated evidence
│       ├── report.md           # test (QA half: test plan + gate verdict + evidence)
│       └── review.md           # review audit record; read by ship
├── change-archive/             # CLOSED changes (immutable); ship moves them here
└── projectwiki/                # durable LLM-wiki (committed, cross-change)
    ├── ASSUMPTIONS.md          # VERIFIED/INFERRED/LOCKED; first file a reader opens
    ├── index.md                # wiki catalog
    ├── decisions/              # one ADR per file (NNNN-slug.md)
    ├── glossary/               # 01-business-terms.md + 02-technical-terms.md
    └── …                       # concepts/, invariants/, gotchas/, architecture/, …
```

### Glossary

A glossary and nothing else — "no implementation details, no specs, no scratch. This is the rule that breaks in the field: left unchecked, models treat 'write to the glossary' as permission to persist every answer."

- `.skillgrid/projectwiki/glossary/01-business-terms.md` — domain, product, workflow terms.
- `.skillgrid/projectwiki/glossary/02-technical-terms.md` — architecture, platform, protocol terms (and the codebase-design vocabulary, seeded here).
- "A glossary that only changes after the interview is a summary, not the session's output." `questioning` writes terms inline as they resolve.
- BDD specs use glossary terms, never implementation jargon.

### Codebase-design vocabulary (deep modules)

`blueprint/references/codebase-design.md` seeds a fixed vocabulary into the technical glossary that must be used exactly — "don't substitute 'component,' 'service,' 'API,' or 'boundary.'"

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

## The Clarity Gate (questioning)

The interview is "**not** done when the frontier is empty. It's done when the frontier is empty **AND** the clarity gate passes."

Score each dimension 0.0 (completely unclear) to 1.0 (crystal clear):

| Dimension | Weight | Minimum | What it measures |
|-----------|--------|---------|------------------|
| Goal Clarity | 35% | 0.75 | Is the outcome specific and measurable? |
| Boundary Clarity | 25% | 0.70 | What's in scope vs out of scope? |
| Constraint Clarity | 20% | 0.65 | Performance, compatibility, data requirements? |
| Acceptance Criteria | 20% | 0.70 | How do we know it's done? |

**Clarity score** = `1.0 − (0.35×goal + 0.25×boundary + 0.20×constraint + 0.20×acceptance)`. The formula is inverted: a *low* score is the pass condition. **Gate:** clarity ≤ 0.20 AND all dimensions ≥ their minimums → shared understanding reached. On fail, ask targeted questions at the lowest dimension, re-score, repeat. A dimension below its minimum is marked **⚠** and treated as an assumption. Final scores land in the briefing's **Clarity Report**.

## Brainstorm: four classified paths + hard approval gate

`brainstorm` classifies every idea onto one path and applies a **HARD-GATE**: "no implementation until your partner says yes … the ceremony scales with the task; the approval gate never does."

| Path | Definition | Terminal state |
|------|------------|----------------|
| **Prototype** | A feasibility question ("can we…") whose output is a verdict, not code you keep — throwaway probe, delegated to `prototype`. | A reported verdict (`findings.md`). |
| **Bounded** | A well-scoped change to code that already exists (a flag, small endpoint, one-file fix). "If there is no existing flow to change, the task is not bounded." | Implement directly; short design in chat, stop. |
| **New Project** | Greenfield, a new subsystem, or a restructure of how components fit. | `00-prd.md` + `ASSUMPTIONS.md` + `ARCHITECTURE.md` + spec `briefing.md`. |
| **New Function** | A new feature/endpoint/capability in a project whose architecture is in place. | Spec `briefing.md` (+ global updates only when they actually change). |

**The ratchet:** "When in doubt between two paths, take the heavier one. The ratchet is one-way: hidden complexity discovered mid-task upgrades the path — stop, say so, and step up. Nothing downgrades mid-task."

The two full paths (New Project / New Function) run long enough to outlive a session; the phase position is read from which spec-zone artifacts exist, and each phase transition commits the artifacts that changed. Prototype/bounded-class work skips the heavy planning — they're short by design. Just-in-time offers (not upfront): the sketch (only when 2+ meaningfully different options whose choice depends on *feeling* it), `research` (first external fact not in the codebase).

## Must-Haves + one-way-door (blueprint)

**Must-Haves** are "the observable outcomes that MUST be true when the plan is complete. The verifier checks the result against this list, not just that tasks exist." Three categories, every item verifiable:

- **Truths** — observable behaviors that must hold.
- **Artifacts** — files that must exist with real implementation, not stubs.
- **Key links** — critical connections between artifacts that must work together.

"If you can't state a truth, artifact, or link for a spec requirement, the requirement is ambiguous — go back to the spec."

**One-way-door** = "hard to reverse: it needs a migration, breaks a published contract, changes a public API shape, or is otherwise expensive to undo." Tag at the top of the task: `> ⚠ one-way: <decision>` and **STOP to get explicit user approval before proceeding**. All one-way-door decisions are listed in the blueprint's Must-Haves (or "None").

## Tracer bullets + execution waves (slicing)

"Break a blueprint into vertical tracer-bullet tickets that each fit a single fresh agent context window. Vertical, not horizontal. Each ticket is a narrow but COMPLETE path through every layer (UI → API → DB → config)."

**Sizing:** "A ticket is the right size when a single agent session can implement it, test it, and commit it without needing to re-read the blueprint. Rough guide: **~500–1500 lines of change (20–50% tests)**." Template size label: `~<lines> (S <500 / M 500-1500 / L >1500)`.

**Dependency edges:** "Dependency = implemented, not sliced. Ticket B is blocked by ticket A until A is done, not until A is written down." Each ticket carries `Blocks:` / `Blocked by:` lines.

**Waves (algorithm):** Wave 1 = all tickets with no blockers (run in parallel); Wave N = tickets blocked only by Wave N−1 tickets. Encoded in `tasks.md` as a mermaid `Dependency Graph` block plus an **Execution Order** section. Self-review check: "no ticket appears in a wave before its blockers are in earlier waves."

## Gates (`G<n>`) + the Iron Law

Every requirement's happy-path scenario carries a `G<n>` oracle — the verifiable definition of done. "NO COMPLETION CLAIMS WITHOUT FRESH VERIFICATION EVIDENCE. If you haven't run the verification command in this message, you cannot claim it passes."

| Form | Meaning |
|------|---------|
| **Runnable** | `G<n>` with `CHECK:` (a repo-owned command) + `EXPECT:` (a success-only marker) + `EVIDENCE: pending`. Met when CHECK exits 0 AND output matches EXPECT (substring, or `/.../` POSIX ERE, optional `i`), **freshly run**. |
| **Manual** | No command can decide; `G<n>` + `EVIDENCE: pending`, reviewed proportionally to risk. |
| **ABANDON** | `G<n>: ABANDON <non-empty reason and handoff>`. "Terminal and non-successful; it is a handoff, never a pass." |

"The declared oracle is the oracle. A scenario is met **only** when its `G<n>` CHECK exits 0 and its output matches EXPECT, freshly run. 'I ran *some* test' is not verification. Report met / unmet / abandoned counts." The gate state engine is `omp/hooks/hooks/gate-state.js` (`--status` non-executing / `--reverify` executing); `gate-stop.js` blocks the agent's Stop while any happy-path gate is unmet; `gate-lint.js` lints gate structure at commit time. The lazy discipline (in `execute`) re-binds the "check" rule to these gates: "Lazy code without its check is unfinished … a non-trivial task MUST leave its happy-path scenario's gate green."

## BDD extraction contract (blueprint Phase 5)

Gherkin-in-Markdown: "Markdown headings carry the capability, requirement, and scenario structure, while ` ```gherkin ` fences contain only Given/When/Then steps. The runner extracts them into real `.feature` files on every run, synthesizing `Feature:`/`Rule:`/`Scenario:` from the headings." "This file is the definition. `javascript/extract-gherkin.cjs` is a binding of it. When behavior changes, this file changes first." (Contract: `blueprint/references/EXTRACTION.md`.)

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

**Zone rule (recurring):** `.skillgrid/change-specs/` is always the **spec zone**; the rest is the **code zone**. "Commit specs before code — never both uncommitted." Enforced by the `precommit-zone-guard` git hook.

## ADRs: immutability, styles, in-force set, manifest (adr-records)

**IRON RULE:** "ADRs are immutable once accepted. You MUST NOT edit a prior accepted ADR under any circumstance — not its status, not its body, not its date. The accepted ADR is a frozen historical record." (Contract + templates: `_shared/adr-records/README.md`.)

| `adr_style` | Template | Shape |
|-------------|----------|-------|
| `madr-full` | madr-full.md | Detailed tradeoff record |
| `madr-minimal` (default) | madr-minimal.md | Context / Considered Options / Decision Outcome / Consequences |
| `nygard` | nygard.md | Classic: Status / Context / Decision / Consequences |
| `y-statement` | y-statement.md | One-sentence decision |
| `custom` | custom.md | Project-specific |

**Fixed machine-readable header** (all styles): `status`: `proposed | accepted | deprecated | superseded`; `date`: `YYYY-MM-DD`; `supersedes`: `ADR-NNNN` (only when replacing a prior in-force ADR). "The header is fixed; the body is not." This is what makes the in-force set computable.

**In-force derivation:** "An ADR is in force when its status is `accepted` **and** no later ADR's `supersedes` points at it." Consumers walk `supersedes` links. Numbering: `0001-slug.md` in `{projectwiki}/decisions/`, sequential, monotonic, never reused; the in-force index is the `## LOCKED` section of `ASSUMPTIONS.md`.

**ADR bar (offer only when all three true):** (1) Hard to reverse, (2) Surprising without context, (3) Result of a real trade-off. "If any of the three is missing, skip the ADR and say which test failed."

**ADR Review Manifest:** one per change at `change-specs/<topic>/adr.md`. "The change's ADR completion marker and the read source for downstream skills. It holds *pointers* only — never duplicate a repo ADR's Context / Decision / Consequences." Sections: **In-Force ADRs Reviewed** / **New Durable ADRs Created** / **Supersessions**. "If nothing meets the bar, say so explicitly … Don't invent ADRs to fill the manifest."

## The research firewall + epistemics + freshness bars (research)

**The research firewall:** "Project context — briefs, specs, code, memory, the glossary — shapes *what to ask*, never *what is true*. It is inadmissible as evidence: every claim in the findings file traces to a source you fetched. The firewall is why a fact your codebase *assumes* still gets re-verified against the source that owns it." In fan-out mode it's stronger: "Every researcher subagent runs behind it: it gets its brief and nothing else — no project files, no ambient context."

**Epistemics (two standing rules):** (1) "Never conclude from training data alone. What you already know proposes hypotheses, queries, and structure — it is not evidence. Conclusions require evidence retrieved *this run*. A claim you cannot evidence is stated as `unverified` or dropped, never asserted." (2) The research firewall.

**Sourcing:** "Primary sources only … Follow every claim back to the source that owns it." "A claim is a sentence with a source — publisher, publication date, access date. No naked numbers." "Answer engines (Perplexity, Grok, and kin) are aggregators too — chase their citations and cite *those*, never the engine." "Thin public data is reported as thin; absence of evidence is a finding; freshness is part of truth."

**Three type packs** (`research/references/research/`), each with dimensions / craft / freshness bars / two-source classes:

| Type | Use when | Freshness bars |
|------|----------|----------------|
| `technical` | adopt a tech, design an integration, ground an architecture, assess feasibility | versions/compat ≤ 1 mo · ecosystem signals ≤ 6 mo · landscape ≤ 12 mo (AI-adjacent ≤ 3 mo) · patterns ≤ 2 yr |
| `competitive` | position against named competitors, battlecard | pricing/features ≤ 3 mo · trajectory ≤ 6 mo · sentiment ≤ 12 mo |
| `domain` | learn a domain's rules, constraints, vocabulary | regulations/protocol ≤ 6 mo · operational patterns ≤ 12 mo · conventions ≤ 2 yr |

Confidence per claim: **high** (verified, fresh, credible) / **medium** (single credible source, fresh) / **low** (stale, weak publisher, or disputed) / `unverified`. Findings land as a `## Research: <question>` section in `change-specs/<topic>/findings.md`; "every load-bearing claim is cited inline `[n]` and resolves in the source appendix."

**Fan-out roles** (the v2 `research` skill absorbs the old deep-research skill): **Researcher** (behind the firewall; returns a digest of claims with source/publisher/date/confidence, never raw results) → **Verifier** (fresh context, checks load-bearing claims against an *independent* source; outcomes verified / disputed / unverified / overturned — "verification happens as material lands, per dimension — never as an end-of-run rewrite pass") → **Red-team** (fresh-context skeptic given the conclusion but *no supporting evidence*; verdict holds / weakened / overturned — "zero findings after a real search is itself reportable").

**The boundary between research / prototype / sketch:** research = a fact not in the codebase, one inline pass; prototype = needs code executed; sketch = needs *feeling* a layout; fan-out = wide or high-stakes.

## Prototype: 3-state verdict + liftable pure module (formerly spike)

Verdict (3 states, evidence-gated): `VALIDATED ✓` / `INVALIDATED ✗` / `PARTIAL ⚠`. Evidence rule: "a verdict is a claim with a demonstration. If you cannot point to specific output, a log line, a measured number, or a screenshot … the verdict is `PARTIAL` at best with an `unverified` flag — never `VALIDATED` by vibes." VALIDATED requires evidence beyond a single happy-path test (at least one edge case exercised, or a comparison prototype). **The liftable pure module:** "The one thing you lift into the real codebase is a **pure module**" — pure (no DOM/fetch/side effects), labeled, named in glossary vocabulary, no prototype-specific constants. "If the question is 'can we build the whole feature this way end-to-end,' you may want a tracer bullet instead; a prototype stays small and single-purpose." (Protocol: `blueprint/references/prototype-phase.md`.)

## Sketch: 2–3 variants, winner + constraints (inline in blueprint)

"Build throwaway interactive UI mockups to answer a 'does this layout/interaction feel right?' question. Produces 2–3 dramatically different variants the user can switch between, a marked winner, and the constraints for the real build." Build **2–3 variants** in one HTML file (tab-switchable), "meaningfully different" in the first round (not the same layout with different colors); "never more than 4." Viewport previews: **375px / 768px / 1280px.** Prefer adjusting an existing page (`?variant=` param or dev toggle) over a throwaway standalone file. Synthesis is first-class ("A's layout, C's palette" → labeled synthesis variant, which is what gets marked the winner). Output: a `## Sketch: NNN-name` section in `findings.md` (Winner / Rationale / What's liftable / Constraints / Mode). **Full mechanic: `blueprint/references/sketch-phase.md`** — and the picked variant goes to `skillgrid:ui-design` to be finished into the deliverable.

## Work-unit commits: `[skillgrid-context]` + atomic sizing

Contract: `_shared/work-unit-commits/README.md`. "Turn 'commit after every task' into an *enforced* checkpoint: a conventional, atomic, independently-revertable commit that carries a machine-readable `[skillgrid-context]` block, guarded by git hooks. The commit is the durable record."

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

**Conventional-commit subject regex:** `^(build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test)(\([a-z0-9._-]+\))?!?: .+` — `type` required, `(scope)` optional lowercase, `!` marks breaking change. "No `Co-Authored-By`, `Generated-By`, or any AI attribution trailer." (The `commit-msg` hook blocks them.)

**Atomic sizing:** "One logical change per commit." "**~100 lines as the soft ceiling**" (a vertical slice that must move together is fine). "**Independently revertable:** prefer additive changes; do not delete and replace in the same commit — split into a delete commit and an add commit." "**Spec before code (zone rule).**" "Commit **after** a verified gate (a green `G<n>` — not 'I think it's done'), and **before** a long-running install/build/test command." **Commit first, then review** — "Review diffs `merge-base...HEAD`; an uncommitted change is invisible to it."

## The state layers + resume (Smart Entry)

"Conversation memory rots (compaction, session death, context overflow). Files do not. When state files and your recollection disagree, **the files win, every time.**" "The conversation is a control channel, the files are the store."

Two durable layers, plus the spec-zone artifacts that name the phase:

| Layer | File | Answers | Lifetime |
|-------|------|---------|----------|
| **Task** | `.skillgrid/work/<topic>/progress.md` | Which task done? What was ruled? | Gitignored; deleted when review clean |
| **Code** | git history + `[skillgrid-context]` blocks | Where in code did the last unit leave off? What task / decisions / remaining? | Always reconstructable from history |

The **phase** is not a freeform journal — it is derived from the deepest artifact present in `change-specs/<topic>/` (the canonical phase-derivation table in `_shared/rules/sdd-structure.md` § Phase derivation; `state.yaml` is a cache, not the source). `state.yaml` carries per-change `phase`/`status`/`gate`/`blocked(...)` entries that the router reads.

**The router (v2 home of resume):** `skillgrid` is the **Smart Entry** router — on "what's next" / "continue" / "resume" it detects state (the `changes.<topic>` entry + the change folder's artifacts), routes to exactly one skill, and stops. On resume it announces the recovered position in one line: "Resuming `<topic>`: phase=execution, wave 2, task 3/5 done, last commit `<sha>`. Next: task 4."

**"Persist before pressure, not after loss":** when a session is clearly long — many subagents dispatched, large tool outputs accumulating, or the harness showing a compaction indicator — STOP and: (1) append the current position to the ledger (during execution) or the spec-zone artifacts (during planning), (2) commit the work unit so the history carries the position, (3) if the remaining work is large, run the save path (persist, then continue). State writes are dual where `mnemonic.enabled: true`: the in-repo file is the source of truth; the memory mirror is an index and backup. If they disagree, the in-repo file wins.

## The 4-state gate + hard rules (test — formerly qa)

"The gate is the decision, not the test suite." "Task completion ≠ goal achievement. Passing tests ≠ verified behavior." The four areas (intent, functional, code quality, security) plus the review area run in order, then one verdict renders — **the matrix is a script**: `test/scripts/verdict-render.mjs` (JSON in, JSON out). Verdict → `report.md` (QA half).

| Verdict | Criteria |
|---------|----------|
| **PASS** | ALL of: every truth VERIFIED (intent); every scenario covered by a test that ran and passed; no CRITICAL finding from any area; all code-quality gates PASS or N/A (coverage / mutation / lint / typecheck / P0 / P1 at thresholds, jscpd duplication ≤ `duplication.fail_on` when > 0); no Trivy finding ≥ `security.trivy.fail_on` (if set); review signal `REVIEW-PASS`; every verification-scope derivation `COMPLETE`. |
| **CONCERNS** | No CRITICAL finding and all hard gates PASS, but ≥ 1 of: a `PRESENT_BEHAVIOR_UNVERIFIED` truth, a WARNING finding, a P0 without triangulation, a changed-file coverage dip, a `TRUNCATED`/`UNSCOPED` verification scope with a named re-run path, or a `STALE` verification. Non-answers the human can clear by re-running. |
| **FAIL** | ANY of: a truth UNVERIFIED; a scenario with no covering test; a test that ran and failed; a CRITICAL finding (including a CRITICAL assertion-quality gap); a CRITICAL security finding; a code-quality gate below threshold; a Trivy finding ≥ `fail_on`; a duplication % above `duplication.fail_on` (if > 0); an unresolved `high` review entry. |
| **WAIVED** | The human explicitly waived a specific criterion. Waiver + criterion + accepted risk recorded in the report. **WAIVED is never a machine decision.** |

**The floor is the weakest dimension, never the average** (`_shared/verification/floor.md`): one FAIL-valued area makes the verdict FAIL; any non-answer area caps it at CONCERNS. Never let a strong overall compensate for a weak part.

**Hard rules:** (1) A human decision always overrides the machine verdict. (2) A change that fails with no human decision is recorded as **not accepted** — never silently accepted. (3) Passing tests do not substitute for running the system — the goal-backward check is the evidence, not the suite count. (4) A gate at threshold 0 is N/A — it cannot FAIL. (5) The floor caps the verdict. (6) **Fail-closed on scope:** only a `COMPLETE`-scope verification satisfies the gate; a `TRUNCATED`/`UNSCOPED`/`UNREADABLE` scope routes to CONCERNS (named re-run) or FAIL, never a silent PASS.

**Fix-loop cap:** "3 rounds. If the gate is still FAIL after 3 rounds of fixes, stop and escalate. The finding is likely not a test gap — it's an architectural or scope problem that needs a human decision." (Review budget per tier: `_shared/planning/rigor-tiers.md` — T2: 3 rounds; T3: 3 + one fresh-model re-review.)

### Goal-backward verification (test intent area)

"Force stance: Assume the goal was NOT achieved until codebase evidence proves it. Your starting hypothesis: tasks completed, goal missed. Falsify the implementation narrative."

Four levels, one named test per level (`test/references/intent-testing.md`):

| Level | Question | Evidence that passes |
|-------|----------|----------------------|
| **Truth** | Is the falsifiable claim actually true? | A test/scenario that exercises the behavior end-to-end and passed. |
| **Artifact** | Does the named file exist, is it substantive, is it wired in? | File with real content (not a stub) + an import/registration connecting it. |
| **Key Link** | Is the wiring between components actually exercised? | A test that calls through the link and asserts the output at the far end. |
| **Data Flow** | Does real data move through the system as claimed? | A test with a concrete input producing an observable output at the boundary. |

Statuses: **VERIFIED** (a test exercised it and passed); **PRESENT_BEHAVIOR_UNVERIFIED** (code exists and is wired, but no test exercises the state transition / cleanup / ordering invariant — "Never counts as VERIFIED. Routes to human."); **UNVERIFIED** (no evidence found). "Presence is not behavior. A grep that confirms a function exists is not verification. A test that calls it and asserts the output is."

### Verification gap + traceability (test)

"Find changed behavior that could break without reliable verification catching it." One question: *"If this behavior broke where it's actually used, would verification fail?"* Gap shapes (the **Verification gaps** review lens hunts all of them): **Regression gap** (regresses where used, no covering test fails); **Missing-adoption gap** (a place that should now use the new behavior doesn't — caller not updated, flag not flipped, migration not run); **Broken-verification gap** (a test appears to cover it but is skipped/flaky/not-run/too weak — mock-only, snapshot-only, asserts the mock not the code); **Missing-oracle gap** (a happy-path scenario with no `G<n>` — "a scenario with no oracle is a traceability gap, not a completed behavior").

**Evidence rule:** "Read the test before claiming what it covers. A test name is not evidence. Search the whole repo by symbol + import references before claiming no test exists. Run the test to confirm it actually executes. Triage trusts a gap finding as filed." Each finding names a **Smallest Regression** — the specific input/state/call sequence that would break.

**Traceability:** matrix = every `acceptance.feature` scenario → covering test → did it run? → pass/fail? "A scenario with no test = traceability gap. A test that ran and failed = CRITICAL finding. A test that exists but was skipped or filtered = treat as `pending`. **Never edit the expectation to match the code.** If a test disagrees with the matrix, fix the code."

## Debug: four phases + the 3-fix rule (formerly structured-debugging)

**Iron Law:** "NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST. If you haven't completed Phase 1, you cannot propose fixes." "ALWAYS find root cause before attempting fixes. Symptom fixes are failure."

1. **Root Cause Investigation** — read errors completely, reproduce consistently, check recent changes, **Goldfish Diagnosis** (a fresh-context subagent that sees only the symptom + repro, never your hypothesis), instrument each boundary, trace the bad value backward to its origin.
2. **Pattern Analysis** — find a working example, read the reference completely, list every difference, check the broken path's assumptions.
3. **Hypothesis and Testing** — "Form a **single** hypothesis … 'I think X is the root cause because Y.' Test minimally, one variable at a time. Didn't work? Form a NEW hypothesis. DON'T add more fixes on top."
4. **Implementation** — "Create a failing test case (MUST have before fixing) … Implement a single fix. No 'while I'm here' improvements. Verify with fresh evidence."

**The 3-fix rule:** "< 3: return to Phase 1, re-analyze. **3+ failed fixes: STOP and question the architecture** — DON'T attempt Fix #4 without the architecture question with the human." "Three patches that each reveal a new problem elsewhere is a wrong architecture, not a hard bug." The trail is a state file (`debug/scripts/trail.mjs` state machine) written to `.skillgrid/work/debug/<date>-<slug>/state.md` (`hypothesis | source | evidence | verdict`), gitignored, survives compaction; `trail status` returns `gate: "architecture-question"` at 3+ attempts — read from the file, not recalled from memory.

**Verdict (three states, evidence-gated):** `RESOLVED` / `NO_ROOT_CAUSE` / `ESCALATED` (with the handoff). "If the root cause is a design decision, record an ADR; if it is a bug, the TDD evidence in the fix commit is the durable record."

This "3 rounds then escalate" mirrors the gate fix-loop cap and the review fix loop — all three cap iteration at 3 and route to a human/architecture question.

**Scope-lock during debugging:** "recurring bugs in the same files are an architectural smell" — if fixes keep landing in the same place, stop patching and question the design. The debug discipline pairs with a **freeze/scope-lock**: block edits outside the area under investigation (fail-closed — a tool call that can't be parsed is denied, not allowed) so the debugger can't accidentally "fix" unrelated code and mask the real cause.

## The Ponytail ladder (the lazy discipline, folded into execute)

"Channels a senior dev who has seen everything, and questions whether the task needs to exist at all (YAGNI)." Fully active on every coding response in `execute` (full protocol: `execute/references/ponytail.md`). The ladder (7 rungs, stop at the first that holds): (1) Does this need to exist at all? (YAGNI) → (2) Already in this codebase? → (3) Stdlib does it? → (4) Native platform feature covers it? → (5) Already-installed dependency solves it? → (6) Can it be one line? → (7) Only then: the minimum code that works. Intensities: `lite` / `full` (default) / `ultra`.

**The check rule:** "Lazy code without its check is unfinished. In the skillgrid flow the 'check' is … the `G<n>` gate the task's requirements carry in `acceptance.feature`." **Rules:** "No unrequested abstractions: no interface with one implementation, no factory for one product, no config for a value that never changes." "Deletion over addition. Boring over clever." Mark deliberate simplifications with a `ponytail:` comment naming the ceiling and upgrade path. Never simplify away input validation at trust boundaries, data-loss error handling, security, accessibility, or anything the spec/blueprint explicitly requests — an explicit requirement outranks the ladder. (The design-side twin of the one-adapter rule.)

## Skill authoring as TDD (skill-creator)

"Writing skills IS Test-Driven Development applied to process documentation. If you didn't watch an agent fail without the skill, you don't know if the skill teaches the right thing. NO SKILL WITHOUT A FAILING TEST FIRST … Write skill before testing? Delete it." The **TDD gate** (RED-GREEN-REFACTOR) applies to **new skills AND edits to existing ones**; the RED eval gives you the failure class, then you apply exactly one fix (script, gate, leading word, Red Flag row, or description change) and re-run. The description field is "When to Use, NOT What the Skill Does" — descriptions that summarize the workflow create a shortcut agents will take. Two conventions that recur in skillgrid's skill format come from here: **close every loophole explicitly** ("Violating the letter of the rules is violating the spirit of the rules") and the **Red Flags / Rationalizations table** — the "these thoughts mean STOP — you're rationalizing" lists at the bottom of load-bearing skills.

## `findings.md` — consolidated pre-design evidence

"The consolidated file is the single downstream contract that `blueprint` reads — it carries the research, prototype, and sketch evidence for the topic in one place." At `.skillgrid/change-specs/YYYY-MM-DD-<topic>/findings.md` (`# Findings — <topic>` header): research appends `## Research: <question>` (cited findings, source appendix, confidence per claim); prototype appends `## Prototype: NNN-name` (Verdict / What we learned / What's liftable / Constraints for the build); sketch appends `## Sketch: NNN-name` (Winner / Rationale / What's liftable / Constraints / Mode). "Every design decision in the blueprint that rests on a research fact, a feasibility result, or a chosen layout must cite it" (the blueprint header's **Findings** line carries the path).

## Branch finishing (ship)

When the tail passes and the branch is clean, `ship` presents the close-out menu and waits — "the integration decision is theirs": **1. Merge back to the base branch locally** (merge, then *re-run the test suite on the merged result* — the iron law), **2. Push and create a Pull Request**, **3. Keep the branch as-is** — plus a fourth v2 option, **4. Fork the worktree** (try option B while A iterates). Discarding the branch requires the user to type the word `discard` to confirm ("Only the typed word `discard` authorizes deletion"). Worktree cleanup is provenance-based — "only remove worktrees the session created … never `--force` on your own initiative." This is the tail of the phase chain: the agent never auto-merges, auto-pushes, or auto-discards. `ship` is also the v2 home of the old `reflect` skill — it completes `report.md`'s **retro half** in place in the archive (acceptance verdict, lessons, decisions; `environment-retro` as a sub-phase).

## The `SATISFIES` traceability link

The explicit wire between BDD scenarios, tasks, and the test matrix. Every blueprint task and every slice ticket carries a `**SATISFIES:** <scenario-name>` pointing at the `acceptance.feature` scenario it makes green. "The `SATISFIES` scenario names in `tasks.md` are the traceability oracle for `skillgrid:test` — every scenario must be referenceable from a test. Name them precisely."

**Acceptance-first (BDD is always on):** "every ticket carries a `SATISFIES: <scenario-name>` pointing at its acceptance scenario … The scenario's RED state is confirmed before the implementation that turns it green. Order scenario/RED tickets ahead of their implementation tickets in the graph."

The `@p0`/`@p1` priority and `@step-NN` trace markers live in the blueprint/tasks `SATISFIES` field, **not** in the Gherkin (the extracted Gherkin carries no tags). Scenario names must be unique and referenceable.

This is how a change is traced end-to-end: scenario (BDD) → task `SATISFIES` → slice ticket → test traceability matrix → the `G<n>` gate that proves it.

## The review discipline (the test review area — formerly code-review)

The review area (`test/references/review-and-fix-loop.md`) runs **after** intent + functional render PASS or CONCERNS — there is no point reviewing standards on a diff that does not do the right thing; at T0 it is skipped entirely. It answers a different question from the intent area: intent verifies the feature exists; review verifies the code is clean and on-spec.

### The topology step

Select the fan-out from diff size / risk / rigor tier, never from a choice between skills (thresholds: `_shared/planning/rigor-tiers.md` § Review escalation threshold):

- **<50 changed lines, low risk** → the 3-axis pass (Standards + Spec + Security).
- **≥50 changed lines, or high-risk** (auth, data migration, money, concurrency, public API) → the core five (Standards, Spec, Edge cases, Verification gaps, Security), plus conditional Accessibility (diff touches UI) / Performance (diff touches data access / rendering) specialists.
- **Red team always last** — reads the other specialists' merged findings.

**T3 fresh-model rule:** at T3, at least one reviewer runs on a model different from the one that implemented the change.

### The Blindness Contract + independence grades

The value of a review is its independence from the producer. Every lens is a **fresh subagent context** that sees only the work product — the diff (passed as a **path**, not inlined text) and the sources named for that lens — never the coordinator's session history, rationale, or self-assessment. **No leaks:** do not pass a reviewer your narrative, your "why I did it this way", your confidence, or a prior score/verdict on the same diff.

**Independence grade:** **A** — fresh context, no shared history with the implementer (the normal case for a dispatched subagent). **B** — fresh context but same model family / toolchain. **C** — the review happens in the implementer's own context, or the implementer's prior verdict leaked in. A **Grade C** review is **diagnostic only**: it can guide a fix but cannot stand as the "independently reviewed" evidence that `ship` or a merge requires.

### Bar guards (the diff must not weaken the bar)

An agent that writes the code and the checks takes the cheapest road to green. This pass watches the diff for the five moves that lower the quality bar instead of fixing the code. It runs on the pinned diff, before the lenses are dispatched, and its findings are **required** findings (they do not go through the triage buckets — a weakened bar is fixed in this pass or the change does not merge):

| # | Move | What to look for in `$BASE_SHA..$HEAD_SHA` |
|---|------|---------------------------------------------|
| 1 | **Threshold moved** | a quality threshold lowered, a severity dropped, a gate removed or zeroed |
| 2 | **Test got easier** | `.skip`/`.only` added, a test file deleted, assertions pulled out, a fixture changed to match the defect |
| 3 | **Checker silenced** | new suppression comments — and especially the four that switch off a check rather than a line (`istanbul ignore`, mutation-testing disables, `nosemgrep`/`gitleaks:allow`) |
| 4 | **Work unfinished** | a stub that throws `Not implemented`, an empty `catch {}`, a `TODO` standing where the implementation should be |
| 5 | **Exception appeared** | a new waiver/exception row that nobody discussed |

**Placement:** tightening the bar is silent; loosening it is a finding. **Ratchet, not aspiration:** when a threshold has no defensible target number, the bar is a ratchet — record the current value, refuse to let it fall. **The circularity check (T2+):** the bar must include at least one check the agent cannot satisfy by writing code (an external scanner, a lint config the change does not touch) — a bar made entirely of checks the change itself edits is self-referential.

### The 6-value verdict + triage

Do this yourself — the lenses file findings, you render the verdict. They lack the full context to grade severity. Deduplicate by fingerprint (`path:line:category`); two findings belong in one entry only when the same defect produced both (tagged `MULTI-REVIEWER CONFIRMED`); the entry's severity is the highest of its members. Re-grade severity yourself — a lens's self-assigned severity is dropped. Assign by consequence for the artifact's main consumer:

| Verdict | Meaning | Ledger severity |
|---|---|---|
| `high` | The bad outcome is real and intolerable (bug, data loss, security break, broken feature) | `CRITICAL` |
| `medium` | Real, tolerable — should be fixed before merge | `WARNING` |
| `low` | Cosmetic | — |
| `false` | You checked; the bad outcome does not happen here. Say what disproves it. | — (drop the entry) |
| `maybe-false` | You couldn't tell. Say what would settle it. | — (human look) |
| `could-not-verify` | Requires evidence this review cannot obtain (held-out test, live environment, long-duration run) | — (human look / held-out test) |

**Reject** (drop) any entry that is `false`; any `low` whose fix adds more complexity than the harm it removes; or any entry whose only fix is to edit the spec under review. `could-not-verify` entries are **never dropped** — they route to a human look / held-out test.

### Triage buckets + the fix loop

Route every surviving entry: **Fix now** (real, in-scope, belongs with this change), **Defer** (real but later — log it in `.skillgrid/deferred-work.md` as a `DW-<n>` entry), **Human look** (needs manual inspection before trusting), **Noise** (won't-fix or misread; say why, drop it). "Don't let the reviewer dictate scope — 'real, but later' is a valid and common call." Fix order: blocking → simple → complex. Test each fix individually. **Cap at 3 rounds** (per tier: `_shared/planning/rigor-tiers.md` § Review budget) — beyond that, surface the remaining items to the human with the rulings you made and why.

**The output:** the lens reports side by side under per-lens headings (finding count + worst issue *within* that lens; **never pick a single cross-lens winner**), then the deduplicated entries with verdicts and dispositions. The audit record goes to `change-specs/<topic>/review.md` — **committed in the spec zone** before continuing code-zone work. `tasks.md` `## Review` records exactly one verdict: **`REVIEW-PASS`** (no unresolved `high`/`medium`) or **`BACK-TO-APPLY`** (unresolved entries itemized with `file:line` + evidence), plus `Critical N · Warning N · Suggestion N` — the contract `skillgrid:ship` consumes.

### The smell baseline

A fixed set of 12 Fowler code smells (*Refactoring*, ch.3) that applies even when the repo documents nothing: Mysterious Name, Duplicated Code, Feature Envy, Data Clumps, Primitive Obsession, Repeated Switches, Shotgun Surgery, Divergent Change, Speculative Generality, Message Chains, Middle Man, Refused Bequest. Two binding rules: "**The repo overrides.** A documented standard (glossary/ADR) always wins; where it endorses something the baseline would flag, suppress the smell." "**Always a judgement call.** Each is a labelled heuristic, never a hard violation. Skip anything tooling (lint) already enforces."

### "Do Not Trust the Report" + the no-subagents contract

"Treat the implementer's report as unverified claims about the code. It may be incomplete, inaccurate, or optimistic. Verify the claims against the diff. Design rationales in the report are claims too … is the implementer grading their own work?" "Judge the code on its merits — a stated rationale never downgrades a finding's severity." **"plan-mandated":** if the plan or brief explicitly mandates something this rubric calls a defect, that IS a finding — report it as `high`, labeled plan-mandated; the plan's authorship does not grade its own work. **"⚠️ Cannot verify from diff":** requirements that live in unchanged code or span tasks — resolve each one yourself before marking the task complete; a confirmed real gap is a failed spec review and enters the fix loop.

**The no-subagents contract:** the implementer never dispatches subagents — not helpers, and never a reviewer. "A reviewer you spawn duplicates that review at full cost, and its approval counts for nothing in the process."

**YAGNI Check:** "IF reviewer suggests 'implementing properly': grep codebase for actual usage. IF unused: 'This endpoint isn't called. Remove it (YAGNI)?' IF used: then implement properly."

## The Depth Tree (Decompose & Gate, in execute)

`execute` decomposes a plan into **leaves (tasks) under branches (shared interfaces)** and gates dispatch on verified dependencies, not file order.

- **Inventory the outcomes:** "Reread the plan/spec and list every independently-omittable outcome. A plan whose tasks omit an outcome nobody owns is a half-done plan."
- **File ownership:** `Task <N>: Owns: <repo-relative globs>` and `Task <N>: Needs: <task ids or interfaces it builds on>`. "These are coordination metadata, not a filesystem sandbox — a task may still read outside its `Owns`, but it may not write to another in-flight task's `Owns`."
- **Dependency-gated dispatch:** dispatch a task only when every task named in its `Needs:` has a `Task <M>: complete` ledger line.
- **Branch integration gate:** if two or more tasks share an interface or file, name the task that *integrates* them and give it an extra `G<n>` gate: the child tasks' outputs compose and the shared interface holds end-to-end.
- **Parent re-verify:** "A leaf's self-pass does not count as verification. Before marking a task complete, re-run its `G<n>` oracles yourself."

**The concurrent coupled build** (the v2 home of parallel-execution; `execute/references/concurrent-leaves.md`): 2+ tasks that must run at once AND touch overlapping files/interfaces. Four conditions (all must be true): 2+ implementation tasks, touch same file(s) or shared interface, write sets can be made **disjoint**, want them in flight at the same time. "If any of these fail — especially 'write sets can be made disjoint' — do not parallelize." "Never dispatch a leaf whose `Owns:` intersects another in-flight leaf's `Owns:`. A wave is a set of leaves whose `Needs:` are all satisfied."

**The implementer report contract:** an implementer reports one of four statuses — `DONE | DONE_WITH_CONCERNS | BLOCKED | NEEDS_CONTEXT` — each with a distinct controller response (DONE → build the review package; DONE_WITH_CONCERNS → read the concerns first; NEEDS_CONTEXT → provide context and re-dispatch; BLOCKED → assess and re-dispatch with a more capable model or break the task). "Never ignore an escalation or force the same model to retry without changes. If the implementer said it's stuck, something needs to change."

**Controller rules of the loop:** **Rulings, not stalls** — "A running plan does not wait on a human … the spec is the binding authority, the plan is its argument, and your judgment settles what neither answers. Record every decision in the ledger as `Ruling: <what> — <why> — <what it costs if wrong>`." Stop conditions (only these): an irreversible/destructive operation, a security-sensitive action, a side effect outside this worktree, or a plan so broken that every path is a guess. **Continuous execution** — "Do not pause to check in between tasks." **Batch small same-shape work** — one dispatch for several small, same-kind edits.

## The ticketing rules (in slicing)

`slicing` publishes the sliced tickets to the configured tracker (`backlogmd | gh | glab | jira | hermes-kanban`) and keeps their status moving as execution waves progress. The tracker is the source of truth for status — without it, work has no recovery point and no audit trail. (Protocol: `slicing/references/slicing-ticketing.md`.)

**The status machine:** `backlog → ready → in-progress → review → done` — `backlog` (set by `slicing` at publish), `ready` ("all blockers done", set at wave start), `in-progress` (set by `execute`), `review` (implementation done), `done` ("review clean, merged", set on a clean review). Epic: `backlog → in-progress (first ticket starts) → done (all tickets done)`.

**The 5 canonical triage roles** (each maps to a label/status string in the active tracker): `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. "If the user's tracker already uses different label names, record the override per-role in `.skillgrid/config.yaml` — do not rename the roles."

**Backlog completeness gate** (when the tracker is Backlog.md, a ticket is not published until all four are present): **Type** (frontmatter `type:`), **References** (frontmatter `references:`), **Definition of Done** (body `## Definition of Done`), **Implementation Plan** (body `## Implementation Plan`) — plus Description, Acceptance Criteria, and `priority:`. "SDD tickets: seed References to `briefing.md` / `tasks.md` / `acceptance.feature`; seed Plan from the blueprint steps; seed DoD from project defaults. Thin one-line description stubs are **forbidden**."

**Pre-submission privacy review** (mandatory, every tracker): scan the body immediately before publishing and replace environment-specific data with placeholders — project names → `<project-name>`, usernames → `<user>`, hostnames → `<hostname>`, keys/tokens/passwords → `<token>`/`<password>`, internal ports/IPs → `<host>:<port>`. "Rule of thumb: if the reader can run the reproduction after the replacement, sanitization is correct."

**Per-phase lifecycle** (creation alone is not enough — every later execution phase owns a tracker update): start → `in-progress` + "apply began" comment; per commit → footer `Refs: <ID>`; review → `review` + verdict comment; done → `done`, closing commit uses `Closes <ID>`. "If no tracker ID exists … skip all tracker mutations. Do not invent an id."

**Title convention:** `[TYPE] Brief description (COMPONENT)`, where `[TYPE]` ∈ `{BUG, FEATURE, ENHANCEMENT, REFACTOR, DOCS, CHORE}`. "Title `[TYPE]` is not a substitute for frontmatter `type:` — both are required." **Priority** (4-tier): Critical (production down, data loss, security), High (blocks users, no workaround), Medium (workaround, subset of users), Low (cosmetic, internal).

## Gherkin authoring discipline (blueprint Phase 5)

The six authoring rules ("folded from the gherkin-authoring discipline"):

1. **Domain language** — "Use the project's glossary terms. Never use implementation jargon (class names, method names, DB column names) in steps."
2. **Observable outcomes** — "Then steps state what the user or system observes — a response, a state change, a message. Never assert internal state ('the cache is populated')."
3. **One behavior per scenario** — "If a scenario needs a second `And` clause that introduces a new precondition, split it."
4. **Falsifiable** — "Every scenario must be able to fail. 'The system works correctly' is not a step."
5. **Given/When/Then balance** — "A scenario with no `When` is a precondition list, not a scenario. A scenario with no `Then` proves nothing."
6. **No implementation details in scenario names** — "'Login succeeds with valid credentials' not 'POST /auth returns 200'."

**The Page Object Model:** "Step definitions must read as intent; all UI knowledge lives in page objects." Page objects live under `acceptance-tests/support/pages/`, one per screen/flow, encapsulating routes, form field names, selectors, and ids; they expose intent-level methods. "Parse responses with the stack's HTML parser, never with regexes over raw HTML." "Step definitions contain no selectors, regexes, or URLs; only page-object calls and assertions."

**The runner invariants:** (1) `acceptance-tests/` is an independent test project; its hooks boot the app before the suite and shut it down after. (2) The default run executes every `acceptance.feature` under `change-specs/`. (3) "A green suite is the gate for archive, and archive must never change suite results." (4) Every run generates an HTML report under `acceptance-tests/reports/`. (5) Verify extraction whenever the runner config, extractor, or `change-specs/` tree changes.

**Workflow cadence:** "Implement one pending step definition at a time: run the suite so the step fails for the right reason, implement until it passes, then commit. The suite's red scenarios at propose time are the change's work list." "Finish only when every scenario passes with zero pending or undefined steps and the HTML report is generated."

## Isolated workspace (execute Step 0)

"Detect existing isolation first. Then use native tools. Then fall back to git. **Never fight the harness.**" A native tool (e.g. `EnterWorktree`) owns placement, branching, and cleanup — bypassing it creates phantom state the harness can't see or manage. The tooling tiers (worktrees under the project-root `.worktree/`): harness-native tool → worktrunk (`wt`) → git. Step 0 detects before creating (`GIT_DIR != GIT_COMMON` → already in a linked worktree, skip; the submodule guard; empty branch → detached HEAD). "Honor any existing declared preference without asking. If the user declines consent, work in place." "MUST verify the directory is ignored before creating a worktree."

## Cross-cutting invariants (short reference)

- **A rule asks; a hook guarantees.** When an invariant is non-negotiable, wire a hook — git hooks at commit time (`git/hooks/`), the skillgrid-guards plugin at write/edit time, gate hooks at Stop time.
- **The smart zone (≤ ~40% context).** Keep the orchestrator under ~40% of the window; offload dumps/reads/research/implementation to subagents, files, and the projectwiki. The effort signal (`effort: low|standard|max` frontmatter, `_shared/craft/effort-budgets.md`) is how a skill declares how much context it will spend.
- **Vertical slices over horizontal layers.** Thin end-to-end paths, not "all DB then all API then all UI."
- **Spec zone XOR code zone.** Commit `.skillgrid/change-specs/` before the code that satisfies it — never both uncommitted.
- **Durable state over chat memory.** Files win over conversation memory, every time.
- **Persist before pressure, not after loss.** Write state when the session gets long, not after compaction has eaten it.
- **Never edit the expectation to match the code.** Fix the code.

## Where v1 concepts live in v2

| v1 concept/skill | v2 home |
|---|---|
| `qa` (4-state gate, goal-backward, gaps) | `test` — the four areas + verdict → `report.md` (QA half) |
| `code-review` / `parallel-code-review` / `requesting-code-review` | `test` review area — `test/references/review-and-fix-loop.md` → `review.md` |
| `receiving-code-review` (triage buckets) | the triage section of the `test` review area |
| `reflect` / `environment-retro` | `ship` — completes `report.md`'s retro half in place in the archive |
| `resume` / `using-skillgrid` | `skillgrid` — the Smart Entry router (state-aware routing) |
| `sketch` | inline capability of `blueprint` — `blueprint/references/sketch-phase.md` |
| `spike` | `prototype` — `blueprint/references/prototype-phase.md` (3-state verdict) |
| `brainstorming` | `brainstorm` |
| `interviewing` | `questioning` |
| `subagent-execution` / `simple-execution` / `parallel-execution` | `execute` modes (subagent / inline / fan-out; `concurrent-leaves.md`) |
| `structured-debugging` | `debug` |
| `ponytail` | `execute` lazy discipline — `execute/references/ponytail.md` |
| `ticketing` | `slicing` — `slicing/references/slicing-ticketing.md` |
| `work-unit-commits` | `_shared/work-unit-commits/` (commit contract) |
| `acceptance-test-authoring` / gherkin discipline | `blueprint` Phase 5 — `references/acceptance-phase.md` + `references/EXTRACTION.md` |
| `isolated-workspace` | `execute` Step 0 |
| `deep-research` | `research` fan-out mode (researcher / verifier / red-team) |
| `mnemonic` (orientation ladder, web cache) | `projectwiki` (durable knowledge vault) + the `mnemonic` memory layer (`_shared/rules/mnemonic-memory.md`) |
