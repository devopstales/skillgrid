/**
 * Spec-zone file-set guard.
 *
 * Source: `_shared/rules/sdd-structure.md` § Directory Structure + the
 * spec-zone-only HARD-GATEs in the brainstorm and blueprint skills. Canonical
 * artifact names are load-bearing — `resume` and the phase-derivation table
 * key off the exact filenames, so an invented name (`plan.md`, `design.md`)
 * silently breaks the pipeline.
 */

/** Canonical artifact files directly inside a change dir. */
export const CANONICAL_FILES: Record<string, true> = {
  "briefing.md": true,
  "blueprint.md": true,
  "acceptance.feature": true,
  "tasks.md": true,
  "adr.md": true,
  "findings.md": true,
  "ledger.md": true,
  "report.md": true,
  "review.md": true,
};

/**
 * Canonical subdirectories of a change dir: multi-phase `shared/`
 * (sdd-structure § Multi-Phase Change Folders), config-driven `modules/`
 * (prototype.module_dir) and `sketches/` (sketch.dir), panel `evaluations/`.
 */
export const CANONICAL_DIRS: Record<string, true> = {
  shared: true,
  modules: true,
  sketches: true,
  evaluations: true,
};

const COUNCIL_RE = /\.council\.md$/;

export interface FileSetDecision {
  block?: boolean;
  reason?: string;
  context?: string;
}

/**
 * Check a path that lives inside the spec zone.
 *
 * @param relPath   path relative to the spec zone root, "/"-separated,
 *                  e.g. "2026-11-23-topic/plan.md"
 * @param isNewFile true when the write creates the file
 */
export function checkFileSet(relPath: string, isNewFile: boolean): FileSetDecision {
  const parts = relPath.split("/");

  // change-archive/ is the closed sink: ship moves the folder, and nothing new
  // may be added there (reflect appends to report.md's retro half in place).
  if (parts[0] === "change-archive") {
    const isReport = parts.length === 3 && parts[2] === "report.md";
    if (isNewFile && !isReport) {
      return {
        block: true,
        reason:
          `change-archive/ is immutable after the ship move — ${relPath} is a new ` +
          "file. A defect found after archiving is a new change, not an edit.",
      };
    }
    return {};
  }

  const slug = parts[0];
  if (!/^\d{4}-\d{2}-\d{2}-/.test(slug)) {
    if (isNewFile) {
      return {
        block: true,
        reason:
          `spec-zone files live inside a change dir (YYYY-MM-DD-<topic>/); ${relPath} ` +
          "is directly under the spec zone.",
      };
    }
    return {};
  }

  if (parts.length === 1) return {}; // a bare slug — not a file target

  const inCanonicalFile =
    CANONICAL_FILES[parts[1]] === true || COUNCIL_RE.test(parts[1]);
  const inCanonicalDir =
    parts.length >= 3 && CANONICAL_DIRS[parts[1]] === true;

  if (inCanonicalFile || inCanonicalDir) return {};

  if (isNewFile) {
    return {
      block: true,
      reason:
        `"${parts[1]}" is not a canonical artifact name for ${slug}/ ` +
        `(allowed: ${Object.keys(CANONICAL_FILES).join(", ")} + *.council.md + ` +
        `${Object.keys(CANONICAL_DIRS).map((d) => d + "/").join(" ")}). ` +
        "Canonical names are load-bearing — resume derives the phase from the " +
        "exact filenames.",
    };
  }
  return {
    context:
      `note: ${slug}/${parts[1]} is not a canonical artifact name (created before ` +
      "the guard was active?). sdd-structure.md § Directory Structure names " +
      "the allowed set.",
  };
}
