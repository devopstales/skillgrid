package extract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// lspFixture writes a Go file whose member call (`s.Do()`) the static pass
// cannot type: the receiver `s` is a locally-typed value, so the static
// extract keeps the call name-only (the LSP tier has something to resolve).
func lspFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	main := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\ts := newSvc()\n\ts.Do(1)\n\tfmt.Println(s)\n}\n\nfunc newSvc() *Svc { return &Svc{} }\n\nfunc (x *Svc) Do(n int) int { return n }\n\ntype Svc struct{}\n"
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(main), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return root
}

// TestLSPAllOrNothingPerCallSite covers the lsp-all-or-nothing-per-call-site
// scenario: a resolved member call → an LSP_RESOLVED edge; a failing/timing-
// out resolver → the static name-only edge stands, never a partial edge set
// (a resolver error mid-run clears the whole run's resolved set).
func TestLSPAllOrNothingPerCallSite(t *testing.T) {
	ctx := context.Background()

	// (1) A resolver that resolves the call → an LSP_RESOLVED edge.
	client := NewLSPClientForTest(LSPClientOptions{
		Timeout: 2 * time.Second,
		Files:   func() []LSPFile { return []LSPFile{{Path: "main.go", Contents: []byte(lspGoSource())}} },
	}, func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool, error) {
		if receiver == "s" && callee == "Do" {
			return "Do", true, nil
		}
		return "", false, nil
	})
	edges, err := client.ResolveMemberCalls(ctx)
	if err != nil {
		t.Fatalf("resolve (good resolver): %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("a resolved call → exactly one LSP edge, got %d: %+v", len(edges), edges)
	}
	if edges[0].Callee != "Do" || edges[0].Receiver != "s" {
		t.Fatalf("the resolved edge carries the resolved callee + receiver, got %+v", edges[0])
	}

	// (2) A resolver that ERRORS mid-run → NO edges (all-or-nothing: the
	// static name-only edge stands, never a partial edge set).
	failing := NewLSPClientForTest(LSPClientOptions{
		Timeout: 2 * time.Second,
		Files:   func() []LSPFile { return []LSPFile{{Path: "main.go", Contents: []byte(lspGoSource())}} },
	}, func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool, error) {
		return "", false, fmt.Errorf("server round-trip failed")
	})
	if _, err := failing.ResolveMemberCalls(ctx); err == nil {
		t.Fatalf("a failing server must be a best-effort no-op (an error → no partial edge set)")
	}

	// (3) The timeout path: a hanging resolver (blocks past the per-call
	// timeout) → an error (the bounded ctx fires — the resolver reports it),
	// no partial edge set.
	hanging := NewLSPClientForTest(LSPClientOptions{
		Timeout: 20 * time.Millisecond, // the per-call bound (the production default is 30 s)
		Files:   func() []LSPFile { return []LSPFile{{Path: "main.go", Contents: []byte(lspGoSource())}} },
	}, func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool, error) {
		<-ctx.Done() // the resolver hangs until the per-call timeout fires
		return "", false, ctx.Err()
	})
	if _, err := hanging.ResolveMemberCalls(ctx); err == nil {
		t.Fatalf("a timing-out server must be a best-effort no-op (an error → no partial edge set)")
	}
}

// lspGoSource is the fixture source (shared by the resolve tests).
func lspGoSource() string {
	return "package main\n\nimport \"fmt\"\n\nfunc main() {\n\ts := newSvc()\n\ts.Do(1)\n\tfmt.Println(s)\n}\n\nfunc newSvc() *Svc { return &Svc{} }\n\nfunc (x *Svc) Do(n int) int { return n }\n\ntype Svc struct{}\n"
}

// TestLSPMemberCallSites covers the call-site walk: a receiver-bound call in
// the fixture is a member call site (the LSP tier's input); a plain call is
// not.
func TestLSPMemberCallSites(t *testing.T) {
	sites := MemberCallSites("main.go", []byte(lspGoSource()))
	if len(sites) == 0 {
		t.Fatalf("the fixture's s.Do() is a member call site (the LSP tier's input)")
	}
	found := false
	for _, s := range sites {
		if s.Receiver == "s" && s.Callee == "Do" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the s.Do() site must be walked (receiver=s, callee=Do), got %+v", sites)
	}
	// An unmapped extension has no sites (the language is not LSP-covered).
	if got := MemberCallSites("a.txt", []byte("s.Do(1)")); got != nil {
		t.Fatalf("an unmapped extension has no member call sites, got %+v", got)
	}
}

// TestLSPWarnsOnAbsentServer covers the best-effort contract: no server on
// PATH (an isolated empty PATH) → NewLSPClient errors (the "no language
// server on PATH" the handler maps to degraded=lsp_unavailable) and the
// pass warns + continues (the static index stands).
func TestLSPWarnsOnAbsentServer(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // an empty PATH → no servers
	_, err := NewLSPClient(LSPClientOptions{
		Files: func() []LSPFile { return []LSPFile{{Path: "main.go", Contents: []byte(lspGoSource())}} },
	})
	if err == nil {
		t.Fatalf("no server on PATH → the client must error (the best-effort no-op signal)")
	}
	if err.Error() != "no language server on PATH" {
		t.Fatalf("the absent-server error is the best-effort signal, got: %v", err)
	}
	// The default timeout is 30 s (the per-call bound, CI-6).
	if DefaultLSPTimeout != 30*time.Second {
		t.Fatalf("the per-call timeout is 30 s (CI-6), got %v", DefaultLSPTimeout)
	}
}

// TestLSPConfidenceTier covers the 4th confidence tier: a resolved member
// call is the LSP_RESOLVED edge (the label the driver's upsert writes).
func TestLSPConfidenceTier(t *testing.T) {
	if LSPConfidence != "LSP_RESOLVED" {
		t.Fatalf("the resolved member call is the LSP_RESOLVED edge (the 4th confidence tier), got %q", LSPConfidence)
	}
}

var _ = fmt.Sprintf // kept for the fixture formatting (the tests' log lines)
