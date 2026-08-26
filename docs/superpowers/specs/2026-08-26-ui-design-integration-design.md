# UI design tool integration — design

> **STATUS: DRAFT (2026-08-26)**

**Plan:** *(not yet written)*

**Related:** [2026-08-26-idd-bdd-design.md](2026-08-26-idd-bdd-design.md), [02-usage.md](../../02-usage.md), [webapp-testing skill](../../config.d/skills/webapp-testing/), [NOTE.md](../../NOTE.md)

## Summary

Integrate **agent-native UI design** into skillgrid: a primary **design skill + CLI**, a repo-level **`DESIGN.md` standard**, and hooks into the IDD/BDD workflow before frontend apply. Visual design stays in the **`docs/` zone** (or repo-root `DESIGN.md` linked from AGENTS.md); implementation stays in application code.

**Proposed primary tool:** [Impeccable](https://github.com/pbakaus/impeccable) (skill + optional CLI via managed npm). **Optional extractor:** [SkillUI](https://github.com/amaancoderx/skillui) for bootstrapping `DESIGN.md` from an existing site or codebase.

## Problem

Agents building UI without design constraints produce generic “AI slop” — purple gradients, card nesting, default Inter, no motion system. skillgrid already ships **Playwright** and **webapp-testing** for verification, but nothing that:

- Captures **brand, tokens, and anti-patterns** before coding
- Provides **critique / audit / polish** commands during apply
- Links visual intent to **`docs/superpowers/specs/*-design.md`** (technical) without conflating the two

`docs/NOTE.md` lists many design tools (taste-skill, impeccable, open-design, penpot, kombai) with no default or file standard.

## Goal

After `skillgrid install` on a UI-capable project:

1. One **primary design skill** is registered (Impeccable).
2. Repo has **`DESIGN.md`** (visual system) — separate from IDD `*-design.md` (technical spec).
3. UI features follow: **brainstorm → shape/craft (Impeccable) → IDD spec → BDD (optional) → TDD apply → impeccable audit/polish → Playwright verify**.
4. No runtime **`npx`** — Impeccable/SkillUI installed via `tools.yaml` + `skills` CLI (same as superpowers).

## Non-goals

- Replacing Figma/Penpot for human design teams as primary workflow (optional export target only)
- Kombai or other closed SaaS as skillgrid default
- Merging visual `DESIGN.md` into `docs/superpowers/specs/*-design.md` (cross-link, don’t collapse)
- Auto-running design tools on every `skillgrid install` (project opt-in)
- Design artifacts in the same commit as application code (zone-guard: `docs/` vs code)

---

## Tool evaluation & recommendation

### What “UI design integration” means here

| Layer | Purpose | Example |
|-------|---------|---------|
| **Design system doc** | Persistent tokens, typography, anti-slop rules | `DESIGN.md` |
| **Design skill** | Agent commands: shape, critique, audit, polish | Impeccable `/impeccable craft` |
| **Extractor** | Bootstrap DESIGN.md from existing UI | SkillUI `skillui --dir .` |
| **Design workspace** | Prototypes, decks, visual iteration in browser | Open Design |
| **Human design tool** | Pixel-perfect mockups, design handoff | Penpot, Figma |

skillgrid optimizes for **agent-in-the-loop** coding (Kilo, OpenCode), not a design department replacing IDE flow.

### Candidates compared

| Criterion | **Impeccable** | **Taste Skill** | **SkillUI** | **Open Design** | **Penpot / Kombai** |
|-----------|----------------|-----------------|-------------|-----------------|---------------------|
| **Type** | Skill + CLI + detectors | Skill pack | CLI extractor → skill folder | Local daemon + web UI | Human design apps |
| **DESIGN.md** | Native (`/impeccable init`, `document`) | Stitch variant exports DESIGN.md | Generates DESIGN.md from crawl | Design systems in OD registry | Export only |
| **Agent commands** | 23 (`craft`, `audit`, `polish`, …) | Brief inference, dials | None (output is files) | Delegates to host CLI | N/A |
| **Quality gates** | 59 deterministic detector rules | Pre-flight checklist | Static analysis only | Skill-driven critique | Manual review |
| **Kilo / OpenCode** | Yes ([install providers](https://github.com/pbakaus/impeccable)) | Via `skills add` | Claude-oriented; folder-based | OpenCode adapter built-in | N/A |
| **Live browser** | Yes (pairs with Playwright) | GSAP / motion focus | Playwright in ultra mode | iframe preview | N/A |
| **Install** | `skills` + optional npm CLI | `skills add` | npm `skillui` package | `pnpm` monorepo / desktop | Separate product |
| **Overlap** | — | High with Impeccable (anti-slop) | Complements (bootstrap) | Different layer (workspace) | Complementary handoff |
| **skillgrid fit** | **Primary skill** | Pick one anti-slop skill | Optional bootstrap step | v2 opt-in workspace | Document only |

### Recommendation

| Role | Tool | Rationale |
|------|------|-----------|
| **Primary (v1)** | **[Impeccable](https://github.com/pbakaus/impeccable)** | Mature design vocabulary; writes `DESIGN.md`; audit/critique/polish; detector rules; OpenCode/Kilo; complements existing Playwright stack |
| **Bootstrap (v1 optional)** | **[SkillUI](https://github.com/amaancoderx/skillui)** | Static extract of tokens/components from URL or repo → seed `DESIGN.md` — install `skillui` via `tools.yaml`, **not `npx skillui`** |
| **Defer (v2)** | **[Open Design](https://github.com/nexu-io/open-design)** | Full design workspace (prototypes, decks, 72 design systems) — heavy; use when team wants OD UI + Kilo/OpenCode as engine |
| **Do not default** | **Taste Skill** | Overlaps Impeccable; choose one anti-slop skill to avoid conflicting rules |
| **Do not default** | **Penpot, Kombai** | Human-first; no skillgrid install path v1 |
| **Keep** | **webapp-testing + Playwright** | Verification after impeccable polish — already in skillgrid |

**Decision (proposed):**

```
Primary design skill:  impeccable (via config.d/skills.yaml)
Optional CLI:          skillui in tools.yaml
Repo visual standard:  DESIGN.md (see below)
IDD technical spec:    docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md (unchanged)
```

**Why Impeccable over Taste Skill?**

Both fight AI slop ([Taste Skill](https://github.com/Leonxlnx/taste-skill) vs [Impeccable](https://github.com/pbakaus/impeccable)). Impeccable adds **deterministic detectors**, explicit **audit/polish** pipeline, and first-class **DESIGN.md** / PRODUCT.md setup — better aligned with skillgrid’s verification-heavy IDD/BDD/TDD stack.

**Why SkillUI alongside Impeccable?**

SkillUI does not replace Impeccable — it **extracts** an existing design system into markdown/tokens when brownfield UI work starts. Run once per project (or when cloning a reference site), then Impeccable maintains and audits.

---

## Two design files (do not merge)

| File | Zone | Purpose | When |
|------|------|---------|------|
| **`DESIGN.md`** | Repo root or `docs/design/DESIGN.md` | **Visual system** — colors, type, spacing, motion, components, anti-patterns | Long-lived; project lifetime |
| **`docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`** | `docs/` | **Technical spec** — requirements, architecture, API, Gherkin | Per change (IDD/SDD) |

**Cross-link rule:** IDD `-design.md` MUST link to `DESIGN.md` for UI features and MUST NOT duplicate token tables. Use a `## UI scope` section for change-specific screens/states only.

---

## `DESIGN.md` standard

### Location

| Project type | Path |
|--------------|------|
| Default | **`DESIGN.md`** at repository root (Impeccable convention) |
| Monorepo / docs umbrella | **`docs/design/DESIGN.md`** + link from root `AGENTS.md` |
| Forbidden | Inside `docs/superpowers/specs/` (that namespace is IDD technical) |

Register path in project `AGENTS.md`:

```markdown
## Visual design
- Source of truth: DESIGN.md (or docs/design/DESIGN.md)
- Before UI apply: read DESIGN.md; run /impeccable shape or craft for new surfaces
- Before promote: /impeccable audit + Playwright smoke on changed routes
```

### Required sections

Every `DESIGN.md` MUST include these headings (empty sections allowed with `TBD` until `/impeccable init`):

```markdown
# Design System

> **STATUS:** draft | active | deprecated
> **Last updated:** YYYY-MM-DD
> **Impeccable:** run `/impeccable document` to refresh from code

## Product context
One paragraph: who uses this UI and what job it does. Link PRODUCT.md if present.

## Brand & voice
- Personality (3–5 adjectives)
- Anti-references (what we are NOT — e.g. "no purple gradient hero", "no nested card stacks")

## Color
| Token | Value | Usage |
|-------|-------|-------|
| `--color-bg` | … | Page background |
| `--color-fg` | … | Primary text |
| `--color-accent` | … | Primary actions |
| … | | |

## Typography
| Role | Font | Size / weight | Notes |
|------|------|---------------|-------|
| Display | … | … | |
| Body | … | … | |
| Mono | … | … | Code only |

## Spacing & layout
- Base unit (e.g. 4px grid)
- Max content width
- Breakpoints (if responsive)

## Motion
- Default duration / easing
- Reduced-motion rule (`prefers-reduced-motion`)
- What animates vs what must not

## Components
| Component | Location in codebase | Notes |
|-----------|----------------------|-------|
| Button | `src/components/Button.tsx` | Variants: primary, ghost |
| … | | |

## Accessibility
- Target: WCAG 2.2 AA (or stricter)
- Focus ring rule
- Minimum touch target

## Implementation stack
- Framework (React, Vue, …)
- CSS approach (Tailwind, CSS modules, …)
- Component library (if any) — and what to customize vs avoid

## Verification
- `/impeccable audit` before merge on UI changes
- Playwright routes: (list critical paths)
- BDD: link to Gherkin in `-design.md` when applicable
```

### Optional sections

- `## Reference URLs` — inspiration (not copy-paste)
- `## Design debt` — known inconsistencies to fix
- `## Export` — Penpot/Figma file links for humans

### Feature-level UI in IDD `-design.md`

For UI changes, add **`## UI scope`** to the technical design file:

```markdown
## UI scope

**Visual system:** [DESIGN.md](../../DESIGN.md)

### Screens / routes
- `/settings/profile` — add avatar upload

### States
- Empty, loading, error, success

### Acceptance (BDD)
```gherkin
Feature: Profile avatar
  Scenario: User uploads a valid image
    …
```

### Impeccable checklist
- [ ] `/impeccable shape` approved
- [ ] `/impeccable audit` clean
```

Gherkin stays in `-design.md`; tokens stay in `DESIGN.md`.

---

## Workflow integration

### Command order (UI feature)

Extends [02-usage.md](../../02-usage.md) IDD + BDD + TDD:

| Step | Invoke | Zone | Output |
|------|--------|------|--------|
| 1 | `brainstorming` | dialogue | scope |
| 2 | `/impeccable shape` or `craft` | `docs/` + preview | UI direction (human approves) |
| 3 | Update **`DESIGN.md`** if new tokens/components | `docs/` or root | visual system |
| 4 | `idd-workflow` | `docs/` | `*-design.md` with `## UI scope` + Gherkin |
| 5 | BDD extract/red (if enabled) | `docs/` | acceptance |
| 6 | `test-driven-development` | code | components |
| 7 | `/impeccable audit` + `/impeccable polish` | code | fixes |
| 8 | Playwright / `webapp-testing` | code | visual smoke |
| 9 | `verification-before-completion` | — | evidence |
| 10 | Promote IDD artifacts | `docs/` | STATUS |

**Zone rule:** steps 2–4 touch `DESIGN.md` and `docs/superpowers/` — commit docs before UI code (same as IDD zone-guard).

### Pairing with skillgrid tools

| Tool | Role in UI pipeline |
|------|---------------------|
| **Impeccable** | Shape, critique, audit, polish, anti-slop detectors |
| **SkillUI** | One-time extract → seed DESIGN.md |
| **Playwright / webapp-testing** | Post-implementation verification |
| **BDD / Gherkin** | User-visible behavior in `-design.md` |
| **Open Design (v2)** | Optional prototype/deck iteration before commit to repo |

---

## config.d deliverables (proposed)

| File | Change |
|------|--------|
| `config.d/skills.yaml` | Add Impeccable skill (repo TBD: submodule or `pbakaus/impeccable` via skills CLI) |
| `config.d/tools.yaml` | Optional: `skillui` package (pinned); Impeccable CLI if not skill-only |
| `config.d/templates/DESIGN.md` | Canonical template (copy on project init) |
| `config.d/skills/ui-design-workflow/` | Thin skill: when to read DESIGN.md, impeccable commands, zone rules |
| `config.d/AGENTS.md` | Marker block: UI features require DESIGN.md + audit |

**Install policy:** register skills via `~/.skillgrid/npm/node_modules/.bin/skills add …` — no `npx skills`, no `npx impeccable`.

---

## Bootstrap flows

### Greenfield UI

1. Copy `config.d/templates/DESIGN.md` → repo `DESIGN.md`
2. Run `/impeccable init` in agent session (fills PRODUCT.md + DESIGN.md)
3. Proceed with IDD proposal/spec for first feature

### Brownfield (match existing site)

1. `skillui --dir . --mode ultra` (binary from skillgrid npm prefix)
2. Merge generated `DESIGN.md` into project standard template
3. Run `/impeccable document` to reconcile with codebase
4. IDD change specs reference merged `DESIGN.md`

### Redesign

1. `/impeccable shape` or Taste-style audit **not** used — use `/impeccable critique` + redesign protocol
2. Update `DESIGN.md` STATUS; link ADR if brand pivot

---

## Success criteria

1. Single primary design skill (Impeccable) in default skillgrid bundle.
2. Documented **`DESIGN.md` standard** distinct from IDD `*-design.md`.
3. UI workflow in `02-usage.md` references impeccable + Playwright gates.
4. No `npx` in install or operator docs for design tools.
5. Zone-guard: `DESIGN.md` under `docs/design/` or committed in docs-only commit when root file is treated as spec adjunct (prefer `docs/design/DESIGN.md` for strict zone purity).

---

## Open questions

1. **Root vs `docs/design/DESIGN.md`:** Impeccable defaults to root — accept root exception to zone-guard or always use `docs/design/`?
2. **Impeccable install path:** git submodule in `config.d/` vs `skills add` from GitHub?
3. **Open Design v2:** document as optional desktop/daemon install separate from skillgrid CLI?
4. **Penpot:** export/import pipeline for human designers — worth a spike?

---

## References

- [Impeccable](https://github.com/pbakaus/impeccable) — recommended primary
- [SkillUI](https://github.com/amaancoderx/skillui) — optional DESIGN.md bootstrap
- [Open Design](https://github.com/nexu-io/open-design) — optional workspace v2
- [Taste Skill](https://github.com/Leonxlnx/taste-skill) — alternative anti-slop (not default)
- [2026-08-26-idd-bdd-design.md](2026-08-26-idd-bdd-design.md) — technical `-design.md` layout
- [intent-driven.dev TDD+BDD+SDD](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/) — macro behavior before micro implementation
