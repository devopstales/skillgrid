// Package logging provides minimal structured logging for skillgrid-cli.
package logging

import (
	"fmt"
	"os"
)

// Info logs an informational message to stderr.
func Info(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

// Infof logs a formatted informational message to stderr.
func Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Error logs an error message to stderr.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
}

// Errorf logs a formatted error message to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}
