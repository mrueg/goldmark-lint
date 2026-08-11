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

// md052DefURLRE captures the URL portion of a reference link definition.
// Group 1 is the label, group 2 is the destination URL (up to whitespace or end).
var md052DefURLRE = regexp.MustCompile(`(?i)^\s*\[[^\]]+\]:\s+(\S+)`)

// md052DefLabelValid returns true if the line looks like a valid link reference
// definition whose URL has balanced parentheses and whose label is not a footnote
// label (which starts with '^').  This extra validation prevents the regex-based
// scanner from accepting malformed definitions that goldmark's parser would reject.
func md052DefLabelValid(line string) bool {
	m := md052DefRE.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	// Footnote definitions [^label]: text are not link reference definitions.
	if strings.HasPrefix(m[1], "^") {
		return false
	}
	// Extract the destination and check for unbalanced trailing ')'.
	um := md052DefURLRE.FindStringSubmatch(line)
	if um == nil {
		return false
	}
	dest := um[1]
	open, close := 0, 0
	for _, c := range dest {
		switch c {
		case '(':
			open++
		case ')':
			close++
		}
	}
	return close <= open
}

// md052FullRE matches full reference links/images: [text][label] or ![text][label].
var md052FullRE = regexp.MustCompile(`!?\[[^\]]*\]\[([^\]]*)\]`)

// md052CollapsedRE matches collapsed references: [label][].
var md052CollapsedRE = regexp.MustCompile(`!?\[([^\]]+)\]\[\]`)

// md052ShortcutRE matches shortcut references: [label] (not followed by ( or [).
var md052ShortcutRE = regexp.MustCompile(`!?\[([^\]]+)\](?:[^(\[{]|$)`)

// isFootnoteLabel reports whether label is a GFM footnote reference such as
// [^note]. markdownlint parses these as footnotes rather than link references,
// so an undefined one is not an MD052 violation. The definition scanner already
// excluded them; the usage side has to agree.
func isFootnoteLabel(label string) bool {
	return strings.HasPrefix(label, "^")
}

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
	// Also skip indented code block lines and HTML block lines to avoid false positives.
	indentedMask := indentedCodeBlockMask(doc)
	htmlMask := htmlBlockLineMask(doc)
	ignored := r.ignoredLabels()

	skipLine := func(i int) bool {
		return mask[i] || indentedMask[i] || htmlMask[i]
	}

	// Collect defined labels.
	// Use goldmark's parsed link references for accurate label detection.
	// This handles multi-line definitions, title-on-next-line, angle-bracket
	// destinations, etc. — all cases that the regex may miss.
	//
	// Only goldmark's references are trusted. Re-scanning the source for
	// "[label]: url" patterns over-collects, because a link reference
	// definition is only a definition while it sits at the start of a block: a
	// malformed one ends the run and turns every following line into paragraph
	// text. In rust-lang/rfcs text/0507-release-channels.md a definition with
	// an unbalanced ")" does exactly that, and the regex scan still registered
	// the definitions below it, hiding two real violations. goldmark already
	// handles definitions in blockquotes and labels containing code spans, so
	// the scan bought nothing that the parser does not provide.
	defined := make(map[string]bool)
	for label := range doc.LinkRefs {
		lower := strings.ToLower(label)
		defined[lower] = true
		// Usage lines are matched after blanking code spans, so register the
		// blanked form of the label too; that is how a reference such as
		// [`name`][`name`] finds its definition.
		if blanked := blankCodeSpans(lower); blanked != lower {
			defined[blanked] = true
		}
	}

	var violations []lint.Violation
	for i, line := range doc.Lines {
		if skipLine(i) {
			continue
		}
		checkLine := blankCodeSpans(line)
		// Full references: [text][label] - label is group 1, may be empty (collapsed).
		for _, m := range md052FullRE.FindAllStringSubmatch(checkLine, -1) {
			label := strings.ToLower(m[1])
			if label == "" {
				continue // collapsed handled below
			}
			if ignored[label] || isFootnoteLabel(label) {
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
		for _, m := range md052CollapsedRE.FindAllStringSubmatch(checkLine, -1) {
			label := strings.ToLower(m[1])
			if ignored[label] || isFootnoteLabel(label) {
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
			for _, m := range md052ShortcutRE.FindAllStringSubmatch(checkLine, -1) {
				label := strings.ToLower(m[1])
				if ignored[label] || isFootnoteLabel(label) {
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
