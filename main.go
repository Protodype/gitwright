package main

import (
	"flag"
	"log"
	"path/filepath"
	"time"
)

func main() {
	configPath := flag.String("config", "/etc/gitwright/config.yml", "path to YAML config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := prepareDataDir(cfg.DataDir); err != nil {
		log.Fatalf("data_dir: %v", err)
	}

	log.Printf("cloning %s into %s", cfg.RepoURL, cfg.DataDir)
	repo, err := clone(cfg.RepoURL, cfg.DataDir)
	if err != nil {
		log.Fatalf("clone: %v", err)
	}

	watchDir := filepath.Join(cfg.DataDir, cfg.WatchPath)
	checkAndDeploy(watchDir, cfg.StackName)

	for range time.Tick(cfg.PollInterval) {
		touched, err := repo.update(cfg.WatchPath)
		if err != nil {
			log.Printf("update: %v", err)
			continue
		}
		if touched {
			checkAndDeploy(watchDir, cfg.StackName)
		}
	}
}
