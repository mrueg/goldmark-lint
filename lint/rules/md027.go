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
var md027RE = regexp.MustCompile(`^( {0,3}>+) {2,}`)

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
		if md027RE.MatchString(line) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  1,
				Message: "Multiple spaces after blockquote symbol",
			})
		}
	}
	return violations
}

func (r MD027) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if !mask[i] {
			lines[i] = md027RE.ReplaceAllString(line, "$1 ")
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
