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
