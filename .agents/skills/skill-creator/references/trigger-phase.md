# Phase 1: Trigger — How It's Invoked

## 2. Phase 1: Trigger — How It's Invoked

Decide the invocation mode before writing anything. This is a trade-off, not a
preference.

### Model-invoked vs User-invoked

- **Model-invoked** (has a `description`). The description sits in the agent's
  context on every request as a context pointer. The agent may follow it and
  load the SKILL.md body. Cost: **context load** — every description burns
  tokens on every request and adds a decision for the agent.
- **User-invoked** (`disable-model-invocation: true`). No description in the
  agent's context. The user must remember the skill exists and invoke it.
  Cost: **cognitive load** — more skills the user must track.

### The unpredictability cost of model-invoked

A context pointer can be ignored. Even when the skill is a perfect match, the
model may choose not to follow it — leaving you to eval triggering accuracy.
You can avoid this class of problem by making the invocation deliberate.

**Choosing rule:** Default to model-invoked. Switch to user-invoked when the
skill is *predictable* (you know exactly when you'll need it) and the marginal
context load of another always-present description is not worth it. Skillgrid
keeps most skills model-invoked; a small number of meta/always-on skills
(`using-skillgrid`, `test-driven-development`, `test-driven-verification`) opt
out.

### Consequences of the choice

The invocation mode is not just a context/cognitive trade-off; it constrains
how the skill *composes* with other skills. Three consequences follow:

**1. Dependency direction.** A user-invoked skill may *call* a model-invoked
skill, but it can never reach another user-invoked skill (no other skill has a
description to fire it). This is what lets a user-invoked skill be a *thin
orchestration shell* over model-invoked primitives: e.g. a user-invoked
`grill-with-docs` is just "Call the Skill tool for `grilling` and
`domain-modeling`." Put the *reusable discipline* in a model-invoked skill;
put the *named entry point* in the user-invoked shell. When a step's
precondition is a user-invoked skill, phrase it as an instruction for the
human to run it — never as a tool call.

**2. Shared-reference placement.** Two *user-invoked* skills both needing the
same reference can't both reach it by description (neither has one). So shared
reference that more than one user-invoked skill needs goes to a **plain file
any skill can point at** — our `_shared/`. A model-invoked skill *can* host
shared reference (other skills can invoke it), but only when it also needs to
fire on its own.

**3. Passive vs active (when NOT to make a thing a skill).** Merely *reading*
a file for vocabulary or a lookup is a one-line context pointer, **not** a
skill. Only the *active* discipline — challenging terms, writing ADRs,
updating the glossary — is a skill. The test: does it do *work* (produce an
artifact, change state, enforce a gate) or just *read*? If it only reads, it
is a pointer into the file that owns the material, not a new skill. This is
the curation rule that stops reference-reading from inflating into skills.

### Description shape

The description is the skill's **top-level context pointer**: the only thing
the agent sees before loading, and it is forced to stay loaded at all times
(permanent context load in exchange for discoverability). It is where the
triggering work happens, so its *wording* — not just its shape — decides how
reliably the agent reaches the body.

Shape:

- First sentence: what it does, third person.
- Second: `Use when {trigger}.` (repeatable)
- No workflow steps (the agent may follow the summary and never read the body)
- Max 1024 chars
- Slightly "pushy" — Claude undertriggers. "Use whenever the user mentions
  dashboards, data visualization, internal metrics, even if they don't
  explicitly ask for a 'dashboard.'"

Pointer quality (the wording rules, in priority order):

1. **Front-load the leading word.** The trigger branch's leading word (per
   `_shared/craft/leading-words.md`) goes first — the pointer is where it does
   its triggering work.
2. **One trigger per branch.** Synonyms that rename a single branch are one
   branch written twice; collapse them and keep only genuinely distinct
   branches.
3. **Cut identity the body already carries.** The description should not
   restate what the body is — that is restating, not routing (per
   `_shared/craft/cite-dont-restate.md`).
4. **Sharpen before inlining.** A must-have target behind a weakly-worded
   description is a *variance bug*: sharpen the wording first; inline the
   material only if sharpening fails.

---

