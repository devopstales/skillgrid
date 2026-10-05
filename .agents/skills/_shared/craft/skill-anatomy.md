# Skill Anatomy (canonical SKILL.md format)

Single source of truth for what a `SKILL.md` *looks like*. Every Skillgrid skill
conforms to this file. If a skill disagrees with it, **this file wins**. This is
the contract that keeps the skill catalog from drifting — when you add or edit a skill,
you shape it here first, not ad hoc.

Companion contracts (this file assumes they exist and are honored):
- [sdd-structure.md](../rules/sdd-structure.md) — directory layout, artifact paths, phase order.
- [commits.md](../rules/commits.md) — commit message contract.
- [mnemonic-memory.md](../rules/mnemonic-memory.md) — save shape, session protocol.
- [verification-ladder.md](../rules/verification-ladder.md) — L1–L4 evidence floors.
- [verification-scope.md](../rules/verification-scope.md) — the scope of a verification (COMPLETE/TRUNCATED/UNSCOPED/UNREADABLE).
- [fast-track.md](../planning/fast-track.md) — trivial/small waiver policy.
- [cite-dont-restate.md](cite-dont-restate.md) — cite a decision/constraint by ID, don't restate it.
- [effort-budgets.md](effort-budgets.md) — the `effort:` signal + when NOT to use the pipeline.
- [../references/threat-matrix.md](../references/threat-matrix.md) — applicability-driven threats.
- [../references/strict-tdd.md](../references/strict-tdd.md) — RED → GREEN → TRIANGULATE → REFACTOR.
- [code-standards.md](../rules/code-standards.md) — global, language-agnostic code standards for any skill that writes or reviews code.

## File location

```
.agents/skills/
  <category>/           # planning, execution, verification, lifecycle, knowledge, design, craft
    skill-name/         # kebab-case, matches frontmatter `name` exactly
      SKILL.md          # required — the skill definition (entry point)
      references/       # optional — on-demand detail, loaded only when needed
      templates/        # optional — fill-in artifacts the skill writes
      scripts/          # optional — executable helpers
```

- `SKILL.md` is always uppercase, always the entry point.
- `name` (frontmatter) == directory name == kebab-case. No exceptions.
- One skill, one directory. No skill may reach into another skill's directory
  except through a named cross-reference (see *Cross-skill references*).

## Frontmatter

 ```yaml
 ---
 name: {kebab-case-name}
 description: {What the skill does, in third person. Use when {trigger conditions}.}
 license: MIT
 effort: {low | standard | max}       # optional — advisory context-spend signal (effort-budgets.md)
  disable-model-invocation: {true}     # optional — omit (false) for model-invoked
  metadata:
    author: devopstales
    version: "1.0"
    part-of: skillgrid
    based_on: {origin, if ported}      # or `source:` when a lighter derivation
  ---
  ```

### Invocation class

One axis splits skills by who can reach them:

- **model-invoked** (the default): reachable by the model or the human. Omit
  `disable-model-invocation`. Its `description` is **model-facing** and keeps the
  rich "Use when …" trigger phrasing so auto-invocation fires. The test for
  keeping a skill here: *could the model usefully reach for it autonomously?*
- **user-invoked**: reachable **only by the human typing its name**. Set
  `disable-model-invocation: true`. Its `description` is **human-facing** — a
  one-line summary a person reads in a slash-command list; strip the "Use when …"
  trigger lists, which exist to fire a model, not to inform a human.

The set of user-invoked entry points is the pipeline's human gates:
`using-skillgrid` (router), `onboarding`, `brainstorming`, `interviewing`,
`reflect`, and the meta `_shared` bucket. Adding a skill to this set is a
deliberate act — it removes the skill from the model's reach.

