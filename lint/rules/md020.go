package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
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
	mask := fencedCodeBlockMask(doc.Lines)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		m := closedATXRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		middle := m[3]
		if !strings.HasPrefix(middle, " ") || !strings.HasSuffix(middle, " ") {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "No space inside hashes on closed ATX style heading",
			})
		}
	}
	return violations
}
