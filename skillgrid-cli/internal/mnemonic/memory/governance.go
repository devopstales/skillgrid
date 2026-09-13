package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Observation visibility levels (change 013, step 01). Private-by-default:
// a new observation is `private` until an explicit mem_share.
const (
	VisibilityPrivate     = "private"
	VisibilityTeam        = "team"
	VisibilityRestricted  = "restricted"
	VisibilityAgent       = "agent"
)

// Observation lifecycle statuses. `active` by default; `superseded` /
// `archived` are set explicitly, never inferred from content.
const (
	StatusActive     = "active"
	StatusSuperseded = "superseded"
	StatusArchived   = "archived"
)

// ErrVisibilityNotSet is the (rare) sentinel for an observation whose
// visibility column is empty — normally impossible since the column defaults
// to 'private'. It surfaces a data problem, not a read denial.
var ErrVisibilityNotSet = errors.New("observation visibility not set")

// legacyOwnerID is the owner identity assigned to legacy (pre-017)
// observations whose owner column is empty. The save path guarantees a new
// observation always has an owner (falling back to the session id); the read
// path mirrors that default so a legacy row's visibility is enforced
// consistently with visibilityFilter. It is a sentinel value — a real
// session/agent id will never match it — so a legacy row is private to every
// live reader until it is re-owned via the save path.
const legacyOwnerID = "legacy"

// ErrRestrictedNoGrants is the condition a `restricted` observation with no
// ACL grants presents to a non-owner reader: it is owner-only. It is surfaced
// (not raised as a failure) by governance queries; read paths treat it as
// "absent" (not found) for the reader.
var ErrRestrictedNoGrants = errors.New("restricted observation has no grants (owner-only)")

// IsValidVisibility reports whether vis is one of the four governance
// visibilities.
func IsValidVisibility(vis string) bool {
	switch vis {
	case VisibilityPrivate, VisibilityTeam, VisibilityRestricted, VisibilityAgent:
		return true
	}
	return false
}

// IsValidStatus reports whether st is one of the three lifecycle statuses.
func IsValidStatus(st string) bool {
	switch st {
	case StatusActive, StatusSuperseded, StatusArchived:
		return true
	}
	return false
}

// ShareInput is the argument to mem_share. Visibility must be one of
// team|restricted|agent (private is the default and is not an explicit share
// target in step 01 — a private observation is simply "not shared"). Grants is
// the ACL grantee list, meaningful for restricted (and agent) visibility.
type ShareInput struct {
	Visibility string
	Grants     []string
}

// Share widens an observation's visibility — the ONLY explicit way a new
// observation leaves `private`. Bad args (unknown visibility) are rejected
// with a clear validation error and visibility is left unchanged. For
// restricted visibility the Grants are validated: an empty grant list is
// allowed (owner-only, surfaced via ErrRestrictedNoGrants) but a non-empty
// list is written to acl_grants.
func (s *Service) Share(ctx context.Context, id int64, in ShareInput) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	vis := strings.TrimSpace(in.Visibility)
	if !IsValidVisibility(vis) || vis == VisibilityPrivate {
		return fmt.Errorf("invalid share target %q (valid: %s)", vis, shareTargets)
	}
	// The observation must exist and belong to this project before we touch it.
	var cur string
	if err := s.store.DB.QueryRowContext(ctx, `
		SELECT visibility FROM observations
		WHERE id = ? AND project = ? AND deleted_at IS NULL`, id, s.projectID).Scan(&cur); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("observation %d not found", id)
		}
		return fmt.Errorf("load visibility: %w", err)
	}
	// Validate + replace grants for restricted / agent visibility.
	if vis == VisibilityRestricted || vis == VisibilityAgent {
		for _, g := range in.Grants {
			if strings.TrimSpace(g) == "" {
				return errors.New("empty grantee in ACL")
			}
		}
		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := s.store.DB.ExecContext(ctx, `
			DELETE FROM acl_grants WHERE observation_id = ?`, id); err != nil {
			return fmt.Errorf("clear grants: %w", err)
		}
		for _, g := range in.Grants {
			if _, err := s.store.DB.ExecContext(ctx, `
				INSERT INTO acl_grants (observation_id, grantee, grant_type, created_at)
				VALUES (?, ?, 'agent', ?)`, id, strings.TrimSpace(g), now); err != nil {
				return fmt.Errorf("insert grant: %w", err)
			}
		}
	}
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations SET visibility = ?, updated_at = ?
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		vis, time.Now().UTC().Format(time.RFC3339), id, s.projectID)
	if err != nil {
		return fmt.Errorf("share: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("observation %d not found", id)
	}
	return nil
}

