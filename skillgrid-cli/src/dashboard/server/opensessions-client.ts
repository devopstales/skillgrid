import path from "node:path";
import { resolveRepoRoot, sessionDirMatchesRepo } from "./fs-utils.js";

/** Mirrors opensessions `AgentStatus` (packages/runtime/src/contracts/agent.ts). */
export type OpenSessionsAgentStatus =
  | "idle"
  | "running"
  | "tool-running"
  | "done"
  | "error"
  | "waiting"
  | "interrupted"
  | "stale";

export type OpenSessionsAgentEvent = {
  agent: string;
  session: string;
  status: OpenSessionsAgentStatus;
  ts: number;
  threadId?: string;
  threadName?: string;
  unseen?: boolean;
};

export type OpenSessionsMetadataTone = "neutral" | "info" | "success" | "warn" | "error";

export type OpenSessionsSessionMetadata = {
  status: { text: string; tone?: OpenSessionsMetadataTone; ts: number } | null;
  progress: { current?: number; total?: number; percent?: number; label?: string; ts: number } | null;
  logs: Array<{ message: string; tone?: OpenSessionsMetadataTone; source?: string; ts: number }>;
};

export type OpenSessionsSessionData = {
  name: string;
  dir: string;
  branch: string;
  dirty: boolean;
  unseen: boolean;
  ports: number[];
  agentState: OpenSessionsAgentEvent | null;
  agents: OpenSessionsAgentEvent[];
  metadata?: OpenSessionsSessionMetadata | null;
};

export type OpenSessionsServerState = {
  type: "state";
  sessions: OpenSessionsSessionData[];
  focusedSession: string | null;
  currentSession: string | null;
  ts: number;
};

export type AgentSessionCard = {
  name: string;
  dir: string;
  branch: string;
  dirty: boolean;
  unseen: boolean;
  matchesRepo: boolean;
  primaryAgent: OpenSessionsAgentEvent | null;
  agents: OpenSessionsAgentEvent[];
  metadataStatus: string | null;
  metadataTone: OpenSessionsMetadataTone | null;
  metadataProgress: string | null;
  ports: number[];
};

export type OpenSessionsStatus = {
  healthy: boolean;
  url: string;
  wsUrl: string;
  /** Active tmux session (attached client), when reported by opensessions. */
  currentSession: string | null;
  /** Sidebar selection in opensessions UI — may differ from currentSession. */
  focusedSession: string | null;
  /** Human-readable active session (name · branch · cwd). */
  activeSessionSummary: string | null;
  /** Set only when sidebar focus differs from active tmux session. */
  sidebarFocusSummary: string | null;
  sessions: AgentSessionCard[];
  startCommand: string;
};

const DEFAULT_HOST = "127.0.0.1";
const DEFAULT_PORT = 7391;

export function resolveOpenSessionsEndpoints(options?: { host?: string; port?: number }): { httpUrl: string; wsUrl: string } {
  const host = options?.host ?? process.env.OPENSESSIONS_HOST?.trim() ?? DEFAULT_HOST;
  const portRaw = options?.port ?? Number.parseInt(process.env.OPENSESSIONS_PORT ?? "", 10);
  const port = Number.isFinite(portRaw) && portRaw > 0 ? portRaw : DEFAULT_PORT;
  return {
    httpUrl: `http://${host}:${port}`,
    wsUrl: `ws://${host}:${port}`
  };
}

export function formatMetadataProgress(
  progress: NonNullable<OpenSessionsSessionMetadata["progress"]>
): string | null {
  if (progress.current != null && progress.total != null) {
    const label = progress.label ? ` ${progress.label}` : "";
    return `${progress.current}/${progress.total}${label}`;
  }
  if (progress.percent != null) {
    const label = progress.label ? ` ${progress.label}` : "";
    return `${Math.round(progress.percent * 100)}%${label}`;
  }
  return progress.label ?? null;
}

/**
 * Resolve session name from opensessions payload (handles mistaken numeric indices).
 */
export function normalizeSessionRef(
  ref: unknown,
  sessions: OpenSessionsSessionData[]
): string | null {
  if (ref === null || ref === undefined) return null;
  if (typeof ref === "number" && Number.isInteger(ref)) {
    if (ref >= 0 && ref < sessions.length) return sessions[ref]!.name;
    return null;
  }
  const text = String(ref).trim();
  if (!text) return null;
  if (sessions.some((s) => s.name === text)) return text;
  if (/^\d+$/.test(text)) {
    const idx = Number(text);
    if (idx >= 0 && idx < sessions.length) return sessions[idx]!.name;
  }
  return text;
}

