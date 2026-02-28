package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD011 checks for reversed link syntax: (text)[url] instead of [text](url).
type MD011 struct{}

func (r MD011) ID() string          { return "MD011" }
func (r MD011) Aliases() []string   { return []string{"no-reversed-links"} }
func (r MD011) Description() string { return "Reversed link syntax" }

// reversedLinkRE matches the pattern (text)[url] which is a reversed link.
var reversedLinkRE = regexp.MustCompile(`\(([^)\n]+)\)\[([^\]\n]+)\]`)

func (r MD011) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if !mask[i] {
			lines[i] = reversedLinkRE.ReplaceAllString(line, "[$1]($2)")
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD011) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if reversedLinkRE.MatchString(line) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "Reversed link syntax",
			})
		}
	}
	return violations
}
