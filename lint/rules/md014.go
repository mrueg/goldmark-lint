package rules

import (
	"strings"

	"github.com/mrueg/goldmark-lint/lint"
)

// MD014 checks that dollar signs are not used before commands without showing output.
type MD014 struct{}

func (r MD014) ID() string { return "MD014" }
func (r MD014) Aliases() []string {
	return []string{"commands-show-output"}
}
func (r MD014) Description() string {
	return "Dollar signs used before commands without showing output"
}

// codeBlockContent returns the content lines (start, end) for each fenced code block.
// start and end are 0-based indices; content is lines[start:end].
func codeBlockContent(lines []string) [][2]int {
	var ranges [][2]int
	n := len(lines)
	i := 0
	for i < n {
		trimmed := strings.TrimLeft(lines[i], " ")
		if len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
			fc := trimmed[0]
			j := 0
			for j < len(trimmed) && trimmed[j] == fc {
				j++
			}
			if j >= 3 {
				fenceLen := j
				fenceChar := fc
				start := i + 1
				end := n
				for k := start; k < n; k++ {
					t := strings.TrimLeft(lines[k], " ")
					if len(t) >= fenceLen && t[0] == fenceChar {
						m := 0
						for m < len(t) && t[m] == fenceChar {
							m++
						}
						if m >= fenceLen && strings.TrimSpace(t[m:]) == "" {
							end = k
							break
						}
					}
				}
				ranges = append(ranges, [2]int{start, end})
				i = end + 1
				continue
			}
		}
		i++
	}
	return ranges
}

// allDollarLines reports whether all non-blank lines in the slice start with "$ " or are "$".
func allDollarLines(lines []string) bool {
	hasAny := false
	for _, cl := range lines {
		if strings.TrimSpace(cl) == "" {
			continue
		}
		trimmed := strings.TrimLeft(cl, " \t")
		if trimmed != "$" && !strings.HasPrefix(trimmed, "$ ") {
			return false
		}
		hasAny = true
	}
	return hasAny
}

func (r MD014) Check(doc *lint.Document) []lint.Violation {
	var violations []lint.Violation
	for _, rng := range codeBlockContent(doc.Lines) {
		start, end := rng[0], rng[1]
		if !allDollarLines(doc.Lines[start:end]) {
			continue
		}
		for k := start; k < end; k++ {
			if strings.TrimSpace(doc.Lines[k]) != "" {
				violations = append(violations, lint.Violation{
					Rule:    r.ID(),
					Line:    k + 1,
					Column:  1,
					Message: "Dollar signs used before commands without showing output",
				})
			}
		}
	}
	return violations
}

func (r MD014) Fix(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	for _, rng := range codeBlockContent(lines) {
		start, end := rng[0], rng[1]
		if !allDollarLines(lines[start:end]) {
			continue
		}
		for k := start; k < end; k++ {
			if strings.TrimSpace(lines[k]) == "" {
				continue
			}
			indent := len(lines[k]) - len(strings.TrimLeft(lines[k], " \t"))
			content := strings.TrimLeft(lines[k], " \t")
			if strings.HasPrefix(content, "$ ") {
				lines[k] = lines[k][:indent] + content[2:]
			} else if content == "$" {
				lines[k] = lines[k][:indent]
			}
		}
	}
	return []byte(strings.Join(lines, "\n"))
}
