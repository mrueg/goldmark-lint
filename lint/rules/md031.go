package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD031 checks that fenced code blocks are surrounded by blank lines.
type MD031 struct {
	// ListItems controls whether the rule is applied to fenced code blocks
	// within list items (default true). Set to false to disable for list items.
	ListItems *bool `json:"list_items"`
}

func (r MD031) ID() string          { return "MD031" }
func (r MD031) Aliases() []string   { return []string{"blanks-around-fences"} }
func (r MD031) Description() string { return "Fenced code blocks should be surrounded by blank lines" }

// detectFence returns (isFence, fenceChar, fenceLen) for a line.
// Per CommonMark spec, a fenced code block may be indented by at most 3 spaces.
func detectFence(line string) (bool, byte, int) {
	// Count leading spaces (at most 3 allowed for a valid fence).
	indent := 0
	for indent < len(line) && line[indent] == ' ' {
		indent++
	}
	if indent > 3 {
		return false, 0, 0
	}
	trimmed := line[indent:]
	if len(trimmed) < 3 {
		return false, 0, 0
	}
	fc := trimmed[0]
	if fc != '`' && fc != '~' {
		return false, 0, 0
	}
	j := 0
	for j < len(trimmed) && trimmed[j] == fc {
		j++
	}
	if j >= 3 {
		return true, fc, j
	}
	return false, 0, 0
}

func (r MD031) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	var result []string
	n := len(lines)
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0

	for i, line := range lines {
		isFence, fc, fl := detectFence(line)
		if !inFence {
			if isFence {
				inFence = true
				fenceChar = fc
				fenceLen = fl
				// Insert blank line before if previous is non-blank
				if i > 0 && len(result) > 0 && strings.TrimSpace(result[len(result)-1]) != "" {
					result = append(result, "")
				}
				result = append(result, line)
			} else {
				result = append(result, line)
			}
		} else {
			// Check for closing fence
			trimmed := strings.TrimLeft(line, " ")
			j := 0
			for j < len(trimmed) && trimmed[j] == fenceChar {
				j++
			}
			if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" && trimmed[0] == fenceChar {
				inFence = false
				result = append(result, line)
				// Insert blank line after if next is non-blank
				if i < n-1 && strings.TrimSpace(lines[i+1]) != "" {
					result = append(result, "")
				}
			} else {
				result = append(result, line)
			}
		}
	}
	return []byte(strings.Join(result, "\n"))
}

func (r MD031) Check(doc *lint.Document) []lint.Violation {
	checkListItems := r.ListItems == nil || *r.ListItems
	var violations []lint.Violation
	lines := doc.Lines
	n := len(lines)

	_ = ast.Walk(doc.AST, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		cb, ok := node.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}

		// If list_items=false, skip fenced code blocks inside list items.
		if !checkListItems {
			for p := node.Parent(); p != nil; p = p.Parent() {
				if _, isLI := p.(*ast.ListItem); isLI {
					return ast.WalkContinue, nil
				}
			}
		}

		// Determine the opening fence line number (1-based).
		openLineNum := fencedCodeBlockLine(cb, doc.Source)
		if openLineNum <= 0 {
			return ast.WalkContinue, nil
		}
		openIdx := openLineNum - 1 // 0-based

		// Determine the closing fence line number.
		// The closing fence is on the line after the last content line.
		var closeIdx int
		if cb.Lines() != nil && cb.Lines().Len() > 0 {
			lastSeg := cb.Lines().At(cb.Lines().Len() - 1)
			// countLine counts newlines before pos; for the end of the last
			// content line (which includes the trailing newline), this gives
			// the line number of the closing fence.
			closeLineNum := countLine(doc.Source, lastSeg.Stop)
			closeIdx = closeLineNum - 1
		} else {
			// Empty code block: closing fence is immediately after opening fence.
			closeIdx = openIdx + 1
		}

		// Check blank line before opening fence (not required at document start).
		if openIdx > 0 && !isBlankOrBlockquoteBlank(lines[openIdx-1]) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    openLineNum,
				Column:  1,
				Message: "Fenced code blocks should be surrounded by blank lines",
			})
		}

		// Check blank line after closing fence (not required at document end).
		if closeIdx >= 0 && closeIdx < n-1 && !isBlankOrBlockquoteBlank(lines[closeIdx+1]) {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    closeIdx + 1,
				Column:  1,
				Message: "Fenced code blocks should be surrounded by blank lines",
			})
		}

		return ast.WalkContinue, nil
	})

	return violations
}
