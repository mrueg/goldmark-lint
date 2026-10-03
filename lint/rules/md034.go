package rules

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/v2/ast"
)

// MD034 checks for bare URLs that are not wrapped in angle brackets or a proper link.
type MD034 struct{}

func (r MD034) ID() string          { return "MD034" }
func (r MD034) Aliases() []string   { return []string{"no-bare-urls"} }
func (r MD034) Description() string { return "Bare URL used" }

// bareURLRE matches an http/https URL or a www. URL within a string, stopping
// at whitespace or common punctuation characters unlikely to be part of the URL.
var bareURLRE = regexp.MustCompile(`(?:https?://|www\.)[^\s<>()\[\]{}'"` + "`\uff08\uff09" + `]+`)

// bareEmailRE matches a bare email address not wrapped in angle brackets.
var bareEmailRE = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// inlineLinkRE matches inline markdown links [text](url) for stripping from scanned content.
var inlineLinkRE = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`)

func (r MD034) Fix(source []byte) []byte {
	// Re-parse source to get an AST for accurate detection.
	docAST := lint.Parse(source)

	lines := strings.Split(string(source), "\n")
	doc := &lint.Document{
		Source: source,
		Lines:  lines,
		AST:    docAST,
	}

	fencedMask := fencedCodeBlockMask(lines)
	indentMask := indentedCodeBlockMask(doc)

	changed := false
	for i, line := range lines {
		if fencedMask[i] || indentMask[i] {
			continue
		}

		// Skip indented code block lines.
		if i > 0 && strings.TrimSpace(lines[i-1]) == "" {
			if len(line) >= 4 && line[0] == ' ' && line[1] == ' ' && line[2] == ' ' && line[3] == ' ' {
				continue
			}
		}

		// Skip link reference definitions.
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "[") {
			// [label]: url
			if idx := strings.Index(trimmed, "]:"); idx > 0 {
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

		var newLine strings.Builder
		prev := 0
		lineChanged := false
		for _, loc := range matches {
			start, end := loc[0], loc[1]
			if start > 0 {
				ch := line[start-1]
				if ch == '<' || ch == '(' || ch == '"' || ch == '\'' {
					newLine.WriteString(line[prev:end])
					prev = end
					continue
				}
			}
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
	r.run(doc, func(lineNum int, url string) {
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    lineNum,
			Column:  1,
			Message: "Bare URL used",
		})
	})
	return violations
}

func (r MD034) run(doc *lint.Document, onViolation func(lineNum int, url string)) {
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
		onViolation(lineNum, url)
	}

	// A URL that starts inside an earlier one's autolink run (common in CJK
	// text, which has no spaces) is part of it rather than a separate bare URL.
	urlRunEnd := 0
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		t, ok := n.(*ast.Text)
		if !ok {
			return ast.WalkContinue, nil
		}

		for p := t.Parent(); p != nil; p = p.Parent() {
			switch p.(type) {
			case *ast.Link, *ast.Image, *ast.CodeSpan:
				return ast.WalkContinue, nil
			}
		}

		seg := textSeg(t)
		text := string(doc.Source[seg.Start:seg.Stop])
		lineBase := doc.LineAt(seg.Start)

		for _, loc := range bareURLRE.FindAllStringIndex(text, -1) {
			lineNum := lineBase + strings.Count(text[:loc[0]], "\n")
			if loc[0] > 0 && text[loc[0]-1] == '(' {
				srcParenPos := seg.Start + loc[0] - 1
				lineStartInSrc := srcParenPos
				for lineStartInSrc > 0 && doc.Source[lineStartInSrc-1] != '\n' {
					lineStartInSrc--
				}
				depth := 0
				for _, b := range doc.Source[lineStartInSrc:srcParenPos] {
					if b == '[' {
						depth++
					} else if b == ']' && depth > 0 {
						depth--
					}
				}
				if depth > 0 {
					continue
				}
			}
			start := seg.Start + loc[0]
			if start < urlRunEnd {
				continue
			}
			urlRunEnd = autolinkRunEnd(doc.Source, start)
			addViolation(lineNum, text[loc[0]:loc[1]])
		}

		for _, loc := range bareEmailRE.FindAllStringIndex(text, -1) {
			lineNum := lineBase + strings.Count(text[:loc[0]], "\n")
			if loc[0] > 0 && text[loc[0]-1] == '<' {
				continue
			}
			addViolation(lineNum, text[loc[0]:loc[1]])
		}
		return ast.WalkContinue, nil
	})

	fencedMask := fencedContentMask(doc)
	indentMask := indentedCodeBlockMask(doc)
	for i, line := range doc.Lines {
		if fencedMask[i] || indentMask[i] {
			continue
		}
		trimmed := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(trimmed, "[^") {
			continue
		}
		labelEnd := strings.Index(trimmed, "]:")
		if labelEnd < 0 {
			continue
		}
		rest := strings.TrimSpace(trimmed[labelEnd+2:])
		rest = inlineLinkRE.ReplaceAllString(rest, "")
		for _, m := range bareURLRE.FindAllString(rest, -1) {
			addViolation(i+1, m)
		}
	}
}

// autolinkRunEnd returns the offset in source where a GFM autolink literal
// starting at start can extend to at most: the first whitespace, '<', or ']'
// followed by '(' or '[', or the end of the source.
func autolinkRunEnd(source []byte, start int) int {
	for i := start; i < len(source); {
		r, size := utf8.DecodeRune(source[i:])
		if unicode.IsSpace(r) || r == '<' {
			return i
		}
		if r == ']' && i+1 < len(source) && (source[i+1] == '(' || source[i+1] == '[') {
			return i
		}
		i += size
	}
	return len(source)
}
