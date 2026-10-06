package memfs

import (
	"errors"
	"fmt"
	"strings"
)

// CodePath is a resolved location in the code-index namespace.
//   - Kind "dir":    Dir is the directory prefix ("" = repo root).
//   - Kind "file":   Dir + File identify one indexed file.
//   - Kind "symbol": Dir + File + Symbol identify one symbol in a file.
type CodePath struct {
	Kind   string // "dir" | "file" | "symbol"
	Dir    string // directory prefix, no leading/trailing slash ("" = root)
	File   string // basename (file/symbol kinds)
	Symbol string // symbol name (symbol kind)
}

// errCodeScopeMismatch is returned when the project id in a code URI does not
// equal the MemFS project id.
var errCodeScopeMismatch = errors.New("memfs: project id mismatch")

// ResolveCodePath parses a code-index path into a CodePath. Accepted forms:
//
//	"" or "."                       → repo root
//	"src/"                          → directory src/
//	"src/auth/login.go"             → file
//	"src/auth/login.go::Handler"    → symbol in file
//	"memfs://project/A/src/..."     → same, with scheme + project id
//	"A/src/..."                     → bare project id + path
//
// When an explicit memfs://project/{id}/ scheme is present, the id must equal
// memfsProjectID, else errCodeScopeMismatch. A bare path (no scheme) whose
// first segment happens to equal the project id is treated as a repo segment.
func ResolveCodePath(memfsProjectID, raw string) (CodePath, error) {
	s := strings.TrimSpace(raw)
	explicit := strings.HasPrefix(s, "memfs://")
	s = strings.TrimPrefix(s, "memfs://")
	// "." (or empty) means the repo root.
	if s == "" || s == "." {
		return CodePath{Kind: "dir", Dir: ""}, nil
	}
	segs := splitPath(s)
	// With the explicit memfs://project/{id}/... form, the path begins with the
	// literal "project" segment; validate {id} against the bound project and
	// drop both. A bare path (no scheme) is always project-relative: its first
	// segment is a repo directory, never a project id. This keeps the explicit
	// and bare forms consistent — only the explicit URI carries a project id to
	// validate (see review #2).
	if explicit && len(segs) >= 2 && segs[0] == "project" {
		if segs[1] != memfsProjectID {
			return CodePath{}, errCodeScopeMismatch
		}
		segs = segs[2:]
	}
	if len(segs) == 0 {
		return CodePath{Kind: "dir", Dir: ""}, nil
	}
	// Symbol: the last segment contains "::".
	if i := strings.LastIndex(segs[len(segs)-1], "::"); i >= 0 {
		last := segs[len(segs)-1]
		fileSeg := last[:i]
		sym := last[i+2:]
		if sym == "" {
			return CodePath{}, fmt.Errorf("memfs: empty symbol name in %q", raw)
		}
		dir := ""
		if len(segs) > 1 {
			dir = strings.Join(segs[:len(segs)-1], "/")
		}
		return CodePath{Kind: "symbol", Dir: dir, File: fileSeg, Symbol: sym}, nil
	}
	// A trailing slash marks a directory at any depth.
	if strings.HasSuffix(s, "/") {
		return CodePath{Kind: "dir", Dir: strings.Join(segs, "/")}, nil
	}
	if len(segs) == 1 {
		return CodePath{Kind: "file", Dir: "", File: segs[0]}, nil
	}
	dir := strings.Join(segs[:len(segs)-1], "/")
	file := segs[len(segs)-1]
	return CodePath{Kind: "file", Dir: dir, File: file}, nil
}

// splitPath splits a path into non-empty segments on "/".
func splitPath(p string) []string {
	var out []string
	for _, s := range strings.Split(p, "/") {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}
