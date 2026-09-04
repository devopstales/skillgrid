# Glossary — Business

Domain, product, workflow, and business terms used across skillgrid changes.

| Term | Definition | Use When | Avoid |
| --- | --- | --- | --- |
| Step | A sequenced phase of an SDD change that ships one testable unit of behaviour. | Writing `intent.md` Step Blueprint, `plan.md` Impacted Files Map, `tasks.md` per-step punch-list. | "phase" (overloaded with sdd phases), "stage". |
| Change | A self-contained SDD unit tracked by a 3-digit NNN number, from `intent.md` to `archive/`. | Referencing any SDD change in docs or skills. | "ticket", "PR" (broader scope). |
| Artifact | A file persisted by an SDD phase (`intent.md`, `plan.md`, `research.md`, `tasks.md`, `acceptance.feature`, `verification.md`, `archive-report`). | Cross-referencing phase outputs. | "document", "doc" (too generic). |
| Companion Glossary Reference | A `<artifact>-glossary-reference.md` file living next to an artifact, listing the glossary terms the artifact uses. | Always-on for `intent.md` and `plan.md`. | "glossary doc" (ambiguous). |
| Long-term Memory | A durable compacted memory entry (L0/L1/L2) produced by session/task compaction for future agent recall. | `mnemonic_commit`, self-evolving context database. | "observation", "mem_save" (session-scoped curated notes). |
| Fact Memory | A structured store of atomic facts (importance, scope, soft delete, decay) beside `mem_*` observations — not an observation overlay. | Hermes memory tools (`fact_add`/`fact_search`/`fact_forget`/`fact_decay`); 004-hermes-memory. | "observation", "mem_save", "Long-term Memory" (different stores). |
| Agent Skill | An agent-writable, searchable, sandbox-executable script/prompt stored as FS + `skills` table metadata. | Hermes skill tools (`write_skill`/`search_skills`/`use_skill`/`list_skills`); 004-hermes-memory. | `.agents/skills` SDD skill packs; "skill" alone when ambiguous. |
| Session Handoff | A Cleave-style continuity package: progress, knowledge, and next-prompt files plus a recorded handoff row so a task can resume in a new session. | `session_handoff` / `session_resume`; `.skillgrid/.cleave/`; 006-structured-session-handoff. | "Long-term Memory" (durable recall via `mnemonic_commit`); Engram `mem_session_*` summaries alone. |
| Session Relay | The subsystem that records, resumes, and status-checks Session Handoffs (schema, MCP, `.cleave/` FS, CLI, optional watchdog). | Naming the 006 capability set; cross-session task continuity. | "session" alone; Engram session lifecycle; Tiered Storage. |
