package rules

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
)

// IntOrArray is a JSON-compatible type that can be either a single integer or
// an array of integers. When used for per-heading-level config (e.g. lines_above
// in MD022), index 0 = h1, index 1 = h2, etc.
type IntOrArray []int

// UnmarshalJSON implements json.Unmarshaler.
func (x *IntOrArray) UnmarshalJSON(data []byte) error {
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*x = IntOrArray{n}
		return nil
	}
	var arr []int
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	*x = IntOrArray(arr)
	return nil
}

// Get returns the configured value for the given 1-based heading level.
// If the array is empty, 0 is returned (callers should use their default).
// If the level exceeds the array length, the last element is returned.
func (x IntOrArray) Get(level int) int {
	if len(x) == 0 {
		return 0
	}
	idx := level - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(x) {
		return x[len(x)-1]
	}
	return x[idx]
}

// countLine counts the 1-based line number of byte offset pos in source.
func countLine(source []byte, pos int) int {
	line := 1
	for i := 0; i < pos && i < len(source); i++ {
		if source[i] == '\n' {
			line++
		}
	}
	return line
}

// headingText returns the text content of a heading node by recursively
// extracting text from all inline descendants. This includes text inside
// code spans, emphasis, strong, links, etc., matching GitHub's anchor
// generation which uses the full heading text.
func headingText(n ast.Node, source []byte) string {
	var b []byte
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		b = append(b, inlineNodeText(c, source)...)
	}
	return string(b)
}

// inlineNodeText recursively extracts raw text bytes from an inline AST node.
func inlineNodeText(n ast.Node, source []byte) []byte {
	if t, ok := n.(*ast.Text); ok {
		return source[t.Segment.Start:t.Segment.Stop]
	}
	// For any other node, recurse into its children.
	var b []byte
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		b = append(b, inlineNodeText(c, source)...)
	}
	return b
}

// fencedCodeBlockLine returns the 1-based line number of the opening fence of a
// FencedCodeBlock node. It tries Info segment first, then first content line minus
// one, and falls back to 1 for empty blocks with no info string.
func fencedCodeBlockLine(n *ast.FencedCodeBlock, source []byte) int {
	if n.Info != nil {
		return countLine(source, n.Info.Segment.Start)
	}
	if n.Lines() != nil && n.Lines().Len() > 0 {
		line := countLine(source, n.Lines().At(0).Start)
		if line > 1 {
			return line - 1
		}
	}
	return 1
}

// emphasisStartPos returns the byte position in source of the opening marker of
// the given Emphasis node. It tries Pos() first, then walks to the first Text
// descendant and subtracts nesting levels and inline-wrapper prefix characters.
func emphasisStartPos(emph *ast.Emphasis) int {
	if pos := emph.Pos(); pos >= 0 {
		return pos
	}
	pos, ok := firstTextStartInInline(emph.FirstChild(), emph.Level)
	if ok {
		return pos
	}
	return -1
}

// firstTextStartInInline recursively finds the first *ast.Text in an inline
// subtree and returns (start_of_emphasis_marker, true) by subtracting
// accumulated prefix character counts.
func firstTextStartInInline(n ast.Node, prefixChars int) (int, bool) {
	if n == nil {
		return 0, false
	}
	switch node := n.(type) {
	case *ast.Text:
		p := node.Segment.Start - prefixChars
		if p >= 0 {
			return p, true
		}
		return 0, false
	case *ast.Emphasis:
		return firstTextStartInInline(node.FirstChild(), prefixChars+node.Level)
	case *ast.Link:
		// Link starts with '['; account for that extra char.
		return firstTextStartInInline(node.FirstChild(), prefixChars+1)
	case *ast.Image:
		// Image starts with '!['; account for those 2 extra chars.
		return firstTextStartInInline(node.FirstChild(), prefixChars+2)
	default:
		// For any other inline node (CodeSpan, etc.), try its first child.
		return firstTextStartInInline(n.FirstChild(), prefixChars)
	}
}

// headingSourceLine returns the 1-based line number of a heading node in source
// using the first content segment, or 0 if no line information is available.
func headingSourceLine(h *ast.Heading, source []byte) int {
	if h.Lines() == nil || h.Lines().Len() == 0 {
		return 0
	}
	return countLine(source, h.Lines().At(0).Start)
}

