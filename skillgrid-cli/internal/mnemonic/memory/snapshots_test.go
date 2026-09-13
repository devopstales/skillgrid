package memory

import (
	"context"
	"database/sql"
	"strconv"
	"testing"
	"time"
)

// seedSnapshotObs saves n observations for projectID and returns their ids in
// save order. Each observation has a distinct title/content so a restore can
// be verified row-by-row.
func seedSnapshotObs(t *testing.T, svc *Service, sid string, n int) []int64 {
	t.Helper()
	ctx := context.Background()
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		in := SaveInput{
			Title:     "snapshot baseline " + string(rune('A'+i)),
			Type:      "decision",
			Content:   "baseline content " + string(rune('a'+i)) + " for snapshot testing",
			SessionID: sid,
			Scope:     "project",
		}
		id, err := svc.Save(ctx, in)
		if err != nil {
			t.Fatalf("seed save %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	return ids
}

// TestSnapshotCreatePointInTime covers 20.1: Snapshot(projectID) captures a
// point-in-time view of the project's observations; RestoreSnapshot(snapshotID)
// rolls the store back to exactly that state (rows added after the capture are
// removed, modified rows return to their captured content).
func TestSnapshotCreatePointInTime(t *testing.T) {
	_, svc := newTestStore(t, "snapshotpoint")
	ctx := context.Background()
	sid := newSession(t, svc)

	ids := seedSnapshotObs(t, svc, sid, 5)

	// Capture the point-in-time state.
	snapID, err := svc.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snapID <= 0 {
		t.Fatalf("snapshot id=%d want >0", snapID)
	}

	// Record the captured state hash (integrity: it must match the BLOB).
	var stateHash string
	var data []byte
	err = svc.store.DB.QueryRowContext(ctx,
		`SELECT state_hash, data FROM snapshots WHERE id = ?`, snapID,
	).Scan(&stateHash, &data)
	if err != nil {
		t.Fatalf("read snapshot row: %v", err)
	}
	if stateHash == "" || len(data) == 0 {
		t.Fatalf("snapshot row incomplete: hash=%q dataLen=%d", stateHash, len(data))
	}
	if got := sha256Hex(data); got != stateHash {
		t.Fatalf("state_hash=%q does not match SHA-256 of data BLOB %q", stateHash, got)
	}

	// Modify 2 observations and add 1 more.
	if err := svc.Update(ctx, ids[0], UpdateInput{Content: "modified content zero"}); err != nil {
		t.Fatalf("update 0: %v", err)
	}
	if err := svc.Update(ctx, ids[1], UpdateInput{Content: "modified content one"}); err != nil {
		t.Fatalf("update 1: %v", err)
	}
	if _, err := svc.Save(ctx, SaveInput{
		Title:     "added after snapshot",
		Type:      "decision",
		Content:   "this observation did not exist at capture time",
		SessionID: sid,
		Scope:     "project",
	}); err != nil {
		t.Fatalf("add after snapshot: %v", err)
	}

	// Before restore: 6 observations, 2 of them modified.
	pre, _ := svc.Recent(ctx, 100)
	if len(pre) != 6 {
		t.Fatalf("pre-restore count=%d want 6", len(pre))
	}
	modified := make(map[int64]string)
	for _, o := range pre {
		if o.Content == "modified content zero" || o.Content == "modified content one" {
			modified[o.ID] = o.Content
		}
	}
	if len(modified) != 2 {
		t.Fatalf("pre-restore modified count=%d want 2", len(modified))
	}

	// Restore to the captured state.
	if err := svc.RestoreSnapshot(ctx, snapID); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}

	// After restore: back to 5 observations with their ORIGINAL content.
	post, err := svc.Recent(ctx, 100)
	if err != nil {
		t.Fatalf("recent after restore: %v", err)
	}
	if len(post) != 5 {
		t.Fatalf("post-restore count=%d want 5", len(post))
	}
	for i, id := range ids {
		found := false
		for _, o := range post {
			if o.ID != id {
				continue
			}
			found = true
			want := "baseline content " + string(rune('a'+i)) + " for snapshot testing"
			if o.Content != want {
				t.Fatalf("restored obs %d content=%q want %q", id, o.Content, want)
			}
		}
		if !found {
			t.Fatalf("restored state missing observation %d", id)
		}
	}
	for _, o := range post {
		if o.Content == "modified content zero" || o.Content == "modified content one" ||
			o.Content == "this observation did not exist at capture time" {
			t.Fatalf("restored state still contains post-snapshot content: %q", o.Content)
		}
	}
}

// TestSnapshotRestoreAtomic proves the restore is one transaction: a failure
// partway through the restore leaves the store in its PRE-restore state (never
// half-restored). The seam restoreRowFn lets the test fail the 2nd row's write.
func TestSnapshotRestoreAtomic(t *testing.T) {
	_, svc := newTestStore(t, "snapshotatomic")
	ctx := context.Background()
	sid := newSession(t, svc)
	ids := seedSnapshotObs(t, svc, sid, 3)

	snapID, err := svc.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if err := svc.Update(ctx, ids[0], UpdateInput{Content: "changed before restore"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	before, _ := svc.Recent(ctx, 100)
	if len(before) != 3 {
		t.Fatalf("pre-restore count=%d want 3", len(before))
	}

	// Fail the 2nd row's write: the whole restore must roll back atomically.
	svc.restoreRowFn = func(ctx context.Context, tx *sql.Tx, row *SnapshotRow) error {
		if row.ID == ids[1] {
			return sql.ErrConnDone
		}
		return nil
	}
	if err := svc.RestoreSnapshot(ctx, snapID); err == nil {
		t.Fatalf("expected restore to fail when a row write fails")
	}

	// After a failed restore, the store is exactly as it was before: 3 rows,
	// the original modification intact (no partial restore).
	after, _ := svc.Recent(ctx, 100)
	if len(after) != 3 {
		t.Fatalf("post-failed-restore count=%d want 3 (rollback must be atomic)", len(after))
	}
	for _, o := range after {
		if o.ID == ids[0] && o.Content != "changed before restore" {
			t.Fatalf("partial restore: obs %d content=%q want pre-restore %q", ids[0], o.Content, "changed before restore")
		}
	}
}

// TestRowLevelLockingConcurrentWrites covers 20.2: two concurrent writers
// update the SAME observation. SQLite's write lock serializes them — one
// completes, the other blocks until the first commits — and the final state is
// exactly one of the two writes (never a mix). The lock is released after the
// transaction, so a subsequent write succeeds.
func TestRowLevelLockingConcurrentWrites(t *testing.T) {
	_, svc := newTestStore(t, "snapshotrowlock")
	ctx := context.Background()
	sid := newSession(t, svc)
	id, err := svc.Save(ctx, SaveInput{
		Title:     "concurrent target",
		Type:      "decision",
		Content:   "initial content",
		SessionID: sid,
		Scope:     "project",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	// Open a second connection to the same file: a true second writer with its
	// own SQLite handle. Two handles on the same file contend for the WAL
	// write lock — this is the concurrency the row-level locking serializes.
	secondDB := openRawSQLDB(t, svc.store.Path())
	defer secondDB.Close()

	writeA := "writer A content"
	writeB := "writer B content"
	start := make(chan struct{})
	type outcome struct {
		err error
	}
	resA := make(chan outcome, 1)
	resB := make(chan outcome, 1)

	go func() {
		_, err := svc.UpdateRow(ctx, svc.store.DB, id, writeA)
		resA <- outcome{err: err}
	}()
	go func() {
		_, err := svc.UpdateRow(ctx, secondDB, id, writeB)
		resB <- outcome{err: err}
	}()

	close(start)
	a := <-resA
	b := <-resB
	if a.err != nil {
		t.Fatalf("writer A: %v", a.err)
	}
	if b.err != nil {
		t.Fatalf("writer B: %v", b.err)
	}

	// No corruption: the final content is exactly one of the two writes.
	obs, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get after writes: %v", err)
	}
	if obs.Content != writeA && obs.Content != writeB {
		t.Fatalf("final content %q is a mix of %q and %q (data corruption)", obs.Content, writeA, writeB)
	}

	// The lock is released after the transaction: a follow-up write succeeds
	// promptly (no 5-second busy-timeout stall).
	deadline := time.Now().Add(3 * time.Second)
	if _, err := svc.UpdateRow(ctx, svc.store.DB, id, "writer C after lock release"); err != nil {
		t.Fatalf("write after lock release: %v", err)
	}
	if time.Now().After(deadline) {
		t.Fatalf("write after lock release took longer than the 3s deadline (lock not released)")
	}
}

// TestSnapshotAutoPrune covers 20.3: PruneSnapshots(projectID, keep) deletes
// all but the keep most recent snapshots; auto-prune runs after each new
// snapshot; retention is configurable.
func TestSnapshotAutoPrune(t *testing.T) {
	_, svc := newTestStore(t, "snapshotprune")
	ctx := context.Background()
	sid := newSession(t, svc)
	ids := seedSnapshotObs(t, svc, sid, 2)

	// Explicit retention of 5 for this service (configurable).
	svc.SetSnapshotRetention(5)

	var snapIDs []int64
	for i := 0; i < 20; i++ {
		id, err := svc.Save(ctx, SaveInput{
			Title:     "prune filler " + string(rune('0'+i%10)),
			Type:      "decision",
			Content:   "filler content " + string(rune('x'+i)),
			SessionID: sid,
			Scope:     "project",
			TopicKey:  "prune-key-" + string(rune('0'+i)),
		})
		if err != nil {
			t.Fatalf("save filler %d: %v", i, err)
		}
		_ = id
		snapID, err := svc.Snapshot(ctx)
		if err != nil {
			t.Fatalf("snapshot %d: %v", i, err)
		}
		snapIDs = append(snapIDs, snapID)
		// After each snapshot, auto-prune keeps only the last 5.
		count := snapshotCount(t, svc)
		if count > 5 {
			t.Fatalf("after snapshot %d, count=%d want <=5 (auto-prune keep=5)", i, count)
		}
	}

	// Exactly the 5 most recent survive; the 15 oldest are gone.
	count := snapshotCount(t, svc)
	if count != 5 {
		t.Fatalf("final count=%d want 5", count)
	}
	remaining := snapshotIDSet(t, svc)
	for i := 0; i < 15; i++ {
		if remaining[snapIDs[i]] {
			t.Fatalf("oldest snapshot %d (id=%d) should have been pruned", i, snapIDs[i])
		}
	}
	for i := 15; i < 20; i++ {
		if !remaining[snapIDs[i]] {
			t.Fatalf("recent snapshot %d (id=%d) should have survived", i, snapIDs[i])
		}
	}

	// Explicit PruneSnapshots honors a different keep value.
	if err := svc.PruneSnapshots(ctx, 2); err != nil {
		t.Fatalf("PruneSnapshots: %v", err)
	}
	if count := snapshotCount(t, svc); count != 2 {
		t.Fatalf("after PruneSnapshots(2) count=%d want 2", count)
	}
	_ = ids
}

// TestListSnapshots covers the ListSnapshots() read path the CLI uses: it
// returns the project's snapshots newest-first, each with its state hash and
// timestamp.
func TestListSnapshots(t *testing.T) {
	_, svc := newTestStore(t, "snapshotlist")
	ctx := context.Background()
	sid := newSession(t, svc)
	seedSnapshotObs(t, svc, sid, 2)

	// No snapshots yet → empty list, no error.
	list, err := svc.ListSnapshots(ctx)
	if err != nil {
		t.Fatalf("ListSnapshots empty: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("ListSnapshots empty count=%d want 0", len(list))
	}

	first, err := svc.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot 1: %v", err)
	}
	second, err := svc.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot 2: %v", err)
	}

	list, err = svc.ListSnapshots(ctx)
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListSnapshots count=%d want 2", len(list))
	}
	// Newest first.
	if list[0].ID != second || list[1].ID != first {
		t.Fatalf("ListSnapshots order: got [%d %d] want [%d %d]", list[0].ID, list[1].ID, second, first)
	}
	for i, s := range list {
		if s.StateHash == "" {
			t.Fatalf("ListSnapshots[%d].StateHash empty", i)
		}
		if s.CreatedAt == "" {
			t.Fatalf("ListSnapshots[%d].CreatedAt empty", i)
		}
		if _, err := time.Parse(time.RFC3339, s.CreatedAt); err != nil {
			t.Fatalf("ListSnapshots[%d].CreatedAt %q not RFC3339: %v", i, s.CreatedAt, err)
		}
	}
}

// openRawSQLDB opens a second, independent *sql.DB on the same SQLite file as
// path (the store's own handle is MaxOpenConns(1), so a true second writer
// needs its own handle). It applies the row-lock pragmas: busy_timeout (the
// 5-second lock-acquisition window) and _txlock=immediate (BEGIN IMMEDIATE).
// The two handles on the same file contend for the WAL write lock — this is
// the concurrency the row-level locking serializes (014 step 20.2).
func openRawSQLDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path+rowLockDsnSuffix)
	if err != nil {
		t.Fatalf("open raw sql db: %v", err)
	}
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=" + strconv.Itoa(rowLockTimeout),
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			t.Fatalf("apply %s: %v", pragma, err)
		}
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// snapshotCount returns the number of snapshots stored for the service's
// project.
func snapshotCount(t *testing.T, svc *Service) int {
	t.Helper()
	var n int
	if err := svc.store.DB.QueryRow(
		`SELECT COUNT(*) FROM snapshots WHERE project_id = ?`, svc.ProjectID(),
	).Scan(&n); err != nil {
		t.Fatalf("count snapshots: %v", err)
	}
	return n
}

// snapshotIDSet returns the set of snapshot ids for the service's project.
func snapshotIDSet(t *testing.T, svc *Service) map[int64]bool {
	t.Helper()
	rows, err := svc.store.DB.Query(
		`SELECT id FROM snapshots WHERE project_id = ?`, svc.ProjectID(),
	)
	if err != nil {
		t.Fatalf("list snapshot ids: %v", err)
	}
	defer rows.Close()
	set := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan snapshot id: %v", err)
		}
		set[id] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate snapshot ids: %v", err)
	}
	return set
}
