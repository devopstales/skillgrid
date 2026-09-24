# Findings — mnemon memory system

## Research: Mnemon (mnemon-dev/mnemon) — LLM-supervised persistent memory

**Decision served:** Compare against our own Mnemonic memory system (skillgrid's
`~/.skillgrid/mnemonic`) to find gaps, borrowable ideas, and differentiation.

**Access date:** 2026-09-24. **Repo:** github.com/mnemon-dev/mnemon (master branch).
**Repo stats at fetch:** 593 stars, 71 forks, 1,453 commits, Go 1.24+, Apache-2.0.
**Note on naming collision:** "Mnemon" (this project, /ˈniːmɒn/) is a distinct
product from "Mnemonic" (ours). Both derive from Greek μνήμων but are unrelated codebases.

---

### 1. What is this project?

Mnemon is **LLM-supervised persistent memory for AI agents** [1]. A single Go
binary (~593★) that gives any LLM CLI (Claude Code, Codex, Cursor, OpenClaw, Pi,
DeepSeek Harness, etc.) cross-session persistent memory. Core pitch: "the `mnemon`
memory path remains one local binary with zero API keys and one setup command" [1].

The defining architectural choice is the **LLM-Supervised pattern** [2]:

| Pattern | LLM Role | Representative |
|---------|----------|----------------|
| **LLM-Embedded** | Executor inside the pipeline | Mem0, Letta |
| **File Injection** | None — reads file at session start | Claude Code Memory |
| **MCP Server** | Tool provider via MCP protocol | claude-mem |
| **LLM-Supervised** | External supervisor of a standalone binary | **Mnemon** |

The binary handles **deterministic computation** (storage, graph indexing, search,
decay); the host LLM makes **judgment calls** (what to remember, how to link, when
to forget). No embedded LLM, no API keys, no middleman [1][2].

The same executable ships a second, independent surface: **`mnemon agency ...`**
(durable, project-local responsibility and effect admission for an existing Pi
agent). Agency and Memory share the binary but have separate state and authority [3].

### 2. Architecture

**Single executable, two product paths** [3]:

```
                         mnemon
                            |
              +-------------+-------------+
              |                           |
       root Memory commands        mnemon agency ...
              |                           |
  model / graph / search / store    View / Intent / admission
  embed / import / setup assets     Artifact / peer / attachment
              |                           |
       named Memory stores          project .mnemon/agency
```

**Package structure** [3] (the `internal/` tree):

```
mnemon/
├── main.go                    # Process entry point
├── cmd/
│   ├── root.go                # Compose the single product command
│   ├── memory/                # Root Memory commands and Memory flags
│   └── agency/                # `mnemon agency` user commands
├── internal/
│   ├── memory/
│   │   ├── model/             # Memory Insight and Edge values
│   │   ├── graph/             # Four-graph edge construction and traversal
│   │   ├── search/            # Recall, intent detection, and deduplication
│   │   ├── embed/             # Optional Ollama embeddings
│   │   ├── importdraft/       # Memory draft validation and import
│   │   ├── store/             # Memory SQLite persistence
│   │   └── setup/             # Memory runtime integration and embedded assets
│   ├── agency/                # Immutable Agency protocol values and projections
│   ├── authority/             # View sealing, Intent admission, durable fact writer
│   ├── artifact/              # Content-addressed immutable evidence
│   ├── peerlink/              # Replaceable authenticated peer transport
│   ├── daemon/                # Local authority process composition and lifecycle
│   ├── agencyclient/          # Runtime-facing terminal and replay journal
│   └── attach/                # Agency Hook, guide, and tool projection
├── test/mnemond/              # Agency boundary and scenario suites
├── testdata/mnemond/          # Data-only Agency fixtures
└── scripts/e2e_test.sh        # Memory CLI end-to-end suite
```

**Design philosophy — "Tools are Organs, Skills are Textbooks"** [2]:

