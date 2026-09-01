// Package project resolves a stable, filesystem-safe project ID for a working
// directory.
//
// Resolution mirrors the proven Engram algorithm (engram/internal/project) so
// that Mnemonic and Engram agree on which memory bucket a directory belongs to:
//
//  0. override   — the MNEMONIC_PROJECT environment variable (process override)
//  1. config     — nearest .skillgrid/config.json "project", bounded to the
//     enclosing repo root (or the cwd outside git)
//  2. identity   — a clone-private binding written once into the git common
//     dir (survives rename, re-clone, remote change, and shared by linked
//     worktrees)
//  3. git-child  — cwd is exactly one git repo's parent (auto-promoted)
//  4. ambiguous  — cwd is the parent of several git repos; callers receive an
//     AmbiguousProjectError with the candidate list
//  5. directory  — {basename}-{hash} fallback for directories with no git
//     repository underneath
//
// The old path-only {basename}-{hash} resolution is still returned as a
// fallback so an ambiguous parent directory never yields an empty ID.
package project

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Source constants describe how the project name was resolved. They are part
// of the user-facing surface (mem_current_project exposes them) so do not
// rename without bumping the protocol.
type ResolveSource string

const (
	SourceConfig   ResolveSource = "config"
	SourceGit      ResolveSource = "git-remote"
	SourceIdentity ResolveSource = "identity"
	SourceChild    ResolveSource = "git-child"
	SourceAmbig    ResolveSource = "ambiguous"
	SourceFallback ResolveSource = "directory-hash"
	SourceOverride ResolveSource = "env-override"
)

// EnvProjectOverride names the process-level project override (mnemonic
// counterpart of Engram's ENGRAM_PROJECT). Callers use this to pin a session
// to a specific bucket before falling back to cwd detection.
const EnvProjectOverride = "MNEMONIC_PROJECT"

// identityFilename is the clone-private identity file written once per repo
// into the git common dir. Linked worktrees share it via common-dir, which is
// what makes Mnemonic's bucket stable across checkouts.
const identityFilename = "skillgrid-mnemonic-identity.json"

const bindingVersion = 1

// ErrAmbiguousProject wraps AmbiguousProjectError and reports that the working
// directory is a parent of several git repositories so a single project cannot
// be inferred safely. Callers can errors.Is to detect the case without casting.
var ErrAmbiguousProject = errors.New("ambiguous project: multiple git repositories found in cwd")

// AmbiguousProjectError carries the candidate list so agents can prompt the
// user to pick one.
type AmbiguousProjectError struct {
	Path              string   // directory that produced the ambiguity (for display)
	AvailableProjects []string // normalized project names, in directory order
}

func (e *AmbiguousProjectError) Error() string {
	items := strings.Join(e.AvailableProjects, ", ")
	if items == "" {
		items = "(none scanned — timed out)"
	}
	return fmt.Sprintf("%s (candidates: %s)", ErrAmbiguousProject, items)
}

func (e *AmbiguousProjectError) Unwrap() error { return ErrAmbiguousProject }

// Resolution is the full result of ResolveDetailed.
type Resolution struct {
	ID     string        // project id (always non-empty except on error)
	Source ResolveSource // how the id was derived
	Path   string        // directory resolved to (repo root for git cases, cwd for dir case)
	// Available is non-empty only when the case is ambiguous; callers can
	// surface it to the user and retry with one of the names via
	// MNEMONIC_PROJECT or a mem_save project= argument.
	Available []string
	// Warning is a soft advisory (e.g. "auto-promoted child repository: x");
	// empty for all other branches.
	Warning string
	// SeedID, when non-empty, is a legacy directory-hash ID that existed before
	// this binding was created. Services can optionally alias that ID to the
	// new identity ID to consolidate prior memory writes.
	SeedID string
	// Err is set when resolution cannot proceed cleanly (currently only
	// on AmbiguousProjectError).
	Err error
}

// Resolve returns the project id for cwd, always non-empty. On ambiguity it
// falls back to the directory-hash so callers (like Engram's DetectProject
// wrapper) never receive an empty string. For a clean API surface use
// ResolveDetailed.
func Resolve(cwd string) (string, error) {
	r, err := ResolveDetailed(cwd)
	if err != nil {
		// Compatibility wrapper: surface the ID when the error carries one.
		if varErr, ok := err.(*AmbiguousProjectError); ok && varErr != nil {
			return r.ID, err
		}
		return r.ID, err
	}
	return r.ID, nil
}

