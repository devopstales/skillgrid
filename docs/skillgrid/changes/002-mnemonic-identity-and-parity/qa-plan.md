# QA plan: 002-mnemonic-identity-and-parity

> Agent verify round 1 (2026-09-05). Human gate required before archive.
> Agent gate: **01 PASS WITH WARNINGS** · **02–04 FAIL** (acceptance gaps) → re-enter `sdd-apply`.

## Purpose

Manual checks that automated tests cannot fully own (multi-repo agent cwd, live MCP session, permission-denied `.git`).

## Environments

| Env | Notes |
|-----|--------|
| Dev checkout | `/data/git/AI/skillgrid` on a machine with git + `skillgrid` MCP |
| Multi-repo parent | e.g. `/data/git/AI` (many child repos) |
| Clean clone / linked worktree | optional second checkout of same repo |

No special accounts. Optional: `MNEMONIC_EMBED=1` only if an embedder endpoint is configured.

## Happy paths

1. From repo root, `mem_session_start` + `mem_save` + `mem_search` land in one stable project id.
2. Linked worktree of the same clone resolves the same project id (`mem_current_project`).
3. With two project stores seeded, `mem_search(all_projects=true)` returns hits from both (after apply closes 02.x gaps).
4. `mem_pin` / `mem_unpin` change what `mem_context` surfaces first.

## Edge / failure

1. **Ambiguous parent:** cwd=`/data/git/AI` (or similar) → session/save **aborts** with `AvailableProjects`; no new `*-########.sqlite` under `~/.skillgrid/mnemonic/`. Recover with `MNEMONIC_PROJECT=<candidate>`.
2. **Binding write fail:** if `.git` is not writable, resolve aborts (no seed-without-binding). Prefer a disposable test repo with `chmod a-w .git`.
3. **`tool_name`:** after apply implements it, save with provenance and confirm it is stored/returned.
4. **`MNEMONIC_EMBED`:** flag off → keyword-only; flag on without embedder → degrade without hard failure.

## Pass / fail / waive

| Outcome | Criteria |
|---------|----------|
| **Pass** | All happy + edge checks match expected; no unexpected store files under ambiguous cwd |
| **Fail** | Silent directory-hash store created from multi-repo parent; binding soft-fallback returns; memories still scatter across ids for same clone |
| **Waive** | Human explicitly waives a named check in chat or on the change (e.g. embedder unavailable in CI) — record waiver text in `tasks.md` Verification notes |

## Archive gate

Do **not** archive until:

1. Every step Verdict is `PASS` or `PASS WITH WARNINGS`
2. No open `- [ ]` under `### Tasks`
3. This QA plan is **accepted** or **explicitly waived** by a human
