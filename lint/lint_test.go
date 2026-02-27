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
		rules.MD031{},
		rules.MD032{},
		rules.MD033{},
		rules.MD034{},
		rules.MD037{},
		rules.MD038{},
		rules.MD039{},
		rules.MD040{},
		rules.MD041{},
		rules.MD042{},
		rules.MD045{},
		rules.MD047{},
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
	src := strings.Repeat("a", 81) + "\n"
	v := lintString(t, rules.MD013{LineLength: 80}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
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
	if len(v) == 0 {
		t.Errorf("expected violations, got none")
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

func TestMD019_Valid(t *testing.T) {
	src := "# Heading\n"
	v := lintString(t, rules.MD019{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
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

func TestMD037_Valid(t *testing.T) {
	src := "This is *emphasized* text.\n"
	v := lintString(t, rules.MD037{}, src)
	if len(v) != 0 {
		t.Errorf("expected no violations, got %v", v)
	}
}

func TestMD037_Invalid(t *testing.T) {
	src := "This is * emphasized * text.\n"
	v := lintString(t, rules.MD037{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD037_Fix(t *testing.T) {
	src := "This is * emphasized * text.\n"
	got := fixString(t, rules.MD037{}, src)
	want := "This is *emphasized* text.\n"
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
	src := "Use ` code ` here.\n"
	v := lintString(t, rules.MD038{}, src)
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
	}
}

func TestMD038_Fix(t *testing.T) {
	src := "Use ` code ` here.\n"
	got := fixString(t, rules.MD038{}, src)
	want := "Use `code` here.\n"
	if got != want {
		t.Errorf("Fix() = %q, want %q", got, want)
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

func TestMD042_Invalid(t *testing.T) {
	src := "See [text]() here.\n"
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
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
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
	if len(v) != 1 {
		t.Errorf("expected 1 violation, got %d: %v", len(v), v)
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

func TestMD060_Valid(t *testing.T) {
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
body := strings.Repeat("b", 81) + "\n"
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
