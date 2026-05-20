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

REPO_ROOT="${SKILLGRID_REPO_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "$REPO_ROOT" || exit 3

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
    echo "  Phases: brainstorm propose spec design tasks apply verify review archive" >&2
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
    brainstorm|propose|spec|design|tasks|apply|verify|review|archive) ;;
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
    apply|verify|review|archive)
      required=("proposal.md" "design.md" "tasks.md")
      ;;
  esac

  local -a missing=()
  for a in "${required[@]}"; do
    [[ -f "${base}/${a}" ]] || missing+=("$a")
  done

  case "$PHASE" in
    brainstorm|spec|design|tasks|apply|verify|review|archive)
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
    kvasir|mimir|thor|tyr|heimdall|frigg|loki|bragi|vidar)
      add_pass "persona" "Known Norse persona '${persona}'"
      ;;
    odin)
      add_warning "persona" "odin is coordinator branding — prefer registry persona+capability for ${PHASE}"
      ;;
    *)
      add_error "persona" "Persona '${persona}' is not a valid Norse persona for ${PHASE}"
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

# --- verify / review ordering helpers ---------------------------------------

research_dir_for_change() {
  printf '%s' ".skillgrid/tasks/research/${CHANGE}"
}

report_has_blocking_findings() {
  local file="$1"
  [[ -f "$file" ]] || return 1
  grep -qE 'CRITICAL|critical.*block|status:[[:space:]]*failed|\*\*FAIL\*\*|^FAIL$|Verdict:[[:space:]]*FAIL|Recommendation:[[:space:]]*CHANGES_REQUESTED' "$file" 2>/dev/null
}

verify_report_passed() {
  local report="openspec/changes/${CHANGE}/verify-report.md"
  [[ -f "$report" ]] || return 1
  report_has_blocking_findings "$report" && return 1
  grep -qiE '\bPASS\b|PASS WITH WARNINGS|status:[[:space:]]*pass' "$report"
}

verify_passed_for_change() {
  local state_per_change=".skillgrid/state/${CHANGE}/verification_status"
  local state_global=".skillgrid/state/verification_status"
  local checkpoint_log=".skillgrid/tasks/checkpoints.log"
  local stage1
  stage1="$(research_dir_for_change)/stage1-spec-compliance.md"

  if [[ -f "$state_per_change" ]] && [[ "$(tr '[:upper:]' '[:lower:]' < "$state_per_change" | tr -d '[:space:]')" == "passed" ]]; then
    return 0
  fi
  if [[ -f "$state_global" ]] && [[ "$(tr '[:upper:]' '[:lower:]' < "$state_global" | tr -d '[:space:]')" == "passed" ]]; then
    return 0
  fi
  if verify_report_passed; then
    return 0
  fi
  if [[ -f "$checkpoint_log" ]] && grep -qE "change=${CHANGE}[[:space:]].*name=verify-pass|name=verify-pass[[:space:]].*change=${CHANGE}" "$checkpoint_log" 2>/dev/null; then
    return 0
  fi
  if [[ -f "$stage1" ]] && ! report_has_blocking_findings "$stage1"; then
    grep -qiE 'PASS|APPROVED|status:[[:space:]]*pass' "$stage1" 2>/dev/null && return 0
  fi
  return 1
}

is_security_sensitive_change() {
  [[ -f ".skillgrid/state/${CHANGE}/security_sensitive" ]]
}

review_security_artifacts_present() {
  local trivy=".agents/reviews/${CHANGE}/trivy-report.json"
  local vuln=".agents/reviews/${CHANGE}/vulnerability-scan.json"
  [[ -f "$trivy" || -f "$vuln" ]]
}

review_truecourse_enabled() {
  grep -qE '"truecourse_enabled"[[:space:]]*:[[:space:]]*true' "${REPO_ROOT}/.skillgrid/config.json" 2>/dev/null
}

review_truecourse_artifacts_present() {
  local analyze=".agents/reviews/${CHANGE}/truecourse-analyze.txt"
  local violations=".agents/reviews/${CHANGE}/truecourse-violations.txt"
  [[ -f "$analyze" && -f "$violations" ]]
}

