package extract

import (
	"path/filepath"
	"strings"
)

// SupportedLanguages is the 30-language scope (the 6 rich + the 25-row map
// table; the c→cpp alias means 24 distinct maps + 1 alias = 25 rows).
var SupportedLanguages = []string{
	"go", "typescript", "tsx", "javascript", "python", "rust", "java", "c",
	"cpp", "c_sharp", "php", "ruby", "kotlin", "swift", "scala", "dart",
	"lua", "r", "matlab", "perl", "elixir", "haskell", "clojure", "zig",
	"nim", "groovy", "objective_c", "bash", "sql", "css",
}

// LanguageSet is the fast-lookup form of SupportedLanguages.
var LanguageSet = func() map[string]bool {
	m := make(map[string]bool, len(SupportedLanguages))
	for _, l := range SupportedLanguages {
		m[l] = true
	}
	return m
}()

// extToLang maps file extensions to their supported language. Grammar reuse is
// applied here (e.g. .mts/.cts → typescript, .cu/.cuh → cpp).
var extToLang = map[string]string{
	".go":     "go",
	".ts":     "typescript",
	".mts":    "typescript",
	".cts":    "typescript",
	".tsx":    "tsx",
	".js":     "javascript",
	".mjs":    "javascript",
	".cjs":    "javascript",
	".py":     "python",
	".rs":     "rust",
	".java":   "java",
	".c":      "c",
	".h":      "c",
	".cpp":    "cpp",
	".cc":     "cpp",
	".cxx":    "cpp",
	".cu":     "cpp",
	".cuh":    "cpp",
	".hpp":    "cpp",
	".hh":     "cpp",
	".cs":     "c_sharp",
	".php":    "php",
	".rb":     "ruby",
	".kt":     "kotlin",
	".kts":    "kotlin",
	".swift":  "swift",
	".scala":  "scala",
	".dart":   "dart",
	".lua":    "lua",
	".r":      "r",
	".R":      "r",
	".m":      "matlab",
	".pl":     "perl",
	".pm":     "perl",
	".ex":     "elixir",
	".exs":    "elixir",
	".hs":     "haskell",
	".clj":    "clojure",
	".cljs":   "clojure",
	".cljc":   "clojure",
	".zig":    "zig",
	".nim":    "nim",
	".groovy": "groovy",
	".gvy":    "groovy",
	".sh":     "bash",
	".bash":   "bash",
	".sql":    "sql",
	".css":    "css",
	".html":   "html",
	".htm":    "html",
}

// DetectLanguage returns the supported language for a file path, or "" if the
// extension is not in the 30-language scope. Grammar reuse maps are applied
// (e.g. .mts → typescript, .cu → cpp).
func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return ""
	}
	lang, ok := extToLang[ext]
	if !ok {
		return ""
	}
	if !LanguageSet[lang] {
		return ""
	}
	return lang
}
