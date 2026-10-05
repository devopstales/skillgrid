# Briefing — Local Ollama model catalog

> **STATUS:** `draft` (2026-10-03) — Model list taken from the user. Library tags checked in `findings.md`. **Embedder locked (round 2): `embeddinggemma:300m`.** `glm:vision-tools` is not an Ollama tag. Live chat model is still an assumption. Do not write `blueprint.md` until that and the vision tag are confirmed.

**Topic:** 2026-10-03-local-ollama-models
**Date:** 2026-10-03
**Classification:** standard (T2)
**Build shape:** Smallest usable whole
**Queued behind:** `2026-10-02-mnemonic-llm-provider` (serial). That change owns `ensureLocal` and currently pulls `llama3.2:3b` + `nomic-embed-text`. This spec replaces that pull list. Do not take `current_change` until that change ships or parks.
**Findings:** `findings.md`

## Problem / Intent

Local install today pulls two models and wires them as the only chat model and the only embedder. The operator wants a fixed catalog of small local models, each used through the API that model actually speaks (chat, embed, or decision). The catalog does not belong in the shared-LLM-client change or in the workflow-updates change.

## Purpose & Success Criteria

- **Purpose:** `skillgrid install` with the local provider ensures Ollama 0.35.1 or newer, pulls this catalog if missing, smoke-tests each model once, and records which tag fills chat, embed, and decision.
- **Success criteria:** Re-running install does not pull a model that `GET /api/tags` already lists. A stubbed test covers pull, skip, version-too-old, and one smoke call per kind. No live download in unit tests.
- **Out of scope:** Cloud / external provider wiring (stays in `2026-10-02-mnemonic-llm-provider`). Attaching tev1 or clef-flash to `AskLLM` / `DedupLLM` / `ExtractionLLM` (those seams speak chat completions; these models speak `/v1/systemone`). A model picker UI. Quantization choices beyond the tags named here. Pulling `llama3.2:3b` or `nomic-embed-text` once this catalog replaces them.

## Context

Fits existing patterns: **yes-with-notes**.

`skillgrid-cli/internal/install/provider.go` (on the llm-provider change, not yet the integration branch) already installs Ollama if missing, starts it, probes `http://localhost:11434/api/tags`, pulls only when absent, and merges `~/.skillgrid/config.d/indexing.yaml` (`mnemonic.llm.model`, `mnemonic.embedder.model`). This change edits that list and adds a version check plus a per-kind smoke call. It does not add a second HTTP client stack for chat: chat smoke uses the existing OpenAI-compatible `/v1/chat/completions` shape. Decision smoke is a separate small POST to `/v1/systemone`.

Locked: Go 1.22+, no new dependencies without an ADR (`net/http` + stdlib JSON), serial one-change, spec before code, fail-open install (a pull failure warns and install still succeeds — same posture as `ensureLocal` today).

## Approaches Considered

- **Chosen: Extend `ensureLocal` with an explicit catalog.** One table of tag, kind, and smoke request. Install pulls missing tags, then runs the matching smoke. Home YAML records chat, embed, and the extra tags. Same ensure/idempotent behavior as today.
- **Rejected: A shell script of the user’s `ollama pull` / `ollama run` lines.** `ollama run` opens a REPL, so it cannot be an install step. A script beside install would fork the provider setup the llm-provider change just added.
- **Rejected: One chat model only, ignore decision and embed tags.** Drops `tev1:0.8b`, `clef-flash`, and `embeddinggemma:300m`, which are different APIs and the reason this is its own spec.

## Requirements

1. **Version floor:** Local provider setup refuses the catalog when `ollama --version` is below 0.35.1.
   - **Current:** Install does not read the Ollama version. EmbeddingGemma needs 0.11.10, tev1 needs 0.35, clef-flash needs 0.35.1.
   - **Target:** Parse `ollama --version`. Below 0.35.1 → warn with the required version, skip pulls and smokes, do not write model names into home YAML. At or above → continue.
   - **Acceptance:** Stubbed version strings `0.35.0` and `0.35.1`. The first skips pulls. The second proceeds.
   - **Acceptance scenario:** pending `acceptance.feature`

2. **Pull the catalog, skip what is already present:** Install runs `ollama pull <tag>` only for tags missing from `GET /api/tags`.
   - **Current:** Pulls `llama3.2:3b` and `nomic-embed-text` only.
   - **Target:** Pull, in this order: `qwen2.5:1.5b`, `tev1:0.8b`, `embeddinggemma:300m`, `gemma2:2b`, `clef-flash`, `llama3.2:1b`. Do not pull `llama3.2:3b` or `nomic-embed-text`. `ollama run` is not invoked. A tag already listed (exact name, or name with an `:latest` alias of the same model) is not pulled again.
   - **Acceptance:** Stubbed probe that already contains `gemma2:2b` pulls the other five and not `gemma2:2b`. Empty probe pulls all six. `glm:vision-tools` is not pulled.
   - **Acceptance scenario:** pending `acceptance.feature`

