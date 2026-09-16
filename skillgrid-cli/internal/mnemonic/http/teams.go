package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func (s *Server) registerTeamsRoutes() {
	s.mux.HandleFunc("POST /teams/tasks", s.requireWriteAuth(s.handleTeamsSpawnTask))
	s.mux.HandleFunc("POST /teams/tasks/pull", s.requireWriteAuth(s.handleTeamsPullTask))
	s.mux.HandleFunc("GET /teams/tasks/{id}", s.handleTeamsGetTask)
	s.mux.HandleFunc("POST /teams/tasks/{id}/output", s.requireWriteAuth(s.handleTeamsSubmitOutput))
	s.mux.HandleFunc("POST /teams/tasks/{id}/reviews", s.requireWriteAuth(s.handleTeamsSubmitReview))
	s.mux.HandleFunc("POST /teams/tasks/{id}/done", s.requireWriteAuth(s.handleTeamsMarkDone))
}

func (s *Server) handleTeamsSpawnTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Directory string `json:"directory"`
		TeamID    string `json:"team_id"`
		Title     string `json:"title"`
		Brief     string `json:"brief"`
		Priority  int    `json:"priority"`
		CreatedBy string `json:"created_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	dir := body.Directory
	if dir == "" {
		dir = r.URL.Query().Get("directory")
	}
	id, err := s.svc.SpawnTask(r.Context(), service.SpawnTaskParams{
		Directory: dir,
		TeamID:    body.TeamID,
		Title:     body.Title,
		Brief:     body.Brief,
		Priority:  body.Priority,
		CreatedBy: body.CreatedBy,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"task_id": id, "status": "pending"})
}

func (s *Server) handleTeamsPullTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Directory string `json:"directory"`
		MemberID  string `json:"member_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	dir := body.Directory
	if dir == "" {
		dir = r.URL.Query().Get("directory")
	}
	view, err := s.svc.PullNextTask(r.Context(), dir, body.MemberID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrNoPendingTasks) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleTeamsGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dir := r.URL.Query().Get("directory")
	view, brief, err := s.svc.ReadTask(r.Context(), dir, id)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrUnknownTask) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"task":  view,
		"brief": brief,
	})
}

func (s *Server) handleTeamsSubmitOutput(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Directory string `json:"directory"`
		Output    string `json:"output"`
		Summary   string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	dir := body.Directory
	if dir == "" {
		dir = r.URL.Query().Get("directory")
	}
	if err := s.svc.SubmitOutput(r.Context(), dir, id, body.Summary, body.Output); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrUnknownTask) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task_id": id, "status": "review_spec"})
}

func (s *Server) handleTeamsSubmitReview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Directory  string `json:"directory"`
		ReviewerID string `json:"reviewer_id"`
		ReviewType string `json:"review_type"`
		Passed     bool   `json:"passed"`
		Comments   string `json:"comments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	dir := body.Directory
	if dir == "" {
		dir = r.URL.Query().Get("directory")
	}
	if err := s.svc.SubmitReview(r.Context(), service.SubmitReviewParams{
		Directory:  dir,
		TaskID:     id,
		ReviewerID: body.ReviewerID,
		ReviewType: body.ReviewType,
		Passed:     body.Passed,
		Comments:   body.Comments,
	}); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrUnknownTask) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task_id": id, "passed": body.Passed})
}

func (s *Server) handleTeamsMarkDone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dir := r.URL.Query().Get("directory")
	if r.Body != nil {
		var body struct {
			Directory string `json:"directory"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Directory != "" {
			dir = body.Directory
		}
	}
	if err := s.svc.MarkDone(r.Context(), dir, id); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrUnknownTask) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task_id": id, "status": "done"})
}

// teamsPathIsDistinctFromMemoryReviews is a compile-time reminder used by tests.
func teamsPathIsDistinctFromMemoryReviews(path string) bool {
	return strings.HasPrefix(path, "/teams/") && !strings.HasPrefix(path, "/memory/reviews")
}
