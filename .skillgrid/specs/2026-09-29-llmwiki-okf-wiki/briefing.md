# Change: llmwiki-okf-wiki — Dual-Compat OKF v0.2 + Obsidian + llmwiki `.wiki/` Compiler (Pillar 1)

> **STATUS:** `draft` (2026-09-29)
>
> **For agentic workers:** REQUIRED: follow `.agents/skills/_shared/conventions/sdd-structure.md`. This file is WHY + HOW (intent + plan). Spec phase instantiates `tasks.md` + `acceptance.feature` from the Step Blueprint and per-step WHAT below.

**Goal:** Add a `wiki` capability to skillgrid — a pure-Go compiler (`skillgrid wiki compile`) that projects the project's knowledge into a single `.wiki/` directory that is **simultaneously** an **OKF v0.2** knowledge bundle, an **Obsidian** vault, and a **microsoft/llmwiki** workspace. Pillar 1 sources: `.skillgrid/` (read directly, source of truth) + `web_cache` SQLite (read directly) + user-dropped `raw/*.md` (indexed). No new Go dependencies.

**Architecture:** A new `skillgrid-cli/internal/mnemonic/wiki/` package: (1) an **OKF v0.2 + llmwiki emitter** (`Emit(concept)→string`, `Conform(page)→[]error`), (2) a **pillar-1 source adapter** that parses `.skillgrid/` files into `[]Concept`, (3) a **web_cache indexer** that reads fresh `context7|exa|deepwiki|fetch` rows → `Finding` concepts, (4) a **raw indexer** that reads user-dropped `raw/*.md` → `Finding`/`Source` concepts, (5) a **compiler** that assembles + writes `.wiki/` deterministically (content-addressed, no-churn on unchanged sources, `generated.at` preserved). Wired to a `skillgrid wiki compile` cobra command via the existing `service` facade (`.Web()` already exposes `web_cache`).

**Tech stack:** Go (`skillgrid-cli`), new `internal/mnemonic/wiki/` package, `cmd/skillgrid/wiki.go`. Markdown + YAML frontmatter only (already in stack — no new deps). The `skillgrid-vscode` extension and the dashboard `features/wiki/` view are **separate follow-up changes** (this change produces the artifact they read).

**Research:** microsoft/llmwiki (VS Code extension) expects `.wiki/{raw, wiki/, index.md, log.md, AGENTS.md}`; our layout matches → works unmodified. OKF v0.2 (GoogleCloudPlatform/knowledge-catalog `okf/SPEC.md`) defines the frontmatter field set: `type` (only required key), `title`, `description`, `resource`, `tags`, `sources[].{resource,author,last_modified}`, `generated.{by,at}`, `verified[].{by,at}`, `status` (draft|stable|deprecated, absent=stable), `stale_after` (**absolute UTC instant**, not a relative TTL). Producers MAY add extension keys → `source_path` (llmwiki backlink target) is a valid extension. Cached in web_cache id 1.

**Ticket:** none

**Depends on:** none (independent of the in-flight memory-improvements / session-events changes; reads `web_cache` + `.skillgrid/` which already exist)

---

## Goal

`.skillgrid/` is the human-authored source of truth; Mnemonic SQLite is the AI-only derived store; but there is **no human-readable, portable artifact** of the project's knowledge. An agent that leaves the repo (or a fresh clone on another machine) has no portable bundle of "what we decided, what we learned, what we researched." This change compiles that knowledge into `.wiki/` — readable by a human in Obsidian, parseable by an LLM in microsoft/llmwiki, and conformant to OKF v0.2 for any other consumer. The compile is **deterministic** so `.wiki/` is a clean git commit (no churn on no-op recompiles).

## Out of scope / Non-Goals (Pillar 1)

- **`skillgrid-vscode`** custom extension (tree views by skillgrid `type` + `.backlog/` tree) — separate sub-change; it *reads* the `.wiki/` this change produces.
- **Dashboard `features/wiki/`** view — separate sub-change; reads `.wiki/` (existing views stay live-SQLite).
- **MCP `wiki_*` tools** — follow-up; the CLI command is the entry point for Pillar 1.
- **Pillar 2:** Mnemonic `observations` + distilled memory as sources — follow-up change.
- **LLM-inferred cross-links** (INFERRED, 0.85) — follow-up; Pillar 1 emits **explicit (EXTRACTED) edges only** so the MVP is fully offline + deterministic. `--no-llm` is a no-op in Pillar 1.
- **Mutating `web_cache` / `.skillgrid/`** — the compiler is read-only over both; it only writes `.wiki/`.
- **`wiki` subcommands other than `compile`** (`lint`, `graph`, `list`) — `lint` may ship with the MVP (it's the OKF conformance gate); `graph`/`list` are follow-ups.
- **ARCHITECTURE.md** — does not exist yet; the adapter is **existence-gated** (emits an `Architecture` concept only if the file is present).

## Definition of Done

This change is done only when **all** of the following are true:

1. `skillgrid wiki compile` produces `.wiki/{AGENTS.md, raw/ (untouched), wiki/adr|concepts|entities|research, wiki/index.md, wiki/log.md}`.
2. Every emitted page has **valid OKF v0.2 frontmatter**: `type` present; `sources[].resource` present where a source exists; timestamps are absolute UTC ISO 8601; `stale_after` (when present) is an absolute instant = `generated.at + 90d`.
3. **Pillar-1 sources compile:** ADRs from `ASSUMPTIONS.md` (in-force + superseded), Terms from `artifacts/01-business-terms.md` + `02-technical-terms.md`, `State` from `state.yaml`, `Spec` from `specs/*/`, `Spike` from `spikes/*/`, `Constraint` from `### Locked constraints`, `Architecture` from `ARCHITECTURE.md` (existence-gated).
4. **web_cache compiles (hybrid, option C):** fresh `context7|exa|deepwiki|fetch` rows that are **cited** by ≥1 page → `Finding` (status per citation); **uncited** fresh rows → `wiki/research/` (status: draft). No file is written to `raw/` for these.
5. **raw/ compiles:** every user-dropped `raw/**/*.md` (any depth) is indexed to a `Finding`/`Source` page with `source_path: raw/<path>`; non-OKF files (no frontmatter) become bare `Source` pages (filename→title, `verified` absent). `raw/` files are never rewritten by the compiler.
6. **Explicit wikilinks (EXTRACTED):** ADR→ADR (Supersedes/Amends), State→Spec (current_change), finding→spec citations, term→term cross-refs, all emitted as `[[wikilink]]` in bodies and resolvable by Obsidian + llmwiki.
7. **Determinism / no-churn:** two consecutive `wiki compile` runs over unchanged sources produce **byte-identical** `.wiki/` (unchanged `generated.at` preserved, stable sort, content-hash gate). A `git diff` after a no-op recompile is empty.
8. **`wiki lint`** (OKF conformance gate): every emitted page passes `Conform` (type present, `sources[].resource` present, timestamp format, `stale_after` is absolute). Lint failure is non-zero exit.
9. **`go test ./...`** passes; the new `internal/mnemonic/wiki/` package is covered by unit tests (emitter, conformance, each adapter, determinism) + one end-to-end compile test over a fixture `.skillgrid/` + seeded `web_cache`.
