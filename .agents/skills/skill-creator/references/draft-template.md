# Draft Template and Testing Loop

## Contents                    ← if > 100 lines
- §1 ...

## Workflow                    ← mermaid map of the steps + gates (§5d),
flowchart TD                   ←   diamond per interactive gate
    A[Step 1] --> B{Gate?}
    B --|confirm| C[Step 2]
    B --|loop back| A

## Input Contract             ← operational skills only (§3e):
| Input | Source | Type | Required? |
       ←   one line per input; skip if already a named section (e.g. ## Config)

## Decision Gates              ← operational skills only (§3e):
| Condition | Action |     ←   condition→action table, rows sourced from existing rules

## Steps                       ← checklist with go-back lines
- [ ] Step 1: ...
- [ ] Step 2: ...
      If X fails: go back to Step 1
- [ ] Step 3: Verify (repeat until all pass)
- [ ] Step 4: Gate — chat summary first, then the single-select question
      (EGM format, §5c; payload in references/interaction-gates.md)

## Output Contract             ← exact final output shape (fields, paths,
                               ←   verdict values); skip if already a named
                               ←   section — cite it

## Anti-Patterns               ← tail: | Anti-pattern | Symptom | Fix |

## References                  ← context pointers, not inlines
- For pricing details: `references/pricing.md`
- For edge cases: `references/edge-cases.md`

## Prerequisites               ← install line next to every script call
Requires: ...

## Model Notes                 ← if model-specific behavior matters
- Haiku: ...
- Sonnet: ...
- Opus/Fable: ...
```

Writing style:
- Imperative form.
- Explain *why* in lieu of MUST/NEVER.
- One sentence per step for script calls: "Run `node scripts/x.mjs`; exit 0 =
  pass, exit 1 = drift (report table)."
- No English if-statements. Multi-branch deterministic logic → script.
- No "you are an elite X" preamble. No explaining what a domain concept is.
- Interactive steps use the EGM gate format (§5c): one question per gate,
  open-ended for exploration, single-select for decisions, chat summary
  before the gate.
- Multi-step workflows include a Mermaid diagram (§5d) mapped to the steps.
- If a line doesn't change agent behavior, delete it. A line that names a
  command AND explains what the command does in words is two lines in one —
  cut the explanation, keep the command.

### 5. Write Test Cases (the RED step — write these BEFORE the body if new)

2-3 realistic prompts a real user would actually say. Save to `evals/evals.json`.
For a new skill, author these *before* the SKILL.md body so you have the failing
test to watch fail (TDD Gate, §7.1); for an edit, the prompt that reproduces
the old behavior is the failing test.

```json
{
  "skill_name": "my-skill",
  "evals": [
    {
      "id": 1,
      "prompt": "User's realistic task prompt",
      "expected_output": "Description of expected result",
      "files": []
    }
  ]
}
```

### 6. Run the 10-Rule Audit

Before showing the user, run the full audit from §6. Fix FAILs, note WARNs.

---

## 8. Testing and Iterating

### Run and Evaluate

Spawn subagents with and without the skill, same prompt, in the same turn.
While runs are in progress, draft assertions (objectively verifiable
statements). For subjective skills, evaluate qualitatively.

When runs complete: grade against assertions → aggregate (pass rate, time,
tokens) → analyst pass (non-discriminating assertions, high-variance evals) →
show the user, collect feedback.

### Iterate

Based on feedback:
1. **Generalize.** Don't overfit to the test examples. If a stubborn issue
   persists, try different metaphors or patterns.
2. **Keep it lean.** Remove things not pulling their weight. Read
   transcripts, not just outputs.
3. **Explain the why.** Reframe MUST/NEVER as reasoning. Claude 5.5 responds
   to understanding, not shouting.
4. **Look for repeated work.** If all test runs independently wrote similar
   helper scripts, that's a signal to bundle a script.
5. **Check the leading words.** Watch reasoning traces — is the agent
   re-emitting the leading word? If not, make it sharper or more consistent.

Rerun tests → new feedback → repeat until the user is satisfied.

### Feedback Loops

The skill checks its own work: run a check → fix whatever fails → repeat until
passes. The check doesn't have to be code — a style-guide review or
"are all required sections present" pass works.

**Self-improving:** when a check fails for a reason not in the guide, have the
model suggest a new rule. Approve it, add it to the reference doc. Next run
checks against the new rule. The model is good at updating skills from what it
learns on the task.

**Verification as a loop, not a gate.** When a skill's *verification step*
(e.g. `test`, `verify-work`) finds failures, those findings should feed back
into the *execution step* (`execute`) as targeted fix tasks, not just produce
a report. The pattern:

1. **Verification produces fix plans, not just verdicts.** The test/verify
   skill's output includes a list of diagnosed gaps, each with a fix plan
   (what to change, why, how to verify the fix). The verdict (`PASS`/`FAIL`)
   is the summary; the fix plans are the deliverable.
2. **Execution accepts gap-closure mode.** The execute skill has a
   `--gaps-only` (or equivalent) mode that reads the verification report and
   executes *only* the fix plans, not the full task list. This keeps the
   re-execution scoped to what failed.
3. **The loop is explicit.** The pipeline is
   `execute → verify → (if gaps) execute --gaps-only → verify → …`
   until verification passes or a human stops it. The skill that runs the
   loop (typically the orchestrator, e.g. `skillgrid`) owns the iteration
   count; the individual skills are single-pass.

**When to apply:** the skill has a verification step whose failures are
*fixable by re-executing a subset of prior work* (bug fixes, test failures,
integration gaps). It does NOT apply when verification failures require
*re-planning* (the plan itself was wrong) — those route back to the
planning skill, not the execution skill.

**Anti-pattern:** a verification skill that writes a report and stops,
leaving the user to manually translate "test X failed because of bug Y"
into a new task for the execution skill. The translation is deterministic —
it belongs in the verification skill's output, not in the user's head.

**Source:** GSD-core's `gsd-verify-work` (conversational UAT that produces
diagnosed gaps + fix plans) feeds directly into
`gsd-execute-phase --gaps-only` (executes only gap-closure plans). The
loop closes without manual intervention.

### Test on the Models You Use

Run your most-used skill on the same task with Haiku, Sonnet, and Opus. Then:

- **Haiku misses a step** → make it clearer or turn it into a script (scripts
  run the same on every model)
- **Opus does worse with the skill than without** → over-explaining; cut
  instructions until Opus performs better
- **Sonnet is in between** → the skill sits in the right place

Skills written for older models are often too prescriptive for 5.5 — remove
older instructions if the model does better without them. Declare intended
models in frontmatter (`metadata.models: [haiku, sonnet, opus]`).

---

