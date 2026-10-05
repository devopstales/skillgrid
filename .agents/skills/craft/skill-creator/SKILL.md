---
name: skill-creator
description: "Create new skills, modify and improve existing skills, and measure skill performance. Use when creating a skill from scratch, editing or optimizing an existing skill, running evals to test a skill, benchmarking skill performance, or optimizing a skill's description for better triggering accuracy. Also use when auditing skills against the 10-rule best practices."
license: MIT
metadata:
  author: devopstales
  version: "2.0"
  part-of: skillgrid
---

## Contents
- §1 Four Phases / §2 Trigger / §3 Structure / §4 Steering / §5 Pruning
- §5b Writing Rules / Red Flags / Common Rationalizations
- §6 10-Rule Audit / §7 Creating / §8 Testing / §9 Description / §10 Anatomy
- §11 Verification Checklist

# Skill Creator

Create, iterate, and audit Skillgrid skills. The process has four phases —
**Trigger, Structure, Steering, Pruning** — run in order, then a 10-rule audit
before the skill ships. The loop: draft → test → review → improve → repeat.

**When NOT to use:** bounded prose edits (typo, rewording); feature design
(`skillgrid:brainstorming`); simplest code inside a script (`skillgrid:ponytail`).

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

### Description shape

The only thing the agent sees before loading:

- First sentence: what it does, third person.
- Second: `Use when {trigger}.` (repeatable)
- No workflow steps (the agent may follow the summary and never read the body)
- Max 1024 chars
- Slightly "pushy" — Claude undertriggers. "Use whenever the user mentions
  dashboards, data visualization, internal metrics, even if they don't
  explicitly ask for a 'dashboard.'"

---

## 3. Phase 2: Structure — Steps and References

### 3a. Classify the Steps (deterministic boundary)

List every step the skill must handle. For each, apply the classification test:

> "If I ran this step twice with identical input, would the correct output be
> identical both times?"

| Step | Deterministic? | Action |
|------|---------------|--------|
| Parse Trivy JSON, filter by severity | Yes | Extract to `scripts/trivy-parse.mjs` |
| Classify an "unconfirmed" OWASP pattern | No | Keep as judgment prose |
| Check if state file matches spec zone | Yes | Extract to `scripts/state-drift-check.mjs` |
| Decide which test layer fits | No | Keep as judgment prose |

**Stop point:** If a step is *mixed* (deterministic parse + judgment
classification), split it: the deterministic part goes to a script, the
judgment part stays in prose. Name both.

### 3b. Map the Branches

A skill may have **branches** — distinct ways it can be used. Branches drive
where reference material lives:

