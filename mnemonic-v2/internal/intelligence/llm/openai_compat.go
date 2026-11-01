package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// defaultTimeout is the per-request budget when NewClient is handed a
// non-positive timeout. 60s: generous enough for a local model's first-token
// latency without wedging a boot path (port of v1 internal/llm/client.go:29).
const defaultTimeout = 60 * time.Second

// maxErrorBody bounds how much of a non-2xx body folds into the returned
// error, so a bad server cannot bloat the log (port of v1 client.go:34).
const maxErrorBody = 1 << 10 // 1KB

// Client is the OpenAI-compatible provider: one HTTP shape covering
// POST {base}/chat/completions (Classify / Judge / Compare / Extract) and
// POST {base}/embeddings (Embed). It implements both the LLM and Embedder
// seams. Ollama is the same client pointed at its /v1 root (spec
// 003-intelligence §5) — one adapter, two configs. Safe for concurrent use.
type Client struct {
	httpClient *http.Client
	baseURL    string
	model      string
	apiKey     string

	// dims caches the inferred embedding dimension (see Dimensions).
	dimMu sync.Mutex
	dims  int
}

// NewClient builds the OpenAI-compatible client. baseURL is the
// OpenAI-compatible root (e.g. http://localhost:11434/v1 for Ollama or
// https://api.openai.com/v1); the trailing slash is trimmed. An empty apiKey
// sends no Authorization header (Ollama local). A non-positive timeout
// defaults to 60s. An empty baseURL is an error (there is no request to make).
func NewClient(baseURL, model, apiKey string, timeout time.Duration) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("llm: base URL is empty")
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		apiKey:     apiKey,
	}, nil
}

// chatRequest is the OpenAI chat-completions request body.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

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

// complete POSTs the system+user turn to {base}/chat/completions and returns
// choices[0].message.content. It never panics: a non-2xx response (folding up
// to 1KB of the body), a malformed body, or a network/timeout error all
// surface as a descriptive error return (the seam is "failing" — the caller's
// degrade.Run retries once, then the floor). The given ctx bounds the whole
// call in addition to the client's Timeout, so a caller can also cancel early.
func (c *Client) complete(ctx context.Context, system, user string) (string, error) {
	payload, err := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
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
		// returns (the seam is failing), never a panic or hang.
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
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("llm: response had no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

// truncateErrorBody folds a non-2xx body into a single-line, bounded string
// for the error message (port of v1 client.go:159).
func truncateErrorBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxErrorBody {
		s = s[:maxErrorBody] + "…"
	}
	return s
}

// classifySystem is the system instruction for the 4-way dedup classification
// (port of v1 service/dedup_llm.go:121).
const classifySystem = "You classify new observations against existing candidates. Return only strict JSON."

// classifyPrompt is the 4-way strict-JSON contract (port of v1
// service/dedup_llm.go:48). The verdict is one of add/update/delete/noop and
// candidate is the 0-based index into the candidate list (or null).
const classifyPrompt = `You are classifying a new observation against existing candidates.
Return ONLY a JSON object: {"verdict": "add"|"update"|"delete"|"noop", "candidate": <0-based index>}.
- "add": the new observation is genuinely new.
- "update": it updates an existing candidate (set candidate to that index).
- "delete": it supersedes/deletes an existing candidate (set candidate to that index).
- "noop": it matches an existing candidate and changes nothing.
Set candidate to null for "add"/"noop". Do not invent candidates that are not listed.
New observation:
`

// classifyResponse is the LLM's strict 4-way verdict. Candidate is a *int so
// an explicit 0 (first candidate) is distinguished from an absent field (null
// = add / noop) (port of v1 service/dedup_llm.go:62).
type classifyResponse struct {
	Verdict   string `json:"verdict"`
	Candidate *int   `json:"candidate"`
}

// Classify implements LLM.Classify: 4-way classify newContent against the
// candidate contents. The LLM's 0-based candidate index maps to a 1-based
// observation id (index+1) for update/delete (v1 service/dedup_llm.go:98-100).
// The deterministic hash floor is the caller's fallback on error (via
// degrade.Run); a malformed or unrecognized verdict is an error, not a guess.
func (c *Client) Classify(ctx context.Context, newContent string, candidates []string) (DedupDecision, error) {
	user := classifyPrompt + newContent + candidateBlock(candidates)
	resp, err := c.complete(ctx, classifySystem, user)
	if err != nil {
		return DedupDecision{}, err
	}
	var out classifyResponse
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		return DedupDecision{}, fmt.Errorf("llm: parse classify response: %w", err)
	}
	verdict, ok := mapVerdict(strings.ToLower(strings.TrimSpace(out.Verdict)))
	if !ok {
		return DedupDecision{}, fmt.Errorf("llm: unknown verdict %q", out.Verdict)
	}
	d := DedupDecision{Verdict: verdict, Reason: "llm"}
	if (verdict == VerdictUpdate || verdict == VerdictDelete) &&
		out.Candidate != nil && *out.Candidate >= 0 && *out.Candidate < len(candidates) {
		d.CandidateID = int64(*out.Candidate + 1)
	}
	return d, nil
}

