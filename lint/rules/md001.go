package rules

import (
	"fmt"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD001 checks that heading levels only increment by one level at a time.
type MD001 struct{}

func (r MD001) ID() string { return "MD001" }
func (r MD001) Description() string {
	return "Heading levels should only increment by one level at a time"
}

func (r MD001) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	prevLevel := 0

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		level := h.Level
		if prevLevel > 0 && level > prevLevel+1 {
			line := 1
			if h.Lines() != nil && h.Lines().Len() > 0 {
				seg := h.Lines().At(0)
				line = countLine(doc.Source, seg.Start)
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    line,
				Column:  1,
				Message: fmt.Sprintf("Heading levels should only increment by one level at a time [Expected: h%d; Actual: h%d]", prevLevel+1, level),
			})
		}
		prevLevel = level
		return ast.WalkContinue, nil
	})

	return violations
}
