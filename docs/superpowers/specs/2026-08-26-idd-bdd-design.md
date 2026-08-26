# IDD + BDD for skillgrid — design

> **STATUS: DRAFT (2026-08-26)**

**Plan:** [2026-08-26-idd-bdd.md](../plans/2026-08-26-idd-bdd.md)

## Summary

Ship **Intent-Driven Development (IDD)** and **Behavior-Driven Development (BDD)** by augmenting [superpowers](https://github.com/obra/superpowers) — vendored skills, agent rules, and a zone-guard hook. No OpenSpec CLI, no `openspec/` tree, no schema bundles.

All agent-facing artifacts live under **`docs/`**: superpowers IDD chain, glossary, and acceptance tests. Application source stays outside `docs/`. Zone-guard treats the whole `docs/` tree as the specs zone.

## Background

obra/superpowers uses flat dated files:

```
docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md
docs/superpowers/plans/YYYY-MM-DD-<topic>.md
```

This design extends superpowers with `proposal/` and `adr/`, and adds sibling folders `docs/glossary/` and `docs/acceptance-tests/` so **everything the agent reads before coding** lives in one place.

## Goal

After `skillgrid install`:

1. Agents discover IDD/BDD skills via `skills.paths`.
2. Opted-in projects use the `docs/` layout below.
3. Zone-guard blocks co-editing `docs/` and application code in one uncommitted unit.

## Decisions (locked)

| Topic | Decision |
|-------|----------|
| Platform | Augment superpowers only |
| No OpenSpec | No CLI, no `openspec/`, no schema bundles |
| Docs umbrella | All IDD/BDD/glossary/acceptance artifacts under `docs/` |
| Superpowers layout | Flat four folders: `proposal/`, `specs/`, `adr/`, `plans/` |
| Change identity | Shared `YYYY-MM-DD-<topic>` slug across proposal, specs, plans |
| Spec + design | One file: `specs/YYYY-MM-DD-<topic>-design.md` |
| BDD source of truth | Gherkin in `-design.md` (spec-as-source) |
| BDD runner input | Extracted `docs/acceptance-tests/<topic>.feature` (generated from `-design.md`, not hand-edited) |
| Acceptance project | Runner config + step defs under `docs/acceptance-tests/` |
| Archive | Plan `STATUS: ARCHIVED`; design `STATUS: DECIDED` |
| Zone root | Entire `docs/` (superpowers + glossary + acceptance-tests) |
| TDD | superpowers only |

## Non-goals

- OpenSpec CLI, python/behave, Windows hooks, `skillgrid init` v1
- Nested `changes/` or `capabilities/` trees
- Repo-root `acceptance-tests/` or `glossary/` (both live under `docs/`)
- Hand-maintained `.feature` files as source of truth (derived from `-design.md` only)

## Architecture

```
skillgrid install → skills.paths + zone-guard + AGENTS rules

Agent session
├── brainstorming → idd-workflow → bdd-workflow (opt-in)
├── test-driven-development → apply
├── verification-before-completion → promote
└── zone-guard → docs/ vs code

Target project
└── docs/
    ├── superpowers/{proposal,specs,adr,plans}/
    ├── glossary/
    └── acceptance-tests/
```

## Project layout

```
project/
├── docs/
│   ├── superpowers/
│   │   ├── adr/
│   │   │   └── YYYY-MM-DD-<topic>.md
│   │   ├── proposal/
│   │   │   └── YYYY-MM-DD-<topic>.md
│   │   ├── specs/
│   │   │   └── YYYY-MM-DD-<topic>-design.md    # design + Gherkin (source)
│   │   └── plans/
│   │       └── YYYY-MM-DD-<topic>.md           # tasks; STATUS header
│   ├── requirements/
|   |   ├── project.md                          # PRD - Project Requiremet Document
│   │   ├── business.md                         # BRD - Business Requiremet Document
│   │   └── technical.md                        # TRD - Technical Requiremet Document
│   └── acceptance-tests/
│       ├── <topic>.feature                     # extracted from -design.md
│       ├── cucumber.cjs                        # runner (first BDD apply)
│       ├── extract-gherkin.cjs
│       ├── bdd-effective-paths.cjs
│       ├── steps/                              # step definitions
│       ├── .gherkin-lintrc
│       ├── .extracted/                         # gitignored scratch
│       └── reports/                            # gitignored
└── AGENTS.md
```

Example change **user-auth** (2026-08-26):

```
docs/superpowers/proposal/2026-08-26-user-auth.md
docs/superpowers/specs/   2026-08-26-user-auth-design.md
docs/superpowers/plans/   2026-08-26-user-auth.md
docs/acceptance-tests/    user-auth.feature          ← extracted at test time
```

Repo-level engineering (this repo):

```
docs/superpowers/specs/2026-08-26-idd-bdd-design.md
docs/superpowers/plans/2026-08-26-idd-bdd.md
```

### Why `docs/` umbrella is better

| Aspect | Repo-root scatter | `docs/` umbrella (chosen) |
|--------|-------------------|---------------------------|
| Zone guard | Must list multiple roots | One zone: `docs/` |
| Agent mental model | Specs here, tests there | "Read `docs/` before coding" |
| Glossary placement | Orphan at root | Next to specs it defines |
| Acceptance tests | Look like app code | Clearly documentation-of-behavior |

Superpowers four-folder layout is unchanged — only the parent is `docs/`.

### Folder roles

| Path | Pattern | Role |
|------|---------|------|
| `docs/superpowers/proposal/` | `YYYY-MM-DD-<topic>.md` | Why, capabilities |
| `docs/superpowers/specs/` | `YYYY-MM-DD-<topic>-design.md` | Design + requirements + Gherkin |
| `docs/superpowers/adr/` | `YYYY-MM-DD-<topic>.md` | Durable ADRs |
| `docs/superpowers/plans/` | `YYYY-MM-DD-<topic>.md` | Checkbox tasks |
| `docs/glossary/` | `business.md`, `technical.md` | Domain terms |
| `docs/acceptance-tests/` | `<topic>.feature` + runner | Executable BDD |

### Correlation rule

Same topic slug links proposal, design, plan, and feature:

```
2026-08-26-user-auth  →  proposal, -design.md, plan, user-auth.feature
```

### STATUS headers

| File | Values |
|------|--------|
| `proposal/*.md` | draft → active → superseded |
| `specs/*-design.md` | draft → active → **decided** → superseded |
| `adr/*.md` | proposed → accepted → superseded (immutable after accepted) |
| `plans/*.md` | **active** → **archived** |

Promote: plan → ARCHIVED; design → DECIDED. Runner uses DECIDED designs + active (non-archived) plans.

### BDD: `-design.md` → `.feature`

1. Author Gherkin in fenced blocks inside `-design.md`.
2. Extract to `docs/acceptance-tests/<topic>.feature` on each test run.
3. `gherkin-lint` runs on extracted output; line numbers map back to `-design.md`.
4. Do **not** edit `.feature` by hand — zone-guard allows it in `docs/`, but skills treat edits as a violation (regenerate from design instead).

Optional: commit `.feature` files as derived artifacts for CI visibility; extraction still runs to verify drift.

## Workflows

### IDD (`idd-workflow`)

```
proposal → specs/design → adr → plan → apply → promote
```

Superpowers: `brainstorming` → (IDD) → `test-driven-development` → `verification-before-completion`.

### BDD (`bdd-workflow`, opt-in)

```
-design.md (Gherkin) → extract → acceptance (red) → implement → green → promote
```

Sub-skills: `gherkin-authoring`, `acceptance-test-authoring`, `bdd-git-discipline`, `bdd-zone-check`.

### The two rules

1. Acceptance suite green before promote.
2. Never co-edit `docs/` and application code in one uncommitted unit.

## Skills inventory

| Skill | Path notes |
|-------|------------|
| `idd-workflow` | Four superpowers folders under `docs/superpowers/` |
| `bdd-workflow` | Gherkin in `docs/superpowers/specs/*-design.md` |
| `acceptance-test-authoring` | Runner under `docs/acceptance-tests/` |
| `glossary` | `docs/glossary/` |
| + git-discipline, zone-check, gherkin, adr, c4, grilling | adapt/vend |

## config.d/ deliverables

| Path | Role |
|------|------|
| `config.d/skills/` | Vendored skills |
| `config.d/hooks/zone-guard.sh` | Zone root = `docs/` |
| `config.d/AGENTS.md` | IDD/BDD marker block |

## skillgrid CLI

1. `ensureSkillPaths` — append `~/.skillgrid/config.d/skills`
2. `installIDDBDD` (step 6b) — zone-guard + AGENTS markers

## Testing

| Area | Bar |
|------|-----|
| Zone-guard | `docs/` dirty → deny code; code dirty → deny `docs/` |
| Runner | Extract `-design.md` → `<topic>.feature`; ARCHIVED plan exclusion |
| Integration | Full lifecycle with `docs/` layout |

## Success criteria

1. Single `docs/` zone; no repo-root glossary or acceptance-tests.
2. Superpowers flat four-folder layout preserved under `docs/superpowers/`.
3. `.feature` files derived from `-design.md`, never diverge silently.
4. Superpowers handoffs work end-to-end.

## Open questions

1. Hook wiring kilo vs opencode — spike doc
2. Commit extracted `.feature` files or gitignore (recommend gitignore + extract in CI)

## References

- [obra/superpowers `docs/superpowers/`](https://github.com/obra/superpowers/tree/main/docs/superpowers)
- [intent-driven-dev/skills](https://github.com/intent-driven-dev/skills) (MIT)
- [behavior-driven-template](https://github.com/intent-driven-dev/behavior-driven-template)
- Plan: [2026-08-26-idd-bdd.md](../plans/2026-08-26-idd-bdd.md)
