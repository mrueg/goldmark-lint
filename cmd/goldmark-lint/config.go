package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/mrueg/goldmark-lint/lint/rules"
)

// GlobOverride allows specifying different rule configurations for files
// matching specific glob patterns, mirroring markdownlint-cli2's "overrides".
type GlobOverride struct {
	Files  []string               `yaml:"files"  json:"files"`
	Config map[string]interface{} `yaml:"config" json:"config"`
}

// ConfigFile represents the top-level markdownlint-cli2 config file structure.
type ConfigFile struct {
	Extends          string                 `yaml:"extends"          json:"extends"`
	Config           map[string]interface{} `yaml:"config"           json:"config"`
	Ignores          []string               `yaml:"ignores"          json:"ignores"`
	Overrides        []GlobOverride         `yaml:"overrides"        json:"overrides"`
	OutputFormatters []interface{}          `yaml:"outputFormatters" json:"outputFormatters"`
	NoInlineConfig   bool                   `yaml:"noInlineConfig"   json:"noInlineConfig"`
	Globs            []string               `yaml:"globs"            json:"globs"`
	Fix              bool                   `yaml:"fix"              json:"fix"`
	FrontMatter      string                 `yaml:"frontMatter"      json:"frontMatter"`
	Gitignore        interface{}            `yaml:"gitignore"        json:"gitignore"`
}

var configFileNames = []string{
	".markdownlint-cli2.yaml",
	".markdownlint-cli2.yml",
	".markdownlint-cli2.jsonc",
	".markdownlint-cli2.json",
	".markdownlint.yaml",
	".markdownlint.yml",
	".markdownlint.jsonc",
	".markdownlint.json",
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

// isSimpleFormatConfig reports whether path is a .markdownlint.* file (not
// .markdownlint-cli2.*). These files use a rule-only format where the entire
// content is the rule config map, with no wrapping "config:" key.
func isSimpleFormatConfig(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, ".markdownlint.") && !strings.HasPrefix(base, ".markdownlint-cli2.")
}

// loadConfig loads and parses a markdownlint-cli2 config file, resolving any
// "extends" references recursively. Circular references are detected and
// reported as errors.
func loadConfig(path string) (*ConfigFile, error) {
	return loadConfigResolved(path, make(map[string]bool))
}

