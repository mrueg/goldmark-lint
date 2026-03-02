package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD059 checks that link text is descriptive (not generic).
type MD059 struct {
	// ProhibitedTexts is a list of generic link text phrases (default ["click here","here","link","more"]).
	ProhibitedTexts []string `json:"prohibited_texts"`
}

func (r MD059) ID() string          { return "MD059" }
func (r MD059) Aliases() []string   { return []string{"descriptive-link-text"} }
func (r MD059) Description() string { return "Link text should be descriptive" }

func (r MD059) prohibited() []string {
	if len(r.ProhibitedTexts) == 0 {
		return []string{"click here", "here", "link", "more"}
	}
	return r.ProhibitedTexts
}

func (r MD059) Check(doc *lint.Document) []lint.Violation {
	prohibited := r.prohibited()
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}
		// Collect all text from the link's children (including inline elements
		// like emphasis, strong, code spans, etc.).
		text := strings.TrimSpace(string(inlineNodeText(link, doc.Source)))
		for _, p := range prohibited {
			if strings.EqualFold(text, p) {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    inlineNodeLine(link, doc.Source),
					Column:  1,
					Message: "Link text should be descriptive [Text: " + text + "]",
				})
				break
			}
		}
		return ast.WalkContinue, nil
	})
	return violations
}
