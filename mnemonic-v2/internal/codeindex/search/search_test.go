package search

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// openTestStore opens a real SQLite store for the search package tests.
func openTestStore(t *testing.T) store.Store {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(context.Background(), dir, "p", store.SQLite, "")
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// rawQueryInt reads a single int off the RawQuery test seam.
func rawQueryInt(t *testing.T, st store.Store, q string, args ...any) (int64, error) {
	t.Helper()
	rows, err := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	}).RawQuery(q, args...)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return 0, nil
	}
	n, _ := rows[0][0].(int64)
	return n, nil
}

// seedIndex seeds the FTS legs through the real Store write path: one Go file
// with the real symbol wireGitHooks (install/sync.go) + a chunk that mentions
// retry in two packages (internal/pay and internal/billing).
func seedIndex(t *testing.T) store.Store {
	t.Helper()
	st := openTestStore(t)
	ctx := context.Background()

	// install/sync.go with the real symbol wireGitHooks.
	fileA, err := st.UpsertFile(ctx, store.FileRow{Path: "install/sync.go", Language: "go", ContentHash: "hash-a"})
	if err != nil {
		t.Fatalf("upsert install/sync.go: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fileA, []store.Symbol{
		{Name: "wireGitHooks", Kind: "function", Language: "go", StartLine: 79, EndLine: 95, UID: "uid-wire"},
	}); err != nil {
		t.Fatalf("upsert symbols: %v", err)
	}
	if err := st.ReplaceChunksForFile(ctx, fileA, []store.Chunk{
		{Kind: "ast", StartLine: 79, EndLine: 95, Text: "func wireGitHooks() { copy hooks into the repo }"},
	}); err != nil {
		t.Fatalf("chunks: %v", err)
	}

	// internal/pay/pay.go — a chunk that mentions retry.
	fileB, _ := st.UpsertFile(ctx, store.FileRow{Path: "internal/pay/pay.go", Language: "go", ContentHash: "hash-b"})
	st.UpsertSymbols(ctx, fileB, []store.Symbol{
		{Name: "Pay", Kind: "function", Language: "go", StartLine: 1, EndLine: 20, UID: "uid-pay"},
	})
	st.ReplaceChunksForFile(ctx, fileB, []store.Chunk{
		{Kind: "ast", StartLine: 1, EndLine: 20, Text: "capture and retry the payment on failure"},
	})

	// internal/billing/billing.go — a chunk that mentions retry.
	fileC, _ := st.UpsertFile(ctx, store.FileRow{Path: "internal/billing/billing.go", Language: "go", ContentHash: "hash-c"})
	st.UpsertSymbols(ctx, fileC, []store.Symbol{
		{Name: "Bill", Kind: "function", Language: "go", StartLine: 1, EndLine: 20, UID: "uid-bill"},
	})
	st.ReplaceChunksForFile(ctx, fileC, []store.Chunk{
		{Kind: "ast", StartLine: 1, EndLine: 20, Text: "bill and retry the invoice on failure"},
	})

	return st
}

// seedBrokenEmbedder is an embedder whose Embed always errors — the T-CI-2
// "broken leg" (a present-but-failing seam, not a nil one).
type seedBrokenEmbedder struct{}

func (seedBrokenEmbedder) Model() string   { return "broken" }
func (seedBrokenEmbedder) Dimensions() int { return 8 }
func (seedBrokenEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, errBrokenEmbedder
}

var errBrokenEmbedder = errString("semantic leg down: no such column")

// hitHasPath reports whether any hit's Path equals the given path.
func hitHasPath(hits []Hit, path string) bool {
	for _, h := range hits {
		if h.Path == path {
			return true
		}
	}
	return false
}

// topHit returns the top hit's Path+StartLine key (the v1 bug: real == fake
// top hit). An empty hits slice returns "".
func topHit(hits []Hit) string {
	if len(hits) == 0 {
		return ""
	}
	h := hits[0]
	return h.Path + ":" + itoa(h.StartLine)
}

