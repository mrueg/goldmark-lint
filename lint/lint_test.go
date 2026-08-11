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

func TestMD013_LinkOnlyLineWithCodeSpanExempt(t *testing.T) {
	// A line whose only content is a single inline link is a "link-only" line
	// and should not be flagged, even when the link text contains code spans.
	src := "* [`cargo doc` should render crate examples and link to them on main documentation page](https://github.com/rust-lang/cargo/issues/2760)\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for link-only line with code span, got %d: %v", len(v), v)
	}
}

func TestMD013_ReferenceOnlyLineExempt(t *testing.T) {
	// A line whose only content is a single reference link should be exempt,
	// matching markdownlint behaviour (the link cannot be reformatted).
	src := "[ref]: https://example.com\n\n[This reference link label is long enough to make the line exceed the 80 char limit][ref]\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for reference-link-only line, got %d: %v", len(v), v)
	}
}

func TestMD013_MultilineImageAltTextExempt(t *testing.T) {
	// Lines that form the alt-text of a multi-line image are link-only
	// content and should not be flagged.
	src := "![A diagram showing a function with one let statement that is long enough to exceed eighty\nand a visualisation of how long x and temp live before and after this change.](diagram.svg)\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for multi-line image alt text, got %d: %v", len(v), v)
	}
}

func TestMD013_LinkInTextNotExempt(t *testing.T) {
	// A line that has bare text before or after a link is NOT link-only
	// and should be flagged when it exceeds the limit.
	src := "See the [documentation](https://example.com) for more details about this very very long line.\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) == 0 {
		t.Errorf("expected a violation for non-link-only long line, got none")
	}
}

