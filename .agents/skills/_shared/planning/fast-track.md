# Fast-Track Convention (trivial / small changes)

The full pipeline is the phase order in [sdd-structure.md](../rules/sdd-structure.md). It is the default.
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

`blueprint` (or the user gate) records, downstream phases honor — never re-derive. Fields:

```markdown
## Fast-Track Waiver
- Class: trivial | small
- Skips: brainstorming, blueprint (trivial) | brainstorming, blueprint (small)
- Reason: {why the full planning chain adds no signal here}
- Approved by: {user name/handle or "user gate"}
```

Carry the same block (or `Waiver: trivial/small — see briefing.md`) into `tasks.md` top and
`report.md`'s QA half as `Fast-track: {trivial|small} (waiver in briefing.md)`.

## The briefing-only shape (worked example)

A `trivial` change carries no blueprint — the "how" lives in the briefing.
The briefing then must stand on its own: a worked example of this shape is the
`2026-10-24-agent-reach-integration` change (skillgrid repo,
`.skillgrid/change-specs/`), which is a briefing with no blueprint and shows
the full fast-track shape:

- **Goal / Problem** — what and why, in two short sections.
- **Research Sources** — a cited table (source, URL, key takeaway) for every
  external fact the design rests on.
- **Design Decisions** — the *how*, numbered, each decision stated as a
  rule the executor follows (e.g. "non-fatal install", "no new skill").
- **Steps** — a table: `# | Step | Deliverable | Key files`, with the **zone
  order** stated (spec-zone steps commit before code-zone steps).
- **Trust Boundaries** — "None" when genuinely none, otherwise the new
  surface named.
- **Verification Floor** — the ladder level and why it is sufficient.
- **Rollback** — Code / Config / Machine, each with the exact revert.
- **Open Questions** — "None" when resolved, otherwise the list.

A briefing that carries the how must carry **all of the above** — a missing
Rollback or a Steps table without zone order is a WARNING at `qa`, not a
silent gap. The worked example's design decisions are also the model for the
two invariants in `_shared/rules/adoption-decisions.md` § Tooling budget
(transport-not-workflow, non-fatal install, gotcha-at-point-of-use).

**Compact steps form (`small` class).** When a `small` briefing's steps are
self-evidently small (no zone order needed, each step verifiable in one
command), the Steps table MAY collapse to the step-with-verify form — each
step names its own check, and the check is what the executor runs:

```
1. [Step] → verify: [command or check that proves it]
2. [Step] → verify: [command or check]
```

The form earns its place because a strong per-step check lets the executor
loop independently (run the step, run the check, move on) — a step with a
weak check ("make it work") is a stall in disguise. The full column table
(`# | Step | Deliverable | Key files`) remains the default whenever zone
order, multiple deliverables, or non-trivial verification apply. `trivial`
changes keep the full table — the briefing is the only artifact, and the
columns are what keep it honest.

## Phase behavior under waiver

- `slicing`: light `tasks.md` allowed (fewer phases, 1–2 lines per ticket).
- `qa`: missing briefing/blueprint artifacts are WARNING (`waived — see briefing.md`), not CRITICAL,
  **iff** the waiver exists and no blocked-from-fast-track condition is present. A missing
  `report.md` QA half (verdict empty / `## Gate Decision` absent) is the same WARNING — not CRITICAL —
  under the same conditions.
- `execution`: proceeds normally; TDD is still mandatory.
- `review`: proceeds normally.
- `ship`: light variant — the merge/PR menu still appears, but no PR-body generation. The
  mechanical move to `change-archive/` is **not** waived. No release mechanics, no docs check.
- `reflect`: light variant — `mem_save` a single compact observation + a one-line acceptance verdict in
  the return text; **no full `report.md` document**. Session close
  (`mem_session_summary` + `mem_session_end`) is **never** waived.
