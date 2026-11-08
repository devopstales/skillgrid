package extract

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// coveredLanguages are the languages the one-pass gotreesitter extractors
// (ExtractDefinitionSpans/ExtractCalls/ExtractHeritage/ExtractImports) cover
// directly — the 6 rich languages. All other supported languages go through
// the per-language node maps in languages.go (the map tier).
var coveredLanguages = map[string]bool{
	"go": true, "python": true, "java": true,
	"javascript": true, "typescript": true, "tsx": true,
}

// extractor is the ladder's internal state: the self-healing pool + the
// grammar loader (the test override hook for injecting crashing grammars).
type extractor struct {
	pool    *Pool
	grammar grammarLoaderFn
}

// ExtractFile is the 3-tier extraction ladder (the ExtractFn seam the
// codeindex.Index driver wires into):
//
//  1. rich tier — the 6 covered languages use gotreesitter's one-pass APIs;
//  2. map tier — the data-table languages walk the AST against the node-type
//     map (languages.go);
//  3. regex tier — unmapped extensions, malformed files, and quarantined
//     grammars fall to the regex battery (the deterministic floor).
//
// A panicking grammar quarantines the file to regex and records a death
// (the circuit breaker). Per-file failures never error: the fallback always
// returns a well-formed (possibly empty) FileGraph. A statically-bound call
// is EXTRACTED with a bound target; a name-only call is kept with a null
// target + an unresolved ledger row (CI-INV-3, drop-not-guess).
func ExtractFile(ctx context.Context, path, content, lang string, opts ...ExtractOption) (FileGraph, Tier, error) {
	_ = ctx
	cfg := applyOptions(opts)
	pool := cfg.pool

	// The oversized outcome (T1): a file over the max is not indexed (the
	// driver counts it in files_oversized). Indexed=false.
	if len(content) > cfg.maxFileSize {
		return FileGraph{Oversized: true, Tier: TierRegex}, TierRegex, nil
	}

	// Resolve the language (the caller may pass "" for an unmapped path).
	if lang == "" {
		lang = DetectLanguage(path)
	}
	// Unmapped extension → regex tier (the floor).
	if lang == "" || !LanguageSet[lang] {
		g := Fallback(path, lang, []byte(content))
		g.Tier = TierRegex
		return g, TierRegex, nil
	}

	e := &extractor{pool: pool, grammar: cfg.grammar}

	// Quarantined or breaker-tripped → straight to regex (no tree-sitter
	// attempt — the "a 4th go file goes straight to regex" decision).
	if pool.TierFor(lang, path) == TierRegex {
		g := Fallback(path, lang, []byte(content))
		g.Tier = TierRegex
		g.Error = "grammar quarantined or breaker tripped"
		return g, TierRegex, nil
	}

	// The tier: rich (covered) or map (data table).
	tier := TierMap
	if coveredLanguages[lang] {
		tier = TierRich
	} else if langMapsFor(lang) == nil {
		// Supported but no map and not covered → regex.
		g := Fallback(path, lang, []byte(content))
		g.Tier = TierRegex
		return g, TierRegex, nil
	}

	// Resolve the grammar. No grammar for the language → regex.
	load, ok := e.resolveGrammarFor(lang)
	if !ok {
		g := Fallback(path, lang, []byte(content))
		g.Tier = TierRegex
		return g, TierRegex, nil
	}

	// Parse + extract with panic recovery. A panicking grammar quarantines
	// the file and falls back; the breaker trips on repeated deaths.
	var (
		res      treeResult
		panicked bool
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		res = parseSource(load, []byte(content))
	}()
	if panicked {
		e.recordDeath(lang)
		pool.Quarantine(lang, path)
		g := Fallback(path, lang, []byte(content))
		g.Tier = TierRegex
		g.Error = "grammar panicked; quarantined to regex"
		return g, TierRegex, nil
	}

	if !res.parseOK() {
		g := Fallback(path, lang, []byte(content))
		g.Tier = TierRegex
		g.Error = "primary parse failed"
		return g, TierRegex, nil
	}

	// Map the tree. A mapping panic is also a grammar-level failure.
	var (
		g   FileGraph
		err error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				err = fmt.Errorf("mapping panic: %v", r)
			}
		}()
		g, err = e.mapTree(path, lang, res.tree)
	}()
	if panicked || err != nil || len(g.Symbols) == 0 {
		if err != nil {
			e.recordDeath(lang)
			pool.Quarantine(lang, path)
		}
		fb := Fallback(path, lang, []byte(content))
		fb.Tier = TierRegex
		fb.Error = "no symbols extracted; fell back to regex"
		return fb, TierRegex, nil
	}

	g.Oversized = false
	g.Tier = tier
	// Chunk the file (AST-first; the driver stores g.Chunks).
	g.Chunks = ChunkFile(g, content)
	return g, tier, nil
}

