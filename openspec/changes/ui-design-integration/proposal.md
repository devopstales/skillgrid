## Why

Agents building UI without design constraints produce generic "AI slop" — purple gradients, card nesting, default Inter, no motion system. Skillgrid already ships Playwright and webapp-testing for verification, but nothing that captures brand/tokens/anti-patterns before coding or provides critique/audit/polish during apply. Integrating a primary design skill (Impeccable) and a `DESIGN.md` standard closes this gap.

## What Changes

- Add Impeccable as the primary design skill (via `config.d/skills.yaml`)
- Define `DESIGN.md` standard (visual system document, distinct from IDD technical spec)
- Add SkillUI as optional bootstrap tool (via `config.d/tools.yaml`)
- Define UI feature workflow: brainstorm → shape/craft → IDD spec → BDD → TDD apply → audit/polish → verify
- Add `config.d/templates/DESIGN.md` canonical template
- Add `config.d/skills/ui-design-workflow/` thin skill for when to read DESIGN.md and run impeccable commands

## Capabilities

### New Capabilities

- `design-skill`: Impeccable skill for shape, critique, audit, polish, anti-slop detectors
- `design-md-standard`: `DESIGN.md` as the visual system document standard
- `workflow-integration`: UI feature workflow integrating design into IDD/BDD/TDD pipeline

### Modified Capabilities

None — UI design integration is a new capability.

## Impact

- **Affected config**: `config.d/skills.yaml` (Impeccable), `config.d/tools.yaml` (SkillUI optional), `config.d/templates/DESIGN.md` (new template)
- **Affected docs**: `02-usage.md` (UI workflow section), `AGENTS.md` (UI features require DESIGN.md + audit)
- **Affected skills**: `brainstorming` (adds impeccable shape step for UI work)
- **Users**: Skillgrid users building UI features
