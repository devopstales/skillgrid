package codeindex

import (
	"fmt"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/extract"
)

// TestChunkLinesAstCoalescesFunctions verifies AST chunking coalesces adjacent
// small functions into one ast chunk, skips non-function symbols (type/var),
// and line-windows a single function over the char target.
func TestChunkLinesAstOneChunkPerFunction(t *testing.T) {
	// Each function becomes its own ast chunk (no cross-contamination between
	// two functions' embeddings). The package clause and a trailing const are
	// line chunks, so the whole file is covered — a gap would lose the const
	// from code_read (and skip secret redaction on it).
	src := "package svc\n\nfunc Load(id int) (*Doc, error) {\n\treturn fetch(id)\n}\nfunc Store(d *Doc) error {\n\treturn save(d)\n}\nconst secret = \"AKIAIOSFODNN7EXAMPLE\"\n"
	syms := []extract.Symbol{
		{Name: "Load", Kind: "function", StartLine: 3, EndLine: 5, Language: "go"},
		{Name: "Store", Kind: "function", StartLine: 6, EndLine: 8, Language: "go"},
	}
	chunks := ChunkLinesAst([]byte(src), syms)
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	// Two ast chunks: Load (3-5) and Store (6-8).
	var astCount int
	for _, c := range chunks {
		if c.Kind != "ast" {
			continue
		}
		astCount++
		if (c.StartLine == 3 && c.EndLine == 5) || (c.StartLine == 6 && c.EndLine == 8) {
			continue
		}
		t.Errorf("unexpected ast chunk %d-%d", c.StartLine, c.EndLine)
	}
	if astCount != 2 {
		t.Errorf("expected 2 ast chunks (one per function), got %d", astCount)
	}
	// Whole-file coverage: every non-blank line in exactly one chunk (the
	// const on line 9 must be covered — a gap would lose it from code_read).
	if err := assertFullCoverage([]byte(src), chunks); err != nil {
		t.Errorf("full coverage: %v", err)
	}
	// The trailing const must be in a (line) chunk.
	var constCovered bool
	for _, c := range chunks {
		if c.StartLine <= 9 && 9 <= c.EndLine {
			constCovered = true
		}
	}
	if !constCovered {
		t.Error("trailing const (line 9) not covered by any chunk")
	}
}

// TestChunkLinesAstCoalescesNestedSpan verifies a single span containing
// nested symbols (a class with methods, extracted as one span) becomes ONE ast
// chunk — the cocoindex "small adjacent functions form a coherent unit" case.
func TestChunkLinesAstCoalescesNestedSpan(t *testing.T) {
	src := "package svc\n\nfunc Outer() {\n\tinner()\n}\nfunc inner() {}\n"
	// One span covering both (e.g. a class with methods extracted as one node).
	syms := []extract.Symbol{
		{Name: "Outer", Kind: "function", StartLine: 3, EndLine: 5, Language: "go"},
	}
	chunks := ChunkLinesAst([]byte(src), syms)
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	var ast *Chunk
	for i := range chunks {
		if chunks[i].Kind == "ast" {
			ast = &chunks[i]
		}
	}
	if ast == nil {
		t.Fatal("expected an ast chunk")
	}
	if !strings.Contains(ast.Text, "func Outer") {
		t.Errorf("ast chunk should contain Outer, got:\n%s", ast.Text)
	}
	if err := assertFullCoverage([]byte(src), chunks); err != nil {
		t.Errorf("full coverage: %v", err)
	}
}

// TestChunkLinesAstFallsBack reports nil for non-code (no function symbols) so
// the caller uses plain line chunking.
func TestChunkLinesAstFallsBack(t *testing.T) {
	if c := ChunkLinesAst([]byte("hello\nworld\n"), nil); c != nil {
		t.Errorf("nil symbols should return nil, got %v", c)
	}
	// Only consts/types (no function-like symbols) → nil.
	syms := []extract.Symbol{{Name: "x", Kind: "var", StartLine: 1, EndLine: 1, Language: "go"}}
	if c := ChunkLinesAst([]byte("x = 1\n"), syms); c != nil {
		t.Errorf("no function symbols should return nil, got %v", c)
	}
}

// assertFullCoverage checks that the chunks' line ranges cover every
// non-blank source line at least once with no overlap — the invariant
// code_read's chunk-reassembly relies on (a gap loses text, e.g. a secret
// const that must still be redacted).
func assertFullCoverage(src []byte, chunks []Chunk) error {
	lines := strings.Split(string(src), "\n")
	covered := make([]bool, len(lines))
	for _, c := range chunks {
		for ln := c.StartLine; ln <= c.EndLine; ln++ {
			if ln < 1 || ln > len(lines) {
				return fmt.Errorf("chunk %d-%d out of range (file has %d lines)", c.StartLine, c.EndLine, len(lines))
			}
			if covered[ln-1] {
				return fmt.Errorf("line %d covered by two chunks", ln)
			}
			covered[ln-1] = true
		}
	}
	for i, c := range covered {
		if c || strings.TrimSpace(lines[i]) == "" {
			continue
		}
		return fmt.Errorf("line %d (non-blank) not covered", i+1)
	}
	return nil
}
