# SQLite as the second-brain store; llm-wiki markdown files rejected

---
status: "accepted"
supersedes:
date: 2026-09-30
---

**Related:** the capability roadmap on top of this store — what makes mnemonic *feel* like a second brain (autonomous capture, `mem_ask` synthesis, knowledge lifecycle, durable sqlite-vec, passive learning), with the claude-os reference mapping — is `08-second-brain-roadmap.md`. This ADR stays scoped to the engine choice; the roadmap is the "what to build next" layer.

## Context and Problem Statement

Skillgrid planned an "llm-wiki" — an LLM-maintained, interlinked markdown wiki (Karpathy pattern) — as the project's second brain. During design it became clear that the data such a wiki would hold (observations, decisions, session summaries, code index) already lives in the mnemonic SQLite store. Adding a markdown wiki layer would create a second source of truth over the same data.

The open question: do we keep the llm-wiki, and is a markdown wiki faster or more token-efficient than SQLite for agent retrieval?

**Layer clarification (2026-09-30, see `.skillgrid/specs/2026-09-30-second-brain-vs-agent-os/findings.md`):** "second brain" is the ROLE (a durable offloaded knowledge store — tool-agnostic), "agent OS" is the RUNTIME that manages an agent's context/memory/scheduling (MemGPT/AIOS framing), and "Claude Code OS" is a product genre that wires the two together with a markdown memory layer. This ADR decides only the **memory ENGINE** for the second-brain role (SQLite over llm-wiki markdown). It does NOT rule on the role itself, and it is orthogonal to the agent-OS runtime: mnemonic is the persistent-memory subsystem such a runtime would drive (a memory/storage manager), not the OS.

## Considered Options

- **llm-wiki (markdown files)**: LLM compiles knowledge once into interlinked pages; index.md as the catalog; query by reading pages. Value is *accumulation* (synthesis compiled at ingest, not re-derived per query), but retrieval is read-whole-page, costs tokens proportional to page size, and grows linearly with the corpus.
- **SQLite + FTS5 (mnemonic's existing store)**: single file, constant retrieval cost (a few hundred tokens for top-k results regardless of corpus size), zero-LLM ops in the hot path, sub-second, BM25 ranking. Weak on paraphrase queries, but skillgrid's queries are mostly literal (task IDs, file paths, error strings, slugs) where lexical matching wins.
- **Hybrid / markdown-as-view**: keep SQLite as the source of truth; export compiled markdown as a derived view only when human-readable documentation becomes a deliverable.

## Decision Outcome

Chosen option: "SQLite as the second-brain store", because the llm-wiki would duplicate data that already exists, and for a single-user local tool SQLite/FTS5 retrieval is cheaper and more predictable than reading compiled markdown pages.

The video that motivated the review (https://www.youtube.com/watch?v=R-5_2nsF_ZM, "How to Build Enterprise Grade Agent Memory (Redis Iris Tutorial)", 2026-09-30) endorses a database over markdown specifically for *shipped multi-user agents* (access control, live data, retrieval at scale, per-user long-term memory). For the personal/single-user case it explicitly calls markdown wikis "ideal to keep things simple and flexible" — it does not claim SQLite beats markdown for the personal case.

### Consequences

- Good, because one source of truth (mnemonic SQLite) — no sync, no divergence between wiki and store.
- Good, because retrieval stays constant-cost and zero-LLM; skillgrid's literal-token queries are exactly where FTS5/BM25 performs best.
- Good, because the decision is reversible by adding a derived markdown view or moving to a server database without changing what is stored.
- Bad, because there is no human-browsable compiled wiki until/unless we export one; cross-source synthesis is not pre-compiled and happens at query time.
- Bad, because paraphrase-heavy queries (~38% hit@3 lexical vs ~50% vector) remain the weak spot; the escape hatch is embeddings (sqlite-vec) on the same store, not a wiki.

Revisit triggers: (1) the second brain must serve **multiple agents/sessions concurrently with access control** (the agent-OS *memory-manager* concern, not a markdown concern) — then per-user DB-backed memory (Redis Iris or equivalent) per the video's argument; note the store is already DB-backed, so this is a concurrency/authorization change, not a storage swap; (2) human-readable compiled docs become a deliverable — then export a markdown *view* over SQLite, never a second source of truth; (3) queries become paraphrase-heavy — then add sqlite-vec embeddings to the existing store.
