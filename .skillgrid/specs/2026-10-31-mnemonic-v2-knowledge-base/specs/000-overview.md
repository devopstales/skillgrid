# 000 — Knowledge-Base: Overview

**Status:** DRAFT · **Owner:** knowledge-base change
**Depends on:** `2026-10-31-mnemonic-v2-memory` (`00-foundation/000-architecture` — the
memory↔wiki model; `00-foundation/001-storage` — the shared store; `00-foundation/002-schema`
— the shared DB; `00-foundation/004-transport-mcp` §5 — the `SaveEvent`), the
`2026-10-31-mnemonic-v2-code-index` change (the code graph — cross-referenced, not yet
written).

---

## 1. What this function is

The knowledge-base (formerly **llm-wiki**, "the brain") is the **single synthesized
surface** of mnemonic-v2. Per the memory↔wiki model
(`00-foundation/000-architecture` §3):

- **Memory is the raw store** — atomic durable facts, no distillation, no retrieval-for-
  understanding. Its retrieval is *lookup*.
- **The KB is the brain** — it reads memory + code + git history + external sources,
  synthesizes them into **grounded markdown pages**, and serves *deep retrieval*
  (wiki-first, memory-fallback).

The KB's one-line model:

> **The markdown files in `.skillgrid/knowledge/` are the source of truth. The FTS index
> is a derived cache in the project's shared DB (the same store as memory). Grounded
> pages are pinned to code and verified.**

## 2. The two layers (and what was dropped)

| Layer | On disk | Ownership |
|---|---|---|
| **Page-type folders** | `decisions/ invariants/ workflows/ architecture/ gotchas/ glossary/` | **Source of truth.** Editable by agent + user (LLM-generated, human-reviewable). |
| **Derived index** | `index/last-update.json` (checkpoint) + `kb_pages` / `kb_pages_fts` / `kb_claims` **tables in the shared DB** | Rebuilt by `kb_index` / `kb_reindex`, never hand-edited. |

**Dropped (decisions KB-8, KB-9):**

- **No `raw/` layer.** There is no source-packet store. Memory facts, the code graph, and
  git history are **read live** from their own stores. External sources (the reborn
  web-cache) are **cited in page frontmatter** (`external:` URL + version), not hoarded.
  The `sources:` frontmatter holds *citations*, not stored artifacts.
- **No `outputs/` layer.** The vault **is** the OKF v0.2 bundle — every page already has
  OKF frontmatter, wikilinks, and stable citations. There is no translation step to a
  separate bundle, no rendered-HTML layer, no digests. Git-track the vault; read it; link
  it.

## 3. The loop

```
   memory store ────┐
   code graph ──────┼─► GENERATE (LLM, incremental, entity-targeted)
   git history ─────┤        │   (all read live — no raw/ packets)
   external (URL) ──┘        ▼
       .skillgrid/knowledge/{decisions,invariants,workflows,architecture,gotchas,glossary}/*.md
                                           ← SOURCE OF TRUTH (files)
                             │  kb_index (rebuild FTS into the shared DB)
                             ▼
      ┌───────────────────────────────────────────────────────┐
      │  project store (SAME DB as memory)                    │
      │  kb_pages · kb_pages_fts · kb_claims · …              │  ← derived, rebuildable
      └───────────────────────────────────────────────────────┘
                             │
             ┌───────────────┼───────────────────────┐
             ▼               ▼                       ▼
        kb_index          kb_query              kb_validate
   (rebuild FTS in    (search → cited        (pin-check: why↔code,
    shared DB)          sections; wiki-first  fails turn on drift)
                          retrieval)
```

**The write path** is eager-per-save: the memory function emits a `SaveEvent`
(`00-foundation/004-transport-mcp` §5 of the memory change); the KB **subscribes** and
runs an **incremental, entity-targeted** regeneration that touches only the pages the
fact affects. A clean run (no new/changed input) is a **no-op** — it writes nothing.

**The read path** is wiki-first: `kb_query` returns cited markdown sections. The agent
falls back to raw memory (`mem_lookup`) only when the KB is thin/empty on a topic
(OPEN-12).

## 4. The ten specs

