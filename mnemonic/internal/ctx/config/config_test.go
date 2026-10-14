// Package config loads the CTX (Context Orchestrator) configuration block.
//
// The loader implements the ADR-0036 config.d convention, reusing the
// findConfigD shape proven in internal/config/pricing.go:
//
//	builtin defaults
//	  -> ~/.skillgrid/config.d/ctx.yaml            (machine-local)
//	    -> first config.d/ctx.yaml found walking up (repo-local, wins per key)
//
// The clm: block is NOT part of this file — it stays in .skillgrid/config.yaml
// (ADR-0027); see TestCLM_StillResolvesFromSkillgridConfig for the sentinel.
package config

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// load is a thin test helper that wires context.Background into Load.
func load(t *testing.T, dir string) (*CtxConfig, error) {
	t.Helper()
	return Load(context.Background(), dir)
}

// writeCtxYAML writes body into config.d/ctx.yaml under dir.
func writeCtxYAML(t *testing.T, dir, body string) string {
	t.Helper()
	d := filepath.Join(dir, "config.d")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, "ctx.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestDefaults verifies every CtxConfig key has a safe default (config-defaults).
func TestDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	c, err := load(t, t.TempDir())
	if err != nil {
		t.Fatalf("Load with no files: %v", err)
	}

	// proxy
	if c.Proxy.Enabled != false {
		t.Errorf("proxy.enabled default: got %v, want false (opt-in)", c.Proxy.Enabled)
	}
	if c.Proxy.Bind != "127.0.0.1" {
		t.Errorf("proxy.bind default: got %q, want 127.0.0.1", c.Proxy.Bind)
	}
	if c.Proxy.Port != 8787 {
		t.Errorf("proxy.port default: got %d, want 8787", c.Proxy.Port)
	}
	if c.Proxy.Mode != "cache" {
		t.Errorf("proxy.mode default: got %q, want cache", c.Proxy.Mode)
	}
	if c.Proxy.NonLoopbackStatus != 404 {
		t.Errorf("proxy.non_loopback_status default: got %d, want 404", c.Proxy.NonLoopbackStatus)
	}
	if c.Proxy.ProviderRetries != 3 {
		t.Errorf("proxy.provider_retries default: got %d, want 3", c.Proxy.ProviderRetries)
	}
	if c.Proxy.ProviderRetryBackoffMs != 200 {
		t.Errorf("proxy.provider_retry_backoff_ms default: got %d, want 200", c.Proxy.ProviderRetryBackoffMs)
	}
	if len(c.Proxy.PassthroughProtocols) != 4 {
		t.Errorf("proxy.passthrough_protocols default: got %d entries, want 4", len(c.Proxy.PassthroughProtocols))
	}
	if !c.Proxy.FailOpen {
		t.Errorf("proxy.fail_open default: got false, want true (ADR-0016 fail-open)")
	}

	// wrap
	if c.Wrap.Enabled != false {
		t.Errorf("wrap.enabled default: got %v, want false (opt-in)", c.Wrap.Enabled)
	}
	if len(c.Wrap.Agents) != 4 {
		t.Errorf("wrap.agents default: got %d, want 4 (core-4: opencode,claude,codex,cursor)", len(c.Wrap.Agents))
	}
	for _, want := range []string{"opencode", "claude", "codex", "cursor"} {
		found := false
		for _, a := range c.Wrap.Agents {
			if a == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("wrap.agents default missing %q: got %v", want, c.Wrap.Agents)
		}
	}

	// judgment
	if c.Judgment.Model != "tev1:0.8b" {
		t.Errorf("judgment.model default: got %q, want tev1:0.8b", c.Judgment.Model)
	}
	if c.Judgment.Endpoint != "http://127.0.0.1:11434/v1/systemone" {
		t.Errorf("judgment.endpoint default: got %q, want http://127.0.0.1:11434/v1/systemone", c.Judgment.Endpoint)
	}
	if c.Judgment.Threshold != 0.22 {
		t.Errorf("judgment.threshold default: got %f, want 0.22", c.Judgment.Threshold)
	}

	// traffic learning
	if c.Traffic.Enabled != false {
		t.Errorf("traffic.enabled default: got %v, want false (opt-in)", c.Traffic.Enabled)
	}
	if c.Traffic.MinConfidence != 0.6 {
		t.Errorf("traffic.min_confidence default: got %f, want 0.6", c.Traffic.MinConfidence)
	}
	if c.Traffic.MaxMemories != 500 {
		t.Errorf("traffic.max_memories default: got %d, want 500", c.Traffic.MaxMemories)
	}

	// observability
	if c.Observability.Enabled != true {
		t.Errorf("observability.enabled default: got %v, want true", c.Observability.Enabled)
	}
	if c.Observability.MetricsPath != "/metrics" {
		t.Errorf("observability.metrics_path default: got %q, want /metrics", c.Observability.MetricsPath)
	}
}