| Game Development | Agent Ecosystem | Mnemon Equivalent |
|-----------------|-----------------|-------------------|
| Game engine (Unity/Unreal) | LLM CLI | Host environment |
| Native plugin (C++ Plugin) | Binary tool | `mnemon` binary |
| Script/Blueprint (C#/Blueprint) | Skill (.md) | `SKILL.md` |
| Gameplay logic | Agent behavior config | `guide.md` |

The binary is the **organ** (defines what *can* be done — deterministic); the skill
markdown is the **textbook** (defines *how* — teaches the LLM when to retrieve, how
to judge). "Memory management logic moves from prompt to code — deterministic,
testable, portable" [2].

**Integration projection** [7] — `mnemon setup` deploys three artifacts into each
runtime:

| Artifact | Role |
|----------|------|
| Runtime-specific `SKILL.md` | Teaches command syntax, output interpretation, guardrails |
| `~/.mnemon/prompt/guide.md` | Shared recall/writeback/linking/no-op guidance |
| Native hooks or extensions | Surface bounded reminders at lifecycle points |

**Four hook phases** (behavioral contract, not hard workflow) [7]:

```
Prime   (session start)  → make skill/guide/active-store visible
Remind  (prompt arrives) → prompt a recall decision
Nudge   (after response) → prompt a writeback decision
Compact (pre-compaction) → preserve critical continuity
```

### 3. Key features

- **Zero user-side operation** — install once; the agent runs all commands [1]
- **LLM-supervised** — host LLM decides what to remember/update/forget [1]
- **Multi-framework support** — Claude Code, Codex, Cursor, ZCode, TRAE, Qoder,
  CodeBuddy, WorkBuddy, Kimi, OpenCode, Hermes, OpenClaw, Pi, MiniMax, Nanobot,
  DeepSeek Harness [1]
- **Four-graph architecture** — temporal, entity, causal, semantic edges [4]
- **Intent-native protocol** — `remember`/`link`/`recall` map to cognitive vocabulary
  (not INSERT/CREATE EDGE/SELECT); structured JSON with signal transparency [1][2]
- **Intent-aware recall** — graph traversal + optional vector search (RRF fusion) [1]
- **Built-in deduplication** — byte-identical skips + advisory similarity suggestions [5]
- **Retention lifecycle** — EI decay, access-count boosting, auto-pruning, GC [6]
- **Privacy-safe receipts** — hashed operation export for memory-boundary audits [4]
- **Optional embeddings** — works fully without; Ollama/OpenAI-compatible for hybrid [6]
- **Named stores** — `MNEMON_STORE=work` for per-project/agent isolation [3]
- **Self-updating CLI** — `mnemon update` (npm-managed) [4]
- **Visualization** — `mnemon viz` exports DOT (Graphviz) or interactive HTML (vis.js) [4]

### 4. Data model

**Insight (memory node)** [3]:

```
┌─────────────────────────────────────────────┐
│ Insight                                     │
├─────────────────────────────────────────────┤
│ id         : UUID                           │
│ content    : "Chose Qdrant over Milvus..."  │
│ category   : decision                       │
│ importance : 5  (1-5)                       │
│ tags       : ["vector-db", "architecture"]  │
│ entities   : ["Qdrant", "Milvus"]           │
│ source     : "user"                         │
│ access_count        : 3                     │
│ effective_importance : 0.85                 │
│ created_at : 2026-02-18T10:00:00Z           │
└─────────────────────────────────────────────┘
```

**Categories** (6 types) [3]: `preference`, `decision`, `fact`, `insight`,
`context`, `general`.

**Importance** (1-5) [3]: 5 = never auto-cleaned; 4 = immune to auto-pruning;
3 = standard; 2 = low priority; 1 = first to be cleaned.

**Edge (relationship)** [3]:

```
┌────────────────────────────────────────────┐
│ Edge                                       │
├────────────────────────────────────────────┤
│ source_id  : UUID  ──→  target_id : UUID   │
│ edge_type  : temporal | semantic |         │
│              causal   | entity             │
│ weight     : 0.0 ~ 1.0                    │
│ metadata   : {"sub_type": "backbone", ...} │
└────────────────────────────────────────────┘
```

**SQLite schema** (WAL mode, one `.db` per named store) [3]:

```sql
insights (
  id, content, category, importance,
  tags, entities, source,
  embedding,                    -- Optional, 768-dim vector (BLOB)
  access_count, last_accessed_at,
  effective_importance,
  created_at, updated_at, deleted_at
)

edges (
  source_id, target_id, edge_type,  -- composite PK
  weight, metadata, created_at
)

oplog (
  id, operation, insight_id, detail, created_at
)
```

**Data directory layout** [3]:

```
~/.mnemon/
├── active                        # Current default store name
├── prompt/                       # Shared across all stores
│   ├── guide.md                  # Behavioral guide
│   └── skill.md                  # Skill definition
└── data/
    ├── default/mnemon.db         # SQLite (WAL)
    ├── work/mnemon.db
    └── <name>/mnemon.db
```

**Store resolution priority** [3]: `--store` flag > `MNEMON_STORE` env >
`~/.mnemon/active` file > `"default"`.

**Embedding storage** [6]: little-endian float64 BLOB in `insights.embedding`
(768 × 8 = 6144 bytes/insight). Ollama `nomic-embed-text` by default.

### 5. CLI / API

**Core memory commands** [4]:

```bash
# Write
mnemon remember "Chose Qdrant over Milvus" \
  --cat decision --imp 5 --entities "Qdrant,Milvus" \
  --tags "architecture,search" --source agent
mnemon remember "Raw note" --no-diff

# Read (intent-aware graph retrieval, default compact JSON)
mnemon recall "vector database" --limit 10
mnemon recall "why did we choose Qdrant" --intent WHY
mnemon recall "auth" --basic          # SQL LIKE fallback
mnemon search "authentication"        # token-scored keyword
mnemon show <id>                      # full content by ID
mnemon recall "vector database" --brief --excerpt-chars 160

# Bulk
mnemon import memory_draft.json [--dry-run] [--no-diff]

# Delete
mnemon forget <id>                     # soft-delete

# Graph
mnemon link <src> <dst> --type semantic --weight 0.85
mnemon link <src> <dst> --type causal --weight 0.8 \
  --meta '{"sub_type":"causes","reason":"..."}'
mnemon related <id> --edge causal --depth 2   # BFS

# Lifecycle
mnemon gc --threshold 0.5 --limit 20
mnemon gc --keep <id>

# Stores
mnemon store list
mnemon store create work
mnemon store set work
mnemon store remove old-project

# Observability
mnemon status
mnemon log [--limit 50]
mnemon receipt [--limit 50]           # hashed audit export

# Embeddings
mnemon embed --status
mnemon embed --all
mnemon embed <id>

# Visualization
mnemon viz --format dot -o graph.dot
mnemon viz --format html -o graph.html

# Setup
mnemon setup [--target claude-code|codex|cursor|...] [--global] [--yes]
mnemon setup --eject
```

**Remember flags** [4]: `--cat` (6 categories), `--imp` (1-5), `--tags`,
`--entities`, `--entity-mode` (merge/provided/auto), `--source` (user/agent/external),
`--no-diff`.

**Recall flags** [4]: `--limit` (default 10), `--intent` (WHY/WHEN/ENTITY/GENERAL),
`--cat`, `--source`, `--basic`, `--brief`, `--excerpt-chars` (default 240),
`--verbose`.

**JSON output shape** (compact, default) [4]:
```json
{"id":"...","content":"...","category":"...","importance":5,
 "intent":"ENTITY","matched_via":"keyword","confidence":"high","score":0.73}
```

**Remember JSON output** [5]:
```json
{
  "id": "abc-123",
  "action": "added",
  "diff_suggestion": "ADD",
  "edges_created": {"temporal": 2, "entity": 3, "causal": 1, "semantic": 1},
  "semantic_candidates": [
    {"id": "def-456", "content": "...", "cosine": 0.72, "auto_linked": false}
  ],
  "causal_candidates": [
    {"id": "ghi-789", "content": "...", "hop": 1, "suggested_sub_type": "causes"}
  ],
  "embedded": true,
  "effective_importance": 0.85,
  "auto_pruned": 0,
  "auto_pruned_ids": []
}
```

**Environment variables** [4]:

| Variable | Default | Purpose |
|----------|---------|---------|
| `MNEMON_DATA_DIR` | `~/.mnemon` | Base data dir |
| `MNEMON_STORE` | *(active/default)* | Named store |
| `MNEMON_MAX_INSIGHTS` | `1000` | Auto-prune ceiling |
| `MNEMON_AUTO_PRUNE_MIN_AGE` | `24h` | Grace period |
| `MNEMON_EMBED_ENDPOINT` | `http://localhost:11434` | Embedding API |
| `MNEMON_EMBED_MODEL` | `nomic-embed-text` | Embedding model |
| `MNEMON_EMBED_PROTOCOL` | *(auto-detect)* | `ollama` / `openai` |
| `MNEMON_EMBED_API_KEY` | *(none)* | Bearer token |
| `MNEMON_EMBED_DIMENSIONS` | *(native)* | Matryoshka truncation |

**Dependencies** [1]: Go 1.24+, `modernc.org/sqlite`, `spf13/cobra`, `google/uuid`.
No Redis, Neo4j, Qdrant, or Python required.

### 6. Notable design patterns, novel approaches

**a) LLM-Supervised pattern** [2] — The binary is an "organ" (deterministic
computation), the host LLM is the "supervisor" (judgment). This is a clean
separation: regex/heuristics handle 80% of automation, LLM judgment handles the
20% requiring deep understanding. Zero API cost, LLM swappable.