// recordDeath is the extractor's pool delegate (the death-recording hook the
// panic path calls).
func (e *extractor) recordDeath(lang string) { e.pool.RecordDeath(lang) }

// resolveGrammarFor returns a lazy loader for the gotreesitter language named
// langName. The grammar field (override hook) is used by tests to inject
// crashes; the production path uses resolveGrammar.
func (e *extractor) resolveGrammarFor(langName string) (func() any, bool) {
	if e.grammar != nil {
		return e.grammar(langName)
	}
	return resolveGrammar(langName)
}

// mapTree converts a parsed tree into a FileGraph using the one-pass
// extractors for covered languages and the per-language node maps otherwise.
// It stamps from-IDs on call/heritage edges (the enclosing definition),
// resolves statically-bound targets to a bound ToID, and keeps name-only
// targets null + emits an unresolved ledger row (CI-INV-3).
func (e *extractor) mapTree(path, lang string, tree *ts.Tree) (FileGraph, error) {
	src := tree.Source()
	g := FileGraph{Tier: TierRich}
	if !coveredLanguages[lang] {
		g.Tier = TierMap
	}

	type defRec struct {
		name, kind, uid, hash string
		start, end            int
		sig                   string
	}
	type edgeRec struct {
		kind    string
		fromID  int64
		toID    *int64
		toName  string
		conf    string
		line    int
		context string
	}
	var defs []defRec
	var edges []edgeRec
	var unresolved []UnresolvedRef

	// symbolID maps a name → in-file symbol id (best-effort edge targeting;
	// cross-file resolution is the graph package's job, not the extractor's).
	idByName := map[string]int64{}
	nextID := int64(1)

	addDef := func(name, kind, sig string, start, end int, spanText []byte) int64 {
		id := nextID
		nextID++
		uid := symbolUID(name, kind, start, end)
		if _, exists := idByName[name]; !exists {
			idByName[name] = id
		}
		defs = append(defs, defRec{
			name: name, kind: kind, uid: uid,
			start: start, end: end, sig: sig,
			hash: contentHash(spanText),
		})
		return id
	}

	addCall := func(target, kind string, line int, fromID int64) {
		if target == "" {
			return
		}
		// A statically-bound target (in-file) → EXTRACTED + bound ToID. A
		// name-only target (not in-file) → to_id NULL + an unresolved row
		// (CI-INV-3, drop-not-guess).
		var toID *int64
		if id, ok := idByName[target]; ok {
			toID = &id
		} else if kind == "calls" {
			// name-only call: keep the edge (null target) + ledger row.
			unresolved = append(unresolved, UnresolvedRef{
				Reference: target, Kind: kind, Line: line,
			})
		}
		edges = append(edges, edgeRec{
			kind: kind, fromID: fromID, toID: toID, toName: target,
			conf: ConfidenceExtracted, line: line,
		})
	}

	addImport := func(target, kind string, line int) {
		if target == "" {
			return
		}
		edges = append(edges, edgeRec{
			kind: kind, fromID: 0, toID: nil, toName: target,
			conf: ConfidenceExtracted, line: line,
		})
	}

	switch {
	case coveredLanguages[lang]:
		for _, span := range ts.ExtractDefinitionSpans(tree) {
			if span.Name == "" {
				continue
			}
			start := lineOf(src, span.StartByte)
			end := lineOf(src, span.EndByte)
			addDef(span.Name, span.Kind, signatureOf(src, span.StartByte, span.EndByte), start, end, src[span.StartByte:span.EndByte])
		}
		for _, c := range ts.ExtractCalls(tree) {
			if c.Name == "" {
				continue
			}
			line := lineOf(src, c.StartByte)
			addCall(c.Name, "calls", line, 0)
		}
		for _, h := range ts.ExtractHeritage(tree) {
			if h.Name == "" || h.Parent == "" {
				continue
			}
			addCall(h.Parent, "extends", lineOf(src, h.StartByte), 0)
		}
		for _, imp := range ts.ExtractImports(tree) {
			target := imp.Path
			if target == "" {
				target = imp.From
			}
			if target == "" {
				continue
			}
			addImport(target, "imports", lineOf(src, imp.StartByte))
		}

	default:
		maps := langMapsFor(lang)
		if maps == nil {
			return g, fmt.Errorf("no node maps for language %q", lang)
		}
		walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
			typ := n.Type()
			if di, ok := maps.defNodes[typ]; ok {
				name := ""
				if di.nameFrom != nil {
					name = di.nameFrom(n, src)
				}
				if name == "" {
					return true
				}
				start := lineOf(src, n.StartByte())
				end := lineOf(src, n.EndByte())
				addDef(name, di.kind, signatureOf(src, n.StartByte(), n.EndByte()), start, end, src[n.StartByte():n.EndByte()])
			}
			if ci, ok := maps.callNodes[typ]; ok {
				if ci.targetFrom == nil {
					return true
				}
				if target := ci.targetFrom(n, src); target != "" {
					addCall(target, "calls", lineOf(src, n.StartByte()), 0)
				}
			}
			return true
		})
		if maps.imports != nil {
			for _, imp := range maps.imports(tree) {
				addImport(imp.path, "imports", imp.line)
			}
		}
		if maps.heritage != nil {
			for _, h := range maps.heritage(tree) {
				addCall(h.parent, h.kind, h.line, 0)
			}
		}
	}

	// Build the symbols (id 1..N, in def order).
	for i, d := range defs {
		g.Symbols = append(g.Symbols, Symbol{
			Name:          d.name,
			QualifiedName: d.name,
			Kind:          d.kind,
			Language:      lang,
			Signature:     d.sig,
			StartLine:     d.start,
			EndLine:       d.end,
			ContentHash:   d.hash,
			UID:           d.uid,
			Visibility:    visibilityOf(d.name, lang),
			IsExported:    isExported(d.name, lang),
		})
		// Re-map the name → actual symbol index (i+1) for edge targeting.
		idByName[d.name] = int64(i + 1)
	}

	// Resolve edge targets to bound ToIDs (the symbols are built now, so the
	// name → id map is final). A statically-bound call → bound ToID +
	// EXTRACTED; a name-only call → null ToID (already flagged unresolved).
	for i := range edges {
		ed := &edges[i]
		if ed.kind == "imports" {
			continue
		}
		// A statically-bound target (in-file) → bound ToID; a name-only
		// target stays null (already flagged unresolved, CI-INV-3). from_id
		// stays 0 — the graph step resolves the enclosing definition.
		if ed.toID == nil {
			if id, ok := idByName[ed.toName]; ok {
				ed.toID = &id
			}
		}
	}

	for _, ed := range edges {
		g.Edges = append(g.Edges, Edge{
			Kind:       ed.kind,
			FromID:     ed.fromID,
			ToID:       ed.toID,
			ToName:     ed.toName,
			Confidence: ed.conf,
			Line:       ed.line,
			Context:    ed.context,
		})
	}
	g.Unresolved = unresolved

	sort.SliceStable(g.Symbols, func(i, j int) bool { return g.Symbols[i].StartLine < g.Symbols[j].StartLine })
	sort.SliceStable(g.Edges, func(i, j int) bool { return g.Edges[i].Line < g.Edges[j].Line })
	return g, nil
}

