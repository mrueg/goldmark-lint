package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD040 checks that fenced code blocks have a language specifier.
type MD040 struct {
	// AllowedLanguages is a list of allowed language identifiers. Empty means any language is allowed.
	AllowedLanguages []string `json:"allowed_languages"`
	// LanguageOnly requires the info string to be only a language identifier (no extra text).
	LanguageOnly bool `json:"language_only"`
}

func (r MD040) ID() string          { return "MD040" }
func (r MD040) Aliases() []string   { return []string{"fenced-code-language"} }
func (r MD040) Description() string { return "Fenced code blocks should have a language specified" }

// Fix adds "text" as the language specifier to fenced code blocks that have no
// language specified. Blocks that already have a language (even a disallowed one)
// are not modified, and the language_only / allowed_languages constraints are not
// enforced by this fix.
func (r MD040) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")

	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	changed := false

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)

		if !inFence {
			if indent > 3 || len(trimmed) < 3 {
				continue
			}
			fc := trimmed[0]
			if fc != '`' && fc != '~' {
				continue
			}
			j := 0
			for j < len(trimmed) && trimmed[j] == fc {
				j++
			}
			if j < 3 {
				continue
			}
			inFence = true
			fenceChar = fc
			fenceLen = j

			info := strings.TrimSpace(trimmed[j:])
			if info == "" {
				// No language: insert "text".
				lines[i] = line[:indent] + strings.Repeat(string(fc), j) + " text"
				changed = true
			}
		} else {
			j := 0
			for j < len(trimmed) && trimmed[j] == fenceChar {
				j++
			}
			if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" && len(trimmed) > 0 {
				inFence = false
			}
		}
	}

	if !changed {
		return source
	}
	return []byte(strings.Join(lines, "\n"))
}

func (r MD040) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation

	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		fcb, ok := n.(*ast.FencedCodeBlock)
		if !ok {
			return ast.WalkContinue, nil
		}

		line := fencedCodeBlockLine(fcb, doc.Source)
		lang := fcb.Language(doc.Source)

		if len(lang) == 0 {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    line,
				Column:  1,
				Message: "Fenced code blocks should have a language specified",
			})
			return ast.WalkContinue, nil
		}

		// Check allowed_languages.
		if len(r.AllowedLanguages) > 0 {
			allowed := false
			for _, al := range r.AllowedLanguages {
				if string(lang) == al {
					allowed = true
					break
				}
			}
			if !allowed {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    line,
					Column:  1,
					Message: "Fenced code blocks should use an allowed language",
				})
			}
		}

		// Check language_only: info string must not contain whitespace after the language.
		if r.LanguageOnly && fcb.Info != nil {
			info := strings.TrimRight(string(fcb.Info.Segment.Value(doc.Source)), " \t\r\n")
			if info != string(lang) {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    line,
					Column:  1,
					Message: "Fenced code blocks should only contain a language identifier",
				})
			}
		}

		return ast.WalkContinue, nil
	})

	return violations
}
