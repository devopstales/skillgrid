import path from "node:path";
import type {
  BoardIssue,
  BoardLane,
  BoardSubtask,
  Checkpoint,
  DashboardData,
  DashboardEvent,
  HandoffLog,
  OpenSpecChange,
  OpenSpecSpec,
  PrdCard,
  PreviewLink,
  TaskStats,
  ToolStatus,
  WorkflowItem
} from "../shared/types.js";
import { listDirectories, listFilesRecursive, readTextIfExists, slugify, toPosixRelative } from "./fs-utils.js";
import { readOpenSessionsStatus } from "./opensessions-client.js";
import { EMPTY_TASK_STATS, firstParagraph, parseMarkdownMetadata, parseTaskStats } from "./parsers.js";

export type BuildDashboardOptions = {
  repoRoot: string;
  /** e.g. http://127.0.0.1:5241 — used for bundled GitNexus at /gitnexus/ */
  dashboardOrigin: string;
  gitnexusUrl?: string;
  openspecUiUrl?: string;
  openSessionsHost?: string;
  openSessionsPort?: number;
};

type SkillgridConfig = {
  fallbackStatus: string;
  lanes: BoardLane[];
  phaseStatusMap: Record<string, string>;
};

export async function buildDashboardData(options: BuildDashboardOptions): Promise<DashboardData> {
  const repoRoot = path.resolve(options.repoRoot);
  const warnings: string[] = [];
  const events = await readEvents(repoRoot, warnings);
  const previews = await readPreviews(repoRoot);
  const changes = await readOpenSpecChanges(repoRoot);
  const specs = await readOpenSpecSpecs(repoRoot);
  const handoffs = await readHandoffs(repoRoot);
  const checkpoints = await readCheckpoints(repoRoot);
  const prds = await readPrds(repoRoot, events);
  const workflow = buildWorkflowItems(changes, events);
  const skillgridConfig = await readSkillgridConfig(repoRoot);
  const issues = buildBoardIssues({ prds, changes, specs, events, previews, skillgridConfig });
  const origin = options.dashboardOrigin.replace(/\/$/, "");
  const openSessions = await readOpenSessionsStatus(repoRoot, {
    host: options.openSessionsHost,
    port: options.openSessionsPort
  });
  const tools = await readToolStatuses({
    gitnexusUrl: options.gitnexusUrl ?? `${origin}/gitnexus/`,
    openspecUiUrl: options.openspecUiUrl ?? "http://localhost:3100",
    openSessionsUrl: openSessions.url,
    openSessionsHealthy: openSessions.healthy
  });

  if (prds.length === 0) {
    warnings.push("No PRDs found under .skillgrid/prd.");
  }
  if (changes.length === 0) {
    warnings.push("No OpenSpec changes found under openspec/changes.");
  }

  return {
    repoRoot,
    repoName: path.basename(repoRoot),
    generatedAt: new Date().toISOString(),
    lanes: skillgridConfig.lanes,
    issues,
    prds,
    workflow,
    events,
    previews,
    changes,
    specs,
    handoffs,
    checkpoints,
    tools,
    openSessions,
    warnings
  };
}

async function readPrds(repoRoot: string, events: DashboardEvent[]): Promise<PrdCard[]> {
  const prdRoot = path.join(repoRoot, ".skillgrid", "prd");
  const files = (await listFilesRecursive(prdRoot)).filter((file) => file.endsWith(".md") && path.basename(file) !== "INDEX.md");
  const cards = await Promise.all(
    files.map(async (file) => {
      const content = (await readTextIfExists(file)) ?? "";
      const metadata = parseMarkdownMetadata(content);
      const changeId = metadata.changeId ?? extractOpenSpecChangeId(content);
      const id = slugify(path.basename(file, ".md")) || path.basename(file, ".md");
      const lastEvent = newestEventFor(events, changeId ?? id);

      return {
        id,
        title: metadata.title ?? titleFromFile(file),
        status: metadata.status ?? "Backlog",
        path: toPosixRelative(repoRoot, file),
        changeId,
        specPath: metadata.specPath,
        contextPath: metadata.contextPath,
        owner: metadata.owner,
        agent: metadata.agent,
        previewUrl: metadata.previewUrl,
        externalUrl: metadata.externalUrl,
        summary: metadata.summary ?? firstParagraph(content),
        body: content,
        taskStats: parseTaskStats(content),
        lastEvent
      };
    })
  );

  return cards.sort((a, b) => a.title.localeCompare(b.title));
}