- Reference used on **every** branch → keep in `SKILL.md`.
- Reference used on **one** branch only → move it behind a **context pointer**
  (a `references/` file pointed to by a one-line "if you're doing X, load
  `references/X.md`" line).

This is how you keep `SKILL.md` small: single-branch material never pays the
token cost on branches that don't use it.

### 3c. Write the Scripts First

Before drafting SKILL.md, write each script. For each:
1. Create `scripts/<name>.mjs` (ESM, Node 18+, no deps) or bash for git
   plumbing with heavy `awk`/`jq` usage
2. `chmod +x`
3. Run against a real input, confirm the output is correct and deterministic
4. Document the exit-code contract (0 = success/no-issue, 1 = issue found,
   2 = usage/parse error)

**Script conventions:**
- ESM (`.mjs`), Node 18+, no dependencies (use `node:fs`, `node:path`,
  `node:child_process`)
- Print a one-line verdict to stdout (e.g. `DRIFT: none`, `PASS: 36/36`)
- Print details to stdout when the verdict is non-zero
- Accept the project root as the first arg (default `.`)
- Data files (thresholds, budgets) live alongside the script or in `_shared/`

**Skip** if the skill is judgment-only. Not every skill needs a script.
**Scripts don't consume context** — they're executed, not read into the model's
window. A 200-line script costs zero tokens at load time.

### 3d. Progressive Disclosure

Keep `SKILL.md` under **500 lines**. The main file is a routing table, not a
manual. Claude loads only what it needs, when it needs it.

- **One level deep only.** Every reference links directly from SKILL.md.
  skill.md → advance.md → details.md means Claude may only preview the first
  100 lines of details.md. Flatten the chain.
- **Split by area** when a skill covers several domains (finance, sales,
  marketing). A question about revenue never loads marketing.md.
- Any file over **100 lines** gets a TOC at the top so Claude can jump to the
  section it needs from a 100-line preview.

```
skill-name/
├── SKILL.md          ← routing table, steps, pointers (< 500 lines)
├── references/       ← loaded on demand, one file per concern
│   ├── pricing.md
│   └── edge-cases.md
└── scripts/          ← executed, not loaded (no token cost)
```

### 3e. Degrees of Freedom

Match the rigidity of each step to the consequence of getting it wrong.

| Level | When | Format |
|-------|------|--------|
| **High** | Brainstorming, code review, creative work | Plain instructions, no MUST/NEVER |
| **Medium** | Structured output with variation allowed | Template with optional fields |
| **Low** | Consequential, fragile steps (billing, migrations, deletes) | Exact script or checklist |

**Decision test:** "What happens if Claude does this differently?" Nothing
changes → high freedom. Something consequential → low freedom.

One skill can mix all three. An invoicing skill: the email is high freedom,
the invoice creation is low freedom. **Low freedom usually means a script, not
more prose** — scripts run the same on every model.

Claude 5.5 is more creative than older models — over-prescriptive skills
hamstring output. If you catch yourself writing ALWAYS or NEVER in all caps,
reframe: explain *why* the constraint exists.

---

## 4. Phase 3: Steering — Get the Agent to Do the Thing

Steering fixes "I specified it, I thought I was clear, and it didn't do the
thing." Two techniques:

### Leading Words

A **leading word** is a short, well-known phrase that packs the behavior you
want into a few tokens. You put it in the skill text, and the agent
re-emits it in its reasoning traces and output. That re-emphasis steers
behavior.

**Example:** Agents code layer by layer (all DB, then all schemas, then all
API, then all frontend). Instead of a paragraph saying "don't do that", use
the leading word **"vertical slice"** — a recognized term that triggers the
agent's priors.

- **How to know it worked:** watch reasoning traces — is the agent repeating
  the leading word back?
- **How to find them:** agents are good at suggesting candidates. English is a
  wide API of small, dense phrases.

### Legwork (hiding the future)

If the agent does too little work on one step, it sees the later goal and
rushes there. Classic case: plan mode — the agent sees "create a plan" is the
end goal, so it does minimal legwork on "ask clarifying questions" and eagerly
plans. The fix: split the step into its own skill so the agent sees only the
current phase. Use where you specifically need a chunk of legwork.

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

Run `node .agents/skills/verification/qa/scripts/skill-size-budget.mjs check .`
from the project root. The skill must be within its tier ceiling (standard:
22000 bytes). If over budget, push the tail into `references/`. Add the skill
to `skill-size-budget.json` with its actual byte size.

Line-by-line: confirm each line changes agent behavior; flag pure
restatements of a script's logic.

---

## 5b. Writing Rules

Rules beyond what `_shared/craft/skill-anatomy.md` covers:

1. **Script-first.** Never write a SKILL.md line that references a script you
   haven't built and tested first.
2. **One contract per script.** One job, one exit-code contract, one stdout
   format. If a script does two things, split it.
3. **The skill line is the interface.** The agent sees command + exit-code
   meanings, not the script's internals. If the line needs to explain
   internals, the script is doing too much.
4. **Judgment stays prose, and stays short.** 1-3 sentences. Longer means it's
   a deterministic procedure in disguise — re-run the classification test.
5. **No English if-statements.** "If missing, error. If stale, drift. If
   current, continue." is three branches → one script with three exit codes.

### Red Flags

- A skill line > 2 sentences describing a procedure (not a judgment)
- "If X, do Y. If Z, do W." in prose (deterministic branching → script)
- References a script path that doesn't exist yet
- A line explaining *why* the script does what it does (internals leaked)
- A judgment paragraph > 3 sentences (procedure in disguise)
- No scripts where the process is clearly algorithmic (parse, filter, count)
- Copying a procedure instead of referencing the script that already does it

### Common Rationalizations

If you (or the user) push back on extracting a script or keeping the skill
small, load `references/rationalizations.md` — six rationalizations and the
reality for each.

---

## 6. The 10-Rule Audit

Run this audit before a skill ships. It covers the four phases plus
cross-cutting concerns from Anthropic's updated Skills Guide.

```
 1. > 500 lines? → Split into references/
 2. > 100 lines, no TOC? → Add one
 3. Too many MUST/NEVER for a creative skill? → Reduce to reasoning
 4. Model-specific behavior? → Add model notes
 5. "You are X" preamble or domain explanations? → Delete
 6. Multi-step where ORDER matters, without copyable checklist? → Add with
    go-back lines. If order doesn't matter, don't use a checklist — give the
    goal instead.
 7. No verify-then-repeat? → Add feedback loop (see §7)
 8. No template/examples/conditional for defined output? → Add
 9. Assumes installed tools without instructions? → Add prerequisites
10. Critical rules in prose that should be code? → Identify hook candidates
```

Four-phase checks (T=Trigger, S=Structure, St=Steering, P=Pruning):

```
 T1. Invocation mode deliberate (model- vs user-invoked is a chosen trade-off)
 T2. Description: third-person, trigger-specific, ≤ 1024 chars, no steps
 S1. Every step classified: deterministic or judgment
 S2. Every deterministic step has a tested, executable script
 S3. Single-branch reference behind a context pointer, not inlined
 S4. Scripts have documented exit-code contracts (0/1/2)
 St1. At least one leading word, repeated consistently
 St2. Legwork split considered where a step needs more effort
 P1. No-ops, sediment, duplication pass the deletion test
 P2. skill-size-budget.mjs exits 0
```

Report findings as a table with columns: Rule, Status (PASS/WARN/FAIL),
Finding, Fix.

---

## 7. Creating a Skill

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

## Contents                    ← if > 100 lines
- §1 ...

## Steps                       ← checklist with go-back lines
- [ ] Step 1: ...
- [ ] Step 2: ...
      If X fails: go back to Step 1
- [ ] Step 3: Verify (repeat until all pass)

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
- If a line doesn't change agent behavior, delete it. A line that names a
  command AND explains what the command does in words is two lines in one —
  cut the explanation, keep the command.

### 5. Write Test Cases

2-3 realistic prompts a real user would actually say. Save to `evals/evals.json`:

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

## 9. Description Optimization

The description field is the primary triggering mechanism. After creating or
improving a skill, offer to optimize it.

Generate 20 eval queries — should-trigger (8-10) and should-not-trigger
(8-10). Negative cases should be **near-misses**, not obvious irrelevances.
Realistic, specific, with file paths, company names, typos.

Run the optimization loop: 60/40 train/test split, 3 runs per query, iterate
up to 5 times. Select by test score (not train) to avoid overfitting.

---

## 10. Anatomy Reference

```
skill-name/
├── SKILL.md              ← required: frontmatter + instructions (< 500 lines)
├── scripts/              ← optional: executable code (no token cost when loaded)
│   └── <name>.mjs
├── references/           ← optional: docs loaded on demand
│   └── <topic>.md
└── assets/               ← optional: files used in output (templates, icons)
```

Frontmatter:
```yaml
---
name: skill-name
description: What it does (third person). Use when {trigger}.
license: MIT
metadata:
  author: name
  version: "1.0"
  part-of: skillgrid
---
```

---

## 11. Verification Checklist

- [ ] **Trigger:** invocation mode is a deliberate choice (T1)
- [ ] **Trigger:** description is third-person, trigger-specific, ≤ 1024
      chars, no workflow steps (T2)
- [ ] **Structure:** every step classified as deterministic or judgment (S1)
- [ ] **Structure:** every deterministic step has a tested script (S2)
- [ ] **Structure:** single-branch reference is behind a context pointer (S3)
- [ ] **Structure:** scripts have documented exit-code contracts (S4)
- [ ] **Steering:** at least one leading word, repeated consistently (St1)
- [ ] **Steering:** legwork split considered where needed (St2)
- [ ] **Pruning:** no-ops, sediment, duplication pass the deletion test (P1)
- [ ] **Pruning:** skill-size-budget.mjs exits 0 (P2)
- [ ] **Audit:** all 10 rules pass (no FAILs)
- [ ] Files > 100 lines have a TOC
- [ ] No domain explanations, no "you are X" preamble
- [ ] Multi-step skills have a copyable checklist with go-back lines
- [ ] Verify-then-repeat loop for consequential outputs
- [ ] Prerequisites section for external tools
- [ ] Critical "never break" rules identified as hook candidates
- [ ] No "English if-statements" in prose
- [ ] No content duplicated from another skill

---

## References

- `skillgrid:ponytail` — for choosing the simplest implementation *inside* the
  scripts this skill produces
- `_shared/craft/deterministic-boundary.md` — script vs. prose classification
- `_shared/craft/skill-anatomy.md` — format contract
- `_shared/templates/skill-template.md` — fill-in skeleton
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
