package extract

import (
	"strings"
	"testing"
)

// [chunker-ast-first] a file with one long method produces a single ast chunk
// under the target size (≤1000 chars); a config file with no symbols falls to
// plain line chunks (the 80/10 window).
func TestChunkerAstFirst(t *testing.T) {
	// one method, comfortably under 1000 chars → one ast chunk
	method := "func doWork() {\n" + strings.Repeat("\tx := 1\n", 20) + "}\n"
	g := FileGraph{
		Symbols: []Symbol{
			{Name: "doWork", Kind: "method", StartLine: 1, EndLine: 22},
		},
	}
	chunks := ChunkFile(g, method)
	if len(chunks) == 0 {
		t.Fatalf("no chunks for a method file")
	}
	for _, c := range chunks {
		if c.Kind != "ast" && c.Kind != "lines" {
			t.Fatalf("bad chunk kind: %s", c.Kind)
		}
	}
	// the method itself must be one ast chunk (under the target size).
	var methodChunks int
	for _, c := range chunks {
		if c.Kind == "ast" {
			methodChunks++
		}
	}
	if methodChunks == 0 {
		t.Fatalf("a function-like symbol under the target must be an ast chunk: %+v", chunks)
	}
}

// [chunker-ast-first] a file with no function-like symbols (a config file)
// falls to the plain line-chunk fallback (the 80-line window).
func TestChunkerLinesFallback(t *testing.T) {
	// 200 lines of config (no symbols) → plain line chunks, not ast.
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = "key=value"
	}
	content := strings.Join(lines, "\n")
	g := FileGraph{} // no symbols
	chunks := ChunkFile(g, content)
	if len(chunks) == 0 {
		t.Fatalf("no chunks for a config file (lines fallback broken)")
	}
	if chunks[0].Kind != "lines" {
		t.Fatalf("a symbol-less file must use the lines fallback, got kind %q", chunks[0].Kind)
	}
}

// [chunker-ast-first] an over-long method (a single symbol above the 1000-char
// target) is split into 80/10 line windows, not one giant ast chunk.
func TestChunkerOverLongMethodSplitsToLines(t *testing.T) {
	// one method far over 1000 chars (300 lines).
	method := "func big() {\n" + strings.Repeat("\tx := 1\n", 300) + "}\n"
	g := FileGraph{
		Symbols: []Symbol{
			{Name: "big", Kind: "method", StartLine: 1, EndLine: 302},
		},
	}
	chunks := ChunkFile(g, method)
	if len(chunks) == 0 {
		t.Fatalf("no chunks")
	}
	// the over-long method must be split: more than one chunk, all "lines".
	if len(chunks) < 2 {
		t.Fatalf("an over-long method must be split into line windows, got %d chunk(s)", len(chunks))
	}
	for _, c := range chunks {
		if c.Kind != "lines" {
			t.Fatalf("an over-long method must split to lines chunks, got kind %q", c.Kind)
		}
	}
}
