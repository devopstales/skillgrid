# add-mcp integration — implementation plan

> **For agentic workers:** Use superpowers:subagent-driven-development or executing-plans. Checkbox syntax for tracking.

**Goal:** Use [add-mcp](https://github.com/neon-solutions/add-mcp) to install MCP servers from `config.d/mcp.yaml` into selected agents' global configs, replacing hand-rolled Go JSONC merge.

**Spec:** [2026-08-26-add-mcp-integration-design.md](../specs/2026-08-26-add-mcp-integration-design.md)

**Architecture:** `mcp.yaml` → `scripts/sync-mcp-from-yaml.mjs` → add-mcp `upsertServer()` → agent-native global configs. Go install step orchestrates backup, dry-run, PrecheckDependencies, and invokes the script.

**Tech Stack:** Go (skillgrid-cli), Node (add-mcp API), YAML (`config.d/mcp.yaml`).

---

## Global constraints

- `config.d/mcp.yaml` remains source of truth — no MCP URLs in Go code.
- Install `add-mcp` via `config.d/tools.yaml` → `npm install --prefix ~/.skillgrid/npm` (same as skills, playwright).
- **No runtime `npx`** — Go runs `node scripts/sync-mcp-from-yaml.mjs`; operators use `~/.skillgrid/npm/node_modules/.bin/add-mcp` (or `add-mcp` after PATH).
- `--dry-run` must not call `upsertServer` or write agent configs.
- Per-server failures: `logging.Warn`, never fatal (existing convention).
- Back up each agent config before sync (keep last 10).
- Verification bar: `go build ./... && go test ./...` in `skillgrid-cli/`.
- Pin add-mcp version in `tools.yaml` after spike (avoid `@latest` drift in production doc; use explicit version once validated).
- Working dir for Go tasks: `skillgrid-cli/`.

---

## File structure

| File | Change | Responsibility |
|------|--------|----------------|
| `config.d/tools.yaml` | Modify | Add `add-mcp` package |
| `config.d/mcp.yaml` | Modify (optional) | Extended fields documented; `agents` comment |
| `scripts/sync-mcp-from-yaml.mjs` | Create | YAML → upsertServer bridge |
| `scripts/sync-mcp-from-yaml.test.mjs` | Create | Unit tests with mocked upsert |
| `skillgrid-cli/cmd/install.go` | Modify | Replace MergeMCP loop with `installMCPViaAddMCP` |
| `skillgrid-cli/cmd/steps.go` | Modify | `installMCPViaAddMCP`, agent ID map |
| `skillgrid-cli/cmd/mcp_addmcp_test.go` | Create | dry-run + fake script tests |
| `skillgrid-cli/internal/config/merger.go` | Deprecate/remove | After parity Task 6 |
| `docs/04-mcp-servers.md` | Modify | add-mcp workflows, operator commands |
| `docs/03-config-reference.md` | Modify | Extended mcp.yaml fields |
| `docs/01-installation.md` | Modify | Step 7 uses add-mcp |
| `docs/02-usage.md` | Modify | Day-loop: find/list/sync via add-mcp |

---

### Task 1: Spike — add-mcp on Kilo + OpenCode

**Purpose:** Validate JSONC handling, server shapes, and programmatic API before wiring install.

- [ ] **Step 1:** Add `add-mcp` to local npm prefix: `npm install add-mcp --prefix ~/.skillgrid/npm --cache ~/.skillgrid/npm/cache` (not `npx`).
- [ ] **Step 2:** Manual upsert via Node REPL or one-off `node -e "…"` importing from `~/.skillgrid/npm/node_modules/add-mcp` — one remote (`context7`) and one local (`engram` or fake stdio) to `kilo-code` and `opencode` globally.
- [ ] **Step 3:** Record whether JSONC comments survive in `kilo.jsonc` / `opencode.jsonc`.
- [ ] **Step 4:** Document findings in spec **Risks** section (update STATUS if comment loss accepted).

**Verify:** Kilo and OpenCode load servers; note any manual enable toggle required.

---

### Task 2: Install add-mcp via tools.yaml

**Files:**
- Modify: `config.d/tools.yaml`

- [ ] **Step 1:** Add `add-mcp` to `tools:` list (pin version after Task 1).
- [ ] **Step 2:** Re-run `skillgrid install --dry-run` — confirm npm plan includes add-mcp.
- [ ] **Step 3:** Real install; confirm binary at `~/.skillgrid/npm/node_modules/.bin/add-mcp`.

**Verify:** `~/.skillgrid/npm/node_modules/.bin/add-mcp list -g -a kilo-code` runs.

---

### Task 3: sync-mcp-from-yaml.mjs

**Files:**
- Create: `scripts/sync-mcp-from-yaml.mjs`
- Create: `scripts/fixtures/mcp.min.yaml` (test fixture)

- [ ] **Step 1:** Parse YAML `servers` — support `type: remote|local`, `url`, `command`, optional `transport`, `headers`, `env`.
- [ ] **Step 2:** Map local `command[]` → add-mcp `{ command, args }` shape.
- [ ] **Step 3:** Accept `--agents` comma list and `--dry-run`.
- [ ] **Step 4:** Loop: `upsertServer(agent, name, config, { global: true })`; log results.
- [ ] **Step 5:** Resolve add-mcp via `createRequire` from `SKILLGRID_NPM_PREFIX/node_modules/add-mcp` — never `npx` or dynamic download.

**Verify:**

```bash
node scripts/sync-mcp-from-yaml.mjs \
  --yaml config.d/mcp.yaml \
  --agents kilo-code,opencode \
  --dry-run
```

---

### Task 4: Go install step

**Files:**
- Modify: `skillgrid-cli/cmd/install.go` (step 7)
- Modify: `skillgrid-cli/cmd/steps.go`
- Create: `skillgrid-cli/cmd/mcp_addmcp_test.go`

- [ ] **Step 1:** Add `skillgridAgentToAddMCP map[string]string` (`kilo` → `kilo-code`, `opencode` → `opencode`).
- [ ] **Step 2:** Implement `installMCPViaAddMCP(baseDir, agents, dryRun)`:
  - Load registry via existing `mcp.LoadRegistry`
  - `PrecheckDependencies` → warn missing
  - Backup each selected agent config (reuse existing backup helper)
  - Exec `node` (not `npx`) on `scripts/sync-mcp-from-yaml.mjs` with `SKILLGRID_NPM_PREFIX` env
- [ ] **Step 3:** Replace `config.MergeMCP` loop in install.go with new function.
- [ ] **Step 4:** dry-run test: no file writes; log contains server names.

**Verify:** `go test ./cmd/... -run AddMCP`

---

### Task 5: Extended mcp.yaml + docs

**Files:**
- Modify: `docs/04-mcp-servers.md`
- Modify: `docs/03-config-reference.md`
- Modify: `docs/02-usage.md`

- [ ] **Step 1:** Document extended YAML fields (`transport`, `headers`, `env`, `auto_approve`).
- [ ] **Step 2:** Add operator table: `find`, `list`, `sync`, `remove` via managed `add-mcp` on PATH.
- [ ] **Step 3:** Clarify managed vs ad-hoc: YAML + install = managed; `add-mcp find` = ad-hoc (after PATH setup).
- [ ] **Step 4:** Link to [add-mcp registry](https://add-mcp.com) and [GitHub](https://github.com/neon-solutions/add-mcp).

**Verify:** Docs match design spec agent mapping table.

---

### Task 6: Remove legacy MergeMCP

**Files:**
- Remove or deprecate: `skillgrid-cli/internal/config/merger.go` MCP functions
- Update: `merger_test.go`, `dryrun_test.go`, `smoke_test.go`

- [ ] **Step 1:** Confirm integration test passes with sync script only.
- [ ] **Step 2:** Delete `MergeMCP` / `buildServerEntry` or keep thin wrapper calling script (prefer delete).
- [ ] **Step 3:** Update tests to cover install step + script contract.

**Verify:** `go test ./...` green; no references to `MergeMCP`.

---

### Task 7: Integration smoke

- [ ] **Step 1:** Fresh install with `--sync-repo $(pwd) --yes`.
- [ ] **Step 2:** Assert all `mcp.yaml` server names in `~/.config/kilo/kilo.jsonc` and opencode config.
- [ ] **Step 3:** Run `~/.skillgrid/npm/node_modules/.bin/add-mcp list -g -a kilo-code` — output matches YAML set.
- [ ] **Step 4:** `add-mcp sync -g -y` — no conflicting header warnings on managed servers.

**Verify:** Manual checklist in PR description.

---

## Agent rollout

| Phase | Agents | Notes |
|-------|--------|-------|
| v1 | kilo, opencode | Map to `kilo-code`, `opencode` |
| v2 | cursor, codex, claude | Extend map when config paths land in skillgrid selector |
| v2+ | `add-mcp --all` | Document only; skillgrid stays selective |

---

## Migration notes (existing users)

1. Re-run `skillgrid install` — sync script upserts YAML servers (idempotent).
2. Servers added only by hand remain unless name collides with YAML entry.
3. Old `name-oldtype` rename entries from MergeMCP are **not** auto-migrated — one-time manual cleanup if present.
4. Rollback: restore from `~/.skillgrid/backups/` as today.

---

## Open questions

1. **Comment preservation spike** — block Task 6 if kilo.jsonc comments are stripped and team rejects rewrite.
2. **Version pin** — which add-mcp semver after Task 1 (track [releases](https://github.com/neon-solutions/add-mcp/releases)).
3. **Project-scoped MCP** — defer; skillgrid v1 is global-only. Projects can use managed `add-mcp` without `-g` separately.

---

## Follow-up: mcp.yaml stdio servers using npx

Current `gitnexus` entry uses `npx -y gitnexus@… mcp` in `command`. Align with no-npx policy:

- [ ] Add `gitnexus` to `tools.yaml` (pinned version)
- [ ] Change `mcp.yaml` command to `[gitnexus, mcp]` (binary on PATH via skillgrid npm prefix)

---

## Manual acceptance

After Task 7:

1. Edit `config.d/mcp.yaml` — add a test remote server entry.
2. `./bin/skillgrid install --sync-repo $(pwd) --verbose`
3. Confirm new server in Kilo MCP settings.
4. `add-mcp find context7 -a kilo-code -g -y` — ad-hoc path (managed binary, not npx) alongside managed YAML.
