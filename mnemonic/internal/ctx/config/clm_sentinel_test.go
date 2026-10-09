package config

import (
	"os"
	"path/filepath"
	"testing"

	// The existing skillgrid config loader — the clm: block still lives here,
	// read from .skillgrid/config.yaml (ADR-0027). This import is the sentinel:
	// it proves the new ctx config package did not move the clm block.
	skillgridconfig "github.com/devopstales/skillgrid/mnemonic/internal/config"
)

// loadCLMFromSkillgridConfig writes a clm block into the repo's
// .skillgrid/config.yaml (the config.d indexing.yaml location the existing
// loader reads) and returns the resolved CLM value. This is the
// config-clm-unchanged sentinel: the clm block is still resolved by the
// existing internal/config loader, untouched by the new ctx config package.
func loadCLMFromSkillgridConfig(t *testing.T, dir string) skillgridconfig.CLM {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "mnemonic:\n  clm:\n    enabled: true\n    budget: 64000\n"
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// Clear the CLM env overrides so only the config value is observed.
	t.Setenv("SKILLGRID_CTX_CLM", "")
	t.Setenv("SKILLGRID_CTX_CLM_BUDGET", "")
	t.Setenv("SKILLGRID_CTX_CLM_RESERVE", "")
	return skillgridconfig.Load(dir).CLM
}
