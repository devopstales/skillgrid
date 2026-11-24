# SDD ledger — plan: v1-learnings-port (2026-11-24)

Change: 2026-11-24-v1-learnings-port — port the v1 audit's P0–P3 recommendations + recorded follow-ups into the v2 rewrite (branch `feat/v2-rewrite`, uncommitted).

Audit (2026-11-23): complete — 3 read-only scouts (content gaps / hook+CI gaps / accrued learnings) → P0–P3 report delivered.

P0 ports: complete — `test-driven-development` skill + `strict-tdd.md` canon; 8 lens templates in `test/references/reviewers/`; `handling-findings` / `test-rigor` / `verification-gap` / `test-driven-verification` references.
P0 redirects: complete — ~70 occurrences of 10 retired v1 skill names across ~35 files; `test/SKILL.md` TDD Evidence Audit + term definitions wired.
P1/P2: complete — ship GO/NO-GO (v2.5); `omp/hooks/hooks/session-start.js` (+ installed copy); `scripts/test-hooks.mjs` 44 cases + `check-lens-contract.mjs` + `hooks-ci.yml`.
P3 docs: complete — `docs/06-concepts.md`, `blueprint/references/sketch-phase.md`, `CONTRIBUTING.md`, docs 00/01.

Follow-ups (2026-11-24): complete — 3 drift scripts + 113 tests + test Step-3 wiring (4 real state.yaml drifts fixed on first run); `precommit-guard.js` detached-HEAD branch revived (test flipped, 44/44); mnemonic-code briefing R14 orientation contract + TTL scope naming; research budget overage trimmed.

Verification: all green — hooks 44/44, lens 8/8, drift/lock suites 113/113, state guard `DRIFT: clean`, frontmatter 22/22, size budget 22/22.

Open (user calls): v2 install / `SKILLGRID_ROUTER` for live router injection; cross-ecosystem refs (regression-dog, codebase-design, codebase-inspection) left as-is; commit pending.
