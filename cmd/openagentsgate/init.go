package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/arnesssr/OpenAgentsGate/internal/config"
)

func initCommand(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	configPath := configFlag(fs)
	project := fs.Bool("project", false, "write .openagentsgate/config.json in the current project")
	force := fs.Bool("force", false, "overwrite an existing config")
	_ = fs.Parse(args)

	loc, err := initConfigPath(*configPath, *project)
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	if loc.Exists && !*force {
		log.Fatalf("init: %s already exists; use -force to overwrite", loc.Path)
	}
	if err := config.WriteDefault(loc.Path, loc.StateDir, *force); err != nil {
		log.Fatalf("init: %v", err)
	}
	if *project {
		writeProjectIgnore(filepath.Dir(loc.Path))
	}
	loc.Exists = true
	loc.Created = true
	printJSON(loc)
}

func configCommand(args []string) {
	if len(args) < 1 {
		configUsage()
		os.Exit(exitUsage)
	}
	switch args[0] {
	case "path":
		fs := flag.NewFlagSet("config path", flag.ExitOnError)
		configPath := configFlag(fs)
		_ = fs.Parse(args[1:])
		loc, err := resolveConfigPath(*configPath, false)
		if err != nil {
			log.Fatalf("config: %v", err)
		}
		printJSON(loc)
	case "doctor":
		fs := flag.NewFlagSet("config doctor", flag.ExitOnError)
		configPath := configFlag(fs)
		_ = fs.Parse(args[1:])
		loc, err := resolveConfigPath(*configPath, true)
		if err != nil {
			log.Fatalf("config: %v", err)
		}
		if _, err := config.Load(loc.Path); err != nil {
			log.Fatalf("config: %v", err)
		}
		printJSON(map[string]any{"ok": true, "config": loc})
	default:
		configUsage()
		os.Exit(exitUsage)
	}
}

func writeProjectIgnore(dir string) {
	path := filepath.Join(dir, ".gitignore")
	if fileExists(path) {
		return
	}
	data := []byte("state/\n")
	if err := os.WriteFile(path, data, 0o600); err != nil && !errors.Is(err, os.ErrExist) {
		log.Printf("init: could not write %s: %v", path, err)
	}
}
