package install

// Provider setup (ADR-0023, TICKET-03). Install runs this as a normal step so
// the LLM/embedder host a machine needs for code indexing is wired naturally
// on first install and re-ensured (idempotent update) on re-run — not an
// opt-in bolt-on. Two hosts:
//
//   - "local" (default, Ollama): ensure the binary + service, pull the
//     six-model catalog only if absent (heavy models gated behind the Ollama
//     0.35.1 floor, skipped with a warning when below it), then merge home
//     mnemonic.llm + mnemonic.embedder (provider: ollama).
//   - "external" (OpenAI-compatible): no Ollama at all; merge home mnemonic.llm
//     + mnemonic.embedder (provider: external) against the operator's host.
//
// The step is NON-FATAL: every failure warns with a manual hint and returns,
// so a flaky install/serve never aborts the rest of the install (mirrors
// installSecurityTools). The home config merge only sets/overrides the keys it
// owns (llm.{enabled,base_url,model}; embedder.{provider,base_url,model}) and
// preserves every unrelated key already present in the operator's file.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// goos isolates runtime.GOOS so tests can force a platform (e.g. "linux").
var goos = runtime.GOOS

// Local provider model names (the live chat + embedder the runtime reads from
// the home config).
const (
	localLLMModel   = "llama3.2:1b"
	localEmbedModel = "embeddinggemma:300m"
)

// modelRole is the install-time role a catalog entry serves.
type modelRole string

const (
	roleSystemOne modelRole = "system-one"
	roleResearch  modelRole = "research"
	roleChat      modelRole = "chat"
	roleEmbedder  modelRole = "embedder"
)

// modelEntry is one tag in the local Ollama catalog. MinOllamaVersion is the
// floor the install requires before pulling it; heavy models carry a floor and
// are skipped (with a version-named warning) below it.
type modelEntry struct {
	Name             string
	Role             modelRole
	MinOllamaVersion string
}

// localModels is the six-model local catalog the install pulls. The chat and
// embedder entries double as the runtime defaults (localLLMModel /
// localEmbedModel). `glm:vision-tools` is intentionally absent (tag typo,
// removed in the interview).
func localModels() []modelEntry {
	return []modelEntry{
		{Name: "qwen2.5:1.5b", Role: roleSystemOne},
		{Name: "tev1:0.8b", Role: roleResearch, MinOllamaVersion: "0.35.1"},
		{Name: "embeddinggemma:300m", Role: roleEmbedder},
		{Name: "gemma2:2b", Role: roleResearch},
		{Name: "clef-flash", Role: roleResearch, MinOllamaVersion: "0.35.1"},
		{Name: "llama3.2:1b", Role: roleChat},
	}
}

