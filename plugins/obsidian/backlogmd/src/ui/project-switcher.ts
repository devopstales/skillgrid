import { ProjectConfig } from "../application/ports";

export function renderProjectSwitcher(
  container: HTMLElement,
  projects: ProjectConfig[],
  selected: number,
  onChange: (i: number) => void,
): void {
  const sel = container.createEl("select", { cls: "backlogmd-project-switcher" });
  projects.forEach((p, i) => {
    const opt = sel.createEl("option", { text: p.name, value: String(i) });
    if (i === selected) opt.selected = true;
  });
  sel.addEventListener("change", () => onChange(Number(sel.value)));
}