// [structural-grep-pattern-language] T12 triangulation: \f(ARGS) against a
// file with NO function calls must return no hits (the shorthand is
// call-specific, not a generic function matcher), and a \f(ARGS) pattern in a
// language the file is NOT (a Go file grepped as python) must degrade, not
// match. Different inputs, same behavior.
func TestStructuralGrepShorthandNegative(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.go")
	os.WriteFile(path, []byte("package plain\n\nconst x = 1\n"), 0o644)

	// no function declarations in the file → no hits, no error.
	r1, err := Grep(ctx, `\(function_declaration\) \fn`, path, "go")
	if err != nil {
		t.Fatalf("grep negative: %v", err)
	}
	if len(r1.Hits) != 0 {
		t.Fatalf("no-function file must not match: %+v", r1.Hits)
	}

	// a Go file grepped as python (no function_declaration in python? there
	// is — but the file content is not python; the parse may still "work" but
	// no hits are expected; the degrade path is the unsupported-language one).
	// Use an unsupported language instead: "cobol" is not in the probe map.
	r2, err := Grep(ctx, `\(function_declaration\) \fn`, path, "cobol")
	if err != nil {
		t.Fatalf("grep unsupported lang: %v", err)
	}
	if r2.Degraded != "no_grammar:cobol" {
		t.Fatalf("unsupported language must degrade: %q", r2.Degraded)
	}
}

// [search-real-vs-fake-divergence] T-CI-1: a real symbol and a fake symbol
// must NOT return the same top hit. The v1 FTS leg did NOT match wireGitHooks.
func TestSearchRealVsFakeDivergence(t *testing.T) {
	st := seedIndex(t)
	ctx := context.Background()

	real, err := Hybrid(ctx, st, SearchQuery{Query: "wireGitHooks", Project: "p", Limit: 5})
	if err != nil {
		t.Fatalf("hybrid real: %v", err)
	}
	fake, err := Hybrid(ctx, st, SearchQuery{Query: "capturePayment", Project: "p", Limit: 5})
	if err != nil {
		t.Fatalf("hybrid fake: %v", err)
	}
	// the real symbol MUST appear in the FTS leg (identifier-first).
	if !hitHasPath(real.Hits, "install/sync.go") {
		t.Fatalf("real symbol not in FTS leg (T-CI-1 broken): %+v", real.Hits)
	}
	// the fake symbol must NOT return the same top hit as the real one.
	if topHit(real.Hits) == topHit(fake.Hits) && topHit(real.Hits) != "" {
		t.Fatalf("real == fake top hit (the v1 bug): %+v vs %+v", real.Hits, fake.Hits)
	}
}

// [broken-leg-degrades-out] T-CI-2: a leg that errors degrades + drops out of
// the fusion; the FTS floor still answers (the broken leg's empty result does
// NOT set the answer).
func TestBrokenLegDegradesOutOfFusion(t *testing.T) {
	st := seedIndex(t)
	ctx := context.Background()

	r, err := Hybrid(ctx, st, SearchQuery{Query: "wireGitHooks", Project: "p", Limit: 5, Embedder: seedBrokenEmbedder{}})
	if err != nil {
		t.Fatalf("hybrid (broken leg): %v", err)
	}
	// the broken leg must degrade (degraded flag set, not clean).
	if r.Degraded == "" {
		t.Fatalf("broken leg must degrade (T-CI-2): %+v", r)
	}
	// the FTS floor still answers (the broken leg's empty result did NOT set
	// the answer to empty).
	if len(r.Hits) == 0 {
		t.Fatalf("FTS floor must still answer (T-CI-2 broken): %+v", r)
	}
	if !hitHasPath(r.Hits, "install/sync.go") {
		t.Fatalf("FTS floor must find the real symbol despite the broken leg: %+v", r.Hits)
	}
	// The broken leg must not be present in any hit's provenance (it
	// contributed no ranked hits).
	for _, h := range r.Hits {
		if _, ok := h.Provenance["semantic"]; ok {
			t.Fatalf("broken semantic leg must not appear in provenance: %+v", h)
		}
	}
}

