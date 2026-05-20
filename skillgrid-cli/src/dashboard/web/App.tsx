import { useEffect, useState } from "react";
import type { ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type {
  AgentSessionSnapshot,
  AgentStatusTone,
  BoardIssue,
  BoardLane,
  DashboardData,
  DashboardEvent,
  ToolStatus
} from "../shared/types.js";

type View = "board" | "agents" | "tools";

export default function App() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [activeView, setActiveView] = useState<View>("board");
  const [issuePageId, setIssuePageId] = useState(readIssueIdFromPath);

  useEffect(() => {
    const onPopState = () => setIssuePageId(readIssueIdFromPath());
    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
  }, []);

  useEffect(() => {
    let cancelled = false;

    const load = () =>
      fetch("/api/dashboard")
        .then(async (response) => {
          if (!response.ok) throw new Error(await response.text());
          return response.json() as Promise<DashboardData>;
        })
        .then((payload) => {
          if (!cancelled) setData(payload);
        })
        .catch((caught: unknown) => {
          if (!cancelled) {
            setError(caught instanceof Error ? caught.message : "Failed to load dashboard data");
          }
        });

    void load();
    const interval =
      activeView === "agents" && !issuePageId ? window.setInterval(() => void load(), 3000) : undefined;

    return () => {
      cancelled = true;
      if (interval) window.clearInterval(interval);
    };
  }, [activeView, issuePageId]);

  const selectedIssue = data?.issues.find((issue) => issue.id === issuePageId);

  const openIssue = (id: string) => {
    window.history.pushState(null, "", `/issue/${encodeURIComponent(id)}`);
    setIssuePageId(id);
  };

  const closeIssue = () => {
    window.history.pushState(null, "", "/");
    setIssuePageId(null);
  };

  return (
    <main className="shell">
      <header className="hero">
        <div>
          <p className="eyebrow">Skillgrid Dashboard</p>
          <h1>{data?.repoName ?? "Loading repo..."}</h1>
          <p className="repo-path">{data?.repoRoot ?? "Reading repo-local Skillgrid and OpenSpec files"}</p>
        </div>
        {data ? (
          <div className="hero-stats">
            <Stat label="Issues" value={data.issues.length} />
            <Stat label="Subtasks" value={data.issues.reduce((count, issue) => count + issue.subtasks.length, 0)} />
            <Stat label="Events" value={data.events.length} />
            <Stat label="Checkpoints" value={data.checkpoints.length} />
            <Stat label="Previews" value={data.previews.length} />
          </div>
        ) : null}
      </header>

      {!selectedIssue ? (
        <nav className="tabs" aria-label="Dashboard views">
          <Tab active={activeView === "board"} onClick={() => setActiveView("board")}>Board</Tab>
          <Tab active={activeView === "agents"} onClick={() => setActiveView("agents")}>Agent View</Tab>
          <Tab active={activeView === "tools"} onClick={() => setActiveView("tools")}>Tools</Tab>
          <Tab active={false} onClick={() => window.location.assign("/gitnexus/")}>
            GitNexus
          </Tab>
          <Tab active={false} onClick={() => window.location.assign("/truecourse/")}>
            TrueCourse
          </Tab>
        </nav>
      ) : null}

      {error ? <div className="error">{error}</div> : null}
      {!data && !error ? <div className="empty">Loading dashboard data...</div> : null}

      {data ? (
        <>
          {selectedIssue ? (
            <IssuePage
              issue={selectedIssue}
              handoffs={data.handoffs}
              checkpoints={data.checkpoints}
              events={data.events}
              onBack={closeIssue}
            />
          ) : (
            <>
          {data.warnings.length > 0 ? (
            <section className="warnings">
              {data.warnings.map((warning) => (
                <span key={warning}>{warning}</span>
              ))}
            </section>
          ) : null}

          {activeView === "board" ? <IssueBoard lanes={data.lanes} issues={data.issues} onOpenIssue={openIssue} /> : null}
          {activeView === "agents" ? <AgentView data={data} /> : null}
          {activeView === "tools" ? <ToolsView tools={data.tools} /> : null}
            </>
          )}
        </>
      ) : null}
    </main>
  );
}

