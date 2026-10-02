//go:build !ui

package http

import "embed"

// uiRoot is the path prefix inside uiDistFS. Plain go test / go build embed
// this shell so a clean checkout compiles before task ui:build.
const uiRoot = "ui/fallback"

//go:embed all:ui/fallback
var uiDistFS embed.FS