**b) Four-graph model (MAGMA)** [4] — Four distinct edge types, each capturing one
dimension:

| Graph | Purpose | Auto-creation |
|-------|---------|---------------|
| **Temporal** | Chronological skeleton | Backbone (new→recent same-source) + 24h proximity (`w = 1/(1+hours_diff)`) |
| **Entity** | Shared-entity links | Regex + 200+ term dictionary; up to 5 edges per shared entity |
| **Causal** | Reason/cause-effect | Causal keywords + token overlap ≥15%; LLM reviews 2-hop BFS candidates |
| **Semantic** | Meaning similarity | cos ≥0.80 auto-link; 0.40-0.79 LLM review; token overlap fallback |

**c) Intent-adaptive retrieval** [4][5] — Different query intents activate
different graph traversal weights:

| Intent | Causal | Temporal | Entity | Semantic |
|--------|--------|----------|--------|----------|
| **WHY** | **0.70** | 0.20 | 0.05 | 0.05 |
| **WHEN** | 0.15 | **0.65** | 0.10 | 0.10 |
| **ENTITY** | 0.10 | 0.05 | **0.55** | 0.30 |
| **GENERAL** | 0.25 | 0.25 | 0.25 | 0.25 |

WHY queries use wider beam (15) and deeper traversal (5) because causal chains span
multiple hops. WHEN uses temporal. ENTITY uses entity+semantic.

