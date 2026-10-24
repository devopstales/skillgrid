// Background distill + write approval (change
// 2026-10-24-global-llm-wiki, Change A, TICKET-06).
//
// RED notes (the failures that prove each test exercises new code):
//   - TestDistillRoutesFactsByScope (BDD: "happy path — distill extracts and
//     routes facts by scope", G1+G2): compile error — facts.NewDistiller,
//     facts.DistillResult, facts.PendingFacts do not exist yet.
//   - TestDistillCoalescingLatestWins (BDD: "edge — distill coalescing
//     (latest snapshot wins)", G3): compile error — no coalescing flag;
//     before the flag exists every trigger fires an LLM call.
//   - TestDistillLLMDownFailsOpen (BDD: "failure — distill failure does not
//     break session close", G4): before fail-open exists, an LLM error
//     surfaces to the caller and/or leaves a partial write.
//   - TestWriteApprovalStagesPassiveWrites (BDD: "happy path — passive write
//     is staged and approved", G6): before the pending_facts table exists the
//     INSERT fails (no such table) — the staged row is absent.
//   - TestPendingQueueSurvivesRestart (BDD: "edge — pending queue survives
//     server restart", G5): before durability the row lives in a memory map.
//   - TestExplicitWritesBypassApprovalGate (BDD: "edge — explicit writes
//     bypass the approval gate", G7): before the gate the staged row is
//     indistinguishable from the explicit one (no row at all → the
//     pending-empty assertion passes for the wrong reason, the fact-row
//     assertion carries the real check).
//   - TestWriteApprovalRejectDiscards (BDD: "failure — rejected pending fact
//     is discarded", G8): before reject exists the pending row survives.
package facts

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// fakeLLM is a scriptable LLM stub: it counts calls and returns the queued
// contents (or the error) in order. The counter is the G1/G3 oracle — the
// tests assert on CALL COUNTS, not on prompt contents.
type fakeLLM struct {
	mu    *sync.Mutex
	calls int
	queue []string
	err   error
}

// Complete implements the DistillLLM signature (it is a DistillLLM value
// through Go's func-to-method-set conversion).
func (f *fakeLLM) Complete(_ context.Context, _, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	if len(f.queue) == 0 {
		return "", errors.New("fakeLLM: empty response queue")
	}
	out := f.queue[0]
	f.queue = f.queue[1:]
	return out, nil
}

func (f *fakeLLM) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// asDistillLLM adapts the fakeLLM stub to the DistillLLM func type.
func (f *fakeLLM) asDistillLLM() DistillLLM {
	return f.Complete
}

// distillFixture wires the two-store fixture with a Distiller bound to the
// given LLM and the given write_approval setting. Returns the distiller,
// the raw global DB (row assertions) and the raw project DB (row assertions).
func distillFixture(t *testing.T, llm DistillLLM, writeApproval bool) (*Distiller, *sql.DB, *sql.DB) {
	t.Helper()
	pst, gst, _ := openTwoStores(t)
	seedTestSession(t, pst, "sess-distill")
	globalStore, _, err := Route(ScopeGlobal, gst.DB, pst.DB)
	if err != nil {
		t.Fatalf("Route global: %v", err)
	}
	pstStore := New(pst.DB, "factstest")
	d := NewDistiller(globalStore, pstStore)
	d.LLM = llm
	d.WriteApproval = writeApproval
	return d, gst.DB, pstStore.DB()
}

