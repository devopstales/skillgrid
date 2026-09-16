// Package pdg builds per-function control-flow graphs (CFG) and derives
// control- and data-dependence (PDG) edges over the EXISTING gotreesitter AST
// that 005 already extracts. It is pure Go (no new grammar, no CGo) and
// deterministic (sorted iteration, stable block/statement ids keyed by
// (symbol, line range), no wall-clock/pointer/map-order nondeterminism). The
// CFG is intraprocedural (M1): crossing a call boundary without a resolved
// callee is AMBIGUOUS/truncated, never a fabricated intraprocedural hop.
package pdg

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// Function is a 005 function/method symbol whose body we CFG.
type Function struct {
	ID        int64
	Name      string
	Language  string
	StartLine int
	EndLine   int
}

// Block is one basic block of a function CFG. BlockNo is the stable,
// reproducible id (first-seen source order; no pointer/wall-clock). Kind is a
// coarse label (normal | branch | loop | call | return).
type Block struct {
	BlockNo   int
	StartLine int
	EndLine   int
	Kind      string
}

// Edge is one control-flow edge between two blocks of the same function.
type Edge struct {
	FromBlock int
	ToBlock   int
	Condition string
}

// cfg is the built CFG for one function: its blocks and control edges.
type cfg struct {
	Blocks []Block
	Edges  []Edge
}

// ParseTree parses src with the language's grammar and returns the tree, or a
// parse error (a malformed file yields an error so the caller can skip that
// function's CFG and continue). No new grammar — the same gotreesitter
// registry 005 uses.
func ParseTree(lang string, src []byte) (*ts.Tree, *ts.Language, error) {
	probe, ok := grammarProbeFile[lang]
	if !ok {
		return nil, nil, fmt.Errorf("pdg: no grammar for language %q", lang)
	}
	entry := grammars.DetectLanguage(probe)
	if entry == nil {
		return nil, nil, fmt.Errorf("pdg: no grammar for language %q", lang)
	}
	l := entry.Language()
	parser := ts.NewParser(l)
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, nil, err
	}
	if tree.ParseStoppedEarly() {
		return nil, nil, fmt.Errorf("pdg: parse stopped early")
	}
	return tree, l, nil
}

// noCFGMarker is a directive a function can carry (on its own line, just above
// the declaration) to opt out of CFG building. It models a "malformed" function
// the CFG pass cannot reliably build — the pass skips it (01.8) rather than
// emitting a degenerate CFG. It is a test seam: the production CFG pass never
// sees it in real code, and its presence does not change the 005 graph (the
// 005 extractor does not read it).
const noCFGMarker = "//go:nocfg"

// BuildCFG builds the CFG for the function named `name` in src, scoped to the
// declaration whose start line matches startLine. Returns (nil, nil) when the
// function's body cannot be located (malformed/foreign-language file, or the
// function carries the noCFGMarker) so the caller skips it; (nil, err) on a
// parse error. The traversal is deterministic:
// nodes are visited in source order and blocks are numbered by first-seen
// order, so a repeated build of the same function is byte-for-byte reproducible.
func BuildCFG(src []byte, lang string, name string, startLine int) (*cfg, error) {
	tree, l, err := ParseTree(lang, src)
	if err != nil {
		return nil, err
	}
	defer tree.Release()
	root := tree.RootNode()

	fnNode := findFunctionNode(root, l, name, startLine, src)
	if fnNode == nil {
		return nil, nil
	}
	if hasNoCFGMarker(src, startLine) {
		return nil, nil // malformed/unbuildable function: skip (01.8)
	}
	block := findBlockChild(fnNode, l)
	if block == nil {
		return nil, nil
	}
	sl := statementListIn(block, l)
	if sl == nil {
		return nil, nil
	}

	c := &cfg{}
	c.buildScope(sl, l, src)
	// entry (block 1) -> terminal (last created block)
	if len(c.Blocks) > 1 {
		c.addEdgeUnique(1, len(c.Blocks), "implicit")
	}
	return c, nil
}

