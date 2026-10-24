# SDD ledger — plan: .skillgrid/specs/2026-10-09-skill-pi/tasks.md

Branch: release/2
Scope: skill-pi — 20 tickets, two tracers (A: foundation T1–T10, B: swarm T11–T20).
Chain strategy: stacked-to-main. PR 1 (foundation) → PR 2 (swarm).

## Delivery Guard (checked before first dispatch)
- 400-line budget risk: High, Chained PRs: Yes, Chain strategy: stacked-to-main.
- Each work unit names focused test + runtime harness + rollback boundary (tasks.md table). PASS.

## One-way doors (human checkpoint)
- TICKET-01 (TASK-069) ADR-0042 pi dependency pin (Reversibility: one-way) — PASSED (user approved in brainstorming R2 Q8)
- TICKET-11 (TASK-079) goals table + tasks.goal_id migration ADR-0039 (Reversibility: one-way)

## Status
| Ticket | Task | Status | Commit |
|--------|------|--------|--------|
| TICKET-01 | TASK-069 | done | c3ad56fa |
| TICKET-02 | TASK-070 | done | f12fc0f |
| (T2 fix) | — | done | 7dc4e1e (pi-compatible boolean trust) |
| TICKET-03 | TASK-071 | done | 0787c19e |
| TICKET-09 | TASK-072 | pending (wave 3, after T1) | — |
| TICKET-08 | TASK-076 | pending (wave 3, after T2) | — |

## Work Unit Evidence
| Unit | Focused test | Runtime harness | Rollback boundary |
|------|--------------|-----------------|-------------------|
| 1 | `node --test skill-pi/version.test.js` | N/A (file-equality) | `skill-pi/` + ADR file |
| 2 | `node --test skill-pi/bin/skill-pi.test.js` | fake child (stubbed exit codes) | `skill-pi/bin/` |
| 3 | `node --test skill-pi/version.test.js -t config` (3 oracles) | N/A (config data + stdlib helper, no runtime boundary yet — pi reads these at T4) | `skill-pi/{mcp,settings}.json`, `skill-pi/system/`, `skill-pi/lib/base-url.js` |

## Dev-loop state (T1+T2 done, pre-T3)
- `task skill-pi:build` = install (cp pkg.json → ~/.skill-pi + npm install) + node --check + chmod + npm link. Verified: `skill-pi --version` → 1.1.0.
- `task skill-pi:test` = `node --test` in skill-pi. 10/10 green.
- Trust fix (7dc4e1e): recordTrust writes `true`, ensureTrusted checks `=== true` (pi readTrustFile contract).
- Known benign error until T4: `--extension <skill-pi-root>` → "Cannot find module" (skill-pi/extensions/*.ts not created yet) + peerDependencies warning. Exit 0, pi continues.
- Stale `skill-pi-root-trust-*` test shims cleaned from ~/.skill-pi/cwd/.
- Untracked in repo: `.skill-pi/` (project-local, R4 — needs gitignore at T8), `undefined.spawn` (removed).
- Extension fix (2816d17): pi's `--extension <dir>` does a direct jiti `import()` of the dir, resolving `package.json` -> `main` -> `index.js`. Root had no entry → "Cannot find module" + peerDep warning. Added no-op meta-extension factory `skill-pi/index.js` + `"main":"index.js"`. `skill-pi --version` now exits 0, no extension error. T4 replaces the factory body with the five port registrations.
- PeerDep fix (727c532): pi loader warns when a host-provided extension pkg lists @earendil-works/pi-coding-agent in `dependencies` (duplicate module). Moved it to `peerDependencies: {"*"}"; the 1.1.0 pin is now enforced at install time — Taskfile `skill-pi:install` runs `npm install @earendil-works/pi-coding-agent@1.1.0 --save-exact` into `~/.skill-pi`. Skeleton test updated to assert the peerDep contract + pi-version.txt pin. 10/10 green, `skill-pi --version` clean (no warnings). "No models available" warning is expected (no provider configured; T14 base-URL territory).
- T3 (0787c19e): `mcp.json` (mnemonic server, no URL literal), `settings.json` (baseUrl env-resolved + systemPrompt delta/append), `system/SYSTEM.md` (pure delta: zone discipline + commit contract, pointer to AGENTS.md), `system/APPEND_SYSTEM.md` (empty placeholder for Task 20 lead prompt), `lib/base-url.js` (CJS stdlib helper `resolveBaseUrl`, mirrors opencode env resolution). version.test.js +3 `-t config` oracles. 4/4 green; grep A empty, grep C = 0.
- Ruling (T3): brief's *example* pointer line contained the literal `gentle-ai:persona`/`gentle-ai:engram-protocol` markers, which oracle (c) (binding BDD: grep count 0) rejects. Resolved in favor of the oracle: SYSTEM.md references the blocks descriptively while still naming `AGENTS.md`. If literal markers are wanted, oracle (c) must be relaxed (grep the `<!-- -->` HTML-comment delimiters instead). Cost if wrong: negligible — wording-only, no behavior.
- Next: T3 task reviewer (spec + quality). Then T4 (TASK-073 five extension ports), unblocked by T3.
