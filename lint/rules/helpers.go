package rules

import "github.com/yuin/goldmark/ast"

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