3. **Smoke each kind on its own API:** After a successful pull (or when the tag was already present), one non-interactive request proves the model answers. Failure warns and does not fail install.
   - **Current:** No smoke call. Presence in `/api/tags` is the only check.
   - **Target:** Chat tags (`qwen2.5:1.5b`, `gemma2:2b`, `llama3.2:1b`) → `POST /v1/chat/completions` with one user message; pass when the first choice content is non-empty. Embed tag (`embeddinggemma:300m`) → `POST /api/embed` with one string; pass when the vector length is > 0, and record that length. Decision tags (`tev1:0.8b`, `clef-flash`) → `POST /v1/systemone` with one yes/no question; pass when that answer field is present. Timeout bounded (same order as the llm client timeout, not an open wait).
   - **Acceptance:** Mock HTTP tests: chat 200 with content passes; embed 200 records dimension; systemone 200 passes; HTTP 500 warns and install returns nil.
   - **Acceptance scenario:** pending `acceptance.feature`

4. **Home config names chat and embed:** The merge into `~/.skillgrid/config.d/indexing.yaml` points Mnemonic’s Ollama embedder at `embeddinggemma:300m` and lists the rest of the catalog.
   - **Current:** Install writes `mnemonic.embedder.model` = `nomic-embed-text` and `embedder.provider` = `ollama`. The Ollama embedder default in code is `nomic-embed-code` (`DefaultOllamaModel`). The ONNX default stays `nomic-embed-code` and is not this change. `mnemonic.llm.model` = `llama3.2:3b`.
   - **Target:** `mnemonic.embedder.provider` = `ollama`, `mnemonic.embedder.model` = `embeddinggemma:300m`, `mnemonic.embedder.dimension` = the length recorded in req 3. The Ollama embedder’s empty-model default becomes `embeddinggemma:300m` so a local Mnemonic uses it without a hand-written model line. `mnemonic.llm.model` stays an assumption (`llama3.2:1b` until the chat question closes). A new list `mnemonic.ollama.models` holds the six tags and their kinds (`chat`, `embed`, `decision`). Unrelated keys in the file stay. Re-run overwrites the embedder model and does not wipe the rest. ONNX / `nomic-embed-code` remains the non-Ollama default.
   - **Acceptance:** Fixture YAML with an unrelated key survives; embedder provider is `ollama` and model is `embeddinggemma:300m`; the list has six entries; an empty Ollama embedder config resolves to `embeddinggemma:300m`.
   - **Acceptance scenario:** pending `acceptance.feature`

5. **Dimension mismatch warns, and the embedder still switches:** An existing index built at a different width must be reindexed. The model line still becomes `embeddinggemma:300m`.
   - **Current:** Switching embed model does not check the indexed dimension. A mismatched dimension silently breaks search.
   - **Target:** Always write `mnemonic.embedder.model` = `embeddinggemma:300m` after a successful embed smoke. If home config or the index metadata already records a dimension and the smoke vector’s length differs, warn with both lengths and that search is stale until reindex. Same length → write the model and the dimension, no reindex warning.
   - **Acceptance:** Fixture dimension 768 vs smoke length 768 writes the model and does not warn. Fixture 768 vs smoke 512 still writes `embeddinggemma:300m` and the warning names both lengths.
   - **Acceptance scenario:** pending `acceptance.feature`

