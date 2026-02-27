package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/yuin/goldmark/ast"
)

// MD051 checks that link fragments point to existing headings.
type MD051 struct {
	// IgnoreCase controls case-insensitive matching (default false).
	IgnoreCase bool `json:"ignore_case"`
	// IgnoredPattern is a regex pattern for fragments to ignore.
	IgnoredPattern string `json:"ignored_pattern"`
}

func (r MD051) ID() string          { return "MD051" }
func (r MD051) Description() string { return "Link fragments should be valid" }

// md051FragRE matches internal links with fragments: [text](#fragment).
var md051FragRE = regexp.MustCompile(`\[([^\]]*)\]\(#([^)]*)\)`)

// md051LineRefRE matches GitHub line reference patterns like #L123 or #L1C1-L2C2.
var md051LineRefRE = regexp.MustCompile(`^L\d+(?:C\d+-L\d+C\d+)?$`)

// md051HTMLAnchorRE matches HTML id or name attributes.
var md051HTMLAnchorRE = regexp.MustCompile(`(?i)(?:id|name)="([^"]+)"`)

func (r MD051) Check(doc *lint.Document) []lint.Violation {
	// Collect heading anchors.
	anchors := make(map[string]bool)
	_ = ast.Walk(doc.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		text := headingText(h, doc.Source)
		anchor := headingAnchor(text)
		anchors[anchor] = true
		if r.IgnoreCase {
			anchors[strings.ToLower(anchor)] = true
		}
		return ast.WalkContinue, nil
	})

	// Collect HTML anchors.
	for _, line := range doc.Lines {
		for _, m := range md051HTMLAnchorRE.FindAllStringSubmatch(line, -1) {
			anchors[m[1]] = true
		}
	}

	var ignoredRE *regexp.Regexp
	if r.IgnoredPattern != "" {
		if re, err := regexp.Compile(r.IgnoredPattern); err == nil {
			ignoredRE = re
		}
	}

	var violations []lint.Violation
	mask := fencedCodeBlockMask(doc.Lines)

	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		for _, m := range md051FragRE.FindAllStringSubmatch(line, -1) {
			fragment := m[2]
			// Always allow #top.
			if fragment == "top" {
				continue
			}
			// Allow GitHub line references.
			if md051LineRefRE.MatchString(fragment) {
				continue
			}
			// Check ignored pattern.
			if ignoredRE != nil && ignoredRE.MatchString(fragment) {
				continue
			}
			// Check if anchor exists.
			checkFrag := fragment
			if r.IgnoreCase {
				checkFrag = strings.ToLower(fragment)
			}
			if !anchors[checkFrag] {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  1,
					Message: "Link fragments should be valid [Fragment: #" + fragment + "]",
				})
			}
		}
	}
	return violations
}