// ShareTargets is the comma-joined list of valid mem_share targets.
const shareTargets = "team|restricted|agent"

// AppendVersion appends a row to observation_versions capturing the
// observation's current (pre-update) state, tagged with the caller-supplied
// revision number. It does NOT advance revision_count — the caller owns that
// (mem_update and the topic-key upsert both increment), so the history row's
// revision always matches the observation's revision_count after the update.
// A mem_update appends (it does not overwrite): the prior content is preserved
// in observation_versions and the latest version becomes the read path.
func (s *Service) AppendVersion(ctx context.Context, id int64, revision int) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	var content string
	if err := s.store.DB.QueryRowContext(ctx, `
		SELECT content FROM observations
		WHERE id = ? AND project = ? AND deleted_at IS NULL`, id, s.projectID).Scan(&content); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("observation %d not found", id)
		}
		return fmt.Errorf("load observation: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.store.DB.ExecContext(ctx, `
		INSERT INTO observation_versions (observation_id, revision, content, created_at)
		VALUES (?, ?, ?, ?)`, id, revision, content, now); err != nil {
		return fmt.Errorf("append version: %w", err)
	}
	return nil
}

// SetStatus sets an observation's lifecycle status explicitly. It is never
// inferred from content: the caller decides. An unknown status is rejected.
func (s *Service) SetStatus(ctx context.Context, id int64, status string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	status = strings.TrimSpace(status)
	if !IsValidStatus(status) {
		return fmt.Errorf("invalid status %q (valid: %s|superseded|archived)", status, StatusActive)
	}
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations SET status = ?, updated_at = ?
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		status, time.Now().UTC().Format(time.RFC3339), id, s.projectID)
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("observation %d not found", id)
	}
	return nil
}

// BumpRetrievalUsage increments an observation's retrieval-usage counter.
// It is the "times returned by a search" counter, distinct from
// duplicate_count (re-saves). Best-effort: callers may ignore the error.
func (s *Service) BumpRetrievalUsage(ctx context.Context, id int64) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return
	}
	_, _ = s.store.DB.ExecContext(ctx, `
		UPDATE observations
		SET retrieval_usage = COALESCE(retrieval_usage, 0) + 1
		WHERE id = ? AND project = ? AND deleted_at IS NULL`, id, s.projectID)
}

// Version is a single row of the append-only version history.
type Version struct {
	Revision  int    `json:"revision"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// Grant is a single ACL grant on a restricted/agent observation.
type Grant struct {
	Grantee   string `json:"grantee"`
	GrantType string `json:"grant_type"`
}

// Governance is the mem_governance <id> result: the asset fields.
type Governance struct {
	ID            int64     `json:"id"`
	Owner         string    `json:"owner"`
	Visibility    string    `json:"visibility"`
	Status        string    `json:"status"`
	RevisionCount int       `json:"revision_count"`
	RetrievalUsage int      `json:"retrieval_usage"`
	Versions      []Version `json:"versions"`
	Grants        []Grant   `json:"grants,omitempty"`
	// RestrictedNoGrants is true when visibility is restricted and no ACL
	// grants exist — the observation is owner-only. It is a condition to be
	// surfaced, not an error.
	RestrictedNoGrants bool `json:"restricted_no_grants,omitempty"`
}

// Governance returns the governed-asset view of an observation: owner,
// append-only version history, status, retrieval usage, visibility, and the
// ACL grants. The latest version (the live content) is the read path; prior
// versions are recoverable here.
func (s *Service) Governance(ctx context.Context, id int64) (Governance, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return Governance{}, errors.New("memory service not initialized")
	}
	var g Governance
	var owner sql.NullString
	if err := s.store.DB.QueryRowContext(ctx, `
		SELECT owner, visibility, status, COALESCE(revision_count,0), COALESCE(retrieval_usage,0)
		FROM observations WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		id, s.projectID).Scan(&owner, &g.Visibility, &g.Status, &g.RevisionCount, &g.RetrievalUsage); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return g, fmt.Errorf("observation %d not found", id)
		}
		return g, fmt.Errorf("load governance: %w", err)
	}
	g.ID = id
	if owner.Valid {
		g.Owner = owner.String
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT revision, content, created_at FROM observation_versions
		WHERE observation_id = ? ORDER BY revision DESC`, id)
	if err != nil {
		return g, fmt.Errorf("load versions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v Version
		if err := rows.Scan(&v.Revision, &v.Content, &v.CreatedAt); err != nil {
			return g, fmt.Errorf("scan version: %w", err)
		}
		g.Versions = append(g.Versions, v)
	}
	if err := rows.Err(); err != nil {
		return g, fmt.Errorf("iterate versions: %w", err)
	}
	grantRows, err := s.store.DB.QueryContext(ctx, `
		SELECT grantee, grant_type FROM acl_grants
		WHERE observation_id = ? ORDER BY id`, id)
	if err != nil {
		return g, fmt.Errorf("load grants: %w", err)
	}
	defer grantRows.Close()
	for grantRows.Next() {
		var gr Grant
		if err := grantRows.Scan(&gr.Grantee, &gr.GrantType); err != nil {
			return g, fmt.Errorf("scan grant: %w", err)
		}
		g.Grants = append(g.Grants, gr)
	}
	if err := grantRows.Err(); err != nil {
		return g, err
	}
	if g.Visibility == VisibilityRestricted && len(g.Grants) == 0 {
		g.RestrictedNoGrants = true
	}
	return g, nil
}

// ErrNotFoundForReader is the read-path result for a visibility-gated
// observation the caller cannot see (private to another owner, restricted to
// a different agent, agent-targeted elsewhere). A reader sees it as "absent"
// (not found), not as an error — the owner always sees their own.
var ErrNotFoundForReader = errors.New("observation not visible to this reader")

// canRead reports whether the given reader (owner identity + reader agent) may
// read the observation, enforcing the visibility/ACL semantics:
//
//   - owner: always.
//   - private: owner only.
//   - team: all members of the project (every same-project reader).
//   - restricted: owner + explicitly granted agents; with no grants it is
//     owner-only (ErrRestrictedNoGrants is surfaced, not an error).
//   - agent: owner + the granted agent (same ACL as restricted).
func (s *Service) canRead(ctx context.Context, obsID int64, owner, readerOwner, readerAgent string) (bool, error) {
	var oOwner sql.NullString
	var vis, status string
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT owner, COALESCE(visibility, 'private'), COALESCE(status, 'active')
		FROM observations WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		obsID, s.projectID).Scan(&oOwner, &vis, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrNotFoundForReader
		}
		return false, fmt.Errorf("load visibility: %w", err)
	}
	// Legacy (pre-017) rows have an empty owner. The save path falls back to
	// the session id so an asset is never unowned; the read path mirrors that
	// default here so a legacy row's visibility is enforced consistently with
	// visibilityFilter (which COALESCEs empty owner to "private"). Without this
	// a legacy row was readable via search (treated private) but errored via
	// ReadAs — the two read paths must agree.
	if !oOwner.Valid || oOwner.String == "" {
		oOwner = sql.NullString{String: legacyOwnerID, Valid: true}
	}
	_ = status
	// The creating owner/agent always sees their own.
	if readerOwner == oOwner.String {
		return true, nil
	}
	// A reader identified only as the owner (no agent) reads via owner match.
	switch vis {
	case VisibilityPrivate:
		return false, nil // owner-only
	case VisibilityTeam:
		return true, nil // all same-project members
	case VisibilityRestricted, VisibilityAgent:
		// owner + explicitly granted agents; no grants ⇒ owner-only.
		var n int
		if err := s.store.DB.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM acl_grants
			WHERE observation_id = ? AND grantee = ?`, obsID, readerAgent).Scan(&n); err != nil {
			return false, fmt.Errorf("load grant: %w", err)
		}
		return n > 0, nil
	default:
		return false, ErrVisibilityNotSet
	}
}

// visibilityFilter builds the WHERE clause (plus its bound args) that a search
// runs under for a reader. It is the "private invisible to other owners"
// enforcement at the search layer: the reader always sees their own; a
// non-owner reader additionally sees team-visible assets and any
// restricted/agent asset they are explicitly granted.
func (s *Service) visibilityFilter(owner, readerOwner, readerAgent string) (clause string, args []any) {
	if readerOwner != "" && readerOwner != owner {
		// Non-owner reader: sees team-visible assets + any restricted/agent
		// asset explicitly granted to them. NOT their own rows (those are
		// owned by owner) and NOT other owners' private assets.
		args = append(args, readerAgent)
		clause = `(COALESCE(o.visibility,'private') = 'team'
			OR (COALESCE(o.visibility,'private') IN ('restricted','agent')
			    AND EXISTS (SELECT 1 FROM acl_grants g
			               WHERE g.observation_id = o.id AND g.grantee = ?)))`
	} else {
		// The reader is the owner (or owner-identity match): see their own +
		// any non-private asset in the project.
		args = append(args, owner)
		clause = `(o.owner = ? OR COALESCE(o.visibility,'private') != 'private')`
	}
	return clause, args
}

// SearchOwner is a search scoped to what a given owner/agent reader may see —
// the read path behind "a second owner's search does not return a private
// observation until it is shared". Within a single store bucket this is the
// single-operator visibility model.
func (s *Service) SearchOwner(ctx context.Context, readerOwner, query, matchMode string, limit int) ([]Observation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	ftsQuery := buildFTSQuery(query, matchMode)
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	clause, clauseArgs := s.visibilityFilter(readerOwner, readerOwner, readerOwner)
	// SQL order: MATCH ?  |  project = ?  |  <clause placeholders>  |  LIMIT ?
	args := []any{ftsQuery, s.projectID}
	args = append(args, clauseArgs...)
	args = append(args, limit)
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.project, o.scope,
		       o.topic_key, o.source, o.normalized_hash, o.revision_count, o.prompt_id, o.created_at, o.updated_at,
		       COALESCE(o.pinned, 0), COALESCE(o.duplicate_count, 0), o.last_seen_at, o.expires_at, o.tool_name,
		       o.owner, COALESCE(o.visibility, 'private'), COALESCE(o.status, 'active'), COALESCE(o.retrieval_usage, 0),
		       o.importance_score, o.recency_decay, o.maturity_tier
		FROM observations o
		INNER JOIN observations_fts ON observations_fts.rowid = o.id
		WHERE observations_fts MATCH ? AND o.project = ? AND o.deleted_at IS NULL
		  AND (o.expires_at IS NULL OR o.expires_at = '' OR strftime('%s', o.expires_at) > strftime('%s', 'now'))
		  AND `+clause+`
		ORDER BY COALESCE(o.pinned, 0) DESC, bm25(observations_fts)
		LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("owner search: %w", err)
	}
	defer rows.Close()
	return scanObservations(rows)
}

// ReadAs returns a single observation for the given reader (owner + agent),
// enforcing visibility/ACL. The creating owner always reads their own; a
// non-owner reads only what they are granted. A reader that cannot see it
// gets ErrNotFoundForReader (absent, not an error).
func (s *Service) ReadAs(ctx context.Context, readerOwner string, id int64, readerAgent string) (Observation, error) {
	ok, err := s.canRead(ctx, id, s.projectID, readerOwner, readerAgent)
	if err != nil {
		return Observation{}, err
	}
	if !ok {
		return Observation{}, ErrNotFoundForReader
	}
	return s.Get(ctx, id)
}

// AdminCrossOwnerList lists the observations an admin (identified as an owner)
// is allowed to surface across owners: the admin's own observations plus
// non-private ones they are granted. Private observations of OTHER owners are
// absent from this list — a private asset is invisible to other owners even
// to an admin's read list.
func (s *Service) AdminCrossOwnerList(ctx context.Context, adminOwner string) ([]Observation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.project, o.scope,
		       o.topic_key, o.source, o.normalized_hash, o.revision_count, o.prompt_id, o.created_at, o.updated_at,
		       COALESCE(o.pinned, 0), COALESCE(o.duplicate_count, 0), o.last_seen_at, o.expires_at, o.tool_name,
		       o.owner, COALESCE(o.visibility, 'private'), COALESCE(o.status, 'active'), COALESCE(o.retrieval_usage, 0),
		       o.importance_score, o.recency_decay, o.maturity_tier
		FROM observations o
		WHERE o.deleted_at IS NULL AND o.project = ?
		  AND (o.owner = ? OR COALESCE(o.visibility, 'private') != 'private')
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT 500`, s.projectID, adminOwner)
	if err != nil {
		return nil, fmt.Errorf("admin cross-owner list: %w", err)
	}
	defer rows.Close()
	return scanObservations(rows)
}
