package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD040 checks that fenced code blocks have a language specifier.
type MD040 struct{}

func (r MD040) ID() string          { return "MD040" }
func (r MD040) Description() string { return "Fenced code blocks should have a language specified" }

// md040FenceRE matches a fenced code block opening line.
// Group 1: indent, Group 2: fence marker, Group 3: language (may be empty).
var md040FenceRE = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})(.*)$")

func (r MD040) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	lines := doc.Lines
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if !inFence {
			m := md040FenceRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			fc := trimmed[0]
			j := 0
			for j < len(trimmed) && trimmed[j] == fc {
				j++
			}
			if j < 3 {
				continue
			}
			inFence = true
			fenceChar = fc
			fenceLen = j
			lang := strings.TrimSpace(m[3])
			if lang == "" {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  1,
					Message: "Fenced code blocks should have a language specified",
				})
			}
		} else {
			j := 0
			for j < len(trimmed) && trimmed[j] == fenceChar {
				j++
			}
			if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" && len(trimmed) > 0 && trimmed[0] == fenceChar {
				inFence = false
			}
		}
	}
	return violations
}