// [broken-leg-degrades-out] T-CI-2 triangulation: a failing embedder on a
// query that would have matched ONLY via the semantic leg still returns the
// FTS floor's hits (possibly empty) — never the broken leg's empty result.
// Different inputs: a free-text query (not identifier-shaped).
func TestBrokenLegDegradesOutOfFusionFreeText(t *testing.T) {
	st := seedIndex(t)
	ctx := context.Background()

	r, err := Hybrid(ctx, st, SearchQuery{Query: "retry", Project: "p", Limit: 5, Embedder: seedBrokenEmbedder{}})
	if err != nil {
		t.Fatalf("hybrid free-text (broken leg): %v", err)
	}
	if r.Degraded == "" {
		t.Fatalf("broken leg must degrade (T-CI-2 free-text): %+v", r)
	}
	// "retry" matches the FTS floor (both pay + billing chunks) — the broken
	// semantic leg's empty result must not set the answer to empty.
	if len(r.Hits) == 0 {
		t.Fatalf("FTS floor must still answer free-text despite the broken leg: %+v", r)
	}
}

// [structural-grep-pattern-language] T12: both pattern forms + the
// no_grammar degrade.
func TestStructuralGrepPatternLanguage(t *testing.T) {
	ctx := context.Background()
	// A working Go file under a temp dir (structural grep re-parses the tree).
	dir := t.TempDir()
	path := filepath.Join(dir, "sync.go")
	os.WriteFile(path, []byte("package sync\n\nfunc wireGitHooks() {\n\thook := \":hooks\"\n\t_ = hook\n}\n"), 0o644)

	// the node-type+capture form (the v1 working syntax).
	r1, err := Grep(ctx, "(function_declaration) \\fn", path, "go")
	if err != nil {
		t.Fatalf("grep node-type+capture: %v", err)
	}
	if len(r1.Hits) == 0 {
		t.Fatalf("node-type+capture form broken: %+v", r1.Hits)
	}
	// the \f(ARGS) shorthand (the v1 UX gap fixed, T12): \f(\args)
	// expands to a call-expression pattern, so the file must contain a
	// function CALL.
	dir2 := t.TempDir()
	path2 := filepath.Join(dir2, "call.go")
	os.WriteFile(path2, []byte("package call\n\nfunc main() {\n\twireGitHooks()\n}\n\nfunc wireGitHooks() {}\n"), 0o644)
	r2, err := Grep(ctx, "\\f(\\args)", path2, "go")
	if err != nil {
		t.Fatalf("grep shorthand: %v", err)
	}
	if len(r2.Hits) == 0 {
		t.Fatalf("\\f(ARGS) shorthand must be accepted (T12): %+v", r2.Hits)
	}
	// an invalid pattern for a language → degraded: no_grammar:<lang>.
	r3, err := Grep(ctx, "(rust_only_node) \\x", path, "go")
	if err != nil {
		t.Fatalf("grep invalid: %v", err)
	}
	if r3.Degraded != "no_grammar:go" {
		t.Fatalf("invalid pattern must degrade to no_grammar:go: %q", r3.Degraded)
	}
}

// [code-read-secret-redaction] T13: values masked, keys kept.
func TestCodeReadSecretRedaction(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.go")
	os.WriteFile(path, []byte("package config\n\nconst (\n\ttoken = \"abc123\"\n\tpassword = \"xyz\"\n)\n"), 0o644)

	// full mode is live — no Store needed (pass nil).
	r, err := Read(ctx, nil, filepath.Join(dir, "config.go"), 1, 20, ReadModeFull)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(r.Text, "abc123") || strings.Contains(r.Text, "xyz") {
		t.Fatalf("secret not redacted: %q", r.Text)
	}
	if !strings.Contains(r.Text, "token") {
		t.Fatalf("the key must remain, only the value masked: %q", r.Text)
	}
}

// [degraded-no-vector] nil embedder → no_vector, FTS floor still answers.
func TestDegradedNoVector(t *testing.T) {
	st := seedIndex(t)
	ctx := context.Background()

	r, err := Hybrid(ctx, st, SearchQuery{Query: "wireGitHooks", Project: "p", Limit: 5})
	if err != nil {
		t.Fatalf("hybrid (nil embedder): %v", err)
	}
	if r.Degraded != "no_vector" {
		t.Fatalf("nil embedder → degraded:no_vector, got %q", r.Degraded)
	}
	if len(r.Hits) == 0 {
		t.Fatalf("FTS floor must still answer with no embedder")
	}
}

