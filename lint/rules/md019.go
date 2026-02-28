package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
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
	mask := fencedCodeBlockMask(doc.Lines)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if md019RE.MatchString(line) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "Multiple spaces after hash on ATX style heading",
			})
		}
	}
	return violations
}
