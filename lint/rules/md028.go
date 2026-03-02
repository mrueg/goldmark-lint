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

	i := 0
	for i < n {
		if mask[i] || !isBlockquoteLine(doc.Lines[i]) {
			i++
			continue
		}
		// Find end of this contiguous blockquote run.
		end := i
		for end+1 < n && !mask[end+1] && isBlockquoteLine(doc.Lines[end+1]) {
			end++
		}
		// Collect blank lines immediately after this blockquote run.
		j := end + 1
		var blanks []int
		for j < n && !mask[j] && strings.TrimSpace(doc.Lines[j]) == "" {
			blanks = append(blanks, j)
			j++
		}
		// If we immediately hit another blockquote, report each blank line.
		if len(blanks) > 0 && j < n && !mask[j] && isBlockquoteLine(doc.Lines[j]) {
			for _, bl := range blanks {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    bl + 1,
					Column:  1,
					Message: "Blank line inside blockquote",
				})
			}
		}
		i = end + 1
	}
	return violations
}
