// SkillGrid TUI plugin source. Built with tsup + esbuild-plugin-solid into
// ../skillgrid-logo.js (committed); solid-js, @opentui/*, and @opencode/* stay
// external because the OpenCode host provides them. OpenCode never transpiles
// raw TSX, so this pre-built bundle is the only artifact the installer ships
// (mirrors Gentle-AI PR #5409 / issue #5364).
import { useTerminalDimensions } from "@opentui/solid";
import { createMemo, createRoot } from "solid-js";

const id = "skillgrid-logo";

// OpenCode tokyonight secondary / darkPurple fallback when theme ctx is absent
const TOKYO_PURPLE = "#9854f1";

// Compact half-block "SKILLGRID" wordmark — the v2 home logo position is a
// hardcoded host component (not slot-reachable), so this renders in the
// home.footer slot. Kept short (4 lines, <= 42 cols) and narrow so it does not
// wrap or distort on typical terminal widths; falls back to a one-line mark on
// small terminals.
const skillgridArt = [
  "███ █ █ ███   █   █ ███ ███ ███ ███ ",
  "█   ██    █   █   █   █ ██    █   █ ",
  "  █   █   █   █   █  ██ ██    █   █ ",
  "███ ███ ███ ███ ███ ███ ███ ███ ███ ",
];

const compactArt = ["✦ SkillGrid ✦"];

// Prefer the live theme color the host passes; fall back to the tokyonight
// purple hex string so an unexpected theme object can never break rendering.
const resolveLogoColor = (theme: any): string => {
  const current = theme?.current ?? theme;
  const candidate = current?.secondary ?? current?.text?.base ?? current?.accent ?? current?.magenta;
  return typeof candidate === "string" && candidate.length > 0 ? candidate : TOKYO_PURPLE;
};

const Logo = (props: { theme?: any }) => {
  const dim = useTerminalDimensions();
  const lines = createMemo(() => {
    const term = dim();
    // skillgridArt lines are ~32 cols; need >= 38 to avoid wrapping, >= 5 rows
    // tall. Otherwise fall back to the one-line mark.
    const widest = skillgridArt.reduce((m, l) => Math.max(m, l.length), 0);
    return term.height >= 5 && term.width >= widest + 4 ? skillgridArt : compactArt;
  });

  return (
    <box flexDirection="column" alignItems="center">
      {lines().map((line) => (
        <text fg={resolveLogoColor(props.theme)}>{line}</text>
      ))}
    </box>
  );
};

const logoFor = (api: any, ctx: any) => (
  <Logo theme={ctx?.theme?.current || ctx?.theme || api?.theme} />
);

// v2: prepend the home footer slot (home_logo was removed in v2). v1: also
// register home_logo so a v1 host still renders it.
const initialize = (api: any) => {
  if (api.ui && typeof api.ui.slot === "function") {
    api.ui.slot({
      prepend: "home.footer",
      render: (ctx: any) => logoFor(api, ctx),
    });
  }
  if (api.slots && typeof api.slots.register === "function") {
    api.slots.register({
      order: 100,
      slots: {
        home_logo: (ctx: any) => logoFor(api, ctx),
      },
    });
  }
};

// v2 validates the default export as { id, setup } and ALSO invokes the tui
// hook for TUI plugins; expose both. setup is the server-side entry.
const tui: any = async (api: any) => {
  createRoot(() => initialize(api));
};

const setup = (context: any) => createRoot(() => initialize(context));

const definition = {
  id,
  setup,
  tui,
};

export default definition;
