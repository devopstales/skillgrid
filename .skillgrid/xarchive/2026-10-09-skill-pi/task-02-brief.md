# T2 Brief — TASK-070: Launcher CLI + cwd shim + exit-75 respawn loop

## Scope
Implement `skill-pi/bin/skill-pi.js`: env relocation, shim build (rebuild-from-scratch every launch), chdir, spawn (not exec), exit-75 loop (capped at 10), owned subcommands (init/sync/version/--where), trust persistence, PI_SKIP_VERSION_CHECK + --no-skills args.

## Files
- `skill-pi/bin/skill-pi.js` — launcher (CJS, executable, shebang)
- `skill-pi/bin/skill-pi.test.js` — 9 TDD tests

## Key behaviors
- Resolve real project root (walk up to .git, realpath)
- buildShim: rm -rf + mkdir + symlink top-level entries (skip node_modules) + write fixed package.json with piConfig.configDir=".skill-pi"
- spawnPi: PI_CODING_AGENT_DIR + PI_SKIP_VERSION_CHECK=1 + --no-skills --no-prompt-templates --no-themes --extension
- Exit-75 loop: capped at 10 respawns
- --where: 8-key JSON
- Trust: ~/.skill-pi/trust.json keyed by shim realpath, SKILL_PI_TRUST_AUTO=1 for tests
- Export buildShim, spawnPi, resolveRoot for testing

## Acceptance
- Fake child exiting 75 → re-spawned; exiting 0 → not; 11× exit-75 → stops
- Shim package.json has piConfig.configDir = ".skill-pi"
- New file in real root visible through shim on rebuild
- Missing root → non-zero exit, no spawn
- --where emits 8-key JSON
- version prints pinned version, no TUI
- Pi flag forwarding (--session)
- Spawn env + args correct
- Trust: fresh run → prompt/auto-accept → trust.json; second run → no prompt

## Focused test
`node --test skill-pi/bin/skill-pi.test.js`

## SATISFIES
`skill-pi-launcher,cwd-shim,spawn-respawn-loop,owned-subcommands,where-json,trust-first-run`
