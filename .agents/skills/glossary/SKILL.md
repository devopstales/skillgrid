---
name: glossary
description: Use when authoring or reviewing specs, requirements, design docs, ADRs, tasks, glossary entries, domain terms, technical terms, or wording consistency across skillgrid artifacts. Enforces term reuse before inventing new terms. Fold terms into main artifacts — no companion glossary-reference files.
license: MIT
metadata:
  author: skillgrid
  version: "2.0"
  source: derived from intent-driven-template's glossary skill
---

# Glossary

Use existing repository terms before creating new ones. Vocabulary drift across `change.md` / `tasks.md` is the most common cause of contradictory decisions in skillgrid changes.

## Workflow

1. Identify the artifact (`change.md`, `research.md`, `tasks.md`, `acceptance.feature`, ADR, etc.).
2. Read `docs/skillgrid/agents/glossary/business.md` and `docs/skillgrid/agents/glossary/technical.md` when present.
3. Extract project-specific, technical, overloaded, ambiguous, or repeated concepts.
4. Compare with existing glossary terms.
5. If close matches exist, ask the user (see **Close-Term Question**).
6. Use the chosen term consistently.
7. Add new terms to the appropriate glossary file.
8. Fold first-use definitions or a short `## Glossary` footer into the **main** artifact (`change.md`). **Do not** create `*-glossary-reference.md` companions.

Do not add generic dictionary words or one-off implementation details.

## Glossary Files

- `docs/skillgrid/agents/glossary/business.md` — domain, product, workflow, and business terms.
- `docs/skillgrid/agents/glossary/technical.md` — architecture, implementation, platform, and protocol terms.

Mnemonic: `sdd/<project>/glossary/business` and `sdd/<project>/glossary/technical` (topic_key upsert).

If a needed glossary file is missing, create it with:

```markdown
| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
```

## Close-Term Question

```text
These glossary terms look close to "<concept>":

- `<existing-term>`: <why it may fit>
- `<another-term>`: <why it may fit>

Do you want to use one of these existing terms, or define a new term?
```

Do not proceed with a new term until the user answers, unless no plausible existing term exists.

## Marking Terms in Artifacts

Bold glossary terms in prose when they appear in the artifact's `## Glossary` footer (or first-use definition). Do not bold in code blocks, frontmatter, or links.

## No companion reference files

SDD v3 forbids `intent-glossary-reference.md` / `plan-glossary-reference.md` / `change-glossary-reference.md`. Terms live in:

1. `docs/skillgrid/agents/glossary/{business,technical}.md`
2. Inline / `## Glossary` in `change.md` (see `template-change.md`)

## Use in skillgrid

- `sdd-propose` — close-term check when writing `change.md`; fill `## Glossary` footer.
- `sdd-spec` — reuse terms from `change.md`; do not invent companions.
- `sdd-apply` — update `docs/skillgrid/agents/glossary/technical.md` if a new technical term lands (`mem_save`, topic_key `sdd/<project>/glossary/technical`).
