package rules

import (
	"fmt"
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

	// Build fenced code block mask using the goldmark AST (includes fence
	// delimiter lines, fenced content, and falls back to string-based
	// detection for empty fenced blocks that have no content lines in the AST).
	codeBlockMask := astFencedCodeBlockMask(doc)
	// Also mark indented code block lines via the AST.
	for i, v := range indentedCodeBlockMask(doc) {
		if v {
			codeBlockMask[i] = true
		}
	}

	// Build table mask using the goldmark AST.
	tableMask := astTableMask(doc)
	// Remove table entries that are actually inside code blocks.
	for i := range tableMask {
		if codeBlockMask[i] {
			tableMask[i] = false
		}
	}

	// Build heading mask using the goldmark AST.
	headingMask := astHeadingMask(doc)
	// Remove heading entries that are actually inside code blocks.
	for i := range headingMask {
		if codeBlockMask[i] {
			headingMask[i] = false
		}
	}

	// Build a set of "link only" line numbers: lines whose only non-whitespace
	// content is links or images (no bare text outside link/image nodes).
	// Markdownlint exempts such lines because they cannot be split at the URL.
	// Also build the set of link reference definition line indices (0-based).
	// Additionally build the set of table row line numbers that contain any
	// resolved Link or Image node — markdownlint exempts ALL such table rows.
	var linkOnlyLines map[int]bool
	var linkRefDefLines map[int]bool
	var tableRowLinkLines map[int]bool
	if !r.Stern {
		linkOnlyLines = md013LinkOnlyLines(doc)
		linkRefDefLines = make(map[int]bool)
		for i, line := range doc.Lines {
			if label := linkRefLabel(line); label != "" {
				linkRefDefLines[i] = true
				// Also exempt the following line if it is the title continuation
				// of a multi-line link reference definition.  CommonMark allows
				// the optional title to appear on the line immediately after the
				// destination:
				//   [label]: url
				//   "title on next line"
				// CommonMark titles can be delimited by "...", '...', or (...).
				// Detecting the opening delimiter character is sufficient: this
				// line can only appear here as a link title because it
				// immediately follows a recognised link reference definition
				// line, so false-positive paragraph text is not a concern.
				if i+1 < len(doc.Lines) {
					next := strings.TrimLeft(doc.Lines[i+1], " \t")
					if len(next) > 0 && (next[0] == '"' || next[0] == '\'' || next[0] == '(') {
						linkRefDefLines[i+1] = true
					}
				}
			}
		}
		// Collect 1-based line numbers of table rows that contain a link or image.
		tableRowLinkLines = make(map[int]bool)
		_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			if n.Kind() != ast.KindLink && n.Kind() != ast.KindImage {
				return ast.WalkContinue, nil
			}
			lineNum := inlineLinkLine(n, doc)
			if lineNum > 0 && lineNum <= len(tableMask) && tableMask[lineNum-1] {
				tableRowLinkLines[lineNum] = true
			}
			return ast.WalkContinue, nil
		})
		// A bare URL counts as a link here too. markdownlint parses GFM autolink
		// literals, so a table row carrying a plain https:// address holds a link
		// node for it and is exempt; goldmark does not enable the Linkify
		// extension, so the same row has only text and would be reported.
		for i, line := range doc.Lines {
			if !tableMask[i] || tableRowLinkLines[i+1] {
				continue
			}
			if bareURLRE.MatchString(line) {
				tableRowLinkLines[i+1] = true
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
		lineLen := utf8.RuneCountInString(line)
		// In non-strict, non-stern mode markdownlint exempts lines where only the
		// trailing non-whitespace "word" causes the line to exceed the limit.
		// It replaces the trailing run of non-whitespace with a single '#' before
		// checking length, so a line that is all one word (no spaces) is never
		// flagged in this mode.
		effectiveLen := lineLen
		if !r.Strict && !r.Stern {
			effectiveLen = trailingWordTrimmedLen(line)
		}
		if effectiveLen > limit {
			// Skip link reference definition lines (e.g. "[label]: url").
			if !r.Stern && linkRefDefLines[i] {
				continue
			}
			// Skip "link only" lines: lines whose non-whitespace content
			// consists entirely of links or images cannot be reformatted,
			// so they are always exempt. This applies to both inline links
			// (where the URL appears on the line) and reference links
			// (where the URL lives in a separate definition), matching
			// markdownlint behaviour.
			if !r.Stern && linkOnlyLines[i+1] {
				continue
			}
			// Table rows that contain any resolved link or image node are exempt.
			// Markdownlint skips the line-length check for all such rows regardless
			// of URL type (inline URL or reference link resolved via a definition).
			if !r.Stern && tableMask[i] && tableRowLinkLines[i+1] {
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

// linkRefLabel returns the link-reference label from a line that looks like a
// link reference definition (e.g. "[foo]: https://..."), or "" if the line
// is not a definition.  The returned string is the raw label text (before
// normalisation).
func linkRefLabel(line string) string {
	// Up to 3 leading spaces, then '['.
	i := 0
	for i < len(line) && i < 3 && line[i] == ' ' {
		i++
	}
	if i >= len(line) || line[i] != '[' {
		return ""
	}
	i++ // skip '['
	start := i
	for i < len(line) {
		if line[i] == '\\' && i+1 < len(line) {
			i += 2 // skip backslash-escaped character
			continue
		}
		if line[i] == ']' || line[i] == '[' {
			break
		}
		i++
	}
	if i >= len(line) || line[i] != ']' || i == start {
		return ""
	}
	label := line[start:i]
	i++ // skip ']'
	if i >= len(line) || line[i] != ':' {
		return ""
	}
	return label
}

// inlineLinkLine returns the 1-based line number for a Link or Image node by
// using Pos() when available, then falling back to the first Text descendant
// or the nearest parent block line.
func inlineLinkLine(n ast.Node, doc *lint.Document) int {
	if pos := n.Pos(); pos >= 0 {
		return doc.LineAt(pos)
	}
	if t := firstTextLeaf(n); t != nil {
		return doc.LineAt(t.Segment.Start)
	}
	return blockFirstLine(n, doc)
}

// firstTextLeaf returns the first *ast.Text leaf under n (depth-first), or nil.
func firstTextLeaf(n ast.Node) *ast.Text {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			return t
		}
		if t := firstTextLeaf(c); t != nil {
			return t
		}
	}
	return nil
}

// lastTextLeaf returns the last *ast.Text leaf under n (depth-first), or nil.
func lastTextLeaf(n ast.Node) *ast.Text {
	var last *ast.Text
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			last = t
		}
		if t := lastTextLeaf(c); t != nil {
			last = t
		}
	}
	return last
}

