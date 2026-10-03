package rules

import (
	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/v2/ast"
)

// MD042 checks that links are not empty (no empty destination or empty text).
type MD042 struct{}

func (r MD042) ID() string          { return "MD042" }
func (r MD042) Aliases() []string   { return []string{"no-empty-links"} }
func (r MD042) Description() string { return "No empty links" }

// inlineNodeLine returns the 1-based line number of an inline node.
// It first tries to find the exact source line via a descendant Text node,
// then falls back to the first line of the nearest ancestor block.
func inlineNodeLine(n ast.Node, doc *lint.Document) int {
	// Use the first text leaf to get the actual line where the node appears,
	// rather than the block's first line.  This is important for multi-line
	// paragraphs where a link may appear on a line other than the first.
	if t := firstTextLeaf(n); t != nil {
		return doc.LineAt(textSeg(t).Start)
	}
	if pos := n.Pos(); pos >= 0 {
		return doc.LineAt(pos)
	}
	for p := n.Parent(); p != nil; p = p.Parent() {
		if !isBlockNode(p) {
			continue
		}
		if len(blockLines(p)) > 0 {
			seg := blockLines(p)[0]
			return doc.LineAt(seg.Start)
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

		dest := link.Destination.Str(doc.Source)
		// Check for empty destination
		if dest == "" || dest == "#" {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    inlineNodeLine(link, doc),
				Column:  1,
				Message: "No empty links",
			})
			return ast.WalkContinue, nil
		}

		// Check for empty link text
		hasText := false
		for c := link.FirstChild(); c != nil; c = c.NextSibling() {
			switch ct := c.(type) {
			case *ast.Text:
				seg := textSeg(ct)
				if seg.Start < seg.Stop {
					hasText = true
				}
			default:
				// Any non-text child (code span, emphasis, image, etc.) counts as text.
				hasText = true
			}
			if hasText {
				break
			}
		}
		if !hasText {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    inlineNodeLine(link, doc),
				Column:  1,
				Message: "No empty links",
			})
		}
		return ast.WalkContinue, nil
	})

	return violations
}
