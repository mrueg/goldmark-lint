package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD050 checks that strong markers use a consistent style (asterisk or underscore).
type MD050 struct {
	// Style is "consistent" (default), "asterisk", or "underscore".
	Style string `json:"style"`
}

func (r MD050) ID() string          { return "MD050" }
func (r MD050) Description() string { return "Strong style should be consistent" }

// md050StarRE matches double-asterisk strong **text**.
var md050StarRE = regexp.MustCompile(`\*\*(?:[^*\n]+)\*\*`)

// md050UnderRE matches double-underscore strong __text__.
var md050UnderRE = regexp.MustCompile(`__(?:[^_\n]+)__`)

func (r MD050) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	firstStyle := ""

	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		cleaned := blankCodeSpans(line)

		hasStar := md050StarRE.MatchString(cleaned)
		hasUnder := md050UnderRE.MatchString(cleaned)

		if !hasStar && !hasUnder {
			continue
		}

		for _, actual := range strongStylesOnLine(cleaned) {
			expected := style
			if style == "consistent" {
				if firstStyle == "" {
					firstStyle = actual
				}
				expected = firstStyle
			}
			if actual != expected {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  1,
					Message: fmt.Sprintf("Strong style [Expected: %s; Actual: %s]", expected, actual),
				})
				break
			}
		}
		if style == "consistent" && firstStyle == "" {
			styles := strongStylesOnLine(cleaned)
			if len(styles) > 0 {
				firstStyle = styles[0]
			}
		}
	}
	return violations
}

// strongStylesOnLine returns the strong styles used on this line.
func strongStylesOnLine(line string) []string {
	var styles []string
	if md050StarRE.MatchString(line) {
		styles = append(styles, "asterisk")
	}
	if md050UnderRE.MatchString(line) {
		styles = append(styles, "underscore")
	}
	return styles
}

func (r MD050) Fix(source []byte) []byte {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	firstStyle := ""

	for i, line := range lines {
		if mask[i] {
			continue
		}
		cleaned := blankCodeSpans(line)
		styles := strongStylesOnLine(cleaned)
		if len(styles) > 0 {
			if style == "consistent" {
				firstStyle = styles[0]
			}
			break
		}
	}

	target := style
	if style == "consistent" {
		target = firstStyle
	}
	if target == "" {
		return source
	}

	for i, line := range lines {
		if mask[i] {
			continue
		}
		switch target {
		case "asterisk":
			lines[i] = md050UnderRE.ReplaceAllStringFunc(line, func(m string) string {
				inner := m[2 : len(m)-2]
				return "**" + inner + "**"
			})
		case "underscore":
			lines[i] = md050StarRE.ReplaceAllStringFunc(line, func(m string) string {
				inner := m[2 : len(m)-2]
				return "__" + inner + "__"
			})
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
