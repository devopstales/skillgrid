# Stage 5 — Initial Zoom Out (Round 1)

**Reviewer:** `flint` (local 40B; `reviewer` backend 429 after initial spawn + one retry — see ledger harness deviation)
**Persona:** `.agents/skills/_shared/elephant-goldfish/zoom-out.md`
**Frozen candidate:** base 3f033a21 → head add39cfa, 1 file, hash 887750e5 (blueprint.md)

Isolation attestation: all four enforced (fresh context, read-only, no
project instructions, no network/MCP)

## Findings (reviewer output, verbatim)

### F-S5-1 — Blocker
**Affected scope:** blueprint.md — Tech Stack (line 19) / §2 Per-host mechanism (line 121)

**Description:** Finalized in the add39cfa compliance pass (2026-11-18 15:21)
against a repo where plugins/opencode/ still held the native TS plugins and
package.json; hours later HEAD d5148778 (2026-11-18 15:33, "fix: clean olld
plugins") deleted that tree, and the installer/setup are mid-refactor
(RunSetup agent-key switch, SetupPi→SetupOmp, nativePluginBases drift).

**Rationale:** The plan's artifact list (plugins/opencode/skillgrid-router.ts +
package.json registration) and Tech Stack (TypeScript, four host injectors
following plugins/** conventions) presuppose a layout that no longer exists, so
the finalized plan cannot be sliced or executed as written.

**Main-agent verification:** CONFIRMED. `git show d5148778 --stat` = 43 files,
10,591 deletions, all under `plugins/opencode/` + `plugins/kilo/` (both trees
now empty). `git status` shows an uncommitted working-tree refactor:
`skill-pi` → `omp` agent-key rename across `skillgrid-cli/cmd/skillgrid/main.go`,
`internal/install/{agents,config,install,hookbridge,lsp}.go`,
`internal/setup/{pi,setup}.go` (SetupPi→SetupOmp, `~/.skill-pi/agent`→`~/.omp/agent`,
`SKILL_PI_AGENT_DIR`→`OMP_AGENT_DIR`, NPM `@devopstales/skill-pi`→
`@oh-my-pi/pi-coding-agent`). External change (user's in-flight work); the repo
is the source of truth (ADR-0013) — the blueprint adapts.

### F-S5-2 — Major
**Affected scope:** blueprint.md — §1 Design core (line 95) / Hypothesis (line 35)

**Description:** The plan adds a new shared bootstrap asset plus five per-host
injectors to close the soft-trigger gap without evaluating the existing
session-start seam (`hooks/session-start.js`, full-content SessionStart
IMPORTANT, wired via the settings hooks bridge) or the AGENTS.md entry line.

**Rationale:** YAGNI/reuse: the deterministic-injection requirement could be met
by adapting or wiring the existing seam; the plan instead introduces a second
injection seam, asset, and installer/sync distribution for a gap the repo
already addressed.

**Main-agent verification:** CONFIRMED with correction. `hooks/session-start.js`
(49 lines, committed since 2026-09-24, predates this change by 2 months)
injects the FULL router SKILL.md as a priority-IMPORTANT SessionStart payload,
fail-open (INFO note + exit 0). `installOmpHookBridge` (committed at HEAD; WT
renames it) writes it as the `SessionStart` hook of
`~/.omp/agent/settings.json`, executed by the `@hsingjui/pi-hooks` npm plugin
(verified loaded in `~/.omp/plugins/omp-plugins.lock.json`; research
omp-local.md §55). So the pi-based host (omp) already has deterministic
router injection — the blueprint's omp TS extension is redundant. The AGENTS.md
entry line (shipped 2026-11-17) is the documented no-code fallback and remains.
Additional fact found: the blueprint's "pi" row (plugins/pi/mnemonic/
extension) and "omp" row (plugins/omp/ extension) both distribute to the SAME
agent dir (`~/.omp/agent/extensions/`) after the rename — two rows, one host.
Plain pi (`~/.pi/agent`, in live use on this machine) has NO installer key and
no skillgrid wiring at all.

### F-S5-3 — Major
**Affected scope:** blueprint.md — §2 Per-host mechanism (line 121) / Key links (line 67)

**Description:** The anti-weed rule ("no host-specific logic beyond load +
inject") is honored only at the loader level; the plan still prescribes five
bespoke injectors and per-host `{load_hint}` variants without first fixing the
shared contract.

**Rationale:** Anti-weeds: adapter proliferation risk; the shape should be one
contract with mechanical adapters rather than five parallel implementations.

**Main-agent verification:** PARTIALLY VALID. §1 does fix the shared contract
first (bootstrap contract + resolution order + project-conditional +
main-agent-only) — the finding's "without first fixing" claim misreads. The
valid kernel: five bespoke injectors where one shipped seam (session-start.js
+ hook bridge) already serves the pi-based host. Resolved together with
F-S5-1/F-S5-2: bespoke injectors drop from 5 to 3 (opencode, kilo, cursor).

### F-S5-4 — Minor
**Affected scope:** blueprint.md — Threat Matrix (line 75) / Global Constraints (line 88)

**Description:** Fail-open (R4) is required and the shell-command threat row is
concrete, but the fail-open/static-fallback behavior is not anchored to the
existing hooks/session-start.js fail-open pattern.

**Rationale:** Altitude: the plan's altitude is otherwise sound; this is a
consistency gap, not a design blocker.

**Main-agent verification:** VALID. `session-start.js` is the in-repo fail-open
precedent (catch → INFO payload → `process.exit(0)`). Fix: R4 + threat matrix
cite it as the pattern; the new cwd guard in session-start.js follows the same
shape.

### F-S5-5 — Nit
**Affected scope:** blueprint.md — §1 Design core (line 95) / Goal (line 15)

**Description:** The plan centers on a ~15-line pointer bootstrap plus live-read
resolution order, static fallback, and sync-repo refresh without weighing the
already-shipped full-content hook or the AGENTS.md-only path for context cost.

**Rationale:** YAGNI: some bootstrap machinery may exceed the simplest fix for
the stated goal.

**Main-agent verification:** VALID. §6 weighs cost vs superpowers' full
4–8KB, but not vs the repo's own shipped full-content hook (14KB/session for
the hook-based host). Fix: §6 states the per-host split explicitly — the
pi-based host keeps the shipped full-content hook (the 2026-09-24 decision for
hook-based harnesses); the extension/plugin hosts (opencode, kilo, cursor) use
the 15-line pointer bootstrap; unification is a follow-up.

## Resolution summary (applied to blueprint.md this round)

- F-S5-1 → fixed: §2/§3 re-anchored to post-rename, post-cleanup repo state;
  opencode/kilo trees re-created minimal; §3 targets the post-rename installer
  API (SetupOmp, `omp` key) with the uncommitted rename as a named prerequisite.
- F-S5-2 → fixed: pi+omp rows merged into one "pi-based host (omp key)" row on
  the shipped hook-bridge seam; plain-pi coverage recorded as Decision note
  (AGENTS.md fallback now; `pi` installer key = follow-up, user ruling at the
  slice gate).
- F-S5-3 → fixed (consequence of F-S5-1/F-S5-2): bespoke injectors 5→3.
- F-S5-4 → fixed: R4 + threat matrix anchored to session-start.js precedent.
- F-S5-5 → fixed: §6 per-host context-cost split stated explicitly.