// fencedCodeBlockMask returns a bool slice with true for each line that is
// inside (not on the fence delimiters of) a fenced code block.
// Per CommonMark spec, a fenced code block may be indented by at most 3 spaces.
func fencedCodeBlockMask(lines []string) []bool {
	mask := make([]bool, len(lines))
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	for i, line := range lines {
		// Count leading spaces; fences require at most 3 spaces of indentation.
		indent := 0
		for indent < len(line) && line[indent] == ' ' {
			indent++
		}
		trimmed := line[indent:]
		if !inFence {
			if indent <= 3 && len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
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

// tableDelimiterCellRE matches a GFM table delimiter cell.
var tableDelimiterCellRE = regexp.MustCompile(`^\s*:?-+:?\s*$`)

// isTableDelimiterRow returns true if line is a GFM table delimiter row.
func isTableDelimiterRow(line string) bool {
	if !strings.Contains(line, "|") {
		return false
	}
	trimmed := strings.TrimPrefix(strings.TrimSpace(line), "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	cells := strings.Split(trimmed, "|")
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if !tableDelimiterCellRE.MatchString(cell) {
			return false
		}
	}
	return true
}

// isTableRow returns true if line looks like a table row (contains |).
func isTableRow(line string) bool {
	return strings.Contains(line, "|")
}

// lastTextStopInInline returns the highest Segment.Stop value found among all
// *ast.Text descendants of n, searching recursively. Returns 0 if none found.
func lastTextStopInInline(n ast.Node) int {
	var best int
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			if t.Segment.Stop > best {
				best = t.Segment.Stop
			}
		} else {
			if v := lastTextStopInInline(c); v > best {
				best = v
			}
		}
	}
	return best
}


// headingAnchor converts heading text to a GitHub-style anchor.
func headingAnchor(text string) string {
	result := strings.ToLower(text)
	var b strings.Builder
	for _, r := range result {
		if r == ' ' || r == '-' {
			b.WriteRune('-')
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// countTableCells counts the cells in a GFM table row. It ignores pipe
// characters that are inside backtick code spans or are escaped with a
// backslash (\|), matching markdownlint's behaviour.
func countTableCells(line string) int {
	// Blank out pipe characters inside code spans and replace \| with a
	// placeholder so they are not counted as cell separators.
	processed := blankInlineCodeSpans(line)
	// Replace backslash-escaped pipes outside code spans.
	processed = strings.ReplaceAll(processed, `\|`, "__")
	trimmed := strings.TrimPrefix(strings.TrimSpace(processed), "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	return len(strings.Split(trimmed, "|"))
}

// frontMatterHasTitle reports whether the document's front matter contains a
// title field. pattern is a field name or regex; if empty, the feature is
// disabled and false is always returned.
func frontMatterHasTitle(doc *lint.Document, pattern string) bool {
	if pattern == "" {
		return false
	}
	if len(doc.FrontMatterFields) == 0 {
		return false
	}
	// First try exact key match (case-insensitive).
	for k, v := range doc.FrontMatterFields {
		if strings.EqualFold(k, pattern) && v != "" {
			return true
		}
	}
	// Try pattern as a regex matched against "key: value" lines.
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return false
	}
	for k, v := range doc.FrontMatterFields {
		if re.MatchString(k + ": " + v) {
			return true
		}
	}
	return false
}

// fencedCodeBlockLanguages returns a map from line index (0-based) to the
// language of the fenced code block that line belongs to (empty string if
// the block has no language specified). Lines outside code blocks are absent.
func fencedCodeBlockLanguages(lines []string) map[int]string {
	result := make(map[int]string)
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	currentLang := ""
	for i, line := range lines {
		// Count leading spaces; fences require at most 3 spaces of indentation.
		indent := 0
		for indent < len(line) && line[indent] == ' ' {
			indent++
		}
		trimmed := line[indent:]
		if !inFence {
			if indent <= 3 && len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
				fc := trimmed[0]
				j := 0
				for j < len(trimmed) && trimmed[j] == fc {
					j++
				}
				if j >= 3 {
					inFence = true
					fenceChar = fc
					fenceLen = j
					info := strings.TrimSpace(trimmed[j:])
					if fields := strings.Fields(info); len(fields) > 0 {
						currentLang = fields[0]
					} else {
						currentLang = ""
					}
				}
			}
		} else {
			if len(trimmed) >= fenceLen && trimmed[0] == fenceChar {
				j := 0
				for j < len(trimmed) && trimmed[j] == fenceChar {
					j++
				}
				if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" {
					inFence = false
					currentLang = ""
					continue
				}
			}
			result[i] = currentLang
		}
	}
	return result
}

func findTables(lines []string, mask []bool) [][2]int {
	var tables [][2]int
	n := len(lines)
	i := 0
	for i < n {
		if mask[i] || !isTableRow(lines[i]) {
			i++
			continue
		}
		// Check that next line is a delimiter row
		if i+1 >= n || mask[i+1] || !isTableDelimiterRow(lines[i+1]) {
			i++
			continue
		}
		// Table starts at i
		start := i
		end := i + 1
		for end+1 < n && !mask[end+1] && isTableRow(lines[end+1]) && !isTableDelimiterRow(lines[end+1]) {
			end++
		}
		tables = append(tables, [2]int{start, end})
		i = end + 1
	}
	return tables
}

// indentedCodeBlockMask returns a bool slice with true for each line that is
// part of an indented code block (ast.CodeBlock). Used by MD012 to avoid
// flagging blank lines inside indented code blocks.
func indentedCodeBlockMask(doc *lint.Document) []bool {
	mask := make([]bool, len(doc.Lines))
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() != ast.KindCodeBlock {
			return ast.WalkContinue, nil
		}
		for i := 0; i < n.Lines().Len(); i++ {
			seg := n.Lines().At(i)
			lineIdx := countLine(doc.Source, seg.Start) - 1
			if lineIdx >= 0 && lineIdx < len(mask) {
				mask[lineIdx] = true
			}
		}
		return ast.WalkContinue, nil
	})
	return mask
}

