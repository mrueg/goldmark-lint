package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD058 checks that tables are surrounded by blank lines.
type MD058 struct{}

func (r MD058) ID() string          { return "MD058" }
func (r MD058) Description() string { return "Tables should be surrounded by blank lines" }

func (r MD058) Check(doc *lint.Document) []lint.Violation {
	mask := fencedCodeBlockMask(doc.Lines)
	tables := findTables(doc.Lines, mask)
	lines := doc.Lines
	var violations []lint.Violation

	for _, t := range tables {
		start, end := t[0], t[1]
		// Check blank line before (unless at document start).
		if start > 0 && strings.TrimSpace(lines[start-1]) != "" {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    start + 1,
				Column:  1,
				Message: "Tables should be surrounded by blank lines",
			})
		}
		// Check blank line after (unless at document end).
		if end < len(lines)-1 && strings.TrimSpace(lines[end+1]) != "" {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    end + 1,
				Column:  1,
				Message: "Tables should be surrounded by blank lines",
			})
		}
	}
	return violations
}

func (r MD058) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	tables := findTables(lines, mask)

	// Process tables from end to start to preserve line indices.
	for ti := len(tables) - 1; ti >= 0; ti-- {
		t := tables[ti]
		start, end := t[0], t[1]

		// Insert blank line after if needed.
		if end < len(lines)-1 && strings.TrimSpace(lines[end+1]) != "" {
			after := make([]string, len(lines)+1)
			copy(after, lines[:end+1])
			after[end+1] = ""
			copy(after[end+2:], lines[end+1:])
			lines = after
		}
		// Insert blank line before if needed.
		if start > 0 && strings.TrimSpace(lines[start-1]) != "" {
			before := make([]string, len(lines)+1)
			copy(before, lines[:start])
			before[start] = ""
			copy(before[start+1:], lines[start:])
			lines = before
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
