package main

import (
	"aiskillgrid-cli/internal/config"
	"aiskillgrid-cli/internal/engram"
	"aiskillgrid-cli/internal/logging"
	"aiskillgrid-cli/internal/mcp"
	"fmt"
	"os"
	"path/filepath"
)

func runInstall(skipClone bool, syncRepo string, dryRun bool) {
	baseDir := mustExpandHome("~/.aiskillgrid")
	if err := logging.Init(baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logging: %v\n", err)
		return
	}
	logging.Info("install started")

	if !skipClone {
		logging.Info("cloning repo")
	}

	if err := engram.InstallBinary(baseDir); err != nil {
		logging.Warn("engram install failed: " + err.Error())
	}

	_, err := config.LoadToolsYAML(filepath.Join(baseDir, "config.d", "tools.yaml"))
	if err != nil {
		logging.Error("load tools config failed: " + err.Error())
		return
	}

	mcpServers, err := mcp.LoadRegistry(filepath.Join(baseDir, "config.d"))
	if err != nil {
		logging.Error("load mcp config failed: " + err.Error())
		return
	}

	agents := []string{"kilo", "opencode"}
	for _, agent := range agents {
		configPath := agentConfigPath(agent)
		plan, err := config.MergeMCP(configPath, mcpServers, dryRun)
		if err != nil {
			logging.Warn("merge failed for " + agent + ": " + err.Error())
			continue
		}
		for _, ch := range plan.Changes {
			logging.Info(fmt.Sprintf("%s %s=%s", ch.Action, ch.Key, ch.Value))
		}
	}

	logging.Info("writing PATH instructions")

	logging.Info("install finished")
}

func runSyncRepo(extraPath string) {
	logging.Info("sync-repo started")
	logging.Info("sync-repo finished")
}

func mustExpandHome(p string) string {
	if len(p) > 0 && p[0] == '~' {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}

func agentConfigPath(agent string) string {
	home := mustExpandHome("~")
	switch agent {
	case "kilo":
		return filepath.Join(home, ".config", "kilo", "kilo.jsonc")
	case "opencode":
		return filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	default:
		return ""
	}
}
