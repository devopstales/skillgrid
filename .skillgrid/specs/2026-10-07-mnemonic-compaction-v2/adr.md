# ADR Review Manifest

- **Change:** `2026-10-07-mnemonic-compaction-v2`
- **Status:** in review
- **Review date:** 2026-10-07

## In-Force ADRs Reviewed

- `.skillgrid/artifacts/04-adr-0027-clm-context-language-model.md` — Go owns state/decisions, Node owns the request path; mirror in temp dir; overflow guard + calibration; opt-in via `clm:` config. This change **extends** it: the structured six-section prompt feeds the existing CLM mirror path; it does not duplicate the mirror.
- `.skillgrid/artifacts/04-adr-0028-context-revisions-session-scoped.md` — `context_revisions` (migration 052) is session-scoped audit, purged at session end. This change **extends** it: migration 053 adds one additive `steering` column; the table's lifecycle and source-of-truth framing are unchanged.
- `.skillgrid/artifacts/04-adr-0023-openai-compatible-llm-provider.md` — one OpenAI-compatible client attaches all LLM seams. The advisory gate reuses this client (no new provider, no new dependency).
- `.skillgrid/artifacts/04-adr-0025-context-harness-owner.md` — the Context Harness owns the session context lifecycle. The proactive build rides the existing `/sessions` end HTTP path the harness exposes.
- `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` — fail-open floors hold: the advisory gate returns `hint:false` on LLM error and never blocks a compaction; the proactive build is async and warns, never fails, the request.
- Locked constraint "No new dependencies without an ADR" — satisfied: no new library (reuses `internal/llm`; Bubbletea/Lipgloss/Bubbles already present in `skillgrid-cli/go.mod` as indirect dependencies, promoted to direct for the context TUI — not a new dependency); the only schema change is one additive column (migration 053), named in ADR-0029.

## New Durable ADRs Created

- `.skillgrid/artifacts/04-adr-0029-compaction-advisory-steering.md` — compaction advisory gate (one combined LLM call, fail-open), `steering` column on `context_revisions`, and proactive build; extends ADR-0027/0028; opt-in via `mnemonic.compaction` config.

## Supersessions

- None. ADR-0027 (CLM) and ADR-0028 (`context_revisions`) remain in force; ADR-0029 adds the advisory/steering/proactive layer above them.