> Audit note: `craft/skill-write` and `craft/skill-creator` describe themselves
> as user-invoked in their own prose but carry no `disable-model-invocation`
> flag in frontmatter. Decide each explicitly: if it should auto-fire ("create a
> skill" is a model-reachable intent) keep it model-invoked and fix the prose;
> if it is a human-run authoring gate, add the flag. The prose and the flag must
> agree.

**The invariant:** a skill can **never invoke a user-invoked skill**. Only the
human can. A user-invoked skill may invoke model-invoked skills, but two
user-invoked skills can never reach each other. This is the rule that catches
the most common wiring bug: a step that says "run the `skillgrid:onboarding`
skill" from inside an autonomous flow will silently fail, because no skill may
call a user-invoked skill. When a step's precondition is user-invoked, phrase it
as an instruction for the human — "tell the user to run `skillgrid:onboarding`"
— never as a Skill-tool call (see *Cross-skill references*).

Rules:
 - **`description` is a routing contract.** It is the *only* thing an agent sees
   before deciding to load the skill. For **model-invoked** skills it carries both
   **what** and **when** — shape `Guides agents through {task}. Use when {trigger}.`
   — because that "when" is what fires auto-invocation. For **user-invoked**
   skills it is a one-line human summary (see *Invocation class*); the "when"
   trigger lists are stripped because the human, not the model, chooses it.
- **No process in the description.** If the description contains workflow steps,
  the agent may follow the summary and never read the body. Describe the outcome
  and the trigger, not the procedure.
- **`based_on` / `source` lives in `metadata`, never as a `# comment`.**
  `based_on` = ported/adapted from a named upstream skill. `source` = a lighter
  "derived from X" note. Native Skillgrid skills have neither.
- **`version` is a quoted string** (`"1.0"`), so `1.0` is not read as `1`.
- **`effort` is optional and advisory** — a context-spend signal (`low` /
  `standard` / `max`), never a gate. Absent = `standard`. Set it only when the
  signal is honest (a real `max` orchestrator, a genuinely `low` status skill);
  do not add it to every skill. See [effort-budgets.md](effort-budgets.md).
- The `_shared` directory is the one skill with **no `license`** and the
  `disable-model-invocation: true` / `user-invocable: false` flags.

## Canonical section order

This is the required skeleton. A skill may add skill-specific sections in the
`[Core]` slot, but the spine below is fixed — same order, same names — so an
agent (or a reviewer) knows exactly where each thing lives in every skill.

```markdown
# {Skill Title}                       # == `name`, title-cased

**Announce at start:** "I'm using the skillgrid:{name} skill to {purpose}."

## Overview                            # what it does + why it matters (2-4 sentences)
## When to Use                         # positive triggers AND "When NOT to use"
## [Core Process / The Workflow]       # the spine — numbered steps or phases
## [skill-specific sections]           # rules, patterns, techniques, examples
## Common Rationalizations             # REQUIRED (table) — excuse → why it's wrong
## Red Flags                           # REQUIRED — observable signs of violation
## Verification                        # REQUIRED — evidence-gated exit checklist
## References                          # optional — links to this skill's references/ + templates/
```

### Title

`# {Name}` in title case, matching `name`. Never a subtitle, never `(TDD)`-style
suffixes, never the `based on` line.

### Announce

One line, right after the title, before `## Overview`. The verbatim phrase the
agent says at start: `I'm using the skillgrid:{name} skill to {purpose}.`

**Skip it for meta/always-on skills** where announcing is noise:
- `using-skillgrid` (the router — announcing the router is circular)
- `test-driven-development` (an always-on habit, not an invoked phase)
- `test-driven-verification` (a claim-check, fires at "done", not at start)

### Overview

The elevator pitch: **what** the skill does and **why** it matters, in 2–4
sentences. This is where an *Iron Law* or core principle goes, if the skill has
one (e.g. "ALWAYS find root cause before attempting fixes"). Keep it to the
principle — the procedure belongs in the Core section.

### When to Use

Both directions, always:
- **Positive** — bullet list of trigger conditions (task types, symptoms, moments).
- **Negative** — a `**When NOT to use:**` line. The exclusion is what stops an
  agent from over-applying the skill. No skill is only-positives.

### Core Process

The heart. Numbered steps or phases, **specific and actionable**:
- **Good:** "Run `npm test` and confirm exit 0."
- **Bad:** "Make sure the tests work."
- Use an ASCII flowchart (in a ` ``` ` fence) when there's a decision point or a
  loop. ASCII over Mermaid: it's cheaper, diffs cleanly, and reads in any terminal.
  (Mermaid is acceptable only when the diagram is large *and* rendered somewhere.)
- Read `.skillgrid/config.yaml` here if the skill's behavior is config-driven,
  and name the exact keys used.

### Common Rationalizations — REQUIRED

The highest-leverage anti-rationalization device. A 2-column table pairing the
excuse an agent uses to skip a step with the factual rebuttal:

```markdown
| Rationalization | Reality |
|---|---|
| "I'll test it all at the end" | Bugs compound. A bug in slice 1 makes slices 2–5 wrong. Test each slice. |
| "This is too small for a skill" | Small changes break builds and skip the guard rails that catch that. |
```

Every skip-worthy step in the Core section needs at least one row here. If you
can't think of an excuse, the step probably isn't worth a rule.

### Red Flags — REQUIRED

A bullet list of **observable** signs the skill is being violated — the things
you'd spot during review or self-monitoring. These are *symptoms*, not rules:
"more than 100 lines written without running tests", "a `console.log` left in
production code", "PR merged without review".

### Verification — REQUIRED

The exit criteria. A `- [ ]` checklist where **every box is evidence-gated** —
it must be provable with test output, a build result, an exit code, a diff, or a
screenshot. "Seems right" is not a checkbox. A skill with no evidence-gated
checklist is a suggestion, not a process.

> Relationship to [verification-ladder.md](../rules/verification-ladder.md): this section
> is the *skill's own* exit checklist. The ladder defines the *level* (L1–L4) the
> `qa` skill enforces by change class. They compose: the skill checks itself,
> `qa` checks the change.

### References — optional

The last section, only if the skill has `references/` or `templates/` files.
List them with one-line descriptions so the agent knows what to load and when.

## Writing principles

The six rules every section must obey. When in doubt, these outrank taste:

1. **Process over knowledge.** A skill is a workflow, not a reference doc.
   Steps, not facts. (If it's facts, it belongs in `_shared/references/`.)
2. **Specific over general.** "Run `npm test` and confirm exit 0" beats
   "verify the tests". Name the command, the file, the expected value.
3. **Evidence over assumption.** Every verification box requires proof.
   An assertion without a captured result is a guess.
4. **Anti-rationalization.** Every skip-worthy step gets a row in the
   Common Rationalizations table. Pre-empt the excuse before the agent makes it.
5. **Progressive disclosure.** `SKILL.md` is the entry point; detail lives in
   `references/` and loads only when needed. Keep what's under ~50 lines inline.
6. **Token-conscious.** Every section must justify its existence. **If removing
   a section wouldn't change agent behavior, remove it.** This is the rule the
   big skills break — when a `SKILL.md` grows past ~400 lines, the first move is
   to push the tail into `references/`, not to add more inline.

## Cross-skill references

Two kinds of reference, and the distinction matters because only one of them
actually fires a skill:

**Operative** — a step tells the agent to *run another skill now*. Name the tool
explicitly. Harnesses expose skill invocation as a tool the model calls; spelling
that out is what raises the hit rate over a bare name left in prose.

```markdown
Invoke the Skill tool with `skillgrid:test-driven-development`.
Invoke the Skill tool with `skillgrid:structured-debugging`.
```

- **One skill per call.** A step that needs two skills is two calls, not one call
  with two names: "Invoke the Skill tool twice, for `skillgrid:grilling` and
  `skillgrid:domain-modeling`."
- **Only when the target is model-invoked.** Per the *Invocation class*
  invariant, a skill can never invoke a user-invoked skill. When a step's
  precondition is user-invoked (e.g. `skillgrid:onboarding`), phrase it for the
  human: "tell the user to run `skillgrid:onboarding`."

**Passive** — a router or prose that merely *names* skills for the reader to
pick from (`using-skillgrid`, the chain table, a "reach for …" suggestion).
These invoke nothing, so keep the backticked name as a plain label:

```markdown
If the build breaks, the `skillgrid:structured-debugging` skill applies.
```

**Never duplicate content across skills.** Reference and link instead. The
phase order and who-calls-whom lives in [sdd-structure.md](../rules/sdd-structure.md);
a skill may restate its own position in that chain (one line) but must not
re-specify the whole chain.

## Line budget

| Tier | Budget | Examples |
|---|---|---|
| Every `SKILL.md` | **≤ 400 lines** | pipeline skills and specialists alike |
| Reference-only (a `references/` file) | ≤ 250 | reviewer prompts, technique notes, procedure tails |

There is no heavy or orchestrator exception. A long control flow (setup, task loop, final review) keeps a short spine in `SKILL.md` and puts the procedure in `references/`. The spine names the decision and the handoff; the reference holds the steps.

Crossing the budget is a code smell, not a permission. Split, don't bloat. Push detail into `references/` first. A skill that cannot reach 400 lines by pushing the tail out should be split into two.

## Starting a new skill

Copy the fill-in skeleton at
[`../templates/skill-template.md`](../templates/skill-template.md) to
`.agents/skills/<category>/{skill-name}/SKILL.md`. It is this anatomy made concrete — every
required section is present in order with `<!-- hint -->` comments and a
compliance gate at the bottom. Replace the placeholders, delete the hints, and
run the checklist below. If the template and this file ever drift, this file
wins.

## New-skill checklist

Before a new `SKILL.md` is complete, confirm:

- [ ] Directory name == `name` == kebab-case
- [ ] `description` has what + when, **no** workflow steps
- [ ] `license: MIT` + `metadata` block present (`based_on`/`source` if derived)
- [ ] Invocation class decided: user-invoked skills carry `disable-model-invocation: true` + a human-facing description; no operative cross-reference targets a user-invoked skill
- [ ] Title is `# {Name}` title-cased; Announce line present (or a documented skip)
- [ ] `## When to Use` has a `**When NOT to use:**` line
- [ ] `## Common Rationalizations` table present (≥1 row per skip-worthy step)
- [ ] `## Red Flags` list present
- [ ] `## Verification` checklist present, every box evidence-gated
- [ ] Total ≤ 400 lines (or tail moved to `references/`)
- [ ] No content duplicated from another skill — referenced by `skillgrid:{name}` instead