// factCount counts live facts in the given store DB (deleted_at IS NULL).
func factCount(t *testing.T, db *sql.DB, content string) int {
	t.Helper()
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL AND content = ?`, content).
		Scan(&n)
	if err != nil {
		t.Fatalf("count facts: %v", err)
	}
	return n
}

// pendingCount counts pending_facts rows (all statuses when content is empty,
// the matching content otherwise).
func pendingCount(t *testing.T, db *sql.DB, content string) int {
	t.Helper()
	q := `SELECT COUNT(*) FROM pending_facts`
	var args []any
	if content != "" {
		q += ` WHERE content = ?`
		args = append(args, content)
	}
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	return n
}

// TestDistillRoutesFactsByScope (BDD: "happy path — distill extracts and
// routes facts by scope", G1+G2): the LLM is scripted to return one project
// fact and one global fact. Distill must make exactly ONE LLM call (G1) and
// route each fact to its own store (G2) — no approval gate in effect, so
// both land directly in `facts`.
func TestDistillRoutesFactsByScope(t *testing.T) {
	var mu sync.Mutex
	llm := &fakeLLM{mu: &mu, queue: []string{
		`{"facts": [{"scope": "project", "target": "memory", "content": "project: always run go vet before commit"},
 {"scope": "global", "target": "memory", "content": "global: the user prefers concise replies"}]}`,
	}}
	d, gdb, pdb := distillFixture(t, llm.asDistillLLM(), false)

	res, err := d.Distill(ctx, "sess-distill", "sess-distill", "the conversation snapshot")
	if err != nil {
		t.Fatalf("Distill returned an error (session close must not break): %v", err)
	}
	if got := llm.count(); got != 1 {
		t.Fatalf("LLM calls = %d, want exactly 1 (one pass, not two)", got)
	}
	if res.ProjectWritten != 1 {
		t.Fatalf("ProjectWritten = %d, want 1", res.ProjectWritten)
	}
	if res.GlobalWritten != 1 {
		t.Fatalf("GlobalWritten = %d, want 1", res.GlobalWritten)
	}
	if got := factCount(t, pdb, "project: always run go vet before commit"); got != 1 {
		t.Fatalf("project fact in project store = %d, want 1", got)
	}
	if got := factCount(t, pdb, "global: the user prefers concise replies"); got != 0 {
		t.Fatalf("global fact leaked into project store = %d, want 0", got)
	}
	if got := factCount(t, gdb, "global: the user prefers concise replies"); got != 1 {
		t.Fatalf("global fact in global store = %d, want 1", got)
	}
	if got := factCount(t, gdb, "project: always run go vet before commit"); got != 0 {
		t.Fatalf("project fact leaked into global store = %d, want 0", got)
	}
}

// TestDistillCoalescingLatestWins (BDD: "edge — distill coalescing (latest
// snapshot wins)", G3): two distill triggers fire for the same session. The
// first blocks INSIDE its LLM call; the second must coalesce into it —
// exactly ONE LLM call in total, and the coalesced trigger writes nothing
// of its own. This test asserts the CALL COUNT and the coalesced trigger's
// zero-write result; "latest snapshot wins" itself is the ponytail ceiling's
// documented boundary (see the ponytail note in Distill) and is not
// verified here — the stub returns a fixed response, ignoring the prompt.
func TestDistillCoalescingLatestWins(t *testing.T) {
	var calls atomic.Int32
	d, gdb, _ := distillFixture(t, nil, false)

	started := make(chan struct{})
	release := make(chan struct{})
	d.LLM = func(_ context.Context, _, user string) (string, error) {
		calls.Add(1)
		if calls.Load() == 1 {
			// The first trigger is now mid-flight in its LLM call: signal
			// the test (the second trigger will coalesce) and block until
			// the coalesced trigger's newer snapshot has been recorded.
			close(started)
			<-release
		}
		// The stub returns a fixed response regardless of the prompt it
		// sees — it does not echo the conversation, so the "latest
		// snapshot" property is not observable from the response; only
		// the call count is.
		return `{"facts": [{"scope": "global", "target": "memory", "content": "latest snapshot fact"}]}`, nil
	}
	done := make(chan struct{})
	go func() {
		_, _ = d.Distill(ctx, "sess-distill", "sess-distill", "early conversation")
		close(done)
	}()
	<-started
	// The second trigger fires while the first is mid-flight: it must
	// coalesce (no second LLM call) and write nothing of its own.
	res, err := d.Distill(ctx, "sess-distill", "sess-distill", "later conversation")
	if err != nil {
		t.Fatalf("second Distill returned an error: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("LLM calls after the second trigger = %d, want 1 (the in-flight one; the second trigger coalesced)", got)
	}
	if res.GlobalWritten != 0 {
		t.Fatalf("second trigger's own writes = %d, want 0 (it coalesced into the first trigger)", res.GlobalWritten)
	}
	close(release) // release the blocked call
	<-done

	if got := calls.Load(); got != 1 {
		t.Fatalf("total LLM calls = %d, want exactly 1", got)
	}
	if got := factCount(t, gdb, "latest snapshot fact"); got != 1 {
		t.Fatalf("latest-snapshot fact written = %d, want 1 (latest snapshot wins)", got)
	}
	if got := factCount(t, gdb, "early snapshot fact"); got != 0 {
		t.Fatalf("early-snapshot fact written = %d, want 0 (superseded)", got)
	}
}

// TestDistillLLMDownFailsOpen (BDD: "failure — distill failure does not
// break session close", G4): the LLM errors (unreachable). Distill must
// return a nil error (the session closes normally), write NO facts to
// either store, and record the failure on the result.
func TestDistillLLMDownFailsOpen(t *testing.T) {
	var mu sync.Mutex
	llm := &fakeLLM{mu: &mu, err: errors.New("connection refused")}
	d, gdb, pdb := distillFixture(t, llm.asDistillLLM(), false)

	res, err := d.Distill(ctx, "sess-distill", "sess-distill", "the conversation snapshot")
	if err != nil {
		t.Fatalf("Distill with a down LLM returned error %v, want nil (fail-open: the session closes normally)", err)
	}
	if res.Err == nil {
		t.Fatalf("DistillResult.Err = nil, want the LLM error recorded (it is logged, not returned)")
	}
	if got := llm.count(); got != 1 {
		t.Fatalf("LLM calls = %d, want 1 (the failed attempt)", got)
	}
	var nG, nP int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL`).Scan(&nG); err != nil {
		t.Fatalf("count global facts: %v", err)
	}
	if err := pdb.QueryRow(`SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL`).Scan(&nP); err != nil {
		t.Fatalf("count project facts: %v", err)
	}
	if nG != 0 {
		t.Fatalf("global facts written = %d, want 0 (no facts written on LLM failure)", nG)
	}
	if nP != 0 {
		t.Fatalf("project facts written = %d, want 0 (no facts written on LLM failure)", nP)
	}
}