export function formatSessionSummary(
  sessionName: string | null,
  sessions: Pick<AgentSessionCard, "name" | "branch" | "dir">[]
): string | null {
  if (!sessionName) return null;
  const session = sessions.find((s) => s.name === sessionName);
  if (!session) return sessionName;
  const bits: string[] = [session.name];
  if (session.branch) bits.push(`branch ${session.branch}`);
  if (session.dir) {
    const home = process.env.HOME ?? "";
    const dir =
      home && session.dir.startsWith(home)
        ? `~${session.dir.slice(home.length)}`
        : session.dir;
    bits.push(dir.length > 52 ? `…${dir.slice(-51)}` : dir);
  }
  return bits.join(" · ");
}

export function mapOpenSessionsForRepo(state: OpenSessionsServerState, repoRoot: string): AgentSessionCard[] {
  const normalizedRepo = resolveRepoRoot(repoRoot);
  return state.sessions
    .map((session) => toAgentSessionCard(session, normalizedRepo))
    .sort((a, b) => {
      if (a.matchesRepo !== b.matchesRepo) return a.matchesRepo ? -1 : 1;
      if (a.unseen !== b.unseen) return a.unseen ? -1 : 1;
      return a.name.localeCompare(b.name);
    });
}

function toAgentSessionCard(session: OpenSessionsSessionData, normalizedRepo: string): AgentSessionCard {
  const matchesRepo = sessionDirMatchesRepo(session.dir, normalizedRepo);
  const metadata = session.metadata ?? null;

  return {
    name: session.name,
    dir: session.dir,
    branch: session.branch,
    dirty: session.dirty,
    unseen: session.unseen,
    matchesRepo,
    primaryAgent: session.agentState,
    agents: session.agents,
    metadataStatus: metadata?.status?.text ?? null,
    metadataTone: metadata?.status?.tone ?? null,
    metadataProgress: metadata?.progress ? formatMetadataProgress(metadata.progress) : null,
    ports: session.ports
  };
}

export async function fetchOpenSessionsSnapshot(options?: {
  host?: string;
  port?: number;
  timeoutMs?: number;
}): Promise<OpenSessionsServerState | null> {
  const WebSocketImpl = globalThis.WebSocket;
  if (typeof WebSocketImpl !== "function") {
    return null;
  }

  const { wsUrl } = resolveOpenSessionsEndpoints(options);
  const timeoutMs = options?.timeoutMs ?? 800;

  return new Promise((resolve) => {
    let settled = false;
    let ws: InstanceType<typeof WebSocketImpl>;

    const finish = (value: OpenSessionsServerState | null) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      try {
        ws.close();
      } catch {
        // ignore
      }
      resolve(value);
    };

    try {
      ws = new WebSocketImpl(wsUrl);
    } catch {
      finish(null);
      return;
    }

    const timer = setTimeout(() => finish(null), timeoutMs);

    ws.addEventListener("message", (event) => {
      try {
        const data = JSON.parse(String(event.data)) as { type?: string };
        if (data.type === "state") {
          finish(data as OpenSessionsServerState);
        }
      } catch {
        // ignore malformed payloads
      }
    });

    ws.addEventListener("error", () => finish(null));
    ws.addEventListener("close", () => {
      if (!settled) finish(null);
    });
  });
}

export async function readOpenSessionsStatus(
  repoRoot: string,
  options?: { host?: string; port?: number; timeoutMs?: number }
): Promise<OpenSessionsStatus> {
  const { httpUrl, wsUrl } = resolveOpenSessionsEndpoints(options);
  const startCommand =
    "Install opensessions in tmux (TPM: set -g @plugin 'Ataraxy-Labs/opensessions'), then open the sidebar with prefix o → s.";

  const snapshot = await fetchOpenSessionsSnapshot(options);
  if (!snapshot) {
    return {
      healthy: false,
      url: httpUrl,
      wsUrl,
      currentSession: null,
      focusedSession: null,
      activeSessionSummary: null,
      sidebarFocusSummary: null,
      sessions: [],
      startCommand
    };
  }

  const sessions = mapOpenSessionsForRepo(snapshot, repoRoot);
  const currentSession = normalizeSessionRef(snapshot.currentSession, snapshot.sessions);
  const focusedSession = normalizeSessionRef(snapshot.focusedSession, snapshot.sessions);
  const activeName = currentSession ?? focusedSession;
  const activeSessionSummary = formatSessionSummary(activeName, sessions);
  const sidebarFocusSummary =
    focusedSession && focusedSession !== currentSession
      ? formatSessionSummary(focusedSession, sessions)
      : null;

  return {
    healthy: true,
    url: httpUrl,
    wsUrl,
    currentSession,
    focusedSession,
    activeSessionSummary,
    sidebarFocusSummary,
    sessions,
    startCommand
  };
}
