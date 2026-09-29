package wiki

import (
	"strings"
	"time"
)

// staleHorizon is the fixed staleness window applied to every web cache
// Finding: stale_after = fetched_at + 90d (R5).
const staleHorizon = 90 * 24 * time.Hour

// webCacheSources are the web cache source tools the indexer reads. Rows with
// any other source (e.g. "manual") are not indexed (R5).
var webCacheSources = map[string]bool{
	"context7": true,
	"exa":      true,
	"deepwiki": true,
	"fetch":    true,
}

// WebRow is one fresh web_cache row handed to ParseWebCache by the CLI. The
// wiki package stays pure (no database import); the CLI reads the SQLite rows
// via the webcache service and maps them to this struct.
//
// FetchedAt and ExpiresAt are RFC3339 strings as stored in the database
// (fetched_at / expires_at columns).
type WebRow struct {
	Source     string
	URL        string
	Title      string
	Query      string
	LibraryID  string
	VersionTag string
	FetchedAt  string
	ExpiresAt  string
	ContentHash string
}

// ParseWebCache converts fresh web_cache rows into Finding concepts.
//
// The caller (CLI) is responsible for fetching only fresh rows
// (expires_at IS NULL OR expires_at > now) and only rows whose source is one
// of context7/exa/deepwiki/fetch. ParseWebCache re-applies the source filter
// defensively and skips rows it cannot date.
//
// Every row becomes a Finding with:
//
//   - ID = Slugify(title) (title fallback: url host, then "web-research")
//   - sources[0].resource = the row's url, or a stable internal descriptor
//     when the url is empty (R5.4)
//   - sources[0].author = "process:<source>"
//   - verified = [{by: "process:<source>", at: fetched_at}]
//   - stale_after = fetched_at + 90d
//   - status = "stable" (a fresh, verified research finding; the compiler
//     demotes uncited findings to "draft" under wiki/research/)
//
// ParseWebCache is pure and deterministic: it never writes to raw/ and never
// touches the database.
func ParseWebCache(rows []WebRow) []Concept {
	var concepts []Concept
	for _, r := range rows {
		if !webCacheSources[r.Source] {
			continue
		}
		fetched, ok := parseTime(r.FetchedAt)
		if !ok {
			continue
		}
		concepts = append(concepts, webRowToConcept(r, fetched))
	}
	return concepts
}

// webRowToConcept builds a Finding from one fresh web_cache row.
func webRowToConcept(r WebRow, fetched time.Time) Concept {
	title := strings.TrimSpace(r.Title)
	if title == "" {
		title = fallbackWebTitle(r)
	}
	id := Slugify(title)
	if id == "" {
		id = "web-research"
	}

	// sources[].resource: the url, or a stable internal descriptor when the
	// url is empty (R5.4 — no panic, always a non-empty resource).
	resource := strings.TrimSpace(r.URL)
	if resource == "" {
		resource = "web:" + strings.ToLower(r.Source)
		if r.ContentHash != "" {
			resource += ":" + r.ContentHash
		}
		resource += ":" + id
	}

	actor := "process:" + r.Source
	return Concept{
		ID:    id,
		Type:  "Finding",
		Title: title,
		Sources: []SourceRef{{
			Resource:     resource,
			Author:       actor,
			LastModified: fetched,
		}},
		Verified:   []Verifier{{By: actor, At: fetched}},
		Status:     "stable",
		StaleAfter: fetched.Add(staleHorizon),
		Body:       title,
	}
}

// fallbackWebTitle derives a page title for a row that has none: the url host
// when available, otherwise a generic descriptor.
func fallbackWebTitle(r WebRow) string {
	if u := strings.TrimSpace(r.URL); u != "" {
		if host := hostOf(u); host != "" {
			return host
		}
		return u
	}
	return "web-research"
}

// hostOf extracts the host from a url string without importing net/url
// (keeping the function table-driven and dependency-free). It returns "" when
// no host can be identified.
func hostOf(u string) string {
	rest := u
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		rest = rest[:i]
	}
	if i := strings.IndexByte(rest, ':'); i >= 0 {
		rest = rest[:i]
	}
	if i := strings.IndexByte(rest, '@'); i >= 0 {
		rest = rest[i+1:]
	}
	return rest
}

// parseTime parses an RFC3339 (or SQLite "YYYY-MM-DD HH:MM:SS") timestamp.
func parseTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}
