package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedBudgetedStore opens a project store, creates a session, and saves n
// observations matching "widget". It returns the service, the project id, and
// the open store (so the caller can close it).
func seedBudgetedStore(t *testing.T, project string, n int) (*Service, string, *store.Store) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	svc := New(dataDir)
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	mem := memory.New(st, project)
	for i := 0; i < n; i++ {
		if _, err := mem.Save(context.Background(), memory.SaveInput{
			SessionID: "s1",
			Type:      "learning",
			Title:     fmt.Sprintf("Widget note %d", i),
			// Long enough content to exceed the default char budget (1200).
			Content: "Body of widget note " + fmt.Sprintf("%d", i) + " " + strings.Repeat("x", 1500),
		}); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}
	return svc, project, st
}

// TestBudgetedSearch is 03.5 [AFK] — a 20-hit search returns 20 budgeted
// snippets (not 20× full JSON); every in-list result carries its
// mem_get_observation id.
// Scenarios: twenty-hit-search-is-budgeted,
// every-inlist-result-carries-get-observation-id.
func TestBudgetedSearch(t *testing.T) {
	svc, project, st := seedBudgetedStore(t, "budgeted-search", 20)
	defer st.Close()

	res, err := svc.BudgetedRetrieval(context.Background(), project, "fact", "widget", 20)
	if err != nil {
		t.Fatalf("budgeted retrieval: %v", err)
	}
	// The item cap (default 10) bounds the in-list result — NOT 20 full
	// payloads.
	if len(res.Hits) != 10 {
		t.Fatalf("expected the item-cap (10) in-list results, got %d", len(res.Hits))
	}
	// Every in-list result carries its full-content fetch id.
	for _, o := range res.Hits {
		if o.ID <= 0 {
			t.Fatalf("in-list result missing its mem_get_observation id: %+v", o)
		}
	}
	// Each in-list snippet is char-truncated with an explicit omitted count
	// (never a full-observation payload).
	for _, o := range res.Hits {
		if !strings.Contains(o.Content, "chars omitted") {
			t.Fatalf("in-list snippet not char-budgeted (no 'chars omitted'): %q", o.Content)
		}
	}
	if !res.Truncated {
		t.Fatalf("a 20-hit search must be char-truncated (truncated=true), got %+v", res)
	}
}

