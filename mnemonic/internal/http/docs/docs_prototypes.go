package docs

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// prototypesPath is the root directory of feasibility prototypes. Relative to docsCwd.
const prototypesPath = ".skillgrid/prototypes"

// Prototype is one throwaway feasibility experiment. A prototype is a directory under
// .skillgrid/prototypes/ containing a prototype.md with a verdict.
type Prototype struct {
	Name       string   `json:"name"`
	Date       string   `json:"date,omitempty"`
	Venue      string   `json:"venue,omitempty"`
	Hypothesis string   `json:"hypothesis,omitempty"`
	Change     string   `json:"change,omitempty"`
	Files      []string `json:"files,omitempty"`
}

// NewPrototypes returns the GET /prototypes handler bound to cwd. It lists the
// directories under .skillgrid/prototypes/ and extracts a light summary (date,
// hypothesis, verdict, file list) from each prototype.md so the Prototypes panel can
// surface the repo's real experiments instead of an empty "no prototypes" state.
func NewPrototypes(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prototypes, err := ListPrototypes(r.Context(), cwd)
		if err != nil {
			code := http.StatusInternalServerError
			if os.IsNotExist(err) {
				code = http.StatusNotFound
			}
			writeJSON(w, code, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"source":     filepath.ToSlash(prototypesPath),
			"count":      len(prototypes),
			"prototypes": prototypes,
		})
	})
}

// NewPrototypeFile serves GET /prototypes/{name}/{file...}: one file under
// .skillgrid/prototypes/{name}/, sandboxed to that tree. The Prototypes panel
// links index.html here so the preview (and any relative asset it loads)
// opens in a new tab. Traversal → 400; missing or a directory → 404.
func NewPrototypeFile(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		full, ok := prototypeFilePath(cwd, r.PathValue("name"), r.PathValue("file"))
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid prototype path"})
			return
		}
		// ServeContent, not ServeFile: ServeFile 301s any URL ending in
		// /index.html to the directory, which would hide the preview.
		f, err := os.Open(full)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "prototype file not found"})
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil || st.IsDir() {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "prototype file not found"})
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, st.Name(), st.ModTime(), f)
	})
}

// prototypeFilePath resolves name/file under .skillgrid/prototypes and rejects
// anything that escapes that root (absolute ids, "..", a name that is itself
// a path).
func prototypeFilePath(cwd, name, file string) (string, bool) {
	if name == "" || file == "" || name == "." || name == ".." {
		return "", false
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return "", false
	}
	if strings.Contains(file, "..") || strings.HasPrefix(file, "/") || filepath.IsAbs(file) {
		return "", false
	}
	root := filepath.Clean(filepath.Join(cwd, prototypesPath))
	full := filepath.Join(root, name, filepath.FromSlash(file))
	rel, err := filepath.Rel(root, filepath.Clean(full))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	if rel != name && !strings.HasPrefix(rel, name+string(filepath.Separator)) {
		return "", false
	}
	return filepath.Clean(full), true
}

// prototype.md header fields. The Date may be `**Date:**` (bold) or a bare
// `Date:` line; Hypothesis is single-paragraph, captured to the next blank
// line or section heading.
var (
	prototypeDateRE  = regexp.MustCompile(`(?m)^\*{0,2}Date:\*{0,2}\s*(.+)$`)
	prototypeHypRE   = regexp.MustCompile(`(?m)^\*{0,2}Hypothesis:\*{0,2}\s*(.+)$`)
	prototypeVenueRE = regexp.MustCompile(`(?m)^\*{0,2}Type:\*{0,2}\s*(.+)$`)
)

