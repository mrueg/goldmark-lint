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

// headingRawContent returns the raw source content of a heading (after stripping
// ATX markers like ##, or the raw first line for setext headings).
// This preserves inline formatting characters and matches markdownlint's behavior.
func headingRawContent(h *ast.Heading, source []byte, lines []string) string {
	if h.Lines() == nil || h.Lines().Len() == 0 {
		return headingText(h, source)
	}
	seg := h.Lines().At(0)
	lineIdx := countLine(source, seg.Start) - 1
	if lineIdx < 0 || lineIdx >= len(lines) {
		return headingText(h, source)
	}
	line := lines[lineIdx]

	// Strip blockquote prefix(es).
	for {
		stripped := strings.TrimLeft(line, " ")
		if len(stripped) == 0 || stripped[0] != '>' {
			break
		}
		line = stripped[1:]
		if len(line) > 0 && line[0] == ' ' {
			line = line[1:]
		}
	}

	// Strip leading ATX markers (up to 3 spaces + # characters + one space).
	j := 0
	for j < len(line) && line[j] == ' ' && j < 3 {
		j++
	}
	for j < len(line) && line[j] == '#' {
		j++
	}
	if j < len(line) && line[j] == ' ' {
		j++
	}
	content := line[j:]
	// Strip optional closing ATX markers.
	content = strings.TrimRight(content, " #")
	return strings.TrimSpace(content)
}

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

		text := headingRawContent(h, doc.Source, doc.Lines)
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
				Message: fmt.Sprintf("Multiple headings with the same content [Context: \"%s\"]", headingRawContent(h, doc.Source, doc.Lines)),
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
			text := headingRawContent(h, doc.Source, doc.Lines)
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
					Message: fmt.Sprintf("Multiple headings with the same content [Context: \"%s\"]", headingRawContent(h, doc.Source, doc.Lines)),
				})
			}
			seen[text] = true
		}
		return ast.WalkContinue, nil
	})

	return violations
}
