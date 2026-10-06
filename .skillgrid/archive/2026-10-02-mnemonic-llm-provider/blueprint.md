# Shared LLM Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Smallest usable whole  
*(why: Completer+attach is the usable core; install provider setup thickens the same change once the client works)*

**Classification:** standard (L2 floor)

**Queued behind:** `2026-10-02-mnemonic-project-init` (memory-checkpoint must clear first) — do not take `state.yaml` `current_change` until this change is next. per ASSUMPTIONS.md § Locked constraints (serial development)

**Goal:** Add one OpenAI-compatible Completer, attach it to all mnemonic LLM seams at boot, and make `skillgrid install` run natural Local/External provider setup for llm + embedder.

**Architecture:** New `internal/mnemonic/llm` Completer (stdlib HTTP chat completions). `service.AttachSharedLLM` adapts Completer → AskLLM / DedupLLMFunc / ExtractionLLM / DreamLLM. Config `mnemonic.llm` in the existing indexing YAML stack (home `~/.skillgrid/config.d/indexing.yaml` overlay). Install `setupProvider` after security tools: Local = Claude OS `setup_ollama` ensure; External = same-host wire. per `.skillgrid/artifacts/04-adr-0023-openai-compatible-llm-provider.md`

**Tech Stack:** Go 1.22+ (`net/http`, `encoding/json`, `os/exec`), existing `internal/mnemonic/config`, `service`, `memory`, `internal/install`

**Spec:** `.skillgrid/specs/2026-10-02-mnemonic-llm-provider/briefing.md`

**Findings:** `.skillgrid/artifacts/08-second-brain-roadmap.md` (J), `.skillgrid/artifacts/09-claude-os-deep-dive.md` (S5), Claude OS `setup-claude-os.sh` `setup_ollama`

## Hypothesis

**Claim:** With a mock OpenAI-compatible server and `mnemonic.llm.enabled=true`, AttachSharedLLM makes ask/dedup/extraction/dream invoke one Completer; Complete error fails open; `--yes` install (stubbed) ensures local models and merges home llm+ollama embedder.
**Right condition:** Tasks 1–5 named tests PASS; G9 matches task-029 → this topic.
**Wrong condition:** A second HTTP client or new LLM SDK appears, or attach requires per-seam constructors at every call site.
**Thinnest MVP:** Task 1 (Completer) + Task 2 (config) + Task 3 (AttachSharedLLM).
**Door check:** Task 1 `TestCompleteSuccess` + Task 3 `TestAttachSharedLLMWiresAllSeams`.

## Terms

- [Shared LLM Provider](../../artifacts/02-technical-terms.md)
- [mem_ask](../../artifacts/02-technical-terms.md) / [Cited Floor](../../artifacts/02-technical-terms.md)
- [AUDN](../../artifacts/02-technical-terms.md)
- [Multi-Embedder](../../artifacts/02-technical-terms.md)

## Must-Haves (goal-backward verification)

**Truths:**
- Default config has `mnemonic.llm.enabled=false`; enabled without `base_url`+`model` refuses attach
- `Complete` returns assistant content from `{base_url}/chat/completions`; 500/timeout → error; no new LLM SDK in go.mod
- After `AttachSharedLLM` with valid cfg, Ask / DedupFunc / Extraction / Dream adapters are non-nil and share one Completer counter (`backstop`)
- Client attached + feature flags off → floors; Complete error + flags on → floors (`backstop`)
- `--yes` install (stubbed) merges home `llm` + `embedder.provider=ollama`; `--provider=external` wires without ollama pull; `--skip-provider` no-ops
- task-029 references this topic

**Artifacts:**
- `skillgrid-cli/internal/mnemonic/llm/client.go` (+ `_test.go`)
- `skillgrid-cli/internal/mnemonic/config/load.go` — `LLM` section
- `skillgrid-cli/internal/mnemonic/service/attach_llm.go` (+ `_test.go`)
- Boot path calls AttachSharedLLM after `config.Load`
- `skillgrid-cli/internal/install/provider.go` (+ `_test.go`)
- `skillgrid-cli/cmd/skillgrid/main.go` — `--provider`, `--skip-provider`
- ADR-0023 (already accepted)

