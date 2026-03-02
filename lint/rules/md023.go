package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD023 checks that headings start at the beginning of the line.
type MD023 struct{}

func (r MD023) ID() string          { return "MD023" }
func (r MD023) Aliases() []string   { return []string{"heading-start-left"} }
func (r MD023) Description() string { return "Headings must start at the beginning of the line" }

// md023atxRE matches an ATX heading with 1–3 leading spaces.
var md023atxRE = regexp.MustCompile(`^ {1,3}#{1,6}( |$)`)

func (r MD023) Check(doc *lint.Document) []lint.Violation {
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
		if strings.HasPrefix(line, " ") {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    lineNum,
				Column:  1,
				Message: "Headings must start at the beginning of the line",
			})
		}
		return ast.WalkContinue, nil
	})
	return violations
}

func (r MD023) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	n := len(lines)

	for i, line := range lines {
		if mask[i] {
			continue
		}
		// Fix ATX heading with leading spaces.
		if md023atxRE.MatchString(line) {
			lines[i] = strings.TrimLeft(line, " ")
			continue
		}
		// Fix setext heading text and underline lines.
		if i+1 < n && !mask[i+1] {
			curTrimmed := strings.TrimLeft(line, " ")
			nextTrimmed := strings.TrimSpace(lines[i+1])
			isSetextUnderline := len(nextTrimmed) > 0 &&
				(strings.Trim(nextTrimmed, "=") == "" || strings.Trim(nextTrimmed, "-") == "")
			if isSetextUnderline && strings.HasPrefix(line, " ") &&
				len(curTrimmed) > 0 && curTrimmed[0] != '#' {
				lines[i] = strings.TrimLeft(line, " ")
				lines[i+1] = strings.TrimLeft(lines[i+1], " ")
			}
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
