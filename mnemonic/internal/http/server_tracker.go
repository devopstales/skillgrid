package http

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/http/tracker"
)

// lookPath resolves a CLI binary on PATH (thin wrapper so tests can stub it).
var lookPath = exec.LookPath

// resolveTracker picks the active provider: per-request ?provider= override,
// then SKILLGRID_TRACKER, then the onboarded tracker doc, then config.yaml,
// defaulting to backlogmd. Unknown values 501 (never guessed).
func resolveTracker(override string) (tracker.TicketProvider, error) {
	if v := strings.TrimSpace(override); v != "" {
		return tracker.ForName(v, "", "")
	}
	env := strings.TrimSpace(os.Getenv("SKILLGRID_TRACKER"))
	var doc, cfgVal string
	if raw, err := os.ReadFile("docs/skillgrid/agents/issue-tracker.md"); err == nil {
		doc = string(raw)
	}
	if raw, err := os.ReadFile("docs/skillgrid/config.yaml"); err == nil {
		cfgVal = configIssueTracker(string(raw))
	}
	return tracker.ForName(env, doc, cfgVal)
}

// configIssueTracker extracts `issue_tracker: <value>` without a YAML dep.
func configIssueTracker(yaml string) string {
	for _, line := range strings.Split(yaml, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "issue_tracker:") {
			v := strings.TrimSpace(strings.TrimPrefix(t, "issue_tracker:"))
			v = strings.Trim(v, `"'`)
			if i := strings.Index(v, "#"); i >= 0 {
				v = strings.TrimSpace(v[:i])
			}
			return v
		}
	}
	return ""
}

// registerTrackerRoutes mounts /tracker/* CRUD + /tracker/stream (SSE) plus the
// /backlog/* alias (which only serves when the active provider is Backlog.md).
func (s *Server) registerTrackerRoutes() {
	s.mux.HandleFunc("GET /tracker/providers", s.handleTrackerProviders)
	s.mux.HandleFunc("GET /tracker/config", s.handleTrackerConfig)
	s.mux.HandleFunc("GET /tracker/tasks", s.handleTrackerList)
	s.mux.HandleFunc("GET /tracker/tasks/{id}", s.handleTrackerGet)
	s.mux.HandleFunc("PATCH /tracker/tasks/{id}", s.requireWriteAuth(s.handleTrackerPatch))
	s.mux.HandleFunc("POST /tracker/tasks/{id}/status", s.requireWriteAuth(s.handleTrackerSetStatus))
	s.mux.HandleFunc("GET /tracker/tasks/{id}/deps", s.handleTrackerDeps)
	s.mux.HandleFunc("GET /tracker/milestones", s.handleTrackerMilestones)
	s.mux.HandleFunc("GET /tracker/stream", s.handleTrackerStream)
	s.mux.HandleFunc("GET /backlog/config", s.handleBacklogAlias)
	s.mux.HandleFunc("GET /backlog/tasks", s.handleBacklogAlias)
	s.mux.HandleFunc("GET /backlog/tasks/{id}", s.handleBacklogAlias)
	s.mux.HandleFunc("PATCH /backlog/tasks/{id}", s.requireWriteAuth(s.handleBacklogAlias))
	s.mux.HandleFunc("POST /backlog/tasks/{id}/status", s.requireWriteAuth(s.handleBacklogAlias))
}

func (s *Server) activeTracker(w http.ResponseWriter, r *http.Request) (tracker.TicketProvider, bool) {
	p, err := resolveTracker(r.URL.Query().Get("provider"))
	if err != nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": err.Error()})
		return nil, false
	}
	return p, true
}

// handleTrackerProviders reports the active provider + whether its CLI is
// available. CLI-backed providers missing their binary report connected:false
// so the UI can show a "not connected" empty state (never a perpetual spinner).
func (s *Server) handleTrackerProviders(w http.ResponseWriter, r *http.Request) {
	p, ok := s.activeTracker(w, r)
	if !ok {
		return
	}
	connected := true
	reason := ""
	if cli := p.CLIName(); cli != "" {
		if _, err := lookPath(cli); err != nil {
			connected = false
			reason = cli + " CLI not found"
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"provider":  p.Name(),
		"connected": connected,
		"reason":    reason,
		"cli":       p.CLIName(),
	})
}