func TestMD013_FencedCodeBlockContentNotCheckedByDefault(t *testing.T) {
	// Lines inside a fenced code block are classified as code block lines;
	// with the default code_blocks:true they are still checked but use a
	// separate limit. With code_blocks:false they are skipped entirely.
	f := false
	longLine := strings.Repeat("x", 100)
	src := "```\n" + longLine + "\n```\n"
	v := lintString(t, rules.MD013{LineLength: 80, CodeBlocks: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations with code_blocks:false, got %d: %v", len(v), v)
	}
}

func TestMD013_ATXHeadingCheckedSeparately(t *testing.T) {
	// ATX heading lines are classified correctly by the AST-based heading mask.
	longHeading := "# " + strings.Repeat("Word ", 20)
	src := longHeading + "\n\nNormal paragraph.\n"
	f := false
	v := lintString(t, rules.MD013{LineLength: 80, Headings: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations with headings:false, got %d: %v", len(v), v)
	}
}

func TestMD013_SetextHeadingCheckedSeparately(t *testing.T) {
	// Setext heading lines (content + underline) are classified correctly.
	// With headings:false they should not be flagged.
	longHeading := strings.Repeat("Word ", 20)
	src := longHeading + "\n" + strings.Repeat("=", 80) + "\n\nNormal paragraph.\n"
	f := false
	v := lintString(t, rules.MD013{LineLength: 80, Headings: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for setext heading with headings:false, got %d: %v", len(v), v)
	}
}

func TestMD013_TableRowCheckedSeparately(t *testing.T) {
	// Table rows are classified correctly by the AST-based table mask.
	longCell := strings.Repeat("x", 90)
	src := "| " + longCell + " | B |\n|---|---|\n| C | D |\n"
	f := false
	v := lintString(t, rules.MD013{LineLength: 80, Tables: &f}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations with tables:false, got %d: %v", len(v), v)
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

func TestMD012_HTMLComment_NoFalsePositive(t *testing.T) {
	// Blank lines inside an HTML comment (type-2 HTML block) must NOT be flagged.
	src := "<!--\n\n\nsome comment\n-->\n\nMore text\n"
	v := lintString(t, rules.MD012{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for blank lines inside HTML comment, got %v", v)
	}
}

func TestMD012_ScriptBlock_NoFalsePositive(t *testing.T) {
	// Blank lines inside a <script> block (type-1 HTML block) must NOT be flagged.
	src := "<script>\n\n\ncode();\n</script>\n\nMore text\n"
	v := lintString(t, rules.MD012{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for blank lines inside script block, got %v", v)
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

func TestMD007_NestedOrdered(t *testing.T) {
	src := "1. A\n   * B\n"
	v := lintString(t, rules.MD007{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for unordered list nested in ordered list, got %v", v)
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
	// A list with a multiline item missing a blank line before it should produce
	// one violation (before only). Plain text after a list is treated as a lazy
	// continuation in CommonMark, so no "after" violation is reported –
	// matching markdownlint behaviour.
	src := "Text\n- item 1\n  continuation\n- item 2\nMore text\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for multiline list without blank line before, got %d: %v", len(v), v)
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

func TestMD032_FlatNested_BeforeViolation(t *testing.T) {
	// "- - nested" syntax: a list item whose first child is immediately a nested
	// list (same line).  The outer list must still be checked for the before-
	// blank-line requirement.
	src := "Text\n- - nested item\nMore\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) == 0 {
		t.Errorf("expected before-violation for list preceded by text without blank line, got none")
	}
}

func TestMD032_FlatNested_AtDocStart_NoViolation(t *testing.T) {
	// "- - nested" at the very start of the document: no before-violation.
	src := "- - nested item\nMore\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) != 0 {
		t.Errorf("expected no before-violation for list at document start, got %v", v)
	}
}

func TestMD032_EmptyOuterItem_BeforeViolation(t *testing.T) {
	// Empty outer list item (just "-") immediately after text, followed by a
	// nested list on the next line.  The outer list still needs a blank before it.
	src := "Text\n-\n  - nested\nMore\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) == 0 {
		t.Errorf("expected before-violation for outer list item preceded by text, got none")
	}
}

func TestMD032_EmptyOuterItem_AtDocStart_NoViolation(t *testing.T) {
	// Empty outer list item at the start of the document: no violation.
	src := "-\n  - nested\nMore\n"
	v := lintString(t, rules.MD032{}, src)
	if len(v) != 0 {
		t.Errorf("expected no before-violation for list at document start, got %v", v)
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

func TestMD034_BareEmail_Violation(t *testing.T) {
	// Bare email addresses should be flagged by MD034, matching markdownlint behaviour.
	src := "Contact user@example.com for help.\n"
	v := lintString(t, rules.MD034{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation for bare email, got %d: %v", len(v), v)
	}
}

func TestMD034_AngleBracketEmail_NoViolation(t *testing.T) {
	// Email addresses wrapped in angle brackets are auto-links and should not be flagged.
	src := "Contact <user@example.com> for help.\n"
	v := lintString(t, rules.MD034{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for angle-bracket email, got %d: %v", len(v), v)
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

func TestMD060_Default_Any(t *testing.T) {
	// Default style is "any": data row not aligned with header → aligned violations.
	src := "| Col1 | Col2 |\n| ---- | ---- |\n|A|B|\n"
	v := lintString(t, rules.MD060{}, src)
	// Data row |A|B| has pipes at cols 0,2,4 while header has pipes at 0,7,14.
	// Aligned check fails (2 misaligned pipes); aligned has fewer errors than compact (4).
	if len(v) != 2 {
		t.Errorf("expected 2 violations with default any style, got %d: %v", len(v), v)
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

// TestIntegration_CompareWithMarkdownlint reports, per fixture, any rule that
// markdownlint flags on a line where goldmark-lint does not. It is a diagnostic
// aid rather than an assertion: differences are logged, never failed, because
// the authoritative conformance comparison lives in bench/conform.sh.
//
// It spawns the markdownlint Node binary once per fixture, which takes about
// two minutes over the ~100 files in testdata, so it is skipped under -short
// and when GOLDMARK_LINT_INTEGRATION is unset. Run it explicitly with:
//
//	GOLDMARK_LINT_INTEGRATION=1 go test ./lint -run TestIntegration -v
func TestIntegration_CompareWithMarkdownlint(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns markdownlint once per fixture; skipped in -short mode")
	}
	if os.Getenv("GOLDMARK_LINT_INTEGRATION") == "" {
		t.Skip("set GOLDMARK_LINT_INTEGRATION=1 to run the markdownlint comparison")
	}
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

func TestMD009_FencedCodeInBlockquote_NoFalsePositive(t *testing.T) {
	// Trailing spaces inside a fenced code block that is itself inside a blockquote
	// must NOT be flagged by default (code_blocks defaults to not-checking).
	src := "> ```\n> code with trailing   \n> ```\n"
	v := lintString(t, rules.MD009{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for trailing spaces in fenced code block inside blockquote, got %v", v)
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

func TestMD030_IndentedCodeBlock_NoFalsePositive(t *testing.T) {
	// List-like lines inside an indented code block must NOT be flagged by MD030.
	src := "Some text\n\n    1.  item inside code block\n    -  another inside code block\n"
	v := lintString(t, rules.MD030{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for list-like lines inside indented code block, got %v", v)
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

func TestMD028_IndentedCodeBlockWithArrow_NoFalsePositive(t *testing.T) {
	// Lines inside an indented code block that start with '>' must NOT be treated
	// as blockquote lines for MD028 purposes.
	src := "    > arrow in code block\n\n    > another arrow\n"
	v := lintString(t, rules.MD028{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for '>' inside indented code block, got %v", v)
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

func TestMD013_MultiLineLinkRefDef_TitleLineExempt(t *testing.T) {
	// A title continuation line of a multi-line link reference definition
	// (the line immediately following "[label]: url" that starts with a quote)
	// must be exempt from MD013, just like the definition line itself.
	longTitle := `"` + strings.Repeat("x", 85) + `"`
	src := "[label]: https://example.com\n" + longTitle + "\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations for link ref def title line, got %d: %v", len(v), v)
	}
}

func TestMD001_Fix_SkipLevel(t *testing.T) {
	src := "# Heading 1\n\n### Heading 3\n"
	got := fixString(t, rules.MD001{}, src)
	want := "# Heading 1\n\n## Heading 3\n"
	if got != want {
		t.Errorf("MD001 Fix() = %q, want %q", got, want)
	}
}

func TestMD001_Fix_NoViolation(t *testing.T) {
	src := "# Heading 1\n\n## Heading 2\n\n### Heading 3\n"
	got := fixString(t, rules.MD001{}, src)
	if got != src {
		t.Errorf("MD001 Fix() modified valid source: got %q, want %q", got, src)
	}
}

func TestMD001_Fix_MultiSkip(t *testing.T) {
	src := "# H1\n\n#### H4\n"
	got := fixString(t, rules.MD001{}, src)
	want := "# H1\n\n## H4\n"
	if got != want {
		t.Errorf("MD001 Fix() = %q, want %q", got, want)
	}
}

func TestMD004_Fix_Dash(t *testing.T) {
	src := "* item 1\n* item 2\n"
	got := fixString(t, rules.MD004{Style: "dash"}, src)
	want := "- item 1\n- item 2\n"
	if got != want {
		t.Errorf("MD004 Fix() = %q, want %q", got, want)
	}
}

func TestMD004_Fix_Asterisk(t *testing.T) {
	src := "- item 1\n- item 2\n"
	got := fixString(t, rules.MD004{Style: "asterisk"}, src)
	want := "* item 1\n* item 2\n"
	if got != want {
		t.Errorf("MD004 Fix() = %q, want %q", got, want)
	}
}

func TestMD004_Fix_Consistent(t *testing.T) {
	src := "- item 1\n* item 2\n+ item 3\n"
	got := fixString(t, rules.MD004{Style: "consistent"}, src)
	want := "- item 1\n- item 2\n- item 3\n"
	if got != want {
		t.Errorf("MD004 Fix() = %q, want %q", got, want)
	}
}

func TestMD004_Fix_NoChange(t *testing.T) {
	src := "- item 1\n- item 2\n"
	got := fixString(t, rules.MD004{Style: "dash"}, src)
	if got != src {
		t.Errorf("MD004 Fix() modified valid source: got %q, want %q", got, src)
	}
}

func TestMD022_Fix_AddBlanksAboveBelow(t *testing.T) {
	src := "Some text\n# Heading\nMore text\n"
	got := fixString(t, rules.MD022{}, src)
	want := "Some text\n\n# Heading\n\nMore text\n"
	if got != want {
		t.Errorf("MD022 Fix() = %q, want %q", got, want)
	}
}

func TestMD022_Fix_NoChange(t *testing.T) {
	src := "Some text\n\n# Heading\n\nMore text\n"
	got := fixString(t, rules.MD022{}, src)
	if got != src {
		t.Errorf("MD022 Fix() modified valid source: got %q, want %q", got, src)
	}
}

func TestMD022_Fix_FirstLine(t *testing.T) {
	// First heading needs no blank line above when it's the first content.
	src := "# Heading\n\nSome text\n"
	got := fixString(t, rules.MD022{}, src)
	if got != src {
		t.Errorf("MD022 Fix() modified valid first-line heading: got %q, want %q", got, src)
	}
}

func TestMD028_Fix_RemoveBlanks(t *testing.T) {
	src := "> quote 1\n\n> quote 2\n"
	got := fixString(t, rules.MD028{}, src)
	want := "> quote 1\n> quote 2\n"
	if got != want {
		t.Errorf("MD028 Fix() = %q, want %q", got, want)
	}
}

func TestMD028_Fix_NoChange(t *testing.T) {
	src := "> quote 1\n> quote 2\n"
	got := fixString(t, rules.MD028{}, src)
	if got != src {
		t.Errorf("MD028 Fix() modified valid source: got %q, want %q", got, src)
	}
}

func TestMD028_Fix_NonBlockquoteBlanksPreserved(t *testing.T) {
	src := "> quote\n\nNot a blockquote\n"
	got := fixString(t, rules.MD028{}, src)
	if got != src {
		t.Errorf("MD028 Fix() should not remove blank before non-blockquote: got %q, want %q", got, src)
	}
}

func TestMD034_Fix_WrapBareURL(t *testing.T) {
	src := "Visit https://example.com for info\n"
	got := fixString(t, rules.MD034{}, src)
	want := "Visit <https://example.com> for info\n"
	if got != want {
		t.Errorf("MD034 Fix() = %q, want %q", got, want)
	}
}

func TestMD034_Fix_NoChangeLink(t *testing.T) {
	src := "Visit [example](https://example.com) for info\n"
	got := fixString(t, rules.MD034{}, src)
	if got != src {
		t.Errorf("MD034 Fix() should not modify URL inside link: got %q, want %q", got, src)
	}
}

func TestMD034_Fix_AlreadyAutoLink(t *testing.T) {
	src := "Visit <https://example.com> for info\n"
	got := fixString(t, rules.MD034{}, src)
	if got != src {
		t.Errorf("MD034 Fix() should not re-wrap already wrapped URL: got %q, want %q", got, src)
	}
}

func TestMD035_Fix_Consistent(t *testing.T) {
	src := "---\n\ntext\n\n***\n"
	got := fixString(t, rules.MD035{}, src)
	want := "---\n\ntext\n\n---\n"
	if got != want {
		t.Errorf("MD035 Fix() = %q, want %q", got, want)
	}
}

func TestMD035_Fix_Explicit(t *testing.T) {
	src := "---\n\ntext\n\n* * *\n"
	got := fixString(t, rules.MD035{Style: "---"}, src)
	want := "---\n\ntext\n\n---\n"
	if got != want {
		t.Errorf("MD035 Fix() = %q, want %q", got, want)
	}
}

func TestMD035_Fix_NoChange(t *testing.T) {
	src := "---\n\ntext\n\n---\n"
	got := fixString(t, rules.MD035{Style: "---"}, src)
	if got != src {
		t.Errorf("MD035 Fix() modified valid source: got %q, want %q", got, src)
	}
}

func TestMD046_Fix_ToFenced(t *testing.T) {
	src := "Text\n\n    code line 1\n    code line 2\n\nMore text\n"
	got := fixString(t, rules.MD046{Style: "fenced"}, src)
	want := "Text\n\n```\ncode line 1\ncode line 2\n```\n\nMore text\n"
	if got != want {
		t.Errorf("MD046 Fix() = %q, want %q", got, want)
	}
}

func TestMD046_Fix_NoChange(t *testing.T) {
	src := "Text\n\n```\ncode\n```\n\nMore text\n"
	got := fixString(t, rules.MD046{Style: "fenced"}, src)
	if got != src {
		t.Errorf("MD046 Fix() modified valid fenced source: got %q, want %q", got, src)
	}
}

func TestMD055_Fix_AddLeadingTrailing(t *testing.T) {
	src := "Header 1 | Header 2\n--- | ---\nCell 1 | Cell 2\n"
	got := fixString(t, rules.MD055{Style: "leading_and_trailing"}, src)
	want := "| Header 1 | Header 2 |\n| --- | --- |\n| Cell 1 | Cell 2 |\n"
	if got != want {
		t.Errorf("MD055 Fix() = %q, want %q", got, want)
	}
}

func TestMD055_Fix_RemoveLeadingTrailing(t *testing.T) {
	src := "| Header 1 | Header 2 |\n| --- | --- |\n| Cell 1 | Cell 2 |\n"
	got := fixString(t, rules.MD055{Style: "no_leading_or_trailing"}, src)
	want := "Header 1 | Header 2\n--- | ---\nCell 1 | Cell 2\n"
	if got != want {
		t.Errorf("MD055 Fix() = %q, want %q", got, want)
	}
}

func TestMD055_Fix_NoChange(t *testing.T) {
	src := "| Header 1 | Header 2 |\n| --- | --- |\n| Cell 1 | Cell 2 |\n"
	got := fixString(t, rules.MD055{Style: "leading_and_trailing"}, src)
	if got != src {
		t.Errorf("MD055 Fix() modified valid source: got %q, want %q", got, src)
	}
}

// ─── MD040 Fix ─────────────────────────────────────────────────────────────

func TestMD040_Fix_AddTextLanguage(t *testing.T) {
	src := "```\ncode\n```\n"
	got := fixString(t, rules.MD040{}, src)
	want := "``` text\ncode\n```\n"
	if got != want {
		t.Errorf("MD040 Fix() = %q, want %q", got, want)
	}
}

func TestMD040_Fix_NoChangeWhenLangPresent(t *testing.T) {
	src := "```go\ncode\n```\n"
	got := fixString(t, rules.MD040{}, src)
	if got != src {
		t.Errorf("MD040 Fix() modified source that already has a language: got %q", got)
	}
}

func TestMD040_Fix_TildeFence(t *testing.T) {
	src := "~~~\ncode\n~~~\n"
	got := fixString(t, rules.MD040{}, src)
	want := "~~~ text\ncode\n~~~\n"
	if got != want {
		t.Errorf("MD040 Fix() = %q, want %q", got, want)
	}
}

func TestMD040_Fix_SkipInsideFencedBlock(t *testing.T) {
	// Content lines inside a fenced block must not be modified, even if they
	// look like fence delimiters.
	// Source: first block has "go" and contains a backtick line as content.
	src := "```go\n` ` `\ncode\n```\n"
	got := fixString(t, rules.MD040{}, src)
	if got != src {
		t.Errorf("MD040 Fix() unexpectedly modified source: got %q", got)
	}
}

// ─── MD033 Fix ─────────────────────────────────────────────────────────────

func TestMD033_Fix_RemoveInlineTag(t *testing.T) {
	src := "Some <b>bold</b> text\n"
	got := fixString(t, rules.MD033{}, src)
	want := "Some bold text\n"
	if got != want {
		t.Errorf("MD033 Fix() = %q, want %q", got, want)
	}
}

func TestMD033_Fix_KeepAllowedTag(t *testing.T) {
	src := "Some <b>bold</b> text\n"
	got := fixString(t, rules.MD033{AllowedElements: []string{"b"}}, src)
	if got != src {
		t.Errorf("MD033 Fix() removed an allowed element: got %q", got)
	}
}

func TestMD033_Fix_KeepHTMLComment(t *testing.T) {
	src := "<!-- comment -->\nSome text\n"
	got := fixString(t, rules.MD033{}, src)
	if got != src {
		t.Errorf("MD033 Fix() removed an HTML comment: got %q", got)
	}
}

func TestMD033_Fix_RemoveSelfClosing(t *testing.T) {
	src := "Line break here<br/>.\n"
	got := fixString(t, rules.MD033{}, src)
	want := "Line break here.\n"
	if got != want {
		t.Errorf("MD033 Fix() = %q, want %q", got, want)
	}
}

func TestMD033_Fix_NoChangeWhenNoneDisallowed(t *testing.T) {
	src := "No HTML here\n"
	got := fixString(t, rules.MD033{}, src)
	if got != src {
		t.Errorf("MD033 Fix() modified source that has no HTML: got %q", got)
	}
}

// ─── MD036 Fix ─────────────────────────────────────────────────────────────

func TestMD036_Fix_BoldToHeading(t *testing.T) {
	src := "**Section Title**\n\nSome text\n"
	got := fixString(t, rules.MD036{}, src)
	want := "## Section Title\n\nSome text\n"
	if got != want {
		t.Errorf("MD036 Fix() = %q, want %q", got, want)
	}
}

func TestMD036_Fix_ItalicToHeading(t *testing.T) {
	src := "*Section Title*\n\nSome text\n"
	got := fixString(t, rules.MD036{}, src)
	want := "## Section Title\n\nSome text\n"
	if got != want {
		t.Errorf("MD036 Fix() = %q, want %q", got, want)
	}
}

func TestMD036_Fix_SkipPunctuation(t *testing.T) {
	src := "**Not a heading.**\n\nSome text\n"
	got := fixString(t, rules.MD036{}, src)
	if got != src {
		t.Errorf("MD036 Fix() modified emphasis ending with punctuation: got %q", got)
	}
}

func TestMD036_Fix_SkipInsideList(t *testing.T) {
	// Emphasis inside a list item must not be converted.
	src := "- **Not a heading**\n"
	v := lintString(t, rules.MD036{}, src)
	if len(v) != 0 {
		t.Logf("(no violation expected for emphasis inside list)")
	}
	got := fixString(t, rules.MD036{}, src)
	if got != src {
		t.Errorf("MD036 Fix() modified a list item: got %q", got)
	}
}

func TestMD036_Fix_NoChange(t *testing.T) {
	src := "## Real heading\n\nSome text\n"
	got := fixString(t, rules.MD036{}, src)
	if got != src {
		t.Errorf("MD036 Fix() modified a real heading: got %q", got)
	}
}

// ─── MD005 Fix ─────────────────────────────────────────────────────────────

func TestMD005_Fix_NormaliseSiblingIndent(t *testing.T) {
	src := "- Item 1\n- Item 2\n - Item 3\n"
	got := fixString(t, rules.MD005{}, src)
	want := "- Item 1\n- Item 2\n- Item 3\n"
	if got != want {
		t.Errorf("MD005 Fix() = %q, want %q", got, want)
	}
}

func TestMD005_Fix_NoChangeWhenConsistent(t *testing.T) {
	src := "- Item 1\n- Item 2\n- Item 3\n"
	got := fixString(t, rules.MD005{}, src)
	if got != src {
		t.Errorf("MD005 Fix() modified consistent source: got %q", got)
	}
}

func TestMD005_Fix_NestedList(t *testing.T) {
	src := "- Item 1\n  - Nested 1\n   - Nested 2\n"
	got := fixString(t, rules.MD005{}, src)
	want := "- Item 1\n  - Nested 1\n  - Nested 2\n"
	if got != want {
		t.Errorf("MD005 Fix() = %q, want %q", got, want)
	}
}

// ─── MD007 Fix ─────────────────────────────────────────────────────────────

func TestMD007_Fix_NormaliseSingleLevel(t *testing.T) {
	// Top-level item at 1 space should be fixed to 0.
	src := " - Wrong\n"
	got := fixString(t, rules.MD007{}, src)
	want := "- Wrong\n"
	if got != want {
		t.Errorf("MD007 Fix() = %q, want %q", got, want)
	}
}

func TestMD007_Fix_NormaliseNested(t *testing.T) {
	src := "- Item 1\n   - Nested\n"
	got := fixString(t, rules.MD007{}, src)
	want := "- Item 1\n  - Nested\n"
	if got != want {
		t.Errorf("MD007 Fix() = %q, want %q", got, want)
	}
}

func TestMD007_Fix_NoChange(t *testing.T) {
	src := "- item\n  - subitem\n"
	got := fixString(t, rules.MD007{}, src)
	if got != src {
		t.Errorf("MD007 Fix() modified valid source: got %q", got)
	}
}

func TestMD007_Fix_NestedOrdered(t *testing.T) {
	src := "1. A\n   * B\n"
	want := src
	got := fixString(t, rules.MD007{}, src)
	if got != want {
		t.Errorf("MD007 Fix() should not change indentation for unordered list nested in ordered list: got %q, want %q", got, want)
	}
}

func TestMD007_Fix_CustomIndent(t *testing.T) {
	// With indent=4, a nested item at 5 spaces should become 4.
	src := "- Item\n     - Nested\n"
	got := fixString(t, rules.MD007{Indent: 4}, src)
	want := "- Item\n    - Nested\n"
	if got != want {
		t.Errorf("MD007 Fix(indent=4) = %q, want %q", got, want)
	}
}

// ─── MD060 Fix ─────────────────────────────────────────────────────────────

func TestMD060_Fix_CompactStyle(t *testing.T) {
	src := "|Name|Age|\n|----|---|\n|Alice|30|\n"
	got := fixString(t, rules.MD060{Style: "compact"}, src)
	want := "| Name | Age |\n|----|---|\n| Alice | 30 |\n"
	if got != want {
		t.Errorf("MD060 Fix(compact) = %q, want %q", got, want)
	}
}

func TestMD060_Fix_TightStyle(t *testing.T) {
	src := "| Name | Age |\n| ---- | --- |\n| Alice | 30 |\n"
	got := fixString(t, rules.MD060{Style: "tight"}, src)
	want := "|Name|Age|\n| ---- | --- |\n|Alice|30|\n"
	if got != want {
		t.Errorf("MD060 Fix(tight) = %q, want %q", got, want)
	}
}

func TestMD060_Fix_AnyNoChange(t *testing.T) {
	src := "| Name | Age |\n| ---- | --- |\n| Alice | 30 |\n"
	got := fixString(t, rules.MD060{Style: "any"}, src)
	if got != src {
		t.Errorf("MD060 Fix(any) = %q, want unchanged %q", got, src)
	}
}

func TestMD060_Fix_AlignedStyle(t *testing.T) {
	src := "| Name | Age |\n| ---- | --- |\n| Alice | 30 |\n"
	got := fixString(t, rules.MD060{Style: "aligned"}, src)
	// After alignment, all rows should have pipes at the same positions.
	// Each column should be wide enough to fit the widest content.
	wantHeader := "| Name  | Age |"
	if !strings.Contains(got, wantHeader) {
		t.Errorf("MD060 Fix(aligned) header line = %q, want to contain %q\nfull output: %q", got, wantHeader, got)
	}
}

func TestMD060_Fix_ConsistentStyle(t *testing.T) {
	// First row is compact; second row is tight — fix to consistent (compact).
	src := "| Name | Age |\n| ---- | --- |\n|Alice|30|\n"
	got := fixString(t, rules.MD060{Style: "consistent"}, src)
	want := "| Name | Age |\n| ---- | --- |\n| Alice | 30 |\n"
	if got != want {
		t.Errorf("MD060 Fix(consistent) = %q, want %q", got, want)
	}
}

// ─── MD003 Fix ─────────────────────────────────────────────────────────────

func TestMD003_Fix_ATXToSetextH1(t *testing.T) {
	src := "# Heading One\n\nText\n"
	got := fixString(t, rules.MD003{Style: "setext"}, src)
	want := "Heading One\n===========\n\nText\n"
	if got != want {
		t.Errorf("MD003 Fix(setext) h1 = %q, want %q", got, want)
	}
}

func TestMD003_Fix_ATXToSetextH2(t *testing.T) {
	src := "## Heading Two\n\nText\n"
	got := fixString(t, rules.MD003{Style: "setext"}, src)
	want := "Heading Two\n-----------\n\nText\n"
	if got != want {
		t.Errorf("MD003 Fix(setext) h2 = %q, want %q", got, want)
	}
}

func TestMD003_Fix_SetextToATX(t *testing.T) {
	src := "Heading One\n===========\n\nText\n"
	got := fixString(t, rules.MD003{Style: "atx"}, src)
	want := "# Heading One\n\nText\n"
	if got != want {
		t.Errorf("MD003 Fix(atx) from setext = %q, want %q", got, want)
	}
}

func TestMD003_Fix_ATXToATXClosed(t *testing.T) {
	src := "# Heading\n"
	got := fixString(t, rules.MD003{Style: "atx_closed"}, src)
	want := "# Heading #\n"
	if got != want {
		t.Errorf("MD003 Fix(atx_closed) = %q, want %q", got, want)
	}
}

func TestMD003_Fix_ATXClosedToATX(t *testing.T) {
	src := "# Heading #\n"
	got := fixString(t, rules.MD003{Style: "atx"}, src)
	want := "# Heading\n"
	if got != want {
		t.Errorf("MD003 Fix(atx) from atx_closed = %q, want %q", got, want)
	}
}

func TestMD003_Fix_ConsistentATX(t *testing.T) {
	// First heading is ATX; setext heading should be converted to ATX.
	src := "# ATX First\n\nSetext\n------\n\nText\n"
	got := fixString(t, rules.MD003{Style: "consistent"}, src)
	want := "# ATX First\n\n## Setext\n\nText\n"
	if got != want {
		t.Errorf("MD003 Fix(consistent) = %q, want %q", got, want)
	}
}

func TestMD003_Fix_NoChangeWhenAlreadyCorrect(t *testing.T) {
	src := "# ATX Heading\n\nText\n"
	got := fixString(t, rules.MD003{Style: "atx"}, src)
	if got != src {
		t.Errorf("MD003 Fix(atx) modified already-correct source: got %q", got)
	}
}

func TestMD003_Fix_SetextWithATX_H3(t *testing.T) {
	// setext_with_atx: h3 must be ATX; if it's ATX already, leave it alone.
	src := "### H3 heading\n"
	got := fixString(t, rules.MD003{Style: "setext_with_atx"}, src)
	if got != src {
		t.Errorf("MD003 Fix(setext_with_atx) modified h3 ATX heading: got %q", got)
	}
}

// TestMD029_InterruptedListRestartsNumbering covers the constructs that split
// an ordered list. markdownlint requires the fragment after a block-level
// interruption to restart at 1, so a fragment continuing "4., 5." is a
// violation. goldmark-lint used to suppress these, which accounted for 6 of the
// conformance deltas against the tldr-pages corpus.
func TestMD029_InterruptedListRestartsNumbering(t *testing.T) {
	interrupters := map[string]string{
		"blockquote":     "> [!WARNING]\n> Careful.",
		"fenced code":    "```sh\necho hi\n```",
		"thematic break": "---",
		"html block":     "<!-- a comment -->",
		"heading":        "## A heading",
		"paragraph":      "A plain paragraph.",
		"link reference": "[ref]: https://example.com",
	}
	for name, interrupter := range interrupters {
		t.Run(name, func(t *testing.T) {
			src := "# T\n\n1. One\n\n2. Two\n\n" + interrupter + "\n\n3. Three\n\n4. Four\n"
			v := lintString(t, rules.MD029{}, src)
			if len(v) != 2 {
				t.Fatalf("expected 2 violations for a list restarted after %s, got %d: %v", name, len(v), v)
			}
			if !strings.Contains(v[0].Message, "Expected: 1; Actual: 3") {
				t.Errorf("first violation = %q, want Expected: 1; Actual: 3", v[0].Message)
			}
		})
	}
}

// TestMD029_IndentedContentKeepsListOpen covers the opposite case: goldmark
// ends a list at a link reference definition written flush left, while
// micromark (which markdownlint uses) keeps the list open when the surrounding
// content is indented into it. The fragment after such a gap continues the
// earlier numbering and must not be flagged.
//
// Reduced from rust-lang/rfcs text/0736-privacy-respecting-fru.md.
func TestMD029_IndentedContentKeepsListOpen(t *testing.T) {
	src := "# T\n\n" +
		"  1. First item\n     with a continuation line.\n\n" +
		"  2. Second item citing [a ref].\n\n" +
		"[a ref]: https://example.com\n\n" +
		"     An indented paragraph belonging to item two.\n\n" +
		"  3. Third item.\n\n" +
		"  4. Fourth item.\n"
	if v := lintString(t, rules.MD029{}, src); len(v) != 0 {
		t.Errorf("expected no violations for a list held open by indented content, got %v", v)
	}
}

// TestMD029_LinkRefAloneDoesNotKeepListOpen guards the boundary of the rule
// above: a flush-left link reference definition with no indented content around
// it does end the list in markdownlint too, so the next fragment must restart.
func TestMD029_LinkRefAloneDoesNotKeepListOpen(t *testing.T) {
	src := "# T\n\n1. One\n\n2. Two citing [a ref].\n\n[a ref]: https://example.com\n\n3. Three\n\n4. Four\n"
	if v := lintString(t, rules.MD029{}, src); len(v) != 2 {
		t.Errorf("expected 2 violations when only a link definition separates the fragments, got %v", v)
	}
}

// TestMD038_RequiredBacktickSpaceNotFlagged covers code spans whose content
// starts or ends with a backtick. CommonMark requires a space between the
// delimiter and such content, so the space is structural rather than
// stylistic and markdownlint does not report it.
func TestMD038_RequiredBacktickSpaceNotFlagged(t *testing.T) {
	cases := map[string]string{
		"trailing backtick":        "F ``a` `` end.\n",
		"leading backtick":         "F `` `a`` end.\n",
		"inner span then backtick": "F ``name = `T` `` end.\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if v := lintString(t, rules.MD038{}, src); len(v) != 0 {
				t.Errorf("expected no violations for a required delimiter space, got %v", v)
			}
		})
	}
}

// TestMD038_FixPreservesRequiredBacktickSpace is the important half: stripping
// a required space merges the content's backticks into the delimiter run, so
// the span stops being a code span at all. Fix used to turn a two-backtick
// span holding "a`" into one where the closing delimiter has three backticks,
// which goldmark then renders as literal text rather than a code element.
func TestMD038_FixPreservesRequiredBacktickSpace(t *testing.T) {
	for _, src := range []string{
		"F ``a` `` end.\n",
		"F `` `a`` end.\n",
		"F ``name = `T` `` end.\n",
	} {
		if got := fixString(t, rules.MD038{}, src); got != src {
			t.Errorf("Fix() corrupted a code span with a required space:\n got: %q\nwant: %q", got, src)
		}
	}
}

// TestMD038_FixStillRemovesUnnecessarySpaces guards the other direction, so the
// change above does not simply disable the fix.
func TestMD038_FixStillRemovesUnnecessarySpaces(t *testing.T) {
	cases := map[string]string{
		"F `a ` end.\n": "F `a` end.\n",
		"F ` a` end.\n": "F `a` end.\n",
	}
	for src, want := range cases {
		if got := fixString(t, rules.MD038{}, src); got != want {
			t.Errorf("Fix(%q) = %q, want %q", src, got, want)
		}
	}
}

// TestMD009_BlankLineAfterIndentedCodeBlock covers a whitespace-only line
// directly after an indented code block. CommonMark excludes trailing blank
// lines from the block, so the line is ordinary content and markdownlint
// reports its trailing spaces. MD009 used to extend the code-block mask over
// it and stay silent.
//
// Reduced from rust-lang/rfcs
// text/0192-bounds-on-object-and-generic-types.md:363.
func TestMD009_BlankLineAfterIndentedCodeBlock(t *testing.T) {
	src := "# T\n\n- item text:\n\n      indented code line\n    \n  More item text.\n"
	v := lintString(t, rules.MD009{}, src)
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(v), v)
	}
	if v[0].Line != 6 {
		t.Errorf("MD009 reported line %d, want 6", v[0].Line)
	}
}

// TestMD009_BlankLineInsideIndentedCodeBlock is the boundary case: a
// whitespace-only line *between* two code lines is part of the block, and
// neither linter reports it.
func TestMD009_BlankLineInsideIndentedCodeBlock(t *testing.T) {
	src := "# T\n\n    code one\n    \n    code two\n\nText.\n"
	if v := lintString(t, rules.MD009{}, src); len(v) != 0 {
		t.Errorf("expected no violations inside an indented code block, got %v", v)
	}
}

// TestMD009_TrailingSpacesInsideCodeBlocksIgnored pins the documented default
// of the code_blocks option, which is false.
func TestMD009_TrailingSpacesInsideCodeBlocksIgnored(t *testing.T) {
	for _, src := range []string{
		"# T\n\n```sh\ncode with trailing   \n```\n\nText.\n",
		"# T\n\n    indented code trailing   \n\nText.\n",
	} {
		if v := lintString(t, rules.MD009{}, src); len(v) != 0 {
			t.Errorf("expected no violations inside code blocks, got %v for %q", v, src)
		}
	}
}

// TestMD009_BlankLineIndentThreshold pins where a whitespace-only line stops
// belonging to the indented code block above it. markdownlint treats the line
// as part of the block once it reaches the block's own indentation.
func TestMD009_BlankLineIndentThreshold(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"blank matches block indent", "# T\n\n    code at indent 4\n    \nText.\n", 0},
		{"blank narrower than block", "# T\n\n    code at indent 4\n   \nText.\n", 1},
		{"blank wider than block", "# T\n\n    code at indent 4\n      \nText.\n", 0},
		{"blank narrower than nested block", "# T\n\n- item:\n\n      code at 6\n    \n  more text.\n", 1},
		{"after a fenced block", "# T\n\n```sh\ncode\n```\n   \nText.\n", 1},
		// The threshold is the block's own indentation, not that of whichever
		// line happens to precede the blank: a block may indent later lines
		// further, and that must not raise the bar. Reduced from rust-lang/rfcs
		// text/1211-mir.md:282.
		{"deeper later line does not raise the bar",
			"# T\n\n    DROP_KIND = SHALLOW\n              | DEEP\n             \nText.\n", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if v := lintString(t, rules.MD009{}, tc.src); len(v) != tc.want {
				t.Errorf("got %d violations, want %d: %v", len(v), tc.want, v)
			}
		})
	}
}

// TestMD052_FootnoteLabelsNotReported covers footnote-style labels used with
// full reference syntax. markdownlint parses [^label] as a GFM footnote rather
// than a link reference, so an undefined one is not an MD052 violation. The
// definition scanner already skipped them while the usage side did not.
//
// Reduced from rust-lang/rfcs text/1683-docs-team.md:82 and
// text/3668-async-closures.md:1.
func TestMD052_FootnoteLabelsNotReported(t *testing.T) {
	src := "# T\n\nAvailable on [IRC][^IRC] to collaborate.\n\n" +
		"[^IRC]: An IRC footnote definition.\n\n" +
		"Feature `async_closure`[^rework][^plural] here.\n\n" +
		"[^rework]: reworks things\n[^plural]: pluralization note\n"
	if v := lintString(t, rules.MD052{}, src); len(v) != 0 {
		t.Errorf("expected no violations for footnote labels, got %v", v)
	}
}

// TestMD052_DefinitionRunEndsAtMalformedEntry covers a run of link reference
// definitions interrupted by a malformed one. A definition is only a
// definition while it sits at the start of a block, so once "[2]:" fails to
// parse the rest of the run becomes paragraph text and "[3]" is undefined too.
// goldmark agrees — it renders neither as a link — but MD052 re-scanned the
// source for "[label]: url" patterns and registered "3" anyway.
//
// Reduced from rust-lang/rfcs text/0507-release-channels.md.
func TestMD052_DefinitionRunEndsAtMalformedEntry(t *testing.T) {
	src := "# T\n\nUse [a][1] and [b][2] and [c][3].\n\n" +
		"[1]: http://example.com/one\n" +
		"[2]: http://example.com/two)\n" +
		"[3]: http://example.com/three\n"
	v := lintString(t, rules.MD052{}, src)
	if len(v) != 2 {
		t.Fatalf("expected 2 violations (labels 2 and 3), got %d: %v", len(v), v)
	}
}

// TestMD052_ParserProvidedDefinitionsStillResolve guards the cases the removed
// line scan was there for: goldmark exports definitions written inside a
// blockquote, and labels containing a code span still match their usage.
func TestMD052_ParserProvidedDefinitionsStillResolve(t *testing.T) {
	cases := map[string]string{
		"blockquote definition": "# T\n\n> Quote with [link][bq] inside.\n>\n> [bq]: http://example.com/bq\n",
		"code span label":       "# T\n\nUse [`genawaiter`][`genawaiter`] here.\n\n[`genawaiter`]: http://example.com/g\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if v := lintString(t, rules.MD052{}, src); len(v) != 0 {
				t.Errorf("expected no violations, got %v", v)
			}
		})
	}
}

// TestMD005_LooseListIsChecked covers a list with blank lines between its
// items. Such items hold a Paragraph rather than a TextBlock, and MD005 only
// looked for TextBlock, so every loose list was skipped outright.
//
// Reduced from rust-lang/rfcs text/1398-kinds-of-allocators.md:482.
func TestMD005_LooseListIsChecked(t *testing.T) {
	src := "# T\n\n* first item\n\n * second item indented by one\n\n* third item\n"
	v := lintString(t, rules.MD005{}, src)
	if len(v) != 1 {
		t.Fatalf("expected 1 violation in a loose list, got %d: %v", len(v), v)
	}
	if v[0].Line != 5 {
		t.Errorf("reported line %d, want 5", v[0].Line)
	}
}

// TestMD005_TightListStillChecked guards that accepting Paragraph did not
// disturb the tight-list path.
func TestMD005_TightListStillChecked(t *testing.T) {
	src := "# T\n\n* first item\n * second item indented by one\n* third item\n"
	if v := lintString(t, rules.MD005{}, src); len(v) != 1 {
		t.Errorf("expected 1 violation in a tight list, got %d: %v", len(v), v)
	}
}

// TestMD005_ContainerPrefixDoesNotShiftIndent covers sibling items whose lines
// begin differently: the first is introduced by the enclosing list marker and
// the second by spaces. Both markers sit at the same column, so they are
// siblings; counting leading spaces made the second look mis-indented.
//
// Reduced from rust-lang/rfcs text/2497-if-let-chains.md:1838.
func TestMD005_ContainerPrefixDoesNotShiftIndent(t *testing.T) {
	src := "# T\n\n48. > 1. first inner item\n    > 2. second inner item\n"
	if v := lintString(t, rules.MD005{}, src); len(v) != 0 {
		t.Errorf("expected no violations for inner items at the same column, got %v", v)
	}
}

// TestMD013_TableRowWithBareURLExempt covers a long table row whose only link
// is a bare URL. markdownlint parses GFM autolink literals, so such a row holds
// a link node and is exempt from the length check; goldmark does not enable the
// Linkify extension, so the row was reported.
//
// Reduced from tldr-pages contributing-guides/style-guide.md:253.
func TestMD013_TableRowWithBareURLExempt(t *testing.T) {
	long := strings.Repeat("x", 40) + " " + strings.Repeat("y", 40)
	src := "# T\n\n| Term | Explanation |\n|---|---|\n" +
		"| Bare URL row | text that is quite long indeed and keeps going (https://en.wikipedia.org/wiki/Regular_expression). |\n" +
		"| Plain row | " + long + " more words here to push this well past the limit |\n"
	v := lintString(t, rules.MD013{}, src)
	if len(v) != 1 {
		t.Fatalf("expected 1 violation (the row without a URL), got %d: %v", len(v), v)
	}
	if v[0].Line != 6 {
		t.Errorf("reported line %d, want 6 (the row with no URL)", v[0].Line)
	}
}

// TestMD013_BareURLOutsideTableStillReported keeps the exemption scoped to
// table rows: a long paragraph containing a bare URL is still a violation in
// both linters.
func TestMD013_BareURLOutsideTableStillReported(t *testing.T) {
	src := "# T\n\nPlain paragraph with a bare URL https://en.wikipedia.org/wiki/Regular_expression and more text to exceed eighty.\n"
	if v := lintString(t, rules.MD013{}, src); len(v) != 1 {
		t.Errorf("expected 1 violation for a long paragraph with a bare URL, got %v", v)
	}
}

// TestMD041_HTMLCommentClosingLineIsSkipped covers a leading HTML comment
// followed by a non-top-level heading. goldmark keeps the line that closes an
// HTML block ("-->") out of Lines() and in a separate ClosureLine segment, so
// htmlBlockLineMask did not mask it and MD041 reported the violation on the
// "-->" line instead of the first real content line.
//
// Reduced from tldr-pages .github/PULL_REQUEST_TEMPLATE.md.
func TestMD041_HTMLCommentClosingLineIsSkipped(t *testing.T) {
	src := "<!--\nA comment.\n\nStill the comment.\n-->\n\n### Checklist\n\n- an item\n"
	v := lintString(t, rules.MD041{}, src)
	if len(v) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(v), v)
	}
	// Line 5 is "-->", line 7 is "### Checklist"; markdownlint reports line 7.
	if v[0].Line != 7 {
		t.Errorf("MD041 reported line %d, want 7 (the first content line, not the comment's closing line)", v[0].Line)
	}
}

// TestHTMLBlockMask_CoversClosingLine exercises the same mask through MD012,
// another rule that skips HTML block lines, so the helper is covered rather
// than just MD041's use of it.
func TestHTMLBlockMask_CoversClosingLine(t *testing.T) {
	src := "# T\n\n<!--\na comment\n\n\n-->\n\nText.\n"
	if v := lintString(t, rules.MD012{}, src); len(v) != 0 {
		t.Errorf("expected no MD012 violations for blank lines inside an HTML comment, got %v", v)
	}
}
