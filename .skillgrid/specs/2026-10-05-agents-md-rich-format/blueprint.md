# Rich AGENTS.md Preamble (kubedash_4 format) — Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread (a greenfield init renders one rich section first, then thickens to the full structure; the reference file `/Users/paladm/git/kubedash_4/AGENTS.md` is the end-to-end target)

**Goal:** Make `skillgrid init` render a rich, kubedash_4-shaped AGENTS.md preamble — `Environment & Tooling`, `Key Directories`, `Security & Escalation Boundaries`, `Dependency Policies`, `Architecture Constraints`, `Definition of Done`, plus reference-pointer sections — instead of the current 3-line `## Commands` skeleton.

**Architecture:** A new fenced preamble template (`agents-preamble.md`) is the single source of truth for the structure; the CLI (`init_boot.go`) renders it from existing `.skillgrid/config.yaml` fields (commands, testing, security) plus new optional `agents:` fields, filling detected values into placeholders. Onboarding only fills the new config fields + re-runs `skillgrid init`; it never hand-writes the preamble. The existing `## Skillgrid` sentinel block is untouched.

**Tech Stack:** Go 1.25 (skillgrid-cli), `gopkg.in/yaml.v3` (already imported), existing `skillgrid init` pipeline.

**Spec:** `/Users/paladm/git/kubedash_4/AGENTS.md` (the reference format/content this generator must reproduce verbatim in structure).

## Terms

- **Preamble** — the region between `<!-- skillgrid-preamble:start -->` and `<!-- skillgrid-preamble:end -->` in AGENTS.md. Owned by `skillgrid init`.
- **Sentinel block** — the `## Skillgrid` region between `<!-- skillgrid:start -->` and `<!-- skillgrid:end -->`. Owned by onboarding (rendered from `block.md`). Not touched by this change.
- **Reference pointer section** — a heading whose body is a single `reference `<path>`` line (shared standards are referenced, never inlined). The reference's `Issue Tracker` / `Memory` / `Code Indexing` / `Testing standards` / `SDD Standards` are this shape.
- None beyond the spec's.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- Given a temp dir with `.skillgrid/config.yaml` (commands + testing + security filled) and no AGENTS.md, `skillgrid init` writes a preamble containing `## Environment & Tooling`, `## Security & Escalation Boundaries`, `## Dependency Policies`, `## Architecture Constraints`, `## Definition of Done`, and the reference-pointer sections (`## Issue Tracker`, `## Memory`, `## Code Indexing`, `## Testing standards`, `## SDD Standards`).
- The `## Environment & Tooling` bullets carry the **actual** commands from config (e.g. `Test: <testing.runner>`), not `<detect>` placeholders, when those fields are present.
- An empty/missing config field renders an `<detect>` placeholder for that bullet, and the section still renders (init does not fail on an empty preamble field).
- `Key Directories` renders only when `agents.directories` is non-empty; otherwise the section is omitted.
- On a second `init` run the preamble is NOT rewritten unless `--force` is passed (merge-safe), and an onboarding-rendered sentinel block is preserved verbatim (existing contract).
- The sentinel block still appears exactly once; the preamble still appears exactly once.
- `skillgrid init` still ingests docs, writes `config.d/indexing.yaml`, and runs the code index (regression: existing `init` behavior intact).
- `TestInitPreambleIsMinimalCommandsFirst` is updated so it no longer asserts the ABSENCE of the rich sections — it instead asserts their PRESENCE and that no `<detect>` placeholder remains when config values exist.
- `block.md`'s `## Skillgrid` block and its Artifacts table (4 rows) are unchanged (regression: `TestInitSentinelTemplateComesFromBlockMD` still green).

**Artifacts** (files that must exist with real implementation, not stubs):
- `.agents/skills/lifecycle/onboarding/templates/agents-preamble.md` — fenced preamble template with placeholders (mirrors the kubedash_4 section order).
- `skillgrid-cli/cmd/skillgrid/init_preamble.go` — renders the preamble from config + template (load/parse/render, with `<detect>` fallbacks).
- `skillgrid-cli/cmd/skillgrid/init_boot.go` (modified) — `writeBootFile` calls the new preamble renderer; `upsertPreamble` operates on the rendered region.
- `skillgrid-cli/cmd/skillgrid/init_preamble_test.go` — tests for config→preamble rendering, `<detect>` fallback, `Key Directories` conditional, idempotence, sentinel preservation, regression (sentinel block + index intact).
- `.agents/skills/lifecycle/onboarding/templates/config.yaml` (modified) — new optional `agents:` block (directories, security boundaries, dependency policies, architecture constraints, definition of done, engineering standards) with commented examples.
- `.agents/skills/lifecycle/onboarding/SKILL.md` (modified) — onboarding detects the new `agents.*` facts; Step 3 writes them to config; Step 4 verifies the rendered preamble; "Keep the AGENTS.md block lean" reconciled with "preamble sections are repo-specific, not the ## Skillgrid block".

