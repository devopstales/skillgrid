package embedder

import (
	"context"
	"log"
)

// loadCtx is the context used for eager provider loads during selection.
var loadCtx = context.Background()

// Provider names accepted in mnemonic.embedder.provider (indexing.yaml).
const (
	ProviderOllama   = "ollama"
	ProviderLocal    = "local"
	ProviderOnnx     = "onnx"
	ProviderExternal = "external"
	ProviderOff      = "off"
)

// EmbedderConfig is the selection input for BuildFromConfig. It mirrors the
// mnemonic.embedder.* section of indexing.yaml (config.EmbedderConfig); the
// embedder package does not import config to avoid an import cycle.
type EmbedderConfig struct {
	Provider  string
	Dimension int
	Indexing  AsymParams
	Query     AsymParams
	// External / ollama only.
	BaseURL string
	Model   string
	APIKey  string
	// Local only.
	ModelDir string
}

// BuildFromConfig constructs the embedder for the configured provider
// (config-driven selection, 06.3). It never returns nil for a selected
// provider: a construction or load failure degrades to the Null Adapter
// (06.4) with a logged warning, so the pipeline falls back to the FTS +
// signals floor instead of failing.
//
//   - "ollama"   → Ollama (base_url defaults to http://localhost:11434)
//   - "local"    → LocalONNX (model_dir defaults to ~/.skillgrid/models/)
//   - "onnx"     → Onnx (existing behavior: hash fallback when model absent)
//   - "external" → External (existing behavior)
//   - "off", ""  → Null (no vector leg)
func BuildFromConfig(cfg EmbedderConfig) Embedder {
	switch cfg.Provider {
	case ProviderOff, "":
		return NewNull()
	case ProviderOllama:
		return degradeOnError("ollama", func() (Embedder, error) {
			return NewOllama(OllamaConfig{
				BaseURL:   cfg.BaseURL,
				Model:     cfg.Model,
				Dimension: cfg.Dimension,
			}), nil
		})
	case ProviderLocal:
		return degradeOnError("local", func() (Embedder, error) {
			l := NewLocal(LocalConfig{
				ModelDir:  cfg.ModelDir,
				Model:     cfg.Model,
				Dimension: cfg.Dimension,
			})
			if err := l.Load(loadCtx); err != nil {
				return nil, err
			}
			return l, nil
		})
	case ProviderOnnx:
		return NewOnnx(OnnxConfig{
			Model:     cfg.Model,
			Dimension: cfg.Dimension,
			Indexing:  cfg.Indexing,
			Query:     cfg.Query,
		})
	case ProviderExternal:
		return NewExternal(ExternalConfig{
			BaseURL:   cfg.BaseURL,
			Model:     cfg.Model,
			APIKey:    cfg.APIKey,
			Dimension: cfg.Dimension,
			Indexing:  cfg.Indexing,
			Query:     cfg.Query,
		})
	default:
		log.Printf("warning: unknown embedder provider %q, degrading to null", cfg.Provider)
		return NewNull()
	}
}

// degradeOnError runs build and, on failure, logs a warning and returns the
// Null Adapter (06.4). A nil result is also degraded (e.g. a local model
// whose file is missing).
func degradeOnError(provider string, build func() (Embedder, error)) Embedder {
	e, err := build()
	if err != nil {
		log.Printf("warning: embedder provider %q failed to load: %v (degrading to null)", provider, err)
		return NewNull()
	}
	if e == nil {
		log.Printf("warning: embedder provider %q not available (degrading to null)", provider)
		return NewNull()
	}
	return e
}