async function readSkillgridConfig(repoRoot: string): Promise<SkillgridConfig> {
  const fallback: SkillgridConfig = {
    fallbackStatus: "draft",
    lanes: [
      { id: "draft", label: "Draft" },
      { id: "todo", label: "Todo" },
      { id: "inprogress", label: "In Progress" },
      { id: "devdone", label: "Dev Done" },
      { id: "done", label: "Done" },
      { id: "archived", label: "Archived" }
    ],
    phaseStatusMap: {
      plan: "draft",
      breakdown: "todo",
      apply: "inprogress",
      validate: "devdone",
      finish: "done"
    }
  };
  const content = await readTextIfExists(path.join(repoRoot, ".skillgrid", "config.json"));
  if (!content) {
    return fallback;
  }

  try {
    const parsed = JSON.parse(content) as {
      prdWorkflow?: {
        fallbackStatus?: string;
        statuses?: Array<{ id?: string; label?: string }>;
        phaseStatusMap?: Record<string, string>;
      };
    };
    const configuredStatuses = (parsed.prdWorkflow?.statuses ?? [])
      .map((status) => {
        const id = normalizeStatusId(status.id);
        return id ? { id, label: status.label ?? titleCase(status.id ?? id) } : undefined;
      })
      .filter((status): status is BoardLane => Boolean(status));
    const lanes = configuredStatuses?.length ? configuredStatuses : fallback.lanes.filter((lane) => lane.id !== "archived");
    if (!lanes.some((lane) => lane.id === "archived")) {
      lanes.push({ id: "archived", label: "Archived" });
    }

    return {
      fallbackStatus: normalizeStatusId(parsed.prdWorkflow?.fallbackStatus ?? fallback.fallbackStatus) ?? fallback.fallbackStatus,
      lanes,
      phaseStatusMap: normalizePhaseStatusMap(parsed.prdWorkflow?.phaseStatusMap ?? fallback.phaseStatusMap)
    };
  } catch {
    return fallback;
  }
}

async function readOpenSpecChanges(repoRoot: string): Promise<OpenSpecChange[]> {
  const changesRoot = path.join(repoRoot, "openspec", "changes");
  const directories = await listDirectories(changesRoot);
  const archiveRoot = path.join(changesRoot, "archive");
  const activeDirs = directories.filter((directory) => path.basename(directory) !== "archive");
  const archiveDirs = await listDirectories(archiveRoot);

  const all = [
    ...activeDirs.map((directory) => ({ directory, status: "active" as const })),
    ...archiveDirs.map((directory) => ({ directory, status: "archived" as const }))
  ];

  return Promise.all(
    all.map(async ({ directory, status }) => {
      const tasksPath = path.join(directory, "tasks.md");
      const proposalPath = path.join(directory, "proposal.md");
      const tasks = (await readTextIfExists(tasksPath)) ?? "";
      const id = path.basename(directory);

      return {
        id,
        status,
        path: toPosixRelative(repoRoot, directory),
        proposalPath: (await readTextIfExists(proposalPath)) === undefined ? undefined : toPosixRelative(repoRoot, proposalPath),
        tasksPath: tasks ? toPosixRelative(repoRoot, tasksPath) : undefined,
        taskStats: parseTaskStats(tasks)
      };
    })
  );
}

async function readOpenSpecSpecs(repoRoot: string): Promise<OpenSpecSpec[]> {
  const specsRoot = path.join(repoRoot, "openspec", "specs");
  const mainSpecFiles = (await listFilesRecursive(specsRoot)).filter((file) => file.endsWith(".md"));
  const changeSpecFiles = (await listFilesRecursive(path.join(repoRoot, "openspec", "changes")))
    .filter((file) => file.endsWith(".md") && file.split(path.sep).includes("specs"));
  const files = [...mainSpecFiles, ...changeSpecFiles];

  return Promise.all(
    files.map(async (file) => {
      const content = (await readTextIfExists(file)) ?? "";
      const metadata = parseMarkdownMetadata(content);
      const changeId = changeIdForSpec(repoRoot, file);
      const relativeRoot = changeId ? path.join(repoRoot, "openspec", "changes", changeId, "specs") : specsRoot;
      const id = slugify(path.relative(relativeRoot, file).replace(/\.md$/, "")) || path.basename(file, ".md");
      return {
        id,
        path: toPosixRelative(repoRoot, file),
        title: metadata.title ?? titleFromFile(file),
        changeId
      };
    })
  );
}

