package rules

import (
	"regexp"
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD054 checks that links and images use allowed styles.
type MD054 struct {
	Autolink  bool `json:"autolink"`
	Collapsed bool `json:"collapsed"`
	Full      bool `json:"full"`
	Inline    bool `json:"inline"`
	Shortcut  bool `json:"shortcut"`
	URLInline bool `json:"url_inline"`
}

func (r MD054) ID() string          { return "MD054" }
func (r MD054) Description() string { return "Link and image style" }

func (r MD054) defaults() MD054 {
	result := r
	// All default to true if not explicitly set (zero value = false means disabled).
	// Since Go zero value is false, we can't distinguish "not set" from "false".
	// The convention is: if all are false, use all defaults = true.
	if !result.Autolink && !result.Collapsed && !result.Full && !result.Inline && !result.Shortcut && !result.URLInline {
		result.Autolink = true
		result.Collapsed = true
		result.Full = true
		result.Inline = true
		result.Shortcut = true
		result.URLInline = true
	}
	return result
}

// md054AutolinkRE matches autolinks <url>.
var md054AutolinkRE = regexp.MustCompile(`<(https?://[^>]+)>`)

// md054InlineRE matches inline links [text](url) or images ![text](url).
var md054InlineRE = regexp.MustCompile(`!?\[([^\]]*)\]\(([^)]*)\)`)

// md054FullRE matches full references [text][label] or images.
var md054FullRefRE = regexp.MustCompile(`!?\[([^\]]*)\]\[([^\]]+)\]`)

// md054CollapsedRE matches collapsed references [label][].
var md054CollapsedRefRE = regexp.MustCompile(`!?\[([^\]]+)\]\[\]`)

// md054ShortcutRE matches shortcut references [label] (not followed by ( or [).
var md054ShortcutRefRE = regexp.MustCompile(`!?\[([^\]]+)\](?:[^(\[]|$)`)

// md054URLInlineRE matches inline links where text is a URL.
// We check equality in code rather than using backreferences.
var md054URLInlineRE = regexp.MustCompile(`\[(https?://[^\]]+)\]\((https?://[^)]+)\)`)

func (r MD054) Check(doc *lint.Document) []lint.Violation {
	cfg := r.defaults()
	mask := fencedCodeBlockMask(doc.Lines)
	var violations []lint.Violation

	for i, line := range doc.Lines {
		if mask[i] {
			continue
		}
		// Check autolinks.
		if !cfg.Autolink {
			for _, m := range md054AutolinkRE.FindAllStringSubmatch(line, -1) {
				violations = append(violations, lint.Violation{
					Rule: r.ID(), Line: i + 1, Column: 1,
					Message: "Link and image style [Autolink not allowed: " + m[0] + "]",
				})
			}
		}
		// Check url_inline (inline link where text == url).
		if !cfg.URLInline {
			for _, m := range md054URLInlineRE.FindAllStringSubmatch(line, -1) {
				if m[1] == m[2] {
					violations = append(violations, lint.Violation{
						Rule: r.ID(), Line: i + 1, Column: 1,
						Message: "Link and image style [URL inline not allowed: " + m[0] + "]",
					})
				}
			}
		}
		// Check inline links.
		if !cfg.Inline {
			for _, m := range md054InlineRE.FindAllStringSubmatch(line, -1) {
				// Skip autolinks.
				if md054AutolinkRE.MatchString(m[0]) {
					continue
				}
				violations = append(violations, lint.Violation{
					Rule: r.ID(), Line: i + 1, Column: 1,
					Message: "Link and image style [Inline link not allowed: " + m[0] + "]",
				})
			}
		}
		// Check full references.
		if !cfg.Full {
			for _, m := range md054FullRefRE.FindAllStringSubmatch(line, -1) {
				violations = append(violations, lint.Violation{
					Rule: r.ID(), Line: i + 1, Column: 1,
					Message: "Link and image style [Full reference not allowed: " + m[0] + "]",
				})
			}
		}
		// Check collapsed references.
		if !cfg.Collapsed {
			for _, m := range md054CollapsedRefRE.FindAllStringSubmatch(line, -1) {
				violations = append(violations, lint.Violation{
					Rule: r.ID(), Line: i + 1, Column: 1,
					Message: "Link and image style [Collapsed reference not allowed: " + m[0] + "]",
				})
			}
		}
		// Check shortcut references.
		if !cfg.Shortcut {
			clean := line
			// Remove full and collapsed refs to avoid false positives.
			clean = md054FullRefRE.ReplaceAllString(clean, "")
			clean = md054CollapsedRefRE.ReplaceAllString(clean, "")
			clean = md054InlineRE.ReplaceAllString(clean, "")
			for range md054ShortcutRefRE.FindAllStringSubmatch(clean, -1) {
				violations = append(violations, lint.Violation{
					Rule: r.ID(), Line: i + 1, Column: 1,
					Message: "Link and image style [Shortcut reference not allowed]",
				})
			}
		}
	}
	return violations
}

// unused import guard
var _ = strings.Contains
