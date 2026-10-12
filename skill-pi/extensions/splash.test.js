"use strict";

const test = require("node:test");
const assert = require("node:assert");

const factory = require("./splash.js");
const t = factory.__test;

// The figlet "SKILLGRID" block art — must match the opencode plugin verbatim.
const EXPECTED_ROSE_ART = [
  " ███████╗██╗  ██╗██╗██╗     ██╗     ██████╗  ██████╗ ██╗██████╗  ",
  " ██╔════╝██║  ██║██║██║     ██║    ██╔════╝  ██╔══██╗██║██╔══██╗ ",
  " ███████╗███████║██║██║     ██║    ██║   ███╗██████╔╝██║██║  ██║ ",
  " ╚════██║██╔══██║██║██║     ██║    ██║    ██║██╔══██╗██║██║  ██║ ",
  " ███████║██║  ██║██║███████╗███████╗╚██████╔╝██║  ██║██║██████╔╝ ",
  " ╚══════╝╚═╝  ╚═╝╚═╝╚══════╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝╚═════╝  ",
];

const EXPECTED_COMPACT = ["✦ SkillGrid ✦"];
// #9854f1 -> rgb(152, 84, 241)
const EXPECTED_FG = "\x1b[38;2;152;84;241m";
const EXPECTED_RESET = "\x1b[39m";

test("exported factory is a function", () => {
  assert.strictEqual(typeof factory, "function");
});

test("brand art matches the opencode plugin figlet SKILLGRID", () => {
  assert.deepStrictEqual(t.SKILLGRID_ART, EXPECTED_ROSE_ART);
  assert.strictEqual(t.SKILLGRID_ART.length, 6);
});

test("compact fallback matches the opencode plugin", () => {
  assert.deepStrictEqual(t.COMPACT_ART, EXPECTED_COMPACT);
});

test("brand color is the tokyonight secondary hex #9854f1", () => {
  assert.strictEqual(t.BRAND_HEX, "#9854f1");
});

test("brand fg is the truecolor SGR escape for rgb(152,84,241)", () => {
  assert.strictEqual(t.BRAND_FG, EXPECTED_FG);
  assert.strictEqual(t.BRAND_RESET, EXPECTED_RESET);
});

test("brand() wraps text in the brand fg and resets", () => {
  assert.strictEqual(t.brand("X"), EXPECTED_FG + "X" + EXPECTED_RESET);
});

test("full-art threshold matches the opencode plugin (height>=12, width>=64)", () => {
  assert.strictEqual(t.FULL_ART_MIN_HEIGHT, t.SKILLGRID_ART.length + 6);
  assert.strictEqual(t.FULL_ART_MIN_HEIGHT, 12);
  assert.strictEqual(t.FULL_ART_MIN_WIDTH, 64);
});

test("pickArt returns full art at/above the width threshold", () => {
  assert.deepStrictEqual(t.pickArt(64), t.SKILLGRID_ART);
  assert.deepStrictEqual(t.pickArt(100), t.SKILLGRID_ART);
});

test("pickArt returns compact below the width threshold", () => {
  assert.deepStrictEqual(t.pickArt(63), t.COMPACT_ART);
  assert.deepStrictEqual(t.pickArt(0), t.COMPACT_ART);
});

test("splashLines colors every line in the brand purple", () => {
  const wide = t.splashLines(100);
  assert.strictEqual(wide.length, 6);
  for (let i = 0; i < wide.length; i++) {
    assert.strictEqual(wide[i], EXPECTED_FG + EXPECTED_ROSE_ART[i] + EXPECTED_RESET);
  }
  const narrow = t.splashLines(30);
  assert.strictEqual(narrow.length, 1);
  assert.strictEqual(narrow[0], EXPECTED_FG + "✦ SkillGrid ✦" + EXPECTED_RESET);
});

test("component renders full art for a wide header and compact for a narrow one", () => {
  const c = t.makeSplashComponent();
  const wide = c.render(100);
  assert.strictEqual(wide.length, 6);
  assert.ok(wide[0].startsWith(EXPECTED_FG));
  assert.ok(typeof c.invalidate === "function");
  c.invalidate(); // no-op must not throw
});

test("factory wires session_start -> setHeader (tui) and registers the command", () => {
  const setHeaders = [];
  const commands = {};
  const onEvents = [];
  const pi = {
    on(event, handler) {
      onEvents.push(event);
      pi._handlers = pi._handlers || {};
      pi._handlers[event] = handler;
    },
    registerCommand(name, def) {
      commands[name] = def;
    },
  };

  factory(pi);

  assert.ok(onEvents.includes("session_start"));
  assert.ok(commands["skillgrid-logo"], "registers the skillgrid-logo command");
  assert.strictEqual(typeof commands["skillgrid-logo"].handler, "function");

  // tui mode: setHeader is called with a factory returning the component
  const setHeaderCalls = [];
  const tuiCtx = {
    mode: "tui",
    ui: {
      setHeader(f) {
        setHeaderCalls.push(f);
      },
      notify() {},
    },
  };
  pi._handlers.session_start({ type: "session_start" }, tuiCtx);
  assert.strictEqual(setHeaderCalls.length, 1);
  const comp = setHeaderCalls[0]({}, {});
  assert.strictEqual(typeof comp.render, "function");
  assert.strictEqual(comp.render(100).length, 6);

  // non-tui mode: setHeader is NOT called
  setHeaderCalls.length = 0;
  pi._handlers.session_start({ type: "session_start" }, { mode: "rpc", ui: { setHeader(f) { setHeaderCalls.push(f); } } });
  assert.strictEqual(setHeaderCalls.length, 0);

  // the command restores the built-in header
  let restored = false;
  commands["skillgrid-logo"].handler("", { mode: "tui", ui: { setHeader(f) { restored = f === undefined; }, notify() {} } });
  assert.strictEqual(restored, true);
});
