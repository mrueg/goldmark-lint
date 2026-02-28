package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD030 checks that list markers are followed by a single space.
type MD030 struct {
	ULSingle int `json:"ul_single"`
	OLSingle int `json:"ol_single"`
	// ULMulti is the required spaces after UL markers for multi-line list items (default: same as ULSingle).
	ULMulti int `json:"ul_multi"`
	// OLMulti is the required spaces after OL markers for multi-line list items (default: same as OLSingle).
	OLMulti int `json:"ol_multi"`
}

func (r MD030) ID() string          { return "MD030" }
func (r MD030) Aliases() []string   { return []string{"list-marker-space"} }
func (r MD030) Description() string { return "Spaces after list markers" }

// md030FullRE captures marker type and the spaces following it.
var md030FullRE = regexp.MustCompile(`^( *)([-*+]|\d+[.)])( +)`)

func (r MD030) ulSpaces() int {
	if r.ULSingle <= 0 {
		return 1
	}
	return r.ULSingle
}

func (r MD030) olSpaces() int {
	if r.OLSingle <= 0 {
		return 1
	}
	return r.OLSingle
}

func (r MD030) ulMultiSpaces() int {
	if r.ULMulti <= 0 {
		return r.ulSpaces()
	}
	return r.ULMulti
}

func (r MD030) olMultiSpaces() int {
	if r.OLMulti <= 0 {
		return r.olSpaces()
	}
	return r.OLMulti
}

// isMultiLineListItem reports whether the list item at index i is a multi-line item.
// A list item is multi-line if the next non-blank continuation line is present
// (i.e., the line after it is blank or there is an indented continuation).
func isMultiLineListItem(lines []string, i int) bool {
	if i+1 >= len(lines) {
		return false
	}
	next := lines[i+1]
	// Multi-line if next line is blank (empty item with continuation) or
	// indented (continuation of this item).
	return strings.TrimSpace(next) == "" || strings.HasPrefix(next, "  ")
}

func (r MD030) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	ulSpaces := r.ulSpaces()
	olSpaces := r.olSpaces()
	ulMultiSpaces := r.ulMultiSpaces()
	olMultiSpaces := r.olMultiSpaces()

	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		m := md030FullRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		marker := m[2]
		spaces := m[3]
		isOrdered := len(marker) > 1 || (marker[0] >= '0' && marker[0] <= '9')
		multi := isMultiLineListItem(doc.Lines, i)
		var expected int
		if isOrdered {
			if multi {
				expected = olMultiSpaces
			} else {
				expected = olSpaces
			}
		} else {
			if multi {
				expected = ulMultiSpaces
			} else {
				expected = ulSpaces
			}
		}
		if len(spaces) != expected {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  len(m[1]) + len(marker) + 1,
				Message: fmt.Sprintf("Spaces after list markers [Expected: %d; Actual: %d]", expected, len(spaces)),
			})
		}
	}
	return violations
}

func (r MD030) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	ulSpaces := r.ulSpaces()
	olSpaces := r.olSpaces()
	ulMultiSpaces := r.ulMultiSpaces()
	olMultiSpaces := r.olMultiSpaces()

	for i, line := range lines {
		if mask[i] {
			continue
		}
		m := md030FullRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		indent := m[1]
		marker := m[2]
		spaces := m[3]
		isOrdered := len(marker) > 1 || (marker[0] >= '0' && marker[0] <= '9')
		multi := isMultiLineListItem(lines, i)
		var expected int
		if isOrdered {
			if multi {
				expected = olMultiSpaces
			} else {
				expected = olSpaces
			}
		} else {
			if multi {
				expected = ulMultiSpaces
			} else {
				expected = ulSpaces
			}
		}
		if len(spaces) != expected {
			rest := line[len(indent)+len(marker)+len(spaces):]
			lines[i] = indent + marker + strings.Repeat(" ", expected) + rest
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
