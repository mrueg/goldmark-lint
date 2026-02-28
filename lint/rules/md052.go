package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD052 checks that reference links and images use defined labels.
type MD052 struct {
	// ShortcutSyntax controls whether shortcut references [label] are checked (default false).
	ShortcutSyntax bool `json:"shortcut_syntax"`
	// IgnoredLabels is a list of labels to ignore (default ["x"]).
	IgnoredLabels []string `json:"ignored_labels"`
}

func (r MD052) ID() string { return "MD052" }
func (r MD052) Aliases() []string {
	return []string{"reference-links-images"}
}
func (r MD052) Description() string {
	return "Reference links and images should use a label that is defined"
}

// md052DefRE matches reference link definitions: [label]: url.
var md052DefRE = regexp.MustCompile(`(?i)^\s*\[([^\]]+)\]:\s+\S`)

// md052FullRE matches full reference links/images: [text][label] or ![text][label].
var md052FullRE = regexp.MustCompile(`!?\[[^\]]*\]\[([^\]]*)\]`)

// md052CollapsedRE matches collapsed references: [label][].
var md052CollapsedRE = regexp.MustCompile(`!?\[([^\]]+)\]\[\]`)

// md052ShortcutRE matches shortcut references: [label] (not followed by ( or [).
var md052ShortcutRE = regexp.MustCompile(`!?\[([^\]]+)\](?:[^(\[{]|$)`)

func (r MD052) ignoredLabels() map[string]bool {
	labels := r.IgnoredLabels
	if len(labels) == 0 {
		labels = []string{"x"}
	}
	m := make(map[string]bool)
	for _, l := range labels {
		m[strings.ToLower(l)] = true
	}
	return m
}

func (r MD052) Check(doc *lint.Document) []lint.Violation {
	mask := fencedCodeBlockMask(doc.Lines)
	ignored := r.ignoredLabels()

	// Collect defined labels.
	defined := make(map[string]bool)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if m := md052DefRE.FindStringSubmatch(line); m != nil {
			defined[strings.ToLower(m[1])] = true
		}
	}

	var violations []lint.Violation
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		// Full references: [text][label] - label is group 1, may be empty (collapsed).
		for _, m := range md052FullRE.FindAllStringSubmatch(line, -1) {
			label := strings.ToLower(m[1])
			if label == "" {
				continue // collapsed handled below
			}
			if ignored[label] {
				continue
			}
			if !defined[label] {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  1,
					Message: "Reference links and images should use a label that is defined [Label: " + m[1] + "]",
				})
			}
		}
		// Collapsed references: [label][].
		for _, m := range md052CollapsedRE.FindAllStringSubmatch(line, -1) {
			label := strings.ToLower(m[1])
			if ignored[label] {
				continue
			}
			if !defined[label] {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    i + 1,
					Column:  1,
					Message: "Reference links and images should use a label that is defined [Label: " + m[1] + "]",
				})
			}
		}
		// Shortcut references: [label].
		if r.ShortcutSyntax {
			for _, m := range md052ShortcutRE.FindAllStringSubmatch(line, -1) {
				label := strings.ToLower(m[1])
				if ignored[label] {
					continue
				}
				if !defined[label] {
					violations = append(violations, lint.Violation{
						Rule:    r.ID(),
						Line:    i + 1,
						Column:  1,
						Message: "Reference links and images should use a label that is defined [Label: " + m[1] + "]",
					})
				}
			}
		}
	}
	return violations
}
