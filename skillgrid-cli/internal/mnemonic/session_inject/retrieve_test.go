package session_inject

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedObservations inserts n FTS-indexable observations into the project
// bucket of dir (dir/project.sqlite); the FTS trigger indexes them on insert.
func seedObservations(t *testing.T, dir, project string, n int) {
	t.Helper()
	st, err := store.Open(dir, project)
	if err != nil {
		t.Fatalf("open store %s: %v", project, err)
	}
	t.Cleanup(func() { st.Close() })
	seedObservationsStore(t, st, project, n)
}

func seedObservationsStore(t *testing.T, st *store.Store, project string, n int) {
	t.Helper()
	for i := 1; i <= n; i++ {
		seedObservation(t, st, project,
			fmt.Sprintf("auth note %d", i),
			fmt.Sprintf("authentication token rotation detail %d", i))
	}
}

func memoryNew(dir, project string) *memory.Service {
	st, err := store.Open(dir, project)
	if err != nil {
		panic(err)
	}
	return memory.New(st, project)
}

func encodeVectorBytes(v memory.Vector) []byte {
	return memory.EncodeVector(v)
}

func seedObservation(t *testing.T, st *store.Store, project, title, content string) {
	t.Helper()
	seedObservationAt(t, st, project, title, content, time.Now().UTC())
}

// seedObservationAt is seedObservation with an explicit timestamp, so tests
// can make one observation strictly older than another (FTS ties break on
// created_at, which is what cross-bucket RRF rank order depends on).
func seedObservationAt(t *testing.T, st *store.Store, project, title, content string, at time.Time) {
	t.Helper()
	ctx := context.Background()
	now := at.UTC().Format(time.RFC3339)
	if _, err := st.DB.ExecContext(ctx,
		`INSERT OR REPLACE INTO sessions (id, project, directory, started_at) VALUES (?,?,?,?)`,
		"sess-t3", project, t.TempDir(), now); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := st.DB.ExecContext(ctx,
		`INSERT INTO observations (session_id, type, title, content, project, scope,
		   normalized_hash, revision_count, created_at, updated_at, source, visibility, status)
		   VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"sess-t3", "decision", title, content, project, "project",
		fmt.Sprintf("hash-%d", len(content)+len(title)), 0, now, now, "test", "team", "active"); err != nil {
		t.Fatalf("insert observation: %v", err)
	}
}

func TestHybridRetrieve_BM25Only(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	dir := t.TempDir()
	seedObservations(t, dir, "tracer", 10)
	svc := memoryNew(dir, "tracer")

	res, err := HybridRetrieve(context.Background(), svc, "tracer", "auth", false, 800)
	if err != nil {
		t.Fatalf("HybridRetrieve: %v", err)
	}
	if !res.Degraded {
		t.Error("expected Degraded=true with no embedder configured")
	}
	if len(res.Items) == 0 {
		t.Fatal("expected items from the FTS floor, got none")
	}
	for _, it := range res.Items {
		if it.TokenCost <= 0 {
			t.Errorf("every item must have TokenCost > 0, got %d", it.TokenCost)
		}
	}
}

func TestHybridRetrieve_VectorLeg(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "1")
	dir := t.TempDir()
	seedObservations(t, dir, "tracer", 10)
	svc := memoryNew(dir, "tracer")

	he := embedder.HashEmbedder{Dim: 64}
	svc.SetDirEmbedder(he)
	ctx := context.Background()
	rows, err := svc.DB().QueryContext(ctx,
		`SELECT id, content FROM observations WHERE project = ?`, "tracer")
	if err != nil {
		t.Fatalf("query observations: %v", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		var content string
		if err := rows.Scan(&id, &content); err != nil {
			t.Fatalf("scan observation: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate observations: %v", err)
	}
	for _, id := range ids {
		var content string
		if err := svc.DB().QueryRowContext(ctx,
			`SELECT content FROM observations WHERE id = ?`, id).Scan(&content); err != nil {
			t.Fatalf("re-read content: %v", err)
		}
		vec, err := he.Embed(ctx, content)
		if err != nil {
			t.Fatalf("embed: %v", err)
		}
		if err := svc.SetEmbedding(ctx, id, encodeVectorBytes(vec), "hash-embedder-v1"); err != nil {
			t.Fatalf("SetEmbedding: %v", err)
		}
	}

	res, err := HybridRetrieve(ctx, svc, "tracer", "authentication", false, 800)
	if err != nil {
		t.Fatalf("HybridRetrieve: %v", err)
	}
	if res.Degraded {
		t.Error("expected Degraded=false when the embedder is active")
	}
	if len(res.Items) == 0 {
		t.Fatal("expected items, got none")
	}
}

func TestHybridRetrieve_ProjectScope(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	dir := t.TempDir()
	seedObservations(t, dir, "project-a", 5)
	seedObservations(t, dir, "project-b", 5)
	svc := memoryNew(dir, "project-a")

	res, err := HybridRetrieve(context.Background(), svc, "project-a", "auth", false, 800)
	if err != nil {
		t.Fatalf("HybridRetrieve: %v", err)
	}
	if len(res.Items) == 0 {
		t.Fatal("expected items, got none")
	}
	for _, it := range res.Items {
		if it.Source != "observation" {
			t.Errorf("unexpected source %q", it.Source)
		}
		if it.Project != "project-a" {
			t.Errorf("item leaked outside the scoped project: %q (id %d)", it.Project, it.ID)
		}
	}
}

func TestHybridRetrieve_AllProjects(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	dir := t.TempDir()
	// Seed >= injectSearchLimit (20) project-a hits, all matching "auth" and
	// all timestamped before the project-b hit. Under concatenation the primary
	// top-20 (all project-a) precedes the project-b hit (b's only match, so it
	// ranks last under RRF too); with maxTokens=205 the token budget fits
	// exactly the 20 project-a items (~10 tokens each, 200 total) but drops the
	// 21st (project-b). Cross-bucket RRF re-ranks b (1/61 vs 1/79..1/99) to the
	// front, so b only surfaces via the fusion.
	base := time.Now().UTC().Add(-time.Hour)
	seedObservations(t, dir, "project-a", 20)
	bSt, err := store.Open(dir, "project-b")
	if err != nil {
		t.Fatalf("open store project-b: %v", err)
	}
	t.Cleanup(func() { bSt.Close() })
	seedObservationAt(t, bSt, "project-b",
		"auth vault rotation",
		"authentication token rotation detail vault",
		base.Add(time.Hour).Add(30*time.Second)) // strictly after every project-a row
	svc := memoryNew(dir, "project-a")

	res, err := HybridRetrieve(context.Background(), svc, "project-a", "auth", true, 205)
	if err != nil {
		t.Fatalf("HybridRetrieve: %v", err)
	}
	if len(res.Items) == 0 {
		t.Fatal("expected items spanning projects, got none")
	}
	seen := map[string]bool{}
	sawB := false
	for _, it := range res.Items {
		if it.Project != "project-a" && it.Project != "project-b" {
			t.Errorf("item from unexpected project %q", it.Project)
		}
		seen[it.Project] = true
		if it.Project == "project-b" {
			sawB = true
		}
	}
	if !seen["project-a"] || !seen["project-b"] {
		t.Errorf("expected items from both projects, saw %v", seen)
	}
	// Fusion proof: the project-b hit outranked enough project-a hits to make
	// the fused top 20. Concatenation (primary first, no cross-list RRF) puts
	// all 20 project-a hits before it, and the token cap drops the rest.
	if !sawB {
		t.Error("project-b hit did not surface despite cross-bucket fusion: concatenation would have hidden it")
	}
}
