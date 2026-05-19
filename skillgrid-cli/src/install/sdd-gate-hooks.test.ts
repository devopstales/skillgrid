import { describe, expect, it } from "vitest";
import { hookContainsSddGate } from "./sdd-gate-hooks.js";
import { mkdtempSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";

describe("hookContainsSddGate", () => {
  it("detects sdd-gate marker in hook file", () => {
    const dir = mkdtempSync(join(tmpdir(), "sdd-hook-"));
    const hook = join(dir, "pre-commit");
    writeFileSync(hook, '#!/bin/bash\necho "[sdd-gate] Running gate"\n');
    expect(hookContainsSddGate(hook)).toBe(true);
  });

  it("returns false for unrelated hook", () => {
    const dir = mkdtempSync(join(tmpdir(), "sdd-hook-"));
    const hook = join(dir, "pre-commit");
    writeFileSync(hook, "#!/bin/bash\nnpm test\n");
    expect(hookContainsSddGate(hook)).toBe(false);
  });
});
