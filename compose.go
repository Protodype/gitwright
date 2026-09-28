package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var composeFiles = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

func checkAndDeploy(dir, stack string) error {
	if err := check(dir); err != nil {
		return fmt.Errorf("repository check failed: %w", err)
	}
	for _, args := range [][]string{
		{"pull"},
		{"down", "--remove-orphans"},
		{"up", "-d", "--remove-orphans"},
	} {
		if err := compose(dir, stack, args...); err != nil {
			return fmt.Errorf("docker-compose %s failed: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}

func check(dir string) error {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("watch path %s does not exist", dir)
	}
	for _, name := range composeFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no compose file found in %s", dir)
}

func compose(dir, stack string, args ...string) error {
	cmd := exec.Command("docker-compose", append([]string{"-p", stack}, args...)...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("exit code %d: %s", exitErr.ExitCode(), msg)
		}
		return fmt.Errorf("exit code %d", exitErr.ExitCode())
	}
	if err != nil {
		return fmt.Errorf("failed to run docker-compose: %w", err)
	}
	return nil
}
