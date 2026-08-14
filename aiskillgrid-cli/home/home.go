package home

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	EnvHome   = "AISKILLGRID_HOME"
	DirName   = ".aiskillgrid"
	KeyPrefix = "aiskillgrid-"
)

type Scope string

const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
)

type Config struct {
	RepoURL       string   `yaml:"repo_url"`
	DefaultScope  string   `yaml:"default_scope"`
	DefaultAgents []string `yaml:"default_agents"`
}

type State struct {
	UpdatedAt    time.Time           `json:"updated_at"`
	Scope        string              `json:"scope"`
	ProjectDir   string              `json:"project_dir,omitempty"`
	Agents       []string            `json:"agents"`
	RepoURL      string              `json:"repo_url"`
	RepoRev      string              `json:"repo_rev,omitempty"`
	WrittenPaths map[string][]string `json:"written_paths"`
}

type Paths struct {
	Root        string
	ConfigFile  string
	StateFile   string
	ToolsDir    string
	DepsDir     string
	DepsBinDir  string
	NpmDir      string
	NpmBinDir   string
	NpmCacheDir string
	LogsDir     string
	SessionsDir string
	MemoriesDir string
}

func DefaultRepoURL() string {
	return "https://github.com/aiskillgrid/aiskillgrid.git"
}

func Root() (string, error) {
	if v := os.Getenv(EnvHome); v != "" {
		return filepath.Clean(v), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DirName), nil
}

func Resolve(root string) Paths {
	return Paths{
		Root:        root,
		ConfigFile:  filepath.Join(root, "config.yaml"),
		StateFile:   filepath.Join(root, "state.json"),
		ToolsDir:    filepath.Join(root, "tools"),
		DepsDir:     filepath.Join(root, "dependencies"),
		DepsBinDir:  filepath.Join(root, "dependencies", "bin"),
		NpmDir:      filepath.Join(root, "npm"),
		NpmBinDir:   filepath.Join(root, "npm", "bin"),
		NpmCacheDir: filepath.Join(root, "npm", "cache"),
		LogsDir:     filepath.Join(root, "logs"),
		SessionsDir: filepath.Join(root, "sessions"),
		MemoriesDir: filepath.Join(root, "memories"),
	}
}

func EnsureLayout(p Paths) error {
	dirs := []string{p.Root, p.ToolsDir, p.DepsDir, p.DepsBinDir, p.NpmDir, p.NpmBinDir, p.NpmCacheDir, p.LogsDir, p.SessionsDir, p.MemoriesDir}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
	}
	if _, err := os.Stat(p.ConfigFile); errors.Is(err, os.ErrNotExist) {
		cfg := DefaultConfig()
		if err := SaveConfig(p.ConfigFile, cfg); err != nil {
			return err
		}
	}
	return nil
}

func DefaultConfig() Config {
	return Config{
		RepoURL:      DefaultRepoURL(),
		DefaultScope: string(ScopeGlobal),
		DefaultAgents: []string{
			"kilo", "opencode", "cursor", "copilot",
		},
	}
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.RepoURL == "" {
		cfg.RepoURL = DefaultRepoURL()
	}
	if cfg.DefaultScope == "" {
		cfg.DefaultScope = string(ScopeGlobal)
	}
	return cfg, nil
}

func SaveConfig(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadState(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{WrittenPaths: map[string][]string{}}, nil
		}
		return State{}, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, err
	}
	if st.WrittenPaths == nil {
		st.WrittenPaths = map[string][]string{}
	}
	return st, nil
}

func SaveState(path string, st State) error {
	st.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func UserConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		if v := os.Getenv("APPDATA"); v != "" {
			return v, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return xdg, nil
		}
		return filepath.Join(home, ".config"), nil
	}
	return filepath.Join(home, ".config"), nil
}

func AppendLog(logsDir, message string) error {
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return err
	}
	name := time.Now().UTC().Format("2006-01-02") + ".log"
	f, err := os.OpenFile(filepath.Join(logsDir, name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s %s\n", time.Now().UTC().Format(time.RFC3339), message)
	return err
}