**Key links** (critical connections between artifacts that must work together):
- `writeBootFile` (init_boot.go) must call the preamble renderer with the parsed config before `upsertPreamble`, and pass the rendered body as the region.
- The renderer must read the template from the same resolved skill tree that `loadSentinelTemplate` already uses (project tree → `setup.FindRepoRoot`), so tests can point at the repo's real `agents-preamble.md`.
- Config `agents.directories` (a list of `{path, purpose}`) must map 1:1 to `Key Directories` table rows.
- The onboarding SKILL.md must direct the agent to fill `agents.*` in config and then run `skillgrid init` (not hand-write the preamble), keeping the template as the single source of truth.

**One-way-door decisions:**
- **Generated-file contract for onboarded projects** — changing the preamble structure means every project's AGENTS.md preamble changes shape on the next `skillgrid init --force` (or re-onboarding). Existing AGENTS.md files keep their old preamble until `--force` (merge-safe), so this is reversible per-project, but the *template* is now the contract. Flag for approval before Task 5 (wiring `writeBootFile` to the new renderer), since it changes what `init` writes to a real repo.

## Global Constraints

- The `## Skillgrid` sentinel block is **not** modified by this change — its structure stays owned by `block.md` (4-row Artifacts table). Per the user's "All references" decision, the new reference-pointer sections in the preamble use **real installed paths** (existing `~/.agents/skills/_shared/rules/mnemonic-memory.md`, `.../sdd-structure.md`, `.../ticketing/`), not the reference's stale `mnemonic-artifacts.md` / `mnemonic-code-indexing.md` / `testing-standards.md` paths (those files do not exist in the installed tree — verified).
- The preamble stays merge-safe: `upsertPreamble` keeps the existing region when no sentinel and no `--force`; user text outside the regions is never clobbered.
- No new external Go dependency beyond `gopkg.in/yaml.v3` (already imported in init_boot.go).
- Onboarding SKILL.md already requires "reference, don't write" for the `## Skillgrid` block; that rule is **extended** (not reversed) to say the *preamble* sections are repo-specific inline, while *shared standards* remain reference pointers. The lean-block rationale for the sentinel block is unchanged.
- Tests run with `go test ./...` from `skillgrid-cli/`. The repo's known flakes (TTL boundary, parallel-search ceiling, TestExecuteGoSkill, TestSkillSearchHybridDegraded timeout) are out of scope; only `TestInit*` and `TestOnboardingSkill*` must be green.
- Conventional commits, no AI-attribution trailer (work-unit-commits).

## File Structure

- `.agents/skills/lifecycle/onboarding/templates/agents-preamble.md` — single source of truth for the preamble structure (fenced, with `{placeholders}`). Created by Task 1.
- `skillgrid-cli/cmd/skillgrid/init_preamble.go` — `loadAgentsPreambleTemplate`, `parseAgentsConfig`, `renderPreamble`. One file, one responsibility: turn config → rendered preamble. Created by Task 2.
- `skillgrid-cli/cmd/skillgrid/init_boot.go` — modified: `writeBootFile` uses the rendered preamble; `preambleBody`/`commandSkeleton` removed or superseded. Task 5.
- `skillgrid-cli/cmd/skillgrid/init_preamble_test.go` — all new preamble tests. Tasks 2, 4, 6.
- `.agents/skills/lifecycle/onboarding/templates/config.yaml` — adds `agents:` block. Task 3.
- `.agents/skills/lifecycle/onboarding/SKILL.md` — onboarding detects + writes `agents.*`, re-runs `skillgrid init`, reconciles the lean guidance. Task 4.

---

### Task 1: Preamble template (single source of truth)

**Files:**
- Create: `.agents/skills/lifecycle/onboarding/templates/agents-preamble.md`
- Test: `skillgrid-cli/cmd/skillgrid/init_preamble_test.go`

