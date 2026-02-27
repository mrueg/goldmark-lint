package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrueg/goldmark-lint/lint"
)

// --- Unit tests for formatter functions ---

func makeViolations() []fileViolation {
	return []fileViolation{
		{
			File: "test.md",
			Violations: []lint.Violation{
				{Rule: "MD001", Line: 3, Column: 1, Message: "Heading levels should only increment by one level at a time", Severity: "error"},
				{Rule: "MD013", Line: 5, Column: 82, Message: "Line length", Severity: "warning"},
			},
		},
	}
}

func TestFormatDefault_Output(t *testing.T) {
	violations := makeViolations()
	var buf bytes.Buffer
	formatDefault(violations, &buf)
	got := buf.String()
	if !strings.Contains(got, "test.md:3:1 MD001") {
		t.Errorf("expected MD001 violation in output, got: %s", got)
	}
	if !strings.Contains(got, "test.md:5:82 MD013") {
		t.Errorf("expected MD013 violation in output, got: %s", got)
	}
}

func TestFormatDefault_Empty(t *testing.T) {
	var buf bytes.Buffer
	formatDefault(nil, &buf)
	if buf.Len() != 0 {
		t.Errorf("expected empty output for no violations, got: %s", buf.String())
	}
}

func TestFormatJSON_ValidJSON(t *testing.T) {
	violations := makeViolations()
	var buf bytes.Buffer
	formatJSON(violations, &buf)

	var results []jsonViolation
	if err := json.Unmarshal(buf.Bytes(), &results); err != nil {
		t.Fatalf("formatJSON produced invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(results) != 2 {
		t.Errorf("expected 2 JSON violations, got %d", len(results))
	}
	if results[0].FileName != "test.md" {
		t.Errorf("fileName = %q, want %q", results[0].FileName, "test.md")
	}
	if results[0].LineNumber != 3 {
		t.Errorf("lineNumber = %d, want 3", results[0].LineNumber)
	}
	if results[0].ColumnNumber != 1 {
		t.Errorf("columnNumber = %d, want 1", results[0].ColumnNumber)
	}
	if len(results[0].RuleNames) != 1 || results[0].RuleNames[0] != "MD001" {
		t.Errorf("ruleNames = %v, want [MD001]", results[0].RuleNames)
	}
	if results[0].RuleDescription == "" {
		t.Error("ruleDescription should not be empty")
	}
	if !strings.Contains(results[0].RuleInformation, "md001") {
		t.Errorf("ruleInformation = %q, want URL containing 'md001'", results[0].RuleInformation)
	}
	if results[0].ErrorDetail != nil {
		t.Error("errorDetail should be null")
	}
}

func TestFormatJSON_Empty(t *testing.T) {
	var buf bytes.Buffer
	formatJSON(nil, &buf)
	var results []jsonViolation
	if err := json.Unmarshal(buf.Bytes(), &results); err != nil {
		t.Fatalf("formatJSON produced invalid JSON for empty violations: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty array, got %d elements", len(results))
	}
}

func TestFormatJUnit_ValidXML(t *testing.T) {
	violations := makeViolations()
	var buf bytes.Buffer
	formatJUnit(violations, &buf)

	var suites xmlTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &suites); err != nil {
		t.Fatalf("formatJUnit produced invalid XML: %v\noutput: %s", err, buf.String())
	}
	if len(suites.Suites) != 1 {
		t.Fatalf("expected 1 testsuite, got %d", len(suites.Suites))
	}
	suite := suites.Suites[0]
	if suite.Name != "markdownlint" {
		t.Errorf("testsuite name = %q, want %q", suite.Name, "markdownlint")
	}
	if suite.Tests != 1 {
		t.Errorf("tests = %d, want 1", suite.Tests)
	}
	if suite.Failures != 2 {
		t.Errorf("failures = %d, want 2", suite.Failures)
	}
	if len(suite.Cases) != 1 {
		t.Fatalf("expected 1 testcase, got %d", len(suite.Cases))
	}
	if suite.Cases[0].Name != "test.md" {
		t.Errorf("testcase name = %q, want %q", suite.Cases[0].Name, "test.md")
	}
	if len(suite.Cases[0].Failures) != 2 {
		t.Errorf("testcase failures = %d, want 2", len(suite.Cases[0].Failures))
	}
}