// loadConfigResolved is the internal recursive implementation of loadConfig.
// visited tracks absolute paths already being loaded to detect circular refs.
func loadConfigResolved(path string, visited map[string]bool) (*ConfigFile, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if visited[absPath] {
		return nil, fmt.Errorf("circular extends reference detected: %s", absPath)
	}
	visited[absPath] = true

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// .markdownlint.* files use a simpler rule-only format where the entire
	// file content is the rule config map (no "config:" wrapper).
	if isSimpleFormatConfig(path) {
		var ruleCfg map[string]interface{}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml":
			if err := yaml.Unmarshal(data, &ruleCfg); err != nil {
				return nil, fmt.Errorf("parsing %s: %w", path, err)
			}
		case ".json", ".jsonc":
			if err := json.Unmarshal(stripJSONComments(data), &ruleCfg); err != nil {
				return nil, fmt.Errorf("parsing %s: %w", path, err)
			}
		default:
			return nil, fmt.Errorf("unsupported config file format: %s", path)
		}
		return &ConfigFile{Config: ruleCfg}, nil
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

	if cfg.Extends == "" {
		return &cfg, nil
	}

	// Resolve the extends path relative to the directory of the current config file.
	extendsPath := cfg.Extends
	if !filepath.IsAbs(extendsPath) {
		extendsPath = filepath.Join(filepath.Dir(absPath), extendsPath)
	}

	baseCfg, err := loadConfigResolved(extendsPath, visited)
	if err != nil {
		return nil, fmt.Errorf("loading extends %q: %w", extendsPath, err)
	}

	// Merge: base config is the foundation; the current config overrides it.
	outputFormatters := baseCfg.OutputFormatters
	if len(cfg.OutputFormatters) > 0 {
		outputFormatters = cfg.OutputFormatters
	}
	// Globs: child overrides base when set; otherwise inherit base globs.
	globs := baseCfg.Globs
	if len(cfg.Globs) > 0 {
		globs = cfg.Globs
	}
	// FrontMatter: child overrides base when set.
	frontMatter := baseCfg.FrontMatter
	if cfg.FrontMatter != "" {
		frontMatter = cfg.FrontMatter
	}
	merged := &ConfigFile{
		Globs:            globs,
		Fix:              baseCfg.Fix || cfg.Fix,
		FrontMatter:      frontMatter,
		Gitignore:        mergeGitignore(baseCfg.Gitignore, cfg.Gitignore),
		Config:           mergeConfigs(baseCfg.Config, cfg.Config),
		Ignores:          append(baseCfg.Ignores, cfg.Ignores...),
		Overrides:        append(baseCfg.Overrides, cfg.Overrides...),
		OutputFormatters: outputFormatters,
	}
	return merged, nil
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
		case string:
			return true
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

// getRuleSeverity returns "warning" if the rule is configured with "warning"
// severity, otherwise "error".
func getRuleSeverity(id string, cfg map[string]interface{}) string {
	if val, ok := cfg[id]; ok {
		if s, ok := val.(string); ok && strings.ToLower(s) == "warning" {
			return "warning"
		}
	}
	return "error"
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

// ruleInfo holds metadata about a rule for the --list-rules display.
type ruleInfo struct {
	rule    lint.Rule
	enabled bool
}

// ruleFactory pairs a rule ID with a factory function that creates the rule
// with config applied.
type ruleFactory struct {
	id      string
	factory func() lint.Rule
}

// makeRuleFactories returns the ordered list of rule factories, each of which
// creates a rule instance with options applied from cfg.
func makeRuleFactories(cfg map[string]interface{}) []ruleFactory {
	return []ruleFactory{
		{"MD001", func() lint.Rule { r := &rules.MD001{}; applyRuleConfig(r, cfg, "MD001"); return r }},
		{"MD003", func() lint.Rule { r := &rules.MD003{}; applyRuleConfig(r, cfg, "MD003"); return r }},
		{"MD004", func() lint.Rule { r := &rules.MD004{}; applyRuleConfig(r, cfg, "MD004"); return r }},
		{"MD005", func() lint.Rule { r := &rules.MD005{}; applyRuleConfig(r, cfg, "MD005"); return r }},
		{"MD007", func() lint.Rule { r := &rules.MD007{}; applyRuleConfig(r, cfg, "MD007"); return r }},
		{"MD009", func() lint.Rule { r := &rules.MD009{}; applyRuleConfig(r, cfg, "MD009"); return r }},
		{"MD010", func() lint.Rule { r := &rules.MD010{}; applyRuleConfig(r, cfg, "MD010"); return r }},
		{"MD011", func() lint.Rule { r := &rules.MD011{}; applyRuleConfig(r, cfg, "MD011"); return r }},
		{"MD012", func() lint.Rule { r := &rules.MD012{}; applyRuleConfig(r, cfg, "MD012"); return r }},
		{"MD013", func() lint.Rule { r := &rules.MD013{}; applyRuleConfig(r, cfg, "MD013"); return r }},
		{"MD014", func() lint.Rule { r := &rules.MD014{}; applyRuleConfig(r, cfg, "MD014"); return r }},
		{"MD018", func() lint.Rule { r := &rules.MD018{}; applyRuleConfig(r, cfg, "MD018"); return r }},
		{"MD019", func() lint.Rule { r := &rules.MD019{}; applyRuleConfig(r, cfg, "MD019"); return r }},
		{"MD020", func() lint.Rule { r := &rules.MD020{}; applyRuleConfig(r, cfg, "MD020"); return r }},
		{"MD021", func() lint.Rule { r := &rules.MD021{}; applyRuleConfig(r, cfg, "MD021"); return r }},
		{"MD022", func() lint.Rule { r := &rules.MD022{}; applyRuleConfig(r, cfg, "MD022"); return r }},
		{"MD023", func() lint.Rule { r := &rules.MD023{}; applyRuleConfig(r, cfg, "MD023"); return r }},
		{"MD024", func() lint.Rule { r := &rules.MD024{}; applyRuleConfig(r, cfg, "MD024"); return r }},
		{"MD025", func() lint.Rule { r := &rules.MD025{}; applyRuleConfig(r, cfg, "MD025"); return r }},
		{"MD026", func() lint.Rule { r := &rules.MD026{}; applyRuleConfig(r, cfg, "MD026"); return r }},
		{"MD027", func() lint.Rule { r := &rules.MD027{}; applyRuleConfig(r, cfg, "MD027"); return r }},
		{"MD028", func() lint.Rule { r := &rules.MD028{}; applyRuleConfig(r, cfg, "MD028"); return r }},
		{"MD029", func() lint.Rule { r := &rules.MD029{}; applyRuleConfig(r, cfg, "MD029"); return r }},
		{"MD030", func() lint.Rule { r := &rules.MD030{}; applyRuleConfig(r, cfg, "MD030"); return r }},
		{"MD031", func() lint.Rule { r := &rules.MD031{}; applyRuleConfig(r, cfg, "MD031"); return r }},
		{"MD032", func() lint.Rule { r := &rules.MD032{}; applyRuleConfig(r, cfg, "MD032"); return r }},
		{"MD033", func() lint.Rule { r := &rules.MD033{}; applyRuleConfig(r, cfg, "MD033"); return r }},
		{"MD034", func() lint.Rule { r := &rules.MD034{}; applyRuleConfig(r, cfg, "MD034"); return r }},
		{"MD035", func() lint.Rule { r := &rules.MD035{}; applyRuleConfig(r, cfg, "MD035"); return r }},
		{"MD036", func() lint.Rule { r := &rules.MD036{}; applyRuleConfig(r, cfg, "MD036"); return r }},
		{"MD037", func() lint.Rule { r := &rules.MD037{}; applyRuleConfig(r, cfg, "MD037"); return r }},
		{"MD038", func() lint.Rule { r := &rules.MD038{}; applyRuleConfig(r, cfg, "MD038"); return r }},
		{"MD039", func() lint.Rule { r := &rules.MD039{}; applyRuleConfig(r, cfg, "MD039"); return r }},
		{"MD040", func() lint.Rule { r := &rules.MD040{}; applyRuleConfig(r, cfg, "MD040"); return r }},
		{"MD041", func() lint.Rule { r := &rules.MD041{}; applyRuleConfig(r, cfg, "MD041"); return r }},
		{"MD042", func() lint.Rule { r := &rules.MD042{}; applyRuleConfig(r, cfg, "MD042"); return r }},
		{"MD043", func() lint.Rule { r := &rules.MD043{}; applyRuleConfig(r, cfg, "MD043"); return r }},
		{"MD044", func() lint.Rule { r := &rules.MD044{}; applyRuleConfig(r, cfg, "MD044"); return r }},
		{"MD045", func() lint.Rule { r := &rules.MD045{}; applyRuleConfig(r, cfg, "MD045"); return r }},
		{"MD046", func() lint.Rule { r := &rules.MD046{}; applyRuleConfig(r, cfg, "MD046"); return r }},
		{"MD047", func() lint.Rule { r := &rules.MD047{}; applyRuleConfig(r, cfg, "MD047"); return r }},
		{"MD048", func() lint.Rule { r := &rules.MD048{}; applyRuleConfig(r, cfg, "MD048"); return r }},
		{"MD049", func() lint.Rule { r := &rules.MD049{}; applyRuleConfig(r, cfg, "MD049"); return r }},
		{"MD050", func() lint.Rule { r := &rules.MD050{}; applyRuleConfig(r, cfg, "MD050"); return r }},
		{"MD051", func() lint.Rule { r := &rules.MD051{}; applyRuleConfig(r, cfg, "MD051"); return r }},
		{"MD052", func() lint.Rule { r := &rules.MD052{}; applyRuleConfig(r, cfg, "MD052"); return r }},
		{"MD053", func() lint.Rule { r := &rules.MD053{}; applyRuleConfig(r, cfg, "MD053"); return r }},
		{"MD054", func() lint.Rule { r := &rules.MD054{}; applyRuleConfig(r, cfg, "MD054"); return r }},
		{"MD055", func() lint.Rule { r := &rules.MD055{}; applyRuleConfig(r, cfg, "MD055"); return r }},
		{"MD056", func() lint.Rule { r := &rules.MD056{}; applyRuleConfig(r, cfg, "MD056"); return r }},
		{"MD058", func() lint.Rule { r := &rules.MD058{}; applyRuleConfig(r, cfg, "MD058"); return r }},
		{"MD059", func() lint.Rule { r := &rules.MD059{}; applyRuleConfig(r, cfg, "MD059"); return r }},
		{"MD060", func() lint.Rule { r := &rules.MD060{}; applyRuleConfig(r, cfg, "MD060"); return r }},
	}
}

// buildAllRulesInfo returns metadata for every known rule, regardless of whether
// it is enabled or disabled in cfg.  The enabled field reflects the effective
// enabled/disabled state according to cfg.
func buildAllRulesInfo(cfg map[string]interface{}) []ruleInfo {
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	factories := makeRuleFactories(cfg)
	infos := make([]ruleInfo, 0, len(factories))
	for _, f := range factories {
		infos = append(infos, ruleInfo{
			rule:    f.factory(),
			enabled: isRuleEnabled(f.id, cfg),
		})
	}
	return infos
}

// buildRules constructs the list of lint rules based on the provided config map.
// If cfg is nil, all rules are enabled with their default options.
func buildRules(cfg map[string]interface{}) []lint.Rule {
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	factories := makeRuleFactories(cfg)
	var result []lint.Rule
	for _, f := range factories {
		if isRuleEnabled(f.id, cfg) {
			result = append(result, f.factory())
		}
	}
	return result
}

// mergeConfigs returns a new config map with entries from overlay deep-merged
// on top of base. When both base and overlay have the same key with map values,
// the maps are recursively merged so that sub-keys not present in overlay are
// preserved from base.
func mergeConfigs(base, overlay map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(base)+len(overlay))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range overlay {
		if baseVal, ok := merged[k]; ok {
			if baseMap, ok := baseVal.(map[string]interface{}); ok {
				if ovMap, ok := v.(map[string]interface{}); ok {
					merged[k] = mergeConfigs(baseMap, ovMap)
					continue
				}
			}
		}
		merged[k] = v
	}
	return merged
}

// matchesAnyPattern reports whether path matches any of the given glob patterns.
func matchesAnyPattern(path string, patterns []string) bool {
	normalized := filepath.ToSlash(filepath.Clean(path))
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)
		if matchPath(pattern, normalized) {
			return true
		}
	}
	return false
}

