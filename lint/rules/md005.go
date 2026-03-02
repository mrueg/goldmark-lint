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
func (r MD005) Aliases() []string {
	return []string{"list-indent"}
}
func (r MD005) Description() string {
	return "Inconsistent indentation for list items at the same level"
}

func (r MD005) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		list, ok := n.(*ast.List)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Determine expected indent from first list item.
		expectedIndent := -1
		for child := list.FirstChild(); child != nil; child = child.NextSibling() {
			li, ok2 := child.(*ast.ListItem)
			if !ok2 {
				continue
			}
			seg, ok3 := listItemFirstSegment(li)
			if !ok3 {
				continue
			}
			lineIdx := countLine(doc.Source, seg.Start) - 1
			if lineIdx < 0 || lineIdx >= len(doc.Lines) {
				continue
			}
			line := doc.Lines[lineIdx]
			spaces := len(line) - len(strings.TrimLeft(line, " "))
			if expectedIndent < 0 {
				expectedIndent = spaces
			} else if spaces != expectedIndent {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    lineIdx + 1,
					Column:  spaces + 1,
					Message: fmt.Sprintf("Inconsistent indentation for list items at the same level [Expected: %d; Actual: %d]", expectedIndent, spaces),
				})
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
