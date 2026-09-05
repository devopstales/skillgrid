# Writing-great-skills checklist

Rubric for authoring and auditing skills. Source framing: escape **skill hell** (many skills, no shared sense of quality) by checking trigger, structure, steering, then pruning.

## 1. Trigger

Every skill can be **user-invoked** (human tells the agent to load it). **Model-invoked** skills also expose a `description` in the agent's always-on context — a **context pointer** to `SKILL.md`.

| | Model-invoked | User-invoked |
|---|---|---|
| Discovery | Agent (and often other skills) can fire it | Only the human (or an explicit load) |
| Frontmatter | Write a model-facing `description` with trigger keywords | Set `disable-model-invocation: true`; description is for humans |
| Cost | **Context load** — tokens + another thing to consider every turn | **Cognitive load** — user must remember the skill exists |
| Risk | Unpredictability: perfect fit and the model still skips → people reach for evals | More pilot skill required |

**Decision rule:** default to user-invoked when you want control and a small always-on context. Choose model-invoked when autonomous discovery (or cross-skill reach) is worth the load. Neither mode is free.

**Harness notes (approximate):** Claude Code uses `disable-model-invocation: true`. Codex-style configs may use `policy.allow_implicit_invocation: false`. Always check how OTHER skills in the target harness declare this.

**Router pattern:** when user-invoked skills outgrow memory, one user-invoked router skill that names the others and when to reach for each reduces pilot load without putting every description in context.

## 2. Structure

Two units:

- **Steps** — ordered procedure
- **Reference** — definitions, templates, catalogs that support those steps

Either unit alone is valid. Thinking in these two units keeps skills decomposable.

### Minimal SKILL.md

Every skill = description (if model-invoked) + `SKILL.md` + optional external reference. Prefer a **small** `SKILL.md`: easier to audit, cheaper in tokens, fewer competing instructions.

### Branches and context pointers

Map how the skill can be used:

- **Single branch** (always the same path) — always-needed reference may live in `SKILL.md`
- **Multiple branches** (do A, or B, or neither) — branch-only reference belongs under `references/` behind a pointer, e.g. "If writing an ADR, read the ADR template reference"

Call that an **external reference**: bundled with the skill, loaded only when the branch needs it.

## 3. Steering

### Leading words

Agents often ignore long negative instructions ("don't code layer by layer…"). Prefer a **leading word** — a short phrase packing pretrained meaning that the agent repeats in thinking and output:

| Weak | Stronger leading word |
|------|------------------------|
| Don't build everything layer by layer | **vertical slice** |
| Ask enough clarifying questions | **grill** / **relentless** (domain-specific) |

Repeat the leading word consistently through the skill. Success signal: it appears in reasoning traces and shapes the plan. If behavior is wrong, make the word stronger or more consistent — English is a wide API; try candidates (agents can help invent them). Prefer words that recruit existing priors over coinages that need long definitions.

### Leg work per step

When the agent under-invests in the current step because a later goal is visible (e.g. skim questions, rush the plan), **split** so the agent only sees the current phase (skill A completes → then skill B). Hiding the future goal increases focus. Use sparingly — only when you need extra depth on one step.

## 4. Pruning

Failure modes that inflate skills:

| Failure | What it looks like | Fix |
|---------|-------------------|-----|
| **Duplication** | Same template/definition restated in multiple places | Single source of truth |
| **Sediment** | Drive-by additions nobody dared delete; stale or off-branch | Restructure; move to branch refs or delete |
| **No-ops** | Paragraphs that don't change behavior vs the agent's default | **Deletion test**: remove it; if behavior unchanged, keep it deleted |
| **Massive skill** | Usually a symptom of the above | Fix structure/prune; don't "just accept" size |

Leading words and deletion tests are how skills stay small without losing steering power.

## Quick audit

When reviewing a skill, ask:

1. Trigger — right mode? description load justified?
2. Structure — steps vs reference clear? branch material externalized? SKILL.md lean?
3. Steering — leading words present and echoed? under-worked steps need a split?
4. Pruning — DRY? sediment? no-ops deleted?
