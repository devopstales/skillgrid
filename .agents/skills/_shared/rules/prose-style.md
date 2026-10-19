# Prose Style (agent-style v0.9.0 — generation-time rules)

Single source of truth for **how prose is written** in this repo — Markdown the
agent authors or revises: specs, blueprints, ADRs, PRDs, reports, changelogs,
commit-adjacent docs, README sections. When prose disagrees with this file,
**this file wins**. Code *structure* and *naming* live in
[code-standards.md](code-standards.md); this file is the **writing** layer and
does not touch code identifiers.

**Upstream:** vendored from [agent-style v0.9.0](https://github.com/yzhao062/agent-style)
(CC-BY-4.0, Yue Zhao) — the literature-backed 21-rule set (canonical rules from
Strunk & White 1959, Orwell 1946, Pinker 2014, Gopen & Swan 1990; field-observed
rules from 2022–2026 LLM-output maintenance). Full directive text, BAD/GOOD
examples, severity, and the house-style section live at
[`../../code-style/style-review/references/rules.md`](../../code-style/style-review/references/rules.md).
This file carries the compact directives that reach context at write time; the
full bodies load only when judging a draft (per the escape hatch below).

Companion contracts:
- [code-standards.md](code-standards.md) — how code is designed and structured (sister file; this one is prose, that one is code).
- `skillgrid:style-review` — the post-hoc audit that scores a finished file against these same 21 rules.
- [cite-dont-restate.md](../craft/cite-dont-restate.md) — cite decisions by ID; RULE-H (below) is the prose-side instance of the same discipline.

## Escape hatch (meta-principle)

> *"Break any of these rules sooner than say anything outright barbarous."*
> — George Orwell, "Politics and the English Language" (1946), Rule 6

Most rules are **defaults to apply with judgment**. Depart when the sentence
reads better for it, not to vary the pattern. A few are **firm** — they never
yield: do not invent citations, evidence, or numbers; leave no placeholders in
finished text. Judge a draft by its **overall pattern**: one that obeys every
default in every sentence reads as machine-made. When a rule fights the
sentence, drop the rule.

## The 21 rules (compact directives)

Canonical rules (Strunk & White 1959, Orwell 1946, Pinker 2014, Gopen & Swan 1990):

- **RULE-01 Curse of knowledge**: Name your intended reader; do not assume they share your tacit knowledge.
- **RULE-02 Passive voice**: Prefer active voice when the agent is known and worth naming.
- **RULE-03 Concrete language**: Prefer concrete, specific terms over abstract category words like "factors" or "aspects"; state a result's pattern and rounded magnitude, say in plain words whether a difference holds, and leave standard deviations, intervals, and full numbers to tables. Never call a difference within noise a gain. Do not stack figures in consecutive sentences, and keep a caption to what the display shows. Report differences in points or percent rather than raw decimals, keep one unit and sign across text, tables, and captions, and add a table when several figures matter and no display holds them.
- **RULE-04 Needless words**: Cut filler phrases like "in order to", "due to the fact that", "may potentially", and needless sentences that announce, restate the method, or repeat a finding. An abstract goes from the problem and its challenge to the approach and findings, and a conclusion stays short.
- **RULE-05 Dying metaphors**: Delete clichés like "pushes the boundaries", "paradigm shift", or "state of the art".
- **RULE-06 Plain English**: Prefer "use" over "leverage", "method" over "methodology", "feature" over "functionality".
- **RULE-07 Positive form**: Prefer "trivial" to "not important", "forgot" to "did not remember"; do not stage "X, not Y" antithesis ("not just X, but Y") for emphasis.
- **RULE-08 Claim calibration**: Calibrate verbs to evidence; do not write "proves" when the evidence is "suggests"; keep qualifiers that change the claim, and cut caveat tails that add nothing.
- **RULE-09 Parallel structure**: Express coordinate ideas in the same grammatical form.
- **RULE-10 Related words together**: Keep subject close to verb and modifier close to modified; split long parentheticals.
- **RULE-11 Stress position**: Place new or important information at the end of the sentence.
- **RULE-12 Long sentences**: Vary sentence length; split a sentence when the reader must hold more than one open clause to reach its point, and let a long sentence stay if it carries one connected thought.

Field-observed rules (maintainer observation of LLM output, 2022–2026):

- **RULE-A Bullet overuse**: Keep prose in paragraphs when ideas connect; bullets only for genuine lists; avoid forced 3-item triads; table cells hold numbers or short labels.
- **RULE-B Dash overuse**: Do not use em or en dashes as casual sentence punctuation; prefer commas, colons, parentheses, or a new sentence; keep semicolons rare.
- **RULE-C Same-starts**: Do not open two or more consecutive sentences with the same word.
- **RULE-D Transitions**: Do not open sentences with additive-only transitions ("Additionally", "Furthermore", "In addition"), and let "Moreover" or "Notably" open at most one sentence in a paragraph; keep words that mark a contrast, cause, or sequence the text already establishes ("However", "Thus", "First"), and never add a cause just to join sentences.
- **RULE-E Summary closers**: Do not end every paragraph with a sentence that restates its point.
- **RULE-F Term consistency**: Once you define a term or abbreviation, keep using it; do not alternate synonyms.
- **RULE-G Title case**: Use title case for section and subsection headings; articles and short prepositions stay lowercase. A full-sentence bold run-in lead takes sentence case.
- **RULE-H Citation discipline (firm)**: Support factual claims with verifiable citation or concrete evidence; never fabricate citations.
- **RULE-I Contractions**: Prefer "it is" / "does not" / "cannot" over "it's" / "doesn't" / "can't" in formal technical prose.

## House style

The rules start from classic guides and are tuned to one author, Yue Zhao. The
House Style section in
[`../../code-style/style-review/references/rules.md`](../../code-style/style-review/references/rules.md)
lists that author's habits, drawn from first-author ML papers written before AI
writing tools: the moves of an abstract, introduction, results paragraph, and
conclusion; about one sentence in four past 30 words, carrying parenthetical
glosses; and sparing signposts. Where a style rule conflicts with a habit there,
follow the habit, but never against the facts or the firm rules. Read that
section before drafting a paper or proposal section.

## Scope

- **Applies to:** `.md`, `.tex`, `.rst`, `.txt`, and the prose sections of
  source files, config comments, and commit-adjacent docs in this repo.
- **Does not apply to:** code identifiers, function names, file names (see
  [code-standards.md](code-standards.md)), generated files, or prose inside code
  fences.
- **Review counterpart:** `skillgrid:style-review` audits a finished file
  against this same rule set; a file that follows this file while being written
  should clear the audit with a low count.

## Red Flags

- A finished spec or report with a placeholder (`TODO`, `TBD`, `...`) in its body.
- A factual claim with no citation and no `ASSUMED` flag (RULE-H, firm).
- Two paragraphs where every sentence opens with the same word (RULE-C) or ends
  by restating its own point (RULE-E).
- "Additionally", "Furthermore", or "Moreover" opening sentences the reader did
  not need handed to them (RULE-D).
- A heading in sentence case where the rest of the document uses title case
  (RULE-G).
