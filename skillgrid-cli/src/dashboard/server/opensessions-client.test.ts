import os from "node:os";
import path from "node:path";
import { describe, expect, it } from "vitest";
import {
  formatMetadataProgress,
  formatSessionSummary,
  mapOpenSessionsForRepo,
  normalizeSessionRef,
  type OpenSessionsServerState
} from "./opensessions-client.js";

describe("mapOpenSessionsForRepo", () => {
  it("marks sessions that match the repo root and sorts repo sessions first", () => {
    const repoRoot = path.join(os.tmpdir(), "skillgrid-repo");
    const state: OpenSessionsServerState = {
      type: "state",
      focusedSession: "feature",
      currentSession: "feature",
      ts: Date.now(),
      sessions: [
        {
          name: "other",
          dir: "/tmp/other",
          branch: "main",
          dirty: false,
          unseen: false,
          ports: [],
          agentState: {
            agent: "claude-code",
            session: "other",
            status: "idle",
            ts: 1
          },
          agents: []
        },
        {
          name: "feature",
          dir: repoRoot,
          branch: "feat/dashboard",
          dirty: true,
          unseen: true,
          ports: [8787],
          agentState: {
            agent: "cursor",
            session: "feature",
            status: "running",
            threadName: "sdd-apply",
            ts: 2
          },
          agents: [
            {
              agent: "cursor",
              session: "feature",
              status: "running",
              threadName: "sdd-apply",
              ts: 2
            }
          ],
          metadata: {
            status: { text: "Applying tasks", tone: "info", ts: 3 },
            progress: { current: 2, total: 5, label: "tasks", ts: 3 },
            logs: []
          }
        }
      ]
    };

    const mapped = mapOpenSessionsForRepo(state, repoRoot);
    expect(mapped).toHaveLength(2);
    expect(mapped[0]).toMatchObject({
      name: "feature",
      matchesRepo: true,
      metadataStatus: "Applying tasks",
      metadataProgress: "2/5 tasks"
    });
    expect(mapped[1]).toMatchObject({
      name: "other",
      matchesRepo: false
    });
  });

  it("matches sessions whose cwd is inside the repo root", () => {
    const repoRoot = path.join(os.tmpdir(), "skillgrid-repo-root");
    const nestedDir = path.join(repoRoot, "packages", "app");
    const state: OpenSessionsServerState = {
      type: "state",
      focusedSession: "work",
      currentSession: "work",
      ts: 1,
      sessions: [
        {
          name: "work",
          dir: nestedDir,
          branch: "main",
          dirty: false,
          unseen: false,
          ports: [],
          agentState: null,
          agents: []
        }
      ]
    };

    const mapped = mapOpenSessionsForRepo(state, repoRoot);
    expect(mapped[0]?.matchesRepo).toBe(true);
  });
});

describe("normalizeSessionRef", () => {
  it("maps numeric index to session name when name is not a literal match", () => {
    const sessions = [
      { name: "kubedash", dir: "/tmp/k", branch: "main", dirty: false, unseen: false, ports: [], agentState: null, agents: [] },
      { name: "other", dir: "/tmp/o", branch: "dev", dirty: false, unseen: false, ports: [], agentState: null, agents: [] }
    ] as OpenSessionsServerState["sessions"];
    expect(normalizeSessionRef(0, sessions)).toBe("kubedash");
    expect(normalizeSessionRef("0", sessions)).toBe("kubedash");
  });

  it("keeps literal session name 0 when present", () => {
    const sessions = [
      { name: "0", dir: "/tmp", branch: "main", dirty: false, unseen: false, ports: [], agentState: null, agents: [] }
    ] as OpenSessionsServerState["sessions"];
    expect(normalizeSessionRef("0", sessions)).toBe("0");
    expect(normalizeSessionRef(0, sessions)).toBe("0");
  });
});

describe("formatSessionSummary", () => {
  it("includes branch and directory", () => {
    const summary = formatSessionSummary("work", [
      { name: "work", branch: "feat/x", dir: "/Users/me/proj", matchesRepo: true } as never
    ]);
    expect(summary).toContain("work");
    expect(summary).toContain("branch feat/x");
    expect(summary).toContain("proj");
  });
});

describe("formatMetadataProgress", () => {
  it("formats percent progress", () => {
    expect(formatMetadataProgress({ percent: 0.75, label: "deploy", ts: 1 })).toBe("75% deploy");
  });
});
