// skillgrid-logo-tui.tsx
import { createComponent as _$createComponent } from "@opentui/solid";
import { memo as _$memo } from "@opentui/solid";
import { effect as _$effect } from "@opentui/solid";
import { insert as _$insert } from "@opentui/solid";
import { setProp as _$setProp } from "@opentui/solid";
import { createElement as _$createElement } from "@opentui/solid";
import { useTerminalDimensions } from "@opentui/solid";
import { createMemo, createRoot } from "solid-js";
var id = "skillgrid-logo";
var TOKYO_PURPLE = "#9854f1";
var skillgridArt = ["\u2588\u2588\u2588 \u2588 \u2588 \u2588\u2588\u2588   \u2588   \u2588 \u2588\u2588\u2588 \u2588\u2588\u2588 \u2588\u2588\u2588 \u2588\u2588\u2588 ", "\u2588   \u2588\u2588    \u2588   \u2588   \u2588   \u2588 \u2588\u2588    \u2588   \u2588 ", "  \u2588   \u2588   \u2588   \u2588   \u2588  \u2588\u2588 \u2588\u2588    \u2588   \u2588 ", "\u2588\u2588\u2588 \u2588\u2588\u2588 \u2588\u2588\u2588 \u2588\u2588\u2588 \u2588\u2588\u2588 \u2588\u2588\u2588 \u2588\u2588\u2588 \u2588\u2588\u2588 \u2588\u2588\u2588 "];
var compactArt = ["\u2726 SkillGrid \u2726"];
var resolveLogoColor = (theme) => {
  const current = theme?.current ?? theme;
  const candidate = current?.secondary ?? current?.text?.base ?? current?.accent ?? current?.magenta;
  return typeof candidate === "string" && candidate.length > 0 ? candidate : TOKYO_PURPLE;
};
var Logo = (props) => {
  const dim = useTerminalDimensions();
  const lines = createMemo(() => {
    const term = dim();
    const widest = skillgridArt.reduce((m, l) => Math.max(m, l.length), 0);
    return term.height >= 5 && term.width >= widest + 4 ? skillgridArt : compactArt;
  });
  return (() => {
    var _el$ = _$createElement("box");
    _$setProp(_el$, "flexDirection", "column");
    _$setProp(_el$, "alignItems", "center");
    _$insert(_el$, () => lines().map((line) => (() => {
      var _el$2 = _$createElement("text");
      _$insert(_el$2, line);
      _$effect((_$p) => _$setProp(_el$2, "fg", resolveLogoColor(props.theme), _$p));
      return _el$2;
    })()));
    return _el$;
  })();
};
var logoFor = (api, ctx) => _$createComponent(Logo, {
  get theme() {
    return ctx?.theme?.current || ctx?.theme || api?.theme;
  }
});
var initialize = (api) => {
  if (api.ui && typeof api.ui.slot === "function") {
    api.ui.slot({
      prepend: "home.footer",
      render: (ctx) => logoFor(api, ctx)
    });
  }
  if (api.slots && typeof api.slots.register === "function") {
    api.slots.register({
      order: 100,
      slots: {
        home_logo: (ctx) => logoFor(api, ctx)
      }
    });
  }
};
var tui = async (api) => {
  createRoot(() => initialize(api));
};
var setup = (context) => createRoot(() => initialize(context));
var definition = {
  id,
  setup,
  tui
};
var skillgrid_logo_tui_default = definition;
export {
  skillgrid_logo_tui_default as default
};
