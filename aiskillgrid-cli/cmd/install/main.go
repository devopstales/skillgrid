package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "install", "in":
		fmt.Println("=== AI Skill Grid Installer ===")
		fmt.Println("This will add all MCP tool configurations for selected agents.\n")
		runInstall()

	case "list", "ls":
		for _, a := range Agents() {
			fmt.Printf("  [%s] %s\n", a.ID, a.Name)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`AI Skill Grid Installer - Install and configure MCP tools for AI agents

Usage:
  aiskillgrid-cli [command]

Commands:
  install   Interactive multi-select agent selection and tool installation
  list      List all available agents
`)
}