// It uses the goldmark AST to accurately detect HTML blocks.
func htmlBlockLineMask(doc *lint.Document) []bool {
	mask := make([]bool, len(doc.Lines))
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() != ast.KindHTMLBlock {
			return ast.WalkContinue, nil
		}
		for i := 0; i < n.Lines().Len(); i++ {
			seg := n.Lines().At(i)
			lineIdx := countLine(doc.Source, seg.Start) - 1
			if lineIdx >= 0 && lineIdx < len(mask) {
				mask[lineIdx] = true
			}
		}
		return ast.WalkContinue, nil
	})
	return mask
}

// astFencedCodeBlockMask returns a bool slice with true for each source line
// that is part of a fenced code block, including the fence delimiter lines
// (opening and closing). It uses the goldmark AST for accurate detection.
func astFencedCodeBlockMask(doc *lint.Document) []bool {
	mask := make([]bool, len(doc.Lines))
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() != ast.KindFencedCodeBlock {
			return ast.WalkContinue, nil
		}
		lines := n.Lines()
		if lines.Len() == 0 {
			// Empty fenced code block: fall through to string-based detection
			// (cannot locate fence delimiters via AST alone when there are no content lines).
			return ast.WalkContinue, nil
		}
		firstSeg := lines.At(0)
		lastSeg := lines.At(lines.Len() - 1)
		firstCodeIdx := countLine(doc.Source, firstSeg.Start) - 1 // 0-indexed
		lastCodeIdx := countLine(doc.Source, lastSeg.Start) - 1   // 0-indexed

		// Opening fence delimiter: line immediately before first code line.
		if firstCodeIdx > 0 {
			mask[firstCodeIdx-1] = true
		}
		// Content lines.
		for lineIdx := firstCodeIdx; lineIdx <= lastCodeIdx; lineIdx++ {
			if lineIdx >= 0 && lineIdx < len(mask) {
				mask[lineIdx] = true
			}
		}
		// Closing fence delimiter: line immediately after last code line, if
		// it exists and starts with the fence character (not another code line).
		closingIdx := lastCodeIdx + 1
		if closingIdx >= 0 && closingIdx < len(mask) {
			mask[closingIdx] = true
		}
		return ast.WalkContinue, nil
	})
	// For empty fenced code blocks (AST Lines() == 0), fall back to the
	// string-based scanner to catch their fence delimiter lines.
	stringMask := fencedCodeBlockMask(doc.Lines)
	for i, v := range stringMask {
		if v {
			mask[i] = true
		}
	}
	return mask
}

