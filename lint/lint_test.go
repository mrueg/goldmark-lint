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
		rules.MD007{},
		rules.MD009{},
		rules.MD010{},
		rules.MD011{},
		rules.MD012{},
		rules.MD013{},
		rules.MD018{},
		rules.MD019{},
		rules.MD020{},
		rules.MD021{},
		rules.MD022{},
		rules.MD024{},
		rules.MD025{},
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
