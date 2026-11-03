# Skillgrid + Mnemonic ↔ Hermes Agent Integration — Implementation Blueprint

> **Role:** this is the *how* — the implementation plan: concrete steps,
> files to touch, code, tests, and verification. The companion `briefing.md`
> is the *what* (requirements & intent).

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread — the MemoryProvider plugin (Python shim → `mnemonic` CLI) works first, then the Go installer wires it into Hermes (MCP config, plugin copy, skills, Kanban board), then the ticketing skill gains the Kanban backend. Integration risk (Python plugin loading, Hermes config YAML, Kanban board creation) is the unknown; the tracer thread proves the provider activates before the installer thickens.

**Goal:** Make Hermes Agent a first-class install target for skillgrid. Primary: native MemoryProvider plugin (Layer 2). Complement: MCP server (Layer 3). Skill deployment. Protocol injection. Hermes Kanban as second ticketing backend. All wired by `skillgrid install --agents hermes`.

**Architecture:** Four integration layers + ticketing:
1. **MemoryProvider** (Python, `plugins/hermes/`) — `MnemonicMemoryProvider(MemoryProvider)` implementing the Hermes lifecycle. Thin shim delegating to the `mnemonic` Go binary via `subprocess.run`.
2. **MCP server** (Go, existing `mnemonic mcp`) — registered in `~/.hermes/config.yaml` with `MNEMONIC_MCP_EXCLUDE=mem` to suppress `mem_*` tools (handled natively by the provider).
3. **Skills** — `.agents/skills/**` copied to `~/.hermes/skills/`.
4. **Protocol** — `system_prompt_block()` primary, SOUL.md marker block fallback.
5. **Ticketing** — `type: hermes-kanban` config, `ticketing` skill publishes to Kanban via `hermes kanban create`.

**Tech Stack:** Go 1.22+ (mnemonic binary, installer), Python 3.11+ (MemoryProvider shim), yaml.v3 (Hermes config), existing `mnemonic/internal/setup` + `skillgrid-cli/internal/install` packages. No new Go dependencies.

**Spec:** `.skillgrid/specs/2026-10-24-hermes-integration/briefing.md`. Per ADR-0031 (Go 1.22+ floor), ADR-0013 (repo is SoT).

## Terms

