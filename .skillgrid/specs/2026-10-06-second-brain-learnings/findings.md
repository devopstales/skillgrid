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
