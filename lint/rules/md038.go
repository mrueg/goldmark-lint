package rules

import (
	"bytes"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/v2/ast"
)

// MD038 checks for spaces inside code span elements.
type MD038 struct{}

func (r MD038) ID() string          { return "MD038" }
func (r MD038) Aliases() []string   { return []string{"no-space-in-code"} }
func (r MD038) Description() string { return "Spaces inside code span elements" }

// leadingSpaceIsRequired reports whether the leading space of a code span's
// content must be kept. CommonMark needs a space between the opening delimiter
// and content that itself begins with a backtick, otherwise the backticks merge
// into the delimiter run and the span stops parsing as code.
func leadingSpaceIsRequired(content string) bool {
	return strings.HasPrefix(strings.TrimLeft(content, " "), "`")
}

// trailingSpaceIsRequired is the closing-delimiter counterpart of
// leadingSpaceIsRequired.
func trailingSpaceIsRequired(content string) bool {
	return strings.HasSuffix(strings.TrimRight(content, " "), "`")
}

// codeSpanEdges returns the content of the first and last source lines of a
// code span, without line endings. One space is stripped from both ends when
// the content, with line endings read as spaces, starts AND ends with a space,
// as CommonMark specifies. A line ending at either end of the span is not
// reported as a space.
func codeSpanEdges(cs *ast.CodeSpan, source []byte) (first, last []byte) {
	indices := cs.Value.Indices()
	if cs.Value.IsOwned() || len(indices) == 0 {
		// The value is not backed by the source (a code span with an escaped
		// pipe in a table cell); Str is already normalised.
		content := []byte(cs.Value.Str(source))
		return content, content
	}
	var raw []byte
	for _, idx := range indices {
		raw = append(raw, source[idx.Start:idx.Stop]...)
	}
	first = bytes.TrimRight(source[indices[0].Start:indices[0].Stop], "\r\n")
	lastIdx := indices[len(indices)-1]
	last = bytes.TrimRight(source[lastIdx.Start:lastIdx.Stop], "\r\n")
	isSpace := func(c byte) bool { return c == ' ' || c == '\r' || c == '\n' }
	if len(bytes.TrimSpace(raw)) > 0 && isSpace(raw[0]) && isSpace(raw[len(raw)-1]) {
		first = bytes.TrimPrefix(first, []byte(" "))
		last = bytes.TrimSuffix(last, []byte(" "))
	}
	return first, last
}

// trimCodeSpanContent strips the leading and trailing spaces of a code span's
// content, keeping any space that CommonMark requires. It returns content
// unchanged when nothing may safely be removed.
func trimCodeSpanContent(content string) string {
	if strings.TrimSpace(content) == "" {
		return content
	}
	out := content
	if strings.HasPrefix(out, " ") && !leadingSpaceIsRequired(out) {
		out = strings.TrimLeft(out, " ")
	}
	if strings.HasSuffix(out, " ") && !trailingSpaceIsRequired(out) {
		out = strings.TrimRight(out, " ")
	}
	return out
}

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
					trimmed := trimCodeSpanContent(content)
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

		if cs.Value.IsEmpty() {
			return ast.WalkContinue, nil
		}
		firstContent, lastContent := codeSpanEdges(cs, doc.Source)

		// Check for leading space in the normalised content.
		hasLeadingSpace := len(firstContent) > 0 && firstContent[0] == ' '

		// Check for trailing space: only check actual content.
		hasTrailingSpace := len(lastContent) > 0 && lastContent[len(lastContent)-1] == ' '

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
		// A space separating the delimiter from content that itself starts or
		// ends with a backtick is required by CommonMark, not stylistic: without
		// it the backticks merge into the delimiter run and the span no longer
		// parses as code. markdownlint does not report these either.
		if hasLeadingSpace && leadingSpaceIsRequired(string(firstContent)) {
			hasLeadingSpace = false
		}
		if hasTrailingSpace && trailingSpaceIsRequired(string(lastContent)) {
			hasTrailingSpace = false
		}

		if !hasLeadingSpace && !hasTrailingSpace {
			return ast.WalkContinue, nil
		}

		line := nodeLine(cs, doc)
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
