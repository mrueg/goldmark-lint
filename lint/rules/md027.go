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

// md027BQLevelRE matches a single blockquote level at the start of a string:
// optional leading spaces (0–3), a '>' character, and a capturing group of trailing spaces.
var md027BQLevelRE = regexp.MustCompile(`^( {0,3}>)( *)`)

// md027ListItemRE matches a list item continuation prefix (spaces before blockquote).
var md027ListItemRE = regexp.MustCompile(`^ {2,}`)

// md027ViolationLine checks whether a line has multiple spaces after any blockquote
// marker at any nesting level. It returns (violated bool, prefix string up to the
// offending spaces, extraSpaces string). Indented-code-block levels (4+ extra spaces
// after the required one) are skipped.
func md027ViolationLine(line string) (violated bool, before, spaces string) {
	rest := line
	consumed := 0
	for {
		m := md027BQLevelRE.FindStringSubmatch(rest)
		if m == nil {
			return false, "", ""
		}
		marker := m[1]  // e.g. ">" or "   >"
		sp := m[2]      // spaces immediately after ">"
		advance := len(marker) + len(sp)
		if len(sp) >= 2 {
			// 5+ spaces after > = indented code block inside blockquote; skip.
			if len(sp) >= 5 {
				return false, "", ""
			}
			return true, line[:consumed+len(marker)], sp
		}
		// 0 or 1 space: no violation at this level; advance and check next level.
		consumed += advance
		rest = rest[advance:]
	}
}

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
		if violated, _, _ := md027ViolationLine(line); violated {
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
		if mask[i] {
			continue
		}
		violated, before, sp := md027ViolationLine(line)
		if !violated {
			continue
		}
		rest := line[len(before)+len(sp):]
		lines[i] = before + " " + rest
	}
	return []byte(strings.Join(lines, "\n"))
}
