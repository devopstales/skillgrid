# Code Style Group

Skills in this group govern **how skillgrid output is shaped** — the prose
conventions for Markdown the agent writes or reviews, and the shared design
vocabulary for the code it structures.

This is the home for style rules, style reviews, and the design vocabulary that
keeps code interfaces consistent. It is separate from
[../craft](../craft/), which is about *how skills are built* (anatomy, effort
budgets, anti-rationalization), and from [../verification](../verification/),
which gates correctness. A skill lives here when its output is **language**
(writing rules, design vocabulary) rather than a gate on whether the code or
process is right.

## Members

| Skill | What it does |
|---|---|
| [`style-review`](style-review/) | Post-hoc audit of Markdown prose against agent-style's 21 rules (RULE-01..12 canonical + RULE-A..I field-observed): deterministic + semantic scorecard, on-confirm polish to `FILE.reviewed.md`, and A/B delta between two drafts. |
| [`codebase-design`](codebase-design/) | Shared vocabulary for designing deep modules (module, interface, depth, seam, adapter, leverage, locality). The design language `writing-blueprints` and `architecture-decision-records` draw on for "deep vs shallow" interface judgments. |

## The two axes

- **Prose (writing style).** The agent-style v0.9.0 rule set (ADR-0039) in two
  phases: **generation-time** — the compact 21-rule directives at
  [`../_shared/rules/prose-style.md`](../_shared/rules/prose-style.md) (shared
  convention, referenced from `AGENTS.md` and `skill-anatomy.md`); agents apply
  them *while writing*. **Review-time** — the `style-review` skill, the opt-in
  post-hoc audit against the same rules (scorecard, write-beside polish, A/B
  compare).
- **Code design vocabulary.** `codebase-design` is the naming standard for
  interface-design talk: "module" not "component", "seam" not "boundary",
  "depth" as leverage at the interface. It is a vocabulary, not a gate — other
  skills invoke it when their step is a design decision about a seam or
  interface.

## Group invariants

- **Vocabulary is load-bearing.** When a skill uses a term from
  `codebase-design` (module, interface, seam, depth, adapter), it uses it with
  that skill's exact definition. Substituting a synonym ("component",
  "boundary", "API") is the failure mode the skill exists to prevent.
- **Write-beside, never in-place.** Style skills report against a file and
  write revised output to a sibling (`FILE.reviewed.md`); the source is never
  mutated by the skill.
- **Rules are defaults, not gates.** Prose rules are judgment rules
  (agent-style's escape hatch: break a rule sooner than say something
  barbarous); design vocabulary is a naming standard, not a correctness check.
  Nothing in this group blocks a QA gate. The firm prose rules (no invented
  citations/numbers, no placeholders in finished text) are non-negotiable.
- **No new facts on revision.** A style pass polishes wording; it never adds
  metrics, claims, or connectives the source does not state (RULE-H stays
  flagged, never "fixed" by inventing support).

## Upstream

The group vendors [agent-style v0.9.0](https://github.com/yzhao062/agent-style)
(CC-BY-4.0, Yue Zhao): the 21-rule prose ruleset plus its `style-review`
skill, fixture corpus, detectors, and revision prompt. When agent-style
releases, bump the vendored copies under
`style-review/references/` and the ruleset it cites; the skill's
self-verification handshake names the pinned version.
