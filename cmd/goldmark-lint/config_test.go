package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFindConfigFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	got := findConfigFile(dir)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFindConfigFile_YAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".markdownlint-cli2.yaml")
	if err := os.WriteFile(cfgPath, []byte("config: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := findConfigFile(dir)
	if got != cfgPath {
		t.Errorf("expected %q, got %q", cfgPath, got)
	}
}

func TestFindConfigFile_JSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".markdownlint-cli2.json")
	if err := os.WriteFile(cfgPath, []byte(`{"config":{}}`), 0644); err != nil {
		t.Fatal(err)
	}
	got := findConfigFile(dir)
	if got != cfgPath {
		t.Errorf("expected %q, got %q", cfgPath, got)
	}
}

func TestFindConfigFile_ParentDirectory(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "sub")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(parent, ".markdownlint-cli2.yaml")
	if err := os.WriteFile(cfgPath, []byte("config: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := findConfigFile(child)
	if got != cfgPath {
		t.Errorf("expected %q, got %q", cfgPath, got)
	}
}

func TestLoadConfig_YAML(t *testing.T) {
	dir := t.TempDir()
	content := `
config:
  MD013:
    line_length: 100
  MD001: false
ignores:
  - "vendor/**"
`
	cfgPath := filepath.Join(dir, ".markdownlint-cli2.yaml")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if len(cfg.Ignores) != 1 || cfg.Ignores[0] != "vendor/**" {
		t.Errorf("ignores = %v, want [vendor/**]", cfg.Ignores)
	}
	if cfg.Config == nil {
		t.Fatal("expected non-nil config.Config")
	}
	if v, ok := cfg.Config["MD001"]; !ok || v != false {
		t.Errorf("MD001 config = %v, want false", v)
	}
}

func TestLoadConfig_JSON(t *testing.T) {
	dir := t.TempDir()
	content := `{"config":{"MD013":{"line_length":100},"MD001":false},"ignores":["vendor/**"]}`
	cfgPath := filepath.Join(dir, ".markdownlint-cli2.json")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Ignores) != 1 || cfg.Ignores[0] != "vendor/**" {
		t.Errorf("ignores = %v, want [vendor/**]", cfg.Ignores)
	}
}

func TestLoadConfig_JSONC(t *testing.T) {
	dir := t.TempDir()
	content := `{
  // Enable line length rule with custom length
  "config": {
    "MD013": {"line_length": 100} /* line length */
  }
}`
	cfgPath := filepath.Join(dir, ".markdownlint-cli2.jsonc")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Config == nil {
		t.Fatal("expected non-nil config.Config")
	}
}

func TestIsRuleEnabled_DefaultTrue(t *testing.T) {
	cfg := map[string]interface{}{}
	if !isRuleEnabled("MD001", cfg) {
		t.Error("expected MD001 to be enabled by default")
	}
}

func TestIsRuleEnabled_ExplicitFalse(t *testing.T) {
	cfg := map[string]interface{}{"MD001": false}
	if isRuleEnabled("MD001", cfg) {
		t.Error("expected MD001 to be disabled")
	}
}

func TestIsRuleEnabled_ExplicitTrue(t *testing.T) {
	cfg := map[string]interface{}{"MD001": true}
	if !isRuleEnabled("MD001", cfg) {
		t.Error("expected MD001 to be enabled")
	}
}

func TestIsRuleEnabled_DefaultFalse(t *testing.T) {
	cfg := map[string]interface{}{"default": false}
	if isRuleEnabled("MD001", cfg) {
		t.Error("expected MD001 to be disabled due to default:false")
	}
}

func TestIsRuleEnabled_DefaultFalse_ExplicitConfig(t *testing.T) {
	cfg := map[string]interface{}{
		"default": false,
		"MD013":   map[string]interface{}{"line_length": 100},
	}
	if !isRuleEnabled("MD013", cfg) {
		t.Error("expected MD013 to be enabled (has config options)")
	}
	if isRuleEnabled("MD001", cfg) {
		t.Error("expected MD001 to be disabled (default:false)")
	}
}

