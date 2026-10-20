---
type: Spec
title: "2026-10-06-second-brain-learnings — findings"
status: draft

---

# Findings — 2026-10-06-second-brain-learnings (consolidated research/prototype/sketch evidence)

> **STATUS:** `draft` (2026-10-10)

## Research: pi-hermes-memory (2026-10-10)

**Source:** `github.com/chandra447/pi-hermes-memory` (477★, MIT) — Hermes-style persistent memory + learning loop for the Pi coding agent. Researched via repo README + DeepWiki (`Overview`, `Architecture Overview`, `Configuration System`, correction-detector/consolidation Q&A) on 2026-10-10.

**What it is:** Two-tier Markdown source-of-truth (`MEMORY.md`/`USER.md`/`failures.md`, `§`-delimited, atomic writes) + SQLite FTS5 mirror (`sessions.db`: memory mirror + `messages` session index) + event-driven learning loop (background review, correction detector, session flush) + `SKILL.md` procedural store. TS extension, `better-sqlite3` native addon.

### Adopted into THIS change (sharpens Features 3 + 5, no new tickets)

1. **Correction-detector structure → Feature 5 (`ClassifyCapture`).** Their `isCorrection` (`src/handlers/correction-detector.ts`) is a two-pass deterministic filter: **strong** patterns ("don't do that", "I said") always fire; **weak** patterns ("no", "wrong") fire only with a **directive word** (`use/do/change/...`, `hasDirectiveWord` over the message remainder); **negative** patterns ("no worries", "no problem") suppress even on strong/weak match. Patterns user-overridable via config (`correctionStrong/Weak/NegativePatterns`, `correctionDirectiveWords`). Blueprint Task 5 + spec Feature 5 updated to this structure; detection stays deterministic (no LLM), threshold 0.75 default retained as the weak-pattern confidence floor.
2. **Category search → Feature 3 (`knowledge_class`).** Their 6 categories (`failure|correction|insight|preference|convention|tool-quirk`) with `memory_search(query, category)` validates the dimension-not-KB design. Borrow under consideration: `tool-quirk` class and `failure_reason` field on failure writes. Recorded as an open micro-decision for TICKET-03 (default: keep 4 classes; add `tool-quirk` only if a test demands it).

### Deferred to a FOLLOW-UP change (out of scope here, do not expand TICKETs)

3. **Background review loop + flush hooks.** Nudge every 10 turns **or** 15 tool calls (fire-and-forget; `direct` in-process transport with `subprocess` fallback), flush on compact/shutdown (`flushOnCompact`, `flushOnShutdown`, `flushMinTurns: 6`). Mnemonic has only passive `Key Learnings` capture — scheduled learning is the missing piece.
4. **Policy-only mode + standing instructions.** Default injects a compact `<memory-policy>` (when to search; "repo evidence wins") instead of full memory — token saving vs our always-inject L1/L2. `STANDING.md`: 20 entries / 2000 chars, **user-only writes** (agent can never promote its memory into it), injected every session. The provenance rule is the key insight.
5. **Write-path content scanner.** `scanContent` + `scanSecrets` (`src/store/content-scanner.ts`) gate every memory/skill write: secret patterns (API keys/tokens/SSH) + prompt-injection threat patterns. We have `<private>` stripping + `private_tools` redaction but no write-path scanner.
6. **Usage-tracked consolidation.** Invisible `hit_count`/`last_hit_at` per entry on every search hit, fed to consolidation as tie-breakers (well-recalled = load-bearing); overflow strategies `auto-consolidate | reject | fifo-evict` + chunked consolidation. Our distill/dream has no recall signals — adopt hit tracking first (no behavior change), consolidation-use later.

### Explicitly NOT adopted (with reason)

- **Markdown as source-of-truth** — conflicts with ADR-0012 (SQLite is our SoT). Take the sync/backfill *discipline* (`syncMemoryEntry` idempotent mirror, `/memory-sync-markdown` backfill command pattern), not the storage model.
- **`better-sqlite3` native addon** — conflicts with our cgo-free invariant (their README documents the Node-ABI rebuild pain; our pure-Go choice stands).

### Traceability

- Items 1–2 → spec Feature 5 § Confidence-gated capture + Feature 3 § Durability (amended 2026-10-10, additive only; gates G3/G4/G6 unchanged).
- Items 3–6 → follow-up change candidate (not sliced, not ticketed).
- Repocodex items 1–7 below → ALL follow-up (current 5 tickets frozen; no blueprint/task deltas).

