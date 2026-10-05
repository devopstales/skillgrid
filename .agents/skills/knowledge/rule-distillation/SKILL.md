---
name: rule-distillation
description: Distills a large source (a book, spec, incident postmortem, or standards doc) into a traced, decision-equivalent mini rule set a coding agent can apply under context pressure. Use when compressing source material into agent rules with an audit trail of what was kept, merged, or dropped.
license: MIT
effort: standard
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  source: derived from ciembor/agent-rules-books _rule-workbench/PROCESS.md
---

# Rule Distillation

**Announce at start:** "I'm using the skillgrid:rule-distillation skill to distill this source into a traced rule set."

## Overview

Turns a bulky source into a **decision-equivalent** rule set: same decisions the
source drives, in ~15–20% of the text. **Decision-equivalent, not
sentence-equivalent** — the test is *does an agent make the same call under
context pressure?*, never *did the wording survive?* The output is three tiers
(`full` / `mini` / `nano`) plus a `traceability.md` that maps every retained
rule to a source section + line range and dispositions every omission.

**Iron Law:** no rule may be dropped as "the agent already knows it" without
evidence, and no rule that blocks a known bad habit may be dropped by the
**removal test** below. Compression must not turn the source into a generic
style guide — the source's distinctive bias must stay recognizable.

## When to Use

- Compressing a book, spec, standards doc, or incident postmortem into agent
  rules that fit a context budget.
