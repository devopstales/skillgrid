package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// DefaultOllamaBaseURL is the stock Ollama server endpoint.
const DefaultOllamaBaseURL = "http://localhost:11434"

// DefaultOllamaModel is the default Ollama embedding model.
const DefaultOllamaModel = "nomic-embed-code"

// OllamaConfig configures the Ollama embedder (provider "ollama"). The base
// URL defaults to the stock Ollama port (11434) and is configurable so tests
// can point at a mock server.
type OllamaConfig struct {
	BaseURL   string
	Model     string
	Dimension int
}

// Ollama is the Ollama /api/embeddings HTTP embedder. Each call POSTs
// {"model","prompt"} to <BaseURL>/api/embeddings and returns the flat
// {"embedding":[...]} vector Ollama produces.
type Ollama struct {
	cfg  OllamaConfig
	http *http.Client
}

// NewOllama returns an Ollama embedder with the given config.
func NewOllama(cfg OllamaConfig) *Ollama {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultOllamaBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = DefaultOllamaModel
	}
	return &Ollama{
		cfg:  cfg,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func (o *Ollama) Model() string { return o.cfg.Model }

func (o *Ollama) Dimension() int { return o.cfg.Dimension }

func (o *Ollama) Embed(ctx context.Context, text string) (memory.Vector, error) {
	return o.embed(ctx, text)
}

func (o *Ollama) EmbedQuery(ctx context.Context, text string) (memory.Vector, error) {
	return o.embed(ctx, text)
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaResponse struct {
	Model     string    `json:"model"`
	Embedding []float32 `json:"embedding"`
}

func (o *Ollama) embed(ctx context.Context, text string) (memory.Vector, error) {
	body, err := json.Marshal(ollamaRequest{Model: o.cfg.Model, Prompt: text})
	if err != nil {
		return memory.Vector{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(o.cfg.BaseURL, "/api/embeddings"), bytes.NewReader(body))
	if err != nil {
		return memory.Vector{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.http.Do(req)
	if err != nil {
		return memory.Vector{}, fmt.Errorf("ollama embedder: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return memory.Vector{}, fmt.Errorf("ollama embedder: %s/api/embeddings -> HTTP %d: %s", o.cfg.BaseURL, resp.StatusCode, string(raw))
	}
	var out ollamaResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return memory.Vector{}, fmt.Errorf("ollama embedder: decode: %w", err)
	}
	if len(out.Embedding) == 0 {
		return memory.Vector{}, fmt.Errorf("ollama embedder: empty embedding from %s", o.cfg.BaseURL)
	}
	return memory.Vector{Data: out.Embedding}, nil
}
