# Orchestration

Dispatch-time discipline for the orchestrator. Load when a Router row leads to `subagent-execution` or `parallel-execution`, or whenever you are about to dispatch more than one subagent. The routing-time anti-patterns (router persona, persona calls persona, catalog scan) stay in `SKILL.md`.

## Dispatch anti-patterns

These failure modes mean STOP — you've turned orchestration into extra work:

| Anti-pattern | Why it's wrong | Do this instead |
|---|---|---|
| **Depth > 1** — a subagent spawns a subagent that spawns a subagent. | Each hop re-reads context. Beyond one level the cost exceeds the benefit. | Max dispatch depth is 1. Workers do not dispatch. |
| **Paraphrasing hop** — the orchestrator re-summarizes a subagent's result before passing it on. | The worker's output is already structured. | Pass the worker's Return Envelope through verbatim. Synthesize only at the final decision (for example `ship`'s GO/NO-GO). |
| **Orchestrator that edits** — the coordinating agent edits code between dispatches. | Mixing coordination with edits muddies who owns the change. | The orchestrator collects verdicts and renders decisions. It edits only to wire up what workers produced. |

**Choosing a pattern — walk this before dispatching:**

```
Is the work one perspective on one artifact?
├── Yes → Direct invocation (a single skill). Stop.
└── No  → Will the same composition repeat?
         ├── No  → Ad-hoc, single skill. Stop.
         └── Yes → Are the sub-tasks independent?
                  ├── No  → Sequential pipeline, user-driven between steps.
                  └── Yes → Parallel fan-out with a merge step in the main context.
                           If the merge would not fit in the main context, fall back to a single skill.
```

**Verdict vs investigation — pick the right primitive:**

| | Fan-out (subagents) | Teams (teammates that message each other) |
|---|---|---|
| Sub-agents see | The same diff, different lenses | A shared task list and each other's findings |
| Output | Independent reports → one merge (a verdict on a known artifact) | Adversarial debate → consensus (an investigation among competing hypotheses) |
| Use for | `code-review` fan-out, `ship`'s GO/NO-GO | `structured-debugging` when a single agent would pick the first plausible theory and stop |

A fan-out produces a verdict on something you already have. A team produces a root cause among things you don't yet know. Do not wrap a team-style investigation in a fan-out.