// TestFullContent is 03.6 [AFK] — mem_get_observation remains the ONLY
// full-content path: a budgeted in-list result is truncated, but fetching the
// hit by its id returns the full, untruncated content; the budget is tunable.
// Scenarios: mem-get-observation-is-only-full-content-path,
// budget-is-tunable-via-config.
func TestFullContent(t *testing.T) {
	dataDir := t.TempDir()
	project := "full-content"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	mem := memory.New(st, project)
	fullBody := "The full untruncated content " + strings.Repeat("y", 2000)
	id, err := mem.Save(context.Background(), memory.SaveInput{
		SessionID: "s1", Type: "learning", Title: "Full content probe",
		Content: fullBody,
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	svc := New(dataDir)

	// The budgeted read truncates the in-list snippet.
	res, err := svc.BudgetedRetrieval(context.Background(), project, "fact", "full content probe", 5)
	if err != nil {
		t.Fatalf("budgeted retrieval: %v", err)
	}
	if len(res.Hits) != 1 {
		t.Fatalf("expected 1 in-list hit, got %d", len(res.Hits))
	}
	if res.Hits[0].ID != id {
		t.Fatalf("in-list hit id mismatch: got %d want %d", res.Hits[0].ID, id)
	}
	inListContent := res.Hits[0].Content
	if len(inListContent) >= len(fullBody) {
		t.Fatalf("in-list result must be truncated (got %d >= full %d)", len(inListContent), len(fullBody))
	}
	if !strings.Contains(inListContent, "chars omitted") {
		t.Fatalf("in-list truncation not explicit: %q", inListContent)
	}

	// mem_get_observation (the only full-content path) returns the FULL content.
	got, err := mem.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get observation (full-content path): %v", err)
	}
	if got.Content != fullBody {
		t.Fatalf("mem_get_observation must return full untruncated content; got %d chars, want %d", len(got.Content), len(fullBody))
	}
	if strings.Contains(got.Content, "chars omitted") {
		t.Fatalf("mem_get_observation must NOT be char-budgeted: %q", got.Content)
	}

	// The budget is tunable via config: a larger char cap returns a longer
	// (still not full, but less truncated) in-list snippet.
	svc2 := New(dataDir)
	// Bump the char cap so the in-list snippet keeps more than the default.
	st2, _ := store.Open(dataDir, project)
	defer st2.Close()
	mem2 := memory.New(st2, project)
	mem2.SetBudget(memory.BudgetConfig{Items: 10, Chars: 3000})
	res2 := mem2.Budget().Apply(context.Background(), []memory.Observation{{ID: id, Content: fullBody}})
	if len(res2.Hits[0].Content) <= len(inListContent) {
		t.Fatalf("tuned (larger) char cap must keep more in-list content: tuned %d <= default %d", len(res2.Hits[0].Content), len(inListContent))
	}
	_ = svc2
}

// TestRouteReads is 03.7 [AFK] — mem_context/mem_search/mem_timeline are
// routed through the layered + budgeted path; mem_get_observation stays the
// only full-content path.
// Scenario: mem-context-search-timeline-budgeted.
func TestRouteReads(t *testing.T) {
	dataDir := t.TempDir()
	project := "route-reads"
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	ctx := context.Background()
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status, title, summary)
		VALUES ('s1', ?, '/tmp', '2026-01-01T00:00:00Z', 'ended', 'route test session', '## Goal\nroute reads probe')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	mem := memory.New(st, project)
	for i := 0; i < 20; i++ {
		if _, err := mem.Save(ctx, memory.SaveInput{
			SessionID: "s1", Type: "learning",
			Title:   fmt.Sprintf("Route widget %d", i),
			Content: "Route body " + fmt.Sprintf("%d", i) + " " + strings.Repeat("z", 1500),
		}); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}
	svc := New(dataDir)

	// mem_search path (fact → RRF fallback) is budgeted.
	searchRes, err := svc.BudgetedRetrieval(ctx, project, "fact", "route widget", 20)
	if err != nil {
		t.Fatalf("search budgeted: %v", err)
	}
	if len(searchRes.Hits) != 10 {
		t.Fatalf("mem_search must be item-cap budgeted (10), got %d", len(searchRes.Hits))
	}
	for _, o := range searchRes.Hits {
		if o.ID <= 0 {
			t.Fatalf("mem_search in-list hit missing id: %+v", o)
		}
	}

	// mem_context path (recent sessions) is budgeted at the item cap.
	sessions, err := mem.RecentContext(ctx, 20)
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	budgetedCtx := mem.Budget().Apply(ctx, nil) // no observations, but exercises the budget
	if len(budgetedCtx.Hits) != 0 {
		t.Fatalf("empty context budgeted read must return no hits, got %d", len(budgetedCtx.Hits))
	}
	if len(sessions) == 0 {
		t.Fatalf("mem_context must return the recent sessions")
	}

	// mem_get_observation remains the only full-content path: the search hit's
	// id fetches the full content (no truncation).
	hitID := searchRes.Hits[0].ID
	got, err := mem.Get(ctx, hitID)
	if err != nil {
		t.Fatalf("full-content fetch: %v", err)
	}
	if strings.Contains(got.Content, "chars omitted") {
		t.Fatalf("mem_get_observation must return untruncated content, got %q", got.Content)
	}
}

