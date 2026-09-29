# <capability — one H1, becomes Feature:>
#
# Source: .skillgrid/specs/<YYYY-MM-DD-topic>/
# Trace: change briefing ## Definition of Done; blueprint tasks SATISFIES lines.
#
# Format rules:
# - Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor does NOT re-indent)
# - No Gherkin tags in the spec — selection/trace is via the SATISFIES field in blueprint/tasks
# - One blank line between the heading and the fence is fine
# - Gates: each requirement carries a `#### Gates` block — the runnable shadow of
#   its happy-path and failure scenarios. Author it BEFORE implementing.

## Requirements

### Requirement: <requirement-name>

The system SHALL <falsifiable SHALL/MUST sentence in domain language>.

#### Scenario: <happy path — observable outcome>

```gherkin
      Given <precondition in domain language>
      When <action>
      Then <observable outcome>
```

#### Scenario: <edge case / boundary>

```gherkin
      Given <precondition>
      When <action>
      Then <expected fallback or boundary behavior>
```

#### Scenario: <failure state>

```gherkin
      Given <precondition>
      When <action>
      Then <expected error or recovery>
```

#### Gates
# Runnable shadow of this requirement's happy-path and failure scenarios.
# Author BEFORE implementing. One entry per happy-path and failure scenario
# (name the scenario it mirrors); edge scenarios may be left to the tests.
# G<n>: <one-line observable outcome, mirrors the scenario name>
#   CHECK: <repo-owned command, exit 0 on success>
#   EXPECT: <success-only marker emitted after every assertion passes>
#   EVIDENCE: pending
# Manual (no command can decide the outcome):
#   G<n>: <observable outcome>
#   EVIDENCE: pending
# Impossible (keep the gate, do not delete it):
#   G<n>: ABANDON <non-empty reason and the handoff>

### Requirement: <second-requirement-name>

The system SHALL <falsifiable SHALL/MUST sentence in domain language>.

#### Scenario: <happy path>

```gherkin
      Given <precondition>
      When <action>
      Then <observable outcome>
```

#### Scenario: <edge case>

```gherkin
      Given <precondition>
      When <action>
      Then <expected fallback or boundary behavior>
```

#### Scenario: <failure state>

```gherkin
      Given <precondition>
      When <action>
      Then <expected error or recovery>
```

#### Gates
# Runnable shadow of this requirement's happy-path and failure scenarios.
# Author BEFORE implementing. One entry per happy-path and failure scenario
# (name the scenario it mirrors); edge scenarios may be left to the tests.
# G<n>: <one-line observable outcome, mirrors the scenario name>
#   CHECK: <repo-owned command, exit 0 on success>
#   EXPECT: <success-only marker emitted after every assertion passes>
#   EVIDENCE: pending
# Manual (no command can decide the outcome):
#   G<n>: <observable outcome>
#   EVIDENCE: pending
# Impossible (keep the gate, do not delete it):
#   G<n>: ABANDON <non-empty reason and the handoff>

# Rules:
# - ≥1 happy + edge + failure scenario per requirement
# - Every briefing requirement → a Rule: (### Requirement:)
# - Scenario names unique and referenceable from blueprint tasks SATISFIES lines
# - Domain language only: use .skillgrid/glossary terms, never implementation jargon
# - Then steps state observable outcomes, never internal state
# - Steps: 6-space indent in the fence (verbatim — the extractor does NOT re-indent)
# - Gates: every requirement's happy-path scenario MUST have a G<n> entry (runnable
#   CHECK+EXPECT, or a manual/ABANDON entry with a reason); a happy path with none
#   is a traceability gap
# - Gates: CHECK must be a repo-owned command (Node script preferred); EXPECT is a
#   success-only marker, never a copied number; a negative/absence oracle names a
#   positive control fixture
# - Gates: ABANDON is terminal and non-successful — it surfaces as a handoff, never
#   a pass; EVIDENCE is bound to the gate's current CHECK+EXPECT
# - Gates: a design task may carry a STRUCTURAL gate — an observable invariant about
#   the module's shape, in the same CHECK+EXPECT format (e.g. "the public interface
#   exposes ≤3 entry points", "tests cross the seam and assert on outcomes, not
#   internal state"). These are the deep-module invariants from
#   skillgrid:writing-blueprints → references/codebase-design.md (deletion test,
#   one-adapter rule). Assert the shape, not the internals.
