# Gryph Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the [gryph](https://github.com/safedep/gryph) agent-audit tool into the skillgrid installer: install its hooks for Kilo + OpenCode and enable its security policy.

**Architecture:** `config.d/tools.yaml` adds `@safedep/gryph` (npm registry package; its postinstall downloads the platform binary — same mechanism as `husky`). A new gated install step in `skillgrid-cli` runs `gryph install --agent opencode` (gryph's only adapter we need; it writes `~/.config/opencode/plugins/gryph.js`) and copies that plugin to `~/.config/kilo/plugins/gryph.js` (Kilo has no gryph adapter; this is the existing engram-bridge pattern at `cmd/steps.go:184`). Policy is scaffolded once (machine-global to gryph) via `policy init` → `policy validate` → `config set policy.enabled true`.

**Tech Stack:** Go (skillgrid-cli), YAML config (`config.d/`), existing `logging` package, stdlib `os/exec`.

**Spec:** `docs/superpowers/specs/2026-08-26-gryph-integration-design.md`

## Global Constraints

- `config.d/` is the single source of truth for what gets installed; no new package names in code except the `hasTool(tools.Tools, "gryph")` gate (existing pattern, see `cmd/install.go:89` for `agent-browser`).
- `--dry-run` must never execute the real `gryph` binary and never write files.
- The gryph plugin invokes the `gryph` binary by name from PATH at agent runtime — do not try to embed an absolute path; the installer prints PATH guidance at the end of every install (`cmd/path.go`).
- All install-step failures are `logging.Warn`, never fatal (existing convention throughout `cmd/steps.go`).
- Verification bar: `go build ./... && go test ./...` in `skillgrid-cli/` must pass.
- Commit messages: short imperative, one line (e.g. `feat: install gryph hooks for kilo and opencode`).
- No new Go dependencies.
- Working dir for all commands: `skillgrid-cli/` unless stated otherwise.

## File Structure

| File | Change | Responsibility |
|------|--------|----------------|
| `config.d/tools.yaml` | Modify | Declare `@safedep/gryph` (single source of truth) |
| `skillgrid-cli/cmd/install.go` | Modify (one block after line 96) | Gate + call `installGryph` from the install pipeline |
| `skillgrid-cli/cmd/steps.go` | Modify (append) | `installGryph` — hooks install + policy enable |
| `skillgrid-cli/cmd/gryph_test.go` | Create | Deterministic tests: dry-run log lines + real-path run with a fake `gryph` binary |
| `docs/02-usage.md` | Modify | Document what the install does for gryph + how to query it |
| `docs/03-config-reference.md` | Modify | Mention gryph's extra install step in `tools.yaml` semantics |
| `docs/NOTE.md` | Modify | Mark gryph as done under "Usage Data" |

---

### Task 1: Gryph hooks install step (config entry + Kilo/OpenCode hook install)

**Files:**
- Modify: `config.d/tools.yaml` (repo root)
- Modify: `skillgrid-cli/cmd/install.go:88-96` (region between the agent-browser step and "Step 5: install plugins")
- Modify: `skillgrid-cli/cmd/steps.go` (append at end of file, after `agentConfigPrefix`)
- Test: `skillgrid-cli/cmd/gryph_test.go` (new file)

**Interfaces:**
- Consumes: `hasTool(tools []string, want string) bool` (`cmd/steps.go:285`), `hasAgent(agents []string, want string) bool` (`cmd/steps.go:276`), `mustExpandHomePath(p string) string` (`cmd/install.go:285`), `logging.Info/Warn` — all exist, unchanged.
- Produces: `installGryph(baseDir string, agents []string, dryRun bool)` — called from `runInstall`; in this task it installs hooks only (policy is added in Task 2).

- [ ] **Step 1: Verify the tools.yaml entry is present**

`config.d/tools.yaml` must contain `@safedep/gryph` in `tools`. If a prior session already applied it, verify with:

Run (repo root): `git diff config.d/tools.yaml`
Expected: diff shows `+  - "@safedep/gryph"` added after `vercel-labs/agent-browser`. If not present, add the line so the file ends:

```yaml
tools:
  - "husky"
  - "vercel-labs/skills"
  - "@colbymchenry/codegraph"
  - "@upstash/context7-mcp"
  - "@playwright/mcp@latest"
  - "@playwright/cli@latest"
  - "@cucumber/cucumber"
  - "vercel-labs/agent-browser"
  - "@safedep/gryph"
```

- [ ] **Step 2: Write the failing tests**

Create `skillgrid-cli/cmd/gryph_test.go`:

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skillgrid-cli/internal/logging"
)

