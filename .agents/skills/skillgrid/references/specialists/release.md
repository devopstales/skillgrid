# Release specialist

Turn a finished change into something another person can install, run and upgrade: the build, the package, the notes, and the rollback.

When to load: Phase 7, and Phase 2 when the distribution shape is a one-way door. Role: orchestrator with the release checklist.

## The release checklist

- [ ] Clean checkout builds with the documented command, and the output is recorded.
- [ ] The test suite runs from a clean checkout, not just the working tree.
- [ ] The version is set once and matches the manifest, the changelog and the tag.
- [ ] The package contains what it claims: `--help`, a smoke test, the licence, the README.
- [ ] Install, upgrade and uninstall each have a command that was run.
- [ ] The rollback is documented and, where the change migrates data, tested on a copy.
- [ ] The release note leads with breaking changes and links the test that proves each fix.
- [ ] Secrets, local paths and build artifacts are not in the package.

## Distribution shape

Choose it in Phase 1 or 2 and record the ADR, because it is a one-way door for users:

- single binary, container image, library package, script, or service deployment;
- the supported platforms and versions, named;
- the upgrade path from the previous release, or an explicit "no upgrade path".

A program with no named distribution shape is not ready to ship, however good the code is.

## Release note

```text
## <version> — <date>
### Breaking
- <what changes for the user> — migration: <step>
### Added / Changed / Fixed
- <one line per user-visible change> — proof: <test or command>
### Known limits
- <what this release does not do>
```

## Rules

- Never publish from a dirty tree or an unrecorded build.
- The programmer makes the ship decision; the specialist makes it safe to say yes.
- A release note that claims a fix without a test is marketing, not a record.
- If the checklist has an unchecked box, say which one and what it would take. Do not round up.
