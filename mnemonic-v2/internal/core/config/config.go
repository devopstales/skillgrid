// Package config loads the mnemonic memory-function configuration (spec
// 006-config): a YAML file discovered by walking up from a start dir to
// .skillgrid/config.yaml, environment overrides, and per-invocation flags.
//
// Resolution order for every setting (spec 006-config §4):
// flag > env > config file > default.
//
// The port is deliberate (v1's mnemonic_back/internal/config/load.go owns the
// proven pattern): walk-up file discovery, per-key "zero/empty keeps default"
// merges, and env fallbacks owned by the loader so the consumers (the llm /
// embedder clients) stay pure. This package is a leaf value — it constructs
// the intelligence seams but is constructed by nothing; handlers own it.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/embedder"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
	"gopkg.in/yaml.v3"
)

// Env var names (spec 006-config §3). Secrets are env-only: the config file
// carries the *name* of the env var holding a key, never the key itself.
//
// EnvHTTPToken and EnvTimezone are declared here (the spec lists them as
// mnemonic env vars) but consumed by the HTTP serve alias and TUI/CLI
// display — not by this loader.
const (
	EnvDataDir   = "MNEMONIC_DATA_DIR"
	EnvProject   = "MNEMONIC_PROJECT"
	EnvPGDSN     = "MNEMONIC_PG_DSN"
	EnvLLMAPIKey = "MNEMONIC_LLM_API_KEY"
	EnvSyncAuto  = "MNEMONIC_SYNC_AUTO"

	EnvHTTPToken = "MNEMONIC_HTTP_TOKEN"
	EnvTimezone  = "MNEMONIC_TIMEZONE"
)

// DefaultDataDir is the store location (spec 006-config §2:
// ~/.skillgrid/mnemonic; the brief's Config comment is the same default).
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".skillgrid/mnemonic"
	}
	return filepath.Join(home, ".skillgrid", "mnemonic")
}

// Datasource selects the physical backend (spec 001-storage §2, INV-2).
type Datasource struct {
	Dialect  store.Dialect
	DSN      string
	MaxConns int
}

// ProjectCfg is the mnemonic.project section. Name is the resolved
// project name (set by the CLI's detector; the autosync manager reads
// it for the peer's schema when the outbox is empty).
type ProjectCfg struct {
	EnvVar string
	Detect bool
	Name   string
}

// LLMConfig is the mnemonic.llm section. Provider "openai-compat" or "local"
// (Ollama-compatible); an unset provider is the "absent" seam (INV-3 floor).
type LLMConfig struct {
	Provider  string
	BaseURL   string
	Model     string
	APIKeyEnv string
	// LocalBaseURL / LocalModel are the local (Ollama-compatible) provider.
	LocalBaseURL string
	LocalModel   string
	TimeoutMs    int
	MaxRetries   int
}

// EmbedderCfg is the mnemonic.embedder section. Provider "openai-compat",
// "local", or "none"; an unset provider is the "absent" seam. Dimensions 0
// means "infer from the provider's first round-trip" (the client caches it);
// a mismatch against the vector column is caught at first write (Task 12),
// not here (spec 006-config §5).
type EmbedderCfg struct {
	Provider   string
	BaseURL    string
	Model      string
	Dimensions int
}

// SyncCfg is the mnemonic.sync section.
type SyncCfg struct {
	Enabled      bool
	Target       string
	Auto         bool
	DebounceMs   int
	PushLimit    int
	MaxPushBytes int
}

// MemoryCfg is the mnemonic.memory lifecycle section. Review is the
// per-type months map (spec 006-config §2 table).
type MemoryCfg struct {
	Review          map[string]int
	ContentMaxBytes int
	PrivateTag      string
}

// Config is the fully resolved configuration (a value, not a seam).
type Config struct {
	Enabled    bool
	DataDir    string
	Datasource Datasource
	Project    ProjectCfg
	LLM        LLMConfig
	Embedder   EmbedderCfg
	Sync       SyncCfg
	Memory     MemoryCfg
}

