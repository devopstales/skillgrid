package memory

// relations_test.go — [conflict-candidates-surfaced], [relevance-gate-terms],
// [no-candidate-when-gate-fails] (acceptance.feature).

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// [relevance-gate-terms] the pinned title pairs (spec 01-memory/003 §3 step
// 2): terms are case-insensitive letter/digit words with +/# preserved
// inside identifiers; connectives + generic change verbs ignored; accept iff
// shared nonempty topic_key OR ≥2 distinct significant shared title terms OR
// both titles reduce to the same single significant term.
func TestRelevanceGateTerms(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"auth-model redesign", "auth-model v2", true},   // shared "auth"+"model" (2)
		{"C++ linker flags", "C# linker flags", true},    // C++ ≠ C#; "linker"+"flags" (2)
		{"updated the parser", "fixed the parser", true}, // both reduce to "parser"
		{"database pool", "database pool sizing", true},  // 2 shared
		{"auth cache ttl", "auth pool sizing", false},    // 1 shared ("auth") — the ≥2 boundary
		{"database pool", "auth cache", false},           // 0 shared
		{"", "auth", false},                              // empty side
	}
	for _, c := range cases {
		if got := candidateGate(titleTopic{Title: c.a}, titleTopic{Title: c.b}); got != c.want {
			t.Errorf("gate(%q,%q)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

// [relevance-gate-terms] tokenizer behavior in isolation: +/# stay attached
// to the preceding run (C++ and C# are distinct single terms),
// case-insensitive, and connectives + generic change verbs drop out.
func TestGateSignificantTerms(t *testing.T) {
	if terms := significantTerms("C++"); len(terms) != 1 || terms[0] != "c++" {
		t.Errorf("C++ → %v, want [c++]", terms)
	}
	if terms := significantTerms("C# linker"); len(terms) != 2 || terms[0] != "c#" || terms[1] != "linker" {
		t.Errorf("C# linker → %v, want [c# linker]", terms)
	}
	if terms := significantTerms("UPDATED The parser and the"); len(terms) != 1 || terms[0] != "parser" {
		t.Errorf("connective/verb filter → %v, want [parser]", terms)
	}
	if terms := significantTerms(""); len(terms) != 0 {
		t.Errorf("empty → %v, want []", terms)
	}
}

// [conflict-candidates-surfaced] a save sharing a topic key with a live
// observation returns it as a pending candidate (judgment required) and
// materializes a pending memory_relations row source=new, target=seed.
func TestConflictCandidatesMaterializedAsPending(t *testing.T) {
	svc := newTestService(t)
	seed := seedObservation(t, svc.st, "p", "decision", "auth-model redesign", "use bearer tokens", "auth/model")
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "auth-model v2",
		Content: "switch to mTLS", Project: "p", Scope: "project", TopicKey: "auth/model"})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Candidates) != 1 || !r.JudgmentRequired {
		t.Fatalf("expected 1 pending candidate: %+v", r.Candidates)
	}
	if r.Candidates[0].Relation != "pending" || r.Candidates[0].JudgmentStatus != "pending" ||
		r.Candidates[0].SourceID != r.SyncID || r.Candidates[0].TargetID != seed.SyncID {
		t.Fatalf("candidate shape wrong: %+v", r.Candidates[0])
	}
	// the pending row is queryable (judged later by Task 10's mem_judge).
	if n := pendingSourceCount(t, svc.st, r.SyncID); n != 1 {
		t.Fatalf("pending relation not materialized: %d", n)
	}
}

// [no-candidate-when-gate-fails] a save whose title shares exactly ONE
// significant term with a live observation (and no shared topic key)
// surfaces no candidate. The seed shares the term "auth" with the save
// ("auth pool"), so the FTS pre-filter returns the seed row and the gate
// actually runs — this pins "the gate rejects one-term overlap", not "the
// net is empty" (a broken gate that accepted on shared >= 1 would surface
// the seed and fail this test).
func TestNoCandidateWhenGateFails(t *testing.T) {
	svc := newTestService(t)
	seedObservation(t, svc.st, "p", "decision", "auth cache sizing", "tune max conns", "")
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "auth pool", Content: "ttl", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	// the pre-filter reaches the gate: the seed shares the term "auth" with
	// the save ("auth pool"), so the seed row is in the FTS pre-filter set.
	// A gate that accepted on shared >= 1 would surface the seed and fail
	// both assertions below.
	if len(r.Candidates) != 0 {
		t.Fatalf("gate must reject one-term overlap: %+v", r.Candidates)
	}
	if pendingSourceCount(t, svc.st, r.SyncID) != 0 {
		t.Fatalf("one-term overlap must not materialize a pending relation")
	}
}

