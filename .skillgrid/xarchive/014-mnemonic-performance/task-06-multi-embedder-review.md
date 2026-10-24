# Task 06 Review — multi-embedder (ollama + local providers)

**Reviewer:** fresh, no prior context
**Scope:** `review-step06-scoped.diff` (952 lines, 9 files)
**Tests run:** `go test ./internal/mnemonic/embedder/ -count=1` → 18/18 PASS (1.05s) from `skillgrid-cli/`
**Build/vet:** `go build ./internal/mnemonic/...` OK; `go vet ./internal/mnemonic/embedder/` OK

## Verdict

**PASS WITH WARNINGS**

## Findings

### Major

**M1. `Default()` runtime behavior change for `MNEMONIC_EMBED=1` — HashEmbedder(64) → `*Onnx`(768), and the existing `resolveEmbedder` call sites are NOT migrated to `BuildFromConfig`.**
File: `skillgrid-cli/internal/mnemonic/embedder/embedder.go:101-106`, `skillgrid-cli/internal/mnemonic/service/service.go:1735-1757`
`Default()` previously returned the `HashEmbedder{}` stub (64-dim). It now returns
`buildDefaultFromConfig(config.Load("."))`. Because `config.Load` falls back to
`DefaultEmbedder()` (`Provider: "onnx"`) and the shipped `config.d/indexing.yaml`
has **no** `embedder` section, an absent provider now yields `*Onnx` (768-dim,
hash-fallback) instead of `HashEmbedder{}` (64-dim). The report calls this "equivalent
because Onnx is itself hash-fallback" — but the **dimension and hash bucketing differ**
(768 vs 64), so any path that reads `Default()` and embeds directly produces
different vector magnitudes/bucket counts than before. The single such caller is
`retrieval.go:71` (`emb := embedder.Default()`); it gates the vector leg on
`len(qVec.Data) > 0` (retrieval.go:93), so the *degraded* (no-model-file) case still
skips the vector leg — ranking is functionally equivalent in practice, but the change
is real, not "byte-identical." The report discloses it, so this is not a silent break.

The more important sub-issue: `resolveEmbedder` (service.go:1735), which is the actual
production resolver used by `service.go:1513/1541/1638/1714`, **was left unchanged** and
still has no `ollama`/`local` cases:

```go
switch cfg.Embedder.Provider {
case "external": return embedder.NewExternal(...)
case "off", "":  return nil
default:         return embedder.NewOnnx(...)   // "onnx"
}
```

So a config with `provider: ollama` or `provider: local` is **silently mapped to
`*Onnx`** at every `resolveEmbedder` call site, while the new `BuildFromConfig`
selector returns the correct type. The two selection paths now **disagree** for the new
providers. This is acceptable as *additive* (the step only wires the new `Default()`
path), but it means the `ollama`/`local` providers are only reachable through
`embedder.Default()` (`retrieval.go`), not through the main service/doctor/http resolver.
**Fix:** either (a) point `resolveEmbedder` at `BuildFromConfig` (the clean dedup), or
(b) add explicit `ollama`/`local` cases to `resolveEmbedder`, or at minimum document the
intentional split. Recommend (a) once `BuildFromConfig` is confirmed stable — it already
encodes the correct degrade-to-Null behavior that `resolveEmbedder` lacks.

### Minor

**M2. `LocalONNX.embedOne` loads the ONNX session but returns a zero vector — no real inference.**
File: `skillgrid-cli/internal/mnemonic/embedder/local.go:299-310`
```go
sess, err := l.loadSession()
if err != nil { return memory.Vector{}, err }
_ = sess
// Full ONNX inference ... not wired in this step.
return memory.Vector{Data: make([]float32, l.cfg.Dimension)}, nil
```
When the model **and** runtime are both present, `Embed` succeeds (no error) but returns
`{0,0,0,0,...}` — cosine similarity is 0 for every candidate, so ranking is flat. It is
not a panic or hard failure, and it mirrors `onnx.go`'s existing hash-fallback pattern
(which at least produces a non-zero deterministic hash). But a present model silently
producing a zero vector is a weaker floor than the `onnx` provider. The load-path
contract (found/missing) is the tested contract and is correct; inference is a disclosed
follow-up. Consider returning the `onnx`-style hash fallback here so a present model at
least ranks non-uniformly, or note it explicitly in the step follow-up list.

**M3. `isLibraryMissing` is string-matching over the onnxer error.**
File: `skillgrid-cli/internal/mnemonic/embedder/local.go:289-297`
```go
return strings.Contains(msg, "failed to load library") ||
    strings.Contains(msg, "dlopen") || strings.Contains(msg, "no such file")
```
Fragile if onnxer rewords its dlopen errors (the matched substrings are onnxer's
current English wording, confirmed live in this run: `dlopen(libonnxruntime.dylib...)`).
`"no such file"` is a broad match that could false-positive on a model-parse error that
happens to mention a missing file. Works today (verified: the dlopen error is correctly
classified as `ErrRuntimeUnavailable`). Acceptable; flag as a follow-up if onnxer exposes
a typed "library not found" error.

