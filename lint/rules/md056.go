package rules

import (
	"fmt"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD056 checks that all rows in a table have the same number of cells.
type MD056 struct{}

func (r MD056) ID() string          { return "MD056" }
func (r MD056) Description() string { return "Table column count" }

func (r MD056) Check(doc *lint.Document) []lint.Violation {
	mask := fencedCodeBlockMask(doc.Lines)
	tables := findTables(doc.Lines, mask)
	var violations []lint.Violation

	for _, t := range tables {
		headerCells := countTableCells(doc.Lines[t[0]])
		for row := t[0] + 1; row <= t[1]; row++ {
			line := doc.Lines[row]
			// Skip delimiter row.
			if isTableDelimiterRow(line) {
				continue
			}
			actual := countTableCells(line)
			if actual != headerCells {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    row + 1,
					Column:  1,
					Message: fmt.Sprintf("Table column count [Expected: %d; Actual: %d]", headerCells, actual),
				})
			}
		}
	}
	return violations
}