// [relevance-gate-terms] (TRIANGULATE, different inputs: a disjoint title
// pair carried solely by the shared topic key) the gate's topic_key branch
// accepts even when the titles share zero terms. The write-path pre-filter
// is FTS over the title, so a fully disjoint title pair is out of the FTS
// net (a spec-accepted heuristic miss, §3 note); this pins the gate's
// topic_key clause directly.
func TestCandidateGateSharedTopicKeyDisjointTitles(t *testing.T) {
	if !candidateGate(
		titleTopic{Title: "zephyr bridge", TopicKey: "infra/zephyr"},
		titleTopic{Title: "quorum lattice", TopicKey: "infra/zephyr"},
	) {
		t.Fatalf("shared nonempty topic_key must pass the gate even with disjoint titles")
	}
	if candidateGate(
		titleTopic{Title: "zephyr bridge"},
		titleTopic{Title: "quorum lattice", TopicKey: "infra/zephyr"},
	) {
		t.Fatalf("an empty topic_key on one side must not pass via the topic_key clause")
	}
}

// (TRIANGULATE, different inputs: the pre-filter's scope boundary) a
// same-title live observation in a DIFFERENT scope is not a candidate —
// the pre-filter is same project+scope only.
func TestCandidateScopeBoundary(t *testing.T) {
	svc := newTestService(t)
	seed := seedObservationInScope(t, svc.st, "p", "personal", "auth-model redesign", "use bearer tokens", "auth/model")
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "auth-model v2",
		Content: "switch to mTLS", Project: "p", Scope: "project", TopicKey: "auth/model"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range r.Candidates {
		if c.TargetID == seed.SyncID {
			t.Fatalf("different-scope observation must not be a candidate: %+v", c)
		}
	}
}

// (TRIANGULATE, different inputs: soft-deleted rows are not live — a
// deleted same-topic observation must not resurface as a candidate).
func TestCandidateExcludesSoftDeleted(t *testing.T) {
	svc := newTestService(t)
	seed := seedObservation(t, svc.st, "p", "decision", "auth-model redesign", "use bearer tokens", "auth/model")
	if err := svc.st.DeleteObservationForProject(ctx, seed.ID, "p", false); err != nil {
		t.Fatal(err)
	}
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "auth-model v2",
		Content: "switch to mTLS", Project: "p", Scope: "project", TopicKey: "auth/model"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range r.Candidates {
		if c.TargetID == seed.SyncID {
			t.Fatalf("soft-deleted observation must not be a candidate: %+v", c)
		}
	}
}

// (TRIANGULATE, different inputs: the per-source bound) a save against 5
// gate-passing live observations materializes at most candidateBound (3).
func TestCandidatePerSourceBound(t *testing.T) {
	svc := newTestService(t)
	for i, title := range []string{
		"auth-model redesign one", "auth-model redesign two", "auth-model redesign three",
		"auth-model redesign four", "auth-model redesign five",
	} {
		seedObservation(t, svc.st, "p", "decision", title, "content "+string(rune('a'+i)), "")
	}
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "auth-model redesign",
		Content: "the new model", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Candidates) != candidateBound {
		t.Fatalf("per-source bound: got %d candidates, want %d: %+v", len(r.Candidates), candidateBound, r.Candidates)
	}
}

// ── test helpers (this file only) ──────────────────────────────────────────

// seedObservationInScope seeds a live observation with an explicit scope.
func seedObservationInScope(t *testing.T, st store.Store, project, scope, title, content, topicKey string) store.Observation {
	t.Helper()
	res, err := st.AddObservation(ctx, store.AddObservationParams{
		SessionID: testSession, Type: "decision", Title: title, Content: content,
		Project: project, Scope: scope, TopicKey: topicKey,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	o, err := st.GetObservation(ctx, res.ID, false)
	if err != nil {
		t.Fatalf("seed get: %v", err)
	}
	return o
}

// pendingSourceCount counts pending memory_relations rows for a source sync id.
func pendingSourceCount(t *testing.T, st store.Store, sourceSyncID string) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(
		`SELECT COUNT(*) FROM memory_relations WHERE source_id = ? AND judgment_status = 'pending'`, sourceSyncID)
	if err != nil {
		t.Fatalf("pending source count: %v", err)
	}
	return n
}
