# Domain foundation specialist

Model the problem before modelling the code: where the boundary is, what must always be true, who acts, what data moves, and how it fails.

When to load: Phase 1, and Phase 2 when a seam is unclear. Role: domain modeller.

## The seven tools

1. **Boundary** — one sentence for what the program is responsible for, and one for what it is not. Every later scope argument is settled here.
2. **Invariants** — the statements that must hold after every operation. Write them as testable claims ("an order total is never negative"; "a rename never loses the file's contents"), not as aspirations.
3. **Actors** — who or what calls the program, what each is allowed to do, and what each must never do. Include the machine actors: cron, CI, another service.
4. **Data shapes** — the core records, their identity, their lifecycle, and which fields are derived. Name the one-way doors: persisted schema, public API, file format, wire protocol.
5. **Failure modes** — for each operation: what happens on bad input, missing dependency, partial write, timeout, retry, concurrent call. The failure path is part of the domain, not an afterthought.
6. **Contradictions** — two requirements that cannot both hold. Name them, choose one, and record the choice in an ADR. A design where nothing conflicts is under-examined.
7. **Contracts** — the interface, the error model, and the config precedence. One error taxonomy for the program; no module invents its own.

## Outputs

- `glossary/01-business-terms.md` and `glossary/02-technical-terms.md`, one meaning per term, no synonyms.
- `invariants/05-locked-constraints.md`, the testable claims.
- `entities/` — one page per thing the program touches: services, APIs, data stores, user roles, external systems.
- `concepts/` — one page per idea: invariants, patterns, error model, config precedence.
- `decisions/NNNN-slug.md`, one per locked choice, plus the pointer row in `ASSUMPTIONS.md` § LOCKED.
- `index.md`, updated with every new page.
- `log.md`, ingest entry for each page.
- The interface, error and config-precedence sections the blueprint will consume.
- The applicable rows of `skillgrid:_shared/references/threat-matrix.md`, with Applicable / Not applicable and a reason.

## Rules

- Prefer a contradiction found now to a bug found in Phase 5.
- Every invariant gets a test in Phase 3. An invariant with no test is a wish.
- An error model is a contract: codes, messages, and what the caller should do. "Something went wrong" is not an error.
- Config precedence is written once and tested: flag over env over file over default.
- If a term has two meanings, split it. A synonym in the code is a defect.
- A one-way door gets an ADR before it gets code. Reversibility is a design property, not a promise.
