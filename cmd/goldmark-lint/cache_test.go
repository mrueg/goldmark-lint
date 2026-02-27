package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrueg/goldmark-lint/lint"
)

func TestHashContent_Deterministic(t *testing.T) {
	data := []byte("# Hello\n\nWorld.\n")
	h1 := hashContent(data)
	h2 := hashContent(data)
	if h1 != h2 {
		t.Errorf("hashContent not deterministic: %q != %q", h1, h2)
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex digest, got %d chars", len(h1))
	}
}

func TestHashContent_Distinct(t *testing.T) {
	h1 := hashContent([]byte("hello"))
	h2 := hashContent([]byte("world"))
	if h1 == h2 {
		t.Error("expected different hashes for different content")
	}
}

func TestLoadCache_Missing(t *testing.T) {
	dir := t.TempDir()
	c := loadCache(dir)
	if len(c) != 0 {
		t.Errorf("expected empty cache for missing file, got %d entries", len(c))
	}
}

func TestLoadCache_Corrupt(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, cacheFileName), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	c := loadCache(dir)
	if len(c) != 0 {
		t.Errorf("expected empty cache for corrupt file, got %d entries", len(c))
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	dir := t.TempDir()
	violations := []lint.Violation{
		{Rule: "MD001", Line: 3, Column: 1, Message: "test"},
	}
	c := lintCache{
		"/some/file.md": {Hash: "abc123", Violations: violations},
	}
	if err := saveCache(dir, c); err != nil {
		t.Fatalf("saveCache error: %v", err)
	}

	loaded := loadCache(dir)
	entry, ok := loaded["/some/file.md"]
	if !ok {
		t.Fatal("expected entry for /some/file.md")
	}
	if entry.Hash != "abc123" {
		t.Errorf("hash = %q, want abc123", entry.Hash)
	}
	if len(entry.Violations) != 1 || entry.Violations[0].Rule != "MD001" {
		t.Errorf("violations = %v, want one MD001 entry", entry.Violations)
	}
}

func TestSaveCache_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	if err := saveCache(dir, make(lintCache)); err != nil {
		t.Fatalf("saveCache error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, cacheFileName)); err != nil {
		t.Errorf("expected cache file to exist: %v", err)
	}
}

func TestSaveCache_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	c := lintCache{"a.md": {Hash: "h", Violations: nil}}
	if err := saveCache(dir, c); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, cacheFileName))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Errorf("saved cache is not valid JSON: %v", err)
	}
}

// TestCLI_Cache verifies that on the second run a file is served from cache
// (the cache file is written and re-read, skipping the re-lint).
func TestCLI_Cache(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// File with a single MD041 violation (no top-level heading).
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// First run: should lint and produce violations, creating the cache.
	cmd1 := exec.Command(bin, mdFile)
	cmd1.Dir = dir
	err1 := cmd1.Run()
	var exitErr1 *exec.ExitError
	if !errors.As(err1, &exitErr1) || exitErr1.ExitCode() != 1 {
		t.Fatalf("first run: expected exit 1, got %v", err1)
	}

	// Cache file must have been created.
	cachePath := filepath.Join(dir, cacheFileName)
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("expected cache file after first run: %v", err)
	}

	// Second run on the same unchanged file: should use cache and still exit 1.
	cmd2 := exec.Command(bin, mdFile)
	cmd2.Dir = dir
	err2 := cmd2.Run()
	var exitErr2 *exec.ExitError
	if !errors.As(err2, &exitErr2) || exitErr2.ExitCode() != 1 {
		t.Fatalf("second run: expected exit 1 (from cache), got %v", err2)
	}
}

// TestCLI_CacheInvalidatedOnChange verifies that modifying a file causes
// the cache to be invalidated and the file to be re-linted.
func TestCLI_CacheInvalidatedOnChange(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// Start with a file that has a violation.
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// First run: produces a violation and caches the result.
	cmd1 := exec.Command(bin, mdFile)
	cmd1.Dir = dir
	if err := cmd1.Run(); err == nil {
		t.Fatal("first run: expected non-zero exit for file with violations")
	}

	// Fix the file (now valid Markdown).
	if err := os.WriteFile(mdFile, []byte("# Heading\n\nContent.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Second run: file changed, cache should be invalidated, no violations.
	cmd2 := exec.Command(bin, mdFile)
	cmd2.Dir = dir
	if err := cmd2.Run(); err != nil {
		t.Errorf("second run: expected exit 0 for fixed file, got: %v", err)
	}
}

// TestCLI_NoCache verifies that --no-cache prevents the cache file from being created.
func TestCLI_NoCache(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Heading\n\nContent.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--no-cache", mdFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 for valid file, got: %v", err)
	}

	cachePath := filepath.Join(dir, cacheFileName)
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Errorf("expected no cache file with --no-cache, but it exists")
	}
}

// TestCLI_ParallelMultipleFiles verifies that multiple files are processed
// correctly and output contains entries for each file with violations.
func TestCLI_ParallelMultipleFiles(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()

	// Create three files, two with violations.
	file1 := filepath.Join(dir, "a.md")
	file2 := filepath.Join(dir, "b.md")
	file3 := filepath.Join(dir, "c.md")
	if err := os.WriteFile(file1, []byte("# Heading\n\n### Skipped level\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("# Heading\n\nContent.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file3, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stderr strings.Builder
	cmd := exec.Command(bin, "--no-cache", file1, file2, file3)
	cmd.Dir = dir
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit 1 for files with violations, got %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "a.md") {
		t.Errorf("expected a.md in output, got: %s", output)
	}
	if strings.Contains(output, "b.md") {
		t.Errorf("expected no b.md in output (valid file), got: %s", output)
	}
	if !strings.Contains(output, "c.md") {
		t.Errorf("expected c.md in output, got: %s", output)
	}
}
