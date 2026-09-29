package wiki

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ParseRaw walks the user-dropped raw/ tree (recursively, any depth) and emits
// one concept per .md file. The walk is read-only: ParseRaw never creates,
// modifies, or deletes any file under raw/ (R6.3).
//
// Concept type depends on the file's frontmatter:
//
//   - a .md file with a `---` frontmatter block that carries a `type:` key is
//     indexed with that type (R6.1).
//   - a .md file with a frontmatter block but no `type:` defaults to Finding.
//   - a .md file WITHOUT a frontmatter block becomes a bare Source (filename →
//     title, verified absent) (R6.2).
//
// SourcePath is the bundle-relative path "raw/<path>" (the raw/ root is the
// bundle-relative base, so the returned path is safe to emit verbatim and can
// never escape the bundle — no absolute or `..` segments are produced).
// A missing or empty raw/ yields no concepts and no error (R6.4).
func ParseRaw(root string) ([]Concept, error) {
	var concepts []Concept
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path == root {
				return nil // R6.4: no raw/ dir → no concepts, no error
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			return nil // R6: non-markdown files are skipped silently
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil // tolerant: unreadable file → no concept
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = filepath.ToSlash(d.Name())
		}
		relPath := "raw/" + filepath.ToSlash(rel)
		concepts = append(concepts, rawFileToConcept(d.Name(), string(data), relPath))
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Deterministic ordering (R8.3): sort by bundle-relative path.
	sort.Slice(concepts, func(i, j int) bool {
		return concepts[i].SourcePath < concepts[j].SourcePath
	})
	return concepts, nil
}

// rawFileToConcept builds a Concept from one raw markdown file's content.
func rawFileToConcept(name, content, relPath string) Concept {
	fm, sources, body, hasFM := splitRawFrontmatter(content)
	base := strings.TrimSuffix(name, filepath.Ext(name))

	if !hasFM {
		// R6.2: bare Source — filename → title (unconditionally; the heading is
		// not consulted for a frontmatter-less file), verified absent.
		return Concept{
			ID:         Slugify(base),
			Type:       "Source",
			Title:      base,
			Status:     "draft",
			SourcePath: relPath,
			Body:       strings.TrimSpace(content),
		}
	}

	// R6.1: frontmatter present → index with its type (default Finding).
	c := Concept{
		ID:         Slugify(fm["title"]),
		Type:       fm["type"],
		Title:      fm["title"],
		Status:     "draft",
		SourcePath: relPath,
		Body:       strings.TrimSpace(body),
		Sources:    sources,
	}
	if c.Type == "" {
		c.Type = "Finding"
	}
	if c.Title == "" {
		c.Title = firstHeading(body)
		if c.Title == "" {
			c.Title = base
		}
	}
	if c.ID == "" {
		c.ID = Slugify(c.Title)
	}
	return c
}

// splitRawFrontmatter splits a markdown document into its `---` frontmatter
// (parsed to a flat map of scalar keys plus a decoded sources list), the
// remaining body, and whether a frontmatter block was present.
//
// Only the minimal YAML surface needed by R6 is parsed: top-level
// `key: value` scalars and a `sources:` list of `- resource: …` entries
// (optionally followed by `author:` / `title:` / `last_modified:` lines).
// Unknown keys are ignored. This is NOT a full YAML parser.
func splitRawFrontmatter(content string) (map[string]string, []SourceRef, string, bool) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, nil, content, false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, nil, content, false // opening --- without a close
	}

	fm := map[string]string{}
	var sources []SourceRef
	inSources := false
	var cur *SourceRef

	for i := 1; i < end; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if inSources {
			if strings.HasPrefix(trimmed, "- ") {
				if cur != nil && cur.Resource != "" {
					sources = append(sources, *cur)
				}
				cur = &SourceRef{}
				if k, v, ok := splitKeyValue(strings.TrimPrefix(trimmed, "- ")); ok {
					setSourceField(cur, k, v)
				}
				continue
			}
			if isIndented(line) {
				if cur != nil {
					if k, v, ok := splitKeyValue(trimmed); ok {
						setSourceField(cur, k, v)
					}
				}
				continue
			}
			// A new top-level key ends the sources list.
			if cur != nil && cur.Resource != "" {
				sources = append(sources, *cur)
			}
			cur = nil
			inSources = false
			// fall through to handle this top-level key below
		}

		if isIndented(line) {
			continue // stray indented line outside a known block
		}
		if k, v, ok := splitKeyValue(line); ok {
			if k == "sources" && v == "" {
				inSources = true
				continue
			}
			fm[k] = v
		}
	}
	if cur != nil && cur.Resource != "" {
		sources = append(sources, *cur)
	}

	body := strings.Join(lines[end+1:], "\n")
	return fm, sources, body, true
}

// setSourceField maps a frontmatter source key onto a SourceRef.
func setSourceField(s *SourceRef, key, value string) {
	switch key {
	case "resource":
		s.Resource = value
	case "title":
		s.Title = value
	case "author":
		s.Author = value
	case "last_modified":
		if t, ok := parseTime(value); ok {
			s.LastModified = t
		}
	}
}