async function readEvents(repoRoot: string, warnings: string[]): Promise<DashboardEvent[]> {
  const eventsRoot = path.join(repoRoot, ".skillgrid", "tasks", "events");
  const files = (await listFilesRecursive(eventsRoot)).filter((file) => file.endsWith(".jsonl"));
  const events: DashboardEvent[] = [];

  for (const file of files) {
    const content = (await readTextIfExists(file)) ?? "";
    const lines = content.split(/\r?\n/);
    lines.forEach((line, index) => {
      if (!line.trim()) {
        return;
      }

      try {
        const raw = JSON.parse(line) as Record<string, unknown>;
        const changeId = stringField(raw.changeId) ?? path.basename(file, ".jsonl");
        events.push({
          id: `${toPosixRelative(repoRoot, file)}:${index + 1}`,
          sourcePath: toPosixRelative(repoRoot, file),
          line: index + 1,
          time: stringField(raw.time),
          changeId,
          prd: stringField(raw.prd),
          phase: stringField(raw.phase),
          node: stringField(raw.node),
          status: stringField(raw.status),
          summary: stringField(raw.summary),
          blocker: stringField(raw.blocker),
          artifacts: raw.artifacts,
          external: raw.external,
          raw
        });
      } catch {
        warnings.push(`Ignored malformed JSONL entry in ${toPosixRelative(repoRoot, file)}:${index + 1}.`);
      }
    });
  }

  return events.sort((a, b) => (b.time ?? "").localeCompare(a.time ?? ""));
}

async function readHandoffs(repoRoot: string): Promise<HandoffLog[]> {
  const tasksRoot = path.join(repoRoot, ".skillgrid", "tasks");
  const files = (await listFilesRecursive(tasksRoot)).filter((file) => /^context_.+\.md$/.test(path.basename(file)));

  return Promise.all(
    files.map(async (file) => ({
      changeId: path.basename(file, ".md").replace(/^context_/, ""),
      path: toPosixRelative(repoRoot, file),
      content: (await readTextIfExists(file)) ?? ""
    }))
  );
}

async function readCheckpoints(repoRoot: string): Promise<Checkpoint[]> {
  const file = path.join(repoRoot, ".skillgrid", "tasks", "checkpoints.log");
  const content = await readTextIfExists(file);
  if (!content) {
    return [];
  }
  const sourcePath = toPosixRelative(repoRoot, file);
  return content
    .split(/\r?\n/)
    .map((line, index) => parseCheckpointLine(line, sourcePath, index + 1))
    .filter((checkpoint): checkpoint is Checkpoint => Boolean(checkpoint))
    .sort((a, b) => (b.time ?? "").localeCompare(a.time ?? ""));
}

function parseCheckpointLine(line: string, sourcePath: string, lineNumber: number): Checkpoint | undefined {
  const trimmed = line.trim();
  if (!trimmed || trimmed.startsWith("#")) {
    return undefined;
  }
  const [time, ...rest] = trimmed.split(/\s+/);
  if (!/^\d{4}-\d{2}-\d{2}T/.test(time)) {
    return undefined;
  }
  const fields = parseCheckpointFields(rest.join(" "));
  if (Object.keys(fields).length === 0) {
    return undefined;
  }
  return {
    id: `${sourcePath}:${lineNumber}`,
    sourcePath,
    line: lineNumber,
    time,
    name: fields.name,
    branch: fields.branch,
    sha: fields.sha,
    dirty: fields.dirty,
    prd: fields.prd,
    changeId: fields.change,
    context: fields.context,
    evidence: fields.evidence,
    raw: trimmed
  };
}

function parseCheckpointFields(value: string): Record<string, string> {
  const fields: Record<string, string> = {};
  const pattern = /(\w+)=("([^"]*)"|[^\s]+)/g;
  let match: RegExpExecArray | null;
  while ((match = pattern.exec(value)) !== null) {
    fields[match[1]] = match[3] ?? match[2];
  }
  return fields;
}

