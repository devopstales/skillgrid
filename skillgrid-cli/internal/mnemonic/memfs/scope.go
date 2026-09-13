package memfs

import (
	"fmt"
	"strings"
)

// ScopeFilter is the structured result of parsing a mem:// URI or a bare
// scope path. It identifies the scope kind (project or user), the scope ID,
// and optionally a memory_type directory under that scope.
type ScopeFilter struct {
	Kind       string // "project" or "user"
	ID         string // the project or user identifier
	MemoryType string // optional directory (memory_type) under the scope; "" = all
}

// Prefix returns the topic_key prefix this filter matches:
//   - with MemoryType set: "project/A/preferences/" (matches children) or
//     exactly "project/A/preferences"
//   - without MemoryType:  "project/A/" (all children under A)
func (f ScopeFilter) Prefix() string {
	if f.MemoryType != "" {
		return f.Kind + "/" + f.ID + "/" + f.MemoryType
	}
	return f.Kind + "/" + f.ID
}

// ResolveURI parses a mem:// URI into a ScopeFilter. Accepted forms:
//
//	mem://project/{id}/            → {Kind: "project", ID: id}
//	mem://project/{id}/{memType}   → {Kind: "project", ID: id, MemoryType: memType}
//	mem://user/{id}/               → {Kind: "user", ID: id}
//	mem://user/{id}/{memType}      → {Kind: "user", ID: id, MemoryType: memType}
//
// A bare path without the mem:// prefix is also accepted (the CLI passes
// positional args without the scheme):
//
//	project/A/            → {Kind: "project", ID: "A"}
//	project/A/preferences → {Kind: "project", ID: "A", MemoryType: "preferences"}
//	user/B/               → {Kind: "user", ID: "B"}
//
// Returns an error for malformed URIs.
func ResolveURI(uri string) (ScopeFilter, error) {
	s := strings.TrimSpace(uri)
	if s == "" {
		return ScopeFilter{}, ErrEmptyScope
	}
	// Strip the mem:// scheme if present.
	s = strings.TrimPrefix(s, "mem://")
	if s == "" {
		return ScopeFilter{}, fmt.Errorf("memfs: URI %q has no scope path after scheme", uri)
	}
	// Normalize: strip trailing slash for segment parsing.
	segs := splitPath(s)
	if len(segs) < 2 {
		return ScopeFilter{}, fmt.Errorf("memfs: scope path %q must have at least kind/{id}", uri)
	}
	kind := segs[0]
	if kind != "project" && kind != "user" {
		return ScopeFilter{}, fmt.Errorf("memfs: unknown scope kind %q (expected \"project\" or \"user\")", kind)
	}
	f := ScopeFilter{Kind: kind, ID: segs[1]}
	if len(segs) >= 3 {
		f.MemoryType = segs[2]
	}
	return f, nil
}

// splitPath splits a scope path into non-empty segments on "/".
func splitPath(p string) []string {
	var out []string
	for _, s := range strings.Split(p, "/") {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}
