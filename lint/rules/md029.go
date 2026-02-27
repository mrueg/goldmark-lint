package rules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD029 checks that ordered list items use a consistent numbering style.
type MD029 struct {
	// Style is the required style: "one_or_ordered" (default), "one", "ordered", or "zero".
	Style string `json:"style"`
}

func (r MD029) ID() string          { return "MD029" }
func (r MD029) Description() string { return "Ordered list item prefix" }

// orderedItemRE matches an ordered list item prefix, capturing leading spaces,
// the number, and the separator character (. or )).
var orderedItemRE = regexp.MustCompile(`^( *)(\d+)([.)]) `)

// Fix rewrites ordered list item numbers to match the configured style.
func (r MD029) Fix(source []byte) []byte {
	style := r.Style
	if style == "" {
		style = "one_or_ordered"
	}

	lines := strings.Split(string(source), "\n")

	// listGroup tracks a contiguous sequence of ordered list items at the same indent level.
	type listGroup struct {
		indent  int
		indices []int // line indices
		numbers []int // original numbers
	}

	// applyGroup calculates expected numbers for a group and records fixes.
	applyGroup := func(g *listGroup, expected map[int]int) {
		if len(g.indices) == 0 {
			return
		}
		allOne := true
		sequential := true
		for i, n := range g.numbers {
			if n != 1 {
				allOne = false
			}
			if n != i+1 {
				sequential = false
			}
		}
		for i, lineIdx := range g.indices {
			var want int
			switch style {
			case "one":
				want = 1
			case "zero":
				want = 0
			case "ordered":
				want = i + 1
			case "one_or_ordered":
				if allOne || sequential {
					want = g.numbers[i] // already valid, no change
				} else {
					want = i + 1
				}
			}
			if want != g.numbers[i] {
				expected[lineIdx] = want
			}
		}
	}

	// Stack of active list groups at different indent levels.
	var stack []*listGroup
	expected := map[int]int{}

	for i, line := range lines {
		m := orderedItemRE.FindStringSubmatch(line)
		if m == nil {
			if strings.TrimSpace(line) == "" {
				continue // blank lines don't interrupt a list
			}
			// Non-blank, non-list line: if not indented, flush all groups.
			if strings.TrimLeft(line, " \t") == line {
				for _, g := range stack {
					applyGroup(g, expected)
				}
				stack = nil
			}
			continue
		}

		indent := len(m[1])
		num, _ := strconv.Atoi(m[2])

		// Pop groups with deeper indent (exiting sub-lists).
		for len(stack) > 0 && stack[len(stack)-1].indent > indent {
			g := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			applyGroup(g, expected)
		}

		// Reuse top group if same indent, else push a new one.
		if len(stack) > 0 && stack[len(stack)-1].indent == indent {
			g := stack[len(stack)-1]
			g.indices = append(g.indices, i)
			g.numbers = append(g.numbers, num)
		} else {
			stack = append(stack, &listGroup{
				indent:  indent,
				indices: []int{i},
				numbers: []int{num},
			})
		}
	}
	// Flush remaining groups.
	for _, g := range stack {
		applyGroup(g, expected)
	}

	// Apply fixes.
	for lineIdx, want := range expected {
		m := orderedItemRE.FindStringSubmatch(lines[lineIdx])
		if m == nil {
			continue
		}
		rest := lines[lineIdx][len(m[0]):]
		lines[lineIdx] = m[1] + strconv.Itoa(want) + m[3] + " " + rest
	}

	return []byte(strings.Join(lines, "\n"))
}

// Check validates ordered list item numbering style.
func (r MD029) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "one_or_ordered"
	}

	var violations []lint.Violation

	// Walk AST ordered lists and check each list independently.
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		list, ok := n.(*ast.List)
		if !ok || !list.IsOrdered() {
			return ast.WalkContinue, nil
		}

		// Collect line numbers and prefix numbers for each list item.
		type item struct {
			line   int
			number int
		}
		var items []item

		for child := list.FirstChild(); child != nil; child = child.NextSibling() {
			li, ok2 := child.(*ast.ListItem)
			if !ok2 {
				continue
			}
			// Find the source line for this list item.
			lineNum := 1
			num := -1
			if li.Lines() != nil && li.Lines().Len() > 0 {
				seg := li.Lines().At(0)
				lineNum = countLine(doc.Source, seg.Start)
			} else {
				// For list items with block children, look at the first child's lines.
				if fc := li.FirstChild(); fc != nil {
					if fc.Lines() != nil && fc.Lines().Len() > 0 {
						seg := fc.Lines().At(0)
						lineNum = countLine(doc.Source, seg.Start)
					}
				}
			}
			// Parse the actual number from the source line (1-based lineNum).
			if lineNum >= 1 && lineNum <= len(doc.Lines) {
				line := doc.Lines[lineNum-1]
				if m := orderedItemRE.FindStringSubmatch(line); m != nil {
					if n, err := strconv.Atoi(m[2]); err == nil {
						num = n
					}
				}
			}
			items = append(items, item{lineNum, num})
		}

		if len(items) == 0 {
			return ast.WalkContinue, nil
		}

		// Determine what style is used in this list.
		allOne := true
		allZero := true
		sequential := true
		for i, it := range items {
			if it.number != 1 {
				allOne = false
			}
			if it.number != 0 {
				allZero = false
			}
			if it.number != i+1 {
				sequential = false
			}
		}

		switch style {
		case "one":
			if !allOne {
				for _, it := range items {
					if it.number != 1 {
						violations = append(violations, lint.Violation{
							Rule:    r.ID(),
							Line:    it.line,
							Column:  1,
							Message: fmt.Sprintf("Ordered list item prefix [Expected: 1; Actual: %d]", it.number),
						})
					}
				}
			}
		case "zero":
			if !allZero {
				for _, it := range items {
					if it.number != 0 {
						violations = append(violations, lint.Violation{
							Rule:    r.ID(),
							Line:    it.line,
							Column:  1,
							Message: fmt.Sprintf("Ordered list item prefix [Expected: 0; Actual: %d]", it.number),
						})
					}
				}
			}
		case "ordered":
			if !sequential {
				for i, it := range items {
					expected := i + 1
					if it.number != expected {
						violations = append(violations, lint.Violation{
							Rule:    r.ID(),
							Line:    it.line,
							Column:  1,
							Message: fmt.Sprintf("Ordered list item prefix [Expected: %d; Actual: %d]", expected, it.number),
						})
					}
				}
			}
		case "one_or_ordered":
			if !allOne && !sequential {
				// Neither all-one nor sequential: report items that don't fit either.
				for i, it := range items {
					expected := i + 1
					if it.number != 1 && it.number != expected {
						violations = append(violations, lint.Violation{
							Rule:    r.ID(),
							Line:    it.line,
							Column:  1,
							Message: fmt.Sprintf("Ordered list item prefix [Expected: %d or 1; Actual: %d]", expected, it.number),
						})
					}
				}
			}
		}

		return ast.WalkContinue, nil
	})

	return violations
}
