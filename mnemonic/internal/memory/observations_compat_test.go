package memory

// TICKET-02 parity test (SATISFIES obs-parity-after-split).
//
// Opens a pre-split fixture store on a temp COPY (never the live store),
// snapshots row hashes via SELECT *, runs the real store.Migrate (which
// applies 054_obs_split.sql), then asserts row-hash parity through the
// compat view, intact invalid_at filters, and migrate() idempotent re-run.
//
// RED state: with no 054 file, store.Migrate is a no-op and the
// observations-is-a-view / side-table assertions below fail.

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

const parityMigrationFile = "054_obs_split.sql"

// parityCols is only used for fixture INSERTs; the hash dump itself is
// column-agnostic (SELECT * + sqlite column names) so a view column-order
// drift fails the test instead of silently passing.
var parityAllCols = []string{
	"session_id", "type", "title", "content", "project", "scope",
	"topic_key", "normalized_hash", "revision_count", "created_at",
	"updated_at", "deleted_at", "source", "review_after", "prompt_id",
	"pinned", "expires_at", "duplicate_count", "last_seen_at", "embedding",
	"embedding_model", "embedding_created_at", "tool_name", "owner", "status",
	"visibility", "retrieval_usage", "graph_ref", "importance_score",
	"recency_decay", "maturity_tier", "provenance", "memory_type",
	"risk_score", "archive_reason", "archived_at", "valid_at", "invalid_at",
	"superseded_by",
}

