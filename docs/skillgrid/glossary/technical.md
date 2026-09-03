# Glossary — Technical

Architecture, implementation, platform, and protocol terms used across skillgrid changes.

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| Module | Anything with an interface and an implementation; deliberately scale-agnostic (function, class, package, slice). | `sdd-design` Architecture Decisions. | "unit", "component", "service". See `codebase-design` skill. |
| Interface | Everything a caller must know to use a module correctly: type signature, invariants, ordering constraints, error modes, configuration, performance. | `sdd-design` Architecture Decisions. | "API", "signature" (too narrow). See `codebase-design` skill. |
| Seam | The location at which a module's interface lives; a place where you can alter behaviour without editing in that place. | `sdd-design` Architecture Decisions, `sdd-apply` testability checks. | "boundary" (overloaded with DDD). See `codebase-design` skill. |
| Adapter | A concrete thing that satisfies an interface at a seam; describes role, not substance. | `sdd-design` Architecture Decisions. | "implementation" (overlap; use adapter only when seam is the topic). |
| Depth | Leverage at the interface: the amount of behaviour a caller exercises per unit of interface they have to learn. | `sdd-design` Architecture Decisions (the "deep vs shallow" check). | "complexity" (depth is leverage, not bulk). |
| Artifact Store Mode | The persistence contract for SDD artifacts: `hybrid` (filesystem + Mnemonic) is the only mode for every phase. | Referencing the persistence layer in any SDD skill. | "store", "DB" (too generic). |
| Topic Key | A stable string used to upsert Mnemonic observations (`mem_save` with same key updates the same row). | Persisting evolving decisions, conventions, or bugfix lineage. | "key" (ambiguous). |
| Step Folder | `docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/` — the isolated unit of execution per step. | Writing or reading per-step files. | "subdir", "module". |
| Acceptance Feature | A Gherkin `.feature` file under `steps/<NN-name>/acceptance.feature`; the executable contract for a step. | `sdd-spec`, `sdd-apply`, `sdd-verify`. | "spec file" (overlap with `plan.md`). |
| Per-step Verification | The `steps/<NN-name>/verification.md` produced by `sdd-verify` with a `PASS`/`PASS WITH WARNINGS`/`FAIL` verdict. | `sdd-verify`, `sdd-archive` gate. | "test report" (broader). |
