# skill-pi

An opinionated distribution of pi (by Mario Zechner) with extensions for memory, code intelligence, and skillgrid session lifecycle. Installing skill-pi installs the pinned pi runtime with it.

## Pi version

- **Exact pin (bundled):** `@earendil-works/pi-coding-agent@1.1.0`
- pi is a bundled dependency — a global install self-contains the runtime under the package's own `node_modules`.

## Install / sync flow

1. `npm i -g skill-pi` (or `npm link` for the dev flow) ships the pinned pi runtime bundled under the package's own `node_modules`.
2. Exposes two bins: `skill-pi` (canonical) and `pi` (distribution pass-through that re-enters the skill-pi launcher, so a bare `pi` runs with skill-pi's packaged extensions).
3. The launcher resolves pi from the package's own `node_modules` (not the global root), so the pinned version always runs.
4. `~/.skill-pi/` holds config/state only — no `node_modules`.

## Test

```
node --test skill-pi/version.test.js
```
