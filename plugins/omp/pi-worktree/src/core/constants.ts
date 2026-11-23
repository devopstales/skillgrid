export const REF_BASE = "refs/skillgrid";
export const AGENT_BRANCH_RE = /^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$/;
export const IGNORED_DIRS = new Set([
  "node_modules", ".venv", "venv", "env", ".env", "dist", "build",
  ".pytest_cache", ".mypy_cache", ".cache", ".tox", "__pycache__", ".worktree",
]);
export const MAX_UNTRACKED_FILE_SIZE = 10 * 1024 * 1024;
export const MAX_UNTRACKED_DIR_FILES = 200;
export const MAX_PER_SESSION = 50;
export const isSafeRefId = (id: string) => /^[\w-]+$/.test(id);