**d) Signal transparency** [5] — Each recall result includes a per-signal
breakdown. The host LLM sees *how* each candidate was scored (keyword, entity,
similarity, graph) and can make better re-ranking judgments than any algorithm
inside the pipeline. "This is a unique innovation in Mnemon" [5].

**e) `remember`/`link`/`recall` as universal algebra** [4] — The three primitives
map to Extract→Candidate→Associate, the minimal complete interface for any agent
memory system. Every existing system (Mem0, Letta, OpenViking, native RAG) is an
instantiation with varying degeneracy of `link`. Mnemon makes `link` a first-class
primitive (not folded into write or read).

**f) Read-write symmetry on graphs** [4] — Write path (text→graph) and read path
(graph→text) both follow Extract→Candidate→Associate. The LLM masters one cognitive
pattern for both. This does NOT hold for relational/document/KV stores.

**g) Memory Gateway protocol** [2] — The LLM↔Database interaction layer has no
standard protocol (MCP covers LLM↔Tool, ODBC/JDBC covers App↔DB, but LLM↔DB with
memory semantics is the gap). Mnemon's protocol surface (`remember`/`link`/`recall`
+ structured JSON + hooks) is analogous to MCP for the memory domain. Future:
protocol decoupled from storage — can sit on PostgreSQL, Neo4j, or any graph DB.

