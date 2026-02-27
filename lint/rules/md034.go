package rules

import (
	"regexp"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD034 checks for bare URLs that are not wrapped in angle brackets or a proper link.
type MD034 struct{}

func (r MD034) ID() string          { return "MD034" }
func (r MD034) Description() string { return "Bare URL used" }

// bareURLRE matches an http or https URL within a string, stopping at whitespace
// or common punctuation characters that are unlikely to be part of the URL.
var bareURLRE = regexp.MustCompile(`https?://[^\s<>()\[\]{}'"]+`)

func (r MD034) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		t, ok := n.(*ast.Text)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Skip text inside links or images (they are properly formatted).
		if isInsideLinkOrImage(t) {
			return ast.WalkContinue, nil
		}

		seg := t.Segment
		text := string(doc.Source[seg.Start:seg.Stop])
		if !bareURLRE.MatchString(text) {
			return ast.WalkContinue, nil
		}

		line := countLine(doc.Source, seg.Start)
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    line,
			Column:  1,
			Message: "Bare URL used",
		})
		return ast.WalkContinue, nil
	})

	return violations
}

// isInsideLinkOrImage reports whether n is a descendant of a Link or Image node.
func isInsideLinkOrImage(n ast.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		switch p.(type) {
		case *ast.Link, *ast.Image:
			return true
		}
	}
	return false
}
