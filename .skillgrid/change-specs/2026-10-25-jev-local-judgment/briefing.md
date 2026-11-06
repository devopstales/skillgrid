# Local Jev Judgment Model — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-10-25)

**Tier:** T2
**Classification:** standard (new Go seam + config block + ADR supersession; no new third-party dependency; no behavior change to existing seams)
**Change class (fast-track check):** not fast-track — adds a decision-model HTTP seam, a config block, and supersedes ADR-0035's default model. A new HTTP shape is a lock-in decision (ADR-0023 precedent), so it lands as an ADR + seam, not a fast-track edit.

## Problem

Skillgrid's memory engine (mnemonic) and the planned CTX context-orchestrator both need a **judgment model** — a small, fast, structured-output model that makes keep / truncate / re-rank / gate decisions on context units. Today there are two problems:

1. **The seam does not exist.** The shipped LLM path (ADR-0023) is an OpenAI-compatible `POST {base}/chat/completions` client (`mnemonic/internal/llm/client.go`) with four seams (Ask / Dedup / Extraction / Dream). It returns *prose*. There is **no client that speaks the decision-model shape** `POST {base}/v1/systemone`, which returns *typed answers with probabilities and confidence*. ADR-0035 *decided* the local Ollama judgment model (`tev1:0.8b` at `/v1/systemone`), but the `DecisionClient` it points at has not been built. The CTX judgment layer (context-orchestrator-plan.md Phase 12) has no implementation to call.

