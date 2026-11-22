# User Gates — Decisions the User Makes With Information

The single source for how a skill presents a decision to the user: gate format,
the per-finding disposition contract, the architecture-option table, and the
honesty tags. Dispatching skills (brainstorm, blueprint, test, execute,
receiving-code-review) name this rule; they do not restate it.

## Gate format invariants

Every interactive decision gate follows this shape:

1. **Artifact first.** The decision-relevant content (draft, report, option
   table) is written to a file. The chat message carries a link, never the
   full content.
2. **Summary before the ask.** The same turn, before the question: what
   changed or what is being decided, the 2-3 key points the user needs, and
   the file link. The ask never arrives cold.
3. **One question.** Strictly one question per gate, the sole tool call in
   the turn (structured question tool if the harness has one, otherwise plain
   chat). No bundled questions, no question lists.
4. **No recommendation labels.** Options are action-oriented and unranked.
   Never `(Recommended)`, `default`, or a bolded first choice — a
   recommendation inside the options is pre-programming through the answer
   choices. If the skill has a genuine default (proceed / continue), it is
   Option 0 by position, not by label.
5. **Yield on ask.** After the gate, the agent stops: zero further tool calls
   until the user answers.
6. **Empty-answer guard.** If the user selects "revise" / "discuss" without
   giving content, do not loop or fail: ask once in chat what is needed and
   yield again.

## Per-finding disposition contract

When a review produces findings and the user must decide what happens to each,
the agent takes a citable stance on **every** finding first, then the user
adjudicates per item.

### Dispositions (exactly one per finding)

- **CONTEST** — the critique is inaccurate or rests on a misunderstanding.
  Cite evidence the reviewer could read: a file and symbol, or a decision
  recorded in the document. "This was intentional" is not evidence.
- **ESCALATE** — the critique is inaccurate, but the evidence is outside
  everything the reviewer could read (team convention, verbal decision). State
  that fact. Do not fix the finding. Give it to the user: this needs their
  knowledge, not their approval.
- **PROPOSE FIX** — the critique is valid. Define the precise replacement or
  structural change.
- **DECLINE** — the finding is a Nit not worth fixing. Record it, do not fix
  it. (A Nit with a worthwhile fix may take PROPOSE FIX instead.)

Rules:

- Do not add context to the artifact only to satisfy a reviewer. Add context
  when a human reader needs it; when only a zero-context reviewer needs it,
  ESCALATE.
- Do not contest a comprehension gap. If a fresh reader could not follow the
  document, the document is missing context.
- Do not change a severity. If a Blocker looks like a Nit, CONTEST it and say
  why.
- The agent wrote the artifact under review. The wish to contest is not
  evidence that the reviewer is wrong.

### Per-item user options

Presented per finding, depending on the disposition:

- PROPOSE FIX → **Accept proposed fix / Skip / Discuss**
- CONTEST → **Agree with contest (skip) / Overrule contest (apply fix) / Discuss**
- ESCALATE → **Confirm (no change) / Add the fact to the document / Discuss**

The review report shows, for each finding: the disposition, the
justification, and the proposed edit. Batch operations ("auto accept" =
apply all PROPOSE FIX, skip the rest) are allowed as a separate gate option
alongside the item-by-item walkthrough.

## Architecture-option gate

When a design decision has more than one defensible answer (data model,
component boundaries, storage, protocol, execution strategy), the agent
presents the options and the user chooses. Mechanical choices with one
defensible answer (naming, formatting) are decided by the agent.

### Option contract

- **Up to three options.** Two variants of the same idea are one option.
  Options must be genuinely distinct in mechanism.
- **No ranking, no default, no pick.** If the user asks the agent to choose
  ("just pick one"), the agent restates the trade-offs and presents the same
  gate again. The agent's own preference may appear as a fact in the
  Advantages/Disadvantages cells, never as a label.
- **Each option carries five cells**, in a table, written to an artifact file
  **before** the gate and linked in chat:

  | Option | Mechanism | Advantages | Disadvantages | Fails-when |
  |---|---|---|---|---|
  | A: <name> | How it works, 1-3 sentences | What it buys | What it costs to build and to operate | The condition under which this option is wrong |

- **Fails-when is mandatory.** An option without a failure condition is a
  preference, not an option. "It is more complex" is not a failure condition;
  "it breaks when the team exceeds 3 services" is.
- After the user chooses, the rejected options and the rejection reason are
  recorded in the artifact (`Alternatives Considered` section:
  `Rejected: <option> — <reason>`) so a later session does not re-propose
  them.

## Honesty tags

Labels that let the user tell what was actually checked:

- **`not run`** — a planned reviewer/scout that timed out or failed. It is
  `not run` in the status table, never "zero findings". Retry once; a second
  failure is `not run`.
- **`[⚠️ UNVERIFIED]`** — a claim resting on an external contract the agent
  could not read (dependency internals, platform behavior, permissions).
  State the risk and agree a verification path.
- **`[⏳ ACK]`** — a factual check that is currently impossible; the risk is
  accepted and an empirical test is scheduled. The tag carries the test that
  will close it.
- **Early-exit phrases** — a reviewer with nothing to report states why:
  "No false compromise detected", "No drift found: all cited symbols verified
  in <files>". An empty findings list with no reason is a failed review.

## Severity vocabulary

Shared across all reviewers and reports: **Blocker** (must fix before
proceeding), **Major** (should fix; deferral needs a recorded reason),
**Nit** (polish; DECLINE is a valid disposition). Reports order findings
Blocker → Major → Nit.
