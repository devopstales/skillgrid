import { stat } from "node:fs/promises";
import type { ExtensionAPI, ExtensionContext } from "@earendil-works/pi-coding-agent";
import { SessionManager } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";
import {
  runDirectedWt,
  validateDirective,
  recordWorktrees,
  clearWorktreeRecord,
  recover,
  supportsDirectiveProtocol,
  applyMergeDefaults,
  checkBranchAllowList,
  observeWorktreeRoot,
  isWorktreeContext,
  approvalCommands,
  isApprovalNeeded,
  handleWtApproval,
  probeUnapprovedHooks,
} from "./core/wt-core.ts";
import { markWaiting, markWorking } from "./markers.ts";

// ---------------------------------------------------------------------------
// Worktrunk extension (extension 2): the `worktrunk` tool, `/wt` command,
// no-replay recovery, project-command approval passthrough, and the
// agent-side marker lifecycle (agent_start → 🤖, agent_end → 💬; markers.ts
// is the sole marker writer). Core (src/core/) stays Pi-free.
// ---------------------------------------------------------------------------

const WORKTRUNK_DESCRIPTION = [
  "Run a Worktrunk (wt) command. Before using a command's arguments or",
  "options, look up its reference by adding --help to its args:",
  '{"command":"switch","args":["--help"]}. Help calls return immediately.',
  "Pi follows wt's directory-change directive: a session move is reported",
  "as {\"moved\":true,\"to\":<path>}; in print/json mode a move reports",
  "{\"stopped\":true,\"restart\":<path>} (restart Pi in that directory).",
  "If the working directory is gone, Pi recovers to a verified surviving",
  "worktree and reports the recovery without running the requested",
  "command. Project commands that need approval are offered in a dialog;",
  "approving saves exactly those commands and retries once.",
].join("\n");

/** True when an arg would skip wt's approval prompts (the user's decision). */
function bypassesApproval(args: readonly string[]): boolean {
  for (const arg of args) {
    if (arg === "--") break;
    if (arg === "--yes" || arg.startsWith("--yes=") || /^-[A-Za-z]*y[A-Za-z]*$/.test(arg)) return true;
  }
  return false;
}

// approvalCommands / isApprovalNeeded live in the core (src/core/wt-core.ts)
// so the approval flow is testable without Pi (R12).

/**
 * The linked-session mechanism (R1 / pi-worktrunk): fork the current
 * session's history into a new session whose cwd is `targetCwd`, then
 * `ctx.switchSession` into it. The source session stays `/resume`-able.
 */
function forkSessionTo(
  ctx: ExtensionContext,
  targetCwd: string,
): { destinationSession: string } {
  const sourceSession = ctx.sessionManager.getSessionFile();
  if (!sourceSession) throw new Error("Cannot move an ephemeral Pi session.");
  const full = ctx.sessionManager as typeof ctx.sessionManager & {
    usesDefaultSessionDir?: () => boolean;
  };
  const targetSessionDir = full.usesDefaultSessionDir?.() ? undefined : ctx.sessionManager.getSessionDir();
  const manager = SessionManager.forkFrom(sourceSession, targetCwd, targetSessionDir);
  const destinationSession = manager.getSessionFile();
  if (!destinationSession) throw new Error("Pi did not create a persisted session copy.");
  return { destinationSession };
}

