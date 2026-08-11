package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

func TestCLI_SeverityOverride(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// A file with an MD041 violation (no top-level heading).
	mdFile := filepath.Join(dir, "warning.md")
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Config has override setting MD041 to warning severity for warning.md.
	cfgContent := "config:\n  MD041: true\noverrides:\n  - files:\n      - \"warning.md\"\n    config:\n      MD041: \"warning\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".markdownlint-cli2.yaml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, mdFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 when violation is overridden to warning, got: %v", err)
	}
}

func TestCLI_ConfigFlag(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// A file that triggers MD041 (no top-level heading).
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Config disables MD041.
	cfgPath := filepath.Join(dir, "custom-config.yaml")
	if err := os.WriteFile(cfgPath, []byte("config:\n  MD041: false\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// With --config pointing to the custom file, MD041 should be disabled, exit 0.
	cmd := exec.Command(bin, "--config", cfgPath, mdFile)
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 with MD041 disabled via --config, got: %v", err)
	}
}

func TestCLI_ConfigFlag_BadPath(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--config", "/nonexistent/config.yaml", "somefile.md")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for bad --config path, got nil error")
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("bad --config exit code = %d, want 2", exitErr.ExitCode())
	}
}

func TestCLI_Format(t *testing.T) {
	bin := buildBinary(t)
	// Input with trailing spaces (MD009) and no final newline (MD047).
	// --format should apply both fixes.
	input := "# Heading\n\nContent   \nNo newline at end"
	cmd := exec.Command(bin, "--format")
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("--format exited with error: %v", err)
	}
	want := "# Heading\n\nContent\nNo newline at end\n"
	if string(out) != want {
		t.Errorf("--format output = %q, want %q", string(out), want)
	}
}

func TestCLI_Format_NoArgs(t *testing.T) {
	bin := buildBinary(t)
	// --format alone (no globs needed) should succeed.
	cmd := exec.Command(bin, "--format")
	cmd.Stdin = strings.NewReader("# Valid\n\nContent.\n")
	if err := cmd.Run(); err != nil {
		t.Errorf("--format with valid input should exit 0, got: %v", err)
	}
}

func TestCLI_ListRules(t *testing.T) {
	bin := buildBinary(t)

	// Default: all rules enabled, no config.
	cmd := exec.Command(bin, "--list-rules")
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() != 0 {
			t.Fatalf("--list-rules exited with code %d, want 0: %s", exitErr.ExitCode(), out)
		}
	} else if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outStr := string(out)
	if !strings.Contains(outStr, "RULE") || !strings.Contains(outStr, "ALIASES") ||
		!strings.Contains(outStr, "ENABLED") || !strings.Contains(outStr, "OPTIONS") {
		t.Errorf("expected table header in --list-rules output, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "MD001") {
		t.Errorf("expected MD001 in --list-rules output, got:\n%s", outStr)
	}
	if !strings.Contains(outStr, "heading-increment") {
		t.Errorf("expected alias 'heading-increment' in --list-rules output, got:\n%s", outStr)
	}

	// With a config that disables MD001: should show enabled=false for MD001.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("config:\n  MD001: false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd2 := exec.Command(bin, "--list-rules", "--config", cfgPath)
	out2, err2 := cmd2.CombinedOutput()
	if errors.As(err2, &exitErr) {
		if exitErr.ExitCode() != 0 {
			t.Fatalf("--list-rules --config exited with code %d, want 0: %s", exitErr.ExitCode(), out2)
		}
	} else if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	outStr2 := string(out2)
	if !strings.Contains(outStr2, "false") {
		t.Errorf("expected 'false' for disabled MD001 in --list-rules output, got:\n%s", outStr2)
	}
}

func TestCLI_FailOnWarning(t *testing.T) {
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

	// Without --fail-on-warning, warnings produce exit code 0.
	cmd := exec.Command(bin, mdFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 when all violations are warnings (no --fail-on-warning), got: %v", err)
	}

	// With --fail-on-warning, warnings produce exit code 1.
	cmd2 := exec.Command(bin, "--fail-on-warning", mdFile)
	cmd2.Dir = dir
	err := cmd2.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit with --fail-on-warning and warning violations, got nil error")
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("--fail-on-warning exit code = %d, want 1", exitErr.ExitCode())
	}
}