// TestRepoLocalWinsOverMachineLocalOverBuiltin is ctx-config-precedence:
// repo-local key beats machine-local beats builtin (per-key).
func TestRepoLocalWinsOverMachineLocalOverBuiltin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()

	// Machine-local: sets port + mode.
	writeCtxYAML(t, filepath.Join(home, ".skillgrid"), "proxy:\n  port: 9001\n  mode: token\n")

	// Repo-local: sets port (overriding machine) but NOT mode.
	writeCtxYAML(t, repo, "proxy:\n  port: 9002\n")

	c, err := load(t, repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Proxy.Port != 9002 {
		t.Errorf("repo-local port: got %d, want 9002 (repo wins)", c.Proxy.Port)
	}
	if c.Proxy.Mode != "token" {
		t.Errorf("mode should fall to machine-local: got %q, want token", c.Proxy.Mode)
	}
	// Keys neither file sets keep their builtin defaults.
	if c.Proxy.Bind != "127.0.0.1" {
		t.Errorf("bind default should survive: got %q, want 127.0.0.1", c.Proxy.Bind)
	}
}

// TestMachineLocalWinsOverBuiltinWhenNoRepoLocal is ctx-config-precedence:
// absent repo-local falls to machine-local.
func TestMachineLocalWinsOverBuiltinWhenNoRepoLocal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir() // no config.d/ctx.yaml here

	writeCtxYAML(t, filepath.Join(home, ".skillgrid"), "proxy:\n  port: 9010\n  mode: passthrough\n")

	c, err := load(t, repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Proxy.Port != 9010 {
		t.Errorf("machine-local port: got %d, want 9010", c.Proxy.Port)
	}
	if c.Proxy.Mode != "passthrough" {
		t.Errorf("machine-local mode: got %q, want passthrough", c.Proxy.Mode)
	}
}

// TestNeitherFilePresentUsesDefaults is ctx-config-precedence:
// neither file present -> all defaults.
func TestNeitherFilePresentUsesDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // empty home
	repo := t.TempDir()           // empty repo
	c, err := load(t, repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	d := DefaultCtxConfig()
	if !reflect.DeepEqual(c.Proxy, d.Proxy) {
		t.Errorf("proxy: got %+v, want default %+v", c.Proxy, d.Proxy)
	}
	if !reflect.DeepEqual(c.Wrap, d.Wrap) {
		t.Errorf("wrap: got %+v, want default %+v", c.Wrap, d.Wrap)
	}
	if c.Judgment != d.Judgment {
		t.Errorf("judgment: got %+v, want default %+v", c.Judgment, d.Judgment)
	}
	if !reflect.DeepEqual(c.Traffic, d.Traffic) {
		t.Errorf("traffic: got %+v, want default %+v", c.Traffic, d.Traffic)
	}
	if !reflect.DeepEqual(c.Observability, d.Observability) {
		t.Errorf("observability: got %+v, want default %+v", c.Observability, d.Observability)
	}
}