**h) Intent detection is lexical, not LLM** [5] — Fixed pattern matching across
13 languages (English, Chinese simplified/traditional, Hindi, Spanish, Arabic,
French, Bengali, Portuguese, Indonesian, Russian, German). No LLM call, no
provider. `--intent` override for language-independent control.

**i) Effective Importance (EI) decay** [6]:

```
EI = base_weight(importance) × access_factor × decay_factor × edge_factor

base_weight:   imp 5→1.0, 4→0.8, 3→0.5, 2→0.3, 1→0.15
access_factor: max(1.0, log(1 + access_count))
decay_factor:  0.5 ^ (days_since_access / 30)   // 30-day half-life
edge_factor:   1.0 + 0.1 × min(edge_count, 5)    // up to +0.5
```

**j) Immunity rules** [6] — `importance >= 4` OR `access_count >= 3` are exempt
from auto-pruning.

**k) Two-stage dedup** [5] — Byte-identical content is skipped (no insert);
near-duplicates are advisory (`diff_suggestion`: UPDATE/CONFLICT/DUPLICATE) and
inserted. Similarity alone never authorizes replacement. The LLM must explicitly
`forget <old-id>` after verifying the new fact.

**l) WHY post-processing: causal topological sort** [5] — Kahn's algorithm
arranges WHY results so causes come first, effects follow.

**m) Theoretical grounding** [2] — Three academic anchors:
- **RLM** (Zhang/Kraska/Khattab, 2025): LLM as orchestrator, not data processor
- **MAGMA** (Zou et al., 2025): Four-graph model + intent-adaptive retrieval
- **Graph-LLM structural insight** (Joshi/Zhu 2025): LLM attention ≡ GNN
  operations — graph memory is a structural match, not engineering convenience

### 7. Context window / token limit handling

Mnemon addresses the context window problem at **three levels** [1][2][4][7]:

**a) Compact output projection** [4] — Default recall output is optimized for LLM
consumption: compact JSON with `id`, `content`, `category`, `importance`, `intent`,
`matched_via`, `confidence`, `score`. `--brief` mode goes further: flattens
whitespace, caps each excerpt (`--excerpt-chars`, default 240), emits unindented
JSON, includes a `detail_command` hint. The agent fetches full content on demand
via `mnemon show <id>` — a **progressive disclosure** pattern that keeps the
context window small during discovery.

**b) `--limit` control** [4] — Default 10 results. The agent can set `--limit`
to match its context budget. `--verbose` restores the full payload (signals,
traversal metadata, timestamps) when the agent needs more detail.

**c) Pre-compaction hook (Compact phase)** [7] — The fourth hook phase fires
*before* context compaction: "preserve only critical continuity." The guide
instructs the agent to save what matters before the runtime compresses the
conversation. This is the direct answer to "context compaction drops critical
decisions" — the problem Mnemon was built to solve [1].

**d) Memory replaces context** [1][2] — The fundamental strategy: persistent
memory means the agent doesn't need to keep decisions in the context window
across sessions. "Long conversations push early information out of the window"
[1] — Mnemon's graph recall brings it back on demand. The RLM finding that an
8B model handles "100x beyond its context window by treating data as external
environment variables" [2] underpins this: the memory store IS the external
environment.