// astHeadingMask returns a bool slice with true for each source line that is
// a heading line (ATX or setext, including the setext underline).
// It uses the goldmark AST for accurate detection.
func astHeadingMask(doc *lint.Document) []bool {
	mask := make([]bool, len(doc.Lines))
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() != ast.KindHeading {
			return ast.WalkContinue, nil
		}
		lines := n.Lines()
		if lines.Len() == 0 {
			return ast.WalkContinue, nil
		}
		firstSeg := lines.At(0)
		lineIdx := countLine(doc.Source, firstSeg.Start) - 1 // 0-indexed
		if lineIdx < 0 || lineIdx >= len(mask) {
			return ast.WalkContinue, nil
		}
		mask[lineIdx] = true

		// For setext headings the content line does NOT start with '#'.
		// The underline (===... or ---...) is on the very next line.
		srcLine := doc.Lines[lineIdx]
		trimmed := strings.TrimLeft(srcLine, " \t")
		if len(trimmed) == 0 || trimmed[0] != '#' {
			// Setext heading – mark the underline line as well.
			if lineIdx+1 < len(mask) {
				mask[lineIdx+1] = true
			}
		}
		return ast.WalkContinue, nil
	})
	return mask
}

// astTableMask returns a bool slice with true for each source line that is
// part of a GFM table (header row, delimiter row, and data rows).
// It uses the goldmark AST for accurate detection.
func astTableMask(doc *lint.Document) []bool {
	mask := make([]bool, len(doc.Lines))
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() != extast.KindTable {
			return ast.WalkContinue, nil
		}
		// Collect the first and last 1-based line numbers from all Text descendants.
		first, last := -1, -1
		_ = ast.Walk(n, func(child ast.Node, childEntering bool) (ast.WalkStatus, error) {
			if !childEntering {
				return ast.WalkContinue, nil
			}
			if txt, ok := child.(*ast.Text); ok {
				ln := countLine(doc.Source, txt.Segment.Start)
				if first == -1 || ln < first {
					first = ln
				}
				if ln > last {
					last = ln
				}
			}
			return ast.WalkContinue, nil
		})
		if first == -1 {
			return ast.WalkContinue, nil
		}
		// The delimiter row is always the line immediately after the header row.
		// Ensure we mark at least header + delimiter (first and first+1 in 1-indexed).
		end := last
		if end < first+1 {
			end = first + 1
		}
		// Mark all table lines (convert from 1-indexed to 0-indexed).
		for lineIdx := first - 1; lineIdx <= end-1; lineIdx++ {
			if lineIdx >= 0 && lineIdx < len(mask) {
				mask[lineIdx] = true
			}
		}
		return ast.WalkContinue, nil
	})
	return mask
}

// blankInlineCodeSpans returns a copy of line with the content of every inline
// code span replaced by underscore characters of the same byte length. This
// prevents patterns inside code spans from being matched by rules that operate
// on raw line text (e.g. MD011 reversed-link detection).
func blankInlineCodeSpans(line string) string {
b := []byte(line)
i := 0
for i < len(b) {
if b[i] != '`' {
i++
continue
}
// Count opening backtick run.
start := i
for i < len(b) && b[i] == '`' {
i++
}
tickLen := i - start
contentStart := i
// Search for matching closing backtick run.
found := false
j := i
for j < len(b) {
if b[j] == '`' {
k := j
for k < len(b) && b[k] == '`' {
k++
}
if k-j == tickLen {
// Blank out content between backticks.
for x := contentStart; x < j; x++ {
b[x] = '_'
}
i = k
found = true
break
}
j = k
} else {
j++
}
}
if !found {
// Unclosed code span — leave the rest of the line unchanged.
break
}
}
return string(b)
}
