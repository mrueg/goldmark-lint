package rules

import (
	"fmt"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD004 checks that unordered list markers are consistent.
type MD004 struct {
	// Style is the required marker style: "consistent" (default), "asterisk", "plus", or "dash".
	Style string
}

func (r MD004) ID() string          { return "MD004" }
func (r MD004) Description() string { return "Unordered list style" }

func (r MD004) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	var violations []lint.Violation
	firstMarker := byte(0)

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		list, ok := n.(*ast.List)
		if !ok || list.IsOrdered() {
			return ast.WalkContinue, nil
		}

		marker := list.Marker

		expected := byte(0)
		switch style {
		case "asterisk":
			expected = '*'
		case "plus":
			expected = '+'
		case "dash":
			expected = '-'
		case "consistent":
			if firstMarker == 0 {
				firstMarker = marker
			}
			expected = firstMarker
		}

		if expected != 0 && marker != expected {
			// Find the line of the first list item.
			line := 1
			if item := list.FirstChild(); item != nil {
				if li, ok2 := item.(*ast.ListItem); ok2 {
					if li.Lines() != nil && li.Lines().Len() > 0 {
						seg := li.Lines().At(0)
						line = countLine(doc.Source, seg.Start)
					}
				}
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    line,
				Column:  1,
				Message: fmt.Sprintf("Unordered list style [Expected: %c; Actual: %c]", expected, marker),
			})
		}
		return ast.WalkContinue, nil
	})

	return violations
}
