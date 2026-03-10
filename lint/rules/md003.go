package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD003 checks that headings use a consistent style (ATX or Setext).
type MD003 struct {
	// Style is the required heading style: "consistent" (default), "atx", "setext",
	// "atx_closed", "setext_with_atx", or "setext_with_atx_closed".
	Style string `json:"style"`
}

func (r MD003) ID() string          { return "MD003" }
func (r MD003) Aliases() []string   { return []string{"heading-style"} }
func (r MD003) Description() string { return "Heading style" }

// md003IsSetextUnderline reports whether s is a setext heading underline:
// a non-empty string consisting entirely of '=' or '-' characters.
func md003IsSetextUnderline(s string) bool {
	if len(s) == 0 {
		return false
	}
	ch := s[0]
	if ch != '=' && ch != '-' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] != ch {
			return false
		}
	}
	return true
}

// md003ParseATX tries to parse line as an ATX heading.
// Returns (level, text, actualStyle, ok).
// actualStyle is "atx" or "atx_closed".
func md003ParseATX(line string) (level int, text, actualStyle string, ok bool) {
	stripped := strings.TrimLeft(line, " ")
	leadingSpaces := len(line) - len(stripped)
	if leadingSpaces > 3 || len(stripped) == 0 || stripped[0] != '#' {
		return 0, "", "", false
	}
	j := 0
	for j < len(stripped) && stripped[j] == '#' {
		j++
	}
	if j < 1 || j > 6 {
		return 0, "", "", false
	}
	if j >= len(stripped) || stripped[j] != ' ' {
		return 0, "", "", false
	}
	level = j
	textPart := strings.TrimRight(stripped[j+1:], " ")
	actualStyle = "atx"
	// Check for closed ATX: ends with one or more '#' preceded by a space.
	if len(textPart) > 0 && textPart[len(textPart)-1] == '#' {
		k := len(textPart) - 1
		for k > 0 && textPart[k] == '#' {
			k--
		}
		if textPart[k] == ' ' {
			actualStyle = "atx_closed"
			textPart = strings.TrimRight(textPart[:k], " ")
		}
	}
	return level, textPart, actualStyle, true
}

// Fix rewrites all headings in source to use the required style.
//
// Conversions supported:
//   - ATX  ↔  ATX closed  (add/remove trailing "##" marker)
//   - ATX  ↔  Setext       (add/remove underline line; only for h1/h2)
//   - ATX closed ↔ Setext
//
// Headings are collected in a forward pass and then processed in reverse order
// so that inserting or removing lines does not invalidate earlier line indices.
func (r MD003) Fix(source []byte) []byte {
	style := r.Style
	if style == "" {
		style = "consistent"
	}

	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)

	type hdgInfo struct {
		lineIdx      int    // 0-based line index of the heading content (ATX) or text (setext)
		underlineIdx int    // setext underline line index, or -1
		level        int
		text         string
		actualStyle  string // "atx", "atx_closed", or "setext"
	}

	var hdgs []hdgInfo

	for i := 0; i < len(lines); i++ {
		if mask[i] {
			continue
		}
		line := lines[i]

		// Try ATX.
		if level, text, actual, ok := md003ParseATX(line); ok {
			hdgs = append(hdgs, hdgInfo{
				lineIdx:      i,
				underlineIdx: -1,
				level:        level,
				text:         text,
				actualStyle:  actual,
			})
			continue
		}

		// Try setext underline: current line is all '=' or '-', previous non-mask
		// line is non-empty and non-ATX.
		if i > 0 && !mask[i-1] {
			trimLine := strings.TrimSpace(line)
			if md003IsSetextUnderline(trimLine) {
				prev := lines[i-1]
				prevStripped := strings.TrimSpace(prev)
				// Previous line must be non-empty and not an ATX heading.
				prevTrimLeft := strings.TrimLeft(prev, " ")
				if prevStripped != "" && (len(prevTrimLeft) == 0 || prevTrimLeft[0] != '#') {
					level := 1
					if trimLine[0] == '-' {
						level = 2
					}
					hdgs = append(hdgs, hdgInfo{
						lineIdx:      i - 1,
						underlineIdx: i,
						level:        level,
						text:         prevStripped,
						actualStyle:  "setext",
					})
				}
			}
		}
	}

	if len(hdgs) == 0 {
		return source
	}

	// Determine "consistent" target style from the first heading.
	if style == "consistent" {
		style = hdgs[0].actualStyle
	}

	changed := false

	// Process in reverse order to keep earlier line indices valid when we
	// insert or delete lines.
	for hi := len(hdgs) - 1; hi >= 0; hi-- {
		h := hdgs[hi]

		// Resolve the expected style for this specific heading.
		expected := style
		switch style {
		case "setext_with_atx":
			if h.level <= 2 {
				expected = "setext"
			} else {
				expected = "atx"
			}
		case "setext_with_atx_closed":
			if h.level <= 2 {
				expected = "setext"
			} else {
				expected = "atx_closed"
			}
		case "setext":
			if h.level > 2 {
				expected = "atx" // setext cannot represent h3+
			}
		}

		if h.actualStyle == expected {
			continue
		}
		changed = true

		origLine := lines[h.lineIdx]
		leadingSpaces := len(origLine) - len(strings.TrimLeft(origLine, " "))
		indent := strings.Repeat(" ", leadingSpaces)
		hashes := strings.Repeat("#", h.level)

		switch expected {
		case "atx":
			if h.actualStyle == "setext" {
				// Remove underline, prepend '#'.
				lines[h.lineIdx] = indent + hashes + " " + h.text
				lines = append(lines[:h.underlineIdx], lines[h.underlineIdx+1:]...)
			} else {
				// atx_closed → atx: remove trailing hashes.
				lines[h.lineIdx] = indent + hashes + " " + h.text
			}

		case "atx_closed":
			if h.actualStyle == "setext" {
				lines[h.lineIdx] = indent + hashes + " " + h.text + " " + hashes
				lines = append(lines[:h.underlineIdx], lines[h.underlineIdx+1:]...)
			} else {
				// atx (or setext already handled) → atx_closed.
				lines[h.lineIdx] = indent + hashes + " " + h.text + " " + hashes
			}

		case "setext":
			// Only h1 and h2 can be setext (h3+ fall through to atx above).
			underlineChar := "="
			if h.level == 2 {
				underlineChar = "-"
			}
			textLen := len([]rune(h.text))
			if textLen < 3 {
				textLen = 3
			}
			underline := indent + strings.Repeat(underlineChar, textLen)

			if h.actualStyle != "setext" {
				// ATX or ATX_closed → setext: replace heading line and insert underline.
				lines[h.lineIdx] = indent + h.text
				// Insert underline immediately after the content line.
				newLines := make([]string, 0, len(lines)+1)
				newLines = append(newLines, lines[:h.lineIdx+1]...)
				newLines = append(newLines, underline)
				newLines = append(newLines, lines[h.lineIdx+1:]...)
				lines = newLines
			}
		}
	}

	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}

