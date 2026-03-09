package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD046 checks code block style consistency.
type MD046 struct {
	// Style is "consistent" (default), "fenced", or "indented".
	Style string `json:"style"`
}

func (r MD046) ID() string          { return "MD046" }
func (r MD046) Aliases() []string   { return []string{"code-block-style"} }
func (r MD046) Description() string { return "Code block style" }

// Fix applies MD046 to source by converting code blocks to the required style.
func (r MD046) Fix(source []byte) []byte {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	lines := strings.Split(string(source), "\n")

	// Determine effective style for "consistent".
	if style == "consistent" {
		inFence := false
		fenceChar := byte(0)
		fenceLen := 0
		for i, line := range lines {
			if !inFence {
				if isFence, fc, fl := detectFence(line); isFence {
					inFence = true
					fenceChar = fc
					fenceLen = fl
					style = "fenced"
					break
				}
			} else {
				if isFence, fc, fl := detectFence(line); isFence && fc == fenceChar && fl >= fenceLen {
					inFence = false
				}
			}
			// Check for indented code block (4+ spaces after blank line).
			if i > 0 && strings.TrimSpace(lines[i-1]) == "" {
				if len(line) >= 4 && line[0] == ' ' && line[1] == ' ' && line[2] == ' ' && line[3] == ' ' {
					style = "indented"
					break
				}
			}
		}
		if style == "consistent" {
			// No code blocks found, nothing to do.
			return source
		}
	}

	switch style {
	case "fenced":
		return md046ConvertToFenced(lines, source)
	case "indented":
		return md046ConvertToIndented(lines, source)
	}
	return source
}

// md046ConvertToFenced converts indented code blocks to fenced code blocks.
func md046ConvertToFenced(lines []string, source []byte) []byte {
	n := len(lines)

	// First, identify which lines are inside fenced code blocks (leave them alone).
	fencedSet := make([]bool, n)
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	for i, line := range lines {
		if !inFence {
			if isFence, fc, fl := detectFence(line); isFence {
				inFence = true
				fenceChar = fc
				fenceLen = fl
				fencedSet[i] = true
			}
		} else {
			fencedSet[i] = true
			if isFence, fc, fl := detectFence(line); isFence && fc == fenceChar && fl >= fenceLen {
				inFence = false
			}
		}
	}

	// Find indented code block regions: contiguous runs of 4+-space-indented lines
	// that are preceded by a blank line (or start of file).
	type region struct{ start, end int }
	var regions []region

	i := 0
	for i < n {
		if fencedSet[i] {
			i++
			continue
		}
		// An indented code block starts after a blank line (or at start of file).
		preceding := i == 0 || strings.TrimSpace(lines[i-1]) == ""
		if !preceding {
			i++
			continue
		}
		// Check if this line has 4+ spaces indent.
		if len(lines[i]) < 4 || lines[i][0] != ' ' || lines[i][1] != ' ' || lines[i][2] != ' ' || lines[i][3] != ' ' {
			i++
			continue
		}
		if fencedSet[i] {
			i++
			continue
		}
		// Found start of indented code block.
		start := i
		end := i
		for j := i + 1; j < n; j++ {
			if fencedSet[j] {
				break
			}
			if strings.TrimSpace(lines[j]) == "" {
				// Blank lines can be part of an indented code block if followed by more indented lines.
				// Look ahead.
				k := j + 1
				for k < n && strings.TrimSpace(lines[k]) == "" {
					k++
				}
				if k < n && !fencedSet[k] && len(lines[k]) >= 4 && lines[k][0] == ' ' && lines[k][1] == ' ' && lines[k][2] == ' ' && lines[k][3] == ' ' {
					end = k
					j = k
				} else {
					break
				}
			} else if len(lines[j]) >= 4 && lines[j][0] == ' ' && lines[j][1] == ' ' && lines[j][2] == ' ' && lines[j][3] == ' ' {
				end = j
			} else {
				break
			}
		}
		regions = append(regions, region{start, end})
		i = end + 1
	}

	if len(regions) == 0 {
		return source
	}

	// Build output, replacing indented blocks with fenced blocks.
	var result []string
	prev := 0
	for _, reg := range regions {
		result = append(result, lines[prev:reg.start]...)
		// Add opening fence.
		result = append(result, "```")
		// Add content lines with 4-space indent removed.
		for j := reg.start; j <= reg.end; j++ {
			line := lines[j]
			if len(line) >= 4 {
				result = append(result, line[4:])
			} else {
				result = append(result, strings.TrimLeft(line, " "))
			}
		}
		// Add closing fence.
		result = append(result, "```")
		prev = reg.end + 1
	}
	result = append(result, lines[prev:]...)
	return []byte(strings.Join(result, "\n"))
}

// md046ConvertToIndented converts fenced code blocks to indented code blocks.
func md046ConvertToIndented(lines []string, source []byte) []byte {
	// Find fenced code block regions.
	type region struct {
		openIdx  int
		closeIdx int
	}
	var regions []region
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	openIdx := 0

	for i, line := range lines {
		if !inFence {
			if isFence, fc, fl := detectFence(line); isFence {
				inFence = true
				fenceChar = fc
				fenceLen = fl
				openIdx = i
			}
		} else {
			if isFence, fc, fl := detectFence(line); isFence && fc == fenceChar && fl >= fenceLen {
				regions = append(regions, region{openIdx, i})
				inFence = false
			}
		}
	}

	if len(regions) == 0 {
		return source
	}

	// Build output replacing fenced blocks with indented blocks.
	var result []string
	prev := 0
	for _, reg := range regions {
		result = append(result, lines[prev:reg.openIdx]...)
		// Ensure blank line before the indented block.
		if reg.openIdx > 0 && len(result) > 0 && strings.TrimSpace(result[len(result)-1]) != "" {
			result = append(result, "")
		}
		// Add content lines with 4-space indent.
		for j := reg.openIdx + 1; j < reg.closeIdx; j++ {
			result = append(result, "    "+lines[j])
		}
		// Ensure blank line after the indented block.
		prev = reg.closeIdx + 1
		if prev < len(lines) && strings.TrimSpace(lines[prev]) != "" {
			result = append(result, "")
		}
	}
	result = append(result, lines[prev:]...)
	return []byte(strings.Join(result, "\n"))
}

func (r MD046) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	var violations []lint.Violation
	firstStyle := ""

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		var blockStyle string
		var lineNum int

		switch node := n.(type) {
		case *ast.FencedCodeBlock:
			blockStyle = "fenced"
			lineNum = fencedCodeBlockLine(node, doc.Source)
		case *ast.CodeBlock:
			blockStyle = "indented"
			if node.Lines() != nil && node.Lines().Len() > 0 {
				lineNum = countLine(doc.Source, node.Lines().At(0).Start)
			}
		default:
			return ast.WalkContinue, nil
		}

		if lineNum == 0 {
			return ast.WalkContinue, nil
		}

		expected := style
		if style == "consistent" {
			if firstStyle == "" {
				firstStyle = blockStyle
			}
			expected = firstStyle
		}

		if blockStyle != expected {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    lineNum,
				Column:  1,
				Message: fmt.Sprintf("Code block style [Expected: %s; Actual: %s]", expected, blockStyle),
			})
		}

		return ast.WalkContinue, nil
	})

	return violations
}
