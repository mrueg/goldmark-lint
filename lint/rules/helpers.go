package rules

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
)

// countLine counts the 1-based line number of byte offset pos in source.
func countLine(source []byte, pos int) int {
	line := 1
	for i := 0; i < pos && i < len(source); i++ {
		if source[i] == '\n' {
			line++
		}
	}
	return line
}

// headingText returns the text content of a heading node.
func headingText(n ast.Node, source []byte) string {
	var text []byte
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			seg := t.Segment
			text = append(text, source[seg.Start:seg.Stop]...)
		}
	}
	return string(text)
}

// fencedCodeBlockMask returns a bool slice with true for each line that is
// inside (not on the fence delimiters of) a fenced code block.
func fencedCodeBlockMask(lines []string) []bool {
	mask := make([]bool, len(lines))
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if !inFence {
			if len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
				fc := trimmed[0]
				j := 0
				for j < len(trimmed) && trimmed[j] == fc {
					j++
				}
				if j >= 3 {
					inFence = true
					fenceChar = fc
					fenceLen = j
				}
			}
			mask[i] = false
		} else {
			if len(trimmed) >= fenceLen && trimmed[0] == fenceChar {
				j := 0
				for j < len(trimmed) && trimmed[j] == fenceChar {
					j++
				}
				if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" {
					inFence = false
					mask[i] = false
					continue
				}
			}
			mask[i] = true
		}
	}
	return mask
}

// tableDelimiterCellRE matches a GFM table delimiter cell.
var tableDelimiterCellRE = regexp.MustCompile(`^\s*:?-+:?\s*$`)

// isTableDelimiterRow returns true if line is a GFM table delimiter row.
func isTableDelimiterRow(line string) bool {
if !strings.Contains(line, "|") {
return false
}
trimmed := strings.TrimSpace(line)
if strings.HasPrefix(trimmed, "|") {
trimmed = trimmed[1:]
}
if strings.HasSuffix(trimmed, "|") {
trimmed = trimmed[:len(trimmed)-1]
}
cells := strings.Split(trimmed, "|")
if len(cells) == 0 {
return false
}
for _, cell := range cells {
if !tableDelimiterCellRE.MatchString(cell) {
return false
}
}
return true
}

// isTableRow returns true if line looks like a table row (contains |).
func isTableRow(line string) bool {
return strings.Contains(line, "|")
}

// headingAnchor converts heading text to a GitHub-style anchor.
func headingAnchor(text string) string {
result := strings.ToLower(text)
var b strings.Builder
for _, r := range result {
if r == ' ' || r == '-' {
b.WriteRune('-')
} else if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
b.WriteRune(r)
}
}
return b.String()
}

// countTableCells counts the cells in a table row.
func countTableCells(line string) int {
trimmed := strings.TrimSpace(line)
if strings.HasPrefix(trimmed, "|") {
trimmed = trimmed[1:]
}
if strings.HasSuffix(trimmed, "|") {
trimmed = trimmed[:len(trimmed)-1]
}
return len(strings.Split(trimmed, "|"))
}

// tableHasLeadingPipe returns true if the (trimmed) row starts with |.
func tableHasLeadingPipe(line string) bool {
return strings.HasPrefix(strings.TrimSpace(line), "|")
}

// tableHasTrailingPipe returns true if the (trimmed) row ends with |.
func tableHasTrailingPipe(line string) bool {
return strings.HasSuffix(strings.TrimSpace(line), "|")
}

// findTables returns slices of [start, end] line indices (0-based, inclusive) for each GFM table.
func findTables(lines []string, mask []bool) [][2]int {
var tables [][2]int
n := len(lines)
i := 0
for i < n {
if mask[i] || !isTableRow(lines[i]) {
i++
continue
}
// Check that next line is a delimiter row
if i+1 >= n || mask[i+1] || !isTableDelimiterRow(lines[i+1]) {
i++
continue
}
// Table starts at i
start := i
end := i + 1
for end+1 < n && !mask[end+1] && isTableRow(lines[end+1]) && !isTableDelimiterRow(lines[end+1]) {
end++
}
tables = append(tables, [2]int{start, end})
i = end + 1
}
return tables
}
