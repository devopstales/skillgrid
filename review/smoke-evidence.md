# Smoke Evidence — TICKET-01 (2026-11-17-skillgrid-host-plugins)

- **Ticket:** TICKET-01 (door check) — Tracker `t_06bb0c13`
- **Change:** `2026-11-17-skillgrid-host-plugins`
- **Worktree:** `/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_06bb0c13` (branch `wt/t_06bb0c13`, base `d3026233`)

## TICKET-01 door check — bootstrap asset + cursor hook

Hard gate for the blueprint tracer thread (blueprint Task 1, Step 5). Hook-level
oracles below are the enforceable gate; the live cursor session clause is the
held-out backstop (recorded ABANDON, see § Live door check).

### Oracle 1 — cwd `/tmp`: no bootstrap content, exit 0

Command:

```bash
printf '{"cwd":"/tmp"}' | hooks/cursor-session-start.sh
```

Output:

```json
{"additional_context":"Mnemonic is active. Start with mem_session_start (title), save decisions/bugfixes via mem_save, and close with mem_session_summary + mem_session_end. Prefer mem_context / mem_search before re-deriving prior work. Call code_explore before rg."}
```

Exit: `0`. Bootstrap content present: **no**. Verdict: **PASS**.

### Oracle 2 — cwd = worktree root: bootstrap present in `additional_context`

Command:

```bash
printf '{"cwd":"/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_06bb0c13"}' | hooks/cursor-session-start.sh
```

Output (excerpt — bootstrap block then mnemonic prime):

```json
{"additional_context":"# Skillgrid Router Bootstrap\n\nThis project is configured with Skillgrid.\n\nFirst action: read `.agents/skills/using-skillgrid/SKILL.md` (or the host's skill-load command).\n\nRequest classes: start / continue / status / review-only / bounded.\n\nAnnounce: `Using skillgrid:using-skillgrid to {purpose}`.\n\nLoad hints:\n- pi: `/skill:using-skillgrid` + `using-skillgrid/references/pi-tools.md`\n- opencode: `skill` / `@using-skillgrid` + `using-skillgrid/references/opencode-tools.md`\n- kilo: read the path\n- cursor: `/using-skillgrid` or read\n- omp: `/skill:using-skillgrid`\n\nMnemonic is active. Start with mem_session_start (title), ..."}
```

Exit: `0`. Bootstrap content present: **yes** (front of `additional_context`). Verdict: **PASS**.

### Oracle 3 — missing asset: static fallback present, exit 0 (fail-open)

Command:

```bash
mv .agents/skills/_shared/agent-config/router-bootstrap.md /tmp/router-bootstrap.md.bak
printf '{"cwd":"/Users/paladm/git/ai-test/skillgrid-v2/.worktrees/t_06bb0c13"}' | hooks/cursor-session-start.sh
mv /tmp/router-bootstrap.md.bak .agents/skills/_shared/agent-config/router-bootstrap.md
```

Output (excerpt):

```json
{"additional_context":"skillgrid: router bootstrap fallback - first action: read `.agents/skills/using-skillgrid/SKILL.md`; announce `Using skillgrid:using-skillgrid to {purpose}`.\n\nMnemonic is active. Start with mem_session_start (title), ..."}
```

Exit: `0`. Static fallback present: **yes**. No user-visible error. Verdict: **PASS**.

### Oracle 4 — asset edit propagation (live read, no reinstall)

- Edit: appended `# ORACLE-MARKER-EDIT-TEST` to `.agents/skills/_shared/agent-config/router-bootstrap.md`.
- Re-ran in-cwd pipe: marker found in `additional_context` output (grep count `1`). Exit `0`.
- Restored: restored byte-identical content (567 bytes, original line set intact).
- Re-ran in-cwd pipe after restore: marker absent. Exit `0`.

Verdict: **PASS** (live read propagates without reinstall).

### Oracle 5 — hook syntax

Command: `bash -n hooks/cursor-session-start.sh` → exit `0`.

