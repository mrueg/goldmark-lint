package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitChangedFiles runs `git diff --name-only <ref>` from cwd and returns the
// changed files as absolute paths.  It also includes staged changes via
// `git diff --name-only --cached <ref>`.
func gitChangedFiles(ref, cwd string) ([]string, error) {
	// Resolve the git repository root so that relative paths from git can be
	// turned into absolute paths.
	rootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	rootCmd.Dir = cwd
	rootOut, err := rootCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-parse --show-toplevel: %w", err)
	}
	gitRoot := strings.TrimSpace(string(rootOut))

	seen := make(map[string]bool)
	var files []string

	for _, args := range [][]string{
		{"diff", "--name-only", ref},
		{"diff", "--name-only", "--cached", ref},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = cwd
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line == "" {
				continue
			}
			abs := filepath.Join(gitRoot, filepath.FromSlash(line))
			if !seen[abs] {
				seen[abs] = true
				files = append(files, abs)
			}
		}
	}

	return files, nil
}