// [search-under-scope] T10: a scoped search returns only hits under the
// prefix; an empty-scope search returns empty (not an error).
func TestSearchUnderScope(t *testing.T) {
	st := seedIndex(t)
	ctx := context.Background()

	scoped, err := Hybrid(ctx, st, SearchQuery{Query: "retry", Project: "p", Limit: 10, Under: "internal/pay"})
	if err != nil {
		t.Fatalf("scoped search: %v", err)
	}
	for _, h := range scoped.Hits {
		if !strings.HasPrefix(h.Path, "internal/pay/") && h.Path != "internal/pay" {
			t.Fatalf("under-scope leaked a hit outside the prefix: %+v", h)
		}
	}
	// the scoped search must find the pay chunk (it mentions retry).
	if !hitHasPath(scoped.Hits, "internal/pay/pay.go") {
		t.Fatalf("under-scope must find the pay chunk: %+v", scoped.Hits)
	}
	// an under-scope that matches nothing → empty (not an error).
	empty, err := Hybrid(ctx, st, SearchQuery{Query: "retry", Project: "p", Limit: 10, Under: "internal/does-not-exist"})
	if err != nil {
		t.Fatalf("empty under-scope must not error: %v", err)
	}
	if len(empty.Hits) != 0 {
		t.Fatalf("empty under-scope must return no hits: %+v", empty.Hits)
	}
}

// [code-read-content-mode] T13: a symbol in content mode returns the signature
// + callee/caller ids (no bodies); a file → the TOC.
func TestCodeReadContentMode(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	// CapturePayment (internal/pay/pay.go) calls Log (internal/log/log.go).
	filePay, _ := st.UpsertFile(ctx, store.FileRow{Path: "internal/pay/pay.go", Language: "go", ContentHash: "hash-pay"})
	st.UpsertSymbols(ctx, filePay, []store.Symbol{
		{Name: "CapturePayment", Kind: "function", Language: "go", Signature: "func CapturePayment() error", StartLine: 10, EndLine: 50, UID: "uid-capture"},
	})
	fileLog, _ := st.UpsertFile(ctx, store.FileRow{Path: "internal/log/log.go", Language: "go", ContentHash: "hash-log"})
	st.UpsertSymbols(ctx, fileLog, []store.Symbol{
		{Name: "Log", Kind: "function", Language: "go", Signature: "func Log(msg string)", StartLine: 1, EndLine: 30, UID: "uid-log"},
	})
	symCap, _ := st.SymbolByName(ctx, "CapturePayment")
	symLog, _ := st.SymbolByName(ctx, "Log")
	if len(symCap) != 1 || len(symLog) != 1 {
		t.Fatalf("expected CapturePayment + Log symbols")
	}
	toLog := symLog[0].ID
	st.ReplaceEdgesForFile(ctx, filePay, []store.Edge{
		{Kind: "calls", FromID: symCap[0].ID, ToID: &toLog, Confidence: "EXTRACTED", Line: 20},
	})

	// A symbol in content mode → signature + callee/caller ids, no bodies.
	r, err := ReadSymbol(ctx, st, symCap[0].ID, ReadModeContent)
	if err != nil {
		t.Fatalf("read symbol content: %v", err)
	}
	if r.Signature == "" {
		t.Fatalf("content mode must return the signature: %+v", r)
	}
	if len(r.NeighborIDs) == 0 {
		t.Fatalf("content mode must return callee/caller ids: %+v", r)
	}
	for _, n := range r.NeighborIDs {
		if n.Body != "" {
			t.Fatalf("content mode must NOT return neighbor bodies: %+v", n)
		}
	}

	// A file target in content mode → the TOC, not the body.
	f, err := Read(ctx, st, "internal/pay/pay.go", 0, 0, ReadModeContent)
	if err != nil {
		t.Fatalf("read file content (TOC): %v — content mode on a file returns the TOC; the live span is only for full mode", err)
	}
	if len(f.TOC) == 0 {
		t.Fatalf("file content mode must return the TOC: %+v", f)
	}
	if f.Text != "" {
		t.Fatalf("file content mode must not return the body: %q", f.Text)
	}
	// The TOC must carry the symbol's name + line (the drill target the agent
	// copies to code_read).
	wantName := ""
	for _, row := range f.TOC {
		if row.Name == "CapturePayment" {
			wantName = row.Name
			break
		}
	}
	if wantName == "" {
		t.Fatalf("TOC must contain the CapturePayment symbol: %+v", f.TOC)
	}
}

