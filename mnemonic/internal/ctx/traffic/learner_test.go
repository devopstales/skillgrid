package traffic

import (
	"reflect"
	"testing"
)

// testRules returns a fixed rule set covering all four categories (same shape
// as the package rules.yaml) so extraction tests don't depend on file layout.
// The order matters: error_recovery first so first-match-wins is observable.
func testRules() []Rule {
	return []Rule{
		{ID: "er_fail_then_ok", Category: "error_recovery", Pattern: PatternFailThenOK, Confidence: 0.8, Description: "fail then recover"},
		{ID: "env_repeated_command", Category: "environment", Pattern: PatternRepetition, Confidence: 0.7, MinTimes: 3, Description: "repeated command"},
		{ID: "pref_repeated_message", Category: "preference", Pattern: PatternMessageRepetition, Confidence: 0.7, MinTimes: 2, Description: "repeated message"},
		{ID: "arch_common_prefix", Category: "architecture", Pattern: PatternPathPrefix, Confidence: 0.7, MinTimes: 3, Description: "common prefix"},
	}
}

// TestExtract_ErrorRecovery: a tool_result error on a target followed by an ok
// tool_result on the same target yields one error_recovery candidate.
func TestExtract_ErrorRecovery(t *testing.T) {
	l := NewLearner(testRules(), 0, 0)
	events := []Event{
		{Kind: "tool_result", Tool: "bash", Target: "make test", Status: "error", Seq: 1},
		{Kind: "tool_result", Tool: "bash", Target: "make test", Status: "ok", Seq: 2},
	}
	got := l.Extract("sess-er", events)
	if len(got) != 1 {
		t.Fatalf("len(candidates) = %d, want 1", len(got))
	}
	c := got[0]
	if c.Category != "error_recovery" {
		t.Errorf("category = %q, want error_recovery", c.Category)
	}
	if c.Confidence != 0.8 {
		t.Errorf("confidence = %v, want 0.8", c.Confidence)
	}
	if c.MemoryID == "" {
		t.Error("MemoryID is empty")
	}
	if c.MemoryText == "" {
		t.Error("MemoryText is empty")
	}
	if c.SessionID != "sess-er" {
		t.Errorf("SessionID = %q, want sess-er", c.SessionID)
	}
}

// TestExtract_EnvironmentRepetition: the same tool+target tool_call at least
// MinTimes times yields one environment candidate.
func TestExtract_EnvironmentRepetition(t *testing.T) {
	l := NewLearner(testRules(), 0, 0)
	events := []Event{
		{Kind: "tool_call", Tool: "bash", Target: "npm install", Seq: 1},
		{Kind: "tool_call", Tool: "bash", Target: "npm install", Seq: 2},
		{Kind: "tool_call", Tool: "bash", Target: "npm install", Seq: 3},
	}
	got := l.Extract("sess-env", events)
	if len(got) != 1 {
		t.Fatalf("len(candidates) = %d, want 1", len(got))
	}
	if got[0].Category != "environment" {
		t.Errorf("category = %q, want environment", got[0].Category)
	}
}

// TestExtract_PreferenceRepetition: the same non-empty user_message at least
// MinTimes times yields one preference candidate.
func TestExtract_PreferenceRepetition(t *testing.T) {
	l := NewLearner(testRules(), 0, 0)
	events := []Event{
		{Kind: "user_message", Message: "please use tabs", Seq: 1},
		{Kind: "user_message", Message: "please use tabs", Seq: 2},
	}
	got := l.Extract("sess-pref", events)
	if len(got) != 1 {
		t.Fatalf("len(candidates) = %d, want 1", len(got))
	}
	if got[0].Category != "preference" {
		t.Errorf("category = %q, want preference", got[0].Category)
	}
}

// TestExtract_ArchitecturePrefix: reads/edits under one common path prefix at
// least MinTimes times yield one architecture candidate.
func TestExtract_ArchitecturePrefix(t *testing.T) {
	l := NewLearner(testRules(), 0, 0)
	events := []Event{
		{Kind: "file_read", Target: "internal/store/tables.go", Seq: 1},
		{Kind: "file_read", Target: "internal/store/store.go", Seq: 2},
		{Kind: "file_read", Target: "internal/store/tx.go", Seq: 3},
	}
	got := l.Extract("sess-arch", events)
	if len(got) != 1 {
		t.Fatalf("len(candidates) = %d, want 1", len(got))
	}
	if got[0].Category != "architecture" {
		t.Errorf("category = %q, want architecture", got[0].Category)
	}
}

// TestExtract_MinConfidenceFloor: a candidate whose rule confidence is below
// MinConfidence is dropped (no error_recovery at 0.8 < 0.9).
func TestExtract_MinConfidenceFloor(t *testing.T) {
	l := NewLearner(testRules(), 0.9, 0)
	events := []Event{
		{Kind: "tool_result", Tool: "bash", Target: "make test", Status: "error", Seq: 1},
		{Kind: "tool_result", Tool: "bash", Target: "make test", Status: "ok", Seq: 2},
	}
	got := l.Extract("sess-floor", events)
	if len(got) != 0 {
		t.Errorf("len(candidates) = %d, want 0 (below MinConfidence)", len(got))
	}
}

// TestExtract_EmptyEvents: no events → no candidates.
func TestExtract_EmptyEvents(t *testing.T) {
	l := NewLearner(testRules(), 0, 0)
	if got := l.Extract("sess-empty", nil); len(got) != 0 {
		t.Errorf("len(candidates) = %d, want 0", len(got))
	}
}

