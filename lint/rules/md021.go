package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD021 checks that closed ATX style headings have only one space inside the hashes.
type MD021 struct{}

func (r MD021) ID() string { return "MD021" }
func (r MD021) Description() string {
	return "Multiple spaces inside hashes on closed ATX style heading"
}

func (r MD021) Fix(source []byte) []byte {
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
		if strings.HasPrefix(middle, "  ") {
			middle = " " + strings.TrimLeft(middle, " ")
			fixed = true
		}
		if strings.HasSuffix(middle, "  ") {
			middle = strings.TrimRight(middle, " ") + " "
			fixed = true
		}
		if fixed {
			lines[i] = indent + opening + middle + closing
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD021) Check(doc *lint.Document) []lint.Violation {
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
		if strings.HasPrefix(middle, "  ") || strings.HasSuffix(middle, "  ") {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "Multiple spaces inside hashes on closed ATX style heading",
			})
		}
	}
	return violations
}
