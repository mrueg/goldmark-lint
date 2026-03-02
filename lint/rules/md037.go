package rules

import (
	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD037 checks for spaces inside emphasis markers.
type MD037 struct{}

func (r MD037) ID() string          { return "MD037" }
func (r MD037) Aliases() []string   { return []string{"no-space-in-emphasis"} }
func (r MD037) Description() string { return "Spaces inside emphasis markers" }

func (r MD037) Fix(source []byte) []byte {
	return source
}

func (r MD037) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		emph, ok := n.(*ast.Emphasis)
		if !ok {
			return ast.WalkContinue, nil
		}
		pos := emphasisStartPos(emph)
		if pos < 0 || pos+emph.Level >= len(doc.Source) {
			return ast.WalkContinue, nil
		}
		// Check for space immediately after opening marker.
		if doc.Source[pos+emph.Level] == ' ' {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    countLine(doc.Source, pos),
				Column:  1,
				Message: "Spaces inside emphasis markers",
			})
			return ast.WalkContinue, nil
		}
		// Check for space immediately before closing marker by examining
		// the last text segment of the emphasis content.
		// Only perform this check if the last direct child is a Text node;
		// if the emphasis ends with a code span, link, or other non-text node,
		// the space between that node and the closing marker is not flagged.
		var lastStop int
		var lastChild ast.Node
		for c := emph.FirstChild(); c != nil; c = c.NextSibling() {
			lastChild = c
			if t, ok2 := c.(*ast.Text); ok2 && t.Segment.Stop > lastStop {
				lastStop = t.Segment.Stop
			}
		}
		if _, ok := lastChild.(*ast.Text); ok {
			if lastStop > 0 && lastStop <= len(doc.Source) && doc.Source[lastStop-1] == ' ' {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    countLine(doc.Source, pos),
					Column:  1,
					Message: "Spaces inside emphasis markers",
				})
			}
		}
		return ast.WalkContinue, nil
	})
	return violations
}