- **MemoryProvider** — Python class subclassing `agent.memory_provider.MemoryProvider`. Registered via `register(ctx)` → `ctx.register_memory_provider(Provider())`. One per profile, one external provider at a time.
- **Hermes home** — `$HERMES_HOME` (default `~/.hermes`). All state (config.yaml, .env, SOUL.md, plugins, skills, kanban.db) lives under it.
- **Kanban board** — A durable task queue with its own SQLite DB, workspaces, and dispatcher loop. One per skillgrid project (slug = project name).
- **Tool split** — `mem_*` tools → MemoryProvider (native). `code_*`, `web_*`, everything else → MCP server. No tool name in both.
- None beyond the briefing's terms.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `plugins/hermes/provider.py` defines `MnemonicMemoryProvider(MemoryProvider)` with all required methods
- `plugins/hermes/__init__.py` defines `register(ctx)` calling `ctx.register_memory_provider(MnemonicMemoryProvider())`
- `plugins/hermes/plugin.yaml` has `name: mnemonic` and a description
- The provider's `is_available()` returns `True` when `mnemonic` is on PATH, `False` otherwise
- The provider's `get_tool_schemas()` returns valid OpenAI function-calling schemas for `mem_save`, `mem_search`, `mem_context`, `mem_get`, `mem_update`
- The provider's `handle_tool_call("mem_save", {...})` calls `mnemonic mem save` via subprocess and returns JSON
- The provider's `handle_tool_call("mem_search", {...})` calls `mnemonic mem search` via subprocess and returns JSON
- The provider's `prefetch(query)` returns the cached result from `queue_prefetch()` (empty on first turn, populated after)
- The provider's `queue_prefetch(query)` spawns a background `mnemonic mem search` subprocess
- The provider's `sync_turn(user, assistant)` calls `mnemonic mem save` via subprocess (non-blocking, background thread)
- The provider's `system_prompt_block()` returns the memory protocol text
- The provider's `on_memory_write(action, target, content)` mirrors native memory writes into mnemonic
- The provider's `shutdown()` flushes pending background work
- `SetupHermes()` copies `plugins/hermes/` → `~/.hermes/plugins/mnemonic/`
- `SetupHermes()` writes `mcp_servers` block to `~/.hermes/config.yaml` (YAML) with all servers from `config.d/mcp.yaml`
- `SetupHermes()` sets `MNEMONIC_MCP_EXCLUDE=mem` in the mnemonic MCP entry's `env`
- `SetupHermes()` writes the SOUL.md marker block (fallback protocol)
- `SetupHermes()` creates the Kanban board if it doesn't exist: `hermes kanban boards create <slug>`
- `SetupHermes()` copies `.agents/skills/**` → `~/.hermes/skills/`
- `SetupHermes()` updates `.skillgrid/config.yaml` with `type: hermes-kanban` under `ticketing:`
- `RunSetup("hermes", ...)` dispatches to `SetupHermes()`
- `skillgrid install --agents hermes` completes and configures all of the above
- `skillgrid install --dry-run --agents hermes` prints planned changes without writing
- After install, `hermes memory setup` lists `mnemonic` as an available provider
- After install, `hermes mcp list` shows `skillgrid-mnemonic` as enabled
- After install, `hermes mcp test skillgrid-mnemonic` exits 0
- After install, a Kanban ticket published via the `ticketing` skill appears in `hermes kanban list`
- A skillgrid ticket with `parents` creates a dependency edge in Kanban
- The Kanban dispatcher promotes the child to `ready` when all parents complete
- `cd mnemonic && go test ./...` passes (new hermes setup tests)
- `cd skillgrid-cli && go test ./...` passes (new hermes install tests)
- Existing agents (opencode, kilocode, cursor, pi) continue to install correctly

**Artifacts** (files that must exist with real implementation, not stubs):

| File | Action |
|---|---|
| `plugins/hermes/__init__.py` | **new** — `register(ctx)` entry point |
| `plugins/hermes/provider.py` | **new** — `MnemonicMemoryProvider(MemoryProvider)` full lifecycle |
| `plugins/hermes/plugin.yaml` | **new** — manifest (name: mnemonic, description) |
| `mnemonic/internal/setup/hermes.go` | **new** — `SetupHermes()`: plugin copy, MCP config, SOUL.md, Kanban board, skills, ticketing config |
| `mnemonic/internal/setup/hermes_test.go` | **new** — tests for all SetupHermes operations |
| `mnemonic/internal/setup/setup.go` | **edit** — add `"hermes"` case to `RunSetup` switch |
| `mnemonic/cmd/mnemonic/mcp.go` | **edit** — add `MNEMONIC_MCP_EXCLUDE` env var handling (suppress matching tool prefixes) |
| `skillgrid-cli/internal/install/hermes.go` | **new** — hermes-specific install (prerequisite check, skill copy) |
| `skillgrid-cli/internal/install/install.go` | **edit** — hermes in agent selection, install flow |
| `skillgrid-cli/internal/install/agents.go` | **edit** — hermes agent config (if needed) |
| `.agents/skills/planning/ticketing/SKILL.md` | **edit** — add `hermes-kanban` backend |
| `.skillgrid/config.yaml` | **edit** — add `type: hermes-kanban` option (schema, not a fixed value) |

