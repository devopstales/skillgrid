package memory

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestCLMConfigDefault(t *testing.T) {
	st, err := store.Open(t.TempDir(), "clmproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	svc := New(st, "clmproj")
	cfg := svc.CLMConfig()
	if cfg.Enabled {
		t.Fatalf("expected CLM disabled by default, got %+v", cfg)
	}
	if cfg.Reserve != 0 {
		t.Fatalf("expected zero-value Reserve for a default-constructed service, got %d", cfg.Reserve)
	}
}

func TestCLMConfigSetCLM(t *testing.T) {
	st, err := store.Open(t.TempDir(), "clmproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	svc := New(st, "clmproj")
	want := config.CLM{Enabled: true, Budget: 1000, Reserve: 512, Guard: true}
	svc.SetCLM(want)
	got := svc.CLMConfig()
	if got != want {
		t.Fatalf("SetCLM did not store the config: got %+v want %+v", got, want)
	}
}