// TestRepoLocalFromSubdir verifies the walking-up discovery: a ctx.yaml in a
// repo root is found when Load is called from a subdirectory.
func TestRepoLocalFromSubdir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	writeCtxYAML(t, repo, "judgment:\n  model: clef-flash\n")
	sub := filepath.Join(repo, "deep", "nested", "dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	c, err := load(t, sub)
	if err != nil {
		t.Fatalf("Load from subdir: %v", err)
	}
	if c.Judgment.Model != "clef-flash" {
		t.Errorf("repo-local judgment.model from subdir: got %q, want clef-flash", c.Judgment.Model)
	}
}

// TestMalformedYAMLFallsBackToDefaults verifies a bad machine-local file does
// not abort the load — the per-key merge skips unparseable files.
func TestMalformedYAMLFallsBackToDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	// Deliberately invalid YAML (unbalanced mapping).
	if err := os.MkdirAll(filepath.Join(home, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.d", "ctx.yaml"),
		[]byte("proxy: {port: \n  mode:"), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := load(t, repo)
	if err != nil {
		t.Fatalf("Load with malformed machine file must not error: %v", err)
	}
	if c.Proxy.Port != 8787 {
		t.Errorf("malformed file: port default should survive, got %d, want 8787", c.Proxy.Port)
	}
}

// TestCLM_StillResolvesFromSkillgridConfig is the sentinel for
// config-clm-unchanged: the clm block is STILL read from .skillgrid/config.yaml
// by the existing internal/config loader — not moved into ctx.yaml. This
// asserts the existing loader still resolves a sentinel clm value, proving no
// regression from introducing the new ctx config package.
func TestCLM_StillResolvesFromSkillgridConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := loadCLMFromSkillgridConfig(t, t.TempDir())
	if !cfg.Enabled {
		t.Fatalf("clm.enabled sentinel from .skillgrid/config.yaml: got false, want true")
	}
	if cfg.Budget != 64000 {
		t.Fatalf("clm.budget sentinel from .skillgrid/config.yaml: got %d, want 64000", cfg.Budget)
	}
	if cfg.Reserve != 2048 {
		t.Fatalf("clm.reserve sentinel from .skillgrid/config.yaml: got %d, want 2048 (default)", cfg.Reserve)
	}
}

// TestJudgmentEndpointLoopbackOnly is the guard for the ADR-0035 "local
// Ollama" trust boundary: a non-loopback judgment.endpoint — especially from a
// repo-local ctx.yaml (the layer that wins per key) — would exfiltrate message
// content to a remote host, so the loader must ignore non-loopback overrides.
func TestJudgmentEndpointLoopbackOnly(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		wantKeep bool // true = the override is honored
	}{
		{"loopback-ipv4", "http://127.0.0.1:9999/v1/systemone", true},
		{"loopback-ipv6", "http://[::1]:9999/v1/systemone", true},
		{"remote-hostname", "http://ollama.example.com:11434/v1/systemone", false},
		{"remote-ipv4", "http://10.0.0.5:11434/v1/systemone", false},
		{"no-host", "http:///v1/systemone", false},
		{"malformed", "not a url", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			start := t.TempDir()
			dir := filepath.Join(start, "config.d")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			yaml := "judgment:\n  endpoint: " + tc.endpoint + "\n"
			if err := os.WriteFile(filepath.Join(dir, "ctx.yaml"), []byte(yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			c, err := load(t, start)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			def := DefaultCtxConfig().Judgment.Endpoint
			if tc.wantKeep {
				if c.Judgment.Endpoint != tc.endpoint {
					t.Fatalf("endpoint = %q, want the loopback override %q", c.Judgment.Endpoint, tc.endpoint)
				}
			} else {
				if c.Judgment.Endpoint != def {
					t.Fatalf("endpoint = %q, want the default %q (non-loopback override must be ignored)", c.Judgment.Endpoint, def)
				}
			}
		})
	}
}
