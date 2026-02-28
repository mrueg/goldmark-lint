package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD026 checks for trailing punctuation in headings.
type MD026 struct {
	// Punctuation is the set of punctuation characters to check (default ".,;:!?").
	Punctuation string `json:"punctuation"`
}

func (r MD026) ID() string          { return "MD026" }
func (r MD026) Description() string { return "Trailing punctuation in heading" }

const defaultMD026Punctuation = ".,;:!。，；：！"

func (r MD026) punct() string {
	if r.Punctuation == "" {
		return defaultMD026Punctuation
	}
	return r.Punctuation
}

// md026openATXRE is reused from md019/md018 patterns – the open ATX heading prefix.
// We detect open ATX headings that don't match closedATXRE.

func (r MD026) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	punct := r.punct()
	n := len(doc.Lines)

	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		var text string

		// Closed ATX heading: ## Heading ##
		if m := closedATXRE.FindStringSubmatch(line); m != nil {
			text = strings.TrimSpace(m[3])
		} else if isOpenATXHeading(line) {
			// Open ATX heading: # Heading
			idx := strings.Index(line, " ")
			if idx >= 0 {
				text = strings.TrimSpace(line[idx:])
			}
		} else if i+1 < n && !mask[i+1] {
			// Setext heading: non-blank text line followed by underline.
			nextTrimmed := strings.TrimSpace(doc.Lines[i+1])
			isSetextUnderline := len(nextTrimmed) > 0 &&
				(strings.Trim(nextTrimmed, "=") == "" || strings.Trim(nextTrimmed, "-") == "")
			curTrimmed := strings.TrimLeft(line, " ")
			if isSetextUnderline && len(curTrimmed) > 0 && curTrimmed[0] != '#' {
				text = strings.TrimSpace(line)
			}
		}

		if len(text) == 0 {
			continue
		}
		runes := []rune(text)
		lastRune := runes[len(runes)-1]
		if strings.ContainsRune(punct, lastRune) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: fmt.Sprintf("Trailing punctuation in heading [Punctuation: '%c']", lastRune),
			})
		}
	}
	return violations
}

func (r MD026) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	punct := r.punct()
	n := len(lines)

	for i, line := range lines {
		if mask[i] {
			continue
		}
		// Closed ATX heading.
		if m := closedATXRE.FindStringSubmatch(line); m != nil {
			indent, opening, middle, closing := m[1], m[2], m[3], m[4]
			text := strings.TrimSpace(middle)
			runes := []rune(text)
			if len(runes) > 0 && strings.ContainsRune(punct, runes[len(runes)-1]) {
				trimmed := strings.TrimRightFunc(text, func(ru rune) bool {
					return strings.ContainsRune(punct, ru)
				})
				lines[i] = indent + opening + " " + trimmed + " " + closing
			}
			continue
		}
		// Open ATX heading.
		if isOpenATXHeading(line) {
			idx := strings.Index(line, " ")
			if idx < 0 {
				continue
			}
			prefix := line[:idx+1]
			text := line[idx+1:]
			runes := []rune(strings.TrimSpace(text))
			if len(runes) > 0 && strings.ContainsRune(punct, runes[len(runes)-1]) {
				trimmed := strings.TrimRightFunc(strings.TrimSpace(text), func(ru rune) bool {
					return strings.ContainsRune(punct, ru)
				})
				lines[i] = prefix + trimmed
			}
			continue
		}
		// Setext heading text line.
		if i+1 < n && !mask[i+1] {
			nextTrimmed := strings.TrimSpace(lines[i+1])
			isSetextUnderline := len(nextTrimmed) > 0 &&
				(strings.Trim(nextTrimmed, "=") == "" || strings.Trim(nextTrimmed, "-") == "")
			curTrimmed := strings.TrimLeft(line, " ")
			if isSetextUnderline && len(curTrimmed) > 0 && curTrimmed[0] != '#' {
				text := strings.TrimSpace(line)
				runes := []rune(text)
				if len(runes) > 0 && strings.ContainsRune(punct, runes[len(runes)-1]) {
					leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))
					trimmed := strings.TrimRightFunc(text, func(ru rune) bool {
						return strings.ContainsRune(punct, ru)
					})
					lines[i] = strings.Repeat(" ", leadingSpaces) + trimmed
				}
			}
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// isOpenATXHeading reports whether line is an open ATX heading (starts with 0–3 spaces,
// then 1–6 '#', then a space – but is not a closed ATX heading).
func isOpenATXHeading(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	leadingSpaces := len(line) - len(trimmed)
	if leadingSpaces > 3 {
		return false
	}
	if len(trimmed) < 2 || trimmed[0] != '#' {
		return false
	}
	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
	}
	if i > 6 {
		return false
	}
	if i >= len(trimmed) || trimmed[i] != ' ' {
		return false
	}
	// Make sure it is not a closed ATX heading (closedATXRE handles those).
	return closedATXRE.FindStringIndex(line) == nil
}
