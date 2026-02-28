package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD053 checks that link/image reference definitions are used.
type MD053 struct {
	// IgnoredDefinitions is a list of definitions to ignore (default ["//"]).
	IgnoredDefinitions []string `json:"ignored_definitions"`
}

func (r MD053) ID() string          { return "MD053" }
func (r MD053) Aliases() []string   { return []string{"link-image-reference-definitions"} }
func (r MD053) Description() string { return "Link and image reference definitions should be needed" }

func (r MD053) ignoredDefs() map[string]bool {
	defs := r.IgnoredDefinitions
	if len(defs) == 0 {
		defs = []string{"//"}
	}
	m := make(map[string]bool)
	for _, d := range defs {
		m[strings.ToLower(d)] = true
	}
	return m
}

func (r MD053) Check(doc *lint.Document) []lint.Violation {
	mask := fencedCodeBlockMask(doc.Lines)
	ignored := r.ignoredDefs()

	// Collect definitions and their line numbers.
	type defEntry struct {
		label string
		line  int
	}
	var defs []defEntry
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		if m := md052DefRE.FindStringSubmatch(line); m != nil {
			label := strings.ToLower(m[1])
			defs = append(defs, defEntry{label, i + 1})
		}
	}

	if len(defs) == 0 {
		return nil
	}

	// Collect used labels.
	used := make(map[string]bool)
	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		// Skip definition lines themselves.
		if md052DefRE.MatchString(line) {
			continue
		}
		// Full references.
		for _, m := range md052FullRE.FindAllStringSubmatch(line, -1) {
			label := strings.ToLower(m[1])
			if label != "" {
				used[label] = true
			}
		}
		// Collapsed references.
		for _, m := range md052CollapsedRE.FindAllStringSubmatch(line, -1) {
			used[strings.ToLower(m[1])] = true
		}
		// Shortcut references.
		for _, m := range md052ShortcutRE.FindAllStringSubmatch(line, -1) {
			used[strings.ToLower(m[1])] = true
		}
	}

	var violations []lint.Violation
	for _, def := range defs {
		if ignored[def.label] {
			continue
		}
		if !used[def.label] {
			violations = append(violations, lint.Violation{
				Rule:    r.ID(),
				Line:    def.line,
				Column:  1,
				Message: "Link and image reference definitions should be needed [Label: " + def.label + "]",
			})
		}
	}
	return violations
}

func (r MD053) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	mask := fencedCodeBlockMask(lines)
	ignored := r.ignoredDefs()

	// Collect used labels.
	used := make(map[string]bool)
	for i, line := range lines {
		if mask[i] {
			continue
		}
		// Skip definition lines themselves.
		if md052DefRE.MatchString(line) {
			continue
		}
		for _, m := range md052FullRE.FindAllStringSubmatch(line, -1) {
			label := strings.ToLower(m[1])
			if label != "" {
				used[label] = true
			}
		}
		for _, m := range md052CollapsedRE.FindAllStringSubmatch(line, -1) {
			used[strings.ToLower(m[1])] = true
		}
		for _, m := range md052ShortcutRE.FindAllStringSubmatch(line, -1) {
			used[strings.ToLower(m[1])] = true
		}
	}

	var result []string
	for i, line := range lines {
		if mask[i] {
			result = append(result, line)
			continue
		}
		if m := md052DefRE.FindStringSubmatch(line); m != nil {
			label := strings.ToLower(m[1])
			if !ignored[label] && !used[label] {
				continue // Remove unused definition.
			}
		}
		result = append(result, line)
	}
	return []byte(strings.Join(result, "\n"))
}
