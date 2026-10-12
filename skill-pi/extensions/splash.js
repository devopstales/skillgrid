"use strict";

// skill-pi startup splash — renders the SKILLGRID wordmark in the header slot.
// Ported from the opencode plugin plugins/opencode/skillgrid-logo.tsx: same figlet
// "SKILLGRID" block art and the tokyonight secondary / darkPurple brand color
// (#9854f1). pi's Theme.fg() only accepts theme tokens, so the exact brand color
// is emitted as a literal truecolor SGR escape, which is what pi-tui itself uses
// for concrete rgb colors (colors.js colorAnsi: "\x1b[38;2;r;g;bm").

const SKILLGRID_ART = [
  " ███████╗██╗  ██╗██╗██╗     ██╗     ██████╗  ██████╗ ██╗██████╗  ",
  " ██╔════╝██║  ██║██║██║     ██║    ██╔════╝  ██╔══██╗██║██╔══██╗ ",
  " ███████╗███████║██║██║     ██║    ██║   ███╗██████╔╝██║██║  ██║ ",
  " ╚════██║██╔══██║██║██║     ██║    ██║    ██║██╔══██╗██║██║  ██║ ",
  " ███████║██║  ██║██║███████╗███████╗╚██████╔╝██║  ██║██║██████╔╝ ",
  " ╚══════╝╚═╝  ╚═╝╚═╝╚══════╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝╚═════╝  ",
];

const COMPACT_ART = ["✦ SkillGrid ✦"];

// tokyonight secondary / darkPurple. #9854f1 -> rgb(152, 84, 241).
const BRAND_HEX = "#9854f1";
const BRAND_FG = "\x1b[38;2;152;84;241m";
const BRAND_RESET = "\x1b[39m";

// Mirrors the opencode threshold: full art only when the header has room.
const FULL_ART_MIN_HEIGHT = SKILLGRID_ART.length + 6;
const FULL_ART_MIN_WIDTH = 64;

function brand(text) {
  return BRAND_FG + text + BRAND_RESET;
}

// The header only knows the render width (it is top-anchored, so terminal
// height is unavailable), so full vs compact is gated on width.
function pickArt(width) {
  return width >= FULL_ART_MIN_WIDTH ? SKILLGRID_ART : COMPACT_ART;
}

// Build the header lines (each line pre-colored in the brand purple).
function splashLines(width) {
  return pickArt(width).map(brand);
}

// The component the TUI renders: render(width) -> string[], invalidate() is a
// no-op because the header is static.
function makeSplashComponent() {
  return {
    render(width) {
      return splashLines(width);
    },
    invalidate() {},
  };
}

// pi extension factory. The header is set on session_start (ui is available and
// the built-in header exists by then); non-tui modes are a no-op. A command lets
// the user restore the built-in header.
function factory(pi) {
  pi.on("session_start", (_event, ctx) => {
    if (ctx.mode === "tui") {
      const component = makeSplashComponent();
      ctx.ui.setHeader((_tui, _theme) => component);
    }
  });

  pi.registerCommand("skillgrid-logo", {
    description: "Restore the built-in header (removes the SKILLGRID splash)",
    handler: async (_args, ctx) => {
      ctx.ui.setHeader(undefined);
      if (ctx.mode === "tui") {
        ctx.ui.notify("Built-in header restored", "info");
      }
    },
  });
}

if (typeof module !== "undefined") {
  module.exports = factory;
  module.exports.__test = {
    SKILLGRID_ART,
    COMPACT_ART,
    BRAND_HEX,
    BRAND_FG,
    BRAND_RESET,
    FULL_ART_MIN_HEIGHT,
    FULL_ART_MIN_WIDTH,
    brand,
    pickArt,
    splashLines,
    makeSplashComponent,
    factory,
  };
}
