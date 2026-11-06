# Task 06 Report — multi-embedder (ollama + local providers)

## 06.1 — Ollama embedder (`ollama.go`) — `dcbf534`

New `OllamaEmbedder` (type `Ollama`) POSTs to `<BaseURL>/api/embeddings`.
Base URL is configurable (`OllamaConfig.BaseURL`), defaulting to
`http://localhost:11434` so tests can point at a mock server.

Request/response handling (quoted):

```go
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaResponse struct {
	Model     string    `json:"model"`
	Embedding []float32 `json:"embedding"`
}

func (o *Ollama) embed(ctx context.Context, text string) (memory.Vector, error) {
	body, err := json.Marshal(ollamaRequest{Model: o.cfg.Model, Prompt: text})
	if err != nil {
		return memory.Vector{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(o.cfg.BaseURL, "/api/embeddings"), bytes.NewReader(body))
	if err != nil {
		return memory.Vector{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		return memory.Vector{}, fmt.Errorf("ollama embedder: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return memory.Vector{}, fmt.Errorf("ollama embedder: %s/api/embeddings -> HTTP %d: %s", o.cfg.BaseURL, resp.StatusCode, string(raw))
	}
	var out ollamaResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return memory.Vector{}, fmt.Errorf("ollama embedder: decode: %w", err)
	}
	if len(out.Embedding) == 0 {
		return memory.Vector{}, fmt.Errorf("ollama embedder: empty embedding from %s", o.cfg.BaseURL)
	}
	return memory.Vector{Data: out.Embedding}, nil
}
```

Registered as provider type `ollama` in `select.go` (06.3). `Verify(ctx)`
performs a real round-trip probe at selection time (added in 06.4 so an
unreachable Ollama server degrades to Null at load, not at first `Embed`).

## 06.2 — Local ONNX embedder (`local.go`) — `d716f95`

New `LocalONNX` loads an ONNX model from a configurable directory
(`LocalConfig.ModelDir`, default `~/.skillgrid/models/`) as
`<Model>.onnx` (following the `onnx.go` file-name convention).
Model path resolution:

```go
// ModelPath returns the resolved model file path (<ModelDir>/<Model>.onnx).
func (l *LocalONNX) ModelPath() string {
	return filepath.Join(l.cfg.ModelDir, l.cfg.Model+".onnx")
}

func (l *LocalONNX) loadSession() (*onnxruntime.Model, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sess != nil {
		return l.sess, nil
	}
	path := l.ModelPath()
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("local onnx embedder: model file %s not found (run the model download or set mnemonic.embedder.model_dir)", path)
	}
	sess, err := onnxruntime.LoadModelFromFile(path, nil)
	if err != nil {
		if isLibraryMissing(err) {
			return nil, fmt.Errorf("local onnx embedder: %w (%v)", ErrRuntimeUnavailable, err)
		}
		return nil, fmt.Errorf("local onnx embedder: load %s: %w", path, err)
	}
	l.sess = sess
	return sess, nil
}
```

Unlike the `onnx` provider (which degrades to a hash fallback when the model
is absent), the local provider is explicit: a missing model file is a
descriptive load error, and the selector degrades to Null (06.4).
`ErrRuntimeUnavailable` is reported when `libonnxruntime` itself cannot be
dlopen'd (the dlopen error text is matched in `isLibraryMissing`), keeping
the load-path contract (file found/missing) distinct from runtime
availability. No CGO: `onnxer` (pure-Go ONNX Runtime binding via purego,
already a direct dep in go.mod) is reused.

## 06.3 — Config-driven selection (`select.go` + `config/load.go`) — `998f1bf`

`config/load.go`: `EmbedderConfig` gains `ModelDir string` (local-only) with
yaml key `model_dir`, plus the merge (`if section.ModelDir != ""`). The
existing `Provider` field now accepts `ollama` and `local` in addition to
`onnx`/`external`/`off`; unknown values are preserved as text (degraded at
selection time, never a parse failure).

`select.go` — the selector (quoted switch):