function IssueBoard({ lanes, issues, onOpenIssue }: { lanes: BoardLane[]; issues: BoardIssue[]; onOpenIssue: (id: string) => void }) {
  const groups = groupBy(issues, (issue) => issue.lane || "Backlog");
  return (
    <section className="board">
      {lanes.map((lane) => {
        const laneIssues = groups[lane.id] ?? [];
        return (
        <div className="lane" key={lane.id}>
          <LaneHeader title={lane.label} count={laneIssues.length} />
          {laneIssues.map((issue) => (
            <article className="card issue-card" key={issue.id}>
              <span className="issue-header">
                <a
                  className="issue-key issue-title-link"
                  href={`/issue/${encodeURIComponent(issue.id)}`}
                  onClick={(event) => {
                    event.preventDefault();
                    onOpenIssue(issue.id);
                  }}
                >
                  {issue.key}
                </a>
                <span className={`status-pill ${statusClass(issue.status)}`}>{issue.status}</span>
              </span>
              <span className="issue-progress">
                <span>{formatTasks(issue.taskStats.completed, issue.taskStats.total) ?? "No tasks"}</span>
                <span className="issue-progress-bar">
                  <span style={{ width: `${taskProgress(issue.taskStats.completed, issue.taskStats.total)}%` }} />
                </span>
              </span>
            </article>
          ))}
          {laneIssues.length === 0 ? <EmptyLane /> : null}
        </div>
      );
      })}
      {issues.length === 0 ? <EmptyState title="No issues yet" body="PRDs and OpenSpec changes will appear here as Jira-style tasks." /> : null}
    </section>
  );
}

function IssuePage({
  issue,
  handoffs,
  checkpoints,
  events,
  onBack
}: {
  issue: BoardIssue;
  handoffs: DashboardData["handoffs"];
  checkpoints: DashboardData["checkpoints"];
  events: DashboardEvent[];
  onBack: () => void;
}) {
  const related = events.filter((event) => event.changeId === issue.changeId || event.prd === issue.id);
  const relatedCheckpoints = checkpoints.filter(
    (checkpoint) => checkpoint.changeId === issue.changeId || checkpoint.prd === issue.prdPath || checkpoint.context === issue.contextPath
  );
  const handoff = handoffs.find((entry) => entry.changeId === issue.changeId);

  return (
    <section className="issue-page">
      <header className="issue-page-header">
        <button className="close issue-page-close" onClick={onBack} aria-label="Close issue detail">Close</button>
      </header>
      <div className="issue-page-grid">
        <article className="issue-main">
          <p className="eyebrow">Task</p>
          <h1>{issue.key}</h1>
          <dl className="issue-fields">
            <div>
              <dt>Status</dt>
              <dd><span className={`status-pill ${statusClass(issue.status)}`}>{issue.status}</span></dd>
            </div>
            <div>
              <dt>PRD</dt>
              <dd>{issue.prdPath ?? "No PRD linked"}</dd>
            </div>
            <div>
              <dt>Spec / change</dt>
              <dd>{issue.specPath || "None"}</dd>
            </div>
            <div>
              <dt>Session context</dt>
              <dd>{issue.contextPath ?? "No session context linked"}</dd>
            </div>
            <div>
              <dt>Change</dt>
              <dd>{issue.changeId ?? "No OpenSpec change linked"}</dd>
            </div>
            <div>
              <dt>Tasks</dt>
              <dd>{formatTaskRatio(issue.taskStats.completed, issue.taskStats.total)}</dd>
            </div>
          </dl>
          <section>
            <h2>Preview</h2>
            {issue.previewUrl ? (
              <a href={issue.previewUrl} target="_blank" rel="noreferrer">Open generated design preview</a>
            ) : (
              <p className="repo-path">No generated preview found for this task.</p>
            )}
          </section>
          <section>
            <h2>Description</h2>
            <MarkdownContent content={issue.prdBody ?? issue.summary ?? issue.title} />
          </section>
          {handoff ? (
            <section>
              <h2>Handoff</h2>
              <MarkdownContent content={handoff.content} />
            </section>
          ) : null}
        </article>
        <aside className="issue-side">
          <h2>Checkpoints</h2>
          <CheckpointList checkpoints={relatedCheckpoints} />
          <h2>Subtasks</h2>
          {issue.subtasks.length > 0 ? (
            <div className="subtasks">
              {issue.subtasks.map((subtask) => (
                <article className="subtask" key={subtask.id}>
                  <strong>{subtask.key}</strong>
                  <span>{subtask.title}</span>
                  <small>{subtask.path}</small>
                </article>
              ))}
            </div>
          ) : (
            <div className="empty compact">No specs found for this task.</div>
          )}
          <h2>Related Events</h2>
          <Timeline events={related} />
        </aside>
      </div>
    </section>
  );
}

