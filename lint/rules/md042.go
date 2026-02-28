package rules

import (
	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD042 checks that links are not empty (no empty destination or empty text).
type MD042 struct{}

func (r MD042) ID() string          { return "MD042" }
func (r MD042) Aliases() []string   { return []string{"no-empty-links"} }
func (r MD042) Description() string { return "No empty links" }

// inlineNodeLine returns the 1-based line number of an inline node by walking
// up to the nearest ancestor block node that has source line information.
func inlineNodeLine(n ast.Node, source []byte) int {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Lines() != nil && p.Lines().Len() > 0 {
			seg := p.Lines().At(0)
			return countLine(source, seg.Start)
		}
	}
	// Fallback: check text children
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			return countLine(source, t.Segment.Start)
		}
	}
	return 1
}

func (r MD042) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}

		dest := string(link.Destination)
		// Check for empty destination
		if dest == "" || dest == "#" {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    inlineNodeLine(link, doc.Source),
				Column:  1,
				Message: "No empty links",
			})
			return ast.WalkContinue, nil
		}

		// Check for empty link text
		hasText := false
		for c := link.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				seg := t.Segment
				if seg.Start < seg.Stop {
					hasText = true
					break
				}
			}
		}
		if !hasText {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    inlineNodeLine(link, doc.Source),
				Column:  1,
				Message: "No empty links",
			})
		}
		return ast.WalkContinue, nil
	})

	return violations
}
