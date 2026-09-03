---
name: glossary
description: Use when authoring or reviewing specs, requirements, design docs, ADRs, tasks, glossary entries, domain terms, technical terms, or wording consistency across skillgrid artifacts. Enforces term reuse before inventing new terms.
license: MIT
metadata:
  author: skillgrid
  version: "1.0"
  source: derived from intent-driven-template's glossary skill
---

# Glossary

Use existing repository terms before creating new ones. Vocabulary drift across `intent.md` / `plan.md` is the most common cause of contradictory decisions in skillgrid changes.

## Workflow

1. Identify the artifact being authored or reviewed (`intent.md`, `plan.md`, `research.md`, `tasks.md`, `acceptance.feature`, `verification.md`, ADR, ADR-equivalent, etc.).
2. Read `docs/skillgrid/glossary/business.md` and `docs/skillgrid/glossary/technical.md` when present.
3. Extract project-specific, technical, overloaded, ambiguous, or repeated concepts from the artifact under construction.
4. Compare extracted concepts with existing glossary terms.
5. If existing terms look close, ask the user whether to reuse one or define a new term (see **Close-Term Question** below).
6. Use the chosen term consistently in the artifact.
7. Add new terms to the appropriate glossary file.
8. Always-on companions: every `intent.md` and every `plan.md` MUST have a companion glossary reference file in the same directory.

Do not add generic dictionary words or one-off implementation details.

## Glossary Files

- `docs/skillgrid/glossary/business.md` — domain, product, workflow, and business terms.
- `docs/skillgrid/glossary/technical.md` — architecture, implementation, platform, and protocol terms.

Mnemonic persistence (recovery across sessions): the two files are also saved as observations under `sdd/<project>/glossary/business` and `sdd/<project>/glossary/technical` (topic_key upsert).

If a needed glossary file is missing, create it with this table:

```markdown
| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
```

New entries use the same table format. Keep definitions to one concise sentence.

## Close-Term Question

Ask before creating a new term when existing terms may fit:

```text
These glossary terms look close to "<concept>":

- `<existing-term>`: <why it may fit>
- `<another-term>`: <why it may fit>

Do you want to use one of these existing terms, or define a new term?
```

Do not proceed with a new term until the user answers, unless no plausible existing term exists.

In skillgrid's interactive mode this blocks the orchestrator and is acceptable. In automatic mode, prefer the existing term and surface the choice in the result envelope so the user can override later.

## Marking Terms in Artifacts

In non-glossary artifacts, bold glossary terms that appear in the artifact's companion glossary reference.

Apply bolding in prose only. Do not bold terms in code blocks, frontmatter, links, or places where markdown formatting would make the artifact harder to read.

## Companion Reference (always-on for intent.md and plan.md)

Every `intent.md` and every `plan.md` gets a companion file in the same directory.

- `intent.md` → `intent-glossary-reference.md` in the same change folder.
- `plan.md` → `plan-glossary-reference.md` in the same change folder.

Other artifacts (`research.md`, `tasks.md`, `acceptance.feature`, `verification.md`) are not required to have companions, but may link to the upstream `plan-glossary-reference.md` if they introduce new terms.

Format:

```markdown
# Glossary Reference

| Term | Source Glossary | Context |
| --- | --- | --- |
| <Term> | `docs/skillgrid/glossary/business.md` | <Short context for how the artifact uses this glossary term.> |
```

`Source Glossary` is the glossary file the term came from or was added to.

If no glossary terms are used, write:

```markdown
# Glossary Reference

No glossary terms referenced.
```

Do not copy definitions into companion files.

## Use in skillgrid

- `sdd-propose` calls the close-term question when shaping Business Rules and Success Criteria.
- `sdd-design` calls the close-term question when shaping Architecture Decisions and writes `plan-glossary-reference.md` next to `plan.md`.
- `sdd-apply` updates `docs/skillgrid/glossary/technical.md` if a new technical term lands during implementation (via `mem_save`, topic_key `sdd/<project>/glossary/technical`).
