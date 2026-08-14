package mcpmerge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StripJSONC removes // and /* */ comments and trailing commas for parsing.
func StripJSONC(in []byte) []byte {
	var out bytes.Buffer
	inStr := false
	escape := false
	i := 0
	for i < len(in) {
		c := in[i]
		if inStr {
			out.WriteByte(c)
			if escape {
				escape = false
			} else if c == '\\' {
				escape = true
			} else if c == '"' {
				inStr = false
			}
			i++
			continue
		}
		if c == '"' {
			inStr = true
			out.WriteByte(c)
			i++
			continue
		}
		if c == '/' && i+1 < len(in) {
			if in[i+1] == '/' {
				i += 2
				for i < len(in) && in[i] != '\n' {
					i++
				}
				continue
			}
			if in[i+1] == '*' {
				i += 2
				for i+1 < len(in) && !(in[i] == '*' && in[i+1] == '/') {
					i++
				}
				if i+1 < len(in) {
					i += 2
				}
				continue
			}
		}
		out.WriteByte(c)
		i++
	}
	return stripTrailingCommas(out.Bytes())
}

func stripTrailingCommas(in []byte) []byte {
	var out bytes.Buffer
	inStr := false
	escape := false
	for i := 0; i < len(in); i++ {
		c := in[i]
		if inStr {
			out.WriteByte(c)
			if escape {
				escape = false
			} else if c == '\\' {
				escape = true
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			out.WriteByte(c)
			continue
		}
		if c == ',' {
			j := i + 1
			for j < len(in) && (in[j] == ' ' || in[j] == '\t' || in[j] == '\n' || in[j] == '\r') {
				j++
			}
			if j < len(in) && (in[j] == '}' || in[j] == ']') {
				continue
			}
		}
		out.WriteByte(c)
	}
	return out.Bytes()
}

func ParseObject(data []byte) (map[string]any, error) {
	clean := StripJSONC(data)
	clean = bytes.TrimSpace(clean)
	if len(clean) == 0 {
		return map[string]any{}, nil
	}
	var obj map[string]any
	if err := json.Unmarshal(clean, &obj); err != nil {
		return nil, err
	}
	if obj == nil {
		obj = map[string]any{}
	}
	return obj, nil
}

func EnsureBackup(path string) error {
	bak := path + ".bak"
	if _, err := os.Stat(bak); err == nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(bak, data, 0o644)
}

// MergeMCPServers merges servers into root under mcpServersKey (e.g. "mcpServers" or "mcp").
// ownedPrefix limits which keys Skillgrid overwrites (e.g. "aiskillgrid-").
func MergeMCPServers(root map[string]any, mcpKey string, incoming map[string]any, ownedPrefix string) map[string]any {
	if root == nil {
		root = map[string]any{}
	}
	existingRaw, _ := root[mcpKey].(map[string]any)
	if existingRaw == nil {
		existingRaw = map[string]any{}
	}
	for k, v := range incoming {
		if ownedPrefix == "" || strings.HasPrefix(k, ownedPrefix) {
			existingRaw[k] = v
		}
	}
	root[mcpKey] = existingRaw
	return root
}

func WriteObject(path string, obj map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		if err := EnsureBackup(path); err != nil {
			return fmt.Errorf("backup: %w", err)
		}
	}
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func LoadOrEmpty(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	return ParseObject(data)
}

func LoadPackServers(packPath string) (map[string]any, error) {
	obj, err := LoadOrEmpty(packPath)
	if err != nil {
		return nil, err
	}
	if servers, ok := obj["mcpServers"].(map[string]any); ok {
		return servers, nil
	}
	if servers, ok := obj["mcp"].(map[string]any); ok {
		return servers, nil
	}
	return map[string]any{}, nil
}
