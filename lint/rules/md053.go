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
	// Also skip indented code block lines and HTML block lines to avoid false negatives
	// from bracket-like patterns in code blocks being counted as usages.
	indentedMask := indentedCodeBlockMask(doc)
	htmlMask := htmlBlockLineMask(doc)
	ignored := r.ignoredDefs()

	skipLine := func(i int) bool {
		return mask[i] || indentedMask[i] || htmlMask[i]
	}

	// Collect definitions and their line numbers.
	// Only count definitions with valid URLs (no unbalanced trailing paren),
	// and skip footnote definitions [^label]: text.
	type defEntry struct {
		label string
		line  int
	}
	var defs []defEntry
	seen := make(map[string]bool) // first-seen label set for duplicate detection
	for i, line := range doc.Lines {
		if skipLine(i) {
			continue
		}
		if !md052DefLabelValid(line) {
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
		if skipLine(i) {
			continue
		}
		// Skip link reference definition lines themselves, but NOT footnote
		// definition lines [^label]: text — those contain inline content that
		// may use link labels, so they must be scanned.
		if md052DefLabelValid(line) && md052DefRE.MatchString(line) {
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
		// A duplicate definition (same label already seen earlier) is always
		// "unused" because the first definition takes precedence (CommonMark).
		isDuplicate := seen[def.label]
		seen[def.label] = true
		if isDuplicate || !used[def.label] {
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
		// Skip link reference definition lines themselves, but not footnote defs.
		if md052DefLabelValid(line) && md052DefRE.MatchString(line) {
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

	seen := make(map[string]bool)
	var result []string
	for i, line := range lines {
		if mask[i] {
			result = append(result, line)
			continue
		}
		if md052DefLabelValid(line) {
			if m := md052DefRE.FindStringSubmatch(line); m != nil {
				label := strings.ToLower(m[1])
				isDuplicate := seen[label]
				seen[label] = true
				if !ignored[label] && (isDuplicate || !used[label]) {
					continue // Remove unused/duplicate definition.
				}
			}
		}
		result = append(result, line)
	}
	return []byte(strings.Join(result, "\n"))
}
