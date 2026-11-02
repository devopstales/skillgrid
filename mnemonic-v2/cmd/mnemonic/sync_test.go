package main

// sync CLI tests (Task 16) — the push/pull/export/import subcommands are
// wired to the transport + chunk codec (Task 14's stubs became real). The
// transport is injectable (installTransport swaps in a fake); the store is
// the real SQLite fixture (a single shared store so pending mutations
// survive across the two commands a test runs).

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

var errFakeTransport = errors.New("transport: fake failure")

// testPGTransport is a fake transport that records pushed batches and
// serves pulls from a pre-seeded bucket (the CLI push/pull wiring is under
// test — not the PG SQL, which lives in the memory package's PG transport).
type testPGTransport struct {
	pushed   []store.Mutation
	seeded   []store.Mutation
	ackSeq   int64
	failPush bool
	failPull bool
}

func (f *testPGTransport) EnsureSchema(ctx context.Context, project string) error { return nil }

func (f *testPGTransport) PushMutations(ctx context.Context, project string, batch []store.Mutation) (int64, error) {
	if f.failPush {
		return 0, errFakeTransport
	}
	f.pushed = append(f.pushed, batch...)
	for _, m := range batch {
		if m.Seq > f.ackSeq {
			f.ackSeq = m.Seq
		}
	}
	return f.ackSeq, nil
}

