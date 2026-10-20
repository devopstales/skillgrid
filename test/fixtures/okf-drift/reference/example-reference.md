---
type: Reference
title: Example Reference
description: Reference page with a broken in-bundle link and a wikilink
status: stable
generated:
  by: human:paladm
  at: 2026-10-18T00:00:00Z
verification:
  engine: ripgrep
  anchors:
    - path: missing.txt
      all_of:
        - "a pinned term"
      scope_lines: 40
---

# Example Reference

A stable page with anchors, so `sources` is required — and missing here, which
the provenance-required check must report.

It also links to a file that does not exist:
[ghost page](ghost-page.md) — a link-broken finding.

And it uses the banned vault syntax: [ghost page](ghost-page.md) and
[[wikilink]] — a no-wikilink-syntax finding.
