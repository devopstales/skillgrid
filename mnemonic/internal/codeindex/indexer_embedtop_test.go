package codeindex

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// embedTopFixture writes the three-file fixture the gate tests index:
//
//   - core.go: a high-in-degree hub symbol `core` called by fn0..fn7 (8
//     symbols), so `core` sits at the top of the centrality ranking.
//   - doc.go: `documented` carries a `// NOTE:` rationale comment — the
//     doc-bearing carve-out (always embedded, whatever its centrality).
//   - recent.go: `recent` calls `core`; its mtime is set into the future so
//     the run classifies it as recently changed — the second carve-out.
//
// Total code symbols: 11 (core + fn0..fn7 + documented + recent).
func embedTopFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	var core strings.Builder
	core.WriteString("package main\n\nfunc core() {}\n")
	for i := 0; i < 8; i++ {
		core.WriteString("\nfunc fn" + itoa(i) + "() {\n\tcore()\n}\n")
	}
	mustWrite(t, filepath.Join(root, "core.go"), core.String())
	mustWrite(t, filepath.Join(root, "doc.go"), "package main\n\nfunc documented() {\n\t// NOTE: documented keeps a rationale comment (the doc carve-out).\n\t_ = 1\n}\n")
	recent := filepath.Join(root, "recent.go")
	mustWrite(t, recent, "package main\n\nfunc recent() {\n\tcore()\n}\n")
	// The recently-changed carve-out input: bump recent.go's mtime ahead of
	// now so the run treats it as a change since the last index.
	forceRecent(t, recent)
	return root
}

// forceRecent bumps a file's mtime into the future so the next index run
// records an mtime newer than indexed_at — the carve-out's change signal
// (a mtime newer than the last index = recently changed).
func forceRecent(t *testing.T, path string) {
	t.Helper()
	future := time.Now().Add(10 * time.Minute)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
}

