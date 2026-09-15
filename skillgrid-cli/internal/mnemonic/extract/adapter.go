package extract

import (
	"fmt"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// coveredLanguages are the languages the one-pass gotreesitter extractors
// (ExtractDefinitionSpans/ExtractCalls/ExtractHeritage/ExtractImports) cover
// directly. All other supported languages go through the per-language node
// maps in languages.go.
var coveredLanguages = map[string]bool{
	"go": true, "python": true, "java": true,
	"javascript": true, "typescript": true, "tsx": true,
}

// ExtractFile implements the Extractor interface: it resolves the language,
// parses with the self-healing pool, and maps the AST to a FileGraph. Per-file
// failures quarantine to the regex fallback and never error.
func (e *extractor) ExtractFile(path string, src []byte) (*FileGraph, error) {
	lang := DetectLanguage(path)
	if lang == "" {
		return Fallback(path, src), nil
	}

	// Already quarantined or breaker tripped: go straight to regex.
	if e.pool.IsQuarantined(lang, path) || e.pool.IsBreakerTripped(lang) {
		g := Fallback(path, src)
		g.Error = "grammar quarantined"
		return g, nil
	}

	load, ok := e.resolveGrammarFor(lang)
	if !ok {
		return Fallback(path, src), nil
	}

	// Parse + extract with panic recovery. A panicking grammar quarantines
	// the file and falls back; the worker respawns up to a bound and the
	// breaker trips on repeated deaths.
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
		res = parseSource(load, src)
	}()
	if panicked {
		e.pool.recordDeath(lang)
		e.pool.Quarantine(lang, path)
		g := Fallback(path, src)
		g.Error = "grammar panicked; quarantined to regex"
		return g, nil
	}

	if !res.parseOK() {
		// Malformed / early-stopped parse: fall back.
		g := Fallback(path, src)
		g.Error = "primary parse failed"
		return g, nil
	}

	tree := res.tree
	treeLang := res.lang
	graph, err := e.mapTree(path, treeLang, tree)
	if err != nil || graph == nil || len(graph.Symbols) == 0 {
		if err != nil {
			// A mapping panic is also a grammar-level failure.
			e.pool.recordDeath(lang)
			e.pool.Quarantine(lang, path)
		}
		g := Fallback(path, src)
		if err != nil {
			g.Error = "mapping failed: " + err.Error()
		} else {
			g.Error = "no symbols extracted; fell back to regex"
		}
		return g, nil
	}
	return graph, nil
}

// dynImportModule extracts the module path argument of an import() call whose
// call expression starts at startByte: it reads the `import` keyword, skips the
// opening paren, and returns the text up to the closing paren. Returns "" when
// the path cannot be determined (the caller then falls back to a keyword target).
func dynImportModule(src []byte, startByte uint32) string {
	i := startByte
	for i < uint32(len(src)) && src[i] != '(' {
		i++
	}
	if i >= uint32(len(src)) {
		return ""
	}
	i++ // skip '('
	if i >= uint32(len(src)) {
		return ""
	}
	j := i
	for j < uint32(len(src)) && src[j] != ')' {
		j++
	}
	if j >= uint32(len(src)) {
		return ""
	}
	return string(src[i:j])
}

