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

// md032HTMLCommentRE matches HTML comments (used for isBlankLikeForMD032).
var md032HTMLCommentRE = regexp.MustCompile(`<!--.*?-->`)

func isListItemLine(line string) bool {
	return listItemRE.MatchString(line)
}

// isBlankLikeForMD032 returns true if a line is "blank" for MD032 purposes.
// This matches markdownlint's isBlankLine() which also treats lines consisting
// only of HTML comments and '>' characters (blockquote markers) as blank.
func isBlankLikeForMD032(line string) bool {
	if strings.TrimSpace(line) == "" {
		return true
	}
	// Remove HTML comments, then remove '>' characters.
	// If nothing meaningful remains, treat as blank.
	cleaned := md032HTMLCommentRE.ReplaceAllString(line, "")
	cleaned = strings.ReplaceAll(cleaned, ">", "")
	return strings.TrimSpace(cleaned) == ""
}

// isBlockLevelBreaker returns true if the line starts a markdown block element
// that cannot be lazily continued as part of a list item paragraph. Plain text
// that follows a list item without a blank line is treated as a lazy
// continuation in CommonMark, so it does not produce an "after" violation.
func isBlockLevelBreaker(line string) bool {
	if len(line) == 0 {
		return false
	}
	switch line[0] {
	case '#':
		return true // ATX heading
	case '>':
		return true // block quote
	case '<':
		return true // HTML block
	case '|':
		return true // GFM table row
	}
	// Fenced code block
	if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
		return true
	}
	// Thematic break or setext heading underline (---, ===, ***, ___)
	trimmed := strings.TrimSpace(line)
	if len(trimmed) >= 3 {
		r := rune(trimmed[0])
		if r == '-' || r == '=' || r == '*' || r == '_' {
			allSame := true
			for _, c := range trimmed {
				if c != r {
					allSame = false
					break
				}
			}
			if allSame {
				return true
			}
		}
	}
	// Link reference definition: [label]: url
	if md052DefRE.MatchString(line) {
		return true
	}
	// List item marker
	if listItemRE.MatchString(line) {
		return true
	}
	return false
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
		// Use isBlankLikeForMD032 which treats HTML comment-only lines and
		// blockquote-marker-only lines as blank (matching markdownlint behavior).
		beforeViolation := -1
		if firstLineIdx > 0 && !isBlankLikeForMD032(lines[firstLineIdx-1]) {
			beforeViolation = firstLine
		}

		// After check: scan source lines that follow the last list marker.
		// In CommonMark, plain text can be "lazily continued" into the last list
		// item paragraph, so it does not produce a violation. Only block-level
		// elements that cannot be lazily continued (headings, blockquotes, code
		// fences, thematic breaks, list markers, tables, HTML blocks) trigger
		// an after violation.
		afterViolation := -1
		offset := lastItem.Offset
		for i := lastItemLineIdx + 1; i < n; i++ {
			line := lines[i]
			if strings.TrimSpace(line) == "" || isBlankLikeForMD032(line) {
				break
			}
			if md032LeadingSpaces(line) >= offset {
				continue // continuation/indented content of the last list item
			}
			// Only flag if this line is a block-level marker (cannot be a lazy
			// continuation of a paragraph inside the last list item).
			if !isBlockLevelBreaker(line) {
				break
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
