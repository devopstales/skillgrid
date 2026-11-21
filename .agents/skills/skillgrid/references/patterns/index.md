# Pattern index

The measured patterns this skill audits against, each with a source and a scope. Load the one a phase needs; do not load them all.

When to load: as `references/pipeline/manifest.yaml` lists them per phase. Role: any reviewer.

| File | Covers | Load in |
|---|---|---|
| `code-patterns.md` | function/file size, nesting, naming, duplication, error and test shape | Phase 3, 4, 5 |
| `structure-and-risk.md` | layering, seams, coupling, one-way doors, blast radius, operability | Phase 2, 4 |
| `ai-tells-code.md` | the machine-written tells a reviewer learns to see in code | Phase 4, 5 |
| `glossary.md` | the roles and terms this skill uses | Phase 0 |
| `skillgrid-folder.md` | the `.skillgrid/` layout the program writes | Phase 0 |
| `glossary.md` | the seed terms for `glossary/`, `entities/` and `concepts/` pages | Phase 0, 1 |
| `assumptions.md` | the `ASSUMPTIONS.md` template | Phase 0 |
| `architecture.md` | the `ARCHITECTURE.md` skeleton | Phase 0 |
| `config.yaml` | the `.skillgrid/config.yaml` template | Phase 0 |
| `agent-config.md` | how a host agent file points at this skill | Phase 0 |
| `architecture.md` | the durable architecture note template | Phase 2 |

## How the patterns are used

- **As a standard, not a score.** A pattern names a defect and its consequence. The reviewer quotes the passage; the score evaluator decides what it is worth.
- **With a source.** Every pattern states where it comes from: a measured range, a named guide, or this skill's own experience. A pattern with no source is a preference.
- **As a floor, not a ceiling.** Meeting every pattern does not make a program good. Failing one does not make it bad. The consequence decides.
- **Deliberately broken sometimes.** A long function with one linear job, a duplicated three-line block that reads better inline, a wide parameter list on a constructor: name the reason, keep it, and do not "fix" it.

## What is not here

- Language style guides. Use the repo's own linter and formatter; this skill audits structure and risk, not taste in braces.
- Prose style. Artifacts follow `skillgrid:_shared/rules/prose-style.md`.
- A PT-BR prose-tells file. Code is language-neutral; the prose artifacts follow the shared style rule, and a second copy here would drift.
