package extract

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// langMaps holds the per-language node-type tables for languages the
// one-pass gotreesitter extractors do not cover. For the six covered
// languages (go, python, java, javascript, typescript, tsx) the adapter uses
// ExtractDefinitionSpans/ExtractCalls/ExtractHeritage/ExtractImports instead.
type langMapDef struct {
	defNodes  map[string]defInfo
	callNodes map[string]callInfo
	imports   func(tree *ts.Tree) []importRef
	heritage  func(tree *ts.Tree) []heritageRef
}

// NodeT is a type-scoped node accessor: Type() resolves against the node's
// language, while Child/Text/Bytes behave like ts.Node.
type NodeT struct {
	n    *ts.Node
	lang *ts.Language
}

func newNodeT(n *ts.Node, lang *ts.Language) *NodeT {
	if n == nil {
		return nil
	}
	return &NodeT{n: n, lang: lang}
}

func (t *NodeT) Node() *ts.Node { return t.n }

func (t *NodeT) Type() string {
	if t.n == nil || t.lang == nil {
		return ""
	}
	return t.n.Type(t.lang)
}

func (t *NodeT) Text(src []byte) string {
	if t.n == nil {
		return ""
	}
	return t.n.Text(src)
}

func (t *NodeT) StartByte() uint32 {
	if t.n == nil {
		return 0
	}
	return t.n.StartByte()
}

func (t *NodeT) EndByte() uint32 {
	if t.n == nil {
		return 0
	}
	return t.n.EndByte()
}

func (t *NodeT) NamedChildCount() int {
	if t.n == nil {
		return 0
	}
	return t.n.NamedChildCount()
}

func (t *NodeT) NamedChild(i int) *NodeT {
	if t.n == nil {
		return nil
	}
	return newNodeT(t.n.NamedChild(i), t.lang)
}

func (t *NodeT) FirstNamed() *NodeT {
	for i := 0; i < t.NamedChildCount(); i++ {
		if c := t.NamedChild(i); c != nil {
			return c
		}
	}
	return nil
}

func (t *NodeT) NextSibling() *NodeT {
	if t.n == nil {
		return nil
	}
	return newNodeT(t.n.NextSibling(), t.lang)
}

func (t *NodeT) Parent() *NodeT {
	if t.n == nil {
		return nil
	}
	return newNodeT(t.n.Parent(), t.lang)
}

func (t *NodeT) IsNamed() bool {
	if t.n == nil {
		return false
	}
	return t.n.IsNamed()
}

type defInfo struct {
	kind     string
	nameFrom func(n *NodeT, src []byte) string
}

type callInfo struct {
	targetFrom func(n *NodeT, src []byte) string
}

type importRef struct {
	path    string
	from    string
	name    string
	line    int
	kind    string
	confid  string
}

type heritageRef struct {
	name   string
	parent string
	kind   string
	line   int
}

var langMaps = map[string]langMapDef{
	"rust":        rustMaps,
	"c_sharp":     cSharpMaps,
	"cpp":         cppMaps,
	"c":           cMaps,
	"php":         phpMaps,
	"ruby":        rubyMaps,
	"kotlin":      kotlinMaps,
	"swift":       swiftMaps,
	"scala":       scalaMaps,
	"dart":        dartMaps,
	"lua":         luaMaps,
	"r":           rMaps,
	"matlab":      matlabMaps,
	"objective_c": objectiveCMaps,
	"perl":        perlMaps,
	"elixir":      elixirMaps,
	"haskell":     haskellMaps,
	"clojure":     clojureMaps,
	"zig":         zigMaps,
	"nim":         nimMaps,
	"groovy":      groovyMaps,
	"bash":        bashMaps,
	"sql":         sqlMaps,
	"html":        htmlMaps,
	"css":         cssMaps,
}

func langMapsFor(lang string) *langMapDef {
	m, ok := langMaps[lang]
	if !ok {
		return nil
	}
	return &m
}

// nameFromChild returns the trimmed text of the first named child of type
// childType, or "".
func nameFromChild(n *NodeT, childType string, src []byte) string {
	if n == nil {
		return ""
	}
	for i := 0; i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c != nil && c.Type() == childType {
			return strings.TrimSpace(c.Text(src))
		}
	}
	return ""
}

// firstChildType returns the first descendant (depth-first) of one of the
// given node types, or nil.
func firstChildType(n *NodeT, types ...string) *NodeT {
	seen := map[string]bool{}
	var walk func(cur *NodeT) *NodeT
	walk = func(cur *NodeT) *NodeT {
		if cur == nil {
			return nil
		}
		typ := cur.Type()
		if !seen[typ] {
			seen[typ] = true
			for _, want := range types {
				if typ == want {
					return cur
				}
			}
		}
		for i := 0; i < cur.NamedChildCount(); i++ {
			if found := walk(cur.NamedChild(i)); found != nil {
				return found
			}
		}
		return nil
	}
	return walk(n)
}

