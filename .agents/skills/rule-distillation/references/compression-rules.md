# Compression Rules

The judgment reference for `rule-distillation`. `SKILL.md` names the steps; this
file holds the classification taxonomy, the keep/drop rules, the heuristics, and
the exit checklist. Loaded in Step 2 of the workflow.

## Classification (eight types)

Assign **exactly one** label per source rule, before writing any output. The
mini/nano keep/drop rules below are keyed off these labels.

| Type | Meaning | Default keep in mini? |
|---|---|---|
| `book-thesis` | The source's central corrective bias, recurring warning, or distinctive point of view | **Always** — even when it acts through local code/API shape rather than architecture |
| `decision-changing` | Changes an architecture, modeling, persistence, error-handling, operational, or refactoring decision | **Always** |
| `conflict-resolver` | Resolves a tradeoff (simple vs rich model, local vs strategic fix, retry vs idempotency) | **Always** |
| `micro-decision` | Changes repeated local implementation choices (naming, function shape, query-vs-command, mutation visibility, parameter design, test discipline, abstraction level) | Keep if the target agent commonly misses it, it blocks a known shortcut, or it would regress if removed |
| `trigger` | Activates only when touching a risky area | Keep if strong and it prevents a shortcut in a risky hotspot |
| `checklist-only` | Useful as a final scan but too weak as a main rule | Collapse into the final checklist, not a main rule |
| `framing` | Context, not an operational rule | Drop (or keep one line of framing in `mini`'s when-to-use) |
| `default` | The target agent reliably follows it **without prompting** | Drop **only with evidence** (see below) |

### The `default` label is evidence-gated

`default` is the only label that licenses dropping a rule, and it is the one most
often misused. Rules:

- Mark a rule `default` **only** when you have evidence the target agent
  reliably follows it unprompted: an eval result, a review finding, or a
  documented known model mistake that *does not apply* to this rule.
- **"Human-obvious is not agent-default."** A rule that sounds obvious to a
  human is not, by that fact alone, a verified agent default. If you have no
  evidence, do **not** use the label — keep the rule.
- When you *do* drop a rule as `default`, record the evidence in
  `traceability.md` (which eval / finding / mistake justifies the label).

## Build `mini`

- Keep every `book-thesis`, `decision-changing`, `conflict-resolver`.
- Keep every `micro-decision` or strong `trigger` that the target agent commonly
  misses, that blocks a known shortcut, or that would plausibly regress if
  removed.
- Remove `default` rules only when verified-default (evidence on record).
- Merge duplicated rules across the source's sections **only** when they share
  the same operational consequence. A merge gets a single `M*` id.
- Prefer short operational rules over explanatory prose.
- Convert long review/testing lists into `trigger` rules + a final checklist.
- Preserve enough of the source's own bias and vocabulary that `mini` still
  *feels like that source*, not a generic style guide.
- **No target line count.** Different sources need materially different sizes.
- If `mini` adds little beyond `nano`, either strengthen `mini` (it's missing
  decision-changing rules) or deliberately collapse the distinction.
- When unsure whether a rule is a harmless omission or source-central, **keep it
  in `mini`**.

## Build `nano`

- Keep **only** rules safe for fallback always-on attachment in a constrained
  context budget.
- Keep the smallest set of `book-thesis` rules that keeps the base bias visible,
  plus the rules that correct **known model biases**, e.g.:
  - shallow wrappers mistaken for good abstraction
  - framework-first design
  - unsafe retries
  - rewrite-first legacy work
  - fake domain-modeling or fake layering
  - hidden consistency contracts
- Keep `trigger` rules that prevent shortcuts in risky areas.
- Drop generic readability / formatting / hygiene **only** when their absence
  does not repeatedly cause model failures.
- **Do not drop** from `nano` a rule that is the smallest always-on reminder of a
  known shortcut in a risky hotspot.

## The two drop tests (recap)

- **Evidence-gated `default`** — drop-as-known requires cited evidence.
- **Removal test** — if deleting a rule brings back a known bad habit, keep it.
  In `mini` always; in `nano` when it's the smallest always-on reminder that
  blocks the habit.

## Interpretation heuristics

- Human-obvious ≠ agent-default. Require evidence for the `default` label.
- Don't strengthen a source just because a rule "seems desirable or modern."
  The justification must come from the source text, the current process, or
  explicit failure evidence.
- Don't import guidance from a different source during verification —
  compression is source-faithful, not best-practices aggregation.
- Prefer rules about boundaries, invariants, data ownership, failure semantics,
  refactoring safety, and repeated local choices that compound over time.
- Prefer explicit tradeoff rules over generic quality slogans.
- Rules about naming, function shape, mutation/query separation, parameter
  design, error handling, test discipline, local reasoning, comment discipline,
  concurrency, invalid-state elimination, misuse-resistant API design,
  cancellation/cleanup, and writing/communication discipline are *often*
  violated by agents even when they sound obvious. Do not drop them
  automatically.
- If you keep rediscovering the same omission class across different sources,
  promote the lesson into **this file** instead of hand-fixing each output.
- If a rule matters only while touching a specific hotspot, move it to trigger
  form.
- Obvious-but-violated rules stay.
- If a source's distinctive point is enforced through repeated local discipline
  (not one big architectural choice), treat that local discipline as core.
- Prefer re-running a source from a better process over manually sculpting the
  output after the fact.
- When unsure, keep the rule in `mini`; make `nano` stricter.

## Process vs book diagnosis

Before editing a distilled output after review:

- **Process bug** — a recurring compression mistake or omission class that could
  affect multiple sources (naming pressure, local reasoning, mutation visibility,
  boundary ownership, validation discipline, cancellation/cleanup semantics,
  anti-shortcut trigger design). The same argument would strengthen several
  sources. The change is driven by a better compression heuristic than by a
  unique property of this text. → **Update this file first**, then re-run the
  affected source. Do not patch the output ad hoc.
- **Book-specific miss** — the rule is grounded in this source's own sections and
  bias, and the current process already implies it should have survived. Fixing
  it does not require a new cross-source heuristic. → Cite the exact source
  sections + the process rule that required retention; update only this source +
  its traceability.

## Output shape (all tiers)

Use the same section order for `mini` and `nano`:

1. `# OBEY {source name} by {author}` (uniform — no `Mini`/`Nano` suffix in the H1)
2. `## When to use`
3. `## Primary bias to correct` (one sentence: the model bias this source fixes)
4. `## Decision rules`
5. `## Trigger rules`
6. `## Final checklist`

The final checklist **restates** a subset of `M*`/`N*` ids as a final scan — it
never introduces new rules.

## Validation checklist

Before a distillation is done:

- [ ] All H1 headings follow `# OBEY {source name} by {author}` (or `# OBEY {source name}` when no author is known); no version labels in the H1.
- [ ] The canonical `full` source is untouched.
- [ ] Each retained rule changes a real agent decision, a repeated local choice, or blocks a known failure mode.
- [ ] Each omitted section or rule has a traceable disposition: `covered by Mx`, `covered by Nx`, or `intentionally lost`.
- [ ] Duplicate guidance is merged (single `M*` id per merged consequence).
- [ ] Long lists are collapsed into triggers or checklist items.
- [ ] `nano` stands alone as a compact always-on fallback.
- [ ] `mini` adds clear value beyond `nano`; it is not a slightly longer restatement.
- [ ] The source's unique point of view survives; the central thesis is recognizable without the title.
- [ ] Removed rules have explicit reasons: verified default (evidence cited), true redundancy, too situational, or preserved only in `full`.
- [ ] Any `default`-labeled rule has explicit supporting evidence, not intuition.
- [ ] A full-to-mini gap review was completed section by section.
- [ ] Post-review edits are justified by source + process, not reviewer preference.
- [ ] Any review finding that is a reusable cross-source principle was written back into **this file** before changing outputs.
- [ ] `traceability.md` explains why anything important was removed or merged.
- [ ] `scripts/rule-metrics.mjs` ran clean and its table is recorded.
