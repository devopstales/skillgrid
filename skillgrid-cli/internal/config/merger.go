package config

import (
	"encoding/json"
	"fmt"
	"os"

	jsonc "github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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
	updated := string(data)
	existing := jsonc.Get(updated, "mcp").Map()

	for name, srv := range servers {
		entry := buildServerEntry(srv)

		path := "mcp." + name
		exists := jsonc.Get(updated, path).Exists()
		updated, err = sjson.Set(updated, path, entry)
		if err != nil {
			return nil, fmt.Errorf("set %s: %w", name, err)
		}
		act := ActionAdd
		if exists {
			act = ActionUpdate
		}
		b, merr := json.Marshal(entry)
		if merr != nil {
			return nil, fmt.Errorf("marshal %s: %w", name, merr)
		}
		plan.Changes = append(plan.Changes, Change{Path: configPath, Action: act, Key: name, Value: string(b)})

		if ex, ok := existing[name]; ok && ex.Exists() && jsonc.Get(ex.Raw, "type").String() != entry["type"] {
			renamed := fmt.Sprintf("%s-%s", name, jsonc.Get(ex.Raw, "type").String())
			renamedPath := "mcp." + renamed
			if !jsonc.Get(updated, renamedPath).Exists() {
				updated, err = sjson.SetRaw(updated, renamedPath, ex.Raw)
				if err != nil {
					return nil, fmt.Errorf("rename %s: %w", name, err)
				}
			}
			plan.Changes = append(plan.Changes, Change{Path: configPath, Action: ActionUpdate, Key: renamed, Value: ex.Raw})
		}
	}

	if dryRun {
		return plan, nil
	}
	if err := os.WriteFile(configPath, []byte(updated), 0644); err != nil {
		return nil, fmt.Errorf("write config: %w", err)
	}
	return plan, nil
}

func buildServerEntry(srv *McpServer) map[string]interface{} {
	entry := map[string]interface{}{"type": srv.Type, "enabled": true}
	if srv.Type == "remote" {
		entry["url"] = srv.URL
	} else if len(srv.Command) > 0 {
		cmds := make([]interface{}, len(srv.Command))
		for i, c := range srv.Command {
			cmds[i] = c
		}
		entry["command"] = cmds
	}
	return entry
}
