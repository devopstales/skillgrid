---
name: acceptance-test-authoring
description: "Use when authoring BDD acceptance tests from a requirement, before implementation. Derives the acceptance.feature scenarios from intent."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  source: derived from intent-driven-template's acceptance-test-authoring skill
---

# Acceptance Test Authoring

**Announce at start:** "I'm using the skillgrid:acceptance-test-authoring skill to derive the acceptance scenarios."

## Overview

This skill derives BDD acceptance specs (Gherkin) from a requirement's intent, **before** implementation. Specs live as Markdown `acceptance.feature` files under `.skillgrid/specs/`; the runner extracts them into real `.feature` files and runs them against the running application. BDD is always on (non-negotiable, like TDD) — a green acceptance suite is the gate for archive.

## When to Use

- When authoring or updating BDD acceptance specs from a requirement, before implementation.
- When converting a spec/blueprint into Gherkin scenarios (capability → requirement → scenario).

**When NOT to use:** For unit tests or integration tests — this is acceptance/BDD only. The Zone Rule keeps `.skillgrid/specs/` (spec zone) separate from code: this skill authors the spec zone, not the code zone.

The acceptance suite executes Gherkin specs that live under `.skillgrid/specs/` against the running application. BDD is always on (non-negotiable, like TDD) — `bdd.enabled` is always `true` in `.skillgrid/config.yaml`. Specs are Markdown files named `acceptance.feature`: Markdown headings carry the capability, requirement, and scenario structure, while ` ```gherkin ` fences contain only Given/When/Then steps. The runner extracts them into real `.feature` files on every run, synthesizing `Feature:`/`Rule:`/`Scenario:` from the headings.

Everything in this file is stack-agnostic. Tool-specific filenames, dependencies, commands, and examples live in the stack pack.

## Choosing The Stack

The project's acceptance stack is declared as `bdd.stack:` in `.skillgrid/config.yaml`:

```yaml
bdd:
  enabled: true
  stack: javascript      # javascript (cucumber-js)
```

Resolve it in this order:

1. Use `bdd.stack:` in `.skillgrid/config.yaml`.
2. If absent and `acceptance-tests/` already exists, infer it from contents: `cucumber.cjs` means `javascript`; offer to record it.
3. Otherwise ask. Never guess silently, and never scaffold a runner without a recorded value.

## Reference Files

| Stack | Pack | Runner |
| --- | --- | --- |
| `javascript` | [references/javascript/SETUP.md](references/javascript/SETUP.md) | cucumber-js |

Each pack has a **Files to copy** table naming every destination filename and why it is load-bearing. Copy those files verbatim; they are the canonical runner.

Shared files at the `references/` root:

| File | Role |
| --- | --- |
| [EXTRACTION.md](references/EXTRACTION.md) | Normative contract for `acceptance.feature` to `.feature` extraction |
| [gherkin-lintrc.json](references/gherkin-lintrc.json) | Shared lint configuration copied to `acceptance-tests/.gherkin-lintrc` |

The Markdown contract is the definition; the JavaScript files are bindings. Change the contract first, then the implementation.

## Spec Format And Extraction

A spec is `.skillgrid/specs/YYYY-MM-DD-<topic>/acceptance.feature`. Structure comes from Markdown headings; fences hold steps only.

- `# <capability>` is the single H1 title and becomes `Feature:`.
- `### Requirement: <name>` becomes `Rule:`. Its SHALL/MUST description remains plain prose.
- `#### Scenario: <name>` or `#### Scenario Outline: <name>` becomes the corresponding Gherkin scenario. Each scenario must have a following gherkin fence before the next heading.
- Fences open with ` ```gherkin ` at column 0 and close with at least as many backticks at column 0.
- Fences contain only steps, `Examples:` tables, and docstrings. Gherkin structure keywords inside a fence are a hard error.

Extraction writes each `acceptance.feature` to `acceptance-tests/.extracted/<same-relative-path>/acceptance.feature`, preserving exactly one output line per input line. `.extracted/` is gitignored, wiped and rebuilt on every run, and never edited by hand.

[references/EXTRACTION.md](references/EXTRACTION.md) is the normative definition of the complete mapping, fence mechanics, edge cases, and hard errors. Read it before modifying or porting an extractor.

## Scenario Classification

Scenarios are classified by naming convention (not Gherkin tags — the extracted Gherkin carries no tags):

| Prefix | Meaning |
| --- | --- |
| `happy path` | Happy path — must cover Definition of Done user-visible criteria |
| `edge` | Edge case / boundary |
| `failure` | Failure state / error handling |

Rules:
- Every requirement has ≥1 happy + edge + failure scenario.
- Scenario names are unique and referenceable from blueprint tasks (`SATISFIES:`) and `tasks.md`.
- The `@p0` / `@p1` priority and `@step-NN` trace markers live in the blueprint/tasks `SATISFIES` field, not in the Gherkin.

## Authoring Rules (Gherkin Discipline)

When drafting or editing Gherkin, follow these rules (folded from the gherkin-authoring discipline):

- **Domain language.** Use the project's glossary terms (`.skillgrid/glossary/`). Never use implementation jargon (class names, method names, DB column names) in steps.
- **Observable outcomes.** `Then` steps state what the user or system observes — a response, a state change, a message. Never assert internal state ("the cache is populated").
- **One behavior per scenario.** If a scenario needs a second `And` clause that introduces a new precondition, split it.
- **Falsifiable.** Every scenario must be able to fail. "The system works correctly" is not a step.
- **Given/When/Then balance.** A scenario with no `When` is a precondition list, not a scenario. A scenario with no `Then` proves nothing.
- **No implementation details in scenario names.** "Login succeeds with valid credentials" not "POST /auth returns 200".

## Runner Invariants

1. `acceptance-tests/` is an independent test project at the repo root. Its hooks boot the application before the suite and shut it down after, so the suite must run with a single command.
2. The default run executes every `acceptance.feature` under `.skillgrid/specs/`.
3. A green suite is the gate for archive, and archive must never change suite results.
4. Every test run generates an HTML report under `acceptance-tests/reports/`.
5. Verify extraction whenever the runner config, extractor, or `.skillgrid/specs/` tree changes.

## Linting Specs

Spec linting is shared: gherkin-lint over the extracted output with the pinned `.gherkin-lintrc`.

- Extract first, then lint; pass `.extracted` as a directory argument.
- gherkin-lint has no default rules; it requires `.gherkin-lintrc`.
- Reported line numbers are valid in source `acceptance.feature` files.

Before an `acceptance-tests/` project exists, run the extractor from this skill:

```sh
node .agents/skills/acceptance-test-authoring/references/javascript/extract-gherkin.cjs .skillgrid/specs acceptance-tests/.extracted \
  && npx gherkin-lint --config .agents/skills/acceptance-test-authoring/references/gherkin-lintrc.json acceptance-tests/.extracted
