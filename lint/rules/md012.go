package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD012 checks for multiple consecutive blank lines.
type MD012 struct {
	// Maximum is the maximum number of consecutive blank lines (default 1).
	Maximum int
}

func (r MD012) ID() string          { return "MD012" }
func (r MD012) Description() string { return "Multiple consecutive blank lines" }

func (r MD012) Check(doc *lint.Document) []lint.Violation {
	maximum := r.Maximum
	if maximum == 0 {
		maximum = 1
	}
	var violations []lint.Violation
	consecutive := 0
	for i, line := range doc.Lines {
		if strings.TrimSpace(line) == "" {
			consecutive++
			if consecutive > maximum {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  1,
					Message: fmt.Sprintf("Multiple consecutive blank lines [Expected: %d; Actual: %d]", maximum, consecutive),
				})
			}
		} else {
			consecutive = 0
		}
	}
	return violations
}
