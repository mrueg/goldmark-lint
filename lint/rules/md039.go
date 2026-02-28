package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD039 checks for spaces inside link text.
type MD039 struct{}

func (r MD039) ID() string          { return "MD039" }
func (r MD039) Aliases() []string   { return []string{"no-space-in-links"} }
func (r MD039) Description() string { return "Spaces inside link text" }

// md039RE matches a link with leading or trailing space in its text.
// Captures: full match includes [<space>content](  or  [content<space>](
var md039RE = regexp.MustCompile(`\[(\s[^\]\n]*|[^\]\n]*\s)\]\(`)

func (r MD039) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if mask[i] {
			continue
		}
		lines[i] = md039RE.ReplaceAllStringFunc(line, func(match string) string {
			// match is like "[ text ](" - trim spaces from inner content
			inner := match[1 : len(match)-2] // content between [ and ](
			trimmed := strings.TrimSpace(inner)
			return "[" + trimmed + "]("
		})
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD039) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if md039RE.MatchString(line) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "Spaces inside link text",
			})
		}
	}
	return violations
}
