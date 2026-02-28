package lint

import (
	"bytes"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Rule defines the interface for a lint rule.
type Rule interface {
	ID() string
	Description() string
	Check(doc *Document) []Violation
}

// FixableRule is an optional interface for rules that support auto-fixing.
type FixableRule interface {
	Rule
	Fix(source []byte) []byte
}

// AliasedRule is an optional interface for rules that have human-readable
// aliases (e.g. "heading-increment" for MD001), matching markdownlint aliases.
type AliasedRule interface {
	Rule
	Aliases() []string
}

// Violation represents a lint violation found in a document.
type Violation struct {
	Rule     string
	Line     int
	Column   int
	Message  string
	Severity string // "error" or "warning"; defaults to "error" when empty
}

// Document holds the parsed markdown document along with source.
type Document struct {
	Source            []byte
	Lines             []string
	AST               ast.Node
	FrontMatterFields map[string]string // key-value pairs from YAML front matter, if any
}

// Linter holds the list of rules and runs them on documents.
type Linter struct {
	Rules              []Rule
	aliasMap           map[string]string // upper(alias) → canonical rule ID
	NoInlineConfig     bool
	FrontMatterRegexp  *regexp.Regexp    // custom front matter pattern; nil uses default
}

// NewLinter creates a new Linter with the given rules.
func NewLinter(rules ...Rule) *Linter {
	aliasMap := make(map[string]string)
	for _, r := range rules {
		if ar, ok := r.(AliasedRule); ok {
			for _, alias := range ar.Aliases() {
				aliasMap[strings.ToUpper(alias)] = r.ID()
			}
		}
	}
	return &Linter{Rules: rules, aliasMap: aliasMap}
}

// resolveRuleID normalises name to a canonical rule ID. If name matches a known
// alias (case-insensitively), the corresponding rule ID is returned; otherwise
// name is returned uppercased (IDs are always stored uppercased).
func (l *Linter) resolveRuleID(name string) string {
	upper := strings.ToUpper(name)
	if id, ok := l.aliasMap[upper]; ok {
		return id
	}
	return upper
}

// fmEnd returns the byte offset of the end of the front matter block in source,
// using the custom FrontMatterRegexp if set, otherwise the default YAML --- detection.
func (l *Linter) fmEnd(source []byte) int {
	if l.FrontMatterRegexp != nil {
		loc := l.FrontMatterRegexp.FindIndex(source)
		if loc != nil && loc[0] == 0 {
			return loc[1]
		}
		return 0
	}
	return frontMatterEnd(source)
}

// Fix applies all fixable rules to source and returns the corrected content.
// Front matter is preserved unchanged.
func (l *Linter) Fix(source []byte) []byte {
	fmEnd := l.fmEnd(source)
	rest := source[fmEnd:]
	for _, rule := range l.Rules {
		if fixable, ok := rule.(FixableRule); ok {
			rest = fixable.Fix(rest)
		}
	}
	return append(source[:fmEnd:fmEnd], rest...)
}

// Lint parses source and runs all rules on it, returning violations sorted by line.
func (l *Linter) Lint(source []byte) []Violation {
	end := l.fmEnd(source)
	fmFields := parseFrontMatterFieldsAt(source, end)
	source = stripFrontMatterAt(source, end)

	reader := text.NewReader(source)
	md := goldmark.New()
	node := md.Parser().Parse(reader)

	lines := splitLines(source)
	doc := &Document{
		Source:            source,
		Lines:             lines,
		AST:               node,
		FrontMatterFields: fmFields,
	}

	var disabled []disableSet
	if !l.NoInlineConfig {
		disabled = parseInlineDisables(lines, l.resolveRuleID)
	}

	var violations []Violation
	for _, rule := range l.Rules {
		for _, v := range rule.Check(doc) {
			idx := v.Line - 1 // convert 1-based to 0-based
			if idx < len(disabled) && disabled[idx].contains(v.Rule) {
				continue
			}
			violations = append(violations, v)
		}
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Line != violations[j].Line {
			return violations[i].Line < violations[j].Line
		}
		return violations[i].Rule < violations[j].Rule
	})

	return violations
}

// parseFrontMatterFields parses the YAML front matter of source and returns a
// map of simple "key: value" pairs. Only scalar (non-nested) fields are supported.
func parseFrontMatterFields(source []byte) map[string]string {
	return parseFrontMatterFieldsAt(source, frontMatterEnd(source))
}

