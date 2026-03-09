package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD035 checks that horizontal rules use a consistent style.
type MD035 struct {
	// Style is the required HR style: "consistent" (default) or a specific pattern like "---".
	Style string `json:"style"`
}

func (r MD035) ID() string          { return "MD035" }
func (r MD035) Aliases() []string   { return []string{"hr-style"} }
func (r MD035) Description() string { return "Horizontal rule style" }

// md035HRRe matches a horizontal rule line (3+ of the same character with optional spaces).
var md035HRRE = regexp.MustCompile(`^[ \t]*((?:\*[ \t]*){3,}|(?:-[ \t]*){3,}|(?:_[ \t]*){3,})[ \t]*$`)

func isHR(line string) bool {
	return md035HRRE.MatchString(line)
}

func normalizeHR(line string) string {
	return strings.TrimSpace(line)
}

// Fix applies MD035 to source by replacing all horizontal rules with the
// required style. If style is "consistent", the first HR's style is used.
func (r MD035) Fix(source []byte) []byte {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	firstStyle := ""
	expected := style

	// First pass: determine the expected style for "consistent".
	if style == "consistent" {
		for i, line := range lines {
			if !mask[i] && isHR(line) {
				firstStyle = normalizeHR(line)
				break
			}
		}
		if firstStyle != "" {
			expected = firstStyle
		}
	}

	if expected == "consistent" {
		// No HRs found; nothing to do.
		return source
	}

	// Second pass: replace non-conforming HRs.
	changed := false
	for i, line := range lines {
		if mask[i] || !isHR(line) {
			continue
		}
		actual := normalizeHR(line)
		if actual != expected {
			// Preserve any leading whitespace.
			leading := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = leading + expected
			changed = true
		}
	}
	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD035) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	var violations []lint.Violation
	firstStyle := ""
	mask := fencedCodeBlockMask(doc.Lines)

	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if !isHR(line) {
			continue
		}
		actual := normalizeHR(line)
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
				Message: fmt.Sprintf("Horizontal rule style [Expected: %s; Actual: %s]", expected, actual),
			})
		}
	}
	return violations
}