// fakeGryph sets up an isolated HOME with a fake `gryph` binary on PATH that
// appends its args to $GRYPH_LOG, so tests exercise the real exec path
// deterministically.
func fakeGryph(t *testing.T) (home, argLog string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	argLog = filepath.Join(home, "gryph-args.log")
	bin := filepath.Join(home, "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho \"$@\" >> \"${GRYPH_LOG:-/dev/null}\"\n"
	if err := os.WriteFile(filepath.Join(bin, "gryph"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRYPH_LOG", argLog)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return home, argLog
}

func captureGryph(t *testing.T, body func()) string {
	t.Helper()
	logging.ResetForTest()
	if err := logging.Init(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })
	body()
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestInstallGryphDryRunEmitsHookSteps(t *testing.T) {
	home := t.TempDir()
	// Pre-create the source plugin so the kilo dry-run branch logs the cp line.
	srcPlugin := filepath.Join(home, ".config", "opencode", "plugins", "gryph.js")
	if err := os.MkdirAll(filepath.Dir(srcPlugin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPlugin, []byte("FAKE-GRYPH-PLUGIN"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	out := captureGryph(t, func() {
		installGryph(t.TempDir(), []string{"kilo", "opencode"}, true)
	})
	for _, want := range []string{
		"[dry-run] gryph install --agent opencode",
		"[dry-run] cp " + srcPlugin + " " + filepath.Join(home, ".config", "kilo", "plugins", "gryph.js"),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "gryph hooks installed for opencode") {
		t.Fatalf("dry-run must not report a real install:\n%s", out)
	}
}

func TestInstallGryphRealRunCopiesKiloPlugin(t *testing.T) {
	home, _ := fakeGryph(t)
	srcPlugin := filepath.Join(home, ".config", "opencode", "plugins", "gryph.js")
	if err := os.MkdirAll(filepath.Dir(srcPlugin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPlugin, []byte("FAKE-GRYPH-PLUGIN"), 0644); err != nil {
		t.Fatal(err)
	}
	dstPlugin := filepath.Join(home, ".config", "kilo", "plugins", "gryph.js")

	out := captureGryph(t, func() {
		installGryph(t.TempDir(), []string{"kilo"}, false)
	})
	if !strings.Contains(out, "gryph plugin copied to "+dstPlugin) {
		t.Fatalf("expected kilo plugin copy, got:\n%s", out)
	}
	got, err := os.ReadFile(dstPlugin)
	if err != nil {
		t.Fatalf("kilo plugin not written: %v", err)
	}
	if string(got) != "FAKE-GRYPH-PLUGIN" {
		t.Fatalf("plugin content mismatch: %q", got)
	}
}

func TestInstallGryphKiloCopySourceMissingWarns(t *testing.T) {
	fakeGryph(t)
	out := captureGryph(t, func() {
		installGryph(t.TempDir(), []string{"kilo"}, false)
	})
	if !strings.Contains(out, "kilo copy skipped") {
		t.Fatalf("expected kilo copy skip warning, got:\n%s", out)
	}
	if strings.Contains(out, "gryph plugin copied to") {
		t.Fatalf("must not report a copy:\n%s", out)
	}
}

func TestInstallGryphAgentSelectionGating(t *testing.T) {
	home, _ := fakeGryph(t)
	srcPlugin := filepath.Join(home, ".config", "opencode", "plugins", "gryph.js")
	if err := os.MkdirAll(filepath.Dir(srcPlugin), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPlugin, []byte("FAKE-GRYPH-PLUGIN"), 0644); err != nil {
		t.Fatal(err)
	}

	// opencode selected, kilo not: no kilo copy must happen.
	out := captureGryph(t, func() {
		installGryph(t.TempDir(), []string{"opencode"}, false)
	})
	if strings.Contains(out, "gryph plugin copied to") {
		t.Fatalf("kilo copy must be skipped when kilo not selected:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "kilo", "plugins", "gryph.js")); !os.IsNotExist(err) {
		t.Fatalf("kilo plugin must not exist, stat err=%v", err)
	}
}
```

Notes on the tests:
- `mustExpandHomePath("~")` calls `os.UserHomeDir()`, which on macOS honors `$HOME` — `t.Setenv("HOME", ...)` makes paths land in the temp dir.
- With `HOME` temp, `~/.skillgrid/npm/bin/gryph` does not exist, so `installGryph` falls back to bare `gryph`, resolved via the test PATH — the fake.
- No test asserts `policy` lines yet (that is Task 2).

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./cmd/ -run TestInstallGryph -v`
Expected: build failure `undefined: installGryph`.

- [ ] **Step 4: Implement `installGryph` (hooks only)**

Append to `skillgrid-cli/cmd/steps.go` (after `agentConfigPrefix`):

```go
// installGryph installs gryph (agent audit trail) hooks for the selected
// agents. OpenCode gets gryph's native plugin via `gryph install --agent
// opencode`; Kilo gets the same plugin file copied to
// ~/.config/kilo/plugins/gryph.js (Kilo has no gryph adapter; pattern from
// the engram bridge above). The gryph plugin calls the `gryph` binary from
// PATH at agent runtime.
func installGryph(baseDir string, agents []string, dryRun bool) {
	gryphBin := filepath.Join(mustExpandHomePath("~/.skillgrid/npm"), "bin", "gryph")
	if _, err := os.Stat(gryphBin); err != nil {
		gryphBin = "gryph"
	}
	if hasAgent(agents, "opencode") {
		if dryRun {
			logging.Info("[dry-run] " + gryphBin + " install --agent opencode")
		} else if err := exec.Command(gryphBin, "install", "--agent", "opencode").Run(); err != nil {
			logging.Warn("gryph install failed: " + err.Error())
		} else {
			logging.Info("gryph hooks installed for opencode")
		}
	}
	if hasAgent(agents, "kilo") {
		cfgHome := mustExpandHomePath("~/.config")
		srcPlugin := filepath.Join(cfgHome, "opencode", "plugins", "gryph.js")
		dstPlugin := filepath.Join(cfgHome, "kilo", "plugins", "gryph.js")
		if dryRun {
			if _, err := os.Stat(srcPlugin); err == nil {
				logging.Info("[dry-run] cp " + srcPlugin + " " + dstPlugin)
			} else {
				logging.Warn("gryph: kilo copy skipped, missing " + srcPlugin + " (install with opencode first)")
			}
		} else if _, err := os.Stat(dstPlugin); os.IsNotExist(err) {
			data, err := os.ReadFile(srcPlugin)
			if err != nil {
				logging.Warn("gryph: kilo copy skipped, missing " + srcPlugin + " (install with opencode first)")
			} else if err := os.MkdirAll(filepath.Dir(dstPlugin), 0755); err != nil {
				logging.Warn("gryph: " + err.Error())
			} else if err := os.WriteFile(dstPlugin, data, 0644); err != nil {
				logging.Warn("gryph: " + err.Error())
			} else {
				logging.Info("gryph plugin copied to " + dstPlugin)
			}
		}
	}
}
```

- [ ] **Step 5: Wire the gated step into the install pipeline**

In `skillgrid-cli/cmd/install.go`, immediately after the agent-browser block (which ends `}` at line 96) and before `// Step 5: install plugins`, insert:

```go
	// Step 4c: install gryph audit hooks if present in tools.yaml
	if hasTool(tools.Tools, "gryph") {
		installGryph(baseDir, agents, dryRun)
	}
```

(`tools` is the already-loaded `config.LoadToolsYAML` result from step 4; no new imports needed in either file.)

- [ ] **Step 6: Run tests to verify they pass**

Run: `go build ./... && go test ./cmd/ -run TestInstallGryph -v`
Expected: all PASS.

- [ ] **Step 7: Run full test suite**

Run: `go test ./...`
Expected: PASS (no regressions).

- [ ] **Step 8: Commit**

Run (repo root):

```bash
git add config.d/tools.yaml skillgrid-cli/cmd/install.go skillgrid-cli/cmd/steps.go skillgrid-cli/cmd/gryph_test.go
git commit -m "feat: install gryph audit hooks for kilo and opencode"
```

---

### Task 2: Gryph policy scaffolding and enablement

Adds the policy block to `installGryph` (from the approved design: "Also ship a default block/warn policy… and enable policy.enabled"). Policy state lives in gryph's own global config dir; `gryph policy init` refuses to overwrite an existing policy, so repeated installs keep the user's policy and only re-verify + re-enable.

**Files:**
- Modify: `skillgrid-cli/cmd/steps.go` — `installGryph` (insert before its final `}`)
- Test: `skillgrid-cli/cmd/gryph_test.go`

**Interfaces:**
- Consumes: `installGryph` from Task 1; `gryphBin` resolution already inside the function.
- Produces: same `installGryph` signature; new observable log lines (listed in tests).

- [ ] **Step 1: Write the failing tests**

Append to `skillgrid-cli/cmd/gryph_test.go`:

```go
func TestInstallGryphDryRunEmitsPolicySteps(t *testing.T) {
	out := captureGryph(t, func() {
		installGryph(t.TempDir(), []string{"kilo", "opencode"}, true)
	})
	for _, want := range []string{
		"[dry-run] gryph policy init",
		"[dry-run] gryph policy validate",
		"[dry-run] gryph config set policy.enabled true",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, out)
		}
	}
}

func TestInstallGryphRealRunInvokesPolicySequence(t *testing.T) {
	_, argLog := fakeGryph(t)

	captureGryph(t, func() {
		installGryph(t.TempDir(), []string{"kilo", "opencode"}, false)
	})

	data, err := os.ReadFile(argLog)
	if err != nil {
		t.Fatalf("fake gryph was never invoked: %v (log: %s)", err, argLog)
	}
	got := string(data)
	for _, want := range []string{
		"install --agent opencode",
		"policy init",
		"policy validate",
		"config set policy.enabled true",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("gryph args missing %q:\n%s", want, got)
		}
	}
	iInstall, iInit, iSet := strings.Index(got, "install --agent"), strings.Index(got, "policy init"), strings.Index(got, "config set policy.enabled true")
	if iInstall > iInit || iInit > iSet {
		t.Fatalf("policy sequence out of order:\n%s", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/ -run TestInstallGryph -v`
Expected: the two new tests FAIL (missing policy lines / `policy init` not in args log); the two Task-1 tests still PASS.

- [ ] **Step 3: Implement the policy block**

In `installGryph` (`skillgrid-cli/cmd/steps.go`), insert before the closing `}` of the function:

```go
	if dryRun {
		logging.Info("[dry-run] " + gryphBin + " policy init")
		logging.Info("[dry-run] " + gryphBin + " policy validate")
		logging.Info("[dry-run] " + gryphBin + " config set policy.enabled true")
		return
	}
	if err := exec.Command(gryphBin, "policy", "init").Run(); err != nil {
		logging.Warn("gryph policy init failed: " + err.Error())
	}
	if err := exec.Command(gryphBin, "policy", "validate").Run(); err != nil {
		logging.Warn("gryph policy validate failed: " + err.Error())
	}
	if err := exec.Command(gryphBin, "config", "set", "policy.enabled", "true").Run(); err != nil {
		logging.Warn("gryph policy enable failed: " + err.Error())
	} else {
		logging.Info("gryph policy enabled")
	}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/ -run TestInstallGryph -v`
Expected: all PASS (4 tests).

- [ ] **Step 5: Run full test suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

Run (repo root):

```bash
git add skillgrid-cli/cmd/steps.go skillgrid-cli/cmd/gryph_test.go
git commit -m "feat: scaffold and enable gryph security policy"
```

---

### Task 3: Documentation

**Files:**
- Modify: `docs/02-usage.md` (new section after `## Logs`, before `## Safety Model`)
- Modify: `docs/03-config-reference.md:23-34` (tools.yaml example + semantics)
- Modify: `docs/NOTE.md:65` (mark gryph done)

**Interfaces:** none — prose only. Verify wording against the actual log lines implemented in Tasks 1-2.

- [ ] **Step 1: `docs/02-usage.md` — add Gryph section**

Insert between the `## Logs` section and `## Safety Model`:

```md
## Gryph (Agent Audit Trail)

When `@safedep/gryph` is in `config.d/tools.yaml`, the installer additionally:

- installs gryph's hooks for OpenCode (`gryph install --agent opencode`)
- copies the resulting `~/.config/opencode/plugins/gryph.js` to `~/.config/kilo/plugins/gryph.js` for Kilo
- scaffolds gryph's security policy once, validates it, and enables it (`policy.enabled true`)

The hooks record every tool call (reads, writes, commands) in a local SQLite DB — no content of sensitive files is stored. Query it:

```bash
gryph logs                              # last 24h, filterable by --agent
gryph query --action exec --since "1w"  # shell commands agents ran
gryph stats                             # interactive stats TUI
```

Policy: `gryph policy edit` to change rules, `gryph policy validate` to check. Remove everything with `gryph uninstall`.
```

(Note: the inner ```bash fence is inside the section; keep normal markdown nesting — one bash fence.)

- [ ] **Step 2: `docs/03-config-reference.md` — tools.yaml semantics**

In the `tools` semantics bullets (around line 33), change:

```md
- `tools` — general CLI tools the agent will use (skills, playwright, agent-browser).
```

to:

```md
- `tools` — general CLI tools the agent will use (skills, playwright, agent-browser).
- `@safedep/gryph` is special-cased: beyond the npm install it triggers `installGryph` (hooks for OpenCode + copied to Kilo, then policy init/validate/enable). See `docs/02-usage.md` → Gryph.
```

- [ ] **Step 3: `docs/NOTE.md` — mark done**

Change line 65:

```md
* [ ] [gryph](https://github.com/safedep/gryph)
```

to:

```md
* [X] [gryph](https://github.com/safedep/gryph)
```

- [ ] **Step 4: Verify**

Run (repo root): `grep -n gryph docs/02-usage.md docs/03-config-reference.md docs/NOTE.md`
Expected: hits in all three files.

- [ ] **Step 5: Commit**

Run (repo root):

```bash
git add docs/02-usage.md docs/03-config-reference.md docs/NOTE.md
git commit -m "docs: document gryph audit trail integration"
```

---

## Self-Review

- **Spec coverage:** all 9 requirements of the spec map to tasks: R1 gate (Task 1 Step 5), R2 binary resolution (Task 1 Step 4), R3 opencode hook (Task 1 Step 4), R4 kilo copy incl. warn path (Task 1 Steps 4 + tests), R5 policy sequence (Task 2 Step 3), R6 dry-run (both tasks), R7 agent gating (Task 1 tests `TestInstallGryphAgentSelectionGating` + `hasAgent` guards), R8 no-uninstall (out of scope, noted in spec), R9 docs (Task 3). Error-handling table covered by the warn/continue implementation.
- **Placeholder scan:** no TBD/TODO; every code step shows full code; tests show expected failure modes.
- **Type consistency:** `installGryph(baseDir string, agents []string, dryRun bool)` used identically in install.go, steps.go, all tests; helper names `fakeGryph`/`captureGryph` consistent across tasks.

## Execution Notes (for the executor)

- `config.d/tools.yaml` may already contain the gryph line from a prior session (Task 1 Step 1 handles both cases).
- The machine may already have `~/.skillgrid/npm/bin/gryph` absent — fine; tests never depend on the real binary (they isolate `HOME` and shadow PATH).
- If `go test` picks up a stale `HOME`-dependent state in another test, tests here set `HOME` per-test via `t.Setenv` (auto-restored).
