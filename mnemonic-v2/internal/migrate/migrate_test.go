// Package migrate_test is the v1→v2 seed migration's test (TICKET-19,
// brief Task 19). It is an external test package (package migrate_test)
// so it can import both memory and migrate without an import cycle (the
// migrator imports memory.Service; the test imports memory for its test
// fixtures + the migrate package for the API under test). The brief
// names the file migrate_test.go; the package is migrate_test (the
// external test package convention).
//
// The tests build a real v1-shaped .sqlite (v1 schema, v1 type
// vocabulary) in a temp dir and assert the seed landed in v2, that the v1
// files came back byte-identical, that a re-run is a no-op, and that the
// doctor sees the migration state.

package migrate_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/migrate"
)

// ctx is the test's background context (the memory package's tests use a
// package var; the external test package defines its own).
var ctx = context.Background()

// v1FirstObservationTerm is a search term guaranteed to be in the first
// seeded observation's title (the seed-import test's retrieval assertion).
const v1FirstObservationTerm = "v1seedfirst"

// ── v1 fixture ──────────────────────────────────────────────────────────────

// makeV1Store builds a real v1-shaped .sqlite at dir/<project>.sqlite: the
// v1 sessions table (v1 001_schema, with the pre-054 observations columns
// merged in — the layout the brief names: observations(id, session_id,
// type, title, content, project, scope, topic_key, normalized_hash,
// importance_score, …) + sessions) plus two live observations (one in a
// v2-vocabulary type, one in a v1-only type — the mapping's default arm)
// and one already-deleted observation (the deleted arm). It returns the
// row count of live observations.
func makeV1Store(t *testing.T, dir, project string) int {
	t.Helper()
	path := filepath.Join(dir, project+".sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("v1 open: %v", err)
	}
	defer db.Close()
	// v1 sessions (mnemonic_back/internal/store/migrations/001_schema.sql).
	if _, err := db.Exec(`CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		project TEXT NOT NULL,
		directory TEXT NOT NULL,
		started_at TEXT NOT NULL,
		ended_at TEXT,
		summary TEXT,
		status TEXT NOT NULL DEFAULT 'active')`); err != nil {
		t.Fatalf("v1 sessions: %v", err)
	}
	// v1 observations (001_schema core + the ALTER-added columns the brief
	// names, merged into one table — a mid-v1 store shape).
	if _, err := db.Exec(`CREATE TABLE observations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL REFERENCES sessions(id),
		type TEXT NOT NULL,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		project TEXT,
		scope TEXT,
		topic_key TEXT,
		normalized_hash TEXT,
		revision_count INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		deleted_at TEXT,
		importance_score REAL,
		tool_name TEXT,
		pinned INTEGER NOT NULL DEFAULT 0)`); err != nil {
		t.Fatalf("v1 observations: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('v1s1', ?, '', datetime('now'), 'active')`, project); err != nil {
		t.Fatalf("v1 session: %v", err)
	}
	live := 0
	ins := func(id int, typ, title, content, topicKey string, deleted bool) {
		t.Helper()
		del := any(nil)
		if deleted {
			del = "2026-01-02 03:04:05"
		}
		// 13 columns, 8 ? placeholders (the rest are literals). Args:
		// id, typ, title, content, project, topicKey, id (for 'hash' ||
		// id), del.
		if _, err := db.Exec(`INSERT INTO observations
			(id, session_id, type, title, content, project, scope, topic_key,
			 normalized_hash, importance_score, created_at, updated_at, deleted_at)
			VALUES (?, 'v1s1', ?, ?, ?, ?, 'project', ?, 'hash' || ?,
				1.0, datetime('now'), datetime('now'), ?)`,
			id, typ, title, content, "p", nullableStr(topicKey), id, del); err != nil {
			t.Fatalf("v1 obs %d: %v", id, err)
		}
		if !deleted {
			live++
		}
	}
	// Live #1: a v2-vocabulary type (decision), the search-term carrier.
	ins(1, "decision", "v1seedfirst architecture choice",
		"the v1 store recorded a first-class decision about caching", "cache", false)
	// Live #2: a v1-only type (learning) — the mapping's default arm.
	ins(2, "learning", "v1seedsecond lesson learned",
		"the v1 store recorded a learning that v2 maps to discovery", "", false)
	// Deleted: must not be seeded (v1's deleted_at arm).
	ins(3, "decision", "v1seedthird already deleted",
		"this v1 row was deleted before the cutover", "", true)
	return live
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// v1DirHash returns the fingerprint the brief prescribes: the sorted
// name:size:mtime list of every file in the v1 dir (the v1 dir must be
// byte-identical after a migrate — a read-only open may not rewrite the
// page or append to a WAL file).
func v1DirHash(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("v1 dir read: %v", err)
	}
	var parts []string
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatalf("v1 dir entry: %v", err)
		}
		parts = append(parts, fmt.Sprintf("%s:%d:%d", e.Name(), info.Size(),
			info.ModTime().UnixNano()))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

// ── store test helpers (the memory package's testutil, re-declared here) ───
// The external test package cannot see memory's _test helpers (they are
// unexported in the memory package's test files). These are the minimal
// re-declarations the migrate tests need.

// openTestStoreIn opens a real SQLite store for project in dir.
func openTestStoreIn(t *testing.T, dir, project string) store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), dir, project, store.SQLite, "")
	if err != nil {
		t.Fatalf("open store (%s): %v", project, err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// probeStore narrows a store to the SQLite Probe test seam.
func probeStore(t *testing.T, st store.Store) interface {
	Probe(query string, args ...any) (bool, error)
	ProbeInt(query string, args ...any) (int, error)
} {
	t.Helper()
	p, ok := st.(interface {
		Probe(query string, args ...any) (bool, error)
		ProbeInt(query string, args ...any) (int, error)
	})
	if !ok {
		t.Fatal("store does not expose the Probe test seam")
	}
	return p
}

// liveCountIn counts live observations of a named project.
func liveCountIn(t *testing.T, st store.Store, project string) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`SELECT COUNT(*) FROM observations WHERE project = ? AND deleted_at IS NULL`, project)
	if err != nil {
		t.Fatalf("live count: %v", err)
	}
	return n
}

