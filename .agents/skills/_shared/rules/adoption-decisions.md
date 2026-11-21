# Adoption Decisions (shared convention)

How the repo adopts an external method or pattern (a published protocol, a
community adaptation, a standard). Cite-don't-restate applies: this rule owns
the decision format; the ADR owns the decision's content.

## When this applies

Adopting the *method* of an external source — not a dependency (that is the
project's dependency policy, which needs its own record), not a one-off
reference (that belongs in the spec's sources list).

## The "method, not plugin" test

Before any adoption, answer: does the repo already orchestrate the part the
external tool owns? If yes, adopt the method and record the split — what was
taken, what was left. Adopting both a method *and* its wrapper is usually
double orchestration.

## The decision record

An adoption is an ADR (per `architectural-decision-records`), with the
Context naming the source(s) — repo, version, date read — and the Decision
containing a **Not adopted** list: what was deliberately *not* taken and why.
The Not adopted list is what keeps a later session from re-deriving the
decision: "why didn't we use their drift reporter?" must have a committed
answer, not a memory.

## The smoke test

The first real run of a newly adopted gate is its own test case. State it in
the change spec: the adoption's hypothesis gets its first data point from
the adoption's own first run, and the run's findings are the evidence.

## The rename sweep

Replacing an existing mechanism renames *every* reference to it in one
change: the grep for the old name returns zero hits outside the new rule file
(a historical "replaces" note is the only permitted survivor). A rename that
leaves stale references in two files is drift, not a rename — the sweep is
one atomic commit, and the verification step is the grep.

## The tooling budget

When the adoption adds a *tool* (a CLI, a transport, a source) rather than a
decision, the tool's allow-list lives in config, not in skill prose: the
`research.sources` list (or the project's equivalent) is the **authoritative
budget** — if the tool is in the list, the skills that call it may; if it is
not, it is not available. Adding a tool is a one-line config edit plus the
pointer updates; editing the skill's sourcing prose to name the tool is the
drift this rule exists to prevent.

Three invariants for budgeted tools:

- **Transport, not workflow.** A capability that fetches the same kind of data
  through a different surface (a CLI that reads login-walled platforms, a
  transcript extractor, a structured-data fetcher) is a *tool the existing
  skills call when the surface demands it* — not a new skill. A new skill is
  justified only by a new *workflow* (new steps, new gates, new artifacts).
- **Non-fatal install.** An optional tool degrades its stack, it does not
  break the install: skip if the binary is already on PATH, warn if its
  package manager is missing, respect `--dry-run`, capture output. A hard
  prerequisite that the tool's manager already is (e.g. `uv`) stays hard; the
  tool itself does not become one.
- **The gotcha lives at the point of use.** If the correct invocation is
  non-obvious (a name collision with a different package on the same
  registry, a required flag, a version pin), the install command and its
  manual-install hint carry the warning — not a separate document the
  installer never reads.

## Park, don't defer

Follow-ups that are *tools* (new binaries, new scanners) or *reworks* (a
larger flow change) are not adopted with the method. They are recorded in the
ADR's Not adopted list with their revisit condition — a dated, checkable
trigger ("revisit after one full pipeline run on the new protocol"), not a
vague "later."
