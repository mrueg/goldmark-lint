package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD059 checks that link text is descriptive (not generic).
type MD059 struct {
	// ProhibitedTexts is a list of generic link text phrases (default ["click here","here","link","more"]).
	ProhibitedTexts []string `json:"prohibited_texts"`
}

func (r MD059) ID() string          { return "MD059" }
func (r MD059) Description() string { return "Link text should be descriptive" }

func (r MD059) prohibited() []string {
	if len(r.ProhibitedTexts) == 0 {
		return []string{"click here", "here", "link", "more"}
	}
	return r.ProhibitedTexts
}

// md059LinkTextRE matches inline link text: [text](url).
var md059LinkTextRE = regexp.MustCompile(`\[([^\]]+)\]\(`)

func (r MD059) Check(doc *lint.Document) []lint.Violation {
	mask := fencedCodeBlockMask(doc.Lines)
	prohibited := r.prohibited()
	var violations []lint.Violation

	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		for _, m := range md059LinkTextRE.FindAllStringSubmatch(line, -1) {
			text := strings.TrimSpace(m[1])
			for _, p := range prohibited {
				if strings.EqualFold(text, p) {
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    i + 1,
						Column:  1,
						Message: "Link text should be descriptive [Text: " + text + "]",
					})
					break
				}
			}
		}
	}
	return violations
}
