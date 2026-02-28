package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD036 checks that emphasis is not used instead of a heading.
type MD036 struct {
	// Punctuation is the set of characters that, if a paragraph ends with one,
	// the emphasis check is skipped. Default: ".,;:!?。，；：！？"
	Punctuation string `json:"punctuation"`
}

func (r MD036) ID() string          { return "MD036" }
func (r MD036) Aliases() []string   { return []string{"no-emphasis-as-heading"} }
func (r MD036) Description() string { return "Emphasis used instead of a heading" }

const defaultMD036Punctuation = ".,;:!?。，；：！？"

// md036EmphRE matches a line consisting entirely of emphasis (single or double * or _).
var md036EmphRE = regexp.MustCompile(`^(?:\*\*[^*\n]+\*\*|\*[^*\n]+\*|__[^_\n]+__|_[^_\n]+_)$`)

func (r MD036) punct() string {
	if r.Punctuation == "" {
		return defaultMD036Punctuation
	}
	return r.Punctuation
}

func (r MD036) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	punct := r.punct()

	// We need to find single-line paragraphs consisting entirely of emphasis.
	// A paragraph is a non-blank line not inside a code block.
	// We check: if the trimmed line matches the emphasis RE and the text inside
	// doesn't end with punctuation.
	lines := doc.Lines
	n := len(lines)

	for i, line := range lines {
		if mask[i] {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !md036EmphRE.MatchString(trimmed) {
			continue
		}
		// Check that surrounding context is paragraph (blank lines before/after,
		// or start/end of document).
		prevBlank := i == 0 || strings.TrimSpace(lines[i-1]) == ""
		nextBlank := i == n-1 || strings.TrimSpace(lines[i+1]) == ""
		if !prevBlank || !nextBlank {
			continue
		}
		// Extract the inner text of the emphasis marker.
		inner := extractEmphasisText(trimmed)
		if inner == "" {
			continue
		}
		// Skip if ends with punctuation.
		runes := []rune(inner)
		lastRune := runes[len(runes)-1]
		if strings.ContainsRune(punct, lastRune) {
			continue
		}
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    i + 1,
			Column:  1,
			Message: "Emphasis used instead of a heading",
		})
	}
	return violations
}

// extractEmphasisText returns the inner text of a single-emphasis span.
func extractEmphasisText(s string) string {
	if strings.HasPrefix(s, "**") && strings.HasSuffix(s, "**") {
		return s[2 : len(s)-2]
	}
	if strings.HasPrefix(s, "__") && strings.HasSuffix(s, "__") {
		return s[2 : len(s)-2]
	}
	if strings.HasPrefix(s, "*") && strings.HasSuffix(s, "*") {
		return s[1 : len(s)-1]
	}
	if strings.HasPrefix(s, "_") && strings.HasSuffix(s, "_") {
		return s[1 : len(s)-1]
	}
	return ""
}