// probeInt is the t-aware ProbeInt (the test's assertion surface for the
// store's ledger / observations counts).
func probeInt(t *testing.T, st store.Store, query string, args ...any) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(query, args...)
	if err != nil {
		t.Fatalf("probe int: %v", err)
	}
	return n
}

// ── [v1-seed-import] ────────────────────────────────────────────────────────

// [v1-seed-import] the migration seeds the v2 store from the v1 file: the
// v1 live observations land in v2 (retrievable by their terms, mapped from
// the v1-only type into the v2 vocabulary) and the report shows zero
// errors with the live rows counted as exported.
func TestV1SeedImport(t *testing.T) {
	v1 := t.TempDir()
	live := makeV1Store(t, v1, "p")
	v2dir := t.TempDir()
	rep, err := migrate.Migrate(ctx, v1, v2dir, "p")
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if rep.Exported != live {
		t.Fatalf("report must count the live rows exported: got %+v, want %d", rep, live)
	}
	if rep.Skipped != 0 {
		t.Fatalf("first run must skip nothing: %+v", rep)
	}
	if len(rep.Errors) != 0 {
		t.Fatalf("report must have zero errors: %+v", rep)
	}
	// the v2 store has the observations, retrievable by search (the
	// acceptance scenario's "searchable by their terms").
	st := openTestStoreIn(t, v2dir, "p")
	r, err := st.Search(ctx, store.SearchQuery{Query: v1FirstObservationTerm, Project: "p", Limit: 5})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Observations) == 0 {
		t.Fatalf("seeded observations not searchable in v2")
	}
	// the door-check assertion: the v1-only type (learning) maps into the
	// v2 vocabulary (discovery) — the field mapping the brief pins. The
	// search is FTS (matches the query term), so the door-check reads the
	// row by its own search term.
	r2, err := st.Search(ctx, store.SearchQuery{Query: "v1seedsecond", Project: "p", Limit: 5})
	if err != nil {
		t.Fatalf("search (door check): %v", err)
	}
	found := false
	for _, o := range r2.Observations {
		if o.Title == "v1seedsecond lesson learned" {
			found = true
			if o.Type != "discovery" {
				t.Fatalf("v1 type 'learning' must map to 'discovery' (the brief's mapping); got %q", o.Type)
			}
		}
	}
	if !found {
		t.Fatalf("the v1-only-type row must be seeded (door check): %+v", r2.Observations)
	}
	// the mapping: v1's importance_score is NOT copied — v2's importance
	// is on-the-fly (the fresh-insert floor, spec 002-schema §2.2.3). The
	// floor is not 0.5 (the store's INSERT default) — the save pipeline
	// recomputes it (Importance(0, 0, type) — a type-based floor above
	// 0.5 for some types). The assertion: the v1 importance_score (1.0)
	// is NOT copied — the v2 importance is the pipeline's computed value,
	// not the v1's 1.0.
	if n := probeInt(t, st,
		`SELECT COUNT(*) FROM observations WHERE importance = 1.0`); n != 0 {
		t.Fatalf("v1 importance_score (1.0) must not be copied; %d rows at 1.0", n)
	}
	// the deleted v1 row must not be seeded. The live count includes the
	// session_summary seed rows (one per v1 session) — the assertion is
	// over the observation-only count.
	if n := probeInt(t, st,
		`SELECT COUNT(*) FROM observations WHERE project = 'p' AND deleted_at IS NULL AND type != 'session_summary'`); n != live {
		t.Fatalf("v2 live observation count must equal the v1 live count (%d); got %d (the deleted row leaked)", live, n)
	}
	// the replay ledger carries the v1migrate: observation operation ids
	// (a re-run is idempotent per source row — Task 6's ledger).
	if n := probeInt(t, st,
		`SELECT COUNT(*) FROM observation_save_operations WHERE operation_id LIKE 'v1migrate:%' AND operation_id NOT LIKE 'v1migrate:session:%'`); n != live {
		t.Fatalf("ledger must hold one v1migrate: row per seeded observation; %d of %d", n, live)
	}
	// provenance.writer (the review's fix): every seeded row carries
	// provenance.writer = "v1-migrate" (the v1 store has no provenance
	// column; the seed is the rows' writer) — read from the INV-1
	// provenance column directly, not the search result (the search
	// result does not project provenance).
	if n := probeInt(t, st,
		`SELECT COUNT(*) FROM observations WHERE project = 'p' AND deleted_at IS NULL AND json_extract(provenance, '$.writer') != 'v1-migrate'`); n != 0 {
		t.Fatalf("every seeded row must carry provenance.writer = 'v1-migrate'; %d rows differ", n)
	}
	// the normal save surfaces leave provenance.writer empty (the
	// existing behavior — the seed's rows are the only ones with a writer).
	svc := memory.NewService(openTestStoreIn(t, v2dir, "p"), v2TestConfig(v2dir), memory.NewEventBus(), nil, nil)
	if _, err := svc.Save(ctx, memory.SaveRequest{
		SessionID: "s-normal", Type: "discovery", Title: "writer-empty normal save",
		Content: "a normal (non-seed) save", Project: "p", Scope: "project",
	}); err != nil {
		t.Fatalf("normal save: %v", err)
	}
	if n := probeInt(t, st,
		`SELECT COUNT(*) FROM observations WHERE title = 'writer-empty normal save' AND json_extract(provenance, '$.writer') != ''`); n != 0 {
		t.Fatalf("a normal save must leave provenance.writer empty; %d rows differ", n)
	}
}

