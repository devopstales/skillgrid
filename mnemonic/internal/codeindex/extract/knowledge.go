package extract

import (
	"path/filepath"
	"strings"
)

// Classify maps a file path to its knowledge-node kind (spec 003 §5 — the
// knowledge nodes: route/doc/config/sql). It is a path-shape classifier:
//
//   - .md            → "doc"
//   - .yaml/.yml/.toml/.json → "config"
//   - .sql           → "table"
//
// An unmapped extension returns ok=false (skipped, NOT classified — a .xyz or a
// README.sh is a knowledge node only if its extension maps; "README.sh" is a
// script, not a doc). The content argument is accepted for future content-based
// classification (e.g. a route file detected by its body) but the current
// classifier is extension-driven.
func Classify(path, content string) (kind string, ok bool) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md":
		return "doc", true
	case ".yaml", ".yml", ".toml", ".json":
		return "config", true
	case ".sql":
		return "table", true
	default:
		return "", false
	}
}
