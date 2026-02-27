package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD040 checks that fenced code blocks have a language specifier.
type MD040 struct {
	// AllowedLanguages is a list of allowed language identifiers. Empty means any language is allowed.
	AllowedLanguages []string `json:"allowed_languages"`
	// LanguageOnly requires the info string to be only a language identifier (no extra text).
	LanguageOnly bool `json:"language_only"`
}

func (r MD040) ID() string          { return "MD040" }
func (r MD040) Description() string { return "Fenced code blocks should have a language specified" }

// md040FenceRE matches a fenced code block opening line.
// Group 1: indent, Group 2: fence marker, Group 3: language (may be empty).
var md040FenceRE = regexp.MustCompile("^( {0,3})(`{3,}|~{3,})(.*)$")

// md040LangOnlyRE matches an info string consisting solely of a language identifier
// (letters, digits, underscores, hyphens, dots, plus signs).
var md040LangOnlyRE = regexp.MustCompile(`^\S+$`)

func (r MD040) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	lines := doc.Lines
	inFence := false
	fenceChar := byte(0)
	fenceLen := 0

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if !inFence {
			m := md040FenceRE.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			fc := trimmed[0]
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
			info := strings.TrimSpace(m[3])
			if info == "" {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  1,
					Message: "Fenced code blocks should have a language specified",
				})
			} else {
				lang := strings.Fields(info)[0]
				// Check allowed_languages.
				if len(r.AllowedLanguages) > 0 {
					allowed := false
					for _, al := range r.AllowedLanguages {
						if lang == al {
							allowed = true
							break
						}
					}
					if !allowed {
						violations = append(violations, lint.Violation{
							Rule:    r.ID(),
							Line:    i + 1,
							Column:  1,
							Message: "Fenced code blocks should use an allowed language",
						})
					}
				}
				// Check language_only: info string must contain no whitespace.
				if r.LanguageOnly && !md040LangOnlyRE.MatchString(info) {
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    i + 1,
						Column:  1,
						Message: "Fenced code blocks should only contain a language identifier",
					})
				}
			}
		} else {
			j := 0
			for j < len(trimmed) && trimmed[j] == fenceChar {
				j++
			}
			if j >= fenceLen && strings.TrimSpace(trimmed[j:]) == "" && len(trimmed) > 0 && trimmed[0] == fenceChar {
				inFence = false
			}
		}
	}
	return violations
}
