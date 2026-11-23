import { join, relative, isAbsolute } from "node:path";
import { readBanner } from "./banner.ts";
import { readFileCached, type ChangeEntry } from "./resolve-change.ts";

/**
 * Strict-sequence guard (artifact order).
 *
 * Sources: `_shared/rules/sdd-structure.md` § Phase Order ("artifact creation
 * order is permanent and sequential: briefing → blueprint → blueprint
 * validation → cucumber test file → tasks → slices"; "a skill MUST NOT run
 * unless every preceding skill has completed") and the User gate contract
 * (the `user-gate-slice` gate must be cleared before execution writes code).
 *
 * Predecessor requirements, per target artifact (fast-track waivers relax
 * exactly what the recorded waiver skips — `_shared/planning/fast-track.md`):
 *
 *   briefing.md          none (chain head)
 *   blueprint.md         briefing.md approved — OR waiver covering the step
 *   acceptance.feature   blueprint.md approved/assumed/accepted — OR waiver
 *   tasks.md             acceptance.feature approved — OR waiver
 *   report.md            execution signal (a checked ticket, or the scratch
 *                        ledger work/<slug>/progress.md)
 *   review.md            report.md present
 *   code zone            an active change in execution (has tasks.md, or is
 *                        at a post-slicing phase) must have a cleared
 *                        user-gate-slice gate
 */

export interface OrderContext {
  root: string;
  specsRoot: string;
  scratchDir: string;
  /** Acted-on change dir (absolute) or null */
  changeDir: string | null;
  /** Acted-on change slug or null */
  slug: string | null;
  /** The acted-on change's state.yaml entry (phase/gate), or null when
   *  ambiguous / no entry */
  entry: ChangeEntry | null;
}

export interface OrderDecision {
  block?: boolean;
  reason?: string;
  context?: string;
}

/** Waiver record per `_shared/planning/fast-track.md` — machine-parseable. */
export interface Waiver {
  cls: "trivial" | "small";
  skips: string[];
  approvedBy: string;
}

/**
 * Parse the `## Fast-Track Waiver` block from a briefing. Valid = Class is
 * trivial/small, Skips is non-empty, Approved by is non-empty.
 */
export function parseWaiver(briefingContent: string | null): Waiver | null {
  if (!briefingContent) return null;
  const m = briefingContent.match(
    /##\s*Fast-Track Waiver[ \t]*\r?\n([\s\S]*?)(?=\r?\n##[ \t]|\r?\n#[ \t]|$)/,
  );
  if (!m) return null;
  const block = m[1];
  const cls = block.match(/-\s*Class:\s*(trivial|small)\b/);
  const skips = block.match(/-\s*Skips:\s*(.+)/);
  const approved = block.match(/-\s*Approved by:\s*(.+)/);
  if (!cls || !skips || !approved || !skips[1].trim() || !approved[1].trim()) {
    return null;
  }
  return {
    cls: cls[1] === "small" ? "small" : "trivial",
    skips: skips[1].split(",").map((s) => s.trim()).filter(Boolean),
    approvedBy: approved[1].trim(),
  };
}

const APPROVED_VALUES: Record<string, true> = {
  approved: true,
  assumed: true,
  accepted: true,
};

async function fileAt(dir: string, name: string): Promise<string | null> {
  return readFileCached(join(dir, name));
}

/**
 * Execution signal (sdd-structure § phase derivation): a checked ticket in
 * tasks.md, or the scratch ledger for the change.
 */
async function executionSignal(
  changeDir: string,
  scratchDir: string,
  slug: string,
): Promise<boolean> {
  const tasks = await fileAt(changeDir, "tasks.md");
  if (tasks !== null && /\[\s*[xX]\s*\]/.test(tasks)) return true;
  const progress = await readFileCached(join(scratchDir, slug, "progress.md"));
  return progress !== null;
}

/** Phases at which an active change must already have a tasks.md. */
const EXECUTION_PHASES: Record<string, true> = {
  slicing: true,
  execution: true,
  qa: true,
  review: true,
  ship: true,
};