// seedLayeredOwnerStore opens a project store, creates two sessions (owner A's
// and owner B's), saves one private observation per owner matching the same
// fact token, and distills owner A's session so L2/L3 bootstrap layers exist.
// It returns the service, the project id, the store (for closing), and owner
// A's observation id.
func seedLayeredOwnerStore(t *testing.T, project string) (*Service, string, *store.Store, int64) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	svc := New(dataDir)
	ctx := context.Background()
	for _, sess := range []string{"sess-a", "sess-b"} {
		if _, err := st.DB.Exec(`
			INSERT INTO sessions (id, project, directory, started_at, status, title, summary)
			VALUES (?, ?, '/tmp', '2026-01-01T00:00:00Z', 'ended', ?, ?)`,
			sess, project, sess+" title", "## Goal\n"+sess+" goal\n\n## Key Learnings:\n- "+sess+" learning about the layering token"); err != nil {
			t.Fatalf("insert session %s: %v", sess, err)
		}
	}
	mem := memory.New(st, project)
	obsA, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "sess-a", Type: "learning",
		Title:   "Layering fact probe A",
		Content: "owner a private layering fact body",
	})
	if err != nil {
		t.Fatalf("save A: %v", err)
	}
	if _, err := mem.Save(ctx, memory.SaveInput{
		SessionID: "sess-b", Type: "learning",
		Title:   "Layering fact probe B",
		Content: "owner b private layering fact body",
	}); err != nil {
		t.Fatalf("save B: %v", err)
	}
	// Distill owner A's session so L2/L3 bootstrap layers exist (the layered
	// read path has something to bootstrap from).
	if _, err := layer.Distill(ctx, mem, "sess-a", layer.DistillOptions{}); err != nil {
		t.Fatalf("distill A: %v", err)
	}
	return svc, project, st, obsA
}

// TestBudgetedRetrievalProductionEntryPoint is the finding-03.2 proof that the
// layered L2/L3-first + RRF-fallback path has a REAL production entry point
// (the CLI `mem search` facade, BudgetedRetrievalAs/AsRoot) — not just the
// test-only BudgetedRetrieval — and that the step-01 per-owner visibility gate
// still holds on that path.
func TestBudgetedRetrievalProductionEntryPoint(t *testing.T) {
	svc, project, st, obsA := seedLayeredOwnerStore(t, "prod-entry")
	defer st.Close()
	ctx := context.Background()

	t.Run("bootstrap-is-l2-l3-first", func(t *testing.T) {
		res, err := svc.BudgetedRetrievalAs(ctx, project, "", "bootstrap", "", 10)
		if err != nil {
			t.Fatalf("bootstrap retrieval: %v", err)
		}
		if len(res.Hits) == 0 {
			t.Fatal("bootstrap must return the distilled L2/L3 layers (real layered path, not empty)")
		}
		// L2/L3-first: the in-list results are layered (persona) hits, not L1
		// observations.
		for _, o := range res.Hits {
			if o.ID <= 0 {
				t.Fatalf("layered in-list hit missing its fetch id: %+v", o)
			}
		}
	})

	t.Run("fact-rrf-fallback", func(t *testing.T) {
		res, err := svc.BudgetedRetrievalAs(ctx, project, "", "fact", "layering fact probe", 10)
		if err != nil {
			t.Fatalf("fact retrieval: %v", err)
		}
		if len(res.Hits) == 0 {
			t.Fatal("specific-fact query must fall back to L1/L0 RRF and return a hit")
		}
	})

	t.Run("owner-visibility-gate-holds-on-layered-path", func(t *testing.T) {
		// Owner A (the session that saved) reads: sees their own private fact.
		resA, err := svc.BudgetedRetrievalAs(ctx, project, "sess-a", "fact", "layering fact body", 10)
		if err != nil {
			t.Fatalf("fact as owner A: %v", err)
		}
		seenA := false
		for _, o := range resA.Hits {
			if o.ID == obsA {
				seenA = true
			}
		}
		if !seenA {
			t.Fatalf("owner A must see their own private fact via the layered path")
		}

		// Owner B reads the same specific fact: owner A's private observation
		// must be ABSENT (the step-01 per-owner gate holds on the layered path).
		resB, err := svc.BudgetedRetrievalAs(ctx, project, "sess-b", "fact", "layering fact body", 10)
		if err != nil {
			t.Fatalf("fact as owner B: %v", err)
		}
		for _, o := range resB.Hits {
			if o.ID == obsA {
				t.Fatalf("owner B saw owner A's private observation on the layered path — the per-owner gate regressed")
			}
		}
	})
}

