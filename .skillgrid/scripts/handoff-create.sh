#!/usr/bin/env bash
set -euo pipefail

mode="${1:-full}"
slug="${2:-session}"
continues_from="${3:-}"
change_id="${4:-}"

if [[ "${mode}" != "full" && "${mode}" != "quick" ]]; then
  echo "Usage: $0 <full|quick> [slug] [continues-from-path] [change-id]"
  exit 1
fi

if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
  echo "Error: run inside a git repository."
  exit 1
fi

repo_root="$(git rev-parse --show-toplevel)"
handoff_dir="${repo_root}/.skillgrid/handoffs"
mkdir -p "${handoff_dir}"

timestamp="$(date -u +"%Y-%m-%d-%H%M%S")"
safe_slug="$(echo "${slug}" | tr '[:space:]' '-' | tr -cd '[:alnum:]-_' | tr '[:upper:]' '[:lower:]')"
[[ -z "${safe_slug}" ]] && safe_slug="session"
safe_slug="$(echo "${safe_slug}" | sed -E 's/[-_]+$//; s/^[-_]+//; s/[-_]{2,}/-/g')"
[[ -z "${safe_slug}" ]] && safe_slug="session"
handoff_path="${handoff_dir}/${timestamp}-${safe_slug}.md"

branch="$(git branch --show-current 2>/dev/null || echo detached-head)"
git_status="$(git status --short || true)"
recent_commits="$(git log --oneline -5 || true)"

template_name="template-handoff-session-full.md"
if [[ "${mode}" == "quick" ]]; then
  template_name="template-handoff-session-quick.md"
fi
template_path="${repo_root}/.skillgrid/templates/${template_name}"

if [[ ! -f "${template_path}" ]]; then
  echo "Error: missing template: ${template_path}"
  exit 1
fi

cp "${template_path}" "${handoff_path}"

generated="$(date -u +"%Y-%m-%d %H:%M UTC")"

sed -i '' "s|<YYYY-MM-DD HH:MM UTC>|${generated}|g" "${handoff_path}"
sed -i '' "s|<task-title>|${slug}|g" "${handoff_path}"
sed -i '' "s|<git-branch>|${branch}|g" "${handoff_path}"

if [[ "${mode}" == "full" ]]; then
  sed -i '' "s|<absolute-project-path>|${repo_root}|g" "${handoff_path}"
  if [[ -n "${continues_from}" ]]; then
    sed -i '' "s|<previous-handoff-path or n/a>|${continues_from}|g" "${handoff_path}"
  else
    sed -i '' "s|<previous-handoff-path or n/a>|n/a|g" "${handoff_path}"
  fi
fi

{
  echo ""
  echo "## Auto-captured Git context"
  echo ""
  echo "### Modified files (git status --short)"
  echo ""
  if [[ -n "${git_status}" ]]; then
    echo "${git_status}" | sed 's/^/- `/' | sed 's/$/`/'
  else
    echo "- `clean working tree`"
  fi
  echo ""
  echo "### Recent commits"
  echo ""
  if [[ -n "${recent_commits}" ]]; then
    echo "${recent_commits}" | sed 's/^/- `/' | sed 's/$/`/'
  else
    echo "- `no commits found`"
  fi
} >> "${handoff_path}"

if [[ -n "${change_id}" && -x "${repo_root}/.skillgrid/scripts/checkpoint-record.sh" ]]; then
  "${repo_root}/.skillgrid/scripts/checkpoint-record.sh" \
    --change "${change_id}" \
    --name handoff-create \
    --trigger handoff-create \
    --phase handoff \
    --evidence "handoff ${mode} snapshot ${handoff_path}" \
    || echo "WARN: checkpoint-record failed (handoff still created)" >&2
fi

echo "${handoff_path}"
