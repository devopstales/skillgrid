import type { AgentId } from "./types.js";

export function agentIsSelected(selected: AgentId[], id: AgentId): boolean {
  return selected.includes(id);
}
