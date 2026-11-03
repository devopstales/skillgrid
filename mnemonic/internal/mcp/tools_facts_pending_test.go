// Write approval pending queue on the MCP fact_* tools (change
// 2026-10-24-global-llm-wiki, Change A, TICKET-06).
//
// RED notes (the failures that prove each test exercises new code):
//   - TestFactPendingQueueMCP (BDD: "happy path — passive write is staged
//     and approved" + "failure — rejected pending fact is discarded", MCP
//     layer): compile error — handleFactPendingList / handleFactPendingApprove
//     / handleFactPendingReject do not exist yet.
package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// seedPendingRow inserts one pending_facts row directly (simulating the
// distill's stage) and returns its id.
func seedPendingRow(t *testing.T, db *sql.DB, content, target, scope, createdAt string) int64 {
	t.Helper()
	res, err := db.Exec(`
		INSERT INTO pending_facts (content, target, scope, created_at, status)
		VALUES (?, ?, ?, ?, 'pending')`, content, target, scope, createdAt)
	if err != nil {
		t.Fatalf("seed pending row: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// TestFactPendingQueueMCP (BDD: "happy path — passive write is staged and
// approved" + "failure — rejected pending fact is discarded", MCP layer):
// the pending queue is managed by the three MCP tools. A staged row is
// listed, one is approved (lands in facts, deleted from pending), one is
// rejected (deleted from pending, never in facts).
func TestFactPendingQueueMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-pending-queue"
	seedFactMCPSession(t, st, project, dataDir, sid)

	// The global store (where the global-scope staged rows live).
	gst, err := store.Open(dataDir, "global")
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	defer gst.Close()

	// Stage two rows (simulating the distill's passive writes): one global,
	// one project-scoped.
	now := "2026-10-24T00:00:00Z"
	gid := seedPendingRow(t, gst.DB, "pending global fact A", "memory", "global", now)
	pid := seedPendingRow(t, st.DB, "pending project fact B", "memory", "project", now)

	// fact_pending_list shows both staged rows.
	res, err := handleFactPendingList(context.Background(), callReq("fact_pending_list", map[string]any{
		"project": project,
	}))
	if err != nil {
		t.Fatalf("handleFactPendingList: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_pending_list error: %s", callResultText(t, res))
	}
	var list struct {
		Pending []struct {
			ID      int64  `json:"id"`
			Content string `json:"content"`
			Target  string `json:"target"`
			Scope   string `json:"scope"`
		} `json:"pending_facts"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &list); err != nil {
		t.Fatalf("parse pending list: %v", err)
	}
	if list.Count != 2 {
		t.Fatalf("pending count = %d, want 2", list.Count)
	}

	// Approve the global row: it lands in the global store's facts and is
	// deleted from pending.
	res, err = handleFactPendingApprove(context.Background(), callReq("fact_pending_approve", map[string]any{
		"project":    project,
		"pending_id": float64(gid),
	}))
	if err != nil {
		t.Fatalf("handleFactPendingApprove: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_pending_approve error: %s", callResultText(t, res))
	}
	var approved struct {
		FactID    int64 `json:"fact_id"`
		PendingID int64 `json:"pending_id"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &approved); err != nil {
		t.Fatalf("parse approve result: %v", err)
	}
	if approved.FactID <= 0 {
		t.Fatalf("approved fact_id = %d, want a positive id", approved.FactID)
	}
	var approvedInFacts int
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM facts WHERE id = ?`, approved.FactID).Scan(&approvedInFacts); err != nil {
		t.Fatalf("approved fact lookup: %v", err)
	}
	if approvedInFacts != 1 {
		t.Fatalf("approved fact in global store = %d, want 1", approvedInFacts)
	}
	var approvedInPending int
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM pending_facts WHERE id = ?`, gid).Scan(&approvedInPending); err != nil {
		t.Fatalf("count approved pending: %v", err)
	}
	if approvedInPending != 0 {
		t.Fatalf("approved pending row still present = %d, want 0 (two-phase: deleted on approve)", approvedInPending)
	}

	// Reject the project-scoped row: it is deleted from pending and never
	// reaches facts.
	res, err = handleFactPendingReject(context.Background(), callReq("fact_pending_reject", map[string]any{
		"project":    project,
		"pending_id": float64(pid),
	}))
	if err != nil {
		t.Fatalf("handleFactPendingReject: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_pending_reject error: %s", callResultText(t, res))
	}
	var rejectedInPending int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pending_facts WHERE id = ?`, pid).Scan(&rejectedInPending); err != nil {
		t.Fatalf("count rejected pending: %v", err)
	}
	if rejectedInPending != 0 {
		t.Fatalf("rejected pending row still present = %d, want 0 (discarded)", rejectedInPending)
	}
	var rejectedInFacts int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM facts WHERE content = ?`, "pending project fact B").Scan(&rejectedInFacts); err != nil {
		t.Fatalf("count rejected facts: %v", err)
	}
	if rejectedInFacts != 0 {
		t.Fatalf("rejected fact in project store = %d, want 0 (a rejected fact never lands in facts)", rejectedInFacts)
	}

	// The queue is now empty.
	res, err = handleFactPendingList(context.Background(), callReq("fact_pending_list", map[string]any{
		"project": project,
	}))
	if err != nil {
		t.Fatalf("handleFactPendingList (after): %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_pending_list (after) error: %s", callResultText(t, res))
	}
	var after struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &after); err != nil {
		t.Fatalf("parse pending list (after): %v", err)
	}
	if after.Count != 0 {
		t.Fatalf("pending count after approve+reject = %d, want 0", after.Count)
	}
}

// TestFactPendingApproveUnknownIDMCP (BDD: "failure — rejected pending fact
// is discarded", MCP edge): approving a pending id that is already gone is a
// clean error, not a panic or a silent no-op.
func TestFactPendingApproveUnknownIDMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-pending-unknown"
	seedFactMCPSession(t, st, project, dataDir, sid)
	gst, err := store.Open(dataDir, "global")
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	defer gst.Close()

	res, err := handleFactPendingApprove(context.Background(), callReq("fact_pending_approve", map[string]any{
		"project":    project,
		"pending_id": float64(99999),
	}))
	if err != nil {
		t.Fatalf("handleFactPendingApprove: %v", err)
	}
	if !res.IsError {
		t.Fatalf("approving an unknown pending id should be a clean error, got success: %s", callResultText(t, res))
	}
}
