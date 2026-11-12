import { PluginContext } from "../compose";

export function renderProjectSwitcher(container: HTMLElement, ctx: PluginContext, onChange: (i: number) => void): void {
  const sel = container.createEl("select");
  ctx.projects.forEach((p, i) => {
    const opt = sel.createEl("option", { text: p.name, value: String(i) });
    if (i === 0) opt.selected = true;
  });
  sel.addEventListener("change", () => onChange(Number(sel.value)));
}