**e) No token budgeting inside the binary** — Mnemon does not tokenize or count
tokens. It operates on character counts (8000-char content cap [5]) and result
counts (`--limit`). Token efficiency is the agent's responsibility, guided by the
compact output and `--brief` projection.

### 8. Recall / search / retrieval pipeline

**Smart Recall** is the default retrieval mode (5-stage pipeline) [5]:

```
Step 1: Intent Detection
    Lexical patterns → WHY | WHEN | ENTITY | GENERAL
    (13 languages, no LLM, --intent override)

Step 2: Multi-Signal Anchor Selection (RRF Fusion)
    Signal 1: Keyword     → KeywordSearch(all_insights, query, top-20)
    Signal 2: Vector      → CosineSimilarity(query_vec, all_embeddings, top-20)
    Signal 3: Recency     → sort by created_at DESC, top-20
    Signal 4: Entity      → insights sharing entities with the query

    RRF Score = Σ 1/(k + rank_i + 1)  (k = 60)

Step 3: Beam Search Graph Traversal
    For each anchor:
      priority_queue = [(anchor, initial_score)]
      while budget_remaining:
        node = pop(priority_queue)
        for edge in GetEdgesFrom(node):
          structural_score = edge.weight × intent_weight[edge.type]
          semantic_score   = cosine(vec_neighbor, vec_query)
          total = score_node + λ₁·structural + λ₂·semantic
          // λ₁ = 1.0 (structural), λ₂ = 0.4 (semantic)
          if total > best_score[neighbor]: update, push

    Adaptive parameters:
      WHY:     beam=15, depth=5, max_visited=500
      WHEN:    beam=10, depth=5, max_visited=400
      ENTITY:  beam=10, depth=4, max_visited=400
      GENERAL: beam=10, depth=4, max_visited=500

Step 4: Multi-Factor Re-Ranking
    keyword_score  = token_intersection / query_token_count
    entity_score   = matched_entities / max(1, query_entities_count)
    similarity     = cosine(vec_candidate, vec_query)
    graph_score    = (traversal_score - min) / (max - min)

    final = w_kw·keyword + w_ent·entity + w_sim·similarity + w_gr·graph

    Intent-specific weights:
      WHY:     kw=0.10, ent=0.10, sim=0.30, graph=0.50
      WHEN:    kw=0.15, ent=0.15, sim=0.30, graph=0.40
      ENTITY:  kw=0.20, ent=0.40, sim=0.20, graph=0.20
      GENERAL: kw=0.25, ent=0.25, sim=0.25, graph=0.25

Step 5: WHY Post-Processing
    Kahn's algorithm topological sort on causal edges
    → causes come first, effects follow
```

**With/without embeddings** [4][6]:

| Capability | Without | With |
|------------|---------|------|
| Recall anchors | Keyword + recency | Keyword + vector + recency (RRF) |
| Semantic edges | Token overlap (coarser) | Cosine ≥0.50 (precise) |
| Traversal scoring | Pure structural | Structural + semantic |
| Rerank weights | KW 45%, Ent 25%, Graph 30% | KW 30%, Ent 15%, Sim 35%, Graph 20% |

**Graceful degradation** [6] — When the embedding provider is unavailable
(2-second timeout check), reranking automatically redistributes similarity weight
to keyword and graph signals. No degraded-mode flag needed.

**Basic recall** [4] — `mnemon recall "query" --basic` bypasses all of the above
and uses simple SQL LIKE matching. Fast, no graph traversal, no intent detection.

**Search command** [4] — `mnemon search "query"` is token-scored keyword search
(separate from recall). Supports `--brief` and `--excerpt-chars`.

**Write-path candidate discovery** [5] — `remember` outputs semantic candidates
(cos 0.40-0.80) and causal candidates (2-hop BFS) for the LLM to evaluate. The
binary does low-cost candidate discovery (regex + token overlap); the LLM does
high-value judgment (should I link this? is this a conflict?).

