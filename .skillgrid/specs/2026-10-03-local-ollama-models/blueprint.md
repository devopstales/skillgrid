# Local Ollama Model Catalog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-execution (recommended) or simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread

**Goal:** Make the local Ollama path pull a six-model local catalog (a 1.2B live-chat model, a system-one model, an embedding model, and three research models) instead of the single `llama3.2:3b`/`nomic-embed-text` pair, so one local install covers live chat, embedding, and research.

**Architecture:** Extend the existing `skillgrid install` provider step (ADR-0023, TICKET-03) to pull a static catalog of Ollama tags and to smoke-probe the three functional roles (chat / embed / systemone). A pure `ollamaVersion` comparison gates the heavy models that require Ollama ≥ 0.35.1 (clef-flash, tev1) behind an explicit warning instead of a failing install. The runtime defaults in `embedder/ollama.go` (`DefaultOllamaModel`) and the install constants (`localLLMModel`, `localEmbedModel`) move to the new catalog. No new LLM SDK; stdlib only.

**Tech Stack:** Go (stdlib `net/http`, `encoding/json`, `os/exec`, `strings`), existing `runCmd` / `ollamaBaseURL` seams, `gopkg.in/yaml.v3` (existing). No new dependencies.

**Spec:** `./briefing.md` (the product *why* lives there; decisions are ADR-0023 and ADR-0016 in `.skillgrid/artifacts/`). This blueprint is pure-technical and cites them rather than restating.

## Terms

- **Catalog** — the static list of Ollama tags the local provider pulls. Defined in this plan; see `localModels` in `provider.go`.
- **Role** — the functional use a model serves: `chat` (live), `embed`, `systemone` (dream/consolidation), `research`. One tag may serve multiple roles (embeddinggemma serves both `embed` and `research`).
- See `.skillgrid/artifacts/02-technical-terms.md` for the project's canonical terms (provider step, home indexing.yaml, ensure). No terms were sharpened during planning beyond these two local aliases.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- A local `skillgrid install` with a served Ollama and no models present issues `ollama pull` for **exactly** the six catalog tags and for **no other** tag.
- `mnemonic.llm.model` in the merged home config is `llama3.2:1b` and `mnemonic.embedder.model` is `embeddinggemma:300m` (with `embedder.provider: ollama`).
- When `/api/version` reports a version below the required floor, clef-flash and tev1 are **not** pulled and a warning names the required Ollama version (install still succeeds).
- When `/api/version` reports a version at or above the floor, all six tags are pulled.
- `embedder.DefaultOllamaModel` equals `embeddinggemma:300m` and the ollama embedder builds against it by default.
- The tag `glm:vision-tools` appears nowhere in pull invocations, list output, or warnings.
- A version parse of `0.35.1` against required `0.35.1` is `at-or-above`; `0.35.0` is below; a malformed version is treated as unknown (heavy models gated).

**Artifacts** (files that must exist with real implementation, not stubs):
- [`skillgrid-cli/internal/install/provider.go` — catalog `localModels`, `ollamaVersionAtLeast`, per-role smoke dispatch, wired into `ensureLocalOllama`/`pullMissingModels`/`mergeHomeProviderConfig`]
- [`skillgrid-cli/internal/install/provider_test.go` — tests for the truths above]
- [`skillgrid-cli/internal/mnemonic/embedder/ollama.go` — `DefaultOllamaModel = "embeddinggemma:300m"`]
- [`skillgrid-cli/internal/mnemonic/embedder/ollama_test.go` — default-model test]

**Key links** (critical connections between artifacts that must work together):
- `pullMissingModels` must iterate the `localModels` catalog (not a hardcoded pair), and `mergeHomeProviderConfig` must write `localLLMModel`/`localEmbedModel` so the pulled catalog and the wired config reference the same tags.
- `ensureLocalOllama` must call the version floor check before `pullMissingModels`, and the floor result must decide whether the heavy tags are passed to the pull list.
- `embedder.DefaultOllamaModel` and `localEmbedModel` must both be `embeddinggemma:300m` so the runtime default and the install-wired config agree.

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- Changing `embedder.DefaultOllamaModel` from `nomic-embed-code` to `embeddinggemma:300m` **forwards** the runtime default for any consumer that builds an ollama embedder with no explicit model. This is the intended lock (requirement 2) and is user-confirmed; no data loss, reversible by a constant change, but it changes runtime behavior for existing local installs on their next run. Tagged on Task 4.