// ProcessOverride returns the project override from the MNEMONIC_PROJECT
// environment variable when set and non-empty.
func ProcessOverride() (string, bool) {
	v := strings.TrimSpace(os.Getenv(EnvProjectOverride))
	if v == "" {
		return "", false
	}
	return v, true
}

// NormalizeID returns the canonical, filesystem-safe form of a project name
// (lowercased, `[-_]+` collapsed to `-`, trimmed of leading/trailing dashes).
// Use this when accepting an explicit project name from a caller so it matches
// the store-file and row naming used by Resolve.
func NormalizeID(id string) string {
	return normalizeProjectID(id)
}

// LegacyFallbackID is the directory-hash ID for absDir, exposed so the service
// layer can alias the old id when a stable identity is being introduced on top
// of previously-saved directory-hash memories.
func LegacyFallbackID(absDir string) string {
	return fallbackProjectID(absDir)
}

// ResolveDetailed returns the full resolution for cwd and its source. It never
// returns an empty ID — ambiguous cases include a fallback ID plus an error
// with the candidate list populated.
func ResolveDetailed(cwd string) (Resolution, error) {
	if strings.TrimSpace(cwd) == "" {
		cwd = "."
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return Resolution{}, err
	}
	if strings.HasPrefix(cwd, "-") {
		cwd = "./" + cwd
	}

	// 0. Process-level override — caller pinned a project.
	if override, ok := ProcessOverride(); ok {
		id := normalizeProjectID(override)
		if id == "" {
			id = "unknown"
		}
		return Resolution{ID: id, Source: SourceOverride, Path: abs}, nil
	}

	// 1. Config within the enclosing repo (bounded walk).
	if repoRoot := gitToplevel(abs); repoRoot != "" {
		if id, ok, err := configProjectBounded(abs, repoRoot); err == nil && ok {
			return Resolution{ID: normalizeProjectID(id), Source: SourceConfig, Path: abs}, nil
		}
	} else if id, ok := configProjectAt(abs); ok {
		return Resolution{ID: normalizeProjectID(id), Source: SourceConfig, Path: abs}, nil
	}

	// 2. Clone-private identity binding in the git common dir — survives move,
	// re-clone, remote change, and shared across linked worktrees.
	if res, ok := identityBinding(abs); ok {
		return res, nil
	}

	// 3. Exactly one git child repo — auto-promote.
	// 4. Several child repos — ambiguous with the full candidate list.
	if children, scanned := scanChildren(abs); scanned {
		switch len(children) {
		case 1:
			name := normalizeProjectID(filepath.Base(children[0]))
			return Resolution{
				ID:      name,
				Source:  SourceChild,
				Path:    children[0],
				Warning: "auto-promoted child repository: " + name,
			}, nil
		default:
			if len(children) > 1 {
				names := make([]string, 0, len(children))
				for _, c := range children {
					names = append(names, normalizeProjectID(filepath.Base(c)))
				}
				fallback := fallbackProjectID(abs)
				res := Resolution{
					ID:        fallback,
					Source:    SourceAmbig,
					Path:      abs,
					Available: names,
					SeedID:    fallback,
				}
				return res, &AmbiguousProjectError{Path: abs, AvailableProjects: names}
			}
		}
	}

	// 5. Fallback directory-hash (no git anywhere in the picture).
	return Resolution{
		ID:     fallbackProjectID(abs),
		Source: SourceFallback,
		Path:   abs,
	}, nil
}

