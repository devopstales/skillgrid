# Glossary — Business

Domain, product, workflow, and business terms used across skillgrid changes.

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| Step | A sequenced unit of an SDD change that ships one testable behaviour slice (Step Blueprint → `@step-NN`). | Writing `change.md` Step Blueprint / Per-step WHAT, `tasks.md` sections, `acceptance.feature`. | "phase" (overloaded with sdd phases), "stage". |
| Change | A self-contained SDD unit tracked by a 3-digit NNN number, authored as `change.md` through `archive/`. | Referencing any SDD change in docs or skills. | "ticket", "PR" (broader scope). |
| Artifact | A file persisted by an SDD phase (`research.md`, `change.md`, `tasks.md`, `acceptance.feature`, archive report). | Cross-referencing phase outputs. | "document", "doc" (too generic). |
| Companion Glossary Reference | Deprecated in SDD v3 — do not create `*-glossary-reference.md`; fold terms into `change.md` `## Glossary` and `docs/skillgrid/agents/glossary/`. | Only when reading legacy pre-v3 changes. | Creating new companion files. |
| Clone-Private Identity Binding | One-time project id written into the git common-dir and reused on every later resolve for that clone (including linked worktrees). | Mnemonic project resolution (002); Engram-parity identity. | "project name", path-hash id, remote basename alone. |
| Child Auto-Promote | When a cwd is the parent of exactly one git repo, resolution promotes that child as the project (with a soft warning). | Parent-directory resolve behaviour (002). | Blind directory-hash fallback. |
| Ambiguous Project | Resolution outcome when a cwd parents multiple git repos; callers receive `AvailableProjects` instead of a silent fallback id. | `ErrAmbiguousProject` / `MNEMONIC_PROJECT` recovery (002). | Silent `directory-hash` pick. |
| Cross-Store Recall | Searching every Mnemonic SQLite under the store dir and merging/re-ranking hits (`all_projects`). | `mem_search(all_projects=true)` (002). | Per-store-only search. |
| Observation Lifecycle | Pinning, expiry, duplicate count, and recency columns that affect recall ordering and exclusion. | Lifecycle parity vs Engram (002). | Soft-delete alone; review_after cycle alone. |
| Long-term Memory | A durable compacted memory entry (L0/L1/L2) produced by session/task compaction for future agent recall. | `mnemonic_commit`, self-evolving context database. | "observation", "mem_save" (session-scoped curated notes). |
| Fact Memory | A structured store of atomic facts (importance, scope, soft delete, decay) beside `mem_*` observations — not an observation overlay. | Hermes memory tools (`fact_add`/`fact_search`/`fact_forget`/`fact_decay`); 004-hermes-memory. | "observation", "mem_save", "Long-term Memory" (different stores). |
| Agent Skill | An agent-writable, searchable, sandbox-executable script/prompt stored as FS + `skills` table metadata. | Hermes skill tools (`write_skill`/`search_skills`/`use_skill`/`list_skills`); 004-hermes-memory. | `.agents/skills` SDD skill packs; "skill" alone when ambiguous. |
| Session Handoff | A Cleave-style continuity package: progress, knowledge, and next-prompt files plus a recorded handoff row so a task can resume in a new session. | `session_handoff` / `session_resume`; `.skillgrid/.cleave/`; 006-structured-session-handoff. | "Long-term Memory" (durable recall via `mnemonic_commit`); Engram `mem_session_*` summaries alone. |
| Session Relay | The subsystem that records, resumes, and status-checks Session Handoffs (schema, MCP, `.cleave/` FS, CLI, optional watchdog). | Naming the 006 capability set; cross-session task continuity. | "session" alone; Engram session lifecycle; Tiered Storage. |
