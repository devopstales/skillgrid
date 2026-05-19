#!/usr/bin/env bash
# Resume, list, or validate session handoffs under .skillgrid/handoffs/
# Usage:
#   handoff-resume.sh list
#   handoff-resume.sh [latest|path] [max-age-days]
set -euo pipefail

if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
  echo "Error: run inside a git repository."
  exit 1
fi

repo_root="$(git rev-parse --show-toplevel)"
handoff_input="${1:-latest}"
max_age_days="${2:-3}"

if [[ "${handoff_input}" == "list" ]]; then
  handoff_dir="${repo_root}/.skillgrid/handoffs"
  if [[ ! -d "${handoff_dir}" ]]; then
    echo "No handoff directory found at ${handoff_dir}"
    exit 0
  fi
  ls -1t "${handoff_dir}"/*.md 2>/dev/null || echo "No handoff files found."
  exit 0
fi

if [[ "${handoff_input}" == "latest" ]]; then
  handoff_path="$(ls -1t "${repo_root}"/.skillgrid/handoffs/*.md 2>/dev/null | sed -n '1p')"
else
  handoff_path="${handoff_input}"
fi

if [[ -z "${handoff_path:-}" ]]; then
  echo "No handoff file found. Create one with:"
  echo "  .skillgrid/scripts/handoff-create.sh full <slug>"
  exit 1
fi

echo "Resume handoff: ${handoff_path}"

if [[ -x "${repo_root}/.skillgrid/scripts/handoff-validate.sh" ]]; then
  echo ""
  echo "Validation:"
  "${repo_root}/.skillgrid/scripts/handoff-validate.sh" "${handoff_path}" || true
fi

if [[ -x "${repo_root}/.skillgrid/scripts/handoff-check-staleness.sh" ]]; then
  echo ""
  echo "Staleness:"
  "${repo_root}/.skillgrid/scripts/handoff-check-staleness.sh" "${handoff_path}" "${max_age_days}" || true
fi

echo ""
echo "Resume checklist:"
echo "1) Read handoff file fully."
echo "2) Read active change handoff: .skillgrid/tasks/context_<change-id>.md (if present)."
echo "3) Read event log: .skillgrid/tasks/events/<change-id>.jsonl (if present)."
echo "4) Read cited research files under .skillgrid/tasks/research/<change-id>/."
echo "5) Continue from 'Next steps' and confirm the 'Next recommended action'."
