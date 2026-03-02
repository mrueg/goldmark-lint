package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD027 checks for multiple spaces after blockquote symbols.
type MD027 struct {
	// ListItems controls whether the rule is applied to blockquotes within
	// list items (default true). Set to false to disable for list items.
	ListItems *bool `json:"list_items"`
}

func (r MD027) ID() string          { return "MD027" }
func (r MD027) Aliases() []string   { return []string{"no-multiple-space-blockquote"} }
func (r MD027) Description() string { return "Multiple spaces after blockquote symbol" }

// md027RE matches a blockquote line where 2+ spaces follow the '>' markers.
// Group 1: optional leading spaces + one or more '>' characters.
// Group 2: the extra spaces (2+). We use this to exclude code-block-level
// indentation (4+ extra spaces after the required one = 5+ total).
var md027RE = regexp.MustCompile(`^( {0,3}>+)( {2,})`)

// md027ListItemContentRE matches a list item at the start of a string.
var md027ListItemContentRE = regexp.MustCompile(`^(?:[-*+]|\d+[.)]) `)

// md027ListItemRE matches a list item continuation prefix (spaces before blockquote).
var md027ListItemRE = regexp.MustCompile(`^ {2,}`)

func (r MD027) Check(doc *lint.Document) []lint.Violation {
	checkListItems := r.ListItems == nil || *r.ListItems
	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if !checkListItems && md027ListItemRE.MatchString(line) {
			// Line is indented (likely a list item); skip.
			continue
		}
		m := md027RE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		spaces := m[2]
		// 5+ spaces after > = indented code block inside blockquote; skip.
		if len(spaces) >= 5 {
			continue
		}
		// If the content after the spaces is a list item marker, skip.
		// Extra spaces are part of the list indentation, not a MD027 violation.
		rest := line[len(m[1])+len(spaces):]
		if md027ListItemContentRE.MatchString(rest) {
			continue
		}
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    i + 1,
			Column:  1,
			Message: "Multiple spaces after blockquote symbol",
		})
	}
	return violations
}

func (r MD027) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if !mask[i] {
			m := md027RE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			spaces := m[2]
			// 5+ spaces = code block level; skip.
			if len(spaces) >= 5 {
				continue
			}
			// Skip if content after spaces is a list item.
			rest := line[len(m[1])+len(spaces):]
			if md027ListItemContentRE.MatchString(rest) {
				continue
			}
			lines[i] = m[1] + " " + rest
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
