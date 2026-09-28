package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RepoURL      string        `yaml:"repo_url"`
	WatchPath    string        `yaml:"watch_path"`
	DataDir      string        `yaml:"data_dir"`
	StackName    string        `yaml:"stack_name"`
	PollInterval time.Duration `yaml:"poll_interval"`
}

func loadConfig(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	if cfg.RepoURL == "" || cfg.WatchPath == "" || cfg.DataDir == "" || cfg.StackName == "" || cfg.PollInterval <= 0 {
		return Config{}, errors.New("repo_url, watch_path, data_dir, stack_name and a positive poll_interval are required")
	}
	if !filepath.IsLocal(cfg.WatchPath) {
		return Config{}, fmt.Errorf("watch_path %q must be relative to the repository root", cfg.WatchPath)
	}
	cfg.WatchPath = filepath.ToSlash(filepath.Clean(cfg.WatchPath))
	return cfg, nil
}

// prepareDataDir verifies dir exists and is writable, then deletes its contents.
func prepareDataDir(dir string) error {
	probe, err := os.CreateTemp(dir, ".probe-*")
	if err != nil {
		return fmt.Errorf("%s is not a writable directory: %w", dir, err)
	}
	_ = probe.Close()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to list contents of %s: %w", dir, err)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("failed to remove %s: %w", filepath.Join(dir, e.Name()), err)
		}
	}
	return nil
}
