## Why

Skillgrid skills currently hardcode file paths and assume a fixed repository layout. This makes skills fragile across different project structures and prevents the OpenSpec workflow from being a first-class citizen inside skillgrid's skill system. A dedicated pipeline change establishes path conventions and integrates OpenSpec operations directly into skillgrid skills.

## What Changes

- Define standard path conventions for skillgrid skills (config, plugins, skills, worktrees)
- Add OpenSpec workflow integration points into skillgrid skills (propose, apply, verify, archive)
- Create `skillgrid-pipeline` skill that encapsulates the full OpenSpec lifecycle
- Update existing skills to use the standard paths

## Capabilities

### New Capabilities

- `paths-in-skills`: Standard path conventions for skillgrid skills (config.d/, plugins/, skills/, worktrees/)
- `integrate-openspec-into-skills`: OpenSpec workflow operations embedded in skillgrid skills

### Modified Capabilities

None — this is a new cross-cutting concern.

## Impact

- **Affected skills**: All skillgrid skills gain standardized path references
- **Affected docs**: `docs/03-workflow.md` updated with pipeline documentation
- **Users**: All skillgrid users gain a consistent OpenSpec workflow via skills