// mapTree converts a parsed tree into a FileGraph using the one-pass
// extractors for covered languages and the per-language node maps otherwise.
func (e *extractor) mapTree(path, lang string, tree *ts.Tree) (*FileGraph, error) {
	src := tree.Source()
	g := &FileGraph{
		Path:      path,
		Language:  lang,
		Extractor: "treesitter",
	}

	type defRec struct {
		name      string
		qualified string
		kind      string
		sig       string
		start     int
		end       int
		uid       string
		hash      string
	}
	type edgeRec struct {
		kind    string
		fromUID string
		toUID   string
		toName  string
		target  string
		conf    string
		line    int
	}
	var defs []defRec
	var edges []edgeRec

	// symbolUIDs maps a name -> uid for in-file edge targeting (best-effort;
	// cross-file resolution is the graph package's job in step 03).
	uidByName := map[string]string{}
	kindByName := map[string]string{}

	addDef := func(name, kind, sig string, start, end int, spanText []byte) string {
		uid := symbolUID(name, kind, start, end)
		_ = spanText
		if _, exists := uidByName[name]; !exists {
			uidByName[name] = uid
			kindByName[name] = kind
		}
		defs = append(defs, defRec{
			name:      name,
			qualified: name,
			kind:      kind,
			sig:       sig,
			start:     start,
			end:       end,
			uid:       uid,
			hash:      contentHash(spanText),
		})
		return uid
	}

	addCall := func(target, kind string, line int, conf string) {
		if target == "" {
			return
		}
		var toUID string
		if u, ok := uidByName[target]; ok {
			toUID = u
		}
		edges = append(edges, edgeRec{
			kind:   kind,
			toUID:  toUID,
			toName: target,
			conf:   conf,
			line:   line,
		})
	}

	addHeritage := func(child, parent string, line int) {
		if child == "" || parent == "" {
			return
		}
		var toUID string
		if u, ok := uidByName[parent]; ok {
			toUID = u
		}
		edges = append(edges, edgeRec{
			kind:   "extends",
			toUID:  toUID,
			toName: parent,
			conf:   ConfidenceExtracted,
			line:   line,
		})
	}

	addImport := func(target, kind string, line int, conf string) {
		if target == "" {
			return
		}
		edges = append(edges, edgeRec{
			kind:   kind,
			toName: target,
			target: target,
			conf:   conf,
			line:   line,
		})
	}

	switch {
	case coveredLanguages[lang]:
		for _, span := range ts.ExtractDefinitionSpans(tree) {
			name := span.Name
			if name == "" {
				continue
			}
			start := lineOf(src, span.StartByte)
			end := lineOf(src, span.EndByte)
			spanText := src[span.StartByte:span.EndByte]
			addDef(name, span.Kind, signatureOf(src, span.StartByte, span.EndByte), start, end, spanText)
		}
		for _, c := range ts.ExtractCalls(tree) {
			if c.Name == "" {
				continue
			}
			conf := ConfidenceExtracted
			if c.Receiver != "" {
				// Member call: the target is name-only against a receiver we
				// cannot statically bind here — keep EXTRACTED (it is AST-
				// derived) but the graph step will refine.
				_ = c.Receiver
			}
			// A bare import() call in JS/TS is a dynamic module load, not a
			// plain call: the gotreesitter lib surfaces it as a call_expression
			// whose callee is the `import` keyword. Emit a dynamic_import edge
			// whose target is the module path (import("./mod") -> "./mod"); the
			// callee `import` is the keyword, not a module.
			if c.Name == "import" && (lang == "javascript" || lang == "typescript" || lang == "tsx") {
				addImport(strings.Trim(dynImportModule(src, c.StartByte), "\"'`"), "dynamic_import", lineOf(src, c.StartByte), conf)
				continue
			}
			addCall(c.Name, "calls", lineOf(src, c.StartByte), conf)
		}
		for _, h := range ts.ExtractHeritage(tree) {
			if h.Name == "" || h.Parent == "" {
				continue
			}
			addHeritage(h.Name, h.Parent, lineOf(src, h.StartByte))
		}
		for _, imp := range ts.ExtractImports(tree) {
			target := imp.Path
			if target == "" {
				target = imp.From
			}
			if target == "" {
				continue
			}
			// import statements are static in this grammar; dynamic module
			// loads arrive as import() calls (handled in the call loop above).
			addImport(target, "imports", lineOf(src, imp.StartByte), ConfidenceExtracted)
		}

	default:
		maps := langMapsFor(lang)
		if maps == nil {
			return nil, fmt.Errorf("no node maps for language %q", lang)
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
				target := ci.targetFrom(n, src)
				if target != "" {
					addCall(target, "calls", lineOf(src, n.StartByte()), ConfidenceExtracted)
				}
			}
			return true
		})
		if maps.imports != nil {
			for _, imp := range maps.imports(tree) {
				addImport(imp.path, "imports", imp.line, imp.confid)
			}
		}
		if maps.heritage != nil {
			for _, h := range maps.heritage(tree) {
				// Heritage refs from the maps carry a parent but not the
				// child name; resolve the nearest enclosing definition.
				child, _ := ts.EnclosingDefinition(tree, uint32(0))
				_ = child
				addHeritage(h.name, h.parent, h.line)
			}
		}
	}

	// Stamp from-UIDs on call/heritage edges using the enclosing definition.
	for i := range edges {
		ed := &edges[i]
		if ed.kind == "imports" {
			continue
		}
		// Find the definition whose span contains the edge line.
		var bestUID string
		bestEnd := -1
		for _, d := range defs {
			if ed.line >= d.start && ed.line <= d.end && d.end > bestEnd {
				bestUID = d.uid
				bestEnd = d.end
			}
		}
		ed.fromUID = bestUID
		if ed.fromUID == "" && len(defs) > 0 {
			// No enclosing def (e.g. top-level call): leave from empty so
			// the graph step can resolve by name.
			continue
		}
	}

	for _, d := range defs {
		g.Symbols = append(g.Symbols, Symbol{
			Name:          d.name,
			QualifiedName: d.qualified,
			Kind:          d.kind,
			Language:      lang,
			Signature:     d.sig,
			StartLine:     d.start,
			EndLine:       d.end,
			ContentHash:   d.hash,
			UID:           d.uid,
		})
	}
	for _, ed := range edges {
		g.Edges = append(g.Edges, Edge{
			Kind:       ed.kind,
			FromUID:    ed.fromUID,
			ToUID:      ed.toUID,
			ToName:     ed.toName,
			TargetPath: ed.target,
			Confidence: ed.conf,
			Line:       ed.line,
		})
	}

	sort.SliceStable(g.Symbols, func(i, j int) bool { return g.Symbols[i].StartLine < g.Symbols[j].StartLine })
	sort.SliceStable(g.Edges, func(i, j int) bool { return g.Edges[i].Line < g.Edges[j].Line })

	// Rationale: scan comments for NOTE/WHY/ADR markers and link each to the
	// nearest enclosing symbol (language-agnostic, comment-text based).
	g.Rationales = ExtractRationale(src, g.Symbols)
	return g, nil
}
