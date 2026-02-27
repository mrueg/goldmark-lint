package lint

import (
	"bytes"
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

// Violation represents a lint violation found in a document.
type Violation struct {
	Rule    string
	Line    int
	Column  int
	Message string
}

// Document holds the parsed markdown document along with source.
type Document struct {
	Source []byte
	Lines  []string
	AST    ast.Node
}

// Linter holds the list of rules and runs them on documents.
type Linter struct {
	Rules []Rule
}

// NewLinter creates a new Linter with the given rules.
func NewLinter(rules ...Rule) *Linter {
	return &Linter{Rules: rules}
}

// Fix applies all fixable rules to source and returns the corrected content.
// Front matter is preserved unchanged.
func (l *Linter) Fix(source []byte) []byte {
	fmEnd := frontMatterEnd(source)
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
	source = stripFrontMatter(source)

	reader := text.NewReader(source)
	md := goldmark.New()
	node := md.Parser().Parse(reader)

	lines := splitLines(source)
	doc := &Document{
		Source: source,
		Lines:  lines,
		AST:    node,
	}

	disabled := parseInlineDisables(lines)

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

// frontMatterEnd returns the byte offset of the first byte after the YAML
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
	end := frontMatterEnd(source)
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
var markdownlintCommentRE = regexp.MustCompile(
	`<!--\s*markdownlint-(disable-next-line|disable-line|disable|enable|capture|restore)((?:\s+\w+)*)\s*-->`,
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
func parseInlineDisables(lines []string) []disableSet {
	n := len(lines)
	result := make([]disableSet, n)
	for i := range result {
		result[i] = disableSet{rules: make(map[string]bool)}
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
					current.rules[r] = true
				}
			}
		case "enable":
			if len(ruleIDs) == 0 {
				current = disableSet{rules: make(map[string]bool)}
			} else {
				for _, r := range ruleIDs {
					delete(current.rules, r)
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

		// Base state for this line comes from running state.
		result[i] = copyDisableSet(current)

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
					result[i].rules[r] = true
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
					extra.rules[r] = true
				}
			}
			nextLineExtra = &extra
		}
	}

	return result
}
