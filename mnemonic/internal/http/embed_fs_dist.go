//go:build ui

package http

import "embed"

// uiRoot is the path prefix inside uiDistFS. Release builds (-tags ui)
// embed the Vite output from ui/dist.
const uiRoot = "ui/dist"

//go:embed all:ui/dist
var uiDistFS embed.FS
