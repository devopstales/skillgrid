package install

// Provider setup (ADR-0023, TICKET-03). Install runs this as a normal step so
// the LLM/embedder host a machine needs for code indexing is wired naturally
// on first install and re-ensured (idempotent update) on re-run — not an
// opt-in bolt-on. Two hosts:
//
//   - "local" (default, Ollama): ensure the binary + service, pull the chat
//     (llama3.2:3b) and embed (nomic-embed-text) models only if absent, then
//     merge home mnemonic.llm + mnemonic.embedder (provider: ollama).
//   - "external" (OpenAI-compatible): no Ollama at all; merge home mnemonic.llm
//     + mnemonic.embedder (provider: external) against the operator's host.
//
// The step is NON-FATAL: every failure warns with a manual hint and returns,
// so a flaky install/serve never aborts the rest of the install (mirrors
// installSecurityTools). The home config merge only sets/overrides the keys it
// owns (llm.{enabled,base_url,model}; embedder.{provider,base_url,model}) and
// preserves every unrelated key already present in the operator's file.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// goos isolates runtime.GOOS so tests can force a platform (e.g. "linux").
var goos = runtime.GOOS

// Local provider model names.
const (
	localLLMModel   = "llama3.2:3b"
	localEmbedModel = "nomic-embed-text"
)

// External provider model defaults (overridable via env).
const (
	defaultExternalLLMModel   = "gpt-4o-mini"
	defaultExternalEmbedModel = "text-embedding-3-small"
)

// ollamaBaseURL is the Ollama API root; /api/tags is probed and the /v1 OpenAI
// root is written to the home config. It is a package var so tests can point
// the probe at an httptest server (restored via t.Cleanup).
var ollamaBaseURL = "http://localhost:11434"

// runCmd executes a command and captures its output, like the existing run
// helper but decoupled from *Config (the provider step passes only what a
// command needs). It is a var so tests record invocations instead of exec.
var runCmd = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		if buf.Len() > 0 {
			Out("      " + strings.TrimSpace(buf.String()))
		}
		return err
	}
	return nil
}

// providerLabel is the human-facing name of the selected provider.
func providerLabel(c *Config) string {
	if c.Provider == "external" {
		return "external"
	}
	return "local (Ollama)"
}

// setupProvider wires the LLM/embedder provider into the home config. It never
// escalates a hard error: failures warn with a manual hint and return, so the
// caller (Run) logs "warning:" and the install continues.
func setupProvider(c *Config) error {
	// G11: --skip-provider means no provider work at all (no binary, serve,
	// pull, or home merge) — checked before any host-specific path runs.
	if c.SkipProvider {
		return nil
	}
	if c.Provider == "external" {
		return setupProviderExternal(c)
	}
	return setupProviderLocal(c)
}

// --- local (Ollama) ---

// setupProviderLocal ensures Ollama (binary, service, models) and merges the
// home llm + ollama embedder. Each sub-step failure is a warn-and-return.
func setupProviderLocal(c *Config) error {
	if err := ensureOllamaBinary(c); err != nil {
		Out("      warn ollama binary:", err, "— manual hint: install Ollama, then re-run: skillgrid install")
		return nil
	}
	base := ollamaBaseURL
	if !ollamaServing(c, base) {
		if err := startOllamaService(c); err != nil {
			Out("      warn ollama service:", err, "— manual hint: start ollama: `ollama serve`")
			return nil
		}
		if !ollamaServing(c, base) {
			Out("      warn ollama not responding after start — manual hint: start ollama: `ollama serve`")
			// Do not pull/config if the service is down.
			return nil
		}
	}
	pullMissingModels(c, base)
	return mergeHomeProviderConfig(c, localLLMProvider{
		enabled: true,
		baseURL: ollamaBaseURL + "/v1",
		model:   localLLMModel,
		embed:   embedOverride{provider: "ollama", baseURL: ollamaBaseURL + "/v1", model: localEmbedModel},
	})
}