// candidateBlock renders the candidate contents as a numbered block the LLM
// can reference by 0-based index (port of v1 service/dedup_llm.go:125).
func candidateBlock(candidates []string) string {
	var b strings.Builder
	for i, cand := range candidates {
		fmt.Fprintf(&b, "[%d] %s\n", i, cand)
	}
	return b.String()
}

// mapVerdict maps the LLM's raw verdict string to the canonical DedupVerdict;
// ok=false on an unrecognized value (the caller falls back to the floor).
func mapVerdict(s string) (DedupVerdict, bool) {
	switch s {
	case "add":
		return VerdictAdd, true
	case "update":
		return VerdictUpdate, true
	case "delete":
		return VerdictDelete, true
	case "noop":
		return VerdictNoop, true
	default:
		return "", false
	}
}

// judgeSystem / judgePrompt are the strict-JSON contract for Judge and
// Compare: the seam returns the relation it judged, a confidence in [0,1], and
// a one-line evidence quote. (spec 003-intelligence §2.1 Judge/Compare.)
const judgeSystem = "You judge semantic relations between observations. Return only strict JSON."

const judgePrompt = `You are judging the semantic relation between a source and a target observation.
Return ONLY a JSON object: {"relation": "<the relation you judged>", "confidence": <0.0-1.0>, "evidence": "<one line of evidence>"}.
Source:
%s
Target:
%s
Pending relation to confirm or refute: %s
Reason: %s
`

// judgmentResponse is the LLM's strict judgment.
type judgmentResponse struct {
	Relation   string  `json:"relation"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}

// Judge implements LLM.Judge: a semantic judgment on the pending relation
// between source and target.
func (c *Client) Judge(ctx context.Context, source, target Observation, rel Relation) (Judgment, error) {
	user := fmt.Sprintf(judgePrompt, describeObservation(source), describeObservation(target), rel.Relation, rel.Reason)
	resp, err := c.complete(ctx, judgeSystem, user)
	if err != nil {
		return Judgment{}, err
	}
	var out judgmentResponse
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		return Judgment{}, fmt.Errorf("llm: parse judge response: %w", err)
	}
	return Judgment{Relation: out.Relation, Confidence: out.Confidence, Evidence: out.Evidence, Model: c.model}, nil
}

// Compare implements LLM.Compare: a proactive semantic comparison of two
// observations → a judged relation.
func (c *Client) Compare(ctx context.Context, a, b Observation) (Judgment, error) {
	return c.Judge(ctx, a, b, Relation{})
}

func describeObservation(o Observation) string {
	return fmt.Sprintf("id=%d type=%s title=%s\n%s", o.ID, o.Type, o.Title, o.Content)
}

// extractSystem / extractPrompt are the strict-JSON contract for Extract:
// one entry per learning, verbatim for structured items (port of v1
// internal/memory/extraction.go:34).
const extractSystem = "You extract structured learnings from session text. Return only strict JSON."

const extractPrompt = `You are extracting structured learnings from a block of session text.
Return ONLY a JSON object: {"learnings": [{"text": "<one learning>", "type": "<decision|bugfix|architecture|pattern|config|preference|discovery|learning|lesson>"}]}.
One entry per learning. If a learning is written in a structured section, quote its item VERBATIM. For a nuanced learning in free-form prose, consolidate the relevant sentences into a single complete sentence. Do not invent content that is not present in the text.
Text:
`

// extractResponse is the LLM's structured extraction.
type extractResponse struct {
	Learnings []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"learnings"`
}