**Key links** (critical connections between artifacts that must work together):
- `RunSetup("hermes", ...)` → `SetupHermes(home, repoRoot, mcpEntries, dryRun)`
- `SetupHermes` reads `mcpEntries` from `LoadMCPConfig(repoRoot)` (same source as opencode/kilo/cursor)
- `SetupHermes` copies `plugins/hermes/` (from repoRoot) → `$HERMES_HOME/plugins/mnemonic/`
- `SetupHermes` writes `$HERMES_HOME/config.yaml` `mcp_servers` block (YAML, not JSON)
- The mnemonic MCP entry gets `env: {MNEMONIC_MCP_EXCLUDE: "mem"}` to suppress `mem_*` tools
- `SetupHermes` runs `hermes kanban boards create <slug>` (non-fatal if board exists)
- `SetupHermes` updates `.skillgrid/config.yaml` `ticketing.type` → `hermes-kanban` + `ticketing.kanban.board` + `ticketing.kanban.assignee`
- The `ticketing` skill reads `ticketing.type` from config and dispatches to the Kanban backend
- The Kanban backend calls `hermes kanban create <title> --assignee <profile> --board <slug>` with `parents` from dependency edges
- The provider's `prefetch()` returns the cached result from `queue_prefetch()`; `sync_turn()` queues the next prefetch
- The provider's `handle_tool_call()` maps tool name → `mnemonic` CLI subcommand, passes args as flags, parses JSON output

## Steps

### Step 1 — MemoryProvider plugin (Wave 1)

**Files:** `plugins/hermes/__init__.py`, `plugins/hermes/provider.py`, `plugins/hermes/plugin.yaml`

**1a. `plugin.yaml`:**
```yaml
name: mnemonic
description: "Skillgrid Mnemonic memory provider — persistent project memory with code intelligence"
```

**1b. `__init__.py`:**
```python
from .provider import MnemonicMemoryProvider

def register(ctx):
    ctx.register_memory_provider(MnemonicMemoryProvider())
```

**1c. `provider.py` — the full lifecycle:**

