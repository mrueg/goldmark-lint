package lint

import (
	"sort"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Rule defines the interface for a lint rule.
type Rule interface {
	ID() string
	Alias() string
	Description() string
	Check(doc *Document) []Violation
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

// Lint parses source and runs all rules on it, returning violations sorted by line.
func (l *Linter) Lint(source []byte) []Violation {
	reader := text.NewReader(source)
	md := goldmark.New()
	node := md.Parser().Parse(reader)

	lines := splitLines(source)
	doc := &Document{
		Source: source,
		Lines:  lines,
		AST:    node,
	}

	var violations []Violation
	for _, rule := range l.Rules {
		violations = append(violations, rule.Check(doc)...)
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Line != violations[j].Line {
			return violations[i].Line < violations[j].Line
		}
		return violations[i].Rule < violations[j].Rule
	})

	return violations
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