// blockFirstLine returns the 1-based line number of the first line of the
// nearest ancestor block node that has line information.
func blockFirstLine(n ast.Node, doc *lint.Document) int {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Type() != ast.TypeBlock {
			continue
		}
		if p.Lines() != nil && p.Lines().Len() > 0 {
			return doc.LineAt(p.Lines().At(0).Start)
		}
	}
	return 0
}

// trailingWordTrimmedLen returns the effective length of line for MD013 in
// non-strict, non-stern mode. It mirrors markdownlint's behaviour of replacing
// the trailing run of non-whitespace with a single '#' before checking length,
// so that a line whose only violation is a long final word is not flagged
// (the last word cannot be wrapped to the next line).
// Note: trailing whitespace is NOT stripped before computing the trailing run.
// This matches markdownlint's line.replace(/\S*$/u, "#") behaviour: when a
// line ends with whitespace, the trailing non-whitespace run is empty and
// the replacement appends '#', making the effective length = lineLen + 1.
func trailingWordTrimmedLen(line string) int {
	runes := []rune(line)
	n := len(runes)
	// Find the start of the trailing non-whitespace run.
	end := n
	for end > 0 && runes[end-1] != ' ' && runes[end-1] != '\t' {
		end--
	}
	// Simulate markdownlint's line.replace(/\S*$/u, "#"):
	// replace the trailing non-whitespace run with a single '#' (1 rune).
	// When end == 0, the entire line is one word; the effective length is 1.
	//
	// Special case: when end == n the line ends with whitespace (no trailing
	// non-whitespace run). Markdownlint's regex replaces the empty match at
	// the end with '#', giving text.length = n+1. However, empirical testing
	// shows markdownlint does NOT flag such lines when n == line_length
	// (i.e., when the content including the trailing space is exactly at the
	// limit). The effective check matches line.length > limit, so we return n
	// rather than n+1 to avoid the off-by-one for exactly-at-limit lines.
	if end == n {
		return n
	}
	return end + 1
}

