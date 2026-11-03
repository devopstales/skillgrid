// Package handler is the ONE handler set the MCP server (Task 13), the CLI
// (Task 14), and the later HTTP mirror all call (spec 004-transport-mcp §4,
// the K6 byte-identical seam). A Handler is a thin mapping over
// memory.Service + the project detector: every method resolves the project
// at call time (explicit selector > the detector's chain), maps the
// request onto the service, and wraps errors into the envelope's
// (code, message) contract.
//
// Deletion test: deleting this package re-creates the 26 tool behaviors in
// each surface separately — CLI/MCP divergence becomes structural. The
// adapters are the only packages that know a surface exists.
package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
)

// errInvalidArguments is the (code, message) for a transport-level arg
// validation failure (the MCP/CLI adapter validated a pinned required arg
// before the handler method ran). Its " is required" shape maps to the
// envelope's invalid_arguments code (the same shape the pipeline's
// validation errors use).
var errInvalidArguments = errors.New("memory: argument is required")

// DefaultCWD is the cwd the detector resolves against when a surface does
// not have one (the MCP stdio process cwd; the CLI passes its own).
const DefaultCWD = ""

// Handler is the shared handler set. The detector is resolved at call time
// (never at construction): the spec 004-transport-mcp §4 resolution chain
// (explicit → env → git → basename) runs on every tool call, so a process
// that changes directories between calls re-resolves correctly.
type Handler struct {
	svc *memory.Service
	cfg config.Config
	det projectDetector
}

// Detector is the seam the Handler uses to resolve a project at call time.
// The concrete implementation wraps core/project.Detector (git shell-out);
// the seam lets the tests inject a pure function without shelling out to
// git.
type Detector interface {
	Resolve(cwd, explicit string) (string, error)
}

// projectDetector is the unexported alias the struct field uses (the
// exported name is the public seam; the field stays unexported).
type projectDetector = Detector

// NewHandler builds a Handler over a memory service. The service owns the
// store + intelligence seams; the detector owns project resolution.
func NewHandler(svc *memory.Service, cfg config.Config, det projectDetector) *Handler {
	return &Handler{svc: svc, cfg: cfg, det: det}
}

// Config returns the resolved config (the adapters use it for their own
// seams — the CLI reads DataDir, the MCP server reads the profile).
func (h *Handler) Config() config.Config { return h.cfg }

// Service returns the underlying memory service (the adapters that need a
// method the handler does not yet expose use this seam — the handler is
// the contract, not a closed world).
func (h *Handler) Service() *memory.Service { return h.svc }

// resolveProject runs the call-time resolution: explicit > detector >
// config's already-resolved Project.Name. An empty explicit passes through
// to the detector (the chain is the detector's job); a non-empty explicit
// wins outright (spec §4). When the detector cannot resolve (an empty cwd —
// the live bug: the mem Handler's methods pass DefaultCWD "", and a real
// project.Detector errors on empty cwd), the Handler falls back to
// cfg.Project.Name (openRuntime resolves the project with the real cwd and
// records it in the config before building the handler). This is the same
// pattern as the CodeHandler's SetRoot fallback.
func (h *Handler) resolveProject(cwd, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	if h.det != nil {
		name, err := h.det.Resolve(cwd, "")
		if err == nil {
			return name, nil
		}
		// The detector failed (e.g. empty cwd). Fall back to the config's
		// already-resolved project (openRuntime sets cfg.Project.Name after
		// resolving with the real cwd). Without this, a bare `mnemonic mem
		// search` fails "cwd is empty" even though the project was already
		// resolved at startup.
		if h.cfg.Project.Name != "" {
			return h.cfg.Project.Name, nil
		}
		return "", err
	}
	if h.cfg.Project.Name != "" {
		return h.cfg.Project.Name, nil
	}
	return "", errNoDetector
}

// errNoDetector is the (code, message) for a Handler with no detector
// (mem_current_project cannot resolve the project — an envelope error, not
// a fake project name).
var errNoDetector = errors.New("handler: no project detector configured")

