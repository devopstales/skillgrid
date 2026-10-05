---
name: skill-write
description: "Guides agents through creating or refactoring a Skillgrid skill, enforcing the deterministic/agent boundary: logic that repeats identically every run lives in a script the skill calls, not in prose the agent re-interprets. Use when creating a new skill under .agents/skills/. Use when refactoring an existing skill to reduce token cost or extract scripts."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

# Skill Write

**Announce at start:** "I'm using the skillgrid:skill-write skill to author this skill."

## Overview

Writes a Skillgrid skill that respects the deterministic boundary: the agent
decides *what* to do, scripts do *it*. A skill that re-interprets deterministic
logic in prose on every invocation burns tokens and risks misreading. The fix
is to extract that logic into a script once, and reduce the skill line to a
command call.

**Core principle:** Per `_shared/craft/deterministic-boundary.md` — if a
step produces the same output with identical input, it is a script, not a skill
line. Skills are decision boundaries, not programs.

## When to Use

- Creating a new skill under `.agents/skills/`
- Refactoring an existing skill that has grown past its budget or has prose
  describing deterministic logic
- Reviewing a skill draft and finding "wall of code" patterns (English if
  statements, multi-step procedures that are really algorithms)

**When NOT to use:**
- Editing a skill's prose without changing its structure (typo fix, rewording)
  — that's a bounded edit, not a skill-authoring task
- Designing the *feature* the skill will support — that's `skillgrid:brainstorming`
- Choosing the simplest code solution inside a script — that's `skillgrid:ponytail`

## The Process