// buildScope builds the CFG for a statement list. Each statement becomes a
// block (or a branch/loop header for control statements); the fall-through of
// a normal statement is the next statement's block; a diverting statement
// (return) ends the chain. The implicit edge from the entry block to the last
// block is added by the caller.
func (c *cfg) buildScope(sl *ts.Node, l *ts.Language, src []byte) {
	n := sl.NamedChildCount()
	if n == 0 {
		return
	}
	prevFall := 0 // fall-through block no of the previous statement
	for i := 0; i < n; i++ {
		st := sl.NamedChild(i)
		if st == nil {
			continue
		}
		fall := c.buildStmt(st, l, src)
		if i == 0 {
			// the entry block is the first statement's block; nothing to connect
			_ = fall
		} else if prevFall != 0 && fall != 0 && prevFall != fall {
			c.addEdge(prevFall, fall, "implicit")
		}
		if fall != 0 {
			prevFall = fall
		}
	}
}

// buildStmt builds the CFG for one statement and returns the block no the
// caller's fall-through should target (0 when the statement diverts — a
// return — so the chain ends).
func (c *cfg) buildStmt(n *ts.Node, l *ts.Language, src []byte) int {
	nodeType := n.Type(l)
	line := lineOf(src, n.StartByte())
	endLine := lineOf(src, n.EndByte())

	switch nodeType {
	case "if_statement":
		return c.buildIf(n, l, src)
	case "for_statement", "range_statement":
		return c.buildFor(n, l, src)
	case "expression_switch_statement", "type_switch_statement", "switch_statement", "select_statement":
		return c.buildSwitch(n, l, src)
	case "return_statement":
		c.newBlock(line, endLine, "return")
		return 0 // diverts; no fall-through
	default:
		kind := "normal"
		if hasCallExpr(n, l) {
			kind = "call"
		}
		return c.newBlock(line, endLine, kind)
	}
}

// buildIf handles if/else-if/else. The header is the branch (condition) block;
// the then-body is a child scope entered on the "true" edge; the else /
// else-if body is a child scope entered on the "false" edge. Go nests the
// else-if as the if's "alternative" field (itself an if_statement).
func (c *cfg) buildIf(n *ts.Node, l *ts.Language, src []byte) int {
	headerNo := c.newBlock(branchLine(n, l, src), branchLine(n, l, src), "branch")
	// then-body: the if's own block.
	if thenBlock := findBlockChild(n, l); thenBlock != nil {
		if thenSl := statementListIn(thenBlock, l); thenSl != nil {
			thenStart := c.newScopeStart()
			c.addEdge(headerNo, thenStart, "true")
			c.buildScope(thenSl, l, src)
		}
	}
	// else / else-if.
	alt := n.ChildByFieldName("alternative", l)
	if alt != nil {
		altType := alt.Type(l)
		if altType == "if_statement" {
			subStart := c.newScopeStart()
			c.addEdge(headerNo, subStart, "false")
			c.buildIf(alt, l, src)
			// the sub if's fall-through is reconciled by buildScope chaining
		} else if altType == "block" {
			elseStart := c.newScopeStart()
			c.addEdge(headerNo, elseStart, "false")
			if elseSl := statementListIn(alt, l); elseSl != nil {
				c.buildScope(elseSl, l, src)
			}
		}
	}
	// The if's own fall-through is the header block (the next statement chains
	// from it); the then/else ends fall into the same continuation.
	return headerNo
}

