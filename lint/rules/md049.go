package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD049 checks that emphasis markers use a consistent style (asterisk or underscore).
type MD049 struct {
	// Style is "consistent" (default), "asterisk", or "underscore".
	Style string `json:"style"`
}

func (r MD049) ID() string          { return "MD049" }
func (r MD049) Description() string { return "Emphasis style should be consistent" }

// md049StarRE matches single-asterisk emphasis *text* (not **).
var md049StarRE = regexp.MustCompile(`(?:^|[^*])(\*(?:[^*\n]+)\*)(?:[^*]|$)`)

// md049UnderRE matches single-underscore emphasis _text_ (not __).
var md049UnderRE = regexp.MustCompile(`(?:^|[^_])(_(?:[^_\n]+)_)(?:[^_]|$)`)

func (r MD049) Check(doc *lint.Document) []lint.Violation {
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

		hasStar := md049StarRE.MatchString(cleaned)
		hasUnder := md049UnderRE.MatchString(cleaned)

		if !hasStar && !hasUnder {
			continue
		}

		// Determine actual style on this line.
		for _, actual := range emphasisStylesOnLine(cleaned) {
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
					Message: fmt.Sprintf("Emphasis style [Expected: %s; Actual: %s]", expected, actual),
				})
				break
			}
		}
		// Update firstStyle from this line if consistent and not yet set.
		if style == "consistent" && firstStyle == "" {
			styles := emphasisStylesOnLine(cleaned)
			if len(styles) > 0 {
				firstStyle = styles[0]
			}
		}
	}
	return violations
}

// emphasisStylesOnLine returns the emphasis styles used on this line ("asterisk", "underscore").
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
func md049ReplaceUnderscore(line string) string {
	re := regexp.MustCompile(`(?:^|([^_]))(_(?:[^_\n]+)_)(?:[^_]|$)`)
	return re.ReplaceAllStringFunc(line, func(m string) string {
		sub := regexp.MustCompile(`_([^_\n]+)_`).FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		return strings.Replace(m, sub[0], "*"+sub[1]+"*", 1)
	})
}

// md049ReplaceAsterisk replaces *text* (not **text**) with _text_.
func md049ReplaceAsterisk(line string) string {
	re := regexp.MustCompile(`(?:^|([^*]))(\*(?:[^*\n]+)\*)(?:[^*]|$)`)
	return re.ReplaceAllStringFunc(line, func(m string) string {
		sub := regexp.MustCompile(`\*([^*\n]+)\*`).FindStringSubmatch(m)
		if sub == nil {
			return m
		}
		return strings.Replace(m, sub[0], "_"+sub[1]+"_", 1)
	})
}
