package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_FormatJSON_NoViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_valid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}
	cmd := exec.Command(bin, "--format=json", testfile)
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit 0 for valid file, got: %v", err)
	}
	var result []interface{}
	if err := json.Unmarshal(stdout, &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if len(result) != 0 {
		t.Errorf("expected empty JSON array, got %d elements", len(result))
	}
}

func TestCLI_FormatJSON_WithViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}
	cmd := exec.Command(bin, "--format=json", testfile)
	stdout, err := cmd.Output()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for file with violations")
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(stdout, &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if len(result) == 0 {
		t.Error("expected at least one violation in JSON output")
	}
	for _, item := range result {
		for _, field := range []string{"file", "line", "column", "rule", "message", "severity"} {
			if _, ok := item[field]; !ok {
				t.Errorf("JSON violation missing field %q", field)
			}
		}
	}
}

func TestCLI_FormatJUnit_NoViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_valid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}
	cmd := exec.Command(bin, "--format=junit", testfile)
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit 0 for valid file, got: %v", err)
	}
	var ts xmlTestSuites
	if err := xml.Unmarshal(stdout, &ts); err != nil {
		t.Fatalf("output is not valid JUnit XML: %v\noutput: %s", err, stdout)
	}
	if len(ts.TestSuites) != 1 {
		t.Fatalf("expected 1 testsuite, got %d", len(ts.TestSuites))
	}
	if ts.TestSuites[0].Failures != 0 {
		t.Errorf("expected 0 failures, got %d", ts.TestSuites[0].Failures)
	}
}

func TestCLI_FormatJUnit_WithViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}
	cmd := exec.Command(bin, "--format=junit", testfile)
	stdout, err := cmd.Output()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for file with violations")
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
	var ts xmlTestSuites
	if err := xml.Unmarshal(stdout, &ts); err != nil {
		t.Fatalf("output is not valid JUnit XML: %v\noutput: %s", err, stdout)
	}
	if len(ts.TestSuites) != 1 {
		t.Fatalf("expected 1 testsuite, got %d", len(ts.TestSuites))
	}
	if ts.TestSuites[0].Failures == 0 {
		t.Error("expected at least one failure in JUnit output")
	}
}

func TestCLI_FormatSARIF_NoViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_valid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}
	cmd := exec.Command(bin, "--format=sarif", testfile)
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("expected exit 0 for valid file, got: %v", err)
	}
	var log sarifLog
	if err := json.Unmarshal(stdout, &log); err != nil {
		t.Fatalf("output is not valid SARIF JSON: %v\noutput: %s", err, stdout)
	}
	if log.Version != "2.1.0" {
		t.Errorf("SARIF version = %q, want 2.1.0", log.Version)
	}
	if len(log.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(log.Runs))
	}
	if len(log.Runs[0].Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(log.Runs[0].Results))
	}
}

func TestCLI_FormatSARIF_WithViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}
	cmd := exec.Command(bin, "--format=sarif", testfile)
	stdout, err := cmd.Output()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for file with violations")
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
	var log sarifLog
	if err := json.Unmarshal(stdout, &log); err != nil {
		t.Fatalf("output is not valid SARIF JSON: %v\noutput: %s", err, stdout)
	}
	if log.Version != "2.1.0" {
		t.Errorf("SARIF version = %q, want 2.1.0", log.Version)
	}
	if len(log.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(log.Runs))
	}
	if len(log.Runs[0].Results) == 0 {
		t.Error("expected at least one result in SARIF output")
	}
}

func TestCLI_FormatUnknown(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_valid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}
	cmd := exec.Command(bin, "--format=csv", testfile)
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for unknown format, got nil error")
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("unknown format exit code = %d, want 2", exitErr.ExitCode())
	}
}

func TestCLI_FormatJSON_Stdin_WithViolations(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--format=json", "-")
	cmd.Stdin = strings.NewReader("# Heading\n\n### Skipped level\n")
	stdout, err := cmd.Output()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for stdin with violations")
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(stdout, &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if len(result) == 0 {
		t.Error("expected at least one violation in JSON output")
	}
	// Ensure file field is "stdin"
	if result[0]["file"] != "stdin" {
		t.Errorf("expected file=stdin, got %v", result[0]["file"])
	}
}
