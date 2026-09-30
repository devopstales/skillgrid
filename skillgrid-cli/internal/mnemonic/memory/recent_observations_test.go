package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func TestRecentObservations(t *testing.T) {
	st, err := store.Open(t.TempDir(), "tracer")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	svc := New(st, "tracer")
	ctx := context.Background()

	sid := "sess-ro"
	base := time.Now().UTC().Add(-time.Hour)
	if _, err := svc.DB().ExecContext(ctx,
		`INSERT OR REPLACE INTO sessions (id, project, directory, started_at) VALUES (?,?,?,?)`,
		sid, "tracer", t.TempDir(), base.Format(time.RFC3339)); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	for i := 1; i <= 5; i++ {
		created := base.Add(time.Duration(i) * time.Minute)
		if _, err := svc.DB().ExecContext(ctx,
			`INSERT INTO observations (session_id, type, title, content, project, scope,
			   normalized_hash, revision_count, created_at, updated_at, source, visibility, status)
			   VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			sid, "learning", fmt.Sprintf("obs-%d", i), fmt.Sprintf("content %d", i),
			"tracer", "project", fmt.Sprintf("hash-%d", i), 0,
			created.Format(time.RFC3339), created.Format(time.RFC3339), "test", "team", "active"); err != nil {
			t.Fatalf("insert obs %d: %v", i, err)
		}
	}

	t.Run("desc order and limit", func(t *testing.T) {
		got, err := svc.RecentObservations(ctx, 3)
		if err != nil {
			t.Fatalf("RecentObservations: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("want 3 obs, got %d", len(got))
		}
		// Newest first: obs-5, obs-4, obs-3.
		if got[0].Title != "obs-5" || got[1].Title != "obs-4" || got[2].Title != "obs-3" {
			t.Errorf("wrong order: got %q, %q, %q", got[0].Title, got[1].Title, got[2].Title)
		}
	})

	t.Run("excludes deleted", func(t *testing.T) {
		created := base.Add(10 * time.Minute)
		if _, err := svc.DB().ExecContext(ctx,
			`INSERT INTO observations (session_id, type, title, content, project, scope,
			   normalized_hash, revision_count, created_at, updated_at, source, visibility, status, deleted_at)
			   VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			sid, "learning", "deleted-obs", "gone", "tracer", "project", "hash-del", 0,
			created.Format(time.RFC3339), created.Format(time.RFC3339), "test", "team", "active",
			created.Format(time.RFC3339)); err != nil {
			t.Fatalf("insert deleted obs: %v", err)
		}
		got, err := svc.RecentObservations(ctx, 10)
		if err != nil {
			t.Fatalf("RecentObservations: %v", err)
		}
		for _, o := range got {
			if o.Title == "deleted-obs" {
				t.Errorf("deleted observation leaked into RecentObservations")
			}
		}
	})

	t.Run("project bound", func(t *testing.T) {
		if _, err := svc.DB().ExecContext(ctx,
			`INSERT INTO observations (session_id, type, title, content, project, scope,
			   normalized_hash, revision_count, created_at, updated_at, source, visibility, status)
			   VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			sid, "learning", "other-project", "x", "otherproj", "project", "hash-other", 0,
			base.Format(time.RFC3339), base.Format(time.RFC3339), "test", "team", "active"); err != nil {
			t.Fatalf("insert other-project obs: %v", err)
		}
		got, err := svc.RecentObservations(ctx, 10)
		if err != nil {
			t.Fatalf("RecentObservations: %v", err)
		}
		for _, o := range got {
			if o.Title == "other-project" {
				t.Errorf("observation from another project leaked into RecentObservations")
			}
		}
	})
}
