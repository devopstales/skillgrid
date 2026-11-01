package store

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// pgTestDSN is the reachable Postgres instance for the PG parity tests
// (gate G15 — CI provides it). Empty → the TestPG* suite skips cleanly.
// If the DSN uses a localhost host and MNEMONIC_PG_LOCALHOST_HOST is set
// (the docker port-mapping case), pgLocalhostDSN rewrites the host.
var pgTestDSN = os.Getenv("MNEMONIC_PG_TEST_DSN")

// pgTestLocalhostHost is the docker bridge IP for localhost DSNs (the test
// runs on the host, PG is in a container). Set in init so the DSN rewrite
// applies to every PG test.
func init() {
	// Only set when the DSN is a localhost host (the docker case).
	if pgTestDSN != "" && os.Getenv("MNEMONIC_PG_LOCALHOST_HOST") == "" {
		os.Setenv("MNEMONIC_PG_LOCALHOST_HOST", "198.18.0.2")
	}
}

// requirePG skips the test when no PG DSN is configured (the tests are
// integration tests — the SQLite suite stays green without a database).
func requirePG(t *testing.T) {
	t.Helper()
	if pgTestDSN == "" {
		t.Skip("MNEMONIC_PG_TEST_DSN not set — skipping the PG parity suite")
	}
}

// pgSchemaTag returns a unique schema tag per test run (the throwaway test
// DB is shared; a per-run project name isolates every test's schema).
var pgSchemaTag = fmt.Sprintf("pgtest-%d", time.Now().UnixNano())

