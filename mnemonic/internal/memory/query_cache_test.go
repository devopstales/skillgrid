package memory

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestQueryCacheHit covers a repeated query+model inside the 7-day window:
// the second call returns the cached vector and does not call inner.
func TestQueryCacheHit(t *testing.T) {
	st, _ := newTestStore(t, "qcachehit")
	db := st.DB
	ctx := context.Background()

	const model = "embed-v1"
	const query = "super-secret-query-text"
	calls := 0
	inner := func(context.Context, string) (Vector, error) {
		calls++
		return Vector{Data: []float32{1, 2, 3}}, nil
	}

	first, err := CachedEmbedQuery(ctx, db, model, inner, query)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := CachedEmbedQuery(ctx, db, model, inner, query)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if calls != 1 {
		t.Fatalf("inner calls = %d, want 1", calls)
	}
	if len(second.Data) != len(first.Data) {
		t.Fatalf("cached len = %d, want %d", len(second.Data), len(first.Data))
	}
	for i := range first.Data {
		if second.Data[i] != first.Data[i] {
			t.Fatalf("cached[%d] = %v, want %v", i, second.Data[i], first.Data[i])
		}
	}
	assertQueryTextNotStored(t, db, query)
}

// TestQueryCacheMiss covers an 8-day-old row and a different model: both
// call inner again.
func TestQueryCacheMiss(t *testing.T) {
	st, _ := newTestStore(t, "qcachemiss")
	db := st.DB
	ctx := context.Background()

	const query = "same-query"
	calls := 0
	inner := func(context.Context, string) (Vector, error) {
		calls++
		return Vector{Data: []float32{float32(calls)}}, nil
	}

	if _, err := CachedEmbedQuery(ctx, db, "model-a", inner, query); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if calls != 1 {
		t.Fatalf("seed calls = %d, want 1", calls)
	}

	stale := time.Now().UTC().Add(-8 * 24 * time.Hour).Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE query_cache SET created_at = ?`, stale); err != nil {
		t.Fatalf("age row: %v", err)
	}
	aged, err := CachedEmbedQuery(ctx, db, "model-a", inner, query)
	if err != nil {
		t.Fatalf("aged: %v", err)
	}
	if calls != 2 {
		t.Fatalf("aged calls = %d, want 2", calls)
	}
	if len(aged.Data) != 1 || aged.Data[0] != 2 {
		t.Fatalf("aged vector = %+v, want [2]", aged.Data)
	}

	other, err := CachedEmbedQuery(ctx, db, "model-b", inner, query)
	if err != nil {
		t.Fatalf("other model: %v", err)
	}
	if calls != 3 {
		t.Fatalf("other-model calls = %d, want 3", calls)
	}
	if len(other.Data) != 1 || other.Data[0] != 3 {
		t.Fatalf("other vector = %+v, want [3]", other.Data)
	}
	assertQueryTextNotStored(t, db, query)
}

func assertQueryTextNotStored(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(query_cache)`)
	if err != nil {
		t.Fatalf("table_info: %v", err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		cols = append(cols, name)
		if strings.Contains(strings.ToLower(name), "query") && name != "query_hash" {
			t.Fatalf("query text column %q", name)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("columns: %v", err)
	}
	want := []string{"query_hash", "embedding", "model", "created_at"}
	if strings.Join(cols, ",") != strings.Join(want, ",") {
		t.Fatalf("columns = %v, want %v", cols, want)
	}

	stored, err := db.Query(`SELECT query_hash, embedding, model, created_at FROM query_cache`)
	if err != nil {
		t.Fatalf("select cache: %v", err)
	}
	defer stored.Close()
	n := 0
	for stored.Next() {
		n++
		var hash, model, created string
		var blob []byte
		if err := stored.Scan(&hash, &blob, &model, &created); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		if strings.Contains(hash, query) || strings.Contains(model, query) || strings.Contains(created, query) || bytes.Contains(blob, []byte(query)) {
			t.Fatal("query text stored in query_cache")
		}
	}
	if err := stored.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if n == 0 {
		t.Fatal("expected a query_cache row")
	}
}

// TestQueryCacheEmbedderError covers an inner embedder failure: the error
// is returned and no cache row is written, so a later call can try again.
func TestQueryCacheEmbedderError(t *testing.T) {
	st, _ := newTestStore(t, "qcacheerr")
	db := st.DB
	ctx := context.Background()
	innerErr := errors.New("model unavailable")
	inner := func(context.Context, string) (Vector, error) {
		return Vector{}, innerErr
	}
	_, err := CachedEmbedQuery(ctx, db, "embed-v1", inner, "banana")
	if !errors.Is(err, innerErr) {
		t.Fatalf("err = %v, want inner error", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM query_cache`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("query_cache rows = %d, want 0", n)
	}
}