// identityBinding reads (or creates on first call) the clone-private identity
// binding in the git common dir of abs. Returns ok=false when abs (or any
// ancestor) is not a git repo or the binding cannot be established.
func identityBinding(path string) (Resolution, bool) {
	commonDir := gitCommonDir(path)
	if commonDir == "" {
		return Resolution{}, false
	}
	if res, ok := readBinding(commonDir); ok {
		return res, true
	}

	// First time we see this repo — seed from the git remote origin, or the
	// repo-root basename when there is no origin (local-only checkouts).
	seed := ""
	src := SourceIdentity
	if remote := gitRemoteOrigin(path); remote != "" {
		if name := repoNameFromRemote(remote); name != "" {
			seed = normalizeProjectID(name)
		}
	}
	if seed == "" {
		if root := gitToplevel(path); root != "" {
			seed = normalizeProjectID(filepath.Base(root))
		}
	}
	if seed == "" {
		seed = normalizeProjectID(filepath.Base(path))
		if seed == "" {
			seed = "unknown"
		}
	}

	id, err := writeBinding(commonDir, seed)
	if err != nil {
		// Best-effort path: when we cannot write the binding (no write
		// permission into .git, read-only FS, etc.) fall back to the seed so
		// the caller still gets a usable project ID. The next call retries
		// the write.
		return Resolution{ID: seed, Source: src, Path: path}, true
	}

	// Seed id for the service layer to alias any pre-existing store under a
	// legacy directory-hash key if the caller cares about continuity.
	legacy := fallbackProjectID(path)
	return Resolution{ID: id, Source: src, Path: path, SeedID: legacy}, true
}

// Binding is the on-disk identity file format (version 1).
type Binding struct {
	Version int    `json:"version"`
	ID      string `json:"id"`      // stable clone-private random id (16 bytes hex)
	Project string `json:"project"` // human-facing project name
}

// writeBinding atomically writes the identity file into commonDir, returning
// the project name it recorded. If the file already exists (lost race), the
// existing project name is returned.
func writeBinding(commonDir, project string) (string, error) {
	binding := Binding{Version: bindingVersion, ID: newRandomID(), Project: project}
	data, err := json.MarshalIndent(binding, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	target := filepath.Join(commonDir, identityFilename)
	tmp := target + ".tmp-" + binding.ID
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return "", err
	}
	if err := os.Link(tmp, target); err == nil {
		// Linked — tmp and target are now the same inode; remove the temp
		// name only, not the file.
		if rmErr := os.Remove(tmp); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			// best-effort; the file is valid at target
			_ = rmErr
		}
		return project, nil
	}
	if !errors.Is(err, os.ErrExist) {
		_ = os.Remove(tmp)
		return "", err
	}
	// Already there (raced). Read the existing binding so the caller sees the
	// name that already owns the repo.
	path := filepath.Join(commonDir, identityFilename)
	if existing, err := jsonReadBinding(path); err == nil {
		return existing.Project, nil
	}
	// Fall back to a name — we still know what we meant to store.
	return project, nil
}

// readBinding loads the identity file when present and valid.
func readBinding(commonDir string) (Resolution, bool) {
	data, err := os.ReadFile(filepath.Join(commonDir, identityFilename))
	if err != nil {
		return Resolution{}, false
	}
	var b Binding
	if err := json.Unmarshal(data, &b); err != nil {
		return Resolution{}, false
	}
	if b.Version != bindingVersion || b.Project == "" {
		return Resolution{}, false
	}
	return Resolution{ID: normalizeProjectID(b.Project), Source: SourceIdentity}, true
}

func jsonReadBinding(path string) (Binding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Binding{}, err
	}
	var b Binding
	if err := json.Unmarshal(data, &b); err != nil {
		return Binding{}, err
	}
	return b, nil
}

// newRandomID returns 32-hex bytes of fresh entropy for the binding ID.
func newRandomID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// Fallback — should not happen, but keep the file shape valid.
		now := uint64(time.Now().UnixNano())
		for i := 0; i < len(buf); i++ {
			buf[i] = byte(now >> (i * 8))
			now = now*6364136223846793005 + 1442695040888963407
		}
	}
	return hex.EncodeToString(buf)
}

// config walk helpers — bounded to the enclosing repo root (or cwd) to stop
// an ancestor .skillgrid/config.json claiming an unrelated checkout.
func configProjectAt(dir string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(dir, ".skillgrid", "config.json"))
	if err != nil {
		return "", false
	}
	var cfg struct {
		Project string `json:"project"`
	}
	if json.Unmarshal(data, &cfg) != nil || strings.TrimSpace(cfg.Project) == "" {
		return "", false
	}
	return cfg.Project, true
}

func configProjectBounded(start, stop string) (string, bool, error) {
	current := filepath.Clean(start)
	stopC := filepath.Clean(stop)
	for {
		if id, ok := configProjectAt(current); ok {
			return id, true, nil
		}
		if current == stopC {
			return "", false, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false, nil
		}
		current = parent
	}
}

