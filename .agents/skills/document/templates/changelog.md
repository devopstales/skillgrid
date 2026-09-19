# Changelog Entry Template

Append to `CHANGELOG.md` (repo root; created on first use with the `# Changelog`
header below). Keep a Changelog format. Every bullet names what changed —
"misc improvements" is not an entry.

First-use file shape:

```markdown
# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
```

Entry shape (under the version heading; create the heading with today's date
and the version if absent):

```markdown
## [Unreleased]

### Added
- [what was added, one line] — source: [ticket/change topic]

### Changed
- [what changed in behavior, one line] — source: [ticket/change topic]

### Fixed
- [what was broken and is now fixed, one line] — source: [ticket/change topic]

### Deprecated
- [what is going away and when, one line]

### Removed
- [what was removed, one line]

### Security
- [the vulnerability and the fix, one line]
```

Rules:
- Drop any section with no entries — do not leave an empty `### Fixed` heading.
- "Source" is the change topic (the `.skillgrid/specs/YYYY-MM-DD-<topic>` name)
  or ticket ID — the reader's path to the full record. Not a commit hash.
- User-visible changes only. Internal refactors with no behavior change do not
  get a bullet (they belong in the PR body, not the changelog).
- `Breaking` changes get their own line under `Changed` prefixed with
  **Breaking:** and a migration note.
- When a version is tagged, move the `[Unreleased]` block under the version
  heading and add a fresh empty `[Unreleased]`.