func TestCLI_NoGlobs(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Valid\n\nContent.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Config has globs key that would normally provide input files.
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("globs:\n  - \"*.md\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Without --no-globs, config globs are used and exit 0 (valid file).
	cmd := exec.Command(bin, "--config", cfgPath)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 using config globs, got: %v", err)
	}

	// With --no-globs, config globs are ignored; no input files → exit 2.
	cmd2 := exec.Command(bin, "--config", cfgPath, "--no-globs")
	cmd2.Dir = dir
	err := cmd2.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit with --no-globs and no CLI args, got nil error")
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("--no-globs exit code = %d, want 2", exitErr.ExitCode())
	}
}

func TestCLI_Summary(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	cmd := exec.Command(bin, "--summary", testfile)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	_ = cmd.Run()
	got := stderr.String()
	if !strings.Contains(got, "Summary:") {
		t.Errorf("expected 'Summary:' in stderr, got: %s", got)
	}
	if !strings.Contains(got, "MD001:") {
		t.Errorf("expected 'MD001:' in summary output, got: %s", got)
	}
}

func TestCLI_Summary_NoViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_valid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	cmd := exec.Command(bin, "--summary", testfile)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 for valid file with --summary, got: %v", err)
	}
	// No summary printed when there are no violations.
	if strings.Contains(stderr.String(), "Summary:") {
		t.Errorf("expected no summary output for zero violations, got: %s", stderr.String())
	}
}

func TestCLI_Quiet_SuppressesSummary(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	// Without --quiet, --summary prints "Summary:" to stderr.
	cmd := exec.Command(bin, "--summary", testfile)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	_ = cmd.Run()
	if !strings.Contains(stderr.String(), "Summary:") {
		t.Fatalf("expected 'Summary:' without --quiet, got: %s", stderr.String())
	}

	// With --quiet, --summary output is suppressed.
	cmd2 := exec.Command(bin, "--quiet", "--summary", testfile)
	var stderr2 strings.Builder
	cmd2.Stderr = &stderr2
	_ = cmd2.Run()
	if strings.Contains(stderr2.String(), "Summary:") {
		t.Errorf("expected no 'Summary:' with --quiet, got: %s", stderr2.String())
	}
}

func TestCLI_Quiet_ShortFlag(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	// -q (short flag) should suppress summary just like --quiet.
	cmd := exec.Command(bin, "-q", "--summary", testfile)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	_ = cmd.Run()
	if strings.Contains(stderr.String(), "Summary:") {
		t.Errorf("expected no 'Summary:' with -q, got: %s", stderr.String())
	}
}

func TestCLI_Quiet_KeepsViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	// --quiet must not suppress violation output.
	cmd := exec.Command(bin, "--quiet", testfile)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for file with violations, got nil error")
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("--quiet exit code = %d, want 1", exitErr.ExitCode())
	}
	// Violations should still appear on stderr.
	if !strings.Contains(stderr.String(), "MD001") {
		t.Errorf("expected violation output with --quiet, got: %s", stderr.String())
	}
}

func TestCLI_Quiet_SuppressesFixDryRunDiff(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// File with trailing spaces (MD009) – a fixable violation.
	content := "# Heading\n\nContent   \nNo newline at end"
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Without --quiet the diff is printed to stdout.
	cmd := exec.Command(bin, "--fix-dry-run", mdFile)
	out, _ := cmd.Output()
	if !strings.Contains(string(out), "diff --git") {
		t.Fatalf("expected diff output without --quiet, got: %s", out)
	}

	// With --quiet the diff is suppressed.
	cmd2 := exec.Command(bin, "--quiet", "--fix-dry-run", mdFile)
	out2, _ := cmd2.Output()
	if strings.Contains(string(out2), "diff --git") {
		t.Errorf("expected no diff output with --quiet, got: %s", out2)
	}
}