// Flags are the per-invocation selectors (CLI flags / MCP params). A
// non-zero field overrides env and file for that key.
type Flags struct {
	DataDir string
	Project string
	Dialect string
	DSN     string
}

// Default returns the built-in default configuration (spec 006-config §2).
func Default() Config {
	return Config{
		Enabled: true,
		DataDir: DefaultDataDir(),
		Datasource: Datasource{
			Dialect:  store.SQLite,
			MaxConns: 8,
		},
		Project: ProjectCfg{EnvVar: EnvProject, Detect: true},
		LLM: LLMConfig{
			APIKeyEnv:  EnvLLMAPIKey,
			TimeoutMs:  60000,
			MaxRetries: 1,
		},
		Sync: SyncCfg{
			Target:       "cloud",
			DebounceMs:   500,
			PushLimit:    100,
			MaxPushBytes: 8 * 1024 * 1024,
		},
		Memory: MemoryCfg{
			Review: map[string]int{
				"config":       3,
				"preference":   3,
				"discovery":    6,
				"pattern":      6,
				"decision":     6,
				"architecture": 12,
				"bugfix":       12,
				"policy":       12,
			},
			ContentMaxBytes: 32000,
			PrivateTag:      "<private>",
		},
	}
}

// configFile is the .skillgrid/config.yaml shape (spec 006-config §2).
type configFile struct {
	Mnemonic struct {
		Enabled    *bool  `yaml:"enabled"`
		DataDir    string `yaml:"data_dir"`
		Datasource struct {
			Dialect  string `yaml:"dialect"`
			DSN      string `yaml:"dsn"`
			MaxConns int    `yaml:"max_conns"`
		} `yaml:"datasource"`
		Project struct {
			EnvVar *string `yaml:"env"`
			Detect *bool   `yaml:"detect"`
		} `yaml:"project"`
		LLM struct {
			Provider  string `yaml:"provider"`
			BaseURL   string `yaml:"base_url"`
			Model     string `yaml:"model"`
			APIKeyEnv string `yaml:"api_key_env"`
			Local     struct {
				BaseURL string `yaml:"base_url"`
				Model   string `yaml:"model"`
			} `yaml:"local"`
			TimeoutMs  int `yaml:"timeout_ms"`
			MaxRetries int `yaml:"max_retries"`
		} `yaml:"llm"`
		Embedder struct {
			Provider   string `yaml:"provider"`
			BaseURL    string `yaml:"base_url"`
			Model      string `yaml:"model"`
			Dimensions int    `yaml:"dimensions"`
		} `yaml:"embedder"`
		Sync struct {
			Enabled      *bool  `yaml:"enabled"`
			Target       string `yaml:"target"`
			Auto         *bool  `yaml:"auto"`
			DebounceMs   int    `yaml:"debounce_ms"`
			PushLimit    int    `yaml:"push_limit"`
			MaxPushBytes int    `yaml:"max_push_bytes"`
		} `yaml:"sync"`
		Memory struct {
			Review struct {
				ConfigMonths       int `yaml:"config_months"`
				PreferenceMonths   int `yaml:"preference_months"`
				DiscoveryMonths    int `yaml:"discovery_months"`
				PatternMonths      int `yaml:"pattern_months"`
				DecisionMonths     int `yaml:"decision_months"`
				ArchitectureMonths int `yaml:"architecture_months"`
				BugfixMonths       int `yaml:"bugfix_months"`
				PolicyMonths       int `yaml:"policy_months"`
			} `yaml:"review"`
			ContentMaxBytes int    `yaml:"content_max_bytes"`
			PrivateTag      string `yaml:"private_tag"`
		} `yaml:"memory"`
	} `yaml:"mnemonic"`
}