```python
"""Mnemonic MemoryProvider for Hermes Agent.

Thin Python shim delegating to the mnemonic Go binary via subprocess.
All logic lives in the Go binary; this class handles the Hermes lifecycle.
"""

from __future__ import annotations

import json
import logging
import shutil
import subprocess
import threading
from typing import Any, Dict, List, Optional

logger = logging.getLogger(__name__)

# Import the ABC from Hermes (available at runtime when loaded as a plugin)
try:
    from agent.memory_provider import MemoryProvider, spawn_context_thread
except ImportError:
    MemoryProvider = object  # fallback for unit tests outside Hermes
    def spawn_context_thread(target, *, name="mnemonic", daemon=True, args=(), kwargs=None):
        t = threading.Thread(target=target, name=name, daemon=daemon, args=args, kwargs=kwargs or {})
        return t

TOOL_SCHEMAS: List[Dict[str, Any]] = [
    {
        "name": "mem_save",
        "description": "Save an observation to project memory",
        "parameters": {
            "type": "object",
            "properties": {
                "title": {"type": "string", "description": "Verb + what — short, searchable"},
                "type": {"type": "string", "enum": ["bugfix","decision","architecture","discovery","pattern","config","preference"]},
                "content": {"type": "string"},
                "scope": {"type": "string", "enum": ["project","personal"], "default": "project"},
                "topic_key": {"type": "string", "description": "Stable key for evolving topics"},
            },
            "required": ["title", "type", "content"],
        },
    },
    {
        "name": "mem_search",
        "description": "Search project memory for relevant observations",
        "parameters": {
            "type": "object",
            "properties": {
                "query": {"type": "string"},
                "limit": {"type": "integer", "default": 10},
            },
            "required": ["query"],
        },
    },
    {
        "name": "mem_context",
        "description": "Get recent session history (fast, cheap)",
        "parameters": {
            "type": "object",
            "properties": {
                "limit": {"type": "integer", "default": 20},
            },
        },
    },
    {
        "name": "mem_get",
        "description": "Get full untruncated content of an observation by ID",
        "parameters": {
            "type": "object",
            "properties": {
                "id": {"type": "string"},
            },
            "required": ["id"],
        },
    },
    {
        "name": "mem_update",
        "description": "Update an existing observation",
        "parameters": {
            "type": "object",
            "properties": {
                "id": {"type": "string"},
                "title": {"type": "string"},
                "content": {"type": "string"},
            },
            "required": ["id"],
        },
    },
]


class MnemonicMemoryProvider(MemoryProvider):
    """Hermes MemoryProvider backed by the mnemonic Go binary."""

    @property
    def name(self) -> str:
        return "mnemonic"

    def is_available(self) -> bool:
        return shutil.which("mnemonic") is not None

    def unavailable_reason(self) -> str:
        return "mnemonic binary not found on PATH. Run: skillgrid install"

    def initialize(self, session_id: str, **kwargs) -> None:
        self._session_id = session_id
        self._hermes_home = kwargs.get("hermes_home", "~/.hermes")
        self._agent_context = kwargs.get("agent_context", "primary")
        self._cache: str = ""
        self._lock = threading.Lock()

    def system_prompt_block(self) -> str:
        # Return the memory protocol text. Read from the protocol file
        # if available, otherwise return a minimal instruction block.
        return (
            "## Mnemonic Memory Protocol\n\n"
            "You have access to persistent project memory via the "
            "mem_save, mem_search, mem_context, mem_get, mem_update tools.\n\n"
            "- Save decisions, bug fixes, patterns, and non-obvious discoveries "
            "with mem_save.\n"
            "- Search memory before re-deriving known information.\n"
            "- Use mem_context at session start to recover context.\n"
        )

    def prefetch(self, query: str, *, session_id: str = "") -> str:
        with self._lock:
            return self._cache

    def queue_prefetch(self, query: str, *, session_id: str = "") -> None:
        def _fetch():
            try:
                result = self._run(
                    ["mem", "search", "--query", query, "--json", "--limit", "5"]
                )
                data = json.loads(result)
                entries = data.get("observations", data.get("results", []))
                if entries:
                    self._cache = "\n".join(
                        f"- [{e.get('type','?')}] {e.get('title','?')}: "
                        f"{e.get('content','')[:200]}"
                        for e in entries
                    )
                else:
                    self._cache = ""
            except Exception as exc:
                logger.debug("mnemonic prefetch failed: %s", exc)
                self._cache = ""
        t = spawn_context_thread(target=_fetch, name="mnemonic-prefetch")
        t.start()

    def sync_turn(
        self, user_content: str, assistant_content: str, *,
        session_id: str = "", messages=None, turn_author=None,
    ) -> None:
        # Non-blocking: spawn a background thread to save the turn.
        def _save():
            if self._agent_context != "primary":
                return
            try:
                self._run([
                    "mem", "save",
                    "--title", f"Session turn {session_id[:8]}",
                    "--type", "discovery",
                    "--content", f"User: {user_content[:500]}\nAssistant: {assistant_content[:500]}",
                    "--json",
                ])
            except Exception as exc:
                logger.debug("mnemonic sync_turn failed: %s", exc)
        t = spawn_context_thread(target=_save, name="mnemonic-sync")
        t.start()
        # Queue the next prefetch for the upcoming turn
        if user_content.strip():
            self.queue_prefetch(user_content, session_id=session_id)

    def get_tool_schemas(self) -> List[Dict[str, Any]]:
        return TOOL_SCHEMAS

    def handle_tool_call(self, tool_name: str, args: Dict[str, Any], **kwargs) -> str:
        cmd = self._tool_to_cmd(tool_name, args)
        result = self._run(cmd)
        return result

    def on_session_end(self, messages: List[Dict[str, Any]]) -> None:
        # End-of-session extraction: distill insights.
        # For v1, just log the session length. The distill pass is a v2 feature.
        logger.info("mnemonic session end: %d messages", len(messages))

    def on_pre_compress(self, messages: List[Dict[str, Any]]) -> str:
        # Extract insights before context compression.
        # For v1, return "" (no extraction). v2: LLM pass over recent messages.
        return ""

    def on_memory_write(
        self, action: str, target: str, content: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> None:
        # Mirror Hermes native memory writes into mnemonic.
        try:
            self._run([
                "mem", "save",
                "--title", f"Native {action} ({target})",
                "--type", "preference" if target == "user" else "discovery",
                "--content", content[:1000],
                "--json",
            ])
        except Exception as exc:
            logger.debug("mnemonic on_memory_write failed: %s", exc)

    def shutdown(self) -> None:
        # Flush pending background work. v1: no-op (threads are daemon).
        pass

    # --- Internal helpers ---

    def _run(self, args: List[str]) -> str:
        """Run the mnemonic CLI and return stdout."""
        cmd = ["mnemonic"] + args
        result = subprocess.run(
            cmd, capture_output=True, text=True, timeout=30
        )
        if result.returncode != 0:
            raise RuntimeError(f"mnemonic {' '.join(args)}: {result.stderr.strip()}")
        return result.stdout

    @staticmethod
    def _tool_to_cmd(tool_name: str, args: Dict[str, Any]) -> List[str]:
        if tool_name == "mem_save":
            cmd = ["mem", "save", "--json"]
            if args.get("title"):
                cmd += ["--title", args["title"]]
            if args.get("type"):
                cmd += ["--type", args["type"]]
            if args.get("content"):
                cmd += ["--content", args["content"]]
            if args.get("scope"):
                cmd += ["--scope", args["scope"]]
            if args.get("topic_key"):
                cmd += ["--topic-key", args["topic_key"]]
            return cmd
        elif tool_name == "mem_search":
            cmd = ["mem", "search", "--json"]
            if args.get("query"):
                cmd += ["--query", args["query"]]
            if args.get("limit"):
                cmd += ["--limit", str(args["limit"])]
            return cmd
        elif tool_name == "mem_context":
            cmd = ["mem", "context", "--json"]
            if args.get("limit"):
                cmd += ["--limit", str(args["limit"])]
            return cmd
        elif tool_name == "mem_get":
            return ["mem", "get", args["id"], "--json"]
        elif tool_name == "mem_update":
            cmd = ["mem", "update", args["id"], "--json"]
            if args.get("title"):
                cmd += ["--title", args["title"]]
            if args.get("content"):
                cmd += ["--content", args["content"]]
            return cmd
        raise ValueError(f"Unknown tool: {tool_name}")
```

