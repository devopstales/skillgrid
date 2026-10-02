# Briefing: Mnemonic Web UI rewrite from prototype 001

> **STATUS:** `approved` (2026-10-02) — user said "execute"; keep the mockup D3 graph.

**Date:** 2026-10-02
**Tier:** T2
**Status:** approved (user: execute; keep mockup D3 graph)
**Classification:** standard (verification floor L2 — full UI suite + build + Go embed build)

## As-built note

The code landed as commit `aa0aa791` ("feat(ui): rewrite mnemonic web admin around mockup IA and D3 graph") on `release/2` before this change folder had a blueprint or tickets. The pipeline artifacts in this folder (`acceptance.feature`, `blueprint.md`, `tasks.md`) were written afterwards against that commit; execution verifies each ticket against the committed code and closes the gaps the verification finds (a briefing task-ref regression; the Prototypes panel's endpoint). Evidence base: `.skillgrid/prototypes/001-mnemonic-webui-mockup/` (prototype.md + findings.md).

## Intent

Rewrite the embedded `skillgrid-ui` SPA to match the information architecture, visual language, and panel set validated in `.skillgrid/prototypes/001-mnemonic-webui-mockup/`, keeping production scaffolding (Vite, React 19, TypeScript, TanStack Router, Vitest, Go embed).

## Locked decisions

- **Code graph:** D3 force layout from the mockup (ADR-0017). Sigma/graphology removed.
- **Theme:** mockup surface/indigo tokens (dark console), not Terminal Ops green.
- **Nav IA:** Overview · Project (Kanban, Changes, Decisions/ADR, Prototypes) · Memory (Sessions, Code Graph, MemFS) · Observe (Telemetry, Compaction, Web Cache) · Docs · System (Security, Settings, Swagger).
- **Demoted from primary nav (routes kept):** Git, Prototypes, Memories list, Search, mnemonic Decisions bridge.
- **Route collisions:** UI uses `/project/prototypes` and `/system/security` so JSON `GET /prototypes` and `GET /security/trivy` stay unambiguous.
- **API:** relative URLs + `project` query param (not hardcoded `127.0.0.1`).

## Out of scope

- Auth / pagination / mobile polish beyond existing slide-over
- Backend FTS5 search gap
- Doc-root configuration changes (empty states remain until roots exist)

## Success

- Nav matches mockup groups; D3 graph loads ≤500 nodes with inspector
- New panels (Prototypes, Compaction, Web Cache, Security) fetch live endpoints
- `npm test` + `npm run build:check` pass; Go embed still builds
