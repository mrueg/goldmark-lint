package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD060 checks table column style consistency.
type MD060 struct {
	// Style is "any" (default), "compact", "tight", "aligned", or "consistent".
	Style            string `json:"style"`
	AlignedDelimiter bool   `json:"aligned_delimiter"`
}

func (r MD060) ID() string          { return "MD060" }
func (r MD060) Aliases() []string   { return []string{"table-column-style"} }
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
		} else if len(cell) < 2 {
			// single-space cell has no content; not compact
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

// md60PipePositions returns the 0-based rune (character) positions of all '|' characters in line.
// Using rune positions ensures correct alignment comparison when lines contain multi-byte
// Unicode characters (e.g., emoji, CJK, ✓), matching markdownlint behaviour.
func md60PipePositions(line string) []int {
	var positions []int
	runeIdx := 0
	for _, r := range line {
		if r == '|' {
			positions = append(positions, runeIdx)
		}
		runeIdx++
	}
	return positions
}

// tableAlignedViolations returns violations for the "aligned" style: each non-header
// row must have pipe characters at the same column positions as the header row.
// Note: column positions are character-based (not visual width), so this approximation
// works well for ASCII text but may differ for files containing emoji or CJK characters.
func tableAlignedViolations(lines []string, t [2]int, ruleID string) []lint.Violation {
	headerPipes := md60PipePositions(lines[t[0]])
	headerSet := make(map[int]bool, len(headerPipes))
	for _, p := range headerPipes {
		headerSet[p] = true
	}

	var violations []lint.Violation
	for row := t[0] + 1; row <= t[1]; row++ {
		remaining := make(map[int]bool, len(headerSet))
		for p := range headerSet {
			remaining[p] = true
		}
		rowPipes := md60PipePositions(lines[row])
		for _, p := range rowPipes {
			if len(remaining) > 0 && !remaining[p] {
				violations = append(violations, lint.Violation{
					Rule:    ruleID,
					Line:    row + 1,
					Column:  p + 1,
					Message: `Table column style [Expected: aligned; Actual: not aligned]`,
				})
			}
			delete(remaining, p)
		}
	}
	return violations
}

// tableCompactTightViolations returns compact and tight violations for all rows in a table
// (including the delimiter row). Compact requires exactly 1 space on each side of cell
// content; tight requires no spaces.
func tableCompactTightViolations(lines []string, t [2]int, ruleID string) (compact, tight []lint.Violation) {
	for row := t[0]; row <= t[1]; row++ {
		c, ti := rowCompactTightViolations(lines[row], ruleID, row+1)
		compact = append(compact, c...)
		tight = append(tight, ti...)
	}
	return
}

// rowCompactTightViolations returns compact and tight violations for a single table row.
// For each '|' divider:
//   - The left side is skipped for the leading edge pipe.
//   - The right side is skipped for the trailing edge pipe or when only whitespace follows.
//   - 0 spaces on a side: compact error (missing space); tight OK.
//   - 1 space on a side: compact OK; tight error (any space).
//   - >1 spaces on a side: compact error (extra space); tight error.
func rowCompactTightViolations(line, ruleID string, lineNum int) (compact, tight []lint.Violation) {
	n := len(line)
	var pipes []int
	for i := 0; i < n; i++ {
		if line[i] == '|' {
			pipes = append(pipes, i)
		}
	}
	if len(pipes) == 0 {
		return
	}
	trimmed := strings.TrimSpace(line)
	hasLeadingPipe := len(trimmed) > 0 && trimmed[0] == '|'
	hasTrailingPipe := len(trimmed) > 0 && trimmed[len(trimmed)-1] == '|'

	for idx, p := range pipes {
		isLeadingEdge := hasLeadingPipe && idx == 0
		isTrailingEdge := hasTrailingPipe && idx == len(pipes)-1

		// Left check (skip for leading edge pipe).
		if !isLeadingEdge {
			leftSpaces := 0
			for j := p - 1; j >= 0 && line[j] == ' '; j-- {
				leftSpaces++
			}
			switch leftSpaces {
			case 0:
				compact = append(compact, lint.Violation{Rule: ruleID, Line: lineNum, Column: p + 1,
					Message: `Table column style [Expected: compact; Actual: missing space to left of pipe]`})
			case 1:
				tight = append(tight, lint.Violation{Rule: ruleID, Line: lineNum, Column: p + 1,
					Message: `Table column style [Expected: tight; Actual: space to left of pipe]`})
			default:
				compact = append(compact, lint.Violation{Rule: ruleID, Line: lineNum, Column: p + 1,
					Message: `Table column style [Expected: compact; Actual: extra space to left of pipe]`})
				tight = append(tight, lint.Violation{Rule: ruleID, Line: lineNum, Column: p + 1,
					Message: `Table column style [Expected: tight; Actual: space to left of pipe]`})
			}
		}

		// Right check: skip for trailing edge pipe and for pipes followed only by whitespace.
		if !isTrailingEdge {
			j := p + 1
			for j < n && line[j] == ' ' {
				j++
			}
			if j >= n {
				// Only whitespace (or nothing) follows until end of line: skip right check.
				continue
			}
			rightSpaces := j - (p + 1)
			switch rightSpaces {
			case 0:
				compact = append(compact, lint.Violation{Rule: ruleID, Line: lineNum, Column: p + 1,
					Message: `Table column style [Expected: compact; Actual: missing space to right of pipe]`})
			case 1:
				tight = append(tight, lint.Violation{Rule: ruleID, Line: lineNum, Column: p + 1,
					Message: `Table column style [Expected: tight; Actual: space to right of pipe]`})
			default:
				compact = append(compact, lint.Violation{Rule: ruleID, Line: lineNum, Column: p + 1,
					Message: `Table column style [Expected: compact; Actual: extra space to right of pipe]`})
				tight = append(tight, lint.Violation{Rule: ruleID, Line: lineNum, Column: p + 1,
					Message: `Table column style [Expected: tight; Actual: space to right of pipe]`})
			}
		}
	}
	return
}

func (r MD060) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	mask := fencedCodeBlockMask(doc.Lines)
	tables := findTables(doc.Lines, mask)
	var violations []lint.Violation

	switch style {
	case "any":
		for _, t := range tables {
			alignedErrs := tableAlignedViolations(doc.Lines, t, r.ID())
			if len(alignedErrs) == 0 {
				// Table is perfectly aligned: no violations.
				continue
			}
			compactErrs, tightErrs := tableCompactTightViolations(doc.Lines, t, r.ID())
			// Report violations for whichever style has the strictly fewest errors.
			// On equal counts, earlier-checked styles win: aligned > compact > tight.
			chosen := alignedErrs
			if len(compactErrs) < len(chosen) {
				chosen = compactErrs
			}
			if len(tightErrs) < len(chosen) {
				chosen = tightErrs
			}
			violations = append(violations, chosen...)
		}
	case "aligned":
		for _, t := range tables {
			violations = append(violations, tableAlignedViolations(doc.Lines, t, r.ID())...)
		}
	case "compact", "tight":
		for _, t := range tables {
			for row := t[0]; row <= t[1]; row++ {
				line := doc.Lines[row]
				if isTableDelimiterRow(line) {
					continue
				}
				actual := tableColumnStyle(line)
				if actual == "other" {
					continue
				}
				if actual != style {
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    row + 1,
						Column:  1,
						Message: fmt.Sprintf("Table column style [Expected: %s; Actual: %s]", style, actual),
					})
				}
			}
		}
	case "consistent":
		for _, t := range tables {
			firstStyle := ""
			for row := t[0]; row <= t[1]; row++ {
				line := doc.Lines[row]
				if isTableDelimiterRow(line) {
					continue
				}
				actual := tableColumnStyle(line)
				if actual == "other" {
					continue
				}
				if firstStyle == "" {
					firstStyle = actual
					continue
				}
				if actual != firstStyle {
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    row + 1,
						Column:  1,
						Message: fmt.Sprintf("Table column style [Expected: %s; Actual: %s]", firstStyle, actual),
					})
				}
			}
		}
	}
	return violations
}