// Load resolves the configuration: defaults, then the first .skillgrid/
// config.yaml found walking up from path ("" = cwd), then env, then flags.
// A found-but-malformed YAML file is an error naming the file and the
// offending line (spec: actionable errors). A missing file is not.
func Load(path string, flags Flags) (Config, error) {
	cfg := Default()
	if p, ok := findConfigYAML(path); ok {
		data, err := os.ReadFile(p)
		if err != nil {
			return cfg, fmt.Errorf("config: read %s: %w", p, err)
		}
		var f configFile
		if err := yaml.Unmarshal(data, &f); err != nil {
			// yaml.v3 errors carry "line N" for structural problems —
			// keep them in the message (spec: actionable errors naming the
			// file + line).
			return cfg, fmt.Errorf("config: %s: %w", p, err)
		}
		mergeFile(&cfg, f)
	}
	applyEnv(&cfg)
	applyFlags(&cfg, flags)
	return cfg, validate(cfg)
}

// LoadWithYAML resolves a config from an inline YAML document (the two
// input modes of spec 006-config §1: file-based and env-only — this is the
// testable form of the file-based mode). Same merge order as Load minus the
// walk-up discovery; flags still apply last.
func LoadWithYAML(yamlText string, flags Flags) (Config, error) {
	cfg := Default()
	var f configFile
	if err := yaml.Unmarshal([]byte(yamlText), &f); err != nil {
		return cfg, fmt.Errorf("config: <inline yaml>: %w", err)
	}
	mergeFile(&cfg, f)
	applyEnv(&cfg)
	applyFlags(&cfg, flags)
	return cfg, validate(cfg)
}

// mergeFile layers a parsed file section over cfg with the v1
// mergeIndexing "zero/empty keeps default" semantics: each key applies only
// when the file sets it (non-zero / non-nil / non-empty).
func mergeFile(cfg *Config, f configFile) {
	m := f.Mnemonic
	if m.Enabled != nil {
		cfg.Enabled = *m.Enabled
	}
	if m.DataDir != "" {
		cfg.DataDir = m.DataDir
	}
	if m.Datasource.Dialect != "" {
		cfg.Datasource.Dialect = dialectFromString(m.Datasource.Dialect)
	}
	if m.Datasource.DSN != "" {
		cfg.Datasource.DSN = m.Datasource.DSN
	}
	if m.Datasource.MaxConns > 0 {
		cfg.Datasource.MaxConns = m.Datasource.MaxConns
	}
	if m.Project.EnvVar != nil {
		cfg.Project.EnvVar = *m.Project.EnvVar
	}
	if m.Project.Detect != nil {
		cfg.Project.Detect = *m.Project.Detect
	}
	if m.LLM.Provider != "" {
		cfg.LLM.Provider = m.LLM.Provider
	}
	if m.LLM.BaseURL != "" {
		cfg.LLM.BaseURL = m.LLM.BaseURL
	}
	if m.LLM.Model != "" {
		cfg.LLM.Model = m.LLM.Model
	}
	if m.LLM.APIKeyEnv != "" {
		cfg.LLM.APIKeyEnv = m.LLM.APIKeyEnv
	}
	if m.LLM.Local.BaseURL != "" {
		cfg.LLM.LocalBaseURL = m.LLM.Local.BaseURL
	}
	if m.LLM.Local.Model != "" {
		cfg.LLM.LocalModel = m.LLM.Local.Model
	}
	if m.LLM.TimeoutMs > 0 {
		cfg.LLM.TimeoutMs = m.LLM.TimeoutMs
	}
	if m.LLM.MaxRetries > 0 {
		cfg.LLM.MaxRetries = m.LLM.MaxRetries
	}
	if m.Embedder.Provider != "" {
		cfg.Embedder.Provider = m.Embedder.Provider
	}
	if m.Embedder.BaseURL != "" {
		cfg.Embedder.BaseURL = m.Embedder.BaseURL
	}
	if m.Embedder.Model != "" {
		cfg.Embedder.Model = m.Embedder.Model
	}
	if m.Embedder.Dimensions > 0 {
		cfg.Embedder.Dimensions = m.Embedder.Dimensions
	}
	if m.Sync.Enabled != nil {
		cfg.Sync.Enabled = *m.Sync.Enabled
	}
	if m.Sync.Target != "" {
		cfg.Sync.Target = m.Sync.Target
	}
	if m.Sync.Auto != nil {
		cfg.Sync.Auto = *m.Sync.Auto
	}
	if m.Sync.DebounceMs > 0 {
		cfg.Sync.DebounceMs = m.Sync.DebounceMs
	}
	if m.Sync.PushLimit > 0 {
		cfg.Sync.PushLimit = m.Sync.PushLimit
	}
	if m.Sync.MaxPushBytes > 0 {
		cfg.Sync.MaxPushBytes = m.Sync.MaxPushBytes
	}
	if r := m.Memory.Review; r.ConfigMonths > 0 || r.PreferenceMonths > 0 ||
		r.DiscoveryMonths > 0 || r.PatternMonths > 0 || r.DecisionMonths > 0 ||
		r.ArchitectureMonths > 0 || r.BugfixMonths > 0 || r.PolicyMonths > 0 {
		review := cfg.Memory.Review
		if v := r.ConfigMonths; v > 0 {
			review["config"] = v
		}
		if v := r.PreferenceMonths; v > 0 {
			review["preference"] = v
		}
		if v := r.DiscoveryMonths; v > 0 {
			review["discovery"] = v
		}
		if v := r.PatternMonths; v > 0 {
			review["pattern"] = v
		}
		if v := r.DecisionMonths; v > 0 {
			review["decision"] = v
		}
		if v := r.ArchitectureMonths; v > 0 {
			review["architecture"] = v
		}
		if v := r.BugfixMonths; v > 0 {
			review["bugfix"] = v
		}
		if v := r.PolicyMonths; v > 0 {
			review["policy"] = v
		}
		cfg.Memory.Review = review
	}
	if m.Memory.ContentMaxBytes > 0 {
		cfg.Memory.ContentMaxBytes = m.Memory.ContentMaxBytes
	}
	if m.Memory.PrivateTag != "" {
		cfg.Memory.PrivateTag = m.Memory.PrivateTag
	}
}

