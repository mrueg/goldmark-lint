package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD001 checks that heading levels only increment by one level at a time.
type MD001 struct {
	// FrontMatterTitle is a field name or regex pattern used to identify a
	// title in YAML front matter that counts as an h1 heading. If empty,
	// "title" is used. Set to an empty string to disable (use "^$").
	FrontMatterTitle string `json:"front_matter_title"`
}

func (r MD001) ID() string          { return "MD001" }
func (r MD001) Aliases() []string   { return []string{"heading-increment"} }
func (r MD001) Description() string {
	return "Heading levels should only increment by one level at a time"
}

// Fix applies MD001 to source by reducing heading levels that skip more than
// one level. For example, if an h1 is followed by an h3, the h3 is reduced
// to h2. Subsequent headings are also adjusted to maintain valid increments.
func (r MD001) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)

	// Determine if front matter counts as h1.
	prevLevel := 0
	// We process front matter in the Linter, so here we just handle the raw source.
	// Since Fix is called on the non-front-matter portion, we start at prevLevel=0.
	// However, to match the Check behavior with FrontMatterTitle, we can't easily
	// detect front matter here. We'll start fresh and not assume front matter title.

	changed := false
	for i, line := range lines {
		if mask[i] {
			continue
		}
		// Detect ATX heading.
		stripped := strings.TrimLeft(line, " ")
		if len(stripped) == 0 || stripped[0] != '#' {
			continue
		}
		j := 0
		for j < len(stripped) && stripped[j] == '#' {
			j++
		}
		if j > 6 {
			continue
		}
		if j < len(stripped) && stripped[j] != ' ' {
			continue
		}
		level := j

		if prevLevel > 0 && level > prevLevel+1 {
			// Reduce this heading level to prevLevel+1.
			newLevel := prevLevel + 1
			// Find the position of the '#' block in the original line.
			leadingSpaces := len(line) - len(stripped)
			newLine := line[:leadingSpaces] + strings.Repeat("#", newLevel) + stripped[j:]
			lines[i] = newLine
			level = newLevel
			changed = true
		}
		prevLevel = level
	}

	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD001) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	prevLevel := 0

	// If the front matter contains a title, treat it as an h1.
	if frontMatterHasTitle(doc, r.FrontMatterTitle) {
		prevLevel = 1
	}

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		level := h.Level
		if prevLevel > 0 && level > prevLevel+1 {
			line := 1
			if h.Lines() != nil && h.Lines().Len() > 0 {
				seg := h.Lines().At(0)
				line = countLine(doc.Source, seg.Start)
			}
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    line,
				Column:  1,
				Message: fmt.Sprintf("Heading levels should only increment by one level at a time [Expected: h%d; Actual: h%d]", prevLevel+1, level),
			})
		}
		prevLevel = level
		return ast.WalkContinue, nil
	})

	return violations
}