| Spec | What it defines |
|---|---|
| `001-structure` | The on-disk layout (`.skillgrid/knowledge/`), the sentinel + ownership model, the code-first page taxonomy, the OKF page format + frontmatter, `.knowledgeignore`. |
| `002-sources` | The four source adapters, **all read live, none stored**: memory (SaveEvent → live read), code (graph), git history, external (cited by URL+version, re-fetchable). The `sources:` citation format. |
| `003-generate` | The generation pipeline: the graduate-rule (which facts → pages), LLM synthesis, **incremental entity-targeted** regeneration (eager-per-save), no-op clean runs, the resumable checkpoint. |
| `004-index` | The **FTS index** as a derived cache in the **shared project DB** (`kb_pages` + `kb_pages_fts` + `kb_claims`), `kb_index`/`kb_reindex` rebuild-from-markdown, freshness, `degraded` when the index is stale. |
| `005-query` | **`kb_query`** — the deep retrieval (wiki-first, OPEN-12): FTS over the index → cited markdown sections; on-the-fly backlinks; the memory-fallback handoff; **`kb_eval`** (the retrieval-eval regression gate over the mem/KB corpus — the measurement leg for the ranking / `thin` / semantic-fallback tunables). |
| `006-validate` | **The pin-check** (repocodex, first-class): `kb_validate` checks anchors against live code, flags stale pages, **fails the turn** on divergence; the hook/CI integration. |
| `007-lint` | Deterministic health (pi): orphans, broken wikilinks, dup aliases, coverage gaps, broken external citations; `kb_lint --fix` auto-repairs what is safe. |
| `008-transport` | The `kb_*` MCP tool surface + CLI (`knowledge` command group) + the event-bus subscription to the memory `SaveEvent`. |
| `009-skill` | **The `kb` management skill** (the behavioral layer, pi-llm-wiki's `SKILL.md` adapted): the golden rules, the recall-at-start / save-at-end discipline, two-stage links-first recall, page conventions, the workflows. |
| `010-decision` | **The System-1 decision-model layer** — the third intelligence seam (beside `Embedder`/`LLM`): a non-chat model that returns a **decision + confidence**. The KB's three consumers: the **steerable reranker** (`005-query` §2.2), the **citation-confidence** check (`006-validate` §2.2), and the **router** (skip the expensive synthesis on a clean run). Default OFF (every consumer uses its deterministic floor; KB-INV-6). |

**Total: 11 spec files** (this overview + 10).

## 5. The invariants (re-stated for this function)

INV-1 (provenance: every page cites its sources), INV-2 (dual-mode — the index tables
live in the shared store, SQLite or PG), INV-3 (LLM-first with explicit `degraded`),
INV-4 (local truth — the markdown files *are* truth; the index is derived), INV-5
(retrieval trails), INV-7 (the agent curates what is durable enough to be a page).

**KB-specific invariants** (added by this function):

| # | Invariant |
|---|---|
| KB-INV-1 | **Files are truth.** The markdown in `.skillgrid/knowledge/` is the source of truth. The index is a derived cache that can always be rebuilt from the files. Delete the index; re-`kb_index`; identical results. |
| KB-INV-2 | **Pages are grounded.** Every grounded page (decision/invariant/workflow) carries `sources:` citations and, where it pins code, `anchors:`. An ungrounded grounded-page is a lint finding. |
| KB-INV-3 | **The index is never hand-edited.** `index/**` and the `kb_*` tables are rebuilt by `kb_index`/`kb_reindex`. Agent edits to them are a guardrail violation. |
| KB-INV-4 | **Generation is a no-op when clean.** A `kb_generate` run with no new/changed input writes nothing and reports `changed: 0`. The vault is not churned by idle runs. |
| KB-INV-5 | **The retrieval is measured, not just tuned.** A change to `kb_query`'s ranking, the `thin` threshold, or the semantic-leg fallback is gated by `kb_eval` (the frozen mem/KB case set's recall@k + fragile-case signal must not regress). `005-query` §8. |
| KB-INV-6 | **The decision model is an enhancement, not a dependency.** With `decision.enabled: false` (the default) every consumer (rerank / citation-confidence / router) uses its deterministic floor and the KB is fully functional. No KB read/gate/write path *requires* the decision model. `010-decision` §7. |

## 6. What this function is NOT

- **Not the raw store.** That is memory. The KB *reads* memory (live, via the event bus)
  and synthesizes; it does not duplicate facts into packets.
- **Not an OKF exporter.** The vault *is* the OKF bundle. There is no "render to OKF"
  step (KB-9).
- **Not a second authority over memory.** A page is a *synthesis* of facts; the
  underlying `sync_id` citations remain the provenance. If a page and a fact disagree,
  the fact wins (the page is stale — `kb_validate` / `kb_lint` surface it).

## 7. Design lineage

- **pi-llm-wiki** (zosmaai): the vault discipline (sentinel dir, page-type folders,
  derived index that is rebuilt not hand-edited), Obsidian-compatible wikilinks,
  deterministic **lint** (orphans / broken links / dup aliases / coverage gaps), the tool
  surface, the guardrails (generated metadata immutable to the agent), and — the key
  behavioral layer — the **management skill** (golden rules, recall-at-start /
  retro-at-end, two-stage links-first recall).
