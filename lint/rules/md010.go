package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD010 checks for hard tabs.
type MD010 struct {
	// SpacesPerTab is the number of spaces used to replace each tab when fixing (default 4).
	SpacesPerTab int `json:"spaces_per_tab"`
	// CodeBlocks controls whether hard tabs in fenced code blocks are checked (default true).
	CodeBlocks *bool `json:"code_blocks"`
	// IgnoreCodeLanguages is a list of fenced code block languages whose tabs are not checked.
	IgnoreCodeLanguages []string `json:"ignore_code_languages"`
}

func (r MD010) ID() string          { return "MD010" }
func (r MD010) Description() string { return "Hard tabs" }

func (r MD010) Fix(source []byte) []byte {
	spaces := r.SpacesPerTab
	if spaces <= 0 {
		spaces = 4
	}
	return []byte(strings.ReplaceAll(string(source), "\t", strings.Repeat(" ", spaces)))
}

func (r MD010) isIgnoredLang(lang string) bool {
	for _, l := range r.IgnoreCodeLanguages {
		if strings.EqualFold(l, lang) {
			return true
		}
	}
	return false
}

func (r MD010) Check(doc *lint.Document) []lint.Violation {
	checkCodeBlocks := r.CodeBlocks == nil || *r.CodeBlocks
	codeMask := fencedCodeBlockMask(doc.Lines)
	var langMap map[int]string
	if len(r.IgnoreCodeLanguages) > 0 {
		langMap = fencedCodeBlockLanguages(doc.Lines)
	}

	var violations []lint.Violation
	for i, line := range doc.Lines {
		if codeMask[i] {
			if !checkCodeBlocks {
				continue
			}
			if langMap != nil {
				if lang, ok := langMap[i]; ok && r.isIgnoredLang(lang) {
					continue
				}
			}
		}
		col := strings.Index(line, "\t")
		if col >= 0 {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    i + 1,
				Column:  col + 1,
				Message: "Hard tabs",
			})
		}
	}
	return violations
}
