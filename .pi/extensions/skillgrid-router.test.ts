import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import assert from "node:assert";
import router from "./skillgrid-router.ts";

type BeforeAgentStartEvent = { systemPromptOptions?: { appendSystemPrompt?: string } };
type BeforeAgentStartContext = { cwd?: string; agent?: { kind?: string } };
type BeforeAgentStartHandler = (event: BeforeAgentStartEvent, ctx: BeforeAgentStartContext) => void;

const here = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(here, "..", "..");
const PROJECT_CWD = REPO_ROOT;

const handlers = new Map<string, BeforeAgentStartHandler>();
const fakePi = {
  on: (name: string, handler: BeforeAgentStartHandler) => { handlers.set(name, handler); },
} as unknown as ExtensionAPI; // test stub: minimal ExtensionAPI surface

const factory = router;
factory(fakePi);

const handler = handlers.get("before_agent_start");
assert.ok(handler, "pi extension must register before_agent_start");

function eventWith(append?: string): BeforeAgentStartEvent {
  return { systemPromptOptions: { appendSystemPrompt: append } };
}
function ctxFor(cwd: string, kind: string = "main"): BeforeAgentStartContext {
  return { cwd, agent: { kind } };
}
function run(event: BeforeAgentStartEvent, ctx: BeforeAgentStartContext): void {
  handler(event, ctx);
}

function makeTempProject(): string {
  const dir = fs.mkdtempSync("/tmp/skillgrid-router-proj-");
  fs.mkdirSync(path.join(dir, ".skillgrid"), { recursive: true });
  fs.writeFileSync(path.join(dir, ".skillgrid", "config.yaml"), "project: skillgrid-test\n");
  return dir;
}

let failures = 0;
function check(name: string, fn: () => void): void {
  try {
    fn();
    console.log(`PASS ${name}`);
  } catch (e) {
    failures += 1;
    console.log(`FAIL ${name}\n${e}`);
  }
}

// TICKET-03 acceptance: main agent + project cwd -> bootstrap appended exactly once.
check("main agent in project: bootstrap appended exactly once", () => {
  const ev = eventWith("EXISTING RULE");
  run(ev, ctxFor(PROJECT_CWD, "main"));
  const appended = ev.systemPromptOptions?.appendSystemPrompt;
  assert.ok(typeof appended === "string", "appendSystemPrompt should be a string");
  assert.ok(appended.startsWith("EXISTING RULE"), "pre-existing append preserved");
  assert.ok(appended.includes("First action: read `.agents/skills/using-skillgrid/SKILL.md`"), "first-action line present");
  assert.equal(appended.split("Using skillgrid:using-skillgrid").length - 1, 1, "bootstrap present exactly once");
});

// TICKET-03 acceptance: subagent -> unchanged.
check("subagent: unchanged", () => {
  const ev = eventWith("EXISTING RULE");
  run(ev, ctxFor(PROJECT_CWD, "sub"));
  assert.equal(ev.systemPromptOptions?.appendSystemPrompt, "EXISTING RULE");
});

// TICKET-03 acceptance: non-project cwd -> unchanged.
check("non-project cwd: unchanged", () => {
  const ev = eventWith("EXISTING RULE");
  run(ev, ctxFor("/tmp", "main"));
  assert.equal(ev.systemPromptOptions?.appendSystemPrompt, "EXISTING RULE");
});

// TICKET-03 acceptance: missing asset -> static fallback.
check("missing asset: static fallback", () => {
  const prevHome = process.env.HOME;
  const emptyHome = fs.mkdtempSync("/tmp/skillgrid-router-home-");
  const tmpProject = makeTempProject();
  process.env.HOME = emptyHome;
  try {
    const ev = eventWith(undefined);
    run(ev, ctxFor(tmpProject, "main"));
    const appended = ev.systemPromptOptions?.appendSystemPrompt;
    assert.ok(typeof appended === "string", "appendSystemPrompt should be a string");
    assert.ok(appended.includes("router bootstrap fallback"), "static fallback used when asset unresolvable");
  } finally {
    process.env.HOME = prevHome;
  }
});

if (failures > 0) {
  console.log(`${failures} test(s) failed`);
  process.exit(1);
}
console.log("ALL TESTS PASS");
