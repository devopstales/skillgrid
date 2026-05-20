import { createReadStream } from "node:fs";
import { createServer as createHttpServer, type IncomingMessage, type ServerResponse } from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { buildDashboardData } from "./adapters.js";
import { isInside, pathExists } from "./fs-utils.js";
import { sendMountedSpa } from "./mounted-spa.js";
import { mimeType } from "./mime.js";

export type DashboardServerOptions = {
  repoRoot: string;
  host: string;
  port: number;
  dev?: boolean;
  /** Built Vite output (directory containing index.html). Default: ../client next to this module. */
  clientRoot?: string;
  /** Vite `root` when `dev` is true. Required when the server runs inside a Bun-compiled bundle. */
  dashboardSrcRoot?: string;
  /** Production build of GitNexus Web (index.html + assets). Default: ../gitnexus next to this module. */
  gitnexusClientRoot?: string;
  gitnexusUrl?: string;
  /** Production build of TrueCourse dashboard client. Default: ../truecourse next to this module. */
  truecourseClientRoot?: string;
  /** TrueCourse API server (default http://127.0.0.1:3001). Bundled UI calls this origin (CORS). */
  truecourseApiUrl?: string;
  openspecUiUrl?: string;
};

export async function startDashboardServer(options: DashboardServerOptions): Promise<{ url: string; close: () => Promise<void> }> {
  const repoRoot = path.resolve(options.repoRoot);
  const compiledServerRoot = path.dirname(fileURLToPath(import.meta.url));
  const clientRoot = path.resolve(options.clientRoot ?? path.join(compiledServerRoot, "..", "client"));
  const dashboardSrcRoot = path.resolve(
    options.dashboardSrcRoot ?? path.join(compiledServerRoot, "..", "..", "src", "dashboard")
  );
  const gitnexusRoot = path.resolve(options.gitnexusClientRoot ?? path.join(compiledServerRoot, "..", "gitnexus"));
  const truecourseRoot = path.resolve(options.truecourseClientRoot ?? path.join(compiledServerRoot, "..", "truecourse"));

  if (!options.dev) {
    const indexHtml = path.join(clientRoot, "index.html");
    if (!(await pathExists(indexHtml))) {
      throw new Error(
        `Dashboard client not found at ${clientRoot}. Build it with: (cd skillgrid-cli && npm run build:dashboard)`
      );
    }
  }

  const vite = options.dev
    ? await import("vite").then((module) =>
        module.createServer({
          root: dashboardSrcRoot,
          server: { middlewareMode: true },
          appType: "spa"
        })
      )
    : undefined;

  const server = createHttpServer(async (request, response) => {
    try {
      const requestUrl = new URL(request.url ?? "/", `http://${request.headers.host ?? `${options.host}:${options.port}`}`);

      if (requestUrl.pathname === "/api/dashboard") {
        await sendJson(
          response,
          await buildDashboardData({
            repoRoot,
            dashboardOrigin: requestUrl.origin,
            gitnexusUrl: options.gitnexusUrl,
            truecourseUrl: options.truecourseApiUrl,
            openspecUiUrl: options.openspecUiUrl
          })
        );
        return;
      }

      if (requestUrl.pathname === "/gitnexus" || requestUrl.pathname.startsWith("/gitnexus/")) {
        if (!(await pathExists(path.join(gitnexusRoot, "index.html")))) {
          response.writeHead(503, { "content-type": "text/plain; charset=utf-8" });
          response.end(
            "GitNexus web bundle missing. From skillgrid-cli run: npm run build:gitnexus (Node 20+, git, network). Or build the hub with: npm run build"
          );
          return;
        }
        await sendMountedSpa(gitnexusRoot, "/gitnexus", requestUrl.pathname, response, {
          missingMessage: "GitNexus asset not found."
        });
        return;
      }

      if (requestUrl.pathname === "/truecourse" || requestUrl.pathname.startsWith("/truecourse/")) {
        if (!(await pathExists(path.join(truecourseRoot, "index.html")))) {
          response.writeHead(503, { "content-type": "text/plain; charset=utf-8" });
          response.end(
            "TrueCourse web bundle missing. From skillgrid-cli run: npm run build:truecourse (Node 20+, git, pnpm, network). Or build the hub with: npm run build"
          );
          return;
        }
        await sendMountedSpa(truecourseRoot, "/truecourse", requestUrl.pathname, response, {
          missingMessage: "TrueCourse asset not found."
        });
        return;
      }

      if (requestUrl.pathname.startsWith("/preview/")) {
        await sendPreview(repoRoot, requestUrl.pathname, response);
        return;
      }

      if (vite) {
        await new Promise<void>((resolve, reject) => {
          vite.middlewares(request, response, (error?: unknown) => {
            if (error) reject(error);
            else resolve();
          });
        });
        return;
      }

      await sendStatic(clientRoot, request, response);
    } catch (error) {
      sendError(response, error);
    }
  });

  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(options.port, options.host, () => {
      server.off("error", reject);
      resolve();
    });
  });

  const address = server.address();
  const port = typeof address === "object" && address ? address.port : options.port;
  const url = `http://${options.host}:${port}`;

  return {
    url,
    close: async () => {
      await vite?.close();
      await new Promise<void>((resolve, reject) => {
        server.close((error) => (error ? reject(error) : resolve()));
      });
    }
  };
}

async function sendJson(response: ServerResponse, value: unknown): Promise<void> {
  const body = JSON.stringify(value);
  response.writeHead(200, {
    "content-type": "application/json; charset=utf-8",
    "cache-control": "no-store"
  });
  response.end(body);
}

async function sendPreview(repoRoot: string, pathname: string, response: ServerResponse): Promise<void> {
  const encodedRelative = pathname.replace(/^\/preview\//, "");
  const relative = decodeURIComponent(encodedRelative);
  const filePath = path.resolve(repoRoot, relative);
  const previewRoot = path.resolve(repoRoot, ".skillgrid", "preview");
  const graphRoot = path.resolve(repoRoot, "graphify-out");

  if (!((isInside(previewRoot, filePath) || filePath === previewRoot) || (isInside(graphRoot, filePath) || filePath === graphRoot))) {
    response.writeHead(403);
    response.end("Preview path is outside allowed preview roots.");
    return;
  }

  if (!(await pathExists(filePath))) {
    response.writeHead(404);
    response.end("Preview file not found.");
    return;
  }

  response.writeHead(200, {
    "content-type": mimeType(filePath),
    "cache-control": "no-store"
  });
  createReadStream(filePath).pipe(response);
}

async function sendStatic(clientRoot: string, request: IncomingMessage, response: ServerResponse): Promise<void> {
  const requestUrl = new URL(request.url ?? "/", "http://localhost");
  const requestedPath = requestUrl.pathname === "/" ? "/index.html" : requestUrl.pathname;
  const filePath = path.resolve(clientRoot, `.${decodeURIComponent(requestedPath)}`);
  const fallback = path.join(clientRoot, "index.html");
  const target = (isInside(clientRoot, filePath) || filePath === clientRoot) && (await pathExists(filePath)) ? filePath : fallback;

  if (!(await pathExists(target))) {
    response.writeHead(404);
    response.end("Dashboard client build not found. Run npm run build first.");
    return;
  }

  response.writeHead(200, {
    "content-type": mimeType(target)
  });
  createReadStream(target).pipe(response);
}

function sendError(response: ServerResponse, error: unknown): void {
  const message = error instanceof Error ? error.message : "Unknown server error";
  response.writeHead(500, { "content-type": "application/json; charset=utf-8" });
  response.end(JSON.stringify({ error: message }));
}

