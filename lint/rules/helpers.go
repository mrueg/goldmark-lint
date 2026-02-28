package rules

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// IntOrArray is a JSON-compatible type that can be either a single integer or
// an array of integers. When used for per-heading-level config (e.g. lines_above
// in MD022), index 0 = h1, index 1 = h2, etc.
type IntOrArray []int

// UnmarshalJSON implements json.Unmarshaler.
func (x *IntOrArray) UnmarshalJSON(data []byte) error {
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*x = IntOrArray{n}
		return nil
	}
	var arr []int
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*x = IntOrArray(arr)
	return nil
}

// Get returns the configured value for the given 1-based heading level.
// If the array is empty, 0 is returned (callers should use their default).
// If the level exceeds the array length, the last element is returned.
func (x IntOrArray) Get(level int) int {
	if len(x) == 0 {
		return 0
	}
	idx := level - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(x) {
		return x[len(x)-1]
	}
	return x[idx]
}

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
	trimmed := strings.TrimPrefix(strings.TrimSpace(line), "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
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
	trimmed := strings.TrimPrefix(strings.TrimSpace(line), "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	return len(strings.Split(trimmed, "|"))
}

// frontMatterHasTitle reports whether the document's front matter contains a
// title field. pattern is a field name or regex; if empty, the feature is
// disabled and false is always returned.
func frontMatterHasTitle(doc *lint.Document, pattern string) bool {
	if pattern == "" {
		return false
	}
	if len(doc.FrontMatterFields) == 0 {
		return false
	}
	// First try exact key match (case-insensitive).
	for k, v := range doc.FrontMatterFields {
		if strings.EqualFold(k, pattern) && v != "" {
			return true
		}
	}
	// Try pattern as a regex matched against "key: value" lines.
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return false
	}
	for k, v := range doc.FrontMatterFields {
		if re.MatchString(k + ": " + v) {
			return true
		}
	}
	return false
}

// fencedCodeBlockLanguages returns a map from line index (0-based) to the
// language of the fenced code block that line belongs to (empty string if
// the block has no language specified). Lines outside code blocks are absent.
func fencedCodeBlockLanguages(lines []string) map[int]string {
	result := make(map[int]string)
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	currentLang := ""
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
					info := strings.TrimSpace(trimmed[j:])
					if fields := strings.Fields(info); len(fields) > 0 {
						currentLang = fields[0]
					} else {
						currentLang = ""
					}
				}
			}
		} else {
			if len(trimmed) >= fenceLen && trimmed[0] == fenceChar {
				j := 0
				for j < len(trimmed) && trimmed[j] == fenceChar {
					j++
				}
				if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" {
					inFence = false
					currentLang = ""
					continue
				}
			}
			result[i] = currentLang
		}
	}
	return result
}

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