**M4. No test for a *successful* local `Embed` returning the configured dimension.**
File: `skillgrid-cli/internal/mnemonic/embedder/local_test.go`
`TestLocalONNXEmbedder` calls `loadSession()` (the load path) but never calls `l.Embed`
on the present model. On a host with `libonnxruntime` installed, `Load` succeeds and
`Embed` would return a 4-dim zero vector — untested. Because the runtime is absent on the
review host, `TestEmbedderConfigDrivenSelection` correctly routes the present-model local
case to `NullEmbedder` (runtime-absent branch, local_test.go:65-71). The load-path
contract is well covered; the inference happy-path is best-effort by design. Low risk.

### Nits

**N1. Custom `itoa` helper in a test file.** `degrade_test.go:111-123` hand-rolls an
int→string instead of `strconv.Itoa` (stdlib). Cosmetic.

**N2. Duplicate struct + config mapping.** `select.go` defines a parallel
`EmbedderConfig` and `buildDefaultFromConfig`/`toEmbedderConfig` both hand-map every field
between `config.EmbedderConfig` and the embedder `EmbedderConfig`. Correct today, but a
field added to one side silently drops from the other (the mapping is not exhaustive by
compiler). Consider a single shared type (move `EmbedderConfig` into `config` and have
`select.go` consume it directly) to delete one of the two mappers.

## Evidence checked (the 8 checks)