// --- Rust ---

var rustMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_item":  {kind: "function", nameFrom: nameFromChildFn("identifier")},
		"struct_item":    {kind: "struct", nameFrom: nameFromChildFn("type_identifier")},
		"enum_item":      {kind: "enum", nameFrom: nameFromChildFn("type_identifier")},
		"trait_item":     {kind: "trait", nameFrom: nameFromChildFn("type_identifier")},
		"impl_item":      {kind: "impl", nameFrom: rustImplName},
		"mod_item":       {kind: "module", nameFrom: nameFromChildFn("identifier")},
		"type_alias":     {kind: "type", nameFrom: nameFromChildFn("type_identifier")},
		"union_item":     {kind: "struct", nameFrom: nameFromChildFn("type_identifier")},
		"constant_item":  {kind: "constant", nameFrom: nameFromChildFn("identifier")},
		"static_item":    {kind: "constant", nameFrom: nameFromChildFn("identifier")},
	},
	callNodes: map[string]callInfo{
		"call_expression":   {targetFrom: rustCallTarget},
		"macro_invocation":  {targetFrom: nameFromChildFn("identifier")},
	},
	imports:  rustImports,
	heritage: nil,
}

func nameFromChildFn(childType string) func(n *NodeT, src []byte) string {
	return func(n *NodeT, src []byte) string { return nameFromChild(n, childType, src) }
}

func rustImplName(n *NodeT, src []byte) string {
	name := nameFromChild(n, "type_identifier", src)
	if name != "" {
		return name
	}
	name = nameFromChild(n, "identifier", src)
	return name
}

func rustCallTarget(n *NodeT, src []byte) string {
	if n == nil || n.NamedChildCount() == 0 {
		return ""
	}
	first := n.NamedChild(0)
	if first == nil {
		return ""
	}
	switch first.Type() {
	case "identifier":
		return first.Text(src)
	case "scoped_identifier":
		return first.Text(src)
	case "method_call":
		return ""
	case "field_expression":
		return ""
	default:
		return ""
	}
}

func rustImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "use_declaration" {
			walkTree(n.Node(), n.lang, func(inner *NodeT) bool {
				if inner.Type() == "scalar" || inner.Type() == "scoped_identifier" {
					path := strings.TrimSpace(inner.Text(src))
					path = strings.TrimSuffix(path, ";")
					out = append(out, importRef{path: path, kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
				}
				return true
			})
		}
		return true
	})
	return out
}

// --- C# ---

var cSharpMaps = langMapDef{
	defNodes: map[string]defInfo{
		"class_declaration":    {kind: "class", nameFrom: nameFromChildFn("identifier")},
		"interface_declaration": {kind: "interface", nameFrom: nameFromChildFn("identifier")},
		"method_declaration":   {kind: "method", nameFrom: nameFromChildFn("identifier")},
		"constructor_declaration": {kind: "constructor", nameFrom: nameFromChildFn("identifier")},
		"record_declaration":   {kind: "struct", nameFrom: nameFromChildFn("identifier")},
		"struct_declaration":   {kind: "struct", nameFrom: nameFromChildFn("identifier")},
		"enum_declaration":     {kind: "enum", nameFrom: nameFromChildFn("identifier")},
		"delegation_declaration": {kind: "function", nameFrom: nameFromChildFn("identifier")},
	},
	callNodes: map[string]callInfo{
		"invocation_expression": {targetFrom: cSharpCallTarget},
	},
	imports:  cSharpImports,
	heritage: cSharpHeritage,
}

func cSharpCallTarget(n *NodeT, src []byte) string {
	if n == nil || n.NamedChildCount() == 0 {
		return ""
	}
	first := n.NamedChild(0)
	if first == nil {
		return ""
	}
	switch first.Type() {
	case "identifier":
		return first.Text(src)
	case "member_access_expression":
		if id := firstChildType(first, "identifier"); id != nil {
			return id.Text(src)
		}
	default:
		return ""
	}
	return ""
}

func cSharpImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "using_directive" {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "using")
			text = strings.TrimSuffix(text, ";")
			text = strings.TrimSpace(text)
			out = append(out, importRef{path: text, kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
		}
		return true
	})
	return out
}

func cSharpHeritage(tree *ts.Tree) []heritageRef {
	var out []heritageRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "base_list" {
			walkTree(n.Node(), n.lang, func(inner *NodeT) bool {
				if inner.Type() == "identifier" {
					out = append(out, heritageRef{parent: inner.Text(src), kind: "extends", line: lineOf(src, inner.StartByte())})
				}
				return true
			})
		}
		return true
	})
	return out
}