// Extract implements LLM.Extract: pull discrete candidate observations from
// free text. Blank items are dropped; an empty parse is an error (the LLM
// found nothing → let the regex floor decide, mirroring v1 extraction.go).
func (c *Client) Extract(ctx context.Context, text string) ([]CandidateObservation, error) {
	resp, err := c.complete(ctx, extractSystem, extractPrompt+text)
	if err != nil {
		return nil, err
	}
	var out extractResponse
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		return nil, fmt.Errorf("llm: parse extract response: %w", err)
	}
	items := make([]CandidateObservation, 0, len(out.Learnings))
	for _, l := range out.Learnings {
		body := strings.TrimSpace(l.Text)
		if body == "" {
			continue
		}
		items = append(items, CandidateObservation{Type: strings.TrimSpace(l.Type), Title: body, Content: body})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("llm: extraction returned no learnings")
	}
	return items, nil
}

// Embed implements Embedder.Embed: POST {base}/embeddings with
// {"model","input":[...]} → data[].embedding (the OpenAI embeddings shape).
// The result order matches the input order; a count mismatch is an error,
// never a silent truncation (the vector leg must line up with the texts).
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	payload, err := json.Marshal(struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}{Model: c.model, Input: texts})
	if err != nil {
		return nil, fmt.Errorf("embed: encode request: %w", err)
	}
	url := c.baseURL + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("embed: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed: post %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("embed: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embed: %s returned %s: %s", url, resp.Status, truncateErrorBody(body))
	}
	var parsed struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("embed: decode response: %w", err)
	}
	if len(parsed.Data) != len(texts) {
		return nil, fmt.Errorf("embed: server returned %d embeddings for %d texts", len(parsed.Data), len(texts))
	}
	out := make([][]float32, len(parsed.Data))
	for i, d := range parsed.Data {
		out[i] = d.Embedding
	}
	// Cache the inferred dimension and reject a vector-length mismatch with
	// what was already cached (mismatch → error, never silent truncation).
	c.dimMu.Lock()
	for _, v := range out {
		if len(v) == 0 {
			continue
		}
		if c.dims == 0 {
			c.dims = len(v)
		} else if c.dims != len(v) {
			c.dimMu.Unlock()
			return nil, fmt.Errorf("embed: inconsistent dimensions (%d vs %d)", c.dims, len(v))
		}
	}
	c.dimMu.Unlock()
	return out, nil
}

// Model implements Embedder.Model (recorded on observations.embedding_model,
// INV-1).
func (c *Client) Model() string { return c.model }

// Dimensions implements Embedder.Dimensions. The OpenAI-compatible endpoint
// does not advertise its embedding dimension, so the client infers it from the
// first round-trip and caches it; a later call whose vector length disagrees
// is an error (mismatch → error at first write, never silent truncation).
func (c *Client) Dimensions() int {
	c.dimMu.Lock()
	defer c.dimMu.Unlock()
	if c.dims == 0 {
		// No round-trip yet: infer from the model-name suffix (the convention
		// for embedding models, e.g. "nomic-embed-text" → unknown → 0 until a
		// real Embed call fixes it). A 0 here means "not yet known"; the first
		// Embed call pins it and a later mismatch errors (never truncates).
		_ = c.model
	}
	return c.dims
}