func TestCLI_Summary_NonDefaultFormats(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	tests := []struct {
		format string
		check  func(t *testing.T, outStr string)
	}{
		{
			format: "json",
			check: func(t *testing.T, outStr string) {
				// The summary is a JSON object appended after the violations array.
				lastNewlineBrace := strings.LastIndex(outStr, "\n{")
				if lastNewlineBrace == -1 {
					t.Fatalf("expected JSON summary object on stdout, got: %s", outStr)
				}
				var counts map[string]int
				if err := json.Unmarshal([]byte(outStr[lastNewlineBrace+1:]), &counts); err != nil {
					t.Fatalf("summary is not a valid JSON object: %v\noutput: %s", err, outStr)
				}
				if counts["MD001"] == 0 {
					t.Errorf("expected MD001 count > 0 in JSON summary, got: %v", counts)
				}
			},
		},
		{
			format: "junit",
			check: func(t *testing.T, outStr string) {
				// Summary is a JUnit XML testsuite appended after the violations testsuite.
				// Expect the summary testsuite name and an MD001 testcase.
				if !strings.Contains(outStr, "markdownlint-summary") {
					t.Errorf("expected 'markdownlint-summary' testsuite in junit summary, got: %s", outStr)
				}
				if !strings.Contains(outStr, "MD001") {
					t.Errorf("expected 'MD001' testcase in junit summary, got: %s", outStr)
				}
			},
		},
		{
			format: "tap",
			check: func(t *testing.T, outStr string) {
				// Summary is a TAP block appended after the violations TAP block.
				if !strings.Contains(outStr, "MD001") {
					t.Errorf("expected 'MD001' in tap summary, got: %s", outStr)
				}
				// TAP summary should say "not ok" with the rule and violation count.
				if !strings.Contains(outStr, "violation") {
					t.Errorf("expected 'violation' in tap summary, got: %s", outStr)
				}
			},
		},
		{
			format: "sarif",
			check: func(t *testing.T, outStr string) {
				// SARIF uses JSON summary: a JSON object appended after the SARIF document.
				lastNewlineBrace := strings.LastIndex(outStr, "\n{")
				if lastNewlineBrace == -1 {
					t.Fatalf("expected JSON summary object on stdout after sarif, got: %s", outStr)
				}
				var counts map[string]int
				if err := json.Unmarshal([]byte(outStr[lastNewlineBrace+1:]), &counts); err != nil {
					t.Fatalf("sarif summary is not a valid JSON object: %v\noutput: %s", err, outStr)
				}
				if counts["MD001"] == 0 {
					t.Errorf("expected MD001 count > 0 in sarif JSON summary, got: %v", counts)
				}
			},
		},
		{
			format: "github",
			check: func(t *testing.T, outStr string) {
				// Summary is ::notice workflow commands, one per rule.
				if !strings.Contains(outStr, "::notice::MD001:") {
					t.Errorf("expected '::notice::MD001:' in github summary, got: %s", outStr)
				}
				if !strings.Contains(outStr, "violation") {
					t.Errorf("expected 'violation' in github summary, got: %s", outStr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			cmd := exec.Command(bin, "--summary", "--output-format", tt.format, testfile)
			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			_ = cmd.Run()

			// No human-readable summary table on stderr for non-default formats.
			if strings.Contains(stderr.String(), "Summary:") {
				t.Errorf("expected no human-readable 'Summary:' on stderr with --output-format %s, got: %s", tt.format, stderr.String())
			}

			tt.check(t, stdout.String())
		})
	}
}

func TestCLI_Watch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal-based test not supported on Windows")
	}
	bin := buildBinary(t)

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Valid\n\nContent.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--watch", mdFile)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start --watch process: %v", err)
	}

	// Wait for the "Watching" message to appear in stderr.
	deadline := make(chan struct{})
	go func() {
		<-time.After(5 * time.Second)
		close(deadline)
	}()

	watching := false
	for !watching {
		select {
		case <-deadline:
			_ = cmd.Process.Kill()
			t.Fatalf("timed out waiting for watch message; stderr: %s", stderr.String())
		default:
			time.Sleep(50 * time.Millisecond)
			if strings.Contains(stderr.String(), "Watching") {
				watching = true
			}
		}
	}

	// Modify the file to trigger a re-lint.
	if err := os.WriteFile(mdFile, []byte("# Updated\n\nContent.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Allow one poll cycle to detect the change.
	time.Sleep(watchInterval + 100*time.Millisecond)

	// Send interrupt to stop the watcher.
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("failed to send interrupt: %v", err)
	}

	err := cmd.Wait()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() != 0 {
			t.Errorf("--watch exit code after interrupt = %d, want 0", exitErr.ExitCode())
		}
	} else if err != nil {
		t.Errorf("unexpected error waiting for --watch process: %v", err)
	}
}

