---
name: style-review
description: >
  Review Markdown prose against agent-style's 21 rules (RULE-01..12 canonical +
  RULE-A..I field-observed). `/style-review FILE` audits the file and (on user
  confirm) writes a polished copy at `FILE.reviewed.md`. `/style-review A.md B.md`
  A/B-compares two drafts and emits a per-rule delta table. Complements the
  generation-time rules installed via `agent-style enable <tool>`.
argument-hint: "[FILE] [B.md]"
license: CC-BY-4.0
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: agent-style v0.9.0 skill `.claude/skills/style-review/SKILL.md` (CC-BY-4.0, Yue Zhao) — vendored into the skillgrid code-style group; rule bodies and fixtures kept under this skill's references/
---

# Style Review

**Announce at start:** "I'm using the skillgrid:style-review skill to audit this prose against the agent-style rules."

## Overview

Post-hoc review for prose an AI agent has already written. The generation-time
rules shipped via `CLAUDE.md` / `AGENTS.md` tilt the model; this skill is the
**opt-in second pass** that catches whatever the first pass missed. It scores a
file against all 21 agent-style rules (deterministic detectors where they exist,
host-model judgment where they do not), reports a per-rule scorecard, and — on
user confirm — writes a polished draft beside the source. It never mutates the
source file.

## When to Use

- After drafting any long-form Markdown — a spec, blueprint, PRD, ADR, report,
  changelog, or doc — to strip AI tells and rule violations before it ships.
- Before committing prose an agent wrote in a hurry, to catch fabricated
  citations (RULE-H) and banned tell-words (RULE-06) that a first pass missed.
- Benchmarking two drafts of the same document with the A/B compare shape.
- **When NOT to use:** code review (use `skillgrid:requesting-code-review`),
  structure/naming conventions in source files (those live in
  `_shared/rules/code-standards.md`), or a file under 10 sentences where the
  audit overhead is not worth it.

## Invocation

| Shape | Behavior |
| --- | --- |
| `/style-review FILE` | Audit FILE and show a 21-row per-rule scorecard. 19 rules are actually checked; RULE-09 and RULE-10 have no detector yet and report `skipped`. Ask the user: "produce a polished draft at FILE.reviewed.md?" On yes, write revised file and print diff. |
| `/style-review A.md B.md` | A/B compare. Emit per-rule delta. No polish, no ask. Used for benchmarking. |

Mode is inferred from argument count. No flags.

## Workflow (single file)

1. **Deterministic audit.** Shell out to `agent-style review --audit-only FILE`. Capture the canonical JSON. Mechanical rules (RULE-B, D, G, I, 12, 05, 06) and structural rules (RULE-A, C, E) return concrete violations with line numbers and excerpts. **Binary missing or failing:** fall back to the semantic path for every rule and say so in the report header (audit mode: semantic-only). The skill still completes; it never fails the audit on tooling.
2. **Semantic audit.** For each rule marked `status: "skipped"` with detector `semantic` (RULE-01, 02, 03, 04, 05, 06, 07, 08, 11, F, H), run a per-rule judgment using the rule's directive plus its BAD/GOOD examples from [`references/rules.md`](references/rules.md) as the judge prompt. Parse the host model's response into the same violation schema.
3. **Merge.** Combine deterministic + semantic rule results into the full scorecard.
4. **Report.** Print the scorecard: per rule, count + first 3 violations. If total is zero, report "no violations" and exit.
5. **Ask.** "Produce a polished draft at FILE.reviewed.md?" Wait for the user's yes/no.
6. **Polish (on yes).** Compose a revision prompt using [`references/revision-prompt.md`](references/revision-prompt.md) + the findings list + [`references/rules.md`](references/rules.md). The prompt treats each finding as a departure to consider: the reviser keeps one that reads well and judges the draft's overall pattern. A banned AI-tell word is always replaced, and a RULE-H unsupported claim stays flagged. Invoke the host model. Write the output to `FILE.reviewed.md`. Never touch `FILE`.
7. **Verify.** Shell out to `agent-style review --audit-only FILE.reviewed.md` (semantic-only re-check if the binary is unavailable). Show before → after scorecard. A remaining finding is not necessarily a failure, since a departure that reads well may stay; the diff in step 8 lets the user judge it.
8. **Diff.** Print `diff FILE FILE.reviewed.md` so the user can accept or discard hunks.