**1d. Verify mnemonic CLI flags** — Check that `mnemonic mem search --query <q> --json --limit <n>`, `mnemonic mem save --title <t> --type <type> --content <c> --json`, `mnemonic mem context --json`, `mnemonic mem get <id> --json`, `mnemonic mem update <id> --title <t> --content <c> --json` all exist. Add `--json` flags if missing.

**Verification:**
```bash
# Copy plugin to Hermes discovery path
cp -r plugins/hermes/ ~/.hermes/plugins/mnemonic/

# Verify discovery
hermes memory setup  # should list "mnemonic"

# Activate
hermes config set memory.provider mnemonic

# Test in a new session
hermes -z "Remember that the database uses Postgres 16"
hermes -z "What database do we use?"  # should return Postgres 16
```

### Step 2 — SetupHermes (Wave 2)

**Files:** `mnemonic/internal/setup/hermes.go`, `mnemonic/internal/setup/hermes_test.go`, `mnemonic/internal/setup/setup.go`, `mnemonic/cmd/mnemonic/mcp.go`

**2a. `hermes.go`:**

```go
package setup

// SetupHermes configures Mnemonic for Hermes Agent:
//  1. Copies the MemoryProvider plugin from plugins/hermes/ to $HERMES_HOME/plugins/mnemonic/
//  2. Writes mcp_servers block to $HERMES_HOME/config.yaml (YAML)
//  3. Sets MNEMONIC_MCP_EXCLUDE=mem on the mnemonic MCP entry
//  4. Writes SOUL.md marker block (fallback protocol)
//  5. Creates the Kanban board (non-fatal if exists)
//  6. Copies .agents/skills/** to $HERMES_HOME/skills/
//  7. Updates .skillgrid/config.yaml ticketing.type to hermes-kanban
func SetupHermes(home, repoRoot string, mcpEntries []MCPServerConfig, dryRun bool) error {
    hermesHome := envOr("HERMES_HOME", filepath.Join(home, ".hermes"))
    // ... 7 operations, each with dry-run support
}
```