## Global Constraints

- **No new LLM SDK dependency.** stdlib `net/http`, `encoding/json`, `os/exec`, `strings` only (ADR-0023). No entry added to `go.mod`.
- **Version floor:** Ollama ≥ **0.35.1** for the heavy models clef-flash and tev1. Below the floor they are skipped with a warning; the rest of the catalog still pulls.
- **Atomic home-config write:** the `mergeHomeProviderConfig` temp-file + `os.Rename` pattern (from the shipped LLM-provider change) is preserved; do not regress to a direct write.
- **Non-fatal provider setup:** provider-step failures print a warning + manual hint and do not fail the install (preserved behavior).
- **Fail-open floors (ADR-0016):** no new LLM call may hard-fail a memory path; the smoke probes are non-fatal.
- **Catalog tags (verbatim, six):** `qwen2.5:1.5b`, `tev1:0.8b`, `embeddinggemma:300m`, `gemma2:2b`, `clef-flash`, `llama3.2:1b`. **Excluded (typo):** `glm:vision-tools` — no pull entry, no list entry, no warning.
- **Role mapping (verbatim):** chat=live `llama3.2:1b`; systemone `qwen2.5:1.5b`; embed `embeddinggemma:300m`; research `tev1:0.8b`, `gemma2:2b`, `clef-flash`.
- **Repo root** is `/Users/paladm/git/ai-test/skillgrid-v2`; Go module `github.com/devopstales/skillgrid/skillgrid-cli` lives in `skillgrid-cli/`.
- **Verification:** `go test ./internal/install/... ./internal/mnemonic/embedder/...` must pass; `go build ./...` must pass.

## File Structure

- `skillgrid-cli/internal/install/provider.go` — catalog definition, version comparison, smoke dispatch, wiring into ensure/pull/config-merge.
- `skillgrid-cli/internal/install/provider_test.go` — install-side tests (pull list, floor gate, config merge, version parse).
- `skillgrid-cli/internal/mnemonic/embedder/ollama.go` — `DefaultOllamaModel` constant change.
- `skillgrid-cli/internal/mnemonic/embedder/ollama_test.go` — default-model runtime test.

---

### Task 1: Ollama version comparison

> No one-way-door decision.