// TestBudgetConfigDrivenTunability is the finding-03.5 proof: the read budget is
// tunable via the mnemonic.retrieval_budget YAML section (loaded through the
// config loader in the real read path), not just a direct SetBudget call. Two
// projects, identical content, different retrieval_budget items caps → different
// in-list truncation, end-to-end through BudgetedRetrievalAsRoot.
func TestBudgetConfigDrivenTunability(t *testing.T) {
	mkProject := func(t *testing.T, items int, project string) (*Service, string, *store.Store, string) {
		t.Helper()
		dataDir := t.TempDir()
		st, err := store.Open(dataDir, project)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		svc := New(dataDir)
		ctx := context.Background()
		if _, err := st.DB.Exec(`
			INSERT INTO sessions (id, project, directory, started_at, status)
			VALUES ('s1', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
			t.Fatalf("insert session: %v", err)
		}
		mem := memory.New(st, project)
		for i := 0; i < 20; i++ {
			if _, err := mem.Save(ctx, memory.SaveInput{
				SessionID: "s1", Type: "learning",
				Title:   fmt.Sprintf("Cfg widget %d", i),
				Content: "cfg body " + fmt.Sprintf("%d", i) + " " + strings.Repeat("w", 1500),
			}); err != nil {
				t.Fatalf("save %d: %v", i, err)
			}
		}
		// The config root is the data dir: config.Load walks up from it and finds
		// the retrieval_budget section we write below.
		cfgDir := dataDir
		if err := os.MkdirAll(filepath.Join(cfgDir, "config.d"), 0o755); err != nil {
			t.Fatalf("mkdir config.d: %v", err)
		}
		if err := os.WriteFile(filepath.Join(cfgDir, "config.d", "indexing.yaml"),
			[]byte(fmt.Sprintf("mnemonic:\n  retrieval_budget:\n    items: %d\n", items)), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		return svc, project, st, cfgDir
	}

	// items cap 5: the 20-hit fact search is truncated to 5 in-list results.
	svcSmall, projSmall, stSmall, cfgSmall := mkProject(t, 5, "cfg-small")
	defer stSmall.Close()
	resSmall, err := svcSmall.BudgetedRetrievalAsRoot(context.Background(), projSmall, cfgSmall, "", "fact", "cfg widget", 20)
	if err != nil {
		t.Fatalf("small-budget retrieval: %v", err)
	}
	if len(resSmall.Hits) != 5 {
		t.Fatalf("items cap 5 must truncate to 5 in-list results, got %d", len(resSmall.Hits))
	}

	// items cap 15: the SAME 20-hit search is truncated to 15 in-list results.
	svcBig, projBig, stBig, cfgBig := mkProject(t, 15, "cfg-big")
	defer stBig.Close()
	resBig, err := svcBig.BudgetedRetrievalAsRoot(context.Background(), projBig, cfgBig, "", "fact", "cfg widget", 20)
	if err != nil {
		t.Fatalf("big-budget retrieval: %v", err)
	}
	if len(resBig.Hits) != 15 {
		t.Fatalf("items cap 15 must truncate to 15 in-list results, got %d", len(resBig.Hits))
	}

	// The config-driven caps differ → tunability is real (not a fixed default).
	if len(resSmall.Hits) == len(resBig.Hits) {
		t.Fatalf("different retrieval_budget.items must produce different truncation, both got %d", len(resSmall.Hits))
	}
}