func TestBuildRules_AllEnabled(t *testing.T) {
	got := buildRules(nil)
	if len(got) != 16 {
		t.Errorf("expected 16 rules, got %d", len(got))
	}
}

func TestBuildRules_DisableRule(t *testing.T) {
	cfg := map[string]interface{}{"MD001": false}
	got := buildRules(cfg)
	for _, r := range got {
		if r.ID() == "MD001" {
			t.Error("expected MD001 to be excluded")
		}
	}
}

func TestBuildRules_DefaultFalse(t *testing.T) {
	cfg := map[string]interface{}{
		"default": false,
		"MD013":   map[string]interface{}{"line_length": 100},
	}
	got := buildRules(cfg)
	if len(got) != 1 {
		t.Errorf("expected 1 rule, got %d", len(got))
	}
	if got[0].ID() != "MD013" {
		t.Errorf("expected MD013, got %s", got[0].ID())
	}
}

func TestIsIgnored(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		want     bool
	}{
		{"vendor/foo.md", []string{"vendor/**"}, true},
		{"docs/foo.md", []string{"vendor/**"}, false},
		{"foo.md", []string{"*.md"}, true},
		{"sub/foo.md", []string{"*.md"}, true},
		{"foo.txt", []string{"*.md"}, false},
		{"node_modules/bar.md", []string{"**/node_modules/**"}, true},
		{"a/node_modules/bar.md", []string{"**/node_modules/**"}, true},
		{"src/bar.md", []string{"**/node_modules/**"}, false},
	}
	for _, tt := range tests {
		got := isIgnored(tt.path, tt.patterns)
		if got != tt.want {
			t.Errorf("isIgnored(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.want)
		}
	}
}

func TestStripJSONComments(t *testing.T) {
	input := `{
  // line comment
  "key": "value", /* block comment */
  "url": "https://example.com/path"
}`
	got := string(stripJSONComments([]byte(input)))
	// Should be valid JSON after stripping
	want := `{
  
  "key": "value", 
  "url": "https://example.com/path"
}`
	if got != want {
		t.Errorf("stripJSONComments() = %q, want %q", got, want)
	}
}

func TestCLI_ConfigDisablesRule(t *testing.T) {
	bin := buildBinary(t)

	// Create temp dir with a config that disables MD041 and a file that would
	// otherwise violate MD041 (no top-level heading).
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, ".markdownlint-cli2.yaml")
	cfgContent := "config:\n  default: false\n  MD047: true\n"
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, mdFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 when all failing rules are disabled, got: %v", err)
	}
}

func TestCLI_ConfigIgnoresFile(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// Create a file with violations in a vendor subdir.
	vendorDir := filepath.Join(dir, "vendor")
	if err := os.Mkdir(vendorDir, 0755); err != nil {
		t.Fatal(err)
	}
	mdFile := filepath.Join(vendorDir, "test.md")
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Config ignores vendor/**
	cfgContent := "ignores:\n  - \"vendor/**\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".markdownlint-cli2.yaml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, mdFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 for ignored file, got: %v", err)
	}
}

func TestCLI_ConfigRuleOptions(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// A line of exactly 100 characters - should pass with line_length:100 but fail with default 80.
	line := "# " + string(make([]byte, 98))
	for i := range line[2:] {
		line = line[:2+i] + "a" + line[2+i+1:]
	}
	mdFile := filepath.Join(dir, "test.md")
	content := line + "\n"
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	// Config sets line_length to 100.
	cfgContent := "config:\n  MD013:\n    line_length: 100\n"
	if err := os.WriteFile(filepath.Join(dir, ".markdownlint-cli2.yaml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, mdFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 with line_length:100 config, got: %v", err)
	}
}
