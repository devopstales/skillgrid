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

