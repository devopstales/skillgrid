package extract

import (
	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// resolveGrammar maps a supported language name to a lazy gotreesitter
// language loader via the registry. Grammar reuse (e.g. .mts -> typescript,
// .cu -> cpp) is handled by extToLang, not here.
func resolveGrammar(langName string) (func() any, bool) {
	// DetectLanguage resolves by filename; we resolve by the canonical name
	// through a small alias table so the registry's filename-based lookup
	// returns the right entry.
	probe, ok := grammarProbeFile[langName]
	if !ok {
		return nil, false
	}
	entry := grammars.DetectLanguage(probe)
	if entry == nil {
		return nil, false
	}
	return func() any { return entry.Language() }, true
}

// grammarProbeFile maps each supported language to a filename whose extension
// the gotreesitter registry resolves to that language.
var grammarProbeFile = map[string]string{
	"go":           "a.go",
	"typescript":   "a.ts",
	"tsx":          "a.tsx",
	"javascript":   "a.js",
	"python":       "a.py",
	"rust":         "a.rs",
	"java":         "a.java",
	"c":            "a.c",
	"cpp":          "a.cpp",
	"c_sharp":      "a.cs",
	"php":          "a.php",
	"ruby":         "a.rb",
	"kotlin":       "a.kt",
	"swift":        "a.swift",
	"scala":        "a.scala",
	"dart":         "a.dart",
	"lua":          "a.lua",
	"r":            "a.r",
	"matlab":       "a.m",
	"perl":         "a.pl",
	"elixir":       "a.ex",
	"haskell":      "a.hs",
	"clojure":      "a.clj",
	"zig":          "a.zig",
	"nim":          "a.nim",
	"groovy":       "a.groovy",
	"objective_c":  "a.m",
	"bash":         "a.sh",
	"sql":          "a.sql",
	"css":          "a.css",
	"html":         "a.html",
}

// treeResult is the parsed tree plus the language name used to parse it.
type treeResult struct {
	tree *ts.Tree
	lang string
}

// parseSource parses src with the language's grammar and returns the tree. It
// panics on a grammar crash; the caller recovers and falls back.
func parseSource(load func() any, src []byte) treeResult {
	lang := load().(*ts.Language)
	parser := ts.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		// A parse error is treated as malformed: return a nil tree so the
		// caller falls back.
		return treeResult{}
	}
	return treeResult{tree: tree, lang: lang.Name}
}

// parseOK reports whether the parse succeeded and did not stop early.
func (r treeResult) parseOK() bool {
	if r.tree == nil || r.lang == "" {
		return false
	}
	return !r.tree.ParseStoppedEarly()
}
