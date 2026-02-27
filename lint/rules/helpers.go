package rules

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// countLine counts the 1-based line number of byte offset pos in source.
func countLine(source []byte, pos int) int {
	line := 1
	for i := 0; i < pos && i < len(source); i++ {
		if source[i] == '\n' {
			line++
		}
	}
	return line
}

// headingText returns the text content of a heading node.
func headingText(n ast.Node, source []byte) string {
	var text []byte
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			seg := t.Segment
			text = append(text, source[seg.Start:seg.Stop]...)
		}
	}
	return string(text)
}

// fencedCodeBlockMask returns a bool slice with true for each line that is
// inside (not on the fence delimiters of) a fenced code block.
func fencedCodeBlockMask(lines []string) []bool {
	mask := make([]bool, len(lines))
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if !inFence {
			if len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
				fc := trimmed[0]
				j := 0
				for j < len(trimmed) && trimmed[j] == fc {
					j++
				}
				if j >= 3 {
					inFence = true
					fenceChar = fc
					fenceLen = j
				}
			}
			mask[i] = false
		} else {
			if len(trimmed) >= fenceLen && trimmed[0] == fenceChar {
				j := 0
				for j < len(trimmed) && trimmed[j] == fenceChar {
					j++
				}
				if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" {
					inFence = false
					mask[i] = false
					continue
				}
			}
			mask[i] = true
		}
	}
	return mask
}