2. **The default model is the weakest local option.** ADR-0035 locks `tev1:0.8b`. The Ollama-published eval (Bespoke Labs' 13 public datasets, 3,880 decisions) puts `tev1:0.8b` at **63.5%** and `tev1:4b` at **73.3%**, against hosted **Jev 1.13 at 76.0%**. The 0.8B is ~10 points below the hosted model it replaces; the 4B is within 3 points. For a local-first tool, 4.4GB on the same machine that already runs the embedder is a near-hosted-quality judgment model for free. The ADR's default is the inverse of what the data supports.

The broader context: the Jev launch wave (2026-09) popularized "System One" decision models. The eight shipped Jev tools (Toolgate, Abide, Jev Browser, Jev Ble, Jev Steer or Q, Compact Advisor, Quicksilver, Jev SEO) all assume a **hosted** endpoint (TypeSafe / OpenRouter / Vercel AI Gateway). None use local Ollama. The local-first gap is the opportunity: port the decision logic to a local `DecisionClient` rather than depend on a hosted endpoint, and apply the decision model to the **hybrid source of truth** (ADR-0048) — where the retrieval layer spans both the live tier (SQLite observations) and the durable tier (the `.skillgrid/` OKF markdown bundle, indexed by SQLite).

## Goal

Introduce a local-first **decision-model seam** (`DecisionClient` over `POST {base}/v1/systemone`) that:

1. Gives the CTX judgment layer (Phase 12) and the fused `mem_search` retrieval path (ADR-0047) a fast, cheap, structured-output judgment model that runs on local Ollama with **zero network dependency** on the judgment path.
2. Supersedes ADR-0035's `tev1:0.8b` default with `tev1` (4B) as the default and `tev1:0.8b` as the tight-memory fallback, on the strength of the published accuracy gap (73.3% vs 63.5%).
3. Composes with the hybrid SoT (ADR-0048): the re-ranker operates on the *fused* result set (live observations + durable ADRs/glossary/constraints), with the bundle file as the cited source.

This is the *seam + config + ADR supersession*. It is **not** the full CTX Phase 12 build (that is the CTX change's scope), and it is **not** a port of all eight Jev tools (those are follow-up changes that consume this seam). This change delivers the one new HTTP shape and the one config block everything else needs.

## Success Criteria (measurable)

1. A `DecisionClient` exists in `mnemonic/internal/llm/` (or a sibling `mnemonic/internal/judge/`) that speaks `POST {base}/v1/systemone`, returning typed answers for `choice`, `noul`, and `score` question types, each with the fields the Ollama API returns (`choice`/`probabilities`/`confidence`; `noul`; `score`/`legend`/`probabilities`/`confidence`). It is the only place in the codebase that constructs a `/v1/systemone` request.
2. The client is **stdlib-only** (`net/http` + JSON). No new third-party dependency (no ADR-0008-style dependency ADR required beyond this provider-shape lock).
3. It is **fail-open**: on transport error, timeout, or a non-2xx, it returns a sentinel error the caller treats as "no judgment" (deterministic floor runs). It never blocks the agent on a judgment error.
4. It enforces the Ollama hard limits **client-side** before the request leaves: state + questions must fit the model's effective context (~2,000 tokens — configurable `max_state_tokens`), the request body must be ≤ 64 KiB, choice/score questions must have 2–24 options, and a single call may carry ≤ 64 questions. Violations return a typed error, not a silent truncation.
5. A `mnemonic.judgment.{enabled, base_url, model, max_state_tokens, timeout}` config block exists in the same indexing-YAML stack as `mnemonic.llm.*` (home `~/.skillgrid/config.d/indexing.yaml` per-key fallback). Defaults: `enabled: false`, `base_url: http://127.0.0.1:11434` (reusing the existing ollama base seam from `skillgrid-cli/internal/install`), `model: tev1` (the 4B).
6. ADR-0035 is **superseded** by a new ADR (0050) that re-locks the default model to `tev1` (4B) with `tev1:0.8b` as the fallback, citing the Ollama eval (73.3% vs 63.5% vs Jev 1.13's 76.0%). The `/v1/systemone` endpoint, fail-open-to-pre-filter, fingerprint-cache, truncate-only, and threshold-0.22 invariants are carried forward unchanged.
7. The install provider step (ADR-0023) is extended to ensure `tev1` (4B) on the Local path (pull it alongside `llama3.2:3b` + `nomic-embed-text`), with `--skip-provider` / `--provider=external` escapes unchanged. Re-run is ensure-not-refresh.
8. The seam is **wired to at least one consumer** so it is not dead code: the re-ranker hook on the fused `mem_search` path (ADR-0047) calls `DecisionClient` to re-rank the fused result set when `mnemonic.judgment.enabled`. When disabled or failing, `mem_search` returns the existing fused order unchanged (fail-open).
9. The `confidence` field is **documented as probability concentration, not P(correct)** (per the Ollama README), and the re-ranker gates on the `score`/`noul` value, not on `confidence` alone. This corrects the hosted-Jev "confidence is the load-bearing dial" framing for the local model, whose calibration is untested.

## Falsifiable Requirements

- **R1 (DecisionClient seam).** `mnemonic/internal/judge/client.go` (or `mnemonic/internal/llm/decision.go`) defines `DecisionClient` with `SystemOne(ctx, state string|map, questions map[string]Question) (Answers, error)`. `Question` carries `Type` (`choice`|`noul`|`score`), `Instructions`, and `Criteria` (object for choice/noul, `[]string` for score). `Answers` is `map[string]Answer` where `Answer` carries the type-specific fields. The request is `POST {BaseURL}/v1/systemone` with body `{model, state, questions}`.
  - *True when:* the file exists, the method signature matches, the request path and body shape match the Ollama API, and `grep -rn "systemone" mnemonic/` shows the seam as the sole constructor.
  - *False when:* the seam reuses `/chat/completions`, returns prose, or there is more than one `/v1/systemone` request site.
- **R2 (stdlib-only).** The `DecisionClient` imports only `net/http`, `encoding/json`, `context`, `time`, `strings`, `fmt`, and the standard library. No third-party module is added to `mnemonic/go.mod` for this change.
  - *True when:* `go mod tidy` after the change adds no new requirement.
  - *False when:* a third-party HTTP or JSON dependency is added.
- **R3 (fail-open).** A `SystemOne` call that times out, hits a connection error, or receives a non-2xx returns `(Answers{}, <typed error>)` where the error is a distinct `ErrNoJudgment` (or `errors.Is`-matchable) that callers branch on to run the deterministic floor. A unit test with a dead `base_url` proves the call returns `ErrNoJudgment` within the configured timeout, not a hang.
  - *True when:* the typed error exists, the dead-URL test passes, and no caller treats a judgment error as fatal.
  - *False when:* a judgment error propagates as a fatal error to the agent, or the dead-URL test hangs past the timeout.
- **R4 (client-side limit enforcement).** Before issuing the request, the client validates: (a) serialized body ≤ 65,536 bytes; (b) choice/score `Criteria` has 2–24 entries; (c) ≤ 64 questions in one call. Each violation returns a distinct typed error (`ErrBodyTooLarge`, `ErrOptionCount`, `ErrTooManyQuestions`). A `max_state_tokens` config (default 2000) bounds the state; a state over the bound returns `ErrStateTooLarge` (or is pre-truncated by the caller, documented).
  - *True when:* all four limits are checked with distinct errors and unit tests for each.
  - *False when:* any limit is checked server-side only, or a violation silently truncates instead of erroring.
- **R5 (config block).** `config.LLMConfig` (or a sibling `config.JudgmentConfig`) gains `JudgmentConfig{Enabled bool, BaseURL, Model, MaxStateTokens int, Timeout time.Duration}`. The home `config.d/indexing.yaml` loader reads `mnemonic.judgment.*` with per-key fallback, defaults `enabled: false`, `model: tev1`. `mnemonic.judgment` is **distinct from** `mnemonic.llm` (two different HTTP shapes, two config blocks).
  - *True when:* the struct exists, the loader reads it, the defaults are correct, and `mnemonic.judgment` and `mnemonic.llm` are independent keys.
  - *False when:* the judgment config is folded into `mnemonic.llm`, the defaults are wrong, or the loader does not fall back per-key.
- **R6 (ADR supersession).** `.skillgrid/knowledge/decisions/04-adr-0050-local-judgment-model-tev1-4b.md` records: Context (ADR-0035's 0.8b default, the Ollama eval table, the gap), Decision (default `tev1` 4B, fallback `tev1:0.8b`; `/v1/systemone` + fail-open + fingerprint-cache + truncate-only + threshold 0.22 carried forward), Consequences (accuracy gain, 4.4GB pull cost on Local install, the calibration caveat). It has `supersedes: 04-adr-0035-local-ollama-judgment-model`. A pointer row for 0050 is added to `ASSUMPTIONS.md` § LOCKED; the 0035 row is marked superseded.
  - *True when:* the file exists with correct frontmatter, the supersession is declared, and the index rows are updated (0050 in force, 0035 superseded).
  - *False when:* 0035 is edited in place (IRON RULE violation), the supersession is not declared, or the index is not updated.
- **R7 (install provider extension).** The Local provider step in `skillgrid-cli/internal/install` adds `tev1` to the pull set (alongside `llama3.2:3b` + `nomic-embed-text`). The pull is ensure-not-refresh on re-run. `--skip-provider` and `--provider=external` skip it. A test asserts `ollama pull tev1` is invoked on the Local happy path and not on the skip/external paths.
  - *True when:* the pull is in the Local path, absent on skip/external, and the test passes.
  - *False when:* `tev1` is not pulled on Local, or is pulled on skip/external.
- **R8 (consumer wiring: fused re-ranker).** The fused `mem_search` path (ADR-0047) gains an optional re-rank step: when `mnemonic.judgment.enabled`, the fused result set (observations leg + web_cache leg + durable-bundle leg) is passed to `DecisionClient.SystemOne` as a `score` question with policy criteria (e.g. `authoritative | current | stale | off-topic`), and the results are re-ordered by the returned `score`. When disabled or `ErrNoJudgment`, the existing fused order is returned unchanged. The re-rank is **additive** — it never drops a result, only re-orders (consistent with the truncate-only invariant).
  - *True when:* the re-rank step exists, is gated on `enabled`, re-orders (not filters) on success, and falls back to the existing order on disable/error.
  - *False when:* the re-rank filters results, runs when disabled, or changes the fused order on error.
- **R9 (confidence semantics documented).** The `Answer` type's doc comment and the ADR both state that `confidence` is probability concentration, not P(correct), and that the re-ranker gates on the `score`/`noul` value. A code comment at the re-rank site states the same.
  - *True when:* all three sites (type doc, ADR, call site) carry the note.
  - *False when:* any site treats `confidence` as P(correct) or the note is absent.

## Trust Boundaries

- **No new network surface for the agent.** The `DecisionClient` talks only to the local Ollama daemon at `127.0.0.1:11434` (the same trust posture as the existing Ollama embedder and chat Completer). It is loopback-only by default; a non-loopback `base_url` requires an explicit config and logs a warning (mirrors the CTX proxy's loopback posture).
- **No new filesystem write path.** The judgment path reads the fused result set and returns a re-ordering. It writes nothing except the optional judgment-decision log (a `ctx_*`-style table or JSONL, additive, behind `mnemonic.judgment.log`).
- **No new Mnemonic tool-contract change.** `mem_search`'s return shape is unchanged; the re-rank re-orders within the existing shape. The `DecisionClient` is an internal seam, not a new MCP tool.
- **Calibration caveat (carried in the ADR).** `tev1`'s calibration and prompt-injection resistance are not fully tested (per the Ollama README). User-controlled content is not sent into `state` without the deterministic pre-filter running first (the CTX pre-filter, or for `mem_search`, the existing lexical/vector scoring that bounds what reaches the judgment). The re-ranker is advisory; it never deletes.

## Scope (files)

- Create: `mnemonic/internal/judge/client.go` (the `DecisionClient` seam) — or `mnemonic/internal/llm/decision.go` if the `llm` package is preferred.
- Create: `mnemonic/internal/judge/client_test.go` (fail-open, limit enforcement, response parsing for all three question types).
- Create: `.skillgrid/knowledge/decisions/04-adr-0050-local-judgment-model-tev1-4b.md`.
- Modify: `mnemonic/internal/mnemonic/config/load.go` (add `JudgmentConfig` to the config struct + loader).
- Modify: `mnemonic/internal/service/attach_llm.go` (or a sibling `attach_judge.go`) — wire the `DecisionClient` into the service when `mnemonic.judgment.enabled`.
- Modify: the fused `mem_search` path (ADR-0047 implementation site) — add the optional re-rank step.
- Modify: `skillgrid-cli/internal/install/install.go` + `provider_test.go` — add `tev1` to the Local pull set.
- Modify: `mnemonic/go.mod` only if a test helper requires it (expected: no change).
- Modify: `.skillgrid/knowledge/ASSUMPTIONS.md` § LOCKED (add 0050 row, mark 0035 superseded).
- Modify: `.skillgrid/knowledge/index.md` (decisions/ catalog row for 0050).
- Modify: `AGENTS.md` only if the Rules section references the judgment model (expected: no change; ADR-0035's pointer is in ASSUMPTIONS, not AGENTS).

## Open Questions

1. **Package location:** `mnemonic/internal/judge/` (new package) vs `mnemonic/internal/llm/decision.go` (sibling file in the existing package). Leaning `judge/` — it is a distinct HTTP shape with distinct config, and keeping it separate from the chat `Completer` makes the ADR-0023 "one client attaches all seams" boundary honest (that boundary is for the *chat* seams; the decision seam is a different contract). Decide at blueprint time.
2. **`max_state_tokens` default.** 2000 is the Ollama README's effective-context figure ("Tev1 runs with a context of about 2,000 tokens; the longest training example is about 1,500 tokens"). Whether the default is 1500 (safer) or 2000 (the documented figure) is a tuning call. Leaning 1500 for the default with 2000 as the documented ceiling, to stay under the longest training example.
3. **Re-rank criteria set.** The `score` criteria for the fused re-ranker (`authoritative | current | stale | off-topic` is a candidate) is a product decision. The criteria are the *executable policy* (the video-4 finding) — they are the most important knob. This should be a `config.d/ctx.yaml` (or `indexing.yaml`) key so it is human-editable without a code change, and benchmarked over time (the video-3 "define the criteria yourself, benchmark changes" discipline). Parked to the blueprint.
4. **`tev1:4b` pull on install.** 4.4GB on first Local install is a real cost (ADR-0023 already flags the multi-GB pull concern for `llama3.2:3b`). Whether `tev1` is pulled by default on `--yes` Local or only when `mnemonic.judgment.enabled` is set to true is an install-posture call. Leaning: pull only when judgment is enabled, to keep the default install lean. Decide at blueprint time.
