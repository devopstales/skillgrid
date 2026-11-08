package handler

// explore_format.go — the code_explore response renderer (TICKET-05): a
// pure function over the typed ExploreResult, the CodeGraph baseline
// layout (the captured reference) plus the v2 provenance (EXTRACTED
// untagged / INFERRED tagged, per-hit fusion-leg ranks, the degraded
// line). No I/O here — the handler assembles the result (the search
// answer, the disk reads, the graph labels); the renderer formats it.

import (
	"fmt"
	"sort"
	"strings"
)

// ExploreRadiusLine is one blast-radius bullet (the per-symbol impact
// line: the symbol, its location, the caller count + files, the
// test-file verdict, and the fusion-leg provenance that surfaced the
// hit — "leg:rank" entries, pre-sorted).
type ExploreRadiusLine struct {
	Name        string
	Path        string
	Line        int
	Callers     int
	CallerFiles []string
	TestFiles   []string
	Provenance  []string
}

// ExploreEdge is one outgoing edge of a shown file (the edge-summary
// line: toName + kind). EXTRACTED edges are untagged; INFERRED edges
// render tagged (inferred).
type ExploreEdge struct {
	ToName   string
	Kind     string
	Inferred bool
}

// SourceSpan is one contiguous numbered range of a file block (read via
// the existing read path — the live source, secret redaction). The
// renderer numbers the lines and elides the gaps between spans.
type SourceSpan struct {
	Start int // the first line number (1-based)
	Text  string
}

// ExploreFileBlock is one per-file source block (the header, the
// edge-summary line, the language tag, the spans).
type ExploreFileBlock struct {
	Path     string
	Language string
	Spans    []SourceSpan
	Edges    []ExploreEdge
}

// ExploreResult is the typed input to RenderExplore (TICKET-05): the
// ranked query, the blast-radius lines (one per resolved symbol), the
// per-file source blocks (rank order), and the degraded flags
// (no_vector excluded — an absent embedder is a deployment state, not a
// degraded leg of this call).
type ExploreResult struct {
	Query    string
	Symbols  int
	Files    int
	Radius   []ExploreRadiusLine
	Blocks   []ExploreFileBlock
	Degraded string
}

// RenderExplore renders the CodeGraph baseline layout: the Exploration
// header, the symbol/file counts, the degraded line (iff a leg
// degraded), the blast-radius section (iff any radius lines), and the
// Source Code section (the provenance note + one line-numbered verbatim
// block per file, gaps elided). Empty result: header + the 0/0 counts
// line, nothing else.
func RenderExplore(r ExploreResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**Exploration: %s**\n\n", r.Query)
	fmt.Fprintf(&b, "Found %d symbols across %d files.\n", r.Symbols, r.Files)
	if r.Degraded != "" {
		fmt.Fprintf(&b, "degraded: %s\n", r.Degraded)
	}
	if len(r.Radius) > 0 {
		b.WriteString("\n**Blast radius — what depends on these (update/verify before editing)**\n\n")
		for _, l := range r.Radius {
			fmt.Fprintf(&b, "- `%s` (%s:%d) — ", l.Name, l.Path, l.Line)
			switch {
			case l.Callers == 0:
				b.WriteString("no callers")
			case len(l.CallerFiles) == 0:
				fmt.Fprintf(&b, "%d caller%s", l.Callers, callerPlural(l.Callers))
			default:
				fmt.Fprintf(&b, "%d caller%s in %s", l.Callers, callerPlural(l.Callers), backtickedList(l.CallerFiles))
			}
			b.WriteString("; ")
			if len(l.TestFiles) > 0 {
				fmt.Fprintf(&b, "tests: %s", backtickedList(l.TestFiles))
			} else {
				b.WriteString("no tests found among callers")
			}
			if len(l.Provenance) > 0 {
				fmt.Fprintf(&b, " [%s]", strings.Join(l.Provenance, ", "))
			}
			b.WriteString("\n")
		}
	}
	if len(r.Blocks) > 0 {
		b.WriteString("\n**Source Code**\n\n")
		b.WriteString("> The code below is the **verbatim, current on-disk source** of these files — re-read from disk on this call and line-numbered, byte-for-byte identical to what the Read tool returns. It is NOT a summary, outline, or stale cache. Treat each block as a Read you have already performed: do not Read a file shown here.\n")
		for _, blk := range r.Blocks {
			b.WriteString("\n")
			fmt.Fprintf(&b, "**`%s`**", blk.Path)
			if len(blk.Edges) > 0 {
				fmt.Fprintf(&b, " — %s", edgeSummary(blk.Edges))
			}
			b.WriteString("\n\n")
			fmt.Fprintf(&b, "```%s\n", blockLanguage(blk.Language))
			for si, s := range blk.Spans {
				if si > 0 {
					b.WriteString("... (gap) ...\n")
				}
				for i, ln := range strings.Split(s.Text, "\n") {
					fmt.Fprintf(&b, "%d\t%s\n", s.Start+i, ln)
				}
			}
			b.WriteString("```\n")
		}
	}
	return b.String()
}

// edgeSummary is the per-file edge-summary line: the first 6 distinct
// edges as toName(kind) — INFERRED tagged (inferred) — then "+N more".
// (The dedup is the handler's; this formats what it is given.)
func edgeSummary(edges []ExploreEdge) string {
	shown := edges
	more := 0
	if len(edges) > 6 {
		shown, more = edges[:6], len(edges)-6
	}
	parts := make([]string, 0, len(shown)+1)
	for _, e := range shown {
		parts = append(parts, fmt.Sprintf("%s(%s)", e.ToName, e.Kind))
		if e.Inferred {
			parts[len(parts)-1] += " (inferred)"
		}
	}
	if more > 0 {
		parts = append(parts, fmt.Sprintf("+%d more", more))
	}
	return strings.Join(parts, ", ")
}

// callerPlural is the bullet's caller noun (1 caller / N callers).
func callerPlural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// backtickedList renders a single path backticked, or a comma-joined
// backticked list.
func backtickedList(paths []string) string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = "`" + p + "`"
	}
	return strings.Join(out, ", ")
}

// blockLanguage is the source fence's language tag (the file's indexed
// language; text when unknown).
func blockLanguage(lang string) string {
	if lang == "" {
		return "text"
	}
	return lang
}

// provenanceSorted renders Hit.Provenance (leg → 1-based rank) as
// sorted "leg:rank" entries (the per-hit fusion-leg provenance).
func provenanceSorted(p map[string]int) []string {
	if len(p) == 0 {
		return nil
	}
	out := make([]string, 0, len(p))
	for leg, rank := range p {
		out = append(out, fmt.Sprintf("%s:%d", leg, rank))
	}
	sort.Strings(out)
	return out
}
