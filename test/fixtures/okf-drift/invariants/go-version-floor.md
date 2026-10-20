---
type: InvariantContract
title: Go 1.23+ build floor
description: Invariant with a claim literal that no longer matches the pinned text
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
    literal: go 1.23
verification:
  engine: ripgrep
  anchors:
    - path: go.mod
      all_of:
        - go 1.22
      near: module example.com
      scope_lines: 40
---

# Go 1.23+ Build Floor

The pinned `go.mod` still reads `go 1.22` (the `all_of` term matches, so the
region is found) but the claim literal says `go 1.23`, which is absent from the
matched region — the anchors-match liveness check must report DRIFT (a
literal-only drift, exercising the union-of-regions claim check).
