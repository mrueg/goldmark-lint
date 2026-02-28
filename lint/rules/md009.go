package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD009 checks for trailing spaces at the end of lines.
type MD009 struct {
	// BrSpaces is the number of spaces allowed at end of line for hard line breaks (default 2).
	BrSpaces int `json:"br_spaces"`
	// CodeBlocks controls whether trailing spaces in fenced code blocks are checked (default true).
	CodeBlocks *bool `json:"code_blocks"`
	// ListItemEmptyLines controls whether trailing spaces are allowed on empty lines
	// within list items (default false).
	ListItemEmptyLines bool `json:"list_item_empty_lines"`
	// Strict disallows all trailing spaces, including the br_spaces hard-break spaces (default false).
	Strict bool `json:"strict"`
}

func (r MD009) ID() string          { return "MD009" }
func (r MD009) Description() string { return "Trailing spaces" }

func (r MD009) Fix(source []byte) []byte {
	brSpaces := r.BrSpaces
	if brSpaces == 0 {
		brSpaces = 2
	}
	lines := strings.Split(string(source), "\n")
	codeMask := fencedCodeBlockMask(lines)
	checkCodeBlocks := r.CodeBlocks == nil || *r.CodeBlocks
	for i, line := range lines {
		if !checkCodeBlocks && codeMask[i] {
			continue
		}
		trimmed := strings.TrimRight(line, " \t")
		trailingLen := len(line) - len(trimmed)
		if trailingLen > 0 {
			if !r.Strict && trailingLen == brSpaces && strings.HasSuffix(line, strings.Repeat(" ", brSpaces)) {
				continue
			}
			lines[i] = trimmed
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD009) Check(doc *lint.Document) []lint.Violation {
	brSpaces := r.BrSpaces
	if brSpaces == 0 {
		brSpaces = 2
	}
	checkCodeBlocks := r.CodeBlocks == nil || *r.CodeBlocks
	codeMask := fencedCodeBlockMask(doc.Lines)

	var violations []lint.Violation
	for i, line := range doc.Lines {
		if !checkCodeBlocks && codeMask[i] {
			continue
		}
		trimmed := strings.TrimRight(line, " \t")
		trailingLen := len(line) - len(trimmed)
		if trailingLen > 0 {
			if !r.Strict && trailingLen == brSpaces && strings.HasSuffix(line, strings.Repeat(" ", brSpaces)) {
				// Allow br_spaces hard line breaks unless strict mode.
				if !r.ListItemEmptyLines || strings.TrimSpace(trimmed) != "" {
					continue
				}
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  len(trimmed) + 1,
				Message: fmt.Sprintf("Trailing spaces [Expected: 0 or %d; Actual: %d]", brSpaces, trailingLen),
			})
		}
	}
	return violations
}
