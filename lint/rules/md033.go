package rules

import (
	"fmt"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD033 checks for inline HTML in Markdown documents.
type MD033 struct {
	// AllowedElements is a list of HTML element names that are permitted.
	AllowedElements []string
}

func (r MD033) ID() string          { return "MD033" }
func (r MD033) Description() string { return "Inline HTML" }

func (r MD033) isAllowed(tag string) bool {
	for _, allowed := range r.AllowedElements {
		if allowed == tag {
			return true
		}
	}
	return false
}

func (r MD033) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.HTMLBlock:
			line := 1
			if node.Lines() != nil && node.Lines().Len() > 0 {
				seg := node.Lines().At(0)
				line = countLine(doc.Source, seg.Start)
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    line,
				Column:  1,
				Message: fmt.Sprintf("Inline HTML [Element: block HTML]"),
			})

		case *ast.RawHTML:
			line := 1
			if node.Segments != nil && node.Segments.Len() > 0 {
				seg := node.Segments.At(0)
				line = countLine(doc.Source, seg.Start)
			}
			// Extract the tag name for allowed-element checking.
			tag := rawHTMLTagName(node, doc.Source)
			if r.isAllowed(tag) {
				return ast.WalkContinue, nil
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    line,
				Column:  1,
				Message: fmt.Sprintf("Inline HTML [Element: %s]", tag),
			})
		}

		return ast.WalkContinue, nil
	})

	return violations
}

// rawHTMLTagName extracts the tag name from a RawHTML node (e.g. "br" from "<br/>").
func rawHTMLTagName(n *ast.RawHTML, source []byte) string {
	if n.Segments == nil || n.Segments.Len() == 0 {
		return "unknown"
	}
	seg := n.Segments.At(0)
	raw := string(seg.Value(source))
	// Strip leading '<' and optional '/'.
	i := 0
	if i < len(raw) && raw[i] == '<' {
		i++
	}
	if i < len(raw) && raw[i] == '/' {
		i++
	}
	// Read the tag name (alphanumeric + hyphen).
	start := i
	for i < len(raw) && (raw[i] == '-' || (raw[i] >= 'a' && raw[i] <= 'z') || (raw[i] >= 'A' && raw[i] <= 'Z') || (raw[i] >= '0' && raw[i] <= '9')) {
		i++
	}
	if i == start {
		return "unknown"
	}
	return raw[start:i]
}
