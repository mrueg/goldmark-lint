package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD013 checks for lines that are too long.
type MD013 struct {
	// LineLength is the maximum line length (default 80).
	LineLength int `json:"line_length"`
	// HeadingLineLength is the maximum line length for headings (default: same as LineLength).
	HeadingLineLength int `json:"heading_line_length"`
	// CodeBlockLineLength is the maximum line length for code block lines (default: same as LineLength).
	CodeBlockLineLength int `json:"code_block_line_length"`
	// CodeBlocks controls whether code block lines are checked (default true).
	CodeBlocks *bool `json:"code_blocks"`
	// Tables controls whether table lines are checked (default true).
	Tables *bool `json:"tables"`
	// Headings controls whether heading lines are checked (default true).
	Headings *bool `json:"headings"`
}

func (r MD013) ID() string          { return "MD013" }
func (r MD013) Aliases() []string   { return []string{"line-length"} }
func (r MD013) Description() string { return "Line length" }

func (r MD013) Check(doc *lint.Document) []lint.Violation {
	defaultLimit := r.LineLength
	if defaultLimit == 0 {
		defaultLimit = 80
	}
	headingLimit := r.HeadingLineLength
	if headingLimit == 0 {
		headingLimit = defaultLimit
	}
	codeBlockLimit := r.CodeBlockLineLength
	if codeBlockLimit == 0 {
		codeBlockLimit = defaultLimit
	}

	checkCodeBlocks := r.CodeBlocks == nil || *r.CodeBlocks
	checkTables := r.Tables == nil || *r.Tables
	checkHeadings := r.Headings == nil || *r.Headings

	// Build fenced code block mask (lines inside a fenced block, not including fence delimiters).
	fenceMask := fencedCodeBlockMask(doc.Lines)
	// Also mark fence delimiter lines themselves as code block lines.
	codeBlockMask := make([]bool, len(doc.Lines))
	copy(codeBlockMask, fenceMask)
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	for i, line := range doc.Lines {
		trimmed := strings.TrimLeft(line, " ")
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
					codeBlockMask[i] = true
				}
			}
			// Also check indented code blocks (4 spaces or tab, after blank line).
			if !codeBlockMask[i] && isIndentedCodeLine(line) {
				if i > 0 && strings.TrimSpace(doc.Lines[i-1]) == "" {
					codeBlockMask[i] = true
				} else if i > 0 && codeBlockMask[i-1] && isIndentedCodeLine(doc.Lines[i-1]) {
					codeBlockMask[i] = true
				}
			}
		} else {
			j := 0
			for j < len(trimmed) && trimmed[j] == fenceChar {
				j++
			}
			if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" && len(trimmed) > 0 && trimmed[0] == fenceChar {
				inFence = false
				codeBlockMask[i] = true
			}
		}
	}

	// Build table mask.
	tableMask := make([]bool, len(doc.Lines))
	for _, tbl := range findTables(doc.Lines, codeBlockMask) {
		for i := tbl[0]; i <= tbl[1]; i++ {
			tableMask[i] = true
		}
	}

	// Build heading mask.
	headingMask := make([]bool, len(doc.Lines))
	n := len(doc.Lines)
	for i, line := range doc.Lines {
		if codeBlockMask[i] {
			continue
		}
		trimmed := strings.TrimLeft(line, " ")
		// ATX heading.
		if len(trimmed) > 0 && trimmed[0] == '#' {
			headingMask[i] = true
			continue
		}
		// Setext heading: non-blank line followed by ==== or ----.
		if i+1 < n && !codeBlockMask[i+1] {
			next := strings.TrimSpace(doc.Lines[i+1])
			if len(next) > 0 && (strings.Trim(next, "=") == "" || strings.Trim(next, "-") == "") {
				if strings.TrimSpace(line) != "" {
					headingMask[i] = true
					headingMask[i+1] = true
				}
			}
		}
	}

	var violations []lint.Violation
	for i, line := range doc.Lines {
		var limit int
		switch {
		case codeBlockMask[i]:
			if !checkCodeBlocks {
				continue
			}
			limit = codeBlockLimit
		case tableMask[i]:
			if !checkTables {
				continue
			}
			limit = defaultLimit
		case headingMask[i]:
			if !checkHeadings {
				continue
			}
			limit = headingLimit
		default:
			limit = defaultLimit
		}
		if len(line) > limit {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  limit + 1,
				Message: fmt.Sprintf("Line length [Expected: %d; Actual: %d]", limit, len(line)),
			})
		}
	}
	return violations
}