// TestDistillMalformedResponseWritesNothing (triangulation for G4): the LLM
// answers with unparseable JSON. Same fail-open contract: no error returned,
// nothing written, the error recorded on the result.
func TestDistillMalformedResponseWritesNothing(t *testing.T) {
	var mu sync.Mutex
	llm := &fakeLLM{mu: &mu, queue: []string{`{"facts": "not-an-array"`}}
	d, gdb, pdb := distillFixture(t, llm.asDistillLLM(), false)

	res, err := d.Distill(ctx, "sess-distill", "sess-distill", "the conversation snapshot")
	if err != nil {
		t.Fatalf("Distill with a malformed LLM response returned error %v, want nil", err)
	}
	if res.Err == nil {
		t.Fatalf("DistillResult.Err = nil, want the parse error recorded")
	}
	var nG, nP int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL`).Scan(&nG); err != nil {
		t.Fatalf("count global facts: %v", err)
	}
	if err := pdb.QueryRow(`SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL`).Scan(&nP); err != nil {
		t.Fatalf("count project facts: %v", err)
	}
	if nG != 0 || nP != 0 {
		t.Fatalf("facts written after malformed response: global=%d project=%d, want 0/0", nG, nP)
	}
}

// TestWriteApprovalStagesPassiveWrites (BDD: "happy path — passive write is
// staged and approved", G6): write_approval=true → the distill's global fact
// is staged in pending_facts (NOT in facts). Approve moves it into facts
// and deletes it from pending_facts (two-phase: approve → apply).
func TestWriteApprovalStagesPassiveWrites(t *testing.T) {
	var mu sync.Mutex
	llm := &fakeLLM{mu: &mu, queue: []string{
		`{"facts": [{"scope": "global", "target": "memory", "content": "staged by distill"}]}`,
	}}
	d, gdb, _ := distillFixture(t, llm.asDistillLLM(), true)

	if _, err := d.Distill(ctx, "sess-distill", "sess-distill", "the conversation snapshot"); err != nil {
		t.Fatalf("Distill: %v", err)
	}
	const content = "staged by distill"
	if got := pendingCount(t, gdb, content); got != 1 {
		t.Fatalf("pending_facts row for %q = %d, want 1 (write_approval=true stages the passive write)", content, got)
	}
	if got := factCount(t, gdb, content); got != 0 {
		t.Fatalf("facts row for %q = %d, want 0 (staged, not yet approved)", content, got)
	}

	// Approve by the staged row's id (the MCP fact_pending_approve param is
	// the pending id, not the content).
	rows, err := d.PendingList(ctx)
	if err != nil {
		t.Fatalf("PendingList: %v", err)
	}
	if len(rows) != 1 || !strings.Contains(rows[0].Content, content) {
		t.Fatalf("PendingList = %+v, want the one staged row", rows)
	}
	id, err := d.ApprovePending(ctx, rows[0].ID)
	if err != nil {
		t.Fatalf("ApprovePending: %v", err)
	}
	if id == 0 {
		t.Fatalf("ApprovePending returned id 0, want the new fact id")
	}
	if got := factCount(t, gdb, content); got != 1 {
		t.Fatalf("facts row after approve = %d, want 1 (approved fact lands in facts)", got)
	}
	if got := pendingCount(t, gdb, content); got != 0 {
		t.Fatalf("pending_facts row after approve = %d, want 0 (two-phase: deleted from pending on approve)", got)
	}
}

// TestPendingQueueSurvivesRestart (BDD: "edge — pending queue survives
// server restart", G5): a staged row is still there after the store is
// closed and reopened (a new service instance over the same data dir) —
// pending_facts is a table, not a memory map. The row is staged BEFORE the
// close (the pre-restart distill's stage) and read through the REOPENED
// handle, so the test proves durability across restart, not merely
// visibility in a fresh handle.
func TestPendingQueueSurvivesRestart(t *testing.T) {
	pst, gst, dataDir := openTwoStores(t)
	seedTestSession(t, pst, "sess-distill")

	// Seed a pending row directly (simulating the pre-restart distill's
	// stage) while the original stores are still open.
	if _, err := gst.DB.Exec(`
		INSERT INTO pending_facts (content, target, scope, created_at, status)
		VALUES (?, 'memory', 'global', ?, 'pending')`,
		"survives the restart", "2026-10-24T00:00:00Z"); err != nil {
		t.Fatalf("seed pending row: %v", err)
	}
	gst.Close()
	pst.Close()

	// Restart: reopen both stores over the same data dir.
	pst2, err := store.Open(dataDir, "factstest")
	if err != nil {
		t.Fatalf("reopen project store: %v", err)
	}
	defer pst2.Close()
	gst2, err := store.Open(dataDir, GlobalProjectID)
	if err != nil {
		t.Fatalf("reopen global store: %v", err)
	}
	defer gst2.Close()

	// The REOPENED store — a fresh handle over the same data dir — must
	// see the row that was staged before the restart.
	if got := pendingCount(t, gst2.DB, "survives the restart"); got != 1 {
		t.Fatalf("pending row after reopen = %d, want 1 (durable across restart)", got)
	}
}

// TestExplicitWritesBypassApprovalGate (BDD: "edge — explicit writes bypass
// the approval gate", G7): write_approval=true. An explicit write (AddWith,
// the agent-initiated fact_add path) lands directly in facts and creates NO
// pending_facts row — the gate applies to passive (distill) writes only.
func TestExplicitWritesBypassApprovalGate(t *testing.T) {
	pst, gst, _ := openTwoStores(t)
	seedTestSession(t, pst, "sess-distill")
	data, trail, err := Route(ScopeGlobal, gst.DB, pst.DB)
	if err != nil {
		t.Fatalf("Route global: %v", err)
	}

	const content = "explicit agent write"
	if _, err := data.AddWith(ctx, "sess-distill", content, CategoryMemory, trail); err != nil {
		t.Fatalf("AddWith (explicit write): %v", err)
	}
	if got := factCount(t, gst.DB, content); got != 1 {
		t.Fatalf("explicit fact in global store = %d, want 1 (explicit writes bypass the gate)", got)
	}
	if got := pendingCount(t, gst.DB, content); got != 0 {
		t.Fatalf("pending_facts row for an explicit write = %d, want 0 (bypass)", got)
	}
}

// TestWriteApprovalRejectDiscards (BDD: "failure — rejected pending fact is
// discarded", G8): a staged row rejected via RejectPending disappears from
// pending_facts and never reaches facts.
func TestWriteApprovalRejectDiscards(t *testing.T) {
	var mu sync.Mutex
	llm := &fakeLLM{mu: &mu, queue: []string{
		`{"facts": [{"scope": "global", "target": "user", "content": "rejected by human"}]}`,
	}}
	d, gdb, _ := distillFixture(t, llm.asDistillLLM(), true)

	if _, err := d.Distill(ctx, "sess-distill", "sess-distill", "the conversation snapshot"); err != nil {
		t.Fatalf("Distill: %v", err)
	}
	const content = "rejected by human"
	if got := pendingCount(t, gdb, content); got != 1 {
		t.Fatalf("pending row before reject = %d, want 1", got)
	}
	rows, err := d.PendingList(ctx)
	if err != nil {
		t.Fatalf("PendingList: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("PendingList = %+v, want the one staged row", rows)
	}
	if err := d.RejectPending(ctx, rows[0].ID); err != nil {
		t.Fatalf("RejectPending: %v", err)
	}
	if got := pendingCount(t, gdb, content); got != 0 {
		t.Fatalf("pending row after reject = %d, want 0 (discarded)", got)
	}
	if got := factCount(t, gdb, content); got != 0 {
		t.Fatalf("facts row after reject = %d, want 0 (a rejected fact never lands in facts)", got)
	}
}

// TestPendingFactsListReturnsStagedRows (gate helper for fact_pending_list):
// the list read returns staged rows with their target/scope/created_at, in
// staging order.
func TestPendingFactsListReturnsStagedRows(t *testing.T) {
	var mu sync.Mutex
	llm := &fakeLLM{mu: &mu, queue: []string{
		`{"facts": [{"scope": "global", "target": "memory", "content": "first pending"},
 {"scope": "global", "target": "user", "content": "second pending"}]}`,
	}}
	d, _, _ := distillFixture(t, llm.asDistillLLM(), true)

	if _, err := d.Distill(ctx, "sess-distill", "sess-distill", "the conversation snapshot"); err != nil {
		t.Fatalf("Distill: %v", err)
	}
	rows, err := d.PendingList(ctx)
	if err != nil {
		t.Fatalf("PendingList: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("PendingList rows = %d, want 2", len(rows))
	}
	if !strings.Contains(rows[0].Content, "first pending") || rows[0].Target != CategoryMemory {
		t.Fatalf("row 0 = %+v, want content 'first pending' target memory", rows[0])
	}
	if !strings.Contains(rows[1].Content, "second pending") || rows[1].Target != CategoryUser {
		t.Fatalf("row 1 = %+v, want content 'second pending' target user", rows[1])
	}
}

// TestDistillWithoutLLMIsNoOp (triangulation for G4): no LLM attached at all
// (seam nil — the LLM is not enabled). Same fail-open contract: nil error,
// nothing written, the no-LLM condition recorded on the result.
func TestDistillWithoutLLMIsNoOp(t *testing.T) {
	d, gdb, pdb := distillFixture(t, nil, false)

	res, err := d.Distill(ctx, "sess-distill", "sess-distill", "the conversation snapshot")
	if err != nil {
		t.Fatalf("Distill without an LLM returned error %v, want nil (fail-open)", err)
	}
	if res.Err == nil {
		t.Fatalf("DistillResult.Err = nil, want the no-LLM condition recorded")
	}
	var nG, nP int
	if err := gdb.QueryRow(`SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL`).Scan(&nG); err != nil {
		t.Fatalf("count global facts: %v", err)
	}
	if err := pdb.QueryRow(`SELECT COUNT(*) FROM facts WHERE deleted_at IS NULL`).Scan(&nP); err != nil {
		t.Fatalf("count project facts: %v", err)
	}
	if nG != 0 || nP != 0 {
		t.Fatalf("facts written without an LLM: global=%d project=%d, want 0/0", nG, nP)
	}
}