// --- C / C++ ---

var cppMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_definition": {kind: "function", nameFrom: cFuncName},
		"class_specifier":     {kind: "class", nameFrom: nameFromChildFn("type_identifier")},
		"struct_specifier":    {kind: "struct", nameFrom: nameFromChildFn("type_identifier")},
		"enum_specifier":      {kind: "enum", nameFrom: nameFromChildFn("type_identifier")},
		"namespace_definition": {kind: "module", nameFrom: nameFromChildFn("identifier")},
		"field_declaration":   {kind: "method", nameFrom: cppFieldName},
		"function_declarator": {kind: "function", nameFrom: nameFromChildFn("identifier")},
	},
	callNodes: map[string]callInfo{
		"call_expression": {targetFrom: cCallTarget},
	},
	imports: cImports,
}

var cMaps = cppMaps

func cFuncName(n *NodeT, src []byte) string {
	if decl := firstChildType(n, "function_declarator"); decl != nil {
		if id := nameFromChild(decl, "identifier", src); id != "" {
			return id
		}
	}
	return nameFromChild(n, "identifier", src)
}

func cppFieldName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.Index(text, "(")
	if idx < 0 {
		return ""
	}
	head := strings.TrimSpace(text[:idx])
	for _, part := range strings.Fields(head) {
		if isIdent(part) {
			return part
		}
	}
	return ""
}

func cCallTarget(n *NodeT, src []byte) string {
	if n == nil || n.NamedChildCount() == 0 {
		return ""
	}
	first := n.NamedChild(0)
	if first == nil {
		return ""
	}
	switch first.Type() {
	case "identifier":
		return first.Text(src)
	case "field_expression":
		if id := firstChildType(first, "field_identifier"); id != nil {
			return id.Text(src)
		}
	default:
		return ""
	}
	return ""
}

func cImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "preproc_include" {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "#include")
			text = strings.TrimSpace(strings.Trim(text, `"<>`))
			if text != "" {
				out = append(out, importRef{path: text, kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
			}
		}
		return true
	})
	return out
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && (r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			continue
		}
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

// --- PHP ---

var phpMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_definition": {kind: "function", nameFrom: phpFuncName},
		"method_declaration":  {kind: "method", nameFrom: phpMethodName},
		"class_declaration":   {kind: "class", nameFrom: phpClassName},
		"interface_declaration": {kind: "interface", nameFrom: phpClassName},
		"trait_declaration":   {kind: "trait", nameFrom: phpClassName},
	},
	callNodes: map[string]callInfo{
		"function_call_expression": {targetFrom: phpCallTarget},
		"method_call_expression":   {targetFrom: phpMethodCallTarget},
		"object_creation_expression": {targetFrom: phpNewTarget},
	},
	imports:  phpImports,
	heritage: phpHeritage,
}

func phpFuncName(n *NodeT, src []byte) string {
	return nameFromChild(n, "identifier", src)
}

func phpMethodName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.Index(text, "function")
	if idx < 0 {
		return ""
	}
	head := strings.TrimSpace(text[idx+len("function"):])
	fields := strings.Fields(head)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimPrefix(fields[0], "&")
}

func phpClassName(n *NodeT, src []byte) string {
	return nameFromChild(n, "name", src)
}

func phpCallTarget(n *NodeT, src []byte) string {
	return nameFromChild(n, "name", src)
}

func phpMethodCallTarget(n *NodeT, src []byte) string {
	if name := nameFromChild(n, "name", src); name != "" {
		return name
	}
	return ""
}

func phpNewTarget(n *NodeT, src []byte) string {
	return nameFromChild(n, "name", src)
}

func phpImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "namespace_use_declaration" {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "use")
			text = strings.TrimSuffix(text, ";")
			text = strings.TrimSpace(text)
			if text != "" {
				out = append(out, importRef{path: text, kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
			}
		}
		return true
	})
	return out
}

func phpHeritage(tree *ts.Tree) []heritageRef {
	var out []heritageRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "extends_clause" {
			if id := firstChildType(n, "name", "scoped_name"); id != nil {
				out = append(out, heritageRef{parent: id.Text(src), kind: "extends", line: lineOf(src, id.StartByte())})
			}
		}
		if n.Type() == "implements_clause" {
			walkTree(n.Node(), n.lang, func(inner *NodeT) bool {
				if inner.Type() == "name" || inner.Type() == "scoped_name" {
					out = append(out, heritageRef{parent: inner.Text(src), kind: "implements", line: lineOf(src, inner.StartByte())})
				}
				return true
			})
		}
		return true
	})
	return out
}

