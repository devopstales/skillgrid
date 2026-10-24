# T03 Report — CTX config block via ADR-0036 config.d

## Status
COMPLETE

## Commit
`b033885c` — `feat(ctx): config block via ADR-0036 config.d`

## Files created
- `mnemonic/internal/ctx/config/config.go` (320 lines)
- `mnemonic/internal/ctx/config/config_test.go` (267 lines)
- `mnemonic/internal/ctx/config/clm_sentinel_test.go` (34 lines)

## What was implemented

### CtxConfig struct
Five sub-blocks, each with safe defaults via `DefaultCtxConfig()`:

| Block | Keys | Defaults |
|---|---|---|
| `Proxy` | enabled, bind, port, mode, non_loopback_status, provider_retries, provider_retry_backoff_ms, passthrough_protocols, fail_open | false, 127.0.0.1, 8787, "cache", 404, 3, 200, [4 protocols], true |
| `Wrap` | enabled, agents | false, [opencode, claude, codex, cursor] |
| `Judgment` | model, endpoint, threshold | "tev1:0.8b", "http://127.0.0.1:11434/v1/systemone", 0.22 |
| `Traffic` | enabled, max_memories, min_confidence | false, 500, 0.6 |
| `Observability` | enabled, metrics_path | true, "/metrics" |

### Load(ctx, startDir)
Implements the ADR-0036 precedence chain:
1. Start from `DefaultCtxConfig()` (builtin).
2. Load `$HOME/.skillgrid/config.d/ctx.yaml` (machine-local) — per-key merge.
3. Walk up from `startDir` to find the first `config.d/ctx.yaml` (repo-local) — per-key merge, wins over machine-local.

Per-key merge semantics: a YAML key that is absent in a file leaves the value from the lower-precedence layer untouched. A key present in a file overwrites the lower layer's value. Malformed YAML files are skipped (not fatal) — the test `TestMalformedYAMLFallsBackToDefaults` proves this.

`findConfigD` is the same walk-up shape as `internal/config/pricing.go`, copied into this package (it was unexported there).

### CLM sentinel
`TestCLM_StillResolvesFromSkillgridConfig` imports the existing `internal/config` package and asserts that a `clm:` block in `.skillgrid/config.yaml` is still resolved by the old loader — proving the new package did not move or shadow the CLM config surface.

## Tests
All 7 tests GREEN:
- `TestDefaults` — every CtxConfig key has a safe default
- `TestRepoLocalWinsOverMachineLocalOverBuiltin` — per-key precedence
- `TestMachineLocalWinsOverBuiltinWhenNoRepoLocal` — absent repo-local falls to machine-local
- `TestNeitherFilePresentUsesDefaults` — no files → all defaults
- `TestRepoLocalFromSubdir` — walk-up discovery from a nested directory
- `TestMalformedYAMLFallsBackToDefaults` — bad YAML does not abort
- `TestCLM_StillResolvesFromSkillgridConfig` — CLM sentinel

## Gates
- G1 `go build ./...` — PASS (exit 0)
- G2 `go test ./internal/ctx/config/...` — PASS (7/7)
- G3 `go test ./internal/config/...` — PASS (CLM regression, exit 0)

## Concerns
None. The package is self-contained; no existing code was modified.
