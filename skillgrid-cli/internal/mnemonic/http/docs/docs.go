// Package docs serves the repo's SDD change docs (change.md + tasks.md) as
// read-only JSON, sandboxed to docs/skillgrid/changes and docs/skillgrid/archive.
// Reads are clean + prefix-checked: `..`, absolute paths, and names escaping
// the two roots are rejected before any file is opened. Nothing is executed —
// files are rendered as text only.
package docs

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// roots are the only directories the package reads from, relative to the
// server's working directory (the repo root, matching the tracker bridge).
var roots = []string{"docs/skillgrid/changes", "docs/skillgrid/archive"}

type ChangeInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Ticket string `json:"ticket,omitempty"`
}

type ChangeDetail struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Ticket   string `json:"ticket,omitempty"`
	ChangeMD string `json:"change_md"`
	TasksMD  string `json:"tasks_md"`
}

// guard rejects raw paths that still carry dot-segments (`..`, `.`) with 400
// — these are traversal attempts that Go 1.22 ServeMux would otherwise clean
// with a 307 redirect, hiding the attempt. The cleaned form still 404s via
// the name check, so the guard is defense-in-depth, not the only wall.
func guard(w http.ResponseWriter, r *http.Request) bool {
	u := r.URL.Path
	if strings.Contains(u, "/../") || strings.HasSuffix(u, "/..") ||
		strings.Contains(u, "/./") || strings.HasSuffix(u, "/.") {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid change name: path segments must be plain (no . or ..)",
		})
		return false
	}
	return true
}

// NewList returns the GET /docs/changes handler (change list) bound to cwd.
func NewList(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !guard(w, r) {
			return
		}
		infos, err := ListChanges(r.Context(), cwd)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"changes": infos})
	})
}

// NewDetail returns the GET /docs/changes/{name} handler bound to cwd.
func NewDetail(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !guard(w, r) {
			return
		}
		d, err := GetChange(r.Context(), cwd, r.PathValue("name"))
		if err != nil {
			writeJSON(w, statusFor(err), map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, d)
	})
}

// ErrBadName marks a name that escapes the sandbox (→ 400).
var ErrBadName = errors.New("bad change name")

// ErrNotFound marks a name that is clean but does not exist (→ 404).
var ErrNotFound = errors.New("change not found")

func statusFor(err error) int {
	switch {
	case errors.Is(err, ErrBadName):
		return http.StatusBadRequest
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

// resolveRoot picks which sandbox root serves name. A name containing a
// slash is rejected up front (change names are directory names); an unknown
// name is a 404, not a guess.
func resolveRoot(cwd, name string) (string, string, error) {
	if strings.TrimSpace(name) == "" {
		return "", "", &wrapErr{err: ErrNotFound, msg: "change name is required"}
	}
	if strings.ContainsAny(name, `/\`) {
		return "", "", &wrapErr{err: ErrBadName, msg: "change name must be a plain directory name (no slashes)"}
	}
	for _, rel := range roots {
		full := filepath.Join(cwd, rel, name)
		abs, err := filepath.Abs(full)
		if err != nil {
			return "", "", &wrapErr{err: ErrBadName, msg: err.Error()}
		}
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			rootAbs, err := filepath.Abs(filepath.Join(cwd, rel))
			if err != nil {
				return "", "", &wrapErr{err: ErrBadName, msg: err.Error()}
			}
			// Prefix check: the resolved dir must sit exactly under a root.
			relPath, err := filepath.Rel(rootAbs, abs)
			if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) || filepath.IsAbs(relPath) {
				return "", "", &wrapErr{err: ErrBadName, msg: "change name escapes the docs sandbox"}
			}
			return rel, abs, nil
		}
	}
	return "", "", &wrapErr{err: ErrNotFound, msg: "change " + name + " not found"}
}

type wrapErr struct {
	err error
	msg string
}

func (w *wrapErr) Error() string { return w.msg }
func (w *wrapErr) Unwrap() error { return w.err }

var ticketRE = regexp.MustCompile(`(?im)^\*{2}Ticket:\*{2}\s*(.+)$`)
var statusRE = regexp.MustCompile("(?im)^>\\s*\\*{2}STATUS:\\*{2}\\s*`?([a-z][a-z0-9-]+)`?")

// parseMeta extracts the header Ticket: and STATUS: lines from change.md text.
// The Ticket: regex is anchored to the bold `**Ticket:**` header form so the
// DoD checkboxes ("ticket id from `Ticket:`") never match; STATUS: is anchored
// to the `> **STATUS:**` quote-block header.
func parseMeta(changeMD string) (status, ticket string) {
	if m := ticketRE.FindStringSubmatch(changeMD); m != nil {
		ticket = strings.TrimSpace(m[1])
	}
	if m := statusRE.FindStringSubmatch(changeMD); m != nil {
		status = strings.ToLower(strings.TrimSpace(m[1]))
	}
	return status, ticket
}

// isMeaningfulTicket decides whether a parsed Ticket: value is a real tracker
// id. "none" (the header convention for no ticket), empty, and `none` variants
// are treated as no-ticket so the UI renders no link instead of a dead one.
func isMeaningfulTicket(t string) bool {
	v := strings.ToLower(strings.TrimSpace(strings.Trim(strings.TrimSpace(t), "`")))
	if v == "" || v == "none" || strings.HasPrefix(v, "none (") {
		return false
	}
	return true
}

// ListChanges scans both roots and returns one entry per change directory.
func ListChanges(ctx context.Context, cwd string) ([]ChangeInfo, error) {
	var out []ChangeInfo
	for _, rel := range roots {
		abs, err := filepath.Abs(filepath.Join(cwd, rel))
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(abs)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			info := ChangeInfo{Name: e.Name(), Status: "unknown"}
			if md, err := os.ReadFile(filepath.Join(abs, e.Name(), "change.md")); err == nil {
				info.Status, info.Ticket = parseMeta(string(md))
				if !isMeaningfulTicket(info.Ticket) {
					info.Ticket = ""
				}
			}
			out = append(out, info)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return dedupeByName(out), nil
}

// dedupeByName keeps the first entry per change name (changes root is scanned
// before archive, so an active change wins over its archived twin).
func dedupeByName(in []ChangeInfo) []ChangeInfo {
	seen := map[string]bool{}
	out := make([]ChangeInfo, 0, len(in))
	for _, c := range in {
		if seen[c.Name] {
			continue
		}
		seen[c.Name] = true
		out = append(out, c)
	}
	return out
}

// GetChange returns the change.md + tasks.md text for a sandboxed name.
func GetChange(ctx context.Context, cwd, name string) (ChangeDetail, error) {
	_, dir, err := resolveRoot(cwd, name)
	if err != nil {
		return ChangeDetail{}, err
	}
	detail := ChangeDetail{Name: name, Status: "unknown"}
	data, err := os.ReadFile(filepath.Join(dir, "change.md"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ChangeDetail{}, &wrapErr{err: ErrNotFound, msg: "change " + name + " has no change.md"}
		}
		return ChangeDetail{}, err
	}
	detail.ChangeMD = string(data)
	detail.Status, detail.Ticket = parseMeta(detail.ChangeMD)
	if !isMeaningfulTicket(detail.Ticket) {
		detail.Ticket = ""
	}
	if tasks, err := os.ReadFile(filepath.Join(dir, "tasks.md")); err == nil {
		detail.TasksMD = string(tasks)
	}
	return detail, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