review_approved_for_change() {
  local state_per_change=".skillgrid/state/${CHANGE}/review_status"
  local research_dir stage2 review_glob f
  research_dir="$(research_dir_for_change)"
  stage2="${research_dir}/stage2-code-quality.md"

  if [[ -f "$state_per_change" ]] && [[ "$(tr '[:upper:]' '[:lower:]' < "$state_per_change" | tr -d '[:space:]')" == "approved" ]]; then
    return 0
  fi
  if [[ -f "$stage2" ]] && ! report_has_blocking_findings "$stage2"; then
    grep -qiE 'APPROVED|status:[[:space:]]*approved|Recommendation:[[:space:]]*APPROVED' "$stage2" 2>/dev/null && return 0
  fi
  shopt -s nullglob
  review_glob=(openspec/changes/${CHANGE}/reviews/*.md)
  shopt -u nullglob
  for f in "${review_glob[@]+"${review_glob[@]}"}"; do
    if grep -qiE 'APPROVED|Recommendation:[[:space:]]*APPROVED|status:[[:space:]]*approved' "$f" && ! report_has_blocking_findings "$f"; then
      return 0
    fi
  done
  return 1
}

# 6. Verify must complete before review (review + archive phases)
gate_verify_before_review() {
  if ! is_gate_active "verify_before_review"; then add_run "verify_before_review"; return; fi

  case "$PHASE" in
    review|archive)
      if verify_passed_for_change; then
        add_pass "verify_before_review" "sdd-verify evidence present for ${CHANGE}"
      else
        add_error "verify_before_review" "Run /sdd-verify first — need PASS in openspec/changes/${CHANGE}/verify-report.md (or verify-pass checkpoint / .skillgrid/state/${CHANGE}/verification_status)"
      fi
      ;;
    *)
      add_pass "verify_before_review" "N/A for ${PHASE}"
      ;;
  esac
}

# 7. Code quality review must complete before archive
gate_review_before_archive() {
  if ! is_gate_active "review_before_archive"; then add_run "review_before_archive"; return; fi

  if [[ "$PHASE" != "archive" ]]; then
    add_pass "review_before_archive" "N/A for ${PHASE}"
    return
  fi

  if ! review_approved_for_change; then
    add_error "review_before_archive" "Run /sdd-review after verify — need APPROVED in openspec/changes/${CHANGE}/reviews/ or stage2-code-quality report"
    return
  fi

  if is_security_sensitive_change && ! review_security_artifacts_present; then
    add_warning "review_before_archive" "security_sensitive change missing scan artifacts under .agents/reviews/${CHANGE}/ (trivy-report.json or vulnerability-scan.json)"
  fi

  if review_truecourse_enabled && ! review_truecourse_artifacts_present; then
    add_warning "review_before_archive" "review.architecture.truecourse_enabled but missing .agents/reviews/${CHANGE}/truecourse-analyze.txt — run run-truecourse-review.sh"
  fi

  add_pass "review_before_archive" "sdd-review evidence present for ${CHANGE}"
}

# 7c. TrueCourse artifacts when architecture review enabled
gate_review_truecourse_artifacts() {
  if ! is_gate_active "review_truecourse_artifacts"; then add_run "review_truecourse_artifacts"; return; fi

  if [[ "$PHASE" != "review" && "$PHASE" != "archive" ]]; then
    add_pass "review_truecourse_artifacts" "N/A for ${PHASE}"
    return
  fi

  if ! review_truecourse_enabled; then
    add_pass "review_truecourse_artifacts" "truecourse_enabled false in config"
    return
  fi

  if review_truecourse_artifacts_present; then
    add_pass "review_truecourse_artifacts" "TrueCourse review artifacts present for ${CHANGE}"
  else
    add_warning "review_truecourse_artifacts" "Run .skillgrid/scripts/run-truecourse-review.sh --change ${CHANGE} (or /sdd-review --architecture)"
  fi
}

# 7b. Review phase: encourage security scan artifacts when sensitive
gate_review_security_artifacts() {
  if ! is_gate_active "review_security_artifacts"; then add_run "review_security_artifacts"; return; fi

  if [[ "$PHASE" != "review" && "$PHASE" != "archive" ]]; then
    add_pass "review_security_artifacts" "N/A for ${PHASE}"
    return
  fi

  if ! is_security_sensitive_change; then
    add_pass "review_security_artifacts" "Change not security_sensitive"
    return
  fi

  if review_security_artifacts_present; then
    add_pass "review_security_artifacts" "Security scan artifacts present for ${CHANGE}"
  else
    add_warning "review_security_artifacts" "Run /sdd-review --security (Trivy) or ensure vulnerability-scan.json for ${CHANGE}"
  fi
}

# 8. Persona hard gates (tyr/heimdall critical, HITL flags in research)
gate_persona_hardgates() {
  local research_dir=".skillgrid/tasks/research/${CHANGE}"
  if is_gate_skipped "persona_hardgates" || is_gate_skipped "persona_board"; then
    add_run "persona_hardgates"
    return
  fi

  if [[ "$PHASE" != "verify" ]] && [[ "$PHASE" != "archive" ]]; then
    add_pass "persona_hardgates" "N/A for ${PHASE}"
    return
  fi

  if [[ ! -d "$research_dir" ]]; then
    add_pass "persona_hardgates" "No persona research dir for ${CHANGE}"
    return
  fi

  local hitl_files critical_files
  hitl_files="$(grep -rl "hitl_required.*true\|\[HitL\]\|HITL required: yes" "$research_dir" 2>/dev/null || true)"
  critical_files="$(grep -rl "findings_severity.*critical\|severity: critical\|CRITICAL" "$research_dir" 2>/dev/null || true)"

  if [[ -n "$hitl_files" ]]; then
    add_error "persona_hardgates" "Persona reports indicate unresolved HITL"
  elif [[ -n "$critical_files" ]]; then
    add_error "persona_hardgates" "Persona reports contain unresolved critical findings"
  else
    add_pass "persona_hardgates" "No unresolved persona hard-gate blocks"
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
      gate_labels; gate_artifacts; gate_phase_state; gate_persona_hardgates; gate_slices ;;
    review)
      gate_labels; gate_artifacts; gate_phase_state; gate_verify_before_review; gate_review_security_artifacts; gate_review_truecourse_artifacts ;;
    archive)
      gate_labels; gate_artifacts; gate_phase_state; gate_verify_before_review; gate_review_before_archive; gate_review_security_artifacts; gate_review_truecourse_artifacts; gate_persona_routing; gate_persona_hardgates ;;
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
