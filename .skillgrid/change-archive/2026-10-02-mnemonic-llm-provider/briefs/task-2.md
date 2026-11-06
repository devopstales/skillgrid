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

