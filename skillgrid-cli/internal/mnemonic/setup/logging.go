// Package setup — inlined from skillgrid-cli/internal/logging.
package setup

import (
	"fmt"
	"os"
)

func logInfo(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func logInfof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func logError(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
}

func logErrorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}
