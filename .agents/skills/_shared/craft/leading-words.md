# Leading Words

The compression and steering lever that keeps a skill short *and* gets the
agent to actually do the thing. Cite this for the lever and its tests;
`skill-creator` §4 owns the process of *finding* the words for a given skill.

## What it is

A **leading word** is a compact concept already living in the model's
pretraining that the agent thinks with while running the document —
`lesson`, `fog of war`, `tracer bullet`, `vertical slice`, `seam`, `tight`,
`red`. Repeated as a **token**, never as a sentence, it recruits priors the
model already holds for free, and it anchors behaviour at the fewest possible
tokens.

## The two tests (run both before keeping a word)

### 1. The no-op test (model-relative, not reader-relative)

Does the word change behaviour **versus the model's default**? A leading word
too weak to beat the default is itself a no-op — it pays load to say nothing.
The test is settled by *running* the document, not by debate: two people
disagreeing over whether a word is a no-op are disagreeing over the default,
and the eval settles it.

- **The fix for a weak word is a stronger word, not a different technique.**
  `thorough` (the agent is already thorough-ish) is a no-op; `relentless`
  beats the default. Sharpen the word before reaching for a new mechanism.

### 2. Pretrain-first, define-second

Reach for an existing pretrained word before coining one. A pretrained word
recruits priors for free; a coined word pays in definition tokens what the
pretrained word gives free. Coining works only when the concept genuinely has
no pretrained home, and then you must define it clearly on first use.

## Positive over negation

State the **target** behaviour; never steer by prohibition. The ban drags the
banned behaviour into context and makes it *more* available, not less — and a
negation is a weak modifier the strongly-activated concept overruns, so the
ban half-reads as an instruction to do the thing.

- Say `write one-line comments`, not `don't write long comments`.
- A prohibition earns its place only as a hard guardrail you cannot phrase
  positively; even then, pair it with the positive target so attention lands on
  what to do.

## The collapse pattern

A triad spelled out at three sites, or a pointer spending a sentence to gesture
at one idea, is a passage begging to become a single token. Hunt for them:

- `fast, deterministic, low-overhead` → **`tight`** (a *tight* loop).
- `a loop you believe in` → **`red`** — turns a fuzzy gate into a binary
  observable (the loop goes *red* on the bug, or it doesn't).

Each collapse wins twice: fewer tokens, and a sharper hook for the agent to
hang its thinking on.

## It anchors twice

- **In the body, execution.** The agent reaches for the same behaviour every
  time the token appears; inside flat reference it focuses attention on a
  class of thing to look for.
- **In a pointer, invocation.** When the same word lives in your prompts, your
  docs, and the codebase, the agent links that shared language to the material
  and reaches it more reliably. A leading word in a description is a trigger
  hook, not just body steering — front-load it (see the description rules in
  `skill-creator` §2).

## How to know it worked

Watch the reasoning traces — is the agent **re-emitting** the word? Re-emission
is the signal the prior was recruited. If not, the word is too weak (run the
no-op test) or too inconsistently placed.