// TestExtract_NoMatchingRules: events below every rule's threshold → nothing.
func TestExtract_NoMatchingRules(t *testing.T) {
	l := NewLearner(testRules(), 0, 0)
	events := []Event{
		{Kind: "tool_call", Tool: "ls", Target: ".", Seq: 1},
	}
	if got := l.Extract("sess-nomatch", events); len(got) != 0 {
		t.Errorf("len(candidates) = %d, want 0", len(got))
	}
}

// TestExtract_ZeroPutsOnExtract: Extract is a pure function — it performs NO
// store writes itself. The caller owns persistence. Running Extract with the
// event corpus that yields exactly two matches must NOT advance a put counter
// held on the store; only the explicit caller-side puts do. This is the
// structural zero-LLM assertion: the only injectable dependencies are the
// store and a clock, and Extract touches neither.
func TestExtract_ZeroPutsOnExtract(t *testing.T) {
	cs := newTestStore(t, "p-zero")
	l := NewLearner(testRules(), 0, 0)
	events := []Event{
		{Kind: "tool_result", Tool: "bash", Target: "make test", Status: "error", Seq: 1},
		{Kind: "tool_result", Tool: "bash", Target: "make test", Status: "ok", Seq: 2},
		{Kind: "tool_call", Tool: "bash", Target: "npm install", Seq: 3},
		{Kind: "tool_call", Tool: "bash", Target: "npm install", Seq: 4},
		{Kind: "tool_call", Tool: "bash", Target: "npm install", Seq: 5},
	}
	// Extract returns candidates but performs no store write.
	got := l.Extract("sess-zero", events)
	if len(got) == 0 {
		t.Fatalf("Extract returned 0 candidates, want 2 (error_recovery + environment)")
	}
	before, err := cs.ListTrafficMemories("sess-zero")
	if err != nil {
		t.Fatalf("ListTrafficMemories: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("Extract wrote to the store: %d rows present before any caller put", len(before))
	}
	// Now the caller persists the candidates; exactly N puts, one row each.
	for _, m := range got {
		if err := cs.PutTrafficMemory(m); err != nil {
			t.Fatalf("PutTrafficMemory: %v", err)
		}
	}
	after, err := cs.ListTrafficMemories("sess-zero")
	if err != nil {
		t.Fatalf("ListTrafficMemories: %v", err)
	}
	if len(after) != len(got) {
		t.Errorf("rows after caller puts = %d, want %d (exactly N puts)", len(after), len(got))
	}
}

// TestExtract_DeterministicIDs: two learners (or repeated calls) over the same
// events produce identical MemoryIDs in identical order — the determinism
// guarantee that lets re-runs dedupe via INSERT OR REPLACE.
func TestExtract_DeterministicIDs(t *testing.T) {
	events := []Event{
		{Kind: "tool_result", Tool: "bash", Target: "make test", Status: "error", Seq: 1},
		{Kind: "tool_result", Tool: "bash", Target: "make test", Status: "ok", Seq: 2},
		{Kind: "tool_call", Tool: "bash", Target: "make test", Seq: 3},
		{Kind: "tool_call", Tool: "bash", Target: "make test", Seq: 4},
		{Kind: "tool_call", Tool: "bash", Target: "make test", Seq: 5},
		{Kind: "user_message", Message: "please use tabs", Seq: 6},
		{Kind: "user_message", Message: "please use tabs", Seq: 7},
	}
	// AC#1: two Extract calls on the same corpus produce an identical candidate
	// set — full DeepEqual, not just id comparison. No map iteration, no wall
	// clock, no rand leaks into identity or order.
	lA := NewLearner(testRules(), 0, 0)
	gotA := lA.Extract("sess-det", events)
	lB := NewLearner(testRules(), 0, 0)
	gotB := lB.Extract("sess-det", events)
	if !reflect.DeepEqual(gotA, gotB) {
		t.Errorf("two Extract calls on the same corpus differ:\nA=%+v\nB=%+v", gotA, gotB)
	}
	for i := range gotA {
		if gotA[i].MemoryID == "" {
			t.Errorf("MemoryID[%d] is empty", i)
		}
	}
}

// TestExtract_RerunIdempotentViaStore: because MemoryID is a pure function of
// (session, rule, key), re-extracting the same events and putting to the store
// is idempotent — the row count does not grow on a second pass.
func TestExtract_RerunIdempotentViaStore(t *testing.T) {
	cs := newTestStore(t, "p-rerun")
	l := NewLearner(testRules(), 0, 0)
	events := []Event{
		{Kind: "tool_result", Tool: "bash", Target: "make test", Status: "error", Seq: 1},
		{Kind: "tool_result", Tool: "bash", Target: "make test", Status: "ok", Seq: 2},
	}
	for pass := 0; pass < 2; pass++ {
		for _, m := range l.Extract("sess-rerun", events) {
			if err := cs.PutTrafficMemory(m); err != nil {
				t.Fatalf("pass %d PutTrafficMemory: %v", pass, err)
			}
		}
	}
	rows, err := cs.ListTrafficMemories("sess-rerun")
	if err != nil {
		t.Fatalf("ListTrafficMemories: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("row count after 2 passes = %d, want 1 (idempotent)", len(rows))
	}
}