## Research: repocodex (2026-10-10)

**Sources:** `github.com/azaylamba/repocodex` (README, `docs/agents.md`, `docs/how-it-works.md`, `docs/memory.md` OKF v0.2 consumer guide, `docs/architecture.md` engine architecture) + live usage in `/data/git/AI/test-project` (`.repocodex.toml`: `engine_version 0.0.2`, `posture = "shadow"`, no `.context/` bundle yet — adoption playground). Docs + config only; no code executed.

**What it is:** Git-native "why next to code" with a pin check. Concepts (`TechnicalDecision`/`InvariantContract`/`BusinessWorkflow`/`GuardrailDecision`, one per why, orthogonal types) live in an OKF bundle at `.context/` beside the code. Each pins live text via `verification.anchors` (`path` + `all_of` terms, optional `near`) and checkable `claims` literals. CLI over ripgrep+git: `repocodex context <paths>` (retrieve why before edit, file-keyed via generated reverse index at `.repocodex/reverse-index.md`) + `repocodex validate --diff` (`LIVE / WEAK / WRITE + skipped_memory / REANCHOR / RECONCILE+DRIFT / CLAIM_BROKEN`). Pre-commit hook + CI + agent skills enforce the loop; `memory-exempt` is the governed hotfix override. Python 3.11+, requires `rg` on PATH.

**Engine architecture (from `docs/architecture.md`):** Deterministic attester (`engine/` package, no LLM) with 11 modules: `gate` (write gate + identity prefixes), `match` (term compile, regions, markers, claims), `liveness` (LIVE/WEAK/REANCHOR/DRIFT + CLAIM_BROKEN), `relocate` (rename + pickaxe candidates), `ratchet` (skipped-memory / first-touch / region discharge), `contradiction` (shared claim subject + double supersede), `impact` (intent-side `impacted_scenarios`), `code_impact` (advisory hit ranking), `blocking` (closed `REQUIRED_CHECK_REASONS`), `dilution` (advisory warnings), `ack` (review-ack token for `memory-exempt`).

**Closed blocking set** (5 reasons only; everything else is advisory):
1. `drift` — unrepaired DRIFT
2. `claim_broken` — any claim finding
3. `skipped_memory` — covered-file maintenance or first-touch of uncovered eligible file, undischarged
4. `index_sync` — committed reverse index out of sync with anchors
5. `contradiction` — two live concepts assert different literals for same claim subject, or two live concepts supersede same predecessor

Never blocks solely because `.context/` changed, WEAK degraded, dilution warned, or an agent-judged finding appeared.

**Posture semantics (as shipped):**

| Posture | Blocking behavior |
|---|---|
| `shadow` (default) | Blocks ONLY on undischarged `skipped_memory` (including first-touch). Other required reasons computed + reported but non-blocking. |
| `ratchet` | Any reason in the closed blocking set denies. |
| `full` | Same as `ratchet` in validate today. Scheduled sampling is the separate `repocodex audit` command. |

