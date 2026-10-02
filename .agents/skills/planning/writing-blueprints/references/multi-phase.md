## Multi-Phase Change Folders

When a blueprint spans multiple phases (e.g. a 4-phase product rollout), split it into one change folder per phase under `.skillgrid/specs/`. Each phase folder is a normal change folder. The spec-folder tree is in `_shared/rules/sdd-structure.md`. There is no `spec.md`.

```
.skillgrid/specs/
  YYYY-MM-DD-<feature>/                ← parent blueprint (index only)
    blueprint.md                       ← header + phases table + shared context link
    shared/context.md                  ← hypothesis, must-haves, global constraints, file structure
  YYYY-MM-DD-<feature>-phase-1/        ← sibling change folder
    briefing.md                        ← phase goal
  YYYY-MM-DD-<feature>-phase-2/
    briefing.md
  ...
```

Rules:
- Phase folders are **siblings** of the parent, not nested inside it
- Each phase folder follows the spec-folder tree: `briefing.md` holds the phase goal; `blueprint.md` and `tasks.md` appear when those phases run for that folder
- The executor reads the phase `briefing.md` plus the parent's `shared/context.md`
- Tasks are **verbatim** — no rewording or code changes when splitting
- Task numbering is **global** (Task 1–N across all phases), not per-phase
- Each phase folder is a valid skillgrid change: independently sliceable, executable, reviewable, archiveable
- The parent `blueprint.md` is the source of truth for the phase table
- `state.yaml` `current_change` points to the **active phase folder**