- Building an on-demand "load this when doing X" rule set with an audit trail.
- Re-deriving a rule set after the source changed (re-run, don't hand-edit).

**When NOT to use:**
- The source is already short — distill a 50-line doc and you've just rewritten it.
- You want to *write new rules* rather than *faithfully compress an existing
  source* (that's authoring; the source is the constraint here).
- You need the full text for reference/audit — that's the `full` tier, not a reason to distill.

## The Workflow

```
load source → classify rules → build mini → build nano → trace → coverage review → validate → metrics
```

1. **Load the canonical source.** Identify the authoritative text (the
   "full"). If the source is a repo of tiers, the canonical file is the
   reference — never edit it; distill *from* it.
2. **Classify every rule.** For each rule in the source, assign exactly one of
   the eight types in [references/compression-rules.md](references/compression-rules.md)
   §Classification. This is the load-bearing step — the mini/nano keep/drop
   rules are keyed off these labels. Do this *before* writing any output.
3. **Build `mini`.** Apply the mini keep/drop rules (compression-rules.md §Mini).
   Keep every `book-thesis`, `decision-changing`, and `conflict-resolver`; keep
   `micro-decision`/`trigger` the target agent commonly misses or that block a
   known shortcut; drop `default` **only with evidence**; merge same-consequence
   duplicates. No target line count.
4. **Build `nano`.** Apply the nano rules (§Nano): only rules safe for always-on
   fallback that block known model biases. `nano` ⊂ `mini` conceptually.
5. **Write `traceability.md`.** Copy [templates/traceability.md](templates/traceability.md).
   Give every retained mini rule an `M*` id and every nano rule an `N*` id,
   each citing source section name(s) + current line range(s) in `full`.
   Disposition **every** omitted or merged source rule: `covered by Mx` /
   `covered by Nx` / `intentionally lost`.
6. **Section-coverage review.** Walk *every* source section in `full` and assign
   it an outcome (kept / merged into `M*` / nano-only / intentionally lost).
   Mandatory even if no text changed.
7. **Validate** against the checklist in compression-rules.md §Validation.
8. **Measure.** Run the metrics script (step 9 is the gate):
   ```
   node .agents/skills/knowledge/rule-distillation/scripts/rule-metrics.mjs <full> <mini> <nano>
   ```
   Exit 0 = all three tiers measured; the `lines/bytes/rules` table is the
   reproducible record for the release.

## The Two Drop Tests

Run these before any rule leaves `mini`. They are the heuristics that keep a
distillation faithful instead of gutted.

- **Evidence-gated `default`.** A rule may be labeled `default` (→ droppable)
  *only* with evidence the target agent reliably follows it unprompted — an eval
  result, a review finding, or a documented known model mistake. "Human-obvious
  is not agent-default." If you can't cite evidence, the label is not allowed
  and the rule stays.
- **Removal test.** Ask: *if I delete this rule, does a known bad habit come
  back?* If yes, keep it — in `mini`, and in `nano` if it's the smallest
  always-on reminder that blocks that habit in a risky hotspot.

## Process vs Book Diagnosis

When a review (yours or a human's) flags a miss, decide **before** editing:

- **Process bug** — the missing rule belongs to a reusable class that would
  strengthen *several* sources (naming pressure, mutation visibility, boundary
  ownership, cancellation/cleanup semantics, anti-shortcut trigger design…).
  → Fix the **process** first (compression-rules.md), then re-run the source.
  Do not hand-patch the one output.
- **Book-specific miss** — the rule is grounded in *this* source's own sections
  and bias, and the current process already implies it should have survived.
  → Cite the exact source sections + the process rule that required retention,
  then patch only this source + its traceability.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "This rule is obvious, I'll drop it." | Obvious-to-human ≠ agent-default. Run the removal test and check for evidence before it leaves `mini`. |
| "I'll hand-tune the output until it reads well." | A one-off edit hides a process bug. Diagnose process-vs-book first; recurring fixes belong in the process, not this file. |
| "The mini is a little longer than the nano, that's fine." | If `mini` adds little beyond `nano`, strengthen `mini` (decision-changing rules it's missing) or deliberately collapse the distinction — don't ship a restatement. |
| "I'll cite the whole file for this rule." | Traceability is section + line range. Whole-file citations don't tell a re-runner where the rule came from. |
| "This source and that other source agree, so I'll import the other's rule." | Compression is source-faithful, not best-practices aggregation. Don't pull guidance in from a different source during verification. |
| "I don't need line ranges, the section name is enough." | Line ranges rot-proof the trace. A re-run after the source edits must be able to relocate the rule without guessing. |

## Red Flags

- A `mini` rule with no `M*` id in `traceability.md`, or an id with no source section + line range.
- An omitted source section with no disposition (`covered by` / `intentionally lost`).
- A rule labeled `default` with no cited evidence (no eval, no review finding).
- `mini` that reads like a generic style guide — the source's distinctive bias is gone.
- A recurring miss patched only in one output instead of in compression-rules.md (a process bug treated as book-specific).
- `nano` containing a rule that is not safe as an always-on fallback (a niche, conditional rule).
- The canonical `full` source was edited during distillation (it must stay untouched).

## Verification

- [ ] Every rule in the source is classified into exactly one of the eight types, before any output was written.
- [ ] `mini` retains all `book-thesis`, `decision-changing`, and `conflict-resolver` rules (spot-check against the source's thesis).
- [ ] Every `default`-labeled (dropped) rule carries cited evidence; any rule without evidence was kept.
- [ ] Every retained `mini` rule has an `M*` id and every `nano` rule an `N*` id, each with a source section name + current line range in `full`.
- [ ] Every omitted/merged source rule is dispositioned `covered by Mx` / `covered by Nx` / `intentionally lost` in `traceability.md`.
- [ ] Section-coverage review walked every `full` section to one of the four outcomes.
- [ ] The removal test was applied to every rule that left `mini`; a kept rule exists for each known-bad-habit it blocks.
- [ ] Any recurring miss was diagnosed process-vs-book and the process (compression-rules.md) updated first when it was a process bug.
- [ ] `node .agents/skills/knowledge/rule-distillation/scripts/rule-metrics.mjs <full> <mini> <nano>` exits 0 and the `lines/bytes/rules` table is recorded.
- [ ] The source's distinctive bias is still recognizable in `mini` without reading the title.

## References

- [references/compression-rules.md](references/compression-rules.md) — the eight-type classification, mini/nano keep-drop rules, interpretation heuristics, and the validation checklist. Load in Step 2.
- [templates/traceability.md](templates/traceability.md) — the `M*`/`N*` → source section + line-range template with omission dispositions. Copy in Step 5.
- `skillgrid:skill-creator` — authoring a skill *from* a distilled rule set (the rule set is the source; the skill is the delivery mechanism).
