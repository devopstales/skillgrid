import { basename } from "node:path";

/**
 * STATUS banner ownership guard.
 *
 * Source of the contract: `_shared/rules/sdd-structure.md` § STATUS Banner.
 * `briefing.md`, `blueprint.md`, `acceptance.feature`, and `tasks.md` carry a
 * machine-readable status banner: `> **STATUS:** \`<value>\` (YYYY-MM-DD)` on
 * line 3 (a blockquote after the `#` title; a blank line may sit between).
 *
 * Ownership rule: the agent may write every value EXCEPT `approved` — the
 * user's hand-edit is the approval record (the user edits the file in their
 * own editor, which the tool hooks do not see, so user flips are allowed by
 * construction). An agent edit after approval reverts the banner to `revised`;
 * the user must then re-read and re-flip.
 */

export const BANNER_RE = /^>\s*\*\*STATUS:\*\*\s*`?([a-z][a-z0-9-]*)`?/;

/** Banner files and the initial value the agent writes at creation. */
export const BANNER_INITIAL: Record<string, string> = {
  "briefing.md": "draft",
  "blueprint.md": "proposed",
  "acceptance.feature": "draft",
  "tasks.md": "sliced",
};

/** Allowed banner values (sdd-structure § STATUS Banner, values row). */
export const BANNER_VALUES: Record<string, true> = {
  proposed: true,
  draft: true,
  approved: true,
  sliced: true,
  "in-progress": true,
  complete: true,
  shipped: true,
  assumed: true,
  accepted: true,
  stalled: true,
  blocked: true,
  revised: true,
  superseded: true,
};

export interface BannerDecision {
  block?: boolean;
  reason?: string;
  context?: string;
}

/**
 * First banner match within the first 10 lines. The contract says line 3; the
 * scan tolerates a blank line after the title (the layout the repo's own
 * briefings use) without trusting a banner buried in the body.
 */
export function readBanner(content: string): string | null {
  const lines = content.split("\n");
  for (let i = 0; i < Math.min(lines.length, 10); i++) {
    const m = lines[i].match(BANNER_RE);
    if (m) return m[1];
  }
  return null;
}

/**
 * Check an agent write/edit of a banner file.
 *
 * @param fileName     basename of the target
 * @param oldContent   content before the edit; null = file does not exist yet
 * @param newContent   content after the edit (simulated for `edit`)
 */
export function checkBannerEdit(
  fileName: string,
  oldContent: string | null,
  newContent: string,
): BannerDecision {
  const initial = BANNER_INITIAL[basename(fileName)];
  if (initial === undefined) return {}; // not a banner file

  if (oldContent === newContent) return {}; // no-op edit

  const oldBanner = oldContent === null ? null : readBanner(oldContent);
  const newBanner = readBanner(newContent);

  if (newBanner === null) {
    return {
      block: true,
      reason:
        `${basename(fileName)} must carry the STATUS banner ` +
        '("> **STATUS:** `<value>` (YYYY-MM-DD)" near the top). ' +
        "The agent edit removed it — restore it.",
    };
  }
  if (!BANNER_VALUES[newBanner]) {
    return {
      block: true,
      reason:
        `${basename(fileName)} STATUS banner value "${newBanner}" is not a valid value ` +
        `(allowed: ${Object.keys(BANNER_VALUES).join(" / ")}).`,
    };
  }

  // Fresh file: the initial value only.
  if (oldContent === null) {
    if (newBanner !== initial) {
      return {
        block: true,
        reason:
          `new ${basename(fileName)} must start with the STATUS banner \`${initial}\` ` +
          `(the initial value the agent writes); found \`${newBanner}\`. ` +
          "`approved` is a user-only value — the user flips it after reading.",
      };
    }
    return {};
  }

  // `approved` is user-only: the user's hand-edit is the approval record.
  if (newBanner === "approved" && oldBanner !== "approved") {
    return {
      block: true,
      reason:
        `STATUS \`approved\` is a user-only value — the agent never writes it. ` +
        `Ask the user to read ${basename(fileName)} and hand-edit the banner to ` +
        "\`approved (YYYY-MM-DD)\` themselves; that edit is the approval record.",
    };
  }

  // An agent edit after approval reverts the banner to `revised`.
  if (oldBanner === "approved" && newBanner !== "revised") {
    return {
      block: true,
      reason:
        `${basename(fileName)} was \`approved\`; an agent edit must revert the ` +
        "banner to `revised` (the **STATUS:** line) so the user re-reads and " +
        `re-flips. Found \`${newBanner}\` after the edit.`,
    };
  }

  return {};
}