// applyEnv layers the environment over the resolved config (spec 006-config
// §3). Only set-and-non-empty vars apply.
func applyEnv(cfg *Config) {
	if v := os.Getenv(EnvDataDir); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv(EnvPGDSN); v != "" {
		cfg.Datasource.DSN = v
	}
	if v := os.Getenv(EnvSyncAuto); v == "1" || strings.EqualFold(v, "true") {
		cfg.Sync.Auto = true
	}
	// MNEMONIC_PROJECT is consumed by the project Detector, not by this
	// value — it is not a Config field. MNEMONIC_LLM_API_KEY is read at seam
	// build time (BuildLLM/BuildEmbedder), so a key set after Load still
	// applies. MNEMONIC_DISABLE_CACHE is read by the store's handle pool
	// (core/store/pool.go), not here — the config never touches pool state.
}

// applyFlags layers per-invocation flags last (spec 006-config §4).
func applyFlags(cfg *Config, f Flags) {
	if f.DataDir != "" {
		cfg.DataDir = f.DataDir
	}
	if f.Dialect != "" {
		cfg.Datasource.Dialect = dialectFromString(f.Dialect)
	}
	if f.DSN != "" {
		cfg.Datasource.DSN = f.DSN
	}
	// f.Project is consumed by the project Detector (Resolve), not here.
}

