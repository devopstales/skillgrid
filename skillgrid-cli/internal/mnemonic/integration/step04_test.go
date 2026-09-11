package integration

import (
	"bytes"
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	mnemonichttp "github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// step04Server builds a dashboard HTTP handler backed by a seeded memory store,
// returning the handler, the active project id, and the id of a saved
// observation to exercise the routes against.
func step04Server(t *testing.T, token string) (stdhttp.Handler, string, int64) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	if token != "" {
		t.Setenv("SKILLGRID_HTTP_TOKEN", token)
	}
	workspace := seedWorkspace(t)
	svc := service.New(dataDir)
	ctx := context.Background()
	sessID, projectID, err := svc.SessionStart(ctx, workspace, "step04 probe")
	if err != nil {
		t.Fatalf("session start: %v", err)
	}
	saveObservation(t, ctx, svc, projectID, memory.SaveInput{
		SessionID: sessID, Type: "decision", Title: "step04 asset", Content: "the full body",
	})
	// The save is the most recent, so its id is the head of the recent list.
	recent, err := handleFor(t, svc, projectID).Memory().Recent(ctx, 5)
	if err != nil || len(recent) == 0 {
		t.Fatalf("recent after save: %v (n=%d)", err, len(recent))
	}
	id := recent[0].ID
	return mnemonichttp.NewServer(svc).Handler(), projectID, id
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// doHTTPAuth is doHTTP with an Authorization bearer header attached (token may
// be "" for the open-read cases).
func doHTTPAuth(t *testing.T, h stdhttp.Handler, method, target string, body any, token string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	r := httptest.NewRequest(method, target, reader)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	var out map[string]any
	_ = json.NewDecoder(bytes.NewReader(rr.Body.Bytes())).Decode(&out)
	return rr, out
}

// TestStep04_Authz pins the write-gate on pin/unpin and the open read on
// GET /observations/{id} when SKILLGRID_HTTP_TOKEN is set.
func TestStep04_Authz(t *testing.T) {
	h, projectID, id := step04Server(t, "secret")
	q := "?project=" + projectID

	// GET /observations/{id} is an open read: 200 without a token.
	rr, out := doHTTP(t, h, stdhttp.MethodGet, "/observations/"+itoa(id)+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("open GET /observations/{id}: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if out["id"] == nil {
		t.Errorf("GET /observations/{id} should return the observation, got %v", out)
	}

	// POST pin without token → 401.
	rr, _ = doHTTP(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/pin"+q, nil)
	if rr.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("pin without token: expected 401, got %d", rr.Code)
	}
	rr, _ = doHTTP(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/unpin"+q, nil)
	if rr.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("unpin without token: expected 401, got %d", rr.Code)
	}

	// POST pin with token → 200.
	rr, _ = doHTTPAuth(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/pin"+q, nil, "secret")
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("pin with token: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestStep04_ObservationDetail: GET /observations/{id} returns the full,
// untruncated content (same shape as mem_get_observation); unknown id → 404.
func TestStep04_ObservationDetail(t *testing.T) {
	h, projectID, id := step04Server(t, "")
	q := "?project=" + projectID
	rr, out := doHTTP(t, h, stdhttp.MethodGet, "/observations/"+itoa(id)+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("GET /observations/{id}: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if c, _ := out["content"].(string); c != "the full body" {
		t.Errorf("content = %q, want full untruncated \"the full body\"", c)
	}
	// Unknown id → 404.
	rr, _ = doHTTP(t, h, stdhttp.MethodGet, "/observations/999999"+q, nil)
	if rr.Code != stdhttp.StatusNotFound {
		t.Fatalf("unknown id: expected 404, got %d", rr.Code)
	}
}

// TestStep04_PinUnpin: pin is idempotent, unpin reverts, and the pinned flag
// is reflected on the read-back.
func TestStep04_PinUnpin(t *testing.T) {
	h, projectID, id := step04Server(t, "")
	q := "?project=" + projectID

	// Pin → 200, then pin again → 200 (idempotent).
	for i := 0; i < 2; i++ {
		rr, _ := doHTTPAuth(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/pin"+q, nil, "")
		if rr.Code != stdhttp.StatusOK {
			t.Fatalf("pin #%d: expected 200, got %d", i+1, rr.Code)
		}
	}
	// Re-fetch: pinned flag true.
	rr, out := doHTTP(t, h, stdhttp.MethodGet, "/observations/"+itoa(id)+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("refetch after pin: got %d", rr.Code)
	}
	if p, _ := out["pinned"].(bool); !p {
		t.Errorf("pinned = %v, want true after pin", out["pinned"])
	}
	// Unpin → 200; refetch: pinned false.
	rr, _ = doHTTPAuth(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/unpin"+q, nil, "")
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("unpin: expected 200, got %d", rr.Code)
	}
	rr, out = doHTTP(t, h, stdhttp.MethodGet, "/observations/"+itoa(id)+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("refetch after unpin: got %d", rr.Code)
	}
	if p, _ := out["pinned"].(bool); p {
		t.Errorf("pinned = %v, want false after unpin", out["pinned"])
	}
}

// TestStep04_GovernanceWriteGated: share + status-change are write-gated
// (401 without token, 200 with). The data model already lands 013 step-01
// governance (visibility/status columns), so these routes just expose them.
func TestStep04_GovernanceWriteGated(t *testing.T) {
	h, projectID, id := step04Server(t, "secret")
	q := "?project=" + projectID

	// share without token → 401.
	rr, _ := doHTTP(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/share"+q, map[string]any{"visibility": "team"})
	if rr.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("share without token: expected 401, got %d", rr.Code)
	}
	// status without token → 401.
	rr, _ = doHTTP(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/status"+q, map[string]any{"status": "superseded"})
	if rr.Code != stdhttp.StatusUnauthorized {
		t.Fatalf("status without token: expected 401, got %d", rr.Code)
	}
	// share with token → 200, visibility now team.
	rr, _ = doHTTPAuth(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/share"+q, map[string]any{"visibility": "team"}, "secret")
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("share with token: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	rr, out := doHTTP(t, h, stdhttp.MethodGet, "/observations/"+itoa(id)+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("refetch after share: got %d", rr.Code)
	}
	if v, _ := out["visibility"].(string); v != "team" {
		t.Errorf("visibility = %q, want team", v)
	}
	// status with token → 200, status now superseded.
	rr, _ = doHTTPAuth(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/status"+q, map[string]any{"status": "superseded"}, "secret")
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("status with token: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	rr, out = doHTTP(t, h, stdhttp.MethodGet, "/observations/"+itoa(id)+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("refetch after status: got %d", rr.Code)
	}
	if v, _ := out["status"].(string); v != "superseded" {
		t.Errorf("status = %q, want superseded", v)
	}
}

// TestStep04_Share: share is idempotent and returns 400 on an unknown target,
// leaving visibility unchanged.
func TestStep04_Share(t *testing.T) {
	h, projectID, id := step04Server(t, "secret")
	q := "?project=" + projectID
	body := map[string]any{"visibility": "team"}

	// share team → 200; share team again → 200 (idempotent).
	for i := 0; i < 2; i++ {
		rr, _ := doHTTPAuth(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/share"+q, body, "secret")
		if rr.Code != stdhttp.StatusOK {
			t.Fatalf("share #%d: expected 200, got %d", i+1, rr.Code)
		}
	}
	// unknown target → 400, visibility unchanged (team).
	rr, _ := doHTTPAuth(t, h, stdhttp.MethodPost, "/memory/observations/"+itoa(id)+"/share"+q, map[string]any{"visibility": "bogus"}, "secret")
	if rr.Code != stdhttp.StatusBadRequest {
		t.Fatalf("share unknown target: expected 400, got %d", rr.Code)
	}
	rr, out := doHTTP(t, h, stdhttp.MethodGet, "/observations/"+itoa(id)+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("refetch after bad share: got %d", rr.Code)
	}
	if v, _ := out["visibility"].(string); v != "team" {
		t.Errorf("visibility = %q after 400, want unchanged team", v)
	}
}

// TestStep04_EditAppendsVersion: PATCHing content appends a 013 version —
// current content is the new value AND the prior content is recoverable via
// the governance version history.
func TestStep04_EditAppendsVersion(t *testing.T) {
	h, projectID, id := step04Server(t, "secret")
	q := "?project=" + projectID

	// In-place edit to a new body (write-gated).
	rr, _ := doHTTPAuth(t, h, stdhttp.MethodPatch, "/memory/observations/"+itoa(id)+q, map[string]any{"content": "v2"}, "secret")
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("edit: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	// Current content is now v2.
	rr, out := doHTTP(t, h, stdhttp.MethodGet, "/observations/"+itoa(id)+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("refetch after edit: got %d", rr.Code)
	}
	if c, _ := out["content"].(string); c != "v2" {
		t.Errorf("content after edit = %q, want v2", c)
	}
	// Governance exposes the prior content ("the full body") in the version history.
	rr, gov := doHTTP(t, h, stdhttp.MethodGet, "/memory/observations/"+itoa(id)+"/governance"+q, nil)
	if rr.Code != stdhttp.StatusOK {
		t.Fatalf("governance: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	versions, _ := gov["versions"].([]any)
	if len(versions) == 0 {
		t.Fatalf("governance versions empty, want prior content recoverable: %v", gov)
	}
	foundPrior := false
	for _, v := range versions {
		vm, _ := v.(map[string]any)
		if c, _ := vm["content"].(string); c == "the full body" {
			foundPrior = true
		}
	}
	if !foundPrior {
		t.Errorf("prior content \"the full body\" not recoverable in version history: %v", versions)
	}
}
