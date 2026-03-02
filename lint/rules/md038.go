package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD038 checks for spaces inside code span elements.
type MD038 struct{}

func (r MD038) ID() string          { return "MD038" }
func (r MD038) Aliases() []string   { return []string{"no-space-in-code"} }
func (r MD038) Description() string { return "Spaces inside code span elements" }

// fixCodeSpanSpaces removes leading/trailing spaces from code span content.
func fixCodeSpanSpaces(line string) string {
	result := []byte(line)
	i := 0
	for i < len(result) {
		if result[i] != '`' {
			i++
			continue
		}
		start := i
		for i < len(result) && result[i] == '`' {
			i++
		}
		tickLen := i - start
		contentStart := i
		end := i
		for end < len(result) {
			if result[end] == '`' {
				k := end
				for k < len(result) && result[k] == '`' {
					k++
				}
				if k-end == tickLen {
					content := string(result[contentStart:end])
					trimmed := strings.TrimSpace(content)
					if trimmed != content && len(trimmed) > 0 {
						// Rebuild this code span without leading/trailing spaces
						newSpan := string(result[start:contentStart]) + trimmed + string(result[end:k])
						result = append(result[:start], append([]byte(newSpan), result[k:]...)...)
						i = start + tickLen + len(trimmed) + tickLen
					} else {
						i = k
					}
					break
				}
				end = k
			} else {
				end++
			}
		}
	}
	return string(result)
}

func (r MD038) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	for i, line := range lines {
		if !mask[i] {
			lines[i] = fixCodeSpanSpaces(line)
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD038) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		cs, ok := n.(*ast.CodeSpan)
		if !ok {
			return ast.WalkContinue, nil
		}

		// CodeSpan children are Text nodes (one per source line).
		first := cs.FirstChild()
		if first == nil {
			return ast.WalkContinue, nil
		}
		firstText, ok := first.(*ast.Text)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Find last text child.
		var lastText *ast.Text
		for c := cs.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				lastText = t
			}
		}
		if lastText == nil {
			return ast.WalkContinue, nil
		}

		firstContent := firstText.Segment.Value(doc.Source)
		lastContent := lastText.Segment.Value(doc.Source)

		// Check for leading space: in content or stripped by goldmark (symmetric stripping).
		hasLeadingSpace := (len(firstContent) > 0 && firstContent[0] == ' ') ||
			(firstText.Segment.Start > 0 && firstText.Segment.Start <= len(doc.Source) && doc.Source[firstText.Segment.Start-1] == ' ')

		// Check for trailing space: in content or stripped by goldmark.
		hasTrailingSpace := (len(lastContent) > 0 && lastContent[len(lastContent)-1] == ' ') ||
			(lastText.Segment.Stop < len(doc.Source) && doc.Source[lastText.Segment.Stop] == ' ')

		if !hasLeadingSpace && !hasTrailingSpace {
			return ast.WalkContinue, nil
		}

		// Only flag leading space if there is non-whitespace content after it.
		// This avoids false positives for space-only code spans like ` ` or `   `.
		if hasLeadingSpace && strings.TrimLeft(string(firstContent), " ") == "" {
			hasLeadingSpace = false
		}
		// Only flag trailing space if there is non-whitespace content before it.
		if hasTrailingSpace && strings.TrimRight(string(lastContent), " ") == "" {
			hasTrailingSpace = false
		}

		if !hasLeadingSpace && !hasTrailingSpace {
			return ast.WalkContinue, nil
		}

		line := countLine(doc.Source, firstText.Segment.Start)
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    line,
			Column:  1,
			Message: "Spaces inside code span elements",
		})
		return ast.WalkContinue, nil
	})

	return violations
}
