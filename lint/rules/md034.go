package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD034 checks for bare URLs that are not wrapped in angle brackets or a proper link.
type MD034 struct{}

func (r MD034) ID() string          { return "MD034" }
func (r MD034) Aliases() []string   { return []string{"no-bare-urls"} }
func (r MD034) Description() string { return "Bare URL used" }

// bareURLRE matches an http or https URL within a string, stopping at whitespace
// or common punctuation characters that are unlikely to be part of the URL.
var bareURLRE = regexp.MustCompile(`https?://[^\s<>()\[\]{}'"` + "`" + `]+`)

// inlineLinkRE matches inline markdown links [text](url) for stripping from scanned content.
var inlineLinkRE = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`)

// Fix applies MD034 to source by wrapping bare URLs in angle brackets.
func (r MD034) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	fencedMask := fencedCodeBlockMask(lines)

	changed := false
	for i, line := range lines {
		if fencedMask[i] {
			continue
		}

		// Skip indented code block lines (4+ spaces at start after a blank line).
		// Simple heuristic: skip lines with 4+ leading spaces preceded by a blank line.
		if i > 0 && strings.TrimSpace(lines[i-1]) == "" {
			if len(line) >= 4 && line[0] == ' ' && line[1] == ' ' && line[2] == ' ' && line[3] == ' ' {
				continue
			}
		}

		// Blank out code spans so we don't wrap URLs inside them.
		blanked := blankCodeSpans(line)

		// Find bare URL positions in the blanked line.
		matches := bareURLRE.FindAllStringIndex(blanked, -1)
		if len(matches) == 0 {
			continue
		}

		// Build the new line by inserting < > around URLs that are truly bare.
		var newLine strings.Builder
		prev := 0
		lineChanged := false
		for _, loc := range matches {
			start, end := loc[0], loc[1]
			// Check the character before the URL in the original line.
			if start > 0 {
				ch := line[start-1]
				// Skip if URL is already inside angle brackets, link syntax,
				// or attribute quotes.
				if ch == '<' || ch == '(' || ch == '"' || ch == '\'' {
					newLine.WriteString(line[prev:end])
					prev = end
					continue
				}
			}
			// Also check if the original char at start is '<' (already wrapped).
			if start > 0 && line[start-1] == '<' {
				newLine.WriteString(line[prev:end])
				prev = end
				continue
			}
			// Wrap this URL.
			newLine.WriteString(line[prev:start])
			newLine.WriteByte('<')
			newLine.WriteString(line[start:end])
			newLine.WriteByte('>')
			prev = end
			lineChanged = true
		}
		if lineChanged {
			newLine.WriteString(line[prev:])
			lines[i] = newLine.String()
			changed = true
		}
	}
	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD034) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	// Track reported (lineNum, url) pairs to avoid duplicate violations.
	type reported struct {
		line int
		url  string
	}
	seen := make(map[reported]bool)

	addViolation := func(lineNum int, url string) {
		key := reported{lineNum, url}
		if seen[key] {
			return
		}
		seen[key] = true
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    lineNum,
			Column:  1,
			Message: "Bare URL used",
		})
	}

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		t, ok := n.(*ast.Text)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Skip text inside links, images, or code spans (these are properly formatted
		// or are not user-visible as bare URLs).
		for p := t.Parent(); p != nil; p = p.Parent() {
			switch p.(type) {
			case *ast.Link, *ast.Image, *ast.CodeSpan:
				return ast.WalkContinue, nil
			}
		}

		seg := t.Segment
		text := string(doc.Source[seg.Start:seg.Stop])
		lineBase := countLine(doc.Source, seg.Start)

		// Report each bare URL on its own line.
		// Use FindAllStringIndex to get precise positions for multi-line text nodes.
		for _, loc := range bareURLRE.FindAllStringIndex(text, -1) {
			lineNum := lineBase + strings.Count(text[:loc[0]], "\n")
			// Skip URLs that appear to be link destinations in broken link syntax.
			// When the source has ['label'(url) or similar (a '[' that was consumed
			// as a link opener by the parser, leaving the label as a text node), and
			// the URL is immediately preceded by '(' in the text, markdownlint treats
			// it as an attempted link destination rather than a bare URL.
			// We detect this by scanning the raw source from the start of the current
			// line up to the '(' character and checking for an unclosed '['.
			if loc[0] > 0 && text[loc[0]-1] == '(' {
				// Position of '(' in the original source.
				srcParenPos := seg.Start + loc[0] - 1
				// Find the start of the current line in the source.
				lineStartInSrc := srcParenPos
				for lineStartInSrc > 0 && doc.Source[lineStartInSrc-1] != '\n' {
					lineStartInSrc--
				}
				// Count unclosed '[' in source from line start up to '('.
				depth := 0
				for _, b := range doc.Source[lineStartInSrc:srcParenPos] {
					if b == '[' {
						depth++
					} else if b == ']' && depth > 0 {
						depth--
					}
				}
				if depth > 0 {
					// Unclosed '[' before this '(url)': looks like an attempted link.
					continue
				}
			}
			addViolation(lineNum, text[loc[0]:loc[1]])
		}
		return ast.WalkContinue, nil
	})

	// Also scan raw lines for footnote definitions containing bare URLs.
	// Goldmark treats [^n]: url as a link reference definition and does not expose
	// the URL as a Text node, so we scan the raw source lines directly.
	// We strip inline links ([text](url)) from the content first to avoid
	// flagging URLs that are already properly wrapped in a link.
	// Skip lines inside fenced or indented code blocks to avoid false positives.
	fencedMask := fencedCodeBlockMask(doc.Lines)
	indentMask := indentedCodeBlockMask(doc)
	for i, line := range doc.Lines {
		if fencedMask[i] || indentMask[i] {
			continue
		}
		trimmed := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(trimmed, "[^") {
			continue
		}
		// Find the colon after the label: [^label]:
		labelEnd := strings.Index(trimmed, "]:")
		if labelEnd < 0 {
			continue
		}
		// Strip inline links to avoid flagging URLs already inside [text](url).
		rest := strings.TrimSpace(trimmed[labelEnd+2:])
		rest = inlineLinkRE.ReplaceAllString(rest, "")
		for _, m := range bareURLRE.FindAllString(rest, -1) {
			addViolation(i+1, m)
		}
	}

	return violations
}
