package main

import (
	"fmt"
	"os"

	mnemonighttp "skillgrid-cli/internal/mnemonic/http"
	"skillgrid-cli/internal/mnemonic/service"
)

func runServe(args []string) error {
	port := os.Getenv("SKILLGRID_MNEMONIC_PORT")
	if port == "" {
		port = "7438"
	}
	if len(args) > 0 && args[0] != "" {
		port = args[0]
	}

	dataDir, err := service.DefaultDataDir()
	if err != nil {
		return fmt.Errorf("data dir: %w", err)
	}

	svc := service.New(dataDir)
	addr := "127.0.0.1:" + port
	fmt.Fprintf(os.Stderr, "skillgrid-mnemonic HTTP listening on %s\n", addr)
	return mnemonighttp.StartHTTP(addr, svc)
}
