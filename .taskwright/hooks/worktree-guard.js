"use strict";
var __create = Object.create;
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __getProtoOf = Object.getPrototypeOf;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toESM = (mod, isNodeMode, target) => (target = mod != null ? __create(__getProtoOf(mod)) : {}, __copyProps(
  // If the importer is in node compatibility mode or this is not an ESM
  // file that has been converted to a CommonJS file using a Babel-
  // compatible transform (i.e. "__esModule" has not been set), then set
  // "default" to the CommonJS "module.exports" for node compatibility.
  isNodeMode || !mod || !mod.__esModule ? __defProp(target, "default", { value: mod, enumerable: true }) : target,
  mod
));

// scripts/hooks/worktree-guard.ts
var import_child_process = require("child_process");
var fs = __toESM(require("fs"));
var path2 = __toESM(require("path"));

// src/core/worktreeGuard.ts
var path = __toESM(require("path"));
function isPrimaryTree(gitDir) {
  return !gitDir.replace(/\\/g, "/").includes("/.git/worktrees/");
}
function shouldBlockCommit(ctx) {
  if (ctx.branch === null) return { block: false };
  if (!isPrimaryTree(ctx.gitDir)) return { block: false };
  if (!ctx.dispatchedBranches.includes(ctx.branch)) return { block: false };
  return {
    block: true,
    message: `Taskwright: branch "${ctx.branch}" is dispatched to .worktrees/${ctx.branch} \u2014 commit inside that worktree, not the primary tree. (To bypass this one commit: git commit --no-verify)`
  };
}
function collectDispatchedBranches(primaryRoot, deps) {
  return deps.listDirs(path.join(primaryRoot, ".worktrees"));
}

// scripts/hooks/worktree-guard.ts
function git(args) {
  return (0, import_child_process.execFileSync)("git", args, { encoding: "utf8" }).trim();
}
function listDirs(dir) {
  try {
    return fs.readdirSync(dir, { withFileTypes: true }).filter((e) => e.isDirectory()).map((e) => e.name);
  } catch {
    return [];
  }
}
function main() {
  let gitDir;
  let commonDir;
  let branch;
  try {
    gitDir = path2.resolve(git(["rev-parse", "--git-dir"]));
    commonDir = path2.resolve(git(["rev-parse", "--git-common-dir"]));
    try {
      branch = git(["symbolic-ref", "--short", "HEAD"]);
    } catch {
      branch = null;
    }
  } catch {
    process.exit(0);
  }
  const primaryRoot = path2.dirname(commonDir);
  const dispatchedBranches = collectDispatchedBranches(primaryRoot, { listDirs });
  const decision = shouldBlockCommit({ gitDir, branch, dispatchedBranches });
  if (decision.block) {
    process.stderr.write(`
${decision.message}

`);
    process.exit(1);
  }
  process.exit(0);
}
main();
//# sourceMappingURL=worktree-guard.js.map
