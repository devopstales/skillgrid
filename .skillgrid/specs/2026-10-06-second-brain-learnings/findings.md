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

**Sources:** `github.com/azaylamba/repocodex` (README, `docs/agents.md`, `docs/how-it-works.md`, `docs/memory.md` OKF v0.2 consumer guide) + live usage in `/data/git/AI/test-project` (`.repocodex.toml`: `engine_version 0.0.2`, `posture = "shadow"`, no `.context/` bundle yet — adoption playground). Docs + config only; no code executed.

**What it is:** Git-native "why next to code" with a pin check. Concepts (`TechnicalDecision`/`InvariantContract`/`BusinessWorkflow`/`GuardrailDecision`, one per why, orthogonal types) live in an OKF bundle at `.context/` beside the code. Each pins live text via `verification.anchors` (`path` + `all_of` terms, optional `near`) and checkable `claims` literals. CLI over ripgrep+git: `repocodex context <paths>` (retrieve why before edit, file-keyed via generated reverse index at `.repocodex/reverse-index.md`) + `repocodex validate --diff` (`LIVE / WEAK / WRITE + skipped_memory / REANCHOR / RECONCILE+DRIFT / CLAIM_BROKEN`). Pre-commit hook + CI + agent skills enforce the loop; `memory-exempt` is the governed hotfix override. Python 3.11+, requires `rg` on PATH.

**Core insight for our problem:** freshness as content-based attestation, not time-based expiry. An ADR is stale when its anchors break, not 7 days after writing — obsoletes the TTL mindset behind D17 (ingest rows expiring) and the re-crawl instinct. Don't crawl for learnings after the fact; record at the point of change and prove liveness continuously.

### Transferable mechanisms (ranked; ALL follow-up — current 5 tickets frozen)

1. **Anchor liveness for ADRs.** Optional `verification:` frontmatter (anchors + claims) on `artifacts/04-adr-*.md` + a check beside the drift guards re-running anchors (`LIVE/DRIFT/CLAIM_BROKEN`). New third guard: spec-zone drift, structure drift, *knowledge liveness*. Land WARNING-only first.
2. **File-path-keyed why retrieval.** Deterministic "which decisions pin these files" lookup (reverse-index shape) next to `ctx_query`/`code_search`: ingest anchored concepts as observations with file-path metadata. Fixes the probabilistic-miss failure mode (same one pi-hermes-memory's standing-instructions names).
3. **Knowledge-coverage ratchet (the structural fix for "how does it learn").** `WRITE` + `skipped_memory` when a substantive edit touches an uncovered file; turn cannot finish until a pinning concept lands in the same change. Ours already owns the cousins (spec zone, zone guard, ship drift check = anticipated-files→diff); this is the reverse (diff-paths→knowledge-coverage). Land in `qa`/pre-commit as WARNING first.
4. **Why-change protocol.** Intent change requires `supersedes` + `rationale` in the same change — completes our ADR IRON RULE + bitemporal chain (ADR-0011); adopt the `rationale` requirement explicitly.
5. **Unanchored pages are still knowledge.** No-`verification` concepts load, just don't arm the pin check. Vocabulary for reports/retrospectives: process learnings first-class without anchors; conventions carry pins. Maps onto `knowledge_class` (episodic vs convention); legitimizes `reflect` → `06-research-findings.md` as designed.
6. **Shadow posture for every new gate.** Every liveness/coverage check lands as `qa` WARNING / `WINDOWS.md` entry before ever blocking (matches fail-open; mirrors test-project `posture = "shadow"`). Borrow `memory-exempt` shape for hotfixes (our fast-track waiver is the same; add one-command repair flow).
7. **Git-native journal + paired commits.** Append-only decision journal (their `.context/log.md` grouped by date) + "concept + reverse index ship in one commit, desync fails" — extend our `[skillgrid-context]` commit discipline to concept+index pairs.

### Explicitly NOT borrowed

- Storage layout (`.context/` OKF bundle) — our SQLite SoT (ADR-0012) and `.skillgrid/` zones stand. Mechanisms only.
- Distribution already matches (their MCP + installed skills ≈ our Hub Content via `install`); imitate their skill copy "follow that skill; do not invent a substitute".

### Proposed shape (recap, for the follow-up change)

- Keep init ingest (bootstrap) + fix D17/D18 (`ExpiresAt` far-future, stable `Owner`).
- Add anchor liveness (1) + file-path retrieval (2): prove freshness, find decisions deterministically.
- Add coverage ratchet (3) in shadow mode: learn at write time, no crawler.
- Reports stay unanchored knowledge via `reflect` (5).