**Key links:**
- Config → Completer → AttachSharedLLM → seams
- Install Local/External → home indexing.yaml → next Load sees llm+embedder
- Feature flags stay separate from attach

**One-way-door decisions:**
- ADR-0023 OpenAI-compatible-only (accepted)
- `--yes` defaults to Local — escape `--skip-provider` / `--provider=external`

## Global Constraints

- Go 1.22+ to build. per ASSUMPTIONS.md § Locked constraints
- No new dependencies without an ADR; Completer is stdlib only. per ASSUMPTIONS.md + ADR-0023
- Serial development. per ASSUMPTIONS.md § Locked constraints
- Fail-open floors. per ADR-0016
- Spec-zone commits before code-zone. per ASSUMPTIONS.md § Locked constraints

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Network to LLM host | Applicable | Timeout 3s; errors; fail-open | `TestCompleteHTTPError`, `TestLLMErrorFailsOpen` |
| Shell out (brew/curl/ollama) | Applicable | Fakeable runner; non-fatal; dry-run | `TestSetupProviderDryRun` |
| Home config write | Applicable | Merge mnemonic keys only | `TestSetupProviderLocal` (asserts file) |
| Secrets (api_key) | Applicable | Env preferred; never log key | `TestLLMAPIKeyFromEnv` + external wire |

## File Structure

- `skillgrid-cli/internal/mnemonic/llm/` — Completer
- `skillgrid-cli/internal/mnemonic/config/load.go` — `LLM` + merge
- `skillgrid-cli/internal/mnemonic/service/attach_llm.go` — AttachSharedLLM
- `skillgrid-cli/internal/install/provider.go` — setupProvider
- `skillgrid-cli/cmd/skillgrid/main.go` — install flags
- Boot: `service.openProject` / `New` path after `config.Load` (same place extraction/dedup flags arm)

---

### Task 1: OpenAI-compatible Completer

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/llm/client.go`
- Test: `skillgrid-cli/internal/mnemonic/llm/client_test.go`

**Interfaces:**
- Consumes: none
- Produces:
  ```go
  type Completer interface {
      Complete(ctx context.Context, system, user string) (string, error)
  }
  type Config struct {
      BaseURL string
      Model   string
      APIKey  string
      Timeout time.Duration // 0 → 3s
  }
  func New(cfg Config) *Client // implements Completer
  ```
- Seam: Completer (HTTP Client + test stub)
- Adapters: ≥2
- Deletion test: without this package, every seam invents its own HTTP client

**SATISFIES:** `happy path openai-compatible complete succeeds`

- [ ] **Step 1: Write the failing test**

```go
package llm_test

func TestCompleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" && r.URL.Path != "/chat/completions" {
			t.Errorf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "hello"}}},
		})
	}))
	defer srv.Close()
	c := llm.New(llm.Config{BaseURL: srv.URL + "/v1", Model: "m", Timeout: time.Second})
	got, err := c.Complete(context.Background(), "sys", "user")
	if err != nil || got != "hello" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestCompleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	c := llm.New(llm.Config{BaseURL: srv.URL, Model: "m", Timeout: time.Second})
	if _, err := c.Complete(context.Background(), "s", "u"); err == nil {
		t.Fatal("want error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./skillgrid-cli/internal/mnemonic/llm -count=1 -run 'TestComplete'
```

Expected: FAIL (package / symbol missing)

- [ ] **Step 3: Write minimal implementation**

```go
package llm

// Client POSTs OpenAI chat completions to {BaseURL}/chat/completions.
type Client struct { /* http.Client, baseURL, model, apiKey */ }

func New(cfg Config) *Client { /* trim trailing /, timeout default 3s */ }

func (c *Client) Complete(ctx context.Context, system, user string) (string, error) {
	// POST JSON: model, messages[{role:system},{role:user}]
	// Authorization: Bearer <apiKey> if non-empty
	// return choices[0].message.content or error on non-2xx / empty
}
```

Also add `TestCompleteTimeout` (handler sleeps past Timeout).

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./skillgrid-cli/internal/mnemonic/llm -count=1 -run 'TestComplete'
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/llm/
git commit -m "$(cat <<'EOF'
feat(mnemonic): openai-compatible chat Completer

EOF
)"
```

---

### Task 2: `mnemonic.llm` config

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/config/load.go`
- Modify: `skillgrid-cli/internal/mnemonic/config/load_test.go`

**Interfaces:**
- Consumes: `mergeIndexing`, `homeIndexingYAML` overlay
- Produces:
  ```go
  type LLM struct {
      Enabled bool
      BaseURL string
      Model   string
      APIKey  string        // may be resolved from env at Validate/Resolve time
      Timeout time.Duration // default 3s
  }
  func (l LLM) ResolveAPIKey() string // field → SKILLGRID_LLM_API_KEY → OPENAI_API_KEY
  func (l LLM) Validate() error       // if Enabled, require BaseURL+Model
  ```
- Add `LLM` to `Indexing` + `mnemonicSection` yaml `llm:`; default `Enabled: false`, `Timeout: 3s`

**SATISFIES:** `happy path llm config defaults off`

- [ ] **Step 1: Write the failing test**

```go
func TestLLMConfigDefaultOff(t *testing.T) {
	dir := t.TempDir()
	got := Load(dir)
	if got.LLM.Enabled {
		t.Fatal("default enabled must be false")
	}
}

func TestLLMConfigRequiresURLAndModel(t *testing.T) {
	l := LLM{Enabled: true, BaseURL: "", Model: ""}
	if err := l.Validate(); err == nil {
		t.Fatal("want validate error")
	}
}

func TestLLMAPIKeyFromEnv(t *testing.T) {
	t.Setenv("SKILLGRID_LLM_API_KEY", "from-env")
	l := LLM{Enabled: true, BaseURL: "http://x", Model: "m"}
	if l.ResolveAPIKey() != "from-env" {
		t.Fatalf("got %q", l.ResolveAPIKey())
	}
}
```

- [ ] **Step 2: Run**

```bash
go test ./skillgrid-cli/internal/mnemonic/config -count=1 -run 'TestLLM'
```

Expected: FAIL

- [ ] **Step 3: Implement** — `LLM` type, yaml section, `DefaultIndexing` zero, `mergeIndexing` copy when section present, `Validate`/`ResolveAPIKey`

- [ ] **Step 4: Run** — expect PASS

- [ ] **Step 5: Commit** `feat(mnemonic): add mnemonic.llm config block`

---

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

### Task 4: Feature flags + fail-open

**Files:**
- Test: `skillgrid-cli/internal/mnemonic/service/attach_llm_failopen_test.go` (or extend attach tests)
- Prefer calling existing ask/dedup paths with error Completer

**SATISFIES:** `happy path attached client with flags off uses floors`, `happy path llm error fails open to floors`

- [ ] **Step 1: Write the failing test**

```go
type errCompleter struct{}

func (errCompleter) Complete(context.Context, string, string) (string, error) {
	return "", errors.New("down")
}

func TestAttachedClientFlagsOffUsesFloors(t *testing.T) {
	// Attach errCompleter; leave extraction.llm / dedup.llm false
	// Run CapturePassive / SaveWithAction hash path
	// Assert Completer never needed (n==0) and floor results OK
}

func TestLLMErrorFailsOpen(t *testing.T) {
	AttachSharedLLM(enabledCfg, errCompleter{}, mem, dream)
	// mem_ask mode=llm → citations present (degraded/cited floor)
	// SaveWithAction with hash-duplicate → noop floor
}
```

Reuse fixtures from `ask_test.go` / `dedup_llm_test.go` patterns.

- [ ] **Step 2: Run**

```bash
go test ./skillgrid-cli/internal/mnemonic/... -count=1 -run 'TestAttachedClientFlagsOffUsesFloors|TestLLMErrorFailsOpen'
```

Expected: FAIL or already PASS (document if existing floors hold)

- [ ] **Step 3: Fix** only if attach forces LLM paths on or swallows fail-open

- [ ] **Step 4: Run** — expect PASS

- [ ] **Step 5: Commit** `test(mnemonic): shared LLM flags off and fail-open`

---

### Task 5: Natural install provider setup

**Files:**
- Create: `skillgrid-cli/internal/install/provider.go`
- Test: `skillgrid-cli/internal/install/provider_test.go`
- Modify: `skillgrid-cli/internal/install/config.go` — `Provider string`, `SkipProvider bool`, `ProviderBaseURL`, `ProviderAPIKey` (external)
- Modify: `skillgrid-cli/internal/install/install.go` — after security tools: `setupProvider(c)`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go` — `--provider`, `--skip-provider`; `--yes` ⇒ Provider=`local` if empty

**Interfaces:**
- Consumes: `Config`, fakeable deps
- Produces:
  ```go
  type providerDeps struct {
      LookPath func(string) (string, error)
      Run      func(name string, args ...string) error
      Probe    func(base string) (models []string, ok bool) // GET /api/tags
      HomeYAML func() string // ~/.skillgrid/config.d/indexing.yaml under test home
  }
  func setupProvider(c *Config) error // fail-soft: log warning, return nil
  func mergeHomeProviderYAML(path string, local bool, baseURL, chatModel, embedModel, apiKey string) error
  ```
- Local: install if missing (darwin brew / linux curl install.sh); start if probe fails; pull `llama3.2:3b` + `nomic-embed-text` if absent; merge llm+ollama embedder
- External: no ollama cmds; merge llm+external embedder with baseURL/apiKey
- Dry-run: log only

**SATISFIES:** `happy path install yes ensures local ollama and wires config`

- [ ] **Step 1: Write the failing test**

```go
func TestSetupProviderLocal(t *testing.T) {
	home := t.TempDir()
	var ran []string
	deps := providerDeps{
		LookPath: func(string) (string, error) { return "", errors.New("missing") },
		Run: func(name string, args ...string) error {
			ran = append(ran, name+" "+strings.Join(args, " "))
			return nil
		},
		Probe: func(string) ([]string, bool) { return nil, true }, // after "install", pretend up with no models then with models — use stateful stub
		HomeYAML: func() string { return filepath.Join(home, "config.d", "indexing.yaml") },
	}
	c := &Config{Yes: true, Provider: "local", /* test home */}
	if err := setupProviderWith(c, deps); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(deps.HomeYAML())
	if !strings.Contains(string(raw), "provider: ollama") || !strings.Contains(string(raw), "llm:") {
		t.Fatalf("home yaml=%s", raw)
	}
}

func TestSetupProviderSkipped(t *testing.T) {
	c := &Config{SkipProvider: true}
	if err := setupProviderWith(c, providerDeps{Run: func(string, ...string) error {
		t.Fatal("must not run")
		return nil
	}}); err != nil {
		t.Fatal(err)
	}
}
```

Also: `TestSetupProviderExternal`, `TestSetupProviderEnsureSkipsPull`, `TestSetupProviderDryRun`, `TestSetupProviderFailureNonFatal`.

- [ ] **Step 2: Run**

```bash
go test ./skillgrid-cli/internal/install -count=1 -run 'TestSetupProvider'
```

Expected: FAIL

- [ ] **Step 3: Implement** `provider.go` + wire `Run` + main flags

- [ ] **Step 4: Run** — expect PASS

- [ ] **Step 5: Commit** `feat(install): natural Local/External LLM provider setup`

---

### Task 6: Close task-029

**Files:**
- Modify: `.backlog/tasks/task-029 - FOLLOWUP-shared-LLM-client-attach-for-ask-extraction-dedup-seams.md` — status Done when code ships

**SATISFIES:** `happy path task-029 superseded by this change`

- [ ] **Step 1:** Confirm G9:

```bash
rg -n "2026-10-02-mnemonic-llm-provider" .backlog/tasks/task-029*
```

Expected: match (already true)

- [ ] **Step 2:** After TICKET-02/03 land, set status Done and check DoD boxes

- [ ] **Step 5: Commit** (with ship, not early)

```bash
git add .backlog/tasks/task-029*
git commit -m "$(cat <<'EOF'
docs(backlog): close task-029 via mnemonic-llm-provider

[skillgrid-context]
task: TASK-029
EOF
)"
```

## Self-Review

| Check | Status |
|-------|--------|
| Spec requirements 1–7 → tasks | yes (1–2 config/client, 3 attach, 4 fail-open, 5 install, 6 tracker) |
| No placeholder steps | code blocks in Tasks 1–5 |
| ADR-0016 / ADR-0023 cited | yes |
| Door check named | Task 1 + 3 |
| Serial queue honored | Queued behind project-init |
| Threat matrix RED tests named | yes |
