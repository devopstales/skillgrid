# llm-wiki Operations: ingest, query, lint, bulk, archive

Deep procedures for `skillgrid:llm-wiki`. Tool mapping: `read_file` → Read,
`search_files` (content) → Grep with `*.md` glob, (filenames) → Glob,
`web_extract` → Exa fetch / WebFetch / mnemonic webcache, `terminal` → Bash,
`execute_code` → `python3 -c` via Bash. `$WIKI` is the active target
(`~/wiki`, `~/.skillgrid/wiki`, or project `.wiki/` read-only).

## Ingest

① **Capture raw.** URL → fetch to markdown, save `raw/articles/<slug>.md`;
papers → `raw/papers/`; pastes/notes → matching `raw/` subdir. Add raw
frontmatter (`source_url`, `ingested`, `sha256` of the body only). On re-ingest:
recompute, skip if identical, flag drift + update if different.

② **Takeaways.** Discuss what's interesting / domain-relevant with the user
(skip in automated contexts — proceed).

③ **Dedupe-check.** Search `index.md` + Grep existing pages for every
mentioned entity/concept before creating anything.

④ **Write/update pages.** New pages only at SCHEMA thresholds; existing pages
gain facts + `updated` bump; contradictions follow the Update Policy (dates,
both positions, `contradictions:` frontmatter, flag for review); ≥ 2
`[[wikilinks]]` each way; taxonomy tags only; `^[...]` provenance markers on
3+ source syntheses; `confidence: medium/low` unless multi-source supported.

⑤ **Navigation.** New pages into `index.md` alphabetically under type;
bump total count + date; append `## [YYYY-MM-DD] ingest | <title>` with every
file touched.

⑥ **Report** every file created/updated. One source routinely touches 5–15
pages — that compounding is the point.

## Query

Read `index.md` → (100+ pages: also Grep key terms) → Read relevant pages →
synthesize citing `[[page-a]]`/`[[page-b]]` → file substantial answers
(comparisons, deep dives, novel syntheses) into `queries/`/`comparisons`,
never trivial lookups → log the query + filed-or-not.

## Lint (all thirteen, severity-grouped report + log line)

1. Orphans (zero inbound `[[wikilinks]]`, programmatic scan). 2. Broken
wikilinks (target missing). 3. Index completeness (filesystem vs index).
4. Frontmatter validity (required fields; tags in taxonomy). 5. Staleness
(`updated` > 90 days behind newest mentioning source). 6. Contradictions
(shared tags/entities, conflicting claims; surface `contested:` pages).
7. Quality signals (`confidence: low`; single-source pages with no confidence
field). 8. Source drift (recompute `raw/` sha256; report mismatches).
9. Page size (> 200 lines → split candidates). 10. Tag audit (in-use vs
taxonomy). 11. Log rotation (> 500 entries → `log-YYYY.md`). 12. Report with
file paths + actions, ordered: broken links > orphans > drift > contested >
stale > style. 13. Append `## [YYYY-MM-DD] lint | N issues`.

## Bulk ingest

Read all sources → identify entities/concepts across all → one dedupe search
pass → create/update in one pass → single index update → single log entry.

## Archiving

Superseded/out-of-scope page → `_archive/` preserving relative path → remove
from `index.md` → replace inbound wikilinks with plain text + "(archived)" →
log the action.
