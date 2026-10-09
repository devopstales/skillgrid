package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCLMConfig(t *testing.T, dir, body string) string {
	t.Helper()
	cfgDir := filepath.Join(dir, ".skillgrid", "config.d")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(cfgDir, "indexing.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func clearCLMEnv(t *testing.T) {
	t.Setenv("SKILLGRID_CTX_CLM", "")
	t.Setenv("SKILLGRID_CTX_CLM_BUDGET", "")
	t.Setenv("SKILLGRID_CTX_CLM_RESERVE", "")
}

// TestDefaultCLM_Off is TICKET-07 (clm-off-by-default): the production default
// is OFF (opt-in) with a 2048-token reserve and the guard on.
func TestDefaultCLM_Off(t *testing.T) {
	c := DefaultCLM()
	if c.Enabled {
		t.Fatalf("CLM must be off by default")
	}
	if c.Budget != 0 {
		t.Fatalf("CLM budget default: got %d, want 0 (model window)", c.Budget)
	}
	if c.Reserve != 2048 {
		t.Fatalf("reserve default: got %d, want 2048", c.Reserve)
	}
	if c.Reminders != "50/75/90" {
		t.Fatalf("reminders default: got %q, want 50/75/90", c.Reminders)
	}
	if !c.Guard {
		t.Fatalf("guard default: must be true")
	}
	if c.Cap != 0 {
		t.Fatalf("cap default: got %d, want 0 (off)", c.Cap)
	}
}

// TestLoadCLM_AbsentBlock is TICKET-07 (clm-off-by-default): with no clm block
// and no env overrides, Load returns the production defaults (off, reserve
// 2048, guard on).
func TestLoadCLM_AbsentBlock(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	clearCLMEnv(t)
	got := Load(t.TempDir()).CLM
	want := DefaultCLM()
	if got != want {
		t.Fatalf("absent clm block: got %+v, want %+v", got, want)
	}
}

// TestLoadCLM_ExplicitOn is TICKET-07 (clm-config-honored): an explicit
// clm.enabled: true + budget are honored from config, and the reserve default
// (2048) survives an explicit budget.
func TestLoadCLM_ExplicitOn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	clearCLMEnv(t)
	dir := t.TempDir()
	writeCLMConfig(t, dir, "mnemonic:\n  clm:\n    enabled: true\n    budget: 64000\n")
	c := Load(dir).CLM
	if !c.Enabled {
		t.Fatalf("clm.enabled not honored")
	}
	if c.Budget != 64000 {
		t.Fatalf("clm.budget not honored: got %d, want 64000", c.Budget)
	}
	if c.Reserve != 2048 {
		t.Fatalf("reserve default should survive explicit budget: got %d, want 2048", c.Reserve)
	}
	if !c.Guard {
		t.Fatalf("guard default should survive: got false, want true")
	}
}

// TestLoadCLM_FullBlock is TICKET-07: a complete clm block (enabled, budget,
// reserve, reminders, guard, cap) parses through.
func TestLoadCLM_FullBlock(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	clearCLMEnv(t)
	dir := t.TempDir()
	writeCLMConfig(t, dir, "mnemonic:\n  clm:\n    enabled: true\n    budget: 4000\n    reserve: 1024\n    reminders: \"40/80\"\n    guard: false\n    cap: 2\n")
	c := Load(dir).CLM
	if !c.Enabled {
		t.Fatalf("clm.enabled: true must apply")
	}
	if c.Budget != 4000 {
		t.Fatalf("clm.budget: got %d, want 4000", c.Budget)
	}
	if c.Reserve != 1024 {
		t.Fatalf("clm.reserve: got %d, want 1024", c.Reserve)
	}
	if c.Reminders != "40/80" {
		t.Fatalf("clm.reminders: got %q, want 40/80", c.Reminders)
	}
	if c.Guard {
		t.Fatalf("clm.guard: false must opt out, got true")
	}
	if c.Cap != 2 {
		t.Fatalf("clm.cap: got %d, want 2", c.Cap)
	}
}

// TestLoadCLM_ExplicitOff is TICKET-07: an explicit clm.enabled: false stays
// off (enabled is opt-in; the default is already off).
func TestLoadCLM_ExplicitOff(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	clearCLMEnv(t)
	dir := t.TempDir()
	writeCLMConfig(t, dir, "mnemonic:\n  clm:\n    enabled: false\n")
	if got := Load(dir).CLM; got.Enabled {
		t.Fatalf("clm.enabled: false must stay off")
	}
}

// TestLoadCLM_EnvOverrideWins is TICKET-07 (clm-env-override-wins): the
// SKILLGRID_CTX_CLM / _BUDGET / _RESERVE env vars, when present and
// parseable, WIN over the config value.
func TestLoadCLM_EnvOverrideWins(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	writeCLMConfig(t, dir, "mnemonic:\n  clm:\n    enabled: true\n    budget: 4000\n    reserve: 1024\n")

	// Env off wins over config enabled:true; the int env vars win over config.
	t.Setenv("SKILLGRID_CTX_CLM", "off")
	t.Setenv("SKILLGRID_CTX_CLM_BUDGET", "8000")
	t.Setenv("SKILLGRID_CTX_CLM_RESERVE", "512")
	c := Load(dir).CLM
	if c.Enabled {
		t.Fatalf("SKILLGRID_CTX_CLM=off must win over config enabled:true")
	}
	if c.Budget != 8000 {
		t.Fatalf("SKILLGRID_CTX_CLM_BUDGET must win, got %d, want 8000", c.Budget)
	}
	if c.Reserve != 512 {
		t.Fatalf("SKILLGRID_CTX_CLM_RESERVE must win, got %d, want 512", c.Reserve)
	}

	// Clear the int overrides, then turn CLM on with SKILLGRID_CTX_CLM=1 even
	// though the config block says enabled: false.
	t.Setenv("SKILLGRID_CTX_CLM", "1")
	t.Setenv("SKILLGRID_CTX_CLM_BUDGET", "")
	t.Setenv("SKILLGRID_CTX_CLM_RESERVE", "")
	if got := Load(dir).CLM; !got.Enabled {
		t.Fatalf("SKILLGRID_CTX_CLM=1 must turn CLM on even over config enabled:false")
	}
}

// TestLoadCLM_EnvMalformedKeepsConfig is TICKET-07: a malformed env int
// (unparseable) does NOT override — the config value survives.
func TestLoadCLM_EnvMalformedKeepsConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	writeCLMConfig(t, dir, "mnemonic:\n  clm:\n    budget: 4000\n    reserve: 1024\n")

	// No on/off env; a malformed budget env keeps the config value.
	t.Setenv("SKILLGRID_CTX_CLM", "")
	t.Setenv("SKILLGRID_CTX_CLM_BUDGET", "not-a-number")
	t.Setenv("SKILLGRID_CTX_CLM_RESERVE", "")
	c := Load(dir).CLM
	if c.Budget != 4000 {
		t.Fatalf("malformed budget env must keep config value, got %d, want 4000", c.Budget)
	}
	if c.Reserve != 1024 {
		t.Fatalf("reserve env absent must keep config value, got %d, want 1024", c.Reserve)
	}
}