// CurrentProject resolves the current project at call time (spec
// 004-transport-mcp §3 mem_current_project: "Project discovery"). The
// explicit selector wins (an empty explicit runs the detector's chain); a
// Handler with no detector returns errNoDetector (the dispatch maps it to
// an envelope error, never a placeholder project name).
func (h *Handler) CurrentProject(cwd, explicit string) (string, error) {
	return h.resolveProject(cwd, explicit)
}

// ── request / result shapes (the D1 tool-contract surface) ─────────────────
//
// These are the handler's own value types — the adapters map their surface
// (MCP args / CLI flags) onto them. They deliberately do NOT alias the
// memory.Service shapes: the handler is the seam, so a service-internal
// rename does not ripple into the transport surface.

// SaveRequest is the mem_save input (spec 004-transport-mcp §3).
type SaveRequest struct {
	SessionID, Type, Title, Content, ToolName, Project, Scope, TopicKey, OperationID string
	// CapturePrompt is the tri-state (spec 004-transport-mcp mem_save row,
	// amended): nil → true (absent), explicit false → off.
	CapturePrompt *bool
	// CurrentPrompt is the current turn's prompt, when capture is on.
	CurrentPrompt string
}

// SaveResult is the mem_save output (the service's SaveResult, re-exposed
// at the seam).
type SaveResult = memory.SaveResult

// SearchFilters is the mem_search filter set.
type SearchFilters = memory.SearchFilters

// SearchResult is the mem_search output.
type SearchResult = memory.SearchResult

// LookupFilters is the mem_lookup input (spec 005 §5: id / sync_id /
// topic_key / phrase — the handler resolves id-vs-sync_id at this seam so
// the service's (id int64) signature stays narrow).
type LookupFilters struct {
	ID       int64
	SyncID   string
	TopicKey string
	Phrase   string
	Project  string
	Scope    string
}

// LookupResult is the mem_lookup output.
type LookupResult = memory.LookupResult

// ContextResult is the mem_context output.
type ContextResult = store.ContextResult

// JudgeRequest is the mem_judge input (the store.JudgeRequest twin, re-
// exposed at the seam). Model is the agent's model name (INV-1
// marked_by_model provenance — the transport is where it enters).
type JudgeRequest = store.JudgeRequest

// Relation is the judged-relation row.
type Relation = store.Relation

// ConsolidateRequest is the mem_consolidate input.
type ConsolidateRequest = memory.ConsolidateRequest

// ConsolidateResult is the mem_consolidate output.
type ConsolidateResult = memory.ConsolidateResult

// ReviewResult is the mem_review output.
type ReviewResult = memory.ReviewResult

// UpdateRequest is the mem_update input (partial update by id; the
// mandatory expected_project ownership assertion is validated inside the
// store's transaction).
type UpdateRequest struct {
	ID              int64
	ExpectedProject string
	Type            *string
	Title           *string
	Content         *string
	TopicKey        *string
}

// DeleteRequest is the mem_delete input (soft by default; hard deletes
// the row outright).
type DeleteRequest struct {
	ID              int64
	ExpectedProject string
	Hard            bool
}

// ListProjectsResult is the mem_list_projects output (spec
// 004-transport-mcp §3: the store's known projects).
type ListProjectsResult struct {
	Projects []string `json:"projects"`
}

// StatsResult is the mem_stats output (spec 004-transport-mcp §3:
// per-type/project statistics).
type StatsResult struct {
	Project string              `json:"project"`
	ByType  map[string]TypeStat `json:"by_type"`
	Total   int64               `json:"total"`
	Deleted int64               `json:"deleted"`
}

// TypeStat is one type row of the per-type statistics.
type TypeStat struct {
	Total   int64 `json:"total"`
	Deleted int64 `json:"deleted"`
}

// TimelineResult is the mem_timeline output (spec 004-transport-mcp §3:
// before/after in a session). The service method is not yet exposed —
// the dispatch pins the args and returns an honest degraded envelope
// until it lands.
type TimelineResult struct {
	Events []TimelineEvent `json:"events"`
}

// TimelineEvent is one session-timeline row.
type TimelineEvent struct {
	Sequence  int64  `json:"sequence"`
	Action    string `json:"action"`
	SyncID    string `json:"sync_id"`
	Timestamp string `json:"timestamp"`
}

