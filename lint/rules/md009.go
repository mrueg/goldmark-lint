package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD009 checks for trailing spaces at the end of lines.
type MD009 struct {
	// BrSpaces is the number of spaces allowed at end of line for hard line breaks (default 2).
	BrSpaces int `json:"br_spaces"`
	// CodeBlocks controls whether trailing spaces inside code blocks are checked.
	// It defaults to false: like markdownlint, trailing whitespace inside a code
	// block is left alone because it can be significant.
	CodeBlocks *bool `json:"code_blocks"`
	// ListItemEmptyLines controls whether trailing spaces are allowed on empty lines
	// within list items (default false).
	ListItemEmptyLines bool `json:"list_item_empty_lines"`
	// Strict disallows all trailing spaces, including the br_spaces hard-break spaces (default false).
	Strict bool `json:"strict"`
}

func (r MD009) ID() string          { return "MD009" }
func (r MD009) Aliases() []string   { return []string{"no-trailing-spaces"} }
func (r MD009) Description() string { return "Trailing spaces" }

func (r MD009) Fix(source []byte) []byte {
	brSpaces := r.BrSpaces
	if brSpaces == 0 {
		brSpaces = 2
	}
	lines := strings.Split(string(source), "\n")
	codeMask := fencedCodeBlockMask(lines)
	checkCodeBlocks := r.CodeBlocks != nil && *r.CodeBlocks
	for i, line := range lines {
		if !checkCodeBlocks && codeMask[i] {
			continue
		}
		trimmed := strings.TrimRight(line, " \t")
		trailingLen := len(line) - len(trimmed)
		if trailingLen > 0 {
			if !r.Strict && trailingLen == brSpaces && strings.HasSuffix(line, strings.Repeat(" ", brSpaces)) {
				continue
			}
			lines[i] = trimmed
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD009) Check(doc *lint.Document) []lint.Violation {
	brSpaces := r.BrSpaces
	if brSpaces == 0 {
		brSpaces = 2
	}
	checkCodeBlocks := r.CodeBlocks != nil && *r.CodeBlocks
	codeMask := fencedCodeBlockMask(doc.Lines)
	if !checkCodeBlocks {
		// Also mark indented and fenced code block lines via the AST.
		// The raw-line fencedCodeBlockMask misses fenced code blocks inside
		// blockquotes (where each line is prefixed with "> "); the AST-based
		// walk handles those correctly.
		// indentedBase records, for each line of an indented code block, the
		// indentation the block itself starts at. Fenced blocks are left at -1
		// because only an indented block can absorb a following blank line.
		indentedBase := make([]int, len(doc.Lines))
		for i := range indentedBase {
			indentedBase[i] = -1
		}
		markBlockLines := func(n ast.Node) {
			var cb *ast.BaseBlock
			indented := false
			switch node := n.(type) {
			case *ast.CodeBlock:
				cb = &node.BaseBlock
				indented = true
			case *ast.FencedCodeBlock:
				cb = &node.BaseBlock
			default:
				return
			}
			if cb.Lines() == nil {
				return
			}
			base := -1
			for i := 0; i < cb.Lines().Len(); i++ {
				seg := cb.Lines().At(i)
				lineNum := doc.LineAt(seg.Start) - 1
				if lineNum < 0 || lineNum >= len(codeMask) {
					continue
				}
				codeMask[lineNum] = true
				if !indented {
					continue
				}
				// The block's own indentation is that of its first line; later
				// lines may be indented further and must not raise the bar.
				if base < 0 {
					base = whitespaceWidth(doc.Lines[lineNum])
				}
				indentedBase[lineNum] = base
			}
		}
		_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			markBlockLines(n)
			return ast.WalkContinue, nil
		})

		// A whitespace-only line that trails an indented code block belongs to
		// that block when it reaches the block's indentation, and is ordinary
		// blank content when it falls short of it. markdownlint draws the line
		// in the same place:
		//
		//	    code at indent 4
		//	<4 spaces>              part of the block, not reported
		//	<3 spaces>              blank content, reported
		//
		// A line after a *fenced* block is never absorbed, which is why only
		// indented blocks seed this.
		lastIndent := -1
		for i, line := range doc.Lines {
			if strings.TrimSpace(line) != "" {
				lastIndent = indentedBase[i]
				continue
			}
			if codeMask[i] || lastIndent < 0 {
				continue
			}
			if whitespaceWidth(line) >= lastIndent {
				codeMask[i] = true
			}
		}
	}

	var violations []lint.Violation
	for i, line := range doc.Lines {
		if !checkCodeBlocks && codeMask[i] {
			continue
		}
		trimmed := strings.TrimRight(line, " \t")
		trailingLen := len(line) - len(trimmed)
		if trailingLen > 0 {
			if !r.Strict && trailingLen == brSpaces && strings.HasSuffix(line, strings.Repeat(" ", brSpaces)) {
				// Allow br_spaces hard line breaks unless strict mode.
				if !r.ListItemEmptyLines || strings.TrimSpace(trimmed) != "" {
					continue
				}
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  len(trimmed) + 1,
				Message: fmt.Sprintf("Trailing spaces [Expected: 0 or %d; Actual: %d]", brSpaces, trailingLen),
			})
		}
	}
	return violations
}

// whitespaceWidth returns the column width of line's leading whitespace,
// expanding tabs to the next four-column stop as CommonMark does.
func whitespaceWidth(line string) int {
	col := 0
	for _, c := range line {
		switch c {
		case ' ':
			col++
		case '\t':
			col += 4 - (col % 4)
		default:
			return col
		}
	}
	return col
}