**Write gate (7 checks, fails closed):** match (anchor terms hit pinned path) → in-file uniqueness (term combo locates unique region) → distinctiveness (≥1 term under ceiling; path-only/import-only rejected) → claims (each literal in owning anchor's `all_of` + matched region) → exclusions (not in gitignore/`.repocodexignore`/config exclusion) → identity prefix (authored types under mapped folders) → markers/regex dialect. Reject payloads include `tighten`, `term_counts`, `suggestions`.

**Telemetry:** `.repocodex/metrics.jsonl` (gitignored; overridable via `REPOCODEX_METRICS_SINK`). Append-only JSONL: `{"event": <name>, "at": <utc>, ...payload}`. `validate` appends drift/reconcile events. `false_drift_rate` (share of DRIFT later shown false by reanchor reconcile) is a computed field in every validate envelope. Purpose: measure pin-check precision over the recorded window to inform shadow→ratchet promotion.

**Validate envelope (shape):** JSON with `result` (LIVE|WEAK|REANCHOR|RECONCILE|CLAIM_BROKEN|CONTRADICTION|WRITE), `posture`, `blocking`, `blocking_reasons[]`, `outcomes[]`, `lost[]`, `weak[]`, `claim_findings[]`, `patches[]`, `candidates[]`, `impacted_scenarios[]` (intent-side impact in every envelope: changed files → reverse index → concepts → OKF links → other pins), `dilution_warnings[]`, `identity_prefix_warnings[]`, `contradictions[]`, `index_sync[]`, `skipped_memory[]`, `changed_files[]`, `memory_exempt`, `exemption_refused`, `audit_entries[]`, `repair_tasks[]`, `false_drift_rate`, `latency_ms`, `engine_version`.

**Tree-mutation honesty:** Engine modules emit verdicts + patches; callers apply. Exceptions: `validate` appends metrics to `.repocodex/metrics.jsonl`; `audit` may deprecate orphan/expired drafts (GC); `write`/`reconcile`/`relocate`/`bootstrap`/`install` intentionally mutate the tree. Default `validate` without `--apply-patches` leaves `.context/` unchanged.

**Core insight for our problem:** freshness as content-based attestation, not time-based expiry. An ADR is stale when its anchors break, not 7 days after writing — obsoletes the TTL mindset behind D17 (ingest rows expiring) and the re-crawl instinct. Don't crawl for learnings after the fact; record at the point of change and prove liveness continuously.

### Transferable mechanisms (ranked; ALL follow-up — current 5 tickets frozen)

1. **Anchor liveness for ADRs.** Optional `verification:` frontmatter (anchors + claims) on `artifacts/04-adr-*.md` + a check beside the drift guards re-running anchors (`LIVE/DRIFT/CLAIM_BROKEN`). New third guard: spec-zone drift, structure drift, *knowledge liveness*. Land WARNING-only first.
2. **File-path-keyed why retrieval.** Deterministic "which decisions pin these files" lookup (reverse-index shape) next to `ctx_query`/`code_search`: ingest anchored concepts as observations with file-path metadata. Fixes the probabilistic-miss failure mode (same one pi-hermes-memory's standing-instructions names).
3. **Knowledge-coverage ratchet (the structural fix for "how does it learn").** `WRITE` + `skipped_memory` when a substantive edit touches an uncovered file; turn cannot finish until a pinning concept lands in the same change, **or** every substantive hunk falls inside a matched region of an attesting anchor (LIVE classification alone does NOT discharge the whole file; comment/whitespace-only edits do not arm; working-tree scope diffs against HEAD). Ours already owns the cousins (spec zone, zone guard, ship drift check = anticipated-files→diff); this is the reverse (diff-paths→knowledge-coverage). Land in `qa`/pre-commit as WARNING first.
4. **Why-change protocol.** Intent change requires `supersedes` + `rationale` in the same change — completes our ADR IRON RULE + bitemporal chain (ADR-0011); adopt the `rationale` requirement explicitly.
5. **Unanchored pages are still knowledge.** No-`verification` concepts load, just don't arm the pin check. Vocabulary for reports/retrospectives: process learnings first-class without anchors; conventions carry pins. Maps onto `knowledge_class` (episodic vs convention); legitimizes `reflect` → `06-research-findings.md` as designed.
6. **Shadow posture for every new gate.** Every liveness/coverage check lands as `qa` WARNING / `WINDOWS.md` entry before ever blocking (matches fail-open; mirrors test-project `posture = "shadow"`). Borrow `memory-exempt` shape for hotfixes (our fast-track waiver is the same; add one-command repair flow).
7. **Git-native journal + paired commits.** Append-only decision journal (their `.context/log.md` grouped by date) + "concept + reverse index ship in one commit, desync fails" — extend our `[skillgrid-context]` commit discipline to concept+index pairs.

### Explicitly NOT borrowed

- Storage layout (`.context/` OKF bundle) — our SQLite SoT (ADR-0012) and `.skillgrid/` zones stand. Mechanisms only.
- Distribution already matches (their MCP + installed skills ≈ our Hub Content via `install`); imitate their skill copy "follow that skill; do not invent a substitute".
- The `repocodex write` 7-check write gate as a standalone command — our OKF writes go through the existing skill pipeline (brainstorming → writing-blueprints), not a separate write gate. The gate *checks* (distinctiveness, identity prefix) inform our lint rules but don't become a separate CLI subcommand.

### Proposed shape (recap, for the follow-up change)

- Keep init ingest (bootstrap) + fix D17/D18 (`ExpiresAt` far-future, stable `Owner`).
- Add anchor liveness (1) + file-path retrieval (2): prove freshness, find decisions deterministically.
- Add coverage ratchet (3) in shadow mode: learn at write time, no crawler.
- Reports stay unanchored knowledge via `reflect` (5).
- Borrow `metrics.jsonl` shape for our shadow-posture telemetry: append-only JSONL of drift/reconcile events + `false_drift_rate` in the validate output. This is the data trail that justifies promoting a shadow gate to ratchet (findings item 6). Gitignored, overridable, not memory.

## Proposal: `.skillgrid` as OKF v0.2 wiki (2026-10-10)

Full text: `okf-wiki-plan.md` (same dir). RESTRUCTURE portion promoted to GOAL
(briefing R6, 2026-10-10) — pending spec-zone delta + re-gate; SKILLS portion
stays follow-up. Frozen TICKET-01..05 unaffected.

- Format OKF v0.2 confirmed; SoT hybrid (OKF bundle durable, SQLite retrieval
  index); DISSOLVE `artifacts/` into top-level `decisions/|invariants/|
  workflows/|glossary/|reference/`, root `index.md|log.md|reverse-index.md`,
  `archive/` → `xarchive/`; OKF frontmatter + indexing for all
  `.skillgrid/**/*.md` except YAML; pins via borrowed pattern, no new dep;
  reference sweep over every in-repo `artifacts/`/`archive/` path.
- Revisits the "Explicitly NOT borrowed — storage layout" line above: layout +
  format adoption is now a GOAL (briefing R6, same change; mechanisms-only
  stance stands for TICKET-01..05, which are frozen).
- New ADR takes the next free number at authoring time (NOT 0037/0038 — 0037 taken by the embedded-ANN index, 0038 reserved by
  TICKET-03 for the knowledge-class dimension; expect 0039); it amends ADR-0012,
  never deletes.

## Research: vector-DB-demotion video (2026-10-10)

**Source:** YouTube `YLkSfz3EFGw` (transcript fetched 2026-10-10 via
youtube-transcript skill; auto/manual captions as available). Topic: "is the
vector database dead" — TurboPuffer v3 demotes the vector index to one index
among keyword/filters; three challengers evaluated (agentic search, live MCP
connectors, compiled knowledge/OKF).

**Key evidence (as claimed in the video; single-source, treat as leads):**

1. **Agentic search beats fixed RAG pipelines on hard questions, ~3× cost.**
   PipesHub on Google FRAMES (824 Qs, 12k Wikipedia articles): agent 92.7% vs
   best-of-18 pipelines 78.9%; multi-hop (5+ articles) 90% vs 62%. Agent ≈
   perfect-retrieval ceiling (93%). Cost: ~3×/question; slowest answers 134s vs
   54s; 7% of questions ate 37% of budget. LREC'26 study: same pattern (>3×
   input tokens, +50% time; plain reranker beat the agent at ranking).
2. **BM25 often beats vectors inside the loop on enterprise data.** Enterprise
   RAGBench (~500k docs: Slack/Gmail/Jira/Confluence/GitHub): BM25 68.8% vs
   vectors 51.4% — vectors lost even on questions written to favor them
   (embeddings trained on public text miss code names/acronyms/ticket IDs:
   "customer IDs need equality, not a soulmate"). Counterpoint: BrowseComp Plus
   with GPT-5 agent — 70.1% embedding retriever vs 55.9% BM25, fewer searches
   (BM25 side called badly tuned); Cursor +12.5% adding semantic over grep.
   Embedding cost itself trivial (OpenAI small: $0.02/1M tokens) — the loop is
   the expensive part.
3. **Live MCP connectors skip the index; apps gatekeep bulk export.** MS
   federated Copilot connectors fetch live over MCP (no index); Anthropic Claw
   marketplace (2,000+ connectors, opened 2026-09-23); OpenAI restricts personal
   app-sync to admins; Slack API bans bulk export since 2025. Price: Glean
   measured live-connector tasks $2.98 vs $0.58 indexed, >3× tokens; Glean added
   182 live MCP templates (2026-09-22).
4. **Compiled-wiki (OKF-style) economics caution — directly relevant to
   `okf-wiki-plan.md`.** August study of the compile-upfront claim: wiki cost
   ~100× a RAG index to build (expected) but ~21× MORE tokens per query —
   "no break-even point exists". Citations far better (40% of wiki claims held
   up vs 19% RAG) — but n=13 questions, single author: warning light, not
   verdict. OKF repo itself quiet: last commit 2026-08-21 (sole contributor),
   88 issue comments unanswered. Idea continues in LangChain OpenWiki, Pinecone
   Nexus (compile company data ahead of time).
5. **Video verdict:** vector DB demoted from center to one tool. Default for new
   builds: agent + cheap search tools (keyword first, vectors for fuzzy
   meaning, live connectors for unowned apps); one-shot pipeline for cheap/fast
   high-volume lookups; grep + notes file for coding agents / small corpora;
   Postgres+pgvector (<10M vectors; patch to 0.8.7) per TurboPuffer CTO; compile
   knowledge only when stable + valuable. Deciding question: how many hops per
   query, and can users wait ~2 min for slow ones?

**Implications for the OKF-wiki proposal (follow-up change, not this one):**

- The proposal's hybrid SoT + FTS-first recall direction is *aligned* with the
  verdict (keyword/BM25 first, vectors as one leg — exactly Mnemonic's
  "FTS5 is the floor" + RRF design). No direction change.
- The August study is a scope warning: keep the bundle SMALL and curated
  (decisions/invariants/workflows/glossary — tens of files, human-scale), never
  a whole-corpus pre-compile. Our per-query cost stays flat because recall is
  index-lookup + one-hop titles, not "paste the wiki into the prompt". Record
  this as a design constraint in the follow-up briefing (bundle size cap;
  no auto-compiled corpus dumps).
- Citation-quality upside (40% vs 19%) supports the anchors + `verified`
  frontmatter discipline — the part of the proposal that makes claims checkable.
- OKF upstream quiet (single contributor, unanswered issues) confirms the
  borrow-pattern-no-dep decision: depend on the *format text* (SPEC v0.2,
  already fetched into this thread), not the repo's future.

**Traceability:** follow-up change candidate (OKF-wiki follow-up briefing,
design-constraint section). Frozen TICKET-01..05 untouched.

## Research: openwiki deep-dive (2026-10-10)

**Sources:** local repo `/data/git/AI/openwiki` (v0.7.2, TypeScript) explored
via 3 parallel research subagents (pipeline / MCP+search / skills+output) +
live example: `openwiki/` bundle the user generated on `/data/git/AI/test-project`
(8 pages, single init run `3dded03a`, `completedBy: opencode`,
`status: complete`). No code executed, no files modified outside this entry.

**What it is:** CLI that writes/maintains agent documentation as a linked
Markdown wiki (OKF v0.2). Durable page-job queue
(`begin → plan → N page workers → finish`), not one big agent. Ownership
split: agent does semantic research + single-page prose; OpenWiki owns
queue/state/claim-IDs/verification/indexes/finalization.

### Generation pipeline (`src/agent/`, `src/generation/`, `src/claims/`, `src/okf/`)

1. **Begin** (`repository-run.ts:388`): refresh `AGENTS.md` managed
   `<!-- OPENWIKI:START/END -->` block, snapshot sources (git fingerprint),
   Claims preflight, atomic write `openwiki/.run.json`. Updates no-op cleanly.
2. **Planner worker** (read-only tools + one `submit_plan` tool): repo-specific
   taxonomy (systems/workflows, NOT source-tree mirror) → ordered `PageJob[]`.
   `init` must include `quickstart.md`; `update` auto-injects stale-claim jobs.
   Prompts: `src/agent/repository-prompts.ts`; claim fragments:
   `src/claims/guidance.ts`.
3. **Page workers** (concurrency 1–8, quickstart held back last): each sandboxed
   to ONE writable path; sparse claim decisions (confirm/new+revised/retract);
   durability proved (`pageVersion == sha256` + verification projected) before
   job completion; failures restore + mark `skipped`.
4. **Finish** (fully deterministic, `wiki-finalizer.ts:248`): mermaid validate →
   `index.md` sync → link check → claim-sources projection → `generated`
   stamps. `.run.json` deleted LAST (the commit point); earlier throws stay
   resumable.

### Output structure (confirmed in test-project example)

`index.md` (generated nav only, root carries `okf_version: "0.2"`) +
`quickstart.md` (task router) + `<topic>/page.md` (H2-sectioned prose) +
per-dir `index.md` + `INSTRUCTIONS.md` (user brief, never rewritten) +
`.claims/<page>.json` (claims + `repo://path#Lx-Ly` versioned evidence) +
`.page-manifest.json` (coverage ledger: page → sha256 + gitHead + runId) +
`.last-update.json` / `.run.json`. Frontmatter split enforced by skill:
agent authors `type/title/description/tags`, system stamps
`verified/sources/generated`. `log.md` reserved-but-unwritten by the pipeline.

### MCP (10 tools, `integrations/core/protocol.ts:14`)

- **Lifecycle:** `begin, submit_plan, next_page, inspect_page_claims,
  submit_page, finish` (Pi bridge re-exposes only these 6).
- **Retrieval (model-free, local, read-only):** `search` (compact ranked
  sections), `read` (exact-anchor full sections, GitHub slugs),
  `list_workspaces`, `list_wikis`. Skill mandates LAZY retrieval (never
  preload; stop once grounded; verify against source).
- Cross-repo via `openwiki link` registry (`~/.openwiki/wiki-workspaces.json`);
  ambiguity returns explicit `workspace_required` instead of guessing.

### Indexing (3 indexes + 1 disposable ranker, deliberately separated)

- `.page-manifest.json` = coverage ledger; `.claims/*.json` = grounding state;
  per-call in-memory FTS5 (`porter unicode61`, BM25 title 8 / heading 6 /
  desc 4 / identifiers 3 / prose 1) over H2-section units = the ONLY search
  index (rebuilt per call → cannot go stale); `graph.json` = visualizer-only
  export, never read by search.

### Skills (3)

`integrations/openwiki/SKILL.md` (read-vs-generate modes, 8-step lifecycle,
page/claims/file contracts, non-negotiable boundaries) +
`skills/mermaid-diagrams/` (diagrams as validated quality gate, degrade to
`text` + repair ticket) + `skills/write-connector/` (built-in-TS-only,
`custom-mcp`-first, strict secret/state discipline).

### Adopted into the OKF-wiki proposal (`okf-wiki-plan.md`, follow-up only)

1. Ownership split (agent prose / scripts own stamps+indexes) → basis of
   proposed `knowledge-authoring` skill.
2. Sparse claim decisions + one-writable-path sandbox → matches our Files-column
   scope guard; adopt "state only what changed" for ADR updates.
3. Lazy retrieval + stopping rule → extend `mnemonic` skill verbatim in spirit.
4. Disposable ranking index + sections-as-units (frontmatter-weighted, H2
   sections) → recall design constraint (query-side, no schema change).
5. Graph/visualizer derived-only, read-only viz, degraded-diagram discipline →
   proposed `knowledge-diagrams` + lint behavior.
6. Explicit-ambiguity (`workspace_required`) → candidate for our
   cross-project reads (currently silent `all_projects`).

### Explicitly NOT adopted

- `.run.json`-style run state (our SDD `tasks.md` + `state.yaml` already own
  queue/state — no duplicate); `.claims/` sidecars (frontmatter-embedded
  `verification.anchors` per repocodex pattern unless bundle exceeds ~100
  concepts); `log.md`-as-reserved-unwritten (we write `log.md` as decision
  journal per repocodex); managed `AGENTS.md` block rewrite (our sentinel
  block stays lean-pointer style).

**Traceability:** seeds the follow-up OKF-wiki change (4 proposed skills:
`knowledge-authoring`, `knowledge-retrieval` via `mnemonic`,
`knowledge-lint`, `knowledge-diagrams`; sidecar-vs-frontmatter open point for
its briefing). Frozen TICKET-01..05 untouched.

## Research: byo-seedx-graph-lessons (2026-10-11)

**Sources:** `Lyra-stellAI/BYO-LLM-WIKI` README (layered KG layers 0–7,
contextual RAG, eval planes, skill gating, MCP deny-by-default) +
`CaoYuhaoCarl/seedx` README + raw `docs/specs/output-artifact-layout.md` +
raw `docs/adr/0001-question-to-mastery-architecture.md` (reader classes,
path-only index, `_agent/`/`_run/` split, contract artifact, main-agent
hygiene, ADR revisit sections). Docs only; no code executed.

**BYO adoptions (all scoped, LLM-heavy parts refused):**
1. Layer vocabulary `source → section → chunk → entity → topic → synthesis`
   → our `workflow → concept → file → symbol` (R10/R12 tour order).
2. Init→ingest→query→lint as the one lifecycle vocabulary across
   onboarding/authoring/recall/lint skills (record in `sdd-structure.md`
   at execution time, not here).
3. Eval-vs-observability planes kept separate (qa verdicts vs
   `session_events`; lint never judges, trails never gate).
4. Deterministic chunk prefixes `path · symbol · scope` (R13; the cheap
   half of contextual retrieval — no LLM summaries per the 21×/query video
   lesson, no OpenAI embeddings per local-first).
5. MMR-vs-RRF ablation before any ranker change (BYO: MMR 0.909 vs
   graph-RAG 0.788); graph-traversal retrieval stays rejected.
6. HNSW question moot: pure-Go adapter already landed (`hnsw.go`,
   migration 057, ADR-0037) with brute-force degrade.

**SeedX adoptions:** reader-class paragraph in root `index.md`;
`_` = internal for future folders; index entries paths-only (new
`okf-lint.mjs` check); ADR revisit sections as template norm (ADR-0012
already has them); `knowledge-base` recall hardened toward paths-first.
Parked: JSONL run-log mirror, `_`-folder reshuffle, committed-agent
`sdd/` question.

**Collision recorded (not just researched):** ADR-0037 was taken by the
landed embedded-ANN index while this sliced change assumed a 0036 highest —
two changes raced one sequence number (live serial-development violation,
surfaced by the LOCKED index as designed). Renumbered in this spec:
knowledge-class → 0038, R6 amendment → 0039, fusion contract → 0040
(verify-highest-at-authoring still applies to each).

**Traceability:** R13 ↔ Task 14 ↔ @step-14 (G15); conventions into
`okf-wiki-plan.md` + R6 scope notes; TICKET-13 after TICKET-04.
Frozen TICKET-01..05 untouched.

## Research: autodidact autonomous wiki loop (2026-10-11)

**Sources:** `DamonChen-anan/autodidact-skill` README (13★, 5 commits, MIT) —
autonomous research agent on Karpathy's llm-wiki: topic in, structured wiki
out. Four skills (`autodidact/questioner/researcher/wiki`), loop
`question → parallel search → raw/ → quality-weighted compile → gap audit →`
next round driven by the audit. Docs only; no code executed.

**Mechanisms worth stealing (all process, no dependency):**
1. **Quantified stop rule.** Halts when audit gaps diminish (≤60% of previous
   round) + question overlap >70% + no pending user questions + no new raw
   files. Our iterative loops (`deep-research`, multi-round ingest) stop on
   judgment; adopt gap-diminishing + overlap thresholds as convergence
   criteria there.
2. **Trajectory persisted, not just the distillation.** `outputs/`
   (`research_state.json` full trajectory + `research_summary.md` +
   `rounds/` per-round audit reports) alongside `raw/` (immutable sources)
   and `wiki/` (compiled). Our `findings.md` keeps only the consolidation —
   adopt per-round audit sections (or a `rounds/` dir) so a future reader can
   see what was asked, what failed, and what each round closed. Same lesson
   as SeedX's run-log, independently arrived at.
3. **User-question inbox with highest priority.** `outputs/user_questions.md`
   (`## Pending`) is picked up next round above agent-generated questions —
   steering without interrupting. Adopt as convention for long-running
   execution/research: a `questions.md` inbox the orchestrator checks per
   wave, rather than stop-the-world gates for every steer.
4. **`--background` to skip basics.** Declared prior knowledge prunes the
   question tree (their FAQ fix for "not deep enough": topic too broad or
   basics un-skipped). Adopt in `brainstorming` interviews: accept a
   background statement up front to collapse rounds 1–N.
5. **Decomposable skills, standalone-usable.** `/wiki compile`, `/wiki
   compile --files`, `/wiki check` run independently — validates our
   authoring-vs-lint satellite split (knowledge-base maintain half vs
   `okf-lint.mjs`).
6. **Pause/resume = rerun same command; context check + `/compact` at 50%.**
   Both match our resume-from-artifacts discipline and compaction advisory
   (ADR-0027..0029) — validation, no change.

**Explicitly NOT adopted:** the fully autonomous web→wiki loop (cost lesson
from the video entry: ~100× build, ~21×/query — our propose+confirm and
human gates stand); agent-browser dependency (our Context7/Exa/fetch stack
stands, same fallback shape they use with plain WebSearch).

**Traceability:** no new requirement — process refinements for existing
skills (`deep-research` stop criteria, `brainstorming` background intake,
per-round audit trail, questions-inbox convention). Slice as follow-up
skill deltas after R6–R13 gate; do not widen this change.

## Ablation: MMR-vs-RRF over the eval fixtures (2026-10-20, TICKET-13)

**Scope:** the deterministic `path · symbol · scope` chunk prefix (the cheap
half of contextual retrieval) plus an MMR-vs-RRF ablation that MEASURES the
shipped ranker without changing it. LLM summaries and embedding-provider
changes are out of scope; a ranker change on winning numbers is a NEW
ticket, never smuggled in here.

### Prefix format (recorded for the commit)

`<path> · <symbol> · <scope>` joined by `" · "` (U+00B7 middle dot, the
`prefixSeparator` const). Chunk embed scope is `L<start>-L<end>` (1-based
inclusive); symbol embed scope is the bare `L` marker (the symbol's own
identity, not a line claim). Empty fields are omitted (a line-window chunk
with no enclosing function-like symbol → `path · L<start>-L<end>`). The
prefix is PURE over (path, symbol, line span) — same input, same prefix, no
timestamps — so a re-embed of unchanged content reproduces the same vectors.
Applied to the EMBEDDING input only; the stored (display) chunk text and the
schema are unchanged. Re-index after landing restores raw vectors (no
migration, no backfill).

### Eval fixture (deterministic, in-tree)

4 docs, 2 code-name-collision queries (hand-labeled relevance, no LLM):

| doc | relevant Q0 | relevant Q1 | sem rank Q0/Q1 | fts rank Q0/Q1 |
|---|---|---|---|---|
| `api/server.go:handler` | ✓ | | 0 / 3 | 1 / 1 |
| `core/service.go:handler` | | | 1 / 0 | 0 / 0 |
| `api/server.go:apiOnly` | | | 2 / 2 | 2 / 2 |
| `core/service.go:coreOnly` | | | 3 / 1 | 3 / 3 |

Q0 = "handler api handler" (want `api/server.go:handler`); Q1 = "handler
core handler" (want `core/service.go:handler`). The cross-file twin
(`api/server.go:handler`) is the false-positive on Q1 — it carries the
colliding name but is relevant to *neither* query. On Q1 the target
(`core/service.go:handler`) tops the FTS leg (FTS-#0) and the semantic leg
(semantic-#0), while the twin sits at semantic-#3; the collision the
prefix targets is that a *raw* (unprefixed) lexical leg cannot tell the
two same-named handlers apart.

### Numbers (reproducible: `cd mnemonic && go test ./internal/codeindex/... -run TestAblation_MMRvsRRF -v`)

| ranker | hit@1 | hit@2 | top-1 Q0 | top-1 Q1 |
|---|---|---|---|---|
| **RRF (k=60)** — shipped, `hybrid.Rank` (rank.go) | 1/2 | 1/2 | `api/server.go:handler` | `core/service.go:handler` |
| **MMR (λ=0.5)** | 1/2 | 1/2 | `api/server.go:handler` | `core/service.go:handler` |

**Verdict: TIED (1/2 each on hit@1 and hit@2).** Neither ranker is perfect
on this fixture; both win Q0 and lose the same single query (Q1). On Q1 the
cross-file twin (`api/server.go:handler`) is the false-positive: RRF credits
its semantic-leg position over the target's FTS-leg position — twin
`1/(60+3)+1/(60+0)` = 0.0328 vs target `1/(60+0)+1/(60+3)` = 0.0319, a 0.0009
delta — so RRF cannot disambiguate a name collision when the false-positive
tops one leg and the target tops the other. MMR's diversity term does not
rescue it either: the twin's near-duplicate penalty (0.5·1.0) exactly cancels
its relevance edge, so MMR keeps the target at #1 / the twin at #2 (tie
broken by id) and loses the same query. **RRF is retained — the ranker is
measured, never changed here.** A ranker swap (e.g. to MMR, or to a
prefix-aware RRF that discounts the cross-file twin) is a NEW ticket.

**Honest note on the prefix's value:** on the hash-embedder (token-overlap)
model the prefix's disambiguation edge is *structural* — the prefixed text of
the target contains its own path, the twin's does not — which is the property
a real embedder (ONNX nomic-embed-code) consumes. The ablation above measures
the RANKER, not the prefix; the prefix's retrieval benefit is asserted
separately by `TestAblation_PrefixedBeatsRawOnCodeNameQueries` (the prefixed
top-1 for a code-name query is a code-name chunk, and the prefixed text is a
superset of the raw text carrying the path disambiguator). Re-running the
ablation with the ONNX embedder is the upgrade path (no model is wired in CI
today).

### Traceability

R13 ↔ Task 14 ↔ @step-14 (G15). Prefix change stales existing vectors —
re-index after landing (no migration, no backfill). Frozen TICKET-01..05
untouched. `rank.go` diff is EMPTY (the ablation RUNS `hybrid.Rank`
read-only).