// gitToplevel, gitCommonDir, gitRemoteOrigin, gitOutput are thin wrappers
// around `git rev-parse` / `git remote` so the test suite can monkey-patch the
// exec path via a single indirection.
var (
	gitToplevel = func(dir string) string {
		p, err := gitOutput(dir, "rev-parse", "--path-format=absolute", "--show-toplevel")
		if err != nil {
			return ""
		}
		return p
	}
	gitCommonDir = func(dir string) string {
		p, err := gitOutput(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
		if err != nil || p == "" {
			return ""
		}
		if !filepath.IsAbs(p) {
			// `--git-common-dir` may return a path relative to the cwd.
			p = filepath.Join(dir, p)
		}
		return filepath.Clean(p)
	}
	gitRemoteOrigin = func(dir string) string {
		u, err := gitOutput(dir, "remote", "get-url", "origin")
		if err != nil {
			return ""
		}
		return u
	}
)

// gitOutput runs a git command in dir and returns trimmed stdout. Errors are
// returned to the caller (no panic on missing git / non-repo directory).
func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

var multiSep = regexp.MustCompile(`[-_]+`)

func normalizeProjectID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	id = multiSep.ReplaceAllString(id, "-")
	return strings.Trim(id, "-")
}

// repoNameFromRemote parses the trailing path segment of a git remote URL —
// the only thing we need to seed a clone identity. Both SSH (git@host:owner/
// repo.git) and HTTPS (https://host/owner/repo.git) forms.
func repoNameFromRemote(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")
	if raw == "" {
		return ""
	}
	// SCP-form: split on ':' then take the last '/' segment.
	if strings.Contains(raw, ":") && strings.Contains(raw, "@") {
		parts := strings.SplitN(raw, ":", 2)
		raw = parts[1]
	}
	if idx := strings.Index(raw, "://"); idx >= 0 {
		raw = raw[idx+3:]
	}
	raw = strings.Trim(raw, "/")
	if raw == "" || !strings.Contains(raw, "/") {
		return ""
	}
	return normalizeProjectID(repoPathLastSegment(raw))
}

func repoPathLastSegment(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func fallbackProjectID(absCWD string) string {
	base := filepath.Base(absCWD)
	sum := sha256.Sum256([]byte(absCWD))
	hash := hex.EncodeToString(sum[:])[:8]
	return normalizeProjectID(base) + "-" + hash
}

// noiseDirs are skipped when scanning a parent directory for child repos. The
// set mirrors Engram's (engram/internal/project/detect.go:75).
var noiseDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	".venv":        true,
	".cache":       true,
	"__pycache__":  true,
	"target":       true,
	"dist":         true,
	"build":        true,
	".idea":        true,
	".vscode":      true,
}

// childScanTimeout bounds scanChildren. Long enough that even a slow NFS
// mount enumerates 2000 files; short enough that the CLI does not stall.
const childScanTimeout = 200 * time.Millisecond

// childScanNow lets tests inject time. Production uses the wall clock.
var childScanNow = time.Now

// scanChildren enumerates all first-level subdirectories of dir that contain
// a .git file, bounded by childScanTimeout. It does NOT short-circuit at two
// repos: when cwd is a parent of several checkouts we want the complete
// candidate list so the ambiguity result is actionable. The timeout (plus the
// one-entry-at-a-time ReadDir) bounds cost even for a very large directory.
//
// The returned `scanned` is false when the directory could not be opened (callers
// fall through to the legacy fallback rather than claiming ambiguity); it is
// true on a clean or time-bounded scan.
func scanChildren(dir string) (repos []string, scanned bool) {
	deadline := childScanNow().Add(childScanTimeout)
	directory, err := os.Open(dir)
	if err != nil {
		return nil, false
	}
	defer directory.Close()

	sawEntries := false
	for {
		if childScanNow().After(deadline) {
			return repos, sawEntries
		}
		entries, readErr := directory.ReadDir(1)
		if errors.Is(readErr, io.EOF) {
			return repos, true
		}
		if readErr != nil {
			return repos, sawEntries // stop; report whatever we saw
		}
		if len(entries) == 0 {
			return repos, true
		}
		sawEntries = true
		entry := entries[0]
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || noiseDirs[name] {
			continue
		}
		gitPath := filepath.Join(dir, name, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			repos = append(repos, filepath.Join(dir, name))
		}
	}
}
