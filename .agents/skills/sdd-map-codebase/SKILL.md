---
name: sdd-map-codebase
description: "Write or refresh an optional brownfield narrative map under docs/skillgrid/codebase/*.md. Use when onboarding an existing codebase, refreshing the map, or sdd-onboard runs map before init."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  family: sdd
---

# SDD Map Codebase

Optional brownfield narrative map. **Skip greenfield.** Primary navigation stays Mnemonic code-index (`code_status` → `code_index` → `code_search` → `code_read`). This map is refreshable context for humans/agents — not a second search index.

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).

## Hard Rules

- Greenfield → exit immediately (no `docs/skillgrid/codebase/` required).
- Prefer Mnemonic for symbol/path lookup; map files hold architecture narrative and entry-point pointers.
- Always cite real paths in backticks. Never paste secrets, tokens, or `.env` values.
- Ask before wiping an existing map (Refresh vs Update vs Skip).

## Workflow

```
[ ] 1. Confirm brownfield (else skip)
[ ] 2. Ensure code index healthy
[ ] 3. Gate existing docs/skillgrid/codebase/
[ ] 4. Write/refresh map docs
[ ] 5. Summarize paths + line counts
```

### 1. Confirm brownfield

No application source → skip. User-forced map on empty repo → write a one-line `README.md` under `codebase/` stating "greenfield; map N/A" and stop.

### 2. Code index first

1. `code_status` — if stale/empty, `code_index`
2. Use `code_search` / `code_read` (and light Glob) to gather evidence
3. Parallel mapper agents OK for large repos via `dispatching-parallel-agents`

### 3. Existing map gate

If `docs/skillgrid/codebase/` exists:

1. **Refresh** — replace docs
2. **Update** — only named files
3. **Skip** — keep as-is

### 4. Write docs

Create `docs/skillgrid/codebase/` and write lean markdown (adapt titles; omit empty topics):

| File | Content |
|---|---|
| `STACK.md` | Languages, runtimes, frameworks, key deps, config entrypoints |
| `ARCHITECTURE.md` | Pattern, layers, data flow, main entry points |
| `STRUCTURE.md` | Directory layout, naming, where features live |
| `TESTING.md` | Runner, layers, how to run tests (optional if covered in config) |
| `CONCERNS.md` | Debt, fragile areas, security hotspots (optional) |

Each doc: short sections, evidence-backed paths, last-mapped note (date or HEAD sha) at top.

### 5. Summary

List created/updated paths and approx line counts. Remind: navigate with `code_*`, use map for orientation.

## Gotchas

- Do not duplicate full `config.yaml` stack tables — point at config; map adds *where code lives*.
- Do not promote change-scoped `research.md` into `codebase/` by default.
- Incremental remap: if user names path prefixes, scope exploration to those prefixes only.
