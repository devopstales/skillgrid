// Package config loads the CTX (Context Orchestrator) configuration block.
//
// The loader implements the ADR-0036 config.d convention, reusing the
// findConfigD shape proven in internal/config/pricing.go:
//
//	builtin defaults
//	  -> ~/.skillgrid/config.d/ctx.yaml            (machine-local)
//	    -> first config.d/ctx.yaml found walking up (repo-local, wins per key)
//
// Every CtxConfig key has a safe default; per-key merge means a file only
// overrides the keys it actually sets. The clm: block is NOT part of this
// package — it stays in .skillgrid/config.yaml (ADR-0027, loaded by
// internal/config). Only the new CTX v1.11 blocks live in ctx.yaml.
package config

import (
	"context"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CtxConfig is the fully-resolved CTX configuration after the config.d merge.
// Every field is populated with a safe default by DefaultCtxConfig; Load only
// overrides the keys that a machine-local or repo-local ctx.yaml sets.
type CtxConfig struct {
	Proxy       Proxy       `yaml:"proxy"`
	Wrap        Wrap        `yaml:"wrap"`
	Judgment    Judgment    `yaml:"judgment"`
	Traffic     Traffic     `yaml:"traffic_learning"`
	Observability Observability `yaml:"observability"`
}

// Proxy is the ctx proxy (loopback LLM proxy) configuration.
//
// Mode selects the compression strategy: token | cache | passthrough. The
// default is "cache" (freeze provider-confirmed cached prefixes, compress only
// the mutable tail). It is also settable at startup / runtime and via the
// CTX_PROXY_MODE env var, checked by the proxy after config (ADR-0016 fail-open).
type Proxy struct {
	Enabled            bool     `yaml:"enabled"`
	Bind               string   `yaml:"bind"`
	Port               int      `yaml:"port"`
	Mode               string   `yaml:"mode"`
	CompressEndpoint   string   `yaml:"compress_endpoint"`
	NonLoopbackStatus  int      `yaml:"non_loopback_status"`
	SavingsHeaders     bool     `yaml:"savings_headers"`
	FailOpen           bool     `yaml:"fail_open"`
	ProviderRetries    int      `yaml:"provider_retries"`
	ProviderRetryBackoffMs int  `yaml:"provider_retry_backoff_ms"`
	PassthroughProtocols []string `yaml:"passthrough_protocols"`
	Upstreams          []string `yaml:"upstreams"`
}

// Wrap is the ctx wrap (tier-2 agent launcher) configuration. Agents is the
// supported core-4 set by default (opencode, claude, codex, cursor); the
// launcher is the only component that knows about specific agents.
type Wrap struct {
	Enabled      bool     `yaml:"enabled"`
	Agents       []string `yaml:"agents"`
	InjectEnv    bool     `yaml:"inject_env"`
	BackupConfig bool     `yaml:"backup_config"`
}

// Judgment is the local-Ollama judgment model (ADR-0035). Model defaults to
// tev1:0.8b at the local Ollama /v1/systemone endpoint; clef-flash is the
// opt-in larger alternative. Threshold is the truncate-only decision
// threshold (range 0.20–0.25, default 0.22). A down endpoint fails open to
// pre-filter-only.
type Judgment struct {
	Model     string  `yaml:"model"`
	Endpoint  string  `yaml:"endpoint"`
	Threshold float64 `yaml:"threshold"`
}

// Traffic is the zero-LLM rule-based traffic learning extractor. Rules are a
// human-editable traffic/rules.yaml; below MinConfidence a memory is not
// stored, and MaxMemories caps the store (evicting oldest low-confidence).
type Traffic struct {
	Enabled       bool     `yaml:"enabled"`
	Realtime      bool     `yaml:"realtime"`
	RulesPath     string   `yaml:"rules_path"`
	Categories    []string `yaml:"categories"`
	MinConfidence float64  `yaml:"min_confidence"`
	MaxMemories   int      `yaml:"max_memories"`
	EmitEvents    bool     `yaml:"emit_events"`
}

// Observability is the four read-only endpoints folded into the existing
// mnemonic serve mux. MetricsPath is the Prometheus text route (default
// /metrics).
type Observability struct {
	Enabled        bool   `yaml:"enabled"`
	Stats          bool   `yaml:"stats"`
	StatsHistory   bool   `yaml:"stats_history"`
	Metrics        bool   `yaml:"metrics"`
	MetricsPath    string `yaml:"metrics_path"`
	HistoryRollups []string `yaml:"history_rollups"`
	HistorySnapshot string `yaml:"history_snapshot"`
	SessionSummary bool   `yaml:"session_summary"`
}

// DefaultCtxConfig returns the production defaults for every CTX key. An
// absent ctx.yaml yields exactly this value.
func DefaultCtxConfig() CtxConfig {
	return CtxConfig{
		Proxy: Proxy{
			Enabled:                false,
			Bind:                   "127.0.0.1",
			Port:                   8787,
			Mode:                   "cache",
			CompressEndpoint:       "/v1/compress",
			NonLoopbackStatus:      404,
			SavingsHeaders:         true,
			FailOpen:               true,
			ProviderRetries:        3,
			ProviderRetryBackoffMs: 200,
			PassthroughProtocols:   []string{"http1.1", "http2", "sse", "ws"},
			Upstreams:              []string{"anthropic", "openai"},
		},
		Wrap: Wrap{
			Enabled:      false,
			Agents:       []string{"opencode", "claude", "codex", "cursor"},
			InjectEnv:    true,
			BackupConfig: true,
		},
		Judgment: Judgment{
			Model:     "tev1:0.8b",
			Endpoint:  "http://127.0.0.1:11434/v1/systemone",
			Threshold: 0.22,
		},
		Traffic: Traffic{
			Enabled:       false,
			Realtime:      false,
			RulesPath:     "traffic/rules.yaml",
			Categories:    []string{"error_recovery", "environment", "preference", "architecture"},
			MinConfidence: 0.6,
			MaxMemories:   500,
			EmitEvents:    true,
		},
		Observability: Observability{
			Enabled:         true,
			Stats:           true,
			StatsHistory:    true,
			Metrics:         true,
			MetricsPath:     "/metrics",
			HistoryRollups:  []string{"hourly", "daily", "weekly", "monthly"},
			HistorySnapshot: ".skillgrid/ctx/stats_history.json",
			SessionSummary:  true,
		},
	}
}

// Load resolves the CTX configuration for startDir using the ADR-0036
// config.d precedence: builtin defaults -> ~/.skillgrid/config.d/ctx.yaml
// (machine-local) -> the first config.d/ctx.yaml found walking up from
// startDir (repo-local). Per-key merge: a later file overrides only the keys
// it sets; keys it leaves unset keep the lower layer's value. A malformed
// file is skipped (defaults survive) rather than aborting the load.
//
// The ctx parameter is accepted for API symmetry with other loaders; the
// loader is pure filesystem I/O and does not observe cancellation.
func Load(ctx context.Context, startDir string) (*CtxConfig, error) {
	cfg := DefaultCtxConfig()

	// Layer 2: machine-local (~/.skillgrid/config.d/ctx.yaml).
	if home, err := os.UserHomeDir(); err == nil {
		cfg.mergeFile(filepath.Join(home, ".skillgrid", "config.d", "ctx.yaml"))
	}

	// Layer 3: repo-local (first config.d/ctx.yaml walking up from startDir).
	if path, ok := findConfigD(startDir, "ctx.yaml"); ok {
		cfg.mergeFile(path)
	}

	return &cfg, nil
}

// mergeFile unmarshals a single ctx.yaml layer on top of the current config.
// A missing or unparseable file is a no-op so a bad layer never aborts the
// load (the lower layer's values survive). Only non-zero / non-empty fields
// override, so a file that sets one key leaves the rest intact.
func (c *CtxConfig) mergeFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// Unmarshal into a CtxConfig directly: each struct's zero value is the
	// "absent" sentinel for that key's merge (see mergeProxy etc.).
	var layer CtxConfig
	if err := yaml.Unmarshal(data, &layer); err != nil {
		return
	}
	mergeProxy(&c.Proxy, layer.Proxy)
	mergeWrap(&c.Wrap, layer.Wrap)
	mergeJudgment(&c.Judgment, layer.Judgment)
	mergeTraffic(&c.Traffic, layer.Traffic)
	mergeObservability(&c.Observability, layer.Observability)
}