// parseFrontMatterFieldsAt parses YAML front matter up to end bytes in source.
func parseFrontMatterFieldsAt(source []byte, end int) map[string]string {
	fields := make(map[string]string)
	if end == 0 {
		return fields
	}
	// Skip the opening "---\n" line.
	start := bytes.IndexByte(source, '\n')
	if start < 0 {
		return fields
	}
	start++ // skip past opening newline
	for start < end {
		next := bytes.IndexByte(source[start:], '\n')
		var lineEnd int
		if next < 0 {
			lineEnd = end
		} else {
			lineEnd = start + next
		}
		line := strings.TrimRight(string(source[start:lineEnd]), "\r")
		if line == "---" || line == "..." {
			break
		}
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			// Strip surrounding quotes.
			if len(val) >= 2 &&
				((val[0] == '"' && val[len(val)-1] == '"') ||
					(val[0] == '\'' && val[len(val)-1] == '\'')) {
				val = val[1 : len(val)-1]
			}
			fields[key] = val
		}
		if next < 0 {
			break
		}
		start = lineEnd + 1
	}
	return fields
}

// front matter block, or 0 if the source does not begin with valid front matter.
// Front matter starts with "---" on the very first line and ends with a line
// containing only "---" or "...".
func frontMatterEnd(source []byte) int {
	if !bytes.HasPrefix(source, []byte("---\n")) && !bytes.HasPrefix(source, []byte("---\r\n")) {
		return 0
	}
	// Advance past the opening delimiter line.
	pos := bytes.IndexByte(source, '\n')
	if pos < 0 {
		return 0
	}
	pos++ // skip the newline

	for pos < len(source) {
		next := bytes.IndexByte(source[pos:], '\n')
		var lineEnd int
		if next < 0 {
			lineEnd = len(source)
		} else {
			lineEnd = pos + next
		}
		line := source[pos:lineEnd]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if bytes.Equal(line, []byte("---")) || bytes.Equal(line, []byte("...")) {
			end := lineEnd
			if end < len(source) && source[end] == '\n' {
				end++
			}
			return end
		}
		if next < 0 {
			break
		}
		pos = lineEnd + 1
	}
	return 0
}

// stripFrontMatter returns a copy of source with the YAML front matter block
// replaced by blank lines, preserving line numbers so that violations reported
// by rules refer to the correct lines in the original file.
func stripFrontMatter(source []byte) []byte {
	return stripFrontMatterAt(source, frontMatterEnd(source))
}

// stripFrontMatterAt strips the front matter block up to end bytes in source.
func stripFrontMatterAt(source []byte, end int) []byte {
	if end == 0 {
		return source
	}
	// Build new source keeping only \r and \n from the front matter region.
	result := make([]byte, 0, len(source))
	for i := 0; i < end; i++ {
		if source[i] == '\n' || source[i] == '\r' {
			result = append(result, source[i])
		}
	}
	result = append(result, source[end:]...)
	return result
}

