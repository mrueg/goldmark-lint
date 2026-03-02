package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD039 checks for spaces inside link text.
type MD039 struct{}

func (r MD039) ID() string          { return "MD039" }
func (r MD039) Aliases() []string   { return []string{"no-space-in-links"} }
func (r MD039) Description() string { return "Spaces inside link text" }

// md039RE matches a link with leading or trailing space in its text.
// Captures: full match includes [<space>content](  or  [content<space>](
var md039RE = regexp.MustCompile(`\[(\s[^\]\n]*|[^\]\n]*\s)\]\(`)

func (r MD039) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if mask[i] {
			continue
		}
		lines[i] = md039RE.ReplaceAllStringFunc(line, func(match string) string {
			// match is like "[ text ](" - trim spaces from inner content
			inner := match[1 : len(match)-2] // content between [ and ](
			trimmed := strings.TrimSpace(inner)
			return "[" + trimmed + "]("
		})
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD039) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Only check inline links (not reference links).
		// For inline links, source[lastTextStop] == ']' and source[lastTextStop+1] == '('.
		var lastTextStop int
		for c := link.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok2 := c.(*ast.Text); ok2 && t.Segment.Stop > lastTextStop {
				lastTextStop = t.Segment.Stop
			}
		}
		isInline := lastTextStop > 0 && lastTextStop < len(doc.Source)-1 &&
			doc.Source[lastTextStop] == ']' && doc.Source[lastTextStop+1] == '('
		if !isInline {
			return ast.WalkContinue, nil
		}

		// Check first and last text children for leading/trailing spaces.
		// The first link child may be a non-Text node (e.g. a CodeSpan), in which case
		// the first Text child is not the first child of the link. We need to distinguish:
		// - leading space: first child is Text AND starts with ' '
		// - trailing space: last Text child ends with ' '
		var firstText *ast.Text
		var lastText *ast.Text
		for c := link.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				if firstText == nil {
					firstText = t
				}
				lastText = t
			}
		}
		if firstText == nil || lastText == nil {
			return ast.WalkContinue, nil
		}

		firstContent := firstText.Segment.Value(doc.Source)
		lastContent := lastText.Segment.Value(doc.Source)

		// Leading space: only if the first child of the link is the Text node
		// (i.e. the link text starts with a space).
		hasLeadingSpace := link.FirstChild() == firstText &&
			len(firstContent) > 0 && firstContent[0] == ' '
		// Trailing space: the last text child of the link ends with a space.
		hasTrailingSpace := len(lastContent) > 0 && lastContent[len(lastContent)-1] == ' '

		if !hasLeadingSpace && !hasTrailingSpace {
			return ast.WalkContinue, nil
		}

		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    inlineNodeLine(link, doc.Source),
			Column:  1,
			Message: "Spaces inside link text",
		})
		return ast.WalkContinue, nil
	})

	return violations
}