- **openwiki** (langchain): **grounded claims** (facts point to versioned source
  evidence, `repo://path#L40-82`), the **relocating version token** (line-range pins that
  survive code movement — `006-validate` §2.1), the **4-layer no-op proof** (clean runs
  are *proven* no-ops — `003-generate` §3), the per-page baseline manifest + resumable
  run checkpoint (`003-generate` §5), the `.openwikiignore` read boundary.
- **repocodex** (azaylamba): **pinned why** — concepts (TechnicalDecision /
  InvariantContract / BusinessWorkflow) stored beside code with **ripgrep anchors**; a
  **pin check** that attests the why still matches live text and **fails the turn** on
  divergence. (The token/span pin-check is repocodex's idea *upgraded* with openwiki's
  relocating version.)
- **neuledge/context** (neuledge): the **external-source model** — a versioned,
  searchable, cited reference (`name@version` identity + `source_url` / `source_commit` /
  `build_fingerprint` meta), **section-granular** FTS retrieval (title/section/content
  weights + token budget), and a **fingerprint-gated staleness check** (`002-sources`
  §2.4, `007-lint`). The v2 inversion: neuledge is download-once-local with freshness at
  *publish* time; the KB cites + **re-fetches on demand** and runs the staleness check at
  *re-fetch / lint* time.

**Convergence:** all four agree the KB is (a) file-based markdown you own, (b)
interlinked, (c) grounded (facts carry provenance to a source), (d) self-maintaining
(regenerated / linted / verified, not hand-kept). That is exactly the "single
synthesized surface, regenerable" model locked in `000-architecture`.

## 8. Open items

See §8 of each spec. Cross-cutting open items (resolved by default, listed here for the
review):

| # | Question | Default |
|---|---|---|
| KB-O1 | Anchor engine: ripgrep vs the code-index graph? | **Graph** for symbol-level, **ripgrep** for token-level (invariant literals). Both available. |
| KB-O3 | Does `kb_validate` fail the *agent turn* or just flag? | **Flag by default, fail in CI / on `--strict`** (repocodex's hook is opt-in). |
| KB-O5 | Page count budget? | **Soft cap** + `kb_status` reports coverage; no hard cap in v2. |
| KB-O6 | Source discovery (auto-find new external sources)? | **Defer to v2.1.** v2 records external citations explicitly (`kb_cite_source`). |
| KQ-O1 | Where the `kb_eval` cases live (vault YAML vs shared-store table)? | **Vault, git-tracked** (KB-INV-1); runs live in the shared store, cases in the vault. `005-query` §8. |
| KQ-O2 | Trail-promote: engine verb vs agent convention? | **Engine verb** `kb_eval promote <trail-id>` (atomic + auditable). `005-query` §8. |
| KQ-O3 | Does `kb_eval` gate the QA step or stay advisory? | **Gating** for changes touching `005-query`'s ranking/`thin`/semantic-fallback; advisory otherwise. `005-query` §8. |
| KQ-O4 | Is the `thin` threshold *tuned* by `kb_eval` (sweep) or just asserted? | **Sweep it** — recall@k across a threshold grid; the value is chosen by measurement. `005-query` §8. |
| DE-O1 | The v2 decision-model implementation: reuse Ollama (`llama3.2:1b`) in structured-output mode, or a dedicated judgment model? | **Ollama structured-output** for v2 (one local model, no new pull); a dedicated API is a v2.1 config option. `010-decision` §8. |
| DE-O2 | Does the rerank **replace** the fused RRF ordering or **re-rank within** it? | **Re-rank within** the fused candidate set (the legs compose; the decision model doesn't re-scan the vault). `010-decision` §8. |
| DE-O3 | Does a `contradicts` citation-confidence **fail the turn** under `--strict`? | **`cite_strict: false` by default** (a finding, surfaced, not a build-fail); `cite_strict: true` makes it a strict gate. `010-decision` §8. |
| DE-O4 | Is the router a **v2** or **v2.1** deliverable? | **v2.1** — weakest fit (single-model removes the "route between tiers" half); a cost/latency optimization, not a correctness concern. `010-decision` §8. |
| DE-O5 | One `Decide` call per form (N questions) vs a batched multi-state call? | **One form per call** (N questions, one state). Multi-state batching is an implementation detail behind the seam. `010-decision` §8. |

Resolved/moot: KB-O2 (no separate `kb.sqlite` — the index lives in the shared store),
KB-O4 (no external TTLs — no packets to expire), KB-O7 (no render targets — the vault is
the bundle).