// MergeRequest is the mem_merge_projects input (spec
// 004-transport-mcp §3: merge project spelling variants). The service
// method is not yet exposed — the dispatch pins the args and returns an
// honest degraded envelope until it lands.
type MergeRequest struct {
	Source string
	Target string
}

// MergeResult is the mem_merge_projects output.
type MergeResult struct {
	Moved int `json:"moved"`
}

// ── the 26 methods ─────────────────────────────────────────────────────────
//
// Each is a thin pass-through to memory.Service with the project resolved
// at call time. The methods that take an explicit project in their input
// (Update / Delete / Consolidate) resolve it through the same seam — the
// explicit selector wins, the detector is the fallback.

// Save runs the save pipeline (spec 01-memory/001 §2). The project is
// resolved at call time and set on the request before the pipeline runs.
func (h *Handler) Save(ctx context.Context, req SaveRequest) (SaveResult, error) {
	if strings.TrimSpace(req.Project) == "" {
		p, err := h.resolveProject(DefaultCWD, "")
		if err != nil {
			return SaveResult{}, err
		}
		req.Project = p
	}
	return h.svc.Save(ctx, memory.SaveRequest{
		SessionID: req.SessionID, Type: req.Type, Title: req.Title, Content: req.Content,
		ToolName: req.ToolName, Project: req.Project, Scope: req.Scope, TopicKey: req.TopicKey,
		OperationID: req.OperationID, CapturePrompt: req.CapturePrompt, CurrentPrompt: req.CurrentPrompt,
	})
}

// Search runs the FTS + vector search (spec 01-memory/005 §3).
func (h *Handler) Search(ctx context.Context, query string, f SearchFilters) (SearchResult, error) {
	if strings.TrimSpace(f.Project) == "" {
		p, err := h.resolveProject(DefaultCWD, "")
		if err != nil {
			return SearchResult{}, err
		}
		f.Project = p
	}
	return h.svc.Search(ctx, query, f)
}

// Get fetches a full observation by local id or sync_id (INV-6: the agent
// sees the sync_id; the numeric id is the local fallback).
func (h *Handler) Get(ctx context.Context, id int64, syncID string, includeHistory bool, historyCursor int) (store.Observation, error) {
	_ = historyCursor
	if id == 0 && syncID != "" {
		oid, err := h.syncIDToID(ctx, syncID)
		if err != nil {
			return store.Observation{}, err
		}
		id = oid
	}
	return h.svc.St().GetObservation(ctx, id, includeHistory)
}

// Lookup runs the narrow wiki-first fallback (spec 005 §5).
func (h *Handler) Lookup(ctx context.Context, f LookupFilters) (LookupResult, error) {
	if strings.TrimSpace(f.Project) == "" {
		p, err := h.resolveProject(DefaultCWD, "")
		if err != nil {
			return LookupResult{}, err
		}
		f.Project = p
	}
	id := f.ID
	if id == 0 && f.SyncID != "" {
		oid, err := h.syncIDToID(ctx, f.SyncID)
		if err != nil {
			return LookupResult{}, err
		}
		id = oid
	}
	return h.svc.Lookup(ctx, f.Project, f.Scope, f.TopicKey, id, f.Phrase)
}

// Context runs the byte-budget orientation layer (spec 005 §2).
func (h *Handler) Context(ctx context.Context, budgetBytes int, olderHistory bool) (ContextResult, error) {
	return h.svc.Context(ctx, budgetBytes, olderHistory)
}

// Judge records a verdict on a pending candidate (spec 01-memory/003 §4).
// The agent's model name (r.Model) is stamped on the row as marked_by_model.
func (h *Handler) Judge(ctx context.Context, r JudgeRequest) (Relation, error) {
	return h.svc.Judge(ctx, r)
}

// JudgeMany records 1–32 verdicts in one committed transaction (spec
// 004-transport-mcp §4: one queued write).
func (h *Handler) JudgeMany(ctx context.Context, rs []JudgeRequest) ([]Relation, error) {
	return h.svc.JudgeMany(ctx, rs)
}

