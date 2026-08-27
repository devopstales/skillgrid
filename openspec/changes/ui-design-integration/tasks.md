# Tasks: UI Design Integration

## Epic 1: Design skill integration

- [ ] 1-1 Add Impeccable to `config.d/skills.yaml`
- [ ] 1-2 Add SkillUI to `config.d/tools.yaml` (optional)
- [ ] 1-3 Verify no `npx` in install path for design tools

## Epic 2: DESIGN.md standard

- [ ] 2-1 Create `config.d/templates/DESIGN.md` with all required sections
- [ ] 2-2 Define cross-link rule (IDD `-design.md` links to `DESIGN.md`)
- [ ] 2-3 Define zone-guard rule for `DESIGN.md` commits

## Epic 3: Workflow integration

- [ ] 3-1 Define UI feature workflow order in `02-usage.md`
- [ ] 3-2 Create `config.d/skills/ui-design-workflow/` thin skill
- [ ] 3-3 Add AGENTS.md marker block: UI features require DESIGN.md + audit

## Epic 4: Documentation

- [ ] 4-1 Update `02-usage.md` with UI workflow section
- [ ] 4-2 Document bootstrap flows (greenfield, brownfield, redesign)
- [ ] 4-3 Document pairing with existing tools (Playwright, BDD, webapp-testing)

## Epic 5: Validation

- [ ] 5-1 Run `openspec validate ui-design-integration --type change --strict` before archive
- [ ] 5-2 Archive via `openspec archive ui-design-integration`
