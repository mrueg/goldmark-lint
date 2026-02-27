package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/mrueg/goldmark-lint/lint/rules"
)

// ConfigFile represents the top-level markdownlint-cli2 config file structure.
type ConfigFile struct {
	Config  map[string]interface{} `yaml:"config"  json:"config"`
	Ignores []string               `yaml:"ignores" json:"ignores"`
}

var configFileNames = []string{
	".markdownlint-cli2.yaml",
	".markdownlint-cli2.yml",
	".markdownlint-cli2.jsonc",
	".markdownlint-cli2.json",
}

// findConfigFile searches for a markdownlint-cli2 config file starting from dir
// and walking up to the filesystem root. Returns the first found config file path,
// or an empty string if none is found.
func findConfigFile(dir string) string {
	for {
		for _, name := range configFileNames {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// loadConfig loads and parses a markdownlint-cli2 config file.
func loadConfig(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ConfigFile
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
	case ".json", ".jsonc":
		if err := json.Unmarshal(stripJSONComments(data), &cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", path)
	}
	return &cfg, nil
}

// stripJSONComments removes // line comments and /* */ block comments from JSON
// data, ignoring comment-like sequences inside strings.
func stripJSONComments(data []byte) []byte {
	result := make([]byte, 0, len(data))
	inString := false
	inLineComment := false
	inBlockComment := false
	i := 0
	for i < len(data) {
		c := data[i]
		if inLineComment {
			if c == '\n' {
				inLineComment = false
				result = append(result, c)
			}
			i++
			continue
		}
		if inBlockComment {
			if c == '*' && i+1 < len(data) && data[i+1] == '/' {
				inBlockComment = false
				i += 2
			} else {
				i++
			}
			continue
		}
		if inString {
			if c == '\\' && i+1 < len(data) {
				result = append(result, c, data[i+1])
				i += 2
				continue
			}
			if c == '"' {
				inString = false
			}
			result = append(result, c)
			i++
			continue
		}
		if c == '"' {
			inString = true
			result = append(result, c)
			i++
			continue
		}
		if c == '/' && i+1 < len(data) {
			if data[i+1] == '/' {
				inLineComment = true
				i += 2
				continue
			}
			if data[i+1] == '*' {
				inBlockComment = true
				i += 2
				continue
			}
		}
		result = append(result, c)
		i++
	}
	return result
}

// isRuleEnabled returns whether the rule with the given ID should be run.
// It checks the rule's config entry and falls back to the "default" key.
func isRuleEnabled(id string, cfg map[string]interface{}) bool {
	if val, ok := cfg[id]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case map[string]interface{}:
			return true
		}
	}
	if d, ok := cfg["default"]; ok {
		if b, ok := d.(bool); ok {
			return b
		}
	}
	return true
}

// applyRuleConfig applies rule-specific config options to the rule pointer by
// marshaling the config map entry to JSON and unmarshaling it into the rule.
func applyRuleConfig(rule interface{}, cfg map[string]interface{}, id string) {
	val, ok := cfg[id]
	if !ok {
		return
	}
	m, ok := val.(map[string]interface{})
	if !ok {
		return
	}
	data, err := json.Marshal(m)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, rule)
}

// newLinterFromConfig creates a Linter using the given rule config map.
// If cfg is nil, all rules are enabled with their default options.
func newLinterFromConfig(cfg map[string]interface{}) *lint.Linter {
	return lint.NewLinter(buildRules(cfg)...)
}

