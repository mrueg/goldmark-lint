package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD020 checks that closed ATX style headings have spaces inside the hashes.
type MD020 struct{}

func (r MD020) ID() string          { return "MD020" }
func (r MD020) Aliases() []string   { return []string{"no-missing-space-closed-atx"} }
func (r MD020) Description() string { return "No space inside hashes on closed ATX style heading" }

// closedATXRE matches a closed ATX heading line.
// Group 1: indent, Group 2: opening hashes, Group 3: middle content, Group 4: closing hashes.
var closedATXRE = regexp.MustCompile(`^( {0,3})(#{1,6})(.+?)(#+)\s*$`)

func (r MD020) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if mask[i] {
			continue
		}
		m := closedATXRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		indent, opening, middle, closing := m[1], m[2], m[3], m[4]
		fixed := false
		if !strings.HasPrefix(middle, " ") {
			middle = " " + middle
			fixed = true
		}
		if !strings.HasSuffix(middle, " ") {
			middle = middle + " "
			fixed = true
		}
		if fixed {
			lines[i] = indent + opening + middle + closing
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD020) Check(doc *lint.Document) []lint.Violation {
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
		m := closedATXRE.FindStringSubmatch(line)
		if m == nil {
			return ast.WalkContinue, nil
		}
		middle := m[3]
		if !strings.HasPrefix(middle, " ") || !strings.HasSuffix(middle, " ") {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    lineNum,
				Column:  1,
				Message: "No space inside hashes on closed ATX style heading",
			})
		}
		return ast.WalkContinue, nil
	})
	return violations
}
