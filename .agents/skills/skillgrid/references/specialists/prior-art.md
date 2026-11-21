# Prior-art specialist

Find the programs that already do this job, learn what they got right and wrong, and give the blueprint something measured to stand on.

When to load: Phase 0 (map), Phase 2 (design evidence), Phase 7 (what changed). Role: researcher.

## What to produce

A map of two to four comparable projects, each with:

| Field | Content |
|---|---|
| Name and link | the project, its repo and its docs |
| What it does | one sentence |
| Stack and shape | language, distribution, program class |
| What it does well | the pattern worth borrowing, quoted from its docs or code |
| Where it hurts | the failure mode its issue tracker or docs admit |
| What we can own | the gap this program can fill, in one sentence |
| Source and date | the URL and the day it was checked |

Plus a short list of measured benchmarks that apply (latency, memory, build time, cost), each with its source.

## Method

1. Search for the job, not the name. "CLI that watches a folder and syncs it" finds more than "sync tool".
2. Read the README, the getting-started page, and the open issues labelled bug or breaking. The issues are where the comps tell the truth.
3. Prefer primary sources: the project's own docs, repo and releases. A blog post is a lead, not evidence.
4. For each claim, write where it came from. Without a source, mark it **unverified hypothesis** and say so in the brief.

## Using the map

- Phase 0: the comps define what "good" looks like for this class of program and feed the panel's primary and hostile users.
- Phase 2: a design choice cites a comp or a benchmark. "Go services are fast" is not evidence; "the comp's release notes cut cold-start from 400 ms to 90 ms by precompiling" is.
- Phase 7: update the map with what this program now does differently. That paragraph is the release note's honest claim.

## Rules

- No browsing means the map is provisional. Say so, and do not let a provisional comp justify a one-way door.
- Do not copy a comp's architecture because it is popular. Name the property this program needs and whether the comp actually has it.
- A comp that failed is as useful as one that succeeded. Record why it stopped.
- Keep the map in `findings.md`, with sources, so the next run does not repeat the search.
