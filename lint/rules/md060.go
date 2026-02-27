package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD060 checks table column style consistency.
type MD060 struct {
	// Style is "any" (default), "compact", "tight", or "aligned".
	Style            string `json:"style"`
	AlignedDelimiter bool   `json:"aligned_delimiter"`
}

func (r MD060) ID() string          { return "MD060" }
func (r MD060) Description() string { return "Table column style" }

func tableColumnStyle(line string) string {
	trimmed := strings.TrimPrefix(strings.TrimSpace(line), "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	cells := strings.Split(trimmed, "|")

	allSingleSpace := true
	allNoSpace := true
	for _, cell := range cells {
		if len(cell) == 0 {
			allSingleSpace = false
			continue
		}
		hasLeadingSpace := cell[0] == ' '
		hasTrailingSpace := cell[len(cell)-1] == ' '
		if hasLeadingSpace || hasTrailingSpace {
			allNoSpace = false
		}
		// compact: exactly one space before and after content
		if !hasLeadingSpace || !hasTrailingSpace {
			allSingleSpace = false
		} else {
			inner := cell[1 : len(cell)-1]
			if strings.HasPrefix(inner, " ") || strings.HasSuffix(inner, " ") {
				allSingleSpace = false
			}
		}
	}
	if allNoSpace {
		return "tight"
	}
	if allSingleSpace {
		return "compact"
	}
	return "other"
}

func (r MD060) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "any"
	}
	if style == "any" {
		return nil
	}

	mask := fencedCodeBlockMask(doc.Lines)
	tables := findTables(doc.Lines, mask)
	var violations []lint.Violation

	for _, t := range tables {
		for row := t[0]; row <= t[1]; row++ {
			line := doc.Lines[row]
			if isTableDelimiterRow(line) {
				continue
			}
			actual := tableColumnStyle(line)
			if actual != style && actual != "other" {
				// Only report if style doesn't match expected.
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    row + 1,
					Column:  1,
					Message: fmt.Sprintf("Table column style [Expected: %s; Actual: %s]", style, actual),
				})
			}
		}
	}
	return violations
}
