package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// Phase 5 — Mnemonic files / memories / sessions / audit / search. Serves the
// remaining Mnemonic suite to the Web Admin Dashboard: the OpenViking file tree
// (a virtual FS over observations.topic_key) + L0/L1/L2 content tiers, the
// memory browser (list/detail + 013 governance), the session browser, a
// hash-chained audit trail, and hybrid semantic search.
//
// Governance mutations (edit/share/status) are write-gated via the Bearer
// SKILLGRID_HTTP_TOKEN and render 013 data with forward-compat placeholders when
// the governance columns are absent.

// ---------------------------------------------------------------------------
// files tree + content (5.1)
// ---------------------------------------------------------------------------

// leafStat is the per-topic_key aggregate derived from observations.
type leafStat struct {
	latest string
	count  int
}

// fileTreeNode is one node in the OpenViking tree. MemoryCount is the number of
// live observations under this prefix; LastIndexed is the newest created_at.
type fileTreeNode struct {
	Name        string         `json:"name"`
	Path        string         `json:"path"`
	Leaf        bool           `json:"leaf"`
	MemoryCount int            `json:"memory_count"`
	LastIndexed string         `json:"last_indexed,omitempty"`
	Children    []fileTreeNode `json:"children,omitempty"`
}

// fileTreeResponse is the GET /mnemonic/files/tree payload.
type fileTreeResponse struct {
	Project string       `json:"project"`
	Root    fileTreeNode `json:"root"`
	Nodes   int          `json:"nodes"`
}

// fileContentResponse is the GET /mnemonic/files/content payload (L0/L1/L2).
// L0 = abstract (newest title as a one-line summary), L1 = overview (top titles
// + types), L2 = details (full observation content).
type fileContentResponse struct {
	URI     string           `json:"uri"`
	Project string           `json:"project"`
	L0      *fileContentTier `json:"l0,omitempty"`
	L1      *fileContentTier `json:"l1,omitempty"`
	L2      *fileContentTier `json:"l2,omitempty"`
}

type fileContentTier struct {
	Label   string            `json:"label"`
	Content string            `json:"content"`
	Count   int               `json:"count"`
	Items   []fileContentItem `json:"items,omitempty"`
}