// visibilityOf classifies a top-level symbol's visibility from its name. Only
// Go has a name-based export rule (leading uppercase); the rest default to
// unexported (the conservative reading).
func visibilityOf(name, lang string) string {
	if name == "" {
		return ""
	}
	if lang == "go" {
		if name[0] >= 'A' && name[0] <= 'Z' {
			return "export"
		}
		return "private"
	}
	return "unexported"
}

func isExported(name, lang string) bool {
	return visibilityOf(name, lang) == "export"
}

// symbolUID derives a stable, path-independent identity for a symbol from its
// name, kind, and span. Content is excluded so a moved symbol keeps its
// identity across files (edges re-target by UID).
func symbolUID(name, kind string, startLine, endLine int) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s\x00%s\x00%d:%d", name, kind, startLine, endLine)
	return fmt.Sprintf("%016x", h.Sum64())
}

// contentHash hashes a symbol's source span for re-extraction detection.
func contentHash(text []byte) string {
	h := fnv.New64a()
	_, _ = h.Write(text)
	return fmt.Sprintf("%016x", h.Sum64())
}

// lineOf converts a byte offset to a 1-based line number.
func lineOf(src []byte, offset uint32) int {
	if int(offset) >= len(src) {
		offset = uint32(len(src))
	}
	n := 1
	for i := uint32(0); i < offset; i++ {
		if src[i] == '\n' {
			n++
		}
	}
	return n
}

// signatureOf extracts the declaration text of a symbol span, trimmed to a
// single line for display (≤200 chars).
func signatureOf(src []byte, start, end uint32) string {
	if start > end || int(end) > len(src) {
		return ""
	}
	text := string(src[start:end])
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		text = text[:idx]
	}
	text = strings.TrimSpace(text)
	if len(text) > 200 {
		text = text[:200]
	}
	return text
}