function AgentView({ data }: { data: DashboardData }) {
  const repoSessions = data.openSessions.sessions.filter((session) => session.matchesRepo);
  const otherSessions = data.openSessions.sessions.filter((session) => !session.matchesRepo);

  return (
    <section className="agent-view">
      <section className="panel">
        <div className="panel-heading">
          <h2>Live agent status</h2>
          <p className="panel-subtitle">
            <span className={`status-dot ${data.openSessions.healthy ? "healthy" : "offline"}`} />
            {data.openSessions.healthy
              ? `opensessions @ ${data.openSessions.url}`
              : "opensessions not reachable — start the tmux sidebar server"}
          </p>
        </div>
        {!data.openSessions.healthy ? (
          <div className="command compact-command">
            <span>Start</span>
            <code>{data.openSessions.startCommand}</code>
          </div>
        ) : null}
        {data.openSessions.activeSessionSummary ? (
          <p className="meta-line">Active tmux session: {data.openSessions.activeSessionSummary}</p>
        ) : null}
        {data.openSessions.sidebarFocusSummary ? (
          <p className="meta-line">OpenSessions sidebar focus: {data.openSessions.sidebarFocusSummary}</p>
        ) : null}
        {repoSessions.length > 0 ? (
          <div className="agent-session-grid">
            <h3>This repository</h3>
            {repoSessions.map((session) => (
              <AgentSessionCard key={session.name} session={session} />
            ))}
          </div>
        ) : data.openSessions.healthy ? (
          <div className="empty compact">
            <p>No tmux sessions matched this repo path.</p>
            <p className="meta-line">
              Dashboard repo: <code>{data.repoRoot}</code>
            </p>
            {otherSessions.length > 0 ? (
              <p className="meta-line">
                {otherSessions.length} session(s) use other directories — check cwd matches the path above (or start
                tmux in the repo root).
              </p>
            ) : (
              <p className="meta-line">No other tmux sessions reported by opensessions.</p>
            )}
          </div>
        ) : null}
        {otherSessions.length > 0 ? (
          <div className="agent-session-grid">
            <h3>Other sessions</h3>
            {otherSessions.map((session) => (
              <AgentSessionCard key={session.name} session={session} />
            ))}
          </div>
        ) : null}
      </section>
      <section className="split">
      <div className="panel">
        <h2>Handoff Logs</h2>
        {data.handoffs.map((handoff) => (
          <article className="log" key={handoff.path}>
            <h3>{handoff.changeId}</h3>
            <p>{handoff.path}</p>
            <MarkdownContent content={handoff.content || "Empty handoff file"} />
          </article>
        ))}
        {data.handoffs.length === 0 ? <EmptyState title="No handoff logs" body="Expected .skillgrid/tasks/context_<change-id>.md files." /> : null}
      </div>
      <div className="panel">
        <h2>Checkpoints</h2>
        <CheckpointList checkpoints={data.checkpoints} />
        <h2>Agent Timeline</h2>
        <Timeline events={data.events} />
      </div>
      </section>
    </section>
  );
}

function AgentSessionCard({ session }: { session: AgentSessionSnapshot }) {
  const primary = session.primaryAgent;
  const statusLabel = primary
    ? `${primary.agent} · ${formatAgentStatus(primary.status)}${primary.threadName ? ` · ${primary.threadName}` : ""}`
    : session.metadataStatus ?? "No agent activity";

  return (
    <article className={`agent-session-card ${session.unseen ? "unseen" : ""}`}>
      <div className="agent-session-header">
        <strong>{session.name}</strong>
        {session.unseen ? <span className="unseen-badge">unseen</span> : null}
      </div>
      <p className={`agent-status-line tone-${primary?.status ?? session.metadataTone ?? "neutral"}`}>{statusLabel}</p>
      {session.metadataProgress ? <p className="meta-line">Progress: {session.metadataProgress}</p> : null}
      <p className="meta-line">
        {[session.branch ? `branch ${session.branch}${session.dirty ? " *" : ""}` : undefined, session.dir || undefined]
          .filter(Boolean)
          .join(" | ")}
      </p>
      {session.ports.length > 0 ? <p className="meta-line">Ports: {session.ports.join(", ")}</p> : null}
      {session.agents.length > 1 ? (
        <ul className="agent-thread-list">
          {session.agents.map((agent) => (
            <li key={`${agent.agent}:${agent.threadId ?? agent.ts}`}>
              {agent.agent} · {formatAgentStatus(agent.status)}
              {agent.threadName ? ` · ${agent.threadName}` : ""}
            </li>
          ))}
        </ul>
      ) : null}
    </article>
  );
}

function formatAgentStatus(status: AgentStatusTone): string {
  return status.replace(/-/g, " ");
}

