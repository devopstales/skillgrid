#!/usr/bin/env bash
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(git rev-parse --show-toplevel)"
gitdir="$(git rev-parse --git-common-dir)"
bin="$(dirname "$gitdir")/dist/mnemonic"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# 1. the fixture v1 store
sqlite3 "$tmp/v1.sqlite" < "$here/v1.sql"

# 2. a fresh v2 store (the repo binary runs the migrations)
"$bin" memory stats --project fixturetest --data-dir "$tmp/v2data" >/dev/null
v2="$tmp/v2data/fixturetest.sqlite"
[ -f "$v2" ] || { echo "no v2 store created"; exit 1; }

# 3. --dry-run writes nothing
cp "$v2" "$v2.pre"
"$repo"/scripts/mnemonic-v1-import.sh --from "$tmp/v1.sqlite" --to "$v2" --dry-run | grep -q 'dry-run'
cmp -s "$v2" "$v2.pre" || { echo "dry-run wrote to the target"; exit 1; }

# 4. the import: rows land, search finds them
out="$("$repo"/scripts/mnemonic-v1-import.sh --from "$tmp/v1.sqlite" --to "$v2")"
echo "$out" | grep -q '^sessions: 2 imported, 0 skipped$' || { echo "fresh sessions counts line wrong: $out"; exit 1; }
echo "$out" | grep -q '^observations: 3 imported, 1 skipped (1 orphan-session, 0 already-imported)$' || { echo "fresh observations counts line wrong: $out"; exit 1; }
n=$(sqlite3 "$v2" "SELECT count(*) FROM observations")
[ "$n" = "3" ] || { echo "want 3 observations, got $n"; exit 1; }
s=$(sqlite3 "$v2" "SELECT count(*) FROM sessions")
[ "$s" = "2" ] || { echo "want 2 sessions, got $s"; exit 1; }
sqlite3 "$v2" "SELECT content FROM observations WHERE title='Fixed N+1 query'" | grep -q 'batched fetch' \
  || { echo "fixture content missing"; exit 1; }
# the marker + the satellite values
sqlite3 "$v2" "SELECT json_extract(provenance,'\$.v1_id') FROM observations WHERE title='Fixed N+1 query'" | grep -q '^[0-9]' \
  || { echo "provenance marker missing"; exit 1; }
sqlite3 "$v2" "SELECT pinned FROM observations WHERE title='Fixed N+1 query'" | grep -q '1' \
  || { echo "satellite pinned not imported"; exit 1; }
# the NULL scope coalesced to the v2 default 'project'
[ "$(sqlite3 "$v2" "SELECT scope FROM observations WHERE title='Chose postgres for kb'")" = "project" ] \
  || { echo "null scope not coalesced"; exit 1; }

# 4b. memory search finds the imported content
"$bin" memory search "batched fetch" --project fixturetest --data-dir "$tmp/v2data" --limit 5 --json > "$tmp/search.out"
grep -q 'batched fetch' "$tmp/search.out" || { echo "memory search did not find the v1 content"; exit 1; }

# 5. re-run adds 0
out="$("$repo"/scripts/mnemonic-v1-import.sh --from "$tmp/v1.sqlite" --to "$v2")"
echo "$out" | grep -q '^observations: 0 imported, 4 skipped (1 orphan-session, 3 already-imported)$' || { echo "re-run not idempotent: $out"; exit 1; }
echo "$out" | grep -q '^sessions: 0 imported, 2 skipped$' || { echo "re-run sessions counts line wrong: $out"; exit 1; }
n=$(sqlite3 "$v2" "SELECT count(*) FROM observations")
[ "$n" = "3" ] || { echo "re-run changed the count: $n"; exit 1; }

# 6. the v1-store-as-target guard
guard_out="$( "$repo"/scripts/mnemonic-v1-import.sh --from "$tmp/v1.sqlite" --to "$tmp/v1.sqlite" 2>&1 || true )"
printf '%s' "$guard_out" | grep -q 'VIEW' \
  || { echo "v1-target guard did not fire"; exit 1; }

echo "v1-import fixture test: all checks passed"
