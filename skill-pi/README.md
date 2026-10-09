# skill-pi

Pi coding-agent extension module for Skillgrid.

## Pi version

- **Minimum pi version for `piConfig.configDir`:** 1.1.0
- **Exact pin:** `@earendil-works/pi-coding-agent@1.1.0`

## Install / sync flow

1. The Go installer copies `skill-pi/` into `~/.skill-pi/`.
2. Runs `npm ci` in `~/.skill-pi/` (pinned via shrinkwrap; no vendoring).
3. Materializes `<project>/.skill-pi/` and the shim.
4. Puts `skill-pi` on `PATH`.

## Test

```
node --test skill-pi/version.test.js
```
