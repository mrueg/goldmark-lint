package rules

import (
	"fmt"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD046 checks code block style consistency.
type MD046 struct {
	// Style is "consistent" (default), "fenced", or "indented".
	Style string `json:"style"`
}

func (r MD046) ID() string          { return "MD046" }
func (r MD046) Aliases() []string   { return []string{"code-block-style"} }
func (r MD046) Description() string { return "Code block style" }

func (r MD046) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	var violations []lint.Violation
	firstStyle := ""

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		var blockStyle string
		var lineNum int

		switch node := n.(type) {
		case *ast.FencedCodeBlock:
			blockStyle = "fenced"
			lineNum = fencedCodeBlockLine(node, doc.Source)
		case *ast.CodeBlock:
			blockStyle = "indented"
			if node.Lines() != nil && node.Lines().Len() > 0 {
				lineNum = countLine(doc.Source, node.Lines().At(0).Start)
			}
		default:
			return ast.WalkContinue, nil
		}

		if lineNum == 0 {
			return ast.WalkContinue, nil
		}

		expected := style
		if style == "consistent" {
			if firstStyle == "" {
				firstStyle = blockStyle
			}
			expected = firstStyle
		}

		if blockStyle != expected {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    lineNum,
				Column:  1,
				Message: fmt.Sprintf("Code block style [Expected: %s; Actual: %s]", expected, blockStyle),
			})
		}

		return ast.WalkContinue, nil
	})

	return violations
}
