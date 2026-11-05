package kb

import (
	"context"
	"strings"
	"testing"
)

func TestTokenSpanInPlace(t *testing.T) {
	lines := []string{
		"package main", "", "import \"fmt\"", "",
		"func Compute(x int) int {", "\treturn x * 2", "}",
		"", "func main() {", "\tfmt.Println(1)", "}",
	}
	content := strings.Join(lines, "\n")
	tok := TokenForSpan(content, 5, 3) // the function, 3 lines
	if tok == "" || !strings.HasPrefix(tok, "kb-lines-v1:sha256:") {
		t.Fatalf("token: %q", tok)
	}
	res, line, snippet := ResolveSpanToken(content, tok, 5)
	if res != SpanOK || line != 5 {
		t.Fatalf("in place: res=%q line=%d", res, line)
	}
	if snippet != "func Compute(x int) int {\n\treturn x * 2\n}" {
		t.Fatalf("snippet: %q", snippet)
	}
}

func TestTokenSpanRelocates(t *testing.T) {
	lines := []string{
		"package main", "", "type A struct{ V int }", "",
		"func Compute(x int) int {", "\treturn x * 2", "}",
		"", "func other() {}",
	}
	content := strings.Join(lines, "\n")
	tok := TokenForSpan(content, 5, 3)

	// Move the function down by 4 lines (line drift free).
	moved := strings.Join(append([]string{"// a", "// b", "// c", "// d"}, lines...), "\n")
	res, line, _ := ResolveSpanToken(moved, tok, 5)
	if res != SpanOK || line != 9 {
		t.Fatalf("relocate: res=%q line=%d (want ok @9)", res, line)
	}
}

func TestTokenSpanStaleUnique(t *testing.T) {
	lines := []string{
		"package main", "", "const Limit = 100", "",
		"func Compute(x int) int {", "\treturn x * 2", "}",
		"", "func other() {}",
	}
	content := strings.Join(lines, "\n")
	tok := TokenForSpan(content, 5, 3)

	// Content changed at the same site (context identical).
	changed := strings.Replace(content, "\treturn x * 2", "\treturn x * 3", 1)
	res, line, snippet := ResolveSpanToken(changed, tok, 5)
	if res != SpanStale || line != 5 {
		t.Fatalf("stale: res=%q line=%d", res, line)
	}
	if !strings.Contains(snippet, "x * 3") {
		t.Fatalf("live value not reported: %q", snippet)
	}
}

func TestTokenSpanAmbiguousUnresolved(t *testing.T) {
	// Two identical spans with identical preceding (X) AND following (Y)
	// context: the decision anchors cannot disambiguate → unresolved (fail
	// closed). File shape: X B Y X B Y.
	x := []string{"var x1 = 1", "var x2 = 2", "var x3 = 3"}
	y := []string{"var y1 = 1", "var y2 = 2", "var y3 = 3"}
	block := []string{"func A() int {", "\treturn 1", "}"}
	lines := append(append(append(append(append([]string{}, x...), block...), y...), x...), append(block, y...)...)
	content := strings.Join(lines, "\n")
	// Pin the first copy (line 4).
	tok := TokenForSpan(content, 4, 3)
	// Resolve with a hint that misses both copies → scan → two matches.
	res, line, _ := ResolveSpanToken(content, tok, 1)
	if res != SpanUnresolved || line != 0 {
		t.Fatalf("ambiguous: res=%q line=%d (want unresolved)", res, line)
	}
}

func TestTokenSpanGarbageToken(t *testing.T) {
	for _, tok := range []string{"", "garbage", "kb-lines-v1:sha256:xx:!!", "kb-lines-v1:sha256:abcd"} {
		if res, _, _ := ResolveSpanToken("a\nb\nc", tok, 1); res != SpanUnresolved {
			t.Fatalf("token %q: res=%q (want unresolved)", tok, res)
		}
	}
}

func TestValidatePageTokenAndClaim(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	root := t.TempDir()
	writeVaultPage(t, root, "internal/auth/oauth.go", "package auth\n\nfunc NewOAuthClient() {}\n\nconst expiry = 15 * time.Minute\n")
	fm := Frontmatter{
		Title: "Auth Model", Type: "Decision",
		Anchors: []Anchor{{Path: "internal/auth/oauth.go", AllOf: []string{"func NewOAuthClient()"}}},
		Claims:  []Claim{{Subject: "expiry", Literal: "15 * time.Minute"}},
	}

	pc, err := ValidatePage(ctx, st, root, "decisions/auth-model.md", fm)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if pc.Status != "fresh" || len(pc.Checks) != 2 {
		t.Fatalf("want fresh with 2 checks: %+v", pc)
	}

	// Token construct renamed → unresolved.
	writeVaultPage(t, root, "internal/auth/oauth.go", "package auth\n\nfunc NewClient() {}\n\nconst expiry = 15 * time.Minute\n")
	pc, err = ValidatePage(ctx, st, root, "decisions/auth-model.md", fm)
	if err != nil {
		t.Fatalf("validate2: %v", err)
	}
	if pc.Status != "unresolved" {
		t.Fatalf("renamed token: %+v", pc)
	}

	// Claim literal gone → stale.
	writeVaultPage(t, root, "internal/auth/oauth.go", "package auth\n\nfunc NewOAuthClient() {}\n\nconst expiry = 30 * time.Minute\n")
	pc, err = ValidatePage(ctx, st, root, "decisions/auth-model.md", fm)
	if err != nil {
		t.Fatalf("validate3: %v", err)
	}
	if pc.Status != "stale" || pc.StaleChecks != 1 {
		t.Fatalf("claim stale: %+v", pc)
	}
}

