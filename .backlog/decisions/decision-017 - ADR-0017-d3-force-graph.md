---
id: decision-017
title: "ADR-0017 Embedded SPA code graph uses D3 force layout (replaces Sigma)"
date: "2026-10-02"
status: "accepted"
---

Source: `.skillgrid/artifacts/04-adr-0017-d3-force-graph.md`

# Embedded SPA code graph uses D3 force layout (replaces Sigma)

---
status: "accepted"
supersedes: none
date: 2026-10-02
---

## Context and Problem Statement

Prototype 001 (`.skillgrid/prototypes/001-mnemonic-webui-mockup`) validated a D3 v7 force-directed code graph with node inspector (callers/callees highlight) against live `/mnemonic/graph/data`. The production SPA used Sigma.js + graphology instead. The operator preference is the mockup's D3 graph. Adding `d3` is a new npm dependency and requires an ADR under the locked "no new dependencies without an ADR" constraint; Sigma/graphology/`@react-sigma` can then be removed.

## Considered Options

- Keep Sigma + graphology (status quo)
- Port mockup D3 force layout into the Vite SPA; add `d3` (+ `@types/d3`); drop Sigma stack
- Build a custom canvas force layout with no new dependency

## Decision Outcome

Chosen option: "Port mockup D3 force layout into the Vite SPA; add `d3` (+ `@types/d3`); drop Sigma stack", because the prototype already proved D3 at the 500-node cap, the inspector UX is preferred, and removing Sigma/graphology/react-sigma is a net dependency simplification after the swap.

### Consequences

- Good, because one viz stack matches the approved mockup; force simulation + selection highlight ship without Sigma-specific layout plumbing
- Bad, because `d3` is a new npm dependency (mitigated by removing the larger Sigma/graphology set); large graphs still require the server `limit` param
