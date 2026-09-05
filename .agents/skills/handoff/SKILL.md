---
name: handoff
description: Peel an out-of-scope problem that appeared mid-work into a compact brief for a subagent or separate session. Use when a side issue must leave the current change context clean — not for closing SDD, archive, or session end.
argument-hint: "What out-of-scope problem should the brief cover?"
---

# handoff

Peel a **side problem** out of the current change so this context stays on-scope. The brief feeds a **subagent** or a separate session — not archive, not session close (those use `sdd-archive` + `mnemonic-memory`).

Save to OS temp (`/tmp` on Linux/macOS, `%TEMP%` on Windows). **Never** under `docs/skillgrid/changes/` or the workspace change folder.

## Required sections

1. **Context** — The spun-off problem only: what appeared mid-work, why it is out of scope for the current change, and what the receiver must understand to start.
2. **Current state** — Branch, commit, worktree, or session details relevant to the spun-off work.
3. **Key decisions** — Constraints already fixed that the receiver must obey (not the parent change's full history).
4. **Artifacts** — Link paths/URLs; do not duplicate content:
   - Parent change (orientation only): `docs/skillgrid/changes/<NNN-slug>/`
   - Issues, code paths, research the receiver needs
5. **Suggested skills** — Skills the receiver should load (e.g. `investigate`, `debugging`, `design-spike`, `issue-creation`). If starting fresh: `mnemonic-memory` + `mem_session_start`.
6. **Next steps** — Concrete ordered actions for the **spun-off** problem only.
7. **Gotchas** — Non-obvious traps that would waste the receiver's time.

## Rules

- Stay on the **out-of-scope problem** — do not dump the parent session history.
- Do not duplicate specs, plans, ADRs, issues, or diffs — reference them.
- Redact secrets (API keys, tokens, PII).
- Be concise; prefer bullets.
- After writing the brief, resume the parent change without the side thread.

## Mnemonic

If peeling surfaced findings the **parent** change should keep, `mem_save` them on the parent session before dispatching. The receiver starts its own session and reads the temp brief.
