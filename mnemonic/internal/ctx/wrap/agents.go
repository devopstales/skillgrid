// Package wrap implements `ctx wrap <agent>` (T07, plan 16.8/16.10): the
// tier-2 launcher that starts the loopback proxy, injects the agent's
// config to point at the proxy, backs up the previous config before
// writing, and launches the agent as a subprocess. The agent's code is
// never modified — only its config (env). Re-running is idempotent.
//
// The launcher is the only component that knows about specific agents
// (config.Wrap.Agents lists what is enabled; the agent table in agents.go
// maps names to providers and env vars).
package wrap

import "fmt"

// provider is the wire protocol family an agent speaks to the proxy. The
// proxy itself (T04) speaks both; the provider only selects the base-URL
// env var the launched subprocess gets (plan 16.8 Tier 1 contract).
type provider string

const (
	providerAnthropic provider = "anthropic"
	providerOpenAI    provider = "openai"
)

// Agent describes one wrappable agent: its canonical name, the provider it
// speaks, and the base-URL env var that must point at the proxy.
type Agent struct {
	Name     string
	Provider provider
	EnvVar   string
}

// agentTable is the launcher's agent → config mapping (T07: keep it a small
// table; the exact per-agent detail can grow here later). Claude is the
// Anthropic-compatible agent; codex/cursor/opencode/aider are OpenAI-
// compatible.
var agentTable = []Agent{
	{Name: "claude", Provider: providerAnthropic, EnvVar: "ANTHROPIC_BASE_URL"},
	{Name: "codex", Provider: providerOpenAI, EnvVar: "OPENAI_BASE_URL"},
	{Name: "cursor", Provider: providerOpenAI, EnvVar: "OPENAI_BASE_URL"},
	{Name: "opencode", Provider: providerOpenAI, EnvVar: "OPENAI_BASE_URL"},
	{Name: "aider", Provider: providerOpenAI, EnvVar: "OPENAI_BASE_URL"},
}

// core4 is the always-available set (no config needed): it matches the
// default of config.Wrap.Agents.
var core4 = map[string]bool{
	"claude":   true,
	"codex":    true,
	"cursor":   true,
	"opencode": true,
}

// lookupAgent returns the table entry for name, or nil when the name is not
// a known agent at all (distinct from "known but not enabled").
func lookupAgent(name string) *Agent {
	for i := range agentTable {
		if agentTable[i].Name == name {
			return &agentTable[i]
		}
	}
	return nil
}

// resolveAgent returns the agent to wrap and whether it is enabled.
// Core-4 agents are always enabled; any other known agent (opt-in: aider,
// cline, ...) is enabled only when it appears in enabledAgents (the
// config.Wrap.Agents list). An unknown name is an error with a clear,
// actionable message — the launcher never guesses.
func resolveAgent(name string, enabledAgents []string) (*Agent, error) {
	a := lookupAgent(name)
	if a == nil {
		return nil, fmt.Errorf("unknown agent %q (add it to wrap.agents to enable it)", name)
	}
	if core4[name] {
		return a, nil
	}
	for _, e := range enabledAgents {
		if e == name {
			return a, nil
		}
	}
	return nil, fmt.Errorf("agent %q is not enabled (add it to wrap.agents to enable it)", name)
}
