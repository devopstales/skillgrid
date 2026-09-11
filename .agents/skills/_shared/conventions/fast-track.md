# Fast-Track Convention (trivial / small changes)

Full pipeline (`propose → design → spec → tasks → apply ⇄ verify → archive`) is the default.
Fast-track is an explicit, recorded waiver — not a silent skip. Policy source: `docs/skillgrid/config.yaml`
`rules.fast_track` (this file defines behavior; the config defines the numeric bounds — on conflict, the config wins).

## Eligibility

| Class | Shape (bounds from `rules.fast_track.classes`) | Skips |
|---|---|---|
| `trivial` | One-line intent (typo, copy, single-line config, pure rename); ≤3 files, no behavior change, no migration/dependency/trust-boundary change | design, spec; `sdd-tasks` writes a light `tasks.md` (1–3 tasks, forecast still required) |
| `small` | Single file or single concern, no new capability, no migration; ≤10 files, `400-line budget risk: Low` | design, spec; `sdd-tasks` writes a light `tasks.md` |

**Blocked from fast-track** (full pipeline mandatory — `rules.fast_track.blocked_when`): any design
threat-matrix row Applicable, `400-line budget risk: High`, new trust boundary or migration, or
multi-domain spec impact. When in doubt, run the full pipeline.

## Waiver record (load-bearing)

`sdd-propose` records, downstream phases honor — never re-derive. Fields per `rules.fast_track.waiver`
(`recorded_in: proposal.md`, `carried_to: tasks.md, verify-report, archive-report`):

```markdown
## Fast-Track Waiver
- Class: trivial | small
- Skips: design, spec (trivial) | design, delta specs (small)
- Reason: {why the full planning chain adds no signal here}
- Approved by: {user name/handle or "user gate"}
```

Carry the same block (or `Waiver: trivial/small — see proposal.md`) into `tasks.md` top and the
verify/archive envelopes as `Fast-track: {trivial|small} (waiver in proposal.md)`.

## Phase behavior under waiver

- `sdd-tasks`: light `tasks.md` allowed (fewer phases, 1–2 lines per task); the four plain-text
  guard lines (`Decision needed before apply:`, `Chained PRs recommended:`, `Chain strategy:`,
  `400-line budget risk:`) stay byte-identical — reviewers match them literally.
- `sdd-verify`: missing design/spec artifacts are WARNING (`waived — see proposal.md`), not CRITICAL,
  **iff** the waiver exists and no blocked-from-fast-track condition is present. Otherwise missing
  artifacts are handled per the normal Graceful Artifact Handling rules.
- `sdd-archive`: a waived change archives as intentional-partial — record the waiver in
  Overrides / Final-State Notes. Verification and Task Completion gates still apply.