// ListPrototypes scans .skillgrid/prototypes/ for sub-directories and parses a summary
// from each prototype.md. Missing root returns os.ErrNotExist (→ 404). Directories
// without a prototype.md are still listed (Files still populated) so a WIP prototype is
// visible rather than silently dropped.
func ListPrototypes(ctx context.Context, cwd string) ([]Prototype, error) {
	root := filepath.Join(cwd, prototypesPath)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []Prototype
	for _, e := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		prototype := Prototype{Name: e.Name()}
		// File list (for the "verdicts in prototype.md" hint + future links).
		if files, err := os.ReadDir(dir); err == nil {
			for _, f := range files {
				if f.IsDir() {
					continue
				}
				prototype.Files = append(prototype.Files, f.Name())
			}
			sort.Strings(prototype.Files)
		}
		// prototype.md is the canonical summary doc.
		if data, err := os.ReadFile(filepath.Join(dir, "prototype.md")); err == nil {
			content := string(data)
			if m := prototypeDateRE.FindStringSubmatch(content); m != nil {
				prototype.Date = cleanMetaValue(m[1])
			}
			if m := prototypeVenueRE.FindStringSubmatch(content); m != nil {
				prototype.Venue = cleanMetaValue(m[1])
			}
			if m := prototypeHypRE.FindStringSubmatch(content); m != nil {
				prototype.Hypothesis = cleanMetaValue(m[1])
			}
			prototype.Change = prototypeChange(cwd, e.Name(), content)
		}
		if prototype.Change == "" {
			prototype.Change = findChangeCiting(cwd, e.Name())
		}
		out = append(out, prototype)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// cleanMetaValue strips trailing backticks/whitespace from a parsed header
// value and trims it.
func cleanMetaValue(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, "`")
	return strings.TrimSpace(v)
}

// A prototype.md may name its change with **Topic:** / **Change:**, a `topic:`
// frontmatter line, or a `.skillgrid/specs|archive/<id>/` path.
var (
	prototypeTopicRE = regexp.MustCompile(`(?m)^\*{0,2}(?:Topic|Change):\*{0,2}\s*(.+)$`)
	prototypeYAMLRE  = regexp.MustCompile(`(?m)^topic:\s*(\S+)`)
	changePathRE     = regexp.MustCompile(`\.skillgrid/(?:specs|archive)/(\d{4}-\d{2}-\d{2}-[^/\s` + "`" + `]+)`)
)

// prototypeChange reads the change id recorded in prototype.md, when that
// directory exists under .skillgrid/specs or .skillgrid/archive.
func prototypeChange(cwd, _, content string) string {
	for _, id := range []string{
		changeID(prototypeTopicRE, content),
		changeID(prototypeYAMLRE, content),
		changeID(changePathRE, content),
	} {
		if id != "" && changeExists(cwd, id) {
			return id
		}
	}
	return ""
}

func changeID(re *regexp.Regexp, content string) string {
	m := re.FindStringSubmatch(content)
	if m == nil {
		return ""
	}
	id := cleanMetaValue(m[1])
	if i := strings.IndexAny(id, " \t"); i > 0 {
		id = id[:i]
	}
	if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return ""
	}
	return id
}

func changeExists(cwd, id string) bool {
	for _, root := range []string{"specs", "archive"} {
		fi, err := os.Stat(filepath.Join(cwd, ".skillgrid", root, id))
		if err == nil && fi.IsDir() {
			return true
		}
	}
	return false
}

// findChangeCiting returns the newest change (specs before archive) whose
// markdown mentions this prototype directory.
func findChangeCiting(cwd, name string) string {
	needles := []string{"prototypes/" + name, "spikes/" + name}
	for _, root := range []string{"specs", "archive"} {
		entries, err := os.ReadDir(filepath.Join(cwd, ".skillgrid", root))
		if err != nil {
			continue
		}
		var hits []string
		for _, e := range entries {
			if e.IsDir() && dirCites(filepath.Join(cwd, ".skillgrid", root, e.Name()), needles) {
				hits = append(hits, e.Name())
			}
		}
		if len(hits) > 0 {
			sort.Strings(hits)
			return hits[len(hits)-1]
		}
	}
	return ""
}

func dirCites(dir string, needles []string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) > 1<<20 {
			return nil
		}
		text := string(data)
		for _, n := range needles {
			if strings.Contains(text, n) {
				found = true
				return fs.SkipAll
			}
		}
		return nil
	})
	return found
}
