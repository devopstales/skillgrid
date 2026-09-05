---
name: sdd-onboard
description: "Bootstrap Skillgrid SDD in safe order (map → init → agent-context → constraints → domain) for greenfield or brownfield repos. Use when the repo is uninitialized, the user says onboard/bootstrap SDD, or use-skillgrid routes to onboard."
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  family: sdd
---

# SDD Onboard

Stage orchestrator for Skillgrid bootstrap. Run helpers in **safe order**. Does **not** implement product changes, write `change.md`, or start propose/apply.

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).

## Hard Rules

- **Safe order** only — never skip ahead to propose/spec/apply from this skill.
- Greenfield: skip map. Brownfield (existing source): map first when useful.
- Stop with a summary + next step. Do not freestyle a product feature here.
- Reuse helpers; do not reimplement `sdd-init` detection or agent-block rendering.

## Workflow

```
[ ] 1. Classify greenfield vs brownfield
[ ] 2. (Brownfield) sdd-map-codebase — optional; skip if user declines
[ ] 3. sdd-init — detect facts, skeleton, tracker, Mnemonic
[ ] 4. sdd-agent-context — AGENTS.md sentinel; one-line pointers elsewhere
[ ] 5. sdd-constraints — quality bar into config.yaml rules.*
[ ] 6. sdd-domain — glossary stubs under docs/skillgrid/glossary/
[ ] 7. Summary + next (propose or idle)
```

### 1. Classify

- **Greenfield** — little/no application source (empty repo, docs-only, or user says new app).
- **Brownfield** — meaningful existing code to navigate.

Ask once if unclear. Record the choice in the summary.

### 2. Map (brownfield only)

Invoke `sdd-map-codebase`. User may skip. Primary navigation remains Mnemonic `code_*` — the map is narrative, not a second index.

### 3–6. Helpers in safe order

Load and run each skill's checklist in order:

1. `sdd-init`
2. `sdd-agent-context` (refresh even if init already wrote a block — ensure v4 workflow line)
3. `sdd-constraints`
4. `sdd-domain`

If `docs/skillgrid/config.yaml` already exists mid-run, ask before overwriting; prefer refresh of missing pieces only.

### 7. Stop — summary + next

Return:

- Classification (greenfield/brownfield)
- Artifacts created/updated (paths)
- What was skipped
- **Next:** if the user already stated a change → suggest `sdd-propose` (via `use-skillgrid`); else **idle** until they name work

Do not auto-enter propose without a stated change.

## Gotchas

- Gentleman/GSD "onboard" that teaches by shipping a full cycle is **out of scope** — Skillgrid onboard only bootstraps context.
- Registry (`skill-registry.md`) is optional — never block onboard on it.
- Initialized? = `docs/skillgrid/config.yaml` + Skillgrid sentinel in `AGENTS.md`. Not CONTEXT.md / CONSTRAINTS.md / registry.
- After onboard, product work starts at **propose**, not explore-as-top-level-stage.
