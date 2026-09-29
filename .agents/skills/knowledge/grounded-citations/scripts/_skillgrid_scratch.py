"""Resolve the skillgrid scratch dir for standalone skill scripts.

Skill scripts may run outside the skillgrid process (system Python, CI)
where no project context is importable. The default citation ledger lives
at ``<project>/.skillgrid/sdd/citations/ledger.json`` (gitignored scratch).

Project root resolution: walk up from the current working directory to the
nearest dir containing ``.skillgrid/`` (a skillgrid project) or ``.git``;
fall back to the cwd itself.
"""

from __future__ import annotations

import os
from pathlib import Path


def project_root() -> Path:
    cur = Path.cwd().resolve()
    for cand in (cur, *cur.parents):
        if (cand / ".skillgrid").is_dir() or (cand / ".git").exists():
            return cand
    return cur


def project_scratch() -> Path:
    """Return ``<project>/.skillgrid/sdd`` (created on first ledger write)."""
    return project_root() / ".skillgrid" / "sdd"


def resolve_ledger_default() -> Path:
    return project_scratch() / "citations" / "ledger.json"
