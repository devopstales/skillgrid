package pdg

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
}
