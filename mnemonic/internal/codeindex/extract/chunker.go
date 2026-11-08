package extract

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// The chunker constants (spec 001 §3 — the chunker constants the stamp's
// pinned input set hashes): a ~1000-char target keeps a chunk coherent AND
// within an encoder window; a symbol below minChunkChars merges with its
// neighbor (no degenerate one-line embeddings); gaps between symbols become
// line segments; an over-long symbol splits into chunkWindowLines/chunkOverlap
// windows.
const (
	semanticChunkTargetChars = 1000 // ≤1000 chars per ast chunk
	minChunkChars            = 250  // 250-char merge threshold
	chunkWindowLines         = 80   // the line-window size
	chunkOverlapLines        = 10   // the line-window overlap
)

// functionKinds are the symbol kinds a chunk is worth building around.
// Constants/variables/types are too small to embed meaningfully on their own;
// functions/classes/methods carry the semantic payload.
var functionKinds = map[string]bool{
	"function": true, "method": true, "class": true,
}

// Chunk is the AST-first chunker (spec 001 §3): one ast chunk per
// function-like symbol whose span fits the char target; symbols over the
// target split into 80/10 line windows; gaps between symbols become line
// segments; a symbol-less file (config, non-code) falls to the plain line
// fallback. It is the v1 ChunkLinesAst shape, adapted to the v2 Chunk type.
func ChunkFile(graph FileGraph, content string) []Chunk {
	syms := graph.Symbols
	if len(syms) == 0 {
		// No function-like symbols → the plain line fallback (config files,
		// non-code, or a file of only consts/types).
		return chunkLines([]byte(content), chunkWindowLines, chunkOverlapLines)
	}
	lines := strings.Split(content, "\n")

	type span struct {
		start, end int // 1-based, inclusive
		kind       string
	}
	var spans []span
	for _, s := range syms {
		if s.EndLine < s.StartLine || !functionKinds[s.Kind] {
			continue
		}
		spans = append(spans, span{start: s.StartLine, end: s.EndLine, kind: s.Kind})
	}
	if len(spans) == 0 {
		// No function-like symbols among the extracted set → line fallback.
		return chunkLines([]byte(content), chunkWindowLines, chunkOverlapLines)
	}
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	spanText := func(a, b int) string {
		if a < 1 {
			a = 1
		}
		if b > len(lines) {
			b = len(lines)
		}
		if a > b {
			return ""
		}
		return strings.Join(lines[a-1:b], "\n")
	}
	charLen := func(a, b int) int { return len(spanText(a, b)) }

	// Build a FULL set of non-overlapping line segments covering the whole
	// file: each function span is one ast segment; lines BETWEEN spans (package
	// clause, top-level consts, blank lines) are line segments. Covering
	// everything keeps code_read's chunk reassembly complete.
	type seg struct {
		start, end int
		kind       string
	}
	var segs []seg
	prevEnd := 0
	for _, s := range spans {
		if s.start > prevEnd+1 && strings.TrimSpace(spanText(prevEnd+1, s.start-1)) != "" {
			segs = append(segs, seg{start: prevEnd + 1, end: s.start - 1, kind: "lines"})
		}
		segs = append(segs, seg{start: s.start, end: s.end, kind: "ast"})
		prevEnd = s.end
	}
	if prevEnd < len(lines) && strings.TrimSpace(spanText(prevEnd+1, len(lines))) != "" {
		segs = append(segs, seg{start: prevEnd + 1, end: len(lines), kind: "lines"})
	}

	// Merge a tiny leading ast segment (under minChunkChars) into the next
	// segment so it doesn't form a degenerate one-line embedding.
	if n := len(segs); n >= 2 && segs[0].kind == "ast" && charLen(segs[0].start, segs[0].end) < minChunkChars {
		segs[1].start = segs[0].start
		segs = segs[1:]
	}

	relabel := func(cs []Chunk, offset int) []Chunk {
		for i := range cs {
			cs[i].StartLine += offset
			cs[i].EndLine += offset
		}
		return cs
	}

	var chunks []Chunk
	for _, r := range segs {
		text := spanText(r.start, r.end)
		if strings.TrimSpace(text) == "" {
			continue
		}
		if r.kind == "ast" && len(text) <= semanticChunkTargetChars {
			sum := sha256.Sum256([]byte(text))
			chunks = append(chunks, Chunk{
				Kind:        "ast",
				StartLine:   r.start,
				EndLine:     r.end,
				Text:        text,
				ContentHash: hex.EncodeToString(sum[:]),
			})
			continue
		}
		// A line segment, or an ast span over the target (one large function):
		// line-window it, offsetting the sub-chunk line numbers back to whole-
		// file coordinates.
		chunks = append(chunks, relabel(chunkLines([]byte(text), chunkWindowLines, chunkOverlapLines), r.start-1)...)
	}
	if len(chunks) == 0 {
		return nil
	}
	return chunks
}

// chunkLines is the plain line-window fallback (the v1 ChunkLines shape):
// fixed chunkWindowLines windows with chunkOverlapLines overlap, 1-based line
// numbers relative to its own input (the caller offsets back to file coords).
func chunkLines(content []byte, chunkLines, chunkOverlap int) []Chunk {
	if chunkLines <= 0 {
		chunkLines = chunkWindowLines
	}
	if chunkOverlap < 0 {
		chunkOverlap = 0
	}
	text := string(content)
	allLines := strings.Split(text, "\n")
	if len(allLines) == 0 {
		return nil
	}
	step := chunkLines - chunkOverlap
	if step <= 0 {
		step = chunkLines
	}
	var chunks []Chunk
	for start := 0; start < len(allLines); start += step {
		end := start + chunkLines
		if end > len(allLines) {
			end = len(allLines)
		}
		chunkText := strings.Join(allLines[start:end], "\n")
		if strings.TrimSpace(chunkText) == "" {
			if end >= len(allLines) {
				break
			}
			continue
		}
		sum := sha256.Sum256([]byte(chunkText))
		chunks = append(chunks, Chunk{
			Kind:        "lines",
			StartLine:   start + 1,
			EndLine:     end,
			Text:        chunkText,
			ContentHash: hex.EncodeToString(sum[:]),
		})
		if end >= len(allLines) {
			break
		}
	}
	return chunks
}
