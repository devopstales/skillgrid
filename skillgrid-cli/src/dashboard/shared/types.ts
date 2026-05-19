export type ToolId = "gitnexus" | "openspecui" | "opensessions";

export type AgentStatusTone =
  | "idle"
  | "running"
  | "tool-running"
  | "done"
  | "error"
  | "waiting"
  | "interrupted"
  | "stale";

export type AgentThreadSnapshot = {
  agent: string;
  status: AgentStatusTone;
  threadId?: string;
  threadName?: string;
  unseen?: boolean;
  ts: number;
};

export type AgentSessionSnapshot = {
  name: string;
  dir: string;
  branch: string;
  dirty: boolean;
  unseen: boolean;
  matchesRepo: boolean;
  primaryAgent: AgentThreadSnapshot | null;
  agents: AgentThreadSnapshot[];
  metadataStatus: string | null;
  metadataTone: "neutral" | "info" | "success" | "warn" | "error" | null;
  metadataProgress: string | null;
  ports: number[];
};

export type OpenSessionsIntegration = {
  healthy: boolean;
  url: string;
  wsUrl: string;
  focusedSession: string | null;
  sessions: AgentSessionSnapshot[];
  startCommand: string;
};

export type TaskStats = {
  total: number;
  completed: number;
  hitl: number;
  afk: number;
  blocked: number;
};

export type DashboardEvent = {
  id: string;
  sourcePath: string;
  line: number;
  time?: string;
  changeId?: string;
  prd?: string;
  phase?: string;
  node?: string;
  status?: string;
  summary?: string;
  blocker?: string;
  artifacts?: unknown;
  external?: unknown;
  raw: Record<string, unknown>;
};

export type PreviewLink = {
  id: string;
  title: string;
  path: string;
  url: string;
  changeId?: string;
};

export type PrdCard = {
  id: string;
  title: string;
  status: string;
  path: string;
  changeId?: string;
  specPath?: string;
  contextPath?: string;
  owner?: string;
  agent?: string;
  previewUrl?: string;
  externalUrl?: string;
  summary?: string;
  body?: string;
  taskStats: TaskStats;
  lastEvent?: DashboardEvent;
};

export type BoardSubtask = {
  id: string;
  key: string;
  title: string;
  status: string;
  path: string;
  changeId: string;
  capability?: string;
};

export type BoardLane = {
  id: string;
  label: string;
};

export type BoardIssue = {
  id: string;
  key: string;
  title: string;
  lane: string;
  status: string;
  source: "prd" | "change" | "merged";
  path: string;
  changeId?: string;
  prdPath?: string;
  specPath?: string;
  contextPath?: string;
  changePath?: string;
  owner?: string;
  agent?: string;
  previewUrl?: string;
  externalUrl?: string;
  summary?: string;
  prdBody?: string;
  phase?: string;
  taskStats: TaskStats;
  subtasks: BoardSubtask[];
  lastEvent?: DashboardEvent;
};

export type WorkflowItem = {
  id: string;
  title: string;
  status: string;
  phase: string;
  sourcePath: string;
  changeId?: string;
  taskStats: TaskStats;
  lastEvent?: DashboardEvent;
};

export type OpenSpecChange = {
  id: string;
  status: "active" | "archived";
  path: string;
  proposalPath?: string;
  tasksPath?: string;
  taskStats: TaskStats;
};

export type OpenSpecSpec = {
  id: string;
  path: string;
  title: string;
  changeId?: string;
};

export type HandoffLog = {
  changeId: string;
  path: string;
  content: string;
};

export type Checkpoint = {
  id: string;
  sourcePath: string;
  line: number;
  time?: string;
  name?: string;
  branch?: string;
  sha?: string;
  dirty?: string;
  prd?: string;
  changeId?: string;
  context?: string;
  evidence?: string;
  raw: string;
};

export type ToolStatus = {
  id: ToolId;
  name: string;
  url: string;
  healthy: boolean;
  startCommand: string;
};

export type DashboardData = {
  repoRoot: string;
  repoName: string;
  generatedAt: string;
  lanes: BoardLane[];
  issues: BoardIssue[];
  prds: PrdCard[];
  workflow: WorkflowItem[];
  events: DashboardEvent[];
  previews: PreviewLink[];
  changes: OpenSpecChange[];
  specs: OpenSpecSpec[];
  handoffs: HandoffLog[];
  checkpoints: Checkpoint[];
  tools: ToolStatus[];
  openSessions: OpenSessionsIntegration;
  warnings: string[];
};