// --- Ruby ---

var rubyMaps = langMapDef{
	defNodes: map[string]defInfo{
		"method":         {kind: "method", nameFrom: rubyMethodName},
		"block":          {kind: "function", nameFrom: nil},
		"class":          {kind: "class", nameFrom: nameFromChildFn("constant")},
		"module":         {kind: "module", nameFrom: nameFromChildFn("constant")},
		"singleton_method": {kind: "method", nameFrom: rubyMethodName},
	},
	callNodes: map[string]callInfo{
		"call": {targetFrom: rubyCallTarget},
	},
	imports:  rubyImports,
	heritage: rubyHeritage,
}

func rubyMethodName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.IndexByte(text, ' ')
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(text[idx:])
}

func rubyCallTarget(n *NodeT, src []byte) string {
	first := n.FirstNamed()
	if first == nil {
		return ""
	}
	if first.Type() == "identifier" {
		return first.Text(src)
	}
	return ""
}

func rubyImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "call" {
			first := n.FirstNamed()
			if first != nil && first.Type() == "identifier" {
				if name := first.Text(src); name == "require" || name == "require_relative" || name == "load" {
					if arg := n.FirstNamed(); arg != nil {
						out = append(out, importRef{path: strings.Trim(arg.Text(src), `'"`), kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
					}
				}
			}
		}
		return true
	})
	return out
}

func rubyHeritage(tree *ts.Tree) []heritageRef {
	var out []heritageRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "class" && n.NamedChildCount() >= 2 {
			if c := n.NamedChild(1); c != nil && c.Type() == "constant" {
				out = append(out, heritageRef{parent: c.Text(src), kind: "extends", line: lineOf(src, c.StartByte())})
			}
		}
		return true
	})
	return out
}

// --- Kotlin ---

var kotlinMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_declaration": {kind: "function", nameFrom: nameFromChildFn("simple_identifier")},
		"class_declaration":    {kind: "class", nameFrom: nameFromChildFn("type_identifier")},
		"object_declaration":   {kind: "class", nameFrom: nameFromChildFn("simple_identifier")},
		"interface_declaration": {kind: "interface", nameFrom: nameFromChildFn("type_identifier")},
		"enum_class_declaration": {kind: "enum", nameFrom: nameFromChildFn("type_identifier")},
	},
	callNodes: map[string]callInfo{
		"call_expression": {targetFrom: nameFromChildFn("simple_identifier")},
	},
	imports:  kotlinImports,
	heritage: kotlinHeritage,
}

func kotlinImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "import_header" {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "import")
			text = strings.TrimSpace(text)
			if text != "" {
				out = append(out, importRef{path: text, kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
			}
		}
		return true
	})
	return out
}

func kotlinHeritage(tree *ts.Tree) []heritageRef {
	var refs []heritageRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "super_type_list" || n.Type() == "type_arguments" {
			walkTree(n.Node(), n.lang, func(inner *NodeT) bool {
				if inner.Type() == "type_identifier" {
					refs = append(refs, heritageRef{parent: inner.Text(src), kind: "extends", line: lineOf(src, inner.StartByte())})
				}
				return true
			})
		}
		return true
	})
	return refs
}

// --- Swift ---

var swiftMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_declaration":  {kind: "function", nameFrom: swiftFuncName},
		"method_declaration":    {kind: "method", nameFrom: swiftFuncName},
		"class_declaration":     {kind: "class", nameFrom: swiftDeclName},
		"struct_declaration":    {kind: "struct", nameFrom: swiftDeclName},
		"enum_declaration":      {kind: "enum", nameFrom: swiftDeclName},
		"protocol_declaration":  {kind: "interface", nameFrom: swiftDeclName},
		"extension_declaration": {kind: "class", nameFrom: swiftDeclName},
	},
	callNodes: map[string]callInfo{
		"call_expression": {targetFrom: swiftCallTarget},
	},
	imports:  swiftImports,
	heritage: swiftHeritage,
}

func swiftFuncName(n *NodeT, src []byte) string {
	return nameFromChild(n, "simple_identifier", src)
}

func swiftDeclName(n *NodeT, src []byte) string {
	return nameFromChild(n, "type_identifier", src)
}

func swiftCallTarget(n *NodeT, src []byte) string {
	if n == nil || n.NamedChildCount() == 0 {
		return ""
	}
	first := n.NamedChild(0)
	if first == nil {
		return ""
	}
	if first.Type() == "simple_identifier" {
		return first.Text(src)
	}
	if first.Type() == "member_access_expression" {
		if id := firstChildType(first, "simple_identifier"); id != nil {
			return id.Text(src)
		}
	}
	return ""
}

func swiftImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "import_declaration" {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "import")
			text = strings.TrimSpace(text)
			if text != "" {
				out = append(out, importRef{path: text, kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
			}
		}
		return true
	})
	return out
}

func swiftHeritage(tree *ts.Tree) []heritageRef {
	var out []heritageRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "inheritance" {
			walkTree(n.Node(), n.lang, func(inner *NodeT) bool {
				if inner.Type() == "type_identifier" {
					out = append(out, heritageRef{parent: inner.Text(src), kind: "extends", line: lineOf(src, inner.StartByte())})
				}
				return true
			})
		}
		return true
	})
	return out
}

// --- Scala ---

var scalaMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_definition": {kind: "function", nameFrom: nameFromChildFn("identifier")},
		"class_definition":    {kind: "class", nameFrom: nameFromChildFn("identifier")},
		"trait_definition":    {kind: "interface", nameFrom: nameFromChildFn("identifier")},
		"object_definition":   {kind: "class", nameFrom: nameFromChildFn("identifier")},
		"enum_definition":     {kind: "enum", nameFrom: nameFromChildFn("identifier")},
	},
	callNodes: map[string]callInfo{
		"call_expression": {targetFrom: nameFromChildFn("identifier")},
	},
	imports: scalaImports,
}

func scalaImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "import_declaration" {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "import")
			text = strings.TrimSpace(text)
			if text != "" {
				out = append(out, importRef{path: text, kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
			}
		}
		return true
	})
	return out
}

// --- Dart ---

var dartMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_signature": {kind: "function", nameFrom: nameFromChildFn("identifier")},
		"class_declaration":  {kind: "class", nameFrom: nameFromChildFn("identifier")},
		"enum_declaration":   {kind: "enum", nameFrom: nameFromChildFn("identifier")},
		"mixin_declaration":  {kind: "class", nameFrom: nameFromChildFn("identifier")},
		"extension_declaration": {kind: "class", nameFrom: nameFromChildFn("identifier")},
		"function_body":      {kind: "function", nameFrom: nil},
	},
	callNodes: map[string]callInfo{
		"function_expression": {targetFrom: nameFromChildFn("identifier")},
	},
	imports: dartImports,
}

func dartImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "import_or_export_declaration" {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "import")
			text = strings.TrimSpace(strings.Trim(text, ";"))
			text = strings.Trim(text, `'"`)
			text = strings.TrimSpace(text)
			if text != "" {
				out = append(out, importRef{path: text, kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
			}
		}
		return true
	})
	return out
}

// --- Lua ---

var luaMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_declaration": {kind: "function", nameFrom: luaFuncName},
		"field_declaration":    {kind: "function", nameFrom: nil},
	},
	callNodes: map[string]callInfo{
		"function_call": {targetFrom: luaCallTarget},
	},
	imports: luaImports,
}

func luaFuncName(n *NodeT, src []byte) string {
	return nameFromChild(n, "identifier", src)
}

func luaCallTarget(n *NodeT, src []byte) string {
	if n == nil || n.NamedChildCount() == 0 {
		return ""
	}
	first := n.NamedChild(0)
	if first == nil {
		return ""
	}
	if first.Type() == "identifier" {
		return first.Text(src)
	}
	return ""
}

func luaImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "function_call" {
			first := n.FirstNamed()
			if first != nil && first.Type() == "identifier" && first.Text(src) == "require" {
				if arg := n.FirstNamed(); arg != nil {
					out = append(out, importRef{path: strings.Trim(arg.Text(src), `'"`), kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
				}
			}
		}
		return true
	})
	return out
}

// --- R ---

var rMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_definition": {kind: "function", nameFrom: nil},
	},
	callNodes: map[string]callInfo{
		"call": {targetFrom: nameFromChildFn("symbol")},
	},
	imports: rImports,
}

func rImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "call" {
			first := n.FirstNamed()
			if first != nil && first.Type() == "symbol" {
				if name := first.Text(src); name == "library" || name == "require" {
					if arg := n.FirstNamed(); arg != nil {
						out = append(out, importRef{path: arg.Text(src), kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
					}
				}
			}
		}
		return true
	})
	return out
}

// --- MATLAB ---

var matlabMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_definition": {kind: "function", nameFrom: matlabFuncName},
		"function":            {kind: "function", nameFrom: matlabFuncName},
	},
	callNodes: map[string]callInfo{
		"function_call": {targetFrom: nameFromChildFn("identifier")},
	},
}

func matlabFuncName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.Index(text, "function")
	if idx < 0 {
		return ""
	}
	rest := text[idx+len("function"):]
	rest = strings.TrimSpace(rest)
	rest = strings.TrimSuffix(rest, "=")
	rest = strings.TrimSpace(rest)
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimRight(fields[0], "= ")
}

