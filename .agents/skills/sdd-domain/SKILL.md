---
name: sdd-domain
description: "Bootstrap docs/skillgrid/glossary/{business,technical}.md (sibling of agents/). Use when onboarding, seeding domain vocabulary, or refreshing the Skillgrid glossary stubs."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  family: sdd
---

# SDD Domain

Bootstrap shared vocabulary under `docs/skillgrid/glossary/` — **sibling of** `agents/`, not `agents/glossary/`. No root `CONTEXT.md` by default (cite glossary + `config.yaml` `context:` instead).

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).  
Ongoing term discipline: load `glossary` when authoring specs later.

## Hard Rules

- Paths are `docs/skillgrid/glossary/business.md` and `technical.md` — never nest under `agents/`.
- Stub tables are enough at onboard; do not invent a large fake glossary.
- Prefer existing project terms (README, domain docs, code names) over new jargon.
- No companion `*-glossary-reference.md` files.

## Workflow

```
[ ] 1. Ensure docs/skillgrid/ exists (else sdd-init first)
[ ] 2. Create glossary/ stubs if missing
[ ] 3. Seed known terms (optional short interview)
[ ] 4. Optional Mnemonic upsert
[ ] 5. Summarize paths
```

### 1. Preconditions

Need `docs/skillgrid/config.yaml` (or skeleton). If uninitialized → hand off to `sdd-init` / `sdd-onboard`.

### 2. Stub files

Create directory + files if absent:

`docs/skillgrid/glossary/business.md` — domain, product, workflow terms.  
`docs/skillgrid/glossary/technical.md` — architecture, platform, protocol terms.

Minimum table:

```markdown
# Business glossary

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
```

(Same header pattern for technical.)

If files exist with the **old** path `docs/skillgrid/agents/glossary/`, migrate content into the sibling `glossary/` and leave a one-line pointer in the old location (or delete only if empty and user agrees).

### 3. Seed (light)

Optional 2–5 terms from README / user interview. Skip generic dictionary words. For close matches later, `glossary` owns the close-term question.

### 4. Mnemonic (optional)

Upsert topic_keys: `sdd/<project>/glossary/business`, `sdd/<project>/glossary/technical` (`scope: project`).

### 5. Summary

Paths created/updated. Remind: no root CONTEXT.md SoT; product narrative stays in `config.yaml` `context:` + glossary.

## Gotchas

- Old skills/docs may still say `agents/glossary/` — **sibling** `glossary/` wins per sdd-structure.
- Do not dump the whole product brief into glossary rows — short definitions only.
- `glossary` skill may still list the old path until updated; when conflicting, follow `sdd-structure.md`.
