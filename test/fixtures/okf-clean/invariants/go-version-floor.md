---
type: InvariantContract
title: Go 1.22+ build floor
description: The repo must keep a Go language floor at 1.22 or above
status: stable
generated:
  by: human:paladm
  at: 2026-10-18T00:00:00Z
sources:
  - id: fixture-gomod
    resource: go.mod
    title: go.mod go directive
claims:
  - subject: go.mod go directive
    literal: go 1.22
verification:
  engine: ripgrep
  anchors:
    - path: go.mod
      all_of:
        - go 1.22
      near: module example.com
      scope_lines: 40
    - path: package.json
      all_of:
        - "clean-fixture"
---

# Go 1.22+ Build Floor

The minimum Go language version stays at 1.22. A `go.mod` below the floor is
a drift signal; raising it is a new superseding decision. Related:
[example decision](../decisions/04-adr-0001-example-decision.md).

The pin above verifies the literal stays in the pinned paths.
