## Context

Skillgrid optimizes for agent-in-the-loop coding (Kilo, OpenCode). Agents building UI without design constraints produce generic results. The existing stack (Playwright + webapp-testing) covers verification but not design capture or critique. Impeccable provides a mature design vocabulary, writes `DESIGN.md`, and has audit/critique/polish commands. SkillUI can bootstrap `DESIGN.md` from an existing site/codebase.

## Goals / Non-Goals

**Goals:**
- Single primary design skill (Impeccable) in default skillgrid bundle
- Documented `DESIGN.md` standard distinct from IDD `*-design.md`
- UI workflow in `02-usage.md` references impeccable + Playwright gates
- No `npx` in install or operator docs for design tools
- Zone-guard: `DESIGN.md` under `docs/design/` or committed in docs-only commit

**Non-Goals:**
- Replacing Figma/Penpot for human design teams
- Kombai or other closed SaaS as skillgrid default
- Merging visual `DESIGN.md` into `docs/superpowers/specs/*-design.md`
- Auto-running design tools on every `skillgrid install` (project opt-in)
- Design artifacts in the same commit as application code

## Decisions

### 1. Primary design tool: Impeccable over Taste Skill

**Decision:** Impeccable as primary design skill.

**Alternatives considered:**
- Taste Skill — rejected: overlaps Impeccable (anti-slop); choosing one avoids conflicting rules
- SkillUI as primary — rejected: it's an extractor, not a design skill
- Open Design — deferred to v2: heavy workspace, not a v1 default

**Rationale:** Impeccable adds deterministic detectors, explicit audit/polish pipeline, and first-class `DESIGN.md` / PRODUCT.md setup — better aligned with skillgrid's verification-heavy stack.

### 2. Two design files (do not merge)

**Decision:** `DESIGN.md` (visual system, long-lived) is separate from `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md` (technical spec, per change).

**Alternatives considered:**
- Single combined file — rejected: conflates visual and technical concerns
- `DESIGN.md` only — rejected: loses per-change technical design traceability

**Rationale:** Cross-link rule: IDD `-design.md` links to `DESIGN.md` for UI features and does NOT duplicate token tables.

### 3. SkillUI as optional bootstrap

**Decision:** SkillUI installed via `tools.yaml` for one-time extraction, not `npx`.

**Alternatives considered:**
- `npx skillui` — rejected: violates no-npx policy
- Skip extraction — rejected: brownfield projects need a starting point

**Rationale:** SkillUI complements Impeccable (extract → Impeccable maintains/audits). Install via `tools.yaml` matches existing convention.

## Risks / Trade-offs

- **Impeccable install path** -> Document as git submodule or `skills add` from GitHub
- **Zone-guard conflict** -> Prefer `docs/design/DESIGN.md` for strict zone purity
- **Two design files confuse agents** -> Cross-link rule + AGENTS.md marker block
- **SkillUI only Claude-oriented** -> Verify Kilo/OpenCode compatibility during implementation

## Migration Plan

1. Add Impeccable to `config.d/skills.yaml`
2. Create `config.d/templates/DESIGN.md` with required sections
3. Define UI feature workflow in `02-usage.md`
4. Add AGENTS.md marker block for UI features
5. Optionally add SkillUI to `config.d/tools.yaml`

## Open Questions

1. **Root vs `docs/design/DESIGN.md`:** Impeccable defaults to root — accept root exception to zone-guard or always use `docs/design/`?
2. **Impeccable install path:** git submodule in `config.d/` vs `skills add` from GitHub?
3. **Open Design v2:** document as optional desktop/daemon install separate from skillgrid CLI?
4. **Penpot:** export/import pipeline for human designers — worth a spike?