// mergeProxy applies the proxy layer on top of base. Scalar zero values are
// the "absent" sentinel; slice fields override only when the file actually
// set them (a present empty list is still an override — operators can clear
// the upstream set explicitly).
func mergeProxy(base *Proxy, l Proxy) {
	if l.Enabled {
		base.Enabled = true
	}
	if l.Bind != "" {
		base.Bind = l.Bind
	}
	if l.Port != 0 {
		base.Port = l.Port
	}
	if l.Mode != "" {
		base.Mode = l.Mode
	}
	if l.CompressEndpoint != "" {
		base.CompressEndpoint = l.CompressEndpoint
	}
	if l.NonLoopbackStatus != 0 {
		base.NonLoopbackStatus = l.NonLoopbackStatus
	}
	if l.SavingsHeaders {
		base.SavingsHeaders = true
	}
	if l.ProviderRetries != 0 {
		base.ProviderRetries = l.ProviderRetries
	}
	if l.ProviderRetryBackoffMs != 0 {
		base.ProviderRetryBackoffMs = l.ProviderRetryBackoffMs
	}
	if len(l.PassthroughProtocols) > 0 {
		base.PassthroughProtocols = l.PassthroughProtocols
	}
	if len(l.Upstreams) > 0 {
		base.Upstreams = l.Upstreams
	}
}

