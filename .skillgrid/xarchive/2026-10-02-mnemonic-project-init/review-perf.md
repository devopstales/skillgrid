# Performance review — mnemonic-project-init

> Specialist: performance · Range `f739d371..17652346` · Diff: `review-range.diff`
> Focus: WalkDir over docs/artifacts, per-file `memory.Save`, `RunCodeIndex` cost, double `OpenForDirectory`

## Findings (JSON)

```json
[
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:55",
    "issue": "WalkDir ingests every regular file with no type or size filter",
    "cost_and_scale": "Binary assets under docs/ (≈3MB PNGs here) become SQLite observation blobs per init",
    "fix": "Limit to text extensions and enforce a MaxFileSize skip like codeindex"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:172",
    "issue": "Each file calls memory.Save with full AUDN and candidate queries",
    "cost_and_scale": "About 4–6 SQLite round-trips per file; bites at tens to hundreds of docs",
    "fix": "Wrap ingest in one transaction or a bulk topic_key upsert path"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd.go:120",
    "issue": "RunCodeIndex opens the store then OpenForDirectory opens it again",
    "cost_and_scale": "Two store.Open/migrate cycles per init once the pool refcount hits zero",
    "fix": "Open once; run codeindex.New(handle.Store()).Run then ingest on same handle"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd.go:96",
    "issue": "init always runs full codeindex Scan even on a warm index",
    "cost_and_scale": "Full-tree walk each init; large repos pay seconds even when FilesIndexed is 0",
    "fix": "Reuse one open with ingest; skip Scan only if product allows warm short-circuit"
  }
]
```

## Notes (for coordinator)

- Spec requires calling `RunCodeIndex` on init; do not drop the call. Finding 4 is about residual Scan cost and sharing the open (finding 3), not removing the index step.
- Evidence for finding 1: this repo’s `docs/assets/` holds ~1MB PNG ×3 plus JPEGs; WalkDir has no `.md` filter or size cap (unlike `codeindex.MaxFileSize` 500KiB).
- `ResolveProject` in `initRunIndex` (line 93) is a cheap extra resolve before `RunCodeIndex` re-resolves; not filed separately.
