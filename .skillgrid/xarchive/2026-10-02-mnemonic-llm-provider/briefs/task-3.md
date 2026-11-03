### Task 3: AttachSharedLLM wires all seams

> ⚠ one-way: ADR-0023 — adapters must wrap Completer only; no second HTTP client.

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/service/attach_llm.go`
- Test: `skillgrid-cli/internal/mnemonic/service/attach_llm_test.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go` in `openProject` (after `config.Load` / where `EnableExtractionLLM` / dedup arm) — call attach once per process when building memory service, or from a package-level `EnsureSharedLLM(cfg)` invoked from `openProject` and MCP boot

**Interfaces:**
- Consumes: `llm.Completer`, `config.LLM`, existing `SetAskLLM`, `SetDedupLLMFunc`, `memory.Service.SetExtractionLLM`, Dream `SetLLM`
- Produces:
  ```go
  // AttachSharedLLM wires or clears all seams. Disabled/invalid cfg or nil c → clear.
  func AttachSharedLLM(cfg config.LLM, c llm.Completer, mem *memory.Service, dream *memory.DreamExecutor)
  ```
- Adapters (same Completer):
  - AskLLM: `Complete` passthrough
  - DedupLLMFunc: `Complete(ctx, system, user)`
  - ExtractionLLM: `Extract` → `Complete(ctx, extractionSystem, text)`
  - DreamLLM: `ConsolidatePrompt` / `SynthesizePrompt` → `Complete` with fixed system strings

**SATISFIES:** `happy path attach wires all seams from one client`

- [ ] **Step 1: Write the failing test**

```go
type countingCompleter struct{ n *int32 }

func (c countingCompleter) Complete(ctx context.Context, system, user string) (string, error) {
	atomic.AddInt32(c.n, 1)
	return `{"verdict":"add"}`, nil
}

func TestAttachSharedLLMWiresAllSeams(t *testing.T) {
	var n int32
	c := countingCompleter{n: &n}
	cfg := config.LLM{Enabled: true, BaseURL: "http://x", Model: "m"}
	mem := /* new memory.Service on temp store, or nil-safe attach test doubles */
	AttachSharedLLM(cfg, c, mem, dream)
	if AskLLMSeam() == nil {
		t.Fatal("AskLLM nil")
	}
	_, _ = AskLLMSeam().Complete(context.Background(), "s", "u")
	// similarly invoke DedupLLMFunc / Extract / Dream consolidate once each
	if atomic.LoadInt32(&n) < 1 {
		t.Fatal("completer not called")
	}
}

func TestAttachSharedLLMDisabledClears(t *testing.T) {
	AttachSharedLLM(config.LLM{Enabled: true, BaseURL: "http://x", Model: "m"}, countingCompleter{n: new(int32)}, mem, dream)
	AttachSharedLLM(config.LLM{Enabled: false}, nil, mem, dream)
	if AskLLMSeam() != nil {
		t.Fatal("want cleared")
	}
}
```

Use the real package seams; for Extraction/Dream, assert getters or invoke methods that call Complete.

- [ ] **Step 2: Run** — expect FAIL

```bash
go test ./skillgrid-cli/internal/mnemonic/service -count=1 -run 'TestAttachSharedLLM'
```

- [ ] **Step 3: Implement** `attach_llm.go` adapters + clear path; from `openProject` after cfg load:

```go
if err := cfg.LLM.Validate(); err == nil && cfg.LLM.Enabled {
    client := llm.New(llm.Config{
        BaseURL: cfg.LLM.BaseURL,
        Model:   cfg.LLM.Model,
        APIKey:  cfg.LLM.ResolveAPIKey(),
        Timeout: cfg.LLM.Timeout,
    })
    AttachSharedLLM(cfg.LLM, client, mem, dreamExec)
} else {
    AttachSharedLLM(config.LLM{}, nil, mem, dreamExec)
}
```

Keep attach idempotent (once per process is OK; tests reset via clear).

- [ ] **Step 4: Run** — expect PASS; confirm `go.mod` has no new LLM SDK

- [ ] **Step 5: Commit** `feat(mnemonic): AttachSharedLLM for ask dedup extraction dream`

---