6. **Unresolved vision tag is reported, not invented:** `glm:vision-tools` is kept out of the pull list until it is a real Ollama tag.
   - **Current:** The string is not a library tag (see `findings.md`). `glm-ocr` is a vision+tools GLM model and is not the same name.
   - **Target:** Install logs one warning that `glm:vision-tools` was requested and has no pull tag. It does not pull `glm-ocr` or any other substitute.
   - **Acceptance:** Test asserts the run stub never receives `glm-ocr` or `glm:vision-tools`.
   - **Acceptance scenario:** pending `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:** `install.ensureLocal` and its stubbed tests. A catalog value (tag, kind) next to the existing `localChatModel` / `localEmbedModel` constants, which this change replaces. Version parse is a pure function. Smoke calls take a base URL so tests use `httptest`.
- **Interfaces:** Catalog entries are `{tag, kind}` with kind `chat | embed | decision`. Smoke returns `{ok bool, dimension int, err}`. Home merge gains the models list and an optional dimension. No new MCP tool. No new module for `/v1/systemone` beyond the smoke POST.
- **Data flow:** Probe tags → pull missing → smoke → merge YAML. Disk artifacts and the Ollama daemon remain the source of truth for “is it installed”.
- **Error handling:** Version too old, pull failure, and smoke failure warn and return nil from the provider step, matching today’s non-fatal install. A failed embed smoke does not write `embeddinggemma:300m`. A successful smoke with a dimension mismatch still writes it and warns that a reindex is required.
- **Dependencies:** None. Stdlib HTTP and the `yaml.v3` the install package already imports.

## Testing Decisions

- **What makes a good test:** Stub `LookPath`, `Run`, and `Probe` the way `provider` tests already do. Point smoke at `httptest.Server`. Never call the network or the real `ollama` binary.
- **Modules to test:** Version compare, pull filtering, smoke dispatch by kind, YAML merge, dimension gate.
- **Prior art:** Install provider tests on the llm-provider change (`ensureLocal`, dry-run, skip-if-present).
- **Edge cases:** Tag present as `llama3.2:1b` vs `llama3.2:latest` (do not treat `:latest` as the 1B tag). Ollama binary missing still follows the existing install-then-pull path before the version check. Dry-run prints the six pulls and writes nothing.

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: None.
- `.skillgrid/ASSUMPTIONS.md`: None in this draft. Embedding model is locked here as `embeddinggemma:300m` for the Ollama provider. A dimension change that forces a reindex is an ADR candidate (ADR-0024+), written when the smoked length is known.
- `.skillgrid/ARCHITECTURE.md`: None (file is not in this repo yet; the component is still the existing Ollama embedder).

## Clarity Report

Interview not started. The model list is the user’s. Role defaults are assumptions.

| Dimension           | Score | Min  | Status | Notes |
|---------------------|-------|------|--------|-------|
| Goal Clarity        | —     | 0.75 | ⚠      | Embedder is `embeddinggemma:300m`. Which tag is the live chat model is still assumed. |
| Boundary Clarity    | —     | 0.70 | ⚠      | Decision models are pulled and smoked, not attached to LLM seams. Vision tag excluded. |
| Constraint Clarity  | —     | 0.65 | ⚠      | Ollama ≥ 0.35.1. Queued behind llm-provider. Embedder model locked; dimension mismatch warns and still switches. |
| Acceptance Criteria | —     | 0.70 | ⚠      | Checks above are draft until the two open questions close. |
| **Clarity**         | —     | ≤0.20| ⚠      | Do not blueprint yet. |

**Interview log:**

| Round | Question summary | Decision locked |
|-------|------------------|-----------------|
| 0 | Where does this list live? | Its own spec, `2026-10-03-local-ollama-models`. Not workflow-updates. Not folded into the in-flight llm-provider change. |
| 1 | Are the names real Ollama tags? | Six are. `glm:vision-tools` is not. See `findings.md`. |
| 2 | Which model embeds for Mnemonic? | **`embeddinggemma:300m`.** Ollama provider and local install write that tag. ONNX `nomic-embed-code` stays the non-Ollama default. A width mismatch warns and still switches; search is stale until reindex. |

## Open Questions & Assumptions

- **Question:** Live chat model written to `mnemonic.llm.model` — `llama3.2:1b`, `gemma2:2b`, or `qwen2.5:1.5b`? The other two would still be pulled.
- **Assumption:** `llama3.2:1b` is the live chat model (the 1B sibling of today’s `llama3.2:3b`). `gemma2:2b` and `qwen2.5:1.5b` are pulled and listed, not selected.
- **Locked (round 2):** Mnemonic’s Ollama embedder is `embeddinggemma:300m`. It replaces `nomic-embed-text` on install and `nomic-embed-code` as `DefaultOllamaModel`. It does not replace the ONNX default.
- **Question:** Is `glm:vision-tools` a local name you will create, a typo, or should the spec drop it? `glm-ocr` is the closest official vision+tools GLM tag and will not be pulled unless you say so.
- **Assumption:** `clef-flash` (9B, multi-GB) and `tev1:0.8b` are worth the disk and the 0.35.1 floor. They are not routed into ask/dedup/extraction in this change.
- **Assumption:** Serial constraint holds. This folder stays queued behind `2026-10-02-mnemonic-llm-provider`.

## Decisions (ADR)

None yet. Embedder model is locked in this briefing (`embeddinggemma:300m`). An ADR is due only if the smoked vector width differs from the indexed width and forces a reindex.

## Terms

None yet. “Decision model” here means a model called through Ollama `/v1/systemone` (tev1, clef-flash). Do not add it to the glossary until the interview keeps the word.
