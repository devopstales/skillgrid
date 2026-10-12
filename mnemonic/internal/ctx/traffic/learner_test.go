package traffic

import (
	"reflect"
	"strings"
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

// TestExtract_PathPrefix_FileStringPrefixIsNotDirectory: two distinct file
// targets where one is a STRING prefix of the other ("src.go", "src.go.bak")
// must yield ZERO candidates — "src.go" is a filename, not a directory
// boundary (C1: pathPrefix false-positive). MinTimes=2 is satisfied by the two
// distinct targets, so only the dir-boundary guard can reject it.
func TestExtract_PathPrefix_FileStringPrefixIsNotDirectory(t *testing.T) {
	l := NewLearner(testRules(), 0, 0)
	// testRules has arch_common_prefix with MinTimes=3; use a dedicated rule
	// set with MinTimes=2 so two distinct targets are enough to match.
	rules := []Rule{
		{ID: "arch_min2", Category: "architecture", Pattern: PatternPathPrefix, Confidence: 0.7, MinTimes: 2, Description: "common prefix (min 2)"},
	}
	l = NewLearner(rules, 0, 0)
	events := []Event{
		{Kind: "file_read", Target: "src.go", Seq: 1},
		{Kind: "file_read", Target: "src.go.bak", Seq: 2},
	}
	got := l.Extract("sess-c1-neg", events)
	if len(got) != 0 {
		t.Fatalf("len(candidates) = %d, want 0 (full-filename prefix is not a directory)", len(got))
	}
}

// TestExtract_PathPrefix_DirectoryPrefixYieldsCandidate: file targets under a
// shared directory (C1 positive) yield exactly one candidate keyed by that
// directory prefix.
func TestExtract_PathPrefix_DirectoryPrefixYieldsCandidate(t *testing.T) {
	rules := []Rule{
		{ID: "arch_min2", Category: "architecture", Pattern: PatternPathPrefix, Confidence: 0.7, MinTimes: 2, Description: "common prefix (min 2)"},
	}
	l := NewLearner(rules, 0, 0)
	events := []Event{
		{Kind: "file_read", Target: "src/a.go", Seq: 1},
		{Kind: "file_read", Target: "src/b.go", Seq: 2},
		{Kind: "file_read", Target: "src/c.go", Seq: 3},
	}
	got := l.Extract("sess-c1-pos", events)
	if len(got) != 1 {
		t.Fatalf("len(candidates) = %d, want 1 (shared directory prefix)", len(got))
	}
	if got[0].Category != "architecture" {
		t.Errorf("category = %q, want architecture", got[0].Category)
	}
	// The evidence key must be the directory prefix "src/" (visible via the
	// evidence JSON's targets — all under src/).
	if !strings.Contains(got[0].MemoryText, "src/") {
		t.Errorf("memory text = %q, want it to reference prefix src/", got[0].MemoryText)
	}
}

// TestExtract_CrossRuleFirstMatchWins: two identical rules over the same event
// corpus — the first rule's emitted candidate consumes the events, so the
// second (later) rule emits ZERO. Total candidates == the first rule's count,
// not double. This asserts the cross-rule event-consumption contract (C2)
// directly, independent of the shipped rules.yaml patterns.
func TestExtract_CrossRuleFirstMatchWins(t *testing.T) {
	// Two messageRepetition rules, same pattern, same MinTimes, both eligible
	// for the same two repeated user_messages.
	rules := []Rule{
		{ID: "pref_first", Category: "preference", Pattern: PatternMessageRepetition, Confidence: 0.7, MinTimes: 2, Description: "first rule"},
		{ID: "pref_second", Category: "preference", Pattern: PatternMessageRepetition, Confidence: 0.7, MinTimes: 2, Description: "second rule"},
	}
	l := NewLearner(rules, 0, 0)
	events := []Event{
		{Kind: "user_message", Message: "please use tabs", Seq: 1},
		{Kind: "user_message", Message: "please use tabs", Seq: 2},
	}
	got := l.Extract("sess-c2", events)
	if len(got) != 1 {
		t.Fatalf("len(candidates) = %d, want 1 (first rule consumes the events; second must emit 0)", len(got))
	}
	// Exactly one candidate, and it must be the FIRST rule's.
	// (MemoryID is derived from rule ID; just assert the count and that the
	// evidence came from a single rule — no duplicate memories.)
}

// TestExtract_CrossRuleBelowFloorDoesNotConsume: a candidate below MinConfidence
// is skipped WITHOUT consuming its events, so a later rule that matches the
// same events still emits (C2 scoping decision).
func TestExtract_CrossRuleBelowFloorDoesNotConsume(t *testing.T) {
	rules := []Rule{
		{ID: "pref_low", Category: "preference", Pattern: PatternMessageRepetition, Confidence: 0.4, MinTimes: 2, Description: "low confidence rule"},
		{ID: "pref_high", Category: "preference", Pattern: PatternMessageRepetition, Confidence: 0.8, MinTimes: 2, Description: "high confidence rule"},
	}
	// MinConfidence floor = 0.6: the first rule (0.4) is below the floor, so it
	// is dropped and must NOT consume the events; the second rule (0.8) then
	// still matches and emits.
	l := NewLearner(rules, 0.6, 0)
	events := []Event{
		{Kind: "user_message", Message: "please use tabs", Seq: 1},
		{Kind: "user_message", Message: "please use tabs", Seq: 2},
	}
	got := l.Extract("sess-c2-floor", events)
	if len(got) != 1 {
		t.Fatalf("len(candidates) = %d, want 1 (below-floor rule must not consume events)", len(got))
	}
	if got[0].Confidence != 0.8 {
		t.Errorf("confidence = %v, want 0.8 (the high-confidence later rule)", got[0].Confidence)
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