func TestValidatePageSpanAnchors(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	root := t.TempDir()
	body := "package kb\n\nfunc Compute(x int) int {\n\treturn x * 2\n}\n\nfunc other() {}\n"
	writeVaultPage(t, root, "internal/kb/pins.go", body)
	tok := TokenForSpan(body, 3, 3)
	fm := Frontmatter{
		Title: "T", Type: "Invariant",
		Anchors: []Anchor{{Path: "internal/kb/pins.go", Line: 3, Version: tok}},
		Claims:  []Claim{{Subject: "compute", Literal: "return x * 2"}},
	}

	// Fresh.
	pc, err := ValidatePage(ctx, st, root, "invariants/i.md", fm)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if pc.Status != "fresh" {
		t.Fatalf("fresh: %+v", pc)
	}

	// Relocated → still fresh (line drift free; the 3-line context is
	// preserved because the insertion is far above the window).
	moved := "// c1\n// c2\n// c3\n// c4\n" + body
	writeVaultPage(t, root, "internal/kb/pins.go", moved)
	pc, err = ValidatePage(ctx, st, root, "invariants/i.md", fm)
	if err != nil {
		t.Fatalf("validate2: %v", err)
	}
	if pc.Status != "fresh" {
		t.Fatalf("relocated should be fresh: %+v", pc)
	}

	// Changed → stale with the live value.
	changed := strings.Replace(body, "return x * 2", "return x * 4", 1)
	writeVaultPage(t, root, "internal/kb/pins.go", changed)
	pc, err = ValidatePage(ctx, st, root, "invariants/i.md", fm)
	if err != nil {
		t.Fatalf("validate3: %v", err)
	}
	if pc.Status != "stale" {
		t.Fatalf("changed should be stale: %+v", pc)
	}
	found := false
	for _, c := range pc.Checks {
		if c.Kind == "span" && c.Result == SpanStale && strings.Contains(c.Actual, "x * 4") {
			found = true
		}
	}
	if !found {
		t.Fatalf("live value not surfaced: %+v", pc.Checks)
	}
}

func TestValidatePageUnattested(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	pc, err := ValidatePage(ctx, st, t.TempDir(), "decisions/bare.md", Frontmatter{Title: "T", Type: "Decision"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if pc.Status != "unresolved" {
		t.Fatalf("grounded page with no pins must be unattested: %+v", pc)
	}
}

func TestValidatePageSymbolDegraded(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t) // empty graph: no indexed files
	fixture := t.TempDir()
	writeVaultPage(t, fixture, "cmd/server/main.go", "package main\n\nfunc main() { NewOAuthClient() }\n")

	fm := Frontmatter{
		Title: "T", Type: "Decision",
		Anchors: []Anchor{{Path: "cmd/server/main.go", Symbol: "NewOAuthClient"}},
	}
	pc, err := ValidatePage(ctx, st, fixture, "decisions/s.md", fm)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if pc.Status != "fresh" || !pc.Degraded {
		t.Fatalf("degraded occurrence should be ok+degraded: %+v", pc)
	}

	// Symbol absent → unresolved even degraded.
	fm2 := Frontmatter{Title: "T", Type: "Decision",
		Anchors: []Anchor{{Path: "cmd/server/main.go", Symbol: "GoneSymbol"}}}
	pc2, err := ValidatePage(ctx, st, fixture, "decisions/s.md", fm2)
	if err != nil {
		t.Fatalf("validate2: %v", err)
	}
	if pc2.Status != "unresolved" {
		t.Fatalf("absent symbol: %+v", pc2)
	}
}

func TestValidatePageAnchorFileMissing(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	fm := Frontmatter{
		Title: "T", Type: "Workflow",
		Anchors: []Anchor{{Path: "gone/file.go", AllOf: []string{"x"}}},
	}
	pc, err := ValidatePage(ctx, st, t.TempDir(), "workflows/w.md", fm)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if pc.Status != "unresolved" {
		t.Fatalf("missing anchor file: %+v", pc)
	}
}