func TestCLI_FixDryRun(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// File with trailing spaces (MD009) and no final newline (MD047) – both fixable.
	content := "# Heading\n\nContent   \nNo newline at end"
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--fix-dry-run", mdFile)
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 after --fix-dry-run (all issues fixable), got: %v", err)
	}

	// The original file must be unchanged.
	got, err := os.ReadFile(mdFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("--fix-dry-run modified the file; want %q, got %q", content, string(got))
	}

	// The diff should be present on stdout.
	diff := stdout.String()
	if !strings.Contains(diff, "diff --git") {
		t.Errorf("expected 'diff --git' header in --fix-dry-run output, got:\n%s", diff)
	}
	if !strings.Contains(diff, "--- a/") {
		t.Errorf("expected '--- a/' in --fix-dry-run output, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+++ b/") {
		t.Errorf("expected '+++ b/' in --fix-dry-run output, got:\n%s", diff)
	}
	// The trailing-spaces fix should appear as a removed line.
	if !strings.Contains(diff, "-Content   ") {
		t.Errorf("expected '-Content   ' (trailing spaces removed) in diff, got:\n%s", diff)
	}
	if !strings.Contains(diff, "+Content") {
		t.Errorf("expected '+Content' (cleaned line) in diff, got:\n%s", diff)
	}
}

func TestCLI_FixDryRun_NoChanges(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// A file with no fixable issues.
	mdFile := filepath.Join(dir, "clean.md")
	if err := os.WriteFile(mdFile, []byte("# Heading\n\nContent.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--fix-dry-run", mdFile)
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 for clean file with --fix-dry-run, got: %v", err)
	}
	if stdout.String() != "" {
		t.Errorf("expected no diff output for clean file, got:\n%s", stdout.String())
	}
}

func TestCLI_FixDryRun_MutualExclusion(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Heading\n\nContent.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--fix", "--fix-dry-run", mdFile)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit when --fix and --fix-dry-run are combined, got nil error")
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("--fix + --fix-dry-run exit code = %d, want 2", exitErr.ExitCode())
	}
}

// TestCLI_ParallelDeterministic verifies that linting multiple files in parallel
// produces output in a consistent (deterministic) order regardless of goroutine
// scheduling. It runs the linter several times on the same set of files and
// checks that the output is identical across runs.
func TestCLI_ParallelDeterministic(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()

	// Create several files with violations so output contains file references.
	files := make([]string, 5)
	for i := range files {
		name := filepath.Join(dir, fmt.Sprintf("file%02d.md", i))
		// MD001: skipped heading level triggers a violation.
		if err := os.WriteFile(name, []byte("# Heading\n\n### Skipped\n"), 0644); err != nil {
			t.Fatal(err)
		}
		files[i] = name
	}

	args := append([]string{"--no-cache"}, files...)
	run := func() string {
		cmd := exec.Command(bin, args...)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		_ = cmd.Run() // violations → exit 1, that's fine
		return stderr.String()
	}

	first := run()
	if first == "" {
		t.Fatal("expected violation output but got nothing")
	}

	// Run several more times and confirm the output is always identical.
	for range 5 {
		if got := run(); got != first {
			t.Errorf("parallel output not deterministic:\nfirst run:\n%s\nlater run:\n%s", first, got)
		}
	}
}

