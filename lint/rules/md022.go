package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
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

	_ = ast.Walk(doc.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := node.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		if h.Lines() == nil || h.Lines().Len() == 0 {
			return ast.WalkContinue, nil
		}

		seg := h.Lines().At(0)
		lineNum := countLine(doc.Source, seg.Start)
		lineIdx := lineNum - 1 // 0-based
		linesAbove := linesAboveFor(h.Level)
		linesBelow := linesBelowFor(h.Level)

		// Determine if this is a setext heading by checking whether the source
		// line starts with '#' (ATX) or not (setext uses an underline on next line).
		// Strip blockquote markers first so that headings inside blockquotes
		// (e.g. "> ## Title") are correctly identified as ATX.
		isATX := func() bool {
			if lineIdx >= len(lines) {
				return false
			}
			line := strings.TrimLeft(lines[lineIdx], " \t")
			// Strip any blockquote prefix characters.
			for len(line) > 0 && line[0] == '>' {
				line = line[1:]
				if len(line) > 0 && line[0] == ' ' {
					line = line[1:]
				}
			}
			return len(line) > 0 && line[0] == '#'
		}()

		// For setext headings the underline is on the following line; check blank
		// lines below the underline, not the text line.
		belowIdx := lineIdx
		if !isATX && belowIdx+1 < n {
			belowIdx = lineIdx + 1
		}

		if lineIdx > 0 {
			blankAbove := countBlankLinesAbove(lines, lineIdx)
			if blankAbove < linesAbove {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    lineNum,
					Column:  1,
					Message: "Headings should be surrounded by blank lines [Expected: 1; Actual: 0; Above]",
				})
			}
		}

		if belowIdx < n-1 {
			blankBelow := countBlankLinesBelow(lines, belowIdx)
			if blankBelow < linesBelow {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    lineNum,
					Column:  1,
					Message: "Headings should be surrounded by blank lines [Expected: 1; Actual: 0; Below]",
				})
			}
		}

		return ast.WalkContinue, nil
	})

	return violations
}

// isBlankOrBlockquoteBlank reports whether a line is blank (empty or whitespace only),
// or is a blank blockquote line (e.g. ">", "> ", ">> ", etc.).
// Such lines count as blank separators for the purposes of MD022.
func isBlankOrBlockquoteBlank(line string) bool {
	rest := strings.TrimLeft(line, " \t")
	// Strip any number of blockquote markers (with optional trailing space each).
	for len(rest) > 0 && rest[0] == '>' {
		rest = rest[1:]
		if len(rest) > 0 && rest[0] == ' ' {
			rest = rest[1:]
		}
	}
	return strings.TrimSpace(rest) == ""
}

func countBlankLinesAbove(lines []string, idx int) int {
	count := 0
	for i := idx - 1; i >= 0; i-- {
		if isBlankOrBlockquoteBlank(lines[i]) {
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
		if isBlankOrBlockquoteBlank(lines[i]) {
			count++
		} else {
			break
		}
	}
	return count
}
