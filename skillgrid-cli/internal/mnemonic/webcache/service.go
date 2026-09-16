// Package webcache provides save, lookup, and search over cached web research snapshots.
package webcache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

const defaultSearchLimit = 20

// ErrContentTooLarge is returned when content exceeds MaxEntryBytes.
var ErrContentTooLarge = errors.New("content exceeds max entry size")

// Service provides save, lookup, and search over cached web research snapshots.
type Service struct {
	store     *store.Store
	projectID string
	cfg       config.WebCache
}

// SaveWebInput holds fields for persisting a cached snapshot.
type SaveWebInput struct {
	Source     string         `json:"source"`
	Content    string         `json:"content"`
	URL        string         `json:"url,omitempty"`
	Title      string         `json:"title,omitempty"`
	Query      string         `json:"query,omitempty"`
	LibraryID  string         `json:"library_id,omitempty"`
	VersionTag string         `json:"version_tag,omitempty"`
	RepoName   string         `json:"repo_name,omitempty"`
	Question   string         `json:"question,omitempty"`
	SortParams string         `json:"sort_params,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
}

// LookupInput identifies a cache entry for lookup.
type LookupInput struct {
	Source      string
	URL         string
	Query       string
	LibraryID   string
	VersionTag  string
	RepoName    string
	Question    string
	SortParams  string
	Title       string
	ContentHash string
}

// LookupResult is the outcome of a cache lookup.
type LookupResult struct {
	Status    string `json:"status"`
	Fresh     bool   `json:"fresh"`
	ID        int64  `json:"id,omitempty"`
	FetchedAt string `json:"fetched_at,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// WebHit is a search result over cached snapshots.
type WebHit struct {
	ID        int64  `json:"id"`
	Source    string `json:"source"`
	Title     string `json:"title,omitempty"`
	Query     string `json:"query,omitempty"`
	URL       string `json:"url,omitempty"`
	LibraryID string `json:"library_id,omitempty"`
	FetchedAt string `json:"fetched_at,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// WebEntry is a full cached snapshot.
type WebEntry struct {
	ID           int64  `json:"id"`
	Project      string `json:"project"`
	Source       string `json:"source"`
	CacheKey     string `json:"cache_key,omitempty"`
	URL          string `json:"url,omitempty"`
	Title        string `json:"title,omitempty"`
	Query        string `json:"query,omitempty"`
	LibraryID    string `json:"library_id,omitempty"`
	VersionTag   string `json:"version_tag,omitempty"`
	Content      string `json:"content,omitempty"`
	MetadataJSON string `json:"metadata_json,omitempty"`
	ContentHash  string `json:"content_hash,omitempty"`
	FetchedAt    string `json:"fetched_at,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// Status holds aggregate web cache statistics.
type Status struct {
	TotalEntries   int            `json:"total_entries"`
	ExpiredEntries int            `json:"expired_entries"`
	BySource       map[string]int `json:"by_source"`
	OldestFetch    string         `json:"oldest_fetch,omitempty"`
	NewestFetch    string         `json:"newest_fetch,omitempty"`
}

// New creates a web cache service for the given store and project ID.
func New(st *store.Store, projectID string, cfg config.WebCache) *Service {
	return &Service{store: st, projectID: projectID, cfg: cfg}
}

// Save stores or upserts a cached snapshot keyed by (project, source, cache_key).
func (s *Service) Save(ctx context.Context, in SaveWebInput) (int64, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, errors.New("web cache service not initialized")
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		return 0, errors.New("source is required")
	}
	if strings.TrimSpace(in.Content) == "" {
		return 0, errors.New("content is required")
	}
	if s.cfg.MaxEntryBytes > 0 && len(in.Content) > s.cfg.MaxEntryBytes {
		return 0, fmt.Errorf("%w (%d > %d)", ErrContentTooLarge, len(in.Content), s.cfg.MaxEntryBytes)
	}
	hash := contentHash(in.Content)
	cacheKey, err := CacheKey(source, keyInputFromSave(in, hash))
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	fetchedAt := now.Format(time.RFC3339)
	expiresAt := s.expiresAt(source, now)
	var metadataJSON sql.NullString
	if len(in.Metadata) > 0 {
		b, err := json.Marshal(in.Metadata)
		if err != nil {
			return 0, fmt.Errorf("marshal metadata: %w", err)
		}
		metadataJSON = sql.NullString{String: string(b), Valid: true}
	}
	res, err := s.store.DB.ExecContext(ctx, `
		INSERT INTO web_cache (
			project, source, cache_key, url, title, query, library_id, version_tag,
			content, metadata_json, content_hash, fetched_at, expires_at, session_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project, source, cache_key) DO UPDATE SET
			url = excluded.url,
			title = excluded.title,
			query = excluded.query,
			library_id = excluded.library_id,
			version_tag = excluded.version_tag,
			content = excluded.content,
			metadata_json = excluded.metadata_json,
			content_hash = excluded.content_hash,
			fetched_at = excluded.fetched_at,
			expires_at = excluded.expires_at,
			session_id = excluded.session_id`,
		s.projectID, source, cacheKey,
		nullString(in.URL), nullString(in.Title), nullString(in.Query),
		nullString(in.LibraryID), nullString(in.VersionTag),
		in.Content, metadataJSON, hash, fetchedAt, nullString(expiresAt),
		nullString(in.SessionID), fetchedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("save web cache: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	if id == 0 {
		err = s.store.DB.QueryRowContext(ctx, `
			SELECT id FROM web_cache
			WHERE project = ? AND source = ? AND cache_key = ?`,
			s.projectID, source, cacheKey,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("lookup upsert id: %w", err)
		}
	}
	return id, nil
}

// Lookup checks cache by source-specific cache_key derivation.
func (s *Service) Lookup(ctx context.Context, in LookupInput) (LookupResult, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return LookupResult{}, errors.New("web cache service not initialized")
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		return LookupResult{}, errors.New("source is required")
	}
	cacheKey, err := CacheKey(source, keyInputFromLookup(in))
	if err != nil {
		return LookupResult{}, err
	}
	var id int64
	var fetchedAt string
	var expiresAt sql.NullString
	err = s.store.DB.QueryRowContext(ctx, `
		SELECT id, fetched_at, expires_at
		FROM web_cache
		WHERE project = ? AND source = ? AND cache_key = ?`,
		s.projectID, source, cacheKey,
	).Scan(&id, &fetchedAt, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return LookupResult{Status: "miss"}, nil
	}
	if err != nil {
		return LookupResult{}, fmt.Errorf("lookup web cache: %w", err)
	}
	result := LookupResult{
		ID:        id,
		FetchedAt: fetchedAt,
		Status:    "hit",
		Fresh:     true,
	}
	if expiresAt.Valid {
		result.ExpiresAt = expiresAt.String
		if isExpired(expiresAt.String) {
			result.Status = "stale"
			result.Fresh = false
		}
	}
	return result, nil
}

// Search runs FTS5 over cached snapshots with optional source and freshness filters.
func (s *Service) Search(ctx context.Context, query string, source string, freshOnly bool, limit int) ([]WebHit, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("web cache service not initialized")
	}
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	source = strings.TrimSpace(source)
	freshClause := ""
	if freshOnly {
		freshClause = `AND (w.expires_at IS NULL OR datetime(w.expires_at) > datetime('now'))`
	}
	sourceClause := ""
	args := []any{ftsQuery, s.projectID}
	if source != "" {
		sourceClause = `AND w.source = ?`
		args = append(args, source)
	}
	args = append(args, limit)
	rows, err := s.store.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT w.id, w.source, w.title, w.query, w.url, w.library_id, w.fetched_at, w.expires_at
		FROM web_cache w
		INNER JOIN web_cache_fts ON web_cache_fts.rowid = w.id
		WHERE web_cache_fts MATCH ? AND w.project = ?
		%s %s
		ORDER BY bm25(web_cache_fts)
		LIMIT ?`, sourceClause, freshClause), args...)
	if err != nil {
		return nil, fmt.Errorf("search web cache: %w", err)
	}
	defer rows.Close()
	return scanWebHits(rows)
}

// Get returns a full snapshot by ID scoped to the project.
func (s *Service) Get(ctx context.Context, id int64) (WebEntry, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return WebEntry{}, errors.New("web cache service not initialized")
	}
	row := s.store.DB.QueryRowContext(ctx, `
		SELECT id, project, source, cache_key, url, title, query, library_id, version_tag,
		       content, metadata_json, content_hash, fetched_at, expires_at, session_id, created_at
		FROM web_cache
		WHERE id = ? AND project = ?`,
		id, s.projectID,
	)
	entry, err := scanWebEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WebEntry{}, fmt.Errorf("web cache entry %d not found", id)
	}
	if err != nil {
		return WebEntry{}, fmt.Errorf("get web cache: %w", err)
	}
	return entry, nil
}

// CacheStatus returns aggregate statistics for the project web cache.
func (s *Service) CacheStatus(ctx context.Context) (Status, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return Status{}, errors.New("web cache service not initialized")
	}
	var st Status
	st.BySource = make(map[string]int)
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM web_cache WHERE project = ?`,
		s.projectID,
	).Scan(&st.TotalEntries)
	if err != nil {
		return Status{}, fmt.Errorf("count web cache: %w", err)
	}
	err = s.store.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM web_cache
		WHERE project = ?
		  AND expires_at IS NOT NULL
		  AND datetime(expires_at) <= datetime('now')`,
		s.projectID,
	).Scan(&st.ExpiredEntries)
	if err != nil {
		return Status{}, fmt.Errorf("count expired web cache: %w", err)
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT source, COUNT(*) FROM web_cache WHERE project = ? GROUP BY source`,
		s.projectID,
	)
	if err != nil {
		return Status{}, fmt.Errorf("count by source: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var src string
		var count int
		if err := rows.Scan(&src, &count); err != nil {
			return Status{}, fmt.Errorf("scan source count: %w", err)
		}
		st.BySource[src] = count
	}
	if err := rows.Err(); err != nil {
		return Status{}, fmt.Errorf("iterate source counts: %w", err)
	}
	var oldest, newest sql.NullString
	err = s.store.DB.QueryRowContext(ctx, `
		SELECT MIN(fetched_at), MAX(fetched_at) FROM web_cache WHERE project = ?`,
		s.projectID,
	).Scan(&oldest, &newest)
	if err != nil {
		return Status{}, fmt.Errorf("fetch range: %w", err)
	}
	if oldest.Valid {
		st.OldestFetch = oldest.String
	}
	if newest.Valid {
		st.NewestFetch = newest.String
	}
	return st, nil
}

