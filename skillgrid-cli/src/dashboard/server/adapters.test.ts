import { promises as fs } from "node:fs";
import os from "node:os";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { buildDashboardData } from "./adapters.js";

const tempRoots: string[] = [];

afterEach(async () => {
  await Promise.all(tempRoots.map((root) => fs.rm(root, { recursive: true, force: true })));
  tempRoots.length = 0;
});

describe("buildDashboardData", () => {
  it("returns useful empty states for sparse repos", async () => {
    const root = await tempRepo();
    const data = await buildDashboardData({ repoRoot: root, dashboardOrigin: "http://127.0.0.1:1" });

    expect(data.prds).toEqual([]);
    expect(data.workflow).toEqual([]);
    expect(data.issues).toEqual([]);
    expect(data.checkpoints).toEqual([]);
    expect(data.lanes.map((lane) => lane.id)).toEqual(["draft", "todo", "inprogress", "devdone", "done", "archived"]);
    expect(data.warnings).toContain("No PRDs found under .skillgrid/prd.");
    expect(data.warnings).toContain("No OpenSpec changes found under openspec/changes.");
    expect(data.openSessions.healthy).toBe(false);
    expect(data.openSessions.sessions).toEqual([]);
    expect(data.tools.some((tool) => tool.id === "opensessions")).toBe(true);
  });

  it("merges a PRD and OpenSpec change into one issue with change-local specs as subtasks", async () => {
    const root = await tempRepo();
    await write(
      root,
      ".skillgrid/prd/PRD01_dashboard.md",
      `# Skillgrid Dashboard
Status: Build
Change ID: dashboard-change
Preview: /preview/.skillgrid%2Fpreview%2Fdashboard.html

- [x] scaffold
- [ ] finish [HITL]
`
    );
    await write(root, "openspec/changes/dashboard-change/tasks.md", "- [x] scaffold\n- [ ] finish [AFK]\n");
    await write(root, "openspec/changes/dashboard-change/specs/ui/spec.md", "# Dashboard UI\n");
    await write(
      root,
      ".skillgrid/tasks/events/dashboard-change.jsonl",
      `{"time":"2026-05-02T10:00:00Z","changeId":"dashboard-change","phase":"Build","status":"in_progress","summary":"Started"}\n`
    );
    await write(root, ".skillgrid/tasks/context_dashboard-change.md", "Agent handoff");
    await write(
      root,
      ".skillgrid/tasks/checkpoints.log",
      '2026-05-01T13:56:00Z name=before-apply branch=main sha=abc123 dirty=yes prd=.skillgrid/prd/PRD01_dashboard.md change=dashboard-change context=.skillgrid/tasks/context_dashboard-change.md evidence="tests passed"\n'
    );
    await write(root, ".skillgrid/preview/dashboard.html", "<h1>Preview</h1>");

    const data = await buildDashboardData({ repoRoot: root, dashboardOrigin: "http://127.0.0.1:1" });

    expect(data.prds).toHaveLength(1);
    expect(data.prds[0]).toMatchObject({
      title: "Skillgrid Dashboard",
      status: "Build",
      changeId: "dashboard-change"
    });
    expect(data.workflow[0]).toMatchObject({
      id: "dashboard-change",
      phase: "Build"
    });
    expect(data.issues).toHaveLength(1);
    expect(data.issues[0]).toMatchObject({
      title: "Skillgrid Dashboard",
      source: "merged",
      changeId: "dashboard-change",
      lane: "inprogress",
      status: "Build"
    });
    expect(data.issues[0].subtasks).toHaveLength(1);
    expect(data.issues[0].subtasks[0]).toMatchObject({
      title: "Dashboard UI",
      path: "openspec/changes/dashboard-change/specs/ui/spec.md"
    });
    expect(data.events).toHaveLength(1);
    expect(data.handoffs[0].content).toBe("Agent handoff");
    expect(data.checkpoints[0]).toMatchObject({
      name: "before-apply",
      changeId: "dashboard-change",
      prd: ".skillgrid/prd/PRD01_dashboard.md",
      context: ".skillgrid/tasks/context_dashboard-change.md",
      evidence: "tests passed"
    });
    expect(data.previews[0].path).toBe(".skillgrid/preview/dashboard.html");
  });

  it("ignores blank, comment, and malformed checkpoint lines", async () => {
    const root = await tempRepo();
    await write(
      root,
      ".skillgrid/tasks/checkpoints.log",
      [
        "",
        "# audit note — not a checkpoint",
        "malformed-no-fields",
        "2026-05-03T12:00:00Z name=valid change=only-good-line"
      ].join("\n")
    );

    const data = await buildDashboardData({ repoRoot: root, dashboardOrigin: "http://127.0.0.1:1" });

    expect(data.checkpoints).toHaveLength(1);
    expect(data.checkpoints[0]).toMatchObject({ name: "valid", changeId: "only-good-line" });
  });

  it("filters checkpoints on the issue detail view by change, prd, or context path", async () => {
    const root = await tempRepo();
    await write(
      root,
      ".skillgrid/prd/PRD01_alpha.md",
      "# Alpha\nStatus: Build\nChange ID: alpha-change\n"
    );
    await write(root, "openspec/changes/alpha-change/tasks.md", "- [ ] work\n");
    await write(
      root,
      ".skillgrid/tasks/checkpoints.log",
      [
        "2026-05-01T10:00:00Z name=for-alpha change=alpha-change prd=.skillgrid/prd/PRD01_alpha.md context=.skillgrid/tasks/context_alpha-change.md",
        "2026-05-01T11:00:00Z name=for-other change=other-change"
      ].join("\n") + "\n"
    );

    const data = await buildDashboardData({ repoRoot: root, dashboardOrigin: "http://127.0.0.1:1" });
    const issue = data.issues.find((entry) => entry.changeId === "alpha-change");
    expect(issue).toBeDefined();

    const related = data.checkpoints.filter(
      (checkpoint) =>
        checkpoint.changeId === issue?.changeId ||
        checkpoint.prd === issue?.prdPath ||
        checkpoint.context === issue?.contextPath
    );
    expect(related).toHaveLength(1);
    expect(related[0].name).toBe("for-alpha");
  });

  it("creates an issue for an OpenSpec change without a PRD", async () => {
    const root = await tempRepo();
    await write(root, "openspec/changes/api-cleanup/tasks.md", "- [ ] remove old route\n");
    await write(root, "openspec/changes/api-cleanup/specs/api/spec.md", "# API Cleanup\n");

    const data = await buildDashboardData({ repoRoot: root, dashboardOrigin: "http://127.0.0.1:1" });

    expect(data.issues).toHaveLength(1);
    expect(data.issues[0]).toMatchObject({
      title: "Api Cleanup",
      source: "change",
      changeId: "api-cleanup"
    });
    expect(data.issues[0].subtasks).toHaveLength(1);
  });

  it("infers OpenSpec change links from Skillgrid PRD bodies and avoids duplicate issues", async () => {
    const root = await tempRepo();
    await write(
      root,
      ".skillgrid/config.json",
      JSON.stringify({
        prdWorkflow: {
          fallbackStatus: "draft",
          statuses: [
            { id: "draft", label: "Draft" },
            { id: "todo", label: "Todo" },
            { id: "inprogress", label: "In Progress" },
            { id: "devdone", label: "Dev Done" },
            { id: "done", label: "Done" }
          ],
          phaseStatusMap: { apply: "inprogress", validate: "devdone" }
        }
      })
    );
    await write(
      root,
      ".skillgrid/prd/PRD02_oidc_kdlogin_improvements.md",
      `# PRD02: OIDC + kdlogin Improvements

**Spec / change:** \`openspec/changes/oidc-kdlogin-improvements/\`
**Status:** inprogress
`
    );
    await write(root, "openspec/changes/oidc-kdlogin-improvements/tasks.md", "- [ ] harden oidc\n");
    await write(root, "openspec/changes/oidc-kdlogin-improvements/specs/oidc-server-hardening/spec.md", "# OIDC Server Hardening\n");

    const data = await buildDashboardData({ repoRoot: root, dashboardOrigin: "http://127.0.0.1:1" });

    expect(data.lanes.map((lane) => lane.id)).toEqual(["draft", "todo", "inprogress", "devdone", "done", "archived"]);
    expect(data.issues).toHaveLength(1);
    expect(data.issues[0]).toMatchObject({
      title: "PRD02: OIDC + kdlogin Improvements",
      source: "merged",
      changeId: "oidc-kdlogin-improvements",
      lane: "inprogress"
    });
    expect(data.issues[0].subtasks).toHaveLength(1);
  });
});

async function tempRepo(): Promise<string> {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), "skillgrid-dashboard-"));
  tempRoots.push(root);
  return root;
}

async function write(root: string, relative: string, content: string): Promise<void> {
  const filePath = path.join(root, relative);
  await fs.mkdir(path.dirname(filePath), { recursive: true });
  await fs.writeFile(filePath, content);
}