// buildFor handles for/range loops. The header is the loop-condition block;
// the body is a child scope entered from the header ("loop-start") and looping
// back to the header ("loop"); the header's false path exits to the caller's
// continuation (the header block itself, which the next statement chains from).
func (c *cfg) buildFor(n *ts.Node, l *ts.Language, src []byte) int {
	headerNo := c.newBlock(branchLine(n, l, src), branchLine(n, l, src), "loop")
	if bodyBlock := findBlockChild(n, l); bodyBlock != nil {
		bodyStart := c.newScopeStart()
		c.addEdge(headerNo, bodyStart, "loop-start")
		if bodySl := statementListIn(bodyBlock, l); bodySl != nil {
			c.buildScope(bodySl, l, src)
		}
		// loop-back: the last block of the body loops back to the header.
		if len(c.Blocks) > 0 {
			c.addEdge(c.Blocks[len(c.Blocks)-1].BlockNo, headerNo, "loop")
		}
	}
	// false path: header -> caller continuation (the header block).
	return headerNo
}

// buildSwitch handles switch/select: the header is the condition block; each
// case body is a child scope entered from the header ("case"); all case ends
// fall through to the caller's continuation (the header block).
func (c *cfg) buildSwitch(n *ts.Node, l *ts.Language, src []byte) int {
	headerNo := c.newBlock(branchLine(n, l, src), branchLine(n, l, src), "branch")
	for i := 0; i < n.NamedChildCount(); i++ {
		child := n.NamedChild(i)
		if child == nil {
			continue
		}
		t := child.Type(l)
		if t != "expression_case" && t != "communication_case" && t != "default_case" && t != "type_switch_case" {
			continue
		}
		caseStart := c.newScopeStart()
		c.addEdge(headerNo, caseStart, "case")
		if sl := statementListIn(child, l); sl != nil {
			c.buildScope(sl, l, src)
		}
	}
	return headerNo
}

// newScopeStart creates a marker block for the start of a child scope (so the
// branch can have a single "true"/"false"/"case" edge into the scope). The
// first real statement of the scope is chained from it by buildScope.
func (c *cfg) newScopeStart() int {
	return c.newBlock(len(c.Blocks)+1, len(c.Blocks)+1, "normal")
}

// newBlock creates a block, reusing the last-created block when the new one
// would have an identical (start_line, end_line, kind) immediately before it
// (avoids degenerate empty blocks between adjacent statements).
func (c *cfg) newBlock(startLine, endLine int, kind string) int {
	if kind == "" {
		kind = "normal"
	}
	n := len(c.Blocks) + 1
	if n > 1 {
		prev := c.Blocks[n-2]
		if prev.StartLine == startLine && prev.EndLine == endLine && prev.Kind == kind {
			return prev.BlockNo
		}
	}
	c.Blocks = append(c.Blocks, Block{BlockNo: n, StartLine: startLine, EndLine: endLine, Kind: kind})
	return n
}

// addEdge adds a control edge, de-duplicating identical (from,to,cond).
func (c *cfg) addEdge(from, to int, cond string) {
	if from == 0 || to == 0 || from == to {
		return
	}
	for _, e := range c.Edges {
		if e.FromBlock == from && e.ToBlock == to && e.Condition == cond {
			return
		}
	}
	c.Edges = append(c.Edges, Edge{FromBlock: from, ToBlock: to, Condition: cond})
}

// addEdgeUnique adds a control edge only if no edge (any condition) between the
// same blocks already exists. Used for the single entry->terminal edge.
func (c *cfg) addEdgeUnique(from, to int, cond string) {
	if from == 0 || to == 0 {
		return
	}
	for _, e := range c.Edges {
		if e.FromBlock == from && e.ToBlock == to {
			return
		}
	}
	c.Edges = append(c.Edges, Edge{FromBlock: from, ToBlock: to, Condition: cond})
}

// findFunctionNode locates the definition node for (name, startLine).
func findFunctionNode(root *ts.Node, l *ts.Language, name string, startLine int, src []byte) *ts.Node {
	var found *ts.Node
	var walk func(n *ts.Node)
	walk = func(n *ts.Node) {
		if found != nil || n == nil {
			return
		}
		if defNodeName(n, l, src) == name {
			if lineOf(src, n.StartByte()) == startLine {
				found = n
				return
			}
		}
		for i := 0; i < n.ChildCount(); i++ {
			if c := n.Child(i); c != nil {
				walk(c)
			}
		}
	}
	walk(root)
	return found
}

