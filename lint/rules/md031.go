package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD031 checks that fenced code blocks are surrounded by blank lines.
type MD031 struct{}

func (r MD031) ID() string          { return "MD031" }
func (r MD031) Description() string { return "Fenced code blocks should be surrounded by blank lines" }

// detectFence returns (isFence, fenceChar, fenceLen) for a line.
func detectFence(line string) (bool, byte, int) {
	trimmed := strings.TrimLeft(line, " ")
	if len(trimmed) < 3 {
		return false, 0, 0
	}
	fc := trimmed[0]
	if fc != '`' && fc != '~' {
		return false, 0, 0
	}
	j := 0
	for j < len(trimmed) && trimmed[j] == fc {
		j++
	}
	if j >= 3 {
		return true, fc, j
	}
	return false, 0, 0
}

func (r MD031) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	var result []string
	n := len(lines)
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0

	for i, line := range lines {
		isFence, fc, fl := detectFence(line)
		if !inFence {
			if isFence {
				inFence = true
				fenceChar = fc
				fenceLen = fl
				// Insert blank line before if previous is non-blank
				if i > 0 && len(result) > 0 && strings.TrimSpace(result[len(result)-1]) != "" {
					result = append(result, "")
				}
				result = append(result, line)
			} else {
				result = append(result, line)
			}
		} else {
			// Check for closing fence
			trimmed := strings.TrimLeft(line, " ")
			j := 0
			for j < len(trimmed) && trimmed[j] == fenceChar {
				j++
			}
			if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" && trimmed[0] == fenceChar {
				inFence = false
				result = append(result, line)
				// Insert blank line after if next is non-blank
				if i < n-1 && strings.TrimSpace(lines[i+1]) != "" {
					result = append(result, "")
				}
			} else {
				result = append(result, line)
			}
		}
	}
	return []byte(strings.Join(result, "\n"))
}

func (r MD031) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	lines := doc.Lines
	n := len(lines)
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0

	for i, line := range lines {
		isFence, fc, fl := detectFence(line)
		if !inFence {
			if isFence {
				inFence = true
				fenceChar = fc
				fenceLen = fl
				// Check blank line before (not required if at document start)
				if i > 0 && strings.TrimSpace(lines[i-1]) != "" {
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    i + 1,
						Column:  1,
						Message: "Fenced code blocks should be surrounded by blank lines",
					})
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
				// Check blank line after (not required if at document end)
				if i < n-1 && strings.TrimSpace(lines[i+1]) != "" {
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    i + 1,
						Column:  1,
						Message: "Fenced code blocks should be surrounded by blank lines",
					})
				}
			}
		}
	}
	return violations
}
