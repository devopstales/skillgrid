# Phase 5 — Acceptance Spec (BDD, always on)

The full protocol for authoring the BDD acceptance spec from the validated
blueprint. The SKILL.md Phase 5 section is the summary; this file is the
reference. Merged from the `acceptance-test-authoring` skill (now folded into
the blueprint as Phase 5).

## Overview

Derive BDD acceptance specs (Gherkin) from a requirement's intent, **before**
implementation. Specs live as Markdown `acceptance.feature` files under
`{specs_root}` (default `.skillgrid/change-specs/`); the runner extracts them
into real `.feature` files and runs them against the running application.
BDD is always on (non-negotiable, like TDD) — `bdd.enabled` is always `true`
in `.skillgrid/config.yaml`. A green acceptance suite is the gate for archive.

**Position in artifact order:** briefing → blueprint → validation →
**acceptance spec (this phase)** → tasks → slices.

**When NOT to use:** for unit tests or integration tests — this is
acceptance/BDD only. The Zone Rule keeps `{specs_root}` (spec zone) separate
from code: this phase authors the spec zone, not the code zone.

Everything in this file is stack-agnostic. Tool-specific filenames,
dependencies, commands, and examples live in the stack pack
([javascript/SETUP.md](javascript/SETUP.md)).

## Choosing the stack

The project's acceptance stack is declared as `bdd.stack:` in
`.skillgrid/config.yaml`:

```yaml
bdd:
  enabled: true
  stack: javascript      # javascript (cucumber-js)
```

Resolve it in this order:

1. Use `bdd.stack:` in `.skillgrid/config.yaml`.
2. If absent and `.skillgrid/acceptance-tests/` already exists, infer it from contents:
   `cucumber.cjs` means `javascript`; offer to record it.
3. Otherwise ask. Never guess silently, and never scaffold a runner without a
   recorded value.

Each stack pack has a **Files to copy** table naming every destination
filename and why it is load-bearing. Copy those files verbatim; they are the
canonical runner.

Shared files at this reference's root:

| File | Role |
| --- | --- |
| [EXTRACTION.md](EXTRACTION.md) | Normative contract for `acceptance.feature` to `.feature` extraction |
| [gherkin-lintrc.json](gherkin-lintrc.json) | Shared lint configuration copied to `.skillgrid/acceptance-tests/.gherkin-lintrc` |

The Markdown contract is the definition; the JavaScript files are bindings.
Change the contract first, then the implementation.

## Spec format and extraction

A spec is `{specs_root}/YYYY-MM-DD-<topic>/acceptance.feature`. Structure
comes from Markdown headings; fences hold steps only.

- `# <capability>` is the single H1 title and becomes `Feature:`.
- `### Requirement: <name>` becomes `Rule:`. Its SHALL/MUST description
  remains plain prose.
- `#### Scenario: <name>` or `#### Scenario Outline: <name>` becomes the
  corresponding Gherkin scenario. Each scenario must have a following gherkin
  fence before the next heading.