export default function (pi: ExtensionAPI) {
  let directiveSupport: boolean | undefined;
  // R7: the observed-root note, cached on first use ("" = standard root).
  let observedRootNote: string | undefined;

  async function wtSupportsDirectives(): Promise<boolean> {
    if (directiveSupport === undefined) directiveSupport = await supportsDirectiveProtocol();
    return directiveSupport;
  }

  async function runWt(
    args: string[],
    cwd: string,
    signal?: AbortSignal,
  ): Promise<{ code: number; stdout: string; stderr: string; directive?: string }> {
    const res = await runDirectedWt(args, { cwd, signal });
    return { code: res.code, stdout: res.stdout, stderr: res.stderr, directive: res.directive };
  }

  /**
 * Approve the unapproved project commands (dialog → PTY store → retry once).
 * The ask, the non-interactive PTY store, and the retry are delegated to the
 * core (`handleWtApproval`), which is testable without Pi (R12); the
 * extension only supplies `ctx.ui.confirm` as the approval ask.
 */
  async function requestApproval(
    ctx: ExtensionContext,
    args: string[],
    output: string,
    cwd: string,
  ): Promise<{ approved: boolean; message: string; code?: number; savedCommands: string[] }> {
    const listing = approvalCommands(output).map((c) => `  ${c}`).join("\n");
    const outcome = await handleWtApproval(cwd, args, output, () =>
      ctx.ui.confirm(
        "Approve Worktrunk project commands?",
        `\`wt ${args.join(" ")}\` needs approval to run these project commands. ` +
          `Approving saves them to Worktrunk's approvals for this project.\n\n${listing}`,
      ),
    );
    return {
      approved: outcome.approved,
      message: outcome.message,
      code: outcome.code,
      savedCommands: outcome.savedCommands,
    };
  }

  async function runWorktrunk(
    command: string,
    args: string[],
    ctx: ExtensionContext,
  ): Promise<{
    content: { type: "text"; text: string }[];
    details: {
      code?: number;
      stderr?: string;
      moved?: boolean;
      to?: string;
      stopped?: boolean;
      restart?: string;
      rejected?: string;
      recovered?: boolean;
    };
  }> {
    // R9 merge invariance: `wt merge` always runs --no-squash --no-commit
    // unless the caller explicitly requested squash (applyMergeDefaults is a
    // no-op for every other command and never mutates its input).
    const allArgs = applyMergeDefaults([command, ...args]);

    // R8 allow-list invariance: `switch --create <branch>` in a worktree
    // context must name an allowed branch; the primary checkout is
    // unconstrained. Rejected before anything runs; the suggestion names the
    // nearest allowed form.
    if (command === "switch") {
      const ci = allArgs.indexOf("--create");
      const branch = ci >= 0 ? allArgs[ci + 1] : undefined;
      if (branch && (await isWorktreeContext(ctx.cwd))) {
        const check = checkBranchAllowList(branch, true);
        if (!check.ok) {
          return {
            content: [
              {
                type: "text",
                text: `Branch ${branch} is not allowed in a worktree context. ` +
                  `Nearest allowed form: ${check.suggestion}.`,
              },
            ],
            details: { code: 0, rejected: branch, to: check.suggestion },
          };
        }
      }
    }

    // Reject model-originated bypass flags before anything runs.
    if (bypassesApproval(allArgs)) {
      throw new Error(
        "The worktrunk tool does not accept -y/--yes: it would skip Worktrunk's approval of " +
          "project commands. Run the command without it; Pi asks the user when approval is needed.",
      );
    }

    // No-replay recovery: when the session's cwd is gone, move to a
    // surviving worktree of the same repository. The pending command is
    // never run (re-evaluate it in the new worktree before retrying).
    try {
      await stat(ctx.cwd);
    } catch {
      const rec = await recover(ctx.cwd, allArgs);
      if (rec.movedTo && ctx.sessionManager.getSessionFile() && (ctx.mode === "tui" || ctx.mode === "rpc")) {
        const { destinationSession } = forkSessionTo(ctx, rec.movedTo);
        const result = await ctx.switchSession(destinationSession, {
          withSession: async (nextCtx) => {
            nextCtx.ui.notify(
              `Session recovered: cwd ${ctx.cwd} is gone; moved to ${rec.movedTo}. ` +
                `The requested command (wt ${allArgs.join(" ")}) was NOT run — review it before retrying.`,
              "warning",
            );
          },
        });
        if (result.cancelled) {
          throw new Error(`Recovery session created at ${rec.movedTo}, but the switch was cancelled.`);
        }
        recordWorktrees(rec.movedTo);
        return {
          content: [
            {
              type: "text",
              text: `Recovered the session to ${rec.movedTo} (cwd ${ctx.cwd} no longer exists). ` +
                `wt ${allArgs.join(" ")} was NOT run. Re-evaluate it in the new worktree before retrying.`,
            },
          ],
          details: { recovered: true, to: rec.movedTo },
        };
      }
      if (rec.movedTo) {
        // print/json: cannot move — report the restart location, no replay.
        return {
          content: [
            {
              type: "text",
              text: `Session's cwd ${ctx.cwd} no longer exists. Restart Pi in ${rec.movedTo}; ` +
                `wt ${allArgs.join(" ")} was NOT run.`,
            },
          ],
          details: { recovered: true, stopped: true, restart: rec.movedTo },
        };
      }
      return {
        content: [
          {
            type: "text",
            text: `Session's cwd ${ctx.cwd} no longer exists and no verified surviving worktree ` +
              `was recorded. Restart Pi in ${ctx.cwd} (its original location); wt ${allArgs.join(" ")} was NOT run.`,
          },
        ],
        details: { recovered: true, restart: ctx.cwd },
      };
    }

    recordWorktrees(ctx.cwd);

    const res = await runWt(allArgs, ctx.cwd, ctx.signal);

    // Non-approval failure: passthrough.
    if (res.code !== 0 && !isApprovalNeeded(res.stdout + res.stderr)) {
      return {
        content: [{ type: "text", text: `wt ${allArgs.join(" ")} failed (exit ${res.code}).\n${[res.stderr.trim(), res.stdout.trim()].filter(Boolean).join("\n")}` }],
        details: { code: res.code, stderr: res.stderr },
      };
    }

    // A hook-triggering command silently skips an unapproved hook (exit 0, no
    // marker). Probe the triggered phases so the approval still surfaces
    // (probeUnapprovedHooks is a no-op for non-hook commands).
    let approvalOutput = res.stdout + res.stderr;
    if (res.code === 0 && !isApprovalNeeded(approvalOutput)) {
      const probe = await probeUnapprovedHooks(ctx.cwd, command);
      if (probe.needsApproval) approvalOutput = res.stdout + "\n" + probe.output;
    }

    // Approval: dialog in interactive modes; fail + list in non-interactive.
    if (isApprovalNeeded(approvalOutput)) {
      if (ctx.mode === "tui" || ctx.mode === "rpc") {
        const outcome = await requestApproval(ctx, allArgs, approvalOutput, ctx.cwd);
        return {
          content: [{ type: "text", text: outcome.message }],
          details: { code: res.code },
        };
      }
      const commands = approvalCommands(approvalOutput);
      const listing = commands.length ? `\n${commands.map((c) => `  ${c}`).join("\n")}` : "";
      return {
        content: [
          {
            type: "text",
            text: `wt ${allArgs.join(" ")} needs project-command approval (non-interactive mode; ` +
              `not run). Approve in a terminal with \`wt config approvals add\`:\n${listing}`,
          },
        ],
        details: { code: res.code },
      };
    }

    // Directive: follow the requested directory (R1 safety: validate first).
    const directive = await wtSupportsDirectives() ? res.directive : undefined;
    if (directive) {
      const validation = validateDirective(ctx.cwd, directive);
      if (!validation.ok) {
        return {
          content: [
            {
              type: "text",
              text: `wt requested a move to ${validation.rejected}, which is outside the original ` +
                `repository; the session stayed in ${ctx.cwd}.\n${[res.stderr.trim(), res.stdout.trim()].filter(Boolean).join("\n")}`,
            },
          ],
          details: { code: res.code, rejected: validation.rejected },
        };
      }
      if (ctx.mode === "tui" || ctx.mode === "rpc") {
        const { destinationSession } = forkSessionTo(ctx, directive);
        const result = await ctx.switchSession(destinationSession, {
          withSession: async (nextCtx) => {
            nextCtx.ui.notify(`Moved to ${directive}. The source session is available via /resume.`, "info");
          },
        });
        if (result.cancelled) {
          throw new Error(`Move session created at ${directive}, but the switch was cancelled.`);
        }
        recordWorktrees(directive);
        return {
          content: [{ type: "text", text: `Moved the session to ${directive}. The source session remains available via /resume.` }],
          details: { code: res.code, moved: true, to: directive },
        };
      }
      // print/json: a directory request stops continuation.
      return {
        content: [{ type: "text", text: `wt requested a move to ${directive}; Pi stopped. Restart Pi in ${directive}.` }],
        details: { code: res.code, stopped: true, restart: directive },
      };
    }

    // R7: observe the worktree root on first use; carry a one-line note on
    // the first report if worktrees landed outside the standard root (never
    // silently follow a foreign layout).
    let rootNote = "";
    if (observedRootNote === undefined) {
      try {
        const obs = await observeWorktreeRoot(ctx.cwd);
        rootNote = obs.note ?? "";
      } catch {
        rootNote = "";
      }
      observedRootNote = rootNote;
    }
    const base = [res.stdout.trim(), res.stderr.trim()].filter(Boolean).join("\n") || "(no output)";
    return {
      content: [{ type: "text", text: rootNote ? `${base}\n${rootNote}` : base }],
      details: { code: res.code },
    };
  }

  // --- tool -----------------------------------------------------------------

  pi.registerTool({
    name: "worktrunk",
    label: "Worktrunk",
    description: WORKTRUNK_DESCRIPTION,
    promptSnippet: "Run Worktrunk commands; add --help to args to look up a command's reference",
    promptGuidelines: [
      "Use worktrunk for Worktrunk (wt) commands. Look up a command's arguments and options by adding --help to its args before using them.",
      "Never run Worktrunk calls in parallel: issue one worktrunk call, wait for its result, then issue the next.",
      "Never pass -y/--yes to worktrunk; approving project commands is the user's decision, and Pi asks the user when Worktrunk needs approval.",
    ],
    parameters: Type.Object({
      command: Type.String({ description: "The Worktrunk command, e.g. \"switch\", \"remove\", \"merge\", \"list\", \"config\"." }),
      args: Type.Optional(Type.Array(Type.String(), { description: "Arguments after the Worktrunk command, in CLI order and without shell expansion." })),
    }, { additionalProperties: false }),
    executionMode: "sequential",
    async execute(_id, params, _signal, _update, ctx) {
      return runWorktrunk(params.command, params.args ?? [], ctx);
    },
  });

  // --- /wt command ------------------------------------------------------------

  pi.registerCommand("wt", {
    description: "Run a Worktrunk (wt) command (alias for the worktrunk tool)",
    handler: async (args, ctx) => {
      const tokens = args.trim().split(/\s+/).filter(Boolean);
      if (tokens.length === 0) {
        ctx.ui.notify("Usage: /wt <command> [args...]", "info");
        return;
      }
      const [command, ...rest] = tokens;
      const result = await runWorktrunk(command, rest, ctx);
      ctx.ui.notify(result.content[0].text, result.details.code !== 0 ? "error" : "info");
    },
  });

  // --- session lifecycle -------------------------------------------------------

  pi.on("session_start", (_event, ctx) => {
    recordWorktrees(ctx.cwd);
  });

  // Marker lifecycle, agent side (findings.md R1 marker table):
  // agent_start → 🤖 working; agent_end → 💬 waiting. The scratch extension
  // owns session_start (💬) and session_shutdown (clear); all writes go
  // through markers.ts.
  pi.on("agent_start", (_e, ctx) => {
    void markWorking(ctx.cwd).catch(() => {});
  });

  pi.on("agent_end", (_e, ctx) => {
    void markWaiting(ctx.cwd).catch(() => {});
  });

  pi.on("session_shutdown", () => {
    clearWorktreeRecord();
  });
}
