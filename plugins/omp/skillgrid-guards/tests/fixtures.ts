import { mkdtemp, mkdir, writeFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { clearFileCache } from "../src/resolve-change.ts";

/**
 * Temp skillgrid project fixture: `.skillgrid/{state.yaml,config.yaml,
 * change-specs/<slug>/}`. Tests clear the file cache per fixture (the guard's
 * mtime cache would otherwise share state across fixtures on one machine).
 */

export interface Fixture {
  root: string;
  specs: string;
  change: string;
  slug: string;
  cleanup: () => Promise<void>;
}

export interface FixtureOpts {
  slug?: string;
  /** Full state.yaml body. Omit for a single in-progress execution change. */
  state?: string;
  /** Full config.yaml body. Omit for default conventions. */
  config?: string;
  /** Create the change dir (default true) */
  changeDir?: boolean;
}

export const SLUG = "2026-11-23-topic";

export async function makeProject(opts: FixtureOpts = {}): Promise<Fixture> {
  const slug = opts.slug ?? SLUG;
  const root = await mkdtemp(join(tmpdir(), "sg-guard-"));
  await mkdir(join(root, ".skillgrid"), { recursive: true });
  if (opts.changeDir !== false) {
    await mkdir(join(root, ".skillgrid", "change-specs", slug), { recursive: true });
  }
  const state =
    opts.state ??
    `schema: skillgrid/state/v2
changes:
  ${slug}:
    phase: "execution"
    status: in_progress
    gate: ""
progress:
  completed_changes: 0
`;
  await writeFile(join(root, ".skillgrid", "state.yaml"), state);
  await writeFile(
    join(root, ".skillgrid", "config.yaml"),
    opts.config ??
      `conventions:
  specs_root: .skillgrid/change-specs
  scratch_dir: .skillgrid/work
`,
  );
  clearFileCache();
  return {
    root,
    slug,
    specs: join(root, ".skillgrid", "change-specs"),
    change: join(root, ".skillgrid", "change-specs", slug),
    cleanup: () => rm(root, { recursive: true, force: true }),
  };
}

/** Banner line in the layout the repo's own briefings use (blank line 2). */
export function bannerDoc(title: string, value: string, date = "2026-11-23"): string {
  return `# ${title}\n\n> **STATUS:** \`${value}\` (${date})\n\nbody\n`;
}

/** A valid Fast-Track Waiver block (fast-track.md § Waiver record). */
export function waiverBlock(cls: "trivial" | "small", skips: string): string {
  return `## Fast-Track Waiver
- Class: ${cls}
- Skips: ${skips}
- Reason: test waiver
- Approved by: programmer
`;
}
