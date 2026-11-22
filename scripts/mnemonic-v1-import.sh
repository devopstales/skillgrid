#!/usr/bin/env bash
# mnemonic-v1-import.sh — one-shot v1→v2 store import (bash + sqlite3 only).
# --from <v1.sqlite> --to <v2.sqlite> [--dry-run]
# Content-keyed idempotency: each imported row carries the provenance marker
# {"imported_from":"mnemonic-v1","v1_id":<v1 id>}; a re-run adds 0 rows.
set -euo pipefail

from=""; to=""; dry=""
while [ $# -gt 0 ]; do
  case "$1" in
    --from) from="$2"; shift 2 ;;
    --to) to="$2"; shift 2 ;;
    --dry-run) dry=1; shift ;;
    *) echo "usage: mnemonic-v1-import.sh --from <v1.sqlite> --to <v2.sqlite> [--dry-run]" >&2; exit 2 ;;
  esac
done
[ -n "$from" ] && [ -n "$to" ] || { echo "usage: mnemonic-v1-import.sh --from <v1.sqlite> --to <v2.sqlite> [--dry-run]" >&2; exit 2; }
[ -f "$from" ] || { echo "error: v1 store not found: $from" >&2; exit 1; }
[ -f "$to" ]   || { echo "error: v2 store not found: $to" >&2; exit 1; }

q() { sqlite3 "$from" "$1"; }
qt() { sqlite3 "$to" "$1"; }

# pre-flight: the source is a v1 store, the target is a v2 store.
[ "$(q "SELECT type FROM sqlite_master WHERE name='observations_core' AND type='table'")" = "table" ] \
  || { echo "error: source is not a v1 store (no observations_core table)" >&2; exit 1; }
[ "$(q "SELECT type FROM sqlite_master WHERE name='sessions' AND type='table'")" = "table" ] \
  || { echo "error: source has no sessions table" >&2; exit 1; }
tt="$(qt "SELECT type FROM sqlite_master WHERE name='observations'")"
[ "$tt" = "table" ] || { echo "error: target observations is ${tt:-missing}, not a table — is $to a v2 store? (the v1 name is a VIEW)" >&2; exit 1; }
q "PRAGMA busy_timeout=5000;"
qt "PRAGMA busy_timeout=5000;"

# render the observations SELECT with only the satellites the source has.
renderSelect() {
  local joins="" st=0 pr=0 sa=0
  if [ "$(q "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='observation_state'")" = "1" ]; then
    joins="$joins LEFT JOIN v1.observation_state st ON st.observation_id = c.id"; st=1
  fi
  if [ "$(q "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='observation_provenance'")" = "1" ]; then
    joins="$joins LEFT JOIN v1.observation_provenance pr ON pr.observation_id = c.id"; pr=1
  fi
  if [ "$(q "SELECT count(*) FROM sqlite_master WHERE type='table' AND name='observation_stats'")" = "1" ]; then
    joins="$joins LEFT JOIN v1.observation_stats sa ON sa.observation_id = c.id"; sa=1
  fi
  local tool_name="NULL" pinned="0" review_after="NULL" dup="0" last_seen="NULL" importance="0.5"
  if [ "$st" = "1" ]; then pinned="COALESCE(st.pinned, 0)"; review_after="st.review_after"; fi
  if [ "$pr" = "1" ]; then tool_name="pr.tool_name"; fi
  if [ "$sa" = "1" ]; then dup="COALESCE(sa.duplicate_count, 0)"; last_seen="sa.last_seen_at"; importance="COALESCE(sa.importance_score, 0.5)"; fi
  printf '%s' "SELECT c.session_id, c.type, c.title, c.content, $tool_name, c.project, COALESCE(c.scope,'project'), c.topic_key, c.normalized_hash, c.revision_count, $pinned, $review_after, $dup, $last_seen, $importance, json_object('imported_from','mnemonic-v1','v1_id',c.id), c.created_at, c.updated_at, c.deleted_at FROM v1.observations_core c $joins WHERE c.session_id IN (SELECT id FROM v1.sessions) AND NOT EXISTS (SELECT 1 FROM observations o2 WHERE json_extract(o2.provenance,'$.imported_from')='mnemonic-v1' AND json_extract(o2.provenance,'$.v1_id')=c.id)"
}

src_sess="$(q "SELECT count(*) FROM sessions")"
planned_sess="$(qt "ATTACH '$from' AS v1; SELECT count(*) FROM v1.sessions s WHERE NOT EXISTS (SELECT 1 FROM sessions t WHERE t.id = s.id)")"
src_valid_obs="$(qt "ATTACH '$from' AS v1; SELECT count(*) FROM v1.observations_core c WHERE c.session_id IN (SELECT id FROM v1.sessions)")"
sel="$(renderSelect)"
planned_obs="$(qt "ATTACH '$from' AS v1; SELECT count(*) FROM ($sel)")"
orphan="$(q "SELECT count(*) FROM observations_core c WHERE c.session_id NOT IN (SELECT id FROM sessions)")"

if [ -n "$dry" ]; then
  echo "dry-run: would import $planned_sess sessions, $planned_obs observations ($orphan orphan-session skipped); no writes"
  exit 0
fi

qt "
ATTACH '$from' AS v1;
BEGIN;
INSERT OR IGNORE INTO sessions (id, project, directory, started_at, ended_at, summary, title)
  SELECT id, project, directory, started_at, ended_at, summary, title FROM v1.sessions;
INSERT INTO observations
  (session_id, type, title, content, tool_name, project, scope, topic_key, normalized_hash,
   revision_count, pinned, review_after, duplicate_count, last_seen_at, importance,
   provenance, created_at, updated_at, deleted_at)
  $sel;
COMMIT;
"

already_obs=$(( src_valid_obs - planned_obs ))
skipped_obs=$(( orphan + already_obs ))
echo "sessions: $planned_sess imported, $(( src_sess - planned_sess )) skipped"
echo "observations: $planned_obs imported, $skipped_obs skipped ($orphan orphan-session, $already_obs already-imported)"