async function readPreviews(repoRoot: string): Promise<PreviewLink[]> {
  const previewRoot = path.join(repoRoot, ".skillgrid", "preview");
  const files = await listFilesRecursive(previewRoot);

  return files.map((file) => {
    const relative = toPosixRelative(repoRoot, file);
    const baseName = path.basename(file);
    const id = slugify(relative) || relative;
    return {
      id,
      title: baseName,
      path: relative,
      url: `/preview/${encodeURIComponent(relative)}`,
      changeId: inferChangeId(baseName)
    };
  });
}

function buildWorkflowItems(changes: OpenSpecChange[], events: DashboardEvent[]): WorkflowItem[] {
  const fromChanges = changes.map((change) => {
    const lastEvent = newestEventFor(events, change.id);
    return {
      id: change.id,
      title: change.id,
      status: workflowStatus(change.taskStats, lastEvent?.status),
      phase: lastEvent?.phase ?? lastEvent?.node ?? (change.status === "archived" ? "Archived" : "OpenSpec"),
      sourcePath: change.tasksPath ?? change.path,
      changeId: change.id,
      taskStats: change.taskStats,
      lastEvent
    };
  });

  const eventOnly = events
    .filter((event) => event.changeId && !changes.some((change) => change.id === event.changeId))
    .map((event) => ({
      id: event.changeId ?? event.id,
      title: event.changeId ?? event.summary ?? event.id,
      status: event.status ?? "Observed",
      phase: event.phase ?? event.node ?? "Event",
      sourcePath: event.sourcePath,
      changeId: event.changeId,
      taskStats: EMPTY_TASK_STATS,
      lastEvent: event
    }));

  return [...fromChanges, ...eventOnly].sort((a, b) => a.title.localeCompare(b.title));
}

function buildBoardIssues(input: {
  prds: PrdCard[];
  changes: OpenSpecChange[];
  specs: OpenSpecSpec[];
  events: DashboardEvent[];
  previews: PreviewLink[];
  skillgridConfig: SkillgridConfig;
}): BoardIssue[] {
  const usedChanges = new Set<string>();
  const issues: BoardIssue[] = [];

  for (const prd of input.prds) {
    const change = findChangeForPrd(prd, input.changes);
    if (change) {
      usedChanges.add(change.id);
    }
    issues.push(issueFromPrdAndChange(prd, change, input.specs, input.events, input.previews, input.skillgridConfig));
  }

  for (const change of input.changes) {
    if (usedChanges.has(change.id)) {
      continue;
    }
    issues.push(issueFromChange(change, input.specs, input.events, input.previews, input.skillgridConfig));
  }

  return issues.sort((a, b) => a.key.localeCompare(b.key));
}

function issueFromPrdAndChange(
  prd: PrdCard,
  change: OpenSpecChange | undefined,
  specs: OpenSpecSpec[],
  events: DashboardEvent[],
  previews: PreviewLink[],
  skillgridConfig: SkillgridConfig
): BoardIssue {
  const changeId = prd.changeId ?? change?.id;
  const lastEvent = changeId ? newestEventFor(events, changeId) : prd.lastEvent;
  const taskStats = mergeTaskStats(prd.taskStats, change?.taskStats);
  const lane = issueLane({
    eventStatus: lastEvent?.status,
    phase: lastEvent?.phase ?? lastEvent?.node,
    prdStatus: prd.status,
    changeStatus: change?.status,
    taskStats,
    skillgridConfig
  });
  const status = issueStatusLabel(prd.status, skillgridConfig);

  return {
    id: prd.id,
    key: issueKey(changeId ?? prd.id),
    title: prd.title,
    lane,
    status,
    source: change ? "merged" : "prd",
    path: prd.path,
    changeId,
    prdPath: prd.path,
    specPath: prd.specPath ?? change?.path,
    contextPath: prd.contextPath ?? (changeId ? `.skillgrid/tasks/context_${changeId}.md` : undefined),
    changePath: change?.path,
    owner: prd.owner,
    agent: prd.agent,
    previewUrl: prd.previewUrl ?? previewForIssue(previews, changeId, prd.id),
    externalUrl: prd.externalUrl,
    summary: prd.summary,
    prdBody: prd.body,
    phase: lastEvent?.phase ?? lastEvent?.node,
    taskStats,
    subtasks: changeId ? subtasksForChange(changeId, specs) : [],
    lastEvent
  };
}

