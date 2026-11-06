## 06-multi-embedder

### Goal

ollama and local embedder providers.

### Out of scope / Non-Goals

- Store pooling, FTS trigram, parallel search, TTL, extraction.

### Definition of Done

This step is done only when:

- [x] All `### Tasks` checkboxes below are `[x]`
- [x] All `@step-06` scenarios in `acceptance.feature` pass
- [x] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [x] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [x] No Global Constraint violated

> Depends on: 01

**Files:**
- `internal/mnemonic/embedder/embedder.go` — MODIFY: ollama + local providers
- `internal/mnemonic/embedder/ollama.go` — CREATE: Ollama embedder
- `internal/mnemonic/embedder/local.go` — CREATE: Local ONNX embedder
- `internal/mnemonic/config/load.go` — MODIFY: ollama/local provider config
- `internal/mnemonic/embedder/embedder_test.go` — MODIFY: new provider tests

**Interfaces:**
- Consumes: existing `embedder.Default()`; ONNX runtime (`onnxer`); Ollama HTTP API
- Produces: `ollama` and `local` embedder providers; config-driven selection via `indexing.yaml`

### Tasks

- [x] 06.1 `[RED]` Ollama embedder calls http://localhost:11434/api/embeddings
  - [x] 06.1.a Write failing test (`TestOllamaEmbedder`): start a mock HTTP server on a test port that mimics the Ollama `/api/embeddings` endpoint; configure the embedder to use that port; call `Embed(ctx, "test text")`; verify it returns a vector of the expected dimensionality; verify the HTTP request body contains the input text
  - [x] 06.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestOllamaEmbedder'` — Expected: FAIL
  - [x] 06.1.c Minimal implementation — create `ollama.go` with an `OllamaEmbedder` struct that POSTs to `http://localhost:11434/api/embeddings` with `{"model": "...", "prompt": "..."}`; parse the `embedding` field from the JSON response; register as provider type `ollama` in `embedder.go`
  - [x] 06.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestOllamaEmbedder'` — Expected: PASS
  - [x] 06.1.e Commit — `feat(mnemonic): add ollama embedder provider`
- [x] 06.2 `[RED]` Local ONNX embedder loads model from ~/.skillgrid/models/
  - [x] 06.2.a Write failing test (`TestLocalONNXEmbedder`): place a test ONNX model in a temp directory; configure the embedder to load from that directory; call `Embed(ctx, "test text")`; verify it returns a vector; verify the model file is found and loaded; verify graceful error when model directory is missing
  - [x] 06.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestLocalONNXEmbedder'` — Expected: FAIL
  - [x] 06.2.c Minimal implementation — create `local.go` with a `LocalONNXEmbedder` struct that loads an ONNX model from `~/.skillgrid/models/` (or a configured path); use `onnxer` to run inference; register as provider type `local` in `embedder.go`; return a descriptive error when the model file is missing
  - [x] 06.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestLocalONNXEmbedder'` — Expected: PASS
  - [x] 06.2.e Commit — `feat(mnemonic): add local ONNX embedder provider`
- [x] 06.3 `[RED]` Config-driven provider selection via indexing.yaml
  - [x] 06.3.a Write failing test (`TestEmbedderConfigDrivenSelection`): write an `indexing.yaml` with `mnemonic.embedder.provider: ollama`; call `embedder.Default()`; verify it returns an Ollama embedder; write `provider: local` and verify it returns a LocalONNX embedder; write `provider: onnx` and verify it returns the existing ONNX embedder; write no provider and verify it returns Null
  - [x] 06.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestEmbedderConfigDrivenSelection'` — Expected: FAIL
  - [x] 06.3.c Minimal implementation — extend `config/load.go` to parse `mnemonic.embedder.provider` from `indexing.yaml`; extend `embedder.go` `Default()` to switch on the provider value and return the appropriate embedder; ensure `onnx`/`external`/`off` values are unchanged
  - [x] 06.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestEmbedderConfigDrivenSelection'` — Expected: PASS
  - [x] 06.3.e Commit — `feat(mnemonic): config-driven embedder provider selection`
- [x] 06.4 `[AFK]` Embedder load failure degrades to Null
  - [x] 06.4.a Write failing test (`TestEmbedderLoadFailureReturnsNull`): configure `provider: ollama` but point to an unreachable port; call `embedder.Default()`; verify it returns a Null embedder (not an error); verify `Embed` on the Null embedder returns zero-vector or empty slice
  - [x] 06.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestEmbedderLoadFailureReturnsNull'` — Expected: FAIL
  - [x] 06.4.c Minimal implementation — in `embedder.Default()`, wrap provider creation in error handling; on failure, log a warning and return `NullEmbedder`; ensure `NullEmbedder.Embed` returns a zero-length vector without error
  - [x] 06.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestEmbedderLoadFailureReturnsNull'` — Expected: PASS
  - [x] 06.4.e Commit — `feat(mnemonic): degrade to Null embedder on load failure`

### Verification

Verdict: `PASS WITH WARNINGS`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/embedder/ -run 'TestOllamaEmbedder\|TestLocalONNXEmbedder\|TestEmbedderConfigDrivenSelection'` | PASS | PASS | all new provider tests green (plus TestOllamaEmbedError, TestLocalONNXEmbedderMissing, TestConfigLoadsNewProviders, TestEmbedderLoadFailureReturnsNull) |
| Acceptance `@step-06` / `@p0` | BDD / mapped unit scenarios | PASS | PASS | 4 test functions map 1:1 to the step's 4 sub-tasks |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/embedder/ -count=1` | PASS | PASS | 18/18 ok (0.94s) |
| Rollback boundary | verify onnx/external/off providers unchanged | PASS | PASS | TestOnnxDefault/TestNullAdapter/TestExternal* green; existing resolveEmbedder call sites untouched |
| Global Constraints | — | held | held | additive only; onnxer pure-Go (no CGO); no tool contract change; no schema rewrite |
| Baseline suites | store/service/memory/route/affected/community/pdg/codeindex `-count=1` | PASS | WARN | service `TestReindexStructuralIsEmbedderFree` fails pre-existing (database is closed; reproduces with step-06 stashed) — see report |

### Commit

When step DoD is met: `feat(mnemonic): ollama and local embedder providers with config-driven selection`

---

