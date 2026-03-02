package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD036 checks that emphasis is not used instead of a heading.
type MD036 struct {
	// Punctuation is the set of characters that, if a paragraph ends with one,
	// the emphasis check is skipped. Default: ".,;:!?。，；：！？"
	Punctuation string `json:"punctuation"`
}

func (r MD036) ID() string          { return "MD036" }
func (r MD036) Aliases() []string   { return []string{"no-emphasis-as-heading"} }
func (r MD036) Description() string { return "Emphasis used instead of a heading" }

const defaultMD036Punctuation = ".,;:!?。，；：！？"

func (r MD036) punct() string {
	if r.Punctuation == "" {
		return defaultMD036Punctuation
	}
	return r.Punctuation
}

func (r MD036) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	punct := r.punct()

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		para, ok := n.(*ast.Paragraph)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Only check top-level paragraphs (not inside lists, list items, etc.)
		// to match markdownlint behavior.
		parent := para.Parent()
		for parent != nil {
			switch parent.(type) {
			case *ast.ListItem, *ast.List, *ast.Blockquote:
				return ast.WalkContinue, nil
			}
			parent = parent.Parent()
		}

		// The paragraph must consist of a single emphasis node (no other children).
		first := para.FirstChild()
		if first == nil || first.NextSibling() != nil {
			return ast.WalkContinue, nil
		}
		emph, ok := first.(*ast.Emphasis)
		if !ok {
			return ast.WalkContinue, nil
		}

		// The emphasis must consist of exactly one plain text child (no code spans,
		// nested emphasis, line breaks, etc.). This matches markdownlint's requirement
		// that the emphasis text token has exactly one "data" child.
		emphChild := emph.FirstChild()
		if emphChild == nil || emphChild.NextSibling() != nil {
			return ast.WalkContinue, nil
		}
		if _, ok := emphChild.(*ast.Text); !ok {
			return ast.WalkContinue, nil
		}

		// Get the text content of the emphasis node.
		text := headingText(emph, doc.Source)
		if text == "" {
			return ast.WalkContinue, nil
		}

		// Skip if ends with punctuation.
		runes := []rune(text)
		lastRune := runes[len(runes)-1]
		if strings.ContainsRune(punct, lastRune) {
			return ast.WalkContinue, nil
		}

		line := 1
		if para.Lines() != nil && para.Lines().Len() > 0 {
			seg := para.Lines().At(0)
			line = countLine(doc.Source, seg.Start)
		}
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    line,
			Column:  1,
			Message: "Emphasis used instead of a heading",
		})
		return ast.WalkContinue, nil
	})

	return violations
}

