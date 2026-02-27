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

func TestIsRuleEnabled_SeverityError(t *testing.T) {
	cfg := map[string]interface{}{"MD013": "error"}
	if !isRuleEnabled("MD013", cfg) {
		t.Error("expected MD013 to be enabled when set to \"error\"")
	}
}

func TestIsRuleEnabled_SeverityWarning(t *testing.T) {
	cfg := map[string]interface{}{"MD013": "warning"}
	if !isRuleEnabled("MD013", cfg) {
		t.Error("expected MD013 to be enabled when set to \"warning\"")
	}
}

func TestGetRuleSeverity_Default(t *testing.T) {
	cfg := map[string]interface{}{}
	if got := getRuleSeverity("MD013", cfg); got != "error" {
		t.Errorf("expected \"error\", got %q", got)
	}
}

func TestGetRuleSeverity_Error(t *testing.T) {
	cfg := map[string]interface{}{"MD013": "error"}
	if got := getRuleSeverity("MD013", cfg); got != "error" {
		t.Errorf("expected \"error\", got %q", got)
	}
}

func TestGetRuleSeverity_Warning(t *testing.T) {
	cfg := map[string]interface{}{"MD013": "warning"}
	if got := getRuleSeverity("MD013", cfg); got != "warning" {
		t.Errorf("expected \"warning\", got %q", got)
	}
}

func TestGetRuleSeverity_BoolTrue(t *testing.T) {
	cfg := map[string]interface{}{"MD013": true}
	if got := getRuleSeverity("MD013", cfg); got != "error" {
		t.Errorf("expected \"error\" for bool true, got %q", got)
	}
}

func TestBuildRules_AllEnabled(t *testing.T) {
	got := buildRules(nil)
	if len(got) != 53 {
		t.Errorf("expected 53 rules, got %d", len(got))
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

func TestMergeConfigs(t *testing.T) {
	base := map[string]interface{}{"MD001": false, "MD013": map[string]interface{}{"line_length": 80, "code_blocks": false}}
	overlay := map[string]interface{}{"MD013": map[string]interface{}{"line_length": 120}, "MD041": false}
	merged := mergeConfigs(base, overlay)
	if merged["MD001"] != false {
		t.Errorf("MD001 should be false, got %v", merged["MD001"])
	}
	if merged["MD041"] != false {
		t.Errorf("MD041 should be false, got %v", merged["MD041"])
	}
	lineLengthCfg, ok := merged["MD013"].(map[string]interface{})
	if !ok {
		t.Fatalf("MD013 config not a map, got %T", merged["MD013"])
	}
	if lineLengthCfg["line_length"] != 120 {
		t.Errorf("MD013.line_length should be 120, got %v", lineLengthCfg["line_length"])
	}
	// Deep merge: code_blocks should be preserved from base even though overlay only sets line_length.
	if lineLengthCfg["code_blocks"] != false {
		t.Errorf("MD013.code_blocks should be preserved as false from base, got %v", lineLengthCfg["code_blocks"])
	}
}

func TestEffectiveConfigForFile_NoOverrides(t *testing.T) {
	base := map[string]interface{}{"MD001": false}
	got := effectiveConfigForFile(base, nil, "docs/foo.md")
	if got["MD001"] != false {
		t.Errorf("expected MD001=false, got %v", got["MD001"])
	}
}

func TestEffectiveConfigForFile_OverrideMatches(t *testing.T) {
	base := map[string]interface{}{"MD013": map[string]interface{}{"line_length": 80}}
	overrides := []GlobOverride{
		{
			Files:  []string{"docs/**"},
			Config: map[string]interface{}{"MD013": map[string]interface{}{"line_length": 120}},
		},
	}
	got := effectiveConfigForFile(base, overrides, "docs/readme.md")
	lineLengthCfg, ok := got["MD013"].(map[string]interface{})
	if !ok || lineLengthCfg["line_length"] != 120 {
		t.Errorf("expected MD013.line_length=120 for docs/ file, got %v", got["MD013"])
	}
}

func TestEffectiveConfigForFile_OverrideDoesNotMatch(t *testing.T) {
	base := map[string]interface{}{"MD013": map[string]interface{}{"line_length": 80}}
	overrides := []GlobOverride{
		{
			Files:  []string{"docs/**"},
			Config: map[string]interface{}{"MD013": map[string]interface{}{"line_length": 120}},
		},
	}
	got := effectiveConfigForFile(base, overrides, "src/foo.md")
	lineLengthCfg, ok := got["MD013"].(map[string]interface{})
	if !ok || lineLengthCfg["line_length"] != 80 {
		t.Errorf("expected MD013.line_length=80 for non-docs file, got %v", got["MD013"])
	}
}

func TestEffectiveConfigForFile_MultipleOverridesLastWins(t *testing.T) {
	base := map[string]interface{}{}
	overrides := []GlobOverride{
		{
			Files:  []string{"**/*.md"},
			Config: map[string]interface{}{"MD041": false},
		},
		{
			Files:  []string{"docs/**"},
			Config: map[string]interface{}{"MD041": true},
		},
	}
	// docs/ matches both overrides; last one should win
	got := effectiveConfigForFile(base, overrides, "docs/foo.md")
	if got["MD041"] != true {
		t.Errorf("expected MD041=true (last override wins), got %v", got["MD041"])
	}
	// non-docs matches only first override
	got2 := effectiveConfigForFile(base, overrides, "readme.md")
	if got2["MD041"] != false {
		t.Errorf("expected MD041=false for non-docs file, got %v", got2["MD041"])
	}
}

func TestLoadConfig_YAML_WithOverrides(t *testing.T) {
	dir := t.TempDir()
	content := `
config:
  MD013:
    line_length: 80
overrides:
  - files:
      - "docs/**"
    config:
      MD013:
        line_length: 120
`
	cfgPath := filepath.Join(dir, ".markdownlint-cli2.yaml")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(cfg.Overrides))
	}
	ov := cfg.Overrides[0]
	if len(ov.Files) != 1 || ov.Files[0] != "docs/**" {
		t.Errorf("override files = %v, want [docs/**]", ov.Files)
	}
	lineLengthCfg, ok := ov.Config["MD013"].(map[string]interface{})
	if !ok {
		t.Fatalf("override MD013 config not a map, got %T", ov.Config["MD013"])
	}
	if lineLengthCfg["line_length"] != 120 {
		t.Errorf("override MD013.line_length = %v, want 120", lineLengthCfg["line_length"])
	}
}

