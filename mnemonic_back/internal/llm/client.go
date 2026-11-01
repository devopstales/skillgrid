// Package llm is the OpenAI-compatible HTTP chat Completer for the mnemonic
// second-brain seams (ADR-0023, TICKET-01). It is stdlib-only — no LLM SDK:
// one HTTP shape (POST {BaseURL}/chat/completions) covers Ollama (via its /v1
// root) and OpenAI-compatible cloud hosts.
//
// The Completer is deliberately thin and fail-open at the caller: Complete
// returns the error (it does NOT swallow it); the seam adapters (Task 2)
// decide the ADR-0016 floor fallback. This package never panics on a non-2xx
// or a network error and never hangs: a timeout is a normal error return.
// The api_key env fallback is NOT here — the config loader (Part B) resolves
// it; the client uses exactly the key it is handed.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultTimeout is the per-request budget when Config.Timeout is zero.
// ADR-0023 locks 60s: generous enough for a local model's first-token latency
// without wedging a boot path; the config loader leaves Timeout zero (it does
// not invent a default), so the client owns this floor.
const defaultTimeout = 60 * time.Second

// maxErrorBody is how much of a non-2xx response body to fold into the
// returned error message. Long bodies (stack traces, HTML) are truncated so a
// bad server cannot bloat the log.
const maxErrorBody = 1 << 10 // 1KB

// Completer is the single method the four mnemonic seams (ask, dedup,
// extraction, dream) need. It matches the existing service.AskLLM contract
// verbatim (Complete(ctx, system, user) (string, error)) so Task 2 can adapt
// a Client to each seam without a shape change.
type Completer interface {
	Complete(ctx context.Context, system, user string) (string, error)
}

// Config is the client's construction input. BaseURL is the OpenAI-compatible
// root (e.g. http://localhost:11434/v1 for Ollama or
// https://api.openai.com/v1); the full endpoint is
// strings.TrimRight(BaseURL, "/") + "/chat/completions". APIKey is the bearer
// token; empty means no Authorization header (Ollama local). Timeout is the
// per-request budget; <= 0 falls back to defaultTimeout (60s).
type Config struct {
	BaseURL string
	Model   string
	APIKey  string
	Timeout time.Duration
}

// Client is the concrete OpenAI-compatible chat Completer. It is safe for
// concurrent use: the http.Client is shared and stateless per request.
type Client struct {
	httpClient *http.Client
	baseURL    string
	model      string
	apiKey     string
}

// New builds a Client from cfg. A non-positive Timeout defaults to 60s. The
// BaseURL trailing slash is trimmed so "…/v1" and "…/v1/" both yield the same
// request path.
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		model:      cfg.Model,
		apiKey:     cfg.APIKey,
	}
}

// chatRequest is the OpenAI chat-completions request body.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

// chatMessage is one role/content entry in the messages array.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the OpenAI chat-completions response envelope. Only the
// fields the client needs are decoded.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete POSTs the system+user turn to {BaseURL}/chat/completions and
// returns choices[0].message.content. It never panics: a non-2xx response
// (folding up to 1KB of the body), a malformed body, or a network/timeout
// error all surface as a descriptive error return. The given ctx bounds the
// whole call in addition to the client's Timeout, so a caller can also cancel
// early.
func (c *Client) Complete(ctx context.Context, system, user string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("llm: encode request: %w", err)
	}
	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("llm: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network error or the client Timeout firing — both are normal error
		// returns (fail open at the caller), never a panic or hang.
		return "", fmt.Errorf("llm: post %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llm: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm: %s returned %s: %s", url, resp.Status, truncateErrorBody(body))
	}
	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("llm: decode response: %w", err)
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("llm: response had no choices[0].message.content")
	}
	return parsed.Choices[0].Message.Content, nil
}

// truncateErrorBody folds a non-2xx body into a single-line, bounded string
// for the error message. Newlines become spaces so the error stays one line
// in logs; anything past maxErrorBody is cut with a marker.
func truncateErrorBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxErrorBody {
		s = s[:maxErrorBody] + "…"
	}
	return s
}
