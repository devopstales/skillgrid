/**
 * Tokyo Night Powerline footer for skill-pi.
 *
 * Replaces pi's built-in footer with a Powerline status bar themed to the
 * tokyo-night-storm palette. Shows: working dir, git branch, model, and
 * context-window usage. Data is read live from the session on every render.
 *
 * Loaded automatically via the pi-package manifest (`pi.extensions`) when
 * skill-pi launches pi with `--extension <package-root>`.
 */

import type {
  ExtensionAPI,
  ExtensionContext,
  ReadonlyFooterDataProvider,
  Theme,
} from "@earendil-works/pi-coding-agent";
import {
  getTerminalColorMode,
  parseColor,
  styleText,
  truncateToWidth,
  type Component,
  type TUI,
} from "@earendil-works/pi-tui";

// --- Tokyo Night Storm palette (from pi-tokyo-night-storm) ---
const C = {
  blue: "#7aa2f7",
  purple: "#bb9af7",
  cyan: "#7dcfff",
  teal: "#2ac3de",
  green: "#73daca",
  red: "#f7768e",
  orange: "#ff9e64",
  yellow: "#e0af68",
  text: "#a9b1d6",
  textBright: "#c0caf5",
  muted: "#8089b3",
  dim: "#545c7e",
  bg: "#24283b",
  bg1: "#1f2335",
  bg2: "#1b1e2e",
  bg3: "#2c324a",
  bg4: "#282e44",
};

type Color = ReturnType<typeof parseColor>;

interface Segment {
  text: string;
  fg: Color;
  bg: Color;
  bold?: boolean;
  /** Powerline arrow for a segment that is not the last. */
  arrow?: string;
}

function compactNumber(n: number): string {
  if (n < 1000) return String(n);
  if (n < 1_000_000) return `${(n / 1000).toFixed(n < 10_000 ? 1 : 0)}k`;
  return `${(n / 1_000_000).toFixed(1)}M`;
}

function homeRelative(cwd: string): string {
  const home = process.env.HOME || process.env.USERPROFILE || "";
  if (!home || !cwd.startsWith(home)) return cwd;
  const rel = cwd.slice(home.length).replace(/^\/+/, "");
  return rel ? `~/${rel}` : "~";
}

/** Build the ordered list of Powerline segments from live session data. */
function collectSegments(
  ctx: ExtensionContext | undefined,
  footerData: ReadonlyFooterDataProvider,
): Segment[] {
  const segments: Segment[] = [];
  const cwd = ctx?.cwd ?? process.cwd();
  const branch = footerData?.getGitBranch();
  const model = ctx?.model;
  const usage = ctx?.getContextUsage();
  const entries = ctx?.sessionManager?.getEntryCount?.();

  segments.push({
    text: homeRelative(cwd),
    fg: parseColor(C.textBright),
    bg: parseColor(C.bg1),
  });

  if (branch) {
    segments.push({
      text: branch,
      fg: parseColor(C.cyan),
      bg: parseColor(C.bg3),
    });
  }

  if (model) {
    const label = model.name || model.id;
    segments.push({
      text: label,
      fg: parseColor(C.purple),
      bg: parseColor(C.bg4),
      bold: true,
    });
  }

  if (usage) {
    const pct =
      usage.percent === null
        ? "?"
        : `${usage.percent.toFixed(1)}%`;
    const win = usage.contextWindow ? ` / ${compactNumber(usage.contextWindow)}` : "";
    segments.push({
      text: `ctx ${pct}${win}`,
      fg: parseColor(C.text),
      bg: parseColor(C.bg3),
    });
  }

  if (typeof entries === "number") {
    segments.push({
      text: `${entries} msgs`,
      fg: parseColor(C.muted),
      bg: parseColor(C.bg1),
    });
  }

  return segments;
}

/** Render segments as Powerline blocks, truncated to the viewport width. */
function renderSegments(segments: Segment[], width: number, mode: "truecolor" | "256color"): string {
  if (segments.length === 0) return "";

  // Each non-final segment carries a trailing powerline arrow glyph.
  const arrow = "⭈";
  const cells: Segment[] = segments.map((s, i) => ({
    ...s,
    arrow: i < segments.length - 1 ? arrow : undefined,
  }));

  // Build the full colored line first, then truncate by visible width.
  let line = "";
  for (const seg of cells) {
    const body = styleText(seg.text, { fg: seg.fg, bg: seg.bg, bold: seg.bold }, mode);
    line += body;
    if (seg.arrow) {
      line += styleText(seg.arrow, { fg: seg.bg, bg: parseColor(C.bg2) }, mode);
    }
  }

  const out = truncateToWidth(line, width);
  return out;
}

function makeFooter(
  ctxRef: { current: ExtensionContext | undefined },
  footerData: ReadonlyFooterDataProvider,
): Component {
  return {
    render(width: number): string[] {
      const mode = getTerminalColorMode();
      const segments = collectSegments(ctxRef.current, footerData);
      const line = renderSegments(segments, width, mode);
      return [line];
    },
    invalidate(): void {
      // No-op: data is read live on every render.
    },
  };
}

export default function (pi: ExtensionAPI) {
  pi.on("session_start", (_event, ctx) => {
    if (ctx.mode !== "tui" || !ctx.hasUI) return;

    const ctxRef = { current: ctx };
    ctx.ui.setFooter((_tui: TUI, _theme: Theme, footerData) =>
      makeFooter(ctxRef, footerData),
    );
  });
}