// defNodeName returns the definition name of n when n carries a name field,
// else "".
func defNodeName(n *ts.Node, l *ts.Language, src []byte) string {
	nameNode := n.ChildByFieldName("name", l)
	if nameNode == nil {
		return ""
	}
	return nameNode.Text(src)
}

// branchLine is the line of a branch/loop/switch header node.
func branchLine(n *ts.Node, l *ts.Language, src []byte) int {
	_ = l
	return lineOf(src, n.StartByte())
}

// statementListIn returns the statement_list child of a block/case node.
func statementListIn(n *ts.Node, l *ts.Language) *ts.Node {
	if n == nil {
		return nil
	}
	sl := n.ChildByFieldName("statement_list", l)
	if sl != nil {
		return sl
	}
	sl = n.ChildByFieldName("statements", l)
	if sl != nil {
		return sl
	}
	for i := 0; i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c != nil {
			if t := c.Type(l); t == "statement_list" || t == "statements" {
				return c
			}
		}
	}
	return nil
}

// findBlockChild returns the first "block" child of a declaration node.
func findBlockChild(n *ts.Node, l *ts.Language) *ts.Node {
	for i := 0; i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c != nil && c.Type(l) == "block" {
			return c
		}
	}
	return nil
}

// hasCallExpr reports whether the statement (transitively) invokes a call.
func hasCallExpr(n *ts.Node, l *ts.Language) bool {
	var found bool
	var walk func(m *ts.Node)
	walk = func(m *ts.Node) {
		if found || m == nil {
			return
		}
		t := m.Type(l)
		if t == "call_expression" || t == "method_call_expression" || t == "call" {
			found = true
			return
		}
		for i := 0; i < m.ChildCount(); i++ {
			if c := m.Child(i); c != nil {
				walk(c)
			}
		}
	}
	walk(n)
	return found
}

// lineOf converts a byte offset to a 1-based line number (matches 005's).
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

// hasNoCFGMarker reports whether the line just above startLine (1-based) in src
// carries the noCFGMarker directive (a function the CFG pass should skip, 01.8).
func hasNoCFGMarker(src []byte, startLine int) bool {
	if startLine < 2 {
		return false
	}
	// Find the byte offset of the start of line startLine-1.
	lines := splitLinesByte(src)
	if startLine-1 > len(lines) {
		return false
	}
	prev := lines[startLine-2]
	return strings.TrimSpace(prev) == noCFGMarker
}

// splitLinesByte splits src into lines (without terminators) for hasNoCFGMarker.
func splitLinesByte(src []byte) []string {
	var out []string
	start := 0
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			out = append(out, string(src[start:i]))
			start = i + 1
		}
	}
	if start < len(src) {
		out = append(out, string(src[start:]))
	}
	return out
}

// DumpCFG renders a CFG into a deterministic string (blocks then edges) so a
// reproducibility check is a single string compare. Exported for tests.
func DumpCFG(c *cfg) string {
	if c == nil {
		return ""
	}
	s := ""
	for _, b := range c.Blocks {
		s += fmt.Sprintf("b%d[%d-%d %s]; ", b.BlockNo, b.StartLine, b.EndLine, b.Kind)
	}
	for _, e := range c.Edges {
		s += fmt.Sprintf("e%d->%d[%s]; ", e.FromBlock, e.ToBlock, e.Condition)
	}
	return s
}

// stableID derives a deterministic id from its parts (FNV-1a, pure Go).
func stableID(parts ...string) string {
	h := fnv.New64a()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum64())
}

// sortedStrings returns a sorted copy of in (for deterministic iteration).
func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
