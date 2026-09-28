package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

var composeFiles = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

func checkAndDeploy(dir, stack string) {
	if err := check(dir); err != nil {
		log.Printf("check: %v", err)
		return
	}
	for _, args := range [][]string{
		{"pull"},
		{"down", "--remove-orphans"},
		{"up", "-d", "--remove-orphans"},
	} {
		if err := compose(dir, stack, args...); err != nil {
			log.Printf("docker-compose %v: %v", args, err)
			return
		}
	}
	log.Printf("stack %s deployed", stack)
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
	return fmt.Errorf("no compose file in %s", dir)
}

func compose(dir, stack string, args ...string) error {
	cmd := exec.Command("docker-compose", append([]string{"-p", stack}, args...)...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
