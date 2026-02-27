package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD018 checks that there is a space after the hash on ATX style headings.
type MD018 struct{}

func (r MD018) ID() string          { return "MD018" }
func (r MD018) Description() string { return "No space after hash on ATX style heading" }

// md018RE matches an ATX-like heading line where the hashes are not followed by a space.
// Group 1: indent, Group 2: hashes, Group 3: rest starting with non-space, non-hash char.
var md018RE = regexp.MustCompile(`^( {0,3})(#{1,6})([^ \t#\n].*)$`)

func (r MD018) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if !mask[i] {
			lines[i] = md018RE.ReplaceAllString(line, "$1$2 $3")
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD018) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if md018RE.MatchString(line) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "No space after hash on ATX style heading",
			})
		}
	}
	return violations
}
