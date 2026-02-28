package rules

import (
	"fmt"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD003 checks that headings use a consistent style (ATX or Setext).
type MD003 struct {
	// Style is the required heading style: "consistent" (default), "atx", "setext",
	// "atx_closed", "setext_with_atx", or "setext_with_atx_closed".
	Style string `json:"style"`
}

func (r MD003) ID() string          { return "MD003" }
func (r MD003) Aliases() []string   { return []string{"heading-style"} }
func (r MD003) Description() string { return "Heading style" }

// headingStyleOf returns "atx" or "setext" for the given heading node by
// looking back in the source to find the start of the line: if it starts
// with '#' it is ATX, otherwise it is setext.
func headingStyleOf(h *ast.Heading, source []byte) string {
	if h.Lines() == nil || h.Lines().Len() == 0 {
		return "atx"
	}
	seg := h.Lines().At(0)
	if seg.Start > len(source) {
		return "atx"
	}
	// Walk back to the start of the line to find the marker.
	pos := seg.Start
	for pos > 0 && source[pos-1] != '\n' {
		pos--
	}
	if pos < len(source) && source[pos] == '#' {
		return "atx"
	}
	return "setext"
}

func (r MD003) Check(doc *lint.Document) []lint.Violation {
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
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		line := 1
		if h.Lines() != nil && h.Lines().Len() > 0 {
			seg := h.Lines().At(0)
			line = countLine(doc.Source, seg.Start)
		}

		actual := headingStyleOf(h, doc.Source)

		expected := style
		if style == "consistent" {
			if firstStyle == "" {
				firstStyle = actual
			}
			expected = firstStyle
		}

		// setext only supports h1 and h2; for deeper levels ATX is required.
		switch expected {
		case "setext_with_atx", "setext_with_atx_closed":
			if h.Level <= 2 {
				expected = "setext"
			} else {
				expected = "atx"
			}
		case "setext":
			if h.Level > 2 {
				// setext cannot represent h3+, so ATX is acceptable.
				expected = "atx"
			}
		}

		if actual != expected {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    line,
				Column:  1,
				Message: fmt.Sprintf("Heading style [Expected: %s; Actual: %s]", expected, actual),
			})
		}
		return ast.WalkContinue, nil
	})

	return violations
}
