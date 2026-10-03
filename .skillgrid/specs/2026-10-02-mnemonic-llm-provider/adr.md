# ADR Review Manifest

- **Change:** `2026-10-02-mnemonic-llm-provider`
- **Status:** completed (review); ADR-0023 accepted with briefing
- **Review date:** 2026-10-02

## In-Force ADRs Reviewed

- `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md` — store unchanged; this is a client attach only
- `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` — fail-open floors and existing seams must stay
- `.skillgrid/artifacts/04-adr-0019-decisions-are-files.md` — provider-shape lock is a new ADR file
- Installer patterns in `skillgrid-cli/internal/install` — provider setup is a natural install step, fail-soft; `--skip-provider` escape

## New Durable ADRs Created

- `.skillgrid/artifacts/04-adr-0023-openai-compatible-llm-provider.md` — OpenAI-compatible HTTP Completer only; one attach; natural Local/External install provider setup

## Supersessions

- None. FOLLOWUP task-029 is owned/closed by this change at ship, not superseded as an ADR.