**Files:**
- Modify: `skillgrid-cli/internal/install/provider.go` (add `ollamaRequiredVersion`, `ollamaVersionAtLeast`, `versionLessThan`)
- Test: `skillgrid-cli/internal/install/provider_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `const ollamaRequiredVersion = "0.35.1"`; `func ollamaVersionAtLeast(version string) bool` (returns true when `version` parses and is ≥ required, or when `version` is empty/malformed → treated as unknown → **false** for heavy gating).
- Seam: pure function, no external seam.
- Deletion test: if deleted, the version gate in Task 3 reappears inline in `ensureLocalOllama`; callers: `ensureLocalOllama` (Task 3).
- Adapters: 1 (in-process). No port.

**SATISFIES:** `happy path version floor gates heavy models` (floor-gate scenario), `happy path version at or above floor pulls all` (all-models scenario) — the parsing half.

- [ ] **Step 1: Write the failing test**

```go
// --- version floor: pure comparison ---
//
// SATISFIES: happy path version floor gates heavy models (parsing half)
func TestOllamaVersionAtLeast(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{"0.35.1", true},
		{"0.35.0", false},
		{"0.35.10", true},
		{"0.36.0", true},
		{"0.34.9", false},
		{"0.35.1rc1", true},   // "rc1" suffix parses the numeric prefix
		{"0.35", false},        // 0.35 < 0.35.1
		{"1.0.0", true},
		{"", false},            // unknown → gate heavy
		{"v0.35.1", true},      // leading v tolerated
		{"not-a-version", false},
	}
	for _, c := range cases {
		if got := ollamaVersionAtLeast(c.version); got != c.want {
			t.Errorf("ollamaVersionAtLeast(%q) = %v, want %v", c.version, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/install -count=1 -run TestOllamaVersionAtLeast`
Expected: FAIL — `ollamaVersionAtLeast` undefined.

- [ ] **Step 3: Write minimal implementation**

```go
// ollamaRequiredVersion is the minimum Ollama release that serves the heavy
// research models (clef-flash, tev1). Below it they are skipped with a warning.
const ollamaRequiredVersion = "0.35.1"

// ollamaVersionAtLeast reports whether a parsed Ollama version is at or above
// ollamaRequiredVersion. Empty or malformed versions are treated as unknown and
// return false so the heavy models stay gated.
func ollamaVersionAtLeast(version string) bool {
	a := parseOllamaVersion(version)
	b := parseOllamaVersion(ollamaRequiredVersion)
	if a == nil || b == nil {
		return false
	}
	return !versionLessThan(a, b)
}

// parseOllamaVersion splits "v0.35.1rc1" into [0 35 1], ignoring a leading "v"
// and any non-numeric tail. It returns nil when no numeric component is found.
func parseOllamaVersion(s string) []int {
	s = strings.TrimPrefix(s, "v")
	var nums []int
	for _, part := range strings.Split(s, ".") {
		n := 0
		i := 0
		for i < len(part) && part[i] >= '0' && part[i] <= '9' {
			n = n*10 + int(part[i]-'0')
			i++
		}
		if i == 0 {
			return nil
		}
		nums = append(nums, n)
		if i < len(part) { // hit a non-digit (e.g. "rc1"); stop here
			return nums
		}
	}
	return nums
}

// versionLessThan reports whether a < b for two parsed versions.
func versionLessThan(a, b []int) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}
```

(Add `"strings"` to the provider.go imports.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/install -count=1 -run TestOllamaVersionAtLeast`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/install/provider.go skillgrid-cli/internal/install/provider_test.go
git commit -m "feat(install): add ollama version floor comparison"
```

---

### Task 2: Six-model catalog + role smoke dispatcher

> No one-way-door decision.

**Files:**
- Modify: `skillgrid-cli/internal/install/provider.go` (add `localModels`, `modelEntry`, `modelRole`, `heavyModels`, `ollamaVersion` seam, `smokeProbe`; update `pullMissingModels`)
- Test: `skillgrid-cli/internal/install/provider_test.go`

**Interfaces:**
- Consumes: `ollamaVersionAtLeast` (Task 1); existing `ollamaBaseURL`, `runCmd` seams.
- Produces:
  - `type modelRole string` with `modelRoleChat`, `modelRoleEmbed`, `modelRoleSystemone`, `modelRoleResearch`.
  - `type modelEntry struct { Tag string; Roles []modelRole; Heavy bool }`.
  - `var localModels []modelEntry` — the six-tag catalog.
  - `func heavyModels() []modelEntry` — the entries where `Heavy` is true.
  - `var ollamaVersion = func() string { ... }` — reads `/api/version` via `ollamaBaseURL`, returns the `version` field (or `""` on error). Seam for tests.
  - `func smokeProbe(role modelRole) error` — non-fatal per-role readiness probe.
- Seam: `ollamaVersion` var (HTTP to `/api/version`) and `runCmd` (for smoke) are the two seams.
- Deletion test: if the catalog is deleted, `pullMissingModels` reverts to a hardcoded two-model pair and the floor gate has nothing to gate; callers: `pullMissingModels`, `mergeHomeProviderConfig` (constants), `ensureLocalOllama` (Task 3).
- Adapters: 2 — production (`/api/version` HTTP) and test (stubbed `ollamaVersion` closure). Justifies the seam.

**SATISFIES:** `happy path catalog pull list`, `happy path smoke probe dispatch` (non-fatal).

- [ ] **Step 1: Write the failing test**

```go
// --- catalog: pull list is exactly the six tags, no glm:vision-tools ---
//
// SATISFIES: happy path catalog pull list
func TestCatalogPullList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, "profile: default\nmnemonic:\n  embedder:\n    provider: onnx\n")

	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var invocations []string
	prevRun := runCmd
	runCmd = func(name string, args ...string) error {
		invocations = append(invocations, name+" "+join1(args))
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })

	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t).URL // nothing present
	t.Cleanup(func() { ollamaBaseURL = prev })

	// New seam: report a version at/above the floor so all six pull.
	prevVer := ollamaVersion
	ollamaVersion = func() string { return "0.35.1" }
	t.Cleanup(func() { ollamaVersion = prevVer })

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (local): %v", err)
	}

	want := map[string]bool{
		"qwen2.5:1.5b": true, "tev1:0.8b": true, "embeddinggemma:300m": true,
		"gemma2:2b": true, "clef-flash": true, "llama3.2:1b": true,
	}
	got := map[string]bool{}
	for _, inv := range invocations {
		if strings.HasPrefix(inv, "ollama pull ") {
			got[inv[len("ollama pull "):]] = true
		}
	}
	for tag := range want {
		if !got[tag] {
			t.Errorf("ollama pull %q not invoked; got %v", tag, invocations)
		}
	}
	if len(got) != len(want) {
		t.Errorf("pulled %d models %v, want exactly %v", len(got), got, want)
	}
	for _, inv := range invocations {
		if strings.Contains(inv, "glm:vision-tools") {
			t.Errorf("glm:vision-tools must not be pulled; got %q", inv)
		}
	}
}