Key implementation details:
- **YAML config write:** Read `config.yaml` with `yaml.v3` (already a dep), merge `mcp_servers`, write back. Backup first (same pattern as JSON configs).
- **Plugin copy:** `copyFromRepo(repoRoot, "plugins/hermes/", dst, dryRun)` for each file. Target: `hermesHome/plugins/mnemonic/`.
- **MCP entries:** For each entry in `mcpEntries`, write `{command, args, env}` to `mcp_servers.<name>`. For the `mnemonic` entry, add `env: {MNEMONIC_MCP_EXCLUDE: "mem"}`.
- **SOUL.md:** Marker-delimited block. If `SOUL.md` doesn't exist, create it. If the marker block exists, replace it. If not, append.
- **Kanban board:** `exec.Command("hermes", "kanban", "boards", "create", slug, "--name", name)` — non-fatal (check exit code, warn on failure).
- **Skills copy:** `copyDir(repoRoot/.agents/skills, hermesHome/skills, dryRun)`.
- **Ticketing config:** Read `.skillgrid/config.yaml`, set `ticketing.type: hermes-kanban`, `ticketing.kanban.board: <slug>`, `ticketing.kanban.assignee: default`. Write back.

**2b. `setup.go` — add case:**
```go
case "hermes":
    return SetupHermes(home, repoRoot, mcpEntries, dryRun)
```

**2c. `mcp.go` — MNEMONIC_MCP_EXCLUDE:**
In `runMCP`, after parsing flags, read `MNEMONIC_MCP_EXCLUDE` env var. If set, filter the tool list before registering with the MCP server. The filter matches tool name prefixes (comma-separated: `mem` matches `mem_save`, `mem_search`, etc.).

**2d. `hermes_test.go` — tests:**
- `TestSetupHermes_MCPConfig` — writes correct YAML, backup created, idempotent
- `TestSetupHermes_PluginCopy` — files copied to correct path
- `TestSetupHermes_SoulMD` — marker block written, replaced on re-run
- `TestSetupHermes_DryRun` — no files written
- `TestSetupHermes_HermesHome` — respects `HERMES_HOME` env var
- `TestSetupHermes_KanbanBoard` — calls `hermes kanban boards create` (mocked)
- `TestMCPExclude` — `MNEMONIC_MCP_EXCLUDE=mem` suppresses `mem_*` tools

**Verification:**
```bash
cd mnemonic && go test ./internal/setup/ -run TestSetupHermes -v
cd mnemonic && go test ./cmd/mnemonic/ -run TestMCPExclude -v
```

### Step 3 — Install flow + Skills + Kanban (Wave 3)

**Files:** `skillgrid-cli/internal/install/hermes.go`, `skillgrid-cli/internal/install/install.go`, `skillgrid-cli/internal/install/agents.go`, `.agents/skills/planning/ticketing/SKILL.md`

**3a. `install.go` — add hermes to agent selection:**
- `selectAgents()`: add "hermes" to the prompt options
- `installAgents()`: if "hermes" selected, check `hermes` on PATH (non-fatal, print `uv tool install hermes-agent`)
- Call `setup.RunSetup("hermes", repoRoot, mcpEntries, home, dryRun)`