func TestLoadConfig_Extends_YAML(t *testing.T) {
	dir := t.TempDir()

	// Base config file
	baseContent := `
config:
  MD001: false
  MD013:
    line_length: 80
ignores:
  - "vendor/**"
`
	basePath := filepath.Join(dir, "base.yaml")
	if err := os.WriteFile(basePath, []byte(baseContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Child config that extends the base
	childContent := `extends: base.yaml
config:
  MD013:
    line_length: 120
ignores:
  - "node_modules/**"
`
	childPath := filepath.Join(dir, ".markdownlint-cli2.yaml")
	if err := os.WriteFile(childPath, []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(childPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// MD001 should be inherited from base
	if v, ok := cfg.Config["MD001"]; !ok || v != false {
		t.Errorf("MD001 = %v, want false (inherited from base)", v)
	}
	// MD013.line_length should be overridden by child
	md013, ok := cfg.Config["MD013"].(map[string]interface{})
	if !ok {
		t.Fatalf("MD013 config not a map, got %T", cfg.Config["MD013"])
	}
	if md013["line_length"] != 120 {
		t.Errorf("MD013.line_length = %v, want 120 (overridden by child)", md013["line_length"])
	}
	// Ignores should be merged (base first, then child)
	if len(cfg.Ignores) != 2 {
		t.Fatalf("ignores = %v, want 2 entries", cfg.Ignores)
	}
	if cfg.Ignores[0] != "vendor/**" || cfg.Ignores[1] != "node_modules/**" {
		t.Errorf("ignores = %v, want [vendor/**, node_modules/**]", cfg.Ignores)
	}
}

func TestLoadConfig_Extends_JSON(t *testing.T) {
	dir := t.TempDir()

	basePath := filepath.Join(dir, "base.json")
	if err := os.WriteFile(basePath, []byte(`{"config":{"MD001":false},"ignores":["vendor/**"]}`), 0644); err != nil {
		t.Fatal(err)
	}

	childContent := `{"extends":"base.json","config":{"MD041":false}}`
	childPath := filepath.Join(dir, ".markdownlint-cli2.json")
	if err := os.WriteFile(childPath, []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(childPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v, ok := cfg.Config["MD001"]; !ok || v != false {
		t.Errorf("MD001 = %v, want false (inherited from base)", v)
	}
	if v, ok := cfg.Config["MD041"]; !ok || v != false {
		t.Errorf("MD041 = %v, want false (from child)", v)
	}
	if len(cfg.Ignores) != 1 || cfg.Ignores[0] != "vendor/**" {
		t.Errorf("ignores = %v, want [vendor/**]", cfg.Ignores)
	}
}

func TestLoadConfig_Extends_CircularReference(t *testing.T) {
	dir := t.TempDir()

	aPath := filepath.Join(dir, "a.yaml")
	bPath := filepath.Join(dir, "b.yaml")

	if err := os.WriteFile(aPath, []byte("extends: b.yaml\nconfig: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte("extends: a.yaml\nconfig: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfig(aPath)
	if err == nil {
		t.Fatal("expected error for circular extends reference, got nil")
	}
}

func TestLoadConfig_Extends_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	childContent := "extends: nonexistent.yaml\nconfig: {}\n"
	childPath := filepath.Join(dir, ".markdownlint-cli2.yaml")
	if err := os.WriteFile(childPath, []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := loadConfig(childPath)
	if err == nil {
		t.Fatal("expected error for missing extends file, got nil")
	}
}

func TestLoadConfig_Extends_ChainInheritance(t *testing.T) {
	dir := t.TempDir()

	// grandparent -> parent -> child
	grandparentContent := "config:\n  MD001: false\n"
	if err := os.WriteFile(filepath.Join(dir, "grandparent.yaml"), []byte(grandparentContent), 0644); err != nil {
		t.Fatal(err)
	}
	parentContent := "extends: grandparent.yaml\nconfig:\n  MD013:\n    line_length: 100\n"
	if err := os.WriteFile(filepath.Join(dir, "parent.yaml"), []byte(parentContent), 0644); err != nil {
		t.Fatal(err)
	}
	childContent := "extends: parent.yaml\nconfig:\n  MD041: false\n"
	childPath := filepath.Join(dir, ".markdownlint-cli2.yaml")
	if err := os.WriteFile(childPath, []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig(childPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v, ok := cfg.Config["MD001"]; !ok || v != false {
		t.Errorf("MD001 = %v, want false (inherited from grandparent)", v)
	}
	md013, ok := cfg.Config["MD013"].(map[string]interface{})
	if !ok {
		t.Fatalf("MD013 config not a map, got %T", cfg.Config["MD013"])
	}
	if md013["line_length"] != 100 {
		t.Errorf("MD013.line_length = %v, want 100 (inherited from parent)", md013["line_length"])
	}
	if v, ok := cfg.Config["MD041"]; !ok || v != false {
		t.Errorf("MD041 = %v, want false (from child)", v)
	}
}

func TestCLI_ExtendsInheritConfig(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// base.yaml disables MD041
	baseContent := "config:\n  MD041: false\n"
	if err := os.WriteFile(filepath.Join(dir, "base.yaml"), []byte(baseContent), 0644); err != nil {
		t.Fatal(err)
	}
	// child config extends base
	childContent := "extends: base.yaml\n"
	if err := os.WriteFile(filepath.Join(dir, ".markdownlint-cli2.yaml"), []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}
	// File that would violate MD041 (no top-level heading)
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, mdFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 when MD041 is disabled via extends, got: %v", err)
	}
}

func TestCLI_ExtendsOverridesInherited(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// base.yaml enables MD041
	baseContent := "config:\n  MD041: true\n"
	if err := os.WriteFile(filepath.Join(dir, "base.yaml"), []byte(baseContent), 0644); err != nil {
		t.Fatal(err)
	}
	// child config extends base and disables MD041
	childContent := "extends: base.yaml\nconfig:\n  MD041: false\n"
	if err := os.WriteFile(filepath.Join(dir, ".markdownlint-cli2.yaml"), []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}
	// File that would violate MD041
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("Not a heading\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, mdFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 when child config overrides MD041: false, got: %v", err)
	}
}

func TestCLI_OverridesApplyToMatchingFile(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.Mkdir(docsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// A line of exactly 100 characters
	line := "# " + string(make([]byte, 98))
	for i := range line[2:] {
		line = line[:2+i] + "a" + line[2+i+1:]
	}
	// docs/file.md has a 100-char line; should pass with override line_length:100
	docsFile := filepath.Join(docsDir, "file.md")
	if err := os.WriteFile(docsFile, []byte(line+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfgContent := `config:
  MD013:
    line_length: 80
overrides:
  - files:
      - "docs/**"
    config:
      MD013:
        line_length: 100
`
	if err := os.WriteFile(filepath.Join(dir, ".markdownlint-cli2.yaml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, docsFile)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 with override line_length:100 for docs/ file, got: %v", err)
	}
}

func TestCLI_OverridesDoNotApplyToNonMatchingFile(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	// A line of exactly 100 characters
	line := "# " + string(make([]byte, 98))
	for i := range line[2:] {
		line = line[:2+i] + "a" + line[2+i+1:]
	}
	// root file.md has a 100-char line; should fail with base line_length:80
	rootFile := filepath.Join(dir, "file.md")
	if err := os.WriteFile(rootFile, []byte(line+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfgContent := `config:
  MD013:
    line_length: 80
overrides:
  - files:
      - "docs/**"
    config:
      MD013:
        line_length: 100
`
	if err := os.WriteFile(filepath.Join(dir, ".markdownlint-cli2.yaml"), []byte(cfgContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, rootFile)
	cmd.Dir = dir
	if err := cmd.Run(); err == nil {
		t.Error("expected non-zero exit for non-docs file with line_length:80, got exit 0")
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