// TestCLI_Fix_LeavesUnchangedFilesUntouched asserts that --fix does not rewrite
// a file it has nothing to change. Rewriting unconditionally bumps the mtime,
// which invalidates downstream build caches and re-triggers file watchers.
func TestCLI_Fix_LeavesUnchangedFilesUntouched(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	clean := filepath.Join(dir, "clean.md")
	if err := os.WriteFile(clean, []byte("# Clean doc\n\nNothing wrong here.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	dirty := filepath.Join(dir, "dirty.md")
	if err := os.WriteFile(dirty, []byte("# Dirty doc\n\nTrailing spaces here.   \n"), 0644); err != nil {
		t.Fatal(err)
	}

	statBefore := func(p string) os.FileInfo {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		return fi
	}
	cleanBefore, dirtyBefore := statBefore(clean), statBefore(dirty)

	// Ensure any rewrite produces a visibly different mtime.
	time.Sleep(1100 * time.Millisecond)

	cmd := exec.Command(bin, "--fix", clean, dirty)
	if out, err := cmd.CombinedOutput(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("unexpected error running --fix: %v\n%s", err, out)
		}
	}

	if got := statBefore(clean).ModTime(); !got.Equal(cleanBefore.ModTime()) {
		t.Errorf("--fix rewrote an unchanged file: mtime went from %v to %v", cleanBefore.ModTime(), got)
	}
	if got := statBefore(dirty).ModTime(); got.Equal(dirtyBefore.ModTime()) {
		t.Errorf("--fix did not rewrite a file that needed fixing (mtime unchanged at %v)", got)
	}
	fixed, err := os.ReadFile(dirty)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(fixed), "here.   ") {
		t.Errorf("--fix left trailing spaces in place: %q", string(fixed))
	}
}

// TestCLI_Fix_PreservesFilePermissions asserts that rewriting a file through
// the atomic write path keeps its original mode rather than resetting it.
func TestCLI_Fix_PreservesFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes not meaningful on Windows")
	}
	bin := buildBinary(t)

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "restricted.md")
	if err := os.WriteFile(mdFile, []byte("# Doc\n\nTrailing spaces.   \n"), 0640); err != nil {
		t.Fatal(err)
	}
	// WriteFile applies the umask, so set the mode explicitly.
	if err := os.Chmod(mdFile, 0640); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--fix", mdFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("unexpected error running --fix: %v\n%s", err, out)
		}
	}

	fi, err := os.Stat(mdFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := fi.Mode().Perm(); got != 0640 {
		t.Errorf("--fix changed file mode from 0640 to %04o", got)
	}
}

// TestCLI_WatchFix_HonoursOverrides asserts that fixes applied during a watch
// cycle use the per-file rule set from "overrides", not the default rule set.
// The watch callback used to fix with the default linter and only then build
// the override-aware one, so a rule disabled by an override was still applied.
func TestCLI_WatchFix_HonoursOverrides(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal-based test not supported on Windows")
	}
	bin := buildBinary(t)

	dir := t.TempDir()
	// MD009 (trailing spaces) is fixable and is disabled for *.md by an override,
	// so --fix must leave trailing spaces alone.
	cfg := "config:\n  MD009: true\noverrides:\n  - files: [\"*.md\"]\n    config:\n      MD009: false\n"
	cfgPath := filepath.Join(dir, ".markdownlint-cli2.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Title\n\nClean line.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--config", cfgPath, "--watch", "--fix", mdFile)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start --watch process: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(stderr.String(), "Watching") {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for watch message; stderr: %s", stderr.String())
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Introduce trailing spaces; the override says MD009 is off for this file.
	withTrailing := "# Title\n\nTrailing spaces here.   \n"
	if err := os.WriteFile(mdFile, []byte(withTrailing), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(watchInterval + 500*time.Millisecond)

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("failed to send interrupt: %v", err)
	}
	_ = cmd.Wait()

	got, err := os.ReadFile(mdFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != withTrailing {
		t.Errorf("watch --fix ignored the MD009 override and rewrote the file:\n got: %q\nwant: %q", string(got), withTrailing)
	}
}