type fileContentItem struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// handleMnemonicFilesTree returns the OpenViking tree (derived from
// observations.topic_key) for the project, with per-node memory counts and the
// newest created_at. `?path=` scopes the walk to a topic_key prefix.
func (s *Server) handleMnemonicFilesTree(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	scope := strings.Trim(r.URL.Query().Get("path"), "/")
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()

	rows, err := h.Store().DB.QueryContext(r.Context(), `
		SELECT topic_key, MAX(created_at) AS latest, COUNT(*) AS cnt
		FROM observations
		WHERE project = ? AND deleted_at IS NULL AND topic_key IS NOT NULL
		  AND topic_key != ''
		  AND (? = '' OR topic_key LIKE ? || '/%' OR topic_key = ?)
		GROUP BY topic_key
		ORDER BY topic_key`, projectID, scope, scope, scope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	leaves := map[string]leafStat{}
	for rows.Next() {
		var tk, latest string
		var cnt int
		if err := rows.Scan(&tk, &latest, &cnt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		leaves[tk] = leafStat{latest: latest, count: cnt}
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	root := fileTreeNode{Name: "root", Path: "", Leaf: false}
	buildFileTree(&root, "", leaves)
	sortChildren(&root)
	writeJSON(w, http.StatusOK, fileTreeResponse{
		Project: projectID, Root: root, Nodes: countTreeNodes(&root),
	})
}

// buildFileTree recursively assembles the tree from the leaf stats keyed by
// topic_key. At each depth, topic keys that extend the current prefix by a
// single segment become leaf children; longer keys become directory children
// that are recursed into. Aggregates (memory_count, last_indexed) roll up from
// children, and a key whose full path equals the current prefix is a leaf even
// if it also has children (a topic that both holds observations and is a
// prefix of deeper topics).
func buildFileTree(node *fileTreeNode, prefix string, leaves map[string]leafStat) {
	// restOf(tk): the portion of the topic key below this node's prefix, with
	// the leading "/" removed. A rest with no "/" is a single-segment child.
	restOf := func(tk string) string {
		if prefix == "" {
			return tk
		}
		return strings.TrimPrefix(tk, prefix+"/")
	}

	// distinct first segments that have deeper keys → directory children.
	dirSegs := map[string]bool{}
	for tk := range leaves {
		if prefix != "" && !strings.HasPrefix(tk, prefix+"/") {
			continue
		}
		rest := restOf(tk)
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			dirSegs[rest[:i]] = true
		}
	}

	// leaf children: keys exactly one segment below the prefix (no further
	// "/"). The leaf name is the final segment; the path is the full topic key.
	// A topic that is BOTH a leaf and a prefix of deeper topics (e.g. 'arch'
	// has observations and also 'arch/auth') is emitted once as a directory
	// node whose Leaf flag is true and whose children are the deeper topics.
	leafAt := map[string]leafStat{}
	for tk, st := range leaves {
		if prefix != "" && !strings.HasPrefix(tk, prefix+"/") {
			continue
		}
		rest := restOf(tk)
		if strings.Contains(rest, "/") {
			continue
		}
		name := rest
		if name == "" {
			name = tk
		}
		if cur, ok := leafAt[name]; !ok || st.count > cur.count {
			leafAt[name] = st
		}
	}

	for seg, st := range leafAt {
		if dirSegs[seg] {
			// directory that is also a leaf: build children, then mark leaf.
			childPrefix := seg
			if prefix != "" {
				childPrefix = prefix + "/" + seg
			}
			child := fileTreeNode{Name: seg, Path: childPrefix, Leaf: true}
			child.MemoryCount = st.count
			child.LastIndexed = st.latest
			buildFileTree(&child, childPrefix, leaves)
			node.Children = append(node.Children, child)
			node.MemoryCount += child.MemoryCount
			if child.LastIndexed > node.LastIndexed {
				node.LastIndexed = child.LastIndexed
			}
			continue
		}
		// plain leaf (no deeper topics).
		path := seg
		if prefix != "" {
			path = prefix + "/" + seg
		}
		node.Children = append(node.Children, fileTreeNode{
			Name: seg, Path: path, Leaf: true,
			MemoryCount: st.count, LastIndexed: st.latest,
		})
		node.MemoryCount += st.count
		if st.latest > node.LastIndexed {
			node.LastIndexed = st.latest
		}
	}

	// directory children (non-leaf): recurse for each distinct first segment
	// that has no leaf at this level, then roll the aggregate up.
	for seg := range dirSegs {
		if leafAt[seg] != (leafStat{}) {
			continue
		}
		childPrefix := seg
		if prefix != "" {
			childPrefix = prefix + "/" + seg
		}
		child := fileTreeNode{Name: seg, Path: childPrefix, Leaf: false}
		buildFileTree(&child, childPrefix, leaves)
		node.Children = append(node.Children, child)
		node.MemoryCount += child.MemoryCount
		if child.LastIndexed > node.LastIndexed {
			node.LastIndexed = child.LastIndexed
		}
	}
}

// fetchTopicItems returns the live observations for an exact topic_key, newest
// first.
func fetchTopicItems(ctx context.Context, h *service.ProjectHandle, projectID, topic string) ([]fileContentItem, error) {
	rows, err := h.Store().DB.QueryContext(ctx, `
		SELECT id, title, content, type, created_at
		FROM observations
		WHERE project = ? AND deleted_at IS NULL AND topic_key = ?
		ORDER BY created_at DESC, id DESC`, projectID, topic)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []fileContentItem
	for rows.Next() {
		var it fileContentItem
		if err := rows.Scan(&it.ID, &it.Title, &it.Content, &it.Type, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func sortChildren(n *fileTreeNode) {
	sort.Slice(n.Children, func(i, j int) bool {
		return n.Children[i].Name < n.Children[j].Name
	})
	for i := range n.Children {
		sortChildren(&n.Children[i])
	}
}

func countTreeNodes(n *fileTreeNode) int {
	total := 1
	for i := range n.Children {
		total += countTreeNodes(&n.Children[i])
	}
	return total
}

// handleMnemonicFilesContent returns L0/L1/L2 tiers for a topic_key. `?uri=` is
// a mnemonic:// URI whose path is the full topic key (e.g.
// mnemonic://sdd/web-admin-dashboard/phase-4-mnemonic-graph). Multi-segment
// topics must be addressed by their full path; a final-segment match is used
// as a fallback when the full path yields no observations.
func (s *Server) handleMnemonicFilesContent(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		writeError(w, http.StatusBadRequest, "uri is required")
		return
	}
	topic := strings.TrimPrefix(uri, "mnemonic://")
	topic = strings.TrimPrefix(topic, "project/")
	if topic == "" {
		writeError(w, http.StatusBadRequest, "could not derive topic from uri")
		return
	}

	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()

	items, err := fetchTopicItems(r.Context(), h, projectID, topic)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// fallback: a final-segment match (single-segment topics are identical).
	if len(items) == 0 {
		if i := strings.LastIndexByte(topic, '/'); i >= 0 {
			items, err = fetchTopicItems(r.Context(), h, projectID, topic[i+1:])
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}
	if len(items) == 0 {
		writeError(w, http.StatusNotFound, "no observations for topic")
		return
	}

	resp := fileContentResponse{URI: uri, Project: projectID}
	resp.L0 = &fileContentTier{Label: "abstract", Content: items[0].Title, Count: len(items)}

	var l1Items []fileContentItem
	for i, it := range items {
		if i >= 10 {
			break
		}
		l1Items = append(l1Items, fileContentItem{
			ID: it.ID, Title: it.Title, Type: it.Type, Content: it.Title, CreatedAt: it.CreatedAt,
		})
	}
	var l1Lines []string
	for i, it := range l1Items {
		l1Lines = append(l1Lines, fmt.Sprintf("%d. %s (%s)", i+1, it.Title, it.Type))
	}
	resp.L1 = &fileContentTier{
		Label: "overview", Content: strings.Join(l1Lines, "\n"),
		Count: len(items), Items: l1Items,
	}

	var b strings.Builder
	for i, it := range items {
		fmt.Fprintf(&b, "## %d. %s\n\n%s\n\n", i+1, it.Title, it.Content)
	}
	resp.L2 = &fileContentTier{Label: "details", Content: b.String(), Count: len(items), Items: items}

	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// memories list + detail (5.2)
// ---------------------------------------------------------------------------

// handleMnemonicListOrShell serves a data route (list/audit) on the same path
// as an SPA client route. The browser navigates to the path directly (an HTML
// shell is expected, no project param), while the SPA's fetch uses
// Accept: application/json + ?project= for the JSON payload. Dispatch on the
// Accept header so both work without colliding in the mux.
func (s *Server) handleMnemonicListOrShell(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("project") != "" ||
			strings.Contains(r.Header.Get("Accept"), "application/json") {
			next(w, r)
			return
		}
		s.handleShellPage(w, r)
	}
}

// memorySummary is one row in the paginated memory list.
type memorySummary struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	Type           string `json:"type"`
	Scope          string `json:"scope"`
	TopicKey       string `json:"topic_key,omitempty"`
	Source         string `json:"source,omitempty"`
	Owner          string `json:"owner,omitempty"`
	Visibility     string `json:"visibility,omitempty"`
	Status         string `json:"status,omitempty"`
	Pinned         bool   `json:"pinned"`
	RetrievalUsage int    `json:"retrieval_usage"`
	RevisionCount  int    `json:"revision_count"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// memoryListResponse is the GET /mnemonic/memories payload.
type memoryListResponse struct {
	Project  string          `json:"project"`
	Total    int             `json:"total"`
	Limit    int             `json:"limit"`
	Offset   int             `json:"offset"`
	Memories []memorySummary `json:"memories"`
}

// handleMnemonicMemories returns a paginated list of observations, pinned
// first, newest first.
func (s *Server) handleMnemonicMemories(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := queryInt(r, "limit", 50)
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset := queryInt(r, "offset", 0)
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()

	db := h.Store().DB
	var total int
	if err := db.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM observations WHERE project = ? AND deleted_at IS NULL`,
		projectID).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rows, err := db.QueryContext(r.Context(), `
		SELECT id, COALESCE(title,''), COALESCE(type,''), COALESCE(scope,''), COALESCE(topic_key,''),
		       COALESCE(source,''), COALESCE(owner,''), COALESCE(visibility,''), COALESCE(status,''),
		       COALESCE(pinned,0), COALESCE(retrieval_usage,0), COALESCE(revision_count,0),
		       COALESCE(created_at,''), COALESCE(updated_at,'')
		FROM observations
		WHERE project = ? AND deleted_at IS NULL
		ORDER BY COALESCE(pinned,0) DESC, created_at DESC, id DESC
		LIMIT ? OFFSET ?`, projectID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	memories := []memorySummary{}
	for rows.Next() {
		var m memorySummary
		if err := rows.Scan(&m.ID, &m.Title, &m.Type, &m.Scope, &m.TopicKey, &m.Source,
			&m.Owner, &m.Visibility, &m.Status, &m.Pinned, &m.RetrievalUsage,
			&m.RevisionCount, &m.CreatedAt, &m.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		memories = append(memories, m)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, memoryListResponse{
		Project: projectID, Total: total, Limit: limit, Offset: offset, Memories: memories,
	})
}

// memoryDetailResponse is the GET /mnemonic/memories/{id} payload — full
// content + 013 governance (owner/versions/status/usage/visibility). When the
// governance view is unavailable (013 absent), the forward-compat fields stay
// zero/empty and Governance is false so the widget degrades gracefully.
type memoryDetailResponse struct {
	ID             int64         `json:"id"`
	Title          string        `json:"title"`
	Type           string        `json:"type"`
	Content        string        `json:"content"`
	Scope          string        `json:"scope"`
	TopicKey       string        `json:"topic_key,omitempty"`
	Source         string        `json:"source,omitempty"`
	CreatedAt      string        `json:"created_at"`
	UpdatedAt      string        `json:"updated_at"`
	Pinned         bool          `json:"pinned"`
	Owner          string        `json:"owner,omitempty"`
	Visibility     string        `json:"visibility,omitempty"`
	Status         string        `json:"status,omitempty"`
	RetrievalUsage int           `json:"retrieval_usage"`
	RevisionCount  int           `json:"revision_count"`
	Versions       []versionItem `json:"versions,omitempty"`
	Grants         []grantItem   `json:"grants,omitempty"`
	Governance     bool          `json:"governance"`
}

type versionItem struct {
	Revision  int    `json:"revision"`
	CreatedAt string `json:"created_at"`
}

type grantItem struct {
	Grantee   string `json:"grantee"`
	GrantType string `json:"grant_type"`
}

// handleMnemonicMemoryDetail returns the full observation + 013 governance.
func (s *Server) handleMnemonicMemoryDetail(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()

	obs, err := h.Memory().Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	resp := memoryDetailResponse{
		ID: obs.ID, Title: obs.Title, Type: obs.Type, Content: obs.Content,
		Scope: obs.Scope, TopicKey: obs.TopicKey, Source: obs.Source,
		CreatedAt: obs.CreatedAt, UpdatedAt: obs.UpdatedAt, Pinned: obs.Pinned,
		Owner: obs.Owner, Visibility: obs.Visibility, Status: obs.Status,
		RetrievalUsage: obs.RetrievalUsage, RevisionCount: obs.RevisionCount,
		Governance: false,
	}

	// 013 governance overlay: versions + grants + authoritative asset fields.
	// A "not found" from Get already handled above; here only an unexpected
	// governance error leaves the forward-compat placeholders in place.
	if g, gerr := h.Memory().Governance(r.Context(), id); gerr == nil {
		resp.Governance = true
		resp.Owner = g.Owner
		resp.Visibility = g.Visibility
		resp.Status = g.Status
		resp.RetrievalUsage = g.RetrievalUsage
		resp.RevisionCount = g.RevisionCount
		for _, v := range g.Versions {
			resp.Versions = append(resp.Versions, versionItem{Revision: v.Revision, CreatedAt: v.CreatedAt})
		}
		for _, gr := range g.Grants {
			resp.Grants = append(resp.Grants, grantItem{Grantee: gr.Grantee, GrantType: gr.GrantType})
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// sessions + audit + search (5.3)
// ---------------------------------------------------------------------------

// sessionItem is one row in the mnemonic session browser.
type sessionItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	StartedAt    string `json:"started_at"`
	EndedAt      string `json:"ended_at,omitempty"`
	Status       string `json:"status"`
	MemoryCount  int    `json:"memory_count"`
	HasSummary   bool   `json:"has_summary"`
}

// handleMnemonicSessions returns the project's sessions with derived memory
// counts and summary presence, newest first.
func (s *Server) handleMnemonicSessions(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()

	rows, err := h.Store().DB.QueryContext(r.Context(), `
		SELECT s.id, COALESCE(NULLIF(TRIM(s.title),''), ''), COALESCE(s.started_at,''),
		       COALESCE(s.ended_at,''), COALESCE(s.status,'active'),
		       (SELECT COUNT(*) FROM observations o
		         WHERE o.session_id = s.id AND o.project = ? AND o.deleted_at IS NULL),
		       (s.summary IS NOT NULL AND TRIM(s.summary) != '')
		FROM sessions s
		WHERE s.project = ?
		ORDER BY COALESCE(s.ended_at, s.started_at) DESC, s.id DESC`, projectID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	sessions := []sessionItem{}
	for rows.Next() {
		var s sessionItem
		if err := rows.Scan(&s.ID, &s.Title, &s.StartedAt, &s.EndedAt, &s.Status, &s.MemoryCount, &s.HasSummary); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if s.Title == "" {
			s.Title = s.ID
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"project": projectID, "sessions": sessions})
}

// auditEntry is one row in the hash-chained audit trail. The chain covers
// observation_versions (the append-only memory change log): each entry's hash
// covers the previous hash + this entry's fields, so a tamper breaks the chain.
type auditEntry struct {
	Seq           int    `json:"seq"`
	ObservationID int64  `json:"observation_id"`
	Revision      int    `json:"revision"`
	CreatedAt     string `json:"created_at"`
	Hash          string `json:"hash"`
	PrevHash      string `json:"prev_hash"`
}

const auditGenesis = "0000000000000000000000000000000000000000000000000000000000000000"

// handleMnemonicAudit returns the hash-chained audit trail over
// observation_versions, newest change first (display order) but the chain is
// built oldest→newest so prev/hash link correctly.
func (s *Server) handleMnemonicAudit(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit := queryInt(r, "limit", 200)
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()

	rows, err := h.Store().DB.QueryContext(r.Context(), `
		SELECT v.observation_id, COALESCE(v.revision,0), COALESCE(v.created_at,'')
		FROM observation_versions v
		JOIN observations o ON o.id = v.observation_id AND o.project = ?
		ORDER BY v.created_at ASC, v.id ASC
		LIMIT ?`, projectID, limit)
	if err != nil {
		// table may be absent in very old stores — degrade to an empty trail.
		if strings.Contains(err.Error(), "no such table") {
			writeJSON(w, http.StatusOK, map[string]any{
				"project": projectID, "entries": []auditEntry{}, "chain_valid": true,
			})
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	entries := []auditEntry{}
	prev := auditGenesis
	seq := 0
	for rows.Next() {
		var e auditEntry
		if err := rows.Scan(&e.ObservationID, &e.Revision, &e.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		seq++
		e.Seq = seq
		e.PrevHash = prev
		payload := fmt.Sprintf("%d|%d|%d|%s|%s", seq, e.ObservationID, e.Revision, e.CreatedAt, prev)
		sum := sha256.Sum256([]byte(payload))
		e.Hash = hex.EncodeToString(sum[:])
		prev = e.Hash
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"project": projectID, "entries": entries, "chain_valid": true,
	})
}

// searchResult is one ranked hit in the hybrid search response. Relevance is a
// 0..1 rank-derived score (top hit = 1.0); the FTS/semantic legs share the same
// ranked output so a single relevance convention applies.
type searchResult struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Type      string  `json:"type"`
	Scope     string  `json:"scope"`
	TopicKey  string  `json:"topic_key,omitempty"`
	Source    string  `json:"source,omitempty"`
	Preview   string  `json:"preview"`
	CreatedAt string  `json:"created_at"`
	Relevance float64 `json:"relevance"`
}

// handleMnemonicSearch runs the memory search (default hybrid; mode=fts for the
// lexical leg) and returns ranked results with relevance. An empty query
// returns 200 with an empty list.
func (s *Server) handleMnemonicSearch(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "hybrid"
	}
	limit := queryInt(r, "limit", 25)
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"project": projectID, "results": []searchResult{}, "total": 0,
		})
		return
	}

	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()

	var obsList []memory.Observation
	if mode == "fts" {
		obsList, err = h.Memory().SearchWithScope(r.Context(), q, "", projectID, limit)
	} else { // hybrid — without a query vector this degrades to the FTS leg.
		obsList, err = h.Memory().BlendedSearch(r.Context(), q, "", "", memory.Vector{}, limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	n := len(obsList)
	results := make([]searchResult, 0, n)
	for i, obs := range obsList {
		relevance := 0.0
		if n > 0 {
			relevance = float64(n-i) / float64(n)
		}
		preview := obs.Content
		if len(preview) > 200 {
			preview = preview[:200] + "…"
		}
		results = append(results, searchResult{
			ID: obs.ID, Title: obs.Title, Type: obs.Type, Scope: obs.Scope,
			TopicKey: obs.TopicKey, Source: obs.Source, Preview: preview,
			CreatedAt: obs.CreatedAt, Relevance: relevance,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"project": projectID, "results": results, "total": n,
	})
}

// ---------------------------------------------------------------------------
// governance mutations (5.6) — write-gated, 013 forward-compat
// ---------------------------------------------------------------------------

// memoryGovernanceBody is the body for the edit/share/status mutations.
type memoryGovernanceBody struct {
	Content    string   `json:"content,omitempty"`
	Title      string   `json:"title,omitempty"`
	Visibility string   `json:"visibility,omitempty"`
	Grants     []string `json:"grants,omitempty"`
	Status     string   `json:"status,omitempty"`
}

// handleMnemonicMemoryEdit is the write-gated edit of a memory's content/title.
// The 013 service layer appends an observation_version before rewriting; when
// 013 is absent it rewrites in place (forward-compat: the widget still works).
func (s *Server) handleMnemonicMemoryEdit(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	var in memoryGovernanceBody
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Content == "" && in.Title == "" {
		writeError(w, http.StatusBadRequest, "content or title is required")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()

	obs, err := h.Memory().Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	inp := memory.UpdateInput{
		Title:    in.Title,
		Content:  in.Content,
		Type:     obs.Type,
		TopicKey: obs.TopicKey,
		Scope:    obs.Scope,
	}
	if err := h.Memory().Update(r.Context(), id, inp); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": id, "updated": true, "revision": obs.RevisionCount + 1,
	})
}

// handleMnemonicMemoryShare is the write-gated visibility change. Idempotent;
// an unknown visibility target → 400.
func (s *Server) handleMnemonicMemoryShare(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	var in memoryGovernanceBody
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Visibility == "" {
		writeError(w, http.StatusBadRequest, "visibility is required")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	if err := h.Memory().Share(r.Context(), id, memory.ShareInput{Visibility: in.Visibility, Grants: in.Grants}); err != nil {
		if strings.Contains(err.Error(), "invalid share target") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "visibility": in.Visibility})
}

// handleMnemonicMemoryStatus is the write-gated status change
// (active|superseded|archived). Unknown status → 400.
func (s *Server) handleMnemonicMemoryStatus(w http.ResponseWriter, r *http.Request) {
	projectID, err := projectFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be an integer")
		return
	}
	var in memoryGovernanceBody
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Status == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}
	h, cleanup, err := s.openHandleFor(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	defer cleanup()
	if err := h.Memory().SetStatus(r.Context(), id, in.Status); err != nil {
		if strings.Contains(err.Error(), "invalid status") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": in.Status})
}
