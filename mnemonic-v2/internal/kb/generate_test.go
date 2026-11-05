package kb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

func TestGraduateGate(t *testing.T) {
	cases := []struct {
		name     string
		f        Fact
		graduate bool
		folder   string
	}{
		{"decision graduates", Fact{Type: "decision", Importance: 0.9}, true, "decisions"},
		{"bugfix maps to gotchas", Fact{Type: "bugfix", Importance: 0.5}, true, "gotchas"},
		{"convention maps to invariants", Fact{Type: "convention", Importance: 0.7}, true, "invariants"},
		{"config maps to invariants", Fact{Type: "config", Importance: 0.6}, true, "invariants"},
		{"architecture maps to architecture", Fact{Type: "architecture", Importance: 0.8}, true, "architecture"},
		{"discovery maps to gotchas", Fact{Type: "discovery", Importance: 0.5}, true, "gotchas"},
		{"learning maps to glossary", Fact{Type: "learning", Importance: 1.0}, true, "glossary"},
		{"exact floor passes", Fact{Type: "decision", Importance: 0.5}, true, "decisions"},
		{"below floor rejected", Fact{Type: "decision", Importance: 0.49}, false, ""},
		{"unknown type rejected", Fact{Type: "note", Importance: 0.9}, false, ""},
		{"empty type rejected", Fact{Type: "", Importance: 0.9}, false, ""},
	}
	for _, c := range cases {
		d := Graduate(c.f)
		if d.Graduate != c.graduate {
			t.Fatalf("%s: graduate = %v, want %v (%s)", c.name, d.Graduate, c.graduate, d.Reason)
		}
		if c.graduate && d.Type != c.folder {
			t.Fatalf("%s: folder = %q, want %q", c.name, d.Type, c.folder)
		}
	}
}

func TestGenerateCreatesNewPageViaFloor(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	st := openKbStore(t)
	seedFact(t, st, "proj-a", "auth", "Use OAuth2 for external auth")
	facts, err := AllFacts(ctx, st, "proj-a")
	if err != nil {
		t.Fatalf("facts: %v", err)
	}
	if len(facts) != 1 {
		t.Fatalf("facts: %d, want 1", len(facts))
	}

	res, err := Generate(ctx, st, vault, "proj-a", false, nil, facts)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if res.Noop {
		t.Fatalf("noop on first run: %+v", res)
	}
	if res.Changed != 1 || len(res.RelPaths) != 1 || res.RelPaths[0] != "decisions/tk-auth.md" {
		t.Fatalf("res: %+v", res)
	}
	if !res.Degraded {
		t.Fatal("nil synthesizer must report degraded")
	}

	body, err := os.ReadFile(filepath.Join(vault, "decisions", "tk-auth.md"))
	if err != nil {
		t.Fatalf("page not written: %v", err)
	}
	fm, _, _ := SplitFrontmatter(string(body))
	if fm.Type != "Decision" || fm.Project != "proj-a" {
		t.Fatalf("skeleton frontmatter: %+v", fm)
	}
	cited := false
	for _, s := range fm.Sources {
		if s.MemorySyncID() == facts[0].SyncID {
			cited = true
		}
	}
	if !cited {
		t.Fatalf("fact not cited in sources: %+v", fm.Sources)
	}
	if !strings.Contains(string(body), facts[0].Content) ||
		!strings.Contains(string(body), "> source: memory:"+facts[0].SyncID) {
		t.Fatalf("floor entry missing: %q", body)
	}

	// INV-5 page-write trail.
	rows, err := st.Query(ctx,
		`SELECT count(*) FROM kb_events WHERE project = 'proj-a' AND event_type = 'page_written' AND rel_path = 'decisions/tk-auth.md'`)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if n, _ := rows[0][0].(int64); n != 1 {
		t.Fatalf("page_written events: %d, want 1", n)
	}
	// The reindex step picked the page up.
	rows, err = st.Query(ctx,
		`SELECT type, status FROM kb_pages WHERE project = 'proj-a' AND rel_path = 'decisions/tk-auth.md'`)
	if err != nil || len(rows) != 1 {
		t.Fatalf("kb_pages: %v %+v", err, rows)
	}
	if s0(rows[0], 0) != "Decision" || s0(rows[0], 1) != "draft" {
		t.Fatalf("kb_pages row: %+v", rows[0])
	}
}