// splitLines splits source bytes into individual lines (without trailing newlines).
func splitLines(source []byte) []string {
	var lines []string
	start := 0
	for i, b := range source {
		if b == '\n' {
			line := string(source[start:i])
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start <= len(source) {
		lines = append(lines, string(source[start:]))
	}
	return lines
}

// markdownlintCommentRE matches markdownlint inline disable/enable comments.
// It captures the command and optional rule IDs.
// Rule names and aliases may contain hyphens (e.g. "heading-increment"), so
// [\w-]+ is used instead of \w+.
var markdownlintCommentRE = regexp.MustCompile(
	`<!--\s*markdownlint-(disable-next-line|disable-line|disable-file|enable-file|disable|enable|capture|restore)((?:\s+[\w-]+)*)\s*-->`,
)

// markdownlintConfigureFileRE matches markdownlint-configure-file comments
// and captures the JSON payload (which may span multiple lines).
var markdownlintConfigureFileRE = regexp.MustCompile(
	`(?s)<!--\s*markdownlint-configure-file\s*(\{.*?\})\s*-->`,
)

// parseMarkdownlintComment extracts the command and rule list from a markdownlint
// inline comment, if one is found anywhere on the given line.
func parseMarkdownlintComment(line string) (cmd string, ruleIDs []string) {
	m := markdownlintCommentRE.FindStringSubmatch(line)
	if m == nil {
		return "", nil
	}
	cmd = m[1]
	ruleIDs = append(ruleIDs, strings.Fields(m[2])...)
	return cmd, ruleIDs
}

// parseConfigureFileComment extracts the JSON payload from a markdownlint-configure-file
// comment, searching across the full source (which may be multi-line).
func parseConfigureFileComment(source string) string {
	m := markdownlintConfigureFileRE.FindStringSubmatch(source)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// applyConfigureFile parses a markdownlint-configure-file JSON payload and returns
// file-level disable/enable overrides. Values of false disable a rule; true enables it.
func applyConfigureFile(jsonPayload string, fileDis *disableSet, resolve func(string) string) {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(jsonPayload), &cfg); err != nil {
		return
	}
	for key, val := range cfg {
		id := resolve(key)
		switch v := val.(type) {
		case bool:
			if !v {
				fileDis.rules[id] = true
			} else {
				delete(fileDis.rules, id)
			}
		}
	}
}

// disableSet tracks which rules are disabled for a single line.
type disableSet struct {
	all   bool
	rules map[string]bool
}

// contains reports whether rule is suppressed.
func (d disableSet) contains(rule string) bool {
	return d.all || d.rules[rule]
}

// copyDisableSet returns a deep copy of d.
func copyDisableSet(d disableSet) disableSet {
	c := disableSet{all: d.all, rules: make(map[string]bool, len(d.rules))}
	for r := range d.rules {
		c.rules[r] = true
	}
	return c
}

// parseInlineDisables scans source lines for markdownlint inline disable
// comments and returns a per-line (0-based) slice of disableSet values.
// resolve maps a rule name (alias or ID) to its canonical uppercase rule ID.
func parseInlineDisables(lines []string, resolve func(string) string) []disableSet {
	n := len(lines)
	result := make([]disableSet, n)
	for i := range result {
		result[i] = disableSet{rules: make(map[string]bool)}
	}

	// First pass: collect file-level disable/enable commands (disable-file,
	// enable-file, configure-file), which apply to every line in the file.
	// fileDisable tracks disable-file/enable-file; fileConfig tracks configure-file.
	// They are kept separate so that enable-file does not undo configure-file settings.
	fileDisable := disableSet{rules: make(map[string]bool)}
	fileConfig := disableSet{rules: make(map[string]bool)}
	fullSource := strings.Join(lines, "\n")
	if payload := parseConfigureFileComment(fullSource); payload != "" {
		applyConfigureFile(payload, &fileConfig, resolve)
	}
	for _, line := range lines {
		cmd, ruleIDs := parseMarkdownlintComment(line)
		switch cmd {
		case "disable-file":
			if len(ruleIDs) == 0 {
				fileDisable.all = true
			} else {
				for _, r := range ruleIDs {
					fileDisable.rules[resolve(r)] = true
				}
			}
		case "enable-file":
			if len(ruleIDs) == 0 {
				fileDisable = disableSet{rules: make(map[string]bool)}
			} else {
				for _, r := range ruleIDs {
					delete(fileDisable.rules, resolve(r))
				}
			}
		}
	}

	current := disableSet{rules: make(map[string]bool)}
	var captured *disableSet
	// nextLineRules holds the extra disable state to apply to the next line.
	var nextLineExtra *disableSet

	for i, line := range lines {
		cmd, ruleIDs := parseMarkdownlintComment(line)

		switch cmd {
		case "disable":
			if len(ruleIDs) == 0 {
				current.all = true
			} else {
				for _, r := range ruleIDs {
					current.rules[resolve(r)] = true
				}
			}
		case "enable":
			if len(ruleIDs) == 0 {
				current = disableSet{rules: make(map[string]bool)}
			} else {
				for _, r := range ruleIDs {
					delete(current.rules, resolve(r))
				}
			}
		case "capture":
			c := copyDisableSet(current)
			captured = &c
		case "restore":
			if captured != nil {
				current = copyDisableSet(*captured)
			}
		}

		// Base state for this line comes from running state merged with file-level.
		result[i] = copyDisableSet(current)
		if fileDisable.all {
			result[i].all = true
		} else {
			for r := range fileDisable.rules {
				result[i].rules[r] = true
			}
		}
		for r := range fileConfig.rules {
			result[i].rules[r] = true
		}

		// Apply disable-next-line extras carried over from previous line.
		if nextLineExtra != nil {
			if nextLineExtra.all {
				result[i].all = true
			} else {
				for r := range nextLineExtra.rules {
					result[i].rules[r] = true
				}
			}
			nextLineExtra = nil
		}

		// Apply disable-line to the current line.
		if cmd == "disable-line" {
			if len(ruleIDs) == 0 {
				result[i].all = true
			} else {
				for _, r := range ruleIDs {
					result[i].rules[resolve(r)] = true
				}
			}
		}

		// Prepare disable-next-line extra for the next line.
		if cmd == "disable-next-line" {
			extra := disableSet{rules: make(map[string]bool)}
			if len(ruleIDs) == 0 {
				extra.all = true
			} else {
				for _, r := range ruleIDs {
					extra.rules[resolve(r)] = true
				}
			}
			nextLineExtra = &extra
		}
	}

	return result
}
