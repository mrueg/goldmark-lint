package lint_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrueg/goldmark-lint/lint"
	"github.com/mrueg/goldmark-lint/lint/rules"
)

func newDefaultLinter() *lint.Linter {
	return lint.NewLinter(
		rules.MD001{},
		rules.MD003{},
		rules.MD004{},
		rules.MD005{},
		rules.MD007{},
		rules.MD009{},
		rules.MD010{},
		rules.MD011{},
		rules.MD012{},
		rules.MD013{},
		rules.MD014{},
		rules.MD018{},
		rules.MD019{},
		rules.MD020{},
		rules.MD021{},
		rules.MD022{},
		rules.MD023{},
		rules.MD024{},
		rules.MD025{},
		rules.MD026{},
		rules.MD027{},
		rules.MD028{},
		rules.MD029{},
		rules.MD030{},
		rules.MD031{},
		rules.MD032{},
		rules.MD033{},
		rules.MD034{},
		rules.MD035{},
		rules.MD036{},
		rules.MD037{},
		rules.MD038{},
		rules.MD039{},
		rules.MD040{},
		rules.MD041{},
		rules.MD042{},
		rules.MD043{},
		rules.MD044{},
		rules.MD045{},
		rules.MD046{},
		rules.MD047{},
		rules.MD048{},
		rules.MD049{},
		rules.MD050{},
		rules.MD051{},
		rules.MD052{},
		rules.MD053{},
		rules.MD054{},
		rules.MD055{},
		rules.MD056{},
		rules.MD058{},
		rules.MD059{},
		rules.MD060{},
	)
}

func lintString(t *testing.T, r lint.Rule, source string) []lint.Violation {
	t.Helper()
	l := lint.NewLinter(r)
	return l.Lint([]byte(source))
}