// buildRules constructs the list of lint rules based on the provided config map.
// If cfg is nil, all rules are enabled with their default options.
func buildRules(cfg map[string]interface{}) []lint.Rule {
	if cfg == nil {
		cfg = map[string]interface{}{}
	}

	type ruleFactory struct {
		id      string
		factory func() lint.Rule
	}

	factories := []ruleFactory{
		{"MD001", func() lint.Rule { r := &rules.MD001{}; applyRuleConfig(r, cfg, "MD001"); return r }},
		{"MD003", func() lint.Rule { r := &rules.MD003{}; applyRuleConfig(r, cfg, "MD003"); return r }},
		{"MD004", func() lint.Rule { r := &rules.MD004{}; applyRuleConfig(r, cfg, "MD004"); return r }},
		{"MD007", func() lint.Rule { r := &rules.MD007{}; applyRuleConfig(r, cfg, "MD007"); return r }},
		{"MD009", func() lint.Rule { r := &rules.MD009{}; applyRuleConfig(r, cfg, "MD009"); return r }},
		{"MD010", func() lint.Rule { r := &rules.MD010{}; applyRuleConfig(r, cfg, "MD010"); return r }},
		{"MD012", func() lint.Rule { r := &rules.MD012{}; applyRuleConfig(r, cfg, "MD012"); return r }},
		{"MD013", func() lint.Rule { r := &rules.MD013{}; applyRuleConfig(r, cfg, "MD013"); return r }},
		{"MD022", func() lint.Rule { r := &rules.MD022{}; applyRuleConfig(r, cfg, "MD022"); return r }},
		{"MD024", func() lint.Rule { r := &rules.MD024{}; applyRuleConfig(r, cfg, "MD024"); return r }},
		{"MD025", func() lint.Rule { r := &rules.MD025{}; applyRuleConfig(r, cfg, "MD025"); return r }},
		{"MD029", func() lint.Rule { r := &rules.MD029{}; applyRuleConfig(r, cfg, "MD029"); return r }},
		{"MD033", func() lint.Rule { r := &rules.MD033{}; applyRuleConfig(r, cfg, "MD033"); return r }},
		{"MD034", func() lint.Rule { r := &rules.MD034{}; applyRuleConfig(r, cfg, "MD034"); return r }},
		{"MD041", func() lint.Rule { r := &rules.MD041{}; applyRuleConfig(r, cfg, "MD041"); return r }},
		{"MD047", func() lint.Rule { r := &rules.MD047{}; applyRuleConfig(r, cfg, "MD047"); return r }},
	}

	var result []lint.Rule
	for _, f := range factories {
		if isRuleEnabled(f.id, cfg) {
			result = append(result, f.factory())
		}
	}
	return result
}

// isIgnored reports whether path matches any of the ignore glob patterns.
func isIgnored(path string, patterns []string) bool {
	normalized := filepath.ToSlash(filepath.Clean(path))
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)
		if matchPath(pattern, normalized) {
			return true
		}
	}
	return false
}

// matchPath checks whether name matches the glob pattern, supporting ** for
// matching across path separators. For patterns containing **, the match is
// attempted at each path component position so that relative patterns such
// as "vendor/**" also match absolute paths like "/abs/path/vendor/file".
func matchPath(pattern, name string) bool {
	if !strings.Contains(pattern, "**") {
		if ok, _ := filepath.Match(pattern, name); ok {
			return true
		}
		ok, _ := filepath.Match(pattern, filepath.Base(name))
		return ok
	}
	patParts := strings.Split(pattern, "/")
	nameParts := strings.Split(name, "/")
	// Try matching the pattern starting from each position in the path so that
	// relative patterns work against both relative and absolute paths.
	for i := range nameParts {
		if matchSegments(patParts, nameParts[i:]) {
			return true
		}
	}
	return false
}

// matchSegments recursively matches pattern segments against name segments,
// handling ** as a wildcard that matches zero or more path segments.
func matchSegments(pat, name []string) bool {
	if len(pat) == 0 {
		return len(name) == 0
	}
	if pat[0] == "**" {
		if matchSegments(pat[1:], name) {
			return true
		}
		for i := range name {
			if matchSegments(pat[1:], name[i+1:]) {
				return true
			}
		}
		return false
	}
	if len(name) == 0 {
		return false
	}
	ok, _ := filepath.Match(pat[0], name[0])
	if !ok {
		return false
	}
	return matchSegments(pat[1:], name[1:])
}