Verdict: **PASS**.

### Live door check (blueprint Task 1 Step 5) — PASS (controller, 2026-11-19)

Run by the controller after the implementer's headless ABANDON, using
`cursor agent -f -p` (cursor v3.29.15 — CLI agent mode fires plugin
sessionStart hooks; no display server needed).

**Plugin install layout (finding, verified empirically):** cursor loads local
plugins only from REAL directories under `~/.cursor/plugins/local/<name>/` — a
symlinked plugin directory is NOT loaded (tested). The manifest must be
`plugin.json` at the directory root; the repo's `.cursor-plugin/plugin.json`
subdirectory layout is not recognized for local plugins. Working layout used:
real dir `~/.cursor/plugins/local/skillgrid/` with a copied root `plugin.json`
+ symlinks to the worktree's `hooks/.agents/rules/agents/plugins` subdirs (hook
edits propagate live). T6 installer must use this layout.

**Positive (skillgrid project = worktree root):** fresh session
(`cursor agent -f -p "…does your session context contain a 'Skillgrid Router
Bootstrap' block?…"`) — the sessionStart hook fired (verified by a temporary
instrumentation line appended to the hook, removed before commit), and the
saved transcript
(`~/.cursor/projects/…-worktrees-t-06bb0c13/agent-transcripts/1662c7ec-…/1662c7ec-….jsonl`)
records the agent's first-turn reasoning: *"…a Skillgrid Router Bootstrap
block, and it does, under hooks_context. The first step is to read the
using-skillgrid SKILL.md file."* — bootstrap present in first-turn context,
first action correctly extracted. Verdict: **PASS**.

**Negative (non-skillgrid dir = /tmp):** fresh session, same prompt — hook
fired (project condition false → no bootstrap), agent answered: *"NO
BOOTSTRAP … my session context has no 'Skillgrid Router Bootstrap' block, so
there's no first action to state."* Verdict: **PASS**.

### Gates

| Gate | Command | Result |
|---|---|---|
| Hook syntax | `bash -n hooks/cursor-session-start.sh` | exit 0, PASS |
| Go build (modules) | `go build ./...` (mnemonic, skillgrid-cli) | exit 0, green |
| Go test | `go test ./...` (mnemonic, skillgrid-cli) | in progress / recorded at QA |

## SATISFIES (TICKET-01 acceptance mapping)

- Happy path — one-line edit reaches next session without reinstall: **Oracle 4 PASS**.
- Happy path — cursor session start injects router bootstrap even with memory server down: **injected before mnemonic block; server-down independent** (injection is a prepend into `additional_context`, independent of the mnemonic health/prime path) — hook-level only.
- Happy path — skillgrid project session receives bootstrap: **Oracle 2 PASS**.
- Edge — non-skillgrid dir receives nothing: **Oracle 1 PASS**.
- Edge — installed copy takes precedence over checkout asset: **in this environment installed copy absent** (`/Users/paladm/.skillgrid/router-bootstrap.md` not present); checkout leg exercised (Oracle 2). Installed-copy precedence is installer-owned (TICKET-06) and not re-testable here without writing to `$HOME` — noted as residual.
- Edge — hosts without tool-mapping file get plain first-action hint: asset line set includes kilo/cursor/omp plain loads — read-back PASS (asset content).
- Failure — bootstrap asset missing at session start: **Oracle 3 PASS**.
- Failure — session-start hook failure never blocks session start: fail-open shape retained (final node `|| echo '{}'`, `exit 0`).
- Failure — neither installed nor checkout asset resolvable: **Oracle 3 PASS** (installed absent + asset moved aside → static fallback).
- Failure — cursor session-start hook absent: not re-testable here (host config), noted.
- Happy path — hosts with tool-mapping file get mapping hint: asset load-hint table references existing `using-skillgrid/references/pi-tools.md` + `opencode-tools.md`; kilo/cursor/omp plain load. Read-back PASS.
