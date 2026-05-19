#!/usr/bin/env bash
# sdd-gate.sh — Executable enforcement gates for SDD workflows.
#
# Converts procedural rules into programmatic gates that exit non-zero on failure.
#
# Usage:
#   sdd-gate.sh <phase> --change <change-id> [options]
#   sdd-gate.sh apply --change my-feature
#   sdd-gate.sh verify --change my-feature
#
# Exit codes:
#   0 = all gates passed
#   1 = gate failure (one or more gates rejected the phase)
#   2 = usage error (wrong phase, missing --change flag, etc.)
#   3 = unexpected error
#
# Options:
#   --change <name>    Required. Change ID (openspec/changes/<name>/).
#   --skip <gate>      Skip a specific gate by name (repeatable).
#   --report           Output gate results as JSON.
#   --strict           Fail on warnings too (default: only errors fail).

set -uo pipefail

cd "$(dirname "$0")/../.." || exit 3  # navigate to repo root from scripts dir

# --- Global state -----------------------------------------------------------

PHASE=""
CHANGE=""
SKIPPED_GATES=()
REPORT_JSON=false
STRICT=false
ERRORS=()
WARNINGS=()
GATES_RUN=()

# --- helpers ----------------------------------------------------------------

is_gate_skipped() {
  local gate="$1"
  for sg in "${SKIPPED_GATES[@]+"${SKIPPED_GATES[@]}"}"; do
    [[ "$sg" == "$gate" ]] && return 0
  done
  return 1
}

add_error() {
  local gate="$1"
  local msg="$2"
  ERRORS+=("${gate}: ${msg}")
  echo "GATE[${gate}] ERROR: ${msg}" >&2
}

add_warning() {
  local gate="$1"
  local msg="$2"
  WARNINGS+=("${gate}: ${msg}")
  if "$STRICT"; then
    add_error "$gate" "$msg"
  else
    echo "GATE[${gate}] WARNING: ${msg}" >&2
  fi
}

add_pass() {
  GATES_RUN+=("${1}: pass")
}

add_run() {
  GATES_RUN+=("${1}: skipped")
}

is_gate_active() {
  local gate="$1"
  ! is_gate_skipped "$gate"
}

# --- CLI parsing ------------------------------------------------------------

parse_args() {
  if [[ $# -lt 1 ]]; then
    echo "Usage: sdd-gate.sh <phase> --change <name> [options]" >&2
    echo "  Phases: brainstorm propose spec design tasks apply verify archive" >&2
    exit 2
  fi

  PHASE="$1"
  shift

  local change_set=false
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --change)
        CHANGE="$2"; shift 2; change_set=true ;;
      --skip)
        SKIPPED_GATES+=("$2"); shift 2 ;;
      --report) REPORT_JSON=true; shift ;;
      --strict) STRICT=true; shift ;;
      *)
        echo "Unknown option: $1" >&2; exit 2 ;;
    esac
  done

  if ! "$change_set"; then
    echo "ERROR: --change is required" >&2; exit 2
  fi

  case "$PHASE" in
    brainstorm|propose|spec|design|tasks|apply|verify|archive) ;;
    *)
      echo "ERROR: Unknown phase '$PHASE'" >&2; exit 2 ;;
  esac
}

# --- individual gates -------------------------------------------------------

# --- artifact helpers -------------------------------------------------------

change_has_ui_scope() {
  local proposal="openspec/changes/${CHANGE}/proposal.md"
  [[ -f "$proposal" ]] && grep -qiE 'ui_scope:[[:space:]]*true' "$proposal"
}

change_has_spec_file() {
  [[ -n "$(find "openspec/changes/${CHANGE}/specs" -name 'spec.md' 2>/dev/null | head -1)" ]]
}

# 1. Label validation (wraps existing validate-task-labels.sh)
gate_labels() {
  local validator=".skillgrid/scripts/validate-task-labels.sh"
  local tasks_file="openspec/changes/${CHANGE}/tasks.md"

  if ! is_gate_active "labels"; then add_run "labels"; return; fi

  if [[ ! -f "$tasks_file" ]]; then
    case "$PHASE" in
      brainstorm|propose|spec|design)
        add_pass "labels" "tasks.md not required for ${PHASE}"
        return
        ;;
    esac
    add_error "labels" "${tasks_file} does not exist"
    return
  fi

  if [[ ! -f "$validator" ]]; then
    add_warning "labels" "Validator not found at ${validator} — cannot enforce label rules"
    return
  fi

  local output
  if ! output="$("$validator" "$tasks_file" 2>&1)"; then
    add_error "labels" "Label validation failed:
