package rules

import (
	"fmt"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD043 checks that headings match a required structure.
type MD043 struct {
	// Headings is the required list of heading texts. Empty = no check.
	Headings []string `json:"headings"`
	// MatchCase controls whether matching is case-sensitive (default false).
	MatchCase bool `json:"match_case"`
}

func (r MD043) ID() string          { return "MD043" }
func (r MD043) Aliases() []string   { return []string{"required-headings"} }
func (r MD043) Description() string { return "Required heading structure" }

func (r MD043) Check(doc *lint.Document) []lint.Violation {
	if len(r.Headings) == 0 {
		return nil
	}

	// Collect all headings from the document.
	var actual []struct {
		text string
		line int
	}
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		text := headingText(h, doc.Source)
		line := 1
		if h.Lines() != nil && h.Lines().Len() > 0 {
			seg := h.Lines().At(0)
			line = countLine(doc.Source, seg.Start)
		}
		// Include level prefix for matching: "# Heading", "## Heading", etc.
		levelPrefix := strings.Repeat("#", h.Level) + " "
		actual = append(actual, struct {
			text string
			line int
		}{levelPrefix + text, line})
		return ast.WalkContinue, nil
	})

	required := r.Headings
	matchCase := r.MatchCase

	// Match actual headings against required pattern.
	// * = zero or more unspecified, + = one or more unspecified, ? = exactly one unspecified.
	if headingsMatch(actual, required, matchCase) {
		return nil
	}

	// Report violation at first mismatch or end of document.
	line := len(doc.Lines)
	if len(actual) > 0 {
		// Find first mismatch.
		mismatchLine := findFirstMismatch(actual, required, matchCase)
		if mismatchLine > 0 {
			line = mismatchLine
		}
	}
	return []lint.Violation{{
		Rule:    r.ID(),
		Line:    line,
		Column:  1,
		Message: fmt.Sprintf("Required heading structure [Expected: %s]", strings.Join(required, ", ")),
	}}
}

// headingsMatch checks whether actual headings satisfy the required pattern.
func headingsMatch(actual []struct {
	text string
	line int
}, required []string, matchCase bool) bool {
	texts := make([]string, len(actual))
	for i, a := range actual {
		texts[i] = a.text
	}
	return matchPattern(texts, required, matchCase)
}

// matchPattern recursively matches texts against pattern using *, +, ? wildcards.
func matchPattern(texts, pattern []string, matchCase bool) bool {
	if len(pattern) == 0 {
		return len(texts) == 0
	}
	p := pattern[0]
	rest := pattern[1:]

	switch p {
	case "*":
		// Zero or more unspecified headings.
		// Try consuming 0, 1, 2, ... headings.
		for i := 0; i <= len(texts); i++ {
			if matchPattern(texts[i:], rest, matchCase) {
				return true
			}
		}
		return false
	case "+":
		// One or more unspecified headings.
		for i := 1; i <= len(texts); i++ {
			if matchPattern(texts[i:], rest, matchCase) {
				return true
			}
		}
		return false
	case "?":
		// Exactly one unspecified heading.
		if len(texts) == 0 {
			return false
		}
		return matchPattern(texts[1:], rest, matchCase)
	default:
		// Exact match.
		if len(texts) == 0 {
			return false
		}
		a := texts[0]
		if !matchCase {
			a = strings.ToLower(a)
			p = strings.ToLower(p)
		}
		if a != p {
			return false
		}
		return matchPattern(texts[1:], rest, matchCase)
	}
}

// findFirstMismatch returns the line number of the first mismatched heading.
func findFirstMismatch(actual []struct {
	text string
	line int
}, required []string, matchCase bool) int {
	// Simple linear scan to find first position that doesn't match.
	ai := 0
	for _, p := range required {
		switch p {
		case "*":
			// consume until next required matches or end
			continue
		case "+", "?":
			if ai < len(actual) {
				ai++
			}
		default:
			if ai >= len(actual) {
				return 0
			}
			a := actual[ai].text
			exp := p
			if !matchCase {
				a = strings.ToLower(a)
				exp = strings.ToLower(exp)
			}
			if a != exp {
				return actual[ai].line
			}
			ai++
		}
	}
	return 0
}
