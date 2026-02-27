package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD044 checks that proper names have correct capitalization.
type MD044 struct {
	// Names is the list of proper names with correct capitalization.
	Names []string `json:"names"`
	// CodeBlocks controls whether to check inside code blocks (default true).
	CodeBlocks bool `json:"code_blocks"`
	// HTMLElements controls whether to check inside HTML elements (default true).
	HTMLElements bool `json:"html_elements"`
}

func (r MD044) ID() string          { return "MD044" }
func (r MD044) Description() string { return "Proper names should have the correct capitalization" }

func (r MD044) Check(doc *lint.Document) []lint.Violation {
	if len(r.Names) == 0 {
		return nil
	}

	mask := fencedCodeBlockMask(doc.Lines)
	var violations []lint.Violation

	for _, name := range r.Names {
		if name == "" {
			continue
		}
		// Build a case-insensitive regex to find incorrect capitalizations.
		pattern := regexp.QuoteMeta(name)
		re := regexp.MustCompile(`(?i)\b` + pattern + `\b`)

		for i, line := range doc.Lines {
			if mask[i] && !r.CodeBlocks {
				continue
			}
			// Skip code spans if CodeBlocks is false.
			checkLine := line
			if !r.CodeBlocks {
				checkLine = blankCodeSpans(line)
			}
			matches := re.FindAllStringIndex(checkLine, -1)
			for _, m := range matches {
				found := line[m[0]:m[1]]
				if found == name {
					continue // Correct capitalization.
				}
				// Check word boundary - found must match case-insensitively.
				if !strings.EqualFold(found, name) {
					continue
				}
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  m[0] + 1,
					Message: "Proper names should have the correct capitalization [Expected: " + name + "; Actual: " + found + "]",
				})
			}
		}
	}
	return violations
}

func (r MD044) Fix(source []byte) []byte {
	if len(r.Names) == 0 {
		return source
	}

	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)

	for _, name := range r.Names {
		if name == "" {
			continue
		}
		pattern := regexp.QuoteMeta(name)
		re := regexp.MustCompile(`(?i)\b` + pattern + `\b`)

		for i, line := range lines {
			if mask[i] && !r.CodeBlocks {
				continue
			}
			lines[i] = re.ReplaceAllStringFunc(line, func(found string) string {
				if found == name {
					return found
				}
				if strings.EqualFold(found, name) {
					return name
				}
				return found
			})
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// blankCodeSpans replaces code span content with spaces.
func blankCodeSpans(line string) string {
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
		end := i
		for end < len(result) {
			if result[end] == '`' {
				k := end
				for k < len(result) && result[k] == '`' {
					k++
				}
				if k-end == tickLen {
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
