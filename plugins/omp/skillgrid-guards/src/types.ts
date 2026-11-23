/**
 * Minimal structural types for the omp hook surface these guards use.
 *
 * Per omp docs (https://omp.sh/docs/hooks): discovered hook factories register
 * `pi.on("tool_call", handler)`; the pre-execution event carries `toolName` and
 * a normalized `input`; a handler may return `{ block, reason, input,
 * additionalContext }`. Handlers that throw fail the call closed, so the guard
 * entrypoint catches everything (fail-open on internal error).
 *
 * We deliberately do not import `@oh-my-pi/pi-coding-agent`: the guards are
 * plain functions over (cwd, toolName, input) so they stay unit-testable
 * without the harness.
 */

export interface ToolCallEvent {
  toolName: string;
  input: Record<string, unknown>;
}

/**
 * Guard decision. `block` short-circuits the tool call (reason is delivered as
 * the error text); `context` is trusted handler-authored context delivered
 * after the tool result. `undefined` = pass, nothing to say.
 */
export interface Decision {
  block?: boolean;
  reason?: string;
  context?: string;
}

/**
 * Normalized input view for the two tools the guards cover. `paths` is the
 * derived policy-inspection field the harness may attach to `edit`.
 */
export interface ToolInput {
  path?: unknown;
  paths?: unknown;
  content?: unknown;
  old_string?: unknown;
  new_string?: unknown;
  replace_all?: unknown;
}
