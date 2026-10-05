package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestCompleteSuccess is TICKET-01 (SATISFIES "happy path openai-compatible
// complete succeeds"): a valid OpenAI chat-completions response yields the
// message content, and the request carries the Bearer header (when a key is
// set), the correct model, and the system+user messages in the right order.
func TestCompleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The client trims BaseURL and appends /chat/completions; the test
		// server's root serves that path.
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("Authorization = %q, want Bearer sk-test", got)
		}
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "qwen3:8b" {
			t.Errorf("model = %q, want qwen3:8b", req.Model)
		}
		if len(req.Messages) != 2 ||
			req.Messages[0].Role != "system" || req.Messages[0].Content != "sys" ||
			req.Messages[1].Role != "user" || req.Messages[1].Content != "user" {
			t.Errorf("messages = %+v, want [system:sys user:user]", req.Messages)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "hello"}}},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Model: "qwen3:8b", APIKey: "sk-test", Timeout: time.Second})
	got, err := c.Complete(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("Complete = %q, want hello", got)
	}
}

// TestCompleteNoKeyOmitsAuthHeader: with an empty APIKey (the Ollama local
// case), the Authorization header is absent — local servers don't need it and
// a stray empty "Bearer " header would confuse some.
func TestCompleteNoKeyOmitsAuthHeader(t *testing.T) {
	sawAuth := ""
	saw := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		saw = true
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "ok"}}},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Model: "m", Timeout: time.Second})
	if _, err := c.Complete(context.Background(), "s", "u"); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if !saw {
		t.Fatal("server was not hit")
	}
	if sawAuth != "" {
		t.Fatalf("Authorization = %q, want empty (no APIKey)", sawAuth)
	}
}

// TestCompleteNon2xx is TICKET-01 (network boundary): a non-2xx response
// yields a non-nil error that names the status code, and the server's error
// body is folded into the message (up to 1KB).
func TestCompleteNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "boom: internal error")
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Model: "m", Timeout: time.Second})
	_, err := c.Complete(context.Background(), "s", "u")
	if err == nil {
		t.Fatal("want error for HTTP 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("error = %q, want it to contain the status code 500", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %q, want it to fold in the server body", err)
	}
}

// TestCompleteTrailingSlashInBaseURL: a BaseURL with a trailing slash must not
// produce a double slash in the request path.
func TestCompleteTrailingSlashInBaseURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions (no double slash)", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "ok"}}},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL + "/", Model: "m", Timeout: time.Second})
	if _, err := c.Complete(context.Background(), "s", "u"); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
}

// TestCompleteTimeout: a server that sleeps past the client's Timeout yields a
// context-deadline error and does not hang. The 50ms client budget is well
// below the handler's 5s sleep, so a correct implementation returns in ~50ms.
// The handler sleeps unconditionally: http.Client.Timeout aborts the
// connection but does not cancel the server-side request context, so the
// handler keeps sleeping and the deferred srv.Close() below blocks ~5s — that
// is server teardown, NOT a client hang. The elapsed-time assertion (timed
// around Complete alone, below) is what proves the client did not hang.
func TestCompleteTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "late"}}},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Model: "m", Timeout: 50 * time.Millisecond})
	start := time.Now()
	_, err := c.Complete(context.Background(), "s", "u")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("want a timeout error, got nil")
	}
	// http.Client.Timeout fires with "context deadline exceeded". Asserting the
	// exact substring makes the test precise (a 200 with no body, or a
	// non-timeout network error, would not match) instead of substring luck.
	if !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("error = %q, want a context-deadline (timeout) error", err)
	}
	// No hang: Complete must return well before the handler's 5s sleep.
	if elapsed > time.Second {
		t.Fatalf("Complete took %v, want < 1s (no hang)", elapsed)
	}
}
