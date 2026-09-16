<!-- skillgrid:start -->
## Skillgrid

This project is configured with Skillgrid. Project: **{project}**.

Config: `.skillgrid/config.yaml` — read it before running any Skillgrid skill.

### Artifacts

| Artifact | Path |
|----------|------|
| Specs (briefing, blueprint, tasks) | `.skillgrid/specs/` |
| Execution ledger | `.skillgrid/sdd/` (gitignored) |
| PRD | `docs/PRD.md` |
| Architecture | `docs/ARCHITECTURE.md` |
| Domain glossary | `.skillgrid/glossary/` (business.md + technical.md) |
| ADRs | `.skillgrid/adr/` |

**Domain model:** Before designing or implementing, read `.skillgrid/glossary/` for the project's vocabulary and the ADRs in `.skillgrid/adr/` for the area you're touching. When terms resolve or a hard-to-reverse decision is made, update them via `skillgrid:architectural-decision-records` — the glossary is a glossary and nothing else.

### Issue Tracker

{tracker_line}

### Memory

{memory_line}

### Workflow

`brainstorming` → `writing-blueprints` → `slicing` → `ticketing` → execution (`subagent-execution` or `simple-execution`) → `requesting-code-review` → `receiving-code-review`

Run `skillgrid:onboarding` to update config after stack changes.
<!-- skillgrid:end -->
