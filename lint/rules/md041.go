package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD041 checks that the first line in a file is a top-level heading.
type MD041 struct {
	// Level is the required top-level heading level (default 1).
	Level int `json:"level"`
	// FrontMatterTitle is a field name or regex pattern used to identify a
	// title in YAML front matter that satisfies the top-level heading requirement.
	// If empty, "title" is used. Set to "^$" to disable.
	FrontMatterTitle string `json:"front_matter_title"`
	// AllowPreamble controls whether some non-heading content is allowed before
	// the first top-level heading (default false). When true, the rule checks
	// that the document contains a top-level heading somewhere, not necessarily
	// as the first non-blank content.
	AllowPreamble bool `json:"allow_preamble"`
}

func (r MD041) ID() string          { return "MD041" }
func (r MD041) Description() string { return "First line in a file should be a top-level heading" }

func (r MD041) Check(doc *lint.Document) []lint.Violation {
	level := r.Level
	if level == 0 {
		level = 1
	}
	if len(doc.Lines) == 0 {
		return nil
	}

	// If the front matter contains a title, the requirement is satisfied.
	if frontMatterHasTitle(doc, r.FrontMatterTitle) {
		return nil
	}

	prefix := strings.Repeat("#", level) + " "

	if r.AllowPreamble {
		// When preamble is allowed, check that the document contains a top-level
		// heading anywhere.
		for _, line := range doc.Lines {
			if strings.HasPrefix(line, prefix) {
				return nil
			}
		}
		return []lint.Violation{{
			Rule:    r.ID(),
			Line:    1,
			Column:  1,
			Message: "First line in a file should be a top-level heading",
		}}
	}

	for i, line := range doc.Lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			return nil
		}
		return []lint.Violation{{
			Rule:    r.ID(),
			Line:    i + 1,
			Column:  1,
			Message: "First line in a file should be a top-level heading",
		}}
	}
	return nil
}
