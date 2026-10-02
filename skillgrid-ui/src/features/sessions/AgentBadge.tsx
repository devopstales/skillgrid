const AGENT_STYLE: Record<string, string> = {
  cursor: 'border-info/40 bg-info/10 text-info',
  opencode: 'border-ok/40 bg-ok/10 text-ok',
  kilo: 'border-violet/40 bg-violet/10 text-violet',
  'claude-code': 'border-warn/40 bg-warn/10 text-warn',
}

// AgentBadge — the harness that owns a session (cursor, opencode, kilo, …).
// Sessions minted by mem_session_start have no harness and render "mnemonic".
export function AgentBadge({ agent, testId }: { agent?: string | null; testId?: string }) {
  const name = agent && agent.trim() !== '' ? agent : 'mnemonic'
  const cls = AGENT_STYLE[name] ?? 'border-edge bg-edge/40 text-ink-4'
  return (
    <span
      data-testid={testId}
      className={`inline-flex shrink-0 items-center rounded border px-1.5 py-px font-mono text-[10px] ${cls}`}
    >
      {name}
    </span>
  )
}

// McpBadge marks a tool call that went to an MCP server, not a harness built-in.
export function McpBadge() {
  return (
    <span className="inline-flex shrink-0 items-center rounded border border-accent/40 bg-accent/10 px-1 py-px font-mono text-[10px] text-accent-light">
      MCP
    </span>
  )
}
