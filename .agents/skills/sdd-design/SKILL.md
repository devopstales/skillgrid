---
name: sdd-design
description: "RETIRED in SDD v3. Use sdd-propose (writes change.md). This stub only redirects."
disable-model-invocation: true
user-invocable: false
license: MIT
metadata:
  author: devopstales
  version: "3.0"
  family: sdd
  part-of: skillgrid
  status: retired
  successor: sdd-propose
  phase-order: "init → explore → propose → spec → apply → verify → archive"
  delegate_only: true
---

# sdd-design — RETIRED

**Do not use this skill.** In SDD v3, `sdd-propose` absorbs design and writes **`change.md`** from [`../_shared/templates/template-change.md`](../_shared/templates/template-change.md).

| Was (v2) | Now (v3) |
|---|---|
| `sdd-propose` → `intent.md` | `sdd-propose` → `change.md` (WHY+HOW) |
| `sdd-design` → `plan.md` | merged into `sdd-propose` |

**Next action:** load [`../sdd-propose/SKILL.md`](../sdd-propose/SKILL.md).

Threat matrix reference moved to [`../sdd-propose/references/threat-matrix.md`](../sdd-propose/references/threat-matrix.md).

Layout: [`../_shared/conventions/sdd-structure.md`](../_shared/conventions/sdd-structure.md).
