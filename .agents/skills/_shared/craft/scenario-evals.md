# Scenario Evals (machine-checkable behavioral tests)

`evals/evals.json` covers prose and trigger cases: a `prompt` plus a free-text
`expected_output` that a human or an LLM eyeballs. A **scenario eval** is the
machine-checkable sibling: a crafted prompt whose *reply* must match a
`required_pattern` (must contain) or `forbidden_pattern` (must not contain)
regex. The assertion is the regex, not a prose expectation — so it tests the
**decision rule**, not the vibe.

## When to use

A skill has a **state machine** or **named negative invariants** (a banned
state transition, a round cap, a settled-vs-fixed rule, a scope gate, a
mode-selection branch). For each such branch, write one scenario that exercises
the branch and asserts the reply's shape.

## When NOT to use

- Subjective output (a design brief, a prose summary) — the `expected_output`
  in `evals.json` is the right form; a regex over it is noise.
- Behavior already covered by a deterministic script's own tests (per
  `_shared/craft/deterministic-boundary.md`) — don't re-assert in prose.
- Trigger/description accuracy — that's `evals.json`'s job.

## File and format

Live beside `evals/evals.json` at `evals/scenarios.yaml`. One `scenarios` list;
one invariant per scenario.

```yaml
skill: "execute"
description: "Behavioral evaluation scenario suite for execute."

scenarios:
  - scenario_id: "scope_gate_large_fanout"
    description: "A 7-domain fan-out must ask for approval, not dispatch."
    eval_type: "deterministic"
    system_prompt:
      skill_file: "skills/execute/SKILL.md"
    turns:
      - user_input: "Seven independent subsystems are broken with different root causes. Run them."
        assertions:
          - type: "required_pattern"
            pattern: "(?i)(proceed|approval|approve|go ahead)"
            message: "A >5-unit fan-out must ask the user before dispatching."
          - type: "forbidden_pattern"
            pattern: "(?i)\\b(6|seven|8|eight|9|nine|10)\\b[^\\n]{0,40}\\bworktrees?\\b"
            message: "Never name >5 concurrent worktrees in a single dispatch."
```

**Assertion types:**
- `required_pattern` — the reply MUST match (case-insensitive by default via
  `(?i)`). The `message` states the invariant the regex encodes.
- `forbidden_pattern` — the reply MUST NOT match. Use the negative-lookbehind /
  alternation form when a word legitimately appears elsewhere
  (e.g. `(?<!not )(?<!never )\\bpass\\b` so "does not pass" is allowed).

**`system_prompt.skill_file`** is the path to the skill under test, relative to
the skills root. The harness loads that skill into the agent's context before
the turn; the scenario asserts the agent *followed it*.

## The RED-GREEN gate applies

Per `_shared/craft/skill-anatomy.md` and the TDD gate in `skill-creator` §8:
- **RED** — run the scenario against an agent *without* the rule in the skill
  body (or against the prior version). It must fail the assertion. That failure
  is the evidence the scenario tests the rule.
- **GREEN** — add the rule. The same scenario passes.
- **One invariant per scenario.** A scenario that asserts two rules at once
  fails to say which rule broke when it goes red.

## Where the pattern comes from

Mine the skill's own **Red Flags** and **Common Rationalizations** lists: each
row is a candidate negative invariant. A Red Flag ("fan-out dispatched with > 5
concurrent worktrees without approval") maps directly to a
`required_pattern` on the approval question and a `forbidden_pattern` on the
oversized dispatch. The `message` on the assertion is the Red Flag row,
abridged.

## Relationship to evals.json

`evals.json` and `scenarios.yaml` are complementary, not competing:
- `evals.json` — "given this task, does the agent do the right *thing*?"
  (prose expectation, human/LLM judged)
- `scenarios.yaml` — "given this *state*, does the agent make the right
  *decision*?" (regex over the reply, machine-judged)

A skill with a state machine has both. A skill without one has only
`evals.json`.
