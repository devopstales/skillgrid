package memory

// Task 16 — sync transport (spec 01-memory/006 §2-3, §7).
//
// The four acceptance scenarios (acceptance.feature, G14):
//
//	[push-acks-seq]                 — a 100-mutation batch acks to the batch
//	                                  end; a 101-mutation batch is refused.
//	[pull-scoped-to-project-schema] — a pull for project "a" returns only
//	                                  schema_a's mutations and touches no
//	                                  other project's schema (the core
//	                                  isolation constraint).
//	[schema-per-project-fail-fast]  — a project whose schema cannot be
//	                                  resolved fails EnsureSchema fast — no
//	                                  public fallback.
//	[export-import-round-trip]      — export → import into a fresh store
//	                                  lands every observation by sync id;
//	                                  re-import is a no-op (chunk dedupe).
//
// The chunk-id-stable scenario is covered in chunk_test.go.

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ────────────────────────────────────────────────────────────────────────────
// fake transport (the in-memory MutationTransport adapter — the second
// adapter of the seam, spec 006 §5: PG (prod) + fake (tests)).
// ────────────────────────────────────────────────────────────────────────────

// fakeTransport is the in-memory MutationTransport: one bucket per project
// schema. Push appends to the bucket, Pull reads the bucket (and records the
// schemas it queried, so the scoping assertions are observable).
type fakeTransport struct {
	buckets   map[string][]store.Mutation // project → appended mutations
	sequences map[string]int64            // project → next seq to assign
	touched   []string                    // projects whose schema was queried
	fails     map[string]bool             // project → EnsureSchema fails
}

// newFakeTransport builds an empty fake transport.
func newFakeTransport(t *testing.T) *fakeTransport {
	t.Helper()
	return &fakeTransport{
		buckets:   map[string][]store.Mutation{},
		sequences: map[string]int64{},
		fails:     map[string]bool{},
	}
}

// seed pre-loads n mutations for a project schema (a pull fixture).
func (f *fakeTransport) seed(project string, n int) {
	for i := 0; i < n; i++ {
		f.ack(project, store.Mutation{TargetKey: "cloud", Entity: "observation",
			EntityKey: "k", Op: "upsert", Payload: pulledObsPayload("k", "s", "T", "C"),
			Source: "local", Project: project})
	}
}

// ack appends one mutation to a project bucket with the next seq.
func (f *fakeTransport) ack(project string, m store.Mutation) int64 {
	f.sequences[project]++
	m.Seq = f.sequences[project]
	f.buckets[project] = append(f.buckets[project], m)
	return m.Seq
}

// schemasTouched returns the projects whose schema a pull queried (the
// isolation assertion surface — a pull for "a" must never touch "b").
func (f *fakeTransport) schemasTouched() string {
	return strings.Join(f.touched, ",")
}

// failResolve makes EnsureSchema fail for the project (an unresolvable
// schema — the fail-fast fixture).
func (f *fakeTransport) failResolve(project string) {
	f.fails[project] = true
}

// EnsureSchema resolves the project's schema (fail-fast fixture: a marked
// project cannot resolve and there is no public fallback).
func (f *fakeTransport) EnsureSchema(ctx context.Context, project string) error {
	if f.fails[project] {
		return &transportSchemaError{project: project}
	}
	f.touched = append(f.touched, project)
	_, ok := f.buckets[project]
	if !ok {
		f.buckets[project] = nil
	}
	return nil
}

// PushMutations appends a batch to the project bucket (one logical
// transaction; the batch-cap and size limits are enforced at the seam
// boundary, the brief's contract).
func (f *fakeTransport) PushMutations(ctx context.Context, project string, batch []store.Mutation) (int64, error) {
	if len(batch) == 0 {
		return f.sequences[project], nil
	}
	total := 0
	for _, m := range batch {
		total += len(m.Payload)
	}
	last := int64(0)
	for _, m := range batch {
		last = f.ack(project, m)
	}
	_ = total
	return last, nil
}

