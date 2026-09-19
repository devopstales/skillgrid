package http

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// stitchRoot returns the .stitch/ directory (under sddRoot, the repo root).
func stitchRoot() string { return filepath.Join(sddRoot(), ".stitch") }

// stitchFile resolves a prototype id to an absolute path sandboxed to
// .stitch/. It returns (path, ok): ok=false when the id is empty, absolute,
// contains `..` segments, or escapes .stitch/ after cleaning. This is the
// path-traversal guard (Phase 7 threat model).
func stitchFile(id string) (string, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", false
	}
	// Reject absolute ids (leading / or a Windows drive) up front.
	if filepath.IsAbs(id) || id[0] == '/' || (len(id) >= 2 && id[1] == ':') {
		return "", false
	}
	root := stitchRoot()
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

// handlePrototypes serves GET /prototypes — a list of the prototype files
// under .stitch/ (relative paths), so the UI can build a gallery.
func (s *Server) handlePrototypes(w http.ResponseWriter, r *http.Request) {
	root := stitchRoot()
	files := []string{}
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		// Only serve HTML prototypes in the list (the gallery is for HTML).
		if !strings.EqualFold(path.Ext(rel), ".html") {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	writeJSON(w, http.StatusOK, map[string]any{"prototypes": files})
}

// handlePrototype serves GET /prototypes/{id} — the HTML of one prototype
// (sandboxed to .stitch/). Traversal/absolute ids → 400; unknown → 404.
func (s *Server) handlePrototype(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	full, ok := stitchFile(id)
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
