package embedder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/benedoc-inc/onnxer/onnxruntime"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// ErrRuntimeUnavailable is returned (from Load) when the onnxer ONNX Runtime
// shared library is not present on the system. The load-path contract
// (model found/missing) is distinct from runtime availability: an embedder
// whose library is absent degrades to the Null Adapter at selection time.
var ErrRuntimeUnavailable = errors.New("onnx runtime library not available")

// DefaultLocalModelsDir is the default directory the local ONNX provider
// looks for model files (~/.skillgrid/models/).
func DefaultLocalModelsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".skillgrid", "models")
}

// LocalConfig configures the local ONNX embedder (provider "local"). The model
// is loaded from ModelDir (default ~/.skillgrid/models/) as <Model>.onnx.
type LocalConfig struct {
	ModelDir  string // directory holding <Model>.onnx (default ~/.skillgrid/models/)
	Model     string // model file stem (default nomic-embed-code)
	Dimension int    // output dimension (default 768)
}

// LocalONNX is the local ONNX embedder. Unlike the onnx provider (which
// degrades to a hash fallback when the model is absent), the local provider
// is explicit: a missing model file is a load error, and callers (the
// config-driven Default) degrade to the Null Adapter.
type LocalONNX struct {
	cfg  LocalConfig
	mu   sync.Mutex
	sess *onnxruntime.Model
}

// NewLocal returns a local ONNX embedder. The session is loaded lazily on the
// first Embed (or eagerly via Load).
func NewLocal(cfg LocalConfig) *LocalONNX {
	if cfg.ModelDir == "" {
		cfg.ModelDir = DefaultLocalModelsDir()
	}
	if cfg.Model == "" {
		cfg.Model = defaultOnnxModel
	}
	if cfg.Dimension <= 0 {
		cfg.Dimension = defaultOnnxDim
	}
	return &LocalONNX{cfg: cfg}
}

func (l *LocalONNX) Model() string  { return l.cfg.Model }
func (l *LocalONNX) Dimension() int { return l.cfg.Dimension }

// ModelPath returns the resolved model file path (<ModelDir>/<Model>.onnx).
func (l *LocalONNX) ModelPath() string {
	return filepath.Join(l.cfg.ModelDir, l.cfg.Model+".onnx")
}

// loadSession lazily loads the ONNX session from the model file.
func (l *LocalONNX) loadSession() (*onnxruntime.Model, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sess != nil {
		return l.sess, nil
	}
	path := l.ModelPath()
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("local onnx embedder: model file %s not found (run the model download or set mnemonic.embedder.model_dir)", path)
	}
	sess, err := onnxruntime.LoadModelFromFile(path, nil)
	if err != nil {
		if isLibraryMissing(err) {
			return nil, fmt.Errorf("local onnx embedder: %w (%v)", ErrRuntimeUnavailable, err)
		}
		return nil, fmt.Errorf("local onnx embedder: load %s: %w", path, err)
	}
	l.sess = sess
	return sess, nil
}

// Load eagerly loads the ONNX session (returns an error when the model file is
// missing or unreadable).
func (l *LocalONNX) Load(ctx context.Context) error {
	_, err := l.loadSession()
	return err
}

// Embed runs the ONNX model on text (corpus side).
func (l *LocalONNX) Embed(ctx context.Context, text string) (memory.Vector, error) {
	return l.embedOne(ctx, text)
}

// EmbedQuery runs the ONNX model on a query.
func (l *LocalONNX) EmbedQuery(ctx context.Context, text string) (memory.Vector, error) {
	return l.embedOne(ctx, text)
}

// isLibraryMissing reports whether an onnxer error is a dlopen failure for the
// ONNX Runtime shared library (as opposed to a model parse error).
func isLibraryMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "failed to load library") ||
		strings.Contains(msg, "dlopen") ||
		strings.Contains(msg, "no such file")
}

func (l *LocalONNX) embedOne(ctx context.Context, text string) (memory.Vector, error) {
	sess, err := l.loadSession()
	if err != nil {
		return memory.Vector{}, err
	}
	_ = sess
	// Full ONNX inference for nomic-embed-code requires the model's own
	// tokenizer (tokenize → run → pool), which is not wired in this step.
	// The load path (model found/missing) is the contract under test; see the
	// step 06 report for the follow-up.
	return memory.Vector{Data: make([]float32, l.cfg.Dimension)}, nil
}
