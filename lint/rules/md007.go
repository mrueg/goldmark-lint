package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
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

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		li, ok := n.(*ast.ListItem)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Only check items in unordered lists.
		parentList, ok2 := li.Parent().(*ast.List)
		if !ok2 || parentList.IsOrdered() {
			return ast.WalkContinue, nil
		}

		// Calculate nesting level: count unordered-list ancestors above parentList.
		// If any ordered list is encountered, skip this item (not checked by markdownlint).
		nesting := 0
		current := ast.Node(parentList)
		skip := false
		for {
			p := current.Parent()
			if p == nil {
				break
			}
			if list, ok3 := p.(*ast.List); ok3 {
				if list.IsOrdered() {
					skip = true
					break
				}
				nesting++
			}
			current = p
		}
		if skip {
			return ast.WalkContinue, nil
		}

		// Get the source line for this list item.
		lineNum := 0
		if fc := li.FirstChild(); fc != nil {
			if fc.Lines() != nil && fc.Lines().Len() > 0 {
				seg := fc.Lines().At(0)
				lineNum = countLine(doc.Source, seg.Start)
			}
		}
		if lineNum < 1 || lineNum > len(doc.Lines) {
			return ast.WalkContinue, nil
		}
		rawLine := doc.Lines[lineNum-1]

		// Strip blockquote prefix(es) to get the indentation within the blockquote.
		line := rawLine
		for {
			stripped := strings.TrimLeft(line, " ")
			if len(stripped) == 0 || stripped[0] != '>' {
				break
			}
			line = stripped[1:]
			if len(line) > 0 && line[0] == ' ' {
				line = line[1:]
			}
		}

		// Count leading spaces.
		trimmed := strings.TrimLeft(line, " ")
		spaces := len(line) - len(trimmed)

		// Must be a list item line.
		if len(trimmed) < 2 || strings.IndexByte(unorderedListMarkers, trimmed[0]) == -1 || trimmed[1] != ' ' {
			return ast.WalkContinue, nil
		}

		// Calculate expected indent.
		expectedIndent := nesting * indent
		if r.StartIndented {
			expectedIndent = startIndent + nesting*indent
		}

		if spaces != expectedIndent {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    lineNum,
				Column:  spaces + 1,
				Message: fmt.Sprintf("Unordered list indentation [Expected: %d; Actual: %d]", expectedIndent, spaces),
			})
		}
		return ast.WalkContinue, nil
	})

	return violations
}
