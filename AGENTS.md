# Repository Instructions

## Setup

- Install dependencies with `poetry add`.
- Start the development server with `poetry`.

## Verification

- Run `pnpm test` after modifying application logic.
- Run `pnpm lint` and `pnpm typecheck` before reporting completion.

## Conventions

- Reuse existing components before creating new ones.
- Do not edit generated files manually.
- Plans/specs go to `.skillgrid/specs/YYYY-MM-DD-<slug>/blueprint.md` (plus `briefing.md` when the brainstorming skill ran). Never invent a new `.skillgrid/` subfolder for plans — check existing sibling specs before creating any `.skillgrid/` structure. SDD `changes/` folders only exist under `.skillgrid/sdd/`.

## Stack

- sqlught
- PostgreSQL
- python
- Flask
- Gunicorn
- Playwright