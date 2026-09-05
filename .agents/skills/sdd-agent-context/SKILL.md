---
name: sdd-agent-context
description: "Write or refresh the Skillgrid sentinel block in AGENTS.md and one-line pointers in other harness files. Use when onboarding, refreshing agent context, or fixing duplicated project facts across harness configs."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  family: sdd
---

# SDD Agent Context

Keep one cross-agent SoT: **full** Skillgrid block in `AGENTS.md`. Cursor / OpenCode / Kilo / VS Code (and CLAUDE.md / GEMINI.md secondaries) get a **one-line pointer** only. Never duplicate project facts (stack, tracker, rules) into harness files.

Payload: [`../_shared/agent-config/block.md`](../_shared/agent-config/block.md).  
Target matrix: [`../_shared/agent-config/README.md`](../_shared/agent-config/README.md).  
Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).

## Hard Rules

- Full block → `AGENTS.md` (create if missing and user agrees; default suggestion AGENTS.md).
- Other harness files → one-line pointer inside secondary sentinels only.
- Idempotent upsert via `<!-- skillgrid-sdd:start -->` … `<!-- skillgrid-sdd:end -->`.
- Workflow line must be: `` `onboard → propose → spec → apply ⇄ verify → archive` ``.
- Registry line is **optional** — omit or mark optional; never invent a registry gate.
- Do not copy `config.yaml` facts into the block beyond the pointers in `block.md`.

## Workflow

```
[ ] 1. Resolve primary (AGENTS.md) and secondaries present
[ ] 2. Fill placeholders from config / tracker
[ ] 3. Upsert full block into AGENTS.md
[ ] 4. Upsert one-line pointers into other harness files
[ ] 5. Confirm no duplicate full blocks
```

### 1. Resolve targets

- Primary: `AGENTS.md` (preferred). If absent, ask before creating.
- Secondaries (if present): `.cursor/rules` pointers only if the project already uses root harness files — prefer root `CLAUDE.md`, `GEMINI.md`, and any documented Cursor/OpenCode/Kilo/VS Code agent entry files the repo already has. Do not spray new harness files.

### 2. Render payload

Copy `block.md` exactly. Fill `{tracker-line}` (and `{registry}` only if the registry file exists — else drop or mark optional). Ensure workflow + entry lines match v4 (`use-skillgrid`, onboard path).

### 3. Upsert AGENTS.md

1. Find `<!-- skillgrid-sdd:start -->`
2. Found → replace through `<!-- skillgrid-sdd:end -->`
3. Missing → append at end
4. Never append a second full block

### 4. Secondaries — one line only

```markdown
<!-- skillgrid-sdd:start (secondary) -->
SDD config lives here: `AGENTS.md` → `## Agent skills`. Do not duplicate that block.
<!-- skillgrid-sdd:end (secondary) -->
```

If a secondary still holds a full old block, replace it with this pointer.

### 5. Verify

Grep for `skillgrid-sdd:start` — exactly one full block (in AGENTS.md); others secondary-only.

## Gotchas

- Two full blocks = two SoTs that drift — always demote extras to pointers.
- Do not put stack/testing/rules prose in harness files; point at `docs/skillgrid/config.yaml`.
- Init may have written an older workflow string — this skill refreshes to the v4 line.
