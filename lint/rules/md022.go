package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD022 checks that headings are surrounded by blank lines.
type MD022 struct {
	// LinesAbove is the number of blank lines required above (default 1).
	// Can be a single integer or an array of integers for per-level configuration
	// (index 0 = h1, index 1 = h2, …).
	LinesAbove IntOrArray `json:"lines_above"`
	// LinesBelow is the number of blank lines required below (default 1).
	// Can be a single integer or an array of integers for per-level configuration.
	LinesBelow IntOrArray `json:"lines_below"`
}

func (r MD022) ID() string          { return "MD022" }
func (r MD022) Aliases() []string   { return []string{"blanks-around-headings"} }
func (r MD022) Description() string { return "Headings should be surrounded by blank lines" }

func (r MD022) Check(doc *lint.Document) []lint.Violation {
	linesAboveFor := func(level int) int {
		v := r.LinesAbove.Get(level)
		if v == 0 {
			return 1
		}
		return v
	}
	linesBelowFor := func(level int) int {
		v := r.LinesBelow.Get(level)
		if v == 0 {
			return 1
		}
		return v
	}

	var violations []lint.Violation
	lines := doc.Lines
	n := len(lines)

	for i, line := range lines {
		if !isHeadingLine(line) {
			continue
		}
		lineNum := i + 1
		level := headingLevel(line)
		linesAbove := linesAboveFor(level)
		linesBelow := linesBelowFor(level)

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

// headingLevel returns the ATX heading level (1-6) for a heading line, or 0.
func headingLevel(line string) int {
	trimmed := strings.TrimLeft(line, " ")
	if len(trimmed) == 0 || trimmed[0] != '#' {
		return 0
	}
	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
	}
	if i > 6 {
		return 0
	}
	if i == len(trimmed) || trimmed[i] == ' ' {
		return i
	}
	return 0
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
