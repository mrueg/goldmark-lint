package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD049 checks that emphasis markers use a consistent style (asterisk or underscore).
type MD049 struct {
	// Style is "consistent" (default), "asterisk", or "underscore".
	Style string `json:"style"`
}

func (r MD049) ID() string          { return "MD049" }
func (r MD049) Aliases() []string   { return []string{"emphasis-style"} }
func (r MD049) Description() string { return "Emphasis style should be consistent" }

// md049StarRE matches single-asterisk emphasis *text* (not **).
// The first character inside the emphasis must not be whitespace (per CommonMark:
// a left-flanking delimiter run cannot be followed by Unicode whitespace).
var md049StarRE = regexp.MustCompile(`(?:^|[^*])(\*(?:[^ \t*\n][^*\n]*)\*)(?:[^*]|$)`)

// md049UnderRE matches single-underscore emphasis _text_ (not __).
// The first character inside the emphasis must not be whitespace.
var md049UnderRE = regexp.MustCompile(`(?:^|[^_])(_(?:[^ \t_\n][^_\n]*)_)(?:[^_]|$)`)

func (r MD049) Check(doc *lint.Document) []lint.Violation {
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
		if !ok || emph.Level != 1 {
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

		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    inlineNodeLine(emph, doc.Source),
			Column:  1,
			Message: fmt.Sprintf("Emphasis style [Expected: %s; Actual: %s]", expected, actual),
		})
		return ast.WalkContinue, nil
	})

	return violations
}

func (r MD049) Fix(source []byte) []byte {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	firstStyle := ""

	// Determine first style.
	for i, line := range lines {
		if mask[i] {
			continue
		}
		cleaned := blankCodeSpans(line)
		styles := emphasisStylesOnLine(cleaned)
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
			// Replace _text_ with *text* (not __text__).
			lines[i] = md049ReplaceUnderscore(line)
		case "underscore":
			// Replace *text* with _text_ (not **text**).
			lines[i] = md049ReplaceAsterisk(line)
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// md049ReplaceUnderscore replaces _text_ (not __text__) with *text*.
// It is code-span-aware: replacements are only applied outside of code spans.
// The first character inside the emphasis must not be whitespace (per CommonMark).
func md049ReplaceUnderscore(line string) string {
	cleaned := blankCodeSpans(line)
	re := regexp.MustCompile(`(?:^|([^_]))(_(?:[^ \t_\n][^_\n]*)_)(?:[^_]|$)`)
	innerRE := regexp.MustCompile(`_([^ \t_\n][^_\n]*)_`)
	return applyReplacementsFromCleaned(line, cleaned, re, func(seg string) string {
		sub := innerRE.FindStringSubmatch(seg)
		if sub == nil {
			return seg
		}
		return strings.Replace(seg, sub[0], "*"+sub[1]+"*", 1)
	})
}

// md049ReplaceAsterisk replaces *text* (not **text**) with _text_.
// It is code-span-aware: replacements are only applied outside of code spans.
// The first character inside the emphasis must not be whitespace (per CommonMark),
// which prevents list bullet markers (* item) from being misidentified as emphasis.
func md049ReplaceAsterisk(line string) string {
	cleaned := blankCodeSpans(line)
	re := regexp.MustCompile(`(?:^|([^*]))(\*(?:[^ \t*\n][^*\n]*)\*)(?:[^*]|$)`)
	innerRE := regexp.MustCompile(`\*([^ \t*\n][^*\n]*)\*`)
	return applyReplacementsFromCleaned(line, cleaned, re, func(seg string) string {
		sub := innerRE.FindStringSubmatch(seg)
		if sub == nil {
			return seg
		}
		return strings.Replace(seg, sub[0], "_"+sub[1]+"_", 1)
	})
}

// applyReplacementsFromCleaned finds match positions using the cleaned line
// (code spans blanked) and applies the replace function to those segments of
// the original line. Processing right-to-left preserves byte offsets.
func applyReplacementsFromCleaned(line, cleaned string, re *regexp.Regexp, replace func(string) string) string {
	allMatches := re.FindAllStringIndex(cleaned, -1)
	if len(allMatches) == 0 {
		return line
	}
	result := []byte(line)
	for i := len(allMatches) - 1; i >= 0; i-- {
		start, end := allMatches[i][0], allMatches[i][1]
		seg := string(result[start:end])
		newSeg := replace(seg)
		result = append(result[:start], append([]byte(newSeg), result[end:]...)...)
	}
	return string(result)
}

// emphasisStylesOnLine returns the emphasis styles used on this line ("asterisk", "underscore").
// Used by the Fix method to determine the first emphasis style in the document.
func emphasisStylesOnLine(line string) []string {
	var styles []string
	if md049StarRE.MatchString(line) {
		styles = append(styles, "asterisk")
	}
	if md049UnderRE.MatchString(line) {
		styles = append(styles, "underscore")
	}
	return styles
}