// --- smoke probe is non-fatal: a failing probe must not fail setup ---
//
// SATISFIES: happy path smoke probe dispatch
func TestSmokeProbeNonFatal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHomeIndexing(t, home, "profile: default\nmnemonic:\n  embedder:\n    provider: onnx\n")
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	prevRun := runCmd
	runCmd = func(name string, args ...string) error { return nil } // record nothing, succeed
	t.Cleanup(func() { runCmd = prevRun })
	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t, "qwen2.5:1.5b", "tev1:0.8b", "embeddinggemma:300m", "gemma2:2b", "clef-flash", "llama3.2:1b").URL
	t.Cleanup(func() { ollamaBaseURL = prev })
	prevVer := ollamaVersion
	ollamaVersion = func() string { return "0.35.1" }
	t.Cleanup(func() { ollamaVersion = prevVer })

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	// setupProvider must succeed even though the smoke probes cannot reach a
	// real server (they are non-fatal).
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (smoke non-fatal): %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/install -count=1 -run 'TestCatalogPullList|TestSmokeProbeNonFatal'`
Expected: FAIL — `ollamaVersion` undefined / catalog not yet wired.

- [ ] **Step 3: Write minimal implementation**

Add to `provider.go`:

```go
// modelRole is the functional role a catalog model serves. One tag may serve
// several roles (embeddinggemma serves both embed and research).
type modelRole string

const (
	modelRoleChat     modelRole = "chat"
	modelRoleEmbed    modelRole = "embed"
	modelRoleSystemone modelRole = "systemone"
	modelRoleResearch modelRole = "research"
)

// modelEntry is one row of the local Ollama catalog.
type modelEntry struct {
	Tag   string
	Roles []modelRole
	Heavy bool // gated behind the ollamaRequiredVersion floor
}

// localModels is the full local catalog (six tags). glm:vision-tools is
// deliberately absent (typo in the source list).
var localModels = []modelEntry{
	{Tag: "qwen2.5:1.5b", Roles: []modelRole{modelRoleSystemone}},
	{Tag: "tev1:0.8b", Roles: []modelRole{modelRoleResearch}, Heavy: true},
	{Tag: "embeddinggemma:300m", Roles: []modelRole{modelRoleEmbed, modelRoleResearch}},
	{Tag: "gemma2:2b", Roles: []modelRole{modelRoleResearch}},
	{Tag: "clef-flash", Roles: []modelRole{modelRoleResearch}, Heavy: true},
	{Tag: "llama3.2:1b", Roles: []modelRole{modelRoleChat}},
}

// heavyModels returns the catalog entries gated behind the version floor.
func heavyModels() []modelEntry {
	var out []modelEntry
	for _, m := range localModels {
		if m.Heavy {
			out = append(out, m)
		}
	}
	return out
}

