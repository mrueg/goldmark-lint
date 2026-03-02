package rules

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
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
	// Strict enforces line_length for all contexts, ignoring the separate
	// heading_line_length and code_block_line_length limits (default false).
	Strict bool `json:"strict"`
	// Stern disables URL exemption: when false (default), lines that exceed
	// the limit only due to a URL are not reported. When true, all lines are
	// checked by their full length including any URLs.
	Stern bool `json:"stern"`
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
	if r.Strict || headingLimit == 0 {
		headingLimit = defaultLimit
	}
	codeBlockLimit := r.CodeBlockLineLength
	if r.Strict || codeBlockLimit == 0 {
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

	// Build a per-line URL length map for the URL exemption (when not in stern mode).
	var lineURLLens map[int][]int
	if !r.Stern {
		lineURLLens = urlLengthsPerLine(doc)
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
		lineLen := utf8.RuneCountInString(line)
		if lineLen > limit {
			// URL exemption: skip lines that exceed the limit only due to a URL.
			// lineURLLens[i+1] returns nil for lines without URLs, and
			// lineExemptByURL handles nil slices by returning false.
			if !r.Stern && lineExemptByURL(lineURLLens[i+1], lineLen, limit) {
				continue
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  limit + 1,
				Message: fmt.Sprintf("Line length [Expected: %d; Actual: %d]", limit, lineLen),
			})
		}
	}
	return violations
}

// linkDefURLRE matches a link reference definition and captures the URL.
// Example: [label]: https://example.com
var linkDefURLRE = regexp.MustCompile(`^\s*\[[^\]]+\]:\s+(\S+)`)

// urlLengthsPerLine returns a map from 1-based line number to a slice of URL
// rune lengths found on that line. It uses the document's AST to find inline
// links and images, and a regex for link reference definitions.
func urlLengthsPerLine(doc *lint.Document) map[int][]int {
	result := make(map[int][]int)
	addURL := func(lineNum, urlLen int) {
		if lineNum > 0 && urlLen > 0 {
			result[lineNum] = append(result[lineNum], urlLen)
		}
	}

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Link:
			lineNum := inlineLinkLine(node, doc.Source)
			addURL(lineNum, utf8.RuneCount(node.Destination))
		case *ast.Image:
			lineNum := inlineLinkLine(node, doc.Source)
			addURL(lineNum, utf8.RuneCount(node.Destination))
		case *ast.AutoLink:
			lineNum := autoLinkSourceLine(node, doc.Source)
			addURL(lineNum, utf8.RuneCount(node.URL(doc.Source)))
		}
		return ast.WalkContinue, nil
	})

	// Link reference definitions are not exposed as link nodes in the AST.
	for i, line := range doc.Lines {
		if m := linkDefURLRE.FindStringSubmatch(line); m != nil {
			addURL(i+1, utf8.RuneCountInString(m[1]))
		}
	}

	return result
}

// inlineLinkLine returns the 1-based line number for a Link or Image node by
// inspecting its child Text nodes. Falls back to the nearest parent block line.
func inlineLinkLine(n ast.Node, source []byte) int {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			return countLine(source, t.Segment.Start)
		}
	}
	return blockFirstLine(n, source)
}

// autoLinkSourceLine returns the 1-based line number for an AutoLink node.
// It checks adjacent Text siblings first (next sibling is preferred because
// it marks the end of the current line), then falls back to the parent block.
func autoLinkSourceLine(n ast.Node, source []byte) int {
	if next := n.NextSibling(); next != nil {
		if t, ok := next.(*ast.Text); ok {
			return countLine(source, t.Segment.Start)
		}
	}
	if prev := n.PreviousSibling(); prev != nil {
		if t, ok := prev.(*ast.Text); ok {
			return countLine(source, t.Segment.Start)
		}
	}
	return blockFirstLine(n, source)
}

// blockFirstLine returns the 1-based line number of the first line of the
// nearest ancestor block node that has line information.
func blockFirstLine(n ast.Node, source []byte) int {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Lines() != nil && p.Lines().Len() > 0 {
			return countLine(source, p.Lines().At(0).Start)
		}
	}
	return 0
}

// lineExemptByURL reports whether a line that exceeds limit is exempt because
// removing any single URL from its length would make it fit within limit.
// Returns false for nil or empty urlLens (no URL on the line).
func lineExemptByURL(urlLens []int, lineLen, limit int) bool {
	for _, ul := range urlLens {
		if lineLen-ul <= limit {
			return true
		}
	}
	return false
}