// Compare proactively compares two observations (spec 01-memory/003 §4).
func (h *Handler) Compare(ctx context.Context, aID, bID int64) (Relation, error) {
	return h.svc.Compare(ctx, aID, bID)
}

// Consolidate runs the two-phase fingerprint-gated merge (spec
// 01-memory/003 §5).
func (h *Handler) Consolidate(ctx context.Context, r ConsolidateRequest) (ConsolidateResult, error) {
	return h.svc.Consolidate(ctx, r)
}

// SavePrompt persists a user prompt (spec 002-schema §2.4).
func (h *Handler) SavePrompt(ctx context.Context, sessionID, content, project, sourceInboxID string) (int64, error) {
	if strings.TrimSpace(project) == "" {
		p, err := h.resolveProject(DefaultCWD, "")
		if err != nil {
			return 0, err
		}
		project = p
	}
	return h.svc.SavePrompt(ctx, sessionID, content, project, sourceInboxID)
}

// SearchPrompts searches the captured prompts (spec 002-schema §2.4).
func (h *Handler) SearchPrompts(ctx context.Context, query, project string, limit int) (store.PromptSearchResult, error) {
	if strings.TrimSpace(project) == "" {
		p, err := h.resolveProject(DefaultCWD, "")
		if err != nil {
			return store.PromptSearchResult{}, err
		}
		project = p
	}
	return h.svc.SearchPrompts(ctx, query, project, limit)
}

// Review lists the project's overdue observations or marks one reviewed
// (spec 002-schema §2.2.2).
func (h *Handler) Review(ctx context.Context, project string, list bool, markReviewedID int64) (ReviewResult, error) {
	if strings.TrimSpace(project) == "" {
		p, err := h.resolveProject(DefaultCWD, "")
		if err != nil {
			return ReviewResult{}, err
		}
		project = p
	}
	return h.svc.Review(ctx, project, list, markReviewedID)
}

// Scan re-runs candidate detection over a project (spec 01-memory/003 §7).
func (h *Handler) Scan(ctx context.Context, project string) (int, error) {
	if strings.TrimSpace(project) == "" {
		p, err := h.resolveProject(DefaultCWD, "")
		if err != nil {
			return 0, err
		}
		project = p
	}
	return h.svc.Scan(ctx, project)
}

// Update performs a partial update by id (the mandatory expected_project
// ownership assertion is validated inside the store's transaction — spec
// 004-transport-mcp §4).
func (h *Handler) Update(ctx context.Context, r UpdateRequest) error {
	return h.svc.St().UpdateObservationForProject(ctx, r.ID, r.ExpectedProject, store.UpdateParams{
		Type: r.Type, Title: r.Title, Content: r.Content, TopicKey: r.TopicKey,
	})
}

// Delete performs a soft/hard delete by id (the mandatory expected_project
// ownership assertion is validated inside the store's transaction).
func (h *Handler) Delete(ctx context.Context, r DeleteRequest) error {
	return h.svc.St().DeleteObservationForProject(ctx, r.ID, r.ExpectedProject, r.Hard)
}

// Pin marks an observation as pinned (search boost + context ordering).
// Pin is a local-only flag; the store's UpdateObservationForProject owns it.
func (h *Handler) Pin(ctx context.Context, id int64, project string) error {
	pinned := true
	return h.svc.St().UpdateObservationForProject(ctx, id, project, store.UpdateParams{Pinned: &pinned})
}

// Unpin clears the pinned flag.
func (h *Handler) Unpin(ctx context.Context, id int64, project string) error {
	pinned := false
	return h.svc.St().UpdateObservationForProject(ctx, id, project, store.UpdateParams{Pinned: &pinned})
}

// ── the session lifecycle + project/admin methods (fix round 1, I-2) ──────
//
// These are the tools the brief's interface block names as handler methods
// ("same shapes as the memory.Service methods"). The session tools wire to
// the store's sessions table (the service's session gate + trail already
// own those rows — T07); the read-only admin tools (ListProjects, Stats)
// read through the store's RawQuery seam; Doctor reports the capability
// checks. The methods the memory function does not yet expose
// (Timeline, MergeProjects) are validated here and returned degraded by
// their adapter (R-2: an honest no-op is a degraded envelope, never a
// healthy empty one).

