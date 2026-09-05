# Interview: 002-mnemonic-identity-and-parity

> User-gate revise via `questioning` (2026-09-05). Spec already written; grill before apply.

## Facts gathered (agent)

- Repo is SDD-initialized; change phase was `spec` with full `tasks.md` + `acceptance.feature`.
- Large parts of the Step Blueprint already exist in tree:
  - `project/resolve.go`: identity binding, ambiguity + `AvailableProjects`, child auto-promote, bounded config, `MNEMONIC_PROJECT`, `SeedID`
  - `service` / MCP: `SearchAllProjects`, `Unify`/`mem_unify`, `mem_pin`/`mem_unpin`, merge-on-seed
  - `memory`: lifecycle columns/migration `008`, pin/expiry, embedding + RRF behind `MNEMONIC_EMBED`
- Focused tests currently green: `go test ./internal/mnemonic/project/ ./internal/mnemonic/memory/` PASS
- Spec↔code tension already visible:
  - Binding write failure: `change.md` Error handling says **abort**; `identityBinding` currently **falls back to seed** and returns ok
  - Ambiguous parent: returns `AmbiguousProjectError` **and** a directory-hash fallback ID (compat wrapper); callers that treat any error as hard-fail (e.g. MCP session start) abort — which matches the live Cursor/MCP failure from `/data/git/AI`
- `change.md` Open questions still say "None" / do not re-interview — this grill reopens that for revise

## Round 1 — asked 2026-09-05

Q1 posture · Q2 binding write fail · Q3 delivery slice

**Answers (user 2026-09-05):** `1, 1, keep all`

## Decisions

| ID | Decision | Choice |
|----|----------|--------|
| D1 | Change posture | **Gap-close** — rewrite punch-list as audit/harden; implement only deltas |
| D2 | Binding write failure | **Abort** — align code to `change.md`; no seed-without-binding |
| D3 | Delivery slice | **Keep 01–04** in this change |
| D4 | Ambiguous parent writes | **Hard abort for writes** — never open/create under directory-hash fallback; require `MNEMONIC_PROJECT` / explicit `project=` |
| D5 | Punch-list shape | **Rewrite tasks** — keep 01–04; gap items + verify-shipped evidence checkboxes |
| D6 | Auto-merge on identity seed | **Keep silent auto-merge** (`MergeProjects` SeedID→canonical) plus `mem_unify` for repairs |

## Round 2 — asked 2026-09-05

Q4 ambiguous writes · Q5 tasks shape · Q6 auto-merge

**Answers (user 2026-09-05):** `ok` → accept recommendations (1, 1, 1)

## Revised design (pending shared-understanding confirmation)

**Posture:** Treat 002 as parity gap-close, not greenfield rebuild. Most of 01–04 already in tree; apply closes semantic gaps and proves acceptance.

**Identity (01):** Clone-private binding in git common-dir stays. On bind write failure → **abort** (remove seed-without-binding). Ambiguous multi-repo parent → **abort writes** with `AvailableProjects`; no store under fallback ID. `MNEMONIC_PROJECT` / explicit `project=` recover. Keep SeedID alias + silent `MergeProjects` on first bind.

**Cross-store (02):** Keep `all_projects` merge + `mem_unify`; verify-shipped + harden error shapes.

**Lifecycle (03) / Embed (04):** Keep existing columns/tools/RRF; verify-shipped against `@step-03`/`@step-04`; only fix deltas found in audit.

**Spec rewrite after confirm:** Update `change.md` Open questions + Error handling (ambiguity writes explicit); rewrite `tasks.md` per D5; adjust `acceptance.feature` if ambiguity/binding failure wording needs tighter “no write” language.

## Shared understanding

**Confirmed** by user `yes` (2026-09-05). Proceed to revise `change.md` / `tasks.md` / `acceptance.feature`.