// ensureOllamaBinary returns nil when `ollama` is on PATH, otherwise attempts a
// best-effort OS install (darwin: brew; linux: the install script via curl).
// A failed install is returned so the caller warns.
func ensureOllamaBinary(c *Config) error {
	if _, err := exec.LookPath("ollama"); err == nil {
		return nil
	}
	switch goos {
	case "darwin":
		Out("      installing ollama (brew)")
		if c.DryRun {
			Out("      [dry-run] brew install ollama")
			return nil
		}
		return runCmd("brew", "install", "ollama")
	case "linux":
		Out("      installing ollama (official script)")
		if c.DryRun {
			Out("      [dry-run] curl -fsSL https://ollama.com/install.sh | sh")
			return nil
		}
		if _, err := exec.LookPath("curl"); err != nil {
			return fmt.Errorf("ollama not installed and curl not on PATH — install Ollama manually (https://ollama.com/download/linux)")
		}
		return runCmd("sh", "-c", "curl -fsSL https://ollama.com/install.sh | sh")
	default:
		return fmt.Errorf("ollama not installed — install it manually for %s (https://ollama.com/download)", goos)
	}
}

// ollamaServing probes GET {base}/api/tags with a short timeout.
func ollamaServing(c *Config, base string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(base + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// startOllamaService backgrounds `ollama serve`, detached in its own process
// group, with stderr to a log under the repo home.
func startOllamaService(c *Config) error {
	if c.DryRun {
		Out("      [dry-run] ollama serve (background)")
		return nil
	}
	logPath := filepath.Join(c.RepoHome, "logs", "ollama-serve.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	cmd := exec.Command("ollama", "serve")
	cmd.Stderr = f
	cmd.Stdout = f
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		f.Close()
		return err
	}
	// Detach: re-parent to init and forget the wait channel.
	_ = cmd.Process.Release()
	f.Close()
	Out("      ollama serve (background, log:", logPath, ")")
	return nil
}

// pullMissingModels lists installed models and pulls only the ones absent.
// Pull failures warn (non-fatal) — the config merge still runs.
func pullMissingModels(c *Config, base string) {
	present := ollamaModels(base)
	for _, name := range []string{localLLMModel, localEmbedModel} {
		if present[name] {
			Out("      skip pull", name, "(already present)")
			continue
		}
		if c.DryRun {
			Out("      [dry-run] ollama pull", name)
			continue
		}
		Out("      ollama pull", name)
		if err := runCmd("ollama", "pull", name); err != nil {
			Out("      warn pull", name, ":", err)
		}
	}
}

// ollamaModels GETs {base}/api/tags and parses the model name list.
func ollamaModels(base string) map[string]bool {
	present := map[string]bool{}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(base + "/api/tags")
	if err != nil {
		return present
	}
	defer resp.Body.Close()
	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return present
	}
	for _, m := range body.Models {
		if m.Name != "" {
			present[m.Name] = true
		}
	}
	return present
}

// --- external (OpenAI-compatible) ---

// setupProviderExternal wires the home llm + external embedder against the
// operator's host. No Ollama is installed, started, or pulled. base_url comes
// from the flag/env; an empty one warns (non-fatal) and returns.
func setupProviderExternal(c *Config) error {
	base := c.LLMBaseURL
	if base == "" {
		base = os.Getenv("SKILLGRID_LLM_BASE_URL")
	}
	if base == "" {
		Out("      warn external provider has no base_url — set --base-url or SKILLGRID_LLM_BASE_URL (ignored api_key if base is empty)")
		return nil
	}
	// The api key is intentionally NOT written to the home yaml: the loader
	// owns the SKILLGRID_LLM_API_KEY env fallback (see mergeLLM, ADR-0023).
	llmModel := os.Getenv("SKILLGRID_LLM_MODEL")
	if llmModel == "" {
		llmModel = defaultExternalLLMModel
	}
	embedModel := os.Getenv("SKILLGRID_EMBED_MODEL")
	if embedModel == "" {
		embedModel = defaultExternalEmbedModel
	}
	// The api key is intentionally NOT persisted to the home yaml (the loader
	// owns the SKILLGRID_LLM_API_KEY env fallback, ADR-0023). When the operator
	// passed --api-key without setting that env var, say so — otherwise the
	// flag looks like it did something when the runtime will read an empty env.
	if c.LLMApiKey != "" && os.Getenv("SKILLGRID_LLM_API_KEY") == "" {
		Out("      note: --api-key is not persisted to the home config; set SKILLGRID_LLM_API_KEY in the environment for the runtime to use it")
	}
	return mergeHomeProviderConfig(c, externalLLMProvider{
		enabled: true,
		baseURL: base,
		model:   llmModel,
		embed:   embedOverride{provider: "external", baseURL: base, model: embedModel},
	})
}

// --- home config merge ---

// providerWire is what a provider step wants written to the home config.
type providerWire struct {
	llm   map[string]any
	embed embedOverride
}

type localLLMProvider struct {
	enabled bool
	baseURL string
	model   string
	embed   embedOverride
}

type externalLLMProvider struct {
	enabled bool
	baseURL string
	model   string
	embed   embedOverride
}

// embedOverride carries only the embedder keys the step owns.
type embedOverride struct {
	provider string
	baseURL  string
	model    string
}

func (p localLLMProvider) wire() providerWire {
	return providerWire{
		llm:   map[string]any{"enabled": p.enabled, "base_url": p.baseURL, "model": p.model},
		embed: p.embed,
	}
}

func (p externalLLMProvider) wire() providerWire {
	return providerWire{
		llm:   map[string]any{"enabled": p.enabled, "base_url": p.baseURL, "model": p.model},
		embed: p.embed,
	}
}

type providerConfig interface {
	wire() providerWire
}

// homeIndexingPath is the machine-local config the step merges into.
func homeIndexingPath(c *Config) string {
	home := c.HomeDir
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".skillgrid", "config.d", "indexing.yaml")
}