func TestGenerateFloorIdempotent(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	st := openKbStore(t)
	seedFact(t, st, "proj-a", "auth", "Use OAuth2 for external auth")
	facts, err := AllFacts(ctx, st, "proj-a")
	if err != nil {
		t.Fatalf("facts: %v", err)
	}

	res1, err := Generate(ctx, st, vault, "proj-a", false, nil, facts)
	if err != nil {
		t.Fatalf("generate 1: %v", err)
	}
	if res1.Changed != 1 {
		t.Fatalf("run 1: %+v, want changed=1", res1)
	}
	page := "decisions/tk-auth.md"
	body1, err := os.ReadFile(filepath.Join(vault, filepath.FromSlash(page)))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// Same facts again: the citation is already in the page → no rewrite.
	// Not a proven no-op (the grounded Decision page carries no pins → layer 2
	// cannot pass), but nothing is written.
	facts2, err := AllFacts(ctx, st, "proj-a")
	if err != nil {
		t.Fatalf("facts 2: %v", err)
	}
	res2, err := Generate(ctx, st, vault, "proj-a", false, nil, facts2)
	if err != nil {
		t.Fatalf("generate 2: %v", err)
	}
	if res2.Noop {
		t.Fatalf("run 2 unexpectedly a no-op: %+v", res2)
	}
	if res2.Changed != 0 || len(res2.RelPaths) != 0 {
		t.Fatalf("run 2: %+v, want changed=0", res2)
	}
	body2, err := os.ReadFile(filepath.Join(vault, filepath.FromSlash(page)))
	if err != nil {
		t.Fatalf("read 2: %v", err)
	}
	if string(body1) != string(body2) {
		t.Fatal("floor rewrote an already-merged fact")
	}
}

