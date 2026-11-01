package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
)

// anyLLMModel reports a seam's Model() for test failure messages ("" when the
// seam has no Model accessor or is nil).
func anyLLMModel(seam interface{}) string {
	if m, ok := seam.(interface{ Model() string }); ok {
		return m.Model()
	}
	return ""
}

// clearMnemonicEnv resets every env var the loader consults so each test's
// Setenv set is the only env input (t.Setenv restores after the test).
func clearMnemonicEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"MNEMONIC_DATA_DIR", "MNEMONIC_PROJECT", "MNEMONIC_PG_DSN",
		"MNEMONIC_LLM_API_KEY", "MNEMONIC_SYNC_AUTO", "MNEMONIC_DISABLE_CACHE",
	} {
		t.Setenv(k, "")
	}
}

// TestConfigResolutionOrder covers the flag > env > file > default precedence
// for data_dir (BDD [config-resolution-order]).
func TestConfigResolutionOrder(t *testing.T) {
	clearMnemonicEnv(t)
	// env must beat default
	t.Setenv("MNEMONIC_DATA_DIR", "/from-env")
	cfg, err := config.Load("", config.Flags{}) // no flag
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != "/from-env" {
		t.Fatalf("env must win over default: %q", cfg.DataDir)
	}
	// flag must beat env
	cfg2, err := config.Load("", config.Flags{DataDir: "/from-flag"})
	if err != nil {
		t.Fatalf("Load with flag: %v", err)
	}
	if cfg2.DataDir != "/from-flag" {
		t.Fatalf("flag must win over env: %q", cfg2.DataDir)
	}
}