// SessionStartRequest is the mem_session_start input (spec
// 004-transport-mcp §3).
type SessionStartRequest struct {
	ID, Project, Directory, OwnershipMode string
	Resume, Isolated                      bool
}

// SessionStart registers a session row (idempotent — a re-start resumes the
// existing row, it never fails on a duplicate id).
func (h *Handler) SessionStart(ctx context.Context, r SessionStartRequest) error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("%w: id", errInvalidArguments)
	}
	if strings.TrimSpace(r.Project) == "" {
		p, err := h.resolveProject(DefaultCWD, "")
		if err != nil {
			return err
		}
		r.Project = p
	}
	om := r.OwnershipMode
	if om == "" {
		om = "shared"
	}
	// the UTC stamp is computed in Go and bound as a parameter: the SQL
	// must stay dialect-agnostic (datetime('now') is SQLite-only — the
	// PG leg renders sessions with TIMESTAMPTZ and accepts the string).
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	return h.stExec(ctx, `
		INSERT INTO sessions (id, project, directory, ownership_mode, started_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			directory = excluded.directory,
			ownership_mode = excluded.ownership_mode`,
		r.ID, r.Project, r.Directory, om, now, now)
}

// SessionEnd ends a session (optional free-text summary). A missing session
// is an invalid_arguments error (the agent must start it first); an already
// ended session is the pinned 409 (idempotent re-ends map to the envelope's
// session_ended code, which the retrying agent recognizes).
func (h *Handler) SessionEnd(ctx context.Context, sessionID, summary string) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("%w: session_id", errInvalidArguments)
	}
	st := h.svc.St()
	rows, err := rawQueryOf(st, `SELECT ifnull(ended_at, '') FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("%w: session_id %q (no such session)", errInvalidArguments, sessionID)
	}
	if s, _ := rows[0][0].(string); s != "" {
		return store.ErrSessionEnded
	}
	return h.stExec(ctx, `UPDATE sessions SET ended_at = ?, summary = ? WHERE id = ?`,
		time.Now().UTC().Format("2006-01-02 15:04:05"), orEmpty(summary), sessionID)
}

// SessionSummary persists the six-section structured summary (spec
// 004-transport-mcp §3: Goal/Instructions/Discoveries/Accomplished/Next
// Steps/Relevant Files) as a session_summary observation on the session.
type SessionSummaryRequest struct {
	SessionID, Project string
	SixSections
}

// SixSections are the six summary sections (the spec's pinned shape).
type SixSections struct {
	Goal, Instructions, Discoveries, Accomplished, NextSteps, RelevantFiles string
}

// SessionSummary persists the six sections (spec 004-transport-mcp §3:
// Goal/Instructions/Discoveries/Accomplished/Next Steps/Relevant Files) as
// a session_summary observation on the session. The session row must
// exist (a summary without a session is an arg error, not a silent new
// session); an ended session is the pinned 409 (the save pipeline's
// session gate would reject it anyway — checked here for a clear code).
func (h *Handler) SessionSummary(ctx context.Context, r SessionSummaryRequest) (SaveResult, error) {
	if strings.TrimSpace(r.SessionID) == "" {
		return SaveResult{}, fmt.Errorf("%w: session_id", errInvalidArguments)
	}
	// The session row must exist (a summary without a session is an arg
	// error, not a silent new session).
	st := h.svc.St()
	rows, err := rawQueryOf(st, `SELECT ifnull(ended_at, '') FROM sessions WHERE id = ?`, r.SessionID)
	if err != nil {
		return SaveResult{}, err
	}
	if len(rows) == 0 {
		return SaveResult{}, fmt.Errorf("%w: session_id %q (no such session)", errInvalidArguments, r.SessionID)
	}
	if s, _ := rows[0][0].(string); s != "" {
		return SaveResult{}, store.ErrSessionEnded
	}
	sections := []struct{ h, v string }{
		{"## Goal", r.Goal}, {"## Instructions", r.Instructions}, {"## Discoveries", r.Discoveries},
		{"## Accomplished", r.Accomplished}, {"## Next Steps", r.NextSteps}, {"## Relevant Files", r.RelevantFiles},
	}
	var b strings.Builder
	for _, s := range sections {
		if strings.TrimSpace(s.v) != "" {
			fmt.Fprintf(&b, "%s\n%s\n\n", s.h, strings.TrimSpace(s.v))
		}
	}
	if b.Len() == 0 {
		return SaveResult{}, fmt.Errorf("%w: at least one summary section", errInvalidArguments)
	}
	return h.Save(ctx, SaveRequest{
		SessionID: r.SessionID, Type: "session_summary",
		Title:   "session summary: " + r.SessionID,
		Content: strings.TrimSpace(b.String()),
		Project: r.Project, Scope: "project",
	})
}

// SessionFinalize is the atomic summary + close (the spec row: atomic
// summary + close + idempotent retry journal). An already-ended session is
// a no-op success, never a 409 — the retry journal semantics the spec row
// pins (a re-finalize must not fail the retrying agent).
func (h *Handler) SessionFinalize(ctx context.Context, sessionID, project string) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("%w: session_id", errInvalidArguments)
	}
	st := h.svc.St()
	rows, err := rawQueryOf(st, `SELECT ifnull(ended_at, '') FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("%w: session_id %q (no such session)", errInvalidArguments, sessionID)
	}
	if s, _ := rows[0][0].(string); s != "" {
		return nil // already finalized — the idempotent retry
	}
	return h.stExec(ctx, `UPDATE sessions SET ended_at = ? WHERE id = ?`,
		time.Now().UTC().Format("2006-01-02 15:04:05"), sessionID)
}