```

## Page Object Model

Step definitions must read as intent; all UI knowledge lives in page objects.

- Page objects live under `acceptance-tests/support/pages/`, one per screen or flow.
- Page objects encapsulate routes, form field names, selectors, and ids.
- Parse responses with the stack's HTML parser, never with regexes over raw HTML.
- Page objects expose intent-level methods such as `open()`, `submit_signup(...)`, `error_message()`, and `confirmation_link()`.
- Step definitions contain no selectors, regexes, or URLs; only page-object calls and assertions.
- The World stays a thin HTTP client and state holder.

## Workflow Cadence

Implement one pending step definition at a time: run the suite so the step fails for the right reason, implement until it passes, then commit. The suite's red scenarios at propose time are the change's work list. Finish only when every scenario passes with zero pending or undefined steps and the HTML report is generated.

## Zone Rule

BDD is always on, so the `.skillgrid/specs/` directory is always the **spec zone** and the rest of the codebase is the **code zone**.

- Edit `.skillgrid/specs/` **or** code in a single commit — never both uncommitted. The `pre-commit` zone guard (skillgrid:work-unit-commits) enforces this: a commit staging both a spec and code files is blocked.
- Commit specs before code: the spec is the contract; the code satisfies it.
- A reviewer reading a PR must be able to see spec changes and code changes as separate, ordered commits.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I'll write the spec after the code works" | The spec is the contract, committed before code (Zone Rule). Post-hoc specs describe the code, not the requirement. |
| "One happy-path scenario is enough" | Every requirement has ≥1 happy + edge + failure scenario (Scenario Classification). One happy path proves nothing about boundaries or errors. |
| "I'll skip the lint" | gherkin-lint over the extracted output catches structural errors the eye misses. Extract first, then lint — always. |
| "Internal state is easier to assert" | `Then` steps state what the user or system observes, never internal state ("the cache is populated" is not a step). |
| "I'll use the class name in the step" | Steps use glossary domain language, never implementation jargon. "POST /auth returns 200" is not a scenario name. |

## Red Flags

- A `Then` step asserts internal state instead of an observable outcome.
- A scenario has no `When` (a precondition list) or no `Then` (proves nothing).
- A single `And` clause introduces a second precondition — the scenario needs splitting.
- Implementation jargon (class, method, DB column) or a URL appears in a step or scenario name.
- The step definition contains a selector, regex, or URL instead of a page-object call.
- A commit stages spec and code files together (Zone Rule violation — the zone guard blocks it).

## Verification

- [ ] Spec passes the linter with 0 errors: `gherkin-lint --config .gherkin-lintrc.json acceptance-tests/.extracted` exits 0.
- [ ] Every requirement has ≥1 `happy path` + `edge` + `failure` scenario (Scenario Classification).
- [ ] Each scenario maps to a spec requirement and is referenceable from blueprint tasks (`SATISFIES:`) and `tasks.md`.
- [ ] Scenario classification (zone) applied — prefixes `happy path` / `edge` / `failure` are present and unique.
- [ ] Runner invariants satisfied: suite boots the app with one command, the HTML report is generated under `acceptance-tests/reports/`, and the suite runs green.
- [ ] Zone Rule honored: specs committed before code, and no commit stages both spec and code files.
