# Norse persona delegation (shared protocol)

Personas are **stateless capability packs**. The **session coordinator** (parent running `/sdd-*` or brainstorm) dispatches them; phase executor skills do the phase work unless explicitly told to delegate.

## Rules

- One subagent run = one **`persona`** + one **`capability`** (see the active phase skill or workflow).
- Fresh context every time; pass handoff, PRD/OpenSpec paths, and output path in the prompt.
- Personas do not coordinate with each other; the coordinator merges reports.
- Reports: `.skillgrid/tasks/research/<change-id>/<persona>-<capability>.md`
- Events: `.agents/skills/_shared/skillgrid-handoff.md`

## Hard gates

- **`tyr`** and **`heimdall`** (`hard_gate`): **critical** findings block phase progression until fixed, risk-accepted, or HITL-resolved.
- Unresolved **critical conflict** between persona reports → `status: blocked`.
- **User** is final authority on **release** and **destructive** actions.

## Return extension (persona subagents)

Add to the standard SDD envelope (`.agents/skills/_shared/sdd-return-envelope.md`):

- `persona`, `capability`, `findings_severity` (`none|info|warn|critical`), `hitl_required`

## Persona capabilities (reference)

| Persona | Capabilities |
| --- | --- |
| `kvasir` | `codebase-recon`, `external-evidence-gap` |
| `mimir` | `bootstrap-readiness`, `architecture-coherence` |
| `thor` | `execution-feasibility`, `implementation-enforcement` |
| `tyr` | `spec-compliance`, `tasks-readiness` |
| `heimdall` | `security-review`, `release-gate` |
| `frigg` | `ux-clarity`, `content-quality` |
| `loki` | `assumption-stress-test`, `risk-acceptance` |
| `bragi` | `structured-artifacts`, `spec-quality` |
| `vidar` | `root-cause-analysis`, `regression-prevention` |

**Phase-specific invocations** live in each `sdd-<phase>/SKILL.md` and `.agents/workflows/sdd-<phase>.md` — not in a central JSON registry.
