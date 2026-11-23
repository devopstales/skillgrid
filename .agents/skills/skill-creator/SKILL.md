---
name: skill-creator
description: "Create new skills, modify and improve existing skills, and measure skill performance. Use when creating a skill from scratch, editing or optimizing an existing skill, running evals to test a skill, benchmarking skill performance, or optimizing a skill's description for better triggering accuracy. Also use when auditing skills against the 10-rule best practices, or reviewing a skill library's lifecycle — usage telemetry, provenance, and curation."
license: MIT
metadata:
  author: devopstales
  version: "2.4"
  part-of: skillgrid
  created_by: human
  related_skills: [execute]
  tags: [skills, authoring, audit, lifecycle, curation, evals, benchmarking]
---

## Contents
- §1 Four Phases / §4 Steering / §5 Pruning (inline)
- Phase 1 Trigger / Phase 2 Structure / 10-Rule Audit / §7 Creating
  / Draft Template / §9 Description / Anatomy + Checklist / Skill Lifecycle
  → `references/` (one pointer each)
- Writing Rules / EGM Gates / Mermaid → `references/writing-and-interaction.md`

# Skill Creator

Create, iterate, and audit Skillgrid skills. The process has four phases —
**Trigger, Structure, Steering, Pruning** — run in order, then a 10-rule audit
before the skill ships. The loop: draft → test → review → improve → repeat.

