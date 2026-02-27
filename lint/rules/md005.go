package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MD005 checks that list items at the same nesting level use consistent indentation.
type MD005 struct{}

func (r MD005) ID() string { return "MD005" }
func (r MD005) Description() string {
	return "Inconsistent indentation for list items at the same level"
}

func (r MD005) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	expectedIndent := make(map[int]int)
	depth := 0

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.List:
			if entering {
				depth++
			} else {
				depth--
			}
		case *ast.ListItem:
			if !entering {
				return ast.WalkContinue, nil
			}
			// ListItem.Lines() is empty in goldmark; content is in the first TextBlock child.
			seg, ok := listItemFirstSegment(n)
			if !ok {
				return ast.WalkContinue, nil
			}
			lineIdx := countLine(doc.Source, seg.Start) - 1
			if lineIdx < 0 || lineIdx >= len(doc.Lines) {
				return ast.WalkContinue, nil
			}
			line := doc.Lines[lineIdx]
			spaces := len(line) - len(strings.TrimLeft(line, " "))

			if exp, ok := expectedIndent[depth]; ok {
				if spaces != exp {
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    lineIdx + 1,
						Column:  spaces + 1,
						Message: fmt.Sprintf("Inconsistent indentation for list items at the same level [Expected: %d; Actual: %d]", exp, spaces),
					})
				}
			} else {
				expectedIndent[depth] = spaces
			}
		}
		return ast.WalkContinue, nil
	})

	return violations
}

// listItemFirstSegment returns the first text segment of a list item by walking
// its first TextBlock child, since ListItem.Lines() is empty in goldmark.
func listItemFirstSegment(item *ast.ListItem) (text.Segment, bool) {
	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		if tb, ok := c.(*ast.TextBlock); ok {
			if tb.Lines() != nil && tb.Lines().Len() > 0 {
				return tb.Lines().At(0), true
			}
		}
	}
	return text.Segment{}, false
}
