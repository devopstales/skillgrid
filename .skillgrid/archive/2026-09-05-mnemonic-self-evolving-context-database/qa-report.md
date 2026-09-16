# QA plan: 003-mnemonic-self-evolving-context-database

> Human gate for verify (2026-09-05). Human gate: **waived** 2026-09-05 (chat: `QA waived`).
> Agent gate: **01–05 PASS** (filled in `tasks.md` `### Verification`).

## Scope

Exercise tiered storage, semantic retrieval, explicit commit, and trail CLI on a real local Mnemonic data dir.

## Environments / data

- Build: `go test` / `go run` from `skillgrid-cli/`
- Data: temporary dir or `SKILLGRID_MNEMONIC_DATA_DIR` pointing at a disposable store
- Project id: any test id (e.g. `qa-003`)
- Embeddings: try with `MNEMONIC_EMBED=` (off) and `MNEMONIC_EMBED=1` (hash embedder)

## Happy path

1. Open/create a store (binary or tests already cover schema).
2. Write an L2 markdown file; run `skillgrid migrate --tier --project qa-003 --root <content> --dir <data>`.
3. `mnemonic_commit` via MCP (or service) with title + lessons; confirm L2 file + LTM row; sidecars appear shortly after without blocking the tool return.
4. `semantic_search` default corpus → only LTM hits; response has overview/abstract/full_path, **no** full `content` body.
5. `load_full_details` on a hit path → full markdown.
6. `skillgrid trail recent --project qa-003 --dir <data>` shows query / directories / files / result path.

## Edge

1. `semantic_search` with `corpus=all` includes migrate-backfilled (non-LTM) paths.
2. Embeddings off: search still returns ranked results and records a trail.
3. `trail recent` on empty store → `[]`, exit 0.

## Failure

1. `load_full_details` unknown path → not-found error.
2. `mnemonic_commit` with empty title and empty lessons → error, no LTM row / no partial file.
3. `trail show 99999` → `not-found`, non-zero exit.

## Pass / fail / waive

| Result | Meaning |
|--------|---------|
| **Accept** | All happy + at least one edge + one failure exercised; no Global Constraint break |
| **Fail** | Any happy path broken, or L2 body appears in `semantic_search`, or session end creates LTM |
| **Waive** | Explicit user message: `QA waived` (or equivalent) — archive may proceed without interactive QA |

## How to waive

Reply in-session: **QA waived** (or **QA accepted** after manual checks).

**Human response:** `QA waived` (2026-09-05).
