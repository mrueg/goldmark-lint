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
func (r MD029) Aliases() []string   { return []string{"ol-prefix"} }
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

// listItemFirstSeg returns the first text segment of a list item by recursively
// searching the AST subtree, but NOT descending into sub-lists. This ensures we
// get the segment of the item's own content (not nested list content).
// Returns (segStart, ok).
func listItemFirstSeg(li ast.Node) (int, bool) {
	if li.Lines() != nil && li.Lines().Len() > 0 {
		return li.Lines().At(0).Start, true
	}
	for c := li.FirstChild(); c != nil; c = c.NextSibling() {
		// Skip sub-lists to avoid mixing up their item numbers with ours.
		if _, isList := c.(*ast.List); isList {
			continue
		}
		if c.Lines() != nil && c.Lines().Len() > 0 {
			return c.Lines().At(0).Start, true
		}
		// Recurse into non-list block children (e.g., blockquotes containing the item).
		if s, ok := listItemFirstSeg(c); ok {
			return s, true
		}
	}
	return 0, false
}

// listItemNumFromSeg extracts the ordered list item number by scanning backward
// in the source from segStart to find the "N." or "N)" marker on the same line.
// Returns -1 if the number cannot be determined.
func listItemNumFromSeg(source []byte, segStart int) int {
	// Step back past the mandatory space after the separator.
	i := segStart - 1
	if i < 0 || source[i] != ' ' {
		return -1
	}
	i--
	// Skip the separator ('.' or ')').
	if i < 0 || (source[i] != '.' && source[i] != ')') {
		return -1
	}
	i--
	// Collect digit(s).
	end := i + 1 // exclusive end of digit run
	for i >= 0 && source[i] >= '0' && source[i] <= '9' {
		i--
	}
	if end == i+1 {
		return -1 // no digits found
	}
	n, err := strconv.Atoi(string(source[i+1 : end]))
	if err != nil {
		return -1
	}
	return n
}

// Check validates ordered list item numbering style.
func (r MD029) Check(doc *lint.Document) []lint.Violation {
	style := r.Style
	if style == "" {
		style = "one_or_ordered"
	}

	var violations []lint.Violation

	// lastListEnd tracks, for each parent node and "one_or_ordered" style, the
	// last-item number, item-count, and list node of the most recent multi-item
	// consecutive ordered list. This lets us detect continuation fragments: when
	// the parser splits a sequential list at a link definition, the second
	// fragment starts at N>1 and follows a previous multi-item list that ended
	// at N-1, with no heading in between.
	type listState struct {
		lastNum int
		count   int
		node    ast.Node
	}
	lastListEnd := map[ast.Node]listState{}

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
			lineNum := -1
			num := -1
			if segStart, found := listItemFirstSeg(li); found {
				lineNum = countLine(doc.Source, segStart)
				// First try fast backward scan from the segment start.
				num = listItemNumFromSeg(doc.Source, segStart)
			}
			// Fallback: parse the number from the raw source line using regex.
			// This handles cases where segStart is inside a blockquote
			// (e.g. "> 1. item") and the backward scan can't find the number.
			if num < 0 && lineNum >= 1 && lineNum <= len(doc.Lines) {
				line := doc.Lines[lineNum-1]
				// Strip blockquote prefixes before matching.
				for {
					stripped := strings.TrimLeft(line, " ")
					if len(stripped) == 0 || stripped[0] != '>' {
						break
					}
					line = stripped[1:]
					if len(line) > 0 && line[0] == ' ' {
						line = line[1:]
					}
				}
				if m := orderedItemRE.FindStringSubmatch(line); m != nil {
					if n2, err := strconv.Atoi(m[2]); err == nil {
						num = n2
					}
				}
			}
			if lineNum <= 0 || num < 0 {
				// Cannot determine line or number; skip to avoid false positives.
				continue
			}
			items = append(items, item{lineNum, num})
		}

		if len(items) == 0 {
			return ast.WalkContinue, nil
		}

		// Determine what style is used in this list.
		allOne := true
		allZero := true
		for _, it := range items {
			if it.number != 1 {
				allOne = false
			}
			if it.number != 0 {
				allZero = false
			}
			if !allOne && !allZero {
				break
			}
		}
		// sequentialFrom1: items are 1, 2, 3, ... (used for "ordered" style).
		sequentialFrom1 := true
		for i, it := range items {
			if it.number != i+1 {
				sequentialFrom1 = false
				break
			}
		}
		// sequentialFromFirst: items form a consecutive sequence starting from
		// items[0], where the starting value is 0 or 1 (used for "one_or_ordered"
		// style; markdownlint treats 0-based and 1-based sequences as valid).
		sequentialFromFirst := items[0].number == 0 || items[0].number == 1
		for i, it := range items {
			if it.number != items[0].number+i {
				sequentialFromFirst = false
				break
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
			if !sequentialFrom1 {
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
			// Valid if all items are the same number, or items form a consecutive
			// sequence starting at 0 or 1.
			if allOne || sequentialFromFirst {
				// Record this valid sequential list so the next fragment (if any)
				// can be identified as a continuation.
				if sequentialFromFirst && len(items) > 1 {
					lastListEnd[list.Parent()] = listState{items[len(items)-1].number, len(items), list}
				}
				break
			}
			first := items[0].number
			if first > 1 {
				// Check if this is a parser-split continuation of a previous
				// multi-item sequential list at the same parent level.
				// Conditions: both lists have > 1 items, items are consecutive,
				// AND there is no heading between the two lists (headings break
				// list context; link definitions / indented paragraphs do not).
				parent := list.Parent()
				if len(items) > 1 {
					if prev, ok := lastListEnd[parent]; ok &&
						prev.count > 1 &&
						first == prev.lastNum+1 {
						// Check this list is itself consecutive.
						consecutive := true
						for i := 1; i < len(items); i++ {
							if items[i].number != items[i-1].number+1 {
								consecutive = false
								break
							}
						}
						// Check there is no heading OR paragraph between prev.node
						// and this list. Headings break list context; regular paragraphs
						// indicate a clear list termination. Link definitions, indented
						// code blocks (CodeBlock/TextBlock), and similar "orphaned"
						// list-continuation content may appear without terminating the
						// logical list (depending on the parser).
						noBreakBetween := consecutive
						if noBreakBetween {
							for sib := prev.node.NextSibling(); sib != nil; sib = sib.NextSibling() {
								if sib == list {
									break
								}
								switch sib.(type) {
								case *ast.Heading, *ast.Paragraph:
									noBreakBetween = false
								}
								if !noBreakBetween {
									break
								}
							}
						}
						if noBreakBetween {
							// Treat as continuation; update state and skip.
							lastListEnd[parent] = listState{items[len(items)-1].number, len(items), list}
							return ast.WalkContinue, nil
						}
					}
				}
				// List starts at non-0, non-1: expect sequential from 1.
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
			} else if len(items) >= 2 && items[1].number == first {
				// First two items are the same: "all same" style; flag deviations.
				for _, it := range items {
					if it.number != first {
						violations = append(violations, lint.Violation{
							Rule:    r.ID(),
							Line:    it.line,
							Column:  1,
							Message: fmt.Sprintf("Ordered list item prefix [Expected: %d; Actual: %d]", first, it.number),
						})
					}
				}
			} else {
				// Sequential from first (0 or 1) but items don't match: flag.
				for i, it := range items {
					expected := first + i
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
		}

		return ast.WalkContinue, nil
	})

	return violations
}