func (s *Server) handleTrackerConfig(w http.ResponseWriter, r *http.Request) {
	p, ok := s.activeTracker(w, r)
	if !ok {
		return
	}
	cfg, err := p.Config(r.Context())
	if err != nil {
		writeJSON(w, tracker.StatusForHTTP(err), map[string]any{"error": err.Error(), "provider": p.Name()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleTrackerList(w http.ResponseWriter, r *http.Request) {
	p, ok := s.activeTracker(w, r)
	if !ok {
		return
	}
	items, err := p.List(r.Context())
	if err != nil {
		writeJSON(w, tracker.StatusForHTTP(err), map[string]any{"error": err.Error(), "provider": p.Name()})
		return
	}
	if items == nil {
		items = []tracker.UnifiedTask{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": items, "provider": p.Name()})
}

func (s *Server) handleTrackerGet(w http.ResponseWriter, r *http.Request) {
	p, ok := s.activeTracker(w, r)
	if !ok {
		return
	}
	it, err := p.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, tracker.StatusForHTTP(err), map[string]any{"error": err.Error(), "provider": p.Name()})
		return
	}
	writeJSON(w, http.StatusOK, it)
}

// handleTrackerPatch is the Phase 2 write path: PATCH /tracker/tasks/{id} with
// {"status": ...} → SetStatus. Token-gated by requireWriteAuth.
func (s *Server) handleTrackerPatch(w http.ResponseWriter, r *http.Request) {
	p, ok := s.activeTracker(w, r)
	if !ok {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Status) == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}
	it, err := p.SetStatus(r.Context(), r.PathValue("id"), body.Status)
	if err != nil {
		writeJSON(w, tracker.StatusForHTTP(err), map[string]any{"error": err.Error(), "provider": p.Name()})
		return
	}
	writeJSON(w, http.StatusOK, it)
}

func (s *Server) handleTrackerSetStatus(w http.ResponseWriter, r *http.Request) {
	p, ok := s.activeTracker(w, r)
	if !ok {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if strings.TrimSpace(body.Status) == "" {
		writeError(w, http.StatusBadRequest, "status is required")
		return
	}
	it, err := p.SetStatus(r.Context(), r.PathValue("id"), body.Status)
	if err != nil {
		writeJSON(w, tracker.StatusForHTTP(err), map[string]any{"error": err.Error(), "provider": p.Name()})
		return
	}
	writeJSON(w, http.StatusOK, it)
}

// handleTrackerDeps returns the in/out dependency edges for one task (the data
// behind the Kanban dependency mini-graph).
func (s *Server) handleTrackerDeps(w http.ResponseWriter, r *http.Request) {
	p, ok := s.activeTracker(w, r)
	if !ok {
		return
	}
	deps, err := p.Dependencies(r.Context(), r.PathValue("id"))
	if err != nil {
		writeJSON(w, tracker.StatusForHTTP(err), map[string]any{"error": err.Error(), "provider": p.Name()})
		return
	}
	writeJSON(w, http.StatusOK, deps)
}

func (s *Server) handleTrackerMilestones(w http.ResponseWriter, r *http.Request) {
	p, ok := s.activeTracker(w, r)
	if !ok {
		return
	}
	ms, err := p.Milestones(r.Context())
	if err != nil {
		writeJSON(w, tracker.StatusForHTTP(err), map[string]any{"error": err.Error(), "provider": p.Name()})
		return
	}
	if ms == nil {
		ms = []tracker.Milestone{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"milestones": ms, "provider": p.Name()})
}

// handleBacklogAlias serves /backlog/* identically when the tracker is
// Backlog.md, else 501 (the generic /tracker/* surface is canonical).
func (s *Server) handleBacklogAlias(w http.ResponseWriter, r *http.Request) {
	p, ok := s.activeTracker(w, r)
	if !ok {
		return
	}
	if p.Name() != tracker.ProviderBacklogMD {
		writeJSON(w, http.StatusNotImplemented, map[string]any{
			"error":    "/backlog/* serves Backlog.md repos only; use /tracker/*",
			"provider": p.Name(),
		})
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/backlog")
	switch {
	case r.Method == http.MethodGet && path == "/config":
		s.handleTrackerConfig(w, r)
	case r.Method == http.MethodGet && path == "/tasks":
		s.handleTrackerList(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/tasks/") && !strings.HasSuffix(path, "/status") && !strings.HasSuffix(path, "/deps"):
		r.SetPathValue("id", strings.TrimPrefix(path, "/tasks/"))
		s.handleTrackerGet(w, r)
	case (r.Method == http.MethodPatch || r.Method == http.MethodPost) && strings.HasPrefix(path, "/tasks/"):
		rest := strings.TrimPrefix(path, "/tasks/")
		if strings.HasSuffix(rest, "/status") {
			r.SetPathValue("id", strings.TrimSuffix(rest, "/status"))
			s.handleTrackerSetStatus(w, r)
		} else if r.Method == http.MethodPatch {
			r.SetPathValue("id", rest)
			s.handleTrackerPatch(w, r)
		} else {
			http.NotFound(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}
