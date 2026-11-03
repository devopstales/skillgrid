# Extraction contract — `acceptance.feature` → `.feature`

**This file is the definition.** `javascript/extract-gherkin.cjs` is a binding of it. When behavior changes, this file changes first.

## What extraction does

A spec is `.skillgrid/specs/YYYY-MM-DD-<topic>/acceptance.feature`. It is a standard Skillgrid Markdown spec: the **structure** lives in Markdown headings and only the **Given/When/Then steps** live inside column-0 ` ```gherkin ` fences.

Extraction **synthesizes** the Gherkin structure from the headings and copies fenced step lines verbatim, writing each `acceptance.feature` to `acceptance-tests/.extracted/<same-relative-path>/acceptance.feature`.

Discovery covers `.skillgrid/specs/`, anchored to the literal basename `acceptance.feature` — so other files in the change directory are excluded structurally.

`.extracted/` is **gitignored, wiped and rebuilt on every run, and never edited by hand**. The wipe is an invariant, not an optimization — a stale extraction would keep deleted or renamed capabilities executing.

## Line fidelity — the core invariant

Every input line maps to **exactly one** output line, so the extracted file has the IDENTICAL line count and **line N of the `.feature` is line N of the `.md`**.

gherkin-lint messages, runner failure locations and line targeting (`acceptance.feature:27:33`) therefore all point at valid source `acceptance.feature` lines with zero translation. Read `.extracted/<id>/acceptance.feature:N` as `.skillgrid/specs/<id>/acceptance.feature:N`, always.

**Never "improve" the extractor to collapse blank lines.** The whole toolchain leans on this.

## The mapping

Every line outside a fence is classified by exactly one row:

| Markdown line (outside any fence) | Emitted Gherkin line |
|---|---|
| `# <title>` (the single H1) | `Feature: <title>` |
| `### Requirement: <name>` | `  Rule: <name>` |
| `#### Scenario: <name>` | `    Scenario: <name>` |
| `#### Scenario Outline: <name>` | `    Scenario Outline: <name>` |
| any line inside a ` ```gherkin ` fence | copied **verbatim**, column unchanged |
| everything else — prose, requirement descriptions, `## Requirements`, other headings, fence markers, non-gherkin fence bodies | blank line |

Heading matching is case-insensitive on the keywords (`Requirement:`, `Scenario:`, `Scenario Outline:`); the emitted Gherkin keyword is always normalized to canonical casing.

Three consequences worth stating explicitly, because each is a decision rather than an accident:

- **Steps are not re-indented.** They keep the column the author wrote. The author **must** indent steps to 6 spaces inside the fence (2 below the `Scenario:` at 4 spaces) so the extracted Gherkin parses. Verbatim copying keeps runner error columns pointing at real source text, and this is why `indentation` is off in the pinned lint config.
- **Requirement description prose is blanked, not emitted as a `Rule:` description.** Free text that could accidentally parse as a Gherkin keyword is exactly the failure this format exists to avoid. The SHALL/MUST sentence stays visible in the rendered Markdown.
- **The `## Requirements` heading and other section headings are blanked.** They carry organizational structure in Markdown but no Gherkin meaning.

## Fence mechanics

Fences follow CommonMark:

- An opener is 3+ backticks at **column 0** with info string **exactly** `gherkin`.
- The closer is at least as many backticks at column 0.
- Non-gherkin fences are tracked too, so a ` ```gherkin ` quoted inside a longer documentation fence cannot false-trigger.
- Gherkin docstrings delimited by ` ``` ` are safe: they are always indented, and the closer requires column 0.

A fence holds **only** steps (Given/When/Then/And/But), plus `Examples:` tables and docstrings.

## Edge cases and hard errors

All deliberate — silent drops are the failure mode to fear:

| Case | Behavior |
|---|---|
| Unclosed fence | Error with `file:line` of the opener |
| No H1 title | **Hard error** — there would be no `Feature:` |
| More than one H1 | **Hard error** — an acceptance.feature is exactly one capability |
| `#### Scenario:` with no fence before the next heading (or before EOF) | **Hard error** — a scenario with no steps would silently pass |
| `Feature:` / `Rule:` / `Scenario:` / `Scenario Outline:` / `Example:` inside a gherkin fence | **Hard error** — structure comes from headings |
| Zero gherkin fences in an `acceptance.feature` | Fine — a spec with only prose (e.g. a placeholder) legitimately has no steps |
| `Examples:` and `Background:` inside a fence | Allowed — both legitimately belong there |
| Indented ` ```gherkin ` opener | **Hard error** — silently ignoring it would silently drop scenarios |
| ` ```gherkin extra-text ` | Not a gherkin opener (info string must be exactly `gherkin`) — treated as an ordinary fence, contents blanked |
| Non-gherkin fences (` ```js `, plain ` ``` `, 4+ backticks) | Tracked, contents blanked |
| Gherkin docstrings delimited by ` ``` ` | Safe — docstrings are indented; fence closers require column 0 |
| Files other than `acceptance.feature` | Ignored — discovery is anchored to `acceptance.feature` |