// countCodeSymbols returns the number of code symbols (kind function/method/
// class) in the store.
func countCodeSymbols(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM symbols WHERE kind IN ('function','method','class')`).Scan(&n); err != nil {
		t.Fatalf("count code symbols: %v", err)
	}
	return n
}

// vectorSet returns the store's symbol vector set keyed by symbol name. A
// symbol with no embedding row (NULL vector) is simply absent.
func vectorSet(t *testing.T, db *sql.DB) map[string][]byte {
	t.Helper()
	rows, err := db.Query(`
		SELECT s.name, e.vector FROM embeddings e
		JOIN symbols s ON s.id = e.symbol_id`)
	if err != nil {
		t.Fatalf("query vectors: %v", err)
	}
	defer rows.Close()
	out := make(map[string][]byte)
	for rows.Next() {
		var name string
		var blob []byte
		if err := rows.Scan(&name, &blob); err != nil {
			t.Fatalf("scan vector: %v", err)
		}
		out[name] = blob
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

// TestIndex_EmbedTop covers @step-04 pagerank-embed (TICKET-04): the
// centrality-gated embedding pass.
//
//   - gate=50: at most ceil(N/2) centrality-selected code symbols are embedded,
//     doc-bearing (rationale) + recently-changed carve-outs are always embedded,
//     and the index output reports embedded/skipped counts.
//   - gate=100 (default): the vector set is identical to ungated indexing.
//   - a skipped symbol has a NULL vector and degrades (no error).
func TestIndex_EmbedTop(t *testing.T) {
	const totalSymbols = 11 // core + fn0..fn7 + documented + recent

	t.Run("gate 50 embeds at most half plus carve-outs and reports counts", func(t *testing.T) {
		idx, clean := newTestIndexerWithEmbedder(t)
		defer clean()
		root := embedTopFixture(t)

		// Cold baseline run (no gate): every file is freshly indexed, so there
		// is no "recently changed" file yet. This also establishes the prior
		// index state the second run's carve-out compares against.
		if _, err := idx.Run(context.Background(), root, testCfg); err != nil {
			t.Fatalf("baseline run: %v", err)
		}
		// Now edit recent.go (new content + a later mtime) so the second run
		// classifies it as recently changed — the carve-out input.
		recent := filepath.Join(root, "recent.go")
		if err := os.WriteFile(recent, []byte("package main\n\nfunc recent() {\n\tcore()\n\trecentChanged()\n}\n"), 0o644); err != nil {
			t.Fatalf("rewrite recent.go: %v", err)
		}
		forceRecent(t, recent)

		// Gated run: gate=50 over the now-changed tree.
		stats, err := idx.WithEmbedGate(50).Run(context.Background(), root, testCfg)
		if err != nil {
			t.Fatalf("run gate 50: %v", err)
		}

		n := countCodeSymbols(t, idx.store.DB)
		if n != totalSymbols {
			t.Fatalf("fixture invariant: expected %d code symbols, got %d", totalSymbols, n)
		}
		// The gate embeds at most ceil(N/2) top-centrality code symbols PLUS
		// the carve-outs (doc-bearing + recently-changed), which are additive.
		// This fixture has 2 carve-outs (documented, recent), so the bound is
		// ceil(N/2) + 2. Without the carve-out the bound is exactly ceil(N/2).
		const carveOutCount = 2
		embedded := vectorSet(t, idx.store.DB)
		if cap := (n+1)/2 + carveOutCount; len(embedded) > cap {
			t.Errorf("gate 50: expected at most %d embedded code symbols (ceil(%d/2)+%d carve-outs), got %d: %v", cap, n, carveOutCount, len(embedded), names(embedded))
		}
		if len(embedded) == 0 {
			t.Fatalf("gate 50: expected some embedded code symbols, got none")
		}
		// The gate must actually cut: it cannot embed every code symbol.
		if len(embedded) == n {
			t.Fatalf("gate 50: expected the gate to cut some symbols, but all %d are embedded: %v", n, names(embedded))
		}
		// Carve-outs are always embedded: the doc-bearing symbol and the
		// symbol in the recently-changed file.
		for _, want := range []string{"documented", "recent"} {
			if _, ok := embedded[want]; !ok {
				t.Errorf("carve-out %q must always be embedded; embedded set: %v", want, names(embedded))
			}
		}
		// The hub (top centrality) survives the cut.
		if _, ok := embedded["core"]; !ok {
			t.Errorf("hub symbol 'core' (top centrality) should be embedded; embedded set: %v", names(embedded))
		}
		// Skipped symbols carry a NULL vector (no embeddings row) — they
		// degrade to lexical, they do not error.
		skipped := 0
		for i := 0; i < 8; i++ {
			if _, ok := embedded["fn"+itoa(i)]; !ok {
				skipped++
			}
		}
		if skipped == 0 {
			t.Fatalf("gate 50: expected at least one fn* symbol skipped (NULL vector); all embedded: %v", names(embedded))
		}
		// A skipped symbol must have NO embeddings row (NULL vector), and the
		// run must not have errored (already checked above).
		var rows int
		if err := idx.store.DB.QueryRow(`SELECT COUNT(*) FROM symbols s
			WHERE s.name = ? AND NOT EXISTS (SELECT 1 FROM embeddings e WHERE e.symbol_id = s.id)`, "fn"+itoa(firstMissing(t, embedded, 8))).Scan(&rows); err != nil {
			t.Fatalf("null vector check: %v", err)
		}
		if rows != 1 {
			t.Errorf("expected the skipped symbol to have a NULL vector (no embeddings row), got %d rows", rows)
		}
		// The index output reports embedded/skipped counts, and they account
		// for every code symbol.
		if stats.EmbeddedSymbols+stats.SkippedSymbols != n {
			t.Errorf("counts must account for all %d code symbols: embedded=%d skipped=%d", n, stats.EmbeddedSymbols, stats.SkippedSymbols)
		}
		if stats.EmbeddedSymbols != len(embedded) {
			t.Errorf("reported embedded=%d but store holds %d vectors", stats.EmbeddedSymbols, len(embedded))
		}
		if stats.SkippedSymbols == 0 {
			t.Errorf("expected skipped symbols to be reported, got 0 (stats: %+v)", stats)
		}
	})

	t.Run("gate 100 is identical to ungated indexing", func(t *testing.T) {
		// Baseline: the default (ungated) run.
		base, baseClean := newTestIndexerWithEmbedder(t)
		defer baseClean()
		baseRoot := embedTopFixture(t)
		if _, err := base.Run(context.Background(), baseRoot, testCfg); err != nil {
			t.Fatalf("baseline run: %v", err)
		}
		baseSet := vectorSet(t, base.store.DB)

		// Explicit gate=100 must reproduce the identical vector set.
		idx, clean := newTestIndexerWithEmbedder(t)
		defer clean()
		root := embedTopFixture(t)
		stats, err := idx.WithEmbedGate(100).Run(context.Background(), root, testCfg)
		if err != nil {
			t.Fatalf("run gate 100: %v", err)
		}
		set := vectorSet(t, idx.store.DB)

		if len(set) != len(baseSet) {
			t.Fatalf("gate 100: expected %d vectors (ungated baseline), got %d", len(baseSet), len(set))
		}
		for name, blob := range set {
			b, ok := baseSet[name]
			if !ok {
				t.Errorf("gate 100 embedded %q which the ungated baseline did not", name)
				continue
			}
			if string(blob) != string(b) {
				t.Errorf("gate 100 vector for %q differs from the ungated baseline", name)
			}
		}
		// Everything is embedded at gate 100: nothing is skipped.
		if stats.SkippedSymbols != 0 {
			t.Errorf("gate 100: expected 0 skipped, got %d", stats.SkippedSymbols)
		}
		// Gate accounting is 0/0 when the gate is inactive (gate >= 100 is
		// today's behavior): it does not restate the full symbol count.
		if stats.EmbeddedSymbols != 0 {
			t.Errorf("gate 100: expected 0 gate accounting (inactive), got embedded=%d", stats.EmbeddedSymbols)
		}
	})
}

// names renders a vector-set key list for test diagnostics.
func names(set map[string][]byte) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// firstMissing returns the index of the first fn<i> (0..n-1) absent from set.
func firstMissing(t *testing.T, set map[string][]byte, n int) int {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, ok := set["fn"+itoa(i)]; !ok {
			return i
		}
	}
	t.Fatalf("no fn* symbol missing from set: %v", names(set))
	return 0
}
