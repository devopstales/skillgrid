import { promises as fs } from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { afterEach, describe, expect, it } from "vitest";
import { buildDashboardData } from "./adapters.js";

const tempRoots: string[] = [];

afterEach(async () => {
  await Promise.all(tempRoots.map((root) => fs.rm(root, { recursive: true, force: true })));
  tempRoots.length = 0;
});

describe("checkpoint-record.sh", () => {
  it("writes log, handoff section, and jsonl event", async () => {
    const root = await tempRepo();
    const script = path.join(root, ".skillgrid/scripts/checkpoint-record.sh");
    await write(
      root,
      ".skillgrid/tasks/context_demo-change.md",
      "# Handoff\n\n## Goal\n\ndemo\n\n## Last checkpoint\n\n- old\n\n## Next steps\n\n1. apply\n"
    );

    execFileSync("bash", [script, "--change", "demo-change", "--name", "before-apply", "--trigger", "before-apply", "--phase", "apply", "--evidence", "gate passed"], {
      cwd: root,
      env: process.env
    });

    const log = await fs.readFile(path.join(root, ".skillgrid/tasks/checkpoints.log"), "utf8");
    expect(log).toContain("name=before-apply");
    expect(log).toContain("trigger=before-apply");
    expect(log).toContain('evidence="gate passed"');

    const handoff = await fs.readFile(path.join(root, ".skillgrid/tasks/context_demo-change.md"), "utf8");
    expect(handoff).toContain("## Last checkpoint");
    expect(handoff).toContain("`before-apply`");
    expect(handoff).not.toContain("- old");

    const events = await fs.readFile(path.join(root, ".skillgrid/tasks/events/demo-change.jsonl"), "utf8");
    const event = JSON.parse(events.trim().split("\n").pop()!);
    expect(event.node).toBe("checkpoint");
    expect(event.trigger).toBe("before-apply");

    const data = await buildDashboardData({ repoRoot: root, dashboardOrigin: "http://127.0.0.1:1" });
    expect(data.checkpoints[0]).toMatchObject({ name: "before-apply", changeId: "demo-change" });
  });
});

async function tempRepo(): Promise<string> {
  const pkgRoot = path.resolve(import.meta.dirname, "../../..");
  const root = await fs.mkdtemp(path.join(pkgRoot, ".tmp-checkpoint-"));
  tempRoots.push(root);
  await fs.mkdir(path.join(root, ".skillgrid/scripts"), { recursive: true });
  const hubScript = path.resolve(import.meta.dirname, "../../../../.skillgrid/scripts/checkpoint-record.sh");
  await fs.copyFile(hubScript, path.join(root, ".skillgrid/scripts/checkpoint-record.sh"));
  await fs.chmod(path.join(root, ".skillgrid/scripts/checkpoint-record.sh"), 0o755);
  const emptyTemplate = path.join(root, ".git-template-empty");
  await fs.mkdir(path.join(emptyTemplate, "hooks"), { recursive: true });
  execFileSync("git", ["init", `--template=${emptyTemplate}`], { cwd: root });
  execFileSync("git", ["config", "user.email", "test@example.com"], { cwd: root });
  execFileSync("git", ["config", "user.name", "Test"], { cwd: root });
  await write(root, "README.md", "# test\n");
  execFileSync("git", ["add", "README.md"], { cwd: root });
  execFileSync("git", ["commit", "-m", "init"], { cwd: root });
  return root;
}

async function write(root: string, relative: string, content: string): Promise<void> {
  const filePath = path.join(root, relative);
  await fs.mkdir(path.dirname(filePath), { recursive: true });
  await fs.writeFile(filePath, content);
}
