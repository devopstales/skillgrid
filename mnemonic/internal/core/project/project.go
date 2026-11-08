// Package project resolves which project a mnemonic call targets (spec
// 006-config §2, spec 005-transport-cli §3, the Engram hostbind pattern).
// Detection is a pure function of (cwd, explicit selector, env, git toplevel)
// so every selector case is unit-testable without a real git repository.
package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrAllWithProject is the CLI usage error: an explicit project selector
// combined with --all (spec 005-transport-cli §3 — "Don't combine an explicit
// project with --all").
var ErrAllWithProject = errors.New("project: an explicit project cannot be combined with --all")

// Detector resolves the current project. EnvVar is the env override name
// (default "MNEMONIC_PROJECT"); the config's ProjectCfg.EnvVar feeds it.
type Detector struct {
	EnvVar string
}

// GitToplevel reports the git working tree's top-level directory for dir, or
// an error when dir is not inside a git repository. Injected so tests do not
// need a real repository (the production implementation shells out to git).
type GitToplevel func(dir string) (string, error)

// Resolve walks the detection chain: explicit → env → cwd git binding →
// basename(cwd). The git binding uses the basename of the *toplevel* (a cwd
// deep in the tree resolves to the repo name, the Engram hostbind rule). An
// empty result is an error — there is no "guess" fallback. A failed git
// lookup is a signal (not a failure): detection falls through to the basename.
func (d Detector) Resolve(cwd, explicit string, git GitToplevel) (string, error) {
	name, err := d.ResolveWithAll(cwd, explicit, false, git)
	if err != nil {
		return "", err
	}
	if name == "" {
		return "", fmt.Errorf("project: cannot resolve a project for cwd %q (set %s or pass an explicit project)", cwd, d.EnvVar)
	}
	return name, nil
}

// ResolveWithAll is Resolve plus the --all usage check. all=true with an
// explicit project is ErrAllWithProject; all=true otherwise resolves to ""
// (the "every project" marker the CLI/handler turns into per-project fan-out).
func (d Detector) ResolveWithAll(cwd, explicit string, all bool, git GitToplevel) (string, error) {
	if all && explicit != "" {
		return "", ErrAllWithProject
	}
	if explicit != "" {
		return explicit, nil
	}
	if v := strings.TrimSpace(os.Getenv(d.EnvVar)); v != "" {
		return v, nil
	}
	if all {
		return "", nil
	}
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return "", errors.New("project: cwd is empty and no project selector was given")
	}
	// The git binding uses the toplevel's basename (hostbind). A lookup
	// error is a signal, not a failure — fall through to basename(cwd).
	// cwd == "/" is the one cwd with no basename: it resolves only via the
	// git binding, otherwise detection fails (an empty project is a worse
	// error than a loud one — the store rejects it at Open anyway).
	if top, err := git(cwd); err == nil && strings.TrimSpace(top) != "" {
		return filepath.Base(top), nil
	}
	if base := filepath.Base(cwd); base != "" && base != "/" {
		return base, nil
	}
	return "", errors.New("project: cwd is the filesystem root and no project selector was given")
}

// Canonicalize normalizes a project name for the on-disk and remote schema
// boundary (the `schema_<sanitized_project>` naming depends on it — Task 18):
// lowercase, '/' → '-', leading separators stripped, then trailing
// separators (and the '-' a trailing slash left behind) trimmed.
//
// ponytail: only '/' is a separator (POSIX platform floor; a Windows-aware
// render would also map '\\' → '-' — the store rejects ".." names, so
// traversal is the caller's trust boundary, not this function's).
func Canonicalize(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "/", "-")
	return strings.TrimSuffix(strings.TrimLeft(name, "/"), "-")
}