$(printf '%s' "$output" | head -10)"
  else
    add_pass "labels" "Label validation passed"
  fi
}

# 2. Artifact existence per phase
gate_artifacts() {
  local base="openspec/changes/${CHANGE}"
  if ! is_gate_active "artifacts"; then add_run "artifacts"; return; fi

  local -a required=()
  case "$PHASE" in
    propose)
      required=("proposal.md")
      ;;
    brainstorm)
      required=("proposal.md" "design.md" "tasks.md")
      ;;
    spec)
      required=("proposal.md")
      ;;
    design)
      required=("proposal.md" "design.md")
      ;;
    tasks)
      required=("proposal.md" "design.md")
      ;;
    apply|verify|archive)
      required=("proposal.md" "design.md" "tasks.md")
      ;;
  esac

  local -a missing=()
  for a in "${required[@]}"; do
    [[ -f "${base}/${a}" ]] || missing+=("$a")
  done

  case "$PHASE" in
    brainstorm|spec|design|tasks|apply|verify|archive)
      change_has_spec_file || missing+=("specs/**/spec.md")
      ;;
  esac

  if [[ "$PHASE" == "brainstorm" ]] && change_has_ui_scope; then
    [[ -f "${base}/ui-wireframes.md" ]] || missing+=("ui-wireframes.md")
    [[ -f "${base}/ui-decisions.md" ]] || missing+=("ui-decisions.md")
  fi

  if [[ ${#missing[@]} -gt 0 ]]; then
    add_error "artifacts" "Missing for ${PHASE}: ${missing[*]}"
  else
    add_pass "artifacts" "All required artifacts present for ${PHASE}"
  fi
}

# 3. Phase state (locked/archived changes)
gate_phase_state() {
  local lock_file="openspec/changes/${CHANGE}/.locked"
  if ! is_gate_active "state"; then add_run "state"; return; fi

  if [[ -f "$lock_file" ]]; then
    add_error "state" "Change ${CHANGE} is locked — ${PHASE} not allowed"
    return
  fi

  # Check openspec status via CLI if available (avoid matching "blocked by:" dependency hints)
  if command -v openspec &>/dev/null; then
    local status_output
    status_output="$(openspec status --change "$CHANGE" 2>/dev/null)" || true
    if echo "$status_output" | grep -qiE '(^|[[:space:]])status:[[:space:]]*blocked|change[[:space:]]+blocked|blocked[[:space:]]*—'; then
      add_error "state" "OpenSpec status reports blocked: ${status_output}"
    else
      add_pass "state" "Change ${CHANGE} is in ${PHASE}-able state"
    fi
  else
    add_pass "state" "openspec CLI unavailable — no state check"
  fi
}

# 4. Persona routing
gate_persona_routing() {
  local persona="${PERSONA:-}"
  if ! is_gate_active "persona"; then add_run "persona"; return; fi

  if [[ -z "$persona" ]]; then
    if [[ "${SKIP_PERSONA:-}" == "1" ]]; then
      add_run "persona"
    else
      add_warning "persona" "PERSONA env not set — routing not enforced for ${PHASE}"
    fi
    return
  fi

  case "$persona" in
    odin|thor|tyr|heimdall|frigg|loki|mimir|bragi)
      add_pass "persona" "Known Nordic persona '${persona}'"
      ;;
    *)
      add_error "persona" "Persona '${persona}' is not a valid Nordic persona for ${PHASE}"
      ;;
  esac
}

# 5. Slice gating (HITL blocks apply)
gate_slices() {
  local tasks_file="openspec/changes/${CHANGE}/tasks.md"
  if ! is_gate_active "slices"; then add_run "slices"; return; fi

  if [[ "$PHASE" != "apply" ]] && [[ "$PHASE" != "verify" ]]; then
    add_pass "slices" "N/A for ${PHASE}"
    return
  fi

  if ! is_gate_active "labels"; then
    add_run "slices"
    return
  fi

  if [[ -f "$tasks_file" ]]; then
    local hitl_count=0
    hitl_count="$(grep -c '\[Label: HITL\]' "$tasks_file" 2>/dev/null || true)"
    hitl_count="${hitl_count:-0}"
    if [[ "$hitl_count" -gt 0 ]]; then
      add_error "slices" "${hitl_count} HITL slice(s) found — human decision required before implementation"
    else
      add_pass "slices" "No HITL slices blocking ${PHASE}"
    fi
  else
    add_pass "slices" "tasks.md not found (label gate covers this)"
  fi
}

