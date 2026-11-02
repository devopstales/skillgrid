package graph

import (
	"context"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// PackageInfo is one top-level Go package (spec 004 §4).
type PackageInfo struct {
	Path    string
	Files   int
	Symbols int
}

// Packages returns the top-level Go package map (spec 004 §4).
// The unit is the top-level Go package (the first path segment under the
// project root that contains Go files). Sub-packages and files are not the unit.
func Packages(ctx context.Context, st store.Store) ([]PackageInfo, error) {
	files, err := st.FilesByLanguage(ctx, "go")
	if err != nil {
		return nil, err
	}

	// Group by top-level package path
	pkgs := map[string]*PackageInfo{}
	for _, f := range files {
		pkgPath := topLevelPackagePath(f.Path)
		if pkgPath == "" {
			continue
		}
		if pkgs[pkgPath] == nil {
			pkgs[pkgPath] = &PackageInfo{Path: pkgPath}
		}
		pkgs[pkgPath].Files++
	}

	// Count symbols per package
	for pkgPath, pkg := range pkgs {
		// Count symbols in files belonging to this package
		symbolCount := 0
		for _, f := range files {
			if topLevelPackagePath(f.Path) != pkgPath {
				continue
			}
			syms, err := st.SymbolsByFile(ctx, f.ID)
			if err != nil {
				continue
			}
			symbolCount += len(syms)
		}
		pkg.Symbols = symbolCount
	}

	// Convert to slice and sort by path
	var out []PackageInfo
	for _, p := range pkgs {
		out = append(out, *p)
	}
	sortPackages(out)

	// Record the trail
	RecordTrail(ctx, st, "code_packages", "", "")

	return out, nil
}

// topLevelPackagePath extracts the top-level package path from a file path.
// For "internal/memory/file.go" → "internal/memory".
// For "cmd/mnemonic/main.go" → "cmd/mnemonic".
// For "file.go" → "" (no package).
func topLevelPackagePath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return ""
	}
	// The top-level package is the first segment (or first two for cmd/xxx)
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return parts[0]
}

func sortPackages(pkgs []PackageInfo) {
	for i := 1; i < len(pkgs); i++ {
		for j := i; j > 0 && pkgs[j].Path < pkgs[j-1].Path; j-- {
			pkgs[j], pkgs[j-1] = pkgs[j-1], pkgs[j]
		}
	}
}