**Interfaces:**
- Consumes: the kubedash_4 reference structure (`/Users/paladm/git/kubedash_4/AGENTS.md`).
- Produces: a fenced markdown template with named placeholders: `{language}`, `{package_manager}`, `{install}`, `{test}`, `{lint}`, `{format}`, `{build}`, `{run}`, `{docker}`, `{security_scan}`, `{directories_table}`, `{engineering_standards}`, `{security_boundaries}`, `{dependency_policies}`, `{architecture_constraints}`, `{definition_of_done}`, `{issue_tracker_line}`, `{memory_line}`, `{code_indexing_line}`, `{testing_standards_line}`, `{sdd_standards_line}`.
- Seam: template file read by `loadAgentsPreambleTemplate` (added in Task 2).
- Deletion test: if the template is deleted, the CLI has no structure to render — the whole preamble feature disappears.
- Adapters: 2 — prod (skill tree) and test (repo's real template via a var override, mirroring `blockMDPath`).

**SATISFIES:** `preamble-template-exists`

- [ ] **Step 1: Write the failing test**

```go
// init_preamble_test.go
func TestLoadAgentsPreambleTemplate(t *testing.T) {
	prev := agentsPreamblePath
	t.Cleanup(func() { agentsPreamblePath = prev })
	agentsPreamblePath = func(dir string) string {
		return "../../../.agents/skills/lifecycle/onboarding/templates/agents-preamble.md"
	}
	tmpl, _, err := loadAgentsPreambleTemplate(t.TempDir())
	if err != nil {
		t.Fatalf("loadAgentsPreambleTemplate: %v", err)
	}
	for _, want := range []string{"## Environment & Tooling", "## Key Directories",
		"## Security & Escalation Boundaries", "## Dependency Policies",
		"## Architecture Constraints", "## Definition of Done",
		"## Issue Tracker", "## Memory", "## Code Indexing",
		"## Testing standards", "## SDD Standards"} {
		if !strings.Contains(tmpl, want) {
			t.Fatalf("template missing section %q\n%s", want, tmpl)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run TestLoadAgentsPreambleTemplate -v`
Expected: FAIL — `agentsPreamblePath` / `loadAgentsPreambleTemplate` undefined.

- [ ] **Step 3: Write the template file**

`.agents/skills/lifecycle/onboarding/templates/agents-preamble.md` — the fenced block mirrors kubedash_4 verbatim in section order/headings, with `{placeholders}`. Example skeleton (fill the real kubedash_4 headings/bullet labels; shared-standards pointers use real installed paths):

````markdown
<!-- skillgrid-preamble:start -->
# Standards

## Environment & Tooling

- **Language**: {language}
- **Package manager**: {package_manager}
- **Task runner**: {task_runner}
- **Install**: {install}
- **Test**: {test}
- **Lint**: {lint}
- **Format**: {format}
- **Build (dev)**: {build}
- **Run (dev)**: {run}
- **Docker (dev deps)**: {docker}
- **Security scan**: {security_scan}

## Key Directories

{directories_table}

## Engineering Standards

{engineering_standards}

## Security & Escalation Boundaries

{security_boundaries}

## Dependency Policies

{dependency_policies}

## Architecture Constraints

{architecture_constraints}

## Definition of Done

{definition_of_done}

## Issue Tracker

{issue_tracker_line}

## Memory

{memory_line}

## Code Indexing

{code_indexing_line}

## Testing standards

{testing_standards_line}

## SDD Standards

{sdd_standards_line}
<!-- skillgrid-preamble:end -->
````

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run TestLoadAgentsPreambleTemplate -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add .agents/skills/lifecycle/onboarding/templates/agents-preamble.md skillgrid-cli/cmd/skillgrid/init_preamble_test.go
git commit -m "feat: add agents-preamble template (kubedash_4 format)"
```

### Task 2: Config parsing + renderPreamble (pure functions)

**Files:**
- Create: `skillgrid-cli/cmd/skillgrid/init_preamble.go`
- Test: `skillgrid-cli/cmd/skillgrid/init_preamble_test.go`

**Interfaces:**
- Consumes: `.skillgrid/config.yaml` (`commands.*`, `testing.*`, `security.trivy`, and the new `agents.*` from Task 3); the template from Task 1.
- Produces:
  - `type agentsConfig struct` — yaml-tagged: `Commands{Build,Lint,Format,Typecheck}`, `Testing{Runner,Setup,Layers,Coverage,Mutation}`, `Security{Trivy{Command,Severities,ScanTypes}}`, `Agents{Directories []DirEntry, SecurityBoundaries []string, DependencyPolicies []string, ArchitectureConstraints []string, DefinitionOfDone []string, EngineeringStandards []string}`; `DirEntry{Path,Purpose string}`.
  - `func loadAgentsConfig(dir string) (agentsConfig, error)` — reads + unmarshals `.skillgrid/config.yaml` (mirrors `loadProjectName`'s yaml read; returns a zero struct on a missing file, not an error).
  - `func renderPreamble(cfg agentsConfig, template string) string` — fills `{placeholders}` from cfg; any empty value → `<detect>` (for command bullets) or the section is dropped (for `Key Directories` when `directories` empty).
  - `var agentsPreamblePath func(dir string) string` — default resolves `dir/.agents/skills/lifecycle/onboarding/templates/agents-preamble.md`, then `setup.FindRepoRoot(dir)` + rel path (mirrors `blockMDPath`).
  - `func loadAgentsPreambleTemplate(dir string) (template, path string, err error)`.
- Seam: `agentsPreamblePath` (test-overridable, like `blockMDPath`).
- Deletion test: delete the file → `writeBootFile` (Task 5) has no renderer.
- Adapters: 2 (prod skill tree, test repo path).

**SATISFIES:** `preamble-renders-from-config`

- [ ] **Step 1: Write the failing tests**

```go
func TestRenderPreambleFillsConfigCommands(t *testing.T) {
	cfg := agentsConfig{
		Commands: struct{ Build, Lint, Format, Typecheck string }{Build: "task build", Lint: "task lint", Format: "task fmt", Typecheck: "task check"},
		Testing:  struct{ Runner, Setup string; Layers []string; Coverage, Mutation string }{Runner: "./run_test.sh", Setup: "task install"},
		Security: struct{ Trivy struct{ Command, Severities, ScanTypes string } }{Trivy: struct{ Command, Severities, ScanTypes string }{Command: "task security", Severities: "CRITICAL,HIGH", ScanTypes: "vuln,secret"}},
	}
	out := renderPreamble(cfg, "- **Test**: {test}\n- **Lint**: {lint}")
	if !strings.Contains(out, "- **Test**: ./run_test.sh") {
		t.Fatalf("test command not filled:\n%s", out)
	}
	if strings.Contains(out, "<detect>") {
		t.Fatalf("placeholder left in filled output:\n%s", out)
	}
}

func TestRenderPreambleDetectFallback(t *testing.T) {
	cfg := agentsConfig{} // all empty
	out := renderPreamble(cfg, "- **Build**: {build}")
	if !strings.Contains(out, "- **Build**: <detect>") {
		t.Fatalf("empty field did not fall back to <detect>:\n%s", out)
	}
}

func TestRenderPreambleDropsEmptyKeyDirectories(t *testing.T) {
	cfg := agentsConfig{} // no directories
	out := renderPreamble(cfg, "## Key Directories\n\n{directories_table}")
	// the whole Key Directories section is dropped when there are no rows
	if strings.Contains(out, "## Key Directories") {
		t.Fatalf("empty Key Directories section should be dropped:\n%s", out)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run 'TestRenderPreamble' -v`
Expected: FAIL — `agentsConfig`, `renderPreamble` undefined.

- [ ] **Step 3: Write `init_preamble.go`**

Implement `agentsConfig` (yaml tags matching config.yaml keys), `loadAgentsConfig` (read `.skillgrid/config.yaml`, `yaml.Unmarshal` into the struct; on `os.IsNotExist` return zero struct + nil error), `loadAgentsPreambleTemplate` (same candidate-walk as `loadSentinelTemplate`: `agentsPreamblePath(dir)` then `setup.FindRepoRoot(dir)+agentsPreambleRel`; extract the first ``` fence with a regexp like `blockTemplateRe`), and `renderPreamble`:
- Replace each command placeholder with the cfg value or `<detect>` when empty.
- `directories_table`: if `len(cfg.Agents.Directories)==0`, strip the whole `## Key Directories` section (heading + body up to the next `## `); else build a markdown table `| Path | Purpose |` from the entries.
- `security_boundaries`/`dependency_policies`/`architecture_constraints`/`definition_of_done`/`engineering_standards`: join `cfg.Agents.*` with `\n` as `- ` bullets, or a one-line `No <topic> yet — see `.skillgrid/ASSUMPTIONS.md`.` default when empty.
- Reference-pointer lines (`issue_tracker_line`, `memory_line`, `code_indexing_line`, `testing_standards_line`, `sdd_standards_line`): static, real installed paths — `~/.agents/skills/_shared/rules/ticketing/backlogmd.md` (for backlogmd), `~/.agents/skills/_shared/rules/mnemonic-memory.md`, `~/.agents/skills/_shared/rules/sdd-structure.md`. (Use `block.md`'s tracker table for the issue-tracker line to stay consistent with the sentinel block.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run 'TestRenderPreamble|TestLoadAgentsPreambleTemplate' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/cmd/skillgrid/init_preamble.go skillgrid-cli/cmd/skillgrid/init_preamble_test.go
git commit -m "feat: render agents preamble from config"
```

### Task 3: Config template — add optional `agents:` block

**Files:**
- Modify: `.agents/skills/lifecycle/onboarding/templates/config.yaml` (after the `commands:` block, ~line 80)

**Interfaces:**
- Consumes: nothing.
- Produces: a commented `agents:` block in the template so onboarding knows the shape to fill:

```yaml
# --- AGENTS.md preamble (repo-specific, rendered by `skillgrid init`) ---
# Optional. Filled by onboarding from detected facts + the existing codebase
# (brownfield: read the repo, don't guess). Empty values render as <detect>
# placeholders; empty lists drop the section. See templates/agents-preamble.md.
agents:
  directories:            # list of {path, purpose} -> Key Directories table
    # - { path: "src/kubedash/lib/k8s/", purpose: "K8s API wrappers — one file per resource type" }
  security_boundaries: [] # bullets -> Security & Escalation Boundaries
  dependency_policies: [] # bullets -> Dependency Policies
  architecture_constraints: [] # bullets -> Architecture Constraints
  definition_of_done: []  # bullets -> Definition of Done
  engineering_standards: [] # bullets -> Engineering Standards
```

- Seam: none (data file).
- Deletion test: without it, onboarding has no documented shape for the new fields.
- Adapters: n/a.

**SATISFIES:** `config-template-has-agents-block`

- [ ] **Step 1: Write the failing test**

```go
func TestConfigTemplateHasAgentsBlock(t *testing.T) {
	body, err := os.ReadFile("../../../.agents/skills/lifecycle/onboarding/templates/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "agents:") {
		t.Fatalf("config template missing agents: block:\n%s", s)
	}
	for _, key := range []string{"directories:", "security_boundaries:", "dependency_policies:", "architecture_constraints:", "definition_of_done:", "engineering_standards:"} {
		if !strings.Contains(s, key) {
			t.Fatalf("config template missing agents key %q", key)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run TestConfigTemplateHasAgentsBlock -v`
Expected: FAIL — `agents:` not in the template yet.

- [ ] **Step 3: Add the `agents:` block** to the template (content above), placed after `commands:` and before `# --- Conventions ---`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run TestConfigTemplateHasAgentsBlock -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add .agents/skills/lifecycle/onboarding/templates/config.yaml skillgrid-cli/cmd/skillgrid/init_preamble_test.go
git commit -m "feat: add agents block to onboarding config template"
```

### Task 4: Onboarding skill — detect + write `agents.*`, re-run init

**Files:**
- Modify: `.agents/skills/lifecycle/onboarding/SKILL.md`

**Interfaces:**
- Consumes: the new `agents.*` config shape (Task 3), the template (Task 1), `skillgrid init` (existing).
- Produces: onboarding instructions that (a) detect `agents.directories` (walk the top-level tree, one row per significant dir), `security_boundaries` (from README/CI/security docs), `dependency_policies` (version pins from manifests), `architecture_constraints` (from `ASSUMPTIONS.md § Locked constraints` + layering docs), `definition_of_done` (from CI/Taskfile gates), and `engineering_standards` (reversed from code); (b) write them to `.skillgrid/config.yaml`; (c) run `skillgrid init` to render the preamble; (d) update the "Keep the AGENTS.md block lean" section to say the *preamble* sections are repo-specific inline while the `## Skillgrid` block stays lean/references; (e) verify the rendered preamble in Step 4.
- Seam: none (prose).
- Deletion test: without it, the rich sections are never filled on a real project (the template renders `<detect>` forever).
- Adapters: n/a.

**SATISFIES:** `onboarding-fills-agents-fields`

- [ ] **Step 1: Write the failing test**

```go
func TestOnboardingSkillDocumentsAgentsPreamble(t *testing.T) {
	body, err := os.ReadFile(onboardingSkillPath(t))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "agents.directories") && !strings.Contains(s, "agents:") {
		t.Fatalf("onboarding must document the agents.* config fields:\n%s", s)
	}
	if !strings.Contains(s, "agents-preamble") {
		t.Fatalf("onboarding must name the agents-preamble template")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run TestOnboardingSkillDocumentsAgentsPreamble -v`
Expected: FAIL.

- [ ] **Step 3: Edit SKILL.md**
  - Step 1 (Detect): add a "AGENTS.md preamble facts" sub-step — detect `agents.directories`, `security_boundaries`, `dependency_policies`, `architecture_constraints`, `definition_of_done`, `engineering_standards` (brownfield: read the repo + `ASSUMPTIONS.md`; greenfield: leave lists empty).
  - Step 2 (Present & Confirm): add a confirmation item for the new fields.
  - Step 3 (Write): after writing config.yaml, note the `agents:` block is filled and `skillgrid init` renders it.
  - "Keep the AGENTS.md block lean": add a paragraph — the *preamble* sections (`Environment & Tooling`, `Key Directories`, etc.) are **repo-specific inline** (they're the one place AGENTS.md states project facts), while the `## Skillgrid` sentinel block stays a lean navigation spine of reference pointers. The "reference, don't write" rule applies to the sentinel block, not to the repo-specific preamble.
  - Step 4 (Verify): add a checklist item — the rendered AGENTS.md contains the rich sections with no leftover `<detect>` when config was filled.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run TestOnboardingSkillDocumentsAgentsPreamble -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add .agents/skills/lifecycle/onboarding/SKILL.md skillgrid-cli/cmd/skillgrid/init_preamble_test.go
git commit -m "docs: onboarding detects + writes agents.* preamble fields"
```

### Task 5: Wire `writeBootFile` to the new renderer

> **Checkpoint:** ⚠ one-way: this changes what `skillgrid init` writes into a real project's AGENTS.md preamble (structure upgrade from 3-line skeleton to rich sections). Existing files keep their old preamble until `--force`. Confirm before proceeding.

**Files:**
- Modify: `skillgrid-cli/cmd/skillgrid/init_boot.go` (`writeBootFile`, `upsertPreamble`, `preambleBody`, `commandSkeleton`, `projectOverview`)
- Test: `skillgrid-cli/cmd/skillgrid/init_preamble_test.go`

**Interfaces:**
- Consumes: `renderPreamble` + `loadAgentsPreambleTemplate` + `loadAgentsConfig` (Task 2).
- Produces: `writeBootFile` renders the rich preamble into the `<!-- skillgrid-preamble:* -->` region; `upsertPreamble` keeps the region when no `--force` (merge-safe) and replaces it on `--force`; the sentinel block is still appended greenfield-only (unchanged).
- Seam: `upsertPreamble`'s region now holds rendered markdown.
- Deletion test: n/a (wiring).
- Adapters: n/a.

**SATISFIES:** `init-writes-rich-preamble`

- [ ] **Step 1: Write the failing tests**

```go
func TestInitWritesRichPreambleFromConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "project: Demo\ncommands:\n  build: \"task build\"\ntesting:\n  runner: \"./run_test.sh\"\n  setup: \"task install\"\nsecurity:\n  trivy:\n    command: \"task security\"\n    severities: \"CRITICAL\"\n    scan_types: \"vuln\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(res.BootFile)
	s := string(body)
	for _, want := range []string{"## Environment & Tooling", "## Definition of Done", "## Security & Escalation Boundaries"} {
		if !strings.Contains(s, want) {
			t.Fatalf("preamble missing %q:\n%s", want, s)
		}
	}
	if !strings.Contains(s, "- **Test**: ./run_test.sh") {
		t.Fatalf("test command not from config:\n%s", s)
	}
}

func TestInitPreambleIdempotentAndKeepsSentinel(t *testing.T) {
	dir := t.TempDir()
	onboarded := "Onboarding custom row: keep this."
	agents := "<!-- skillgrid:start -->\n## Skillgrid\n\n" + onboarded + "\n<!-- skillgrid:end -->\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(agents), 0o644); err != nil {
		t.Fatal(err)
	}
	res1, _ := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	res2, _ := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	body, _ := os.ReadFile(res2.BootFile)
	s := string(body)
	if !strings.Contains(s, onboarded) {
		t.Fatalf("onboarding sentinel clobbered:\n%s", s)
	}
	if strings.Count(s, "<!-- skillgrid:start -->") != 1 || strings.Count(s, "<!-- skillgrid-preamble:start -->") != 1 {
		t.Fatalf("sentinel/preamble duplicated:\n%s", s)
	}
	_ = res1
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run 'TestInitWritesRichPreamble|TestInitPreambleIdempotentAndKeepsSentinel' -v`
Expected: FAIL — `writeBootFile` still emits the old 3-line skeleton.

- [ ] **Step 3: Wire it in `init_boot.go`**
  - In `writeBootFile`, after `loadProjectName`/`loadRulesBlock`, call `cfg, _ := loadAgentsConfig(dir)` and `tpl, _, _ := loadAgentsPreambleTemplate(dir)`, then `preamble := renderPreamble(cfg, tpl)`.
  - Pass `preamble` to `upsertPreamble` as the region (replace the `projectOverview(project)` + `commandSkeleton` pair). Keep the `<!-- skillgrid-preamble:start/end -->` wrap.
  - `upsertPreamble` logic stays: if the region exists and no `--force` → keep existing; else replace the region. `--force` rebuilds from the rendered preamble.
  - Remove `commandSkeleton` + `preambleBody` (or keep `projectOverview` only if still referenced by a test).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run 'TestInit' -v`
Expected: PASS (including the updated `TestInitPreambleIsMinimalCommandsFirst` from Task 6).

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/cmd/skillgrid/init_boot.go skillgrid-cli/cmd/skillgrid/init_preamble_test.go
git commit -m "feat: init renders the rich agents preamble"
```

### Task 6: Update the minimal-preamble test + full-suite regression

**Files:**
- Modify: `skillgrid-cli/cmd/skillgrid/init_cmd_test.go` (`TestInitPreambleIsMinimalCommandsFirst`)
- Test: full `go test ./...`

**Interfaces:**
- Consumes: the new rendered preamble (Task 5).
- Produces: a corrected test that asserts the rich sections are PRESENT (when config is filled) and that no `<detect>` placeholder survives when config has values — replacing the old "assert absence" logic.
- Seam: none.
- Deletion test: n/a (test).
- Adapters: n/a.

**SATISFIES:** `minimal-preamble-test-updated`

- [ ] **Step 1: Rewrite the test**

Replace `TestInitPreambleIsMinimalCommandsFirst`'s "absence of rich sections" assertions with:
- Assert the preamble contains `## Environment & Tooling`, `## Definition of Done`, etc. (with a filled config, per Task 5's test dir setup).
- Assert no literal `<detect>` remains when the config provided the value (i.e. the section is filled, not skeleton).
- Keep the sentinel-count + no-duplicate assertions.
- Drop the `< 150` line cap if the rich preamble legitimately exceeds it (the reference is ~93 lines; with a filled config the preamble stays well under 150 — keep the cap but raise to 200 to be safe).

- [ ] **Step 2: Run the test to verify it passes**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run TestInitPreambleIsMinimalCommandsFirst -v`
Expected: PASS.

- [ ] **Step 3: Run the full `init` + onboarding test set**

Run: `cd skillgrid-cli && go test ./cmd/skillgrid/ -run 'TestInit|TestOnboardingSkill' -v`
Expected: PASS (all `TestInit*` + `TestOnboardingSkill*` green).

- [ ] **Step 4: Run the full suite (regression)**

Run: `cd skillgrid-cli && go test ./...`
Expected: PASS, except the known pre-existing flakes (TTL boundary, parallel-search ceiling, TestExecuteGoSkill, TestSkillSearchHybridDegraded) which are out of scope — verify they pass in isolation if they appear.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/cmd/skillgrid/init_cmd_test.go
git commit -m "test: update minimal-preamble test for the rich format"
```

## Plan Review

- Verdict: _pending (run after self-review)_
- Findings: _pending_
- Reviewed: 2026-10-05
