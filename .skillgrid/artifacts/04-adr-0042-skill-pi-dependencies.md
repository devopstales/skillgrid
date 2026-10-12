# ADR-0042: skill-pi npm dependency surface (@earendil-works/pi-*)

---
status: "accepted"
supersedes: none
date: 2026-10-09
---

## Context and Problem Statement

The `skill-pi` module (the Pi coding-agent extension for Skillgrid) needs
third-party npm dependencies to function. The first ticket (TICKET-01)
introduces the module skeleton and pins the primary dependency; later
TUI-focused tickets (TICKET-06) will add `@earendil-works/pi-tui`.

The "no new dependencies without an ADR" rule (enforced by AGENTS.md)
requires this ADR to record the exact pin and the rationale behind it.

Two third-party packages are in scope:

- `@earendil-works/pi-coding-agent` — the Pi coding-agent API; the
  extension imports it directly.
- `@earendil-works/pi-tui` — the Pi TUI rendering layer; added in
  TICKET-06 when the TUI tasks land.

The exact pin and the `piConfig.configDir` minimum-version contract are
shaped around the pinned version. In brainstorming Round 2, Q8, the user
approved "exact pin + documented min-version" as the dependency strategy.

## Considered Options

- **A. Exact pin (`1.1.0`)** — pin `@earendil-works/pi-coding-agent` to
  `1.1.0` with no range operators (`^`, `~`, `>=`, etc.). The installer
  runs `npm ci` against the shrinkwrap, so the byte-for-byte resolved tree
  is deterministic. The `piConfig.configDir` field was confirmed present
  in pi v1.1.0 source; the min-version guard in the installer reads
  `pi-version.txt` and fails if the installed version is below 1.1.0.
- **B. Caret range (`^1.1.0`)** — allows patch and minor upgrades within
  the 1.x line. Simpler to maintain but silent behavior changes are
  possible: a 1.2.0 release could shift the `piConfig` surface or change
  TUI import paths without a major bump. The min-version guard would
  still pass, hiding the drift.
- **C. Vendor the dependency** — copy the pi packages into the repo and
  commit them. No network at install time, but the vendor tree grows with
  every pi release and diverges from upstream. The repo's existing npm
  dependency policy (ADR-0008) uses `node_modules` + shrinkwrap, not
  vendoring.

## Decision Outcome

Chosen option: **A. Exact pin `1.1.0`**, because the installer flow and
the TUI import surface are both shaped around the pinned version, and a
silent minor bump is the single most likely source of drift. The pin is
enforced by `skill-pi/version.test.js`, which asserts the version string
matches `/^\d+\.\d+\.\d+$/` and that `pi-version.txt` equals the
`package.json` dependency string.

`piConfig.configDir` requires pi >= 1.1.0. The minimum-version guard is
documented in `skill-pi/README.md` and the installer reads
`skill-pi/pi-version.txt` to enforce it at install time.

Supply chain posture: pinned dependencies + npm shrinkwrap (no vendoring).
The shrinkwrap is generated at install time by the Go installer running
`npm ci`; the repo does not commit `node_modules` or a local shrinkwrap
file for the `skill-pi` module (the installer produces it in
`~/.skill-pi/`).

The `pi-tui` pin will be added in TICKET-06 with the same exact-pin
strategy when the TUI tasks land.

Option B (caret range) was rejected: the TUI import paths and the
`piConfig` surface are version-sensitive, and a silent minor bump would
pass the min-version guard while changing behavior. The cost of an exact
pin (a manual version bump per pi release) is lower than the cost of
debugging a silent TUI break.

Option C (vendoring) was rejected: it conflicts with the repo's existing
npm dependency policy (ADR-0008 uses shrinkwrap, not vendoring) and adds
a large committed tree that must be re-synced on every pi release.

### Implementation

- `skill-pi/package.json` — `"@earendil-works/pi-coding-agent": "1.1.0"`.
- `skill-pi/pi-version.txt` — `1.1.0` (must equal the package.json
  dependency string).
- `skill-pi/version.test.js` — asserts the exact pin, the
  `pi-version.txt` equality, the `piConfig` fields, and the `bin` entry.

### Consequences

- Good, because the pin is the one-way part: removing it means
  re-validating the min-version guard, the installer `npm ci` flow, and
  the TUI imports. The cost of that re-validation is the price of the
  safety the pin buys.
- Good, because `pi-version.txt` and `package.json` are checked by the
  same test, so a version drift between them (R12↔R13) is caught at
  `node --test` time, not at install time.
- Bad, because every pi release requires a manual bump of both
  `package.json` and `pi-version.txt` (and the min-version guard
  threshold if the release is below 1.1.0). Mitigation: the bump is a
  two-line change; the test catches any mismatch.
- Bad, because a version drift between `pi-version.txt` and
  `package.json` is the single most likely silent failure (R12↔R13).
  Mitigation: `version.test.js` asserts equality; the test runs in the
  QA gate.

### Revisit criteria

Revisit this ADR (write a superseding ADR) when **any** of:

1. Pi ships a 2.x major release — the exact-pin strategy is re-evaluated
   against the 2.x surface; the min-version threshold may need to move
   to 2.0.0.
