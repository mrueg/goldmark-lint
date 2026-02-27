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
}

func (r MD030) ID() string          { return "MD030" }
func (r MD030) Description() string { return "Spaces after list markers" }

// md030RE matches a list item line capturing: indent, marker, spaces after marker.
var md030RE = regexp.MustCompile(`^( *)(?:([-*+])|(\d+)[.)]) +`)

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

func (r MD030) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	ulSpaces := r.ulSpaces()
	olSpaces := r.olSpaces()

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
		expected := ulSpaces
		if isOrdered {
			expected = olSpaces
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
		expected := ulSpaces
		if isOrdered {
			expected = olSpaces
		}
		if len(spaces) != expected {
			rest := line[len(indent)+len(marker)+len(spaces):]
			lines[i] = indent + marker + strings.Repeat(" ", expected) + rest
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
