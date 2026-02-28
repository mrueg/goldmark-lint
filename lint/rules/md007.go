package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD007 checks that unordered list items are indented correctly.
type MD007 struct {
	// Indent is the number of spaces per indentation level (default 2).
	Indent int `json:"indent"`
	// StartIndented controls whether the first nesting level is indented
	// (default false: top-level items have zero leading spaces).
	StartIndented bool `json:"start_indented"`
	// StartIndent is the number of spaces used for the first nesting level
	// when StartIndented is true (default: same as Indent).
	StartIndent int `json:"start_indent"`
}

func (r MD007) ID() string          { return "MD007" }
func (r MD007) Aliases() []string   { return []string{"ul-indent"} }
func (r MD007) Description() string { return "Unordered list indentation" }

// unorderedListMarkers holds the valid unordered list marker bytes.
const unorderedListMarkers = "*-+"

func (r MD007) Check(doc *lint.Document) []lint.Violation {
	indent := r.Indent
	if indent == 0 {
		indent = 2
	}

	startIndent := 0
	if r.StartIndented {
		startIndent = r.StartIndent
		if startIndent == 0 {
			startIndent = indent
		}
	}

	var violations []lint.Violation

	for i, line := range doc.Lines {
		// Count leading spaces.
		trimmed := strings.TrimLeft(line, " ")
		spaces := len(line) - len(trimmed)

		// Must be followed by an unordered marker and a space.
		if len(trimmed) < 2 {
			continue
		}
		if strings.IndexByte(unorderedListMarkers, trimmed[0]) == -1 || trimmed[1] != ' ' {
			continue
		}

		// Determine the expected indent for this line.
		// The first level should have startIndent spaces; subsequent levels add indent.
		// Since we can't easily determine the nesting level without AST traversal,
		// we use the following heuristic: the expected indent is a multiple of indent
		// plus startIndent, and the actual spaces must equal one of those values.
		//
		// Valid indents: startIndent, startIndent+indent, startIndent+2*indent, ...
		if startIndent == 0 {
			// Default: must be multiple of indent.
			if spaces%indent != 0 {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  spaces + 1,
					Message: fmt.Sprintf("Unordered list indentation [Expected: multiple of %d; Actual: %d]", indent, spaces),
				})
			}
		} else {
			// First level: startIndent; subsequent: startIndent + n*indent.
			if spaces < startIndent || (spaces-startIndent)%indent != 0 {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  spaces + 1,
					Message: fmt.Sprintf("Unordered list indentation [Expected: %d or %d+n*%d; Actual: %d]", startIndent, startIndent, indent, spaces),
				})
			}
		}
	}

	return violations
}