func TestFormatJUnit_XMLHeader(t *testing.T) {
	var buf bytes.Buffer
	formatJUnit(makeViolations(), &buf)
	if !strings.HasPrefix(buf.String(), "<?xml") {
		t.Errorf("expected XML header, got: %s", buf.String()[:min(50, buf.Len())])
	}
}

func TestFormatTAP_Output(t *testing.T) {
	violations := makeViolations()
	var buf bytes.Buffer
	formatTAP(violations, &buf)
	got := buf.String()
	if !strings.HasPrefix(got, "TAP version 13\n") {
		t.Errorf("expected TAP version header, got: %s", got[:min(50, len(got))])
	}
	if !strings.Contains(got, "1..2\n") {
		t.Errorf("expected plan '1..2', got: %s", got)
	}
	if !strings.Contains(got, "not ok 1") {
		t.Errorf("expected 'not ok 1' line, got: %s", got)
	}
	if !strings.Contains(got, "not ok 2") {
		t.Errorf("expected 'not ok 2' line, got: %s", got)
	}
}

func TestFormatTAP_Empty(t *testing.T) {
	var buf bytes.Buffer
	formatTAP(nil, &buf)
	got := buf.String()
	if !strings.Contains(got, "1..0") {
		t.Errorf("expected plan '1..0' for no violations, got: %s", got)
	}
}

func TestRuleInfoURL(t *testing.T) {
	got := ruleInfoURL("MD001")
	want := "https://github.com/DavidAnson/markdownlint/blob/main/doc/md001.md"
	if got != want {
		t.Errorf("ruleInfoURL(%q) = %q, want %q", "MD001", got, want)
	}
}

func TestParseOutputFormatters_JSON(t *testing.T) {
	raw := []interface{}{
		[]interface{}{"markdownlint-cli2-formatter-json", map[string]interface{}{"outfile": "results.json"}},
	}
	specs := parseOutputFormatters(raw)
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].format != "json" {
		t.Errorf("format = %q, want %q", specs[0].format, "json")
	}
	if specs[0].outfile != "results.json" {
		t.Errorf("outfile = %q, want %q", specs[0].outfile, "results.json")
	}
}

func TestParseOutputFormatters_NoOutfile(t *testing.T) {
	raw := []interface{}{
		[]interface{}{"markdownlint-cli2-formatter-tap"},
	}
	specs := parseOutputFormatters(raw)
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].format != "tap" {
		t.Errorf("format = %q, want %q", specs[0].format, "tap")
	}
	if specs[0].outfile != "" {
		t.Errorf("outfile should be empty, got %q", specs[0].outfile)
	}
}

