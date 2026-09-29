# Findings — llmwiki-okf-wiki (Pillar 1 `.wiki/` compiler)

Research captured 2026-09-29 from the user's source list (11 sources). Text sources are cached in web_cache (ids 2–7). Findings that constrain the change are called out under **Design impact**; the rest is context.

## Sources (all 11)

### Primary pattern + spec (normative for our change)
1. **Karpathy — "LLM Wiki" gist** (https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f) — *the* source pattern. web_cache id 2.
2. **OKF v0.2 SPEC** (https://raw.githubusercontent.com/GoogleCloudPlatform/knowledge-catalog/main/okf/SPEC.md) — already cached as web_cache id 1; normative field set.
   - Note: a parallel canonical repo now exists at `github.com/GoogleCloudPlatform/open-knowledge-format` (SPEC.md) and `.../knowledge-catalog/okf/SPEC.md` is what we cite. v0.2 added provenance/trust/lifecycle/attestation (§5–§10); `type` is the only required key.

### Commentary + practitioner write-ups
3. **Marie Haynes — "OKF from Google is a new layer for agents"** (https://www.mariehaynes.com/okf/, 2026-06-16). web_cache id 4.
4. **Marie Haynes — "Build an OKF brain like mine!"** (https://www.mariehaynes.com/build-an-okf-brain-like-mine/, 2026-06-26). web_cache id 5.
5. **Alden Do Rosario — "I Benchmarked 14 LLMs for LLM-Wiki and OKF"** (https://medium.com/@aldendorosario/…91f47c4de1e0). web_cache id 3. Repo: `adorosario/okf-model-bakeoff`.

### Reference implementations (feature long-list / follow-up backlog)
6. **nashsu/llm_wiki** (https://github.com/nashsu/llm_wiki, 17k★, Rust+TS+Python) — full desktop app of the pattern. web_cache id 7.
7. **sniperunder123/okf-knowledge** (https://github.com/sniperunder123/okf-knowledge, MIT, Python) — portable `/okf` Claude Code skill. web_cache id 6.
8. **dsiu/okf-llm-wiki** (https://github.com/dsiu/okf-llm-wiki) — installable skill; uses Obsidian `[[wikilink]]` inside an OKF v0.1 bundle.
9. **pumblus/okf-harness** (https://github.com/pumblus/okf-harness) — agent-first, local-first, terminal-native harness (`okfh --json` tool surface, source-trail checks, local HTML graph report).
10. **supachai-j/open-knowledge-format-starter** (https://github.com/supachai-j/open-knowledge-format-starter) — ready-to-fork starter: `AGENTS.md` schema + `raw/` + `wiki/`, tools (`validate`, `viz`, `index` BM25, `embed`, `search` RRF), MCP server, conformance CI.
11. **real-jiakai/okf_llm_wiki** (https://github.com/real-jiakai/okf_llm_wiki) — side-by-side demo of llm-wiki vs OKF on the same Claude domain.
12. **starmynd-org/infinite-brain-os** (https://github.com/starmynd-org/infinite-brain-os, MIT, Shell, 238★) — git-backed "knowledge OS" for a business: namespaces + **canon-vs-synthesis promotion path**, typed entities (commands/agents/skills/rules/workflows/tools), a `_system/` contract layer + `validate.sh` ("errors are the contract"), `PROVENANCE.yml` lineage, intake routing, outputs-with-lineage, `data/` = pointers not bytes. web_cache id 8.

### Videos (YouTube — titles resolved where possible)
- `T33iI6izAKw` — **"Finally, an Open Standard for the Karpathy LLM Wiki is HERE"** — Cole Medin (2026-07-02). OKF standard + live migration walkthrough + a shareable AI-coding knowledge bundle (two-layer index + a small CLI to list bundles / view index / read by bundle+concept-id).
- The other four IDs (`_bieksxg6oY`, `mWLDn49_8HA`, `z02Y-1OvWSM`, `xUnVQkPrnrA`) are the same topic cluster (OKF/llm-wiki/second-brain builds). Related same-topic videos surfaced during retrieval: *"The Ultimate Knowledge Base: Bring YouTube Into Your AI Second Brain"* (Cole Medin, 200-video OKF bundle), *"Karpathy's LLM Wiki: What It Means & How to Build One"* (Tonbi's AI Garage), *"Google Just Replaced Karpathy's LLM Wiki (OKF EXPLAINED)"* (EverydayAI School).
- **Open:** user can supply exact titles/dates for those 4 IDs if we want precise citations; not needed for design.

## Key takeaways

### Design impact (constrains our Pillar 1)
- **Link syntax is a real fork.** Karpathy's pattern + community (nashsu, dsiu, Tonbi) use Obsidian `[[wikilinks]]`; the **OKF SPEC §6 standard uses plain markdown links** (`/tables/customers.md` or relative). Our spec already chose `[[slug]]`/`[[slug|alias]]` + `source_path` backlinks to satisfy Obsidian/llmwiki. **This is the one place our output is deliberately *not* OKF-canonical link-syntax** — worth an explicit note in `design.md` (extension/profile choice, like dsiu) so the conformance gate doesn't try to force markdown links.
- **`type` is the only required key; everything else is recommended/extension.** Our minimal-frontmatter + no-null-omission rule is exactly right. `source_path`, `related_videos`, `okf_version` are all legitimate extension keys.
- **Progressive disclosure is universal.** Every implementation navigates `index.md` first, then drills in. Our `index.md` (type taxonomy + slug rule + lint rule) matches; consider whether `index.md` should also enumerate pages (Cole Medin's two-layer index) — optional, follow-up.
- **`index.md`/`log.md` are reserved files.** Root `index.md` MAY carry `okf_version` (openknowledge.ai linter confirms v0.2 convention; no frontmatter otherwise). Keep `log.md` append-only.
- **Lint is a first-class operation** in the pattern (Karpathy), nashsu, openknowledge.ai, okf-harness. Our `wiki lint` (conformance) is the structural half; the *semantic* health check (contradictions, orphans, missing pages, gaps) is a follow-up (Pillar 2 / LLM pass) — matches our out-of-scope note.
- **infinite-brain-os validates the substrate, and adds a contract layer.** Same bet we're making — plain md + YAML, git-native, the agent is the runtime, no DB/server. Its `_system/validate.sh` ("errors are the contract: a fresh clone has zero, and every change must keep it that way") is a stronger form of our `wiki lint`: a repo-wide validator as the invariant, not just per-page conformance. Note as a Pillar-2 stretch: a `wiki validate` that gates the *whole* bundle (cross-page + link + registry checks), not just frontmatter.
- **`data/` = pointers, never bytes.** infinite-brain-os stores *pointers* to where numbers live, not the numbers themselves. Reinforces our rule that the wiki carries references/sources, not duplicated raw data.

### Generator / model choice
- **Pick a cheap model, verify it clears your gate on your corpus, stop.** Premium tier buys no detectable quality difference on structured doc summarization at small scale (Do Rosario, 14-model bakeoff, $6.69 total). CIs on the headline comparison were `judge_degraded` → treat rankings as provisional. Re-run periodically; expect the decision to go stale.
- **Implication for us:** Pillar 1 is the *deterministic* (no-LLM) extract of explicit/structural knowledge, so it sidesteps this entirely. When Pillar 2 adds the LLM synthesis/extract pass, a cheap model is the right default.

### Feature long-list (candidates for follow-up changes, NOT Pillar 1)
From nashsu/llm_wiki (product-scale reference): two-step CoT ingest + incremental cache, multimodal image ingestion, multi-format parsing (PDF/Office/EPUB/Org), source-grounded "read sources only" mode, 4-signal knowledge graph (direct links, source overlap, Adamic-Adar, type affinity), Louvain community detection, graph insights (surprising connections / gaps), optional LanceDB vector search, persistent ingest queue (crash recovery), source-folder auto-watch, deep research, MCP server + local HTTP API.
From supachai-j: conformance CI gate, `viz.html` single-file Cytoscape+marked graph viewer, BM25 index + hybrid (RRF) search, MCP `okf_search`/`okf_get_concept`/`okf_read_index`/`okf_propose_change` (PR-gated writes), advisory leases for concurrent writers.
From okf-harness: source-trail verification (does each citation land on registered-source bytes), local self-contained graph report.
From openknowledge.ai: frontmatter JSON-schema rulesets (required/recommended/provenance/computation + index), `ok lint` (headless, non-zero on warnings) + `ok audit` (server-backed link/whole-project checks).
From Marie Haynes: typed folders (concepts/entities/playbooks/references/systems), playbooks as an executable type, automated daily-doc-ingest that auto-updates reference files.
From infinite-brain-os (business-scale, Pillar 2+): **namespaces with a canon-vs-synthesis promotion path** (operator-approved canon carries verification + changelog; synthesis is derived/agent-authored), **typed entities** (commands/agents/skills/rules/workflows/tools), **`PROVENANCE.yml`** machine-readable lineage (source commit, export date, pipeline version), **intake routing** (source captures → routing → processed receipts), **outputs-with-lineage**, **reviewed-learnings `memory/`**, a **whole-repo validator** as the contract, and a **wager-ledger** closing OODA Act→Orient.

### Consensus mental model (stable across all 12)
- **llm-wiki is the workflow, OKF is the wire format.** (real-jiakai's line, repeated everywhere.)
- Three layers: immutable `raw/` sources (source of truth) → LLM-compiled `wiki/` → schema (`AGENTS.md`/`CLAUDE.md`). LLM owns the wiki layer; human curates sources + asks questions.
- Compile once, keep current — not RAG re-derivation per query. The wiki is a persistent, compounding artifact; good answers get filed back as new pages.
- **Git-native, validator-as-contract, agent-as-runtime** is the shared spine (OKF bundles, infinite-brain-os, supachai-j, okf-harness all converge here).
- Identity by path; links ARE the graph; frontmatter = the query layer (filter by `type`/`tags` without opening the body); git-native.
