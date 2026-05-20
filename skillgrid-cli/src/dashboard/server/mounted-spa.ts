import { createReadStream } from "node:fs";
import path from "node:path";
import type { ServerResponse } from "node:http";
import { isInside, pathExists } from "./fs-utils.js";
import { mimeType } from "./mime.js";

/** Serve a Vite-built SPA under `mount` (e.g. /gitnexus, /truecourse). */
export async function sendMountedSpa(
  clientRoot: string,
  mount: string,
  pathname: string,
  response: ServerResponse,
  options?: { missingMessage?: string }
): Promise<void> {
  let rel =
    pathname === mount || pathname === `${mount}/`
      ? "/index.html"
      : pathname.startsWith(`${mount}/`)
        ? pathname.slice(mount.length)
        : "/index.html";
  if (!rel.startsWith("/")) rel = `/${rel}`;
  const filePath = path.resolve(clientRoot, `.${decodeURIComponent(rel)}`);
  const fallback = path.join(clientRoot, "index.html");
  const target =
    (isInside(clientRoot, filePath) || filePath === clientRoot) && (await pathExists(filePath)) ? filePath : fallback;

  if (!(await pathExists(target))) {
    response.writeHead(404, { "content-type": "text/plain; charset=utf-8" });
    response.end(options?.missingMessage ?? "SPA asset not found.");
    return;
  }

  const cache =
    pathname.startsWith(`${mount}/assets/`) && path.extname(target) !== ".html"
      ? "public, max-age=86400"
      : "no-store";
  response.writeHead(200, {
    "content-type": mimeType(target),
    "cache-control": cache
  });
  createReadStream(target).pipe(response);
}