function issueFromChange(
  change: OpenSpecChange,
  specs: OpenSpecSpec[],
  events: DashboardEvent[],
  previews: PreviewLink[],
  skillgridConfig: SkillgridConfig
): BoardIssue {
  const lastEvent = newestEventFor(events, change.id);
  const lane = issueLane({
    eventStatus: lastEvent?.status,
    phase: lastEvent?.phase ?? lastEvent?.node,
    changeStatus: change.status,
    taskStats: change.taskStats,
    skillgridConfig
  });
  const status = labelForLane(lane, skillgridConfig);

  return {
    id: change.id,
    key: issueKey(change.id),
    title: titleFromChangeId(change.id),
    lane,
    status,
    source: "change",
    path: change.path,
    changeId: change.id,
    changePath: change.path,
    previewUrl: previewForIssue(previews, change.id, undefined),
    phase: lastEvent?.phase ?? lastEvent?.node,
    taskStats: change.taskStats,
    subtasks: subtasksForChange(change.id, specs),
    lastEvent
  };
}

function subtasksForChange(changeId: string, specs: OpenSpecSpec[]): BoardSubtask[] {
  return specs
    .filter((spec) => spec.changeId === changeId)
    .map((spec, index) => ({
      id: `${changeId}:${spec.id}`,
      key: `${issueKey(changeId)}-${index + 1}`,
      title: spec.title,
      status: "Spec",
      path: spec.path,
      changeId,
      capability: spec.id
    }));
}

function findChangeForPrd(prd: PrdCard, changes: OpenSpecChange[]): OpenSpecChange | undefined {
  if (prd.changeId) {
    return changes.find((change) => change.id === prd.changeId);
  }
  const normalizedPrdTitle = slugify(prd.title.replace(/^PRD\d+:\s*/i, ""));
  return changes.find((change) => {
    const normalizedChange = slugify(stripArchiveDate(change.id));
    return normalizedChange === prd.id || change.id === prd.id || normalizedChange === normalizedPrdTitle;
  });
}

function previewForIssue(previews: PreviewLink[], changeId: string | undefined, prdSlug: string | undefined): string | undefined {
  const needles = [changeId, prdSlug].filter((value): value is string => Boolean(value)).map((value) => value.toLowerCase());
  return previews.find((preview) => {
    const file = path.basename(preview.path).toLowerCase();
    return needles.some((needle) => file.startsWith(needle));
  })?.url;
}

function mergeTaskStats(primary: TaskStats, secondary?: TaskStats): TaskStats {
  if (!secondary) {
    return primary;
  }
  return {
    total: primary.total + secondary.total,
    completed: primary.completed + secondary.completed,
    hitl: primary.hitl + secondary.hitl,
    afk: primary.afk + secondary.afk,
    blocked: primary.blocked + secondary.blocked
  };
}

function issueLane(input: {
  eventStatus?: string;
  phase?: string;
  prdStatus?: string;
  changeStatus?: OpenSpecChange["status"];
  taskStats: TaskStats;
  skillgridConfig: SkillgridConfig;
}): string {
  if (input.changeStatus === "archived") return "archived";
  const laneIds = new Set(input.skillgridConfig.lanes.map((lane) => lane.id));
  const normalizedPhase = normalizeStatusId(input.phase);
  const phaseStatus = normalizedPhase ? input.skillgridConfig.phaseStatusMap[normalizedPhase] : undefined;
  const candidates = [
    input.prdStatus,
    phaseStatus,
    input.eventStatus,
    workflowStatus(input.taskStats, undefined),
    input.skillgridConfig.fallbackStatus
  ];

  for (const candidate of candidates) {
    const normalized = normalizeStatusId(candidate);
    if (normalized && laneIds.has(normalized)) {
      return normalized;
    }
  }

  return input.skillgridConfig.fallbackStatus;
}