export async function checkArtifactOrder(
  target: string,
  oc: OrderContext,
): Promise<OrderDecision | undefined> {
  const rel = relative(oc.specsRoot, target);
  const inSpecs = rel !== "" && !rel.startsWith("..") && !isAbsolute(rel);
  const parts = rel.split("/");

  if (inSpecs) {
    if (parts[0] === "change-archive") return {}; // immutability: fileset's message
    const changeDir = join(oc.specsRoot, parts[0]);
    const name = parts[parts.length - 1];

    const briefing = await fileAt(changeDir, "briefing.md");
    const waiver = parseWaiver(briefing);
    // The waiver lives in the briefing and its "Approved by" is the approval
    // record, so a valid waiver stands in for the briefing's own approval.
    const briefingOk =
      readBanner(briefing ?? "") === "approved" || waiver !== null;

    const blueprint = await fileAt(changeDir, "blueprint.md");
    const blueprintBanner = readBanner(blueprint ?? "");
    const blueprintOk =
      (blueprintBanner !== null && APPROVED_VALUES[blueprintBanner] === true) ||
      (waiver !== null && waiver.skips.includes("blueprint"));

    const acceptance = await fileAt(changeDir, "acceptance.feature");
    const acceptanceBanner = readBanner(acceptance ?? "");
    const acceptanceOk =
      acceptanceBanner === "approved" ||
      (waiver !== null && waiver.skips.includes("acceptance.feature"));

    switch (name) {
      case "briefing.md":
        return {}; // chain head

      case "blueprint.md":
        if (!briefingOk) {
          return {
            block: true,
            reason:
              `blueprint.md requires ${parts[0]}/briefing.md with STATUS ` +
              "`approved` (the user's hand-edit) — or a valid Fast-Track " +
              "Waiver block in the briefing. The briefing is not approved " +
              "and no waiver is recorded.",
          };
        }
        return {};

      case "acceptance.feature":
        if (!blueprintOk) {
          return {
            block: true,
            reason:
              `acceptance.feature requires ${parts[0]}/blueprint.md with STATUS ` +
              "`approved` / `assumed` / `accepted` — or a Fast-Track Waiver " +
              "that skips blueprint. Neither is present.",
          };
        }
        return {};

      case "tasks.md":
        if (!acceptanceOk) {
          return {
            block: true,
            reason:
              `tasks.md requires ${parts[0]}/acceptance.feature with STATUS ` +
              "`approved` — or a Fast-Track Waiver that skips " +
              "acceptance.feature (trivial). Neither is present.",
          };
        }
        return {};

      case "report.md": {
        const signal = await executionSignal(
          changeDir,
          oc.scratchDir,
          parts[0],
        );
        if (!signal) {
          return {
            block: true,
            reason:
              `report.md (the QA half) requires an execution signal first: a ` +
              `checked ticket ([x]) in ${parts[0]}/tasks.md, or the scratch ` +
              `ledger ${oc.scratchDir.split("/").pop()}/${parts[0]}/progress.md. ` +
              "Execute must have run before QA writes the report.",
          };
        }
        return {};
      }

      case "review.md": {
        const report = await fileAt(changeDir, "report.md");
        if (report === null) {
          return {
            block: true,
            reason:
              `review.md requires ${parts[0]}/report.md (the QA half) to exist — ` +
              "the test phase produces the report before the review record.",
          };
        }
        return {};
      }

      default:
        // adr.md, findings.md, ledger.md, evaluations/, modules/, sketches/,
        // shared/, *.council.md — no ordering predecessor.
        return {};
    }
  }

  // Code-zone write (anything outside the spec zone). The gate fires only when
  // the acted-on change is in execution: it has a tasks.md, or state.yaml
  // already says a post-slicing phase without one. A change still in planning
  // (spec/blueprint/intake) is not attributed code writes — parallel bounded
  // work stays unblocked; out-of-order code in a full change is caught later
  // by the execution-signal check on report.md and by ship-drift.
  if (!oc.changeDir || !oc.slug) return {};
  const tasks = await fileAt(oc.changeDir, "tasks.md");
  if (tasks === null) {
    if (oc.entry && EXECUTION_PHASES[oc.entry.phase] === true) {
      return {
        block: true,
        reason:
          `code write while change ${oc.slug} is at phase \`${oc.entry.phase}\` ` +
          "with no tasks.md — slicing must produce tasks.md before execution " +
          "writes code.",
      };
    }
    return {};
  }
  if (oc.entry && oc.entry.gate === "user-gate-slice") {
    return {
      block: true,
      reason:
        `user-gate-slice is pending on ${oc.slug}: the user must confirm the ` +
        "slice (say Implement) before execution writes code. Clear " +
        `changes.${oc.slug}.gate in state.yaml after confirmation.`,
    };
  }
  return {};
}
