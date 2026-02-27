package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD007 checks that unordered list items are indented correctly.
type MD007 struct {
	// Indent is the number of spaces per indentation level (default 2).
	Indent int `json:"indent"`
}

func (r MD007) ID() string          { return "MD007" }
func (r MD007) Description() string { return "Unordered list indentation" }

// unorderedListMarkers holds the valid unordered list marker bytes.
const unorderedListMarkers = "*-+"

func (r MD007) Check(doc *lint.Document) []lint.Violation {
	indent := r.Indent
	if indent == 0 {
		indent = 2
	}

	var violations []lint.Violation

	for i, line := range doc.Lines {
		// Count leading spaces.
		trimmed := strings.TrimLeft(line, " ")
		spaces := len(line) - len(trimmed)

		// Must be followed by an unordered marker and a space.
		if len(trimmed) < 2 {
			continue
		}
		if strings.IndexByte(unorderedListMarkers, trimmed[0]) == -1 || trimmed[1] != ' ' {
			continue
		}

		// Leading spaces must be a multiple of indent.
		if spaces%indent != 0 {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  spaces + 1,
				Message: fmt.Sprintf("Unordered list indentation [Expected: multiple of %d; Actual: %d]", indent, spaces),
			})
		}
	}

	return violations
}