**When NOT to use:** bounded prose edits (typo, rewording); feature design
(`skillgrid:brainstorming`); simplest code inside a script (`skillgrid:execute`'s lazy discipline).

## Workflow

```mermaid
flowchart TD
    A[Capture intent + interview] --> B[Phase 1: Trigger<br/>invocation mode]
    B --> C[Phase 2: Structure<br/>classify steps, scripts first]
    C --> D[Phase 3: Steering<br/>leading words, legwork]
    D --> E[Phase 4: Pruning<br/>deletion test, token audit]
    E --> F[Draft SKILL.md<br/>with gates + mermaid map]
    F --> G{Readiness gate<br/>confirm draft?}
    G --|confirm| H[Write evals<br/>evals/evals.json]
    G --|clarify in chat| A
    H --> I[10-rule audit]
    I --> J{All PASS?}
    J --|FAIL: fix| F
    J --|PASS| K[Run + evaluate<br/>skill vs no-skill]
    K --> L{User satisfied?}
    L --|iterate: feedback loop| A
    L --|done| M[Skill ships]
```

---

## 1. The Four Phases

| Phase | Question | Failure mode if skipped |
|-------|----------|------------------------|
| **Trigger** | How is this skill invoked, and what does that cost? | Context load or cognitive load nobody chose deliberately |
| **Structure** | What is this skill made of, and where does each piece live? | A 1400-line SKILL.md that re-loads the same reference every invocation |
| **Steering** | How do we get the agent to actually do the thing? | "I specified it, I thought I was clear, and it didn't do the thing" |
| **Pruning** | What can be deleted without changing behavior? | Sediment, no-ops, and duplicated reference material |

Determinism runs through all four — it decides what becomes a script and what
stays as prose. A skill is two units: **steps** (the procedure the agent walks)
and **reference** (the supporting information those steps need). A skill may be
all steps, all reference, or a mix.

---

## Phase 1: Trigger

See `references/trigger-phase.md` for the full trigger phase: model-invoked vs user-invoked, the unpredictability cost, description shape rules, and the pointer-quality contract.
## Phase 2: Structure

See `references/structure-phase.md` for the full structure phase: step classification, branch mapping, script conventions, progressive disclosure, spine sections, and degrees of freedom.
## 4. Phase 3: Steering — Get the Agent to Do the Thing

Steering fixes "I specified it, I thought I was clear, and it didn't do the
thing." Two techniques:

### Leading Words

The lever, its two tests (the no-op test, pretrain-first), the
positive-over-negation rule, the collapse pattern, and the double-anchoring
(execution + invocation) rule live in `per _shared/craft/leading-words.md`.
This phase owns only the *search*: which words fit *this* skill.

**Example:** Agents code layer by layer (all DB, then all schemas, then all
API, then all frontend). Instead of a paragraph saying "don't do that", use
the leading word **"vertical slice"** — a recognized term that triggers the
agent's priors.

- **How to find them:** agents are good at suggesting candidates. English is a
  wide API of small, dense phrases. Before keeping a candidate, run both tests
  from `per _shared/craft/leading-words.md` (no-op test, pretrain-first).

### Legwork (hiding the future)

If the agent does too little work on one step, it sees the later goal and
rushes there. Classic case: plan mode — the agent sees "create a plan" is the
end goal, so it does minimal legwork on "ask clarifying questions" and eagerly
plans. The fix: split the step into its own skill so the agent sees only the
current phase. Use where you specifically need a chunk of legwork.

### CRITICAL callouts (implicit side-effects)

When the *environment* (gateway, platform, harness) does something implicitly
that the agent cannot see — a reply is auto-delivered, a command has a hidden
side-effect, an `@mention` is auto-converted — the skill must state it in a
`**CRITICAL:**` block, and the block must name the agent's *default failure
mode* with NEVER directives, because the agent's default is to hedge:

```markdown
## CRITICAL: How messaging works

Your text reply IS the message sent to the group. The gateway delivers your
response automatically — there is no "send message" tool.

**NEVER say you cannot send messages. NEVER suggest the user send it
manually. NEVER add a disclaimer about permissions. Just reply with the text
you want sent.**
```

The pattern: one sentence on the *mechanism* (what the platform does), then
NEVER lines on each *default wrong behavior* the agent would otherwise
produce. The NEVERs are justified here (unlike the §3e over-prescription
caution) because the failure mode is *verifiable and repeated* — the agent
saying "I can't send that, you'll need to do it" is observable, and a leading
word won't fix a gap in the agent's model of the platform. Use a CRITICAL
block only when (a) the side-effect is invisible in the tool contract, and
(b) you have observed the agent get it wrong without the block.

### Match the form to the failure

The TDD gate (§7.1) tells you the skill failed; this tells you *which fix* the
failure calls for. Do not reach for a new paragraph of prose by default — the
form of the fix must match the shape of the failure. First name the failure
class from the eval, then pick the matching form:

| Failure shape (what the eval shows) | Fix form | Do NOT |
|---|---|---|
| Agent skipped a deterministic step / got the order wrong | A **script** (deterministic boundary, §3a/§3c) or a numbered gate | more prose exhorting it to remember the step |
| Agent rushed a step to a later goal | **Legwork** — split it into its own skill / phase | a "be thorough" sentence |
| Agent ignored a constraint it *read* | A **leading word** it must re-emit, or a **Red Flag** row naming the exact rationalization | a longer restatement of the constraint |
| Agent hedged about an invisible platform side-effect | A **CRITICAL block** naming the mechanism + NEVER lines on the default wrong behavior (§4) | a one-line "note" the agent will skim past |
| Agent rationalized its way out of a rule | An **Anti-Patterns** row (anti-pattern / symptom / fix) or a **Red Flag** | rewording the rule more firmly |
| Agent picked the wrong skill / didn't trigger | **Description** optimization (§9) | adding body the agent never reads when not triggered |
| Agent did the step but in the wrong *shape* (output format, structure) | An **Output Contract** section pinning the shape, plus a **template** or **evals/evals.json** assertion (§3e) | a prose description of the shape |
| Agent took the wrong branch under a known condition | A **Decision Gates** row for that condition (§3e) | a new paragraph re-deriving the condition |

The discipline: the RED eval gives you the failure class; you apply exactly one
matching form; the GREEN eval confirms that form closed *that* failure. If you
find yourself adding prose to fix a skip, you are treating a deterministic
failure as a judgment failure — the form is wrong, not the words.

---

## 5. Phase 4: Pruning — Make It as Small as Possible

A final pass with a checklist of failure modes. **Deletion test:** for each
line, ask "would removing this change agent behavior?" If no, delete.

### 1. Don't Repeat Yourself

Every piece of reference material has one source of truth. If a template or
definition appears in two places, it will drift. Check across SKILL.md and
all references/.

### 2. No Sediment

Old additions accumulate — irrelevant, stale, or branch-specific. Check
structure first: move branch-specific material to its branch, delete the rest.

### 3. No No-Ops

Lines that don't change behavior. Classic: "write detailed commit messages"
when the agent does that anyway. If behavior is unchanged, the line was a
no-op. Delete it.

### 4. Token Audit

Run `node .agents/skills/skill-creator/scripts/skill-size-budget.mjs check .`
from the project root (the script ships with this skill). The skill must be
within its tier ceiling (tiers in `_shared/skill-size-budget.json`: standard
22000 / large 38000 / xl 40000 / external 90000 bytes). If over budget, push
the tail into `references/` — the newest section first (anti-patterns and
decision-gates rows are the most re-derivable from the prose). Record new
skills in `skill-size-budget.json` at authoring; update the recorded size
after any content edit, or re-tier deliberately — a silent overage is a
drift signal.

Line-by-line: confirm each line changes agent behavior; flag pure
restatements of a script's logic.

---

## Writing Rules, EGM Gates, and Mermaid

See `references/writing-and-interaction.md` for §5b writing rules, §5c EGM user-interaction format, and §5d Mermaid workflow diagram rules.
## 10-Rule Audit

See `references/ten-rule-audit.md` for the full 10-rule audit checklist and the four-phase checks (T/S/St/U/P/F).
## 7. Creating a Skill

### 0. The Scope Bar (is this skill justified at all?)

Before the TDD gate, clear the curation bar. A new skill stays only if it
passes **both**:

1. **Observed failure.** It addresses something that went wrong in a real
   session: what you ran, what the skill (or its absence) did, what you
   expected. A hypothetical improvement ("it would be better if…") does not
   clear this part.
2. **It composes, or it is new.** If the behavior can be built by *calling*
   existing skills (model-invoked primitives, a `_shared/` reference, a
   user-invoked shell over them), it does not get a new skill — express it as
   that composition instead. A new skill is warranted only when no existing
   skill or reference owns the behavior.

**Rejected concepts get a record.** When a proposed skill is rejected (clears
one part but not both, or composes from existing), record it in
`references/rejected.md` beside the skill (create it on first rejection): one
line each — the concept, the why, the date. Read it before proposing anything
similar, so decided rejections are not re-litigated. This is the `.out-of-scope/`
convention applied inside the skill: the antidote to re-proposing.

The TDD gate below is the *failing-test* proof; this bar is the *justification*
proof. A skill that fails the bar is not one the TDD gate can rescue.

### 1. The TDD Gate (no skill without a failing test first)

A skill is code that shapes agent behavior, so it is tested like code:
RED-GREEN-REFACTOR. This applies to **new skills AND edits to existing ones**.

- **RED** — before writing (or changing) the skill body, write the eval
  prompts (`evals/evals.json`) that describe the behavior you want, and run at
  least one against an agent **without** the skill (or against the prior
  version for an edit). Document what the agent does wrong — the exact
  rationalization or the missing behavior. This is the failing test.
- **GREEN** — write the minimal skill that addresses *those specific*
  failures. Run the same evals with the skill present. The agent should now
  comply.
- **REFACTOR** — if the agent found a new loophole, add an explicit counter
  and re-run. Do not add content for hypothetical failures you have not
  observed.

A skill is **not deployed** until its evals pass. Do not batch several skills
without testing each. A skill you did not watch fail without is a skill you do
not know teaches the right thing. (For pure reference skills with no
behavior-shaping, the eval is a retrieval test: can the agent find and apply
the right section? The RED step is still "try to answer the question without
the file.")

**Scenario evals for decision logic.** When the skill has a state machine or named negative invariants (a banned transition, a round cap, a scope gate, a mode-selection branch), add `evals/scenarios.yaml` beside `evals/evals.json` — machine-checkable regex assertions over crafted replies, per `_shared/craft/scenario-evals.md`. The RED-GREEN gate applies to each scenario: it must fail without the rule in the body and pass with it. `evals.json` covers the prose/trigger cases; `scenarios.yaml` covers the decision branches.


### 1. Capture Intent and Interview

Extract from the current conversation first (tools used, steps, corrections,
input/output formats). Then ask: what should it do, when should it trigger,
what's the expected output, should we set up test cases? (Verifiable outputs
benefit; subjective outputs often don't.) Ask about edge cases and
dependencies. Check available MCPs.

### 2. Run the Four Phases

Execute Trigger → Structure → Steering → Pruning in order.

### 3. Draft SKILL.md

```markdown
---
name: skill-name
description: What it does (third person). Use when {trigger}.
license: MIT
metadata:
  author: name
  version: "1.0"
  part-of: skillgrid
---

## Draft Template

See `references/draft-template.md` for the fill-in skeleton and the §8 testing/iteration loop.

## 9. Description Optimization

The description field is the primary triggering mechanism. After creating or
improving a skill, offer to optimize it.

Generate 20 eval queries — should-trigger (8-10) and should-not-trigger
(8-10). Negative cases should be **near-misses**, not obvious irrelevances.
Realistic, specific, with file paths, company names, typos.

Run the optimization loop: 60/40 train/test split, 3 runs per query, iterate
up to 5 times. Select by test score (not train) to avoid overfitting.

---

## Anatomy and Checklist

See `references/anatomy-and-checklist.md` for the §10 anatomy reference (directory shape, frontmatter, router/bucket index) and the §11 verification checklist.

## Output Contract

The deliverable is the skill directory itself, verified:

- `SKILL.md` at the anatomy shape (per `_shared/craft/skill-anatomy.md`) —
  frontmatter field set and order, body under the line ceiling, fixed section
  order.
- `references/` / `scripts/` split — one concern per reference, one level
  deep; every deterministic step has its tested script with an exit-code
  contract.
- The TDD evidence — the failing test run (RED) captured before the edit, and
  the passing run (GREEN) after.
- `skill-size-budget.mjs` exit 0 (the size check).
- `check-frontmatter.mjs check .` exit 0 (the F1 frontmatter contract check).
- The 10-Rule Audit findings table (Rule / Status / Finding / Fix) with no
  unresolved FAIL.

## References

- `references/interaction-gates.md` — EGM user-interaction format: invariants,
  open-ended vs single-select, gate turn order, canonical payloads, Mermaid
  diagram rules
- `references/rationalizations.md` — the authoring rationalizations and the
  reality for each (script extraction, size, spine sections)
- `skillgrid:execute`'s lazy discipline — for choosing the simplest
  implementation *inside* the scripts this skill produces
- `_shared/craft/deterministic-boundary.md` — script vs. prose classification
- `_shared/craft/skill-anatomy.md` — format contract
- `_shared/templates/skill-template.md` — fill-in skeleton
- `.agents/agents/goldfish.md` — Goldfish persona briefs
  (synthesizer + lenses, one section each) dispatched by the
  EGM-format skills in this repo; the R36/R37/R38 protocol lives in
  `_shared/craft/goldfish-protocol.md`
- `brainstorm`, `blueprint` (this repo) — the EGM interaction format in
  practice: open-ended drilling, single-select gates, artifact-first delivery
- Anthropic Skills Guide (updated for Claude 5.5) — source of the 10 rules
- Anthropic skill-creator (github.com/anthropics/skills) — eval loop,
  description optimization, benchmarking
- Matt Pocock, "The Missing Manual: How to Write Great Skills" (UNzCG3lw6O0)
  — four-phase framework, leading words, legwork/hide-the-future, deletion
  test, context load vs cognitive load, context pointers, branches
- Matt Pocock `write-a-skill` skill (github.com/mattpocock/skills) —
  description shape, when-to-add-scripts, when-to-split-files
- Jay (100K subs), "6 New Rules for Claude Skills" (e7TY56-yIvM)
  — head-100 behavior, degrees of freedom decision test, model-test
  procedure, self-improving feedback loops, install-line-next-to-script,
  area-based reference splitting
- Hermes Agent skill library (github.com/NousResearch/hermes-agent,
  `~/.hermes/skills/`) — source of §12 Skill Lifecycle: usage-telemetry
  sidecar, provenance-scoped curation, never-delete archive transitions,
  pin exemption; and the CRITICAL-callout pattern for implicit platform
  side-effects (§4)
- GSD-core skill library (`gsd-core/skills/`, `gsd-core/workflows/`,
  `gsd-core/agents/`) — source of §3g thin-router/thick-workflow split,
  §8 verification-as-a-loop (verify → fix-plan → execute --gaps-only),
  §10a smart-entry state-aware routing (`gsd-next`), and §12b surface
  profiles (`gsd-surface`)
