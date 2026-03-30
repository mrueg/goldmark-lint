package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
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
	// Re-parse source to get an AST for accurate detection.
	pctx := parser.NewContext()
	reader := text.NewReader(source)
	md := goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough, extension.TaskList, extension.CJK))
	node := md.Parser().Parse(reader, parser.WithContext(pctx))

	lines := strings.Split(string(source), "\n")
	doc := &lint.Document{
		Source: source,
		Lines:  lines,
		AST:    node,
		// Note: We don't have FrontMatterFields here because Fix is called on
		// the part of the source AFTER the front matter.
		// However, if the user provided FrontMatterTitle, they expect it to
		// be used. But since Fix is called by Linter.Fix which already
		// stripped front matter, doc.AST starts AFTER front matter.
		// This means prevLevel should always start at 0 (or 1 if we're in a
		// sub-task where we know the front matter title was there).
		//
		// WAIT: Linter.Fix calls rule.Fix(source[fmEnd:]).
		// So the 'source' passed to MD001.Fix is already stripped.
		// BUT the user's config for FrontMatterTitle is about the ORIGINAL
		// document's front matter.
		//
		// If the original document had a title, the first heading in the
		// remaining source should be compared against prevLevel=1.
		// But how does rule.Fix know if the stripped front matter had a title?
		//
		// This suggests a flaw in the FixableRule interface if it needs
		// document-level context like front matter.
	}

	// For now, let's at least make it AST-aware for the increments within
	// the stripped source.
	
	type fix struct {
		lineNum  int
		newLevel int
	}
	var fixes []fix

	r.run(doc, func(lineNum, expectedLevel, actualLevel int) {
		fixes = append(fixes, fix{lineNum, expectedLevel})
	})

	if len(fixes) == 0 {
		return source
	}

	changed := false
	// Track level changes to adjust subsequent headings if needed.
	// Actually, the run() method already calculates the expected level
	// based on the (hypothetically) fixed previous levels.
	for _, f := range fixes {
		lineIdx := f.lineNum - 1
		if lineIdx < 0 || lineIdx >= len(lines) {
			continue
		}
		line := lines[lineIdx]
		stripped := strings.TrimLeft(line, " ")
		j := 0
		for j < len(stripped) && stripped[j] == '#' {
			j++
		}
		// Only fix ATX headings (Setext headings don't have increments > 1
		// because they are only H1/H2).
		if j > 0 && j <= 6 {
			leadingSpaces := len(line) - len(stripped)
			lines[lineIdx] = line[:leadingSpaces] + strings.Repeat("#", f.newLevel) + stripped[j:]
			changed = true
		}
	}

	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD001) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	r.run(doc, func(lineNum, expectedLevel, actualLevel int) {
		violations = append(violations, lint.Violation{
			Rule:    r.ID(),
			Line:    lineNum,
			Column:  1,
			Message: fmt.Sprintf("Heading levels should only increment by one level at a time [Expected: h%d; Actual: h%d]", expectedLevel, actualLevel),
		})
	})
	return violations
}

func (r MD001) run(doc *lint.Document, onViolation func(lineNum, expectedLevel, actualLevel int)) {
	prevLevel := 0
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
			onViolation(line, prevLevel+1, level)
			level = prevLevel + 1
		}
		prevLevel = level
		return ast.WalkContinue, nil
	})
}
