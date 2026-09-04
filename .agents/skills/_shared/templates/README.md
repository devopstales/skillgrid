# SDD artifact templates

Canonical blanks for skillgrid SDD v3 change artifacts. **Naming:** every template is `template-<kebab-case>` (lowercase, hyphenated). Skills copy the template, fill placeholders, and write to the change folder.

Inspired by [skillgrid `.skillgrid/templates`](https://github.com/devopstales/skillgrid/tree/main/.skillgrid/templates) and Superpowers spec/plan artifacts (Goal / Architecture / Global Constraints / Error handling / Testing strategy / TDD micro-cycle with `Run:` + `Expected:`).

| File | Written by | Destination |
|------|------------|-------------|
| `template-change.md` | `sdd-propose` | `docs/skillgrid/changes/<NNN-slug>/change.md` |
| `template-tasks.md` | `sdd-spec` | `docs/skillgrid/changes/<NNN-slug>/tasks.md` |
| `template-acceptance.feature` | `sdd-spec` | `docs/skillgrid/changes/<NNN-slug>/acceptance.feature` |

## Usage (mandatory)

1. Read the matching template from this directory **before** writing the artifact.
2. Copy its structure verbatim — do not invent a parallel outline.
3. Replace every `<placeholder>`; delete optional sections only when marked optional and truly N/A (leave an `N/A: reason` line, do not silently drop required sections).
4. Keep checkbox syntax (`- [ ]`) for Definition of Done and tasks so apply/verify can mark progress.
5. On `tasks.md`, every verify line MUST use `Run: <command>` — `Expected: FAIL|PASS` (Superpowers plan contract).
6. Layout contract: [`../conventions/sdd-structure.md`](../conventions/sdd-structure.md).