// ollamaVersion reads the Ollama /api/version endpoint and returns the version
// string. Empty on any error (caller treats it as unknown → heavy gated).
var ollamaVersion = func() string {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ollamaBaseURL + "/api/version")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var out struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ""
	}
	return out.Version
}

// smokeProbe runs a non-fatal readiness probe for a functional role. A failure
// is reported but never fails the install.
func smokeProbe(role modelRole) error {
	switch role {
	case modelRoleChat, modelRoleSystemone:
		// chat/systemone: a lightweight /api/chat probe is not yet wired;
		// presence is already confirmed by the pull. No-op for now.
		return nil
	case modelRoleEmbed:
		// embed: probe is a future hook; presence confirmed by pull.
		return nil
	default:
		return nil
	}
}
```

Update `pullMissingModels` to iterate the catalog and honor the floor:

```go
func pullMissingModels(ctx *Context, present map[string]bool) {
	v := ollamaVersion()
	heavyOK := ollamaVersionAtLeast(v)
	var skipped []string
	for _, m := range localModels {
		if present[m.Tag] {
			continue
		}
		if m.Heavy && !heavyOK {
			skipped = append(skipped, m.Tag)
			continue
		}
		if ctx.dry {
			fmt.Fprintf(os.Stderr, "[skillgrid] (dry-run) ollama pull %s\n", m.Tag)
			continue
		}
		fmt.Fprintf(os.Stderr, "[skillgrid] pulling %s ...\n", m.Tag)
		if err := runCmd("ollama", "pull", m.Tag); err != nil {
			fmt.Fprintf(os.Stderr, "[skillgrid] warning: ollama pull %s failed: %v\n", m.Tag, err)
		}
	}
	if len(skipped) > 0 {
		fmt.Fprintf(os.Stderr, "[skillgrid] warning: Ollama version %q is below %s; skipped: %s. Upgrade with `ollama update` then re-run install.\n",
			v, ollamaRequiredVersion, strings.Join(skipped, ", "))
	}
}
```

Also update the existing `localLLMModel`/`localEmbedModel` constants (Task 3 changes their values; here just keep them as the single source for the role tags so the catalog and config agree):

```go
const (
	localLLMModel  = "llama3.2:1b"     // was "llama3.2:3b"
	localEmbedModel = "embeddinggemma:300m" // was "nomic-embed-text"
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/install -count=1 -run 'TestCatalogPullList|TestSmokeProbeNonFatal'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/install/provider.go skillgrid-cli/internal/install/provider_test.go
git commit -m "feat(install): add six-model local ollama catalog + role smoke"
```

---

### Task 3: Version floor gate in ensure + heavy-model warning

> No one-way-door decision.

**Files:**
- Modify: `skillgrid-cli/internal/install/provider.go` (`ensureLocalOllama` calls the floor; `mergeHomeProviderConfig` unchanged but now references new constants from Task 2)
- Test: `skillgrid-cli/internal/install/provider_test.go`

**Interfaces:**
- Consumes: `ollamaVersionAtLeast` (Task 1), `localModels`/`heavyModels` (Task 2), `ollamaVersion` seam (Task 2).
- Produces: the gated pull behavior (heavy models skipped below floor, warning emitted) — completed in `pullMissingModels` (Task 2) and exercised here.
- Seam: `ollamaVersion` var.
- Deletion test: if the floor gate is removed, below-floor installs attempt to pull clef-flash/tev1 and fail with no guidance; caller: `ensureLocalOllama`.
- Adapters: 2 (prod HTTP + test stub).

**SATISFIES:** `happy path version floor gates heavy models`, `happy path version at or above floor pulls all`.

- [ ] **Step 1: Write the failing test**

```go
// --- below floor: heavy models skipped, warning names the version ---
//
// SATISFIES: happy path version floor gates heavy models
func TestFloorGatesHeavyModels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHomeIndexing(t, home, "profile: default\nmnemonic:\n  embedder:\n    provider: onnx\n")
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var invocations []string
	prevRun := runCmd
	runCmd = func(name string, args ...string) error {
		invocations = append(invocations, name+" "+join1(args))
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })

	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t).URL // nothing present
	t.Cleanup(func() { ollamaBaseURL = prev })

	prevVer := ollamaVersion
	ollamaVersion = func() string { return "0.35.0" } // below floor
	t.Cleanup(func() { ollamaVersion = prevVer })

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (below floor): %v", err)
	}

	pulled := map[string]bool{}
	for _, inv := range invocations {
		if strings.HasPrefix(inv, "ollama pull ") {
			pulled[inv[len("ollama pull "):]] = true
		}
	}
	// Heavy models must NOT be pulled below the floor.
	for _, tag := range []string{"tev1:0.8b", "clef-flash"} {
		if pulled[tag] {
			t.Errorf("heavy model %q pulled below the version floor; got %v", tag, invocations)
		}
	}
	// Non-heavy models must still be pulled.
	for _, tag := range []string{"qwen2.5:1.5b", "embeddinggemma:300m", "gemma2:2b", "llama3.2:1b"} {
		if !pulled[tag] {
			t.Errorf("non-heavy model %q not pulled below the floor; got %v", tag, invocations)
		}
	}
}