## Workflow (A/B compare)

1. Shell out to `agent-style review --mechanical-only --compare A.md B.md` (semantic-only per-rule comparison if the binary is unavailable).
2. Print the per-rule delta table from the JSON output.
3. Exit.

No ask, no polish, no file writes. This is the benchmark / regression-check shape.

## Polish invariants (hard)

The revision prompt enforces these in its template (see
[`references/revision-prompt.md`](references/revision-prompt.md)):

- **No new facts.** Never add metrics, numbers, citations, references, links, code behavior, or claims that are not already in the source. Do not add a connective (however, thus, because, then) that asserts a cause, contrast, or sequence the source does not state. If a claim is flagged as unsupported (RULE-H), leave it flagged. Do not invent support.
- **Preserve Markdown structure.** Code fences, tables, frontmatter, link syntax, heading levels, list nesting, and inline code spans are preserved exactly unless the finding IS the structure.
- **Preserve meaning.** Revisions address only flagged findings, and a finding that reads well as written may stay. Do not rewrite paragraphs the audit did not flag.
- **Write-beside only.** Output at `FILE.reviewed.md`. Never in-place.

## Self-verification

When asked "is style-review active?", reply on one line using this exact format:

`style-review active: 21-row scorecard, 19 rules checked (deterministic: RULE-B, D, G, I, 12, 05, 06, A, C, E; semantic via host: RULE-01, 02, 03, 04, 05, 06, 07, 08, 11, F, H; skipped: RULE-09, 10); workflow at skills/style-review/SKILL.md.`

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The deterministic pass is enough" | Seven mechanical rules miss every judgment rule (RULE-01..08, F, H) — the rules that actually catch fabricated citations. Run the semantic pass on every skipped-semantic rule. |
| "I'll just fix the source file in place" | Write-beside is a group invariant. In-place edits hide what the audit changed and burn the user's diff. `FILE.reviewed.md` exists so every hunk is an explicit decision. |
| "The findings are nits; polish is optional" | RULE-H (unsupported claims) and RULE-06 (tell-words) are trust issues, not nits. A scorecard with zero findings is the exit, not a low count. |
| "agent-style binary is missing, so the audit is impossible" | The semantic path works without it. Report "audit mode: semantic-only" and continue; only the mechanical detectors need the binary. |
| "I'll add a citation to clear RULE-H" | RULE-H findings stay flagged. Inventing support is exactly the violation; the fix is sourcing it or hedging the claim. |

## Red Flags

- A `FILE.reviewed.md` produced when the user never confirmed the ask (step 5 skipped).
- Source file modified by the skill (mtime/`diff` shows changes to `FILE`, not just a new sibling).
- A scorecard that lists `skipped` for a semantic rule that was never judged — a skip reported, not a skip performed.
- Revised prose that adds a number, claim, or connective the source lacks (invariant 1 broken).
- A diff where an unflagged paragraph was rewritten (invariant 3 broken).
- "Audit complete" claimed without the before → after scorecard printed after polish.

## Verification

- [ ] Scorecard printed with all 21 rows: 19 rules show counts (or a `skipped` row for RULE-09/10 with no detector); zero rows carry a judgment the audit did not perform.
- [ ] If the `agent-style` binary was unavailable, the report header says "audit mode: semantic-only".
- [ ] On a yes to the ask: `FILE.reviewed.md` exists, `FILE` is byte-identical to its pre-audit state (`diff` against the original shows no change).
- [ ] Re-audit of `FILE.reviewed.md` ran (binary or semantic) and the before → after scorecard was printed.
- [ ] `diff FILE FILE.reviewed.md` was printed for the user to accept or discard.
- [ ] Every RULE-H finding is either flagged in the polished draft or hedged — none silently "fixed" by added support.

## References

- [`references/rules.md`](references/rules.md) — the 21 agent-style rule bodies (directives, BAD/GOOD examples, severity, house style). The judge prompt for the semantic pass; the subset for the revision prompt.
- [`references/rule-detectors.md`](references/rule-detectors.md) — per-rule detection approach (mechanical regex + structural heuristics + semantic judge prompts).
- [`references/revision-prompt.md`](references/revision-prompt.md) — the polish-pass template with the hard invariants above.
- [`references/fixture-prose/`](references/fixture-prose/) — regression corpus: `*.md` inputs with paired `*.expected.json` audit output. Use to check the audit itself has not drifted.
