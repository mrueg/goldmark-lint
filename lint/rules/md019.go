package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD019 checks that there is only one space after the hash on ATX style headings.
type MD019 struct{}

func (r MD019) ID() string          { return "MD019" }
func (r MD019) Aliases() []string   { return []string{"no-multiple-space-atx"} }
func (r MD019) Description() string { return "Multiple spaces after hash on ATX style heading" }

// md019RE matches an ATX heading line where there are 2+ spaces after the hashes.
// Group 1: indent, Group 2: hashes, Group 3: two or more spaces, Group 4: rest.
var md019RE = regexp.MustCompile(`^( {0,3})(#{1,6})( {2,})(.*)$`)

func (r MD019) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if !mask[i] {
			lines[i] = md019RE.ReplaceAllString(line, "$1$2 $4")
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD019) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		lineNum := headingSourceLine(h, doc.Source)
		if lineNum == 0 {
			return ast.WalkContinue, nil
		}
		line := doc.Lines[lineNum-1]
		// Closed ATX headings (e.g. "##  Heading  ##") are handled by MD021;
		// MD019 should only flag open ATX headings.
		if closedATXRE.MatchString(line) {
			return ast.WalkContinue, nil
		}
		if md019RE.MatchString(line) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    lineNum,
				Column:  1,
				Message: "Multiple spaces after hash on ATX style heading",
			})
		}
		return ast.WalkContinue, nil
	})
	return violations
}
