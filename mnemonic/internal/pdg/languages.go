package pdg

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// grammarProbeFile maps each supported language to a filename whose extension
// the gotreesitter registry resolves to that language. It mirrors the map 005's
// extractor uses (extract.grammarProbeFile) but is self-contained so the pdg
// package has no dependency on the extract package (the CFG rides on the same
// gotreesitter registry, not a new grammar).
var grammarProbeFile = map[string]string{
	"go":         "a.go",
	"typescript": "a.ts",
	"tsx":        "a.tsx",
	"javascript": "a.js",
	"python":     "a.py",
	"rust":       "a.rs",
	"java":       "a.java",
	"c":          "a.c",
	"cpp":        "a.cpp",
	"c_sharp":    "a.cs",
	"php":        "a.php",
	"ruby":       "a.rb",
	"kotlin":     "a.kt",
	"swift":      "a.swift",
	"scala":      "a.scala",
	"dart":       "a.dart",
	"lua":        "a.lua",
	"r":          "a.r",
	"matlab":     "a.m",
	"perl":       "a.pl",
	"elixir":     "a.ex",
	"haskell":    "a.hs",
	"clojure":    "a.clj",
	"zig":        "a.zig",
	"nim":        "a.nim",
	"groovy":     "a.groovy",
	"bash":       "a.sh",
	"sql":        "a.sql",
}

// grammarByName lists languages the gotreesitter registry cannot reach by
// filename (their extension is claimed by another language — objective_c's .m
// is owned by matlab, .mm by xml) but resolves by canonical name. ParseTree
// falls back to DetectLanguageByName when the probe-file lookup misses.
var grammarByName = map[string]string{
	"objective_c": "objc",
}

// langShape captures the per-language AST facts the CFG relies on: which node
// type holds a function definition, which of its named children is the body,
// how to read the definition's name, and which node types are
// branch/loop/return. A language absent from this map (go/typescript/...) uses
// the Go-style defaults already hardcoded in cfg.go. The shapes mirror the
// extractor's defNodes (extract/languages.go) so PDG locates the same def node
// the 005 pass indexed (same StartLine, same name).
type langShape struct {
	defNodes  []string // node types that hold a function definition
	bodyTypes map[string]struct{}
	nameFrom  func(n *ts.Node, l *ts.Language, src []byte) string
	brTypes   map[string]struct{}
	loopTypes map[string]struct{}
	retTypes  map[string]struct{}
}

var setOf = func(s ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return m
}