func (f *testPGTransport) PullMutations(ctx context.Context, project string, sinceSeq int64) ([]store.Mutation, error) {
	if f.failPull {
		return nil, errFakeTransport
	}
	var out []store.Mutation
	for _, m := range f.seeded {
		if m.Seq > sinceSeq {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *testPGTransport) Close() error { return nil }

// installSharedStore wires the run fixture to ONE store for the test (the
// sync push/pull fixture needs pending mutations to survive across the two
// CLI invocations a test runs). The store is opened for project "p" with
// sync journaling on and the project enrolled, so a save enqueues a pending
// mutation. The transport seam is swapped for the fake.
func installSharedStore(t *testing.T, tr memory.MutationTransport) store.Store {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(context.Background(), dataDir, "p", store.SQLite, "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	// the test keeps its own pool ref so the commands' Close does not
	// drop the last ref and close the shared handle (the pool is
	// refcounted by the sqlite path — the commands each open + close
	// their own ref).
	ref, err := store.Open(context.Background(), dataDir, "p", store.SQLite, "")
	if err != nil {
		t.Fatalf("open ref: %v", err)
	}
	t.Cleanup(func() { ref.Close() })
	seam := st.(store.SyncSeam)
	seam.SetSyncEnabled(true)
	if err := seam.SetEnrolledForTest("p"); err != nil {
		t.Fatalf("enroll: %v", err)
	}

	prevStoreFor, prevHandlerFor, prevGit := storeFor, handlerFor, gitToplevel
	prevTransport := testTransportFor
	storeFor = func(cfg config.Config, p string) (store.Store, error) { return st, nil }
	handlerFor = func(cfg config.Config, s store.Store) (*handler.Handler, error) {
		return newTestHandlerWithStore(t, s), nil
	}
	gitToplevel = func(string) (string, error) { return "", errors.New("no git in tests") }
	testTransportFor = func(cfg *memory.TransportConfig) (memory.MutationTransport, error) { return tr, nil }

	t.Cleanup(func() {
		storeFor, handlerFor, gitToplevel = prevStoreFor, prevHandlerFor, prevGit
		testTransportFor = prevTransport
		st.Close()
	})
	return st
}

// TestSyncPushWiresTransport: `sync push` reads the local pending outbox,
// pushes it through the transport, and acks the local journal (the command
// reports the pushed count; the outbox drains).
func TestSyncPushWiresTransport(t *testing.T) {
	tr := &testPGTransport{}
	st := installSharedStore(t, tr)
	// seed one pending mutation through a real save.
	if _, err := newTestHandlerWithStore(t, st).Save(context.Background(),
		handler.SaveRequest{SessionID: "s", Type: "decision", Title: "seeded for push",
			Content: "synced", Project: "p"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, code := captureRun(t, "sync", "push", "--project", "p")
	if code != exitOK {
		t.Fatalf("sync push exited %d (output: %s)", code, out)
	}
	if !strings.Contains(out, "pushed 1") {
		t.Fatalf("sync push must report the pushed count: %s", out)
	}
	if len(tr.pushed) != 1 {
		t.Fatalf("the transport must receive the pending batch, got %d", len(tr.pushed))
	}
}

// TestSyncPullApplies: `sync pull` fetches the peer's mutations and applies
// them through the store (the pulled observation is live on the local store
// — INV-4 holds for the CLI pull path too).
func TestSyncPullApplies(t *testing.T) {
	st := installSharedStore(t, &testPGTransport{seeded: []store.Mutation{
		{Seq: 1, TargetKey: "cloud", Entity: "observation", EntityKey: "obs-pull-cli", Op: "upsert",
			Payload: `{"sync_id":"obs-pull-cli","session_id":"s","type":"decision","title":"Pulled via CLI","content":"pulled content","project":"p","scope":"project"}`,
			Source:  "remote", Project: "p"},
	}})
	out, code := captureRun(t, "sync", "pull", "--project", "p")
	if code != exitOK {
		t.Fatalf("sync pull exited %d (output: %s)", code, out)
	}
	if id := cliFindIDBySyncID(t, st, "obs-pull-cli"); id == 0 {
		t.Fatalf("pulled observation must be live on the local store (output: %s)", out)
	}
}

// TestSyncExportImportCLI: `sync export` writes the chunk archive to the
// dir, and `sync import <dir>` applies it (the CLI surfaces of the same
// codec the memory package tests exercise).
func TestSyncExportImportCLI(t *testing.T) {
	st := installSharedStore(t, &testPGTransport{})
	if _, err := newTestHandlerWithStore(t, st).Save(context.Background(),
		handler.SaveRequest{SessionID: "s", Type: "decision", Title: "export me",
			Content: "export content", Project: "p"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	dir := t.TempDir()
	if _, code := captureRun(t, "sync", "export", "--project", "p", "--dir", dir); code != exitOK {
		t.Fatalf("sync export must succeed")
	}
	if _, err := os.Stat(dir + "/manifest.json"); err != nil {
		t.Fatalf("export must write manifest.json: %v", err)
	}
	if _, code := captureRun(t, "sync", "import", "--project", "p", "--dir", dir); code != exitOK {
		t.Fatalf("sync import must succeed")
	}
	// the archive's chunk carries the saved content (the round-trip is
	// proven where the import applies it — the memory package's
	// TestExportImportRoundTrip asserts store-state reproduction; this
	// CLI test asserts the archive the CLI wrote actually contains the
	// exported row).
	chunkFiles, err := os.ReadDir(dir + "/chunks")
	if err != nil {
		t.Fatalf("export must write a chunks dir: %v", err)
	}
	found := false
	for _, cf := range chunkFiles {
		f, err := os.Open(dir + "/chunks/" + cf.Name())
		if err != nil {
			t.Fatalf("open chunk: %v", err)
		}
		gz, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			t.Fatalf("gunzip chunk: %v", err)
		}
		data, err := io.ReadAll(gz)
		gz.Close()
		f.Close()
		if err != nil {
			t.Fatalf("read chunk: %v", err)
		}
		if strings.Contains(string(data), "export me") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("the exported chunk must carry the saved row's content (title 'export me' not in any chunk)")
	}
}

// TestSyncPushFailsLoud (TRIANGULATE, error path): a transport failure on
// push exits non-zero and leaves the outbox intact (nothing is acked on a
// failed push — the next cycle retries the same batch).
func TestSyncPushFailsLoud(t *testing.T) {
	tr := &testPGTransport{failPush: true}
	st := installSharedStore(t, tr)
	if _, err := newTestHandlerWithStore(t, st).Save(context.Background(),
		handler.SaveRequest{SessionID: "s", Type: "decision", Title: "fail push",
			Content: "stays pending", Project: "p"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	_, code := captureRun(t, "sync", "push", "--project", "p")
	if code == exitOK {
		t.Fatalf("a failed push must exit non-zero")
	}
	// the outbox stays intact (nothing is acked on a failed push — the
	// next cycle retries the same batch).
	pending, err := st.ListPendingMutations(context.Background(), 0, memory.MaxBatch)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) == 0 {
		t.Fatalf("a failed push must leave the outbox intact (pending: %d)", len(pending))
	}
	if len(tr.pushed) != 0 {
		t.Fatalf("a failed push must not ack any batch (pushed: %d)", len(tr.pushed))
	}
}

// ── fixture helpers ────────────────────────────────────────────────────────

// cliFindIDBySyncID returns the local observation id for a sync id (0 if
// absent) — the probe-seam assertion surface for the pull test.
func cliFindIDBySyncID(t *testing.T, st store.Store, syncID string) int64 {
	t.Helper()
	n, err := st.(interface {
		ProbeInt(query string, args ...any) (int, error)
	}).ProbeInt(`SELECT id FROM observations WHERE sync_id = ? AND deleted_at IS NULL`, syncID)
	if err != nil {
		t.Fatalf("find by sync id: %v", err)
	}
	return int64(n)
}