function CheckpointList({ checkpoints }: { checkpoints: DashboardData["checkpoints"] }) {
  if (checkpoints.length === 0) {
    return <div className="empty compact">No checkpoints found.</div>;
  }

  return (
    <div className="checkpoints">
      {checkpoints.map((checkpoint) => (
        <article className="checkpoint" key={checkpoint.id}>
          <strong>{checkpoint.name ?? "Checkpoint"}</strong>
          <span>{checkpoint.time ?? checkpoint.sourcePath}</span>
          <p>{checkpoint.evidence ?? checkpoint.raw}</p>
          <small>
            {[checkpoint.changeId ? `change ${checkpoint.changeId}` : undefined, checkpoint.branch ? `branch ${checkpoint.branch}` : undefined, checkpoint.sha ? `sha ${checkpoint.sha}` : undefined]
              .filter(Boolean)
              .join(" | ")}
          </small>
        </article>
      ))}
    </div>
  );
}

function ToolsView({ tools }: { tools: ToolStatus[] }) {
  return (
    <section className="tools-grid">
      {tools.map((tool) => (
        <article className="tool-card" key={tool.id}>
          <div>
            <span className={`status-dot ${tool.healthy ? "healthy" : "offline"}`} />
            <h2>{tool.name}</h2>
            <p>{tool.healthy ? "Running locally" : "Not detected locally"}</p>
          </div>
          <a href={tool.url} target="_blank" rel="noreferrer" className="primary-link">
            Open in new tab
          </a>
          <div className="command">
            <span>Start command</span>
            <code>{tool.startCommand}</code>
          </div>
        </article>
      ))}
    </section>
  );
}

function Timeline({ events }: { events: DashboardEvent[] }) {
  if (events.length === 0) {
    return <div className="empty compact">No events found.</div>;
  }

  return (
    <div className="timeline">
      {events.slice(0, 30).map((event) => (
        <article className="event" key={event.id}>
          <strong>{event.status ?? event.phase ?? event.node ?? "Event"}</strong>
          <span>{event.time ?? event.sourcePath}</span>
          <p>{event.summary ?? event.blocker ?? "No summary"}</p>
        </article>
      ))}
    </div>
  );
}

function MarkdownContent({ content }: { content: string }) {
  return (
    <div className="markdown-body">
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="stat">
      <strong>{value}</strong>
      <span>{label}</span>
    </div>
  );
}

function Tab({ active, onClick, children }: { active: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button className={active ? "active" : ""} onClick={onClick}>
      {children}
    </button>
  );
}

function LaneHeader({ title, count }: { title: string; count: number }) {
  return (
    <header className="lane-header">
      <h2>{title}</h2>
      <span>{count}</span>
    </header>
  );
}

function EmptyLane() {
  return <div className="empty compact">No cards</div>;
}

function EmptyState({ title, body }: { title: string; body: string }) {
  return (
    <div className="empty">
      <h2>{title}</h2>
      <p>{body}</p>
    </div>
  );
}

function CardMeta({ items }: { items: Array<string | undefined> }) {
  return (
    <span className="meta">
      {items.filter(Boolean).map((item) => (
        <span key={item}>{item}</span>
      ))}
    </span>
  );
}

function LinkRow({ previewUrl, externalUrl }: { previewUrl?: string; externalUrl?: string }) {
  if (!previewUrl && !externalUrl) {
    return null;
  }
  return (
    <span className="links">
      {previewUrl ? <a href={previewUrl} target="_blank" rel="noreferrer">Preview</a> : null}
      {externalUrl ? <a href={externalUrl} target="_blank" rel="noreferrer">External</a> : null}
    </span>
  );
}

function groupBy<T>(items: T[], getKey: (item: T) => string): Record<string, T[]> {
  return items.reduce<Record<string, T[]>>((groups, item) => {
    const key = getKey(item);
    groups[key] = groups[key] ?? [];
    groups[key].push(item);
    return groups;
  }, {});
}

function formatTasks(completed: number, total: number): string | undefined {
  if (total === 0) {
    return undefined;
  }
  return `${completed}/${total} tasks`;
}

function formatTaskRatio(completed: number, total: number): string {
  return `${completed}/${total}`;
}

function taskProgress(completed: number, total: number): number {
  if (total === 0) {
    return 0;
  }
  return Math.round((completed / total) * 100);
}

function statusClass(status: string): string {
  const normalized = status.toLowerCase().replace(/\s+/g, "");
  if (normalized.includes("done")) return "done";
  if (normalized.includes("progress")) return "progress";
  if (normalized.includes("todo")) return "todo";
  if (normalized.includes("draft")) return "draft";
  return "";
}

function readIssueIdFromPath(): string | null {
  const match = window.location.pathname.match(/^\/issue\/(.+)$/);
  return match ? decodeURIComponent(match[1]) : null;
}