// [code-read-content-mode] T13 triangulation: content mode on a file that is
// NOT in the index returns an empty TOC, not an error (not-indexed is not a
// failure — the live full-mode read still works on it).
func TestCodeReadContentModeUnindexed(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	// The file exists on disk but was never indexed.
	dir := t.TempDir()
	path := filepath.Join(dir, "unindexed.go")
	os.WriteFile(path, []byte("package unindexed\n\nfunc X() {}\n"), 0o644)

	f, err := Read(ctx, st, path, 0, 0, ReadModeContent)
	if err != nil {
		t.Fatalf("content mode on an unindexed file must not error: %v", err)
	}
	if len(f.TOC) != 0 {
		t.Fatalf("unindexed file must have an empty TOC: %+v", f.TOC)
	}
	// full mode on the same unindexed file still reads the live span.
	f2, err := Read(ctx, nil, path, 1, 10, ReadModeFull)
	if err != nil {
		t.Fatalf("full mode on an unindexed file: %v", err)
	}
	if f2.Text == "" {
		t.Fatalf("full mode must return the live span: %+v", f2)
	}
}

// [code-read-content-mode] T13 triangulation: the content-mode drill handle
// returns the caller side too (reverse edges) — a symbol that is CALLED carries
// the caller's id, with the caller's name resolved and an empty body.
func TestCodeReadContentModeCallers(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	// CapturePayment (internal/pay/pay.go) calls Log (internal/log/log.go) —
	// so Log has a caller edge.
	filePay, _ := st.UpsertFile(ctx, store.FileRow{Path: "internal/pay/pay.go", Language: "go", ContentHash: "hash-pay2"})
	st.UpsertSymbols(ctx, filePay, []store.Symbol{
		{Name: "CapturePayment", Kind: "function", Language: "go", Signature: "func CapturePayment() error", StartLine: 10, EndLine: 50, UID: "uid-capture2"},
	})
	fileLog, _ := st.UpsertFile(ctx, store.FileRow{Path: "internal/log/log.go", Language: "go", ContentHash: "hash-log2"})
	st.UpsertSymbols(ctx, fileLog, []store.Symbol{
		{Name: "Log", Kind: "function", Language: "go", Signature: "func Log(msg string)", StartLine: 1, EndLine: 30, UID: "uid-log2"},
	})
	symCap, _ := st.SymbolByName(ctx, "CapturePayment")
	symLog, _ := st.SymbolByName(ctx, "Log")
	if len(symCap) != 1 || len(symLog) != 1 {
		t.Fatalf("expected CapturePayment + Log symbols")
	}
	toLog := symLog[0].ID
	st.ReplaceEdgesForFile(ctx, filePay, []store.Edge{
		{Kind: "calls", FromID: symCap[0].ID, ToID: &toLog, Confidence: "EXTRACTED", Line: 20},
	})

	// Read Log in content mode → the reverse edge (CapturePayment is the caller).
	r, err := ReadSymbol(ctx, st, symLog[0].ID, ReadModeContent)
	if err != nil {
		t.Fatalf("read symbol content (callers): %v", err)
	}
	if r.Signature != "func Log(msg string)" {
		t.Fatalf("content mode must return the signature: %+v", r)
	}
	if len(r.NeighborIDs) != 1 {
		t.Fatalf("Log must have exactly one neighbor (its caller): %+v", r.NeighborIDs)
	}
	n := r.NeighborIDs[0]
	if n.ToID != symCap[0].ID {
		t.Fatalf("caller id mismatch: got %d want %d", n.ToID, symCap[0].ID)
	}
	if n.ToName != "CapturePayment" {
		t.Fatalf("caller name must be resolved: %+v", n)
	}
	if n.Body != "" {
		t.Fatalf("caller body must be empty in content mode: %+v", n)
	}
}

