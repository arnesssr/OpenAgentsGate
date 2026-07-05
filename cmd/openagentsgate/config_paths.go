package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arnesssr/OpenAgentsGate/internal/config"
)

const configFlagHelp = "path to config file (default: nearest .openagentsgate/config.json, else user config)"

type configLocation struct {
	Path     string `json:"path"`
	StateDir string `json:"state_dir"`
	Source   string `json:"source"`
	Exists   bool   `json:"exists"`
	Created  bool   `json:"created,omitempty"`
}

func configFlag(fs *flag.FlagSet) *string {
	return fs.String("config", "", configFlagHelp)
}

func resolveConfigPath(explicit string, create bool) (configLocation, error) {
	loc, err := locateConfig(explicit)
	if err != nil {
		return configLocation{}, err
	}
	if loc.Exists || !create {
		return loc, nil
	}
	if loc.Source == "explicit" {
		return configLocation{}, fmt.Errorf("%s does not exist; run openagentsgate init -config %s", loc.Path, loc.Path)
	}
	if err := config.WriteDefault(loc.Path, loc.StateDir, false); err != nil {
		return configLocation{}, err
	}
	loc.Exists = true
	loc.Created = true
	return loc, nil
}

func locateConfig(explicit string) (configLocation, error) {
	if strings.TrimSpace(explicit) != "" {
		path, err := normalizePath(explicit)
		if err != nil {
			return configLocation{}, err
		}
		stateDir, err := userStateDir()
		if err != nil {
			return configLocation{}, err
		}
		return configLocation{
			Path:     path,
			StateDir: stateDir,
			Source:   "explicit",
			Exists:   fileExists(path),
		}, nil
	}

	if path, ok := findProjectConfig(); ok {
		return configLocation{
			Path:     path,
			StateDir: filepath.Join(filepath.Dir(path), "state"),
			Source:   "project",
			Exists:   true,
		}, nil
	}

	path, err := userConfigPath()
	if err != nil {
		return configLocation{}, err
	}
	stateDir, err := userStateDir()
	if err != nil {
		return configLocation{}, err
	}
	return configLocation{
		Path:     path,
		StateDir: stateDir,
		Source:   "user",
		Exists:   fileExists(path),
	}, nil
}

func initConfigPath(explicit string, project bool) (configLocation, error) {
	if strings.TrimSpace(explicit) != "" && project {
		return configLocation{}, fmt.Errorf("use either -config or -project, not both")
	}
	if strings.TrimSpace(explicit) != "" {
		path, err := normalizePath(explicit)
		if err != nil {
			return configLocation{}, err
		}
		stateDir, err := userStateDir()
		if err != nil {
			return configLocation{}, err
		}
		return configLocation{Path: path, StateDir: stateDir, Source: "explicit", Exists: fileExists(path)}, nil
	}
	if project {
		cwd, err := os.Getwd()
		if err != nil {
			return configLocation{}, err
		}
		path := filepath.Join(cwd, ".openagentsgate", "config.json")
		return configLocation{
			Path:     path,
			StateDir: filepath.Join(cwd, ".openagentsgate", "state"),
			Source:   "project",
			Exists:   fileExists(path),
		}, nil
	}
	return locateConfig("")
}

func findProjectConfig() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	dir, err := filepath.Abs(cwd)
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, ".openagentsgate", "config.json")
		if fileExists(candidate) {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func userConfigPath() (string, error) {
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "openagentsgate", "config.json"), nil
}

func userStateDir() (string, error) {
	base := strings.TrimSpace(os.Getenv("XDG_STATE_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "openagentsgate"), nil
}

func normalizePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return filepath.Abs(path)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
