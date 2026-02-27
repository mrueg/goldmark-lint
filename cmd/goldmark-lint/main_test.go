package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "goldmark-lint")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, out)
	}
	return bin
}

func TestCLI_Version(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--version")
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() != 0 {
			t.Fatalf("--version exited with code %d, want 0", exitErr.ExitCode())
		}
	} else if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected version output, got nothing")
	}
}

func TestCLI_Help(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--help")
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() != 0 {
			t.Fatalf("--help exited with code %d, want 0", exitErr.ExitCode())
		}
	} else if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected help output, got nothing")
	}
}

func TestCLI_NoArgs(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit, got nil error")
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("no-args exit code = %d, want 2", exitErr.ExitCode())
	}
}

func TestCLI_FileNotFound(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "nonexistent_file.md")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit, got nil error")
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("file-not-found exit code = %d, want 2", exitErr.ExitCode())
	}
}

func TestCLI_WithViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}
	cmd := exec.Command(bin, testfile)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for file with violations, got nil error")
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("violations exit code = %d, want 1", exitErr.ExitCode())
	}
}

func TestCLI_NoViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_valid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}
	cmd := exec.Command(bin, testfile)
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 for valid file, got: %v", err)
	}
}

func TestCLI_ViolationsToStderr(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}
	cmd := exec.Command(bin, testfile)
	// Only capture stdout; violations should go to stderr
	stdout, err := cmd.Output()
	if err == nil {
		t.Error("expected non-zero exit for file with violations")
	}
	if len(stdout) != 0 {
		t.Errorf("expected no output on stdout, got: %s", stdout)
	}
}

func TestCLI_Fix(t *testing.T) {
	bin := buildBinary(t)

	// Create a temp file with fixable violations (trailing spaces, no final newline)
	tmp, err := os.CreateTemp(t.TempDir(), "test*.md")
	if err != nil {
		t.Fatal(err)
	}
	content := "# Heading\n\nContent   \nNo newline at end"
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--fix", tmp.Name())
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 after fixing all issues, got: %v", err)
	}

	fixed, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	want := "# Heading\n\nContent\nNo newline at end\n"
	if string(fixed) != want {
		t.Errorf("fixed content = %q, want %q", string(fixed), want)
	}
}

func TestCLI_Stdin_NoViolations(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "-")
	cmd.Stdin = strings.NewReader("# Heading\n\nValid content.\n")
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 for valid stdin input, got: %v", err)
	}
}

func TestCLI_Stdin_WithViolations(t *testing.T) {
	bin := buildBinary(t)
	// MD001: heading levels should only increment by one
	cmd := exec.Command(bin, "-")
	cmd.Stdin = strings.NewReader("# Heading\n\n### Skipped level\n")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for stdin with violations, got nil error")
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("stdin violations exit code = %d, want 1", exitErr.ExitCode())
	}
	if !strings.Contains(stderr.String(), "stdin:") {
		t.Errorf("expected 'stdin:' prefix in output, got: %s", stderr.String())
	}
}

func TestCLI_WarningSeverityExitZero(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// A file with an MD041 violation (no top-level heading).
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Config sets MD041 to warning severity.
	cfgContent := "config:\n  MD041: \"warning\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".markdownlint-cli2.yaml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, mdFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 when all violations are warnings, got: %v", err)
	}
}

func TestCLI_ErrorSeverityExitOne(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// A file with an MD041 violation (no top-level heading).
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Config sets MD041 to error severity (explicit).
	cfgContent := "config:\n  MD041: \"error\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".markdownlint-cli2.yaml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, mdFile)
	cmd.Dir = dir
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for error-severity violation, got nil error")
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("error severity exit code = %d, want 1", exitErr.ExitCode())
	}
}
