package clm

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func TestResume_NoRevisions(t *testing.T) {
	st, err := store.Open(t.TempDir(), "rproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	rev, err := Resume(context.Background(), st.DB, "s1")
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if rev != nil {
		t.Fatalf("expected nil revision, got %+v", rev)
	}
}

func TestResume_HighestRevision(t *testing.T) {
	st, err := store.Open(t.TempDir(), "rproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	_, err = st.DB.Exec(`INSERT INTO context_revisions (id, session_id, project_id, revision, anchor_count, anchor_digest, messages, size_estimate, calibration_factor, created_at)
		VALUES (1,'s1','rproj',1,1,'d1','m1',100,4.0,'2026-10-06T00:00:00Z'),
		        (2,'s1','rproj',2,2,'d2','m2',200,4.0,'2026-10-06T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	rev, err := Resume(context.Background(), st.DB, "s1")
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if rev == nil || rev.Revision != 2 {
		t.Fatalf("expected revision 2, got %+v", rev)
	}
	if rev.AnchorDigest != "d2" {
		t.Fatalf("expected digest d2, got %q", rev.AnchorDigest)
	}
}
