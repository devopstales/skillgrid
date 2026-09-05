---
name: sdd-explore
description: Optional change-scoped research helper — write research.md (lifetime = this change; may rot) when explore is hard (external API, rare docs, costly rediscovery). Use when sdd-propose or use-skillgrid needs hard research before locking change.md; not a top-level v4 stage.
disable-model-invocation: true
license: MIT
metadata:
  author: devopstales
  version: "4.0"
  part-of: skillgrid
---

# SDD Explore

**Helper, not a top-level stage.** v4 pipeline is `onboard → propose → spec → apply ⇄ verify → archive`. Explore runs only when research is hard; fresh agents **read the cache** instead of re-exploring blindly.

## When to run

Run when any of:

- External / poorly documented API
- Rare or scattered docs that are costly to rediscover
- `sdd-propose` / orchestrator stops because research is missing or stale

Skip for ordinary in-repo features the code index already covers.

## Hard Rules

- ONLY writable artifact: `docs/skillgrid/changes/<NNN-slug>/research.md` (plus Mnemonic upsert).
- Do **not** promote to `docs/skillgrid/codebase/` or `docs/adr/` by default.
- Do **not** allocate NNN — `sdd-propose` reserves it. Create/update under an existing or reserved slug.
- Always read real code / docs — never invent the codebase.
- Hybrid: filesystem + Mnemonic `sdd/<NNN-slug>/research`.

## Workflow

```
[ ] 1. Check cache first
[ ] 2. Clarify the question
[ ] 3. Investigate (call investigate when needed)
[ ] 4. Compare approaches
[ ] 5. Write research.md with lifetime header
[ ] 6. Return envelope
```

### 1. Check cache first

- Read `docs/skillgrid/changes/<NNN-slug>/research.md` if present → UPDATE, do not blind overwrite.
- `mem_search` → `mem_get_observation` for `sdd/<NNN-slug>/research`.
- Read `docs/skillgrid/config.yaml`; skim related `archive/` changes for prior art.
- Code ladder: `code_status` → `code_index` if stale → `code_search` → `code_read`.

If existing research answers the question and is not stale, return it — do not re-explore.

### 2. Clarify the question

Feature / bug / refactor? Domain? If too vague to investigate, stop and state what clarification is needed.

### 3. Investigate

Prefer **`investigate`** for high-trust primary sources (external APIs, rare docs). For large surfaces, `dispatching-parallel-agents`. Map: entry points, affected modules, existing tests/gaps, coupling.

### 4. Compare approaches

| Approach | Pros | Cons | Effort |
|---|---|---|---|
| A | … | … | Low/Med/High |

Recommendation ≠ commitment — propose locks the choice.

### 5. Write research.md

Path: `docs/skillgrid/changes/<NNN-slug>/research.md`.

**Mandatory lifetime header** (first lines):

```markdown
> **Lifetime:** This change only (`<NNN-slug>`). May rot. Do not promote to
> `docs/skillgrid/codebase/` or `docs/adr/` by default. Fresh agents: read this
> cache before re-exploring.
```

Then:

```markdown
## Research: {topic}

### Current State
### Affected Areas
### Approaches
### Recommendation
### Risks
### Ready for Proposal
```

Mnemonic upsert `sdd/<NNN-slug>/research` (full markdown). Apply `rules.explore` from config if present.

### 6. Return envelope

```markdown
**Status**: success | partial | blocked
**Summary**: …
**Artifacts**: docs/skillgrid/changes/<NNN-slug>/research.md · Mnemonic sdd/<NNN-slug>/research
**Next**: sdd-propose (or design-spike if taste/unknown shape remains)
**Risks**: …
```

## Gotchas

- Re-exploring when `research.md` already answers the question wastes context — read the cache first.
- `mem_search` returns previews — always `mem_get_observation(id)`.
- You do not own NNN; do not invent a parallel numbering scheme.
- Recommendation is analysis, not a locked Architecture decision — that lives in `change.md`.

## References

- [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md)
- [`../_shared/conventions/mnemonic-memory.md`](../_shared/conventions/mnemonic-memory.md)
- [`../_shared/conventions/mnemonic-code-indexing.md`](../_shared/conventions/mnemonic-code-indexing.md)
- [`../investigate/SKILL.md`](../investigate/SKILL.md)
- [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md)