// mergeHomeProviderConfig merges the provider's llm + embedder keys into the
// home indexing.yaml, preserving every unrelated key. It never overwrites the
// whole file: the existing document is unmarshaled, the owned keys are set
// (llm as an exact block; embedder per-key, leaving e.g. dimension/model_dir
// untouched), and it is marshaled back with 0600 perms.
func mergeHomeProviderConfig(c *Config, p providerConfig) error {
	path := homeIndexingPath(c)
	w := p.wire()

	doc := map[string]any{"profile": "default"}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		var existing map[string]any
		if err := yaml.Unmarshal(data, &existing); err == nil {
			doc = existing
		}
	}
	if doc["profile"] == nil {
		doc["profile"] = "default"
	}

	mn, ok := doc["mnemonic"].(map[string]any)
	if !ok {
		mn = map[string]any{}
	}
	mn["llm"] = w.llm
	if emb, ok := mn["embedder"].(map[string]any); ok {
		emb["provider"] = w.embed.provider
		emb["base_url"] = w.embed.baseURL
		emb["model"] = w.embed.model
	} else {
		mn["embedder"] = map[string]any{
			"provider": w.embed.provider,
			"base_url": w.embed.baseURL,
			"model":    w.embed.model,
		}
	}
	doc["mnemonic"] = mn

	if c.DryRun {
		Out("      [dry-run] write", path, "(mnemonic.llm + mnemonic.embedder)")
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	// Atomic write: a crash or concurrent reader mid-WriteFile would otherwise
	// see a truncated/empty indexing.yaml and lose the operator's home config.
	// Write a temp file in the same directory and rename over the target —
	// rename is atomic on the same filesystem, so readers see either the old
	// or the new whole file, never a partial one.
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".indexing-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
