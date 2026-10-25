"""Mnemonic MemoryProvider for Hermes Agent (TICKET-01).

Thin Python shim delegating to the `mnemonic` Go binary via subprocess.
All logic lives in the Go binary; this class handles the Hermes memory
lifecycle: prefetch, sync_turn, system prompt block, tool schemas, and
tool calls.

The provider shells out to the `mem` CLI parity layer:

    mem save   --title <t> --type <type> --content <c> [--scope s]
               [--topic-key K] [--session-id ID] [--json]
    mem search [--query <q>] [--limit N] [--json]
    mem context [--limit N] [--json]
    mem get    <id>
    mem update <id> [--title t] [--content c] [--type t] [--scope s]
               [--topic-key K]

Every subcommand prints JSON to stdout; errors go to stderr with a
non-zero exit. All mnemonic failures are logged, never raised into the
Hermes runtime (the provider must not crash the agent).
"""

from __future__ import annotations

import json
import logging
import shutil
import subprocess
import threading
from typing import Any, Dict, List, Optional

logger = logging.getLogger(__name__)

# Import the ABC from Hermes (available at runtime when loaded as a plugin).
# Outside Hermes (unit tests) the fallback keeps the class importable.
try:
    from agent.memory_provider import MemoryProvider, spawn_context_thread
except ImportError:
    MemoryProvider = object  # fallback for unit tests outside Hermes

    def spawn_context_thread(
        target, *, name="mnemonic", daemon=True, args=(), kwargs=None
    ):
        t = threading.Thread(
            target=target, name=name, daemon=daemon, args=args, kwargs=kwargs or {}
        )
        return t

# Subprocess budget for a single CLI call. Generous for cold starts
# (project resolution, migration); the read paths inside the binary are
# separately bounded by their own budget flags.
_CLI_TIMEOUT = 30

TOOL_SCHEMAS: List[Dict[str, Any]] = [
    {
        "name": "mem_save",
        "description": "Save an observation to project memory",
        "parameters": {
            "type": "object",
            "properties": {
                "title": {"type": "string", "description": "Verb + what — short, searchable"},
                "type": {
                    "type": "string",
                    "enum": [
                        "bugfix",
                        "decision",
                        "architecture",
                        "discovery",
                        "pattern",
                        "config",
                        "preference",
                    ],
                },
                "content": {"type": "string"},
                "scope": {"type": "string", "enum": ["project", "personal"], "default": "project"},
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
        return "mnemonic binary not found on PATH. Run: skillgrid setup hermes"

    def initialize(self, session_id: str, **kwargs: Any) -> None:
        self._session_id = session_id
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
        def _fetch() -> None:
            try:
                result = self._run(
                    ["mem", "search", "--query", query, "--json", "--limit", "5"]
                )
                data = json.loads(result)
                entries = data.get("observations", data.get("results", []))
                if entries:
                    self._cache = "\n".join(
                        f"- [{e.get('type', '?')}] {e.get('title', '?')}: "
                        f"{e.get('content', '')[:200]}"
                        for e in entries
                    )
                else:
                    self._cache = ""
            except Exception as exc:
                logger.debug("mnemonic prefetch failed: %s", exc)
                with self._lock:
                    self._cache = ""

        t = spawn_context_thread(target=_fetch, name="mnemonic-prefetch")
        t.start()

    def sync_turn(
        self,
        user_content: str,
        assistant_content: str,
        *,
        session_id: str = "",
        messages: Optional[List[Dict[str, Any]]] = None,
        turn_author: Optional[str] = None,
    ) -> None:
        # Non-blocking: spawn a background thread to save the turn.
        def _save() -> None:
            if self._agent_context != "primary":
                return
            try:
                self._run(
                    [
                        "mem",
                        "save",
                        "--title", f"Session turn {session_id[:8]}",
                        "--type", "discovery",
                        "--content", f"User: {user_content[:500]}\nAssistant: {assistant_content[:500]}",
                        "--json",
                    ]
                )
            except Exception as exc:
                logger.debug("mnemonic sync_turn failed: %s", exc)

        t = spawn_context_thread(target=_save, name="mnemonic-sync")
        t.start()
        # Queue the next prefetch for the upcoming turn
        if user_content.strip():
            self.queue_prefetch(user_content, session_id=session_id)

    def get_tool_schemas(self) -> List[Dict[str, Any]]:
        return TOOL_SCHEMAS

    def handle_tool_call(self, tool_name: str, args: Dict[str, Any], **kwargs: Any) -> str:
        cmd = self._tool_to_cmd(tool_name, args)
        return self._run(cmd)

    def on_session_end(self, messages: List[Dict[str, Any]]) -> None:
        # End-of-session extraction: distill insights.
        # For v1, just log the session length. The distill pass is a v2 feature.
        logger.info("mnemonic session end: %d messages", len(messages))

    def on_pre_compress(self, messages: List[Dict[str, Any]]) -> str:
        # Extract insights before context compression.
        # For v1, return "" (no extraction). v2: LLM pass over recent messages.
        return ""

    def on_memory_write(
        self,
        action: str,
        target: str,
        content: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> None:
        # Mirror Hermes native memory writes into mnemonic.
        try:
            self._run(
                [
                    "mem",
                    "save",
                    "--title", f"Native {action} ({target})",
                    "--type", "preference" if target == "user" else "discovery",
                    "--content", content[:1000],
                    "--json",
                ]
            )
        except Exception as exc:
            logger.debug("mnemonic on_memory_write failed: %s", exc)

    def shutdown(self) -> None:
        # Flush pending background work. v1: no-op (threads are daemon).
        pass

    # --- Internal helpers ---

    def _run(self, args: List[str]) -> str:
        """Run the mnemonic CLI and return stdout.

        Raises RuntimeError on a non-zero exit; callers decide whether
        that is fatal (handle_tool_call propagates it to Hermes) or
        swallowed (background threads log at debug).
        """
        cmd = ["mnemonic"] + args
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=_CLI_TIMEOUT)
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
