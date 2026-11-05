package kb

import (
	"fmt"
	"strings"
)

// pages.go — vault path rules shared by the indexer, the plan validator, and
// the reserved-root-page check (001-structure: page-type directories + the
// structural root pages index.md / log.md / reverse-index.md).

// ReservedRootPages are structural root pages (not plan/validate targets).
var ReservedRootPages = map[string]bool{
	"index.md": true, "log.md": true, "reverse-index.md": true,
	VaultSchemaFile: true, VaultConfigFile: true,
}

// IsPageTypeDir reports whether dir is a vault page-type folder.
func IsPageTypeDir(dir string) bool {
	for _, d := range PageTypeDirs {
		if d == dir {
			return true
		}
	}
	return false
}

// ValidatePagePath enforces the plan path contract: <type-folder>/<name>.md,
// no traversal, .md only.
func ValidatePagePath(relPath string) error {
	if relPath == "" || strings.Contains(relPath, "..") || strings.Contains(relPath, ":") {
		return fmt.Errorf("kb: invalid page path %q", relPath)
	}
	parts := strings.Split(relPath, "/")
	if len(parts) != 2 || !strings.HasSuffix(parts[1], ".md") {
		return fmt.Errorf("kb: page path must be <type-folder>/<name>.md: %q", relPath)
	}
	if !IsPageTypeDir(parts[0]) {
		return fmt.Errorf("kb: unknown page-type folder %q", parts[0])
	}
	return nil
}
