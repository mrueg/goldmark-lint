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

// Fix applies MD022 to source by adding blank lines around headings.
func (r MD022) Fix(source []byte) []byte {
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

	lines := strings.Split(string(source), "\n")
	// Build fenced code block mask.
	mask := fencedCodeBlockMask(lines)

	// Identify heading lines and their levels.
	// Only ATX headings are fixed (setext headings are more complex).
	type headingInfo struct {
		idx   int
		level int
	}
	var headings []headingInfo
	for i, line := range lines {
		if mask[i] {
			continue
		}
		// ATX heading: optional leading spaces (up to 3), then 1-6 '#' chars, then space.
		stripped := strings.TrimLeft(line, " ")
		if len(stripped) == 0 || stripped[0] != '#' {
			// Check for setext heading (= or - underline).
			continue
		}
		// Count '#' chars.
		j := 0
		for j < len(stripped) && stripped[j] == '#' {
			j++
		}
		if j > 6 {
			continue
		}
		if j < len(stripped) && stripped[j] != ' ' {
			continue
		}
		headings = append(headings, headingInfo{i, j})
	}

	// Process headings in reverse order to preserve indices.
	for k := len(headings) - 1; k >= 0; k-- {
		h := headings[k]
		linesAbove := linesAboveFor(h.level)
		linesBelow := linesBelowFor(h.level)

		// Determine the end of the heading (for setext headings there's an underline, but
		// ATX headings are single-line).
		headEnd := h.idx

		// Add blank lines below if needed.
		if headEnd+1 < len(lines) {
			blankBelow := 0
			for j := headEnd + 1; j < len(lines) && strings.TrimSpace(lines[j]) == ""; j++ {
				blankBelow++
			}
			if blankBelow < linesBelow {
				// Insert blank lines after the heading.
				insert := make([]string, linesBelow-blankBelow)
				newLines := make([]string, 0, len(lines)+(linesBelow-blankBelow))
				newLines = append(newLines, lines[:headEnd+1]...)
				newLines = append(newLines, insert...)
				newLines = append(newLines, lines[headEnd+1:]...)
				lines = newLines
				// Rebuild mask.
				mask = fencedCodeBlockMask(lines)
			}
		}

		// Add blank lines above if needed (only if this is not the first line).
		if h.idx > 0 {
			blankAbove := 0
			for j := h.idx - 1; j >= 0 && strings.TrimSpace(lines[j]) == ""; j-- {
				blankAbove++
			}
			if blankAbove < linesAbove {
				// Insert blank lines before the heading.
				insert := make([]string, linesAbove-blankAbove)
				newLines := make([]string, 0, len(lines)+(linesAbove-blankAbove))
				newLines = append(newLines, lines[:h.idx]...)
				newLines = append(newLines, insert...)
				newLines = append(newLines, lines[h.idx:]...)
				lines = newLines
				// Rebuild mask.
				mask = fencedCodeBlockMask(lines)
			}
		}
	}

	result := strings.Join(lines, "\n")
	if result == string(source) {
		return source
	}
	return []byte(result)
}

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
	htmlMask := htmlBlockLineMask(doc)

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
				// Suppress the "above" violation if the nearest non-blank preceding line
				// is an HTML block (e.g. <!-- comment -->).  Markdownlint does not require
				// a blank line between an HTML comment block and the heading below it.
				// Check only the immediately preceding non-blank line.
				prevIdx := lineIdx - 1
				precededByHTML := prevIdx >= 0 &&
					strings.TrimSpace(lines[prevIdx]) != "" &&
					htmlMask[prevIdx]
				if !precededByHTML {
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    lineNum,
						Column:  1,
						Message: "Headings should be surrounded by blank lines [Expected: 1; Actual: 0; Above]",
					})
				}
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
