package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD032 checks that lists are surrounded by blank lines.
type MD032 struct{}

func (r MD032) ID() string          { return "MD032" }
func (r MD032) Description() string { return "Lists should be surrounded by blank lines" }

// listItemRE matches unordered or ordered list item lines.
var listItemRE = regexp.MustCompile(`^( *)(?:[-*+]|\d+\.) `)

func isListItemLine(line string) bool {
	return listItemRE.MatchString(line)
}

func (r MD032) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	n := len(lines)
	var result []string

	for i, line := range lines {
		if isListItemLine(line) {
			prevIsList := i > 0 && isListItemLine(lines[i-1])
			// Insert blank line before first item in a list
			if !prevIsList && i > 0 && strings.TrimSpace(lines[i-1]) != "" {
				result = append(result, "")
			}
			result = append(result, line)
			nextIsList := i < n-1 && isListItemLine(lines[i+1])
			// Insert blank line after last item in a list
			if !nextIsList && i < n-1 && strings.TrimSpace(lines[i+1]) != "" {
				result = append(result, "")
			}
		} else {
			result = append(result, line)
		}
	}
	return []byte(strings.Join(result, "\n"))
}

func (r MD032) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	lines := doc.Lines
	n := len(lines)

	for i, line := range lines {
		if !isListItemLine(line) {
			continue
		}
		prevIsList := i > 0 && isListItemLine(lines[i-1])
		// Check blank line before first item
		if !prevIsList && i > 0 && strings.TrimSpace(lines[i-1]) != "" {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "Lists should be surrounded by blank lines",
			})
		}
		nextIsList := i < n-1 && isListItemLine(lines[i+1])
		// Check blank line after last item
		if !nextIsList && i < n-1 && strings.TrimSpace(lines[i+1]) != "" {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "Lists should be surrounded by blank lines",
			})
		}
	}
	return violations
}
