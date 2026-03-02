package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD032 checks that lists are surrounded by blank lines.
type MD032 struct{}

func (r MD032) ID() string          { return "MD032" }
func (r MD032) Aliases() []string   { return []string{"blanks-around-lists"} }
func (r MD032) Description() string { return "Lists should be surrounded by blank lines" }

// listItemRE matches unordered or ordered list item lines.
var listItemRE = regexp.MustCompile(`^( *)(?:[-*+]|\d+\.) `)

func isListItemLine(line string) bool {
	return listItemRE.MatchString(line)
}

// listItemFirstLine returns the 1-based source line number of the first content
// line of the given list item. The direct children of a ListItem are always
// block-level nodes (TextBlock, Paragraph, nested List, etc.) so it is safe to
// call Lines() on them.
func listItemFirstLine(item *ast.ListItem, source []byte) int {
	child := item.FirstChild()
	if child == nil {
		return 0
	}
	if child.Lines() != nil && child.Lines().Len() > 0 {
		return countLine(source, child.Lines().At(0).Start)
	}
	return 0
}

// md032LeadingSpaces counts the number of leading space/tab characters in line.
// Tabs are counted as a single unit (not expanded).
func md032LeadingSpaces(line string) int {
	for i, c := range line {
		if c != ' ' && c != '\t' {
			return i
		}
	}
	return len(line)
}

func (r MD032) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	n := len(lines)
	mask := fencedCodeBlockMask(lines)

	// Identify list regions: maximal spans of list-item lines and their
	// continuation lines (indented non-blank lines that follow a list item).
	type region struct{ start, end int }
	var regions []region
	inList := false
	regionStart, regionEnd := 0, 0

	for i, line := range lines {
		if mask[i] {
			if inList {
				regions = append(regions, region{regionStart, regionEnd})
				inList = false
			}
			continue
		}
		if isListItemLine(line) {
			if !inList {
				inList = true
				regionStart = i
			}
			regionEnd = i
		} else if inList {
			if strings.TrimSpace(line) != "" && md032LeadingSpaces(line) > 0 {
				// Indented non-blank line: continuation of the current list item.
				regionEnd = i
			} else {
				regions = append(regions, region{regionStart, regionEnd})
				inList = false
			}
		}
	}
	if inList {
		regions = append(regions, region{regionStart, regionEnd})
	}

	// Rebuild the output, inserting blank lines around each region as needed.
	var result []string
	prev := 0
	for _, reg := range regions {
		result = append(result, lines[prev:reg.start]...)
		if reg.start > 0 && len(result) > 0 && strings.TrimSpace(result[len(result)-1]) != "" {
			result = append(result, "")
		}
		result = append(result, lines[reg.start:reg.end+1]...)
		if reg.end+1 < n && strings.TrimSpace(lines[reg.end+1]) != "" {
			result = append(result, "")
		}
		prev = reg.end + 1
	}
	result = append(result, lines[prev:]...)
	return []byte(strings.Join(result, "\n"))
}

func (r MD032) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	lines := doc.Lines
	n := len(lines)

	_ = ast.Walk(doc.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		list, ok := node.(*ast.List)
		if !ok {
			return ast.WalkContinue, nil
		}
		// Skip nested lists (whose parent is a list item): blank-line rules
		// apply only to the outermost list in each context.
		if _, parentIsItem := list.Parent().(*ast.ListItem); parentIsItem {
			return ast.WalkContinue, nil
		}
		// Skip lists inside blockquotes: the blank-line check operates on raw
		// source lines and does not understand blockquote context. Markdownlint
		// does not enforce blank lines around lists that are inside blockquotes.
		for p := list.Parent(); p != nil; p = p.Parent() {
			if _, inBQ := p.(*ast.Blockquote); inBQ {
				return ast.WalkContinue, nil
			}
		}

		firstItem, _ := list.FirstChild().(*ast.ListItem)
		if firstItem == nil {
			return ast.WalkContinue, nil
		}
		lastItem, _ := list.LastChild().(*ast.ListItem)
		if lastItem == nil {
			return ast.WalkContinue, nil
		}

		firstLine := listItemFirstLine(firstItem, doc.Source)
		if firstLine <= 0 {
			return ast.WalkContinue, nil
		}
		firstLineIdx := firstLine - 1 // 0-based

		lastItemLine := listItemFirstLine(lastItem, doc.Source)
		if lastItemLine <= 0 {
			// Cannot determine the last item's position; skip this list.
			return ast.WalkContinue, nil
		}
		lastItemLineIdx := lastItemLine - 1 // 0-based

		// Before check: the line immediately preceding the first list item must
		// be blank (or the list must be at the start of the document).
		beforeViolation := -1
		if firstLineIdx > 0 && strings.TrimSpace(lines[firstLineIdx-1]) != "" {
			beforeViolation = firstLine
		}

		// After check: scan source lines that follow the last list marker.
		// ast.ListItem.Offset is the column offset of the item's content after
		// the marker (e.g. 2 for "- item", 3 for "1. item"). Lines with at
		// least that many leading spaces are continuations of the item and are
		// skipped. The first non-blank, non-continuation line triggers a
		// violation.
		afterViolation := -1
		offset := lastItem.Offset
		for i := lastItemLineIdx + 1; i < n; i++ {
			line := lines[i]
			if strings.TrimSpace(line) == "" {
				break
			}
			if md032LeadingSpaces(line) >= offset {
				continue // continuation of the last list item
			}
			afterViolation = lastItemLine
			break
		}

		if beforeViolation > 0 {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    beforeViolation,
				Column:  1,
				Message: "Lists should be surrounded by blank lines",
			})
		}
		// Avoid double-reporting on the same line (e.g. a single-item list
		// that is missing blank lines both before and after it).
		if afterViolation > 0 && afterViolation != beforeViolation {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    afterViolation,
				Column:  1,
				Message: "Lists should be surrounded by blank lines",
			})
		}

		return ast.WalkContinue, nil
	})

	return violations
}