// effectiveConfigForFile returns the rule-config map to use when linting the
// given file.  It starts from base and applies every override whose Files
// patterns match the file path (in declaration order, last override wins).
func effectiveConfigForFile(base map[string]interface{}, overrides []GlobOverride, filePath string) map[string]interface{} {
	cfg := base
	for _, ov := range overrides {
		if matchesAnyPattern(filePath, ov.Files) {
			cfg = mergeConfigs(cfg, ov.Config)
		}
	}
	return cfg
}

// isIgnored reports whether path matches any of the ignore glob patterns.
func isIgnored(path string, patterns []string) bool {
	return matchesAnyPattern(path, patterns)
}

// gitignoreIsEnabled reports whether the gitignore config value is enabled
// (either bool true or a non-empty string glob pattern).
func gitignoreIsEnabled(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	}
	return false
}

// gitignoreGlobPattern returns the explicit glob pattern when v is a non-empty
// string, or "" when v is bool true (meaning: use the default walk-to-git-root logic).
func gitignoreGlobPattern(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// mergeGitignore combines base and child gitignore values: if child is enabled
// (bool true or non-empty string) the child value takes precedence; otherwise
// the base value is preserved. This mirrors the existing bool || semantics.
func mergeGitignore(base, child interface{}) interface{} {
	if gitignoreIsEnabled(child) {
		return child
	}
	return base
}

// findGitRoot walks up from dir to find the git repository root (the directory
// containing a .git entry). Returns "" if not found.
func findGitRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// collectGitignorePatterns reads all .gitignore files from cwd up to the git
// repository root and returns the combined list of ignore patterns.
func collectGitignorePatterns(cwd string) []string {
	gitRoot := findGitRoot(cwd)
	var patterns []string
	dir := cwd
	for {
		patterns = append(patterns, parseGitignore(filepath.Join(dir, ".gitignore"))...)
		if dir == gitRoot || gitRoot == "" {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return patterns
}

// findFilesMatchingGlob walks root and returns the absolute paths of all files
// whose path relative to root matches the given glob pattern (supports **).
// Walk errors (e.g. permission denied) are ignored; any accessible matches are returned.
func findFilesMatchingGlob(root, pattern string) []string {
	var matches []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if matchPath(pattern, filepath.ToSlash(rel)) {
			matches = append(matches, path)
		}
		return nil
	})
	return matches
}

// parseGitignore reads a .gitignore file and returns glob patterns for files
// that should be ignored. Comment lines (starting with #) and empty lines are
// skipped. Negation patterns (starting with !) are also skipped.
func parseGitignore(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
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
