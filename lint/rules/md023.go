package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD023 checks that headings start at the beginning of the line.
type MD023 struct{}

func (r MD023) ID() string          { return "MD023" }
func (r MD023) Description() string { return "Headings must start at the beginning of the line" }

// md023atxRE matches an ATX heading with 1–3 leading spaces.
var md023atxRE = regexp.MustCompile(`^ {1,3}#{1,6}( |$)`)

func (r MD023) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	n := len(doc.Lines)

	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		// ATX heading with leading spaces.
		if md023atxRE.MatchString(line) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "Headings must start at the beginning of the line",
			})
			continue
		}
		// Setext heading: non-blank text line with leading spaces followed by underline.
		if i+1 < n && !mask[i+1] {
			curTrimmed := strings.TrimLeft(line, " ")
			nextTrimmed := strings.TrimSpace(doc.Lines[i+1])
			isSetextUnderline := len(nextTrimmed) > 0 &&
				(strings.Trim(nextTrimmed, "=") == "" || strings.Trim(nextTrimmed, "-") == "")
			if isSetextUnderline && strings.HasPrefix(line, " ") &&
				len(curTrimmed) > 0 && curTrimmed[0] != '#' {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  1,
					Message: "Headings must start at the beginning of the line",
				})
			}
		}
	}
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
