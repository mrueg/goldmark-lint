package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD050 checks that strong markers use a consistent style (asterisk or underscore).
type MD050 struct {
	// Style is "consistent" (default), "asterisk", or "underscore".
	Style string `json:"style"`
}

func (r MD050) ID() string          { return "MD050" }
func (r MD050) Aliases() []string   { return []string{"strong-style"} }
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
	firstStyle := ""

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		emph, ok := n.(*ast.Emphasis)
		if !ok || emph.Level != 2 {
			return ast.WalkContinue, nil
		}

		pos := emphasisStartPos(emph)
		if pos < 0 || pos >= len(doc.Source) {
			return ast.WalkContinue, nil
		}

		var actual string
		switch doc.Source[pos] {
		case '*':
			actual = "asterisk"
		case '_':
			actual = "underscore"
		default:
			return ast.WalkContinue, nil
		}

		expected := style
		if style == "consistent" {
			if firstStyle == "" {
				firstStyle = actual
			}
			expected = firstStyle
		}

		if actual == expected {
			return ast.WalkContinue, nil
		}

		// Report opening marker violation.
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    inlineNodeLine(emph, doc.Source),
			Column:  1,
			Message: fmt.Sprintf("Strong style [Expected: %s; Actual: %s]", expected, actual),
		})
		// Report closing marker violation (markdownlint reports both opening and closing).
		// Find the last text stop position recursively in case of complex inline children.
		lastTextStop := lastTextStopInInline(emph)
		if lastTextStop > 0 && lastTextStop < len(doc.Source) {
			closingLine := countLine(doc.Source, lastTextStop)
			// Calculate column relative to the start of the line.
			lineStart := lastTextStop
			for lineStart > 0 && doc.Source[lineStart-1] != '\n' {
				lineStart--
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    closingLine,
				Column:  lastTextStop - lineStart + 1,
				Message: fmt.Sprintf("Strong style [Expected: %s; Actual: %s]", expected, actual),
			})
		}
		return ast.WalkContinue, nil
	})

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