# 6. Two-stage review (verify phase only)
gate_two_stage_review() {
  local research_dir=".skillgrid/tasks/research/${CHANGE}"
  local stage1="${research_dir}/stage1-spec-compliance.md"
  local stage2="${research_dir}/stage2-code-quality.md"
  if ! is_gate_active "two_stage_review"; then add_run "two_stage_review"; return; fi

  if [[ "$PHASE" != "verify" ]]; then
    add_pass "two_stage_review" "N/A for ${PHASE}"
    return
  fi

  local -a missing_stages=()
  [[ ! -f "$stage1" ]] && missing_stages+=("stage1-spec-compliance")
  [[ ! -f "$stage2" ]] && missing_stages+=("stage2-code-quality")

  if [[ ${#missing_stages[@]} -gt 0 ]]; then
    add_error "two_stage_review" "Missing review stage(s): ${missing_stages[*]}"
    return
  fi

  # Check for critical findings in stage reports
  local critical_found=false
  grep -q "CRITICAL\|critical.*block\|status: failed" "$stage1" 2>/dev/null && critical_found=true
  grep -q "CRITICAL\|critical.*block\|status: failed" "$stage2" 2>/dev/null && critical_found=true

  if "$critical_found"; then
    add_error "two_stage_review" "Review reports contain critical findings — blocks progression"
  else
    add_pass "two_stage_review" "Both stages passed without critical findings"
  fi
}

# 7. Persona board hard gates (verify/archive)
gate_persona_board() {
  local research_dir=".skillgrid/tasks/research/${CHANGE}"
  if ! is_gate_active "persona_board"; then add_run "persona_board"; return; fi

  if [[ "$PHASE" != "verify" ]] && [[ "$PHASE" != "archive" ]]; then
    add_pass "persona_board" "N/A for ${PHASE}"
    return
  fi

  if [[ -d "$research_dir" ]]; then
    local hitl_files
    hitl_files="$(grep -rl "hitl_required.*true\|\[HitL\]\|HITL" "$research_dir" 2>/dev/null || true)"
    if [[ -n "$hitl_files" ]]; then
      add_error "persona_board" "Board reports indicate unresolved HITL decision"
    else
      add_pass "persona_board" "No unresolved board blocks"
    fi
  else
    add_pass "persona_board" "No board reports for ${CHANGE}"
  fi
}

# --- orchestrate gates for a phase ------------------------------------------

run_gates() {
  case "$PHASE" in
    brainstorm)
      gate_labels; gate_artifacts; gate_phase_state ;;
    propose)
      gate_artifacts; gate_phase_state ;;
    spec)
      gate_labels; gate_artifacts; gate_phase_state ;;
    design)
      gate_labels; gate_artifacts; gate_phase_state; gate_persona_routing ;;
    tasks)
      gate_labels; gate_artifacts; gate_phase_state; gate_persona_routing ;;
    apply)
      gate_labels; gate_artifacts; gate_phase_state; gate_persona_routing; gate_slices ;;
    verify)
      gate_labels; gate_artifacts; gate_phase_state; gate_two_stage_review; gate_persona_board; gate_slices ;;
    archive)
      gate_labels; gate_artifacts; gate_phase_state; gate_two_stage_review; gate_persona_routing; gate_persona_board ;;
  esac

  add_pass "total" "All gates complete"
}

# --- report & exit ----------------------------------------------------------

output_report() {
  if "$REPORT_JSON"; then
    local status="pass"
    (( ${#ERRORS[@]} > 0 )) && status="fail"

    local earr="" warearr="" grarr=""
    for e in "${ERRORS[@]+"${ERRORS[@]}"}"; do earr+="${e}"; done
    for w in "${WARNINGS[@]+"${WARNINGS[@]}"}"; do warearr+="${w}"; done
    for g in "${GATES_RUN[@]}"; do grarr+="${g}"; done

    cat <<EOF
{"phase":"${PHASE}","change":"${CHANGE}","status":"${status}","errors":["${earr}"],"warnings":["${warearr}"],"gates":["${grarr}"]}
EOF
  else
    if (( ${#ERRORS[@]} > 0 )); then
      echo "=== Gate failure ==="
      echo "Phase: ${PHASE} | Change: ${CHANGE}"
      echo "Errors:"
      for e in "${ERRORS[@]}"; do echo "  - $e"; done
      echo ""
      echo "Run with --report for JSON output." >&2
    else
      echo "=== All gates passed ==="
      echo "Phase: ${PHASE} | Change: ${CHANGE} | Gates: ${#GATES_RUN[@]}"
    fi
  fi
}

# --- main --------------------------------------------------------------------

main() {
  parse_args "$@"
  run_gates
  output_report

  (( ${#ERRORS[@]} > 0 )) && exit 1
  exit 0
}

main "$@"