// TestConfigFileBeatsDefault pins the third precedence rung with different
// inputs (env unset, no flag): the discovered YAML value must win over the
// built-in default, and an unset env key must not clobber it.
func TestConfigFileBeatsDefault(t *testing.T) {
	clearMnemonicEnv(t)
	dir := t.TempDir()
	writeConfig(t, dir, "mnemonic:\n  data_dir: /from-file\n")
	got, err := config.Load(dir, config.Flags{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.DataDir != "/from-file" {
		t.Fatalf("file must win over default: %q", got.DataDir)
	}
}

// TestConfigValidation covers the spec 006-config §5 load-time validation
// (BDD [config-validation]).
func TestConfigValidation(t *testing.T) {
	clearMnemonicEnv(t)
	// postgres without a DSN must fail at load, not at first query.
	_, err := config.Load("", config.Flags{Dialect: "postgres"})
	if err == nil {
		t.Fatalf("postgres without DSN must fail at load/open, not at first query")
	}
	// The DSN env var must satisfy the same validation.
	t.Setenv("MNEMONIC_PG_DSN", "postgres://u:p@localhost/x")
	if _, err := config.Load("", config.Flags{Dialect: "postgres"}); err != nil {
		t.Fatalf("postgres with MNEMONIC_PG_DSN must load: %v", err)
	}
	// llm.provider=local without local.base_url must error.
	if _, err := config.LoadWithYAML(`mnemonic:
  llm:
    provider: local
`, config.Flags{}); err == nil {
		t.Fatalf("llm.provider=local without local.base_url must error")
	}
	// An unknown provider must error at load (actionable), not degrade
	// silently or surface only at BuildLLM time.
	if _, err := config.LoadWithYAML("mnemonic:\n  llm:\n    provider: anthropic\n", config.Flags{}); err == nil {
		t.Fatal("unknown llm.provider must error")
	}
	if _, err := config.LoadWithYAML("mnemonic:\n  embedder:\n    provider: onnx\n", config.Flags{}); err == nil {
		t.Fatal("unknown embedder.provider must error")
	}
	// A valid unknown-adjacent case must NOT error: an openai-compat provider
	// with a blank base_url degrades to the nil seam (INV-3), it is not
	// malformed.
	if _, err := config.LoadWithYAML("mnemonic:\n  llm:\n    provider: openai-compat\n", config.Flags{}); err != nil {
		t.Fatalf("openai-compat with blank base_url must load (nil seam at BuildLLM): %v", err)
	}
}

// TestConfigInvalidYAMLNamesFileAndLine is the TRIANGULATE test for the
// ticket's acceptance line "invalid YAML → error naming the file+line":
// different inputs than the brief's validation set (a structural YAML error,
// not a domain-validation error), and it asserts the error carries both the
// file path and the offending line number.
func TestConfigInvalidYAMLNamesFileAndLine(t *testing.T) {
	clearMnemonicEnv(t)
	dir := t.TempDir()
	// The syntax error is on line 3 ("  data_dir [1" — unquoted bracket).
	writeConfig(t, dir, "mnemonic:\n  enabled: true\n  data_dir [1\n")
	_, err := config.Load(dir, config.Flags{})
	if err == nil {
		t.Fatal("malformed YAML must error")
	}
	msg := err.Error()
	if !strings.Contains(msg, ".skillgrid/config.yaml") {
		t.Fatalf("error must name the file, got: %q", msg)
	}
	if !strings.Contains(msg, "3") {
		t.Fatalf("error must carry the offending line number, got: %q", msg)
	}
}

// TestConfigBuildSeams pins BuildLLM / BuildEmbedder across the provider
// matrix. The nil seam is the INV-3 "absent" state: the deterministic floor
// runs and results are flagged degraded.
func TestConfigBuildSeams(t *testing.T) {
	clearMnemonicEnv(t)
	cases := []struct {
		yaml       string
		envKey     string
		wantLLMNil bool
		wantEmbNil bool
	}{
		{"", "", true, true}, // no provider set: both seams absent
		{"mnemonic:\n  llm:\n    provider: openai-compat\n    base_url: http://127.0.0.1:9\n", "", false, true},
		{"mnemonic:\n  llm:\n    provider: local\n    local:\n      base_url: http://127.0.0.1:11434\n      model: ollama-model\n", "MNEMONIC_LLM_API_KEY", false, true},
		{"mnemonic:\n  embedder:\n    provider: openai-compat\n    base_url: http://127.0.0.1:9\n", "", true, false},
		{"mnemonic:\n  embedder:\n    provider: local\n    base_url: http://127.0.0.1:9\n", "", true, false},
		{"mnemonic:\n  embedder:\n    provider: none\n", "", true, false}, // "none" is present-and-absent: Null adapter, not a nil seam
	}
	for i, tc := range cases {
		cfg, err := config.LoadWithYAML(tc.yaml, config.Flags{})
		if err != nil {
			t.Fatalf("case %d: LoadWithYAML: %v", i, err)
		}
		if tc.envKey != "" {
			t.Setenv(tc.envKey, "secret-value")
		}
		llm, err := cfg.BuildLLM()
		if err != nil {
			t.Fatalf("case %d: BuildLLM: %v", i, err)
		}
		if (llm == nil) != tc.wantLLMNil {
			t.Fatalf("case %d: BuildLLM nil = %v, want %v", i, llm == nil, tc.wantLLMNil)
		}
		emb, err := cfg.BuildEmbedder()
		if err != nil {
			t.Fatalf("case %d: BuildEmbedder: %v", i, err)
		}
		if (emb == nil) != tc.wantEmbNil {
			t.Fatalf("case %d: BuildEmbedder nil = %v, want %v", i, emb == nil, tc.wantEmbNil)
		}
		// Assert on a constructed field, not just nil-ness: the local LLM
		// case must carry the configured local.base_url + model (BuildLLM
		// resolves per-provider), and the Null embedder must report its
		// absent identity.
		if i == 2 {
			// Assert on a constructed field, not just nil-ness: the local
			// provider must resolve local.model onto the client (the
			// OpenAI-compatible client exposes Model() on the seam).
			if got := anyLLMModel(llm); got != "ollama-model" {
				t.Fatalf("case %d: LLM seam Model() = %q, want %q (local.model must win)", i, got, "ollama-model")
			}
		}
		if i == 5 {
			if m := emb.Model(); m != "" {
				t.Fatalf("case %d: Null embedder Model() = %q, want empty", i, m)
			}
		}
	}
}

// TestConfigEnvOverrides pins the env-over-file resolution for the DSN (the
// precedence matrix rung the brief names "sync.dsn"), with different inputs
// than the data_dir tests: an env DSN must override a file DSN.
func TestConfigEnvOverrides(t *testing.T) {
	clearMnemonicEnv(t)
	dir := t.TempDir()
	writeConfig(t, dir, "mnemonic:\n  datasource:\n    dialect: postgres\n    dsn: postgres://file@localhost/file\n")
	t.Setenv("MNEMONIC_PG_DSN", "postgres://env@localhost/env")
	cfg, err := config.Load(dir, config.Flags{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Datasource.DSN != "postgres://env@localhost/env" {
		t.Fatalf("env DSN must override the file DSN: %q", cfg.Datasource.DSN)
	}
}

// TestConfigDefaults pins the spec 006-config §2 default table: a config with
// no file and no env carries the documented built-ins, not zeros.
func TestConfigDefaults(t *testing.T) {
	clearMnemonicEnv(t)
	// Load from an empty temp dir (not the repo): the walk-up from cwd would
	// otherwise discover the repo's own .skillgrid/config.yaml, which sets
	// data_dir: ~/.skillgrid/mnemonic — the file is legitimate input, and the
	// defaults must be observed without it.
	cfg, err := config.Load(t.TempDir(), config.Flags{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Enabled {
		t.Fatal("Enabled must default to true")
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".skillgrid", "mnemonic")
	if cfg.DataDir != want {
		t.Fatalf("DataDir default = %q, want %q", cfg.DataDir, want)
	}
	if cfg.Datasource.Dialect != storeDialectSQLite() {
		t.Fatalf("Dialect default = %v, want SQLite", cfg.Datasource.Dialect)
	}
	if cfg.Datasource.MaxConns != 8 {
		t.Fatalf("MaxConns default = %d, want 8", cfg.Datasource.MaxConns)
	}
	if cfg.Project.EnvVar != "MNEMONIC_PROJECT" || !cfg.Project.Detect {
		t.Fatalf("Project default = %+v", cfg.Project)
	}
	if cfg.LLM.TimeoutMs != 60000 || cfg.LLM.MaxRetries != 1 {
		t.Fatalf("LLM default = %+v", cfg.LLM)
	}
	if cfg.LLM.APIKeyEnv != "MNEMONIC_LLM_API_KEY" {
		t.Fatalf("APIKeyEnv default = %q", cfg.LLM.APIKeyEnv)
	}
	if cfg.Sync.Enabled || cfg.Sync.Auto || cfg.Sync.Target != "cloud" ||
		cfg.Sync.DebounceMs != 500 || cfg.Sync.PushLimit != 100 || cfg.Sync.MaxPushBytes != 8*1024*1024 {
		t.Fatalf("Sync default = %+v", cfg.Sync)
	}
	if cfg.Memory.ContentMaxBytes != 32000 || cfg.Memory.PrivateTag != "<private>" {
		t.Fatalf("Memory default = %+v", cfg.Memory)
	}
	for typ, months := range map[string]int{
		"config": 3, "preference": 3, "discovery": 6, "pattern": 6,
		"decision": 6, "architecture": 12, "bugfix": 12, "policy": 12,
	} {
		if got := cfg.Memory.Review[typ]; got != months {
			t.Fatalf("Review[%q] = %d, want %d", typ, got, months)
		}
	}
}

func writeConfig(t *testing.T, dir, body string) {
	t.Helper()
	p := filepath.Join(dir, ".skillgrid", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
