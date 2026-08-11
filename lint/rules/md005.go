package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MD005 checks that list items at the same nesting level use consistent indentation.
type MD005 struct{}

func (r MD005) ID() string { return "MD005" }
func (r MD005) Aliases() []string {
	return []string{"list-indent"}
}
func (r MD005) Description() string {
	return "Inconsistent indentation for list items at the same level"
}

// md005IsListItem returns true when rest (the line after stripping leading spaces)
// starts with an unordered or ordered list marker followed by a space.
func md005IsListItem(rest string) bool {
	if len(rest) < 2 {
		return false
	}
	// Unordered
	if (rest[0] == '-' || rest[0] == '*' || rest[0] == '+') && rest[1] == ' ' {
		return true
	}
	// Ordered
	j := 0
	for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
		j++
	}
	if j > 0 && j < len(rest) && (rest[j] == '.' || rest[j] == ')') && j+1 < len(rest) && rest[j+1] == ' ' {
		return true
	}
	return false
}

// Fix normalises indentation for list items that are siblings but have
// inconsistent leading spaces.  The first item at each nesting level sets the
// canonical indentation; subsequent siblings that differ by at most one space
// are adjusted to match.
func (r MD005) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)

	// Stack-based tracking: each entry is the canonical indentation for the
	// list level it represents.
	type levelEntry struct{ canonical int }
	var stack []levelEntry
	changed := false

	for i, line := range lines {
		if mask[i] {
			continue
		}

		if strings.TrimSpace(line) == "" {
			stack = nil
			continue
		}

		spaces := 0
		for spaces < len(line) && line[spaces] == ' ' {
			spaces++
		}
		rest := line[spaces:]

		if !md005IsListItem(rest) {
			// Non-list content at indent 0 ends any list context.
			if spaces == 0 {
				stack = nil
			}
			continue
		}

		// Pop stack entries whose canonical indentation is greater than the
		// current item's indentation (we've gone back up the nesting tree).
		for len(stack) > 0 && stack[len(stack)-1].canonical > spaces {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			stack = append(stack, levelEntry{canonical: spaces})
			continue
		}

		top := stack[len(stack)-1].canonical
		if spaces == top {
			// Exact match — no fix needed.
			continue
		}
		// Items within 1 space of the canonical are considered siblings with
		// a minor indentation error and are adjusted to the canonical value.
		// We use AND so that both directions hold: |spaces - top| <= 1.
		if spaces-top <= 1 && top-spaces <= 1 {
			lines[i] = strings.Repeat(" ", top) + rest
			changed = true
			continue
		}
		// Larger difference — this item starts a new, deeper nesting level.
		stack = append(stack, levelEntry{canonical: spaces})
	}

	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD005) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		list, ok := n.(*ast.List)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Determine expected indent from first list item.
		expectedIndent := -1
		for child := list.FirstChild(); child != nil; child = child.NextSibling() {
			li, ok2 := child.(*ast.ListItem)
			if !ok2 {
				continue
			}
			seg, ok3 := listItemFirstSegment(li)
			if !ok3 {
				continue
			}
			lineIdx := doc.LineAt(seg.Start) - 1
			if lineIdx < 0 || lineIdx >= len(doc.Lines) {
				continue
			}
			line := doc.Lines[lineIdx]
			// Compare where the markers sit, not how many spaces precede them,
			// so that items introduced by a container prefix (a blockquote, or
			// an enclosing list marker on the same line) line up correctly.
			spaces := listMarkerColumn(line, columnOfOffset(doc.Source, seg.Start))
			if spaces < 0 {
				spaces = len(line) - len(strings.TrimLeft(line, " "))
			}
			if expectedIndent < 0 {
				expectedIndent = spaces
			} else if spaces != expectedIndent {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    lineIdx + 1,
					Column:  spaces + 1,
					Message: fmt.Sprintf("Inconsistent indentation for list items at the same level [Expected: %d; Actual: %d]", expectedIndent, spaces),
				})
			}
		}
		return ast.WalkContinue, nil
	})

	return violations
}

// listItemFirstSegment returns the first text segment of a list item, since
// ListItem.Lines() is empty in goldmark. A tight item holds a TextBlock and a
// loose one — any list with blank lines between its items — holds a Paragraph,
// so both have to be accepted; matching only TextBlock silently skipped every
// loose list.
func listItemFirstSegment(item *ast.ListItem) (text.Segment, bool) {
	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		switch c.(type) {
		case *ast.TextBlock, *ast.Paragraph:
			if c.Lines() != nil && c.Lines().Len() > 0 {
				return c.Lines().At(0), true
			}
		}
	}
	return text.Segment{}, false
}

// listMarkerColumn returns the column at which the list marker introducing
// contentCol sits on line, or -1 when no marker can be found.
//
// The marker's column, not the count of leading spaces, is what makes items
// comparable: an item can be introduced by a container prefix rather than by
// plain indentation. In a nested list written inside a blockquote inside an
// outer list item, the first inner item's line starts with the outer marker
// ("48. > 1. ...") while the next starts with spaces ("    > 2. ..."). Both
// inner markers sit at the same column, so the items are siblings; counting
// leading spaces would make them look inconsistent.
func listMarkerColumn(line string, contentCol int) int {
	j := contentCol - 1
	if j >= len(line) {
		j = len(line) - 1
	}
	// Step back over the whitespace between the marker and the content.
	for j >= 0 && (line[j] == ' ' || line[j] == '\t') {
		j--
	}
	if j < 0 {
		return -1
	}
	switch line[j] {
	case '-', '*', '+':
		return j
	case '.', ')':
		// Walk back over the digits of an ordered marker such as "48.".
		k := j - 1
		for k >= 0 && line[k] >= '0' && line[k] <= '9' {
			k--
		}
		if k == j-1 {
			return -1 // separator with no digits before it
		}
		return k + 1
	}
	return -1
}

// columnOfOffset returns the 0-based column of byte offset within its source
// line.
func columnOfOffset(source []byte, offset int) int {
	if offset > len(source) {
		offset = len(source)
	}
	start := offset
	for start > 0 && source[start-1] != '\n' {
		start--
	}
	return offset - start
}
