package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD048 checks that code fences use a consistent style (backtick or tilde).
type MD048 struct {
	// Style is "consistent" (default), "backtick", or "tilde".
	Style string `json:"style"`
}

func (r MD048) ID() string          { return "MD048" }
func (r MD048) Aliases() []string   { return []string{"code-fence-style"} }
func (r MD048) Description() string { return "Code fence style" }

func (r MD048) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	var violations []lint.Violation
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	firstChar := byte(0)

	for i, line := range doc.Lines {
		if !inFence {
			isFence, fc, fl := detectFence(line)
			if !isFence {
				continue
			}
			inFence = true
			fenceChar = fc
			fenceLen = fl

			expected := byte(0)
			switch style {
			case "backtick":
				expected = '`'
			case "tilde":
				expected = '~'
			case "consistent":
				if firstChar == 0 {
					firstChar = fc
				}
				expected = firstChar
			}
			if expected != 0 && fc != expected {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  1,
					Message: fmt.Sprintf("Code fence style [Expected: %s; Actual: %s]", fenceCharName(expected), fenceCharName(fc)),
				})
			}
		} else {
			trimmed := strings.TrimLeft(line, " ")
			j := 0
			for j < len(trimmed) && trimmed[j] == fenceChar {
				j++
			}
			if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" && len(trimmed) > 0 && trimmed[0] == fenceChar {
				inFence = false
			}
		}
	}
	return violations
}

func (r MD048) Fix(source []byte) []byte {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	lines := strings.Split(string(source), "\n")
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	firstChar := byte(0)

	for i, line := range lines {
		if !inFence {
			isFence, fc, fl := detectFence(line)
			if !isFence {
				continue
			}
			inFence = true
			fenceChar = fc
			fenceLen = fl

			expected := byte(0)
			switch style {
			case "backtick":
				expected = '`'
			case "tilde":
				expected = '~'
			case "consistent":
				if firstChar == 0 {
					firstChar = fc
				}
				expected = firstChar
			}
			if expected != 0 && fc != expected {
				// Replace fence character.
				trimmed := strings.TrimLeft(line, " ")
				indent := line[:len(line)-len(trimmed)]
				// Replace opening fence chars.
				newFence := strings.Repeat(string(expected), fl)
				rest := trimmed[fl:]
				lines[i] = indent + newFence + rest
			}
		} else {
			trimmed := strings.TrimLeft(line, " ")
			j := 0
			for j < len(trimmed) && trimmed[j] == fenceChar {
				j++
			}
			if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" && len(trimmed) > 0 && trimmed[0] == fenceChar {
				inFence = false
			}
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func fenceCharName(c byte) string {
	if c == '`' {
		return "backtick"
	}
	return "tilde"
}