// --- Objective-C ---

var objectiveCMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_definition":   {kind: "function", nameFrom: cFuncName},
		"class_interface":       {kind: "class", nameFrom: nameFromChildFn("identifier")},
		"class_implementation":  {kind: "class", nameFrom: nameFromChildFn("identifier")},
		"protocol_interface":    {kind: "interface", nameFrom: nameFromChildFn("identifier")},
		"message_expression":    {kind: "method", nameFrom: nil},
		"method_definition":     {kind: "method", nameFrom: objcMethodName},
		"category_interface":    {kind: "class", nameFrom: nameFromChildFn("identifier")},
	},
	callNodes: map[string]callInfo{
		"call_expression":    {targetFrom: cCallTarget},
		"message_expression": {targetFrom: objcSelector},
	},
	imports: cImports,
}

func objcMethodName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.Index(text, ":")
	if idx < 0 {
		idx = strings.Index(text, "(")
	}
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(text[:idx])
}

func objcSelector(n *NodeT, src []byte) string {
	text := n.Text(src)
	if !strings.HasPrefix(text, "@") {
		return ""
	}
	rest := strings.TrimSpace(text[1:])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimPrefix(fields[0], "[")
}

// --- Perl ---

var perlMaps = langMapDef{
	defNodes: map[string]defInfo{
		"subroutine_declaration_statement": {kind: "function", nameFrom: perlSubName},
	},
	callNodes: map[string]callInfo{
		"ambiguous_function_call_expression": {targetFrom: nameFromChildFn("word")},
		"term_list":                          {targetFrom: nil},
	},
	imports: perlImports,
}

func perlSubName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.Index(text, "sub")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(text[idx+len("sub"):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSuffix(fields[0], "{")
}

func perlImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "call" || n.Type() == "call_signature" {
			first := n.FirstNamed()
			if first != nil && first.Type() == "word" {
				if name := first.Text(src); name == "use" || name == "require" {
					if arg := n.FirstNamed(); arg != nil {
						out = append(out, importRef{path: strings.Trim(arg.Text(src), `'"`), kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
					}
				}
			}
		}
		return true
	})
	return out
}

// --- Elixir ---

var elixirMaps = langMapDef{
	defNodes: map[string]defInfo{
		"module":      {kind: "module", nameFrom: elixirModuleName},
		"call":        {kind: "function", nameFrom: elixirDefName},
	},
	callNodes: map[string]callInfo{
		"call": {targetFrom: elixirCallTarget},
	},
}

func elixirModuleName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.Index(text, "do")
	if idx >= 0 {
		text = text[:idx]
	}
	text = strings.TrimSpace(strings.TrimPrefix(text, "defmodule"))
	return strings.TrimSpace(text)
}

func elixirDefName(n *NodeT, src []byte) string {
	text := n.Text(src)
	text = strings.TrimPrefix(text, "def")
	text = strings.TrimSpace(text)
	fields := strings.FieldsFunc(text, func(r rune) bool { return r == '(' || r == ' ' || r == ',' })
	if len(fields) == 0 {
		return ""
	}
	name := fields[0]
	if strings.HasPrefix(name, "def") {
		return ""
	}
	return name
}

func elixirCallTarget(n *NodeT, src []byte) string {
	first := n.FirstNamed()
	if first == nil {
		return ""
	}
	if first.Type() == "identifier" || first.Type() == "remote_call" {
		return first.Text(src)
	}
	return ""
}

// --- Haskell ---

var haskellMaps = langMapDef{
	defNodes: map[string]defInfo{
		"declaration":   {kind: "function", nameFrom: haskellDeclName},
		"signature":     {kind: "function", nameFrom: haskellDeclName},
		"import_declaration": {kind: "import", nameFrom: nil},
		"qualified_declaration": {kind: "type", nameFrom: haskellDeclName},
		"type_alias":    {kind: "type", nameFrom: haskellDeclName},
	},
	callNodes: map[string]callInfo{
		"explicit_function_application": {targetFrom: haskellCallTarget},
	},
	imports: haskellImports,
}

func haskellDeclName(n *NodeT, src []byte) string {
	text := n.Text(src)
	if idx := strings.IndexByte(text, ':'); idx >= 0 {
		text = text[:idx]
	}
	if idx := strings.IndexByte(text, '='); idx >= 0 {
		text = text[:idx]
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	name := fields[0]
	if strings.ContainsAny(name, "<>") {
		return ""
	}
	return name
}

func haskellCallTarget(n *NodeT, src []byte) string {
	first := n.FirstNamed()
	if first == nil {
		return ""
	}
	return first.Text(src)
}

func haskellImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "import_declaration" {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "import")
			text = strings.TrimSpace(text)
			fields := strings.Fields(text)
			if len(fields) > 0 {
				out = append(out, importRef{path: fields[0], kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
			}
		}
		return true
	})
	return out
}

