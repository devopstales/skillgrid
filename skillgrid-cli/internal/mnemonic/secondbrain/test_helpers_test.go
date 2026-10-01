package secondbrain

import (
	"context"
	"fmt"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// newTestService builds a real *service.Service over a temp data dir. When
// embedder is true, MNEMONIC_EMBED is on so BlendedSearch's vector leg is not
// structurally degraded; a seeded query-aligned vector then drives a true
// hybrid (non-degraded) result.
func newTestService(t *testing.T, embedder bool) *service.Service {
	t.Helper()
	if embedder {
		t.Setenv("MNEMONIC_EMBED", "1")
	}
	return service.New(t.TempDir())
}

// ensureSession creates the session row observations reference (session_id FK).
// Direct SQL mirrors the house pattern in memory/types_test.go: SessionStart
// would re-resolve the project from CWD and not land in the fixture bucket.
func ensureSession(t *testing.T, svc *service.Service, project string) {
	t.Helper()
	hh, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	if _, err := hh.Store().DB.Exec(
		`INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		 VALUES (?, ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, sessionA, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
}

// seedEmbedded stores n observations in project, embeds each with a hash
// embedder, and attaches that embedder as the handle's dir embedder — so
// queryVector derives a query-aligned vector and BlendedSearch reports
// degraded=false (hybrid).
func seedEmbedded(t *testing.T, svc *service.Service, project string, n int) {
	t.Helper()
	ctx := context.Background()
	he := embedder.HashEmbedder{Dim: 64}
	ensureSession(t, svc, project)
	// Set the override BEFORE opening so the freshly-opened handle (and the
	// one AskCited opens) both carry the dir embedder — openProject forwards
	// it to every memory service it mints.
	svc.SetDirEmbedderOverride(he)
	hh, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	for i := 0; i < n; i++ {
		content := fmt.Sprintf("authentication content %d", i)
		id, err := hh.Memory().Save(ctx, memory.SaveInput{
			Title:     fmt.Sprintf("authentication design %d", i),
			Type:      "decision",
			Content:   content,
			Scope:     "project",
			SessionID: sessionA,
		})
		if err != nil {
			t.Fatalf("seed embedded save %d: %v", i, err)
		}
		vec, err := he.Embed(ctx, content)
		if err != nil {
			t.Fatalf("embed %d: %v", i, err)
		}
		if err := hh.Memory().SetEmbedding(ctx, id, memory.EncodeVector(vec), "hash-embedder-v1"); err != nil {
			t.Fatalf("set embedding %d: %v", i, err)
		}
	}
}
