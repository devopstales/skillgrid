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
| [`go`](go/) | Google's official Go style guide: gofmt, naming, error handling, interfaces, concurrency, package organization. |
| [`typescript`](typescript/) | Google's official TypeScript style guide: strict mode, types, interfaces, null handling, naming, imports. |
| [`javascript`](javascript/) | Google's official JavaScript (ES6+) style guide: const/let, arrow functions, template literals, JSDoc. |
| [`python`](python/) | Google's official Python style guide (PEP 8 +): type annotations, docstrings, imports, comprehensions. |
| [`java`](java/) | Google's official Java style guide: naming, formatting, Javadoc, exceptions, lambdas, streams. |
| [`cpp`](cpp/) | Google's official C++ style guide: headers, naming, RAII, smart pointers, modern C++. |
| [`csharp`](csharp/) | Google's official C# style guide: naming, LINQ, async/await, XML docs. |
| [`html-css`](html-css/) | Google's official HTML/CSS style guide: formatting, naming, semantics, accessibility. |
| [`angularjs`](angularjs/) | Google's official AngularJS style guide: controllers, services, directives, modules. |
| [`objective-c`](objective-c/) | Google's official Objective-C style guide: naming, memory management, protocols. |
| [`swift`](swift/) | Google's official Swift style guide: naming, optionals, protocols, error handling. |
| [`shell`](shell/) | Google's official Shell scripting style guide: Bash, naming, error handling, portability. |
| [`r`](r/) | Google's official R style guide: naming, formatting, functions, documentation. |
| [`common-lisp`](common-lisp/) | Google's official Common Lisp style guide: naming, macros, packages. |
| [`vim-script`](vim-script/) | Google's official Vim script style guide: plugin structure, naming, portability. |
| [`json`](json/) | Google's official JSON style guide: formatting, naming, structure. |
| [`markdown`](markdown/) | Google's official Markdown style guide: formatting, structure, links, lists. |

## The three axes

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
- **Language style guides.** 17 Google official style guides (ADR-0039), one
  skill per language. Each fires when the agent writes, reviews, or refactors
  code in that language. The guides are advisory (naming, formatting, common
  patterns) and defer to the repo's formatter (`gofmt` for Go, `tsc` for
  TypeScript, etc.) where the two disagree.

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

Two upstreams, both vendored:

- **agent-style v0.9.0** (CC-BY-4.0, Yue Zhao) — the 21-rule prose ruleset
  plus its `style-review` skill, fixture corpus, detectors, and revision
  prompt. When agent-style releases, bump the vendored copies under
  `style-review/references/` and the ruleset it cites; the skill's
  self-verification handshake names the pinned version.
- **Google style guides** (MIT, TestDino 2026, from
  [testdino-hq/google-styleguides-skills](https://github.com/testdino-hq/google-styleguides-skills))
  — 17 language style guides derived from
  [google.github.io/styleguide](https://google.github.io/styleguide/). One
  skill directory per language; each `SKILL.md` is the quick-reference entry
  point, each `<lang>.md` is the full guide. When the upstream updates,
  re-copy the language directories and re-apply the skillgrid frontmatter
  (license, metadata, based_on, description trigger).
