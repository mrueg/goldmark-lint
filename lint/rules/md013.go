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

	// Build a set of "link only" line numbers: lines whose only non-whitespace
	// content is links or images (no bare text outside link/image nodes).
	// Markdownlint exempts such lines because they cannot be split at the URL.
	// Also build the set of link reference definition line indices (0-based).
	var linkOnlyLines map[int]bool
	var linkRefDefLines map[int]bool
	if !r.Stern {
		linkOnlyLines = md013LinkOnlyLines(doc)
		linkRefDefLines = make(map[int]bool)
		for i, line := range doc.Lines {
			if label := linkRefLabel(line); label != "" {
				linkRefDefLines[i] = true
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
			// Skip "link only" lines: lines whose non-whitespace content consists
			// entirely of links/images, with no bare text outside them.
			// Such lines mirror markdownlint's linkOnlyLineNumbers exemption.
			if !r.Stern && linkOnlyLines[i+1] {
				continue
			}
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

// urlLengthsPerLine returns a map from 1-based line number to a slice of URL
// rune lengths found on that line. It uses the document's AST to find inline
// links and images, and doc.LinkRefs (populated from the goldmark parser
// context) to find link reference definition lines.
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
			dest := node.Destination
			// Only apply the URL exemption for inline links where the URL actually
			// appears on the source line. Reference links resolve their URL from a
			// definition elsewhere; attributing that URL length to the using line
			// would incorrectly exempt lines that only contain a short reference label.
			if lineNum >= 1 && lineNum <= len(doc.Lines) && strings.Contains(doc.Lines[lineNum-1], string(dest)) {
				addURL(lineNum, utf8.RuneCount(dest))
			}
		case *ast.Image:
			lineNum := inlineLinkLine(node, doc.Source)
			dest := node.Destination
			if lineNum >= 1 && lineNum <= len(doc.Lines) && strings.Contains(doc.Lines[lineNum-1], string(dest)) {
				addURL(lineNum, utf8.RuneCount(dest))
			}
		case *ast.AutoLink:
			lineNum := autoLinkSourceLine(node, doc.Source)
			addURL(lineNum, utf8.RuneCount(node.URL(doc.Source)))
		}
		return ast.WalkContinue, nil
	})

	// Use goldmark's parsed link reference definitions (doc.LinkRefs) to find
	// definition lines. We scan the raw lines for a matching label and look up
	// the destination length from the goldmark-parsed map, so angle-bracket
	// destinations and backslash escapes are handled correctly.
	if len(doc.LinkRefs) > 0 {
		for i, line := range doc.Lines {
			label := linkRefLabel(line)
			if label == "" {
				continue
			}
			key := strings.ToLower(label)
			if dest, ok := doc.LinkRefs[key]; ok {
				addURL(i+1, utf8.RuneCount(dest))
			}
		}
	}

	// Also detect bare URLs (not part of link syntax) in raw lines.
	// markdownlint exempts lines where a bare URL is the reason for exceeding
	// the limit. We scan each line for plain http(s):// URLs.
	for i, line := range doc.Lines {
		for _, m := range md013BareURLRE.FindAllString(line, -1) {
			addURL(i+1, utf8.RuneCountInString(m))
		}
	}

	return result
}

// md013BareURLRE matches bare http/https URLs in plain text (not wrapped in
// markdown link or angle-bracket auto-link syntax).
var md013BareURLRE = regexp.MustCompile(`https?://\S+`)

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
// inspecting its descendant Text nodes (recursing into CodeSpan and other
// inline containers). Falls back to the nearest parent block line.
func inlineLinkLine(n ast.Node, source []byte) int {
	if t := firstTextLeaf(n); t != nil {
		return countLine(source, t.Segment.Start)
	}
	return blockFirstLine(n, source)
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

// trailingWordTrimmedLen returns the effective length of line for MD013 in
// non-strict, non-stern mode. It mirrors markdownlint's behaviour of replacing
// the trailing run of non-whitespace with a single '#' before checking length,
// so that a line whose only violation is a long final word is not flagged
// (the last word cannot be wrapped to the next line).
func trailingWordTrimmedLen(line string) int {
	// Trim trailing whitespace first, matching markdownlint's behaviour.
	// A line that ends with whitespace does not have a "trailing word" to trim,
	// so its effective length is the trimmed length.
	trimmed := strings.TrimRight(line, " \t")
	runes := []rune(trimmed)
	n := len(runes)
	// Find the start of the trailing non-whitespace run.
	end := n
	for end > 0 && runes[end-1] != ' ' && runes[end-1] != '\t' {
		end--
	}
	// Simulate markdownlint's line.replace(/\S*$/u, "#"):
	// replace the trailing non-whitespace run with a single '#' (1 rune).
	// When end == 0, the entire line is one word; the effective length is 1.
	return end + 1
}

// md013LinkOnlyLines returns a set of 1-based line numbers for lines whose
// non-whitespace content consists entirely of links or images, with no bare
// text nodes outside link/image containers at the immediate paragraph/block level.
// Markdownlint exempts such lines because the URL is the unavoidable cause of
// the length (the line cannot be reformatted to fit within the limit).
func md013LinkOnlyLines(doc *lint.Document) map[int]bool {
	// linkLines: 1-based line numbers that contain at least one link or image.
	linkLines := make(map[int]bool)
	// paragraphDataLines: 1-based line numbers that have a Text node whose
	// DIRECT parent is NOT a link or image (i.e. bare text in the paragraph).
	paragraphDataLines := make(map[int]bool)

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.Kind() {
		case ast.KindLink, ast.KindImage:
			lineNum := inlineLinkLine(n, doc.Source)
			if lineNum > 0 {
				linkLines[lineNum] = true
			}
		case ast.KindText:
			// If this Text node is a direct child of a link or image, it is the
			// link/image label text – not bare paragraph content.
			p := n.Parent()
			if p != nil && (p.Kind() == ast.KindLink || p.Kind() == ast.KindImage) {
				break
			}
			t, ok := n.(*ast.Text)
			if !ok {
				break
			}
			lineNum := countLine(doc.Source, t.Segment.Start)
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
