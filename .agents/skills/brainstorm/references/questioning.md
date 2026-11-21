# Questioning

The interview loop inside brainstorm. Grill the programmer relentlessly until you share a model of the problem. Mapping the problem as a **design tree** and working its frontier in rounds is how vague intent becomes a decision with a paper trail — the glossary and ADRs are written *during* the interview, not backfilled after.

When to load: when a brainstorm request is ambiguous or the programmer wants to stress-test their thinking. Skip it when the request is fully specified and unambiguous — the clarity gate is the exit, not the entry; don't interview for the sake of it.

## The relationship to the domain model

This loop drives the ADR records underneath it (`_shared/adr-records/`). The interview is where terms crystallize and decisions are made — so the paper trail is written *here*, in the conversation, not batched at the end:

- **Terms:** the moment a domain term is resolved, write it to the terms file (`.skillgrid/projectwiki/glossary/01-business-terms.md` for domain/product/workflow, `02-technical-terms.md` for architecture/platform) — challenge conflicts, sharpen fuzzy language, cross-reference the code. A terms file that only changes after the interview is a summary, not the session's output.
- **ADRs:** the moment a decision clears the bar (hard to reverse + surprising without context + real trade-off), offer to record it as `.skillgrid/projectwiki/decisions/NNNN-slug.md` and add a pointer row to the `## LOCKED` ADR index in `.skillgrid/projectwiki/ASSUMPTIONS.md`. The "why" is freshest right now; a future reader needs exactly this moment's reasoning.

If `conventions.artifacts` is set in `.skillgrid/config.yaml`, use that path for the terms files. The terms files and the ADR files are created lazily on first use.

## The design tree and rounds

Map the problem as a **design tree**: every decision branches into the decisions that hang off it. Work the tree in **rounds**. The **frontier** is every decision whose prerequisites are already settled — the questions you can ask *now* without guessing at answers you haven't heard yet. Ask the whole frontier in one round: number each question and give your recommended answer. Then wait for the programmer's answers before the next round.

Format a round like so:

```
❓ **Q1** — **<question title>**: <question body, may include multiple choices>

➡️ <your recommended answer>

---

❓ **Q2** — **<question title>**: <question body>

➡️ <your recommended answer>
```

Each round the programmer's answers reshape the tree: settled decisions push the frontier outward and unblock questions that depended on them. Recompute the frontier and ask the next round. A question whose answer depends on another question still open in this round belongs to a *later* round, not this one.

## How to ask: the agent's built-in question tool

Present every frontier question through the agent's **built-in question tool** — the native structured-input primitive, not plain chat text. This is the primary channel for all questions in every round.

| Agent | Tool | Notes |
|---|---|---|
| OpenCode | `question` | structured options + free-text fallback |
| Claude Code | `AskUserQuestion` | structured options |
| Codex / other | equivalent native tool | use whatever the host exposes |

Rules for the built-in tool:

- **One round = one call** (or one call per question, depending on host limits). Do not paste the whole frontier as a wall of markdown; use the tool's option mechanism so the programmer gets tappable choices.
- **Every question has options.** Even free-text questions get a short list of realistic choices plus a "type your own" fallback. The options are the frontier's shape made concrete.
- **The `➡️` recommendation is the first option**, labelled "(recommended)" or placed first in the list. The programmer can override.
- **Multiple-choice where the decision is genuinely multi-select** (e.g. "which of these constraints apply?"). Use the tool's multi-select where available.
- **Wait for the answer before the next round.** The tool blocks; do not fire the next round until the programmer has responded.

The plain-text `❓ Q1 / ➡️` format above is the **fallback** when no structured question tool is available (e.g. a plain terminal or a chat-only host). Prefer the tool; fall back to text only when the tool is absent.

## Posting questions (optional persistence, not the question channel)

The built-in question tool is the question channel. The mnemonic decision bridge below is **optional persistence** — a way to keep a durable, team-visible record of the decisions — not the mechanism by which questions are asked. Use it when the programmer has a mnemonic dashboard and wants the decisions visible outside the session.