func mergeWrap(base *Wrap, l Wrap) {
	if l.Enabled {
		base.Enabled = true
	}
	if len(l.Agents) > 0 {
		base.Agents = l.Agents
	}
}

func mergeJudgment(base *Judgment, l Judgment) {
	if l.Model != "" {
		base.Model = l.Model
	}
	if l.Endpoint != "" {
		base.Endpoint = l.Endpoint
	}
	if l.Threshold != 0 {
		base.Threshold = l.Threshold
	}
}

func mergeTraffic(base *Traffic, l Traffic) {
	if l.Enabled {
		base.Enabled = true
	}
	if l.Realtime {
		base.Realtime = true
	}
	if l.RulesPath != "" {
		base.RulesPath = l.RulesPath
	}
	if len(l.Categories) > 0 {
		base.Categories = l.Categories
	}
	if l.MinConfidence != 0 {
		base.MinConfidence = l.MinConfidence
	}
	if l.MaxMemories != 0 {
		base.MaxMemories = l.MaxMemories
	}
}

func mergeObservability(base *Observability, l Observability) {
	if l.MetricsPath != "" {
		base.MetricsPath = l.MetricsPath
	}
	if len(l.HistoryRollups) > 0 {
		base.HistoryRollups = l.HistoryRollups
	}
	if l.HistorySnapshot != "" {
		base.HistorySnapshot = l.HistorySnapshot
	}
}

// findConfigD returns the first config.d/<name> found walking up from
// startDir to the filesystem root, or ("", false) when none exists. This is
// the same helper shape as internal/config/pricing.go:findConfigD, copied into
// this package so the ctx config loader is self-contained (the two live in
// different modules of the config tree and the helper is unexported there).
func findConfigD(startDir, name string) (string, bool) {
	if startDir == "" {
		return "", false
	}
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, "config.d", name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
