package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func fatal(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func main() {
	configPath := flag.String("config", "/etc/gitwright/config.yml", "path to YAML config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fatal("failed to load config: %v", err)
	}
	if err := prepareDataDir(cfg.DataDir); err != nil {
		fatal("failed to prepare data directory: %v", err)
	}

	slog.Info(fmt.Sprintf("cloning %s into %s", cfg.RepoURL, cfg.DataDir))
	repo, err := clone(cfg.RepoURL, cfg.DataDir)
	if err != nil {
		fatal("failed to clone %s: %v", cfg.RepoURL, err)
	}

	watchDir := filepath.Join(cfg.DataDir, cfg.WatchPath)
	if err := checkAndDeploy(watchDir, cfg.StackName); err != nil {
		slog.Error(fmt.Sprintf("failed to deploy stack %s: %v", cfg.StackName, err))
	} else {
		slog.Info(fmt.Sprintf("stack %s deployed", cfg.StackName))
	}

	for range time.Tick(cfg.PollInterval) {
		touched, err := repo.update(cfg.WatchPath)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to update repository: %v", err))
			continue
		}
		if !touched {
			continue
		}
		slog.Info(fmt.Sprintf("changes detected in %s", watchDir))
		if err := checkAndDeploy(watchDir, cfg.StackName); err != nil {
			slog.Error(fmt.Sprintf("failed to deploy stack %s: %v", cfg.StackName, err))
			continue
		}
		slog.Info(fmt.Sprintf("stack %s deployed", cfg.StackName))
	}
}