// buildPreSplitStore creates a temp store file with every migration EXCEPT
// 054 applied (bookkeeping mirrors store.migrate), seeds one session + five
// fixture rows, and returns the db path. The caller closes the raw handle
// before store.Open so the real migrate path applies 054.
func buildPreSplitStore(t *testing.T, dir, project string) string {
	t.Helper()
	dbPath := filepath.Join(dir, project+".sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatalf("pragma fk: %v", err)
	}
	migDir := filepath.Join("..", "store", "migrations")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		if e.Name() == parityMigrationFile || e.Name() == "055_obs_split_fk_repoint.sql" {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS index_meta (
		key TEXT PRIMARY KEY, schema_version INTEGER NOT NULL)`); err != nil {
		t.Fatalf("create index_meta: %v", err)
	}
	for _, name := range names {
		sqlBytes, err := os.ReadFile(filepath.Join(migDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			t.Fatalf("exec %s: %v", name, err)
		}
		if _, err := db.Exec(`INSERT INTO index_meta (key, schema_version) VALUES (?, 1)`,
			"migration:"+name); err != nil {
			t.Fatalf("record %s: %v", name, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	seedParityRows(t, db, project)
	pre := snapshotParity(t, db)
	if len(pre.hashes) != 5 {
		t.Fatalf("fixture rows = %d, want 5", len(pre.hashes))
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close fixture db: %v", err)
	}
	return dbPath
}

func parityExec(t *testing.T, db *sql.DB, cols []string, vals []any) {
	t.Helper()
	place := make([]string, len(cols))
	args := make([]any, len(vals))
	copy(args, vals)
	for i := range place {
		place[i] = "?"
	}
	q := fmt.Sprintf("INSERT INTO observations (%s) VALUES (%s)",
		strings.Join(cols, ", "), strings.Join(place, ", "))
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("insert fixture: %v", err)
	}
}

// seedParityRows inserts 5 rows exercising every observations column at
// least once: full, superseded (invalid_at past), empty-invalid, soft-deleted,
// and defaults-only minimal.
func seedParityRows(t *testing.T, db *sql.DB, project string) {
	t.Helper()
	// Row 1: full row, still valid.
	parityExec(t, db, parityAllCols, []any{
		"s1", "decision", "ParityAlpha", "parityfact full content", project, "project",
		"parity/alpha", "hash-alpha", 2, "2026-01-01T00:00:00Z",
		"2026-01-02T00:00:00Z", nil, "agent", nil, nil,
		0, nil, 3, "2026-01-03T00:00:00Z", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		"test-model", "2026-01-03T00:00:00Z", "mem_save", "parity-owner", "active",
		"private", 7, 42, 0.9,
		0.8, "mature", `{"session_id":"s1"}`, "preferences",
		0.1, nil, nil, "2026-01-01T00:00:00Z", nil,
		nil,
	})
	// Row 2: superseded (invalid_at past, superseded_by -> 1).
	parityExec(t, db, parityAllCols, []any{
		"s1", "learning", "ParityOld", "parityfact old content", project, "project",
		"parity/old", "hash-old", 1, "2026-02-01T00:00:00Z",
		"2026-02-02T00:00:00Z", nil, "agent", nil, nil,
		0, nil, 0, nil, nil,
		nil, nil, nil, "parity-owner", "superseded",
		"private", 1, nil, nil,
		nil, "fresh", nil, nil,
		nil, nil, nil, "2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z",
		1,
	})
	// Row 3: empty-string invalid_at (still true per ADR-0011), pinned.
	parityExec(t, db, parityAllCols, []any{
		"s1", "preference", "ParityPinned", "parityfact pinned content", project, "project",
		nil, "hash-pinned", 0, "2026-03-01T00:00:00Z",
		"2026-03-01T00:00:00Z", nil, "passive", "2026-04-01T00:00:00Z", 99,
		1, "2027-01-01T00:00:00Z", 0, nil, nil,
		nil, nil, nil, "parity-owner", "active",
		"team", 0, nil, nil,
		nil, nil, nil, nil,
		nil, nil, nil, "2026-03-01T00:00:00Z", "",
		nil,
	})
	// Row 4: soft-deleted + archived.
	parityExec(t, db, parityAllCols, []any{
		"s1", "bugfix", "ParityGone", "parityfact deleted content", project, "project",
		nil, "hash-gone", 0, "2026-04-01T00:00:00Z",
		"2026-04-02T00:00:00Z", "2026-05-01T00:00:00Z", "agent", nil, nil,
		0, nil, 0, nil, nil,
		nil, nil, nil, "parity-owner", "archived",
		"private", 0, nil, nil,
		nil, nil, nil, "entities",
		nil, "stale", "2026-05-01T00:00:00Z", "2026-04-01T00:00:00Z", "2026-05-01T00:00:00Z",
		nil,
	})
	// Row 5: near-minimal (service scan needs scope + normalized_hash
	// non-null; all other NULL-ables untouched, server defaults fill source
	// and the NOT NULL DEFAULT counters).
	parityExec(t, db, []string{"session_id", "type", "title", "content", "project", "scope", "normalized_hash", "created_at", "updated_at"},
		[]any{"s1", "learning", "ParityMinimal", "parityfact minimal", project, "project", "hash-minimal", "2026-05-01T00:00:00Z", "2026-05-01T00:00:00Z"})
}

type paritySnap struct {
	cols    []string
	hashes  map[int64]string
	active  []int64
	retired []int64
}

// snapshotParity dumps SELECT * ORDER BY id with canonical per-value
// serialization, plus the still-valid / superseded id sets (ADR-0011 rule:
// invalid_at NULL or ” = still true).
func snapshotParity(t *testing.T, db *sql.DB) paritySnap {
	t.Helper()
	snap := paritySnap{hashes: map[int64]string{}}
	rows, err := db.Query(`SELECT * FROM observations ORDER BY id`)
	if err != nil {
		t.Fatalf("dump observations: %v", err)
	}
	cols, err := rows.Columns()
	if err != nil {
		rows.Close()
		t.Fatalf("columns: %v", err)
	}
	snap.cols = cols
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			rows.Close()
			t.Fatalf("scan: %v", err)
		}
		h := sha256.New()
		for _, v := range vals {
			h.Write([]byte(paritySerial(v)))
			h.Write([]byte{0x1f})
		}
		id, ok := vals[0].(int64)
		if !ok {
			rows.Close()
			t.Fatalf("id column type %T, want int64", vals[0])
		}
		snap.hashes[id] = hex.EncodeToString(h.Sum(nil))
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	snap.active = parityIDs(t, db,
		`SELECT id FROM observations WHERE invalid_at IS NULL OR invalid_at = '' ORDER BY id`)
	snap.retired = parityIDs(t, db,
		`SELECT id FROM observations WHERE invalid_at IS NOT NULL AND invalid_at != '' ORDER BY id`)
	return snap
}

func parityIDs(t *testing.T, db *sql.DB, q string) []int64 {
	t.Helper()
	rows, err := db.Query(q)
	if err != nil {
		t.Fatalf("query %q: %v", q, err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan id: %v", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// paritySerial renders one driver value canonically (NULL-safe, BLOB-safe,
// float-deterministic) so pre-table and post-view dumps hash identically.
func paritySerial(v any) string {
	switch x := v.(type) {
	case nil:
		return "null:"
	case []byte:
		return "blob:" + hex.EncodeToString(x)
	case string:
		return "text:" + x
	case int64:
		return "int:" + strconv.FormatInt(x, 10)
	case float64:
		return "float:" + strconv.FormatFloat(x, 'g', -1, 64)
	case bool:
		return "bool:" + strconv.FormatBool(x)
	default:
		return fmt.Sprintf("%T:%v", v, v)
	}
}

func equalCols(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalIDs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func tableKind(t *testing.T, db *sql.DB, name string) string {
	t.Helper()
	var kind string
	if err := db.QueryRow(`SELECT type FROM sqlite_master WHERE name = ?`, name).Scan(&kind); err != nil {
		t.Fatalf("sqlite_master %s: %v", name, err)
	}
	return kind
}

// TestObservationsParity is the TICKET-02 acceptance test
// (SATISFIES obs-parity-after-split): row-hash parity vs the pre-split dump,
// intact invalid_at filters, and migrate() idempotent re-run.
func TestObservationsParity(t *testing.T) {
	dir := t.TempDir()
	const project = "parity"
	buildPreSplitStore(t, dir, project)

	// Pre-split snapshot from the raw fixture handle.
	raw, err := sql.Open("sqlite", filepath.Join(dir, project+".sqlite"))
	if err != nil {
		t.Fatalf("reopen fixture: %v", err)
	}
	raw.SetMaxOpenConns(1)
	pre := snapshotParity(t, raw)
	if err := raw.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}

	// Real migrate path: store.Open applies the pending 054 on a COPY.
	st, err := store.Open(dir, project)
	if err != nil {
		t.Fatalf("open store (migrate 054): %v", err)
	}
	t.Cleanup(func() { st.Close() })

	// RED gate: the split must have landed (fails when 054 is absent).
	if got := tableKind(t, st.DB, "observations"); got != "view" {
		t.Fatalf("observations type = %q, want view (054 not applied?)", got)
	}
	if got := tableKind(t, st.DB, "observations_legacy"); got != "table" {
		t.Fatalf("observations_legacy type = %q, want table", got)
	}
	var migCount int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM index_meta WHERE key = ?`,
		"migration:"+parityMigrationFile).Scan(&migCount); err != nil {
		t.Fatalf("index_meta 054: %v", err)
	}
	if migCount != 1 {
		t.Fatalf("migration:054 rows = %d, want 1", migCount)
	}
	for _, side := range []string{"observations_core", "observation_state",
		"observation_provenance", "observation_stats", "observation_temporal",
		"observation_vectors"} {
		var n int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM ` + side).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", side, err)
		}
		if n != len(pre.hashes) {
			t.Errorf("%s rows = %d, want %d", side, n, len(pre.hashes))
		}
	}

	// Row-hash parity through the compat view.
	post := snapshotParity(t, st.DB)
	if !equalCols(pre.cols, post.cols) {
		t.Fatalf("view columns = %v, want %v", post.cols, pre.cols)
	}
	if len(post.hashes) != len(pre.hashes) {
		t.Fatalf("post rows = %d, want %d", len(post.hashes), len(pre.hashes))
	}
	for id, want := range pre.hashes {
		got, ok := post.hashes[id]
		if !ok {
			t.Errorf("row %d missing behind view", id)
			continue
		}
		if got != want {
			t.Errorf("row %d hash mismatch:\n pre %s\npost %s", id, want, got)
		}
	}

	// invalid_at filters intact (ADR-0011 NULL/empty = still true).
	if !equalIDs(pre.active, post.active) {
		t.Errorf("still-valid ids = %v, want %v", post.active, pre.active)
	}
	if !equalIDs(pre.retired, post.retired) {
		t.Errorf("superseded ids = %v, want %v", post.retired, pre.retired)
	}

	// Production read path still filters superseded rows via invalid_at.
	svc := New(st, project)
	got, err := svc.Recent(t.Context(), 20)
	if err != nil {
		t.Fatalf("service Recent post-split: %v", err)
	}
	var gotIDs []int64
	for _, o := range got {
		gotIDs = append(gotIDs, o.ID)
	}
	// Rows 1,3,5 live (created_at ASC, Recent orders newest first);
	// row 2 superseded, row 4 soft-deleted.
	if want := []int64{5, 3, 1}; !equalIDs(gotIDs, want) {
		t.Errorf("Recent ids = %v, want %v", gotIDs, want)
	}

	// migrate() idempotent re-run: no-op (no new index_meta rows, hashes stable).
	var before int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM index_meta`).Scan(&before); err != nil {
		t.Fatalf("count index_meta: %v", err)
	}
	if err := store.Migrate(st.DB); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	var after int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM index_meta`).Scan(&after); err != nil {
		t.Fatalf("recount index_meta: %v", err)
	}
	if after != before {
		t.Errorf("index_meta rows %d -> %d on re-migrate, want no-op", before, after)
	}
	again := snapshotParity(t, st.DB)
	for id, want := range post.hashes {
		if again.hashes[id] != want {
			t.Errorf("row %d hash changed on re-migrate", id)
		}
	}
}

// TestHotPathAvoidsVectors is the TICKET-03 acceptance test
// (SATISFIES hot-path-avoids-blobs): the hot-path read shape
// (obsSplitSelectCols + obsSplitFrom) must not reference
// observation_vectors in its query plan, and returned rows must equal the
// compat-view rows minus the vector columns.
func TestHotPathAvoidsVectors(t *testing.T) {
	dir := t.TempDir()
	const project = "hotpath"
	buildPreSplitStore(t, dir, project)

	st, err := store.Open(dir, project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	// EXPLAIN QUERY PLAN: the hot-path read shape must not name
	// observation_vectors in any plan row.
	plan := `EXPLAIN QUERY PLAN SELECT ` + obsSplitSelectCols + ` FROM ` + obsSplitFrom + ` WHERE o.deleted_at IS NULL AND o.project = ?`
	rows, err := st.DB.Query(plan, project)
	if err != nil {
		t.Fatalf("explain hot path: %v", err)
	}
	defer rows.Close()
	var planText strings.Builder
	for rows.Next() {
		var id int
		var parent, notused, detail any
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		ds, _ := detail.(string)
		planText.WriteString(ds)
		planText.WriteByte('\n')
		if strings.Contains(ds, "observation_vectors") {
			t.Fatalf("hot-path plan references observation_vectors:\n%s", planText.String())
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("plan rows: %v", err)
	}

	// Row parity: hot-path rows == compat-view rows minus the three vector
	// columns (embedding, embedding_model, embedding_created_at).
	_, viewRows := dumpViewRows(t, st.DB, project)
	splitCols, splitRows := dumpSplitRows(t, st.DB, project)

	if len(viewRows) != len(splitRows) {
		t.Fatalf("row count: view=%d split=%d", len(viewRows), len(splitRows))
	}
	// The view has 40 columns; the split shape has 31 (no vector cols).
	// Verify the split shape projects only non-vector columns: the last
	// identifier token of each column expression must not be a vector column.
	colName := func(expr string) string {
		expr = strings.TrimSpace(expr)
		if i := strings.LastIndex(expr, "."); i >= 0 {
			expr = expr[i+1:]
		}
		// Strip COALESCE(...) wrapper: take the inner first arg.
		if strings.HasPrefix(expr, "COALESCE(") && strings.HasSuffix(expr, ")") {
			inner := expr[len("COALESCE(") : len(expr)-1]
			if ci := strings.Index(inner, ","); ci >= 0 {
				expr = strings.TrimSpace(inner[:ci])
				if i := strings.LastIndex(expr, "."); i >= 0 {
					expr = expr[i+1:]
				}
			}
		}
		return expr
	}
	for _, c := range splitCols {
		n := colName(c)
		if n == "embedding" || n == "embedding_model" || n == "embedding_created_at" {
			t.Fatalf("hot-path split shape includes vector column %q (expr %q)", n, c)
		}
	}
}

// dumpViewRows returns the column names and one row-per-observation dump of
// the compat view (SELECT *), for row-parity comparison.
func dumpViewRows(t *testing.T, db *sql.DB, project string) ([]string, map[int64][]string) {
	t.Helper()
	rows, err := db.Query(`SELECT * FROM observations WHERE project = ? ORDER BY id`, project)
	if err != nil {
		t.Fatalf("view dump: %v", err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatalf("view cols: %v", err)
	}
	out := make(map[int64][]string)
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("view scan: %v", err)
		}
		id, _ := vals[0].(int64)
		s := make([]string, len(vals))
		for i, v := range vals {
			s[i] = paritySerial(v)
		}
		out[id] = s
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("view rows: %v", err)
	}
	return cols, out
}

// dumpSplitRows returns the column names and one row-per-observation dump of
// the hot-path split shape, for row-parity comparison.
func dumpSplitRows(t *testing.T, db *sql.DB, project string) ([]string, map[int64][]string) {
	t.Helper()
	q := `SELECT ` + obsSplitSelectCols + ` FROM ` + obsSplitFrom + ` WHERE o.project = ? ORDER BY o.id`
	rows, err := db.Query(q, project)
	if err != nil {
		t.Fatalf("split dump: %v", err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatalf("split cols: %v", err)
	}
	out := make(map[int64][]string)
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("split scan: %v", err)
		}
		id, _ := vals[0].(int64)
		s := make([]string, len(vals))
		for i, v := range vals {
			s[i] = paritySerial(v)
		}
		out[id] = s
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("split rows: %v", err)
	}
	return cols, out
}