// --- Clojure ---

var clojureMaps = langMapDef{
	defNodes: map[string]defInfo{
		"list_lit": {kind: "function", nameFrom: clojureDefnName},
		"sym_lit":  {kind: "function", nameFrom: nil},
	},
	callNodes: map[string]callInfo{
		"list_lit": {targetFrom: clojureCallTarget},
	},
	imports: clojureImports,
}

func clojureDefnName(n *NodeT, src []byte) string {
	first := n.FirstNamed()
	if first == nil || first.Type() != "sym_lit" {
		return ""
	}
	text := first.Text(src)
	if text != "defn" && text != "defn-" {
		return ""
	}
	if second := nextNamed(first, n); second != nil && second.Type() == "sym_lit" {
		return second.Text(src)
	}
	return ""
}

func clojureCallTarget(n *NodeT, src []byte) string {
	first := n.FirstNamed()
	if first == nil {
		return ""
	}
	if first.Type() == "sym_lit" {
		return first.Text(src)
	}
	return ""
}

func clojureImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "list_lit" {
			first := n.FirstNamed()
			if first == nil || first.Type() != "sym_lit" {
				return true
			}
			if text := first.Text(src); text == "ns" {
				walkTree(n.Node(), n.lang, func(inner *NodeT) bool {
					if inner.Type() == "require" || inner.Type() == "use" || inner.Type() == "import" {
						out = append(out, importRef{path: strings.TrimSpace(inner.Text(src)), kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
					}
					return true
				})
			}
		}
		return true
	})
	return out
}

func nextNamed(target, parent *NodeT) *NodeT {
	cur := target
	for cur != nil && cur != parent {
		cur = cur.NextSibling()
	}
	for cur != nil && !cur.IsNamed() {
		cur = cur.NextSibling()
	}
	return cur
}

// --- Zig ---

var zigMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_declaration": {kind: "function", nameFrom: nameFromChildFn("identifier")},
		"const_declaration":    {kind: "constant", nameFrom: nameFromChildFn("identifier")},
		"var_declaration":      {kind: "variable", nameFrom: nameFromChildFn("identifier")},
		"struct_declaration":   {kind: "struct", nameFrom: nameFromChildFn("identifier")},
		"enum_declaration":     {kind: "enum", nameFrom: nameFromChildFn("identifier")},
		"union_declaration":    {kind: "struct", nameFrom: nameFromChildFn("identifier")},
	},
	callNodes: map[string]callInfo{
		"function_call":     {targetFrom: zigCallTarget},
		"builtin_function":  {targetFrom: nameFromChildFn("identifier")},
	},
	imports: zigImports,
}

func zigCallTarget(n *NodeT, src []byte) string {
	if n == nil || n.NamedChildCount() == 0 {
		return ""
	}
	first := n.NamedChild(0)
	if first == nil {
		return ""
	}
	if first.Type() == "identifier" || first.Type() == "qualified_name" {
		return first.Text(src)
	}
	return ""
}

func zigImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "builtin_function" && strings.HasPrefix(n.Text(src), "@import") {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "@import")
			text = strings.TrimSpace(strings.Trim(text, "()"))
			if text != "" {
				out = append(out, importRef{path: strings.Trim(text, `'"`), kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
			}
		}
		return true
	})
	return out
}

// --- Nim ---

var nimMaps = langMapDef{
	defNodes: map[string]defInfo{
		"proc_declaration":   {kind: "function", nameFrom: nimProcName},
		"type_section":       {kind: "type", nameFrom: nil},
		"iterator_declaration": {kind: "function", nameFrom: nimProcName},
		"const_section":      {kind: "constant", nameFrom: nil},
		"let_section":        {kind: "variable", nameFrom: nil},
	},
	callNodes: map[string]callInfo{
		"call":          {targetFrom: nameFromChildFn("identifier")},
		"command":       {targetFrom: nameFromChildFn("identifier")},
	},
	imports: nimImports,
}

func nimProcName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.Index(text, "proc")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(text[idx+len("proc"):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSuffix(fields[0], ":")
}

func nimImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "import_stmt" {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "import")
			text = strings.TrimSpace(text)
			for _, p := range strings.Split(text, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					out = append(out, importRef{path: p, kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
				}
			}
		}
		return true
	})
	return out
}

// --- Groovy ---

var groovyMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_definition":  {kind: "function", nameFrom: groovyFuncName},
		"class_definition":     {kind: "class", nameFrom: nameFromChildFn("identifier")},
		"interface_definition": {kind: "interface", nameFrom: nameFromChildFn("identifier")},
		"method_definition":    {kind: "method", nameFrom: groovyFuncName},
	},
	callNodes: map[string]callInfo{
		"function_call":     {targetFrom: nameFromChildFn("identifier")},
		"method_call":       {targetFrom: nameFromChildFn("identifier")},
		"call_expression":   {targetFrom: nameFromChildFn("identifier")},
	},
	imports: groovyImports,
}

func groovyFuncName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.Index(text, "def")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(text[idx+len("def"):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSuffix(fields[0], "(")
}

func groovyImports(tree *ts.Tree) []importRef {
	var out []importRef
	src := tree.Source()
	walkTree(tree.RootNode(), tree.Language(), func(n *NodeT) bool {
		if n.Type() == "import_definition" {
			text := strings.TrimSpace(n.Text(src))
			text = strings.TrimPrefix(text, "import")
			text = strings.TrimSpace(strings.Trim(text, ";"))
			if text != "" {
				out = append(out, importRef{path: text, kind: "import", confid: ConfidenceExtracted, line: lineOf(src, n.StartByte())})
			}
		}
		return true
	})
	return out
}

// --- Bash ---

var bashMaps = langMapDef{
	defNodes: map[string]defInfo{
		"function_definition": {kind: "function", nameFrom: bashFuncName},
	},
	callNodes: map[string]callInfo{
		"command": {targetFrom: bashCmdName},
	},
}

func bashFuncName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.IndexByte(text, '(')
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(text[:idx])
}

func bashCmdName(n *NodeT, src []byte) string {
	first := n.FirstNamed()
	if first == nil {
		return ""
	}
	if first.Type() == "word" {
		return first.Text(src)
	}
	return ""
}

// --- SQL ---

var sqlMaps = langMapDef{
	defNodes: map[string]defInfo{
		"create_function_statement": {kind: "function", nameFrom: sqlFuncName},
		"create_procedure_statement": {kind: "function", nameFrom: sqlFuncName},
		"create_table_statement":    {kind: "type", nameFrom: sqlObjectName},
		"create_view_statement":     {kind: "type", nameFrom: sqlObjectName},
		"create_index_statement":    {kind: "type", nameFrom: sqlObjectName},
	},
	callNodes: map[string]callInfo{
		"function_call": {targetFrom: nameFromChildFn("name")},
	},
}

func sqlFuncName(n *NodeT, src []byte) string {
	text := n.Text(src)
	idx := strings.Index(text, "FUNCTION")
	if idx < 0 {
		idx = strings.Index(text, "PROCEDURE")
	}
	if idx < 0 {
		return ""
	}
	rest := text[idx:]
	rest = strings.TrimSpace(rest)
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], `()"`)
}

func sqlObjectName(n *NodeT, src []byte) string {
	for _, ct := range []string{"name", "object_name", "table_name", "view_name", "index_name"} {
		if name := nameFromChild(n, ct, src); name != "" {
			return name
		}
	}
	text := n.Text(src)
	fields := strings.Fields(text)
	for _, f := range fields[1:] {
		if isIdent(f) {
			return f
		}
	}
	return ""
}

// --- HTML ---

var htmlMaps = langMapDef{
	defNodes: map[string]defInfo{
		"script_element":  {kind: "function", nameFrom: nil},
		"element":         {kind: "element", nameFrom: nameFromChildFn("tag_name")},
	},
	callNodes: map[string]callInfo{},
}

// --- CSS ---

var cssMaps = langMapDef{
	defNodes: map[string]defInfo{
		"selector_list":   {kind: "selector", nameFrom: cssSelectorName},
		"rule_set":        {kind: "selector", nameFrom: nil},
		"keyframes":       {kind: "keyframes", nameFrom: nameFromChildFn("identifier")},
		"at_rule":         {kind: "at_rule", nameFrom: nil},
		"custom_property": {kind: "variable", nameFrom: nil},
	},
	callNodes: map[string]callInfo{},
}

func cssSelectorName(n *NodeT, src []byte) string {
	text := n.Text(src)
	text = strings.TrimLeft(text, ".#&: ")
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ' ' || r == '>' || r == '+' || r == '~'
	})
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimLeft(fields[0], ".#")
}

// walkTree is a pre-order walk over named children, yielding NodeT.
func walkTree(n *ts.Node, lang *ts.Language, fn func(*NodeT) bool) {
	if n == nil {
		return
	}
	t := newNodeT(n, lang)
	if fn(t) {
		for i := 0; i < t.NamedChildCount(); i++ {
			walkTree(t.NamedChild(i).Node(), lang, fn)
		}
	}
}