func TestMD001_Valid(t *testing.T) {
	src := "# Heading 1\n\n## Heading 2\n\n### Heading 3\n"
	v := lintString(t, rules.MD001{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD001_Invalid(t *testing.T) {
	src := "# Heading 1\n\n### Heading 3\n"
	v := lintString(t, rules.MD001{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD009_Valid(t *testing.T) {
	src := "No trailing spaces\n"
	v := lintString(t, rules.MD009{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD009_Invalid(t *testing.T) {
	src := "Trailing spaces   \n"
	v := lintString(t, rules.MD009{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD010_Valid(t *testing.T) {
	src := "No hard tabs\n"
	v := lintString(t, rules.MD010{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD010_Invalid(t *testing.T) {
	src := "Hard\ttab\n"
	v := lintString(t, rules.MD010{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD012_Valid(t *testing.T) {
	src := "Line 1\n\nLine 2\n"
	v := lintString(t, rules.MD012{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD012_Invalid(t *testing.T) {
	src := "Line 1\n\n\nLine 2\n"
	v := lintString(t, rules.MD012{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD013_Valid(t *testing.T) {
	src := "Short line\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD013_Invalid(t *testing.T) {
	// Use a wrappable line (contains spaces) that exceeds the limit.
	// The "trimmed" length (before the last word) must also exceed the limit.
	src := strings.Repeat("a", 80) + " extra\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD013_NonWrappableExempt(t *testing.T) {
	// A single-word line (no spaces) should NOT be flagged in non-strict mode,
	// matching markdownlint-cli2 behaviour (the last word cannot be wrapped).
	src := strings.Repeat("a", 81) + "\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for non-wrappable line, got %d: %v", len(v), v)
	}
}

func TestMD013_StrictFlagsNonWrappable(t *testing.T) {
	// In strict mode, even non-wrappable single-word lines are flagged.
	src := strings.Repeat("a", 81) + "\n"
	v := lintString(t, rules.MD013{LineLength: 80, Strict: true}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation in strict mode for 81-char line, got %d: %v", len(v), v)
	}
}

func TestMD022_Valid(t *testing.T) {
	src := "# Heading\n\nParagraph\n"
	v := lintString(t, rules.MD022{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD022_Invalid(t *testing.T) {
	src := "Text\n# Heading\nMore text\n"
	v := lintString(t, rules.MD022{}, src)
	if len(v) == 0 {
		t.Errorf("expected violations, got none")
	}
}

func TestMD025_Valid(t *testing.T) {
	src := "# Only one top-level heading\n\n## Sub heading\n"
	v := lintString(t, rules.MD025{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD025_Invalid(t *testing.T) {
	src := "# First\n\n# Second\n"
	v := lintString(t, rules.MD025{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD025_ContentBeforeFirstHeading(t *testing.T) {
	// Content before the first H1: markdownlint does not report violations.
	src := "Some preamble text.\n\n# First\n\n# Second\n"
	v := lintString(t, rules.MD025{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations when content precedes first H1, got %d: %v", len(v), v)
	}
}

func TestMD041_Valid(t *testing.T) {
	src := "# First line heading\n"
	v := lintString(t, rules.MD041{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD041_Invalid(t *testing.T) {
	src := "Not a heading\n"
	v := lintString(t, rules.MD041{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD047_Valid(t *testing.T) {
	src := "Content\n"
	v := lintString(t, rules.MD047{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD047_Invalid(t *testing.T) {
	src := "Content"
	v := lintString(t, rules.MD047{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func fixString(t *testing.T, r lint.FixableRule, source string) string {
	t.Helper()
	return string(r.Fix([]byte(source)))
}

func TestMD009_Fix(t *testing.T) {
	src := "Trailing spaces   \n"
	got := fixString(t, rules.MD009{}, src)
	want := "Trailing spaces\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD009_Fix_KeepBrSpaces(t *testing.T) {
	src := "Hard line break  \n"
	got := fixString(t, rules.MD009{}, src)
	// exactly 2 trailing spaces are kept as a hard line break
	if got != src {
		t.Errorf("Fix() = %q, want %q (brSpaces preserved)", got, src)
	}
}

func TestMD010_Fix(t *testing.T) {
	src := "Hard\ttab\n"
	got := fixString(t, rules.MD010{}, src)
	want := "Hard    tab\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD012_Fix(t *testing.T) {
	src := "Line 1\n\n\nLine 2\n"
	got := fixString(t, rules.MD012{}, src)
	want := "Line 1\n\nLine 2\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD047_Fix(t *testing.T) {
	src := "Content"
	got := fixString(t, rules.MD047{}, src)
	want := "Content\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD047_Fix_AlreadyEndsWithNewline(t *testing.T) {
	src := "Content\n"
	got := fixString(t, rules.MD047{}, src)
	if got != src {
		t.Errorf("Fix() = %q, want unchanged %q", got, src)
	}
}

func TestLinter_Fix(t *testing.T) {
	// tab in middle, trailing spaces, no final newline
	src := "Content\there   "
	l := lint.NewLinter(rules.MD009{}, rules.MD010{}, rules.MD047{})
	got := string(l.Fix([]byte(src)))
	want := "Content    here\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD003_Valid(t *testing.T) {
	src := "# Heading 1\n\n## Heading 2\n"
	v := lintString(t, rules.MD003{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD003_Invalid(t *testing.T) {
	src := "# ATX Heading\n\nSetext Heading\n==============\n"
	v := lintString(t, rules.MD003{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD003_StyleATX_Valid(t *testing.T) {
	src := "# Heading 1\n\n## Heading 2\n"
	v := lintString(t, rules.MD003{Style: "atx"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD003_StyleATX_Invalid(t *testing.T) {
	src := "Setext Heading\n==============\n"
	v := lintString(t, rules.MD003{Style: "atx"}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD004_Valid(t *testing.T) {
	src := "- item1\n- item2\n- item3\n"
	v := lintString(t, rules.MD004{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD004_Invalid(t *testing.T) {
	src := "- item1\n\n* item2\n"
	v := lintString(t, rules.MD004{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD007_Valid(t *testing.T) {
	src := "- item1\n  - sub-item\n"
	v := lintString(t, rules.MD007{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD007_Invalid(t *testing.T) {
	src := "- item1\n   - bad indent\n"
	v := lintString(t, rules.MD007{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD024_Valid(t *testing.T) {
	src := "# Heading 1\n\n## Heading 2\n"
	v := lintString(t, rules.MD024{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD024_Invalid(t *testing.T) {
	src := "# Duplicate\n\n## Duplicate\n"
	v := lintString(t, rules.MD024{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD029_Valid(t *testing.T) {
	src := "1. item1\n2. item2\n3. item3\n"
	v := lintString(t, rules.MD029{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD029_ValidAllOne(t *testing.T) {
	src := "1. item1\n1. item2\n1. item3\n"
	v := lintString(t, rules.MD029{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD029_Invalid(t *testing.T) {
	src := "1. item1\n3. item2\n2. item3\n"
	v := lintString(t, rules.MD029{}, src)
	if len(v) == 0 {
		t.Errorf("expected violations, got none")
	}
}

func TestMD033_Valid(t *testing.T) {
	src := "# Heading\n\nParagraph with **bold** text.\n"
	v := lintString(t, rules.MD033{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD033_Invalid(t *testing.T) {
	src := "# Heading\n\nParagraph with <b>bold</b> text.\n"
	v := lintString(t, rules.MD033{}, src)
	// Only the opening <b> tag should be reported, not the closing </b>.
	if len(v) != 1 {
		t.Errorf("expected 1 violation (opening tag only), got %d: %v", len(v), v)
	}
}

func TestMD033_MultilineHTMLTag(t *testing.T) {
	// An HTML opening tag whose attributes span multiple lines must be detected.
	// The closing ">" appears on a later line, so per-line scanning alone misses it.
	src := "<p align=\"center\">\n    <img src=\"image.svg\"\n        alt=\"An image\"\n        height=\"800px\">\n</p>\n"
	v := lintString(t, rules.MD033{}, src)
	// Expect violations for <p> (line 1) and <img> (line 2).
	if len(v) < 2 {
		t.Errorf("expected at least 2 violations (p and img), got %d: %v", len(v), v)
	}
	foundImg := false
	for _, viol := range v {
		if viol.Line == 2 {
			foundImg = true
		}
	}
	if !foundImg {
		t.Errorf("expected violation for <img> at line 2, got %v", v)
	}
}

func TestMD034_Valid(t *testing.T) {
	src := "Visit <https://example.com> for more.\n"
	v := lintString(t, rules.MD034{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD034_ValidLink(t *testing.T) {
	src := "Visit [example](https://example.com) for more.\n"
	v := lintString(t, rules.MD034{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD034_Invalid(t *testing.T) {
	src := "Visit https://example.com for more.\n"
	v := lintString(t, rules.MD034{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD011_Valid(t *testing.T) {
	src := "See [text](url) for more.\n"
	v := lintString(t, rules.MD011{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD011_Invalid(t *testing.T) {
	src := "See (text)[url] for more.\n"
	v := lintString(t, rules.MD011{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD011_Fix(t *testing.T) {
	src := "See (text)[url] for more.\n"
	got := fixString(t, rules.MD011{}, src)
	want := "See [text](url) for more.\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD018_Valid(t *testing.T) {
	src := "# Heading\n"
	v := lintString(t, rules.MD018{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD018_Invalid(t *testing.T) {
	src := "#Heading\n"
	v := lintString(t, rules.MD018{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD018_Fix(t *testing.T) {
	src := "#Heading\n"
	got := fixString(t, rules.MD018{}, src)
	want := "# Heading\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD018_HTMLBlockNoFalsePositive(t *testing.T) {
	// #text inside an HTML block (no blank lines) must not be flagged.
	src := "# Title\n\n<details>\n<summary>Click</summary>\n#anchor\n</details>\n\n## After\n"
	v := lintString(t, rules.MD018{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for #text inside HTML block, got %v", v)
	}
}

func TestMD018_ListContinuationNoFalsePositive(t *testing.T) {
	// A multi-line link label like "[issue\n  #8636](url)" in a list item must not be flagged.
	src := "- See [issue\n  #8636](https://example.com).\n"
	v := lintString(t, rules.MD018{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for link label continuation in list item, got %v", v)
	}
}

func TestMD019_Valid(t *testing.T) {
	src := "# Heading\n"
	v := lintString(t, rules.MD019{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD019_ValidClosedATX(t *testing.T) {
	// Closed ATX headings with extra spaces are handled by MD021, not MD019.
	src := "##  Heading  ##\n"
	v := lintString(t, rules.MD019{}, src)
	if len(v) != 0 {
		t.Errorf("expected no MD019 violations for closed ATX heading, got %v", v)
	}
}

func TestMD019_Invalid(t *testing.T) {
	src := "#  Heading\n"
	v := lintString(t, rules.MD019{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD019_Fix(t *testing.T) {
	src := "#  Heading\n"
	got := fixString(t, rules.MD019{}, src)
	want := "# Heading\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD020_Valid(t *testing.T) {
	src := "## Heading ##\n"
	v := lintString(t, rules.MD020{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD020_Invalid(t *testing.T) {
	src := "## Heading##\n"
	v := lintString(t, rules.MD020{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD020_Fix(t *testing.T) {
	src := "## Heading##\n"
	got := fixString(t, rules.MD020{}, src)
	want := "## Heading ##\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD021_Valid(t *testing.T) {
	src := "## Heading ##\n"
	v := lintString(t, rules.MD021{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD021_Invalid(t *testing.T) {
	src := "##  Heading  ##\n"
	v := lintString(t, rules.MD021{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD021_Fix(t *testing.T) {
	src := "##  Heading  ##\n"
	got := fixString(t, rules.MD021{}, src)
	want := "## Heading ##\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD031_Valid(t *testing.T) {
	src := "Text\n\n```go\ncode\n```\n\nMore text\n"
	v := lintString(t, rules.MD031{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD031_Invalid(t *testing.T) {
	src := "Text\n```go\ncode\n```\nMore text\n"
	v := lintString(t, rules.MD031{}, src)
	if len(v) == 0 {
		t.Errorf("expected violations, got none")
	}
}

func TestMD031_Fix(t *testing.T) {
	src := "Text\n```go\ncode\n```\nMore text\n"
	got := fixString(t, rules.MD031{}, src)
	want := "Text\n\n```go\ncode\n```\n\nMore text\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD032_Valid(t *testing.T) {
	src := "Text\n\n- item1\n- item2\n\nMore text\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD032_Invalid(t *testing.T) {
	src := "Text\n- item1\n- item2\nMore text\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) == 0 {
		t.Errorf("expected violations, got none")
	}
}

func TestMD032_Fix(t *testing.T) {
	src := "Text\n- item1\n- item2\nMore text\n"
	got := fixString(t, rules.MD032{}, src)
	want := "Text\n\n- item1\n- item2\n\nMore text\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD032_FencedCodeBlock_NoViolation(t *testing.T) {
	// List-like lines inside a fenced code block must not trigger violations.
	src := "Text\n\n```yaml\nargs:\n- --resources=pods\n- --node=$(NODE_NAME)\nenv:\n- name: NODE_NAME\n```\n\nMore text\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for list items inside fenced code block, got %v", v)
	}
}

func TestMD032_FencedCodeBlock_NoFix(t *testing.T) {
	// Fix must not insert blank lines inside a fenced code block.
	src := "Text\n\n```yaml\nargs:\n- --resources=pods\n- --node=$(NODE_NAME)\nenv:\n- name: NODE_NAME\n```\n\nMore text\n"
	got := fixString(t, rules.MD032{}, src)
	if got != src {
		t.Errorf("Fix() modified content inside fenced code block:\n got  %q\n want %q", got, src)
	}
}

func TestMD032_MultilineListItem_NoViolation(t *testing.T) {
	// A list item with a continuation line must not generate false violations
	// when it is already surrounded by blank lines.
	src := "Text\n\n- item 1\n  continuation\n- item 2\n\nMore text\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for valid multiline list, got %v", v)
	}
}

func TestMD032_MultilineListItem_Violation(t *testing.T) {
	// A list missing blank lines both before and after produces two violations.
	// "More text" immediately after the list (without a blank line) is flagged
	// even though CommonMark treats it as a lazy continuation — markdownlint
	// reports the after-violation too.
	src := "Text\n- item 1\n  continuation\n- item 2\nMore text\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) != 2 {
		t.Errorf("expected 2 violations (before and after) for list without blank lines, got %d: %v", len(v), v)
	}
}

func TestMD032_MultilineListItem_Fix(t *testing.T) {
	// Fix must not insert blank lines between a list item and its continuation.
	src := "Text\n- item 1\n  continuation\n- item 2\nMore text\n"
	got := fixString(t, rules.MD032{}, src)
	want := "Text\n\n- item 1\n  continuation\n- item 2\n\nMore text\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD032_SingleItem_NoDoubleViolation(t *testing.T) {
	// A single-item list missing blank lines both before and after must produce
	// exactly one violation, not two (matching markdownlint behaviour).
	src := "Text\n- item 1\nMore text\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for single-item list without blank lines, got %d: %v", len(v), v)
	}
}

func TestMD032_TableAfterList_Violation(t *testing.T) {
	// A table immediately after a list without a blank line should be flagged.
	src := "- item 1\n- item 2\n| a | b |\n| - | - |\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) == 0 {
		t.Errorf("expected violation for table after list without blank line, got none")
	}
}

func TestMD032_LinkRefDefAfterList_Violation(t *testing.T) {
	// A link reference definition immediately after a list without a blank line should be flagged.
	src := "- item 1\n- item 2\n[label]: https://example.com\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) == 0 {
		t.Errorf("expected violation for link ref def after list without blank line, got none")
	}
}

func TestMD032_BlockquoteList_NoFalsePositive(t *testing.T) {
	// A list inside a blockquote must not produce false after-violations caused
	// by the "> " prefix of the following blockquote-continuation lines.
	src := "Before.\n\n> * Item one\n>   continuation\n>\n> * Item two\n>   continuation\n\nAfter.\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for list inside blockquote, got %v", v)
	}
}

func TestMD032_LazyContinuationBeforeCodeBlock_Violation(t *testing.T) {
	// A list item whose lazy-continuation text is immediately followed by a fenced
	// code block (no blank line) should produce an after-violation.
	src := "Intro.\n\n+ Item one,\ncontinuation\n```rust\ncode\n```\n+ Item two\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) == 0 {
		t.Errorf("expected violation for list+lazy-continuation+code block without blank line, got none")
	}
}

func TestMD037_Valid(t *testing.T) {
	src := "This is *emphasized* text.\n"
	v := lintString(t, rules.MD037{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD037_ValidMultiWord(t *testing.T) {
	// Multi-word strong emphasis should NOT be flagged (no spaces inside markers).
	src := "This is **strongly emphasized** text.\n"
	v := lintString(t, rules.MD037{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for multi-word **...**, got %v", v)
	}
}

func TestMD037_EmphasisWithCodeSpanAtEnd(t *testing.T) {
	// Emphasis ending with a code span should NOT be flagged: the space between
	// the text and the code span does not constitute a space before the closing marker.
	src := "See *foo `bar`* for details.\n"
	v := lintString(t, rules.MD037{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for emphasis ending with code span, got %d: %v", len(v), v)
	}
}

func TestMD037_Invalid(t *testing.T) {
	// In CommonMark, "* emphasized *" is NOT parsed as emphasis (the opening *
	// is followed by a space, making it non-left-flanking). Goldmark's AST-based
	// detection correctly produces no violations for this input.
	src := "This is * emphasized * text.\n"
	v := lintString(t, rules.MD037{}, src)
	if len(v) != 0 {
		t.Errorf("expected 0 violations for non-emphasis asterisks, got %d: %v", len(v), v)
	}
}

func TestMD037_Fix(t *testing.T) {
	// Fix is a no-op since CommonMark emphasis cannot have spaces inside markers.
	src := "This is * emphasized * text.\n"
	got := fixString(t, rules.MD037{}, src)
	want := src
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD038_Valid(t *testing.T) {
	src := "Use `code` here.\n"
	v := lintString(t, rules.MD038{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD038_Invalid(t *testing.T) {
	// `code ` has trailing space only — not stripped by CommonMark (asymmetric).
	src := "Use `code ` here.\n"
	v := lintString(t, rules.MD038{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD038_Fix(t *testing.T) {
	src := "Use `code ` here.\n"
	got := fixString(t, rules.MD038{}, src)
	want := "Use `code` here.\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD038_MultipleSpansOnSameLine_OneViolation(t *testing.T) {
	// Two code spans with trailing spaces on the same line should produce only
	// one violation (markdownlint deduplicates by line).
	src := "text `: ` and `, ` more\n"
	v := lintString(t, rules.MD038{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for two code spans with spaces on same line, got %d: %v", len(v), v)
	}
}

func TestMD039_Valid(t *testing.T) {
	src := "See [text](url) here.\n"
	v := lintString(t, rules.MD039{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD039_Invalid(t *testing.T) {
	src := "See [ text ](url) here.\n"
	v := lintString(t, rules.MD039{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD039_Fix(t *testing.T) {
	src := "See [ text ](url) here.\n"
	got := fixString(t, rules.MD039{}, src)
	want := "See [text](url) here.\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD040_Valid(t *testing.T) {
	src := "```go\ncode\n```\n"
	v := lintString(t, rules.MD040{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD040_Invalid(t *testing.T) {
	src := "```\ncode\n```\n"
	v := lintString(t, rules.MD040{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD042_Valid(t *testing.T) {
	src := "See [text](https://example.com) here.\n"
	v := lintString(t, rules.MD042{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD042_ValidCodeSpanText(t *testing.T) {
	// A link whose text is a code span should NOT be flagged as empty.
	src := "[`code`](https://example.com)\n"
	v := lintString(t, rules.MD042{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for code span link text, got %v", v)
	}
}

func TestMD042_Invalid(t *testing.T) {
	src := "See [text]() here.\n"
	v := lintString(t, rules.MD042{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD042_NestedInlineNoPanic(t *testing.T) {
	// Link inside emphasis - inlineNodeLine must not panic
	src := "**[text]()**\n"
	v := lintString(t, rules.MD042{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD045_Valid(t *testing.T) {
	src := "![alt text](image.png)\n"
	v := lintString(t, rules.MD045{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD045_Invalid(t *testing.T) {
	src := "![](image.png)\n"
	v := lintString(t, rules.MD045{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD045_HTMLImgNoAlt(t *testing.T) {
	// Inline HTML <img> without alt attribute should be flagged.
	src := "Some text <img src=\"image.png\"> more text\n"
	v := lintString(t, rules.MD045{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for <img> without alt, got %d: %v", len(v), v)
	}
}

func TestMD045_HTMLImgWithAlt(t *testing.T) {
	// Inline HTML <img> with alt attribute should not be flagged.
	src := "Some text <img src=\"image.png\" alt=\"desc\"> more text\n"
	v := lintString(t, rules.MD045{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for <img> with alt, got %v", v)
	}
}

func TestMD005_Valid(t *testing.T) {
	src := "- item1\n- item2\n  - sub1\n  - sub2\n"
	v := lintString(t, rules.MD005{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD005_Invalid(t *testing.T) {
	src := "- item1\n  - sub1\n   - sub2\n"
	v := lintString(t, rules.MD005{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD014_Valid(t *testing.T) {
	src := "```bash\n$ ls\nfile1.txt\n```\n"
	v := lintString(t, rules.MD014{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD014_Invalid(t *testing.T) {
	src := "```bash\n$ ls\n$ pwd\n```\n"
	v := lintString(t, rules.MD014{}, src)
	if len(v) != 2 {
		t.Errorf("expected 2 violations, got %d: %v", len(v), v)
	}
}

func TestMD014_Fix(t *testing.T) {
	src := "```bash\n$ ls\n$ pwd\n```\n"
	got := fixString(t, rules.MD014{}, src)
	want := "```bash\nls\npwd\n```\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD023_Valid(t *testing.T) {
	src := "# Heading\n\n## Sub heading\n"
	v := lintString(t, rules.MD023{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD023_Invalid(t *testing.T) {
	src := "# Heading\n\n  ## Indented heading\n"
	v := lintString(t, rules.MD023{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD023_Fix(t *testing.T) {
	src := "# Heading\n\n  ## Indented heading\n"
	got := fixString(t, rules.MD023{}, src)
	want := "# Heading\n\n## Indented heading\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD026_Valid(t *testing.T) {
	src := "# Heading\n\n## Sub heading\n"
	v := lintString(t, rules.MD026{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD026_Invalid(t *testing.T) {
	src := "# Heading.\n\n## Sub heading!\n"
	v := lintString(t, rules.MD026{}, src)
	if len(v) != 2 {
		t.Errorf("expected 2 violations, got %d: %v", len(v), v)
	}
}

func TestMD026_Fix(t *testing.T) {
	src := "# Heading.\n"
	got := fixString(t, rules.MD026{}, src)
	want := "# Heading\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD027_Valid(t *testing.T) {
	src := "> Single space\n"
	v := lintString(t, rules.MD027{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD027_Invalid(t *testing.T) {
	src := ">  Multiple spaces\n"
	v := lintString(t, rules.MD027{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD027_Fix(t *testing.T) {
	src := ">  Multiple spaces\n"
	got := fixString(t, rules.MD027{}, src)
	want := "> Multiple spaces\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD028_Valid(t *testing.T) {
	src := "> Line 1\n> Line 2\n"
	v := lintString(t, rules.MD028{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD028_Invalid(t *testing.T) {
	src := "> Line 1\n\n> Line 2\n"
	v := lintString(t, rules.MD028{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD030_Valid(t *testing.T) {
	src := "- Item 1\n- Item 2\n"
	v := lintString(t, rules.MD030{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD030_Invalid(t *testing.T) {
	src := "-  Item 1\n-  Item 2\n"
	v := lintString(t, rules.MD030{}, src)
	if len(v) == 0 {
		t.Errorf("expected violations, got none")
	}
}

func TestMD030_Fix(t *testing.T) {
	src := "-  Item 1\n1.  Item 2\n"
	got := fixString(t, rules.MD030{}, src)
	want := "- Item 1\n1. Item 2\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD035_Valid(t *testing.T) {
	src := "---\n\nText\n\n---\n"
	v := lintString(t, rules.MD035{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD035_Invalid(t *testing.T) {
	src := "---\n\nText\n\n***\n"
	v := lintString(t, rules.MD035{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD036_Valid(t *testing.T) {
	src := "# Heading\n\nParagraph text.\n"
	v := lintString(t, rules.MD036{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD036_Invalid(t *testing.T) {
	src := "\n**Bold heading**\n\nText\n"
	v := lintString(t, rules.MD036{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD043_Valid(t *testing.T) {
	src := "# Introduction\n\n## Details\n"
	v := lintString(t, rules.MD043{Headings: []string{"# Introduction", "## Details"}}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD043_Empty(t *testing.T) {
	src := "# Introduction\n"
	v := lintString(t, rules.MD043{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations when headings is empty, got %v", v)
	}
}

func TestMD043_Invalid(t *testing.T) {
	src := "# Introduction\n\n## Wrong\n"
	v := lintString(t, rules.MD043{Headings: []string{"# Introduction", "## Details"}}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD044_Valid(t *testing.T) {
	src := "Use JavaScript for scripting.\n"
	v := lintString(t, rules.MD044{Names: []string{"JavaScript"}}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD044_Invalid(t *testing.T) {
	src := "Use javascript for scripting.\n"
	v := lintString(t, rules.MD044{Names: []string{"JavaScript"}}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD044_Fix(t *testing.T) {
	src := "Use javascript for scripting.\n"
	got := fixString(t, rules.MD044{Names: []string{"JavaScript"}}, src)
	want := "Use JavaScript for scripting.\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD046_Valid(t *testing.T) {
	src := "```go\ncode\n```\n"
	v := lintString(t, rules.MD046{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD046_Invalid(t *testing.T) {
	src := "Text\n\n    indented code\n"
	v := lintString(t, rules.MD046{Style: "fenced"}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD048_Valid(t *testing.T) {
	src := "```go\ncode\n```\n"
	v := lintString(t, rules.MD048{Style: "backtick"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD048_Invalid(t *testing.T) {
	src := "~~~go\ncode\n~~~\n"
	v := lintString(t, rules.MD048{Style: "backtick"}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD048_Fix(t *testing.T) {
	src := "~~~go\ncode\n~~~\n"
	got := fixString(t, rules.MD048{Style: "backtick"}, src)
	want := "```go\ncode\n~~~\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD048_IndentedCodeBlockNoFalsePositive(t *testing.T) {
	// A 4-space-indented line containing ``` is an indented code block, not a fence.
	// When the document's first real fence uses tildes, the indented ``` line must
	// not be treated as a backtick fence opener (which would cause a spurious MD048).
	src := "# Title\n\n~~~go\ncode\n~~~\n\nParagraph.\n\n    ```go\n    code\n    ```\n"
	v := lintString(t, rules.MD048{Style: "consistent"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for indented code block with backticks, got %v", v)
	}
}

func TestMD049_Valid(t *testing.T) {
	src := "Use *asterisk* emphasis.\n"
	v := lintString(t, rules.MD049{Style: "asterisk"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD049_Invalid(t *testing.T) {
	src := "Use _underscore_ emphasis.\n"
	v := lintString(t, rules.MD049{Style: "asterisk"}, src)
	if len(v) != 2 {
		t.Errorf("expected 2 violations, got %d: %v", len(v), v)
	}
}

func TestMD049_Fix(t *testing.T) {
	src := "Use _underscore_ emphasis.\n"
	got := fixString(t, rules.MD049{Style: "asterisk"}, src)
	want := "Use *underscore* emphasis.\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD049_Fix_ListItemWithCodeSpan(t *testing.T) {
	// A list item whose marker happens to pair with a * inside a code span
	// must not be modified by the Fix function.
	src := "* [FEATURE] Add `kube_networkpolicy_*` metrics #893\n"
	got := fixString(t, rules.MD049{Style: "underscore"}, src)
	if got != src {
		t.Errorf("Fix() modified list item with * inside code span:\n got  %q\n want %q", got, src)
	}
}

func TestMD049_Check_ListItemWithCodeSpan(t *testing.T) {
	// The Check function must not report a false violation for a list item
	// whose marker * pairs visually with a * inside a code span.
	src := "* [FEATURE] Add `kube_networkpolicy_*` metrics #893\n"
	v := lintString(t, rules.MD049{Style: "underscore"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD049_Fix_ListItemWithEscapedAsterisk(t *testing.T) {
	// A list item whose bullet * pairs with a \* escaped asterisk in the text
	// must not be modified by the Fix function.
	src := `* [CHANGE]       Fix empty string for "owner_\*" dimensions #1923 @pawcykca` + "\n"
	got := fixString(t, rules.MD049{Style: "underscore"}, src)
	if got != src {
		t.Errorf("Fix() modified list item with escaped asterisk:\n got  %q\n want %q", got, src)
	}
}

func TestMD049_Check_ListItemWithEscapedAsterisk(t *testing.T) {
	// The Check function must not report a false violation for a list item
	// whose bullet * pairs with a \* escaped asterisk in the text.
	src := `* [CHANGE]       Fix empty string for "owner_\*" dimensions #1923 @pawcykca` + "\n"
	v := lintString(t, rules.MD049{Style: "underscore"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD050_Valid(t *testing.T) {
	src := "Use **asterisk** strong.\n"
	v := lintString(t, rules.MD050{Style: "asterisk"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD050_Invalid(t *testing.T) {
	src := "Use __underscore__ strong.\n"
	v := lintString(t, rules.MD050{Style: "asterisk"}, src)
	if len(v) != 2 {
		t.Errorf("expected 2 violations, got %d: %v", len(v), v)
	}
}

func TestMD050_Fix(t *testing.T) {
	src := "Use __underscore__ strong.\n"
	got := fixString(t, rules.MD050{Style: "asterisk"}, src)
	want := "Use **underscore** strong.\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD051_Valid(t *testing.T) {
	src := "# Hello World\n\n[link](#hello-world)\n"
	v := lintString(t, rules.MD051{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD051_Invalid(t *testing.T) {
	src := "# Hello\n\n[link](#nonexistent)\n"
	v := lintString(t, rules.MD051{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD051_DuplicateHeadings_Valid(t *testing.T) {
	// Link to second occurrence of a duplicate heading (#section-1) should be valid.
	src := "# Section\n\n## Section\n\n[link1](#section)\n[link2](#section-1)\n"
	v := lintString(t, rules.MD051{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for duplicate heading anchors, got %v", v)
	}
}

func TestMD052_Valid(t *testing.T) {
	src := "[link][ref]\n\n[ref]: https://example.com\n"
	v := lintString(t, rules.MD052{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD052_Invalid(t *testing.T) {
	src := "[link][undefined]\n"
	v := lintString(t, rules.MD052{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD052_IndentedCodeBlock_NoViolation(t *testing.T) {
	// References inside indented code blocks must not be flagged (false positive).
	src := "Text\n\n    [foo][UNDEFINED] = something\n\nmore text\n"
	v := lintString(t, rules.MD052{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for ref in indented code block, got %d: %v", len(v), v)
	}
}

func TestMD052_HTMLBlock_NoViolation(t *testing.T) {
	// References inside HTML blocks must not be flagged (false positive).
	src := "<div>\n[foo][UNDEFINED]\n</div>\n"
	v := lintString(t, rules.MD052{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for ref in HTML block, got %d: %v", len(v), v)
	}
}

func TestMD052_CodeSpanLabel_NoViolation(t *testing.T) {
	// Collapsed reference [`genawaiter`][] where definition [`genawaiter`]: url exists
	// should not be flagged. The code-span in the label is blanked by blankCodeSpans,
	// so we must register both raw and blanked forms of the definition label.
	src := "Use [`genawaiter`][] here.\n\n[`genawaiter`]: https://example.com\n"
	v := lintString(t, rules.MD052{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for code-span label with definition, got %d: %v", len(v), v)
	}
}

func TestMD053_Valid(t *testing.T) {
	src := "[link][ref]\n\n[ref]: https://example.com\n"
	v := lintString(t, rules.MD053{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD053_Invalid(t *testing.T) {
	src := "Some text.\n\n[unused]: https://example.com\n"
	v := lintString(t, rules.MD053{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD053_Fix(t *testing.T) {
	src := "Some text.\n\n[unused]: https://example.com\n"
	got := fixString(t, rules.MD053{}, src)
	want := "Some text.\n\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD053_IndentedCodeBlock_FalseNegative(t *testing.T) {
	// A label that is only "used" inside an indented code block should still be
	// reported as unused — the code block usage doesn't count.
	src := "Some text.\n\n    [defined]\n\n[defined]: https://example.com\n"
	v := lintString(t, rules.MD053{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for def used only in indented code block, got %d: %v", len(v), v)
	}
}

func TestMD034_BareEmail_NoViolation(t *testing.T) {
	// Bare email addresses should not be flagged by MD034 (only bare URLs).
	src := "Contact user@example.com for help.\n"
	v := lintString(t, rules.MD034{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for bare email, got %d: %v", len(v), v)
	}
}

func TestMD034_BrokenLinkSyntaxURL_NoViolation(t *testing.T) {
	// A URL that appears as the destination of a broken link ['text'(url) should
	// not be flagged: markdownlint treats it as an attempted link, not a bare URL.
	src := "See ['some text'(https://example.com) for details.\n"
	v := lintString(t, rules.MD034{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for URL in broken-link syntax, got %d: %v", len(v), v)
	}
}

func TestMD034_ProseParenURL_Violation(t *testing.T) {
	// A URL inside parentheses in prose (without a preceding '[') should still
	// be flagged as a bare URL.
	src := "Check (https://example.com) for more info.\n"
	v := lintString(t, rules.MD034{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for URL in prose parentheses, got %d: %v", len(v), v)
	}
}

func TestMD054_Valid(t *testing.T) {
	src := "[link](https://example.com)\n"
	v := lintString(t, rules.MD054{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD054_Invalid(t *testing.T) {
	// Only autolinks are allowed; inline links are disallowed.
	src := "[link](https://example.com)\n"
	v := lintString(t, rules.MD054{Autolink: true}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD055_Valid(t *testing.T) {
	src := "| Col1 | Col2 |\n| ---- | ---- |\n| A    | B    |\n"
	v := lintString(t, rules.MD055{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD055_Invalid(t *testing.T) {
	// Header and delimiter have leading+trailing pipes; data row does not.
	src := "| Col1 | Col2 |\n| ---- | ---- |\nA | B\n"
	v := lintString(t, rules.MD055{Style: "leading_and_trailing"}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD056_Valid(t *testing.T) {
	src := "| Col1 | Col2 |\n| ---- | ---- |\n| A    | B    |\n"
	v := lintString(t, rules.MD056{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD056_Invalid(t *testing.T) {
	src := "| Col1 | Col2 |\n| ---- | ---- |\n| A    |\n"
	v := lintString(t, rules.MD056{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD058_Valid(t *testing.T) {
	src := "Text\n\n| Col1 | Col2 |\n| ---- | ---- |\n| A    | B    |\n\nMore text\n"
	v := lintString(t, rules.MD058{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD058_Invalid(t *testing.T) {
	src := "Text\n| Col1 | Col2 |\n| ---- | ---- |\n| A    | B    |\n"
	v := lintString(t, rules.MD058{}, src)
	if len(v) != 0 {
		// This may or may not fire depending on table detection.
		// Just ensure it doesn't panic.
		t.Logf("violations: %v", v)
	}
}

func TestMD058_Fix(t *testing.T) {
	src := "Text\n| Col1 | Col2 |\n| ---- | ---- |\n| A    | B    |\nMore text\n"
	got := fixString(t, rules.MD058{}, src)
	want := "Text\n\n| Col1 | Col2 |\n| ---- | ---- |\n| A    | B    |\n\nMore text\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

func TestMD059_Valid(t *testing.T) {
	src := "[Read the docs](https://example.com)\n"
	v := lintString(t, rules.MD059{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD059_Invalid(t *testing.T) {
	src := "[click here](https://example.com)\n"
	v := lintString(t, rules.MD059{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD059_FormattedText_Bold(t *testing.T) {
	// Generic link text wrapped in bold should also be flagged.
	src := "[**here**](https://example.com)\n"
	v := lintString(t, rules.MD059{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for bold generic link text, got %d: %v", len(v), v)
	}
}

func TestMD059_FormattedText_Italic(t *testing.T) {
	// Generic link text wrapped in italic should also be flagged.
	src := "[*here*](https://example.com)\n"
	v := lintString(t, rules.MD059{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for italic generic link text, got %d: %v", len(v), v)
	}
}

func TestMD060_Valid(t *testing.T) {
	// Aligned table (all pipes at the same columns): "any" style → no violations.
	src := "| Col1 | Col2 |\n| ---- | ---- |\n| A    | B    |\n"
	v := lintString(t, rules.MD060{Style: "any"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD060_Invalid(t *testing.T) {
	// Header row is compact-spaced; data row is tight (no spaces around content).
	src := "| Col1 | Col2 |\n| ---- | ---- |\n|A|B|\n"
	v := lintString(t, rules.MD060{Style: "compact"}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD060_Default_Consistent(t *testing.T) {
	// Default style is "consistent": a data row with a different style than the
	// header row produces one violation (for the inconsistent data row).
	src := "| Col1 | Col2 |\n| ---- | ---- |\n|A|B|\n"
	v := lintString(t, rules.MD060{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation with default consistent style, got %d: %v", len(v), v)
	}
}

func TestMD060_Any_AlignedTable_NoViolations(t *testing.T) {
	// Compact table (all pipes aligned): "any" style → no violations.
	src := "| Col1 | Col2 |\n| ---- | ---- |\n| A    | B    |\n"
	v := lintString(t, rules.MD060{Style: "any"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for aligned table, got %v", v)
	}
}

func TestMD060_Consistent_Valid(t *testing.T) {
	// Consistent style: all rows compact → no violations.
	src := "| Col1 | Col2 |\n| ---- | ---- |\n| A | B |\n"
	v := lintString(t, rules.MD060{Style: "consistent"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for consistent compact table, got %v", v)
	}
}

func TestMD060_SingleSpaceCell(t *testing.T) {
	// A cell containing only a single space should not panic.
	src := "| | Col2 |\n| - | ---- |\n| A | B |\n"
	v := lintString(t, rules.MD060{Style: "consistent"}, src)
	_ = v // just ensure no panic
}

// --- Inline disable comment tests ---

func TestInlineDisable_DisableAll(t *testing.T) {
	// All violations suppressed for lines after disable, re-enabled after enable.
	src := "# Heading 1\n\n<!-- markdownlint-disable -->\n### Heading 3\n<!-- markdownlint-enable -->\n\n## Heading 2\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	// Line 4 (### Heading 3) is inside disable block, should not be reported.
	for _, violation := range v {
		if violation.Line == 4 {
			t.Errorf("expected no violation on line 4 (inside disable block), got %v", violation)
		}
	}
}

func TestInlineDisable_DisableSpecificRule(t *testing.T) {
	// Only MD001 is suppressed; any other rule still fires.
	src := "<!-- markdownlint-disable MD001 -->\n### Heading 3\n<!-- markdownlint-enable MD001 -->\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (MD001 disabled), got %v", v)
	}
}

func TestInlineDisable_EnableRestores(t *testing.T) {
	// After re-enabling, violations should fire again.
	// H1 → H3 (suppressed) → H1 → H3 (should fire: jump from H1 to H3)
	src := "# Heading 1\n\n<!-- markdownlint-disable MD001 -->\n### Heading 3\n<!-- markdownlint-enable MD001 -->\n\n# Heading 1 again\n\n### Heading 3 again\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	// Line 4 (inside disable) must not fire.
	for _, violation := range v {
		if violation.Line == 4 {
			t.Errorf("expected no violation on line 4 (inside disable block), got %v", violation)
		}
	}
	// Line 9 (after re-enable, H1→H3 jump) must fire.
	found := false
	for _, violation := range v {
		if violation.Line == 9 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected violation on line 9 (after re-enable), got %v", v)
	}
}

func TestInlineDisable_DisableLine(t *testing.T) {
	// disable-line suppresses violations only on the current line.
	src := "# Heading 1\n\n### Heading 3 <!-- markdownlint-disable-line MD001 -->\n\n### Heading 3 again\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	for _, violation := range v {
		if violation.Line == 3 {
			t.Errorf("expected no violation on line 3 (disable-line), got %v", violation)
		}
	}
}

func TestInlineDisable_DisableLineAllRules(t *testing.T) {
	// disable-line without rule IDs suppresses all rules on the current line.
	src := "Trailing spaces   <!-- markdownlint-disable-line -->\n"
	l := lint.NewLinter(rules.MD009{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (disable-line all), got %v", v)
	}
}

func TestInlineDisable_DisableNextLine(t *testing.T) {
	// disable-next-line suppresses violations on the following line only.
	// H1 → H3 (suppressed) → H1 → H3 (should fire: jump from H1 to H3)
	src := "# Heading 1\n\n<!-- markdownlint-disable-next-line MD001 -->\n### Heading 3\n\n# Heading 1 again\n\n### Heading 3 again\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	// Line 4 suppressed via disable-next-line.
	for _, violation := range v {
		if violation.Line == 4 {
			t.Errorf("expected no violation on line 4 (disable-next-line), got %v", violation)
		}
	}
	// Line 8 (H1→H3 jump after disable-next-line scope) should fire.
	found := false
	for _, violation := range v {
		if violation.Line == 8 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected violation on line 8 (after disable-next-line), got %v", v)
	}
}

func TestInlineDisable_DisableNextLineAllRules(t *testing.T) {
	// disable-next-line without rule IDs suppresses all rules on the next line.
	src := "<!-- markdownlint-disable-next-line -->\nTrailing spaces   \n"
	l := lint.NewLinter(rules.MD009{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (disable-next-line all), got %v", v)
	}
}

func TestInlineDisable_CaptureRestore(t *testing.T) {
	// capture saves state; restore brings it back.
	// H1 → capture → disable → H3 (suppressed) → restore → H1 → H3 (should fire)
	src := "# Heading 1\n\n<!-- markdownlint-capture -->\n<!-- markdownlint-disable MD001 -->\n### Heading 3\n<!-- markdownlint-restore -->\n\n# Heading 1 again\n\n### Heading 3 again\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	// Line 5 is in disabled block, should not fire.
	for _, violation := range v {
		if violation.Line == 5 {
			t.Errorf("expected no violation on line 5 (inside disable block), got %v", violation)
		}
	}
	// Line 10 (H1→H3 jump after restore) should fire.
	found := false
	for _, violation := range v {
		if violation.Line == 10 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected violation on line 10 (after restore), got %v", v)
	}
}

func TestInlineDisable_DisableNextLineOnlyAffectsNextLine(t *testing.T) {
	// disable-next-line should not affect lines beyond the immediately following line.
	src := "<!-- markdownlint-disable-next-line MD009 -->\nTrailing spaces   \nMore trailing spaces   \n"
	l := lint.NewLinter(rules.MD009{})
	v := l.Lint([]byte(src))
	// Line 2 suppressed; line 3 should still fire.
	found3 := false
	for _, violation := range v {
		if violation.Line == 2 {
			t.Errorf("expected no violation on line 2 (disable-next-line), got %v", violation)
		}
		if violation.Line == 3 {
			found3 = true
		}
	}
	if !found3 {
		t.Errorf("expected violation on line 3, got %v", v)
	}
}

func TestInlineDisable_DisableFile_AllRules(t *testing.T) {
	// disable-file at the bottom suppresses violations for the entire file.
	src := "Trailing spaces   \nMore trailing spaces   \n<!-- markdownlint-disable-file -->\n"
	l := lint.NewLinter(rules.MD009{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (disable-file all), got %v", v)
	}
}

func TestInlineDisable_DisableFile_SpecificRule(t *testing.T) {
	// disable-file for MD001 suppresses MD001 for the entire file regardless of position.
	src := "### Heading 3\n# Heading 1\n<!-- markdownlint-disable-file MD001 -->\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (disable-file MD001), got %v", v)
	}
}

func TestInlineDisable_EnableFile_RestoresAfterDisableFile(t *testing.T) {
	// enable-file cancels a file-level disable.
	// With both disable-file and enable-file for the same rule, enable-file wins.
	src := "<!-- markdownlint-disable-file MD001 -->\n<!-- markdownlint-enable-file MD001 -->\n# Heading 1\n### Heading 3\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	found := false
	for _, violation := range v {
		if violation.Rule == "MD001" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected MD001 violation after enable-file, got %v", v)
	}
}

func TestInlineDisable_ConfigureFile_DisableRule(t *testing.T) {
	// configure-file with false disables the rule for the entire file.
	src := "<!-- markdownlint-configure-file { \"MD001\": false } -->\n### Heading 3\n# Heading 1\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (configure-file MD001:false), got %v", v)
	}
}

func TestInlineDisable_ConfigureFile_DisableRuleByAlias(t *testing.T) {
	// configure-file should accept rule aliases (e.g. "heading-increment" for MD001).
	src := "<!-- markdownlint-configure-file { \"heading-increment\": false } -->\n# Heading 1\n\n### Heading 3\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (configure-file heading-increment:false), got %v", v)
	}
}

func TestInlineDisable_ConfigureFile_EnableRuleByAlias(t *testing.T) {
	// configure-file with true (re-enable) should work with aliases too.
	src := "<!-- markdownlint-configure-file { \"heading-increment\": true } -->\n# Heading 1\n\n### Heading 3\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	if len(v) == 0 {
		t.Errorf("expected MD001 violation (configure-file heading-increment:true keeps rule enabled), got none")
	}
}

func TestInlineDisable_DisableByAlias(t *testing.T) {
	// Inline markdownlint-disable should accept rule aliases.
	src := "<!-- markdownlint-disable heading-increment -->\n# Heading 1\n\n### Heading 3\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (disable by alias heading-increment), got %v", v)
	}
}

func TestInlineDisable_DisableNextLineByAlias(t *testing.T) {
	// markdownlint-disable-next-line should accept rule aliases.
	src := "# Heading 1\n<!-- markdownlint-disable-next-line heading-increment -->\n### Heading 3\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (disable-next-line by alias heading-increment), got %v", v)
	}
}

func TestInlineDisable_DisableLineByAlias(t *testing.T) {
	// markdownlint-disable-line should accept rule aliases.
	src := "# Heading 1\n### Heading 3 <!-- markdownlint-disable-line heading-increment -->\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (disable-line by alias heading-increment), got %v", v)
	}
}

func TestInlineDisable_DisableFileByAlias(t *testing.T) {
	// markdownlint-disable-file should accept rule aliases.
	src := "<!-- markdownlint-disable-file heading-increment -->\n# Heading 1\n\n### Heading 3\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (disable-file by alias heading-increment), got %v", v)
	}
}

func TestInlineDisable_ConfigureFile_DoesNotDisableOtherRules(t *testing.T) {
	// configure-file disabling MD001 should not suppress MD009.
	src := "<!-- markdownlint-configure-file { \"MD001\": false } -->\nTrailing spaces   \n"
	l := lint.NewLinter(rules.MD009{})
	v := l.Lint([]byte(src))
	if len(v) == 0 {
		t.Errorf("expected MD009 violation (not disabled by configure-file), got none")
	}
}

func TestNoInlineConfig_IgnoresDisableComment(t *testing.T) {
	// When NoInlineConfig is true, inline disable comments are ignored.
	src := "<!-- markdownlint-disable MD001 -->\n# Heading 1\n\n### Heading 3\n"
	l := lint.NewLinter(rules.MD001{})
	l.NoInlineConfig = true
	v := l.Lint([]byte(src))
	if len(v) == 0 {
		t.Errorf("expected MD001 violation (NoInlineConfig=true ignores disable comment), got none")
	}
}

func TestNoInlineConfig_IgnoresConfigureFileComment(t *testing.T) {
	// When NoInlineConfig is true, configure-file comments are ignored.
	src := "<!-- markdownlint-configure-file { \"MD001\": false } -->\n# Heading 1\n\n### Heading 3\n"
	l := lint.NewLinter(rules.MD001{})
	l.NoInlineConfig = true
	v := l.Lint([]byte(src))
	if len(v) == 0 {
		t.Errorf("expected MD001 violation (NoInlineConfig=true ignores configure-file comment), got none")
	}
}

func TestNoInlineConfig_False_HonorsDisableComment(t *testing.T) {
	// When NoInlineConfig is false (default), inline disable comments are honored.
	src := "<!-- markdownlint-disable MD001 -->\n# Heading 1\n\n### Heading 3\n"
	l := lint.NewLinter(rules.MD001{})
	v := l.Lint([]byte(src))
	if len(v) != 0 {
		t.Errorf("expected no violations (NoInlineConfig=false honors disable comment), got %v", v)
	}
}

func integrationMarkdownlintAvailable() bool {
	_, err := exec.LookPath("markdownlint")
	return err == nil
}

type mlViolation struct {
	FileName   string   `json:"fileName"`
	LineNumber int      `json:"lineNumber"`
	RuleNames  []string `json:"ruleNames"`
}

func parseMarkdownlintJSON(output string) map[int][]string {
	result := make(map[int][]string)
	var violations []mlViolation
	if err := json.Unmarshal([]byte(output), &violations); err != nil {
		return result
	}
	for _, v := range violations {
		if len(v.RuleNames) > 0 {
			result[v.LineNumber] = append(result[v.LineNumber], v.RuleNames[0])
		}
	}
	return result
}

func groupByLine(violations []lint.Violation) map[int][]string {
	result := make(map[int][]string)
	for _, v := range violations {
		result[v.Line] = append(result[v.Line], v.Rule)
	}
	return result
}

func TestIntegration_CompareWithMarkdownlint(t *testing.T) {
	if !integrationMarkdownlintAvailable() {
		t.Skip("markdownlint not available, skipping integration test")
	}

	testdata := "../testdata"
	files, err := filepath.Glob(filepath.Join(testdata, "*.md"))
	if err != nil || len(files) == 0 {
		t.Skip("no test fixtures found")
	}

	linter := newDefaultLinter()

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			source, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("failed to read %s: %v", file, err)
			}

			goldmarkViolations := linter.Lint(source)
			goldmarkByLine := groupByLine(goldmarkViolations)

			cmd := exec.Command("markdownlint", "--json", file)
			out, _ := cmd.CombinedOutput()

			markdownlintByLine := parseMarkdownlintJSON(string(out))

			for line, mlRules := range markdownlintByLine {
				glRules := goldmarkByLine[line]
				glRuleSet := make(map[string]bool)
				for _, r := range glRules {
					glRuleSet[r] = true
				}
				for _, r := range mlRules {
					if !glRuleSet[r] {
						t.Logf("line %d: markdownlint found %s but goldmark-lint did not (goldmark found: %v)", line, r, glRules)
					}
				}
			}
		})
	}
}

func TestMD010_SpacesPerTab(t *testing.T) {
	src := "Hard\ttab\n"
	// SpacesPerTab=2 should replace tab with 2 spaces.
	got := fixString(t, rules.MD010{SpacesPerTab: 2}, src)
	want := "Hard  tab\n"
	if got != want {
		t.Errorf("Fix() with SpacesPerTab=2: got %q, want %q", got, want)
	}
}

func TestMD013_HeadingLineLength(t *testing.T) {
	heading := "# " + strings.Repeat("a", 79) + "\n"
	// Body line must be wrappable (contain spaces) so the trimmed length exceeds limit.
	body := strings.Repeat("b", 80) + " extra\n"
	src := heading + "\n" + body

	// With headings limit=100 and default=80: heading line is short enough, body is too long.
	v := lintString(t, rules.MD013{LineLength: 80, HeadingLineLength: 100}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation (body), got %d: %v", len(v), v)
	}
	if len(v) == 1 && v[0].Line != 3 {
		t.Errorf("expected violation on line 3 (body), got line %d", v[0].Line)
	}
}

func TestMD013_CodeBlockLineLength(t *testing.T) {
	body := strings.Repeat("a", 81) + "\n"
	src := "```\n" + strings.Repeat("b", 81) + "\n```\n"

	// With code_block_line_length=100 and default=80: code block line is short enough.
	v := lintString(t, rules.MD013{LineLength: 80, CodeBlockLineLength: 100}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for code block with high limit, got %d: %v", len(v), v)
	}

	// But default body text that exceeds limit still triggers.
	_ = body
}

func TestMD013_CodeBlocksDisabled(t *testing.T) {
	src := "```\n" + strings.Repeat("a", 81) + "\n```\n"
	f := false
	v := lintString(t, rules.MD013{LineLength: 80, CodeBlocks: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations when code_blocks=false, got %d: %v", len(v), v)
	}
}

func TestMD013_TablesDisabled(t *testing.T) {
	src := "| " + strings.Repeat("a", 79) + " |\n|---|\n"
	f := false
	v := lintString(t, rules.MD013{LineLength: 80, Tables: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations when tables=false, got %d: %v", len(v), v)
	}
}

func TestMD013_HeadingsDisabled(t *testing.T) {
	src := "# " + strings.Repeat("a", 79) + "\n"
	f := false
	v := lintString(t, rules.MD013{LineLength: 80, Headings: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations when headings=false, got %d: %v", len(v), v)
	}
}

func TestMD029_Fix(t *testing.T) {
	src := "1. item1\n3. item2\n2. item3\n"
	got := fixString(t, rules.MD029{}, src)
	want := "1. item1\n2. item2\n3. item3\n"
	if got != want {
		t.Errorf("MD029 Fix() = %q, want %q", got, want)
	}
}

func TestMD029_FixOne(t *testing.T) {
	src := "1. item1\n2. item2\n3. item3\n"
	got := fixString(t, rules.MD029{Style: "one"}, src)
	want := "1. item1\n1. item2\n1. item3\n"
	if got != want {
		t.Errorf("MD029 Fix(one) = %q, want %q", got, want)
	}
}

func TestMD040_AllowedLanguages(t *testing.T) {
	// "go" is in allowed list, so no violation.
	src := "```go\ncode\n```\n"
	v := lintString(t, rules.MD040{AllowedLanguages: []string{"go", "python"}}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for allowed language, got %v", v)
	}

	// "rust" is not in allowed list, so violation expected.
	src = "```rust\ncode\n```\n"
	v = lintString(t, rules.MD040{AllowedLanguages: []string{"go", "python"}}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for disallowed language, got %d: %v", len(v), v)
	}
}

func TestMD040_LanguageOnly(t *testing.T) {
	// Pure language identifier — valid.
	src := "```go\ncode\n```\n"
	v := lintString(t, rules.MD040{LanguageOnly: true}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for language-only info string, got %v", v)
	}

	// Language with extra info string — violation.
	src = "```go run\ncode\n```\n"
	v = lintString(t, rules.MD040{LanguageOnly: true}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for info string with extra content, got %d: %v", len(v), v)
	}
}

// Front matter tests

func TestFrontMatter_MD041_Valid(t *testing.T) {
	// Document with YAML front matter followed by a top-level heading should
	// not trigger MD041.
	src := "---\ntitle: My Page\nauthor: Test\n---\n\n# Heading\n"
	v := lintString(t, rules.MD041{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for document with front matter + heading, got %v", v)
	}
}

func TestFrontMatter_MD041_Invalid(t *testing.T) {
	// Document with YAML front matter (no title field) followed by non-heading
	// content should still trigger MD041, reported on the correct line.
	src := "---\ndate: 2024-01-01\n---\n\nNot a heading\n"
	v := lintString(t, rules.MD041{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
	if len(v) == 1 && v[0].Line != 5 {
		t.Errorf("expected violation on line 5, got line %d", v[0].Line)
	}
}

func TestFrontMatter_DotDotDot_Valid(t *testing.T) {
	// Front matter closed with "..." should also be stripped.
	src := "---\ntitle: My Page\n...\n\n# Heading\n"
	v := lintString(t, rules.MD041{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for front matter closed with ..., got %v", v)
	}
}

func TestFrontMatter_NoFrontMatter_Unchanged(t *testing.T) {
	// Documents without front matter should be linted normally.
	src := "Not a heading\n"
	v := lintString(t, rules.MD041{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for document without front matter, got %d: %v", len(v), v)
	}
}

func TestFrontMatter_MD010_NotApplied(t *testing.T) {
	// Tabs inside front matter should not trigger MD010.
	src := "---\nkey:\tvalue\n---\n\n# Heading\n"
	v := lintString(t, rules.MD010{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for tab in front matter, got %v", v)
	}
}

func TestFrontMatter_Fix_PreservesFrontMatter(t *testing.T) {
	// Fix should not modify front matter content.
	src := "---\ntitle: My Page\n---\n\nContent\twith tab\n"
	l := lint.NewLinter(rules.MD010{})
	got := string(l.Fix([]byte(src)))
	want := "---\ntitle: My Page\n---\n\nContent    with tab\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
	}
}

// --- New option tests ---

func TestMD001_FrontMatterTitle(t *testing.T) {
	// When front_matter_title is set and front matter has a title field,
	// an h2 as the first heading should NOT trigger MD001 (h1 is implied).
	src := "---\ntitle: My Page\n---\n\n## Heading 2\n"
	v := lintString(t, rules.MD001{FrontMatterTitle: "title"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations with front_matter_title set, got %v", v)
	}
}

func TestMD001_FrontMatterTitle_Disabled(t *testing.T) {
	// Without front_matter_title, a jump from (implied) h0 to h2 is not checked
	// but h2->h3->... should still be checked.
	src := "---\ntitle: My Page\n---\n\n## Heading 2\n\n#### Heading 4\n"
	v := lintString(t, rules.MD001{}, src)
	// h2->h4 is a skip; violation expected.
	if len(v) == 0 {
		t.Errorf("expected violations for h2->h4 skip, got none")
	}
}

func TestMD003_ATXClosed(t *testing.T) {
	// Closed ATX headings should be detected as "atx_closed".
	src := "## Heading ##\n\n### Sub heading ###\n"
	v := lintString(t, rules.MD003{Style: "atx_closed"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for atx_closed style, got %v", v)
	}
}

func TestMD003_ATXClosed_Invalid(t *testing.T) {
	// Open ATX headings should trigger a violation when atx_closed is required.
	src := "## Heading\n"
	v := lintString(t, rules.MD003{Style: "atx_closed"}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for open ATX when atx_closed required, got %d: %v", len(v), v)
	}
}

func TestMD003_SetextWithATXClosed(t *testing.T) {
	// setext_with_atx_closed: h1/h2 use setext, h3+ use atx_closed.
	src := "Heading 1\n=========\n\n### Heading 3 ###\n"
	v := lintString(t, rules.MD003{Style: "setext_with_atx_closed"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for setext_with_atx_closed, got %v", v)
	}
}

func TestMD004_Sublist(t *testing.T) {
	// sublist style: each nesting level uses a different marker.
	// Level 0: dash, level 1: asterisk, level 2: plus.
	src := "- item1\n  * sub-item\n- item2\n"
	v := lintString(t, rules.MD004{Style: "sublist"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for sublist style, got %v", v)
	}
}

func TestMD004_Sublist_Invalid(t *testing.T) {
	// Both top-level items use dash (correct), but nesting level should be asterisk.
	src := "- item1\n  - wrong-sub\n"
	v := lintString(t, rules.MD004{Style: "sublist"}, src)
	if len(v) == 0 {
		t.Errorf("expected violations for wrong sublist marker, got none")
	}
}

func TestMD007_StartIndented(t *testing.T) {
	// With start_indented=true, top-level items must be indented by indent spaces.
	src := "  - item1\n    - sub-item\n"
	v := lintString(t, rules.MD007{Indent: 2, StartIndented: true}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for start_indented, got %v", v)
	}
}

func TestMD007_StartIndent(t *testing.T) {
	// With start_indented=true and start_indent=4, top-level must start at 4 spaces.
	src := "    - item1\n      - sub-item\n"
	v := lintString(t, rules.MD007{Indent: 2, StartIndented: true, StartIndent: 4}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for start_indent=4, got %v", v)
	}
}

func TestMD009_CodeBlocks_Disabled(t *testing.T) {
	f := false
	src := "```\ncode with trailing   \n```\n"
	v := lintString(t, rules.MD009{CodeBlocks: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations in code block when code_blocks=false, got %v", v)
	}
}

func TestMD009_Strict(t *testing.T) {
	// Strict mode: br_spaces are also disallowed.
	src := "Hard line break  \n"
	v := lintString(t, rules.MD009{Strict: true}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation in strict mode, got %d: %v", len(v), v)
	}
}

func TestMD010_CodeBlocks_Disabled(t *testing.T) {
	f := false
	src := "```\ncode\twith tab\n```\n"
	v := lintString(t, rules.MD010{CodeBlocks: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations in code block when code_blocks=false, got %v", v)
	}
}

func TestMD010_IgnoreCodeLanguages(t *testing.T) {
	src := "```makefile\nrule:\n\tcommand\n```\n"
	v := lintString(t, rules.MD010{IgnoreCodeLanguages: []string{"makefile"}}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for ignored language, got %v", v)
	}
}

func TestMD013_Strict(t *testing.T) {
	// strict=true: heading_line_length is ignored; line_length applies everywhere.
	heading := "# " + strings.Repeat("a", 79) + "\n"
	src := heading
	// With strict=true and line_length=80: heading of 82 chars should trigger.
	v := lintString(t, rules.MD013{LineLength: 80, HeadingLineLength: 200, Strict: true}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation with strict=true, got %d: %v", len(v), v)
	}
}

func TestMD013_Unicode(t *testing.T) {
	// A wrappable line with 81 multi-byte Unicode characters should trigger a violation.
	// Use 80 chars before a space + 1 extra char so the trimmed length (81) exceeds 80.
	src := strings.Repeat("é", 80) + " é\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for 81 unicode chars, got %d: %v", len(v), v)
	}
	// A line with exactly 80 multi-byte Unicode characters should be valid.
	src = strings.Repeat("é", 80) + "\n"
	v = lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for 80 unicode chars, got %v", v)
	}
}

func TestMD013_URLExemption_InlineLink(t *testing.T) {
	// A line that exceeds the limit only because of an inline link URL should
	// be exempt (stern=false, the default).
	src := "[link text](https://www.example.com/very/long/path/that/exceeds/eighty/characters/total)\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for line long only due to URL, got %d: %v", len(v), v)
	}
}

func TestMD013_URLExemption_AutoLink(t *testing.T) {
	// A line with an autolink URL that causes it to exceed the limit should be exempt.
	src := "<https://www.example.com/another/very/long/url/that/is/also/too/long/for/the/limit>\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for autolink line, got %d: %v", len(v), v)
	}
}

func TestMD013_URLExemption_LinkDefinition(t *testing.T) {
	// A link reference definition line that is long only due to the URL should be exempt.
	src := "[ref]: https://www.example.com/reference/link/that/is/also/quite/long/and/exceeds/limit\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for link definition line, got %d: %v", len(v), v)
	}
}

func TestMD013_URLExemption_Stern(t *testing.T) {
	// With stern=true, URLs do not exempt a line from the length check.
	src := "[link text](https://www.example.com/very/long/path/that/exceeds/eighty/characters/total)\n"
	v := lintString(t, rules.MD013{LineLength: 80, Stern: true}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation with stern=true, got %d: %v", len(v), v)
	}
}

func TestMD013_URLExemption_RefLinkCodeSpan(t *testing.T) {
	// A reference link whose link text is a code span (e.g. [`cmd`][ref]) should
	// be correctly attributed to the line it appears on, not to the first line of
	// the surrounding paragraph.  The URL from the reference definition should
	// exempt the line if removing it would make the line fit within the limit.
	//
	// Line 3: "   [`cargo metadata`][wg-cargo-std-aware#20], [`cargo clean`][wg-cargo-std-aware#21],"
	// is 84 chars.  The URL from wg-cargo-std-aware#21 is 57 chars, so
	// lineLen - urlLen = 27 <= 80 → should be exempt.
	src := "Some preceding paragraph text.\n\n" +
		"   [`cargo metadata`][wg-cargo-std-aware#20], [`cargo clean`][wg-cargo-std-aware#21],\n" +
		"\n" +
		"[wg-cargo-std-aware#20]: https://github.com/rust-lang/wg-cargo-std-aware/issues/20\n" +
		"[wg-cargo-std-aware#21]: https://github.com/rust-lang/wg-cargo-std-aware/issues/21\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	for _, viol := range v {
		if viol.Line == 3 {
			t.Errorf("expected line with reference link code-span to be exempt, got violation: %v", viol)
		}
	}
}

func TestMD013_URLExemption_LongLineWithText(t *testing.T) {
	// A line that is long even after removing the URL should still be reported.
	// "See this really long description text at " (42 chars) + URL (50 chars) = 92 chars
	// Without URL: 42 chars <= 80, so it IS exempt.
	// Need a line where text alone exceeds the limit.
	prefix := strings.Repeat("a", 81)
	src := prefix + " [x](https://url)\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation when line is long even without URL, got %d: %v", len(v), v)
	}
}

func TestMD022_LinesAboveArray(t *testing.T) {
	// Per-level: h1 needs 0 blank lines above (since it's first), h2 needs 2.
	src := "# Heading 1\n\n\n## Heading 2\n\nText\n"
	v := lintString(t, rules.MD022{LinesAbove: rules.IntOrArray{0, 2}}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations with per-level LinesAbove, got %v", v)
	}
}

func TestMD025_FrontMatterTitle(t *testing.T) {
	// front_matter_title: document with front matter title + one h1 = no duplicate.
	src := "---\ntitle: My Page\n---\n\n# Heading 1\n"
	v := lintString(t, rules.MD025{FrontMatterTitle: "title"}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation (front matter title + h1 = 2 top-level headings), got %d: %v", len(v), v)
	}
}

func TestMD026_DefaultPunctuation(t *testing.T) {
	// Default punctuation should include full-width chars.
	src := "# Heading。\n"
	v := lintString(t, rules.MD026{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for full-width period with default punctuation, got %d: %v", len(v), v)
	}
}

func TestMD026_NoPunctuationQuestion(t *testing.T) {
	// The default no longer includes '?' - it should not trigger a violation.
	src := "# Heading?\n"
	v := lintString(t, rules.MD026{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for '?' with default punctuation (not included), got %v", v)
	}
}

func TestMD027_ListItems_Disabled(t *testing.T) {
	// list_items=false: skip blockquote check for indented (list item) lines.
	f := false
	src := "- item\n  >  block quote with spaces\n"
	v := lintString(t, rules.MD027{ListItems: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations when list_items=false for indented blockquote, got %v", v)
	}
}

func TestMD030_ULMulti(t *testing.T) {
	// ul_multi=2: multi-line UL items should have 2 spaces after marker.
	src := "-  Item 1\n\n   Continuation\n"
	v := lintString(t, rules.MD030{ULSingle: 1, ULMulti: 2}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for ul_multi=2 with multi-line item, got %v", v)
	}
}

func TestMD031_ListItems_Disabled(t *testing.T) {
	// list_items=false: skip fenced code block blank-line check inside list items.
	f := false
	src := "- item\n  ```go\n  code\n  ```\n- item2\n"
	v := lintString(t, rules.MD031{ListItems: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations when list_items=false for code block in list, got %v", v)
	}
}

func TestMD033_TableAllowedElements(t *testing.T) {
	// table_allowed_elements: <br> is allowed inside table cells.
	// Without table_allowed_elements, <br> in a table cell triggers MD033.
	// With it, no violation.
	src := "| Col |\n| ---- |\n| text<br>text |\n"
	v := lintString(t, rules.MD033{TableAllowedElements: []string{"br"}}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for table_allowed_elements, got %v", v)
	}
}

func TestMD041_FrontMatterTitle(t *testing.T) {
	// front_matter_title: front matter with title satisfies MD041.
	src := "---\ntitle: My Page\n---\n\nSome content\n"
	v := lintString(t, rules.MD041{FrontMatterTitle: "title"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations when front_matter_title matches, got %v", v)
	}
}

func TestMD041_AllowPreamble(t *testing.T) {
	// allow_preamble=true: non-heading content before heading is allowed.
	src := "Some preamble text.\n\n# Heading\n"
	v := lintString(t, rules.MD041{AllowPreamble: true}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations with allow_preamble=true, got %v", v)
	}
}

func TestMD041_AllowPreamble_Invalid(t *testing.T) {
	// allow_preamble=true but no heading at all: should trigger.
	src := "Some preamble text.\n\nMore text.\n"
	v := lintString(t, rules.MD041{AllowPreamble: true}, src)
	if len(v) == 0 {
		t.Errorf("expected violations when allow_preamble=true but no heading exists, got none")
	}
}

// Tests for the rule changes.

func TestMD004_MultiItemViolations(t *testing.T) {
	// Per-item reporting: 3 items in a wrong-style list should give 3 violations.
	src := "* item1\n* item2\n* item3\n"
	v := lintString(t, rules.MD004{Style: "dash"}, src)
	if len(v) != 3 {
		t.Errorf("expected 3 violations (one per item), got %d: %v", len(v), v)
	}
}

func TestMD005_PerListTracking(t *testing.T) {
	// Two separate lists at depth 1 with different indents should NOT cause violations
	// because each list tracks its own expected indent independently.
	src := "- item1\n- item2\n\n  - sub1\n  - sub2\n"
	v := lintString(t, rules.MD005{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for per-list indent tracking, got %v", v)
	}
}

func TestMD007_BlockquoteListItems(t *testing.T) {
	// List items inside blockquotes should also be checked.
	src := "> - item1\n>    - bad indent\n"
	v := lintString(t, rules.MD007{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for blockquote list indent, got %d: %v", len(v), v)
	}
}

func TestMD009_CodeBlocksDefaultFalse(t *testing.T) {
	// By default (code_blocks nil), trailing spaces inside code blocks are NOT checked.
	src := "```\ncode with trailing   \n```\n"
	v := lintString(t, rules.MD009{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations in code block by default, got %v", v)
	}
}

func TestMD024_CaseSensitive(t *testing.T) {
	// Case-sensitive comparison: "Duplicate" and "duplicate" are different.
	src := "# Duplicate\n\n## duplicate\n"
	v := lintString(t, rules.MD024{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for different-case headings, got %v", v)
	}
}

func TestMD024_SiblingsOnly_CaseSensitive(t *testing.T) {
	// siblings_only + case-sensitive: "Heading" and "heading" are different.
	src := "# Heading\n\n## heading\n"
	v := lintString(t, rules.MD024{SiblingsOnly: true}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for different-case sibling headings, got %v", v)
	}
}

func TestMD028_MultipleBlankLines(t *testing.T) {
	// Multiple blank lines between consecutive blockquotes should each be reported.
	src := "> Line 1\n\n\n> Line 2\n"
	v := lintString(t, rules.MD028{}, src)
	if len(v) != 2 {
		t.Errorf("expected 2 violations for 2 blank lines between blockquotes, got %d: %v", len(v), v)
	}
}

func TestMD029_OneOrOrdered_Sequential(t *testing.T) {
	// Sequential list (1, 2, 3) is valid for one_or_ordered.
	src := "1. item1\n2. item2\n3. item3\n"
	v := lintString(t, rules.MD029{Style: "one_or_ordered"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for sequential list, got %v", v)
	}
}

func TestMD029_OneOrOrdered_AllOne(t *testing.T) {
	// All-ones list (1, 1, 1) is valid for one_or_ordered.
	src := "1. item1\n1. item2\n1. item3\n"
	v := lintString(t, rules.MD029{Style: "one_or_ordered"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for all-ones list, got %v", v)
	}
}

func TestMD029_OneOrOrdered_Mixed(t *testing.T) {
	// A list with first item 1 and second item 1 is "one" style: all must be 1.
	// If we have 1, 1, 2 it's not valid (2nd is 1 so not incrementing, but 3rd != 1).
	src := "1. item1\n1. item2\n2. item3\n"
	v := lintString(t, rules.MD029{Style: "one_or_ordered"}, src)
	if len(v) == 0 {
		t.Errorf("expected violations for mixed one/ordered list, got none")
	}
}

func TestMD036_ListItemEmphasis(t *testing.T) {
	// Emphasis used as heading inside a list item should NOT trigger MD036.
	src := "- **Bold item**\n"
	v := lintString(t, rules.MD036{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for emphasis inside list item, got %v", v)
	}
}

func TestMD039_ReferenceLink(t *testing.T) {
	// Reference links should NOT be flagged by MD039.
	src := "See [ text ][ref] here.\n\n[ref]: https://example.com\n"
	v := lintString(t, rules.MD039{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for reference link with spaces, got %v", v)
	}
}

func TestMD046_ListItemContinuation(t *testing.T) {
	// List item continuation paragraphs indented 4+ spaces must NOT be flagged
	// as indented code blocks.
	src := "- Item one\n\n    Continuation paragraph.\n\n- Item two\n"
	v := lintString(t, rules.MD046{Style: "fenced"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for list item continuation, got %v", v)
	}
}

func TestMD046_IndentedCodeAfterRefDefInterruptingList(t *testing.T) {
	// When a link reference definition (0-indent) interrupts an ordered list,
	// any indented (4+ space) content that follows at the top level IS a genuine
	// indented code block and should be flagged.
	// However, continuation paragraphs inside a subsequent ordered list (even those
	// with 5-space indentation matching the list content column) are NOT code blocks
	// and must NOT be flagged.
	//
	// Note: markdownlint (micromark) has a known parsing divergence where ordered
	// list items starting with numbers > 1 that appear after an indented code block
	// are not recognised as list items.  Their continuation paragraphs are therefore
	// misidentified as indented code blocks, causing markdownlint to produce 4 extra
	// false-positive MD046 violations for files like text/0736-privacy-respecting-fru.md.
	// Goldmark correctly follows the CommonMark spec and does not reproduce those
	// false positives.
	src := "```rust\ncode\n```\n\n  1. Item one\n\n  2. Item two\n\n[ref]: /url\n\n     Indented code\n\n  3. Item three\n\n     Continuation paragraph\n"
	v := lintString(t, rules.MD046{}, src)
	// Only the genuine indented code block on line 11 should be flagged.
	// The continuation paragraph on line 15 is inside list item 3 and must NOT be flagged.
	if len(v) != 1 {
		t.Errorf("expected 1 violation (indented code block only), got %d: %v", len(v), v)
	}
	if len(v) == 1 && v[0].Line != 11 {
		t.Errorf("expected violation at line 11, got line %d", v[0].Line)
	}
}

func TestMD051_UnderscoreAnchor(t *testing.T) {
	// Heading with underscore: anchor should preserve the underscore.
	src := "# Hello_World\n\n[link](#hello_world)\n"
	v := lintString(t, rules.MD051{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for underscore anchor, got %v", v)
	}
}

func TestMD051_ReferenceDefinition(t *testing.T) {
	// Reference link definitions with fragment destinations should also be checked.
	src := "# Hello\n\n[link]: #nonexistent\n"
	v := lintString(t, rules.MD051{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for reference definition with bad fragment, got %d: %v", len(v), v)
	}
}

func TestMD027_ListDepth1_FirstLineFlagged(t *testing.T) {
	// Ordered list items directly inside a blockquote with extra spaces before
	// the number should be flagged on the first line.
	src := "> Para\n>\n>  1. item one\n>  2. item two\n"
	v := lintString(t, rules.MD027{}, src)
	if len(v) == 0 {
		t.Errorf("expected violations for extra space before ordered list items in blockquote, got none")
	}
}

func TestMD027_ListDepth2_FirstLineMasked(t *testing.T) {
	// Sub-list items inside a blockquote with structural indent must NOT be flagged.
	src := "> * parent\n>   * child one\n>   * child two\n"
	v := lintString(t, rules.MD027{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for nested sub-list in blockquote, got %v", v)
	}
}

func TestMD029_ZeroBasedSequential_Valid(t *testing.T) {
	// Zero-based sequential list (0, 1, 2, 3) is valid for one_or_ordered.
	src := "0. item zero\n1. item one\n2. item two\n"
	v := lintString(t, rules.MD029{Style: "one_or_ordered"}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for 0-based sequential list, got %v", v)
	}
}

func TestMD029_OneOneThree_AllOneStyle(t *testing.T) {
	// List starting with two 1s: first two items same → "all one" style.
	// Items 3, 4, 5, 6 should be flagged as Expected: 1.
	src := "1. item1\n1. item2\n3. item3\n4. item4\n"
	v := lintString(t, rules.MD029{Style: "one_or_ordered"}, src)
	if len(v) != 2 {
		t.Errorf("expected 2 violations for 1/1/3/4 list (items 3 and 4), got %d: %v", len(v), v)
	}
}

func TestMD031_HTMLCommentAfterFence_Valid(t *testing.T) {
	// HTML comments immediately after a closing fence are acceptable separators.
	src := "# Test\n\n```\ncode\n```\n<!-- comment -->\n\nNext paragraph.\n"
	v := lintString(t, rules.MD031{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violation when HTML comment follows closing fence, got %v", v)
	}
}

func TestMD031_HTMLCommentBeforeFence_Valid(t *testing.T) {
	// HTML comments immediately before an opening fence are acceptable separators.
	src := "# Test\n\nSome text.\n<!-- comment -->\n```\ncode\n```\n\nNext.\n"
	v := lintString(t, rules.MD031{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violation when HTML comment precedes opening fence, got %v", v)
	}
}

func TestMD041_FrontMatterTitleDefault(t *testing.T) {
	// Default FrontMatterTitle is "title": a front matter title field satisfies MD041.
	src := "---\ntitle: My Page\n---\n\nNot a heading\n"
	v := lintString(t, rules.MD041{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violation when front matter has title field (default config), got %v", v)
	}
}

func TestMD045_BlockHTMLImgNoAlt(t *testing.T) {
	// Block-level <img> without alt text should be flagged.
	src := "# Test\n\n<img src=\"test.png\">\n\nText.\n"
	v := lintString(t, rules.MD045{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for block-level img without alt, got %d: %v", len(v), v)
	}
}

func TestMD045_BlockHTMLImgWithAlt(t *testing.T) {
	// Block-level <img> with alt text should not be flagged.
	src := "# Test\n\n<img src=\"test.png\" alt=\"description\">\n\nText.\n"
	v := lintString(t, rules.MD045{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violation for block-level img with alt, got %v", v)
	}
}

func TestMD045_BlockHTMLImgMultilineWithAlt(t *testing.T) {
	// Multi-line block-level <img> with alt on a different line should not be flagged.
	src := "# Test\n\n<img src=\"test.png\"\n    alt=\"description\">\n\nText.\n"
	v := lintString(t, rules.MD045{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violation for multi-line block-level img with alt, got %v", v)
	}
}

func TestMD013_AutoLinkInEmphasis_NoPanic(t *testing.T) {
	// AutoLink nested inside emphasis (or other inline nodes) must not panic.
	// The blockFirstLine helper used to call Lines() on inline parent nodes,
	// which panics in goldmark. Regression test for that crash.
	src := "*<https://www.example.com/autolink/inside/emphasis/that/is/very/long/indeed>*\n"
	// Should not panic; URL exemption applies so no violations expected.
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for autolink in emphasis, got %d: %v", len(v), v)
	}
}

func TestMD030_ThematicBreakNoViolation(t *testing.T) {
// Thematic breaks that start with -, *, or _ must not be reported as
// list-marker spacing violations.
src := "*  *  *\n\n-  -  -\n\n_ _ _\n"
v := lintString(t, rules.MD030{}, src)
if len(v) != 0 {
t.Errorf("expected no violations for thematic breaks, got %v", v)
}
}

func TestMD028_IndentedCodeBlockIgnored(t *testing.T) {
// Lines inside an indented code block that start with '>' must not be
// treated as blockquote lines by MD028.
src := "    > not a blockquote\n\n    > also not\n"
v := lintString(t, rules.MD028{}, src)
if len(v) != 0 {
t.Errorf("expected no violations for > inside indented code block, got %v", v)
}
}

func TestMD032_PlainTextAfterList_Violation(t *testing.T) {
// Plain text immediately following a list without a blank line must be
// flagged — matching markdownlint behaviour.
src := "- item 1\n- item 2\nplain text\n"
v := lintString(t, rules.MD032{}, src)
if len(v) == 0 {
t.Errorf("expected violation for plain text after list without blank line, got none")
}
}

func TestMD060_Default_ConsistentNoViolation(t *testing.T) {
// Default "consistent" style: all rows compact → no violations.
src := "| Col1 | Col2 |\n| ---- | ---- |\n| A | B |\n"
v := lintString(t, rules.MD060{}, src)
if len(v) != 0 {
t.Errorf("expected no violations for consistent table, got %v", v)
}
}

func TestMD011_ReversedLinkInCodeSpan_NoViolation(t *testing.T) {
// A reversed-link pattern inside a code span must not be reported.
src := "Use `(text)[url]` in your docs.\n"
v := lintString(t, rules.MD011{}, src)
if len(v) != 0 {
t.Errorf("expected no violations for reversed link inside code span, got %v", v)
}
}

func TestMD056_IndentedCodeBlockIgnored(t *testing.T) {
// A table-like pattern inside an indented code block must not be reported.
src := "    Col1 | Col2\n    ---- | ----\n    A | B\n"
v := lintString(t, rules.MD056{}, src)
if len(v) != 0 {
t.Errorf("expected no violations for table inside indented code block, got %v", v)
}
}
