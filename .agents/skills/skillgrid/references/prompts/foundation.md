# Phase 1: Foundation

Give the program a domain worth modelling, a vocabulary worth keeping, decisions worth recording, and contracts that survive the first refactor.

When to load: Phase 1. Role: orchestrator as domain modeller, with the domain-foundation specialist.

## Method

Follow `references/specialists/domain-foundation.md`. It holds the tools: boundaries, invariants, actors, data shapes, failure modes, contradictions, and the interface/error contract.

## Outputs

- `.skillgrid/projectwiki/glossary/01-business-terms.md` — the product terms, one meaning each.
- `.skillgrid/projectwiki/glossary/02-technical-terms.md` — the stack terms the code will use.
- `.skillgrid/projectwiki/invariants/05-locked-constraints.md` — what the code must never break.
- `.skillgrid/projectwiki/entities/` — one page per thing the program touches: services, APIs, data stores, user roles, external systems.
- `.skillgrid/projectwiki/concepts/` — one page per idea: invariants, patterns, error model, config precedence.
- `.skillgrid/projectwiki/decisions/NNNN-slug.md` — one ADR per locked decision, plus the pointer row in `ASSUMPTIONS.md` § LOCKED.
- `.skillgrid/projectwiki/index.md` — updated with every new page.
- `.skillgrid/projectwiki/log.md` — ingest entries for each page created.
- `.skillgrid/change-specs/<topic>/findings.md` — research, prototype and sketch evidence, if any.
- The interface, data, error and config-precedence sections the blueprint will consume.

## Rules

- Contradiction over consistency: a design where nothing can fail is a finding, not a virtue.
- Every design choice names a comparable project, a measured benchmark, or an ADR. A stack label alone is never evidence ("Go services are fast" is not a reason).
- The threat matrix rows that apply are carried forward: `skillgrid:_shared/references/threat-matrix.md`.
- A one-way door (schema, public API, dependency, data format) gets an ADR before it gets code.
- Terms are recorded once, in the glossary, and used everywhere. A synonym in the code is a defect.

## Check-in

Show the domain boundary, the three or four invariants that matter most, the one-way doors found, the ADRs written, and the assumptions worth correcting.
