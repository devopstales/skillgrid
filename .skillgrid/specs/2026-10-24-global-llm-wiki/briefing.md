---
type: Spec
title: "2026-10-24-global-llm-wiki — briefing"
archive: "2026-10-24-global-llm-wiki"
status: draft

---

# Change: Global Operational Memory + LLM-Wiki Pipeline

> **STATUS:** `ready-for-tasks` — all open questions resolved (Q1–Q8).
>
> **REVISED 2026-10-24 (v4):** All 8 open questions resolved. Q2: hybrid score-based auto-approve. Q4: OpenCode plugin hook + MCP resource fallback. Q5: hybrid explicit + passive distill. Q6: credential redaction (deterministic regex). Q7: ADR-0049 + ADR-0050 (amends ADR-0048 layout). Q8: agent judgment for scope.

## Goal

Two changes:

1. **Global Operational Memory (SQLite):** A cross-project, cross-session, cross-agent fact store in `~/.skillgrid/mnemonic/global.sqlite`. Hermes-style: bounded (char limit), importance-ranked (AKL), with prefetch injection at session start. This is the "brain" — small, curated, always-available.

2. **LLM-Wiki Pipeline (over `knowledge/` Obsidian vaults):** The ingest/review/lint/index pipeline that makes the `knowledge/` vaults living LLM-maintained wikis. Raw sources → LLM extraction → HITL review → wiki pages → FTS index. Works at both global and project scope. This is the "library" — large, on-demand, distillable into the brain.

## Problem

skillgrid has two gaps:

**Gap 1 — No global operational memory.** A fact about "user prefers concise responses" stored in project A's SQLite is invisible in project B. There is no:
- Cross-project, cross-agent fact store
- Bounded, curated, always-available memory (Hermes MEMORY.md pattern)
- System-prompt injection of memory at session start

