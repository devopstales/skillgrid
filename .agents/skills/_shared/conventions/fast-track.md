# Fast-Track Convention (trivial / small changes)

Full pipeline (`brainstorming → blueprints → slicing → execution → qa → review`) is the default.
Fast-track is an explicit, recorded waiver — not a silent skip. Policy source: `.skillgrid/config.yaml`
`rules.fast_track` (this file defines behavior; the config defines the numeric bounds — on conflict, the config wins).

## Eligibility

| Class | Shape | Skips |
|---|---|---|
| `trivial` | One-line intent (typo, copy, single-line config, pure rename); ≤3 files, no behavior change, no migration/dependency/trust-boundary change | brainstorming, blueprint, acceptance.feature; `slicing` writes a light `tasks.md` (1–3 tasks) |
| `small` | Single file or single concern, no new capability, no migration; ≤10 files | brainstorming, blueprint; acceptance.feature still required; `slicing` writes a light `tasks.md` |

**Blocked from fast-track** (full pipeline mandatory):
- Any threat-matrix row Applicable
- New trust boundary or migration
- Multi-domain impact
- 400+ line change

When in doubt, run the full pipeline.

## Waiver record (load-bearing)

`writing-blueprints` (or the user gate) records, downstream phases honor — never re-derive. Fields:

```markdown
## Fast-Track Waiver
- Class: trivial | small
- Skips: brainstorming, blueprint (trivial) | brainstorming, blueprint (small)
- Reason: {why the full planning chain adds no signal here}
- Approved by: {user name/handle or "user gate"}
```

Carry the same block (or `Waiver: trivial/small — see briefing.md`) into `tasks.md` top and the
qa-report as `Fast-track: {trivial|small} (waiver in briefing.md)`.

## Phase behavior under waiver

- `slicing`: light `tasks.md` allowed (fewer phases, 1–2 lines per ticket).
- `qa`: missing briefing/blueprint artifacts are WARNING (`waived — see briefing.md`), not CRITICAL,
  **iff** the waiver exists and no blocked-from-fast-track condition is present.
- `execution`: proceeds normally; TDD is still mandatory.
- `review`: proceeds normally.
- `ship`: light variant — the merge/PR menu still appears, but no PR-body generation. The
  mechanical move to `archive/` is **not** waived. No release mechanics, no docs check.
- `reflect`: light variant — `mem_save` a single compact observation + a one-line acceptance verdict in
  the return text; **no full `report.md` document**. Session close
  (`mem_session_summary` + `mem_session_end`) is **never** waived.
