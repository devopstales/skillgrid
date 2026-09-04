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
| Hybrid Search | Multi-signal code retrieval that fuses identifier-aware FTS, deterministic similarity signals, and optional dense embeddings. | Code-intelligence intents/plans (e.g. 005); ranking and MCP search tools. | Memory-only `semantic_search` (003); plain `code_search` chunk FTS. |
| Symbol | A code unit keyed by qualified name (function, method, type, etc.) with path, kind, and span metadata. | Graph index, orientation tools, hybrid ranking units. | "chunk" (line window); memory observation. |
| Edge | A typed, directed relation between symbols with a confidence label and optional properties. | Call graph, dependents, implementors, hierarchy. | Undirected "link"; memory association. |
| Extractor | A per-language adapter that turns a source file into symbols and edges (tree-sitter primary, regex fallback). | Index pipeline; language Tier-1 rollout. | "parser" alone (must also emit graph). |
| Confidence Label | Trust class on an edge: `EXTRACTED`, `INFERRED`, or `AMBIGUOUS`. | Graph tools and resolver transparency. | Free-form score without a label. |
| Index Freshness | Per-path (and result-level) metadata stating whether indexed bytes still match the working tree. | Staleness banners, `index_status`, reindex UX. | Binary empty-index "stale" from chunk `code_status` alone. |
| Identifier-Aware FTS | FTS5 tokenization that splits camelCase and snake_case so symbol names match partial identifiers. | Symbol search and hybrid keyword channel. | Trigram-only `chunks_fts` without identifier split. |
| Tiered Storage | Progressive L0 (abstract) / L1 (overview) / L2 (full details) content held as filesystem sidecars with SQL path columns. | Mnemonic recall design, `migrate --tier`, content-write hooks. | "summary files", "compressed memory" (vague). |
| Retrieval Trail | A persisted record of a retrieval: query, directories traversed, files read, and final result path. | Debugging why an agent missed context; `skillgrid trail`. | "search log", "audit log" (broader). |
| Semantic Search | Ranked recall over tiered content using embeddings (with title/L0 fallback when vectors are absent). | `semantic_search` MCP tool; directory-first then in-directory re-rank. | "FTS search", "mem_search" (observation FTS only). |
| Team Task | A multi-agent work unit with SQL metadata (status, ownership, paths) and markdown brief/output on the filesystem data plane. | Hybrid teams MCP/HTTP; spawn/pull/submit/done flows (001). | Backlog.md tracker "task"; SDD Step; Session Handoff. |
| Team Member | An agent role row on a team (lead, developer, reviewer, etc.) that can be assigned tasks or submit reviews. | `team_members` table; pull/claim and review authorship. | Human user; generic "agent" without team binding. |
| Agent Review | A peer review of a Team Task (spec compliance or code quality) with a passed flag and comments markdown path. | `agent_submit_review`; `reviews` table under teams schema. | Memory observation `mem_review` / `/memory/reviews` lifecycle. |
| Inbox | Unread agent-to-agent messages for a Team Member, with subject in SQL and body markdown on disk. | Messages table / future inbox MCP; HTTP message CRUD. | Email inbox; MCP tool list; observation timeline. |
