package rules

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
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
	// Split on unescaped pipes only.
	var cells []string
	start := 0
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == '|' && (i == 0 || trimmed[i-1] != '\\') {
			cells = append(cells, trimmed[start:i])
			start = i + 1
		}
	}
	cells = append(cells, trimmed[start:])

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

// md60PipePositions returns the 0-based display-width positions of all '|'
// characters in line that are not escaped (i.e., not preceded by '\'). Display
// width is used (CJK/fullwidth chars count as 2) to match markdownlint's
// "aligned" style comparison behaviour.
func md60PipePositions(line string) []int {
	var positions []int
	dispWidth := 0
	runes := []rune(line)
	for i, r := range runes {
		if r == '|' && (i == 0 || runes[i-1] != '\\') {
			positions = append(positions, dispWidth)
		}
		dispWidth += runeDisplayWidth(r)
	}
	return positions
}

// runeDisplayWidth returns the display width of a rune: 2 for fullwidth/wide
// Unicode characters (e.g. CJK ideographs, emoji), 0 for nonspacing and
// spacing-combining marks (Indic vowel signs, Thai diacritics, etc.), and 1
// for everything else. This matches the behaviour of markdownlint's
// string-width npm package.
func runeDisplayWidth(r rune) int {
	// Combining marks (Unicode categories Mn and Mc) are zero-width: they
	// render onto the preceding base character without advancing the cursor.
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) {
		return 0
	}
	return runewidth.RuneWidth(r)
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
		if line[i] == '|' && (i == 0 || line[i-1] != '\\') {
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
		style = "any"
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

// md060CellsOf splits a GFM table row into its raw cell strings (the content
// between '|' separators, including surrounding whitespace).  Leading and
// trailing edge pipes are consumed but the content between them is returned
// verbatim.  For example "|a|b|" yields ["a", "b"].
func md060CellsOf(line string) (cells []string, hasLeading, hasTrailing bool) {
	trimmed := strings.TrimSpace(line)
	hasLeading = strings.HasPrefix(trimmed, "|")
	hasTrailing = len(trimmed) > 1 && strings.HasSuffix(trimmed, "|")

	work := trimmed
	if hasLeading {
		work = work[1:]
	}
	if hasTrailing {
		work = work[:len(work)-1]
	}
	cells = strings.Split(work, "|")
	return
}

// md060RebuildRow reassembles cells into a table row using the given padding
// function and preserving whether the original row had leading/trailing pipes.
func md060RebuildRow(cells []string, hasLeading, hasTrailing bool, padFn func(string) string) string {
	var b strings.Builder
	if hasLeading {
		b.WriteByte('|')
	}
	for ci, cell := range cells {
		b.WriteString(padFn(cell))
		if ci < len(cells)-1 || hasTrailing {
			b.WriteByte('|')
		}
	}
	return b.String()
}

// md060ConvertRowToCompact converts a table row to compact style (one space on
// each side of cell content).  The delimiter row is left unchanged.
func md060ConvertRowToCompact(line string) string {
	if isTableDelimiterRow(line) {
		return line
	}
	cells, hasLeading, hasTrailing := md060CellsOf(line)
	pad := func(c string) string { return " " + strings.TrimSpace(c) + " " }
	return md060RebuildRow(cells, hasLeading, hasTrailing, pad)
}

// md060ConvertRowToTight converts a table row to tight style (no spaces around
// cell content).  The delimiter row is left unchanged.
func md060ConvertRowToTight(line string) string {
	if isTableDelimiterRow(line) {
		return line
	}
	cells, hasLeading, hasTrailing := md060CellsOf(line)
	pad := func(c string) string { return strings.TrimSpace(c) }
	return md060RebuildRow(cells, hasLeading, hasTrailing, pad)
}

// md060ConvertTableToAligned reformats all rows in the table slice so that
// pipe characters align at the same column positions across every row.
func md060ConvertTableToAligned(tableLines []string) []string {
	if len(tableLines) == 0 {
		return tableLines
	}

	type rowData struct {
		cells       []string
		hasLeading  bool
		hasTrailing bool
		isDelim     bool
	}
	rows := make([]rowData, len(tableLines))
	maxWidths := []int{}

	for ri, line := range tableLines {
		cells, hasLeading, hasTrailing := md060CellsOf(line)
		trimmed := make([]string, len(cells))
		for ci, c := range cells {
			trimmed[ci] = strings.TrimSpace(c)
		}
		rows[ri] = rowData{
			cells:       trimmed,
			hasLeading:  hasLeading,
			hasTrailing: hasTrailing,
			isDelim:     isTableDelimiterRow(line),
		}
		for ci, c := range trimmed {
			if ci >= len(maxWidths) {
				maxWidths = append(maxWidths, 0)
			}
			if len(c) > maxWidths[ci] {
				maxWidths[ci] = len(c)
			}
		}
	}

	result := make([]string, len(tableLines))
	for ri, rd := range rows {
		var b strings.Builder
		if rd.hasLeading {
			b.WriteByte('|')
		}
		for ci, cell := range rd.cells {
			colWidth := 0
			if ci < len(maxWidths) {
				colWidth = maxWidths[ci]
			}
			if colWidth < len(cell) {
				colWidth = len(cell)
			}
			if rd.isDelim {
				// Pad delimiter dashes to align.
				dashes := cell
				if len(dashes) == 0 {
					dashes = "-"
				}
				padded := dashes + strings.Repeat("-", colWidth-len(dashes))
				b.WriteByte(' ')
				b.WriteString(padded)
				b.WriteByte(' ')
			} else {
				b.WriteByte(' ')
				b.WriteString(cell)
				b.WriteString(strings.Repeat(" ", colWidth-len(cell)))
				b.WriteByte(' ')
			}
			if ci < len(rd.cells)-1 || rd.hasTrailing {
				b.WriteByte('|')
			}
		}
		result[ri] = b.String()
	}
	return result
}

// Fix rewrites table rows in source to match the configured column style.
// "compact" and "tight" are fully supported; "aligned" pads columns so that
// pipes align across all rows; "consistent" applies the first row's detected
// style to every row; "any" performs no changes.
func (r MD060) Fix(source []byte) []byte {
	style := r.Style
	if style == "" {
		style = "any"
	}
	if style == "any" {
		return source
	}

	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	tables := findTables(lines, mask)
	if len(tables) == 0 {
		return source
	}

	changed := false

	for _, t := range tables {
		targetStyle := style

		if style == "consistent" {
			firstStyle := ""
			for row := t[0]; row <= t[1]; row++ {
				if isTableDelimiterRow(lines[row]) {
					continue
				}
				s := tableColumnStyle(lines[row])
				if s != "other" {
					firstStyle = s
					break
				}
			}
			if firstStyle == "" {
				continue
			}
			targetStyle = firstStyle
		}

		switch targetStyle {
		case "compact":
			for row := t[0]; row <= t[1]; row++ {
				newLine := md060ConvertRowToCompact(lines[row])
				if newLine != lines[row] {
					lines[row] = newLine
					changed = true
				}
			}
		case "tight":
			for row := t[0]; row <= t[1]; row++ {
				newLine := md060ConvertRowToTight(lines[row])
				if newLine != lines[row] {
					lines[row] = newLine
					changed = true
				}
			}
		case "aligned":
			newRows := md060ConvertTableToAligned(lines[t[0] : t[1]+1])
			for j, newLine := range newRows {
				if newLine != lines[t[0]+j] {
					lines[t[0]+j] = newLine
					changed = true
				}
			}
		}
	}

	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}