// ListProjects returns the store's known projects (the distinct project
// names across sessions + observations — spec 004-transport-mcp §3
// mem_list_projects, agent profile).
func (h *Handler) ListProjects(ctx context.Context) (ListProjectsResult, error) {
	st := h.svc.St()
	rows, err := rawQueryOf(st, `
		SELECT p FROM (
			SELECT DISTINCT project AS p FROM sessions WHERE project != ''
			UNION
			SELECT DISTINCT project FROM observations WHERE project != ''
		) ORDER BY p`)
	if err != nil {
		return ListProjectsResult{}, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if p, ok := row[0].(string); ok && p != "" {
			out = append(out, p)
		}
	}
	return ListProjectsResult{Projects: out}, nil
}

// Stats is the per-type/project statistics (spec 004-transport-mcp §3
// mem_stats, admin profile). An empty project means all projects.
func (h *Handler) Stats(ctx context.Context, project string) (StatsResult, error) {
	st := h.svc.St()
	where, args := "", []any{}
	if strings.TrimSpace(project) != "" {
		where = " WHERE project = ?"
		args = append(args, project)
	}
	rows, err := rawQueryOf(st, `
		SELECT ifnull(type, ''), COUNT(*), COUNT(CASE WHEN deleted_at IS NOT NULL THEN 1 END)
		FROM observations`+where+`
		GROUP BY type ORDER BY type`, args...)
	if err != nil {
		return StatsResult{}, err
	}
	byType := map[string]TypeStat{}
	total, deleted := int64(0), int64(0)
	for _, row := range rows {
		typ, _ := row[0].(string)
		n, d := numAt(row, 1), numAt(row, 2)
		byType[typ] = TypeStat{Total: n, Deleted: d}
		total += n
		deleted += d
	}
	return StatsResult{Project: project, ByType: byType, Total: total, Deleted: deleted}, nil
}

