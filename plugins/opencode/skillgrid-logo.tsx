// @ts-nocheck
/** @jsxImportSource @opentui/solid */
import type { TuiPlugin } from "@opencode-ai/plugin/tui"
import { RGBA } from "@opentui/core"
import { useTerminalDimensions } from "@opentui/solid"
import { createMemo } from "solid-js"

const id = "skillgrid-logo"

// OpenCode tokyonight secondary / darkPurple — used when theme ctx is unavailable
const TOKYO_PURPLE = RGBA.fromHex("#9854f1")

const roseArt = [
  " ███████╗██╗  ██╗██╗██╗     ██╗     ██████╗  ██████╗ ██╗██████╗  ",
  " ██╔════╝██║  ██║██║██║     ██║    ██╔════╝  ██╔══██╗██║██╔══██╗ ",
  " ███████╗███████║██║██║     ██║    ██║   ███╗██████╔╝██║██║  ██║ ",
  " ╚════██║██╔══██║██║██║     ██║    ██║    ██║██╔══██╗██║██║  ██║ ",
  " ███████║██║  ██║██║███████╗███████╗╚██████╔╝██║  ██║██║██████╔╝ ",
  " ╚══════╝╚═╝  ╚═╝╚═╝╚══════╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝╚═════╝  ",
]

const compactArt = ["✦ SkillGrid ✦"]

const Logo = (props: { fg?: RGBA }) => {
  const dim = useTerminalDimensions()
  const lines = createMemo(() => {
    const term = dim()
    return term.height >= roseArt.length + 6 && term.width >= 64 ? roseArt : compactArt
  })
  const fg = () => props.fg ?? TOKYO_PURPLE

  return (
    <box flexDirection="column" alignItems="center">
      {lines().map((line) => (
        <text fg={fg()}>{line}</text>
      ))}
    </box>
  )
}

const tui: TuiPlugin = async (api) => {
  api.slots.register({
    id,
    order: 100,
    slots: {
      home_logo(ctx) {
        return <Logo fg={ctx.theme.current.secondary} />
      },
    },
  })
}

const plugin = { id: "skillgrid-logo", tui }
export default plugin
