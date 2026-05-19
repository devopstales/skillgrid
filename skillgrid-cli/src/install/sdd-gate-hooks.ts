import { chmodSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { logInfo, logSuccess, logWarn } from "./log.js";

const PRE_COMMIT_HOOK = `#!/usr/bin/env bash
# Pre-commit hook: run sdd-gate.sh on SDD changes when openspec/ changes are staged.

set -uo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root" || exit 1

staged_files="$(git diff --cached --name-only 2>/dev/null || true)"
if [[ -z "$staged_files" ]]; then
  exit 0
fi

change_dirs=()
while IFS= read -r f; do
  [[ -z "$f" ]] && continue
  if [[ "$f" == openspec/changes/* ]] || [[ "$f" == .openspec/changes/* ]]; then
    dir="$(echo "$f" | cut -d'/' -f1-3)"
    change_dirs+=("$dir")
  fi
done <<< "$staged_files"

if [[ \${#change_dirs[@]} -eq 0 ]]; then
  exit 0
fi

unique_dirs=()
for d in "\${change_dirs[@]}"; do
  found=false
  for u in "\${unique_dirs[@]+"\${unique_dirs[@]}"}"; do
    [[ "$u" == "$d" ]] && found=true && break
  done
  "$found" || unique_dirs+=("$d")
done

sdd_detect_phase() {
  local change_name="$1"
  local prefix="openspec/changes/\${change_name}/"
  local staged
  staged="$(git diff --cached --name-only -- "\${prefix}" 2>/dev/null || true)"
  if echo "$staged" | grep -qE 'reviews/.*\\.md$'; then
    echo "review"
  elif echo "$staged" | grep -qE 'verify-report\\.md$'; then
    echo "verify"
  elif echo "$staged" | grep -qE 'tasks\\.md$'; then
    echo "apply"
  elif echo "$staged" | grep -qE 'design\\.md$'; then
    echo "design"
  elif echo "$staged" | grep -qE 'specs/.*/spec\\.md$'; then
    echo "spec"
  elif echo "$staged" | grep -qE 'proposal\\.md$'; then
    echo "propose"
  elif echo "$staged" | grep -qE 'ui-(wireframes|decisions)\\.md$'; then
    echo "design"
  else
    echo "tasks"
  fi
}

errors=0
for d in "\${unique_dirs[@]}"; do
  dir_name="$(basename "$d")"
  [[ "$dir_name" == "archive" ]] && continue
  phase="$(sdd_detect_phase "$dir_name")"
  echo "[sdd-gate] Running gate for: \${d} (phase=\${phase})" >&2

  if ! "\${repo_root}/.skillgrid/scripts/sdd-gate.sh" "$phase" --change "$dir_name" 2>&1; then
    echo "" >&2
    echo "=== sdd-gate PRE-COMMIT BLOCKED ===" >&2
    echo "Phase: $phase | Change: $dir_name" >&2
    echo "Fix gate failures before committing. Run manually:" >&2
    echo "  .skillgrid/scripts/sdd-gate.sh $phase --change $dir_name" >&2
    errors=1
  fi
done

exit $errors
`;

const PRE_PUSH_HOOK = `#!/usr/bin/env bash
# Pre-push hook: verify only SDD changes included in this push.

set -uo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root" || exit 1
errors=0

change_names=()
sdd_collect_change_from_path() {
  local f="$1"
  local name=""
  if [[ "$f" == openspec/changes/*/* ]]; then
    name="$(echo "$f" | cut -d'/' -f3)"
  elif [[ "$f" == .openspec/changes/*/* ]]; then
    name="$(echo "$f" | cut -d'/' -f3)"
  else
    return 0
  fi
  [[ -z "$name" || "$name" == "archive" ]] && return 0
  local found=false
  for c in "\${change_names[@]+"\${change_names[@]}"}"; do
    [[ "$c" == "$name" ]] && found=true && break
  done
  "$found" || change_names+=("$name")
}

while read -r local_ref local_sha remote_ref remote_sha; do
  [[ -z "$local_sha" ]] && continue
  if [[ "$local_sha" == "0000000000000000000000000000000000000000" ]]; then
    continue
  fi
  if [[ "$remote_sha" == "0000000000000000000000000000000000000000" ]]; then
    range="$local_sha"
  else
    range="\${remote_sha}..\${local_sha}"
  fi
  while IFS= read -r f; do
    [[ -z "$f" ]] && continue
    sdd_collect_change_from_path "$f"
  done < <(git diff --name-only "$range" 2>/dev/null || true)
done

if [[ \${#change_names[@]} -eq 0 ]]; then
  exit 0
fi

for change_name in "\${change_names[@]}"; do
  change_dir="$repo_root/openspec/changes/$change_name"
  [[ -f "\${change_dir}/tasks.md" ]] || continue

  echo "[sdd-gate-pre-push] Checking: $change_name" >&2

  if ! "\${repo_root}/.skillgrid/scripts/sdd-gate.sh" verify --change "$change_name" 2>&1; then
    echo "" >&2
    echo "=== sdd-gate PRE-PUSH BLOCKED ===" >&2
    echo "Change: $change_name" >&2
    echo "Resolve gate failures before pushing. Run manually:" >&2
    echo "  .skillgrid/scripts/sdd-gate.sh verify --change $change_name" >&2
    errors=1
  fi
done

exit $errors
`;

function gitDir(projectPath: string): string | null {
  const r = spawnSync("git", ["-C", projectPath, "rev-parse", "--git-dir"], { encoding: "utf8" });
  if (r.status !== 0 || !r.stdout?.trim()) return null;
  const gitDirRel = r.stdout.trim();
  return gitDirRel.startsWith("/") ? gitDirRel : join(projectPath, gitDirRel);
}

export function hookContainsSddGate(filePath: string): boolean {
  if (!existsSync(filePath)) return false;
  try {
    return readFileSync(filePath, "utf8").includes("sdd-gate");
  } catch {
    return false;
  }
}

export function isGitRepository(projectPath: string): boolean {
  return gitDir(projectPath) !== null;
}

export function installSddGateHooks(projectPath: string, dryRun: boolean): boolean {
  const gateScript = join(projectPath, ".skillgrid", "scripts", "sdd-gate.sh");
  if (!existsSync(gateScript)) {
    logWarn(`SDD gate hooks: skipped — missing ${gateScript}`);
    return false;
  }

  const hooksDir = gitDir(projectPath);
  if (!hooksDir) {
    logInfo("SDD gate hooks: skipped (not a git repository)");
    return false;
  }

  const hooksPath = join(hooksDir, "hooks");
  const preCommit = join(hooksPath, "pre-commit");
  const prePush = join(hooksPath, "pre-push");

  if (dryRun) {
    logInfo(`[DRY-RUN] Would install SDD gate hooks in ${hooksPath}`);
    return true;
  }

  if (!dryRun) chmodSync(gateScript, 0o755);

  mkdirSync(hooksPath, { recursive: true });
  writeFileSync(preCommit, `${PRE_COMMIT_HOOK}\n`, { mode: 0o755 });
  writeFileSync(prePush, `${PRE_PUSH_HOOK}\n`, { mode: 0o755 });
  chmodSync(preCommit, 0o755);
  chmodSync(prePush, 0o755);

  logSuccess(`Installed SDD gate hooks: ${preCommit}, ${prePush}`);
  return true;
}

export function uninstallSddGateHooks(projectPath: string, dryRun: boolean): void {
  const hooksDir = gitDir(projectPath);
  if (!hooksDir) {
    logInfo("SDD gate hooks: skipped uninstall (not a git repository)");
    return;
  }

  const hooksPath = join(hooksDir, "hooks");
  const preCommit = join(hooksPath, "pre-commit");
  const prePush = join(hooksPath, "pre-push");

  if (dryRun) {
    logInfo(`[DRY-RUN] Would remove sdd-gate hooks from ${hooksPath} (if present)`);
    return;
  }

  if (hookContainsSddGate(preCommit)) {
    rmSync(preCommit, { force: true });
    logSuccess(`Removed ${preCommit}`);
  }
  if (hookContainsSddGate(prePush)) {
    rmSync(prePush, { force: true });
    logSuccess(`Removed ${prePush}`);
  }
}

export function showSddGateHookStatus(projectPath: string): void {
  const gateScript = join(projectPath, ".skillgrid", "scripts", "sdd-gate.sh");
  const hooksDir = gitDir(projectPath);
  console.log(`SDD gate hook status (repo: ${projectPath}):`);
  console.log(`  Script: ${existsSync(gateScript) ? "exists" : "MISSING"}`);
  if (!hooksDir) {
    console.log("  (not a git repository)");
    return;
  }
  const preCommit = join(hooksDir, "hooks", "pre-commit");
  const prePush = join(hooksDir, "hooks", "pre-push");
  for (const [label, path] of [
    ["pre-commit", preCommit],
    ["pre-push", prePush],
  ] as const) {
    if (!existsSync(path)) {
      console.log(`  ${label}: not installed`);
    } else if (hookContainsSddGate(path)) {
      console.log(`  ${label}: installed (sdd-gate active)`);
    } else {
      console.log(`  ${label}: installed (not sdd-gate — left unchanged)`);
    }
  }
}