// PullMutations returns the project bucket's mutations after sinceSeq,
// recording the schema it queried (the scoping assertion).
func (f *fakeTransport) PullMutations(ctx context.Context, project string, sinceSeq int64) ([]store.Mutation, error) {
	f.touched = append(f.touched, project)
	var out []store.Mutation
	for _, m := range f.buckets[project] {
		if m.Seq > sinceSeq {
			out = append(out, m)
		}
	}
	return out, nil
}

// Close closes the fake (the interface method; nothing to release).
func (f *fakeTransport) Close() error { return nil }

// transportSchemaError is the fake's unresolvable-schema error.
type transportSchemaError struct{ project string }

func (e *transportSchemaError) Error() string {
	return "transport: cannot resolve schema for project " + e.project
}

// ────────────────────────────────────────────────────────────────────────────
// [push-acks-seq]
// ────────────────────────────────────────────────────────────────────────────

// TestPushAcksSeq: a full batch of one hundred mutations acks the cursor to
// the batch end (the highest seq pushed), and a batch of one hundred and
// one is refused (the seam enforces the batch cap, spec 006 §3).
func TestPushAcksSeq(t *testing.T) {
	tr := newFakeTransport(t)

	acked, err := Push(ctx, tr, "p", makeMutations(100)) // exactly the batch cap
	if err != nil {
		t.Fatalf("push 100: %v", err)
	}
	if acked != 100 {
		t.Fatalf("acked seq wrong: %d, want 100 (the batch end)", acked)
	}
	// the push actually landed: pulling the schema returns the batch.
	got, err := tr.PullMutations(ctx, "p", 0)
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if len(got) != 100 {
		t.Fatalf("push must write the batch to the schema, got %d rows", len(got))
	}

	// a batch over the cap is refused (the transport never receives it).
	if _, err := Push(ctx, tr, "p", makeMutations(101)); err == nil {
		t.Fatalf(">100 batch must be refused")
	}
	// and it wrote nothing: the bucket is still 100 rows.
	got, _ = tr.PullMutations(ctx, "p", 0)
	if len(got) != 100 {
		t.Fatalf("a refused batch must write nothing, got %d rows", len(got))
	}
}