**3b. `agents.go` — hermes agent config:**
- Hermes doesn't use a JSON config for harness-level settings (logo, theme). The config is YAML (`config.yaml`). No logo/theme to set. Skip if not needed.

**3c. `ticketing/SKILL.md` — add hermes-kanban backend:**

Add a section to the ticketing skill:

```markdown
## Hermes Kanban Backend

When `.skillgrid/config.yaml` has `ticketing.type: hermes-kanban`:

1. Read `ticketing.kanban.board` (board slug) and `ticketing.kanban.assignee` (profile name)
2. For each ticket from `slicing`:
   - `hermes kanban create "<title>" --assignee <profile> --board <slug>`
   - If the ticket has dependencies: add `parents` (comma-separated task IDs)
   - If the ticket is a coding task: use `worktree` workspace
   - If the ticket is research/planning: use `scratch` workspace
   - Pin skillgrid skills: `--skill test-driven-development --skill ...`
3. Update the local Backlog.md entry with the Kanban task ID (for git archaeology)
4. Track status via `hermes kanban list --board <slug>`
5. On completion: `hermes kanban complete <id> --result "<summary>"`
```

**Verification:**
```bash
skillgrid install --dry-run --agents hermes  # prints planned changes
# After real install:
hermes kanban list  # shows the project board
hermes kanban create "test ticket" --assignee default  # appears in list
```

### Step 4 — Protocol + Cron + Kanban execution (Wave 4)

**4a. Protocol verification:**
- Verify `system_prompt_block()` is injected into the Hermes system prompt
- Verify SOUL.md marker block is present (fallback)
- Verify the protocol text matches `ProtocolMarkdown()` output

**4b. Cron templates (printed as next-steps after install):**
```
Optional: create maintenance cron jobs
  hermes cron create --name skillgrid-sync '0 */4 * * *' 'Run mnemonic index sync and report'
  hermes cron create --name skillgrid-consolidate '0 2 * * *' 'Consolidate memory: mark stale observations'
```

**4c. Kanban execution verification:**
```bash
# Create a ticket via the ticketing skill
# Verify the dispatcher picks it up
hermes kanban list  # status: ready → running → done
hermes kanban show <id>  # shows worker, workspace, completion summary
```

**4d. Round-trip test:**
```bash
hermes -z "Remember that the database uses Postgres 16"
hermes -z "What database do we use?"  # returns Postgres 16
```

## Verification (end-to-end)

```bash
# 1. Install
skillgrid install --agents hermes

# 2. Memory round-trip
hermes -z "Remember that the API rate limit is 100 req/min"
hermes -z "What is the API rate limit?"  # returns 100 req/min

# 3. MCP tools
hermes mcp list  # skillgrid-mnemonic enabled
hermes mcp test skillgrid-mnemonic  # exits 0, shows code_*, web_* tools

# 4. Skills
hermes -z "What skills do you have?"  # lists skillgrid skills

# 5. Kanban
hermes kanban list  # shows project board
hermes kanban create "Implement feature X" --assignee default
hermes kanban list  # shows the task in ready/running

# 6. No regression
skillgrid install --agents opencode  # still works
cd mnemonic && go test ./...  # all pass
cd skillgrid-cli && go test ./...  # all pass
```

## Out of Scope (implementation)

- **Gbrain learnings (v2)** — source tiering, trust tiers, dream cycle, synthesis, auto-linking, hybrid search. These improve the mnemonic binary's retrieval quality but are orthogonal to the Hermes integration. See briefing "GBrain Learnings" section.
- **Upstreaming to hermes-agent** — PR to add `plugins/memory/mnemonic/` to the Hermes repo. Follow-up.
- **Persistent `mnemonic serve` connection** — v1 uses `subprocess.run` per call. v2: HTTP connection for sub-millisecond recall.
- **Hermes MemoryProvider `config_schema.py`** — dashboard integration for `hermes memory setup`. v2.
- **Hermes MemoryProvider `cli.py`** — `hermes mnemonic` CLI subcommands. v2.
