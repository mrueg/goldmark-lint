package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD037 checks for spaces inside emphasis markers.
type MD037 struct{}

func (r MD037) ID() string          { return "MD037" }
func (r MD037) Description() string { return "Spaces inside emphasis markers" }

// md037 regexes detect emphasis markers with a space immediately after the
// opening marker or immediately before the closing marker.
// We match pairs so we don't flag lone asterisks (e.g. list items).
var (
	md037DoubleStarRE = regexp.MustCompile(`\*\* [^\n*]* \*\*|\*\* [^\n*]*[^\n* ](?:\*\*)|(?:[^ \n*][^\n*]*) \*\*`)
	md037SingleStarRE = regexp.MustCompile(`(?:^|[^*])\* [^\n*]* \*(?:[^*]|$)`)
	md037DoubleUndRE  = regexp.MustCompile(`__ [^\n_]* __|__ [^\n_]*[^\n_ ](?:__)|(?:[^ \n_][^\n_]*) __`)
	md037SingleUndRE  = regexp.MustCompile(`(?:^|[^_])_ [^\n_]* _(?:[^_]|$)`)
)

// stripCodeSpans replaces code span content with spaces to avoid false positives.
func stripCodeSpans(line string) string {
	result := []byte(line)
	i := 0
	for i < len(result) {
		if result[i] != '`' {
			i++
			continue
		}
		// Count opening backticks
		start := i
		for i < len(result) && result[i] == '`' {
			i++
		}
		tickLen := i - start
		// Find matching closing backtick sequence
		end := i
		for end < len(result) {
			if result[end] == '`' {
				k := end
				for k < len(result) && result[k] == '`' {
					k++
				}
				if k-end == tickLen {
					// Replace entire span including backticks with spaces
					for x := start; x < k; x++ {
						result[x] = ' '
					}
					i = k
					break
				}
				end = k
			} else {
				end++
			}
		}
	}
	return string(result)
}

func hasEmphasisSpaces(line string) bool {
	s := stripCodeSpans(line)
	return md037DoubleStarRE.MatchString(s) ||
		md037SingleStarRE.MatchString(s) ||
		md037DoubleUndRE.MatchString(s) ||
		md037SingleUndRE.MatchString(s)
}

func fixEmphasisSpaces(line string) string {
	// Apply fixes in order: double markers before single to avoid partial replacement
	s := line
	// ** text ** -> **text**
	s = regexp.MustCompile(`\*\* ([^\n*]*) \*\*`).ReplaceAllStringFunc(s, func(m string) string {
		inner := m[3 : len(m)-3]
		return "**" + strings.TrimSpace(inner) + "**"
	})
	// * text * -> *text* (but not **)
	s = regexp.MustCompile(`(?:^|([^*]))\* ([^\n*]*) \*(?:[^*]|$)`).ReplaceAllStringFunc(s, func(m string) string {
		// find the * ... * part within match
		sub := regexp.MustCompile(`\* ([^\n*]*) \*`).FindString(m)
		if sub == "" {
			return m
		}
		inner := sub[2 : len(sub)-2]
		fixed := "*" + strings.TrimSpace(inner) + "*"
		return strings.Replace(m, sub, fixed, 1)
	})
	// __ text __ -> __text__
	s = regexp.MustCompile(`__ ([^\n_]*) __`).ReplaceAllStringFunc(s, func(m string) string {
		inner := m[3 : len(m)-3]
		return "__" + strings.TrimSpace(inner) + "__"
	})
	// _ text _ -> _text_ (but not __)
	s = regexp.MustCompile(`(?:^|([^_]))_ ([^\n_]*) _(?:[^_]|$)`).ReplaceAllStringFunc(s, func(m string) string {
		sub := regexp.MustCompile(`_ ([^\n_]*) _`).FindString(m)
		if sub == "" {
			return m
		}
		inner := sub[2 : len(sub)-2]
		fixed := "_" + strings.TrimSpace(inner) + "_"
		return strings.Replace(m, sub, fixed, 1)
	})
	return s
}

func (r MD037) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if !mask[i] {
			lines[i] = fixEmphasisSpaces(line)
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD037) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if hasEmphasisSpaces(line) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "Spaces inside emphasis markers",
			})
		}
	}
	return violations
}