**Gap 2 — The wiki is static, not living, and entangled with config.** The `.skillgrid/` bundle (ADR-0048) has a **role collision**: the bundle root is simultaneously the config/state root (`config.yaml`, `state.yaml`, `sdd/`, `specs/`) AND the wiki (`decisions/`, `glossary/`, `reference/`, etc.). The wiki has:
- No LLM compilation pipeline (raw source → extracted concepts → interlinked pages)
- No HITL trust layer (silver tier between raw and wiki)
- No FTS index over the wiki content (the SQLite store indexes observations, not the markdown)
- No maintenance/lint loop beyond `okf-lint.mjs` (orphan pages, broken links, contradictions)
- No distillation path (wiki insight → operational memory fact)
- No global scope (cross-project knowledge has no home)
- No graph view (the wiki-links exist but there's no visual knowledge graph)

The Hermes Agent ecosystem solved the first gap with bounded memory files + pluggable providers. The Wanderloots playlist solved the second gap with the LLM-Wiki pattern (Karpathy): "knowledge is compiled once and kept current, not re-derived on every query." Obsidian provides the vault + graph + clipper that makes the wiki *usable*.

## Research Sources

| Source | URL | Key takeaway |
|---|---|---|
| Hermes Agent memory docs (official) + Timi's "as it exists in code" article | `hermes-agent.nousresearch.com/docs/user-guide/features/memory` + `medium.com/@xpf6677/hermes-agent-memory-system` | Two targets (MEMORY.md 2200 / USER.md 1375), frozen snapshot injection with `§` separators + usage headers, `operations` batch with atomic limit check, substring matching for patch/remove, duplicate rejection, security scanning, `write_approval` staging (`/memory pending/approve/reject`), **background self-improvement review** (post-turn fork: cheaper model option, `max_input_tokens` cap, `defer` for local GPU, `extra_tools` whitelist, coalescing queue), **learning journey** (`/journey` timeline of learned skills+memory, prune/correct), "memory needs session boundaries" (frozen snapshot = fresh entries invisible until next session), "model claimed save without tool call" troubleshooting (weak models), two-store troubleshooting (profile scoping) |
| hermesatlas.com memory guidebook | `hermesatlas.com/guide/memory/` | 3-layer stack: native (L1) + MemoryProvider (L2, 8 official) + community plugins (L3). Providers: Mnemosyne (SQLite, BEAM tiers, local), Hindsight (server, LLM, knowledge graph, reflect), Honcho, Mem0, ByteRover, OpenViking, RetainDB, Supermemory. One provider at a time. |
| glukhov.org Hermes memory deep-dive | `glukhov.org/ai-systems/hermes/hermes-agent-memory-system/` | "Memory isn't retrieved — it's always there." Prefetch/sync runtime flow. Frozen snapshot for prefix caching. Character limits as a feature. Internal memory (brain) vs external knowledge bases (library). Distillation pattern. |
| Wanderloots: Improved AI Memory (Mnemosyne & Hindsight) | `youtu.be/R1TNGOZAOZs` | Mnemosyne = memory *layer* (SQLite, zero-dep, BEAM: working/episodic/semantic/scratchpad). Hindsight = memory *engine* (server, LLM, SPO triples, reflect, constellation view). "Global" scope is the expected default, not an afterthought. |
| Wanderloots: Why LLM Wiki? | `youtu.be/n4EVksU_EOs` | Knowledge graph = nodes + edges + triples. RAG vs graph RAG. Karpathy's LLM-Wiki: "knowledge is compiled once and kept current, not re-derived on every query." 3 layers: raw sources → wiki → maintenance. Human vault vs agentic vault. |
| Wanderloots: How To Build LLM Wiki In Obsidian | `youtu.be/QbjAQFJJyt0` | 3 layers: raw (bronze) → wiki (gold) → schema (rules). AGENTS.md as constitution. Templates + frontmatter schema. Python scripts for deterministic tooling (index, catalog, search, lint). Git for version control. Obsidian CLI + skills. Local LLM via Ollama with HITL review folder. Firewall between vaults. |
| Wanderloots: Multi-Agent Memory (Hindsight) | `youtu.be/UVEjThz8DSo` | One Hindsight memory bank shared across Hermes + Codex + any agent. "Global user" bank for solo dev with multiple agents. Memory banks + tags for organization. Cross-agent: Codex stores → Hermes recalls immediately. |
| Wanderloots: Obsidian + Agentic AI | `youtu.be/znj-WpMj1dI` | 3 access levels: private (agent can't see), readonly, read+write. Obsidian as "library of world knowledge." Separate from agentic memory. Skill-based integration (SKILL.md teaches agent how to work with notes). |
| Wanderloots: Agentic Librarian (HITL) — full transcript | `youtu.be/_bieksxg6oY` | 3 paths: (1) built-in skill (raw→wiki auto), (2) review companion skill (silver HITL wrapper), (3) Agentic Librarian kit (deterministic control plane). **Key patterns**: hash verification on raw sources (`processed: false` → `true`), contradiction detection → `comparisons/` page + `contested: true`, two-phase approval (approve → apply), proposals with revision number + target + outline + open questions + human feedback section, routing (`routine: true` → batch-approve), git checkpoint per run, "the AI interprets the knowledge; the tools control admission," settings as frontmatter, dashboard with source coverage %, dual review interface (Obsidian Bases + decision studio web UI) |
| Wanderloots: Always On Agentic AI (Abacus) | `youtu.be/ZL-8IhjQDvE` | Cloud-hosted agent infrastructure. Always-on toggle. Route LLM for model selection. Context for session continuity. |
| skillgrid ADR-0012 | `.skillgrid/decisions/04-adr-0012-sqlite-as-second-brain.md` | SQLite over llm-wiki markdown as the *engine*. Constant retrieval cost, zero-LLM hot path, single SoT. Reversible by adding a derived markdown view. |
| skillgrid ADR-0048 | `.skillgrid/decisions/04-adr-0048-hybrid-sot.md` | SQLite = SoT for observations + code intelligence. `.skillgrid/` Markdown bundle = SoT for knowledge base (decisions/invariants/glossary/reference/workflows). |
| skillgrid 004-hermes-memory (shipped) | `.skillgrid/xarchive/2026-09-04-hermes-memory/` | Per-project `facts` table + FTS5 + AKL importance + sandboxed skills. `fact_*` MCP tools + CLI. All 6 tickets done, 12/12 acceptance scenarios. |
| Future AGI: 9 Open-Source Self-Improving AI Agent Frameworks | `futureagi.com/blog/self-improving-ai-agents-open-source-frameworks/` | Four self-improvement mechanisms: (1) prompt optimization (DSPy, TextGrad, Trace), (2) **memory/experience** (Reflexion, Voyager — skill library), (3) weight update (SEAL), (4) architecture search (ADAS, Darwin Gödel Machine, EvoAgentX). skillgrid's memory system is **mechanism #2**. Key patterns: **Reflexion** (verbal self-critique → episodic buffer → read back on retry; skillgrid's distill hook is the session-end equivalent), **Voyager** (skill library — memory can be executable, not just text; skillgrid's `skills` table is this), **Darwin Gödel Machine** (acceptance rule: "no change without a measured win" — benchmark before, benchmark after, keep only on improvement, archive what you replaced), **frozen benchmark** (every self-modifying loop needs a fixed eval set it cannot edit; without it "self-improving" is unfalsifiable) |
| Cognee: Building Self-Improving Skills for Agents | `cognee.ai/building-self-improving-skills-for-agents` | **Closed-loop skill improvement**: observe → inspect → amend → evaluate. skillgrid already has `skill_usage` (per-execution log: skill_id, session_id, timestamp, stdout, stderr). What's missing: **outcome tracking** (success/failure on `skill_usage`), **inspection** (pattern detection: recurring failures for a skill), **amend** (LLM proposes a patch to SKILL.md grounded in evidence from real runs), **evaluate** (did the amendment actually improve outcomes? rollback if not). The `.amendify()` pattern: skills stop being static prompt files and become evolving components. Every change tracked with rationale + results; original instructions never lost (git) |

## Layout

### Global (`~/.skillgrid/`)

```
~/.skillgrid/
├── config.yaml                    # global config
├── config.d/                      # global config includes
├── sdd/                           # global execution ledger (gitignored)
├── knowledge/                     # GLOBAL wiki = Obsidian vault
│   ├── .obsidian/                 #   vault config (gitignored: workspace, graph)
│   │   └── (committed: app.json, core-plugins.json, appearance.json)
│   ├── index.md                   #   catalog (LLM-maintained, OKF)
│   ├── reverse-index.md           #   anchor → concept (derived from FTS)
│   ├── okf-metrics.jsonl          #   lint audit trail
│   ├── log.md                     #   ingest/approval/changelog
│   ├── raw/                       #   BRONZE: captured sources (untouched)
│   ├── proposed/                  #   SILVER: HITL review queue
│   ├── concepts/                  #   GOLD: cross-project concepts
│   ├── entities/                  #   GOLD: people, companies, tools
│   └── topics/                    #   GOLD: cross-project topics
└── mnemonic/
    ├── global.sqlite              #   Tier 2: operational memory
    │                              #   + Tier 1: global wiki FTS index
    └── <project-id>.sqlite        #   Tier 3: sessions, code index
                                   #   + Tier 1: project wiki FTS index
```

### Project (`<project>/.skillgrid/`)

```
<project>/.skillgrid/
├── config.yaml                    # project config
├── config.d/                      # project config includes
├── sdd/                           # project execution ledger (gitignored)
├── knowledge/                     # PROJECT wiki = Obsidian vault
│   ├── .obsidian/                 #   vault config (gitignored: workspace, graph)
│   ├── index.md                   #   catalog (LLM-maintained, OKF)
│   ├── reverse-index.md           #   anchor → concept (derived from FTS)
│   ├── okf-metrics.jsonl          #   lint audit trail
│   ├── log.md                     #   ingest/approval/changelog
│   ├── raw/                       #   BRONZE: captured sources
│   ├── proposed/                  #   SILVER: HITL review queue
│   ├── decisions/                 #   GOLD: ADRs (OKF)
│   ├── invariants/                #   GOLD: locked constraints
│   ├── glossary/                  #   GOLD: terms
│   ├── reference/                 #   GOLD: PRD, research, roadmaps
│   ├── workflows/                 #   GOLD: workflow docs
│   ├── concepts/                  #   GOLD: project concepts
│   ├── entities/                  #   GOLD: project entities
│   └── topics/                    #   GOLD: project topics
├── specs/                         # active spec folders (SDD)
├── xarchive/                      # closed change folders (immutable)
├── prototypes/                    # SDD prototypes
└── state.yaml                     # machine resume-pointer
```

### What moves (from current layout)

| Current | New | Why |
|---|---|---|
| `.skillgrid/decisions/` | `.skillgrid/knowledge/decisions/` | Wiki content, not config |
| `.skillgrid/glossary/` | `.skillgrid/knowledge/glossary/` | Same |
| `.skillgrid/reference/` | `.skillgrid/knowledge/reference/` | Same |
| `.skillgrid/invariants/` | `.skillgrid/knowledge/invariants/` | Same |
| `.skillgrid/workflows/` | `.skillgrid/knowledge/workflows/` | Same |
| `.skillgrid/index.md` | `.skillgrid/knowledge/index.md` | Catalog for the wiki |
| `.skillgrid/reverse-index.md` | `.skillgrid/knowledge/reverse-index.md` | Same |
| `.skillgrid/okf-metrics.jsonl` | `.skillgrid/knowledge/okf-metrics.jsonl` | Same |
| `.skillgrid/log.md` | `.skillgrid/knowledge/log.md` | Same |
| `.skillgrid/ASSUMPTIONS.md` | `.skillgrid/knowledge/ASSUMPTIONS.md` | ADR index — wiki content |
| `.skillgrid/ARCHITECTURE.md` | `.skillgrid/knowledge/ARCHITECTURE.md` | Same |
| `.skillgrid/config.yaml` | stays `.skillgrid/config.yaml` | Config |
| `.skillgrid/config.d/` | stays `.skillgrid/config.d/` | Config |
| `.skillgrid/sdd/` | stays `.skillgrid/sdd/` | Execution state |
| `.skillgrid/specs/` | stays `.skillgrid/specs/` | SDD |
| `.skillgrid/xarchive/` | stays `.skillgrid/xarchive/` | Closed changes |
| `.skillgrid/state.yaml` | stays `.skillgrid/state.yaml` | State |
| `.skillgrid/prototypes/` | stays `.skillgrid/prototypes/` | SDD artifacts |

**Rule:** if it's curated knowledge (human or LLM-written, git-tracked, cited by agents), it's in `knowledge/`. If it's config, state, or execution ledger, it stays at the root.

## Knowledge Architecture (3 tiers)

```
┌─────────────────────────────────────────────────────────────────────┐
│  TIER 1: WORLD KNOWLEDGE (LLM Wiki) — GLOBAL + PER-PROJECT          │
│  "Knowledge you want preserved and organized"                       │
│                                                                     │
│  Storage:   Markdown in knowledge/ Obsidian vaults                  │
│             Global:   ~/.skillgrid/knowledge/                       │
│             Project:  <project>/.skillgrid/knowledge/               │
│  Structure: OKF v0.2 frontmatter + [[wiki-links]] (Obsidian)        │
│  Compilation: LLM reads raw/ → extracts → interlinked pages         │
│  Access:    On-demand (agent searches/reads when needed)            │
│  Scope:     Global vault (cross-project) + project vault (scoped)   │
│  Trust:     Bronze (raw/) → Silver (proposed/) → Gold (zones)      │
│  Index:     SQLite FTS in the store that owns the vault             │
│             Global wiki → global.sqlite                             │
│             Project wiki → <project-id>.sqlite                      │
│  Graph:     Obsidian graph view ([[wiki-links]] visualized)         │
├─────────────────────────────────────────────────────────────────────┤
│  TIER 2: OPERATIONAL MEMORY (Global Memory) — CROSS-PROJECT         │
│  "Facts about how you like to work, what you're working on"         │
│                                                                     │
│  Storage:   SQLite (~/.skillgrid/mnemonic/global.sqlite)            │
│             + FTS5 + AKL importance                                 │
│  Structure: Flat facts, importance-ranked, decay-managed            │
│  Compilation: Agent self-curates (add/patch/forget/decay)           │
│  Access:    Bounded prefetch injection (frozen snapshot)            │
│  Scope:     Global — cross-project, cross-session, cross-agent      │
│  Trust:     Agent-managed, char-limited, dedup, security scan       │
├─────────────────────────────────────────────────────────────────────┤
│  TIER 3: SESSION ARCHIVE — PER-PROJECT                              │
│  "What we discussed last week" — historical lookups                 │
│                                                                     │
│  Storage:   Per-project SQLite (session_events + observations)      │
│  Structure: Raw session data, FTS5 indexed                          │
│  Access:    On-demand (session_search / fact_search)                │
│  Scope:     Per-project (optionally global)                         │
│  Trust:     Append-only, not curated                                │
└─────────────────────────────────────────────────────────────────────┘
```

**Where skillgrid is today:**
- Tier 1: `.skillgrid/` OKF bundle at the bundle root (ADR-0048) — markdown, human-curated, **no LLM pipeline**, no HITL, no FTS index, no graph, no global scope, entangled with config
- Tier 2: **Missing** — this change builds it
- Tier 3: Per-project `session_events` + `observations` — exists

**Key insight (2026-10-24 v3):** The wiki moves into `knowledge/` to resolve the role collision. The bundle root is config + state; `knowledge/` is the wiki. The wiki exists at **both** global and project scope. Each is an Obsidian vault — plain markdown, git-tracked, graph-viewable, clipper-able. The LLM-Wiki pipeline merges with OKF: one maintenance model, one lint.

## What "Obsidian vault" provides

| Capability | How it helps |
|---|---|
| **Graph view** | Visual knowledge graph of `[[wiki-links]]` — the "constellation view" from Hindsight, for free |
| **Wiki-links** | `[[concept-name]]` syntax is what the LLM-Wiki uses. Obsidian makes them clickable, resolvable, graph-able |
| **Web Clipper** | Clip articles/transcripts/PDFs directly into `knowledge/raw/` — the bronze layer |
| **Frontmatter** | Obsidian's properties UI reads OKF v0.2 frontmatter. Human-editable metadata |
| **Bases** (Obsidian 1.9+) | Database-like views over frontmatter: "show me all `proposed` pages" |
| **Local-first** | Plain markdown on disk. Git-tracked. No cloud. Consistent with skillgrid's local-first invariant |
| **Agent skill** | The existing `obsidian` skill teaches the agent to read/search/create/link notes |

### `.obsidian/` git strategy

```gitignore
# knowledge/.gitignore (committed)
.obsidian/workspace.json
.obsidian/workspace-mobile.json
.obsidian/graph.json
.obsidian/cache/
.obsidian/plugins/
```

Committed: `.obsidian/app.json`, `.obsidian/appearance.json`, `.obsidian/core-plugins.json` (vault-level config: theme, enabled core plugins). Gitignored: per-user workspace state, graph view state, plugin installs.

## OKF + LLM-Wiki: One Maintenance Model

The LLM-Wiki pipeline **merges with OKF v0.2**. OKF is the frontmatter spec the pipeline enforces; the pipeline is the engine that maintains the catalog, reverse-index, and lint.

| Concern | OKF v0.2 (today) | Merged (this change) |
|---|---|---|
| **Catalog** (`index.md`) | Human-curated, linted | LLM-maintained, rebuilt on ingest; lint validates |
| **Backlinks** (`reverse-index.md`) | From `verification.anchors`, hand-synced | Derived from FTS index (`wiki_links` table); auto-rebuilt |
| **Changelog** (`log.md`) | Human-authored entries | LLM-generated ingest log + HITL approval log + human entries |
| **Lint** (`okf-lint.mjs`) | Frontmatter, index completeness, wikilink syntax | Extended: orphan pages, broken `[[links]]`, contradictions, frontmatter, index completeness |
| **Trust** | All gold (human-curated) | Bronze (raw/) → Silver (proposed/) → Gold (zones) |

`okf-lint.mjs` is extended into `wiki lint` (step B5). The OKF v0.2 contract gains: "the catalog and reverse-index may be LLM-maintained, not just human-curated; the lint validates them either way."

## Change A: Global Operational Memory (SQLite)

Hermes-style global memory. SQLite is the right engine:
- Access pattern is **bounded prefetch** (top-k by importance → frozen block in prompt)
- Needs **AKL decay + purge** (SQL operations)
- Needs **FTS5 search** for on-demand recall
- Needs **char limit enforcement** (SQL aggregate)
- Cross-agent access via MCP

### Design (from Hermes official docs + Timi's "as it exists in code" article)

**Two fact targets** (Hermes MEMORY.md / USER.md pattern → skillgrid `category` column):
- **`memory`** — agent's operational notes: environment facts, conventions, lessons learned, completed work
- **`user`** — user profile: preferences, communication style, expectations, pet peeves

Both live in the same `global.sqlite` `facts` table, distinguished by `category TEXT NOT NULL DEFAULT 'memory'`. The prefetch injection renders them as two labeled blocks (like Hermes' `[MEMORY]` / `[USER PROFILE]` headers with usage percentages).

**Batch operations with atomic limit check** (Hermes `operations` batch pattern):
The `fact_add` / `fact_patch` / `fact_remove` MCP tools accept an `operations` array. The char limit is checked **only on the batch's final result**, not per-operation. This lets the agent free space (remove/patch stale entries) AND add a new entry in one atomic call — the Hermes pattern for "memory is full, make room and retry."

```json
{
  "target": "memory",
  "scope": "global",
  "operations": [
    {"action": "remove", "old_text": "stale entry substring"},
    {"action": "add", "content": "new fact that needs the freed space"}
  ]
}
```

**Injection format** (Hermes frozen snapshot pattern):
```
══════════════════════════════════════════════
GLOBAL MEMORY [67% — 1,474/2,200 chars]
══════════════════════════════════════════════
User's project is a Go service using chi + SQLite
§
This machine runs macOS 14, has Docker and Homebrew
§
...

══════════════════════════════════════════════
USER PROFILE [42% — 575/1,375 chars]
══════════════════════════════════════════════
Prefers concise responses, dislikes verbose explanations
§
...
```
Entries separated by `§`. Header shows usage % + char counts. Frozen at session start (prefix cache).

**Memory needs session boundaries** (Hermes insight): The frozen snapshot means entries saved during a session are invisible to the running session. They appear in the *next* session. The agent should be told this in the system prompt guidance: "Facts you save now will be visible to future sessions, not this one."

**Background self-improvement review** (Hermes post-turn fork pattern → skillgrid distill hook):
After a session, the 013 distill hook runs an LLM pass (ADR-0023) over the conversation. Hermes enhancements to adopt:
- **Cheaper model option**: `mnemonic.distill.model: "auto"` (default = main model, warm cache) or a specific cheaper model (digest replay instead of full transcript)
- **Token cap**: `mnemonic.distill.max_input_tokens` — caps the sum of replayed input tokens for one distill run
- **Disable option**: `mnemonic.distill.enabled: false` — skip automatic post-session distill
- **Coalescing**: if multiple distill triggers fire for the same session, the latest snapshot wins (the distill replays the whole conversation, so nothing is lost)
- **`extra_tools` whitelist**: the distill fork can use `fact_add`/`fact_remove` + read-only tools. Narrow whitelist, no arbitrary tools.

**Write approval** (Hermes `write_approval` pattern → skillglobal gate):
Config: `mnemonic.global_memory.write_approval: false` (default) | `true`. When `true`, `fact_add`/`fact_patch`/`fact_remove` calls from the distill hook (passive) are **staged** — they don't write to the store until approved. The agent (explicit) writes freely by default. CLI: `skillgrid memory global pending` / `approve <id>|all` / `reject <id>|all`. MCP tools: `fact_pending_list`, `fact_pending_approve`, `fact_pending_reject`.

**What to save vs skip** (Hermes guidance, goes into the distill prompt):
- Save: user preferences, environment facts, corrections, conventions, completed work, explicit "remember this"
- Skip: trivial/obvious, easily re-discoverable, raw data dumps, session-specific ephemera, facts already in AGENTS.md

### Steps

| # | Step | Deliverable | Key files |
|---|---|---|---|
| A1 | Global store bootstrap | `Service.OpenGlobal()` → `__global__` pseudo-project → `global.sqlite`; reuses 011 schema + adds `category` column; `ListProjects`/`RunTTLExpiry` skip it | `service.go`, `store.go`, migration 072 |
| A2 | Scope + category params on fact tools | `fact_*` accept `scope: "project" \| "global"` + `target: "memory" \| "user"`; routing; `session_events` trail gets scope | `tools_facts.go`, `facts.go` |
| A3 | Fused search | `scope: "all"` RRF-fuses both stores; labels each hit with scope + category | `tools_facts.go`, `facts.go` |
| A4 | Batch operations + bounded curation | `operations` array on fact tools; char limit (2200 memory / 1375 user) checked on batch final result; error-on-overflow with `current_entries` + `usage`; dedup; substring matching for `patch`/`remove` (`old_text` param) | `facts.go`, `config.d/mnemonic.yaml` |
| A5 | Prefetch injection | **OpenCode plugin hook** (Q4): reads global facts by category at session start, formats two labeled blocks (MEMORY + USER PROFILE) with `§` separators + usage headers, injects frozen. **MCP resource fallback**: `memory/global` resource. **CLI**: `skillgrid memory global prefetch`. Config: `mnemonic.global_memory.prefetch: true` | `plugins/opencode/`, `mcp/server.go` |
| A6 | Background distill (extends 013) | Post-session LLM pass (ADR-0023) over conversation → `fact_add scope=global`. Config: `mnemonic.distill.enabled/model/max_input_tokens`. Coalescing: latest snapshot wins. `extra_tools` whitelist. **Write approval gate** for passive writes: `write_approval: true` → staged, pending list, approve/reject | `facts.go`, `distill.go`, `tools_facts.go` |
| A7 | CLI + tests | `--scope global --target memory\|user`; `skillgrid memory global pending/approve/reject`; threat-matrix RED for isolation, overflow, fused, injection shape, batch atomicity | `memory.go`, `facts_test.go` |
| A8 | Skill improvement loop (cognee pattern) | **Observe**: extend `skill_usage` with `outcome` (success/failure/error) + `feedback` (optional user/agent note). **Inspect**: `skillgrid skill inspect <name>` — aggregates skill_usage rows, detects recurring failure patterns (N failures in M runs, common stderr patterns), returns an evidence summary. **Amend**: `skillgrid skill amend <name>` — LLM (ADR-0023) reads the evidence summary + current SKILL.md, proposes a patch (tighten triggers, add edge case, fix tool call signature). Writes to `SKILL.md.amended` (not in-place). **Evaluate**: `skillgrid skill evaluate <name>` — compares outcome rate before vs after the amendment (using skill_usage timestamps). If improvement is measurable → `git mv SKILL.md.amended SKILL.md` + commit. If regression → discard the amendment, keep original. Config: `mnemonic.skills.auto_amend: false` (default — human reviews the proposed patch) | `skills.go`, `tools_skills.go`, migration 073 |

**~1100-1500 lines. ADR-0049.**

## Change B: LLM-Wiki Pipeline (over `knowledge/` Obsidian vaults)

The `knowledge/` vault is the wiki. This change adds the **pipeline** that makes it living: ingest, HITL review, lint, FTS index, distillation. Works at both global and project scope. Markdown is the SoT; SQLite is a derived index for fast search.

### Architecture

```
RAW (bronze)              WIKI (gold)               INDEX (SQLite)
knowledge/raw/            knowledge/ vault          owning store
┌──────────────┐     ┌──────────────────────┐     ┌──────────────┐
│ sources/     │     │ decisions/           │     │ wiki_pages   │
│ articles/    │ CODE│ glossary/            │ FTS │ wiki_links   │
│ transcripts/ │+LLM │ reference/           │───► │ wiki_fts     │
│ (hash,      │────► │ concepts/            │     │ (catalog)    │
│  processed)  │     │ entities/            │     └──────────────┘
└──────────────┘     │ topics/              │           │
                     │ comparisons/         │           │
                     │ (all with OKF        │           │
                     │  frontmatter +       │           │
                     │  [[wiki-links]])     │           │
                     └──────────────────────┘           │
PROPOSED (silver)                                      │
knowledge/proposed/                                    │
┌──────────────────────────┐                            │
│ revision: N              │   Phase 1: approve         │
│ status: needs_review     │──► (record decision)       │
│ target: concepts/...     │                            │
│ confidence: 0.92         │   Phase 2: apply           │
│ routine: true            │──► move to gold + reindex  │
│ Human feedback: ...      │    + git commit            │
└──────────────────────────┘

At GLOBAL scope:   ~/.skillgrid/knowledge/    → global.sqlite
At PROJECT scope:  <project>/.skillgrid/
                   knowledge/                 → <project-id>.sqlite
```

**Deterministic control plane** (from the Agentic Librarian pattern):
```
┌─────────────────────────────────────────────────────────────────┐
│  LIBRARIAN RUN (skillgrid wiki ingest / review apply)           │
│                                                                 │
│  1. PRE-FLIGHT (code)        scan raw/ for unprocessed or       │
│                              changed-hash files                 │
│  2. REDACT (code)            credential regex scan (Q6)         │
│  3. EXTRACT (LLM)            semantic extraction, [[links]],    │
│                              confidence, open questions         │
│  4. CONTRADICTION (LLM)      check claims vs gold pages;        │
│                              conflict → comparisons/ +          │
│                              contested: true                    │
│  5. VALIDATE (code)          frontmatter, link targets,         │
│                              no orphans                         │
│  6. ROUTE (code)             score ≥ threshold + routine →      │
│                              auto-approve; else → proposed/     │
│  7. COMMIT (code)            git commit the vault               │
│                                                                 │
│  "The AI interprets the knowledge; the tools control admission" │
└─────────────────────────────────────────────────────────────────┘
```

### Steps

| # | Step | Deliverable |
|---|---|---|
| B1 | Knowledge zones + Obsidian vault | Create `knowledge/` at global + project scope. Move existing bundle content (`decisions/`, `glossary/`, etc.) from `.skillgrid/` root into `.skillgrid/knowledge/`. Add `raw/`, `proposed/`, `comparisons/`, `concepts/`, `entities/`, `topics/` zones. Initialize `.obsidian/` vault (app.json, core-plugins.json, .gitignore). Frontmatter schema: OKF v0.2 + wiki-specific fields (see below). **Git checkpoint**: the vault is git-tracked (part of the repo). Every librarian run (ingest, review-apply) creates a git commit — rollback on mistake |

**Wiki frontmatter schema** (beyond OKF v0.2):
```yaml
# Raw source (bronze)
processed: false              # toggled to true after ingest
hash: "sha256:abc123..."      # cryptographic hash of raw content; change → re-ingest
source_url: "https://..."     # origin (Web Clipper, manual, URL)
tags: [video, youtube]        # classification tags

# Proposed page (silver)
revision: 1                   # increment on each human feedback round
status: needs_review          # needs_review | approved | rejected | applied
target: "concepts/graph-rag"  # where it will land in gold
confidence: 0.92              # LLM confidence score (Q2)
contested: false              # true if contradiction detected
open_questions:               # things the LLM is unsure about
  - "Does this conflict with the existing RAG page?"

# Gold page
contested: false              # flipped to true when a contradiction is detected
contradicted_by: "comparisons/persistent-wiki-vs-rag.md"  # link to comparison page
```

**Proposals structure** (each file in `proposed/`):
- `revision: N` — increment on each feedback round
- `status: needs_review | approved | rejected | applied`
- `target: <zone>/<page>` — where it will land
- `confidence: 0.0-1.0` — LLM score
- **Proposed change description** — what will be created/modified
- **Exact outline** with `[[wiki-links]]` — the planned content structure
- **Open questions** — what the LLM is unsure about
- **Human feedback** section — reviewer writes feedback here; next run produces `revision: N+1`
- **Routing flag**: `routine: true|false` — routine proposals can be batch-approved; complex ones need individual review
| B2 | Wiki FTS index | Tables in the **owning** SQLite store: `wiki_pages` (id, path, title, wiki_type, tags, status, created, updated), `wiki_links` (source_page, target_page), `wiki_fts` (FTS5 over title + body). Migration 071. Index builder: walks `knowledge/` vault, parses frontmatter + `[[links]]`, populates tables. CLI: `skillgrid wiki index --scope global\|project` (rebuild), `skillgrid wiki search --scope global\|project` (FTS query → page paths) |
| B3 | Ingest pipeline (deterministic control plane) | `skillgrid wiki ingest --scope global\|project` — **deterministic control plane**: the pipeline is code, not a skill. The LLM does semantic work (extraction, linking, scoring); the code does everything else (detection, hashing, validation, routing, commit). Flow: (1) **Pre-flight check** (code): scan `raw/` for files with `processed: false` or changed hash. **Also scan `web_cache`** (SQLite): `web_cache_lookup` finds cached remote content (Context7/Exa/DeepWiki/WebFetch snapshots, 256 KB cap, TTL-based). Agent or CLI can promote a `web_cache` entry into `raw/` as a markdown file (`skillgrid wiki ingest --from-cache <id>`), giving it durable, git-tracked storage. This bridges the ephemeral web cache (SQLite, TTL) to the durable vault (markdown, git). (2) **Redact credentials** (code, deterministic regex, Q6). (3) **Semantic extraction** (LLM, ADR-0023): read redacted source, extract concepts/entities, generate `[[links]]` + frontmatter + confidence score + open questions. The `sources` frontmatter field (`[{ id, resource, title }]`) records provenance — the URL/path + title of the origin. (4) **Contradiction detection** (LLM): check extracted claims against existing gold pages. On conflict → create a `comparisons/` page with both positions, mark affected pages `contested: true`. (5) **Validation** (code): check frontmatter validity, link targets exist, no orphan proposals. (6) **Route** (code): score ≥ `score_threshold` AND `routine: true` → auto-approve to gold. Otherwise → `proposed/` for HITL. (7) **Commit** (code): git commit the vault. **Key principle**: "The AI interprets the knowledge; the tools control admission." The LLM never writes directly to gold — only to `proposed/` or (if auto-approved) via the code path |
| B4 | HITL review (two-phase) | `skillgrid wiki review list/approve/reject --scope global\|project`. MCP tools: `wiki_review_list`, `wiki_review_approve`, `wiki_review_reject`, `wiki_review_revise` (with `scope` param). **Two-phase approval**: (1) **Approve** = record the decision in the proposal (`status: approved`), stop. No vault changes yet. (2) **Apply** = `skillgrid wiki review apply --scope global\|project` reads all approved proposals, moves them to the gold zone, creates/updates pages, updates `[[links]]` in existing pages, updates the index, reindexes, git commits. **Revise** = human writes feedback in the proposal's `Human feedback` section; next ingest run produces `revision: N+1`. On approve, **credential scan** runs (Q6): matches are flagged in the review UI. Config: `mnemonic.wiki.auto_approve: "score"` (default), `score_threshold: 0.8` |
| B5 | Maintenance / lint | `skillgrid wiki lint --scope global\|project` — validates frontmatter, checks orphan pages (no inbound links), broken `[[links]]`, contradictions (same entity, conflicting facts), index completeness. **Extends and replaces** `okf-lint.mjs`. MCP tool: `wiki_lint`. Output to `okf-metrics.jsonl` |
| B6 | Wiki query (agent-facing) | MCP tool: `wiki_query(query, limit, scope)` — searches `wiki_fts` in the owning store, returns top-k pages with path + title + snippet + scope label. Agent reads page via `read_file` for full content. `scope: "all"` searches both global + project vaults (RRF-fused) |
| B7 | Distill to operational memory | **Hybrid trigger** (Q5). Explicit: agent calls `fact_add scope=global` during the session when it decides a fact is durable. Passive: at session end, the 013 distill hook (extended to global scope) runs an LLM pass over the conversation + wiki pages read, auto-extracts durable facts into `global.sqlite`. Two distillation paths: (a) **to memory**: small, durable fact → `global.sqlite` (Tier 2); (b) **to global wiki**: cross-project knowledge → `~/.skillgrid/knowledge/proposed/` (Tier 1, global scope). Agent judgment determines which path (Q8) |
| B8 | Migration: bundle root → knowledge/ | Move all existing wiki content from `.skillgrid/` root to `.skillgrid/knowledge/`. Update all path references: `ASSUMPTIONS.md` LOCKED rows, `AGENTS.md`, `config.yaml` path keys, skills, hooks, scripts, `okf-lint.mjs`, `check-adr-invariants.mjs`, `state-drift-check.mjs`, `ship-drift-check.mjs`. Create `~/.skillgrid/knowledge/` global vault. `git mv` preserves history (`git log --follow`) |
| B9 | Skill + Web-UI docs update | **Mnemonic skill** (`.agents/skills/knowledge/mnemonic/SKILL.md`): add a **Step 3.6 — Wiki knowledge (on-demand)** section teaching the agent how to use `wiki_query(query, limit, scope)` to search the `knowledge/` vaults (global + project) and read pages for full content. Add `wiki_query`, `wiki_review_list`, `wiki_review_approve`, `wiki_review_reject`, `wiki_lint` to the **Tools** reference table. Add `wiki index/search/ingest/review/lint` to the **Subcommands** CLI reference. Update **What You Must Do When Invoked** to include "recall from the wiki before doing" alongside the existing `mem_search` step. Add the prefetch injection note (global facts are auto-injected at session start via the OpenCode plugin; the agent does not need to call anything). **Web-UI** (`skillgrid-ui/src/features/mnemonic/`): (a) **WikiPage.tsx** — new feature page: vault browser (tree of `knowledge/` zones), page viewer (markdown render + frontmatter chips, reuse `FrontmatterChips` + `MarkdownView` from `features/docs/`), wiki search bar (calls `wiki_query` via REST), scope toggle (global/project). (b) **WikiReviewPage.tsx** — HITL review queue: lists `proposed/` pages with confidence score, revision number, routing flag (`routine`/`complex`), target path, proposed change description, outline with `[[links]]`, open questions, human feedback text area. **Two-phase**: approve/reject buttons record the decision (Phase 1). A separate "Apply All Approved" button triggers Phase 2 (move to gold + reindex + commit). **Batch approve** for routine proposals. Revise = write feedback → next ingest produces `revision: N+1`. (c) **WikiGraphPage.tsx** — graph view of `wiki_links` (reuse the existing `KnowledgeGraphPage` pattern with `wiki_pages`/`wiki_links` as nodes/edges). Contested pages highlighted in a distinct color.
(d) **WikiDashboardPage.tsx** — dashboard: counts per zone, raw source coverage % (processed/total), pending review count, contested page count, recently updated pages. (Mirrors the "LLM Wiki Home" dashboard from the video.) (e) **Route + nav** — add the four pages to the SPA router and the sidebar nav under the Mnemonic section. (f) **API** — `wiki/api.ts`: REST clients for `GET /api/wiki/search`, `GET /api/wiki/pages`, `GET /api/wiki/links`, `GET /api/wiki/proposed`, `GET /api/wiki/dashboard`, `POST /api/wiki/review/approve`, `POST /api/wiki/review/reject`, `POST /api/wiki/review/apply`, `POST /api/wiki/lint`. (g) **Server endpoints** (Go, `mnemonic/internal/mcp/` or a new `mnemonic/internal/wiki/http.go`): REST handlers that read from the `wiki_pages`/`wiki_links`/`wiki_fts` tables and the `knowledge/` filesystem. |

**~2200-3200 lines. ADR-0050.**

## How the Tiers Interact

```
                    ┌──────────────────────┐
                    │   AGENT (any agent)   │
                    └──────┬───────────────┘
              ┌────────────┼────────────────┐
              ▼            ▼                ▼
     ┌────────────┐ ┌────────────┐  ┌────────────┐
     │ TIER 2     │ │ TIER 1     │  │ TIER 3     │
     │ Global Mem │ │ LLM Wiki   │  │ Session    │
     │ (global.   │ │ Global:    │  │ Archive    │
     │  sqlite)   │ │  ~/.skill- │  │ (per-proj. │
     │            │ │  grid/     │  │  SQLite)   │
     │            │ │  knowledge/│  │            │
     │            │ │ Project:   │  │            │
     │            │ │  <proj>/   │  │            │
     │            │ │  .skill-   │  │            │
     │            │ │  grid/     │  │            │
     │            │ │  knowledge/│  │            │
     └─────┬──────┘ └────┬───────┘  └────────────┘
           │         ┌────┴──────┐
           │         │SQLite FTS │
           │         │Index      │
           │         │(global.   │
           │         │ sqlite or │
           │         │ per-proj) │
           │         └───────────┘
           ▼
     ┌─────────────────────┐
     │  Frozen snapshot     │
     │  in system prompt    │
     │  (bounded, 2200 ch)  │
     └─────────────────────┘

     Distillation paths:
     Agent reads project wiki page (Tier 1, project scope)
       → finds a small durable fact (preference, convention)
       → calls fact_add scope=global (Tier 2) → next session's prefetch

     Agent reads project wiki page (Tier 1, project scope)
       → finds cross-project knowledge (shared entity, global topic)
       → proposes page in ~/.skillgrid/knowledge/proposed/ (Tier 1, global scope)
       → HITL review → global vault
```

## Key Design Decisions

1. **`knowledge/` is the wiki, at both scopes.** The bundle root (`.skillgrid/`) is config + state. `knowledge/` is the wiki. Global wiki at `~/.skillgrid/knowledge/`, project wiki at `<project>/.skillgrid/knowledge/`. Same name, same structure, same pipeline.

2. **Each `knowledge/` is an Obsidian vault.** Plain markdown + `.obsidian/` config. Git-tracked. Graph-viewable. Clipper-able. The agent already knows how to work in an Obsidian vault (existing `obsidian` skill).

3. **SQLite is the index, not the SoT, for the wiki.** Markdown files are the source of truth. SQLite is a derived index for fast search. Rebuildable from markdown at any time (`skillgrid wiki index`).

4. **Wiki FTS index lives in the store that owns the vault.** Global wiki → `global.sqlite`. Project wiki → `<project-id>.sqlite`. A corrupted wiki index is rebuildable; a corrupted `global.sqlite` is not (it's the memory).

5. **OKF and LLM-Wiki are one maintenance model.** The pipeline is the engine; OKF is the frontmatter spec it enforces. `okf-lint.mjs` is extended into `wiki lint`. The catalog and reverse-index may be LLM-maintained. One lint, one model.

6. **HITL is the default.** Ingested pages land in `proposed/`, not the gold zones. Approval required before they enter the index. The "silver" layer from the medallion architecture.

7. **No LLM in the hot path for wiki search.** `wiki_query` is pure FTS5 (sub-millisecond). LLM only used during ingest (offline, batch). Consistent with ADR-0012.

8. **Cross-agent by construction for Tier 2.** Global operational memory in `global.sqlite`, accessible via MCP. Any agent that speaks MCP can `fact_add scope=global`, `fact_search scope=global`.

9. **Global scope for memory AND wiki.** The Hindsight multi-agent pattern (one memory bank, multiple agents) applies to Tier 2. The global wiki (`~/.skillgrid/knowledge/`) provides cross-project knowledge at Tier 1. The agent in project A sees both: its project vault + the global vault. Cross-project knowledge sharing works at both tiers.

10. **Two distillation paths.** A fact from a project wiki page can be distilled (a) to global memory (Tier 2, small, always-injected) or (b) to the global wiki (Tier 1, large, on-demand). The agent picks based on the fact's scope and size.

11. **Score-based auto-approve (Q2).** Ingested pages get a confidence score from the LLM. Score ≥ `score_threshold` (default 0.8) → auto-approve to gold zone. Score < threshold → `proposed/` for HITL. Config: `auto_approve: "score" | "always" | "never"`.

12. **Plugin hook + MCP resource for prefetch (Q4).** OpenCode plugin `context` hook is the primary injection path. MCP resource (`memory/global`) is the portability fallback for non-OpenCode agents. CLI (`skillgrid memory global prefetch`) for debugging.

13. **Hybrid distillation (Q5).** Explicit `fact_add scope=global` during the session + passive LLM sweep at session end (extends the 013 distill hook to global scope). The passive sweep is the net that catches what the agent didn't explicitly save.

14. **Credential redaction, not injection scanning (Q6).** Deterministic regex scan strips credential patterns (API keys, tokens, passwords, SSH keys, connection strings) on ingest and on approve. Lighter than Hermes' full injection/Unicode scan because the threat model is local-first, single-user. The agent is the primary reader; the HITL review is the human gate.

15. **Agent judgment for scope (Q8).** The agent decides whether a fact belongs in the global or project wiki. Default guidance in the ingest prompt (ADRs/invariants → project; entities/topics → global). Optional `scope_defaults` config for hard rules. HITL review confirms.

16. **Deterministic control plane, not skill-driven.** The pipeline is code (Go), not a prompt. The LLM does semantic work (extraction, linking, scoring, contradiction detection); the code does detection, hashing, validation, routing, and git commit. "The AI interprets the knowledge; the tools control admission." This is the Agentic Librarian pattern: a skill tells the agent to *run code*, not to *do the work itself*. The LLM is replaceable (any OpenAI-compatible model via ADR-0023); the control plane is stable.

17. **Contradiction detection + comparison pages.** When new content conflicts with existing gold pages, the pipeline does NOT silently resolve. It creates a `comparisons/` page with both positions, a clear question ("should X be Y or Z?"), and marks all affected pages `contested: true`. The human resolves in HITL review. This prevents a single bad source from poisoning the vault with a false "opposing truth."

18. **Two-phase approval (approve → apply).** Approving a proposal records the decision but changes nothing in the vault. Applying (separate step) moves approved proposals to gold, updates links, reindexes, and git commits. This gives the human a window to reconsider before changes are material. A mistake in one proposal doesn't cascade.

19. **Git checkpoint on every run.** The `knowledge/` vault is git-tracked (part of the repo). Every librarian run (ingest, review-apply) creates a commit. Rollback is `git revert`. This is the safety net for the LLM: if extraction went wrong, the vault is one commit behind.

20. **Two fact targets with separate char budgets** (Hermes MEMORY.md / USER.md). The global store has a `category` column: `memory` (agent's operational notes, 2200 chars) and `user` (user profile, 1375 chars). The prefetch injection renders them as two labeled blocks with usage percentages. This mirrors Hermes' insight that "facts about how you work" (user) and "facts about the environment" (memory) are different curation decisions with different lifetimes.

21. **Batch operations with atomic limit check.** The char limit is checked only on the batch's final result, not per-operation. `fact_add` accepts an `operations` array: the agent can free space (remove stale entries) AND add a new entry in one atomic call. This is the Hermes pattern for "memory is full" — the error tells you how many chars to free, you batch the frees + the add together, and the limit is checked once on the result.

22. **Write approval for passive writes.** Explicit agent writes (`fact_add` during a session) go through freely. Passive writes (background distill hook) can be gated by `write_approval: true` → staged in a pending queue → `skillgrid memory global approve <id>|all`. This is the Hermes `/memory pending` pattern: "the agent saved a wrong assumption" → the human catches it before it enters the store.

23. **Frozen snapshot = session boundary.** Facts saved during a session are invisible to the running session (frozen prefix cache). They appear in the next session. The system prompt guidance tells the agent this: "Facts you save now will be visible to future sessions, not this one." This is why the distill hook runs at session *end* — it's the last chance to capture facts before the boundary.

24. **skillgrid is a memory/experience self-improvement system** (Future AGI taxonomy, mechanism #2). It does NOT optimize prompts (mechanism #1), update weights (mechanism #3), or search architectures (mechanism #4). It accumulates **durable facts** (memory) and **reusable skills** (Voyager pattern: "memory can be executable, not just text"). The distill hook is the Reflexion pattern at session-end granularity: verbal self-critique → episodic buffer → read back in the next session.

25. **Darwin Gödel Machine acceptance rule for the wiki.** "No change without a measured win." Every librarian run (ingest, review-apply) that modifies gold pages should be checkable: `wiki lint` before vs after. If the run introduces broken links, orphans, or contradictions, it's a regression — git revert. The frozen benchmark is the existing `okf-lint.mjs` finding count: a run that increases drift findings is suspect. This doesn't mean every run needs a numeric score (the wiki isn't an optimizer), but the **gate** exists: lint must not regress.

26. **`web_cache` → `raw/` bridge.** The existing `web_cache_save` (mnemonic skill, per-project SQLite, 256 KB cap, TTL-based) is the ephemeral capture layer. The `knowledge/raw/` vault zone is the durable layer (markdown, git-tracked, no TTL). `wiki ingest --from-cache <id>` promotes a web_cache entry into `raw/` as a markdown file with `source_url` + `hash` frontmatter. This bridges the two: research captured during a session (`web_cache_save`) becomes a durable bronze source when the agent (or human) decides it's worth preserving. The web_cache entry stays in SQLite (for fast FTS recall); the `raw/` file is the git-tracked, versioned, clipper-compatible copy.

27. **Skill improvement loop (cognee pattern): observe → inspect → amend → evaluate.** skillgrid's `skills` table + `skill_usage` (per-execution log) is the Voyager pattern: executable memory. The cognee loop closes it: (1) **Observe**: every `use_skill` execution logs outcome (success/failure) + optional feedback to `skill_usage`. (2) **Inspect**: `skill inspect <name>` aggregates usage rows, detects recurring failure patterns. (3) **Amend**: LLM reads evidence + current SKILL.md, proposes a patch → `SKILL.md.amended` (not in-place). (4) **Evaluate**: compare outcome rate before vs after. Improvement → apply + commit. Regression → discard, keep original. "Not just agents with skills, but agents with skills that can improve over time." The git checkpoint (Decision 19) is the rollback safety net: the original SKILL.md is never lost.

## Sizing & Sequencing

| Change | Lines | ADR | Depends on |
|---|---|---|---|
| A: Global Operational Memory + Skill Improvement Loop | ~1100-1500 | ADR-0049 | 003, 014, 011 (facts + skills schema) |
| B: LLM-Wiki Pipeline + Skill + Web-UI | ~2200-3200 | ADR-0050 | A (uses `global.sqlite`), ADR-0048 (amended), ADR-0023 |

**Order: A first, then B.** A is smaller, unblocks Hermes-style memory immediately, creates `global.sqlite` which B needs for the global wiki index.

**Total: ~3300-4700 lines. Two ADRs. No new external dependencies.** (Obsidian is an optional viewer, not a dependency — the vault is plain markdown. The Web-UI reuses existing React components from `features/docs/` and `features/mnemonic/`. The skill improvement loop reuses the existing `skills`/`skill_usage` tables + migration 011.)

## Open Questions (research pending)

### ~~Q1: Wiki scope — per-project or global?~~ **RESOLVED**

Both. Global wiki at `~/.skillgrid/knowledge/` (cross-project). Project wiki at `<project>/.skillgrid/knowledge/` (scoped). Same name, same structure, same pipeline. The agent sees both.

### ~~Q2: Auto-approve vs HITL?~~ **RESOLVED**

**Hybrid based on score.** The LLM returns a confidence score with each extraction. Pages scoring above a threshold auto-approve (move directly to the gold zone + reindex). Pages below the threshold land in `proposed/` for HITL review.

Config:
```yaml
mnemonic.wiki:
  auto_approve: "score"       # "score" (default) | "always" | "never"
  score_threshold: 0.8        # 0.0–1.0; pages ≥ threshold auto-approve
```

- `score` (default): score ≥ threshold → auto-approve; score < threshold → `proposed/`
- `always`: everything auto-approves (fast path, low-trust environments)
- `never`: everything goes to `proposed/` (HITL, high-trust environments)

The LLM prompt for ingest (B3) includes: "Return a confidence score (0.0–1.0) for each extracted concept. Score reflects how certain you are the extraction is accurate, complete, and non-duplicative." The score is stored in the page frontmatter (`confidence: 0.92`) and in the `wiki_pages` table.

### ~~Q3: Wiki index in `global.sqlite` or a separate `wiki.sqlite`?~~ **RESOLVED**

The wiki FTS index lives in the **store that owns the vault**. Global wiki → `global.sqlite`. Project wiki → `<project-id>.sqlite`. A corrupted wiki index is rebuildable (`skillgrid wiki index`); a corrupted `global.sqlite` is not (it's the memory). The index is derived data; the markdown is the SoT.

### ~~Q4: Prefetch injection mechanism~~ **RESOLVED**

**OpenCode plugin hook.** A `context` hook in the OpenCode plugin (ADR-0045 V2 `Plugin.define`) reads the global store's top-k facts by AKL importance at session start and injects a frozen block into the system prompt. This is the same mechanism the existing 6 native plugins use.

For non-OpenCode agents (Kilo, Cursor, Codex): the frozen snapshot is also exposed as an **MCP resource** (`memory/global`) as a fallback. Agents that implement MCP resource reading get the snapshot automatically. Agents that don't can call `fact_search scope=global` (on-demand, no injection).

The plugin hook is the primary path (OpenCode is the primary agent). The MCP resource is the portability fallback. The CLI (`skillgrid memory global prefetch`) prints the block for debugging.

### ~~Q5: Distillation trigger — explicit or passive?~~ **RESOLVED**

**Hybrid.** Explicit during the session + passive sweep at session end.

- **Explicit (during session):** the agent calls `fact_add scope=global` when it decides a fact is durable. Simple, agent-controlled. (Hermes pattern.)
- **Passive (at session end):** the existing distill hook (change 013) extends to global scope. At session close, an LLM pass (ADR-0023 OpenAI-compatible client) scans the conversation + wiki pages read and auto-extracts durable facts into `global.sqlite`. The hook is async + best-effort (a distill failure never breaks session close), same as the existing per-project distill.

Config:
```yaml
mnemonic.global_memory:
  distill: true              # passive sweep at session end (default: true)
  distill_llm: "auto"        # "auto" = ADR-0023 client; model from config
```

The passive sweep is the net that catches what the agent didn't explicitly save. The explicit path is for facts the agent is confident about in the moment. Together they cover both the "I just learned something important" case (explicit) and the "the conversation had durable facts I didn't flag" case (passive).

### ~~Q6: Security scanning for wiki content~~ **RESOLVED**

**Remove credentials.** The security scan is a deterministic pre-scan that strips credential patterns (API keys, tokens, passwords, SSH private keys, connection strings) from wiki content before it enters the index. It does **not** block prompt injection or invisible Unicode — those are the agent's responsibility (the agent reads the page and decides if it's trustworthy), and the HITL review (Q2) is the human gate.

The scan runs:
1. **On ingest** (B3): before the LLM extracts concepts, raw sources are scanned for credential patterns. Matches are redacted (`[REDACTED:api_key]`) before the LLM sees them. This prevents the LLM from embedding a credential into a wiki page.
2. **On approve** (B4): before a proposed page moves to the gold zone, it's scanned for credential patterns. Matches are flagged in the review UI (the human sees `[REDACTED:api_key]` and can decide to keep or remove).

The scan is regex-based (deterministic, zero-LLM, sub-millisecond). Patterns: AWS keys (`AKIA[0-9A-Z]{16}`), GitHub tokens (`ghp_[A-Za-z0-9]{36}`), OpenAI keys (`sk-[A-Za-z0-9]{20,}`), generic `password=...`, `token=...`, `secret=...`, SSH private key blocks (`-----BEGIN ... PRIVATE KEY-----`), connection strings (`postgresql://user:pass@...`).

This is lighter than Hermes' full injection/exfiltration/Unicode scan because the threat model is local-first, single-user. The agent is the primary reader; the human is the reviewer. The scan is a safety net, not a gate.

### Q7: ADR-0048 amendment (layout) + ADR-0050 (pipeline)

ADR-0048 put the bundle root at `.skillgrid/` (R6 dissolution). This change:
1. **Moves the wiki content** from `.skillgrid/` root to `.skillgrid/knowledge/` (layout change)
2. **Adds the LLM-Wiki pipeline** (ingest, HITL, lint, FTS index)
3. **Adds global scope** (`~/.skillgrid/knowledge/`)

Two ADRs needed:
- **ADR-0049** (Change A): Global operational memory
- **ADR-0050** (Change B): LLM-Wiki pipeline + layout move. Amends ADR-0048 (the bundle root is no longer the wiki; `knowledge/` is). References ADR-0012 (the wiki index is SQLite; the wiki content is markdown — ADR-0012's engine decision applies to the index, not the content).

The layout move is a **supersession of ADR-0048's layout decision** (point 5: "the bundle root is `.skillgrid/` itself"), not a supersession of ADR-0048's SoT decision (SQLite for observations, markdown for knowledge). The SoT is unchanged; the location changes.

### ~~Q8: What's in the global wiki vs the project wiki?~~ **RESOLVED**

**Agent judgment.** The agent decides where a fact belongs based on its scope. The HITL review (Q2) confirms. No hard content-type rules.

Default guidance (in the ingest prompt, not enforced by code):
- ADRs, invariants, project architecture → project wiki
- People, companies, tools, shared conventions → global wiki
- Cross-project topics, research that applies broadly → global wiki
- Project-specific reference, workflows → project wiki

The agent can override the default. A fact about a project's database choice that's also a shared convention ("we use PostgreSQL 16 across all projects") goes global. A fact about a specific ADR goes project. The HITL reviewer (or the score threshold, Q2) is the check.

Config (optional, for environments that want hard rules):
```yaml
mnemonic.wiki:
  scope_defaults:
    decisions: "project"      # ADRs always project
    invariants: "project"     # locked constraints always project
    entities: "global"        # people/companies/tools default global
    topics: "global"          # cross-project topics default global
```

When `scope_defaults` is set, the ingest prompt includes the defaults and the LLM is instructed to follow them. When unset (default), the LLM uses judgment. Either way, the `scope` field in the page frontmatter is what the review/approve step uses to route the page to the correct vault.

## Out of Scope (this change)

- **Passive extraction** (LLM-powered sync after each turn) — Phase 2, needs ADR-0023 LLM in the sync path
- **Knowledge graphs / SPO triples / reflect** (Hindsight's advanced features) — Phase 2+
- **Multiple wiki providers** (Mnemosyne-style plugin slot) — skillgrid's mnemonic IS the provider
- **Team-shared memory** (ADR-0024, multi-user PG sync) — separate change
- **Obsidian plugin development** — the vault uses core Obsidian features (wiki-links, frontmatter, graph, clipper). No custom Obsidian plugins required
- **Cloud sync for the vault** — the vault is git-tracked, local-first. Sync is via git (or the user's existing git workflow)

## Glossary

| Term | Definition |
|---|---|
| **Global Operational Memory** | Cross-project, cross-session, cross-agent fact store in `global.sqlite` with AKL importance, char limit, and prefetch injection |
| **LLM-Wiki** | LLM-maintained, interlinked markdown knowledge base (Karpathy pattern) with a SQLite FTS index for fast retrieval, stored in an Obsidian vault |
| **Knowledge vault** | A `knowledge/` directory that is an Obsidian vault: plain markdown + OKF frontmatter + `[[wiki-links]]` + `.obsidian/` config. Exists at global (`~/.skillgrid/knowledge/`) and project (`<project>/.skillgrid/knowledge/`) scope |
| **Prefetch injection** | At session start, read top-k global facts by AKL importance and inject a frozen block into the system prompt (Hermes frozen-snapshot pattern) |
| **Distillation** | Extracting a durable fact from a wiki page (Tier 1) into operational memory (Tier 2) via `fact_add scope=global`, OR proposing a page in the global wiki (Tier 1, global scope) |
| **Medallion architecture** | Bronze (raw/) → Silver (proposed/) → Gold (zones) trust pipeline for LLM-Wiki content |
| **Frozen snapshot** | System-prompt injection captured once at session start, never changes mid-session (preserves prefix cache) |
| **AKL importance** | 014 importance scoring: `retrieval_count × exp(-decay × age_days)` + maturity tiers |
| **OKF v0.2** | skillgrid's knowledge format spec: frontmatter schema, catalog (`index.md`), reverse-index, lint. The frontmatter spec the LLM-Wiki pipeline enforces |
