package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD046 checks code block style consistency.
type MD046 struct {
	// Style is "consistent" (default), "fenced", or "indented".
	Style string `json:"style"`
}

func (r MD046) ID() string          { return "MD046" }
func (r MD046) Aliases() []string   { return []string{"code-block-style"} }
func (r MD046) Description() string { return "Code block style" }

func (r MD046) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	var violations []lint.Violation
	lines := doc.Lines
	n := len(lines)
	firstStyle := ""
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0

	for i := 0; i < n; i++ {
		line := lines[i]

		if !inFence {
			isFence, fc, fl := detectFence(line)
			if isFence {
				inFence = true
				fenceChar = fc
				fenceLen = fl
				blockStyle := "fenced"
				expected := style
				if style == "consistent" {
					if firstStyle == "" {
						firstStyle = blockStyle
					}
					expected = firstStyle
				}
				if blockStyle != expected {
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    i + 1,
						Column:  1,
						Message: fmt.Sprintf("Code block style [Expected: %s; Actual: %s]", expected, blockStyle),
					})
				}
				continue
			}
			// Check for indented code block: 4+ spaces or tab at start
			// but not inside a list or other block context.
			// A simple heuristic: line starts with 4 spaces or a tab,
			// previous line is blank or it's also indented.
			if isIndentedCodeLine(line) {
				// Check prev line is blank (start of indented block).
				if i > 0 && strings.TrimSpace(lines[i-1]) == "" && !isIndentedCodeLine(lines[i-1]) {
					blockStyle := "indented"
					expected := style
					if style == "consistent" {
						if firstStyle == "" {
							firstStyle = blockStyle
						}
						expected = firstStyle
					}
					if blockStyle != expected {
						violations = append(violations, lint.Violation{
							Rule:    r.ID(),
							Line:    i + 1,
							Column:  1,
							Message: fmt.Sprintf("Code block style [Expected: %s; Actual: %s]", expected, blockStyle),
						})
					}
				}
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

// isIndentedCodeLine returns true if the line is an indented code block line.
func isIndentedCodeLine(line string) bool {
	if len(line) == 0 {
		return false
	}
	if line[0] == '\t' {
		return true
	}
	if strings.HasPrefix(line, "    ") {
		return true
	}
	return false
}
