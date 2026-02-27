package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD009 checks for trailing spaces at the end of lines.
type MD009 struct {
	// BrSpaces is the number of spaces allowed at end of line for hard line breaks (default 2).
	BrSpaces int
}

func (r MD009) ID() string          { return "MD009" }
func (r MD009) Description() string { return "Trailing spaces" }

func (r MD009) Check(doc *lint.Document) []lint.Violation {
	brSpaces := r.BrSpaces
	if brSpaces == 0 {
		brSpaces = 2
	}
	var violations []lint.Violation
	for i, line := range doc.Lines {
		trimmed := strings.TrimRight(line, " \t")
		trailing := line[len(trimmed):]
		if len(trailing) > 0 {
			if trailing == strings.Repeat(" ", brSpaces) {
				continue
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  len(trimmed) + 1,
				Message: fmt.Sprintf("Trailing spaces [Expected: 0 or %d; Actual: %d]", brSpaces, len(trailing)),
			})
		}
	}
	return violations
}