func (s *Service) expiresAt(source string, fetched time.Time) string {
	ttl := s.cfg.TTL[source]
	if ttl <= 0 {
		return ""
	}
	return fetched.Add(ttl).Format(time.RFC3339)
}

func isExpired(expiresAt string) bool {
	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return false
	}
	return !t.After(time.Now().UTC())
}

func buildFTSQuery(query string) string {
	terms := strings.Fields(strings.TrimSpace(query))
	if len(terms) == 0 {
		return ""
	}
	escaped := make([]string, len(terms))
	for i, term := range terms {
		term = strings.ReplaceAll(term, `"`, `""`)
		escaped[i] = `"` + term + `"`
	}
	return strings.Join(escaped, " OR ")
}

func nullString(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func scanWebHits(rows *sql.Rows) ([]WebHit, error) {
	var hits []WebHit
	for rows.Next() {
		var hit WebHit
		var title, query, url, libraryID, expiresAt sql.NullString
		if err := rows.Scan(
			&hit.ID, &hit.Source, &title, &query, &url, &libraryID,
			&hit.FetchedAt, &expiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan web hit: %w", err)
		}
		if title.Valid {
			hit.Title = title.String
		}
		if query.Valid {
			hit.Query = query.String
		}
		if url.Valid {
			hit.URL = url.String
		}
		if libraryID.Valid {
			hit.LibraryID = libraryID.String
		}
		if expiresAt.Valid {
			hit.ExpiresAt = expiresAt.String
		}
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate web hits: %w", err)
	}
	return hits, nil
}

func scanWebEntry(row *sql.Row) (WebEntry, error) {
	var entry WebEntry
	var cacheKey, url, title, query, libraryID, versionTag, metadataJSON, expiresAt, sessionID, createdAt sql.NullString
	err := row.Scan(
		&entry.ID, &entry.Project, &entry.Source, &cacheKey,
		&url, &title, &query, &libraryID, &versionTag,
		&entry.Content, &metadataJSON, &entry.ContentHash,
		&entry.FetchedAt, &expiresAt, &sessionID, &createdAt,
	)
	if err != nil {
		return WebEntry{}, err
	}
	if cacheKey.Valid {
		entry.CacheKey = cacheKey.String
	}
	if url.Valid {
		entry.URL = url.String
	}
	if title.Valid {
		entry.Title = title.String
	}
	if query.Valid {
		entry.Query = query.String
	}
	if libraryID.Valid {
		entry.LibraryID = libraryID.String
	}
	if versionTag.Valid {
		entry.VersionTag = versionTag.String
	}
	if metadataJSON.Valid {
		entry.MetadataJSON = metadataJSON.String
	}
	if expiresAt.Valid {
		entry.ExpiresAt = expiresAt.String
	}
	if sessionID.Valid {
		entry.SessionID = sessionID.String
	}
	if createdAt.Valid {
		entry.CreatedAt = createdAt.String
	}
	return entry, nil
}

var validSources = map[string]struct{}{
	"context7": {},
	"exa":      {},
	"deepwiki": {},
	"fetch":    {},
	"manual":   {},
}

// KeyInput holds fields used to derive a normalized cache_key per source.
type KeyInput struct {
	URL         string
	Query       string
	SortParams  string
	LibraryID   string
	VersionTag  string
	RepoName    string
	Question    string
	Title       string
	ContentHash string
}

// CacheKey returns a sha256 hex digest for dedup keyed by source-specific rules.
func CacheKey(source string, in KeyInput) (string, error) {
	if _, ok := validSources[source]; !ok {
		return "", fmt.Errorf("unsupported web cache source %q", source)
	}
	var material string
	switch source {
	case "fetch":
		if strings.TrimSpace(in.URL) == "" {
			return "", fmt.Errorf("url is required for fetch cache_key")
		}
		material = normalizeURL(in.URL)
	case "exa":
		if strings.TrimSpace(in.Query) == "" {
			return "", fmt.Errorf("query is required for exa cache_key")
		}
		material = in.Query + "|" + in.SortParams
	case "context7":
		if strings.TrimSpace(in.LibraryID) == "" {
			return "", fmt.Errorf("library_id is required for context7 cache_key")
		}
		if strings.TrimSpace(in.Query) == "" {
			return "", fmt.Errorf("query is required for context7 cache_key")
		}
		material = in.LibraryID + "|" + in.VersionTag + "|" + in.Query
	case "deepwiki":
		if strings.TrimSpace(in.RepoName) == "" {
			return "", fmt.Errorf("repo_name is required for deepwiki cache_key")
		}
		question := in.Question
		if question == "" {
			question = in.Query
		}
		if strings.TrimSpace(question) == "" {
			return "", fmt.Errorf("question is required for deepwiki cache_key")
		}
		material = in.RepoName + "|" + question
	case "manual":
		if strings.TrimSpace(in.Title) == "" {
			return "", fmt.Errorf("title is required for manual cache_key")
		}
		if strings.TrimSpace(in.ContentHash) == "" {
			return "", fmt.Errorf("content_hash is required for manual cache_key")
		}
		material = in.Title + "|" + in.ContentHash
	}
	return sha256Hex(material), nil
}

func keyInputFromSave(in SaveWebInput, contentHash string) KeyInput {
	return KeyInput{
		URL:         in.URL,
		Query:       in.Query,
		SortParams:  in.SortParams,
		LibraryID:   in.LibraryID,
		VersionTag:  in.VersionTag,
		RepoName:    in.RepoName,
		Question:    in.Question,
		Title:       in.Title,
		ContentHash: contentHash,
	}
}

func keyInputFromLookup(in LookupInput) KeyInput {
	return KeyInput{
		URL:         in.URL,
		Query:       in.Query,
		SortParams:  in.SortParams,
		LibraryID:   in.LibraryID,
		VersionTag:  in.VersionTag,
		RepoName:    in.RepoName,
		Question:    in.Question,
		Title:       in.Title,
		ContentHash: in.ContentHash,
	}
}

func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return strings.ToLower(raw)
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	if len(u.Path) > 1 {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}
	return u.String()
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func contentHash(content string) string {
	return sha256Hex(content)
}