// validate enforces spec 006-config §5 at load time (fail at open, not at
// first query). The embedder.dimensions-vs-vector-column check is deferred
// to first write (Task 12) per the spec.
func validate(cfg Config) error {
	if cfg.Datasource.Dialect == store.Postgres && strings.TrimSpace(cfg.Datasource.DSN) == "" {
		return fmt.Errorf("config: datasource.dialect=postgres requires a DSN (set datasource.dsn or %s)", EnvPGDSN)
	}
	if cfg.LLM.Provider == "local" && strings.TrimSpace(cfg.LLM.LocalBaseURL) == "" {
		return fmt.Errorf("config: llm.provider=local requires llm.local.base_url")
	}
	// An openai-compat provider with a blank base_url is NOT a load error:
	// BuildLLM returns the nil seam (INV-3 "absent" → degraded floor), which
	// is the default-less mode. Failures that cannot be degraded (a dialect
	// that needs a DSN) error here instead.
	switch cfg.LLM.Provider {
	case "", "openai-compat", "local":
	default:
		return fmt.Errorf("config: unknown llm.provider %q (openai-compat | local)", cfg.LLM.Provider)
	}
	switch cfg.Embedder.Provider {
	case "", "openai-compat", "local", "none":
	default:
		return fmt.Errorf("config: unknown embedder.provider %q (openai-compat | local | none)", cfg.Embedder.Provider)
	}
	return nil
}

// BuildLLM constructs the LLM seam from the resolved config. A nil (LLM, nil)
// is the INV-3 "absent" state: the deterministic floor runs and results are
// flagged degraded. The provider's base URL is resolved per provider
// (local → local.base_url, openai-compat → base_url); the API key is read
// from the env var named by APIKeyEnv (secrets never live in the file).
func (c Config) BuildLLM() (llm.LLM, error) {
	base := c.LLM.BaseURL
	if c.LLM.Provider == "local" {
		base = c.LLM.LocalBaseURL
	}
	if strings.TrimSpace(base) == "" {
		return nil, nil // absent seam (provider unset or URL left blank)
	}
	model := c.LLM.Model
	if c.LLM.Provider == "local" && c.LLM.LocalModel != "" {
		model = c.LLM.LocalModel
	}
	// The llm client applies its own 60s default when the timeout is
	// non-positive (LLM.TimeoutMs defaults to 60000, so this is a belt for
	// a config that forced it to 0).
	timeout := time.Duration(c.LLM.TimeoutMs) * time.Millisecond
	client, err := llm.NewClient(base, model, os.Getenv(c.LLM.APIKeyEnv), timeout)
	if err != nil {
		return nil, fmt.Errorf("config: build llm: %w", err)
	}
	return client, nil
}

// BuildEmbedder constructs the Embedder seam. Provider "none" is the
// present-but-absent Null adapter (a zero-dimension vector is never written);
// an unset or blank-provider config returns the nil seam (absent).
func (c Config) BuildEmbedder() (embedder.Embedder, error) {
	switch c.Embedder.Provider {
	case "none":
		return embedder.NewNull(), nil
	case "":
		return nil, nil // absent seam (validate rejects unknown providers at Load)
	}
	base := c.Embedder.BaseURL
	if strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("config: embedder.provider=%s requires embedder.base_url", c.Embedder.Provider)
	}
	// The LLM timeout is the shared HTTP timeout: the OpenAI-compatible
	// client serves both chat and embeddings, and EmbedderCfg has no timeout
	// field of its own (brief's struct), so c.LLM.TimeoutMs bounds the
	// embedder calls too.
	client, err := llm.NewClient(base, c.Embedder.Model, os.Getenv(EnvLLMAPIKey), time.Duration(c.LLM.TimeoutMs)*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("config: build embedder: %w", err)
	}
	return client, nil
}

// findConfigYAML walks up from startDir looking for .skillgrid/config.yaml
// (the v1 findIndexingYAML walk-up pattern, load.go:632-649). startDir ""
// means cwd.
func findConfigYAML(startDir string) (string, bool) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, ".skillgrid", "config.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

// dialectFromString maps a YAML/flag dialect name onto the store dialect.
func dialectFromString(s string) store.Dialect {
	if s == "postgres" {
		return store.Postgres
	}
	return store.SQLite
}
