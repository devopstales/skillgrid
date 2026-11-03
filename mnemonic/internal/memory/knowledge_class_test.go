package memory

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// ADR-0038 (TICKET-03): observations carry a nullable knowledge_class
// durability dimension (episodic | convention | doc | unclassified) with a
// type-driven heuristic inference on save and a re-runnable backfill.

// TestSave_KnowledgeClassInfer: a type=decision save without knowledge_class
// reads back the inferred class "convention".
func TestSave_KnowledgeClassInfer(t *testing.T) {
	fx := newFixture(t, "kc-test")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: session1,
		Type:      "decision",
		Title:     "Adopt knowledge class dimension",
		Content:   "Why: durability distinction. Where: internal/memory.",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := fx.svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.KnowledgeClass != "convention" {
		t.Errorf("knowledge_class = %q, want %q", got.KnowledgeClass, "convention")
	}
}

// TestSave_KnowledgeClassInfer_Episodic: a type=session_log save reads back
// the inferred class "episodic".
func TestSave_KnowledgeClassInfer_Episodic(t *testing.T) {
	fx := newFixture(t, "kc-test")
	ctx := context.Background()
	id, err := fx.svc.Save(ctx, SaveInput{
		SessionID: session1,
		Type:      "session_log",
		Title:     "Session log for today",
		Content:   "What happened this session.",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := fx.svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.KnowledgeClass != "episodic" {
		t.Errorf("knowledge_class = %q, want %q", got.KnowledgeClass, "episodic")
	}
}

// seedClassRow inserts one observation row with a NULL knowledge_class, the
// pre-backfill shape the backfill must fill from the type column. It writes
// core + side rows + FTS directly (the INSTEAD OF insert trigger leaves
// LastInsertId stale, so the test cannot recover the new id from the view;
// the explicit side rows mirror what the trigger would fan out, minus the
// columns the search read path COALESCEs).
func seedClassRow(t *testing.T, st *store.Store, project, typ, title, content string) int64 {
	t.Helper()
	res, err := st.DB.Exec(`
		INSERT INTO observations_core (
			session_id, type, title, content, project, scope,
			normalized_hash, revision_count, created_at, updated_at
		) VALUES ('s1', ?, ?, ?, ?, 'project', ?, 0, '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z')`,
		typ, title, content, project, normalizedHash(title, content, typ))
	if err != nil {
		t.Fatalf("seed %s row: %v", typ, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("seed last insert id: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO observation_state (observation_id, status, visibility, expires_at, pinned)
		VALUES (?, 'active', 'private', NULL, 0)`, id); err != nil {
		t.Fatalf("seed state %s row: %v", typ, err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO observation_provenance (observation_id, source, owner)
		VALUES (?, 'agent', 's1')`, id); err != nil {
		t.Fatalf("seed provenance %s row: %v", typ, err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO observation_stats (observation_id, duplicate_count, retrieval_usage, importance_score, recency_decay, maturity_tier)
		VALUES (?, 0, 0, 0, 1, 'fresh')`, id); err != nil {
		t.Fatalf("seed stats %s row: %v", typ, err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO observation_temporal (observation_id, valid_at)
		VALUES (?, '2026-01-02T00:00:00Z')`, id); err != nil {
		t.Fatalf("seed temporal %s row: %v", typ, err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO observations_fts(rowid, title, content, type, project)
		VALUES (?, ?, ?, ?, ?)`, id, title, content, typ, project); err != nil {
		t.Fatalf("seed fts %s row: %v", typ, err)
	}
	return id
}

func nullClassCount(t *testing.T, st *store.Store, project string) int {
	t.Helper()
	var n int
	if err := st.DB.QueryRow(`
		SELECT COUNT(*) FROM observations
		WHERE project = ? AND knowledge_class IS NULL`, project).Scan(&n); err != nil {
		t.Fatalf("count NULL knowledge_class: %v", err)
	}
	return n
}

// TestBackfill_Rerunnable: after seeding NULL-class rows, BackfillKnowledgeClass
// fills them from the type heuristic; a second run changes nothing (no-op).
func TestBackfill_Rerunnable(t *testing.T) {
	fx := newFixture(t, "kc-test")
	ctx := context.Background()
	// Seed both rows straight into observations_core with no side rows: the
	// backfill is a pure core-table UPDATE and Get falls back to column
	// defaults for missing side rows, so this is the minimal pre-backfill
	// shape (the view seed would add side rows the test does not need).
	idC := seedCoreRow(t, fx.st, "kc-test", "decision", "Convention row for backfill", "Body A")
	idE := seedCoreRow(t, fx.st, "kc-test", "session_log", "Episodic row for backfill", "Body B")

	// Sanity: both rows are NULL before the backfill (no save-path inference
	// touched them — they went straight into the table).
	if n := nullClassCount(t, fx.st, "kc-test"); n != 2 {
		t.Fatalf("NULL-class rows before backfill = %d, want 2", n)
	}

	n, err := fx.svc.BackfillKnowledgeClass(ctx)
	if err != nil {
		t.Fatalf("backfill 1: %v", err)
	}
	if n != 2 {
		t.Fatalf("backfill 1 affected = %d, want 2", n)
	}

	gotC, err := fx.svc.Get(ctx, idC)
	if err != nil {
		t.Fatalf("get convention: %v", err)
	}
	if gotC.KnowledgeClass != "convention" {
		t.Errorf("backfilled decision row = %q, want %q", gotC.KnowledgeClass, "convention")
	}
	gotE, err := fx.svc.Get(ctx, idE)
	if err != nil {
		t.Fatalf("get episodic: %v", err)
	}
	if gotE.KnowledgeClass != "episodic" {
		t.Errorf("backfilled session_log row = %q, want %q", gotE.KnowledgeClass, "episodic")
	}

	// Second run: nothing left to fill.
	n, err = fx.svc.BackfillKnowledgeClass(ctx)
	if err != nil {
		t.Fatalf("backfill 2: %v", err)
	}
	if n != 0 {
		t.Errorf("backfill 2 affected = %d, want 0 (re-runnable no-op)", n)
	}
	if n := nullClassCount(t, fx.st, "kc-test"); n != 0 {
		t.Errorf("NULL-class rows after backfills = %d, want 0", n)
	}
}

// seedCoreRow inserts a minimal pre-backfill shape: a bare observations_core
// row plus the one side row the Get read path strictly needs (pr.source is
// scanned unqualified; every other side column COALESCEs to its default). No
// FTS entry — the backfill test reads through Get, not search.
func seedCoreRow(t *testing.T, st *store.Store, project, typ, title, content string) int64 {
	t.Helper()
	res, err := st.DB.Exec(`
		INSERT INTO observations_core (
			session_id, type, title, content, project, scope,
			normalized_hash, created_at, updated_at
		) VALUES ('s1', ?, ?, ?, ?, 'project', ?, '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z')`,
		typ, title, content, project, normalizedHash(title, content, typ))
	if err != nil {
		t.Fatalf("seed core %s row: %v", typ, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("seed core last insert id: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO observation_provenance (observation_id, source)
		VALUES (?, 'agent')`, id); err != nil {
		t.Fatalf("seed core provenance %s row: %v", typ, err)
	}
	return id
}

// seedSearchRows seeds five FTS-searchable rows spanning the class space:
// two convention (decision, convention), two episodic (session_log,
// discovery), and one that the heuristic maps to unclassified (pattern),
// and returns their ids in seed order. Every row's content carries the
// test-unique term kebabcase: FTS is global per project, so a shared term
// would match rows seeded by other tests in the same store.
func seedSearchRows(t *testing.T, st *store.Store, project string) []int64 {
	t.Helper()
	return []int64{
		seedClassRow(t, st, project, "decision", "Auth uses token rotation policy", "kebabcase alpha body"),
		seedClassRow(t, st, project, "convention", "Naming rules are kebab-case", "kebabcase beta body"),
		seedClassRow(t, st, project, "session_log", "Session notes from the morning run", "kebabcase gamma body"),
		seedClassRow(t, st, project, "discovery", "Found a parser off-by-one bug", "kebabcase delta body"),
		seedClassRow(t, st, project, "pattern", "Legacy row without class", "kebabcase epsilon body"),
	}
}

// TestSearch_KnowledgeClassFilter: with knowledge_class=convention, the search
// returns only convention rows (the filter is exact-match; unclassified rows
// are not convention).
func TestSearch_KnowledgeClassFilter(t *testing.T) {
	fx := newFixture(t, "kc-test")
	ctx := context.Background()
	ids := seedSearchRows(t, fx.st, "kc-test")
	if _, err := fx.svc.BackfillKnowledgeClass(ctx); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	// "kebabcase" is in every seeded row's body (and in no other row of this
	// project), so the match set is exactly the five seeded rows; the
	// convention filter must then return exactly the two convention rows and
	// drop the two episodic + one unclassified rows. FTS is global per
	// project, so the query must be a word unique to THIS test's rows (see
	// TestSearch_UnfilteredUnchanged).
	const q = "kebabcase"
	// The filter must return exactly this test's two convention rows (the
	// decision + convention seeds); every other matched row — the two
	// episodic and the one unclassified — must be dropped.
	hits, err := fx.svc.SearchWithKnowledgeClass(ctx, q, "any", "convention", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	want := map[int64]bool{ids[0]: true, ids[1]: true}
	if len(hits) != len(want) {
		t.Fatalf("filtered hits = %d, want %d (only convention rows)", len(hits), len(want))
	}
	for _, h := range hits {
		if !want[h.ID] {
			t.Errorf("filtered search returned non-convention row %d (%q)", h.ID, h.Title)
		}
		if h.KnowledgeClass != "convention" {
			t.Errorf("hit %d labeled %q, want %q", h.ID, h.KnowledgeClass, "convention")
		}
	}
	// The unfiltered read of the same query sees all five rows, so the
	// filtered path dropped exactly three (the two episodic + one
	// unclassified): the filter is doing work, not a no-op.
	all, err := fx.svc.SearchWithKnowledgeClass(ctx, q, "any", "", 10)
	if err != nil {
		t.Fatalf("search all: %v", err)
	}
	if len(all) != len(ids) {
		t.Fatalf("unfiltered search = %d rows, want %d (all seeded rows)", len(all), len(ids))
	}
	if len(all) != len(hits)+3 {
		t.Fatalf("filter dropped %d rows, want exactly 3 (2 episodic + 1 unclassified)", len(all)-len(hits))
	}
	// The same query with the episodic filter returns the complementary pair
	// (triangulate: the filter is exact-match, not "not convention").
	episodic, err := fx.svc.SearchWithKnowledgeClass(ctx, q, "any", "episodic", 10)
	if err != nil {
		t.Fatalf("search episodic: %v", err)
	}
	wantE := map[int64]bool{ids[2]: true, ids[3]: true}
	if len(episodic) != len(wantE) {
		t.Fatalf("episodic hits = %d, want %d", len(episodic), len(wantE))
	}
	for _, h := range episodic {
		if !wantE[h.ID] {
			t.Errorf("episodic search returned non-episodic row %d (%q)", h.ID, h.Title)
		}
	}
}

// TestSearch_UnfilteredUnchanged: a search without a knowledge_class filter
// returns exactly the pre-ADR-0038 row set (including unclassified rows) with
// per-row labeling — no regression from adding the dimension.
func TestSearch_UnfilteredUnchanged(t *testing.T) {
	fx := newFixture(t, "kc-test")
	ctx := context.Background()
	ids := seedSearchRows(t, fx.st, "kc-test")
	if _, err := fx.svc.BackfillKnowledgeClass(ctx); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	// The pre-dimension contract: every live, non-superseded row matching the
	// query is returned, regardless of class. The empty-class call must
	// produce the SAME row set as the plain Search — the no-regression check.
	// "kebabcase" is in every seeded row's body (and in no other row of this
	// project), so the unfiltered match set is exactly the five seeded rows
	// (the pre-ADR-0038 contract: every live, non-superseded matching row,
	// regardless of class). Plain Search is the pre-ADR-0038 read path: the
	// row set is unchanged by the new dimension (the class column is
	// additive on the hot-read shape; labeling below proves the filtered
	// path surfaces it). FTS is global per project (rows carry no session),
	// so the query must be a word unique to THIS test's rows — a shared term
	// would match rows seeded by other tests in the same store.
	const q = "kebabcase"
	hits, err := fx.svc.Search(ctx, q, "any", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	filtered, err := fx.svc.SearchWithKnowledgeClass(ctx, q, "any", "", 10)
	if err != nil {
		t.Fatalf("search unfiltered: %v", err)
	}
	if len(filtered) != len(hits) {
		t.Fatalf("SearchWithKnowledgeClass(\"\") = %d hits, plain Search = %d (unfiltered must be unchanged)", len(filtered), len(hits))
	}
	plainIDs := map[int64]bool{}
	for _, h := range hits {
		plainIDs[h.ID] = true
	}
	for _, h := range filtered {
		if !plainIDs[h.ID] {
			t.Errorf("unfiltered search returned row %d that plain Search missed", h.ID)
		}
	}
	// Every seeded row must be in the unfiltered row set (the pre-ADR-0038
	// contract: no live matching row is dropped by the new dimension). The
	// set is allowed to be SUPERSET only by rows other tests seeded in the
	// same project store (shared temp dir) whose content happens to contain
	// the query term — never the other way around.
	byID := map[int64]Observation{}
	for _, h := range hits {
		byID[h.ID] = h
	}
	for _, id := range ids {
		if _, ok := byID[id]; !ok {
			t.Errorf("unfiltered search dropped seeded row %d", id)
		}
	}
	// Per-class labeling on the filtered (class-aware) read path: the same
	// unfiltered row set, with backfilled rows carrying their inferred class
	// and the heuristic-uncovered type surfaced as unclassified.
	labelByID := map[int64]string{}
	for _, h := range filtered {
		labelByID[h.ID] = h.KnowledgeClass
	}
	if got := labelByID[ids[0]]; got != "convention" {
		t.Errorf("row %d labeled %q, want %q", ids[0], got, "convention")
	}
	if got := labelByID[ids[2]]; got != "episodic" {
		t.Errorf("row %d labeled %q, want %q", ids[2], got, "episodic")
	}
	if got := labelByID[ids[4]]; got != "unclassified" {
		t.Errorf("row %d labeled %q, want %q", ids[4], got, "unclassified")
	}
}