**Dedup/conflict** [5] — Built into `remember`:
1. Byte-identical → skip (`action="skipped"`)
2. Different content → insert (`action="added"`) with advisory `diff_suggestion`
3. `--no-diff` → unconditional insert

---

### Comparison notes (Mnemon vs Mnemonic)

Key differentiators worth considering for our system:

| Aspect | Mnemon | Mnemonic (ours) |
|--------|--------|-----------------|
| **Storage** | SQLite WAL, single file per store | SQLite (per-project) |
| **Graph model** | 4 edge types (temporal/entity/causal/semantic) | ? |
| **LLM role** | Supervisor (external, no embedded LLM) | ? |
| **Recall** | Intent-adaptive Beam Search + RRF fusion + 4-signal rerank | Semantic search (FTS5) |
| **Dedup** | Byte-identical skip + advisory similarity | ? |
| **Lifecycle** | EI decay (30-day half-life) + immunity + auto-prune | ? |
| **Protocol** | `remember`/`link`/`recall` CLI primitives | `mem_save`/`mem_search`/`mem_get_observation` MCP tools |
| **Embeddings** | Optional (Ollama/OpenAI), graceful degradation | ? |
| **Context handling** | Compact JSON + `--brief` + pre-compaction hook | ? |
| **Multi-runtime** | 15+ runtimes via `mnemon setup --target` | ? |
| **Signal transparency** | Per-signal breakdown in recall output | ? |
| **Intent detection** | 13-language lexical patterns, no LLM | ? |
| **Causal reasoning** | 4th graph type + topological sort for WHY | ? |
| **Agency** | Second surface (durable work, peer collaboration) | ? |
| **Distribution** | Single Go binary, npm/brew/go install | Go binary via skillgrid CLI |

**Notable gaps in Mnemon** (potential Mnemonic advantages):
- No built-in code indexing (ours has `code_search`/`code_explore`/`code_impact`)
- No web research caching (ours has `web_cache_lookup`/`web_cache_save`)
- No team task management (ours has `team_spawn_task`/`agent_pull_next_task`)
- No session relay/handoff (ours has `session_handoff`/`session_resume`)
- No LLM-based recall (ours uses semantic search over L1 overviews with L2 full details)
- Mnemon's "recall" is the agent's job (the LLM decides when to recall) — more
  cognitive load on the agent; ours provides structured MCP tools the agent can
  call directly

**Notable Mnemon innovations** (potential Mnemonic gaps):
- Four-graph model with intent-adaptive traversal
- RRF fusion of 4 signals (keyword/vector/recency/entity)
- Beam Search with intent-specific budget parameters
- Causal topological sort for WHY queries
- Signal transparency in recall output
- EI decay with 30-day half-life + immunity rules
- 13-language lexical intent detection
- Memory Gateway protocol vision (LLM↔DB standard layer)
- Read-write symmetry (single cognitive pattern for read+write)
- Named stores with env-var isolation
- Privacy-safe hashed receipts
- Pre-compaction hook (Compact phase)

---

### Sources

[1] Mnemon README — https://github.com/mnemon-dev/mnemon (accessed 2026-09-24)
[2] Mnemon Design Philosophy — docs/design/02-philosophy.md (accessed 2026-09-24)
[3] Mnemon Core Concepts — docs/design/03-concepts.md (accessed 2026-09-24)
[4] Mnemon Usage & Reference — docs/USAGE.md (accessed 2026-09-24)
[5] Mnemon Read & Write Pipelines — docs/design/05-pipelines.md (accessed 2026-09-24)
[6] Mnemon Lifecycle & Embedding — docs/design/06-lifecycle.md (accessed 2026-09-24)
[7] Mnemon LLM CLI Integration — docs/design/07-integration.md (accessed 2026-09-24)
[8] Mnemon Graph Model & Theory — docs/design/04-graph-model.md (accessed 2026-09-24)
[9] Mnemon Design Decisions — docs/design/08-decisions.md (accessed 2026-09-24)
