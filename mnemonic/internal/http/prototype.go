package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// prototypeRoot returns the .skillgrid/prototype/ directory (under sddRoot,
// the repo root). This is the on-disk store for the companion's throwaway
// interactive prototypes (agent-authored HTML), separate from the feasibility
// prototypes listed by GET /prototypes (.skillgrid/prototypes/).
func prototypeRoot() string { return filepath.Join(sddRoot(), ".skillgrid", "prototype") }

// prototypeFile resolves a prototype id to an absolute path sandboxed to
// .skillgrid/prototype/ — the same path-traversal guard shape as the specs
// reader (plans.go). ok=false for empty/absolute ids, '..' segments, or any
// path that escapes the root after filepath.Clean.
func prototypeFile(id string) (string, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", false
	}
	// Reject absolute ids (leading / or a Windows drive) up front.
	if filepath.IsAbs(id) || id[0] == '/' || (len(id) >= 2 && id[1] == ':') {
		return "", false
	}
	root := prototypeRoot()
	full := filepath.Join(root, filepath.FromSlash(id))
	// Clean both and ensure the resolved path stays under root.
	cleanFull := filepath.Clean(full)
	cleanRoot := filepath.Clean(root)
	rel, err := filepath.Rel(cleanRoot, cleanFull)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return cleanFull, true
}

// handlePrototypeDecision serves GET /prototype/{id...} — the HTML of one
// companion prototype, sandboxed to .skillgrid/prototype/. Traversal/absolute
// ids → 400; unknown → 404. Rendered client-side in a sandboxed iframe
// (sandbox="allow-scripts", no allow-same-origin).
func (s *Server) handlePrototypeDecision(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	full, ok := prototypeFile(id)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid prototype id: "+id)
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "unknown prototype: "+id)
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
