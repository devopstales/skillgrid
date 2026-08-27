## Context

Skillgrid workflows (SDD, IDD, BDD, strict TDD) need a shared vocabulary for testing. Based on [How TDD and BDD Fit Into SDD](https://intent-driven.dev/blog/2026/08/23/tdd-bdd-spec-driven-development/): BDD = macro, Mockist TDD = micro, SDD/IDD = spec anchor. Testing enforcement is layered: agent skills during apply, project manifest for commands/thresholds, hooks for commit boundaries, CI for gates before merge.

## Goals / Non-Goals

**Goals:**
- Every workflow maps to concrete test layers and tools
- Projects declare capabilities in one manifest; agents and CI use the same commands
- BDD promote blocked unless L3 GREEN; code promote blocked unless L1 GREEN
- Strict TDD anti-patterns documented and checkable

**Non-Goals:**
- Mandating Playwright for every project
- Mandating mutation testing on every PR (opt-in per module)
- Replacing superpowers TDD skill with a skillgrid copy
- Python/behave as default (Node cucumber-js pack is reference)

## Decisions

### 1. Test layer model

**Decision:** Six layers (L0 static, L1 unit, L2 integration, L3 acceptance, L4 E2E, L5 gates) with a matrix mapping change type to required layers.

**Alternatives considered:**
- Three layers (static, unit, acceptance) — rejected: too coarse; integration and E2E have distinct roles
- Per-stack layer definitions — rejected: layers are stack-agnostic; tools are stack-specific

**Rationale:** Six layers cover the full spectrum from commit-time checks to merge-time gates without overlap.

### 2. Project manifest (`testing-capabilities.yaml`)

**Decision:** Each project declares its test capabilities in `docs/testing-capabilities.yaml`.

**Alternatives considered:**
- Hardcoded in AGENTS.md — rejected: not machine-parseable for CI
- Separate config per workflow — rejected: too fragmented

**Rationale:** Single source of truth for both agents and CI. Detected at init or hand-authored.

### 3. Strict TDD checklist

**Decision:** Agent Definition of Done per behavior follows a 10-item checklist (red before green, minimal code, refactor, coverage, etc.).

**Alternatives considered:**
- RED-GREEN-REFACTOR only — rejected: insufficient for strict TDD (missing coverage, boundary cases)
- Full TDD methodology doc — rejected: too long; checklist is actionable

**Rationale:** Checklist is verifiable via commit history + coverage/mutation tools.

## Risks / Trade-offs

- **Manifest drift** -> CI validates manifest against actual project structure
- **Agent ignores manifest** -> Skills reference the manifest; hooks enforce at commit boundaries
- **Tool version mismatch** -> Pin versions in `config.d/tools.yaml`
- **CI pipeline complexity** -> Document order; projects adopt incrementally

## Migration Plan

1. Define the test layer matrix and tool reference matrix
2. Create `docs/testing-capabilities.yaml` template
3. Update `02-usage.md` with testing enforcement section
4. Update skills (`test-driven-development`, `bdd-workflow`) to reference the manifest
5. Document CI pipeline order for BDD-enabled projects

## Open Questions

None — this is a definition/specification change, not an implementation change.
