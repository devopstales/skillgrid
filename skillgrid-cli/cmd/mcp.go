package main

import (
	"fmt"
	"os"

	"skillgrid-cli/internal/mnemonic/mcp"
)

func runMCP() {
	if err := mcp.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