// heavyModels returns the subset of the catalog gated behind an Ollama version
// floor (the models skipped with a warning when the local version is below the
// floor).
func heavyModels() map[string]string {
	heavy := map[string]string{}
	for _, e := range localModels() {
		if e.MinOllamaVersion != "" {
			heavy[e.Name] = e.MinOllamaVersion
		}
	}
	return heavy
}

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
	// Reset the smoke-recorded embed dimension for this run; the embed smoke
	// (in pullMissingModels) repopulates it, and the home merge writes it.
	lastEmbedDimension = 0
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
	embed := embedOverride{provider: "ollama", baseURL: ollamaBaseURL + "/v1", model: localEmbedModel}
	if lastEmbedDimension > 0 {
		embed.embedDimension = lastEmbedDimension
		// Req 5: if the operator's existing home config records a different
		// dimension, warn that the index is stale until reindexed. The model
		// still switches (fail-open).
		if prev, ok := readHomeEmbedderDimension(c); ok && prev > 0 && prev != lastEmbedDimension {
			Out("      warn embedder dimension mismatch: home config has", prev, "but the smoke recorded", lastEmbedDimension,
				"— search is stale until you reindex (mnemonic reindex)")
		}
	}
	return mergeHomeProviderConfig(c, localLLMProvider{
		enabled: true,
		baseURL: ollamaBaseURL + "/v1",
		model:   localLLMModel,
		embed:   embed,
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

// ollamaVersion returns the Ollama version string reported by the server, or
// "" when it cannot be determined. It is an HTTP seam (a package var) so tests
// can pin a version without a live `ollama --version` round-trip.
var ollamaVersion = func(base string) string {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(base + "/api/version")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var body struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return ""
	}
	return body.Version
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

// pullMissingModels lists installed models and pulls only the catalog entries
// absent from the server. Heavy models gated behind an Ollama version floor are
// skipped (with a version-named warning) when the local version is below the
// floor; the install still succeeds. Pull failures warn (non-fatal) — the
// config merge still runs.
func pullMissingModels(c *Config, base string) {
	present := ollamaModels(base)
	version := ollamaVersion(base)
	heavy := heavyModels()
	for _, e := range localModels() {
		if floor, gated := heavy[e.Name]; gated && !ollamaVersionAtLeast(version, floor) {
			Out("      skip", e.Name, "(requires Ollama >=", floor, "— current version", orUnknown(version), "; heavy model, not pulled)")
			continue
		}
		if present[e.Name] {
			Out("      skip pull", e.Name, "(already present)")
			continue
		}
		if c.DryRun {
			Out("      [dry-run] ollama pull", e.Name)
			continue
		}
		Out("      ollama pull", e.Name)
		if err := runCmd("ollama", "pull", e.Name); err != nil {
			Out("      warn pull", e.Name, ":", err)
		} else {
			// Non-fatal per-model health check: a failing probe warns and is
			// recorded, but never aborts the install (ADR-0016 fail-open).
			if !smokeProbe(base, localModels(), e.Role) {
				Out("      warn smoke", e.Name, ":", e.Role, "— post-pull probe did not report healthy (non-fatal)")
			}
		}
	}
}

// orUnknown renders a possibly-empty version string for a human-facing warning.
func orUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}

// smokeProbe is a non-fatal per-model health check run after a successful
// pull (req 3). It dispatches to the endpoint the model's role actually
// speaks: chat → /v1/chat/completions (pass on a non-empty first choice),
// embed → /api/embed (pass on a vector of length > 0, and records its length
// on the embedder so the home merge can write mnemonic.embedder.dimension),
// systemone → /v1/systemone (pass on a present answer field), research → a
// no-op pass (the tag is present after the pull; there is no dedicated
// research endpoint). It is a package var so tests can make it fail without
// reshaping the install step. A failing probe must never abort the install
// (ADR-0016 fail-open); the caller warns and continues.
var smokeProbe = func(base string, models []modelEntry, role modelRole) bool {
	switch role {
	case roleChat:
		return chatSmokeOK(base, chatModel(models, roleChat))
	case roleEmbedder:
		return embedSmokeOK(base, embedModel(models, roleEmbedder))
	case roleSystemOne:
		return systemoneSmokeOK(base, chatModel(models, roleSystemOne))
	case roleResearch:
		// Research tags (tev1, gemma2, clef-flash) have no dedicated
		// readiness endpoint; presence after a successful pull is the check.
		return true
	default:
		return true
	}
}

// chatModel returns the first catalog entry carrying the given role, falling
// back to the runtime default for that role when the catalog is searched for a
// smoke after a pull.
func chatModel(models []modelEntry, role modelRole) string {
	for _, e := range models {
		if e.Role == role {
			return e.Name
		}
	}
	if role == roleChat {
		return localLLMModel
	}
	return localLLMModel
}

// embedModel returns the catalog embedder tag (the runtime embedder default).
func embedModel(models []modelEntry, role modelRole) string {
	for _, e := range models {
		if e.Role == role {
			return e.Name
		}
	}
	return localEmbedModel
}

// chatSmokeOK POSTs one user message to /v1/chat/completions and reports
// whether the first choice returned non-empty content.
func chatSmokeOK(base, model string) bool {
	body, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]any{
			{"role": "user", "content": "ping"},
		},
	})
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(base+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false
	}
	if len(out.Choices) == 0 {
		return false
	}
	return strings.TrimSpace(out.Choices[0].Message.Content) != ""
}

// embedSmokeOK POSTs one string to /api/embed and reports whether the returned
// vector has length > 0. On success it records the vector length so the home
// merge can write mnemonic.embedder.dimension (req 3).
func embedSmokeOK(base, model string) bool {
	dim := ollamaEmbedDimension(base, model)
	if dim == 0 {
		return false
	}
	lastEmbedDimension = dim
	return true
}

// systemoneSmokeOK POSTs one yes/no question to /v1/systemone and reports
// whether the response carries an answer field.
func systemoneSmokeOK(base, model string) bool {
	body, _ := json.Marshal(map[string]any{
		"model":  model,
		"prompt": "Is 2+2 equal to 4? Answer yes or no.",
	})
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(base+"/v1/systemone", "application/json", bytes.NewReader(body))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var out struct {
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false
	}
	return strings.TrimSpace(out.Answer) != ""
}

// lastEmbedDimension holds the vector length recorded by the most recent
// successful embed smoke. The home merge reads it to write
// mnemonic.embedder.dimension (req 3/5). It is reset at the start of each
// install run by setupProviderLocal.
var lastEmbedDimension int

