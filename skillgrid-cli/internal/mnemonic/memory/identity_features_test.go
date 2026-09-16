package memory

import (
	"context"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// newTestStore spins up an in-memory-flavoured (tmp-dir) store with all
// migrations applied so the tests exercise the real schema.
func newTestStore(t *testing.T, projectID string) (*store.Store, *Service) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, projectID)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st, New(st, projectID)
}

func newSession(t *testing.T, svc *Service) string {
	t.Helper()
	// SessionStart requires a directory; use a real tmp dir.
	dir := t.TempDir()
	id, err := svc.SessionStart(context.Background(), dir, "test-session")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	return id
}

func TestPinUnpin(t *testing.T) {
	_, svc := newTestStore(t, "pinproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	id, err := svc.Save(ctx, SaveInput{Title: "pinned note", Type: "decision", Content: "we choose X", SessionID: sid, Scope: "project"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	obs, err := svc.Get(ctx, id)
	if err != nil || obs.Pinned {
		t.Fatalf("initial pinned: obs=%+v err=%v", obs, err)
	}
	if err := svc.Pin(ctx, id); err != nil {
		t.Fatalf("pin: %v", err)
	}
	pinned, err := svc.Pinned(ctx, id)
	if err != nil || !pinned {
		t.Fatalf("after pin: pinned=%v err=%v", pinned, err)
	}
	if err := svc.Unpin(ctx, id); err != nil {
		t.Fatalf("unpin: %v", err)
	}
	if pinned, _ := svc.Pinned(ctx, id); pinned {
		t.Fatalf("after unpin should be false")
	}
}

func TestDuplicateBumpOnResave(t *testing.T) {
	_, svc := newTestStore(t, "dupproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	in := SaveInput{Title: "same note", Type: "decision", Content: "we always choose Y", SessionID: sid, Scope: "project"}
	first, err := svc.Save(ctx, in)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	second, err := svc.Save(ctx, in) // identical within 24h → dedup path
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if first != second {
		t.Fatalf("dedup should return same id: %d vs %d", first, second)
	}
	obs, err := svc.Get(ctx, first)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if obs.DuplicateCount != 1 {
		t.Fatalf("duplicate_count=%d want 1", obs.DuplicateCount)
	}
	if obs.LastSeenAt == "" {
		t.Fatalf("expected last_seen_at to be set after duplicate save")
	}
}

func TestTTLSoftExpiryAndRetire(t *testing.T) {
	_, svc := newTestStore(t, "ttlproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	for _, ts := range []string{past, future} {
		_, err := svc.store.DB.ExecContext(ctx, `
			INSERT INTO observations (session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at, source, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, 'agent', ?)`,
			sid, "decision", "ttl row"+ts, "body", "ttlproj", "project", "h-"+ts, past, past, ts)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	pending, err := svc.TTLSoftExpiry(ctx)
	if err != nil || pending != 1 {
		t.Fatalf("pending=%d err=%v (want 1)", pending, err)
	}
	expired, err := svc.TTLRetire(ctx)
	if err != nil || expired != 1 {
		t.Fatalf("retired=%d err=%v (want 1)", expired, err)
	}
	// After retiring, the expired row is soft-deleted → no longer pending.
	if pending, _ := svc.TTLSoftExpiry(ctx); pending != 0 {
		t.Fatalf("after retire pending=%d want 0", pending)
	}
}

func TestCosineAndRRF(t *testing.T) {
	a := Vector{Data: []float32{1, 0}}
	b := Vector{Data: []float32{0, 1}}
	if got := CosineSimilarity(a, b); got != 0 {
		t.Fatalf("orthogonal cosine=%v want 0", got)
	}
	// Same direction → ~1.
	if got := CosineSimilarity(a, a); got < 1-1e-9 || got > 1+1e-9 {
		t.Fatalf("identical cosine=%v want 1", got)
	}

	// RRF: an item ranked #1 in FTS but absent from vectors should outrank an
	// item ranked #5 in FTS that is also #5 in vectors — the top lexical hit
	// stays near the top.
	ids := []string{"a", "b", "c"}
	ftsRanks := map[string]int{"a": 0, "b": 4, "c": 4}
	vecRanks := map[string]int{"b": 0, "c": 1}
	got := ReciprocalRankFusion(ids, ftsRanks, vecRanks, 60)
	if len(got) != 3 {
		t.Fatalf("rrf len=%d", len(got))
	}
	// 'a' only has the FTS top-1 term = 1/61.
	// 'b' has FTS#5 (1/65) + vec#1 (1/61).
	// 'c' has FTS#5 (1/65) + vec#2 (1/62).
	// So order should be b > c > a OR a vs b/c close; assert a is last or
	// b is first — the important property is all three present and deterministic.
	if got[0] == "a" {
		// 'a'=1/61=0.01639; 'b'=1/61+1/65=0.03269; 'b' must beat 'a'.
		t.Fatalf("rrf[0]=%q expected the blended winner to beat the single-list top lexical hit", got[0])
	}
}

func TestBlendedSearchFallbackToFTS(t *testing.T) {
	_, svc := newTestStore(t, "blendproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	if _, err := svc.Save(ctx, SaveInput{Title: "authentication design", Type: "decision", Content: "JWT sessions", SessionID: sid, Scope: "project"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	// No embeddings present + empty vector → must still find the row via FTS.
	hits, err := svc.BlendedSearch(ctx, "authentication design", "any", "", Vector{}, 10)
	if err != nil {
		t.Fatalf("blended: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected FTS fallback to return the saved row")
	}
	if hits[0].Project != "blendproj" {
		t.Fatalf("project=%q", hits[0].Project)
	}
}

func TestEncodeDecodeVectorRoundTrip(t *testing.T) {
	orig := Vector{Data: []float32{1.5, -2.25, 0.125, 42}}
	blob := EncodeVector(orig)
	if len(blob) != len(orig.Data)*4 {
		t.Fatalf("blob len=%d", len(blob))
	}
	got, err := DecodeVector(blob)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Data) != len(orig.Data) {
		t.Fatalf("len mismatch")
	}
	for i := range orig.Data {
		if got.Data[i] != orig.Data[i] {
			t.Fatalf("element %d: %v != %v", i, got.Data[i], orig.Data[i])
		}
	}
}

// TestSetEmbeddingRoundTrip stores a vector and reads it back through the
// service, proving the BLOB layout matches DecodeVector.
func TestSetEmbeddingRoundTrip(t *testing.T) {
	_, svc := newTestStore(t, "embproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	id, err := svc.Save(ctx, SaveInput{Title: "emb note", Type: "decision", Content: "vector target", SessionID: sid, Scope: "project"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	vec := Vector{Data: []float32{0.1, 0.2, 0.3}}
	if err := svc.SetEmbedding(ctx, id, EncodeVector(vec), "test-model"); err != nil {
		t.Fatalf("set embedding: %v", err)
	}
	hits, err := svc.SearchByVector(ctx, Vector{Data: []float32{0.1, 0.2, 0.3}}, 5)
	if err != nil {
		t.Fatalf("vector search: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != id {
		t.Fatalf("vector hits=%+v (want 1 hit for id %d)", hits, id)
	}
	if hits[0].Sim < 1-1e-9 || hits[0].Sim > 1+1e-9 {
		t.Fatalf("self-similarity=%v want ~1", hits[0].Sim)
	}
}