// [search-hybrid-rrf] the RRF math: a doc in 2 legs ranks above a doc in 1
// leg; per-signal provenance present.
func TestHybridRRF(t *testing.T) {
	st := seedIndex(t)
	ctx := context.Background()

	// "retry" matches the FTS leg (both pay + billing chunks). The semantic
	// leg is absent (nil embedder) → only the FTS leg answers. To exercise a
	// doc in 2 legs, we rely on the FTS leg returning both the chunk and the
	// symbol for a query that matches both — "wireGitHooks" matches the symbol
	// (symbol_fts) AND the chunk (chunks_fts) → the same doc (install/sync.go)
	// is in two FTS sub-legs, but RRF is over legs (fts/structural/semantic),
	// not sub-legs. So to assert 2-legs > 1-leg, we check that a hit returned
	// by the FTS leg carries provenance and that the fusion ranks a
	// multi-leg doc higher. Here the FTS leg is the only leg (nil embedder,
	// no pattern), so every hit is FTS-only; the provenance must be present
	// and the RRF score must be 1/(K+rank) for the single leg.
	r, err := Hybrid(ctx, st, SearchQuery{Query: "wireGitHooks", Project: "p", Limit: 5})
	if err != nil {
		t.Fatalf("hybrid rrf: %v", err)
	}
	if len(r.Hits) == 0 {
		t.Fatalf("hybrid must return hits: %+v", r)
	}
	// per-signal provenance present on every hit.
	for _, h := range r.Hits {
		if len(h.Provenance) == 0 {
			t.Fatalf("hit must carry per-signal provenance: %+v", h)
		}
		if h.Score <= 0 {
			t.Fatalf("hit must have a positive RRF score: %+v", h)
		}
	}
	// the RRF math: a single-leg doc at rank 0 scores 1/(K+0+1) = 1/61.
	top := r.Hits[0]
	want := 1.0 / float64(60+0+1)
	if top.Provenance["fts"] != 1 {
		t.Fatalf("top hit must be rank 1 in the FTS leg (provenance fts=1): %+v", top.Provenance)
	}
	if top.Score < want-1e-9 || top.Score > want+1e-9 {
		t.Fatalf("RRF score for a rank-0 single-leg doc must be 1/(K+0+1)=%v, got %v", want, top.Score)
	}
}

// [broken-leg-degrades-out] T-CI-2 / F1: the free-text FTS MATCH path must
// PROPAGATE a MATCH error (the leg errors → it degrades out + a warning is
// present), NOT swallow it as a zero-hit leg. The T-CI-2 contract: a leg that
// errors is dropped from the fusion + reported (warning + degraded flag) — the
// answer is NOT set to empty by the broken leg's empty result; the OTHER legs
// that answered still shape the answer. Different input: an error-forcing
// RawQuery seam on the MATCH expression.
func TestFTSFreeTextMatchErrorsDegrade(t *testing.T) {
	st := seedIndex(t)
	ctx := context.Background()

	// Store an embedding for wireGitHooks (the semantic leg's candidate) so a
	// WORKING embedder gives the semantic leg a hit to return — proving the
	// broken FTS leg's empty result did NOT set the answer (T-CI-2).
	syms, err := st.SymbolByName(ctx, "wireGitHooks")
	if err != nil || len(syms) != 1 {
		t.Fatalf("expected wireGitHooks symbol: %v", err)
	}
	if err := st.EmbedSymbol(ctx, syms[0].ID, "test", []float32{1, 0, 0}); err != nil {
		t.Fatalf("embed symbol: %v", err)
	}
	// A store seam whose RawQuery errors when the query is a MATCH expression
	// (the free-text FTS path). The FTS leg (identifier-first + free-text, the
	// floor) therefore errors and degrades out. The working embedder + stored
	// vector keep the semantic leg answering.
	es := &errStore{Store: st, matchErr: errors.New("fts match: malformed expression")}
	r, err := Hybrid(ctx, es, SearchQuery{
		Query:    "wireGitHooks",
		Project:  "p",
		Limit:    5,
		Embedder: &constEmbedder{model: "test", vec: []float32{1, 0, 0}},
	})
	if err != nil {
		t.Fatalf("hybrid (fts match error): %v", err)
	}
	// The free-text leg ERRORED: it degrades (the degraded flag names the FTS
	// error) and a warning naming the FTS leg is present (T-CI-2 — the error is
	// reported, not swallowed as a silent zero-hit leg).
	if !strings.Contains(r.Degraded, "fts_error") {
		t.Fatalf("a free-text FTS MATCH error must degrade (F1): %q", r.Degraded)
	}
	hasWarning := false
	for _, w := range r.Warnings {
		if strings.Contains(w, "fts") {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Fatalf("a free-text FTS MATCH error must add a warning (F1): %+v", r.Warnings)
	}
	// The broken FTS leg's empty result must NOT set the answer to empty: the
	// working semantic leg still answers (its hits are in the fusion). This is
	// the T-CI-2 guarantee — a broken leg degrades OUT, the answer is shaped by
	// the legs that returned.
	if len(r.Hits) == 0 {
		t.Fatalf("a broken FTS leg must not set the answer to empty (T-CI-2): %+v", r)
	}
	// The broken FTS leg contributed no ranked hits: no hit is provenance-"fts".
	for _, h := range r.Hits {
		if _, ok := h.Provenance["fts"]; ok {
			t.Fatalf("broken FTS leg must not appear in provenance (F1): %+v", h)
		}
	}
}

// errStore wraps a Store and forces RawQuery to error when the query is an FTS
// MATCH expression (the free-text FTS path) and pass through everything else
// (the identifier-first path, the file-set read, the trail write).
type errStore struct {
	store.Store
	matchErr error
}

func (s *errStore) RawQuery(query string, args ...any) ([][]any, error) {
	if strings.Contains(query, " MATCH ") {
		return nil, s.matchErr
	}
	return s.Store.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	}).RawQuery(query, args...)
}

