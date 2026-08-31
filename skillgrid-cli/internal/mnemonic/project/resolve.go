// Package project resolves a stable, filesystem-safe project ID for a
// working directory. Resolution order: nearest .skillgrid/config.json
// "project" field, then git remote origin repo basename, then a
// {basename}-{hash} fallback.
package project

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Resolve determines the project ID for cwd.
func Resolve(cwd string) (string, error) {
	id, _, err := ResolveDetailed(cwd)
	return id, err
}

// ResolveSource is the source of a project ID (e.g. "config", "git", "fallback").
type ResolveSource string

const (
	SourceConfig   ResolveSource = "config"
	SourceGit      ResolveSource = "git-remote"
	SourceFallback ResolveSource = "directory-hash"
)

// ResolveDetailed returns the project ID for cwd and its *source* (config,
// git remote, or directory-hash fallback). Agents can surface this to the
// user when they suspect a project-name collision.
func ResolveDetailed(cwd string) (id string, src ResolveSource, err error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", "", err
	}
	if id, ok := configProject(abs); ok {
		return normalizeProjectID(id), SourceConfig, nil
	}
	if id, ok := gitRemoteProject(abs); ok {
		return normalizeProjectID(id), SourceGit, nil
	}
	return fallbackProjectID(abs), SourceFallback, nil
}

func configProject(cwd string) (string, bool) {
	dir := cwd
	for {
		cfgPath := filepath.Join(dir, ".skillgrid", "config.json")
		if data, err := os.ReadFile(cfgPath); err == nil {
			var cfg struct {
				Project string `json:"project"`
			}
			if json.Unmarshal(data, &cfg) == nil && strings.TrimSpace(cfg.Project) != "" {
				return cfg.Project, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func gitRemoteProject(cwd string) (string, bool) {
	root, err := gitOutput(cwd, "rev-parse", "--show-toplevel")
	if err != nil || root == "" {
		return "", false
	}
	remote, err := gitOutput(root, "remote", "get-url", "origin")
	if err != nil || remote == "" {
		return "", false
	}
	id, ok := normalizeRemoteURL(remote)
	return id, ok
}

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

func normalizeRemoteURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	// SCP-style: git@github.com:owner/repo.git
	if strings.Contains(raw, "@") && strings.Contains(raw, ":") && !strings.Contains(raw, "://") {
		parts := strings.SplitN(raw, ":", 2)
		host := strings.TrimPrefix(parts[0], "git@")
		path := strings.TrimSuffix(parts[1], ".git")
		path = strings.Trim(path, "/")
		if host == "" || path == "" {
			return "", false
		}
		return repoNameFromPath(path), true
	}
	if idx := strings.Index(raw, "://"); idx >= 0 {
		raw = raw[idx+3:]
	}
	raw = strings.TrimPrefix(raw, "git@")
	raw = strings.TrimSuffix(raw, ".git")
	raw = strings.Trim(raw, "/")
	if at := strings.Index(raw, "@"); at >= 0 {
		raw = raw[at+1:]
	}
	if raw == "" || !strings.Contains(raw, "/") {
		return "", false
	}
	return repoNameFromPath(raw), true
}

func repoNameFromPath(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	return normalizeProjectID(name)
}

func fallbackProjectID(absCWD string) string {
	base := filepath.Base(absCWD)
	sum := sha256.Sum256([]byte(absCWD))
	hash := hex.EncodeToString(sum[:])[:8]
	return normalizeProjectID(base) + "-" + hash
}
