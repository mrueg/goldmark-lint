package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD055 checks that table pipe style is consistent.
type MD055 struct {
	// Style is "consistent" (default), "leading_and_trailing", "leading_only",
	// "no_leading_or_trailing", or "trailing_only".
	Style string `json:"style"`
}

func (r MD055) ID() string          { return "MD055" }
func (r MD055) Aliases() []string   { return []string{"table-pipe-style"} }
func (r MD055) Description() string { return "Table pipe style" }

func rowPipeStyle(line string) string {
	trimmed := strings.TrimSpace(line)
	hasLeading := strings.HasPrefix(trimmed, "|")
	hasTrailing := strings.HasSuffix(trimmed, "|")
	switch {
	case hasLeading && hasTrailing:
		return "leading_and_trailing"
	case hasLeading:
		return "leading_only"
	case hasTrailing:
		return "trailing_only"
	default:
		return "no_leading_or_trailing"
	}
}

// Fix applies MD055 to source by normalizing table pipe style.
func (r MD055) Fix(source []byte) []byte {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	tables := findTables(lines, mask)

	if len(tables) == 0 {
		return source
	}

	firstStyle := ""
	changed := false

	for _, t := range tables {
		for row := t[0]; row <= t[1]; row++ {
			line := lines[row]
			actual := rowPipeStyle(line)
			expected := style
			if style == "consistent" {
				if firstStyle == "" {
					firstStyle = actual
				}
				expected = firstStyle
			}
			if actual == expected {
				continue
			}
			// Convert the row to the expected style.
			newLine := convertRowPipeStyle(line, expected)
			if newLine != line {
				lines[row] = newLine
				changed = true
			}
		}
	}
	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}

// convertRowPipeStyle converts a table row to the specified pipe style.
// It preserves the internal cell content (spacing and text) unchanged.
func convertRowPipeStyle(line, style string) string {
	trimmed := strings.TrimSpace(line)
	hasLeading := strings.HasPrefix(trimmed, "|")
	hasTrailing := strings.HasSuffix(trimmed, "|") && len(trimmed) > 1

	wantLeading := style == "leading_and_trailing" || style == "leading_only"
	wantTrailing := style == "leading_and_trailing" || style == "trailing_only"

	result := trimmed
	// Remove/add leading pipe.
	if hasLeading && !wantLeading {
		result = strings.TrimLeft(result[1:], " ")
	} else if !hasLeading && wantLeading {
		result = "| " + result
	}
	// Re-check trailing after potentially modifying result.
	// The actual trailing of trimmed is authoritative.
	if hasTrailing && !wantTrailing {
		result = strings.TrimRight(result[:len(result)-1], " ")
	} else if !hasTrailing && wantTrailing {
		result = result + " |"
	}
	return result
}

func (r MD055) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	mask := fencedCodeBlockMask(doc.Lines)
	tables := findTables(doc.Lines, mask)
	var violations []lint.Violation
	firstStyle := ""

	for _, t := range tables {
		for row := t[0]; row <= t[1]; row++ {
			line := doc.Lines[row]
			actual := rowPipeStyle(line)
			expected := style
			if style == "consistent" {
				if firstStyle == "" {
					firstStyle = actual
				}
				expected = firstStyle
			}
			if actual != expected {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    row + 1,
					Column:  1,
					Message: fmt.Sprintf("Table pipe style [Expected: %s; Actual: %s]", expected, actual),
				})
			}
		}
	}
	return violations
}