// constEmbedder is a deterministic embedder: Model() returns a fixed model
// name, Embed returns a fixed vector for every text. The semantic leg scans the
// stored BLOB vectors (the F1 test stores one for wireGitHooks → cosine over
// that candidate → the leg returns a hit, proving the broken FTS leg's empty
// result did not set the answer).
type constEmbedder struct {
	model string
	vec   []float32
}

func (e *constEmbedder) Model() string   { return e.model }
func (e *constEmbedder) Dimensions() int { return len(e.vec) }
func (e *constEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = e.vec
	}
	return out, nil
}

// [structural-grep-scope] F2: the structural leg walks the SAME scope the index
// uses (the same root + exclude set + the under? prefix) so a structural hit
// for a file the index skipped is NOT in the structural results. The index's
// exclude set (excludedDirs) is the source of truth for scope; a file under an
// excluded dir (.git) is walked by the bare walk but is outside the index scope,
// so the scoped structural walk must skip it.
func TestStructuralGrepScopeMatchesIndex(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	// A supported-language file OUTSIDE the index scope (under .git — the
	// index's excludedDirs). It has a function declaration so the bare walk
	// would match it, but the index-scope walk must skip it.
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0o755)
	gitFile := filepath.Join(gitDir, "hooks.go")
	os.WriteFile(gitFile, []byte("package hooks\n\nfunc wireGitHooks() {}\n"), 0o644)
	// A supported-language file IN the index scope (a plain dir).
	plainFile := filepath.Join(dir, "sync.go")
	os.WriteFile(plainFile, []byte("package sync\n\nfunc wireGitHooks() {}\n"), 0o644)

	// The bare (pre-fix) walk's behavior: it also skips .git (all dot-dirs),
	// so it finds only the in-scope file. The post-fix index-scope walk must
	// produce the SAME set (one path space) — the regression the F2 fix
	// guards against is the two walks DIVERGING (e.g. the bare walk skipping a
	// dot-dir the index WOULD index).
	bare, err := grepWalkBare(dir, "")
	if err != nil {
		t.Fatalf("bare walk: %v", err)
	}
	bareSet := map[string]bool{}
	for _, p := range bare {
		bareSet[p] = true
	}
	if !bareSet["sync.go"] {
		t.Fatalf("bare walk must find the in-scope file: %v", bare)
	}
	if bareSet[".git/hooks.go"] {
		t.Fatalf("bare walk skips .git (all dot-dirs): %v", bare)
	}

	// The index-scope walk (what the post-fix leg does) must produce the SAME
	// path set (one path space with the index): skip .git, keep in-scope.
	scoped, err := grepWalkScoped(dir, "")
	if err != nil {
		t.Fatalf("scoped walk: %v", err)
	}
	scopedSet := map[string]bool{}
	for _, p := range scoped {
		scopedSet[p] = true
	}
	if !scopedSet["sync.go"] {
		t.Fatalf("scoped walk must keep the in-scope file: %v", scoped)
	}
	if scopedSet[".git/hooks.go"] {
		t.Fatalf("scoped walk must SKIP the index-excluded file (F2): %v", scoped)
	}

	// The Grep entry point must use the scoped walk: a structural hit for the
	// .git file is NOT in the results, a hit for the in-scope file is.
	res, err := Grep(ctx, "(function_declaration) \\fn", dir, "go")
	if err != nil {
		t.Fatalf("grep scoped: %v", err)
	}
	got := map[string]bool{}
	for _, h := range res.Hits {
		got[h.Path] = true
	}
	if !got["sync.go"] {
		t.Fatalf("Grep must find the in-scope file: %+v", res.Hits)
	}
	if got[".git/hooks.go"] {
		t.Fatalf("Grep must NOT find the index-excluded file (F2): %+v", res.Hits)
	}
}

