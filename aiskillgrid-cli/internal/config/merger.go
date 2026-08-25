package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type ChangeAction string

const (
	ActionAdd    ChangeAction = "add"
	ActionUpdate ChangeAction = "update"
	ActionDelete ChangeAction = "delete"
)

type Change struct {
	Path   string
	Action ChangeAction
	Key    string
	Value  string
}

type Plan struct {
	Changes []Change
}

func MergeMCP(configPath string, servers map[string]*McpServer, dryRun bool) (*Plan, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	plan := &Plan{}

	if dryRun {
		for name := range servers {
			if bytes.Contains(data, []byte(`"`+name+`"`)) {
				plan.Changes = append(plan.Changes, Change{
					Path: configPath, Action: ActionUpdate, Key: name,
					Value: formatServer(name, servers[name]),
				})
			} else {
				plan.Changes = append(plan.Changes, Change{
					Path: configPath, Action: ActionAdd, Key: name,
					Value: formatServer(name, servers[name]),
				})
			}
		}
		return plan, nil
	}

	updated := string(data)
	for name, srv := range servers {
		entry := formatServer(name, srv)
		if idx := bytes.Index(data, []byte(`"`+name+`"`)); idx >= 0 {
			start := bytes.LastIndex(data[:idx], []byte("{"))
			end := bytes.Index(data[idx:], []byte("}"))
			if start >= 0 && end >= 0 {
				oldBlock := data[start : idx+end+1]
				updated = strings.Replace(updated, string(oldBlock), entry, 1)
				plan.Changes = append(plan.Changes, Change{Path: configPath, Action: ActionUpdate, Key: name, Value: entry})
				continue
			}
		}
		updated = strings.Replace(updated, `"mcp": {`, `"mcp": {\n    `+entry+",", 1)
		plan.Changes = append(plan.Changes, Change{Path: configPath, Action: ActionAdd, Key: name, Value: entry})
	}

	if err := os.WriteFile(configPath, []byte(updated), 0644); err != nil {
		return nil, fmt.Errorf("write config: %w", err)
	}
	return plan, nil
}

func formatServer(name string, srv *McpServer) string {
	b, _ := json.MarshalIndent(srv, "    ", "  ")
	return `"` + name + `": ` + string(b)
}
