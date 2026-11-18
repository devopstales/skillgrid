---
type: Gotcha
title: "projectwiki generate runs in degraded append-only mode"
status: stable
sources:
  - "code:mnemonic/internal/transport/handler/handler_projectwiki.go"
---

# `projectwiki generate` runs in degraded append-only mode

`NewProjectWikiHandler` wires no `Synthesizer`, so `projectwiki generate`
degrades to the append-only floor and writes raw memory facts as pages instead of
merged concepts. Running it on a vault that is already curated will pollute it
with fact-dump pages.

## Guidance

- Do not run `projectwiki generate` as a maintenance step on this repo's vault.
- Prefer hand-authored pages + `projectwiki index` (the same-edit rule).
- If a synthesizer seam is ever wired, revisit this gotcha and flip it to
  `status: deprecated`.

## Related

- [[architecture/01-module-map]] — the handler seam
- [[gotchas/stale-installed-mnemonic-binary]] — another mask that hides drift