1. **Config-driven selection (06.3) — PASS.** `BuildFromConfig` switch (select.go:743-791)
   routes `ollama`→`*Ollama` (eager `Verify`), `local`→`*LocalONNX` (eager `Load`),
   `onnx`→`*Onnx`, `external`→`*External`, `off`/`""`→`NullEmbedder`, default→`Null` +
   logged warning. The **no-provider (absent key) case is preserved**: `config.Load`
   falls back to `DefaultEmbedder()` (`Provider:"onnx"`), confirmed by
   `TestEmbedderConfigDrivenSelection` ("absent provider key → config provider `onnx`,
   want the onnx default" + returns `*Onnx`). A *present-but-empty* provider maps to
   `Null` (the `""` branch) — distinct from absent-key. `onnx`/`external`/`off`
   construction is byte-identical to the existing `resolveEmbedder` (service.go:1735) —
   same `NewOnnx`/`NewExternal` constructors, same "lazy, no eager verification" behavior
   (the report's claim holds; the only delta is the new `ollama`/`local`/degrade paths).
   **Caveat:** the `Default()`-driven *runtime* change is M1.

2. **Ollama HTTP (06.1) — PASS.** Base URL configurable (`OllamaConfig.BaseURL`, default
   `http://localhost:11434`, ollama.go:458). Request body is `{"model","prompt"}`
   (ollama.go:513-516). Response `embedding` parsed to `[]float32` → `memory.Vector`
   (ollama.go:518-521, 549). **Timeout present:** `http.Client{Timeout: 30*time.Second}`
   (ollama.go:490) — not infinite. Non-200 → error incl. status + body
   (ollama.go:539-541); malformed JSON → `decode: %w` error (ollama.go:543-545); empty
   embedding → descriptive error (ollama.go:546-548). No panic path. `TestOllamaEmbedder`
   asserts returned dim=12 equals the server's deterministic vector, request body carried
   `model`+`prompt`, path `/api/embeddings`, `Content-Type: application/json`.
   `TestOllamaEmbedError` covers the 404 path.

3. **Local ONNX (06.2) — PASS (load path) / disclosed best-effort (inference).** Model path
   configurable: `LocalConfig.ModelDir` default `~/.skillgrid/models/`
   (local.go:199-205, 227-230), file resolved as `<ModelDir>/<Model>.onnx` (local.go:244-246),
   following the `onnx.go` file-name convention. **Load path is actually tested:**
   `TestLocalONNXEmbedder` writes a real 61-byte ONNX protobuf to a temp dir, asserts
   `ModelPath()` resolution, and that `loadSession()` returns nil-or-`ErrRuntimeUnavailable`
   (model present, library absent on host); `TestLocalONNXEmbedderMissing` asserts a
   missing file/dir yields a descriptive error naming `ghost-model.onnx` (Load and Embed).
   The `ErrRuntimeUnavailable` classification is verified live (the dlopen error is matched).
   It **reuses** the existing `onnx.go` pattern (`onnxruntime.LoadModelFromFile`, `defaultOnnxModel`/
   `defaultOnnxDim` consts) but is **more explicit** than `onnx.go`: `onnx.go` degrades to a
   hash fallback when the model is absent, whereas `LocalONNX` returns a load error (and the
   selector degrades to Null). Not a bug — a deliberate distinction the report explains.
   **Does NOT duplicate** `onnx.go`; it shares the consts and onnxer API.

4. **Null degradation (06.4) — PASS.** `degradeOnError` (select.go:796-807) wraps provider
   creation: on error (or nil) it logs `warning: embedder provider %q failed to load: %v
   (degrading to null)` and returns `NewNull()`. `NullEmbedder.Embed`/`EmbedQuery` return
   `memory.Vector{}` (zero-length, `Data` nil) with no error (null.go:20-25, unchanged).
   `TestEmbedderLoadFailureReturnsNull` points ollama at a dead port → `NullEmbedder`, and
   asserts `Embed`/`EmbedQuery` return `len==0, err==nil`; also covers missing local model →
   Null and unknown provider → Null. Verified live: the warning logs fire in the test output.

5. **No CGO (CRITICAL) — PASS.** `onnxer` is a **direct** dep in `skillgrid-cli/go.mod:6`
   (`github.com/benedoc-inc/onnxer v0.7.0`) with `purego` as `// indirect`
   (`go.mod:32`) — pure-Go binding, no CGO. `ollama.go` imports only stdlib
   (`bytes`,`context`,`encoding/json`,`fmt`,`io`,`net/http`,`time`) + the `memory` package.
   **None of the 4 step-06 commits touched `go.mod`/`go.sum`** (verified via `git show --stat`),
   so no new module dep was introduced — the existing `onnxer` is reused, exactly as the
   report claims. No CGO dependency added.

6. **Additive-ness / tool contract — PASS.** The diff touches only
   `config/load.go` (additive `ModelDir` field + merge) and 8 files in `embedder/`. No
   `mem_*`/`code_*`/`web_*` tool registration or signature appears anywhere in the diff.
   `Default()` is still no-arg and returns `Embedder` (nil or a provider), so existing
   callers (`retrieval.go:71`) compile and behave as before for the nil/zero-length gate.
   The only additive struct change is `config.EmbedderConfig.ModelDir` (a new field) — no
   existing field renamed or removed. `HashEmbedder` and the `Embedder` interface are
   untouched.

7. **TDD discipline — PASS (with M4 caveat).**
   - `TestOllamaEmbedder`: mock server asserts body contains the input text (`prompt ==
     "test text"`) + model, returns a known 12-dim vector; test verifies returned
     dimensionality and value. ✓
   - `TestLocalONNXEmbedder`: model-found (loads → nil or `ErrRuntimeUnavailable`) +
     model-missing (descriptive error). The "no libonnxruntime" case is handled gracefully
     (classified as `ErrRuntimeUnavailable`, logged, and the selector degrades to Null). ✓
     (Caveat: the present-model *Embed* happy-path is untested — M4.)
   - `TestEmbedderConfigDrivenSelection`: covers ollama (`*Ollama`), local
     (`*LocalONNX` or Null if runtime absent), onnx (`*Onnx`), external (`*External`), off
     (Null), absent key (`*Onnx` default), empty provider (Null). ✓
   - `TestEmbedderLoadFailureReturnsNull`: ollama on an unreachable port → Null (not an
     error); `Embed` returns zero-length vector without error. ✓
   - RED→GREEN evidence table in the report is consistent with the test set.

8. **Error handling — PASS.** All provider errors are descriptive and wrap the root cause:
   ollama wraps the `http` error and the decode error (`ollama.go:535,544`); local names the
   exact model file on missing (`local.go:257`) and wraps the onnxer load error
   (`local.go:262,264`). **No nil-pointer risk from a "created but not loaded" provider:**
   `LocalONNX` is nil-safe — `loadSession` guards on `l.sess != nil` under a mutex and is
   called from `embedOne` (local.go:300), so a `*LocalONNX` that skipped `Load` lazily loads
   on first `Embed` (degrading to Null only at *selection* time when `Load` is eager). The
   selector's `degradeOnError` also guards a nil result (select.go:802). `NewOllama`/`NewLocal`
   never return nil (they return pointers). The only theoretical nil-deref would be a
   hand-constructed `&LocalONNX{}` with a zero `cfg`, which the package constructors prevent
   by defaulting `ModelDir`/`Model`/`Dimension`.

## Approved to merge

**Yes — with the M1 follow-up tracked.** The step delivers everything the brief requires:
config-driven selection, ollama HTTP provider, local ONNX load-path, Null degradation,
no CGO, no tool-contract change, additive only, and 18/18 tests green. The one major item
(M1) is a *disclosed* behavior nuance plus a deliberate split between `Default()`
(new selector) and `resolveEmbedder` (legacy) for the new providers — it does not break
compilation, the tool contract, or the degraded-case ranking, and the step is correctly
additive. Recommend either pointing `resolveEmbedder` at `BuildFromConfig` or documenting
the split before the next step, so `ollama`/`local` are reachable through the main service
path (or the split is explicitly intended).