```go
func BuildFromConfig(cfg EmbedderConfig) Embedder {
	switch cfg.Provider {
	case ProviderOff, "":
		return NewNull()
	case ProviderOllama:
		return degradeOnError("ollama", func() (Embedder, error) {
			o := NewOllama(OllamaConfig{
				BaseURL:   cfg.BaseURL,
				Model:     cfg.Model,
				Dimension: cfg.Dimension,
			})
			if err := o.Verify(loadCtx); err != nil {
				return nil, err
			}
			return o, nil
		})
	case ProviderLocal:
		return degradeOnError("local", func() (Embedder, error) {
			l := NewLocal(LocalConfig{
				ModelDir:  cfg.ModelDir,
				Model:     cfg.Model,
				Dimension: cfg.Dimension,
			})
			if err := l.Load(loadCtx); err != nil {
				return nil, err
			}
			return l, nil
		})
	case ProviderOnnx:
		return NewOnnx(OnnxConfig{...})   // unchanged: lazy, hash fallback
	case ProviderExternal:
		return NewExternal(ExternalConfig{...}) // unchanged
	default:
		log.Printf("warning: unknown embedder provider %q, degrading to null", cfg.Provider)
		return NewNull()
	}
}
```

`embedder.Default()` (no-arg, backward compatible) now routes through the
config: `MNEMONIC_EMBED` off → `nil` (as before); on →
`BuildFromConfig` over `config.Load(".")`. Behavior notes:

- The shipped `config.d/indexing.yaml` has no `provider` key, and
  `config.Load` falls back to `DefaultEmbedder()` (`provider: onnx`), so an
  absent key keeps the onnx default (previously the HashEmbedder stub;
  Onnx is itself hash-fallback when the model file is absent, so ranking
  behavior is equivalent).
- `onnx`/`external`/`off` construction is unchanged (no eager verification),
  so the existing `resolveEmbedder`/`cmdEmbedder`/`httpResolveEmbedder`
  call sites (service, doctor, http) keep their exact behavior; they were
  left untouched to keep the step additive.

## 06.4 — Null degradation on load failure — `404a05a`

`degradeOnError` wraps provider creation: on error (or nil result) it logs
`warning: embedder provider %q failed to load: %v (degrading to null)` and
returns `NewNull()`. `NullEmbedder.Embed`/`EmbedQuery` already return a
zero-length vector without error (`null.go`, unchanged). Verified:
unreachable Ollama port → `NullEmbedder`; missing local model →
`NullEmbedder`; unknown provider → `NullEmbedder`; `Embed` on Null returns
`len == 0`, `err == nil`.

## Tests — how they mock ollama + provide a local model

- `TestOllamaEmbedder` (ollama_test.go): `httptest.NewServer` via the shared
  `newOllamaMockServer` mimics `/api/embeddings` — records path,
  Content-Type, and JSON body (thread-safe, shared with the 06.3 selection
  test via `newOllamaMock`), and replies `{"model","embedding":[...]}` with
  a deterministic 12-dim vector. Asserts the returned vector equals the
  server's, and that the request body carried `model` + the input text
  (`prompt == "test text"`) and hit `/api/embeddings` with
  `Content-Type: application/json`. `TestOllamaEmbedError` covers the HTTP
  404 path.
- `TestLocalONNXEmbedder` (local_test.go): a hand-written minimal ONNX
  protobuf (Identity Constant node, 61 bytes, no weights) is written to a
  temp dir; the embedder is configured with `ModelDir` + `Model` and the
  test asserts `ModelPath()` resolution, that the file is found, and that
  `loadSession()` returns nil-or-`ErrRuntimeUnavailable`. Missing dir/file
  → `Load`/`Embed` errors name the exact file (`ghost-model.onnx not
  found ...`).
- `TestEmbedderConfigDrivenSelection` (select_test.go): writes real
  `indexing.yaml` fixtures into temp `config.d/` dirs and asserts
  `config.Load` + `BuildFromConfig` return `*Ollama` / `*LocalONNX` /
  `*Onnx` / `*External` / `NullEmbedder` per provider; absent key → onnx
  default; empty provider → Null.
- `TestEmbedderLoadFailureReturnsNull` (degrade_test.go): `provider:
  ollama` + `base_url` to a freshly closed port → `NullEmbedder`; `Embed`
  returns a zero-length vector without error.

## RED → GREEN evidence

