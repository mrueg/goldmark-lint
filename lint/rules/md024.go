package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD024 checks that no two headings have the same text content.
type MD024 struct {
	// SiblingsOnly, when true, only checks headings with the same parent node.
	SiblingsOnly bool `json:"siblings_only"`
}

func (r MD024) ID() string          { return "MD024" }
func (r MD024) Aliases() []string   { return []string{"no-duplicate-heading"} }
func (r MD024) Description() string { return "Multiple headings with the same content" }

func (r MD024) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	if r.SiblingsOnly {
		return r.checkSiblings(doc)
	}

	// Global check: no two headings in the whole document may share the same text.
	seen := make(map[string]bool)

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		text := strings.ToLower(strings.TrimSpace(headingText(h, doc.Source)))
		if seen[text] {
			line := 1
			if h.Lines() != nil && h.Lines().Len() > 0 {
				seg := h.Lines().At(0)
				line = countLine(doc.Source, seg.Start)
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    line,
				Column:  1,
				Message: fmt.Sprintf("Multiple headings with the same content [Context: \"%s\"]", headingText(h, doc.Source)),
			})
		}
		seen[text] = true
		return ast.WalkContinue, nil
	})

	return violations
}

func (r MD024) checkSiblings(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		// Walk container nodes, checking their heading children among siblings.
		seen := make(map[string]bool)
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			h, ok := child.(*ast.Heading)
			if !ok {
				continue
			}
			text := strings.ToLower(strings.TrimSpace(headingText(h, doc.Source)))
			if seen[text] {
				line := 1
				if h.Lines() != nil && h.Lines().Len() > 0 {
					seg := h.Lines().At(0)
					line = countLine(doc.Source, seg.Start)
				}
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    line,
					Column:  1,
					Message: fmt.Sprintf("Multiple headings with the same content [Context: \"%s\"]", headingText(h, doc.Source)),
				})
			}
			seen[text] = true
		}
		return ast.WalkContinue, nil
	})

	return violations
}