- Fences open with ` ```gherkin ` at column 0 and close with at least as many
  backticks at column 0.
- Fences contain only steps, `Examples:` tables, and docstrings. Gherkin
  structure keywords inside a fence are a hard error.

Extraction writes each `acceptance.feature` to
`.skillgrid/acceptance-tests/.extracted/<same-relative-path>/acceptance.feature`,
preserving exactly one output line per input line. `.extracted/` is
gitignored, wiped and rebuilt on every run, and never edited by hand.

[EXTRACTION.md](EXTRACTION.md) is the normative definition of the complete
mapping, fence mechanics, edge cases, and hard errors. Read it before
modifying or porting an extractor.

## Scenario classification

Scenarios are classified by naming convention (not Gherkin tags — the
extracted Gherkin carries no tags):

| Prefix | Meaning |
| --- | --- |
| `happy path` | Happy path — must cover Definition of Done user-visible criteria |
| `edge` | Edge case / boundary |
| `failure` | Failure state / error handling |

Rules:

- Every requirement has ≥1 happy + edge + failure scenario.
- Scenario names are unique and referenceable from blueprint tasks
  (`SATISFIES:`) and `tasks.md`.
- The `@p0` / `@p1` priority and `@step-NN` trace markers live in the
  blueprint/tasks `SATISFIES` field, not in the Gherkin.

## Authoring rules (Gherkin discipline)

- **Domain language.** Use the project's terms
  (`glossary/01-business-terms.md` / `02-technical-terms.md`, per
  `references/adr-business-template.md` / `adr-technical-template.md`). Never use
  implementation jargon (class names, method names, DB column names) in steps.
- **Observable outcomes.** `Then` steps state what the user or system
  observes — a response, a state change, a message. Never assert internal
  state ("the cache is populated").
- **One behavior per scenario.** If a scenario needs a second `And` clause
  that introduces a new precondition, split it.
- **Falsifiable.** Every scenario must be able to fail. "The system works
  correctly" is not a step.
- **Given/When/Then balance.** A scenario with no `When` is a precondition
  list, not a scenario. A scenario with no `Then` proves nothing.
- **No implementation details in scenario names.** "Login succeeds with valid
  credentials" not "POST /auth returns 200".

## Syntax validation (mandatory)

After writing `acceptance.feature`, validate the Gherkin syntax before
presenting it for review. Run the BDD runner in dry-run/parse mode (e.g.
`cucumber-js --dry-run` for the javascript stack) to confirm the file parses.
Fix any syntax errors before the file is considered complete. Invalid syntax
blocks the User Review Gate.

## User Review Gate (STATUS banner)

The acceptance spec is the contract the code satisfies — it deserves the same
approval ceremony as `briefing.md` and `blueprint.md`. When
`acceptance.feature` is first written, its line-3 STATUS banner is `draft` —
the agent writes it; the user's hand-edit at the User Review Gate sets
`approved`, and the agent never writes `approved` itself (same contract as
`briefing.md` and `blueprint.md`, per `_shared/rules/sdd-structure.md` §
STATUS Banner). The template
([templates/acceptance.feature](../templates/acceptance.feature)) carries the
banner.

After syntax validation passes, present the spec for review:

> "Acceptance spec written to `<path>` (Gherkin, lint-clean, N requirements
> × M scenarios). Please read it — the scenarios are the contract the code
> must satisfy, and they are committed before any code. When you approve,
> edit the STATUS line yourself: change `draft` to `approved (YYYY-MM-DD)`
> on line 3."

Wait for the flip. Confirm by re-reading line 3 from disk and quote the
banner in the reply. If the user requests changes, make them, re-run syntax
validation, and set the banner to `revised` (the user re-reads and re-flips).
**Never proceed to Phase 6 without the banner showing `approved`.**

## Linting specs

Spec linting is shared: gherkin-lint over the extracted output with the pinned
`.gherkin-lintrc`.

- Extract first, then lint; pass `.extracted` as a directory argument.
- gherkin-lint has no default rules; it requires `.gherkin-lintrc`.
- Reported line numbers are valid in source `acceptance.feature` files.

Before a `.skillgrid/acceptance-tests/` project exists, run the extractor from this
reference:

```sh
node .agents/skills/blueprint/references/javascript/extract-gherkin.cjs .skillgrid/change-specs .skillgrid/acceptance-tests/.extracted \
  && npx gherkin-lint --config .agents/skills/blueprint/references/gherkin-lintrc.json .skillgrid/acceptance-tests/.extracted
```

## Runner invariants

1. `.skillgrid/acceptance-tests/` is an independent test project under
   `.skillgrid/`. Its hooks boot the application before the suite and shut it
   down after, so the suite must run with a single command.
2. The default run executes every `acceptance.feature` under `{specs_root}`.
3. A green suite is the gate for archive, and archive must never change suite
   results.
4. Every test run generates an HTML report under `.skillgrid/acceptance-tests/reports/`.
5. Verify extraction whenever the runner config, extractor, or `{specs_root}`
   tree changes.

## Page object model

Step definitions must read as intent; all UI knowledge lives in page objects.

- Page objects live under `.skillgrid/acceptance-tests/support/pages/`, one per screen
  or flow.
- Page objects encapsulate routes, form field names, selectors, and ids.
- Parse responses with the stack's HTML parser, never with regexes over raw
  HTML.
- Page objects expose intent-level methods such as `open()`,
  `submit_signup(...)`, `error_message()`, and `confirmation_link()`.
- Step definitions contain no selectors, regexes, or URLs; only page-object
  calls and assertions.
- The World stays a thin HTTP client and state holder.

## Workflow cadence

Implement one pending step definition at a time: run the suite so the step
fails for the right reason, implement until it passes, then commit (per
`_shared/work-unit-commits/README.md`). The suite's red scenarios at propose
time are the change's work list. Finish only when every scenario passes with
zero pending or undefined steps and the HTML report is generated.

## Zone rule

BDD is always on, so `{specs_root}` is always the **spec zone** and the rest
of the codebase is the **code zone** (full commit contract:
`_shared/work-unit-commits/README.md`).

- Commit specs before code: the spec is the contract; the code satisfies it.
- A reviewer reading a PR must be able to see spec changes and code changes
  as separate, ordered commits.
- `blueprint.md` and `acceptance.feature` are both spec-zone files and commit
  together.

## Verification

- [ ] Spec passes the linter with 0 errors:
      `gherkin-lint --config .gherkin-lintrc.json .skillgrid/acceptance-tests/.extracted`
      exits 0.
- [ ] The runner's dry-run/parse mode confirms the file parses.
- [ ] Line-3 STATUS banner is `approved` (the user's hand-edit — re-read from
      disk, banner quoted). The agent never writes `approved`.
- [ ] Every requirement has ≥1 `happy path` + `edge` + `failure` scenario.
- [ ] Each scenario maps to a spec requirement and is referenceable from
      blueprint tasks (`SATISFIES:`) and `tasks.md`.
- [ ] Runner invariants satisfied: suite boots the app with one command, the
      HTML report is generated under `.skillgrid/acceptance-tests/reports/`.
- [ ] Zone rule honored: specs committed before code, and no commit stages
      both spec and code files.