| Sub-task | RED (confirmed FAIL) | GREEN (confirmed PASS) |
|----------|----------------------|------------------------|
| 06.1 | `undefined: OllamaConfig / NewOllama` (build fail) | `go test -run 'TestOllamaEmbedder\|TestOllamaEmbedError'` → PASS |
| 06.2 | `undefined: LocalConfig / NewLocal` (build fail) | `go test -run 'TestLocalONNXEmbedder'` → PASS (2 tests) |
| 06.3 | `c.ModelDir undefined (type config.EmbedderConfig ...)` (build fail) | `go test -run 'TestEmbedderConfigDrivenSelection\|TestConfigLoadsNewProviders'` → PASS |
| 06.4 | `unreachable ollama → *embedder.Ollama, want NullEmbedder (degraded)` | `go test -run 'TestEmbedderLoadFailureReturnsNull'` → PASS |

## Full relevant suite

`go test ./skillgrid-cli/internal/mnemonic/embedder/ -count=1` (from repo
root): all 18 tests PASS — `TestEmbedderLoadFailureReturnsNull`,
`TestHashEmbedderDeterministic`, `TestExternalConfigurable`,
`TestExternalNoBaseURL`, `TestLocalONNXEmbedder`,
`TestLocalONNXEmbedderMissing`, `TestOllamaEmbedder`,
`TestOllamaEmbedError`, `TestOnnxDefault`, `TestNullAdapter`,
`TestEmbedderConfigDrivenSelection`, `TestConfigLoadsNewProviders`,
`TestConfigLoadValidYAML`, `TestWarmEmbedder`, `TestWarmEvict`,
`TestWarmFallback`, `TestWarmFailedFactoryDegradesToFTS`,
`TestWarmReload` — `ok ... embedder 0.938s`.

## Baseline suites (005/008/010/011/013)

`go test ./skillgrid-cli/internal/mnemonic/store/... ./internal/mnemonic/service/... ./internal/mnemonic/memory/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1`:

- store ok (5.4s), memory ok (13.6s) + memory/layer ok (7.6s), route ok
  (4.5s), affected ok (13.6s), community ok (5.4s), pdg ok (9.3s),
  codeindex ok (20.0s)
- **service FAIL: `TestReindexStructuralIsEmbedderFree`** — "sql: database
  is closed" at reindex_structural_test.go:107. **Pre-existing**: fails
  identically with all step-06 changes stashed (verified via `git stash`).
  Root cause is unrelated to this step: `Indexer.Run` (codeindex/indexer.go:498-506)
  closes `idx.store.DB` and swaps in a fresh pass DB after commit; with the
  refcounted pooled store (449f536) the pooled `*sql.DB` is closed
  out-from-under the test's second `OpenForDirectory` handle.
- Also pre-existing (same stash check): `integration:
  TestSeedAllStoresVisibleOverHTTP` ("save web cache: sql: database is
  closed") and the tracker suite `TestStep02_CLI_Exit1` is flaky under full
  run load (passes in isolation, 0.63s).

## Commits

- `dcbf534` feat(mnemonic): add ollama embedder provider
- `d716f95` feat(mnemonic): add local ONNX embedder provider
- `998f1bf` feat(mnemonic): config-driven embedder provider selection
- `404a05a` feat(mnemonic): degrade to Null embedder on load failure

## Concerns

1. **Local ONNX model-file test**: `libonnxruntime` is NOT installed on
   this host, so full ONNX session creation cannot be exercised in the test.
   The test therefore asserts the load-path contract (file found →
   resolution + `loadSession` returns nil-or-`ErrRuntimeUnavailable`;
   file missing → descriptive error naming the file) and treats inference
   as best-effort. A real nomic-embed-code `.onnx` + tokenizer pipeline
   (tokenize → run → pool) is still a follow-up, same as the existing
   `onnx.go` embedder (which falls through to hash after load).
2. **`Default()` behavior change (MNEMONIC_EMBED=1 path)**: previously
   returned `HashEmbedder{}`; now returns the config-selected provider
   (onnx default → `*Onnx`, which is hash-fallback when the model file is
   absent). Tool contracts unchanged; only the `SemanticSearch` path that
   calls `embedder.Default()` is affected, and it degrades identically
   without a model file.
3. **Pre-existing service/integration failures** (database is closed /
   tracker flake) are not caused by this step — verified by stashing all
   step-06 changes and re-running; they fail identically on the base.