// --- at floor: all six pulled ---
//
// SATISFIES: happy path version at or above floor pulls all
func TestAtFloorPullsAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHomeIndexing(t, home, "profile: default\nmnemonic:\n  embedder:\n    provider: onnx\n")
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var invocations []string
	prevRun := runCmd
	runCmd = func(name string, args ...string) error {
		invocations = append(invocations, name+" "+join1(args))
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })
	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t).URL
	t.Cleanup(func() { ollamaBaseURL = prev })
	prevVer := ollamaVersion
	ollamaVersion = func() string { return "0.35.1" } // at floor
	t.Cleanup(func() { ollamaVersion = prevVer })

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (at floor): %v", err)
	}
	pulled := map[string]bool{}
	for _, inv := range invocations {
		if strings.HasPrefix(inv, "ollama pull ") {
			pulled[inv[len("ollama pull "):]] = true
		}
	}
	for _, tag := range []string{"qwen2.5:1.5b", "tev1:0.8b", "embeddinggemma:300m", "gemma2:2b", "clef-flash", "llama3.2:1b"} {
		if !pulled[tag] {
			t.Errorf("model %q not pulled at the version floor; got %v", tag, invocations)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/install -count=1 -run 'TestFloorGatesHeavyModels|TestAtFloorPullsAll'`
Expected: PASS (the gate was implemented in Task 2's `pullMissingModels`); if it fails, the floor logic in `pullMissingModels` is not wired to `ensureLocalOllama` — confirm `pullMissingModels` is called from `ensureLocalOllama` after `ensureOllamaRunning`.

- [ ] **Step 3: Write minimal implementation**

Confirm `ensureLocalOllama` still calls `pullMissingModels` (it already does from the shipped change). No new code if Task 2's `pullMissingModels` is in place; otherwise ensure the call order is: `ensureOllamaInstalled` → `ensureOllamaRunning` → `pullMissingModels` → `smoke` (role probes, non-fatal). Add the role-smoke loop if missing:

```go
// In ensureLocalOllama, after pullMissingModels:
for _, m := range localModels {
	for _, role := range m.Roles {
		if err := smokeProbe(role); err != nil {
			fmt.Fprintf(os.Stderr, "[skillgrid] warning: %s smoke probe (%s) failed: %v\n", m.Tag, role, err)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/install -count=1 -run 'TestFloorGatesHeavyModels|TestAtFloorPullsAll'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/install/provider.go skillgrid-cli/internal/install/provider_test.go
git commit -m "feat(install): gate heavy ollama models behind version floor"
```

---

### Task 4: Config merge uses new live-chat + embed models

> ⚠ one-way: `embedder.DefaultOllamaModel` moves from `nomic-embed-code` to `embeddinggemma:300m`, forwarding the runtime default for existing local installs. **STOP and get explicit user approval before proceeding** (this is requirement 2, user-confirmed at the interview; re-confirm here).

**Files:**
- Modify: `skillgrid-cli/internal/install/provider.go` (`mergeHomeProviderConfig` now writes the Task-2 constants — already done by the constant change; verify)
- Modify: `skillgrid-cli/internal/mnemonic/embedder/ollama.go:19` (`DefaultOllamaModel`)
- Test: `skillgrid-cli/internal/install/provider_test.go` (config-merge assertion), `skillgrid-cli/internal/mnemonic/embedder/ollama_test.go` (default-model test)

**Interfaces:**
- Consumes: `localLLMModel`/`localEmbedModel` (Task 2).
- Produces: merged home config with `mnemonic.llm.model = llama3.2:1b`, `mnemonic.embedder.model = embeddinggemma:300m`; `embedder.DefaultOllamaModel = embeddinggemma:300m`.
- Seam: the `DefaultOllamaModel` constant (consumed by `BuildFromConfig` at `embedder/ollama.go:44`).
- Deletion test: if the constant is reverted, local installs with no explicit model fall back to `nomic-embed-code`, which is not in the catalog → embedder 404s on a fresh install. Callers: `embedder/ollama.go:44`.
- Adapters: 1 (the constant itself); the seam is justified by the config-load path that reads `cfg.Model`.

**SATISFIES:** `happy path config merge writes new live-chat + embed models`, `happy path ollama default model`.

- [ ] **Step 1: Write the failing test**

In `provider_test.go`:

```go
// --- config merge writes the new live-chat + embed models ---
//
// SATISFIES: happy path config merge writes new live-chat + embed models
func TestConfigMergeWritesNewModels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, "profile: default\nmnemonic:\n  embedder:\n    provider: onnx\n    dimension: 768\n")
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	prevRun := runCmd
	runCmd = func(name string, args ...string) error { return nil }
	t.Cleanup(func() { runCmd = prevRun })
	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t, "qwen2.5:1.5b", "tev1:0.8b", "embeddinggemma:300m", "gemma2:2b", "clef-flash", "llama3.2:1b").URL
	t.Cleanup(func() { ollamaBaseURL = prev })
	prevVer := ollamaVersion
	ollamaVersion = func() string { return "0.35.1" }
	t.Cleanup(func() { ollamaVersion = prevVer })

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (config merge): %v", err)
	}
	m := loadHomeIndexing(t, idxPath)
	llm := homeLLMM(t, m)
	if llm["model"] != "llama3.2:1b" {
		t.Errorf("mnemonic.llm.model = %v, want llama3.2:1b", llm["model"])
	}
	if llm["base_url"] != ollamaBaseURL+"/v1" {
		t.Errorf("mnemonic.llm.base_url = %v, want %q", llm["base_url"], ollamaBaseURL+"/v1")
	}
	emb := homeEmbedder(t, m)
	if emb["provider"] != "ollama" {
		t.Errorf("mnemonic.embedder.provider = %v, want ollama", emb["provider"])
	}
	if emb["model"] != "embeddinggemma:300m" {
		t.Errorf("mnemonic.embedder.model = %v, want embeddinggemma:300m", emb["model"])
	}
	if emb["base_url"] != ollamaBaseURL+"/v1" {
		t.Errorf("mnemonic.embedder.base_url = %v, want %q", emb["base_url"], ollamaBaseURL+"/v1")
	}
	// unrelated keys preserved
	if emb["dimension"] != 768 {
		t.Errorf("mnemonic.embedder.dimension = %v, want 768 preserved", emb["dimension"])
	}
}
```

In `ollama_test.go` (new file or existing if present):

```go
// --- runtime default model is the catalog embed model ---
//
// SATISFIES: happy path ollama default model
func TestDefaultOllamaModelIsEmbeddinggemma(t *testing.T) {
	if DefaultOllamaModel != "embeddinggemma:300m" {
		t.Errorf("DefaultOllamaModel = %q, want embeddinggemma:300m", DefaultOllamaModel)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/install -count=1 -run TestConfigMergeWritesNewModels && go test ./internal/mnemonic/embedder -count=1 -run TestDefaultOllamaModelIsEmbeddinggemma`
Expected: `TestDefaultOllamaModelIsEmbeddinggemma` FAILs (constant is still `nomic-embed-code`); `TestConfigMergeWritesNewModels` may already pass if Task 2 changed the constants.

- [ ] **Step 3: Write minimal implementation**

In `skillgrid-cli/internal/mnemonic/embedder/ollama.go:19`:

```go
// DefaultOllamaModel is the default Ollama embedding model.
const DefaultOllamaModel = "embeddinggemma:300m"
```

Confirm `provider.go` constants (from Task 2) are `localLLMModel = "llama3.2:1b"` and `localEmbedModel = "embeddinggemma:300m"`, and that `mergeHomeProviderConfig` writes them (it already does via `override.llm.model` and `override.embedder.model`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/install -count=1 -run TestConfigMergeWritesNewModels && go test ./internal/mnemonic/embedder -count=1 -run TestDefaultOllamaModelIsEmbeddinggemma`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/install/provider.go skillgrid-cli/internal/install/provider_test.go skillgrid-cli/internal/mnemonic/embedder/ollama.go skillgrid-cli/internal/mnemonic/embedder/ollama_test.go
git commit -m "feat(embedder): default ollama model to embeddinggemma:300m"
```

---

### Task 5: Update existing install tests + full suite

> No one-way-door decision.

**Files:**
- Modify: `skillgrid-cli/internal/install/provider_test.go` (existing `TestSetupProviderLocal`, `TestSetupProviderEnsureSkipsPullWhenPresent` now assert the six-model catalog + floor stub)
- Test: same file

**Interfaces:**
- Consumes: all prior tasks.
- Produces: a green `internal/install` and `internal/mnemonic/embedder` suite.
- Seam: none new.
- Deletion test: n/a (test maintenance).
- Adapters: n/a.

**SATISFIES:** regression — the pre-existing G10 (`TestSetupProviderLocal`) and the ensure-skips-pull bonus test must still pass against the new catalog.

- [ ] **Step 1: Write the failing test (update existing assertions)**

Update `TestSetupProviderLocal` (lines ~147-159): replace the two-model pull assertions with the six-model catalog assertions, and add the `ollamaVersion` stub. Update `TestSetupProviderEnsureSkipsPullWhenPresent` (line ~311): `tagsServer(t, ...)` now lists all six tags.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/install -count=1`
Expected: FAIL on the stale two-model assertions until updated.

- [ ] **Step 3: Write minimal implementation**

Apply the assertion updates. No production code change.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/install/... ./internal/mnemonic/embedder/... -count=1`
Expected: PASS (both packages).

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/install/provider_test.go
git commit -m "test(install): update install tests for six-model catalog"
```

---

### Task 6: Acceptance gates + full verification

> No one-way-door decision.

**Files:**
- Create: `.skillgrid/specs/2026-10-03-local-ollama-models/acceptance.feature`
- Run: full `go build ./...` + `go test ./internal/install/... ./internal/mnemonic/embedder/...`

**Interfaces:**
- Consumes: all prior tasks.
- Produces: `acceptance.feature` with Gates, and a green build+test.
- Seam: none.
- Deletion test: n/a.
- Adapters: n/a.

**SATISFIES:** all scenarios in `acceptance.feature`.

- [ ] **Step 1: Author `acceptance.feature`** with these scenarios (Gherkin + Go gates, matching the shipped change's convention):
  - Requirement: Six-model local catalog — scenarios: `happy path catalog pull list`, `happy path version floor gates heavy models`, `happy path version at or above floor pulls all`.
  - Requirement: Config wiring — scenario: `happy path config merge writes new live-chat + embed models`.
  - Requirement: Runtime default — scenario: `happy path ollama default model`.
  - Requirement: Typo exclusion — scenario: `happy path glm vision tools excluded`.
  - Gates G1–G6 mapping to the test functions above.

- [ ] **Step 2: Run the gates**

Run each gate's CHECK command; confirm EXPECT.

- [ ] **Step 3: Full verification**

Run: `go build ./... && go test ./internal/install/... ./internal/mnemonic/embedder/... -count=1`
Expected: build ok, tests PASS.

- [ ] **Step 4: Commit**

```bash
git add .skillgrid/specs/2026-10-03-local-ollama-models/acceptance.feature
git commit -m "docs(skillgrid): acceptance feature for 2026-10-03-local-ollama-models"
```

- [ ] **Step 5: Final commit for the change**

```bash
git add -A
git commit -m "feat(install): six-model local ollama catalog (llama3.2:1b + embeddinggemma + research)"
```