When the programmer has a mnemonic dashboard, post each frontier question after asking it (via the built-in tool) so the Decisions view is a live inbox. For every question you put to them, `mem_save` it as a governed `type: decision` observation, then widen it so the team-scoped dashboard reader sees it:

When the programmer has a mnemonic dashboard, post each frontier question so the Decisions view is a live inbox. For every question you put to them, `mem_save` it as a governed `type: decision` observation, then widen it so the team-scoped dashboard reader sees it:

```
mem_save(
  type: "decision",
  title: "<question title>",
  topic_key: "interview/<slug>/<question-id>",   # <slug>=the interview, <question-id>=this Q (stable)
  content: {
    "question": "<question body>",
    "options": [ { "id": "a", "label": "<choice>" }, { "id": "b", "label": "<choice>" } ],
    "recommended": "<id of your recommended answer>",
    "state": "pending"
  })
mem_share(id, target: "team")
```

Rules:

- **`content` is JSON, not markdown** — a decision's `content` MUST be the structured payload above (the dashboard parses it; markdown or a missing `state` is excluded from the inbox).
- **`recommended` is your `➡️` answer**, encoded as an option `id` — the UI highlights it. Free-text questions still get options; never a decision with zero options.
- **`topic_key` is stable per question** — re-saving the same key updates the question in place (upsert + version), which is how you revise a recommendation mid-interview.
- **`visual` (optional):** when a question is clearer shown than told and the programmer has accepted the visual companion, write throwaway HTML to `.skillgrid/prototype/<topic>/<variant>.html` and add `"visual": "<topic>/<variant>.html"` to `content` — the card renders it in a sandboxed iframe.
- **Read the answer back by polling.** After the programmer answers (dashboard or chat), poll `mem_search` on a word of the question; when the returned `content.state` is `"answered"`, fold `answeredOption` / `answerNote` into the tree and clear the question. A chat answer the programmer never posts to the dashboard: record it by re-saving the same `topic_key` with `state: "answered"` + the chosen option so the durable trail matches.
- **Skip persistence** when the interview is purely conversational and the programmer has no dashboard — the bridge is a convenience view, not a requirement for the clarity gate.

**Round state:** the glossary and ADRs capture the *output*, but the in-flight position — which round you are on, which questions are settled, which are still open — lives in conversation only. A crashed interview loses it; the decision bridge (when used) is the durable exception.

- When the topic has a spec dir, **commit the glossary and ADR changes after each round** (spec zone). The committed glossary + ADRs ARE the in-flight index.
- If `mnemonic.enabled: true`, mirror the transition with `mem_save(topic_key: skillgrid/<topic>/briefing, ...)` (upsert).
- On resume, read the glossary + ADRs **and the decision bridge** to recompute the frontier: `pending` rows are the unsettled frontier, `answered` rows are settled with their recorded choice. What is in the glossary is settled; what is not is open.

## Rules

- **Finding facts is your job, never the programmer's.** When a frontier question needs a fact from the environment (filesystem, codebase, tools), look it up yourself (or dispatch a subagent). Don't ask the programmer for anything you could find out. Don't block on it: a running lookup is an unsettled prerequisite, so only the questions downstream of it wait; ask the rest of the frontier now.
- **The decisions are the programmer's.** Put each decision to them via the built-in question tool and wait. Never decide for them.
- **One round at a time.** Ask the whole frontier, wait for answers, recompute, next round. Don't ask questions whose prerequisites are still open.
- **Maintain the domain model as you go.** As terms resolve, write them to the glossary inline. As a load-bearing decision clears the ADR bar, offer to record it. The interview ends with the glossary and any ADRs already reflecting what was decided — not a to-do to backfill.

## Exit: clarity gate

The interview is **not** done when the frontier is empty. It's done when the frontier is empty **AND** the clarity gate passes.

Score each dimension 0.0 (completely unclear) to 1.0 (crystal clear):

