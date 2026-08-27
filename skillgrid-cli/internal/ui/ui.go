package ui

// ui provides minimal TTY color and input helpers without external deps.

import (
	"bufio"
	"io/fs"
	"os"
)

var ttyColor bool

func init() {
	if fi, err := os.Stdout.Stat(); err == nil && fi.Mode()&fs.ModeDevice != 0 {
		ttyColor = true
	}
	if os.Getenv("NO_COLOR") != "" {
		ttyColor = false
	}
}

// C wraps s with an ANSI color code (open) + reset when a TTY is available.
func C(open, s string) string {
	if !ttyColor {
		return s
	}
	return open + s + "\x1b[0m"
}

func Bold(s string) string      { return C("\x1b[1m", s) }
func Green(s string) string     { return C("\x1b[32m", s) }
func Yellow(s string) string    { return C("\x1b[33m", s) }
func Red(s string) string       { return C("\x1b[31m", s) }
func Cyan(s string) string      { return C("\x1b[36m", s) }
func Dim(s string) string       { return C("\x1b[2m", s) }
func Magenta(s string) string   { return C("\x1b[35m", s) }
func Underline(s string) string { return C("\x1b[4m", s) }

// Ask reads a line from stdin, trimmed.
func Ask() string {
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	return trimLine(line)
}

func trimLine(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