// TestPushEmptyOutbox (TRIANGULATE [push-acks-seq], different inputs): an
// empty batch is a valid no-op push — it acks the current cursor (0 on a
// fresh schema) and writes nothing. The autosync's idle cycle rides this
// path, so it must not error.
func TestPushEmptyOutbox(t *testing.T) {
	tr := newFakeTransport(t)
	acked, err := Push(ctx, tr, "p", nil)
	if err != nil {
		t.Fatalf("push empty: %v", err)
	}
	if acked != 0 {
		t.Fatalf("empty push must ack the current cursor 0, got %d", acked)
	}
	if _, err := tr.PullMutations(ctx, "p", 0); err != nil {
		t.Fatalf("pull: %v", err)
	}
	if len(tr.buckets["p"]) != 0 {
		t.Fatalf("empty push must write nothing")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [pull-scoped-to-project-schema]
// ────────────────────────────────────────────────────────────────────────────

// TestPullScopedToProjectSchema: two project schemas seeded with different
// mutation counts; a pull for project "a" returns only a's mutations and
// touches no other schema (spec 006 §2 — the core isolation constraint).
func TestPullScopedToProjectSchema(t *testing.T) {
	tr := newFakeTransport(t)
	tr.seed("a", 3)
	tr.seed("b", 5)

	got, err := Pull(ctx, tr, "a", 0)
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("pull must be scoped to the project's schema, got %d, want 3", len(got))
	}
	// every returned mutation belongs to a's schema (project column "a").
	for _, m := range got {
		if m.Project != "a" {
			t.Fatalf("pull returned a mutation from another schema: %+v", m)
		}
	}
	// the scoping proof: the only schema queried was "a".
	if tr.schemasTouched() != "a" {
		t.Fatalf("pull touched other schemas: %q", tr.schemasTouched())
	}
}

// TestPullSinceCursor (TRIANGULATE [pull-scoped-to-project-schema],
// different inputs): the since-cursor is honored — a pull from cursor 2
// returns only mutations with seq > 2, in seq order.
func TestPullSinceCursor(t *testing.T) {
	tr := newFakeTransport(t)
	tr.seed("a", 5)

	got, err := Pull(ctx, tr, "a", 2)
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("pull from cursor 2 of 5 must return 3, got %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i].Seq <= got[i-1].Seq {
			t.Fatalf("pull must be in seq order: %d then %d", got[i-1].Seq, got[i].Seq)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [schema-per-project-fail-fast]
// ────────────────────────────────────────────────────────────────────────────

// TestSchemaPerProjectFailFast: a project whose schema cannot be resolved
// fails EnsureSchema — no public fallback, no silent no-op (spec 006 §2).
func TestSchemaPerProjectFailFast(t *testing.T) {
	tr := newFakeTransport(t)
	tr.failResolve("ghost")
	if err := EnsureSchema(ctx, tr, "ghost"); err == nil {
		t.Fatalf("unresolvable project schema must fail fast (no public fallback)")
	}
}

// TestSchemaPerProjectOk (TRIANGULATE [schema-per-project-fail-fast],
// different inputs): a resolvable project's EnsureSchema succeeds and
// idempotently re-runs (the auto-sync cycle calls it every cycle).
func TestSchemaPerProjectOk(t *testing.T) {
	tr := newFakeTransport(t)
	if err := EnsureSchema(ctx, tr, "p"); err != nil {
		t.Fatalf("resolvable schema must succeed: %v", err)
	}
	if err := EnsureSchema(ctx, tr, "p"); err != nil {
		t.Fatalf("EnsureSchema must be idempotent: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [export-import-round-trip]
// ────────────────────────────────────────────────────────────────────────────

// TestExportImportRoundTrip: a project with 20 saved observations is
// exported to a chunk archive and imported into a fresh store; every source
// observation is present on the destination by its sync id, and re-importing
// the same archive is a no-op (chunk-id dedupe through sync_chunks).
func TestExportImportRoundTrip(t *testing.T) {
	src := openTestStore(t)
	seedProject(t, src, "p", 20) // 20 observations + sessions
	dir := t.TempDir()

	man, err := Export(ctx, src, "p", dir)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if man.Project != "p" {
		t.Fatalf("manifest project wrong: %+v", man)
	}
	if len(man.Chunks) == 0 {
		t.Fatalf("manifest must list the exported chunks")
	}

	dst := openTestStore(t)
	n, err := Import(ctx, dst, "p", dir)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n == 0 {
		t.Fatalf("import must apply the archive through the store, applied %d", n)
	}

	// every source observation is present on the destination by sync id.
	for _, syncID := range syncIDsOf(t, src, "p") {
		if _, err := dst.GetObservation(ctx, findIDBySyncID(t, dst, syncID), false); err != nil {
			t.Fatalf("missing %s after import: %v", syncID, err)
		}
	}

	// re-import is idempotent (chunk id dedupe: sync_chunks).
	if _, err := Import(ctx, dst, "p", dir); err != nil {
		t.Fatalf("re-import must be idempotent: %v", err)
	}
	// and it left exactly one row per source observation (no duplicates).
	if got := liveObsCount(t, dst); got != liveObsCount(t, src) {
		t.Fatalf("re-import must not duplicate rows: dst=%d src=%d", got, liveObsCount(t, src))
	}
}

// seedProject saves n observations for project (through the store, so each
// carries a real minted sync id). The session auto-registers on the first
// save.
func seedProject(t *testing.T, st store.Store, project string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		seedObservation(t, st, project, "decision",
			"seed "+project, "seeded content "+project, "")
	}
}
