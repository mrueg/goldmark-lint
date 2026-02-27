package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD055 checks that table pipe style is consistent.
type MD055 struct {
	// Style is "consistent" (default), "leading_and_trailing", "leading_only",
	// "no_leading_or_trailing", or "trailing_only".
	Style string `json:"style"`
}

func (r MD055) ID() string          { return "MD055" }
func (r MD055) Description() string { return "Table pipe style" }

func rowPipeStyle(line string) string {
	trimmed := strings.TrimSpace(line)
	hasLeading := strings.HasPrefix(trimmed, "|")
	hasTrailing := strings.HasSuffix(trimmed, "|")
	switch {
	case hasLeading && hasTrailing:
		return "leading_and_trailing"
	case hasLeading:
		return "leading_only"
	case hasTrailing:
		return "trailing_only"
	default:
		return "no_leading_or_trailing"
	}
}

func (r MD055) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	mask := fencedCodeBlockMask(doc.Lines)
	tables := findTables(doc.Lines, mask)
	var violations []lint.Violation
	firstStyle := ""

	for _, t := range tables {
		for row := t[0]; row <= t[1]; row++ {
			line := doc.Lines[row]
			actual := rowPipeStyle(line)
			expected := style
			if style == "consistent" {
				if firstStyle == "" {
					firstStyle = actual
				}
				expected = firstStyle
			}
			if actual != expected {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    row + 1,
					Column:  1,
					Message: fmt.Sprintf("Table pipe style [Expected: %s; Actual: %s]", expected, actual),
				})
			}
		}
	}
	return violations
}
