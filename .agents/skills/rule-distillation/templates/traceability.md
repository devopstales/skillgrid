# Traceability — {source name}

Canonical full source: [full](<path-to-full.md>) — **untouched** by this distillation.
Re-run date: {YYYY-MM-DD}. Process version: compression-rules.md @ {commit or date}.

## Compression decisions

- {One line per notable call: what was merged, what was intentionally lost, whether a
  process vs book diagnosis was triggered, and whether compression-rules.md was updated.}
- {No retained rule is treated as `default` unless evidence is recorded below.}

## Mini mapping

Decision rules:

- `M1` {one-line rule summary}. Source: `{section}` ({start}-{end}), `{section}` ({start}-{end}).
- `M2` {summary}. Source: `{section}` ({start}-{end}).

Trigger rules:

- `M{15}` {summary}. Source: `{section}` ({start}-{end}).

Final checklist items restate: M1, M2, M5, … (no new rules introduced).

## Nano mapping

- `N1` {summary}. Source: `{section}` ({start}-{end}).
- `N2` {summary}. Source: `{section}` ({start}-{end}).

## Omission dispositions

Walk every `full` section. Each gets exactly one outcome:

| Source section (full lines) | Outcome | Disposition |
|---|---|---|
| `{section}` ({start}-{end}) | kept in mini | M1 |
| `{section}` ({start}-{end}) | merged | M2 (same operational consequence as …) |
| `{section}` ({start}-{end}) | nano-only | N1 |
| `{section}` ({start}-{end}) | intentionally lost | {reason: too situational / framing / preserved only in full} |
| `{section}` ({start}-{end}) | dropped as `default` | evidence: {eval / review finding / known model mistake} |

## Default-label evidence

- `M?`-related rule dropped as `default` → {cited evidence}. (List every one; an
  empty list with "no rule treated as default" is a valid state.)

## Section-coverage review

Mandatory. Confirm every `full` section appears in the Omission dispositions table
above (kept / merged / nano-only / intentionally lost / default-dropped). If a section
is missing from the table, the review is incomplete.

- [ ] Every `full` section is dispositioned.
- [ ] No `M*`/`N*` id lacks a source section + line range.
- [ ] Every `default` drop cites evidence.
- [ ] The canonical `full` file was not edited.
