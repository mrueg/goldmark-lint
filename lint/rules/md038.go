package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD038 checks for spaces inside code span elements.
type MD038 struct{}

func (r MD038) ID() string          { return "MD038" }
func (r MD038) Description() string { return "Spaces inside code span elements" }

// findCodeSpanViolations returns true if the line has a code span with leading
// or trailing space in its content.
func findCodeSpanViolations(line string) bool {
	i := 0
	for i < len(line) {
		if line[i] != '`' {
			i++
			continue
		}
		start := i
		for i < len(line) && line[i] == '`' {
			i++
		}
		tickLen := i - start
		// Find matching closing backtick sequence
		contentStart := i
		end := i
		for end < len(line) {
			if line[end] == '`' {
				k := end
				for k < len(line) && line[k] == '`' {
					k++
				}
				if k-end == tickLen {
					content := line[contentStart:end]
					if len(content) > 0 && (content[0] == ' ' || content[len(content)-1] == ' ') {
						return true
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
	return false
}

// fixCodeSpanSpaces removes leading/trailing spaces from code span content.
func fixCodeSpanSpaces(line string) string {
	result := []byte(line)
	i := 0
	for i < len(result) {
		if result[i] != '`' {
			i++
			continue
		}
		start := i
		for i < len(result) && result[i] == '`' {
			i++
		}
		tickLen := i - start
		contentStart := i
		end := i
		for end < len(result) {
			if result[end] == '`' {
				k := end
				for k < len(result) && result[k] == '`' {
					k++
				}
				if k-end == tickLen {
					content := string(result[contentStart:end])
					trimmed := strings.TrimSpace(content)
					if trimmed != content && len(trimmed) > 0 {
						// Rebuild this code span without leading/trailing spaces
						newSpan := string(result[start:contentStart]) + trimmed + string(result[end:k])
						result = append(result[:start], append([]byte(newSpan), result[k:]...)...)
						i = start + tickLen + len(trimmed) + tickLen
					} else {
						i = k
					}
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

func (r MD038) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if !mask[i] {
			lines[i] = fixCodeSpanSpaces(line)
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD038) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if findCodeSpanViolations(line) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "Spaces inside code span elements",
			})
		}
	}
	return violations
}
