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
	for i, line := range doc.Lines {
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