2. The `pi-tui` pin (TICKET-06) requires a different pinning strategy —
   if `pi-tui` and `pi-coding-agent` have independent release cadences,
   the two pins may need separate version.txt files or a combined
   version manifest.
 3. The repo's npm dependency policy changes (ADR-0008 superseded) —
    if the repo moves to vendoring or a different lock strategy, the
    supply chain posture section of this ADR must be re-validated.

## Supersession: pi is a bundled distribution dependency (2026-10-12)

This section amends the above: it records the shift from *peer
(host-provided) pi* to *bundled exact-dependency pi*, made so a single
global `skill-pi` install ships a working `pi` without a separate host
install. The **exact-pin strategy of this ADR is unchanged** — pi is still
pinned to `1.1.0` and still guarded by `pi-version.txt` + `version.test.js`.
What changes is *how the pinned pi reaches the user*: it is now shipped
inside the `skill-pi` package itself, rather than expected to already exist
on the host.

### What changed

- `@earendil-works/pi-coding-agent` moved from `peerDependencies` to
  `dependencies` (exact `1.1.0`), with `bundleDependencies` so `npm i -g
  skill-pi` materializes pi under the package's own `node_modules`.
  `~/.skill-pi/` is now config/state only — no `node_modules`, no
  `package-lock.json`, no `package.json` there (the shrinkwrap claim in the
  original Decision Outcome above was about the install *target* and no longer
  applies; the lock now lives in the package, not `~/.skill-pi/`).
- A second bin, `bin/pi.mjs`, re-enters the `skill-pi` launcher so a bare
  `pi` carries the Skillgrid extensions/system prompt (modelled on
  omegon-pi's `bin/pi.mjs`). `bin/pi.mjs` is ESM and does a fire-and-forget
  `import('./skill-pi.js')`.
- The launcher's `piBin()` resolves pi in order:
  `SKILL_PI_PI_BIN` (env override) → `<SKILL_PI_ROOT>/node_modules/.bin/pi`
  (the bundled copy a global install places there) → global-root
  `node_modules/.bin/pi` → bare `pi` on PATH. The bundled path wins in a
  global install, so the shipped `pi` is the pinned `1.1.0`.

### Why it is not a contradiction

Option A of this ADR is *"exact pin, never a range"*; that is preserved.
What the original text implied as *"never vendored / host-provided"* is the
part superseded: pi is now a **dependency** of `skill-pi` (exact `1.1.0`)
that npm bundles into the published tarball on `npm i -g`. It is not a
vendored committed copy (Option C is still rejected — pi comes from npm at
install time, not from the repo tree) and it is not a host prerequisite
(anymore). The supply-chain posture is the same: pinned, from npm, verified
by the same `pi-version.txt` equality test.

### Implementation (amendment)

- `skill-pi/package.json` — `dependencies['@earendil-works/pi-coding-agent'] = "1.1.0"`,
  `bundleDependencies` includes it, `bin.pi = "./bin/pi.mjs"`, `bin['skill-pi']`
  unchanged; `files` ships `skills/` and `prompts/`.
- `skill-pi/bin/pi.mjs` — ESM re-entry shim.
- `skill-pi/bin/skill-pi.js` — `piBin()` bundled-first resolution.
- `skill-pi/version.test.js` — now asserts the bundled-dep invariant (exact
  `dependencies` pin == `pi-version.txt`, `bundleDependencies` includes pi,
  no `peerDependencies` entry, `bin.pi` + `bin/pi.mjs` present).

### Consequences (amendment)

- Good: one `npm i -g skill-pi` gives a working `pi` + `skill-pi` with the
  extensions; no separate host pi install; `~/.skill-pi/` stays config-only.
- Good: the exact pin and `pi-version.txt` guard are unchanged, so R12 is
  still enforced at `node --test` time.
- Bad: `npm i -g skill-pi` now downloads the full pi tree (the 1.1.0 bundle),
  so install is heavier than the peer model. Accepted: it is the price of a
  self-contained distribution, and the version is still exact-pinned.
- Bad: `npm pack` from the source dir does **not** ship a
  `bundleDependencies` dependency (verified: 0 pi files in the tarball); only
  a real `npm i -g .` / `npm link` materializes it. The consumption test for
  this module is a global install, not `npm pack` inspection.
- Bad: a dev `npm link` republishes the `pi` bin and collides with any
  pre-existing global `pi` (`EEXIST`). Mitigation: the `skill-pi:build` Task
  runs with `SKILL_PI_TRUST_AUTO=1` and the stale standalone pi global is
  removed once the bundled model is in place.

### Revisit criteria (amendment)

Revisit (supersede again) when any of:

1. pi ships a 2.x major — bump the exact pin and `pi-version.txt`; the
   bundle is unaffected except for the pinned version.
2. pi's published tarball is restructured such that `bundleDependencies`
   no longer yields a resolvable `node_modules/.bin/pi` under a global
   install — `piBin()`'s bundled-first resolution must be re-validated.
3. Skillgrid needs to run under an *externally* managed pi (e.g. a host that
   already ships a different pi) — reintroduce a host-provided override
   path above the bundled one (the `SKILL_PI_PI_BIN` env already allows
   per-invocation override without a code change).
