---
id: doc-001
title: 'Session summary: Shipped mnemonic-llm-provider (2026-10-02)'
type: other
created_date: '2026-10-05 19:11'
tags:
  - session-summary
  - architecture/mnemonic-llm-provider
  - release/2
  - decision
  - aiskillgrid
---
# Session summary: Shipped mnemonic-llm-provider

- change: 2026-10-02-mnemonic-llm-provider
- branch: release/2
- project: aiskillgrid
- topic_key: architecture/mnemonic-llm-provider
- intended mnemonic MCP save: type=decision, scope=project

## What
Delivered all 4 tasks of the 2026-10-02-mnemonic-llm-provider change on release/2.

1. New `skillgrid-cli/internal/mnemonic/llm` package: OpenAI-compatible HTTP Completer (`Complete(ctx, system, user) (string, error)` -> `{base_url}/chat/completions`), stdlib-only, api_key omits Bearer header when empty.
2. `mnemonic.llm` config section (`enabled, base_url, model, api_key, timeout`) in `internal/mnemonic/config/load.go` with env fallback `SKILLGRID_LLM_API_KEY` (precedence: explicit YAML > env > base).
3. `service.AttachSharedLLM(cfg config.LLMConfig) (*llm.Client, error)` — the ONE attach point: attaches process-level AskLLM + dedup func directly (client satisfies both verbatim), returns the client so openProject attaches memory-level extraction (adapter) + dedup seams. Dream adapter built+tested but NOT boot-attached (no *DreamExecutor at boot; future one-liner `dreamExec.SetLLM(service.NewDreamLLMAdapter(client))`).
4. `skillgrid install` provider step (`internal/install/provider.go`): Local Ollama ensure+wire or External OpenAI wire, merges home `~/.skillgrid/config.d/indexing.yaml` preserving unrelated keys; flags `--provider`/`-p`, `--skip-provider`, `--base-url`, `--api-key`. Closed FOLLOWUP task-029 (status done, superseded).

## Why
task-029 debt — production had no LLM client; seams only attached in tests -> "no LLM configured" floors. ADR-0023 locks OpenAI-compatible HTTP only, one attach, natural install provider setup.

## Where
- skillgrid-cli/internal/mnemonic/llm/
- skillgrid-cli/internal/mnemonic/config/load.go
- skillgrid-cli/internal/mnemonic/service/attach_llm.go
- skillgrid-cli/internal/mnemonic/service/service.go
- skillgrid-cli/internal/mnemonic/service/dedup_llm.go
- skillgrid-cli/internal/install/provider.go
- skillgrid-cli/internal/install/config.go
- skillgrid-cli/internal/install/install.go
- skillgrid-cli/cmd/skillgrid/main.go
- .backlog/tasks/task-029*
- .skillgrid/artifacts/04-adr-0023-*.md

## Learned
- `*llm.Client.Complete` signature `Complete(ctx, system, user) (string, error)` matches BOTH `service.AskLLM` and `service.dedupLLMFunc` verbatim — zero adapters for AskLLM/dedup; only `memory.ExtractionLLM` (method `Extract`) and `memory.DreamLLM` (methods `ConsolidatePrompt`/`SynthesizePrompt`) need thin Completer adapters.
- Memory-level seams (`SetExtractionLLM`, `SetDedupLLM`) live on `*memory.Service` which only exists in `openProject`, so the attach function must RETURN the client rather than attach everything — "ONE attach point" is satisfied by returning the shared client for the memory-level attach at the only place it can happen.
- Go's `http.Client.Timeout` does NOT cancel the server-side `r.Context()`; it aborts the client connection only. A sleeping `httptest` handler keeps sleeping, so `srv.Close()` teardown takes the full sleep — that's server teardown, not a client hang. Time `Complete` alone for timeout tests.
- This host runs a REAL Ollama on :11434 that returns its own tags, which masked the httptest server in install provider tests until the probe host/port was made an injectable package var (`ollamaBaseURL`) / parameter.
- `go test ./internal/mnemonic/service/` takes ~145-156s (timeout tests with 5s sleeps) — exceeds the default 120s shell timeout; run in background or with a larger timeout.
- Pre-existing failures on clean HEAD (NOT from this diff): cmd/skillgrid `TestOnboardingSkillPassesDocsThrough` (reads `.agents/skills/lifecycle/onboarding/SKILL.md` which lacks `--docs`), `TestLoadArchitectureTemplateNotFound`, `TestWriteBootFile*` (boot-file tracker tests); mnemonic/service `TestSearchObservationsAllParallelManyStores` (flaky 30s timeout, passes at 120s).
- gofmt flags struct-tag column alignment: a field with a longer name re-aligns `json:"..."` tags on all siblings — run `gofmt -l` after adding struct fields.
