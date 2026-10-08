package context_harness

import (
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestDecideGate_SmallPasses(t *testing.T) {
	d := DecideGate("short output", false, SandboxThreshold)
	if d.Gate {
		t.Fatalf("small output should not gate")
	}
}

func TestDecideGate_LargeGates(t *testing.T) {
	out := strings.Repeat("abcdefgh ", 625)
	d := DecideGate(out, false, SandboxThreshold)
	if !d.Gate {
		t.Fatalf("large output should gate")
	}
	if len(d.Summary) > 200 {
		t.Fatalf("summary too long: %d", len(d.Summary))
	}
	if d.Pointer == "" {
		t.Fatalf("expected ctx_search pointer")
	}
}

func TestDecideGate_Bypass(t *testing.T) {
	d := DecideGate(string(make([]byte, 5000)), true, SandboxThreshold)
	if d.Gate {
		t.Fatalf("bypass should not gate")
	}
}

func TestGateStores(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	out := "line one\nline two\n" + strings.Repeat("abcdefgh ", 625)
	id, err := StoreToolOutput(t.Context(), st.DB, "s1", "ctxproj", "bash", out)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}
	var stored int
	err = st.DB.QueryRow(`SELECT length(output) FROM tool_outputs WHERE id = ?`, id).Scan(&stored)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if stored != len(out) {
		t.Fatalf("stored length %d != %d", stored, len(out))
	}
	var fts int
	err = st.DB.QueryRow(`SELECT count(*) FROM tool_outputs_fts WHERE tool_outputs_fts MATCH 'sandbox' OR tool_outputs_fts MATCH 'line'`).Scan(&fts)
	if err != nil {
		t.Fatalf("fts: %v", err)
	}
	if fts < 1 {
		t.Fatalf("expected fts hit for stored output, got %d", fts)
	}
}
