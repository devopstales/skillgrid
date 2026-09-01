# Delta + Full Spec Format (skillgrid)

Two document shapes, chosen by the proposal's Capabilities entry:

- **Full spec** — for a **New** capability (no existing main spec). Sections `## Purpose` + `## Requirements`.
- **Delta spec** — for a **Modified** capability (a main spec already exists). Sections `## ADDED` / `## MODIFIED` / `## REMOVED` / `## RENAMED` Requirements. Omit a section if it has no content.

Both use **RFC 2119** strength keywords in requirement text and **Given/When/Then** in scenarios. The delta shape is the source of truth for `sdd-design` and `conventions/openspec.md`.

## The one rule that matters: MODIFIED is REPLACE semantics

`sdd-archive` replaces the main-spec requirement **block-for-block** with your `## MODIFIED` block. It does not merge.

```
1. Locate the requirement in openspec/specs/{domain}/spec.md
2. COPY the ENTIRE `### Requirement:` block — name, body, and ALL its scenarios
3. PASTE it under `## MODIFIED Requirements` in the delta spec
4. EDIT the copy to reflect the new behavior
5. Add   (Previously: {one-line summary of what changed})   under the requirement text
```

If your MODIFIED block is partial, every scenario you did not copy is **gone** the moment archive runs. Common pitfall: writing only the changed scenario and silently dropping the rest. If you are adding new behavior without changing existing behavior, use `## ADDED` instead.

## Full spec (New capability)

```markdown
# {Domain} Specification

## Purpose
{High-level description of this domain and why it exists.}

## Requirements

### Requirement: {Name}
The system {MUST | SHALL | SHOULD | MAY} {behavior}.

#### Scenario: {Happy path}
- GIVEN {precondition}
- WHEN {action}
- THEN {expected outcome}
- AND {additional outcome, if any}

#### Scenario: {Edge case / failure}
- GIVEN {precondition}
- WHEN {action}
- THEN {expected outcome}
```

## Delta spec (Modified capability)

```markdown
# Delta for {Domain}

## ADDED Requirements

### Requirement: {New Requirement Name}
The system {MUST|SHALL|SHOULD|MAY} {new behavior}.

#### Scenario: {Name}
- GIVEN {precondition}
- WHEN {action}
- THEN {expected outcome}

## MODIFIED Requirements

### Requirement: {Existing Requirement Name}
{FULL updated requirement text — replaces the existing one entirely.}
(Previously: {what it was before, in one line})

#### Scenario: {Unchanged scenario — keep if still valid}
- GIVEN {precondition}
- WHEN {action}
- THEN {outcome}

#### Scenario: {Updated or new scenario}
- GIVEN {updated precondition}
- WHEN {updated action}
- THEN {updated outcome}

## REMOVED Requirements

### Requirement: {Requirement Being Removed}
(Reason: {why this requirement is being deprecated/removed})
(Migration: {what replaces it, or "None"})

## RENAMED Requirements

### Requirement: {Old Name} → {New Name}
(Reason: {why it is being renamed})
(Migration: {how references/tests/docs update, or "None"})
```

## Carrying the design's threat rows

The `sdd-design` phase marks an applicability-driven threat matrix, and **applicable rows are spec inputs.** For each row marked `Applicable`, ensure at least one scenario in the spec set covers its planned RED test. A design-applicable row with no covering scenario is a handoff gap — add the scenario or flag it in the envelope `risks`. `N/A` rows need no scenario.

## RFC 2119 Quick Reference

| Keyword | Meaning |
|---|---|
| **MUST / SHALL** | Absolute requirement |
| **MUST NOT / SHALL NOT** | Absolute prohibition |
| **SHOULD** | Recommended; exceptions may exist with justification |
| **SHOULD NOT** | Not recommended; may be acceptable with justification |
| **MAY** | Optional |

Use the **uppercase** forms in requirement text. Lowercase `must`/`should` are ordinary words, not strength markers.
