# Stage 8 — Intermediate Zoom Out (Round 1, post-revision; fresh subagent)

**Reviewer:** `flint` (local 40B)
**Persona:** `.agents/skills/_shared/elephant-goldfish/zoom-out.md`
**Frozen candidate:** blueprint.md blob hash 33a7b7f0 (post-revision; prior candidate 887750e5 reviewed in Stage 5)

Isolation attestation:
1. Fresh context — this reviewer has not seen this blueprint before this session.
2. Read-only on the repo — no file was edited except this report.
3. Did not read the other stage reports or `review-ledger.md`.
4. No network, no MCP tools — local file reads and git commands only.

## Findings (reviewer output, verbatim)

### F-S8-1 — Major

**Affected scope:** Blueprint Task 6 (installer distribution) and the Design contract (bootstrap resolution order).

**Description:** Task 6 requires installer work that has no counterpart in the current code: a "host-agnostic bootstrap-distribution step" (`.agents/skills/_shared/agent-config/router-bootstrap.md` → `~/.skillgrid/router-bootstrap.md`), a `routerPluginRel` distribution fact, and "`sync-repo` refreshes the bootstrap". Task 6 Step 1 also requires `--agents omp` to copy the bootstrap to `~/.skillgrid/router-bootstrap.md`. Repo evidence: `skillgrid-cli/internal/install/install.go:216-232` (`SyncRepo`) mirrors only `mirrorDirs` (`skillgrid-cli/internal/install/hooktree.go:26` = `.agents`, `docs`, `git-hooks`, `hooks`) into `~/.skillgrid/` and contains no bootstrap copy; a repo-wide grep over `skillgrid-cli/` for `router-bootstrap` returns no source hits; nothing writes `~/.skillgrid/router-bootstrap.md`. The Design contract's resolution order names `~/.skillgrid/router-bootstrap.md` (installer-managed) as the first hop, so the plan leaves the new step's placement (installer vs setup vs memory) implicit and undisposed.

**Rationale:** Delivery-path coherence. The plan cannot be executed as written without an explicit installer decision for the bootstrap copy/refresh step; this is a must-fix-before-slicing gap.

### F-S8-2 — Major

**Affected scope:** Task 4 context / File Structure, Task 6, per-host mechanism table (opencode).

**Description:** The plan re-creates `plugins/opencode/` as the router tree (Task 4) based on the premise that it "was emptied by d5148778", but the ADR-0032 installer surface still keys source paths on `plugins/opencode/<base>.ts` — `skillgrid-cli/internal/install/agents.go:20-38` (`nativePluginRels`) and `skillgrid-cli/internal/setup/setup.go:38-51,166-175` (`nativePluginBases`) — a directory with **zero** tracked files after d5148778. Live memory-plugin sources now sit at `plugins/pi/mnemonic/<base>.ts`, and `SetupOmp` reads `plugins/pi/mnemonic/`. Task 4 creates only the router module plus a minimal `{"type":"module"}` package.json; the memory plugin bases are delegated to the ADR-0067 rework, whose tree shape is not pinned against the installer contracts.

**Rationale:** Internal consistency / executability. The plan's own Task 4 context ("tree re-created", "idempotent, create-if-missing") does not reconcile the source-path drift between the plan, the installer, and the post-d5148778 repo state.

### F-S8-3 — Minor

**Affected scope:** Design contract (bootstrap content) and Task 5 (Kilo).

**Description:** The proven Kilo session-start pattern (`/tmp/skillgrid-plugin-research/ide-rules.md`, §2.4) injects the full router body via `experimental.chat.messages.transform` with a present-guard, while the blueprint's Kilo row (Task 5) and the shared Design contract instead specify a ~15-line pointer bootstrap for the four pointer hosts, with no Kilo-specific rationale recorded.

**Rationale:** Design-decision audit; cheap to fix before slicing.

### F-S8-4 — Minor

**Affected scope:** Requirement R3 (main-agent-only) and the cursor injection path (Design contract, Task 1, Task 2).

**Description:** The main-agent-only subagent skip is stated globally (Design contract) but the cursor hook's stdin contract (`hooks/cursor-session-start.sh:8-13,20-28`: `session_id`, `conversation_id`, `cwd`, `workspace_roots`) exposes no subagent/parent signal, and Task 1 specifies no subagent guard for the cursor path; the subagent skip is only specified for the omp path (Task 2) and as an "explicit main-agent check" per injector.

**Rationale:** Requirement coverage; minor because the pointer-host injectors are expected to carry the guard and the plan defers it to per-injector implementation.

### F-S8-5 — Minor

**Affected scope:** Threat Matrix (executable-file classification) and the installer copy helper.

**Description:** The Threat Matrix row "Executable-file classification (the `.sh` hook must stay executable through the installer copy; `.ts`/`.md`/`.json` distribute as data)" is not reflected in the generic installer copy path: `skillgrid-cli/internal/install/hooktree.go:196-210` (`copyFile`) writes destination files with mode `0o644`; the `0o755` behavior exists only in `skillgrid-cli/internal/setup/cursor.go:130-133` (`installCursorHookScripts`). If Task 6 adds a `.sh` copy through the installer set, the threat row's precondition is not met by the shared helper.

**Rationale:** Threat-row/implementation drift; minor.

## Verdict: REVISE (5 findings)
