package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitChangedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()

	gitRun := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	gitRun("init")
	gitRun("config", "user.email", "test@example.com")
	gitRun("config", "user.name", "Test")

	// Create two files and commit.
	f1 := filepath.Join(dir, "a.md")
	f2 := filepath.Join(dir, "b.md")
	if err := os.WriteFile(f1, []byte("# A\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("# B\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun("add", ".")
	gitRun("commit", "-m", "init")

	// Modify only f2 (working tree change, not staged).
	if err := os.WriteFile(f2, []byte("# B modified\n"), 0644); err != nil {
		t.Fatal(err)
	}

	changed, err := gitChangedFiles("HEAD", dir)
	if err != nil {
		t.Fatalf("gitChangedFiles: %v", err)
	}
	if len(changed) != 1 {
		t.Fatalf("expected 1 changed file, got %d: %v", len(changed), changed)
	}
	absF2, _ := filepath.Abs(f2)
	if changed[0] != absF2 {
		t.Errorf("expected changed file %s, got %s", absF2, changed[0])
	}

	// Stage f1 as changed too.
	if err := os.WriteFile(f1, []byte("# A modified\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun("add", f1)

	changed2, err := gitChangedFiles("HEAD", dir)
	if err != nil {
		t.Fatalf("gitChangedFiles: %v", err)
	}
	if len(changed2) != 2 {
		t.Fatalf("expected 2 changed files, got %d: %v", len(changed2), changed2)
	}
}

func TestGitChangedFiles_NotARepo(t *testing.T) {
	dir := t.TempDir()
	_, err := gitChangedFiles("HEAD", dir)
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}
