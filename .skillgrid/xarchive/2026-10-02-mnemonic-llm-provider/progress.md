# SDD ledger — plan: .skillgrid/specs/2026-10-02-mnemonic-llm-provider/tasks.md

## Pre-flight conflict scan

| Pair / Self | Produces vs Consumes | Found | Ruling |
|---|---|---|---|
| Task1 llm.Client vs Task2 attach adapters | Client (Completer) → attach wraps it into 4 seams | Clean. One Completer, ADR-0023 honored. | — |
| Task1 llm.Config vs Task2 config.LLM | Blueprint names `llm.Config` (Client cfg) AND `config.LLM` (config section). Task2 attach signature uses `config.LLM`. | Two types: `config.LLM` (YAML section) and `llm.Config` (client params). Attach converts config.LLM → llm.Config. Not a conflict — two distinct layers. | Ruling: implementer must keep them distinct; AttachSharedLLM takes `config.LLM` + `llm.Completer`, boot path converts. |
| Task2 config.LLM vs Task5 mergeHomeProviderYAML | Task2 defines `mnemonic.llm` yaml section + `LLM` Go struct; Task5 merges `llm:` + `embedder:` into `~/.skillgrid/config.d/indexing.yaml`. | Shared yaml keys. Task5 must write keys that Task2's loader reads (`mnemonic.llm.enabled/base_url/model/api_key`, `mnemonic.embedder.provider/base_url/model`). | Ruling: Task5 depends on Task1's yaml key names landing first (it does — Task1 in same ticket group, sequential). Task5 merges only these keys, never wipes unrelated YAML. |
| Task2 attach in openProject vs Task3 dream | openProject has no `dream` executor today. Task2 attach_llm.go signature names `dream *memory.DreamExecutor`. | Gap: no DreamExecutor in openProject boot path. | Ruling: attach wires the three process-level seams that exist at boot (AskLLM via SetAskLLM, DedupLLM via SetDedupLLMFunc + mem.SetDedupLLM+EnableDedupLLM, Extraction via mem.SetExtractionLLM). Dream wiring is attached via a separate exported function or wired lazily in the dream close path; do NOT block Task2 on inventing a boot-time DreamExecutor. Document which seams are attached at boot vs lazily. |
| Task3 install provider vs security tools | install.Run has no provider step yet; Task5 adds `setupProvider(c)` after installSecurityTools. | Clean insertion point (install.go:124, after security tools block). | — |
| main.go flags (--provider/--skip-provider) vs Config struct | Task5 adds Provider/SkipProvider fields + flags; main.go assembles install.Config at line 325. | Clean. Add flags near vSkipTools. | — |

## Delivery Strategy guard

- 400-line budget risk: **High** (700–1000 lines)
- Chained PRs recommended: **No**
- Chain strategy: **size-exception**
- Decision needed before apply: **No**

Resolved: single PR on release/2, two revertable work units (Tasks 1+2 = Unit 1; Tasks 3+4 = Unit 2). Ruling (in tasks.md already): auto-chain + single series. Proceed.

## Work Units & Gates

- Unit 1 (Tasks 1+2): Completer + config + AttachSharedLLM + fail-open → gates G1–G8
- Unit 2 (Tasks 3+4): install provider + close task-029 → gates G9, G10, G11

## Task ownership & dependencies

- Task 1: Owns: skillgrid-cli/internal/mnemonic/llm/*, skillgrid-cli/internal/mnemonic/config/load.go(+test) :: Needs: (none)
- Task 2: Owns: skillgrid-cli/internal/mnemonic/service/attach_llm.go(+test), service.go (openProject attach call) :: Needs: Task 1
- Task 3: Owns: skillgrid-cli/internal/install/provider.go(+test), config.go, install.go, cmd/skillgrid/main.go :: Needs: Task 1 (yaml keys)
- Task 4: Owns: .backlog/tasks/task-029* :: Needs: Task 2 (and ideally Task 3)

## Branch integration gate

- G-int-1: after Task 2 — openProject boot path attaches the client and all four seams (or documented lazy-dream) hold end-to-end; parent re-verifies G5–G8.
- G-int-2: after Task 3 — install provider step composes with existing Run; parent re-verifies G10–G11.

## Progress

Task 1: (pending)
Task 2: (pending)
Task 3: (pending)
Task 4: (pending)
