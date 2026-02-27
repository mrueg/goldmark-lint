package rules

import (
	"fmt"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD013 checks for lines that are too long.
type MD013 struct {
	// LineLength is the maximum line length (default 80).
	LineLength int
}

func (r MD013) ID() string          { return "MD013" }
func (r MD013) Description() string { return "Line length" }

func (r MD013) Check(doc *lint.Document) []lint.Violation {
	limit := r.LineLength
	if limit == 0 {
		limit = 80
	}
	var violations []lint.Violation
	for i, line := range doc.Lines {
		if len(line) > limit {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  limit + 1,
				Message: fmt.Sprintf("Line length [Expected: %d; Actual: %d]", limit, len(line)),
			})
		}
	}
	return violations
}
