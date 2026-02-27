package rules

import (
	"github.com/mrueg/goldmark-lint/lint"
)

// MD047 checks that files end with a single newline character.
type MD047 struct{}

func (r MD047) ID() string          { return "MD047" }
func (r MD047) Description() string { return "Files should end with a single newline character" }

func (r MD047) Check(doc *lint.Document) []lint.Violation {
	if len(doc.Source) == 0 {
		return nil
	}
	last := doc.Source[len(doc.Source)-1]
	if last != '\n' {
		return []lint.Violation{{
			Rule:    r.ID(),
			Line:    len(doc.Lines),
			Column:  len(doc.Lines[len(doc.Lines)-1]) + 1,
			Message: "Files should end with a single newline character",
		}}
	}
	return nil
}
