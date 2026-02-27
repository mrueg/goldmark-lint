package rules

import (
	"fmt"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD025 checks for multiple top-level headings in a document.
type MD025 struct {
	// Level is the top-level heading level (default 1).
	Level int `json:"level"`
}

func (r MD025) ID() string          { return "MD025" }
func (r MD025) Description() string { return "Multiple top-level headings in the same document" }

func (r MD025) Check(doc *lint.Document) []lint.Violation {
	level := r.Level
	if level == 0 {
		level = 1
	}
	var violations []lint.Violation
	count := 0

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		if h.Level == level {
			count++
			if count > 1 {
				line := 1
				if h.Lines() != nil && h.Lines().Len() > 0 {
					seg := h.Lines().At(0)
					line = countLine(doc.Source, seg.Start)
				}
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    line,
					Column:  1,
					Message: fmt.Sprintf("Multiple top-level headings in the same document [Context: \"%s\"]", headingText(n, doc.Source)),
				})
			}
		}
		return ast.WalkContinue, nil
	})
	return violations
}
