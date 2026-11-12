import type { ProjectConfig, FileSystemPort, MilestoneRepository } from "../application/ports";
import type { Milestone } from "../domain/milestone";
import { parseMilestone, serializeMilestone } from "../domain/milestone";

export class FileMilestoneRepository implements MilestoneRepository {
  constructor(private fs: FileSystemPort) {}

  private dir(project: ProjectConfig): string {
    return `${project.vaultBacklogDir}/milestones`;
  }

  async list(project: ProjectConfig): Promise<Milestone[]> {
    const dir = this.dir(project);
    let files: string[] = [];
    try {
      files = await this.fs.listFiles(dir);
    } catch {
      return [];
    }
    const out: Milestone[] = [];
    for (const f of files.filter((x) => x.endsWith(".md"))) {
      const content = await this.fs.readText(`${dir}/${f}`);
      out.push(parseMilestone(`${dir}/${f}`, content).milestone);
    }
    return out;
  }

  async write(project: ProjectConfig, m: Milestone, body: string): Promise<void> {
    const filePath = m.path || `${this.dir(project)}/${m.id}.md`;
    await this.fs.writeText(filePath, serializeMilestone({ ...m, path: filePath }, body));
  }

  async create(project: ProjectConfig, m: Milestone, body: string): Promise<string> {
    const dir = this.dir(project);
    const slug = m.title
      .trim()
      .replace(/[^\p{L}\p{N}.-]+/gu, "-")
      .replace(/-{2,}/g, "-")
      .replace(/^-+|-+$/g, "")
      .slice(0, 80);
    const filePath = slug ? `${dir}/${m.id} - ${slug}.md` : `${dir}/${m.id}.md`;
    await this.fs.writeText(filePath, serializeMilestone({ ...m, path: filePath }, body));
    return filePath;
  }
}