func TestGenerateProvenNoop(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	st := openKbStore(t)
	// A non-grounded page type (Gotcha): layers 2/3 skip, so a clean run is
	// provable from layer 1 (processed) + layer 4 (fingerprint stable).
	if _, err := st.AddObservation(ctx, store.AddObservationParams{
		SessionID: "s", Type: "bugfix", Title: "bug", Content: "retry storm on 502",
		Project: "proj-a", Scope: "project", TopicKey: "tk-bug",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	facts, err := AllFacts(ctx, st, "proj-a")
	if err != nil {
		t.Fatalf("facts: %v", err)
	}

	res1, err := Generate(ctx, st, vault, "proj-a", false, nil, facts)
	if err != nil {
		t.Fatalf("generate 1: %v", err)
	}
	if res1.Noop || res1.Changed != 1 {
		t.Fatalf("run 1: %+v, want changed=1", res1)
	}

	res2, err := Generate(ctx, st, vault, "proj-a", false, nil, facts)
	if err != nil {
		t.Fatalf("generate 2: %v", err)
	}
	if !res2.Noop {
		t.Fatalf("run 2 not a proven no-op: %+v", res2)
	}
	if res2.Changed != 0 || !strings.Contains(res2.Reason, "no-op") {
		t.Fatalf("run 2: %+v", res2)
	}
}

func TestGenerateLayer1BlocksNoop(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	st := openKbStore(t)
	if _, err := st.AddObservation(ctx, store.AddObservationParams{
		SessionID: "s", Type: "bugfix", Title: "bug", Content: "retry storm on 502",
		Project: "proj-a", Scope: "project", TopicKey: "tk-bug",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	facts, _ := AllFacts(ctx, st, "proj-a")
	if _, err := Generate(ctx, st, vault, "proj-a", false, nil, facts); err != nil {
		t.Fatalf("generate 1: %v", err)
	}

	// A new fact that does not graduate: unprocessed → layer 1 fails → the
	// run is not a no-op, but nothing changes either.
	if _, err := st.AddObservation(ctx, store.AddObservationParams{
		SessionID: "s", Type: "note", Title: "note", Content: "low signal",
		Project: "proj-a", Scope: "project", TopicKey: "tk-note",
	}); err != nil {
		t.Fatalf("seed note: %v", err)
	}
	facts2, _ := AllFacts(ctx, st, "proj-a")
	res2, err := Generate(ctx, st, vault, "proj-a", false, nil, facts2)
	if err != nil {
		t.Fatalf("generate 2: %v", err)
	}
	if res2.Noop || res2.Changed != 0 {
		t.Fatalf("run 2: %+v, want noop=false changed=0", res2)
	}

	// Once the new fact is processed, the same inputs are a proven no-op.
	res3, err := Generate(ctx, st, vault, "proj-a", false, nil, facts2)
	if err != nil {
		t.Fatalf("generate 3: %v", err)
	}
	if !res3.Noop {
		t.Fatalf("run 3: %+v, want proven no-op", res3)
	}
}

// citedPage writes a grounded page that cites the given fact sync_id, so
// TargetPages resolves the fact onto it.
func citedPage(t *testing.T, vault, relPath, syncID, fileContent string) {
	t.Helper()
	writeVaultPage(t, vault, relPath,
		"---\n"+
			"title: Cache TTL\n"+
			"type: Decision\n"+
			"project: proj-a\n"+
			"anchors:\n"+
			"  - path: internal/cache/cache.go\n"+
			"    all_of:\n"+
			"      - ttl = 15 * time.Minute\n"+
			"sources:\n"+
			"  - memory:"+syncID+"\n"+
			"---\n\n"+
			"We cache for 15 minutes.\n")
	if fileContent != "" {
		writeVaultPage(t, vault, "internal/cache/cache.go", fileContent)
	}
}

func TestGenerateLayer2BlocksNoop(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	st := openKbStore(t)
	id, err := st.AddObservation(ctx, store.AddObservationParams{
		SessionID: "s", Type: "decision", Title: "cache ttl", Content: "ttl 15m",
		Project: "proj-a", Scope: "project", TopicKey: "tk-cache",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Grounded page, pin UNRESOLVABLE (anchor file missing).
	citedPage(t, vault, "decisions/cache-ttl.md", id.SyncID, "")
	if _, err := Index(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("index: %v", err)
	}
	facts, _ := AllFacts(ctx, st, "proj-a")

	if _, err := Generate(ctx, st, vault, "proj-a", false, nil, facts); err != nil {
		t.Fatalf("generate 1: %v", err)
	}
	// Layer 3 is satisfied by a manually recorded current baseline, so the
	// only failing layer is 2 (the pin re-check).
	fp, err := SourceFingerprint(ctx, st, "proj-a", vault)
	if err != nil {
		t.Fatalf("fp: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(vault, "decisions", "cache-ttl.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := RecordBaseline(ctx, st, vault, "proj-a", "decisions/cache-ttl.md",
		fp, hashBody(string(body)), "test", "test"); err != nil {
		t.Fatalf("baseline: %v", err)
	}

	facts2, _ := AllFacts(ctx, st, "proj-a")
	res2, err := Generate(ctx, st, vault, "proj-a", false, nil, facts2)
	if err != nil {
		t.Fatalf("generate 2: %v", err)
	}
	if res2.Noop {
		t.Fatalf("run 2: %+v, want blocked by stale pin (layer 2)", res2)
	}
	if res2.Changed != 0 {
		t.Fatalf("run 2: %+v, want changed=0", res2)
	}
}

func TestGenerateLayer3BlocksNoop(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	st := openKbStore(t)
	id, err := st.AddObservation(ctx, store.AddObservationParams{
		SessionID: "s", Type: "decision", Title: "cache ttl", Content: "ttl 15m",
		Project: "proj-a", Scope: "project", TopicKey: "tk-cache",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Grounded page with a RESOLVABLE pin (fresh) but no baseline.
	citedPage(t, vault, "decisions/cache-ttl.md", id.SyncID, "package cache\n\nvar ttl = 15 * time.Minute\n")
	if _, err := Index(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("index: %v", err)
	}
	facts, _ := AllFacts(ctx, st, "proj-a")

	if _, err := Generate(ctx, st, vault, "proj-a", false, nil, facts); err != nil {
		t.Fatalf("generate 1: %v", err)
	}
	facts2, _ := AllFacts(ctx, st, "proj-a")
	res2, err := Generate(ctx, st, vault, "proj-a", false, nil, facts2)
	if err != nil {
		t.Fatalf("generate 2: %v", err)
	}
	if res2.Noop {
		t.Fatalf("run 2: %+v, want blocked by missing baseline (layer 3)", res2)
	}

	// A current baseline completes the proof.
	fp, err := SourceFingerprint(ctx, st, "proj-a", vault)
	if err != nil {
		t.Fatalf("fp: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(vault, "decisions", "cache-ttl.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := RecordBaseline(ctx, st, vault, "proj-a", "decisions/cache-ttl.md",
		fp, hashBody(string(body)), "test", "test"); err != nil {
		t.Fatalf("baseline: %v", err)
	}
	res3, err := Generate(ctx, st, vault, "proj-a", false, nil, facts2)
	if err != nil {
		t.Fatalf("generate 3: %v", err)
	}
	if !res3.Noop {
		t.Fatalf("run 3: %+v, want proven no-op once baseline recorded", res3)
	}
}

func TestGenerateLayer4BlocksNoop(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	st := openKbStore(t)
	if _, err := st.AddObservation(ctx, store.AddObservationParams{
		SessionID: "s", Type: "bugfix", Title: "bug", Content: "retry storm on 502",
		Project: "proj-a", Scope: "project", TopicKey: "tk-bug",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	facts, _ := AllFacts(ctx, st, "proj-a")
	res1, err := Generate(ctx, st, vault, "proj-a", false, nil, facts)
	if err != nil {
		t.Fatalf("generate 1: %v", err)
	}
	if res1.Noop || res1.Changed != 1 {
		t.Fatalf("run 1: %+v", res1)
	}

	// An unstable HEAD: the first and second fingerprint re-takes differ.
	orig := gitHead
	n := 0
	t.Cleanup(func() { gitHead = orig })
	gitHead = func(string) string { n++; return fmt.Sprintf("head-%d", n) }

	res2, err := Generate(ctx, st, vault, "proj-a", false, nil, facts)
	if err != nil {
		t.Fatalf("generate 2: %v", err)
	}
	if res2.Noop {
		t.Fatalf("run 2: %+v, want blocked by fingerprint drift (layer 4)", res2)
	}
	if res2.Changed != 0 {
		t.Fatalf("run 2: %+v, want changed=0", res2)
	}
}
