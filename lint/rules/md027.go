package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
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

// md027FencedCodeMask returns a bool mask marking lines that are inside fenced
// code blocks, including fenced code blocks inside blockquotes. It strips any
// leading blockquote markers ("> ") before checking for fence delimiters.
func md027FencedCodeMask(lines []string) []bool {
	mask := make([]bool, len(lines))
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	for i, line := range lines {
		// Strip up to one level of blockquote prefix so that fenced blocks
		// inside blockquotes are correctly detected.
		stripped := line
		for {
			s := strings.TrimLeft(stripped, " ")
			if len(s) > 0 && s[0] == '>' {
				s = s[1:]
				if len(s) > 0 && s[0] == ' ' {
					s = s[1:]
				}
				stripped = s
			} else {
				break
			}
		}
		// Strip up to 3 leading spaces (CommonMark fence indentation).
		indent := 0
		for indent < len(stripped) && indent < 3 && stripped[indent] == ' ' {
			indent++
		}
		trimmed := stripped[indent:]
		if !inFence {
			if len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
				fc := trimmed[0]
				j := 0
				for j < len(trimmed) && trimmed[j] == fc {
					j++
				}
				if j >= 3 {
					inFence = true
					fenceChar = fc
					fenceLen = j
				}
			}
			mask[i] = false
		} else {
			if len(trimmed) >= fenceLen && trimmed[0] == fenceChar {
				j := 0
				for j < len(trimmed) && trimmed[j] == fenceChar {
					j++
				}
				if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" {
					inFence = false
					mask[i] = false
					continue
				}
			}
			mask[i] = true
		}
	}
	return mask
}


// optional leading spaces (0–3), a '>' character, and a capturing group of trailing spaces.
var md027BQLevelRE = regexp.MustCompile(`^( {0,3}>)( *)`)

// md027ListItemRE matches a list item continuation prefix (spaces before blockquote).
var md027ListItemRE = regexp.MustCompile(`^ {2,}`)

// md027ViolationLine checks whether a line has multiple spaces after any blockquote
// marker at any nesting level. It returns (violated bool, prefix string up to the
// offending spaces, extraSpaces string).
//
// An optional ordered/unordered list-item prefix is stripped first so that lines
// like "9. >  text" (blockquote inside a list item) are checked correctly.
func md027ViolationLine(line string) (violated bool, before, spaces string) {
	rest := line
	consumed := 0

	// Strip an optional list-item prefix at the very start of the line so
	// that blockquotes embedded inside list items (e.g. "9. >  text") are checked.
	if m := orderedItemRE.FindStringSubmatch(rest); m != nil {
		consumed += len(m[0])
		rest = rest[len(m[0]):]
	} else if len(rest) >= 2 && (rest[0] == '-' || rest[0] == '*' || rest[0] == '+') && rest[1] == ' ' {
		consumed += 2
		rest = rest[2:]
	}

	for {
		m := md027BQLevelRE.FindStringSubmatch(rest)
		if m == nil {
			return false, "", ""
		}
		marker := m[1] // e.g. ">" or "   >"
		sp := m[2]     // spaces immediately after ">"
		advance := len(marker) + len(sp)
		if len(sp) >= 2 {
			return true, line[:consumed+len(marker)], sp
		}
		// 0 or 1 space: no violation at this level; advance and check next level.
		consumed += advance
		rest = rest[advance:]
	}
}

// md027ListInBQMask returns a boolean mask marking lines whose content is inside
// a list that is itself inside a blockquote. Those lines should not be flagged by
// MD027 because the extra spaces after ">" are list-indentation, not a style error.
//
// Only lines where the BLOCKQUOTE is the OUTER container (blockquote contains
// the list, not list contains the blockquote) are masked. This preserves
// violations like "1. >  text" where a list item contains a blockquote.
func md027ListInBQMask(doc *lint.Document) []bool {
	mask := make([]bool, len(doc.Lines))
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		// Only process block-level nodes that carry line information.
		// Calling Lines() on inline nodes panics in goldmark.
		switch n.(type) {
		case *ast.Paragraph, *ast.TextBlock, *ast.Heading,
			*ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock:
		default:
			return ast.WalkContinue, nil
		}
		if n.Lines() == nil || n.Lines().Len() == 0 {
			return ast.WalkContinue, nil
		}
		// Walk up ancestors: check that a List is found before (more immediate
		// than) a Blockquote. This means the structure is BQ → … → List → … →
		// node, i.e. the blockquote is the outer container. When a List contains
		// a Blockquote (List → … → BQ → … → node), the BQ would be found first
		// and we must NOT mask (those are legitimate blockquote violations).
		inList := false
		inBQ := false
		listFoundBeforeBQ := false
		for p := n.Parent(); p != nil; p = p.Parent() {
			if _, ok := p.(*ast.List); ok {
				if !inList {
					inList = true
					if !inBQ {
						listFoundBeforeBQ = true
					}
				}
			}
			if _, ok := p.(*ast.Blockquote); ok {
				inBQ = true
			}
		}
		// Mask only when Blockquote is the outer container (List found first).
		if inList && inBQ && listFoundBeforeBQ {
			for i := 0; i < n.Lines().Len(); i++ {
				seg := n.Lines().At(i)
				lineNum := countLine(doc.Source, seg.Start) - 1
				if lineNum >= 0 && lineNum < len(mask) {
					mask[lineNum] = true
				}
			}
		}
		return ast.WalkContinue, nil
	})
	return mask
}

func (r MD027) Check(doc *lint.Document) []lint.Violation {
	checkListItems := r.ListItems == nil || *r.ListItems
	var violations []lint.Violation
	mask := md027FencedCodeMask(doc.Lines)
	listInBQ := md027ListInBQMask(doc)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if !checkListItems && md027ListItemRE.MatchString(line) {
			// Line is indented (likely a list item); skip.
			continue
		}
		// Skip lines that are list-item content inside a blockquote: their extra
		// spaces after ">" are list indentation, not a multiple-space violation.
		if listInBQ[i] {
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