// langShapes is the per-language CFG shape table. Node types below were verified
// against the gotreesitter v0.52.0 grammars; the nameFrom functions reproduce
// the extractor's def-name extraction verbatim.
var langShapes = map[string]langShape{
	"ruby": {
		defNodes:  []string{"method", "singleton_method"},
		bodyTypes: setOf("body_statement"),
		nameFrom:  childName("identifier"),
		brTypes:   setOf("if"),
		loopTypes: setOf("while", "for"),
		retTypes:  setOf("return"),
	},
	"php": {
		defNodes:  []string{"function_definition"},
		bodyTypes: setOf("compound_statement"),
		nameFrom: func(n *ts.Node, l *ts.Language, src []byte) string {
			name := childNameValue(n, l, "name", src)
			if name == "" {
				name = childNameValue(n, l, "identifier", src)
			}
			return strings.TrimPrefix(name, "&")
		},
		brTypes:   setOf("if_statement"),
		loopTypes: setOf("for_statement", "foreach_statement", "while_statement", "do_statement"),
		retTypes:  setOf("return_statement"),
	},
	"swift": {
		defNodes:  []string{"function_declaration"},
		bodyTypes: setOf("function_body"),
		nameFrom:  childName("simple_identifier"),
		brTypes:   setOf("if_statement", "switch_statement"),
		loopTypes: setOf("for_statement", "while_statement"),
		retTypes:  setOf("control_transfer_statement"),
	},
	"kotlin": {
		defNodes:  []string{"function_declaration"},
		bodyTypes: setOf("function_body"),
		nameFrom:  childName("simple_identifier"),
		brTypes:   setOf("if_expression", "when_expression"),
		loopTypes: setOf("for_statement", "while_statement", "do_statement"),
		retTypes:  setOf("jump_expression"),
	},
	"scala": {
		defNodes:  []string{"function_definition"},
		bodyTypes: setOf("block"),
		nameFrom:  childName("identifier"),
		brTypes:   setOf("if_expression"),
		loopTypes: setOf("for_statement", "while_expression"),
		retTypes:  setOf("return_expression"),
	},
	"lua": {
		defNodes:  []string{"function_declaration"},
		bodyTypes: setOf("block"),
		nameFrom:  childName("identifier"),
		brTypes:   setOf("if_statement"),
		loopTypes: setOf("while_statement", "for_statement", "generic_for_statement"),
		retTypes:  setOf("return_statement"),
	},
	"r": {
		// The R extractor (nameFrom nil) reports function_definition defs with
		// no name; PDG therefore locates by def-node + StartLine only (name "").
		defNodes:  []string{"function_definition"},
		bodyTypes: setOf("braced_expression"),
		nameFrom:  nil,
		brTypes:   setOf("if_statement"),
		loopTypes: setOf("for_statement", "while_statement", "repeat_statement"),
		retTypes:  setOf("return"),
	},
	"matlab": {
		defNodes:  []string{"function_definition"},
		bodyTypes: setOf("block"),
		nameFrom: func(n *ts.Node, l *ts.Language, src []byte) string {
			text := n.Text(src)
			idx := strings.Index(text, "function")
			if idx < 0 {
				return ""
			}
			rest := strings.TrimSpace(text[idx+len("function"):])
			rest = strings.TrimSuffix(rest, "=")
			rest = strings.TrimSpace(rest)
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				return ""
			}
			return strings.TrimRight(fields[0], "= ")
		},
		brTypes:   setOf("if_statement"),
		loopTypes: setOf("for_statement", "while_statement"),
		retTypes:  setOf("return_statement"),
	},
	"perl": {
		defNodes:  []string{"subroutine_declaration_statement"},
		bodyTypes: setOf("block"),
		nameFrom: func(n *ts.Node, l *ts.Language, src []byte) string {
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
		},
		brTypes:   setOf("conditional_statement"),
		loopTypes: setOf("for_statement", "loop_statement"),
		retTypes:  setOf("return_expression"),
	},
	"elixir": {
		defNodes:  []string{"call"},
		bodyTypes: setOf("do_block"),
		nameFrom: func(n *ts.Node, l *ts.Language, src []byte) string {
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
		},
		brTypes:   setOf("if_expression", "case_expression"),
		loopTypes: setOf("for_expression", "while_expression"),
		retTypes:  setOf("return"),
	},
	"haskell": {
		// The gotreesitter haskell grammar emits `function` for top-level defs
		// (signature + patterns + match). `bind` is the body-level binding.
		defNodes:  []string{"function", "bind"},
		bodyTypes: setOf("match"),
		nameFrom: func(n *ts.Node, l *ts.Language, src []byte) string {
			name := childNameValue(n, l, "variable", src)
			if name == "" {
				name = childNameValue(n, l, "identifier", src)
			}
			return name
		},
		brTypes:   setOf("expression"),
		loopTypes: setOf(),
		retTypes:  setOf(),
	},
	"clojure": {
		defNodes:  []string{"list_lit"},
		bodyTypes: setOf("list_lit"),
		nameFrom: func(n *ts.Node, l *ts.Language, src []byte) string {
			first := firstNamed(n)
			if first == nil || first.Type(l) != "sym_lit" {
				return ""
			}
			text := first.Text(src)
			if text != "defn" && text != "defn-" {
				return ""
			}
			// the defn's own name is the next sym_lit after the defn keyword
			// (skipping the optional map of metadata, which is a map_lit).
			for i := 1; i < n.NamedChildCount(); i++ {
				c := n.NamedChild(i)
				if c != nil && c.Type(l) == "sym_lit" {
					return c.Text(src)
				}
			}
			return ""
		},
		brTypes:   setOf("list_lit"),
		loopTypes: setOf(),
		retTypes:  setOf(),
	},
	"zig": {
		defNodes:  []string{"function_declaration"},
		bodyTypes: setOf("block"),
		nameFrom:  childName("identifier"),
		brTypes:   setOf("if_statement", "switch_statement"),
		loopTypes: setOf("for_statement", "while_statement"),
		retTypes:  setOf("return_statement"),
	},
	"nim": {
		defNodes:  []string{"proc_declaration", "iterator_declaration"},
		bodyTypes: setOf("statement_list"),
		nameFrom:  childName("identifier"),
		brTypes:   setOf("if"),
		loopTypes: setOf("for", "while", "block"),
		retTypes:  setOf("return"),
	},
	"groovy": {
		defNodes:  []string{"function_definition", "method_definition"},
		bodyTypes: setOf("closure"),
		nameFrom:  childName("identifier"),
		brTypes:   setOf("if_statement"),
		loopTypes: setOf("for_in_loop", "for_statement", "while_loop"),
		retTypes:  setOf("return"),
	},
	"bash": {
		defNodes:  []string{"function_definition"},
		bodyTypes: setOf("compound_statement"),
		nameFrom: func(n *ts.Node, l *ts.Language, src []byte) string {
			text := n.Text(src)
			idx := strings.IndexByte(text, '(')
			if idx < 0 {
				return ""
			}
			return strings.TrimSpace(text[:idx])
		},
		brTypes:   setOf("if_statement"),
		loopTypes: setOf("for_statement", "while_statement", "until_statement"),
		retTypes:  setOf("return"),
	},
	"sql": {
		defNodes:  []string{"create_function_statement", "create_procedure_statement"},
		bodyTypes: setOf("function_body"),
		nameFrom:  childName("identifier"),
		brTypes:   setOf("if_statement"),
		loopTypes: setOf("loop_statement", "while_loop"),
		retTypes:  setOf("return_statement"),
	},
	"objective_c": {
		defNodes:  []string{"method_definition"},
		bodyTypes: setOf("compound_statement"),
		nameFrom:  childName("identifier"),
		brTypes:   setOf("if_statement", "switch_statement"),
		loopTypes: setOf("for_statement", "while_statement"),
		retTypes:  setOf("return_statement"),
	},
}