async function readToolStatuses(urls: {
  gitnexusUrl: string;
  openspecUiUrl: string;
  openSessionsUrl?: string;
  openSessionsHealthy: boolean;
}): Promise<ToolStatus[]> {
  const [gitnexusHealthy, openspecHealthy] = await Promise.all([isHealthy(urls.gitnexusUrl), isHealthy(urls.openspecUiUrl)]);

  return [
    {
      id: "gitnexus",
      name: "GitNexus Web UI",
      url: urls.gitnexusUrl,
      healthy: gitnexusHealthy,
      startCommand:
        "Bundled under this dashboard at /gitnexus/ (build: npm run build:gitnexus in skillgrid-cli). Start the GitNexus API server separately and open /gitnexus/?server=http://127.0.0.1:4747"
    },
    {
      id: "openspecui",
      name: "OpenSpecUI",
      url: urls.openspecUiUrl,
      healthy: openspecHealthy,
      startCommand: "npx openspecui@latest"
    },
    {
      id: "opensessions",
      name: "opensessions",
      url: urls.openSessionsUrl ?? "http://127.0.0.1:7391",
      healthy: urls.openSessionsHealthy,
      startCommand:
        "TPM: set -g @plugin 'Ataraxy-Labs/opensessions' — then prefix o → s in tmux. Agent status in this dashboard uses the opensessions WebSocket API on port 7391."
    }
  ];
}

async function isHealthy(url: string): Promise<boolean> {
  try {
    const response = await fetch(url, { method: "GET", signal: AbortSignal.timeout(800) });
    return response.ok;
  } catch {
    return false;
  }
}

function newestEventFor(events: DashboardEvent[], id: string): DashboardEvent | undefined {
  return events.find((event) => event.changeId === id || event.prd === id);
}

function workflowStatus(stats: TaskStats, fallback?: string): string {
  if (fallback) {
    return fallback;
  }
  if (stats.total === 0) {
    return "No Tasks";
  }
  if (stats.completed === stats.total) {
    return "Done";
  }
  if (stats.blocked > 0) {
    return "Blocked";
  }
  if (stats.completed > 0) {
    return "In Progress";
  }
  return "Planned";
}

function titleFromFile(file: string): string {
  return path
    .basename(file, path.extname(file))
    .replace(/[_-]+/g, " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function titleFromChangeId(changeId: string): string {
  return changeId.replace(/[_-]+/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function issueKey(id: string): string {
  return slugify(id).toUpperCase().replace(/-/g, "-") || id.toUpperCase();
}

function titleCase(value: string): string {
  return value.replace(/[_-]+/g, " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function labelForLane(laneId: string, config: SkillgridConfig): string {
  return config.lanes.find((lane) => lane.id === laneId)?.label ?? titleCase(laneId);
}

function issueStatusLabel(prdStatus: string, config: SkillgridConfig): string {
  const normalized = normalizeStatusId(prdStatus);
  if (normalized) {
    return labelForLane(normalized, config);
  }
  return titleCase(prdStatus);
}

function normalizeStatusId(value: string | undefined): string | undefined {
  if (!value) {
    return undefined;
  }
  return value.trim().toLowerCase().replace(/[\s_-]+/g, "");
}

function normalizePhaseStatusMap(map: Record<string, string>): Record<string, string> {
  return Object.fromEntries(
    Object.entries(map).flatMap(([phase, status]) => {
      const normalizedPhase = normalizeStatusId(phase);
      const normalizedStatus = normalizeStatusId(status);
      return normalizedPhase && normalizedStatus ? [[normalizedPhase, normalizedStatus]] : [];
    })
  );
}

function extractOpenSpecChangeId(content: string): string | undefined {
  const matches = [...content.matchAll(/openspec\/changes\/((?:archive\/)?[^)`\]\s]+)/gi)];
  for (const candidate of matches) {
    const match = candidate[1]?.replace(/\/$/, "");
    if (!match) {
      continue;
    }
    const parts = match.split("/");
    const changeId = parts[parts.length - 1].replace(/[`.,]+$/g, "");
    if (/^[a-z0-9][a-z0-9-]*$/i.test(changeId)) {
      return changeId;
    }
  }
  return undefined;
}

function stripArchiveDate(changeId: string): string {
  return changeId.replace(/^\d{4}-\d{2}-\d{2}-/, "");
}

function stringField(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

function inferChangeId(fileName: string): string | undefined {
  const match = fileName.match(/^([a-z0-9][a-z0-9-]+)/i);
  return match?.[1];
}

function changeIdForSpec(repoRoot: string, file: string): string | undefined {
  const relativeParts = path.relative(path.join(repoRoot, "openspec", "changes"), file).split(path.sep);
  if (relativeParts.length >= 3 && relativeParts[1] === "specs") {
    return relativeParts[0];
  }
  if (relativeParts.length >= 4 && relativeParts[0] === "archive" && relativeParts[2] === "specs") {
    return relativeParts[1];
  }
  return undefined;
}