// md013LinkOnlyLines returns a set of 1-based line numbers for lines whose
// non-whitespace content consists entirely of links or images, with no bare
// text nodes at the paragraph level (i.e. direct block children).
// Markdownlint exempts such lines because the URL is the unavoidable cause of
// the length (the line cannot be reformatted to fit within the limit).
//
// This mirrors markdownlint's micromark-based logic: only "data" tokens that
// are direct children of a "paragraph" token contribute to paragraphDataLines.
// Text inside inline containers (Strong, Emphasis, Link, Image, CodeSpan, etc.)
// is therefore NOT counted as bare paragraph content, matching the way
// markdownlint handles lines like "**text [link](url)**".
func md013LinkOnlyLines(doc *lint.Document) map[int]bool {
	// linkLines: 1-based line numbers that contain at least one link or image.
	linkLines := make(map[int]bool)
	// paragraphDataLines: 1-based line numbers that have a Text node whose
	// direct parent is a block node (Paragraph, ListItem, etc.) — not inside
	// an inline container like Strong, Emphasis, Link, Image, or CodeSpan.
	paragraphDataLines := make(map[int]bool)

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.Kind() {
		case ast.KindLink, ast.KindImage:
			// Mark ALL lines spanned by this link/image as link-containing.
			// Multiline images/links (alt text spanning multiple lines) must have
			// each line marked; otherwise those continuation lines would not be
			// considered link-only and would be incorrectly flagged.
			if first := firstTextLeaf(n); first != nil {
				startLine := doc.LineAt(first.Segment.Start)
				endLine := startLine
				if last := lastTextLeaf(n); last != nil {
					endLine = doc.LineAt(last.Segment.Start)
					if endLine < startLine {
						endLine = startLine
					}
				}
				for ln := startLine; ln <= endLine; ln++ {
					if ln > 0 {
						linkLines[ln] = true
					}
				}
			}
		case ast.KindAutoLink:
			// AutoLinks (angle-bracket URLs like <https://...>) also make a
			// line "link-containing" for the link-only check.
			lineNum := 0
			if pos := n.Pos(); pos >= 0 {
				lineNum = doc.LineAt(pos)
			} else if next := n.NextSibling(); next != nil {
				if t, ok := next.(*ast.Text); ok {
					lineNum = doc.LineAt(t.Segment.Start)
				}
			} else if prev := n.PreviousSibling(); prev != nil {
				if t, ok := prev.(*ast.Text); ok {
					lineNum = doc.LineAt(t.Segment.Start)
				}
			} else {
				lineNum = blockFirstLine(n, doc)
			}
			if lineNum > 0 {
				linkLines[lineNum] = true
			}
		case ast.KindText:
			// Only count this Text node as "bare paragraph data" when its direct
			// parent is a block node that is NOT an inline container and NOT a
			// heading. This mirrors markdownlint's micromark behaviour:
			//   - text inside inline containers (Strong, Emphasis, Link, Image,
			//     CodeSpan — all TypeInline) is embedded in markup and not counted
			//   - text inside headings is not counted (headings are not "paragraph"
			//     tokens in micromark, so their text never enters paragraphDataLines)
			parent := n.Parent()
			if parent == nil || parent.Type() == ast.TypeInline || parent.Kind() == ast.KindHeading {
				break
			}
			t, ok := n.(*ast.Text)
			if !ok {
				break
			}
			// Skip empty text segments (e.g. paragraph-end markers generated
			// by the goldmark parser at line breaks adjacent to inline nodes).
			if t.Segment.Start == t.Segment.Stop {
				break
			}
			lineNum := doc.LineAt(t.Segment.Start)
			if lineNum > 0 {
				paragraphDataLines[lineNum] = true
			}
		}
		return ast.WalkContinue, nil
	})

	result := make(map[int]bool)
	for lineNum := range linkLines {
		if !paragraphDataLines[lineNum] {
			result[lineNum] = true
		}
	}
	return result
}
