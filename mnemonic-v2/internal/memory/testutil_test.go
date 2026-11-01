package memory

// Test helpers shared by every memory-package test (created in Task 5; later
// tasks add to this file only when a new helper is needed — never redefine an
// existing one).

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
)

// testSession is the fixed session id every helper seeds (keeps assertions
// deterministic across tests).
const testSession = "s"

// openTestStoreIn opens a real SQLite store for project in dir.
func openTestStoreIn(t *testing.T, dir, project string) store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), dir, project, store.SQLite, "")
	if err != nil {
		t.Fatalf("open store (%s): %v", project, err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// openTestStore opens a real SQLite store for the default test project.
func openTestStore(t *testing.T) store.Store {
	return openTestStoreIn(t, t.TempDir(), "p")
}

// newTestServiceWith builds a Service over a real SQLite store with cfg
// layered over config.Default(). cfg.DataDir is always t.TempDir() and
// cfg.Project.Detect is forced off so the pipeline never shells out to git.
func newTestServiceWith(t *testing.T, cfg config.MemoryCfg) *Service {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	c.Memory = cfg
	st := openTestStoreIn(t, c.DataDir, "p")
	return NewService(st, c, NewEventBus(), nil, nil)
}

// newTestService is newTestServiceWith on the default memory config.
func newTestService(t *testing.T) *Service {
	return newTestServiceWith(t, config.Default().Memory)
}

// stubLLM is the LLM seam test fake. Verdicts are set per test; the Call
// adapter forwards the Classify decision to degrade.Run's seam path.
type stubLLM struct {
	classify func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error)
	// Task 8: judge/compare/extract land here.
}

// setClassify sets the Classify verdict (the degrade.Seam Call adapter
// invokes it with the pre-filtered candidate contents).
func (s *stubLLM) setClassify(fn func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error)) {
	s.classify = fn
}

func (s *stubLLM) Classify(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
	return s.classify(ctx, newContent, candidates)
}
func (s *stubLLM) Judge(ctx context.Context, source, target llm.Observation, rel llm.Relation) (llm.Judgment, error) {
	return llm.Judgment{}, nil // Task 9
}
func (s *stubLLM) Compare(ctx context.Context, a, b llm.Observation) (llm.Judgment, error) {
	return llm.Judgment{}, nil // Task 9
}
func (s *stubLLM) Extract(ctx context.Context, text string) ([]llm.CandidateObservation, error) {
	return nil, nil // Task 11
}

// Call is the degrade.Seam adapter: the (a, b) arguments are
// (newContent string, candidates []string) from the dedup call site.
func (s *stubLLM) Call(ctx context.Context, a, b any) (any, error) {
	d, err := s.Classify(ctx, a.(string), b.([]string))
	if err != nil {
		return llm.DedupDecision{}, err
	}
	return d, nil
}

func (s *Service) stub() *stubLLM { return s.llm.(*stubLLM) }

// newTestServiceWithLLM builds a Service whose LLM seam is the stub.
func newTestServiceWithLLM(t *testing.T, stub *stubLLM) *Service {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	st := openTestStoreIn(t, c.DataDir, "p")
	return NewService(st, c, NewEventBus(), stub, nil)
}

// newTestServiceWithStore builds a Service over a caller-provided store.
func newTestServiceWithStore(t *testing.T, st store.Store) *Service {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	return NewService(st, c, NewEventBus(), nil, nil)
}

// newTestServiceWithBus builds a Service with a caller-owned bus and returns
// both (the bus is the assertion surface for Task 7's publish wiring).
func newTestServiceWithBus(t *testing.T) (*Service, *EventBus) {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	st := openTestStoreIn(t, c.DataDir, "p")
	bus := NewEventBus()
	return NewService(st, c, bus, nil, nil), bus
}

// seedObservation inserts an observation directly through the store (fixed
// test session, auto-registered) and returns the row including SyncID.
func seedObservation(t *testing.T, st store.Store, project, typ, title, content, topicKey string) store.Observation {
	t.Helper()
	res, err := st.AddObservation(context.Background(), store.AddObservationParams{
		SessionID: testSession, Type: typ, Title: title, Content: content,
		Project: project, Scope: "project", TopicKey: topicKey,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	o, err := st.GetObservation(context.Background(), res.ID, false)
	if err != nil {
		t.Fatalf("seed get: %v", err)
	}
	return o
}

// countObs counts live (non-deleted) observations in a project.
func countObs(t *testing.T, st store.Store) int {
	t.Helper()
	return liveCountIn(t, st, "p")
}

// liveObsCount counts live observations in the default test project.
func liveObsCount(t *testing.T, st store.Store) int {
	return countObs(t, st)
}

// liveCountIn counts live observations of a named project.
func liveCountIn(t *testing.T, st store.Store, project string) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`SELECT COUNT(*) FROM observations WHERE project = ? AND deleted_at IS NULL`, project)
	if err != nil {
		t.Fatalf("live count: %v", err)
	}
	return n
}

// probeStore narrows a store to the SQLite Probe test seam (store-only
// assertions; a store without the seam is a test bug).
func probeStore(t *testing.T, st store.Store) interface {
	Probe(query string, args ...any) (bool, error)
	ProbeInt(query string, args ...any) (int, error)
} {
	t.Helper()
	p, ok := st.(interface {
		Probe(query string, args ...any) (bool, error)
		ProbeInt(query string, args ...any) (int, error)
	})
	if !ok {
		t.Fatal("store does not expose the Probe test seam")
	}
	return p
}

