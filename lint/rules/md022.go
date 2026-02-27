package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD022 checks that headings are surrounded by blank lines.
type MD022 struct {
	// LinesAbove is the number of blank lines required above (default 1).
	LinesAbove int `json:"lines_above"`
	// LinesBelow is the number of blank lines required below (default 1).
	LinesBelow int `json:"lines_below"`
}

func (r MD022) ID() string          { return "MD022" }
func (r MD022) Description() string { return "Headings should be surrounded by blank lines" }

func (r MD022) Check(doc *lint.Document) []lint.Violation {
	linesAbove := r.LinesAbove
	if linesAbove == 0 {
		linesAbove = 1
	}
	linesBelow := r.LinesBelow
	if linesBelow == 0 {
		linesBelow = 1
	}

	var violations []lint.Violation
	lines := doc.Lines
	n := len(lines)

	for i, line := range lines {
		if !isHeadingLine(line) {
			continue
		}
		lineNum := i + 1

		if i > 0 {
			blankAbove := countBlankLinesAbove(lines, i)
			if blankAbove < linesAbove {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    lineNum,
					Column:  1,
					Message: "Headings should be surrounded by blank lines [Expected: 1; Actual: 0; Above]",
				})
			}
		}

		if i < n-1 {
			blankBelow := countBlankLinesBelow(lines, i)
			if blankBelow < linesBelow {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    lineNum,
					Column:  1,
					Message: "Headings should be surrounded by blank lines [Expected: 1; Actual: 0; Below]",
				})
			}
		}
	}
	return violations
}

func isHeadingLine(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	if len(trimmed) == 0 {
		return false
	}
	if trimmed[0] != '#' {
		return false
	}
	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
	}
	if i > 6 {
		return false
	}
	return i == len(trimmed) || trimmed[i] == ' '
}

func countBlankLinesAbove(lines []string, idx int) int {
	count := 0
	for i := idx - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "" {
			count++
		} else {
			break
		}
	}
	return count
}

func countBlankLinesBelow(lines []string, idx int) int {
	count := 0
	for i := idx + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			count++
		} else {
			break
		}
	}
	return count
}
