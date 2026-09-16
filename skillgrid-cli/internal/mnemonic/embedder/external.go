package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// ExternalConfig configures the OpenAI-compatible external embedder.
type ExternalConfig struct {
	BaseURL   string // e.g. http://localhost:11434 (Ollama) or https://api.openai.com/v1
	Model     string
	APIKey    string
	Dimension int
	Indexing  AsymParams
	Query     AsymParams
}

// AsymParams is one side (corpus or query) of the asymmetric external embedder.
type AsymParams struct {
	Instructions string
	InputType    string
	MaxTokens    int
}

// External is the OpenAI-compatible /embeddings HTTP embedder (Ollama,
// OpenAI, etc.). It is asymmetric-capable: the indexing (corpus) side and the
// query side are sent with their own param sets.
type External struct {
	cfg  ExternalConfig
	http *http.Client
}

// NewExternal returns an External embedder.
func NewExternal(cfg ExternalConfig) *External {
	return &External{
		cfg:  cfg,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *External) Model() string {
	if e.cfg.Model != "" {
		return e.cfg.Model
	}
	return "external"
}

func (e *External) Dimension() int { return e.cfg.Dimension }

func (e *External) Embed(ctx context.Context, text string) (memory.Vector, error) {
	return e.embed(ctx, text, e.cfg.Indexing, false)
}

func (e *External) EmbedQuery(ctx context.Context, text string) (memory.Vector, error) {
	return e.embed(ctx, text, e.cfg.Query, true)
}

type externalRequest struct {
	Model        string `json:"model"`
	Input        string `json:"input"`
	InputType    string `json:"input_type,omitempty"`
	Instructions string `json:"instructions,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
}

type externalResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (e *External) embed(ctx context.Context, text string, p AsymParams, isQuery bool) (memory.Vector, error) {
	if e.cfg.BaseURL == "" {
		return memory.Vector{}, fmt.Errorf("external embedder: base_url is required")
	}
	// The output dimension is model-wide; the side-specific params only
	// change how the input is treated (instructions / input_type).
	reqBody := externalRequest{
		Model:     e.cfg.Model,
		Input:     text,
		InputType: p.InputType,
	}
	if !isQuery {
		reqBody.Instructions = p.Instructions
	} else {
		reqBody.Instructions = p.Instructions
	}
	if p.MaxTokens > 0 {
		reqBody.MaxTokens = p.MaxTokens
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return memory.Vector{}, err
	}

	// Ollama uses /api/embed; OpenAI uses /embeddings. Try /embeddings first,
	// fall back to /api/embed (Ollama).
	endpoints := []string{
		joinURL(e.cfg.BaseURL, "/embeddings"),
		joinURL(e.cfg.BaseURL, "/api/embed"),
	}
	var lastErr error
	for _, ep := range endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep, bytes.NewReader(body))
		if err != nil {
			return memory.Vector{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		if e.cfg.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+e.cfg.APIKey)
		}
		resp, err := e.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("external embedder: %s -> HTTP %d: %s", ep, resp.StatusCode, string(respBody))
			continue
		}
		var out externalResponse
		if err := json.Unmarshal(respBody, &out); err != nil {
			// Ollama /api/embed returns a flat {"embedding": [...]}.
			var flat struct {
				Embedding []float32 `json:"embedding"`
			}
			if err2 := json.Unmarshal(respBody, &flat); err2 == nil && len(flat.Embedding) > 0 {
				return memory.Vector{Data: flat.Embedding}, nil
			}
			return memory.Vector{}, fmt.Errorf("external embedder: decode: %w", err)
		}
		if len(out.Data) == 0 || len(out.Data[0].Embedding) == 0 {
			lastErr = fmt.Errorf("external embedder: empty embedding from %s", ep)
			continue
		}
		return memory.Vector{Data: out.Data[0].Embedding}, nil
	}
	return memory.Vector{}, fmt.Errorf("external embedder: %w", lastErr)
}

func joinURL(base, path string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base + path
	}
	u.Path = u.Path + path
	return u.String()
}
