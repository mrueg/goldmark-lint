package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD004 checks that unordered list markers are consistent.
type MD004 struct {
	// Style is the required marker style: "consistent" (default), "asterisk",
	// "plus", "dash", or "sublist" (different symbol per nesting level).
	Style string `json:"style"`
}

func (r MD004) ID() string          { return "MD004" }
func (r MD004) Aliases() []string   { return []string{"ul-style"} }
func (r MD004) Description() string { return "Unordered list style" }

// Fix applies MD004 to source by replacing unordered list markers with the required character.
func (r MD004) Fix(source []byte) []byte {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)

	// Determine the first marker (for "consistent" style).
	firstMarker := byte(0)
	if style == "consistent" {
		for i, line := range lines {
			if mask[i] {
				continue
			}
			if m := unorderedListMarkerAt(line); m != 0 {
				firstMarker = m
				break
			}
		}
	}

	changed := false
	// For "sublist", track nesting depth.
	sublistMarkers := []byte{'-', '*', '+'}
	depth := 0
	// depthStack tracks the indent levels to determine nesting.
	type indentEntry struct{ indent int }
	var depthStack []indentEntry

	for i, line := range lines {
		if mask[i] {
			depthStack = nil
			continue
		}
		m := unorderedListMarkerAt(line)
		if m == 0 {
			if strings.TrimSpace(line) == "" {
				// blank line resets sublist depth tracking
				depthStack = nil
			}
			continue
		}

		var expected byte
		switch style {
		case "asterisk":
			expected = '*'
		case "plus":
			expected = '+'
		case "dash":
			expected = '-'
		case "consistent":
			if firstMarker == 0 {
				firstMarker = m
			}
			expected = firstMarker
		case "sublist":
			// Determine nesting level by indent.
			indent := 0
			for indent < len(line) && line[indent] == ' ' {
				indent++
			}
			// Pop stack entries deeper than current indent.
			for len(depthStack) > 0 && depthStack[len(depthStack)-1].indent >= indent {
				depthStack = depthStack[:len(depthStack)-1]
			}
			depth = len(depthStack)
			depthStack = append(depthStack, indentEntry{indent})
			expected = sublistMarkers[depth%len(sublistMarkers)]
		}

		if expected != 0 && m != expected {
			// Replace the marker character on this line.
			idx := strings.IndexByte(line, m)
			if idx >= 0 {
				bs := []byte(line)
				bs[idx] = expected
				lines[i] = string(bs)
				changed = true
			}
		}
	}
	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}

// unorderedListMarkerAt returns the unordered list marker byte for line if it
// is a list item line (e.g. "- item", "* item", "+ item"), otherwise 0.
func unorderedListMarkerAt(line string) byte {
	i := 0
	// Allow up to 3 leading spaces.
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) {
		return 0
	}
	m := line[i]
	if m != '-' && m != '*' && m != '+' {
		return 0
	}
	// Must be followed by a space.
	if i+1 >= len(line) || line[i+1] != ' ' {
		return 0
	}
	return m
}

func (r MD004) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	var violations []lint.Violation
	firstMarker := byte(0)

	// For "sublist" style, each nesting level uses a different marker.
	// The cycle is: dash (level 1), asterisk (level 2), plus (level 3), repeat.
	sublistMarkers := []byte{'-', '*', '+'}
	depth := 0

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		list, ok := n.(*ast.List)
		if !ok || list.IsOrdered() {
			if _, isListItem := n.(*ast.ListItem); !isListItem {
				return ast.WalkContinue, nil
			}
			return ast.WalkContinue, nil
		}

		if style == "sublist" {
			if entering {
				expectedMarker := sublistMarkers[depth%len(sublistMarkers)]
				depth++
				if list.Marker != expectedMarker {
					for child := list.FirstChild(); child != nil; child = child.NextSibling() {
						li, ok2 := child.(*ast.ListItem)
						if !ok2 {
							continue
						}
						lineNum := 1
						if li.Lines() != nil && li.Lines().Len() > 0 {
							seg := li.Lines().At(0)
							lineNum = countLine(doc.Source, seg.Start)
						} else if fc := li.FirstChild(); fc != nil {
							if fc.Lines() != nil && fc.Lines().Len() > 0 {
								seg := fc.Lines().At(0)
								lineNum = countLine(doc.Source, seg.Start)
							}
						}
						violations = append(violations, lint.Violation{
							Rule:    r.ID(),
							Line:    lineNum,
							Column:  1,
							Message: fmt.Sprintf("Unordered list style [Expected: %c; Actual: %c]", expectedMarker, list.Marker),
						})
					}
				}
			} else {
				depth--
			}
			return ast.WalkContinue, nil
		}

		if !entering {
			return ast.WalkContinue, nil
		}

		marker := list.Marker

		expected := byte(0)
		switch style {
		case "asterisk":
			expected = '*'
		case "plus":
			expected = '+'
		case "dash":
			expected = '-'
		case "consistent":
			if firstMarker == 0 {
				firstMarker = marker
			}
			expected = firstMarker
		}

		if expected != 0 && marker != expected {
			// Instead of one violation for the whole list, report one per item.
			for child := list.FirstChild(); child != nil; child = child.NextSibling() {
				li, ok2 := child.(*ast.ListItem)
				if !ok2 {
					continue
				}
				lineNum := 1
				if li.Lines() != nil && li.Lines().Len() > 0 {
					seg := li.Lines().At(0)
					lineNum = countLine(doc.Source, seg.Start)
				} else if fc := li.FirstChild(); fc != nil {
					if fc.Lines() != nil && fc.Lines().Len() > 0 {
						seg := fc.Lines().At(0)
						lineNum = countLine(doc.Source, seg.Start)
					}
				}
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    lineNum,
					Column:  1,
					Message: fmt.Sprintf("Unordered list style [Expected: %c; Actual: %c]", expected, marker),
				})
			}
		}
		return ast.WalkContinue, nil
	})

	return violations
}