// shapeFor returns the CFG shape for lang, or nil for the Go-style defaults.
func shapeFor(lang string) *langShape {
	if s, ok := langShapes[lang]; ok {
		return &s
	}
	return nil
}

// childName returns a nameFrom function that reads the first named child of
// type typ (e.g. the "identifier" under a function_definition).
func childName(typ string) func(n *ts.Node, l *ts.Language, src []byte) string {
	return func(n *ts.Node, l *ts.Language, src []byte) string {
		return childNameValue(n, l, typ, src)
	}
}

// childNameValue returns the text of n's first named child of type typ, or "".
func childNameValue(n *ts.Node, l *ts.Language, typ string, src []byte) string {
	for i := 0; i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c != nil && c.Type(l) == typ {
			return strings.TrimSpace(c.Text(src))
		}
	}
	return ""
}

// firstNamed returns n's first named child, or nil. The vendored gotreesitter
// Node exposes NamedChild(i) but not FirstNamed, so the shape table uses this.
func firstNamed(n *ts.Node) *ts.Node {
	if n == nil || n.NamedChildCount() == 0 {
		return nil
	}
	return n.NamedChild(0)
}

// nextNamed returns the first named sibling after target within parent, or
// nil. Ported from the extractor's nextNamed (extract/languages.go) for the
// clojure defn name lookup.
func nextNamed(target, parent *ts.Node, l *ts.Language) *ts.Node {
	cur := target
	for cur != nil && cur != parent {
		cur = cur.NextSibling()
	}
	for cur != nil && !cur.IsNamed() {
		cur = cur.NextSibling()
	}
	return cur
}
