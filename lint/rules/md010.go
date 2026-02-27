package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD010 checks for hard tabs.
type MD010 struct{}

func (r MD010) ID() string          { return "MD010" }
func (r MD010) Description() string { return "Hard tabs" }

func (r MD010) Fix(source []byte) []byte {
	return []byte(strings.ReplaceAll(string(source), "\t", "    "))
}

func (r MD010) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	for i, line := range doc.Lines {
		col := strings.Index(line, "\t")
		if col >= 0 {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  col + 1,
				Message: "Hard tabs",
			})
		}
	}
	return violations
}