// v2TestConfig is the v2 store config the test's non-seed save runs under
// (the migrate package's v2Config, mirrored here — default config, the
// test's data dir, project detection off).
func v2TestConfig(dataDir string) config.Config {
	c := config.Default()
	c.DataDir = dataDir
	c.Project.Detect = false
	return c
}

// ── [v1-files-untouched] ────────────────────────────────────────────────────

// [v1-files-untouched] the migration is read-only over the v1 files: every
// file in the v1 dir is byte-identical (name, size, mtime) before and after
// (different schema philosophy — export, don't migrate in place).
func TestV1FilesUntouched(t *testing.T) {
	v1 := t.TempDir()
	makeV1Store(t, v1, "p")
	before := v1DirHash(t, v1)
	if _, err := migrate.Migrate(ctx, v1, t.TempDir(), "p"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	after := v1DirHash(t, v1)
	if before != after {
		t.Fatalf("migrate must be read-only over the v1 files (different schema philosophy — export, don't migrate in place): %q -> %q", before, after)
	}
}

// ── idempotency (re-run is a no-op) ────────────────────────────────────────

// [migrate-idempotent] a second run of the migration over the same v2 dir
// is a no-op: every row replays through the ledger (skipped), no
// duplicates, the v2 store is unchanged.
func TestV1MigrateIdempotent(t *testing.T) {
	v1 := t.TempDir()
	live := makeV1Store(t, v1, "p")
	v2dir := t.TempDir()
	if _, err := migrate.Migrate(ctx, v1, v2dir, "p"); err != nil {
		t.Fatalf("migrate 1: %v", err)
	}
	st := openTestStoreIn(t, v2dir, "p")
	// The live count includes the session_summary seed rows (one per v1
	// session) — the brief's seed unit is the observations, so the
	// idempotency assertion is over the observation-only count.
	before := probeInt(t, st,
		`SELECT COUNT(*) FROM observations WHERE project = 'p' AND deleted_at IS NULL AND type != 'session_summary'`)
	ledBefore := probeInt(t, st,
		`SELECT COUNT(*) FROM observation_save_operations WHERE operation_id LIKE 'v1migrate:%'`)
	if before != live {
		t.Fatalf("after run 1: %d live observations, want %d", before, live)
	}
	if _, err := migrate.Migrate(ctx, v1, v2dir, "p"); err != nil {
		t.Fatalf("migrate 2: %v", err)
	}
	after := probeInt(t, st,
		`SELECT COUNT(*) FROM observations WHERE project = 'p' AND deleted_at IS NULL AND type != 'session_summary'`)
	ledAfter := probeInt(t, st,
		`SELECT COUNT(*) FROM observation_save_operations WHERE operation_id LIKE 'v1migrate:%'`)
	if after != before {
		t.Fatalf("re-run must not write new observations: %d -> %d", before, after)
	}
	if ledAfter != ledBefore {
		t.Fatalf("re-run must not write new ledger rows: %d -> %d", ledBefore, ledAfter)
	}
}

// ── triangulation ───────────────────────────────────────────────────────────

// [v1-seed-empty-store] a v1 store with zero live observations seeds an
// empty v2 store: the report is all zeros and no rows land (different
// inputs than the two-row seed — the trivial implementation check).
func TestV1MigrateEmptyStore(t *testing.T) {
	v1 := t.TempDir()
	path := filepath.Join(v1, "p.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("v1 open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE sessions (
		id TEXT PRIMARY KEY, project TEXT NOT NULL, directory TEXT NOT NULL,
		started_at TEXT NOT NULL, ended_at TEXT, summary TEXT,
		status TEXT NOT NULL DEFAULT 'active')`); err != nil {
		t.Fatalf("v1 sessions: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE observations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL, type TEXT NOT NULL, title TEXT NOT NULL,
		content TEXT NOT NULL, project TEXT, scope TEXT, topic_key TEXT,
		normalized_hash TEXT, revision_count INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT,
		importance_score REAL, tool_name TEXT,
		pinned INTEGER NOT NULL DEFAULT 0)`); err != nil {
		t.Fatalf("v1 observations: %v", err)
	}
	v2dir := t.TempDir()
	rep, err := migrate.Migrate(ctx, v1, v2dir, "p")
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if rep.Exported != 0 || rep.Skipped != 0 || len(rep.Errors) != 0 {
		t.Fatalf("empty v1 must report all zeros: %+v", rep)
	}
	st := openTestStoreIn(t, v2dir, "p")
	if n := liveCountIn(t, st, "p"); n != 0 {
		t.Fatalf("empty v1 must seed an empty v2 store; %d live rows", n)
	}
}

// [v1-seed-type-mapping] the v1 type vocabulary maps into the v2 vocabulary
// (the brief's mapping, each arm): decision/architecture/bugfix stay
// themselves, learning (a v1-only type) falls to the default arm
// (discovery), and an unmapped v1 type falls to discovery. Different
// inputs: five distinct type arms.
func TestV1MigrateTypeMapping(t *testing.T) {
	v1 := t.TempDir()
	path := filepath.Join(v1, "p.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("v1 open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE sessions (
		id TEXT PRIMARY KEY, project TEXT NOT NULL, directory TEXT NOT NULL,
		started_at TEXT NOT NULL, ended_at TEXT, summary TEXT,
		status TEXT NOT NULL DEFAULT 'active')`); err != nil {
		t.Fatalf("v1 sessions: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE observations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL, type TEXT NOT NULL, title TEXT NOT NULL,
		content TEXT NOT NULL, project TEXT, scope TEXT, topic_key TEXT,
		normalized_hash TEXT, revision_count INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT,
		importance_score REAL, tool_name TEXT,
		pinned INTEGER NOT NULL DEFAULT 0)`); err != nil {
		t.Fatalf("v1 observations: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s', 'p', '', datetime('now'), 'active')`); err != nil {
		t.Fatalf("v1 session: %v", err)
	}
	// One row per v1 type arm, in separate subtests (each subtest runs the
	// full migrate + probe cycle over its own v1 store — the type mapping
	// is per-row, so the test is per-row too).
	rows := []struct{ v1, want string }{
		{"decision", "decision"},
		{"architecture", "architecture"},
		{"bugfix", "bugfix"},
		{"learning", "discovery"}, // v1-only type → default arm
		{"epiphany", "discovery"}, // unmapped v1 type → default arm
	}
	for _, r := range rows {
		t.Run(r.v1, func(t *testing.T) {
			// shared v1 store, fresh v2 store per subtest (the probe
			// filters by title, so the shared v1 rows don't interfere).
			if _, err := db.Exec(`INSERT INTO observations
				(session_id, type, title, content, project, scope,
				 normalized_hash, created_at, updated_at)
				VALUES ('s', ?, ?, ?, 'p', 'project', 'h' || ?, datetime('now'), datetime('now'))`,
				r.v1, r.v1+" title", r.v1+" content", r.v1); err != nil {
				t.Fatalf("v1 insert: %v", err)
			}
			v2dir := t.TempDir()
			if _, err := migrate.Migrate(ctx, v1, v2dir, "p"); err != nil {
				t.Fatalf("migrate: %v", err)
			}
			st := openTestStoreIn(t, v2dir, "p")
			found, err := probeStore(t, st).Probe(
				`SELECT COUNT(*) > 0 FROM observations WHERE title = ? AND type = ?`,
				r.v1+" title", r.want)
			if err != nil {
				t.Fatalf("probe %q: %v", r.v1, err)
			}
			if !found {
				// The probe is over the v2 store — the seeded row's
				// title is the v1 title (the seed copies it verbatim),
				// and its type is the mapped v2 type. A miss is a real
				// mapping failure.
				t.Fatalf("v1 type %q must map to %q (the seeded row's v2 type)", r.v1, r.want)
			}
		})
	}
}

// ── doctor migration state ──────────────────────────────────────────────────

// [doctor-migration-state] the doctor's migration state check reports the
// v1 dir + the migration state: a v1 store with zero v1migrate: ledger rows
// is migrate-pending; a migrated store (ledger rows present) is
// migrate-done; an absent v1 store is advisory-ok (the check never fails
// the doctor — a pending migration is a report line, not a capability gap).
func TestV1DoctorMigrationState(t *testing.T) {
	// pending: a v1 dir exists but no rows are seeded yet.
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	st := openTestStoreIn(t, c.DataDir, "p")
	svc := memory.NewService(st, c, memory.NewEventBus(), nil, nil)
	// The doctor's wiring: bind the one-way hook to the service's store
	// (the migrate package's SetMigrationStateHook is the one-way wire;
	// the doctor's runtime overrides it with its own store binding — the
	// store's SetApplyPulledMutationHook pattern).
	prev := memory.MigrationStateHook
	migrate.SetMigrationStateHook(func(ctx context.Context, v1Dir, project string) (any, error) {
		return migrate.Report(ctx, v1Dir, c.DataDir, project, st)
	})
	t.Cleanup(func() { memory.MigrationStateHook = prev })
	v1dir := t.TempDir()
	if n := makeV1Store(t, v1dir, "p"); n == 0 {
		t.Fatal("fixture must have live rows")
	}
	stsAny, err := svc.MigrationState(ctx, v1dir, "p")
	if err != nil {
		t.Fatalf("migration state: %v", err)
	}
	sts, ok := stsAny.(migrate.State)
	if !ok {
		t.Fatalf("hook must return a migrate.State; got %T", stsAny)
	}
	if !sts.V1Present {
		t.Fatalf("a v1 dir with a project store must be detected: %+v", sts)
	}
	if sts.MigrationState != migrate.StatePending {
		t.Fatalf("zero v1migrate: ledger rows must be migrate-pending; got %q", sts.MigrationState)
	}

	// done: run the migration, re-check.
	if _, err := migrate.Migrate(ctx, v1dir, c.DataDir, "p"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	stsAny, err = svc.MigrationState(ctx, v1dir, "p")
	if err != nil {
		t.Fatalf("migration state (after): %v", err)
	}
	sts = stsAny.(migrate.State)
	if sts.MigrationState != migrate.StateDone {
		t.Fatalf("v1migrate: ledger rows present must be migrate-done; got %q", sts.MigrationState)
	}
	if sts.SeededRows == 0 || sts.V1Rows == 0 {
		t.Fatalf("state must count the v1 rows and the seeded ledger rows: %+v", sts)
	}

	// absent: no v1 dir → V1Present false, the check stays advisory-ok.
	emptyV1 := t.TempDir()
	stsAny, err = svc.MigrationState(ctx, emptyV1, "p")
	if err != nil {
		t.Fatalf("migration state (absent): %v", err)
	}
	sts = stsAny.(migrate.State)
	if sts.V1Present {
		t.Fatalf("an empty v1 dir must not be detected as a v1 store: %+v", sts)
	}
	if sts.MigrationState != "" {
		t.Fatalf("an absent v1 store must have an empty state (advisory-ok): %+v", sts)
	}
}
