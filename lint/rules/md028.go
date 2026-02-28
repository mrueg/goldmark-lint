package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD028 checks for blank lines inside blockquotes.
type MD028 struct{}

func (r MD028) ID() string          { return "MD028" }
func (r MD028) Aliases() []string   { return []string{"no-blanks-blockquote"} }
func (r MD028) Description() string { return "Blank line inside blockquote" }

// isBlockquoteLine reports whether the line is a blockquote line (starts with '>').
func isBlockquoteLine(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " "), ">")
}

func (r MD028) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	n := len(doc.Lines)

	for i := 1; i < n-1; i++ {
		if mask[i] {
			continue
		}
		// Only check blank lines.
		if strings.TrimSpace(doc.Lines[i]) != "" {
			continue
		}
		// The line before must be a blockquote line (non-blank).
		if !isBlockquoteLine(doc.Lines[i-1]) {
			continue
		}
		// The line after must be a blockquote line.
		if !isBlockquoteLine(doc.Lines[i+1]) {
			continue
		}
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    i + 1,
			Column:  1,
			Message: "Blank line inside blockquote",
		})
	}
	return violations
}
