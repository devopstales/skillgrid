package http

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// TestCLMRevision_FailOpenDefault verifies the CLM capture endpoint is
// fail-open with CLM disabled (the default): a malformed body, a valid body,
// and an empty body all respond 200 {"ok":true} and persist nothing.
func TestCLMRevision_FailOpenDefault(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()

	// Malformed JSON body: fail-open 200.
	rr, out := do(t, s.Handler(), http.MethodPost, "/sessions/s1/clm/revision?project="+proj, map[string]any{"messages": []string{"a"}})
	if rr.Code != http.StatusOK {
		t.Fatalf("malformed: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if out["ok"] != true {
		t.Errorf("malformed: expected ok=true, got %v", out)
	}

	// Valid body while CLM is off (default): 200, no revision persisted.
	rr, out = do(t, s.Handler(), http.MethodPost, "/sessions/s1/clm/revision?project="+proj, map[string]any{
		"document": "doc1",
		"messages": []string{"one", "two"},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("valid-off: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, present := out["revision"]; present {
		t.Errorf("valid-off: expected no revision key while CLM is off, got %v", out)
	}

	// Empty body: fail-open 200.
	rr, out = do(t, s.Handler(), http.MethodPost, "/sessions/s1/clm/revision?project="+proj, map[string]any{})
	if rr.Code != http.StatusOK {
		t.Fatalf("empty: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if out["ok"] != true {
		t.Errorf("empty: expected ok=true, got %v", out)
	}
}

// TestCLMRevision_PersistsWhenEnabled turns CLM on in the config and asserts
// a valid mirror edit is persisted as revision 1 (and a second edit as
// revision 2), proving the happy path stores the validated edit.
func TestCLMRevision_PersistsWhenEnabled(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()

	// Enable CLM in the config the test cwd resolves to (config.Load searches
	// up from the cwd, which is the temp dir newToolCallsServer chdir'd into).
	cfgDir := filepath.Join(".", ".skillgrid", "config.d")
	_ = os.MkdirAll(cfgDir, 0o755)
	_ = os.WriteFile(filepath.Join(cfgDir, "indexing.yaml"), []byte("mnemonic:\n  hooks:\n    enabled: true\n  clm:\n    enabled: true\n    budget: 1000\n"), 0o644)

	post := map[string]any{"document": "doc1", "messages": []string{"one", "two"}}
	rr, out := do(t, s.Handler(), http.MethodPost, "/sessions/s1/clm/revision?project="+proj, post)
	if rr.Code != http.StatusOK {
		t.Fatalf("first: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if rev, ok := out["revision"].(float64); !ok || rev != 1 {
		t.Fatalf("first: expected revision 1, got %v", out["revision"])
	}

	rr, out = do(t, s.Handler(), http.MethodPost, "/sessions/s1/clm/revision?project="+proj, post)
	if rr.Code != http.StatusOK {
		t.Fatalf("second: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if rev, ok := out["revision"].(float64); !ok || rev != 2 {
		t.Fatalf("second: expected revision 2, got %v", out["revision"])
	}
}