// headingStyleOf returns "atx", "atx_closed", or "setext" for the given heading node by
// looking back in the source to find the start of the line: if it starts
// with '#' it is ATX (possibly closed), otherwise it is setext.
func headingStyleOf(h *ast.Heading, source []byte) string {
	if h.Lines() == nil || h.Lines().Len() == 0 {
		return "atx"
	}
	seg := h.Lines().At(0)
	if seg.Start > len(source) {
		return "atx"
	}
	// Walk back to the start of the line to find the marker.
	pos := seg.Start
	for pos > 0 && source[pos-1] != '\n' {
		pos--
	}
	// Skip any leading spaces and blockquote markers ("> ") so that headings
	// inside blockquotes like "> # Heading" are correctly identified as ATX.
	cur := pos
outer:
	for cur < len(source) {
		switch source[cur] {
		case ' ':
			cur++
		case '>':
			cur++
			if cur < len(source) && source[cur] == ' ' {
				cur++
			}
		default:
			break outer
		}
	}
	if cur >= len(source) || source[cur] != '#' {
		return "setext"
	}
	// It's ATX. Find the end of the line to detect closed ATX (## Heading ##).
	end := cur
	for end < len(source) && source[end] != '\n' {
		end++
	}
	lineStr := strings.TrimRight(string(source[cur:end]), " ")
	// Count leading '#' characters (the heading marker).
	leadingHashes := 0
	for leadingHashes < len(lineStr) && lineStr[leadingHashes] == '#' {
		leadingHashes++
	}
	// Closed ATX: the line ends with one or more '#' preceded by a space,
	// and there is content between the opening and closing markers.
	if leadingHashes < len(lineStr) && lineStr[len(lineStr)-1] == '#' {
		// Find where the trailing '#' run starts.
		i := len(lineStr) - 1
		for i > leadingHashes && lineStr[i] == '#' {
			i--
		}
		// The character before the trailing '#' run must be a space.
		if lineStr[i] == ' ' {
			return "atx_closed"
		}
	}
	return "atx"
}

func (r MD003) Check(doc *lint.Document) []lint.Violation {
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
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		line := 1
		if h.Lines() != nil && h.Lines().Len() > 0 {
			seg := h.Lines().At(0)
			line = countLine(doc.Source, seg.Start)
		}

		actual := headingStyleOf(h, doc.Source)

		expected := style
		if style == "consistent" {
			if firstStyle == "" {
				firstStyle = actual
			}
			expected = firstStyle
		}

		// setext only supports h1 and h2; for deeper levels ATX is required.
		switch expected {
		case "setext_with_atx":
			if h.Level <= 2 {
				expected = "setext"
			} else {
				expected = "atx"
			}
		case "setext_with_atx_closed":
			if h.Level <= 2 {
				expected = "setext"
			} else {
				expected = "atx_closed"
			}
		case "setext":
			if h.Level > 2 {
				// setext cannot represent h3+, so ATX is acceptable.
				expected = "atx"
			}
		}

		matches := actual == expected
		if !matches {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    line,
				Column:  1,
				Message: fmt.Sprintf("Heading style [Expected: %s; Actual: %s]", expected, actual),
			})
		}
		return ast.WalkContinue, nil
	})

	return violations
}
