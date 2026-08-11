package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD036 checks that emphasis is not used instead of a heading.
type MD036 struct {
	// Punctuation is the set of characters that, if a paragraph ends with one,
	// the emphasis check is skipped. Default: ".,;:!?。，；：！？"
	Punctuation string `json:"punctuation"`
}

func (r MD036) ID() string          { return "MD036" }
func (r MD036) Aliases() []string   { return []string{"no-emphasis-as-heading"} }
func (r MD036) Description() string { return "Emphasis used instead of a heading" }

const defaultMD036Punctuation = ".,;:!?。，；：！？"

func (r MD036) punct() string {
	if r.Punctuation == "" {
		return defaultMD036Punctuation
	}
	return r.Punctuation
}

// md036EmphasisText tries to extract the inner text of a line that consists
// entirely of a single emphasis span (*text*, **text**, _text_, or __text__).
// Returns ("", 0, false) if the line does not match.
func md036EmphasisText(line string) (text string, level int, ok bool) {
	if len(line) == 0 {
		return "", 0, false
	}
	marker := line[0]
	if marker != '*' && marker != '_' {
		return "", 0, false
	}
	count := 0
	for count < len(line) && line[count] == marker {
		count++
	}
	if count > 2 {
		return "", 0, false
	}
	// Line must end with the same marker repeated the same number of times.
	if len(line) < count*2+1 {
		return "", 0, false
	}
	for k := 0; k < count; k++ {
		if line[len(line)-1-k] != marker {
			return "", 0, false
		}
	}
	inner := line[count : len(line)-count]
	// Inner text must not start or end with the marker itself.
	if len(inner) == 0 || inner[0] == marker || inner[len(inner)-1] == marker {
		return "", 0, false
	}
	return inner, count, true
}

// Fix converts emphasis-only paragraphs that would trigger MD036 to ATX
// headings. Bold (**text**) and italic (*text*) are both converted to ## headings.
// Only top-level lines (no leading spaces) outside fenced code blocks are fixed;
// emphasis inside blockquotes, lists, or multi-line paragraphs is left unchanged.
func (r MD036) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	punct := r.punct()
	changed := false

	for i, line := range lines {
		if mask[i] {
			continue
		}

		// Only handle lines with no leading spaces (top-level paragraphs).
		// Lines inside lists or blockquotes will have leading spaces or '>' prefix.
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' || line[0] == '>' || line[0] == '#' {
			continue
		}

		// The line must not look like a list item.
		rest := strings.TrimLeft(line, " ")
		if len(rest) >= 2 && (rest[0] == '-' || rest[0] == '*' || rest[0] == '+') && rest[1] == ' ' {
			continue
		}

		text, _, ok := md036EmphasisText(line)
		if !ok {
			continue
		}

		// Skip if text ends with punctuation.
		runes := []rune(text)
		if len(runes) == 0 || strings.ContainsRune(punct, runes[len(runes)-1]) {
			continue
		}

		lines[i] = "## " + text
		changed = true
	}

	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD036) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	punct := r.punct()

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		para, ok := n.(*ast.Paragraph)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Only check top-level paragraphs (not inside lists, list items, etc.)
		// to match markdownlint behavior.
		parent := para.Parent()
		for parent != nil {
			switch parent.(type) {
			case *ast.ListItem, *ast.List, *ast.Blockquote:
				return ast.WalkContinue, nil
			}
			parent = parent.Parent()
		}

		// The paragraph must consist of a single emphasis node (no other children).
		first := para.FirstChild()
		if first == nil || first.NextSibling() != nil {
			return ast.WalkContinue, nil
		}
		emph, ok := first.(*ast.Emphasis)
		if !ok {
			return ast.WalkContinue, nil
		}

		// The emphasis must consist of exactly one plain text child (no code spans,
		// nested emphasis, line breaks, etc.). This matches markdownlint's requirement
		// that the emphasis text token has exactly one "data" child.
		emphChild := emph.FirstChild()
		if emphChild == nil || emphChild.NextSibling() != nil {
			return ast.WalkContinue, nil
		}
		if _, ok := emphChild.(*ast.Text); !ok {
			return ast.WalkContinue, nil
		}

		// Get the text content of the emphasis node.
		text := headingText(emph, doc.Source)
		if text == "" {
			return ast.WalkContinue, nil
		}

		// Skip if ends with punctuation.
		runes := []rune(text)
		lastRune := runes[len(runes)-1]
		if strings.ContainsRune(punct, lastRune) {
			return ast.WalkContinue, nil
		}

		line := 1
		if para.Lines() != nil && para.Lines().Len() > 0 {
			seg := para.Lines().At(0)
			line = doc.LineAt(seg.Start)
		}
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    line,
			Column:  1,
			Message: "Emphasis used instead of a heading",
		})
		return ast.WalkContinue, nil
	})

	return violations
}