| Dimension | Weight | Minimum | What it measures |
|---|---|---|---|
| Goal Clarity | 35% | 0.75 | Is the outcome specific and measurable? |
| Boundary Clarity | 25% | 0.70 | What's in scope vs out of scope? |
| Constraint Clarity | 20% | 0.65 | Performance, compatibility, data requirements? |
| Acceptance Criteria | 20% | 0.70 | How do we know it's done? |

**Clarity score** = 1.0 − (0.35×goal + 0.25×boundary + 0.20×constraint + 0.20×acceptance)

**Gate:** clarity ≤ 0.20 AND all dimensions ≥ their minimums → shared understanding reached.

If the gate fails, identify which dimension is lowest, ask targeted questions to raise it, and re-score. Repeat until the gate passes.

**Confidence stop (number, not feel).** A clarity gate you score yourself is self-graded. Before declaring the gate passed, put a number on your confidence that the *programmer* — not your read of the conversation — would confirm each dimension is settled: state "I'm N/10 confident the goal is settled" (and so on) in the chat summary. If any dimension is under 8/10, the gate is *not* passed no matter what your self-score says — the low number names the next question to ask. This converts "I think we're clear" into a falsifiable claim the programmer can veto in one word.

**Out-of-scope line (explicit, not implied).** The Boundary dimension is the one that decays silently. Do not let "what's out of scope" live only in your inference — close the interview with an explicit **Out-of-scope** list in the chat summary (and carry it into the briefing's scope section): the specific adjacent capabilities, systems, or edge cases you are *not* building. Each entry is one line — "<capability> is out of scope because <reason>". An out-of-scope list with zero entries is a red flag, not a clean bill: it usually means the boundary was never stated, only assumed.

**Domain-model completeness (exit check):** before declaring done, confirm the paper trail is current — every resolved term is in the glossary, and every decision that cleared the ADR bar was either recorded or explicitly declined. A glossary untouched by a long interview means the ADR half got skipped; go back and backfill, or say why there was nothing to record.

Record the final scores in the design brief's **Clarity Report** section.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "I can infer the intent" | The decisions are the programmer's. Put each one to them and wait — inference is a guess, not a decision. |
| "One question is enough" | The frontier is every decision whose prerequisites are settled. Ask the whole frontier; one question leaves the rest unblocked-by-nothing. |
| "The programmer will just say 'you decide'" | If they do, that *is* the answer — record it. But you may not decide for them first; the ask is the decision point. |
| "I'll look it up later, after the answers" | Finding facts is your job. A running lookup is an unsettled prerequisite — only the questions downstream of it wait, the rest of the frontier goes out now. |

## Red Flags

- A question asked whose prerequisite is still open in the same round.
- The programmer is asked to provide a fact you could have looked up yourself.
- One round after another with no new frontier — recompute or the frontier is empty and you haven't.
- The glossary is untouched by a long interview: the domain model half was skipped.
- Declaring "done" when the frontier is empty but the clarity gate hasn't been scored.
- A decision recorded as made that the programmer never actually put to.
- The clarity gate passed on a self-score while a dimension was under 8/10 confidence in the programmer's own confirmation.
- An Out-of-scope list with zero entries on a non-trivial design — the boundary was assumed, not stated.

## Verification

- [ ] The clarity gate passed — clarity ≤ 0.20 AND every dimension ≥ its minimum, with final scores recorded in the Clarity Report.
- [ ] The confidence stop was applied: each dimension's "programmer would confirm" number was stated in chat and all were ≥ 8/10.
- [ ] An explicit Out-of-scope list was closed in the chat summary and carried into the briefing's scope section (zero entries flagged, not accepted).
- [ ] Each question targeted a real decision point with a recommended answer, not a yes/no rubber-stamp.
- [ ] No frontier question was asked before its prerequisites were settled.
- [ ] The programmer was never asked for a fact you could have looked up yourself.
- [ ] The paper trail is current: every resolved term is in the glossary, and every ADR-cleared decision was recorded or explicitly declined.
- [ ] When the bridge was used: each posted question is a `type: decision` observation (JSON `content` + `state`), shared to `team`, with a stable `topic_key`; answered questions were read back via `mem_search` and folded into the tree (or the programmer confirmed a chat-only interview).
