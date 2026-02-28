package rules

import (
	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD045 checks that images have alternate text (alt text).
type MD045 struct{}

func (r MD045) ID() string          { return "MD045" }
func (r MD045) Aliases() []string   { return []string{"no-alt-text"} }
func (r MD045) Description() string { return "Images should have alternate text (alt text)" }

func (r MD045) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		img, ok := n.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Check if the image has non-empty alt text
		hasAltText := false
		for c := img.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				seg := t.Segment
				if seg.Start < seg.Stop {
					hasAltText = true
					break
				}
			}
		}

		if !hasAltText {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    inlineNodeLine(img, doc.Source),
				Column:  1,
				Message: "Images should have alternate text (alt text)",
			})
		}
		return ast.WalkContinue, nil
	})

	return violations
}