// ollamaEmbedDimension POSTs /api/embed for the embedder model and returns the
// vector length, or 0 when the server is unreachable or returns no vector.
func ollamaEmbedDimension(base, model string) int {
	body, _ := json.Marshal(map[string]any{"model": model, "input": "skillgrid smoke"})
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(base+"/api/embed", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	var out struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0
	}
	return len(out.Embedding)
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
	// ollamaModels is the catalog list written to mnemonic.ollama.models
	// (req 5). nil when the provider does not publish a local catalog (external).
	ollamaModels []map[string]any
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

// embedOverride carries only the embedder keys the step owns. embedDimension
// is the smoke-recorded vector length (req 3); 0 means "do not write
// mnemonic.embedder.dimension" (the smoke did not record one).
type embedOverride struct {
	provider       string
	baseURL        string
	model          string
	embedDimension int
}

func (p localLLMProvider) wire() providerWire {
	return providerWire{
		llm:          map[string]any{"enabled": p.enabled, "base_url": p.baseURL, "model": p.model},
		embed:        p.embed,
		ollamaModels: ollamaCatalogList(),
	}
}

func (p externalLLMProvider) wire() providerWire {
	return providerWire{
		llm:   map[string]any{"enabled": p.enabled, "base_url": p.baseURL, "model": p.model},
		embed: p.embed,
	}
}

// ollamaCatalogList projects the local catalog into the mnemonic.ollama.models
// list (req 5): one entry per catalog tag with its role as the kind. The
// research-only and systemone roles are folded into "systemone" and "embed"
// so the list mirrors the functional roles the runtime actually speaks.
func ollamaCatalogList() []map[string]any {
	out := make([]map[string]any, 0, len(localModels()))
	for _, e := range localModels() {
		out = append(out, map[string]any{"name": e.Name, "kind": kindForRole(e.Role)})
	}
	return out
}

// kindForRole maps a catalog role to the human-facing kind recorded in
// mnemonic.ollama.models.
func kindForRole(role modelRole) string {
	switch role {
	case roleChat:
		return "chat"
	case roleEmbedder:
		return "embed"
	case roleSystemOne, roleResearch:
		return "systemone"
	default:
		return string(role)
	}
}

type providerConfig interface {
	wire() providerWire
}

// parseOllamaVersion extracts the first "major.minor.patch" version string from
// `ollama --version` output. It returns the matched string, or "" when no
// semver-like version is present.
func parseOllamaVersion(out string) string {
	fields := strings.Fields(out)
	for _, f := range fields {
		if f == "" {
			continue
		}
		if strings.Count(f, ".") >= 1 && isNumericSegments(f) {
			return f
		}
	}
	return ""
}

// isNumericSegments reports whether s is a dot-separated sequence of numeric
// segments (e.g. "0.35.1" or "0.35").
func isNumericSegments(s string) bool {
	for _, seg := range strings.Split(s, ".") {
		if seg == "" {
			return false
		}
		for _, r := range seg {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// versionLessThan reports whether a is a strictly lower numeric version than b.
// Both must be dot-separated numeric segments; missing segments are treated as
// zero, so "0.35" < "0.35.1" and "0.35.1" < "0.36.0".
func versionLessThan(a, b string) bool {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var av, bv int
		if i < len(as) {
			av, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(bs[i])
		}
		if av != bv {
			return av < bv
		}
	}
	return false
}

// ollamaVersionAtLeast reports whether the parsed Ollama version in out is at
// least floor. An unparseable version is reported as below the floor so the
// caller can warn without failing the install.
func ollamaVersionAtLeast(out, floor string) bool {
	v := parseOllamaVersion(out)
	if v == "" {
		return false
	}
	return !versionLessThan(v, floor)
}

// readHomeEmbedderDimension reads the existing mnemonic.embedder.dimension from
// the home indexing.yaml (the pre-merge value, for the req-5 mismatch warning).
// It returns the dimension and ok=false when the file is absent, unparseable,
// or the key is not a positive number.
func readHomeEmbedderDimension(c *Config) (int, bool) {
	data, err := os.ReadFile(homeIndexingPath(c))
	if err != nil || len(data) == 0 {
		return 0, false
	}
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return 0, false
	}
	mn, ok := doc["mnemonic"].(map[string]any)
	if !ok {
		return 0, false
	}
	emb, ok := mn["embedder"].(map[string]any)
	if !ok {
		return 0, false
	}
	switch v := emb["dimension"].(type) {
	case int:
		if v > 0 {
			return v, true
		}
	case float64:
		if v > 0 {
			return int(v), true
		}
	}
	return 0, false
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
		if w.embed.embedDimension > 0 {
			emb["dimension"] = w.embed.embedDimension
		}
	} else {
		emb := map[string]any{
			"provider": w.embed.provider,
			"base_url": w.embed.baseURL,
			"model":    w.embed.model,
		}
		if w.embed.embedDimension > 0 {
			emb["dimension"] = w.embed.embedDimension
		}
		mn["embedder"] = emb
	}
	// Req 5: publish the local catalog list (only the local provider carries
	// one; external leaves mnemonic.ollama untouched).
	if w.ollamaModels != nil {
		mn["ollama"] = map[string]any{"models": w.ollamaModels}
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
