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

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
)

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

// resolveProject runs the call-time resolution: explicit > detector. An
// empty explicit passes through to the detector (the chain is the
// detector's job); a non-empty explicit wins outright (spec §4).
func (h *Handler) resolveProject(cwd, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	if h.det == nil {
		return "", errNoDetector
	}
	name, err := h.det.Resolve(cwd, "")
	if err != nil {
		return "", err
	}
	return name, nil
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