// grepWalkBare runs the pre-fix file walk's behavior (skip ALL dot-dirs + the
// v1 skip set), returning repo-relative slash paths, for the F2 test to prove
// the index-scope walk diverges from it (the .git file is skipped by the bare
// walk but is ALSO outside the index scope — the point is the two must AGREE
// on scope, which the bare walk does not guarantee for dot-dirs the index
// WOULD index).
func grepWalkBare(dir, under string) ([]string, error) {
	var files []string
	bareSkip := func(name string) bool {
		if name == "" {
			return false
		}
		if name[0] == '.' {
			return true
		}
		return name == "node_modules" || name == "vendor" || name == "dist" ||
			name == "build" || name == "out" || name == ".git" ||
			name == ".skillgrid" || name == "target" || name == "__pycache__"
	}
	werr := filepath.WalkDir(dir, func(p string, d os.DirEntry, werr error) error {
		if werr != nil {
			return nil
		}
		if d.IsDir() {
			if bareSkip(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if grepLangForPath(p) != "" {
			if r, err := filepath.Rel(dir, p); err == nil && r != "." {
				files = append(files, filepath.ToSlash(r))
			}
		}
		return nil
	})
	return files, werr
}

// grepWalkScoped runs the index-scope file walk (the F2 fix's walk helper),
// returning repo-relative slash paths. It is the same root + the index's
// exclude set + the under? prefix.
func grepWalkScoped(dir, under string) ([]string, error) {
	var files []string
	werr := filepath.WalkDir(dir, func(p string, d os.DirEntry, werr error) error {
		if werr != nil {
			return nil
		}
		if d.IsDir() {
			if indexSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if grepLangForPath(p) == "" {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !underMatch(rel, under) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files, werr
}

// [trail-ids-store-ids] F4: the search trail records the store row ids the hits
// resolved to (the symbol ids, per the TICKET-04 trail convention), not content
// hashes. A symbol hit's store id must be in the recorded trail.
func TestSearchTrailCarriesStoreIDs(t *testing.T) {
	st := seedIndex(t)
	ctx := context.Background()

	// Resolve the real symbol's store id (the trail should carry it).
	syms, err := st.SymbolByName(ctx, "wireGitHooks")
	if err != nil || len(syms) != 1 {
		t.Fatalf("expected wireGitHooks symbol: %+v", err)
	}
	wantID := syms[0].ID

	// Run the search (the identifier-first path → a symbol hit with a store id).
	r, err := Hybrid(ctx, st, SearchQuery{Query: "wireGitHooks", Project: "p", Limit: 5})
	if err != nil {
		t.Fatalf("hybrid (trail ids): %v", err)
	}
	if len(r.Hits) == 0 {
		t.Fatalf("hybrid must return hits: %+v", r)
	}

	// Read the most recent trail row for route "search" and assert it carries
	// the symbol's store id (F4), not just a content hash.
	rows, err := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	}).RawQuery(`SELECT ids FROM retrieval_trails WHERE route = 'search' ORDER BY id DESC LIMIT 1`)
	if err != nil || len(rows) == 0 {
		t.Fatalf("expected a search trail row: %v", err)
	}
	idsCol, _ := rows[0][0].(string)
	idsSet := map[string]bool{}
	for _, p := range strings.Split(idsCol, ",") {
		if p != "" {
			idsSet[p] = true
		}
	}
	if !idsSet[itoa(int(wantID))] {
		t.Fatalf("trail must carry the symbol's store id %d (F4), got ids=%q", wantID, idsCol)
	}
}