// openPGTestStore opens a PG store on a per-run unique project so schemas
// never collide within or across runs.
func openPGTestStore(t *testing.T) Store {
	t.Helper()
	requirePG(t)
	project := pgTestProject(t)
	st, err := Open(context.Background(), "", project, Postgres, pgTestDSN)
	if err != nil {
		t.Fatalf("open pg store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// pgTestProject returns the per-run unique project for a test (the schema
// tag makes every test's schema fresh; the project name is the schema's
// project column value).
func pgTestProject(t *testing.T) string {
	return pgSchemaTag + "-" + strings.ReplaceAll(t.Name(), "/", "-")
}

// ── shared parity helpers (the S-1 rule: the same assertions, both dialects) ──

// runObservationRoundTrip: Add + Get round trip — sync id format, provenance,
// virtual state. The SAME body the SQLite tests assert (Task 2's pattern).
func runObservationRoundTrip(t *testing.T, st Store, project string) {
	t.Helper()
	ctx := context.Background()
	res, err := st.AddObservation(ctx, AddObservationParams{
		SessionID: "ses-1", Type: "decision", Title: "Use Postgres",
		Content: "**What** chose Postgres\n**Why** tsvector FTS + pgvector",
		Project: project, Scope: "project",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if res.ID == 0 || !strings.HasPrefix(res.SyncID, "obs-") || len(res.SyncID) != 4+16 {
		t.Fatalf("bad result: %+v", res)
	}
	got, err := st.GetObservation(ctx, res.ID, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "Use Postgres" || got.Provenance.CreatedAt == "" || got.State() != "active" {
		t.Fatalf("bad row: %+v", got)
	}
	if got.Project == nil || *got.Project != project {
		t.Fatalf("project not stored: %+v", got)
	}
	// the nullable fields come back nil, not empty (last_seen_at is set by
	// the INSERT default now(), so it's non-nil on a fresh row)
	if got.ToolName != nil || got.TopicKey != nil {
		t.Fatalf("nullable fields must be nil on a fresh row: %+v", got)
	}
}

// runFTSSync: the FTS index must track every observation mutation — the
// insert trigger indexes a new row, and a title update must remove the old
// index entry before the new one is indexed (the BEFORE UPDATE trigger).
func runFTSSync(t *testing.T, st Store, project string) {
	t.Helper()
	ctx := context.Background()
	res, err := st.AddObservation(ctx, AddObservationParams{
		SessionID: "s", Type: "decision", Title: "Use trigram search",
		Content: "trigram recall beats word splitting for C++",
		Project: project, Scope: "project",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	r, err := st.Search(ctx, SearchQuery{Query: "trigram", Project: project, Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Observations) != 1 || r.Observations[0].ID != res.ID {
		t.Fatalf("fts insert trigger broken: %+v", r)
	}
	// update the title — the BEFORE UPDATE trigger must remove the old entry
	title := "Bigger trigrams"
	if err := st.UpdateObservationForProject(ctx, res.ID, project, UpdateParams{Title: &title}); err != nil {
		t.Fatalf("update: %v", err)
	}
	r2, err := st.Search(ctx, SearchQuery{Query: "Bigger", Project: project, Limit: 10})
	if err != nil {
		t.Fatalf("search bigger: %v", err)
	}
	if len(r2.Observations) != 1 || r2.Observations[0].ID != res.ID {
		t.Fatalf("fts update trigger broken: %+v", r2)
	}
	r3, err := st.Search(ctx, SearchQuery{Query: "Use", Project: project, Limit: 10})
	if err != nil {
		t.Fatalf("search stale: %v", err)
	}
	for _, obs := range r3.Observations {
		if obs.ID == res.ID {
			t.Fatalf("stale FTS row after update: %+v", obs)
		}
	}
}

// runSearchParity: the composite rank — two otherwise-identical rows differ
// ONLY in last_seen_at / revision_count; the fresher, more stable row must
// rank first (the pinned/recency/stability boosts are load-bearing).
func runSearchParity(t *testing.T, st Store, project string) {
	t.Helper()
	ctx := context.Background()
	seed := func() int64 {
		res, err := st.AddObservation(ctx, AddObservationParams{
			SessionID: "s", Type: "decision", Title: "identical title",
			Content: "identical content body",
			Project: project, Scope: "project",
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		return res.ID
	}
	a := seed()
	b := seed()
	raw := st.(interface {
		TouchForTest(context.Context, int64, string, int) error
	})
	// b is fresher and more stable; a is stale. Absolute parseable
	// timestamps (the rank formula measures age from the timestamp).
	future := time.Now().UTC().Add(30 * time.Minute).Format("2006-01-02 15:04:05")
	stale := time.Now().UTC().Add(-30 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	if err := raw.TouchForTest(ctx, b, future, 3); err != nil {
		t.Fatalf("touch b: %v", err)
	}
	if err := raw.TouchForTest(ctx, a, stale, 0); err != nil {
		t.Fatalf("touch a: %v", err)
	}
	r, err := st.Search(ctx, SearchQuery{Query: "identical", Project: project, Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Observations) != 2 {
		t.Fatalf("expected both rows, got %+v", r.Observations)
	}
	if r.Observations[0].ID != b {
		t.Fatalf("recency/stability boosts must order b first: %+v", r.Observations)
	}
}

// runShortTermFallback: a term below the trigram minimum (< 3 runes) must
// still match via the bounded LIKE/pg_trgm fallback, project-scoped.
func runShortTermFallback(t *testing.T, st Store, project string) {
	t.Helper()
	ctx := context.Background()
	if _, err := st.AddObservation(ctx, AddObservationParams{
		SessionID: "s", Type: "discovery", Title: "Go",
		Content: "Go 1.26 is the floor for the v2 module",
		Project: project, Scope: "project",
	}); err != nil {
		t.Fatalf("add: %v", err)
	}
	r, err := st.Search(ctx, SearchQuery{Query: "go", Project: project, Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Observations) != 1 || !strings.Contains(r.Observations[0].Title, "Go") {
		t.Fatalf("short-term fallback broken (<3-rune term): %+v", r)
	}
}

// ── the PG parity suite (brief Step 1) ──────────────────────────────────────

// TestPGStoreParity [pg-dialect-parity]: the SAME test bodies as the SQLite
// parity tests run against the PG store — observation round trip, FTS sync,
// composite-rank search, short-term fallback.
func TestPGStoreParity(t *testing.T) {
	st := openPGTestStore(t)
	proj := pgTestProject(t)
	runObservationRoundTrip(t, st, proj)
	runFTSSync(t, st, proj)
	runSearchParity(t, st, proj)
	runShortTermFallback(t, st, proj)
}

// TestPGMigrationFromEmpty [migration-runs-from-empty-pg]: fresh schema
// applies migrations_pg in order from empty; close/reopen succeeds and the
// store is usable (idempotent — applied migrations are skipped).
func TestPGMigrationFromEmpty(t *testing.T) {
	requirePG(t)
	project := pgSchemaTag + "-migr-" + strings.ReplaceAll(t.Name(), "/", "-")
	ctx := context.Background()
	st, err := Open(ctx, "", project, Postgres, pgTestDSN)
	if err != nil {
		t.Fatalf("fresh open: %v", err)
	}
	// the full schema must be present after the first open
	raw := st.(*pgStore)
	for _, table := range []string{"observations", "sync_mutations", "observation_save_operations", "user_prompts", "memory_relations"} {
		found, err := raw.Probe(`SELECT COUNT(*) > 0 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = $1 AND c.relname = $2`, raw.schema, table)
		if err != nil {
			t.Fatalf("probe %s: %v", table, err)
		}
		if !found {
			t.Fatalf("missing table %s after fresh open", table)
		}
	}
	// the tsv column + GIN index + pgvector HNSW index must be present
	tsvCol, err := raw.Probe(`SELECT COUNT(*) > 0 FROM information_schema.columns WHERE table_schema = $1 AND table_name = 'observations' AND column_name = 'tsv'`, raw.schema)
	if err != nil || !tsvCol {
		t.Fatalf("tsv column missing after fresh open: %v", err)
	}
	hnswIdx, err := raw.Probe(`SELECT COUNT(*) > 0 FROM pg_indexes WHERE schemaname = $1 AND indexname = 'idx_obs_embedding_hnsw'`, raw.schema)
	if err != nil || !hnswIdx {
		t.Fatalf("HNSW vector index missing after fresh open: %v", err)
	}
	// index_meta records the applied migrations
	mig, err := raw.Probe(fmt.Sprintf(`SELECT COUNT(*) > 0 FROM "%s".index_meta WHERE key = 'migration:001_schema.sql'`, raw.schema))
	if err != nil || !mig {
		t.Fatalf("index_meta must record migration:001_schema.sql: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// reopen: migrations must be idempotent and the store usable
	st2, err := Open(ctx, "", project, Postgres, pgTestDSN)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()
	res, err := st2.AddObservation(ctx, AddObservationParams{
		SessionID: "s", Type: "learning", Title: "after reopen",
		Content: "the store works after an idempotent reopen",
		Project: project, Scope: "project",
	})
	if err != nil {
		t.Fatalf("add after reopen: %v", err)
	}
	r, err := st2.Search(ctx, SearchQuery{Query: "idempotent", Project: project, Limit: 10})
	if err != nil {
		t.Fatalf("search after reopen: %v", err)
	}
	if len(r.Observations) != 1 || r.Observations[0].ID != res.ID {
		t.Fatalf("search after reopen broken: %+v", r)
	}
}

// TestPGMigrationReopenSkipsAppliedFiles [migration-runs-from-empty-pg]:
// the runner records applied migrations in index_meta and MUST skip them on
// re-run — proven directly: a re-run of the fully-applied set leaves the
// catalog object count unchanged (the SQLite leg's parity pattern, re-run
// against migrations_pg/ counting pg_class rows).
func TestPGMigrationReopenSkipsAppliedFiles(t *testing.T) {
	requirePG(t)
	project := pgSchemaTag + "-skip-" + strings.ReplaceAll(t.Name(), "/-", "-")
	ctx := context.Background()
	st, err := Open(ctx, "", project, Postgres, pgTestDSN)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	raw := st.(*pgStore)
	before, err := pgCatalogObjectCount(raw)
	if err != nil {
		t.Fatalf("count before: %v", err)
	}
	if before == 0 {
		t.Fatalf("expected a populated catalog after the first open")
	}
	// an explicit Migrate re-run must skip every applied file
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate re-run: %v", err)
	}
	after, err := pgCatalogObjectCount(raw)
	if err != nil {
		t.Fatalf("count after: %v", err)
	}
	if before != after {
		t.Fatalf("Migrate re-run changed the catalog: %d before, %d after — applied files were not skipped", before, after)
	}
}

// pgCatalogObjectCount returns the number of tables/indexes in the project
// schema — a re-run-everything migration runner would inflate it.
func pgCatalogObjectCount(raw *pgStore) (int, error) {
	return raw.ProbeInt(`SELECT COUNT(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = $1 AND c.relkind IN ('r','i','v')`, raw.schema)
}

// ── TRIANGULATE (beyond the brief's set) ────────────────────────────────────

// TestPGVectorSearchParity: the pgvector leg — ORDER BY the cosine-distance
// operator. A query close to a stored vector ranks that row first; an empty
// vector set degrades (never a silent empty-that-passes-as-success).
func TestPGVectorSearchParity(t *testing.T) {
	st := openPGTestStore(t)
	ctx := context.Background()
	proj := pgTestProject(t)
	raw := st.(*pgStore)

	// two rows with orthogonal 2-D embeddings (stored as the vector literal)
	// pgvector pins vector(768) in the migration; pad 2-D test vectors to 768.
	pad := func(v ...float32) string {
		full := make([]float32, 768)
		copy(full, v)
		return "[" + strings.Join(fmts(full), ",") + "]"
	}
	addVec := func(title string, v ...float32) int64 {
		res, err := st.AddObservation(ctx, AddObservationParams{
			SessionID: "s", Type: "discovery", Title: title, Content: "content " + title,
			Project: proj, Scope: "project",
		})
		if err != nil {
			t.Fatalf("add %s: %v", title, err)
		}
		if err := raw.RawExec(`UPDATE `+raw.qualified("observations")+` SET embedding = $1::vector WHERE id = $2`, pad(v...), res.ID); err != nil {
			t.Fatalf("set embedding %s: %v", title, err)
		}
		return res.ID
	}
	a := addVec("near", 0.99, 0.01)
	_ = addVec("far", 0.01, 0.99)

	// the query vector must be 768-D (matching the stored vectors)
	queryVec := make([]float32, 768)
	queryVec[0], queryVec[1] = 0.98, 0.02
	res, err := st.VectorSearch(ctx, VecQuery{QueryVector: queryVec, Project: proj, TopK: 2})
	if err != nil {
		t.Fatalf("vector search: %v", err)
	}
	if res.Degraded {
		t.Fatalf("vectors present → the leg must not degrade: %+v", res)
	}
	if len(res.Observations) < 1 {
		t.Fatalf("expected at least one vector result: %+v", res)
	}
	if res.Observations[0].ID != a {
		t.Fatalf("closest row must rank first, got %d (want %d)", res.Observations[0].ID, a)
	}

	// the empty-vector leg degrades
	empty, err := st.VectorSearch(ctx, VecQuery{QueryVector: nil, Project: proj, TopK: 5})
	if err != nil {
		t.Fatalf("vector search (empty) must not error: %v", err)
	}
	if !empty.Degraded {
		t.Fatalf("empty query vector → the leg must degrade: %+v", empty)
	}
}

// TestPGConcurrentWrites [pg-dialect-parity, triangulate]: N concurrent
// saves through one store must all commit (PG row-level locking; the SQLite
// leg's _txlock serialization has no PG twin, but the Store contract is
// the same — no lost writes).
func TestPGConcurrentWrites(t *testing.T) {
	st := openPGTestStore(t)
	ctx := context.Background()
	proj := pgTestProject(t)
	const n = 20
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := st.AddObservation(ctx, AddObservationParams{
				SessionID: "s", Type: "learning",
				Title:   fmt.Sprintf("concurrent save %d", i),
				Content: fmt.Sprintf("the %d-th concurrent write body", i),
				Project: proj, Scope: "project",
			})
			if err != nil {
				errs <- fmt.Errorf("save %d: %w", i, err)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent save failed: %v", err)
	}
	// all N rows must be present (no lost writes)
	raw := st.(*pgStore)
	n2, err := raw.ProbeInt(`SELECT COUNT(*) FROM ` + raw.qualified("observations"))
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n2 != n {
		t.Fatalf("lost writes: expected %d rows, got %d", n, n2)
	}
}

// TestPGSyncJournalParity [pg-dialect-parity, triangulate]: the sync domain
// (EnqueueMutation / ListPendingMutations / Tombstone) works on PG with the
// same shapes — seq is returned, pending list is ordered, tombstone upserts.
func TestPGSyncJournalParity(t *testing.T) {
	st := openPGTestStore(t)
	ctx := context.Background()
	seq1, err := st.EnqueueMutation(ctx, Mutation{
		TargetKey: "local", Entity: "observation", EntityKey: "obs-1",
		Op: "upsert", Payload: "{}", Source: "local", Project: "p",
	})
	if err != nil {
		t.Fatalf("enqueue 1: %v", err)
	}
	seq2, err := st.EnqueueMutation(ctx, Mutation{
		TargetKey: "local", Entity: "observation", EntityKey: "obs-2",
		Op: "delete", Payload: "{}", Source: "local", Project: "p",
	})
	if err != nil {
		t.Fatalf("enqueue 2: %v", err)
	}
	if seq2 <= seq1 {
		t.Fatalf("seq must increase: %d then %d", seq1, seq2)
	}
	pending, err := st.ListPendingMutations(ctx, 0, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pending) != 2 || pending[0].EntityKey != "obs-1" || pending[1].EntityKey != "obs-2" {
		t.Fatalf("pending list wrong: %+v", pending)
	}
	// the cursor: only mutations after seq1
	after, err := st.ListPendingMutations(ctx, seq1, 100)
	if err != nil {
		t.Fatalf("list cursor: %v", err)
	}
	if len(after) != 1 || after[0].Seq != seq2 {
		t.Fatalf("cursor list wrong: %+v", after)
	}
	// tombstone upsert: a second call with the same key updates, not fails
	if err := st.Tombstone(ctx, "observation", "obs-1"); err != nil {
		t.Fatalf("tombstone 1: %v", err)
	}
	if err := st.Tombstone(ctx, "observation", "obs-1"); err != nil {
		t.Fatalf("tombstone 2 (upsert): %v", err)
	}
}

// TestPGPromptParity [pg-fts-tsvector, triangulate]: SearchPrompts over the
// prompts tsvector returns the full PromptRow identity (the SQLite leg's
// SearchPromptsReturnsFullRow contract on PG).
func TestPGPromptParity(t *testing.T) {
	st := openPGTestStore(t)
	ctx := context.Background()
	proj := pgTestProject(t)
	raw := st.(*pgStore)
	id, err := raw.InsertPrompt(ctx, "ses-p", "the deployment runs on port 7431", proj)
	if err != nil {
		t.Fatalf("insert prompt: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero prompt id")
	}
	r, err := st.SearchPrompts(ctx, PromptSearchQuery{Query: "deployment", Project: proj, Limit: 10})
	if err != nil {
		t.Fatalf("search prompts: %v", err)
	}
	if len(r.Prompts) != 1 {
		t.Fatalf("expected exactly 1 prompt hit, got %+v", r.Prompts)
	}
	row := r.Prompts[0]
	if row.ID != id {
		t.Fatalf("bad id: got %d want %d", row.ID, id)
	}
	if row.SyncID == "" || row.SessionID != "ses-p" || row.Project != proj {
		t.Fatalf("identity fields not populated: %+v", row)
	}
	if row.Content != "the deployment runs on port 7431" {
		t.Fatalf("bad content: %q", row.Content)
	}
}

// TestPGIdentifierEdge [pg-dialect-parity, triangulate]: project ids with
// '/' render to a schema name with '-' (Canonicalize: '/' → '-'), and
// distinct project ids get distinct schemas (S-3: one schema per project).
// A search on one project never leaks the other's rows.
func TestPGIdentifierEdge(t *testing.T) {
	requirePG(t)
	ctx := context.Background()
	base := pgSchemaTag + "-edge"
	// "org/alpha" canonicalizes to "org-alpha" (with a dash)
	stA, err := Open(ctx, "", base+"/alpha", Postgres, pgTestDSN)
	if err != nil {
		t.Fatalf("open a: %v", err)
	}
	defer stA.Close()
	// "orgbeta" has no slash — different schema
	stB, err := Open(ctx, "", base+"beta", Postgres, pgTestDSN)
	if err != nil {
		t.Fatalf("open b: %v", err)
	}
	defer stB.Close()
	rawA := stA.(*pgStore)
	rawB := stB.(*pgStore)
	// the two project ids MUST resolve to distinct schemas
	if rawA.schema == rawB.schema {
		t.Fatalf("distinct project ids must get distinct schemas: both %q", rawA.schema)
	}
	// the '/' project must have a '-' in its schema name (Canonicalize)
	if !strings.Contains(rawA.schema, "-") {
		t.Fatalf("Canonicalize must map '/' → '-': %q has no dash", rawA.schema)
	}
	projA := base + "/alpha"
	projB := base + "beta"
	resB, err := stB.AddObservation(ctx, AddObservationParams{
		SessionID: "s-b", Type: "discovery", Title: "unique edge term",
		Content: "unique edge term content", Project: projB, Scope: "project",
	})
	if err != nil {
		t.Fatalf("add b: %v", err)
	}
	// searching projA (different schema) must not leak projB's row
	rA, err := stA.Search(ctx, SearchQuery{Query: "unique", Project: projA, Limit: 10})
	if err != nil {
		t.Fatalf("search a: %v", err)
	}
	if len(rA.Observations) != 0 {
		t.Fatalf("proj-a search leaked proj-b rows: %+v", rA)
	}
	rB, err := stB.Search(ctx, SearchQuery{Query: "unique", Project: projB, Limit: 10})
	if err != nil {
		t.Fatalf("search b: %v", err)
	}
	if len(rB.Observations) != 1 || rB.Observations[0].ID != resB.ID {
		t.Fatalf("proj-b search broken: %+v", rB)
	}
}

// fmts formats a float slice as decimal strings for the pgvector literal.
func fmts(v []float32) []string {
	out := make([]string, len(v))
	for i, f := range v {
		out[i] = fmt.Sprintf("%g", f)
	}
	return out
}