Four phases, in order: **Trigger** (how the skill is invoked), **Structure**
(what it's made of), **Steering** (how you get the agent to do the thing),
**Pruning** (make it as small as possible). Determinism runs through all four —
it decides what becomes a script.

### Phase 1: Trigger — How It's Invoked

Decide the invocation mode before writing anything. This is a trade-off, not a
preference:

- **Model-invoked** (has a `description`). The agent sees the description and
  may load the skill. Cost: **context load** — every description burns tokens on
  every request and adds a decision for the agent.
- **User-invoked** (`disable-model-invocation: true` in frontmatter). No
  description in the agent's context. Cost: **cognitive load** — the user must
  remember the skill exists and invoke it.

**Choosing rule:** Default to model-invoked. Switch to user-invoked when the
skill is *predictable* (you know exactly when you'll need it) and the marginal
context load of another always-present description is not worth it. Skillgrid
keeps most skills model-invoked; a small number of meta/always-on skills
(`using-skillgrid`, `test-driven-development`, `test-driven-verification`) opt
out.

**Description shape** (the only thing an agent sees before loading):

- First sentence: what it does, third person.
- Second sentence: `Use when {specific trigger}.` (repeatable for multiple triggers)
- No workflow steps. If the description contains a procedure, the agent may
  follow the summary and never read the body.
- Max 1024 chars.

### Phase 2: Structure — Steps and References

A skill is two units: **steps** (the step-by-step procedure the agent walks)
and **reference** (the supporting information those steps need). A skill may be
all steps, all reference, or a mix. The first job is to classify each step:
deterministic (→ script) or judgment (→ prose).

#### 2a. Classify the Steps

List every step the skill must handle. For each, apply the classification test
from `_shared/craft/deterministic-boundary.md`:

> "If I ran this step twice with identical input, would the correct output be
> identical both times?"

Produce a table:

| Step | Deterministic? | Action |
|------|---------------|--------|
| Parse Trivy JSON, filter by severity | Yes | Extract to `scripts/trivy-parse.mjs` |
| Classify an "unconfirmed" OWASP pattern | No | Keep as judgment prose |
| Check if state file matches spec zone | Yes | Extract to `scripts/state-drift-check.mjs` |
| Decide which test layer fits | No | Keep as judgment prose |

**Stop point:** If a step is *mixed* (deterministic parse + judgment
classification), split it: the deterministic part goes to a script, the
judgment part stays in prose. Name both.

#### 2b. Map the Branches

A skill may have **branches** — distinct ways it can be used (e.g. a skill that
updates a glossary *and* creates ADRs, where either or both may be skipped).
Branches drive the next decision: which reference material lives in `SKILL.md`
vs. behind a context pointer.

- Reference used on **every** branch (e.g. a template always needed) → keep in
  `SKILL.md` or `references/` loaded on demand.
- Reference used on **one** branch only → move it out of the main body behind a
  context pointer (a `references/` file or a script), pointed to by a one-line
  "if you're doing X, load `references/X.md`" line.

This is how you keep `SKILL.md` small: single-branch material never pays the
token cost on branches that don't use it.

#### 2c. Write the Scripts First

Before drafting SKILL.md, write each script. For each:

1. Create `.agents/skills/<group>/<skill>/scripts/<name>.mjs` (or bash for git plumbing
   with heavy `awk`/`jq` usage).
2. Make it executable (`chmod +x`).
3. Run it against a real input. Confirm the output is correct and deterministic.
4. Note the exit-code contract (0 = success/no-issue, 1 = issue found,
   2 = usage/parse error) — this becomes the skill line's interpretation guide.

**Script conventions:**
- ESM (`.mjs`), Node 18+, no dependencies (use `node:fs`, `node:path`, `node:child_process`)
- Print a one-line verdict to stdout (e.g. `DRIFT: none`, `PASS: 36/36 within ceiling`)
- Print details to stdout when the verdict is non-zero (tables, lists)
- Accept the project root as the first arg (default `.`)
- Data files (thresholds, budgets) live alongside the script or in `_shared/`

**Skip this step** if the skill is judgment-only (no deterministic steps).
Not every skill needs a script. A skill that is purely "decide X, then do Y"
with no fixed computation is correct as pure prose.

#### 2d. Draft SKILL.md

Copy `_shared/templates/skill-template.md` to `.agents/skills/<group>/<skill>/SKILL.md`.
Fill it in following `_shared/craft/skill-anatomy.md`. For each step from
the §2a classification table:

- **Script step** → one line: "Run `node scripts/<name>.mjs`; exit 0 = X,
  exit 1 = Y (report the table), exit 2 = Z (report the error)." Do NOT
  re-describe what the script does internally.
- **Judgment step** → 1-3 sentences of prose describing what the agent decides,
  what inputs it considers, and what the possible outputs are.
- **Mixed step** → the script line for the deterministic half + prose for the
  judgment half, in sequence.

**Token rule:** If removing a line would not change agent behavior, remove it.
A line that names a command AND explains what the command does in words is two
lines in one — cut the explanation, keep the command.

### Phase 3: Steering — Get the Agent to Do the Thing

Steering fixes "I specified it, I thought I was clear, and it didn't do the
thing." Two techniques:

**Leading words.** Pick a short, well-known phrase that packs the behavior you
want into a few tokens, and repeat it consistently throughout the skill. The
agent re-emits the phrase in its reasoning traces, and that re-emphasis steers
its behavior. "Vertical slice" beats a paragraph about "don't code layer by
layer" because it's a recognized term that triggers the agent's priors.

- **How to know it worked:** watch the reasoning traces. If the agent repeats
  the leading word back ("we'll do this as a thin vertical slice"), it landed.
- **How to find them:** agents are good at suggesting candidates. When the agent
  isn't doing what you want, the move is to make the leading word more
  consistent, more precise, and to look for a sharper one — English is a wide
  API of small, dense phrases.

**Legwork (hiding the future).** If the agent does too little work on one step,
it's often because it sees the later goal and rushes there. The classic case is
plan mode: the agent sees "create a plan" is the end goal, so it does minimal
legwork on "ask clarifying questions" and eagerly plans. The fix is to split
the step into its own skill so the agent sees only the current phase, not the
future one. More legwork on the current step, because the next goal isn't
visible yet. Not every skill needs this — use it where you specifically need a
chunk of legwork on one step.

### Phase 4: Pruning — Make It as Small as Possible

Pruning is a final pass with a checklist of failure modes. **Deletion test:**
for each line, ask "would removing this change agent behavior?" If no, delete.

1. **Don't repeat yourself.** Every piece of reference material has one source
   of truth. If a template or definition appears in two places, it will drift.
2. **No sediment.** When a skill grows over time, old additions accumulate —
   material that's irrelevant, stale, or only relevant to one branch. Check
   structure first: move branch-specific material to its branch, delete the
   rest.
3. **No no-ops.** Lines that appear to do something but don't change behavior.
   Classic: a paragraph telling the agent to "write detailed commit messages"
   when it does that anyway. Delete the line — if behavior is unchanged, the
   line was a no-op.

After the pruning pass, run the token audit.

#### 4a. Token Audit

1. Run `node .agents/skills/verification/qa/scripts/skill-size-budget.mjs check .` from the
   project root. The new skill must be within its tier ceiling (standard:
   22000 bytes).
2. If over budget: push the tail (examples, detailed procedures, edge cases)
   into `references/`. The SKILL.md spine stays; the detail loads on demand.
3. Line-by-line check: for each line in SKILL.md, confirm it changes agent
   behavior. Flag lines that are pure restatement of a script's logic.
4. Add the skill to `skill-size-budget.json` under `skills` with its actual
   byte size.

## Writing Rules (beyond anatomy)

These are the skill-write-specific rules that the anatomy file doesn't cover:

1. **Script-first.** If a step is deterministic, the script exists and runs
   before the SKILL.md line that references it is written. Never write a
   skill line that references a script you haven't built and tested.
2. **One contract per script.** Each script has one job, one exit-code
   contract, one stdout format. If a script does two things, split it.
3. **The skill line is the interface.** The agent sees: command + exit-code
   meanings. It does NOT see: the script's internal logic, its file reads,
   its parsing. If the skill line needs to explain the script's internals,
   the script is doing too much or the skill is in the wrong place.
4. **Judgment stays prose, and stays short.** A judgment line is 1-3
   sentences. If it's longer, it's probably a deterministic procedure
   disguised as judgment — run the classification test again.
5. **No English if-statements.** "If the file is missing, report an error.
   If the file is present but stale, report drift. If the file is current,
   continue." is three branches of a deterministic check → one script with
   three exit codes or a structured output.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The script is overkill for a 5-line check" | Five lines of prose × every invocation = 5 lines of tokens × every invocation. A 15-line script is written once. The crossover is at ~3 invocations. |
| "The agent can figure out what to do from the description" | "Figure out" means re-derive the logic every time, with a risk of misreading. A script is a fixed, tested derivation. The agent's job is to *call* it, not to *be* it. |
| "I'll write the skill first and extract scripts later" | "Later" is when the skill is already bloated and the extraction is a refactor instead of a design decision. Script-first makes the boundary explicit at authoring time. |
| "This step is 'almost deterministic' — just one judgment call at the end" | Split it. The deterministic 90% goes to a script (JSON output); the 10% judgment stays in prose. A mixed step is two steps. |
| "The skill is only 350 lines, it's within budget" | Budget measures size, not correctness. A 300-line skill that re-interprets deterministic logic on every run is more expensive than a 150-line skill that calls three scripts. The budget is a floor, not a goal. |
| "The agent is smart enough to parse the JSON itself" | "Smart enough" is "usually, but not always." A script that parses the JSON and prints a verdict is deterministic. The agent parsing it is a coin flip with a 95% hit rate — and that 5% is the bug you chase at 2am. |

## Red Flags

- A skill line longer than 2 sentences that describes a procedure (not a judgment)
- "If X, do Y. If Z, do W." in skill prose (deterministic branching → script)
- A skill that references a script path that doesn't exist yet
- A skill line that explains *why* the script does what it does (the script's internals leaked into the interface)
- A judgment paragraph longer than 3 sentences (probably a procedure in disguise)
- A new skill with no scripts where the process is clearly algorithmic (parse, filter, count, compare)
- Copying a procedure from an existing skill instead of referencing the script that already does it

## Verification

- [ ] **Trigger:** `description` has what + when (third person), no workflow steps, ≤ 1024 chars. Invocation mode is deliberate (model- vs. user-invoked is a chosen trade-off, not an accident)
- [ ] **Structure:** every step is classified as deterministic or judgment. Every deterministic step has a corresponding script that exists, is executable, and runs successfully against a real input
- [ ] **Structure:** every script has a documented exit-code contract (0/1/2) that the SKILL.md line references
- [ ] **Structure:** single-branch reference material is moved behind a context pointer, not inlined in the main body
- [ ] **Steering:** at least one leading word is chosen for the primary behavior you want steered, and it's repeated consistently. Legwork split is considered where a step needs more effort
- [ ] **Pruning:** no-ops, sediment, and repeated reference material all pass the deletion test
- [ ] SKILL.md has no "English if-statements" — no multi-branch prose describing deterministic logic
- [ ] `node .agents/skills/verification/qa/scripts/skill-size-budget.mjs check .` exits 0 (all skills within ceiling)
- [ ] The new skill's byte size is recorded in `skill-size-budget.json`
- [ ] SKILL.md passes the anatomy checklist (`_shared/craft/skill-anatomy.md` §New-skill checklist)
- [ ] No content is duplicated from another skill — cross-references use `skillgrid:{name}`

## References

- `_shared/craft/deterministic-boundary.md` — the rule (script vs. prose) with the classification test and examples
- `_shared/craft/skill-anatomy.md` — the format contract (section order, frontmatter, line budget)
- `_shared/templates/skill-template.md` — the fill-in skeleton to copy
- `skillgrid:ponytail` — for choosing the simplest implementation *inside* the scripts this skill produces

## Sources

- Matt Pocock, "The Missing Manual: How to Write Great Skills" (video, UNzCG3lw6O0) — the four-phase checklist (Trigger / Structure / Steering / Pruning), leading words, legwork/hide-the-future, deletion test, single source of truth, context pointers
- Matt Pocock `write-a-skill` skill (github.com/mattpocock/skills) — description shape, when-to-add-scripts, when-to-split-files
