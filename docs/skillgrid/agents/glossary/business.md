# Glossary — Business

Domain, product, workflow, and business terms used across skillgrid changes.

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| Step | A sequenced phase of an SDD change that ships one testable unit of behaviour. | Writing `intent.md` Step Blueprint, `plan.md` Impacted Files Map, `tasks.md` per-step punch-list. | "phase" (overloaded with sdd phases), "stage". |
| Change | A self-contained SDD unit tracked by a 3-digit NNN number, from `intent.md` to `archive/`. | Referencing any SDD change in docs or skills. | "ticket", "PR" (broader scope). |
| Artifact | A file persisted by an SDD phase (`intent.md`, `plan.md`, `research.md`, `tasks.md`, `acceptance.feature`, `verification.md`, `archive-report`). | Cross-referencing phase outputs. | "document", "doc" (too generic). |
| Companion Glossary Reference | A `<artifact>-glossary-reference.md` file living next to an artifact, listing the glossary terms the artifact uses. | Always-on for `intent.md` and `plan.md`. | "glossary doc" (ambiguous). |