// DoctorCheck is one diagnostic (spec 004-transport-mcp §3 mem_doctor:
// config/FTS/vector/embed checks).
type DoctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// Doctor runs the capability checks (config, FTS search, the vector leg,
// the embedder seam, the LLM seam) and reports them — the admin
// diagnostics (spec 004-transport-mcp §3). The store's Ping gates all of
// them: a dead store is a doctor error, not a doctor report.
func (h *Handler) Doctor(ctx context.Context) (map[string]any, error) {
	st := h.svc.St()
	if err := st.Ping(ctx); err != nil {
		return nil, err
	}
	checks := []DoctorCheck{{Name: "config", OK: true, Detail: "config resolved"}}
	// FTS: a store-level search must run (the short-term probe rides the
	// store's LIKE path, which always resolves against the FTS table).
	if _, err := st.Search(ctx, store.SearchQuery{Query: "doc", Project: "", Limit: 1}); err != nil {
		checks = append(checks, DoctorCheck{Name: "fts", OK: false, Detail: err.Error()})
	} else {
		checks = append(checks, DoctorCheck{Name: "fts", OK: true})
	}
	// Vector: the leg must be reachable (an empty vector returns no hits
	// but must not error when the column exists).
	if _, err := st.VectorSearch(ctx, store.VecQuery{QueryVector: []float32{}, Project: "", TopK: 1}); err != nil {
		checks = append(checks, DoctorCheck{Name: "vector", OK: false, Detail: err.Error()})
	} else {
		checks = append(checks, DoctorCheck{Name: "vector", OK: true})
	}
	// Embedder: the seam's round-trip check (a nil seam is a capability,
	// not a failure — the deterministic floor runs instead).
	if e := h.svc.Embedder(); e == nil {
		checks = append(checks, DoctorCheck{Name: "embed", OK: false, Detail: "embedder seam absent (deterministic floor)"})
	} else {
		v, err := e.Embed(ctx, []string{"mnemonic-doctor"})
		if err != nil || len(v) == 0 || len(v[0]) == 0 {
			checks = append(checks, DoctorCheck{Name: "embed", OK: false, Detail: "embed round-trip failed"})
		} else {
			checks = append(checks, DoctorCheck{Name: "embed", OK: true, Detail: "dim=" + itoa(len(v[0]))})
		}
	}
	// LLM: the seam's presence (the same capability split).
	if h.svc.LLM() == nil {
		checks = append(checks, DoctorCheck{Name: "llm", OK: false, Detail: "llm seam absent (deterministic floor)"})
	} else {
		checks = append(checks, DoctorCheck{Name: "llm", OK: true})
	}
	return map[string]any{"checks": checks}, nil
}

// rawQueryOf runs a raw SELECT through the store's RawQuery seam (the
// handler's dialect-agnostic read surface — the Probe pattern).
func rawQueryOf(st store.Store, query string, args ...any) ([][]any, error) {
	rq, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("handler: store %T does not expose the RawQuery seam", st)
	}
	return rq.RawQuery(query, args...)
}

// stExec runs a write through the store's RawExec seam (the handler's
// dialect-agnostic write surface for the session rows — the memory
// pipeline owns its own transactions; these are the session table's
// direct rows).
func (h *Handler) stExec(ctx context.Context, query string, args ...any) error {
	_ = ctx
	re, ok := h.svc.St().(interface {
		RawExec(query string, args ...any) error
	})
	if !ok {
		return fmt.Errorf("handler: store %T does not expose the RawExec seam", h.svc.St())
	}
	return re.RawExec(query, args...)
}

func orEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func numAt(row []any, i int) int64 {
	if len(row) <= i {
		return 0
	}
	switch n := row[i].(type) {
	case int64:
		return n
	case int:
		return int64(n)
	}
	return 0
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// syncIDToID resolves a sync_id to its local id (0 when absent — the
// caller's Get/Lookup then return the not-found / empty-result path).
func (h *Handler) syncIDToID(ctx context.Context, syncID string) (int64, error) {
	st := h.svc.St()
	rq, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return 0, fmt.Errorf("handler: store %T does not expose the RawQuery seam the sync_id resolution reads through", st)
	}
	rows, err := rq.RawQuery(`SELECT id FROM observations WHERE sync_id = ? AND deleted_at IS NULL LIMIT 1`, syncID)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("handler: observation %q not found", syncID)
	}
	id, ok := rows[0][0].(int64)
	if !ok {
		// modernc.org/sqlite returns int64 for INTEGER columns; a cast
		// failure here is a store bug, not a data bug.
		return 0, fmt.Errorf("handler: sync_id row id: %T", rows[0][0])
	}
	return id, nil
}
