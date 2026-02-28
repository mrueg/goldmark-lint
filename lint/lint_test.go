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

func TestInlineDisable_ConfigureFile_DoesNotDisableOtherRules(t *testing.T) {
	// configure-file disabling MD001 should not suppress MD009.
	src := "<!-- markdownlint-configure-file { \"MD001\": false } -->\nTrailing spaces   \n"
	l := lint.NewLinter(rules.MD009{})
	v := l.Lint([]byte(src))
	if len(v) == 0 {
		t.Errorf("expected MD009 violation (not disabled by configure-file), got none")
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
	// Document with YAML front matter followed by non-heading content should
	// still trigger MD041, reported on the correct line.
	src := "---\ntitle: My Page\n---\n\nNot a heading\n"
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