func TestFormatterNameToFormat(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"markdownlint-cli2-formatter-json", "json"},
		{"markdownlint-cli2-formatter-junit", "junit"},
		{"markdownlint-cli2-formatter-tap", "tap"},
		{"markdownlint-cli2-formatter-default", "default"},
		{"json", "json"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := formatterNameToFormat(tt.name)
		if got != tt.want {
			t.Errorf("formatterNameToFormat(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// --- CLI integration tests for --output-format ---

func TestCLI_OutputFormat_JSON(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	cmd := exec.Command(bin, "--output-format", "json", testfile)
	stdout, err := cmd.Output()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for file with violations, got nil error")
	}

	var results []jsonViolation
	if err := json.Unmarshal(stdout, &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if len(results) == 0 {
		t.Error("expected at least one JSON violation, got empty array")
	}
	if results[0].FileName == "" {
		t.Error("fileName should not be empty")
	}
	if len(results[0].RuleNames) == 0 {
		t.Error("ruleNames should not be empty")
	}
}

func TestCLI_OutputFormat_JSON_NoViolations(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_valid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	cmd := exec.Command(bin, "--output-format", "json", testfile)
	stdout, err := cmd.Output()
	if err != nil {
		t.Errorf("expected exit 0 for valid file, got: %v", err)
	}
	var results []jsonViolation
	if err := json.Unmarshal(stdout, &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if len(results) != 0 {
		t.Errorf("expected empty array for valid file, got %d violations", len(results))
	}
}

func TestCLI_OutputFormat_JSON_ToStdout(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	cmd := exec.Command(bin, "--output-format", "json", testfile)
	// cmd.Output() captures stdout; verify it contains JSON (non-empty)
	stdout, _ := cmd.Output()
	if len(stdout) == 0 {
		t.Error("expected JSON output on stdout, got nothing")
	}
	if stdout[0] != '[' {
		t.Errorf("expected JSON array on stdout, got: %s", stdout[:min(20, len(stdout))])
	}
}

func TestCLI_OutputFormat_JUnit(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	cmd := exec.Command(bin, "--output-format", "junit", testfile)
	stdout, _ := cmd.Output()
	if !strings.Contains(string(stdout), "<testsuites>") {
		t.Errorf("expected JUnit XML output, got: %s", stdout[:min(100, len(stdout))])
	}

	var suites xmlTestSuites
	if err := xml.Unmarshal(stdout, &suites); err != nil {
		t.Fatalf("output is not valid JUnit XML: %v\noutput: %s", err, stdout)
	}
}

func TestCLI_OutputFormat_TAP(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	cmd := exec.Command(bin, "--output-format", "tap", testfile)
	stdout, _ := cmd.Output()
	got := string(stdout)
	if !strings.HasPrefix(got, "TAP version 13\n") {
		t.Errorf("expected TAP version header, got: %s", got[:min(50, len(got))])
	}
	if !strings.Contains(got, "not ok") {
		t.Errorf("expected 'not ok' lines in TAP output, got: %s", got)
	}
}

func TestCLI_OutputFormat_Default_ToStderr(t *testing.T) {
	bin := buildBinary(t)
	testfile := filepath.Join("..", "..", "testdata", "md001_invalid.md")
	if _, err := os.Stat(testfile); err != nil {
		t.Skip("testdata not available")
	}

	cmd := exec.Command(bin, "--output-format", "default", testfile)
	// Only capture stdout; default output goes to stderr.
	stdout, _ := cmd.Output()
	if len(stdout) != 0 {
		t.Errorf("expected no output on stdout for default format, got: %s", stdout)
	}
}

func TestCLI_OutputFormat_Invalid(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--output-format", "invalid-format", "dummy.md")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for invalid format, got nil error")
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2 for invalid format, got %d", exitErr.ExitCode())
	}
}

func TestCLI_OutputFormatters_Config_JSON(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	// File that violates MD041 (no top-level heading)
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(dir, "results.json")
	cfgContent := `outputFormatters:
  - - markdownlint-cli2-formatter-json
    - outfile: ` + outFile + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".markdownlint-cli2.yaml"), []byte(cfgContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, mdFile)
	cmd.Dir = dir
	_ = cmd.Run() // ignore exit code; we just want to check the outfile

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("expected outfile %s to be created, got error: %v", outFile, err)
	}
	var results []jsonViolation
	if err := json.Unmarshal(data, &results); err != nil {
		t.Fatalf("outfile is not valid JSON: %v\ndata: %s", err, data)
	}
	if len(results) == 0 {
		t.Error("expected at least one JSON violation in outfile")
	}
}

func TestCLI_OutputFormat_Stdin_JSON(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--output-format", "json", "-")
	cmd.Stdin = strings.NewReader("# Heading\n\n### Skipped level\n")
	stdout, err := cmd.Output()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected non-zero exit for stdin with violations, got nil error")
	}
	var results []jsonViolation
	if err := json.Unmarshal(stdout, &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if len(results) == 0 {
		t.Error("expected at least one violation from stdin")
	}
	if results[0].FileName != "stdin" {
		t.Errorf("fileName = %q, want %q", results[0].FileName, "stdin")
	}
}
