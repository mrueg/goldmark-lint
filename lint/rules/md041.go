package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD041 checks that the first line in a file is a top-level heading.
type MD041 struct {
	// Level is the required top-level heading level (default 1).
	Level int `json:"level"`
}

func (r MD041) ID() string          { return "MD041" }
func (r MD041) Description() string { return "First line in a file should be a top-level heading" }

func (r MD041) Check(doc *lint.Document) []lint.Violation {
	level := r.Level
	if level == 0 {
		level = 1
	}
	if len(doc.Lines) == 0 {
		return nil
	}

	for i, line := range doc.Lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		prefix := strings.Repeat("#", level) + " "
		if strings.HasPrefix(line, prefix) {
			return nil
		}
		return []lint.Violation{{
			Rule:    r.ID(),
			Line:    i + 1,
			Column:  1,
			Message: "First line in a file should be a top-level heading",
		}}
	}
	return nil
}
