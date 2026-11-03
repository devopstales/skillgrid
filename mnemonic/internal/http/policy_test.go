package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const testPolicy = `
policy:
  enabled: true
  rules:
    - name: no-secret-writes
      match: { action: file_write, path: "secrets/**" }
      effect: block
      message: secrets are read-only
    - name: careful-rm
      match: { action: command_exec, command: "*rm -rf *" }
      effect: warn
      message: careful
    - name: prefer-mem
      match: { tool: WebSearch }
      effect: guide
      message: try mem_search
`

func writePolicy(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "policy.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func evalPolicy(t *testing.T, s *Server, body map[string]any) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/policy/evaluate?project="+proj, bytes.NewReader(raw))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("evaluate %v: %d %s", body, w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out
}

func TestPolicyEvaluate_EffectsPersistAndCount(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s, _ := newToolCallsServer(t)
	repo := t.TempDir()
	writePolicy(t, repo, testPolicy)

	base := map[string]any{"session_id": "cur-1", "agent": "cursor", "directory": repo}
	with := func(kv ...any) map[string]any {
		m := map[string]any{}
		for k, v := range base {
			m[k] = v
		}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	}

	cases := []struct {
		body   map[string]any
		effect string
		rule   string
	}{
		{with("tool", "Write", "path", filepath.Join(repo, "secrets", "db.env")), "block", "no-secret-writes"},
		{with("action", "command_exec", "tool", "Shell", "command", "rm -rf build"), "warn", "careful-rm"},
		{with("tool", "WebSearch"), "guide", "prefer-mem"},
		{with("tool", "Read", "path", "secrets/db.env"), "allow", ""},
	}
	for _, c := range cases {
		got := evalPolicy(t, s, c.body)
		if got["effect"] != c.effect || got["rule"] != c.rule {
			t.Errorf("%v: got %v, want %s/%s", c.body, got, c.effect, c.rule)
		}
		if (c.effect != "allow") != (got["recorded"] == true) {
			t.Errorf("%v: recorded=%v", c.body, got["recorded"])
		}
	}

	evs := eventsOf(getToolJSON(t, s, "/sessions/cur-1/events?project="+proj))
	results := map[string]bool{}
	for _, e := range evs {
		results[e["result"].(string)] = true
		if e["result"] == "blocked" && e["preview"] != "no-secret-writes: secrets are read-only" {
			t.Errorf("blocked preview = %q", e["preview"])
		}
	}
	for _, want := range []string{"blocked", "warned", "guided"} {
		if !results[want] {
			t.Errorf("timeline missing a %s decision: %v", want, evs)
		}
	}
	raw, _ := getToolJSON(t, s, "/mnemonic/sessions?project="+proj)["sessions"].([]any)
	for _, r := range raw {
		row := r.(map[string]any)
		if row["id"] != "cur-1" {
			continue
		}
		if row["blocked_actions"] != float64(1) || row["policy_decisions"] != float64(3) || row["tool_calls"] != float64(0) {
			t.Errorf("session row blocked=%v policy=%v calls=%v, want 1/3/0",
				row["blocked_actions"], row["policy_decisions"], row["tool_calls"])
		}
	}
	st := getToolJSON(t, s, "/events/stats?project="+proj)
	if st["blocked"] != float64(1) || st["warned"] != float64(1) || st["guided"] != float64(1) || st["total"] != float64(0) {
		t.Errorf("stats blocked/warned/guided/total = %v/%v/%v/%v, want 1/1/1/0",
			st["blocked"], st["warned"], st["guided"], st["total"])
	}
}

func TestPolicyEvaluate_DisabledAndBrokenFailOpen(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s, _ := newToolCallsServer(t)

	off := t.TempDir()
	writePolicy(t, off, "policy:\n  enabled: false\n  rules:\n    - match: {action: file_write}\n      effect: block\n      message: x\n")
	if got := evalPolicy(t, s, map[string]any{"session_id": "s1", "tool": "Write", "path": "a", "directory": off}); got["effect"] != "allow" || got["recorded"] == true {
		t.Errorf("disabled = %v", got)
	}

	broken := t.TempDir()
	writePolicy(t, broken, "policy:\n  enabled: true\n  rules:\n    - effect: nope\n")
	got := evalPolicy(t, s, map[string]any{"session_id": "s1", "tool": "Write", "path": "a", "directory": broken})
	if got["effect"] != "allow" || got["error"] == nil {
		t.Errorf("broken file must fail open with an error: %v", got)
	}

	view := getToolJSON(t, s, "/policy?project="+proj+"&directory="+off)
	if view["enabled"] != false {
		t.Errorf("GET /policy = %v", view)
	}
	rules, _ := view["rules"].([]any)
	if len(rules) != 1 {
		t.Errorf("GET /policy rules = %v", rules)
	}
}
