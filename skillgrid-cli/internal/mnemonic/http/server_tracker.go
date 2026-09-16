package http

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http/tracker"
)

// resolveTracker picks the active provider: per-request ?provider= override,
// then SKILLGRID_TRACKER, then the onboarded tracker doc, then config.yaml,
// defaulting to backlogmd. Unknown values 501 (never guessed).
func resolveTracker(override string) (tracker.Provider, error) {
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

// registerTrackerRoutes mounts /tracker/* plus the /backlog/* alias (which
// only serves when the active provider is Backlog.md).
func (s *Server) registerTrackerRoutes() {
	s.mux.HandleFunc("GET /tracker/config", s.handleTrackerConfig)
	s.mux.HandleFunc("GET /tracker/tasks", s.handleTrackerList)
	s.mux.HandleFunc("GET /tracker/tasks/{id}", s.handleTrackerGet)
	s.mux.HandleFunc("POST /tracker/tasks/{id}/status", s.requireWriteAuth(s.handleTrackerSetStatus))
	s.mux.HandleFunc("GET /backlog/config", s.handleBacklogAlias)
	s.mux.HandleFunc("GET /backlog/tasks", s.handleBacklogAlias)
	s.mux.HandleFunc("GET /backlog/tasks/{id}", s.handleBacklogAlias)
	s.mux.HandleFunc("POST /backlog/tasks/{id}/status", s.requireWriteAuth(s.handleBacklogAlias))
}

func (s *Server) activeTracker(w http.ResponseWriter, r *http.Request) (tracker.Provider, bool) {
	p, err := resolveTracker(r.URL.Query().Get("provider"))
	if err != nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": err.Error()})
		return nil, false
	}
	return p, true
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
		items = []tracker.Item{}
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
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/tasks/"):
		r.SetPathValue("id", strings.TrimPrefix(path, "/tasks/"))
		s.handleTrackerGet(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(path, "/tasks/") && strings.HasSuffix(path, "/status"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/tasks/"), "/status")
		r.SetPathValue("id", id)
		s.handleTrackerSetStatus(w, r)
	default:
		http.NotFound(w, r)
	}
}