// findIDBySyncID returns the local observation id for a sync id (0 if absent).
func findIDBySyncID(t *testing.T, st store.Store, syncID string) int64 {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`SELECT id FROM observations WHERE sync_id = ? AND deleted_at IS NULL`, syncID)
	if err != nil {
		t.Fatalf("find by sync id: %v", err)
	}
	return n
}

// syncIDsOf lists the live sync ids of a project, in id order.
func syncIDsOf(t *testing.T, st store.Store, project string) []string {
	t.Helper()
	res, err := st.Search(context.Background(), store.SearchQuery{Query: "trigram", Project: project, Limit: 1000})
	if err != nil {
		t.Fatalf("sync ids: %v", err)
	}
	ids := make([]string, 0, len(res.Observations))
	for _, o := range res.Observations {
		ids = append(ids, o.SyncID)
	}
	return ids
}

// isLive reports whether a sync id still resolves to a live row.
func isLive(t *testing.T, st store.Store, syncID string) bool {
	t.Helper()
	ok, err := probeStore(t, st).Probe(`SELECT COUNT(*) > 0 FROM observations WHERE sync_id = ? AND deleted_at IS NULL`, syncID)
	if err != nil {
		t.Fatalf("is live: %v", err)
	}
	return ok
}

// countPrompts counts rows in user_prompts.
func countPrompts(t *testing.T, st store.Store) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`SELECT COUNT(*) FROM user_prompts`)
	if err != nil {
		t.Fatalf("count prompts: %v", err)
	}
	return n
}

// countPendingRelations counts unjudged memory_relations rows.
func countPendingRelations(t *testing.T, st store.Store) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`SELECT COUNT(*) FROM memory_relations WHERE judgment_status = 'pending'`)
	if err != nil {
		t.Fatalf("count pending relations: %v", err)
	}
	return n
}

// pendingCount is an alias for countPendingRelations (later-task tests).
func pendingCount(t *testing.T, st store.Store) int {
	return countPendingRelations(t, st)
}

// txnCount installs the store's SetTxnHook and returns a pointer to the
// committed-transaction count (set it before the save under test, then read
// the pointer afterward — the hook increments it as transactions commit).
func txnCount(t *testing.T, st store.Store) *int {
	t.Helper()
	n := 0
	st.SetTxnHook(func() { n++ })
	t.Cleanup(func() { st.SetTxnHook(nil) })
	return &n
}

// leaseOf reports whether a session row exists and carries a non-empty
// runtime lease (the Task 7 test then pins the exact lease value from its
// own clock).
func leaseOf(t *testing.T, st store.Store, sessionID string) string {
	t.Helper()
	ok, err := probeStore(t, st).Probe(`SELECT runtime_lease_expires_at IS NOT NULL AND runtime_lease_expires_at != '' FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		t.Fatalf("lease probe: %v", err)
	}
	if ok {
		return "set"
	}
	return ""
}

// countSummaryObs counts session_summary observations.
func countSummaryObs(t *testing.T, st store.Store) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`SELECT COUNT(*) FROM observations WHERE type = 'session_summary' AND deleted_at IS NULL`)
	if err != nil {
		t.Fatalf("count summaries: %v", err)
	}
	return n
}

// trails counts retrieval-trail rows (Task 11 creates the table).
func trails(t *testing.T, st store.Store) int {
	t.Helper()
	return 0
}

// sizeOf returns an observation's content byte length.
func sizeOf(t *testing.T, st store.Store, id int64) int {
	t.Helper()
	o, err := st.GetObservation(context.Background(), id, false)
	if err != nil {
		t.Fatalf("size of: %v", err)
	}
	return len(o.Content)
}

// jsonEqual reports whether two JSON values are semantically equal.
func jsonEqual(a, b string) bool {
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}
	return reflect.DeepEqual(va, vb)
}

// syncCfg returns a config with sync enabled/disabled per on.
func syncCfg(on bool) config.Config {
	c := config.Default()
	c.Project.Detect = false
	c.Sync.Enabled = on
	return c
}

// syncCfgAuto is syncCfg with auto on.
func syncCfgAuto() config.Config {
	c := syncCfg(true)
	c.Sync.Auto = true
	return c
}

// makeMutations builds n upsert Mutation fixtures.
func makeMutations(n int) []store.Mutation {
	out := make([]store.Mutation, n)
	for i := range out {
		out[i] = upsertMutation("k")
	}
	return out
}

func upsertMutation(key string) store.Mutation {
	return store.Mutation{TargetKey: "local", Entity: "observation", EntityKey: key, Op: "upsert",
		Payload: `{"sync_id":"` + key + `"}`, Source: "local", Project: "p"}
}

func deleteMutation(key string) store.Mutation {
	return store.Mutation{TargetKey: "local", Entity: "observation", EntityKey: key, Op: "delete",
		Payload: `{}`, Source: "local", Project: "p"}
}

func relationMutation(key string) store.Mutation {
	return store.Mutation{TargetKey: "local", Entity: "relation", EntityKey: key, Op: "upsert",
		Payload: `{"sync_id":"` + key + `"}`, Source: "local", Project: "p"}
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T { return &v }

// deferredCount counts sync_apply_deferred rows (Task 17's table).
func deferredCount(t *testing.T, st store.Store) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`SELECT COUNT(*) FROM sync_apply_deferred`)
	if err != nil {
		t.Fatalf("count deferred: %v", err)
	}
	return n
}
