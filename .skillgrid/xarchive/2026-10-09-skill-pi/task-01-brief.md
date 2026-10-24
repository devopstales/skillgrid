# T1 Brief — TASK-069: skill-pi/ module skeleton + ADR-0042 dependency pin

## Scope
Create the `skill-pi/` top-level module with `package.json` (piConfig identity + exact pi pin), `pi-version.txt`, ADR-0042, README, and version test.

## Files
- `skill-pi/package.json` — name "skill-pi", piConfig { name: "skill-pi", configDir: ".skill-pi" }, bin, exact dep `@earendil-works/pi-coding-agent: 1.1.0`
- `skill-pi/pi-version.txt` — single line: `1.1.0`
- `skill-pi/README.md` — min pi version, pin, install/sync flow
- `skill-pi/version.test.js` — node:test, CJS, asserts version-pair equality + piConfig + bin + ADR index
- `.skillgrid/artifacts/04-adr-0042-skill-pi-dependencies.md` — dependency ADR
- `.skillgrid/ASSUMPTIONS.md` — index ADR-0042 in LOCKED

## Acceptance (R12)
- pi-version.txt content equals package.json dependency string
- Exact version (no ^/~/range)
- ADR-0042 indexed in ASSUMPTIONS.md § LOCKED
- README names min pi version for piConfig.configDir

## Focused test
`node --test skill-pi/version.test.js`

## SATISFIES
`skill-pi-module-skeleton`
