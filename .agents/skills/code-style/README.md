# Code Style Group

Skills in this group govern **how skillgrid output is written** — the prose,
naming, and formatting conventions applied when an agent writes or reviews
Markdown, specs, reports, or code identifiers.

This is the home for style rules and style reviews. It is separate from
[../craft](../craft/), which is about *how skills are built* (anatomy, effort
budgets, anti-rationalization), and from [../verification](../verification/),
which gates correctness. A skill lives here when its verdict is about **surface
quality of writing**, not about whether the code or process is right.

The group has two phases, both from the same agent-style v0.9.0 rule set
(ADR-0039):

- **Generation-time** — the compact 21-rule directives live at
  [`../_shared/rules/prose-style.md`](../_shared/rules/prose-style.md) (shared
  convention, referenced from `AGENTS.md` and `skill-anatomy.md`). Agents apply
  the rules *while writing*; it is not a skill and does not live in this group's
  directory.
- **Review-time** — the `style-review` skill below is the opt-in post-hoc audit
  against the same rules (scorecard, write-beside polish, A/B compare).

## Members

| Skill | What it does |
|---|---|
| [`style-review`](style-review/) | Post-hoc audit of Markdown prose against agent-style's 21 rules (RULE-01..12 canonical + RULE-A..I field-observed): deterministic + semantic scorecard, on-confirm polish to `FILE.reviewed.md`, and A/B delta between two drafts. |

## Group invariants

- **Write-beside, never in-place.** Style skills report against a file and
  write revised output to a sibling (`FILE.reviewed.md`); the source is never
  mutated by the skill.
- **Rules are defaults, not gates.** Most style rules are judgment rules
  (agent-style's escape hatch: break a rule sooner than say something
  barbarous). Findings are reported with severity; nothing here blocks a QA
  gate. Only the firm rules (no invented citations/numbers, no placeholders in
  finished text) are non-negotiable.
- **No new facts on revision.** A style pass polishes wording; it never adds
  metrics, claims, or connectives the source does not state (RULE-H stays
  flagged, never "fixed" by inventing support).

## Upstream

This group vendors [agent-style v0.9.0](https://github.com/yzhao062/agent-style)
(CC-BY-4.0, Yue Zhao): the 21-rule prose ruleset plus its `style-review`
skill, fixture corpus, detectors, and revision prompt. When agent-style
releases, bump the vendored copies under
`style-review/references/` and the ruleset it cites; the skill's
self-verification handshake names the pinned version.
