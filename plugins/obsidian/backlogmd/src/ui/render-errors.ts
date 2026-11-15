import type { TaskRepository } from "../application/ports";

/** Vault-relative paths of task files the most recent `repo.list()` call had to
 *  skip because they failed to parse (C4). The data layer records them on
 *  `repo.lastErrors` (cleared at the start of every `list()`); the defensive
 *  read tolerates the property being absent so a view degrades to no card. */
export function skippedFiles(repo: TaskRepository): string[] {
  return (repo as TaskRepository & { lastErrors?: string[] }).lastErrors ?? [];
}

/** Render the skipped-files error card into `host` (C4), above the board or
 *  sections. Empties the host first so it can be re-rendered; renders nothing
 *  when there are no skipped files. */
export function renderErrorCard(host: HTMLElement, errors: string[]): void {
  host.empty();
  if (errors.length === 0) return;
  const card = host.createDiv({ cls: "backlogmd-error" });
  card.createEl("div", {
    text: `Skipped ${errors.length} unreadable task file${errors.length === 1 ? "" : "s"}`,
    cls: "backlogmd-error-title",
  });
  const ul = card.createEl("ul", { cls: "backlogmd-error-list" });
  for (const p of errors) ul.createEl("li", { text: p });
}
