# SDD ledger — plan: docs/skillgrid/changes/010-mnemonic-framework-routes-affected/tasks.md

## Steps
- Task 01 (framework-routes): COMPLETE (d3ca0fc..8c099a5). Verdict PASS.
- Task 02 (code-affected): COMPLETE (d068db4..c109f2f). Verdict PASS.
- Task 03 (autosync-watcher): PENDING.

## Task 02 detail
- Implementer: d068db4 (affected traversal) + 8288d21 (--stdin) + d708a64 (--base merge-base + areas + owners) + e0a9a57 (rename buckets) + b59c0b1 (rename apply) + dd593b9 (tools registration).
- Review: approved with fixes — package-level pathCache (stale-path/parallelism) + parseBlameAuthor digit-truncation.
- Fix round 1/5: 22aef81 (pathCache scoped per-call) + c109f2f (structural blame-author parse). Re-review: both ADDRESSED, no new breakage.
- code_affected: query-only (no node/edge writes), resolved-edges-only blast radius (drop policy), --stdin, --base merge-base diff + areas + git-blame owners (hermetic temp repo), --json/--quiet.
- code_rename: graph (high-conf, e.to_id=target.ID) vs text-search (low-conf) buckets; ambiguous -> ranked candidates; dry_run default true; non-dry edits only planned files, no commit/push.
- Tool surface 71->73 additive. No step-02 migration (pure code).
- Parallel setup/install refactor had LANDED by this step (go build ./... + all e2e green).

## Next
- Task 03 (autosync-watcher): fsnotify (pure-Go) watcher + staleness banner + fingerprint gate + warm embedder. Blocked by: 02 (done).

## Task 03 detail (autosync-watcher)
- Implementer: c436a30 (watcher+debounce) + 9d097eb (copy-and-swap) + 07bec0a (auto-reopen) + eeb9275 (fingerprint gate) + cb49fcf (staleness banner) + bee853c (writer lock + catch-up) + 8dad267 (warm load-once) + 2b0d3aa (warm evict+heartbeat) + 53057ea (index status + serve wiring) + 8b68453 (tool-surface stability).
- Review: approved with fixes — (a) fingerprint gate embedder-free was comment-only; (b) warmSearchEmbedder read CWD not h.root; (c) failed-factory FTS branch untested.
- Fix round 1/5: 981f128 (counting-spy counterfactual for embedder-free) + 3da7fbb (h.root config) + 1ee735f (failed-factory FTS test).
- Re-review: 2 & 3 ADDRESSED, 1 PARTIAL-but-improved (gate embedder-free enforced by not attaching via ReindexStructural; generic Run with attached embedder legitimately embeds — a flag-free design, not a bug).
- Controller follow-up: ca15bb8 (clarified the embedder-free test rationale after a wrong direct-attach form was tried — reverted to fresh unattached gate Indexer + counterfactual; no behavior change).
- fsnotify pure-Go; 73-tool baseline unchanged; no step-03 migration.

## 010: ALL 3 STEPS APPLIED — State: verify
- Next: sdd-verify (independent quality gate over all 3 steps) → sdd-archive.
